package main

// test_budget_sweep.go - Test different growth budget exponents

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== GROWTH BUDGET EXPONENT SWEEP ===")
	fmt.Println()

	// Generate icosphere once
	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\\n\\n", len(icosphereSites))

	// Test different budget exponents
	exponents := []float64{2.0, 2.5, 3.0, 3.5, 4.0, 5.0}

	fmt.Println("Testing budget exponents...")
	fmt.Println()

	for _, exponent := range exponents {
		fmt.Printf("=== Budget Exponent: %.1f ===\\n", exponent)

		settings := tectonics.DefaultConvectionSettings()
		settings.PowerLawExponent = 0.39
		settings.NumConvectionCells = 40
		settings.GrowthIterations = 300
		settings.TargetPlateCount = 40
		settings.DivergenceThreshold = 0.2
		settings.MinBoundaryDistance = 300.0
		settings.MinCellStrength = 0.01
		settings.MaxCellStrength = 1.0
		settings.GrowthBudgetExponent = exponent
		settings.Seed = 12345
		settings.Verbose = false

		// Generate plates
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

		// Quick evaluation
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

		// Print summary
		fmt.Printf("  Plates: %d (Major: %d, Minor: %d, Micro: %d)\\n",
			len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount)
		fmt.Printf("  Score: %.1f%% | Gini: %.3f | β: %.3f | Size ratio: %.0fx\\n",
			metrics.Benchmark.OverallScore*100,
			metrics.SizeDistribution.GiniCoefficient,
			metrics.PowerLaw.Exponent,
			metrics.SizeDistribution.SizeRatio)
		fmt.Println()
	}

	fmt.Println("=== SWEEP COMPLETE ===")
	fmt.Println()
	fmt.Println("Earth Target: 7 major, 13 minor, 19 micro")
	fmt.Println("Earth Metrics: Score >75%, Gini 0.811, β 0.390")
}
