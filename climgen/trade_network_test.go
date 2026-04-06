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

	result := BuildTradeNetwork(make([]VoronoiCell, 3), network, proto, nil, DefaultTradeNetworkSettings())
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

func TestBuildTradeNetworkAnnotatesLandRouteModeAndRisk(t *testing.T) {
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.66},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, Score: 0.64},
		},
		Links: []SettlementLink{
			{From: 0, To: 1, TravelCost: 4.0, Path: []int{0, 1}},
		},
		Regions: []SettlementRegion{
			{ID: 0, NodeIndices: []int{0}, CenterNode: 0, MeanScore: 0.66},
			{ID: 1, NodeIndices: []int{1}, CenterNode: 1, MeanScore: 0.64},
		},
	}
	proto := &ProtoCivilizationResult{
		Civilizations: []ProtoCivilization{
			{ID: 0, RegionID: 0, CenterNode: 0, CenterKind: SettlementNodeTown, TerritoryCells: 40, MeanSupport: 0.40},
			{ID: 1, RegionID: 1, CenterNode: 1, CenterKind: SettlementNodeTown, TerritoryCells: 32, MeanSupport: 0.38},
		},
	}
	landRoutes := &LandRouteResult{
		Mode: LandRouteModeSettings{
			Name:                  "pack-lizard",
			PayloadCapacity:       0.7,
			DailyRange:            1.0,
			LongHaulTolerance:     0.8,
			InterCivilizationFlow: 1.0,
			InternalFlow:          1.0,
		},
		Diagnostics: &LandRouteDiagnostics{
			ModeCost:              []float64{1.2, 1.4},
			RouteRisk:             []float64{0.3, 0.5},
			WaystationSuitability: []float64{0.6, 0.4},
		},
	}
	result := BuildTradeNetwork(make([]VoronoiCell, 2), network, proto, landRoutes, DefaultTradeNetworkSettings())
	if len(result.Corridors) == 0 {
		t.Fatalf("expected corridor with land route diagnostics")
	}
	if result.Corridors[0].Mode != "pack-lizard" {
		t.Fatalf("expected corridor mode to reflect land route mode, got %q", result.Corridors[0].Mode)
	}
	if result.Corridors[0].MeanRisk <= 0 || result.Corridors[0].MeanSupport <= 0 {
		t.Fatalf("expected corridor risk/support diagnostics to be populated, got risk=%.2f support=%.2f", result.Corridors[0].MeanRisk, result.Corridors[0].MeanSupport)
	}
	if result.Corridors[0].Role != TradeCorridorRoleInterPolityTrunk {
		t.Fatalf("expected inter-civilization corridor to be trunk role, got %s", TradeCorridorRoleName(result.Corridors[0].Role))
	}
}

func TestBuildTradeNetworkCreatesPorterFeederCorridor(t *testing.T) {
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeHamlet, Score: 0.42},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeVillage, Score: 0.54},
			{ID: 2, CellIndex: 2, Kind: SettlementNodeTown, Score: 0.66, CarryingCapacity: 0.58, UrbanPotential: 0.52},
		},
		Links: []SettlementLink{
			{From: 0, To: 1, TravelCost: 2.0, Path: []int{0, 1}},
			{From: 1, To: 2, TravelCost: 2.4, Path: []int{1, 2}},
		},
		Regions: []SettlementRegion{
			{ID: 0, NodeIndices: []int{0, 1, 2}, CenterNode: 2, MeanScore: 0.54},
		},
	}
	proto := &ProtoCivilizationResult{
		Civilizations: []ProtoCivilization{
			{ID: 0, RegionID: 0, CenterNode: 2, CenterKind: SettlementNodeTown, TerritoryCells: 12, MeanSupport: 0.34},
		},
	}
	landRoutes := &LandRouteResult{
		Mode: LandRouteModeSettings{
			Name:                  "porter",
			PayloadCapacity:       0.22,
			DailyRange:            0.78,
			LongHaulTolerance:     0.18,
			InterCivilizationFlow: 0.06,
			InternalFlow:          0.48,
			FeederFlow:            0.95,
			FeederReach:           0.42,
		},
		Diagnostics: &LandRouteDiagnostics{
			ModeCost:              []float64{1.0, 1.0, 1.0},
			RouteRisk:             []float64{0.1, 0.1, 0.1},
			WaystationSuitability: []float64{0.6, 0.6, 0.6},
		},
	}

	result := BuildTradeNetwork(make([]VoronoiCell, 2), network, proto, landRoutes, DefaultTradeNetworkSettings())
	if len(result.Corridors) == 0 {
		t.Fatalf("expected porter mode to produce local feeder corridor")
	}
	if result.Corridors[0].Mode != "porter" {
		t.Fatalf("expected porter feeder corridor mode, got %q", result.Corridors[0].Mode)
	}
	if result.Corridors[0].InterCivilization {
		t.Fatalf("expected porter feeder corridor to be local/internal")
	}
	if result.Corridors[0].Role != TradeCorridorRoleFeeder {
		t.Fatalf("expected porter corridor to be feeder role, got %s", TradeCorridorRoleName(result.Corridors[0].Role))
	}
	if len(result.Handoffs) == 0 {
		t.Fatalf("expected feeder corridor to produce handoff record")
	}
}

func TestBuildTradeNetworkCreatesLocalGraphFeederCorridor(t *testing.T) {
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 2, Kind: SettlementNodeTown, Score: 0.68, CarryingCapacity: 0.60, UrbanPotential: 0.56},
		},
		Regions: []SettlementRegion{
			{ID: 0, NodeIndices: []int{0}, CenterNode: 0, MeanScore: 0.68},
		},
	}
	proto := &ProtoCivilizationResult{
		Civilizations: []ProtoCivilization{
			{ID: 0, RegionID: 0, CenterNode: 0, CenterKind: SettlementNodeTown, TerritoryCells: 18, MeanSupport: 0.38},
		},
	}
	landRoutes := &LandRouteResult{
		Mode: LandRouteModeSettings{
			Name:                  "pack-lizard",
			PayloadCapacity:       0.46,
			DailyRange:            0.88,
			LongHaulTolerance:     0.32,
			InterCivilizationFlow: 0.08,
			InternalFlow:          0.44,
			FeederFlow:            0.92,
			FeederReach:           0.48,
		},
		Diagnostics: &LandRouteDiagnostics{
			ModeCost:              []float64{1.4, 1.2, 1.0},
			RouteRisk:             []float64{0.10, 0.12, 0.08},
			WaterSupport:          []float64{0.70, 0.60, 0.40},
			ForageSupport:         []float64{0.65, 0.45, 0.30},
			WaystationSuitability: []float64{0.58, 0.36, 0.28},
			RoadQuality:           []float64{0.18, 0.22, 0.42},
			CrossingPressure:      []float64{0.12, 0.18, 0.08},
			BridgeProxy:           []float64{0.02, 0.04, 0.03},
			FordProxy:             []float64{0.03, 0.06, 0.02},
		},
	}

	result := BuildTradeNetwork(cells, network, proto, landRoutes, DefaultTradeNetworkSettings())
	if len(result.LocalNodes) == 0 {
		t.Fatalf("expected local trade graph to create feeder nodes")
	}
	foundLocalFeeder := false
	for _, corridor := range result.Corridors {
		if corridor.Role != TradeCorridorRoleFeeder || corridor.FromLocalNode < 0 {
			continue
		}
		foundLocalFeeder = true
		if corridor.HandoffNode != 0 {
			t.Fatalf("expected feeder handoff into anchor node 0, got %d", corridor.HandoffNode)
		}
		if len(corridor.CellPath) < 2 {
			t.Fatalf("expected local feeder corridor to include cell path, got %v", corridor.CellPath)
		}
	}
	if !foundLocalFeeder {
		t.Fatalf("expected at least one local-graph feeder corridor")
	}
}
