package climgen

func tradeGoodFishPotential(inputs TradeGoodInputs, idx int) float64 {
	coastalFish := 0.0
	if inputs.Coastal != nil && inputs.Coastal.Diagnostics != nil {
		shellfish := safeSliceValue(inputs.Coastal.Diagnostics.ShellfishPotential, idx)
		coastalFish = clamp01(
			0.55*safeSliceValue(inputs.Coastal.Diagnostics.OpenFishery, idx) +
				0.35*safeSliceValue(inputs.Coastal.Diagnostics.EstuarineFishery, idx) +
				0.10*shellfish,
		)
	}
	surface := 0.0
	lake := 0.0
	if inputs.Water != nil && inputs.Water.Diagnostics != nil {
		surface = safeSliceValue(inputs.Water.Diagnostics.SurfaceReliability, idx)
		lake = safeSliceValue(inputs.Water.Diagnostics.LakeAccess, idx)
	}
	runoff := hydrologyValue(inputs.Hydro, idx, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
	channel := hydrologyChannelCorridorStrength(inputs.Hydro, idx)
	wetland := 0.0
	riparian := 0.0
	if inputs.Vegetation != nil && inputs.Vegetation.Diagnostics != nil {
		wetland = safeSliceValue(inputs.Vegetation.Diagnostics.WetlandCover, idx)
		riparian = safeSliceValue(inputs.Vegetation.Diagnostics.RiparianAffinity, idx)
	}
	freshwaterFish := clamp01(
		0.28*surface +
			0.24*lake +
			0.18*smoothstep01(0.55, 2.1, channel) +
			0.14*smoothstep01(8, 90, runoff) +
			0.10*riparian +
			0.06*wetland,
	)
	return clamp01(maxFloat(coastalFish, freshwaterFish))
}

func tradeGoodSaltPotential(inputs TradeGoodInputs, idx int, evaporite float64) float64 {
	coastalSalt := 0.0
	if inputs.Coastal != nil && inputs.Coastal.Diagnostics != nil {
		coastalSalt = safeSliceValue(inputs.Coastal.Diagnostics.SaltworksPotential, idx)
	}

	soilSalinity := 0.0
	if inputs.Soils != nil && inputs.Soils.Diagnostics != nil {
		soilSalinity = safeSliceValue(inputs.Soils.Diagnostics.Salinity, idx)
	}

	lakeAccess := 0.0
	groundwater := 0.0
	if inputs.Water != nil && inputs.Water.Diagnostics != nil {
		lakeAccess = safeSliceValue(inputs.Water.Diagnostics.LakeAccess, idx)
		groundwater = safeSliceValue(inputs.Water.Diagnostics.GroundwaterPotential, idx)
	}

	aridity := 0.0
	if inputs.Biome != nil && inputs.Biome.Diagnostics != nil {
		aridity = smoothstep01(1.10, 0.32, safeSliceValue(inputs.Biome.Diagnostics.AridityRatio, idx))
	}

	runoff := hydrologyValue(inputs.Hydro, idx, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
	inlandBasin := inlandSalineBasinFactor(inputs.Hydro, idx)
	lowRunoff := 1 - smoothstep01(12, 90, runoff)

	inlandSalt := clamp01(
		0.30*soilSalinity +
			0.20*lakeAccess +
			0.18*evaporite +
			0.16*inlandBasin +
			0.08*groundwater +
			0.08*aridity +
			0.08*lowRunoff,
	)
	return clamp01(maxFloat(coastalSalt, inlandSalt))
}

func tradeGoodIronOrePotential(inputs TradeGoodInputs, idx int, hardRock float64) float64 {
	wetland := 0.0
	if inputs.Vegetation != nil && inputs.Vegetation.Diagnostics != nil {
		wetland = safeSliceValue(inputs.Vegetation.Diagnostics.WetlandCover, idx)
	}

	alluvial := 0.0
	organic := 0.0
	if inputs.Soils != nil && inputs.Soils.Diagnostics != nil {
		alluvial = safeSliceValue(inputs.Soils.Diagnostics.Alluvial, idx)
		organic = safeSliceValue(inputs.Soils.Diagnostics.Organic, idx)
	}

	runoff := hydrologyValue(inputs.Hydro, idx, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
	channel := hydrologyChannelCorridorStrength(inputs.Hydro, idx)
	basinWet := hydrologyClassFactor(inputs.Hydro, idx)
	bogIron := clamp01(
		0.30*wetland +
			0.22*organic +
			0.20*alluvial +
			0.14*basinWet +
			0.08*smoothstep01(10, 85, runoff) +
			0.06*smoothstep01(0.5, 2.2, channel),
	)
	return clamp01(maxFloat(hardRock, 0.42*bogIron))
}

func tradeGoodClayPotential(inputs TradeGoodInputs, idx int, base float64) float64 {
	alluvial := 0.0
	drainage := 0.0
	rockiness := 0.0
	if inputs.Soils != nil && inputs.Soils.Diagnostics != nil {
		alluvial = safeSliceValue(inputs.Soils.Diagnostics.Alluvial, idx)
		drainage = safeSliceValue(inputs.Soils.Diagnostics.Drainage, idx)
		rockiness = safeSliceValue(inputs.Soils.Diagnostics.Rockiness, idx)
	}

	lake := 0.0
	if inputs.Water != nil && inputs.Water.Diagnostics != nil {
		lake = safeSliceValue(inputs.Water.Diagnostics.LakeAccess, idx)
	}

	wetland := 0.0
	if inputs.Vegetation != nil && inputs.Vegetation.Diagnostics != nil {
		wetland = safeSliceValue(inputs.Vegetation.Diagnostics.WetlandCover, idx)
	}

	runoff := hydrologyValue(inputs.Hydro, idx, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
	channel := hydrologyChannelCorridorStrength(inputs.Hydro, idx)
	depositionalClass := hydrologyClassFactor(inputs.Hydro, idx)
	depositionalSignal := clamp01(
		0.30*alluvial +
			0.20*depositionalClass +
			0.16*lake +
			0.12*wetland +
			0.10*(1-smoothstep01(0.45, 0.92, drainage)) +
			0.07*smoothstep01(10, 80, runoff) +
			0.05*smoothstep01(0.5, 2.0, channel) +
			0.08*(1-rockiness),
	)
	return clamp01(maxFloat(base, 0.62*depositionalSignal))
}

func tradeGoodCopperOrePotential(inputs TradeGoodInputs, idx int, base float64, placerBoost float64) float64 {
	alluvial := 0.0
	if inputs.Soils != nil && inputs.Soils.Diagnostics != nil {
		alluvial = safeSliceValue(inputs.Soils.Diagnostics.Alluvial, idx)
	}

	wetland := 0.0
	riparian := 0.0
	if inputs.Vegetation != nil && inputs.Vegetation.Diagnostics != nil {
		wetland = safeSliceValue(inputs.Vegetation.Diagnostics.WetlandCover, idx)
		riparian = safeSliceValue(inputs.Vegetation.Diagnostics.RiparianAffinity, idx)
	}

	runoff := hydrologyValue(inputs.Hydro, idx, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
	channel := hydrologyChannelCorridorStrength(inputs.Hydro, idx)
	placerReach := preciousPlacerClassFactor(inputs.Hydro, idx)
	placerSignal := clamp01(
		0.32*alluvial +
			0.24*smoothstep01(0.4, 2.0, channel) +
			0.18*smoothstep01(8, 85, runoff) +
			0.14*placerReach +
			0.07*riparian +
			0.05*wetland,
	)
	return clamp01(maxFloat(base, 0.44*placerBoost*placerSignal))
}

func tradeGoodPreciousOrePotential(inputs TradeGoodInputs, idx int, base float64, placerBoost float64) float64 {
	alluvial := 0.0
	if inputs.Soils != nil && inputs.Soils.Diagnostics != nil {
		alluvial = safeSliceValue(inputs.Soils.Diagnostics.Alluvial, idx)
	}

	wetland := 0.0
	riparian := 0.0
	if inputs.Vegetation != nil && inputs.Vegetation.Diagnostics != nil {
		wetland = safeSliceValue(inputs.Vegetation.Diagnostics.WetlandCover, idx)
		riparian = safeSliceValue(inputs.Vegetation.Diagnostics.RiparianAffinity, idx)
	}

	runoff := hydrologyValue(inputs.Hydro, idx, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
	channel := hydrologyChannelCorridorStrength(inputs.Hydro, idx)
	placerReach := preciousPlacerClassFactor(inputs.Hydro, idx)
	placerSignal := clamp01(
		0.34*alluvial +
			0.22*placerReach +
			0.18*smoothstep01(8, 85, runoff) +
			0.14*smoothstep01(0.4, 2.0, channel) +
			0.07*riparian +
			0.05*wetland,
	)
	return clamp01(maxFloat(base, 0.60*placerBoost*placerSignal))
}

func inlandSalineBasinFactor(hydro *HydrologyBiomeInputs, idx int) float64 {
	if hydro == nil || idx < 0 || idx >= len(hydro.CellClass) {
		return 0
	}
	switch hydro.CellClass[idx] {
	case "endorheic_basin":
		return 1.0
	case "lake", "lake_complex":
		return 0.82
	case "lake_reach":
		return 0.45
	default:
		return 0
	}
}

func preciousPlacerClassFactor(hydro *HydrologyBiomeInputs, idx int) float64 {
	if hydro == nil || idx < 0 || idx >= len(hydro.CellClass) {
		return 0
	}
	switch hydro.CellClass[idx] {
	case "delta":
		return 1.0
	case "floodplain":
		return 0.78
	case "confluence":
		return 0.66
	case "lake_reach":
		return 0.52
	default:
		return 0
	}
}

func tradeGoodResourceFieldValue(idx int, affinity []float64) float64 {
	return clamp01(safeSliceValue(affinity, idx))
}

func tradeGoodPreciousOreComponent(inputs TradeGoodInputs, idx int, affinity []float64, placerAffinity []float64, placerBoost float64) float64 {
	value := tradeGoodResourceFieldValue(idx, affinity)
	effectivePlacerBoost := maxFloat(placerBoost, safeSliceValue(placerAffinity, idx))
	return tradeGoodPreciousOrePotential(inputs, idx, clamp01(value), clamp01(effectivePlacerBoost))
}

func tradeGoodCopperOreComponent(inputs TradeGoodInputs, idx int, affinity []float64, placerAffinity []float64) float64 {
	base := tradeGoodResourceFieldValue(idx, affinity)
	placerBoost := safeSliceValue(placerAffinity, idx)
	return tradeGoodCopperOrePotential(inputs, idx, clamp01(base), clamp01(placerBoost))
}
