package climgen

import (
	"math"
	"testing"

	"worldgen/icosphere"
)

func TestGenerateWindFieldRemainsTangentToSphere(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(4)

	settings := DefaultWindSettings()
	settings.Seed = 42

	result, err := GenerateWindField(vertices, elevation, 0.0, adj, settings)
	if err != nil {
		t.Fatalf("GenerateWindField failed: %v", err)
	}

	for i, wind := range result.SurfaceWind {
		radial := math.Abs(Dot(vertices[i], wind))
		if radial > 1e-6 {
			t.Fatalf("wind at %d has radial component %.6g", i, radial)
		}
	}

	for i, wind := range result.MarineWind {
		radial := math.Abs(Dot(vertices[i], wind))
		if radial > 1e-6 {
			t.Fatalf("marine wind at %d has radial component %.6g", i, radial)
		}
	}
}

func TestGenerateWindFieldProducesExpectedZonalBands(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(4)

	settings := DefaultWindSettings()
	settings.Seed = 42
	settings.Orographic.DeflectionStrength = 0

	result, err := GenerateWindField(vertices, elevation, 0.0, adj, settings)
	if err != nil {
		t.Fatalf("GenerateWindField failed: %v", err)
	}

	type bandExpectation struct {
		name      string
		minAbsLat float64
		maxAbsLat float64
		wantEast  float64
	}

	bands := []bandExpectation{
		{name: "trades", minAbsLat: 8, maxAbsLat: 25, wantEast: -1},
		{name: "westerlies", minAbsLat: 35, maxAbsLat: 55, wantEast: 1},
		{name: "polar easterlies", minAbsLat: 70, maxAbsLat: 82, wantEast: -1},
	}

	for _, band := range bands {
		eastSum := 0.0
		count := 0
		for i, vertex := range vertices {
			latDeg := math.Asin(vertex.Y) * 180.0 / math.Pi
			absLat := math.Abs(latDeg)
			if absLat < band.minAbsLat || absLat >= band.maxAbsLat {
				continue
			}
			east, _ := GetTangentVectors(vertex)
			eastSum += Dot(result.SurfaceWind[i], east)
			count++
		}
		if count == 0 {
			t.Fatalf("no samples found for band %s", band.name)
		}
		meanEast := eastSum / float64(count)
		if meanEast*band.wantEast <= 0.01 {
			t.Fatalf("band %s has wrong zonal sign: mean east component %.4f", band.name, meanEast)
		}
	}
}

func TestGenerateWindFieldShowsTradeWindConvergence(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(4)

	settings := DefaultWindSettings()
	settings.Seed = 42
	settings.Orographic.DeflectionStrength = 0

	result, err := GenerateWindField(vertices, elevation, 0.0, adj, settings)
	if err != nil {
		t.Fatalf("GenerateWindField failed: %v", err)
	}

	northTrades := meanMeridionalComponent(vertices, result.SurfaceWind, 8, 20, true)
	southTrades := meanMeridionalComponent(vertices, result.SurfaceWind, 8, 20, false)
	northWesterlies := meanMeridionalComponent(vertices, result.SurfaceWind, 35, 55, true)
	southWesterlies := meanMeridionalComponent(vertices, result.SurfaceWind, 35, 55, false)

	if northTrades >= -0.01 {
		t.Fatalf("expected NH trades to be equatorward, got %.4f", northTrades)
	}
	if southTrades <= 0.01 {
		t.Fatalf("expected SH trades to be equatorward, got %.4f", southTrades)
	}
	if northWesterlies <= 0.01 {
		t.Fatalf("expected NH westerlies to be poleward, got %.4f", northWesterlies)
	}
	if southWesterlies >= -0.01 {
		t.Fatalf("expected SH westerlies to be poleward, got %.4f", southWesterlies)
	}
}

func TestGenerateWindFieldSeparatesMarineAndSurfaceWind(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(4)
	for i, v := range vertices {
		if v.X > 0.65 && math.Abs(v.Z) < 0.25 {
			elevation[i] = 2200
		}
	}

	settings := DefaultWindSettings()
	settings.Seed = 42

	result, err := GenerateWindField(vertices, elevation, 0.0, adj, settings)
	if err != nil {
		t.Fatalf("GenerateWindField failed: %v", err)
	}

	diffCount := 0
	for i := range vertices {
		if elevation[i] >= 0 {
			continue
		}
		diff := Length(Sub(result.SurfaceWind[i], result.MarineWind[i]))
		if diff > 1e-4 {
			diffCount++
		}
	}

	if diffCount == 0 {
		t.Fatalf("expected marine wind to differ from terrain-perturbed surface wind")
	}
}

func buildWindTestWorld(level int) ([]Vector3D, []VoronoiCell, []float64, *FlatAdjacency) {
	rawVertices, faces := icosphere.CreateIcosphere(level)
	_, rawCells := icosphere.GenerateSphericalVoronoi(rawVertices, faces)

	vertices := make([]Vector3D, len(rawVertices))
	for i, v := range rawVertices {
		vertices[i] = Vector3D{X: v.X, Y: v.Y, Z: v.Z}
	}

	cells := make([]VoronoiCell, len(rawCells))
	for i, cell := range rawCells {
		cells[i] = VoronoiCell{
			SiteIndex:           int32(cell.SiteIndex),
			NeighborSiteIndices: make([]int32, len(cell.NeighborSiteIndices)),
		}
		copy(cells[i].NeighborSiteIndices, cell.NeighborSiteIndices)
	}

	elevation := make([]float64, len(vertices))
	for i := range elevation {
		elevation[i] = -1000
	}

	return vertices, cells, elevation, BuildFlatAdjacency(cells)
}

func meanMeridionalComponent(
	vertices []Vector3D,
	wind []Vector3D,
	minAbsLat float64,
	maxAbsLat float64,
	northernHemisphere bool,
) float64 {
	sum := 0.0
	count := 0
	for i, vertex := range vertices {
		latDeg := math.Asin(vertex.Y) * 180.0 / math.Pi
		if northernHemisphere && latDeg <= 0 {
			continue
		}
		if !northernHemisphere && latDeg >= 0 {
			continue
		}
		absLat := math.Abs(latDeg)
		if absLat < minAbsLat || absLat >= maxAbsLat {
			continue
		}

		_, north := GetTangentVectors(vertex)
		sum += Dot(wind[i], north)
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}
