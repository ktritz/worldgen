package climgen

import "fmt"

// SeasonalClimateSnapshot stores one seasonal temperature + precipitation state.
type SeasonalClimateSnapshot struct {
	SeasonIndex        int
	SeasonPhase        float64
	Label              string
	Temperature        []float64
	TemperatureCelsius []float64
	Insolation         []float64
	Albedo             []float64
	IceFraction        []float64
	BaseResponse       []float64
	BlendResponse      []float64
	SurfaceWind        []Vector3D
	Stats              TemperatureStats
	Precipitation      []float64 // cm for this seasonal slice
	Rainfall           []float64 // cm for this seasonal slice
	Snowfall           []float64 // cm for this seasonal slice
	Moisture           []float64
	PrecipDebug        *PrecipitationDebugFields
	LandMeanPrecip     float64
	LandMaxPrecip      float64
}

// SeasonalClimateResult bundles seasonal climate snapshots and annualized
// summary fields derived from them.
type SeasonalClimateResult struct {
	AnnualMean                               *TemperatureResult
	Snapshots                                []SeasonalClimateSnapshot
	TemperatureEquilibriumSnapshots          []SeasonalTemperatureSnapshot
	TemperatureReferenceEquilibriumSnapshots []SeasonalTemperatureSnapshot
	Currents                                 []Vector3D
	AnnualPrecipitation                      []float64
	WettestSeason                            []int
	DriestSeason                             []int
	PrecipitationRange                       []float64
}

// GenerateSeasonalClimate computes seasonal temperature snapshots, then drives
// seasonally shifted winds and per-season precipitation from those snapshots.
func GenerateSeasonalClimate(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	windSettings WindSettings,
	currents *OceanCurrentResult,
	tempSettings TemperatureSettings,
	precipSettings PrecipitationSettings,
	seasonal SeasonalTemperatureSettings,
) (*SeasonalClimateResult, error) {
	annualWind, err := GenerateWindField(vertices, elevation, seaLevelThreshold, adj, windSettings)
	if err != nil {
		return nil, fmt.Errorf("generate annual wind: %w", err)
	}

	seasonalTemps, err := GenerateSeasonalTemperature(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		annualWind,
		currents,
		tempSettings,
		seasonal,
	)
	if err != nil {
		return nil, err
	}

	snapshots := make([]SeasonalClimateSnapshot, len(seasonalTemps.Snapshots))
	annualPrecip := make([]float64, len(vertices))
	landInterior := ComputeSurfaceInteriorFraction(elevation, seaLevelThreshold, adj, 2200.0, true)
	landStorage := InitializeSeasonalLandStorage(elevation, seaLevelThreshold, landInterior)
	for cycle := 0; cycle < seasonal.NumCycles; cycle++ {
		for i, snapshot := range seasonalTemps.Snapshots {
			seasonSolar := tempSettings.Solar
			seasonSolar.SeasonPhase = snapshot.SeasonPhase
			seasonWind, err := GenerateSeasonalWindField(
				vertices,
				elevation,
				seaLevelThreshold,
				adj,
				windSettings,
				seasonSolar,
				snapshot.Temperature,
				seasonalTemps.AnnualMean.Temperature,
			)
			if err != nil {
				return nil, fmt.Errorf("generate seasonal wind %s: %w", snapshot.Label, err)
			}
			transportWind := BuildTemperatureTransportWindField(seasonWind, elevation, seaLevelThreshold)

			seasonPrecipSettings := DeriveSeasonalPrecipitationSettings(
				precipSettings,
				vertices,
				elevation,
				seaLevelThreshold,
				seasonSolar,
				snapshot.Temperature,
				seasonalTemps.AnnualMean.Temperature,
			)
			seasonPrecipSettings = ApplySeasonalLandStorageToSettings(
				seasonPrecipSettings,
				elevation,
				seaLevelThreshold,
				snapshot.Temperature,
				landStorage,
			)
			seasonPrecipSettings = ApplySeasonalDynamicPrecipitationForcing(
				seasonPrecipSettings,
				vertices,
				elevation,
				seaLevelThreshold,
				adj,
				transportWind,
				seasonSolar,
				snapshot.Temperature,
				seasonalTemps.AnnualMean.Temperature,
				landInterior,
			)
			precip := ComputePrecipitation(
				vertices,
				elevation,
				seaLevelThreshold,
				adj,
				transportWind,
				snapshot.Temperature,
				seasonPrecipSettings,
			)
			seasonalPrecip := ApplySeasonalPrecipitationResidual(
				vertices,
				elevation,
				seaLevelThreshold,
				adj,
				transportWind,
				seasonSolar,
				snapshot.Temperature,
				seasonalTemps.AnnualMean.Temperature,
				landInterior,
				precip.Precipitation,
			)
			precip.Precipitation = seasonalPrecip
			scale := 1.0 / float64(len(seasonalTemps.Snapshots))
			scaledPrecip, scaledMoisture := scaleSeasonalHydrology(precip, scale)
			scaledResult := &PrecipitationResult{
				Precipitation:        scaledPrecip,
				Moisture:             scaledMoisture,
				MarinePrecipitation:  make([]float64, len(scaledPrecip)),
				LandPrecipitation:    make([]float64, len(scaledPrecip)),
				FrontalPrecipitation: make([]float64, len(scaledPrecip)),
				Debug:                CloneScaledPrecipitationDebugFields(precip.Debug, scale),
				Rainfall:             make([]float64, len(scaledPrecip)),
				Snowfall:             make([]float64, len(scaledPrecip)),
			}
			scaleSlice(scaledResult.MarinePrecipitation, precip.MarinePrecipitation, scale)
			scaleSlice(scaledResult.LandPrecipitation, precip.LandPrecipitation, scale)
			scaleSlice(scaledResult.FrontalPrecipitation, precip.FrontalPrecipitation, scale)
			partitionPrecipitationPhase(scaledResult, elevation, seaLevelThreshold, snapshot.Temperature)
			landStorage = AdvanceSeasonalLandStorage(
				landStorage,
				elevation,
				seaLevelThreshold,
				snapshot.Temperature,
				scaledResult.Rainfall,
				scaledResult.Snowfall,
				landInterior,
			)

			if cycle != seasonal.NumCycles-1 {
				continue
			}

			landCells, meanPrecip, maxPrecip := GetPrecipitationStats(
				&PrecipitationResult{Precipitation: scaledPrecip, Moisture: scaledMoisture},
				elevation,
				seaLevelThreshold,
			)
			_ = landCells

			for j, p := range scaledPrecip {
				annualPrecip[j] += p
			}

			snapshots[i] = SeasonalClimateSnapshot{
				SeasonIndex:        snapshot.SeasonIndex,
				SeasonPhase:        snapshot.SeasonPhase,
				Label:              snapshot.Label,
				Temperature:        append([]float64(nil), snapshot.Temperature...),
				TemperatureCelsius: append([]float64(nil), snapshot.TemperatureCelsius...),
				Insolation:         append([]float64(nil), snapshot.Insolation...),
				Albedo:             append([]float64(nil), snapshot.Albedo...),
				IceFraction:        append([]float64(nil), snapshot.IceFraction...),
				BaseResponse:       append([]float64(nil), snapshot.BaseResponse...),
				BlendResponse:      append([]float64(nil), snapshot.BlendResponse...),
				SurfaceWind:        append([]Vector3D(nil), seasonWind.SurfaceWind...),
				Stats:              snapshot.Stats,
				Precipitation:      scaledResult.Precipitation,
				Rainfall:           append([]float64(nil), scaledResult.Rainfall...),
				Snowfall:           append([]float64(nil), scaledResult.Snowfall...),
				Moisture:           scaledResult.Moisture,
				PrecipDebug:        scaledResult.Debug,
				LandMeanPrecip:     meanPrecip,
				LandMaxPrecip:      maxPrecip,
			}
		}
	}

	wettest, driest, precipRange := summarizeSeasonalPrecipitation(snapshots)
	var copiedCurrents []Vector3D
	if currents != nil && len(currents.Currents) == len(vertices) {
		copiedCurrents = append([]Vector3D(nil), currents.Currents...)
	}
	return &SeasonalClimateResult{
		AnnualMean:                               seasonalTemps.AnnualMean,
		Snapshots:                                snapshots,
		TemperatureEquilibriumSnapshots:          seasonalTemps.EquilibriumSnapshots,
		TemperatureReferenceEquilibriumSnapshots: seasonalTemps.ReferenceEquilibriumSnapshots,
		Currents:                                 copiedCurrents,
		AnnualPrecipitation:                      annualPrecip,
		WettestSeason:                            wettest,
		DriestSeason:                             driest,
		PrecipitationRange:                       precipRange,
	}, nil
}

func scaleSeasonalHydrology(result *PrecipitationResult, scale float64) ([]float64, []float64) {
	precip := make([]float64, len(result.Precipitation))
	moisture := make([]float64, len(result.Moisture))
	for i := range result.Precipitation {
		precip[i] = result.Precipitation[i] * scale
		moisture[i] = result.Moisture[i] * scale
	}
	return precip, moisture
}

func summarizeSeasonalPrecipitation(
	snapshots []SeasonalClimateSnapshot,
) ([]int, []int, []float64) {
	if len(snapshots) == 0 {
		return nil, nil, nil
	}
	n := len(snapshots[0].Precipitation)
	wettest := make([]int, n)
	driest := make([]int, n)
	rng := make([]float64, n)

	for i := 0; i < n; i++ {
		minV := snapshots[0].Precipitation[i]
		maxV := minV
		minIdx := 0
		maxIdx := 0
		for s := 1; s < len(snapshots); s++ {
			v := snapshots[s].Precipitation[i]
			if v < minV {
				minV = v
				minIdx = s
			}
			if v > maxV {
				maxV = v
				maxIdx = s
			}
		}
		wettest[i] = maxIdx
		driest[i] = minIdx
		rng[i] = maxV - minV
	}

	return wettest, driest, rng
}
