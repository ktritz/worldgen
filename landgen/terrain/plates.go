package terrain

// Plate generation and assignment for tectonic simulation
// Uses weighted BFS for power-law size distribution

import (
	"math"
	"math/rand"
)

// GeneratePlates creates plates using weighted BFS for power law size distribution
// Returns: plateR (list of plate center region indices), rPlate (region -> plate center mapping)
func GeneratePlates(sites []Vector3D, cells []VoronoiCell, numPlates int, rng *rand.Rand) ([]int, []int) {
	numRegions := len(sites)

	// Pick random regions as plate centers
	plateR := pickRandomRegions(numRegions, numPlates, rng)

	// Assign power law growth weights to each plate
	// Higher weight = grows faster = ends up larger
	plateWeight := make(map[int]float64)
	exponent := 1.5 // Controls how skewed the distribution is
	for _, centerR := range plateR {
		// Power law: weight = (1/uniform_random)^exponent
		// This gives few large values, many small values
		u := rng.Float64()*0.9 + 0.1 // Avoid 0, range [0.1, 1.0]
		plateWeight[centerR] = math.Pow(1.0/u, exponent)
	}

	// Initialize r_plate: maps each region to its plate's center region
	rPlate := make([]int, numRegions)
	for i := range rPlate {
		rPlate[i] = -1
	}

	// Initialize queue with plate centers
	type queueItem struct {
		region int
		weight float64
	}
	var queue []queueItem
	for _, r := range plateR {
		rPlate[r] = r
		queue = append(queue, queueItem{r, plateWeight[r]})
	}

	// Weighted BFS - higher weight plates expand more often
	for len(queue) > 0 {
		// Weighted random selection
		totalWeight := 0.0
		for _, item := range queue {
			totalWeight += item.weight
		}

		target := rng.Float64() * totalWeight
		cumulative := 0.0
		selectedIdx := 0
		for i, item := range queue {
			cumulative += item.weight
			if cumulative >= target {
				selectedIdx = i
				break
			}
		}

		// Remove selected item
		selected := queue[selectedIdx]
		queue[selectedIdx] = queue[len(queue)-1]
		queue = queue[:len(queue)-1]

		// Expand to unassigned neighbors
		plateCenter := rPlate[selected.region]
		weight := plateWeight[plateCenter]
		for _, neighborIdx := range cells[selected.region].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR < numRegions && rPlate[neighborR] == -1 {
				rPlate[neighborR] = plateCenter
				queue = append(queue, queueItem{neighborR, weight})
			}
		}
	}

	return plateR, rPlate
}

// SmoothPlateBoundaries eliminates single-cell plate protrusions
// This creates smoother plate boundaries while preserving coastline irregularity
// (coastlines depend on elevation, not plate assignment)
// Uses majority-neighbor voting: if >50% of neighbors are a different plate, reassign
func SmoothPlateBoundaries(cells []VoronoiCell, rPlate []int, iterations int) int {
	numRegions := len(rPlate)
	totalChanges := 0

	for iter := 0; iter < iterations; iter++ {
		changes := 0
		for r := 0; r < numRegions; r++ {
			currentPlate := rPlate[r]

			// Count neighbors by plate
			neighborCounts := make(map[int]int)
			totalNeighbors := 0
			for _, neighborIdx := range cells[r].NeighborSiteIndices {
				nIdx := int(neighborIdx)
				if nIdx < numRegions {
					neighborCounts[rPlate[nIdx]]++
					totalNeighbors++
				}
			}

			// Find if any other plate has majority of neighbors
			for plate, count := range neighborCounts {
				// Reassign if >50% neighbors are a different plate
				if plate != currentPlate && count*2 > totalNeighbors {
					rPlate[r] = plate
					changes++
					break
				}
			}
		}

		totalChanges += changes
		if changes == 0 {
			break // Converged
		}
	}

	return totalChanges
}

// AssignPlateTypes assigns plates as oceanic or continental
// Uses BFS growth from 2-3 seeds to create multiple continents
func AssignPlateTypes(
	sortedPlates []PlateSize,
	plateSizes map[int]int,
	plateNeighbors map[int]map[int]bool,
	totalRegions int,
	targetLandFraction float64,
) map[int]bool {
	plateIsOcean := make(map[int]bool)

	// Start with all plates as oceanic
	for _, ps := range sortedPlates {
		plateIsOcean[ps.Center] = true
	}

	// Target continental fraction based on desired land coverage
	// Continental shelf floods ~30% of continental area, so target = landFraction / 0.7
	targetContinentalFraction := targetLandFraction / 0.7
	if targetContinentalFraction > 0.6 {
		targetContinentalFraction = 0.6 // Cap at 60%
	}
	targetContinentalRegions := int(float64(totalRegions) * targetContinentalFraction)

	// Pick 2-3 separate continental seeds to create multiple continents
	// Largest plate stays oceanic (Pacific), pick spread-out plates as seeds
	continentalRegions := 0
	expanded := make(map[int]bool)

	// Select seeds: 2nd, 4th, and 6th largest plates (spread out by size)
	seedIndices := []int{1, 3, 5}
	var seeds []int
	for _, idx := range seedIndices {
		if idx < len(sortedPlates) {
			seedPlate := sortedPlates[idx].Center
			if !expanded[seedPlate] {
				plateIsOcean[seedPlate] = false
				continentalRegions += plateSizes[seedPlate]
				seeds = append(seeds, seedPlate)
				expanded[seedPlate] = true
			}
		}
	}

	// BFS grow from all seeds simultaneously until we hit target
	queue := make([]int, len(seeds))
	copy(queue, seeds)

	for len(queue) > 0 && continentalRegions < targetContinentalRegions {
		current := queue[0]
		queue = queue[1:]

		for neighborPlate := range plateNeighbors[current] {
			if expanded[neighborPlate] {
				continue
			}
			expanded[neighborPlate] = true

			if plateIsOcean[neighborPlate] {
				plateIsOcean[neighborPlate] = false
				continentalRegions += plateSizes[neighborPlate]
				queue = append(queue, neighborPlate)

				if continentalRegions >= targetContinentalRegions {
					break
				}
			}
		}
	}

	return plateIsOcean
}

// PlateSize holds plate center and size for sorting
type PlateSize struct {
	Center int
	Size   int
}

// FindConnectedOceanGroups finds groups of oceanic plates that are connected to each other
func FindConnectedOceanGroups(plateR []int, plateIsOcean map[int]bool, plateNeighbors map[int]map[int]bool) map[int][]int {
	visited := make(map[int]bool)
	groups := make(map[int][]int)
	groupID := 0

	for _, centerR := range plateR {
		if !plateIsOcean[centerR] || visited[centerR] {
			continue
		}

		// BFS to find all connected oceanic plates
		var group []int
		queue := []int{centerR}
		visited[centerR] = true

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			group = append(group, current)

			for neighbor := range plateNeighbors[current] {
				if plateIsOcean[neighbor] && !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}

		groups[groupID] = group
		groupID++
	}

	return groups
}

// FindPlateNeighbors returns a map of plate -> set of neighboring plates
func FindPlateNeighbors(cells []VoronoiCell, rPlate []int, plateR []int) map[int]map[int]bool {
	numRegions := len(cells)
	neighbors := make(map[int]map[int]bool)

	// Initialize empty neighbor sets for each plate
	for _, centerR := range plateR {
		neighbors[centerR] = make(map[int]bool)
	}

	// Find all plate boundaries
	for r := 0; r < numRegions; r++ {
		myPlate := rPlate[r]
		for _, neighborIdx := range cells[r].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR < numRegions {
				neighborPlate := rPlate[neighborR]
				if myPlate != neighborPlate {
					neighbors[myPlate][neighborPlate] = true
				}
			}
		}
	}

	return neighbors
}

// pickRandomRegions selects n random distinct regions
func pickRandomRegions(numRegions, n int, rng *rand.Rand) []int {
	chosen := make(map[int]bool)
	var result []int
	for len(result) < n && len(result) < numRegions {
		r := rng.Intn(numRegions)
		if !chosen[r] {
			chosen[r] = true
			result = append(result, r)
		}
	}
	return result
}

// AssignPlateRotations assigns Euler pole rotations to plates for realistic curved motion.
// - Oceanic plates rotate toward continental neighbors (creates subduction)
// - Continental plates have some bias toward each other (creates mountain ranges)
// - Rotation around Euler poles creates curved velocity fields and realistic hotspot tracks
func AssignPlateRotations(
	sites []Vector3D,
	cells []VoronoiCell,
	plateR []int,
	plateIsOcean map[int]bool,
	plateNeighbors map[int]map[int]bool,
	rng *rand.Rand,
) map[int]PlateRotation {
	plateRot := make(map[int]PlateRotation)

	for _, centerR := range plateR {
		isOcean := plateIsOcean[centerR]
		neighbors := plateNeighbors[centerR]
		centerPos := sites[centerR]

		var targetDir Vector3D
		hasTarget := false

		if isOcean {
			// Oceanic plate: find continental neighbors and move toward them
			for neighborPlate := range neighbors {
				if !plateIsOcean[neighborPlate] {
					neighborPos := sites[neighborPlate]
					targetDir.X += neighborPos.X - centerPos.X
					targetDir.Y += neighborPos.Y - centerPos.Y
					targetDir.Z += neighborPos.Z - centerPos.Z
					hasTarget = true
				}
			}
		} else {
			// Continental plate: 50% chance to move toward another continental neighbor
			if rng.Float64() < 0.5 {
				for neighborPlate := range neighbors {
					if !plateIsOcean[neighborPlate] {
						neighborPos := sites[neighborPlate]
						targetDir.X += neighborPos.X - centerPos.X
						targetDir.Y += neighborPos.Y - centerPos.Y
						targetDir.Z += neighborPos.Z - centerPos.Z
						hasTarget = true
					}
				}
			}
		}

		var pole Vector3D
		// Angular velocity with wide variation (like real plates: 1-10 cm/year)
		// Use log-uniform distribution for natural spread: 0.1 to 1.0 (10x range)
		angularVel := 0.1 * math.Pow(10, rng.Float64()) // Range: 0.1 to 1.0

		if hasTarget {
			// Project target direction onto tangent plane at centerPos
			dot := centerPos.X*targetDir.X + centerPos.Y*targetDir.Y + centerPos.Z*targetDir.Z
			tangent := Vector3D{
				X: targetDir.X - dot*centerPos.X,
				Y: targetDir.Y - dot*centerPos.Y,
				Z: targetDir.Z - dot*centerPos.Z,
			}

			// Normalize tangent
			mag := math.Sqrt(tangent.X*tangent.X + tangent.Y*tangent.Y + tangent.Z*tangent.Z)
			if mag > 1e-6 {
				tangent.X /= mag
				tangent.Y /= mag
				tangent.Z /= mag

				// Euler pole = centerPos × tangent (perpendicular to both)
				// This creates rotation that moves centerPos in the tangent direction
				pole = Vector3D{
					X: centerPos.Y*tangent.Z - centerPos.Z*tangent.Y,
					Y: centerPos.Z*tangent.X - centerPos.X*tangent.Z,
					Z: centerPos.X*tangent.Y - centerPos.Y*tangent.X,
				}

				// Normalize pole
				poleMag := math.Sqrt(pole.X*pole.X + pole.Y*pole.Y + pole.Z*pole.Z)
				if poleMag > 1e-6 {
					pole.X /= poleMag
					pole.Y /= poleMag
					pole.Z /= poleMag
				}

				// Add significant jitter to pole position for variety in chain curvature
				// Larger jitter = more curved hotspot chains (pole closer to/further from 90°)
				jitter := 0.6
				pole.X += (rng.Float64() - 0.5) * jitter
				pole.Y += (rng.Float64() - 0.5) * jitter
				pole.Z += (rng.Float64() - 0.5) * jitter

				// Renormalize
				poleMag = math.Sqrt(pole.X*pole.X + pole.Y*pole.Y + pole.Z*pole.Z)
				if poleMag > 0 {
					pole.X /= poleMag
					pole.Y /= poleMag
					pole.Z /= poleMag
				}
			} else {
				// Fallback to random pole
				pole = randomUnitVector(rng)
			}
		} else {
			// Random Euler pole for plates without clear targets
			pole = randomUnitVector(rng)
			// Random sign for rotation direction
			if rng.Float64() < 0.5 {
				angularVel = -angularVel
			}
		}

		plateRot[centerR] = PlateRotation{
			Pole:            pole,
			AngularVelocity: angularVel,
		}
	}

	return plateRot
}

// randomUnitVector generates a uniformly random unit vector
func randomUnitVector(rng *rand.Rand) Vector3D {
	z := 2*rng.Float64() - 1
	theta := 2 * math.Pi * rng.Float64()
	r := math.Sqrt(1 - z*z)
	return Vector3D{
		X: r * math.Cos(theta),
		Y: r * math.Sin(theta),
		Z: z,
	}
}

