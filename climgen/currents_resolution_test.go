package climgen

import (
	"math"
	"testing"
)

// buildTestAdjacency constructs a FlatAdjacency for total vertices where only
// the first len(neighborLists) vertices have neighbors.
func buildTestAdjacency(total int, neighborLists [][]int) *FlatAdjacency {
	offsets := make([]int, total+1)
	neighbors := make([]int, 0, 16)
	for i := 0; i < total; i++ {
		offsets[i] = len(neighbors)
		if i < len(neighborLists) {
			neighbors = append(neighbors, neighborLists[i]...)
		}
	}
	offsets[total] = len(neighbors)
	return &FlatAdjacency{Neighbors: neighbors, Offsets: offsets}
}

// makeChainAdjacency links the first chainLen vertices into a simple chain.
func makeChainAdjacency(total, chainLen int) *FlatAdjacency {
	lists := make([][]int, chainLen)
	for i := 0; i < chainLen; i++ {
		if i > 0 {
			lists[i] = append(lists[i], i-1)
		}
		if i+1 < chainLen {
			lists[i] = append(lists[i], i+1)
		}
	}
	return buildTestAdjacency(total, lists)
}

func TestFilterComponentsBySizeUsesBaselineEquivalentArea(t *testing.T) {
	makeComponent := func(id, start, count int) OceanComponent {
		verts := make([]int, count)
		for i := range verts {
			verts[i] = start + i
		}
		return OceanComponent{ID: id, Vertices: verts}
	}
	makeAssignments := func(total int, components []OceanComponent) []int {
		assignments := make([]int, total)
		for i := range assignments {
			assignments[i] = -1
		}
		for _, c := range components {
			for _, v := range c.Vertices {
				assignments[v] = c.ID
			}
		}
		return assignments
	}

	// Baseline (L5): raw counts equal baseline-equivalent counts.
	baseComponents := []OceanComponent{
		makeComponent(0, 0, 100),
		makeComponent(1, 100, 40),
	}
	baseAssignments, baseFiltered := FilterComponentsBySize(
		makeAssignments(10242, baseComponents), baseComponents, 50)
	if len(baseFiltered) != 1 || len(baseFiltered[0].Vertices) != 100 {
		t.Fatalf("expected only the 100-cell component to survive at L5, got %d components", len(baseFiltered))
	}
	if baseAssignments[0] != 0 || baseAssignments[100] != -1 {
		t.Fatalf("expected L5 assignments remapped to kept=0/dropped=-1, got %d %d", baseAssignments[0], baseAssignments[100])
	}

	// L6: the same physical seas cover ~4x the cells; pass/fail must match L5.
	// The 100-cell component is ~25 baseline-equivalent cells at L6 and must be
	// dropped even though its raw count exceeds the threshold.
	fineComponents := []OceanComponent{
		makeComponent(0, 0, 400),
		makeComponent(1, 400, 160),
		makeComponent(2, 560, 100),
	}
	fineAssignments, fineFiltered := FilterComponentsBySize(
		makeAssignments(40962, fineComponents), fineComponents, 50)
	if len(fineFiltered) != 1 || len(fineFiltered[0].Vertices) != 400 {
		t.Fatalf("expected only the 400-cell component to survive at L6, got %d components", len(fineFiltered))
	}
	if fineAssignments[0] != 0 || fineAssignments[400] != -1 || fineAssignments[560] != -1 {
		t.Fatalf("expected L6 assignments remapped to kept=0/dropped=-1, got %d %d %d",
			fineAssignments[0], fineAssignments[400], fineAssignments[560])
	}
}

// ringComponentScale builds a mesh of totalCells vertices whose first ringCells
// vertices form one water component evenly spread on a circle of the given
// angular radius around the north pole, and returns its component scale.
func ringComponentScale(t *testing.T, totalCells, ringCells int, angularRadius float64) float64 {
	t.Helper()
	vertices := make([]Vector3D, totalCells)
	for i := range vertices {
		vertices[i] = Vector3D{Y: -1} // filler far from the component
	}
	assignments := make([]int, totalCells)
	for i := range assignments {
		assignments[i] = -1
	}
	component := OceanComponent{ID: 0, Vertices: make([]int, ringCells)}
	for i := 0; i < ringCells; i++ {
		phi := 2 * math.Pi * float64(i) / float64(ringCells)
		vertices[i] = Vector3D{
			X: math.Sin(angularRadius) * math.Cos(phi),
			Y: math.Cos(angularRadius),
			Z: math.Sin(angularRadius) * math.Sin(phi),
		}
		component.Vertices[i] = i
		assignments[i] = 0
	}
	field := BuildComponentScaleField(vertices, assignments, []OceanComponent{component})
	if field == nil {
		t.Fatal("expected non-nil component scale field")
	}
	return field[0]
}

func TestBuildComponentScaleFieldMatchesAcrossResolution(t *testing.T) {
	// Same physical sea: 100 cells at L5, 400 cells at L6, identical angular radius.
	base := ringComponentScale(t, 10242, 100, 0.3)
	fine := ringComponentScale(t, 40962, 400, 0.3)
	if base <= 0 {
		t.Fatalf("expected mid-size component to keep a positive scale, got %.4f", base)
	}
	if diff := fine - base; diff < -1e-3 || diff > 1e-3 {
		t.Fatalf("expected equivalent component scale across resolutions, got base=%.4f fine=%.4f", base, fine)
	}

	// Tiny sea below the 8-baseline-cell gate: 7 cells at L5 and its 28-cell L6
	// counterpart must both be zeroed (28 raw cells passed the old raw-count gate).
	if got := ringComponentScale(t, 10242, 7, 0.1); got != 0 {
		t.Fatalf("expected sub-threshold L5 component scale 0, got %.4f", got)
	}
	if got := ringComponentScale(t, 40962, 28, 0.1); got != 0 {
		t.Fatalf("expected sub-threshold L6 component scale 0, got %.4f", got)
	}
}

// coastalChainLandDirs builds a chain of vertices along the equator with the
// chain head as the only land cell, and returns the coastline land directions.
func coastalChainLandDirs(totalCells, chainLen int) []Vector3D {
	vertices := make([]Vector3D, totalCells)
	elevation := make([]float64, totalCells)
	for i := range elevation {
		elevation[i] = -1
	}
	const spacing = 0.005
	for i := 0; i < chainLen; i++ {
		angle := float64(i) * spacing
		vertices[i] = Vector3D{X: math.Cos(angle), Z: math.Sin(angle)}
	}
	elevation[0] = 1 // land at the chain head
	adj := makeChainAdjacency(totalCells, chainLen)
	return CalculateCoastlineLandDirs(vertices, elevation, 0, adj)
}

func TestCalculateCoastlineLandDirsBandScalesWithResolution(t *testing.T) {
	// L5 baseline: 3 rings with the historical ramp 1, 2/3, 1/3, then free water.
	base := coastalChainLandDirs(10242, 12)
	wantStrength := []float64{1, 2.0 / 3.0, 1.0 / 3.0}
	for ring, want := range wantStrength {
		got := Length(base[ring+1])
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("expected L5 ring %d constraint strength %.4f, got %.4f", ring+1, want, got)
		}
	}
	if Length(base[4]) != 0 {
		t.Fatalf("expected L5 band to end after ring 3, got strength %.4f at ring 4", Length(base[4]))
	}

	// L6: the same physical band needs 6 graph rings.
	fine := coastalChainLandDirs(40962, 12)
	for ring := 1; ring <= 6; ring++ {
		if Length(fine[ring]) == 0 {
			t.Fatalf("expected L6 coast-parallel band to cover ring %d", ring)
		}
	}
	if Length(fine[7]) != 0 {
		t.Fatalf("expected L6 band to end after ring 6, got strength %.4f at ring 7", Length(fine[7]))
	}
}

// coastBearingVertices returns [water vertex, land neighbor, water neighbor]
// where the land neighbor sits at the given chord spacing from the water
// vertex with the given direction cosine against local east (positive = land
// toward the east), and the water neighbor lies due west.
func coastBearingVertices(spacing, eastCosine float64) []Vector3D {
	v := Vector3D{X: 1}
	east, north := GetTangentVectors(v)
	dir := Add(Scale(east, eastCosine), Scale(north, math.Sqrt(1-eastCosine*eastCosine)))
	land := Normalize(Add(v, Scale(dir, spacing)))
	water := Normalize(Add(v, Scale(east, -spacing)))
	return []Vector3D{v, land, water}
}

func TestCoastBearingTestsAreResolutionIndependent(t *testing.T) {
	elevation := []float64{-1, 1, -1}
	adj := buildTestAdjacency(3, [][]int{{1, 2}, {0}, {0}})
	assignments := []int{0, -1, 0}

	// L5-like (~0.035 rad) and L6-like (~0.0175 rad) neighbor spacings must
	// classify identical bearings identically. With the old unnormalized 0.001
	// dot test, the 0.05-cosine bearing passed at L5 spacing but failed at L6.
	for _, spacing := range []float64{0.035, 0.0175} {
		eastern := coastBearingVertices(spacing, 0.05)
		if !isEasternBoundaryVertex(0, 0, eastern, elevation, 0, adj, assignments) {
			t.Fatalf("expected 0.05-cosine land bearing to read as eastern coast at spacing %.4f", spacing)
		}
		fetch := ComputeEasternBoundaryFetch(eastern, elevation, 0, adj)
		if fetch[2] <= 0.5 {
			t.Fatalf("expected eastern-coast fetch to propagate west at spacing %.4f, got %.3f", spacing, fetch[2])
		}

		grazing := coastBearingVertices(spacing, 0.02)
		if isEasternBoundaryVertex(0, 0, grazing, elevation, 0, adj, assignments) {
			t.Fatalf("expected 0.02-cosine land bearing to stay below the eastern-coast threshold at spacing %.4f", spacing)
		}
		if fetch := ComputeEasternBoundaryFetch(grazing, elevation, 0, adj); fetch[2] != 0 {
			t.Fatalf("expected no eastern-coast fetch for grazing bearing at spacing %.4f, got %.3f", spacing, fetch[2])
		}

		western := coastBearingVertices(spacing, -0.05)
		if intensity := ComputeWesternBoundaryLayer(western, elevation, 0, adj); intensity[0] <= 0 {
			t.Fatalf("expected -0.05-cosine land bearing to form a western boundary layer at spacing %.4f", spacing)
		}
		grazingWest := coastBearingVertices(spacing, -0.02)
		if intensity := ComputeWesternBoundaryLayer(grazingWest, elevation, 0, adj); intensity[0] != 0 {
			t.Fatalf("expected no western boundary layer for grazing bearing at spacing %.4f, got %.3f", spacing, intensity[0])
		}
	}
}
