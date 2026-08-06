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

// TestHydrologyChannelThresholdFollowsAccumulationHierarchy pins the threshold
// to the accumulation hierarchy at a FIXED mesh resolution: doubling every
// accumulation value must double the threshold. Resolution behaviour is a
// separate axis and is covered by
// TestHydrologyChannelInitiationScalesWithChannelLength.
func TestHydrologyChannelThresholdFollowsAccumulationHierarchy(t *testing.T) {
	baseElevation := make([]float64, 10000)
	baseRunoff := make([]float64, len(baseElevation))
	baseAccumulation := make([]float64, len(baseElevation))
	doubledAccumulation := make([]float64, len(baseElevation))
	for i := range baseElevation {
		baseElevation[i] = 100
		baseRunoff[i] = 1
		baseAccumulation[i] = 1 + 99*float64(i)/float64(len(baseElevation)-1)
		doubledAccumulation[i] = 2 * baseAccumulation[i]
	}

	baseThreshold := hydrologyChannelThreshold(baseElevation, baseRunoff, baseAccumulation, len(baseElevation), 1)
	doubledThreshold := hydrologyChannelThreshold(baseElevation, baseRunoff, doubledAccumulation, len(baseElevation), 1)
	ratio := doubledThreshold / baseThreshold
	if ratio < 1.95 || ratio > 2.05 {
		t.Fatalf("expected channel threshold to follow linear accumulation hierarchy, got ratio %.3f", ratio)
	}
}

// buildSyntheticDrainage lays the SAME physical drainage network onto a mesh of
// meshCells cells, sampled by a width x width grid of land cells.
//
// The network is a trunk (column x == 0) fed by eight lateral tributaries whose
// spacing is a fixed FRACTION of the grid (width/8 rows), i.e. a fixed physical
// distance: refining the mesh resolves the same channels more finely, it does
// not invent new ones. Accumulation is the upstream cell count, so a cell's
// physical contributing area is accumulation / width^2 and is a function of
// position in the unit square alone — identical at both resolutions.
func buildSyntheticDrainage(meshCells, width int) (elevation, runoff, accumulation []float64, landCount int) {
	elevation = make([]float64, meshCells)
	runoff = make([]float64, meshCells)
	accumulation = make([]float64, meshCells)
	rowSpacing := width / 8
	landCount = width * width
	for y := 0; y < width; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			elevation[idx] = 100
			runoff[idx] = 1
			switch {
			case x == 0:
				// Trunk: every tributary strip at or upstream of this row.
				strips := y/rowSpacing + 1
				accumulation[idx] = float64(strips * width * rowSpacing)
			case y%rowSpacing == 0:
				// Tributary: its own strip plus every strip further from the trunk.
				accumulation[idx] = float64((width - x) * rowSpacing)
			default:
				// Hillslope: only the cells above it inside its own strip.
				accumulation[idx] = float64(rowSpacing - y%rowSpacing)
			}
		}
	}
	return elevation, runoff, accumulation, landCount
}

func countChannelCells(elevation, accumulation []float64, threshold float64) int {
	count := 0
	for i, elev := range elevation {
		if elev > 0 && accumulation[i] >= threshold {
			count++
		}
	}
	return count
}

// TestHydrologyChannelInitiationScalesWithChannelLength is the regression test
// for river extent scaling with mesh AREA. The same physical channel network is
// sampled at L5 (10242 cells) and L6 (40962 cells): land cells quadruple, but
// the channel network is one-dimensional, so the number of cells on it must
// only double. A fixed upper-tail percentile pins the channel FRACTION and
// therefore quadruples the channel set; a critical drainage area does not.
func TestHydrologyChannelInitiationScalesWithChannelLength(t *testing.T) {
	const (
		baselineMeshCells = 10242
		refinedMeshCells  = 40962
	)

	coarseElevation, coarseRunoff, coarseAccumulation, coarseLand := buildSyntheticDrainage(baselineMeshCells, 64)
	fineElevation, fineRunoff, fineAccumulation, fineLand := buildSyntheticDrainage(refinedMeshCells, 128)

	if got := float64(fineLand) / float64(coarseLand); math.Abs(got-4) > 1e-9 {
		t.Fatalf("fixture land cells must scale with mesh area, got %.3f", got)
	}

	coarseThreshold := hydrologyChannelThreshold(coarseElevation, coarseRunoff, coarseAccumulation, coarseLand, 1)
	fineThreshold := hydrologyChannelThreshold(fineElevation, fineRunoff, fineAccumulation, fineLand, 1)

	coarseChannels := countChannelCells(coarseElevation, coarseAccumulation, coarseThreshold)
	fineChannels := countChannelCells(fineElevation, fineAccumulation, fineThreshold)
	if coarseChannels == 0 {
		t.Fatalf("expected a non-empty baseline channel set")
	}

	// Baseline behaviour is unchanged: the L5 channel set is still the P93.5
	// upper tail of land accumulation.
	coarseFraction := float64(coarseChannels) / float64(coarseLand)
	if coarseFraction < 0.055 || coarseFraction > 0.075 {
		t.Fatalf("baseline channel fraction = %.4f, want ~0.065", coarseFraction)
	}

	countRatio := float64(fineChannels) / float64(coarseChannels)
	if countRatio < 1.8 || countRatio > 2.2 {
		t.Fatalf("channel cell count ratio L6/L5 = %.3f (coarse=%d fine=%d), want ~2.0 (linear), not ~4.0 (area)",
			countRatio, coarseChannels, fineChannels)
	}

	// The same channel count is also the same physical contributing area: the
	// upstream cell count that marks a channel must quadruple when cells become
	// four times smaller.
	thresholdRatio := fineThreshold / coarseThreshold
	if thresholdRatio < 3.6 || thresholdRatio > 4.4 {
		t.Fatalf("channel accumulation threshold ratio L6/L5 = %.3f, want ~4.0 (a fixed physical area)", thresholdRatio)
	}
}
