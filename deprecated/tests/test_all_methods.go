package main

// test_all_methods.go - Comprehensive comparison of all 7 plate generation methods

import (
	"fmt"
	"math"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
	"worldgen/landgen/tectonics/evaluation"
)

func main() {
	fmt.Println("=== COMPREHENSIVE PLATE GENERATION METHOD COMPARISON ===")
	fmt.Println()
	fmt.Println("Research-based expected scores:")
	fmt.Println("  Method 1 (Hierarchical Voronoi):     50-65%")
	fmt.Println("  Method 2 (Stochastic Fracturing):    55-70%")
	fmt.Println("  Method 3 (Mantle Convection):        70-85%")
	fmt.Println("  Method 4 (Modified Accretion):       60-75%")
	fmt.Println("  Method 5 (Cellular Automata):        55-70%")
	fmt.Println("  Method 6 (Graph Partition):          65-80%")
	fmt.Println("  Method 7 (Hybrid Optimization):      75-90%")
	fmt.Println()
	fmt.Println("Target: 75%+ Earth benchmark score")
	fmt.Println("Target Distribution: 39 plates (7 major, 13 minor, 19 micro)")
	fmt.Println()

	planetRadius := 6371000.0
	subdivision := 5

	fmt.Println("Generating icosphere...")
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices
	fmt.Printf("  Generated %d sites\n\n", len(icosphereSites))

	type methodResult struct {
		name         string
		plates       []tectonics.TectonicPlate
		score        float64
		majorCount   int
		minorCount   int
		microCount   int
		gini         float64
		powerLawBeta float64
		error        error
	}

	results := make([]methodResult, 0)

	// Method 1: Hierarchical Voronoi
	fmt.Println("Testing Method 1: Hierarchical Voronoi...")
	{
		settings := tectonics.DefaultMethod1Settings()
		settings.TargetPlateCount = 39
		settings.PowerLawExponent = 0.39
		settings.Seed = 12345

		plates, _, err := tectonics.Method1HierarchicalVoronoi(
			voronoiCells, icosphereSites, planetRadius, settings)

		if err != nil {
			fmt.Printf("  ERROR: %v\n\n", err)
			results = append(results, methodResult{name: "Method 1", error: err})
		} else {
			metrics := evaluatePlates(plates, planetRadius)
			results = append(results, methodResult{
				name:         "Method 1: Hierarchical Voronoi",
				plates:       plates,
				score:        metrics.Benchmark.OverallScore,
				majorCount:   metrics.SizeDistribution.MajorCount,
				minorCount:   metrics.SizeDistribution.MinorCount,
				microCount:   metrics.SizeDistribution.MicroCount,
				gini:         metrics.SizeDistribution.GiniCoefficient,
				powerLawBeta: metrics.PowerLaw.Exponent,
			})
			fmt.Printf("  Score: %.1f%% | Distribution: %d/%d/%d\n\n",
				metrics.Benchmark.OverallScore*100,
				metrics.SizeDistribution.MajorCount,
				metrics.SizeDistribution.MinorCount,
				metrics.SizeDistribution.MicroCount)
		}
	}

	// Method 2: Stochastic Fracturing
	fmt.Println("Testing Method 2: Stochastic Fracturing...")
	{
		settings := tectonics.DefaultMethod2Settings()
		settings.TargetPlateCount = 39
		settings.PowerLawExponent = 1.5
		settings.Seed = 12345

		plates, _, err := tectonics.Method2StochasticFracturing(
			voronoiCells, icosphereSites, planetRadius, settings)

		if err != nil {
			fmt.Printf("  ERROR: %v\n\n", err)
			results = append(results, methodResult{name: "Method 2", error: err})
		} else {
			metrics := evaluatePlates(plates, planetRadius)
			results = append(results, methodResult{
				name:         "Method 2: Stochastic Fracturing",
				plates:       plates,
				score:        metrics.Benchmark.OverallScore,
				majorCount:   metrics.SizeDistribution.MajorCount,
				minorCount:   metrics.SizeDistribution.MinorCount,
				microCount:   metrics.SizeDistribution.MicroCount,
				gini:         metrics.SizeDistribution.GiniCoefficient,
				powerLawBeta: metrics.PowerLaw.Exponent,
			})
			fmt.Printf("  Score: %.1f%% | Distribution: %d/%d/%d\n\n",
				metrics.Benchmark.OverallScore*100,
				metrics.SizeDistribution.MajorCount,
				metrics.SizeDistribution.MinorCount,
				metrics.SizeDistribution.MicroCount)
		}
	}

	// Method 3: Mantle Convection (with TRI-MODAL distribution)
	fmt.Println("Testing Method 3: Mantle Convection (Tri-Modal)...")
	{
		settings := tectonics.DefaultConvectionSettings()
		settings.NumConvectionCells = 50  // 50 cells: 7 major + 13 minor + 30 micro
		settings.UseTriModalStrength = true
		settings.MajorCellStrength = 1.0   // High strength for 7 major plates
		settings.MinorCellStrength = 0.25  // Medium strength for 13 minor plates
		settings.MicroCellStrength = 0.05  // Low strength for micro plates
		settings.GrowthBudgetExponent = 3.5  // Less extreme than before
		settings.GrowthIterations = 300
		settings.TargetPlateCount = 55
		settings.Seed = 12345

		plates, _, err := tectonics.GeneratePlatesFromMantleConvection(
			voronoiCells, icosphereSites, planetRadius, settings)

		if err != nil {
			fmt.Printf("  ERROR: %v\n\n", err)
			results = append(results, methodResult{name: "Method 3", error: err})
		} else {
			metrics := evaluatePlates(plates, planetRadius)
			results = append(results, methodResult{
				name:         "Method 3: Mantle Convection",
				plates:       plates,
				score:        metrics.Benchmark.OverallScore,
				majorCount:   metrics.SizeDistribution.MajorCount,
				minorCount:   metrics.SizeDistribution.MinorCount,
				microCount:   metrics.SizeDistribution.MicroCount,
				gini:         metrics.SizeDistribution.GiniCoefficient,
				powerLawBeta: metrics.PowerLaw.Exponent,
			})
			fmt.Printf("  Score: %.1f%% | Distribution: %d/%d/%d\n\n",
				metrics.Benchmark.OverallScore*100,
				metrics.SizeDistribution.MajorCount,
				metrics.SizeDistribution.MinorCount,
				metrics.SizeDistribution.MicroCount)
		}
	}

	// Method 4: Modified Accretion
	fmt.Println("Testing Method 4: Modified Accretion...")
	{
		settings := tectonics.DefaultMethod4Settings()
		settings.TargetPlateCount = 39
		settings.OverseedingFactor = 15
		settings.Seed = 12345

		plates, _, err := tectonics.Method4ModifiedAccretion(
			voronoiCells, icosphereSites, planetRadius, settings)

		if err != nil {
			fmt.Printf("  ERROR: %v\n\n", err)
			results = append(results, methodResult{name: "Method 4", error: err})
		} else {
			metrics := evaluatePlates(plates, planetRadius)
			results = append(results, methodResult{
				name:         "Method 4: Modified Accretion",
				plates:       plates,
				score:        metrics.Benchmark.OverallScore,
				majorCount:   metrics.SizeDistribution.MajorCount,
				minorCount:   metrics.SizeDistribution.MinorCount,
				microCount:   metrics.SizeDistribution.MicroCount,
				gini:         metrics.SizeDistribution.GiniCoefficient,
				powerLawBeta: metrics.PowerLaw.Exponent,
			})
			fmt.Printf("  Score: %.1f%% | Distribution: %d/%d/%d\n\n",
				metrics.Benchmark.OverallScore*100,
				metrics.SizeDistribution.MajorCount,
				metrics.SizeDistribution.MinorCount,
				metrics.SizeDistribution.MicroCount)
		}
	}

	// Method 5: Cellular Automata
	fmt.Println("Testing Method 5: Cellular Automata...")
	{
		settings := tectonics.DefaultMethod5Settings()
		settings.TargetPlateCount = 39
		settings.PowerLawExponent = 0.39
		settings.BudgetMultiplier = 1.5
		settings.Iterations = 200
		settings.Seed = 12345

		plates, _, err := tectonics.Method5CellularAutomata(
			voronoiCells, icosphereSites, planetRadius, settings)

		if err != nil {
			fmt.Printf("  ERROR: %v\n\n", err)
			results = append(results, methodResult{name: "Method 5", error: err})
		} else {
			metrics := evaluatePlates(plates, planetRadius)
			results = append(results, methodResult{
				name:         "Method 5: Cellular Automata",
				plates:       plates,
				score:        metrics.Benchmark.OverallScore,
				majorCount:   metrics.SizeDistribution.MajorCount,
				minorCount:   metrics.SizeDistribution.MinorCount,
				microCount:   metrics.SizeDistribution.MicroCount,
				gini:         metrics.SizeDistribution.GiniCoefficient,
				powerLawBeta: metrics.PowerLaw.Exponent,
			})
			fmt.Printf("  Score: %.1f%% | Distribution: %d/%d/%d\n\n",
				metrics.Benchmark.OverallScore*100,
				metrics.SizeDistribution.MajorCount,
				metrics.SizeDistribution.MinorCount,
				metrics.SizeDistribution.MicroCount)
		}
	}

	// Method 6: Graph Partition
	fmt.Println("Testing Method 6: Graph Partition...")
	{
		settings := tectonics.DefaultMethod6Settings()
		settings.TargetPlateCount = 39
		settings.PowerLawExponent = 0.39
		settings.RefinementIterations = 100
		settings.Seed = 12345

		plates, _, err := tectonics.Method6GraphPartition(
			voronoiCells, icosphereSites, planetRadius, settings)

		if err != nil {
			fmt.Printf("  ERROR: %v\n\n", err)
			results = append(results, methodResult{name: "Method 6", error: err})
		} else {
			metrics := evaluatePlates(plates, planetRadius)
			results = append(results, methodResult{
				name:         "Method 6: Graph Partition",
				plates:       plates,
				score:        metrics.Benchmark.OverallScore,
				majorCount:   metrics.SizeDistribution.MajorCount,
				minorCount:   metrics.SizeDistribution.MinorCount,
				microCount:   metrics.SizeDistribution.MicroCount,
				gini:         metrics.SizeDistribution.GiniCoefficient,
				powerLawBeta: metrics.PowerLaw.Exponent,
			})
			fmt.Printf("  Score: %.1f%% | Distribution: %d/%d/%d\n\n",
				metrics.Benchmark.OverallScore*100,
				metrics.SizeDistribution.MajorCount,
				metrics.SizeDistribution.MinorCount,
				metrics.SizeDistribution.MicroCount)
		}
	}

	// Method 7: Hybrid (Convection + Optimization) with TRI-MODAL
	fmt.Println("Testing Method 7: Hybrid (Tri-Modal + Optimization)...")
	{
		settings := tectonics.DefaultHybridSettings()
		settings.Verbose = false
		settings.RefinementIterations = 150

		// Use tri-modal convection
		settings.ConvectionSettings.NumConvectionCells = 50
		settings.ConvectionSettings.UseTriModalStrength = true
		settings.ConvectionSettings.MajorCellStrength = 1.0
		settings.ConvectionSettings.MinorCellStrength = 0.25
		settings.ConvectionSettings.MicroCellStrength = 0.05
		settings.ConvectionSettings.GrowthBudgetExponent = 3.5
		settings.ConvectionSettings.GrowthIterations = 300
		settings.ConvectionSettings.TargetPlateCount = 55
		settings.ConvectionSettings.Seed = 12345

		plates, _, err := tectonics.HybridPlateGeneration(
			voronoiCells, icosphereSites, planetRadius, settings)

		if err != nil {
			fmt.Printf("  ERROR: %v\n\n", err)
			results = append(results, methodResult{name: "Method 7", error: err})
		} else {
			metrics := evaluatePlates(plates, planetRadius)
			results = append(results, methodResult{
				name:         "Method 7: Hybrid Optimization",
				plates:       plates,
				score:        metrics.Benchmark.OverallScore,
				majorCount:   metrics.SizeDistribution.MajorCount,
				minorCount:   metrics.SizeDistribution.MinorCount,
				microCount:   metrics.SizeDistribution.MicroCount,
				gini:         metrics.SizeDistribution.GiniCoefficient,
				powerLawBeta: metrics.PowerLaw.Exponent,
			})
			fmt.Printf("  Score: %.1f%% | Distribution: %d/%d/%d\n\n",
				metrics.Benchmark.OverallScore*100,
				metrics.SizeDistribution.MajorCount,
				metrics.SizeDistribution.MinorCount,
				metrics.SizeDistribution.MicroCount)
		}
	}

	// Print summary table
	fmt.Println()
	fmt.Println("=== SUMMARY COMPARISON ===")
	fmt.Println()
	fmt.Println("Method                        | Score  | Plates (Maj/Min/Mic) | Gini  | β     | 7 Maj? | Target?")
	fmt.Println("------------------------------|--------|----------------------|-------|-------|--------|--------")

	for _, result := range results {
		if result.error != nil {
			fmt.Printf("%-29s | ERROR\n", result.name)
			continue
		}

		has7Majors := " "
		if result.majorCount == 7 {
			has7Majors = "✓"
		}

		meetsTarget := " "
		if result.score >= 0.75 {
			meetsTarget = "✓✓✓"
		} else if result.score >= 0.70 {
			meetsTarget = "○"
		}

		fmt.Printf("%-29s | %5.1f%% | %2d (%d/%d/%d)         | %.3f | %.3f | %-6s | %-7s\n",
			result.name,
			result.score*100,
			len(result.plates),
			result.majorCount,
			result.minorCount,
			result.microCount,
			result.gini,
			result.powerLawBeta,
			has7Majors,
			meetsTarget)
	}

	fmt.Println()
	fmt.Println("Earth Reference:              |        | 39 (7/13/19)         | 0.811 | 0.390 | ✓      |")
	fmt.Println()

	// Find best method
	bestIdx := -1
	bestScore := 0.0
	for i, result := range results {
		if result.error == nil && result.score > bestScore {
			bestScore = result.score
			bestIdx = i
		}
	}

	if bestIdx >= 0 {
		fmt.Println("=== BEST METHOD ===")
		fmt.Printf("Method: %s\n", results[bestIdx].name)
		fmt.Printf("Score: %.1f%%\n", results[bestIdx].score*100)
		fmt.Printf("Distribution: %d major, %d minor, %d micro (total: %d)\n",
			results[bestIdx].majorCount,
			results[bestIdx].minorCount,
			results[bestIdx].microCount,
			len(results[bestIdx].plates))
		fmt.Println()

		if bestScore >= 0.75 {
			fmt.Println("✓✓✓ TARGET ACHIEVED! Score >= 75%")
		} else {
			fmt.Printf("Gap to target: -%.1f%%\n", 75.0-bestScore*100)
			fmt.Println("Further optimization needed.")
		}
	}
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

	return evaluation.CalculateDetailedMetrics(
		plateSizes,
		plateTypes,
		plateCentroids,
		sphereArea,
		planetRadius,
	)
}
