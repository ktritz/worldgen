package tectonics

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

// HybridPlateGeneration implements Method 7: Convection + Optimization
// Expected score: 0.75-0.90
func HybridPlateGeneration(
	voronoiCells []VoronoiCell,
	icosphereSites []Vector3D,
	planetRadius float64,
	settings HybridSettings,
) ([]TectonicPlate, []int, error) {

	// Create RNG for plate generation
	rng := rand.New(rand.NewSource(settings.ConvectionSettings.Seed))

	if settings.Verbose {
		fmt.Println("=== HYBRID PLATE GENERATION (Method 7) ===")
		fmt.Printf("Target: 39 plates (7 major, 13 minor, 19 micro)\n")
		fmt.Println()
	}

	// Phase 1: Generate initial plates from mantle convection
	if settings.Verbose {
		fmt.Println("Phase 1: Mantle Convection Initialization...")
	}

	convectionSettings := settings.ConvectionSettings
	initialPlates, initialAssignments, err := GeneratePlatesFromMantleConvection(
		voronoiCells,
		icosphereSites,
		planetRadius,
		convectionSettings,
	)

	if err != nil {
		return nil, nil, err
	}

	if settings.Verbose {
		fmt.Printf("  Initial: %d plates generated\n", len(initialPlates))
	}

	// Phase 2: Iterative refinement
	if settings.Verbose {
		fmt.Println("\nPhase 2: Iterative Refinement...")
	}

	plates := initialPlates
	cellAssignments := initialAssignments

	for iteration := 0; iteration < settings.RefinementIterations; iteration++ {
		if settings.Verbose && iteration%10 == 0 {
			fmt.Printf("  Iteration %d/%d: %d plates\n", iteration, settings.RefinementIterations, len(plates))
		}

		// Analyze current distribution
		analysis := analyzePlateDistribution(plates, planetRadius)

		// Apply targeted modifications
		modified := false

		// Strategy 1: Only if we need MORE majors, merge small minors
		// (Phase 2 config should start with 7 majors, so skip this usually)
		if analysis.majorCount < 7 && analysis.minorCount > 15 {
			// Only merge if we have excess minors
			numToMerge := 2  // Very conservative
			plates, cellAssignments = mergeSmallMinorsIntoMajors(
				plates, cellAssignments, voronoiCells, icosphereSites, planetRadius, numToMerge, rng)
			modified = true
		}

		// Strategy 2: Split large plates to create micro plates
		if analysis.microCount < 19 && len(plates) < 50 {
			plates, cellAssignments = splitLargePlatesForMicros(
				plates, cellAssignments, voronoiCells, icosphereSites, planetRadius, 3, rng)
			modified = true
		}

		// Strategy 3: Merge tiny plates that are too small
		if analysis.tinyCount > 0 {
			plates, cellAssignments = mergeTinyPlates(
				plates, cellAssignments, voronoiCells, icosphereSites, planetRadius, rng)
			modified = true
		}

		// Rebuild plates from assignments
		if modified {
			plates = createPlatesFromAssignments(voronoiCells, icosphereSites, cellAssignments, []int{}, planetRadius, rng)
		}

		// Check if we've reached target distribution
		if analysis.majorCount == 7 && analysis.minorCount >= 10 && analysis.minorCount <= 16 &&
			analysis.microCount >= 15 && analysis.microCount <= 25 {
			if settings.Verbose {
				fmt.Printf("  ✓ Target distribution reached at iteration %d\n", iteration)
			}
			break
		}
	}

	// Final analysis
	finalAnalysis := analyzePlateDistribution(plates, planetRadius)
	if settings.Verbose {
		fmt.Println("\n=== HYBRID GENERATION COMPLETE ===")
		fmt.Printf("Final: %d plates (%d major, %d minor, %d micro)\n",
			len(plates), finalAnalysis.majorCount, finalAnalysis.minorCount, finalAnalysis.microCount)
	}

	return plates, cellAssignments, nil
}

// HybridSettings controls hybrid generation
type HybridSettings struct {
	ConvectionSettings    ConvectionSettings
	RefinementIterations  int
	TargetMajorCount      int
	TargetMinorCount      int
	TargetMicroCount      int
	Verbose               bool
}

// DefaultHybridSettings returns recommended settings
func DefaultHybridSettings() HybridSettings {
	return HybridSettings{
		ConvectionSettings:   DefaultConvectionSettings(),
		RefinementIterations: 100,
		TargetMajorCount:     7,
		TargetMinorCount:     13,
		TargetMicroCount:     19,
		Verbose:              false,
	}
}

type plateAnalysis struct {
	majorCount int
	minorCount int
	microCount int
	tinyCount  int    // <0.01%
	sizes      []float64
}

func analyzePlateDistribution(plates []TectonicPlate, planetRadius float64) plateAnalysis {
	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius

	analysis := plateAnalysis{
		sizes: make([]float64, len(plates)),
	}

	for i, plate := range plates {
		percent := (plate.Area / sphereArea) * 100.0
		analysis.sizes[i] = percent

		if percent >= 6.0 {
			analysis.majorCount++
		} else if percent >= 0.18 {
			analysis.minorCount++
		} else if percent >= 0.01 {
			analysis.microCount++
		} else {
			analysis.tinyCount++
		}
	}

	return analysis
}

// mergeSmallMinorsIntoMajors merges the smallest minor plates into their largest neighbors
func mergeSmallMinorsIntoMajors(
	plates []TectonicPlate,
	cellAssignments []int,
	voronoiCells []VoronoiCell,
	icosphereSites []Vector3D,
	planetRadius float64,
	numToMerge int,
	rng *rand.Rand,
) ([]TectonicPlate, []int) {

	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius

	// Find minor plates
	type plateSizeInfo struct {
		index   int
		size    float64
		percent float64
	}

	minorPlates := make([]plateSizeInfo, 0)
	for i, plate := range plates {
		percent := (plate.Area / sphereArea) * 100.0
		if percent >= 0.18 && percent < 6.0 {
			minorPlates = append(minorPlates, plateSizeInfo{i, plate.Area, percent})
		}
	}

	// Sort by size (smallest first)
	sort.Slice(minorPlates, func(i, j int) bool {
		return minorPlates[i].size < minorPlates[j].size
	})

	// Merge smallest N
	mergeMap := make(map[int]int)
	for i := range cellAssignments {
		mergeMap[i] = i
	}

	for i := 0; i < numToMerge && i < len(minorPlates); i++ {
		plateToMerge := minorPlates[i].index

		// Find cells of this plate
		plateCells := make([]int, 0)
		for cellIdx, plateIdx := range cellAssignments {
			if plateIdx == plateToMerge {
				plateCells = append(plateCells, cellIdx)
			}
		}

		// Find largest neighbor
		neighborSizes := make(map[int]float64)
		for _, cellIdx := range plateCells {
			for _, neighborIdx := range voronoiCells[cellIdx].NeighborSiteIndices {
				if int(neighborIdx) < len(cellAssignments) {
					neighborPlate := cellAssignments[neighborIdx]
					if neighborPlate != plateToMerge && neighborPlate >= 0 {
						neighborSizes[neighborPlate] += plates[neighborPlate].Area
					}
				}
			}
		}

		// Merge into largest neighbor
		largestNeighbor := -1
		largestSize := 0.0
		for neighbor, size := range neighborSizes {
			if size > largestSize {
				largestSize = size
				largestNeighbor = neighbor
			}
		}

		if largestNeighbor >= 0 {
			mergeMap[plateToMerge] = largestNeighbor
		}
	}

	// Apply merges
	newAssignments := make([]int, len(cellAssignments))
	for i, plateIdx := range cellAssignments {
		newPlate := plateIdx
		for mergeMap[newPlate] != newPlate {
			newPlate = mergeMap[newPlate]
		}
		newAssignments[i] = newPlate
	}

	// Renumber
	newAssignments = renumberAssignments(newAssignments)
	newPlates := createPlatesFromAssignments(voronoiCells, icosphereSites, newAssignments, []int{}, planetRadius, rng)

	return newPlates, newAssignments
}

// splitLargePlatesForMicros splits large plates to create micro plates
func splitLargePlatesForMicros(
	plates []TectonicPlate,
	cellAssignments []int,
	voronoiCells []VoronoiCell,
	icosphereSites []Vector3D,
	planetRadius float64,
	numToSplit int,
	rng *rand.Rand,
) ([]TectonicPlate, []int) {

	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius

	// Find large minor plates (good candidates for splitting)
	type plateSizeInfo struct {
		index   int
		size    float64
		percent float64
	}

	candidates := make([]plateSizeInfo, 0)
	for i, plate := range plates {
		percent := (plate.Area / sphereArea) * 100.0
		// Split large minors (2-5%) - creates 1 minor + 1-2 micros
		if percent >= 2.0 && percent < 6.0 {
			candidates = append(candidates, plateSizeInfo{i, plate.Area, percent})
		}
	}

	// Sort by size (largest first - they can afford to lose territory)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].size > candidates[j].size
	})

	newAssignments := make([]int, len(cellAssignments))
	copy(newAssignments, cellAssignments)
	nextPlateID := len(plates)

	for i := 0; i < numToSplit && i < len(candidates); i++ {
		plateToSplit := candidates[i].index

		// Find cells of this plate
		plateCells := make([]int, 0)
		for cellIdx, plateIdx := range newAssignments {
			if plateIdx == plateToSplit {
				plateCells = append(plateCells, cellIdx)
			}
		}

		if len(plateCells) < 20 {
			continue // Too small to split
		}

		// Take ~10-15% of cells from edge and make new plate
		splitSize := len(plateCells) / 8
		if splitSize < 5 {
			splitSize = 5
		}

		// Find edge cells (have neighbors in other plates)
		edgeCells := make([]int, 0)
		for _, cellIdx := range plateCells {
			hasExternalNeighbor := false
			for _, neighborIdx := range voronoiCells[cellIdx].NeighborSiteIndices {
				if int(neighborIdx) < len(newAssignments) {
					if newAssignments[neighborIdx] != plateToSplit {
						hasExternalNeighbor = true
						break
					}
				}
			}
			if hasExternalNeighbor {
				edgeCells = append(edgeCells, cellIdx)
			}
		}

		// Assign first N edge cells to new plate
		for j := 0; j < splitSize && j < len(edgeCells); j++ {
			newAssignments[edgeCells[j]] = nextPlateID
		}
		nextPlateID++
	}

	newPlates := createPlatesFromAssignments(voronoiCells, icosphereSites, newAssignments, []int{}, planetRadius, rng)
	return newPlates, newAssignments
}

// mergeTinyPlates merges plates <0.01% into neighbors
func mergeTinyPlates(
	plates []TectonicPlate,
	cellAssignments []int,
	voronoiCells []VoronoiCell,
	icosphereSites []Vector3D,
	planetRadius float64,
	rng *rand.Rand,
) ([]TectonicPlate, []int) {

	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius
	mergeMap := make(map[int]int)
	for i := range cellAssignments {
		mergeMap[i] = i
	}

	for i, plate := range plates {
		percent := (plate.Area / sphereArea) * 100.0
		if percent < 0.01 {
			// Find cells
			plateCells := make([]int, 0)
			for cellIdx, plateIdx := range cellAssignments {
				if plateIdx == i {
					plateCells = append(plateCells, cellIdx)
				}
			}

			// Find any neighbor
			for _, cellIdx := range plateCells {
				for _, neighborIdx := range voronoiCells[cellIdx].NeighborSiteIndices {
					if int(neighborIdx) < len(cellAssignments) {
						neighborPlate := cellAssignments[neighborIdx]
						if neighborPlate != i && neighborPlate >= 0 {
							mergeMap[i] = neighborPlate
							goto nextPlate
						}
					}
				}
			}
		nextPlate:
		}
	}

	// Apply merges
	newAssignments := make([]int, len(cellAssignments))
	for i, plateIdx := range cellAssignments {
		newPlate := plateIdx
		for mergeMap[newPlate] != newPlate {
			newPlate = mergeMap[newPlate]
		}
		newAssignments[i] = newPlate
	}

	newAssignments = renumberAssignments(newAssignments)
	newPlates := createPlatesFromAssignments(voronoiCells, icosphereSites, newAssignments, []int{}, planetRadius, rng)

	return newPlates, newAssignments
}

// renumberAssignments makes plate indices contiguous (0, 1, 2, ...)
func renumberAssignments(assignments []int) []int {
	oldToNew := make(map[int]int)
	newIdx := 0

	for _, oldIdx := range assignments {
		if _, exists := oldToNew[oldIdx]; !exists {
			oldToNew[oldIdx] = newIdx
			newIdx++
		}
	}

	result := make([]int, len(assignments))
	for i, oldIdx := range assignments {
		result[i] = oldToNew[oldIdx]
	}

	return result
}
