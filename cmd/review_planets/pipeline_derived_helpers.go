package main

import (
	"fmt"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

type derivedReviewSettings struct {
	Resource    climgen.ResourceAbundanceSettings
	TradeGoods  climgen.TradeGoodsSettings
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

	hydro := hydrologyBiomeInputsFromScaffold(diagnostics.Hydrology.Scaffold)
	biome := computeHydrologyAwareBiomes(climate, elevation, diagnostics.Hydrology.Scaffold)
	soils := computeSoils(cells, climate, biome, elevation, diagnostics.Hydrology.Scaffold)
	vegetation := computeVegetation(cells, climate, biome, elevation, diagnostics.Hydrology.Scaffold, soils)
	agriculture := computeAgriculture(biome, soils, elevation, diagnostics.Hydrology.Scaffold, settings.Agriculture)
	wildlife := computeWildlife(biome, vegetation, soils, elevation, diagnostics.Hydrology.Scaffold, settings.Wildlife)
	waterResources := computeWaterResources(biome, soils, elevation, diagnostics.Hydrology.Scaffold, settings.Water)
	coastalResources := computeCoastalResources(sites, cells, climate, biome, soils, vegetation, elevation, diagnostics.Hydrology.Scaffold, settings.Coastal)
	resources := computeResources(climate, biome, soils, elevation, diagnostics.Hydrology.Scaffold, diagnostics.HotspotChains, settings.Resource)
	tradeGoods := climgen.ComputeTradeGoodEndowments(
		climgen.TradeGoodInputs{
			Biome:       biome,
			Vegetation:  vegetation,
			Soils:       soils,
			Agriculture: agriculture,
			Wildlife:    wildlife,
			Water:       waterResources,
			Coastal:     coastalResources,
			Resources:   resources,
			Elevation:   elevation,
			SeaLevel:    0,
			Hydro:       hydro,
		},
		settings.TradeGoods,
	)
	settlement := computeSettlement(cells, climate, biome, soils, vegetation, waterResources, resources, elevation, diagnostics.Hydrology.Scaffold)
	population := computePopulation(settlement, agriculture, wildlife, waterResources, coastalResources, resources, elevation, settings.Population)

	derived := &cachedDerivedReview{
		Biome:            biome,
		Soils:            soils,
		Vegetation:       vegetation,
		Agriculture:      agriculture,
		Wildlife:         wildlife,
		WaterResources:   waterResources,
		CoastalResources: coastalResources,
		Resources:        resources,
		TradeGoods:       tradeGoods,
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
