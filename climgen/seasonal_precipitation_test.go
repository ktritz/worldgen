package climgen

import (
	"math"
	"testing"
)

func TestApplySeasonalPrecipitationPatternShiftsRainBelt(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(8, 0),
		seasonalLatLonVertex(-8, 0),
	}
	elevation := []float64{100, 100}
	precip := []float64{100, 100}

	summer := ApplySeasonalPrecipitationPattern(
		vertices,
		elevation,
		0.0,
		&FlatAdjacency{Offsets: []int{0, 0, 0}},
		nil,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5},
		nil,
		nil,
		nil,
		precip,
	)
	if summer[0] <= summer[1] {
		t.Fatalf("expected NH summer rain belt to favor northern tropics: %.2f <= %.2f", summer[0], summer[1])
	}

	winter := ApplySeasonalPrecipitationPattern(
		vertices,
		elevation,
		0.0,
		&FlatAdjacency{Offsets: []int{0, 0, 0}},
		nil,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.0},
		nil,
		nil,
		nil,
		precip,
	)
	if winter[1] <= winter[0] {
		t.Fatalf("expected NH winter rain belt to favor southern tropics: %.2f <= %.2f", winter[1], winter[0])
	}
}

func TestApplySeasonalPrecipitationPatternFlipsSubtropicalOnshoreWetSeason(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(20, 0),
		seasonalLatLonVertex(-20, 0),
		seasonalLatLonVertex(20, -8),
		seasonalLatLonVertex(-20, -8),
	}
	elevation := []float64{100, 100, -100, -100}
	adj := &FlatAdjacency{
		Neighbors: []int{
			2,
			3,
			0,
			1,
		},
		Offsets: []int{0, 1, 2, 3, 4},
	}

	wind := []Vector3D{
		Normalize(Sub(vertices[0], vertices[2])),
		Normalize(Sub(vertices[1], vertices[3])),
		Vector3D{},
		Vector3D{},
	}
	precip := []float64{100, 100, 0, 0}

	summer := ApplySeasonalPrecipitationPattern(
		vertices,
		elevation,
		0.0,
		adj,
		wind,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5},
		nil,
		nil,
		nil,
		precip,
	)
	if summer[0] <= summer[1] {
		t.Fatalf("expected NH summer subtropical wet season in north: %.2f <= %.2f", summer[0], summer[1])
	}

	winter := ApplySeasonalPrecipitationPattern(
		vertices,
		elevation,
		0.0,
		adj,
		wind,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.0},
		nil,
		nil,
		nil,
		precip,
	)
	if winter[1] <= winter[0] {
		t.Fatalf("expected NH winter subtropical wet season in south: %.2f <= %.2f", winter[1], winter[0])
	}
}

func TestCoastalOnshoreScoreUsesPhysicalCoastalBand(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(20, 0),
		seasonalLatLonVertex(20, -4),
	}
	elevation := []float64{100, -100}
	adj := &FlatAdjacency{
		Neighbors: []int{1, 0},
		Offsets:   []int{0, 1, 2},
	}
	wind := []Vector3D{
		Normalize(Sub(vertices[0], vertices[1])),
		Vector3D{},
	}

	coastalBand := coastalOnshoreScore(0, vertices, elevation, 0, adj, wind)
	if coastalBand <= 0 {
		t.Fatalf("expected inland cell within physical coastal band to receive onshore support")
	}
}

func TestApplySeasonalPrecipitationPatternDriesWinterPolarLand(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(70, 0),
	}
	elevation := []float64{100}
	precip := []float64{100}

	summer := ApplySeasonalPrecipitationPattern(
		vertices,
		elevation,
		0.0,
		&FlatAdjacency{Offsets: []int{0, 0}},
		nil,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5},
		nil,
		nil,
		nil,
		precip,
	)
	winter := ApplySeasonalPrecipitationPattern(
		vertices,
		elevation,
		0.0,
		&FlatAdjacency{Offsets: []int{0, 0}},
		nil,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.0},
		nil,
		nil,
		nil,
		precip,
	)

	if winter[0] >= summer[0] {
		t.Fatalf("expected winter polar land to be drier than summer: winter=%.2f summer=%.2f", winter[0], summer[0])
	}
}

func TestSeasonalHumidityFactorUsesTemperatureAnomaly(t *testing.T) {
	factorCold := seasonalHumidityFactor(
		0,
		55.0,
		[]float64{273.15},
		[]float64{283.15},
		nil,
	)
	factorWarm := seasonalHumidityFactor(
		0,
		55.0,
		[]float64{293.15},
		[]float64{283.15},
		nil,
	)

	if factorWarm <= factorCold {
		t.Fatalf("expected warm seasonal anomaly to support more humidity: warm=%.3f cold=%.3f", factorWarm, factorCold)
	}
}

func TestSeasonalHumidityFactorDriesColdPolarInteriorMoreThanCoast(t *testing.T) {
	temperature := []float64{258.15}
	annual := []float64{268.15}
	coastal := seasonalHumidityFactor(0, 72.0, temperature, annual, []float64{0.0})
	interior := seasonalHumidityFactor(0, 72.0, temperature, annual, []float64{1.0})

	if interior >= coastal {
		t.Fatalf("expected cold polar interior to be drier than coast: interior=%.3f coastal=%.3f", interior, coastal)
	}
}

func TestDeriveSeasonalPrecipitationSettingsRespondsToOceanThermalState(t *testing.T) {
	base := DefaultPrecipitationSettings()
	vertices := []Vector3D{seasonalLatLonVertex(0, 0), seasonalLatLonVertex(0, 30)}
	elevation := []float64{-100, -100}
	annual := []float64{288.15, 288.15}
	warmSeason := []float64{294.15, 292.15}
	coldSeason := []float64{282.15, 284.15}

	warm := DeriveSeasonalPrecipitationSettings(
		base, vertices, elevation, 0.0,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5},
		warmSeason, annual,
	)
	cold := DeriveSeasonalPrecipitationSettings(
		base, vertices, elevation, 0.0,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.0},
		coldSeason, annual,
	)

	if warm.EvaporationRate <= cold.EvaporationRate {
		t.Fatalf("expected warm seasonal ocean state to raise evaporation: warm=%.3f cold=%.3f", warm.EvaporationRate, cold.EvaporationRate)
	}
	if warm.OceanEvaporationTempEffect < base.OceanEvaporationTempEffect {
		t.Fatalf("expected seasonal settings to preserve or raise ocean temp effect: got %.3f base %.3f", warm.OceanEvaporationTempEffect, base.OceanEvaporationTempEffect)
	}
}

func TestDeriveSeasonalPrecipitationSettingsRespondsToWarmLandSeason(t *testing.T) {
	base := DefaultPrecipitationSettings()
	vertices := []Vector3D{
		seasonalLatLonVertex(20, 0),
		seasonalLatLonVertex(25, 10),
		seasonalLatLonVertex(0, 0),
	}
	elevation := []float64{100, 200, -100}
	annual := []float64{285.15, 286.15, 289.15}
	warmSeason := []float64{293.15, 294.15, 290.15}
	coldSeason := []float64{277.15, 278.15, 288.15}

	warm := DeriveSeasonalPrecipitationSettings(
		base, vertices, elevation, 0.0,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5},
		warmSeason, annual,
	)
	cold := DeriveSeasonalPrecipitationSettings(
		base, vertices, elevation, 0.0,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.0},
		coldSeason, annual,
	)

	if warm.LandSourceScale <= cold.LandSourceScale {
		t.Fatalf("expected warm land season to raise land moisture source: warm=%.3f cold=%.3f", warm.LandSourceScale, cold.LandSourceScale)
	}
	if warm.LandRecyclingScale <= cold.LandRecyclingScale {
		t.Fatalf("expected warm land season to raise land recycling: warm=%.3f cold=%.3f", warm.LandRecyclingScale, cold.LandRecyclingScale)
	}
}

func TestSeasonalResidualIsLighterThanFullPattern(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(8, 0),
		seasonalLatLonVertex(-8, 0),
	}
	elevation := []float64{100, 100}
	precip := []float64{100, 100}
	solar := SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5}

	full := ApplySeasonalPrecipitationPattern(
		vertices, elevation, 0.0, &FlatAdjacency{Offsets: []int{0, 0, 0}}, nil,
		solar, nil, nil, nil, precip,
	)
	residual := ApplySeasonalPrecipitationResidual(
		vertices, elevation, 0.0, &FlatAdjacency{Offsets: []int{0, 0, 0}}, nil,
		solar, nil, nil, nil, precip,
	)

	fullShift := math.Abs(full[0] - full[1])
	residualShift := math.Abs(residual[0] - residual[1])
	if residualShift >= fullShift {
		t.Fatalf("expected residual seasonal correction to be lighter than full pattern: residual=%.2f full=%.2f", residualShift, fullShift)
	}
}

func seasonalLatLonVertex(latDeg, lonDeg float64) Vector3D {
	lat := latDeg * math.Pi / 180.0
	lon := lonDeg * math.Pi / 180.0
	cosLat := math.Cos(lat)
	return Vector3D{
		X: cosLat * math.Cos(lon),
		Y: math.Sin(lat),
		Z: cosLat * math.Sin(lon),
	}
}
