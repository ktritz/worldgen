package terrain

import "testing"

func TestBuildLocalRefinementPatchZeroNoiseTracksBoundaryAnchors(t *testing.T) {
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
	if diag.Scaffold == nil || diag.TerrainRefinement == nil {
		t.Fatalf("expected scaffolds to be present")
	}

	settings := DefaultLocalRefinementSettings()
	settings.Resolution = 96
	settings.Radius = 1
	settings.NoiseAmplitude = 0
	settings.ChannelDepthScale = 0

	patch, err := BuildLocalRefinementPatch(sites, cells, elevation, diag.Scaffold, diag.TerrainRefinement, 1, 42, settings, nil)
	if err != nil {
		t.Fatalf("build local patch: %v", err)
	}
	if patch.Debug.NumBoundarySamples == 0 {
		t.Fatalf("expected boundary samples")
	}
	if patch.Debug.MeanBoundaryMismatch > 220 {
		t.Fatalf("mean boundary mismatch %.2f too large for zero-noise prototype", patch.Debug.MeanBoundaryMismatch)
	}
	if patch.Debug.MaxBoundaryMismatch > 420 {
		t.Fatalf("max boundary mismatch %.2f too large for zero-noise prototype", patch.Debug.MaxBoundaryMismatch)
	}
}
