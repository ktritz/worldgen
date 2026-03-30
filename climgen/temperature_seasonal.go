package climgen

import "fmt"

// SeasonalTemperatureSettings controls the seasonal overlay model that sits on
// top of the annual-mean climate solve.
type SeasonalTemperatureSettings struct {
	NumSeasons               int
	NumCycles                int
	LandResponse             float64
	OceanResponse            float64
	ThawResponseBoost        float64
	LandIceThawPersistence   float64
	LandIceFreezePersistence float64
	ContinentalityDistanceKm float64
	ReferenceEquilibrium     bool
	Verbose                  bool
}

// DefaultSeasonalTemperatureSettings returns practical defaults for seasonal
// climate snapshots without attempting full transient weather simulation.
func DefaultSeasonalTemperatureSettings() SeasonalTemperatureSettings {
	return SeasonalTemperatureSettings{
		NumSeasons:               8,
		NumCycles:                3,
		LandResponse:             0.85,
		OceanResponse:            0.30,
		ThawResponseBoost:        0.35,
		LandIceThawPersistence:   0.35,
		LandIceFreezePersistence: 0.80,
		ContinentalityDistanceKm: 1500.0,
		ReferenceEquilibrium:     false,
		Verbose:                  false,
	}
}

// Validate checks that seasonal settings are within reasonable bounds.
func (s SeasonalTemperatureSettings) Validate() error {
	if s.NumSeasons < 2 || s.NumSeasons > 24 {
		return fmt.Errorf("numSeasons must be in [2, 24], got %d", s.NumSeasons)
	}
	if s.NumCycles < 1 || s.NumCycles > 20 {
		return fmt.Errorf("numCycles must be in [1, 20], got %d", s.NumCycles)
	}
	if s.LandResponse < 0 || s.LandResponse > 1 {
		return fmt.Errorf("landResponse must be in [0, 1], got %f", s.LandResponse)
	}
	if s.OceanResponse < 0 || s.OceanResponse > 1 {
		return fmt.Errorf("oceanResponse must be in [0, 1], got %f", s.OceanResponse)
	}
	if s.ThawResponseBoost < 0 || s.ThawResponseBoost > 1 {
		return fmt.Errorf("thawResponseBoost must be in [0, 1], got %f", s.ThawResponseBoost)
	}
	if s.LandIceThawPersistence < 0 || s.LandIceThawPersistence > 1 {
		return fmt.Errorf("landIceThawPersistence must be in [0, 1], got %f", s.LandIceThawPersistence)
	}
	if s.LandIceFreezePersistence < 0 || s.LandIceFreezePersistence > 1 {
		return fmt.Errorf("landIceFreezePersistence must be in [0, 1], got %f", s.LandIceFreezePersistence)
	}
	if s.ContinentalityDistanceKm <= 0 {
		return fmt.Errorf("continentalityDistanceKm must be positive, got %f", s.ContinentalityDistanceKm)
	}
	return nil
}

// SeasonalTemperatureSnapshot stores one seasonal climate state.
type SeasonalTemperatureSnapshot struct {
	SeasonIndex        int
	SeasonPhase        float64
	Label              string
	Temperature        []float64
	TemperatureCelsius []float64
	Insolation         []float64
	Albedo             []float64
	AbsorbedSolar      []float64
	HeatTransport      []float64
	IceFraction        []float64
	BaseResponse       []float64
	BlendResponse      []float64
	Iterations         int
	FinalMaxDelta      float64
	Converged          bool
	Stats              TemperatureStats
}

// SeasonalTemperatureResult contains seasonal snapshots plus annual summary
// fields derived from them.
type SeasonalTemperatureResult struct {
	AnnualMean                    *TemperatureResult
	Snapshots                     []SeasonalTemperatureSnapshot
	EquilibriumSnapshots          []SeasonalTemperatureSnapshot
	ReferenceEquilibriumSnapshots []SeasonalTemperatureSnapshot
	SeasonalMin                   []float64
	SeasonalMax                   []float64
	SeasonalRange                 []float64
	WarmestSeason                 []int
	ColdestSeason                 []int
}

// GenerateSeasonalTemperature computes an annual-mean baseline and then a set
// of seasonal snapshots using seasonal insolation plus damped land/ocean
// response. This is an overlay model, not a full transient weather solver.
func GenerateSeasonalTemperature(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind *WindResult,
	currents *OceanCurrentResult,
	settings TemperatureSettings,
	seasonal SeasonalTemperatureSettings,
) (*SeasonalTemperatureResult, error) {
	if err := settings.Validate(); err != nil {
		return nil, fmt.Errorf("invalid temperature settings: %w", err)
	}
	if err := seasonal.Validate(); err != nil {
		return nil, fmt.Errorf("invalid seasonal settings: %w", err)
	}

	annualSettings := settings
	annualSettings.ApplyVerbose()
	if seasonal.Verbose {
		fmt.Printf("=== Seasonal Temperature Generation ===\n")
		fmt.Printf("  Seasons: %d, cycles: %d\n", seasonal.NumSeasons, seasonal.NumCycles)
		fmt.Printf("  Axial tilt: %.1f°\n", annualSettings.Solar.AxialTilt)
	}

	annualMean, err := GenerateTemperature(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		currents,
		annualSettings,
	)
	if err != nil {
		return nil, err
	}

	continentality := ComputeContinentality(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		seasonal.ContinentalityDistanceKm,
	)

	currentState := append([]float64(nil), annualMean.Temperature...)
	landIceState := computeSeasonalIceFraction(annualMean.Temperature, elevation, seaLevelThreshold)
	finalSnapshots := make([]SeasonalTemperatureSnapshot, seasonal.NumSeasons)
	equilibriumSnapshots := make([]SeasonalTemperatureSnapshot, seasonal.NumSeasons)
	var referenceEquilibriumSnapshots []SeasonalTemperatureSnapshot
	if seasonal.ReferenceEquilibrium {
		referenceEquilibriumSnapshots = make([]SeasonalTemperatureSnapshot, seasonal.NumSeasons)
	}
	var windVectors []Vector3D
	if wind != nil {
		windVectors = BuildTemperatureTransportWindField(wind, elevation, seaLevelThreshold)
	}
	var currentVectors []Vector3D
	if currents != nil {
		currentVectors = currents.Currents
	}

	for cycle := 0; cycle < seasonal.NumCycles; cycle++ {
		if seasonal.Verbose {
			fmt.Printf("  Seasonal cycle %d/%d\n", cycle+1, seasonal.NumCycles)
		}
		for seasonIdx := 0; seasonIdx < seasonal.NumSeasons; seasonIdx++ {
			phase := float64(seasonIdx) / float64(seasonal.NumSeasons)
			seasonSettings := settings
			seasonSettings.Verbose = false
			seasonSettings.Solar.SeasonPhase = phase

			insolation := ComputeSeasonalInsolation(vertices, seasonSettings.Solar)
			equilibrium, err := generateTemperatureForInsolation(
				vertices,
				elevation,
				seaLevelThreshold,
				adj,
				wind,
				currents,
				seasonSettings,
				insolation,
				annualMean.Temperature,
			)
			if err != nil {
				return nil, err
			}
			equilibriumIce := computeSeasonalIceFraction(equilibrium.Temperature, elevation, seaLevelThreshold)

			var referenceEquilibrium *TemperatureResult
			if seasonal.ReferenceEquilibrium && cycle == seasonal.NumCycles-1 {
				referenceEquilibrium, err = generateTemperatureForInsolation(
					vertices,
					elevation,
					seaLevelThreshold,
					adj,
					wind,
					currents,
					seasonSettings,
					insolation,
					currentState,
				)
				if err != nil {
					return nil, err
				}
			}

			blended, blendResponse, baseResponse := blendSeasonalTemperature(
				currentState,
				equilibrium.Temperature,
				elevation,
				seaLevelThreshold,
				continentality,
				seasonal,
				landIceState,
				equilibriumIce,
			)
			currentState = blended
			landIceState = updateSeasonalLandIceState(
				landIceState,
				equilibriumIce,
				elevation,
				seaLevelThreshold,
				seasonal,
			)

			if cycle == seasonal.NumCycles-1 {
				equilibriumSnapshots[seasonIdx] = buildSeasonalSnapshot(
					seasonIdx,
					phase,
					seasonLabel(seasonIdx, seasonal.NumSeasons),
					equilibrium,
					vertices,
					elevation,
					seaLevelThreshold,
					adj,
					windVectors,
					currentVectors,
					settings.Transport,
				)
				if referenceEquilibrium != nil {
					referenceEquilibriumSnapshots[seasonIdx] = buildSeasonalSnapshot(
						seasonIdx,
						phase,
						seasonLabel(seasonIdx, seasonal.NumSeasons),
						referenceEquilibrium,
						vertices,
						elevation,
						seaLevelThreshold,
						adj,
						windVectors,
						currentVectors,
						settings.Transport,
					)
				}
				snapshot := buildSeasonalSnapshot(
					seasonIdx,
					phase,
					seasonLabel(seasonIdx, seasonal.NumSeasons),
					&TemperatureResult{
						Temperature:        append([]float64(nil), blended...),
						TemperatureCelsius: kelvinToCelsius(blended),
						Insolation:         append([]float64(nil), insolation...),
					},
					vertices,
					elevation,
					seaLevelThreshold,
					adj,
					windVectors,
					currentVectors,
					settings.Transport,
				)
				snapshot.BaseResponse = baseResponse
				snapshot.BlendResponse = blendResponse
				snapshot.IceFraction = append([]float64(nil), landIceState...)
				snapshot.Albedo = computeSeasonalAlbedoFromIceState(landIceState, elevation, seaLevelThreshold)
				finalSnapshots[seasonIdx] = snapshot
			}
		}
	}

	seasonalMin, seasonalMax, seasonalRange, warmestSeason, coldestSeason := summarizeSeasonalSnapshots(finalSnapshots)
	return &SeasonalTemperatureResult{
		AnnualMean:                    annualMean,
		Snapshots:                     finalSnapshots,
		EquilibriumSnapshots:          equilibriumSnapshots,
		ReferenceEquilibriumSnapshots: referenceEquilibriumSnapshots,
		SeasonalMin:                   seasonalMin,
		SeasonalMax:                   seasonalMax,
		SeasonalRange:                 seasonalRange,
		WarmestSeason:                 warmestSeason,
		ColdestSeason:                 coldestSeason,
	}, nil
}

func blendSeasonalTemperature(
	previous []float64,
	equilibrium []float64,
	elevation []float64,
	seaLevelThreshold float64,
	continentality []float64,
	settings SeasonalTemperatureSettings,
	landIceState []float64,
	equilibriumIce []float64,
) ([]float64, []float64, []float64) {
	blended := make([]float64, len(equilibrium))
	blendResponse := make([]float64, len(equilibrium))
	baseResponse := make([]float64, len(equilibrium))
	for i := range equilibrium {
		response := seasonalResponseFactor(
			elevation[i],
			seaLevelThreshold,
			continentality[i],
			settings,
		)
		baseResponse[i] = response
		response = seasonalBlendResponse(
			response,
			previous[i],
			equilibrium[i],
			elevation[i],
			seaLevelThreshold,
			continentality[i],
			settings,
			landIceState[i],
			equilibriumIce[i],
		)
		blendResponse[i] = response
		blended[i] = previous[i] + response*(equilibrium[i]-previous[i])
	}
	return blended, blendResponse, baseResponse
}

func seasonalBlendResponse(
	baseResponse float64,
	previous float64,
	equilibrium float64,
	elevation float64,
	seaLevelThreshold float64,
	continentality float64,
	settings SeasonalTemperatureSettings,
	currentIce float64,
	equilibriumIce float64,
) float64 {
	response := baseResponse
	if elevation < seaLevelThreshold || equilibrium <= previous || settings.ThawResponseBoost <= 0 {
		return response
	}
	if continentality < 0 {
		continentality = 0
	} else if continentality > 1 {
		continentality = 1
	}
	if previous >= FreezingPoint && equilibrium < FreezingPoint+2.0 {
		return response
	}

	frozenDeficit := clampUnitSeasonal((FreezingPoint - previous) / 20.0)
	thawExcess := clampUnitSeasonal((equilibrium - FreezingPoint + 4.0) / 16.0)
	warmingSpan := clampUnitSeasonal((equilibrium - previous) / 24.0)
	iceRelease := clampUnitSeasonal(currentIce - equilibriumIce)
	thawSignal := seasonalMaxFloat(frozenDeficit, thawExcess) * warmingSpan * continentality
	thawSignal = seasonalMaxFloat(thawSignal, iceRelease*warmingSpan*continentality)
	response += settings.ThawResponseBoost * thawSignal
	if response > 1 {
		response = 1
	}
	return response
}

func seasonalResponseFactor(
	elevation float64,
	seaLevelThreshold float64,
	continentality float64,
	settings SeasonalTemperatureSettings,
) float64 {
	if elevation < seaLevelThreshold {
		return settings.OceanResponse
	}
	if continentality < 0 {
		continentality = 0
	} else if continentality > 1 {
		continentality = 1
	}
	return settings.OceanResponse + continentality*(settings.LandResponse-settings.OceanResponse)
}

func buildSeasonalSnapshot(
	seasonIdx int,
	phase float64,
	label string,
	result *TemperatureResult,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	windVectors []Vector3D,
	currentVectors []Vector3D,
	transport TransportSettings,
) SeasonalTemperatureSnapshot {
	tempK := append([]float64(nil), result.Temperature...)
	tempC := append([]float64(nil), result.TemperatureCelsius...)
	insolation := append([]float64(nil), result.Insolation...)
	albedo := result.Albedo
	if len(albedo) != len(tempK) {
		albedo = ComputeAlbedo(tempK, elevation, seaLevelThreshold, true)
	}
	absorbedSolar := result.AbsorbedSolar
	if len(absorbedSolar) != len(tempK) {
		absorbedSolar = ComputeAbsorbedSolar(insolation, albedo)
	}
	heatTransport := result.HeatTransport
	if len(heatTransport) != len(tempK) {
		heatTransport = ComputeTotalHeatTransport(
			tempK,
			vertices,
			elevation,
			seaLevelThreshold,
			adj,
			windVectors,
			currentVectors,
			transport,
		)
	}
	iceFraction := computeSeasonalIceFraction(tempK, elevation, seaLevelThreshold)
	return SeasonalTemperatureSnapshot{
		SeasonIndex:        seasonIdx,
		SeasonPhase:        phase,
		Label:              label,
		Temperature:        tempK,
		TemperatureCelsius: tempC,
		Insolation:         insolation,
		Albedo:             append([]float64(nil), albedo...),
		AbsorbedSolar:      append([]float64(nil), absorbedSolar...),
		HeatTransport:      append([]float64(nil), heatTransport...),
		IceFraction:        iceFraction,
		Iterations:         result.Iterations,
		FinalMaxDelta:      result.FinalMaxDelta,
		Converged:          result.Converged,
		Stats:              (&TemperatureResult{Temperature: tempK, TemperatureCelsius: tempC}).ComputeStats(elevation, seaLevelThreshold),
	}
}

func kelvinToCelsius(temperature []float64) []float64 {
	tempC := make([]float64, len(temperature))
	for i, t := range temperature {
		tempC[i] = t - FreezingPoint
	}
	return tempC
}

func computeSeasonalIceFraction(
	temperature []float64,
	elevation []float64,
	seaLevelThreshold float64,
) []float64 {
	frac := make([]float64, len(temperature))
	for i, t := range temperature {
		if elevation[i] < seaLevelThreshold {
			if t <= 271.15 {
				frac[i] = 1.0
			}
			continue
		}
		if t <= FreezingPoint {
			frac[i] = 1.0
		}
	}
	return frac
}

func updateSeasonalLandIceState(
	current []float64,
	equilibrium []float64,
	elevation []float64,
	seaLevelThreshold float64,
	settings SeasonalTemperatureSettings,
) []float64 {
	next := make([]float64, len(current))
	for i := range current {
		if elevation[i] < seaLevelThreshold {
			next[i] = equilibrium[i]
			continue
		}
		persistence := settings.LandIceFreezePersistence
		if equilibrium[i] < current[i] {
			persistence = settings.LandIceThawPersistence
		}
		next[i] = persistence*current[i] + (1.0-persistence)*equilibrium[i]
	}
	return next
}

func computeSeasonalAlbedoFromIceState(
	iceState []float64,
	elevation []float64,
	seaLevelThreshold float64,
) []float64 {
	albedo := make([]float64, len(iceState))
	for i, iceFrac := range iceState {
		base := AlbedoLand
		iceAlbedo := AlbedoIce
		if elevation[i] < seaLevelThreshold {
			base = AlbedoWater
			iceAlbedo = 0.35
		}
		albedo[i] = iceFrac*iceAlbedo + (1.0-iceFrac)*base
	}
	return albedo
}

func clampUnitSeasonal(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func seasonalMaxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func summarizeSeasonalSnapshots(
	snapshots []SeasonalTemperatureSnapshot,
) ([]float64, []float64, []float64, []int, []int) {
	if len(snapshots) == 0 {
		return nil, nil, nil, nil, nil
	}

	n := len(snapshots[0].Temperature)
	seasonalMin := make([]float64, n)
	seasonalMax := make([]float64, n)
	seasonalRange := make([]float64, n)
	warmestSeason := make([]int, n)
	coldestSeason := make([]int, n)

	copy(seasonalMin, snapshots[0].Temperature)
	copy(seasonalMax, snapshots[0].Temperature)

	for seasonIdx := 1; seasonIdx < len(snapshots); seasonIdx++ {
		for i, t := range snapshots[seasonIdx].Temperature {
			if t < seasonalMin[i] {
				seasonalMin[i] = t
				coldestSeason[i] = seasonIdx
			}
			if t > seasonalMax[i] {
				seasonalMax[i] = t
				warmestSeason[i] = seasonIdx
			}
		}
	}

	for i := range seasonalRange {
		seasonalRange[i] = seasonalMax[i] - seasonalMin[i]
	}

	return seasonalMin, seasonalMax, seasonalRange, warmestSeason, coldestSeason
}

func seasonLabel(index, total int) string {
	if total == 4 {
		labels := []string{"NH Winter", "NH Spring", "NH Summer", "NH Autumn"}
		return labels[index%len(labels)]
	}
	if total == 8 {
		labels := []string{
			"NH Winter",
			"Late Winter",
			"NH Spring",
			"Late Spring",
			"NH Summer",
			"Late Summer",
			"NH Autumn",
			"Late Autumn",
		}
		return labels[index%len(labels)]
	}
	return fmt.Sprintf("Season %d", index+1)
}
