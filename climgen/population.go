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

	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Classes[i] = PopulationOcean
			continue
		}

		food := clamp01(
			(0.40*agricultureCropSupport(agriculture, i) +
				0.18*agriculturePastureSupport(agriculture, i) +
				0.10*agricultureFloodplainSupport(agriculture, i) +
				0.14*wildlifeGameSupport(wildlife, i) +
				0.08*wildlifeTimberSupport(wildlife, i) +
				0.10*coastalFisherySupport(coastalResources, i)) *
				settings.FoodMultiplier,
		)
		water := clamp01(
			(0.48*settlements.Diagnostics.WaterScore[i] +
				0.38*waterResourceSupport(waterResources, i) +
				0.14*settlements.Diagnostics.RiverBonus[i]) *
				settings.WaterMultiplier,
		)
		trade := clamp01(
			(0.44*settlements.Diagnostics.AccessScore[i] +
				0.24*settlements.Diagnostics.CoastalBonus[i] +
				0.22*settlements.Diagnostics.RiverBonus[i] +
				0.10*coastalFisherySupport(coastalResources, i)) *
				settings.TradeMultiplier,
		)
		resource := clamp01(
			(0.62*settlements.Diagnostics.ResourceScore[i] +
				0.18*wildlifeTimberSupport(wildlife, i) +
				0.12*resourceFuelSupport(resources, i) +
				0.08*resourceLuxurySupport(resources, i)) *
				settings.ResourceMultiplier,
		)

		carrying := clamp01(
			0.34*settlements.Diagnostics.Suitability[i] +
				0.34*food +
				0.20*water +
				0.07*resource +
				0.05*trade,
		)
		urban := clamp01(
			(0.26*settlements.Diagnostics.Suitability[i]+
				0.18*food+
				0.19*water+
				0.22*trade+
				0.15*resource)*
				settings.UrbanMultiplier -
				0.18*settlements.Diagnostics.HazardPenalty[i],
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
