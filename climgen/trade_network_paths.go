package climgen

import (
	"container/heap"
	"math"
)

type tradeAdjEdge struct {
	neighbor  int
	linkIndex int
	cost      float64
}

func buildTradeAdjacency(network *SettlementNetworkResult, landRoutes *LandRouteResult) [][]tradeAdjEdge {
	adj := make([][]tradeAdjEdge, len(network.Nodes))
	for i, link := range network.Links {
		if link.From < 0 || link.From >= len(adj) || link.To < 0 || link.To >= len(adj) {
			continue
		}
		cost := tradeLinkTravelCost(link, landRoutes)
		adj[link.From] = append(adj[link.From], tradeAdjEdge{neighbor: link.To, linkIndex: i, cost: cost})
		adj[link.To] = append(adj[link.To], tradeAdjEdge{neighbor: link.From, linkIndex: i, cost: cost})
	}
	return adj
}

type tradeNodePath struct {
	ok        bool
	cost      float64
	nodes     []int
	linkOrder []int
}

type tradePQState struct {
	node int
	cost float64
}

type tradePathHeap []tradePQState

func (h tradePathHeap) Len() int            { return len(h) }
func (h tradePathHeap) Less(i, j int) bool  { return h[i].cost < h[j].cost }
func (h tradePathHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *tradePathHeap) Push(x interface{}) { *h = append(*h, x.(tradePQState)) }
func (h *tradePathHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func shortestTradeNodePath(start, goal int, network *SettlementNetworkResult, adj [][]tradeAdjEdge, maxCost float64) tradeNodePath {
	if start < 0 || goal < 0 || start >= len(network.Nodes) || goal >= len(network.Nodes) {
		return tradeNodePath{}
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
	pq := &tradePathHeap{{node: start, cost: 0}}
	heap.Init(pq)
	for pq.Len() > 0 {
		cur := heap.Pop(pq).(tradePQState)
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
				heap.Push(pq, tradePQState{node: edge.neighbor, cost: nextCost})
			}
		}
	}
	if math.IsInf(dist[goal], 1) {
		return tradeNodePath{}
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
	return tradeNodePath{ok: true, cost: dist[goal], nodes: nodes, linkOrder: links}
}

func buildTradeCorridor(network *SettlementNetworkResult, path tradeNodePath, fromCiv, toCiv int, flow float64, inter bool, role TradeCorridorRole, handoffNode int, landRoutes *LandRouteResult) TradeCorridor {
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
	meanRisk, meanSupport := summarizeTradeCellPath(cellPath, landRoutes)
	modeName := "generic-land"
	if landRoutes != nil {
		modeName = landRoutes.Mode.Name
	}
	return TradeCorridor{
		FromLocalNode:     -1,
		ToLocalNode:       -1,
		FromNode:          path.nodes[0],
		ToNode:            path.nodes[len(path.nodes)-1],
		FromCivilization:  fromCiv,
		ToCivilization:    toCiv,
		Mode:              modeName,
		Role:              role,
		HandoffNode:       handoffNode,
		TravelCost:        path.cost,
		Flow:              flow,
		MeanRisk:          meanRisk,
		MeanSupport:       meanSupport,
		NodePath:          append([]int(nil), path.nodes...),
		CellPath:          cellPath,
		InterCivilization: inter,
	}
}

func tradeLinkTravelCost(link SettlementLink, landRoutes *LandRouteResult) float64 {
	if landRoutes == nil || landRoutes.Diagnostics == nil || len(link.Path) == 0 {
		return link.TravelCost
	}
	diag := landRoutes.Diagnostics
	totalCost := 0.0
	totalRisk := 0.0
	totalSupport := 0.0
	count := 0.0
	for _, cellIdx := range link.Path {
		if cellIdx < 0 || cellIdx >= len(diag.ModeCost) || math.IsInf(diag.ModeCost[cellIdx], 1) {
			continue
		}
		totalCost += diag.ModeCost[cellIdx]
		totalRisk += diag.RouteRisk[cellIdx]
		totalSupport += diag.WaystationSuitability[cellIdx]
		count++
	}
	if count == 0 {
		return link.TravelCost
	}
	meanCost := totalCost / count
	meanRisk := totalRisk / count
	meanSupport := totalSupport / count
	return count * meanCost * (1 + 0.55*meanRisk + 0.22*(1-meanSupport))
}

func summarizeTradeCellPath(cellPath []int, landRoutes *LandRouteResult) (float64, float64) {
	if landRoutes == nil || landRoutes.Diagnostics == nil || len(cellPath) == 0 {
		return 0, 0
	}
	diag := landRoutes.Diagnostics
	totalRisk := 0.0
	totalSupport := 0.0
	count := 0.0
	for _, cellIdx := range cellPath {
		if cellIdx < 0 || cellIdx >= len(diag.RouteRisk) {
			continue
		}
		totalRisk += diag.RouteRisk[cellIdx]
		totalSupport += diag.WaystationSuitability[cellIdx]
		count++
	}
	if count == 0 {
		return 0, 0
	}
	return totalRisk / count, totalSupport / count
}

func interCivilizationFlowMultiplier(landRoutes *LandRouteResult) float64 {
	if landRoutes == nil {
		return 1.0
	}
	mode := landRoutes.Mode
	return (0.62 + 0.45*mode.PayloadCapacity) * maxFloat(mode.InterCivilizationFlow, 0)
}

func internalFlowMultiplier(landRoutes *LandRouteResult) float64 {
	if landRoutes == nil {
		return 1.0
	}
	mode := landRoutes.Mode
	return (0.48 + 0.40*mode.PayloadCapacity + 0.20*mode.DailyRange) * maxFloat(mode.InternalFlow, 0)
}

func interCivilizationReachMultiplier(landRoutes *LandRouteResult) float64 {
	if landRoutes == nil {
		return 1.0
	}
	mode := landRoutes.Mode
	return 0.42 + 0.58*clamp01(0.55*mode.LongHaulTolerance+0.45*mode.DailyRange)
}

func internalReachMultiplier(landRoutes *LandRouteResult) float64 {
	if landRoutes == nil {
		return 1.0
	}
	mode := landRoutes.Mode
	return 0.55 + 0.45*clamp01(0.45*mode.LongHaulTolerance+0.55*mode.DailyRange)
}

func reversedInts(values []int) []int {
	out := append([]int(nil), values...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

