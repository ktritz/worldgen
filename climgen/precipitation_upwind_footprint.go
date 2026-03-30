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
	donors, weights := computeWeightedUpwindDonors(i, vertices, adj, wind, minAlignment)
	if len(donors) == 0 {
		return 0, false
	}
	sum := 0.0
	weightSum := 0.0
	for idx, donor := range donors {
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

func computeUpwindFootprintWeights(
	i int,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	maxDepth int,
	minAlignment float64,
) map[int]float64 {
	if i < 0 || i >= len(vertices) || i >= len(wind) || maxDepth <= 0 {
		return nil
	}
	frontier := map[int]float64{i: 1.0}
	accum := make(map[int]float64, 16)
	minUpwind := Clamp(minAlignment, 0, 0.5)

	for depth := 0; depth < maxDepth; depth++ {
		next := make(map[int]float64, 16)
		depthDecay := math.Pow(precipUpwindFootprintDecay, float64(depth))
		for cell, incomingWeight := range frontier {
			if incomingWeight <= 1e-12 {
				continue
			}
			donors, weights := computeWeightedUpwindDonors(cell, vertices, adj, wind, minUpwind)
			for idx, donor := range donors {
				weight := incomingWeight * weights[idx] * depthDecay
				if weight <= 1e-12 {
					continue
				}
				accum[donor] += weight
				next[donor] += weight
			}
		}
		if len(next) == 0 {
			break
		}
		frontier = next
	}

	total := 0.0
	for _, weight := range accum {
		total += weight
	}
	if total <= 1e-12 {
		return nil
	}
	invTotal := 1.0 / total
	for donor, weight := range accum {
		accum[donor] = weight * invTotal
	}
	return accum
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
	weights := computeUpwindFootprintWeights(i, vertices, adj, wind, maxDepth, minAlignment)
	if len(weights) == 0 {
		return 0, false
	}
	sum := 0.0
	weightSum := 0.0
	for donor, weight := range weights {
		if donor < 0 || donor >= len(field) {
			continue
		}
		if include != nil && !include(donor) {
			continue
		}
		sum += field[donor] * weight
		weightSum += weight
	}
	if weightSum <= 1e-12 {
		return 0, false
	}
	return sum / weightSum, true
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
	weights := computeUpwindFootprintWeights(i, vertices, adj, wind, maxDepth, minAlignment)
	if len(weights) == 0 {
		return 0, false
	}
	best := 0.0
	found := false
	for donor := range weights {
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
