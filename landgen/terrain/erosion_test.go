package terrain

import (
	"math"
	"testing"
)

func TestComputeDrainageReceiversRoutesDownhill(t *testing.T) {
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1, 3}},
		{NeighborSiteIndices: []int32{2}},
	}
	elevation := []float64{900, 600, 200, -50}

	receivers := ComputeDrainageReceivers(cells, elevation)
	want := []int{1, 2, 3, -1}
	for i, expected := range want {
		if receivers[i] != expected {
			t.Fatalf("receiver[%d] = %d, want %d", i, receivers[i], expected)
		}
	}
}

func TestComputeLongTermRunoffProxyKeepsSubtropicsDrierThanTropics(t *testing.T) {
	tropical := Vector3D{X: math.Cos(5 * math.Pi / 180), Y: 0, Z: math.Sin(5 * math.Pi / 180)}
	subtropical := Vector3D{X: math.Cos(25 * math.Pi / 180), Y: 0, Z: math.Sin(25 * math.Pi / 180)}
	sites := []Vector3D{tropical, subtropical}
	elevation := []float64{900, 900}
	distFromCoast := []float64{2, 2}

	runoff := ComputeLongTermRunoffProxy(sites, elevation, distFromCoast, 10, 42)
	if runoff[0] <= runoff[1] {
		t.Fatalf("expected tropical runoff %.3f to exceed subtropical runoff %.3f", runoff[0], runoff[1])
	}
}

func TestBreachDrainageSinksCutsShallowOutlet(t *testing.T) {
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1}},
	}
	elevation := []float64{100, 120, 20}

	receivers := ComputeDrainageReceivers(cells, elevation)
	if receivers[0] != -1 {
		t.Fatalf("expected initial sink at cell 0, got receiver %d", receivers[0])
	}

	breached := BreachDrainageSinks(cells, elevation, receivers, 40)
	if breached == 0 {
		t.Fatalf("expected sink breach to modify terrain")
	}

	receivers = ComputeDrainageReceivers(cells, elevation)
	if receivers[0] != 1 {
		t.Fatalf("expected breached sink to drain to cell 1, got %d", receivers[0])
	}
	if elevation[1] >= 100 {
		t.Fatalf("expected outlet saddle to be lowered below sink elevation, got %.1f", elevation[1])
	}
}

func TestApplyFluvialErosionIncisesMainStem(t *testing.T) {
	sites := []Vector3D{
		{X: math.Cos(5 * math.Pi / 180), Y: 0, Z: math.Sin(5 * math.Pi / 180)},
		{X: math.Cos(5 * math.Pi / 180), Y: 0.1, Z: math.Sin(5 * math.Pi / 180)},
		{X: math.Cos(5 * math.Pi / 180), Y: 0.2, Z: math.Sin(5 * math.Pi / 180)},
		{X: math.Cos(5 * math.Pi / 180), Y: 0.3, Z: math.Sin(5 * math.Pi / 180)},
	}
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1, 3}},
		{NeighborSiteIndices: []int32{2}},
	}
	elevation := []float64{1200, 800, 300, -20}
	distFromCoast := []float64{8, 5, 2, 0}

	before := append([]float64(nil), elevation...)
	ApplyFluvialErosion(sites, cells, elevation, distFromCoast, 8, 42)

	if elevation[0] >= before[0] || elevation[1] >= before[1] || elevation[2] >= before[2] {
		t.Fatalf("expected fluvial incision to lower upstream cells, before=%v after=%v", before, elevation)
	}
	if elevation[3] != before[3] {
		t.Fatalf("ocean outlet should remain unchanged, before=%.1f after=%.1f", before[3], elevation[3])
	}
}

// landmassErosionChainFixture builds a 1-D chain of 30 connected cells embedded in
// a mesh of cellCount cells. Cells [0, landCells) are 5000 m land; the rest ocean.
func landmassErosionChainFixture(cellCount, landCells int) ([]VoronoiCell, []float64, []int, []float64, []float64) {
	cells := make([]VoronoiCell, cellCount)
	elevation := make([]float64, cellCount)
	rPlate := make([]int, cellCount)
	distFromCoast := make([]float64, cellCount)
	distFromMountain := make([]float64, cellCount)
	for i := range cells {
		cells[i].SiteIndex = int32(i)
		elevation[i] = -50
		distFromMountain[i] = math.Inf(1)
	}
	for i := 0; i < landCells; i++ {
		elevation[i] = 5000
	}
	for i := 0; i < 30; i++ {
		if i > 0 {
			cells[i].NeighborSiteIndices = append(cells[i].NeighborSiteIndices, int32(i-1))
		}
		if i < 29 {
			cells[i].NeighborSiteIndices = append(cells[i].NeighborSiteIndices, int32(i+1))
		}
	}
	return cells, elevation, rPlate, distFromCoast, distFromMountain
}

func TestLandmassErosionScalesOceanRadiusByMeshResolution(t *testing.T) {
	// checkDepth is 4 hops at the L5 baseline and 8 at L6, so a 6-cell land chain
	// is entirely inside the L5 radius but straddles the coast at L6. Identical
	// physical geometry must therefore produce the same cap decision only when the
	// radius scales with resolution; a fixed hop count would not see the ocean.
	const landCells = 6

	baseCells, baseElev, basePlate, baseCoast, baseMountain := landmassErosionChainFixture(10242, landCells)
	ApplyLandmassErosion(baseCells, baseElev, basePlate, map[int]bool{0: false}, baseCoast, baseMountain)
	if baseElev[0] < 5000 {
		t.Fatalf("L5 radius should stay inside the land chain and leave the peak uncapped, got %.1f", baseElev[0])
	}

	fineCells, fineElev, finePlate, fineCoast, fineMountain := landmassErosionChainFixture(40962, landCells)
	ApplyLandmassErosion(fineCells, fineElev, finePlate, map[int]bool{0: false}, fineCoast, fineMountain)
	if fineElev[0] >= 5000 {
		t.Fatalf("expected scaled landmass radius to see nearby ocean and cap coastal range, got %.1f", fineElev[0])
	}
}

func TestLandmassErosionCheckDepthMatchesPhysicalRadius(t *testing.T) {
	// The base constant is expressed in L5 hops; ~16 hops at L7 reproduces the
	// radius the original hardcoded 15 was calibrated against.
	want := map[int]int{10242: 4, 40962: 8, 163842: 16, 655362: 32}
	for cellCount, expected := range want {
		got := meshResolutionAdjustedSteps(landmassErosionBaseCheckDepth, cellCount)
		if got != expected {
			t.Errorf("checkDepth at %d cells = %d, want %d", cellCount, got, expected)
		}
	}
}

func TestFluvialChannelIncisionUsesPhysicalAccumulationAndSlope(t *testing.T) {
	baseCells := make([]VoronoiCell, 10242)
	baseCells[0].NeighborSiteIndices = []int32{1}
	baseCells[1].NeighborSiteIndices = []int32{0}
	baseElevation := make([]float64, 10242)
	baseElevation[0] = 1000
	baseElevation[1] = 900
	baseReceivers := make([]int, 10242)
	for i := range baseReceivers {
		baseReceivers[i] = -1
	}
	baseReceivers[0] = 1
	baseRunoff := make([]float64, 10242)
	baseRunoff[0] = 1
	baseAccumulation := make([]float64, 10242)
	baseAccumulation[0] = 20

	fineCells := make([]VoronoiCell, 40962)
	fineCells[0].NeighborSiteIndices = []int32{1}
	fineCells[1].NeighborSiteIndices = []int32{0}
	fineElevation := make([]float64, 40962)
	fineElevation[0] = 1000
	fineElevation[1] = 950
	fineReceivers := make([]int, 40962)
	for i := range fineReceivers {
		fineReceivers[i] = -1
	}
	fineReceivers[0] = 1
	fineRunoff := make([]float64, 40962)
	fineRunoff[0] = 1
	fineAccumulation := make([]float64, 40962)
	fineAccumulation[0] = 80

	carveFluvialChannels(baseCells, baseElevation, baseReceivers, baseRunoff, scaleFlowAccumulationForMesh(baseAccumulation, len(baseCells)))
	carveFluvialChannels(fineCells, fineElevation, fineReceivers, fineRunoff, scaleFlowAccumulationForMesh(fineAccumulation, len(fineCells)))

	baseIncision := 1000 - baseElevation[0]
	fineIncision := 1000 - fineElevation[0]
	if math.Abs(baseIncision-fineIncision) > 0.01 {
		t.Fatalf("expected equivalent physical catchments to incise equally, base=%.3f fine=%.3f", baseIncision, fineIncision)
	}
}

func TestFloodplainSpreadUsesPhysicalRadius(t *testing.T) {
	cellCount := 40962
	cells := make([]VoronoiCell, cellCount)
	elevation := make([]float64, cellCount)
	buffer := make([]float64, cellCount)
	for i := range cells {
		cells[i].SiteIndex = int32(i)
	}
	cells[0].NeighborSiteIndices = []int32{1}
	cells[1].NeighborSiteIndices = []int32{0, 2}
	cells[2].NeighborSiteIndices = []int32{1}
	elevation[0] = 80
	elevation[1] = 80
	elevation[2] = 80
	copy(buffer, elevation)

	spreadFloodplainDeposit(cells, elevation, buffer, 0, 10)
	if buffer[2] <= elevation[2] {
		t.Fatalf("expected refined floodplain spread to reach physical one-step cell 2, before=%.1f after=%.1f", elevation[2], buffer[2])
	}
}

func TestApplyPostDetailDrainageConditioningBreachesShallowNoiseTrap(t *testing.T) {
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1}},
		{NeighborSiteIndices: []int32{0, 2}},
		{NeighborSiteIndices: []int32{1}},
	}
	elevation := []float64{100, 120, 20}

	receivers := ComputeDrainageReceivers(cells, elevation)
	if receivers[0] != -1 {
		t.Fatalf("expected initial shallow trap at cell 0, receivers=%v", receivers)
	}

	breached := ApplyPostDetailDrainageConditioning(cells, elevation)
	if breached == 0 {
		t.Fatalf("expected post-detail conditioning to breach the shallow trap")
	}

	receivers = ComputeDrainageReceivers(cells, elevation)
	if receivers[0] != 1 {
		t.Fatalf("expected conditioned terrain to restore drainage from cell 0, receivers=%v elevation=%v", receivers, elevation)
	}
}

// breachFixture builds a sink at cell 0 whose only outlet saddle (cell 1) sits
// `rise` metres above it, draining onward through cell 2 to ocean cell 3. The
// remaining cells (up to total) are inert ocean so len(cells) sets the mesh
// resolution seen by BreachDrainageSinks.
func breachFixture(total int, rise float64) ([]VoronoiCell, []float64) {
	cells := make([]VoronoiCell, total)
	elevation := make([]float64, total)
	cells[0] = VoronoiCell{NeighborSiteIndices: []int32{1}}
	cells[1] = VoronoiCell{NeighborSiteIndices: []int32{0, 2}}
	cells[2] = VoronoiCell{NeighborSiteIndices: []int32{1, 3}}
	cells[3] = VoronoiCell{NeighborSiteIndices: []int32{2}}
	elevation[0] = 100
	elevation[1] = 100 + rise
	elevation[2] = 60
	elevation[3] = -10
	return cells, elevation
}

// The maxRise threshold and carve depths are per-hop metre comparisons, so
// they must shrink with cell size: a saddle breached at baseline resolution
// represents a 4x steeper physical slope at L7 and must survive there.
func TestBreachDrainageSinksThresholdScalesWithMeshResolution(t *testing.T) {
	const l7Cells = 163842

	// Baseline-scale mesh (stepScale clamps to 1): rise 100 < 220 is breached,
	// with the full 6m spillway carve.
	cells, elevation := breachFixture(4, 100)
	receivers := ComputeDrainageReceivers(cells, elevation)
	if breached := BreachDrainageSinks(cells, elevation, receivers, 220); breached == 0 {
		t.Fatal("expected baseline-resolution breach of 100m saddle with maxRise 220")
	}
	if math.Abs(elevation[1]-94) > 1e-9 {
		t.Fatalf("baseline spillway carve: elevation[1] = %.6f, want 94 (100 - 6m carve)", elevation[1])
	}

	// L7 mesh: effective threshold is 220 * stepScale (~55m), so the same 100m
	// saddle is a legitimate physical barrier and must NOT be breached.
	cells, elevation = breachFixture(l7Cells, 100)
	receivers = ComputeDrainageReceivers(cells, elevation)
	if breached := BreachDrainageSinks(cells, elevation, receivers, 220); breached != 0 {
		t.Fatalf("L7 mesh breached %d sinks through a 100m saddle, want 0 (threshold ~55m)", breached)
	}
	if elevation[1] != 200 {
		t.Fatalf("L7 saddle elevation changed to %.4f, want untouched 200", elevation[1])
	}

	// L7 mesh with a genuinely shallow saddle (40m < ~55m threshold): breached,
	// and the carve depth scales with the mesh step too.
	cells, elevation = breachFixture(l7Cells, 40)
	receivers = ComputeDrainageReceivers(cells, elevation)
	if breached := BreachDrainageSinks(cells, elevation, receivers, 220); breached == 0 {
		t.Fatal("expected L7 breach of 40m saddle with effective threshold ~55m")
	}
	stepScale := meshPathCostResolutionScale(l7Cells)
	wantCarved := 100 - 6.0*stepScale
	if math.Abs(elevation[1]-wantCarved) > 1e-9 {
		t.Fatalf("L7 spillway carve: elevation[1] = %.6f, want %.6f (carve depth 6*stepScale)", elevation[1], wantCarved)
	}
}
