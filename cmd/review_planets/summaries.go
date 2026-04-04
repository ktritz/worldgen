package main

import (
	"fmt"
	"math"
	"sort"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

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
