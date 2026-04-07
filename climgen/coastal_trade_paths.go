package climgen

import (
	"container/heap"
	"math"
)

type coastalCellPath struct {
	ok           bool
	cost         float64
	cells        []int
	meanExposure float64
	meanAssist   float64
}

type coastalCellState struct {
	cell int
	cost float64
}

type coastalCellHeap []coastalCellState

func (h coastalCellHeap) Len() int            { return len(h) }
func (h coastalCellHeap) Less(i, j int) bool  { return h[i].cost < h[j].cost }
func (h coastalCellHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *coastalCellHeap) Push(x interface{}) { *h = append(*h, x.(coastalCellState)) }
func (h *coastalCellHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func shortestCoastalCellPath(
	sites []Vector3D,
	cells []VoronoiCell,
	climate *SeasonalClimateResult,
	elevation []float64,
	seaLevel float64,
	start, goal int,
	mode MaritimeVesselSettings,
	maxCost float64,
) coastalCellPath {
	return shortestMaritimeCellPath(sites, cells, climate, elevation, seaLevel, start, goal, mode, maxCost, false)
}

func shortestMaritimeCellPath(
	sites []Vector3D,
	cells []VoronoiCell,
	climate *SeasonalClimateResult,
	elevation []float64,
	seaLevel float64,
	start, goal int,
	mode MaritimeVesselSettings,
	maxCost float64,
	allowOpenWater bool,
) coastalCellPath {
	if start < 0 || goal < 0 || start >= len(cells) || goal >= len(cells) {
		return coastalCellPath{}
	}
	adj := BuildFlatAdjacency(cells)
	var coastDist []int
	if allowOpenWater {
		coastDist = coastalWaterDistanceFromLand(adj, elevation, seaLevel, openWaterHopAllowance(mode))
	}
	dist := make([]float64, len(cells))
	prev := make([]int, len(cells))
	for i := range dist {
		dist[i] = math.Inf(1)
		prev[i] = -1
	}
	dist[start] = 0
	stepScale := meshPathCostResolutionScale(len(cells))
	pq := &coastalCellHeap{{cell: start, cost: 0}}
	heap.Init(pq)
	for pq.Len() > 0 {
		cur := heap.Pop(pq).(coastalCellState)
		if cur.cost > dist[cur.cell] || cur.cost > maxCost {
			continue
		}
		if cur.cell == goal {
			break
		}
		for _, raw := range cells[cur.cell].NeighborSiteIndices {
			neighbor := int(raw)
			if neighbor < 0 || neighbor >= len(cells) {
				continue
			}
			stepCost := coastalCellStepCost(sites, climate, elevation, seaLevel, adj, coastDist, cur.cell, neighbor, mode, allowOpenWater)
			if math.IsInf(stepCost, 1) {
				continue
			}
			nextCost := cur.cost + stepCost*stepScale
			if nextCost < dist[neighbor] && nextCost <= maxCost {
				dist[neighbor] = nextCost
				prev[neighbor] = cur.cell
				heap.Push(pq, coastalCellState{cell: neighbor, cost: nextCost})
			}
		}
	}
	if math.IsInf(dist[goal], 1) {
		return coastalCellPath{}
	}
	path := make([]int, 0)
	for cur := goal; cur >= 0; cur = prev[cur] {
		path = append(path, cur)
		if cur == start {
			break
		}
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	meanExposure, meanAssist := summarizeCoastalPath(sites, climate, elevation, seaLevel, adj, path)
	return coastalCellPath{
		ok:           true,
		cost:         dist[goal],
		cells:        path,
		meanExposure: meanExposure,
		meanAssist:   meanAssist,
	}
}

func coastalCellStepCost(
	sites []Vector3D,
	climate *SeasonalClimateResult,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	coastDist []int,
	from, to int,
	mode MaritimeVesselSettings,
	allowOpenWater bool,
) float64 {
	if from < 0 || to < 0 || from >= len(elevation) || to >= len(elevation) {
		return math.Inf(1)
	}
	if !coastalTradeTraversableCell(from, elevation, seaLevel, adj, coastDist, mode, allowOpenWater) || !coastalTradeTraversableCell(to, elevation, seaLevel, adj, coastDist, mode, allowOpenWater) {
		return math.Inf(1)
	}
	exposure := 0.5 * (coastalOceanExposure(from, elevation, seaLevel, adj) + coastalOceanExposure(to, elevation, seaLevel, adj))
	currentAssist := directedCurrentAssist(sites, climate, from, to)
	windAssist := directedWindAssist(sites, climate, from, to)
	cost := 1.0 +
		0.55*exposure*(1-mode.StormTolerance) +
		0.22*exposure*(1-mode.CoastalCapability) +
		0.18*(1-mode.SeasonalityTolerance)
	cost -= 0.22 * mode.CurrentAssist * currentAssist
	cost += 0.24 * mode.AdverseCurrentPenalty * math.Max(0, -currentAssist)
	cost -= 0.18 * mode.WindAssist * windAssist
	cost += 0.22 * mode.UpwindPenalty * math.Max(0, -windAssist)
	if allowOpenWater {
		openPenalty := 0.0
		if elevation[from] < seaLevel && !isCoastalOcean(from, elevation, seaLevel, adj) {
			openPenalty += 0.5
		}
		if elevation[to] < seaLevel && !isCoastalOcean(to, elevation, seaLevel, adj) {
			openPenalty += 0.5
		}
		if openPenalty > 0 {
			cost += openPenalty * (0.55 + 0.55*(1-mode.OpenOceanCapability) + 0.25*mode.StopoverNeed + 0.20*(1-mode.StormTolerance))
		}
	}
	return math.Max(0.18, cost)
}

func coastalTradeTraversableCell(i int, elevation []float64, seaLevel float64, adj *FlatAdjacency, coastDist []int, mode MaritimeVesselSettings, allowOpenWater bool) bool {
	if elevation[i] < seaLevel {
		if isCoastalOcean(i, elevation, seaLevel, adj) {
			return true
		}
		if allowOpenWater && mode.OpenOceanCapability > 0 && mode.MaxOpenWaterLeg > 0 && i < len(coastDist) && coastDist[i] >= 0 && coastDist[i] <= openWaterHopAllowance(mode) {
			return true
		}
		return false
	}
	return isCoastalLand(i, elevation, seaLevel, adj)
}

func openWaterHopAllowance(mode MaritimeVesselSettings) int {
	if mode.OpenOceanCapability <= 0 || mode.MaxOpenWaterLeg <= 0 {
		return 0
	}
	hops := 1 + int(math.Round(8*mode.MaxOpenWaterLeg*(0.35+0.65*mode.OpenOceanCapability)))
	if hops < 1 {
		return 1
	}
	if hops > 6 {
		return 6
	}
	return hops
}

func coastalWaterDistanceFromLand(adj *FlatAdjacency, elevation []float64, seaLevel float64, maxHops int) []int {
	dist := make([]int, len(elevation))
	for i := range dist {
		dist[i] = -1
	}
	if maxHops < 0 {
		return dist
	}
	type state struct {
		cell int
		hops int
	}
	queue := make([]state, 0)
	for i := range elevation {
		if elevation[i] < seaLevel && isCoastalOcean(i, elevation, seaLevel, adj) {
			dist[i] = 0
			queue = append(queue, state{cell: i, hops: 0})
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.hops >= maxHops {
			continue
		}
		for _, raw := range adj.GetNeighbors(cur.cell) {
			neighbor := raw
			if neighbor < 0 || neighbor >= len(elevation) || elevation[neighbor] >= seaLevel || dist[neighbor] >= 0 {
				continue
			}
			dist[neighbor] = cur.hops + 1
			queue = append(queue, state{cell: neighbor, hops: cur.hops + 1})
		}
	}
	return dist
}

func coastalOceanExposure(i int, elevation []float64, seaLevel float64, adj *FlatAdjacency) float64 {
	if i < 0 || i >= len(elevation) || elevation[i] >= seaLevel {
		return 0
	}
	neighbors := adj.GetNeighbors(i)
	if len(neighbors) == 0 {
		return 1
	}
	ocean := 0.0
	total := 0.0
	for _, k := range neighbors {
		if k < 0 || k >= len(elevation) {
			continue
		}
		total++
		if elevation[k] < seaLevel {
			ocean++
		}
	}
	if total == 0 {
		return 1
	}
	return clamp01(ocean / total)
}

func directedCurrentAssist(sites []Vector3D, climate *SeasonalClimateResult, from, to int) float64 {
	if climate == nil || from < 0 || to < 0 || from >= len(climate.Currents) || to >= len(sites) || from >= len(sites) {
		return 0
	}
	return directedAssist(sites[from], sites[to], climate.Currents[from])
}

func directedWindAssist(sites []Vector3D, climate *SeasonalClimateResult, from, to int) float64 {
	if climate == nil || len(climate.Snapshots) == 0 || from < 0 || to < 0 || from >= len(sites) || to >= len(sites) {
		return 0
	}
	sum := 0.0
	count := 0.0
	for _, snap := range climate.Snapshots {
		if from >= len(snap.SurfaceWind) {
			continue
		}
		sum += directedAssist(sites[from], sites[to], snap.SurfaceWind[from])
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / count
}

func directedAssist(from, to Vector3D, flow Vector3D) float64 {
	delta := Normalize(Sub(to, from))
	speed := Length(flow)
	if speed < 1e-9 || LengthSq(delta) < 1e-12 {
		return 0
	}
	return Dot(delta, Normalize(flow))
}

func summarizeCoastalPath(sites []Vector3D, climate *SeasonalClimateResult, elevation []float64, seaLevel float64, adj *FlatAdjacency, path []int) (float64, float64) {
	if len(path) == 0 {
		return 0, 0
	}
	sumExposure := 0.0
	sumAssist := 0.0
	count := 0.0
	for i, cell := range path {
		sumExposure += coastalOceanExposure(cell, elevation, seaLevel, adj)
		if i+1 < len(path) {
			sumAssist += 0.5 * (directedCurrentAssist(sites, climate, cell, path[i+1]) + directedWindAssist(sites, climate, cell, path[i+1]))
		}
		count++
	}
	if count == 0 {
		return 0, 0
	}
	return sumExposure / count, sumAssist / count
}

func buildCoastalTradeCorridor(fromNode, toNode, fromCiv, toCiv int, flow float64, path coastalEndpointPath) CoastalTradeCorridor {
	corridor := CoastalTradeCorridor{
		FromNode:          fromNode,
		ToNode:            toNode,
		FromCivilization:  fromCiv,
		ToCivilization:    toCiv,
		TravelCost:        path.cost,
		Flow:              flow,
		MeanExposure:      path.meanExposure,
		MeanCurrentAssist: path.meanAssist,
		CellPath:          append([]int(nil), path.cells...),
		InterCivilization: fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv,
	}
	if len(path.stopovers) > 0 {
		corridor.FromStopover = path.stopovers[0]
		corridor.ToStopover = path.stopovers[len(path.stopovers)-1]
	} else {
		corridor.FromStopover = -1
		corridor.ToStopover = -1
	}
	return corridor
}
