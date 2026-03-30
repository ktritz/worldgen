package terrain

import (
	"math"
	"testing"
)

func TestComputeDrainageMetricsDetectsEndorheicLake(t *testing.T) {
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1, 3, 4}},
		{NeighborSiteIndices: []int32{2}},
		{NeighborSiteIndices: []int32{2}},
		{NeighborSiteIndices: []int32{6}},
		{NeighborSiteIndices: []int32{5, 7}},
		{NeighborSiteIndices: []int32{6, 8}},
		{NeighborSiteIndices: []int32{7}},
	}
	elevation := []float64{300, 140, -12, -18, -15, -200, -220, -180, -210}

	var metrics TerrainMetrics
	computeDrainageMetrics(&metrics, cells, elevation)

	if metrics.InlandLakeCoverage <= 0 {
		t.Fatalf("expected inland lake coverage > 0, got %.4f", metrics.InlandLakeCoverage)
	}
	if metrics.FluvialChannelCoverage < 0 {
		t.Fatalf("expected non-negative fluvial channel coverage, got %.3f", metrics.FluvialChannelCoverage)
	}
}

func TestHydrologyScaffoldBuildsBoundaryFlowContracts(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: math.Cos(10 * math.Pi / 180), Y: math.Sin(10 * math.Pi / 180), Z: 0},
		{X: math.Cos(20 * math.Pi / 180), Y: math.Sin(20 * math.Pi / 180), Z: 0},
		{X: math.Cos(30 * math.Pi / 180), Y: math.Sin(30 * math.Pi / 180), Z: 0},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1, 3}},
		{NeighborSiteIndices: []int32{2}},
	}
	elevation := []float64{900, 600, 200, -50}

	diag := ComputeHydrologyDiagnostics(sites, cells, elevation, 42)
	if diag.Scaffold == nil {
		t.Fatalf("expected scaffold to be present")
	}
	if diag.TerrainRefinement == nil {
		t.Fatalf("expected terrain refinement scaffold to be present")
	}
	flow := diag.Scaffold.BoundaryFlow
	if len(flow) != len(elevation) {
		t.Fatalf("boundary flow count = %d, want %d", len(flow), len(elevation))
	}
	if flow[0].OutflowNeighbor != 1 {
		t.Fatalf("cell 0 outflow neighbor = %d, want 1", flow[0].OutflowNeighbor)
	}
	if flow[1].OutflowNeighbor != 2 {
		t.Fatalf("cell 1 outflow neighbor = %d, want 2", flow[1].OutflowNeighbor)
	}
	if len(flow[1].InflowNeighbors) != 1 || flow[1].InflowNeighbors[0] != 0 {
		t.Fatalf("cell 1 inflows = %v, want [0]", flow[1].InflowNeighbors)
	}
	if len(flow[2].InflowNeighbors) != 1 || flow[2].InflowNeighbors[0] != 1 {
		t.Fatalf("cell 2 inflows = %v, want [1]", flow[2].InflowNeighbors)
	}
	if flow[2].OutflowNeighbor != 3 {
		t.Fatalf("cell 2 outflow neighbor = %d, want 3", flow[2].OutflowNeighbor)
	}
	if flow[3].OutflowNeighbor != -1 {
		t.Fatalf("ocean outlet cell outflow neighbor = %d, want -1", flow[3].OutflowNeighbor)
	}
	if flow[2].OutflowStrength <= flow[1].OutflowStrength {
		t.Fatalf("expected downstream crossing strength to accumulate upstream discharge: cell2=%.3f cell1=%.3f",
			flow[2].OutflowStrength, flow[1].OutflowStrength)
	}
	classes := diag.Scaffold.CellClass
	if classes[0] != "headwater" {
		t.Fatalf("cell 0 class = %q, want headwater", classes[0])
	}
	if classes[1] != "hillslope" && classes[1] != "trunk" {
		t.Fatalf("cell 1 class = %q, want hillslope or trunk", classes[1])
	}
	if classes[2] != "coast_outlet" {
		t.Fatalf("cell 2 class = %q, want coast_outlet", classes[2])
	}
	if diag.Scaffold.MaxOutflows[2] != 1 {
		t.Fatalf("cell 2 max outflows = %d, want 1", diag.Scaffold.MaxOutflows[2])
	}
	side := diag.Scaffold.BoundarySideFlow
	if len(side) != len(elevation) {
		t.Fatalf("boundary side flow count = %d, want %d", len(side), len(elevation))
	}
	sumOut := 0.0
	for _, sector := range side[1] {
		sumOut += sector.OutflowStrength
	}
	if math.Abs(sumOut-flow[1].OutflowStrength) > 1e-9 {
		t.Fatalf("aggregated outflow %.6f != direct outflow %.6f", sumOut, flow[1].OutflowStrength)
	}
	sumIn := 0.0
	for _, sector := range side[1] {
		sumIn += sector.InflowStrength
	}
	if math.Abs(sumIn-flow[1].InflowStrength[0]) > 1e-9 {
		t.Fatalf("aggregated inflow %.6f != direct inflow %.6f", sumIn, flow[1].InflowStrength[0])
	}
	refine := diag.TerrainRefinement.Cells
	if len(refine) != len(elevation) {
		t.Fatalf("terrain refinement cell count = %d, want %d", len(refine), len(elevation))
	}
	if refine[0].ChannelBearingDeg == 0 && flow[0].OutflowBearingDeg != 0 {
		t.Fatalf("expected terrain refinement to carry channel bearing for cell 0")
	}
	if refine[2].LocalRelief <= 0 {
		t.Fatalf("expected coast outlet cell to report local relief > 0, got %.3f", refine[2].LocalRelief)
	}
	if len(refine[1].Boundary) != 2 {
		t.Fatalf("expected cell 1 to have 2 terrain boundary anchors, got %d", len(refine[1].Boundary))
	}
}

func TestComputeHydrologyDiagnosticsFromRunoffUsesProvidedRunoff(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: math.Cos(10 * math.Pi / 180), Y: math.Sin(10 * math.Pi / 180), Z: 0},
		{X: math.Cos(20 * math.Pi / 180), Y: math.Sin(20 * math.Pi / 180), Z: 0},
		{X: math.Cos(30 * math.Pi / 180), Y: math.Sin(30 * math.Pi / 180), Z: 0},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1, 3}},
		{NeighborSiteIndices: []int32{2}},
	}
	elevation := []float64{1200, 600, 150, -20}
	runoff := []float64{2.5, 1.5, 0.5, 0}

	diag := ComputeHydrologyDiagnosticsFromRunoff(sites, cells, elevation, runoff)
	if diag.Scaffold == nil {
		t.Fatalf("expected scaffold from runoff diagnostics")
	}
	got := diag.Scaffold.Runoff
	if len(got) != len(runoff) {
		t.Fatalf("runoff len = %d, want %d", len(got), len(runoff))
	}
	for i := range runoff {
		if math.Abs(got[i]-runoff[i]) > 1e-9 {
			t.Fatalf("runoff[%d] = %.6f, want %.6f", i, got[i], runoff[i])
		}
	}
	if diag.Scaffold.Accumulation[2] <= runoff[2] {
		t.Fatalf("expected downstream accumulation %.3f to exceed local runoff %.3f", diag.Scaffold.Accumulation[2], runoff[2])
	}
}

func TestHydrologyChannelCoverageIsStableAcrossRunoffUnits(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: math.Cos(10 * math.Pi / 180), Y: math.Sin(10 * math.Pi / 180), Z: 0},
		{X: math.Cos(20 * math.Pi / 180), Y: math.Sin(20 * math.Pi / 180), Z: 0},
		{X: math.Cos(30 * math.Pi / 180), Y: math.Sin(30 * math.Pi / 180), Z: 0},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1, 3}},
		{NeighborSiteIndices: []int32{2}},
	}
	elevation := []float64{1200, 600, 150, -20}
	unitRunoff := []float64{1.5, 0.8, 0.4, 0}
	cmRunoff := []float64{150, 80, 40, 0}

	unitDiag := ComputeHydrologyDiagnosticsFromRunoff(sites, cells, elevation, unitRunoff)
	cmDiag := ComputeHydrologyDiagnosticsFromRunoff(sites, cells, elevation, cmRunoff)

	if math.Abs(unitDiag.FluvialChannelCoverage-cmDiag.FluvialChannelCoverage) > 1e-9 {
		t.Fatalf("channel coverage changed with runoff units: unit=%.6f cm=%.6f",
			unitDiag.FluvialChannelCoverage, cmDiag.FluvialChannelCoverage)
	}
}
