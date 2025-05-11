package icosphere

import (
	"math"
	"runtime"
	"sort"
	"sync"
	// "fmt" // Uncomment for debugging
)

// VoronoiCell defines the structure for a single Voronoi cell.
// This type is exported.
type VoronoiCell struct {
	SiteIndex           int32   // Index of the original icosphere site this cell corresponds to
	NeighborSiteIndices []int32 // Indices of neighboring icosphere sites (0-based, from original sites), sorted angularly (clockwise).
	VertexIndices       []int32 // Indices into the global list of Voronoi vertices (0-based, from voronoiVertices), sorted angularly.
}

// CalculateSphericalCircumcenter computes the circumcenter of a spherical triangle (A,B,P).
// This function is exported.
func CalculateSphericalCircumcenter(vA, vB, vP Vector3D) Vector3D {
	// Calculate the sum of cross products: (A x B) + (B x P) + (P x A)
	// The direction of this vector is orthogonal to the plane of the triangle,
	// and its magnitude is related to the triangle's area.
	// Normalizing this vector gives the circumcenter on the sphere, provided
	// the sites are on the unit sphere.
	xVec := vA.Cross(vB).Add(vB.Cross(vP)).Add(vP.Cross(vA))

	// Ensure the circumcenter is on the same hemisphere as the triangle's centroid.
	// This handles potential orientation issues with the cross product sum.
	triangleCentroid := vA.Add(vB).Add(vP).Scale(1.0 / 3.0)
	if xVec.Dot(triangleCentroid) < 0 {
		xVec = xVec.Scale(-1) // Flip direction if it's pointing away
	}
	return xVec.Normalize()
}

// GenerateSphericalVoronoi creates a spherical Voronoi diagram.
// sites: The input points on the sphere (e.g., relaxed icosphere vertices). These must be normalized.
// delaunayFaces: The triangular mesh assumed to be the Delaunay triangulation of the sites.
// Returns: voronoiVertices (the circumcenters of Delaunay triangles) and a slice of VoronoiCell.
func GenerateSphericalVoronoi(sites []Vector3D, delaunayFaces []Triangle) ([]Vector3D, []VoronoiCell) {
	numSites := len(sites)
	numDelaunayTriangles := len(delaunayFaces)

	if numSites == 0 || numDelaunayTriangles == 0 {
		return []Vector3D{}, []VoronoiCell{}
	}

	numWorkers := runtime.NumCPU()
	var wg sync.WaitGroup

	// Step 1: Calculate all Voronoi vertices (circumcenters of Delaunay triangles) in parallel.
	voronoiVertices := make([]Vector3D, numDelaunayTriangles)
	if numDelaunayTriangles > 0 {
		actualNumWorkers := numWorkers
		if actualNumWorkers > numDelaunayTriangles {
			actualNumWorkers = numDelaunayTriangles
		}
		if actualNumWorkers <= 0 { // Should not happen if numDelaunayTriangles > 0
			actualNumWorkers = 1
		}

		for w := 0; w < actualNumWorkers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				startIndex := workerID * (numDelaunayTriangles / actualNumWorkers)
				endIndex := (workerID + 1) * (numDelaunayTriangles / actualNumWorkers)
				if workerID == actualNumWorkers-1 {
					endIndex = numDelaunayTriangles
				}
				for i := startIndex; i < endIndex; i++ {
					tri := delaunayFaces[i]
					voronoiVertices[i] = CalculateSphericalCircumcenter(sites[tri.V1], sites[tri.V2], sites[tri.V3])
				}
			}(w)
		}
		wg.Wait()
	}

	// Step 2: Build mappings:
	// - siteToTrianglesMap: For each site, lists Delaunay triangles incident to it.
	// - siteNeighborsMap: For each site, lists its neighboring sites (from Delaunay edges).
	siteToTrianglesMap := make([][]int, numSites)        // Stores indices of voronoiVertices (which correspond to Delaunay triangles)
	siteNeighborsMap := make([]map[int32]bool, numSites) // Stores indices of neighbor sites
	for i := 0; i < numSites; i++ {
		siteNeighborsMap[i] = make(map[int32]bool)
	}

	for triIdx, tri := range delaunayFaces {
		s1, s2, s3 := int32(tri.V1), int32(tri.V2), int32(tri.V3)

		// Each vertex of a Delaunay triangle is a site.
		// The circumcenter of this triangle (voronoiVertices[triIdx]) is a vertex of the Voronoi cells of s1, s2, s3.
		siteToTrianglesMap[s1] = append(siteToTrianglesMap[s1], triIdx)
		siteToTrianglesMap[s2] = append(siteToTrianglesMap[s2], triIdx)
		siteToTrianglesMap[s3] = append(siteToTrianglesMap[s3], triIdx)

		// Sites sharing an edge in the Delaunay triangulation are Voronoi neighbors.
		siteNeighborsMap[s1][s2] = true
		siteNeighborsMap[s1][s3] = true
		siteNeighborsMap[s2][s1] = true
		siteNeighborsMap[s2][s3] = true
		siteNeighborsMap[s3][s1] = true
		siteNeighborsMap[s3][s2] = true
	}

	// Step 3: Construct Voronoi cells in parallel.
	voronoiCells := make([]VoronoiCell, numSites)
	if numSites > 0 {
		actualNumWorkers := numWorkers
		if actualNumWorkers > numSites {
			actualNumWorkers = numSites
		}
		if actualNumWorkers <= 0 { // Should not happen if numSites > 0
			actualNumWorkers = 1
		}

		for w := 0; w < actualNumWorkers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				startIndex := workerID * (numSites / actualNumWorkers)
				endIndex := (workerID + 1) * (numSites / actualNumWorkers)
				if workerID == actualNumWorkers-1 {
					endIndex = numSites
				}

				for siteIdx := startIndex; siteIdx < endIndex; siteIdx++ {
					currentSiteIndex32 := int32(siteIdx)
					voronoiCells[siteIdx].SiteIndex = currentSiteIndex32
					siteVec := sites[siteIdx] // Current site's vector

					// Define a local 2D tangent plane at siteVec for angular sorting.
					// upApprox helps define a consistent 'up' direction to establish the plane.
					upApprox := Vector3D{0, 0, 1} // Initial guess for 'up'
					// If siteVec is too close to upApprox (e.g., at a pole), choose a different 'up'.
					if math.Abs(siteVec.Dot(upApprox)) > 0.99 {
						upApprox = Vector3D{0, 1, 0} // Use Y-axis as 'up'
					}
					tangentX := siteVec.Cross(upApprox).Normalize() // First axis in the tangent plane
					tangentY := tangentX.Cross(siteVec).Normalize() // Second axis, orthogonal to tangentX and siteVec (completes LH basis if siteVec is 'normal')
					// Or, tangentY = siteVec.Cross(tangentX).Normalize() for RH basis with siteVec as Z.
					// The original code used tangentX.Cross(siteVec), let's maintain that.
					// This means (tangentX, tangentY, siteVec) might be LH or RH depending on Cross order.
					// For Atan2, the relative orientation of X and Y matters.

					// Populate and sort NeighborSiteIndices angularly (clockwise).
					neighborSiteIndices := make([]int32, 0, len(siteNeighborsMap[siteIdx]))
					for neighborIdx := range siteNeighborsMap[siteIdx] {
						neighborSiteIndices = append(neighborSiteIndices, neighborIdx)
					}

					sort.SliceStable(neighborSiteIndices, func(i, j int) bool {
						neighbor1SiteVec := sites[neighborSiteIndices[i]]
						neighbor2SiteVec := sites[neighborSiteIndices[j]]

						// Project neighbor vectors onto the tangent plane of siteVec.
						// The dot product of a vector with tangentX/tangentY gives its coordinate
						// along that axis in the plane, effectively projecting it.
						n1LocalX := neighbor1SiteVec.Dot(tangentX)
						n1LocalY := neighbor1SiteVec.Dot(tangentY)
						angle1 := math.Atan2(n1LocalY, n1LocalX) // Angle for neighbor 1

						n2LocalX := neighbor2SiteVec.Dot(tangentX)
						n2LocalY := neighbor2SiteVec.Dot(tangentY)
						angle2 := math.Atan2(n2LocalY, n2LocalX) // Angle for neighbor 2

						return angle1 > angle2 // Sort by decreasing angle for clockwise order
					})
					voronoiCells[siteIdx].NeighborSiteIndices = neighborSiteIndices

					// Populate and sort VertexIndices (vertices of the Voronoi cell polygon) angularly.
					incidentTriangleIndices := siteToTrianglesMap[siteIdx] // These are indices into voronoiVertices
					if len(incidentTriangleIndices) < 3 {
						// Not enough vertices to form a polygon, should ideally not happen for valid spherical Voronoi.
						voronoiCells[siteIdx].VertexIndices = []int32{}
						continue
					}

					// Copy triangle indices to a slice of int32 for VoronoiCell.VertexIndices
					sortedVoronoiVertexIndicesForCell := make([]int32, len(incidentTriangleIndices))
					for i, triIdxVal := range incidentTriangleIndices {
						sortedVoronoiVertexIndicesForCell[i] = int32(triIdxVal)
					}

					// Sort the Voronoi vertices that form this cell's polygon angularly.
					sort.SliceStable(sortedVoronoiVertexIndicesForCell, func(i, j int) bool {
						voronoiVertexIdx1 := sortedVoronoiVertexIndicesForCell[i]
						voronoiVertexIdx2 := sortedVoronoiVertexIndicesForCell[j]

						v1 := voronoiVertices[voronoiVertexIdx1] // 3D coord of Voronoi vertex 1
						v2 := voronoiVertices[voronoiVertexIdx2] // 3D coord of Voronoi vertex 2

						// Project Voronoi vertices onto the tangent plane of siteVec.
						v1LocalX := v1.Dot(tangentX)
						v1LocalY := v1.Dot(tangentY)
						angle1 := math.Atan2(v1LocalY, v1LocalX) // Angle for Voronoi vertex 1

						v2LocalX := v2.Dot(tangentX)
						v2LocalY := v2.Dot(tangentY)
						angle2 := math.Atan2(v2LocalY, v2LocalX) // Angle for Voronoi vertex 2

						// Standard sort is counter-clockwise (increasing angle).
						return angle1 < angle2
					})
					voronoiCells[siteIdx].VertexIndices = sortedVoronoiVertexIndicesForCell
				}
			}(w)
		}
		wg.Wait()
	}
	return voronoiVertices, voronoiCells
}
