package climgen

import (
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

type TradeCorridorRole int

const (
	TradeCorridorRoleFeeder TradeCorridorRole = iota
	TradeCorridorRoleInternalTrunk
	TradeCorridorRoleInterPolityTrunk
)

func TradeCorridorRoleName(role TradeCorridorRole) string {
	names := []string{"Feeder", "Internal Trunk", "Inter-Polity Trunk"}
	if int(role) < len(names) {
		return names[role]
	}
	return "Unknown"
}

type TradeCorridor struct {
	ID                int
	FromLocalNode     int
	ToLocalNode       int
	FromNode          int
	ToNode            int
	FromCivilization  int
	ToCivilization    int
	Mode              string
	Role              TradeCorridorRole
	HandoffNode       int
	TravelCost        float64
	Flow              float64
	MeanRisk          float64
	MeanSupport       float64
	Tier              TradeCorridorTier
	NodePath          []int
	CellPath          []int
	InterCivilization bool
}

type TradeHandoff struct {
	Node            int
	FeederCorridors int
	TrunkCorridors  int
	DominantMode    string
	Hub             bool
}

type TradeNetworkDiagnostics struct {
	CivilizationByNode []int
	NodeCentrality     []float64
	TrunkCentrality    []float64
	FeederCentrality   []float64
	HubScore           []float64
	RouteIntensity     []float64
	RouteRiskIntensity []float64
}

type TradeNetworkResult struct {
	Corridors   []TradeCorridor
	Handoffs    []TradeHandoff
	MajorHubs   []int
	LocalNodes  []LocalTradeNode
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
	landRoutes *LandRouteResult,
	settings TradeNetworkSettings,
) *TradeNetworkResult {
	out := &TradeNetworkResult{}
	if network == nil || len(network.Nodes) == 0 || proto == nil || len(proto.Civilizations) == 0 {
		return out
	}
	out.Diagnostics = &TradeNetworkDiagnostics{
		CivilizationByNode: civilizationByNode(network, proto),
		NodeCentrality:     make([]float64, len(network.Nodes)),
		TrunkCentrality:    make([]float64, len(network.Nodes)),
		FeederCentrality:   make([]float64, len(network.Nodes)),
		HubScore:           make([]float64, len(network.Nodes)),
		RouteIntensity:     make([]float64, len(cells)),
		RouteRiskIntensity: make([]float64, len(cells)),
	}

	adj := buildTradeAdjacency(network, landRoutes, len(cells))
	localGraph := BuildLocalTradeGraph(cells, network, landRoutes)
	corridors := collectInterCivilizationCorridors(network, proto, adj, landRoutes, settings, len(cells))
	corridors = append(corridors, collectInternalCivilizationCorridors(network, proto, out.Diagnostics.CivilizationByNode, adj, landRoutes, settings)...)
	corridors = append(corridors, collectAnchorFeederCorridors(network, adj, landRoutes, settings)...)
	corridors = append(corridors, collectLocalFeederCorridors(cells, network, localGraph, landRoutes, settings)...)
	corridors = dedupeTradeCorridors(corridors)
	classifyTradeCorridors(corridors, settings)
	applyTradeDiagnostics(corridors, out.Diagnostics)
	out.MajorHubs = identifyMajorTradeHubs(network, proto, out.Diagnostics, settings)
	out.Handoffs = buildTradeHandoffs(corridors, out.MajorHubs)
	if localGraph != nil {
		out.LocalNodes = localGraph.Nodes
	}
	out.Corridors = corridors
	return out
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

func collectInterCivilizationCorridors(
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	adj [][]tradeAdjEdge,
	landRoutes *LandRouteResult,
	settings TradeNetworkSettings,
	meshCellCount int,
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
			path := shortestTradeNodePath(a.CenterNode, b.CenterNode, network, adj, settings.MaxRouteCost*interCivilizationReachMultiplier(landRoutes))
			if !path.ok {
				continue
			}
			flow := tradeFlowBetweenCivilizations(a, b, network.Nodes[a.CenterNode], network.Nodes[b.CenterNode], path.cost, meshCellCount)
			flow *= interCivilizationFlowMultiplier(landRoutes)
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
		out = append(out, buildTradeCorridor(network, cand.path, cand.from, cand.to, cand.flow, true, TradeCorridorRoleInterPolityTrunk, -1, landRoutes))
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
	landRoutes *LandRouteResult,
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
			path := shortestTradeNodePath(civ.CenterNode, targets[i].node, network, adj, settings.MaxRouteCost*0.75*internalReachMultiplier(landRoutes))
			if !path.ok {
				continue
			}
			flow := tradeFlowWithinCivilization(civ, network.Nodes[civ.CenterNode], network.Nodes[targets[i].node], path.cost)
			flow *= internalFlowMultiplier(landRoutes)
			if flow < settings.MinFlow {
				continue
			}
			out = append(out, buildTradeCorridor(network, path, civ.ID, civ.ID, flow, false, TradeCorridorRoleInternalTrunk, -1, landRoutes))
		}
	}
	return out
}

func collectAnchorFeederCorridors(
	network *SettlementNetworkResult,
	adj [][]tradeAdjEdge,
	landRoutes *LandRouteResult,
	settings TradeNetworkSettings,
) []TradeCorridor {
	if network == nil || landRoutes == nil || landRoutes.Mode.FeederFlow <= 0 || len(network.Regions) == 0 {
		return nil
	}
	maxCost := settings.MaxRouteCost * landRoutes.Mode.FeederReach
	if maxCost <= 0 {
		return nil
	}
	out := make([]TradeCorridor, 0)
	for _, region := range network.Regions {
		candidates := make([]int, 0, len(region.NodeIndices))
		for _, nodeIdx := range region.NodeIndices {
			if nodeIdx < 0 || nodeIdx >= len(network.Nodes) {
				continue
			}
			node := network.Nodes[nodeIdx]
			if node.Kind > SettlementNodeVillage {
				continue
			}
			candidates = append(candidates, nodeIdx)
		}
		for _, fromIdx := range candidates {
			bestPath := tradeNodePath{}
			bestTarget := -1
			for _, toIdx := range region.NodeIndices {
				if toIdx == fromIdx || toIdx < 0 || toIdx >= len(network.Nodes) {
					continue
				}
				fromNode := network.Nodes[fromIdx]
				toNode := network.Nodes[toIdx]
				if toNode.Kind < fromNode.Kind || (toNode.Kind == fromNode.Kind && toNode.Score <= fromNode.Score) {
					continue
				}
				path := shortestTradeNodePath(fromIdx, toIdx, network, adj, maxCost)
				if !path.ok {
					continue
				}
				if bestTarget < 0 || path.cost < bestPath.cost {
					bestTarget = toIdx
					bestPath = path
				}
			}
			if bestTarget < 0 {
				continue
			}
			flow := localFeederFlow(network.Nodes[fromIdx], network.Nodes[bestTarget], bestPath.cost, landRoutes.Mode)
			if flow < settings.MinFlow*0.45 {
				continue
			}
			out = append(out, buildTradeCorridor(network, bestPath, -1, -1, flow, false, TradeCorridorRoleFeeder, bestTarget, landRoutes))
		}
	}
	return out
}

func tradeFlowBetweenCivilizations(a, b ProtoCivilization, centerA, centerB SettlementNode, cost float64, meshCellCount int) float64 {
	base := math.Sqrt(meshScaledTerritoryAreaCells(a.TerritoryCells, meshCellCount)*meshScaledTerritoryAreaCells(b.TerritoryCells, meshCellCount)) / 32.0
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

func localFeederFlow(from, to SettlementNode, cost float64, mode LandRouteModeSettings) float64 {
	base := 0.20 + 0.22*from.Score + 0.18*to.Score + 0.08*float64(from.Kind) + 0.06*float64(to.Kind)
	return base * maxFloat(mode.FeederFlow, 0) / math.Max(cost*1.15, 1.0)
}
