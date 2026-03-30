package terrain

import "testing"

func TestLocalRefinementManagerRefineCell(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0.98, Y: 0.20, Z: 0},
		{X: 0.94, Y: 0.34, Z: 0},
	}
	for i := range sites {
		sites[i] = sites[i].Normalize()
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1}},
	}
	elevation := []float64{900, 500, 100}
	diag := ComputeHydrologyDiagnostics(sites, cells, elevation, 42)
	settings := DefaultLocalRefinementSettings()
	settings.Resolution = 96
	settings.NoiseAmplitude = 0
	settings.ChannelDepthScale = 0

	store := NewLocalRefinementStore(t.TempDir(), 42)
	manager := &LocalRefinementManager{
		Sites:      sites,
		Cells:      cells,
		Elevation:  elevation,
		Hydrology:  diag.Scaffold,
		Refinement: diag.TerrainRefinement,
		Seed:       42,
		Settings:   settings,
		Store:      store,
	}

	result, err := manager.RefineCell(1)
	if err != nil {
		t.Fatalf("refine cell: %v", err)
	}
	if result == nil || result.CenterPatch == nil || result.CenterArtifact == nil {
		t.Fatalf("unexpected refine result: %+v", result)
	}
	if _, err := store.LoadArtifact(1); err != nil {
		t.Fatalf("load saved artifact: %v", err)
	}
}
