package climgen

import "math"

// SeasonalRunoffResult converts seasonal rain/snow fields into a routed-runoff
// input field for coarse hydrology. It intentionally stays simple and
// deterministic, while allowing runoff efficiency to vary with aridity,
// seasonality, warmth, and snowmelt timing.
type SeasonalRunoffResult struct {
	AnnualRunoff     []float64
	PeakSeasonRunoff []float64
	PeakSeasonIndex  []int
	SnowmeltFraction []float64
	RainRunoff       []float64
	MeltRunoff       []float64
	RunoffRatio      []float64
}

func ComputeSeasonalRunoff(
	climate *SeasonalClimateResult,
	elevation []float64,
) SeasonalRunoffResult {
	n := len(elevation)
	result := SeasonalRunoffResult{
		AnnualRunoff:     make([]float64, n),
		PeakSeasonRunoff: make([]float64, n),
		PeakSeasonIndex:  make([]int, n),
		SnowmeltFraction: make([]float64, n),
		RainRunoff:       make([]float64, n),
		MeltRunoff:       make([]float64, n),
		RunoffRatio:      make([]float64, n),
	}
	snowpack := make([]float64, n)
	annualPrecip := climate.AnnualPrecipitation
	annualMeanTemp := climate.AnnualMean.TemperatureCelsius
	wettestSeasonPrecip := make([]float64, n)
	driestSeasonPrecip := make([]float64, n)
	warmSeasonPrecipShare := make([]float64, n)
	annualRainCoeff := make([]float64, n)
	annualMeltCoeff := make([]float64, n)
	seasonalityStrength := make([]float64, n)
	for i := range driestSeasonPrecip {
		driestSeasonPrecip[i] = math.MaxFloat64
		result.PeakSeasonIndex[i] = -1
	}

	for i := range elevation {
		if elevation[i] < 0 {
			continue
		}
		warmestSeason := 0
		secondWarmestSeason := 0
		for s, snapshot := range climate.Snapshots {
			precip := snapshot.Precipitation[i]
			if precip > wettestSeasonPrecip[i] {
				wettestSeasonPrecip[i] = precip
			}
			if precip < driestSeasonPrecip[i] {
				driestSeasonPrecip[i] = precip
			}
			if s == 0 || snapshot.TemperatureCelsius[i] > climate.Snapshots[warmestSeason].TemperatureCelsius[i] {
				secondWarmestSeason = warmestSeason
				warmestSeason = s
			} else if s != warmestSeason && (secondWarmestSeason == warmestSeason || snapshot.TemperatureCelsius[i] > climate.Snapshots[secondWarmestSeason].TemperatureCelsius[i]) {
				secondWarmestSeason = s
			}
		}
		if len(climate.Snapshots) > 0 {
			warmHalf := climate.Snapshots[warmestSeason].Precipitation[i]
			if len(climate.Snapshots) > 1 && secondWarmestSeason != warmestSeason {
				warmHalf += climate.Snapshots[secondWarmestSeason].Precipitation[i]
			}
			warmSeasonPrecipShare[i] = warmHalf / math.Max(annualPrecip[i], 1e-6)
		}

		dryRatio := driestSeasonPrecip[i] / math.Max(wettestSeasonPrecip[i], 1e-6)
		seasonalityStrength[i] = 1 - clamp01Local(dryRatio)
		seasonAdjCm := 14.0
		if warmSeasonPrecipShare[i] >= 0.70 {
			seasonAdjCm = 28.0
		} else if warmSeasonPrecipShare[i] < 0.30 {
			seasonAdjCm = 0.0
		}
		threshold := 2.0*math.Max(annualMeanTemp[i], 0.0) + seasonAdjCm
		if threshold < 5.0 {
			threshold = 5.0
		}
		aridityRatio := annualPrecip[i] / threshold
		humidity := smoothStepLocal(0.30, 2.20, aridityRatio)
		warmth := smoothStepLocal(8, 28, annualMeanTemp[i])
		cold := smoothStepLocal(6, -12, annualMeanTemp[i])
		annualRainCoeff[i] = clamp01Local(
			0.08 +
				0.52*humidity +
				0.14*cold -
				0.12*warmth -
				0.10*seasonalityStrength[i],
		)
		if aridityRatio < 0.40 {
			annualRainCoeff[i] *= 0.85
		}
		if annualPrecip[i] < 20 {
			annualRainCoeff[i] *= 0.8
		}
		annualMeltCoeff[i] = clamp01Local(
			0.48 +
				0.25*humidity +
				0.20*cold -
				0.05*warmth,
		)
	}

	for seasonIdx, snapshot := range climate.Snapshots {
		for i := range elevation {
			if elevation[i] < 0 {
				continue
			}
			snowpack[i] += snapshot.Snowfall[i]

			tempC := snapshot.TemperatureCelsius[i]
			meanSeasonPrecip := annualPrecip[i] / math.Max(float64(len(climate.Snapshots)), 1)
			seasonWetness := snapshot.Precipitation[i] / math.Max(meanSeasonPrecip, 1e-6)
			wetPulse := 0.70 + 0.55*smoothStepLocal(0.70, 1.80, seasonWetness)
			dryPenalty := 1.0 - 0.28*seasonalityStrength[i]*smoothStepLocal(0.9, 0.25, seasonWetness)
			rainEff := annualRainCoeff[i] * wetPulse * dryPenalty
			rainEff *= 0.96 + 0.10*smoothStepLocal(-2, 12, tempC)
			rainEff = clamp01Local(rainEff)
			rainRunoff := snapshot.Rainfall[i] * rainEff

			meltFrac := 0.10 + 0.80*smoothStepLocal(-1, 8, tempC)
			if tempC < -6 {
				meltFrac = 0
			}
			melt := snowpack[i] * clamp01Local(meltFrac) * annualMeltCoeff[i]
			snowpack[i] -= melt

			seasonRunoff := rainRunoff + melt
			result.AnnualRunoff[i] += seasonRunoff
			result.RainRunoff[i] += rainRunoff
			result.MeltRunoff[i] += melt
			if seasonRunoff > result.PeakSeasonRunoff[i] {
				result.PeakSeasonRunoff[i] = seasonRunoff
				result.PeakSeasonIndex[i] = seasonIdx
			}
		}
	}

	for i := range elevation {
		if elevation[i] < 0 {
			continue
		}
		total := result.AnnualRunoff[i]
		if total > 0 {
			result.SnowmeltFraction[i] = result.MeltRunoff[i] / total
		}
		if i < len(climate.AnnualPrecipitation) && climate.AnnualPrecipitation[i] > 0 {
			result.RunoffRatio[i] = total / climate.AnnualPrecipitation[i]
		}
	}

	return result
}

func smoothStepLocal(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		if x >= edge1 {
			return 1
		}
		return 0
	}
	t := clamp01Local((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func clamp01Local(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
