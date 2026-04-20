package climgen

import "testing"

func TestValidateTradeGoodsSettingsRejectsInvalidLocalInputCapabilityFloor(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{
				Name:                      "paper",
				Category:                  "processed",
				BaseValue:                 0.6,
				Bulkiness:                 0.2,
				LocalInputCapabilityFloor: map[string]float64{"resin": 0.20},
				Inputs:                    map[string]float64{"fiber": 0.30},
			},
		},
	}
	if err := ValidateTradeGoodsSettings(settings); err == nil {
		t.Fatalf("expected invalid local input capability floor to be rejected")
	}
}

func TestNodeAndPolityGoodsLocalInputCapabilityFloorSuppressesWeakManufacturing(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{
				Name:                      "fiber",
				Category:                  "raw",
				BaseValue:                 0.4,
				Bulkiness:                 0.6,
				SourceWeights:             map[string]float64{"crop": 1},
				ProfileProductionAffinity: map[string]float64{},
			},
			{
				Name:                      "resin",
				Category:                  "raw",
				BaseValue:                 0.4,
				Bulkiness:                 0.3,
				SourceWeights:             map[string]float64{"resin": 1},
				ProfileProductionAffinity: map[string]float64{},
			},
			{
				Name:                      "paper",
				Category:                  "processed",
				BaseValue:                 0.6,
				Bulkiness:                 0.2,
				Inputs:                    map[string]float64{"fiber": 0.30, "resin": 0.10},
				LocalInputCapabilityFloor: map[string]float64{"fiber": 0.40, "resin": 0.20},
			},
		},
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "fiber", Category: "raw", Potential: []float64{0.45, 0.45}},
			{Good: "resin", Category: "raw", Potential: []float64{0.12, 0.12}},
			{Good: "paper", Category: "processed", Potential: []float64{0, 0}},
		},
	}
	polities := &PolitySphereResult{
		Spheres:     []PolitySphere{{ID: 0, CapitalNode: 0, TerritoryCells: 2, MeanSupport: 0.36}},
		Diagnostics: &PolitySphereDiagnostics{PolityByCell: []int{0, 0}},
	}
	network := &SettlementNetworkResult{Nodes: []SettlementNode{{ID: 0, Kind: SettlementNodeTown, CellIndex: 0}}}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.2}}}
	cells := []VoronoiCell{{}, {}}

	nodeGoods := ComputeNodeGoods(cells, goods, settings, polities, nil, network, trade)
	if got := nodeGoods.Balances[0].Supply["paper"]; got != 0 {
		t.Fatalf("expected node paper supply to be gated to zero, got %.3f", got)
	}

	baselinePolityGoods := ComputePolityGoods(goods, settings, polities, nil, network, trade)
	polityGoods := ComputePolityGoodsWithNodeMarkets(goods, settings, polities, nil, network, trade, nodeGoods)
	if polityGoods.Balances[0].Supply["paper"] >= baselinePolityGoods.Balances[0].Supply["paper"] {
		t.Fatalf(
			"expected node-aware polity paper supply to be reduced, got baseline=%.3f nodeAware=%.3f",
			baselinePolityGoods.Balances[0].Supply["paper"],
			polityGoods.Balances[0].Supply["paper"],
		)
	}
}
