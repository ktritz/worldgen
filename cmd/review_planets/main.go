package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"worldgen/climgen"
	"worldgen/icosphere"
	"worldgen/landgen/terrain"
)

func main() {
	level := flag.Int("level", 6, "icosphere subdivision level")
	numPlates := flag.Int("plates", 12, "number of tectonic plates")
	landFrac := flag.Float64("land", 0.29, "target land fraction")
	seedsFlag := flag.String("seeds", "4,6,7,42,84", "comma-separated seed list")
	outputDir := flag.String("out", "output/review_planets", "output directory")
	renderWidth := flag.Int("width", 0, "render width (defaults based on level)")
	climateHydrology := flag.Bool("climate-hydrology", true, "use climate-driven runoff for hydrology diagnostics")
	climateBiomes := flag.Bool("climate-biomes", true, "report seasonal hydrology-aware biome summaries")
	climateVegetation := flag.Bool("climate-vegetation", true, "report vegetation summaries from seasonal climate, hydrology, and biomes")
	climateSoils := flag.Bool("climate-soils", true, "report coarse soil summaries from seasonal climate, hydrology, and relief")
	climateAgriculture := flag.Bool("climate-agriculture", true, "report coarse agricultural and pastoral productivity from climate, soils, terrain, and hydrology")
	climateWildlife := flag.Bool("climate-wildlife", true, "report coarse biological resources from climate, vegetation, hydrology, and soils")
	climateCoastalResources := flag.Bool("climate-coastal-resources", true, "report coarse coastal and marine-adjacent resource access from climate, hydrology, soils, and vegetation")
	climateWaterResources := flag.Bool("climate-water-resources", true, "report coarse freshwater resource access from climate, soils, and hydrology")
	climateResources := flag.Bool("climate-resources", true, "report coarse geological resource provinces")
	climateSettlements := flag.Bool("climate-settlements", true, "report coarse settlement suitability from climate, hydrology, soils, vegetation, and resources")
	climatePopulation := flag.Bool("climate-population", true, "report coarse population support and urban potential from settlement, food, water, access, and resources")
	climateSettlementNetwork := flag.Bool("climate-settlement-network", true, "extract coarse settlement nodes and connectivity corridors from support fields")
	climateProtoCivilizations := flag.Bool("climate-proto-civilizations", true, "derive coarse proto-civilization seeds and hinterlands from settlement regions")
	climateTradeNetwork := flag.Bool("climate-trade-network", true, "derive coarse trade corridors and hub hierarchy from proto-civilizations and settlement links")
	climatePolitySpheres := flag.Bool("climate-polity-spheres", true, "derive coarse polity spheres from proto-civilization capitals and trade hubs")
	settlementProfiles := flag.Bool("settlement-profiles", true, "report fantasy settlement preference overlays")
	profileCatalogFile := flag.String("profile-catalog-file", "config/profile_catalog_fantasy.json", "JSON profile catalog describing ancestry/culture modifiers")
	agricultureFile := flag.String("agriculture-productivity-file", "config/agriculture_productivity_earthlike.json", "JSON agriculture/pastoral productivity configuration")
	wildlifeFile := flag.String("wildlife-productivity-file", "config/wildlife_productivity_earthlike.json", "JSON wildlife/biological productivity configuration")
	coastalResourceFile := flag.String("coastal-resources-file", "config/coastal_resources_earthlike.json", "JSON coastal resource productivity configuration")
	waterResourceFile := flag.String("water-resources-file", "config/water_resources_earthlike.json", "JSON freshwater resource productivity configuration")
	resourceAbundanceFile := flag.String("resource-abundance-file", "config/resource_abundance_earthlike.json", "JSON resource abundance/scarcity configuration")
	populationSupportFile := flag.String("population-support-file", "config/population_support_earthlike.json", "JSON population support configuration")
	landRouteFile := flag.String("land-route-file", "config/land_routes_earthlike.json", "JSON overland trade route mode configuration")
	useCache := flag.Bool("cache", true, "reuse cached terrain and seasonal climate artifacts under the review output directory")
	flag.Parse()

	seeds, err := parseSeeds(*seedsFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid seeds: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output dir: %v\n", err)
		os.Exit(1)
	}

	width := *renderWidth
	if width <= 0 {
		switch {
		case *level >= 8:
			width = 4096
		case *level >= 7:
			width = 3072
		default:
			width = 2048
		}
	}
	height := width / 2

	fmt.Printf("Generating review set: level=%d plates=%d land=%.2f width=%d seeds=%v\n",
		*level, *numPlates, *landFrac, width, seeds)

	vertices, faces := icosphere.CreateIcosphere(*level)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)

	sites := make([]terrain.Vector3D, len(vertices))
	climateSites := make([]climgen.Vector3D, len(vertices))
	for i, v := range vertices {
		sites[i] = terrain.Vector3D{X: v.X, Y: v.Y, Z: v.Z}
		climateSites[i] = climgen.Vector3D{X: v.X, Y: v.Y, Z: v.Z}
	}
	cells := make([]terrain.VoronoiCell, len(voronoiCells))
	climateCells := make([]climgen.VoronoiCell, len(voronoiCells))
	for i, cell := range voronoiCells {
		cells[i] = terrain.VoronoiCell{
			SiteIndex:           int32(cell.SiteIndex),
			NeighborSiteIndices: append([]int32(nil), cell.NeighborSiteIndices...),
		}
		climateCells[i] = climgen.VoronoiCell{
			SiteIndex:           int32(cell.SiteIndex),
			NeighborSiteIndices: append([]int32(nil), cell.NeighborSiteIndices...),
		}
	}
	climateAdj := climgen.BuildFlatAdjacency(climateCells)
	index := terrain.BuildSpatialIndex(sites)

	var cacheStore *reviewCacheStore
	if *useCache {
		cacheStore = newReviewCacheStore(*outputDir)
	}

	profileCatalog := loadProfileCatalogWithFallback(*profileCatalogFile)
	profiles := extractSettlementProfilesFromCatalog(*settlementProfiles, profileCatalog)
	resourceSettings := loadResourceAbundanceSettings(*resourceAbundanceFile)
	agricultureSettings := loadAgricultureSettings(*agricultureFile)
	wildlifeSettings := loadWildlifeSettings(*wildlifeFile)
	coastalResourceSettings := loadCoastalResourceSettings(*coastalResourceFile)
	waterResourceSettings := loadWaterResourceSettings(*waterResourceFile)
	populationSettings := loadPopulationSupportSettings(*populationSupportFile)
	landRouteSettings := loadLandRouteSettings(*landRouteFile)
	baselineRecords := make([]baselineSeedMetrics, 0, len(seeds))

	for _, seed := range seeds {
		fmt.Printf("\nseed=%d\n", seed)
		record := newBaselineSeedMetrics(seed)
		terrainKey := terrainCacheKey(*level, *numPlates, *landFrac, seed)
		elevation, isLand, diagnostics := loadOrGenerateTerrain(cacheStore, terrainKey, sites, cells, *numPlates, seed, *landFrac)
		seasonalClimate := loadOrGenerateClimate(cacheStore, terrainKey, climateSites, climateCells, elevation, climateAdj, seed, *climateHydrology || *climateBiomes)

		if *climateHydrology {
			if seasonalClimate == nil {
				fmt.Printf("  climate hydrology unavailable, keeping proxy runoff\n")
			} else {
				hydro := computeClimateDrivenHydrologyFromClimate(climateSites, climateCells, elevation, seasonalClimate)
				hydro.PostDetailBreachedSinks = diagnostics.Hydrology.PostDetailBreachedSinks
				diagnostics.Hydrology = hydro
				fmt.Println("  Hydrology diagnostics: climate-driven runoff override enabled")
			}
		}

		result := terrain.EvaluateTerrainWithHotspots(sites, cells, elevation, diagnostics.HotspotChains)
		record.Score = result.Score
		record.Drain = result.Metrics.FluvialChannelCoverage * 100
		record.Endo = result.Metrics.EndorheicCatchmentPct * 100
		printSummary(result, diagnostics)
		prefix := filepath.Join(*outputDir, fmt.Sprintf("seed_%d", seed))

		if *climateBiomes && seasonalClimate != nil {
			biomeResult := computeHydrologyAwareBiomes(seasonalClimate, elevation, diagnostics.Hydrology.Scaffold)
			record.Arid, record.Forest, record.Wetland = collectBiomeMetrics(biomeResult)
			printBiomeSummary(biomeResult)

			var soilResult *climgen.SoilResult
			var vegetationResult *climgen.VegetationResult
			var waterResourceResult *climgen.WaterResourceResult
			var agricultureResult *climgen.AgricultureResult
			var wildlifeResult *climgen.WildlifeResult
			var coastalResourceResult *climgen.CoastalResourceResult
			if *climateSoils || *climateVegetation {
				soilResult = computeSoils(climateCells, seasonalClimate, biomeResult, elevation, diagnostics.Hydrology.Scaffold)
			}
			if *climateVegetation {
				vegetationResult = computeVegetation(climateCells, seasonalClimate, biomeResult, elevation, diagnostics.Hydrology.Scaffold, soilResult)
				record.Woody = collectVegetationMetrics(vegetationResult)
				printVegetationSummary(vegetationResult)
				renderVegetationMap(sites, index, vegetationResult, prefix+"_vegetation.png", width, height)
			}
			if *climateSoils && soilResult != nil {
				printSoilSummary(soilResult)
				renderSoilMap(sites, index, soilResult, prefix+"_soils.png", width, height)
			}
			if *climateAgriculture && soilResult != nil {
				agricultureResult = computeAgriculture(biomeResult, soilResult, elevation, diagnostics.Hydrology.Scaffold, agricultureSettings)
				record.Crop, record.Pasture = collectAgricultureMetrics(agricultureResult)
				printAgricultureSummary(agricultureResult)
				renderAgricultureMap(sites, index, agricultureResult, prefix+"_agriculture.png", width, height)
			}
			if *climateWildlife && vegetationResult != nil {
				wildlifeResult = computeWildlife(biomeResult, vegetationResult, soilResult, elevation, diagnostics.Hydrology.Scaffold, wildlifeSettings)
				record.Game, record.Timber = collectWildlifeMetrics(wildlifeResult)
				printWildlifeSummary(wildlifeResult)
				renderWildlifeMap(sites, index, wildlifeResult, prefix+"_wildlife.png", width, height)
			}
			if *climateWaterResources && soilResult != nil {
				waterResourceResult = computeWaterResources(biomeResult, soilResult, elevation, diagnostics.Hydrology.Scaffold, waterResourceSettings)
				record.Reliable, record.Groundwater = collectWaterMetrics(waterResourceResult)
				printWaterResourceSummary(waterResourceResult)
				renderWaterResourceMap(sites, index, waterResourceResult, prefix+"_water_resources.png", width, height)
			}
			if *climateCoastalResources && vegetationResult != nil && soilResult != nil {
				coastalResourceResult = computeCoastalResources(climateSites, climateCells, seasonalClimate, biomeResult, soilResult, vegetationResult, elevation, diagnostics.Hydrology.Scaffold, coastalResourceSettings)
				record.Fishery, record.Shellfish = collectCoastalMetrics(coastalResourceResult)
				printCoastalResourceSummary(coastalResourceResult)
				renderCoastalResourceMap(sites, index, coastalResourceResult, prefix+"_coastal_resources.png", width, height)
				renderCoastalUpwellingMap(sites, index, coastalResourceResult, prefix+"_coastal_upwelling.png", width, height)
			}
			var resourceResult *climgen.ResourceResult
			if *climateResources && soilResult != nil {
				resourceResult = computeResources(seasonalClimate, biomeResult, soilResult, elevation, diagnostics.Hydrology.Scaffold, diagnostics.HotspotChains, resourceSettings)
				record.Metallic, record.Fuel, record.Luxury = collectResourceMetrics(resourceResult)
				printResourceSummary(resourceResult, len(sites))
				renderResourceMap(sites, index, resourceResult, prefix+"_resources.png", width, height)
				renderResourcePotentialMap(sites, index, resourceResult, climgen.ResourceGoldOre, prefix+"_resource_gold_potential.png", width, height)
				renderResourcePotentialMap(sites, index, resourceResult, climgen.ResourceLeadSilverOre, prefix+"_resource_leadsilver_potential.png", width, height)
				renderResourcePotentialMap(sites, index, resourceResult, climgen.ResourceGemstones, prefix+"_resource_gems_potential.png", width, height)
			}
			if *climateSettlements && soilResult != nil {
				settlementResult := computeSettlement(climateCells, seasonalClimate, biomeResult, soilResult, vegetationResult, waterResourceResult, resourceResult, elevation, diagnostics.Hydrology.Scaffold)
				record.Favorable, record.Prime = collectSettlementMetrics(settlementResult)
				printSettlementSummary(settlementResult)
				renderSettlementMap(sites, index, settlementResult, prefix+"_settlements.png", width, height)
				if *settlementProfiles {
					preferences := computeSettlementPreferences(biomeResult, soilResult, vegetationResult, settlementResult, elevation, profiles)
					printSettlementPreferenceSummary(preferences)
					renderSettlementPreferenceMap(sites, index, preferences, prefix+"_settlement_preferences.png", width, height)
				}
				if *climatePopulation {
					populationResult := computePopulation(
						settlementResult,
						agricultureResult,
						wildlifeResult,
						waterResourceResult,
						coastalResourceResult,
						resourceResult,
						elevation,
						populationSettings,
					)
					record.Frontier, record.Settled, record.DensePop, record.UrbanPop = collectPopulationMetrics(populationResult)
					printPopulationSummary(populationResult)
					renderPopulationMap(sites, index, populationResult, prefix+"_population.png", width, height)
					if *climateSettlementNetwork {
						networkResult := computeSettlementNetwork(climateSites, climateCells, settlementResult, populationResult, biomeResult, soilResult, resourceResult, elevation)
						printSettlementNetworkSummary(networkResult)
						renderSettlementNetworkMap(sites, elevation, index, networkResult, prefix+"_settlement_network.png", width, height)
						renderSettlementRegionMap(sites, elevation, index, networkResult, prefix+"_settlement_regions.png", width, height)
						if *climateProtoCivilizations {
							protoResult := computeProtoCivilizations(climateCells, networkResult, settlementResult, populationResult, biomeResult, soilResult, elevation)
							printProtoCivilizationSummary(protoResult)
							renderProtoCivilizationMap(sites, elevation, index, protoResult, networkResult, prefix+"_proto_civilizations.png", width, height)
							if *climateTradeNetwork {
								landRouteResult := computeLandRoutes(
									settlementResult,
									populationResult,
									biomeResult,
									vegetationResult,
									soilResult,
									wildlifeResult,
									waterResourceResult,
									elevation,
									hydrologyBiomeInputsFromScaffold(diagnostics.Hydrology.Scaffold),
									landRouteSettings,
								)
								printLandRouteSummary(landRouteResult)
								renderLandRouteRiskMap(sites, elevation, index, landRouteResult, prefix+"_land_route_risk.png", width, height)
								tradeResult := computeTradeNetwork(climateCells, networkResult, protoResult, landRouteResult)
								printTradeNetworkSummary(tradeResult, networkResult)
								renderTradeNetworkMap(sites, elevation, index, tradeResult, networkResult, prefix+"_trade_network.png", width, height)
								if *climatePolitySpheres {
									polityResult := computePolitySpheres(climateCells, networkResult, protoResult, tradeResult, populationResult, settlementResult, elevation)
									printPolitySphereSummary(polityResult, networkResult)
									renderPolitySphereMap(sites, elevation, index, polityResult, networkResult, prefix+"_polity_spheres.png", width, height)
									polityProfiles := computePolityProfiles(climateCells, polityResult, networkResult, tradeResult, biomeResult, vegetationResult, soilResult, diagnostics.Hydrology.Scaffold, profileCatalog)
									printPolityProfileSummary(polityProfiles)
								}
							}
						}
					}
				}
			}
		}

		terrain.RenderShadedElevationMap(sites, elevation, index, prefix+"_shaded.png", width, height)
		terrain.RenderLandOceanMap(sites, elevation, isLand, index, prefix+"_landocean.png")
		if diagnostics.Hydrology.Scaffold != nil {
			terrain.RenderHydrologyOverlayMap(sites, elevation, diagnostics.Hydrology.Scaffold, index, prefix+"_hydrology.png", width, height)
		}
		terrain.RenderOrthoView(sites, elevation, index, 0, 0, prefix+"_globe.png")
		baselineRecords = append(baselineRecords, record)
	}

	if err := writeBaselineReport(*outputDir, baselineRecords); err != nil {
		fmt.Fprintf(os.Stderr, "write baseline report: %v\n", err)
	} else if len(baselineRecords) > 0 {
		fmt.Printf("\nSaved %s\n", filepath.Join(*outputDir, "baseline_summary.txt"))
	}
}
