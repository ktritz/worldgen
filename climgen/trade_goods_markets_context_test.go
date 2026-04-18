package climgen

import "testing"

func TestComputeTradeNodeMarketsUsesContextForSpecializedManufacturing(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.0, "strategic": 1.0, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "strategic": 0.3, "default": 0.0},
			ManufacturingBaseScale:      map[string]float64{"strategic": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"strategic": 0.90, "default": 0.55},
			MarketWorkshopBias:          map[string]float64{"strategic": 1.15, "default": 1.0},
			MarketInputRichnessScale:    map[string]float64{"strategic": 0.18, "default": 0.14},
			MarketConversionShare:       map[string]float64{"strategic": 0.36, "default": 0.40},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "timber", Category: "raw", BaseValue: 0.42, Bulkiness: 0.90, SourceWeights: map[string]float64{"timber": 1}},
			{Name: "fiber", Category: "raw", BaseValue: 0.40, Bulkiness: 0.58, SourceWeights: map[string]float64{"fiber": 1}},
			{Name: "resin", Category: "raw", BaseValue: 0.56, Bulkiness: 0.42, SourceWeights: map[string]float64{"resin": 1}},
			{
				Name:                     "ships",
				Category:                 "strategic",
				BaseValue:                0.86,
				Bulkiness:                0.98,
				MarketConversionScale:    0.40,
				MarketContextSensitivity: 1.40,
				Inputs:                   map[string]float64{"timber": 0.55, "fiber": 0.22, "resin": 0.18},
				DemandDrivers:            map[string]float64{"coastal": 0.8, "mercantile": 0.5, "urban": 0.2},
			},
		},
	}
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{
			{
				NodeID:   0,
				PolityID: 0,
				Wealth:   0.62,
				Supply:   map[string]float64{"timber": 1.40, "fiber": 0.90, "resin": 0.80, "ships": 0.00},
				Demand:   map[string]float64{"timber": 0.20, "fiber": 0.16, "resin": 0.12, "ships": 0.18},
				Surplus:  map[string]float64{"timber": 1.20, "fiber": 0.74, "resin": 0.68, "ships": -0.18},
			},
			{
				NodeID:   1,
				PolityID: 1,
				Wealth:   0.62,
				Supply:   map[string]float64{"timber": 1.40, "fiber": 0.90, "resin": 0.80, "ships": 0.00},
				Demand:   map[string]float64{"timber": 0.20, "fiber": 0.16, "resin": 0.12, "ships": 0.18},
				Surplus:  map[string]float64{"timber": 1.20, "fiber": 0.74, "resin": 0.68, "ships": -0.18},
			},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70, Coastal: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70},
		},
	}
	trade := &TradeNetworkResult{
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.32, 0.32}},
	}
	result := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, settings, nil, nil, network, trade, nodeGoods)
	if len(result.Markets) != 2 {
		t.Fatalf("expected two markets, got %d", len(result.Markets))
	}
	if result.Markets[0].Supply["ships"] <= result.Markets[1].Supply["ships"] {
		t.Fatalf("expected coastal market to outperform inland market for ships, got coastal=%.3f inland=%.3f", result.Markets[0].Supply["ships"], result.Markets[1].Supply["ships"])
	}
}

func TestComputeTradeNodeMarketsGoodContextSensitivityAmplifiesContextGap(t *testing.T) {
	baseSettings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.0, "strategic": 1.0, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "strategic": 0.3, "default": 0.0},
			ManufacturingBaseScale:      map[string]float64{"strategic": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"strategic": 0.90, "default": 0.55},
			MarketWorkshopBias:          map[string]float64{"strategic": 1.15, "default": 1.0},
			MarketInputRichnessScale:    map[string]float64{"strategic": 0.18, "default": 0.14},
			MarketConversionShare:       map[string]float64{"strategic": 0.36, "default": 0.40},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "timber", Category: "raw", BaseValue: 0.42, Bulkiness: 0.90, SourceWeights: map[string]float64{"timber": 1}},
			{Name: "fiber", Category: "raw", BaseValue: 0.40, Bulkiness: 0.58, SourceWeights: map[string]float64{"fiber": 1}},
			{Name: "resin", Category: "raw", BaseValue: 0.56, Bulkiness: 0.42, SourceWeights: map[string]float64{"resin": 1}},
			{
				Name:                     "ships",
				Category:                 "strategic",
				BaseValue:                0.86,
				Bulkiness:                0.98,
				MarketConversionScale:    0.40,
				MarketContextSensitivity: 0.80,
				Inputs:                   map[string]float64{"timber": 0.55, "fiber": 0.22, "resin": 0.18},
				DemandDrivers:            map[string]float64{"coastal": 0.8, "mercantile": 0.5, "urban": 0.2},
			},
		},
	}
	highSettings := baseSettings
	highSettings.Goods = append([]TradeGoodSpec(nil), baseSettings.Goods...)
	highSettings.Goods[3].MarketContextSensitivity = 1.80
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{
			{
				NodeID:   0,
				PolityID: 0,
				Wealth:   0.62,
				Supply:   map[string]float64{"timber": 1.40, "fiber": 0.90, "resin": 0.80, "ships": 0.00},
				Demand:   map[string]float64{"timber": 0.20, "fiber": 0.16, "resin": 0.12, "ships": 0.18},
				Surplus:  map[string]float64{"timber": 1.20, "fiber": 0.74, "resin": 0.68, "ships": -0.18},
			},
			{
				NodeID:   1,
				PolityID: 1,
				Wealth:   0.62,
				Supply:   map[string]float64{"timber": 1.40, "fiber": 0.90, "resin": 0.80, "ships": 0.00},
				Demand:   map[string]float64{"timber": 0.20, "fiber": 0.16, "resin": 0.12, "ships": 0.18},
				Surplus:  map[string]float64{"timber": 1.20, "fiber": 0.74, "resin": 0.68, "ships": -0.18},
			},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70, Coastal: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70},
		},
	}
	trade := &TradeNetworkResult{
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.32, 0.32}},
	}
	base := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, baseSettings, nil, nil, network, trade, nodeGoods)
	high := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, highSettings, nil, nil, network, trade, nodeGoods)
	baseGap := base.Markets[0].Supply["ships"] - base.Markets[1].Supply["ships"]
	highGap := high.Markets[0].Supply["ships"] - high.Markets[1].Supply["ships"]
	if highGap <= baseGap {
		t.Fatalf("expected higher good context sensitivity to widen coastal ship advantage, got baseGap=%.3f highGap=%.3f", baseGap, highGap)
	}
}

func TestComputeTradeNodeMarketsProductionDriversScopeManufacturingContext(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.0, "processed": 1.0, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "processed": 0.2, "default": 0.0},
			ManufacturingBaseScale:      map[string]float64{"processed": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"processed": 0.75, "default": 0.55},
			MarketWorkshopBias:          map[string]float64{"processed": 1.10, "default": 1.0},
			MarketInputRichnessScale:    map[string]float64{"processed": 0.16, "default": 0.14},
			MarketConversionShare:       map[string]float64{"processed": 0.40, "default": 0.40},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "fish", Category: "raw", BaseValue: 0.38, Bulkiness: 0.62, SourceWeights: map[string]float64{"fish": 1}},
			{Name: "salt", Category: "raw", BaseValue: 0.62, Bulkiness: 0.38, SourceWeights: map[string]float64{"salt": 1}},
			{
				Name:                     "preserved_food",
				Category:                 "processed",
				BaseValue:                0.56,
				Bulkiness:                0.62,
				MarketConversionScale:    0.36,
				MarketContextSensitivity: 1.20,
				ProductionDrivers:        map[string]float64{"coastal": 0.8, "river": 0.7},
				DemandDrivers:            map[string]float64{"urban": 0.8},
				Inputs:                   map[string]float64{"fish": 0.35, "salt": 0.20},
			},
		},
	}
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{
			{
				NodeID:   0,
				PolityID: 0,
				Wealth:   0.62,
				Supply:   map[string]float64{"fish": 1.20, "salt": 0.80, "preserved_food": 0.00},
				Demand:   map[string]float64{"fish": 0.20, "salt": 0.16, "preserved_food": 0.30},
				Surplus:  map[string]float64{"fish": 1.00, "salt": 0.64, "preserved_food": -0.30},
			},
			{
				NodeID:   1,
				PolityID: 1,
				Wealth:   0.62,
				Supply:   map[string]float64{"fish": 1.20, "salt": 0.80, "preserved_food": 0.00},
				Demand:   map[string]float64{"fish": 0.20, "salt": 0.16, "preserved_food": 0.30},
				Surplus:  map[string]float64{"fish": 1.00, "salt": 0.64, "preserved_food": -0.30},
			},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70, Coastal: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70},
		},
	}
	trade := &TradeNetworkResult{
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.32, 0.32}},
	}

	result := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, settings, nil, nil, network, trade, nodeGoods)
	if len(result.Markets) != 2 {
		t.Fatalf("expected two markets, got %d", len(result.Markets))
	}
	if result.Markets[0].Supply["preserved_food"] <= result.Markets[1].Supply["preserved_food"] {
		t.Fatalf("expected coastal/river production drivers to favor preserved food in contextual market, got contextual=%.3f inland=%.3f", result.Markets[0].Supply["preserved_food"], result.Markets[1].Supply["preserved_food"])
	}
}

func TestComputeTradeNodeMarketsMarketMinNodeKindScopesManufacturing(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.0, "processed": 1.0, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "processed": 0.2, "default": 0.0},
			ManufacturingBaseScale:      map[string]float64{"processed": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"processed": 0.75, "default": 0.55},
			MarketWorkshopBias:          map[string]float64{"processed": 1.10, "default": 1.0},
			MarketInputRichnessScale:    map[string]float64{"processed": 0.16, "default": 0.14},
			MarketConversionShare:       map[string]float64{"processed": 0.40, "default": 0.40},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "fish", Category: "raw", BaseValue: 0.38, Bulkiness: 0.62, SourceWeights: map[string]float64{"fish": 1}},
			{Name: "salt", Category: "raw", BaseValue: 0.62, Bulkiness: 0.38, SourceWeights: map[string]float64{"salt": 1}},
			{
				Name:                  "preserved_food",
				Category:              "processed",
				BaseValue:             0.56,
				Bulkiness:             0.62,
				MarketConversionScale: 0.36,
				MarketMinNodeKind:     "town",
				Inputs:                map[string]float64{"fish": 0.35, "salt": 0.20},
			},
		},
	}
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{
			{
				NodeID:   0,
				PolityID: 0,
				Wealth:   0.62,
				Supply:   map[string]float64{"fish": 1.20, "salt": 0.80, "preserved_food": 0.00},
				Demand:   map[string]float64{"fish": 0.20, "salt": 0.16, "preserved_food": 0.30},
				Surplus:  map[string]float64{"fish": 1.00, "salt": 0.64, "preserved_food": -0.30},
			},
			{
				NodeID:   1,
				PolityID: 1,
				Wealth:   0.62,
				Supply:   map[string]float64{"fish": 1.20, "salt": 0.80, "preserved_food": 0.00},
				Demand:   map[string]float64{"fish": 0.20, "salt": 0.16, "preserved_food": 0.30},
				Surplus:  map[string]float64{"fish": 1.00, "salt": 0.64, "preserved_food": -0.30},
			},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70, Coastal: true, River: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeHamlet, CarryingCapacity: 0.60, UrbanPotential: 0.70, Coastal: true, River: true},
		},
	}
	trade := &TradeNetworkResult{
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.32, 0.32}},
	}

	result := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, settings, nil, nil, network, trade, nodeGoods)
	if len(result.Markets) != 2 {
		t.Fatalf("expected two markets, got %d", len(result.Markets))
	}
	if result.Markets[0].Manufactured["preserved_food"] <= 0.01 {
		t.Fatalf("expected town market to manufacture preserved food, got %.3f", result.Markets[0].Manufactured["preserved_food"])
	}
	if result.Markets[1].Manufactured["preserved_food"] > 0.01 {
		t.Fatalf("expected hamlet market to skip preserved food manufacturing, got %.3f", result.Markets[1].Manufactured["preserved_food"])
	}
	if result.Markets[1].Diagnostics.CandidateCount != 0 {
		t.Fatalf("expected ineligible hamlet market to omit preserved food candidate, got %d candidates", result.Markets[1].Diagnostics.CandidateCount)
	}
}

func TestComputeTradeNodeMarketsMarketInputCapabilityFloorScopesManufacturing(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.0, "processed": 1.0, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "processed": 0.2, "default": 0.0},
			ManufacturingBaseScale:      map[string]float64{"processed": 0.55, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"processed": 0.75, "default": 0.55},
			MarketWorkshopBias:          map[string]float64{"processed": 1.10, "default": 1.0},
			MarketInputRichnessScale:    map[string]float64{"processed": 0.16, "default": 0.14},
			MarketConversionShare:       map[string]float64{"processed": 0.40, "default": 0.40},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.40,
		},
		Goods: []TradeGoodSpec{
			{Name: "fish", Category: "raw", BaseValue: 0.38, Bulkiness: 0.62, SourceWeights: map[string]float64{"fish": 1}},
			{Name: "salt", Category: "raw", BaseValue: 0.62, Bulkiness: 0.38, SourceWeights: map[string]float64{"salt": 1}},
			{
				Name:                       "preserved_food",
				Category:                   "processed",
				BaseValue:                  0.56,
				Bulkiness:                  0.62,
				MarketConversionScale:      0.36,
				MarketMinNodeKind:          "town",
				MarketInputCapabilityFloor: map[string]float64{"salt": 0.35},
				Inputs:                     map[string]float64{"fish": 0.35, "salt": 0.20},
			},
		},
	}
	nodeGoods := &NodeGoodsResult{
		Balances: []NodeGoodBalance{
			{
				NodeID:   0,
				PolityID: 0,
				Wealth:   0.62,
				Supply:   map[string]float64{"fish": 1.20, "salt": 0.06, "preserved_food": 0.00},
				Demand:   map[string]float64{"fish": 0.20, "salt": 0.16, "preserved_food": 0.30},
				Surplus:  map[string]float64{"fish": 1.00, "salt": -0.10, "preserved_food": -0.30},
			},
			{
				NodeID:   1,
				PolityID: 1,
				Wealth:   0.62,
				Supply:   map[string]float64{"fish": 1.20, "salt": 0.80, "preserved_food": 0.00},
				Demand:   map[string]float64{"fish": 0.20, "salt": 0.16, "preserved_food": 0.30},
				Surplus:  map[string]float64{"fish": 1.00, "salt": 0.64, "preserved_food": -0.30},
			},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70, Coastal: true, River: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, CarryingCapacity: 0.60, UrbanPotential: 0.70, Coastal: true, River: true},
		},
	}
	trade := &TradeNetworkResult{
		Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.32, 0.32}},
	}

	result := ComputeTradeNodeMarkets(nil, &TradeGoodResult{}, settings, nil, nil, network, trade, nodeGoods)
	if len(result.Markets) != 2 {
		t.Fatalf("expected two markets, got %d", len(result.Markets))
	}
	if result.Markets[0].Manufactured["preserved_food"] > 0.01 {
		t.Fatalf("expected salt-poor town market to skip preserved food manufacturing, got %.3f", result.Markets[0].Manufactured["preserved_food"])
	}
	if result.Markets[1].Manufactured["preserved_food"] <= 0.01 {
		t.Fatalf("expected salt-capable town market to manufacture preserved food, got %.3f", result.Markets[1].Manufactured["preserved_food"])
	}
	if result.Markets[0].Diagnostics.CandidateCount != 0 {
		t.Fatalf("expected salt-poor market to omit preserved food candidate, got %d candidates", result.Markets[0].Diagnostics.CandidateCount)
	}
}
