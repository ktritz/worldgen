package climgen

import "testing"

func TestBuildOceanOpennessFieldPenalizesNarrowGateways(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(4)
	for i, v := range vertices {
		if v.X > 0.35 && v.X < 0.75 && absFloat(v.Z) < 0.22 {
			elevation[i] = 500
		}
		if v.X > 0.52 && v.X < 0.58 && absFloat(v.Z) < 0.04 {
			elevation[i] = -1000
		}
	}

	assignments, components := FindOceanComponents(elevation, 0.0, adj)
	openness := BuildOceanOpennessField(vertices, elevation, 0.0, adj, assignments, components)

	gateway := -1
	openOcean := -1
	for i, v := range vertices {
		if elevation[i] >= 0 {
			continue
		}
		if gateway < 0 && v.X > 0.50 && v.X < 0.62 && absFloat(v.Z) < 0.08 {
			gateway = i
		}
		if openOcean < 0 && v.X < -0.4 && absFloat(v.Z) > 0.25 {
			openOcean = i
		}
	}

	if gateway < 0 || openOcean < 0 {
		t.Fatalf("failed to locate comparison cells")
	}
	if openness[gateway] >= openness[openOcean] {
		t.Fatalf("gateway openness %.3f should be below open-ocean openness %.3f", openness[gateway], openness[openOcean])
	}
}

func TestBuildGatewayFieldDetectsStraitAxis(t *testing.T) {
	vertices := []Vector3D{
		Normalize(Vector3D{X: 1, Y: 0, Z: 0}),
		Normalize(Vector3D{X: 1, Y: 0.18, Z: 0}),
		Normalize(Vector3D{X: 1, Y: -0.18, Z: 0}),
		Normalize(Vector3D{X: 1, Y: 0, Z: 0.14}),
		Normalize(Vector3D{X: 1, Y: 0, Z: -0.14}),
	}
	elevation := []float64{-1000, -1000, -1000, 500, 500}
	assignments := []int{0, 0, 0, -1, -1}
	adj := &FlatAdjacency{
		Neighbors: []int{1, 2, 3, 4, 0, 0, 0, 0},
		Offsets:   []int{0, 4, 5, 6, 7, 8},
	}
	openness := []float64{0.15, 0.4, 0.4, 0, 0}

	strength, axis := BuildGatewayField(vertices, elevation, 0.0, adj, assignments, openness)
	if strength[0] <= 0 {
		t.Fatalf("expected synthetic strait center to register as gateway, got %.3f", strength[0])
	}
	if Length(axis[0]) < 0.1 {
		t.Fatalf("expected gateway axis to be defined")
	}
}

func TestApplyStructuredCurrentScalingBoostsPolewardWesternFlow(t *testing.T) {
	vertices := []Vector3D{
		{X: 0, Y: 0.6, Z: 0.8},
	}
	elevation := []float64{-1000}
	currents := []Vector3D{
		{X: 0.3, Y: 0.4, Z: 0.0},
	}
	western := []float64{1.0}
	openness := []float64{1.0}
	gateway := []float64{0.0}
	gatewayAxis := []Vector3D{{}}

	boosted := ApplyStructuredCurrentScaling(currents, vertices, elevation, 0.0, western, openness, gateway, gatewayAxis)
	if Length(boosted[0]) <= Length(currents[0]) {
		t.Fatalf("expected western boundary poleward flow to intensify")
	}

	currents[0] = Vector3D{X: 0.3, Y: -0.4, Z: 0.0}
	damped := ApplyStructuredCurrentScaling(currents, vertices, elevation, 0.0, western, openness, gateway, gatewayAxis)
	if Length(damped[0]) >= Length(currents[0]) {
		t.Fatalf("expected equatorward western-boundary flow to damp")
	}
}

func TestApplyStructuredCurrentScalingPreservesAlongGatewayFlow(t *testing.T) {
	vertices := []Vector3D{
		Normalize(Vector3D{X: 0.8, Y: 0.2, Z: 0.5}),
	}
	elevation := []float64{-1000}
	currents := []Vector3D{
		{X: 0.6, Y: 0.0, Z: 0.0},
	}
	axis := Normalize(Vector3D{X: 1, Y: 0, Z: 0})

	scaled := ApplyStructuredCurrentScaling(
		currents,
		vertices,
		elevation,
		0.0,
		[]float64{0.0},
		[]float64{0.2},
		[]float64{0.9},
		[]Vector3D{axis},
	)
	if Length(scaled[0]) < 0.3 {
		t.Fatalf("expected gateway throughflow to be preserved, got %.3f", Length(scaled[0]))
	}
}
