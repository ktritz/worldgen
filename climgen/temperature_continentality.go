package climgen

import "math"

// =============================================================================
// TEMPERATURE - CONTINENTALITY EFFECTS
// =============================================================================
// Computes continentality index and applies temperature adjustments for:
//   - Distance from ocean (continental interiors are more extreme)
//   - Marine influence (coasts are moderated by ocean)

// ComputeContinentality calculates a continentality index for each vertex.
// Higher values = more continental (larger temperature extremes, less ocean influence).
// Based on distance from nearest ocean, weighted by latitude.
//
// Returns values from 0 (coastal/ocean) to 1 (deep continental interior).
func ComputeContinentality(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	maxDistanceKm float64,
) []float64 {
	numVertices := len(vertices)
	continentality := make([]float64, numVertices)

	// First pass: identify coastal land (distance = 0)
	distanceFromCoast := make([]float64, numVertices)
	for i := range distanceFromCoast {
		distanceFromCoast[i] = -1 // Unvisited
	}

	// BFS from all coastal cells
	queue := make([]int, 0, numVertices/10)

	for i := 0; i < numVertices; i++ {
		if elevation[i] < seaLevelThreshold {
			distanceFromCoast[i] = 0
			continentality[i] = 0
		} else {
			// Check if coastal land (has ocean neighbor)
			for _, k := range adj.GetNeighbors(i) {
				if k >= 0 && k < numVertices && elevation[k] < seaLevelThreshold {
					distanceFromCoast[i] = 0
					queue = append(queue, i)
					break
				}
			}
		}
	}

	// BFS to compute distance from coast for all land cells
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, k := range adj.GetNeighbors(current) {
			if k >= 0 && k < numVertices && distanceFromCoast[k] < 0 {
				distanceFromCoast[k] = distanceFromCoast[current] + 1
				queue = append(queue, k)
			}
		}
	}

	// Convert hop distance to km
	earthRadius := 6371.0
	avgCellSizeKm := earthRadius * math.Sqrt(4*math.Pi/float64(numVertices))

	for i := 0; i < numVertices; i++ {
		if elevation[i] < seaLevelThreshold {
			continue
		}

		distKm := distanceFromCoast[i] * avgCellSizeKm
		if distKm < 0 {
			distKm = 0
		}

		// Continentality increases with distance, saturating at maxDistanceKm
		continentality[i] = math.Min(distKm/maxDistanceKm, 1.0)

		// Stronger at higher latitudes
		lat := math.Abs(getLatitude(vertices[i]))
		latFactor := 0.5 + 0.5*math.Sin(lat)
		continentality[i] *= latFactor
	}

	return continentality
}

// ApplyContinentalityEffect adjusts temperatures based on continentality.
// Continental interiors are:
//   - Warmer in tropics (less evaporative cooling)
//   - Colder at high latitudes (harsh winters dominate)
func ApplyContinentalityEffect(
	temperature []float64,
	vertices []Vector3D,
	continentality []float64,
	strength float64,
) []float64 {
	numVertices := len(temperature)
	adjusted := make([]float64, numVertices)

	for i := 0; i < numVertices; i++ {
		c := continentality[i]
		if c < 1e-6 {
			adjusted[i] = temperature[i]
			continue
		}

		lat := getLatitude(vertices[i])
		absLat := math.Abs(lat)

		var tempAdjust float64
		if absLat < 20.0*math.Pi/180.0 {
			// Tropics: warming
			tempAdjust = strength * 0.5 * c
		} else if absLat < 50.0*math.Pi/180.0 {
			// Mid-latitudes: slight warming
			t := (absLat - 20.0*math.Pi/180.0) / (30.0 * math.Pi / 180.0)
			tempAdjust = strength * (0.5 - 0.7*t) * c
		} else {
			// High latitudes: cooling
			t := (absLat - 50.0*math.Pi/180.0) / (40.0 * math.Pi / 180.0)
			tempAdjust = -strength * (0.2 + 0.8*t) * c
		}

		adjusted[i] = temperature[i] + tempAdjust
	}

	return adjusted
}

// ApplyMarineInfluence pulls coastal temperatures toward ocean values.
// Creates maritime climate with moderate temperatures near coasts.
// distanceKm specifies the marine influence radius in kilometers.
func ApplyMarineInfluence(
	temperature []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	strength float64,
	distanceKm float64,
) []float64 {
	numVertices := len(temperature)
	influenced := make([]float64, numVertices)
	copy(influenced, temperature)

	// Compute cell size and convert km to cell hops
	earthRadius := 6371.0
	avgCellSizeKm := earthRadius * math.Sqrt(4*math.Pi/float64(numVertices))
	distanceCells := int(math.Ceil(distanceKm / avgCellSizeKm))
	if distanceCells < 1 {
		distanceCells = 1
	}

	for iter := 0; iter < distanceCells; iter++ {
		next := make([]float64, numVertices)
		copy(next, influenced)

		decayFactor := strength * float64(distanceCells-iter) / float64(distanceCells)

		for i := 0; i < numVertices; i++ {
			if elevation[i] < seaLevelThreshold {
				continue
			}

			neighbors := adj.GetNeighbors(i)
			oceanSum := 0.0
			oceanCount := 0

			for _, k := range neighbors {
				if k >= 0 && k < numVertices && elevation[k] < seaLevelThreshold {
					oceanSum += temperature[k]
					oceanCount++
				}
			}

			if oceanCount > 0 {
				oceanAvg := oceanSum / float64(oceanCount)
				next[i] = influenced[i]*(1.0-decayFactor) + oceanAvg*decayFactor
			}
		}

		influenced = next
	}

	return influenced
}
