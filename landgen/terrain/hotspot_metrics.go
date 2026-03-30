package terrain

import (
	"math"
	"sort"
)

func computeHotspotMetrics(metrics *TerrainMetrics, sites []Vector3D, chains []HotspotChain) {
	oceanic := make([]HotspotChain, 0, len(chains))
	for _, chain := range chains {
		if chain.IsOceanic && len(chain.Islands) >= MinChainLength {
			oceanic = append(oceanic, chain)
		}
	}
	if len(oceanic) == 0 {
		return
	}

	metrics.HotspotChainCount = len(oceanic)

	spacingCVs := make([]float64, 0, len(oceanic))
	burstiness := make([]float64, 0, len(oceanic))
	bentChains := 0
	bendEligible := 0

	for _, chain := range oceanic {
		spacings := chainAngularSpacings(sites, chain)
		if len(spacings) >= 2 {
			spacingCVs = append(spacingCVs, coefficientOfVariation(spacings))
			p25 := percentile(spacings, 0.25)
			p75 := percentile(spacings, 0.75)
			if p25 > 1e-6 {
				burstiness = append(burstiness, p75/p25)
			}
		}

		bent, ok := detectChainBend(sites, chain)
		if ok {
			bendEligible++
			if bent {
				bentChains++
			}
		}
	}

	if len(spacingCVs) > 0 {
		metrics.HotspotSpacingCV = mean(spacingCVs)
	}
	if len(burstiness) > 0 {
		metrics.HotspotBurstiness = percentile(burstiness, 0.5)
	}
	if bendEligible > 0 {
		metrics.HotspotBendFraction = float64(bentChains) / float64(bendEligible)
	}
}

func chainAngularSpacings(sites []Vector3D, chain HotspotChain) []float64 {
	if len(chain.Islands) < 2 {
		return nil
	}
	spacings := make([]float64, 0, len(chain.Islands)-1)
	for i := 1; i < len(chain.Islands); i++ {
		prev := chain.Islands[i-1].CellIndex
		curr := chain.Islands[i].CellIndex
		if prev < 0 || prev >= len(sites) || curr < 0 || curr >= len(sites) {
			continue
		}
		spacings = append(spacings, angularDistance(sites[prev], sites[curr]))
	}
	return spacings
}

func detectChainBend(sites []Vector3D, chain HotspotChain) (bool, bool) {
	if len(chain.Islands) < 5 {
		return false, false
	}

	found := false
	for i := 2; i < len(chain.Islands)-2; i++ {
		prevHeading, prevSpread, approachLen, ok := chainHeadingWindow(sites, chain, i-2, i, i)
		if !ok || approachLen < 0.04 {
			continue
		}
		nextHeading, nextSpread, departLen, ok := chainHeadingWindow(sites, chain, i, i+2, i)
		if !ok || departLen < 0.04 {
			continue
		}

		localTurn, ok := chainTurnAtPivot(sites, chain, i-1, i, i+1)
		if !ok || localTurn < 16.0*math.Pi/180.0 {
			continue
		}

		found = true
		broadTurn := math.Acos(Clamp(prevHeading.Dot(nextHeading), -1, 1))
		requiredBroadTurn := 30.0*math.Pi/180.0 + 1.4*(prevSpread+nextSpread)
		if broadTurn >= requiredBroadTurn {
			return true, true
		}
	}

	if !found {
		return false, false
	}

	return false, true
}

func chainTurnAtPivot(sites []Vector3D, chain HotspotChain, prevOffset, pivotOffset, nextOffset int) (float64, bool) {
	prevIdx := chain.Islands[prevOffset].CellIndex
	currIdx := chain.Islands[pivotOffset].CellIndex
	nextIdx := chain.Islands[nextOffset].CellIndex
	if prevIdx < 0 || prevIdx >= len(sites) || currIdx < 0 || currIdx >= len(sites) || nextIdx < 0 || nextIdx >= len(sites) {
		return 0, false
	}

	curr := sites[currIdx].Normalize()
	toPrev := projectOntoTangentPlane(curr, sites[prevIdx])
	toNext := projectOntoTangentPlane(curr, sites[nextIdx])
	if toPrev.Length() < 1e-9 || toNext.Length() < 1e-9 {
		return 0, false
	}
	angle := math.Acos(Clamp(toPrev.Normalize().Dot(toNext.Normalize()), -1, 1))
	return math.Abs(math.Pi - angle), true
}

func chainWindowLength(sites []Vector3D, chain HotspotChain, start, end int) float64 {
	total := 0.0
	for i := start + 1; i <= end; i++ {
		prevIdx := chain.Islands[i-1].CellIndex
		currIdx := chain.Islands[i].CellIndex
		if prevIdx < 0 || prevIdx >= len(sites) || currIdx < 0 || currIdx >= len(sites) {
			continue
		}
		total += angularDistance(sites[prevIdx], sites[currIdx])
	}
	return total
}

func chainHeadingWindow(sites []Vector3D, chain HotspotChain, start, end, pivotOffset int) (Vector3D, float64, float64, bool) {
	if end-start < 2 {
		return Vector3D{}, 0, 0, false
	}

	pivotIdx := chain.Islands[pivotOffset].CellIndex
	if pivotIdx < 0 || pivotIdx >= len(sites) {
		return Vector3D{}, 0, 0, false
	}
	pivot := sites[pivotIdx].Normalize()

	headings := make([]Vector3D, 0, end-start)
	totalLen := 0.0
	mean := Vector3D{}
	for i := start + 1; i <= end; i++ {
		prevIdx := chain.Islands[i-1].CellIndex
		currIdx := chain.Islands[i].CellIndex
		if prevIdx < 0 || prevIdx >= len(sites) || currIdx < 0 || currIdx >= len(sites) {
			continue
		}

		prev := projectOntoTangentPlane(pivot, sites[prevIdx])
		curr := projectOntoTangentPlane(pivot, sites[currIdx])
		dir := curr.Subtract(prev)
		length := dir.Length()
		if length < 1e-9 {
			continue
		}

		heading := dir.Scale(1 / length)
		headings = append(headings, heading)
		mean = mean.Add(heading)
		totalLen += angularDistance(sites[prevIdx], sites[currIdx])
	}

	if len(headings) < 2 || mean.Length() < 1e-9 {
		return Vector3D{}, 0, 0, false
	}

	mean = mean.Normalize()
	maxDeviation := 0.0
	for _, heading := range headings {
		deviation := math.Acos(Clamp(mean.Dot(heading), -1, 1))
		if deviation > maxDeviation {
			maxDeviation = deviation
		}
	}

	return mean, maxDeviation, totalLen, true
}

func projectOntoTangentPlane(origin, target Vector3D) Vector3D {
	target = target.Normalize()
	return target.Subtract(origin.Scale(target.Dot(origin)))
}

func coefficientOfVariation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	avg := mean(values)
	if avg <= 1e-9 {
		return 0
	}

	variance := 0.0
	for _, value := range values {
		diff := value - avg
		variance += diff * diff
	}
	variance /= float64(len(values))
	return math.Sqrt(variance) / avg
}

func percentile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if q <= 0 {
		return minFloat(values)
	}
	if q >= 1 {
		return maxFloat(values)
	}

	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if len(sorted) == 1 {
		return sorted[0]
	}
	pos := q * float64(len(sorted)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}
	frac := pos - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func minFloat(values []float64) float64 {
	best := values[0]
	for _, value := range values[1:] {
		if value < best {
			best = value
		}
	}
	return best
}

func maxFloat(values []float64) float64 {
	best := values[0]
	for _, value := range values[1:] {
		if value > best {
			best = value
		}
	}
	return best
}
