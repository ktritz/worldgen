package tectonics

import (
	"math"
	"math/rand"
)

// Method1HierarchicalVoronoi implements weighted Voronoi with power-law distributed weights
// Expected score: 0.50-0.65
func Method1HierarchicalVoronoi(
	voronoiCells []VoronoiCell,
	icosphereSites []Vector3D,
	planetRadius float64,
	settings Method1Settings,
) ([]TectonicPlate, []int, error) {

	rng := rand.New(rand.NewSource(settings.Seed))

	// Step 1: Generate plate seed centroids with power-law spacing
	numPlates := settings.TargetPlateCount
	seeds := make([]int, numPlates)

	// Use Poisson disk sampling for well-separated seeds
	seeds = poissonDiskSampling(icosphereSites, numPlates, planetRadius*0.3, rng)

	// Step 2: Assign hierarchical weights to create 7 major, 13 minor, 19 micro plates
	// Use explicit stratified weights instead of power-law
	weights := make([]float64, numPlates)

	// Major plates (first 7 seeds): very high weights
	for i := 0; i < 7 && i < numPlates; i++ {
		weights[i] = 10.0  // 10x multiplier for major plates
	}

	// Minor plates (next 13 seeds): medium weights
	for i := 7; i < 20 && i < numPlates; i++ {
		weights[i] = 1.0   // 1x multiplier for minor plates
	}

	// Micro plates (remaining seeds): low weights
	for i := 20; i < numPlates; i++ {
		weights[i] = 0.1   // 0.1x multiplier for micro plates
	}

	// Apply additional power-law variation within each tier
	for i := 0; i < numPlates; i++ {
		tierRank := float64((i % 7) + 1)  // Rank within tier
		weights[i] *= math.Pow(tierRank, -settings.PowerLawExponent)
	}

	// Step 3: Weighted Voronoi partition
	// Each cell assigned to nearest seed, but distance weighted by seed strength
	cellAssignments := make([]int, len(voronoiCells))

	for cellIdx := range voronoiCells {
		if cellIdx >= len(icosphereSites) {
			continue
		}

		bestPlate := 0
		bestScore := math.Inf(1)

		for plateIdx, seedIdx := range seeds {
			if seedIdx >= len(icosphereSites) {
				continue
			}

			// Calculate distance
			dist := CalculateSphericalDistance(
				icosphereSites[cellIdx],
				icosphereSites[seedIdx],
				planetRadius,
			)

			// Weight by inverse of plate weight (stronger plates "attract" more)
			// Use squared weight for more extreme size differences
			// Lower score = more likely to assign
			effectiveWeight := weights[plateIdx] * weights[plateIdx]
			score := dist / (effectiveWeight + 0.01)

			if score < bestScore {
				bestScore = score
				bestPlate = plateIdx
			}
		}

		cellAssignments[cellIdx] = bestPlate
	}

	// Step 4: Create plate structures
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

// Method1Settings controls hierarchical Voronoi generation
type Method1Settings struct {
	TargetPlateCount   int
	PowerLawExponent   float64
	Seed               int64
}

// DefaultMethod1Settings returns reasonable defaults
func DefaultMethod1Settings() Method1Settings {
	return Method1Settings{
		TargetPlateCount: 39,
		PowerLawExponent: 0.39,
		Seed:             12345,
	}
}

// poissonDiskSampling generates well-separated points on sphere
func poissonDiskSampling(
	sites []Vector3D,
	numSamples int,
	minDistance float64,
	rng *rand.Rand,
) []int {
	if numSamples > len(sites) {
		numSamples = len(sites)
	}

	samples := make([]int, 0, numSamples)

	// Start with random point
	first := rng.Intn(len(sites))
	samples = append(samples, first)

	// Try to add more points
	maxAttempts := numSamples * 100
	attempts := 0

	for len(samples) < numSamples && attempts < maxAttempts {
		attempts++

		// Pick random candidate
		candidate := rng.Intn(len(sites))

		// Check if far enough from all existing samples
		valid := true
		for _, existing := range samples {
			dist := vectorDistance(sites[candidate], sites[existing])
			if dist < minDistance {
				valid = false
				break
			}
		}

		if valid {
			samples = append(samples, candidate)
		}
	}

	// If we didn't get enough, just add random ones
	for len(samples) < numSamples {
		candidate := rng.Intn(len(sites))
		// Check not duplicate
		duplicate := false
		for _, s := range samples {
			if s == candidate {
				duplicate = true
				break
			}
		}
		if !duplicate {
			samples = append(samples, candidate)
		}
	}

	return samples
}

func vectorDistance(a, b Vector3D) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}
