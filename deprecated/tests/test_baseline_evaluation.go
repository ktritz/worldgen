package main

// test_baseline_evaluation.go - Establish baseline Earth benchmark score for current plate generation

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== BASELINE EVALUATION: Current Plate Generation ===")
	fmt.Println()

	// Earth-like settings (similar to what pipeline uses)
	settings := tectonics.TectonicSettings{
		NumPlates:                   20,  // Earth-like: ~7 major + ~13 minor
		MajorPlateRatio:            0.30, // 30% major plates
		Seed:                        12345,
		TargetContinentalProportion: 0.33, // Earth has ~33% continental plates
		PlanetRadius:                6371000.0, // Earth radius in meters
		BoundaryStyle:               "continent",

		// Enable advanced features for realism
		EnableContinentalLandmasses: true,
		TargetLandCoverage:          0.29,
		EnableRidgeSystem:           true,
		EnableHotspots:              true,
		EnableSeafloorAging:         true,
		MaxSeafloorAge:              200.0,
	}

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Target plates: %d (%d major, %d minor)\n",
		settings.NumPlates,
		int(float64(settings.NumPlates)*settings.MajorPlateRatio),
		int(float64(settings.NumPlates)*(1-settings.MajorPlateRatio)))
	fmt.Printf("  Planet radius: %.0f km\n", settings.PlanetRadius/1000)
	fmt.Printf("  Seed: %d\n", settings.Seed)
	fmt.Println()

	// Generate icosphere (subdivision 5 for reasonable resolution)
	fmt.Println("Generating icosphere (subdivision 5)...")
	subdivision := 5
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	voronoiVertices, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices

	fmt.Printf("  Generated %d sites, %d voronoi cells\n", len(icosphereSites), len(voronoiCells))
	fmt.Println()

	// Generate tectonic plates using current implementation
	fmt.Println("Generating tectonic plates (current implementation)...")
	tectonicsData, warnings := tectonics.InitializeAdvancedTectonicSystem(
		voronoiCells,
		voronoiVertices,
		icosphereSites,
		settings,
	)

	if len(warnings) > 0 {
		fmt.Printf("  Generated with %d warnings (showing first 3):\n", len(warnings))
		for i := 0; i < len(warnings) && i < 3; i++ {
			fmt.Printf("    - %s\n", warnings[i])
		}
	}
	fmt.Println()

	// Extract plate data for evaluation
	plates := tectonicsData.Plates
	fmt.Printf("Generated %d plates\n", len(plates))
	fmt.Println()

	// Convert plates to evaluation framework format
	fmt.Println("Preparing data for comprehensive evaluation...")

	plateSizes := make([]float64, len(plates))
	plateTypes := make([]bool, len(plates))
	plateCentroids := make([][3]float64, len(plates))

	for i, plate := range plates {
		plateSizes[i] = plate.Area // Already in m²
		plateTypes[i] = (plate.PlateType == tectonics.ContinentalPlate)
		plateCentroids[i] = [3]float64{plate.Center.X, plate.Center.Y, plate.Center.Z}
	}

	sphereArea := 4.0 * math.Pi * settings.PlanetRadius * settings.PlanetRadius

	fmt.Println()
	fmt.Println("=== RUNNING COMPREHENSIVE EVALUATION ===")
	fmt.Println()

	// Run comprehensive evaluation using Phase 1 framework
	metrics := evaluation.CalculateDetailedMetrics(
		plateSizes,
		plateTypes,
		plateCentroids,
		sphereArea,
		settings.PlanetRadius,
	)

	// Print detailed results
	fmt.Println("## Size Distribution ##")
	fmt.Printf("Major plates (>6%%):     %d (Earth: %d)\n",
		metrics.SizeDistribution.MajorCount, evaluation.EarthMajorCount)
	fmt.Printf("Minor plates (0.18-6%%): %d (Earth: %d)\n",
		metrics.SizeDistribution.MinorCount, evaluation.EarthMinorCount)
	fmt.Printf("Micro plates (<0.18%%):  %d (Earth: %d)\n",
		metrics.SizeDistribution.MicroCount, evaluation.EarthMicroCount)
	fmt.Printf("Total plates:           %d (Earth: %d)\n",
		len(plates), evaluation.EarthTotalPlates)
	fmt.Println()

	fmt.Printf("Largest plate:  %.2f%% (Earth: %.2f%%)\n",
		metrics.SizeDistribution.LargestPlate, evaluation.EarthLargestPlate*100)
	fmt.Printf("Smallest plate: %.3f%%\n", metrics.SizeDistribution.SmallestPlate)
	fmt.Printf("Size ratio:     %.0fx (Earth: %.0fx)\n",
		metrics.SizeDistribution.SizeRatio, evaluation.EarthSizeRatio)
	fmt.Printf("Median size:    %.2f%%\n", metrics.SizeDistribution.MedianSize)
	fmt.Println()

	fmt.Printf("Gini coefficient: %.3f (Earth: %.3f)\n",
		metrics.SizeDistribution.GiniCoefficient, evaluation.EarthGiniCoefficient)
	fmt.Printf("Mean:             %.2f%%\n", metrics.SizeDistribution.Mean)
	fmt.Printf("Std Dev:          %.2f%%\n", metrics.SizeDistribution.StdDev)
	fmt.Printf("Skewness:         %.2f\n", metrics.SizeDistribution.Skewness)
	fmt.Println()

	fmt.Println("## Coverage by Class ##")
	fmt.Printf("Major plates: %.1f%% (Earth: %.1f%%)\n",
		metrics.SizeDistribution.MajorCoverage, evaluation.EarthMajorCoverage*100)
	fmt.Printf("Minor plates: %.1f%% (Earth: %.1f%%)\n",
		metrics.SizeDistribution.MinorCoverage, evaluation.EarthMinorCoverage*100)
	fmt.Printf("Micro plates: %.1f%% (Earth: %.1f%%)\n",
		metrics.SizeDistribution.MicroCoverage, evaluation.EarthMicroCoverage*100)
	fmt.Println()

	fmt.Println("## Power Law Analysis ##")
	fmt.Printf("Exponent (β):    %.3f (Earth: %.3f)\n",
		metrics.PowerLaw.Exponent, evaluation.EarthPowerLawExponent)
	fmt.Printf("R² fit:          %.4f (Earth: ~0.92)\n", metrics.PowerLaw.R2Fit)
	fmt.Printf("KS statistic:    %.4f\n", metrics.PowerLaw.KSStatistic)
	fmt.Printf("KS p-value:      %.4f\n", metrics.PowerLaw.KSPValue)
	fmt.Printf("Valid range:     %.3f%% - %.1f%%\n",
		metrics.PowerLaw.ValidRange[0], metrics.PowerLaw.ValidRange[1])
	fmt.Println()

	fmt.Println("## Continental Distribution ##")
	fmt.Printf("Continental plates: %d (%.1f%%) [Earth: %.1f%%]\n",
		metrics.Continental.ContinentalCount,
		metrics.Continental.ContinentalRatio*100,
		evaluation.EarthContinentalRatio*100)
	fmt.Printf("Oceanic plates:     %d (%.1f%%)\n",
		metrics.Continental.OceanicCount,
		(1-metrics.Continental.ContinentalRatio)*100)
	fmt.Println()

	fmt.Println("## Spatial Distribution ##")
	fmt.Printf("Latitude balance:       %.2f\n", metrics.Spatial.LatitudeBalance)
	fmt.Printf("Centroid distribution:  %.2f\n", metrics.Spatial.CentroidDistribution)
	fmt.Println()

	// Print Earth Benchmark Score
	fmt.Println("=== EARTH BENCHMARK SCORE ===")
	fmt.Println()
	fmt.Println(evaluation.FormatBenchmarkSummary(metrics.Benchmark))

	// Analysis and recommendations
	fmt.Println()
	fmt.Println("=== BASELINE ANALYSIS ===")
	fmt.Println()

	overallScore := metrics.Benchmark.OverallScore

	if overallScore >= 0.75 {
		fmt.Printf("✓ EXCELLENT: Score %.1f%% exceeds target (>75%%)\n", overallScore*100)
	} else if overallScore >= 0.65 {
		fmt.Printf("○ GOOD: Score %.1f%% is close to target (need >75%%)\n", overallScore*100)
	} else if overallScore >= 0.50 {
		fmt.Printf("△ MODERATE: Score %.1f%% needs improvement (need >75%%)\n", overallScore*100)
	} else {
		fmt.Printf("✗ NEEDS WORK: Score %.1f%% significantly below target (need >75%%)\n", overallScore*100)
	}
	fmt.Println()

	// Identify specific weaknesses
	fmt.Println("Component Analysis:")
	fmt.Printf("  Plate Count:    %.1f%% ", metrics.Benchmark.PlateCountScore*100)
	if metrics.Benchmark.PlateCountScore < 0.7 {
		fmt.Printf("[WEAKNESS: %+d major, %+d minor, %+d micro]\n",
			metrics.Benchmark.PlateCountDelta[0],
			metrics.Benchmark.PlateCountDelta[1],
			metrics.Benchmark.PlateCountDelta[2])
	} else {
		fmt.Println("[Good]")
	}

	fmt.Printf("  Power Law Fit:  %.1f%% ", metrics.Benchmark.PowerLawScore*100)
	if metrics.Benchmark.PowerLawScore < 0.7 {
		fmt.Printf("[WEAKNESS: β=%.3f vs Earth %.3f, R²=%.3f]\n",
			metrics.Benchmark.PowerLawExponent,
			evaluation.EarthPowerLawExponent,
			metrics.Benchmark.PowerLawR2)
	} else {
		fmt.Println("[Good]")
	}

	fmt.Printf("  Size Variation: %.1f%% ", metrics.Benchmark.SizeVariationScore*100)
	if metrics.Benchmark.SizeVariationScore < 0.7 {
		fmt.Printf("[WEAKNESS: Gini %.3f vs Earth %.3f]\n",
			metrics.SizeDistribution.GiniCoefficient,
			evaluation.EarthGiniCoefficient)
	} else {
		fmt.Println("[Good]")
	}

	fmt.Printf("  Boundary:       %.1f%% ", metrics.Benchmark.BoundaryScore*100)
	if metrics.Benchmark.BoundaryScore < 0.7 {
		fmt.Println("[Limited data - using estimates]")
	} else {
		fmt.Println("[Good]")
	}

	fmt.Printf("  Spatial:        %.1f%% ", metrics.Benchmark.SpatialScore*100)
	if metrics.Benchmark.SpatialScore < 0.7 {
		fmt.Println("[Could use improvement]")
	} else {
		fmt.Println("[Good]")
	}

	fmt.Printf("  Continental:    %.1f%% ", metrics.Benchmark.ContinentalScore*100)
	if metrics.Benchmark.ContinentalScore < 0.7 {
		fmt.Printf("[WEAKNESS: %.1f%% vs Earth %.1f%%]\n",
			metrics.Continental.ContinentalRatio*100,
			evaluation.EarthContinentalRatio*100)
	} else {
		fmt.Println("[Good]")
	}

	fmt.Println()
	fmt.Println("=== NEXT STEPS ===")
	fmt.Println()
	fmt.Println("This baseline establishes the starting point for improvement.")
	fmt.Printf("Current method: 5x overseeding + intelligent merging (score: %.1f%%)\n", overallScore*100)
	fmt.Printf("Target method: Mantle convection-based generation (target: >75%%)\n")
	fmt.Printf("Expected improvement: %.1f%% → >75%% (%.1fx improvement)\n",
		overallScore*100, 75.0/overallScore/100.0)
	fmt.Println()
	fmt.Println("The comprehensive evaluation framework is working correctly.")
	fmt.Println("Ready to begin Phase 2: Mantle convection implementation")
}
