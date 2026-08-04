package climgen

import "testing"

func TestPopulationSupportFavorsFoodAndWaterRichCell(t *testing.T) {
	settings := DefaultPopulationSupportSettings()
	settlements := &SettlementResult{
		Classes: []SettlementClass{SettlementFavorable, SettlementFavorable},
		Diagnostics: &SettlementDiagnostics{
			ClimateScore:  []float64{0.7, 0.7},
			WaterScore:    []float64{0.8, 0.2},
			TerrainScore:  []float64{0.7, 0.7},
			SoilScore:     []float64{0.7, 0.7},
			AccessScore:   []float64{0.6, 0.3},
			ResourceScore: []float64{0.4, 0.4},
			HazardPenalty: []float64{0.1, 0.1},
			RiverBonus:    []float64{0.7, 0.1},
			CoastalBonus:  []float64{0.1, 0.0},
			Suitability:   []float64{0.65, 0.45},
		},
	}
	agriculture := &AgricultureResult{
		Types: []AgricultureType{AgricultureMixedFarming, AgricultureUnsuitable},
		Diagnostics: &AgricultureDiagnostics{
			CropPotential:       []float64{0.8, 0.1},
			PasturePotential:    []float64{0.4, 0.1},
			IrrigationPotential: []float64{0.6, 0.0},
			FloodplainPotential: []float64{0.5, 0.0},
		},
	}
	wildlife := &WildlifeResult{
		Types: []WildlifeType{WildlifeForestGame, WildlifeSparse},
		Diagnostics: &WildlifeDiagnostics{
			GamePotential:   []float64{0.5, 0.1},
			TimberPotential: []float64{0.4, 0.0},
		},
	}
	water := &WaterResourceResult{
		Types: []WaterResourceType{WaterResourceReliableSurface, WaterResourceScarce},
		Diagnostics: &WaterResourceDiagnostics{
			SurfaceReliability:   []float64{0.9, 0.0},
			SeasonalAvailability: []float64{0.4, 0.1},
			GroundwaterPotential: []float64{0.5, 0.1},
			LakeAccess:           []float64{0.0, 0.0},
		},
	}
	elevation := []float64{100, 100}

	result := ClassifyPopulationSupport(settlements, agriculture, wildlife, water, nil, nil, nil, elevation, 0, settings)
	if result.Diagnostics.CarryingCapacity[0] <= result.Diagnostics.CarryingCapacity[1] {
		t.Fatalf("expected richer cell to have higher carrying capacity: got %.2f <= %.2f",
			result.Diagnostics.CarryingCapacity[0], result.Diagnostics.CarryingCapacity[1])
	}
	if result.Classes[0] < PopulationRural {
		t.Fatalf("expected richer cell to support at least rural population, got %v", result.Classes[0])
	}
}
