package climgen

import "testing"

func TestSeasonalDynamicPrecipitationForcingFavorsSummerHemisphereTropics(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(10, 0),
		seasonalLatLonVertex(-10, 0),
	}
	elevation := []float64{100, 100}
	adj := &FlatAdjacency{Offsets: []int{0, 0, 0}}
	wind := []Vector3D{{X: 1}, {X: 1}}
	temperature := []float64{295.15, 287.15}
	annual := []float64{289.15, 289.15}
	interior := []float64{0.2, 0.2}

	settings := ApplySeasonalDynamicPrecipitationForcing(
		DefaultPrecipitationSettings(),
		vertices,
		elevation,
		0.0,
		adj,
		wind,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5},
		temperature,
		annual,
		interior,
	)

	if len(settings.CondensationLocalScale) != 2 {
		t.Fatalf("expected condensation scale for both cells, got %d", len(settings.CondensationLocalScale))
	}
	if settings.CondensationLocalScale[0] <= settings.CondensationLocalScale[1] {
		t.Fatalf("expected NH summer to favor northern tropics: north=%.3f south=%.3f",
			settings.CondensationLocalScale[0], settings.CondensationLocalScale[1])
	}
	if settings.LandSourceLocalScale[0] <= settings.LandSourceLocalScale[1] {
		t.Fatalf("expected NH summer to favor northern tropical moisture source: north=%.3f south=%.3f",
			settings.LandSourceLocalScale[0], settings.LandSourceLocalScale[1])
	}
	if settings.LandRecyclingLocalScale[0] <= settings.LandRecyclingLocalScale[1] {
		t.Fatalf("expected NH summer to favor northern tropical recycling: north=%.3f south=%.3f",
			settings.LandRecyclingLocalScale[0], settings.LandRecyclingLocalScale[1])
	}
	if settings.TropicalSourceLocalScale[0] <= settings.TropicalSourceLocalScale[1] {
		t.Fatalf("expected NH summer to favor northern tropical marine source: north=%.3f south=%.3f",
			settings.TropicalSourceLocalScale[0], settings.TropicalSourceLocalScale[1])
	}
}

func TestSeasonalDynamicPrecipitationForcingBoostsWinterStormTrack(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(50, 0),
	}
	elevation := []float64{100}
	adj := &FlatAdjacency{Offsets: []int{0, 0}}
	wind := []Vector3D{{X: 1}}
	temperature := []float64{276.15}
	annual := []float64{281.15}
	interior := []float64{0.4}

	winter := ApplySeasonalDynamicPrecipitationForcing(
		DefaultPrecipitationSettings(),
		vertices,
		elevation,
		0.0,
		adj,
		wind,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.0},
		temperature,
		annual,
		interior,
	)
	summer := ApplySeasonalDynamicPrecipitationForcing(
		DefaultPrecipitationSettings(),
		vertices,
		elevation,
		0.0,
		adj,
		wind,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5},
		[]float64{286.15},
		annual,
		interior,
	)

	if winter.CondensationLocalScale[0] <= summer.CondensationLocalScale[0] {
		t.Fatalf("expected winter-hemisphere midlatitudes to get stronger storm-track forcing: winter=%.3f summer=%.3f",
			winter.CondensationLocalScale[0], summer.CondensationLocalScale[0])
	}
	if winter.FrontalSourceLocalScale[0] <= summer.FrontalSourceLocalScale[0] {
		t.Fatalf("expected winter-hemisphere midlatitudes to get stronger frontal source forcing: winter=%.3f summer=%.3f",
			winter.FrontalSourceLocalScale[0], summer.FrontalSourceLocalScale[0])
	}
}

func TestSeasonalStormMoisturePotentialSpreadsInland(t *testing.T) {
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

	frontal := computeSeasonalFrontalExposureField(vertices, elevation, 0.0, adj, wind, solar)
	storm := computeSeasonalStormMoisturePotentialField(vertices, elevation, 0.0, adj, wind, solar, landInterior, frontal)

	if storm[2] <= 0 {
		t.Fatalf("expected nonzero inland storm potential at first interior cell, got %.3f", storm[2])
	}
	if storm[3] <= frontal[3] {
		t.Fatalf("expected storm potential to propagate beyond raw frontal exposure at deeper inland cell: frontal=%.3f storm=%.3f", frontal[3], storm[3])
	}
	if storm[3] <= storm[2]*0.5 {
		t.Fatalf("expected deeper inland storm memory to retain at least half of prior inland band strength: near=%.3f deep=%.3f", storm[2], storm[3])
	}
}
