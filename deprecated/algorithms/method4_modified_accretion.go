package tectonics

import (
	"math"
	"math/rand"
)

// Method4ModifiedAccretion implements heavy overseeding + intelligent merging
// Expected score: 0.60-0.75
func Method4ModifiedAccretion(
	voronoiCells []VoronoiCell,
	icosphereSites []Vector3D,
	planetRadius float64,
	settings Method4Settings,
) ([]TectonicPlate, []int, error) {

	rng := rand.New(rand.NewSource(settings.Seed))

	// Step 1: Heavy overseeding (10-20x target plate count)
	initialSeeds := settings.TargetPlateCount * settings.OverseedingFactor
	seeds := make([]int, initialSeeds)

	// Random well-distributed seeds
	used := make(map[int]bool)
	for i := 0; i < initialSeeds; i++ {
		for {
			candidate := rng.Intn(len(icosphereSites))
			if !used[candidate] {
				seeds[i] = candidate
				used[candidate] = true
				break
			}
		}
	}

	// Step 2: Initial Voronoi assignment
	cellAssignments := make([]int, len(voronoiCells))
	for cellIdx := range voronoiCells {
		if cellIdx >= len(icosphereSites) {
			continue
		}

		bestPlate := 0
		bestDist := math.Inf(1)

		for plateIdx, seedIdx := range seeds {
			if seedIdx >= len(icosphereSites) {
				continue
			}

			dist := CalculateSphericalDistance(
				icosphereSites[cellIdx],
				icosphereSites[seedIdx],
				planetRadius,
			)

			if dist < bestDist {
				bestDist = dist
				bestPlate = plateIdx
			}
		}

		cellAssignments[cellIdx] = bestPlate
	}

	// Step 3: Multi-phase quota-based merging
	// Phase 1: Create 7 major plates
	// Phase 2: Create 13 minor plates
	// Phase 3: Keep remaining as micro plates
	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius
	cellArea := sphereArea / float64(len(voronoiCells))

	// Phase 1: Aggressively merge to create 7 major plates (each >= 6% of sphere)
	for {
		// Calculate current plate sizes
		plateSizes := make(map[int]float64)
		for cellIdx, plateIdx := range cellAssignments {
			if cellIdx < len(voronoiCells) {
				plateSizes[plateIdx] += cellArea
			}
		}

		// Count how many major plates we have
		majorCount := 0
		for _, size := range plateSizes {
			if size/sphereArea >= 0.06 {
				majorCount++
			}
		}

		if majorCount >= 7 {
			break // We have enough major plates
		}

		// Find two large plates to merge into a major
		// Strategy: Merge the two largest non-major plates
		type plateSizeInfo struct {
			idx  int
			size float64
		}

		nonMajors := make([]plateSizeInfo, 0)
		for idx, size := range plateSizes {
			if size/sphereArea < 0.06 {
				nonMajors = append(nonMajors, plateSizeInfo{idx, size})
			}
		}

		if len(nonMajors) < 2 {
			break // Can't merge anymore
		}

		// Sort by size (largest first)
		for i := 0; i < len(nonMajors)-1; i++ {
			for j := i + 1; j < len(nonMajors); j++ {
				if nonMajors[j].size > nonMajors[i].size {
					nonMajors[i], nonMajors[j] = nonMajors[j], nonMajors[i]
				}
			}
		}

		// Merge the two largest non-majors if they're neighbors
		plate1 := nonMajors[0].idx
		plate2 := nonMajors[1].idx

		// Check if they're neighbors
		areNeighbors := false
		for cellIdx, plateIdx := range cellAssignments {
			if plateIdx == plate1 && cellIdx < len(voronoiCells) {
				for _, neighborIdx := range voronoiCells[cellIdx].NeighborSiteIndices {
					if int(neighborIdx) < len(cellAssignments) && cellAssignments[neighborIdx] == plate2 {
						areNeighbors = true
						break
					}
				}
			}
			if areNeighbors {
				break
			}
		}

		if areNeighbors {
			// Merge plate2 into plate1
			for i, plateIdx := range cellAssignments {
				if plateIdx == plate2 {
					cellAssignments[i] = plate1
				}
			}
		} else {
			// If not neighbors, merge largest non-major with ANY neighbor
			foundMerge := false
			for cellIdx, plateIdx := range cellAssignments {
				if plateIdx == plate1 && cellIdx < len(voronoiCells) {
					for _, neighborIdx := range voronoiCells[cellIdx].NeighborSiteIndices {
						if int(neighborIdx) < len(cellAssignments) {
							neighborPlate := cellAssignments[neighborIdx]
							if neighborPlate != plate1 && plateSizes[neighborPlate]/sphereArea < 0.06 {
								// Merge neighbor into plate1
								for i, p := range cellAssignments {
									if p == neighborPlate {
										cellAssignments[i] = plate1
									}
								}
								foundMerge = true
								break
							}
						}
					}
					if foundMerge {
						break
					}
				}
			}
			if !foundMerge {
				break // Can't find anything to merge
			}
		}
	}

	// Phase 2: Intelligent merging to preserve plate tiers
	// Strategy: Only merge plates that are too small to be micro (<0.01%)
	// This preserves the natural size distribution while removing tiny artifacts
	minMicroSize := sphereArea * 0.0001  // 0.01% of sphere

	for {
		plateSizes := make(map[int]float64)
		plateIDs := make([]int, 0)
		for cellIdx, plateIdx := range cellAssignments {
			if cellIdx < len(voronoiCells) {
				if plateSizes[plateIdx] == 0 {
					plateIDs = append(plateIDs, plateIdx)
				}
				plateSizes[plateIdx] += cellArea
			}
		}

		// Count plates by tier
		majorCount, minorCount, microCount := 0, 0, 0
		for _, size := range plateSizes {
			percent := size / sphereArea
			if percent >= 0.06 {
				majorCount++
			} else if percent >= 0.0018 {
				minorCount++
			} else if percent >= 0.0001 {
				microCount++
			}
		}

		// Stop if we have good distribution
		// Target: 7 major, 13 minor, 19 micro = 39 total
		// Allow flexibility but enforce upper limit on total
		totalPlates := len(plateIDs)
		if totalPlates <= 50 &&  // Upper limit to prevent explosion
			majorCount >= 6 && majorCount <= 8 &&
			minorCount >= 10 && minorCount <= 16 &&
			microCount >= 15 && microCount <= 25 {
			break
		}

		// If we have way too many plates, be more aggressive
		if totalPlates > 100 {
			minMicroSize = sphereArea * 0.001  // Merge anything < 0.1%
		}

		// Find smallest plate below micro threshold
		smallestPlate := -1
		smallestSize := math.Inf(1)
		for _, plateID := range plateIDs {
			if plateSizes[plateID] < minMicroSize && plateSizes[plateID] < smallestSize {
				smallestSize = plateSizes[plateID]
				smallestPlate = plateID
			}
		}

		if smallestPlate < 0 {
			// No plates below threshold, try merging excessive minors if we have too many
			if minorCount > 16 {
				// Find smallest minor
				for _, plateID := range plateIDs {
					size := plateSizes[plateID]
					percent := size / sphereArea
					if percent >= 0.0018 && percent < 0.06 && size < smallestSize {
						smallestSize = size
						smallestPlate = plateID
					}
				}
			}
		}

		if smallestPlate < 0 {
			break // Nothing to merge
		}

		// Merge into any neighbor
		merged := false
		for cellIdx, plateIdx := range cellAssignments {
			if plateIdx == smallestPlate && cellIdx < len(voronoiCells) {
				for _, neighborIdx := range voronoiCells[cellIdx].NeighborSiteIndices {
					if int(neighborIdx) < len(cellAssignments) {
						neighborPlate := cellAssignments[neighborIdx]
						if neighborPlate != smallestPlate {
							// Merge into neighbor
							for i, p := range cellAssignments {
								if p == smallestPlate {
									cellAssignments[i] = neighborPlate
								}
							}
							merged = true
							break
						}
					}
				}
				if merged {
					break
				}
			}
		}

		if !merged {
			break
		}
	}

	// Renumber to make plate indices contiguous
	cellAssignments = renumberAssignments(cellAssignments)

	// Create plate structures
	plates := createPlatesFromAssignments(
		voronoiCells,
		icosphereSites,
		cellAssignments,
		[]int{},
		planetRadius,
		rng,
	)

	return plates, cellAssignments, nil
}

// Method4Settings controls modified accretion generation
type Method4Settings struct {
	TargetPlateCount   int
	OverseedingFactor  int     // Multiplier for initial seeds (10-20)
	Seed               int64
}

// DefaultMethod4Settings returns reasonable defaults
func DefaultMethod4Settings() Method4Settings {
	return Method4Settings{
		TargetPlateCount:  39,
		OverseedingFactor: 15,  // 15x overseeding
		Seed:              12345,
	}
}

type plateSizeTarget struct {
	percent      float64
	continental  bool
}
