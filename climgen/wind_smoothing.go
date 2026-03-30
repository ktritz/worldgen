package climgen

// SmoothScalarField performs iterative diffusion on a scalar field.
func SmoothScalarField(
	field []float64,
	vertices []Vector3D,
	adj *FlatAdjacency,
	iterations int,
	factor float64,
) []float64 {
	numVertices := len(field)
	smoothed := make([]float64, numVertices)
	copy(smoothed, field)
	next := make([]float64, numVertices)

	for iter := 0; iter < iterations; iter++ {
		for i := 0; i < numVertices; i++ {
			sum := 0.0
			count := 0
			for _, k := range adj.GetNeighbors(i) {
				if k >= 0 && k < numVertices {
					sum += smoothed[k]
					count++
				}
			}
			if count > 0 {
				avg := sum / float64(count)
				next[i] = smoothed[i]*(1-factor) + avg*factor
			} else {
				next[i] = smoothed[i]
			}
		}
		smoothed, next = next, smoothed
	}

	return smoothed
}

// SmoothScalarFieldMasked smooths within a mask and does not diffuse through
// cells where mask=false.
func SmoothScalarFieldMasked(
	field []float64,
	mask []bool,
	adj *FlatAdjacency,
	iterations int,
	factor float64,
) []float64 {
	smoothed := make([]float64, len(field))
	copy(smoothed, field)
	next := make([]float64, len(field))

	for iter := 0; iter < iterations; iter++ {
		for i := range field {
			if i >= len(mask) || !mask[i] {
				next[i] = field[i]
				continue
			}
			sum := 0.0
			count := 0
			for _, k := range adj.GetNeighbors(i) {
				if k >= 0 && k < len(field) && k < len(mask) && mask[k] {
					sum += smoothed[k]
					count++
				}
			}
			if count > 0 {
				avg := sum / float64(count)
				next[i] = smoothed[i]*(1-factor) + avg*factor
			} else {
				next[i] = smoothed[i]
			}
		}
		smoothed, next = next, smoothed
	}

	return smoothed
}

// SmoothVectorField performs diffusion on a vector field, maintaining tangent-plane constraint.
func SmoothVectorField(
	field []Vector3D,
	vertices []Vector3D,
	adj *FlatAdjacency,
	iterations int,
	factor float64,
) []Vector3D {
	numVertices := len(field)
	smoothed := make([]Vector3D, numVertices)
	copy(smoothed, field)
	next := make([]Vector3D, numVertices)

	for iter := 0; iter < iterations; iter++ {
		for i := 0; i < numVertices; i++ {
			var sum Vector3D
			count := 0
			for _, k := range adj.GetNeighbors(i) {
				if k >= 0 && k < numVertices {
					sum = Add(sum, smoothed[k])
					count++
				}
			}
			if count > 0 {
				avg := Scale(sum, 1.0/float64(count))
				blended := Add(Scale(smoothed[i], 1.0-factor), Scale(avg, factor))

				normal := vertices[i]
				dotN := Dot(blended, normal)
				next[i] = Sub(blended, Scale(normal, dotN))
			} else {
				next[i] = smoothed[i]
			}
		}
		smoothed, next = next, smoothed
	}

	return smoothed
}

// SmoothVectorFieldBySurface smooths vectors preferentially within the same
// surface type so ocean winds stay coherent over water without smearing across
// coastlines.
func SmoothVectorFieldBySurface(
	field []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	iterations int,
	factor float64,
) []Vector3D {
	smoothed := make([]Vector3D, len(field))
	copy(smoothed, field)
	next := make([]Vector3D, len(field))

	for iter := 0; iter < iterations; iter++ {
		for i := range field {
			var sum Vector3D
			count := 0
			sameSurface := elevation[i] >= seaLevelThreshold

			for _, k := range adj.GetNeighbors(i) {
				if k < 0 || k >= len(field) {
					continue
				}
				if (elevation[k] >= seaLevelThreshold) != sameSurface {
					continue
				}
				sum = Add(sum, smoothed[k])
				count++
			}

			if count == 0 {
				for _, k := range adj.GetNeighbors(i) {
					if k >= 0 && k < len(field) {
						sum = Add(sum, smoothed[k])
						count++
					}
				}
			}

			if count > 0 {
				avg := Scale(sum, 1.0/float64(count))
				blended := Add(Scale(smoothed[i], 1.0-factor), Scale(avg, factor))
				normal := vertices[i]
				dotN := Dot(blended, normal)
				next[i] = Sub(blended, Scale(normal, dotN))
			} else {
				next[i] = smoothed[i]
			}
		}
		smoothed, next = next, smoothed
	}

	return smoothed
}
