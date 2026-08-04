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

func TestBuildOceanTradeNetworkSeparatesInternalAndExternalPortCaps(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0.99, Y: 0.05, Z: 0},
		{X: 0.98, Y: 0.10, Z: 0},
		{X: 0.97, Y: 0.15, Z: 0},
		{X: 0.96, Y: 0.20, Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2, 4}},
		{SiteIndex: 4, NeighborSiteIndices: []int32{3}},
	}
	elevation := []float64{10, -1000, 10, -1000, 10}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.9, Coastal: true},
			{ID: 1, CellIndex: 2, Kind: SettlementNodeTown, Score: 0.9, Coastal: true},
			{ID: 2, CellIndex: 4, Kind: SettlementNodeTown, Score: 0.9, Coastal: true},
		},
		Regions: []SettlementRegion{
			{ID: 0, NodeIndices: []int{0, 1}, CenterNode: 0},
			{ID: 1, NodeIndices: []int{2}, CenterNode: 2},
		},
	}
	proto := &ProtoCivilizationResult{Civilizations: []ProtoCivilization{
		{ID: 0, RegionID: 0, CenterNode: 0},
		{ID: 1, RegionID: 1, CenterNode: 2},
	}}
	ports := &CoastalPortResult{
		Mode: MaritimeVesselSettings{
			Name:                 "bluewater-test",
			PayloadCapacity:      0.8,
			DailyRange:           0.9,
			LongHaulTolerance:    0.9,
			OpenOceanCapability:  0.9,
			MaxOpenWaterLeg:      0.6,
			StormTolerance:       0.8,
			SeasonalityTolerance: 0.8,
		},
		MajorDeepwaterPorts: []int{0, 1, 2},
		Diagnostics: &CoastalPortDiagnostics{
			NodeDeepwaterScore:    []float64{0.9, 0.9, 0.9},
			NodeDeepwaterTermCell: []int{0, 2, 4},
			DeepwaterSuitability:  []float64{0.8, 0, 0.8, 0, 0.8},
			PortSuitability:       []float64{0.8, 0, 0.8, 0, 0.8},
			StopoverValue:         []float64{0, 0, 0, 0, 0},
		},
	}
	settings := DefaultOceanTradeSettings()
	settings.MinFlow = 0.001
	settings.MaxRouteCost = 200
	settings.BaseLegCost = 20
	settings.LegScale = 120
	settings.MaxCandidatePortsPerCiv = 2
	settings.MaxPartnersPerPort = 1
	settings.MaxPartnersPerCivilization = 3

	result := BuildOceanTradeNetwork(sites, cells, nil, network, proto, ports, elevation, 0, settings)
	internal := 0
	external := 0
	for _, corridor := range result.Corridors {
		if corridor.FromCivilization == corridor.ToCivilization {
			internal++
		} else {
			external++
		}
	}
	if internal == 0 || external == 0 {
		t.Fatalf("expected separate internal and external ocean routes, got internal=%d external=%d corridors=%+v diag=%+v", internal, external, result.Corridors, result.PairDiagnostics)
	}
}

func TestCandidateOceanPortsRequiresPhysicalDeepwaterScore(t *testing.T) {
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.9, Coastal: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, Score: 0.9, Coastal: true},
		},
	}
	ports := &CoastalPortResult{
		MajorDeepwaterPorts: []int{0},
		Diagnostics: &CoastalPortDiagnostics{
			NodeDeepwaterScore:     []float64{0.90, 0.82},
			NodeBaseDeepwaterScore: []float64{0.20, 0.52},
			NodeDeepwaterTermCell:  []int{0, 1},
		},
	}
	settings := DefaultOceanTradeSettings()
	settings.CandidatePortThreshold = 0.56
	settings.CandidateSecondaryPortFloor = 0.48
	settings.CandidatePhysicalDeepwaterFloor = 0.48

	candidates := candidateOceanPorts(network, ports, settings)
	if len(candidates) != 1 || candidates[0] != 1 {
		t.Fatalf("expected physically viable secondary candidate only, got %v", candidates)
	}
}

func TestCandidateOceanPortsRequiresSettlementSupportWhenRecorded(t *testing.T) {
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, PhysicalSupportArea: 0.06, Score: 0.9, Coastal: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeVillage, PhysicalSupportArea: 0.75, Score: 0.9, Coastal: true},
		},
	}
	ports := &CoastalPortResult{
		Diagnostics: &CoastalPortDiagnostics{
			NodeDeepwaterScore:     []float64{0.82, 0.82},
			NodeBaseDeepwaterScore: []float64{0.52, 0.52},
			NodeDeepwaterTermCell:  []int{0, 1},
		},
	}
	settings := DefaultOceanTradeSettings()
	settings.CandidateSecondaryPortFloor = 0.48
	settings.CandidatePhysicalDeepwaterFloor = 0.48

	candidates := candidateOceanPorts(network, ports, settings)
	if len(candidates) != 1 || candidates[0] != 1 {
		t.Fatalf("expected only physically supported ocean candidate, got %v", candidates)
	}
}

func TestCandidateOceanPortsRequiresSettlementSupportForMajorPorts(t *testing.T) {
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, PhysicalSupportArea: 0.06, Score: 0.9, Coastal: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, PhysicalSupportArea: 0.75, Score: 0.9, Coastal: true},
		},
	}
	ports := &CoastalPortResult{
		MajorDeepwaterPorts: []int{0, 1},
		Diagnostics: &CoastalPortDiagnostics{
			NodeDeepwaterScore:     []float64{0.90, 0.82},
			NodeBaseDeepwaterScore: []float64{0.52, 0.52},
			NodeDeepwaterTermCell:  []int{0, 1},
		},
	}
	settings := DefaultOceanTradeSettings()
	settings.CandidatePortThreshold = 0.56
	settings.CandidatePhysicalDeepwaterFloor = 0.48

	candidates := candidateOceanPorts(network, ports, settings)
	if len(candidates) != 1 || candidates[0] != 1 {
		t.Fatalf("expected only physically supported major ocean candidate, got %v", candidates)
	}
}

func TestSelectOceanStopoversScalesSpacingByMeshResolution(t *testing.T) {
	cellCount := 40962
	cells := make([]VoronoiCell, cellCount)
	for i := range cells {
		cells[i].SiteIndex = int32(i)
	}
	for i := 0; i < 6; i++ {
		cells[i].NeighborSiteIndices = []int32{int32(i + 1)}
	}
	cells[6].NeighborSiteIndices = []int32{5}
	for i := 1; i < 6; i++ {
		cells[i].NeighborSiteIndices = []int32{int32(i - 1), int32(i + 1)}
	}
	stopovers := []MaritimeStopoverNode{
		{ID: 0, CellIndex: 0, Score: 0.9},
		{ID: 1, CellIndex: 6, Score: 0.8},
	}
	settings := DefaultOceanTradeSettings()
	settings.StopoverSpacingHops = 4
	settings.StopoverScoreFloor = 0.1

	selected, _ := selectOceanStopovers(stopovers, nil, cells, nil, settings, MaritimeStopoverDiagnostics{BaseSelectedCount: len(stopovers)})
	if len(selected) != 1 || selected[0].CellIndex != 0 {
		t.Fatalf("expected scaled ocean stopover spacing to keep only cell 0, got %+v", selected)
	}
}

func TestCapOceanCandidatePortsByCivilizationLimitsDensePortClusters(t *testing.T) {
	candidates := []int{0, 1, 2, 3, 4}
	civByNode := []int{0, 0, 0, 1, 1}
	capped, rejected := capOceanCandidatePortsByCivilization(candidates, civByNode, 2)
	if rejected != 1 {
		t.Fatalf("expected one redundant candidate rejected, got %d", rejected)
	}
	want := []int{0, 1, 3, 4}
	if len(capped) != len(want) {
		t.Fatalf("expected %d candidates, got %+v", len(want), capped)
	}
	for i := range want {
		if capped[i] != want[i] {
			t.Fatalf("candidate order changed: got %+v want %+v", capped, want)
		}
	}
}
