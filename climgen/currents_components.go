package climgen

import "math"

// OceanComponent is a connected water body used to scope wind-driven fetch
// and expose basin-like diagnostics for visualization.
type OceanComponent struct {
	ID       int
	Vertices []int
}

// FindOceanComponents identifies connected water bodies.
func FindOceanComponents(
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
) ([]int, []OceanComponent) {
	assignments := make([]int, len(elevation))
	for i := range assignments {
		assignments[i] = -1
	}

	var components []OceanComponent
	queue := make([]int, 0, 128)

	for start := range elevation {
		if elevation[start] >= seaLevelThreshold || assignments[start] >= 0 {
			continue
		}

		component := OceanComponent{ID: len(components)}
		assignments[start] = component.ID
		queue = append(queue[:0], start)

		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			component.Vertices = append(component.Vertices, curr)

			for _, k := range adj.GetNeighbors(curr) {
				if k < 0 || k >= len(elevation) {
					continue
				}
				if elevation[k] >= seaLevelThreshold || assignments[k] >= 0 {
					continue
				}
				assignments[k] = component.ID
				queue = append(queue, k)
			}
		}

		components = append(components, component)
	}

	return assignments, components
}

// BuildBasinsFromComponents converts connected water bodies into Basin records
// for diagnostics and visualization.
func BuildBasinsFromComponents(
	vertices []Vector3D,
	components []OceanComponent,
	polarLimitDeg float64,
) []Basin {
	basins := make([]Basin, 0, len(components))
	for _, component := range components {
		if len(component.Vertices) == 0 {
			continue
		}

		centroid := calculateCentroid(component.Vertices, vertices)
		basins = append(basins, Basin{
			ID:        component.ID,
			Zone:      GetLatitudeZone(centroid, polarLimitDeg),
			Centroid:  centroid,
			MaxRadius: calculateMaxRadius(component.Vertices, vertices, centroid),
			Vertices:  component.Vertices,
		})
	}
	return basins
}

// FilterComponentsBySize keeps only components large enough to treat as
// basin-level features in diagnostics and visualization, and remaps
// assignments accordingly.
func FilterComponentsBySize(
	componentAssignments []int,
	components []OceanComponent,
	minSize int,
) ([]int, []OceanComponent) {
	if minSize < 1 {
		minSize = 1
	}

	oldToNew := make(map[int]int, len(components))
	filtered := make([]OceanComponent, 0, len(components))
	for _, component := range components {
		if len(component.Vertices) < minSize {
			continue
		}
		oldToNew[component.ID] = len(filtered)
		filtered = append(filtered, OceanComponent{
			ID:       len(filtered),
			Vertices: component.Vertices,
		})
	}

	assignments := make([]int, len(componentAssignments))
	for i, id := range componentAssignments {
		assignments[i] = -1
		if newID, ok := oldToNew[id]; ok {
			assignments[i] = newID
		}
	}

	return assignments, filtered
}

// BuildComponentScaleField attenuates current strength in very small or narrow
// enclosed water bodies, which should not behave like open-ocean gyre domains.
func BuildComponentScaleField(
	vertices []Vector3D,
	componentAssignments []int,
	components []OceanComponent,
) []float64 {
	if len(componentAssignments) != len(vertices) {
		return nil
	}

	scales := make([]float64, len(components))
	for _, component := range components {
		if len(component.Vertices) == 0 {
			continue
		}

		centroid := calculateCentroid(component.Vertices, vertices)
		radius := calculateMaxRadius(component.Vertices, vertices, centroid)

		switch {
		case len(component.Vertices) < 8 || radius < 0.03:
			scales[component.ID] = 0
		default:
			countScale := Clamp((float64(len(component.Vertices))-24.0)/232.0, 0, 1)
			radiusScale := Clamp((radius-0.05)/0.17, 0, 1)
			scale := math.Sqrt(countScale * radiusScale)
			scales[component.ID] = 0.15 + 0.85*scale
		}
	}

	field := make([]float64, len(componentAssignments))
	for i, id := range componentAssignments {
		field[i] = 1
		if id >= 0 && id < len(scales) {
			field[i] = scales[id]
		}
	}
	return field
}

// ComputeEasternBoundaryFetchByComponent normalizes fetch separately inside each
// connected water body, preventing inland seas and gateway-separated oceans
// from sharing one global east-west distance scale.
func ComputeEasternBoundaryFetchByComponent(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	componentAssignments []int,
	components []OceanComponent,
) []float64 {
	if len(componentAssignments) != len(vertices) || len(components) == 0 {
		return ComputeEasternBoundaryFetch(vertices, elevation, seaLevelThreshold, adj)
	}

	eastFetch := make([]float64, len(vertices))
	eastDist := make([]int, len(vertices))
	for i := range eastDist {
		eastDist[i] = -1
	}

	queue := make([]int, 0, 256)

	for _, component := range components {
		if len(component.Vertices) == 0 {
			continue
		}

		seedCount := 0
		queue = queue[:0]
		for _, idx := range component.Vertices {
			if isEasternBoundaryVertex(idx, component.ID, vertices, elevation, seaLevelThreshold, adj, componentAssignments) {
				eastDist[idx] = 0
				queue = append(queue, idx)
				seedCount++
			}
		}

		if seedCount == 0 {
			componentSeeds := fallbackEasternSeeds(vertices, component)
			for _, idx := range componentSeeds {
				eastDist[idx] = 0
				queue = append(queue, idx)
			}
		}

		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			for _, k := range adj.GetNeighbors(curr) {
				if k < 0 || k >= len(vertices) {
					continue
				}
				if componentAssignments[k] != component.ID || eastDist[k] >= 0 {
					continue
				}
				eastDist[k] = eastDist[curr] + 1
				queue = append(queue, k)
			}
		}

		maxDist := 1
		for _, idx := range component.Vertices {
			if eastDist[idx] > maxDist {
				maxDist = eastDist[idx]
			}
		}
		for _, idx := range component.Vertices {
			if eastDist[idx] >= 0 {
				eastFetch[idx] = float64(eastDist[idx]) / float64(maxDist)
			}
		}
	}

	return eastFetch
}

func isEasternBoundaryVertex(
	idx int,
	componentID int,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	componentAssignments []int,
) bool {
	east, _ := GetTangentVectors(vertices[idx])
	for _, k := range adj.GetNeighbors(idx) {
		if k < 0 || k >= len(vertices) {
			continue
		}
		if elevation[k] < seaLevelThreshold && componentAssignments[k] == componentID {
			continue
		}
		toNeighbor := Sub(vertices[k], vertices[idx])
		if Dot(toNeighbor, east) > 0.001 {
			return true
		}
	}
	return false
}

func fallbackEasternSeeds(vertices []Vector3D, component OceanComponent) []int {
	centroid := calculateCentroid(component.Vertices, vertices)
	east, _ := GetTangentVectors(centroid)
	maxProj := -math.MaxFloat64
	for _, idx := range component.Vertices {
		proj := Dot(vertices[idx], east)
		if proj > maxProj {
			maxProj = proj
		}
	}

	seeds := make([]int, 0, 8)
	const tol = 0.03
	for _, idx := range component.Vertices {
		if maxProj-Dot(vertices[idx], east) <= tol {
			seeds = append(seeds, idx)
		}
	}
	if len(seeds) == 0 && len(component.Vertices) > 0 {
		seeds = append(seeds, component.Vertices[0])
	}
	return seeds
}
