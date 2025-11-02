package main

// test_fine_tuning.go - Fine-tune around best configuration (45 cells, β=0.10)

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== FINE TUNING BEST CONFIGURATION ===")
	fmt.Println()
	fmt.Println("Base: 45 cells, cell_β=0.10")
	fmt.Println("Testing budget exponents to increase micro plates")
	fmt.Println()

	// Generate icosphere once
	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\\n\\n", len(icosphereSites))

	// Test budget exponents
	budgetExps := []float64{4.0, 4.5, 5.0, 5.5, 6.0, 6.5, 7.0}

	fmt.Println("Budget Exp | Plates (Maj/Min/Mic) | Score | Gini | β | Ratio")
	fmt.Println("-----------|----------------------|-------|------|---|-------")

	bestScore := 0.0
	bestBudgetExp := 0.0
	var bestMetrics evaluation.DetailedMetrics

	for _, budgetExp := range budgetExps {
		settings := tectonics.DefaultConvectionSettings()
		settings.PowerLawExponent = 0.10
		settings.NumConvectionCells = 45
		settings.GrowthIterations = 300
		settings.TargetPlateCount = 55
		settings.DivergenceThreshold = 0.2
		settings.MinBoundaryDistance = 300.0
		settings.MinCellStrength = 0.01
		settings.MaxCellStrength = 1.0
		settings.GrowthBudgetExponent = budgetExp
		settings.Seed = 12345
		settings.Verbose = false

		plates, _, err := tectonics.GeneratePlatesFromMantleConvection(
			voronoiCells,
			icosphereSites,
			planetRadius,
			settings,
		)

		if err != nil {
			fmt.Printf("Error at %.1f: %v\\n", budgetExp, err)
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
			bestBudgetExp = budgetExp
			bestMetrics = metrics
		}

		// Print compact results
		fmt.Printf("%.1f        | %2d (%d/%d/%d)          | %.1f%% | %.3f | %.3f | %.0fx\\n",
			budgetExp,
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
	fmt.Println("=== BEST RESULT ===")
	fmt.Printf("Budget exponent: %.1f\\n", bestBudgetExp)
	fmt.Printf("Overall score: %.1f%% (baseline: 49.4%%, improvement: +%.1f%%)\\n",
		bestScore*100, (bestScore-0.494)*100)
	fmt.Printf("Plates: %d total\\n", bestMetrics.SizeDistribution.MajorCount+
		bestMetrics.SizeDistribution.MinorCount+
		bestMetrics.SizeDistribution.MicroCount)
	fmt.Printf("  Major: %d (target: 7)\\n", bestMetrics.SizeDistribution.MajorCount)
	fmt.Printf("  Minor: %d (target: 13)\\n", bestMetrics.SizeDistribution.MinorCount)
	fmt.Printf("  Micro: %d (target: 19)\\n", bestMetrics.SizeDistribution.MicroCount)
	fmt.Printf("Gini: %.3f (target: 0.811)\\n", bestMetrics.SizeDistribution.GiniCoefficient)
	fmt.Printf("Power law β: %.3f (target: 0.390)\\n", bestMetrics.PowerLaw.Exponent)
	fmt.Printf("Size ratio: %.0fx (target: >1000x)\\n", bestMetrics.SizeDistribution.SizeRatio)
	fmt.Println()

	// Component breakdown
	fmt.Println("Component Scores:")
	fmt.Printf("  Plate Count: %.1f%%\\n", bestMetrics.Benchmark.PlateCountScore*100)
	fmt.Printf("  Power Law: %.1f%%\\n", bestMetrics.Benchmark.PowerLawScore*100)
	fmt.Printf("  Size Variation: %.1f%%\\n", bestMetrics.Benchmark.SizeVariationScore*100)
	fmt.Printf("  Boundary Quality: %.1f%%\\n", bestMetrics.Benchmark.BoundaryScore*100)
	fmt.Printf("  Spatial Distribution: %.1f%%\\n", bestMetrics.Benchmark.SpatialScore*100)
	fmt.Printf("  Continental Ratio: %.1f%%\\n", bestMetrics.Benchmark.ContinentalScore*100)
}
