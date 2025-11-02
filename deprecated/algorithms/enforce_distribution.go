package tectonics

import (
	"math"
	"math/rand"
)

type plateInfo struct {
	idx     int
	size    float64
	percent float64
	tier    string
}

// EnforceExactDistribution surgically adjusts plates to hit exact major/minor/micro counts
// This provides truly deterministic control over the distribution
func EnforceExactDistribution(
	plates []TectonicPlate,
	cellAssignments []int,
	voronoiCells []VoronoiCell,
	icosphereSites []Vector3D,
	targetMajor, targetMinor, targetMicro int,
	planetRadius float64,
	rng *rand.Rand,
) ([]TectonicPlate, []int) {

	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius
	cellArea := sphereArea / float64(len(voronoiCells))

	// Iteratively adjust until we hit exact targets
	for iteration := 0; iteration < 100; iteration++ {
		// Recalculate plate sizes and classifications
		plateSizes := make([]float64, len(plates))
		for cellIdx, plateIdx := range cellAssignments {
			if plateIdx >= 0 && cellIdx < len(voronoiCells) {
				plateSizes[plateIdx] += cellArea
			}
		}

		// Count by tier
		plateInfos := make([]plateInfo, len(plates))
		majorCount, minorCount, microCount := 0, 0, 0

		for i, size := range plateSizes {
			percent := (size / sphereArea) * 100.0
			var tier string
			if percent >= 6.0 {
				tier = "major"
				majorCount++
			} else if percent >= 0.18 {
				tier = "minor"
				minorCount++
			} else {
				tier = "micro"
				microCount++
			}
			plateInfos[i] = plateInfo{i, size, percent, tier}
		}

		// Check if we've achieved targets
		if majorCount == targetMajor && minorCount == targetMinor && microCount == targetMicro {
			break // Success!
		}

		// Determine what needs to change
		majorGap := targetMajor - majorCount
		minorGap := targetMinor - minorCount
		microGap := targetMicro - microCount

		// Strategy: Move cells between plates to change their tier classifications
		if majorGap > 0 {
			// Need more majors - grow largest minors into major tier
			growMinorsIntoMajors(plateInfos, cellAssignments, voronoiCells, sphereArea, cellArea, majorGap)
		} else if majorGap < 0 {
			// Too many majors - shrink smallest majors into minor tier
			shrinkMajorsIntoMinors(plateInfos, cellAssignments, voronoiCells, sphereArea, cellArea, -majorGap)
		} else if minorGap > 0 {
			// Need more minors - grow largest micros or shrink smallest majors
			if microCount > targetMicro {
				growMicrosIntoMinors(plateInfos, cellAssignments, voronoiCells, sphereArea, cellArea, minorGap)
			}
		} else if minorGap < 0 {
			// Too many minors - convert to micros or majors as needed
			if microGap > 0 {
				shrinkMinorsIntoMicros(plateInfos, cellAssignments, voronoiCells, sphereArea, cellArea, -minorGap)
			}
		}
	}

	// Rebuild plates from adjusted assignments
	adjustedPlates := createPlatesFromAssignments(
		voronoiCells,
		icosphereSites,
		cellAssignments,
		[]int{},
		planetRadius,
		rng,
	)

	return adjustedPlates, cellAssignments
}

// Helper functions for tier adjustments

func growMinorsIntoMajors(
	plateInfos []plateInfo,
	cellAssignments []int,
	voronoiCells []VoronoiCell,
	sphereArea, cellArea float64,
	count int,
) {
	// Find largest minors close to major threshold (6%)
	minors := make([]plateInfo, 0)
	for _, info := range plateInfos {
		if info.tier == "minor" && info.percent > 4.0 {
			minors = append(minors, info)
		}
	}

	// Sort by size (largest first)
	for i := 0; i < len(minors)-1; i++ {
		for j := i + 1; j < len(minors); j++ {
			if minors[j].percent > minors[i].percent {
				minors[i], minors[j] = minors[j], minors[i]
			}
		}
	}

	// Grow the largest minors
	for i := 0; i < count && i < len(minors); i++ {
		plateIdx := minors[i].idx
		targetSize := sphereArea * 0.061 // Just above major threshold

		// Steal cells from neighbors until we reach target
		stealCellsFromNeighbors(plateIdx, cellAssignments, voronoiCells, targetSize, cellArea)
	}
}

func shrinkMajorsIntoMinors(
	plateInfos []plateInfo,
	cellAssignments []int,
	voronoiCells []VoronoiCell,
	sphereArea, cellArea float64,
	count int,
) {
	// Find smallest majors
	majors := make([]plateInfo, 0)
	for _, info := range plateInfos {
		if info.tier == "major" {
			majors = append(majors, info)
		}
	}

	// Sort by size (smallest first)
	for i := 0; i < len(majors)-1; i++ {
		for j := i + 1; j < len(majors); j++ {
			if majors[j].percent < majors[i].percent {
				majors[i], majors[j] = majors[j], majors[i]
			}
		}
	}

	// Shrink smallest majors
	for i := 0; i < count && i < len(majors); i++ {
		plateIdx := majors[i].idx
		targetSize := sphereArea * 0.059 // Just below major threshold

		// Give cells to neighbors
		giveCellsToNeighbors(plateIdx, cellAssignments, voronoiCells, targetSize, cellArea)
	}
}

func growMicrosIntoMinors(
	plateInfos []plateInfo,
	cellAssignments []int,
	voronoiCells []VoronoiCell,
	sphereArea, cellArea float64,
	count int,
) {
	// Find largest micros
	micros := make([]plateInfo, 0)
	for _, info := range plateInfos {
		if info.tier == "micro" && info.percent > 0.1 {
			micros = append(micros, info)
		}
	}

	// Sort by size (largest first)
	for i := 0; i < len(micros)-1; i++ {
		for j := i + 1; j < len(micros); j++ {
			if micros[j].percent > micros[i].percent {
				micros[i], micros[j] = micros[j], micros[i]
			}
		}
	}

	// Grow largest micros
	for i := 0; i < count && i < len(micros); i++ {
		plateIdx := micros[i].idx
		targetSize := sphereArea * 0.0019 // Just above minor threshold

		stealCellsFromNeighbors(plateIdx, cellAssignments, voronoiCells, targetSize, cellArea)
	}
}

func shrinkMinorsIntoMicros(
	plateInfos []plateInfo,
	cellAssignments []int,
	voronoiCells []VoronoiCell,
	sphereArea, cellArea float64,
	count int,
) {
	// Find smallest minors
	minors := make([]plateInfo, 0)
	for _, info := range plateInfos {
		if info.tier == "minor" {
			minors = append(minors, info)
		}
	}

	// Sort by size (smallest first)
	for i := 0; i < len(minors)-1; i++ {
		for j := i + 1; j < len(minors); j++ {
			if minors[j].percent < minors[i].percent {
				minors[i], minors[j] = minors[j], minors[i]
			}
		}
	}

	// Shrink smallest minors
	for i := 0; i < count && i < len(minors); i++ {
		plateIdx := minors[i].idx
		targetSize := sphereArea * 0.0017 // Just below minor threshold

		giveCellsToNeighbors(plateIdx, cellAssignments, voronoiCells, targetSize, cellArea)
	}
}

func stealCellsFromNeighbors(
	plateIdx int,
	cellAssignments []int,
	voronoiCells []VoronoiCell,
	targetSize, cellArea float64,
) {
	currentSize := 0.0
	for cellIdx, pIdx := range cellAssignments {
		if pIdx == plateIdx && cellIdx < len(voronoiCells) {
			currentSize += cellArea
		}
	}

	for currentSize < targetSize {
		// Find neighbor cells we can steal
		for cellIdx, pIdx := range cellAssignments {
			if pIdx != plateIdx && cellIdx < len(voronoiCells) {
				// Check if this cell is adjacent to our plate
				for _, neighborIdx := range voronoiCells[cellIdx].NeighborSiteIndices {
					if int(neighborIdx) < len(cellAssignments) && cellAssignments[neighborIdx] == plateIdx {
						// Steal this cell
						cellAssignments[cellIdx] = plateIdx
						currentSize += cellArea
						if currentSize >= targetSize {
							return
						}
						break
					}
				}
			}
		}
		break // No more cells to steal
	}
}

func giveCellsToNeighbors(
	plateIdx int,
	cellAssignments []int,
	voronoiCells []VoronoiCell,
	targetSize, cellArea float64,
) {
	currentSize := 0.0
	for cellIdx, pIdx := range cellAssignments {
		if pIdx == plateIdx && cellIdx < len(voronoiCells) {
			currentSize += cellArea
		}
	}

	for currentSize > targetSize {
		// Find edge cells to give away
		for cellIdx, pIdx := range cellAssignments {
			if pIdx == plateIdx && cellIdx < len(voronoiCells) {
				// Check if edge cell
				for _, neighborIdx := range voronoiCells[cellIdx].NeighborSiteIndices {
					if int(neighborIdx) < len(cellAssignments) {
						neighborPlate := cellAssignments[neighborIdx]
						if neighborPlate != plateIdx && neighborPlate >= 0 {
							// Give this cell to neighbor
							cellAssignments[cellIdx] = neighborPlate
							currentSize -= cellArea
							if currentSize <= targetSize {
								return
							}
							break
						}
					}
				}
			}
		}
		break // No more cells to give
	}
}
