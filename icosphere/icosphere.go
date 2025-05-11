package icosphere

import "math"

// newEdgeKey creates a canonical EdgeKey from two vertex indices.
// This function is internal to the icosphere package (lowercase 'n').
// If it were needed outside, it would be NewEdgeKey.
func newEdgeKey(i1, i2 int) EdgeKey {
	if i1 < i2 {
		return EdgeKey{i1, i2}
	}
	return EdgeKey{i2, i1}
}

// getMidpoint calculates (or retrieves from cache) the normalized midpoint of an edge.
// This function is internal to the icosphere package.
func getMidpoint(p1Index, p2Index int, vertices *[]Vector3D, cache map[EdgeKey]int) int {
	key := newEdgeKey(p1Index, p2Index)
	if index, found := cache[key]; found {
		return index
	}
	v1 := (*vertices)[p1Index]
	v2 := (*vertices)[p2Index]
	midpoint := v1.Add(v2).Scale(0.5)
	normalizedMidpoint := midpoint.Normalize()
	*vertices = append(*vertices, normalizedMidpoint)
	newIndex := len(*vertices) - 1
	cache[key] = newIndex
	return newIndex
}

// createInitialIcosahedron creates the 12 vertices and 20 faces of a base icosahedron.
// This function is internal to the icosphere package.
func createInitialIcosahedron() ([]Vector3D, []Triangle) {
	phi := (1.0 + math.Sqrt(5.0)) / 2.0
	initialVerticesRaw := []Vector3D{
		{-1, phi, 0}, {1, phi, 0}, {-1, -phi, 0}, {1, -phi, 0},
		{0, -1, phi}, {0, 1, phi}, {0, -1, -phi}, {0, 1, -phi},
		{phi, 0, -1}, {phi, 0, 1}, {-phi, 0, -1}, {-phi, 0, 1},
	}
	vertices := make([]Vector3D, len(initialVerticesRaw))
	for i, v := range initialVerticesRaw {
		vertices[i] = v.Normalize()
	}
	faces := []Triangle{
		{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10}, {0, 10, 11},
		{1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6}, {7, 1, 8},
		{3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8}, {3, 8, 9},
		{4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7}, {9, 8, 1},
	}
	return vertices, faces
}

// CreateIcosphere generates an icosphere mesh by subdividing an icosahedron.
// This function is exported and can be called from other packages.
func CreateIcosphere(subdivisions int) ([]Vector3D, []Triangle) {
	vertices, faces := createInitialIcosahedron()
	midpointCache := make(map[EdgeKey]int)

	for i := 0; i < subdivisions; i++ {
		newFaces := make([]Triangle, 0, len(faces)*4)
		for _, face := range faces {
			v1 := face.V1
			v2 := face.V2
			v3 := face.V3

			m12 := getMidpoint(v1, v2, &vertices, midpointCache)
			m23 := getMidpoint(v2, v3, &vertices, midpointCache)
			m31 := getMidpoint(v3, v1, &vertices, midpointCache)

			newFaces = append(newFaces, Triangle{v1, m12, m31})
			newFaces = append(newFaces, Triangle{v2, m23, m12})
			newFaces = append(newFaces, Triangle{v3, m31, m23})
			newFaces = append(newFaces, Triangle{m12, m23, m31})
		}
		faces = newFaces
	}
	return vertices, faces
}
