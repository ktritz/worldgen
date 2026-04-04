package climgen

type WildlifeType int

const (
	WildlifeOcean WildlifeType = iota
	WildlifeSparse
	WildlifeGrazingGame
	WildlifeForestGame
	WildlifeWetlandGame
	WildlifePelts
	WildlifeTimber
)

func WildlifeName(w WildlifeType) string {
	names := []string{
		"Ocean",
		"Sparse",
		"Grazing Game",
		"Forest Game",
		"Wetland Game",
		"Pelts",
		"Timber",
	}
	if int(w) < len(names) {
		return names[w]
	}
	return "Unknown"
}

type WildlifeDiagnostics struct {
	GamePotential        []float64
	GrazingPotential     []float64
	ForestGamePotential  []float64
	WetlandGamePotential []float64
	PeltPotential        []float64
	TimberPotential      []float64
}

type WildlifeResult struct {
	Types       []WildlifeType
	Diagnostics *WildlifeDiagnostics
}

func ClassifyWildlife(
	biomes *BiomeResult,
	vegetation *VegetationResult,
	soils *SoilResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	settings WildlifeProductivitySettings,
) *WildlifeResult {
	n := len(elevation)
	out := &WildlifeResult{
		Types: make([]WildlifeType, n),
		Diagnostics: &WildlifeDiagnostics{
			GamePotential:        make([]float64, n),
			GrazingPotential:     make([]float64, n),
			ForestGamePotential:  make([]float64, n),
			WetlandGamePotential: make([]float64, n),
			PeltPotential:        make([]float64, n),
			TimberPotential:      make([]float64, n),
		},
	}
	if biomes == nil || biomes.Diagnostics == nil || vegetation == nil || vegetation.Diagnostics == nil {
		return out
	}
	diag := biomes.Diagnostics
	vdiag := vegetation.Diagnostics
	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Types[i] = WildlifeOcean
			continue
		}
		runoff := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
		channel := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.ChannelStrength })
		soilFertility := 0.5
		if soils != nil && soils.Diagnostics != nil && i < len(soils.Diagnostics.Fertility) {
			soilFertility = soils.Diagnostics.Fertility[i]
		}

		grazing := clamp01(
			(0.45*vdiag.GrassCover[i] +
				0.20*diag.GrasslandAffinity[i] +
				0.15*peak01(diag.AridityRatio[i], 0.65, 1.25, 2.10) +
				0.10*soilFertility +
				0.10*smoothstep01(8, 24, diag.WarmestSeasonTempC[i])) *
				settings.GrazingMultiplier,
		)
		forestGame := clamp01(
			(0.45*vdiag.TreeCover[i] +
				0.18*diag.ForestAffinity[i] +
				0.12*diag.BorealAffinity[i] +
				0.10*soilFertility +
				0.10*smoothstep01(25, 140, diag.AnnualPrecipCm[i]) +
				0.05*smoothstep01(6, 22, diag.WarmestSeasonTempC[i])) *
				settings.ForestGameMultiplier,
		)
		wetlandGame := clamp01(
			(0.42*vdiag.WetlandCover[i] +
				0.22*diag.WetlandAffinity[i] +
				0.12*smoothstep01(10, 95, runoff) +
				0.12*smoothstep01(0.8, 2.2, channel) +
				0.12*soilFertility) *
				settings.WetlandGameMultiplier,
		)
		pelts := 0.0
		if settings.FurredAnimalsPresent {
			pelts = clamp01(
				(0.34*diag.BorealAffinity[i] +
					0.24*diag.TundraAffinity[i] +
					0.16*vdiag.TreeCover[i] +
					0.14*vdiag.ShrubCover[i] +
					0.12*peak01(diag.WarmestSeasonTempC[i], -2, 6, 14)) *
					settings.PeltMultiplier,
			)
		}
		timber := 0.0
		if settings.TimberPresent {
			timber = clamp01(
				(0.48*vdiag.TreeCover[i] +
					0.20*diag.ForestAffinity[i] +
					0.16*diag.TropicalWetAffinity[i] +
					0.08*soilFertility +
					0.08*smoothstep01(40, 180, diag.AnnualPrecipCm[i])) *
					settings.TimberMultiplier,
			)
		}
		game := clamp01(0.38*grazing + 0.34*forestGame + 0.28*wetlandGame)

		out.Diagnostics.GrazingPotential[i] = grazing
		out.Diagnostics.ForestGamePotential[i] = forestGame
		out.Diagnostics.WetlandGamePotential[i] = wetlandGame
		out.Diagnostics.PeltPotential[i] = pelts
		out.Diagnostics.TimberPotential[i] = timber
		out.Diagnostics.GamePotential[i] = game

		out.Types[i] = determineWildlifeType(grazing, forestGame, wetlandGame, pelts, timber, settings)
	}
	return out
}

type wildlifeCandidate struct {
	typ WildlifeType
	val float64
}

func determineWildlifeType(grazing, forestGame, wetlandGame, pelts, timber float64, settings WildlifeProductivitySettings) WildlifeType {
	candidates := []wildlifeCandidate{
		{WildlifeGrazingGame, clamp01(grazing + settings.GrazingPrimaryBias)},
		{WildlifeForestGame, clamp01(forestGame + settings.ForestGamePrimaryBias)},
		{WildlifeWetlandGame, clamp01(wetlandGame + settings.WetlandGamePrimaryBias)},
		{WildlifePelts, clamp01(pelts + settings.PeltPrimaryBias)},
		{WildlifeTimber, clamp01(timber + settings.TimberPrimaryBias)},
	}
	best := 0.0
	bestType := WildlifeSparse
	for _, c := range candidates {
		if c.val > best {
			best = c.val
			bestType = c.typ
		}
	}
	if best < 0.35 {
		return WildlifeSparse
	}
	return bestType
}
