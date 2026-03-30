package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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

	for _, seed := range seeds {
		fmt.Printf("\nseed=%d\n", seed)
		elevation, isLand, diagnostics := terrain.GeneratePlanetElevationWithDiagnostics(sites, cells, *numPlates, seed, *landFrac)
		var seasonalClimate *climgen.SeasonalClimateResult
		if *climateHydrology || *climateBiomes {
			climate, err := computeSeasonalClimate(climateSites, climateCells, elevation, climateAdj, seed)
			if err != nil {
				fmt.Printf("  seasonal climate failed, keeping terrain-only review: %v\n", err)
			} else {
				seasonalClimate = climate
			}
		}
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
		printSummary(result, diagnostics)
		prefix := filepath.Join(*outputDir, fmt.Sprintf("seed_%d", seed))
		if *climateBiomes && seasonalClimate != nil {
			biomeResult := computeHydrologyAwareBiomes(seasonalClimate, elevation, diagnostics.Hydrology.Scaffold)
			printBiomeSummary(biomeResult)
			if *climateVegetation {
				vegetationResult := computeVegetation(climateCells, seasonalClimate, biomeResult, elevation, diagnostics.Hydrology.Scaffold)
				printVegetationSummary(vegetationResult)
				renderVegetationMap(sites, index, vegetationResult, prefix+"_vegetation.png", width, height)
			}
		}

		terrain.RenderShadedElevationMap(sites, elevation, index, prefix+"_shaded.png", width, height)
		terrain.RenderLandOceanMap(sites, elevation, isLand, index, prefix+"_landocean.png")
		if diagnostics.Hydrology.Scaffold != nil {
			terrain.RenderHydrologyOverlayMap(sites, elevation, diagnostics.Hydrology.Scaffold, index, prefix+"_hydrology.png", width, height)
		}
		terrain.RenderOrthoView(sites, elevation, index, 0, 0, prefix+"_globe.png")
	}
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

func printSummary(result terrain.EvaluationResult, diagnostics terrain.PlanetGenerationDiagnostics) {
	m := result.Metrics
	fmt.Printf("  score=%.1f passed=%v landMean=%.0f oceanMean=%.0f deep=%.3f shelf=%.3f mountains=%.3f major=%d largest=%.3f gini=%.3f fractal=%.3f tort=%.3f drain=%.3f endo=%.3f lakes=%.4f basins=%d hotspotCV=%.3f burst=%.3f bend=%.3f failed=%v\n",
		result.Score, result.Passed, m.MeanLandElevation, m.MeanOceanDepth, m.DeepOceanCoverage, m.ShelfCoverage,
		m.MountainCoverage, m.NumMajorLandmasses, m.LargestContinentPct, m.ContinentGini,
		m.FractalDimension, m.TortuosityRatio, m.FluvialChannelCoverage, m.EndorheicCatchmentPct,
		m.InlandLakeCoverage, m.NumMajorEndorheicBasins, m.HotspotSpacingCV, m.HotspotBurstiness,
		m.HotspotBendFraction, result.FailedMetrics)
	for _, region := range diagnostics.Hydrology.Regions {
		fmt.Printf("    hydro[%s]: cells=%d runoff=%.3f accum=%.2f channels=%.1f%% endo=%.1f%% lakes=%.1f%%\n",
			region.Name, region.CellCount, region.MeanRunoff, region.MeanAccumulation,
			region.ChannelCoverage*100, region.EndorheicCatchmentPct*100, region.InlandLakeReachPct*100)
	}
	for _, class := range diagnostics.Hydrology.Classes {
		fmt.Printf("    hydroClass[%s]=%d\n", class.Class, class.CellCount)
	}
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
) *climgen.VegetationResult {
	coastalExposure := climgen.ComputeCoastalExposure(cells, elevation, 0.0)
	hydro := hydrologyBiomeInputsFromScaffold(scaffold)
	return climgen.ClassifyVegetation(climate, biomes, elevation, 0.0, hydro, coastalExposure)
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

func printBiomeSummary(result *climgen.BiomeResult) {
	if result == nil {
		return
	}
	stats := climgen.GetBiomeStats(result)
	landCells := 0
	for _, biome := range result.Biomes {
		if biome != climgen.BiomeOcean {
			landCells++
		}
	}
	if landCells == 0 {
		return
	}
	aridCount := stats[climgen.BiomeDesertHot] + stats[climgen.BiomeDesertCold] + stats[climgen.BiomeSemiArid]
	forestCount := stats[climgen.BiomeTemperateRainforest] +
		stats[climgen.BiomeTemperateForest] +
		stats[climgen.BiomeBorealForest] +
		stats[climgen.BiomeTropicalSeasonalForest] +
		stats[climgen.BiomeTropicalRainforest]
	wetlandCount := stats[climgen.BiomeWetland]
	fmt.Printf("    biomeMetrics: arid=%.1f%% forest=%.1f%% wetland=%.1f%%\n",
		100*float64(aridCount)/float64(landCells),
		100*float64(forestCount)/float64(landCells),
		100*float64(wetlandCount)/float64(landCells),
	)

	type biomeCount struct {
		biome climgen.Biome
		count int
	}
	var sorted []biomeCount
	for biome, count := range stats {
		sorted = append(sorted, biomeCount{biome: biome, count: count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	fmt.Println("    biome summary:")
	for i := 0; i < limit; i++ {
		entry := sorted[i]
		fmt.Printf("      biome[%s]=%d (%.1f%%)\n",
			climgen.BiomeName(entry.biome),
			entry.count,
			100*float64(entry.count)/float64(landCells),
		)
	}
}

func printVegetationSummary(result *climgen.VegetationResult) {
	if result == nil {
		return
	}
	counts := make(map[climgen.VegetationType]int)
	landCells := 0
	for _, veg := range result.Types {
		if veg == climgen.VegetationOcean {
			continue
		}
		landCells++
		counts[veg]++
	}
	if landCells == 0 {
		return
	}
	woodyCount := counts[climgen.VegetationWoodland] +
		counts[climgen.VegetationForest] +
		counts[climgen.VegetationRainforest] +
		counts[climgen.VegetationRiparianForest] +
		counts[climgen.VegetationCloudForest] +
		counts[climgen.VegetationMangrove]
	openCount := counts[climgen.VegetationDesertSparse] +
		counts[climgen.VegetationShrubland] +
		counts[climgen.VegetationGrassland]
	wetCount := counts[climgen.VegetationWetland] +
		counts[climgen.VegetationSaltMarsh] +
		counts[climgen.VegetationPeatland] +
		counts[climgen.VegetationMangrove]
	riparianCount := counts[climgen.VegetationRiparianForest]
	coastalSpecialCount := counts[climgen.VegetationMangrove] + counts[climgen.VegetationSaltMarsh]
	fmt.Printf("    vegetationMetrics: woody=%.1f%% open=%.1f%% wet=%.1f%% riparian=%.1f%% coastalSpecial=%.1f%%\n",
		100*float64(woodyCount)/float64(landCells),
		100*float64(openCount)/float64(landCells),
		100*float64(wetCount)/float64(landCells),
		100*float64(riparianCount)/float64(landCells),
		100*float64(coastalSpecialCount)/float64(landCells),
	)

	type vegetationCount struct {
		veg   climgen.VegetationType
		count int
	}
	var sorted []vegetationCount
	for veg, count := range counts {
		sorted = append(sorted, vegetationCount{veg: veg, count: count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	fmt.Println("    vegetation summary:")
	for i := 0; i < limit; i++ {
		entry := sorted[i]
		fmt.Printf("      vegetation[%s]=%d (%.1f%%)\n",
			climgen.VegetationName(entry.veg),
			entry.count,
			100*float64(entry.count)/float64(landCells),
		)
	}
}

func renderVegetationMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	result *climgen.VegetationResult,
	filename string,
	width, height int,
) {
	if result == nil || index == nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(result.Types) {
				img.Set(px, py, climgen.VegetationColor(result.Types[cellIdx]))
			}
		}
	}
	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("  create vegetation map %s: %v\n", filename, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Printf("  encode vegetation map %s: %v\n", filename, err)
		return
	}
	fmt.Printf("  Saved %s\n", filename)
}
