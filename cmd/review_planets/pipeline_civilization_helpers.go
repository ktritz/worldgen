package main

import (
	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

func computeSettlement(
	cells []climgen.VoronoiCell,
	climate *climgen.SeasonalClimateResult,
	biomes *climgen.BiomeResult,
	soils *climgen.SoilResult,
	vegetation *climgen.VegetationResult,
	waterResources *climgen.WaterResourceResult,
	resources *climgen.ResourceResult,
	elevation []float64,
	scaffold *terrain.HydrologyScaffold,
) *climgen.SettlementResult {
	hydro := hydrologyBiomeInputsFromScaffold(scaffold)
	coastalExposure := climgen.ComputeCoastalExposure(cells, elevation, 0.0)
	return climgen.ClassifySettlementSuitability(climate, biomes, soils, vegetation, waterResources, resources, elevation, 0.0, hydro, coastalExposure)
}

func computePopulation(
	settlements *climgen.SettlementResult,
	agriculture *climgen.AgricultureResult,
	wildlife *climgen.WildlifeResult,
	waterResources *climgen.WaterResourceResult,
	coastalResources *climgen.CoastalResourceResult,
	resources *climgen.ResourceResult,
	elevation []float64,
	settings climgen.PopulationSupportSettings,
) *climgen.PopulationResult {
	return climgen.ClassifyPopulationSupport(
		settlements,
		agriculture,
		wildlife,
		waterResources,
		coastalResources,
		resources,
		elevation,
		0.0,
		settings,
	)
}

func computeSettlementNetwork(
	sites []climgen.Vector3D,
	cells []climgen.VoronoiCell,
	settlements *climgen.SettlementResult,
	population *climgen.PopulationResult,
	biomes *climgen.BiomeResult,
	soils *climgen.SoilResult,
	resources *climgen.ResourceResult,
	elevation []float64,
) *climgen.SettlementNetworkResult {
	return climgen.BuildSettlementNetwork(
		sites,
		cells,
		settlements,
		population,
		biomes,
		soils,
		resources,
		elevation,
		0.0,
		climgen.DefaultSettlementNetworkSettings(),
	)
}

func computeProtoCivilizations(
	cells []climgen.VoronoiCell,
	network *climgen.SettlementNetworkResult,
	settlements *climgen.SettlementResult,
	population *climgen.PopulationResult,
	biomes *climgen.BiomeResult,
	soils *climgen.SoilResult,
	elevation []float64,
) *climgen.ProtoCivilizationResult {
	return climgen.BuildProtoCivilizations(
		cells,
		network,
		settlements,
		population,
		biomes,
		soils,
		elevation,
		0.0,
		climgen.DefaultProtoCivilizationSettings(),
	)
}

func computeTradeNetwork(
	cells []climgen.VoronoiCell,
	network *climgen.SettlementNetworkResult,
	proto *climgen.ProtoCivilizationResult,
	landRoutes *climgen.LandRouteResult,
) *climgen.TradeNetworkResult {
	return climgen.BuildTradeNetwork(cells, network, proto, landRoutes, climgen.DefaultTradeNetworkSettings())
}

func computeRiverRoutes(
	settlements *climgen.SettlementResult,
	population *climgen.PopulationResult,
	soils *climgen.SoilResult,
	waterResources *climgen.WaterResourceResult,
	elevation []float64,
	hydro *climgen.HydrologyBiomeInputs,
	settings climgen.RiverRouteSettings,
) *climgen.RiverRouteResult {
	return climgen.BuildRiverRouteDiagnostics(
		settlements,
		population,
		soils,
		waterResources,
		elevation,
		0.0,
		hydro,
		settings,
	)
}

func computeRiverTrade(
	cells []climgen.VoronoiCell,
	network *climgen.SettlementNetworkResult,
	proto *climgen.ProtoCivilizationResult,
	riverRoutes *climgen.RiverRouteResult,
	elevation []float64,
) *climgen.RiverTradeResult {
	return climgen.BuildRiverTradeNetwork(cells, network, proto, riverRoutes, elevation, climgen.DefaultRiverTradeSettings())
}

func computeCoastalPorts(
	cells []climgen.VoronoiCell,
	climate *climgen.SeasonalClimateResult,
	network *climgen.SettlementNetworkResult,
	trade *climgen.TradeNetworkResult,
	riverTrade *climgen.RiverTradeResult,
	coastalResources *climgen.CoastalResourceResult,
	riverRoutes *climgen.RiverRouteResult,
	soils *climgen.SoilResult,
	elevation []float64,
	scaffold *terrain.HydrologyScaffold,
	maritimeRoutes climgen.MaritimeRouteSettings,
	settings climgen.MaritimePortSettings,
) *climgen.CoastalPortResult {
	hydro := hydrologyBiomeInputsFromScaffold(scaffold)
	return climgen.BuildCoastalPorts(
		cells,
		climate,
		network,
		trade,
		riverTrade,
		coastalResources,
		riverRoutes,
		soils,
		elevation,
		0.0,
		hydro,
		maritimeRoutes,
		settings,
	)
}

func computeCoastalTrade(
	sites []climgen.Vector3D,
	cells []climgen.VoronoiCell,
	climate *climgen.SeasonalClimateResult,
	network *climgen.SettlementNetworkResult,
	proto *climgen.ProtoCivilizationResult,
	ports *climgen.CoastalPortResult,
	elevation []float64,
	settings climgen.CoastalTradeSettings,
) *climgen.CoastalTradeResult {
	return climgen.BuildCoastalTradeNetwork(sites, cells, climate, network, proto, ports, elevation, 0.0, settings)
}

func computeOceanTrade(
	sites []climgen.Vector3D,
	cells []climgen.VoronoiCell,
	climate *climgen.SeasonalClimateResult,
	network *climgen.SettlementNetworkResult,
	proto *climgen.ProtoCivilizationResult,
	ports *climgen.CoastalPortResult,
	elevation []float64,
	settings climgen.OceanTradeSettings,
) *climgen.OceanTradeResult {
	return climgen.BuildOceanTradeNetwork(sites, cells, climate, network, proto, ports, elevation, 0.0, settings)
}

func computeLandRoutes(
	settlements *climgen.SettlementResult,
	population *climgen.PopulationResult,
	biomes *climgen.BiomeResult,
	vegetation *climgen.VegetationResult,
	soils *climgen.SoilResult,
	wildlife *climgen.WildlifeResult,
	waterResources *climgen.WaterResourceResult,
	elevation []float64,
	hydro *climgen.HydrologyBiomeInputs,
	settings climgen.LandRouteSettings,
) *climgen.LandRouteResult {
	return climgen.BuildLandRouteDiagnostics(
		settlements,
		population,
		biomes,
		vegetation,
		soils,
		wildlife,
		waterResources,
		elevation,
		0.0,
		hydro,
		settings,
	)
}

func computePolitySpheres(
	cells []climgen.VoronoiCell,
	network *climgen.SettlementNetworkResult,
	proto *climgen.ProtoCivilizationResult,
	trade *climgen.TradeNetworkResult,
	population *climgen.PopulationResult,
	settlements *climgen.SettlementResult,
	elevation []float64,
) *climgen.PolitySphereResult {
	return climgen.BuildPolitySpheres(
		cells,
		network,
		proto,
		trade,
		population,
		settlements,
		elevation,
		0.0,
		climgen.DefaultPolitySphereSettings(),
	)
}

func computePolityProfiles(
	cells []climgen.VoronoiCell,
	polities *climgen.PolitySphereResult,
	network *climgen.SettlementNetworkResult,
	trade *climgen.TradeNetworkResult,
	biomes *climgen.BiomeResult,
	vegetation *climgen.VegetationResult,
	soils *climgen.SoilResult,
	scaffold *terrain.HydrologyScaffold,
	catalog *climgen.ProfileCatalog,
) *climgen.PolityProfileResult {
	hydro := hydrologyBiomeInputsFromScaffold(scaffold)
	return climgen.BuildPolityProfiles(cells, polities, network, trade, biomes, vegetation, soils, hydro, catalog)
}

func computeSettlementPreferences(
	biomes *climgen.BiomeResult,
	soils *climgen.SoilResult,
	vegetation *climgen.VegetationResult,
	settlements *climgen.SettlementResult,
	elevation []float64,
	profiles []climgen.SettlementPreferenceProfile,
) []*climgen.SettlementPreferenceResult {
	results := make([]*climgen.SettlementPreferenceResult, 0, len(profiles))
	for _, profile := range profiles {
		result := climgen.ClassifySettlementPreference(settlements, biomes, soils, vegetation, elevation, 0.0, profile)
		results = append(results, result)
	}
	return results
}
