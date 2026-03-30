package climgen

import "math"

const defaultThermalEquatorShiftScale = 0.35

// SeasonalThermalEquatorShiftDeg converts seasonal solar declination into a
// practical migration of the atmospheric circulation belts.
func SeasonalThermalEquatorShiftDeg(solar SolarSettings) float64 {
	if solar.AxialTilt < 0.01 {
		return 0
	}
	declinationDeg := -solar.AxialTilt * math.Cos(2.0*math.Pi*solar.SeasonPhase)
	return defaultThermalEquatorShiftScale * declinationDeg
}

func effectiveCirculationLatitude(lat float64, settings CirculationSettings) float64 {
	return lat - settings.ThermalEquatorShiftDeg*math.Pi/180.0
}

// GenerateSeasonalWindField reuses the normal wind pipeline but migrates the
// circulation cells with the seasonal thermal equator.
func GenerateSeasonalWindField(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	settings WindSettings,
	solar SolarSettings,
	temperature []float64,
	annualMeanTemperature []float64,
) (*WindResult, error) {
	shifted := settings
	shifted.Circulation.ThermalEquatorShiftDeg = SeasonalThermalEquatorShiftDeg(solar)
	base, err := GenerateWindField(vertices, elevation, seaLevelThreshold, adj, shifted)
	if err != nil {
		return nil, err
	}
	return ApplySeasonalTropicalWindAdjustment(
		base,
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		solar,
		temperature,
		annualMeanTemperature,
	), nil
}
