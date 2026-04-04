package climgen

import "testing"

func TestBuildProtoCivilizationsClaimsSupportedTerritory(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0.99, Y: 0.1, Z: 0},
		{X: 0.97, Y: 0.2, Z: 0},
		{X: 0.95, Y: 0.3, Z: 0},
		{X: 0.93, Y: 0.4, Z: 0},
		{X: 0.91, Y: 0.5, Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2, 4}},
		{SiteIndex: 4, NeighborSiteIndices: []int32{3, 5}},
		{SiteIndex: 5, NeighborSiteIndices: []int32{4}},
	}
	settlements := &SettlementResult{
		Classes: []SettlementClass{
			SettlementMarginal,
			SettlementFavorable,
			SettlementFavorable,
			SettlementFavorable,
			SettlementFavorable,
			SettlementMarginal,
		},
		Diagnostics: &SettlementDiagnostics{
			AccessScore:  []float64{0.55, 0.74, 0.80, 0.78, 0.74, 0.60},
			RiverBonus:   []float64{0.10, 0.35, 0.40, 0.42, 0.30, 0.10},
			CoastalBonus: []float64{0, 0, 0, 0, 0, 0},
			Suitability:  []float64{0.45, 0.72, 0.82, 0.80, 0.70, 0.48},
		},
	}
	population := &PopulationResult{
		Classes: []PopulationClass{
			PopulationSparseFrontier,
			PopulationRural,
			PopulationDenseRural,
			PopulationDenseRural,
			PopulationRural,
			PopulationSparseFrontier,
		},
		Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: []float64{0.28, 0.48, 0.61, 0.60, 0.47, 0.24},
			UrbanPotential:   []float64{0.16, 0.36, 0.55, 0.54, 0.34, 0.14},
		},
	}
	soils := &SoilResult{Diagnostics: &SoilDiagnostics{
		Relief:    []float64{110, 130, 160, 170, 140, 120},
		Rockiness: []float64{0.12, 0.12, 0.16, 0.16, 0.12, 0.12},
	}}
	biomes := &BiomeResult{Diagnostics: &BiomeDiagnostics{
		AnnualIceFraction: []float64{0, 0, 0, 0, 0, 0},
		AridityRatio:      []float64{0.35, 0.32, 0.30, 0.30, 0.34, 0.38},
		WetlandAffinity:   []float64{0.10, 0.24, 0.34, 0.36, 0.22, 0.10},
	}}
	elevation := []float64{100, 100, 100, 100, 100, 100}

	network := BuildSettlementNetwork(sites, cells, settlements, population, biomes, soils, nil, elevation, 0, DefaultSettlementNetworkSettings())
	settings := DefaultProtoCivilizationSettings()
	settings.MinRegionAnchors = 1
	settings.MinTerritoryCells = 3
	result := BuildProtoCivilizations(cells, network, settlements, population, biomes, soils, elevation, 0, settings)
	if len(result.Civilizations) == 0 {
		t.Fatalf("expected at least one proto-civilization")
	}
	if result.Civilizations[0].TerritoryCells == 0 {
		t.Fatalf("expected claimed territory for first proto-civilization")
	}
	if result.Civilizations[0].Style != ProtoCivilizationRiverine {
		t.Fatalf("expected riverine civilization style, got %s", ProtoCivilizationStyleName(result.Civilizations[0].Style))
	}
}
