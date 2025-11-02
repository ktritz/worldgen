package main

// test_aggressive_params.go - Test very aggressive parameters for extreme differentiation

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== AGGRESSIVE PARAMETER TESTING ===")
	fmt.Println()
	fmt.Println("Strategy: Higher budget exponents + more cells for extreme differentiation")
	fmt.Println()

	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\n\n", len(icosphereSites))

	type config struct {
		cells      int
		cellBeta   float64
		budgetExp  float64
	}

	// Test very aggressive settings
	configs := []config{
		// Keep original cell beta (0.10), try higher budget exponents
		{45, 0.10, 5.0},
		{45, 0.10, 5.5},
		{45, 0.10, 6.0},
		{50, 0.10, 5.0},
		{50, 0.10, 5.5},
		{50, 0.10, 6.0},
		{55, 0.10, 5.0},
		{55, 0.10, 5.5},
		{55, 0.10, 6.0},
		{60, 0.10, 5.0},
		{60, 0.10, 5.5},
		{60, 0.10, 6.0},
	}

	fmt.Println("Cells | β | Budget | Plates (Maj/Min/Mic) | Score | Power β | Gini | Ratio")
	fmt.Println("------|---|--------|----------------------|-------|---------|------|-------")

	bestScore := 0.0
	var bestConfig config
	var bestMetrics evaluation.DetailedMetrics

	for _, cfg := range configs {
		settings := tectonics.DefaultConvectionSettings()
		settings.PowerLawExponent = cfg.cellBeta
		settings.NumConvectionCells = cfg.cells
		settings.GrowthIterations = 300
		settings.TargetPlateCount = cfg.cells + 15
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
			fmt.Printf("%2d    |%.2f| %.1f    | Error\n", cfg.cells, cfg.cellBeta, cfg.budgetExp)
			continue
		}

		plateSizes := make([]float64, len(plates))
		plateTypes := make([]bool, len(plates))
		plateCentroids := make([][3]float64, len(plates))

		for i, plate := range plates {
			plateSizes[i] = plate.Area
			plateTypes[i] = (plate.PlateType == tectonics.ContinentalPlate)
			plateCentroids[i] = [3]float64{plate.Center.X, plate.Center.Y, plate.Center.Z}
		}

		sphereArea := 4.0 * math.Pi * planetRadius * planetRadius
		metrics := evaluation.CalculateDetailedMetrics(plateSizes, plateTypes, plateCentroids, sphereArea, planetRadius)

		if metrics.Benchmark.OverallScore > bestScore {
			bestScore = metrics.Benchmark.OverallScore
			bestConfig = cfg
			bestMetrics = metrics
		}

		fmt.Printf("%2d    |%.2f| %.1f    | %2d (%d/%d/%d)          | %.1f%% | %.3f   | %.3f | %.0fx\n",
			cfg.cells, cfg.cellBeta, cfg.budgetExp, len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount,
			metrics.Benchmark.OverallScore*100,
			metrics.PowerLaw.Exponent,
			metrics.SizeDistribution.GiniCoefficient,
			metrics.SizeDistribution.SizeRatio)
	}

	fmt.Println()
	fmt.Println("=== BEST CONFIGURATION ===")
	fmt.Printf("Cells: %d, Cell β: %.2f, Budget: %.1f\n", bestConfig.cells, bestConfig.cellBeta, bestConfig.budgetExp)
	fmt.Printf("Score: %.1f%% (baseline: 49.4%%, phase2: 55.8%%)\n", bestScore*100)
	fmt.Printf("Distribution: %d major, %d minor, %d micro\n",
		bestMetrics.SizeDistribution.MajorCount,
		bestMetrics.SizeDistribution.MinorCount,
		bestMetrics.SizeDistribution.MicroCount)
	fmt.Printf("Power law β: %.3f (target: 0.390)\n", bestMetrics.PowerLaw.Exponent)
	fmt.Printf("Gini: %.3f (target: 0.811)\n", bestMetrics.SizeDistribution.GiniCoefficient)
	fmt.Println()

	if bestScore > 0.558 {
		fmt.Println("✓ IMPROVEMENT over Phase 2!")
	} else if bestScore > 0.494 {
		fmt.Println("○ Better than baseline but not Phase 2")
	} else {
		fmt.Println("△ Worse than baseline")
	}
}
