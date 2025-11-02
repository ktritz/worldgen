package main

// test_convection_cells.go - Test convection cell generation and power law distribution

import (
	"fmt"
	"math"
	"sort"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
)

func main() {
	fmt.Println("=== TESTING CONVECTION CELL GENERATION ===")
	fmt.Println()

	// Generate a simple icosphere for testing
	subdivision := 4 // Lower subdivision for quick testing
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices

	planetRadius := 6371000.0 // Earth radius in meters

	fmt.Printf("Test icosphere: %d sites\n", len(icosphereSites))
	fmt.Printf("Planet radius: %.0f km\n\n", planetRadius/1000)

	// Test different power law exponents
	exponents := []float64{0.3, 0.39, 0.5, 1.0, 1.5}

	for _, exponent := range exponents {
		fmt.Printf("=== Testing β = %.2f ===\n", exponent)

		settings := tectonics.DefaultConvectionSettings()
		settings.PowerLawExponent = exponent
		settings.NumConvectionCells = 30
		settings.Verbose = false

		// Generate test plates using convection method
		plates, _, err := tectonics.GeneratePlatesFromMantleConvection(
			voronoiCells,
			icosphereSites,
			planetRadius,
			settings,
		)

		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
			continue
		}

		// Analyze the distribution
		if len(plates) == 0 {
			fmt.Println("Warning: No plates generated (implementation incomplete)")
			fmt.Println()
			continue
		}

		analyzePlateDistribution(plates, planetRadius, exponent)
		fmt.Println()
	}

	fmt.Println("=== Convection Cell Test Complete ===")
}

func analyzePlateDistribution(plates []tectonics.TectonicPlate, planetRadius float64, targetExponent float64) {
	if len(plates) == 0 {
		return
	}

	// Calculate sizes as percentages
	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius
	sizes := make([]float64, len(plates))
	for i, plate := range plates {
		sizes[i] = plate.Area / sphereArea * 100.0
	}

	// Sort sizes descending
	sort.Float64s(sizes)
	for i, j := 0, len(sizes)-1; i < j; i, j = i+1, j-1 {
		sizes[i], sizes[j] = sizes[j], sizes[i]
	}

	// Calculate power law fit
	exponent, r2 := fitPowerLaw(sizes)

	// Calculate Gini coefficient
	gini := calculateGini(sizes)

	// Calculate size ratio
	sizeRatio := sizes[0] / sizes[len(sizes)-1]

	// Print results
	fmt.Printf("  Generated %d plates\n", len(plates))
	fmt.Printf("  Largest: %.2f%%, Smallest: %.3f%%\n", sizes[0], sizes[len(sizes)-1])
	fmt.Printf("  Size ratio: %.0fx\n", sizeRatio)
	fmt.Printf("  Gini coefficient: %.3f (Earth: 0.811)\n", gini)
	fmt.Printf("  Power law fit: β=%.3f, R²=%.3f\n", exponent, r2)
	fmt.Printf("  Target β: %.3f, Error: %+.3f\n", targetExponent, exponent-targetExponent)

	// Quality assessment
	exponentError := math.Abs(exponent - targetExponent)
	if exponentError < 0.05 {
		fmt.Println("  ✓ Excellent exponent match!")
	} else if exponentError < 0.15 {
		fmt.Println("  ○ Good exponent match")
	} else {
		fmt.Println("  △ Exponent needs tuning")
	}
}

func fitPowerLaw(sizes []float64) (float64, float64) {
	// Log-log linear regression: log(P(x)) = -β * log(x) + C
	// Using rank-size: rank = C * size^(-β)

	if len(sizes) < 2 {
		return 0, 0
	}

	n := len(sizes)
	logRanks := make([]float64, n)
	logSizes := make([]float64, n)

	for i := 0; i < n; i++ {
		logRanks[i] = math.Log(float64(i + 1))
		logSizes[i] = math.Log(sizes[i])
	}

	// Linear regression
	meanLogRank := 0.0
	meanLogSize := 0.0
	for i := 0; i < n; i++ {
		meanLogRank += logRanks[i]
		meanLogSize += logSizes[i]
	}
	meanLogRank /= float64(n)
	meanLogSize /= float64(n)

	numerator := 0.0
	denominator := 0.0
	for i := 0; i < n; i++ {
		numerator += (logSizes[i] - meanLogSize) * (logRanks[i] - meanLogRank)
		denominator += (logSizes[i] - meanLogSize) * (logSizes[i] - meanLogSize)
	}

	if denominator == 0 {
		return 0, 0
	}

	slope := numerator / denominator
	exponent := -slope

	// Calculate R²
	predicted := make([]float64, n)
	intercept := meanLogRank - slope*meanLogSize
	for i := 0; i < n; i++ {
		predicted[i] = slope*logSizes[i] + intercept
	}

	ssRes := 0.0
	ssTot := 0.0
	for i := 0; i < n; i++ {
		ssRes += (logRanks[i] - predicted[i]) * (logRanks[i] - predicted[i])
		ssTot += (logRanks[i] - meanLogRank) * (logRanks[i] - meanLogRank)
	}

	r2 := 1.0 - ssRes/ssTot

	return exponent, r2
}

func calculateGini(sizes []float64) float64 {
	if len(sizes) == 0 {
		return 0
	}

	// Sort ascending for Gini calculation
	sorted := make([]float64, len(sizes))
	copy(sorted, sizes)
	sort.Float64s(sorted)

	n := float64(len(sorted))
	sum := 0.0
	weightedSum := 0.0

	for i, val := range sorted {
		sum += val
		weightedSum += val * float64(i+1)
	}

	if sum == 0 {
		return 0
	}

	return (2.0*weightedSum)/(n*sum) - (n+1.0)/n
}
