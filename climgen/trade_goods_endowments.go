package climgen

func ComputeTradeGoodEndowments(inputs TradeGoodInputs, settings TradeGoodsSettings) *TradeGoodResult {
	n := len(inputs.Elevation)
	out := &TradeGoodResult{
		Goods: make([]TradeGoodEndowment, 0, len(settings.Goods)),
		Diagnostics: &TradeGoodDiagnostics{
			SourceFields:       tradeGoodSourceFields(inputs),
			AvailabilityByGood: map[string]float64{},
			ScarcityByGood:     map[string]float64{},
		},
	}
	for _, spec := range settings.Goods {
		potential := make([]float64, n)
		for i := 0; i < n; i++ {
			if i < len(inputs.Elevation) && inputs.Elevation[i] < inputs.SeaLevel {
				continue
			}
			potential[i] = tradeGoodCellPotential(spec, out.Diagnostics.SourceFields, i)
		}
		out.Goods = append(out.Goods, TradeGoodEndowment{
			Good:      spec.Name,
			Category:  spec.Category,
			Potential: potential,
		})
	}
	populateTradeGoodScarcity(out, settings)
	return out
}

func tradeGoodCellPotential(spec TradeGoodSpec, sources map[string][]float64, idx int) float64 {
	if len(spec.SourceWeights) == 0 {
		return 0
	}
	total := 0.0
	weight := 0.0
	for source, sourceWeight := range spec.SourceWeights {
		if sourceWeight <= 0 {
			continue
		}
		field := sources[source]
		if idx < 0 || idx >= len(field) {
			continue
		}
		total += sourceWeight * field[idx]
		weight += sourceWeight
	}
	if weight == 0 {
		return 0
	}
	return clamp01(total / weight)
}

func tradeGoodSourceFields(inputs TradeGoodInputs) map[string][]float64 {
	n := len(inputs.Elevation)
	fields := map[string][]float64{}
	add := func(name string) []float64 {
		field := make([]float64, n)
		fields[name] = field
		return field
	}
	crop := add("crop")
	pasture := add("pasture")
	fish := add("fish")
	shellfish := add("shellfish")
	salt := add("salt")
	timber := add("timber")
	game := add("game")
	pelts := add("pelts")
	iron := add("iron_ore")
	copper := add("copper_ore")
	leadSilver := add("lead_silver_ore")
	gold := add("gold_ore")
	gems := add("gemstones")
	coal := add("coal")
	evaporite := add("evaporite")
	clay := add("clay")
	stone := add("stone")
	herbs := add("herbs")
	dyes := add("dye_plants")
	fiber := add("fiber")
	resin := add("resin")

	for i := 0; i < n; i++ {
		evaporitePotential := 0.0
		if inputs.Agriculture != nil && inputs.Agriculture.Diagnostics != nil {
			crop[i] = safeSliceValue(inputs.Agriculture.Diagnostics.CropPotential, i)
			pasture[i] = safeSliceValue(inputs.Agriculture.Diagnostics.PasturePotential, i)
		}
		fish[i] = tradeGoodFishPotential(inputs, i)
		if inputs.Coastal != nil && inputs.Coastal.Diagnostics != nil {
			shellfish[i] = safeSliceValue(inputs.Coastal.Diagnostics.ShellfishPotential, i)
		}
		if inputs.Wildlife != nil && inputs.Wildlife.Diagnostics != nil {
			timber[i] = safeSliceValue(inputs.Wildlife.Diagnostics.TimberPotential, i)
			game[i] = safeSliceValue(inputs.Wildlife.Diagnostics.GamePotential, i)
			pelts[i] = safeSliceValue(inputs.Wildlife.Diagnostics.PeltPotential, i)
		}
		if inputs.Resources != nil && inputs.Resources.Diagnostics != nil {
			iron[i] = tradeGoodIronOrePotential(
				inputs,
				i,
				tradeGoodResourceFieldValue(i, inputs.Resources.Diagnostics.IronAffinity),
			)
			copper[i] = tradeGoodCopperOreComponent(inputs, i, inputs.Resources.Diagnostics.CopperAffinity, inputs.Resources.Diagnostics.PlacerAffinity)
			leadSilver[i] = tradeGoodPreciousOreComponent(inputs, i, inputs.Resources.Diagnostics.LeadSilverAffinity, inputs.Resources.Diagnostics.PlacerAffinity, 0.58)
			gold[i] = tradeGoodPreciousOreComponent(inputs, i, inputs.Resources.Diagnostics.GoldAffinity, inputs.Resources.Diagnostics.PlacerAffinity, 0.62)
			gems[i] = tradeGoodPreciousOreComponent(inputs, i, inputs.Resources.Diagnostics.GemAffinity, inputs.Resources.Diagnostics.PlacerAffinity, 0.56)
			coal[i] = tradeGoodResourceFieldValue(i, inputs.Resources.Diagnostics.CoalAffinity)
			evaporitePotential = tradeGoodResourceFieldValue(i, inputs.Resources.Diagnostics.EvaporiteAffinity)
			evaporite[i] = evaporitePotential
			clay[i] = tradeGoodClayPotential(
				inputs,
				i,
				tradeGoodResourceFieldValue(i, inputs.Resources.Diagnostics.ClayAffinity),
			)
			stone[i] = tradeGoodResourceFieldValue(i, inputs.Resources.Diagnostics.StoneAffinity)
		}
		if inputs.Resources == nil || inputs.Resources.Diagnostics == nil {
			iron[i] = tradeGoodIronOrePotential(inputs, i, 0)
			clay[i] = tradeGoodClayPotential(inputs, i, 0)
		}
		salt[i] = tradeGoodSaltPotential(inputs, i, evaporitePotential)
		if inputs.Vegetation != nil && inputs.Vegetation.Diagnostics != nil {
			v := inputs.Vegetation.Diagnostics
			grass := safeSliceValue(v.GrassCover, i)
			shrub := safeSliceValue(v.ShrubCover, i)
			wetland := safeSliceValue(v.WetlandCover, i)
			tree := safeSliceValue(v.TreeCover, i)
			riparian := safeSliceValue(v.RiparianAffinity, i)
			mangrove := safeSliceValue(v.MangroveAffinity, i)
			saltMarsh := safeSliceValue(v.SaltMarshAffinity, i)
			herbs[i] = clamp01(0.28*shrub + 0.22*wetland + 0.18*riparian + 0.16*grass + 0.16*mangrove)
			dyes[i] = clamp01(0.30*wetland + 0.22*shrub + 0.20*mangrove + 0.18*saltMarsh + 0.10*tree)
			fiber[i] = clamp01(0.36*crop[i] + 0.24*grass + 0.22*wetland + 0.18*pasture[i])
			resin[i] = clamp01(0.58*tree + 0.24*shrub + 0.18*timber[i])
		}
	}
	return fields
}
