package climgen

import (
	"math"
	"testing"
)

// TestOceanMarineColumnAttractorIsMeshInvariant pins the fixed point of the
// marine column relaxation.
//
// The per-step update is affine in the incoming humidity, so a streamline
// relaxes to b/(1-a). Converting retention, evaporation and condensation to
// per-step form separately leaves that attractor mesh-dependent, because the
// supersaturation drain relaxes toward the moisture capacity rather than toward
// zero and the capacity therefore acts as an effective source whose per-distance
// strength grows as the step shrinks. That lifted the whole marine field about
// 11% per mesh halving and raised its ceiling with it.
func TestOceanMarineColumnAttractorIsMeshInvariant(t *testing.T) {
	const (
		source        = 0.769
		capacity      = 0.923
		retentionBase = 0.97
		// rainfallFractionPerCell is a per-km rate times the cell width, so it
		// carries the step; the baseline value is scaled per level below.
		baselineRainfallPerCell = 0.001 * 223.2
	)
	relax := func(cellCount int) float64 {
		stepScale := meshPathCostResolutionScale(cellCount)
		m := 0.0
		for i := 0; i < 20000; i++ {
			var next float64
			if stepScale == 1 {
				q := m + source
				next = (q - computeOceanCondensation(q, capacity, baselineRainfallPerCell, cellCount)) * retentionBase
			} else {
				next = advanceOceanMarineColumn(m, source, capacity, retentionBase, baselineRainfallPerCell*stepScale, stepScale)
			}
			if math.Abs(next-m) < 1e-12 {
				return next
			}
			m = next
		}
		return m
	}
	baseline := relax(10242)
	if baseline <= 0 {
		t.Fatalf("degenerate baseline attractor %.6f", baseline)
	}
	for _, tc := range []struct {
		name      string
		cellCount int
	}{{"L6", 40962}, {"L7", 163842}, {"L8", 655362}} {
		got := relax(tc.cellCount)
		if ratio := got / baseline; ratio < 0.99 || ratio > 1.01 {
			t.Errorf("%s: marine attractor %.4f is %.3fx the L5 attractor %.4f; the per-term conversion gave about 1.11x per halving",
				tc.name, got, ratio, baseline)
		}
	}
}
