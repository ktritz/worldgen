package main

// test_final_optimization.go - Final optimization with more cells + higher budget exp

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== FINAL OPTIMIZATION ===")
	fmt.Println()
	fmt.Println("Goal: Maintain 7 major plates while increasing micro plates")
	fmt.Println("Strategy: More cells + higher budget exponent")
	fmt.Println()

	// Generate icosphere once
	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\\n\\n", len(icosphereSites))

	// Test promising combinations
	type config struct {
		cells     int
		budgetExp float64
	}

	configs := []config{
		{50, 5.0},
		{50, 5.5},
		{50, 6.0},
		{52, 5.5},
		{52, 6.0},
		{55, 5.0},
		{55, 5.5},
		{55, 6.0},
		{48, 5.0},
		{48, 5.5},
	}

	fmt.Println("Cells | Budget | Plates (Maj/Min/Mic) | Score | Gini | β | Ratio")
	fmt.Println("------|--------|----------------------|-------|------|---|-------")

	bestScore := 0.0
	var bestConfig config
	var bestMetrics evaluation.DetailedMetrics

	for _, cfg := range configs {
		settings := tectonics.DefaultConvectionSettings()
		settings.PowerLawExponent = 0.10
		settings.NumConvectionCells = cfg.cells
		settings.GrowthIterations = 300
		settings.TargetPlateCount = cfg.cells + 10
		settings.DivergenceThreshold = 0.2
		settings.MinBoundaryDistance = 300.0
		settings.MinCellStrength = 0.01
		settings.MaxCellStrength = 1.0
		settings.GrowthBudgetExponent = cfg.budgetExp
		settings.Seed = 12345
		settings.Verbose = false

		plates, _, err := tectonics.GeneratePlatesFromMantleConvection(
			voronoiCells,
			icosphereSites,
			planetRadius,
			settings,
		)

		if err != nil {
			fmt.Printf("%2d    | %.1f    | Error: %v\\n", cfg.cells, cfg.budgetExp, err)
			continue
		}

		// Evaluation
		plateSizes := make([]float64, len(plates))
		plateTypes := make([]bool, len(plates))
		plateCentroids := make([][3]float64, len(plates))

		for i, plate := range plates {
			plateSizes[i] = plate.Area
			plateTypes[i] = (plate.PlateType == tectonics.ContinentalPlate)
			plateCentroids[i] = [3]float64{plate.Center.X, plate.Center.Y, plate.Center.Z}
		}

		sphereArea := 4.0 * math.Pi * planetRadius * planetRadius

		metrics := evaluation.CalculateDetailedMetrics(
			plateSizes,
			plateTypes,
			plateCentroids,
			sphereArea,
			planetRadius,
		)

		// Track best
		if metrics.Benchmark.OverallScore > bestScore {
			bestScore = metrics.Benchmark.OverallScore
			bestConfig = cfg
			bestMetrics = metrics
		}

		// Print results
		fmt.Printf("%2d    | %.1f    | %2d (%d/%d/%d)          | %.1f%% | %.3f | %.3f | %.0fx\\n",
			cfg.cells,
			cfg.budgetExp,
			len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount,
			metrics.Benchmark.OverallScore*100,
			metrics.SizeDistribution.GiniCoefficient,
			metrics.PowerLaw.Exponent,
			metrics.SizeDistribution.SizeRatio)
	}

	fmt.Println()
	fmt.Println("=== BEST CONFIGURATION ===")
	fmt.Printf("Cells: %d, Budget exponent: %.1f\\n", bestConfig.cells, bestConfig.budgetExp)
	fmt.Printf("Score: %.1f%% (baseline: 49.4%%, improvement: +%.1f%%)\\n",
		bestScore*100, (bestScore-0.494)*100)
	fmt.Println()
	fmt.Printf("Distribution: %d major, %d minor, %d micro (total: %d)\\n",
		bestMetrics.SizeDistribution.MajorCount,
		bestMetrics.SizeDistribution.MinorCount,
		bestMetrics.SizeDistribution.MicroCount,
		bestMetrics.SizeDistribution.MajorCount+
			bestMetrics.SizeDistribution.MinorCount+
			bestMetrics.SizeDistribution.MicroCount)
	fmt.Printf("Earth target: 7 major, 13 minor, 19 micro (total: 39)\\n")
	fmt.Println()
	fmt.Printf("Gini: %.3f (target: 0.811, Δ%.3f)\\n",
		bestMetrics.SizeDistribution.GiniCoefficient,
		bestMetrics.SizeDistribution.GiniCoefficient-0.811)
	fmt.Printf("Power law β: %.3f (target: 0.390, Δ+%.3f)\\n",
		bestMetrics.PowerLaw.Exponent,
		bestMetrics.PowerLaw.Exponent-0.390)
	fmt.Printf("Size ratio: %.0fx (target: >1000x)\\n", bestMetrics.SizeDistribution.SizeRatio)
	fmt.Println()
	fmt.Println("Component Scores:")
	fmt.Printf("  Plate Count: %.1f%%\\n", bestMetrics.Benchmark.PlateCountScore*100)
	fmt.Printf("  Power Law: %.1f%%\\n", bestMetrics.Benchmark.PowerLawScore*100)
	fmt.Printf("  Size Variation: %.1f%%\\n", bestMetrics.Benchmark.SizeVariationScore*100)
	fmt.Printf("  Boundary Quality: %.1f%%\\n", bestMetrics.Benchmark.BoundaryScore*100)
	fmt.Printf("  Spatial: %.1f%%\\n", bestMetrics.Benchmark.SpatialScore*100)
	fmt.Printf("  Continental: %.1f%%\\n", bestMetrics.Benchmark.ContinentalScore*100)
}
