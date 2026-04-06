package main

import (
	"fmt"
	"strconv"
	"strings"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

func parseSeeds(raw string) ([]int64, error) {
	parts := strings.Split(raw, ",")
	seeds := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", part, err)
		}
		seeds = append(seeds, value)
	}
	if len(seeds) == 0 {
		return nil, fmt.Errorf("no seeds provided")
	}
	return seeds, nil
}

func loadSettlementProfiles(enabled bool, path string) []climgen.SettlementPreferenceProfile {
	profiles := climgen.DefaultFantasySettlementProfiles()
	if !enabled {
		return profiles
	}
	if catalog, err := climgen.LoadProfileCatalog(path); err != nil {
		fmt.Printf("Using built-in profile catalog, failed to load %s: %v\n", path, err)
		return profiles
	} else {
		profiles = climgen.ExtractComposedSettlementProfiles(catalog)
		if len(profiles) == 0 {
			profiles = climgen.ExtractSettlementProfiles(catalog)
			fmt.Printf("Loaded %d ancestry settlement profiles from %s\n", len(profiles), path)
		} else {
			fmt.Printf("Loaded %d composed settlement profiles from %s\n", len(profiles), path)
		}
	}
	return profiles
}

func loadProfileCatalogWithFallback(path string) *climgen.ProfileCatalog {
	catalog, err := climgen.LoadProfileCatalog(path)
	if err != nil {
		fmt.Printf("Using built-in profile catalog, failed to load %s: %v\n", path, err)
		return climgen.DefaultFantasyProfileCatalog()
	}
	fmt.Printf("Loaded profile catalog from %s\n", path)
	return catalog
}

func extractSettlementProfilesFromCatalog(enabled bool, catalog *climgen.ProfileCatalog) []climgen.SettlementPreferenceProfile {
	profiles := climgen.DefaultFantasySettlementProfiles()
	if !enabled || catalog == nil {
		return profiles
	}
	profiles = climgen.ExtractComposedSettlementProfiles(catalog)
	if len(profiles) == 0 {
		profiles = climgen.ExtractSettlementProfiles(catalog)
	}
	return profiles
}

func loadResourceAbundanceSettings(path string) climgen.ResourceAbundanceSettings {
	settings := climgen.DefaultResourceAbundanceSettings()
	if loaded, err := climgen.LoadResourceAbundanceSettings(path); err != nil {
		fmt.Printf("Using built-in resource abundance settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded resource abundance settings from %s\n", path)
	}
	return settings
}

func loadAgricultureSettings(path string) climgen.AgricultureProductivitySettings {
	settings := climgen.DefaultAgricultureProductivitySettings()
	if loaded, err := climgen.LoadAgricultureProductivitySettings(path); err != nil {
		fmt.Printf("Using built-in agriculture productivity settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded agriculture productivity settings from %s\n", path)
	}
	return settings
}

func loadWildlifeSettings(path string) climgen.WildlifeProductivitySettings {
	settings := climgen.DefaultWildlifeProductivitySettings()
	if loaded, err := climgen.LoadWildlifeProductivitySettings(path); err != nil {
		fmt.Printf("Using built-in wildlife productivity settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded wildlife productivity settings from %s\n", path)
	}
	return settings
}

func loadCoastalResourceSettings(path string) climgen.CoastalResourceSettings {
	settings := climgen.DefaultCoastalResourceSettings()
	if loaded, err := climgen.LoadCoastalResourceSettings(path); err != nil {
		fmt.Printf("Using built-in coastal resource settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded coastal resource settings from %s\n", path)
	}
	return settings
}

func loadWaterResourceSettings(path string) climgen.WaterResourceSettings {
	settings := climgen.DefaultWaterResourceSettings()
	if loaded, err := climgen.LoadWaterResourceSettings(path); err != nil {
		fmt.Printf("Using built-in water resource settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded water resource settings from %s\n", path)
	}
	return settings
}

func loadPopulationSupportSettings(path string) climgen.PopulationSupportSettings {
	settings := climgen.DefaultPopulationSupportSettings()
	if loaded, err := climgen.LoadPopulationSupportSettings(path); err != nil {
		fmt.Printf("Using built-in population support settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded population support settings from %s\n", path)
	}
	return settings
}

func loadLandRouteSettings(path string) climgen.LandRouteSettings {
	settings := climgen.DefaultLandRouteSettings()
	if loaded, err := climgen.LoadLandRouteSettings(path); err != nil {
		fmt.Printf("Using built-in land route settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded land route settings from %s\n", path)
	}
	return settings
}

func loadRiverRouteSettings(path string) climgen.RiverRouteSettings {
	settings := climgen.DefaultRiverRouteSettings()
	if loaded, err := climgen.LoadRiverRouteSettings(path); err != nil {
		fmt.Printf("Using built-in river route settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded river route settings from %s\n", path)
	}
	return settings
}

func loadMaritimeRouteSettings(path string) climgen.MaritimeRouteSettings {
	settings := climgen.DefaultMaritimeRouteSettings()
	if loaded, err := climgen.LoadMaritimeRouteSettings(path); err != nil {
		fmt.Printf("Using built-in maritime vessel settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded maritime vessel settings from %s\n", path)
	}
	return settings
}

func loadMaritimePortSettings(path string) climgen.MaritimePortSettings {
	settings := climgen.DefaultMaritimePortSettings()
	if loaded, err := climgen.LoadMaritimePortSettings(path); err != nil {
		fmt.Printf("Using built-in maritime port settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded maritime port settings from %s\n", path)
	}
	return settings
}

func loadCoastalTradeSettings(path string) climgen.CoastalTradeSettings {
	settings := climgen.DefaultCoastalTradeSettings()
	if loaded, err := climgen.LoadCoastalTradeSettings(path); err != nil {
		fmt.Printf("Using built-in coastal trade settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded coastal trade settings from %s\n", path)
	}
	return settings
}

func loadOceanTradeSettings(path string) climgen.OceanTradeSettings {
	settings := climgen.DefaultOceanTradeSettings()
	if loaded, err := climgen.LoadOceanTradeSettings(path); err != nil {
		fmt.Printf("Using built-in ocean trade settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded ocean trade settings from %s\n", path)
	}
	return settings
}

func selectMaritimeComparisonVessels(raw string, settings climgen.MaritimeRouteSettings) []string {
	addUnique := func(out *[]string, seen map[string]struct{}, name string) {
		if name == "" {
			return
		}
		if _, ok := settings.VesselByName(name); !ok {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		*out = append(*out, name)
	}

	out := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	if strings.TrimSpace(raw) == "" {
		for _, name := range []string{settings.DefaultVessel, "lateen-dhow", "knarr", "caravel"} {
			addUnique(&out, seen, name)
		}
	} else {
		for _, name := range strings.Split(raw, ",") {
			addUnique(&out, seen, strings.TrimSpace(name))
		}
	}
	if len(out) == 0 {
		if settings.DefaultVessel != "" {
			addUnique(&out, seen, settings.DefaultVessel)
		} else if len(settings.Vessels) > 0 {
			addUnique(&out, seen, settings.Vessels[0].Name)
		}
	}
	return out
}

func maritimeSettingsForVessel(settings climgen.MaritimeRouteSettings, vesselName string) climgen.MaritimeRouteSettings {
	override := settings
	override.DefaultVessel = vesselName
	return override
}

func maritimeOutputSuffix(defaultVessel, vesselName string) string {
	if vesselName == "" || vesselName == defaultVessel {
		return ""
	}
	replacer := strings.NewReplacer(" ", "_", "/", "_")
	return "_" + replacer.Replace(vesselName)
}

func loadOrGenerateTerrain(
	cacheStore *reviewCacheStore,
	terrainKey string,
	sites []terrain.Vector3D,
	cells []terrain.VoronoiCell,
	numPlates int,
	seed int64,
	landFrac float64,
) ([]float64, []bool, terrain.PlanetGenerationDiagnostics) {
	var elevation []float64
	var isLand []bool
	var diagnostics terrain.PlanetGenerationDiagnostics
	if cacheStore != nil {
		if cached, ok, err := cacheStore.LoadTerrain(terrainKey); err != nil {
			fmt.Printf("  terrain cache load failed, regenerating: %v\n", err)
		} else if ok {
			fmt.Printf("  terrain cache hit: %s\n", terrainKey)
			elevation = cached.Elevation
			isLand = cached.IsLand
			diagnostics = cached.Diagnostics
		}
	}
	if elevation == nil {
		elevation, isLand, diagnostics = terrain.GeneratePlanetElevationWithDiagnostics(sites, cells, numPlates, seed, landFrac)
		if cacheStore != nil {
			if err := cacheStore.SaveTerrain(terrainKey, &cachedTerrainReview{
				Elevation:   elevation,
				IsLand:      isLand,
				Diagnostics: diagnostics,
			}); err != nil {
				fmt.Printf("  terrain cache save failed: %v\n", err)
			}
		}
	}
	return elevation, isLand, diagnostics
}

func loadOrGenerateClimate(
	cacheStore *reviewCacheStore,
	terrainKey string,
	climateSites []climgen.Vector3D,
	climateCells []climgen.VoronoiCell,
	elevation []float64,
	climateAdj *climgen.FlatAdjacency,
	seed int64,
	enabled bool,
) *climgen.SeasonalClimateResult {
	if !enabled {
		return nil
	}
	var seasonalClimate *climgen.SeasonalClimateResult
	climateKey := climateCacheKey(terrainKey, seed)
	if cacheStore != nil {
		if cached, ok, err := cacheStore.LoadClimate(climateKey); err != nil {
			fmt.Printf("  climate cache load failed, recomputing: %v\n", err)
		} else if ok {
			fmt.Printf("  climate cache hit: %s\n", climateKey)
			seasonalClimate = cached
		}
	}
	if seasonalClimate == nil {
		climate, err := computeSeasonalClimate(climateSites, climateCells, elevation, climateAdj, seed)
		if err != nil {
			fmt.Printf("  seasonal climate failed, keeping terrain-only review: %v\n", err)
		} else {
			seasonalClimate = climate
			if cacheStore != nil {
				if err := cacheStore.SaveClimate(climateKey, climate); err != nil {
					fmt.Printf("  climate cache save failed: %v\n", err)
				}
			}
		}
	}
	return seasonalClimate
}

func computeSeasonalClimate(
	sites []climgen.Vector3D,
	cells []climgen.VoronoiCell,
	elevation []float64,
	adj *climgen.FlatAdjacency,
	seed int64,
) (*climgen.SeasonalClimateResult, error) {
	currentSettings := climgen.DefaultOceanCurrentSettings()
	currentSettings.Seed = seed
	currentResult, err := climgen.GenerateOceanCurrents(sites, cells, elevation, 0.0, currentSettings)
	if err != nil {
		return nil, fmt.Errorf("generate currents: %w", err)
	}

	tempSettings := climgen.DefaultTemperatureSettings()
	tempSettings.Seed = seed
	tempSettings.Solar.AxialTilt = 23.5
	tempSettings.Balance.MaxIterations = 500

	seasonalSettings := climgen.DefaultSeasonalTemperatureSettings()
	seasonalSettings.NumSeasons = 4
	seasonalSettings.NumCycles = 3
	seasonalSettings.ReferenceEquilibrium = true

	windSettings := climgen.DefaultWindSettings()
	windSettings.Seed = seed
	precipSettings := climgen.DefaultPrecipitationSettings()

	climate, err := climgen.GenerateSeasonalClimate(
		sites,
		elevation,
		0.0,
		adj,
		windSettings,
		currentResult,
		tempSettings,
		precipSettings,
		seasonalSettings,
	)
	if err != nil {
		return nil, fmt.Errorf("generate seasonal climate: %w", err)
	}
	return climate, nil
}

func computeClimateDrivenHydrologyFromClimate(
	sites []climgen.Vector3D,
	cells []climgen.VoronoiCell,
	elevation []float64,
	climate *climgen.SeasonalClimateResult,
) terrain.HydrologyDiagnostics {
	runoff := climgen.ComputeSeasonalRunoff(climate, elevation)
	terrainSites := make([]terrain.Vector3D, len(sites))
	terrainCells := make([]terrain.VoronoiCell, len(cells))
	for i, v := range sites {
		terrainSites[i] = terrain.Vector3D{X: v.X, Y: v.Y, Z: v.Z}
		terrainCells[i] = terrain.VoronoiCell{
			SiteIndex:           cells[i].SiteIndex,
			NeighborSiteIndices: append([]int32(nil), cells[i].NeighborSiteIndices...),
		}
	}
	return terrain.ComputeHydrologyDiagnosticsFromRunoff(terrainSites, terrainCells, elevation, runoff.AnnualRunoff)
}

func computeHydrologyAwareBiomes(
	climate *climgen.SeasonalClimateResult,
	elevation []float64,
	scaffold *terrain.HydrologyScaffold,
) *climgen.BiomeResult {
	hydro := hydrologyBiomeInputsFromScaffold(scaffold)
	return climgen.ClassifyBiomesSeasonalWithHydrology(climate, elevation, 0.0, hydro)
}

func computeVegetation(
	cells []climgen.VoronoiCell,
	climate *climgen.SeasonalClimateResult,
	biomes *climgen.BiomeResult,
	elevation []float64,
	scaffold *terrain.HydrologyScaffold,
	soils *climgen.SoilResult,
) *climgen.VegetationResult {
	coastalExposure := climgen.ComputeCoastalExposure(cells, elevation, 0.0)
	hydro := hydrologyBiomeInputsFromScaffold(scaffold)
	return climgen.ClassifyVegetation(climate, biomes, elevation, 0.0, hydro, coastalExposure, soils)
}

func computeSoils(
	cells []climgen.VoronoiCell,
	climate *climgen.SeasonalClimateResult,
	biomes *climgen.BiomeResult,
	elevation []float64,
	scaffold *terrain.HydrologyScaffold,
) *climgen.SoilResult {
	coastalExposure := climgen.ComputeCoastalExposure(cells, elevation, 0.0)
	hydro := hydrologyBiomeInputsFromScaffold(scaffold)
	return climgen.ClassifySoils(cells, climate, biomes, elevation, 0.0, hydro, coastalExposure)
}

func computeAgriculture(
	biomes *climgen.BiomeResult,
	soils *climgen.SoilResult,
	elevation []float64,
	scaffold *terrain.HydrologyScaffold,
	settings climgen.AgricultureProductivitySettings,
) *climgen.AgricultureResult {
	hydro := hydrologyBiomeInputsFromScaffold(scaffold)
	return climgen.ClassifyAgriculture(biomes, soils, elevation, 0.0, hydro, settings)
}

func computeWildlife(
	biomes *climgen.BiomeResult,
	vegetation *climgen.VegetationResult,
	soils *climgen.SoilResult,
	elevation []float64,
	scaffold *terrain.HydrologyScaffold,
	settings climgen.WildlifeProductivitySettings,
) *climgen.WildlifeResult {
	hydro := hydrologyBiomeInputsFromScaffold(scaffold)
	return climgen.ClassifyWildlife(biomes, vegetation, soils, elevation, 0.0, hydro, settings)
}

func computeCoastalResources(
	sites []climgen.Vector3D,
	cells []climgen.VoronoiCell,
	climate *climgen.SeasonalClimateResult,
	biomes *climgen.BiomeResult,
	soils *climgen.SoilResult,
	vegetation *climgen.VegetationResult,
	elevation []float64,
	scaffold *terrain.HydrologyScaffold,
	settings climgen.CoastalResourceSettings,
) *climgen.CoastalResourceResult {
	hydro := hydrologyBiomeInputsFromScaffold(scaffold)
	coastalExposure := climgen.ComputeCoastalExposure(cells, elevation, 0.0)
	return climgen.ClassifyCoastalResources(sites, cells, climate, biomes, soils, vegetation, elevation, 0.0, hydro, coastalExposure, settings)
}

func computeWaterResources(
	biomes *climgen.BiomeResult,
	soils *climgen.SoilResult,
	elevation []float64,
	scaffold *terrain.HydrologyScaffold,
	settings climgen.WaterResourceSettings,
) *climgen.WaterResourceResult {
	hydro := hydrologyBiomeInputsFromScaffold(scaffold)
	return climgen.ClassifyWaterResources(biomes, soils, elevation, 0.0, hydro, settings)
}

func computeResources(
	climate *climgen.SeasonalClimateResult,
	biomes *climgen.BiomeResult,
	soils *climgen.SoilResult,
	elevation []float64,
	scaffold *terrain.HydrologyScaffold,
	chains []terrain.HotspotChain,
	settings climgen.ResourceAbundanceSettings,
) *climgen.ResourceResult {
	hydro := hydrologyBiomeInputsFromScaffold(scaffold)
	geology := hotspotResourceInputs(len(elevation), chains)
	return climgen.ClassifyResourcesWithSettings(climate, biomes, soils, elevation, 0.0, hydro, geology, settings)
}

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

func hydrologyBiomeInputsFromScaffold(scaffold *terrain.HydrologyScaffold) *climgen.HydrologyBiomeInputs {
	if scaffold == nil {
		return nil
	}
	return &climgen.HydrologyBiomeInputs{
		Runoff:          append([]float64(nil), scaffold.Runoff...),
		ChannelStrength: append([]float64(nil), scaffold.ChannelStrength...),
		CellClass:       append([]string(nil), scaffold.CellClass...),
		WaterBodyLabel:  append([]int(nil), scaffold.WaterBodyLabel...),
	}
}

func hotspotResourceInputs(n int, chains []terrain.HotspotChain) *climgen.ResourceGeologyInputs {
	out := &climgen.ResourceGeologyInputs{
		HotspotStrength:    make([]float64, n),
		ContinentalHotspot: make([]float64, n),
	}
	for _, chain := range chains {
		for _, island := range chain.Islands {
			if island.CellIndex < 0 || island.CellIndex >= n {
				continue
			}
			if island.Strength > out.HotspotStrength[island.CellIndex] {
				out.HotspotStrength[island.CellIndex] = island.Strength
			}
			if !chain.IsOceanic && island.Strength > out.ContinentalHotspot[island.CellIndex] {
				out.ContinentalHotspot[island.CellIndex] = island.Strength
			}
		}
	}
	return out
}
