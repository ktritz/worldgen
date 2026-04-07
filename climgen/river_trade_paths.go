package climgen

import (
	"container/heap"
	"math"
)

type riverCellPath struct {
	ok    bool
	cost  float64
	cells []int
}

type riverCellState struct {
	cell int
	cost float64
}

type riverCellHeap []riverCellState

func (h riverCellHeap) Len() int            { return len(h) }
func (h riverCellHeap) Less(i, j int) bool  { return h[i].cost < h[j].cost }
func (h riverCellHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *riverCellHeap) Push(x interface{}) { *h = append(*h, x.(riverCellState)) }
func (h *riverCellHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

type riverAdjEdge struct {
	neighbor  int
	linkIndex int
	cost      float64
}

func buildRiverTradeAdjacency(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	riverRoutes *RiverRouteResult,
	transitNodes map[int]struct{},
	elevation []float64,
) [][]riverAdjEdge {
	adj := make([][]riverAdjEdge, len(network.Nodes))
	for i, link := range network.Links {
		if _, ok := transitNodes[link.From]; !ok {
			continue
		}
		if _, ok := transitNodes[link.To]; !ok {
			continue
		}
		forwardCost, reverseCost := riverLinkDirectionalTravelCost(network, link, riverRoutes, elevation)
		if !math.IsInf(forwardCost, 1) {
			adj[link.From] = append(adj[link.From], riverAdjEdge{neighbor: link.To, linkIndex: i, cost: forwardCost})
		}
		if !math.IsInf(reverseCost, 1) {
			adj[link.To] = append(adj[link.To], riverAdjEdge{neighbor: link.From, linkIndex: i, cost: reverseCost})
		}
	}
	return adj
}

func riverLinkDirectionalTravelCost(
	network *SettlementNetworkResult,
	link SettlementLink,
	riverRoutes *RiverRouteResult,
	elevation []float64,
) (float64, float64) {
	if riverRoutes == nil || riverRoutes.Diagnostics == nil {
		return math.Inf(1), math.Inf(1)
	}
	diag := riverRoutes.Diagnostics
	if len(link.Path) == 0 {
		return math.Inf(1), math.Inf(1)
	}
	totalDown := 0.0
	totalUp := 0.0
	usable := 0.0
	for _, cellIdx := range link.Path {
		if cellIdx < 0 || cellIdx >= len(diag.Navigability) {
			continue
		}
		nav := diag.Navigability[cellIdx]
		if nav >= riverRoutes.Mode.MinNavigability*0.50 {
			downCost := diag.DownstreamTravelCost[cellIdx]
			upCost := diag.UpstreamTravelCost[cellIdx]
			if math.IsInf(downCost, 1) || math.IsInf(upCost, 1) {
				continue
			}
			totalDown += downCost
			totalUp += upCost
			usable++
			continue
		}
		portage := diag.PortageSuitability[cellIdx]
		if portage < 0.18*math.Max(riverRoutes.Mode.PortageTolerance, 0.2) {
			continue
		}
		transfer := diag.TransferSupport[cellIdx]
		portageBase := (1.6 + 1.4*(1-riverRoutes.Mode.PortageTolerance)) * (1 + 0.30*(1-transfer))
		totalDown += portageBase
		totalUp += portageBase
		usable++
	}
	if usable == 0 {
		return math.Inf(1), math.Inf(1)
	}
	coverage := usable / float64(len(link.Path))
	if coverage < 0.25 {
		return math.Inf(1), math.Inf(1)
	}
	coveragePenalty := 1 + 0.55*(1-coverage)
	forwardDownstream := riverLinkForwardIsDownstream(network, link, elevation)
	if forwardDownstream {
		return totalDown * coveragePenalty, totalUp * coveragePenalty
	}
	return totalUp * coveragePenalty, totalDown * coveragePenalty
}

func riverLinkForwardIsDownstream(network *SettlementNetworkResult, link SettlementLink, elevation []float64) bool {
	if network == nil || link.From < 0 || link.To < 0 || link.From >= len(network.Nodes) || link.To >= len(network.Nodes) {
		return true
	}
	fromCell := network.Nodes[link.From].CellIndex
	toCell := network.Nodes[link.To].CellIndex
	if fromCell < 0 || toCell < 0 || fromCell >= len(elevation) || toCell >= len(elevation) {
		return true
	}
	if math.Abs(elevation[fromCell]-elevation[toCell]) < 0.005 {
		return true
	}
	return elevation[fromCell] > elevation[toCell]
}

func shortestRiverCellPath(
	cells []VoronoiCell,
	start, goal int,
	riverRoutes *RiverRouteResult,
	maxCost float64,
) riverCellPath {
	if start < 0 || goal < 0 || start >= len(cells) || goal >= len(cells) || riverRoutes == nil || riverRoutes.Diagnostics == nil {
		return riverCellPath{}
	}
	dist := make([]float64, len(cells))
	prev := make([]int, len(cells))
	for i := range dist {
		dist[i] = math.Inf(1)
		prev[i] = -1
	}
	dist[start] = 0
	stepScale := riverCellCostResolutionScale(len(cells))
	pq := &riverCellHeap{{cell: start, cost: 0}}
	heap.Init(pq)
	for pq.Len() > 0 {
		cur := heap.Pop(pq).(riverCellState)
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
			stepCost := riverCellStepCost(cur.cell, neighbor, riverRoutes)
			if math.IsInf(stepCost, 1) {
				continue
			}
			nextCost := cur.cost + stepCost*stepScale
			if nextCost < dist[neighbor] && nextCost <= maxCost {
				dist[neighbor] = nextCost
				prev[neighbor] = cur.cell
				heap.Push(pq, riverCellState{cell: neighbor, cost: nextCost})
			}
		}
	}
	if math.IsInf(dist[goal], 1) {
		return riverCellPath{}
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
	return riverCellPath{ok: true, cost: dist[goal], cells: path}
}

func riverCellCostResolutionScale(cellCount int) float64 {
	if cellCount <= 0 {
		return 1
	}
	scale := math.Sqrt(10242.0 / float64(cellCount))
	if scale < 0.25 {
		return 0.25
	}
	if scale > 2.0 {
		return 2.0
	}
	return scale
}

func riverCellStepCost(from, to int, riverRoutes *RiverRouteResult) float64 {
	diag := riverRoutes.Diagnostics
	if from < 0 || to < 0 ||
		from >= len(diag.Navigability) || to >= len(diag.Navigability) ||
		from >= len(diag.TransferSupport) || to >= len(diag.TransferSupport) ||
		from >= len(diag.PortageSuitability) || to >= len(diag.PortageSuitability) {
		return math.Inf(1)
	}
	navFrom := diag.Navigability[from]
	navTo := diag.Navigability[to]
	if navFrom >= riverRoutes.Mode.MinNavigability*0.40 || navTo >= riverRoutes.Mode.MinNavigability*0.40 {
		down := meanFinite(diag.DownstreamTravelCost[from], diag.DownstreamTravelCost[to])
		up := meanFinite(diag.UpstreamTravelCost[from], diag.UpstreamTravelCost[to])
		if math.IsInf(down, 1) || math.IsInf(up, 1) {
			return math.Inf(1)
		}
		if diag.DownstreamTravelCost[to] <= diag.DownstreamTravelCost[from] {
			return down
		}
		return up
	}
	portage := 0.5 * (diag.PortageSuitability[from] + diag.PortageSuitability[to])
	transfer := 0.5 * (diag.TransferSupport[from] + diag.TransferSupport[to])
	if portage < 0.18*math.Max(riverRoutes.Mode.PortageTolerance, 0.2) {
		return math.Inf(1)
	}
	return (1.6 + 1.4*(1-riverRoutes.Mode.PortageTolerance)) * (1 + 0.30*(1-transfer))
}

func buildRiverTradeCorridorFromCells(
	fromNode, toNode int,
	fromCiv, toCiv int,
	flow float64,
	inter bool,
	path riverCellPath,
	riverRoutes *RiverRouteResult,
) RiverTradeCorridor {
	meanNav, meanTransfer := summarizeRiverTradeCellPath(path.cells, riverRoutes)
	return RiverTradeCorridor{
		FromNode:          fromNode,
		ToNode:            toNode,
		FromCivilization:  fromCiv,
		ToCivilization:    toCiv,
		TravelCost:        path.cost,
		Flow:              flow,
		MeanNavigability:  meanNav,
		MeanTransfer:      meanTransfer,
		NodePath:          []int{fromNode, toNode},
		CellPath:          append([]int(nil), path.cells...),
		InterCivilization: inter,
	}
}

func meanFinite(a, b float64) float64 {
	if math.IsInf(a, 1) && math.IsInf(b, 1) {
		return math.Inf(1)
	}
	if math.IsInf(a, 1) {
		return b
	}
	if math.IsInf(b, 1) {
		return a
	}
	return 0.5 * (a + b)
}

type riverNodePath struct {
	ok        bool
	cost      float64
	nodes     []int
	linkOrder []int
}

type riverPQState struct {
	node int
	cost float64
}

type riverPathHeap []riverPQState

func (h riverPathHeap) Len() int            { return len(h) }
func (h riverPathHeap) Less(i, j int) bool  { return h[i].cost < h[j].cost }
func (h riverPathHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *riverPathHeap) Push(x interface{}) { *h = append(*h, x.(riverPQState)) }
func (h *riverPathHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func shortestRiverNodePath(start, goal int, network *SettlementNetworkResult, adj [][]riverAdjEdge, maxCost float64) riverNodePath {
	if start < 0 || goal < 0 || start >= len(network.Nodes) || goal >= len(network.Nodes) {
		return riverNodePath{}
	}
	dist := make([]float64, len(network.Nodes))
	prevNode := make([]int, len(network.Nodes))
	prevLink := make([]int, len(network.Nodes))
	for i := range dist {
		dist[i] = math.Inf(1)
		prevNode[i] = -1
		prevLink[i] = -1
	}
	dist[start] = 0
	pq := &riverPathHeap{{node: start, cost: 0}}
	heap.Init(pq)
	for pq.Len() > 0 {
		cur := heap.Pop(pq).(riverPQState)
		if cur.cost > dist[cur.node] || cur.cost > maxCost {
			continue
		}
		if cur.node == goal {
			break
		}
		for _, edge := range adj[cur.node] {
			nextCost := cur.cost + edge.cost
			if nextCost < dist[edge.neighbor] && nextCost <= maxCost {
				dist[edge.neighbor] = nextCost
				prevNode[edge.neighbor] = cur.node
				prevLink[edge.neighbor] = edge.linkIndex
				heap.Push(pq, riverPQState{node: edge.neighbor, cost: nextCost})
			}
		}
	}
	if math.IsInf(dist[goal], 1) {
		return riverNodePath{}
	}
	nodes := make([]int, 0)
	links := make([]int, 0)
	for cur := goal; cur >= 0; cur = prevNode[cur] {
		nodes = append(nodes, cur)
		if cur == start {
			break
		}
		links = append(links, prevLink[cur])
	}
	for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
	for i, j := 0, len(links)-1; i < j; i, j = i+1, j-1 {
		links[i], links[j] = links[j], links[i]
	}
	return riverNodePath{ok: true, cost: dist[goal], nodes: nodes, linkOrder: links}
}

func buildRiverTradeCorridor(
	network *SettlementNetworkResult,
	path riverNodePath,
	fromCiv, toCiv int,
	flow float64,
	inter bool,
	riverRoutes *RiverRouteResult,
) RiverTradeCorridor {
	cellPath := make([]int, 0)
	for i, linkIdx := range path.linkOrder {
		link := network.Links[linkIdx]
		fromNode := path.nodes[i]
		segment := link.Path
		if link.From != fromNode {
			segment = reversedInts(segment)
		}
		if len(cellPath) > 0 && len(segment) > 0 && cellPath[len(cellPath)-1] == segment[0] {
			cellPath = append(cellPath, segment[1:]...)
		} else {
			cellPath = append(cellPath, segment...)
		}
	}
	meanNav, meanTransfer := summarizeRiverTradeCellPath(cellPath, riverRoutes)
	return RiverTradeCorridor{
		FromNode:          path.nodes[0],
		ToNode:            path.nodes[len(path.nodes)-1],
		FromCivilization:  fromCiv,
		ToCivilization:    toCiv,
		TravelCost:        path.cost,
		Flow:              flow,
		MeanNavigability:  meanNav,
		MeanTransfer:      meanTransfer,
		NodePath:          append([]int(nil), path.nodes...),
		CellPath:          cellPath,
		InterCivilization: inter,
	}
}

func summarizeRiverTradeCellPath(cellPath []int, riverRoutes *RiverRouteResult) (float64, float64) {
	if riverRoutes == nil || riverRoutes.Diagnostics == nil || len(cellPath) == 0 {
		return 0, 0
	}
	totalNav := 0.0
	totalTransfer := 0.0
	count := 0.0
	for _, cellIdx := range cellPath {
		if cellIdx < 0 || cellIdx >= len(riverRoutes.Diagnostics.Navigability) {
			continue
		}
		totalNav += riverRoutes.Diagnostics.Navigability[cellIdx]
		totalTransfer += riverRoutes.Diagnostics.TransferSupport[cellIdx]
		count++
	}
	if count == 0 {
		return 0, 0
	}
	return totalNav / count, totalTransfer / count
}
