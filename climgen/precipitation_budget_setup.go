package climgen

import "math"

func estimateClimateCellSizeKm(n int) float64 {
	earthRadiusKm := 6371.0
	return earthRadiusKm * math.Sqrt(4*math.Pi/float64(n))
}

func scaledPrecipIterations(avgCellSizeKm float64) int {
	maxIterations := int(precipMinIterations * precipBaseCellSizeKm / avgCellSizeKm)
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
		source[i] = scale * precipLandSourceFraction * (0.18 + 0.82*tropicalConvergence) * capacity * warmth
	}
	return source
}
