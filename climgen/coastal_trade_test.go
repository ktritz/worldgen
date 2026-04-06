package climgen

import "testing"

func TestBuildCoastalTradeNetworkConnectsNearbyCoastalPorts(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0.98, Y: 0.05, Z: 0},
		{X: 0.96, Y: 0.10, Z: 0},
		{X: 0.94, Y: 0.15, Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2}},
	}
	elevation := []float64{10, -10, -10, 10}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.7, Coastal: true},
			{ID: 1, CellIndex: 3, Kind: SettlementNodeTown, Score: 0.72, Coastal: true},
		},
	}
	proto := &ProtoCivilizationResult{
		Civilizations: []ProtoCivilization{
			{ID: 0, RegionID: 0, CenterNode: 0},
			{ID: 1, RegionID: 1, CenterNode: 1},
		},
	}
	network.Regions = []SettlementRegion{
		{ID: 0, NodeIndices: []int{0}, CenterNode: 0},
		{ID: 1, NodeIndices: []int{1}, CenterNode: 1},
	}
	ports := &CoastalPortResult{
		Mode: MaritimeVesselSettings{
			Name:                  "coastal-sloop",
			TechLevel:             "medieval",
			Propulsion:            "sail",
			RouteClass:            "coastal",
			PayloadCapacity:       0.46,
			DailyRange:            0.72,
			LongHaulTolerance:     0.46,
			CoastalCapability:     0.88,
			MaxCoastalLeg:         0.28,
			CurrentAssist:         0.34,
			AdverseCurrentPenalty: 0.30,
			WindAssist:            0.54,
			UpwindPenalty:         0.34,
			StormTolerance:        0.38,
			SeasonalityTolerance:  0.50,
		},
		MajorPorts: []int{0, 1},
		Diagnostics: &CoastalPortDiagnostics{
			NodePortScore:    []float64{0.8, 0.82},
			NodeTerminalCell: []int{0, 3},
			PortSuitability:  []float64{0.72, 0.74, 0, 0},
		},
	}
	result := BuildCoastalTradeNetwork(sites, cells, nil, network, proto, ports, elevation, 0.0, DefaultCoastalTradeSettings())
	if len(result.Corridors) == 0 {
		t.Fatalf("expected coastal trade corridor between nearby coastal ports")
	}
}

func TestCandidateCoastalPortsIncludesStrongDistrictTerminal(t *testing.T) {
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, Score: 0.42, Coastal: true, River: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeHamlet, Score: 0.34, Coastal: true},
		},
	}
	ports := &CoastalPortResult{
		Diagnostics: &CoastalPortDiagnostics{
			NodePortScore:    []float64{0.30, 0.30},
			NodeTerminalCell: []int{0, 1},
			PortSuitability:  []float64{0.34, 0.34},
			EstuaryAccess:    []float64{0.36, 0.10},
			RiverTransfer:    []float64{0.30, 0.08},
			StopoverValue:    []float64{0.16, 0.12},
		},
	}

	candidates := candidateCoastalPorts(network, ports, DefaultCoastalTradeSettings())
	if len(candidates) != 1 || candidates[0] != 0 {
		t.Fatalf("expected only strong district coastal terminal to qualify, got %v", candidates)
	}
}

func TestCandidateCoastalPortsIncludesStrongRiverEstuaryTerminal(t *testing.T) {
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, Score: 0.44, Coastal: false, River: true},
		},
	}
	ports := &CoastalPortResult{
		Diagnostics: &CoastalPortDiagnostics{
			NodePortScore:    []float64{0.45},
			NodeTerminalCell: []int{0},
			PortSuitability:  []float64{0.22},
			EstuaryAccess:    []float64{0.31},
			RiverTransfer:    []float64{0.33},
			StopoverValue:    []float64{0.10},
		},
	}

	candidates := candidateCoastalPorts(network, ports, DefaultCoastalTradeSettings())
	if len(candidates) != 1 || candidates[0] != 0 {
		t.Fatalf("expected strong river estuary terminal to qualify, got %v", candidates)
	}
}

func TestBuildCoastalTradeNetworkCanUseShortOpenWaterHop(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0.99, Y: 0.02, Z: 0},
		{X: 0.98, Y: 0.03, Z: 0},
		{X: 0.97, Y: 0.04, Z: 0},
		{X: 0.96, Y: 0.05, Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2, 4}},
		{SiteIndex: 4, NeighborSiteIndices: []int32{3}},
	}
	elevation := []float64{10, -10, -10, -10, 10}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.7, Coastal: true},
			{ID: 1, CellIndex: 4, Kind: SettlementNodeTown, Score: 0.72, Coastal: true},
		},
	}
	proto := &ProtoCivilizationResult{
		Civilizations: []ProtoCivilization{
			{ID: 0, RegionID: 0, CenterNode: 0},
			{ID: 1, RegionID: 1, CenterNode: 1},
		},
	}
	network.Regions = []SettlementRegion{
		{ID: 0, NodeIndices: []int{0}, CenterNode: 0},
		{ID: 1, NodeIndices: []int{1}, CenterNode: 1},
	}
	ports := &CoastalPortResult{
		Mode: MaritimeVesselSettings{
			Name:                  "coastal-sloop",
			TechLevel:             "medieval",
			Propulsion:            "sail",
			RouteClass:            "coastal",
			PayloadCapacity:       0.46,
			DailyRange:            0.72,
			LongHaulTolerance:     0.46,
			CoastalCapability:     0.88,
			OpenOceanCapability:   0.24,
			MaxCoastalLeg:         0.28,
			MaxOpenWaterLeg:       0.10,
			CurrentAssist:         0.34,
			AdverseCurrentPenalty: 0.30,
			WindAssist:            0.54,
			UpwindPenalty:         0.34,
			StormTolerance:        0.38,
			SeasonalityTolerance:  0.50,
			StopoverNeed:          0.62,
		},
		MajorPorts: []int{0, 1},
		Diagnostics: &CoastalPortDiagnostics{
			NodePortScore:    []float64{0.8, 0.82},
			NodeTerminalCell: []int{0, 4},
			PortSuitability:  []float64{0.72, 0, 0, 0, 0.74},
		},
	}

	result := BuildCoastalTradeNetwork(sites, cells, nil, network, proto, ports, elevation, 0.0, DefaultCoastalTradeSettings())
	if len(result.Corridors) == 0 {
		t.Fatalf("expected coastal trade corridor across short open-water hop")
	}
}
