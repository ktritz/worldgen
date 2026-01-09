package terrain

// Main pipeline for tectonic planet generation
// Produces Earth-like terrain with plates, mountains, trenches, and hotspot islands

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

// GeneratePlanetElevation is the main entry point for tectonic planet generation
// targetLandFraction: 0.0-1.0, the desired fraction of land (e.g., 0.29 for Earth-like)
// Returns elevation in meters with Earth-like hypsometric distribution
func GeneratePlanetElevation(
	sites []Vector3D,
	cells []VoronoiCell,
	numPlates int,
	seed int64,
	targetLandFraction float64,
) (elevation []float64, isLand []bool) {
	numRegions := len(sites)
	rng := rand.New(rand.NewSource(seed))

	fmt.Println("=== TECTONIC PLANET GENERATION ===")
	fmt.Printf("Regions: %d, Plates: %d\n", numRegions, numPlates)

	// Step 1: Generate plates using randomized BFS
	fmt.Println("Step 1: Generating plates...")
	plateR, rPlate := GeneratePlates(sites, cells, numPlates, rng)
	fmt.Printf("  Created %d plates\n", len(plateR))

	// Smooth plate boundaries to eliminate single-cell protrusions
	// This reduces isolated single-cell trenches while keeping coastlines irregular
	smoothedCells := SmoothPlateBoundaries(cells, rPlate, 3)
	if smoothedCells > 0 {
		fmt.Printf("  Smoothed %d boundary cells\n", smoothedCells)
	}

	// Calculate plate sizes
	plateSizes := make(map[int]int)
	for _, plate := range rPlate {
		plateSizes[plate]++
	}

	// Sort plates by size
	var sortedPlates []PlateSize
	for _, centerR := range plateR {
		sortedPlates = append(sortedPlates, PlateSize{
			Center: centerR,
			Size:   plateSizes[centerR],
		})
	}
	sort.Slice(sortedPlates, func(i, j int) bool {
		return sortedPlates[i].Size > sortedPlates[j].Size
	})

	// Step 2: Find plate neighbors
	plateNeighbors := FindPlateNeighbors(cells, rPlate, plateR)

	// Step 3: Assign plates as oceanic/continental
	fmt.Println("Step 2: Assigning plate types...")
	plateIsOcean := AssignPlateTypes(sortedPlates, plateSizes, plateNeighbors, numRegions, targetLandFraction)

	// Report plate distribution
	oceanicPlates, continentalPlates := 0, 0
	totalOceanicRegions, totalContinentalRegions := 0, 0
	for _, ps := range sortedPlates {
		if plateIsOcean[ps.Center] {
			oceanicPlates++
			totalOceanicRegions += ps.Size
		} else {
			continentalPlates++
			totalContinentalRegions += ps.Size
		}
	}
	fmt.Printf("  %d oceanic, %d continental\n", oceanicPlates, continentalPlates)
	fmt.Printf("  Oceanic plates cover %d regions (%.1f%% of surface)\n",
		totalOceanicRegions, 100*float64(totalOceanicRegions)/float64(numRegions))
	fmt.Printf("  Continental plates cover %d regions (%.1f%% of surface)\n",
		totalContinentalRegions, 100*float64(totalContinentalRegions)/float64(numRegions))

	// Step 4: Assign plate rotations (Euler poles for realistic curved motion)
	fmt.Println("Step 3: Assigning plate rotations (Euler poles)...")
	plateRot := AssignPlateRotations(sites, cells, plateR, plateIsOcean, plateNeighbors, rng)

	// Step 5: Compute elevation
	fmt.Println("Step 4: Computing elevation...")
	elevation, coastlineR := ComputeElevation(sites, cells, plateIsOcean, rPlate, plateRot, seed)

	// Step 6: Compute distance from coast for continental slope
	fmt.Println("  Computing continental distance from coast...")
	distFromCoast := ComputeDistanceFromCoast(cells, coastlineR, rPlate, plateIsOcean)

	// Find max distance for normalization
	maxDist := 0.0
	for r := 0; r < numRegions; r++ {
		if !plateIsOcean[rPlate[r]] && !math.IsInf(distFromCoast[r], 1) {
			if distFromCoast[r] > maxDist {
				maxDist = distFromCoast[r]
			}
		}
	}
	if maxDist == 0 {
		maxDist = 1
	}

	// Step 7: Apply bimodal elevation
	fmt.Println("  Applying bimodal elevation (interior high, coast low)...")
	ApplyBimodalElevation(elevation, distFromCoast, rPlate, plateIsOcean, maxDist)

	// Step 8: Apply erosion to smooth sharp peaks
	fmt.Println("Step 5: Applying erosion passes...")
	ApplySelectiveErosion(cells, elevation, rPlate, plateIsOcean, 3)

	// Step 9: Apply Earth hypsometric mapping
	fmt.Printf("Step 6: Applying Earth hypsometry (target land: %.1f%%)...\n", targetLandFraction*100)
	elevation = ApplyEarthHypsometry(elevation, targetLandFraction)

	// Step 9b: Apply elevation-scaled noise for terrain detail
	// baseFrequency=64 gives ~625km largest features, ~10km smallest (6 octaves)
	// Amplitudes based on Earth's actual terrain roughness at these scales:
	// - Mountains: ±800m (Himalayan peaks vary 1-2km within ranges)
	// - Plains: ±80m (rolling hills, river terraces)
	// - Ocean: ±150m (abyssal hills typically 50-300m)
	ApplyElevationScaledNoise(sites, elevation, seed, 64.0, 800.0, 80.0, 150.0)

	// Step 10: Apply landmass-aware erosion (caps peaks on small islands, in meters)
	fmt.Println("  Applying landmass erosion (size + coastal proximity)...")
	ApplyLandmassErosion(cells, elevation, rPlate, plateIsOcean, distFromCoast)

	// Step 11: Add hotspot island chains (AFTER hypsometry, works in meters)
	// This slightly increases land percentage but creates realistic volcanic islands
	fmt.Println("Step 7: Generating hotspot island chains...")
	hotspotChains := PlaceHotspots(sites, cells, rPlate, plateRot, plateIsOcean, rng)
	numIslandCells, _, hotspotCells := ApplyHotspotElevation(elevation, cells, sites, hotspotChains, rPlate, plateIsOcean, rng)
	// Show chain lengths and type breakdown
	oceanicCount, continentalCount := 0, 0
	oceanicLengths := make([]int, 0)
	continentalLengths := make([]int, 0)
	for _, chain := range hotspotChains {
		if chain.IsOceanic {
			oceanicCount++
			oceanicLengths = append(oceanicLengths, len(chain.Islands))
		} else {
			continentalCount++
			continentalLengths = append(continentalLengths, len(chain.Islands))
		}
	}
	fmt.Printf("  Created %d chains (%d oceanic, %d continental), %d modified cells\n",
		len(hotspotChains), oceanicCount, continentalCount, numIslandCells)
	fmt.Printf("  Oceanic chain lengths: %v\n", oceanicLengths)
	fmt.Printf("  Continental chain lengths: %v\n", continentalLengths)

	// Step 12: Ensure hotspot cells stay above their minimum elevation
	// This prevents erosion from completely erasing islands
	floored := 0
	maxOceanicElev, maxContinentalElev := 0.0, 0.0
	for cellIdx, info := range hotspotCells {
		if elevation[cellIdx] < info.MinElevation {
			elevation[cellIdx] = info.MinElevation
			floored++
		}
		// Track max elevations
		if info.IsOceanic && elevation[cellIdx] > maxOceanicElev {
			maxOceanicElev = elevation[cellIdx]
		}
		if !info.IsOceanic && elevation[cellIdx] > maxContinentalElev {
			maxContinentalElev = elevation[cellIdx]
		}
	}
	fmt.Printf("  Hotspot elevations - oceanic max: %.0fm, continental max: %.0fm\n", maxOceanicElev, maxContinentalElev)
	if floored > 0 {
		fmt.Printf("  Applied minimum elevation to %d hotspot cells\n", floored)
	}

	// Step 13: Determine land/ocean
	isLand = make([]bool, numRegions)
	landCount := 0
	for r := 0; r < numRegions; r++ {
		plate := rPlate[r]
		// Land = continental plate above sea level OR hotspot island (oceanic but positive elevation)
		isLand[r] = elevation[r] > 0
		if isLand[r] {
			landCount++
		}
		// But for display purposes, we still track plate type
		_ = plate
	}
	fmt.Printf("  Actual land coverage: %.1f%%\n", float64(landCount)/float64(numRegions)*100)

	return elevation, isLand
}
