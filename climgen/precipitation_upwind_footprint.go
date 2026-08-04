package climgen

import "math"

const (
	precipUpwindFootprintMinAlignment = 0.04
	precipUpwindFootprintDecay        = 0.84
)

func localUpwindMean(
	i int,
	field []float64,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	minAlignment float64,
	include func(int) bool,
) (float64, bool) {
	ws := acquireUpwindWorkspace(len(vertices))
	defer releaseUpwindWorkspace(ws)
	donors32, weights := ws.weightedUpwindDonorsInto(i, vertices, adj, wind, Clamp(minAlignment, 0, 0.5), ws.edgeDirs(vertices, adj))
	if len(donors32) == 0 {
		return 0, false
	}
	sum := 0.0
	weightSum := 0.0
	for idx, donor32 := range donors32 {
		donor := int(donor32)
		if donor < 0 || donor >= len(field) {
			continue
		}
		if include != nil && !include(donor) {
			continue
		}
		weight := weights[idx]
		sum += field[donor] * weight
		weightSum += weight
	}
	if weightSum <= 1e-12 {
		return 0, false
	}
	return sum / weightSum, true
}

// computeUpwindFootprintWeights returns the normalized upwind footprint of cell
// i as a map. It is retained for tests and any caller that needs an owned
// result; production call sites use computeUpwindFootprintWeightsInto, which
// writes into reusable workspace buffers instead of allocating maps.
func computeUpwindFootprintWeights(
	i int,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	maxDepth int,
	minAlignment float64,
) map[int]float64 {
	ws := acquireUpwindWorkspace(len(vertices))
	defer releaseUpwindWorkspace(ws)
	cells, weights := computeUpwindFootprintWeightsInto(ws, i, vertices, adj, wind, maxDepth, minAlignment)
	if len(cells) == 0 {
		return nil
	}
	out := make(map[int]float64, len(cells))
	for idx, cell := range cells {
		out[int(cell)] = weights[idx]
	}
	return out
}

// computeUpwindFootprintWeightsInto performs the frontier BFS using the
// workspace's stamped index arrays. The returned slices alias workspace storage
// and stay valid only until the next call on the same workspace.
func computeUpwindFootprintWeightsInto(
	ws *upwindFootprintWorkspace,
	i int,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	maxDepth int,
	minAlignment float64,
) ([]int32, []float64) {
	if i < 0 || i >= len(vertices) || i >= len(wind) || maxDepth <= 0 || adj == nil {
		return nil, nil
	}
	dirs := ws.edgeDirs(vertices, adj)
	ws.gen++
	ws.accCell = ws.accCell[:0]
	ws.accW = ws.accW[:0]
	ws.curCell = append(ws.curCell[:0], int32(i))
	ws.curW = append(ws.curW[:0], 1.0)

	minUpwind := Clamp(minAlignment, 0, 0.5)
	stepScale := precipitationPhysicalStepScale(len(vertices))

	for depth := 0; depth < maxDepth; depth++ {
		ws.dgen++
		ws.nextCell = ws.nextCell[:0]
		ws.nextW = ws.nextW[:0]
		// The per-depth decay compounds through the frontier, so the baseline
		// (L5) kernel after m hops is 0.84^(m*(m-1)/2). Choose the per-hop
		// exponent so that the compounded decay at the same physical distance
		// x = m*stepScale reproduces that baseline kernel on finer meshes:
		// sum_{d=0}^{m-1} [stepScale^2*(d+0.5) - stepScale/2] = G(m*stepScale)
		// with G(x) = x*(x-1)/2. Exact no-op at L5 where stepScale == 1.
		exponent := stepScale * (stepScale*(float64(depth)+0.5) - 0.5)
		depthDecay := math.Pow(precipUpwindFootprintDecay, exponent)
		for fi, cell32 := range ws.curCell {
			incomingWeight := ws.curW[fi]
			if incomingWeight <= 1e-12 {
				continue
			}
			donors, weights := ws.weightedUpwindDonorsInto(int(cell32), vertices, adj, wind, minUpwind, dirs)
			for idx, donor := range donors {
				weight := incomingWeight * weights[idx] * depthDecay
				if weight <= 1e-12 {
					continue
				}
				if ws.accGen[donor] == ws.gen {
					ws.accW[ws.accSlot[donor]] += weight
				} else {
					ws.accGen[donor] = ws.gen
					ws.accSlot[donor] = int32(len(ws.accCell))
					ws.accCell = append(ws.accCell, donor)
					ws.accW = append(ws.accW, weight)
				}
				if ws.nextGen[donor] == ws.dgen {
					ws.nextW[ws.nextSlot[donor]] += weight
				} else {
					ws.nextGen[donor] = ws.dgen
					ws.nextSlot[donor] = int32(len(ws.nextCell))
					ws.nextCell = append(ws.nextCell, donor)
					ws.nextW = append(ws.nextW, weight)
				}
			}
		}
		if len(ws.nextCell) == 0 {
			break
		}
		ws.curCell = append(ws.curCell[:0], ws.nextCell...)
		ws.curW = append(ws.curW[:0], ws.nextW...)
	}

	total := 0.0
	for _, weight := range ws.accW {
		total += weight
	}
	if total <= 1e-12 {
		return nil, nil
	}
	invTotal := 1.0 / total
	for idx := range ws.accW {
		ws.accW[idx] *= invTotal
	}
	return ws.accCell, ws.accW
}

func upwindFootprintMean(
	i int,
	field []float64,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	maxDepth int,
	minAlignment float64,
	include func(int) bool,
) (float64, bool) {
	ws := acquireUpwindWorkspace(len(vertices))
	defer releaseUpwindWorkspace(ws)
	cells, weights := computeUpwindFootprintWeightsInto(ws, i, vertices, adj, wind, maxDepth, minAlignment)
	if len(cells) == 0 {
		return 0, false
	}
	sum := 0.0
	weightSum := 0.0
	for idx, donor32 := range cells {
		donor := int(donor32)
		if donor < 0 || donor >= len(field) {
			continue
		}
		if include != nil && !include(donor) {
			continue
		}
		weight := weights[idx]
		sum += field[donor] * weight
		weightSum += weight
	}
	if weightSum <= 1e-12 {
		return 0, false
	}
	return sum / weightSum, true
}

// upwindFootprintMeanMax returns both the footprint-weighted mean and the
// footprint maximum from a single BFS. The two used to be computed by separate
// identical traversals.
func upwindFootprintMeanMax(
	i int,
	field []float64,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	maxDepth int,
	minAlignment float64,
	include func(int) bool,
) (meanValue float64, meanOK bool, maxValue float64, maxOK bool) {
	ws := acquireUpwindWorkspace(len(vertices))
	defer releaseUpwindWorkspace(ws)
	cells, weights := computeUpwindFootprintWeightsInto(ws, i, vertices, adj, wind, maxDepth, minAlignment)
	if len(cells) == 0 {
		return 0, false, 0, false
	}
	sum := 0.0
	weightSum := 0.0
	best := 0.0
	for idx, donor32 := range cells {
		donor := int(donor32)
		if donor < 0 || donor >= len(field) {
			continue
		}
		if include != nil && !include(donor) {
			continue
		}
		weight := weights[idx]
		sum += field[donor] * weight
		weightSum += weight
		if !maxOK || field[donor] > best {
			best = field[donor]
			maxOK = true
		}
	}
	if weightSum > 1e-12 {
		meanValue = sum / weightSum
		meanOK = true
	}
	return meanValue, meanOK, best, maxOK
}

func upwindFootprintMax(
	i int,
	field []float64,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	maxDepth int,
	minAlignment float64,
	include func(int) bool,
) (float64, bool) {
	ws := acquireUpwindWorkspace(len(vertices))
	defer releaseUpwindWorkspace(ws)
	cells, _ := computeUpwindFootprintWeightsInto(ws, i, vertices, adj, wind, maxDepth, minAlignment)
	if len(cells) == 0 {
		return 0, false
	}
	best := 0.0
	found := false
	for _, donor32 := range cells {
		donor := int(donor32)
		if donor < 0 || donor >= len(field) {
			continue
		}
		if include != nil && !include(donor) {
			continue
		}
		if !found || field[donor] > best {
			best = field[donor]
			found = true
		}
	}
	return best, found
}
