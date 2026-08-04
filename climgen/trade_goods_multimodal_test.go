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

func TestTradeNodePolityPrefersNodeOwnershipOverBoundaryCellClaim(t *testing.T) {
	network := &SettlementNetworkResult{Nodes: []SettlementNode{
		{ID: 0, CellIndex: 0},
	}}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{
			{ID: 0, CapitalNode: 0},
			{ID: 1, CapitalNode: 1},
		},
		Diagnostics: &PolitySphereDiagnostics{
			PolityByCell: []int{1},
			PolityByNode: []int{0},
		},
	}

	if got := tradeNodePolity(0, polities, network); got != 0 {
		t.Fatalf("expected route endpoint to use explicit node polity ownership, got %d", got)
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

func TestComputeMultimodalTradeDoesNotTruncateRouteGoodsToTopFive(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "g1", Category: "raw", BaseValue: 0.60, Bulkiness: 0.20, Perishability: 0.02},
			{Name: "g2", Category: "raw", BaseValue: 0.59, Bulkiness: 0.20, Perishability: 0.02},
			{Name: "g3", Category: "raw", BaseValue: 0.58, Bulkiness: 0.20, Perishability: 0.02},
			{Name: "g4", Category: "raw", BaseValue: 0.57, Bulkiness: 0.20, Perishability: 0.02},
			{Name: "g5", Category: "raw", BaseValue: 0.56, Bulkiness: 0.20, Perishability: 0.02},
			{Name: "g6", Category: "processed", BaseValue: 0.55, Bulkiness: 0.20, Perishability: 0.02},
		},
	}
	goods := &PolityGoodsResult{Balances: []PolityGoodBalance{
		{PolityID: 0, Surplus: map[string]float64{"g1": 0.30, "g2": 0.30, "g3": 0.30, "g4": 0.30, "g5": 0.30, "g6": 0.30}},
		{PolityID: 1, Surplus: map[string]float64{"g1": -0.20, "g2": -0.20, "g3": -0.20, "g4": -0.20, "g5": -0.20, "g6": -0.20}},
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
		TravelCost:  3,
		Flow:        0.50,
		MeanSupport: 0.75,
		MeanRisk:    0.10,
	}}}

	result := ComputeMultimodalTrade(goods, settings, polities, network, land, nil, nil, nil)
	if len(result.Exchanges) == 0 {
		t.Fatalf("expected exchanges")
	}
	found := map[string]bool{}
	for _, good := range result.Exchanges[0].Goods {
		found[good.Good] = true
	}
	if len(found) != 6 || !found["g6"] {
		t.Fatalf("expected all route goods to be retained, got %+v", result.Exchanges[0].Goods)
	}
}

func TestComputeMultimodalTradeDoesNotTruncatePairGoodsToTopFive(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "g1", Category: "raw", BaseValue: 0.60, Bulkiness: 0.20, Perishability: 0.02},
			{Name: "g2", Category: "raw", BaseValue: 0.59, Bulkiness: 0.20, Perishability: 0.02},
			{Name: "g3", Category: "raw", BaseValue: 0.58, Bulkiness: 0.20, Perishability: 0.02},
			{Name: "g4", Category: "raw", BaseValue: 0.57, Bulkiness: 0.20, Perishability: 0.02},
			{Name: "g5", Category: "raw", BaseValue: 0.56, Bulkiness: 0.20, Perishability: 0.02},
			{Name: "g6", Category: "processed", BaseValue: 0.55, Bulkiness: 0.20, Perishability: 0.02},
		},
	}
	goods := &PolityGoodsResult{Balances: []PolityGoodBalance{
		{PolityID: 0, Surplus: map[string]float64{"g1": 0.30, "g2": 0.30, "g3": 0.30, "g4": 0.30, "g5": 0.30, "g6": 0.30}},
		{PolityID: 1, Surplus: map[string]float64{"g1": -0.20, "g2": -0.20, "g3": -0.20, "g4": -0.20, "g5": -0.20, "g6": -0.20}},
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
	land := &TradeNetworkResult{Corridors: []TradeCorridor{
		{
			FromNode:    0,
			ToNode:      1,
			Role:        TradeCorridorRoleInterPolityTrunk,
			TravelCost:  3,
			Flow:        0.50,
			MeanSupport: 0.75,
			MeanRisk:    0.10,
		},
		{
			FromNode:    0,
			ToNode:      1,
			Role:        TradeCorridorRoleInterPolityTrunk,
			TravelCost:  3,
			Flow:        0.45,
			MeanSupport: 0.70,
			MeanRisk:    0.10,
		},
	}}

	result := ComputeMultimodalTrade(goods, settings, polities, network, land, nil, nil, nil)
	if len(result.Pairs) == 0 {
		t.Fatalf("expected pair flows")
	}
	found := map[string]bool{}
	for _, good := range result.Pairs[0].Goods {
		found[good.Good] = true
	}
	if len(found) != 6 || !found["g6"] {
		t.Fatalf("expected all pair goods to be retained, got %+v", result.Pairs[0].Goods)
	}
}

func TestComputeMultimodalTradeExcludesInternalCivilizationFromExternalPairs(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "soap", Category: "processed", BaseValue: 0.60, Bulkiness: 0.20, Perishability: 0.02},
		},
	}
	goods := &PolityGoodsResult{Balances: []PolityGoodBalance{
		{PolityID: 0, Surplus: map[string]float64{"soap": 0.30}},
		{PolityID: 1, Surplus: map[string]float64{"soap": -0.20}},
	}}
	network := &SettlementNetworkResult{Nodes: []SettlementNode{
		{ID: 0, CellIndex: 0},
		{ID: 1, CellIndex: 1},
	}}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{{ID: 0}, {ID: 1}},
		Diagnostics: &PolitySphereDiagnostics{
			PolityByCell: []int{0, 1},
			PolityByNode: []int{0, 1},
		},
	}
	coastal := &CoastalTradeResult{Corridors: []CoastalTradeCorridor{{
		ID:                1,
		FromNode:          0,
		ToNode:            1,
		FromCivilization:  7,
		ToCivilization:    7,
		Flow:              0.60,
		TravelCost:        3,
		MeanCurrentAssist: 0.10,
	}}}

	result := ComputeMultimodalTrade(goods, settings, polities, network, nil, nil, coastal, nil)
	if len(result.Exchanges) == 0 {
		t.Fatalf("expected internal exchange to remain auditable")
	}
	if !result.Exchanges[0].Internal {
		t.Fatalf("expected exchange to be marked internal: %+v", result.Exchanges[0])
	}
	if len(result.Pairs) != 0 {
		t.Fatalf("expected internal trade to be excluded from external pairs, got %+v", result.Pairs)
	}
	if result.Diagnostics.TotalScore != 0 || result.Diagnostics.InternalScore <= 0 {
		t.Fatalf("expected headline external score zero and internal score recorded, got %+v", result.Diagnostics)
	}
}

func TestComputeMultimodalTradeTracksCategoryDiagnostics(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.45, Bulkiness: 0.82, Perishability: 0.40},
			{Name: "paper", Category: "processed", BaseValue: 0.60, Bulkiness: 0.18, Perishability: 0.04},
		},
	}
	goods := &PolityGoodsResult{Balances: []PolityGoodBalance{
		{PolityID: 0, Surplus: map[string]float64{"grain": 0.30, "paper": 0.25}},
		{PolityID: 1, Surplus: map[string]float64{"grain": -0.24, "paper": 0.00}},
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
		TravelCost:  3,
		Flow:        0.50,
		MeanSupport: 0.75,
		MeanRisk:    0.10,
	}}}

	result := ComputeMultimodalTrade(goods, settings, polities, network, land, nil, nil, nil)
	processed := result.Diagnostics.ByCategory["processed"]
	raw := result.Diagnostics.ByCategory["raw"]
	if processed.CandidateGoods != 2 || processed.NoSinkNeed == 0 {
		t.Fatalf("expected processed diagnostics to record rejected sink need, got %+v", processed)
	}
	if raw.AcceptedGoods == 0 || raw.TotalScore <= 0 {
		t.Fatalf("expected raw diagnostics to record accepted trade, got %+v", raw)
	}
}

func TestComputeMultimodalTradeCapsDuplicatePairGoodFlows(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.45, Bulkiness: 0.82, Perishability: 0.40},
		},
	}
	goods := &PolityGoodsResult{
		Balances: []PolityGoodBalance{
			{
				PolityID: 0,
				Surplus:  map[string]float64{"grain": 0.25},
			},
			{
				PolityID: 1,
				Surplus:  map[string]float64{"grain": -0.10},
			},
		},
		GlobalScarcityByGood: map[string]float64{"grain": 0.5},
	}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{{ID: 0}, {ID: 1}},
		Diagnostics: &PolitySphereDiagnostics{
			PolityByCell: []int{0, 1},
			PolityByNode: []int{0, 1},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown},
		},
	}
	land := &TradeNetworkResult{
		Corridors: []TradeCorridor{
			{ID: 0, FromNode: 0, ToNode: 1, Role: TradeCorridorRoleInterPolityTrunk, Flow: 0.9, TravelCost: 1, MeanSupport: 0.8},
			{ID: 1, FromNode: 0, ToNode: 1, Role: TradeCorridorRoleInterPolityTrunk, Flow: 0.9, TravelCost: 1, MeanSupport: 0.8},
		},
	}

	result := ComputeMultimodalTrade(goods, settings, polities, network, land, nil, nil, nil)
	matched := 0.0
	for _, exchange := range result.Exchanges {
		if exchange.FromPolity != 0 || exchange.ToPolity != 1 {
			continue
		}
		for _, good := range exchange.Goods {
			if good.Good == "grain" {
				matched += good.Matched
			}
		}
	}
	if matched < 0.099 || matched > 0.101 {
		t.Fatalf("expected duplicate route matches to be capped to sink need 0.10, got %.3f", matched)
	}
}

func TestComputeMultimodalTradeCapsSourceGoodAcrossDestinations(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.45, Bulkiness: 0.82, Perishability: 0.40},
		},
	}
	goods := &PolityGoodsResult{
		Balances: []PolityGoodBalance{
			{PolityID: 0, Surplus: map[string]float64{"grain": 0.12}},
			{PolityID: 1, Surplus: map[string]float64{"grain": -0.10}},
			{PolityID: 2, Surplus: map[string]float64{"grain": -0.10}},
		},
		GlobalScarcityByGood: map[string]float64{"grain": 0.5},
	}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{{ID: 0}, {ID: 1}, {ID: 2}},
		Diagnostics: &PolitySphereDiagnostics{
			PolityByCell: []int{0, 1, 2},
			PolityByNode: []int{0, 1, 2},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown},
			{ID: 2, CellIndex: 2, Kind: SettlementNodeTown},
		},
	}
	land := &TradeNetworkResult{
		Corridors: []TradeCorridor{
			{ID: 0, FromNode: 0, ToNode: 1, Role: TradeCorridorRoleInterPolityTrunk, Flow: 0.9, TravelCost: 1, MeanSupport: 0.8},
			{ID: 1, FromNode: 0, ToNode: 2, Role: TradeCorridorRoleInterPolityTrunk, Flow: 0.9, TravelCost: 1, MeanSupport: 0.8},
		},
	}

	result := ComputeMultimodalTrade(goods, settings, polities, network, land, nil, nil, nil)
	matched := 0.0
	for _, exchange := range result.Exchanges {
		if exchange.FromPolity != 0 {
			continue
		}
		for _, good := range exchange.Goods {
			if good.Good == "grain" {
				matched += good.Matched
			}
		}
	}
	if matched < 0.119 || matched > 0.121 {
		t.Fatalf("expected source exports to be capped to polity surplus 0.12, got %.3f", matched)
	}
}

func TestComputeMultimodalTradeCapsEndpointNeedAcrossSources(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Multimodal: TradeGoodsMultimodalSettings{
			EndpointNeedShareByCategory: map[string]float64{"processed": 1.0},
		},
		Goods: []TradeGoodSpec{
			{Name: "paper", Category: "processed", BaseValue: 0.64, Bulkiness: 0.18, Perishability: 0.04},
		},
	}
	goods := &PolityGoodsResult{
		Balances: []PolityGoodBalance{
			{PolityID: 0, Surplus: map[string]float64{"paper": 0.20}},
			{PolityID: 1, Surplus: map[string]float64{"paper": 0.20}},
			{PolityID: 2, Surplus: map[string]float64{"paper": 0.00}},
		},
		GlobalScarcityByGood: map[string]float64{"paper": 0.5},
	}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{{ID: 0}, {ID: 1}, {ID: 2}},
		Diagnostics: &PolitySphereDiagnostics{
			PolityByCell: []int{0, 1, 2},
			PolityByNode: []int{0, 1, 2},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown},
			{ID: 2, CellIndex: 2, Kind: SettlementNodeTown},
		},
	}
	land := &TradeNetworkResult{
		Corridors: []TradeCorridor{
			{ID: 0, FromNode: 0, ToNode: 2, Role: TradeCorridorRoleInterPolityTrunk, Flow: 0.9, TravelCost: 1, MeanSupport: 0.8},
			{ID: 1, FromNode: 1, ToNode: 2, Role: TradeCorridorRoleInterPolityTrunk, Flow: 0.9, TravelCost: 1, MeanSupport: 0.8},
		},
	}
	nodeMarkets := &TradeNodeMarketResult{Markets: []TradeNodeMarket{
		{NodeID: 0, PolityID: 0, Supply: map[string]float64{"paper": 0.25}, Demand: map[string]float64{"paper": 0.05}, Surplus: map[string]float64{"paper": 0.20}},
		{NodeID: 1, PolityID: 1, Supply: map[string]float64{"paper": 0.25}, Demand: map[string]float64{"paper": 0.05}, Surplus: map[string]float64{"paper": 0.20}},
		{NodeID: 2, PolityID: 2, Supply: map[string]float64{"paper": 0.05}, Demand: map[string]float64{"paper": 0.15}, Surplus: map[string]float64{"paper": -0.10}},
	}}

	result := ComputeMultimodalTradeWithNodeMarkets(goods, settings, polities, network, land, nil, nil, nil, nodeMarkets)
	matched := 0.0
	for _, exchange := range result.Exchanges {
		if exchange.ToPolity != 2 {
			continue
		}
		for _, good := range exchange.Goods {
			if good.Good == "paper" {
				matched += good.Matched
			}
		}
	}
	if matched < 0.099 || matched > 0.101 {
		t.Fatalf("expected endpoint imports to be capped to destination market need 0.10, got %.3f", matched)
	}
}

func TestComputeMultimodalTradeCapsDuplicateEndpointNeedForPair(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Multimodal: TradeGoodsMultimodalSettings{
			EndpointNeedShareByCategory: map[string]float64{"processed": 1.0},
		},
		Goods: []TradeGoodSpec{
			{Name: "paper", Category: "processed", BaseValue: 0.64, Bulkiness: 0.18, Perishability: 0.04},
		},
	}
	goods := &PolityGoodsResult{
		Balances: []PolityGoodBalance{
			{PolityID: 0, Surplus: map[string]float64{"paper": 0.30}},
			{PolityID: 1, Surplus: map[string]float64{"paper": 0.00}},
		},
		GlobalScarcityByGood: map[string]float64{"paper": 0.5},
	}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{{ID: 0}, {ID: 1}},
		Diagnostics: &PolitySphereDiagnostics{
			PolityByCell: []int{0, 1},
			PolityByNode: []int{0, 1},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown},
		},
	}
	land := &TradeNetworkResult{
		Corridors: []TradeCorridor{
			{ID: 0, FromNode: 0, ToNode: 1, Role: TradeCorridorRoleInterPolityTrunk, Flow: 0.9, TravelCost: 1, MeanSupport: 0.8},
			{ID: 1, FromNode: 0, ToNode: 1, Role: TradeCorridorRoleInterPolityTrunk, Flow: 0.9, TravelCost: 1, MeanSupport: 0.8},
		},
	}
	nodeMarkets := &TradeNodeMarketResult{Markets: []TradeNodeMarket{
		{NodeID: 0, PolityID: 0, Supply: map[string]float64{"paper": 0.35}, Demand: map[string]float64{"paper": 0.05}, Surplus: map[string]float64{"paper": 0.30}},
		{NodeID: 1, PolityID: 1, Supply: map[string]float64{"paper": 0.05}, Demand: map[string]float64{"paper": 0.15}, Surplus: map[string]float64{"paper": -0.10}},
	}}

	result := ComputeMultimodalTradeWithNodeMarkets(goods, settings, polities, network, land, nil, nil, nil, nodeMarkets)
	matched := 0.0
	for _, exchange := range result.Exchanges {
		if exchange.FromPolity != 0 || exchange.ToPolity != 1 {
			continue
		}
		for _, good := range exchange.Goods {
			if good.Good == "paper" {
				matched += good.Matched
			}
		}
	}
	if matched < 0.099 || matched > 0.101 {
		t.Fatalf("expected duplicate endpoint pair imports to be capped to market need 0.10, got %.3f", matched)
	}
}

func TestComputeMultimodalTradeUsesEndpointMarketNeedShare(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Multimodal: TradeGoodsMultimodalSettings{
			EndpointNeedShareByCategory: map[string]float64{
				"processed": 1.0,
			},
		},
		Goods: []TradeGoodSpec{
			{Name: "paper", Category: "processed", BaseValue: 0.64, Bulkiness: 0.18, Perishability: 0.04},
		},
	}
	goods := &PolityGoodsResult{Balances: []PolityGoodBalance{
		{PolityID: 0, Surplus: map[string]float64{"paper": 0.20}},
		{PolityID: 1, Surplus: map[string]float64{"paper": 0.00}},
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
		TravelCost:  3,
		Flow:        0.50,
		MeanSupport: 0.75,
		MeanRisk:    0.10,
	}}}
	nodeMarkets := &TradeNodeMarketResult{Markets: []TradeNodeMarket{
		{
			NodeID:   0,
			PolityID: 0,
			Wealth:   0.50,
			Supply:   map[string]float64{"paper": 0.30},
			Demand:   map[string]float64{"paper": 0.05},
			Surplus:  map[string]float64{"paper": 0.25},
		},
		{
			NodeID:   1,
			PolityID: 1,
			Wealth:   0.52,
			Supply:   map[string]float64{"paper": 0.10},
			Demand:   map[string]float64{"paper": 0.30},
			Surplus:  map[string]float64{"paper": -0.20},
		},
	}}

	result := ComputeMultimodalTradeWithNodeMarkets(goods, settings, polities, network, land, nil, nil, nil, nodeMarkets)
	if len(result.Exchanges) == 0 || len(result.Exchanges[0].Goods) == 0 {
		t.Fatalf("expected endpoint market demand share to unlock a processed exchange")
	}
	if result.Diagnostics.ByCategory["processed"].AcceptedGoods == 0 {
		t.Fatalf("expected processed category diagnostics to record accepted goods, got %+v", result.Diagnostics.ByCategory["processed"])
	}
}

func TestComputeMultimodalTradeEndpointNeedRelievedByAggregateSurplus(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Multimodal: TradeGoodsMultimodalSettings{
			EndpointNeedShareByCategory: map[string]float64{"processed": 1.0},
		},
		Goods: []TradeGoodSpec{
			{Name: "paper", Category: "processed", BaseValue: 0.64, Bulkiness: 0.18, Perishability: 0.04},
		},
	}
	relieved := settings
	relieved.Multimodal.EndpointSurplusReliefByCategory = map[string]float64{"processed": 10.0}
	goods := &PolityGoodsResult{
		Balances: []PolityGoodBalance{
			{PolityID: 0, Surplus: map[string]float64{"paper": 0.40}},
			{PolityID: 1, Surplus: map[string]float64{"paper": 1.00}},
		},
		GlobalScarcityByGood: map[string]float64{"paper": 0.5},
	}
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
		TravelCost:  3,
		Flow:        0.50,
		MeanSupport: 0.75,
	}}}
	nodeMarkets := &TradeNodeMarketResult{Markets: []TradeNodeMarket{
		{NodeID: 0, PolityID: 0, Wealth: 0.50, Supply: map[string]float64{"paper": 0.40}, Demand: map[string]float64{"paper": 0.05}, Surplus: map[string]float64{"paper": 0.35}},
		{NodeID: 1, PolityID: 1, Wealth: 0.52, Supply: map[string]float64{"paper": 0.10}, Demand: map[string]float64{"paper": 0.30}, Surplus: map[string]float64{"paper": -0.20}},
	}}

	base := ComputeMultimodalTradeWithNodeMarkets(goods, settings, polities, network, land, nil, nil, nil, nodeMarkets)
	tuned := ComputeMultimodalTradeWithNodeMarkets(goods, relieved, polities, network, land, nil, nil, nil, nodeMarkets)
	if base.Diagnostics.TotalMatched <= 0.05 {
		t.Fatalf("expected endpoint need share to create baseline matched flow, got %.3f", base.Diagnostics.TotalMatched)
	}
	if tuned.Diagnostics.TotalMatched >= base.Diagnostics.TotalMatched*0.25 {
		t.Fatalf("expected aggregate surplus relief to damp endpoint-only imports, base=%.3f tuned=%.3f", base.Diagnostics.TotalMatched, tuned.Diagnostics.TotalMatched)
	}
}
