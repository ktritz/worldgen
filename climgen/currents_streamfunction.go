package climgen

import (
	"math"
)

// =============================================================================
// STREAMFUNCTION-BASED CURRENT GENERATION
// =============================================================================
// This approach guarantees divergence-free flow by construction:
// 1. Define scalar streamfunction Ψ at each vertex
// 2. Compute velocity as v = n × ∇Ψ (perpendicular to gradient)
// 3. Set Ψ = 0 at coastlines → flow automatically parallel to coast

// GenerateWindDrivenStreamfunction creates a streamfunction from wind stress curl.
// This is the Sverdrup model: wind stress curl drives interior flow, with
// western boundary intensification to close the circulation.
//
// Wind zones (Earth-like):
//   - Trade winds (0-30°): blow westward, τx < 0
//   - Westerlies (30-60°): blow eastward, τx > 0
//   - Polar easterlies (60-90°): blow westward, τx < 0
//
// Wind stress curl drives gyres:
//   - Subtropical (15-45°): positive curl → clockwise NH, counter-clockwise SH
//   - Subpolar (45-70°): negative curl → counter-clockwise NH, clockwise SH
func GenerateWindDrivenStreamfunction(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	strength float64,
) []float64 {
	numVertices := len(vertices)
	psi := make([]float64, numVertices)

	// Identify coastline vertices
	isWater := make([]bool, numVertices)
	isCoastline := make([]bool, numVertices)
	for i := 0; i < numVertices; i++ {
		isWater[i] = elevation[i] < seaLevelThreshold
		if !isWater[i] {
			continue
		}
		for _, k := range adj.GetNeighbors(i) {
			if k >= 0 && k < numVertices && elevation[k] >= seaLevelThreshold {
				isCoastline[i] = true
				break
			}
		}
	}

	// Compute fetch from EASTERN boundary (proper Sverdrup dynamics)
	// Ψ starts at 0 on eastern coast and builds westward
	eastFetch := ComputeEasternBoundaryFetch(vertices, elevation, seaLevelThreshold, adj)

	// Compute wind stress curl contribution to streamfunction
	// Sverdrup balance: β * v = curl(τ) / ρH
	// Integrated westward: Ψ ∝ curl(τ) / β * (distance from eastern boundary)

	for i := 0; i < numVertices; i++ {
		if !isWater[i] || isCoastline[i] {
			continue
		}

		v := vertices[i]
		lat := math.Asin(v.Y) // Latitude in radians

		// Wind stress curl: sin(3φ) * cos(φ) pattern
		// The cos(φ) taper weakens wind-driven circulation at poles
		// Positive in subtropical bands (~15-45°), negative in subpolar bands (~45-70°)
		windCurl := math.Sin(3.0*lat) * math.Cos(lat)

		// Beta parameter (Coriolis gradient) - larger near equator
		// β = 2Ω*cos(φ)/R, simplified version
		cosLat := math.Cos(lat)
		if cosLat < 0.1 {
			cosLat = 0.1 // Avoid division issues near poles
		}
		beta := cosLat

		// Sverdrup streamfunction: Ψ ∝ curl(τ) / β * fetch_from_east
		// This naturally gives Ψ=0 at eastern boundary, max Ψ in western interior
		psi[i] = strength * windCurl / beta * eastFetch[i]
	}

	return psi
}

// SmoothStreamfunction diffuses the streamfunction while preserving coastline boundary conditions.
func SmoothStreamfunction(
	psi []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	iterations int,
	factor float64,
) []float64 {
	numVertices := len(vertices)

	// Identify coastlines
	isCoastline := make([]bool, numVertices)
	for i := 0; i < numVertices; i++ {
		if elevation[i] >= seaLevelThreshold {
			continue
		}
		for _, k := range adj.GetNeighbors(i) {
			if k >= 0 && k < numVertices && elevation[k] >= seaLevelThreshold {
				isCoastline[i] = true
				break
			}
		}
	}

	smoothed := make([]float64, numVertices)
	copy(smoothed, psi)
	next := make([]float64, numVertices)

	for iter := 0; iter < iterations; iter++ {
		for i := 0; i < numVertices; i++ {
			if elevation[i] >= seaLevelThreshold {
				continue // Skip land
			}
			if isCoastline[i] {
				next[i] = 0 // Enforce Ψ = 0 at coastlines
				continue
			}

			// Average with neighbors
			sum := 0.0
			count := 0
			for _, k := range adj.GetNeighbors(i) {
				if k >= 0 && k < numVertices && elevation[k] < seaLevelThreshold {
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

// ComputeVelocityFromStreamfunction computes velocity as v = n × ∇Ψ
// This guarantees divergence-free flow by construction.
func ComputeVelocityFromStreamfunction(
	psi []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
) []Vector3D {
	numVertices := len(vertices)
	velocity := make([]Vector3D, numVertices)

	// Compute coast distance for dampening using BFS (efficient)
	// Then convert cell distance to approximate angular distance
	cellSize := estimateCellSize(vertices, adj)
	const dampeningAngular = 0.03 // radians ≈ 1.7 degrees
	dampeningCells := int(dampeningAngular/cellSize) + 1
	if dampeningCells < 2 {
		dampeningCells = 2
	}

	coastDist := make([]int, numVertices)
	for i := range coastDist {
		coastDist[i] = -1
	}
	queue := make([]int, 0)
	for i := 0; i < numVertices; i++ {
		if elevation[i] >= seaLevelThreshold {
			continue
		}
		for _, k := range adj.GetNeighbors(i) {
			if k >= 0 && k < numVertices && elevation[k] >= seaLevelThreshold {
				coastDist[i] = 0
				queue = append(queue, i)
				break
			}
		}
	}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		if coastDist[curr] >= dampeningCells {
			continue
		}
		for _, k := range adj.GetNeighbors(curr) {
			if k >= 0 && k < numVertices && elevation[k] < seaLevelThreshold && coastDist[k] == -1 {
				coastDist[k] = coastDist[curr] + 1
				queue = append(queue, k)
			}
		}
	}

	for i := 0; i < numVertices; i++ {
		if elevation[i] >= seaLevelThreshold {
			continue // Skip land
		}

		// Compute gradient of Ψ using neighbors (least-squares fit)
		// Gradient in tangent plane at vertex i
		normal := vertices[i] // Unit sphere, vertex IS the normal
		east, north := GetTangentVectors(vertices[i])

		// Accumulate gradient components
		var sumDpsiDe, sumDpsiDn float64
		var sumWe, sumWn float64
		waterNeighbors := 0

		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= numVertices || elevation[k] >= seaLevelThreshold {
				continue
			}
			waterNeighbors++

			// Vector from i to k, projected onto tangent plane
			diff := Sub(vertices[k], vertices[i])
			dotN := Dot(diff, normal)
			tangentDiff := Sub(diff, Scale(normal, dotN))

			// Components in east/north directions
			de := Dot(tangentDiff, east)
			dn := Dot(tangentDiff, north)
			dist := math.Sqrt(de*de + dn*dn)

			if dist < 1e-12 {
				continue
			}

			// Ψ difference
			dpsi := psi[k] - psi[i]

			// Weighted contribution to gradient (inverse distance weighting)
			weight := 1.0 / dist
			sumDpsiDe += weight * dpsi * de / dist
			sumDpsiDn += weight * dpsi * dn / dist
			sumWe += weight * de * de / (dist * dist)
			sumWn += weight * dn * dn / (dist * dist)
		}

		// Gradient components
		var gradE, gradN float64
		if sumWe > 1e-12 {
			gradE = sumDpsiDe / sumWe
		}
		if sumWn > 1e-12 {
			gradN = sumDpsiDn / sumWn
		}

		// Velocity = n × ∇Ψ = n × (gradE * east + gradN * north)
		// = gradE * (n × east) + gradN * (n × north)
		// = gradE * north - gradN * east  (using right-hand rule on unit sphere)
		vel := Add(Scale(north, gradE), Scale(east, -gradN))

		// Coastal dampening: reduce velocity near coastlines (friction + numerical stability)
		// Uses BFS cell distance converted to approximate angular distance for resolution independence
		if coastDist[i] >= 0 && coastDist[i] < dampeningCells {
			// Ramp from 0.2 at coast to 1.0 at dampeningCells
			t := float64(coastDist[i]) / float64(dampeningCells)
			dampen := 0.2 + 0.8*t
			vel = Scale(vel, dampen)
		}

		velocity[i] = vel
	}

	return velocity
}

// SmoothVelocityField applies diffusion to the velocity field to remove spikes.
// Unlike Ψ smoothing, this operates on vectors and preserves tangent-plane constraint.
// Isolated spikes get averaged away quickly; coherent bulk currents survive.
func SmoothVelocityField(
	velocity []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	iterations int,
	factor float64,
) []Vector3D {
	numVertices := len(vertices)

	smoothed := make([]Vector3D, numVertices)
	copy(smoothed, velocity)
	next := make([]Vector3D, numVertices)

	for iter := 0; iter < iterations; iter++ {
		for i := 0; i < numVertices; i++ {
			if elevation[i] >= seaLevelThreshold {
				continue // Skip land
			}

			// Average with water neighbors
			var sum Vector3D
			count := 0
			for _, k := range adj.GetNeighbors(i) {
				if k >= 0 && k < numVertices && elevation[k] < seaLevelThreshold {
					sum = Add(sum, smoothed[k])
					count++
				}
			}

			if count == 0 {
				next[i] = smoothed[i]
				continue
			}

			avg := Scale(sum, 1.0/float64(count))
			blended := Add(Scale(smoothed[i], 1.0-factor), Scale(avg, factor))

			// Project back onto tangent plane (velocity must be tangent to sphere)
			normal := vertices[i]
			dotN := Dot(blended, normal)
			tangent := Sub(blended, Scale(normal, dotN))

			next[i] = tangent
		}
		smoothed, next = next, smoothed
	}

	return smoothed
}
