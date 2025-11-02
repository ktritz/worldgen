package main

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== QUICK TRI-MODAL TEST: FEWER CELLS ===\n")

	planetRadius := 6371000.0
	subdivision := 5

	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices

	// Try with 40 cells instead of 50 (7+13+20)
	configs := []struct {
		cells int
		desc  string
	}{
		{40, "40 cells (7+13+20)"},
		{45, "45 cells (7+13+25)"},
		{35, "35 cells (7+13+15)"},
		{30, "30 cells (7+13+10)"},
	}

	for _, cfg := range configs {
		settings := tectonics.DefaultConvectionSettings()
		settings.NumConvectionCells = cfg.cells
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

		maj7 := ""
		if metrics.SizeDistribution.MajorCount == 7 {
			maj7 = "✓"
		}

		fmt.Printf("%s: %d plates (%d/%d/%d) - %.1f%% %s\n",
			cfg.desc,
			len(plates),
			metrics.SizeDistribution.MajorCount,
			metrics.SizeDistribution.MinorCount,
			metrics.SizeDistribution.MicroCount,
			metrics.Benchmark.OverallScore*100,
			maj7)
	}
}
