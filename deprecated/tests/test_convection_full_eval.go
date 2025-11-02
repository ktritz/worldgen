package main

// test_convection_full_eval.go - Full evaluation of mantle convection method

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== FULL EVALUATION: Mantle Convection Method ===")
	fmt.Println()

	// Test configuration
	planetRadius := 6371000.0 // Earth radius in meters
	subdivision := 5           // Same as baseline for comparison

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Subdivision: %d\n", subdivision)
	fmt.Printf("  Planet radius: %.0f km\n\n", planetRadius/1000)

	// Generate icosphere
	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices

	fmt.Printf("  Generated %d sites\n\n", len(icosphereSites))

	// Configure convection settings - OPTIMIZED PARAMETERS
	settings := tectonics.DefaultConvectionSettings()
	settings.PowerLawExponent = 0.10     // Optimized for plate distribution
	settings.NumConvectionCells = 45     // Best balance of plate types
	settings.GrowthIterations = 300      // Sufficient for convergence
	settings.TargetPlateCount = 55       // Allow enough boundaries
	settings.DivergenceThreshold = 0.2   // Boundary detection sensitivity
	settings.MinBoundaryDistance = 300.0 // Minimum separation (km)
	settings.MinCellStrength = 0.01      // Allows micro plates
	settings.MaxCellStrength = 1.0       // Maximum cell strength
	settings.GrowthBudgetExponent = 4.5  // KEY: Creates extreme size differentiation
	settings.Seed = 12345
	settings.Verbose = true

	fmt.Println("Convection Settings (OPTIMIZED):")
	fmt.Printf("  Power law exponent: %.2f\n", settings.PowerLawExponent)
	fmt.Printf("  Convection cells: %d\n", settings.NumConvectionCells)
	fmt.Printf("  Growth budget exponent: %.1f\n", settings.GrowthBudgetExponent)
	fmt.Printf("  Target plates: %d\n", settings.TargetPlateCount)
	fmt.Printf("  Growth iterations: %d\n\n", settings.GrowthIterations)

	// Generate plates
	fmt.Println("Generating plates using mantle convection...")
	plates, _, err := tectonics.GeneratePlatesFromMantleConvection(
		voronoiCells,
		icosphereSites,
		planetRadius,
		settings,
	)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nGenerated %d plates\n\n", len(plates))

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
	fmt.Println("=== RUNNING COMPREHENSIVE EVALUATION ===")
	fmt.Println()

	metrics := evaluation.CalculateDetailedMetrics(
		plateSizes,
		plateTypes,
		plateCentroids,
		sphereArea,
		planetRadius,
	)

	// Print results
	fmt.Println("## Size Distribution ##")
	fmt.Printf("Major plates (>6%%):     %d (Earth: %d, Baseline: 4)\n",
		metrics.SizeDistribution.MajorCount, evaluation.EarthMajorCount)
	fmt.Printf("Minor plates (0.18-6%%): %d (Earth: %d, Baseline: 16)\n",
		metrics.SizeDistribution.MinorCount, evaluation.EarthMinorCount)
	fmt.Printf("Micro plates (<0.18%%):  %d (Earth: %d, Baseline: 0)\n",
		metrics.SizeDistribution.MicroCount, evaluation.EarthMicroCount)
	fmt.Printf("Total plates:           %d (Earth: %d, Baseline: 20)\n",
		len(plates), evaluation.EarthTotalPlates)
	fmt.Println()

	fmt.Printf("Largest plate:  %.2f%% (Earth: %.2f%%, Baseline: 19.8%%)\n",
		metrics.SizeDistribution.LargestPlate, evaluation.EarthLargestPlate*100)
	fmt.Printf("Size ratio:     %.0fx (Earth: %.0fx, Baseline: 11x)\n",
		metrics.SizeDistribution.SizeRatio, evaluation.EarthSizeRatio)
	fmt.Println()

	fmt.Printf("Gini coefficient: %.3f (Earth: %.3f, Baseline: 0.433)\n",
		metrics.SizeDistribution.GiniCoefficient, evaluation.EarthGiniCoefficient)
	fmt.Println()

	fmt.Println("## Power Law Analysis ##")
	fmt.Printf("Exponent (β):    %.3f (Earth: %.3f, Baseline: 1.098)\n",
		metrics.PowerLaw.Exponent, evaluation.EarthPowerLawExponent)
	fmt.Printf("R² fit:          %.4f (Earth: ~0.92, Baseline: 0.948)\n", metrics.PowerLaw.R2Fit)
	fmt.Println()

	// Print Earth Benchmark Score
	fmt.Println("=== EARTH BENCHMARK SCORE ===")
	fmt.Println()
	fmt.Println(evaluation.FormatBenchmarkSummary(metrics.Benchmark))

	// Comparison
	fmt.Println()
	fmt.Println("=== COMPARISON TO BASELINE ===")
	fmt.Println()

	baselineScore := 0.494
	convectionScore := metrics.Benchmark.OverallScore

	fmt.Printf("Baseline Method:    %.1f%%\n", baselineScore*100)
	fmt.Printf("Convection Method:  %.1f%%\n", convectionScore*100)
	fmt.Printf("Improvement:        %+.1f%% (%.1fx)\n",
		(convectionScore-baselineScore)*100,
		convectionScore/baselineScore)
	fmt.Println()

	if convectionScore > 0.75 {
		fmt.Println("✓✓✓ EXCELLENT: Exceeded target score (>75%)!")
	} else if convectionScore > 0.65 {
		fmt.Println("✓✓ GREAT: Significant improvement, close to target!")
	} else if convectionScore > baselineScore {
		fmt.Println("✓ GOOD: Improvement over baseline")
	} else {
		fmt.Println("△ Needs more tuning")
	}
}
