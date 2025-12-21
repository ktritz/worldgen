package terrain

// DEPRECATED: Continental mask generation is replaced by collision-based elevation.
// The new approach (collision_elevation.go) uses plate tectonics to drive elevation
// instead of noise-based continental masks.
//
// Use terrain.GenerateCollisionElevation() instead of GenerateContinentalMask().
// This file remains for reference but is no longer the recommended approach.

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"runtime"

	"worldgen/icosphere"
	"worldgen/procnoise"
)

// --- Continental Mask Generation (DEPRECATED) ---

// GenerateContinentalMask creates a continental mask using domain-warped noise.
// The mask values are in [-1, 1] range. Higher values = more continental.
// Use FindOptimalThreshold to determine the land/ocean cutoff.
func GenerateContinentalMask(sites []Vector3D, settings MaskSettings) ([]float64, error) {
	if err := settings.Validate(); err != nil {
		return nil, fmt.Errorf("invalid mask settings: %w", err)
	}

	if len(sites) == 0 {
		return nil, fmt.Errorf("no sites provided")
	}

	if settings.Verbose {
		fmt.Println("=== CONTINENTAL MASK GENERATION ===")
		fmt.Printf("  Sites: %d\n", len(sites))
		fmt.Printf("  Seed: %d\n", settings.Seed)
		fmt.Printf("  Continental Frequency: %.2f\n", settings.ContinentalFrequency)
		fmt.Printf("  Warp Amplitude: %.2f\n", settings.WarpAmplitude)
		fmt.Printf("  Warp Frequency: %.2f\n", settings.WarpFrequency)
		fmt.Printf("  Octaves: %d\n", settings.Octaves)
	}

	// Create noise generators
	continentalNoise := createContinentalNoise(settings)
	curlGen := createCurlNoiseGenerator(settings)

	// Generate mask values in parallel
	mask := generateMaskParallel(sites, continentalNoise, curlGen, settings)

	if settings.Verbose {
		min, max := maskMinMax(mask)
		fmt.Printf("  Mask range: [%.3f, %.3f]\n", min, max)
	}

	return mask, nil
}

// createContinentalNoise creates the base noise for continental shapes.
func createContinentalNoise(settings MaskSettings) *procnoise.FastNoiseLiteScalarField {
	state := procnoise.New[float32]()
	state.Seed = int(settings.Seed)
	state.Frequency = float32(settings.ContinentalFrequency)
	state.NoiseType(procnoise.OpenSimplex2)   // Method - caches function pointer
	state.FractalType(procnoise.FractalFBm)   // Method - caches function pointer
	state.Octaves = settings.Octaves          // Field - just a parameter
	state.Lacunarity = float32(settings.Lacunarity)
	state.Gain = float32(settings.Persistence)

	return procnoise.NewFastNoiseLiteScalarField(state)
}

// createCurlNoiseGenerator creates the curl noise generator for domain warping.
func createCurlNoiseGenerator(settings MaskSettings) *procnoise.CurlNoiseGenerator3D {
	// Create three independent noise fields for the curl potential
	// Each with a different seed for variation
	stateX := procnoise.New[float32]()
	stateX.Seed = int(settings.Seed + 1000)
	stateX.Frequency = float32(settings.WarpFrequency)
	stateX.NoiseType(procnoise.OpenSimplex2)

	stateY := procnoise.New[float32]()
	stateY.Seed = int(settings.Seed + 2000)
	stateY.Frequency = float32(settings.WarpFrequency)
	stateY.NoiseType(procnoise.OpenSimplex2)

	stateZ := procnoise.New[float32]()
	stateZ.Seed = int(settings.Seed + 3000)
	stateZ.Frequency = float32(settings.WarpFrequency)
	stateZ.NoiseType(procnoise.OpenSimplex2)

	potentialX := procnoise.NewFastNoiseLiteScalarField(stateX)
	potentialY := procnoise.NewFastNoiseLiteScalarField(stateY)
	potentialZ := procnoise.NewFastNoiseLiteScalarField(stateZ)

	epsilon := 0.001 // Small step for derivative approximation
	return procnoise.NewCurlNoiseGenerator3D(potentialX, potentialY, potentialZ, epsilon)
}

// generateMaskParallel generates mask values for all sites using parallel processing.
func generateMaskParallel(sites []Vector3D, noise *procnoise.FastNoiseLiteScalarField,
	curlGen *procnoise.CurlNoiseGenerator3D, settings MaskSettings) []float64 {

	mask := make([]float64, len(sites))
	numWorkers := runtime.NumCPU()

	var wg sync.WaitGroup
	chunkSize := (len(sites) + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			start := workerID * chunkSize
			end := start + chunkSize
			if end > len(sites) {
				end = len(sites)
			}

			for i := start; i < end; i++ {
				site := sites[i]

				// Apply domain warping
				warped := applyDomainWarp(site, curlGen, settings.WarpAmplitude)

				// Sample continental noise at warped position
				value := noise.GetNoise(warped)
				mask[i] = float64(value)
			}
		}(w)
	}

	wg.Wait()
	return mask
}

// applyDomainWarp displaces a position using curl noise for irregular boundaries.
func applyDomainWarp(pos Vector3D, curlGen *procnoise.CurlNoiseGenerator3D, amplitude float64) Vector3D {
	if amplitude <= 0 {
		return pos
	}

	// Get curl noise at this position
	curl := curlGen.GetTangentCurl(pos)

	// Apply warp
	warped := icosphere.Vector3D{
		X: pos.X + curl.X*amplitude,
		Y: pos.Y + curl.Y*amplitude,
		Z: pos.Z + curl.Z*amplitude,
	}

	// Re-project onto sphere
	return warped.Normalize()
}

// maskMinMax returns the min and max values in the mask.
func maskMinMax(mask []float64) (min, max float64) {
	if len(mask) == 0 {
		return 0, 0
	}
	min, max = mask[0], mask[0]
	for _, v := range mask {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return
}

// --- Threshold Calibration ---

// FindOptimalThreshold finds the threshold that achieves the target land coverage.
// Uses binary search for efficiency.
func FindOptimalThreshold(mask []float64, targetLandPct float64) float64 {
	if len(mask) == 0 {
		return 0
	}

	// Find the actual min/max of the mask
	minVal, maxVal := maskMinMax(mask)

	// Binary search for threshold
	low, high := minVal, maxVal
	tolerance := 0.0001 // 0.01% accuracy

	for high-low > tolerance {
		mid := (low + high) / 2
		landPct := countAboveThreshold(mask, mid)

		if landPct > targetLandPct {
			low = mid // Need higher threshold to reduce land
		} else {
			high = mid // Need lower threshold to increase land
		}
	}

	return (low + high) / 2
}

// countAboveThreshold returns the fraction of mask values above the threshold.
func countAboveThreshold(mask []float64, threshold float64) float64 {
	if len(mask) == 0 {
		return 0
	}

	count := 0
	for _, v := range mask {
		if v > threshold {
			count++
		}
	}
	return float64(count) / float64(len(mask))
}

// --- Mask to Binary Conversion ---

// MaskToBinary converts a continuous mask to binary land/ocean classification.
// Returns a slice where true = land, false = ocean.
func MaskToBinary(mask []float64, threshold float64) []bool {
	binary := make([]bool, len(mask))
	for i, v := range mask {
		binary[i] = v > threshold
	}
	return binary
}

// MaskToLandOcean converts a continuous mask to discrete land/ocean values.
// Land sites get value 1.0, ocean sites get value 0.0.
func MaskToLandOcean(mask []float64, threshold float64) []float64 {
	result := make([]float64, len(mask))
	for i, v := range mask {
		if v > threshold {
			result[i] = 1.0
		} else {
			result[i] = 0.0
		}
	}
	return result
}

// --- Mask Statistics ---

// MaskStats contains statistics about the continental mask.
type MaskStats struct {
	Min          float64
	Max          float64
	Mean         float64
	StdDev       float64
	Threshold    float64 // Optimal threshold for target land coverage
	LandCoverage float64 // Actual land coverage with that threshold
}

// ComputeMaskStats computes statistics about the mask and finds optimal threshold.
func ComputeMaskStats(mask []float64, targetLandPct float64) MaskStats {
	if len(mask) == 0 {
		return MaskStats{}
	}

	min, max := maskMinMax(mask)

	// Compute mean
	sum := 0.0
	for _, v := range mask {
		sum += v
	}
	mean := sum / float64(len(mask))

	// Compute std dev
	sumSq := 0.0
	for _, v := range mask {
		diff := v - mean
		sumSq += diff * diff
	}
	stdDev := math.Sqrt(sumSq / float64(len(mask)))

	// Find optimal threshold
	threshold := FindOptimalThreshold(mask, targetLandPct)
	landCoverage := countAboveThreshold(mask, threshold)

	return MaskStats{
		Min:          min,
		Max:          max,
		Mean:         mean,
		StdDev:       stdDev,
		Threshold:    threshold,
		LandCoverage: landCoverage,
	}
}

// --- Mask Normalization ---

// NormalizeMask rescales mask values to [0, 1] range.
func NormalizeMask(mask []float64) []float64 {
	if len(mask) == 0 {
		return mask
	}

	min, max := maskMinMax(mask)
	rangeVal := max - min
	if rangeVal == 0 {
		rangeVal = 1 // Avoid division by zero
	}

	normalized := make([]float64, len(mask))
	for i, v := range mask {
		normalized[i] = (v - min) / rangeVal
	}
	return normalized
}

// --- Continent Detection ---

// ContinentInfo holds information about a detected continent.
type ContinentInfo struct {
	ID        int       // Continent identifier
	Size      int       // Number of sites
	Fraction  float64   // Fraction of total land area
	Centroid  Vector3D  // Approximate centroid
}

// DetectContinents identifies separate landmasses in the mask.
// Returns continent assignments for each site (-1 = ocean) and continent info.
// NOTE: This is a placeholder - full implementation requires mesh connectivity.
func DetectContinents(sites []Vector3D, mask []float64, threshold float64) ([]int, []ContinentInfo) {
	// Simple placeholder: assign all land to one continent
	// Full implementation would use flood-fill on mesh connectivity

	assignments := make([]int, len(mask))
	landCount := 0
	centroid := Vector3D{}

	for i, v := range mask {
		if v > threshold {
			assignments[i] = 0 // All land is continent 0
			landCount++
			centroid.X += sites[i].X
			centroid.Y += sites[i].Y
			centroid.Z += sites[i].Z
		} else {
			assignments[i] = -1 // Ocean
		}
	}

	if landCount == 0 {
		return assignments, nil
	}

	// Normalize centroid
	centroid.X /= float64(landCount)
	centroid.Y /= float64(landCount)
	centroid.Z /= float64(landCount)
	centroid = centroid.Normalize()

	totalSites := float64(len(mask))
	continents := []ContinentInfo{
		{
			ID:       0,
			Size:     landCount,
			Fraction: float64(landCount) / totalSites,
			Centroid: centroid,
		},
	}

	return assignments, continents
}

// --- Mask Smoothing ---

// SmoothMaskValues applies a simple smoothing pass to the mask.
// This uses the sorted percentile approach to reduce extreme values.
func SmoothMaskValues(mask []float64, percentileCutoff float64) []float64 {
	if len(mask) == 0 || percentileCutoff <= 0 || percentileCutoff >= 50 {
		return mask
	}

	// Sort to find percentile cutoffs
	sorted := make([]float64, len(mask))
	copy(sorted, mask)
	sort.Float64s(sorted)

	lowIdx := int(percentileCutoff / 100 * float64(len(sorted)))
	highIdx := len(sorted) - 1 - lowIdx

	lowCutoff := sorted[lowIdx]
	highCutoff := sorted[highIdx]

	// Clamp values
	smoothed := make([]float64, len(mask))
	for i, v := range mask {
		smoothed[i] = Clamp(v, lowCutoff, highCutoff)
	}

	return smoothed
}

// --- Convenience Functions ---

// GenerateContinentalMaskWithThreshold generates a mask and finds the optimal threshold.
// Returns the mask, threshold, and actual land coverage achieved.
func GenerateContinentalMaskWithThreshold(sites []Vector3D, settings MaskSettings) (mask []float64, threshold float64, landCoverage float64, err error) {
	mask, err = GenerateContinentalMask(sites, settings)
	if err != nil {
		return nil, 0, 0, err
	}

	threshold = FindOptimalThreshold(mask, settings.TargetLandCoverage)
	landCoverage = countAboveThreshold(mask, threshold)

	if settings.Verbose {
		fmt.Printf("  Threshold: %.4f\n", threshold)
		fmt.Printf("  Land Coverage: %.1f%% (target: %.1f%%)\n",
			landCoverage*100, settings.TargetLandCoverage*100)
	}

	return mask, threshold, landCoverage, nil
}

// DefaultMaskWithEarthCoverage generates a mask with default settings targeting Earth's land coverage.
func DefaultMaskWithEarthCoverage(sites []Vector3D, seed int64) (mask []float64, threshold float64, err error) {
	settings := DefaultMaskSettings()
	settings.Seed = seed
	settings.TargetLandCoverage = EarthLandCoverage

	mask, threshold, _, err = GenerateContinentalMaskWithThreshold(sites, settings)
	return
}
