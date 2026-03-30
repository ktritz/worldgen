package climgen

import "testing"

func TestApplyWesternBoundaryJetShapingImprovesCoastAlignment(t *testing.T) {
	vertices := []Vector3D{
		Normalize(Vector3D{X: 1, Y: 0.35, Z: 0.2}),
	}
	elevation := []float64{-1000}
	currents := []Vector3D{
		{X: 0.2, Y: 0.25, Z: 0.35},
	}
	landDir := Normalize(Vector3D{X: -0.2, Y: 0.0, Z: 1.0})
	western := []float64{1.0}

	before := mathAbs(Dot(Normalize(currents[0]), Normalize(Cross(vertices[0], landDir))))
	afterField := ApplyWesternBoundaryJetShaping(
		currents, vertices, elevation, 0.0, []Vector3D{landDir}, western,
	)
	after := mathAbs(Dot(Normalize(afterField[0]), Normalize(Cross(vertices[0], landDir))))
	if after <= before {
		t.Fatalf("expected coast-aligned jet shaping to improve alignment: before=%.3f after=%.3f", before, after)
	}
}

func TestSmoothOpenOceanCurrentsReducesInteriorTexture(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(3)
	currents := make([]Vector3D, len(vertices))
	openness := make([]float64, len(vertices))
	western := make([]float64, len(vertices))
	gateway := make([]float64, len(vertices))

	for i, v := range vertices {
		if elevation[i] >= 0 {
			continue
		}
		currents[i] = Vector3D{X: 0.3 * v.Z, Y: 0.2 - 0.1*v.X, Z: -0.25 * v.Y}
		openness[i] = 0.9
	}
	// Add one noisy interior cell.
	currents[0] = Vector3D{X: 0.9, Y: -0.4, Z: 0.2}

	smoothed := SmoothOpenOceanCurrents(
		currents, vertices, elevation, 0.0, adj, openness, western, gateway,
	)
	if Length(Sub(smoothed[0], currents[0])) <= 0.01 {
		t.Fatalf("expected open-ocean smoother to change noisy interior current")
	}
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
