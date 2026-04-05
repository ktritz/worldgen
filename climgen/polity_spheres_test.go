package climgen

import "testing"

func TestBuildPolitySpheresCreatesCapitalClaims(t *testing.T) {
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.66, CarryingCapacity: 0.60, UrbanPotential: 0.58, River: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeVillage, Score: 0.56, CarryingCapacity: 0.50, UrbanPotential: 0.42},
			{ID: 2, CellIndex: 2, Kind: SettlementNodeTown, Score: 0.68, CarryingCapacity: 0.61, UrbanPotential: 0.60, Coastal: true},
			{ID: 3, CellIndex: 3, Kind: SettlementNodeVillage, Score: 0.54, CarryingCapacity: 0.46, UrbanPotential: 0.40},
		},
		Links: []SettlementLink{
			{From: 0, To: 1, TravelCost: 3.8, Path: []int{0, 1}},
			{From: 1, To: 2, TravelCost: 5.1, Path: []int{1, 2}},
			{From: 2, To: 3, TravelCost: 4.0, Path: []int{2, 3}},
		},
		Diagnostics: &SettlementNetworkDiagnostics{
			MovementCost: []float64{1, 1, 1, 1},
		},
	}
	proto := &ProtoCivilizationResult{
		Civilizations: []ProtoCivilization{
			{ID: 0, RegionID: 0, CenterNode: 0, Style: ProtoCivilizationRiverine, River: true},
			{ID: 1, RegionID: 1, CenterNode: 2, Style: ProtoCivilizationMaritime, Coastal: true},
		},
	}
	trade := &TradeNetworkResult{
		MajorHubs: []int{0, 2},
		Diagnostics: &TradeNetworkDiagnostics{
			CivilizationByNode: []int{0, 0, 1, 1},
			NodeCentrality:     []float64{0.7, 0.2, 0.8, 0.2},
			HubScore:           []float64{1.02, 0.0, 1.04, 0.0},
		},
	}
	population := &PopulationResult{
		Classes: []PopulationClass{PopulationDenseRural, PopulationRural, PopulationDenseRural, PopulationRural},
		Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: []float64{0.60, 0.44, 0.61, 0.42},
		},
	}
	settlements := &SettlementResult{
		Classes: []SettlementClass{SettlementFavorable, SettlementFavorable, SettlementFavorable, SettlementMarginal},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2}},
	}
	elevation := []float64{100, 100, 100, 100}

	settings := DefaultPolitySphereSettings()
	settings.MinTerritoryCells = 1
	result := BuildPolitySpheres(cells, network, proto, trade, population, settlements, elevation, 0, settings)
	if len(result.Spheres) == 0 {
		t.Fatalf("expected at least one polity sphere")
	}
	if result.Diagnostics.PolityByCell[0] < 0 {
		t.Fatalf("expected capital cell to belong to a polity sphere")
	}
}
