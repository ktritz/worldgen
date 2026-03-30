package climgen

import "testing"

func TestBuildTemperatureTransportWindFieldPrefersMarineOverOcean(t *testing.T) {
	elevation := []float64{-1000, 500, -1000}
	wind := &WindResult{
		MarineWind: []Vector3D{
			{X: 1, Y: 0, Z: 0},
			{X: 2, Y: 0, Z: 0},
			{X: 3, Y: 0, Z: 0},
		},
		SurfaceWind: []Vector3D{
			{X: 10, Y: 0, Z: 0},
			{X: 20, Y: 0, Z: 0},
			{X: 30, Y: 0, Z: 0},
		},
	}

	got := BuildTemperatureTransportWindField(wind, elevation, 0.0)
	if got[0].X != 1 || got[1].X != 20 || got[2].X != 3 {
		t.Fatalf("unexpected blended wind field: %+v", got)
	}
}

func TestApplyResolvedMaritimeInfluenceUsesOnshoreMarineWind(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(2)
	land := 0
	elevation[land] = 500

	oceanNeighbor := -1
	for _, k := range adj.GetNeighbors(land) {
		if k >= 0 && elevation[k] < 0 {
			oceanNeighbor = k
			break
		}
	}
	if oceanNeighbor < 0 {
		t.Fatalf("expected a coastal ocean neighbor")
	}

	marineWind := make([]Vector3D, len(vertices))
	fromOcean := Normalize(Sub(vertices[land], vertices[oceanNeighbor]))
	marineWind[land] = fromOcean

	temperature := make([]float64, len(vertices))
	for i := range temperature {
		temperature[i] = 280
	}
	temperature[oceanNeighbor] = 300

	adjusted := ApplyResolvedMaritimeInfluence(
		temperature,
		vertices,
		elevation,
		0.0,
		adj,
		&WindResult{MarineWind: marineWind},
		nil,
		DefaultTransportSettings(),
	)

	if adjusted[land] <= temperature[land] {
		t.Fatalf("expected coastal land to warm from maritime influence: %.2f <= %.2f", adjusted[land], temperature[land])
	}
}

func TestApplyMaritimeAnomalyEffectTransfersWarmCurrentSignal(t *testing.T) {
	temperature := []float64{280, 280}
	elevation := []float64{-1000, 500}
	maritime := &MaritimeResult{
		Influence:  []float64{0, 0.9},
		SourceTemp: []float64{0, 4.0},
	}

	adjusted := ApplyMaritimeAnomalyEffect(
		temperature, elevation, 0.0, maritime, DefaultMaritimeSettings(),
	)
	if adjusted[1] <= temperature[1] {
		t.Fatalf("expected warm anomaly to warm land: %.2f <= %.2f", adjusted[1], temperature[1])
	}
}
