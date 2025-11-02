package main

import (
	"fmt"
	"math"
	"sort"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== DETAILED DISTRIBUTION ANALYSIS ===\n")

	planetRadius := 6371000.0
	subdivision := 5

	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius

	// Generate both methods
	fmt.Println("QUOTA-BASED METHOD:")
	fmt.Println("-------------------")
	{
		settings := tectonics.DefaultQuotaSettings()
		settings.Verbose = false
		plates, _, _ := tectonics.QuotaBasedGeneration(
			voronoiCells, icosphereSites, planetRadius, settings)

		analyzePlateDistribution(plates, sphereArea, "Quota-Based")
	}

	fmt.Println("\nTRI-MODAL CONVECTION:")
	fmt.Println("---------------------")
	{
		settings := tectonics.DefaultConvectionSettings()
		settings.NumConvectionCells = 50
		settings.UseTriModalStrength = true
		settings.MajorCellStrength = 1.0
		settings.MinorCellStrength = 0.25
		settings.MicroCellStrength = 0.05
		settings.GrowthBudgetExponent = 3.5
		settings.Verbose = false

		plates, _, _ := tectonics.GeneratePlatesFromMantleConvection(
			voronoiCells, icosphereSites, planetRadius, settings)

		analyzePlateDistribution(plates, sphereArea, "Tri-Modal")
	}
}

func analyzePlateDistribution(plates []tectonics.TectonicPlate, sphereArea float64, name string) {
	// Get sizes in percent
	sizes := make([]float64, len(plates))
	for i, plate := range plates {
		sizes[i] = (plate.Area / sphereArea) * 100.0
	}

	// Sort largest to smallest
	sort.Float64s(sizes)
	for i, j := 0, len(sizes)-1; i < j; i, j = i+1, j-1 {
		sizes[i], sizes[j] = sizes[j], sizes[i]
	}

	// Categorize
	var majors, minors, micros []float64
	for _, size := range sizes {
		if size >= 6.0 {
			majors = append(majors, size)
		} else if size >= 0.18 {
			minors = append(minors, size)
		} else {
			micros = append(micros, size)
		}
	}

	// Calculate metrics
	plateSizes := make([]float64, len(plates))
	plateTypes := make([]bool, len(plates))
	plateCentroids := make([][3]float64, len(plates))
	for i, plate := range plates {
		plateSizes[i] = plate.Area
		plateTypes[i] = (plate.PlateType == tectonics.ContinentalPlate)
		plateCentroids[i] = [3]float64{plate.Center.X, plate.Center.Y, plate.Center.Z}
	}

	metrics := evaluation.CalculateDetailedMetrics(plateSizes, plateTypes, plateCentroids, sphereArea, 6371000.0)

	fmt.Printf("Score: %.1f%%\n", metrics.Benchmark.OverallScore*100)
	fmt.Printf("Gini: %.3f (Earth: 0.811)\n", metrics.SizeDistribution.GiniCoefficient)
	fmt.Printf("Power law β: %.3f (Earth: 0.390)\n\n", metrics.PowerLaw.Exponent)

	fmt.Printf("Major plates (%d): ", len(majors))
	for _, size := range majors {
		fmt.Printf("%.1f%% ", size)
	}
	fmt.Println()

	fmt.Printf("Minor plates (%d): ", len(minors))
	if len(minors) <= 15 {
		for _, size := range minors {
			fmt.Printf("%.2f%% ", size)
		}
	} else {
		for i := 0; i < 5; i++ {
			fmt.Printf("%.2f%% ", minors[i])
		}
		fmt.Printf("... ")
		for i := len(minors) - 3; i < len(minors); i++ {
			fmt.Printf("%.2f%% ", minors[i])
		}
	}
	fmt.Println()

	fmt.Printf("Micro plates (%d): ", len(micros))
	if len(micros) <= 20 {
		for _, size := range micros {
			fmt.Printf("%.3f%% ", size)
		}
	} else {
		for i := 0; i < 5; i++ {
			fmt.Printf("%.3f%% ", micros[i])
		}
		fmt.Printf("... ")
		for i := len(micros) - 3; i < len(micros); i++ {
			fmt.Printf("%.3f%% ", micros[i])
		}
	}
	fmt.Println()

	// Calculate size ratio (largest/smallest)
	if len(sizes) > 0 {
		fmt.Printf("\nSize ratio (largest/smallest): %.0fx\n", sizes[0]/sizes[len(sizes)-1])
	}

	// Check for size variation issues
	if len(majors) > 0 {
		majorRange := majors[0] - majors[len(majors)-1]
		fmt.Printf("Major plate range: %.1f%% - %.1f%% (span: %.1f%%)\n",
			majors[0], majors[len(majors)-1], majorRange)
	}
}
