package main

// test_aggressive_hybrid_optimization.go - Aggressive parameter search for Method 7

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== AGGRESSIVE HYBRID OPTIMIZATION ===")
	fmt.Println()
	fmt.Println("Goal: Find configuration that achieves 75%+ score")
	fmt.Println("Strategy: Aggressive parameter sweep with focus on 7 major plates")
	fmt.Println()

	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\n\n", len(icosphereSites))

	type config struct {
		cells         int
		powerLaw      float64
		budgetExp     float64
		growthIter    int
		targetPlates  int
		refinements   int
		description   string
	}

	// Based on Phase 2 success: 55.8% with 7 majors came from:
	// cells=45, powerLaw=0.10, budgetExp=4.5, growthIter=300
	// Let's explore variations that might push past 75%

	configs := []config{
		// Phase 2 baseline (should reproduce 55.8%)
		{45, 0.10, 4.5, 300, 55, 100, "Phase 2 baseline"},

		// More aggressive major plate creation
		{40, 0.05, 5.0, 400, 50, 150, "Fewer cells, very low power, high budget"},
		{35, 0.05, 5.5, 500, 45, 200, "Very few cells, extreme budget"},
		{50, 0.08, 5.2, 350, 60, 120, "Mid cells, tuned params"},

		// Focus on microplate generation
		{45, 0.10, 4.0, 300, 65, 200, "Baseline + more target plates + refinement"},
		{40, 0.08, 4.8, 400, 70, 250, "Aggressive micro generation"},

		// Balance attempts
		{42, 0.09, 4.7, 350, 58, 175, "Balanced mid-range"},
		{38, 0.07, 5.3, 450, 52, 180, "Aggressive majors, careful micros"},

		// Extreme variations
		{30, 0.05, 6.0, 600, 45, 300, "Extreme: few cells, max budget, long growth"},
		{55, 0.12, 4.2, 250, 70, 150, "Many cells, higher power, lower budget"},

		// Fine-tuned around Phase 2
		{44, 0.10, 4.6, 320, 56, 110, "Phase 2 + slight tweaks 1"},
		{46, 0.09, 4.4, 280, 54, 105, "Phase 2 + slight tweaks 2"},
		{45, 0.11, 4.7, 310, 57, 115, "Phase 2 + slight tweaks 3"},
	}

	fmt.Println("Config | Cells | β     | Budget | Growth | Target | Refine | Plates (Maj/Min/Mic) | Score | 7Maj? | Description")
	fmt.Println("-------|-------|-------|--------|--------|--------|--------|----------------------|-------|-------|------------------")

	bestScore := 0.0
	var bestConfig config
	var bestMetrics evaluation.DetailedMetrics
	bestHas7Majors := false

	for i, cfg := range configs {
		settings := tectonics.DefaultHybridSettings()
		settings.Verbose = false
		settings.RefinementIterations = cfg.refinements

		settings.ConvectionSettings.NumConvectionCells = cfg.cells
		settings.ConvectionSettings.PowerLawExponent = cfg.powerLaw
		settings.ConvectionSettings.GrowthBudgetExponent = cfg.budgetExp
		settings.ConvectionSettings.GrowthIterations = cfg.growthIter
		settings.ConvectionSettings.MinCellStrength = 0.01
		settings.ConvectionSettings.MaxCellStrength = 1.0
		settings.ConvectionSettings.DivergenceThreshold = 0.2
		settings.ConvectionSettings.MinBoundaryDistance = 300.0
		settings.ConvectionSettings.TargetPlateCount = cfg.targetPlates
		settings.ConvectionSettings.Seed = 12345

		plates, _, err := tectonics.HybridPlateGeneration(
			voronoiCells,
			icosphereSites,
			planetRadius,
			settings,
		)

		if err != nil {
			fmt.Printf("%2d     | ERROR | %s\n", i+1, cfg.description)
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

		scoreMarker := ""
		if metrics.Benchmark.OverallScore >= 0.75 {
			scoreMarker = " ✓✓✓"
		} else if metrics.Benchmark.OverallScore >= 0.70 {
			scoreMarker = " ○○"
		} else if metrics.Benchmark.OverallScore >= 0.60 {
			scoreMarker = " ○"
		}

		fmt.Printf("%2d     | %2d    | %.2f  | %.1f    | %3d    | %2d     | %3d    | %2d (%d/%d/%d)          | %.1f%%%s | %-5s | %s\n",
			i+1,
			cfg.cells,
			cfg.powerLaw,
			cfg.budgetExp,
			cfg.growthIter,
			cfg.targetPlates,
			cfg.refinements,
			len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount,
			metrics.Benchmark.OverallScore*100,
			scoreMarker,
			majorMarker,
			cfg.description)
	}

	fmt.Println()
	fmt.Println("=== BEST CONFIGURATION ===")
	fmt.Printf("Description: %s\n", bestConfig.description)
	fmt.Printf("Parameters:\n")
	fmt.Printf("  Cells: %d, PowerLaw: %.2f, Budget: %.1f\n", bestConfig.cells, bestConfig.powerLaw, bestConfig.budgetExp)
	fmt.Printf("  Growth: %d, Target: %d, Refinements: %d\n", bestConfig.growthIter, bestConfig.targetPlates, bestConfig.refinements)
	fmt.Printf("Score: %.1f%% (target: 75.0%%, gap: %.1f%%)\n",
		bestScore*100, 75.0-bestScore*100)
	fmt.Printf("Distribution: %d major, %d minor, %d micro (total: %d)\n",
		bestMetrics.SizeDistribution.MajorCount,
		bestMetrics.SizeDistribution.MinorCount,
		bestMetrics.SizeDistribution.MicroCount,
		bestMetrics.SizeDistribution.MajorCount+bestMetrics.SizeDistribution.MinorCount+bestMetrics.SizeDistribution.MicroCount)
	fmt.Printf("Gini: %.3f (Earth: 0.811)\n", bestMetrics.SizeDistribution.GiniCoefficient)
	fmt.Printf("Power law β: %.3f (Earth: 0.390)\n", bestMetrics.PowerLaw.Exponent)
	fmt.Println()

	if bestHas7Majors {
		fmt.Println("✓ PERFECT major plate count!")
	} else {
		fmt.Printf("△ Major plate count: %d (target: 7, gap: %d)\n",
			bestMetrics.SizeDistribution.MajorCount, 7-bestMetrics.SizeDistribution.MajorCount)
	}

	if bestScore >= 0.75 {
		fmt.Println("✓✓✓ TARGET ACHIEVED! Score >= 75%")
	} else if bestScore >= 0.70 {
		fmt.Printf("○○ Very close to target (%.1f%% gap)\n", 75.0-bestScore*100)
	} else if bestScore >= 0.60 {
		fmt.Printf("○ Approaching target (%.1f%% gap)\n", 75.0-bestScore*100)
	} else {
		fmt.Printf("△ Significant gap to target (%.1f%% gap)\n", 75.0-bestScore*100)
		fmt.Println("\nRecommendation: May need algorithmic changes, not just parameter tuning.")
	}
}
