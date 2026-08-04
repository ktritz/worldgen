package main

import (
	"fmt"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

type derivedReviewSettings struct {
	Resource    climgen.ResourceAbundanceSettings
	Agriculture climgen.AgricultureProductivitySettings
	Wildlife    climgen.WildlifeProductivitySettings
	Coastal     climgen.CoastalResourceSettings
	Water       climgen.WaterResourceSettings
	Population  climgen.PopulationSupportSettings
}

func loadOrGenerateDerivedReview(
	cacheStore *reviewCacheStore,
	terrainKey string,
	climateKey string,
	climateHydrology bool,
	sites []climgen.Vector3D,
	cells []climgen.VoronoiCell,
	climate *climgen.SeasonalClimateResult,
	elevation []float64,
	diagnostics terrain.PlanetGenerationDiagnostics,
	settings derivedReviewSettings,
) *cachedDerivedReview {
	settingsDigest := cacheSettingsDigest(settings)
	derivedKey := derivedCacheKey(terrainKey, climateKey, climateHydrology, settingsDigest)
	phaseStart := reviewPhaseStart("derived")
	cacheStatus := "miss"
	if cacheStore != nil {
		if cached, ok, err := cacheStore.LoadDerived(derivedKey); err != nil {
			fmt.Printf("  derived cache load failed, recomputing: %v\n", err)
		} else if ok {
			fmt.Printf("  derived cache hit: %s\n", derivedKey)
			reviewPhaseDone("derived", "hit", phaseStart, fmt.Sprintf("key=%s", derivedKey))
			return cached
		}
	}

	hydro := resolutionAdjustedHydrologyBiomeInputsFromScaffold(cells, elevation, diagnostics.Hydrology.Scaffold)
	coastalExposure := climgen.ComputeCoastalExposure(cells, elevation, 0.0)
	biome := computeHydrologyAwareBiomes(climate, elevation, hydro)
	soils := computeSoils(cells, climate, biome, elevation, hydro, coastalExposure)
	vegetation := computeVegetation(cells, climate, biome, elevation, hydro, coastalExposure, soils)
	agriculture := computeAgriculture(biome, soils, elevation, hydro, settings.Agriculture)
	wildlife := computeWildlife(biome, vegetation, soils, elevation, hydro, settings.Wildlife)
	waterResources := computeWaterResources(biome, soils, elevation, hydro, settings.Water)
	coastalResources := computeCoastalResources(sites, cells, climate, biome, soils, vegetation, elevation, hydro, coastalExposure, settings.Coastal)
	resources := computeResources(climate, biome, soils, elevation, hydro, diagnostics.HotspotChains, settings.Resource)
	settlement := computeSettlement(cells, climate, biome, soils, vegetation, waterResources, resources, elevation, hydro, coastalExposure)
	population := computePopulation(cells, settlement, agriculture, wildlife, waterResources, coastalResources, resources, elevation, settings.Population)

	derived := &cachedDerivedReview{
		Biome:            biome,
		Soils:            soils,
		Vegetation:       vegetation,
		Agriculture:      agriculture,
		Wildlife:         wildlife,
		WaterResources:   waterResources,
		CoastalResources: coastalResources,
		Resources:        resources,
		Settlement:       settlement,
		Population:       population,
	}
	if cacheStore != nil {
		if err := cacheStore.SaveDerived(derivedKey, derived); err != nil {
			fmt.Printf("  derived cache save failed: %v\n", err)
		}
	}
	reviewPhaseDone("derived", cacheStatus, phaseStart, fmt.Sprintf("key=%s", derivedKey))
	return derived
}

func computeTradeGoodsForReview(
	derived *cachedDerivedReview,
	cells []climgen.VoronoiCell,
	elevation []float64,
	diagnostics terrain.PlanetGenerationDiagnostics,
	settings climgen.TradeGoodsSettings,
) *climgen.TradeGoodResult {
	if derived == nil || derived.Biome == nil || derived.Vegetation == nil || derived.Soils == nil || derived.Agriculture == nil || derived.Wildlife == nil || derived.WaterResources == nil || derived.CoastalResources == nil || derived.Resources == nil {
		return nil
	}
	return climgen.ComputeTradeGoodEndowments(
		climgen.TradeGoodInputs{
			Biome:       derived.Biome,
			Vegetation:  derived.Vegetation,
			Soils:       derived.Soils,
			Agriculture: derived.Agriculture,
			Wildlife:    derived.Wildlife,
			Water:       derived.WaterResources,
			Coastal:     derived.CoastalResources,
			Resources:   derived.Resources,
			Elevation:   elevation,
			SeaLevel:    0,
			Hydro:       resolutionAdjustedHydrologyBiomeInputsFromScaffold(cells, elevation, diagnostics.Hydrology.Scaffold),
		},
		settings,
	)
}

func loadOrGenerateTradeGoodsReview(
	cacheStore *reviewCacheStore,
	derivedKey string,
	derived *cachedDerivedReview,
	cells []climgen.VoronoiCell,
	elevation []float64,
	diagnostics terrain.PlanetGenerationDiagnostics,
	settings climgen.TradeGoodsSettings,
) (*climgen.TradeGoodResult, string) {
	cacheKey := tradeGoodsCacheKey(derivedKey, cacheSettingsDigest(settings))
	phaseStart := reviewPhaseStart("trade_goods")
	cacheStatus := "miss"
	if cacheStore != nil {
		if cached, ok, err := cacheStore.LoadTradeGoods(cacheKey); err != nil {
			fmt.Printf("  trade goods cache load failed, recomputing: %v\n", err)
		} else if ok {
			fmt.Printf("  trade goods cache hit: %s\n", cacheKey)
			reviewPhaseDone("trade_goods", "hit", phaseStart, fmt.Sprintf("key=%s", cacheKey))
			return cached, cacheKey
		}
	}
	out := computeTradeGoodsForReview(derived, cells, elevation, diagnostics, settings)
	if cacheStore != nil {
		if err := cacheStore.SaveTradeGoods(cacheKey, out); err != nil {
			fmt.Printf("  trade goods cache save failed: %v\n", err)
		}
	}
	reviewPhaseDone("trade_goods", cacheStatus, phaseStart, fmt.Sprintf("key=%s", cacheKey))
	return out, cacheKey
}
