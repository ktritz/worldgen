package main

// test_combined_optimization.go - Test cell exponent + budget exponent combinations

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== COMBINED PARAMETER OPTIMIZATION ===")
	fmt.Println()
	fmt.Println("Theory: plate_β ≈ cell_β × budget_exponent")
	fmt.Println("Target: plate_β = 0.39")
	fmt.Println()

	// Generate icosphere once
	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\\n\\n", len(icosphereSites))

	// Test combinations that should give β ≈ 0.39
	type testConfig struct {
		cells        int
		cellExponent float64
		budgetExp    float64
		expectedBeta float64
	}

	configs := []testConfig{
		{45, 0.10, 4.0, 0.40},  // 0.10 * 4.0 = 0.40
		{50, 0.10, 4.0, 0.40},
		{55, 0.10, 4.0, 0.40},
		{50, 0.08, 5.0, 0.40},  // 0.08 * 5.0 = 0.40
		{55, 0.13, 3.0, 0.39},  // 0.13 * 3.0 = 0.39
		{50, 0.13, 3.0, 0.39},
	}

	fmt.Println("Testing parameter combinations...")
	fmt.Println()

	bestScore := 0.0
	var bestConfig testConfig
	var bestMetrics evaluation.DetailedMetrics

	for _, config := range configs {
		fmt.Printf("=== %d cells, cell_β=%.2f, budget_exp=%.1f (expect plate_β≈%.2f) ===\\n",
			config.cells, config.cellExponent, config.budgetExp, config.expectedBeta)

		settings := tectonics.DefaultConvectionSettings()
		settings.PowerLawExponent = config.cellExponent
		settings.NumConvectionCells = config.cells
		settings.GrowthIterations = 300
		settings.TargetPlateCount = config.cells + 10
		settings.DivergenceThreshold = 0.2
		settings.MinBoundaryDistance = 300.0
		settings.MinCellStrength = 0.01
		settings.MaxCellStrength = 1.0
		settings.GrowthBudgetExponent = config.budgetExp
		settings.Seed = 12345
		settings.Verbose = false

		plates, _, err := tectonics.GeneratePlatesFromMantleConvection(
			voronoiCells,
			icosphereSites,
			planetRadius,
			settings,
		)

		if err != nil {
			fmt.Printf("Error: %v\\n\\n", err)
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

		// Print results
		fmt.Printf("  Plates: %d (%d/%d/%d maj/min/mic) | Target: 39 (7/13/19)\\n",
			len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount)
		fmt.Printf("  Score: %.1f%% | Gini: %.3f | β: %.3f (Δ%.3f) | Ratio: %.0fx\\n",
			metrics.Benchmark.OverallScore*100,
			metrics.SizeDistribution.GiniCoefficient,
			metrics.PowerLaw.Exponent,
			metrics.PowerLaw.Exponent-0.39,
			metrics.SizeDistribution.SizeRatio)

		if metrics.Benchmark.OverallScore > bestScore {
			bestScore = metrics.Benchmark.OverallScore
			bestConfig = config
			bestMetrics = metrics
		}

		fmt.Println()
	}

	fmt.Println("=== BEST CONFIGURATION ===")
	fmt.Printf("Cells: %d, Cell β: %.2f, Budget exp: %.1f\\n",
		bestConfig.cells, bestConfig.cellExponent, bestConfig.budgetExp)
	fmt.Printf("Score: %.1f%% (baseline: 49.4%%)\\n", bestScore*100)
	fmt.Printf("Plates: %d major, %d minor, %d micro\\n",
		bestMetrics.SizeDistribution.MajorCount,
		bestMetrics.SizeDistribution.MinorCount,
		bestMetrics.SizeDistribution.MicroCount)
	fmt.Printf("β: %.3f | Gini: %.3f\\n",
		bestMetrics.PowerLaw.Exponent,
		bestMetrics.SizeDistribution.GiniCoefficient)
}
