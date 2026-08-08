package climgen

import "math"

// The upwind footprint operator is linear. For a single cell i the frontier BFS
// in precipitation_upwind_footprint.go builds the weight distribution
//
//	acc_i = sum_{k=1..maxDepth} c_k * (e_i P^k)
//
// where P is the one-step upwind transition (row-normalized squared alignment
// weights, the same numbers weightedUpwindDonorsInto produces) and c_k is the
// compounded depth decay. Every consumer only ever contracts acc_i against a
// vector: the footprint mean is (acc_i . field) / (acc_i . mask). Both sides are
// linear in acc_i, so
//
//	(acc . v)_i = sum_k c_k (P^k v)_i
//
// can be evaluated for ALL cells with maxDepth sparse mat-vec products instead
// of one depth-limited BFS per cell. That turns O(N*depth^2) into O(N*depth),
// and it also removes the repeated per-frontier-cell donor recomputation (the
// transition is built once per minAlignment and reused).
//
// The per-cell implementation is kept as the reference oracle;
// precipitation_upwind_batch_test.go asserts the two agree.

// upwindTransition is the sparse one-step upwind operator P in CSR form for a
// fixed (mesh, wind, minAlignment). Row i lists the upwind donors of cell i with
// weights summing to 1. Rows that weightedUpwindDonorsInto would reject (no
// wind, no aligned neighbor, degenerate weight sum) are stored empty.
type upwindTransition struct {
	n       int
	offsets []int32
	donors  []int32
	weights []float64
}

func newUpwindTransition(
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	minAlignment float64,
) *upwindTransition {
	n := len(vertices)
	p := &upwindTransition{n: n, offsets: make([]int32, n+1)}
	if n == 0 || adj == nil {
		return p
	}
	minUpwind := Clamp(minAlignment, 0, 0.5)
	p.donors = make([]int32, 0, len(adj.Neighbors))
	p.weights = make([]float64, 0, len(adj.Neighbors))
	for i := 0; i < n; i++ {
		p.offsets[i] = int32(len(p.donors))
		if i >= len(wind) || i+1 >= len(adj.Offsets) {
			continue
		}
		windVec := wind[i]
		windSpeed := Length(windVec)
		if windSpeed < 1e-9 {
			continue
		}
		windDir := Scale(windVec, 1.0/windSpeed)
		rowStart := len(p.donors)
		weightSum := 0.0
		start := adj.Offsets[i]
		end := adj.Offsets[i+1]
		for e := start; e < end; e++ {
			k := adj.Neighbors[e]
			if k < 0 || k >= n {
				continue
			}
			upwind := Dot(windDir, Normalize(Sub(vertices[i], vertices[k])))
			if upwind <= minUpwind {
				continue
			}
			weight := upwind * upwind
			p.donors = append(p.donors, int32(k))
			p.weights = append(p.weights, weight)
			weightSum += weight
		}
		if weightSum <= 1e-9 {
			// Mirrors weightedUpwindDonorsInto returning no donors.
			p.donors = p.donors[:rowStart]
			p.weights = p.weights[:rowStart]
			continue
		}
		for idx := rowStart; idx < len(p.weights); idx++ {
			p.weights[idx] /= weightSum
		}
	}
	p.offsets[n] = int32(len(p.donors))
	return p
}

func (p *upwindTransition) rowEmpty(i int) bool {
	if p == nil || i < 0 || i >= p.n {
		return true
	}
	return p.offsets[i] == p.offsets[i+1]
}

// upwindTransitionCache builds and memoizes transitions per minAlignment for one
// (mesh, wind) pair. Wind changes between seasons, so the cache is created at
// the top of each precipitation pass rather than kept globally.
type upwindTransitionCache struct {
	vertices []Vector3D
	adj      *FlatAdjacency
	wind     []Vector3D
	keys     []float64
	values   []*upwindTransition
}

func newUpwindTransitionCache(vertices []Vector3D, adj *FlatAdjacency, wind []Vector3D) *upwindTransitionCache {
	return &upwindTransitionCache{vertices: vertices, adj: adj, wind: wind}
}

func (c *upwindTransitionCache) get(minAlignment float64) *upwindTransition {
	if c == nil {
		return nil
	}
	key := Clamp(minAlignment, 0, 0.5)
	for idx, k := range c.keys {
		if k == key {
			return c.values[idx]
		}
	}
	p := newUpwindTransition(c.vertices, c.adj, c.wind, key)
	c.keys = append(c.keys, key)
	c.values = append(c.values, p)
	return p
}

// upwindFootprintCoeffs returns c_k for k = 1..maxDepth, the compounded depth
// decay of the frontier BFS. The per-hop exponent is the resolution-correctness
// form from computeUpwindFootprintWeightsInto and must stay bit-identical.
func upwindFootprintCoeffs(maxDepth int, cellCount int) []float64 {
	if maxDepth <= 0 {
		return nil
	}
	stepScale := precipitationPhysicalStepScale(cellCount)
	coeffs := make([]float64, maxDepth)
	running := 1.0
	for depth := 0; depth < maxDepth; depth++ {
		exponent := stepScale * (stepScale*(float64(depth)+0.5) - 0.5)
		running *= math.Pow(precipUpwindFootprintDecay, exponent)
		coeffs[depth] = running
	}
	return coeffs
}

// applyStep computes y = P x for every supplied vector in a single sweep of the
// CSR structure, which amortizes the row traversal across the (at most three)
// vectors a consumer needs.
func (p *upwindTransition) applyStep(xs [][]float64, ys [][]float64) {
	for _, y := range ys {
		for i := range y {
			y[i] = 0
		}
	}
	for i := 0; i < p.n; i++ {
		start := p.offsets[i]
		end := p.offsets[i+1]
		if start == end {
			continue
		}
		for e := start; e < end; e++ {
			donor := int(p.donors[e])
			weight := p.weights[e]
			for v := range xs {
				ys[v][i] += weight * xs[v][donor]
			}
		}
	}
}

// accumulateFootprint returns W v = sum_k c_k P^k v for each input vector.
func (p *upwindTransition) accumulateFootprint(coeffs []float64, vecs [][]float64) [][]float64 {
	out := make([][]float64, len(vecs))
	xs := make([][]float64, len(vecs))
	ys := make([][]float64, len(vecs))
	for v := range vecs {
		out[v] = make([]float64, p.n)
		xs[v] = append([]float64(nil), vecs[v]...)
		ys[v] = make([]float64, p.n)
	}
	for k := 0; k < len(coeffs); k++ {
		p.applyStep(xs, ys)
		xs, ys = ys, xs
		coeff := coeffs[k]
		for v := range out {
			dst := out[v]
			src := xs[v]
			for i := range dst {
				dst[i] += coeff * src[i]
			}
		}
	}
	return out
}

// maskedField returns field*mask and mask as dense vectors of length p.n, with
// cells outside len(field) treated as excluded (the per-cell reference skips
// donors with donor >= len(field)).
func maskedFieldVectors(n int, field []float64, mask []bool) (masked []float64, indicator []float64) {
	masked = make([]float64, n)
	indicator = make([]float64, n)
	for i := 0; i < n; i++ {
		if i >= len(field) {
			continue
		}
		if mask != nil && (i >= len(mask) || !mask[i]) {
			continue
		}
		masked[i] = field[i]
		indicator[i] = 1
	}
	return masked, indicator
}

// batchUpwindFootprintMean is the all-cells form of upwindFootprintMean.
func batchUpwindFootprintMean(
	p *upwindTransition,
	coeffs []float64,
	field []float64,
	mask []bool,
) ([]float64, []bool) {
	values := make([]float64, p.n)
	ok := make([]bool, p.n)
	if len(coeffs) == 0 {
		return values, ok
	}
	masked, indicator := maskedFieldVectors(p.n, field, mask)
	ones := make([]float64, p.n)
	for i := range ones {
		ones[i] = 1
	}
	res := p.accumulateFootprint(coeffs, [][]float64{masked, indicator, ones})
	num, den, tot := res[0], res[1], res[2]
	for i := 0; i < p.n; i++ {
		// total <= 1e-12 is the reference's "no footprint" early return, and
		// weightSum is the mask share of the normalized weights, den/tot.
		if tot[i] <= 1e-12 || den[i] <= 1e-12*tot[i] {
			continue
		}
		values[i] = num[i] / den[i]
		ok[i] = true
	}
	return values, ok
}

// batchUpwindFootprintMax is the all-cells form of upwindFootprintMax. The max
// is not linear, but it batches the same way: "my max = max over my upwind
// neighbours of (their value, their running max)", one pass per depth, over the
// same transition structure.
func batchUpwindFootprintMax(
	p *upwindTransition,
	coeffs []float64,
	field []float64,
	mask []bool,
) ([]float64, []bool) {
	values := make([]float64, p.n)
	ok := make([]bool, p.n)
	if len(coeffs) == 0 {
		return values, ok
	}
	negInf := math.Inf(-1)
	val := make([]float64, p.n)
	for i := 0; i < p.n; i++ {
		val[i] = negInf
		if i >= len(field) {
			continue
		}
		if mask != nil && (i >= len(mask) || !mask[i]) {
			continue
		}
		val[i] = field[i]
	}
	// s[i] = best value within 0..k hops of i; reach[i] = best within 1..k hops.
	s := append([]float64(nil), val...)
	reach := make([]float64, p.n)
	next := make([]float64, p.n)
	for i := range reach {
		reach[i] = negInf
	}
	for k := 0; k < len(coeffs); k++ {
		for i := 0; i < p.n; i++ {
			best := negInf
			for e := p.offsets[i]; e < p.offsets[i+1]; e++ {
				if v := s[p.donors[e]]; v > best {
					best = v
				}
			}
			next[i] = best
		}
		reach, next = next, reach
		for i := 0; i < p.n; i++ {
			if val[i] > reach[i] {
				s[i] = val[i]
			} else {
				s[i] = reach[i]
			}
		}
	}
	// The reference reports no max when the footprint total underflows, so the
	// weight totals still gate the result.
	ones := make([]float64, p.n)
	for i := range ones {
		ones[i] = 1
	}
	tot := p.accumulateFootprint(coeffs, [][]float64{ones})[0]
	for i := 0; i < p.n; i++ {
		if tot[i] <= 1e-12 || math.IsInf(reach[i], -1) {
			continue
		}
		values[i] = reach[i]
		ok[i] = true
	}
	return values, ok
}

// batchLocalUpwindMean is the all-cells form of localUpwindMean (the depth-1
// case, with no depth decay and already-normalized weights).
func batchLocalUpwindMean(p *upwindTransition, field []float64, mask []bool) ([]float64, []bool) {
	values := make([]float64, p.n)
	ok := make([]bool, p.n)
	masked, indicator := maskedFieldVectors(p.n, field, mask)
	num := make([]float64, p.n)
	den := make([]float64, p.n)
	p.applyStep([][]float64{masked, indicator}, [][]float64{num, den})
	for i := 0; i < p.n; i++ {
		if p.rowEmpty(i) || den[i] <= 1e-12 {
			continue
		}
		values[i] = num[i] / den[i]
		ok[i] = true
	}
	return values, ok
}

// landMaskAtOrAbove builds the "donor is land" predicate used by the frontal
// consumers as a dense mask.
func landMaskAtOrAbove(elevation []float64, seaLevel float64, n int) []bool {
	mask := make([]bool, n)
	for i := 0; i < n && i < len(elevation); i++ {
		mask[i] = elevation[i] >= seaLevel
	}
	return mask
}

// batchUpwindFootprintMaskShare returns, for every cell, the share of its
// normalized footprint weight that lands on masked cells. That is the ratio of
// two linear reductions (W*mask and W*1), so it batches the same way as the
// footprint mean.
func batchUpwindFootprintMaskShare(p *upwindTransition, coeffs []float64, mask []bool) []float64 {
	out := make([]float64, p.n)
	if len(coeffs) == 0 {
		return out
	}
	selected := make([]float64, p.n)
	ones := make([]float64, p.n)
	for i := range ones {
		ones[i] = 1
		if i < len(mask) && mask[i] {
			selected[i] = 1
		}
	}
	res := p.accumulateFootprint(coeffs, [][]float64{selected, ones})
	for i := 0; i < p.n; i++ {
		if res[1][i] <= 1e-12 {
			continue
		}
		out[i] = Clamp(res[0][i]/res[1][i], 0, 1)
	}
	return out
}

// precipIterationFootprintSteps is the footprint depth used by the advection
// operator *inside* the land-budget relaxation loop, in mesh cells rather than
// physical distance.
//
// Footprints applied once to a static field (ocean fetch, tropical and frontal
// upwind support) describe a fixed physical catchment and must scale with the
// mesh. The advection footprint is different: the relaxation iteration is itself
// the transport step, and the loop runs meshResolutionAdjustedSteps(18, n)
// times. Giving that footprint a physical depth as well made its per-iteration
// displacement nearly resolution-independent (409/357/332 km at L5/L6/L7) while
// the iteration count grew as 1/stepScale, so total transport ran 7361 km at L5
// against 23894 km at L7 -- a 3.2x over-advection that flushed moisture downwind
// and starved continental interiors, which received 35% less precipitation at L7
// than at L5.
//
// Every rule in the resolution table assumes an advective operator moves one
// cell per pass. Fixing the depth in cells restores that.
const precipIterationFootprintSteps = 3

// upwindFootprintBaselineCoeffs returns the footprint kernel in mesh-cell units,
// for the two call sites that run inside the relaxation loop. At the baseline
// this is exactly what upwindFootprintCoeffs already produces, so L5 output is
// unchanged.
func upwindFootprintBaselineCoeffs() []float64 {
	coeffs := make([]float64, precipIterationFootprintSteps)
	running := 1.0
	for depth := 0; depth < precipIterationFootprintSteps; depth++ {
		running *= math.Pow(precipUpwindFootprintDecay, float64(depth))
		coeffs[depth] = running
	}
	return coeffs
}

