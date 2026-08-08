package climgen

import "math"

// BiomeDiagnostics stores the climate summaries that drive the seasonal biome
// classifier and the corresponding debug render outputs.
type BiomeDiagnostics struct {
	AnnualMeanTempC          []float64
	WarmestSeasonTempC       []float64
	ColdestSeasonTempC       []float64
	AnnualIceFraction        []float64
	WarmestSeasonIceFraction []float64
	ContinentalityC          []float64
	AnnualPrecipCm           []float64
	WettestSeasonPrecipCm    []float64
	DriestSeasonPrecipCm     []float64
	DrySeasonRatio           []float64
	WarmSeasonPrecipShare    []float64
	AridityThresholdCm       []float64
	AridityRatio             []float64
	DesertAffinity           []float64
	GrasslandAffinity        []float64
	ForestAffinity           []float64
	TropicalWetAffinity      []float64
	ColdAffinity             []float64
	IceAffinity              []float64
	TundraAffinity           []float64
	BorealAffinity           []float64
	WetlandAffinity          []float64
	AlpineAffinity           []float64
}

// HydrologyBiomeInputs carries just enough routed-hydrology context for biome
// overrides without coupling the biome package to terrain implementation types.
type HydrologyBiomeInputs struct {
	Runoff                   []float64
	ChannelStrength          []float64
	CellClass                []string
	WaterBodyLabel           []int
	WetlandClassSupport      []float64
	LakeClassSupport         []float64
	RiparianChannelSupport   []float64
	DepositionalClassSupport []float64
	// ChannelCorridorStrength is ChannelStrength widened to a constant physical
	// corridor. Consumers that ask "is this cell part of a river landscape"
	// cover area and must read this; consumers that follow the watercourse
	// itself keep reading the ChannelStrength centerline.
	ChannelCorridorStrength []float64
}

// SummarizeBiomeClimate derives seasonal climate summaries used by the biome
// classifier from the seasonal climate result.
func SummarizeBiomeClimate(result *SeasonalClimateResult) *BiomeDiagnostics {
	if result == nil || len(result.Snapshots) == 0 || len(result.Snapshots[0].TemperatureCelsius) == 0 {
		return &BiomeDiagnostics{}
	}

	n := len(result.Snapshots[0].TemperatureCelsius)
	diag := &BiomeDiagnostics{
		AnnualMeanTempC:          make([]float64, n),
		WarmestSeasonTempC:       make([]float64, n),
		ColdestSeasonTempC:       make([]float64, n),
		AnnualIceFraction:        make([]float64, n),
		WarmestSeasonIceFraction: make([]float64, n),
		ContinentalityC:          make([]float64, n),
		AnnualPrecipCm:           append([]float64(nil), result.AnnualPrecipitation...),
		WettestSeasonPrecipCm:    make([]float64, n),
		DriestSeasonPrecipCm:     make([]float64, n),
		DrySeasonRatio:           make([]float64, n),
		WarmSeasonPrecipShare:    make([]float64, n),
		AridityThresholdCm:       make([]float64, n),
		AridityRatio:             make([]float64, n),
		DesertAffinity:           make([]float64, n),
		GrasslandAffinity:        make([]float64, n),
		ForestAffinity:           make([]float64, n),
		TropicalWetAffinity:      make([]float64, n),
		ColdAffinity:             make([]float64, n),
		IceAffinity:              make([]float64, n),
		TundraAffinity:           make([]float64, n),
		BorealAffinity:           make([]float64, n),
		WetlandAffinity:          make([]float64, n),
		AlpineAffinity:           make([]float64, n),
	}
	if len(result.Snapshots) == 0 {
		return diag
	}

	type seasonRank struct {
		temp   float64
		precip float64
	}

	for i := 0; i < n; i++ {
		warmest := result.Snapshots[0].TemperatureCelsius[i]
		coldest := warmest
		warmestIce := 0.0
		wettest := result.Snapshots[0].Precipitation[i]
		driest := wettest
		ranks := make([]seasonRank, len(result.Snapshots))
		for s, snapshot := range result.Snapshots {
			temp := snapshot.TemperatureCelsius[i]
			precip := snapshot.Precipitation[i]
			diag.AnnualMeanTempC[i] += temp
			if len(snapshot.IceFraction) > i {
				diag.AnnualIceFraction[i] += snapshot.IceFraction[i]
			}
			if temp > warmest {
				warmest = temp
				if len(snapshot.IceFraction) > i {
					warmestIce = snapshot.IceFraction[i]
				} else {
					warmestIce = 0
				}
			}
			if temp < coldest {
				coldest = temp
			}
			if precip > wettest {
				wettest = precip
			}
			if precip < driest {
				driest = precip
			}
			ranks[s] = seasonRank{temp: temp, precip: precip}
		}
		diag.AnnualMeanTempC[i] /= float64(len(result.Snapshots))
		diag.AnnualIceFraction[i] /= float64(len(result.Snapshots))

		diag.WarmestSeasonTempC[i] = warmest
		diag.ColdestSeasonTempC[i] = coldest
		diag.WarmestSeasonIceFraction[i] = warmestIce
		diag.ContinentalityC[i] = warmest - coldest
		diag.WettestSeasonPrecipCm[i] = wettest
		diag.DriestSeasonPrecipCm[i] = driest
		diag.DrySeasonRatio[i] = driest / math.Max(wettest, 1e-6)

		// Warm-half precipitation share: use the two warmest seasonal slices.
		first := 0
		second := 0
		for s := 1; s < len(ranks); s++ {
			if ranks[s].temp > ranks[first].temp {
				second = first
				first = s
			} else if s != first && (second == first || ranks[s].temp > ranks[second].temp) {
				second = s
			}
		}
		if len(ranks) > 1 {
			warmHalf := ranks[first].precip
			if second != first {
				warmHalf += ranks[second].precip
			}
			diag.WarmSeasonPrecipShare[i] = warmHalf / math.Max(diag.AnnualPrecipCm[i], 1e-6)
		}

		// Köppen-style aridity threshold adapted to cm/year.
		seasonAdjCm := 14.0
		if diag.WarmSeasonPrecipShare[i] >= 0.70 {
			seasonAdjCm = 28.0
		} else if diag.WarmSeasonPrecipShare[i] < 0.30 {
			seasonAdjCm = 0.0
		}
		threshold := 2.0*math.Max(diag.AnnualMeanTempC[i], 0.0) + seasonAdjCm
		if threshold < 5.0 {
			threshold = 5.0
		}
		diag.AridityThresholdCm[i] = threshold
		diag.AridityRatio[i] = diag.AnnualPrecipCm[i] / threshold
	}

	return diag
}

// ClassifyBiomesSeasonal assigns biomes using seasonal thermal and moisture
// summaries derived from the climate solver.
func ClassifyBiomesSeasonal(
	climate *SeasonalClimateResult,
	elevation []float64,
	seaLevel float64,
) *BiomeResult {
	return ClassifyBiomesSeasonalWithHydrology(climate, elevation, seaLevel, nil)
}

// ClassifyBiomesSeasonalWithHydrology extends the seasonal climate classifier
// with routed-hydrology context so wetlands/floodplains can be mapped where
// climate and drainage both support them.
func ClassifyBiomesSeasonalWithHydrology(
	climate *SeasonalClimateResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
) *BiomeResult {
	diag := SummarizeBiomeClimate(climate)
	computeBiomeAffinities(diag, elevation, seaLevel, hydro)
	n := len(elevation)
	result := &BiomeResult{
		Biomes:      make([]Biome, n),
		Diagnostics: diag,
	}
	for i := 0; i < n; i++ {
		result.Biomes[i] = classifySeasonalCell(
			elevation[i],
			seaLevel,
			diag.AnnualMeanTempC[i],
			diag.WarmestSeasonTempC[i],
			diag.ColdestSeasonTempC[i],
			diag.ContinentalityC[i],
			diag.AnnualPrecipCm[i],
			diag.DriestSeasonPrecipCm[i],
			diag.DrySeasonRatio[i],
			diag.WarmSeasonPrecipShare[i],
			diag.AridityRatio[i],
			diag.AnnualIceFraction[i],
			diag.WarmestSeasonIceFraction[i],
			diag.WetlandAffinity[i],
		)
	}
	return result
}

func computeBiomeAffinities(diag *BiomeDiagnostics, elevation []float64, seaLevel float64, hydro *HydrologyBiomeInputs) {
	if diag == nil {
		return
	}
	n := len(elevation)
	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			continue
		}
		diag.DesertAffinity[i] = clamp01(
			0.75*smoothstep01(1.15, 0.35, diag.AridityRatio[i]) +
				0.25*smoothstep01(35, 0, diag.AnnualPrecipCm[i]),
		)
		diag.GrasslandAffinity[i] = clamp01(
			0.65*peak01(diag.AridityRatio[i], 0.9, 1.45, 2.2) +
				0.35*smoothstep01(20, 120, diag.AnnualPrecipCm[i]),
		)
		diag.ForestAffinity[i] = clamp01(
			smoothstep01(0.95, 2.2, diag.AridityRatio[i]) *
				smoothstep01(50, 160, diag.AnnualPrecipCm[i]),
		)
		diag.TropicalWetAffinity[i] = clamp01(
			smoothstep01(18, 24, diag.ColdestSeasonTempC[i]) *
				smoothstep01(100, 220, diag.AnnualPrecipCm[i]) *
				smoothstep01(2, 8, diag.DriestSeasonPrecipCm[i]),
		)
		diag.IceAffinity[i] = clamp01(
			(0.55*smoothstep01(3, -8, diag.WarmestSeasonTempC[i]) +
				0.20*smoothstep01(-8, -28, diag.AnnualMeanTempC[i]) +
				0.25*smoothstep01(0.20, 0.70, diag.WarmestSeasonIceFraction[i])) *
				smoothstep01(0.05, 0.55, diag.AnnualIceFraction[i]),
		)
		diag.TundraAffinity[i] = clamp01(
			peak01(diag.WarmestSeasonTempC[i], 0, 6, 12) *
				smoothstep01(-2, -20, diag.ColdestSeasonTempC[i]) *
				smoothstep01(0.65, 0.05, diag.WarmestSeasonIceFraction[i]),
		)
		diag.BorealAffinity[i] = clamp01(
			smoothstep01(7, 12, diag.WarmestSeasonTempC[i]) *
				smoothstep01(20, 6, diag.WarmestSeasonTempC[i]) *
				smoothstep01(-4, -18, diag.ColdestSeasonTempC[i]) *
				smoothstep01(25, 70, diag.AnnualPrecipCm[i]) *
				smoothstep01(0.45, 0.0, diag.WarmestSeasonIceFraction[i]),
		)
		wetlandRunoff := 0.0
		wetlandChannel := 0.0
		wetlandClass := 0.0
		if hydro != nil {
			if i < len(hydro.Runoff) {
				wetlandRunoff = smoothstep01(18, 110, hydro.Runoff[i])
			}
			if i < len(hydro.ChannelStrength) {
				wetlandChannel = smoothstep01(0.9, 2.6, hydrologyChannelCorridorStrength(hydro, i))
			}
			wetlandClass = hydrologyClassFactor(hydro, i)
		}
		diag.WetlandAffinity[i] = clamp01(
			(0.45*wetlandRunoff +
				0.30*wetlandChannel +
				0.25*wetlandClass) *
				smoothstep01(0.55, 1.8, diag.AridityRatio[i]) *
				smoothstep01(20, 70, diag.AnnualPrecipCm[i]) *
				smoothstep01(-6, 10, diag.WarmestSeasonTempC[i]) *
				smoothstep01(0.55, 0.10, diag.AnnualIceFraction[i]),
		)
		diag.ColdAffinity[i] = clamp01(math.Max(
			diag.IceAffinity[i],
			math.Max(diag.TundraAffinity[i], math.Max(diag.BorealAffinity[i], diag.WetlandAffinity[i]*0.4)),
		))
		diag.AlpineAffinity[i] = clamp01(
			smoothstep01(2200, 4000, elevation[i]) *
				smoothstep01(16, 8, diag.WarmestSeasonTempC[i]),
		)
	}
}

func classifySeasonalCell(
	elev, seaLevel float64,
	annualTempC, warmestTempC, coldestTempC, continentalityC float64,
	annualPrecipCm, driestSeasonCm, drySeasonRatio, warmSeasonShare, aridityRatio float64,
	annualIceFraction, warmestSeasonIceFraction float64,
	wetlandAffinity float64,
) Biome {
	if elev < seaLevel {
		return BiomeOcean
	}

	if elev > 3200 && warmestTempC < 10 {
		if warmestTempC < 1 {
			return BiomeIceCap
		}
		return BiomeAlpine
	}

	if warmestTempC < 0 && warmestSeasonIceFraction > 0.60 {
		return BiomeIceCap
	}
	if warmestTempC < 10 {
		if annualIceFraction > 0.65 && warmestSeasonIceFraction > 0.45 {
			return BiomeIceCap
		}
		if annualPrecipCm < 12 {
			return BiomeDesertCold
		}
		return BiomeTundra
	}

	if aridityRatio < 0.50 {
		if coldestTempC < 0 || annualTempC < 12 {
			return BiomeDesertCold
		}
		return BiomeDesertHot
	}
	if aridityRatio < 1.00 {
		return BiomeSemiArid
	}

	if wetlandAffinity >= 0.62 && annualPrecipCm >= 35 {
		return BiomeWetland
	}

	if coldestTempC >= 18 {
		if annualPrecipCm >= 180 && driestSeasonCm >= 6 {
			return BiomeTropicalRainforest
		}
		if annualPrecipCm >= 110 && (driestSeasonCm >= 3 || drySeasonRatio >= 0.18) {
			return BiomeTropicalSeasonalForest
		}
		return BiomeSavanna
	}

	summerDry := warmestTempC >= 18 &&
		warmSeasonShare < 0.35 &&
		driestSeasonCm < 4 &&
		drySeasonRatio < 0.35 &&
		annualPrecipCm >= 45

	if summerDry {
		return BiomeMediterranean
	}

	if coldestTempC <= -8 && warmestTempC < 20 && warmestSeasonIceFraction < 0.35 {
		return BiomeBorealForest
	}

	if annualPrecipCm >= 160 && driestSeasonCm >= 5 && continentalityC < 20 {
		return BiomeTemperateRainforest
	}

	if annualPrecipCm < 85 || driestSeasonCm < 3 || (continentalityC > 26 && annualPrecipCm < 110) {
		return BiomeTemperateGrassland
	}

	if elev > 2600 && warmestTempC < 14 {
		return BiomeAlpine
	}

	return BiomeTemperateForest
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func smoothstep01(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		if x >= edge1 {
			return 1
		}
		return 0
	}
	t := clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func peak01(x, left, peak, right float64) float64 {
	if x <= left || x >= right {
		return 0
	}
	if x == peak {
		return 1
	}
	if x < peak {
		return smoothstep01(left, peak, x)
	}
	return smoothstep01(right, peak, x)
}
