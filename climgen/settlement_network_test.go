package climgen

import "testing"

func TestBuildSettlementNetworkCreatesLinkBetweenNearbyPeaks(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0.99, Y: 0.1, Z: 0},
		{X: 0.96, Y: 0.2, Z: 0},
		{X: 0.93, Y: 0.3, Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2}},
	}
	settlements := &SettlementResult{
		Classes: []SettlementClass{SettlementFavorable, SettlementMarginal, SettlementMarginal, SettlementFavorable},
		Diagnostics: &SettlementDiagnostics{
			WaterScore:    []float64{0.8, 0.5, 0.5, 0.8},
			TerrainScore:  []float64{0.8, 0.7, 0.7, 0.8},
			SoilScore:     []float64{0.8, 0.6, 0.6, 0.8},
			AccessScore:   []float64{0.7, 0.4, 0.4, 0.7},
			ResourceScore: []float64{0.4, 0.3, 0.3, 0.4},
			HazardPenalty: []float64{0.0, 0.0, 0.0, 0.0},
			RiverBonus:    []float64{0.4, 0.2, 0.2, 0.4},
			CoastalBonus:  []float64{0.0, 0.0, 0.0, 0.0},
			Suitability:   []float64{0.75, 0.40, 0.40, 0.74},
		},
	}
	population := &PopulationResult{
		Classes: []PopulationClass{PopulationDenseRural, PopulationSparseFrontier, PopulationSparseFrontier, PopulationDenseRural},
		Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: []float64{0.66, 0.24, 0.25, 0.64},
			UrbanPotential:   []float64{0.57, 0.18, 0.19, 0.56},
		},
	}
	soils := &SoilResult{Diagnostics: &SoilDiagnostics{
		Relief:    []float64{100, 120, 120, 100},
		Rockiness: []float64{0.1, 0.1, 0.1, 0.1},
	}}
	biomes := &BiomeResult{Diagnostics: &BiomeDiagnostics{
		AnnualIceFraction: []float64{0, 0, 0, 0},
		AridityRatio:      []float64{1, 1, 1, 1},
		WetlandAffinity:   []float64{0, 0, 0, 0},
	}}
	elevation := []float64{100, 100, 100, 100}

	result := BuildSettlementNetwork(sites, cells, settlements, population, biomes, soils, nil, elevation, 0, DefaultSettlementNetworkSettings())
	if len(result.Nodes) < 2 {
		t.Fatalf("expected at least 2 nodes, got %d", len(result.Nodes))
	}
	if len(result.Links) == 0 {
		t.Fatalf("expected at least 1 link between nearby supported nodes")
	}
}
