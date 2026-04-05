package climgen

import "testing"

func TestBuildTradeNetworkCreatesInterCivilizationRoute(t *testing.T) {
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.66, CarryingCapacity: 0.62, UrbanPotential: 0.58, River: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeVillage, Score: 0.54, CarryingCapacity: 0.50, UrbanPotential: 0.38},
			{ID: 2, CellIndex: 2, Kind: SettlementNodeTown, Score: 0.68, CarryingCapacity: 0.60, UrbanPotential: 0.60, Coastal: true},
		},
		Links: []SettlementLink{
			{From: 0, To: 1, TravelCost: 4.0, Path: []int{0, 1}},
			{From: 1, To: 2, TravelCost: 4.5, Path: []int{1, 2}},
		},
		Regions: []SettlementRegion{
			{ID: 0, NodeIndices: []int{0, 1}, CenterNode: 0, MeanScore: 0.60, River: true},
			{ID: 1, NodeIndices: []int{2}, CenterNode: 2, MeanScore: 0.68, Coastal: true},
		},
	}
	proto := &ProtoCivilizationResult{
		Civilizations: []ProtoCivilization{
			{ID: 0, RegionID: 0, CenterNode: 0, AnchorCount: 2, CenterKind: SettlementNodeTown, River: true, TerritoryCells: 40, MeanSupport: 0.40},
			{ID: 1, RegionID: 1, CenterNode: 2, AnchorCount: 1, CenterKind: SettlementNodeTown, Coastal: true, TerritoryCells: 32, MeanSupport: 0.38},
		},
	}

	result := BuildTradeNetwork(make([]VoronoiCell, 3), network, proto, DefaultTradeNetworkSettings())
	if len(result.Corridors) == 0 {
		t.Fatalf("expected at least one trade corridor")
	}
	foundInter := false
	for _, corridor := range result.Corridors {
		if corridor.InterCivilization {
			foundInter = true
			if len(corridor.CellPath) == 0 {
				t.Fatalf("expected inter-civilization corridor cell path to be populated")
			}
			break
		}
	}
	if !foundInter {
		t.Fatalf("expected at least one inter-civilization trade corridor")
	}
}
