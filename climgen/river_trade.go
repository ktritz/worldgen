package climgen

import (
	"math"
	"sort"
)

type RiverTradeCorridorTier int

const (
	RiverTradeCorridorLocal RiverTradeCorridorTier = iota
	RiverTradeCorridorRegional
	RiverTradeCorridorPrimary
)

func RiverTradeCorridorTierName(tier RiverTradeCorridorTier) string {
	names := []string{"Local River Route", "Regional River Route", "Primary River Route"}
	if int(tier) < len(names) {
		return names[tier]
	}
	return "Unknown"
}

type RiverTradeCorridor struct {
	ID                int
	FromNode          int
	ToNode            int
	FromCivilization  int
	ToCivilization    int
	TravelCost        float64
	Flow              float64
	MeanNavigability  float64
	MeanTransfer      float64
	Tier              RiverTradeCorridorTier
	NodePath          []int
	CellPath          []int
	InterCivilization bool
}

type RiverTradeDiagnostics struct {
	RouteIntensity []float64
	NodeCentrality []float64
}

type RiverTradeResult struct {
	Corridors   []RiverTradeCorridor
	MajorPorts  []int
	Diagnostics *RiverTradeDiagnostics
}

type RiverTradeSettings struct {
	MaxPartnersPerCivilization int
	MaxInternalCorridors       int
	MaxRouteCost               float64
	MinFlow                    float64
	RegionalFlow               float64
	PrimaryFlow                float64
	PortThreshold              float64
}

func DefaultRiverTradeSettings() RiverTradeSettings {
	return RiverTradeSettings{
		MaxPartnersPerCivilization: 2,
		MaxInternalCorridors:       2,
		MaxRouteCost:               26.0,
		MinFlow:                    0.05,
		RegionalFlow:               0.10,
		PrimaryFlow:                0.18,
		PortThreshold:              0.58,
	}
}

func BuildRiverTradeNetwork(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	riverRoutes *RiverRouteResult,
	elevation []float64,
	settings RiverTradeSettings,
) *RiverTradeResult {
	out := &RiverTradeResult{}
	if network == nil || proto == nil || riverRoutes == nil || riverRoutes.Diagnostics == nil {
		return out
	}
	out.Diagnostics = &RiverTradeDiagnostics{
		RouteIntensity: make([]float64, len(cells)),
		NodeCentrality: make([]float64, len(network.Nodes)),
	}
	civByNode := civilizationByNode(network, proto)
	riverNodes := eligibleRiverTradeNodes(network, riverRoutes)
	if len(riverNodes) < 2 {
		return out
	}
	transitNodes := eligibleRiverTransitNodes(network, riverRoutes)
	adj := buildRiverTradeAdjacency(cells, network, riverRoutes, transitNodes, elevation)
	out.Corridors = collectRiverTradeCorridors(cells, network, proto, civByNode, adj, riverNodes, riverRoutes, settings)
	classifyRiverTradeCorridors(out.Corridors, settings)
	applyRiverTradeDiagnostics(out.Corridors, out.Diagnostics)
	out.MajorPorts = identifyMajorRiverPorts(network, riverNodes, out.Diagnostics, settings)
	return out
}

func eligibleRiverTradeNodes(network *SettlementNetworkResult, riverRoutes *RiverRouteResult) map[int]struct{} {
	out := map[int]struct{}{}
	if network == nil || riverRoutes == nil || riverRoutes.Diagnostics == nil {
		return out
	}
	for _, node := range network.Nodes {
		if node.CellIndex < 0 || node.CellIndex >= len(riverRoutes.Diagnostics.Navigability) {
			continue
		}
		navigability := riverRoutes.Diagnostics.Navigability[node.CellIndex]
		transfer := riverRoutes.Diagnostics.TransferSupport[node.CellIndex]
		if node.River && navigability >= riverRoutes.Mode.MinNavigability*0.60 {
			out[node.ID] = struct{}{}
			continue
		}
		if navigability >= riverRoutes.Mode.MinNavigability*0.85 && transfer >= riverRoutes.Mode.TransferSupportFloor*0.85 {
			out[node.ID] = struct{}{}
		}
	}
	return out
}

func eligibleRiverTransitNodes(network *SettlementNetworkResult, riverRoutes *RiverRouteResult) map[int]struct{} {
	out := eligibleRiverTradeNodes(network, riverRoutes)
	if network == nil || riverRoutes == nil || riverRoutes.Diagnostics == nil {
		return out
	}
	for _, node := range network.Nodes {
		if node.CellIndex < 0 ||
			node.CellIndex >= len(riverRoutes.Diagnostics.TransferSupport) ||
			node.CellIndex >= len(riverRoutes.Diagnostics.PortageSuitability) ||
			node.CellIndex >= len(riverRoutes.Diagnostics.Navigability) {
			continue
		}
		transfer := riverRoutes.Diagnostics.TransferSupport[node.CellIndex]
		portage := riverRoutes.Diagnostics.PortageSuitability[node.CellIndex]
		nav := riverRoutes.Diagnostics.Navigability[node.CellIndex]
		if nav >= riverRoutes.Mode.MinNavigability*0.45 {
			out[node.ID] = struct{}{}
			continue
		}
		if transfer >= riverRoutes.Mode.TransferSupportFloor*0.80 && portage >= 0.22 {
			out[node.ID] = struct{}{}
		}
	}
	return out
}

func collectRiverTradeCorridors(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	civByNode []int,
	adj [][]riverAdjEdge,
	riverNodes map[int]struct{},
	riverRoutes *RiverRouteResult,
	settings RiverTradeSettings,
) []RiverTradeCorridor {
	type candidate struct {
		from int
		to   int
		path riverNodePath
		flow float64
	}
	cands := make([]candidate, 0)
	for i := 0; i < len(proto.Civilizations); i++ {
		for j := i + 1; j < len(proto.Civilizations); j++ {
			a := proto.Civilizations[i]
			b := proto.Civilizations[j]
			aNodes := topRiverNodesForCivilization(network, civByNode, riverNodes, a.ID, 3)
			bNodes := topRiverNodesForCivilization(network, civByNode, riverNodes, b.ID, 3)
			if len(aNodes) == 0 || len(bNodes) == 0 {
				continue
			}
			for _, startNode := range aNodes {
				for _, endNode := range bNodes {
					startCell := network.Nodes[startNode].CellIndex
					endCell := network.Nodes[endNode].CellIndex
					cellPath := shortestRiverCellPath(cells, startCell, endCell, riverRoutes, settings.MaxRouteCost*(0.58+0.42*riverRoutes.Mode.LongHaulTolerance))
					if !cellPath.ok {
						continue
					}
					flow := riverTradeFlowBetweenCivilizations(a, b, network.Nodes[startNode], network.Nodes[endNode], cellPath.cost, riverRoutes.Mode)
					if flow < settings.MinFlow {
						continue
					}
					path := riverNodePath{ok: true, cost: cellPath.cost, nodes: []int{startNode, endNode}}
					cands = append(cands, candidate{from: i, to: j, path: path, flow: flow})
				}
			}
		}
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].flow > cands[j].flow })
	used := make([]int, len(proto.Civilizations))
	out := make([]RiverTradeCorridor, 0)
	for _, cand := range cands {
		if used[cand.from] >= settings.MaxPartnersPerCivilization || used[cand.to] >= settings.MaxPartnersPerCivilization {
			continue
		}
		startCell := network.Nodes[cand.path.nodes[0]].CellIndex
		endCell := network.Nodes[cand.path.nodes[len(cand.path.nodes)-1]].CellIndex
		cellPath := shortestRiverCellPath(cells, startCell, endCell, riverRoutes, settings.MaxRouteCost*(0.58+0.42*riverRoutes.Mode.LongHaulTolerance))
		if !cellPath.ok {
			continue
		}
		out = append(out, buildRiverTradeCorridorFromCells(cand.path.nodes[0], cand.path.nodes[len(cand.path.nodes)-1], cand.from, cand.to, cand.flow, true, cellPath, riverRoutes))
		used[cand.from]++
		used[cand.to]++
	}

	for _, civ := range proto.Civilizations {
		if _, ok := riverNodes[civ.CenterNode]; !ok {
			continue
		}
		type target struct {
			node  int
			score float64
		}
		targets := make([]target, 0)
		for nodeIdx := range riverNodes {
			if nodeIdx == civ.CenterNode {
				continue
			}
			if nodeIdx < 0 || nodeIdx >= len(civByNode) || civByNode[nodeIdx] != civ.ID {
				continue
			}
			node := network.Nodes[nodeIdx]
			score := node.Score + 0.10*node.CarryingCapacity + 0.08*node.UrbanPotential
			targets = append(targets, target{node: nodeIdx, score: score})
		}
		sort.Slice(targets, func(i, j int) bool { return targets[i].score > targets[j].score })
		limit := settings.MaxInternalCorridors
		if len(targets) < limit {
			limit = len(targets)
		}
		for i := 0; i < limit; i++ {
			path := shortestRiverNodePath(civ.CenterNode, targets[i].node, network, adj, settings.MaxRouteCost*0.72)
			if !path.ok {
				continue
			}
			flow := riverTradeFlowWithinCivilization(civ, network.Nodes[civ.CenterNode], network.Nodes[targets[i].node], path.cost, riverRoutes.Mode)
			if flow < settings.MinFlow {
				continue
			}
			out = append(out, buildRiverTradeCorridor(network, path, civ.ID, civ.ID, flow, false, riverRoutes))
		}
	}
	out = dedupeRiverTradeCorridors(out)
	return out
}

func topRiverNodesForCivilization(
	network *SettlementNetworkResult,
	civByNode []int,
	riverNodes map[int]struct{},
	civID int,
	limit int,
) []int {
	type scoredNode struct {
		node  int
		score float64
	}
	cands := make([]scoredNode, 0)
	for nodeIdx := range riverNodes {
		if nodeIdx < 0 || nodeIdx >= len(civByNode) || civByNode[nodeIdx] != civID {
			continue
		}
		node := network.Nodes[nodeIdx]
		score := node.Score + 0.12*node.CarryingCapacity + 0.10*node.UrbanPotential
		if node.River {
			score += 0.08
		}
		if node.Coastal {
			score += 0.04
		}
		cands = append(cands, scoredNode{node: nodeIdx, score: score})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	if len(cands) > limit {
		cands = cands[:limit]
	}
	out := make([]int, 0, len(cands))
	for _, cand := range cands {
		out = append(out, cand.node)
	}
	return out
}

func riverTradeFlowBetweenCivilizations(a, b ProtoCivilization, centerA, centerB SettlementNode, cost float64, mode RiverRouteModeSettings) float64 {
	base := math.Sqrt(float64(maxInt(a.TerritoryCells, 1))*float64(maxInt(b.TerritoryCells, 1))) / 36.0
	support := 0.60 + 0.45*(a.MeanSupport+b.MeanSupport)
	nodeScore := 0.45 + 0.40*(centerA.Score+centerB.Score)
	bonus := 1.0 + 0.26*mode.PayloadCapacity + 0.18*mode.LongHaulTolerance
	return base * support * nodeScore * bonus / math.Max(cost, 1.0)
}

func riverTradeFlowWithinCivilization(civ ProtoCivilization, center, node SettlementNode, cost float64, mode RiverRouteModeSettings) float64 {
	base := (0.46 + civ.MeanSupport) * (0.42 + center.Score + node.Score)
	bonus := 1.0 + 0.18*mode.PayloadCapacity + 0.12*mode.DailyRange
	return base * bonus / math.Max(cost*1.2, 1.0)
}
