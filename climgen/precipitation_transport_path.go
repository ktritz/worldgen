package climgen

import "math"

const precipInlandTransportSteps = 10

func resolutionAdjustedPrecipSteps(baseSteps int, cellCount int) int {
	return meshResolutionAdjustedSteps(baseSteps, cellCount)
}

func precipitationPhysicalStepScale(cellCount int) float64 {
	return meshPathCostResolutionScale(cellCount)
}

func precipitationPerStepFactor(baseFactor float64, cellCount int) float64 {
	if baseFactor <= 0 {
		return 0
	}
	if baseFactor == 1 {
		return 1
	}
	return math.Pow(baseFactor, precipitationPhysicalStepScale(cellCount))
}

func precipitationPerStepFraction(baseFraction float64, cellCount int) float64 {
	frac := Clamp(baseFraction, 0, 1)
	if frac <= 0 || frac >= 1 {
		return frac
	}
	// math.Pow(x, 1) is exact but 1-(1-f) is not: 0.20 round-trips to
	// 0.19999999999999996. These fractions feed the iterative land budget,
	// which early-exits on a convergence test, so a one-ulp drift can change
	// the iteration count at the baseline. Short-circuit to keep L5 exact.
	scale := precipitationPhysicalStepScale(cellCount)
	if scale == 1 {
		return frac
	}
	return 1.0 - math.Pow(1.0-frac, scale)
}

func computeUpwindLandTravel(
	i int,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	maxSteps int,
) float64 {
	if i < 0 || i >= len(vertices) || i >= len(elevation) || elevation[i] < seaLevel {
		return 0
	}
	current := i
	steps := 0
	for step := 0; step < maxSteps; step++ {
		next, upwindness := strongestUpwindNeighbor(current, vertices, adj, wind)
		if next < 0 || upwindness <= 0.05 {
			break
		}
		if elevation[next] < seaLevel {
			break
		}
		steps++
		current = next
	}
	return Clamp(float64(steps)/float64(maxSteps), 0, 1)
}

func computeUpwindCorridorHumidity(
	i int,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	specificHumidity []float64,
	maxSteps int,
) float64 {
	if i < 0 || i >= len(vertices) || i >= len(wind) || i >= len(specificHumidity) {
		return 0
	}
	current := i
	sum := 0.0
	weightSum := 0.0
	stepScale := precipitationPhysicalStepScale(len(vertices))
	for step := 0; step < maxSteps; step++ {
		next, upwindness := strongestUpwindNeighbor(current, vertices, adj, wind)
		if next < 0 || upwindness <= 0.05 || next >= len(specificHumidity) {
			break
		}
		// Harmonic falloff per physical distance, not per hop (no-op at L5).
		weight := upwindness / (1.0 + float64(step)*stepScale)
		sum += specificHumidity[next] * weight
		weightSum += weight
		current = next
	}
	if weightSum <= 1e-9 {
		return 0
	}
	return sum / weightSum
}

func marineCorridorBlendWeight(landTravel float64) float64 {
	travel := Clamp(landTravel, 0, 1)
	return Clamp(0.25+0.45*transportCorridorWeight(travel)+0.10*travel, 0.25, 0.75)
}
