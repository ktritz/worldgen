package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"worldgen/climgen"
)

type baselineSeedMetrics struct {
	Seed        int64
	Score       float64
	Drain       float64
	Endo        float64
	Arid        float64
	Forest      float64
	Wetland     float64
	Woody       float64
	Crop        float64
	Pasture     float64
	Game        float64
	Timber      float64
	Fishery     float64
	Shellfish   float64
	Reliable    float64
	Groundwater float64
	Metallic    float64
	Fuel        float64
	Luxury      float64
	Favorable   float64
	Prime       float64
	Frontier    float64
	Settled     float64
	DensePop    float64
	UrbanPop    float64
}

func newBaselineSeedMetrics(seed int64) baselineSeedMetrics {
	na := math.NaN()
	return baselineSeedMetrics{
		Seed:        seed,
		Score:       na,
		Drain:       na,
		Endo:        na,
		Arid:        na,
		Forest:      na,
		Wetland:     na,
		Woody:       na,
		Crop:        na,
		Pasture:     na,
		Game:        na,
		Timber:      na,
		Fishery:     na,
		Shellfish:   na,
		Reliable:    na,
		Groundwater: na,
		Metallic:    na,
		Fuel:        na,
		Luxury:      na,
		Favorable:   na,
		Prime:       na,
		Frontier:    na,
		Settled:     na,
		DensePop:    na,
		UrbanPop:    na,
	}
}

func collectBiomeMetrics(result *climgen.BiomeResult) (aridPct, forestPct, wetlandPct float64) {
	if result == nil {
		return math.NaN(), math.NaN(), math.NaN()
	}
	stats := climgen.GetBiomeStats(result)
	landCells := 0
	for _, biome := range result.Biomes {
		if biome != climgen.BiomeOcean {
			landCells++
		}
	}
	if landCells == 0 {
		return math.NaN(), math.NaN(), math.NaN()
	}
	aridCount := stats[climgen.BiomeDesertHot] + stats[climgen.BiomeDesertCold] + stats[climgen.BiomeSemiArid]
	forestCount := stats[climgen.BiomeTemperateRainforest] +
		stats[climgen.BiomeTemperateForest] +
		stats[climgen.BiomeBorealForest] +
		stats[climgen.BiomeTropicalSeasonalForest] +
		stats[climgen.BiomeTropicalRainforest]
	wetlandCount := stats[climgen.BiomeWetland]
	return 100 * float64(aridCount) / float64(landCells),
		100 * float64(forestCount) / float64(landCells),
		100 * float64(wetlandCount) / float64(landCells)
}

func collectVegetationMetrics(result *climgen.VegetationResult) float64 {
	if result == nil {
		return math.NaN()
	}
	landCells := 0
	woodyCount := 0
	for _, veg := range result.Types {
		if veg == climgen.VegetationOcean {
			continue
		}
		landCells++
		switch veg {
		case climgen.VegetationWoodland,
			climgen.VegetationForest,
			climgen.VegetationRainforest,
			climgen.VegetationRiparianForest,
			climgen.VegetationCloudForest,
			climgen.VegetationMangrove:
			woodyCount++
		}
	}
	if landCells == 0 {
		return math.NaN()
	}
	return 100 * float64(woodyCount) / float64(landCells)
}

func collectAgricultureMetrics(result *climgen.AgricultureResult) (cropPct, pasturePct float64) {
	if result == nil {
		return math.NaN(), math.NaN()
	}
	landCells := 0
	cropCount := 0
	pastureCount := 0
	for _, ag := range result.Types {
		if ag == climgen.AgricultureOcean {
			continue
		}
		landCells++
		switch ag {
		case climgen.AgricultureDryFarming,
			climgen.AgricultureMixedFarming,
			climgen.AgricultureIntensiveCropland,
			climgen.AgricultureFloodplainCropland:
			cropCount++
		case climgen.AgriculturePastoral:
			pastureCount++
		}
	}
	if landCells == 0 {
		return math.NaN(), math.NaN()
	}
	return 100 * float64(cropCount) / float64(landCells),
		100 * float64(pastureCount) / float64(landCells)
}

func collectWildlifeMetrics(result *climgen.WildlifeResult) (gamePct, timberPct float64) {
	if result == nil {
		return math.NaN(), math.NaN()
	}
	landCells := 0
	gameCount := 0
	timberCount := 0
	for _, w := range result.Types {
		if w == climgen.WildlifeOcean {
			continue
		}
		landCells++
		switch w {
		case climgen.WildlifeGrazingGame, climgen.WildlifeForestGame, climgen.WildlifeWetlandGame:
			gameCount++
		case climgen.WildlifeTimber:
			timberCount++
		}
	}
	if landCells == 0 {
		return math.NaN(), math.NaN()
	}
	return 100 * float64(gameCount) / float64(landCells),
		100 * float64(timberCount) / float64(landCells)
}

func collectWaterMetrics(result *climgen.WaterResourceResult) (reliablePct, groundwaterPct float64) {
	if result == nil {
		return math.NaN(), math.NaN()
	}
	landCells := 0
	reliableCount := 0
	groundwaterCount := 0
	for _, water := range result.Types {
		if water == climgen.WaterResourceOcean {
			continue
		}
		landCells++
		if water == climgen.WaterResourceReliableSurface {
			reliableCount++
		}
		if water == climgen.WaterResourceGroundwater {
			groundwaterCount++
		}
	}
	if landCells == 0 {
		return math.NaN(), math.NaN()
	}
	return 100 * float64(reliableCount) / float64(landCells),
		100 * float64(groundwaterCount) / float64(landCells)
}

func collectCoastalMetrics(result *climgen.CoastalResourceResult) (fisheryPct, shellfishPct float64) {
	if result == nil {
		return math.NaN(), math.NaN()
	}
	landCells := 0
	fisheryCount := 0
	shellfishCount := 0
	for _, typ := range result.Types {
		if typ == climgen.CoastalResourceOcean {
			continue
		}
		landCells++
		if typ == climgen.CoastalResourceOpenFishery || typ == climgen.CoastalResourceEstuarineFishery {
			fisheryCount++
		}
		if typ == climgen.CoastalResourceShellfish {
			shellfishCount++
		}
	}
	if landCells == 0 {
		return math.NaN(), math.NaN()
	}
	return 100 * float64(fisheryCount) / float64(landCells),
		100 * float64(shellfishCount) / float64(landCells)
}

func collectResourceMetrics(result *climgen.ResourceResult) (metallicPct, fuelPct, luxuryPct float64) {
	if result == nil {
		return math.NaN(), math.NaN(), math.NaN()
	}
	landCells := 0
	metallicCount := 0
	fuelCount := 0
	luxuryCount := 0
	for _, typ := range result.Types {
		if typ == climgen.ResourceOcean {
			continue
		}
		landCells++
		switch typ {
		case climgen.ResourceIronOre,
			climgen.ResourceCopperOre,
			climgen.ResourceLeadSilverOre,
			climgen.ResourceGoldOre:
			metallicCount++
		case climgen.ResourceCoal, climgen.ResourceOilGas:
			fuelCount++
		case climgen.ResourceGemstones:
			luxuryCount++
		}
	}
	if landCells == 0 {
		return math.NaN(), math.NaN(), math.NaN()
	}
	return 100 * float64(metallicCount) / float64(landCells),
		100 * float64(fuelCount) / float64(landCells),
		100 * float64(luxuryCount) / float64(landCells)
}

func collectSettlementMetrics(result *climgen.SettlementResult) (favorablePct, primePct float64) {
	if result == nil {
		return math.NaN(), math.NaN()
	}
	landCells := 0
	favorableCount := 0
	primeCount := 0
	for _, class := range result.Classes {
		if class == climgen.SettlementOcean {
			continue
		}
		landCells++
		if class == climgen.SettlementFavorable {
			favorableCount++
		}
		if class == climgen.SettlementPrime {
			primeCount++
		}
	}
	if landCells == 0 {
		return math.NaN(), math.NaN()
	}
	return 100 * float64(favorableCount) / float64(landCells),
		100 * float64(primeCount) / float64(landCells)
}

func collectPopulationMetrics(result *climgen.PopulationResult) (frontierPct, settledPct, densePct, urbanPct float64) {
	if result == nil {
		return math.NaN(), math.NaN(), math.NaN(), math.NaN()
	}
	landCells := 0
	frontierCount := 0
	settledCount := 0
	denseCount := 0
	urbanCount := 0
	for _, class := range result.Classes {
		if class == climgen.PopulationOcean {
			continue
		}
		landCells++
		if class == climgen.PopulationSparseFrontier {
			frontierCount++
		}
		if class == climgen.PopulationRural || class == climgen.PopulationDenseRural || class == climgen.PopulationUrban {
			settledCount++
		}
		if class == climgen.PopulationDenseRural || class == climgen.PopulationUrban {
			denseCount++
		}
		if class == climgen.PopulationUrban {
			urbanCount++
		}
	}
	if landCells == 0 {
		return math.NaN(), math.NaN(), math.NaN(), math.NaN()
	}
	return 100 * float64(frontierCount) / float64(landCells),
		100 * float64(settledCount) / float64(landCells),
		100 * float64(denseCount) / float64(landCells),
		100 * float64(urbanCount) / float64(landCells)
}

func writeBaselineReport(outputDir string, records []baselineSeedMetrics) error {
	if len(records) == 0 {
		return nil
	}
	path := filepath.Join(outputDir, "baseline_summary.txt")
	var b strings.Builder
	fmt.Fprintln(&b, "Baseline review summary")
	fmt.Fprintln(&b, "columns: seed score drain endo arid forest wetland woody crop pasture game timber fishery shellfish reliable groundwater metallic fuel luxury favorable prime frontierSupport settledSupport denseSupport urbanSupport")
	for _, r := range records {
		fmt.Fprintf(&b, "%5d %6s %6s %6s %6s %6s %7s %6s %6s %7s %6s %6s %7s %8s %8s %11s %8s %5s %7s %9s %5s %8s %7s %5s %5s\n",
			r.Seed,
			formatMetric(r.Score),
			formatMetric(r.Drain),
			formatMetric(r.Endo),
			formatMetric(r.Arid),
			formatMetric(r.Forest),
			formatMetric(r.Wetland),
			formatMetric(r.Woody),
			formatMetric(r.Crop),
			formatMetric(r.Pasture),
			formatMetric(r.Game),
			formatMetric(r.Timber),
			formatMetric(r.Fishery),
			formatMetric(r.Shellfish),
			formatMetric(r.Reliable),
			formatMetric(r.Groundwater),
			formatMetric(r.Metallic),
			formatMetric(r.Fuel),
			formatMetric(r.Luxury),
			formatMetric(r.Favorable),
			formatMetric(r.Prime),
			formatMetric(r.Frontier),
			formatMetric(r.Settled),
			formatMetric(r.DensePop),
			formatMetric(r.UrbanPop),
		)
	}

	metrics := []struct {
		name   string
		values []float64
	}{
		{"score", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.Score })},
		{"drain", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.Drain })},
		{"arid", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.Arid })},
		{"woody", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.Woody })},
		{"crop", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.Crop })},
		{"game", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.Game })},
		{"reliable", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.Reliable })},
		{"fishery", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.Fishery })},
		{"metallic", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.Metallic })},
		{"favorable", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.Favorable })},
		{"settledSupport", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.Settled })},
		{"denseSupport", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.DensePop })},
		{"urbanSupport", collectMetricSeries(records, func(r baselineSeedMetrics) float64 { return r.UrbanPop })},
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "ranges:")
	for _, metric := range metrics {
		min, max, ok := metricRange(metric.values)
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "  %s: %s .. %s\n", metric.name, formatMetric(min), formatMetric(max))
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func collectMetricSeries(records []baselineSeedMetrics, getter func(baselineSeedMetrics) float64) []float64 {
	values := make([]float64, 0, len(records))
	for _, record := range records {
		values = append(values, getter(record))
	}
	return values
}

func metricRange(values []float64) (float64, float64, bool) {
	min := math.Inf(1)
	max := math.Inf(-1)
	found := false
	for _, value := range values {
		if math.IsNaN(value) {
			continue
		}
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
		found = true
	}
	return min, max, found
}

func formatMetric(value float64) string {
	if math.IsNaN(value) {
		return "n/a"
	}
	return fmt.Sprintf("%.1f", value)
}
