package climgen

type PopulationClass int

const (
	PopulationOcean PopulationClass = iota
	PopulationUninhabited
	PopulationSparseFrontier
	PopulationRural
	PopulationDenseRural
	PopulationUrban
)

func PopulationClassName(c PopulationClass) string {
	names := []string{
		"Ocean",
		"Uninhabited",
		"Sparse Frontier",
		"Rural",
		"Dense Rural",
		"Urban",
	}
	if int(c) < len(names) {
		return names[c]
	}
	return "Unknown"
}

type PopulationDiagnostics struct {
	FoodSupport      []float64
	WaterSupport     []float64
	TradeSupport     []float64
	ResourceSupport  []float64
	CarryingCapacity []float64
	UrbanPotential   []float64
}

type PopulationResult struct {
	Classes     []PopulationClass
	Diagnostics *PopulationDiagnostics
}

func ClassifyPopulationSupport(
	settlements *SettlementResult,
	agriculture *AgricultureResult,
	wildlife *WildlifeResult,
	waterResources *WaterResourceResult,
	coastalResources *CoastalResourceResult,
	resources *ResourceResult,
	cells []VoronoiCell,
	elevation []float64,
	seaLevel float64,
	settings PopulationSupportSettings,
) *PopulationResult {
	n := len(elevation)
	out := &PopulationResult{
		Classes: make([]PopulationClass, n),
		Diagnostics: &PopulationDiagnostics{
			FoodSupport:      make([]float64, n),
			WaterSupport:     make([]float64, n),
			TradeSupport:     make([]float64, n),
			ResourceSupport:  make([]float64, n),
			CarryingCapacity: make([]float64, n),
			UrbanPotential:   make([]float64, n),
		},
	}
	if settlements == nil || settlements.Diagnostics == nil {
		return out
	}

	rawFood := make([]float64, n)
	rawWater := make([]float64, n)
	rawTrade := make([]float64, n)
	rawResource := make([]float64, n)

	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Classes[i] = PopulationOcean
			continue
		}

		rawFood[i] = clamp01(
			(0.40*agricultureCropSupport(agriculture, i) +
				0.18*agriculturePastureSupport(agriculture, i) +
				0.10*agricultureFloodplainSupport(agriculture, i) +
				0.14*wildlifeGameSupport(wildlife, i) +
				0.08*wildlifeTimberSupport(wildlife, i) +
				0.10*coastalFisherySupport(coastalResources, i)) *
				settings.FoodMultiplier,
		)
		rawWater[i] = clamp01(
			(0.48*settlements.Diagnostics.WaterScore[i] +
				0.38*waterResourceSupport(waterResources, i) +
				0.14*settlements.Diagnostics.RiverBonus[i]) *
				settings.WaterMultiplier,
		)
		rawTrade[i] = clamp01(
			(0.44*settlements.Diagnostics.AccessScore[i] +
				0.24*settlements.Diagnostics.CoastalBonus[i] +
				0.22*settlements.Diagnostics.RiverBonus[i] +
				0.10*coastalFisherySupport(coastalResources, i)) *
				settings.TradeMultiplier,
		)
		rawResource[i] = clamp01(
			(0.62*settlements.Diagnostics.ResourceScore[i] +
				0.18*wildlifeTimberSupport(wildlife, i) +
				0.12*resourceFuelSupport(resources, i) +
				0.08*resourceLuxurySupport(resources, i)) *
				settings.ResourceMultiplier,
		)
	}

	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Classes[i] = PopulationOcean
			continue
		}

		food, water, trade, resource, suitability, hazard := populationCatchmentSupports(
			i,
			cells,
			elevation,
			seaLevel,
			rawFood,
			rawWater,
			rawTrade,
			rawResource,
			settlements.Diagnostics.Suitability,
			settlements.Diagnostics.HazardPenalty,
			settings,
		)

		carrying := clamp01(
			0.34*suitability +
				0.34*food +
				0.20*water +
				0.07*resource +
				0.05*trade,
		)
		urban := clamp01(
			(0.26*suitability+
				0.18*food+
				0.19*water+
				0.22*trade+
				0.15*resource)*
				settings.UrbanMultiplier -
				0.18*hazard,
		)

		out.Diagnostics.FoodSupport[i] = food
		out.Diagnostics.WaterSupport[i] = water
		out.Diagnostics.TradeSupport[i] = trade
		out.Diagnostics.ResourceSupport[i] = resource
		out.Diagnostics.CarryingCapacity[i] = carrying
		out.Diagnostics.UrbanPotential[i] = urban
		out.Classes[i] = determinePopulationClass(carrying, urban, settings)
	}

	return out
}

func populationCatchmentSupports(
	idx int,
	cells []VoronoiCell,
	elevation []float64,
	seaLevel float64,
	food []float64,
	water []float64,
	trade []float64,
	resource []float64,
	suitability []float64,
	hazard []float64,
	settings PopulationSupportSettings,
) (float64, float64, float64, float64, float64, float64) {
	rawFood := valueAt(food, idx)
	rawWater := valueAt(water, idx)
	rawTrade := valueAt(trade, idx)
	rawResource := valueAt(resource, idx)
	rawSuitability := valueAt(suitability, idx)
	rawHazard := valueAt(hazard, idx)
	if settings.CatchmentBlend <= 0 || settings.CatchmentHops <= 0 || len(cells) == 0 {
		return rawFood, rawWater, rawTrade, rawResource, rawSuitability, rawHazard
	}
	radius := meshResolutionAdjustedSteps(settings.CatchmentHops, len(cells))
	sumFood := rawFood
	sumWater := rawWater
	sumTrade := rawTrade
	sumResource := rawResource
	sumSuitability := rawSuitability
	sumHazard := rawHazard
	count := 1.0
	for _, cellIdx := range cellsWithinHops(cells, idx, radius) {
		if cellIdx < 0 || cellIdx >= len(elevation) || elevation[cellIdx] < seaLevel {
			continue
		}
		sumFood += valueAt(food, cellIdx)
		sumWater += valueAt(water, cellIdx)
		sumTrade += valueAt(trade, cellIdx)
		sumResource += valueAt(resource, cellIdx)
		sumSuitability += valueAt(suitability, cellIdx)
		sumHazard += valueAt(hazard, cellIdx)
		count++
	}
	blend := settings.CatchmentBlend
	return blendValue(rawFood, sumFood/count, blend),
		blendValue(rawWater, sumWater/count, blend),
		blendValue(rawTrade, sumTrade/count, blend),
		blendValue(rawResource, sumResource/count, blend),
		blendValue(rawSuitability, sumSuitability/count, blend),
		blendValue(rawHazard, sumHazard/count, blend)
}

func valueAt(values []float64, idx int) float64 {
	if idx < 0 || idx >= len(values) {
		return 0
	}
	return values[idx]
}

func blendValue(raw, catchment, blend float64) float64 {
	return clamp01((1-blend)*raw + blend*catchment)
}

func determinePopulationClass(carrying, urban float64, settings PopulationSupportSettings) PopulationClass {
	switch {
	case urban >= settings.UrbanThreshold && carrying >= settings.DenseRuralThreshold && urban >= carrying+0.02:
		return PopulationUrban
	case carrying >= settings.DenseRuralThreshold:
		return PopulationDenseRural
	case carrying >= settings.RuralThreshold:
		return PopulationRural
	case carrying >= settings.SparseThreshold:
		return PopulationSparseFrontier
	default:
		return PopulationUninhabited
	}
}

func agricultureCropSupport(result *AgricultureResult, idx int) float64 {
	if result == nil || result.Diagnostics == nil || idx < 0 || idx >= len(result.Types) {
		return 0
	}
	score := 0.0
	if idx < len(result.Diagnostics.CropPotential) {
		score += 0.68 * result.Diagnostics.CropPotential[idx]
	}
	if idx < len(result.Diagnostics.IrrigationPotential) {
		score += 0.18 * result.Diagnostics.IrrigationPotential[idx]
	}
	if idx < len(result.Diagnostics.FloodplainPotential) {
		score += 0.14 * result.Diagnostics.FloodplainPotential[idx]
	}
	switch result.Types[idx] {
	case AgricultureFloodplainCropland:
		score += 0.16
	case AgricultureIntensiveCropland:
		score += 0.14
	case AgricultureMixedFarming:
		score += 0.10
	case AgricultureDryFarming:
		score += 0.06
	}
	return clamp01(score)
}

func agriculturePastureSupport(result *AgricultureResult, idx int) float64 {
	if result == nil || result.Diagnostics == nil || idx < 0 || idx >= len(result.Types) {
		return 0
	}
	score := 0.0
	if idx < len(result.Diagnostics.PasturePotential) {
		score = result.Diagnostics.PasturePotential[idx]
	}
	if result.Types[idx] == AgriculturePastoral {
		score += 0.08
	}
	return clamp01(score)
}

func agricultureFloodplainSupport(result *AgricultureResult, idx int) float64 {
	if result == nil || result.Diagnostics == nil || idx < 0 || idx >= len(result.Types) {
		return 0
	}
	score := 0.0
	if idx < len(result.Diagnostics.FloodplainPotential) {
		score = result.Diagnostics.FloodplainPotential[idx]
	}
	if result.Types[idx] == AgricultureFloodplainCropland {
		score += 0.10
	}
	return clamp01(score)
}

func wildlifeGameSupport(result *WildlifeResult, idx int) float64 {
	if result == nil || result.Diagnostics == nil || idx < 0 || idx >= len(result.Types) {
		return 0
	}
	score := 0.0
	if idx < len(result.Diagnostics.GamePotential) {
		score = result.Diagnostics.GamePotential[idx]
	}
	switch result.Types[idx] {
	case WildlifeGrazingGame, WildlifeForestGame, WildlifeWetlandGame:
		score += 0.08
	}
	return clamp01(score)
}

func wildlifeTimberSupport(result *WildlifeResult, idx int) float64 {
	if result == nil || result.Diagnostics == nil || idx < 0 || idx >= len(result.Types) {
		return 0
	}
	score := 0.0
	if idx < len(result.Diagnostics.TimberPotential) {
		score = result.Diagnostics.TimberPotential[idx]
	}
	if result.Types[idx] == WildlifeTimber {
		score += 0.06
	}
	return clamp01(score)
}

func waterResourceSupport(result *WaterResourceResult, idx int) float64 {
	if result == nil || result.Diagnostics == nil || idx < 0 || idx >= len(result.Types) {
		return 0
	}
	score := 0.0
	if idx < len(result.Diagnostics.SurfaceReliability) {
		score += 0.40 * result.Diagnostics.SurfaceReliability[idx]
	}
	if idx < len(result.Diagnostics.SeasonalAvailability) {
		score += 0.18 * result.Diagnostics.SeasonalAvailability[idx]
	}
	if idx < len(result.Diagnostics.GroundwaterPotential) {
		score += 0.24 * result.Diagnostics.GroundwaterPotential[idx]
	}
	if idx < len(result.Diagnostics.LakeAccess) {
		score += 0.18 * result.Diagnostics.LakeAccess[idx]
	}
	switch result.Types[idx] {
	case WaterResourceReliableSurface:
		score += 0.14
	case WaterResourceGroundwater:
		score += 0.08
	case WaterResourceLakeOasis:
		score += 0.10
	case WaterResourceSeasonal:
		score += 0.04
	}
	return clamp01(score)
}

func coastalFisherySupport(result *CoastalResourceResult, idx int) float64 {
	if result == nil || result.Diagnostics == nil || idx < 0 || idx >= len(result.Types) {
		return 0
	}
	score := 0.0
	if idx < len(result.Diagnostics.OpenFishery) {
		score += 0.36 * result.Diagnostics.OpenFishery[idx]
	}
	if idx < len(result.Diagnostics.EstuarineFishery) {
		score += 0.38 * result.Diagnostics.EstuarineFishery[idx]
	}
	if idx < len(result.Diagnostics.ShellfishPotential) {
		score += 0.16 * result.Diagnostics.ShellfishPotential[idx]
	}
	if idx < len(result.Diagnostics.CoastalAccess) {
		score += 0.10 * result.Diagnostics.CoastalAccess[idx]
	}
	switch result.Types[idx] {
	case CoastalResourceEstuarineFishery:
		score += 0.12
	case CoastalResourceOpenFishery:
		score += 0.08
	case CoastalResourceShellfish:
		score += 0.06
	}
	return clamp01(score)
}

func resourceFuelSupport(result *ResourceResult, idx int) float64 {
	if result == nil || result.Diagnostics == nil || idx < 0 || idx >= len(result.Types) {
		return 0
	}
	score := 0.0
	if idx < len(result.Diagnostics.CoalAffinity) {
		score += 0.52 * result.Diagnostics.CoalAffinity[idx]
	}
	if idx < len(result.Diagnostics.OilGasAffinity) {
		score += 0.48 * result.Diagnostics.OilGasAffinity[idx]
	}
	switch result.Types[idx] {
	case ResourceCoal, ResourceOilGas:
		score += 0.10
	}
	return clamp01(score)
}

func resourceLuxurySupport(result *ResourceResult, idx int) float64 {
	if result == nil || result.Diagnostics == nil || idx < 0 || idx >= len(result.Types) {
		return 0
	}
	score := 0.0
	if idx < len(result.Diagnostics.GoldAffinity) {
		score += 0.46 * result.Diagnostics.GoldAffinity[idx]
	}
	if idx < len(result.Diagnostics.GemAffinity) {
		score += 0.54 * result.Diagnostics.GemAffinity[idx]
	}
	switch result.Types[idx] {
	case ResourceGoldOre, ResourceGemstones:
		score += 0.08
	}
	return clamp01(score)
}
