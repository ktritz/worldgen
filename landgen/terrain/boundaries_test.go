package terrain

import (
	"math"
	"testing"
)

func TestFindCollisionsClassifiesContinentalRift(t *testing.T) {
	theta := 0.1
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: math.Cos(theta), Y: math.Sin(theta), Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0}},
	}
	rPlate := []int{0, 1}
	plateIsOcean := map[int]bool{0: false, 1: false}
	plateRot := map[int]PlateRotation{
		0: {Pole: Vector3D{X: 0, Y: 0, Z: 1}, AngularVelocity: -1.0},
		1: {Pole: Vector3D{X: 0, Y: 0, Z: 1}, AngularVelocity: 1.0},
	}

	seeds := FindCollisions(sites, cells, plateIsOcean, rPlate, plateRot)
	if len(seeds.Rift) == 0 {
		t.Fatal("expected divergent continental boundary to create rift seeds")
	}
	if len(seeds.Mountain) != 0 {
		t.Fatalf("expected no mountain seeds for divergent continental boundary, got %d", len(seeds.Mountain))
	}
	if len(seeds.Collision) != 0 {
		t.Fatalf("expected no collision seeds for divergent boundary, got %d", len(seeds.Collision))
	}
}

func TestFindCollisionsClassifiesContinentalCollision(t *testing.T) {
	theta := 0.1
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: math.Cos(theta), Y: math.Sin(theta), Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0}},
	}
	rPlate := []int{0, 1}
	plateIsOcean := map[int]bool{0: false, 1: false}
	plateRot := map[int]PlateRotation{
		0: {Pole: Vector3D{X: 0, Y: 0, Z: 1}, AngularVelocity: 1.0},
		1: {Pole: Vector3D{X: 0, Y: 0, Z: 1}, AngularVelocity: -1.0},
	}

	seeds := FindCollisions(sites, cells, plateIsOcean, rPlate, plateRot)
	if len(seeds.Mountain) == 0 {
		t.Fatal("expected convergent continental boundary to create mountain seeds")
	}
	if len(seeds.Collision) == 0 {
		t.Fatal("expected convergent continental boundary to create collision seeds")
	}
	if len(seeds.Rift) != 0 {
		t.Fatalf("expected no rift seeds for convergent continental boundary, got %d", len(seeds.Rift))
	}
}

func TestPlaceVolcanicArcsCreatesArcBelt(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: math.Cos(0.010), Y: math.Sin(0.010), Z: 0},
		{X: math.Cos(0.022), Y: math.Sin(0.022), Z: 0},
		{X: math.Cos(0.030), Y: math.Sin(0.030), Z: 0},
		{X: math.Cos(0.038), Y: math.Sin(0.038), Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2, 4}},
		{SiteIndex: 4, NeighborSiteIndices: []int32{3}},
	}
	rPlate := []int{0, 1, 1, 1, 1}
	plateIsOcean := map[int]bool{0: true, 1: false}
	trenchR := map[int]bool{0: true}
	arcR := make(map[int]bool)
	mountainR := make(map[int]bool)

	placeVolcanicArcs(sites, cells, rPlate, plateIsOcean, trenchR, arcR, mountainR)

	if len(arcR) < 2 {
		t.Fatalf("expected widened volcanic arc belt, got %d arc seeds", len(arcR))
	}
	if !arcR[2] && !arcR[3] {
		t.Fatalf("expected volcanic arc near target inland distance, got seeds %v", arcR)
	}
}

func TestPlaceContinentalCollisionBeltsExpandsInland(t *testing.T) {
	sites := []Vector3D{
		{X: math.Cos(-0.025), Y: math.Sin(-0.025), Z: 0},
		{X: math.Cos(-0.010), Y: math.Sin(-0.010), Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: math.Cos(0.010), Y: math.Sin(0.010), Z: 0},
		{X: math.Cos(0.025), Y: math.Sin(0.025), Z: 0},
		{X: math.Cos(0.040), Y: math.Sin(0.040), Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2, 4}},
		{SiteIndex: 4, NeighborSiteIndices: []int32{3, 5}},
		{SiteIndex: 5, NeighborSiteIndices: []int32{4}},
	}
	rPlate := []int{0, 0, 0, 1, 1, 1}
	plateIsOcean := map[int]bool{0: false, 1: false}
	collisionBoundary := map[int]bool{2: true, 3: true}
	collisionR := map[int]bool{2: true, 3: true}
	mountainR := map[int]bool{2: true, 3: true}

	placeContinentalCollisionBelts(sites, cells, rPlate, plateIsOcean, collisionBoundary, collisionR, mountainR)

	if !collisionR[1] || !collisionR[4] {
		t.Fatalf("expected inland collision belt on both continents, got %v", collisionR)
	}
}

func TestConnectSeedCentersOnPlateBridgesGap(t *testing.T) {
	sites := []Vector3D{
		{X: math.Cos(0.000), Y: math.Sin(0.000), Z: 0},
		{X: math.Cos(0.010), Y: math.Sin(0.010), Z: 0},
		{X: math.Cos(0.020), Y: math.Sin(0.020), Z: 0},
		{X: math.Cos(0.030), Y: math.Sin(0.030), Z: 0},
		{X: math.Cos(0.040), Y: math.Sin(0.040), Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2, 4}},
		{SiteIndex: 4, NeighborSiteIndices: []int32{3}},
	}
	rPlate := []int{1, 1, 1, 1, 1}
	seeds := map[int]bool{0: true, 4: true}
	target := map[int]bool{0: true, 4: true}

	connectSeedCentersOnPlate(sites, cells, rPlate, seeds, 0.06, 1, 0, target)

	if !target[1] || !target[2] || !target[3] {
		t.Fatalf("expected linked path to fill gap, got %v", target)
	}
}

// placeVolcanicArcs feeds trench cells through a greedy, order-sensitive
// spacing filter. Iterating the trench map directly made arc placement depend
// on Go's randomized map order; the sorted iteration must yield identical
// output on every run.
func TestPlaceVolcanicArcsIsDeterministic(t *testing.T) {
	latLon := func(lat, lon float64) Vector3D {
		return Vector3D{
			X: math.Cos(lat) * math.Cos(lon),
			Y: math.Cos(lat) * math.Sin(lon),
			Z: math.Sin(lat),
		}
	}

	// Four trench cells (0..3) packed within ArcSeedSpacingRadians of each
	// other, each with its own single-cell continental column (4..7) close
	// enough that the first placed arc suppresses every other trench. Which
	// trench wins is exactly what map iteration order used to randomize.
	const numTrenches = 4
	sites := make([]Vector3D, 0, 2*numTrenches)
	cells := make([]VoronoiCell, 0, 2*numTrenches)
	rPlate := make([]int, 0, 2*numTrenches)
	trenchR := map[int]bool{}
	for i := 0; i < numTrenches; i++ {
		sites = append(sites, latLon(0, float64(i)*0.004))
		cells = append(cells, VoronoiCell{SiteIndex: int32(i), NeighborSiteIndices: []int32{int32(numTrenches + i)}})
		rPlate = append(rPlate, 0)
		trenchR[i] = true
	}
	for i := 0; i < numTrenches; i++ {
		sites = append(sites, latLon(0.01, float64(i)*0.004))
		cells = append(cells, VoronoiCell{SiteIndex: int32(numTrenches + i), NeighborSiteIndices: []int32{int32(i)}})
		rPlate = append(rPlate, 1)
	}
	plateIsOcean := map[int]bool{0: true, 1: false}

	var firstArcs, firstMountains map[int]bool
	for run := 0; run < 6; run++ {
		arcR := map[int]bool{}
		mountainR := map[int]bool{}
		placeVolcanicArcs(sites, cells, rPlate, plateIsOcean, trenchR, arcR, mountainR)

		if run == 0 {
			firstArcs = arcR
			firstMountains = mountainR
			// Sorted iteration processes trench 0 first, so its column wins.
			if !arcR[numTrenches] || len(arcR) != 1 {
				t.Fatalf("expected single arc at region %d, got %v", numTrenches, arcR)
			}
			continue
		}
		if len(arcR) != len(firstArcs) || len(mountainR) != len(firstMountains) {
			t.Fatalf("run %d produced different arc counts: %v vs %v", run, arcR, firstArcs)
		}
		for region := range firstArcs {
			if !arcR[region] {
				t.Fatalf("run %d arc set %v differs from first run %v", run, arcR, firstArcs)
			}
		}
		for region := range firstMountains {
			if !mountainR[region] {
				t.Fatalf("run %d mountain set %v differs from first run %v", run, mountainR, firstMountains)
			}
		}
	}
}
