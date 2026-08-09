package climgen

type AgricultureType int

const (
	AgricultureOcean AgricultureType = iota
	AgricultureUnsuitable
	AgriculturePastoral
	AgricultureDryFarming
	AgricultureMixedFarming
	AgricultureIntensiveCropland
	AgricultureFloodplainCropland
)

func AgricultureName(a AgricultureType) string {
	names := []string{
		"Ocean",
		"Unsuitable",
		"Pastoral",
		"Dry Farming",
		"Mixed Farming",
		"Intensive Cropland",
		"Floodplain Cropland",
	}
	if int(a) < len(names) {
		return names[a]
	}
	return "Unknown"
}

type AgricultureDiagnostics struct {
	CropPotential       []float64
	PasturePotential    []float64
	IrrigationPotential []float64
	FloodplainPotential []float64
	ClimateSuitability  []float64
	SoilSuitability     []float64
	TerrainSuitability  []float64
	ColdStress          []float64
	DryStress           []float64
	WaterloggingHazard  []float64
}

type AgricultureResult struct {
	Types       []AgricultureType
	Diagnostics *AgricultureDiagnostics
}

func ClassifyAgriculture(
	biomes *BiomeResult,
	soils *SoilResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	settings AgricultureProductivitySettings,
) *AgricultureResult {
	n := len(elevation)
	out := &AgricultureResult{
		Types: make([]AgricultureType, n),
		Diagnostics: &AgricultureDiagnostics{
			CropPotential:       make([]float64, n),
			PasturePotential:    make([]float64, n),
			IrrigationPotential: make([]float64, n),
			FloodplainPotential: make([]float64, n),
			ClimateSuitability:  make([]float64, n),
			SoilSuitability:     make([]float64, n),
			TerrainSuitability:  make([]float64, n),
			ColdStress:          make([]float64, n),
			DryStress:           make([]float64, n),
			WaterloggingHazard:  make([]float64, n),
		},
	}
	if biomes == nil || biomes.Diagnostics == nil || soils == nil || soils.Diagnostics == nil {
		return out
	}
	diag := biomes.Diagnostics
	sdiag := soils.Diagnostics
	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Types[i] = AgricultureOcean
			continue
		}
		runoff := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
		channel := hydrologyChannelCorridorStrength(hydro, i)
		classWet := hydrologyClassFactor(hydro, i)

		coldStress := clamp01(maxf(
			diag.AnnualIceFraction[i],
			smoothstep01(12, -8, diag.WarmestSeasonTempC[i]),
		))
		dryStress := clamp01(
			smoothstep01(0.35, 0.05, diag.DrySeasonRatio[i]) *
				smoothstep01(0.75, 0.18, diag.AridityRatio[i]),
		)
		waterlogging := clamp01(
			0.40*smoothstep01(18, 110, runoff) +
				0.25*smoothstep01(0.9, 2.4, channel) +
				0.20*(1-sdiag.Drainage[i]) +
				0.15*classWet,
		)

		climateSuit := clamp01(
			0.28*smoothstep01(4, 24, diag.AnnualMeanTempC[i]) +
				0.22*smoothstep01(10, 28, diag.WarmestSeasonTempC[i]) +
				0.22*peak01(diag.AnnualPrecipCm[i], 25, 95, 220) +
				0.18*smoothstep01(0.55, 2.10, diag.AridityRatio[i]) +
				0.10*(1-coldStress),
		)
		soilSuit := clamp01(
			0.42*sdiag.Fertility[i] +
				0.20*sdiag.Alluvial[i] +
				0.16*sdiag.Drainage[i] +
				0.12*(1-sdiag.Salinity[i]) +
				0.10*(1-sdiag.Rockiness[i]),
		)
		terrainSuit := clamp01(
			0.65*(1-smoothstep01(120, 1000, sdiag.Relief[i])) +
				0.35*(1-smoothstep01(1200, 3600, elevation[i])),
		)
		irrigation := clamp01(
			(0.45*smoothstep01(0.8, 2.2, channel) +
				0.30*smoothstep01(10, 95, runoff) +
				0.25*sdiag.Alluvial[i]) *
				(0.45 + 0.55*dryStress),
		)
		floodplain := clamp01(
			0.40*sdiag.Alluvial[i] +
				0.25*classWet +
				0.20*irrigation +
				0.15*smoothstep01(12, 90, runoff),
		)

		crop := clamp01(
			(0.36*climateSuit+
				0.28*soilSuit+
				0.16*terrainSuit+
				0.10*smoothstep01(0.55, 1.8, diag.AridityRatio[i])+
				0.10*irrigation)*
				settings.CropMultiplier -
				0.18*coldStress -
				0.14*waterlogging,
		)
		pasture := clamp01(
			(0.26*peak01(diag.AridityRatio[i], 0.6, 1.35, 2.3)+
				0.24*diag.GrasslandAffinity[i]+
				0.18*terrainSuit+
				0.16*soilSuit+
				0.10*smoothstep01(6, 24, diag.WarmestSeasonTempC[i])+
				0.06*irrigation)*
				settings.PastureMultiplier -
				0.12*coldStress -
				0.08*waterlogging,
		)
		floodplain = clamp01(floodplain * settings.FloodplainMultiplier)
		irrigation = clamp01(irrigation * settings.IrrigationMultiplier)

		out.Diagnostics.ColdStress[i] = coldStress
		out.Diagnostics.DryStress[i] = dryStress
		out.Diagnostics.WaterloggingHazard[i] = waterlogging
		out.Diagnostics.ClimateSuitability[i] = climateSuit
		out.Diagnostics.SoilSuitability[i] = soilSuit
		out.Diagnostics.TerrainSuitability[i] = terrainSuit
		out.Diagnostics.IrrigationPotential[i] = irrigation
		out.Diagnostics.FloodplainPotential[i] = floodplain
		out.Diagnostics.CropPotential[i] = crop
		out.Diagnostics.PasturePotential[i] = pasture
		out.Types[i] = determineAgricultureType(crop, pasture, floodplain, coldStress, settings)
	}
	return out
}

func determineAgricultureType(
	crop, pasture, floodplain, coldStress float64,
	settings AgricultureProductivitySettings,
) AgricultureType {
	switch {
	case coldStress >= 0.82:
		return AgricultureUnsuitable
	case floodplain >= settings.FloodplainThreshold && crop >= settings.MixedFarmingThreshold:
		return AgricultureFloodplainCropland
	case crop >= settings.IntensiveThreshold:
		return AgricultureIntensiveCropland
	case crop >= settings.MixedFarmingThreshold && pasture >= 0.35:
		return AgricultureMixedFarming
	case pasture >= settings.PastoralThreshold && pasture >= crop-0.05:
		return AgriculturePastoral
	case crop >= settings.DryFarmingThreshold:
		return AgricultureDryFarming
	default:
		return AgricultureUnsuitable
	}
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
