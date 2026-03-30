package climgen

import "testing"

func TestApplyFrontalLandDiffusionBroadensInteriorStormBand(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(50, -30),
		seasonalLatLonVertex(50, -20),
		seasonalLatLonVertex(50, -10),
		seasonalLatLonVertex(50, 0),
	}
	elevation := []float64{-100, 100, 100, 100}
	adj := &FlatAdjacency{
		Neighbors: []int{1, 0, 2, 1, 3, 2},
		Offsets:   []int{0, 1, 3, 5, 6},
	}
	wind := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
	}
	oceanFetch := []float64{0, 1.0, 0.8, 0.5}
	landTravel := []float64{0, 0.0, 0.4, 0.8}
	landInterior := []float64{0, 0.1, 0.5, 0.9}
	frontal := []float64{0, 1.0, 0.0, 0.0}

	applyFrontalLandDiffusion(frontal, vertices, elevation, 0.0, adj, wind, oceanFetch, landTravel, landInterior)

	if frontal[2] <= 0 {
		t.Fatalf("expected frontal diffusion to spread storm moisture inland, got %.3f", frontal[2])
	}
	if frontal[3] <= frontal[2]*0.15 {
		t.Fatalf("expected deeper inland frontal diffusion to retain some corridor moisture: near=%.3f deep=%.3f", frontal[2], frontal[3])
	}
}

func TestApplyFrontalStormTransportCarriesBandFartherInland(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(50, -40),
		seasonalLatLonVertex(50, -30),
		seasonalLatLonVertex(50, -20),
		seasonalLatLonVertex(50, -10),
		seasonalLatLonVertex(50, 0),
	}
	elevation := []float64{-100, 100, 100, 100, 100}
	adj := &FlatAdjacency{
		Neighbors: []int{1, 0, 2, 1, 3, 2, 4, 3},
		Offsets:   []int{0, 1, 3, 5, 7, 8},
	}
	wind := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
	}
	oceanFetch := []float64{0, 1.0, 0.9, 0.6, 0.3}
	landTravel := []float64{0, 0.0, 0.3, 0.6, 0.9}
	landInterior := []float64{0, 0.1, 0.35, 0.7, 0.95}
	sourceScale := []float64{0, 1.2, 1.4, 1.4, 1.2}
	retentionScale := []float64{0, 1.1, 1.4, 1.5, 1.3}
	transportScale := []float64{0, 1.0, 1.6, 1.8, 1.6}
	frontal := []float64{0, 1.0, 0.2, 0.0, 0.0}

	applyFrontalLandDiffusion(frontal, vertices, elevation, 0.0, adj, wind, oceanFetch, landTravel, landInterior)
	beforeDeep := frontal[4]
	applyFrontalStormTransport(
		frontal,
		vertices,
		elevation,
		0.0,
		adj,
		wind,
		oceanFetch,
		landTravel,
		landInterior,
		sourceScale,
		retentionScale,
		transportScale,
	)

	if frontal[3] <= 0.02 {
		t.Fatalf("expected frontal storm transport to keep a meaningful inland corridor, got %.3f", frontal[3])
	}
	if frontal[4] <= beforeDeep {
		t.Fatalf("expected frontal storm transport to carry moisture farther inland: before=%.3f after=%.3f", beforeDeep, frontal[4])
	}
}

func TestComputeFrontalStormSourceUsesUpwindMarineSupport(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(50, -40),
		seasonalLatLonVertex(50, -30),
		seasonalLatLonVertex(50, -20),
		seasonalLatLonVertex(50, -10),
	}
	elevation := []float64{-100, 100, 100, 100}
	adj := &FlatAdjacency{
		Neighbors: []int{1, 0, 2, 1, 3, 2},
		Offsets:   []int{0, 1, 3, 5, 6},
	}
	wind := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
	}
	marine := []float64{0.9, 0.7, 0.3, 0.1}
	oceanFetch := []float64{0, 1.0, 0.8, 0.4}
	landTravel := []float64{0, 0.0, 0.4, 0.8}
	landInterior := []float64{0, 0.1, 0.5, 0.9}
	sourceScale := []float64{0, 1.2, 1.4, 1.3}
	transportScale := []float64{0, 1.0, 1.8, 1.8}

	near := computeFrontalStormSource(2, marine, vertices, elevation, 0.0, adj, wind, oceanFetch, landTravel, landInterior, sourceScale, transportScale)
	deep := computeFrontalStormSource(3, marine, vertices, elevation, 0.0, adj, wind, oceanFetch, landTravel, landInterior, sourceScale, transportScale)
	if near <= 0 {
		t.Fatalf("expected inland corridor cell to receive frontal storm source, got %.3f", near)
	}
	if deep <= 0 {
		t.Fatalf("expected deeper inland cell to still receive some storm source, got %.3f", deep)
	}
}
