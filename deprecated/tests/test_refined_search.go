package main

// test_refined_search.go - Refined search for 7 major plates + more micros

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== REFINED SEARCH: Targeting 7 Major Plates ===")
	fmt.Println()
	fmt.Println("Goal: Keep 7 majors from Phase 2, increase micros")
	fmt.Println()

	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\n\n", len(icosphereSites))

	type config struct {
		cells     int
		budgetExp float64
	}

	// Focus on configurations near Phase 2's best (45 cells, 4.5 budget)
	configs := []config{
		// Around 45 cells
		{45, 4.3},
		{45, 4.5},  // Phase 2 best
		{45, 4.7},
		{46, 4.5},
		{47, 4.5},
		{48, 4.5},
		// Around 50 cells
		{48, 4.7},
		{48, 5.0},
		{50, 4.5},
		{50, 4.7},
		{52, 4.5},
	}

	fmt.Println("Cells | Budget | Plates (Maj/Min/Mic) | Score | Major? | Micros | Power β")
	fmt.Println("------|--------|----------------------|-------|--------|--------|--------")

	bestScore := 0.0
	var bestConfig config
	var bestMetrics evaluation.DetailedMetrics
	bestHas7Majors := false

	for _, cfg := range configs {
		settings := tectonics.DefaultConvectionSettings()
		settings.PowerLawExponent = 0.10
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
			fmt.Printf("%2d    | %.1f    | Error\n", cfg.cells, cfg.budgetExp)
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

		has7Majors := (metrics.SizeDistribution.MajorCount == 7)
		majorMarker := " "
		if has7Majors {
			majorMarker = "✓"
		}

		// Prefer configurations with 7 majors, then by score
		isBetter := false
		if has7Majors && !bestHas7Majors {
			isBetter = true
		} else if has7Majors == bestHas7Majors && metrics.Benchmark.OverallScore > bestScore {
			isBetter = true
		}

		if isBetter {
			bestScore = metrics.Benchmark.OverallScore
			bestConfig = cfg
			bestMetrics = metrics
			bestHas7Majors = has7Majors
		}

		fmt.Printf("%2d    | %.1f    | %2d (%d/%d/%d)          | %.1f%% | %s      | %2d     | %.3f\n",
			cfg.cells, cfg.budgetExp, len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount,
			metrics.Benchmark.OverallScore*100,
			majorMarker,
			metrics.SizeDistribution.MicroCount,
			metrics.PowerLaw.Exponent)
	}

	fmt.Println()
	fmt.Println("=== BEST CONFIGURATION ===")
	fmt.Printf("Cells: %d, Budget: %.1f\n", bestConfig.cells, bestConfig.budgetExp)
	fmt.Printf("Score: %.1f%% (baseline: 49.4%%, phase2: 55.8%%)\n", bestScore*100)
	fmt.Printf("Distribution: %d major, %d minor, %d micro\n",
		bestMetrics.SizeDistribution.MajorCount,
		bestMetrics.SizeDistribution.MinorCount,
		bestMetrics.SizeDistribution.MicroCount)
	fmt.Printf("Target: 7 major, 13 minor, 19 micro\n")
	fmt.Printf("Power law β: %.3f (target: 0.390)\n", bestMetrics.PowerLaw.Exponent)
	fmt.Printf("Gini: %.3f (target: 0.811)\n", bestMetrics.SizeDistribution.GiniCoefficient)
	fmt.Println()

	if bestHas7Majors {
		fmt.Println("✓ PERFECT major plate count!")
	}
	if bestScore > 0.558 {
		fmt.Println("✓ Improved over Phase 2")
	}
}
