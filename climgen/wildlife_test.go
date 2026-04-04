package climgen

import "testing"

func TestDefaultWildlifeSettingsValidate(t *testing.T) {
	if err := ValidateWildlifeProductivitySettings(DefaultWildlifeProductivitySettings()); err != nil {
		t.Fatalf("default wildlife settings should validate: %v", err)
	}
}

func TestWildlifeSettingsCanDisablePelts(t *testing.T) {
	biomes := &BiomeResult{
		Diagnostics: &BiomeDiagnostics{
			AnnualPrecipCm:      []float64{70},
			WarmestSeasonTempC:  []float64{7},
			AridityRatio:        []float64{1.1},
			GrasslandAffinity:   []float64{0.3},
			ForestAffinity:      []float64{0.4},
			WetlandAffinity:     []float64{0.1},
			BorealAffinity:      []float64{0.7},
			TundraAffinity:      []float64{0.4},
			TropicalWetAffinity: []float64{0.0},
		},
	}
	vegetation := &VegetationResult{
		Diagnostics: &VegetationDiagnostics{
			TreeCover:    []float64{0.45},
			GrassCover:   []float64{0.15},
			ShrubCover:   []float64{0.25},
			WetlandCover: []float64{0.05},
		},
	}
	settings := DefaultWildlifeProductivitySettings()
	withPelts := ClassifyWildlife(biomes, vegetation, nil, []float64{300}, 0, nil, settings)
	settings.FurredAnimalsPresent = false
	withoutPelts := ClassifyWildlife(biomes, vegetation, nil, []float64{300}, 0, nil, settings)
	if withPelts.Diagnostics.PeltPotential[0] <= 0 {
		t.Fatalf("expected positive pelt potential when furred animals are present")
	}
	if withoutPelts.Diagnostics.PeltPotential[0] != 0 {
		t.Fatalf("expected zero pelt potential when furred animals are absent, got %.3f", withoutPelts.Diagnostics.PeltPotential[0])
	}
}
