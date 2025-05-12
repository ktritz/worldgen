package icosphere

import "math"

// Vector3D represents a 3D vector or point.
type Vector3D struct {
	X, Y, Z float64
}

// Triangle represents a face with 3 vertex indices.
// The indices refer to positions in a vertices slice.
type Triangle struct {
	V1, V2, V3 int
}

// EdgeKey is used as a key in the midpointCache for icosphere generation.
// It stores two vertex indices in a canonical order.
type EdgeKey [2]int

// Add performs vector addition.
func (v Vector3D) Add(other Vector3D) Vector3D {
	return Vector3D{v.X + other.X, v.Y + other.Y, v.Z + other.Z}
}

// Subtract performs vector subtraction (v - other).
func (v Vector3D) Subtract(other Vector3D) Vector3D {
	return Vector3D{v.X - other.X, v.Y - other.Y, v.Z - other.Z}
}

// Scale performs scalar multiplication.
func (v Vector3D) Scale(scalar float64) Vector3D {
	return Vector3D{v.X * scalar, v.Y * scalar, v.Z * scalar}
}

// Dot calculates the dot product with another vector.
func (v Vector3D) Dot(other Vector3D) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// Cross calculates the cross product with another vector (v x other).
func (v Vector3D) Cross(other Vector3D) Vector3D {
	return Vector3D{
		X: v.Y*other.Z - v.Z*other.Y,
		Y: v.Z*other.X - v.X*other.Z,
		Z: v.X*other.Y - v.Y*other.X,
	}
}

// Length calculates the magnitude (length) of the vector.
func (v Vector3D) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// LengthSq calculates the squared magnitude of the vector.
func (v Vector3D) LengthSq() float64 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

// Normalize returns a unit vector in the same direction.
func (v Vector3D) Normalize() Vector3D {
	l := v.Length()
	if l < 1e-9 { // Avoid division by zero for zero-length vectors
		return Vector3D{0, 0, 0}
	}
	return Vector3D{v.X / l, v.Y / l, v.Z / l}
}

// --- Methods for kdtree.Point interface ---

// Dimensions returns the number of dimensions for the point (3 for Vector3D).
func (v Vector3D) Dimensions() int {
	return 3
}

// Dimension returns the value of the i-th dimension (0 for X, 1 for Y, 2 for Z).
func (v Vector3D) Dimension(i int) float64 {
	switch i {
	case 0:
		return v.X
	case 1:
		return v.Y
	case 2:
		return v.Z
	default:
		// Or panic, depending on how strict you want to be.
		// Returning 0 for out-of-bounds is a common approach for KD-trees
		// if they somehow query an invalid dimension, though it shouldn't happen
		// for a fixed-dimension point type.
		return 0
	}
}
