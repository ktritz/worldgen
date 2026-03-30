package climgen

import "math"

// AddGeographicPressureAnomalies modifies the base zonal pressure field using
// broad ocean-basin and continental-interior context rather than a simple
// per-cell land/ocean toggle.
func AddGeographicPressureAnomalies(
	pressure []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	settings CirculationSettings,
) []float64 {
	numVertices := len(pressure)
	result := make([]float64, numVertices)
	copy(result, pressure)

	if settings.SubtropicalHighStrength == 0 &&
		settings.SubpolarLowStrength == 0 &&
		settings.ContinentalLowStrength == 0 {
		return result
	}

	landInterior := ComputeSurfaceInteriorFraction(elevation, seaLevelThreshold, adj, 1800.0, true)
	oceanExposure := ComputeSurfaceInteriorFraction(elevation, seaLevelThreshold, adj, 2600.0, false)

	subtropicalLat := settings.HadleyEdgeLat * math.Pi / 180
	subpolarLat := settings.FerrelEdgeLat * math.Pi / 180
	bandWidth := 15.0 * math.Pi / 180

	anomaly := make([]float64, numVertices)
	for i, v := range vertices {
		lat := effectiveCirculationLatitude(math.Asin(v.Y), settings)
		absLat := math.Abs(lat)
		isOcean := elevation[i] < seaLevelThreshold

		if isOcean {
			basinFactor := 0.15 + 0.85*math.Sqrt(oceanExposure[i])

			if settings.SubtropicalHighStrength > 0 {
				dist := absLat - subtropicalLat
				if math.Abs(dist) < bandWidth {
					shape := math.Cos(dist / bandWidth * math.Pi / 2)
					shape *= shape
					anomaly[i] += settings.SubtropicalHighStrength * shape * basinFactor
				}
			}

			if settings.SubpolarLowStrength > 0 {
				dist := absLat - subpolarLat
				if math.Abs(dist) < bandWidth {
					shape := math.Cos(dist / bandWidth * math.Pi / 2)
					shape *= shape
					anomaly[i] -= settings.SubpolarLowStrength * shape * basinFactor
				}
			}
			continue
		}

		if settings.ContinentalLowStrength > 0 && absLat < 60*math.Pi/180 {
			thermalBand := math.Max(0, math.Cos(lat))
			interiorFactor := 0.2 + 0.8*landInterior[i]
			anomaly[i] -= settings.ContinentalLowStrength * thermalBand * interiorFactor
		}
	}

	cellSize := estimateCellSize(vertices, adj)
	smoothIters := int(0.10/cellSize) + 1
	if smoothIters < 2 {
		smoothIters = 2
	}

	oceanMask := make([]bool, numVertices)
	landMask := make([]bool, numVertices)
	for i := range elevation {
		oceanMask[i] = elevation[i] < seaLevelThreshold
		landMask[i] = !oceanMask[i]
	}

	smoothedOcean := SmoothScalarFieldMasked(anomaly, oceanMask, adj, smoothIters, 0.42)
	smoothedLand := SmoothScalarFieldMasked(anomaly, landMask, adj, smoothIters, 0.35)
	for i := range result {
		if oceanMask[i] {
			result[i] += smoothedOcean[i] * settings.PressureStrength
		} else {
			result[i] += smoothedLand[i] * settings.PressureStrength
		}
	}

	return result
}

// ComputePressureGradientWind computes surface-wind perturbations from pressure
// gradients, using a friction-modified geostrophic blend.
func ComputePressureGradientWind(
	pressure []float64,
	vertices []Vector3D,
	adj *FlatAdjacency,
	strength float64,
) []Vector3D {
	numVertices := len(pressure)
	wind := make([]Vector3D, numVertices)

	for i := range vertices {
		normal := vertices[i]
		east, north := GetTangentVectors(vertices[i])

		var gradE, gradN float64
		var totalWeight float64

		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= numVertices {
				continue
			}

			diff := Sub(vertices[k], vertices[i])
			dotN := Dot(diff, normal)
			tangentDiff := Sub(diff, Scale(normal, dotN))

			de := Dot(tangentDiff, east)
			dn := Dot(tangentDiff, north)
			dist := math.Sqrt(de*de + dn*dn)
			if dist < 1e-12 {
				continue
			}

			dp := pressure[k] - pressure[i]
			weight := 1.0 / dist
			gradE += weight * dp * de / (dist * dist)
			gradN += weight * dp * dn / (dist * dist)
			totalWeight += weight
		}

		if totalWeight < 1e-12 {
			continue
		}
		gradE /= totalWeight
		gradN /= totalWeight

		lat := math.Asin(vertices[i].Y)
		sinLat := math.Sin(lat)
		coriolisFactor := math.Abs(sinLat)
		if coriolisFactor < 0.2 {
			coriolisFactor = 0.2
		}

		downGradE := -gradE
		downGradN := -gradN

		hemisphere := 1.0
		if sinLat < 0 {
			hemisphere = -1.0
		}
		geoE := downGradN * hemisphere
		geoN := -downGradE * hemisphere

		geoWeight := 0.65 * coriolisFactor
		crossWeight := 1.0 - geoWeight
		latScale := math.Sqrt(coriolisFactor)

		windE := (geoE*geoWeight + downGradE*crossWeight) * strength * latScale
		windN := (geoN*geoWeight + downGradN*crossWeight) * strength * latScale
		wind[i] = Add(Scale(east, windE), Scale(north, windN))
	}

	return wind
}

// ComputeSurfaceInteriorFraction estimates how deep a point lies within a land
// or ocean region, normalized to [0,1].
func ComputeSurfaceInteriorFraction(
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	maxDistanceKm float64,
	targetLand bool,
) []float64 {
	n := len(elevation)
	distance := make([]int, n)
	for i := range distance {
		distance[i] = -1
	}

	queue := make([]int, 0, n/8)
	isTarget := func(i int) bool {
		if targetLand {
			return elevation[i] >= seaLevelThreshold
		}
		return elevation[i] < seaLevelThreshold
	}
	isOpposite := func(i int) bool { return !isTarget(i) }

	for i := 0; i < n; i++ {
		if !isTarget(i) {
			continue
		}
		for _, k := range adj.GetNeighbors(i) {
			if k >= 0 && k < n && isOpposite(k) {
				distance[i] = 0
				queue = append(queue, i)
				break
			}
		}
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, k := range adj.GetNeighbors(curr) {
			if k < 0 || k >= n || !isTarget(k) || distance[k] >= 0 {
				continue
			}
			distance[k] = distance[curr] + 1
			queue = append(queue, k)
		}
	}

	earthRadiusKm := 6371.0
	avgCellSizeKm := earthRadiusKm * math.Sqrt(4*math.Pi/float64(n))
	field := make([]float64, n)
	for i := 0; i < n; i++ {
		if !isTarget(i) {
			continue
		}
		if distance[i] < 0 {
			field[i] = 1
			continue
		}
		field[i] = Clamp(float64(distance[i])*avgCellSizeKm/maxDistanceKm, 0, 1)
	}
	return field
}
