package climgen

import (
	"container/heap"
	"math"
	"sort"
)

type TradeCorridorTier int

const (
	TradeCorridorLocal TradeCorridorTier = iota
	TradeCorridorRegional
	TradeCorridorPrimary
)

func TradeCorridorTierName(tier TradeCorridorTier) string {
	names := []string{"Local Corridor", "Regional Corridor", "Primary Corridor"}
	if int(tier) < len(names) {
		return names[tier]
	}
	return "Unknown"
}

type TradeCorridor struct {
	ID                int
	FromNode          int
	ToNode            int
	FromCivilization  int
	ToCivilization    int
	TravelCost        float64
	Flow              float64
	Tier              TradeCorridorTier
	NodePath          []int
	CellPath          []int
	InterCivilization bool
}

type TradeNetworkDiagnostics struct {
	CivilizationByNode []int
	NodeCentrality     []float64
	HubScore           []float64
	RouteIntensity     []float64
}

type TradeNetworkResult struct {
	Corridors   []TradeCorridor
	MajorHubs   []int
	Diagnostics *TradeNetworkDiagnostics
}

type TradeNetworkSettings struct {
	MaxPartnersPerCivilization int
	MaxInternalCorridors       int
	MaxRouteCost               float64
	MinFlow                    float64
	RegionalFlow               float64
	PrimaryFlow                float64
	HubThreshold               float64
}

func DefaultTradeNetworkSettings() TradeNetworkSettings {
	return TradeNetworkSettings{
		MaxPartnersPerCivilization: 2,
		MaxInternalCorridors:       2,
		MaxRouteCost:               28.0,
		MinFlow:                    0.06,
		RegionalFlow:               0.12,
		PrimaryFlow:                0.20,
		HubThreshold:               0.86,
	}
}

func BuildTradeNetwork(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	settings TradeNetworkSettings,
) *TradeNetworkResult {
	out := &TradeNetworkResult{}
	if network == nil || len(network.Nodes) == 0 || proto == nil || len(proto.Civilizations) == 0 {
		return out
	}
	out.Diagnostics = &TradeNetworkDiagnostics{
		CivilizationByNode: civilizationByNode(network, proto),
		NodeCentrality:     make([]float64, len(network.Nodes)),
		HubScore:           make([]float64, len(network.Nodes)),
		RouteIntensity:     make([]float64, len(cells)),
	}

	adj := buildTradeAdjacency(network)
	corridors := collectInterCivilizationCorridors(network, proto, adj, settings)
	corridors = append(corridors, collectInternalCivilizationCorridors(network, proto, out.Diagnostics.CivilizationByNode, adj, settings)...)
	corridors = dedupeTradeCorridors(corridors)
	classifyTradeCorridors(corridors, settings)
	applyTradeDiagnostics(corridors, out.Diagnostics)
	out.MajorHubs = identifyMajorTradeHubs(network, proto, out.Diagnostics, settings)
	out.Corridors = corridors
	return out
}

type tradeAdjEdge struct {
	neighbor  int
	linkIndex int
	cost      float64
}

func civilizationByNode(network *SettlementNetworkResult, proto *ProtoCivilizationResult) []int {
	byNode := make([]int, len(network.Nodes))
	for i := range byNode {
		byNode[i] = -1
	}
	regionToCiv := make(map[int]int, len(proto.Civilizations))
	for _, civ := range proto.Civilizations {
		regionToCiv[civ.RegionID] = civ.ID
	}
	for _, region := range network.Regions {
		civID, ok := regionToCiv[region.ID]
		if !ok {
			continue
		}
		for _, nodeIdx := range region.NodeIndices {
			if nodeIdx >= 0 && nodeIdx < len(byNode) {
				byNode[nodeIdx] = civID
			}
		}
	}
	return byNode
}

func buildTradeAdjacency(network *SettlementNetworkResult) [][]tradeAdjEdge {
	adj := make([][]tradeAdjEdge, len(network.Nodes))
	for i, link := range network.Links {
		if link.From < 0 || link.From >= len(adj) || link.To < 0 || link.To >= len(adj) {
			continue
		}
		adj[link.From] = append(adj[link.From], tradeAdjEdge{neighbor: link.To, linkIndex: i, cost: link.TravelCost})
		adj[link.To] = append(adj[link.To], tradeAdjEdge{neighbor: link.From, linkIndex: i, cost: link.TravelCost})
	}
	return adj
}

func collectInterCivilizationCorridors(
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	adj [][]tradeAdjEdge,
	settings TradeNetworkSettings,
) []TradeCorridor {
	type candidate struct {
		from int
		to   int
		path tradeNodePath
		flow float64
	}
	candidates := make([]candidate, 0)
	for i := 0; i < len(proto.Civilizations); i++ {
		for j := i + 1; j < len(proto.Civilizations); j++ {
			a := proto.Civilizations[i]
			b := proto.Civilizations[j]
			path := shortestTradeNodePath(a.CenterNode, b.CenterNode, network, adj, settings.MaxRouteCost)
			if !path.ok {
				continue
			}
			flow := tradeFlowBetweenCivilizations(a, b, network.Nodes[a.CenterNode], network.Nodes[b.CenterNode], path.cost)
			if flow < settings.MinFlow {
				continue
			}
			candidates = append(candidates, candidate{from: i, to: j, path: path, flow: flow})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].flow > candidates[j].flow })
	used := make([]int, len(proto.Civilizations))
	out := make([]TradeCorridor, 0)
	for _, cand := range candidates {
		if used[cand.from] >= settings.MaxPartnersPerCivilization || used[cand.to] >= settings.MaxPartnersPerCivilization {
			continue
		}
		out = append(out, buildTradeCorridor(network, cand.path, cand.from, cand.to, cand.flow, true))
		used[cand.from]++
		used[cand.to]++
	}
	return out
}

func collectInternalCivilizationCorridors(
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	civByNode []int,
	adj [][]tradeAdjEdge,
	settings TradeNetworkSettings,
) []TradeCorridor {
	out := make([]TradeCorridor, 0)
	for _, civ := range proto.Civilizations {
		type target struct {
			node  int
			score float64
		}
		targets := make([]target, 0)
		for nodeIdx, owner := range civByNode {
			if owner != civ.ID || nodeIdx == civ.CenterNode {
				continue
			}
			node := network.Nodes[nodeIdx]
			score := node.Score + 0.10*node.CarryingCapacity + 0.08*node.UrbanPotential + 0.04*float64(node.Kind)
			if node.Coastal || node.River {
				score += 0.04
			}
			targets = append(targets, target{node: nodeIdx, score: score})
		}
		sort.Slice(targets, func(i, j int) bool { return targets[i].score > targets[j].score })
		limit := settings.MaxInternalCorridors
		if len(targets) < limit {
			limit = len(targets)
		}
		for i := 0; i < limit; i++ {
			path := shortestTradeNodePath(civ.CenterNode, targets[i].node, network, adj, settings.MaxRouteCost*0.75)
			if !path.ok {
				continue
			}
			flow := tradeFlowWithinCivilization(civ, network.Nodes[civ.CenterNode], network.Nodes[targets[i].node], path.cost)
			if flow < settings.MinFlow {
				continue
			}
			out = append(out, buildTradeCorridor(network, path, civ.ID, civ.ID, flow, false))
		}
	}
	return out
}

func tradeFlowBetweenCivilizations(a, b ProtoCivilization, centerA, centerB SettlementNode, cost float64) float64 {
	base := math.Sqrt(float64(maxInt(a.TerritoryCells, 1))*float64(maxInt(b.TerritoryCells, 1))) / 32.0
	support := 0.6 + 0.5*(a.MeanSupport+b.MeanSupport)
	nodeScore := 0.4 + 0.45*(centerA.Score+centerB.Score)
	bonus := 1.0
	if a.Coastal && b.Coastal {
		bonus += 0.22
	}
	if a.River || b.River {
		bonus += 0.12
	}
	return base * support * nodeScore * bonus / math.Max(cost, 1.0)
}

func tradeFlowWithinCivilization(civ ProtoCivilization, center, node SettlementNode, cost float64) float64 {
	base := (0.55 + civ.MeanSupport) * (0.40 + center.Score + node.Score)
	bonus := 1.0 + 0.10*float64(node.Kind)
	if center.River || node.River {
		bonus += 0.08
	}
	if center.Coastal && node.Coastal {
		bonus += 0.10
	}
	return base * bonus / math.Max(cost*1.3, 1.0)
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

func buildTradeCorridor(network *SettlementNetworkResult, path tradeNodePath, fromCiv, toCiv int, flow float64, inter bool) TradeCorridor {
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
	return TradeCorridor{
		FromNode:          path.nodes[0],
		ToNode:            path.nodes[len(path.nodes)-1],
		FromCivilization:  fromCiv,
		ToCivilization:    toCiv,
		TravelCost:        path.cost,
		Flow:              flow,
		NodePath:          append([]int(nil), path.nodes...),
		CellPath:          cellPath,
		InterCivilization: inter,
	}
}

func reversedInts(values []int) []int {
	out := append([]int(nil), values...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func dedupeTradeCorridors(corridors []TradeCorridor) []TradeCorridor {
	type key struct{ a, b int }
	best := make(map[key]TradeCorridor)
	for _, corridor := range corridors {
		k := key{a: corridor.FromNode, b: corridor.ToNode}
		if k.a > k.b {
			k.a, k.b = k.b, k.a
		}
		current, ok := best[k]
		if !ok || corridor.Flow > current.Flow {
			best[k] = corridor
		}
	}
	out := make([]TradeCorridor, 0, len(best))
	for _, corridor := range best {
		out = append(out, corridor)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Flow > out[j].Flow })
	for i := range out {
		out[i].ID = i
	}
	return out
}

func classifyTradeCorridors(corridors []TradeCorridor, settings TradeNetworkSettings) {
	for i := range corridors {
		switch {
		case corridors[i].Flow >= settings.PrimaryFlow:
			corridors[i].Tier = TradeCorridorPrimary
		case corridors[i].Flow >= settings.RegionalFlow || corridors[i].InterCivilization:
			corridors[i].Tier = TradeCorridorRegional
		default:
			corridors[i].Tier = TradeCorridorLocal
		}
	}
}

func applyTradeDiagnostics(corridors []TradeCorridor, diagnostics *TradeNetworkDiagnostics) {
	if diagnostics == nil {
		return
	}
	for _, corridor := range corridors {
		for _, nodeIdx := range corridor.NodePath {
			if nodeIdx >= 0 && nodeIdx < len(diagnostics.NodeCentrality) {
				diagnostics.NodeCentrality[nodeIdx] += corridor.Flow
			}
		}
		for _, cellIdx := range corridor.CellPath {
			if cellIdx >= 0 && cellIdx < len(diagnostics.RouteIntensity) {
				diagnostics.RouteIntensity[cellIdx] += corridor.Flow
			}
		}
	}
}

func identifyMajorTradeHubs(
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	diagnostics *TradeNetworkDiagnostics,
	settings TradeNetworkSettings,
) []int {
	maxCentrality := 0.0
	for _, value := range diagnostics.NodeCentrality {
		if value > maxCentrality {
			maxCentrality = value
		}
	}
	hubs := make([]int, 0)
	for i, node := range network.Nodes {
		centerBonus := 0.0
		for _, civ := range proto.Civilizations {
			if civ.CenterNode == i {
				centerBonus = 0.12
				break
			}
		}
		centralityNorm := 0.0
		if maxCentrality > 0 {
			centralityNorm = diagnostics.NodeCentrality[i] / maxCentrality
		}
		score := node.Score + 0.18*(float64(node.Kind)/3.0) + 0.24*centralityNorm + centerBonus
		if node.Coastal {
			score += 0.05
		}
		if node.River {
			score += 0.04
		}
		diagnostics.HubScore[i] = score
		if node.Kind >= SettlementNodeTown && score >= settings.HubThreshold {
			hubs = append(hubs, i)
		}
	}
	sort.Slice(hubs, func(i, j int) bool {
		return diagnostics.HubScore[hubs[i]] > diagnostics.HubScore[hubs[j]]
	})
	return hubs
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
