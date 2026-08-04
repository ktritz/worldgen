package climgen

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"worldgen/icosphere"
	"worldgen/landgen/terrain"
)

type benchMesh struct {
	sites     []Vector3D
	cells     []VoronoiCell
	elevation []float64
	adj       *FlatAdjacency
}

var benchMeshCache = map[int]*benchMesh{}

func buildBenchMesh(tb testing.TB, level int) *benchMesh {
	if m, ok := benchMeshCache[level]; ok {
		return m
	}
	start := time.Now()
	vertices, faces := icosphere.CreateIcosphere(level)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)

	tSites := make([]terrain.Vector3D, len(vertices))
	cSites := make([]Vector3D, len(vertices))
	for i, v := range vertices {
		tSites[i] = terrain.Vector3D{X: v.X, Y: v.Y, Z: v.Z}
		cSites[i] = Vector3D{X: v.X, Y: v.Y, Z: v.Z}
	}
	tCells := make([]terrain.VoronoiCell, len(voronoiCells))
	cCells := make([]VoronoiCell, len(voronoiCells))
	for i, cell := range voronoiCells {
		tCells[i] = terrain.VoronoiCell{
			SiteIndex:           int32(cell.SiteIndex),
			NeighborSiteIndices: append([]int32(nil), cell.NeighborSiteIndices...),
		}
		cCells[i] = VoronoiCell{
			SiteIndex:           int32(cell.SiteIndex),
			NeighborSiteIndices: append([]int32(nil), cell.NeighborSiteIndices...),
		}
	}
	elevation, _, _ := terrain.GeneratePlanetElevationWithDiagnostics(tSites, tCells, 12, 42, 0.30)
	m := &benchMesh{sites: cSites, cells: cCells, elevation: elevation, adj: BuildFlatAdjacency(cCells)}
	benchMeshCache[level] = m
	tb.Logf("mesh level=%d cells=%d built in %s", level, len(cSites), time.Since(start))
	return m
}

func runClimate(tb testing.TB, m *benchMesh) *SeasonalClimateResult {
	seed := int64(42)
	currentSettings := DefaultOceanCurrentSettings()
	currentSettings.Seed = seed
	currentResult, err := GenerateOceanCurrents(m.sites, m.cells, m.elevation, 0.0, currentSettings)
	if err != nil {
		tb.Fatalf("currents: %v", err)
	}
	tempSettings := DefaultTemperatureSettings()
	tempSettings.Seed = seed
	tempSettings.Solar.AxialTilt = 23.5
	tempSettings.Balance.MaxIterations = 500

	seasonalSettings := DefaultSeasonalTemperatureSettings()
	seasonalSettings.NumSeasons = 4
	seasonalSettings.NumCycles = 3
	seasonalSettings.ReferenceEquilibrium = true

	windSettings := DefaultWindSettings()
	windSettings.Seed = seed
	precipSettings := DefaultPrecipitationSettings()

	climate, err := GenerateSeasonalClimate(m.sites, m.elevation, 0.0, m.adj,
		windSettings, currentResult, tempSettings, precipSettings, seasonalSettings)
	if err != nil {
		tb.Fatalf("climate: %v", err)
	}
	return climate
}

// TestProfileClimateStage is a manual timing harness. Run with:
//
//	CLIMPROF_LEVEL=7 go test ./climgen/ -run TestProfileClimateStage -timeout 3h -cpuprofile cpu.out
func TestProfileClimateStage(t *testing.T) {
	lvlStr := os.Getenv("CLIMPROF_LEVEL")
	if lvlStr == "" {
		t.Skip("set CLIMPROF_LEVEL to run")
	}
	level, err := strconv.Atoi(lvlStr)
	if err != nil {
		t.Fatal(err)
	}
	m := buildBenchMesh(t, level)
	start := time.Now()
	res := runClimate(t, m)
	elapsed := time.Since(start)
	fmt.Printf("CLIMATE level=%d cells=%d elapsed=%s snapshots=%d\n", level, len(m.sites), elapsed, len(res.Snapshots))
}
