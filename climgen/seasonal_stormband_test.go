package climgen

import "testing"

func TestSeasonalStormBandSupportBroadensInlandWinterMidlatitudes(t *testing.T) {
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
	landInterior := []float64{0.0, 0.1, 0.5, 0.9}
	solar := SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.0}

	support := computeSeasonalStormBandSupportField(vertices, elevation, 0.0, adj, wind, solar, landInterior)
	if support[2] <= support[1] {
		t.Fatalf("expected storm-band support to extend inland beyond first landfall coast: coast=%.3f inland=%.3f", support[1], support[2])
	}
	if support[3] <= support[2]*0.7 {
		t.Fatalf("expected deeper inland support to retain most corridor strength: near=%.3f deep=%.3f", support[2], support[3])
	}
}
