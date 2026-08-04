package terrain

import (
	"math"
	"testing"

	"worldgen/icosphere"
)

// benchMesh builds a real icosphere mesh at the requested subdivision level and
// synthesizes a plausible land/ocean elevation field so erosion passes see a
// realistic land fraction and coastline topology.
type benchMesh struct {
	sites            []Vector3D
	cells            []VoronoiCell
	elevation        []float64
	rPlate           []int
	plateIsOcean     map[int]bool
	distFromCoast    []float64
	distFromMountain []float64
}

var benchMeshCache = map[int]*benchMesh{}

func buildBenchMesh(level int) *benchMesh {
	if m, ok := benchMeshCache[level]; ok {
		return m
	}
	sites, faces := icosphere.CreateIcosphere(level)
	_, cells := icosphere.GenerateSphericalVoronoi(sites, faces)

	n := len(sites)
	m := &benchMesh{
		sites:            sites,
		cells:            cells,
		elevation:        make([]float64, n),
		rPlate:           make([]int, n),
		plateIsOcean:     map[int]bool{0: false, 1: true},
		distFromCoast:    make([]float64, n),
		distFromMountain: make([]float64, n),
	}

	// Continent field: low-order spherical harmonics give ~30% land with a
	// convoluted coastline (islands, isthmuses) - the case checkDepth cares about.
	for i, s := range sites {
		f := math.Sin(3.1*s.X)*math.Cos(2.7*s.Y) +
			0.6*math.Sin(5.3*s.Z+1.1) +
			0.4*math.Cos(7.9*s.X*s.Y)
		if f > 0.35 {
			m.elevation[i] = 200 + 3000*(f-0.35)
			m.rPlate[i] = 0
		} else {
			m.elevation[i] = -200 - 3000*(0.35-f)
			m.rPlate[i] = 1
		}
		m.distFromCoast[i] = math.Abs(f-0.35) * 20
		m.distFromMountain[i] = math.Abs(f-0.9) * 20
	}
	benchMeshCache[level] = m
	return m
}

func (m *benchMesh) elevationCopy() []float64 {
	out := make([]float64, len(m.elevation))
	copy(out, m.elevation)
	return out
}

func benchLandmassErosion(b *testing.B, level int) {
	m := buildBenchMesh(level)
	b.ReportMetric(float64(len(m.sites)), "cells")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elev := m.elevationCopy()
		ApplyLandmassErosion(m.cells, elev, m.rPlate, m.plateIsOcean, m.distFromCoast, m.distFromMountain)
	}
}

func BenchmarkLandmassErosionL5(b *testing.B) { benchLandmassErosion(b, 5) }
func BenchmarkLandmassErosionL6(b *testing.B) { benchLandmassErosion(b, 6) }
func BenchmarkLandmassErosionL7(b *testing.B) { benchLandmassErosion(b, 7) }

func benchSelectiveErosion(b *testing.B, level int) {
	m := buildBenchMesh(level)
	b.ReportMetric(float64(len(m.sites)), "cells")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elev := m.elevationCopy()
		ApplySelectiveErosion(m.cells, elev, m.rPlate, m.plateIsOcean, 3)
	}
}

func BenchmarkSelectiveErosionL5(b *testing.B) { benchSelectiveErosion(b, 5) }
func BenchmarkSelectiveErosionL6(b *testing.B) { benchSelectiveErosion(b, 6) }
func BenchmarkSelectiveErosionL7(b *testing.B) { benchSelectiveErosion(b, 7) }

// TestReportResolutionScaledCounts prints the derived hop / iteration counts per
// mesh level so the calibration is visible without rerunning the audit.
func TestReportResolutionScaledCounts(t *testing.T) {
	levels := []struct {
		name  string
		cells int
	}{
		{"L4", 2562}, {"L5", 10242}, {"L6", 40962}, {"L7", 163842}, {"L8", 655362},
	}
	for _, lv := range levels {
		t.Logf("%s n=%6d scale=%.4f checkDepth(base=%d)=%d selectiveIters(base=3)=%d",
			lv.name, lv.cells, meshPathCostResolutionScale(lv.cells),
			landmassErosionBaseCheckDepth,
			meshResolutionAdjustedSteps(landmassErosionBaseCheckDepth, lv.cells),
			meshResolutionAdjustedDiffusionIterations(3, lv.cells))
	}
}

func benchFullTerrain(b *testing.B, level int) {
	m := buildBenchMesh(level)
	b.ReportMetric(float64(len(m.sites)), "cells")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GeneratePlanetElevation(m.sites, m.cells, 24, 12345, 0.29)
	}
}

func BenchmarkFullTerrainL6(b *testing.B) { benchFullTerrain(b, 6) }
func BenchmarkFullTerrainL7(b *testing.B) { benchFullTerrain(b, 7) }
