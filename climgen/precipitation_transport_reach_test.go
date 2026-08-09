package climgen

import "testing"

// TestScaledPrecipIterationsHoldsPhysicalReach pins the land-budget sweep as a
// physical transport reach rather than a convergence tolerance. One iteration
// advects one cell, so the budget has to grow as the cells shrink or continental
// interiors are cut off from marine moisture.
//
// The previous form derived the count from a cell-size ratio and floored it at
// precipMinIterations, and the floor swallowed the scaling: L6 and L7 both got
// essentially the baseline count, collapsing inland reach roughly 3x across the
// supported envelope.
func TestScaledPrecipIterationsHoldsPhysicalReach(t *testing.T) {
	cases := []struct {
		name      string
		cellCount int
		want      int
	}{
		{name: "L5", cellCount: 10242, want: 18},
		{name: "L6", cellCount: 40962, want: 36},
		{name: "L7", cellCount: 163842, want: 72},
	}
	baselineReach := 0.0
	for _, tc := range cases {
		got := scaledPrecipIterations(tc.cellCount)
		if got != tc.want {
			t.Errorf("%s: scaledPrecipIterations = %d, want %d", tc.name, got, tc.want)
		}
		// Reach in baseline cell widths: iterations x the physical step size.
		reach := float64(got) * meshPathCostResolutionScale(tc.cellCount)
		if tc.name == "L5" {
			baselineReach = reach
			continue
		}
		if ratio := reach / baselineReach; ratio < 0.95 || ratio > 1.05 {
			t.Errorf("%s: inland reach %.2f is %.2fx the L5 reach %.2f", tc.name, reach, ratio, baselineReach)
		}
	}
}
