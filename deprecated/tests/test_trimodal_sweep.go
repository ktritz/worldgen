package main

// test_trimodal_sweep.go - Fine-tune tri-modal parameters to achieve 7 major plates

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== TRI-MODAL PARAMETER OPTIMIZATION ===")
	fmt.Println()
	fmt.Println("Current: 69.2% with 4/12/19 distribution")
	fmt.Println("Goal: Find parameters that achieve 7 major plates while keeping 12-13 minor and ~19 micro")
	fmt.Println()

	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\n\n", len(icosphereSites))

	type config struct {
		majorStr  float64
		minorStr  float64
		microStr  float64
		budgetExp float64
		desc      string
	}

	configs := []config{
		// Current baseline (achieved 4/12/19)
		{1.0, 0.25, 0.05, 3.5, "Baseline (69.2%)"},

		// Try flatter strength ratios to allow more majors
		{1.0, 0.35, 0.08, 3.5, "Flatter strengths, same budget"},
		{1.0, 0.40, 0.10, 3.5, "Even flatter strengths"},

		// Try lower budget exponents
		{1.0, 0.25, 0.05, 3.0, "Lower budget exp 3.0"},
		{1.0, 0.25, 0.05, 2.5, "Lower budget exp 2.5"},
		{1.0, 0.30, 0.06, 2.8, "Balanced: flatter + lower budget"},

		// Try making majors weaker to prevent dominance
		{0.85, 0.25, 0.05, 3.5, "Weaker majors"},
		{0.80, 0.25, 0.05, 3.0, "Weaker majors + lower budget"},

		// Try making minors stronger
		{1.0, 0.35, 0.05, 3.0, "Stronger minors + lower budget"},
		{1.0, 0.40, 0.08, 2.8, "Much stronger minors"},

		// Extreme variations
		{1.0, 0.30, 0.05, 2.2, "Very low budget exp"},
		{0.90, 0.35, 0.08, 2.5, "All moderate"},
	}

	fmt.Println("Config | MajStr | MinStr | MicStr | Budget | Plates (Maj/Min/Mic) | Score | 7Maj? | Description")
	fmt.Println("-------|--------|--------|--------|--------|----------------------|-------|-------|------------------")

	bestScore := 0.0
	var bestConfig config
	var bestMetrics evaluation.DetailedMetrics
	bestHas7Majors := false

	for i, cfg := range configs {
		settings := tectonics.DefaultConvectionSettings()
		settings.NumConvectionCells = 50
		settings.UseTriModalStrength = true
		settings.MajorCellStrength = cfg.majorStr
		settings.MinorCellStrength = cfg.minorStr
		settings.MicroCellStrength = cfg.microStr
		settings.GrowthBudgetExponent = cfg.budgetExp
		settings.GrowthIterations = 300
		settings.TargetPlateCount = 55
		settings.Seed = 12345
		settings.Verbose = false

		plates, _, err := tectonics.GeneratePlatesFromMantleConvection(
			voronoiCells,
			icosphereSites,
			planetRadius,
			settings,
		)

		if err != nil {
			fmt.Printf("%2d     | ERROR  | %s\n", i+1, cfg.desc)
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
		} else if metrics.Benchmark.OverallScore >= 0.65 {
			scoreMarker = " ○"
		}

		fmt.Printf("%2d     | %.2f   | %.2f   | %.2f   | %.1f    | %2d (%d/%d/%d)          | %.1f%%%s | %-5s | %s\n",
			i+1,
			cfg.majorStr,
			cfg.minorStr,
			cfg.microStr,
			cfg.budgetExp,
			len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount,
			metrics.Benchmark.OverallScore*100,
			scoreMarker,
			majorMarker,
			cfg.desc)
	}

	fmt.Println()
	fmt.Println("=== BEST CONFIGURATION ===")
	fmt.Printf("Parameters: MajStr=%.2f, MinStr=%.2f, MicStr=%.2f, BudgetExp=%.1f\n",
		bestConfig.majorStr, bestConfig.minorStr, bestConfig.microStr, bestConfig.budgetExp)
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
		fmt.Println("✓✓✓ PERFECT major plate count!")
	} else {
		fmt.Printf("△ Major plate count: %d (target: 7, gap: %d)\n",
			bestMetrics.SizeDistribution.MajorCount, 7-bestMetrics.SizeDistribution.MajorCount)
	}

	if bestScore >= 0.75 {
		fmt.Println("✓✓✓ TARGET ACHIEVED! Score >= 75%")
	} else if bestScore >= 0.70 {
		fmt.Printf("○○ Very close! Gap: %.1f%%\n", 75.0-bestScore*100)
	} else {
		fmt.Printf("○ Gap to target: %.1f%%\n", 75.0-bestScore*100)
	}
}
