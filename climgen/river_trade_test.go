package climgen

import "testing"

func TestBuildRiverTradeNetworkCreatesCorridorBetweenRiverCenters(t *testing.T) {
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.66, CarryingCapacity: 0.58, UrbanPotential: 0.54, River: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeVillage, Score: 0.52, CarryingCapacity: 0.48, UrbanPotential: 0.30, River: true},
			{ID: 2, CellIndex: 2, Kind: SettlementNodeTown, Score: 0.69, CarryingCapacity: 0.60, UrbanPotential: 0.56, River: true},
		},
		Links: []SettlementLink{
			{From: 0, To: 1, TravelCost: 2.0, Path: []int{0, 1}},
			{From: 1, To: 2, TravelCost: 2.1, Path: []int{1, 2}},
		},
		Regions: []SettlementRegion{
			{ID: 0, NodeIndices: []int{0, 1}, CenterNode: 0, MeanScore: 0.59, River: true},
			{ID: 1, NodeIndices: []int{2}, CenterNode: 2, MeanScore: 0.69, River: true},
		},
	}
	proto := &ProtoCivilizationResult{
		Civilizations: []ProtoCivilization{
			{ID: 0, RegionID: 0, CenterNode: 0, CenterKind: SettlementNodeTown, TerritoryCells: 28, MeanSupport: 0.40},
			{ID: 1, RegionID: 1, CenterNode: 2, CenterKind: SettlementNodeTown, TerritoryCells: 24, MeanSupport: 0.36},
		},
	}
	riverRoutes := &RiverRouteResult{
		Mode: DefaultRiverRouteSettings().Modes[0],
		Diagnostics: &RiverRouteDiagnostics{
			Navigability:         []float64{0.78, 0.82, 0.80},
			MainChannel:          []float64{0.84, 0.88, 0.86},
			TransferSupport:      []float64{0.42, 0.38, 0.44},
			PortageSuitability:   []float64{0.22, 0.24, 0.20},
			UpstreamTravelCost:   []float64{1.2, 1.1, 1.2},
			DownstreamTravelCost: []float64{0.9, 0.8, 0.9},
		},
	}

	result := BuildRiverTradeNetwork(cells, network, proto, riverRoutes, []float64{0.5, 0.3, 0.1}, DefaultRiverTradeSettings())
	if len(result.Corridors) == 0 {
		t.Fatalf("expected river trade corridor")
	}
	foundInter := false
	for _, corridor := range result.Corridors {
		if corridor.InterCivilization {
			foundInter = true
			if corridor.MeanNavigability <= 0 {
				t.Fatalf("expected inter-civilization river corridor to report mean navigability")
			}
			break
		}
	}
	if !foundInter {
		t.Fatalf("expected at least one inter-civilization river corridor")
	}
	if len(result.MajorPorts) == 0 {
		t.Fatalf("expected at least one major river port candidate")
	}
}

func TestBuildRiverTradeNetworkUsesNearbyRiverTerminalCells(t *testing.T) {
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2, 4}},
		{SiteIndex: 4, NeighborSiteIndices: []int32{3}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.66, CarryingCapacity: 0.58, UrbanPotential: 0.54, River: true},
			{ID: 1, CellIndex: 4, Kind: SettlementNodeTown, Score: 0.69, CarryingCapacity: 0.60, UrbanPotential: 0.56, River: true},
		},
		Regions: []SettlementRegion{
			{ID: 0, NodeIndices: []int{0}, CenterNode: 0, MeanScore: 0.59, River: true},
			{ID: 1, NodeIndices: []int{1}, CenterNode: 1, MeanScore: 0.69, River: true},
		},
	}
	proto := &ProtoCivilizationResult{
		Civilizations: []ProtoCivilization{
			{ID: 0, RegionID: 0, CenterNode: 0, CenterKind: SettlementNodeTown, TerritoryCells: 28, MeanSupport: 0.40},
			{ID: 1, RegionID: 1, CenterNode: 1, CenterKind: SettlementNodeTown, TerritoryCells: 24, MeanSupport: 0.36},
		},
	}
	riverRoutes := &RiverRouteResult{
		Mode: DefaultRiverRouteSettings().Modes[0],
		Diagnostics: &RiverRouteDiagnostics{
			Navigability:         []float64{0.0, 0.78, 0.82, 0.80, 0.0},
			MainChannel:          []float64{0.0, 0.84, 0.88, 0.86, 0.0},
			TransferSupport:      []float64{0.22, 0.42, 0.38, 0.44, 0.22},
			PortageSuitability:   []float64{0.18, 0.22, 0.24, 0.20, 0.18},
			UpstreamTravelCost:   []float64{0, 1.2, 1.1, 1.2, 0},
			DownstreamTravelCost: []float64{0, 0.9, 0.8, 0.9, 0},
		},
	}

	result := BuildRiverTradeNetwork(cells, network, proto, riverRoutes, []float64{0.5, 0.4, 0.3, 0.2, 0.1}, DefaultRiverTradeSettings())
	if len(result.Corridors) == 0 {
		t.Fatalf("expected river trade corridor through nearby terminal cells")
	}
	if result.Diagnostics.NodeTerminalCell[0] != 1 {
		t.Fatalf("expected node 0 to use nearby river terminal 1, got %d", result.Diagnostics.NodeTerminalCell[0])
	}
	if result.Diagnostics.NodeTerminalCell[1] != 3 {
		t.Fatalf("expected node 1 to use nearby river terminal 3, got %d", result.Diagnostics.NodeTerminalCell[1])
	}
}

func TestRiverTradeTerminalCatchmentStepsScaleWithResolution(t *testing.T) {
	if steps := riverTradeTerminalCatchmentSteps(10242); steps != 1 {
		t.Fatalf("expected level-5-ish catchment of 1 step, got %d", steps)
	}
	if steps := riverTradeTerminalCatchmentSteps(40962); steps != 2 {
		t.Fatalf("expected level-6-ish catchment of 2 steps, got %d", steps)
	}
}
