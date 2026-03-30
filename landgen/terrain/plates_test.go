package terrain

import (
	"math/rand"
	"testing"

	"worldgen/icosphere"
)

func TestAssignPlateTypesKeepsDominantLargestPlateOceanic(t *testing.T) {
	sortedPlates := []PlateSize{
		{Center: 0, Size: 420},
		{Center: 1, Size: 140},
		{Center: 2, Size: 110},
		{Center: 3, Size: 90},
		{Center: 4, Size: 70},
		{Center: 5, Size: 50},
		{Center: 6, Size: 40},
		{Center: 7, Size: 30},
		{Center: 8, Size: 20},
		{Center: 9, Size: 12},
		{Center: 10, Size: 10},
		{Center: 11, Size: 8},
	}
	plateSizes := make(map[int]int, len(sortedPlates))
	for _, ps := range sortedPlates {
		plateSizes[ps.Center] = ps.Size
	}

	plateNeighbors := ringPlateNeighbors(len(sortedPlates))
	plateIsOcean := AssignPlateTypes(sortedPlates, plateSizes, plateNeighbors, 1000, 0.29)

	if !plateIsOcean[0] {
		t.Fatal("expected largest plate to remain oceanic")
	}

	mask := assignmentMask(sortedPlates, plateIsOcean)
	stats := evaluatePlateAssignment(mask, sortedPlates, plateSizes, plateNeighbors, 1000)
	fraction := float64(stats.continentalRegions) / 1000.0
	if fraction < 0.24 || fraction > 0.48 {
		t.Fatalf("continental fraction = %.3f, want reasonable range", fraction)
	}
	if stats.largestContinentShare > 0.62 {
		t.Fatalf("largest continent share = %.3f, want <= 0.62", stats.largestContinentShare)
	}
}

func TestAssignPlateTypesProducesMultipleContinents(t *testing.T) {
	sortedPlates := []PlateSize{
		{Center: 0, Size: 190},
		{Center: 1, Size: 150},
		{Center: 2, Size: 130},
		{Center: 3, Size: 120},
		{Center: 4, Size: 110},
		{Center: 5, Size: 90},
		{Center: 6, Size: 70},
		{Center: 7, Size: 55},
		{Center: 8, Size: 35},
		{Center: 9, Size: 20},
		{Center: 10, Size: 18},
		{Center: 11, Size: 12},
	}
	plateSizes := make(map[int]int, len(sortedPlates))
	for _, ps := range sortedPlates {
		plateSizes[ps.Center] = ps.Size
	}

	plateNeighbors := map[int]map[int]bool{
		0:  {1: true, 4: true, 8: true},
		1:  {0: true, 2: true, 5: true},
		2:  {1: true, 3: true, 6: true},
		3:  {2: true, 4: true, 7: true},
		4:  {0: true, 3: true, 9: true},
		5:  {1: true, 6: true, 10: true},
		6:  {2: true, 5: true, 7: true},
		7:  {3: true, 6: true, 11: true},
		8:  {0: true, 9: true},
		9:  {4: true, 8: true, 10: true},
		10: {5: true, 9: true, 11: true},
		11: {7: true, 10: true},
	}

	plateIsOcean := AssignPlateTypes(sortedPlates, plateSizes, plateNeighbors, 1000, 0.29)
	mask := assignmentMask(sortedPlates, plateIsOcean)
	stats := evaluatePlateAssignment(mask, sortedPlates, plateSizes, plateNeighbors, 1000)

	if stats.continentalComponents < 2 || stats.continentalComponents > 5 {
		t.Fatalf("continental components = %d, want between 2 and 5", stats.continentalComponents)
	}
	if stats.oceanComponents < 1 || stats.oceanComponents > 3 {
		t.Fatalf("ocean components = %d, want between 1 and 3", stats.oceanComponents)
	}
	if stats.largestContinentShare > 0.65 {
		t.Fatalf("largest continent share = %.3f, want <= 0.65", stats.largestContinentShare)
	}
}

func TestGeneratePlatesAvoidsSuperplates(t *testing.T) {
	sites, faces := icosphere.CreateIcosphere(3)
	_, cells := icosphere.GenerateSphericalVoronoi(sites, faces)

	testSeeds := []int64{1, 2, 4, 8, 42}
	for _, seed := range testSeeds {
		rng := rand.New(rand.NewSource(seed))
		plateR, rPlate := GeneratePlates(sites, cells, 12, rng)

		plateSizes := make(map[int]int, len(plateR))
		for _, plate := range rPlate {
			plateSizes[plate]++
		}

		maxPlate := 0
		for _, center := range plateR {
			if plateSizes[center] > maxPlate {
				maxPlate = plateSizes[center]
			}
		}

		fraction := float64(maxPlate) / float64(len(sites))
		if fraction > 0.45 {
			t.Fatalf("seed %d produced superplate fraction %.3f, want <= 0.45", seed, fraction)
		}
	}
}

func TestGenerateOptimizedPlateLayoutImprovesInitialSeed6Layout(t *testing.T) {
	sites, faces := icosphere.CreateIcosphere(3)
	_, cells := icosphere.GenerateSphericalVoronoi(sites, faces)

	targetLandFraction := 0.29
	initial := generatePlateLayout(sites, cells, 12, rand.New(rand.NewSource(6)), 0)
	initialCandidate, ok := findBestPlateTypeAssignmentCandidate(
		initial.sortedPlates,
		initial.plateSizes,
		initial.plateNeighbors,
		len(sites),
		targetLandFraction,
	)
	if !ok {
		t.Fatal("initial layout did not produce a continental assignment")
	}

	optimized := GenerateOptimizedPlateLayout(sites, cells, 12, 6, targetLandFraction)
	optimizedCandidate, ok := findBestPlateTypeAssignmentCandidate(
		optimized.sortedPlates,
		optimized.plateSizes,
		optimized.plateNeighbors,
		len(sites),
		targetLandFraction,
	)
	if !ok {
		t.Fatal("optimized layout did not produce a continental assignment")
	}

	if optimizedCandidate.score > initialCandidate.score+1e-9 {
		t.Fatalf("optimized layout score %.3f worse than initial %.3f", optimizedCandidate.score, initialCandidate.score)
	}
	if optimizedCandidate.stats.largestContinentShare > initialCandidate.stats.largestContinentShare+0.05 {
		t.Fatalf("optimized layout largest continent %.3f unexpectedly much worse than initial %.3f",
			optimizedCandidate.stats.largestContinentShare, initialCandidate.stats.largestContinentShare)
	}
}

func ringPlateNeighbors(numPlates int) map[int]map[int]bool {
	neighbors := make(map[int]map[int]bool, numPlates)
	for i := 0; i < numPlates; i++ {
		neighbors[i] = map[int]bool{
			(i - 1 + numPlates) % numPlates: true,
			(i + 1) % numPlates:             true,
		}
	}
	return neighbors
}

func assignmentMask(sortedPlates []PlateSize, plateIsOcean map[int]bool) uint64 {
	var mask uint64
	for i, ps := range sortedPlates {
		if !plateIsOcean[ps.Center] {
			mask |= uint64(1) << i
		}
	}
	return mask
}
