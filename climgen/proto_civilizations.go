package climgen

import "math"

type ProtoCivilizationStyle int

const (
	ProtoCivilizationInland ProtoCivilizationStyle = iota
	ProtoCivilizationRiverine
	ProtoCivilizationMaritime
	ProtoCivilizationHighland
	ProtoCivilizationArid
)

func ProtoCivilizationStyleName(style ProtoCivilizationStyle) string {
	names := []string{
		"Inland Realm",
		"River Realm",
		"Maritime Realm",
		"Highland Realm",
		"Arid Realm",
	}
	if int(style) < len(names) {
		return names[style]
	}
	return "Unknown"
}

type ProtoCivilization struct {
	ID             int
	RegionID       int
	CenterNode     int
	AnchorCount    int
	CenterKind     SettlementNodeKind
	Style          ProtoCivilizationStyle
	Coastal        bool
	River          bool
	TerritoryCells int
	CoreCells      int
	MeanSupport    float64
}

type ProtoCivilizationDiagnostics struct {
	CivilizationByCell []int
	ClaimCost          []float64
}

type ProtoCivilizationResult struct {
	Civilizations  []ProtoCivilization
	OutpostRegions int
	Diagnostics    *ProtoCivilizationDiagnostics
}

type ProtoCivilizationSettings struct {
	MinRegionAnchors     int
	MinCenterScore       float64
	MinCenterKind        SettlementNodeKind
	MinTerritoryCells    int
	ClaimBaseTravel      float64
	ClaimPerAnchorTravel float64
	ClaimMaxTravel       float64
	CoreCarryThreshold   float64
}

func DefaultProtoCivilizationSettings() ProtoCivilizationSettings {
	return ProtoCivilizationSettings{
		MinRegionAnchors:     3,
		MinCenterScore:       0.46,
		MinCenterKind:        SettlementNodeTown,
		MinTerritoryCells:    12,
		ClaimBaseTravel:      9.5,
		ClaimPerAnchorTravel: 1.4,
		ClaimMaxTravel:       24.0,
		CoreCarryThreshold:   0.46,
	}
}

func BuildProtoCivilizations(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	settlements *SettlementResult,
	population *PopulationResult,
	biomes *BiomeResult,
	soils *SoilResult,
	elevation []float64,
	seaLevel float64,
	settings ProtoCivilizationSettings,
) *ProtoCivilizationResult {
	out := &ProtoCivilizationResult{
		Diagnostics: &ProtoCivilizationDiagnostics{
			CivilizationByCell: make([]int, len(elevation)),
			ClaimCost:          make([]float64, len(elevation)),
		},
	}
	for i := range out.Diagnostics.CivilizationByCell {
		out.Diagnostics.CivilizationByCell[i] = -1
		out.Diagnostics.ClaimCost[i] = math.Inf(1)
	}
	if network == nil || network.Diagnostics == nil || len(network.Nodes) == 0 || len(network.Regions) == 0 {
		return out
	}
	if settlements == nil || settlements.Diagnostics == nil || population == nil || population.Diagnostics == nil {
		return out
	}

	civs := make([]ProtoCivilization, 0, len(network.Regions))
	for _, region := range network.Regions {
		if !eligibleProtoCivilizationRegion(region, network, settings) {
			out.OutpostRegions++
			continue
		}
		center := network.Nodes[region.CenterNode]
		civs = append(civs, ProtoCivilization{
			ID:          len(civs),
			RegionID:    region.ID,
			CenterNode:  region.CenterNode,
			AnchorCount: len(region.NodeIndices),
			CenterKind:  center.Kind,
			Style:       classifyProtoCivilizationStyle(center.CellIndex, center, biomes, soils),
			Coastal:     region.Coastal,
			River:       region.River,
		})
	}
	if len(civs) == 0 {
		return out
	}

	assignProtoCivilizationTerritories(cells, network, settlements, population, biomes, elevation, seaLevel, civs, out.Diagnostics, settings)
	civs = finalizeProtoCivilizations(civs, out.Diagnostics, population, settings)
	out.OutpostRegions += len(network.Regions) - len(civs) - out.OutpostRegions
	out.Civilizations = civs
	return out
}

func eligibleProtoCivilizationRegion(region SettlementRegion, network *SettlementNetworkResult, settings ProtoCivilizationSettings) bool {
	center := network.Nodes[region.CenterNode]
	if len(region.NodeIndices) >= settings.MinRegionAnchors {
		return true
	}
	if center.Kind >= settings.MinCenterKind && center.Score >= settings.MinCenterScore {
		return true
	}
	if center.Kind >= SettlementNodeTown && (center.Coastal || center.River) && center.Score >= settings.MinCenterScore-0.04 {
		return true
	}
	return false
}

func classifyProtoCivilizationStyle(idx int, center SettlementNode, biomes *BiomeResult, soils *SoilResult) ProtoCivilizationStyle {
	aridity := 0.0
	wetland := 0.0
	relief := 0.0
	rockiness := 0.0
	if biomes != nil && biomes.Diagnostics != nil {
		if idx < len(biomes.Diagnostics.AridityRatio) {
			aridity = biomes.Diagnostics.AridityRatio[idx]
		}
		if idx < len(biomes.Diagnostics.WetlandAffinity) {
			wetland = biomes.Diagnostics.WetlandAffinity[idx]
		}
	}
	if soils != nil && soils.Diagnostics != nil {
		if idx < len(soils.Diagnostics.Relief) {
			relief = soils.Diagnostics.Relief[idx]
		}
		if idx < len(soils.Diagnostics.Rockiness) {
			rockiness = soils.Diagnostics.Rockiness[idx]
		}
	}
	switch {
	case center.Coastal:
		return ProtoCivilizationMaritime
	case center.River || wetland >= 0.45:
		return ProtoCivilizationRiverine
	case relief >= 700 || rockiness >= 0.58:
		return ProtoCivilizationHighland
	case aridity >= 0.70:
		return ProtoCivilizationArid
	default:
		return ProtoCivilizationInland
	}
}

func assignProtoCivilizationTerritories(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	settlements *SettlementResult,
	population *PopulationResult,
	biomes *BiomeResult,
	elevation []float64,
	seaLevel float64,
	civs []ProtoCivilization,
	diagnostics *ProtoCivilizationDiagnostics,
	settings ProtoCivilizationSettings,
) {
	bestCost := diagnostics.ClaimCost
	for civIdx, civ := range civs {
		center := network.Nodes[civ.CenterNode]
		maxTravel := protoCivilizationClaimLimit(civ, center, settings)
		dist, _ := shortestPathsFromNode(center.CellIndex, cells, network.Diagnostics.MovementCost, maxTravel)
		influence := 1.0 + 0.22*float64(center.Kind) + 0.28*center.Score + 0.06*math.Log1p(float64(civ.AnchorCount))
		for cellIdx := range elevation {
			if !claimableProtoCivilizationCell(cellIdx, settlements, population, biomes, elevation, seaLevel) {
				continue
			}
			if math.IsInf(dist[cellIdx], 1) {
				continue
			}
			adjusted := dist[cellIdx] / influence
			if adjusted < bestCost[cellIdx] {
				bestCost[cellIdx] = adjusted
				diagnostics.CivilizationByCell[cellIdx] = civIdx
			}
		}
	}
}

func claimableProtoCivilizationCell(
	idx int,
	settlements *SettlementResult,
	population *PopulationResult,
	biomes *BiomeResult,
	elevation []float64,
	seaLevel float64,
) bool {
	if idx < 0 || idx >= len(elevation) || elevation[idx] < seaLevel {
		return false
	}
	if population.Classes[idx] == PopulationUninhabited {
		return false
	}
	if population.Diagnostics.CarryingCapacity[idx] < 0.16 && settlements.Classes[idx] < SettlementMarginal {
		return false
	}
	if biomes != nil && biomes.Diagnostics != nil && idx < len(biomes.Diagnostics.AnnualIceFraction) && biomes.Diagnostics.AnnualIceFraction[idx] >= 0.92 {
		return false
	}
	return true
}

func protoCivilizationClaimLimit(civ ProtoCivilization, center SettlementNode, settings ProtoCivilizationSettings) float64 {
	limit := settings.ClaimBaseTravel +
		settings.ClaimPerAnchorTravel*math.Sqrt(float64(civ.AnchorCount)) +
		2.0*float64(center.Kind)
	if civ.Coastal {
		limit += 1.5
	}
	if civ.River {
		limit += 1.0
	}
	return math.Min(limit, settings.ClaimMaxTravel)
}

func finalizeProtoCivilizations(
	civs []ProtoCivilization,
	diagnostics *ProtoCivilizationDiagnostics,
	population *PopulationResult,
	settings ProtoCivilizationSettings,
) []ProtoCivilization {
	territory := make([]int, len(civs))
	core := make([]int, len(civs))
	support := make([]float64, len(civs))
	for cellIdx, civIdx := range diagnostics.CivilizationByCell {
		if civIdx < 0 || civIdx >= len(civs) {
			continue
		}
		territory[civIdx]++
		support[civIdx] += population.Diagnostics.CarryingCapacity[cellIdx]
		if population.Diagnostics.CarryingCapacity[cellIdx] >= settings.CoreCarryThreshold {
			core[civIdx]++
		}
	}

	filtered := make([]ProtoCivilization, 0, len(civs))
	oldToNew := make([]int, len(civs))
	for i := range oldToNew {
		oldToNew[i] = -1
	}
	for i, civ := range civs {
		if territory[i] < settings.MinTerritoryCells {
			continue
		}
		civ.ID = len(filtered)
		civ.TerritoryCells = territory[i]
		civ.CoreCells = core[i]
		if territory[i] > 0 {
			civ.MeanSupport = support[i] / float64(territory[i])
		}
		oldToNew[i] = civ.ID
		filtered = append(filtered, civ)
	}
	for i, civIdx := range diagnostics.CivilizationByCell {
		if civIdx >= 0 {
			diagnostics.CivilizationByCell[i] = oldToNew[civIdx]
		}
	}
	return filtered
}
