package landgen

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"worldgen/icosphere" // Assuming icosphere package provides Vector3D, Triangle

	"github.com/kyroy/kdtree" // KD-Tree library
)

// Vector3D uses the definition from the icosphere package.
type Vector3D = icosphere.Vector3D

// Triangle uses the definition from the icosphere package.
type Triangle = icosphere.Triangle

// VoronoiCell uses the definition from the icosphere package.
// We need access to its fields, so it's assumed to be defined in icosphere package
// or a shared types package. For this example, we'll assume it's:
//
//	type VoronoiCell struct {
//	    SiteIndex           int32
//	    NeighborSiteIndices []int32
//	    VertexIndices       []int32
//	}
//
// Ensure this matches the actual definition in your icosphere package.
type VoronoiCell = icosphere.VoronoiCell

// PlateNature defines the fundamental type of a tectonic plate.
type PlateNature string

const (
	OceanicPlate     PlateNature = "oceanic"
	ContinentalPlate PlateNature = "continental"
	DefaultPlate     PlateNature = "default" // Should be replaced during assignment
)

// TectonicPlate stores information about a single tectonic plate.
type TectonicPlate struct {
	ID                      int32       // Unique ID, typically matches VoronoiCell.SiteIndex
	PlateType               PlateNature // Oceanic or Continental
	InitialVoronoiCellIndex int32       // Index in the original voronoiCells slice
	Center                  Vector3D    // Geometric center, from icosphereSites[voronoiCell.SiteIndex]
	Area                    float64     // Surface area of the plate on the sphere
	RotationAxis            Vector3D    // Axis of rotation for this plate (normalized)
	RotationSpeed           float64     // Angular speed in radians per unit of time (e.g., per million years)
	// MemberSiteIndices []int32 // Populated by AssignSitesToPlates if needed explicitly per plate
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

// TectonicSettings holds parameters for tectonic plate simulation.
type TectonicSettings struct {
	NumPlates                   int     `json:"numPlates"` // This will be derived from len(voronoiCells)
	Seed                        int64   `json:"seed"`
	BaseSpeed                   float64 `json:"baseSpeed"`                   // Base for angular speed (e.g., degrees per million years)
	SpeedVariationFactor        float64 `json:"speedVariationFactor"`        // Factor to vary speed (e.g., 0.5 means speed can be BaseSpeed +/- 50%)
	TargetContinentalProportion float64 `json:"targetContinentalProportion"` // e.g., 0.3 to 0.4 for Earth-like
	NumInitialContinentalSeeds  int     `json:"numInitialContinentalSeeds"`  // e.g., 3-5 or 5-10% of NumPlates
	PlanetRadius                float64 `json:"planetRadius"`                // Radius of the planet, needed for area calculations. Assume 1.0 for unit sphere if not set.
}

// TectonicsData holds all generated tectonic data.
type TectonicsData struct {
	Plates                     []TectonicPlate
	SitePlateIDs               []int32                               // Maps icosphere site index to Plate.ID
	CellToPlateID              map[int32]int32                       // Maps Voronoi cell SiteIndex to Plate.ID
	IsBoundarySite             []bool                                // True if an icosphere site is on a plate boundary
	SiteBoundaryTypes          map[int32]PlateInteractionType        // Interaction type for boundary icosphere sites
	SiteDistancesToBoundary    []float64                             // Distance from icosphere site to nearest boundary site
	NearestBoundarySiteIndices []int32                               // Index of the nearest boundary site for each icosphere site
	AdjacentPlateInteractions  map[frozensetVal]PlateInteractionType // Interaction type between pairs of adjacent plates
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
	if itemCount > 0 && numWorkers > itemCount {
		numWorkers = itemCount
	}
	if numWorkers <= 0 {
		numWorkers = 1
	}
	return numWorkers
}

// calculateSphericalTriangleArea calculates the area of a spherical triangle on a sphere of a given radius.
// Vertices vA, vB, vC are 3D vectors from the sphere's origin, assumed to be normalized if radius is 1.
// Uses L'Huilier's Theorem for spherical excess E: tan(E/4)^2 = tan(s/2) * tan((s-a)/2) * tan((s-b)/2) * tan((s-c)/2)
// where a, b, c are side lengths (angles subtended by sides at sphere center), and s = (a+b+c)/2.
// Area = E * radius^2.
func calculateSphericalTriangleArea(vA, vB, vC Vector3D, radius float64) float64 {
	if radius == 0 {
		return 0
	}
	// Ensure vectors are normalized for angle calculation (representing points on unit sphere)
	// If radius != 1, the input vectors should ideally already be on the unit sphere,
	// or we scale them down, calculate unit sphere area, then scale up.
	// Assuming vA, vB, vC are directions, their length doesn't affect angles between them.
	// Side lengths (angles subtended at the center of the sphere)
	// cos(angle) = (v1 . v2) / (|v1|*|v2|). If v1, v2 are normalized, cos(angle) = v1 . v2
	// We need to ensure inputs are normalized if they aren't already.
	// Let's assume they are normalized for simplicity here.
	a := math.Acos(math.Max(-1.0, math.Min(1.0, vB.Dot(vC)))) // Angle between B and C
	b := math.Acos(math.Max(-1.0, math.Min(1.0, vA.Dot(vC)))) // Angle between A and C
	c := math.Acos(math.Max(-1.0, math.Min(1.0, vA.Dot(vB)))) // Angle between A and B

	s := (a + b + c) / 2.0 // Semi-perimeter

	// Numerator of tan(E/4)^2. Handle potential negative arguments to tan due to floating point issues.
	tanS2 := math.Tan(s / 2.0)
	tanSa2 := math.Tan((s - a) / 2.0)
	tanSb2 := math.Tan((s - b) / 2.0)
	tanSc2 := math.Tan((s - c) / 2.0)

	// Ensure arguments to tan are not problematic (e.g. from s-a < 0 slightly)
	// This can happen if a+b < c (triangle inequality violation for spherical triangles)
	// or if points are collinear/identical.
	if s < a || s < b || s < c { // Check for triangle inequality issues
		// This indicates degenerate or invalid triangle, area is effectively 0
		// A more robust check would be if tan values become negative.
		return 0.0
	}
	// Product for tan(E/4)^2
	prod := tanS2 * tanSa2 * tanSb2 * tanSc2
	if prod < 0 { // Should not happen for valid spherical triangles if s > a,b,c
		return 0.0 // Degenerate triangle
	}

	sphericalExcess := 4.0 * math.Atan(math.Sqrt(math.Max(0, prod))) // Max(0,prod) to avoid NaN from tiny negatives

	if math.IsNaN(sphericalExcess) || math.IsInf(sphericalExcess, 0) {
		// Fallback for very small triangles or numerical instability:
		// Use vector cross product method for small triangles (approx flat area)
		// Area approx 0.5 * |(B-A) x (C-A)| projected onto sphere normal at A (or centroid)
		// For simplicity, return 0 for now on failure.
		// A robust solution might use Girard's theorem directly with angles if sides are problematic.
		return 0.0
	}
	return sphericalExcess * radius * radius
}

// calculateSphericalPolygonArea calculates the area of a spherical polygon.
// polygonVertexCoords: A list of 3D vectors representing the ordered vertices of the polygon on the sphere.
// These vectors should be from the sphere's origin and ideally normalized.
// radius: The radius of the sphere.
// Uses fan triangulation: picks V0, sums areas of triangles (V0, Vi, V(i+1)).
func calculateSphericalPolygonArea(polygonVertexCoords []Vector3D, radius float64) float64 {
	if len(polygonVertexCoords) < 3 {
		return 0.0
	}
	if radius == 0 {
		return 0.0
	}

	totalArea := 0.0
	// v0 := polygonVertexCoords[0] // This line was removed as v0 was unused. v0Normalized is used instead.

	// Normalize all vertices once to ensure they are on the unit sphere for triangle calculation,
	// then scale the final area by radius^2.
	normalizedVertices := make([]Vector3D, len(polygonVertexCoords))
	for i, v := range polygonVertexCoords {
		normalizedVertices[i] = v.Normalize() // Ensure they are on unit sphere
	}

	v0Normalized := normalizedVertices[0]

	for i := 1; i < len(normalizedVertices)-1; i++ {
		v1Normalized := normalizedVertices[i]
		v2Normalized := normalizedVertices[i+1]
		// Ensure no zero vectors from normalization if original was zero (should not happen for sphere points)
		if v0Normalized.LengthSq() == 0 || v1Normalized.LengthSq() == 0 || v2Normalized.LengthSq() == 0 {
			continue
		}
		totalArea += calculateSphericalTriangleArea(v0Normalized, v1Normalized, v2Normalized, 1.0) // Calculate on unit sphere
	}
	return totalArea * radius * radius // Scale to actual radius
}

// InitializeTectonicPlates creates the initial set of tectonic plates based on Voronoi cells.
// It assigns types (oceanic/continental) using a seeding and growth algorithm.
func InitializeTectonicPlates(
	voronoiCells []VoronoiCell,
	voronoiVertices []Vector3D, // Global list of all Voronoi diagram vertices
	icosphereSites []Vector3D, // Sites that generated the Voronoi cells (potential plate centers)
	settings TectonicSettings,
) []TectonicPlate {
	numPlates := len(voronoiCells)
	if numPlates == 0 {
		fmt.Println("Warning: No Voronoi cells provided to InitializeTectonicPlates.")
		return []TectonicPlate{}
	}
	fmt.Printf("Initializing %d tectonic plates with seed %d...\n", numPlates, settings.Seed)
	source := rand.NewSource(settings.Seed)
	rng := rand.New(source)

	plates := make([]TectonicPlate, numPlates)
	totalSphereSurfaceArea := 0.0 // Will be sum of all plate areas

	// Temporary map to quickly find a plate by its ID (which is VoronoiCell.SiteIndex)
	plateMap := make(map[int32]*TectonicPlate)

	for i := 0; i < numPlates; i++ {
		cell := voronoiCells[i]
		plateID := cell.SiteIndex // Assuming SiteIndex is unique and serves as Plate ID

		// Collect vertex coordinates for this cell's polygon
		polygonCoords := make([]Vector3D, len(cell.VertexIndices))
		for j, vIdx := range cell.VertexIndices {
			if int(vIdx) < len(voronoiVertices) {
				polygonCoords[j] = voronoiVertices[vIdx]
			} else {
				fmt.Printf("Warning: Vertex index %d out of bounds for Voronoi cell %d (SiteIndex %d).\n", vIdx, i, cell.SiteIndex)
				// This plate will have an invalid area. Handle downstream or ensure valid inputs.
				polygonCoords = []Vector3D{} // Mark as invalid for area calculation
				break
			}
		}

		plateArea := 0.0
		if len(polygonCoords) >= 3 {
			plateArea = calculateSphericalPolygonArea(polygonCoords, settings.PlanetRadius)
		}
		if plateArea <= 0 && len(polygonCoords) >= 3 {
			// fmt.Printf("Warning: Plate %d (Cell %d) calculated area %f. Polygon vertices: %d\n", plateID, i, plateArea, len(polygonCoords))
			// This might happen for very small/degenerate cells or numerical issues. Assign a tiny positive area.
			plateArea = 1e-9
		}
		totalSphereSurfaceArea += plateArea

		// Rotation Axis (random unit vector)
		rotAxis := Vector3D{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: rng.NormFloat64()}.Normalize()
		// Rotation Speed (degrees per million years, then convert to radians)
		// BaseSpeed could be e.g., 0.5 deg/Myr, SpeedVariationFactor e.g., 0.8 (for +/- 80% variation)
		speedDegMyr := settings.BaseSpeed + settings.BaseSpeed*settings.SpeedVariationFactor*(rng.Float64()*2.0-1.0)
		speedRadMyr := speedDegMyr * (math.Pi / 180.0)

		plates[i] = TectonicPlate{
			ID:                      plateID,
			PlateType:               DefaultPlate, // Will be set below
			InitialVoronoiCellIndex: int32(i),
			Center:                  icosphereSites[cell.SiteIndex], // Center is the generating site of the Voronoi cell
			Area:                    plateArea,
			RotationAxis:            rotAxis,
			RotationSpeed:           speedRadMyr,
		}
		plateMap[plateID] = &plates[i]
	}
	fmt.Printf("  Total sphere surface area from sum of plate areas: %f (expected for unit radius sphere: ~12.56)\n", totalSphereSurfaceArea/(settings.PlanetRadius*settings.PlanetRadius))

	// Assign Plate Types: Continental Seeding and Growth
	fmt.Println("  Assigning plate types (Continental Seeding and Growth)...")
	// Default all to oceanic first
	for i := range plates {
		plates[i].PlateType = OceanicPlate
	}

	numInitialSeeds := settings.NumInitialContinentalSeeds
	if numInitialSeeds <= 0 || numInitialSeeds > numPlates {
		numInitialSeeds = int(math.Max(1.0, float64(numPlates)*0.05)) // Default to 5% or at least 1
		fmt.Printf("  Adjusted NumInitialContinentalSeeds to %d\n", numInitialSeeds)
	}

	targetContinentalArea := totalSphereSurfaceArea * settings.TargetContinentalProportion
	currentContinentalArea := 0.0

	// Create a list of indices to shuffle for random seed selection
	plateIndices := make([]int, numPlates)
	for i := 0; i < numPlates; i++ {
		plateIndices[i] = i
	}
	rng.Shuffle(len(plateIndices), func(i, j int) {
		plateIndices[i], plateIndices[j] = plateIndices[j], plateIndices[i]
	})

	queue := make([]*TectonicPlate, 0, numInitialSeeds)
	continentalSeedCount := 0

	for _, plateIdx := range plateIndices {
		if continentalSeedCount < numInitialSeeds {
			if plates[plateIdx].PlateType == OceanicPlate { // Ensure we don't pick a seed that somehow got converted
				plates[plateIdx].PlateType = ContinentalPlate
				currentContinentalArea += plates[plateIdx].Area
				queue = append(queue, &plates[plateIdx])
				continentalSeedCount++
				// fmt.Printf("    Seed %d: Plate ID %d, Area %f\n", continentalSeedCount, plates[plateIdx].ID, plates[plateIdx].Area)
			}
		} else {
			break
		}
	}

	head := 0
	for head < len(queue) && currentContinentalArea < targetContinentalArea {
		currentPlate := queue[head]
		head++

		// Find original Voronoi cell for currentPlate to get its neighbors
		var currentCell VoronoiCell
		foundCell := false
		for _, cell := range voronoiCells {
			if cell.SiteIndex == currentPlate.ID {
				currentCell = cell
				foundCell = true
				break
			}
		}
		if !foundCell {
			fmt.Printf("Warning: Could not find Voronoi cell for plate ID %d during type assignment.\n", currentPlate.ID)
			continue
		}

		// Shuffle neighbors for more organic growth
		neighborSiteIndices := make([]int32, len(currentCell.NeighborSiteIndices))
		copy(neighborSiteIndices, currentCell.NeighborSiteIndices)
		rng.Shuffle(len(neighborSiteIndices), func(i, j int) {
			neighborSiteIndices[i], neighborSiteIndices[j] = neighborSiteIndices[j], neighborSiteIndices[i]
		})

		for _, neighborSiteIdx := range neighborSiteIndices {
			neighborPlate, ok := plateMap[neighborSiteIdx]
			if !ok {
				// This might happen if a neighbor site index doesn't correspond to any initialized plate.
				// Should not occur if voronoiCells and icosphereSites are consistent.
				// fmt.Printf("Warning: Neighbor plate with SiteIndex %d not found in plateMap.\n", neighborSiteIdx)
				continue
			}

			if neighborPlate.PlateType == OceanicPlate {
				neighborPlate.PlateType = ContinentalPlate
				currentContinentalArea += neighborPlate.Area
				queue = append(queue, neighborPlate)
				// fmt.Printf("      Converted neighbor Plate ID %d to Continental. New total continental area: %f\n", neighborPlate.ID, currentContinentalArea)
				if currentContinentalArea >= targetContinentalArea {
					break // Exit neighbor loop
				}
			}
		}
	}

	// Report final counts
	oceanicCount := 0
	continentalCount := 0
	finalContinentalArea := 0.0
	for _, p := range plates {
		if p.PlateType == ContinentalPlate {
			continentalCount++
			finalContinentalArea += p.Area
		} else {
			oceanicCount++
		}
	}
	fmt.Printf("  Plate type assignment complete: %d Continental plates, %d Oceanic plates.\n", continentalCount, oceanicCount)
	fmt.Printf("  Target continental area: %f, Actual continental area: %f (Proportion: %f)\n",
		targetContinentalArea, finalContinentalArea, finalContinentalArea/totalSphereSurfaceArea)

	// Log plate speed distribution
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
		fmt.Printf("  Generated %d plates. Rotation speeds (rad/Myr): min=%.4f, max=%.4f\n",
			len(plates), minSpeed, maxSpeed)
	}

	return plates
}

// kdtreePlatePoint wraps TectonicPlate for kdtree.
// It uses the TectonicPlate.Center for KD-Tree positioning.
type kdtreePlatePoint struct {
	Plate *TectonicPlate
}

func (p kdtreePlatePoint) Dimensions() int         { return p.Plate.Center.Dimensions() }
func (p kdtreePlatePoint) Dimension(i int) float64 { return p.Plate.Center.Dimension(i) }

// AssignSitesToPlates assigns icosphere sites to the nearest tectonic plate centers using a KD-Tree.
// vertices: These are the icosphere sites.
// plates: The initialized tectonic plates.
// Returns a slice where each element is the ID of the plate the corresponding icosphere site belongs to.
func AssignSitesToPlates(icosphereSites []Vector3D, plates []TectonicPlate) []int32 {
	fmt.Println("Assigning icosphere sites to nearest plate centers (KD-Tree, parallel)...")
	startTime := time.Now()
	numSites := len(icosphereSites)
	sitePlateIDs := make([]int32, numSites)

	if len(plates) == 0 {
		fmt.Println("  Warning: No tectonic plates defined. Cannot assign sites.")
		for i := range sitePlateIDs {
			sitePlateIDs[i] = -1 // Or a special "no plate" ID
		}
		return sitePlateIDs
	}

	// Prepare points for KD-Tree (using plate centers)
	kdPoints := make([]kdtree.Point, len(plates))
	for i := range plates {
		// Important: Pass pointer to the plate in the original slice, not a copy.
		kdPoints[i] = kdtreePlatePoint{Plate: &plates[i]}
	}

	fmt.Println("  Building KD-Tree for plate centers...")
	kdTree := kdtree.New(kdPoints)
	fmt.Println("  KD-Tree built.")

	numWorkers := getNumWorkers(numSites)
	var wg sync.WaitGroup
	chunkSize := (numSites + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			start := workerID * chunkSize
			end := (workerID + 1) * chunkSize
			if end > numSites {
				end = numSites
			}
			for i := start; i < end; i++ {
				siteVec := icosphereSites[i]
				// Find the nearest plate center to this site
				nearestNeighbors := kdTree.KNN(siteVec, 1) // KNN returns a slice of kdtree.Point
				if len(nearestNeighbors) > 0 {
					// Type assert to get back our kdtreePlatePoint
					if nearestPlatePt, ok := nearestNeighbors[0].(kdtreePlatePoint); ok {
						sitePlateIDs[i] = nearestPlatePt.Plate.ID
					} else {
						fmt.Printf("Error: KD-Tree (plates) returned unexpected type for site %d\n", i)
						sitePlateIDs[i] = -1
					}
				} else {
					// This should ideally not happen if there's at least one plate.
					fmt.Printf("Error: KD-Tree (plates) returned no neighbors for site %d\n", i)
					sitePlateIDs[i] = -1
				}
			}
		}(w)
	}
	wg.Wait()

	fmt.Printf("  Icosphere site to plate assignment took %v using %d workers.\n", time.Since(startTime), numWorkers)
	return sitePlateIDs
}

// buildSiteAdjacencyList creates a site adjacency list from Delaunay faces.
// faces: Delaunay triangulation of the icosphere sites.
// numSites: Total number of icosphere sites.
func buildSiteAdjacencyList(faces []Triangle, numSites int) map[int32][]int32 {
	fmt.Println("Building icosphere site adjacency list...")
	adjTemp := make(map[int32]map[int32]bool) // Using map to handle duplicates easily
	for i := 0; i < numSites; i++ {
		adjTemp[int32(i)] = make(map[int32]bool)
	}

	for _, face := range faces {
		s := [3]int32{int32(face.V1), int32(face.V2), int32(face.V3)}
		// Each edge in a face implies adjacency
		adjTemp[s[0]][s[1]] = true
		adjTemp[s[0]][s[2]] = true
		adjTemp[s[1]][s[0]] = true
		adjTemp[s[1]][s[2]] = true
		adjTemp[s[2]][s[0]] = true
		adjTemp[s[2]][s[1]] = true
	}

	adjList := make(map[int32][]int32)
	for siteIdx, neighborsMap := range adjTemp {
		neighbors := make([]int32, 0, len(neighborsMap))
		for neighborIdx := range neighborsMap {
			neighbors = append(neighbors, neighborIdx)
		}
		adjList[siteIdx] = neighbors
	}
	fmt.Printf("  Site adjacency list built for %d sites.\n", numSites)
	return adjList
}

// kdtreeBoundarySitePoint wraps a Vector3D (site coordinate) and its original index for KD-Tree.
type kdtreeBoundarySitePoint struct {
	Coordinates   Vector3D
	OriginalIndex int32
}

func (p kdtreeBoundarySitePoint) Dimensions() int         { return p.Coordinates.Dimensions() }
func (p kdtreeBoundarySitePoint) Dimension(i int) float64 { return p.Coordinates.Dimension(i) }

// FindPlateBoundariesAndTypes identifies boundaries on the icosphere site mesh,
// calculates distances for sites to these boundaries, and determines interaction types
// based on relative plate velocities.
func FindPlateBoundariesAndTypes(
	icosphereSites []Vector3D, // Coordinates of all icosphere sites.
	icosphereFaces []Triangle, // Delaunay triangulation of icosphereSites.
	sitePlateIDs []int32, // Plate ID assigned to each icosphere site.
	plates []TectonicPlate, // List of all tectonic plates with their motion data.
	// settings TectonicSettings,    // Not directly used in this version for type assignment logic, but could be for thresholds.
) (
	isBoundarySiteResult []bool,
	siteBoundaryTypesResult map[int32]PlateInteractionType, // For each boundary site, its dominant interaction type
	siteDistancesToBoundaryResult []float64, // For each site, its distance to the nearest boundary site
	nearestBoundarySiteIndicesResult []int32, // For each site, index of the nearest boundary site
	adjacentPlateInteractionsResult map[frozensetVal]PlateInteractionType, // Interaction type for each pair of adjacent plates
) {
	fmt.Println("Finding plate boundaries, distances, and types (velocity-based)...")
	numSites := len(icosphereSites)
	if numSites == 0 || len(sitePlateIDs) != numSites {
		fmt.Println("Error: Invalid input to FindPlateBoundariesAndTypes.")
		return
	}

	isBoundarySiteResult = make([]bool, numSites)
	siteDistancesToBoundaryResult = make([]float64, numSites)
	nearestBoundarySiteIndicesResult = make([]int32, numSites)
	siteBoundaryTypesResult = make(map[int32]PlateInteractionType)
	adjacentPlateInteractionsResult = make(map[frozensetVal]PlateInteractionType)

	// Step 1: Identify boundary SITES and unique adjacent plate pairs
	// A site is a boundary site if any Delaunay edge connected to it spans two different plates.
	adjacentPlatePairsSet := make(map[frozensetVal]bool)
	for _, face := range icosphereFaces {
		pIDs := [3]int32{sitePlateIDs[face.V1], sitePlateIDs[face.V2], sitePlateIDs[face.V3]}
		vIndices := [3]int32{int32(face.V1), int32(face.V2), int32(face.V3)}

		// Check edges of the triangle: (v0,v1), (v1,v2), (v2,v0)
		for i := 0; i < 3; i++ {
			s1Idx, s2Idx := vIndices[i], vIndices[(i+1)%3] // Indices of sites forming an edge
			p1ID, p2ID := pIDs[i], pIDs[(i+1)%3]

			if p1ID != p2ID && p1ID != -1 && p2ID != -1 { // -1 might mean unassigned
				isBoundarySiteResult[s1Idx] = true
				isBoundarySiteResult[s2Idx] = true
				adjacentPlatePairsSet[newFrozenset(p1ID, p2ID)] = true
			}
		}
	}
	boundarySiteCount := 0
	for _, isBoundary := range isBoundarySiteResult {
		if isBoundary {
			boundarySiteCount++
		}
	}
	fmt.Printf("  Identified %d boundary sites.\n", boundarySiteCount)
	fmt.Printf("  Found %d unique adjacent plate pairs.\n", len(adjacentPlatePairsSet))

	// Step 2: Calculate distance from each SITE to the nearest boundary SITE
	fmt.Println("  Calculating distances for sites to nearest boundary site (KD-Tree, parallel)...")
	startDistTime := time.Now()
	if boundarySiteCount > 0 {
		kdBoundaryPoints := make([]kdtree.Point, 0, boundarySiteCount)
		for i, isBoundary := range isBoundarySiteResult {
			if isBoundary {
				kdBoundaryPoints = append(kdBoundaryPoints, kdtreeBoundarySitePoint{Coordinates: icosphereSites[i], OriginalIndex: int32(i)})
			}
		}
		boundaryKdTree := kdtree.New(kdBoundaryPoints)

		numWorkers := getNumWorkers(numSites)
		var wg sync.WaitGroup
		chunkSize := (numSites + numWorkers - 1) / numWorkers

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				start := workerID * chunkSize
				end := (workerID + 1) * chunkSize
				if end > numSites {
					end = numSites
				}
				for i := start; i < end; i++ {
					if isBoundarySiteResult[i] {
						siteDistancesToBoundaryResult[i] = 0.0
						nearestBoundarySiteIndicesResult[i] = int32(i)
						continue
					}
					siteVec := icosphereSites[i]
					nearestNeighbors := boundaryKdTree.KNN(siteVec, 1)
					if len(nearestNeighbors) > 0 {
						if nearestBoundaryPt, ok := nearestNeighbors[0].(kdtreeBoundarySitePoint); ok {
							// Using Great Circle distance for points on a sphere
							// Assuming radius 1 for distance calculation here, can be scaled later if needed.
							// For simplicity, using Euclidean distance on normalized vectors as an approximation
							// which is related to chord length. True spherical distance is Acos(v1.Dot(v2)).
							dist := math.Acos(math.Max(-1.0, math.Min(1.0, siteVec.Dot(nearestBoundaryPt.Coordinates))))
							siteDistancesToBoundaryResult[i] = dist // This is angular distance if radius is 1
							nearestBoundarySiteIndicesResult[i] = nearestBoundaryPt.OriginalIndex
						} else {
							siteDistancesToBoundaryResult[i] = math.Inf(1)
							nearestBoundarySiteIndicesResult[i] = -1
						}
					} else {
						siteDistancesToBoundaryResult[i] = math.Inf(1)
						nearestBoundarySiteIndicesResult[i] = -1
					}
				}
			}(w)
		}
		wg.Wait()
		fmt.Printf("    Distances to boundary site calculated in %v.\n", time.Since(startDistTime))
	} else {
		fmt.Println("    Warning: No boundary sites found. Skipping distance calculation.")
		for i := 0; i < numSites; i++ {
			siteDistancesToBoundaryResult[i] = math.Inf(1)
			nearestBoundarySiteIndicesResult[i] = -1
		}
	}

	// Step 3: Determine interaction type for each ADJACENT PLATE PAIR
	fmt.Println("  Assigning boundary interaction types for adjacent plate pairs...")
	plateMap := make(map[int32]TectonicPlate)
	for _, p := range plates {
		plateMap[p.ID] = p
	}

	// Threshold for dot product to classify as convergent/divergent vs. passive
	// This value might need tuning. A larger value makes it easier to classify as passive.
	// Based on typical plate speeds (e.g., 0.01 to 0.1 m/yr, or similar in rad/Myr)
	// A small relative speed component along the normal.
	interactionThreshold := 1e-7 // Relative speed component threshold

	for pair := range adjacentPlatePairsSet {
		plate1ID := pair[0]
		plate2ID := pair[1]

		plate1, ok1 := plateMap[plate1ID]
		plate2, ok2 := plateMap[plate2ID]

		if !ok1 || !ok2 {
			fmt.Printf("Warning: Could not find plate data for pair (%d, %d).\n", plate1ID, plate2ID)
			adjacentPlateInteractionsResult[pair] = Passive
			continue
		}

		// Find a representative point on the boundary between these two plates.
		// Iterate over Delaunay edges. If an edge connects a site on plate1 and a site on plate2,
		// its midpoint is a candidate boundary point.
		var boundaryMidpoints []Vector3D
		var boundaryNormals []Vector3D // Normals pointing from plate1's domain towards plate2's

		for _, face := range icosphereFaces {
			sIndices := [3]int32{int32(face.V1), int32(face.V2), int32(face.V3)}
			pIDs := [3]int32{sitePlateIDs[sIndices[0]], sitePlateIDs[sIndices[1]], sitePlateIDs[sIndices[2]]}

			for i := 0; i < 3; i++ {
				s1Idx, s2Idx := sIndices[i], sIndices[(i+1)%3]
				currentP1ID, currentP2ID := pIDs[i], pIDs[(i+1)%3]

				// Check if this edge (s1Idx, s2Idx) is on the boundary of the current plate pair
				if (currentP1ID == plate1ID && currentP2ID == plate2ID) || (currentP1ID == plate2ID && currentP2ID == plate1ID) {
					site1Vec := icosphereSites[s1Idx]
					site2Vec := icosphereSites[s2Idx]
					midpoint := site1Vec.Add(site2Vec).Scale(0.5).Normalize()
					boundaryMidpoints = append(boundaryMidpoints, midpoint)

					// Normal to the boundary segment at the midpoint, pointing from plate1's region towards plate2's region.
					// The vector connecting the two sites is perpendicular to the Voronoi edge.
					interSiteVector := site2Vec.Subtract(site1Vec)
					if currentP1ID == plate2ID { // Ensure interSiteVector points from plate1's domain to plate2's
						interSiteVector = site1Vec.Subtract(site2Vec)
					}
					// Project interSiteVector onto the tangent plane at midpoint
					tangentPlaneNormal := midpoint // Normal of the sphere at midpoint
					boundaryNormal := interSiteVector.Subtract(tangentPlaneNormal.Scale(interSiteVector.Dot(tangentPlaneNormal))).Normalize()
					if boundaryNormal.LengthSq() > 1e-9 { // Ensure normal is valid
						boundaryNormals = append(boundaryNormals, boundaryNormal)
					}
				}
			}
		}

		if len(boundaryMidpoints) == 0 || len(boundaryNormals) == 0 {
			// Should not happen if pair was in adjacentPlatePairsSet from valid edges
			adjacentPlateInteractionsResult[pair] = Passive
			continue
		}

		avgNormalComponent := 0.0
		validSamples := 0
		for i, bp := range boundaryMidpoints {
			if i >= len(boundaryNormals) {
				break
			} // Should not happen if collected correctly

			vel1 := plate1.GetVelocityAtPoint(bp)
			vel2 := plate2.GetVelocityAtPoint(bp)
			relativeVel := vel2.Subtract(vel1) // Relative velocity of plate2 w.r.t. plate1

			normal := boundaryNormals[i]
			avgNormalComponent += relativeVel.Dot(normal)
			validSamples++
		}

		if validSamples > 0 {
			avgNormalComponent /= float64(validSamples)
			if avgNormalComponent > interactionThreshold { // Moving apart
				adjacentPlateInteractionsResult[pair] = Divergent
			} else if avgNormalComponent < -interactionThreshold { // Moving together
				adjacentPlateInteractionsResult[pair] = Convergent
			} else {
				// Could check tangential component for true transform/passive
				adjacentPlateInteractionsResult[pair] = Passive
			}
		} else {
			adjacentPlateInteractionsResult[pair] = Passive
		}
	}

	// Step 4: Assign a representative boundary type to each BOUNDARY SITE
	// A boundary site might be at the junction of multiple plate pairs.
	// Prioritize Convergent > Divergent > Passive.
	fmt.Println("  Mapping boundary sites to interaction types...")
	siteAdj := buildSiteAdjacencyList(icosphereFaces, numSites)

	for siteIdx := 0; siteIdx < numSites; siteIdx++ {
		if !isBoundarySiteResult[siteIdx] {
			continue // Only assign types to boundary sites
		}

		currentSitePlateID := sitePlateIDs[siteIdx]
		if currentSitePlateID == -1 {
			continue
		} // Unassigned site

		// Collect interaction types from all adjacent plate pairs this site is part of
		var siteInteractionTypes []PlateInteractionType

		neighborSiteIndices, ok := siteAdj[int32(siteIdx)]
		if !ok {
			continue
		}

		for _, neighborIdx := range neighborSiteIndices {
			neighborPlateID := sitePlateIDs[neighborIdx]
			if neighborPlateID != -1 && neighborPlateID != currentSitePlateID {
				pair := newFrozenset(currentSitePlateID, neighborPlateID)
				if interaction, exists := adjacentPlateInteractionsResult[pair]; exists {
					siteInteractionTypes = append(siteInteractionTypes, interaction)
				}
			}
		}

		// Determine dominant type for the site
		finalSiteType := Passive // Default
		hasConvergent := false
		hasDivergent := false
		for _, itype := range siteInteractionTypes {
			if itype == Convergent {
				hasConvergent = true
			}
			if itype == Divergent {
				hasDivergent = true
			}
		}

		if hasConvergent {
			finalSiteType = Convergent
		} else if hasDivergent {
			finalSiteType = Divergent
		}
		// If only Passive types, it remains Passive.

		siteBoundaryTypesResult[int32(siteIdx)] = finalSiteType
	}

	fmt.Println("Boundary finding and type assignment process complete.")
	return
}
