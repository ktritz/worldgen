package climgen

import "testing"

func TestComputeMultimodalTradeMatchesSurplusToRouteDemand(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.45, Bulkiness: 0.82, Perishability: 0.40},
			{Name: "iron_goods", Category: "finished", BaseValue: 0.72, Bulkiness: 0.44, Perishability: 0.02},
		},
	}
	goods := &PolityGoodsResult{Balances: []PolityGoodBalance{
		{PolityID: 0, Surplus: map[string]float64{"grain": 0.32, "iron_goods": -0.20}},
		{PolityID: 1, Surplus: map[string]float64{"grain": -0.25, "iron_goods": 0.24}},
	}}
	network := &SettlementNetworkResult{Nodes: []SettlementNode{
		{ID: 0, CellIndex: 0},
		{ID: 1, CellIndex: 1},
	}}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{{ID: 0, CapitalNode: 0}, {ID: 1, CapitalNode: 1}},
		Diagnostics: &PolitySphereDiagnostics{
			PolityByCell: []int{0, 1},
			PolityByNode: []int{0, 1},
		},
	}
	land := &TradeNetworkResult{Corridors: []TradeCorridor{{
		FromNode:    0,
		ToNode:      1,
		Role:        TradeCorridorRoleInterPolityTrunk,
		TravelCost:  4,
		Flow:        0.35,
		MeanSupport: 0.65,
		MeanRisk:    0.10,
	}}}

	result := ComputeMultimodalTrade(goods, settings, polities, network, land, nil, nil, nil)
	if len(result.Exchanges) != 2 {
		t.Fatalf("expected two directional exchanges, got %d", len(result.Exchanges))
	}
	if len(result.Pairs) != 2 {
		t.Fatalf("expected two directed pair flows, got %d", len(result.Pairs))
	}
	if result.Exchanges[0].Value <= 0 || len(result.Exchanges[0].Goods) == 0 {
		t.Fatalf("expected positive matched goods, got %+v", result.Exchanges[0])
	}
	if result.Exchanges[0].Volume <= 0 || result.Exchanges[0].Matched <= 0 {
		t.Fatalf("expected positive trade volume diagnostics, got %+v", result.Exchanges[0])
	}
	if result.Exchanges[0].VolumeCapacity <= 0 {
		t.Fatalf("expected positive route volume capacity, got %+v", result.Exchanges[0])
	}
	if result.Diagnostics.TotalVolume <= 0 || result.Diagnostics.TotalMatched <= 0 {
		t.Fatalf("expected aggregate trade diagnostics, got %+v", result.Diagnostics)
	}
	if result.Diagnostics.CandidateGoods <= 0 || result.Diagnostics.AcceptedGoods <= 0 {
		t.Fatalf("expected goods diagnostic counters, got %+v", result.Diagnostics)
	}
}

func TestTradeGoodTransportValueDistinguishesModeFit(t *testing.T) {
	grain := TradeGoodSpec{Name: "grain", BaseValue: 0.45, Bulkiness: 0.86, Perishability: 0.40}
	ironGoods := TradeGoodSpec{Name: "iron_goods", BaseValue: 0.72, Bulkiness: 0.42, Perishability: 0.02}

	if tradeGoodTransportValue(grain, "river") <= tradeGoodTransportValue(grain, "land") {
		t.Fatalf("expected bulky grain to move better by river than land")
	}
	if tradeGoodTransportValue(ironGoods, "land") <= tradeGoodTransportValue(grain, "land") {
		t.Fatalf("expected durable finished goods to outperform bulky grain on land")
	}
}

func TestApplyMultimodalTradeToPolityProfilesAddsDirectedTradeBonus(t *testing.T) {
	profiles := &PolityProfileResult{Attitudes: []PolityAttitude{
		{From: 0, To: 1, Score: -0.05, Stance: PolityAttitudeNeutral},
		{From: 1, To: 0, Score: -0.05, Stance: PolityAttitudeNeutral},
	}}
	trade := &MultimodalTradeResult{Pairs: []TradeGoodPairFlow{
		{FromPolity: 0, ToPolity: 1, Value: 0.08},
	}}

	result := ApplyMultimodalTradeToPolityProfiles(profiles, trade)
	if result == profiles {
		t.Fatalf("expected profile result to be copied before attitude adjustment")
	}
	if result.Attitudes[0].TradeBonus <= profiles.Attitudes[0].TradeBonus {
		t.Fatalf("expected directed importer attitude to receive trade bonus")
	}
	if result.Attitudes[0].Score <= profiles.Attitudes[0].Score {
		t.Fatalf("expected directed importer score to improve")
	}
	if result.Attitudes[1].TradeBonus <= 0 {
		t.Fatalf("expected exporter to receive a smaller reverse-market bonus")
	}
	if result.Attitudes[0].TradeBonus <= result.Attitudes[1].TradeBonus {
		t.Fatalf("expected direct dependency bonus to exceed reverse-market bonus")
	}
}

func TestComputeMultimodalTradeWithNodeMarketsRespectsEndpointMarketGoods(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.45, Bulkiness: 0.82, Perishability: 0.40},
			{Name: "iron_goods", Category: "finished", BaseValue: 0.72, Bulkiness: 0.44, Perishability: 0.02},
		},
	}
	goods := &PolityGoodsResult{Balances: []PolityGoodBalance{
		{PolityID: 0, Surplus: map[string]float64{"grain": 0.32, "iron_goods": 0.30}},
		{PolityID: 1, Surplus: map[string]float64{"grain": -0.25, "iron_goods": -0.22}},
	}}
	network := &SettlementNetworkResult{Nodes: []SettlementNode{
		{ID: 0, CellIndex: 0},
		{ID: 1, CellIndex: 1},
	}}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{{ID: 0, CapitalNode: 0}, {ID: 1, CapitalNode: 1}},
		Diagnostics: &PolitySphereDiagnostics{
			PolityByCell: []int{0, 1},
			PolityByNode: []int{0, 1},
		},
	}
	land := &TradeNetworkResult{Corridors: []TradeCorridor{{
		FromNode:    0,
		ToNode:      1,
		Role:        TradeCorridorRoleInterPolityTrunk,
		TravelCost:  4,
		Flow:        0.35,
		MeanSupport: 0.65,
		MeanRisk:    0.10,
	}}}
	nodeMarkets := &TradeNodeMarketResult{Markets: []TradeNodeMarket{
		{
			NodeID:   0,
			PolityID: 0,
			Wealth:   0.50,
			Supply:   map[string]float64{"grain": 0.28, "iron_goods": 0.02},
			Demand:   map[string]float64{"grain": 0.08, "iron_goods": 0.10},
			Surplus:  map[string]float64{"grain": 0.20, "iron_goods": -0.08},
		},
		{
			NodeID:   1,
			PolityID: 1,
			Wealth:   0.52,
			Supply:   map[string]float64{"grain": 0.10, "iron_goods": 0.18},
			Demand:   map[string]float64{"grain": 0.26, "iron_goods": 0.18},
			Surplus:  map[string]float64{"grain": -0.16, "iron_goods": 0.00},
		},
	}}

	result := ComputeMultimodalTradeWithNodeMarkets(goods, settings, polities, network, land, nil, nil, nil, nodeMarkets)
	if len(result.Exchanges) == 0 || len(result.Exchanges[0].Goods) == 0 {
		t.Fatalf("expected node-market constrained exchange goods")
	}
	for _, good := range result.Exchanges[0].Goods {
		if good.Good == "iron_goods" {
			t.Fatalf("expected iron goods to be suppressed by endpoint markets, got %+v", result.Exchanges[0].Goods)
		}
	}
	if len(result.Pairs) == 0 || result.Pairs[0].Volume <= 0 {
		t.Fatalf("expected pair volume diagnostics, got %+v", result.Pairs)
	}
}
