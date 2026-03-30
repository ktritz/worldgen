package climgen

import "math"

const (
	seasonalDynamicStormCenterDeg    = 50.0
	seasonalDynamicStormHalfWidthDeg = 16.0
	seasonalDynamicStormSpreadBlend  = 0.42
	seasonalDynamicStormSpreadIters  = 4
	seasonalDynamicStormMemoryBlend  = 0.52
	seasonalDynamicStormMemoryIters  = 3
	seasonalDynamicStormMemoryCarry  = 0.85

	seasonalResidualStormBandBoost = 0.18
)

func computeSeasonalFrontalExposureField(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	solar SolarSettings,
) []float64 {
	field := make([]float64, len(vertices))
	for i := range vertices {
		field[i] = seasonalFrontalExposureAt(i, vertices, elevation, seaLevelThreshold, adj, wind, solar)
	}
	return smoothSeasonalLandField(field, elevation, seaLevelThreshold, adj, 0.35, 2)
}

func computeSeasonalStormMoisturePotentialField(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	solar SolarSettings,
	landInterior []float64,
	frontalExposureField []float64,
) []float64 {
	raw := make([]float64, len(vertices))
	shiftDeg := SeasonalThermalEquatorShiftDeg(solar)
	declinationDeg := -solar.AxialTilt * math.Cos(2.0*math.Pi*solar.SeasonPhase)
	summerHemisphere := 0.0
	switch {
	case declinationDeg > 0.1:
		summerHemisphere = 1.0
	case declinationDeg < -0.1:
		summerHemisphere = -1.0
	}

	for i, v := range vertices {
		if i >= len(elevation) || elevation[i] < seaLevelThreshold {
			continue
		}
		latDeg := getLatitudeDeg(v)
		climateLat := latDeg - shiftDeg
		stormTrack := seasonalStormTrackWeight(latDeg, climateLat, summerHemisphere)
		if stormTrack <= 0 {
			continue
		}
		interior := 0.0
		if i < len(landInterior) {
			interior = Clamp(landInterior[i], 0, 1)
		}
		frontalExposure := 0.0
		if i < len(frontalExposureField) {
			frontalExposure = frontalExposureField[i]
		}
		onshore := coastalOnshoreScore(i, vertices, elevation, seaLevelThreshold, adj, wind)
		raw[i] = Clamp(
			0.55*frontalExposure+
				0.35*stormTrack*(0.20+0.80*interior)+
				0.10*onshore*(0.25+0.75*stormTrack)*(1.0-0.55*interior),
			0,
			1,
		)
	}

	smoothed := smoothSeasonalLandField(raw, elevation, seaLevelThreshold, adj, seasonalDynamicStormSpreadBlend, seasonalDynamicStormSpreadIters)
	memory := computeSeasonalStormMemoryField(
		smoothed,
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		landInterior,
	)
	out := make([]float64, len(raw))
	for i := range raw {
		interior := 0.0
		if i < len(landInterior) {
			interior = Clamp(landInterior[i], 0, 1)
		}
		out[i] = Clamp(
			0.28*raw[i]+
				0.52*smoothed[i]*(0.30+0.70*interior)+
				0.62*memory[i]*(0.25+0.75*interior),
			0,
			1,
		)
	}
	return out
}

// computeSeasonalStormBandSupportField builds a broader low-frequency winter
// storm-band support field over land. It is meant to widen inland frontal
// precipitation support without replacing the physical moisture transport.
func computeSeasonalStormBandSupportField(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	solar SolarSettings,
	landInterior []float64,
) []float64 {
	frontalExposure := computeSeasonalFrontalExposureField(vertices, elevation, seaLevelThreshold, adj, wind, solar)
	parent, strength := computeStrongestUpwindGraph(vertices, adj, wind)
	upwindLandSteps := computeUpwindLandStepCounts(parent, strength, elevation, seaLevelThreshold, precipInlandTransportSteps+2)

	raw := make([]float64, len(vertices))
	shiftDeg := SeasonalThermalEquatorShiftDeg(solar)
	declinationDeg := -solar.AxialTilt * math.Cos(2.0*math.Pi*solar.SeasonPhase)
	summerHemisphere := 0.0
	switch {
	case declinationDeg > 0.1:
		summerHemisphere = 1.0
	case declinationDeg < -0.1:
		summerHemisphere = -1.0
	default:
		return raw
	}

	for i, v := range vertices {
		if i >= len(elevation) || elevation[i] < seaLevelThreshold {
			continue
		}
		latDeg := getLatitudeDeg(v)
		climateLat := latDeg - shiftDeg
		stormTrack := seasonalStormTrackWeight(latDeg, climateLat, summerHemisphere)
		if stormTrack <= 0 {
			continue
		}

		interior := 0.0
		if i < len(landInterior) {
			interior = Clamp(landInterior[i], 0, 1)
		}
		travel := 0.0
		if i < len(upwindLandSteps) && upwindLandSteps[i] >= 0 {
			travel = Clamp(float64(upwindLandSteps[i])/float64(precipInlandTransportSteps+2), 0, 1)
		}
		corridor := transportCorridorWeight(travel)
		onshore := coastalOnshoreScore(i, vertices, elevation, seaLevelThreshold, adj, wind)
		coastalImmediate := Clamp(onshore*(1.0-travel), 0, 1)
		inlandCarry := Clamp(corridor*(0.35+0.65*interior)+travel*(0.20+0.80*interior), 0, 1)
		broadInterior := stormTrack * (0.30 + 0.70*(0.25+0.75*interior))

		raw[i] = Clamp(
			0.22*stormTrack+
				0.26*broadInterior+
				0.32*frontalExposure[i]*(0.20+0.80*interior)+
				0.22*inlandCarry*(0.25+0.75*stormTrack)+
				0.10*onshore*(0.15+0.85*stormTrack)*(0.20+0.80*interior)-
				0.12*coastalImmediate*(0.70+0.30*stormTrack),
			0,
			1,
		)
	}

	smoothed := smoothSeasonalLandField(raw, elevation, seaLevelThreshold, adj, 0.50, 6)
	memory := computeSeasonalStormMemoryField(
		smoothed,
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		landInterior,
	)
	out := make([]float64, len(raw))
	for i := range raw {
		interior := 0.0
		if i < len(landInterior) {
			interior = Clamp(landInterior[i], 0, 1)
		}
		out[i] = Clamp(
			0.14*raw[i]+
				0.46*smoothed[i]*(0.25+0.75*interior)+
				0.64*memory[i]*(0.20+0.80*interior),
			0,
			1,
		)
	}
	return out
}

func computeSeasonalStormMemoryField(
	field []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	landInterior []float64,
) []float64 {
	if len(field) == 0 {
		return nil
	}
	current := append([]float64(nil), field...)
	for iter := 0; iter < seasonalDynamicStormMemoryIters; iter++ {
		next := append([]float64(nil), current...)
		for i := range current {
			if i >= len(elevation) || elevation[i] < seaLevelThreshold {
				continue
			}
			upwindMean, ok := seasonalUpwindPotentialMean(i, current, vertices, elevation, seaLevelThreshold, adj, wind)
			if !ok {
				continue
			}
			interior := 0.0
			if i < len(landInterior) {
				interior = Clamp(landInterior[i], 0, 1)
			}
			carry := seasonalDynamicStormMemoryCarry * (0.30 + 0.70*interior)
			next[i] = Clamp(
				(1.0-seasonalDynamicStormMemoryBlend)*current[i]+
					seasonalDynamicStormMemoryBlend*(0.35*current[i]+carry*upwindMean),
				0,
				1,
			)
		}
		current = next
	}
	return current
}

func seasonalUpwindPotentialMean(
	i int,
	field []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
) (float64, bool) {
	if i < 0 || i >= len(vertices) || i >= len(wind) {
		return 0, false
	}
	windVec := wind[i]
	windSpeed := Length(windVec)
	if windSpeed < 1e-9 {
		return 0, false
	}
	windDir := Scale(windVec, 1.0/windSpeed)
	sum := 0.0
	weightSum := 0.0
	for _, k := range adj.GetNeighbors(i) {
		if k < 0 || k >= len(field) || k >= len(vertices) || k >= len(elevation) || elevation[k] < seaLevelThreshold {
			continue
		}
		fromNeighbor := Normalize(Sub(vertices[i], vertices[k]))
		upwind := Dot(windDir, fromNeighbor)
		if upwind <= 0.05 {
			continue
		}
		weight := upwind * upwind
		sum += field[k] * weight
		weightSum += weight
	}
	if weightSum <= 1e-9 {
		return 0, false
	}
	return sum / weightSum, true
}

func seasonalStormTrackWeight(latDeg float64, climateLat float64, summerHemisphere float64) float64 {
	if summerHemisphere == 0 || latDeg*(-summerHemisphere) <= 0 {
		return 0
	}
	absClimateLat := math.Abs(climateLat)
	distFromStormCore := math.Abs(absClimateLat - seasonalDynamicStormCenterDeg)
	if distFromStormCore >= seasonalDynamicStormHalfWidthDeg {
		return 0
	}
	return 1.0 - distFromStormCore/seasonalDynamicStormHalfWidthDeg
}

func smoothSeasonalLandField(
	field []float64,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	neighborBlend float64,
	iterations int,
) []float64 {
	if len(field) == 0 {
		return nil
	}
	current := append([]float64(nil), field...)
	blend := Clamp(neighborBlend, 0, 0.6)
	for iter := 0; iter < iterations; iter++ {
		next := append([]float64(nil), current...)
		for i := range current {
			if i >= len(elevation) || elevation[i] < seaLevelThreshold {
				continue
			}
			sum := 0.0
			count := 0
			for _, k := range adj.GetNeighbors(i) {
				if k < 0 || k >= len(current) || k >= len(elevation) || elevation[k] < seaLevelThreshold {
					continue
				}
				sum += current[k]
				count++
			}
			if count == 0 {
				continue
			}
			neighborMean := sum / float64(count)
			next[i] = Clamp((1.0-blend)*current[i]+blend*neighborMean, 0, 1)
		}
		current = next
	}
	return current
}

func seasonalFrontalExposureAt(
	i int,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	solar SolarSettings,
) float64 {
	if i < 0 || i >= len(vertices) || i >= len(elevation) || elevation[i] < seaLevelThreshold {
		return 0
	}
	declinationDeg := -solar.AxialTilt * math.Cos(2.0*math.Pi*solar.SeasonPhase)
	summerHemisphere := 0.0
	switch {
	case declinationDeg > 0.1:
		summerHemisphere = 1.0
	case declinationDeg < -0.1:
		summerHemisphere = -1.0
	default:
		return 0
	}
	lat := getLatitudeDeg(vertices[i])
	if lat*(-summerHemisphere) <= 0 {
		return 0
	}

	current := i
	oceanHits := 0.0
	stormHits := 0.0
	for step := 0; step < precipFetchMaxSteps; step++ {
		next, upwindness := strongestUpwindNeighbor(current, vertices, adj, wind)
		if next < 0 || upwindness <= 0.05 {
			break
		}
		latDeg := getLatitudeDeg(vertices[next])
		climateLat := latDeg - SeasonalThermalEquatorShiftDeg(solar)
		stormWeight := seasonalStormTrackWeight(latDeg, climateLat, summerHemisphere)
		weight := upwindness / float64(step+1)
		if elevation[next] < seaLevelThreshold {
			oceanHits += weight
		}
		stormHits += weight * stormWeight
		current = next
	}
	return Clamp(0.55*Clamp(oceanHits, 0, 1)+0.75*Clamp(stormHits, 0, 1), 0, 1)
}
