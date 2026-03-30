package climgen

import "testing"

func TestApplyGyreInteriorGuidanceSuppressesInteriorSwirl(t *testing.T) {
	vertices := []Vector3D{
		Normalize(Vector3D{X: 1.0, Y: 0.00, Z: 0.00}),
		Normalize(Vector3D{X: 0.98, Y: 0.03, Z: 0.18}),
		Normalize(Vector3D{X: 0.98, Y: -0.04, Z: -0.17}),
		Normalize(Vector3D{X: 0.97, Y: 0.02, Z: 0.23}),
		Normalize(Vector3D{X: 0.97, Y: -0.02, Z: -0.24}),
	}
	elevation := []float64{-1000, -1000, -1000, -1000, -1000}
	openness := []float64{0.92, 0.92, 0.92, 0.92, 0.92}
	western := []float64{0, 0, 0, 0, 0}
	gateway := []float64{0, 0, 0, 0, 0}
	componentAssignments := []int{0, 0, 0, 0, 0}

	currents := make([]Vector3D, len(vertices))
	for i, v := range vertices {
		east, _ := GetTangentVectors(v)
		currents[i] = Scale(east, 0.8)
	}

	east0, north0 := GetTangentVectors(vertices[0])
	currents[0] = Normalize(Add(Scale(east0, 0.15), Scale(north0, 0.95)))

	beforeEast := Dot(currents[0], east0)
	beforeNorth := absFloat(Dot(currents[0], north0))

	guided := ApplyGyreInteriorGuidance(
		currents, vertices, elevation, 0.0, componentAssignments, openness, western, gateway,
	)

	afterEast := Dot(guided[0], east0)
	afterNorth := absFloat(Dot(guided[0], north0))

	if afterEast <= beforeEast {
		t.Fatalf("expected gyre guidance to increase zonal alignment: before=%.3f after=%.3f", beforeEast, afterEast)
	}
	if afterNorth >= beforeNorth {
		t.Fatalf("expected gyre guidance to reduce meridional anomaly: before=%.3f after=%.3f", beforeNorth, afterNorth)
	}
}
