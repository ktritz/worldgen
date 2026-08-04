package main

import (
	"fmt"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

type civilizationReviewSettings struct {
	SettlementNetwork climgen.SettlementNetworkSettings
	ProtoCivilization climgen.ProtoCivilizationSettings
	LandRoutes        climgen.LandRouteSettings
	TradeNetwork      climgen.TradeNetworkSettings
	RiverRoutes       climgen.RiverRouteSettings
	RiverTrade        climgen.RiverTradeSettings
	PolitySpheres     climgen.PolitySphereSettings
	Profiles          *climgen.ProfileCatalog
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
	phaseStart := reviewPhaseStart("civilization")
	cacheStatus := "miss"
	if cacheStore != nil {
		if cached, ok, err := cacheStore.LoadCivilization(cacheKey); err != nil {
			fmt.Printf("  civilization cache load failed, recomputing: %v\n", err)
		} else if ok {
			fmt.Printf("  civilization cache hit: %s\n", cacheKey)
			reviewPhaseDone("civilization", "hit", phaseStart, fmt.Sprintf("key=%s", cacheKey))
			return cached, cacheKey
		}
	}
	out := &cachedCivilizationReview{}
	if derived == nil || derived.Settlement == nil || derived.Population == nil || derived.Biome == nil || derived.Soils == nil {
		return out, cacheKey
	}
	hydro := resolutionAdjustedHydrologyBiomeInputsFromScaffold(cells, elevation, diagnostics.Hydrology.Scaffold)
	out.Network = computeSettlementNetwork(sites, cells, derived.Settlement, derived.Population, derived.Biome, derived.Soils, derived.Resources, elevation, settings.SettlementNetwork)
	if out.Network != nil {
		out.Proto = computeProtoCivilizations(cells, out.Network, derived.Settlement, derived.Population, derived.Biome, derived.Soils, elevation, settings.ProtoCivilization)
		out.LandRoutes = computeLandRoutes(
			derived.Settlement,
			derived.Population,
			derived.Biome,
			derived.Vegetation,
			derived.Soils,
			derived.Wildlife,
			derived.WaterResources,
			elevation,
			hydro,
			settings.LandRoutes,
		)
		if out.Proto != nil && out.LandRoutes != nil {
			out.Trade = computeTradeNetwork(cells, out.Network, out.Proto, out.LandRoutes, settings.TradeNetwork)
			out.RiverRoutes = computeRiverRoutes(
				derived.Settlement,
				derived.Population,
				derived.Soils,
				derived.WaterResources,
				elevation,
				hydro,
				settings.RiverRoutes,
			)
			if out.RiverRoutes != nil {
				out.RiverTrade = computeRiverTrade(cells, out.Network, out.Proto, out.RiverRoutes, elevation, settings.RiverTrade)
			}
			out.Polities = computePolitySpheres(cells, out.Network, out.Proto, out.Trade, derived.Population, derived.Settlement, elevation, settings.PolitySpheres)
			if out.Polities != nil {
				out.Profiles = computePolityProfiles(cells, out.Polities, out.Network, out.Trade, derived.Biome, derived.Vegetation, derived.Soils, hydro, settings.Profiles)
			}
		}
	}
	if cacheStore != nil {
		if err := cacheStore.SaveCivilization(cacheKey, out); err != nil {
			fmt.Printf("  civilization cache save failed: %v\n", err)
		}
	}
	reviewPhaseDone("civilization", cacheStatus, phaseStart, fmt.Sprintf("key=%s", cacheKey))
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
	phaseStart := reviewPhaseStart("maritime")
	cacheStatus := "miss"
	if cacheStore != nil {
		if cached, ok, err := cacheStore.LoadMaritime(cacheKey); err != nil {
			fmt.Printf("  maritime cache load failed, recomputing: %v\n", err)
		} else if ok {
			fmt.Printf("  maritime cache hit: %s\n", cacheKey)
			reviewPhaseDone("maritime", "hit", phaseStart, fmt.Sprintf("key=%s", cacheKey))
			return cached, cacheKey
		}
	}
	out := &cachedMaritimeReview{}
	if civilization == nil || civilization.Network == nil || civilization.Trade == nil || civilization.Proto == nil || derived == nil {
		return out, cacheKey
	}
	hydro := resolutionAdjustedHydrologyBiomeInputsFromScaffold(cells, elevation, diagnostics.Hydrology.Scaffold)
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
		hydro,
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
	reviewPhaseDone("maritime", cacheStatus, phaseStart, fmt.Sprintf("key=%s", cacheKey), fmt.Sprintf("vessel=%s", settings.VesselName))
	return out, cacheKey
}

func loadOrGenerateEconomyReview(
	cacheStore *reviewCacheStore,
	civilizationKey string,
	maritimeKey string,
	cells []climgen.VoronoiCell,
	tradeGoods *climgen.TradeGoodResult,
	civilization *cachedCivilizationReview,
	maritime *cachedMaritimeReview,
	settings climgen.TradeGoodsSettings,
) (*cachedEconomyReview, string) {
	cacheKey := economyCacheKey(civilizationKey, maritimeKey, cacheSettingsDigest(settings))
	phaseStart := reviewPhaseStart("economy")
	cacheStatus := "miss"
	if cacheStore != nil {
		if cached, ok, err := cacheStore.LoadEconomy(cacheKey); err != nil {
			fmt.Printf("  economy cache load failed, recomputing: %v\n", err)
		} else if ok {
			fmt.Printf("  economy cache hit: %s\n", cacheKey)
			reviewPhaseDone("economy", "hit", phaseStart, fmt.Sprintf("key=%s", cacheKey))
			return cached, cacheKey
		}
	}
	out := &cachedEconomyReview{}
	if civilization == nil || tradeGoods == nil || civilization.Polities == nil || civilization.Profiles == nil || civilization.Network == nil || civilization.Trade == nil {
		return out, cacheKey
	}
	out.NodeGoods = climgen.ComputeNodeGoods(cells, tradeGoods, settings, civilization.Polities, civilization.Profiles, civilization.Network, civilization.Trade)
	out.PolityGoods = climgen.ComputePolityGoodsWithNodeMarkets(tradeGoods, settings, civilization.Polities, civilization.Profiles, civilization.Network, civilization.Trade, out.NodeGoods)
	var coastalTrade *climgen.CoastalTradeResult
	var oceanTrade *climgen.OceanTradeResult
	if maritime != nil {
		coastalTrade = maritime.CoastalTrade
		oceanTrade = maritime.OceanTrade
	}
	out.NodeMarkets = climgen.ComputeTradeNodeMarketsWithRouteSupport(
		cells,
		tradeGoods,
		settings,
		civilization.Polities,
		civilization.Profiles,
		civilization.Network,
		civilization.Trade,
		civilization.RiverTrade,
		coastalTrade,
		oceanTrade,
		out.PolityGoods,
		out.NodeGoods,
	)
	out.Multimodal = climgen.ComputeMultimodalTradeWithNodeMarkets(
		out.PolityGoods,
		settings,
		civilization.Polities,
		civilization.Network,
		civilization.Trade,
		civilization.RiverTrade,
		coastalTrade,
		oceanTrade,
		out.NodeMarkets,
	)
	if cacheStore != nil {
		if err := cacheStore.SaveEconomy(cacheKey, out); err != nil {
			fmt.Printf("  economy cache save failed: %v\n", err)
		}
	}
	reviewPhaseDone("economy", cacheStatus, phaseStart, fmt.Sprintf("key=%s", cacheKey))
	return out, cacheKey
}
