package landgen

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"worldgen/icosphere"

	"github.com/kyroy/kdtree" // KD-Tree library
)

// Vector3D uses the definition from the icosphere package.
type Vector3D = icosphere.Vector3D

// Triangle uses the definition from the icosphere package.
type Triangle = icosphere.Triangle

// TectonicPlate stores information about a single tectonic plate.
type TectonicPlate struct {
	ID            int32
	Center        Vector3D // Implements kdtree.Point via icosphere.Vector3D
	MotionVector  Vector3D // Placeholder, primary motion is via RotationAxis and RotationSpeed
	RotationAxis  Vector3D // Axis of rotation for this plate (normalized)
	RotationSpeed float64  // Angular speed in radians per unit of time
}

// GetVelocityAtPoint calculates the linear velocity of a point on the sphere
// due to this plate's rotation.
// point: A Vector3D on the unit sphere.
func (plate TectonicPlate) GetVelocityAtPoint(point Vector3D) Vector3D {
	// Angular velocity vector ω = RotationAxis * RotationSpeed
	angularVelocityVector := plate.RotationAxis.Scale(plate.RotationSpeed)
	// Linear velocity v = ω × r (cross product of angular velocity and position vector of the point)
	return angularVelocityVector.Cross(point)
}

// PlateInteractionType defines the nature of interaction at a plate boundary.
type PlateInteractionType string

const (
	Convergent PlateInteractionType = "convergent"
	Divergent  PlateInteractionType = "divergent"
	Passive    PlateInteractionType = "passive" // Or Transform/Neutral
)

// PlateBoundaryInfo stores information about the interaction between two adjacent plates.
type PlateBoundaryInfo struct {
	Plate1ID        int32
	Plate2ID        int32
	InteractionType PlateInteractionType
}

// TectonicSettings holds parameters for tectonic plate simulation.
type TectonicSettings struct {
	NumPlates   int     `json:"numPlates"`
	Seed        int64   `json:"seed"`
	BaseSpeed   float64 `json:"baseSpeed"`
	SpeedFactor float64 `json:"speedFactor"`
	PConvergent float64 `json:"pConvergent"` // Kept for fallback or mixed models, but primary logic will be velocity-based
	PDivergent  float64 `json:"pDivergent"`  // Kept for fallback or mixed models
	// NumWorkers is now determined automatically based on CPU cores.
}

// TectonicsData holds all generated tectonic data.
type TectonicsData struct {
	Plates                    []TectonicPlate
	VertexPlateIDs            []int32
	IsBoundaryVertex          []bool
	VertexBoundaryTypes       map[int32]PlateInteractionType
	VertexDistancesToBoundary []float64
	NearestBoundaryIndices    []int32
	AdjacentPlateInteractions map[frozensetVal]PlateInteractionType
}

// frozensetVal is a helper type for map keys representing unordered pairs of plate IDs.
type frozensetVal [2]int32

// newFrozenset creates a canonical frozensetVal.
func newFrozenset(id1, id2 int32) frozensetVal {
	if id1 > id2 {
		id1, id2 = id2, id1
	}
	return [2]int32{id1, id2}
}

// getNumWorkers determines the number of goroutines to use.
func getNumWorkers(itemCount int) int {
	numWorkers := runtime.NumCPU()
	if numWorkers > itemCount && itemCount > 0 {
		numWorkers = itemCount
	}
	if numWorkers <= 0 {
		numWorkers = 1
	}
	return numWorkers
}

// CreateTectonicPlates generates initial plate data.
func CreateTectonicPlates(settings TectonicSettings) []TectonicPlate {
	fmt.Printf("Generating %d tectonic plates with seed %d...\n", settings.NumPlates, settings.Seed)
	source := rand.NewSource(settings.Seed)
	rng := rand.New(source)
	plates := make([]TectonicPlate, settings.NumPlates)

	for i := 0; i < settings.NumPlates; i++ {
		center := Vector3D{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: rng.NormFloat64()}.Normalize()
		rotationAxis := Vector3D{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: rng.NormFloat64()}.Normalize()
		speed := (rng.Float64()*settings.BaseSpeed + (settings.BaseSpeed * 0.5))
		rotationSpeed := speed * settings.SpeedFactor
		motionVector := Vector3D{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: rng.NormFloat64()}.Normalize()
		plates[i] = TectonicPlate{
			ID: int32(i), Center: center, MotionVector: motionVector,
			RotationAxis: rotationAxis, RotationSpeed: rotationSpeed,
		}
	}

	if len(plates) > 0 {
		minSpeed, maxSpeed := plates[0].RotationSpeed, plates[0].RotationSpeed
		for _, p := range plates {
			if p.RotationSpeed < minSpeed {
				minSpeed = p.RotationSpeed
			}
			if p.RotationSpeed > maxSpeed {
				maxSpeed = p.RotationSpeed
			}
		}
		fmt.Printf("  Generated %d plates. Scaled plate rotation speeds range: %.4f to %.4f rad/unit_time\n",
			len(plates), minSpeed, maxSpeed)
	}
	return plates
}

// kdtreePlatePoint wraps TectonicPlate for kdtree.
type kdtreePlatePoint struct {
	Plate *TectonicPlate
}

func (p kdtreePlatePoint) Dimensions() int         { return p.Plate.Center.Dimensions() }
func (p kdtreePlatePoint) Dimension(i int) float64 { return p.Plate.Center.Dimension(i) }

// AssignVerticesToPlates assigns vertices to plates using KD-Tree and goroutines.
func AssignVerticesToPlates(vertices []Vector3D, plates []TectonicPlate) []int32 {
	fmt.Println("Assigning mesh vertices to nearest plate centers (KD-Tree, parallel)...")
	startTime := time.Now()
	numVertices := len(vertices)
	vertexPlateIDs := make([]int32, numVertices)

	if len(plates) == 0 {
		fmt.Println("  Warning: No tectonic plates defined. Cannot assign vertices.")
		for i := range vertexPlateIDs {
			vertexPlateIDs[i] = -1
		}
		return vertexPlateIDs
	}

	kdPoints := make([]kdtree.Point, len(plates))
	for i, plate := range plates {
		kdPoints[i] = kdtreePlatePoint{Plate: &plate}
	}

	fmt.Println("  Building KD-Tree for plate centers...")
	kdTree := kdtree.New(kdPoints)
	fmt.Println("  KD-Tree built.")

	numWorkers := getNumWorkers(numVertices)
	var wg sync.WaitGroup
	chunkSize := (numVertices + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			start := workerID * chunkSize
			end := (workerID + 1) * chunkSize
			if end > numVertices {
				end = numVertices
			}
			for i := start; i < end; i++ {
				vertex := vertices[i]
				nearestNeighbors := kdTree.KNN(vertex, 1)
				if len(nearestNeighbors) > 0 {
					if nearestPlatePoint, ok := nearestNeighbors[0].(kdtreePlatePoint); ok {
						vertexPlateIDs[i] = nearestPlatePoint.Plate.ID
					} else {
						fmt.Printf("Error: KD-Tree (plates) returned unexpected type for vertex %d\n", i)
						vertexPlateIDs[i] = -1
					}
				} else {
					fmt.Printf("Error: KD-Tree (plates) returned no neighbors for vertex %d\n", i)
					vertexPlateIDs[i] = -1
				}
			}
		}(w)
	}
	wg.Wait()

	fmt.Printf("  Vertex to plate assignment took %v using %d workers and KD-Tree.\n", time.Since(startTime), numWorkers)
	return vertexPlateIDs
}

// buildAdjacencyList creates a vertex adjacency list.
func buildAdjacencyList(faces []Triangle, numVertices int) map[int32][]int32 {
	// (Implementation remains the same as previous version)
	fmt.Println("Building vertex adjacency list...")
	adjTemp := make(map[int32]map[int32]bool)
	for i := 0; i < numVertices; i++ {
		adjTemp[int32(i)] = make(map[int32]bool)
	}
	for _, face := range faces {
		v := [3]int32{int32(face.V1), int32(face.V2), int32(face.V3)}
		adjTemp[v[0]][v[1]] = true
		adjTemp[v[0]][v[2]] = true
		adjTemp[v[1]][v[0]] = true
		adjTemp[v[1]][v[2]] = true
		adjTemp[v[2]][v[0]] = true
		adjTemp[v[2]][v[1]] = true
	}
	adjList := make(map[int32][]int32)
	for v, neighborsMap := range adjTemp {
		neighbors := make([]int32, 0, len(neighborsMap))
		for neighbor := range neighborsMap {
			neighbors = append(neighbors, neighbor)
		}
		adjList[v] = neighbors
	}
	fmt.Printf("  Adjacency list built for %d vertices.\n", numVertices)
	return adjList
}

// kdtreeBoundaryPoint wraps a Vector3D and its original index for KD-Tree.
type kdtreeBoundaryPoint struct {
	Coordinates   Vector3D
	OriginalIndex int32
}

func (p kdtreeBoundaryPoint) Dimensions() int         { return p.Coordinates.Dimensions() }
func (p kdtreeBoundaryPoint) Dimension(i int) float64 { return p.Coordinates.Dimension(i) }

// FindPlateBoundariesAndTypes identifies boundaries, calculates distances,
// and assigns interaction types based on relative plate velocities.
func FindPlateBoundariesAndTypes(
	vertices []Vector3D, // These are the icosphere sites if analyzing site boundaries
	faces []Triangle, // These are the Delaunay triangles connecting the sites
	vertexPlateIDs []int32, // Plate ID for each icosphere site
	plates []TectonicPlate,
	settings TectonicSettings,
) (
	isBoundaryVertexResult []bool,
	vertexBoundaryTypesResult map[int32]PlateInteractionType,
	vertexDistancesToBoundaryResult []float64,
	nearestBoundaryIndicesResult []int32,
	adjacentPlateInteractionsResult map[frozensetVal]PlateInteractionType,
) {
	fmt.Println("Finding plate boundaries, distances, and types (velocity-based)...")
	numVertices := len(vertexPlateIDs) // This is the number of sites
	isBoundaryVertexResult = make([]bool, numVertices)
	adjacentPlatePairsSet := make(map[frozensetVal]bool)

	// Step 1: Identify boundary SITES and unique adjacent plate pairs
	// A site is a boundary site if any Delaunay edge connected to it spans two different plates.
	for _, face := range faces {
		pIDs := [3]int32{vertexPlateIDs[face.V1], vertexPlateIDs[face.V2], vertexPlateIDs[face.V3]}
		vIndices := [3]int32{int32(face.V1), int32(face.V2), int32(face.V3)}
		for i := 0; i < 3; i++ {
			s1Idx, s2Idx := vIndices[i], vIndices[(i+1)%3] // Indices of sites forming an edge
			p1ID, p2ID := pIDs[i], pIDs[(i+1)%3]
			if p1ID != p2ID {
				isBoundaryVertexResult[s1Idx] = true // Mark site as boundary
				isBoundaryVertexResult[s2Idx] = true // Mark site as boundary
				adjacentPlatePairsSet[newFrozenset(p1ID, p2ID)] = true
			}
		}
	}
	boundarySiteCount := 0
	for _, isBoundary := range isBoundaryVertexResult {
		if isBoundary {
			boundarySiteCount++
		}
	}
	fmt.Printf("  Identified %d boundary sites.\n", boundarySiteCount)
	fmt.Printf("  Found %d unique adjacent plate pairs.\n", len(adjacentPlatePairsSet))

	// Step 2: Calculate distance from each SITE to the nearest boundary SITE (remains the same logic)
	fmt.Println("  Calculating distances for sites to nearest boundary site (KD-Tree, parallel)...")
	startDistTime := time.Now()
	vertexDistancesToBoundaryResult = make([]float64, numVertices)
	nearestBoundaryIndicesResult = make([]int32, numVertices)
	if boundarySiteCount > 0 {
		kdBoundaryPoints := make([]kdtree.Point, 0, boundarySiteCount)
		for i, isBoundary := range isBoundaryVertexResult {
			if isBoundary {
				kdBoundaryPoints = append(kdBoundaryPoints, kdtreeBoundaryPoint{Coordinates: vertices[i], OriginalIndex: int32(i)})
			}
		}
		boundaryKdTree := kdtree.New(kdBoundaryPoints)
		numWorkers := getNumWorkers(numVertices)
		var wg sync.WaitGroup
		chunkSize := (numVertices + numWorkers - 1) / numWorkers
		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				start := workerID * chunkSize
				end := (workerID + 1) * chunkSize
				if end > numVertices {
					end = numVertices
				}
				for i := start; i < end; i++ {
					if isBoundaryVertexResult[i] {
						vertexDistancesToBoundaryResult[i] = 0.0
						nearestBoundaryIndicesResult[i] = int32(i)
						continue
					}
					vertex := vertices[i]
					nearestNeighbors := boundaryKdTree.KNN(vertex, 1)
					if len(nearestNeighbors) > 0 {
						if nearestBoundaryPt, ok := nearestNeighbors[0].(kdtreeBoundaryPoint); ok {
							distSq := vertex.Subtract(nearestBoundaryPt.Coordinates).LengthSq()
							vertexDistancesToBoundaryResult[i] = math.Sqrt(distSq)
							nearestBoundaryIndicesResult[i] = nearestBoundaryPt.OriginalIndex
						} else {
							vertexDistancesToBoundaryResult[i] = math.Inf(1)
							nearestBoundaryIndicesResult[i] = -1
						}
					} else {
						vertexDistancesToBoundaryResult[i] = math.Inf(1)
						nearestBoundaryIndicesResult[i] = -1
					}
				}
			}(w)
		}
		wg.Wait()
		fmt.Printf("    Distances to boundary site calculated in %v.\n", time.Since(startDistTime))
	} else {
		fmt.Println("    Warning: No boundary sites found. Skipping distance calculation.")
		for i := 0; i < numVertices; i++ {
			vertexDistancesToBoundaryResult[i] = math.Inf(1)
			nearestBoundaryIndicesResult[i] = -1
		}
	}

	// --- Step 3: Assign interaction types based on relative velocities ---
	fmt.Println("  Assigning boundary interaction types based on relative velocities...")
	adjacentPlateInteractionsResult = make(map[frozensetVal]PlateInteractionType)
	// Threshold for dot product to classify as convergent/divergent vs. passive
	// This value might need tuning based on typical plate speeds.
	interactionThreshold := 1e-5 // A small non-zero value

	for pair := range adjacentPlatePairsSet {
		plate1ID := pair[0]
		plate2ID := pair[1]

		// Ensure plates exist (should always be true if pair was formed correctly)
		if int(plate1ID) >= len(plates) || int(plate2ID) >= len(plates) {
			fmt.Printf("Warning: Invalid plate ID in pair (%d, %d). Max plate ID: %d\n", plate1ID, plate2ID, len(plates)-1)
			adjacentPlateInteractionsResult[pair] = Passive // Default
			continue
		}

		plate1 := plates[plate1ID]
		plate2 := plates[plate2ID]

		var totalNormalComponent float64
		var countBoundarySegments int

		// Iterate over Delaunay edges that form the boundary between these two plates
		for _, face := range faces {
			sIndices := [3]int32{int32(face.V1), int32(face.V2), int32(face.V3)}
			pIDs := [3]int32{vertexPlateIDs[sIndices[0]], vertexPlateIDs[sIndices[1]], vertexPlateIDs[sIndices[2]]}

			for i := 0; i < 3; i++ {
				site1Idx := sIndices[i]
				site2Idx := sIndices[(i+1)%3]
				site1PlateID := pIDs[i]
				site2PlateID := pIDs[(i+1)%3]

				// Check if this edge is on the boundary of the current plate pair
				if (site1PlateID == plate1ID && site2PlateID == plate2ID) || (site1PlateID == plate2ID && site2PlateID == plate1ID) {
					site1Vec := vertices[site1Idx]
					site2Vec := vertices[site2Idx]

					// Midpoint of the Delaunay edge (approximates a point on the Voronoi boundary)
					boundaryPoint := site1Vec.Add(site2Vec).Scale(0.5).Normalize()

					// Velocities of this boundary point if it were on plate1 or plate2
					vel1 := plate1.GetVelocityAtPoint(boundaryPoint)
					vel2 := plate2.GetVelocityAtPoint(boundaryPoint)

					// Relative velocity of plate2 with respect to plate1 at the boundary point
					relativeVel := vel2.Subtract(vel1)
					if site1PlateID == plate2ID { // Ensure relativeVel is plate2 relative to plate1
						relativeVel = vel1.Subtract(vel2)
					}

					// Normal to the boundary segment at the boundaryPoint, pointing from plate1's region towards plate2's region.
					// The vector connecting the two sites is perpendicular to the Voronoi edge.
					// We project this vector onto the tangent plane at boundaryPoint to get a proper normal in that plane.
					interSiteVector := site2Vec.Subtract(site1Vec)
					if site1PlateID == plate2ID { // Ensure interSiteVector points from plate1's domain to plate2's
						interSiteVector = site1Vec.Subtract(site2Vec)
					}

					// Project interSiteVector onto the tangent plane at boundaryPoint
					// Normal of the sphere at boundaryPoint is boundaryPoint itself (since it's normalized)
					tangentPlaneNormal := boundaryPoint
					boundaryNormal := interSiteVector.Subtract(tangentPlaneNormal.Scale(interSiteVector.Dot(tangentPlaneNormal))).Normalize()

					if boundaryNormal.LengthSq() < 1e-9 { // Avoid issues if sites are diametrically opposite or too close making normal ill-defined
						// Fallback or skip this segment
						// For simplicity, we can use the direct vector between sites if projection is zero,
						// though this isn't strictly in the tangent plane.
						boundaryNormal = interSiteVector.Normalize()
						if boundaryNormal.LengthSq() < 1e-9 {
							continue
						} // Still bad, skip
					}

					// Component of relative velocity along this boundary normal
					// Positive: plates moving apart (divergent) along this normal
					// Negative: plates moving together (convergent) along this normal
					normalComponent := relativeVel.Dot(boundaryNormal)
					totalNormalComponent += normalComponent
					countBoundarySegments++
				}
			}
		}

		if countBoundarySegments > 0 {
			avgNormalComponent := totalNormalComponent / float64(countBoundarySegments)
			if avgNormalComponent > interactionThreshold {
				adjacentPlateInteractionsResult[pair] = Divergent
			} else if avgNormalComponent < -interactionThreshold {
				adjacentPlateInteractionsResult[pair] = Convergent
			} else {
				// Further check for tangential motion for Transform/Passive
				// For now, default to Passive if not clearly convergent/divergent
				adjacentPlateInteractionsResult[pair] = Passive
			}
		} else {
			// No shared boundary segments found (e.g., single plate or error)
			adjacentPlateInteractionsResult[pair] = Passive
		}
	}

	// --- Step 4: Map each boundary SITE to a representative boundary interaction type (sequential) ---
	// This uses the interaction types determined for the plate pairs.
	fmt.Println("  Mapping boundary sites to interaction types...")
	vertexBoundaryTypesResult = make(map[int32]PlateInteractionType)
	adjList := buildAdjacencyList(faces, numVertices) // Adjacency of SITES

	for i := 0; i < numVertices; i++ { // Iterate over SITES
		if isBoundaryVertexResult[i] { // If this site is a boundary site
			myPlateID := vertexPlateIDs[i]
			var assignedType PlateInteractionType = Passive // Default

			neighborSitePlateIDs := make(map[int32]bool) // Plate IDs of neighboring sites that are on different plates
			if neighbors, ok := adjList[int32(i)]; ok {
				for _, neighborSiteIdx := range neighbors {
					if vertexPlateIDs[neighborSiteIdx] != myPlateID {
						neighborSitePlateIDs[vertexPlateIDs[neighborSiteIdx]] = true
					}
				}
			}

			// Prioritize Convergent/Divergent if any adjacent pair has it
			priorityOrder := []PlateInteractionType{Convergent, Divergent, Passive}
			foundType := false
			for _, pType := range priorityOrder {
				for neighborPlateID := range neighborSitePlateIDs {
					pair := newFrozenset(myPlateID, neighborPlateID)
					if typeVal, ok := adjacentPlateInteractionsResult[pair]; ok && typeVal == pType {
						assignedType = typeVal
						foundType = true
						break
					}
				}
				if foundType {
					break
				}
			}
			vertexBoundaryTypesResult[int32(i)] = assignedType
		}
	}

	fmt.Println("Boundary finding and type assignment process complete.")
	return
}
