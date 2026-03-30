package climgen

import "math"

const (
	seasonalLocalTropicalShiftMaxDeg = 8.0
	seasonalLocalTropicalShiftIters  = 6
	seasonalLocalTropicalShiftBlend  = 0.38
)

// computeSeasonalTropicalConvergenceLatitudeField builds a locally shifted
// tropical convergence latitude, rather than relying on one global thermal
// equator offset. This lets heated tropical continents pull the seasonal
// convergence zone poleward locally without forcing the whole planet to do the
// same thing.
func computeSeasonalTropicalConvergenceLatitudeField(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	solar SolarSettings,
	temperature []float64,
	annualMeanTemperature []float64,
) []float64 {
	baseShift := SeasonalThermalEquatorShiftDeg(solar)
	field := make([]float64, len(vertices))
	for i := range field {
		field[i] = baseShift
	}
	if len(temperature) != len(vertices) || len(annualMeanTemperature) != len(vertices) {
		return field
	}

	declinationDeg := -solar.AxialTilt * math.Cos(2.0*math.Pi*solar.SeasonPhase)
	summerHemisphere := 0.0
	switch {
	case declinationDeg > 0.1:
		summerHemisphere = 1.0
	case declinationDeg < -0.1:
		summerHemisphere = -1.0
	default:
		return field
	}

	landInterior := ComputeSurfaceInteriorFraction(elevation, seaLevelThreshold, adj, 2400.0, true)
	rawExtra := make([]float64, len(vertices))
	for i, v := range vertices {
		latDeg := getLatitudeDeg(v)
		absLat := math.Abs(latDeg)
		tropicalReach := 1.0 - smoothRamp(4.0, 30.0, absLat)
		if tropicalReach <= 0 {
			continue
		}

		tempAnom := Clamp((temperature[i]-annualMeanTemperature[i])/14.0, -1.0, 1.0)
		heat := math.Max(0, tempAnom)
		cool := math.Max(0, -tempAnom)
		interior := 0.0
		if i < len(landInterior) {
			interior = Clamp(landInterior[i], 0, 1)
		}

		if i < len(elevation) && elevation[i] >= seaLevelThreshold {
			rawExtra[i] = summerHemisphere *
				seasonalLocalTropicalShiftMaxDeg *
				tropicalReach *
				(0.80*heat*(0.25+0.75*interior) - 0.18*cool*(0.20+0.80*interior))
			continue
		}

		// Over ocean, keep the direct local anomaly weak; the global smoothing pass
		// below lets nearby continental heating bleed offshore and keeps the marine
		// convergence belt coherent.
		rawExtra[i] = summerHemisphere *
			seasonalLocalTropicalShiftMaxDeg *
			tropicalReach *
			0.10 *
			heat
	}

	smoothedExtra := SmoothScalarField(
		rawExtra,
		vertices,
		adj,
		seasonalLocalTropicalShiftIters,
		seasonalLocalTropicalShiftBlend,
	)
	for i := range field {
		field[i] = baseShift + Clamp(smoothedExtra[i], -seasonalLocalTropicalShiftMaxDeg, seasonalLocalTropicalShiftMaxDeg)
	}
	return field
}
