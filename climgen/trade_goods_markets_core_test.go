package climgen

import "testing"

func TestComputeTradeNodeMarketsAggregatesLocalFeederSupplyIntoHandoff(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.42, Bulkiness: 0.86, SourceWeights: map[string]float64{"crop": 1}},
		},
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "grain", Category: "raw", Potential: []float64{0.15, 0.70, 0.20}},
		},
	}
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{{
			NodeID:   0,
			PolityID: 0,
			Wealth:   0.30,
			Supply:   map[string]float64{"grain": 0.12},
			Demand:   map[string]float64{"grain": 0.10},
			Surplus:  map[string]float64{"grain": 0.02},
			Exports:  []PolityGoodValue{{Good: "grain", Value: 0.02}},
		}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown},
		},
	}
	trade := &TradeNetworkResult{
		LocalNodes: []LocalTradeNode{{
			ID:          0,
			CellIndex:   1,
			HandoffNode: 0,
			Kind:        LocalTradeNodeWaystation,
			Score:       0.64,
			Support:     0.58,
			Waystation:  0.52,
		}},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1}},
	}

	result := ComputeTradeNodeMarkets(cells, goods, settings, nil, nil, network, trade, nodeGoods)
	if len(result.Markets) != 1 {
		t.Fatalf("expected one market, got %d", len(result.Markets))
	}
	market := result.Markets[0]
	if market.FeederNodes != 1 {
		t.Fatalf("expected one feeder node, got %d", market.FeederNodes)
	}
	if market.Supply["grain"] <= nodeGoods.Balances[0].Supply["grain"] {
		t.Fatalf("expected feeder supply to raise handoff grain supply, got base=%.3f total=%.3f", nodeGoods.Balances[0].Supply["grain"], market.Supply["grain"])
	}
}

func TestLocalTradeNodeContributionUsesWiderCatchmentForCompactRaws(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.42, Bulkiness: 0.86, SourceWeights: map[string]float64{"crop": 1}},
			{Name: "salt", Category: "raw", BaseValue: 0.62, Bulkiness: 0.38, Perishability: 0.0, SourceWeights: map[string]float64{"salt": 1}},
		},
	}
	endowmentByGood := map[string]TradeGoodEndowment{
		"grain": {Good: "grain", Category: "raw", Potential: []float64{0.00, 0.00, 0.72}},
		"salt":  {Good: "salt", Category: "raw", Potential: []float64{0.00, 0.00, 0.72}},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1}},
	}
	node := LocalTradeNode{
		ID:         0,
		CellIndex:  0,
		Kind:       LocalTradeNodeWaystation,
		Score:      0.62,
		Support:    0.56,
		Waystation: 0.48,
	}

	_, supply := localTradeNodeContribution(node, cells, settings, endowmentByGood, PolityProfileAssignment{}, settings.EffectiveProductionSettings())

	if supply["salt"] <= 0 {
		t.Fatalf("expected compact salt to pull from a wider feeder catchment, got %.3f", supply["salt"])
	}
	if supply["grain"] != 0 {
		t.Fatalf("expected bulky grain to stay out of the wider compact-only feeder catchment, got %.3f", supply["grain"])
	}
}

func TestComputeTradeNodeMarketsManufacturesProcessedGoodsFromPooledInputs(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.0, "finished": 1.0, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "finished": 0.5, "default": 0.0},
			ManufacturingBaseScale:      map[string]float64{"finished": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"finished": 1.05, "default": 0.55},
			MarketWorkshopBias:          map[string]float64{"finished": 1.25, "default": 1.0},
			MarketInputRichnessScale:    map[string]float64{"finished": 0.35, "default": 0.14},
			MarketConversionShare:       map[string]float64{"finished": 0.72, "default": 0.40},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.40,
		},
		Demand: TradeGoodsDemandSettings{
			CategoryDemandScale:         map[string]float64{"finished": 1.0, "default": 1.0},
			LocalSupplyReliefByCategory: map[string]float64{"finished": 0.0, "default": 0.0},
			DriverSpecializationScale:   map[string]float64{"finished": 0.0, "default": 0.0},
			MarketCategoryDemandScale:   map[string]float64{"finished": 1.10, "default": 1.0},
			MarketWealthPullScale:       map[string]float64{"finished": 0.25, "default": 0.0},
			MarketFeederPullScale:       map[string]float64{"finished": 0.20, "default": 0.0},
			DriverSpecializationPivot:   0.50,
		},
		Goods: []TradeGoodSpec{
			{Name: "iron_ore", Category: "raw", BaseValue: 0.58, Bulkiness: 0.95, SourceWeights: map[string]float64{"iron_ore": 1}},
			{Name: "iron_goods", Category: "finished", BaseValue: 0.72, Bulkiness: 0.42, Inputs: map[string]float64{"iron_ore": 0.5}},
		},
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "iron_ore", Category: "raw", Potential: []float64{0.15, 0.92, 0.20}},
			{Good: "iron_goods", Category: "finished", Potential: []float64{0.0, 0.0, 0.0}},
		},
	}
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{{
			NodeID:   0,
			PolityID: 0,
			Wealth:   0.52,
			Supply:   map[string]float64{"iron_ore": 0.22, "iron_goods": 0.08},
			Demand:   map[string]float64{"iron_ore": 0.10, "iron_goods": 0.14},
			Surplus:  map[string]float64{"iron_ore": 0.12, "iron_goods": -0.06},
		}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.58, UrbanPotential: 0.68},
		},
	}
	trade := &TradeNetworkResult{
		LocalNodes: []LocalTradeNode{{
			ID:          0,
			CellIndex:   1,
			HandoffNode: 0,
			Kind:        LocalTradeNodeCrossingDepot,
			Score:       0.72,
			Support:     0.64,
			Waystation:  0.60,
		}},
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.32}},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1}},
	}

	result := ComputeTradeNodeMarkets(cells, goods, settings, nil, nil, network, trade, nodeGoods)
	if len(result.Markets) != 1 {
		t.Fatalf("expected one market, got %d", len(result.Markets))
	}
	market := result.Markets[0]
	if market.Supply["iron_goods"] <= nodeGoods.Balances[0].Supply["iron_goods"] {
		t.Fatalf("expected pooled inputs to raise finished goods supply, got base=%.3f total=%.3f", nodeGoods.Balances[0].Supply["iron_goods"], market.Supply["iron_goods"])
	}
	if market.Supply["iron_ore"] >= 0.80 {
		t.Fatalf("expected market conversion to consume iron ore surplus, got iron_ore=%.3f", market.Supply["iron_ore"])
	}
	if market.Surplus["iron_goods"] <= 0 {
		t.Fatalf("expected market manufacturing to create iron goods export capacity, got surplus=%.3f", market.Surplus["iron_goods"])
	}
}

func TestComputeTradeNodeMarketsAppliesMarketDemandShaping(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Demand: TradeGoodsDemandSettings{
			CategoryDemandScale:         map[string]float64{"finished": 1.0, "default": 1.0},
			LocalSupplyReliefByCategory: map[string]float64{"finished": 0.0, "default": 0.0},
			DriverSpecializationScale:   map[string]float64{"finished": 0.0, "default": 0.0},
			MarketCategoryDemandScale:   map[string]float64{"finished": 1.20, "default": 1.0},
			MarketWealthPullScale:       map[string]float64{"finished": 0.30, "default": 0.0},
			MarketFeederPullScale:       map[string]float64{"finished": 0.25, "default": 0.0},
			DriverSpecializationPivot:   0.50,
		},
		Goods: []TradeGoodSpec{
			{Name: "iron_goods", Category: "finished", BaseValue: 0.72, Bulkiness: 0.42, Inputs: map[string]float64{"iron_ore": 0.5}},
		},
	}
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{{
			NodeID:   0,
			PolityID: 0,
			Wealth:   0.60,
			Supply:   map[string]float64{"iron_goods": 0.08},
			Demand:   map[string]float64{"iron_goods": 0.20},
			Surplus:  map[string]float64{"iron_goods": -0.12},
		}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.58, UrbanPotential: 0.68},
		},
	}
	trade := &TradeNetworkResult{
		LocalNodes: []LocalTradeNode{{
			ID:          0,
			CellIndex:   1,
			HandoffNode: 0,
			Kind:        LocalTradeNodeCrossingDepot,
			Score:       0.72,
			Support:     0.64,
			Waystation:  0.60,
		}},
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.32}},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0}},
	}

	result := ComputeTradeNodeMarkets(cells, &TradeGoodResult{}, settings, nil, nil, network, trade, nodeGoods)
	if len(result.Markets) != 1 {
		t.Fatalf("expected one market, got %d", len(result.Markets))
	}
	market := result.Markets[0]
	if market.Demand["iron_goods"] <= nodeGoods.Balances[0].Demand["iron_goods"] {
		t.Fatalf("expected market demand shaping to raise finished goods demand, got base=%.3f total=%.3f", nodeGoods.Balances[0].Demand["iron_goods"], market.Demand["iron_goods"])
	}
}

func TestComputeTradeNodeMarketsSupportsChainedManufacturing(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.0, "finished": 1.0, "luxury": 1.0, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "finished": 0.4, "luxury": 0.4, "default": 0.0},
			ManufacturingBaseScale:      map[string]float64{"finished": 0.55, "luxury": 0.50, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"finished": 1.00, "luxury": 1.00, "default": 0.55},
			MarketWorkshopBias:          map[string]float64{"finished": 1.20, "luxury": 1.18, "default": 1.0},
			MarketInputRichnessScale:    map[string]float64{"finished": 0.30, "luxury": 0.24, "default": 0.14},
			MarketConversionShare:       map[string]float64{"finished": 0.72, "luxury": 0.64, "default": 0.40},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "clay", Category: "raw", BaseValue: 0.28, Bulkiness: 0.96, SourceWeights: map[string]float64{"clay": 1}},
			{Name: "timber", Category: "raw", BaseValue: 0.42, Bulkiness: 0.90, SourceWeights: map[string]float64{"timber": 1}},
			{Name: "herbs", Category: "raw", BaseValue: 0.64, Bulkiness: 0.24, SourceWeights: map[string]float64{"herbs": 1}},
			{Name: "resin", Category: "raw", BaseValue: 0.56, Bulkiness: 0.42, SourceWeights: map[string]float64{"resin": 1}},
			{Name: "ceramics", Category: "finished", BaseValue: 0.48, Bulkiness: 0.58, Inputs: map[string]float64{"clay": 0.42, "timber": 0.12}},
			{Name: "perfumes", Category: "luxury", BaseValue: 0.82, Bulkiness: 0.12, Inputs: map[string]float64{"herbs": 0.35, "resin": 0.18, "ceramics": 0.08}},
		},
	}
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{{
			NodeID:   0,
			PolityID: 0,
			Wealth:   0.62,
			Supply:   map[string]float64{"clay": 1.20, "timber": 0.80, "herbs": 0.90, "resin": 0.70, "ceramics": 0.00, "perfumes": 0.00},
			Demand:   map[string]float64{"clay": 0.20, "timber": 0.18, "herbs": 0.16, "resin": 0.12, "ceramics": 0.20, "perfumes": 0.18},
			Surplus:  map[string]float64{"clay": 1.00, "timber": 0.62, "herbs": 0.74, "resin": 0.58, "ceramics": -0.20, "perfumes": -0.18},
		}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70},
		},
	}
	trade := &TradeNetworkResult{
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.30}},
	}

	result := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, settings, nil, nil, network, trade, nodeGoods)
	if len(result.Markets) != 1 {
		t.Fatalf("expected one market, got %d", len(result.Markets))
	}
	market := result.Markets[0]
	if market.Supply["ceramics"] <= 0 {
		t.Fatalf("expected ceramics to be manufactured from raw inputs")
	}
	if market.Supply["perfumes"] <= 0 {
		t.Fatalf("expected chained market manufacturing to produce perfumes, got %.3f", market.Supply["perfumes"])
	}
}

func TestComputeTradeNodeMarketsRespectsGoodSpecificMarketConversionScale(t *testing.T) {
	baseSettings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.0, "processed": 1.0, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "processed": 0.2, "default": 0.0},
			ManufacturingBaseScale:      map[string]float64{"processed": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"processed": 0.45, "default": 0.55},
			MarketWorkshopBias:          map[string]float64{"processed": 1.15, "default": 1.0},
			MarketInputRichnessScale:    map[string]float64{"processed": 0.10, "default": 0.14},
			MarketConversionShare:       map[string]float64{"processed": 0.28, "default": 0.40},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "livestock", Category: "raw", BaseValue: 0.44, Bulkiness: 0.72, SourceWeights: map[string]float64{"pasture": 1}},
			{Name: "leather", Category: "processed", BaseValue: 0.48, Bulkiness: 0.42, Inputs: map[string]float64{"livestock": 0.34}},
		},
	}
	scaledSettings := baseSettings
	scaledSettings.Goods = append([]TradeGoodSpec(nil), baseSettings.Goods...)
	scaledSettings.Goods[1].MarketConversionScale = 0.25
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{{
			NodeID:   0,
			PolityID: 0,
			Wealth:   0.60,
			Supply:   map[string]float64{"livestock": 1.20, "leather": 0.00},
			Demand:   map[string]float64{"livestock": 0.20, "leather": 0.10},
			Surplus:  map[string]float64{"livestock": 1.00, "leather": -0.10},
		}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70},
		},
	}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.30}}}

	base := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, baseSettings, nil, nil, network, trade, nodeGoods)
	scaled := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, scaledSettings, nil, nil, network, trade, nodeGoods)

	if scaled.Markets[0].Supply["leather"] >= base.Markets[0].Supply["leather"] {
		t.Fatalf("expected good-specific market conversion scale to reduce leather output, got base=%.3f scaled=%.3f", base.Markets[0].Supply["leather"], scaled.Markets[0].Supply["leather"])
	}
}
