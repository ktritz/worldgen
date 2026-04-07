package climgen

import (
	"container/heap"
	"math"
	"sort"
)

type LocalTradeNodeKind int

const (
	LocalTradeNodeCollectionPoint LocalTradeNodeKind = iota
	LocalTradeNodeWaystation
	LocalTradeNodeCrossingDepot
)

func LocalTradeNodeKindName(kind LocalTradeNodeKind) string {
	names := []string{"Collection Point", "Waystation", "Crossing Depot"}
	if int(kind) < len(names) {
		return names[kind]
	}
	return "Unknown"
}

type LocalTradeNode struct {
	ID          int
	CellIndex   int
	HandoffNode int
	Kind        LocalTradeNodeKind
	Score       float64
	Support     float64
	Waystation  float64
}

type LocalTradeGraphResult struct {
	Nodes []LocalTradeNode
}

func BuildLocalTradeGraph(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	landRoutes *LandRouteResult,
) *LocalTradeGraphResult {
	out := &LocalTradeGraphResult{}
	if network == nil || landRoutes == nil || landRoutes.Diagnostics == nil || len(cells) == 0 {
		return out
	}
	diag := landRoutes.Diagnostics
	anchorCells := make(map[int]struct{}, len(network.Nodes))
	for _, node := range network.Nodes {
		anchorCells[node.CellIndex] = struct{}{}
	}

	type candidate struct {
		node LocalTradeNode
	}
	byCell := map[int]candidate{}
	for _, anchor := range network.Nodes {
		if anchor.Kind < SettlementNodeVillage || anchor.CellIndex < 0 || anchor.CellIndex >= len(cells) {
			continue
		}
		hopRadius := localFeederHopRadius(anchor.Kind, landRoutes.Mode)
		limit := localFeederNodeLimit(anchor.Kind, landRoutes.Mode)
		if limit <= 0 {
			continue
		}
		ring := cellsWithinHops(cells, anchor.CellIndex, hopRadius)
		scored := make([]LocalTradeNode, 0, len(ring))
		for _, cellIdx := range ring {
			if cellIdx == anchor.CellIndex {
				continue
			}
			if _, anchored := anchorCells[cellIdx]; anchored {
				continue
			}
			if cellIdx < 0 || cellIdx >= len(diag.ModeCost) || math.IsInf(diag.ModeCost[cellIdx], 1) {
				continue
			}
			support := clamp01(0.55*diag.WaterSupport[cellIdx] + 0.45*diag.ForageSupport[cellIdx])
			waystation := diag.WaystationSuitability[cellIdx]
			crossing := diag.CrossingPressure[cellIdx]
			if waystation < 0.24 && support < 0.22 && crossing < 0.18 {
				continue
			}
			score := clamp01(
				0.34*waystation +
					0.26*support +
					0.16*(1-diag.RouteRisk[cellIdx]) +
					0.14*diag.RoadQuality[cellIdx] +
					0.10*crossing,
			)
			if score < 0.26 {
				continue
			}
			scored = append(scored, LocalTradeNode{
				CellIndex:   cellIdx,
				HandoffNode: anchor.ID,
				Kind:        classifyLocalTradeNodeKind(diag, cellIdx, support, waystation),
				Score:       score,
				Support:     support,
				Waystation:  waystation,
			})
		}
		sort.Slice(scored, func(i, j int) bool {
			if scored[i].Score != scored[j].Score {
				return scored[i].Score > scored[j].Score
			}
			if scored[i].Kind != scored[j].Kind {
				return scored[i].Kind > scored[j].Kind
			}
			return scored[i].CellIndex < scored[j].CellIndex
		})
		if len(scored) > limit {
			scored = scored[:limit]
		}
		for _, node := range scored {
			current, exists := byCell[node.CellIndex]
			if !exists || node.Score > current.node.Score ||
				(node.Score == current.node.Score && network.Nodes[node.HandoffNode].Kind > network.Nodes[current.node.HandoffNode].Kind) {
				byCell[node.CellIndex] = candidate{node: node}
			}
		}
	}

	out.Nodes = make([]LocalTradeNode, 0, len(byCell))
	for _, entry := range byCell {
		out.Nodes = append(out.Nodes, entry.node)
	}
	sort.Slice(out.Nodes, func(i, j int) bool {
		if out.Nodes[i].Score != out.Nodes[j].Score {
			return out.Nodes[i].Score > out.Nodes[j].Score
		}
		return out.Nodes[i].CellIndex < out.Nodes[j].CellIndex
	})
	for i := range out.Nodes {
		out.Nodes[i].ID = i
	}
	return out
}

func collectLocalFeederCorridors(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	local *LocalTradeGraphResult,
	landRoutes *LandRouteResult,
	settings TradeNetworkSettings,
) []TradeCorridor {
	if network == nil || local == nil || len(local.Nodes) == 0 || landRoutes == nil || landRoutes.Diagnostics == nil || landRoutes.Mode.FeederFlow <= 0 {
		return nil
	}
	maxCost := settings.MaxRouteCost * maxFloat(0.32, landRoutes.Mode.FeederReach)
	out := make([]TradeCorridor, 0, len(local.Nodes))
	for _, node := range local.Nodes {
		if node.HandoffNode < 0 || node.HandoffNode >= len(network.Nodes) {
			continue
		}
		path := shortestFeederCellPath(cells, node.CellIndex, network.Nodes[node.HandoffNode].CellIndex, landRoutes, maxCost)
		if !path.ok || len(path.cells) < 2 {
			continue
		}
		flow := localGraphFeederFlow(node, network.Nodes[node.HandoffNode], path.cost, landRoutes.Mode)
		if flow < settings.MinFlow*0.28 {
			continue
		}
		out = append(out, buildLocalTradeCorridor(node, network.Nodes[node.HandoffNode], path, flow, landRoutes))
	}
	return out
}

type feederCellPath struct {
	ok    bool
	cost  float64
	cells []int
}

type feederCellState struct {
	cell int
	cost float64
}

type feederCellHeap []feederCellState

func (h feederCellHeap) Len() int            { return len(h) }
func (h feederCellHeap) Less(i, j int) bool  { return h[i].cost < h[j].cost }
func (h feederCellHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *feederCellHeap) Push(x interface{}) { *h = append(*h, x.(feederCellState)) }
func (h *feederCellHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func shortestFeederCellPath(
	cells []VoronoiCell,
	start, goal int,
	landRoutes *LandRouteResult,
	maxCost float64,
) feederCellPath {
	if start < 0 || goal < 0 || start >= len(cells) || goal >= len(cells) || landRoutes == nil || landRoutes.Diagnostics == nil {
		return feederCellPath{}
	}
	diag := landRoutes.Diagnostics
	dist := make([]float64, len(cells))
	prev := make([]int, len(cells))
	for i := range dist {
		dist[i] = math.Inf(1)
		prev[i] = -1
	}
	dist[start] = 0
	stepScale := meshPathCostResolutionScale(len(cells))
	pq := &feederCellHeap{{cell: start, cost: 0}}
	heap.Init(pq)
	for pq.Len() > 0 {
		cur := heap.Pop(pq).(feederCellState)
		if cur.cost > dist[cur.cell] || cur.cost > maxCost {
			continue
		}
		if cur.cell == goal {
			break
		}
		for _, rawNeighbor := range cells[cur.cell].NeighborSiteIndices {
			neighbor := int(rawNeighbor)
			if neighbor < 0 || neighbor >= len(cells) {
				continue
			}
			stepCost := feederStepTravelCost(cur.cell, neighbor, diag)
			if math.IsInf(stepCost, 1) {
				continue
			}
			nextCost := cur.cost + stepCost*stepScale
			if nextCost < dist[neighbor] && nextCost <= maxCost {
				dist[neighbor] = nextCost
				prev[neighbor] = cur.cell
				heap.Push(pq, feederCellState{cell: neighbor, cost: nextCost})
			}
		}
	}
	if math.IsInf(dist[goal], 1) {
		return feederCellPath{}
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
	return feederCellPath{ok: true, cost: dist[goal], cells: path}
}

func feederStepTravelCost(from, to int, diag *LandRouteDiagnostics) float64 {
	if diag == nil || from < 0 || to < 0 || from >= len(diag.ModeCost) || to >= len(diag.ModeCost) {
		return math.Inf(1)
	}
	if math.IsInf(diag.ModeCost[from], 1) || math.IsInf(diag.ModeCost[to], 1) {
		return math.Inf(1)
	}
	meanCost := 0.5 * (diag.ModeCost[from] + diag.ModeCost[to])
	meanRisk := 0.5 * (diag.RouteRisk[from] + diag.RouteRisk[to])
	meanSupport := 0.5 * (diag.WaystationSuitability[from] + diag.WaystationSuitability[to])
	return meanCost * (1 + 0.24*meanRisk + 0.10*(1-meanSupport))
}

func buildLocalTradeCorridor(
	node LocalTradeNode,
	handoff SettlementNode,
	path feederCellPath,
	flow float64,
	landRoutes *LandRouteResult,
) TradeCorridor {
	meanRisk, meanSupport := summarizeTradeCellPath(path.cells, landRoutes)
	modeName := "generic-land"
	if landRoutes != nil {
		modeName = landRoutes.Mode.Name
	}
	return TradeCorridor{
		FromLocalNode:     node.ID,
		ToLocalNode:       -1,
		FromNode:          -1,
		ToNode:            handoff.ID,
		FromCivilization:  -1,
		ToCivilization:    -1,
		Mode:              modeName,
		Role:              TradeCorridorRoleFeeder,
		HandoffNode:       handoff.ID,
		TravelCost:        path.cost,
		Flow:              flow,
		MeanRisk:          meanRisk,
		MeanSupport:       meanSupport,
		NodePath:          []int{handoff.ID},
		CellPath:          append([]int(nil), path.cells...),
		InterCivilization: false,
	}
}

func localGraphFeederFlow(from LocalTradeNode, to SettlementNode, cost float64, mode LandRouteModeSettings) float64 {
	base := 0.16 + 0.30*from.Score + 0.18*from.Support + 0.16*from.Waystation + 0.16*to.Score + 0.04*float64(to.Kind)
	return base * maxFloat(mode.FeederFlow, 0) / math.Max(cost*1.05, 1.0)
}

func classifyLocalTradeNodeKind(diag *LandRouteDiagnostics, cellIdx int, support, waystation float64) LocalTradeNodeKind {
	if diag == nil || cellIdx < 0 || cellIdx >= len(diag.CrossingPressure) {
		return LocalTradeNodeCollectionPoint
	}
	crossing := diag.CrossingPressure[cellIdx]
	bridge := diag.BridgeProxy[cellIdx]
	ford := diag.FordProxy[cellIdx]
	road := diag.RoadQuality[cellIdx]
	if crossing >= 0.30 &&
		(bridge >= 0.08 || ford >= 0.10) &&
		(waystation >= 0.24 || support >= 0.28 || road >= 0.24) {
		return LocalTradeNodeCrossingDepot
	}
	if waystation >= 0.28 || (support >= 0.30 && road >= 0.20) {
		return LocalTradeNodeWaystation
	}
	return LocalTradeNodeCollectionPoint
}

func localFeederHopRadius(kind SettlementNodeKind, mode LandRouteModeSettings) int {
	base := 1 + int(math.Round(2.0*maxFloat(mode.FeederReach, 0)))
	switch {
	case kind >= SettlementNodeTown:
		base++
	case kind == SettlementNodeVillage && base < 2:
		base = 2
	}
	if base < 1 {
		base = 1
	}
	if base > 4 {
		base = 4
	}
	return base
}

func localFeederNodeLimit(kind SettlementNodeKind, mode LandRouteModeSettings) int {
	limit := 1 + int(math.Round(mode.FeederFlow+0.6*mode.PayloadCapacity))
	if kind >= SettlementNodeTown {
		limit++
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 3 {
		limit = 3
	}
	return limit
}

func cellsWithinHops(cells []VoronoiCell, start, maxHops int) []int {
	if start < 0 || start >= len(cells) || maxHops < 1 {
		return nil
	}
	type state struct {
		cell int
		hops int
	}
	seen := map[int]struct{}{start: {}}
	queue := []state{{cell: start, hops: 0}}
	out := make([]int, 0)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.hops > 0 {
			out = append(out, cur.cell)
		}
		if cur.hops == maxHops {
			continue
		}
		for _, rawNeighbor := range cells[cur.cell].NeighborSiteIndices {
			neighbor := int(rawNeighbor)
			if neighbor < 0 || neighbor >= len(cells) {
				continue
			}
			if _, ok := seen[neighbor]; ok {
				continue
			}
			seen[neighbor] = struct{}{}
			queue = append(queue, state{cell: neighbor, hops: cur.hops + 1})
		}
	}
	return out
}
