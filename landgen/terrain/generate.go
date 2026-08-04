package terrain

// Main pipeline for tectonic planet generation
// Produces Earth-like terrain with plates, mountains, trenches, and hotspot islands

import (
	"fmt"
	"math"
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
	elevation, isLand, _ = GeneratePlanetElevationWithDiagnostics(sites, cells, numPlates, seed, targetLandFraction)
	return elevation, isLand
}

// GeneratePlanetElevationWithDiagnostics runs the full generator and returns
// optional generation-side metadata useful for review tooling.
func GeneratePlanetElevationWithDiagnostics(
	sites []Vector3D,
	cells []VoronoiCell,
	numPlates int,
	seed int64,
	targetLandFraction float64,
) (elevation []float64, isLand []bool, diagnostics PlanetGenerationDiagnostics) {
	numRegions := len(sites)

	fmt.Println("=== TECTONIC PLANET GENERATION ===")
	fmt.Printf("Regions: %d, Plates: %d\n", numRegions, numPlates)

	// Step 1: Generate plates using randomized BFS
	fmt.Println("Step 1: Generating plates...")
	layout := GenerateOptimizedPlateLayout(sites, cells, numPlates, seed, targetLandFraction)
	plateR := layout.plateR
	rPlate := layout.rPlate
	plateSizes := layout.plateSizes
	sortedPlates := layout.sortedPlates
	plateNeighbors := layout.plateNeighbors
	fmt.Printf("  Created %d active plates (selected layout attempt %d of up to %d)\n",
		len(plateR), layout.attempt+1, plateLayoutSearchAttempts+plateLayoutSearchExtraAttempts)

	// Step 3: Assign plates as oceanic/continental
	fmt.Println("Step 2: Assigning plate types...")
	plateIsOcean := layout.plateIsOcean
	if len(plateIsOcean) == 0 {
		plateIsOcean = AssignPlateTypes(sortedPlates, plateSizes, plateNeighbors, numRegions, targetLandFraction)
	}

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
	plateRot := assignPlateRotations(layout, plateIsOcean, plateNeighbors, seed)

	// Step 5: Compute elevation
	fmt.Println("Step 4: Computing elevation...")
	elevation, boundarySeeds := ComputeElevationWithSeeds(sites, cells, plateIsOcean, rPlate, plateRot, seed)
	coastlineR := boundarySeeds.Coastline
	mountainR := boundarySeeds.Mountain
	collisionR := boundarySeeds.Collision
	arcR := boundarySeeds.Arc
	ridgeR := boundarySeeds.Ridge
	trenchR := boundarySeeds.Trench

	// Step 6: Compute distance from coast for continental slope
	fmt.Println("  Computing continental distance from coast...")
	distFromCoast := ComputeDistanceFromCoast(cells, coastlineR, rPlate, plateIsOcean)
	oceanDistFromCoast := ComputeOceanDistanceFromCoast(cells, coastlineR, rPlate, plateIsOcean)
	distFromRidge := ComputeOceanDistanceFromSeeds(cells, ridgeR, rPlate, plateIsOcean)
	maxRidgeDist := ComputeOceanPlateMaxDistance(rPlate, plateIsOcean, distFromRidge)
	distFromTrench := ComputeOceanDistanceFromSeeds(cells, trenchR, rPlate, plateIsOcean)
	componentMaxDist := ComputeContinentalComponentMaxDistance(cells, rPlate, plateIsOcean, distFromCoast)
	distFromMountain := ComputeDistanceFromMountainSeeds(cells, mountainR, rPlate, plateIsOcean)
	componentMaxMountainDist := ComputeContinentalComponentMaxTectonicDistance(cells, rPlate, plateIsOcean, distFromMountain)
	distFromCollision := ComputeDistanceFromMountainSeeds(cells, collisionR, rPlate, plateIsOcean)
	componentMaxCollisionDist := ComputeContinentalComponentMaxTectonicDistance(cells, rPlate, plateIsOcean, distFromCollision)
	distFromArc := ComputeDistanceFromMountainSeeds(cells, arcR, rPlate, plateIsOcean)
	componentMaxArcDist := ComputeContinentalComponentMaxTectonicDistance(cells, rPlate, plateIsOcean, distFromArc)
	// Continental rifts are seeded but were never rasterised; reuse the
	// continental seed-distance walk so rift proximity is exportable too.
	// Nothing downstream consumes this field, so terrain output is unchanged.
	distFromRift := ComputeDistanceFromMountainSeeds(cells, boundarySeeds.Rift, rPlate, plateIsOcean)

	// Export the tectonic scaffolding while the fields are still pristine. The
	// snapshot copies every slice, so later elevation passes cannot alias or
	// perturb what review and lithology tooling reads back.
	diagnostics.Tectonics = buildTectonicDiagnostics(rPlate, plateIsOcean, boundarySeeds, tectonicDistanceFields{
		coast:      distFromCoast,
		oceanCoast: oceanDistFromCoast,
		mountain:   distFromMountain,
		collision:  distFromCollision,
		arc:        distFromArc,
		ridge:      distFromRidge,
		trench:     distFromTrench,
		rift:       distFromRift,
	})

	// Find max distance for normalization
	maxDist := 0.0
	maxOceanDist := 0.0
	for r := 0; r < numRegions; r++ {
		if !plateIsOcean[rPlate[r]] && !math.IsInf(distFromCoast[r], 1) {
			if distFromCoast[r] > maxDist {
				maxDist = distFromCoast[r]
			}
		}
		if plateIsOcean[rPlate[r]] && !math.IsInf(oceanDistFromCoast[r], 1) {
			if oceanDistFromCoast[r] > maxOceanDist {
				maxOceanDist = oceanDistFromCoast[r]
			}
		}
	}
	if maxDist == 0 {
		maxDist = 1
	}
	if maxOceanDist == 0 {
		maxOceanDist = 1
	}

	// Step 7: Apply bimodal elevation
	fmt.Println("  Applying bimodal elevation (interior high, coast low)...")
	ApplyBimodalElevation(
		elevation,
		distFromCoast,
		oceanDistFromCoast,
		componentMaxDist,
		distFromMountain,
		componentMaxMountainDist,
		distFromCollision,
		componentMaxCollisionDist,
		distFromArc,
		componentMaxArcDist,
		rPlate,
		plateIsOcean,
		maxDist,
		maxOceanDist,
	)
	ApplyOceanBasinStructure(
		elevation,
		sites,
		rPlate,
		plateIsOcean,
		plateRot,
		oceanDistFromCoast,
		distFromRidge,
		maxRidgeDist,
		distFromTrench,
		maxOceanDist,
		seed,
	)

	// Step 8: Apply erosion to smooth sharp peaks
	fmt.Println("Step 5: Applying erosion passes...")
	ApplySelectiveErosion(cells, elevation, rPlate, plateIsOcean, 3)

	// Step 9: Apply Earth hypsometric mapping
	fmt.Printf("Step 6: Applying Earth hypsometry (target land: %.1f%%)...\n", targetLandFraction*100)
	elevation = ApplyEarthHypsometry(elevation, targetLandFraction)
	ReinforceTectonicMountains(
		elevation,
		rPlate,
		plateIsOcean,
		distFromMountain,
		componentMaxMountainDist,
		distFromCollision,
		componentMaxCollisionDist,
		distFromArc,
		componentMaxArcDist,
	)
	ApplyFluvialErosion(sites, cells, elevation, distFromCoast, maxDist, seed)

	// Step 9b: Apply elevation-scaled noise for terrain detail
	// baseFrequency=64 gives ~625km largest features, ~10km smallest (6 octaves)
	// Amplitudes based on Earth's actual terrain roughness at these scales:
	// - Mountains: ±800m (Himalayan peaks vary 1-2km within ranges)
	// - Plains: ±80m (rolling hills, river terraces)
	// - Ocean: ±150m (abyssal hills typically 50-300m)
	coastalExposure := ComputeCoastalExposure(cells, elevation, oceanDistFromCoast)
	ApplyElevationScaledNoise(sites, cells, elevation, oceanDistFromCoast, seed, 64.0, 800.0, 80.0, 150.0)
	RegularizeCoastlines(cells, elevation, coastalExposure, 2)

	// Step 10: Apply landmass-aware erosion (caps peaks on small islands, in meters)
	fmt.Println("  Applying landmass erosion (size + coastal proximity)...")
	ApplyLandmassErosion(cells, elevation, rPlate, plateIsOcean, distFromCoast, distFromMountain)

	// Step 11: Add hotspot island chains (AFTER hypsometry, works in meters)
	// This slightly increases land percentage but creates realistic volcanic islands
	fmt.Println("Step 7: Generating hotspot island chains...")
	hotspotChains := placeHotspots(sites, cells, layout, plateRot, plateIsOcean, seed)
	numIslandCells, _, hotspotCells := ApplyHotspotElevation(elevation, cells, sites, hotspotChains, rPlate, plateIsOcean, seed)
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

	// Step 12b: Rebalance mean land and ocean elevations toward Earth targets.
	elevation = AdjustElevationForHypsometry(elevation)

	// Preserve hotspot minimum elevations after the global rebalance.
	refloored := 0
	for cellIdx, info := range hotspotCells {
		if elevation[cellIdx] < info.MinElevation {
			elevation[cellIdx] = info.MinElevation
			refloored++
		}
	}
	if refloored > 0 {
		fmt.Printf("  Restored %d hotspot cells after mean-elevation rebalance\n", refloored)
	}

	// Final drainage conditioning after late-stage terrain detail. This cleans
	// up shallow synthetic traps introduced by noise/coast edits before any
	// downstream hydrology consumes the final DEM.
	postDetailBreaches := ApplyPostDetailDrainageConditioning(cells, elevation)
	diagnostics.Hydrology = ComputeHydrologyDiagnostics(sites, cells, elevation, seed)
	diagnostics.Hydrology.PostDetailBreachedSinks = postDetailBreaches
	fmt.Printf("  Post-detail terrain-proxy drainage: breached %d shallow sinks, channels %.1f%%, endorheic %.1f%%, inland lakes %.2f%%\n",
		postDetailBreaches,
		diagnostics.Hydrology.FluvialChannelCoverage*100,
		diagnostics.Hydrology.EndorheicCatchmentPct*100,
		diagnostics.Hydrology.InlandLakeCoverage*100,
	)

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
	diagnostics.HotspotChains = hotspotChains

	return elevation, isLand, diagnostics
}
