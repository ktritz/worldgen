package climgen

import "math"

type PolityEnvironmentContext struct {
	Tags   []string
	Traits map[string]float64
}

type PolityEcologyMetrics struct {
	MeanAridity      float64
	MeanWetland      float64
	MeanRelief       float64
	MeanRock         float64
	MeanIce          float64
	WetlandBiomeFrac float64
	WoodedFrac       float64
	ForestFrac       float64
	FloodplainFrac   float64
	DeltaFrac        float64
	LakeFrac         float64
	CoastOutletFrac  float64
}

func (ctx PolityEnvironmentContext) affinityContext() ProfileAffinityContext {
	return ProfileAffinityContext{
		Tags:   append([]string(nil), ctx.Tags...),
		Traits: cloneTraitMap(ctx.Traits),
	}
}

func selectPolityResolvedProfile(
	catalog *ProfileCatalog,
	sphere PolitySphere,
	polities *PolitySphereResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	biomes *BiomeResult,
	vegetation *VegetationResult,
	soils *SoilResult,
	hydro *HydrologyBiomeInputs,
) (ResolvedProfile, PolityProfileContext, PolityEnvironmentContext, float64) {
	context := buildPolityProfileContext(sphere, network, trade)
	metrics := computePolityEcologyMetrics(sphere.ID, polities, biomes, vegetation, soils, hydro)
	env := buildPolityEnvironmentContext(sphere, network, trade, metrics)
	ancestry, ancestryScore := selectPolityAncestry(catalog.Ancestries, env, context)
	if ancestry == nil {
		return ResolvedProfile{}, context, env, -math.MaxFloat64
	}
	culture, cultureScore := selectPolityCulture(catalog, *ancestry, env, context)
	resolved := resolvePolityProfile(catalog, *ancestry, culture)
	return resolved, context, env, ancestryScore + cultureScore
}

func buildPolityEnvironmentContext(
	sphere PolitySphere,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	metrics PolityEcologyMetrics,
) PolityEnvironmentContext {
	tags := []string{"surface"}
	traits := map[string]float64{}
	switch sphere.Style {
	case ProtoCivilizationMaritime:
		tags = append(tags, "coastal", "maritime")
		traits["mercantile"] = 0.28
	case ProtoCivilizationRiverine:
		tags = append(tags, "river", "lowland")
		traits["order"] = 0.34
	case ProtoCivilizationHighland:
		tags = append(tags, "mountain", "rugged", "fortress")
		traits["warlike"] = 0.48
	case ProtoCivilizationArid:
		tags = append(tags, "arid", "frontier", "pastoral")
		traits["warlike"] = 0.36
		traits["chaos"] = 0.26
	default:
		tags = append(tags, "inland")
	}
	if sphere.Coastal {
		tags = append(tags, "coastal")
	}
	if sphere.River {
		tags = append(tags, "river")
	}
	if metrics.FloodplainFrac >= 0.12 {
		tags = append(tags, "floodplain", "alluvial")
		if metrics.MeanAridity >= 0.90 && metrics.MeanWetland < 0.24 {
			tags = append(tags, "agrarian")
		}
	}
	if metrics.DeltaFrac >= 0.04 || (metrics.DeltaFrac >= 0.02 && metrics.CoastOutletFrac >= 0.05 && metrics.WetlandBiomeFrac >= 0.10) {
		tags = append(tags, "delta", "estuary")
	}
	if metrics.WetlandBiomeFrac >= 0.14 || (hasProfileTag(tags, "delta") && metrics.MeanWetland >= 0.30) {
		tags = append(tags, "wetland", "marsh")
	} else if metrics.MeanWetland >= 0.44 && metrics.FloodplainFrac < 0.10 && metrics.LakeFrac < 0.14 && metrics.CoastOutletFrac < 0.10 {
		tags = append(tags, "wetland")
	}
	if metrics.LakeFrac >= 0.18 {
		tags = append(tags, "lacustrine")
	}
	if metrics.MeanAridity <= 0.62 {
		tags = append(tags, "arid")
	}
	if metrics.MeanRelief >= 520 || metrics.MeanRock >= 0.58 {
		tags = append(tags, "mountain", "rugged")
	}
	if metrics.MeanIce >= 0.35 {
		tags = append(tags, "cold")
	}
	if metrics.ForestFrac >= 0.22 {
		tags = append(tags, "forest")
	}
	if metrics.WoodedFrac >= 0.38 {
		tags = append(tags, "wooded")
	}
	if sphere.CapitalNode >= 0 && sphere.CapitalNode < len(network.Nodes) {
		node := network.Nodes[sphere.CapitalNode]
		if node.Coastal {
			tags = append(tags, "coastal")
		}
		if node.River {
			tags = append(tags, "river")
		}
		if node.Kind >= SettlementNodeTown {
			tags = append(tags, "settled-core")
		}
	}
	if sphere.TerritoryCells >= 140 {
		tags = append(tags, "large-polity")
	}
	if trade != nil && trade.Diagnostics != nil && sphere.CapitalNode >= 0 && sphere.CapitalNode < len(trade.Diagnostics.NodeCentrality) {
		centrality := trade.Diagnostics.NodeCentrality[sphere.CapitalNode]
		if centrality >= 0.24 && sphere.Coastal {
			tags = append(tags, "seaborne")
		}
		if centrality >= 0.28 {
			tags = append(tags, "crossroads")
		}
	}
	return PolityEnvironmentContext{
		Tags:   mergeProfileTags(nil, tags),
		Traits: traits,
	}
}

func computePolityEcologyMetrics(
	polityID int,
	polities *PolitySphereResult,
	biomes *BiomeResult,
	vegetation *VegetationResult,
	soils *SoilResult,
	hydro *HydrologyBiomeInputs,
) PolityEcologyMetrics {
	var out PolityEcologyMetrics
	if polities == nil || polities.Diagnostics == nil {
		return out
	}
	count := 0.0
	for idx, owner := range polities.Diagnostics.PolityByCell {
		if owner != polityID {
			continue
		}
		count++
		if biomes != nil && biomes.Diagnostics != nil {
			if idx < len(biomes.Diagnostics.AridityRatio) {
				out.MeanAridity += biomes.Diagnostics.AridityRatio[idx]
			}
			if idx < len(biomes.Diagnostics.WetlandAffinity) {
				out.MeanWetland += biomes.Diagnostics.WetlandAffinity[idx]
			}
			if idx < len(biomes.Diagnostics.AnnualIceFraction) {
				out.MeanIce += biomes.Diagnostics.AnnualIceFraction[idx]
			}
			if idx < len(biomes.Biomes) && biomes.Biomes[idx] == BiomeWetland {
				out.WetlandBiomeFrac++
			}
		}
		if vegetation != nil && vegetation.Types != nil && idx < len(vegetation.Types) {
			switch vegetation.Types[idx] {
			case VegetationWoodland:
				out.WoodedFrac++
			case VegetationForest, VegetationRainforest, VegetationRiparianForest, VegetationCloudForest:
				out.WoodedFrac++
				out.ForestFrac++
			}
		}
		if soils != nil && soils.Diagnostics != nil {
			if idx < len(soils.Diagnostics.Relief) {
				out.MeanRelief += soils.Diagnostics.Relief[idx]
			}
			if idx < len(soils.Diagnostics.Rockiness) {
				out.MeanRock += soils.Diagnostics.Rockiness[idx]
			}
		}
		if hydro != nil && idx < len(hydro.CellClass) {
			switch hydro.CellClass[idx] {
			case "floodplain":
				out.FloodplainFrac++
			case "delta":
				out.DeltaFrac++
			case "lake_reach", "lake_complex":
				out.LakeFrac++
			case "coast_outlet":
				out.CoastOutletFrac++
			}
		}
	}
	if count == 0 {
		return out
	}
	out.MeanAridity /= count
	out.MeanWetland /= count
	out.MeanIce /= count
	out.MeanRelief /= count
	out.MeanRock /= count
	out.WetlandBiomeFrac /= count
	out.WoodedFrac /= count
	out.ForestFrac /= count
	out.FloodplainFrac /= count
	out.DeltaFrac /= count
	out.LakeFrac /= count
	out.CoastOutletFrac /= count
	return out
}

func selectPolityAncestry(
	ancestries []AncestryProfile,
	env PolityEnvironmentContext,
	context PolityProfileContext,
) (*AncestryProfile, float64) {
	if len(ancestries) == 0 {
		return nil, 0
	}
	bestIdx := 0
	bestScore := -math.MaxFloat64
	for i := range ancestries {
		score := scoreAncestryForPolity(ancestries[i], env, context)
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	return &ancestries[bestIdx], bestScore
}

func scoreAncestryForPolity(ancestry AncestryProfile, env PolityEnvironmentContext, context PolityProfileContext) float64 {
	profile := ComposeResolvedProfile(ancestry, nil)
	if profile == nil {
		return -math.MaxFloat64
	}
	score := ScoreProfileAffinity(profile, env.affinityContext())
	score += 0.28 * scoreTagSimilarity(profile.Tags, env.Tags)
	score += 0.16 * scoreTraitSimilarity(profile.Traits, env.Traits)
	score += 0.12 * scoreAttitudeSimilarity(profile.Attitudes, context.Attitudes)
	score += ancestryPrevalenceAdjustment(ancestry, env)
	score += ancestrySpecializationAdjustment(ancestry, env, context)
	return score
}

func ancestryPrevalenceAdjustment(ancestry AncestryProfile, env PolityEnvironmentContext) float64 {
	prevalence := ancestry.BaselinePrevalence
	if prevalence <= 0 {
		prevalence = 0.35
	}
	base := 0.70 * math.Log(prevalence/0.35)
	if base <= 0 {
		return base
	}
	mismatch := ancestrySpecialistMismatch(ancestry, env)
	return base * (1 - 0.75*mismatch)
}

func ancestrySpecialistMismatch(ancestry AncestryProfile, env PolityEnvironmentContext) float64 {
	specialistWeights := map[string]float64{
		"marsh":    1.00,
		"delta":    0.90,
		"mountain": 0.90,
		"highland": 0.90,
		"flight":   0.95,
		"forest":   0.75,
		"arid":     0.80,
	}
	total := 0.0
	missing := 0.0
	for tag, weight := range specialistWeights {
		if !hasProfileTag(env.Tags, tag) {
			continue
		}
		total += weight
		if ancestrySupportsSpecialistTag(ancestry, tag) < 0.5 {
			missing += weight
		}
	}
	if total == 0 {
		return 0
	}
	return clamp01(missing / total)
}

func ancestrySupportsSpecialistTag(ancestry AncestryProfile, tag string) float64 {
	if hasProfileTag(ancestry.Tags, tag) {
		return 1
	}
	switch tag {
	case "marsh":
		if hasProfileTag(ancestry.Tags, "wetland") || hasProfileTag(ancestry.Tags, "delta") {
			return 0.9
		}
	case "delta":
		if hasProfileTag(ancestry.Tags, "wetland") || hasProfileTag(ancestry.Tags, "marsh") {
			return 0.9
		}
	case "mountain":
		if hasProfileTag(ancestry.Tags, "highland") || hasProfileTag(ancestry.Tags, "subterranean") {
			return 0.8
		}
	case "highland":
		if hasProfileTag(ancestry.Tags, "mountain") || hasProfileTag(ancestry.Tags, "flight") {
			return 0.8
		}
	case "forest":
		if hasProfileTag(ancestry.Tags, "wooded") {
			return 0.75
		}
	case "arid":
		if hasProfileTag(ancestry.Tags, "frontier") || hasProfileTag(ancestry.Tags, "pastoral") {
			return 0.65
		}
	}
	for _, rule := range ancestry.Affinities {
		if rule.TargetType == "tag" && rule.Target == tag && rule.Weight > 0 {
			return clamp01(0.5 + rule.Weight)
		}
	}
	if hasProfileTag(ancestry.Tags, "adaptable") {
		return 0.35
	}
	return 0
}

func ancestrySpecializationAdjustment(ancestry AncestryProfile, env PolityEnvironmentContext, context PolityProfileContext) float64 {
	score := 0.0
	if hasProfileTag(ancestry.Tags, "adaptable") {
		score += 0.06
	}
	if hasProfileTag(ancestry.Tags, "river-valley") {
		if hasProfileTag(env.Tags, "river") {
			score += 0.08
		}
		if hasProfileTag(context.Tags, "urban") {
			score -= 0.08
		}
	}
	if hasProfileTag(ancestry.Tags, "smallhold") && hasProfileTag(context.Tags, "urban") {
		score -= 0.10
	}
	if hasProfileTag(ancestry.Tags, "low-density") && hasProfileTag(context.Tags, "urban") {
		score -= 0.14
	}
	if hasProfileTag(ancestry.Tags, "subterranean") && !hasProfileTag(env.Tags, "mountain") {
		score -= 0.16
	}
	if hasProfileTag(ancestry.Tags, "mountain") && hasProfileTag(env.Tags, "mountain") {
		score += 0.12
	}
	if hasProfileTag(ancestry.Tags, "wetland") {
		if hasProfileTag(env.Tags, "wetland") || hasProfileTag(env.Tags, "delta") || hasProfileTag(env.Tags, "marsh") {
			score += 0.10
		} else if hasProfileTag(env.Tags, "floodplain") || hasProfileTag(env.Tags, "river") {
			score -= 0.05
		} else {
			score -= 0.18
		}
	}
	if hasProfileTag(ancestry.Tags, "flight") {
		if hasProfileTag(env.Tags, "mountain") || hasProfileTag(env.Tags, "coastal") || hasProfileTag(env.Tags, "crossroads") {
			score += 0.10
		} else {
			score -= 0.06
		}
		if hasProfileTag(context.Tags, "urban") {
			score -= 0.04
		}
	}
	if hasProfileTag(ancestry.Tags, "arid") && hasProfileTag(env.Tags, "arid") {
		score += 0.14
	}
	if hasProfileTag(ancestry.Tags, "frontier") && hasProfileTag(env.Tags, "frontier") {
		score += 0.10
	}
	if hasProfileTag(ancestry.Tags, "imperial") && hasProfileTag(context.Tags, "imperial") {
		score += 0.12
	}
	if hasProfileTag(ancestry.Tags, "martial") && context.Attitudes != nil {
		score += 0.08 * context.Attitudes.Aggression
	}
	return score
}

func selectPolityCulture(
	catalog *ProfileCatalog,
	ancestry AncestryProfile,
	env PolityEnvironmentContext,
	context PolityProfileContext,
) (*CultureProfile, float64) {
	cultures := compatibleCulturesForAncestry(catalog, ancestry.Name)
	if len(cultures) == 0 {
		return nil, 0
	}
	bestIdx := 0
	bestScore := -math.MaxFloat64
	for i := range cultures {
		score := scoreCultureForPolity(cultures[i], ancestry, env, context)
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	if bestScore < 0.25 {
		return nil, bestScore
	}
	return &cultures[bestIdx], bestScore
}

func compatibleCulturesForAncestry(catalog *ProfileCatalog, ancestryName string) []CultureProfile {
	if catalog == nil {
		return nil
	}
	if len(catalog.Compositions) == 0 {
		return append([]CultureProfile(nil), catalog.Cultures...)
	}
	allowed := make(map[string]struct{})
	for _, spec := range catalog.Compositions {
		if spec.Ancestry == ancestryName && spec.Culture != "" {
			allowed[spec.Culture] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return nil
	}
	out := make([]CultureProfile, 0, len(allowed))
	for _, culture := range catalog.Cultures {
		if _, ok := allowed[culture.Name]; ok {
			out = append(out, culture)
		}
	}
	return out
}

func scoreCultureForPolity(culture CultureProfile, ancestry AncestryProfile, env PolityEnvironmentContext, context PolityProfileContext) float64 {
	resolved := ComposeResolvedProfile(ancestry, &culture)
	if resolved == nil {
		return -math.MaxFloat64
	}
	score := ScoreProfileAffinity(resolved, context.affinityContext())
	score += ScoreProfileAffinity(resolved, env.affinityContext())
	score += 0.24 * scoreTagSimilarity(culture.Tags, env.Tags)
	score += 0.10 * scoreTraitSimilarity(culture.Traits, env.Traits)
	score += 0.18 * scoreTagSimilarity(culture.Tags, context.Tags)
	score += 0.14 * scoreTraitSimilarity(culture.Traits, context.Traits)
	score += 0.40 * scoreSocialSimilarity(resolved.Social, context.Social)
	score += 0.45 * scoreGovernanceSimilarity(resolved.Governance, context.Governance)
	score += 0.55 * scoreEconomicSimilarity(resolved.Economy, context.Economy)
	score += 0.30 * scoreAttitudeSimilarity(resolved.Attitudes, context.Attitudes)
	score += cultureSpecializationAdjustment(culture, env, context)
	return score
}

func cultureSpecializationAdjustment(culture CultureProfile, env PolityEnvironmentContext, context PolityProfileContext) float64 {
	score := 0.0
	if hasProfileTag(culture.Tags, "mercantile") && hasProfileTag(context.Tags, "mercantile") {
		score += 0.10
	}
	if hasProfileTag(culture.Tags, "urban") && hasProfileTag(context.Tags, "urban") {
		score += 0.08
	}
	if hasProfileTag(culture.Tags, "imperial") {
		if hasProfileTag(context.Tags, "large-polity") {
			score += 0.10
		}
		if context.Governance != nil {
			score += 0.08 * context.Governance.CentralizationPreference
		}
	}
	if hasProfileTag(culture.Tags, "disciplined") && context.Attitudes != nil {
		score += 0.06 * context.Attitudes.HonorBias
	}
	if hasProfileTag(culture.Tags, "wetland") && !hasProfileTag(env.Tags, "wetland") && !hasProfileTag(env.Tags, "delta") && !hasProfileTag(env.Tags, "marsh") {
		score -= 0.20
	}
	if hasProfileTag(culture.Tags, "flight") && !hasProfileTag(env.Tags, "mountain") && !hasProfileTag(env.Tags, "coastal") {
		score -= 0.16
	}
	return score
}

func resolvePolityProfile(catalog *ProfileCatalog, ancestry AncestryProfile, culture *CultureProfile) ResolvedProfile {
	if culture == nil {
		return *ComposeResolvedProfile(ancestry, nil)
	}
	for _, spec := range catalog.Compositions {
		if spec.Ancestry == ancestry.Name && spec.Culture == culture.Name {
			if resolved := ResolveProfileComposition(catalog, spec); resolved != nil {
				return *resolved
			}
		}
	}
	return *ComposeResolvedProfile(ancestry, culture)
}
