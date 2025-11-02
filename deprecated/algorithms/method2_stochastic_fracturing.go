package tectonics

import (
	"math"
	"math/rand"
)

// Method2StochasticFracturing implements top-down recursive plate splitting
// Expected score: 0.55-0.70
func Method2StochasticFracturing(
	voronoiCells []VoronoiCell,
	icosphereSites []Vector3D,
	planetRadius float64,
	settings Method2Settings,
) ([]TectonicPlate, []int, error) {

	rng := rand.New(rand.NewSource(settings.Seed))

	// Start with one plate covering entire sphere
	cellAssignments := make([]int, len(voronoiCells))
	for i := range cellAssignments {
		cellAssignments[i] = 0
	}

	plateCount := 1
	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius
	cellArea := sphereArea / float64(len(voronoiCells))

	// Recursively split plates until we have enough
	for plateCount < settings.TargetPlateCount {
		// Calculate current plate sizes
		plateSizes := make(map[int]float64)
		for cellIdx, plateIdx := range cellAssignments {
			if cellIdx < len(voronoiCells) {
				plateSizes[plateIdx] += cellArea
			}
		}

		// Select plate to split (bias toward larger plates)
		plateToSplit := selectPlateToSplit(plateSizes, sphereArea, settings.PowerLawExponent, rng)
		if plateToSplit < 0 {
			break
		}

		// Split the selected plate
		cellAssignments = splitPlate(
			plateToSplit,
			plateCount,
			cellAssignments,
			voronoiCells,
			icosphereSites,
			planetRadius,
			rng,
		)

		plateCount++
	}

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

// Method2Settings controls stochastic fracturing
type Method2Settings struct {
	TargetPlateCount int
	PowerLawExponent float64  // Controls split probability (higher = prefer large plates)
	Seed             int64
}

// DefaultMethod2Settings returns reasonable defaults
func DefaultMethod2Settings() Method2Settings {
	return Method2Settings{
		TargetPlateCount: 39,
		PowerLawExponent: 1.5,  // Strongly prefer splitting large plates
		Seed:             12345,
	}
}

// selectPlateToSplit chooses which plate to split based on size
func selectPlateToSplit(
	plateSizes map[int]float64,
	sphereArea float64,
	exponent float64,
	rng *rand.Rand,
) int {
	if len(plateSizes) == 0 {
		return -1
	}

	// Calculate selection probabilities (proportional to size^exponent)
	type plateProbability struct {
		plateIdx int
		prob     float64
	}

	probs := make([]plateProbability, 0, len(plateSizes))
	totalProb := 0.0

	for plateIdx, size := range plateSizes {
		// Only split plates larger than 0.02% of sphere (to allow microplate creation)
		// This is slightly above the 0.01% microplate threshold
		if size/sphereArea > 0.0002 {
			prob := math.Pow(size, exponent)
			probs = append(probs, plateProbability{plateIdx, prob})
			totalProb += prob
		}
	}

	if len(probs) == 0 {
		return -1
	}

	// Weighted random selection
	threshold := rng.Float64() * totalProb
	cumulative := 0.0
	for _, p := range probs {
		cumulative += p.prob
		if cumulative >= threshold {
			return p.plateIdx
		}
	}

	return probs[len(probs)-1].plateIdx
}

// splitPlate divides a plate into two parts along a fracture line
func splitPlate(
	plateToSplit int,
	newPlateIdx int,
	cellAssignments []int,
	voronoiCells []VoronoiCell,
	icosphereSites []Vector3D,
	planetRadius float64,
	rng *rand.Rand,
) []int {

	// Find all cells in this plate
	plateCells := make([]int, 0)
	for cellIdx, plateIdx := range cellAssignments {
		if plateIdx == plateToSplit {
			plateCells = append(plateCells, cellIdx)
		}
	}

	if len(plateCells) < 2 {
		return cellAssignments
	}

	// Calculate plate centroid
	centerX, centerY, centerZ := 0.0, 0.0, 0.0
	for _, cellIdx := range plateCells {
		if cellIdx < len(icosphereSites) {
			centerX += icosphereSites[cellIdx].X
			centerY += icosphereSites[cellIdx].Y
			centerZ += icosphereSites[cellIdx].Z
		}
	}
	n := float64(len(plateCells))
	centerX /= n
	centerY /= n
	centerZ /= n

	// Normalize center to sphere surface
	length := math.Sqrt(centerX*centerX + centerY*centerY + centerZ*centerZ)
	if length > 0 {
		centerX /= length
		centerY /= length
		centerZ /= length
	}

	// Generate random fracture direction (great circle normal)
	// For a great circle on a sphere, we need a plane through the origin
	// The normal vector to this plane defines the great circle
	theta := rng.Float64() * 2.0 * math.Pi
	phi := math.Acos(2.0*rng.Float64() - 1.0)

	normalX := math.Sin(phi) * math.Cos(theta)
	normalY := math.Sin(phi) * math.Sin(theta)
	normalZ := math.Cos(phi)

	// To ensure roughly even split, rotate the normal to be perpendicular to the centroid
	// This makes the great circle pass through or near the plate
	// Cross product of centroid with a random vector gives perpendicular
	tempX, tempY, tempZ := normalX, normalY, normalZ
	normalX = centerY*tempZ - centerZ*tempY
	normalY = centerZ*tempX - centerX*tempZ
	normalZ = centerX*tempY - centerY*tempX

	// Normalize
	normLength := math.Sqrt(normalX*normalX + normalY*normalY + normalZ*normalZ)
	if normLength > 0 {
		normalX /= normLength
		normalY /= normLength
		normalZ /= normLength
	}

	// Split cells based on which side of fracture plane they're on
	newAssignments := make([]int, len(cellAssignments))
	copy(newAssignments, cellAssignments)

	for _, cellIdx := range plateCells {
		if cellIdx >= len(icosphereSites) {
			continue
		}

		// For a great circle split, check which side of the plane through origin
		// The cell position vector dotted with the normal tells us which side
		dot := icosphereSites[cellIdx].X*normalX +
		       icosphereSites[cellIdx].Y*normalY +
		       icosphereSites[cellIdx].Z*normalZ

		if dot > 0 {
			newAssignments[cellIdx] = newPlateIdx
		}
		// else stays in original plate
	}

	return newAssignments
}
