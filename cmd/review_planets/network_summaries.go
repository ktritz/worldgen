package main

import (
	"fmt"
	"math"
	"sort"

	"worldgen/climgen"
)

func printSettlementNetworkSummary(sites []climgen.Vector3D, cells []climgen.VoronoiCell, population *climgen.PopulationResult, result *climgen.SettlementNetworkResult) {
	if result == nil {
		return
	}
	nodeCount := len(result.Nodes)
	linkCount := len(result.Links)
	if nodeCount == 0 {
		fmt.Println("    settlementNetwork: nodes=0 links=0")
		return
	}
	classCounts := make(map[climgen.SettlementNodeKind]int)
	for _, node := range result.Nodes {
		classCounts[node.Kind]++
	}
	linkCosts := make([]float64, 0, len(result.Links))
	degree := make([]int, len(result.Nodes))
	for _, link := range result.Links {
		linkCosts = append(linkCosts, link.TravelCost)
		if link.From >= 0 && link.From < len(degree) {
			degree[link.From]++
		}
		if link.To >= 0 && link.To < len(degree) {
			degree[link.To]++
		}
	}
	nearestCosts := nearestNeighborCosts(result)
	isolated := 0
	for _, d := range degree {
		if d == 0 {
			isolated++
		}
	}
	fmt.Printf(
		"    settlementNetwork: nodes=%d links=%d meanLinkCost=%.2f medianLinkCost=%.2f nearestMean=%.2f nearestMedian=%.2f isolated=%.1f%%\n",
		nodeCount,
		linkCount,
		meanFloat(linkCosts),
		medianFloat(linkCosts),
		meanFloat(nearestCosts),
		medianFloat(nearestCosts),
		100*float64(isolated)/float64(nodeCount),
	)
	printMovementCostDistribution(result.Diagnostics.MovementCost)
	printSettlementLinkPhysicalDiagnostics(sites, result)
	printSettlementLinkSupportDiagnostics(cells, population, result)
	printSettlementHighRankFormationDiagnostics(cells, population, result)
	printSettlementNodePhysicalDiagnostics(sites, result)
	printSettlementNodePhysicalSupportDiagnostics(cells, population, result)

	type nodeClassCount struct {
		kind  climgen.SettlementNodeKind
		count int
	}
	sorted := make([]nodeClassCount, 0, len(classCounts))
	for kind, count := range classCounts {
		sorted = append(sorted, nodeClassCount{kind: kind, count: count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].kind > sorted[j].kind })
	for _, entry := range sorted {
		fmt.Printf("      node[%s]=%d\n", climgen.SettlementNodeKindName(entry.kind), entry.count)
	}
	printSettlementNodeFormationDiagnostics(result.Diagnostics.NodeFormation)
	printSettlementLinkFormationDiagnostics(result.Diagnostics.LinkFormation)
	printSettlementRegionFormationDiagnostics(result.Diagnostics.RegionFormation)
	protoSettings := climgen.DefaultProtoCivilizationSettings()
	networkSettings := reviewSettlementNetworkSettings(result)
	for _, region := range topSettlementRegions(result, 4) {
		center := result.Nodes[region.CenterNode]
		ok, reason := climgen.EligibleProtoCivilizationRegionWithPhysicalSupport(region, result, cells, population, protoSettings, networkSettings)
		rawStrength := climgen.ProtoCivilizationRegionAnchorStrength(region, result)
		physicalStrength := settlementRegionPhysicalAnchorStrength(cells, population, result, region, networkSettings)
		areaSupportStrength := climgen.ProtoCivilizationRegionPopulationSupportStrength(region, result, cells, population, networkSettings)
		supportedRegional, totalRegional := settlementRegionSupportedRegionalAnchors(cells, population, result, region, networkSettings)
		meanExtent, maxExtent := settlementRegionExtentDeg(sites, result, region)
		fmt.Printf(
			"      region[%d]: anchors=%d center=%s coastal=%v river=%v meanScore=%.2f extentMeanDeg=%.2f extentMaxDeg=%.2f protoEligible=%v reason=%s rawStrength=%.1f physicalStrength=%.1f areaSupportStrength=%.1f supportedRegional=%d/%d\n",
			region.ID,
			len(region.NodeIndices),
			climgen.SettlementNodeKindName(center.Kind),
			region.Coastal,
			region.River,
			region.MeanScore,
			meanExtent,
			maxExtent,
			ok,
			reason,
			rawStrength,
			physicalStrength,
			areaSupportStrength,
			supportedRegional,
			totalRegional,
		)
	}
	printSettlementRegionEligibilityDiagnostics(cells, population, result)
	printSettlementProtoRegionDetails(sites, cells, population, result)
}

func printSettlementNodePhysicalSupportDiagnostics(cells []climgen.VoronoiCell, population *climgen.PopulationResult, result *climgen.SettlementNetworkResult) {
	if result == nil || len(result.Nodes) == 0 || len(cells) == 0 || population == nil {
		return
	}
	settings := reviewSettlementNetworkSettings(result)
	for kind := climgen.SettlementNodeCity; kind >= climgen.SettlementNodeHamlet; kind-- {
		values := make([]float64, 0)
		for _, node := range result.Nodes {
			if node.Kind != kind {
				continue
			}
			area := climgen.SettlementNodePhysicalSupportArea(node.CellIndex, node.Kind, cells, population, settings)
			if area > 0 {
				values = append(values, area)
			}
		}
		if len(values) == 0 {
			continue
		}
		sort.Float64s(values)
		fmt.Printf(
			"      nodePhysicalSupport[%s]: count=%d meanArea=%.2f p50Area=%.2f p10Area=%.2f below1=%.1f%%\n",
			climgen.SettlementNodeKindName(kind),
			len(values),
			meanFloat(values),
			percentileFloat64(values, 0.50),
			percentileFloat64(values, 0.10),
			100*shareBelow(values, 1.0),
		)
	}
}

func printSettlementNodePhysicalDiagnostics(sites []climgen.Vector3D, result *climgen.SettlementNetworkResult) {
	if result == nil || len(result.Nodes) < 2 || len(sites) == 0 {
		return
	}
	for kind := climgen.SettlementNodeCity; kind >= climgen.SettlementNodeHamlet; kind-- {
		nearestSameOrHigher := nearestNodeDistancesByKind(sites, result, kind, true)
		if len(nearestSameOrHigher) == 0 {
			continue
		}
		sort.Float64s(nearestSameOrHigher)
		fmt.Printf(
			"      nodePhysicalSpacing[%s:sameOrHigher]: count=%d meanDeg=%.2f p50Deg=%.2f p90Deg=%.2f\n",
			climgen.SettlementNodeKindName(kind),
			len(nearestSameOrHigher),
			meanFloat(nearestSameOrHigher),
			percentileFloat64(nearestSameOrHigher, 0.50),
			percentileFloat64(nearestSameOrHigher, 0.90),
		)
	}
}

func nearestNodeDistancesByKind(sites []climgen.Vector3D, result *climgen.SettlementNetworkResult, kind climgen.SettlementNodeKind, sameOrHigher bool) []float64 {
	out := make([]float64, 0)
	for i, node := range result.Nodes {
		if node.Kind != kind || node.CellIndex < 0 || node.CellIndex >= len(sites) {
			continue
		}
		best := math.Inf(1)
		for j, other := range result.Nodes {
			if i == j || other.CellIndex < 0 || other.CellIndex >= len(sites) {
				continue
			}
			if sameOrHigher && other.Kind < kind {
				continue
			}
			d := greatCircleDistanceDegForReview(sites[node.CellIndex], sites[other.CellIndex])
			if d < best {
				best = d
			}
		}
		if !math.IsInf(best, 1) {
			out = append(out, best)
		}
	}
	return out
}

func printSettlementLinkPhysicalDiagnostics(sites []climgen.Vector3D, result *climgen.SettlementNetworkResult) {
	if result == nil || len(result.Links) == 0 || len(sites) == 0 {
		return
	}
	pathLengths := make([]float64, 0, len(result.Links))
	directLengths := make([]float64, 0, len(result.Links))
	tortuosities := make([]float64, 0, len(result.Links))
	costPerDeg := make([]float64, 0, len(result.Links))
	for _, link := range result.Links {
		pathLength := settlementPathLengthDegForReview(link.Path, sites)
		if pathLength <= 0 || link.From < 0 || link.From >= len(result.Nodes) || link.To < 0 || link.To >= len(result.Nodes) {
			continue
		}
		fromCell := result.Nodes[link.From].CellIndex
		toCell := result.Nodes[link.To].CellIndex
		if fromCell < 0 || fromCell >= len(sites) || toCell < 0 || toCell >= len(sites) {
			continue
		}
		directLength := greatCircleDistanceDegForReview(sites[fromCell], sites[toCell])
		if directLength <= 0 {
			continue
		}
		pathLengths = append(pathLengths, pathLength)
		directLengths = append(directLengths, directLength)
		tortuosities = append(tortuosities, pathLength/directLength)
		costPerDeg = append(costPerDeg, link.TravelCost/pathLength)
	}
	if len(pathLengths) == 0 {
		return
	}
	sort.Float64s(pathLengths)
	sort.Float64s(directLengths)
	sort.Float64s(tortuosities)
	sort.Float64s(costPerDeg)
	fmt.Printf(
		"      linkPhysicalDist: pathMeanDeg=%.2f pathP50Deg=%.2f pathP90Deg=%.2f directMeanDeg=%.2f directP50Deg=%.2f tortuosityP50=%.2f costPerDegP50=%.2f\n",
		meanFloat(pathLengths),
		percentileFloat64(pathLengths, 0.50),
		percentileFloat64(pathLengths, 0.90),
		meanFloat(directLengths),
		percentileFloat64(directLengths, 0.50),
		percentileFloat64(tortuosities, 0.50),
		percentileFloat64(costPerDeg, 0.50),
	)
}

func printSettlementLinkSupportDiagnostics(cells []climgen.VoronoiCell, population *climgen.PopulationResult, result *climgen.SettlementNetworkResult) {
	if result == nil || len(result.Links) == 0 || len(cells) == 0 || population == nil {
		return
	}
	settings := reviewSettlementNetworkSettings(result)
	weakHighEndpointLinks := 0
	highEndpointLinks := 0
	minSupportBelow025 := 0
	minSupportBelow050 := 0
	minSupportBelow075 := 0
	var weakPairLinks [4][4]int
	var tinyPairLinks [4][4]int
	var totalPairLinks [4][4]int
	for _, link := range result.Links {
		if link.From < 0 || link.From >= len(result.Nodes) || link.To < 0 || link.To >= len(result.Nodes) {
			continue
		}
		from := result.Nodes[link.From]
		to := result.Nodes[link.To]
		recordReviewPairKind(&totalPairLinks, from.Kind, to.Kind)
		weakHigh := false
		hasHigh := false
		minHighSupport := math.Inf(1)
		for _, node := range []climgen.SettlementNode{from, to} {
			if node.Kind < climgen.SettlementNodeTown {
				continue
			}
			hasHigh = true
			support := climgen.SettlementNodePhysicalSupportArea(node.CellIndex, node.Kind, cells, population, settings)
			if support < minHighSupport {
				minHighSupport = support
			}
			if support < 0.5 {
				weakHigh = true
			}
		}
		if hasHigh {
			highEndpointLinks++
			if minHighSupport < 0.25 {
				minSupportBelow025++
				recordReviewPairKind(&tinyPairLinks, from.Kind, to.Kind)
			}
			if minHighSupport < 0.5 {
				minSupportBelow050++
			}
			if minHighSupport < 0.75 {
				minSupportBelow075++
			}
		}
		if weakHigh {
			weakHighEndpointLinks++
			recordReviewPairKind(&weakPairLinks, from.Kind, to.Kind)
		}
	}
	if highEndpointLinks == 0 {
		return
	}
	fmt.Printf(
		"      linkSupportDiag: highEndpointLinks=%d weakHighEndpointLinks=%d weakHighEndpointShare=%.1f%% minHighSupportBelow025=%d minHighSupportBelow050=%d minHighSupportBelow075=%d\n",
		highEndpointLinks,
		weakHighEndpointLinks,
		percentFromCount(weakHighEndpointLinks, highEndpointLinks),
		minSupportBelow025,
		minSupportBelow050,
		minSupportBelow075,
	)
	printSettlementPairKindTotals("linkTotalSupportPairs", totalPairLinks)
	printSettlementPairKindTotals("linkTinyHighSupportPairs", tinyPairLinks)
	printSettlementPairKindTotals("linkWeakHighSupportPairs", weakPairLinks)
}

func printSettlementHighRankFormationDiagnostics(cells []climgen.VoronoiCell, population *climgen.PopulationResult, result *climgen.SettlementNetworkResult) {
	if len(cells) == 0 || population == nil || population.Diagnostics == nil {
		return
	}
	settings := reviewSettlementNetworkSettings(result)
	classified := settlementHighRankSupportStats{}
	peaks := settlementHighRankSupportStats{}
	for idx := range cells {
		kind, ok := climgenClassifySettlementNodeCandidateAt(idx, cells, population, settings)
		if !ok || kind < climgen.SettlementNodeTown {
			continue
		}
		score := clamp01ForReview(0.58*population.Diagnostics.CarryingCapacity[idx] + 0.42*population.Diagnostics.UrbanPotential[idx])
		support := climgen.SettlementNodePhysicalSupportArea(idx, kind, cells, population, settings)
		classified.add(support)
		if isLocalSettlementPeakForReview(idx, score, cells, population) {
			peaks.add(support)
		}
	}
	printSettlementHighRankSupportStats("classifiedHighRankSupport", classified, len(cells))
	printSettlementHighRankSupportStats("peakHighRankSupport", peaks, len(cells))
}

type settlementHighRankSupportStats struct {
	count   int
	sum     float64
	below25 int
	below50 int
	below75 int
}

func (s *settlementHighRankSupportStats) add(support float64) {
	s.count++
	s.sum += support
	if support < 0.25 {
		s.below25++
	}
	if support < 0.5 {
		s.below50++
	}
	if support < 0.75 {
		s.below75++
	}
}

func printSettlementHighRankSupportStats(label string, stats settlementHighRankSupportStats, cellCount int) {
	if stats.count == 0 {
		return
	}
	fmt.Printf(
		"      %s: count=%d eqArea=%.1f meanSupport=%.2f below025=%.1f%% below050=%.1f%% below075=%.1f%%\n",
		label,
		stats.count,
		meshScaledTerritoryAreaCellsForReview(stats.count, cellCount),
		stats.sum/float64(stats.count),
		percentFromCount(stats.below25, stats.count),
		percentFromCount(stats.below50, stats.count),
		percentFromCount(stats.below75, stats.count),
	)
}

func meshScaledTerritoryAreaCellsForReview(territoryCells int, meshCellCount int) float64 {
	if territoryCells <= 0 {
		return 0
	}
	return climgen.MeshScaledTerritoryAreaCells(territoryCells, meshCellCount)
}

// reviewSettlementNetworkSettings pins the review-side copies of the
// classification rules to the kind thresholds the network was actually built
// with, so a quantile-calibrated mesh is not re-measured against the absolute
// defaults.
func reviewSettlementNetworkSettings(result *climgen.SettlementNetworkResult) climgen.SettlementNetworkSettings {
	return climgen.SettlementNetworkSettingsWithKindThresholds(climgen.DefaultSettlementNetworkSettings(), result)
}

func climgenClassifySettlementNodeCandidateAt(idx int, cells []climgen.VoronoiCell, population *climgen.PopulationResult, settings climgen.SettlementNetworkSettings) (climgen.SettlementNodeKind, bool) {
	carrying := population.Diagnostics.CarryingCapacity[idx]
	urban := population.Diagnostics.UrbanPotential[idx]
	cityUrban := climgen.SettlementUrbanKindThreshold(climgen.SettlementNodeCity, settings)
	townUrban := climgen.SettlementUrbanKindThreshold(climgen.SettlementNodeTown, settings)
	townCarrying := climgen.SettlementCarryingKindThreshold(climgen.SettlementNodeTown, settings)
	switch {
	case urban >= cityUrban && carrying >= townCarrying:
		return climgen.SettlementNodeCity, true
	case urban >= townUrban || carrying >= townCarrying:
		return climgen.SettlementNodeTown, true
	case carrying >= climgen.SettlementCarryingKindThreshold(climgen.SettlementNodeVillage, settings):
		return climgen.SettlementNodeVillage, true
	case carrying >= climgen.SettlementCarryingKindThreshold(climgen.SettlementNodeHamlet, settings):
		return climgen.SettlementNodeHamlet, true
	default:
		return 0, false
	}
}

func isLocalSettlementPeakForReview(idx int, score float64, cells []climgen.VoronoiCell, population *climgen.PopulationResult) bool {
	if score <= 0 {
		return false
	}
	peakRadius := meshResolutionAdjustedStepsForReview(1, len(cells))
	for _, j := range cellsWithinHopsForReview(cells, idx, peakRadius) {
		if j < 0 || j >= len(population.Diagnostics.CarryingCapacity) {
			continue
		}
		neighborScore := clamp01ForReview(0.58*population.Diagnostics.CarryingCapacity[j] + 0.42*population.Diagnostics.UrbanPotential[j])
		if neighborScore > score+0.02 {
			return false
		}
	}
	return true
}

func meshResolutionAdjustedStepsForReview(baseSteps int, cellCount int) int {
	return climgen.MeshResolutionAdjustedSteps(baseSteps, cellCount)
}

func cellsWithinHopsForReview(cells []climgen.VoronoiCell, start int, radius int) []int {
	if radius <= 0 || start < 0 || start >= len(cells) {
		return nil
	}
	visited := map[int]struct{}{start: {}}
	frontier := []int{start}
	out := make([]int, 0)
	for step := 0; step < radius; step++ {
		next := make([]int, 0)
		for _, idx := range frontier {
			for _, neighbor := range cells[idx].NeighborSiteIndices {
				j := int(neighbor)
				if j < 0 || j >= len(cells) {
					continue
				}
				if _, ok := visited[j]; ok {
					continue
				}
				visited[j] = struct{}{}
				out = append(out, j)
				next = append(next, j)
			}
		}
		frontier = next
		if len(frontier) == 0 {
			break
		}
	}
	return out
}

func clamp01ForReview(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func recordReviewPairKind(matrix *[4][4]int, a, b climgen.SettlementNodeKind) {
	if matrix == nil || int(a) < 0 || int(a) >= len(matrix) || int(b) < 0 || int(b) >= len(matrix) {
		return
	}
	x, y := int(a), int(b)
	if x > y {
		x, y = y, x
	}
	matrix[x][y]++
}

func settlementPathLengthDegForReview(path []int, sites []climgen.Vector3D) float64 {
	if len(path) < 2 || len(sites) == 0 {
		return 0
	}
	total := 0.0
	for i := 1; i < len(path); i++ {
		prev := path[i-1]
		cur := path[i]
		if prev < 0 || prev >= len(sites) || cur < 0 || cur >= len(sites) {
			continue
		}
		total += greatCircleDistanceDegForReview(sites[prev], sites[cur])
	}
	return total
}

func greatCircleDistanceDegForReview(a, b climgen.Vector3D) float64 {
	dot := a.X*b.X + a.Y*b.Y + a.Z*b.Z
	if dot > 1 {
		dot = 1
	}
	if dot < -1 {
		dot = -1
	}
	return math.Acos(dot) * 180 / math.Pi
}

func printMovementCostDistribution(values []float64) {
	finite := make([]float64, 0, len(values))
	for _, value := range values {
		if !math.IsInf(value, 1) && !math.IsNaN(value) && value < 1e6 {
			finite = append(finite, value)
		}
	}
	if len(finite) == 0 {
		return
	}
	sort.Float64s(finite)
	fmt.Printf(
		"      movementCostDist: mean=%.2f p50=%.2f p90=%.2f p95=%.2f p99=%.2f max=%.2f\n",
		meanFloat(finite),
		percentileFloat64(finite, 0.50),
		percentileFloat64(finite, 0.90),
		percentileFloat64(finite, 0.95),
		percentileFloat64(finite, 0.99),
		finite[len(finite)-1],
	)
}

func printSettlementLinkFormationDiagnostics(diag climgen.SettlementLinkFormationDiagnostics) {
	if diag.SourceNodes == 0 {
		return
	}
	meanReachable := float64(diag.ReachableTargets) / float64(diag.SourceNodes)
	meanSelected := float64(diag.SelectedTargets) / float64(diag.SourceNodes)
	fmt.Printf(
		"      linkFormation: sources=%d reachable=%d meanReachable=%.2f near10=%d selected=%d meanSelected=%.2f created=%d duplicates=%d noReachable=%d noSelected=%d targetLimited=%d\n",
		diag.SourceNodes,
		diag.ReachableTargets,
		meanReachable,
		diag.NearTargets,
		diag.SelectedTargets,
		meanSelected,
		diag.CreatedLinks,
		diag.DuplicateSelections,
		diag.NoReachableSources,
		diag.NoSelectedSources,
		diag.TargetLimitedNodes,
	)
	printSettlementKindFormation("linkSources", diag.SourceByKind)
	printSettlementKindFormation("linkNoReachable", diag.NoReachableByKind)
	printSettlementKindTotals("linkReachableTargets", diag.ReachableByKind)
	printSettlementKindTotals("linkReachableTargetKinds", diag.ReachableTargetKind)
	printSettlementKindTotals("linkNear10Targets", diag.NearByKind)
	printSettlementKindTotals("linkNear10TargetKinds", diag.NearTargetKind)
	printSettlementKindTotals("linkSelectedTargets", diag.SelectedByKind)
	printSettlementKindTotals("linkSelectedTargetKinds", diag.SelectedTargetKind)
	printSettlementPairKindTotals("linkSelectedPairs", diag.SelectedPairKind)
	printSettlementPairKindTotals("linkCreatedPairs", diag.CreatedPairKind)
}

func printSettlementRegionFormationDiagnostics(diag climgen.RegionFormationDiagnostics) {
	if diag.TransportLinks == 0 && diag.PhysicalClusterLinks == 0 && diag.RegionCount == 0 {
		return
	}
	fmt.Printf(
		"      regionFormation: transportLinks=%d physicalClusterLinks=%d physicalReachable=%d physicalAlreadyLinked=%d physicalSkippedTransport=%d skippedSameComponent=%d skippedCrossComponent=%d physicalSingletonCandidates=%d regions=%d\n",
		diag.TransportLinks,
		diag.PhysicalClusterLinks,
		diag.PhysicalReachablePairs,
		diag.PhysicalAlreadyLinkedPairs,
		diag.PhysicalSkippedTransportConnectedPairs,
		diag.PhysicalSkippedSameComponentPairs,
		diag.PhysicalSkippedCrossComponentPairs,
		diag.PhysicalSingletonCandidatePairs,
		diag.RegionCount,
	)
}

func printSettlementNodeFormationDiagnostics(diag climgen.SettlementNodeFormationDiagnostics) {
	if diag.LandCells == 0 && diag.ClassifiedCount == 0 && diag.FinalCount == 0 {
		return
	}
	fmt.Printf(
		"      nodeFormation: land=%d classified=%d candidates=%d spacingKept=%d spacingRejected=%d waystations=%d prePrune=%d pruned=%d final=%d peakRejected=%d classRejected=%d\n",
		diag.LandCells,
		diag.ClassifiedCount,
		diag.CandidateCount,
		diag.SpacingKept,
		diag.SpacingRejected,
		diag.WaystationsAdded,
		diag.PrePruneCount,
		diag.PrunedCount,
		diag.FinalCount,
		diag.PeakRejected,
		diag.SettlementClassRejected,
	)
	fmt.Printf(
		"      nodeFormation.funnel: thresholdEligible=%d supportRejected=%d supportDowngraded=%d classified=%d peakRejected=%d peakRejectedUnscaledPass=%d peakRejectedRankPass=%d peakRadiusHops=%d peakDiscCells=%.1f\n",
		diag.ThresholdEligible,
		diag.SupportRejected,
		diag.SupportDowngraded,
		diag.ClassifiedCount,
		diag.PeakRejected,
		diag.PeakRejectedUnscaledPass,
		diag.PeakRejectedRankPass,
		diag.PeakRadiusHops,
		diag.PeakDiscCellsMean,
	)
	printSettlementFieldDistribution("carryingCapacity", diag.CarryingDistribution)
	printSettlementFieldDistribution("urbanPotential", diag.UrbanDistribution)
	printSettlementKindFormation("thresholdEligible", diag.ThresholdEligibleByKind)
	printSettlementKindFormation("peakRejected", diag.PeakRejectedByKind)
	printSettlementKindFormation("classified", diag.ClassifiedByKind)
	printSettlementKindFormation("candidates", diag.CandidateByKind)
	printSettlementKindFormation("spacingKept", diag.SpacingKeptByKind)
	printSettlementKindFormation("spacingRejected", diag.SpacingRejectedByKind)
	printSettlementKindFormation("spacingBlocker", diag.SpacingBlockerByKind)
	printSettlementPairKindTotals("spacingRejectedPairs", diag.SpacingRejectedPairKind)
	printSettlementKindFloatTotals("spacingKeptSupportArea", diag.SpacingKeptSupportArea)
	printSettlementKindFloatTotals("spacingRejectedSupportArea", diag.SpacingRejectSupportArea)
	printSettlementKindFormation("final", diag.FinalByKind)
}

func printSettlementFieldDistribution(label string, dist climgen.SettlementFieldDistribution) {
	if dist.Count == 0 {
		return
	}
	fmt.Printf(
		"      nodeFormation[dist:%s]: n=%d mean=%.4f p50=%.4f p90=%.4f p99=%.4f max=%.4f aboveHamlet=%.4f aboveVillage=%.4f aboveTown=%.4f aboveCity=%.4f\n",
		label,
		dist.Count,
		dist.Mean,
		dist.P50,
		dist.P90,
		dist.P99,
		dist.Max,
		dist.AboveThresholdFraction[climgen.SettlementNodeHamlet],
		dist.AboveThresholdFraction[climgen.SettlementNodeVillage],
		dist.AboveThresholdFraction[climgen.SettlementNodeTown],
		dist.AboveThresholdFraction[climgen.SettlementNodeCity],
	)
	fmt.Printf(
		"      nodeFormation[thresholds:%s]: hamlet=%.4f village=%.4f town=%.4f city=%.4f\n",
		label,
		dist.Thresholds[climgen.SettlementNodeHamlet],
		dist.Thresholds[climgen.SettlementNodeVillage],
		dist.Thresholds[climgen.SettlementNodeTown],
		dist.Thresholds[climgen.SettlementNodeCity],
	)
}

func printSettlementKindFormation(label string, counts [4]int) {
	total := 0
	for _, count := range counts {
		total += count
	}
	if total == 0 {
		return
	}
	for i := len(counts) - 1; i >= 0; i-- {
		if counts[i] == 0 {
			continue
		}
		fmt.Printf("      nodeFormation[%s:%s]=%d\n", label, climgen.SettlementNodeKindName(climgen.SettlementNodeKind(i)), counts[i])
	}
}

func printSettlementPairKindTotals(label string, counts [4][4]int) {
	total := 0
	for i := range counts {
		for _, count := range counts[i] {
			total += count
		}
	}
	if total == 0 {
		return
	}
	fmt.Printf("      nodeFormation[%s]:", label)
	for i := len(counts) - 1; i >= 0; i-- {
		for j := len(counts[i]) - 1; j >= i; j-- {
			count := counts[i][j]
			if count == 0 {
				continue
			}
			fmt.Printf(" %s-%s=%d", climgen.SettlementNodeKindName(climgen.SettlementNodeKind(i)), climgen.SettlementNodeKindName(climgen.SettlementNodeKind(j)), count)
		}
	}
	fmt.Println()
}

func printSettlementKindTotals(label string, counts [4]int) {
	total := 0
	for _, count := range counts {
		total += count
	}
	if total == 0 {
		return
	}
	for i := len(counts) - 1; i >= 0; i-- {
		if counts[i] == 0 {
			continue
		}
		fmt.Printf("      nodeFormation[%s:%s]=%d\n", label, climgen.SettlementNodeKindName(climgen.SettlementNodeKind(i)), counts[i])
	}
}

func printSettlementKindFloatTotals(label string, values [4]float64) {
	total := 0.0
	for _, value := range values {
		total += value
	}
	if total <= 0 {
		return
	}
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] <= 0 {
			continue
		}
		fmt.Printf("      nodeFormation[%s:%s]=%.1f\n", label, climgen.SettlementNodeKindName(climgen.SettlementNodeKind(i)), values[i])
	}
}

func printSettlementRegionEligibilityDiagnostics(cells []climgen.VoronoiCell, population *climgen.PopulationResult, result *climgen.SettlementNetworkResult) {
	if result == nil || len(result.Regions) == 0 {
		return
	}
	settings := climgen.DefaultProtoCivilizationSettings()
	networkSettings := reviewSettlementNetworkSettings(result)
	eligible := 0
	eligibleAnchors := 0
	outpostAnchors := 0
	eligibleStrength := 0.0
	outpostStrength := 0.0
	eligibleRegionalSupport := 0.0
	outpostRegionalSupport := 0.0
	eligibleRegionalSupportCount := 0
	outpostRegionalSupportCount := 0
	eligibleLowRegionalSupport := 0
	outpostLowRegionalSupport := 0
	eligiblePhysicalStrength := 0.0
	outpostPhysicalStrength := 0.0
	coastalEligible := 0
	riverEligible := 0
	reasonCounts := make(map[string]int)
	outpostReasonCounts := make(map[string]int)
	anchorCounts := make([]float64, 0, len(result.Regions))
	for _, region := range result.Regions {
		anchors := len(region.NodeIndices)
		anchorCounts = append(anchorCounts, float64(anchors))
		ok, reason := climgen.EligibleProtoCivilizationRegionWithPhysicalSupport(region, result, cells, population, settings, networkSettings)
		strength := climgen.ProtoCivilizationRegionAnchorStrength(region, result)
		physicalStrength := settlementRegionPhysicalAnchorStrength(cells, population, result, region, networkSettings)
		regionalSupport, regionalCount, lowRegionalCount := settlementRegionRegionalSupport(cells, population, result, region, networkSettings)
		if ok {
			eligible++
			eligibleAnchors += anchors
			eligibleStrength += strength
			eligiblePhysicalStrength += physicalStrength
			eligibleRegionalSupport += regionalSupport
			eligibleRegionalSupportCount += regionalCount
			eligibleLowRegionalSupport += lowRegionalCount
			reasonCounts[reason]++
			if region.Coastal {
				coastalEligible++
			}
			if region.River {
				riverEligible++
			}
			continue
		}
		outpostAnchors += anchors
		outpostStrength += strength
		outpostPhysicalStrength += physicalStrength
		outpostRegionalSupport += regionalSupport
		outpostRegionalSupportCount += regionalCount
		outpostLowRegionalSupport += lowRegionalCount
		outpostReasonCounts[reason]++
	}
	eligibleMeanAnchors := 0.0
	if eligible > 0 {
		eligibleMeanAnchors = float64(eligibleAnchors) / float64(eligible)
		eligibleStrength /= float64(eligible)
		eligiblePhysicalStrength /= float64(eligible)
	}
	outpostCount := len(result.Regions) - eligible
	outpostMeanAnchors := 0.0
	if outpostCount > 0 {
		outpostMeanAnchors = float64(outpostAnchors) / float64(outpostCount)
		outpostStrength /= float64(outpostCount)
		outpostPhysicalStrength /= float64(outpostCount)
	}
	fmt.Printf(
		"      settlementRegionDiag: regions=%d protoEligible=%d outpostLike=%d meanAnchors=%.1f medianAnchors=%.1f eligibleMeanAnchors=%.1f outpostMeanAnchors=%.1f eligibleMeanStrength=%.1f outpostMeanStrength=%.1f eligiblePhysicalStrength=%.1f outpostPhysicalStrength=%.1f eligibleCoastal=%d eligibleRiver=%d\n",
		len(result.Regions),
		eligible,
		outpostCount,
		meanFloat(anchorCounts),
		medianFloat(anchorCounts),
		eligibleMeanAnchors,
		outpostMeanAnchors,
		eligibleStrength,
		outpostStrength,
		eligiblePhysicalStrength,
		outpostPhysicalStrength,
		coastalEligible,
		riverEligible,
	)
	if eligibleRegionalSupportCount > 0 || outpostRegionalSupportCount > 0 {
		fmt.Printf(
			"      settlementRegionSupportDiag: eligibleRegionalMeanArea=%.2f eligibleRegionalLow=%.1f%% outpostRegionalMeanArea=%.2f outpostRegionalLow=%.1f%%\n",
			meanFromSum(eligibleRegionalSupport, eligibleRegionalSupportCount),
			percentFromCount(eligibleLowRegionalSupport, eligibleRegionalSupportCount),
			meanFromSum(outpostRegionalSupport, outpostRegionalSupportCount),
			percentFromCount(outpostLowRegionalSupport, outpostRegionalSupportCount),
		)
	}
	for _, reason := range sortedCountKeys(reasonCounts) {
		fmt.Printf("      protoEligibility[%s]=%d\n", reason, reasonCounts[reason])
	}
	for _, reason := range sortedCountKeys(outpostReasonCounts) {
		fmt.Printf("      protoOutpostReason[%s]=%d\n", reason, outpostReasonCounts[reason])
	}
}

func printSettlementProtoRegionDetails(sites []climgen.Vector3D, cells []climgen.VoronoiCell, population *climgen.PopulationResult, result *climgen.SettlementNetworkResult) {
	if result == nil || len(result.Regions) == 0 {
		return
	}
	protoSettings := climgen.DefaultProtoCivilizationSettings()
	// Must pin to the thresholds the run actually resolved. Using the defaults
	// here evaluates every gate at the L5 cut points regardless of mesh, which
	// would show a threshold-induced L6/L7 difference that never happened.
	networkSettings := reviewSettlementNetworkSettings(result)
	details := make([]regionDetail, 0, len(result.Regions))
	for _, region := range result.Regions {
		if region.CenterNode < 0 || region.CenterNode >= len(result.Nodes) {
			continue
		}
		ok, reason := climgen.EligibleProtoCivilizationRegionWithPhysicalSupport(region, result, cells, population, protoSettings, networkSettings)
		rawStrength := climgen.ProtoCivilizationRegionAnchorStrength(region, result)
		physicalStrength := settlementRegionPhysicalAnchorStrength(cells, population, result, region, networkSettings)
		areaSupportStrength := climgen.ProtoCivilizationRegionPopulationSupportStrength(region, result, cells, population, networkSettings)
		supportedRegional, totalRegional := settlementRegionSupportedRegionalAnchors(cells, population, result, region, networkSettings)
		center := result.Nodes[region.CenterNode]
		centerLat, centerLon := reviewLatLonDeg(sites, center.CellIndex)
		meanExtent, maxExtent := settlementRegionExtentDeg(sites, result, region)
		centerSupport := 0.0
		if len(cells) > 0 && population != nil {
			centerSupport = climgen.SettlementNodePhysicalSupportArea(center.CellIndex, center.Kind, cells, population, networkSettings)
		}
		var kindCounts [4]int
		for _, nodeIdx := range region.NodeIndices {
			if nodeIdx < 0 || nodeIdx >= len(result.Nodes) {
				continue
			}
			kind := result.Nodes[nodeIdx].Kind
			if int(kind) >= 0 && int(kind) < len(kindCounts) {
				kindCounts[kind]++
			}
		}
		details = append(details, regionDetail{
			region:              region,
			eligible:            ok,
			reason:              reason,
			rawStrength:         rawStrength,
			physicalStrength:    physicalStrength,
			areaSupportStrength: areaSupportStrength,
			supportedRegional:   supportedRegional,
			totalRegional:       totalRegional,
			centerSupport:       centerSupport,
			meanExtentDeg:       meanExtent,
			maxExtentDeg:        maxExtent,
			centerLatDeg:        centerLat,
			centerLonDeg:        centerLon,
			kindCounts:          kindCounts,
		})
	}
	sort.Slice(details, func(i, j int) bool {
		if details[i].eligible != details[j].eligible {
			return details[i].eligible
		}
		if details[i].reason != details[j].reason {
			return details[i].reason < details[j].reason
		}
		if details[i].region.Coastal != details[j].region.Coastal {
			return details[i].region.Coastal
		}
		if len(details[i].region.NodeIndices) != len(details[j].region.NodeIndices) {
			return len(details[i].region.NodeIndices) > len(details[j].region.NodeIndices)
		}
		return details[i].region.MeanScore > details[j].region.MeanScore
	})
	limit := 10
	if len(details) < limit {
		limit = len(details)
	}
	for i := 0; i < limit; i++ {
		printProtoRegionDetail("protoRegionDetail", details[i], result)
	}
	weakLimit := 6
	weakPrinted := 0
	minPhysicalStrength := math.Max(1, float64(protoSettings.MinRegionAnchors)-0.5)
	for _, detail := range details {
		if detail.physicalStrength <= 0 || detail.rawStrength < minPhysicalStrength || detail.physicalStrength >= minPhysicalStrength {
			continue
		}
		if weakPrinted >= weakLimit {
			break
		}
		printProtoRegionDetail("protoWeakPhysicalDetail", detail, result)
		weakPrinted++
	}
}

type regionDetail struct {
	region              climgen.SettlementRegion
	eligible            bool
	reason              string
	rawStrength         float64
	physicalStrength    float64
	areaSupportStrength float64
	supportedRegional   int
	totalRegional       int
	centerSupport       float64
	meanExtentDeg       float64
	maxExtentDeg        float64
	centerLatDeg        float64
	centerLonDeg        float64
	kindCounts          [4]int
}

func printProtoRegionDetail(label string, detail regionDetail, result *climgen.SettlementNetworkResult) {
	center := result.Nodes[detail.region.CenterNode]
	fmt.Printf(
		"      %s[%d]: eligible=%v reason=%s anchors=%d kinds=hub:%d regional:%d district:%d local:%d center=%s coastal=%v river=%v meanScore=%.2f centerLat=%.2f centerLon=%.2f extentMeanDeg=%.2f extentMaxDeg=%.2f rawStrength=%.1f physicalStrength=%.1f areaSupportStrength=%.1f supportedRegional=%d/%d centerSupport=%.2f centerCell=%d\n",
		label,
		detail.region.ID,
		detail.eligible,
		detail.reason,
		len(detail.region.NodeIndices),
		detail.kindCounts[climgen.SettlementNodeCity],
		detail.kindCounts[climgen.SettlementNodeTown],
		detail.kindCounts[climgen.SettlementNodeVillage],
		detail.kindCounts[climgen.SettlementNodeHamlet],
		climgen.SettlementNodeKindName(center.Kind),
		detail.region.Coastal,
		detail.region.River,
		detail.region.MeanScore,
		detail.centerLatDeg,
		detail.centerLonDeg,
		detail.meanExtentDeg,
		detail.maxExtentDeg,
		detail.rawStrength,
		detail.physicalStrength,
		detail.areaSupportStrength,
		detail.supportedRegional,
		detail.totalRegional,
		detail.centerSupport,
		center.CellIndex,
	)
}

func reviewLatLonDeg(sites []climgen.Vector3D, cellIdx int) (float64, float64) {
	if cellIdx < 0 || cellIdx >= len(sites) {
		return 0, 0
	}
	v := sites[cellIdx]
	return math.Asin(v.Y) * 180 / math.Pi, math.Atan2(v.Z, v.X) * 180 / math.Pi
}

func settlementRegionExtentDeg(sites []climgen.Vector3D, result *climgen.SettlementNetworkResult, region climgen.SettlementRegion) (float64, float64) {
	if len(sites) == 0 || result == nil || region.CenterNode < 0 || region.CenterNode >= len(result.Nodes) {
		return 0, 0
	}
	centerCell := result.Nodes[region.CenterNode].CellIndex
	if centerCell < 0 || centerCell >= len(sites) {
		return 0, 0
	}
	sum := 0.0
	maxDist := 0.0
	count := 0
	for _, nodeIdx := range region.NodeIndices {
		if nodeIdx < 0 || nodeIdx >= len(result.Nodes) {
			continue
		}
		cellIdx := result.Nodes[nodeIdx].CellIndex
		if cellIdx < 0 || cellIdx >= len(sites) {
			continue
		}
		dist := greatCircleDistanceDegForReview(sites[centerCell], sites[cellIdx])
		sum += dist
		if dist > maxDist {
			maxDist = dist
		}
		count++
	}
	if count == 0 {
		return 0, maxDist
	}
	return sum / float64(count), maxDist
}

func settlementRegionPhysicalAnchorStrength(
	cells []climgen.VoronoiCell,
	population *climgen.PopulationResult,
	result *climgen.SettlementNetworkResult,
	region climgen.SettlementRegion,
	settings climgen.SettlementNetworkSettings,
) float64 {
	if len(cells) == 0 || population == nil || result == nil {
		return 0
	}
	return climgen.ProtoCivilizationRegionPhysicalAnchorStrength(region, result, cells, population, settings)
}

func settlementRegionSupportedRegionalAnchors(
	cells []climgen.VoronoiCell,
	population *climgen.PopulationResult,
	result *climgen.SettlementNetworkResult,
	region climgen.SettlementRegion,
	settings climgen.SettlementNetworkSettings,
) (int, int) {
	if len(cells) == 0 || population == nil || result == nil {
		return 0, 0
	}
	supported := 0
	total := 0
	for _, nodeIdx := range region.NodeIndices {
		if nodeIdx < 0 || nodeIdx >= len(result.Nodes) {
			continue
		}
		node := result.Nodes[nodeIdx]
		if node.Kind < climgen.SettlementNodeTown {
			continue
		}
		total++
		if climgen.SettlementNodePhysicalSupportArea(node.CellIndex, node.Kind, cells, population, settings) >= 0.5 {
			supported++
		}
	}
	return supported, total
}

func settlementRegionRegionalSupport(
	cells []climgen.VoronoiCell,
	population *climgen.PopulationResult,
	result *climgen.SettlementNetworkResult,
	region climgen.SettlementRegion,
	settings climgen.SettlementNetworkSettings,
) (float64, int, int) {
	if len(cells) == 0 || population == nil {
		return 0, 0, 0
	}
	sum := 0.0
	count := 0
	low := 0
	for _, nodeIdx := range region.NodeIndices {
		if nodeIdx < 0 || nodeIdx >= len(result.Nodes) {
			continue
		}
		node := result.Nodes[nodeIdx]
		if node.Kind < climgen.SettlementNodeTown {
			continue
		}
		area := climgen.SettlementNodePhysicalSupportArea(node.CellIndex, node.Kind, cells, population, settings)
		sum += area
		count++
		if area < 0.5 {
			low++
		}
	}
	return sum, count, low
}

func meanFromSum(sum float64, count int) float64 {
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func percentFromCount(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return 100 * float64(part) / float64(total)
}

func meanFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func shareBelow(values []float64, threshold float64) float64 {
	if len(values) == 0 {
		return 0
	}
	count := 0
	for _, value := range values {
		if value < threshold {
			count++
		}
	}
	return float64(count) / float64(len(values))
}

func medianFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return 0.5 * (sorted[mid-1] + sorted[mid])
}

func nearestNeighborCosts(result *climgen.SettlementNetworkResult) []float64 {
	if result == nil || len(result.Nodes) == 0 {
		return nil
	}
	nearest := make([]float64, 0, len(result.Nodes))
	for i := range result.Nodes {
		best := math.Inf(1)
		for _, link := range result.Links {
			switch {
			case link.From == i && link.TravelCost < best:
				best = link.TravelCost
			case link.To == i && link.TravelCost < best:
				best = link.TravelCost
			}
		}
		if !math.IsInf(best, 1) {
			nearest = append(nearest, best)
		}
	}
	return nearest
}

func topSettlementRegions(result *climgen.SettlementNetworkResult, limit int) []climgen.SettlementRegion {
	if result == nil || len(result.Regions) == 0 {
		return nil
	}
	sorted := append([]climgen.SettlementRegion(nil), result.Regions...)
	sort.Slice(sorted, func(i, j int) bool {
		if len(sorted[i].NodeIndices) != len(sorted[j].NodeIndices) {
			return len(sorted[i].NodeIndices) > len(sorted[j].NodeIndices)
		}
		return sorted[i].MeanScore > sorted[j].MeanScore
	})
	if len(sorted) < limit {
		limit = len(sorted)
	}
	return sorted[:limit]
}

func sortedCountKeys(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return keys[i] < keys[j]
	})
	return keys
}
