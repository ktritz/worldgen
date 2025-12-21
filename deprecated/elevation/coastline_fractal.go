package elevation

// coastline_fractal.go - Coastline fractal dimension measurement and enhancement

import (
	"fmt"
	"math"
)

// MeasureCoastlineFractalDimension calculates the fractal dimension of coastlines
// using a simplified box-counting method
func MeasureCoastlineFractalDimension(sites []Vector3D, elevations []float64, planetRadius float64) float64 {
	// Find coastal sites (within ±50m of sea level)
	coastalSites := make([]int, 0)
	for i, elev := range elevations {
		if math.Abs(elev) < 50.0 {
			coastalSites = append(coastalSites, i)
		}
	}

	if len(coastalSites) < 100 {
		return 1.0 // Not enough coastal sites
	}

	// Use box-counting at different scales
	// Test at scales: 100km, 50km, 25km, 12.5km
	scales := []float64{100000, 50000, 25000, 12500} // meters
	counts := make([]float64, len(scales))

	for scaleIdx, scale := range scales {
		// Count how many boxes of this size contain coastline
		boxSize := scale
		gridSize := int(math.Ceil(2.0 * math.Pi * planetRadius / boxSize))

		// Simple grid-based counting
		boxes := make(map[[2]int]bool)
		for _, siteIdx := range coastalSites {
			site := sites[siteIdx]
			// Convert to lat/lon
			lat := math.Asin(site.Z / planetRadius)
			lon := math.Atan2(site.Y, site.X)

			// Grid coordinates
			gridX := int((lon + math.Pi) / (2.0 * math.Pi) * float64(gridSize))
			gridY := int((lat + math.Pi/2.0) / math.Pi * float64(gridSize/2))

			boxes[[2]int{gridX, gridY}] = true
		}

		counts[scaleIdx] = float64(len(boxes))
	}

	// Calculate fractal dimension from slope of log(N) vs log(1/ε)
	// D ≈ -slope of log(N) vs log(ε)
	logScales := make([]float64, len(scales))
	logCounts := make([]float64, len(scales))
	for i := range scales {
		logScales[i] = math.Log(scales[i])
		logCounts[i] = math.Log(counts[i])
	}

	// Linear regression to find slope
	dimension := calculateSlope(logScales, logCounts)

	return -dimension // Negative because N decreases as ε increases
}

// calculateSlope performs simple linear regression
func calculateSlope(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}

	n := float64(len(x))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	return slope
}

// EnhanceCoastlineFractalDetail uses midpoint-displacement-inspired technique
// to create jagged, irregular coastlines by aggressively perturbing elevations near sea level
func EnhanceCoastlineFractalDetail(sites []Vector3D, currentElevations []float64, params ElevationParameters) []float64 {
	enhancements := make([]float64, len(sites))

	// Target fractal dimension: 1.3-1.5 (realistic coastlines)
	// Key insight: To change coastline SHAPE, we need perturbations large enough
	// to push land below water (+1000m → -500m) and vice versa

	for siteIdx, site := range sites {
		currentElev := currentElevations[siteIdx]
		distFromSeaLevel := math.Abs(currentElev)

		// Apply to sites within ±1000m of sea level (coastal zone)
		if distFromSeaLevel > 1000.0 {
			continue
		}

		// Strong proximity weight - maximum effect right at coastline
		proximityWeight := 1.0 - (distFromSeaLevel / 1000.0)
		proximityWeight = math.Pow(proximityWeight, 0.8) // Gentle falloff for wide effect

		// Apply fractal Brownian motion with Hurst exponent H=0.5 (coastline roughness)
		// Starting amplitude MUST be large enough to cross sea level (±800m)
		totalPerturbation := 0.0
		amplitude := 800.0 // Start with ±800m - large enough to drastically reshape coastline
		frequency := params.NoiseScale * 20.0 // High frequency for fine detail
		H := 0.5                               // Hurst exponent for coastline fractals

		// 6 octaves of fractal noise
		for octave := 0; octave < 6; octave++ {
			noiseVal := perlinNoise3D(
				site.X*frequency,
				site.Y*frequency,
				site.Z*frequency,
				params.ElevationSeed+int64(7000+octave),
			)

			totalPerturbation += noiseVal * amplitude

			// Decrease amplitude by 2^(-H) for fractal character
			frequency *= 2.0
			amplitude *= math.Pow(2.0, -H)
		}

		// Apply perturbation weighted by proximity to coastline
		// This will create peninsulas, bays, fjords, islands
		enhancements[siteIdx] = totalPerturbation * proximityWeight
	}

	return enhancements
}

// PrintCoastlineFractalReport outputs coastline fractal dimension analysis
func PrintCoastlineFractalReport(sites []Vector3D, elevations []float64, planetRadius float64) {
	dimension := MeasureCoastlineFractalDimension(sites, elevations, planetRadius)

	fmt.Printf("  Coastline Fractal Analysis:\n")
	fmt.Printf("    Measured fractal dimension: %.3f\n", dimension)
	fmt.Printf("    Interpretation:\n")

	if dimension < 1.15 {
		fmt.Printf("      TOO SMOOTH - coastlines are too straight (like desert coasts)\n")
		fmt.Printf("      Target: 1.3-1.5 for realistic coastlines\n")
	} else if dimension < 1.25 {
		fmt.Printf("      SMOOTH - wave-modified coastlines (California: 1.1-1.2)\n")
		fmt.Printf("      Could use more detail for realistic fractal character\n")
	} else if dimension < 1.4 {
		fmt.Printf("      REALISTIC - good fractal character (typical coastlines: 1.2-1.4)\n")
	} else if dimension < 1.6 {
		fmt.Printf("      COMPLEX - glacially carved coastlines (Norway/Maine: ~1.5)\n")
	} else {
		fmt.Printf("      VERY COMPLEX - extremely irregular coastlines\n")
	}

	fmt.Printf("    Reference fractal dimensions:\n")
	fmt.Printf("      Smooth line: 1.0\n")
	fmt.Printf("      California coast: 1.1-1.2\n")
	fmt.Printf("      Typical Earth coast: 1.3\n")
	fmt.Printf("      Norwegian fjords: 1.5\n")
}
