package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	// Command line flags
	testEarth := flag.Bool("test-earth", false, "Test evaluation framework using Earth's actual plate data")
	outputDir := flag.String("output", "evaluation_output", "Output directory for reports")

	flag.Parse()

	fmt.Println("=== Tectonic Plate Evaluation Tool ===")
	fmt.Println()

	if *testEarth {
		fmt.Println("Running validation test using Earth's actual plate distribution...")
		testEarthData(*outputDir)
	} else {
		fmt.Println("Usage:")
		fmt.Println("  -test-earth    Test evaluation framework with Earth's plate data")
		fmt.Println("  -output DIR    Output directory (default: evaluation_output)")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  evaluate_plates -test-earth")
	}
}

// testEarthData validates the evaluation framework using Earth's known plate distribution
func testEarthData(outputDir string) {
	fmt.Println("\n--- Loading Earth Plate Data ---")

	// Get Earth's actual plate distribution
	earthPlates := tectonics.GetEarthPlateDistribution()

	fmt.Printf("Loaded %d plates from Earth data\n", len(earthPlates))

	// Extract plate sizes
	plateSizes := make([]float64, len(earthPlates))
	plateTypes := make([]bool, len(earthPlates))
	totalArea := 0.0

	for i, plate := range earthPlates {
		// AreaPercent is already % of sphere
		plateSizes[i] = plate.AreaPercent / 100.0 // Convert to fraction
		totalArea += plateSizes[i]

		// Continental if >50% continental crust
		plateTypes[i] = plate.ContinentalPct > 0.5
	}

	fmt.Printf("Total area coverage: %.1f%%\n", totalArea*100)

	// Earth parameters
	earthRadius := 6371000.0                                 // meters
	sphereArea := 4.0 * math.Pi * earthRadius * earthRadius

	// Scale plate sizes to actual areas
	plateAreas := make([]float64, len(plateSizes))
	for i, sizeFraction := range plateSizes {
		plateAreas[i] = sizeFraction * sphereArea
	}

	// Generate dummy centroids (we don't have actual positions, but this won't affect most metrics)
	plateCentroids := make([][3]float64, len(plateAreas))
	for i := range plateCentroids {
		// Distribute somewhat evenly
		lat := float64(i)/float64(len(plateAreas))*180 - 90
		lon := float64(i%10) * 36
		plateCentroids[i] = latLonToCartesian(lat, lon, earthRadius)
	}

	fmt.Println("\n--- Calculating Metrics ---")

	// Calculate all metrics
	metrics := evaluation.CalculateDetailedMetrics(
		plateAreas,
		plateTypes,
		plateCentroids,
		sphereArea,
		earthRadius,
	)

	fmt.Println("\n--- Results ---")
	printMetricsSummary(metrics)

	fmt.Println("\n--- Earth Benchmark Score ---")
	fmt.Println(evaluation.FormatBenchmarkSummary(metrics.Benchmark))

	// Expected: Earth data should score high (>0.85)
	// Note: Won't be perfect because we're using dummy centroids (no actual positions)
	expectedScore := 0.85
	tolerance := 0.10

	fmt.Println("\n--- Validation ---")
	if math.Abs(metrics.Benchmark.OverallScore-expectedScore) < tolerance {
		fmt.Printf("✓ PASS: Earth data scored %.3f (expected ~%.1f)\n",
			metrics.Benchmark.OverallScore, expectedScore)
		fmt.Println("Evaluation framework is working correctly!")
	} else {
		fmt.Printf("✗ FAIL: Earth data scored %.3f (expected ~%.1f)\n",
			metrics.Benchmark.OverallScore, expectedScore)
		fmt.Println("There may be an issue with the evaluation framework.")
		fmt.Println("Check the component scores above for details.")
	}

	// Create output directory
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		fmt.Printf("Warning: Could not create output directory: %v\n", err)
		return
	}

	fmt.Printf("\n--- Generating Visualizations (to %s) ---\n", outputDir)

	// Generate placeholder visualizations
	evaluation.GenerateSizeHistogram(metrics.SizeDistribution, metrics.PowerLaw,
		outputDir+"/earth_histogram.png")
	evaluation.GeneratePowerLawPlot(metrics.PowerLaw,
		outputDir+"/earth_powerlaw.png")
	evaluation.GenerateComparisonRadar(metrics.Benchmark,
		outputDir+"/earth_radar.png")

	fmt.Println("\nEvaluation complete!")
}

// printMetricsSummary prints a formatted summary of all metrics
func printMetricsSummary(metrics evaluation.DetailedMetrics) {
	fmt.Println("\n## Size Distribution ##")
	fmt.Printf("Major plates (>6%%):     %d\n", metrics.SizeDistribution.MajorCount)
	fmt.Printf("Minor plates (0.18-6%%): %d\n", metrics.SizeDistribution.MinorCount)
	fmt.Printf("Micro plates (<0.18%%):  %d\n", metrics.SizeDistribution.MicroCount)
	fmt.Printf("Total plates:           %d\n", metrics.PlateCount)
	fmt.Println()
	fmt.Printf("Largest plate:  %.2f%%\n", metrics.SizeDistribution.LargestPlate)
	fmt.Printf("Smallest plate: %.3f%%\n", metrics.SizeDistribution.SmallestPlate)
	fmt.Printf("Size ratio:     %.0fx\n", metrics.SizeDistribution.SizeRatio)
	fmt.Printf("Median size:    %.2f%%\n", metrics.SizeDistribution.MedianSize)
	fmt.Println()
	fmt.Printf("Gini coefficient: %.3f\n", metrics.SizeDistribution.GiniCoefficient)
	fmt.Printf("Mean:             %.2f%%\n", metrics.SizeDistribution.Mean)
	fmt.Printf("Std Dev:          %.2f%%\n", metrics.SizeDistribution.StdDev)
	fmt.Printf("Skewness:         %.2f\n", metrics.SizeDistribution.Skewness)

	fmt.Println("\n## Coverage by Class ##")
	fmt.Printf("Major plates: %.1f%%\n", metrics.SizeDistribution.MajorCoverage)
	fmt.Printf("Minor plates: %.1f%%\n", metrics.SizeDistribution.MinorCoverage)
	fmt.Printf("Micro plates: %.1f%%\n", metrics.SizeDistribution.MicroCoverage)

	fmt.Println("\n## Power Law Analysis ##")
	fmt.Printf("Exponent (β):    %.3f\n", metrics.PowerLaw.Exponent)
	fmt.Printf("R² fit:          %.4f\n", metrics.PowerLaw.R2Fit)
	fmt.Printf("KS statistic:    %.4f\n", metrics.PowerLaw.KSStatistic)
	fmt.Printf("KS p-value:      %.4f\n", metrics.PowerLaw.KSPValue)
	fmt.Printf("Valid range:     %.3f%% - %.1f%%\n",
		metrics.PowerLaw.ValidRange[0], metrics.PowerLaw.ValidRange[1])

	fmt.Println("\n## Continental Distribution ##")
	fmt.Printf("Continental plates: %d (%.1f%%)\n",
		metrics.Continental.ContinentalCount,
		metrics.Continental.ContinentalRatio*100)
	fmt.Printf("Oceanic plates:     %d (%.1f%%)\n",
		metrics.Continental.OceanicCount,
		(1-metrics.Continental.ContinentalRatio)*100)
	fmt.Printf("Land coverage:      %.1f%%\n", metrics.Continental.LandCoverage)

	fmt.Println("\n## Spatial Distribution ##")
	fmt.Printf("Latitude balance:       %.2f\n", metrics.Spatial.LatitudeBalance)
	fmt.Printf("Centroid distribution:  %.2f\n", metrics.Spatial.CentroidDistribution)
}

// latLonToCartesian converts lat/lon to 3D Cartesian coordinates
func latLonToCartesian(lat, lon, radius float64) [3]float64 {
	latRad := lat * math.Pi / 180.0
	lonRad := lon * math.Pi / 180.0

	x := radius * math.Cos(latRad) * math.Cos(lonRad)
	y := radius * math.Cos(latRad) * math.Sin(lonRad)
	z := radius * math.Sin(latRad)

	return [3]float64{x, y, z}
}
