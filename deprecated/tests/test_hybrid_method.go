package main

// test_hybrid_method.go - Test Method 7: Hybrid (Convection + Optimization)
// Expected score: 0.75-0.90

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== METHOD 7: HYBRID GENERATION (Convection + Optimization) ===")
	fmt.Println()
	fmt.Println("Research expected score: 75-90%")
	fmt.Println("Target: 39 plates (7 major, 13 minor, 19 micro)")
	fmt.Println()

	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\n\n", len(icosphereSites))

	// Configure hybrid settings
	settings := tectonics.DefaultHybridSettings()
	settings.Verbose = true
	settings.RefinementIterations = 50

	// Start with Phase 2 settings that achieved 7 majors!
	settings.ConvectionSettings.NumConvectionCells = 45
	settings.ConvectionSettings.PowerLawExponent = 0.10
	settings.ConvectionSettings.GrowthBudgetExponent = 4.5
	settings.ConvectionSettings.GrowthIterations = 300
	settings.ConvectionSettings.MinCellStrength = 0.01
	settings.ConvectionSettings.MaxCellStrength = 1.0
	settings.ConvectionSettings.DivergenceThreshold = 0.2
	settings.ConvectionSettings.MinBoundaryDistance = 300.0
	settings.ConvectionSettings.TargetPlateCount = 55
	settings.ConvectionSettings.Seed = 12345

	// Generate plates using hybrid method
	plates, _, err := tectonics.HybridPlateGeneration(
		voronoiCells,
		icosphereSites,
		planetRadius,
		settings,
	)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\n=== EVALUATION ===\n\n")

	// Prepare data for evaluation
	plateSizes := make([]float64, len(plates))
	plateTypes := make([]bool, len(plates))
	plateCentroids := make([][3]float64, len(plates))

	for i, plate := range plates {
		plateSizes[i] = plate.Area
		plateTypes[i] = (plate.PlateType == tectonics.ContinentalPlate)
		plateCentroids[i] = [3]float64{plate.Center.X, plate.Center.Y, plate.Center.Z}
	}

	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius

	// Run comprehensive evaluation
	metrics := evaluation.CalculateDetailedMetrics(
		plateSizes,
		plateTypes,
		plateCentroids,
		sphereArea,
		planetRadius,
	)

	// Print results
	fmt.Println("## Size Distribution ##")
	fmt.Printf("Major plates (>6%%):     %d (Earth: %d, Baseline: 4, Target: 7)\n",
		metrics.SizeDistribution.MajorCount, evaluation.EarthMajorCount)
	fmt.Printf("Minor plates (0.18-6%%): %d (Earth: %d, Baseline: 16, Target: 13)\n",
		metrics.SizeDistribution.MinorCount, evaluation.EarthMinorCount)
	fmt.Printf("Micro plates (<0.18%%):  %d (Earth: %d, Baseline: 0, Target: 19)\n",
		metrics.SizeDistribution.MicroCount, evaluation.EarthMicroCount)
	fmt.Printf("Total plates:           %d (Earth: %d)\n", len(plates), evaluation.EarthTotalPlates)
	fmt.Println()

	fmt.Printf("Gini coefficient: %.3f (Earth: %.3f, Baseline: 0.433)\n",
		metrics.SizeDistribution.GiniCoefficient, evaluation.EarthGiniCoefficient)
	fmt.Printf("Power law β:      %.3f (Earth: %.3f, Baseline: 1.098)\n",
		metrics.PowerLaw.Exponent, evaluation.EarthPowerLawExponent)
	fmt.Printf("Size ratio:       %.0fx (Earth: %.0fx, Baseline: 11x)\n",
		metrics.SizeDistribution.SizeRatio, evaluation.EarthSizeRatio)
	fmt.Println()

	// Print Earth Benchmark Score
	fmt.Println("=== EARTH BENCHMARK SCORE ===")
	fmt.Println()
	fmt.Println(evaluation.FormatBenchmarkSummary(metrics.Benchmark))

	// Compare to targets
	fmt.Println()
	fmt.Println("=== COMPARISON TO TARGETS ===")
	fmt.Println()

	baselineScore := 0.494
	phase2Score := 0.558
	targetScore := 0.75
	currentScore := metrics.Benchmark.OverallScore

	fmt.Printf("Baseline:    %.1f%%\n", baselineScore*100)
	fmt.Printf("Phase 2:     %.1f%%\n", phase2Score*100)
	fmt.Printf("Current:     %.1f%%\n", currentScore*100)
	fmt.Printf("Target:      %.1f%%\n", targetScore*100)
	fmt.Println()

	if currentScore >= targetScore {
		fmt.Println("✓✓✓ SUCCESS: Target achieved!")
	} else if currentScore >= phase2Score {
		fmt.Printf("○ Improvement over Phase 2 (+%.1f%%), but below target\n",
			(currentScore-phase2Score)*100)
		fmt.Printf("  Gap to target: -%.1f%%\n", (targetScore-currentScore)*100)
	} else {
		fmt.Printf("△ Below Phase 2 result\n")
	}
}
