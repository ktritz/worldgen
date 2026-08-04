package climgen

import (
	"math"
	"sort"
)

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

// ProtoCivilizationEligibilityTally aggregates the regions that resolved to one
// eligibility reason, together with the mean values of the quantities the gate
// compares. The means make it possible to tell a region that missed a threshold
// by a hair from one that is structurally excluded.
type ProtoCivilizationEligibilityTally struct {
	Reason               string
	Eligible             bool
	Regions              int
	MeanAnchorStrength   float64
	MeanPhysicalStrength float64
	MeanAreaSupport      float64
	MeanMinStrength      float64
	MeanCenterKind       float64
}

type ProtoCivilizationDiagnostics struct {
	CivilizationByCell  []int
	ClaimCost           []float64
	EligibilityRegions  int
	EligibilityByReason map[string]ProtoCivilizationEligibilityTally
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
			CivilizationByCell:  make([]int, len(elevation)),
			ClaimCost:           make([]float64, len(elevation)),
			EligibilityByReason: make(map[string]ProtoCivilizationEligibilityTally),
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

	// Re-derive physical support with the same kind thresholds the network was
	// built with, otherwise a quantile-calibrated mesh would be gated against
	// the absolute defaults.
	networkSettings := SettlementNetworkSettingsWithKindThresholds(DefaultSettlementNetworkSettings(), network)
	tallies := newProtoCivilizationEligibilityTallies()
	civs := make([]ProtoCivilization, 0, len(network.Regions))
	for _, region := range network.Regions {
		detail := protoCivilizationRegionEligibilityDetail(region, network, cells, population, settings, networkSettings)
		tallies.add(region, network, cells, population, networkSettings, detail)
		if !detail.Eligible {
			out.OutpostRegions++
			continue
		}
		center := network.Nodes[region.CenterNode]
		anchorStrength := protoRegionClaimAnchorStrength(region, network, cells, population, networkSettings)
		civs = append(civs, ProtoCivilization{
			ID:          len(civs),
			RegionID:    region.ID,
			CenterNode:  region.CenterNode,
			AnchorCount: int(math.Ceil(anchorStrength)),
			CenterKind:  center.Kind,
			Style:       classifyProtoCivilizationStyle(center.CellIndex, center, biomes, soils),
			Coastal:     region.Coastal,
			River:       region.River,
		})
	}
	tallies.finalize(out.Diagnostics)
	if len(civs) == 0 {
		return out
	}

	assignProtoCivilizationTerritories(cells, network, settlements, population, biomes, elevation, seaLevel, civs, out.Diagnostics, settings)
	civs = finalizeProtoCivilizations(civs, out.Diagnostics, population, settings)
	out.OutpostRegions += len(network.Regions) - len(civs) - out.OutpostRegions
	out.Civilizations = civs
	return out
}

type protoCivilizationEligibilityAccumulator struct {
	eligible         bool
	regions          int
	anchorStrength   float64
	physicalStrength float64
	areaSupport      float64
	areaSamples      int
	minStrength      float64
	centerKind       float64
	centerSamples    int
}

type protoCivilizationEligibilityTallies struct {
	total  int
	byName map[string]*protoCivilizationEligibilityAccumulator
}

func newProtoCivilizationEligibilityTallies() *protoCivilizationEligibilityTallies {
	return &protoCivilizationEligibilityTallies{byName: make(map[string]*protoCivilizationEligibilityAccumulator)}
}

// add folds one region's verdict into the per-reason tally. The area-support
// strength is only computed by the gate on one branch, so it is filled in here
// for the remaining regions: the walk is one hop disc per region node and the
// region count is small, and without it the rejected reasons carry no
// area-support signal at all.
func (t *protoCivilizationEligibilityTallies) add(
	region SettlementRegion,
	network *SettlementNetworkResult,
	cells []VoronoiCell,
	population *PopulationResult,
	networkSettings SettlementNetworkSettings,
	detail protoCivilizationEligibilityDetail,
) {
	if t == nil {
		return
	}
	t.total++
	acc := t.byName[detail.Reason]
	if acc == nil {
		acc = &protoCivilizationEligibilityAccumulator{eligible: detail.Eligible}
		t.byName[detail.Reason] = acc
	}
	acc.regions++
	acc.anchorStrength += detail.AnchorStrength
	acc.physicalStrength += detail.PhysicalStrength
	acc.minStrength += detail.MinStrength
	if detail.HasCenter {
		acc.centerKind += float64(detail.CenterKind)
		acc.centerSamples++
	}
	areaSupport := detail.AreaSupport
	if !detail.AreaSupportComputed {
		if !detail.HasCenter || len(cells) == 0 || population == nil {
			return
		}
		areaSupport = ProtoCivilizationRegionPopulationSupportStrength(region, network, cells, population, networkSettings)
	}
	acc.areaSupport += areaSupport
	acc.areaSamples++
}

func (t *protoCivilizationEligibilityTallies) finalize(diagnostics *ProtoCivilizationDiagnostics) {
	if t == nil || diagnostics == nil {
		return
	}
	if diagnostics.EligibilityByReason == nil {
		diagnostics.EligibilityByReason = make(map[string]ProtoCivilizationEligibilityTally, len(t.byName))
	}
	diagnostics.EligibilityRegions = t.total
	for reason, acc := range t.byName {
		tally := ProtoCivilizationEligibilityTally{
			Reason:   reason,
			Eligible: acc.eligible,
			Regions:  acc.regions,
		}
		if acc.regions > 0 {
			tally.MeanAnchorStrength = acc.anchorStrength / float64(acc.regions)
			tally.MeanPhysicalStrength = acc.physicalStrength / float64(acc.regions)
			tally.MeanMinStrength = acc.minStrength / float64(acc.regions)
		}
		if acc.areaSamples > 0 {
			tally.MeanAreaSupport = acc.areaSupport / float64(acc.areaSamples)
		}
		if acc.centerSamples > 0 {
			tally.MeanCenterKind = acc.centerKind / float64(acc.centerSamples)
		}
		diagnostics.EligibilityByReason[reason] = tally
	}
}

// ProtoCivilizationEligibilityTallyOrder returns the reason keys of the tally
// map in a stable order: accepted reasons first, then rejected, each by
// descending region count and then by name.
func ProtoCivilizationEligibilityTallyOrder(tallies map[string]ProtoCivilizationEligibilityTally) []string {
	if len(tallies) == 0 {
		return nil
	}
	reasons := make([]string, 0, len(tallies))
	for reason := range tallies {
		reasons = append(reasons, reason)
	}
	sort.Slice(reasons, func(i, j int) bool {
		a := tallies[reasons[i]]
		b := tallies[reasons[j]]
		if a.Eligible != b.Eligible {
			return a.Eligible
		}
		if a.Regions != b.Regions {
			return a.Regions > b.Regions
		}
		return reasons[i] < reasons[j]
	})
	return reasons
}

func eligibleProtoCivilizationRegion(region SettlementRegion, network *SettlementNetworkResult, settings ProtoCivilizationSettings) bool {
	eligible, _ := ProtoCivilizationRegionEligibilityReason(region, network, settings)
	return eligible
}

func EligibleProtoCivilizationRegionWithPhysicalSupport(
	region SettlementRegion,
	network *SettlementNetworkResult,
	cells []VoronoiCell,
	population *PopulationResult,
	settings ProtoCivilizationSettings,
	networkSettings SettlementNetworkSettings,
) (bool, string) {
	return protoCivilizationRegionEligibilityReason(region, network, cells, population, settings, networkSettings)
}

func ProtoCivilizationRegionEligibilityReason(region SettlementRegion, network *SettlementNetworkResult, settings ProtoCivilizationSettings) (bool, string) {
	networkSettings := SettlementNetworkSettingsWithKindThresholds(DefaultSettlementNetworkSettings(), network)
	return protoCivilizationRegionEligibilityReason(region, network, nil, nil, settings, networkSettings)
}

// protoCivilizationEligibilityDetail carries the eligibility verdict together
// with the quantities the gates compared, so callers can tally how far from
// passing a rejected region was.
type protoCivilizationEligibilityDetail struct {
	Eligible            bool
	Reason              string
	AnchorStrength      float64
	PhysicalStrength    float64
	AreaSupport         float64
	AreaSupportComputed bool
	MinStrength         float64
	CenterKind          SettlementNodeKind
	HasCenter           bool
}

func protoCivilizationRegionEligibilityReason(
	region SettlementRegion,
	network *SettlementNetworkResult,
	cells []VoronoiCell,
	population *PopulationResult,
	settings ProtoCivilizationSettings,
	networkSettings SettlementNetworkSettings,
) (bool, string) {
	detail := protoCivilizationRegionEligibilityDetail(region, network, cells, population, settings, networkSettings)
	return detail.Eligible, detail.Reason
}

func protoCivilizationRegionEligibilityDetail(
	region SettlementRegion,
	network *SettlementNetworkResult,
	cells []VoronoiCell,
	population *PopulationResult,
	settings ProtoCivilizationSettings,
	networkSettings SettlementNetworkSettings,
) protoCivilizationEligibilityDetail {
	detail := protoCivilizationEligibilityDetail{}
	if network == nil || region.CenterNode < 0 || region.CenterNode >= len(network.Nodes) {
		detail.Reason = "missing-center"
		return detail
	}
	center := network.Nodes[region.CenterNode]
	detail.HasCenter = true
	detail.CenterKind = center.Kind
	anchorStrength := ProtoCivilizationRegionAnchorStrength(region, network)
	physicalStrength := 0.0
	hasPhysicalInputs := len(cells) > 0 && population != nil
	if hasPhysicalInputs {
		physicalStrength = ProtoCivilizationRegionPhysicalAnchorStrength(region, network, cells, population, networkSettings)
	}
	minStrength := math.Max(1, float64(settings.MinRegionAnchors)-0.5)
	detail.AnchorStrength = anchorStrength
	detail.PhysicalStrength = physicalStrength
	detail.MinStrength = minStrength
	if anchorStrength >= minStrength && center.Kind >= settings.MinCenterKind {
		if hasPhysicalInputs {
			hasSupportedRegional := regionHasPhysicallySupportedRegionalAnchor(region, network, cells, population, networkSettings)
			if !hasSupportedRegional {
				detail.Reason = "regional-support"
				return detail
			}
			if physicalStrength < minStrength && !regionHasCompactHighRankSupport(region, network, physicalStrength, minStrength) {
				detail.Reason = "regional-support"
				return detail
			}
		}
		detail.Eligible = true
		detail.Reason = "anchor-kind"
		return detail
	}
	if hasPhysicalInputs && anchorStrength < minStrength && center.Kind >= settings.MinCenterKind {
		hasSupportedRegional := regionHasPhysicallySupportedRegionalAnchor(region, network, cells, population, networkSettings)
		areaSupportStrength := ProtoCivilizationRegionPopulationSupportStrength(region, network, cells, population, networkSettings)
		detail.AreaSupport = areaSupportStrength
		detail.AreaSupportComputed = true
		if hasSupportedRegional &&
			anchorStrength >= minStrength*0.55 &&
			physicalStrength >= minStrength*0.55 &&
			areaSupportStrength >= float64(settings.MinRegionAnchors)*3.0 {
			detail.Eligible = true
			detail.Reason = "area-support"
			return detail
		}
	}
	if len(region.NodeIndices) >= settings.MinRegionAnchors*2 && center.Kind >= SettlementNodeVillage && region.MeanScore >= settings.MinCenterScore {
		if hasPhysicalInputs && physicalStrength < float64(settings.MinRegionAnchors) {
			detail.Reason = "broad-strength"
			return detail
		}
		if center.Kind < settings.MinCenterKind && anchorStrength < float64(settings.MinRegionAnchors) {
			detail.Reason = "broad-strength"
			return detail
		}
		detail.Eligible = true
		detail.Reason = "broad-cluster"
		return detail
	}
	if center.Kind >= SettlementNodeCity && center.Score >= settings.MinCenterScore {
		detail.Eligible = true
		detail.Reason = "city-score"
		return detail
	}
	if anchorStrength < minStrength {
		detail.Reason = "anchor-shortfall"
		return detail
	}
	if center.Kind < settings.MinCenterKind {
		detail.Reason = "center-kind"
		return detail
	}
	detail.Reason = "score"
	return detail
}

func eligibleProtoCivilizationRegionWithPhysicalSupport(
	region SettlementRegion,
	network *SettlementNetworkResult,
	cells []VoronoiCell,
	population *PopulationResult,
	settings ProtoCivilizationSettings,
	networkSettings SettlementNetworkSettings,
) bool {
	eligible, _ := protoCivilizationRegionEligibilityReason(region, network, cells, population, settings, networkSettings)
	return eligible
}

func regionHasCompactHighRankSupport(region SettlementRegion, network *SettlementNetworkResult, physicalStrength, minStrength float64) bool {
	if network == nil || len(region.NodeIndices) == 0 {
		return false
	}
	anchorDensity := ProtoCivilizationRegionAnchorStrength(region, network) / float64(len(region.NodeIndices))
	return physicalStrength >= minStrength*0.65 && anchorDensity >= 0.55
}

func regionHasPhysicallySupportedRegionalAnchor(
	region SettlementRegion,
	network *SettlementNetworkResult,
	cells []VoronoiCell,
	population *PopulationResult,
	settings SettlementNetworkSettings,
) bool {
	for _, nodeIdx := range region.NodeIndices {
		if nodeIdx < 0 || nodeIdx >= len(network.Nodes) {
			continue
		}
		node := network.Nodes[nodeIdx]
		if node.Kind < SettlementNodeTown {
			continue
		}
		if SettlementNodePhysicalSupportArea(node.CellIndex, node.Kind, cells, population, settings) >= 0.5 {
			return true
		}
	}
	return false
}

func protoRegionAnchorStrength(region SettlementRegion, network *SettlementNetworkResult) float64 {
	return ProtoCivilizationRegionAnchorStrength(region, network)
}

func protoRegionClaimAnchorStrength(
	region SettlementRegion,
	network *SettlementNetworkResult,
	cells []VoronoiCell,
	population *PopulationResult,
	settings SettlementNetworkSettings,
) float64 {
	physicalStrength := ProtoCivilizationRegionPhysicalAnchorStrength(region, network, cells, population, settings)
	if physicalStrength > 0 {
		return math.Max(1, physicalStrength)
	}
	return math.Max(1, protoRegionAnchorStrength(region, network))
}

func ProtoCivilizationRegionAnchorStrength(region SettlementRegion, network *SettlementNetworkResult) float64 {
	if network == nil {
		return 0
	}
	strength := 0.0
	for _, nodeIdx := range region.NodeIndices {
		if nodeIdx < 0 || nodeIdx >= len(network.Nodes) {
			continue
		}
		switch network.Nodes[nodeIdx].Kind {
		case SettlementNodeCity, SettlementNodeTown:
			strength += 1
		case SettlementNodeVillage:
			strength += 0.5
		}
	}
	return strength
}

func ProtoCivilizationRegionPhysicalAnchorStrength(
	region SettlementRegion,
	network *SettlementNetworkResult,
	cells []VoronoiCell,
	population *PopulationResult,
	settings SettlementNetworkSettings,
) float64 {
	if network == nil || len(cells) == 0 || population == nil {
		return 0
	}
	strength := 0.0
	for _, nodeIdx := range region.NodeIndices {
		if nodeIdx < 0 || nodeIdx >= len(network.Nodes) {
			continue
		}
		node := network.Nodes[nodeIdx]
		base := 0.0
		switch node.Kind {
		case SettlementNodeCity, SettlementNodeTown:
			base = 1
		case SettlementNodeVillage:
			base = 0.5
		}
		if base <= 0 {
			continue
		}
		area := SettlementNodePhysicalSupportArea(node.CellIndex, node.Kind, cells, population, settings)
		strength += base * math.Min(area, 1.0)
	}
	return strength
}

func ProtoCivilizationRegionPopulationSupportStrength(
	region SettlementRegion,
	network *SettlementNetworkResult,
	cells []VoronoiCell,
	population *PopulationResult,
	settings SettlementNetworkSettings,
) float64 {
	if network == nil || len(cells) == 0 || population == nil || population.Diagnostics == nil {
		return 0
	}
	seen := make(map[int]struct{})
	radius := meshResolutionAdjustedSteps(2, len(cells))
	townCells := 0
	villageCells := 0
	discCells := 0
	discSamples := 0
	for _, nodeIdx := range region.NodeIndices {
		if nodeIdx < 0 || nodeIdx >= len(network.Nodes) {
			continue
		}
		node := network.Nodes[nodeIdx]
		if node.CellIndex < 0 || node.CellIndex >= len(cells) {
			continue
		}
		disc := cellsWithinHops(cells, node.CellIndex, radius)
		discCells += len(disc)
		discSamples++
		for _, cellIdx := range disc {
			if cellIdx < 0 || cellIdx >= len(population.Diagnostics.CarryingCapacity) || cellIdx >= len(population.Diagnostics.UrbanPotential) {
				continue
			}
			if _, ok := seen[cellIdx]; ok {
				continue
			}
			seen[cellIdx] = struct{}{}
			if physicallySupportsSettlementNodeKind(cellIdx, SettlementNodeTown, population, settings) {
				townCells++
			} else if physicallySupportsSettlementNodeKind(cellIdx, SettlementNodeVillage, population, settings) {
				villageCells++
			}
		}
	}
	cellArea := regionSupportBaselineCellArea(discCells, discSamples)
	townArea := float64(townCells) * cellArea
	villageArea := float64(villageCells) * cellArea
	return townArea + 0.5*villageArea
}

// protoRegionSupportDiscBaselineCells is the baseline (L5) size of the two-hop
// ring set walked above — the six one-hop plus twelve two-hop neighbours of a
// hexagonal cell. cellsWithinHops excludes the centre, so the centre is excluded
// here too.
const protoRegionSupportDiscBaselineCells = 18.0

// regionSupportBaselineCellArea returns the baseline-equivalent area of one mesh
// cell, measured from the two-hop discs actually walked rather than from scale²
// alone. A hop disc holds ~3r²+3r cells, so its baseline-equivalent area drifts
// (18.0 at L5, 15.0 at L6, 13.5 at L7 — a ~25% shrink) when the raw count is
// converted with meshScaledTerritoryAreaCells, while the consumer at
// protoCivilizationRegionEligibilityReason compares against a hard constant.
// Normalizing by the measured disc size removes that drift and is exactly the
// old conversion at L5 for hexagonal cells (18/18 = 1.0 baseline cells each).
func regionSupportBaselineCellArea(discCells, discSamples int) float64 {
	if discSamples <= 0 || discCells <= 0 {
		return 0
	}
	// Capped at one baseline cell per supporting cell, for the same reason as
	// settlementSupportDiscArea: pentagon-centred windows are smaller, so the
	// uncapped ratio would over-credit them. Keeps L5 bit-identical; inert at
	// finer meshes (0.47 at L6, 0.30 at L7).
	perCell := protoRegionSupportDiscBaselineCells * float64(discSamples) / float64(discCells)
	if perCell > 1 {
		perCell = 1
	}
	return perCell
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
		influence := 1.0 + 0.22*settlementNodeEffectiveRank(center) + 0.28*center.Score*SettlementNodePhysicalSupportWeight(center) + 0.06*math.Log1p(float64(civ.AnchorCount))
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
		2.0*settlementNodeEffectiveRank(center)
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
	cellCount := len(diagnostics.CivilizationByCell)
	for i, civ := range civs {
		if meshScaledTerritoryAreaCells(territory[i], cellCount) < float64(settings.MinTerritoryCells) {
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
