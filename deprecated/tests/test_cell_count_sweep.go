package main

// test_cell_count_sweep.go - Test different convection cell counts

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== CONVECTION CELL COUNT SWEEP ===")
	fmt.Println()

	// Generate icosphere once
	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\\n\\n", len(icosphereSites))

	// Test different cell counts with budget exponent 4.0 (best from previous sweep)
	cellCounts := []int{30, 40, 50, 60, 70}

	fmt.Println("Testing convection cell counts (budget exponent = 4.0)...")
	fmt.Println()

	for _, numCells := range cellCounts {
		fmt.Printf("=== %d Convection Cells ===\\n", numCells)

		settings := tectonics.DefaultConvectionSettings()
		settings.PowerLawExponent = 0.39
		settings.NumConvectionCells = numCells
		settings.GrowthIterations = 300
		settings.TargetPlateCount = numCells + 10 // Allow more boundaries
		settings.DivergenceThreshold = 0.2
		settings.MinBoundaryDistance = 300.0
		settings.MinCellStrength = 0.01
		settings.MaxCellStrength = 1.0
		settings.GrowthBudgetExponent = 4.0 // Best from previous sweep
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
		fmt.Printf("  Total plates: %d (target: 39)\\n", len(plates))
		fmt.Printf("  Distribution: %d major, %d minor, %d micro\\n",
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount)
		fmt.Printf("  Score: %.1f%% | Gini: %.3f | β: %.3f | Ratio: %.0fx\\n",
			metrics.Benchmark.OverallScore*100,
			metrics.SizeDistribution.GiniCoefficient,
			metrics.PowerLaw.Exponent,
			metrics.SizeDistribution.SizeRatio)
		fmt.Println()
	}

	fmt.Println("=== SWEEP COMPLETE ===")
	fmt.Println()
	fmt.Println("Earth Target: 39 total (7 major, 13 minor, 19 micro)")
	fmt.Println("Best baseline: 49.4%")
}
