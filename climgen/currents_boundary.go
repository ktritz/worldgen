package climgen

import (
	"math"
)

// =============================================================================
// BOUNDARY LAYER UTILITIES
// =============================================================================
// Functions for computing coastline properties and boundary currents.

// estimateCellSize computes the average angular distance between neighboring vertices.
// This allows us to convert angular distance targets to cell counts for BFS,
// making the algorithms resolution-independent.
func estimateCellSize(vertices []Vector3D, adj *FlatAdjacency) float64 {
	// Sample a few vertices to estimate average edge length
	totalDist := 0.0
	count := 0
	sampleSize := min(100, len(vertices))
	step := len(vertices) / sampleSize

	for i := 0; i < len(vertices) && count < sampleSize; i += step {
		for _, k := range adj.GetNeighbors(i) {
			if k >= 0 && k < len(vertices) {
				totalDist += AngularDistance(vertices[i], vertices[k])
				count++
			}
		}
	}

	if count == 0 {
		return 0.02 // Default fallback ~1 degree
	}
	return totalDist / float64(count)
}

// ComputeEasternBoundaryFetch computes the distance from the eastern boundary for each water vertex.
// In Sverdrup dynamics, Ψ builds up as you move westward from the eastern coast.
// Returns fetch values normalized to [0,1] where 0 = at eastern coast, 1 = far from eastern coast.
func ComputeEasternBoundaryFetch(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
) []float64 {
	numVertices := len(vertices)
	eastFetch := make([]float64, numVertices)

	// First, find all coastline vertices and determine if they're "eastern" coasts
	// Eastern coast = land is to the EAST of the water
	isWater := make([]bool, numVertices)
	isEasternCoast := make([]bool, numVertices)

	for i := 0; i < numVertices; i++ {
		isWater[i] = elevation[i] < seaLevelThreshold
		if !isWater[i] {
			continue
		}

		// Check if this water vertex is adjacent to land
		east, _ := GetTangentVectors(vertices[i])

		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= numVertices {
				continue
			}
			if elevation[k] >= seaLevelThreshold {
				// This is a land neighbor - which direction?
				toNeighbor := Sub(vertices[k], vertices[i])
				eastComponent := Dot(toNeighbor, east)
				if eastComponent > 0.001 { // Land is to the east
					isEasternCoast[i] = true
					break
				}
			}
		}
	}

	// BFS westward from eastern coastlines
	// We propagate distance, but prioritize westward movement
	eastDist := make([]int, numVertices)
	for i := range eastDist {
		eastDist[i] = -1
	}

	queue := make([]int, 0, numVertices)
	for i := 0; i < numVertices; i++ {
		if isEasternCoast[i] {
			eastDist[i] = 0
			queue = append(queue, i)
		}
	}

	// BFS to propagate distance from eastern boundaries
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, k := range adj.GetNeighbors(curr) {
			if k >= 0 && k < numVertices && isWater[k] && eastDist[k] == -1 {
				eastDist[k] = eastDist[curr] + 1
				queue = append(queue, k)
			}
		}
	}

	// Find max distance for normalization
	maxDist := 1
	for _, d := range eastDist {
		if d > maxDist {
			maxDist = d
		}
	}

	// Convert to normalized fetch (0 at east coast, 1 at max distance)
	for i := 0; i < numVertices; i++ {
		if eastDist[i] >= 0 {
			eastFetch[i] = float64(eastDist[i]) / float64(maxDist)
		}
	}

	return eastFetch
}

// ComputeWesternBoundaryLayer computes the western boundary layer for the entire domain.
// Returns a slice where each water vertex has a value 0-1 indicating how much
// western boundary intensification to apply (1 = strong boundary current zone).
func ComputeWesternBoundaryLayer(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
) []float64 {
	numVertices := len(vertices)
	westIntensity := make([]float64, numVertices)

	// First, find all coastline vertices
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

	// For each coastline vertex, determine if it's a "western" coast
	// Western coast = land is to the WEST of the water
	// This is where western boundary currents form
	isWesternCoast := make([]bool, numVertices)
	for i := 0; i < numVertices; i++ {
		if !isCoastline[i] {
			continue
		}

		east, _ := GetTangentVectors(vertices[i])

		// Check direction to land neighbors
		landToWest := false
		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= numVertices {
				continue
			}
			if elevation[k] >= seaLevelThreshold {
				// This is a land neighbor - which direction?
				toNeighbor := Sub(vertices[k], vertices[i])
				eastComponent := Dot(toNeighbor, east)
				if eastComponent < -0.001 { // Land is to the west
					landToWest = true
					break
				}
			}
		}
		isWesternCoast[i] = landToWest
	}

	// BFS from western coastlines to create boundary layer
	// Use angular distance for resolution independence
	// Target: ~0.15 radians ≈ 8.5 degrees ≈ 950 km on Earth
	const boundaryLayerAngular = 0.15 // radians
	cellSize := estimateCellSize(vertices, adj)
	boundaryLayerWidth := int(boundaryLayerAngular/cellSize) + 1
	if boundaryLayerWidth < 3 {
		boundaryLayerWidth = 3
	}

	westDist := make([]int, numVertices)
	for i := range westDist {
		westDist[i] = -1
	}

	queue := make([]int, 0, numVertices)
	for i := 0; i < numVertices; i++ {
		if isWesternCoast[i] {
			westDist[i] = 0
			queue = append(queue, i)
		}
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if westDist[curr] >= boundaryLayerWidth {
			continue
		}

		for _, k := range adj.GetNeighbors(curr) {
			if k >= 0 && k < numVertices && isWater[k] && westDist[k] == -1 {
				westDist[k] = westDist[curr] + 1
				queue = append(queue, k)
			}
		}
	}

	// Convert distance to Stommel layer profile
	// Create a "ridge" away from coast - the gradient between ridge and Ψ=0 at coast creates boundary current
	// Ridge parameters in angular distance, converted to cell counts
	const ridgePeakAngular = 0.08  // ~4.5 degrees from coast
	const ridgeWidthAngular = 0.05 // ~3 degrees width
	ridgePeak := ridgePeakAngular / cellSize
	ridgeWidth := ridgeWidthAngular / cellSize
	if ridgeWidth < 1.0 {
		ridgeWidth = 1.0
	}

	for i := 0; i < numVertices; i++ {
		if westDist[i] >= 0 && westDist[i] <= boundaryLayerWidth {
			d := float64(westDist[i])
			westIntensity[i] = math.Exp(-(d-ridgePeak)*(d-ridgePeak) / (ridgeWidth * ridgeWidth))
		}
	}

	return westIntensity
}

// CalculateCoastlineLandDirs pre-computes the tangent direction toward land for vertices near coastlines.
// Extends several rings out from the coast to ensure smooth boundary conditions.
// This is used for coast-parallel flow boundary conditions during smoothing.
func CalculateCoastlineLandDirs(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
) []Vector3D {
	numVertices := len(vertices)
	landDirs := make([]Vector3D, numVertices)

	// First pass: find immediate coastline vertices and their land direction
	coastDist := make([]int, numVertices) // Distance to nearest land (in edges)
	for i := range coastDist {
		coastDist[i] = -1 // -1 = not computed yet
	}

	// Mark land vertices
	for i := 0; i < numVertices; i++ {
		if elevation[i] >= seaLevelThreshold {
			coastDist[i] = 0
		}
	}

	// BFS to find distance from land for water vertices (up to 3 rings)
	maxRings := 3
	queue := make([]int, 0, numVertices)

	// Start with water vertices adjacent to land
	for i := 0; i < numVertices; i++ {
		if elevation[i] >= seaLevelThreshold {
			continue // Skip land
		}

		for _, k := range adj.GetNeighbors(i) {
			if k >= 0 && k < numVertices && elevation[k] >= seaLevelThreshold {
				coastDist[i] = 1
				queue = append(queue, i)
				break
			}
		}
	}

	// Propagate distance outward
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if coastDist[curr] >= maxRings {
			continue
		}

		for _, k := range adj.GetNeighbors(curr) {
			if k >= 0 && k < numVertices && coastDist[k] == -1 && elevation[k] < seaLevelThreshold {
				coastDist[k] = coastDist[curr] + 1
				queue = append(queue, k)
			}
		}
	}

	// Second pass: compute land direction for all near-coast vertices
	for i := 0; i < numVertices; i++ {
		if coastDist[i] <= 0 || coastDist[i] > maxRings {
			continue // Skip land and far-from-coast water
		}

		// Find nearest land vertices by looking at neighbors recursively
		// For simplicity, find the direction toward lower coastDist values
		var towardLandSum Vector3D
		towardLandCount := 0

		for _, k := range adj.GetNeighbors(i) {
			if k >= 0 && k < numVertices {
				if elevation[k] >= seaLevelThreshold {
					// Direct land neighbor
					diff := Sub(vertices[k], vertices[i])
					towardLandSum = Add(towardLandSum, diff)
					towardLandCount += 2 // Weight direct land more
				} else if coastDist[k] >= 0 && coastDist[k] < coastDist[i] {
					// Neighbor closer to land
					diff := Sub(vertices[k], vertices[i])
					towardLandSum = Add(towardLandSum, diff)
					towardLandCount++
				}
			}
		}

		if towardLandCount == 0 {
			continue
		}

		// Average and project onto tangent plane
		vecToLand := Scale(towardLandSum, 1.0/float64(towardLandCount))
		vecLen := Length(vecToLand)
		if vecLen < 1e-9 {
			continue
		}

		surfaceNormal := Normalize(vertices[i])
		dotNormal := Dot(vecToLand, surfaceNormal)
		landDirTangent := Sub(vecToLand, Scale(surfaceNormal, dotNormal))

		landDirLen := Length(landDirTangent)
		if landDirLen > 1e-6 {
			// Scale by distance - closer to coast = stronger constraint
			strength := 1.0 - float64(coastDist[i]-1)/float64(maxRings)
			landDirs[i] = Scale(landDirTangent, strength/landDirLen)
		}
	}

	return landDirs
}
