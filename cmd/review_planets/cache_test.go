package main

import (
	"math"
	"path/filepath"
	"testing"

	"worldgen/climgen"
)

func TestReviewCacheStoreRoundTrip(t *testing.T) {
	store := newReviewCacheStore(t.TempDir())
	terrainKey := terrainCacheKey(5, 12, 0.29, 55)
	climateKey := climateCacheKey(terrainKey, 55)
	derivedKey := derivedCacheKey(terrainKey, climateKey, true, cacheSettingsDigest("settings-v1"))
	civilizationKey := civilizationCacheKey(derivedKey, cacheSettingsDigest("civilization-v1"))
	maritimeKey := maritimeCacheKey(civilizationKey, "coastal-sloop", cacheSettingsDigest("maritime-v1"))
	economyKey := economyCacheKey(civilizationKey, maritimeKey)

	terrainValue := &cachedTerrainReview{
		Elevation: []float64{1, 2, 3},
		IsLand:    []bool{true, false, true},
	}
	if err := store.SaveTerrain(terrainKey, terrainValue); err != nil {
		t.Fatalf("save terrain: %v", err)
	}
	loadedTerrain, ok, err := store.LoadTerrain(terrainKey)
	if err != nil || !ok {
		t.Fatalf("load terrain ok=%v err=%v", ok, err)
	}
	if len(loadedTerrain.Elevation) != 3 || !loadedTerrain.IsLand[0] || loadedTerrain.IsLand[1] {
		t.Fatalf("unexpected terrain payload: %+v", loadedTerrain)
	}

	climateValue := &climgen.SeasonalClimateResult{
		AnnualPrecipitation: []float64{12, 34},
		WettestSeason:       []int{1, 2},
	}
	if err := store.SaveClimate(climateKey, climateValue); err != nil {
		t.Fatalf("save climate: %v", err)
	}
	loadedClimate, ok, err := store.LoadClimate(climateKey)
	if err != nil || !ok {
		t.Fatalf("load climate ok=%v err=%v", ok, err)
	}
	if len(loadedClimate.AnnualPrecipitation) != 2 || loadedClimate.AnnualPrecipitation[1] != 34 {
		t.Fatalf("unexpected climate payload: %+v", loadedClimate)
	}

	if filepath.Ext(store.terrainPath(terrainKey)) != ".json" || filepath.Ext(store.climatePath(climateKey)) != ".json" {
		t.Fatalf("expected json cache paths")
	}
	if filepath.Ext(store.derivedPath(derivedKey)) != ".json" {
		t.Fatalf("expected derived cache path")
	}
	if filepath.Ext(store.civilizationPath(civilizationKey)) != ".json" || filepath.Ext(store.maritimePath(maritimeKey)) != ".json" || filepath.Ext(store.economyPath(economyKey)) != ".json" {
		t.Fatalf("expected downstream cache paths")
	}

	derivedValue := &cachedDerivedReview{
		Biome: &climgen.BiomeResult{Biomes: []climgen.Biome{climgen.BiomeSavanna, climgen.BiomeTemperateForest}},
		TradeGoods: &climgen.TradeGoodResult{
			Goods: []climgen.TradeGoodEndowment{{Good: "grain", Category: "raw", Potential: []float64{0.3, 0.5}}},
		},
		Population: &climgen.PopulationResult{Classes: []climgen.PopulationClass{climgen.PopulationRural, climgen.PopulationUrban}},
	}
	if err := store.SaveDerived(derivedKey, derivedValue); err != nil {
		t.Fatalf("save derived: %v", err)
	}
	loadedDerived, ok, err := store.LoadDerived(derivedKey)
	if err != nil || !ok {
		t.Fatalf("load derived ok=%v err=%v", ok, err)
	}
	if len(loadedDerived.Biome.Biomes) != 2 || loadedDerived.TradeGoods.Goods[0].Good != "grain" || loadedDerived.Population.Classes[1] != climgen.PopulationUrban {
		t.Fatalf("unexpected derived payload: %+v", loadedDerived)
	}

	civilizationValue := &cachedCivilizationReview{
		Network:   &climgen.SettlementNetworkResult{Regions: []climgen.SettlementRegion{{ID: 1, CenterNode: 3}}},
		NodeGoods: &climgen.NodeGoodsResult{Balances: []climgen.NodeGoodBalance{{NodeID: 2}}},
	}
	if err := store.SaveCivilization(civilizationKey, civilizationValue); err != nil {
		t.Fatalf("save civilization: %v", err)
	}
	loadedCivilization, ok, err := store.LoadCivilization(civilizationKey)
	if err != nil || !ok {
		t.Fatalf("load civilization ok=%v err=%v", ok, err)
	}
	if len(loadedCivilization.Network.Regions) != 1 || loadedCivilization.Network.Regions[0].CenterNode != 3 || loadedCivilization.NodeGoods.Balances[0].NodeID != 2 {
		t.Fatalf("unexpected civilization payload: %+v", loadedCivilization)
	}

	maritimeValue := &cachedMaritimeReview{
		CoastalTrade: &climgen.CoastalTradeResult{Corridors: []climgen.CoastalTradeCorridor{{Flow: 0.5}}},
	}
	if err := store.SaveMaritime(maritimeKey, maritimeValue); err != nil {
		t.Fatalf("save maritime: %v", err)
	}
	loadedMaritime, ok, err := store.LoadMaritime(maritimeKey)
	if err != nil || !ok {
		t.Fatalf("load maritime ok=%v err=%v", ok, err)
	}
	if len(loadedMaritime.CoastalTrade.Corridors) != 1 || loadedMaritime.CoastalTrade.Corridors[0].Flow != 0.5 {
		t.Fatalf("unexpected maritime payload: %+v", loadedMaritime)
	}

	economyValue := &cachedEconomyReview{
		Multimodal: &climgen.MultimodalTradeResult{Diagnostics: climgen.MultimodalTradeDiagnostics{TotalScore: 4.2}},
	}
	if err := store.SaveEconomy(economyKey, economyValue); err != nil {
		t.Fatalf("save economy: %v", err)
	}
	loadedEconomy, ok, err := store.LoadEconomy(economyKey)
	if err != nil || !ok {
		t.Fatalf("load economy ok=%v err=%v", ok, err)
	}
	if loadedEconomy.Multimodal.Diagnostics.TotalScore != 4.2 {
		t.Fatalf("unexpected economy payload: %+v", loadedEconomy)
	}
}

func TestReviewCacheStoreSanitizesNonFiniteFloats(t *testing.T) {
	store := newReviewCacheStore(t.TempDir())
	key := civilizationCacheKey("derived-key", cacheSettingsDigest("civilization-v1"))
	value := &cachedCivilizationReview{
		LandRoutes: &climgen.LandRouteResult{
			Diagnostics: &climgen.LandRouteDiagnostics{
				BaseCost: []float64{math.Inf(1), 1.5},
				ModeCost: []float64{math.Inf(1), 2.0},
			},
		},
	}
	if err := store.SaveCivilization(key, value); err != nil {
		t.Fatalf("save civilization with inf: %v", err)
	}
	loaded, ok, err := store.LoadCivilization(key)
	if err != nil || !ok {
		t.Fatalf("load civilization ok=%v err=%v", ok, err)
	}
	if math.IsInf(loaded.LandRoutes.Diagnostics.BaseCost[0], 0) || math.IsNaN(loaded.LandRoutes.Diagnostics.BaseCost[0]) || loaded.LandRoutes.Diagnostics.BaseCost[0] <= 1e8 {
		t.Fatalf("expected sanitized large finite value, got %v", loaded.LandRoutes.Diagnostics.BaseCost[0])
	}
}
