package climgen

const precipInlandTransportSteps = 10

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
	for step := 0; step < maxSteps; step++ {
		next, upwindness := strongestUpwindNeighbor(current, vertices, adj, wind)
		if next < 0 || upwindness <= 0.05 || next >= len(specificHumidity) {
			break
		}
		weight := upwindness / float64(step+1)
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
