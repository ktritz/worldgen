package climgen

import "testing"

func TestComputeTradeGoodEndowmentsUsesExistingResourceLayers(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", SourceWeights: map[string]float64{"crop": 1}},
			{Name: "timber", Category: "raw", SourceWeights: map[string]float64{"timber": 1}},
		},
	}
	result := ComputeTradeGoodEndowments(TradeGoodInputs{
		Elevation: []float64{100, 100},
		Agriculture: &AgricultureResult{Diagnostics: &AgricultureDiagnostics{
			CropPotential:    []float64{0.9, 0.1},
			PasturePotential: []float64{0.1, 0.1},
		}},
		Wildlife: &WildlifeResult{Diagnostics: &WildlifeDiagnostics{
			TimberPotential: []float64{0.2, 0.8},
			GamePotential:   []float64{0.1, 0.1},
		}},
	}, settings)
	if len(result.Goods) != 2 {
		t.Fatalf("expected two goods, got %d", len(result.Goods))
	}
	if result.Goods[0].Potential[0] <= result.Goods[0].Potential[1] {
		t.Fatalf("expected grain to follow crop potential, got %.2f %.2f", result.Goods[0].Potential[0], result.Goods[0].Potential[1])
	}
	if result.Goods[1].Potential[1] <= result.Goods[1].Potential[0] {
		t.Fatalf("expected timber to follow timber potential, got %.2f %.2f", result.Goods[1].Potential[0], result.Goods[1].Potential[1])
	}
}

func TestComputeTradeGoodEndowmentsSupportsFreshwaterFish(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "fish", Category: "raw", SourceWeights: map[string]float64{"fish": 1}},
		},
	}
	result := ComputeTradeGoodEndowments(TradeGoodInputs{
		Elevation: []float64{100},
		Water: &WaterResourceResult{Diagnostics: &WaterResourceDiagnostics{
			SurfaceReliability: []float64{0.82},
			LakeAccess:         []float64{0.74},
		}},
		Vegetation: &VegetationResult{Diagnostics: &VegetationDiagnostics{
			WetlandCover:     []float64{0.30},
			RiparianAffinity: []float64{0.64},
		}},
		Hydro: &HydrologyBiomeInputs{
			Runoff:          []float64{72},
			ChannelStrength: []float64{1.8},
			CellClass:       []string{"river"},
		},
	}, settings)
	if result.Goods[0].Potential[0] <= 0.45 {
		t.Fatalf("expected inland water diagnostics to create freshwater fish potential, got %.2f", result.Goods[0].Potential[0])
	}
}

func TestComputeTradeGoodEndowmentsSupportsCoastalShellfishForFish(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "fish", Category: "raw", SourceWeights: map[string]float64{"fish": 1}},
		},
	}
	result := ComputeTradeGoodEndowments(TradeGoodInputs{
		Elevation: []float64{100},
		Coastal: &CoastalResourceResult{Diagnostics: &CoastalResourceDiagnostics{
			OpenFishery:      []float64{0.20},
			EstuarineFishery: []float64{0.18},
			ShellfishPotential: []float64{0.92},
		}},
	}, settings)
	if result.Goods[0].Potential[0] <= 0.20 {
		t.Fatalf("expected shellfish-rich coast to contribute to fish potential, got %.2f", result.Goods[0].Potential[0])
	}
}

func TestComputeTradeGoodEndowmentsSupportsInlandSaltBasins(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "salt", Category: "raw", SourceWeights: map[string]float64{"salt": 1}},
		},
	}
	result := ComputeTradeGoodEndowments(TradeGoodInputs{
		Elevation: []float64{100},
		Biome: &BiomeResult{Diagnostics: &BiomeDiagnostics{
			AridityRatio: []float64{0.42},
		}},
		Soils: &SoilResult{Diagnostics: &SoilDiagnostics{
			Salinity: []float64{0.78},
		}},
		Water: &WaterResourceResult{Diagnostics: &WaterResourceDiagnostics{
			LakeAccess:           []float64{0.84},
			GroundwaterPotential: []float64{0.58},
		}},
		Resources: &ResourceResult{
			Types: []ResourceType{ResourceEvaporite},
			Diagnostics: &ResourceDiagnostics{
				EvaporiteAffinity: []float64{0.46},
			},
		},
		Hydro: &HydrologyBiomeInputs{
			Runoff:    []float64{8},
			CellClass: []string{"endorheic_basin"},
		},
	}, settings)
	if result.Goods[0].Potential[0] <= 0.50 {
		t.Fatalf("expected inland saline basin to create salt potential, got %.2f", result.Goods[0].Potential[0])
	}
}

func TestComputeTradeGoodEndowmentsSupportsBogIronSources(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "iron_ore", Category: "raw", SourceWeights: map[string]float64{"iron_ore": 1}},
		},
	}
	result := ComputeTradeGoodEndowments(TradeGoodInputs{
		Elevation: []float64{100},
		Soils: &SoilResult{Diagnostics: &SoilDiagnostics{
			Alluvial: []float64{0.78},
			Organic:  []float64{0.72},
		}},
		Vegetation: &VegetationResult{Diagnostics: &VegetationDiagnostics{
			WetlandCover: []float64{0.84},
		}},
		Hydro: &HydrologyBiomeInputs{
			Runoff:          []float64{48},
			ChannelStrength: []float64{1.3},
			CellClass:       []string{"floodplain"},
		},
	}, settings)
	if result.Goods[0].Potential[0] <= 0.20 {
		t.Fatalf("expected wet alluvial context to create low-grade bog iron potential, got %.2f", result.Goods[0].Potential[0])
	}
}

func TestComputeTradeGoodEndowmentsSupportsPlacerPreciousOre(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "precious_ore", Category: "raw", SourceWeights: map[string]float64{"gold_ore": 1}},
		},
	}
	result := ComputeTradeGoodEndowments(TradeGoodInputs{
		Elevation: []float64{100},
		Soils: &SoilResult{Diagnostics: &SoilDiagnostics{
			Alluvial: []float64{0.82},
		}},
		Vegetation: &VegetationResult{Diagnostics: &VegetationDiagnostics{
			WetlandCover:     []float64{0.36},
			RiparianAffinity: []float64{0.74},
		}},
		Hydro: &HydrologyBiomeInputs{
			Runoff:          []float64{62},
			ChannelStrength: []float64{1.6},
			CellClass:       []string{"delta"},
		},
		Resources: &ResourceResult{
			Types: []ResourceType{ResourceNone},
			Diagnostics: &ResourceDiagnostics{
				GoldAffinity: []float64{0.0},
			},
		},
	}, settings)
	if result.Goods[0].Potential[0] <= 0.18 {
		t.Fatalf("expected alluvial placer context to create precious ore potential, got %.2f", result.Goods[0].Potential[0])
	}
}

func TestComputeTradeGoodEndowmentsUsesResourcePlacerAffinityForPreciousOre(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "precious_ore", Category: "raw", SourceWeights: map[string]float64{"gold_ore": 1}},
		},
	}
	result := ComputeTradeGoodEndowments(TradeGoodInputs{
		Elevation: []float64{100},
		Soils: &SoilResult{Diagnostics: &SoilDiagnostics{
			Alluvial: []float64{0.70},
		}},
		Vegetation: &VegetationResult{Diagnostics: &VegetationDiagnostics{
			WetlandCover:     []float64{0.20},
			RiparianAffinity: []float64{0.58},
		}},
		Hydro: &HydrologyBiomeInputs{
			Runoff:          []float64{54},
			ChannelStrength: []float64{1.4},
			CellClass:       []string{"floodplain"},
		},
		Resources: &ResourceResult{
			Types: []ResourceType{ResourceNone},
			Diagnostics: &ResourceDiagnostics{
				GoldAffinity:   []float64{0.0},
				PlacerAffinity: []float64{0.78},
			},
		},
	}, settings)
	if result.Goods[0].Potential[0] <= 0.18 {
		t.Fatalf("expected placer affinity to contribute to precious ore potential, got %.2f", result.Goods[0].Potential[0])
	}
}

func TestComputeTradeGoodEndowmentsSupportsPlacerCopperOre(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "copper_ore", Category: "raw", SourceWeights: map[string]float64{"copper_ore": 1}},
		},
	}
	result := ComputeTradeGoodEndowments(TradeGoodInputs{
		Elevation: []float64{100},
		Soils: &SoilResult{Diagnostics: &SoilDiagnostics{
			Alluvial: []float64{0.78},
		}},
		Vegetation: &VegetationResult{Diagnostics: &VegetationDiagnostics{
			WetlandCover:     []float64{0.22},
			RiparianAffinity: []float64{0.66},
		}},
		Hydro: &HydrologyBiomeInputs{
			Runoff:          []float64{58},
			ChannelStrength: []float64{1.5},
			CellClass:       []string{"confluence"},
		},
		Resources: &ResourceResult{
			Types: []ResourceType{ResourceNone},
			Diagnostics: &ResourceDiagnostics{
				CopperAffinity: []float64{0.0},
				PlacerAffinity: []float64{0.72},
			},
		},
	}, settings)
	if result.Goods[0].Potential[0] <= 0.12 {
		t.Fatalf("expected alluvial placer context to create copper ore potential, got %.2f", result.Goods[0].Potential[0])
	}
}

func TestComputeTradeGoodEndowmentsBoostsClassifiedResourceDeposits(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "iron_ore", Category: "raw", SourceWeights: map[string]float64{"iron_ore": 1}},
			{Name: "coal", Category: "raw", SourceWeights: map[string]float64{"coal": 1}},
		},
	}
	result := ComputeTradeGoodEndowments(TradeGoodInputs{
		Elevation: []float64{100, 100},
		Resources: &ResourceResult{
			Types: []ResourceType{ResourceIronOre, ResourceCoal},
			Diagnostics: &ResourceDiagnostics{
				IronAffinity: []float64{0.22, 0.10},
				CoalAffinity: []float64{0.08, 0.24},
			},
		},
	}, settings)
	if result.Goods[0].Potential[0] <= 0.70 {
		t.Fatalf("expected classified iron deposit to boost iron ore potential, got %.2f", result.Goods[0].Potential[0])
	}
	if result.Goods[1].Potential[1] <= 0.68 {
		t.Fatalf("expected classified coal deposit to boost coal potential, got %.2f", result.Goods[1].Potential[1])
	}
}

func TestComputePolityGoodsAppliesProfileSupplyAndDemandPreferences(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{
				Name:                      "iron_ore",
				Category:                  "raw",
				BaseValue:                 0.5,
				Bulkiness:                 0.9,
				SourceWeights:             map[string]float64{"iron_ore": 1},
				ProfileProductionAffinity: map[string]float64{"Dwarf": 1.4},
				ProfileDemandAffinity:     map[string]float64{"Human": 1.2},
			},
			{
				Name:      "iron_goods",
				Category:  "finished",
				BaseValue: 0.8,
				Bulkiness: 0.4,
				Inputs:    map[string]float64{"iron_ore": 0.4},
				DemandDrivers: map[string]float64{
					"urban":   0.5,
					"warlike": 0.5,
				},
				ProfileProductionAffinity: map[string]float64{"Dwarf": 1.3},
			},
		},
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "iron_ore", Category: "raw", Potential: []float64{0.9, 0.8, 0.1, 0.1}},
			{Good: "iron_goods", Category: "finished", Potential: []float64{0, 0, 0, 0}},
		},
	}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{
			{ID: 0, CapitalNode: 0, TerritoryCells: 2, MeanSupport: 0.36},
			{ID: 1, CapitalNode: 1, TerritoryCells: 2, MeanSupport: 0.36},
		},
		Diagnostics: &PolitySphereDiagnostics{PolityByCell: []int{0, 0, 1, 1}},
	}
	profiles := &PolityProfileResult{Assignments: []PolityProfileAssignment{
		{PolityID: 0, Profile: ResolvedProfile{AncestryName: "Dwarf", Attitudes: &ProfileAttitudeModule{Aggression: 0.2}}},
		{PolityID: 1, Profile: ResolvedProfile{AncestryName: "Human", Attitudes: &ProfileAttitudeModule{Aggression: 0.2}}},
	}}
	network := &SettlementNetworkResult{Nodes: []SettlementNode{
		{ID: 0, Kind: SettlementNodeTown},
		{ID: 1, Kind: SettlementNodeVillage},
	}}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.3, 0.05}}}

	result := ComputePolityGoods(goods, settings, polities, profiles, network, trade)
	if len(result.Balances) != 2 {
		t.Fatalf("expected two polity balances, got %d", len(result.Balances))
	}
	dwarf := result.Balances[0]
	human := result.Balances[1]
	if dwarf.Supply["iron_ore"] <= human.Supply["iron_ore"] {
		t.Fatalf("expected dwarf polity to produce more iron ore, got dwarf=%.2f human=%.2f", dwarf.Supply["iron_ore"], human.Supply["iron_ore"])
	}
	if dwarf.Supply["iron_goods"] <= 0 {
		t.Fatalf("expected dwarf polity to produce processed iron goods")
	}
	if human.Demand["iron_ore"] <= dwarf.Demand["iron_ore"] {
		t.Fatalf("expected human demand affinity to raise iron ore demand, got human=%.2f dwarf=%.2f", human.Demand["iron_ore"], dwarf.Demand["iron_ore"])
	}
}

func TestComputePolityGoodsWithNodeMarketsRaisesLuxuryDemandForWealthyPolity(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{
				Name:      "jewelry",
				Category:  "luxury",
				BaseValue: 0.90,
				Bulkiness: 0.18,
				Inputs:    map[string]float64{"precious_ore": 0.3},
			},
			{
				Name:          "precious_ore",
				Category:      "raw",
				BaseValue:     0.82,
				Bulkiness:     0.22,
				SourceWeights: map[string]float64{"gold_ore": 1},
			},
		},
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "jewelry", Category: "luxury", Potential: []float64{0, 0, 0, 0}},
			{Good: "precious_ore", Category: "raw", Potential: []float64{0.4, 0.4, 0.4, 0.4}},
		},
	}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{
			{ID: 0, CapitalNode: 0, TerritoryCells: 2, MeanSupport: 0.36},
			{ID: 1, CapitalNode: 1, TerritoryCells: 2, MeanSupport: 0.36},
		},
		Diagnostics: &PolitySphereDiagnostics{PolityByCell: []int{0, 0, 1, 1}},
	}
	network := &SettlementNetworkResult{Nodes: []SettlementNode{
		{ID: 0, Kind: SettlementNodeTown},
		{ID: 1, Kind: SettlementNodeTown},
	}}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.15, 0.15}}}
	nodeGoods := &NodeGoodsResult{PolityMarketWealth: map[int]float64{0: 0.18, 1: 0.82}}

	result := ComputePolityGoodsWithNodeMarkets(goods, settings, polities, nil, network, trade, nodeGoods)
	if result.Balances[1].MarketWealth <= result.Balances[0].MarketWealth {
		t.Fatalf("expected polity 1 to keep higher market wealth")
	}
	if result.Balances[1].Demand["jewelry"] <= result.Balances[0].Demand["jewelry"] {
		t.Fatalf("expected wealthy polity to demand more jewelry, got poor=%.3f wealthy=%.3f", result.Balances[0].Demand["jewelry"], result.Balances[1].Demand["jewelry"])
	}
}

func TestComputeTradeGoodEndowmentsComputesInputBasedScarcity(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.42, Bulkiness: 0.86, SourceWeights: map[string]float64{"crop": 1}},
			{Name: "cloth", Category: "processed", BaseValue: 0.62, Bulkiness: 0.32, Inputs: map[string]float64{"grain": 0.4}},
		},
	}
	result := ComputeTradeGoodEndowments(TradeGoodInputs{
		Elevation: []float64{100, 100},
		Agriculture: &AgricultureResult{Diagnostics: &AgricultureDiagnostics{
			CropPotential: []float64{0.8, 0.6},
		}},
	}, settings)
	if result.Diagnostics == nil {
		t.Fatalf("expected diagnostics")
	}
	if result.Diagnostics.ScarcityByGood["cloth"] > 0.80 {
		t.Fatalf("expected processed cloth scarcity to inherit from grain rather than collapse to maximal scarcity; grain=%.3f cloth=%.3f", result.Diagnostics.ScarcityByGood["grain"], result.Diagnostics.ScarcityByGood["cloth"])
	}
}

func TestComputePolityGoodsScarcityRaisesDemand(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "common_good", Category: "raw", BaseValue: 0.50, Bulkiness: 0.40},
			{Name: "rare_good", Category: "raw", BaseValue: 0.50, Bulkiness: 0.40},
		},
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "common_good", Category: "raw", Potential: []float64{0.8, 0.8}},
			{Good: "rare_good", Category: "raw", Potential: []float64{0.1, 0.1}},
		},
		Diagnostics: &TradeGoodDiagnostics{
			ScarcityByGood: map[string]float64{"common_good": 0.10, "rare_good": 0.90},
		},
	}
	polities := &PolitySphereResult{
		Spheres:     []PolitySphere{{ID: 0, CapitalNode: 0, TerritoryCells: 2, MeanSupport: 0.36}},
		Diagnostics: &PolitySphereDiagnostics{PolityByCell: []int{0, 0}},
	}
	network := &SettlementNetworkResult{Nodes: []SettlementNode{{ID: 0, Kind: SettlementNodeTown}}}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.1}}}

	result := ComputePolityGoods(goods, settings, polities, nil, network, trade)
	if result.Balances[0].Demand["rare_good"] <= result.Balances[0].Demand["common_good"] {
		t.Fatalf("expected scarce good to have stronger demand, got common=%.3f rare=%.3f", result.Balances[0].Demand["common_good"], result.Balances[0].Demand["rare_good"])
	}
}

func TestComputePolityGoodsProductionSpecializationAmplifiesStrongProducer(t *testing.T) {
	baseSettings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.00, "default": 1.00},
			CategorySpecializationScale: map[string]float64{"raw": 0.00, "default": 0.00},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.50,
		},
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.42, Bulkiness: 0.86, SourceWeights: map[string]float64{"crop": 1}},
		},
	}
	specializedSettings := baseSettings
	specializedSettings.Production = TradeGoodsProductionSettings{
		CategorySupplyScale:         map[string]float64{"raw": 1.00, "default": 1.00},
		CategorySpecializationScale: map[string]float64{"raw": 1.50, "default": 0.00},
		RawPotentialPivot:           0.50,
		ManufacturingPivot:          0.50,
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "grain", Category: "raw", Potential: []float64{0.95, 0.90, 0.18, 0.16}},
		},
	}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{
			{ID: 0, CapitalNode: 0, TerritoryCells: 2, MeanSupport: 0.38},
			{ID: 1, CapitalNode: 1, TerritoryCells: 2, MeanSupport: 0.38},
		},
		Diagnostics: &PolitySphereDiagnostics{PolityByCell: []int{0, 0, 1, 1}},
	}
	network := &SettlementNetworkResult{Nodes: []SettlementNode{
		{ID: 0, Kind: SettlementNodeTown},
		{ID: 1, Kind: SettlementNodeTown},
	}}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.12, 0.12}}}

	baseline := ComputePolityGoods(goods, baseSettings, polities, nil, network, trade)
	specialized := ComputePolityGoods(goods, specializedSettings, polities, nil, network, trade)

	baseGap := baseline.Balances[0].Supply["grain"] - baseline.Balances[1].Supply["grain"]
	specializedGap := specialized.Balances[0].Supply["grain"] - specialized.Balances[1].Supply["grain"]
	if specializedGap <= baseGap {
		t.Fatalf("expected specialization tuning to widen producer advantage, got baselineGap=%.3f specializedGap=%.3f", baseGap, specializedGap)
	}
}

func TestComputePolityGoodsManufacturingTuningAmplifiesFinishedGoods(t *testing.T) {
	baseSettings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 1.00, "finished": 1.00, "default": 1.00},
			CategorySpecializationScale: map[string]float64{"raw": 0.00, "finished": 0.00, "default": 0.00},
			ManufacturingBaseScale:      map[string]float64{"finished": 0.30, "default": 0.45},
			ManufacturingWorkshopScale:  map[string]float64{"finished": 0.30, "default": 0.55},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.50,
		},
		Goods: []TradeGoodSpec{
			{Name: "iron_ore", Category: "raw", BaseValue: 0.58, Bulkiness: 0.95, SourceWeights: map[string]float64{"iron_ore": 1}},
			{Name: "iron_goods", Category: "finished", BaseValue: 0.72, Bulkiness: 0.42, Inputs: map[string]float64{"iron_ore": 0.5}},
		},
	}
	tunedSettings := baseSettings
	tunedSettings.Production = TradeGoodsProductionSettings{
		CategorySupplyScale:         map[string]float64{"raw": 1.00, "finished": 1.00, "default": 1.00},
		CategorySpecializationScale: map[string]float64{"raw": 0.00, "finished": 0.60, "default": 0.00},
		ManufacturingBaseScale:      map[string]float64{"finished": 0.55, "default": 0.45},
		ManufacturingWorkshopScale:  map[string]float64{"finished": 1.05, "default": 0.55},
		RawPotentialPivot:           0.50,
		ManufacturingPivot:          0.40,
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "iron_ore", Category: "raw", Potential: []float64{0.90, 0.82}},
			{Good: "iron_goods", Category: "finished", Potential: []float64{0.00, 0.00}},
		},
	}
	polities := &PolitySphereResult{
		Spheres:     []PolitySphere{{ID: 0, CapitalNode: 0, TerritoryCells: 2, MeanSupport: 0.40}},
		Diagnostics: &PolitySphereDiagnostics{PolityByCell: []int{0, 0}},
	}
	network := &SettlementNetworkResult{Nodes: []SettlementNode{{ID: 0, Kind: SettlementNodeTown}}}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.24}}}

	baseline := ComputePolityGoods(goods, baseSettings, polities, nil, network, trade)
	tuned := ComputePolityGoods(goods, tunedSettings, polities, nil, network, trade)

	if tuned.Balances[0].Supply["iron_goods"] <= baseline.Balances[0].Supply["iron_goods"] {
		t.Fatalf("expected manufacturing tuning to raise finished goods supply, got baseline=%.3f tuned=%.3f", baseline.Balances[0].Supply["iron_goods"], tuned.Balances[0].Supply["iron_goods"])
	}
}

func TestComputePolityGoodsDemandReliefReducesSelfSufficientRawDemand(t *testing.T) {
	settings := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Production: TradeGoodsProductionSettings{
			CategorySupplyScale:         map[string]float64{"raw": 0.80, "default": 1.0},
			CategorySpecializationScale: map[string]float64{"raw": 0.0, "default": 0.0},
			RawPotentialPivot:           0.50,
			ManufacturingPivot:          0.50,
		},
		Demand: TradeGoodsDemandSettings{
			CategoryDemandScale:         map[string]float64{"raw": 1.0, "default": 1.0},
			LocalSupplyReliefByCategory: map[string]float64{"raw": 0.90, "default": 0.0},
			DriverSpecializationScale:   map[string]float64{"raw": 0.0, "default": 0.0},
			DriverSpecializationPivot:   0.50,
		},
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw", BaseValue: 0.42, Bulkiness: 0.86, SourceWeights: map[string]float64{"crop": 1}},
		},
	}
	goods := &TradeGoodResult{
		Goods: []TradeGoodEndowment{
			{Good: "grain", Category: "raw", Potential: []float64{0.95, 0.90, 0.12, 0.10}},
		},
	}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{
			{ID: 0, CapitalNode: 0, TerritoryCells: 2, MeanSupport: 0.38},
			{ID: 1, CapitalNode: 1, TerritoryCells: 2, MeanSupport: 0.38},
		},
		Diagnostics: &PolitySphereDiagnostics{PolityByCell: []int{0, 0, 1, 1}},
	}
	network := &SettlementNetworkResult{Nodes: []SettlementNode{
		{ID: 0, Kind: SettlementNodeTown},
		{ID: 1, Kind: SettlementNodeTown},
	}}
	trade := &TradeNetworkResult{Diagnostics: &TradeNetworkDiagnostics{NodeCentrality: []float64{0.12, 0.12}}}

	result := ComputePolityGoods(goods, settings, polities, nil, network, trade)
	if result.Balances[0].Demand["grain"] >= result.Balances[1].Demand["grain"] {
		t.Fatalf("expected self-sufficient grain polity to demand less grain, got richSupply=%.3f weakSupply=%.3f", result.Balances[0].Demand["grain"], result.Balances[1].Demand["grain"])
	}
}
