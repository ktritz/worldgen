package climgen

import "testing"

func TestBuildLandRouteDiagnosticsRespectsModeSpecialization(t *testing.T) {
	biomes := &BiomeResult{
		Diagnostics: &BiomeDiagnostics{
			AridityRatio:       []float64{0.6, 0.7},
			WetlandAffinity:    []float64{0.8, 0.1},
			AnnualIceFraction:  []float64{0.0, 0.0},
			WarmestSeasonTempC: []float64{26, 26},
			GrasslandAffinity:  []float64{0.3, 0.4},
		},
	}
	vegetation := &VegetationResult{
		Diagnostics: &VegetationDiagnostics{
			TreeCover:    []float64{0.1, 0.1},
			ShrubCover:   []float64{0.2, 0.2},
			GrassCover:   []float64{0.2, 0.3},
			WetlandCover: []float64{0.9, 0.0},
		},
	}
	soils := &SoilResult{
		Diagnostics: &SoilDiagnostics{
			Relief:    []float64{120, 120},
			Rockiness: []float64{0.1, 0.1},
		},
	}
	water := &WaterResourceResult{
		Diagnostics: &WaterResourceDiagnostics{
			SurfaceReliability:   []float64{0.7, 0.3},
			SeasonalAvailability: []float64{0.5, 0.3},
			GroundwaterPotential: []float64{0.4, 0.2},
			LakeAccess:           []float64{0.1, 0.0},
		},
	}
	elevation := []float64{1, 1}

	settings := DefaultLandRouteSettings()
	settings.DefaultMode = "horse-wagon"
	wagon := BuildLandRouteDiagnostics(nil, nil, biomes, vegetation, soils, nil, water, elevation, 0, nil, settings)
	settings.DefaultMode = "pack-lizard"
	lizard := BuildLandRouteDiagnostics(nil, nil, biomes, vegetation, soils, nil, water, elevation, 0, nil, settings)

	if !(lizard.Diagnostics.ModeCost[0] < wagon.Diagnostics.ModeCost[0]) {
		t.Fatalf("expected pack-lizard to outperform wagon in marsh cell, got lizard=%.2f wagon=%.2f", lizard.Diagnostics.ModeCost[0], wagon.Diagnostics.ModeCost[0])
	}
}

func TestBuildLandRouteDiagnosticsUsesCrossingProxies(t *testing.T) {
	biomes := &BiomeResult{
		Diagnostics: &BiomeDiagnostics{
			AridityRatio:       []float64{0.7},
			WetlandAffinity:    []float64{0.1},
			AnnualIceFraction:  []float64{0.0},
			WarmestSeasonTempC: []float64{22},
			GrasslandAffinity:  []float64{0.3},
		},
	}
	settlements := &SettlementResult{
		Diagnostics: &SettlementDiagnostics{
			AccessScore:  []float64{0.7},
			Suitability:  []float64{0.7},
			RiverBonus:   []float64{0.6},
			CoastalBonus: []float64{0.0},
		},
	}
	population := &PopulationResult{
		Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: []float64{0.7},
			UrbanPotential:   []float64{0.4},
		},
	}
	soils := &SoilResult{
		Diagnostics: &SoilDiagnostics{
			Relief:    []float64{120},
			Rockiness: []float64{0.1},
		},
	}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{50},
		ChannelStrength: []float64{2.1},
		CellClass:       []string{"confluence"},
	}
	elevation := []float64{1}

	settings := DefaultLandRouteSettings()
	settings.DefaultMode = "horse-wagon"
	wagon := BuildLandRouteDiagnostics(settlements, population, biomes, nil, soils, nil, nil, elevation, 0, hydro, settings)
	settings.DefaultMode = "pack-mule"
	pack := BuildLandRouteDiagnostics(settlements, population, biomes, nil, soils, nil, nil, elevation, 0, hydro, settings)

	if wagon.Diagnostics.CrossingPressure[0] <= 0 {
		t.Fatalf("expected crossing pressure proxy to be populated")
	}
	if wagon.Diagnostics.BridgeProxy[0] <= 0 || wagon.Diagnostics.FordProxy[0] <= 0 {
		t.Fatalf("expected bridge and ford proxies to be populated")
	}
	if !(wagon.Diagnostics.ModeCost[0] > pack.Diagnostics.ModeCost[0]) {
		t.Fatalf("expected wagon-like mode to pay more than pack-mule across crossing-heavy terrain, got wagon=%.2f pack=%.2f", wagon.Diagnostics.ModeCost[0], pack.Diagnostics.ModeCost[0])
	}
}
