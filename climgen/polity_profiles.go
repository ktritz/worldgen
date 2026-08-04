package climgen

import (
	"math"
	"sort"
)

type PolityAttitudeStance int

const (
	PolityAttitudeHostile PolityAttitudeStance = iota
	PolityAttitudeWary
	PolityAttitudeNeutral
	PolityAttitudeFriendly
	PolityAttitudeAllied
)

func PolityAttitudeStanceName(stance PolityAttitudeStance) string {
	switch stance {
	case PolityAttitudeHostile:
		return "Hostile"
	case PolityAttitudeWary:
		return "Wary"
	case PolityAttitudeNeutral:
		return "Neutral"
	case PolityAttitudeFriendly:
		return "Friendly"
	case PolityAttitudeAllied:
		return "Allied"
	default:
		return "Unknown"
	}
}

type PolityProfileAssignment struct {
	PolityID        int
	Profile         ResolvedProfile
	Score           float64
	ContextTags     []string
	EnvironmentTags []string
	ContextTraits   map[string]float64
}

type PolityAttitude struct {
	From             int
	To               int
	Score            float64
	Stance           PolityAttitudeStance
	Affinity         float64
	Cultural         float64
	StrategicTension float64
	TradeBonus       float64
	AllianceBonus    float64
	BorderPenalty    float64
	Competition      float64
}

type PolityProfileResult struct {
	Assignments []PolityProfileAssignment
	Attitudes   []PolityAttitude
}

func ExtractResolvedProfiles(catalog *ProfileCatalog) []ResolvedProfile {
	if catalog == nil {
		return nil
	}
	out := make([]ResolvedProfile, 0, len(catalog.Compositions)+len(catalog.Ancestries))
	seen := make(map[string]struct{}, len(catalog.Compositions)+len(catalog.Ancestries))
	if len(catalog.Compositions) > 0 {
		for _, spec := range catalog.Compositions {
			resolved := ResolveProfileComposition(catalog, spec)
			if resolved != nil {
				if _, ok := seen[resolved.Name]; ok {
					continue
				}
				seen[resolved.Name] = struct{}{}
				out = append(out, *resolved)
			}
		}
	}
	for _, ancestry := range catalog.Ancestries {
		resolved := ComposeResolvedProfile(ancestry, nil)
		if resolved != nil {
			if _, ok := seen[resolved.Name]; ok {
				continue
			}
			seen[resolved.Name] = struct{}{}
			out = append(out, *resolved)
		}
	}
	return out
}

func BuildPolityProfiles(
	cells []VoronoiCell,
	polities *PolitySphereResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	biomes *BiomeResult,
	vegetation *VegetationResult,
	soils *SoilResult,
	hydro *HydrologyBiomeInputs,
	catalog *ProfileCatalog,
) *PolityProfileResult {
	out := &PolityProfileResult{}
	if polities == nil || len(polities.Spheres) == 0 || network == nil || trade == nil || catalog == nil {
		return out
	}
	if len(catalog.Ancestries) == 0 {
		return out
	}

	assignments := make([]PolityProfileAssignment, 0, len(polities.Spheres))
	for _, sphere := range polities.Spheres {
		resolved, context, env, bestScore := selectPolityResolvedProfile(catalog, sphere, polities, network, trade, biomes, vegetation, soils, hydro)
		assignments = append(assignments, PolityProfileAssignment{
			PolityID:        sphere.ID,
			Profile:         resolved,
			Score:           bestScore,
			ContextTags:     append([]string(nil), context.Tags...),
			EnvironmentTags: append([]string(nil), env.Tags...),
			ContextTraits:   cloneTraitMap(context.Traits),
		})
	}

	out.Assignments = assignments
	out.Attitudes = buildPolityAttitudes(cells, polities, network, trade, assignments)
	return out
}

func buildPolityAttitudes(
	cells []VoronoiCell,
	polities *PolitySphereResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	assignments []PolityProfileAssignment,
) []PolityAttitude {
	if len(assignments) == 0 {
		return nil
	}
	borderPenalty := polityBorderPressure(cells, polities)
	tradeBonus := polityTradeBonus(polities, network, trade)
	attitudes := make([]PolityAttitude, 0, len(assignments)*(len(assignments)-1))
	assignmentByID := make(map[int]PolityProfileAssignment, len(assignments))
	for _, assignment := range assignments {
		assignmentByID[assignment.PolityID] = assignment
	}
	for _, from := range polities.Spheres {
		fromAssignment := assignmentByID[from.ID]
		for _, to := range polities.Spheres {
			if from.ID == to.ID {
				continue
			}
			toAssignment := assignmentByID[to.ID]
			target := BuildResolvedProfileAffinityContext(&toAssignment.Profile)
			target.Tags = mergeProfileTags(target.Tags, toAssignment.ContextTags)
			target.Traits = mergeTraitMaps(target.Traits, toAssignment.ContextTraits)
			affinity := scaledAffinityScore(ScoreProfileAffinity(&fromAssignment.Profile, target))
			tradeAdj := tradeBonus[[2]int{from.ID, to.ID}]
			borderAdj := borderPenalty[[2]int{from.ID, to.ID}]
			competitionAdj := polityCompetitionPenalty(fromAssignment, toAssignment, borderAdj, tradeAdj)
			competitionAdj += polityTerritorialRivalryPenalty(from, to, fromAssignment, toAssignment, borderAdj, tradeAdj, polities.Relations, len(cells))
			allianceAdj := polityAllianceBonus(from.ID, to.ID, from, to, fromAssignment, toAssignment, affinity, tradeAdj, borderAdj, competitionAdj, polities.Relations)
			strategicTension := borderAdj + competitionAdj
			score := affinity + tradeAdj + allianceAdj - strategicTension
			stance := classifyDetailedPolityAttitude(score, allianceAdj, hasSuzeraintyRelation(from.ID, to.ID, polities.Relations))
			attitudes = append(attitudes, PolityAttitude{
				From:             from.ID,
				To:               to.ID,
				Score:            score,
				Stance:           stance,
				Affinity:         affinity,
				Cultural:         affinity,
				StrategicTension: strategicTension,
				TradeBonus:       tradeAdj,
				AllianceBonus:    allianceAdj,
				BorderPenalty:    borderAdj,
				Competition:      competitionAdj,
			})
		}
	}
	sort.Slice(attitudes, func(i, j int) bool {
		if attitudes[i].From != attitudes[j].From {
			return attitudes[i].From < attitudes[j].From
		}
		return attitudes[i].Score > attitudes[j].Score
	})
	return attitudes
}

func polityBorderPressure(cells []VoronoiCell, polities *PolitySphereResult) map[[2]int]float64 {
	out := make(map[[2]int]float64)
	if polities == nil || polities.Diagnostics == nil {
		return out
	}
	borderCounts := make(map[[2]int]int)
	for i, owner := range polities.Diagnostics.PolityByCell {
		if owner < 0 || i >= len(cells) {
			continue
		}
		for _, neighbor := range cells[i].NeighborSiteIndices {
			j := int(neighbor)
			if j < 0 || j >= len(polities.Diagnostics.PolityByCell) {
				continue
			}
			other := polities.Diagnostics.PolityByCell[j]
			if other < 0 || other == owner {
				continue
			}
			key := [2]int{owner, other}
			borderCounts[key]++
		}
	}
	for key, count := range borderCounts {
		out[key] = borderPressureFromCount(count, len(cells))
	}
	return out
}

// borderPressureFromCount converts a count of border cell-adjacency pairs (a linear
// measure that grows with sqrt(cellCount) for a fixed physical border) into a
// baseline-equivalent pressure so the saturation cap is resolution-independent.
func borderPressureFromCount(count int, meshCellCount int) float64 {
	scaled := float64(count) * meshPathCostResolutionScale(meshCellCount)
	return 0.03 * math.Min(scaled, 12)
}

func polityTradeBonus(polities *PolitySphereResult, network *SettlementNetworkResult, trade *TradeNetworkResult) map[[2]int]float64 {
	out := make(map[[2]int]float64)
	if polities == nil || trade == nil {
		return out
	}
	polityByCapital := make(map[int]int, len(polities.Spheres))
	for _, sphere := range polities.Spheres {
		polityByCapital[sphere.CapitalNode] = sphere.ID
	}
	for _, corridor := range trade.Corridors {
		from, okA := polityByCapital[corridor.FromNode]
		to, okB := polityByCapital[corridor.ToNode]
		if !okA || !okB || from == to {
			continue
		}
		out[[2]int{from, to}] += 0.16 * corridor.Flow
		out[[2]int{to, from}] += 0.16 * corridor.Flow
	}
	return out
}

func scaledAffinityScore(raw float64) float64 {
	return 0.60 * math.Tanh(0.75*raw)
}

func polityCompetitionPenalty(
	from PolityProfileAssignment,
	to PolityProfileAssignment,
	borderAdj float64,
	tradeAdj float64,
) float64 {
	if borderAdj <= 0 {
		return 0
	}
	borderFactor := clamp01(borderAdj / 0.40)
	contested := overlapScore(
		from.EnvironmentTags,
		to.EnvironmentTags,
		[]string{"floodplain", "alluvial", "marsh", "delta", "forest", "mountain", "arid", "coastal", "lacustrine"},
	)
	economicRivalry := overlapScore(
		from.ContextTags,
		to.ContextTags,
		[]string{"mercantile", "agrarian", "urban"},
	)
	sameness := 0.0
	if from.Profile.AncestryName != "" && from.Profile.AncestryName == to.Profile.AncestryName {
		sameness += 0.08
	}
	if from.Profile.CultureName != "" && from.Profile.CultureName == to.Profile.CultureName {
		sameness += 0.06
	}
	penalty := borderFactor * (0.16*contested + 0.10*economicRivalry + 0.60*sameness)
	if tradeAdj > 0 {
		penalty *= 1 - 0.50*clamp01(tradeAdj/0.32)
	}
	return penalty
}

func polityTerritorialRivalryPenalty(
	from PolitySphere,
	to PolitySphere,
	fromAssignment PolityProfileAssignment,
	toAssignment PolityProfileAssignment,
	borderAdj float64,
	tradeAdj float64,
	relations []PolitySphereRelation,
	meshCellCount int,
) float64 {
	if borderAdj <= 0 {
		return 0
	}
	borderFactor := clamp01(borderAdj / 0.30)
	nicheOverlap := overlapScore(
		fromAssignment.EnvironmentTags,
		toAssignment.EnvironmentTags,
		[]string{"river", "lowland", "floodplain", "alluvial", "marsh", "delta", "forest", "mountain", "arid", "coastal", "lacustrine"},
	)
	economicOverlap := overlapScore(
		fromAssignment.ContextTags,
		toAssignment.ContextTags,
		[]string{"mercantile", "agrarian", "urban", "pastoral", "frontier", "clan", "fortress"},
	)

	kinRivalry := 0.0
	if fromAssignment.Profile.AncestryName != "" && fromAssignment.Profile.AncestryName == toAssignment.Profile.AncestryName {
		kinRivalry += 0.08
	}
	if fromAssignment.Profile.CultureName != "" && fromAssignment.Profile.CultureName == toAssignment.Profile.CultureName {
		kinRivalry += 0.18
	}

	meanSupport := math.Min(from.MeanSupport, to.MeanSupport)
	if meanSupport <= 0 {
		meanSupport = 0.30
	}
	scarcity := clamp01((0.36 - meanSupport) / 0.22)
	crowding := clamp01((170.0 - math.Min(meshScaledTerritoryAreaCells(from.TerritoryCells, meshCellCount), meshScaledTerritoryAreaCells(to.TerritoryCells, meshCellCount))) / 140.0)
	ambition := profileRivalryDisposition(fromAssignment.Profile, toAssignment.Profile)

	penalty := borderFactor * (0.16 + 0.28*nicheOverlap + 0.18*economicOverlap + kinRivalry + 0.16*scarcity + 0.12*crowding)
	penalty *= 0.72 + 0.56*ambition
	if tradeAdj > 0 {
		penalty *= 1 - 0.34*clamp01(tradeAdj/0.36)
	}
	if hasSuzeraintyRelation(from.ID, to.ID, relations) {
		penalty *= 0.70
	}
	return penalty
}

func profileRivalryDisposition(a, b ResolvedProfile) float64 {
	return (profileConflictDisposition(a) + profileConflictDisposition(b)) / 2
}

func profileConflictDisposition(profile ResolvedProfile) float64 {
	aggression := 0.30
	honor := 0.35
	xenophilia := 0.45
	if profile.Attitudes != nil {
		aggression = profile.Attitudes.Aggression
		honor = profile.Attitudes.HonorBias
		xenophilia = profile.Attitudes.Xenophilia
	}
	warlike := profile.Traits["warlike"]
	chaos := profile.Traits["chaos"]
	return clamp01(0.42*aggression + 0.22*honor + 0.16*(1-xenophilia) + 0.12*warlike + 0.08*chaos)
}

func overlapScore(a, b, candidates []string) float64 {
	if len(candidates) == 0 {
		return 0
	}
	matches := 0.0
	total := 0.0
	for _, tag := range candidates {
		total++
		if hasProfileTag(a, tag) && hasProfileTag(b, tag) {
			matches++
		}
	}
	if total == 0 {
		return 0
	}
	return matches / total
}

func suzeraintyBonus(from, to int, relations []PolitySphereRelation) float64 {
	for _, relation := range relations {
		if relation.Overlord == from && relation.Subject == to {
			return 0.22 + 0.28*relation.Strength
		}
		if relation.Subject == from && relation.Overlord == to {
			return 0.08 + 0.16*relation.Strength
		}
	}
	return 0
}

func hasSuzeraintyRelation(from, to int, relations []PolitySphereRelation) bool {
	for _, relation := range relations {
		if relation.Kind != PolityRelationSuzerain {
			continue
		}
		if (relation.Overlord == from && relation.Subject == to) || (relation.Subject == from && relation.Overlord == to) {
			return true
		}
	}
	return false
}

func polityAllianceBonus(
	fromID, toID int,
	fromSphere PolitySphere,
	toSphere PolitySphere,
	from PolityProfileAssignment,
	to PolityProfileAssignment,
	affinity float64,
	tradeAdj float64,
	borderAdj float64,
	competitionAdj float64,
	relations []PolitySphereRelation,
) float64 {
	bonus := suzeraintyBonus(fromID, toID, relations)
	if borderAdj > 0.20 || competitionAdj > 0.18 {
		return bonus
	}

	tradeStrength := clamp01(tradeAdj / 0.18)
	if tradeStrength < 0.55 {
		return bonus
	}

	kinship := 0.0
	if from.Profile.CultureName != "" && from.Profile.CultureName == to.Profile.CultureName {
		kinship += 0.20
	}
	if from.Profile.AncestryName != "" && from.Profile.AncestryName == to.Profile.AncestryName {
		kinship += 0.12
	}
	if kinship == 0 && !polityRolesComplement(fromSphere, toSphere) {
		return bonus
	}

	affinityStrength := clamp01((affinity - 0.30) / 0.20)
	roleComplement := 0.0
	if polityRolesComplement(fromSphere, toSphere) {
		roleComplement = 0.10 + 0.12*tradeStrength
	}
	if roleComplement == 0 && affinityStrength < 0.35 {
		return bonus
	}
	ease := 1 - 0.48*clamp01(borderAdj/0.20) - 0.52*clamp01(competitionAdj/0.18)
	if ease <= 0 {
		return bonus
	}
	return bonus + (kinship*(0.45*tradeStrength+0.55*affinityStrength)+roleComplement)*ease
}

func polityRolesComplement(a, b PolitySphere) bool {
	if a.Coastal && b.River && !b.Coastal {
		return true
	}
	if b.Coastal && a.River && !a.Coastal {
		return true
	}
	if a.Style == ProtoCivilizationMaritime && b.Style == ProtoCivilizationRiverine {
		return true
	}
	if b.Style == ProtoCivilizationMaritime && a.Style == ProtoCivilizationRiverine {
		return true
	}
	if a.Style == ProtoCivilizationHighland && (b.Style == ProtoCivilizationRiverine || b.Style == ProtoCivilizationMaritime) {
		return true
	}
	if b.Style == ProtoCivilizationHighland && (a.Style == ProtoCivilizationRiverine || a.Style == ProtoCivilizationMaritime) {
		return true
	}
	return false
}

func classifyPolityAttitude(score float64) PolityAttitudeStance {
	switch {
	case score >= 0.55:
		return PolityAttitudeAllied
	case score >= 0.16:
		return PolityAttitudeFriendly
	case score <= -0.55:
		return PolityAttitudeHostile
	case score <= -0.12:
		return PolityAttitudeWary
	default:
		return PolityAttitudeNeutral
	}
}

func classifyDetailedPolityAttitude(score, allianceBonus float64, suzerainty bool) PolityAttitudeStance {
	if suzerainty && allianceBonus >= 0.38 && score >= 0.00 {
		return PolityAttitudeAllied
	}
	if suzerainty && allianceBonus >= 0.18 && score >= 0.24 {
		return PolityAttitudeAllied
	}
	if allianceBonus >= 0.16 && score >= 0.52 {
		return PolityAttitudeAllied
	}
	return classifyPolityAttitude(score)
}
