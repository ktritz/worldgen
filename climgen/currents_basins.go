package climgen

import (
	"fmt"
)

// BuildFlatAdjacency converts VoronoiCell neighbor data to flat adjacency arrays.
// This format is optimized for cache-friendly iteration in hot loops.
func BuildFlatAdjacency(cells []VoronoiCell) *FlatAdjacency {
	numVertices := len(cells)
	offsets := make([]int, numVertices+1)

	// First pass: count total neighbors and build offsets
	totalNeighbors := 0
	for i, cell := range cells {
		offsets[i] = totalNeighbors
		totalNeighbors += len(cell.NeighborSiteIndices)
	}
	offsets[numVertices] = totalNeighbors

	// Second pass: fill neighbor array
	neighbors := make([]int, totalNeighbors)
	pos := 0
	for _, cell := range cells {
		for _, neighIdx := range cell.NeighborSiteIndices {
			neighbors[pos] = int(neighIdx)
			pos++
		}
	}

	return &FlatAdjacency{
		Neighbors: neighbors,
		Offsets:   offsets,
	}
}

// FindOceanBasins identifies ocean basins using latitude zones and spherical cap fitting.
// It partitions water vertices by latitude zone, then iteratively fits spherical caps
// to find coherent ocean regions.
//
// FUTURE USE: This function is no longer used for ocean current generation (which now
// uses wind-driven Sverdrup model). It is preserved for future features:
//   - Ocean body identification (Atlantic, Pacific, etc.)
//   - Naming and labeling separate seas/oceans
//   - Regional climate properties (temperature, salinity basins)
//   - Visualization (coloring different ocean bodies)
func FindOceanBasins(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	settings BasinSettings,
) ([]int, []Basin) {
	numVertices := len(vertices)

	if settings.Verbose {
		fmt.Println("=== Ocean Basin Finding ===")
	}

	// Step 1: Partition water vertices by latitude zone
	zoneVertices := partitionByZone(vertices, elevation, seaLevelThreshold, settings)

	if settings.Verbose {
		for zone := LatitudeZone(1); zone <= LatitudeZone(settings.NumZones); zone++ {
			fmt.Printf("  Zone %s: %d water vertices\n", zone, len(zoneVertices[zone]))
		}
	}

	// Step 2: Iteratively fit basins within each zone
	basinAssignments := make([]int, numVertices)
	for i := range basinAssignments {
		basinAssignments[i] = -1 // Unassigned
	}

	var basins []Basin
	basinID := 0
	allAssigned := make(map[int]bool)

	for zone := LatitudeZone(1); zone <= LatitudeZone(settings.NumZones); zone++ {
		unassigned := copySet(zoneVertices[zone])
		if len(unassigned) == 0 {
			continue
		}

		if settings.Verbose {
			fmt.Printf("  Processing zone %s...\n", zone)
		}

		iteration := 0
		for len(unassigned) > 0 {
			iteration++

			// Find largest connected component in unassigned water
			component := findLargestComponent(unassigned, adj, numVertices)
			if len(component) < settings.MinComponentSize {
				if settings.Verbose {
					fmt.Printf("    Remaining component too small (%d < %d), done with zone\n",
						len(component), settings.MinComponentSize)
				}
				break
			}

			// Find seed vertex (farthest from boundaries)
			seed := findFarthestFromBoundary(
				component, vertices, elevation, seaLevelThreshold,
				zone, allAssigned, adj, numVertices, settings,
			)

			if seed < 0 {
				break
			}

			// Fit spherical cap from seed
			capVertices := fitSphericalCap(
				seed, unassigned, vertices, elevation, seaLevelThreshold,
				zone, allAssigned, adj, numVertices, settings,
			)

			if len(capVertices) == 0 {
				break
			}

			// Calculate basin properties
			centroid := calculateCentroid(capVertices, vertices)
			maxRadius := calculateMaxRadius(capVertices, vertices, centroid)

			if maxRadius < settings.MinBasinRadiusRad {
				if settings.Verbose {
					fmt.Printf("    Fitted basin too small (radius %.3f < %.3f), discarding\n",
						maxRadius, settings.MinBasinRadiusRad)
				}
				break
			}

			// Create basin
			basin := Basin{
				ID:        basinID,
				Zone:      zone,
				Centroid:  centroid,
				MaxRadius: maxRadius,
				Vertices:  capVertices,
			}
			basins = append(basins, basin)

			// Mark vertices as assigned
			for _, idx := range capVertices {
				basinAssignments[idx] = basinID
				allAssigned[idx] = true
				delete(unassigned, idx)
			}

			if settings.Verbose {
				fmt.Printf("    Basin %d: %d vertices, radius %.3f rad\n",
					basinID, len(capVertices), maxRadius)
			}

			basinID++
		}
	}

	if settings.Verbose {
		fmt.Printf("  Found %d basins total\n", len(basins))
	}

	return basinAssignments, basins
}

// partitionByZone divides water vertices into latitude zones.
func partitionByZone(
	vertices []Vector3D,
	elevation []float64,
	threshold float64,
	settings BasinSettings,
) map[LatitudeZone]map[int]bool {
	zones := make(map[LatitudeZone]map[int]bool)
	for z := LatitudeZone(1); z <= LatitudeZone(settings.NumZones); z++ {
		zones[z] = make(map[int]bool)
	}

	for i, v := range vertices {
		if elevation[i] >= threshold {
			continue // Skip land
		}
		zone := GetLatitudeZone(v, settings.PolarLimitDeg)
		zones[zone][i] = true
	}

	return zones
}

// findLargestComponent finds the largest connected component in the given vertex set.
func findLargestComponent(vertexSet map[int]bool, adj *FlatAdjacency, numVertices int) []int {
	visited := make([]bool, numVertices)
	var largest []int

	for startNode := range vertexSet {
		if visited[startNode] {
			continue
		}

		// BFS from this node
		component := bfs(startNode, vertexSet, adj, visited)

		if len(component) > len(largest) {
			largest = component
		}
	}

	return largest
}

// bfs performs breadth-first search and returns all reachable vertices in the set.
func bfs(start int, vertexSet map[int]bool, adj *FlatAdjacency, visited []bool) []int {
	queue := []int{start}
	visited[start] = true
	component := []int{start}

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]

		for _, v := range adj.GetNeighbors(u) {
			if !visited[v] && vertexSet[v] {
				visited[v] = true
				queue = append(queue, v)
				component = append(component, v)
			}
		}
	}

	return component
}

// findFarthestFromBoundary finds the vertex in component farthest from zone/land boundaries.
func findFarthestFromBoundary(
	component []int,
	vertices []Vector3D,
	elevation []float64,
	threshold float64,
	zone LatitudeZone,
	assigned map[int]bool,
	adj *FlatAdjacency,
	numVertices int,
	settings BasinSettings,
) int {
	if len(component) == 0 {
		return -1
	}

	// Build set for quick lookup
	inComponent := make([]bool, numVertices)
	for _, idx := range component {
		inComponent[idx] = true
	}

	// Find boundary vertices (adjacent to land, different zone, or already assigned)
	var boundaryNodes []int
	for _, u := range component {
		isBoundary := false
		for _, v := range adj.GetNeighbors(u) {
			if !inComponent[v] {
				// Check what type of boundary
				if v >= 0 && v < numVertices {
					if elevation[v] >= threshold ||
						GetLatitudeZone(vertices[v], settings.PolarLimitDeg) != zone ||
						assigned[v] {
						isBoundary = true
						break
					}
				}
			}
		}
		if isBoundary {
			boundaryNodes = append(boundaryNodes, u)
		}
	}

	if len(boundaryNodes) == 0 {
		return component[0]
	}

	// Multi-source BFS from boundary nodes to find farthest interior node
	dist := make([]int, numVertices)
	for i := range dist {
		dist[i] = -1
	}

	queue := boundaryNodes
	for _, node := range boundaryNodes {
		dist[node] = 0
	}

	farthestNode := boundaryNodes[0]
	maxDist := 0

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]

		if dist[u] > maxDist {
			maxDist = dist[u]
			farthestNode = u
		}

		for _, v := range adj.GetNeighbors(u) {
			if inComponent[v] && dist[v] == -1 {
				dist[v] = dist[u] + 1
				queue = append(queue, v)
			}
		}
	}

	return farthestNode
}

// fitSphericalCap expands a spherical cap from seed vertex within the zone.
func fitSphericalCap(
	seed int,
	available map[int]bool,
	vertices []Vector3D,
	elevation []float64,
	threshold float64,
	zone LatitudeZone,
	assigned map[int]bool,
	adj *FlatAdjacency,
	numVertices int,
	settings BasinSettings,
) []int {
	if seed < 0 || !available[seed] {
		return nil
	}

	center := vertices[seed]
	currentRadius := settings.CapRadiusIncrement
	var validExpansion []int

	for iter := 0; iter < settings.MaxExpansionIters; iter++ {
		// Find all available vertices within current radius
		var inExpansion []int
		landCount := 0

		for idx := range available {
			dist := AngularDistance(vertices[idx], center)
			if dist <= currentRadius {
				inExpansion = append(inExpansion, idx)
				if elevation[idx] >= threshold {
					landCount++
				}
			}
		}

		if len(inExpansion) == 0 {
			break
		}

		// Stop if we hit land
		if landCount > 0 {
			break
		}

		// Check for boundary-hitting perimeter vertices
		boundaryHitters := findBoundaryHitters(
			inExpansion, available, vertices, elevation, threshold,
			zone, assigned, adj, numVertices, settings,
		)

		if len(boundaryHitters) > 0 {
			// Try to shift center away from boundary
			avgBoundary := calculateCentroid(boundaryHitters, vertices)

			// Vector from boundary centroid to current center
			diffVec := Sub(center, avgBoundary)
			diffLen := Length(diffVec)

			if diffLen < 1e-7 {
				break
			}

			// Project to tangent plane and shift
			east, north := GetTangentVectors(center)
			awayDir := Add(Scale(east, Dot(diffVec, east)), Scale(north, Dot(diffVec, north)))
			awayLen := Length(awayDir)

			if awayLen < 1e-9 {
				break
			}

			awayDir = Scale(awayDir, 1.0/awayLen)
			angleStep := 0.25 * currentRadius

			// Calculate new center
			newCenter := Add(
				Scale(center, Clamp(1.0-angleStep*angleStep/2, 0, 1)), // approximate cos
				Scale(awayDir, angleStep),
			)
			newCenter = Normalize(newCenter)

			shiftDist := AngularDistance(newCenter, center)
			if shiftDist < settings.CenterShiftTolerance {
				break
			}

			// Check if new center would leave the zone
			newZone := GetLatitudeZone(newCenter, settings.PolarLimitDeg)
			if newZone != zone {
				break
			}

			center = newCenter
			continue
		}

		// No boundary hit - this is a valid expansion
		validExpansion = inExpansion
		currentRadius += settings.CapRadiusIncrement
	}

	return validExpansion
}

// findBoundaryHitters finds perimeter vertices adjacent to zone/land boundaries.
func findBoundaryHitters(
	inExpansion []int,
	available map[int]bool,
	vertices []Vector3D,
	elevation []float64,
	threshold float64,
	zone LatitudeZone,
	assigned map[int]bool,
	adj *FlatAdjacency,
	numVertices int,
	settings BasinSettings,
) []int {
	// Build quick lookup
	inExp := make(map[int]bool, len(inExpansion))
	for _, idx := range inExpansion {
		inExp[idx] = true
	}

	var boundaryHitters []int

	for _, u := range inExpansion {
		isPerimeter := false

		for _, v := range adj.GetNeighbors(u) {
			if !inExp[v] {
				isPerimeter = true

				// Check if neighbor is a boundary (land, different zone, or assigned)
				if v >= 0 && v < numVertices {
					if elevation[v] >= threshold ||
						GetLatitudeZone(vertices[v], settings.PolarLimitDeg) != zone ||
						assigned[v] {
						boundaryHitters = append(boundaryHitters, u)
						break
					}
				}
			}
		}
		_ = isPerimeter // Used in logic above
	}

	return boundaryHitters
}

// calculateCentroid computes the normalized centroid of vertices.
func calculateCentroid(indices []int, vertices []Vector3D) Vector3D {
	if len(indices) == 0 {
		return Vector3D{}
	}

	var sum Vector3D
	for _, idx := range indices {
		sum = Add(sum, vertices[idx])
	}

	n := float64(len(indices))
	avg := Vector3D{X: sum.X / n, Y: sum.Y / n, Z: sum.Z / n}

	return Normalize(avg)
}

// calculateMaxRadius finds the maximum angular distance from centroid to any vertex.
func calculateMaxRadius(indices []int, vertices []Vector3D, centroid Vector3D) float64 {
	maxR := 0.0
	for _, idx := range indices {
		dist := AngularDistance(vertices[idx], centroid)
		if dist > maxR {
			maxR = dist
		}
	}
	return maxR
}

// copySet creates a copy of a map[int]bool set.
func copySet(s map[int]bool) map[int]bool {
	copy := make(map[int]bool, len(s))
	for k, v := range s {
		copy[k] = v
	}
	return copy
}
