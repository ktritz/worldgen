package climgen

import "testing"

func TestDefaultAgricultureProductivitySettingsValidate(t *testing.T) {
	if err := ValidateAgricultureProductivitySettings(DefaultAgricultureProductivitySettings()); err != nil {
		t.Fatalf("default agriculture settings should validate: %v", err)
	}
}

func TestClassifyAgricultureFloodplainAndPastoral(t *testing.T) {
	biomes := &BiomeResult{
		Diagnostics: &BiomeDiagnostics{
			AnnualMeanTempC:    []float64{20, 14},
			WarmestSeasonTempC: []float64{29, 22},
			AnnualIceFraction:  []float64{0, 0},
			AnnualPrecipCm:     []float64{120, 40},
			DrySeasonRatio:     []float64{0.45, 0.18},
			AridityRatio:       []float64{1.4, 0.75},
			GrasslandAffinity:  []float64{0.2, 0.7},
		},
	}
	soils := &SoilResult{
		Diagnostics: &SoilDiagnostics{
			Fertility: []float64{0.85, 0.45},
			Drainage:  []float64{0.55, 0.72},
			Alluvial:  []float64{0.92, 0.10},
			Salinity:  []float64{0.02, 0.02},
			Rockiness: []float64{0.10, 0.18},
			Relief:    []float64{60, 120},
		},
	}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{95, 10},
		ChannelStrength: []float64{1.9, 0.2},
		CellClass:       []string{"floodplain", "hillslope"},
	}
	got := ClassifyAgriculture(biomes, soils, []float64{120, 450}, 0, hydro, DefaultAgricultureProductivitySettings())
	if got.Types[0] != AgricultureFloodplainCropland {
		t.Fatalf("expected floodplain cropland, got %s", AgricultureName(got.Types[0]))
	}
	if got.Types[1] != AgriculturePastoral && got.Types[1] != AgricultureDryFarming {
		t.Fatalf("expected pastoral or dry farming on semi-arid grassland, got %s", AgricultureName(got.Types[1]))
	}
}
