package terrain

import (
	"fmt"
	"runtime"
	"sync"

	"worldgen/procnoise"
)

// --- Base Elevation Generation ---

// GenerateBaseElevation converts a continental mask to base elevation values.
// Continental areas get positive elevation, oceanic areas get negative depth.
// The mask threshold determines the land/ocean boundary.
func GenerateBaseElevation(sites []Vector3D, mask []float64, threshold float64, settings ElevationSettings) ([]float64, error) {
	if err := settings.Validate(); err != nil {
		return nil, fmt.Errorf("invalid elevation settings: %w", err)
	}

	if len(sites) != len(mask) {
		return nil, fmt.Errorf("sites (%d) and mask (%d) length mismatch", len(sites), len(mask))
	}

	if len(sites) == 0 {
		return nil, fmt.Errorf("no sites provided")
	}

	if settings.Verbose {
		fmt.Println("=== BASE ELEVATION GENERATION ===")
		fmt.Printf("  Sites: %d\n", len(sites))
		fmt.Printf("  Threshold: %.4f\n", threshold)
		fmt.Printf("  Continental base: %.0fm\n", settings.ContinentalBase)
		fmt.Printf("  Oceanic base: %.0fm\n", settings.OceanicBase)
	}

	// Generate base elevation in parallel
	elevation := generateElevationParallel(mask, threshold, settings)

	if settings.Verbose {
		landCount := 0
		for _, e := range elevation {
			if e > 0 {
				landCount++
			}
		}
		fmt.Printf("  Land sites: %d (%.1f%%)\n", landCount, 100*float64(landCount)/float64(len(elevation)))
	}

	return elevation, nil
}

// generateElevationParallel generates elevation values using parallel processing.
func generateElevationParallel(mask []float64, threshold float64, settings ElevationSettings) []float64 {
	elevation := make([]float64, len(mask))
	numWorkers := runtime.NumCPU()

	var wg sync.WaitGroup
	chunkSize := (len(mask) + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			start := workerID * chunkSize
			end := start + chunkSize
			if end > len(mask) {
				end = len(mask)
			}

			for i := start; i < end; i++ {
				elevation[i] = computeSiteElevation(mask[i], threshold, settings)
			}
		}(w)
	}

	wg.Wait()
	return elevation
}

// computeSiteElevation computes elevation for a single site based on mask value.
func computeSiteElevation(maskValue, threshold float64, settings ElevationSettings) float64 {
	if maskValue > threshold {
		// Continental - above threshold
		return computeContinentalElevation(maskValue, threshold, settings)
	} else if maskValue > threshold-settings.ShelfWidth {
		// Continental shelf - transition zone
		return computeShelfElevation(maskValue, threshold, settings)
	} else {
		// Oceanic - below shelf
		return computeOceanicElevation(maskValue, threshold, settings)
	}
}

// computeContinentalElevation calculates elevation for continental sites.
func computeContinentalElevation(maskValue, threshold float64, settings ElevationSettings) float64 {
	// Inlandness: 0 at coast, 1 at highest mask values
	// Assuming mask max is around 1.0
	maxMask := 1.0
	inlandness := (maskValue - threshold) / (maxMask - threshold)
	inlandness = Clamp(inlandness, 0, 1)

	// Base elevation increases with inlandness
	elevation := settings.ContinentalBase + inlandness*settings.ContinentalVariation

	return elevation
}

// computeShelfElevation calculates elevation for continental shelf sites.
func computeShelfElevation(maskValue, threshold float64, settings ElevationSettings) float64 {
	// Shelf position: 0 at ocean edge, 1 at land edge
	shelfPos := (maskValue - (threshold - settings.ShelfWidth)) / settings.ShelfWidth
	shelfPos = Clamp(shelfPos, 0, 1)

	// Smooth interpolation from shelf depth to sea level
	// Use smoothstep for more natural transition
	t := SmoothStep(0, 1, shelfPos)

	// Interpolate from shelf depth to just above 0
	return Lerp(settings.ShelfDepth, 10.0, t) // 10m above sea level at coast
}

// computeOceanicElevation calculates depth for oceanic sites.
func computeOceanicElevation(maskValue, threshold float64, settings ElevationSettings) float64 {
	// Ocean depth: deeper for lower mask values
	// Assuming mask min is around -1.0
	minMask := -1.0
	shelfEdge := threshold - settings.ShelfWidth

	// Depth factor: 0 at shelf edge, 1 at lowest mask values
	depthFactor := (shelfEdge - maskValue) / (shelfEdge - minMask)
	depthFactor = Clamp(depthFactor, 0, 1)

	// Base depth with variation
	depth := settings.OceanicBase - depthFactor*settings.OceanicVariation

	return depth
}

// --- Mountain Generation ---

// AddMountains adds mountain elevation to continental areas based on noise peaks.
func AddMountains(sites []Vector3D, elevation []float64, mask []float64, threshold float64, settings MountainSettings) error {
	if err := settings.Validate(); err != nil {
		return fmt.Errorf("invalid mountain settings: %w", err)
	}

	if len(sites) != len(elevation) || len(sites) != len(mask) {
		return fmt.Errorf("array length mismatch: sites=%d, elevation=%d, mask=%d",
			len(sites), len(elevation), len(mask))
	}

	if settings.Verbose {
		fmt.Println("=== MOUNTAIN GENERATION ===")
		fmt.Printf("  Mountain frequency: %.1f\n", settings.MountainFrequency)
		fmt.Printf("  Mountain threshold: %.2f\n", settings.MountainThreshold)
		fmt.Printf("  Max height: %.0fm\n", settings.MaxMountainHeight)
	}

	// Create mountain noise
	mountainNoise := createMountainNoise(settings)

	// Add mountains in parallel
	addMountainsParallel(sites, elevation, mask, threshold, mountainNoise, settings)

	if settings.Verbose {
		mountainCount := 0
		for _, e := range elevation {
			if e > 3000 {
				mountainCount++
			}
		}
		fmt.Printf("  Mountain sites (>3000m): %d (%.2f%%)\n",
			mountainCount, 100*float64(mountainCount)/float64(len(elevation)))
	}

	return nil
}

// createMountainNoise creates the noise field for mountain potential.
func createMountainNoise(settings MountainSettings) *procnoise.FastNoiseLiteScalarField {
	state := procnoise.New[float32]()
	state.Seed = int(settings.Seed)
	state.Frequency = float32(settings.MountainFrequency)
	state.NoiseType(procnoise.OpenSimplex2)
	state.FractalType(procnoise.FractalRidged) // Ridged noise for mountain chains
	state.Octaves = 4
	state.Lacunarity = 2.0
	state.Gain = 0.5

	return procnoise.NewFastNoiseLiteScalarField(state)
}

// addMountainsParallel adds mountain elevation using parallel processing.
func addMountainsParallel(sites []Vector3D, elevation []float64, mask []float64,
	threshold float64, noise *procnoise.FastNoiseLiteScalarField, settings MountainSettings) {

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
				// Only add mountains on land
				if mask[i] <= threshold {
					continue
				}

				// Sample mountain noise
				mountainValue := float64(noise.GetNoise(sites[i]))

				// Normalize from [-1,1] to [0,1]
				mountainValue = (mountainValue + 1) / 2

				// Only create mountains where noise exceeds threshold
				if mountainValue > settings.MountainThreshold {
					// Peak factor: how far above threshold
					peakFactor := (mountainValue - settings.MountainThreshold) / (1.0 - settings.MountainThreshold)

					// Add mountain height
					elevation[i] += peakFactor * settings.MaxMountainHeight
				}
			}
		}(w)
	}

	wg.Wait()
}

// --- Terrain Detail ---

// AddTerrainDetail adds fractal detail to elevation while preserving distribution.
func AddTerrainDetail(sites []Vector3D, elevation []float64, settings DetailSettings) error {
	if err := settings.Validate(); err != nil {
		return fmt.Errorf("invalid detail settings: %w", err)
	}

	if len(sites) != len(elevation) {
		return fmt.Errorf("sites (%d) and elevation (%d) length mismatch", len(sites), len(elevation))
	}

	if settings.Verbose {
		fmt.Println("=== TERRAIN DETAIL ===")
		fmt.Printf("  Detail frequency: %.1f\n", settings.DetailFrequency)
		fmt.Printf("  Detail octaves: %d\n", settings.DetailOctaves)
		fmt.Printf("  Base amplitude: %.0fm\n", settings.BaseAmplitude)
	}

	// Create detail noise
	detailNoise := createDetailNoise(settings)

	// Add detail in parallel
	addDetailParallel(sites, elevation, detailNoise, settings)

	return nil
}

// createDetailNoise creates the noise field for terrain detail.
func createDetailNoise(settings DetailSettings) *procnoise.FastNoiseLiteScalarField {
	state := procnoise.New[float32]()
	state.Seed = int(settings.Seed)
	state.Frequency = float32(settings.DetailFrequency)
	state.NoiseType(procnoise.OpenSimplex2)
	state.FractalType(procnoise.FractalFBm)
	state.Octaves = settings.DetailOctaves
	state.Lacunarity = float32(settings.Lacunarity)
	state.Gain = float32(settings.Persistence)

	return procnoise.NewFastNoiseLiteScalarField(state)
}

// addDetailParallel adds terrain detail using parallel processing.
func addDetailParallel(sites []Vector3D, elevation []float64,
	noise *procnoise.FastNoiseLiteScalarField, settings DetailSettings) {

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
				// Sample detail noise
				detail := float64(noise.GetNoise(sites[i]))

				// Scale amplitude by terrain type
				amplitude := settings.BaseAmplitude
				elev := elevation[i]

				if elev > 3000 {
					// More detail in mountains
					amplitude *= settings.MountainDetailScale
				} else if elev < -3000 {
					// Less detail in deep ocean (abyssal plains are flat)
					amplitude *= settings.DeepOceanDetailScale
				}

				elevation[i] += detail * amplitude
			}
		}(w)
	}

	wg.Wait()
}

// --- Full Pipeline ---

// GenerateFullElevation runs the complete elevation pipeline.
// Returns elevation values and the threshold used for land/ocean.
func GenerateFullElevation(sites []Vector3D, settings TerrainSettings) ([]float64, float64, error) {
	settings.ApplyMasterSeed()

	if err := settings.Validate(); err != nil {
		return nil, 0, fmt.Errorf("invalid terrain settings: %w", err)
	}

	if settings.Verbose {
		fmt.Println("========================================")
		fmt.Println("       FULL TERRAIN GENERATION")
		fmt.Println("========================================")
		fmt.Printf("Sites: %d\n", len(sites))
		fmt.Printf("Master seed: %d\n", settings.Seed)
		fmt.Println()
	}

	// Step 1: Generate continental mask
	if settings.Verbose {
		fmt.Println("Step 1: Continental Mask")
	}
	mask, threshold, _, err := GenerateContinentalMaskWithThreshold(sites, settings.Mask)
	if err != nil {
		return nil, 0, fmt.Errorf("continental mask failed: %w", err)
	}

	// Step 2: Generate base elevation
	if settings.Verbose {
		fmt.Println("\nStep 2: Base Elevation")
	}
	elevation, err := GenerateBaseElevation(sites, mask, threshold, settings.Elevation)
	if err != nil {
		return nil, 0, fmt.Errorf("base elevation failed: %w", err)
	}

	// Step 3: Add mountains
	if settings.Verbose {
		fmt.Println("\nStep 3: Mountains")
	}
	err = AddMountains(sites, elevation, mask, threshold, settings.Mountain)
	if err != nil {
		return nil, 0, fmt.Errorf("mountain generation failed: %w", err)
	}

	// Step 4: Add terrain detail
	if settings.Verbose {
		fmt.Println("\nStep 4: Terrain Detail")
	}
	err = AddTerrainDetail(sites, elevation, settings.Detail)
	if err != nil {
		return nil, 0, fmt.Errorf("terrain detail failed: %w", err)
	}

	if settings.Verbose {
		fmt.Println("\n========================================")
		fmt.Println("       GENERATION COMPLETE")
		fmt.Println("========================================")
	}

	return elevation, threshold, nil
}

// --- Elevation Adjustment ---

// AdjustElevationForHypsometry adjusts elevation to better match Earth's hypsometric curve.
// This is a post-processing step that redistributes elevations.
func AdjustElevationForHypsometry(elevation []float64) []float64 {
	if len(elevation) == 0 {
		return elevation
	}

	// Compute current metrics
	metrics := ComputeMetrics(nil, elevation)

	// Target adjustments based on Earth values
	adjusted := make([]float64, len(elevation))
	copy(adjusted, elevation)

	// Adjust mean land elevation if needed
	if metrics.MeanLandElevation > 0 {
		landScale := EarthMeanLandElevation / metrics.MeanLandElevation
		// Don't scale too aggressively
		landScale = Clamp(landScale, 0.8, 1.2)

		for i, e := range adjusted {
			if e > 0 {
				scale := landScale
				if e > 2500 {
					// Preserve high mountain tails; scaling lowlands and midlands is enough
					// to correct the mean without erasing too many mountains.
					t := SmoothStep(2500, 4500, e)
					scale = Lerp(landScale, 1.0, t)
				}
				adjusted[i] = e * scale
			}
		}

		// If the preserved mountain tail still leaves land too high overall, apply
		// one more gentle pass that scales mid/high terrain a bit more while
		// keeping the very highest mountains closer to their original heights.
		updatedLandMean := computeMeanLandElevation(adjusted)
		if updatedLandMean > 980 {
			secondaryScale := EarthMeanLandElevation / updatedLandMean
			secondaryScale = Clamp(secondaryScale, 0.94, 1.0)

			if secondaryScale < 0.999 {
				for i, e := range adjusted {
					if e <= 0 {
						continue
					}

					scale := secondaryScale
					if e > 2800 {
						t := SmoothStep(2800, 5000, e)
						scale = Lerp(secondaryScale, 1.0, t)
					}
					adjusted[i] = e * scale
				}
			}
		}
	}

	// Adjust mean ocean depth if needed
	if metrics.MeanOceanDepth < 0 {
		oceanScale := EarthMeanOceanDepth / metrics.MeanOceanDepth
		// Don't scale too aggressively
		oceanScale = Clamp(oceanScale, 0.8, 1.2)

		for i, e := range adjusted {
			if e <= 0 {
				adjusted[i] = e * oceanScale
			}
		}
	}

	return adjusted
}

// --- Convenience Functions ---

// QuickGenerateElevation generates elevation with default settings.
func QuickGenerateElevation(sites []Vector3D, seed int64) ([]float64, error) {
	settings := DefaultTerrainSettings()
	settings.Seed = seed
	elevation, _, err := GenerateFullElevation(sites, settings)
	return elevation, err
}

// GenerateAndEvaluate generates terrain and evaluates it against Earth metrics.
func GenerateAndEvaluate(sites []Vector3D, settings TerrainSettings) ([]float64, EvaluationResult, error) {
	elevation, _, err := GenerateFullElevation(sites, settings)
	if err != nil {
		return nil, EvaluationResult{}, err
	}

	result := EvaluateTerrain(sites, elevation)
	return elevation, result, nil
}
