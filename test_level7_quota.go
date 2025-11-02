package main

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== QUOTA-BASED GENERATION: LEVEL 7 TEST ===")
	fmt.Println()

	planetRadius := 6371000.0
	subdivision := 7

	fmt.Printf("Creating icosphere subdivision %d...\n", subdivision)
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	fmt.Printf("Generated %d vertices, %d faces\n", len(vertices), len(faces))

	fmt.Println("Generating spherical Voronoi tessellation...")
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	fmt.Printf("Created %d Voronoi cells\n", len(voronoiCells))

	icosphereSites := vertices

	fmt.Println()
	fmt.Println("Running quota-based plate generation...")
	settings := tectonics.DefaultQuotaSettings()
	settings.Verbose = true

	plates, _, _ := tectonics.QuotaBasedGeneration(
		voronoiCells, icosphereSites, planetRadius, settings)

	fmt.Println()
	fmt.Println("Evaluating results...")

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

	fmt.Println()
	fmt.Println("=== RESULTS ===")
	fmt.Printf("Total plates: %d\n", len(plates))
	fmt.Printf("Distribution: %d major, %d minor, %d micro\n",
		metrics.SizeDistribution.MajorCount,
		metrics.SizeDistribution.MinorCount,
		metrics.SizeDistribution.MicroCount)
	fmt.Println()
	fmt.Printf("Earth Benchmark Score: %.1f%%\n", metrics.Benchmark.OverallScore*100)
	fmt.Printf("Gini coefficient: %.3f (Earth: 0.811)\n", metrics.SizeDistribution.GiniCoefficient)
	fmt.Printf("Power law β: %.3f (Earth: 0.390)\n", metrics.PowerLaw.Exponent)
	fmt.Println()

	// Show size distribution
	fmt.Println("Plate size distribution:")
	sizes := make([]float64, len(plates))
	for i, plate := range plates {
		sizes[i] = (plate.Area / sphereArea) * 100.0
	}

	// Sort largest to smallest
	for i := 0; i < len(sizes)-1; i++ {
		for j := i + 1; j < len(sizes); j++ {
			if sizes[j] > sizes[i] {
				sizes[i], sizes[j] = sizes[j], sizes[i]
			}
		}
	}

	fmt.Printf("Largest 10 plates: ")
	for i := 0; i < 10 && i < len(sizes); i++ {
		fmt.Printf("%.2f%% ", sizes[i])
	}
	fmt.Println()

	fmt.Printf("Smallest 10 plates: ")
	for i := len(sizes) - 10; i < len(sizes) && i >= 0; i++ {
		if i < len(sizes) {
			fmt.Printf("%.3f%% ", sizes[i])
		}
	}
	fmt.Println()

	if len(sizes) > 0 {
		fmt.Printf("\nSize ratio (largest/smallest): %.0fx\n", sizes[0]/sizes[len(sizes)-1])
	}
}
