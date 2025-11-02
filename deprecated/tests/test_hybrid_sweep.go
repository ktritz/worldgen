package main

// test_hybrid_sweep.go - Sweep hybrid parameters to find optimal configuration

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== HYBRID METHOD PARAMETER SWEEP ===")
	fmt.Println()
	fmt.Println("Goal: Find configuration that achieves 75%+ with 7 major plates")
	fmt.Println()

	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\n\n", len(icosphereSites))

	type config struct {
		cells        int
		budgetExp    float64
		refinements  int
		description  string
	}

	configs := []config{
		{45, 5.5, 100, "Lower cells, high budget, many refinements"},
		{50, 5.0, 100, "Mid cells, mid-high budget, many refinements"},
		{55, 4.5, 100, "Mid-high cells, mid budget, many refinements"},
		{60, 5.5, 50, "High cells, high budget (current)"},
		{40, 6.0, 150, "Low cells, very high budget, very many refinements"},
		{50, 6.0, 100, "Mid cells, very high budget"},
	}

	fmt.Println("Config | Cells | Budget | Refine | Plates (Maj/Min/Mic) | Score | Major=7?")
	fmt.Println("-------|-------|--------|--------|----------------------|-------|----------")

	bestScore := 0.0
	var bestConfig config
	var bestMetrics evaluation.DetailedMetrics
	bestHas7Majors := false

	for i, cfg := range configs {
		settings := tectonics.DefaultHybridSettings()
		settings.Verbose = false
		settings.RefinementIterations = cfg.refinements

		settings.ConvectionSettings.NumConvectionCells = cfg.cells
		settings.ConvectionSettings.PowerLawExponent = 0.10
		settings.ConvectionSettings.GrowthBudgetExponent = cfg.budgetExp
		settings.ConvectionSettings.GrowthIterations = 200
		settings.ConvectionSettings.MinCellStrength = 0.01
		settings.ConvectionSettings.MaxCellStrength = 1.0
		settings.ConvectionSettings.DivergenceThreshold = 0.2
		settings.ConvectionSettings.MinBoundaryDistance = 300.0
		settings.ConvectionSettings.TargetPlateCount = cfg.cells + 20
		settings.ConvectionSettings.Seed = 12345

		plates, _, err := tectonics.HybridPlateGeneration(
			voronoiCells,
			icosphereSites,
			planetRadius,
			settings,
		)

		if err != nil {
			fmt.Printf("%d      | Error\n", i+1)
			continue
		}

		plateSizes := make([]float64, len(plates))
		plateTypes := make([]bool, len(plates))
		plateCentroids := make([][3]float64, len(plates))

		for j, plate := range plates {
			plateSizes[j] = plate.Area
			plateTypes[j] = (plate.PlateType == tectonics.ContinentalPlate)
			plateCentroids[j] = [3]float64{plate.Center.X, plate.Center.Y, plate.Center.Z}
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

		fmt.Printf("%d      | %2d    | %.1f    | %3d    | %2d (%d/%d/%d)          | %.1f%% | %s\n",
			i+1,
			cfg.cells,
			cfg.budgetExp,
			cfg.refinements,
			len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount,
			metrics.Benchmark.OverallScore*100,
			majorMarker)
	}

	fmt.Println()
	fmt.Println("=== BEST CONFIGURATION ===")
	fmt.Printf("Description: %s\n", bestConfig.description)
	fmt.Printf("Cells: %d, Budget: %.1f, Refinements: %d\n",
		bestConfig.cells, bestConfig.budgetExp, bestConfig.refinements)
	fmt.Printf("Score: %.1f%% (target: 75%%, gap: %.1f%%)\n",
		bestScore*100, 75.0-bestScore*100)
	fmt.Printf("Distribution: %d major, %d minor, %d micro\n",
		bestMetrics.SizeDistribution.MajorCount,
		bestMetrics.SizeDistribution.MinorCount,
		bestMetrics.SizeDistribution.MicroCount)
	fmt.Println()

	if bestHas7Majors {
		fmt.Println("✓ PERFECT major plate count!")
	}
	if bestScore >= 0.75 {
		fmt.Println("✓✓✓ TARGET ACHIEVED!")
	} else if bestScore >= 0.70 {
		fmt.Println("○ Very close to target")
	}
}
