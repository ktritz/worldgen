package terrain

import (
	"math"
	"testing"
)

func TestDetectChainBendIgnoresSmoothCurvature(t *testing.T) {
	sites := []Vector3D{
		lonLatPoint(0, 0),
		lonLatPoint(8, 2),
		lonLatPoint(16, 5),
		lonLatPoint(24, 9),
		lonLatPoint(32, 14),
	}

	chain := HotspotChain{
		IsOceanic: true,
		Islands: []HotspotIsland{
			{CellIndex: 0},
			{CellIndex: 1},
			{CellIndex: 2},
			{CellIndex: 3},
			{CellIndex: 4},
		},
	}

	bent, _ := detectChainBend(sites, chain)
	if bent {
		t.Fatalf("expected smooth curvature to avoid bend classification")
	}
}

func TestDetectChainBendFindsPivotedReorientation(t *testing.T) {
	sites := []Vector3D{
		lonLatPoint(0, 0),
		lonLatPoint(8, 0),
		lonLatPoint(16, 0),
		lonLatPoint(20, 6),
		lonLatPoint(24, 12),
		lonLatPoint(28, 18),
	}

	chain := HotspotChain{
		IsOceanic: true,
		Islands: []HotspotIsland{
			{CellIndex: 0},
			{CellIndex: 1},
			{CellIndex: 2},
			{CellIndex: 3},
			{CellIndex: 4},
			{CellIndex: 5},
		},
	}

	bent, ok := detectChainBend(sites, chain)
	if !ok {
		t.Fatalf("expected pivoted chain to be eligible for bend detection")
	}
	if !bent {
		t.Fatalf("expected pivoted chain to count as bent")
	}
}

func TestComputeMetricsWithHotspotsPopulatesTrackMetrics(t *testing.T) {
	sites := []Vector3D{
		lonLatPoint(0, 0),
		lonLatPoint(7, 0),
		lonLatPoint(16, 0),
		lonLatPoint(21, 6),
		lonLatPoint(28, 12),
		lonLatPoint(37, 17),
		{X: 0, Y: 0, Z: 1},
	}

	elevation := []float64{100, 80, 50, 30, 10, -50, -4000}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1, 3}},
		{NeighborSiteIndices: []int32{2, 4}},
		{NeighborSiteIndices: []int32{3, 5}},
		{NeighborSiteIndices: []int32{4}},
		{NeighborSiteIndices: []int32{}},
	}

	chains := []HotspotChain{
		{
			IsOceanic: true,
			Islands: []HotspotIsland{
				{CellIndex: 0, Age: 0.0, Strength: 1.2},
				{CellIndex: 1, Age: 0.2, Strength: 0.8},
				{CellIndex: 2, Age: 0.5, Strength: 1.0},
				{CellIndex: 3, Age: 0.8, Strength: 0.6},
				{CellIndex: 4, Age: 1.0, Strength: 0.9},
				{CellIndex: 5, Age: 1.4, Strength: 0.5},
			},
		},
	}

	metrics := ComputeMetricsWithHotspots(sites, cells, elevation, chains)
	if metrics.HotspotChainCount != 1 {
		t.Fatalf("expected 1 hotspot chain, got %d", metrics.HotspotChainCount)
	}
	if metrics.HotspotSpacingCV <= 0 {
		t.Fatalf("expected spacing CV to be populated, got %.3f", metrics.HotspotSpacingCV)
	}
	if metrics.HotspotBurstiness <= 1 {
		t.Fatalf("expected burstiness > 1, got %.3f", metrics.HotspotBurstiness)
	}
	if metrics.HotspotBendFraction <= 0 {
		t.Fatalf("expected bend fraction > 0, got %.3f", metrics.HotspotBendFraction)
	}
}

func lonLatPoint(lonDeg, latDeg float64) Vector3D {
	lon := lonDeg * math.Pi / 180.0
	lat := latDeg * math.Pi / 180.0
	cosLat := math.Cos(lat)
	return Vector3D{
		X: cosLat * math.Cos(lon),
		Y: cosLat * math.Sin(lon),
		Z: math.Sin(lat),
	}
}
