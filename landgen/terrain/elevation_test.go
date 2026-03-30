package terrain

import (
	"math"
	"testing"
)

func TestApplyBimodalElevationPrefersTectonicallySupportedInterior(t *testing.T) {
	elevation := []float64{0, 0}
	distFromCoast := []float64{6, 6}
	oceanDistFromCoast := []float64{math.Inf(1), math.Inf(1)}
	componentMaxDist := []float64{10, 10}
	distFromMountain := []float64{0, 10}
	componentMaxMountainDist := []float64{10, 10}
	distFromCollision := []float64{0, 10}
	componentMaxCollisionDist := []float64{10, 10}
	distFromArc := []float64{math.Inf(1), math.Inf(1)}
	componentMaxArcDist := []float64{1, 1}
	rPlate := []int{0, 0}
	plateIsOcean := map[int]bool{0: false}

	ApplyBimodalElevation(
		elevation,
		distFromCoast,
		oceanDistFromCoast,
		componentMaxDist,
		distFromMountain,
		componentMaxMountainDist,
		distFromCollision,
		componentMaxCollisionDist,
		distFromArc,
		componentMaxArcDist,
		rPlate,
		plateIsOcean,
		10,
		1,
	)

	if elevation[0] <= elevation[1] {
		t.Fatalf("tectonically supported interior elevation %.3f should exceed unsupported basin %.3f", elevation[0], elevation[1])
	}
	if elevation[1] >= 0.15 {
		t.Fatalf("unsupported inland basin elevation %.3f too high; expected basin suppression", elevation[1])
	}
}

func TestRegularizeCoastlinesReducesSmallShorelineSpike(t *testing.T) {
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1, 2, 3, 4}},
		{NeighborSiteIndices: []int32{0}},
		{NeighborSiteIndices: []int32{0}},
		{NeighborSiteIndices: []int32{0}},
		{NeighborSiteIndices: []int32{0}},
	}
	elevation := []float64{120, 160, -90, -70, -40}
	coastalExposure := []float64{0.1, 0.0, 0.0, 0.0, 0.0}

	RegularizeCoastlines(cells, elevation, coastalExposure, 2)

	if elevation[0] >= 120 {
		t.Fatalf("coastal spike stayed too sharp: got %.1f, want < 120", elevation[0])
	}
	if elevation[1] != 160 {
		t.Fatalf("interior land cell changed unexpectedly: got %.1f, want 160", elevation[1])
	}
}

func TestApplyOceanBasinStructureCreatesRidgeToTrenchGradient(t *testing.T) {
	elevation := []float64{-0.60, -0.60, -0.60}
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0, Y: 1, Z: 0},
		{X: -1, Y: 0, Z: 0},
	}
	rPlate := []int{0, 0, 0}
	plateIsOcean := map[int]bool{0: true}
	plateRot := map[int]PlateRotation{
		0: {
			Pole:            Vector3D{X: 0, Y: 0, Z: 1},
			AngularVelocity: 0.2,
		},
	}
	oceanDistFromCoast := []float64{12, 12, 12}
	distFromRidge := []float64{0, 4, 9}
	maxRidgeDist := []float64{10, 10, 10}
	distFromTrench := []float64{math.Inf(1), math.Inf(1), 0}

	ApplyOceanBasinStructure(
		elevation,
		sites,
		rPlate,
		plateIsOcean,
		plateRot,
		oceanDistFromCoast,
		distFromRidge,
		maxRidgeDist,
		distFromTrench,
		12,
		42,
	)

	if elevation[0] <= elevation[1] {
		t.Fatalf("ridge flank should stay shallower than mature basin: ridge %.3f basin %.3f", elevation[0], elevation[1])
	}
	if elevation[2] >= elevation[1] {
		t.Fatalf("trench margin should deepen relative to mature basin: trench %.3f basin %.3f", elevation[2], elevation[1])
	}
}
