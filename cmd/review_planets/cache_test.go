package main

import (
	"path/filepath"
	"testing"

	"worldgen/climgen"
)

func TestReviewCacheStoreRoundTrip(t *testing.T) {
	store := newReviewCacheStore(t.TempDir())
	terrainKey := terrainCacheKey(5, 12, 0.29, 55)
	climateKey := climateCacheKey(terrainKey, 55)

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
}
