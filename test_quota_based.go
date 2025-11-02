package main

import (
	"fmt"
	"math"
	"strings"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== QUOTA-BASED PLATE GENERATION TEST ===")
	fmt.Println()
	fmt.Println("Demonstrates deterministic control over plate size distribution")
	fmt.Println()

	planetRadius := 6371000.0
	subdivision := 5

	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices

	// Test 1: Earth-like distribution (7/13/19)
	fmt.Println("TEST 1: Earth-like Distribution (7 major, 13 minor, 19 micro)")
	fmt.Println(strings.Repeat("=", 70))
	{
		settings := tectonics.DefaultQuotaSettings()
		settings.MajorCount = 7
		settings.MinorCount = 13
		settings.MicroCount = 19
		settings.Verbose = false

		plates, _, _ := tectonics.QuotaBasedGeneration(
			voronoiCells, icosphereSites, planetRadius, settings)

		metrics := evaluatePlates(plates, planetRadius)

		fmt.Printf("Result: %d plates (%d major, %d minor, %d micro)\n",
			len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount)
		fmt.Printf("Score: %.1f%%\n", metrics.Benchmark.OverallScore*100)

		if metrics.SizeDistribution.MajorCount == 7 {
			fmt.Println("✓ Exactly 7 major plates achieved!")
		}
		fmt.Println()
	}

	// Test 2: More major plates, fewer micro
	fmt.Println("TEST 2: Modified Distribution (12 major, 10 minor, 10 micro)")
	fmt.Println(strings.Repeat("=", 70))
	{
		settings := tectonics.DefaultQuotaSettings()
		settings.MajorCount = 12
		settings.MinorCount = 10
		settings.MicroCount = 10
		settings.Verbose = false

		plates, _, _ := tectonics.QuotaBasedGeneration(
			voronoiCells, icosphereSites, planetRadius, settings)

		metrics := evaluatePlates(plates, planetRadius)

		fmt.Printf("Result: %d plates (%d major, %d minor, %d micro)\n",
			len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount)
		fmt.Printf("Score: %.1f%%\n", metrics.Benchmark.OverallScore*100)
		fmt.Println()
	}

	// Test 3: Extreme microplate world
	fmt.Println("TEST 3: Microplate World (2 major, 5 minor, 50 micro)")
	fmt.Println(strings.Repeat("=", 70))
	{
		settings := tectonics.DefaultQuotaSettings()
		settings.MajorCount = 2
		settings.MinorCount = 5
		settings.MicroCount = 50
		settings.Verbose = false

		plates, _, _ := tectonics.QuotaBasedGeneration(
			voronoiCells, icosphereSites, planetRadius, settings)

		metrics := evaluatePlates(plates, planetRadius)

		fmt.Printf("Result: %d plates (%d major, %d minor, %d micro)\n",
			len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount)
		fmt.Printf("Score: %.1f%%\n", metrics.Benchmark.OverallScore*100)
		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("DEMONSTRATION COMPLETE")
	fmt.Println()
	fmt.Println("Key Takeaway: Quota-based generation provides deterministic control.")
}

func evaluatePlates(plates []tectonics.TectonicPlate, planetRadius float64) evaluation.DetailedMetrics {
	plateSizes := make([]float64, len(plates))
	plateTypes := make([]bool, len(plates))
	plateCentroids := make([][3]float64, len(plates))

	for i, plate := range plates {
		plateSizes[i] = plate.Area
		plateTypes[i] = (plate.PlateType == tectonics.ContinentalPlate)
		plateCentroids[i] = [3]float64{plate.Center.X, plate.Center.Y, plate.Center.Z}
	}

	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius

	return evaluation.CalculateDetailedMetrics(plateSizes, plateTypes, plateCentroids, sphereArea, planetRadius)
}
