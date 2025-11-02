package tectonics

import (
	"math"
	"math/rand"
)

// Method5CellularAutomata implements iterative smoothing with growth constraints
// Expected score: 0.55-0.70
func Method5CellularAutomata(
	voronoiCells []VoronoiCell,
	icosphereSites []Vector3D,
	planetRadius float64,
	settings Method5Settings,
) ([]TectonicPlate, []int, error) {

	rng := rand.New(rand.NewSource(settings.Seed))
	sphereArea := 4.0 * math.Pi * planetRadius * planetRadius
	cellArea := sphereArea / float64(len(voronoiCells))

	// Step 1: Random initial assignment to target number of plates
	cellAssignments := make([]int, len(voronoiCells))
	for i := range cellAssignments {
		cellAssignments[i] = rng.Intn(settings.TargetPlateCount)
	}

	// Step 2: Assign power-law size budgets to each plate
	// Using rank-size distribution
	plateBudgets := make([]float64, settings.TargetPlateCount)
	totalBudget := 0.0

	for i := 0; i < settings.TargetPlateCount; i++ {
		rank := float64(i + 1)
		budget := math.Pow(rank, -settings.PowerLawExponent)
		plateBudgets[i] = budget
		totalBudget += budget
	}

	// Normalize budgets to sum to 1.0
	for i := range plateBudgets {
		plateBudgets[i] /= totalBudget
	}

	// Shuffle to randomize which plates get which budgets
	for i := range plateBudgets {
		j := rng.Intn(i + 1)
		plateBudgets[i], plateBudgets[j] = plateBudgets[j], plateBudgets[i]
	}

	// Step 3: Cellular automata iteration
	// Each cell votes to join majority of neighbors, but plates can't exceed budget
	for iteration := 0; iteration < settings.Iterations; iteration++ {
		newAssignments := make([]int, len(cellAssignments))
		copy(newAssignments, cellAssignments)

		// Calculate current plate sizes
		plateSizes := make([]float64, settings.TargetPlateCount)
		for cellIdx, plateIdx := range cellAssignments {
			if cellIdx < len(voronoiCells) {
				plateSizes[plateIdx] += cellArea
			}
		}

		// For each cell, consider switching to neighbor majority
		processOrder := rng.Perm(len(voronoiCells))

		for _, cellIdx := range processOrder {
			if cellIdx >= len(voronoiCells) || cellIdx >= len(icosphereSites) {
				continue
			}

			currentPlate := cellAssignments[cellIdx]
			thisCellArea := cellArea

			// Count neighbor votes
			votes := make(map[int]int)
			for _, neighborIdx := range voronoiCells[cellIdx].NeighborSiteIndices {
				if int(neighborIdx) < len(cellAssignments) {
					neighborPlate := cellAssignments[neighborIdx]
					votes[neighborPlate]++
				}
			}

			// Find majority vote
			bestPlate := currentPlate
			bestVotes := votes[currentPlate]

			for plateIdx, voteCount := range votes {
				if voteCount > bestVotes {
					// Check if target plate has budget available
					targetSize := plateSizes[plateIdx]
					if plateIdx != currentPlate {
						targetSize += cellArea
					}
					maxSize := plateBudgets[plateIdx] * sphereArea * settings.BudgetMultiplier

					if targetSize <= maxSize {
						bestPlate = plateIdx
						bestVotes = voteCount
					}
				}
			}

			// Update assignment if changed
			if bestPlate != currentPlate {
				newAssignments[cellIdx] = bestPlate

				// Update running sizes for budget check
				plateSizes[currentPlate] -= thisCellArea
				plateSizes[bestPlate] += thisCellArea
			}
		}

		cellAssignments = newAssignments

		// Check for convergence every 10 iterations
		if iteration%10 == 0 && iteration > 0 {
			// Count changes
			changes := 0
			for i := range cellAssignments {
				if cellAssignments[i] != newAssignments[i] {
					changes++
				}
			}

			if changes < len(cellAssignments)/100 {
				// Less than 1% changed, consider converged
				break
			}
		}
	}

	// Step 4: Merge any plates that ended up too small
	cellAssignments = mergeSmallPlates(
		cellAssignments,
		voronoiCells,
		sphereArea,
		0.0001,  // Min 0.01% of sphere
	)

	// Renumber to make contiguous
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

// Method5Settings controls cellular automata generation
type Method5Settings struct {
	TargetPlateCount   int
	PowerLawExponent   float64  // For size budget distribution
	BudgetMultiplier   float64  // How strict the budget constraint (1.0 = strict, 2.0 = loose)
	Iterations         int      // Number of CA iterations
	Seed               int64
}

// DefaultMethod5Settings returns reasonable defaults
func DefaultMethod5Settings() Method5Settings {
	return Method5Settings{
		TargetPlateCount: 39,
		PowerLawExponent: 0.39,   // Earth-like distribution
		BudgetMultiplier: 1.5,    // Allow some flexibility
		Iterations:       200,
		Seed:             12345,
	}
}

// mergeSmallPlates merges plates below minimum size into neighbors
func mergeSmallPlates(
	assignments []int,
	voronoiCells []VoronoiCell,
	sphereArea float64,
	minPercent float64,
) []int {

	cellArea := sphereArea / float64(len(voronoiCells))

	for {
		// Calculate plate sizes
		plateSizes := make(map[int]float64)
		for cellIdx, plateIdx := range assignments {
			if cellIdx < len(voronoiCells) {
				plateSizes[plateIdx] += cellArea
			}
		}

		// Find smallest plate below threshold
		smallestPlate := -1
		smallestSize := math.Inf(1)

		for plateIdx, size := range plateSizes {
			percent := size / sphereArea
			if percent < minPercent && size < smallestSize {
				smallestSize = size
				smallestPlate = plateIdx
			}
		}

		if smallestPlate < 0 {
			break  // No more small plates
		}

		// Find cells of this plate
		plateCells := make([]int, 0)
		for cellIdx, plateIdx := range assignments {
			if plateIdx == smallestPlate {
				plateCells = append(plateCells, cellIdx)
			}
		}

		// Find any neighbor
		targetPlate := -1
		for _, cellIdx := range plateCells {
			if cellIdx >= len(voronoiCells) {
				continue
			}
			for _, neighborIdx := range voronoiCells[cellIdx].NeighborSiteIndices {
				if int(neighborIdx) < len(assignments) {
					neighborPlate := assignments[neighborIdx]
					if neighborPlate != smallestPlate {
						targetPlate = neighborPlate
						break
					}
				}
			}
			if targetPlate >= 0 {
				break
			}
		}

		if targetPlate < 0 {
			break  // Isolated plate
		}

		// Merge
		for i, plateIdx := range assignments {
			if plateIdx == smallestPlate {
				assignments[i] = targetPlate
			}
		}
	}

	return assignments
}
