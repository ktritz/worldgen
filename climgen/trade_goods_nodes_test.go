package climgen

import "testing"

func TestComputeNodeGoodsFavorsUrbanNodesForProcessedGoods(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "iron_ore", Category: "raw", BaseValue: 0.58, Bulkiness: 0.95, SourceWeights: map[string]float64{"iron_ore": 1}},
			{Name: "iron_goods", Category: "finished", BaseValue: 0.72, Bulkiness: 0.42, Inputs: map[string]float64{"iron_ore": 0.5}},
		},
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "iron_ore", Category: "raw", Potential: []float64{0.85, 0.80, 0.10}},
			{Good: "iron_goods", Category: "finished", Potential: []float64{0.00, 0.00, 0.00}},
		},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, CarryingCapacity: 0.42, UrbanPotential: 0.26},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, CarryingCapacity: 0.46, UrbanPotential: 0.60},
		},
	}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{{ID: 0, CapitalNode: 1}},
		Diagnostics: &PolitySphereDiagnostics{
			PolityByCell: []int{0, 0, -1},
			PolityByNode: []int{-1, 0},
		},
	}
	profiles := &PolityProfileResult{Assignments: []PolityProfileAssignment{
		{PolityID: 0, Profile: ResolvedProfile{AncestryName: "Dwarf"}},
	}}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.08, 0.30}}}

	result := ComputeNodeGoods(cells, goods, settings, polities, profiles, network, trade)
	if len(result.Balances) != 2 {
		t.Fatalf("expected two node balances, got %d", len(result.Balances))
	}
	if result.Balances[1].Supply["iron_goods"] <= result.Balances[0].Supply["iron_goods"] {
		t.Fatalf("expected town to lead iron goods supply, got village=%.3f town=%.3f", result.Balances[0].Supply["iron_goods"], result.Balances[1].Supply["iron_goods"])
	}
}

func TestComputeNodeGoodsFollowsLocalCatchmentForRawGoods(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.42, Bulkiness: 0.86, SourceWeights: map[string]float64{"crop": 1}},
		},
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "grain", Category: "raw", Potential: []float64{0.88, 0.20, 0.10}},
		},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, CarryingCapacity: 0.40, UrbanPotential: 0.20},
			{ID: 1, CellIndex: 2, Kind: SettlementNodeTown, CarryingCapacity: 0.52, UrbanPotential: 0.56},
		},
	}

	result := ComputeNodeGoods(cells, goods, settings, nil, nil, network, nil)
	if len(result.Balances) != 2 {
		t.Fatalf("expected two node balances, got %d", len(result.Balances))
	}
	if result.Balances[0].Supply["grain"] <= result.Balances[1].Supply["grain"] {
		t.Fatalf("expected farm catchment to outrank urban node for grain, got rural=%.3f urban=%.3f", result.Balances[0].Supply["grain"], result.Balances[1].Supply["grain"])
	}
}

func TestComputeNodeGoodsAssignsHigherWealthToTradeConnectedTown(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.42, Bulkiness: 0.86, SourceWeights: map[string]float64{"crop": 1}},
			{Name: "precious_ore", Category: "raw", BaseValue: 0.82, Bulkiness: 0.22, SourceWeights: map[string]float64{"gold_ore": 1}},
		},
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "grain", Category: "raw", Potential: []float64{0.60, 0.40, 0.10}},
			{Good: "precious_ore", Category: "raw", Potential: []float64{0.10, 0.55, 0.10}},
		},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, CarryingCapacity: 0.42, UrbanPotential: 0.22},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, CarryingCapacity: 0.52, UrbanPotential: 0.66, Coastal: true},
		},
	}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.04, 0.32}}}

	result := ComputeNodeGoods(cells, goods, settings, nil, nil, network, trade)
	if result.Balances[1].Wealth <= result.Balances[0].Wealth {
		t.Fatalf("expected connected town to be wealthier, got village=%.3f town=%.3f", result.Balances[0].Wealth, result.Balances[1].Wealth)
	}
}

func TestComputeNodeGoodsRawCatchmentSpecializationWidensDominantResourceGap(t *testing.T) {
	baseSettings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:             map[string]float64{"raw": 1.0, "default": 1.0},
			CategorySpecializationScale:     map[string]float64{"raw": 0.0, "default": 0.0},
			RawCatchmentSpecializationScale: 0.0,
			RawCatchmentSpecializationFloor: 1.0,
			RawPotentialPivot:               0.50,
			ManufacturingPivot:              0.50,
		},
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.42, Bulkiness: 0.86, SourceWeights: map[string]float64{"crop": 1}},
			{Name: "herbs", Category: "raw", BaseValue: 0.64, Bulkiness: 0.24, SourceWeights: map[string]float64{"herbs": 1}},
			{Name: "timber", Category: "raw", BaseValue: 0.42, Bulkiness: 0.90, SourceWeights: map[string]float64{"timber": 1}},
		},
	}
	specializedSettings := baseSettings
	specializedSettings.Production = TradeGoodsProductionSettings{
		CategorySupplyScale:             map[string]float64{"raw": 1.0, "default": 1.0},
		CategorySpecializationScale:     map[string]float64{"raw": 0.0, "default": 0.0},
		RawCatchmentSpecializationScale: 0.80,
		RawCatchmentSpecializationFloor: 0.65,
		RawPotentialPivot:               0.50,
		ManufacturingPivot:              0.50,
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "grain", Category: "raw", Potential: []float64{0.52, 0.26, 0.10}},
			{Good: "herbs", Category: "raw", Potential: []float64{0.96, 0.05, 0.02}},
			{Good: "timber", Category: "raw", Potential: []float64{0.18, 0.14, 0.08}},
		},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, CarryingCapacity: 0.42, UrbanPotential: 0.24},
		},
	}

	base := ComputeNodeGoods(cells, goods, baseSettings, nil, nil, network, nil)
	specialized := ComputeNodeGoods(cells, goods, specializedSettings, nil, nil, network, nil)
	baseGap := base.Balances[0].Supply["herbs"] - base.Balances[0].Supply["grain"]
	specializedGap := specialized.Balances[0].Supply["herbs"] - specialized.Balances[0].Supply["grain"]
	if specializedGap <= baseGap {
		t.Fatalf("expected raw catchment specialization to widen dominant-resource gap, got baseGap=%.3f specializedGap=%.3f", baseGap, specializedGap)
	}
}

func TestComputeNodeGoodsRawCatchmentSensitivityAmplifiesSpecialistRaw(t *testing.T) {
	baseSettings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:             map[string]float64{"raw": 1.0, "default": 1.0},
			CategorySpecializationScale:     map[string]float64{"raw": 0.0, "default": 0.0},
			RawCatchmentSpecializationScale: 0.75,
			RawCatchmentSpecializationFloor: 0.68,
			RawPotentialPivot:               0.50,
			ManufacturingPivot:              0.50,
		},
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.42, Bulkiness: 0.86, SourceWeights: map[string]float64{"crop": 1}},
			{Name: "herbs", Category: "raw", BaseValue: 0.64, Bulkiness: 0.24, SourceWeights: map[string]float64{"herbs": 1}},
			{Name: "timber", Category: "raw", BaseValue: 0.42, Bulkiness: 0.90, SourceWeights: map[string]float64{"timber": 1}},
		},
	}
	tunedSettings := baseSettings
	tunedSettings.Goods = []TradeGoodSpec{
		{Name: "grain", Category: "raw", BaseValue: 0.42, Bulkiness: 0.86, RawCatchmentSensitivity: 0.72, SourceWeights: map[string]float64{"crop": 1}},
		{Name: "herbs", Category: "raw", BaseValue: 0.64, Bulkiness: 0.24, RawCatchmentSensitivity: 1.40, SourceWeights: map[string]float64{"herbs": 1}},
		{Name: "timber", Category: "raw", BaseValue: 0.42, Bulkiness: 0.90, RawCatchmentSensitivity: 0.90, SourceWeights: map[string]float64{"timber": 1}},
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "grain", Category: "raw", Potential: []float64{0.70, 0.28, 0.12}},
			{Good: "herbs", Category: "raw", Potential: []float64{0.92, 0.06, 0.02}},
			{Good: "timber", Category: "raw", Potential: []float64{0.25, 0.18, 0.10}},
		},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, CarryingCapacity: 0.42, UrbanPotential: 0.24},
		},
	}

	base := ComputeNodeGoods(cells, goods, baseSettings, nil, nil, network, nil)
	tuned := ComputeNodeGoods(cells, goods, tunedSettings, nil, nil, network, nil)
	baseGap := base.Balances[0].Supply["herbs"] - base.Balances[0].Supply["grain"]
	tunedGap := tuned.Balances[0].Supply["herbs"] - tuned.Balances[0].Supply["grain"]
	if tunedGap <= baseGap {
		t.Fatalf("expected raw catchment sensitivity to favor specialist raw, got baseGap=%.3f tunedGap=%.3f", baseGap, tunedGap)
	}
}

func TestComputeNodeGoodsProductionDriversFavorCoastalRawSupply(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:             map[string]float64{"raw": 1.0, "default": 1.0},
			CategorySpecializationScale:     map[string]float64{"raw": 0.0, "default": 0.0},
			RawCatchmentSpecializationScale: 0.0,
			RawCatchmentSpecializationFloor: 1.0,
			RawPotentialPivot:               0.50,
			ManufacturingPivot:              0.50,
		},
		Goods: []TradeGoodSpec{
			{
				Name:              "fish",
				Category:          "raw",
				BaseValue:         0.36,
				Bulkiness:         0.62,
				Perishability:     0.78,
				SourceWeights:     map[string]float64{"fish": 1},
				ProductionDrivers: map[string]float64{"coastal": 0.8, "river": 0.3},
			},
		},
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "fish", Category: "raw", Potential: []float64{0.72, 0.72, 0.10}},
		},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, CarryingCapacity: 0.42, UrbanPotential: 0.24, Coastal: false},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeVillage, CarryingCapacity: 0.42, UrbanPotential: 0.24, Coastal: true},
		},
	}

	result := ComputeNodeGoods(cells, goods, settings, nil, nil, network, nil)
	if len(result.Balances) != 2 {
		t.Fatalf("expected two node balances, got %d", len(result.Balances))
	}
	if result.Balances[1].Supply["fish"] <= result.Balances[0].Supply["fish"] {
		t.Fatalf("expected coastal node to lead fish supply, got inland=%.3f coastal=%.3f", result.Balances[0].Supply["fish"], result.Balances[1].Supply["fish"])
	}
}
