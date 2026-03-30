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
