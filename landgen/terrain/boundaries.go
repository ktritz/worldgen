package terrain

// Plate boundary detection and classification
// Identifies mountains, trenches, ridges, coastlines based on plate interactions

import (
	"fmt"
	"math"
)

// BoundarySeeds holds the different types of boundary region seeds
type BoundarySeeds struct {
	Mountain  map[int]bool // Continental collision / volcanic arc
	Coastline map[int]bool // Ocean-land boundaries
	Ocean     map[int]bool // Generic ocean regions
	Ridge     map[int]bool // Mid-ocean spreading ridges
	Trench    map[int]bool // Subduction trenches
}

// FindCollisions classifies boundary regions based on plate interactions
// Uses rotational plate motion (Euler poles) to compute velocity at each boundary point
func FindCollisions(
	sites []Vector3D,
	cells []VoronoiCell,
	plateIsOcean map[int]bool,
	rPlate []int,
	plateRot map[int]PlateRotation,
) BoundarySeeds {
	numRegions := len(sites)
	seeds := BoundarySeeds{
		Mountain:  make(map[int]bool),
		Coastline: make(map[int]bool),
		Ocean:     make(map[int]bool),
		Ridge:     make(map[int]bool),
		Trench:    make(map[int]bool),
	}

	// Debug counters
	var oceanLandBoundaryCount, oceanLandConvergentCount int

	// First pass: classify all boundary regions
	for currentR := 0; currentR < numRegions; currentR++ {
		bestCompression := math.Inf(-1)
		bestR := -1
		bestDistBefore := 0.0

		// Find the neighbor from a different plate with maximum compression
		for _, neighborIdx := range cells[currentR].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR >= numRegions {
				continue
			}
			if rPlate[currentR] != rPlate[neighborR] {
				currentPos := sites[currentR]
				neighborPos := sites[neighborR]

				currentPlate := rPlate[currentR]
				neighborPlate := rPlate[neighborR]

				// Compute velocity at each position using Euler pole rotation
				currentVec := plateRot[currentPlate].VelocityAt(currentPos)
				neighborVec := plateRot[neighborPlate].VelocityAt(neighborPos)

				// Simulate movement
				distBefore := Distance(currentPos, neighborPos)
				currentAfter := Vector3D{
					X: currentPos.X + currentVec.X*DeltaTime,
					Y: currentPos.Y + currentVec.Y*DeltaTime,
					Z: currentPos.Z + currentVec.Z*DeltaTime,
				}
				neighborAfter := Vector3D{
					X: neighborPos.X + neighborVec.X*DeltaTime,
					Y: neighborPos.Y + neighborVec.Y*DeltaTime,
					Z: neighborPos.Z + neighborVec.Z*DeltaTime,
				}
				distAfter := Distance(currentAfter, neighborAfter)

				compression := distBefore - distAfter
				if compression > bestCompression {
					bestCompression = compression
					bestR = neighborR
					bestDistBefore = distBefore
				}
			}
		}

		if bestR == -1 {
			continue
		}

		// Normalize compression by cell distance to be resolution-independent
		// This makes the collision detection work consistently across mesh resolutions
		normalizedCompression := bestCompression / (bestDistBefore * DeltaTime)
		collided := normalizedCompression > CollisionThreshold
		currentPlate := rPlate[currentR]
		bestPlate := rPlate[bestR]
		currentIsOcean := plateIsOcean[currentPlate]
		bestIsOcean := plateIsOcean[bestPlate]

		if currentIsOcean && bestIsOcean {
			// Ocean-Ocean boundary
			if collided {
				seeds.Coastline[currentR] = true // Island arc
			} else {
				seeds.Ocean[currentR] = true
			}
		} else if !currentIsOcean && !bestIsOcean {
			// Land-Land boundary
			if collided {
				seeds.Mountain[currentR] = true // Continental collision (Himalayas)
			} else {
				seeds.Coastline[currentR] = true // Rift valley
			}
		} else {
			// Land-Ocean boundary
			oceanLandBoundaryCount++
			if currentIsOcean {
				if collided {
					oceanLandConvergentCount++
					seeds.Trench[currentR] = true
					seeds.Ocean[currentR] = true
				} else {
					seeds.Coastline[currentR] = true
				}
			} else {
				// Continental side - always coastline, volcanic arc placed inland later
				seeds.Coastline[currentR] = true
			}
		}
	}

	fmt.Printf("    Ocean-land boundaries: %d total, %d convergent (%.1f%%)\n",
		oceanLandBoundaryCount, oceanLandConvergentCount,
		100*float64(oceanLandConvergentCount)/float64(oceanLandBoundaryCount+1))

	// Second pass: classify ridges from ocean seeds
	for currentR := range seeds.Ocean {
		for _, neighborIdx := range cells[currentR].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR >= numRegions {
				continue
			}
			if rPlate[currentR] != rPlate[neighborR] {
				currentPlate := rPlate[currentR]
				neighborPlate := rPlate[neighborR]
				currentIsOcean := plateIsOcean[currentPlate]
				neighborIsOcean := plateIsOcean[neighborPlate]

				if currentIsOcean && neighborIsOcean {
					seeds.Ridge[currentR] = true
				} else if currentIsOcean && !neighborIsOcean {
					seeds.Trench[currentR] = true
				}
				break
			}
		}
	}

	// Third pass: place volcanic arc mountains INLAND from subduction trenches
	placeVolcanicArcs(sites, cells, rPlate, plateIsOcean, seeds.Trench, seeds.Mountain)

	return seeds
}

// placeVolcanicArcs places mountain seeds inland from trenches
// Real subduction: trench -> forearc basin -> volcanic arc ~100-300km inland
// Only places arcs for representative trenches (not every single trench cell)
func placeVolcanicArcs(
	sites []Vector3D,
	cells []VoronoiCell,
	rPlate []int,
	plateIsOcean map[int]bool,
	trenchR map[int]bool,
	mountainR map[int]bool,
) {
	numRegions := len(cells)

	// Cluster trenches - only use representative ones spaced well apart
	// This prevents creating hundreds of overlapping volcanic arcs
	// On Earth, there are ~10-20 major volcanic arc segments, not hundreds
	usedTrenches := make(map[int]bool)
	minTrenchSpacing := VolcanoDistanceRadians * 3.0 // Much larger spacing between volcanic arcs

	for trenchRegion := range trenchR {
		// Skip if too close to an already-used trench
		tooClose := false
		for usedTrench := range usedTrenches {
			if Distance(sites[trenchRegion], sites[usedTrench]) < minTrenchSpacing {
				tooClose = true
				break
			}
		}
		if tooClose {
			continue
		}
		usedTrenches[trenchRegion] = true

		trenchPos := sites[trenchRegion]

		// Find the continental neighbor of this trench
		for _, neighborIdx := range cells[trenchRegion].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR >= numRegions {
				continue
			}
			neighborPlate := rPlate[neighborR]
			if !plateIsOcean[neighborPlate] {
				// BFS inland to find volcanic arc location at target distance
				visited := make(map[int]bool)
				queue := []int{neighborR}
				visited[neighborR] = true
				bestRegion := neighborR
				bestDistDiff := math.Abs(Distance(sites[neighborR], trenchPos) - VolcanoDistanceRadians)

				for len(queue) > 0 {
					current := queue[0]
					queue = queue[1:]

					currentDist := Distance(sites[current], trenchPos)

					// If we've gone too far past the target, stop searching
					if currentDist > VolcanoDistanceRadians*1.5 {
						continue
					}

					// Track region closest to target distance
					distDiff := math.Abs(currentDist - VolcanoDistanceRadians)
					if distDiff < bestDistDiff {
						bestDistDiff = distDiff
						bestRegion = current
					}

					for _, nextIdx := range cells[current].NeighborSiteIndices {
						nextR := int(nextIdx)
						if nextR >= numRegions || visited[nextR] {
							continue
						}
						if rPlate[nextR] == neighborPlate {
							visited[nextR] = true
							queue = append(queue, nextR)
						}
					}
				}

				mountainR[bestRegion] = true
				break
			}
		}
	}
}

// Distance computes Euclidean distance between two 3D points
func Distance(a, b Vector3D) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}
