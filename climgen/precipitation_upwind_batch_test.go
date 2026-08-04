package climgen

import (
	"math"
	"math/rand"
	"testing"
)

// Differential tolerance for the batched upwind footprint operator. The batched
// form sums the same terms in a different order (depth-major mat-vec instead of
// per-source frontier BFS), so results agree to summation-order noise rather
// than bit-for-bit.
const upwindBatchRelTol = 1e-9

func upwindBatchClose(a, b float64) bool {
	diff := math.Abs(a - b)
	scale := math.Max(math.Abs(a), math.Abs(b))
	if scale < 1e-12 {
		return diff <= 1e-12
	}
	return diff/scale <= upwindBatchRelTol
}

// buildRandomSphereMesh builds an irregular mesh: n pseudo-random points on the
// unit sphere, each linked to its k nearest neighbours with the adjacency
// symmetrized, so cells end up with varying degree.
func buildRandomSphereMesh(n, k int, seed int64) ([]Vector3D, *FlatAdjacency) {
	rng := rand.New(rand.NewSource(seed))
	vertices := make([]Vector3D, n)
	for i := range vertices {
		z := 2*rng.Float64() - 1
		phi := 2 * math.Pi * rng.Float64()
		r := math.Sqrt(math.Max(0, 1-z*z))
		vertices[i] = Vector3D{X: r * math.Cos(phi), Y: r * math.Sin(phi), Z: z}
	}
	sets := make([]map[int]bool, n)
	for i := range sets {
		sets[i] = map[int]bool{}
	}
	for i := 0; i < n; i++ {
		type cand struct {
			idx int
			d   float64
		}
		best := make([]cand, 0, n)
		for j := 0; j < n; j++ {
			if j == i {
				continue
			}
			d := Dot(vertices[i], vertices[j])
			best = append(best, cand{j, -d})
		}
		for a := 0; a < len(best); a++ {
			for b := a + 1; b < len(best); b++ {
				if best[b].d < best[a].d {
					best[a], best[b] = best[b], best[a]
				}
			}
		}
		limit := k
		if i%3 == 0 && k > 2 {
			limit = k - 1 // vary degree so adjacency is genuinely irregular
		}
		if limit > len(best) {
			limit = len(best)
		}
		for _, c := range best[:limit] {
			sets[i][c.idx] = true
			sets[c.idx][i] = true
		}
	}
	offsets := make([]int, n+1)
	neighbors := make([]int, 0, n*k*2)
	for i := 0; i < n; i++ {
		offsets[i] = len(neighbors)
		for j := 0; j < n; j++ {
			if sets[i][j] {
				neighbors = append(neighbors, j)
			}
		}
	}
	offsets[n] = len(neighbors)
	return vertices, &FlatAdjacency{Neighbors: neighbors, Offsets: offsets}
}

// buildLatLonMesh builds a regular lat/lon grid mesh with 4-connectivity and
// wrap-around in longitude.
func buildLatLonMesh(rows, cols int) ([]Vector3D, *FlatAdjacency) {
	n := rows * cols
	vertices := make([]Vector3D, n)
	idx := func(r, c int) int { return r*cols + ((c%cols)+cols)%cols }
	for r := 0; r < rows; r++ {
		lat := (float64(r)/float64(rows-1) - 0.5) * math.Pi * 0.9
		for c := 0; c < cols; c++ {
			lon := 2 * math.Pi * float64(c) / float64(cols)
			vertices[idx(r, c)] = Vector3D{
				X: math.Cos(lat) * math.Cos(lon),
				Y: math.Cos(lat) * math.Sin(lon),
				Z: math.Sin(lat),
			}
		}
	}
	offsets := make([]int, n+1)
	neighbors := make([]int, 0, n*4)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			offsets[idx(r, c)] = len(neighbors)
			neighbors = append(neighbors, idx(r, c-1), idx(r, c+1))
			if r > 0 {
				neighbors = append(neighbors, idx(r-1, c))
			}
			if r+1 < rows {
				neighbors = append(neighbors, idx(r+1, c))
			}
		}
	}
	offsets[n] = len(neighbors)
	return vertices, &FlatAdjacency{Neighbors: neighbors, Offsets: offsets}
}

type upwindBatchCase struct {
	name     string
	vertices []Vector3D
	adj      *FlatAdjacency
	wind     []Vector3D
	field    []float64
	mask     []bool
}

func buildUpwindBatchCases() []upwindBatchCase {
	var cases []upwindBatchCase

	// Irregular random mesh, smoothly varying wind, smooth field.
	{
		vertices, adj := buildRandomSphereMesh(220, 5, 12345)
		n := len(vertices)
		wind := make([]Vector3D, n)
		field := make([]float64, n)
		mask := make([]bool, n)
		rng := rand.New(rand.NewSource(777))
		for i := range vertices {
			v := vertices[i]
			// Zonal flow with a meridional perturbation.
			wind[i] = Vector3D{X: -v.Y, Y: v.X, Z: 0.35 * (1 - v.Z*v.Z)}
			field[i] = 3 + 2*math.Sin(4*v.X) + math.Cos(3*v.Z) + 0.1*rng.Float64()
			mask[i] = v.Z > -0.2
		}
		cases = append(cases, upwindBatchCase{"irregular-zonal", vertices, adj, wind, field, mask})
	}

	// Irregular mesh, fully random wind directions (many rows with few or no
	// aligned donors), spiky field so the max path is exercised hard.
	{
		vertices, adj := buildRandomSphereMesh(180, 4, 98765)
		n := len(vertices)
		wind := make([]Vector3D, n)
		field := make([]float64, n)
		mask := make([]bool, n)
		rng := rand.New(rand.NewSource(4242))
		for i := range vertices {
			wind[i] = Vector3D{
				X: 2*rng.Float64() - 1,
				Y: 2*rng.Float64() - 1,
				Z: 2*rng.Float64() - 1,
			}
			if i%17 == 0 {
				wind[i] = Vector3D{} // becalmed cells: no donors at all
			}
			field[i] = rng.Float64() * 10
			if i%23 == 0 {
				field[i] = 500 // isolated spikes
			}
			mask[i] = i%5 != 0
		}
		cases = append(cases, upwindBatchCase{"irregular-random-wind", vertices, adj, wind, field, mask})
	}

	// Regular lat/lon grid with a strong zonal jet.
	{
		vertices, adj := buildLatLonMesh(11, 20)
		n := len(vertices)
		wind := make([]Vector3D, n)
		field := make([]float64, n)
		mask := make([]bool, n)
		for i := range vertices {
			v := vertices[i]
			wind[i] = Vector3D{X: -v.Y, Y: v.X, Z: 0}
			field[i] = 1 + v.Z*v.Z + 0.5*v.X
			mask[i] = v.X < 0.5
		}
		cases = append(cases, upwindBatchCase{"latlon-zonal", vertices, adj, wind, field, mask})
	}

	// Degenerate chain world reused from the resolution-invariance tests.
	{
		vertices, adj, wind := buildUpwindChainWorld(40, 24, 0.05)
		field := make([]float64, len(vertices))
		mask := make([]bool, len(vertices))
		for i := range field {
			field[i] = float64(i%7) + 0.25
			mask[i] = i%2 == 0
		}
		cases = append(cases, upwindBatchCase{"chain", vertices, adj, wind, field, mask})
	}

	return cases
}

func maskPredicate(mask []bool) func(int) bool {
	if mask == nil {
		return nil
	}
	return func(idx int) bool {
		return idx >= 0 && idx < len(mask) && mask[idx]
	}
}

// TestBatchedUpwindFootprintMatchesPerCell is the differential gate for the
// linear (batched) upwind footprint operator: for every cell, on several meshes
// and several (maxDepth, minAlignment) settings, with and without an include
// mask, the batched result must match the per-cell frontier BFS reference.
func TestBatchedUpwindFootprintMatchesPerCell(t *testing.T) {
	depths := []int{1, 2, 3, 5, 8}
	// Every production call site uses minAlignment >= 0.02. minAlignment == 0 is
	// covered separately by TestBatchedUpwindFootprintZeroAlignment because the
	// reference BFS then prunes paths whose accumulated weight falls under
	// 1e-12, which the batched (source-independent) form cannot reproduce.
	alignments := []float64{0.02, 0.04, 0.2}

	for _, tc := range buildUpwindBatchCases() {
		for _, useMask := range []bool{false, true} {
			var mask []bool
			if useMask {
				mask = tc.mask
			}
			include := maskPredicate(mask)
			for _, align := range alignments {
				p := newUpwindTransition(tc.vertices, tc.adj, tc.wind, align)

				// Depth-1 (localUpwindMean) path.
				lv, lok := batchLocalUpwindMean(p, tc.field, mask)
				for i := range tc.vertices {
					wantV, wantOK := localUpwindMean(i, tc.field, tc.vertices, tc.adj, tc.wind, align, include)
					if lok[i] != wantOK {
						t.Fatalf("%s mask=%v align=%v cell %d: localUpwindMean ok %v want %v",
							tc.name, useMask, align, i, lok[i], wantOK)
					}
					if wantOK && !upwindBatchClose(lv[i], wantV) {
						t.Fatalf("%s mask=%v align=%v cell %d: localUpwindMean %.17g want %.17g",
							tc.name, useMask, align, i, lv[i], wantV)
					}
				}

				for _, depth := range depths {
					coeffs := upwindFootprintCoeffs(depth, len(tc.vertices))
					mv, mok := batchUpwindFootprintMean(p, coeffs, tc.field, mask)
					xv, xok := batchUpwindFootprintMax(p, coeffs, tc.field, mask)
					for i := range tc.vertices {
						wantMean, wantMeanOK, wantMax, wantMaxOK := upwindFootprintMeanMax(
							i, tc.field, tc.vertices, tc.adj, tc.wind, depth, align, include)
						if mok[i] != wantMeanOK {
							t.Fatalf("%s mask=%v align=%v depth=%d cell %d: mean ok %v want %v",
								tc.name, useMask, align, depth, i, mok[i], wantMeanOK)
						}
						if wantMeanOK && !upwindBatchClose(mv[i], wantMean) {
							t.Fatalf("%s mask=%v align=%v depth=%d cell %d: mean %.17g want %.17g",
								tc.name, useMask, align, depth, i, mv[i], wantMean)
						}
						if xok[i] != wantMaxOK {
							t.Fatalf("%s mask=%v align=%v depth=%d cell %d: max ok %v want %v",
								tc.name, useMask, align, depth, i, xok[i], wantMaxOK)
						}
						if wantMaxOK && !upwindBatchClose(xv[i], wantMax) {
							t.Fatalf("%s mask=%v align=%v depth=%d cell %d: max %.17g want %.17g",
								tc.name, useMask, align, depth, i, xv[i], wantMax)
						}

						// upwindFootprintMean / upwindFootprintMax must agree
						// with the combined reference too.
						gotMean, gotMeanOK := upwindFootprintMean(
							i, tc.field, tc.vertices, tc.adj, tc.wind, depth, align, include)
						if gotMeanOK != wantMeanOK || (wantMeanOK && gotMean != wantMean) {
							t.Fatalf("%s cell %d: reference mean/meanMax disagree", tc.name, i)
						}
						gotMax, gotMaxOK := upwindFootprintMax(
							i, tc.field, tc.vertices, tc.adj, tc.wind, depth, align, include)
						if gotMaxOK != wantMaxOK || (wantMaxOK && gotMax != wantMax) {
							t.Fatalf("%s cell %d: reference max/meanMax disagree", tc.name, i)
						}
					}
				}
			}
		}
	}
}

// TestBatchedUpwindFootprintZeroAlignment documents the one regime where the
// batched form is not exactly equivalent to the per-cell reference.
//
// With minAlignment == 0 a donor may sit almost exactly perpendicular to the
// wind, giving a transition weight near 0. The reference BFS drops any frontier
// contribution whose accumulated weight falls to <= 1e-12, and that test depends
// on the source cell, so a batched operator (which is source-independent by
// construction) cannot reproduce it. The consequences are bounded and one-sided:
// the batched footprint set is a superset of the reference set, so the mean
// differs only by the dropped sub-1e-12 mass, and the batched max is always >=
// the reference max. No production call site uses minAlignment == 0 (they use
// 0.02 and precipUpwindFootprintMinAlignment = 0.04), where the divergence
// disappears entirely.
func TestBatchedUpwindFootprintZeroAlignment(t *testing.T) {
	const looseTol = 1e-6
	for _, tc := range buildUpwindBatchCases() {
		p := newUpwindTransition(tc.vertices, tc.adj, tc.wind, 0)
		for _, depth := range []int{1, 3, 8} {
			coeffs := upwindFootprintCoeffs(depth, len(tc.vertices))
			mv, mok := batchUpwindFootprintMean(p, coeffs, tc.field, nil)
			xv, xok := batchUpwindFootprintMax(p, coeffs, tc.field, nil)
			for i := range tc.vertices {
				wantMean, wantMeanOK, wantMax, wantMaxOK := upwindFootprintMeanMax(
					i, tc.field, tc.vertices, tc.adj, tc.wind, depth, 0, nil)
				if mok[i] != wantMeanOK {
					t.Fatalf("%s depth=%d cell %d: mean ok %v want %v", tc.name, depth, i, mok[i], wantMeanOK)
				}
				if wantMeanOK {
					if d := math.Abs(mv[i]-wantMean) / math.Max(math.Abs(wantMean), 1e-12); d > looseTol {
						t.Fatalf("%s depth=%d cell %d: mean rel diff %.3g exceeds %.3g",
							tc.name, depth, i, d, looseTol)
					}
				}
				if wantMaxOK && !xok[i] {
					t.Fatalf("%s depth=%d cell %d: batched max missing where reference found one",
						tc.name, depth, i)
				}
				if wantMaxOK && xv[i] < wantMax {
					t.Fatalf("%s depth=%d cell %d: batched max %.17g below reference %.17g",
						tc.name, depth, i, xv[i], wantMax)
				}
			}
		}
	}
}

// TestUpwindTransitionMatchesDonorWeights pins the batched transition operator
// against the per-cell donor weights it replaces.
func TestUpwindTransitionMatchesDonorWeights(t *testing.T) {
	ws := acquireUpwindWorkspace(0)
	defer releaseUpwindWorkspace(ws)
	for _, tc := range buildUpwindBatchCases() {
		for _, align := range []float64{0.0, 0.02, 0.04, 0.2} {
			p := newUpwindTransition(tc.vertices, tc.adj, tc.wind, align)
			dirs := ws.edgeDirs(tc.vertices, tc.adj)
			for i := range tc.vertices {
				donors, weights := ws.weightedUpwindDonorsInto(
					i, tc.vertices, tc.adj, tc.wind, Clamp(align, 0, 0.5), dirs)
				start, end := p.offsets[i], p.offsets[i+1]
				if int(end-start) != len(donors) {
					t.Fatalf("%s align=%v cell %d: %d donors want %d",
						tc.name, align, i, end-start, len(donors))
				}
				for idx := range donors {
					if p.donors[int(start)+idx] != donors[idx] {
						t.Fatalf("%s align=%v cell %d: donor %d mismatch", tc.name, align, i, idx)
					}
					if !upwindBatchClose(p.weights[int(start)+idx], weights[idx]) {
						t.Fatalf("%s align=%v cell %d: weight %d = %.17g want %.17g",
							tc.name, align, i, idx, p.weights[int(start)+idx], weights[idx])
					}
				}
			}
		}
	}
}

// TestBatchedOceanFootprintSupportMatchesPerCell is the differential gate for
// computeUpwindOceanFootprintSupportField, the batched form of
// computeUpwindOceanFootprintSupport (the ocean share of the footprint weight).
func TestBatchedOceanFootprintSupportMatchesPerCell(t *testing.T) {
	for _, tc := range buildUpwindBatchCases() {
		// Reuse the case mask as "is land": support counts the complement.
		elevation := make([]float64, len(tc.vertices))
		for i := range elevation {
			if tc.mask[i] {
				elevation[i] = 100
			} else {
				elevation[i] = -100
			}
		}
		cache := newUpwindTransitionCache(tc.vertices, tc.adj, tc.wind)
		for _, depth := range []int{1, 3, 8} {
			got := computeUpwindOceanFootprintSupportField(tc.vertices, elevation, 0, depth, cache)
			for i := range tc.vertices {
				want := computeUpwindOceanFootprintSupport(
					i, tc.vertices, elevation, 0, tc.adj, tc.wind, depth)
				if !upwindBatchClose(got[i], want) {
					t.Fatalf("%s depth=%d cell %d: support %.17g want %.17g",
						tc.name, depth, i, got[i], want)
				}
			}
		}
	}
}
