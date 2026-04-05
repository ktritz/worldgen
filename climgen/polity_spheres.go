package climgen

import "math"

type PolityRelationType int

const (
	PolityRelationSuzerain PolityRelationType = iota
)

func PolityRelationTypeName(kind PolityRelationType) string {
	switch kind {
	case PolityRelationSuzerain:
		return "Suzerainty"
	default:
		return "Unknown"
	}
}

type PolitySphere struct {
	ID                  int
	ProtoCivilizationID int
	CapitalNode         int
	Secondary           bool
	Style               ProtoCivilizationStyle
	Coastal             bool
	River               bool
	TerritoryCells      int
	CoreCells           int
	MeanSupport         float64
	MeanInfluence       float64
}

type PolitySphereRelation struct {
	Kind     PolityRelationType
	Overlord int
	Subject  int
	Strength float64
}

type PolitySphereDiagnostics struct {
	PolityByCell  []int
	InfluenceCost []float64
	CapitalByNode []bool
	PolityByNode  []int
}

type PolitySphereResult struct {
	Spheres     []PolitySphere
	Relations   []PolitySphereRelation
	MergedMinor int
	Diagnostics *PolitySphereDiagnostics
}

type PolitySphereSettings struct {
	SecondaryHubThreshold      float64
	SecondaryLargeHubThreshold float64
	SecondaryLargeProtoCells   int
	SecondaryMinDistance       float64
	MinTerritoryCells          int
	ClaimBaseTravel            float64
	ClaimMaxTravel             float64
	CoreCarryThreshold         float64
}

func DefaultPolitySphereSettings() PolitySphereSettings {
	return PolitySphereSettings{
		SecondaryHubThreshold:      0.98,
		SecondaryLargeHubThreshold: 0.82,
		SecondaryLargeProtoCells:   140,
		SecondaryMinDistance:       8.0,
		MinTerritoryCells:          18,
		ClaimBaseTravel:            11.5,
		ClaimMaxTravel:             28.0,
		CoreCarryThreshold:         0.44,
	}
}

func BuildPolitySpheres(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	trade *TradeNetworkResult,
	population *PopulationResult,
	settlements *SettlementResult,
	elevation []float64,
	seaLevel float64,
	settings PolitySphereSettings,
) *PolitySphereResult {
	out := &PolitySphereResult{
		Diagnostics: &PolitySphereDiagnostics{
			PolityByCell:  make([]int, len(elevation)),
			InfluenceCost: make([]float64, len(elevation)),
			CapitalByNode: make([]bool, len(network.Nodes)),
			PolityByNode:  make([]int, len(network.Nodes)),
		},
	}
	for i := range out.Diagnostics.PolityByCell {
		out.Diagnostics.PolityByCell[i] = -1
		out.Diagnostics.InfluenceCost[i] = math.Inf(1)
	}
	for i := range out.Diagnostics.PolityByNode {
		out.Diagnostics.PolityByNode[i] = -1
	}
	if network == nil || proto == nil || trade == nil || population == nil || settlements == nil {
		return out
	}

	spheres := politySphereSeeds(network, proto, trade, settings)
	initialSphereCount := len(spheres)
	if len(spheres) == 0 {
		return out
	}
	for _, sphere := range spheres {
		if sphere.CapitalNode >= 0 && sphere.CapitalNode < len(out.Diagnostics.CapitalByNode) {
			out.Diagnostics.CapitalByNode[sphere.CapitalNode] = true
		}
	}

	assignPolitySphereTerritories(cells, network, trade, spheres, population, settlements, elevation, seaLevel, out.Diagnostics, settings)
	spheres = finalizePolitySpheres(spheres, out.Diagnostics, population, settings)
	out.MergedMinor = initialSphereCount - len(spheres)
	out.Relations = buildPolitySphereRelations(spheres)
	out.Spheres = spheres
	for i, sphere := range spheres {
		if sphere.CapitalNode >= 0 && sphere.CapitalNode < len(out.Diagnostics.PolityByNode) {
			out.Diagnostics.PolityByNode[sphere.CapitalNode] = i
		}
	}
	return out
}

func politySphereSeeds(
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	trade *TradeNetworkResult,
	settings PolitySphereSettings,
) []PolitySphere {
	spheres := make([]PolitySphere, 0, len(proto.Civilizations))
	existingCapital := make(map[int]struct{}, len(proto.Civilizations))
	for _, civ := range proto.Civilizations {
		spheres = append(spheres, PolitySphere{
			ID:                  len(spheres),
			ProtoCivilizationID: civ.ID,
			CapitalNode:         civ.CenterNode,
			Style:               civ.Style,
			Coastal:             civ.Coastal,
			River:               civ.River,
		})
		existingCapital[civ.CenterNode] = struct{}{}
	}

	adj := buildTradeAdjacency(network)
	for _, civ := range proto.Civilizations {
		bestNode := -1
		bestScore := -1.0
		for nodeIdx, owner := range trade.Diagnostics.CivilizationByNode {
			if owner != civ.ID {
				continue
			}
			if _, ok := existingCapital[nodeIdx]; ok {
				continue
			}
			node := network.Nodes[nodeIdx]
			if node.Kind < SettlementNodeVillage {
				continue
			}
			hubScore := 0.0
			if nodeIdx < len(trade.Diagnostics.HubScore) {
				hubScore = trade.Diagnostics.HubScore[nodeIdx]
			}
			threshold := settings.SecondaryHubThreshold
			if civ.TerritoryCells >= settings.SecondaryLargeProtoCells {
				threshold = settings.SecondaryLargeHubThreshold
			}
			if hubScore < threshold {
				continue
			}
			path := shortestTradeNodePath(civ.CenterNode, nodeIdx, network, adj, settings.ClaimMaxTravel)
			if !path.ok || path.cost < settings.SecondaryMinDistance {
				continue
			}
			score := hubScore + 0.02*math.Min(path.cost, 12.0)
			if score > bestScore {
				bestScore = score
				bestNode = nodeIdx
			}
		}
		if bestNode >= 0 {
			node := network.Nodes[bestNode]
			spheres = append(spheres, PolitySphere{
				ID:                  len(spheres),
				ProtoCivilizationID: civ.ID,
				CapitalNode:         bestNode,
				Secondary:           true,
				Style:               civ.Style,
				Coastal:             node.Coastal,
				River:               node.River,
			})
			existingCapital[bestNode] = struct{}{}
		}
	}
	return spheres
}

func assignPolitySphereTerritories(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	spheres []PolitySphere,
	population *PopulationResult,
	settlements *SettlementResult,
	elevation []float64,
	seaLevel float64,
	diagnostics *PolitySphereDiagnostics,
	settings PolitySphereSettings,
) {
	for sphereIdx, sphere := range spheres {
		node := network.Nodes[sphere.CapitalNode]
		maxTravel := polityClaimLimit(node, sphere, trade, settings)
		dist, _ := shortestPathsFromNode(node.CellIndex, cells, network.Diagnostics.MovementCost, maxTravel)
		centrality := 0.0
		if trade.Diagnostics != nil && sphere.CapitalNode < len(trade.Diagnostics.NodeCentrality) {
			centrality = trade.Diagnostics.NodeCentrality[sphere.CapitalNode]
		}
		influence := 1.0 + 0.30*node.Score + 0.14*float64(node.Kind) + 0.18*centrality
		if sphere.Secondary {
			influence *= 0.88
		}
		for cellIdx := range elevation {
			if !claimablePolityCell(cellIdx, settlements, population, elevation, seaLevel) {
				continue
			}
			if math.IsInf(dist[cellIdx], 1) {
				continue
			}
			adjusted := dist[cellIdx] / influence
			if adjusted < diagnostics.InfluenceCost[cellIdx] {
				diagnostics.InfluenceCost[cellIdx] = adjusted
				diagnostics.PolityByCell[cellIdx] = sphereIdx
			}
		}
	}
}

func claimablePolityCell(idx int, settlements *SettlementResult, population *PopulationResult, elevation []float64, seaLevel float64) bool {
	if idx < 0 || idx >= len(elevation) || elevation[idx] < seaLevel {
		return false
	}
	if population.Classes[idx] == PopulationUninhabited {
		return false
	}
	if settlements.Classes[idx] < SettlementMarginal && population.Diagnostics.CarryingCapacity[idx] < 0.16 {
		return false
	}
	return true
}

func polityClaimLimit(node SettlementNode, sphere PolitySphere, trade *TradeNetworkResult, settings PolitySphereSettings) float64 {
	centrality := 0.0
	if trade.Diagnostics != nil && node.ID < len(trade.Diagnostics.NodeCentrality) {
		centrality = trade.Diagnostics.NodeCentrality[node.ID]
	}
	limit := settings.ClaimBaseTravel + 2.0*float64(node.Kind) + 4.0*math.Min(centrality, 1.0)
	if sphere.Coastal {
		limit += 1.2
	}
	if sphere.River {
		limit += 1.0
	}
	if sphere.Secondary {
		limit -= 1.0
	}
	return math.Min(limit, settings.ClaimMaxTravel)
}

func finalizePolitySpheres(
	spheres []PolitySphere,
	diagnostics *PolitySphereDiagnostics,
	population *PopulationResult,
	settings PolitySphereSettings,
) []PolitySphere {
	territory := make([]int, len(spheres))
	core := make([]int, len(spheres))
	support := make([]float64, len(spheres))
	influence := make([]float64, len(spheres))
	for cellIdx, polityIdx := range diagnostics.PolityByCell {
		if polityIdx < 0 || polityIdx >= len(spheres) {
			continue
		}
		territory[polityIdx]++
		support[polityIdx] += population.Diagnostics.CarryingCapacity[cellIdx]
		influence[polityIdx] += diagnostics.InfluenceCost[cellIdx]
		if population.Diagnostics.CarryingCapacity[cellIdx] >= settings.CoreCarryThreshold {
			core[polityIdx]++
		}
	}

	filtered := make([]PolitySphere, 0, len(spheres))
	oldToNew := make([]int, len(spheres))
	for i := range oldToNew {
		oldToNew[i] = -1
	}
	for i, sphere := range spheres {
		if territory[i] < settings.MinTerritoryCells {
			continue
		}
		sphere.ID = len(filtered)
		sphere.TerritoryCells = territory[i]
		sphere.CoreCells = core[i]
		sphere.MeanSupport = support[i] / math.Max(float64(territory[i]), 1)
		sphere.MeanInfluence = influence[i] / math.Max(float64(territory[i]), 1)
		oldToNew[i] = sphere.ID
		filtered = append(filtered, sphere)
	}
	for i, polityIdx := range diagnostics.PolityByCell {
		if polityIdx >= 0 {
			diagnostics.PolityByCell[i] = oldToNew[polityIdx]
		}
	}
	return filtered
}

func buildPolitySphereRelations(spheres []PolitySphere) []PolitySphereRelation {
	primaryByProto := make(map[int]int)
	for i, sphere := range spheres {
		if sphere.Secondary {
			continue
		}
		current, ok := primaryByProto[sphere.ProtoCivilizationID]
		if !ok || spheres[current].TerritoryCells < sphere.TerritoryCells {
			primaryByProto[sphere.ProtoCivilizationID] = i
		}
	}
	relations := make([]PolitySphereRelation, 0)
	for i, sphere := range spheres {
		if !sphere.Secondary {
			continue
		}
		overlord, ok := primaryByProto[sphere.ProtoCivilizationID]
		if !ok || overlord == i {
			continue
		}
		strength := 0.40 + 0.35*math.Min(sphere.MeanSupport, 1.0)
		if spheres[overlord].TerritoryCells > 0 {
			ratio := float64(spheres[overlord].TerritoryCells) / math.Max(float64(sphere.TerritoryCells), 1.0)
			strength += 0.20 * math.Min(ratio, 2.0) / 2.0
		}
		relations = append(relations, PolitySphereRelation{
			Kind:     PolityRelationSuzerain,
			Overlord: overlord,
			Subject:  i,
			Strength: strength,
		})
	}
	return relations
}
