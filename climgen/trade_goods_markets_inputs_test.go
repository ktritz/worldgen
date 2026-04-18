package climgen

import "testing"

func TestComputeTradeNodeMarketsSaltReservationFavorsPreservedFoodAtFishRichMarkets(t *testing.T) {
	baseSettings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:            map[string]float64{"raw": 1.0, "processed": 1.0, "default": 1.0},
			CategorySpecializationScale:    map[string]float64{"raw": 0.0, "processed": 0.2, "default": 0.0},
			ManufacturingBaseScale:         map[string]float64{"processed": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:     map[string]float64{"processed": 0.75, "default": 0.55},
			MarketWorkshopBias:             map[string]float64{"processed": 1.10, "default": 1.0},
			MarketInputRichnessScale:       map[string]float64{"processed": 0.16, "default": 0.14},
			MarketConversionShare:          map[string]float64{"processed": 0.40, "default": 0.40},
			MarketInputReservationByInput:  map[string]float64{"salt": 0.25, "default": 0.0},
			MarketInputReservationStrength: 0.30,
			MarketInputReservationCap:      0.35,
			RawPotentialPivot:              0.50,
			ManufacturingPivot:             0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "fish", Category: "raw", BaseValue: 0.38, Bulkiness: 0.62, SourceWeights: map[string]float64{"fish": 1}},
			{Name: "salt", Category: "raw", BaseValue: 0.62, Bulkiness: 0.38, SourceWeights: map[string]float64{"salt": 1}},
			{Name: "livestock", Category: "raw", BaseValue: 0.46, Bulkiness: 0.72, SourceWeights: map[string]float64{"pasture": 1}},
			{Name: "herbs", Category: "raw", BaseValue: 0.60, Bulkiness: 0.20, SourceWeights: map[string]float64{"herbs": 1}},
			{
				Name:                       "preserved_food",
				Category:                   "processed",
				BaseValue:                  0.56,
				Bulkiness:                  0.62,
				MarketConversionScale:      0.36,
				MarketInputReservePriority: 0.15,
				MarketMinNodeKind:          "town",
				Inputs:                     map[string]float64{"fish": 0.35, "salt": 0.20},
			},
			{
				Name:                  "soap",
				Category:              "processed",
				BaseValue:             0.62,
				Bulkiness:             0.20,
				MarketConversionScale: 0.38,
				MarketMinNodeKind:     "town",
				Inputs:                map[string]float64{"salt": 0.14, "livestock": 0.10, "herbs": 0.03},
			},
		},
	}
	tunedSettings := baseSettings
	tunedSettings.Production.MarketInputReservationByInput = map[string]float64{"salt": 0.65, "default": 0.0}
	goods := append([]TradeGoodSpec(nil), baseSettings.Goods...)
	for i := range goods {
		if goods[i].Name == "preserved_food" {
			goods[i].MarketInputReservePriority = 0.90
		}
	}
	tunedSettings.Goods = goods

	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{
			{
				NodeID:   0,
				PolityID: 0,
				Wealth:   0.66,
				Supply:   map[string]float64{"fish": 1.40, "salt": 0.30, "livestock": 0.90, "herbs": 0.40},
				Demand:   map[string]float64{"fish": 0.20, "salt": 0.10, "livestock": 0.10, "herbs": 0.05, "preserved_food": 0.40, "soap": 0.24},
				Surplus:  map[string]float64{"fish": 1.20, "salt": 0.20, "livestock": 0.80, "herbs": 0.35, "preserved_food": -0.40, "soap": -0.24},
			},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70, Coastal: true, River: true},
		},
	}
	trade := &TradeNetworkResult{
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.32}},
	}

	base := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, baseSettings, nil, nil, network, trade, nodeGoods)
	tuned := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, tunedSettings, nil, nil, network, trade, nodeGoods)
	if len(base.Markets) != 1 || len(tuned.Markets) != 1 {
		t.Fatalf("expected one market in each result, got base=%d tuned=%d", len(base.Markets), len(tuned.Markets))
	}
	if tuned.Markets[0].Manufactured["preserved_food"] <= base.Markets[0].Manufactured["preserved_food"] {
		t.Fatalf("expected stronger salt reservation to favor preserved food, got base=%.3f tuned=%.3f", base.Markets[0].Manufactured["preserved_food"], tuned.Markets[0].Manufactured["preserved_food"])
	}
}

func TestComputeTradeNodeMarketsImportedRawInputSupportRedistributesCompactInputsWithinPolity(t *testing.T) {
	baseSettings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:             map[string]float64{"raw": 1.0, "processed": 1.0, "default": 1.0},
			CategorySpecializationScale:     map[string]float64{"raw": 0.0, "processed": 0.2, "default": 0.0},
			ManufacturingBaseScale:          map[string]float64{"processed": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:      map[string]float64{"processed": 0.75, "default": 0.55},
			MarketWorkshopBias:              map[string]float64{"processed": 1.10, "default": 1.0},
			MarketInputRichnessScale:        map[string]float64{"processed": 0.16, "default": 0.14},
			MarketConversionShare:           map[string]float64{"processed": 0.40, "default": 0.40},
			MarketImportedInputSupportScale: map[string]float64{"default": 0.0},
			RawPotentialPivot:               0.50,
			ManufacturingPivot:              0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "fish", Category: "raw", BaseValue: 0.38, Bulkiness: 0.62, SourceWeights: map[string]float64{"fish": 1}},
			{Name: "salt", Category: "raw", BaseValue: 0.62, Bulkiness: 0.38, Perishability: 0.02, SourceWeights: map[string]float64{"salt": 1}},
			{
				Name:                       "preserved_food",
				Category:                   "processed",
				BaseValue:                  0.56,
				Bulkiness:                  0.62,
				MarketConversionScale:      0.36,
				MarketInputReservePriority: 0.90,
				MarketMinNodeKind:          "town",
				Inputs:                     map[string]float64{"fish": 0.35, "salt": 0.20},
			},
		},
	}
	tunedSettings := baseSettings
	tunedSettings.Production.MarketImportedInputSupportScale = map[string]float64{"salt": 1.60, "default": 0.0}

	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{
			{
				NodeID:   0,
				PolityID: 0,
				Wealth:   0.68,
				Supply:   map[string]float64{"fish": 1.40, "salt": 0.03},
				Demand:   map[string]float64{"fish": 0.20, "salt": 0.36, "preserved_food": 0.40},
				Surplus:  map[string]float64{"fish": 1.20, "salt": -0.33, "preserved_food": -0.40},
			},
			{
				NodeID:   1,
				PolityID: 0,
				Wealth:   0.60,
				Supply:   map[string]float64{"fish": 0.00, "salt": 0.80},
				Demand:   map[string]float64{"fish": 0.00, "salt": 0.10, "preserved_food": 0.00},
				Surplus:  map[string]float64{"fish": 0.00, "salt": 0.70, "preserved_food": 0.00},
			},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70, Coastal: true, River: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, CarryingCapacity: 0.55, UrbanPotential: 0.52},
		},
	}
	trade := &TradeNetworkResult{
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.48, 0.44}},
	}

	base := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, baseSettings, nil, nil, network, trade, nodeGoods)
	tuned := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, tunedSettings, nil, nil, network, trade, nodeGoods)
	if len(base.Markets) != 2 || len(tuned.Markets) != 2 {
		t.Fatalf("expected two markets in each result, got base=%d tuned=%d", len(base.Markets), len(tuned.Markets))
	}
	if tuned.Markets[0].Manufactured["preserved_food"] <= base.Markets[0].Manufactured["preserved_food"] {
		t.Fatalf("expected imported raw support to raise preserved food output, got base=%.3f tuned=%.3f", base.Markets[0].Manufactured["preserved_food"], tuned.Markets[0].Manufactured["preserved_food"])
	}
	if tuned.Markets[1].Supply["salt"] >= base.Markets[1].Supply["salt"] {
		t.Fatalf("expected donor salt market to give up some salt, got base=%.3f tuned=%.3f", base.Markets[1].Supply["salt"], tuned.Markets[1].Supply["salt"])
	}
}

func TestComputeTradeNodeMarketsImportedRawInputSupportFavorsFeasibleSaltChain(t *testing.T) {
	baseSettings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:             map[string]float64{"raw": 1.0, "processed": 1.0, "default": 1.0},
			CategorySpecializationScale:     map[string]float64{"raw": 0.0, "processed": 0.2, "default": 0.0},
			ManufacturingBaseScale:          map[string]float64{"processed": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:      map[string]float64{"processed": 0.75, "default": 0.55},
			MarketWorkshopBias:              map[string]float64{"processed": 1.10, "default": 1.0},
			MarketInputRichnessScale:        map[string]float64{"processed": 0.16, "default": 0.14},
			MarketConversionShare:           map[string]float64{"processed": 0.40, "default": 0.40},
			MarketImportedInputSupportScale: map[string]float64{"default": 0.0},
			RawPotentialPivot:               0.50,
			ManufacturingPivot:              0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "fish", Category: "raw", BaseValue: 0.38, Bulkiness: 0.62, SourceWeights: map[string]float64{"fish": 1}},
			{Name: "salt", Category: "raw", BaseValue: 0.62, Bulkiness: 0.38, Perishability: 0.02, SourceWeights: map[string]float64{"salt": 1}},
			{Name: "livestock", Category: "raw", BaseValue: 0.46, Bulkiness: 0.72, SourceWeights: map[string]float64{"pasture": 1}},
			{Name: "herbs", Category: "raw", BaseValue: 0.60, Bulkiness: 0.20, SourceWeights: map[string]float64{"herbs": 1}},
			{
				Name:                       "preserved_food",
				Category:                   "processed",
				BaseValue:                  0.56,
				Bulkiness:                  0.62,
				MarketConversionScale:      0.36,
				MarketInputReservePriority: 0.90,
				MarketMinNodeKind:          "town",
				Inputs:                     map[string]float64{"fish": 0.35, "salt": 0.20},
			},
			{
				Name:                  "soap",
				Category:              "processed",
				BaseValue:             0.62,
				Bulkiness:             0.20,
				MarketConversionScale: 0.38,
				MarketMinNodeKind:     "town",
				Inputs:                map[string]float64{"salt": 0.14, "livestock": 0.10, "herbs": 0.03},
			},
		},
	}
	tunedSettings := baseSettings
	tunedSettings.Production.MarketImportedInputSupportScale = map[string]float64{"salt": 0.80, "default": 0.0}

	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{
			{
				NodeID:   0,
				PolityID: 0,
				Wealth:   0.68,
				Supply:   map[string]float64{"fish": 1.40, "salt": 0.03, "livestock": 0.10, "herbs": 0.05},
				Demand:   map[string]float64{"fish": 0.20, "salt": 0.22, "preserved_food": 0.42, "soap": 0.10},
				Surplus:  map[string]float64{"fish": 1.20, "salt": -0.19, "preserved_food": -0.42, "soap": -0.10},
			},
			{
				NodeID:   1,
				PolityID: 0,
				Wealth:   0.64,
				Supply:   map[string]float64{"fish": 0.00, "salt": 0.14, "livestock": 1.20, "herbs": 0.70},
				Demand:   map[string]float64{"fish": 0.00, "salt": 0.08, "preserved_food": 0.00, "soap": 0.34},
				Surplus:  map[string]float64{"fish": 0.00, "salt": 0.06, "preserved_food": 0.00, "soap": -0.34},
			},
			{
				NodeID:   2,
				PolityID: 0,
				Wealth:   0.60,
				Supply:   map[string]float64{"fish": 0.00, "salt": 0.80},
				Demand:   map[string]float64{"fish": 0.00, "salt": 0.10},
				Surplus:  map[string]float64{"fish": 0.00, "salt": 0.70},
			},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70, Coastal: true, River: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, CarryingCapacity: 0.58, UrbanPotential: 0.64},
			{ID: 2, CellIndex: 2, Kind: SettlementNodeTown, CarryingCapacity: 0.55, UrbanPotential: 0.52},
		},
	}
	trade := &TradeNetworkResult{
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.52, 0.46, 0.44}},
	}

	base := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, baseSettings, nil, nil, network, trade, nodeGoods)
	tuned := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, tunedSettings, nil, nil, network, trade, nodeGoods)
	if len(base.Markets) != 3 || len(tuned.Markets) != 3 {
		t.Fatalf("expected three markets in each result, got base=%d tuned=%d", len(base.Markets), len(tuned.Markets))
	}
	if tuned.Markets[0].Manufactured["preserved_food"] <= base.Markets[0].Manufactured["preserved_food"] {
		t.Fatalf("expected salt support to favor fish-rich preserved food market, got base=%.3f tuned=%.3f", base.Markets[0].Manufactured["preserved_food"], tuned.Markets[0].Manufactured["preserved_food"])
	}
	preservedGain := tuned.Markets[0].Manufactured["preserved_food"] - base.Markets[0].Manufactured["preserved_food"]
	soapGain := tuned.Markets[1].Manufactured["soap"] - base.Markets[1].Manufactured["soap"]
	if preservedGain <= soapGain {
		t.Fatalf("expected fish-rich preserved food gain to dominate competing soap gain, got preserved=%.3f soap=%.3f", preservedGain, soapGain)
	}
}

func TestComputeTradeNodeMarketsExternalRawInputSupportUnlocksCompactSaltChainWithoutDonor(t *testing.T) {
	baseSettings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:             map[string]float64{"raw": 1.0, "processed": 1.0, "default": 1.0},
			CategorySpecializationScale:     map[string]float64{"raw": 0.0, "processed": 0.2, "default": 0.0},
			ManufacturingBaseScale:          map[string]float64{"processed": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:      map[string]float64{"processed": 0.75, "default": 0.55},
			MarketWorkshopBias:              map[string]float64{"processed": 1.10, "default": 1.0},
			MarketInputRichnessScale:        map[string]float64{"processed": 0.16, "default": 0.14},
			MarketConversionShare:           map[string]float64{"processed": 0.40, "default": 0.40},
			MarketImportedInputSupportScale: map[string]float64{"default": 0.0},
			MarketExternalInputSupportScale: map[string]float64{"default": 0.0},
			RawPotentialPivot:               0.50,
			ManufacturingPivot:              0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "fish", Category: "raw", BaseValue: 0.38, Bulkiness: 0.62, SourceWeights: map[string]float64{"fish": 1}},
			{Name: "salt", Category: "raw", BaseValue: 0.62, Bulkiness: 0.38, Perishability: 0.02, SourceWeights: map[string]float64{"salt": 1}},
			{
				Name:                       "preserved_food",
				Category:                   "processed",
				BaseValue:                  0.56,
				Bulkiness:                  0.62,
				MarketConversionScale:      0.36,
				MarketInputReservePriority: 0.90,
				MarketMinNodeKind:          "town",
				Inputs:                     map[string]float64{"fish": 0.35, "salt": 0.20},
			},
		},
	}
	tunedSettings := baseSettings
	tunedSettings.Production.MarketExternalInputSupportScale = map[string]float64{"salt": 0.60, "default": 0.0}

	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{
			{
				NodeID:   0,
				PolityID: 0,
				Wealth:   0.68,
				Supply:   map[string]float64{"fish": 1.40, "salt": 0.03},
				Demand:   map[string]float64{"fish": 0.20, "salt": 0.22, "preserved_food": 0.42},
				Surplus:  map[string]float64{"fish": 1.20, "salt": -0.19, "preserved_food": -0.42},
			},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70, Coastal: true, River: true},
		},
	}
	trade := &TradeNetworkResult{
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.52}},
	}

	base := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, baseSettings, nil, nil, network, trade, nodeGoods)
	tuned := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, tunedSettings, nil, nil, network, trade, nodeGoods)
	if len(base.Markets) != 1 || len(tuned.Markets) != 1 {
		t.Fatalf("expected one market in each result, got base=%d tuned=%d", len(base.Markets), len(tuned.Markets))
	}
	if tuned.Markets[0].Manufactured["preserved_food"] <= base.Markets[0].Manufactured["preserved_food"] {
		t.Fatalf("expected external salt support to unlock preserved food without a donor, got base=%.3f tuned=%.3f", base.Markets[0].Manufactured["preserved_food"], tuned.Markets[0].Manufactured["preserved_food"])
	}
}

func TestComputeTradeNodeMarketsWithRouteSupportImportsSaltFromConnectedPolity(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:             map[string]float64{"raw": 1.0, "processed": 1.0, "default": 1.0},
			CategorySpecializationScale:     map[string]float64{"raw": 0.0, "processed": 0.2, "default": 0.0},
			ManufacturingBaseScale:          map[string]float64{"processed": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:      map[string]float64{"processed": 0.75, "default": 0.55},
			MarketWorkshopBias:              map[string]float64{"processed": 1.10, "default": 1.0},
			MarketInputRichnessScale:        map[string]float64{"processed": 0.16, "default": 0.14},
			MarketConversionShare:           map[string]float64{"processed": 0.40, "default": 0.40},
			MarketImportedInputSupportScale: map[string]float64{"default": 0.0},
			MarketExternalInputSupportScale: map[string]float64{"salt": 1.40, "default": 0.0},
			RawPotentialPivot:               0.50,
			ManufacturingPivot:              0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "fish", Category: "raw", BaseValue: 0.38, Bulkiness: 0.62, SourceWeights: map[string]float64{"fish": 1}},
			{Name: "salt", Category: "raw", BaseValue: 0.62, Bulkiness: 0.38, Perishability: 0.02, SourceWeights: map[string]float64{"salt": 1}},
			{
				Name:                       "preserved_food",
				Category:                   "processed",
				BaseValue:                  0.56,
				Bulkiness:                  0.62,
				MarketConversionScale:      0.36,
				MarketInputReservePriority: 0.90,
				MarketMinNodeKind:          "town",
				Inputs:                     map[string]float64{"fish": 0.35, "salt": 0.20},
			},
		},
	}
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{
			{
				NodeID:   0,
				PolityID: 0,
				Wealth:   0.68,
				Supply:   map[string]float64{"fish": 1.40, "salt": 0.03},
				Demand:   map[string]float64{"fish": 0.20, "salt": 0.22, "preserved_food": 0.42},
				Surplus:  map[string]float64{"fish": 1.20, "salt": -0.19, "preserved_food": -0.42},
			},
			{
				NodeID:   1,
				PolityID: 1,
				Wealth:   0.64,
				Supply:   map[string]float64{"salt": 0.80},
				Demand:   map[string]float64{"salt": 0.08},
				Surplus:  map[string]float64{"salt": 0.72},
			},
		},
	}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{
			{ID: 0, CapitalNode: 0, TerritoryCells: 2, MeanSupport: 0.38},
			{ID: 1, CapitalNode: 1, TerritoryCells: 2, MeanSupport: 0.36},
		},
		Diagnostics: &PolitySphereDiagnostics{
			PolityByCell: []int{0, 1},
			PolityByNode: []int{0, 1},
		},
	}
	polityGoods := &PolityGoodsResult{
		Balances: []PolityGoodBalance{
			{PolityID: 0, Surplus: map[string]float64{"salt": -0.20}},
			{PolityID: 1, Surplus: map[string]float64{"salt": 0.60}},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70, Coastal: true, River: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, CarryingCapacity: 0.55, UrbanPotential: 0.52},
		},
	}
	land := &TradeNetworkResult{
		Corridors: []TradeCorridor{{
			FromNode:    1,
			ToNode:      0,
			Role:        TradeCorridorRoleInterPolityTrunk,
			TravelCost:  4,
			Flow:        0.35,
			MeanSupport: 0.65,
			MeanRisk:    0.10,
		}},
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.52, 0.44}},
	}

	base := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, settings, polities, nil, network, land, nodeGoods)
	tuned := ComputeTradeNodeMarketsWithRouteSupport(nil, &TradeGoodResult{}, settings, polities, nil, network, land, nil, nil, nil, polityGoods, nodeGoods)
	if len(base.Markets) != 2 || len(tuned.Markets) != 2 {
		t.Fatalf("expected two markets in each result, got base=%d tuned=%d", len(base.Markets), len(tuned.Markets))
	}
	if tuned.Markets[0].Manufactured["preserved_food"] <= base.Markets[0].Manufactured["preserved_food"] {
		t.Fatalf("expected connected donor polity to improve preserved food output, got base=%.3f tuned=%.3f", base.Markets[0].Manufactured["preserved_food"], tuned.Markets[0].Manufactured["preserved_food"])
	}
}

func TestComputeTradeNodeMarketsDiagnosticsReportBlockedManufacturingInputs(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.0, "processed": 1.0, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "processed": 0.2, "default": 0.0},
			ManufacturingBaseScale:      map[string]float64{"processed": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"processed": 0.60, "default": 0.55},
			MarketWorkshopBias:          map[string]float64{"processed": 1.10, "default": 1.0},
			MarketInputRichnessScale:    map[string]float64{"processed": 0.12, "default": 0.14},
			MarketConversionShare:       map[string]float64{"processed": 0.32, "default": 0.40},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "salt", Category: "raw", BaseValue: 0.62, Bulkiness: 0.38, SourceWeights: map[string]float64{"salt": 1}},
			{Name: "livestock", Category: "raw", BaseValue: 0.46, Bulkiness: 0.72, SourceWeights: map[string]float64{"pasture": 1}},
			{Name: "herbs", Category: "raw", BaseValue: 0.64, Bulkiness: 0.24, SourceWeights: map[string]float64{"herbs": 1}},
			{
				Name:                  "soap",
				Category:              "processed",
				BaseValue:             0.62,
				Bulkiness:             0.20,
				MarketConversionScale: 0.34,
				Inputs:                map[string]float64{"salt": 0.14, "livestock": 0.10, "herbs": 0.03},
			},
		},
	}
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{{
			NodeID:   0,
			PolityID: 0,
			Wealth:   0.60,
			Supply:   map[string]float64{"salt": 0.12, "livestock": 0.80, "herbs": 0.60, "soap": 0.00},
			Demand:   map[string]float64{"salt": 0.12, "livestock": 0.20, "herbs": 0.10, "soap": 0.35},
			Surplus:  map[string]float64{"salt": 0.00, "livestock": 0.60, "herbs": 0.50, "soap": -0.35},
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
	diag := result.Markets[0].Diagnostics
	if diag.CandidateCount != 1 {
		t.Fatalf("expected one manufacturable candidate, got %d", diag.CandidateCount)
	}
	if len(diag.Blocked) == 0 || diag.Blocked[0].Good != "soap" || diag.Blocked[0].Bottleneck != "salt" {
		t.Fatalf("expected soap to be blocked by salt, got %+v", diag.Blocked)
	}
}

func TestMarketEffectiveManufacturingInputsReserveContestedSupplyForUnlockChain(t *testing.T) {
	production := TradeGoodsProductionSettings{
		MarketInputReservationByInput: map[string]float64{
			"iron_ore": 1.0,
			"coal":     0.7,
		},
		MarketInputReservationStrength: 0.80,
		MarketInputReservationCap:      0.70,
	}
	market := &TradeNodeMarket{
		Supply: map[string]float64{
			"iron_ore": 1.00,
			"coal":     0.40,
		},
		Demand: map[string]float64{
			"iron_ore": 0.10,
			"coal":     0.06,
		},
	}
	ornaments := TradeGoodSpec{
		Name:     "ornaments",
		Category: "finished",
		Inputs:   map[string]float64{"iron_ore": 0.45, "coal": 0.18},
	}
	ironGoods := TradeGoodSpec{
		Name:                       "iron_goods",
		Category:                   "finished",
		MarketInputReservePriority: 0.80,
		Inputs:                     map[string]float64{"iron_ore": 0.45, "coal": 0.18},
	}
	chainPressure := map[string]float64{
		"iron_ore":   1.00,
		"coal":       0.72,
		"iron_goods": 0.68,
	}

	ornamentAccess, _, ornamentCapacity := marketEffectiveManufacturingInputs(ornaments, market, chainPressure, production)
	ironGoodsAccess, _, ironGoodsCapacity := marketEffectiveManufacturingInputs(ironGoods, market, chainPressure, production)

	if ironGoodsAccess <= ornamentAccess {
		t.Fatalf("expected unlock chain to retain better protected input access, got iron_goods=%.3f ornaments=%.3f", ironGoodsAccess, ornamentAccess)
	}
	if ironGoodsCapacity <= ornamentCapacity {
		t.Fatalf("expected unlock chain to retain more effective reserved capacity, got iron_goods=%.3f ornaments=%.3f", ironGoodsCapacity, ornamentCapacity)
	}
}
