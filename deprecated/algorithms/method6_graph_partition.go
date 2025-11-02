package tectonics

import (
	"math"
	"math/rand"
)

// Method6GraphPartition implements balanced graph partitioning with power-law size targets
// Expected score: 0.65-0.80
func Method6GraphPartition(
	voronoiCells []VoronoiCell,
	icosphereSites []Vector3D,
	planetRadius float64,
	settings Method6Settings,
) ([]TectonicPlate, []int, error) {

	rng := rand.New(rand.NewSource(settings.Seed))
	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius
	cellArea := sphereArea / float64(len(voronoiCells))

	// Step 1: Create size targets using power-law distribution
	sizeTargets := make([]float64, settings.TargetPlateCount)
	totalTarget := 0.0

	for i := 0; i < settings.TargetPlateCount; i++ {
		rank := float64(i + 1)
		target := math.Pow(rank, -settings.PowerLawExponent)
		sizeTargets[i] = target
		totalTarget += target
	}

	// Normalize to sum to sphere area
	for i := range sizeTargets {
		sizeTargets[i] = (sizeTargets[i] / totalTarget) * sphereArea
	}

	// Step 2: Initialize with k-means++ style seed selection
	seeds := make([]int, settings.TargetPlateCount)

	// First seed is random
	seeds[0] = rng.Intn(len(icosphereSites))

	// Subsequent seeds maximize distance from existing seeds
	for i := 1; i < settings.TargetPlateCount; i++ {
		bestCandidate := 0
		bestMinDist := 0.0

		// Try multiple candidates
		for attempt := 0; attempt < 50; attempt++ {
			candidate := rng.Intn(len(icosphereSites))

			// Find minimum distance to existing seeds
			minDist := math.Inf(1)
			for j := 0; j < i; j++ {
				dist := CalculateSphericalDistance(
					icosphereSites[candidate],
					icosphereSites[seeds[j]],
					planetRadius,
				)
				if dist < minDist {
					minDist = dist
				}
			}

			// Keep candidate with maximum minimum distance
			if minDist > bestMinDist {
				bestMinDist = minDist
				bestCandidate = candidate
			}
		}

		seeds[i] = bestCandidate
	}

	// Step 3: Initial assignment based on closest seed
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

	// Step 4: Iterative refinement with size balancing
	// Similar to k-means but with size constraints
	for iteration := 0; iteration < settings.RefinementIterations; iteration++ {
		// Calculate current plate sizes
		plateSizes := make([]float64, settings.TargetPlateCount)
		for cellIdx, plateIdx := range cellAssignments {
			if cellIdx < len(voronoiCells) {
				plateSizes[plateIdx] += cellArea
			}
		}

		// Process boundary cells in random order
		processOrder := rng.Perm(len(voronoiCells))
		changed := 0

		for _, cellIdx := range processOrder {
			if cellIdx >= len(voronoiCells) || cellIdx >= len(icosphereSites) {
				continue
			}

			currentPlate := cellAssignments[cellIdx]
			thisCellArea := cellArea

			// Check if this is a boundary cell
			hasDifferentNeighbor := false
			for _, neighborIdx := range voronoiCells[cellIdx].NeighborSiteIndices {
				if int(neighborIdx) < len(cellAssignments) {
					if cellAssignments[neighborIdx] != currentPlate {
						hasDifferentNeighbor = true
						break
					}
				}
			}

			if !hasDifferentNeighbor {
				continue  // Interior cell, skip
			}

			// Try reassigning to each neighbor plate
			bestPlate := currentPlate
			bestScore := calculatePartitionScore(
				cellIdx,
				currentPlate,
				cellAssignments,
				voronoiCells,
				icosphereSites,
				plateSizes,
				sizeTargets,
				planetRadius,
			)

			for _, neighborIdx := range voronoiCells[cellIdx].NeighborSiteIndices {
				if int(neighborIdx) >= len(cellAssignments) {
					continue
				}

				neighborPlate := cellAssignments[neighborIdx]
				if neighborPlate == currentPlate {
					continue
				}

				// Calculate score if we move this cell
				score := calculatePartitionScore(
					cellIdx,
					neighborPlate,
					cellAssignments,
					voronoiCells,
					icosphereSites,
					plateSizes,
					sizeTargets,
					planetRadius,
				)

				if score < bestScore {
					bestScore = score
					bestPlate = neighborPlate
				}
			}

			// Move if better
			if bestPlate != currentPlate {
				cellAssignments[cellIdx] = bestPlate
				plateSizes[currentPlate] -= thisCellArea
				plateSizes[bestPlate] += thisCellArea
				changed++
			}
		}

		// Early stopping if converged
		if changed < len(voronoiCells)/200 {
			break
		}
	}

	// Create plate structures
	plates := createPlatesFromAssignments(
		voronoiCells,
		icosphereSites,
		cellAssignments,
		seeds,
		planetRadius,
		rng,
	)

	return plates, cellAssignments, nil
}

// calculatePartitionScore evaluates how good it would be to assign cellIdx to plateIdx
// Lower score is better
func calculatePartitionScore(
	cellIdx int,
	plateIdx int,
	assignments []int,
	voronoiCells []VoronoiCell,
	icosphereSites []Vector3D,
	plateSizes []float64,
	sizeTargets []float64,
	planetRadius float64,
) float64 {

	// Component 1: Size deviation from target
	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius
	cellArea := sphereArea / float64(len(voronoiCells))
	newSize := plateSizes[plateIdx]
	if assignments[cellIdx] != plateIdx {
		newSize += cellArea
	}

	targetSize := sizeTargets[plateIdx]
	sizeDeviation := math.Abs(newSize - targetSize) / targetSize

	// Component 2: Compactness (distance from plate's current centroid)
	// Calculate approximate centroid of plate
	centerX, centerY, centerZ := 0.0, 0.0, 0.0
	count := 0

	for i, p := range assignments {
		if p == plateIdx && i < len(icosphereSites) {
			centerX += icosphereSites[i].X
			centerY += icosphereSites[i].Y
			centerZ += icosphereSites[i].Z
			count++
		}
	}

	if count > 0 {
		centerX /= float64(count)
		centerY /= float64(count)
		centerZ /= float64(count)

		// Normalize to sphere
		length := math.Sqrt(centerX*centerX + centerY*centerY + centerZ*centerZ)
		if length > 0 {
			centerX /= length
			centerY /= length
			centerZ /= length
		}

		// Distance from cell to centroid
		dist := CalculateSphericalDistance(
			icosphereSites[cellIdx],
			Vector3D{centerX * planetRadius, centerY * planetRadius, centerZ * planetRadius},
			planetRadius,
		)

		compactness := dist / (planetRadius * math.Pi)  // Normalize by max possible distance

		// Combined score: balance size fit and compactness
		return sizeDeviation + 0.5*compactness
	}

	return sizeDeviation
}

// Method6Settings controls graph partition generation
type Method6Settings struct {
	TargetPlateCount       int
	PowerLawExponent       float64
	RefinementIterations   int
	Seed                   int64
}

// DefaultMethod6Settings returns reasonable defaults
func DefaultMethod6Settings() Method6Settings {
	return Method6Settings{
		TargetPlateCount:     39,
		PowerLawExponent:     0.39,   // Earth-like power law
		RefinementIterations: 100,
		Seed:                 12345,
	}
}
