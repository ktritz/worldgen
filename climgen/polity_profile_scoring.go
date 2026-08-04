package climgen

import "math"

type PolityProfileContext struct {
	Tags       []string
	Traits     map[string]float64
	Social     *ProfileSocialModule
	Governance *ProfileGovernanceModule
	Economy    *ProfileEconomicModule
	Attitudes  *ProfileAttitudeModule
}

func (ctx PolityProfileContext) affinityContext() ProfileAffinityContext {
	return ProfileAffinityContext{
		Tags:   append([]string(nil), ctx.Tags...),
		Traits: cloneTraitMap(ctx.Traits),
	}
}

func buildPolityProfileContext(sphere PolitySphere, network *SettlementNetworkResult, trade *TradeNetworkResult, meshCellCount int) PolityProfileContext {
	tags := []string{"polity"}
	traits := map[string]float64{}
	social := &ProfileSocialModule{}
	governance := &ProfileGovernanceModule{}
	economy := &ProfileEconomicModule{}
	attitudes := &ProfileAttitudeModule{}

	switch sphere.Style {
	case ProtoCivilizationMaritime:
		tags = append(tags, "coastal", "maritime")
		traits["mercantile"] = 0.62
		traits["order"] = 0.56
		social.Openness = 0.72
		social.GuildBias = 0.82
		social.HierarchyPreference = 0.42
		social.TraditionPreference = 0.36
		governance.RepublicBias = 0.66
		governance.LegalismPreference = 0.60
		governance.MeritPreference = 0.56
		governance.CentralizationPreference = 0.42
		governance.AutocracyBias = 0.16
		economy.TradeBias = 0.90
		economy.AgrarianBias = 0.24
		economy.CraftBias = 0.56
		economy.ExtractiveBias = 0.14
		attitudes.Xenophilia = 0.70
		attitudes.Curiosity = 0.64
		attitudes.Aggression = 0.22
		attitudes.HonorBias = 0.34
	case ProtoCivilizationRiverine:
		tags = append(tags, "river", "agrarian")
		traits["order"] = 0.58
		social.Openness = 0.48
		social.HierarchyPreference = 0.38
		social.TraditionPreference = 0.54
		social.ClanBias = 0.42
		social.GuildBias = 0.36
		governance.CentralizationPreference = 0.46
		governance.LegalismPreference = 0.54
		governance.MeritPreference = 0.42
		governance.RepublicBias = 0.30
		governance.AutocracyBias = 0.26
		economy.TradeBias = 0.34
		economy.AgrarianBias = 0.82
		economy.CraftBias = 0.28
		economy.ExtractiveBias = 0.10
		attitudes.Xenophilia = 0.42
		attitudes.Curiosity = 0.38
		attitudes.Aggression = 0.22
		attitudes.HonorBias = 0.34
	case ProtoCivilizationHighland:
		tags = append(tags, "mountain", "fortress", "clan")
		traits["warlike"] = 0.56
		traits["order"] = 0.66
		social.Openness = 0.24
		social.HierarchyPreference = 0.76
		social.TraditionPreference = 0.80
		social.ClanBias = 0.84
		social.GuildBias = 0.24
		governance.CentralizationPreference = 0.58
		governance.LegalismPreference = 0.66
		governance.MeritPreference = 0.46
		governance.RepublicBias = 0.10
		governance.AutocracyBias = 0.58
		economy.TradeBias = 0.20
		economy.AgrarianBias = 0.18
		economy.CraftBias = 0.74
		economy.ExtractiveBias = 0.84
		attitudes.Xenophilia = 0.18
		attitudes.Curiosity = 0.26
		attitudes.Aggression = 0.46
		attitudes.HonorBias = 0.78
	case ProtoCivilizationArid:
		tags = append(tags, "arid", "frontier")
		traits["warlike"] = 0.42
		traits["chaos"] = 0.34
		social.Openness = 0.30
		social.HierarchyPreference = 0.54
		social.TraditionPreference = 0.62
		social.ClanBias = 0.56
		social.GuildBias = 0.18
		governance.CentralizationPreference = 0.42
		governance.LegalismPreference = 0.40
		governance.MeritPreference = 0.34
		governance.RepublicBias = 0.16
		governance.AutocracyBias = 0.44
		governance.TheocracyBias = 0.28
		economy.TradeBias = 0.50
		economy.AgrarianBias = 0.14
		economy.CraftBias = 0.28
		economy.ExtractiveBias = 0.24
		attitudes.Xenophilia = 0.24
		attitudes.Curiosity = 0.34
		attitudes.Aggression = 0.40
		attitudes.HonorBias = 0.54
	default:
		tags = append(tags, "inland", "agrarian")
		traits["order"] = 0.46
		social.Openness = 0.42
		social.HierarchyPreference = 0.42
		social.TraditionPreference = 0.54
		social.ClanBias = 0.46
		social.GuildBias = 0.30
		governance.CentralizationPreference = 0.42
		governance.LegalismPreference = 0.50
		governance.MeritPreference = 0.42
		governance.RepublicBias = 0.24
		governance.AutocracyBias = 0.28
		governance.TheocracyBias = 0.16
		economy.TradeBias = 0.28
		economy.AgrarianBias = 0.64
		economy.CraftBias = 0.34
		economy.ExtractiveBias = 0.18
		attitudes.Xenophilia = 0.34
		attitudes.Curiosity = 0.34
		attitudes.Aggression = 0.28
		attitudes.HonorBias = 0.40
	}

	if sphere.Coastal {
		tags = append(tags, "coastal")
		economy.TradeBias = clamp01(economy.TradeBias + 0.12)
		attitudes.Xenophilia = clamp01(attitudes.Xenophilia + 0.08)
		governance.RepublicBias = clamp01(governance.RepublicBias + 0.06)
	}
	if sphere.River {
		tags = append(tags, "river")
		economy.AgrarianBias = clamp01(economy.AgrarianBias + 0.08)
	}
	if sphere.Secondary {
		tags = append(tags, "secondary")
		traits["hierarchy"] = 0.46
		governance.CentralizationPreference = clamp01(governance.CentralizationPreference + 0.06)
	}
	if meshScaledTerritoryAreaCells(sphere.TerritoryCells, meshCellCount) >= 140 {
		tags = append(tags, "large-polity")
		traits["hierarchy"] = math.Max(traits["hierarchy"], 0.54)
		governance.CentralizationPreference = clamp01(governance.CentralizationPreference + 0.08)
	}
	if sphere.CapitalNode >= 0 && sphere.CapitalNode < len(network.Nodes) {
		node := network.Nodes[sphere.CapitalNode]
		if node.Kind >= SettlementNodeTown {
			tags = append(tags, "urban")
			social.GuildBias = clamp01(social.GuildBias + 0.10)
			economy.CraftBias = clamp01(economy.CraftBias + 0.06)
		}
	}
	if trade != nil && trade.Diagnostics != nil && sphere.CapitalNode >= 0 && sphere.CapitalNode < len(trade.Diagnostics.NodeCentrality) {
		centrality := trade.Diagnostics.NodeCentrality[sphere.CapitalNode]
		if centrality >= 0.20 || (sphere.Coastal && centrality >= 0.14) {
			tags = append(tags, "mercantile")
			traits["mercantile"] = math.Max(traits["mercantile"], clamp01(0.52+centrality))
		}
		economy.TradeBias = clamp01(economy.TradeBias + 0.25*centrality)
		social.Openness = clamp01(social.Openness + 0.18*centrality)
		social.GuildBias = clamp01(social.GuildBias + 0.24*centrality)
		attitudes.Xenophilia = clamp01(attitudes.Xenophilia + 0.16*centrality)
	}

	return PolityProfileContext{
		Tags:       mergeProfileTags(nil, tags),
		Traits:     traits,
		Social:     social,
		Governance: governance,
		Economy:    economy,
		Attitudes:  attitudes,
	}
}

func scoreProfileForPolity(profile ResolvedProfile, context PolityProfileContext) float64 {
	score := ScoreProfileAffinity(&profile, context.affinityContext())
	score += 0.22 * scoreTagSimilarity(profile.Tags, context.Tags)
	score += 0.18 * scoreTraitSimilarity(profile.Traits, context.Traits)
	score += 0.35 * scoreSocialSimilarity(profile.Social, context.Social)
	score += 0.35 * scoreGovernanceSimilarity(profile.Governance, context.Governance)
	score += 0.45 * scoreEconomicSimilarity(profile.Economy, context.Economy)
	score += 0.25 * scoreAttitudeSimilarity(profile.Attitudes, context.Attitudes)
	return score
}

func scoreTagSimilarity(profileTags, contextTags []string) float64 {
	if len(profileTags) == 0 || len(contextTags) == 0 {
		return 0
	}
	shared := 0
	for _, tag := range profileTags {
		if hasProfileTag(contextTags, tag) {
			shared++
		}
	}
	return (2 * float64(shared)) / float64(len(profileTags)+len(contextTags))
}

func scoreTraitSimilarity(profileTraits, contextTraits map[string]float64) float64 {
	if len(profileTraits) == 0 || len(contextTraits) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(profileTraits)+len(contextTraits))
	total := 0.0
	count := 0
	for trait, value := range profileTraits {
		total += 1 - math.Abs(value-contextTraits[trait])
		seen[trait] = struct{}{}
		count++
	}
	for trait, value := range contextTraits {
		if _, ok := seen[trait]; ok {
			continue
		}
		total += 1 - math.Abs(profileTraits[trait]-value)
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func scoreSocialSimilarity(profile *ProfileSocialModule, context *ProfileSocialModule) float64 {
	if profile == nil || context == nil {
		return 0
	}
	return averageCloseness(
		profile.Openness, context.Openness,
		profile.HierarchyPreference, context.HierarchyPreference,
		profile.TraditionPreference, context.TraditionPreference,
		profile.ClanBias, context.ClanBias,
		profile.GuildBias, context.GuildBias,
	)
}

func scoreGovernanceSimilarity(profile *ProfileGovernanceModule, context *ProfileGovernanceModule) float64 {
	if profile == nil || context == nil {
		return 0
	}
	return averageCloseness(
		profile.CentralizationPreference, context.CentralizationPreference,
		profile.LegalismPreference, context.LegalismPreference,
		profile.MeritPreference, context.MeritPreference,
		profile.RepublicBias, context.RepublicBias,
		profile.AutocracyBias, context.AutocracyBias,
		profile.TheocracyBias, context.TheocracyBias,
	)
}

func scoreEconomicSimilarity(profile *ProfileEconomicModule, context *ProfileEconomicModule) float64 {
	if profile == nil || context == nil {
		return 0
	}
	return averageCloseness(
		profile.TradeBias, context.TradeBias,
		profile.AgrarianBias, context.AgrarianBias,
		profile.CraftBias, context.CraftBias,
		profile.ExtractiveBias, context.ExtractiveBias,
	)
}

func scoreAttitudeSimilarity(profile *ProfileAttitudeModule, context *ProfileAttitudeModule) float64 {
	if profile == nil || context == nil {
		return 0
	}
	return averageCloseness(
		profile.Xenophilia, context.Xenophilia,
		profile.Aggression, context.Aggression,
		profile.HonorBias, context.HonorBias,
		profile.Curiosity, context.Curiosity,
	)
}

func averageCloseness(values ...float64) float64 {
	if len(values) == 0 || len(values)%2 != 0 {
		return 0
	}
	total := 0.0
	pairs := 0
	for i := 0; i < len(values); i += 2 {
		total += 1 - math.Abs(values[i]-values[i+1])
		pairs++
	}
	if pairs == 0 {
		return 0
	}
	return total / float64(pairs)
}
