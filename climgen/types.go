// Package climgen implements climate generation including ocean currents,
// atmospheric circulation, and temperature/precipitation modeling.
package climgen

import (
	"fmt"
	"math"

	"worldgen/icosphere"
)

// --- Type Aliases for External Dependencies ---

// Vector3D uses the definition from the icosphere package.
type Vector3D = icosphere.Vector3D

// VoronoiCell uses the definition from the icosphere package.
type VoronoiCell = icosphere.VoronoiCell

// --- Ocean Current Constants ---
// Based on Earth ocean circulation parameters

const (
	// Basin Finding Parameters
	DefaultMinComponentSize      = 50    // Min baseline-equivalent (L5) cells for a component to start fitting; consumers rescale raw counts by the mesh area scale
	DefaultMinBasinRadiusRad     = 0.18  // ~10 degrees - balance between structure and coalescence
	DefaultCapRadiusIncrement    = 0.01  // Angular radius increment per iteration
	DefaultMaxExpansionIters     = 150   // Safety limit for cap expansion
	DefaultCenterShiftTolerance  = 1e-5  // Stop shifting if movement is below this
	DefaultPolarLimitDeg         = 60.0  // Latitude boundary for polar zones (degrees)
	DefaultNumZones              = 4     // Number of latitude zones

	// Current Generation Parameters
	DefaultTargetEdgeSpeed       = 0.5   // Target linear speed at gyre edge (normalized, ~1.25 m/s for Gulf Stream)
	DefaultSmoothingIterations   = 2     // Number of smoothing passes (less = preserve boundary gradients)
	DefaultSmoothingFactor       = 0.2   // Blend weight for neighbor averaging (0-1)
	DefaultCoastParallelBoost    = 1.0   // Multiplier for coast-parallel flow (1.0 = preserve magnitude)
	DefaultMaxAllowedSpeedSq     = 100.0 // Stability clamp (10.0^2)
)

// LatitudeZone represents the 4 latitude zones used for basin partitioning.
type LatitudeZone int

const (
	ZonePolarNorth LatitudeZone = 1
	ZoneMidNorth   LatitudeZone = 2
	ZoneMidSouth   LatitudeZone = 3
	ZonePolarSouth LatitudeZone = 4
)

// String returns the zone name.
func (z LatitudeZone) String() string {
	names := map[LatitudeZone]string{
		ZonePolarNorth: "PolarNorth",
		ZoneMidNorth:   "MidNorth",
		ZoneMidSouth:   "MidSouth",
		ZonePolarSouth: "PolarSouth",
	}
	if name, ok := names[z]; ok {
		return name
	}
	return "Unknown"
}

// --- Settings Structs ---

// BasinSettings controls ocean basin detection parameters.
type BasinSettings struct {
	MinComponentSize     int     `json:"minComponentSize"`     // Min baseline-equivalent (L5) cells to consider a component
	MinBasinRadiusRad    float64 `json:"minBasinRadiusRad"`    // Min angular radius for valid basin
	CapRadiusIncrement   float64 `json:"capRadiusIncrement"`   // Expansion step size (radians)
	MaxExpansionIters    int     `json:"maxExpansionIters"`    // Max iterations for cap expansion
	CenterShiftTolerance float64 `json:"centerShiftTolerance"` // Convergence threshold for shifting
	PolarLimitDeg        float64 `json:"polarLimitDeg"`        // Latitude boundary for polar zones
	NumZones             int     `json:"numZones"`             // Number of latitude zones
	Verbose              bool    `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s BasinSettings) Validate() error {
	if s.MinComponentSize < 1 {
		return fmt.Errorf("minComponentSize must be >= 1, got %d", s.MinComponentSize)
	}
	if s.MinBasinRadiusRad <= 0 || s.MinBasinRadiusRad > math.Pi {
		return fmt.Errorf("minBasinRadiusRad must be in (0, pi], got %f", s.MinBasinRadiusRad)
	}
	if s.CapRadiusIncrement <= 0 || s.CapRadiusIncrement > 0.5 {
		return fmt.Errorf("capRadiusIncrement must be in (0, 0.5], got %f", s.CapRadiusIncrement)
	}
	if s.MaxExpansionIters < 1 || s.MaxExpansionIters > 1000 {
		return fmt.Errorf("maxExpansionIters must be in [1, 1000], got %d", s.MaxExpansionIters)
	}
	if s.CenterShiftTolerance <= 0 {
		return fmt.Errorf("centerShiftTolerance must be positive, got %f", s.CenterShiftTolerance)
	}
	if s.PolarLimitDeg <= 0 || s.PolarLimitDeg >= 90 {
		return fmt.Errorf("polarLimitDeg must be in (0, 90), got %f", s.PolarLimitDeg)
	}
	if s.NumZones < 2 || s.NumZones > 8 {
		return fmt.Errorf("numZones must be in [2, 8], got %d", s.NumZones)
	}
	return nil
}

// DefaultBasinSettings returns Earth-like defaults for basin detection.
func DefaultBasinSettings() BasinSettings {
	return BasinSettings{
		MinComponentSize:     DefaultMinComponentSize,
		MinBasinRadiusRad:    DefaultMinBasinRadiusRad,
		CapRadiusIncrement:   DefaultCapRadiusIncrement,
		MaxExpansionIters:    DefaultMaxExpansionIters,
		CenterShiftTolerance: DefaultCenterShiftTolerance,
		PolarLimitDeg:        DefaultPolarLimitDeg,
		NumZones:             DefaultNumZones,
		Verbose:              false,
	}
}

// CurrentSettings controls ocean current generation parameters.
type CurrentSettings struct {
	TargetEdgeSpeed      float64 `json:"targetEdgeSpeed"`      // Target speed at gyre edge
	SmoothingIterations  int     `json:"smoothingIterations"`  // Number of diffusion passes
	SmoothingFactor      float64 `json:"smoothingFactor"`      // Blend weight (0-1)
	CoastParallelBoost   float64 `json:"coastParallelBoost"`   // Coast-parallel flow multiplier
	MaxAllowedSpeedSq    float64 `json:"maxAllowedSpeedSq"`    // Stability clamp
	Verbose              bool    `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s CurrentSettings) Validate() error {
	if s.TargetEdgeSpeed <= 0 || s.TargetEdgeSpeed > 1.0 {
		return fmt.Errorf("targetEdgeSpeed must be in (0, 1], got %f", s.TargetEdgeSpeed)
	}
	if s.SmoothingIterations < 0 || s.SmoothingIterations > 500 {
		return fmt.Errorf("smoothingIterations must be in [0, 500], got %d", s.SmoothingIterations)
	}
	if s.SmoothingFactor < 0 || s.SmoothingFactor > 1 {
		return fmt.Errorf("smoothingFactor must be in [0, 1], got %f", s.SmoothingFactor)
	}
	if s.CoastParallelBoost < 0 || s.CoastParallelBoost > 5.0 {
		return fmt.Errorf("coastParallelBoost must be in [0, 5], got %f", s.CoastParallelBoost)
	}
	if s.MaxAllowedSpeedSq <= 0 {
		return fmt.Errorf("maxAllowedSpeedSq must be positive, got %f", s.MaxAllowedSpeedSq)
	}
	return nil
}

// DefaultCurrentSettings returns Earth-like defaults for current generation.
func DefaultCurrentSettings() CurrentSettings {
	return CurrentSettings{
		TargetEdgeSpeed:     DefaultTargetEdgeSpeed,
		SmoothingIterations: DefaultSmoothingIterations,
		SmoothingFactor:     DefaultSmoothingFactor,
		CoastParallelBoost:  DefaultCoastParallelBoost,
		MaxAllowedSpeedSq:   DefaultMaxAllowedSpeedSq,
		Verbose:             false,
	}
}

// OceanCurrentSettings is the composite settings for full ocean current generation.
type OceanCurrentSettings struct {
	Seed    int64           `json:"seed"`
	Basin   BasinSettings   `json:"basin"`
	Current CurrentSettings `json:"current"`
	Verbose bool            `json:"verbose"`
}

// Validate checks all nested settings.
func (s OceanCurrentSettings) Validate() error {
	if err := s.Basin.Validate(); err != nil {
		return fmt.Errorf("basin settings: %w", err)
	}
	if err := s.Current.Validate(); err != nil {
		return fmt.Errorf("current settings: %w", err)
	}
	return nil
}

// DefaultOceanCurrentSettings returns Earth-like defaults.
func DefaultOceanCurrentSettings() OceanCurrentSettings {
	return OceanCurrentSettings{
		Seed:    42,
		Basin:   DefaultBasinSettings(),
		Current: DefaultCurrentSettings(),
		Verbose: false,
	}
}

// ApplyVerbose propagates verbose flag to all sub-settings.
func (s *OceanCurrentSettings) ApplyVerbose() {
	if s.Verbose {
		s.Basin.Verbose = true
		s.Current.Verbose = true
	}
}

// --- Data Structures ---

// Basin represents an ocean basin (gyre region).
type Basin struct {
	ID        int       // Unique basin identifier
	Zone      LatitudeZone
	Centroid  Vector3D  // Normalized centroid on unit sphere
	MaxRadius float64   // Maximum angular radius from centroid (radians)
	Vertices  []int     // Indices of vertices in this basin
}

// IsNorthern returns true if the basin is in the northern hemisphere.
func (b Basin) IsNorthern() bool {
	// Y-up coordinate system: positive Y is north
	return b.Centroid.Y >= 0
}

// FlatAdjacency is a Numba-style flat adjacency structure for efficient iteration.
type FlatAdjacency struct {
	Neighbors []int // Flat array of neighbor indices
	Offsets   []int // Offsets[i] = start position for vertex i's neighbors
}

// GetNeighbors returns the neighbors of vertex i.
func (fa *FlatAdjacency) GetNeighbors(i int) []int {
	if i < 0 || i >= len(fa.Offsets)-1 {
		return nil
	}
	start := fa.Offsets[i]
	end := fa.Offsets[i+1]
	return fa.Neighbors[start:end]
}

// OceanCurrentResult contains the output from ocean current generation.
type OceanCurrentResult struct {
	Currents         []Vector3D // Current vector at each vertex
	BasinAssignments []int      // Basin ID for each vertex (-1 if not in basin)
	Basins           []Basin    // List of detected basins
}

// --- Helper Functions ---

// Normalize returns a normalized copy of the vector.
func Normalize(v Vector3D) Vector3D {
	normSq := v.X*v.X + v.Y*v.Y + v.Z*v.Z
	if normSq < 1e-18 {
		return v
	}
	norm := math.Sqrt(normSq)
	return Vector3D{X: v.X / norm, Y: v.Y / norm, Z: v.Z / norm}
}

// Dot returns the dot product of two vectors.
func Dot(a, b Vector3D) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

// Cross returns the cross product of two vectors.
func Cross(a, b Vector3D) Vector3D {
	return Vector3D{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

// Scale returns the vector scaled by s.
func Scale(v Vector3D, s float64) Vector3D {
	return Vector3D{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

// Add returns the sum of two vectors.
func Add(a, b Vector3D) Vector3D {
	return Vector3D{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}

// Sub returns a - b.
func Sub(a, b Vector3D) Vector3D {
	return Vector3D{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}

// Length returns the magnitude of the vector.
func Length(v Vector3D) float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// LengthSq returns the squared magnitude of the vector.
func LengthSq(v Vector3D) float64 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

// AngularDistance returns the angular distance in radians between two unit vectors.
func AngularDistance(v1, v2 Vector3D) float64 {
	dot := Dot(v1, v2)
	// Clamp to [-1, 1] for numerical safety
	if dot > 1.0 {
		dot = 1.0
	} else if dot < -1.0 {
		dot = -1.0
	}
	return math.Acos(dot)
}

// GetLatitudeZone returns the latitude zone for a vertex (Y-up coordinate system).
func GetLatitudeZone(v Vector3D, polarLimitDeg float64) LatitudeZone {
	// Clamp Y to valid range
	y := v.Y
	if y > 1.0 {
		y = 1.0
	} else if y < -1.0 {
		y = -1.0
	}

	latRad := math.Asin(y)
	latDeg := latRad * 180.0 / math.Pi

	polarLimit := polarLimitDeg

	if latDeg > polarLimit {
		return ZonePolarNorth
	} else if latDeg >= 0 {
		return ZoneMidNorth
	} else if latDeg > -polarLimit {
		return ZoneMidSouth
	}
	return ZonePolarSouth
}

// GetTangentVectors returns local East and North tangent vectors at a point on unit sphere.
// Uses Y-up coordinate system where Y points to north pole.
func GetTangentVectors(v Vector3D) (east, north Vector3D) {
	up := Vector3D{X: 0, Y: 1, Z: 0}

	// East = v cross up (right-hand rule: at equator lon=0, this points toward +Z which is 90°E)
	east = Cross(v, up)
	eastLen := Length(east)

	if eastLen < 1e-12 {
		// Near poles - use alternative reference
		if v.Y > 0.999 {
			// North pole: east toward +X, north toward -Z
			east = Vector3D{X: 1, Y: 0, Z: 0}
			north = Vector3D{X: 0, Y: 0, Z: -1}
		} else if v.Y < -0.999 {
			// South pole: east toward +X, north toward +Z
			east = Vector3D{X: 1, Y: 0, Z: 0}
			north = Vector3D{X: 0, Y: 0, Z: 1}
		} else {
			// Use X-axis as stable reference
			stableRef := Vector3D{X: 1, Y: 0, Z: 0}
			north = Normalize(Cross(v, stableRef))
			east = Normalize(Cross(north, v))
		}
		return
	}

	east = Scale(east, 1.0/eastLen)
	// North = east cross v (perpendicular to both, pointing toward pole)
	north = Normalize(Cross(east, v))
	return
}

// Clamp constrains a value to [min, max].
func Clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// IsFinite returns true if the value is neither NaN nor infinite.
func IsFinite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

// CoalescenceMetrics contains quality metrics for ocean current generation.
type CoalescenceMetrics struct {
	FlowCoherence      float64 // Average alignment between neighboring currents (0-1)
	MultiBsinPct       float64 // Fraction of vertices influenced by 2+ basins
	SpeedCoV           float64 // Coefficient of variation of current speeds
	BoundarySmoothness float64 // Smoothness of transitions at basin boundaries (0-1)
	AvgVorticity       float64 // Average absolute vorticity (higher = more circular flow)
	VorticityRatio     float64 // Ratio of rotational to total flow (0=linear, 1=circular)
}

// ComputeCoalescenceMetrics calculates quality metrics for the generated currents.
func ComputeCoalescenceMetrics(
	currents []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	basins []Basin,
) CoalescenceMetrics {
	var metrics CoalescenceMetrics

	// 1. Flow coherence: average dot product between neighboring current vectors
	totalCoherence := 0.0
	coherenceCount := 0
	for i := range vertices {
		if elevation[i] >= seaLevelThreshold {
			continue
		}
		ci := currents[i]
		ciLen := Length(ci)
		if ciLen < 1e-9 {
			continue
		}
		ciNorm := Scale(ci, 1.0/ciLen)

		for _, j := range adj.GetNeighbors(i) {
			if j <= i || elevation[j] >= seaLevelThreshold {
				continue // Avoid double-counting
			}
			cj := currents[j]
			cjLen := Length(cj)
			if cjLen < 1e-9 {
				continue
			}
			cjNorm := Scale(cj, 1.0/cjLen)

			dot := Dot(ciNorm, cjNorm)
			totalCoherence += dot
			coherenceCount++
		}
	}
	if coherenceCount > 0 {
		metrics.FlowCoherence = totalCoherence / float64(coherenceCount)
	}

	// 2. Multi-basin influence: count vertices receiving influence from 2+ basins
	multiBsinCount := 0
	waterCount := 0
	for i := range vertices {
		if elevation[i] >= seaLevelThreshold {
			continue
		}
		waterCount++

		basinInfluences := 0
		for _, basin := range basins {
			dist := AngularDistance(vertices[i], basin.Centroid)
			influenceRadius := basin.MaxRadius * 3.5
			if dist <= influenceRadius {
				basinInfluences++
			}
		}
		if basinInfluences >= 2 {
			multiBsinCount++
		}
	}
	if waterCount > 0 {
		metrics.MultiBsinPct = float64(multiBsinCount) / float64(waterCount)
	}

	// 3. Speed uniformity: coefficient of variation (std dev / mean)
	speeds := make([]float64, 0, waterCount)
	for i := range vertices {
		if elevation[i] >= seaLevelThreshold {
			continue
		}
		speed := Length(currents[i])
		if speed > 1e-9 {
			speeds = append(speeds, speed)
		}
	}
	if len(speeds) > 1 {
		mean := 0.0
		for _, s := range speeds {
			mean += s
		}
		mean /= float64(len(speeds))

		variance := 0.0
		for _, s := range speeds {
			d := s - mean
			variance += d * d
		}
		variance /= float64(len(speeds))
		stdDev := math.Sqrt(variance)

		if mean > 1e-9 {
			metrics.SpeedCoV = stdDev / mean
		}
	}

	// 4. Boundary smoothness: coherence specifically at basin boundaries
	boundaryCoherence := 0.0
	boundaryCount := 0
	for i := range vertices {
		if elevation[i] >= seaLevelThreshold {
			continue
		}

		// Check if this vertex is near a basin boundary (influenced by 2+ basins with similar weight)
		var influences []struct {
			basinID int
			weight  float64
		}
		for bi, basin := range basins {
			dist := AngularDistance(vertices[i], basin.Centroid)
			influenceRadius := basin.MaxRadius * 3.5
			if dist <= influenceRadius {
				weight := 1.0 - (dist / influenceRadius)
				weight = weight * weight * basin.MaxRadius
				influences = append(influences, struct {
					basinID int
					weight  float64
				}{bi, weight})
			}
		}

		// If influenced by 2+ basins with comparable weights, this is a boundary
		if len(influences) >= 2 {
			// Sort by weight and check if top 2 are comparable
			maxW, secondW := 0.0, 0.0
			for _, inf := range influences {
				if inf.weight > maxW {
					secondW = maxW
					maxW = inf.weight
				} else if inf.weight > secondW {
					secondW = inf.weight
				}
			}

			// Boundary if second strongest is at least 30% of strongest
			if maxW > 0 && secondW/maxW > 0.3 {
				ci := currents[i]
				ciLen := Length(ci)
				if ciLen > 1e-9 {
					ciNorm := Scale(ci, 1.0/ciLen)

					for _, j := range adj.GetNeighbors(i) {
						if elevation[j] >= seaLevelThreshold {
							continue
						}
						cj := currents[j]
						cjLen := Length(cj)
						if cjLen < 1e-9 {
							continue
						}
						cjNorm := Scale(cj, 1.0/cjLen)

						dot := Dot(ciNorm, cjNorm)
						boundaryCoherence += dot
						boundaryCount++
					}
				}
			}
		}
	}
	if boundaryCount > 0 {
		metrics.BoundarySmoothness = boundaryCoherence / float64(boundaryCount)
	}

	// 5. Vorticity: measure how much flow rotates around each vertex
	// Higher vorticity = more circular gyre structure, lower = more linear flow
	totalVorticity := 0.0
	totalSpeed := 0.0
	vorticityCount := 0
	for i := range vertices {
		if elevation[i] >= seaLevelThreshold {
			continue
		}

		ci := currents[i]
		ciSpeed := Length(ci)
		if ciSpeed < 1e-9 {
			continue
		}

		// Get surface normal at this vertex
		normal := vertices[i] // Unit sphere, so vertex IS the normal

		// Compute discrete curl by looking at circulation around vertex
		// Sum of (edge_tangent · neighbor_velocity) around the vertex
		neighbors := adj.GetNeighbors(i)
		if len(neighbors) < 3 {
			continue
		}

		circulation := 0.0
		edgeSum := 0.0
		for ni, j := range neighbors {
			if j < 0 || j >= len(vertices) || elevation[j] >= seaLevelThreshold {
				continue
			}

			// Next neighbor (wrapping)
			nextIdx := (ni + 1) % len(neighbors)
			k := neighbors[nextIdx]
			if k < 0 || k >= len(vertices) {
				continue
			}

			// Edge from j to k (tangent around vertex i)
			edge := Sub(vertices[k], vertices[j])
			// Project edge onto tangent plane
			dotN := Dot(edge, normal)
			edgeTangent := Sub(edge, Scale(normal, dotN))
			edgeLen := Length(edgeTangent)
			if edgeLen < 1e-9 {
				continue
			}
			edgeTangent = Scale(edgeTangent, 1.0/edgeLen)

			// Velocity at j projected to tangent plane
			cj := currents[j]
			dotNj := Dot(cj, normal)
			cjTangent := Sub(cj, Scale(normal, dotNj))

			// Contribution to circulation
			circulation += Dot(edgeTangent, cjTangent)
			edgeSum += edgeLen
		}

		if edgeSum > 1e-9 {
			// Normalize by edge lengths to get vorticity
			localVorticity := math.Abs(circulation / edgeSum)
			totalVorticity += localVorticity
			totalSpeed += ciSpeed
			vorticityCount++
		}
	}

	if vorticityCount > 0 {
		metrics.AvgVorticity = totalVorticity / float64(vorticityCount)
		avgSpeed := totalSpeed / float64(vorticityCount)
		if avgSpeed > 1e-9 {
			// Ratio of rotational flow to total flow
			// Normalized so 1.0 means all flow is rotational
			metrics.VorticityRatio = metrics.AvgVorticity / avgSpeed
		}
	}

	return metrics
}
