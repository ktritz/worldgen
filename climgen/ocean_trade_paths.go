package climgen

import (
	"container/heap"
	"math"
)

func buildOceanTradeEndpointGraph(
	sites []Vector3D,
	cells []VoronoiCell,
	climate *SeasonalClimateResult,
	network *SettlementNetworkResult,
	ports *CoastalPortResult,
	candidatePorts []int,
	stopovers []MaritimeStopoverNode,
	elevation []float64,
	seaLevel float64,
	settings OceanTradeSettings,
	civByNode []int,
) ([]coastalTradeEndpoint, [][]coastalEndpointEdge, int) {
	endpoints := make([]coastalTradeEndpoint, 0, len(ports.MajorDeepwaterPorts)+len(stopovers)+8)
	for _, nodeIdx := range candidatePorts {
		cellIdx := network.Nodes[nodeIdx].CellIndex
		if ports.Diagnostics != nil && nodeIdx >= 0 && nodeIdx < len(ports.Diagnostics.NodeDeepwaterTermCell) && ports.Diagnostics.NodeDeepwaterTermCell[nodeIdx] >= 0 {
			cellIdx = ports.Diagnostics.NodeDeepwaterTermCell[nodeIdx]
		}
		endpoints = append(endpoints, coastalTradeEndpoint{
			Node:   nodeIdx,
			Cell:   cellIdx,
			Civ:    civIDForNode(civByNode, nodeIdx),
			Score:  ports.Diagnostics.NodeDeepwaterScore[nodeIdx],
			IsPort: true,
		})
	}
	for _, stopover := range stopovers {
		endpoints = append(endpoints, coastalTradeEndpoint{
			Node:     -1,
			Stopover: stopover.ID,
			Cell:     stopover.CellIndex,
			Civ:      -1,
			Score:    stopover.Score,
			IsPort:   false,
		})
	}
	adj := make([][]coastalEndpointEdge, len(endpoints))
	flatAdj := BuildFlatAdjacency(cells)
	legBudget := oceanLegBudget(ports.Mode, settings)
	meanNeighborDeg := maritimeMeanNeighborDegrees(sites, cells)
	distancePrunedPairs := 0
	workspace := newMaritimeCellPathWorkspace(len(cells))
	for i := 0; i < len(endpoints); i++ {
		for j := i + 1; j < len(endpoints); j++ {
			if !maritimeEndpointPairWithinBudgetLowerBound(sites, cells, endpoints[i].Cell, endpoints[j].Cell, legBudget, 0.16, meanNeighborDeg) {
				distancePrunedPairs++
				continue
			}
			path := shortestOceanCellPathWithWorkspace(sites, cells, flatAdj, climate, elevation, seaLevel, endpoints[i].Cell, endpoints[j].Cell, ports.Mode, legBudget, workspace)
			if !path.ok {
				continue
			}
			adj[i] = append(adj[i], coastalEndpointEdge{neighbor: j, path: path})
			adj[j] = append(adj[j], coastalEndpointEdge{neighbor: i, path: path})
		}
	}
	return endpoints, adj, distancePrunedPairs
}

func shortestOceanCellPath(
	sites []Vector3D,
	cells []VoronoiCell,
	adj *FlatAdjacency,
	climate *SeasonalClimateResult,
	elevation []float64,
	seaLevel float64,
	start, goal int,
	mode MaritimeVesselSettings,
	maxCost float64,
) coastalCellPath {
	return shortestOceanCellPathWithWorkspace(sites, cells, adj, climate, elevation, seaLevel, start, goal, mode, maxCost, newMaritimeCellPathWorkspace(len(cells)))
}

func shortestOceanCellPathWithWorkspace(
	sites []Vector3D,
	cells []VoronoiCell,
	adj *FlatAdjacency,
	climate *SeasonalClimateResult,
	elevation []float64,
	seaLevel float64,
	start, goal int,
	mode MaritimeVesselSettings,
	maxCost float64,
	workspace *maritimeCellPathWorkspace,
) coastalCellPath {
	if start < 0 || goal < 0 || start >= len(cells) || goal >= len(cells) || maxCost <= 0 {
		return coastalCellPath{}
	}
	if workspace == nil {
		workspace = newMaritimeCellPathWorkspace(len(cells))
	}
	workspace.begin(len(cells))
	workspace.setDist(start, 0, -1)
	stepScale := meshPathCostResolutionScale(len(cells))
	pq := &coastalCellHeap{{cell: start, cost: 0}}
	heap.Init(pq)
	for pq.Len() > 0 {
		cur := heap.Pop(pq).(coastalCellState)
		if cur.cost > workspace.getDist(cur.cell) || cur.cost > maxCost {
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
			stepCost := oceanCellStepCost(sites, climate, elevation, seaLevel, adj, cur.cell, neighbor, start, goal, mode)
			if math.IsInf(stepCost, 1) {
				continue
			}
			nextCost := cur.cost + stepCost*stepScale
			if nextCost < workspace.getDist(neighbor) && nextCost <= maxCost {
				workspace.setDist(neighbor, nextCost, cur.cell)
				heap.Push(pq, coastalCellState{cell: neighbor, cost: nextCost})
			}
		}
	}
	goalCost := workspace.getDist(goal)
	if math.IsInf(goalCost, 1) {
		return coastalCellPath{}
	}
	path := make([]int, 0)
	for cur := goal; cur >= 0; cur = workspace.prevCell(cur) {
		path = append(path, cur)
		if cur == start {
			break
		}
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	meanExposure, meanAssist := summarizeCoastalPath(sites, climate, elevation, seaLevel, adj, path)
	return coastalCellPath{ok: true, cost: goalCost, cells: path, meanExposure: meanExposure, meanAssist: meanAssist}
}

func oceanCellStepCost(
	sites []Vector3D,
	climate *SeasonalClimateResult,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	from, to, start, goal int,
	mode MaritimeVesselSettings,
) float64 {
	if !oceanTradeTraversableCell(from, elevation, seaLevel, start, goal) || !oceanTradeTraversableCell(to, elevation, seaLevel, start, goal) {
		return math.Inf(1)
	}
	exposure := 0.5 * (coastalOceanExposure(from, elevation, seaLevel, adj) + coastalOceanExposure(to, elevation, seaLevel, adj))
	currentAssist := directedCurrentAssist(sites, climate, from, to)
	windAssist := directedWindAssist(sites, climate, from, to)
	cost := 1.0 +
		0.62*exposure*(1-mode.StormTolerance) +
		0.45*exposure*(1-mode.OpenOceanCapability) +
		0.22*(1-mode.SeasonalityTolerance)
	cost -= 0.28 * mode.CurrentAssist * currentAssist
	cost += 0.28 * mode.AdverseCurrentPenalty * math.Max(0, -currentAssist)
	cost -= 0.26 * mode.WindAssist * windAssist
	cost += 0.30 * mode.UpwindPenalty * math.Max(0, -windAssist)
	if elevation[from] >= seaLevel || elevation[to] >= seaLevel {
		cost += 0.35 * mode.HarborDependence
	}
	return math.Max(0.16, cost)
}

func oceanTradeTraversableCell(cellIdx int, elevation []float64, seaLevel float64, start, goal int) bool {
	if cellIdx == start || cellIdx == goal {
		return true
	}
	return cellIdx >= 0 && cellIdx < len(elevation) && elevation[cellIdx] < seaLevel
}
