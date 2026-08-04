package terrain

// Plate boundary detection and classification
// Identifies mountains, trenches, ridges, coastlines based on plate interactions

import (
	"fmt"
	"math"
	"sort"
)

// BoundarySeeds holds the different types of boundary region seeds
type BoundarySeeds struct {
	Mountain  map[int]bool // Continental collision / volcanic arc
	Collision map[int]bool // Continental collision belts / sutures
	Arc       map[int]bool // Subduction volcanic arcs
	Coastline map[int]bool // Ocean-land boundaries
	Ocean     map[int]bool // Generic ocean regions
	Ridge     map[int]bool // Mid-ocean spreading ridges
	Trench    map[int]bool // Subduction trenches
	Rift      map[int]bool // Divergent continental boundaries
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
		Collision: make(map[int]bool),
		Arc:       make(map[int]bool),
		Coastline: make(map[int]bool),
		Ocean:     make(map[int]bool),
		Ridge:     make(map[int]bool),
		Trench:    make(map[int]bool),
		Rift:      make(map[int]bool),
	}

	// Debug counters
	var oceanLandBoundaryCount, oceanLandConvergentCount, continentalRiftCount int
	collisionBoundary := make(map[int]bool)

	// First pass: classify all boundary regions
	for currentR := 0; currentR < numRegions; currentR++ {
		bestCompression := math.Inf(-1)
		bestDivergence := math.Inf(1)
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
				if compression < bestDivergence {
					bestDivergence = compression
				}
			}
		}

		if bestR == -1 {
			continue
		}

		// Normalize compression by cell distance to be resolution-independent
		// This makes the collision detection work consistently across mesh resolutions
		normalizedCompression := bestCompression / (bestDistBefore * DeltaTime)
		normalizedDivergence := bestDivergence / (bestDistBefore * DeltaTime)
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
				seeds.Collision[currentR] = true
				collisionBoundary[currentR] = true
			} else if normalizedDivergence < -DivergenceThreshold {
				seeds.Rift[currentR] = true
				seeds.Coastline[currentR] = true
				continentalRiftCount++
			} else {
				seeds.Coastline[currentR] = true // Passive continental contact / transform margin
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
	if continentalRiftCount > 0 {
		fmt.Printf("    Continental rift seeds: %d\n", continentalRiftCount)
	}

	// Expand continent-continent sutures inland on both sides so collision
	// mountains read as long fold belts rather than a one-cell boundary trace.
	placeContinentalCollisionBelts(sites, cells, rPlate, plateIsOcean, collisionBoundary, seeds.Collision, seeds.Mountain)

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
	placeVolcanicArcs(sites, cells, rPlate, plateIsOcean, seeds.Trench, seeds.Arc, seeds.Mountain)

	return seeds
}

func placeContinentalCollisionBelts(
	sites []Vector3D,
	cells []VoronoiCell,
	rPlate []int,
	plateIsOcean map[int]bool,
	collisionBoundary map[int]bool,
	collisionR map[int]bool,
	mountainR map[int]bool,
) {
	if len(collisionBoundary) == 0 {
		return
	}

	type queueItem struct {
		region     int
		sourceSite int
	}

	distFromSuture := make([]float64, len(cells))
	for i := range distFromSuture {
		distFromSuture[i] = math.Inf(1)
	}

	queue := make([]queueItem, 0, len(collisionBoundary))
	for r := range collisionBoundary {
		if r < 0 || r >= len(cells) || plateIsOcean[rPlate[r]] {
			continue
		}
		distFromSuture[r] = 0
		queue = append(queue, queueItem{region: r, sourceSite: r})
	}

	for queueIdx := 0; queueIdx < len(queue); queueIdx++ {
		item := queue[queueIdx]
		if distFromSuture[item.region] > CollisionBeltDistanceRadians {
			continue
		}

		// Keep the immediate suture as the strongest seed, but broaden the belt
		// on both sides with nearby same-plate regions.
		if distFromSuture[item.region] <= CollisionBeltDistanceRadians*0.85 {
			collisionR[item.region] = true
			mountainR[item.region] = true
		}

		sourcePlate := rPlate[item.region]
		for _, neighborIdx := range cells[item.region].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR < 0 || neighborR >= len(cells) || plateIsOcean[rPlate[neighborR]] {
				continue
			}
			// Expand within the same continental plate so each side of the suture
			// grows its own fold belt rather than jumping across the collision.
			if rPlate[neighborR] != sourcePlate {
				continue
			}

			neighborDist := Distance(sites[neighborR], sites[item.sourceSite])
			if neighborDist < distFromSuture[neighborR] {
				distFromSuture[neighborR] = neighborDist
				queue = append(queue, queueItem{region: neighborR, sourceSite: item.sourceSite})
			}
		}
	}

	connectSeedCentersOnPlate(
		sites,
		cells,
		rPlate,
		collisionBoundary,
		CollisionLinkDistanceRadians,
		2,
		CollisionBeltDistanceRadians*0.55,
		collisionR,
		mountainR,
	)
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
	arcR map[int]bool,
	mountainR map[int]bool,
) {
	numRegions := len(cells)

	// Iterate trench regions in sorted order: the greedy spacing filter below is
	// order-sensitive, and Go map iteration order is randomized, so iterating
	// the map directly would make arc placement nondeterministic per seed.
	trenchRegions := make([]int, 0, len(trenchR))
	for trenchRegion := range trenchR {
		trenchRegions = append(trenchRegions, trenchRegion)
	}
	sort.Ints(trenchRegions)

	usedArcSites := make([]int, 0, len(trenchR))
	arcCenters := make(map[int]bool, len(trenchR))
	for _, trenchRegion := range trenchRegions {
		tooClose := false
		for _, usedArc := range usedArcSites {
			if Distance(sites[trenchRegion], sites[usedArc]) < ArcSeedSpacingRadians {
				tooClose = true
				break
			}
		}
		if tooClose {
			continue
		}
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

				widenMountainSeed(sites, cells, bestRegion, neighborPlate, ArcHalfWidthRadians, rPlate, arcR)
				widenMountainSeed(sites, cells, bestRegion, neighborPlate, ArcHalfWidthRadians, rPlate, mountainR)
				usedArcSites = append(usedArcSites, bestRegion)
				arcCenters[bestRegion] = true
				break
			}
		}
	}

	connectSeedCentersOnPlate(
		sites,
		cells,
		rPlate,
		arcCenters,
		ArcLinkDistanceRadians,
		1,
		ArcHalfWidthRadians*0.8,
		arcR,
		mountainR,
	)
}

func connectSeedCentersOnPlate(
	sites []Vector3D,
	cells []VoronoiCell,
	rPlate []int,
	seedCenters map[int]bool,
	maxLinkDistance float64,
	maxLinksPerSeed int,
	linkHalfWidth float64,
	targetSets ...map[int]bool,
) {
	if len(seedCenters) < 2 || maxLinksPerSeed <= 0 {
		return
	}

	centersByPlate := make(map[int][]int)
	for center := range seedCenters {
		if center < 0 || center >= len(rPlate) {
			continue
		}
		centersByPlate[rPlate[center]] = append(centersByPlate[rPlate[center]], center)
	}

	linkedPairs := make(map[[2]int]bool)
	for plateID, centers := range centersByPlate {
		for _, center := range centers {
			type candidate struct {
				region int
				dist   float64
			}

			candidates := make([]candidate, 0, len(centers))
			for _, other := range centers {
				if other == center {
					continue
				}
				dist := angularDistance(sites[center], sites[other])
				if dist <= maxLinkDistance {
					candidates = append(candidates, candidate{region: other, dist: dist})
				}
			}
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].dist < candidates[j].dist
			})

			links := 0
			for _, candidate := range candidates {
				pair := orderedPair(center, candidate.region)
				if linkedPairs[pair] {
					continue
				}

				path := shortestPathWithinPlate(cells, rPlate, plateID, center, candidate.region)
				if len(path) == 0 {
					continue
				}
				markLinkedSeedPath(sites, cells, rPlate, plateID, path, linkHalfWidth, targetSets...)
				linkedPairs[pair] = true
				links++
				if links >= maxLinksPerSeed {
					break
				}
			}
		}
	}
}

func orderedPair(a, b int) [2]int {
	if a > b {
		a, b = b, a
	}
	return [2]int{a, b}
}

func shortestPathWithinPlate(cells []VoronoiCell, rPlate []int, plateID int, start, goal int) []int {
	if start == goal {
		return []int{start}
	}

	prev := make([]int, len(cells))
	for i := range prev {
		prev[i] = -1
	}
	visited := make([]bool, len(cells))
	queue := []int{start}
	visited[start] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, neighborIdx := range cells[current].NeighborSiteIndices {
			neighbor := int(neighborIdx)
			if neighbor < 0 || neighbor >= len(cells) || visited[neighbor] || rPlate[neighbor] != plateID {
				continue
			}
			visited[neighbor] = true
			prev[neighbor] = current
			if neighbor == goal {
				path := []int{goal}
				for node := goal; prev[node] != -1; node = prev[node] {
					path = append(path, prev[node])
				}
				for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
					path[i], path[j] = path[j], path[i]
				}
				return path
			}
			queue = append(queue, neighbor)
		}
	}

	return nil
}

func markLinkedSeedPath(
	sites []Vector3D,
	cells []VoronoiCell,
	rPlate []int,
	plateID int,
	path []int,
	linkHalfWidth float64,
	targetSets ...map[int]bool,
) {
	for _, region := range path {
		for _, targetSet := range targetSets {
			targetSet[region] = true
		}
		if linkHalfWidth <= 0 {
			continue
		}
		for _, targetSet := range targetSets {
			widenMountainSeed(sites, cells, region, plateID, linkHalfWidth, rPlate, targetSet)
		}
	}
}

func widenMountainSeed(
	sites []Vector3D,
	cells []VoronoiCell,
	centerRegion int,
	plateID int,
	maxDistance float64,
	rPlate []int,
	mountainR map[int]bool,
) {
	type queueItem struct {
		region int
	}

	visited := map[int]bool{centerRegion: true}
	queue := []queueItem{{region: centerRegion}}

	for queueIdx := 0; queueIdx < len(queue); queueIdx++ {
		current := queue[queueIdx].region
		if Distance(sites[current], sites[centerRegion]) > maxDistance {
			continue
		}

		mountainR[current] = true
		for _, neighborIdx := range cells[current].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR < 0 || neighborR >= len(cells) || visited[neighborR] || rPlate[neighborR] != plateID {
				continue
			}
			visited[neighborR] = true
			queue = append(queue, queueItem{region: neighborR})
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
