package main

import (
	"fmt"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

func loadOrGenerateTerrain(
	cacheStore *reviewCacheStore,
	terrainKey string,
	sites []terrain.Vector3D,
	cells []terrain.VoronoiCell,
	numPlates int,
	seed int64,
	landFrac float64,
) ([]float64, []bool, terrain.PlanetGenerationDiagnostics) {
	phaseStart := reviewPhaseStart("terrain")
	var elevation []float64
	var isLand []bool
	var diagnostics terrain.PlanetGenerationDiagnostics
	cacheStatus := "miss"
	if cacheStore != nil {
		if cached, ok, err := cacheStore.LoadTerrain(terrainKey); err != nil {
			fmt.Printf("  terrain cache load failed, regenerating: %v\n", err)
		} else if ok {
			fmt.Printf("  terrain cache hit: %s\n", terrainKey)
			elevation = cached.Elevation
			isLand = cached.IsLand
			diagnostics = cached.Diagnostics
			cacheStatus = "hit"
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
	reviewPhaseDone("terrain", cacheStatus, phaseStart, fmt.Sprintf("key=%s", terrainKey))
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
	phaseStart := reviewPhaseStart("climate")
	var seasonalClimate *climgen.SeasonalClimateResult
	climateKey := climateCacheKey(terrainKey, seed)
	cacheStatus := "miss"
	if cacheStore != nil {
		if cached, ok, err := cacheStore.LoadClimate(climateKey); err != nil {
			fmt.Printf("  climate cache load failed, recomputing: %v\n", err)
		} else if ok {
			fmt.Printf("  climate cache hit: %s\n", climateKey)
			seasonalClimate = cached
			cacheStatus = "hit"
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
	reviewPhaseDone("climate", cacheStatus, phaseStart, fmt.Sprintf("key=%s", climateKey))
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
