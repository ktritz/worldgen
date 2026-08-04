package terrain

import (
	"math"
	"testing"
)

func TestCoalesceHotspotIslandsMergesSubSpacingEvents(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: math.Cos(0.006), Y: math.Sin(0.006), Z: 0},
		{X: math.Cos(0.060), Y: math.Sin(0.060), Z: 0},
	}
	islands := []HotspotIsland{
		{CellIndex: 0, Position: sites[0], Age: 0.10, Strength: 0.80},
		{CellIndex: 1, Position: sites[1], Age: 0.20, Strength: 1.00},
		{CellIndex: 2, Position: sites[2], Age: 0.70, Strength: 0.60},
	}

	merged := coalesceHotspotIslands(sites, islands)
	if len(merged) != 2 {
		t.Fatalf("merged island count = %d, want 2", len(merged))
	}
	if merged[0].Strength <= islands[1].Strength {
		t.Fatalf("merged strength = %.3f, want combined complex stronger than max source %.3f", merged[0].Strength, islands[1].Strength)
	}
	if angularDistance(merged[0].Position, sites[0]) > IslandSpacingRadians*0.75 {
		t.Fatalf("merged position moved outside expected cluster radius")
	}
	if merged[1].CellIndex != 2 {
		t.Fatalf("far island cell index = %d, want 2", merged[1].CellIndex)
	}
}
