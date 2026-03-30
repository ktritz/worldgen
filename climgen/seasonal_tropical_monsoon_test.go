package climgen

import "testing"

func TestSeasonalTropicalMoisturePlacementFieldSpreadsIntoSummerInterior(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(12, -30),
		seasonalLatLonVertex(12, -20),
		seasonalLatLonVertex(12, -10),
		seasonalLatLonVertex(12, 0),
		seasonalLatLonVertex(-12, 0),
	}
	elevation := []float64{-100, 100, 100, 100, 100}
	adj := &FlatAdjacency{
		Neighbors: []int{
			1,
			0, 2,
			1, 3,
			2, 4,
			3,
		},
		Offsets: []int{0, 1, 3, 5, 7, 8},
	}
	wind := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
	}
	temperature := []float64{299.15, 301.15, 302.15, 303.15, 295.15}
	annual := []float64{297.15, 297.15, 297.15, 297.15, 297.15}
	landInterior := []float64{0.0, 0.15, 0.45, 0.85, 0.85}

	field := computeSeasonalTropicalMoisturePlacementField(
		vertices,
		elevation,
		0.0,
		adj,
		wind,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5},
		temperature,
		annual,
		landInterior,
	)

	if field[2] <= 0 {
		t.Fatalf("expected monsoon placement support at first inland summer-tropical cell, got %.3f", field[2])
	}
	if field[3] <= field[2]*0.5 {
		t.Fatalf("expected deeper inland tropical support to retain at least half of prior inland support: near=%.3f deep=%.3f", field[2], field[3])
	}
	if field[3] <= field[4] {
		t.Fatalf("expected NH summer tropical interior to exceed opposite-hemisphere counterpart: north=%.3f south=%.3f", field[3], field[4])
	}
}

func TestSeasonalTropicalRegimeFieldsSeparatePersistentWetAndDryPocket(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(6, -30),
		seasonalLatLonVertex(6, -20),
		seasonalLatLonVertex(6, -10),
		seasonalLatLonVertex(18, 0),
	}
	elevation := []float64{-100, 100, 100, 100}
	adj := &FlatAdjacency{
		Neighbors: []int{
			1,
			0, 2,
			1, 3,
			2,
		},
		Offsets: []int{0, 1, 3, 5, 6},
	}
	wind := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
	}
	temperature := []float64{299.15, 301.15, 302.15, 304.15}
	annual := []float64{297.15, 297.15, 297.15, 297.15}
	landInterior := []float64{0.0, 0.15, 0.45, 0.95}

	fields := computeSeasonalTropicalRegimeFields(
		vertices,
		elevation,
		0.0,
		adj,
		wind,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5},
		temperature,
		annual,
		landInterior,
	)

	if fields.PersistentWet[2] <= 0 {
		t.Fatalf("expected persistent equatorial wet support at low-lat inland cell, got %.3f", fields.PersistentWet[2])
	}
	if fields.ITCZCrossing[2] <= 0 {
		t.Fatalf("expected ITCZ-crossing support at low-lat inland cell, got %.3f", fields.ITCZCrossing[2])
	}
	if fields.DryPocket[3] <= fields.DryPocket[2] {
		t.Fatalf("expected drier pocket signal at hotter inland subtropical cell: near=%.3f dry=%.3f", fields.DryPocket[2], fields.DryPocket[3])
	}
	if fields.PersistentWet[3] >= fields.PersistentWet[2] {
		t.Fatalf("expected persistent wet support to weaken away from equatorial core: eq=%.3f subtrop=%.3f", fields.PersistentWet[2], fields.PersistentWet[3])
	}
	if fields.ITCZCrossing[3] >= fields.ITCZCrossing[2] {
		t.Fatalf("expected ITCZ-crossing support to weaken away from the equatorial migration envelope: eq=%.3f subtrop=%.3f", fields.ITCZCrossing[2], fields.ITCZCrossing[3])
	}
}

func TestSeasonalTropicalRegimeFieldsNeedOceanGeometryForStrongInlandSupport(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(10, -30),
		seasonalLatLonVertex(10, -20),
		seasonalLatLonVertex(10, -10),
		seasonalLatLonVertex(10, 0),
	}
	elevation := []float64{100, 100, 100, 100}
	adj := &FlatAdjacency{
		Neighbors: []int{
			1,
			0, 2,
			1, 3,
			2,
		},
		Offsets: []int{0, 1, 3, 5, 6},
	}
	wind := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
	}
	temperature := []float64{302.15, 303.15, 304.15, 305.15}
	annual := []float64{297.15, 297.15, 297.15, 297.15}
	landInterior := []float64{0.2, 0.4, 0.7, 0.95}

	fields := computeSeasonalTropicalRegimeFields(
		vertices,
		elevation,
		0.0,
		adj,
		wind,
		SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5},
		temperature,
		annual,
		landInterior,
	)

	if fields.MaritimeAccess[3] > 0.15 {
		t.Fatalf("expected landlocked tropical interior to keep low maritime access, got %.3f", fields.MaritimeAccess[3])
	}
	if fields.Placement[3] > 0.25 {
		t.Fatalf("expected landlocked tropical interior to avoid strong monsoon placement without ocean geometry, got %.3f", fields.Placement[3])
	}
}
