package main

import (
	"fmt"
	"math"
	"sort"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

func printSummary(result terrain.EvaluationResult, diagnostics terrain.PlanetGenerationDiagnostics, sites []climgen.Vector3D, cells []climgen.VoronoiCell) {
	m := result.Metrics
	fmt.Printf("  score=%.1f passed=%v landMean=%.0f oceanMean=%.0f deep=%.3f shelf=%.3f mountains=%.3f major=%d largest=%.3f gini=%.3f fractal=%.3f tort=%.3f drain=%.3f endo=%.3f lakes=%.4f basins=%d hotspotChains=%d hotspotCV=%.3f burst=%.3f bend=%.3f failed=%v\n",
		result.Score, result.Passed, m.MeanLandElevation, m.MeanOceanDepth, m.DeepOceanCoverage, m.ShelfCoverage,
		m.MountainCoverage, m.NumMajorLandmasses, m.LargestContinentPct, m.ContinentGini,
		m.FractalDimension, m.TortuosityRatio, m.FluvialChannelCoverage, m.EndorheicCatchmentPct,
		m.InlandLakeCoverage, m.NumMajorEndorheicBasins, m.HotspotChainCount, m.HotspotSpacingCV, m.HotspotBurstiness,
		m.HotspotBendFraction, result.FailedMetrics)
	printHotspotDiagnostics(diagnostics.HotspotChains, sites, cells)
	for _, region := range diagnostics.Hydrology.Regions {
		fmt.Printf("    hydro[%s]: cells=%d runoff=%.3f accum=%.2f channels=%.1f%% endo=%.1f%% lakes=%.1f%%\n",
			region.Name, region.CellCount, region.MeanRunoff, region.MeanAccumulation,
			region.ChannelCoverage*100, region.EndorheicCatchmentPct*100, region.InlandLakeReachPct*100)
	}
	for _, class := range diagnostics.Hydrology.Classes {
		fmt.Printf("    hydroClass[%s]=%d\n", class.Class, class.CellCount)
	}
}

func printHotspotDiagnostics(chains []terrain.HotspotChain, sites []climgen.Vector3D, cells []climgen.VoronoiCell) {
	if len(chains) == 0 {
		return
	}
	oceanicChains, continentalChains := 0, 0
	oceanicIslands, continentalIslands := 0, 0
	oceanicSpacing := make([]float64, 0)
	for _, chain := range chains {
		if chain.IsOceanic {
			oceanicChains++
			oceanicIslands += len(chain.Islands)
			oceanicSpacing = append(oceanicSpacing, hotspotIslandSpacingsDeg(chain, sites)...)
		} else {
			continentalChains++
			continentalIslands += len(chain.Islands)
		}
	}
	meanSpacing := meanFloat64(oceanicSpacing)
	p10Spacing := percentileFloat64(oceanicSpacing, 0.10)
	meanNeighborDeg := reviewMeanNeighborDegrees(sites, cells)
	spacingToMesh := 0.0
	if meanNeighborDeg > 0 {
		spacingToMesh = meanSpacing / meanNeighborDeg
	}
	fmt.Printf(
		"    hotspotDiag: chains=%d oceanicChains=%d continentalChains=%d islands=%d oceanicIslands=%d continentalIslands=%d meanOceanicSpacingDeg=%.3f p10OceanicSpacingDeg=%.3f meanNeighborDeg=%.3f spacingToMesh=%.2f\n",
		len(chains),
		oceanicChains,
		continentalChains,
		oceanicIslands+continentalIslands,
		oceanicIslands,
		continentalIslands,
		meanSpacing,
		p10Spacing,
		meanNeighborDeg,
		spacingToMesh,
	)
}

func hotspotIslandSpacingsDeg(chain terrain.HotspotChain, sites []climgen.Vector3D) []float64 {
	if len(chain.Islands) < 2 || len(sites) == 0 {
		return nil
	}
	out := make([]float64, 0, len(chain.Islands)-1)
	for i := 1; i < len(chain.Islands); i++ {
		prev := chain.Islands[i-1].CellIndex
		next := chain.Islands[i].CellIndex
		if prev < 0 || next < 0 || prev >= len(sites) || next >= len(sites) {
			continue
		}
		out = append(out, reviewGreatCircleDistanceDeg(sites[prev], sites[next]))
	}
	return out
}

func reviewMeanNeighborDegrees(sites []climgen.Vector3D, cells []climgen.VoronoiCell) float64 {
	if len(sites) != len(cells) {
		return 0
	}
	total := 0.0
	count := 0
	for i, cell := range cells {
		for _, raw := range cell.NeighborSiteIndices {
			j := int(raw)
			if j <= i || j < 0 || j >= len(sites) {
				continue
			}
			total += reviewGreatCircleDistanceDeg(sites[i], sites[j])
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func reviewGreatCircleDistanceDeg(a, b climgen.Vector3D) float64 {
	dot := a.X*b.X + a.Y*b.Y + a.Z*b.Z
	if dot > 1 {
		dot = 1
	}
	if dot < -1 {
		dot = -1
	}
	return math.Acos(dot) * 180 / math.Pi
}

func meanFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total / float64(len(values))
}

func percentileFloat64(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func printLandComponentDiagnostics(cells []climgen.VoronoiCell, elevation []float64, seaLevel float64) {
	if len(cells) == 0 || len(elevation) == 0 {
		return
	}
	seen := make([]bool, len(cells))
	componentSizes := make([]int, 0)
	for i := range cells {
		if i >= len(elevation) || elevation[i] < seaLevel || seen[i] {
			continue
		}
		size := 0
		queue := []int{i}
		seen[i] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			size++
			for _, raw := range cells[cur].NeighborSiteIndices {
				neighbor := int(raw)
				if neighbor < 0 || neighbor >= len(cells) || neighbor >= len(elevation) || seen[neighbor] || elevation[neighbor] < seaLevel {
					continue
				}
				seen[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
		componentSizes = append(componentSizes, size)
	}
	if len(componentSizes) == 0 {
		return
	}
	sort.Sort(sort.Reverse(sort.IntSlice(componentSizes)))
	totalLand := 0
	for _, size := range componentSizes {
		totalLand += size
	}
	areaScale := climgen.MeshAreaResolutionScale(len(cells))
	tiny, small, medium, large := 0, 0, 0, 0
	islandAreaEq := 0.0
	for i, size := range componentSizes {
		if i > 0 {
			islandAreaEq += float64(size) * areaScale
		}
		areaEq := float64(size) * areaScale
		switch {
		case areaEq < 1:
			tiny++
		case areaEq < 4:
			small++
		case areaEq < 16:
			medium++
		default:
			large++
		}
	}
	meanIslandAreaEq := 0.0
	if len(componentSizes) > 1 {
		meanIslandAreaEq = islandAreaEq / float64(len(componentSizes)-1)
	}
	fmt.Printf(
		"    landComponentDiag: components=%d largestCells=%d largestPct=%.3f islands=%d tinyEq=%d smallEq=%d mediumEq=%d largeEq=%d meanIslandAreaEq=%.2f\n",
		len(componentSizes),
		componentSizes[0],
		float64(componentSizes[0])/float64(totalLand),
		len(componentSizes)-1,
		tiny,
		small,
		medium,
		large,
		meanIslandAreaEq,
	)
}

func printLandLatitudeDiagnostics(sites []climgen.Vector3D, elevation []float64, seaLevel float64) {
	if len(sites) == 0 || len(elevation) == 0 {
		return
	}
	count := 0
	meanLatitude := 0.0
	meanAbsLatitude := 0.0
	tropical, subtropical, temperate, subpolar, polar := 0, 0, 0, 0, 0
	for i, site := range sites {
		if i >= len(elevation) || elevation[i] < seaLevel {
			continue
		}
		lat := math.Asin(site.Normalize().Z) * 180 / math.Pi
		absLat := math.Abs(lat)
		count++
		meanLatitude += lat
		meanAbsLatitude += absLat
		switch {
		case absLat < 23.5:
			tropical++
		case absLat < 35:
			subtropical++
		case absLat < 50:
			temperate++
		case absLat < 66.5:
			subpolar++
		default:
			polar++
		}
	}
	if count == 0 {
		return
	}
	denom := float64(count)
	fmt.Printf(
		"    landLatitudeDiag: mean=%.2f meanAbs=%.2f tropical=%.1f%% subtropical=%.1f%% temperate=%.1f%% subpolar=%.1f%% polar=%.1f%%\n",
		meanLatitude/denom,
		meanAbsLatitude/denom,
		100*float64(tropical)/denom,
		100*float64(subtropical)/denom,
		100*float64(temperate)/denom,
		100*float64(subpolar)/denom,
		100*float64(polar)/denom,
	)
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

func printSoilSummary(result *climgen.SoilResult) {
	if result == nil {
		return
	}
	counts := make(map[climgen.SoilType]int)
	landCells := 0
	for _, soil := range result.Types {
		if soil == climgen.SoilOcean {
			continue
		}
		landCells++
		counts[soil]++
	}
	if landCells == 0 {
		return
	}
	fertileCount := counts[climgen.SoilTemperateLoam] + counts[climgen.SoilAlluvial]
	wetCount := counts[climgen.SoilOrganicWet] + counts[climgen.SoilPeat] + counts[climgen.SoilAlluvial]
	aridCount := counts[climgen.SoilAridMineral] + counts[climgen.SoilDrySteppe] + counts[climgen.SoilSalineCoastal]
	fmt.Printf("    soilMetrics: fertile=%.1f%% wet=%.1f%% arid=%.1f%%\n",
		100*float64(fertileCount)/float64(landCells),
		100*float64(wetCount)/float64(landCells),
		100*float64(aridCount)/float64(landCells),
	)

	type soilCount struct {
		soil  climgen.SoilType
		count int
	}
	var sorted []soilCount
	for soil, count := range counts {
		sorted = append(sorted, soilCount{soil: soil, count: count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	fmt.Println("    soil summary:")
	for i := 0; i < limit; i++ {
		entry := sorted[i]
		fmt.Printf("      soil[%s]=%d (%.1f%%)\n",
			climgen.SoilName(entry.soil),
			entry.count,
			100*float64(entry.count)/float64(landCells),
		)
	}
}

func printAgricultureSummary(result *climgen.AgricultureResult) {
	if result == nil {
		return
	}
	counts := make(map[climgen.AgricultureType]int)
	landCells := 0
	for _, ag := range result.Types {
		if ag == climgen.AgricultureOcean {
			continue
		}
		landCells++
		counts[ag]++
	}
	if landCells == 0 {
		return
	}
	cropCount := counts[climgen.AgricultureDryFarming] +
		counts[climgen.AgricultureMixedFarming] +
		counts[climgen.AgricultureIntensiveCropland] +
		counts[climgen.AgricultureFloodplainCropland]
	pastureCount := counts[climgen.AgriculturePastoral]
	floodplainCount := counts[climgen.AgricultureFloodplainCropland]
	fmt.Printf("    agricultureMetrics: crop=%.1f%% pasture=%.1f%% floodplain=%.1f%%\n",
		100*float64(cropCount)/float64(landCells),
		100*float64(pastureCount)/float64(landCells),
		100*float64(floodplainCount)/float64(landCells),
	)

	type agricultureCount struct {
		typ   climgen.AgricultureType
		count int
	}
	var sorted []agricultureCount
	for typ, count := range counts {
		sorted = append(sorted, agricultureCount{typ: typ, count: count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	fmt.Println("    agriculture summary:")
	for i := 0; i < limit; i++ {
		entry := sorted[i]
		fmt.Printf("      agriculture[%s]=%d (%.1f%%)\n",
			climgen.AgricultureName(entry.typ),
			entry.count,
			100*float64(entry.count)/float64(landCells),
		)
	}
}

func printWildlifeSummary(result *climgen.WildlifeResult) {
	if result == nil {
		return
	}
	counts := make(map[climgen.WildlifeType]int)
	landCells := 0
	for _, w := range result.Types {
		if w == climgen.WildlifeOcean {
			continue
		}
		landCells++
		counts[w]++
	}
	if landCells == 0 {
		return
	}
	gameCount := counts[climgen.WildlifeGrazingGame] + counts[climgen.WildlifeForestGame] + counts[climgen.WildlifeWetlandGame]
	peltCount := counts[climgen.WildlifePelts]
	timberCount := counts[climgen.WildlifeTimber]
	fmt.Printf("    wildlifeMetrics: game=%.1f%% pelts=%.1f%% timber=%.1f%%\n",
		100*float64(gameCount)/float64(landCells),
		100*float64(peltCount)/float64(landCells),
		100*float64(timberCount)/float64(landCells),
	)

	type wildlifeCount struct {
		typ   climgen.WildlifeType
		count int
	}
	var sorted []wildlifeCount
	for typ, count := range counts {
		sorted = append(sorted, wildlifeCount{typ: typ, count: count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	fmt.Println("    wildlife summary:")
	for i := 0; i < limit; i++ {
		entry := sorted[i]
		fmt.Printf("      wildlife[%s]=%d (%.1f%%)\n",
			climgen.WildlifeName(entry.typ),
			entry.count,
			100*float64(entry.count)/float64(landCells),
		)
	}
}

func printResourceSummary(result *climgen.ResourceResult, totalCells int) {
	if result == nil {
		return
	}
	counts := make(map[climgen.ResourceType]int)
	landCells := 0
	for _, r := range result.Types {
		if r == climgen.ResourceOcean {
			continue
		}
		landCells++
		counts[r]++
	}
	if landCells == 0 {
		return
	}
	metallic := counts[climgen.ResourceIronOre] +
		counts[climgen.ResourceCopperOre] +
		counts[climgen.ResourceLeadSilverOre] +
		counts[climgen.ResourceGoldOre]
	fuel := counts[climgen.ResourceCoal] + counts[climgen.ResourceOilGas]
	bulk := counts[climgen.ResourceClayAggregate] + counts[climgen.ResourceIndustrialStone] + counts[climgen.ResourceEvaporite]
	surficial := counts[climgen.ResourcePlacerAlluvial]
	luxury := counts[climgen.ResourceGemstones]
	fmt.Printf("    resourceMetrics: metallic=%.1f%% fuel=%.1f%% bulk=%.1f%% surficial=%.1f%% luxury=%.1f%%\n",
		100*float64(metallic)/float64(landCells),
		100*float64(fuel)/float64(landCells),
		100*float64(bulk)/float64(landCells),
		100*float64(surficial)/float64(landCells),
		100*float64(luxury)/float64(landCells),
	)
	fmt.Printf("    metallicDetail: iron=%.1f%% copper=%.1f%% leadsilver=%.1f%% gold=%.1f%% gems=%.1f%%\n",
		100*float64(counts[climgen.ResourceIronOre])/float64(landCells),
		100*float64(counts[climgen.ResourceCopperOre])/float64(landCells),
		100*float64(counts[climgen.ResourceLeadSilverOre])/float64(landCells),
		100*float64(counts[climgen.ResourceGoldOre])/float64(landCells),
		100*float64(counts[climgen.ResourceGemstones])/float64(landCells),
	)
	highGoldPotential := 0
	highLeadSilverPotential := 0
	highGemPotential := 0
	if result.Diagnostics != nil {
		for i, r := range result.Types {
			if r == climgen.ResourceOcean {
				continue
			}
			if climgen.ResourcePotential(result.Diagnostics, climgen.ResourceGoldOre, i) >= 0.35 {
				highGoldPotential++
			}
			if climgen.ResourcePotential(result.Diagnostics, climgen.ResourceLeadSilverOre, i) >= 0.35 {
				highLeadSilverPotential++
			}
			if climgen.ResourcePotential(result.Diagnostics, climgen.ResourceGemstones, i) >= 0.35 {
				highGemPotential++
			}
		}
	}
	fmt.Printf("    resourcePotentialHotspots: leadsilver=%.1f%% gold=%.1f%% gems=%.1f%%\n",
		100*float64(highLeadSilverPotential)/float64(landCells),
		100*float64(highGoldPotential)/float64(landCells),
		100*float64(highGemPotential)/float64(landCells),
	)
	if totalCells > 0 && result.Diagnostics != nil {
		cellAreaKm2 := 4 * math.Pi * 6371.0 * 6371.0 / float64(totalCells)
		integral := func(resource climgen.ResourceType) float64 {
			total := 0.0
			for i, r := range result.Types {
				if r == climgen.ResourceOcean {
					continue
				}
				total += climgen.ResourcePotential(result.Diagnostics, resource, i) * cellAreaKm2
			}
			return total
		}
		fmt.Printf("    resourcePotentialIntegral[km2eq]: iron=%.3e copper=%.3e leadsilver=%.3e gold=%.3e gems=%.3e coal=%.3e oilgas=%.3e\n",
			integral(climgen.ResourceIronOre),
			integral(climgen.ResourceCopperOre),
			integral(climgen.ResourceLeadSilverOre),
			integral(climgen.ResourceGoldOre),
			integral(climgen.ResourceGemstones),
			integral(climgen.ResourceCoal),
			integral(climgen.ResourceOilGas),
		)
	}
	type resourceCount struct {
		resource climgen.ResourceType
		count    int
	}
	var sorted []resourceCount
	for resource, count := range counts {
		sorted = append(sorted, resourceCount{resource: resource, count: count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	fmt.Println("    resource summary:")
	for i := 0; i < limit; i++ {
		entry := sorted[i]
		fmt.Printf("      resource[%s]=%d (%.1f%%)\n",
			climgen.ResourceName(entry.resource),
			entry.count,
			100*float64(entry.count)/float64(landCells),
		)
	}
}

func printSettlementSummary(result *climgen.SettlementResult) {
	if result == nil {
		return
	}
	counts := make(map[climgen.SettlementClass]int)
	landCells := 0
	scoreSum := 0.0
	for i, c := range result.Classes {
		if c == climgen.SettlementOcean {
			continue
		}
		landCells++
		counts[c]++
		if result.Diagnostics != nil && i < len(result.Diagnostics.Suitability) {
			scoreSum += result.Diagnostics.Suitability[i]
		}
	}
	if landCells == 0 {
		return
	}
	meanSuitability := scoreSum / float64(landCells)
	favorable := counts[climgen.SettlementFavorable] + counts[climgen.SettlementPrime]
	fmt.Printf("    settlementMetrics: mean=%.2f favorable=%.1f%% prime=%.1f%%\n",
		meanSuitability,
		100*float64(favorable)/float64(landCells),
		100*float64(counts[climgen.SettlementPrime])/float64(landCells),
	)
	if result.Diagnostics != nil {
		printFieldDistribution("settlementSuitability", landFieldValues(result.Classes, result.Diagnostics.Suitability))
		printFieldDistribution("settlementAccess", landFieldValues(result.Classes, result.Diagnostics.AccessScore))
		printFieldDistribution("settlementWater", landFieldValues(result.Classes, result.Diagnostics.WaterScore))
	}
	type settlementCount struct {
		class climgen.SettlementClass
		count int
	}
	var sorted []settlementCount
	for class, count := range counts {
		sorted = append(sorted, settlementCount{class: class, count: count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
	fmt.Println("    settlement summary:")
	for _, entry := range sorted {
		fmt.Printf("      settlement[%s]=%d (%.1f%%)\n",
			climgen.SettlementClassName(entry.class),
			entry.count,
			100*float64(entry.count)/float64(landCells),
		)
	}
}

func landFieldValues(classes []climgen.SettlementClass, values []float64) []float64 {
	out := make([]float64, 0, len(values))
	for i, class := range classes {
		if class == climgen.SettlementOcean || i >= len(values) {
			continue
		}
		out = append(out, values[i])
	}
	return out
}

func printFieldDistribution(label string, values []float64) {
	if len(values) == 0 {
		return
	}
	fmt.Printf(
		"      %sDist: mean=%.3f p90=%.3f p95=%.3f p99=%.3f max=%.3f ge42=%.1f%% ge52=%.1f%% ge55=%.1f%% ge56=%.1f%% ge58=%.1f%% ge64=%.1f%% ge66=%.1f%%\n",
		label,
		meanFloat64(values),
		percentileFloat64(values, 0.90),
		percentileFloat64(values, 0.95),
		percentileFloat64(values, 0.99),
		percentileFloat64(values, 1.00),
		100*fieldShareAtLeast(values, 0.42),
		100*fieldShareAtLeast(values, 0.52),
		100*fieldShareAtLeast(values, 0.55),
		100*fieldShareAtLeast(values, 0.56),
		100*fieldShareAtLeast(values, 0.58),
		100*fieldShareAtLeast(values, 0.64),
		100*fieldShareAtLeast(values, 0.66),
	)
}

func fieldShareAtLeast(values []float64, threshold float64) float64 {
	if len(values) == 0 {
		return 0
	}
	count := 0
	for _, value := range values {
		if value >= threshold {
			count++
		}
	}
	return float64(count) / float64(len(values))
}

func printSettlementPreferenceSummary(results []*climgen.SettlementPreferenceResult) {
	if len(results) == 0 {
		return
	}
	fmt.Println("    settlement preferences:")
	for _, result := range results {
		landCells := 0
		favorable := 0
		prime := 0
		for _, class := range result.Classes {
			if class == climgen.SettlementOcean {
				continue
			}
			landCells++
			if class == climgen.SettlementFavorable || class == climgen.SettlementPrime {
				favorable++
			}
			if class == climgen.SettlementPrime {
				prime++
			}
		}
		if landCells == 0 {
			continue
		}
		fmt.Printf("      pref[%s]: favorable=%.1f%% prime=%.1f%%\n",
			result.Profile.Name,
			100*float64(favorable)/float64(landCells),
			100*float64(prime)/float64(landCells),
		)
	}
}

// printPrecipitationStrataSummary reports annual precipitation split by
// continentality, absolute latitude and elevation.
//
// Global percentiles hide compensating errors between strata: a measured
// coastal band running +58% against an interior running -30% nets out to a
// tame-looking -17% global median, and a whole biome class can swap for another
// with nothing visible in the aggregate at all. Every cross-level comparison
// that reads only global aggregates shares that blind spot, so the strata are
// reported alongside them.
func printPrecipitationStrataSummary(
	sites []climgen.Vector3D,
	cells []climgen.VoronoiCell,
	elevation []float64,
	seaLevel float64,
	biomes *climgen.BiomeResult,
) {
	if biomes == nil || biomes.Diagnostics == nil || len(elevation) == 0 {
		return
	}
	precip := biomes.Diagnostics.AnnualPrecipCm
	if len(precip) == 0 {
		return
	}
	adj := climgen.BuildFlatAdjacency(cells)
	interior := climgen.ComputeSurfaceInteriorFraction(elevation, seaLevel, adj, 1800.0, true)

	type stratum struct {
		name string
		vals []float64
	}
	strata := []stratum{
		{name: "cont:coastal"}, {name: "cont:middle"}, {name: "cont:interior"},
		{name: "lat:00-30"}, {name: "lat:30-60"}, {name: "lat:60-90"},
		{name: "elev:0-500"}, {name: "elev:500-1500"}, {name: "elev:1500+"},
	}
	add := func(i int, v float64) { strata[i].vals = append(strata[i].vals, v) }
	for i := range elevation {
		if elevation[i] < seaLevel || i >= len(precip) {
			continue
		}
		v := precip[i]
		switch f := interiorAt(interior, i); {
		case f < 0.2:
			add(0, v)
		case f < 0.6:
			add(1, v)
		default:
			add(2, v)
		}
		absLat := math.Abs(latitudeDegOf(sites, i))
		switch {
		case absLat < 30:
			add(3, v)
		case absLat < 60:
			add(4, v)
		default:
			add(5, v)
		}
		switch e := elevation[i]; {
		case e < 500:
			add(6, v)
		case e < 1500:
			add(7, v)
		default:
			add(8, v)
		}
	}
	for _, s := range strata {
		if len(s.vals) == 0 {
			continue
		}
		sort.Float64s(s.vals)
		q := func(p float64) float64 {
			idx := int(p * float64(len(s.vals)-1))
			return s.vals[idx]
		}
		mean := 0.0
		for _, v := range s.vals {
			mean += v
		}
		mean /= float64(len(s.vals))
		fmt.Printf("      precipStrata[%s]: n=%d p10=%.1f p50=%.1f p90=%.1f mean=%.1f\n",
			s.name, len(s.vals), q(0.10), q(0.50), q(0.90), mean)
	}
}

func interiorAt(interior []float64, i int) float64 {
	if i < 0 || i >= len(interior) {
		return 0
	}
	return interior[i]
}

func latitudeDegOf(sites []climgen.Vector3D, i int) float64 {
	if i < 0 || i >= len(sites) {
		return 0
	}
	return math.Asin(math.Max(-1, math.Min(1, sites[i].Z))) * 180.0 / math.Pi
}
