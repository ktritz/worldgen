package main

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== QUOTA-BASED VS TRI-MODAL CONVECTION ===\n")

	planetRadius := 6371000.0
	subdivision := 5

	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices

	fmt.Println("Earth Target: 39 plates (7 major, 13 minor, 19 micro)")
	fmt.Println()

	// Test Quota-Based
	fmt.Println("METHOD: Quota-Based Generation")
	fmt.Println("------------------------------")
	{
		settings := tectonics.DefaultQuotaSettings()
		settings.MajorCount = 7
		settings.MinorCount = 13
		settings.MicroCount = 19
		settings.Verbose = true

		plates, _, _ := tectonics.QuotaBasedGeneration(
			voronoiCells, icosphereSites, planetRadius, settings)

		metrics := evaluatePlates(plates, planetRadius)

		fmt.Printf("\nFinal Distribution: %d major, %d minor, %d micro (total: %d)\n",
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount,
			len(plates))
		fmt.Printf("Score: %.1f%%\n", metrics.Benchmark.OverallScore*100)
		fmt.Printf("Gini: %.3f (Earth: 0.811)\n", metrics.SizeDistribution.GiniCoefficient)
		fmt.Printf("Power law β: %.3f (Earth: 0.390)\n", metrics.PowerLaw.Exponent)
		fmt.Println()
	}

	// Test Tri-Modal Convection
	fmt.Println("METHOD: Tri-Modal Convection")
	fmt.Println("-----------------------------")
	{
		settings := tectonics.DefaultConvectionSettings()
		settings.NumConvectionCells = 50
		settings.UseTriModalStrength = true
		settings.MajorCellStrength = 1.0
		settings.MinorCellStrength = 0.25
		settings.MicroCellStrength = 0.05
		settings.GrowthBudgetExponent = 3.5
		settings.GrowthIterations = 300
		settings.TargetPlateCount = 55
		settings.Seed = 12345
		settings.Verbose = false

		plates, _, _ := tectonics.GeneratePlatesFromMantleConvection(
			voronoiCells, icosphereSites, planetRadius, settings)

		metrics := evaluatePlates(plates, planetRadius)

		fmt.Printf("\nFinal Distribution: %d major, %d minor, %d micro (total: %d)\n",
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount,
			len(plates))
		fmt.Printf("Score: %.1f%%\n", metrics.Benchmark.OverallScore*100)
		fmt.Printf("Gini: %.3f (Earth: 0.811)\n", metrics.SizeDistribution.GiniCoefficient)
		fmt.Printf("Power law β: %.3f (Earth: 0.390)\n", metrics.PowerLaw.Exponent)
		fmt.Println()
	}

	fmt.Println("COMPARISON:")
	fmt.Println("- Quota-Based: Direct control, can hit any distribution")
	fmt.Println("- Tri-Modal: Physics-based, higher score but less control")
	fmt.Println()
	fmt.Println("For deterministic control → use Quota-Based")
	fmt.Println("For highest Earth-benchmark score → use Tri-Modal Convection")
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
