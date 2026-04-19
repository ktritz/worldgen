package main

import (
	"fmt"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

type civilizationReviewSettings struct {
	LandRoutes  climgen.LandRouteSettings
	RiverRoutes climgen.RiverRouteSettings
	TradeGoods  climgen.TradeGoodsSettings
	Profiles    *climgen.ProfileCatalog
}

type maritimeReviewSettings struct {
	VesselName string
	Routes     climgen.MaritimeRouteSettings
	Ports      climgen.MaritimePortSettings
	Coastal    climgen.CoastalTradeSettings
	Ocean      climgen.OceanTradeSettings
}

func loadOrGenerateCivilizationReview(
	cacheStore *reviewCacheStore,
	derivedKey string,
	sites []climgen.Vector3D,
	cells []climgen.VoronoiCell,
	climate *climgen.SeasonalClimateResult,
	derived *cachedDerivedReview,
	elevation []float64,
	diagnostics terrain.PlanetGenerationDiagnostics,
	settings civilizationReviewSettings,
) (*cachedCivilizationReview, string) {
	settingsDigest := cacheSettingsDigest(settings)
	cacheKey := civilizationCacheKey(derivedKey, settingsDigest)
	if cacheStore != nil {
		if cached, ok, err := cacheStore.LoadCivilization(cacheKey); err != nil {
			fmt.Printf("  civilization cache load failed, recomputing: %v\n", err)
		} else if ok {
			fmt.Printf("  civilization cache hit: %s\n", cacheKey)
			return cached, cacheKey
		}
	}
	out := &cachedCivilizationReview{}
	if derived == nil || derived.Settlement == nil || derived.Population == nil || derived.Biome == nil || derived.Soils == nil {
		return out, cacheKey
	}
	out.Network = computeSettlementNetwork(sites, cells, derived.Settlement, derived.Population, derived.Biome, derived.Soils, derived.Resources, elevation)
	if out.Network != nil {
		out.Proto = computeProtoCivilizations(cells, out.Network, derived.Settlement, derived.Population, derived.Biome, derived.Soils, elevation)
		out.LandRoutes = computeLandRoutes(
			derived.Settlement,
			derived.Population,
			derived.Biome,
			derived.Vegetation,
			derived.Soils,
			derived.Wildlife,
			derived.WaterResources,
			elevation,
			hydrologyBiomeInputsFromScaffold(diagnostics.Hydrology.Scaffold),
			settings.LandRoutes,
		)
		if out.Proto != nil && out.LandRoutes != nil {
			out.Trade = computeTradeNetwork(cells, out.Network, out.Proto, out.LandRoutes)
			out.RiverRoutes = computeRiverRoutes(
				derived.Settlement,
				derived.Population,
				derived.Soils,
				derived.WaterResources,
				elevation,
				hydrologyBiomeInputsFromScaffold(diagnostics.Hydrology.Scaffold),
				settings.RiverRoutes,
			)
			if out.RiverRoutes != nil {
				out.RiverTrade = computeRiverTrade(cells, out.Network, out.Proto, out.RiverRoutes, elevation)
			}
			out.Polities = computePolitySpheres(cells, out.Network, out.Proto, out.Trade, derived.Population, derived.Settlement, elevation)
			if out.Polities != nil {
				out.Profiles = computePolityProfiles(cells, out.Polities, out.Network, out.Trade, derived.Biome, derived.Vegetation, derived.Soils, diagnostics.Hydrology.Scaffold, settings.Profiles)
				if derived.TradeGoods != nil && out.Profiles != nil {
					out.NodeGoods = climgen.ComputeNodeGoods(cells, derived.TradeGoods, settings.TradeGoods, out.Polities, out.Profiles, out.Network, out.Trade)
					out.PolityGoods = climgen.ComputePolityGoodsWithNodeMarkets(derived.TradeGoods, settings.TradeGoods, out.Polities, out.Profiles, out.Network, out.Trade, out.NodeGoods)
				}
			}
		}
	}
	if cacheStore != nil {
		if err := cacheStore.SaveCivilization(cacheKey, out); err != nil {
			fmt.Printf("  civilization cache save failed: %v\n", err)
		}
	}
	return out, cacheKey
}

func loadOrGenerateMaritimeReview(
	cacheStore *reviewCacheStore,
	civilizationKey string,
	sites []climgen.Vector3D,
	cells []climgen.VoronoiCell,
	climate *climgen.SeasonalClimateResult,
	derived *cachedDerivedReview,
	civilization *cachedCivilizationReview,
	elevation []float64,
	diagnostics terrain.PlanetGenerationDiagnostics,
	settings maritimeReviewSettings,
) (*cachedMaritimeReview, string) {
	settingsDigest := cacheSettingsDigest(settings)
	cacheKey := maritimeCacheKey(civilizationKey, settings.VesselName, settingsDigest)
	if cacheStore != nil {
		if cached, ok, err := cacheStore.LoadMaritime(cacheKey); err != nil {
			fmt.Printf("  maritime cache load failed, recomputing: %v\n", err)
		} else if ok {
			fmt.Printf("  maritime cache hit: %s\n", cacheKey)
			return cached, cacheKey
		}
	}
	out := &cachedMaritimeReview{}
	if civilization == nil || civilization.Network == nil || civilization.Trade == nil || civilization.Proto == nil || derived == nil {
		return out, cacheKey
	}
	out.CoastalPorts = computeCoastalPorts(
		cells,
		climate,
		civilization.Network,
		civilization.Trade,
		civilization.RiverTrade,
		derived.CoastalResources,
		civilization.RiverRoutes,
		derived.Soils,
		elevation,
		diagnostics.Hydrology.Scaffold,
		settings.Routes,
		settings.Ports,
	)
	if out.CoastalPorts != nil {
		out.CoastalTrade = computeCoastalTrade(
			sites,
			cells,
			climate,
			civilization.Network,
			civilization.Proto,
			out.CoastalPorts,
			elevation,
			settings.Coastal,
		)
		out.OceanTrade = computeOceanTrade(
			sites,
			cells,
			climate,
			civilization.Network,
			civilization.Proto,
			out.CoastalPorts,
			elevation,
			settings.Ocean,
		)
	}
	if cacheStore != nil {
		if err := cacheStore.SaveMaritime(cacheKey, out); err != nil {
			fmt.Printf("  maritime cache save failed: %v\n", err)
		}
	}
	return out, cacheKey
}

func loadOrGenerateEconomyReview(
	cacheStore *reviewCacheStore,
	civilizationKey string,
	maritimeKey string,
	cells []climgen.VoronoiCell,
	derived *cachedDerivedReview,
	civilization *cachedCivilizationReview,
	maritime *cachedMaritimeReview,
	settings climgen.TradeGoodsSettings,
) (*cachedEconomyReview, string) {
	cacheKey := economyCacheKey(civilizationKey, maritimeKey, cacheSettingsDigest(settings))
	if cacheStore != nil {
		if cached, ok, err := cacheStore.LoadEconomy(cacheKey); err != nil {
			fmt.Printf("  economy cache load failed, recomputing: %v\n", err)
		} else if ok {
			fmt.Printf("  economy cache hit: %s\n", cacheKey)
			return cached, cacheKey
		}
	}
	out := &cachedEconomyReview{}
	if civilization == nil || derived == nil || derived.TradeGoods == nil || civilization.Polities == nil || civilization.Profiles == nil || civilization.Network == nil || civilization.Trade == nil || civilization.PolityGoods == nil || civilization.NodeGoods == nil {
		return out, cacheKey
	}
	out.NodeMarkets = climgen.ComputeTradeNodeMarketsWithRouteSupport(
		cells,
		derived.TradeGoods,
		settings,
		civilization.Polities,
		civilization.Profiles,
		civilization.Network,
		civilization.Trade,
		civilization.RiverTrade,
		maritime.CoastalTrade,
		maritime.OceanTrade,
		civilization.PolityGoods,
		civilization.NodeGoods,
	)
	out.Multimodal = climgen.ComputeMultimodalTradeWithNodeMarkets(
		civilization.PolityGoods,
		settings,
		civilization.Polities,
		civilization.Network,
		civilization.Trade,
		civilization.RiverTrade,
		maritime.CoastalTrade,
		maritime.OceanTrade,
		out.NodeMarkets,
	)
	if cacheStore != nil {
		if err := cacheStore.SaveEconomy(cacheKey, out); err != nil {
			fmt.Printf("  economy cache save failed: %v\n", err)
		}
	}
	return out, cacheKey
}
