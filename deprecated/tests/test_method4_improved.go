package main

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== TESTING IMPROVED METHOD 4 ===\n")

	planetRadius := 6371000.0
	subdivision := 5

	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices

	settings := tectonics.DefaultMethod4Settings()
	settings.TargetPlateCount = 39
	settings.OverseedingFactor = 15
	settings.Seed = 12345

	plates, _, _ := tectonics.Method4ModifiedAccretion(
		voronoiCells, icosphereSites, planetRadius, settings)

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

	fmt.Printf("Plates: %d total (%d major, %d minor, %d micro)\n",
		len(plates),
		metrics.SizeDistribution.MajorCount,
		metrics.SizeDistribution.MinorCount,
		metrics.SizeDistribution.MicroCount)
	fmt.Printf("Score: %.1f%%\n", metrics.Benchmark.OverallScore*100)

	if metrics.SizeDistribution.MajorCount == 7 {
		fmt.Println("✓ 7 majors achieved!")
	}
	if metrics.SizeDistribution.MicroCount >= 15 {
		fmt.Println("✓ Micro plates generated!")
	}
	if metrics.Benchmark.OverallScore >= 0.75 {
		fmt.Println("✓✓✓ TARGET SCORE ACHIEVED!")
	}
}
