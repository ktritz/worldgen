package terrain

import (
	"math"
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

func TestGeneratePlatesUsesStablePhysicalCentersAcrossResolution(t *testing.T) {
	coarseSites, coarseFaces := icosphere.CreateIcosphere(3)
	_, coarseCells := icosphere.GenerateSphericalVoronoi(coarseSites, coarseFaces)
	fineSites, fineFaces := icosphere.CreateIcosphere(4)
	_, fineCells := icosphere.GenerateSphericalVoronoi(fineSites, fineFaces)

	coarseRng := rand.New(rand.NewSource(42))
	fineRng := rand.New(rand.NewSource(42))
	coarsePlateR, _ := GeneratePlates(coarseSites, coarseCells, 12, coarseRng)
	finePlateR, _ := GeneratePlates(fineSites, fineCells, 12, fineRng)

	if len(coarsePlateR) != len(finePlateR) {
		t.Fatalf("plate center counts differ: coarse=%d fine=%d", len(coarsePlateR), len(finePlateR))
	}
	for i := range coarsePlateR {
		distance := angularDistance(coarseSites[coarsePlateR[i]], fineSites[finePlateR[i]])
		if distance > 0.12 {
			t.Fatalf("plate center %d moved %.3f radians across resolution", i, distance)
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

func TestOptimizedPlateBlueprintProjectsContinentsAcrossResolution(t *testing.T) {
	refSites, refFaces := icosphere.CreateIcosphere(3)
	_, refCells := icosphere.GenerateSphericalVoronoi(refSites, refFaces)
	fineSites, fineFaces := icosphere.CreateIcosphere(4)
	_, fineCells := icosphere.GenerateSphericalVoronoi(fineSites, fineFaces)

	blueprint, ok := buildPlateBlueprintFromReference(refSites, refCells, 12, 314, 0.29)
	if !ok {
		t.Fatal("expected reference blueprint")
	}

	refLayout := buildPlateLayoutFromBlueprint(refSites, refCells, blueprint)
	fineLayout := buildPlateLayoutFromBlueprint(fineSites, fineCells, blueprint)

	if len(refLayout.plateR) != len(fineLayout.plateR) {
		t.Fatalf("projected active plate counts differ: ref=%d fine=%d", len(refLayout.plateR), len(fineLayout.plateR))
	}
	for i := range refLayout.plateR {
		distance := angularDistance(refLayout.plateCenters[i], fineLayout.plateCenters[i])
		if distance > 1e-9 {
			t.Fatalf("plate blueprint center %d changed by %.6f radians", i, distance)
		}
		if refLayout.plateIsOcean[refLayout.plateR[i]] != fineLayout.plateIsOcean[fineLayout.plateR[i]] {
			t.Fatalf("plate %d changed ocean/continent assignment across resolution", i)
		}
	}

	refMeanLat := meanContinentalLatitude(refSites, refLayout)
	fineMeanLat := meanContinentalLatitude(fineSites, fineLayout)
	if math.Abs(refMeanLat-fineMeanLat) > 0.08 {
		t.Fatalf("continental mean latitude drifted %.3f radians across projection", math.Abs(refMeanLat-fineMeanLat))
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

func meanContinentalLatitude(sites []Vector3D, layout plateLayout) float64 {
	total := 0.0
	count := 0
	for r, site := range sites {
		if layout.plateIsOcean[layout.rPlate[r]] {
			continue
		}
		total += math.Asin(site.Normalize().Z)
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}
