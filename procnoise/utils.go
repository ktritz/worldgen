package procnoise

import (
	"math"
	"math/rand"
	"worldgen/icosphere"
)

// ====================
// Private/implemenation
// ====================

// Utilities

func fastMin[T Float](x, y T) T {
	if x < y {
		return x
	}
	return y
}

func fastMax[T Float](x, y T) T {
	if x > y {
		return x
	}
	return y
}

func fastAbs[T Float](f T) T {
	if f < 0 {
		return -f
	}
	return f
}

func fastSqrt[T Float](a T) T {
	// Benchmarks using Quake's famous "inverse square root" were actually slightly slower than
	// using the built-in math library.
	return T(math.Sqrt(float64(a)))
}

func fastFloor[T Float](f T) int {
	if f >= 0 {
		return int(f)
	}
	return int(f) - 1
}

func fastRound[T Float](f T) int {
	if f >= 0 {
		return int(f + 0.5)
	}
	return int(f - 0.5)
}

func lerp[T Float](a, b, t T) T {
	return a + t*(b-a)
}

func interpHermite[T Float](t T) T {
	return t * t * (3 - 2*t)
}

func interpQuintic[T Float](t T) T {
	return t * t * t * (t*(t*6-15) + 10)
}

func cubicLerp[T Float](a, b, c, d, t T) T {
	var p T = (d - c) - (a - b)
	return t*t*t*p + t*t*((a-b)-p) + t*(c-a) + b
}

func pingPong[T Float](t T) T {
	t -= T(int(t*0.5)) * 2
	if t < 1 {
		return t
	}
	return 2 - t
}

func calculateFractalBounding[T Float](state *State[T]) T {
	gain := fastAbs(state.Gain)
	amp := gain
	var ampFractal T = 1.0
	for i := 1; i < state.Octaves; i++ {
		ampFractal += amp
		amp *= gain
	}
	return 1.0 / ampFractal
}

// Vec2 represents a 2D vector or point.
type Vec2 struct {
	X, Y float64
}

// Sub subtracts vector v2 from v1.
func (v1 Vec2) Sub(v2 Vec2) Vec2 {
	return Vec2{X: v1.X - v2.X, Y: v1.Y - v2.Y}
}

// LengthSq returns the squared length of the vector.
func (v Vec2) LengthSq() float64 {
	return v.X*v.X + v.Y*v.Y
}

// Vec3 represents a 3D vector or point.
type Vec3 struct {
	X, Y, Z float64
}

// Sub subtracts vector v2 from v1.
func (v1 Vec3) Sub(v2 Vec3) Vec3 {
	return Vec3{X: v1.X - v2.X, Y: v1.Y - v2.Y, Z: v1.Z - v2.Z}
}

// LengthSq returns the squared length of the vector.
func (v Vec3) LengthSq() float64 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

// Quaternion represents a quaternion for 3D rotations.
type Quaternion struct {
	X, Y, Z, W float64
}

// NewQuaternionIdentity creates an identity quaternion.
func NewQuaternionIdentity() Quaternion {
	return Quaternion{X: 0, Y: 0, Z: 0, W: 1}
}

// Inverse returns the inverse of the quaternion.
// Assumes it's a unit quaternion (rotation quaternion).
func (q Quaternion) Inverse() Quaternion {
	return Quaternion{X: -q.X, Y: -q.Y, Z: -q.Z, W: q.W}
}

// Rotate transforms vector v by the quaternion q.
func (q Quaternion) Rotate(v Vec3) Vec3 {
	// Simplified version for q * v * q^-1
	// Create pure quaternion from vector v
	p := Quaternion{X: v.X, Y: v.Y, Z: v.Z, W: 0}

	// q * p
	qp := Quaternion{
		W: q.W*p.W - q.X*p.X - q.Y*p.Y - q.Z*p.Z,
		X: q.W*p.X + q.X*p.W + q.Y*p.Z - q.Z*p.Y,
		Y: q.W*p.Y - q.X*p.Z + q.Y*p.W + q.Z*p.X,
		Z: q.W*p.Z + q.X*p.Y - q.Y*p.X + q.Z*p.W,
	}

	// (q * p) * q^-1
	qInv := q.Inverse()
	resQ := Quaternion{
		W: qp.W*qInv.W - qp.X*qInv.X - qp.Y*qInv.Y - qp.Z*qInv.Z,
		X: qp.W*qInv.X + qp.X*qInv.W + qp.Y*qInv.Z - qp.Z*qInv.Y,
		Y: qp.W*qInv.Y - qp.X*qInv.Z + qp.Y*qInv.W + qp.Z*qInv.X,
		Z: qp.W*qInv.Z + qp.X*qInv.Y - qp.Y*qInv.X + qp.Z*qInv.W,
	}
	return Vec3{X: resQ.X, Y: resQ.Y, Z: resQ.Z}
}

// FromAngleAxis creates a quaternion from an angle (in radians) and an axis.
// Axis must be a unit vector.
func FromAngleAxis(angle float64, axis Vec3) Quaternion {
	s := math.Sin(angle / 2)
	return Quaternion{
		X: axis.X * s,
		Y: axis.Y * s,
		Z: axis.Z * s,
		W: math.Cos(angle / 2),
	}
}

// BarycentricCoords stores barycentric coordinates for a point in a triangle.
type BarycentricCoords struct {
	U, V, W float32 // Weights for vertices v0, v1, v2 respectively
}

// SphericalGrid is an interface representing the geometry of a spherical grid.
type SphericalGrid interface {
	NumVertices() int
	GetVertex(index int) icosphere.Vector3D // Uses Vector3D from icosphere package
	FindTriangleAndBarycentricCoords(p icosphere.Vector3D) (v0Idx int, v1Idx int, v2Idx int, coords BarycentricCoords, found bool)
	GetVertexNeighbors(vertexIndex int) []int // Returns indices of 1-ring neighbors
}

// IcosphereModel implements SphericalGrid using vertex and face data.
// It's designed to wrap the output of your icosphere.CreateIcosphere function.
type IcosphereModel struct {
	Vertices      []icosphere.Vector3D // Uses Vector3D from icosphere package
	Faces         []icosphere.Triangle // Uses Triangle from icosphere package
	AdjacencyList [][]int              // AdjacencyList[i] stores neighbors of vertex i
}

// NewIcosphereModel creates an IcosphereModel.
// vertices and faces are typically from your icosphere.CreateIcosphere.
func NewIcosphereModel(vertices []icosphere.Vector3D, faces []icosphere.Triangle) *IcosphereModel {
	numVerts := len(vertices)
	adj := make([][]int, numVerts)
	for i := range adj {
		adj[i] = make([]int, 0)
	}

	processedEdges := make(map[[2]int]bool)

	for _, face := range faces {
		faceVertices := [3]int{face.V1, face.V2, face.V3}
		for i := 0; i < 3; i++ {
			u := faceVertices[i]
			v := faceVertices[(i+1)%3]

			edgeKey := [2]int{u, v}
			if u > v {
				edgeKey = [2]int{v, u}
			}

			if !processedEdges[edgeKey] {
				adj[u] = append(adj[u], v)
				adj[v] = append(adj[v], u)
				processedEdges[edgeKey] = true
			}
		}
	}

	return &IcosphereModel{
		Vertices:      vertices,
		Faces:         faces,
		AdjacencyList: adj,
	}
}

func (ic *IcosphereModel) NumVertices() int                       { return len(ic.Vertices) }
func (ic *IcosphereModel) GetVertex(index int) icosphere.Vector3D { return ic.Vertices[index] }
func (ic *IcosphereModel) GetVertexNeighbors(vertexIndex int) []int {
	if vertexIndex < 0 || vertexIndex >= len(ic.AdjacencyList) {
		return nil
	}
	return ic.AdjacencyList[vertexIndex]
}

// calculateBarycentric computes barycentric coordinates of p for planar triangle v0,v1,v2.
// Uses icosphere.Vector3D.
func calculateBarycentric(p, v0, v1, v2 icosphere.Vector3D) (float64, float64, float64, bool) {
	// Uses Subtract and Dot methods from icosphere.Vector3D
	v0v1 := v1.Subtract(v0)
	v0v2 := v2.Subtract(v0)
	v0p := p.Subtract(v0)

	d00 := v0v1.Dot(v0v1)
	d01 := v0v1.Dot(v0v2)
	d11 := v0v2.Dot(v0v2)
	d20 := v0p.Dot(v0v1)
	d21 := v0p.Dot(v0v2)

	denom := d00*d11 - d01*d01
	if math.Abs(denom) < 1e-9 {
		return 0, 0, 0, false
	}

	v_coord := (d11*d20 - d01*d21) / denom
	w_coord := (d00*d21 - d01*d20) / denom
	u_coord := 1.0 - v_coord - w_coord

	epsilon := 1e-5
	isInside := (u_coord >= -epsilon && u_coord <= 1.0+epsilon &&
		v_coord >= -epsilon && v_coord <= 1.0+epsilon &&
		w_coord >= -epsilon && w_coord <= 1.0+epsilon)

	return u_coord, v_coord, w_coord, isInside
}

func (ic *IcosphereModel) FindTriangleAndBarycentricCoords(p icosphere.Vector3D) (int, int, int, BarycentricCoords, bool) {
	// Uses Normalize method from icosphere.Vector3D
	pNormalized := p.Normalize()

	for _, face := range ic.Faces {
		v0 := ic.Vertices[face.V1]
		v1 := ic.Vertices[face.V2]
		v2 := ic.Vertices[face.V3]

		u, v, w, isInside := calculateBarycentric(pNormalized, v0, v1, v2)

		if isInside {
			return face.V1, face.V2, face.V3, BarycentricCoords{U: float32(u), V: float32(v), W: float32(w)}, true
		}
	}
	return 0, 0, 0, BarycentricCoords{}, false
}

// SphericalNoiseLayer stores noise values for a single spherical grid.
type SphericalNoiseLayer struct {
	Grid   SphericalGrid // Interface, implementations will use icosphere types
	Values []float32
}

// NewSphericalNoiseLayerRandom creates a new noise layer for the given spherical grid
// and initializes its values randomly.
func NewSphericalNoiseLayerRandom(grid SphericalGrid, rng *rand.Rand) *SphericalNoiseLayer {
	numVerts := grid.NumVertices()
	if numVerts <= 0 {
		panic("SphericalGrid must have a positive number of vertices")
	}
	values := make([]float32, numVerts)
	sqrt3 := float32(math.Sqrt(3))
	for i := 0; i < numVerts; i++ {
		values[i] = (rng.Float32()*2 - 1) * sqrt3
	}
	return &SphericalNoiseLayer{
		Grid:   grid,
		Values: values,
	}
}

// downsampleSphericalLayer creates a new noise layer for 'targetCoarserGrid'
// by averaging values from 'finerNoiseLayer'.
func downsampleSphericalLayer(finerNoiseLayer *SphericalNoiseLayer, targetCoarserGrid SphericalGrid) *SphericalNoiseLayer {
	numCoarseVerts := targetCoarserGrid.NumVertices()
	coarseValues := make([]float32, numCoarseVerts)
	finerGrid := finerNoiseLayer.Grid

	for i := 0; i < numCoarseVerts; i++ {
		var sumValues float32
		var count int

		if i >= len(finerNoiseLayer.Values) {
			if numCoarseVerts > 0 {
				coarseValues[i] = 0
			}
			continue
		}
		sumValues += finerNoiseLayer.Values[i]
		count++

		neighborsOnFinerGrid := finerGrid.GetVertexNeighbors(i)
		for _, neighborIdx := range neighborsOnFinerGrid {
			if neighborIdx >= 0 && neighborIdx < len(finerNoiseLayer.Values) {
				sumValues += finerNoiseLayer.Values[neighborIdx]
				count++
			}
		}

		if count > 0 {
			coarseValues[i] = sumValues / float32(count)
		} else {
			if i < len(finerNoiseLayer.Values) {
				coarseValues[i] = finerNoiseLayer.Values[i]
			} else if numCoarseVerts > 0 {
				coarseValues[i] = 0
			}
		}
	}

	return &SphericalNoiseLayer{
		Grid:   targetCoarserGrid,
		Values: coarseValues,
	}
}
