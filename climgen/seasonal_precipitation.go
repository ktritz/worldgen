package climgen

import "math"

const (
	seasonalITCZWidthDeg        = 12.0
	seasonalITCZBoostScale      = 0.75
	seasonalDryBeltCenterDeg    = 24.0
	seasonalDryBeltHalfWidth    = 12.0
	seasonalDryBeltSuppression  = 0.18
	seasonalStormTrackCenterDeg = 55.0
	seasonalStormTrackHalfWidth = 13.0
	seasonalStormTrackBoost     = 0.22
	seasonalPolarDryStartDeg    = 62.0
	seasonalPolarDryEndDeg      = 82.0
	seasonalPolarDryScale       = 0.40
	seasonalMonsoonBoostScale   = 0.55
	seasonalWinterDryScale      = 0.22
)

// ApplySeasonalPrecipitationPattern strengthens seasonal rain-belt migration
// and summer-hemisphere onshore wetting for the seasonal climate path without
// changing the annual precipitation solver.
func ApplySeasonalPrecipitationPattern(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	solar SolarSettings,
	temperature []float64,
	annualMeanTemperature []float64,
	landInterior []float64,
	precipitation []float64,
) []float64 {
	return applySeasonalPrecipitationPatternScaled(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		solar,
		temperature,
		annualMeanTemperature,
		landInterior,
		precipitation,
		seasonalITCZBoostScale,
		seasonalDryBeltSuppression,
		seasonalStormTrackBoost,
		seasonalPolarDryScale,
		seasonalMonsoonBoostScale,
		seasonalWinterDryScale,
	)
}

func applySeasonalPrecipitationPatternScaled(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	solar SolarSettings,
	temperature []float64,
	annualMeanTemperature []float64,
	landInterior []float64,
	precipitation []float64,
	itczBoostScale float64,
	dryBeltSuppression float64,
	stormTrackBoost float64,
	polarDryScale float64,
	monsoonBoostScale float64,
	winterDryScale float64,
) []float64 {
	adjusted := append([]float64(nil), precipitation...)
	if len(adjusted) != len(vertices) {
		return adjusted
	}

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
		if elevation[i] < seaLevelThreshold {
			continue
		}

		latDeg := getLatitudeDeg(v)
		climateLat := latDeg - shiftDeg
		absClimateLat := math.Abs(climateLat)

		if absClimateLat < seasonalITCZWidthDeg {
			boost := 1.0 + itczBoostScale*(1.0-absClimateLat/seasonalITCZWidthDeg)
			adjusted[i] *= boost
		}

		distFromDryCore := math.Abs(absClimateLat - seasonalDryBeltCenterDeg)
		if distFromDryCore < seasonalDryBeltHalfWidth {
			suppression := 1.0 - dryBeltSuppression*(1.0-distFromDryCore/seasonalDryBeltHalfWidth)
			adjusted[i] *= suppression
		}

		adjusted[i] *= seasonalHumidityFactor(i, absClimateLat, temperature, annualMeanTemperature, landInterior)

		if summerHemisphere == 0 {
			continue
		}

		winterHemisphere := -summerHemisphere
		if latDeg*winterHemisphere > 0 {
			distFromStormCore := math.Abs(absClimateLat - seasonalStormTrackCenterDeg)
			if distFromStormCore < seasonalStormTrackHalfWidth {
				boost := 1.0 + stormTrackBoost*(1.0-distFromStormCore/seasonalStormTrackHalfWidth)
				adjusted[i] *= boost
			}

			polarDry := smoothRamp(seasonalPolarDryStartDeg, seasonalPolarDryEndDeg, absClimateLat)
			adjusted[i] *= 1.0 - polarDryScale*polarDry
		}

		onshore := coastalOnshoreScore(i, vertices, elevation, seaLevelThreshold, adj, wind)
		if onshore <= 0 {
			continue
		}

		absLat := math.Abs(latDeg)
		if absLat < 8 || absLat > 35 {
			continue
		}

		if latDeg*summerHemisphere > 0 {
			adjusted[i] *= 1.0 + monsoonBoostScale*onshore
		} else {
			adjusted[i] *= 1.0 - winterDryScale*onshore
		}
	}

	return adjusted
}

func seasonalHumidityFactor(
	i int,
	absClimateLat float64,
	temperature []float64,
	annualMeanTemperature []float64,
	landInterior []float64,
) float64 {
	if i < 0 || i >= len(temperature) {
		return 1
	}

	tempC := temperature[i] - 273.15
	highLat := smoothRamp(35.0, 75.0, absClimateLat)

	// Cold air carries and precipitates less moisture overall, especially in
	// high-latitude winters where the annual solver's baseline precipitation can
	// otherwise stay unrealistically wet.
	coldDry := 1.0 - smoothRamp(-20.0, 8.0, tempC)
	interior := 0.0
	if i < len(landInterior) {
		interior = Clamp(landInterior[i], 0, 1)
	}
	coldAridityWeight := 0.55 + 0.45*interior
	factor := 1.0 - 0.62*coldDry*highLat*coldAridityWeight

	// Absolute humidity drops rapidly in very cold air. Preserve some maritime
	// snowfall near coasts, but let deep polar interiors dry out much harder.
	coldCapacity := Clamp(math.Pow(2, (tempC-5.0)/18.0), 0.04, 1.0)
	factor *= 0.42 + (0.58-0.18*interior)*coldCapacity

	// Use seasonal temperature anomaly relative to annual mean to break the
	// spring/autumn symmetry. Warmer-than-annual seasons support more moisture;
	// colder-than-annual seasons suppress it, with the effect concentrated away
	// from the tropics where seasonality is actually strong.
	if i < len(annualMeanTemperature) {
		annualMeanC := annualMeanTemperature[i] - 273.15
		anomalyNorm := Clamp((tempC-annualMeanC)/18.0, -1.0, 1.0)
		factor *= 1.0 + 0.24*anomalyNorm*highLat
	}

	return Clamp(factor, 0.08, 1.35)
}

func coastalOnshoreScore(
	i int,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
) float64 {
	if i < 0 || i >= len(vertices) || i >= len(wind) || elevation[i] < seaLevelThreshold {
		return 0
	}
	speed := Length(wind[i])
	if speed < 1e-9 {
		return 0
	}
	windDir := Scale(wind[i], 1.0/speed)

	radius := resolutionAdjustedPrecipSteps(1, len(vertices))
	stepScale := precipitationPhysicalStepScale(len(vertices))
	type queueItem struct {
		idx   int
		depth int
	}
	queue := []queueItem{{idx: i, depth: 0}}
	visited := map[int]bool{i: true}
	sum := 0.0
	weightSum := 0.0
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if item.depth >= radius {
			continue
		}
		for _, k := range adj.GetNeighbors(item.idx) {
			if k < 0 || k >= len(vertices) || k >= len(elevation) || visited[k] {
				continue
			}
			visited[k] = true
			depth := item.depth + 1
			if elevation[k] < seaLevelThreshold {
				fromOcean := Normalize(Sub(vertices[i], vertices[k]))
				onshore := Dot(windDir, fromOcean)
				if onshore > 0 {
					distance := float64(depth) * stepScale
					weight := math.Exp(-1.25 * distance)
					sum += onshore * weight
					weightSum += weight
				}
			}
			if depth < radius {
				queue = append(queue, queueItem{idx: k, depth: depth})
			}
		}
	}
	if weightSum <= 0 {
		return 0
	}
	return Clamp(sum/weightSum, 0, 1)
}

func smoothRamp(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		if x >= edge1 {
			return 1
		}
		return 0
	}
	t := Clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}
