package climgen

import "testing"

func TestClassifyWaterResourcesDistinguishesRiverAndAquifer(t *testing.T) {
	biomes := &BiomeResult{
		Biomes: []Biome{BiomeTemperateForest, BiomeSemiArid},
		Diagnostics: &BiomeDiagnostics{
			AnnualPrecipCm:    []float64{110, 35},
			AridityRatio:      []float64{1.5, 0.85},
			DrySeasonRatio:    []float64{0.45, 0.25},
			AnnualIceFraction: []float64{0, 0},
		},
	}
	soils := &SoilResult{
		Types: []SoilType{SoilAlluvial, SoilTemperateLoam},
		Diagnostics: &SoilDiagnostics{
			Drainage: []float64{0.45, 0.62},
			Alluvial: []float64{0.9, 0.2},
			Salinity: []float64{0.0, 0.0},
			Organic:  []float64{0.2, 0.1},
		},
	}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{95, 8},
		ChannelStrength: []float64{2.4, 0.2},
		CellClass:       []string{"floodplain", "hillslope"},
	}
	result := ClassifyWaterResources(biomes, soils, []float64{100, 120}, 0, hydro, DefaultWaterResourceSettings())
	if result.Types[0] != WaterResourceReliableSurface {
		t.Fatalf("expected reliable surface water, got %s", WaterResourceName(result.Types[0]))
	}
	if result.Types[1] != WaterResourceGroundwater {
		t.Fatalf("expected groundwater, got %s", WaterResourceName(result.Types[1]))
	}
}

func TestLakeClassCanProduceLakeWater(t *testing.T) {
	biomes := &BiomeResult{
		Biomes: []Biome{BiomeSemiArid},
		Diagnostics: &BiomeDiagnostics{
			AnnualPrecipCm:    []float64{25},
			AridityRatio:      []float64{0.45},
			DrySeasonRatio:    []float64{0.18},
			AnnualIceFraction: []float64{0},
		},
	}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{6},
		ChannelStrength: []float64{0.4},
		CellClass:       []string{"lake_complex"},
	}
	settings := DefaultWaterResourceSettings()
	settings.LakeAccessMultiplier = 1.0
	result := ClassifyWaterResources(biomes, nil, []float64{50}, 0, hydro, settings)
	if result.Types[0] != WaterResourceLakeOasis {
		t.Fatalf("expected lake/oasis water, got %s", WaterResourceName(result.Types[0]))
	}
}
