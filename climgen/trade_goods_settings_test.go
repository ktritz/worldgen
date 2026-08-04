package climgen

import "testing"

func TestLoadTradeGoodsSettingsDataValidatesCatalog(t *testing.T) {
	settings, err := loadTradeGoodsSettingsData([]byte(`{
		"schemaVersion": "trade-goods/v1",
		"goods": [
			{
				"name": "grain",
				"category": "raw",
				"baseValue": 0.4,
				"bulkiness": 0.8,
				"perishability": 0.3,
				"productionDrivers": {"river": 0.6},
				"sourceWeights": {"crop": 1.0},
				"profileDemandAffinity": {"Dwarf": 1.2}
			}
		]
	}`))
	if err != nil {
		t.Fatalf("expected settings to load: %v", err)
	}
	if len(settings.Goods) != 1 || settings.Goods[0].Name != "grain" {
		t.Fatalf("unexpected goods catalog: %+v", settings.Goods)
	}
	if settings.Goods[0].MarketConversionScale != 0 {
		t.Fatalf("expected omitted market conversion scale to remain zero, got %+v", settings.Goods[0])
	}
	if settings.Goods[0].MarketContextSensitivity != 0 {
		t.Fatalf("expected omitted market context sensitivity to remain zero, got %+v", settings.Goods[0])
	}
	if settings.Goods[0].MarketDominancePenalty != 0 {
		t.Fatalf("expected omitted market dominance penalty to remain zero, got %+v", settings.Goods[0])
	}
	if settings.Goods[0].MarketInputReservePriority != 0 {
		t.Fatalf("expected omitted market input reserve priority to remain zero, got %+v", settings.Goods[0])
	}
	if settings.Goods[0].MarketMinNodeKind != "" {
		t.Fatalf("expected omitted market min node kind to remain empty, got %+v", settings.Goods[0])
	}
	if len(settings.Goods[0].MarketInputCapabilityFloor) != 0 {
		t.Fatalf("expected omitted market input capability floor to remain empty, got %+v", settings.Goods[0])
	}
	if settings.Goods[0].RawCatchmentSensitivity != 0 {
		t.Fatalf("expected omitted raw catchment sensitivity to remain zero, got %+v", settings.Goods[0])
	}
	if settings.Goods[0].ProductionDrivers["river"] != 0.6 {
		t.Fatalf("expected production drivers to load, got %+v", settings.Goods[0])
	}
	if _, ok := settings.GoodByName("grain"); !ok {
		t.Fatalf("expected GoodByName to find grain")
	}
}

func TestValidateTradeGoodsSettingsRejectsDuplicateGoods(t *testing.T) {
	err := ValidateTradeGoodsSettings(TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw"},
			{Name: "grain", Category: "raw"},
		},
	})
	if err == nil {
		t.Fatalf("expected duplicate good validation error")
	}
}

func TestValidateTradeGoodsSettingsRejectsInvalidMarketMinNodeKind(t *testing.T) {
	err := ValidateTradeGoodsSettings(TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "preserved_food", Category: "processed", BaseValue: 0.4, MarketMinNodeKind: "outpost"},
		},
	})
	if err == nil {
		t.Fatalf("expected invalid market min node kind validation error")
	}
}

func TestValidateTradeGoodsSettingsRejectsInvalidMarketInputCapabilityFloor(t *testing.T) {
	err := ValidateTradeGoodsSettings(TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{
				Name:                       "preserved_food",
				Category:                   "processed",
				BaseValue:                  0.4,
				Inputs:                     map[string]float64{"fish": 0.35, "salt": 0.20},
				MarketInputCapabilityFloor: map[string]float64{"coal": 0.25},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected invalid market input capability floor validation error")
	}
}

func TestDefaultTradeGoodsSettingsUsesEmbeddedCatalog(t *testing.T) {
	settings := DefaultTradeGoodsSettings()
	if len(settings.Goods) < 8 {
		t.Fatalf("expected embedded trade goods catalog, got %d goods", len(settings.Goods))
	}
	if _, ok := settings.GoodByName("iron_goods"); !ok {
		t.Fatalf("expected embedded catalog to include processed iron goods")
	}
	if settings.Scarcity.RawAvailability.CoverageThreshold <= 0 {
		t.Fatalf("expected embedded/default scarcity tuning to be populated")
	}
	if settings.Multimodal.LowCapacityVolumeThreshold <= 0 {
		t.Fatalf("expected embedded/default multimodal tuning to be populated")
	}
	if settings.Production.RawPotentialPivot <= 0 {
		t.Fatalf("expected embedded/default production tuning to be populated")
	}
	if settings.Production.ManufacturingBaseScale["finished"] <= 0 || settings.Production.ManufacturingWorkshopScale["finished"] <= 0 {
		t.Fatalf("expected embedded/default manufacturing tuning to be populated")
	}
	if settings.Production.MarketWorkshopBias["finished"] <= 0 || settings.Production.MarketInputRichnessScale["finished"] < 0 {
		t.Fatalf("expected embedded/default market workshop tuning to be populated")
	}
	if settings.Production.MarketConversionShare["finished"] <= 0 {
		t.Fatalf("expected embedded/default market conversion tuning to be populated")
	}
	if settings.Production.RawCatchmentSpecializationScale <= 0 || settings.Production.RawCatchmentSpecializationFloor <= 0 {
		t.Fatalf("expected embedded/default raw catchment specialization tuning to be populated")
	}
	if settings.Demand.DriverSpecializationPivot <= 0 {
		t.Fatalf("expected embedded/default demand tuning to be populated")
	}
	if settings.Demand.MarketCategoryDemandScale["finished"] <= 0 || settings.Demand.MarketWealthPullScale["finished"] < 0 {
		t.Fatalf("expected embedded/default market demand tuning to be populated")
	}
}

func TestLoadTradeGoodsSettingsDataAppliesScarcityDefaultsAndOverrides(t *testing.T) {
	settings, err := loadTradeGoodsSettingsData([]byte(`{
		"schemaVersion": "trade-goods/v1",
		"scarcity": {
			"categoryScarcityPower": {
				"raw": 0.45
			}
		},
		"goods": [
			{
				"name": "grain",
				"category": "raw",
				"baseValue": 0.4,
				"bulkiness": 0.8,
				"perishability": 0.3,
				"sourceWeights": {"crop": 1.0}
			}
		]
	}`))
	if err != nil {
		t.Fatalf("expected settings to load: %v", err)
	}
	if settings.Scarcity.CategoryScarcityPower["raw"] != 0.45 {
		t.Fatalf("expected raw scarcity power override, got %.2f", settings.Scarcity.CategoryScarcityPower["raw"])
	}
	if settings.Scarcity.DemandResponse["raw"].Base != 0.28 {
		t.Fatalf("expected missing demand response tuning to inherit defaults, got %+v", settings.Scarcity.DemandResponse["raw"])
	}
	if settings.Scarcity.InputAvailability.MinWeight <= 0 || settings.Scarcity.InputAvailability.AvgWeight <= 0 {
		t.Fatalf("expected missing scarcity tuning fields to inherit defaults: %+v", settings.Scarcity.InputAvailability)
	}
	if settings.Multimodal.VolumeBaseByMode["river"] <= 0 {
		t.Fatalf("expected missing multimodal tuning fields to inherit defaults: %+v", settings.Multimodal)
	}
	if settings.Production.CategorySupplyScale["raw"] <= 0 {
		t.Fatalf("expected missing production tuning fields to inherit defaults: %+v", settings.Production)
	}
	if settings.Production.ManufacturingBaseScale["processed"] <= 0 {
		t.Fatalf("expected missing manufacturing tuning fields to inherit defaults: %+v", settings.Production)
	}
	if settings.Production.MarketConversionShare["processed"] <= 0 {
		t.Fatalf("expected missing market conversion tuning fields to inherit defaults: %+v", settings.Production)
	}
	if settings.Production.RawCatchmentSpecializationScale <= 0 {
		t.Fatalf("expected missing raw catchment specialization tuning to inherit defaults: %+v", settings.Production)
	}
	if settings.Demand.CategoryDemandScale["raw"] <= 0 {
		t.Fatalf("expected missing demand tuning fields to inherit defaults: %+v", settings.Demand)
	}
	if settings.Demand.MarketCategoryDemandScale["finished"] <= 0 {
		t.Fatalf("expected missing market demand tuning fields to inherit defaults: %+v", settings.Demand)
	}
}

func TestValidateTradeGoodsSettingsRejectsInvalidScarcityThresholds(t *testing.T) {
	err := ValidateTradeGoodsSettings(TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Scarcity: TradeGoodsScarcitySettings{
			RawAvailability: TradeGoodsRawAvailabilitySettings{
				MeanWeight:              0.4,
				CoverageWeight:          0.3,
				StrongCoverageWeight:    0.2,
				PeakWeight:              0.1,
				CoverageThreshold:       0.4,
				StrongCoverageThreshold: 0.2,
			},
			InputAvailability:       DefaultTradeGoodsScarcitySettings().InputAvailability,
			CategoryAvailabilityFit: DefaultTradeGoodsScarcitySettings().CategoryAvailabilityFit,
			CategoryScarcityPower:   DefaultTradeGoodsScarcitySettings().CategoryScarcityPower,
		},
		Goods: []TradeGoodSpec{
			{Name: "grain", Category: "raw"},
		},
	})
	if err == nil {
		t.Fatalf("expected invalid scarcity threshold validation error")
	}
}

func TestLoadTradeGoodsSettingsDataAppliesResponseCurveOverrides(t *testing.T) {
	settings, err := loadTradeGoodsSettingsData([]byte(`{
		"schemaVersion": "trade-goods/v1",
		"scarcity": {
			"tradeValueResponse": {
				"luxury": {"base": 1.05, "slope": 0.22}
			}
		},
		"goods": [
			{
				"name": "jewelry",
				"category": "luxury",
				"baseValue": 0.9,
				"bulkiness": 0.2,
				"perishability": 0.0,
				"sourceWeights": {"gemstones": 1.0}
			}
		]
	}`))
	if err != nil {
		t.Fatalf("expected settings to load: %v", err)
	}
	curve := settings.Scarcity.TradeValueResponse["luxury"]
	if curve.Base != 1.05 || curve.Slope != 0.22 {
		t.Fatalf("expected trade value response override, got %+v", curve)
	}
}

func TestLoadTradeGoodsSettingsDataAppliesMultimodalOverrides(t *testing.T) {
	settings, err := loadTradeGoodsSettingsData([]byte(`{
		"schemaVersion": "trade-goods/v1",
		"multimodal": {
			"endpointNeedShareByCategory": {
				"processed": 0.4
			},
			"volumeBaseByMode": {
				"river": 30.0
			},
			"lowCapacityVolumeThreshold": 3.5
		},
		"goods": [
			{
				"name": "grain",
				"category": "raw",
				"baseValue": 0.4,
				"bulkiness": 0.8,
				"perishability": 0.3,
				"sourceWeights": {"crop": 1.0}
			}
		]
	}`))
	if err != nil {
		t.Fatalf("expected settings to load: %v", err)
	}
	if settings.Multimodal.VolumeBaseByMode["river"] != 30.0 {
		t.Fatalf("expected multimodal river volume override, got %+v", settings.Multimodal.VolumeBaseByMode)
	}
	if settings.Multimodal.EndpointNeedShareByCategory["processed"] != 0.4 {
		t.Fatalf("expected multimodal endpoint need override, got %+v", settings.Multimodal.EndpointNeedShareByCategory)
	}
	if settings.Multimodal.LowCapacityVolumeThreshold != 3.5 {
		t.Fatalf("expected multimodal low-capacity threshold override, got %+v", settings.Multimodal)
	}
	if settings.Multimodal.LocalNeedResponse["raw"].Base <= 0 {
		t.Fatalf("expected multimodal response defaults to remain populated, got %+v", settings.Multimodal)
	}
}

func TestLoadTradeGoodsSettingsDataAppliesProductionOverrides(t *testing.T) {
	settings, err := loadTradeGoodsSettingsData([]byte(`{
		"schemaVersion": "trade-goods/v1",
		"production": {
			"categorySupplyScale": {
				"raw": 1.25
			},
			"manufacturingWorkshopScale": {
				"finished": 1.10
			},
			"marketWorkshopBias": {
				"finished": 1.30
			},
			"marketConversionShare": {
				"finished": 0.72
			},
			"marketDominancePenalty": {
				"processed": 0.20
			},
			"marketInputReservationByInput": {
				"iron_ore": 1.0
			},
			"marketInputReservationStrength": 0.44,
			"marketInputReservationCap": 0.52,
			"rawCatchmentSpecializationScale": 0.90,
			"rawPotentialPivot": 0.28
		},
		"goods": [
			{
				"name": "grain",
				"category": "raw",
				"baseValue": 0.4,
				"bulkiness": 0.8,
				"perishability": 0.3,
				"sourceWeights": {"crop": 1.0}
			}
		]
	}`))
	if err != nil {
		t.Fatalf("expected settings to load: %v", err)
	}
	if settings.Production.CategorySupplyScale["raw"] != 1.25 {
		t.Fatalf("expected production raw supply override, got %+v", settings.Production.CategorySupplyScale)
	}
	if settings.Production.RawPotentialPivot != 0.28 {
		t.Fatalf("expected production raw pivot override, got %+v", settings.Production)
	}
	if settings.Production.ManufacturingWorkshopScale["finished"] != 1.10 {
		t.Fatalf("expected production manufacturing workshop override, got %+v", settings.Production)
	}
	if settings.Production.MarketWorkshopBias["finished"] != 1.30 {
		t.Fatalf("expected production market workshop bias override, got %+v", settings.Production)
	}
	if settings.Production.MarketConversionShare["finished"] != 0.72 {
		t.Fatalf("expected production market conversion share override, got %+v", settings.Production)
	}
	if settings.Production.MarketDominancePenalty["processed"] != 0.20 {
		t.Fatalf("expected production market dominance penalty override, got %+v", settings.Production)
	}
	if settings.Production.MarketInputReservationStrength != 0.44 || settings.Production.MarketInputReservationCap != 0.52 {
		t.Fatalf("expected production market input reservation overrides, got %+v", settings.Production)
	}
	if settings.Production.MarketInputReservationByInput["iron_ore"] != 1.0 {
		t.Fatalf("expected production market input reservation-by-input override, got %+v", settings.Production)
	}
	if settings.Production.RawCatchmentSpecializationScale != 0.90 {
		t.Fatalf("expected production raw catchment specialization override, got %+v", settings.Production)
	}
}

func TestLoadTradeGoodsSettingsDataAppliesDemandOverrides(t *testing.T) {
	settings, err := loadTradeGoodsSettingsData([]byte(`{
		"schemaVersion": "trade-goods/v1",
		"demand": {
			"goodDemandScale": {
				"fine_clothing": 1.25
			},
			"localSupplyReliefByCategory": {
				"raw": 0.65
			},
			"localSupplyReliefByGood": {
				"fine_clothing": 0.0
			},
			"marketGoodDemandScale": {
				"fine_clothing": 1.35
			},
			"marketWealthPullScale": {
				"finished": 0.40
			},
			"driverSpecializationPivot": 0.35
		},
		"goods": [
			{
				"name": "grain",
				"category": "raw",
				"baseValue": 0.4,
				"bulkiness": 0.8,
				"perishability": 0.3,
				"sourceWeights": {"crop": 1.0}
			}
		]
	}`))
	if err != nil {
		t.Fatalf("expected settings to load: %v", err)
	}
	if settings.Demand.LocalSupplyReliefByCategory["raw"] != 0.65 {
		t.Fatalf("expected demand local supply relief override, got %+v", settings.Demand)
	}
	if settings.Demand.GoodDemandScale["fine_clothing"] != 1.25 {
		t.Fatalf("expected demand good demand scale override, got %+v", settings.Demand)
	}
	if settings.Demand.LocalSupplyReliefByGood["fine_clothing"] != 0.0 {
		t.Fatalf("expected demand local supply relief by good override, got %+v", settings.Demand)
	}
	if settings.Demand.MarketGoodDemandScale["fine_clothing"] != 1.35 {
		t.Fatalf("expected demand market good demand scale override, got %+v", settings.Demand)
	}
	if settings.Demand.DriverSpecializationPivot != 0.35 {
		t.Fatalf("expected demand specialization pivot override, got %+v", settings.Demand)
	}
	if settings.Demand.MarketWealthPullScale["finished"] != 0.40 {
		t.Fatalf("expected demand market wealth pull override, got %+v", settings.Demand)
	}
}

func TestLoadTradeGoodsSettingsDataAppliesGoodMarketConversionScale(t *testing.T) {
	settings, err := loadTradeGoodsSettingsData([]byte(`{
		"schemaVersion": "trade-goods/v1",
		"goods": [
			{
				"name": "cloth",
				"category": "processed",
				"baseValue": 0.62,
				"bulkiness": 0.38,
				"perishability": 0.06,
				"rawCatchmentSensitivity": 1.10,
				"marketConversionScale": 0.35,
				"marketContextSensitivity": 1.25,
				"marketDominancePenalty": 0.40,
				"marketInputReservePriority": 0.80,
				"inputs": {"fiber": 0.45}
			}
		]
	}`))
	if err != nil {
		t.Fatalf("expected settings to load: %v", err)
	}
	if settings.Goods[0].MarketConversionScale != 0.35 {
		t.Fatalf("expected good market conversion scale override, got %+v", settings.Goods[0])
	}
	if settings.Goods[0].MarketContextSensitivity != 1.25 {
		t.Fatalf("expected good market context sensitivity override, got %+v", settings.Goods[0])
	}
	if settings.Goods[0].MarketDominancePenalty != 0.40 {
		t.Fatalf("expected good market dominance penalty override, got %+v", settings.Goods[0])
	}
	if settings.Goods[0].MarketInputReservePriority != 0.80 {
		t.Fatalf("expected good market input reserve priority override, got %+v", settings.Goods[0])
	}
	if settings.Goods[0].RawCatchmentSensitivity != 1.10 {
		t.Fatalf("expected good raw catchment sensitivity override, got %+v", settings.Goods[0])
	}
}

func TestValidateTradeGoodsSettingsRejectsNegativeProductionDriverWeight(t *testing.T) {
	err := ValidateTradeGoodsSettings(TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{
				Name:              "fish",
				Category:          "raw",
				BaseValue:         0.4,
				Bulkiness:         0.6,
				Perishability:     0.7,
				SourceWeights:     map[string]float64{"fish": 1.0},
				ProductionDrivers: map[string]float64{"coastal": -0.2},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected invalid production driver validation error")
	}
}
