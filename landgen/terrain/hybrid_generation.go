// Package terrain - hybrid_generation.go
// Hybrid approach: continent seeds for structure + noise for coastlines + separate mountains
package terrain

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"worldgen/procnoise"
)

// ContinentSeed represents a continent or ocean basin center
type ContinentSeed struct {
	Center       Vector3D
	IsContinental bool
	Radius       float64 // Base influence radius
}

// HybridSettings controls the hybrid generation approach
type HybridSettings struct {
	Seed                   int64
	NumContinents          int     // Number of continental seeds (5-8 for Earth-like)
	NumOceanBasins         int     // Number of oceanic seeds (3-5)
	TargetLandCoverage     float64 // Target land percentage (0.29 for Earth)
	CoastlineNoiseAmp      float64 // Amplitude of coastline noise (0.1-0.3)
	CoastlineNoiseFreq     float64 // Frequency of coastline noise (2.0-4.0)
	CoastlineNoiseOctaves  int     // Octaves for coastline detail
	MountainNoiseFreq      float64 // Frequency for mountain ridges
	MountainThreshold      float64 // Threshold for mountain peaks
	MaxMountainHeight      float64 // Maximum mountain elevation
	Verbose                bool
}

// DefaultHybridSettings returns Earth-like defaults
func DefaultHybridSettings() HybridSettings {
	return HybridSettings{
		Seed:                  42,
		NumContinents:         6,    // Like Earth's major landmasses
		NumOceanBasins:        4,    // Major ocean basins
		TargetLandCoverage:    0.30, // 30% land
		CoastlineNoiseAmp:     0.15, // Moderate coastline irregularity
		CoastlineNoiseFreq:    3.0,  // Medium-scale coastline features
		CoastlineNoiseOctaves: 3,
		MountainNoiseFreq:     4.0,
		MountainThreshold:     0.82, // Tuned for ~2% mountain coverage
		MaxMountainHeight:     6000.0,
		Verbose:               false,
	}
}

// GenerateHybridTerrain generates terrain using the hybrid approach:
// 1. Place continent/ocean seeds for structure
// 2. Compute membership with noisy boundaries
// 3. Add separate mountain layer
func GenerateHybridTerrain(sites []Vector3D, settings HybridSettings) ([]float64, error) {
	rng := rand.New(rand.NewSource(settings.Seed))

	if settings.Verbose {
		fmt.Println("=== HYBRID TERRAIN GENERATION ===")
		fmt.Printf("  Sites: %d\n", len(sites))
		fmt.Printf("  Continents: %d, Ocean basins: %d\n", settings.NumContinents, settings.NumOceanBasins)
	}

	// Step 1: Place seeds with good spacing
	seeds := placeSeedsWithSpacing(settings.NumContinents, settings.NumOceanBasins, rng)

	if settings.Verbose {
		fmt.Printf("  Placed %d seeds (%d continental, %d oceanic)\n",
			len(seeds), settings.NumContinents, settings.NumOceanBasins)
	}

	// Step 2: Generate continental mask using noisy distance field
	mask := generateNoisyMask(sites, seeds, settings, rng)

	// Step 3: Find threshold for target land coverage
	threshold := findThresholdForCoverage(mask, settings.TargetLandCoverage)

	if settings.Verbose {
		landCount := 0
		for _, m := range mask {
			if m > threshold {
				landCount++
			}
		}
		fmt.Printf("  Threshold: %.3f, Land coverage: %.1f%%\n",
			threshold, float64(landCount)/float64(len(sites))*100)
	}

	// Step 4: Convert mask to base elevation
	elevation := maskToElevation(mask, threshold, settings)

	// Step 5: Add mountain layer (separate from boundaries)
	addMountainLayer(elevation, sites, mask, threshold, settings, rng)

	// Step 6: Add terrain detail
	addDetailLayer(elevation, sites, settings, rng)

	return elevation, nil
}

// placeSeedsWithSpacing places continent and ocean seeds with good spacing
func placeSeedsWithSpacing(numContinents, numOceanBasins int, rng *rand.Rand) []ContinentSeed {
	totalSeeds := numContinents + numOceanBasins
	seeds := make([]ContinentSeed, 0, totalSeeds)

	// Use Fibonacci sphere for even initial distribution, then jitter
	for i := 0; i < totalSeeds; i++ {
		// Fibonacci sphere point
		phi := math.Acos(1 - 2*float64(i+1)/float64(totalSeeds+1))
		theta := math.Pi * (1 + math.Sqrt(5)) * float64(i)

		// Add random jitter (up to 20 degrees)
		phi += (rng.Float64() - 0.5) * 0.35
		theta += (rng.Float64() - 0.5) * 0.35

		// Clamp phi to valid range
		if phi < 0.1 {
			phi = 0.1
		}
		if phi > math.Pi-0.1 {
			phi = math.Pi - 0.1
		}

		center := Vector3D{
			X: math.Sin(phi) * math.Cos(theta),
			Y: math.Sin(phi) * math.Sin(theta),
			Z: math.Cos(phi),
		}

		seeds = append(seeds, ContinentSeed{
			Center:        center,
			IsContinental: i < numContinents,
			Radius:        0.3 + rng.Float64()*0.2, // Varied sizes
		})
	}

	// Shuffle so continents aren't all at one end
	rng.Shuffle(len(seeds), func(i, j int) {
		seeds[i], seeds[j] = seeds[j], seeds[i]
	})

	return seeds
}

// generateNoisyMask creates a continental mask using Voronoi-like assignment with noise
func generateNoisyMask(sites []Vector3D, seeds []ContinentSeed, settings HybridSettings, rng *rand.Rand) []float64 {
	mask := make([]float64, len(sites))

	// Create noise generator for coastline irregularity
	state := procnoise.New[float32]()
	state.Seed = int(settings.Seed + 1000)
	state.NoiseType(procnoise.OpenSimplex2S)
	state.Frequency = float32(settings.CoastlineNoiseFreq)
	state.FractalType(procnoise.FractalFBm)
	state.Octaves = settings.CoastlineNoiseOctaves
	state.Lacunarity = 2.0
	state.Gain = 0.5
	noiseField := procnoise.NewFastNoiseLiteScalarField(state)

	for i, site := range sites {
		// Find distances to all seeds
		type seedDist struct {
			seed *ContinentSeed
			dist float64
		}
		distances := make([]seedDist, len(seeds))

		for j := range seeds {
			// Great circle distance (angular distance on sphere)
			dot := site.X*seeds[j].Center.X + site.Y*seeds[j].Center.Y + site.Z*seeds[j].Center.Z
			if dot > 1 {
				dot = 1
			}
			if dot < -1 {
				dot = -1
			}
			dist := math.Acos(dot) // Radians

			// Adjust by seed radius (larger seeds have more influence)
			adjustedDist := dist / seeds[j].Radius

			distances[j] = seedDist{seed: &seeds[j], dist: adjustedDist}
		}

		// Sort by distance
		sort.Slice(distances, func(a, b int) bool {
			return distances[a].dist < distances[b].dist
		})

		// Get noise at this point for boundary perturbation
		noise := noiseField.GetNoise(site)
		noiseOffset := float64(noise) * settings.CoastlineNoiseAmp

		// Compute mask value based on nearest seed and boundary noise
		nearestSeed := distances[0].seed
		nearestDist := distances[0].dist

		if nearestSeed.IsContinental {
			// Continental seed: positive mask value, decreasing with distance
			// Add noise to the boundary
			mask[i] = 1.0 - nearestDist + noiseOffset
		} else {
			// Oceanic seed: negative mask value
			mask[i] = -1.0 + nearestDist + noiseOffset
		}
	}

	return mask
}

// findThresholdForCoverage finds the mask threshold that gives target land coverage
func findThresholdForCoverage(mask []float64, targetCoverage float64) float64 {
	// Sort mask values to find percentile
	sorted := make([]float64, len(mask))
	copy(sorted, mask)
	sort.Float64s(sorted)

	// Find threshold at (1 - targetCoverage) percentile
	// E.g., for 30% land, find the 70th percentile value
	idx := int(float64(len(sorted)) * (1.0 - targetCoverage))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

// maskToElevation converts mask values to base elevation
func maskToElevation(mask []float64, threshold float64, settings HybridSettings) []float64 {
	elevation := make([]float64, len(mask))

	for i, m := range mask {
		if m > threshold {
			// Land: scale from 0 to ~800m based on distance from coast
			landAmount := (m - threshold) / (1.0 - threshold + 0.001)
			// Use power curve to keep most land at moderate elevation
			elevation[i] = 100.0 + math.Pow(landAmount, 0.7)*700.0 // 100-800m base
		} else {
			// Ocean: shelf near coast, then drop to deep ocean
			oceanAmount := (threshold - m) / (threshold + 1.0 + 0.001)

			if oceanAmount < 0.08 {
				// Continental shelf: -20 to -200m (narrower shelf)
				shelfProgress := oceanAmount / 0.08
				elevation[i] = -20.0 - shelfProgress*180.0
			} else if oceanAmount < 0.20 {
				// Continental slope: -200 to -3000m (steep drop)
				slopeProgress := (oceanAmount - 0.08) / 0.12
				elevation[i] = -200.0 - slopeProgress*2800.0
			} else {
				// Abyssal plain: -3000 to -5500m (most of ocean here)
				abyssProgress := (oceanAmount - 0.20) / 0.80
				elevation[i] = -3000.0 - abyssProgress*2500.0
			}
		}
	}

	return elevation
}

// addMountainLayer adds mountain ridges as a separate layer (not tied to boundaries)
func addMountainLayer(elevation []float64, sites []Vector3D, mask []float64, threshold float64, settings HybridSettings, rng *rand.Rand) {
	// Create ridged noise for mountain chains
	state := procnoise.New[float32]()
	state.Seed = int(settings.Seed + 2000)
	state.NoiseType(procnoise.OpenSimplex2S)
	state.Frequency = float32(settings.MountainNoiseFreq)
	state.FractalType(procnoise.FractalRidged)
	state.Octaves = 3
	state.Lacunarity = 2.0
	state.Gain = 0.5
	noiseField := procnoise.NewFastNoiseLiteScalarField(state)

	mountainCount := 0
	for i, site := range sites {
		// Only add mountains on land
		if mask[i] <= threshold {
			continue
		}

		// Get ridged noise value
		noise := noiseField.GetNoise(site)

		// Normalize to 0-1
		noiseVal := (float64(noise) + 1.0) / 2.0

		if noiseVal > settings.MountainThreshold {
			// Mountain! Scale height by how much above threshold
			mountainAmount := (noiseVal - settings.MountainThreshold) / (1.0 - settings.MountainThreshold)
			mountainHeight := mountainAmount * settings.MaxMountainHeight

			// Add to existing elevation
			elevation[i] += mountainHeight
			mountainCount++
		}
	}

	if settings.Verbose {
		fmt.Printf("  Mountains: %d sites (%.1f%% of land)\n",
			mountainCount, float64(mountainCount)/float64(len(sites))*100)
	}
}

// addDetailLayer adds fine-scale terrain detail
func addDetailLayer(elevation []float64, sites []Vector3D, settings HybridSettings, rng *rand.Rand) {
	state := procnoise.New[float32]()
	state.Seed = int(settings.Seed + 3000)
	state.NoiseType(procnoise.OpenSimplex2S)
	state.Frequency = 8.0
	state.FractalType(procnoise.FractalFBm)
	state.Octaves = 4
	state.Lacunarity = 2.0
	state.Gain = 0.5
	noiseField := procnoise.NewFastNoiseLiteScalarField(state)

	for i, site := range sites {
		noise := noiseField.GetNoise(site)

		// Scale detail by absolute elevation (more detail in mountains/deep ocean)
		baseAmp := 150.0
		if elevation[i] > 2000 {
			baseAmp = 300.0 // More detail in mountains
		} else if elevation[i] < -3000 {
			baseAmp = 200.0 // Some detail in deep ocean
		}

		elevation[i] += float64(noise) * baseAmp
	}
}

// GenerateHybridAndEvaluate generates terrain and evaluates it
func GenerateHybridAndEvaluate(sites []Vector3D, settings HybridSettings) ([]float64, *EvaluationResult, error) {
	elevation, err := GenerateHybridTerrain(sites, settings)
	if err != nil {
		return nil, nil, err
	}

	metrics := ComputeMetrics(sites, elevation)
	result := EvaluateMetrics(metrics)

	return elevation, &result, nil
}
