package main

// test_original_beta.go - Test with original β=0.39 for cells

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== TESTING WITH ORIGINAL CELL β=0.39 ===")
	fmt.Println()
	fmt.Println("Hypothesis: Using Earth's actual β for cells might work better")
	fmt.Println("Testing various cell counts and budget exponents")
	fmt.Println()

	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\\n\\n", len(icosphereSites))

	type config struct {
		cells     int
		budgetExp float64
	}

	configs := []config{
		{45, 2.0},
		{45, 2.5},
		{45, 3.0},
		{50, 2.0},
		{50, 2.5},
		{50, 3.0},
		{55, 2.0},
		{55, 2.5},
		{55, 3.0},
		{60, 2.0},
		{60, 2.5},
	}

	fmt.Println("Cells | Budget | Plates (Maj/Min/Mic) | Score | Gini | β | Ratio")
	fmt.Println("------|--------|----------------------|-------|------|---|-------")

	bestScore := 0.0
	var bestConfig config
	var bestMetrics evaluation.DetailedMetrics

	for _, cfg := range configs {
		settings := tectonics.DefaultConvectionSettings()
		settings.PowerLawExponent = 0.39  // Earth's actual value
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
			fmt.Printf("%2d    | %.1f    | Error\\n", cfg.cells, cfg.budgetExp)
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

		fmt.Printf("%2d    | %.1f    | %2d (%d/%d/%d)          | %.1f%% | %.3f | %.3f | %.0fx\\n",
			cfg.cells, cfg.budgetExp, len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount,
			metrics.Benchmark.OverallScore*100,
			metrics.SizeDistribution.GiniCoefficient,
			metrics.PowerLaw.Exponent,
			metrics.SizeDistribution.SizeRatio)
	}

	fmt.Println()
	fmt.Println("=== BEST WITH β=0.39 ===")
	fmt.Printf("Cells: %d, Budget: %.1f → Score: %.1f%%\\n",
		bestConfig.cells, bestConfig.budgetExp, bestScore*100)
	fmt.Printf("Distribution: %d/%d/%d (maj/min/mic)\\n",
		bestMetrics.SizeDistribution.MajorCount,
		bestMetrics.SizeDistribution.MinorCount,
		bestMetrics.SizeDistribution.MicroCount)
	fmt.Printf("Plate β: %.3f (target: 0.390)\\n", bestMetrics.PowerLaw.Exponent)
	fmt.Println()

	// Also try our previous best (45 cells, β=0.10, budget=4.5)
	fmt.Println("=== COMPARISON TO PREVIOUS BEST (45 cells, β=0.10, budget=4.5) ===")
	fmt.Println("Previous best: 56.4% score, 7/22/7 distribution")
	fmt.Printf("New best: %.1f%% score, %d/%d/%d distribution\\n",
		bestScore*100,
		bestMetrics.SizeDistribution.MajorCount,
		bestMetrics.SizeDistribution.MinorCount,
		bestMetrics.SizeDistribution.MicroCount)
}
