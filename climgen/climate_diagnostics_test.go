package climgen

import (
	"math"
	"testing"
)

func TestComputeClimateDiagnosticsWindBands(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(4)
	settings := DefaultWindSettings()
	settings.Seed = 42
	result, err := GenerateWindField(vertices, elevation, 0.0, adj, settings)
	if err != nil {
		t.Fatalf("GenerateWindField failed: %v", err)
	}

	d := ComputeClimateDiagnostics(vertices, elevation, 0.0, adj, result, nil, nil, nil)
	if d.Wind.TradeWestFraction < 0.8 {
		t.Fatalf("TradeWestFraction = %.3f, want >= 0.8", d.Wind.TradeWestFraction)
	}
	if d.Wind.WesterlyEastFraction < 0.7 {
		t.Fatalf("WesterlyEastFraction = %.3f, want >= 0.7", d.Wind.WesterlyEastFraction)
	}
	if d.Wind.TangencyMaxError > 1e-6 {
		t.Fatalf("TangencyMaxError = %.6g, want <= 1e-6", d.Wind.TangencyMaxError)
	}
}

func TestComputeClimateDiagnosticsCurrentCoastConstraint(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(4)
	for i, v := range vertices {
		if v.X > 0.82 && absFloat(v.Z) < 0.35 {
			elevation[i] = 500
		}
	}

	windSettings := DefaultWindSettings()
	windSettings.Seed = 42
	windSettings.Orographic.DeflectionStrength = 0
	windResult, err := GenerateWindField(vertices, elevation, 0.0, adj, windSettings)
	if err != nil {
		t.Fatalf("GenerateWindField failed: %v", err)
	}

	assignments, components := FindOceanComponents(elevation, 0.0, adj)
	currents := GenerateCurrentsStreamfunctionFromWind(
		vertices, elevation, 0.0, adj, windResult.MarineWind, assignments, components, DefaultCurrentSettings(),
	)
	currentResult := &OceanCurrentResult{Currents: currents}

	d := ComputeClimateDiagnostics(vertices, elevation, 0.0, adj, nil, currentResult, nil, nil)
	if d.Currents.CoastNormalP95 > 0.05 {
		t.Fatalf("CoastNormalP95 = %.3f, want <= 0.05", d.Currents.CoastNormalP95)
	}
}

func TestComputeClimateDiagnosticsTemperatureGradient(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(3)
	temp := make([]float64, len(vertices))
	for i, v := range vertices {
		temp[i] = FreezingPoint + 30 - 50*absFloat(v.Y)
	}
	result := &TemperatureResult{
		Temperature:        temp,
		TemperatureCelsius: shiftKelvinToCelsius(temp),
		Converged:          true,
	}
	d := ComputeClimateDiagnostics(vertices, elevation, 0.0, adj, nil, nil, result, nil)
	if d.Temperature.EquatorPoleGradientC < 20 {
		t.Fatalf("EquatorPoleGradientC = %.1f, want >= 20", d.Temperature.EquatorPoleGradientC)
	}
	if d.Temperature.AbsLatitudeTempCorr > -0.8 {
		t.Fatalf("AbsLatitudeTempCorr = %.3f, want <= -0.8", d.Temperature.AbsLatitudeTempCorr)
	}
}

func TestComputeClimateDiagnosticsOceanClimateSignals(t *testing.T) {
	vertices := []Vector3D{
		latLonVertex(5, 0),
		latLonVertex(35, 0),
		latLonVertex(35, -12),
		latLonVertex(55, 120),
		latLonVertex(30, 120),
		latLonVertex(30, 132),
	}
	elevation := []float64{-1000, -1000, 500, -1000, -1000, 500}
	adj := &FlatAdjacency{
		Neighbors: []int{
			1,
			0, 2,
			1,
			4,
			3, 5,
			4,
		},
		Offsets: []int{0, 1, 3, 4, 5, 7, 8},
	}

	currents := make([]Vector3D, len(vertices))
	_, north1 := GetTangentVectors(vertices[1])
	currents[1] = north1
	_, north4 := GetTangentVectors(vertices[4])
	currents[4] = Scale(north4, -1)

	tempK := make([]float64, len(vertices))
	tempC := make([]float64, len(vertices))
	for i, v := range vertices {
		localEqC := LatitudeEquilibriumTemp(v.Y) - FreezingPoint
		residual := 0.0
		switch i {
		case 2:
			residual = 3.0
		case 5:
			residual = -3.0
		}
		tempC[i] = localEqC + residual
		tempK[i] = tempC[i] + FreezingPoint
	}

	currentResult := &OceanCurrentResult{Currents: currents}
	tempResult := &TemperatureResult{
		Temperature:        tempK,
		TemperatureCelsius: tempC,
		Converged:          true,
	}

	d := ComputeClimateDiagnostics(vertices, elevation, 0.0, adj, nil, currentResult, tempResult, nil)
	if d.OceanClimate.SourceAnomalyP90AbsC < 5 {
		t.Fatalf("SourceAnomalyP90AbsC = %.2f, want >= 5", d.OceanClimate.SourceAnomalyP90AbsC)
	}
	if d.OceanClimate.WarmWesternBoundarySignalC < 5 {
		t.Fatalf("WarmWesternBoundarySignalC = %.2f, want >= 5", d.OceanClimate.WarmWesternBoundarySignalC)
	}
	if d.OceanClimate.ColdEasternBoundarySignalC < 5 {
		t.Fatalf("ColdEasternBoundarySignalC = %.2f, want >= 5", d.OceanClimate.ColdEasternBoundarySignalC)
	}
	if d.OceanClimate.CoastalLandCouplingCorr < 0.9 {
		t.Fatalf("CoastalLandCouplingCorr = %.3f, want >= 0.9", d.OceanClimate.CoastalLandCouplingCorr)
	}
}

func TestComputeClimateDiagnosticsColdPrecipSignals(t *testing.T) {
	vertices := []Vector3D{
		latLonVertex(68, 0),
		latLonVertex(68, 8),
		latLonVertex(68, 16),
		latLonVertex(68, 24),
	}
	elevation := []float64{500, -1000, 500, 2200}
	adj := &FlatAdjacency{
		Neighbors: []int{
			1, 2,
			0, 2,
			0, 1, 3,
			2,
		},
		Offsets: []int{0, 2, 4, 7, 8},
	}
	tempK := []float64{265.15, 270.15, 263.15, 258.15}
	tempResult := &TemperatureResult{
		Temperature:        tempK,
		TemperatureCelsius: shiftKelvinToCelsius(tempK),
		Converged:          true,
	}
	precip := &PrecipitationResult{
		Precipitation: []float64{120, 0, 80, 30},
		Rainfall:      []float64{0, 0, 0, 0},
		Snowfall:      []float64{120, 0, 80, 30},
	}

	d := ComputeClimateDiagnostics(vertices, elevation, 0.0, adj, nil, nil, tempResult, precip)
	if d.Precipitation.ColdCoastalMean <= d.Precipitation.ColdInteriorMean {
		t.Fatalf("expected cold coastal precipitation to exceed cold interior: coast=%.1f interior=%.1f", d.Precipitation.ColdCoastalMean, d.Precipitation.ColdInteriorMean)
	}
	if d.Precipitation.ColdAlpineMean < 25 {
		t.Fatalf("expected cold alpine precipitation to be tracked, got %.1f", d.Precipitation.ColdAlpineMean)
	}
	if d.Precipitation.SnowFraction < 0.99 {
		t.Fatalf("expected all-land precip to be snow in this test, got %.3f", d.Precipitation.SnowFraction)
	}
}

func shiftKelvinToCelsius(temp []float64) []float64 {
	out := make([]float64, len(temp))
	for i, t := range temp {
		out[i] = t - FreezingPoint
	}
	return out
}

func latLonVertex(latDeg, lonDeg float64) Vector3D {
	lat := latDeg * (math.Pi / 180.0)
	lon := lonDeg * (math.Pi / 180.0)
	x := math.Cos(lat) * math.Cos(lon)
	y := math.Sin(lat)
	z := math.Cos(lat) * math.Sin(lon)
	return Normalize(Vector3D{X: x, Y: y, Z: z})
}
