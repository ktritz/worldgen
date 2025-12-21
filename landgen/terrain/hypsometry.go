package terrain

import (
	"sort"
)

// Earth's hypsometric curve data (elevation in meters, cumulative fraction)
// Data from standard hypsometric curves - fraction of Earth's surface BELOW this elevation
var earthHypsometry = []struct {
	elevation float64
	fraction  float64
}{
	{-10000, 0.00},
	{-9000, 0.001},
	{-8000, 0.004},
	{-7000, 0.01},
	{-6000, 0.04},
	{-5500, 0.10},
	{-5000, 0.20},
	{-4500, 0.35},
	{-4000, 0.50},
	{-3500, 0.58},
	{-3000, 0.62},
	{-2500, 0.65},
	{-2000, 0.67},
	{-1500, 0.68},
	{-1000, 0.69},
	{-500, 0.70},
	{-200, 0.705},
	{0, 0.71},      // Sea level - 71% of Earth is ocean
	{200, 0.78},
	{500, 0.85},
	{1000, 0.92},
	{1500, 0.96},
	{2000, 0.98},
	{3000, 0.99},
	{4000, 0.995},
	{5000, 0.998},
	{6000, 0.999},
	{8848, 1.00},   // Everest
}

// ApplyEarthHypsometry remaps elevations to match Earth's hypsometric distribution
// while allowing control over land fraction.
//
// targetLandFraction: desired fraction of land (0.29 for Earth-like, 0.5 for 50% land)
//
// The algorithm:
// 1. Find the elevation that gives targetLandFraction as our "sea level"
// 2. Map our below-sea-level elevations to Earth's ocean depths (0-71% of Earth)
// 3. Map our above-sea-level elevations to Earth's land elevations (71-100% of Earth)
func ApplyEarthHypsometry(elevation []float64, targetLandFraction float64) []float64 {
	n := len(elevation)
	result := make([]float64, n)

	// Sort elevations to find percentiles
	sorted := make([]float64, n)
	copy(sorted, elevation)
	sort.Float64s(sorted)

	// Find sea level that gives target land fraction
	oceanFraction := 1.0 - targetLandFraction
	seaLevelIdx := int(oceanFraction * float64(n))
	if seaLevelIdx >= n {
		seaLevelIdx = n - 1
	}
	seaLevel := sorted[seaLevelIdx]

	// Earth's sea level is at 71% ocean (fraction 0.71)
	earthOceanFraction := 0.71

	// Map each elevation
	for i, e := range elevation {
		// Find this elevation's rank in our distribution
		rank := sort.SearchFloat64s(sorted, e)
		ourFraction := float64(rank) / float64(n)

		var earthFraction float64
		if e <= seaLevel {
			// Ocean: map our [0, oceanFraction] to Earth's [0, 0.71]
			if oceanFraction > 0 {
				normalizedOcean := ourFraction / oceanFraction
				earthFraction = normalizedOcean * earthOceanFraction
			} else {
				earthFraction = 0
			}
		} else {
			// Land: map our [oceanFraction, 1] to Earth's [0.71, 1]
			if targetLandFraction > 0 {
				normalizedLand := (ourFraction - oceanFraction) / targetLandFraction
				earthFraction = earthOceanFraction + normalizedLand*(1-earthOceanFraction)
			} else {
				earthFraction = 1
			}
		}

		// Clamp
		if earthFraction < 0 {
			earthFraction = 0
		}
		if earthFraction > 0.9999 {
			earthFraction = 0.9999
		}

		// Look up Earth elevation for this fraction
		result[i] = earthElevationAtFraction(earthFraction)
	}

	return result
}

// earthElevationAtFraction interpolates Earth's hypsometric curve
func earthElevationAtFraction(fraction float64) float64 {
	// Find the two points to interpolate between
	for i := 1; i < len(earthHypsometry); i++ {
		if earthHypsometry[i].fraction >= fraction {
			// Interpolate between i-1 and i
			prev := earthHypsometry[i-1]
			curr := earthHypsometry[i]
			t := (fraction - prev.fraction) / (curr.fraction - prev.fraction)
			return prev.elevation + t*(curr.elevation-prev.elevation)
		}
	}
	return earthHypsometry[len(earthHypsometry)-1].elevation
}

// GetSeaLevel returns sea level (always 0 after hypsometry mapping)
func GetSeaLevel() float64 {
	return 0.0
}
