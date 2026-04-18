package climgen

import "testing"

func TestComputeTradeNodeMarketsDominancePenaltyReducesRunawayChain(t *testing.T) {
	baseSettings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.0, "processed": 1.0, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "processed": 0.2, "default": 0.0},
			ManufacturingBaseScale:      map[string]float64{"processed": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"processed": 0.75, "default": 0.55},
			MarketWorkshopBias:          map[string]float64{"processed": 1.10, "default": 1.0},
			MarketInputRichnessScale:    map[string]float64{"processed": 0.16, "default": 0.14},
			MarketConversionShare:       map[string]float64{"processed": 0.60, "default": 0.40},
			MarketDominancePenalty:      map[string]float64{"processed": 0.0, "default": 0.0},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "fiber", Category: "raw", BaseValue: 0.40, Bulkiness: 0.58, SourceWeights: map[string]float64{"fiber": 1}},
			{Name: "resin", Category: "raw", BaseValue: 0.56, Bulkiness: 0.42, SourceWeights: map[string]float64{"resin": 1}},
			{Name: "cloth", Category: "processed", BaseValue: 0.62, Bulkiness: 0.38, Inputs: map[string]float64{"fiber": 0.45}},
			{Name: "paper", Category: "processed", BaseValue: 0.64, Bulkiness: 0.18, Inputs: map[string]float64{"fiber": 0.34, "resin": 0.10}},
		},
	}
	penalizedSettings := baseSettings
	penalizedSettings.Goods = append([]TradeGoodSpec(nil), baseSettings.Goods...)
	penalizedSettings.Goods[2].MarketDominancePenalty = 0.28
	penalizedSettings.Goods[3].MarketDominancePenalty = 0.28
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{{
			NodeID:   0,
			PolityID: 0,
			Wealth:   0.62,
			Supply:   map[string]float64{"fiber": 3.0, "resin": 1.2, "cloth": 0.0, "paper": 0.0},
			Demand:   map[string]float64{"fiber": 0.20, "resin": 0.10, "cloth": 0.22, "paper": 0.22},
			Surplus:  map[string]float64{"fiber": 2.80, "resin": 1.10, "cloth": -0.22, "paper": -0.22},
		}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70},
		},
	}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.30}}}

	base := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, baseSettings, nil, nil, network, trade, nodeGoods)
	penalized := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, penalizedSettings, nil, nil, network, trade, nodeGoods)

	baseTotal := base.Markets[0].Supply["cloth"] + base.Markets[0].Supply["paper"]
	penalizedTotal := penalized.Markets[0].Supply["cloth"] + penalized.Markets[0].Supply["paper"]
	if penalizedTotal >= baseTotal {
		t.Fatalf("expected dominance penalty to reduce runaway processed output, got base=%.3f penalized=%.3f", baseTotal, penalizedTotal)
	}
	if len(penalized.Markets[0].Diagnostics.Penalized) == 0 {
		t.Fatalf("expected dominance penalty diagnostics to record penalized goods")
	}
}

func TestComputeTradeNodeMarketsPrioritizesHigherNeedManufacturingChain(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.0, "processed": 1.0, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "processed": 0.2, "default": 0.0},
			ManufacturingBaseScale:      map[string]float64{"processed": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"processed": 0.75, "default": 0.55},
			MarketWorkshopBias:          map[string]float64{"processed": 1.10, "default": 1.0},
			MarketInputRichnessScale:    map[string]float64{"processed": 0.16, "default": 0.14},
			MarketConversionShare:       map[string]float64{"processed": 0.72, "default": 0.40},
			MarketDominancePenalty:      map[string]float64{"processed": 0.0, "default": 0.0},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "fiber", Category: "raw", BaseValue: 0.40, Bulkiness: 0.58, SourceWeights: map[string]float64{"fiber": 1}},
			{Name: "cloth", Category: "processed", BaseValue: 0.60, Bulkiness: 0.36, Inputs: map[string]float64{"fiber": 0.45}},
			{Name: "paper", Category: "processed", BaseValue: 0.58, Bulkiness: 0.18, Inputs: map[string]float64{"fiber": 0.45}},
		},
	}
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{{
			NodeID:   0,
			PolityID: 0,
			Wealth:   0.62,
			Supply:   map[string]float64{"fiber": 0.62, "cloth": 0.00, "paper": 0.00},
			Demand:   map[string]float64{"fiber": 0.12, "cloth": 0.08, "paper": 0.42},
			Surplus:  map[string]float64{"fiber": 0.50, "cloth": -0.08, "paper": -0.42},
		}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70},
		},
	}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.30}}}

	result := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, settings, nil, nil, network, trade, nodeGoods)
	if len(result.Markets) != 1 {
		t.Fatalf("expected one market, got %d", len(result.Markets))
	}
	market := result.Markets[0]
	if market.Manufactured["paper"] <= market.Manufactured["cloth"] {
		t.Fatalf(
			"expected higher-need paper chain to outrank earlier cloth chain, got paper=%.3f cloth=%.3f",
			market.Manufactured["paper"],
			market.Manufactured["cloth"],
		)
	}
}

func TestComputeTradeNodeMarketsPrioritizesIntermediateThatUnlocksBlockedChain(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.0, "finished": 1.0, "strategic": 1.0, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "finished": 0.2, "strategic": 0.2, "default": 0.0},
			ManufacturingBaseScale:      map[string]float64{"finished": 0.55, "strategic": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"finished": 0.75, "strategic": 0.75, "default": 0.55},
			MarketWorkshopBias:          map[string]float64{"finished": 1.10, "strategic": 1.10, "default": 1.0},
			MarketInputRichnessScale:    map[string]float64{"finished": 0.16, "strategic": 0.16, "default": 0.14},
			MarketConversionShare:       map[string]float64{"finished": 0.72, "strategic": 0.72, "default": 0.40},
			MarketDominancePenalty:      map[string]float64{"finished": 0.0, "strategic": 0.0, "default": 0.0},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "iron_ore", Category: "raw", BaseValue: 0.58, Bulkiness: 0.95, SourceWeights: map[string]float64{"iron_ore": 1}},
			{Name: "coal", Category: "raw", BaseValue: 0.52, Bulkiness: 0.90, SourceWeights: map[string]float64{"coal": 1}},
			{Name: "livestock", Category: "raw", BaseValue: 0.44, Bulkiness: 0.72, SourceWeights: map[string]float64{"pasture": 1}},
			{Name: "ornaments", Category: "finished", BaseValue: 0.62, Bulkiness: 0.26, Inputs: map[string]float64{"iron_ore": 0.45, "coal": 0.18}},
			{Name: "iron_goods", Category: "finished", BaseValue: 0.72, Bulkiness: 0.40, Inputs: map[string]float64{"iron_ore": 0.45, "coal": 0.18}},
			{Name: "weapons_armor", Category: "strategic", BaseValue: 0.84, Bulkiness: 0.52, Inputs: map[string]float64{"iron_goods": 0.38, "livestock": 0.12}},
		},
	}
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{{
			NodeID:   0,
			PolityID: 0,
			Wealth:   0.62,
			Supply: map[string]float64{
				"iron_ore":      0.62,
				"coal":          0.24,
				"livestock":     0.40,
				"ornaments":     0.00,
				"iron_goods":    0.00,
				"weapons_armor": 0.00,
			},
			Demand: map[string]float64{
				"iron_ore":      0.12,
				"coal":          0.08,
				"livestock":     0.10,
				"ornaments":     0.34,
				"iron_goods":    0.05,
				"weapons_armor": 0.46,
			},
			Surplus: map[string]float64{
				"iron_ore":      0.50,
				"coal":          0.16,
				"livestock":     0.30,
				"ornaments":     -0.34,
				"iron_goods":    -0.05,
				"weapons_armor": -0.46,
			},
		}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70},
		},
	}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.30}}}

	result := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, settings, nil, nil, network, trade, nodeGoods)
	if len(result.Markets) != 1 {
		t.Fatalf("expected one market, got %d", len(result.Markets))
	}
	market := result.Markets[0]
	if market.Manufactured["iron_goods"] <= market.Manufactured["ornaments"] {
		t.Fatalf(
			"expected downstream-unlocking iron goods to outrank locally needy ornaments, got iron_goods=%.3f ornaments=%.3f",
			market.Manufactured["iron_goods"],
			market.Manufactured["ornaments"],
		)
	}
	if market.Supply["weapons_armor"] <= 0 {
		t.Fatalf("expected prioritized iron goods to unlock weapons_armor manufacturing, got %.3f", market.Supply["weapons_armor"])
	}
}
