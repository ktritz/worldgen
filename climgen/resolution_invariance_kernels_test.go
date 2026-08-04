package climgen

import (
	"math"
	"testing"
)

// buildUpwindChainWorld builds a synthetic world of `total` cells where the
// first `chainLen` cells form a chain along the equator with the given angular
// spacing, and the wind at every chain cell points along the chain (so cell
// j-1 is the strict upwind donor of cell j). All other cells are isolated.
func buildUpwindChainWorld(total, chainLen int, spacing float64) ([]Vector3D, *FlatAdjacency, []Vector3D) {
	vertices := make([]Vector3D, total)
	wind := make([]Vector3D, total)
	for j := 0; j < chainLen; j++ {
		theta := float64(j) * spacing
		vertices[j] = Vector3D{X: math.Cos(theta), Y: 0, Z: math.Sin(theta)}
		wind[j] = Vector3D{X: -math.Sin(theta), Y: 0, Z: math.Cos(theta)}
	}
	offsets := make([]int, total+1)
	neighbors := make([]int, 0, chainLen*2)
	for i := 0; i < total; i++ {
		offsets[i] = len(neighbors)
		if i < chainLen {
			if i > 0 {
				neighbors = append(neighbors, i-1)
			}
			if i+1 < chainLen {
				neighbors = append(neighbors, i+1)
			}
		}
	}
	offsets[total] = len(neighbors)
	return vertices, &FlatAdjacency{Neighbors: neighbors, Offsets: offsets}, wind
}

func relativeDiff(a, b float64) float64 {
	scale := math.Max(math.Abs(a), math.Abs(b))
	if scale < 1e-12 {
		return 0
	}
	return math.Abs(a-b) / scale
}

// R1.5: the lee-shadow per-hop decay must produce the same shadow strength at
// the same physical distance regardless of mesh resolution.
func TestPropagateLeeShadowDecayIsResolutionInvariant(t *testing.T) {
	speeds := func(total, chainLen, iterations int, spacing float64) []float64 {
		vertices, adj, wind := buildUpwindChainWorld(total, chainLen, spacing)
		wind[0] = Vector3D{} // becalmed source cell seeds shadow = 1.0
		elevation := make([]float64, total)
		for j := 0; j < chainLen; j++ {
			// Descending terrain downwind so the shadow persists at full strength.
			elevation[j] = 1000 - 10*float64(j)
		}
		result := PropagateLeeShadow(wind, vertices, elevation, adj, iterations, 0.75)
		out := make([]float64, chainLen)
		for j := 0; j < chainLen; j++ {
			out[j] = Length(result[j])
		}
		return out
	}

	coarse := speeds(10242, 6, 8, 0.03)
	fine := speeds(40962, 11, 16, 0.015)
	for j := 1; j <= 5; j++ {
		if diff := relativeDiff(coarse[j], fine[2*j]); diff > 0.01 {
			t.Fatalf(
				"lee shadow at physical hop %d: coarse speed %.5f vs fine speed %.5f (diff %.3f)",
				j, coarse[j], fine[2*j], diff,
			)
		}
	}
	// Sanity: the shadow actually decays along the chain at the baseline rate.
	if diff := relativeDiff(1.0-coarse[1], 0.8*0.75); diff > 1e-9 {
		t.Fatalf("expected baseline first-hop shadow 0.75, got speed %.5f", coarse[1])
	}
}

// R1.6: the lee-shadow iteration count must track the fixed 0.05 rad physical
// span across resolutions instead of being pinned by the old [3,10] clamp.
func TestLeeShadowIterationCountTracksPhysicalSpan(t *testing.T) {
	cases := []struct {
		cellSize float64
		want     int
	}{
		{0.035, 2},     // ~L5: old floor of 3 overshot the 0.05 rad target
		{0.0175, 3},    // ~L6: unchanged from old behavior
		{0.00875, 6},   // ~L7: unchanged from old behavior
		{0.004375, 12}, // ~L8: old cap of 10 truncated the physical span
		{0.0001, 40},   // safety cap
		{1.0, 1},       // floor
		{0, 40},        // degenerate input falls back to the safety cap
	}
	for _, tc := range cases {
		if got := leeShadowIterationCount(tc.cellSize); got != tc.want {
			t.Fatalf("leeShadowIterationCount(%.6f) = %d, want %d", tc.cellSize, got, tc.want)
		}
	}
}

// R1.7: the upwind footprint decay kernel must assign the same relative weight
// to donors at the same physical distance regardless of mesh resolution. The
// weight ratio between two physical positions cancels the normalization, so it
// isolates the decay kernel itself.
func TestUpwindFootprintDecayKernelIsResolutionInvariant(t *testing.T) {
	kernel := func(total, chainLen, source, maxDepth int, spacing float64) map[int]float64 {
		vertices, adj, wind := buildUpwindChainWorld(total, chainLen, spacing)
		return computeUpwindFootprintWeights(source, vertices, adj, wind, maxDepth, precipUpwindFootprintMinAlignment)
	}

	// Coarse: donors at 1, 2, 3 baseline hops upwind of cell 4.
	coarse := kernel(10242, 5, 4, 3, 0.03)
	// Fine: same physical span is 6 half-length hops upwind of cell 8.
	fine := kernel(40962, 9, 8, resolutionAdjustedPrecipSteps(3, 40962), 0.015)

	if len(coarse) != 3 {
		t.Fatalf("expected 3 coarse donors, got %d", len(coarse))
	}
	if len(fine) != 6 {
		t.Fatalf("expected 6 fine donors over the same physical span, got %d", len(fine))
	}

	// w(x=2)/w(x=1) must be the baseline per-hop decay 0.84 on both meshes.
	coarse21 := coarse[2] / coarse[3]
	fine21 := fine[4] / fine[6]
	if diff := relativeDiff(coarse21, 0.84); diff > 1e-9 {
		t.Fatalf("baseline kernel ratio w(2)/w(1) = %.5f, want 0.84", coarse21)
	}
	if diff := relativeDiff(fine21, coarse21); diff > 0.01 {
		t.Fatalf("fine kernel ratio w(2)/w(1) = %.5f vs baseline %.5f (diff %.3f)", fine21, coarse21, diff)
	}

	// w(x=3)/w(x=1) = 0.84^3 (the compounded baseline kernel 0.84^(m(m-1)/2)).
	coarse31 := coarse[1] / coarse[3]
	fine31 := fine[2] / fine[6]
	if diff := relativeDiff(coarse31, math.Pow(0.84, 3)); diff > 1e-9 {
		t.Fatalf("baseline kernel ratio w(3)/w(1) = %.5f, want 0.84^3", coarse31)
	}
	if diff := relativeDiff(fine31, coarse31); diff > 0.01 {
		t.Fatalf("fine kernel ratio w(3)/w(1) = %.5f vs baseline %.5f (diff %.3f)", fine31, coarse31, diff)
	}
}

// R1.8: hardcoded footprint depths wrapped in resolutionAdjustedPrecipSteps
// must reach the same physical distance on finer meshes. A moisture spike at
// 4 baseline hops (8 fine hops) upwind must be visible on both meshes; before
// the fix the fine mesh footprint stopped at half the physical span and
// returned zero.
func TestFrontalFootprintReachesEquivalentPhysicalSpanAcrossResolution(t *testing.T) {
	run := func(total, chainLen, source, spikeHops int, spacing float64) float64 {
		vertices, adj, wind := buildUpwindChainWorld(total, chainLen, spacing)
		elevation := make([]float64, total)
		for j := 0; j < chainLen; j++ {
			elevation[j] = 100
		}
		field := make([]float64, total)
		field[source-spikeHops] = 1
		return frontalTwoHopUpwindMean(source, field, vertices, elevation, 50, adj, wind)
	}

	coarse := run(10242, 6, 5, 4, 0.03)
	fine := run(40962, 10, 9, 8, 0.015)
	if coarse <= 0 {
		t.Fatalf("expected baseline footprint to see a spike 4 hops upwind, got %.5f", coarse)
	}
	if fine <= 0 {
		t.Fatalf("expected refined footprint to reach the same physical span (8 hops), got %.5f", fine)
	}
}

// R1.9: storm memory advects one hop per iteration, so the scaled iteration
// count must hold the physical propagation span fixed and the rescaled carry
// must keep mid-field amplitudes comparable at the same physical distance.
func TestSeasonalStormMemoryPropagationScalesWithResolution(t *testing.T) {
	run := func(total, chainLen int, spacing float64) []float64 {
		vertices, adj, wind := buildUpwindChainWorld(total, chainLen, spacing)
		elevation := make([]float64, total)
		interior := make([]float64, total)
		for j := 0; j < chainLen; j++ {
			elevation[j] = 100
			interior[j] = 1
		}
		field := make([]float64, total)
		field[0] = 1 // persistent source: cell 0 has no upwind donor, so it stays 1
		return computeSeasonalStormMemoryField(field, vertices, elevation, 50, adj, wind, interior)
	}

	coarse := run(10242, 5, 0.03) // 3 iterations reach 3 hops
	fine := run(40962, 9, 0.015)  // 6 iterations reach 6 hops = same physical span

	if coarse[3] <= 0 {
		t.Fatalf("expected baseline memory to reach 3 hops, got %.5f", coarse[3])
	}
	if fine[6] <= 0 {
		t.Fatalf("expected refined memory to reach the same physical span (6 hops), got %.5f", fine[6])
	}
	if fine[7] != 0 {
		t.Fatalf("expected refined memory to stop at the scaled span, got %.5f at hop 7", fine[7])
	}
	// Mid-field amplitude at the same physical distance. The per-iteration
	// blend factor is intentionally left unscaled (only iterations and carry
	// are scaled), so agreement is approximate away from the wavefront tip.
	for j := 1; j <= 2; j++ {
		if diff := relativeDiff(coarse[j], fine[2*j]); diff > 0.15 {
			t.Fatalf(
				"storm memory at physical hop %d: coarse %.5f vs fine %.5f (diff %.3f)",
				j, coarse[j], fine[2*j], diff,
			)
		}
	}
}

// R1.10: the hop-harmonic corridor weight 1/(1+step*stepScale) must weight
// donors by physical distance, so a linearly varying humidity field yields the
// same corridor mean at both resolutions (up to discretization error of the
// harmonic kernel near the origin).
func TestUpwindCorridorHumidityWeightsPhysicalDistance(t *testing.T) {
	run := func(total, chainLen, source, maxSteps int, spacing, hopLength float64) float64 {
		vertices, adj, wind := buildUpwindChainWorld(total, chainLen, spacing)
		humidity := make([]float64, total)
		for j := 0; j < chainLen; j++ {
			// Linear in physical distance upwind of the source cell.
			humidity[j] = float64(source-j) * hopLength
		}
		return computeUpwindCorridorHumidity(source, vertices, adj, wind, humidity, maxSteps)
	}

	coarse := run(10242, 11, 10, 10, 0.03, 1.0)
	fine := run(40962, 21, 20, 20, 0.015, 0.5)
	if coarse <= 0 || fine <= 0 {
		t.Fatalf("expected positive corridor means, got coarse %.5f fine %.5f", coarse, fine)
	}
	// Analytic values: coarse 10/H(10) ~ 3.414, fine ~ 3.280 (~4% apart). The
	// old per-hop weights gave ~2.78 on the fine mesh (~19% off baseline).
	if diff := relativeDiff(coarse, fine); diff > 0.06 {
		t.Fatalf("corridor humidity mean: coarse %.5f vs fine %.5f (diff %.3f)", coarse, fine, diff)
	}
}
