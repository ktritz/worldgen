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
