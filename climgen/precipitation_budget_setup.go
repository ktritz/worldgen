package climgen

import "math"

func estimateClimateCellSizeKm(n int) float64 {
	earthRadiusKm := 6371.0
	return earthRadiusKm * math.Sqrt(4*math.Pi/float64(n))
}

// scaledPrecipIterations sizes the land-budget sweep. One iteration advects one
// cell, so this is a physical transport reach, not a convergence tolerance --
// instrumentation shows the budget is fully consumed at the baseline rather than
// converging early, so a short budget truncates the field.
//
// The previous form derived the count from a cell-size ratio and then floored it
// at precipMinIterations, and the floor swallowed the scaling: L6 got 18
// iterations where it needed 36 and L7 got 22 where it needed 72, collapsing
// inland reach from 4018 km at L5 to 2009 km at L6 and 1228 km at L7. That
// starved continental interiors, which is most of why the precipitation P10
// halved with refinement.
func scaledPrecipIterations(cellCount int) int {
	maxIterations := meshResolutionAdjustedSteps(precipMinIterations, cellCount)
	if maxIterations < precipMinIterations {
		maxIterations = precipMinIterations
	}
	if maxIterations > precipMaxIterations {
		maxIterations = precipMaxIterations
	}
	return maxIterations
}

func computeMoistureCapacity(temperature []float64) []float64 {
	capacity := make([]float64, len(temperature))
	for i, tempK := range temperature {
		tempC := tempK - 273.15
		capacity[i] = math.Pow(2, (tempC-18.0)/14.0)
		capacity[i] = Clamp(capacity[i], 0.04, 1.2)
	}
	return capacity
}

func computeOceanMoistureSource(isOcean []bool, temperature []float64, settings PrecipitationSettings) []float64 {
	source := make([]float64, len(isOcean))
	for i, ocean := range isOcean {
		if !ocean {
			continue
		}
		evap := settings.EvaporationRate
		if temperature != nil && settings.OceanEvaporationTempEffect > 0 {
			evap *= seasonalOceanEvaporationFactor(temperature[i], settings.OceanEvaporationTempEffect)
		}
		capacity := 1.0
		if temperature != nil && i < len(temperature) {
			capacity = computeMoistureCapacity([]float64{temperature[i]})[0]
		}
		source[i] = evap * (0.30 + 0.50*capacity)
	}
	return source
}

func computeLandMoistureSource(
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	temperature []float64,
	settings PrecipitationSettings,
) []float64 {
	source := make([]float64, len(vertices))
	if temperature == nil {
		return source
	}
	// The land budget adds this source once per relaxation iteration, and one
	// iteration advects moisture one cell. A fixed per-iteration increment is
	// therefore a per-step quantity, so a finer mesh injects it more times over
	// the same physical distance. Convert to a per-physical-step increment
	// (exact no-op at the L5 baseline).
	sourceFraction := precipitationPerStepFraction(precipLandSourceFraction, len(vertices))
	for i, v := range vertices {
		if elevation[i] < seaLevel {
			continue
		}
		absLat := math.Abs(getLatitudeDeg(v))
		tropicalConvergence := 1.0 - smoothRamp(precipTropicalConvergenceMinLatDeg, precipTropicalConvergenceMaxLatDeg, absLat)
		capacity := computeMoistureCapacity([]float64{temperature[i]})[0]
		tempC := temperature[i] - 273.15
		warmth := 0.55 + 0.45*smoothRamp(6.0, 30.0, tempC)
		scale := settings.LandSourceScale * localPrecipitationScale(settings.LandSourceLocalScale, i)
		source[i] = scale * sourceFraction * (0.18 + 0.82*tropicalConvergence) * capacity * warmth
	}
	return source
}
