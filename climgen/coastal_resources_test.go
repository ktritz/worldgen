package climgen

import "testing"

func TestClassifyCoastalResourcesFindsEstuaryAndSaltworks(t *testing.T) {
	biomes := &BiomeResult{
		Biomes: []Biome{BiomeTemperateForest, BiomeDesertHot},
		Diagnostics: &BiomeDiagnostics{
			AnnualMeanTempC:    []float64{16, 25},
			WarmestSeasonTempC: []float64{24, 34},
			AnnualPrecipCm:     []float64{120, 18},
			AridityRatio:       []float64{1.3, 0.22},
			AnnualIceFraction:  []float64{0, 0},
		},
	}
	soils := &SoilResult{
		Types: []SoilType{SoilAlluvial, SoilSalineCoastal},
		Diagnostics: &SoilDiagnostics{
			Alluvial: []float64{0.9, 0.1},
			Salinity: []float64{0.1, 0.9},
		},
	}
	vegetation := &VegetationResult{
		Types: []VegetationType{VegetationWetland, VegetationSaltMarsh},
		Diagnostics: &VegetationDiagnostics{
			WetlandCover:      []float64{0.8, 0.4},
			MangroveAffinity:  []float64{0.0, 0.0},
			SaltMarshAffinity: []float64{0.1, 0.7},
		},
	}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{95, 8},
		ChannelStrength: []float64{2.4, 0.2},
		CellClass:       []string{"delta", "coast_outlet"},
	}
	settings := DefaultCoastalResourceSettings()

	result := ClassifyCoastalResources(
		nil,
		nil,
		nil,
		biomes,
		soils,
		vegetation,
		[]float64{100, 50},
		0,
		hydro,
		[]float64{0.9, 0.8},
		settings,
	)

	if result.Types[0] != CoastalResourceEstuarineFishery {
		t.Fatalf("expected estuarine fishery for delta cell, got %s", CoastalResourceName(result.Types[0]))
	}
	if result.Types[1] != CoastalResourceSaltworks {
		t.Fatalf("expected saltworks for arid saline coast, got %s", CoastalResourceName(result.Types[1]))
	}
}

func TestCoastalResourceSettingsCanPromoteShellfish(t *testing.T) {
	biomes := &BiomeResult{
		Biomes: []Biome{BiomeTemperateForest},
		Diagnostics: &BiomeDiagnostics{
			AnnualMeanTempC:    []float64{14},
			WarmestSeasonTempC: []float64{22},
			AnnualPrecipCm:     []float64{70},
			AridityRatio:       []float64{1.0},
			AnnualIceFraction:  []float64{0},
		},
	}
	vegetation := &VegetationResult{
		Types: []VegetationType{VegetationSaltMarsh},
		Diagnostics: &VegetationDiagnostics{
			WetlandCover:      []float64{0.55},
			MangroveAffinity:  []float64{0.0},
			SaltMarshAffinity: []float64{0.8},
		},
	}
	settings := DefaultCoastalResourceSettings()
	settings.ShellfishMultiplier = 4.0
	settings.ShellfishPrimaryBias = 0.2

	result := ClassifyCoastalResources(
		nil,
		nil,
		nil,
		biomes,
		nil,
		vegetation,
		[]float64{30},
		0,
		&HydrologyBiomeInputs{
			Runoff:          []float64{25},
			ChannelStrength: []float64{0.6},
			CellClass:       []string{"coast_outlet"},
		},
		[]float64{0.95},
		settings,
	)
	if result.Types[0] != CoastalResourceShellfish {
		t.Fatalf("expected shellfish with boosted shellfish settings, got %s", CoastalResourceName(result.Types[0]))
	}
}
