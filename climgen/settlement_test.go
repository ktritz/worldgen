package climgen

import "testing"

func TestClassifySettlementClass(t *testing.T) {
	tests := []struct {
		score float64
		want  SettlementClass
	}{
		{0.10, SettlementUnsuitable},
		{0.30, SettlementMarginal},
		{0.55, SettlementFavorable},
		{0.82, SettlementPrime},
	}
	for _, tt := range tests {
		if got := classifySettlementClass(tt.score); got != tt.want {
			t.Fatalf("score %.2f => %s, want %s", tt.score, SettlementClassName(got), SettlementClassName(tt.want))
		}
	}
}

func TestSettlementResourceScoreRewardsUsefulClasses(t *testing.T) {
	result := &ResourceResult{
		Types: []ResourceType{ResourceNone, ResourcePlacerAlluvial, ResourceIndustrialStone},
		Diagnostics: &ResourceDiagnostics{
			PlacerAffinity:    []float64{0.0, 0.8, 0.1},
			StoneAffinity:     []float64{0.0, 0.2, 0.7},
			IronAffinity:      []float64{0.0, 0.1, 0.0},
			CopperAffinity:    []float64{0.0, 0.0, 0.0},
			CoalAffinity:      []float64{0.0, 0.2, 0.1},
			OilGasAffinity:    []float64{0.0, 0.0, 0.0},
			EvaporiteAffinity: []float64{0.0, 0.0, 0.0},
		},
	}
	if settlementResourceScore(result, 1) <= settlementResourceScore(result, 0) {
		t.Fatalf("placer resource should improve settlement resource score")
	}
	if settlementResourceScore(result, 2) <= settlementResourceScore(result, 0) {
		t.Fatalf("industrial stone should improve settlement resource score")
	}
}

func TestSettlementWaterResourcesImproveSuitability(t *testing.T) {
	biomes := &BiomeResult{
		Biomes: []Biome{BiomeTemperateForest, BiomeTemperateForest},
		Diagnostics: &BiomeDiagnostics{
			AnnualMeanTempC:    []float64{15, 15},
			WarmestSeasonTempC: []float64{24, 24},
			AnnualPrecipCm:     []float64{90, 90},
			AridityRatio:       []float64{1.2, 1.2},
			DrySeasonRatio:     []float64{0.40, 0.40},
			AnnualIceFraction:  []float64{0, 0},
		},
	}
	soils := &SoilResult{
		Types: []SoilType{SoilTemperateLoam, SoilTemperateLoam},
		Diagnostics: &SoilDiagnostics{
			Fertility: []float64{0.6, 0.6},
			Alluvial:  []float64{0.1, 0.1},
			Salinity:  []float64{0, 0},
			Drainage:  []float64{0.7, 0.7},
			Relief:    []float64{120, 120},
			Rockiness: []float64{0.1, 0.1},
		},
	}
	water := &WaterResourceResult{
		Types: []WaterResourceType{WaterResourceReliableSurface, WaterResourceScarce},
		Diagnostics: &WaterResourceDiagnostics{
			SurfaceReliability:   []float64{0.85, 0.05},
			SeasonalAvailability: []float64{0.30, 0.10},
			GroundwaterPotential: []float64{0.25, 0.10},
			LakeAccess:           []float64{0.00, 0.00},
		},
	}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{20, 20},
		ChannelStrength: []float64{0.4, 0.4},
		CellClass:       []string{"hillslope", "hillslope"},
	}

	result := ClassifySettlementSuitability(nil, biomes, soils, nil, water, nil, []float64{100, 100}, 0, hydro, []float64{0, 0})
	if result.Diagnostics.Suitability[0] <= result.Diagnostics.Suitability[1] {
		t.Fatalf("expected reliable water cell to score higher: %.3f <= %.3f", result.Diagnostics.Suitability[0], result.Diagnostics.Suitability[1])
	}
}
