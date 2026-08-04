package terrain

import (
	"math"
	"sort"
)

func ComputeHydrologyDiagnostics(sites []Vector3D, cells []VoronoiCell, elevation []float64, seed int64) HydrologyDiagnostics {
	state := buildHydrologyScaffold(sites, cells, elevation, seed)
	return buildHydrologyDiagnosticsFromState(sites, cells, elevation, state)
}

// ComputeHydrologyDiagnosticsFromRunoff exposes the coarse hydrology scaffold
// and summary metrics for callers that already have a climate-driven runoff
// field and want to route that water across the final DEM.
func ComputeHydrologyDiagnosticsFromRunoff(
	sites []Vector3D,
	cells []VoronoiCell,
	elevation []float64,
	runoff []float64,
) HydrologyDiagnostics {
	state := buildHydrologyStateFromRunoff(cells, elevation, runoff)
	return buildHydrologyDiagnosticsFromState(sites, cells, elevation, state)
}

func buildHydrologyDiagnosticsFromState(
	sites []Vector3D,
	cells []VoronoiCell,
	elevation []float64,
	state hydrologyState,
) HydrologyDiagnostics {
	var metrics TerrainMetrics
	populateDrainageMetricsFromState(&metrics, elevation, state)
	return HydrologyDiagnostics{
		FluvialChannelCoverage:  metrics.FluvialChannelCoverage,
		EndorheicCatchmentPct:   metrics.EndorheicCatchmentPct,
		InlandLakeCoverage:      metrics.InlandLakeCoverage,
		NumMajorEndorheicBasins: metrics.NumMajorEndorheicBasins,
		Regions:                 summarizeHydrologyRegions(elevation, state),
		Classes:                 summarizeHydrologyClasses(state.CellClass),
		Scaffold: func() *HydrologyScaffold {
			boundaryFlow := buildBoundaryFlowContracts(sites, cells, elevation, state)
			return &HydrologyScaffold{
				Receivers:        append([]int(nil), state.Receivers...),
				TerminalSinks:    append([]int(nil), state.TerminalSinks...),
				Runoff:           append([]float64(nil), state.Runoff...),
				Accumulation:     append([]float64(nil), state.Accumulation...),
				ChannelStrength:  append([]float64(nil), state.ChannelStrength...),
				WaterBodyLabel:   append([]int(nil), state.WaterLabels...),
				CellClass:        append([]string(nil), state.CellClass...),
				OutletMode:       append([]string(nil), state.OutletMode...),
				MaxOutflows:      append([]int(nil), state.MaxOutflows...),
				BoundaryFlow:     boundaryFlow,
				BoundarySideFlow: buildBoundarySideFlowContracts(boundaryFlow),
			}
		}(),
		TerrainRefinement: buildTerrainRefinementScaffold(sites, cells, elevation, state),
	}
}

func computeDrainageMetrics(metrics *TerrainMetrics, cells []VoronoiCell, elevation []float64) {
	state := buildUniformHydrologyState(cells, elevation)
	populateDrainageMetricsFromState(metrics, elevation, state)
}

type hydrologyState struct {
	Receivers        []int
	TerminalSinks    []int
	Runoff           []float64
	Accumulation     []float64
	ChannelStrength  []float64
	WaterLabels      []int
	WaterSizes       map[int]int
	LargestWater     int
	CoastDistance    []float64
	InflowCount      []int
	LandCount        int
	ChannelThreshold float64
	CellClass        []string
	OutletMode       []string
	MaxOutflows      []int
}

func buildUniformHydrologyState(cells []VoronoiCell, elevation []float64) hydrologyState {
	runoff := make([]float64, len(elevation))
	for i, elev := range elevation {
		if elev > 0 {
			runoff[i] = 1
		}
	}
	return buildHydrologyStateFromRunoff(cells, elevation, runoff)
}

func buildHydrologyScaffold(sites []Vector3D, cells []VoronoiCell, elevation []float64, seed int64) hydrologyState {
	distFromCoast, maxDist := computeFinalLandDistanceFromCoast(cells, elevation)
	runoff := ComputeLongTermRunoffProxy(sites, elevation, distFromCoast, maxDist, seed)
	return buildHydrologyStateFromRunoff(cells, elevation, runoff)
}

func buildHydrologyStateFromRunoff(cells []VoronoiCell, elevation []float64, runoff []float64) hydrologyState {
	conditioned := append([]float64(nil), elevation...)
	receivers := ComputeDrainageReceivers(cells, conditioned)
	BreachDrainageSinks(cells, conditioned, receivers, 260)
	receivers = ComputeDrainageReceivers(cells, conditioned)
	accumulation, _ := ComputeFlowAccumulation(receivers, conditioned, runoff)
	waterLabels, waterSizes, largestWater := computeWaterComponentLabels(cells, elevation)

	landCount := 0
	runoffSum := 0.0
	for _, elev := range elevation {
		if elev > 0 {
			landCount++
		}
	}
	for i, elev := range elevation {
		if elev <= 0 {
			continue
		}
		runoffSum += math.Max(runoff[i], 0)
	}
	meanRunoff := 1.0
	if landCount > 0 && runoffSum > 0 {
		meanRunoff = runoffSum / float64(landCount)
	}
	channelThreshold := hydrologyChannelThreshold(elevation, runoff, accumulation, landCount, meanRunoff)
	channelStrength := make([]float64, len(elevation))
	inflowCount := make([]int, len(elevation))
	for i, elev := range elevation {
		if elev <= 0 {
			continue
		}
		channelStrength[i] = accumulation[i] / channelThreshold
		receiver := receivers[i]
		if receiver >= 0 && receiver < len(elevation) {
			inflowCount[receiver]++
		}
	}

	sinkMemo := make([]int, len(elevation))
	for i := range sinkMemo {
		sinkMemo[i] = math.MinInt
	}
	terminal := make([]int, len(elevation))
	for i := range terminal {
		terminal[i] = -1
	}
	for i, elev := range elevation {
		if elev <= 0 {
			continue
		}
		terminal[i] = terminalDrainageDestination(i, receivers, elevation, waterLabels, largestWater, sinkMemo)
	}
	coastDistance, _ := computeFinalLandDistanceFromCoast(cells, elevation)
	cellClass, outletMode, maxOutflows := classifyHydrologyCells(cells, elevation, waterLabels, waterSizes, largestWater, receivers, terminal, channelStrength, coastDistance, inflowCount)

	return hydrologyState{
		Receivers:        receivers,
		TerminalSinks:    terminal,
		Runoff:           runoff,
		Accumulation:     accumulation,
		ChannelStrength:  channelStrength,
		WaterLabels:      waterLabels,
		WaterSizes:       waterSizes,
		LargestWater:     largestWater,
		CoastDistance:    coastDistance,
		InflowCount:      inflowCount,
		LandCount:        landCount,
		ChannelThreshold: channelThreshold,
		CellClass:        cellClass,
		OutletMode:       outletMode,
		MaxOutflows:      maxOutflows,
	}
}

func hydrologyChannelThreshold(elevation []float64, runoff []float64, accumulation []float64, landCount int, meanRunoff float64) float64 {
	if landCount <= 0 {
		return 1
	}
	landAccumulation := make([]float64, 0, landCount)
	for i, elev := range elevation {
		if elev <= 0 || i >= len(accumulation) {
			continue
		}
		landAccumulation = append(landAccumulation, math.Max(accumulation[i], 0))
	}
	if len(landAccumulation) == 0 {
		return math.Max(meanRunoff, 1e-6)
	}
	sort.Float64s(landAccumulation)
	// Flow accumulation is computed on the receiver graph, not a raster area
	// integral. Normalizing by a fixed upper-tail rank keeps channel hierarchy
	// comparable as the mesh is refined and local catchments split.
	quantileThreshold := sortedPercentile(landAccumulation, 93.5)
	runoffFloor := math.Max(12*meanRunoff, maxRunoff(runoff))
	return math.Max(runoffFloor, quantileThreshold)
}

func sortedPercentile(sortedValues []float64, pct float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	if pct <= 0 {
		return sortedValues[0]
	}
	if pct >= 100 {
		return sortedValues[len(sortedValues)-1]
	}
	idx := (float64(len(sortedValues)-1) * pct) / 100
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return sortedValues[lo]
	}
	return sortedValues[lo]*(float64(hi)-idx) + sortedValues[hi]*(idx-float64(lo))
}

func maxRunoff(runoff []float64) float64 {
	maxValue := 0.0
	for _, value := range runoff {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func populateDrainageMetricsFromState(metrics *TerrainMetrics, elevation []float64, state hydrologyState) {
	if state.LandCount == 0 {
		return
	}

	channelCells := 0
	for i, elev := range elevation {
		if elev > 0 && state.Accumulation[i] >= state.ChannelThreshold {
			channelCells++
		}
	}
	metrics.FluvialChannelCoverage = float64(channelCells) / float64(state.LandCount)

	sinkMemo := make([]int, len(elevation))
	for i := range sinkMemo {
		sinkMemo[i] = math.MinInt
	}
	catchmentSizes := make(map[int]int)
	majorThreshold := int(math.Max(60, 0.003*float64(state.LandCount)))

	for i, elev := range elevation {
		if elev <= 0 {
			continue
		}
		sink := state.TerminalSinks[i]
		if sink == math.MinInt {
			sink = terminalDrainageDestination(i, state.Receivers, elevation, state.WaterLabels, state.LargestWater, sinkMemo)
		}
		if sink != -1 {
			catchmentSizes[sink]++
		}
	}

	endorheicCells := 0
	for _, size := range catchmentSizes {
		if size >= majorThreshold {
			metrics.NumMajorEndorheicBasins++
			endorheicCells += size
		}
	}
	metrics.EndorheicCatchmentPct = float64(endorheicCells) / float64(state.LandCount)
	metrics.InlandLakeCoverage = computeInlandLakeCoverageFromLabels(elevation, state.WaterLabels, state.WaterSizes, state.LargestWater)
}

func summarizeHydrologyRegions(elevation []float64, state hydrologyState) []HydrologyRegionSummary {
	landRunoff := make([]float64, 0, state.LandCount)
	for i, elev := range elevation {
		if elev > 0 {
			landRunoff = append(landRunoff, state.Runoff[i])
		}
	}
	if len(landRunoff) == 0 {
		return nil
	}
	sort.Float64s(landRunoff)
	q1 := landRunoff[len(landRunoff)/3]
	q2 := landRunoff[(2*len(landRunoff))/3]
	names := []string{"dry", "mid", "wet"}
	stats := make([]HydrologyRegionSummary, 3)
	channelThreshold := state.ChannelThreshold
	for i := range stats {
		stats[i].Name = names[i]
	}

	for i, elev := range elevation {
		if elev <= 0 {
			continue
		}
		bin := 1
		if state.Runoff[i] <= q1 {
			bin = 0
		} else if state.Runoff[i] > q2 {
			bin = 2
		}
		s := &stats[bin]
		s.CellCount++
		s.MeanRunoff += state.Runoff[i]
		s.MeanAccumulation += state.Accumulation[i]
		if state.Accumulation[i] >= channelThreshold {
			s.ChannelCoverage++
		}
		if state.TerminalSinks[i] != -1 {
			s.EndorheicCatchmentPct++
		}
		if state.TerminalSinks[i] < -1 {
			s.InlandLakeReachPct++
		}
	}

	out := make([]HydrologyRegionSummary, 0, 3)
	for _, s := range stats {
		if s.CellCount == 0 {
			continue
		}
		inv := 1.0 / float64(s.CellCount)
		s.MeanRunoff *= inv
		s.MeanAccumulation *= inv
		s.ChannelCoverage *= inv
		s.EndorheicCatchmentPct *= inv
		s.InlandLakeReachPct *= inv
		out = append(out, s)
	}
	return out
}

func summarizeHydrologyClasses(classes []string) []HydrologyClassSummary {
	counts := make(map[string]int)
	for _, className := range classes {
		if className == "" {
			continue
		}
		counts[className]++
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]HydrologyClassSummary, 0, len(keys))
	for _, key := range keys {
		out = append(out, HydrologyClassSummary{Class: key, CellCount: counts[key]})
	}
	return out
}

func computeFinalLandDistanceFromCoast(cells []VoronoiCell, elevation []float64) ([]float64, float64) {
	dist := make([]float64, len(elevation))
	for i := range dist {
		dist[i] = math.Inf(1)
	}
	queue := make([]int, 0)
	for i, elev := range elevation {
		if elev <= 0 {
			continue
		}
		for _, neighborIdx := range cells[i].NeighborSiteIndices {
			neighbor := int(neighborIdx)
			if neighbor < 0 || neighbor >= len(elevation) {
				continue
			}
			if elevation[neighbor] <= 0 {
				dist[i] = 0
				queue = append(queue, i)
				break
			}
		}
	}
	head := 0
	for head < len(queue) {
		current := queue[head]
		head++
		nextDist := dist[current] + 1
		for _, neighborIdx := range cells[current].NeighborSiteIndices {
			neighbor := int(neighborIdx)
			if neighbor < 0 || neighbor >= len(elevation) || elevation[neighbor] <= 0 {
				continue
			}
			if nextDist < dist[neighbor] {
				dist[neighbor] = nextDist
				queue = append(queue, neighbor)
			}
		}
	}
	maxDist := 0.0
	for i, elev := range elevation {
		if elev > 0 && !math.IsInf(dist[i], 1) && dist[i] > maxDist {
			maxDist = dist[i]
		}
	}
	if maxDist < 1 {
		maxDist = 1
	}
	return dist, maxDist
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func buildBoundaryFlowContracts(
	sites []Vector3D,
	cells []VoronoiCell,
	elevation []float64,
	state hydrologyState,
) []HydrologyBoundaryFlow {
	contracts := make([]HydrologyBoundaryFlow, len(elevation))
	inflowCounts := make([]int, len(elevation))
	for i, receiver := range state.Receivers {
		if i < 0 || i >= len(elevation) || elevation[i] <= 0 || receiver < 0 || receiver >= len(elevation) {
			continue
		}
		inflowCounts[receiver]++
	}
	for i := range contracts {
		contracts[i].OutflowNeighbor = -1
		if inflowCounts[i] > 0 {
			contracts[i].InflowNeighbors = make([]int, 0, inflowCounts[i])
			contracts[i].InflowBearingDeg = make([]float64, 0, inflowCounts[i])
			contracts[i].InflowStrength = make([]float64, 0, inflowCounts[i])
		}
	}

	for i, receiver := range state.Receivers {
		if elevation[i] <= 0 {
			continue
		}

		if receiver >= 0 && receiver < len(elevation) {
			contracts[i].OutflowNeighbor = receiver
			contracts[i].OutflowBearingDeg = boundaryBearingDegrees(sites[i], sites[receiver])
			contracts[i].OutflowStrength = state.Accumulation[i]

			dst := &contracts[receiver]
			dst.InflowNeighbors = append(dst.InflowNeighbors, i)
			dst.InflowBearingDeg = append(dst.InflowBearingDeg, boundaryBearingDegrees(sites[receiver], sites[i]))
			dst.InflowStrength = append(dst.InflowStrength, state.Accumulation[i])
		}
	}

	for i := range contracts {
		if len(contracts[i].InflowNeighbors) > 1 {
			sortBoundaryInflows(&contracts[i])
		}
	}

	return contracts
}

func buildBoundarySideFlowContracts(boundary []HydrologyBoundaryFlow) [][]HydrologyBoundarySideFlow {
	sectors := []struct {
		name   string
		center float64
	}{
		{"N", 0},
		{"NE", 45},
		{"E", 90},
		{"SE", 135},
		{"S", 180},
		{"SW", 225},
		{"W", 270},
		{"NW", 315},
	}

	out := make([][]HydrologyBoundarySideFlow, len(boundary))
	for i, flow := range boundary {
		agg := make([]HydrologyBoundarySideFlow, len(sectors))
		for j, sector := range sectors {
			agg[j] = HydrologyBoundarySideFlow{
				Sector:           sector.name,
				BearingCenterDeg: sector.center,
			}
		}

		for j, bearing := range flow.InflowBearingDeg {
			idx := bearingSectorIndex(bearing)
			if idx >= 0 && idx < len(agg) {
				agg[idx].InflowStrength += flow.InflowStrength[j]
			}
		}
		if flow.OutflowNeighbor >= 0 && flow.OutflowStrength > 0 {
			idx := bearingSectorIndex(flow.OutflowBearingDeg)
			if idx >= 0 && idx < len(agg) {
				agg[idx].OutflowStrength += flow.OutflowStrength
			}
		}
		out[i] = agg
	}
	return out
}

func bearingSectorIndex(bearing float64) int {
	b := math.Mod(bearing, 360.0)
	if b < 0 {
		b += 360.0
	}
	return int(math.Floor((b+22.5)/45.0)) % 8
}

func buildTerrainRefinementScaffold(
	sites []Vector3D,
	cells []VoronoiCell,
	elevation []float64,
	state hydrologyState,
) *TerrainRefinementScaffold {
	scaffold := &TerrainRefinementScaffold{
		Cells: make([]TerrainRefinementCellConstraint, len(elevation)),
	}
	for i := range elevation {
		cell := TerrainRefinementCellConstraint{
			BaseElevation: elevation[i],
		}
		if i >= len(cells) {
			scaffold.Cells[i] = cell
			continue
		}

		meanNeighbor, relief, downslopeBearing, downslopeStrength, boundary := computeTerrainCellConstraint(
			i, sites, cells, elevation, state,
		)
		cell.MeanNeighborElevation = meanNeighbor
		cell.LocalRelief = relief
		cell.DownslopeBearingDeg = downslopeBearing
		cell.DownslopeStrength = downslopeStrength
		cell.Boundary = boundary

		if i < len(state.Receivers) {
			receiver := state.Receivers[i]
			if receiver >= 0 && receiver < len(sites) {
				cell.ChannelBearingDeg = boundaryBearingDegrees(sites[i], sites[receiver])
			} else {
				cell.ChannelBearingDeg = downslopeBearing
			}
		}
		if i < len(state.ChannelStrength) {
			cell.ChannelStrength = state.ChannelStrength[i]
		}

		scaffold.Cells[i] = cell
	}
	return scaffold
}

func computeTerrainCellConstraint(
	idx int,
	sites []Vector3D,
	cells []VoronoiCell,
	elevation []float64,
	state hydrologyState,
) (meanNeighbor, relief, downslopeBearing, downslopeStrength float64, boundary []TerrainBoundaryConstraint) {
	centerElev := elevation[idx]
	minElev := centerElev
	maxElev := centerElev
	sum := 0.0
	count := 0.0
	steepestDrop := 0.0
	steepestBearing := 0.0

	boundary = make([]TerrainBoundaryConstraint, 0, len(cells[idx].NeighborSiteIndices))
	for _, neighborIdx := range cells[idx].NeighborSiteIndices {
		neighbor := int(neighborIdx)
		if neighbor < 0 || neighbor >= len(elevation) {
			continue
		}
		neighborElev := elevation[neighbor]
		sum += neighborElev
		count++
		if neighborElev < minElev {
			minElev = neighborElev
		}
		if neighborElev > maxElev {
			maxElev = neighborElev
		}

		bearing := boundaryBearingDegrees(sites[idx], sites[neighbor])
		drop := centerElev - neighborElev
		if drop > steepestDrop {
			steepestDrop = drop
			steepestBearing = bearing
		}

		crossingClass := ""
		crossingStrength := 0.0
		if idx < len(state.Receivers) && state.Receivers[idx] == neighbor {
			crossingClass = "outflow"
			if idx < len(state.Accumulation) {
				crossingStrength = state.Accumulation[idx]
			}
		} else if neighbor < len(state.Receivers) && state.Receivers[neighbor] == idx {
			crossingClass = "inflow"
			if neighbor < len(state.Accumulation) {
				crossingStrength = state.Accumulation[neighbor]
			}
		}

		boundary = append(boundary, TerrainBoundaryConstraint{
			Neighbor:          neighbor,
			BearingDeg:        bearing,
			BoundaryElevation: 0.5 * (centerElev + neighborElev),
			NeighborElevation: neighborElev,
			CrossingClass:     crossingClass,
			CrossingStrength:  crossingStrength,
		})
	}

	if count > 0 {
		meanNeighbor = sum / count
	}
	relief = maxElev - minElev
	downslopeBearing = steepestBearing
	downslopeStrength = steepestDrop
	return meanNeighbor, relief, downslopeBearing, downslopeStrength, boundary
}

func classifyHydrologyCells(
	cells []VoronoiCell,
	elevation []float64,
	waterLabels []int,
	waterSizes map[int]int,
	largestWater int,
	receivers []int,
	terminal []int,
	channelStrength []float64,
	coastDistance []float64,
	inflowCount []int,
) ([]string, []string, []int) {
	classes := make([]string, len(elevation))
	outletMode := make([]string, len(elevation))
	maxOutflows := make([]int, len(elevation))

	for i, elev := range elevation {
		if elev <= 0 {
			label := waterLabels[i]
			if label >= 0 && label != largestWater && waterSizes[label] >= 3 {
				classes[i] = "lake"
				outletMode[i] = "none"
				maxOutflows[i] = 0
			} else {
				classes[i] = "ocean"
				outletMode[i] = "none"
				maxOutflows[i] = 0
			}
			continue
		}

		cellInflowCount := 0
		if i < len(inflowCount) {
			cellInflowCount = inflowCount[i]
		}
		receiver := -1
		if i < len(receivers) {
			receiver = receivers[i]
		}
		outToOcean := receiver >= 0 && receiver < len(elevation) && elevation[receiver] <= 0
		isTerminal := receiver < 0 || terminal[i] == i
		lakeReach := terminal[i] < -1
		channel := channelStrength[i]
		lowSlope := true
		if receiver >= 0 && receiver < len(elevation) {
			lowSlope = (elev - elevation[receiver]) < 80.0
		}
		nearCoast := i < len(coastDistance) && !math.IsInf(coastDistance[i], 1) && coastDistance[i] <= 1.0
		adjacentInlandLake := false
		for _, neighborIdx := range cells[i].NeighborSiteIndices {
			neighbor := int(neighborIdx)
			if neighbor < 0 || neighbor >= len(elevation) {
				continue
			}
			label := waterLabels[neighbor]
			if elevation[neighbor] <= 0 && label >= 0 && label != largestWater && waterSizes[label] >= 3 {
				adjacentInlandLake = true
				break
			}
		}

		switch {
		case isTerminal:
			classes[i] = "endorheic_basin"
			outletMode[i] = "none"
			maxOutflows[i] = 0
		case outToOcean && channel >= 2.0 && lowSlope:
			classes[i] = "delta"
			outletMode[i] = "multiple"
			maxOutflows[i] = 2
		case adjacentInlandLake && lowSlope && (cellInflowCount >= 2 || channel >= 1.4):
			classes[i] = "lake_complex"
			outletMode[i] = "multiple"
			maxOutflows[i] = 2
		case outToOcean:
			classes[i] = "coast_outlet"
			outletMode[i] = "single"
			maxOutflows[i] = 1
		case cellInflowCount >= 2 && channel >= 1.4 && lowSlope && !nearCoast:
			classes[i] = "floodplain"
			outletMode[i] = "multiple"
			maxOutflows[i] = 2
		case cellInflowCount >= 2 && channel >= 1.0:
			classes[i] = "confluence"
			outletMode[i] = "single"
			maxOutflows[i] = 1
		case channel >= 1.6:
			classes[i] = "trunk"
			outletMode[i] = "single"
			maxOutflows[i] = 1
		case lakeReach:
			classes[i] = "lake_reach"
			outletMode[i] = "single"
			maxOutflows[i] = 1
		case cellInflowCount == 0:
			classes[i] = "headwater"
			outletMode[i] = "single"
			maxOutflows[i] = 1
		default:
			classes[i] = "hillslope"
			outletMode[i] = "single"
			maxOutflows[i] = 1
		}
	}

	return classes, outletMode, maxOutflows
}

func sortBoundaryInflows(flow *HydrologyBoundaryFlow) {
	order := make([]int, len(flow.InflowNeighbors))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i], order[j]
		if flow.InflowStrength[a] == flow.InflowStrength[b] {
			return flow.InflowNeighbors[a] < flow.InflowNeighbors[b]
		}
		return flow.InflowStrength[a] > flow.InflowStrength[b]
	})
	neighbors := make([]int, len(flow.InflowNeighbors))
	bearings := make([]float64, len(flow.InflowBearingDeg))
	strengths := make([]float64, len(flow.InflowStrength))
	for i, src := range order {
		neighbors[i] = flow.InflowNeighbors[src]
		bearings[i] = flow.InflowBearingDeg[src]
		strengths[i] = flow.InflowStrength[src]
	}
	flow.InflowNeighbors = neighbors
	flow.InflowBearingDeg = bearings
	flow.InflowStrength = strengths
}

func boundaryBearingDegrees(origin, target Vector3D) float64 {
	v := origin.Normalize()
	pole := Vector3D{X: 0, Y: 0, Z: 1}
	east := pole.Cross(v)
	if east.LengthSq() < 1e-12 {
		east = Vector3D{X: 1, Y: 0, Z: 0}
	}
	east = east.Normalize()
	north := v.Cross(east).Normalize()

	toTarget := target.Subtract(origin)
	toTarget = toTarget.Subtract(v.Scale(toTarget.Dot(v)))
	if toTarget.LengthSq() < 1e-12 {
		return 0
	}
	toTarget = toTarget.Normalize()
	eastComp := toTarget.Dot(east)
	northComp := toTarget.Dot(north)
	bearing := math.Atan2(eastComp, northComp) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	return bearing
}

func terminalDrainageDestination(
	start int,
	receivers []int,
	elevation []float64,
	waterLabels []int,
	largestWaterLabel int,
	memo []int,
) int {
	if memo[start] != math.MinInt {
		return memo[start]
	}

	visited := make(map[int]bool)
	path := make([]int, 0, 8)
	current := start
	result := -1

	for current >= 0 && current < len(receivers) {
		if memo[current] != math.MinInt {
			result = memo[current]
			break
		}
		if elevation[current] <= 0 {
			if waterLabels[current] >= 0 && waterLabels[current] != largestWaterLabel {
				result = -(waterLabels[current] + 2)
			} else {
				result = -1
			}
			break
		}
		if visited[current] {
			// Loop: treat as an internal sink.
			result = current
			break
		}
		visited[current] = true
		path = append(path, current)

		next := receivers[current]
		if next < 0 {
			result = current
			break
		}
		current = next
	}

	for _, idx := range path {
		memo[idx] = result
	}
	return result
}

func computeInlandLakeCoverage(cells []VoronoiCell, elevation []float64) float64 {
	waterLabels, componentSizes, largestWaterLabel := computeWaterComponentLabels(cells, elevation)
	return computeInlandLakeCoverageFromLabels(elevation, waterLabels, componentSizes, largestWaterLabel)
}

func computeInlandLakeCoverageFromLabels(elevation []float64, waterLabels []int, componentSizes map[int]int, largestWaterLabel int) float64 {
	inlandWaterCells := 0
	for i, elev := range elevation {
		if elev <= 0 && waterLabels[i] >= 0 && waterLabels[i] != largestWaterLabel && componentSizes[waterLabels[i]] >= 3 {
			inlandWaterCells++
		}
	}

	if len(elevation) == 0 {
		return 0
	}
	return float64(inlandWaterCells) / float64(len(elevation))
}

func computeWaterComponentLabels(cells []VoronoiCell, elevation []float64) ([]int, map[int]int, int) {
	visited := make([]bool, len(elevation))
	labels := make([]int, len(elevation))
	for i := range labels {
		labels[i] = -1
	}
	componentSizes := make(map[int]int)
	largestWaterComponent := 0
	largestLabel := -1
	labelID := 0

	for start, elev := range elevation {
		if visited[start] || elev > 0 {
			continue
		}

		queue := []int{start}
		visited[start] = true
		size := 0

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			size++
			labels[current] = labelID

			for _, neighborIdx := range cells[current].NeighborSiteIndices {
				neighbor := int(neighborIdx)
				if neighbor < 0 || neighbor >= len(elevation) || visited[neighbor] || elevation[neighbor] > 0 {
					continue
				}
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}

		componentSizes[labelID] = size
		if size > largestWaterComponent {
			largestWaterComponent = size
			largestLabel = labelID
		}
		labelID++
	}

	return labels, componentSizes, largestLabel
}
