package climgen

import "testing"

func TestBuildOceanTradeNetworkConnectsDeepwaterPortsForBlueWaterVessel(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0.7, Y: 0.7, Z: 0},
		{X: 0, Y: 1, Z: 0},
		{X: -0.7, Y: 0.7, Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2}},
	}
	elevation := []float64{10, -1000, -1200, 10}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.8, Coastal: true},
			{ID: 1, CellIndex: 3, Kind: SettlementNodeTown, Score: 0.8, Coastal: true},
		},
		Regions: []SettlementRegion{
			{ID: 0, NodeIndices: []int{0}},
			{ID: 1, NodeIndices: []int{1}},
		},
	}
	proto := &ProtoCivilizationResult{
		Civilizations: []ProtoCivilization{
			{ID: 0, RegionID: 0, CenterNode: 0},
			{ID: 1, RegionID: 1, CenterNode: 1},
		},
	}
	ports := &CoastalPortResult{
		Mode: MaritimeVesselSettings{
			Name:                 "bluewater-test",
			TechLevel:            "test",
			Propulsion:           "sail",
			RouteClass:           "blue-water",
			PayloadCapacity:      0.8,
			DailyRange:           0.9,
			LongHaulTolerance:    0.9,
			OpenOceanCapability:  0.9,
			MaxOpenWaterLeg:      0.6,
			StormTolerance:       0.8,
			SeasonalityTolerance: 0.8,
		},
		MajorDeepwaterPorts: []int{0, 1},
		Diagnostics: &CoastalPortDiagnostics{
			NodeDeepwaterScore:    []float64{0.9, 0.9},
			NodeDeepwaterTermCell: []int{0, 3},
			DeepwaterSuitability:  []float64{0.8, 0, 0, 0.8},
			PortSuitability:       []float64{0.8, 0, 0, 0.8},
			StopoverValue:         []float64{0, 0, 0, 0},
		},
	}
	settings := DefaultOceanTradeSettings()
	settings.MinFlow = 0.001
	settings.MaxRouteCost = 200
	settings.BaseLegCost = 20
	settings.LegScale = 120
	result := BuildOceanTradeNetwork(sites, cells, nil, network, proto, ports, elevation, 0, settings)
	if len(result.Corridors) != 1 {
		t.Fatalf("expected one ocean corridor, got %d diagnostics=%+v", len(result.Corridors), result.PairDiagnostics)
	}
	if !result.Corridors[0].InterCivilization {
		t.Fatalf("expected inter-civilization ocean corridor")
	}
}
