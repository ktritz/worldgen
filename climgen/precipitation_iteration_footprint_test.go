package climgen

import (
	"math"
	"testing"
)

// TestIterationFootprintTransportIsMeshInvariant pins the advection footprint
// that runs inside the land-budget relaxation loop to mesh-cell units.
//
// The loop iterates meshResolutionAdjustedSteps(precipMinIterations, n) times,
// so the iteration is the transport step. Giving the footprint a physical depth
// as well left its per-iteration displacement nearly resolution-independent
// while the iteration count grew as 1/stepScale, over-advecting the land
// reservoir about 3.2x at L7 and starving continental interiors.
func TestIterationFootprintTransportIsMeshInvariant(t *testing.T) {
	// Kernel centroid in mesh cells, weighted by the footprint coefficients.
	centroidCells := func(coeffs []float64) float64 {
		var num, den float64
		for d, c := range coeffs {
			num += c * float64(d+1)
			den += c
		}
		return num / den
	}
	baselineCoeffs := upwindFootprintBaselineCoeffs()
	baselineTransport := centroidCells(baselineCoeffs) *
		float64(meshResolutionAdjustedSteps(precipMinIterations, 10242)) *
		meshPathCostResolutionScale(10242)

	for _, tc := range []struct {
		name      string
		cellCount int
	}{{"L5", 10242}, {"L6", 40962}, {"L7", 163842}} {
		// Total transport in baseline cell widths: centroid (cells) x iterations
		// x the physical width of a cell.
		got := centroidCells(upwindFootprintBaselineCoeffs()) *
			float64(meshResolutionAdjustedSteps(precipMinIterations, tc.cellCount)) *
			meshPathCostResolutionScale(tc.cellCount)
		if ratio := got / baselineTransport; ratio < 0.98 || ratio > 1.02 {
			t.Errorf("%s: total footprint transport %.1f is %.2fx the L5 value %.1f",
				tc.name, got, ratio, baselineTransport)
		}

		// The physical-depth kernel is what regressed: assert it would fail, so
		// this test cannot silently pass if the call sites revert.
		physical := centroidCells(upwindFootprintCoeffs(resolutionAdjustedPrecipSteps(3, tc.cellCount), tc.cellCount)) *
			float64(meshResolutionAdjustedSteps(precipMinIterations, tc.cellCount)) *
			meshPathCostResolutionScale(tc.cellCount)
		if tc.cellCount == 10242 {
			if math.Abs(physical-baselineTransport) > 1e-9 {
				t.Fatalf("L5: the mesh-relative and physical kernels must agree at the baseline, got %.6f vs %.6f", physical, baselineTransport)
			}
			continue
		}
		if physical/baselineTransport < 1.3 {
			t.Errorf("%s: expected the physical-depth kernel to over-advect by >1.3x, got %.2fx", tc.name, physical/baselineTransport)
		}
	}
}
