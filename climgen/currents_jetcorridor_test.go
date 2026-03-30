package climgen

import "testing"

func TestApplyWesternBoundaryJetCorridorImprovesModerateBoundaryAlignment(t *testing.T) {
	vertices := []Vector3D{
		Normalize(Vector3D{X: 1.0, Y: 0.30, Z: 0.00}),
		Normalize(Vector3D{X: 1.0, Y: 0.34, Z: 0.02}),
	}
	elevation := []float64{-1000, -1000}
	landDir := Normalize(Vector3D{X: -0.1, Y: 0.0, Z: 1.0})
	coastDirs := []Vector3D{landDir, landDir}
	western := []float64{0.45, 0.12}
	gateway := []float64{0.0, 0.0}
	componentAssignments := []int{0, 0}

	seedAxis := coastParallelAxis(vertices[0], coastDirs[0], Vector3D{X: 0.2, Y: 0.3, Z: 0.4})
	target := Scale(seedAxis, 1.0)
	currents := []Vector3D{
		target,
		{},
	}
	perp := Cross(vertices[1], seedAxis)
	perp = Normalize(Sub(perp, Scale(vertices[1], Dot(perp, vertices[1]))))
	currents[1] = Normalize(Add(Scale(seedAxis, 0.20), Scale(perp, 0.98)))

	before := Dot(Normalize(currents[1]), Normalize(seedAxis))
	shaped := ApplyWesternBoundaryJetCorridor(
		currents, vertices, elevation, 0.0, coastDirs, western, gateway, componentAssignments,
	)
	after := Dot(Normalize(shaped[1]), Normalize(seedAxis))
	if after <= before {
		t.Fatalf("expected jet corridor to improve along-coast alignment: before=%.3f after=%.3f", before, after)
	}
}
