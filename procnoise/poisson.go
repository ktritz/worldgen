package procnoise

import (
	"math"
	"math/rand"
	"worldgen/icosphere"
)

// --- Poisson Disk Sampling Implementation ---

// PoissonSample2D represents a single sample point in 2D space.
type PoissonSample2D struct {
	Position Vec2    // Position of the sample
	Value    float64 // Optional value associated with the sample
}

// PoissonSample3D represents a single sample point in 3D space.
type PoissonSample3D struct {
	Position icosphere.Vector3D // Position of the sample
	Value    float64            // Optional value associated with the sample
}

// SphericalPoissonSample represents a sample point on a sphere surface.
type SphericalPoissonSample struct {
	Position icosphere.Vector3D // Normalized position on unit sphere
	Value    float64            // Optional value associated with the sample
}

// PoissonDiskSampler2D generates Poisson disk samples in 2D space.
type PoissonDiskSampler2D struct {
	MinDistance float64              // Minimum distance between samples
	MaxDistance float64              // Maximum distance for new sample attempts
	Domain      [4]float64           // Domain bounds [minX, minY, maxX, maxY]
	Samples     []PoissonSample2D    // Generated samples
	Grid        [][]int              // Spatial grid for fast neighbor lookup
	GridSize    float64              // Size of each grid cell
	GridWidth   int                  // Width of the grid
	GridHeight  int                  // Height of the grid
	RNG         *rand.Rand           // Random number generator
	MaxAttempts int                  // Maximum attempts per sample
}

// NewPoissonDiskSampler2D creates a new 2D Poisson disk sampler.
func NewPoissonDiskSampler2D(seed int64, minDistance float64, domain [4]float64) *PoissonDiskSampler2D {
	if minDistance <= 0 {
		panic("MinDistance must be positive")
	}
	if domain[2] <= domain[0] || domain[3] <= domain[1] {
		panic("Invalid domain bounds")
	}

	// Grid cell size should be minDistance / sqrt(2) for 2D
	gridSize := minDistance / math.Sqrt(2)
	gridWidth := int(math.Ceil((domain[2] - domain[0]) / gridSize))
	gridHeight := int(math.Ceil((domain[3] - domain[1]) / gridSize))

	grid := make([][]int, gridHeight)
	for i := range grid {
		grid[i] = make([]int, gridWidth)
		for j := range grid[i] {
			grid[i][j] = -1 // -1 indicates empty cell
		}
	}

	return &PoissonDiskSampler2D{
		MinDistance: minDistance,
		MaxDistance: minDistance * 2.0, // Default max distance
		Domain:      domain,
		Samples:     make([]PoissonSample2D, 0),
		Grid:        grid,
		GridSize:    gridSize,
		GridWidth:   gridWidth,
		GridHeight:  gridHeight,
		RNG:         rand.New(rand.NewSource(seed)),
		MaxAttempts: 30, // Default attempts per sample
	}
}

// SetMaxDistance sets the maximum distance for new sample generation.
func (pds *PoissonDiskSampler2D) SetMaxDistance(maxDistance float64) {
	if maxDistance >= pds.MinDistance {
		pds.MaxDistance = maxDistance
	}
}

// SetMaxAttempts sets the maximum number of attempts per sample.
func (pds *PoissonDiskSampler2D) SetMaxAttempts(maxAttempts int) {
	if maxAttempts > 0 {
		pds.MaxAttempts = maxAttempts
	}
}

// getGridCoords converts world coordinates to grid coordinates.
func (pds *PoissonDiskSampler2D) getGridCoords(pos Vec2) (int, int) {
	x := int((pos.X - pds.Domain[0]) / pds.GridSize)
	y := int((pos.Y - pds.Domain[1]) / pds.GridSize)
	return x, y
}

// isValidSample checks if a sample position is valid (not too close to existing samples).
func (pds *PoissonDiskSampler2D) isValidSample(pos Vec2) bool {
	// Check if position is within domain
	if pos.X < pds.Domain[0] || pos.X >= pds.Domain[2] ||
		pos.Y < pds.Domain[1] || pos.Y >= pds.Domain[3] {
		return false
	}

	gx, gy := pds.getGridCoords(pos)
	
	// Check surrounding grid cells for nearby samples
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			nx, ny := gx+dx, gy+dy
			if nx >= 0 && nx < pds.GridWidth && ny >= 0 && ny < pds.GridHeight {
				sampleIdx := pds.Grid[ny][nx]
				if sampleIdx >= 0 {
					existingSample := pds.Samples[sampleIdx]
					dist := pos.Sub(existingSample.Position).LengthSq()
					if dist < pds.MinDistance*pds.MinDistance {
						return false
					}
				}
			}
		}
	}
	return true
}

// addSample adds a sample to the grid and sample list.
func (pds *PoissonDiskSampler2D) addSample(pos Vec2, value float64) {
	sample := PoissonSample2D{Position: pos, Value: value}
	sampleIdx := len(pds.Samples)
	pds.Samples = append(pds.Samples, sample)
	
	gx, gy := pds.getGridCoords(pos)
	pds.Grid[gy][gx] = sampleIdx
}

// generateAroundSample generates a new sample in the annulus around an existing sample.
func (pds *PoissonDiskSampler2D) generateAroundSample(centerSample PoissonSample2D) (Vec2, bool) {
	for attempt := 0; attempt < pds.MaxAttempts; attempt++ {
		// Generate random point in annulus between MinDistance and MaxDistance
		angle := pds.RNG.Float64() * 2 * math.Pi
		radius := pds.MinDistance + pds.RNG.Float64()*(pds.MaxDistance-pds.MinDistance)
		
		newPos := Vec2{
			X: centerSample.Position.X + radius*math.Cos(angle),
			Y: centerSample.Position.Y + radius*math.Sin(angle),
		}
		
		if pds.isValidSample(newPos) {
			return newPos, true
		}
	}
	return Vec2{}, false
}

// Generate creates Poisson disk samples using Mitchell's algorithm.
func (pds *PoissonDiskSampler2D) Generate() []PoissonSample2D {
	pds.Samples = pds.Samples[:0] // Clear existing samples
	
	// Reset grid
	for i := range pds.Grid {
		for j := range pds.Grid[i] {
			pds.Grid[i][j] = -1
		}
	}

	// Generate initial sample
	initialPos := Vec2{
		X: pds.Domain[0] + pds.RNG.Float64()*(pds.Domain[2]-pds.Domain[0]),
		Y: pds.Domain[1] + pds.RNG.Float64()*(pds.Domain[3]-pds.Domain[1]),
	}
	pds.addSample(initialPos, 1.0)

	// Active list for generating new samples
	activeList := []int{0}

	for len(activeList) > 0 {
		// Pick random active sample
		activeIdx := pds.RNG.Intn(len(activeList))
		sampleIdx := activeList[activeIdx]
		centerSample := pds.Samples[sampleIdx]

		newPos, found := pds.generateAroundSample(centerSample)
		if found {
			pds.addSample(newPos, 1.0)
			activeList = append(activeList, len(pds.Samples)-1)
		} else {
			// Remove from active list
			activeList[activeIdx] = activeList[len(activeList)-1]
			activeList = activeList[:len(activeList)-1]
		}
	}

	return pds.Samples
}

// GetSamples returns the generated samples.
func (pds *PoissonDiskSampler2D) GetSamples() []PoissonSample2D {
	return pds.Samples
}

// GetSampleCount returns the number of generated samples.
func (pds *PoissonDiskSampler2D) GetSampleCount() int {
	return len(pds.Samples)
}

// --- Spherical Poisson Disk Sampling ---

// SphericalPoissonDiskSampler generates Poisson disk samples on a sphere surface.
type SphericalPoissonDiskSampler struct {
	MinAngularDistance float64                      // Minimum angular distance between samples (radians)
	MaxAngularDistance float64                      // Maximum angular distance for new sample attempts
	Samples            []SphericalPoissonSample     // Generated samples
	RNG                *rand.Rand                   // Random number generator
	MaxAttempts        int                          // Maximum attempts per sample
}

// NewSphericalPoissonDiskSampler creates a new spherical Poisson disk sampler.
func NewSphericalPoissonDiskSampler(seed int64, minAngularDistance float64) *SphericalPoissonDiskSampler {
	if minAngularDistance <= 0 || minAngularDistance >= math.Pi {
		panic("MinAngularDistance must be between 0 and π")
	}

	return &SphericalPoissonDiskSampler{
		MinAngularDistance: minAngularDistance,
		MaxAngularDistance: minAngularDistance * 2.0,
		Samples:            make([]SphericalPoissonSample, 0),
		RNG:                rand.New(rand.NewSource(seed)),
		MaxAttempts:        30,
	}
}

// SetMaxAngularDistance sets the maximum angular distance for new sample generation.
func (spds *SphericalPoissonDiskSampler) SetMaxAngularDistance(maxAngularDistance float64) {
	if maxAngularDistance >= spds.MinAngularDistance && maxAngularDistance <= math.Pi {
		spds.MaxAngularDistance = maxAngularDistance
	}
}

// SetMaxAttempts sets the maximum number of attempts per sample.
func (spds *SphericalPoissonDiskSampler) SetMaxAttempts(maxAttempts int) {
	if maxAttempts > 0 {
		spds.MaxAttempts = maxAttempts
	}
}

// angularDistance calculates the angular distance between two points on a unit sphere.
func (spds *SphericalPoissonDiskSampler) angularDistance(p1, p2 icosphere.Vector3D) float64 {
	// Ensure vectors are normalized
	p1 = p1.Normalize()
	p2 = p2.Normalize()
	
	// Use dot product to find angle
	dot := p1.Dot(p2)
	// Clamp to avoid numerical issues
	if dot > 1.0 {
		dot = 1.0
	} else if dot < -1.0 {
		dot = -1.0
	}
	
	return math.Acos(dot)
}

// isValidSphericalSample checks if a sample position is valid on the sphere.
func (spds *SphericalPoissonDiskSampler) isValidSphericalSample(pos icosphere.Vector3D) bool {
	pos = pos.Normalize()
	
	for _, sample := range spds.Samples {
		dist := spds.angularDistance(pos, sample.Position)
		if dist < spds.MinAngularDistance {
			return false
		}
	}
	return true
}

// generateRandomPointOnSphere generates a random point on the unit sphere.
func (spds *SphericalPoissonDiskSampler) generateRandomPointOnSphere() icosphere.Vector3D {
	// Use marsaglia method for uniform distribution on sphere
	for {
		x := spds.RNG.Float64()*2 - 1
		y := spds.RNG.Float64()*2 - 1
		if x*x+y*y < 1 {
			z := spds.RNG.Float64()*2 - 1
			point := icosphere.Vector3D{X: x, Y: y, Z: z}
			return point.Normalize()
		}
	}
}

// generateAroundSphericalSample generates a new sample in the spherical cap around an existing sample.
func (spds *SphericalPoissonDiskSampler) generateAroundSphericalSample(centerSample SphericalPoissonSample) (icosphere.Vector3D, bool) {
	center := centerSample.Position.Normalize()
	
	for attempt := 0; attempt < spds.MaxAttempts; attempt++ {
		// Generate random angle in the annulus
		minAngle := spds.MinAngularDistance
		maxAngle := spds.MaxAngularDistance
		angle := minAngle + spds.RNG.Float64()*(maxAngle-minAngle)
		
		// Use spherical linear interpolation to get point at desired angular distance
		// This is a simplified approach - more sophisticated methods exist
		weight := math.Cos(angle)
		perpComponent := math.Sin(angle)
		
		// Create a vector perpendicular to center
		var perpVector icosphere.Vector3D
		if math.Abs(center.X) < 0.9 {
			perpVector = icosphere.Vector3D{X: 1, Y: 0, Z: 0}
		} else {
			perpVector = icosphere.Vector3D{X: 0, Y: 1, Z: 0}
		}
		
		// Make it truly perpendicular
		perpVector = perpVector.Subtract(center.Scale(perpVector.Dot(center))).Normalize()
		
		// Generate random rotation around center
		rotAngle := spds.RNG.Float64() * 2 * math.Pi
		rotatedPerp := perpVector.Scale(math.Cos(rotAngle)).Add(
			center.Cross(perpVector).Scale(math.Sin(rotAngle)))
		
		// Combine to get final position
		newPos := center.Scale(weight).Add(rotatedPerp.Scale(perpComponent))
		newPos = newPos.Normalize()
		
		if spds.isValidSphericalSample(newPos) {
			return newPos, true
		}
	}
	return icosphere.Vector3D{}, false
}

// Generate creates Poisson disk samples on the sphere surface.
func (spds *SphericalPoissonDiskSampler) Generate() []SphericalPoissonSample {
	spds.Samples = spds.Samples[:0] // Clear existing samples
	
	// Generate initial sample
	initialPos := spds.generateRandomPointOnSphere()
	spds.Samples = append(spds.Samples, SphericalPoissonSample{
		Position: initialPos,
		Value:    1.0,
	})

	// Active list for generating new samples
	activeList := []int{0}

	for len(activeList) > 0 {
		// Pick random active sample
		activeIdx := spds.RNG.Intn(len(activeList))
		sampleIdx := activeList[activeIdx]
		centerSample := spds.Samples[sampleIdx]

		newPos, found := spds.generateAroundSphericalSample(centerSample)
		if found {
			spds.Samples = append(spds.Samples, SphericalPoissonSample{
				Position: newPos,
				Value:    1.0,
			})
			activeList = append(activeList, len(spds.Samples)-1)
		} else {
			// Remove from active list
			activeList[activeIdx] = activeList[len(activeList)-1]
			activeList = activeList[:len(activeList)-1]
		}
	}

	return spds.Samples
}

// GetSamples returns the generated samples.
func (spds *SphericalPoissonDiskSampler) GetSamples() []SphericalPoissonSample {
	return spds.Samples
}

// GetSampleCount returns the number of generated samples.
func (spds *SphericalPoissonDiskSampler) GetSampleCount() int {
	return len(spds.Samples)
}

// --- Poisson Disk Noise Generator ---

// PoissonDiskNoise2D generates noise values based on Poisson disk samples.
type PoissonDiskNoise2D struct {
	Sampler      *PoissonDiskSampler2D
	FalloffType  string  // "linear", "quadratic", "exponential"
	MaxInfluence float64 // Maximum distance of influence for each sample
}

// NewPoissonDiskNoise2D creates a new 2D Poisson disk noise generator.
func NewPoissonDiskNoise2D(sampler *PoissonDiskSampler2D, falloffType string, maxInfluence float64) *PoissonDiskNoise2D {
	if sampler == nil {
		panic("PoissonDiskSampler2D cannot be nil")
	}
	if maxInfluence <= 0 {
		maxInfluence = sampler.MinDistance * 2.0
	}

	return &PoissonDiskNoise2D{
		Sampler:      sampler,
		FalloffType:  falloffType,
		MaxInfluence: maxInfluence,
	}
}

// GetNoise evaluates the Poisson disk noise at a given 2D position.
func (pdn *PoissonDiskNoise2D) GetNoise(pos Vec2) float64 {
	if len(pdn.Sampler.Samples) == 0 {
		return 0.0
	}

	totalWeight := 0.0
	totalValue := 0.0

	for _, sample := range pdn.Sampler.Samples {
		distance := pos.Sub(sample.Position).Length()
		
		if distance <= pdn.MaxInfluence {
			var weight float64
			normalizedDist := distance / pdn.MaxInfluence
			
			switch pdn.FalloffType {
			case "linear":
				weight = 1.0 - normalizedDist
			case "quadratic":
				weight = (1.0 - normalizedDist) * (1.0 - normalizedDist)
			case "exponential":
				weight = math.Exp(-normalizedDist * 3.0)
			default:
				weight = 1.0 - normalizedDist
			}
			
			totalWeight += weight
			totalValue += weight * sample.Value
		}
	}

	if totalWeight > 0 {
		return totalValue / totalWeight
	}
	return 0.0
}

// --- Spherical Poisson Disk Noise Generator ---

// SphericalPoissonDiskNoise generates noise values based on spherical Poisson disk samples.
type SphericalPoissonDiskNoise struct {
	Sampler      *SphericalPoissonDiskSampler
	FalloffType  string  // "linear", "quadratic", "exponential"
	MaxInfluence float64 // Maximum angular distance of influence (radians)
}

// NewSphericalPoissonDiskNoise creates a new spherical Poisson disk noise generator.
func NewSphericalPoissonDiskNoise(sampler *SphericalPoissonDiskSampler, falloffType string, maxInfluence float64) *SphericalPoissonDiskNoise {
	if sampler == nil {
		panic("SphericalPoissonDiskSampler cannot be nil")
	}
	if maxInfluence <= 0 {
		maxInfluence = sampler.MinAngularDistance * 2.0
	}

	return &SphericalPoissonDiskNoise{
		Sampler:      sampler,
		FalloffType:  falloffType,
		MaxInfluence: maxInfluence,
	}
}

// GetNoise evaluates the spherical Poisson disk noise at a given 3D position.
// Implements ScalarField3D interface.
func (spdn *SphericalPoissonDiskNoise) GetNoise(pos icosphere.Vector3D) float32 {
	if len(spdn.Sampler.Samples) == 0 {
		return 0.0
	}

	pos = pos.Normalize()
	totalWeight := 0.0
	totalValue := 0.0

	for _, sample := range spdn.Sampler.Samples {
		distance := spdn.Sampler.angularDistance(pos, sample.Position)
		
		if distance <= spdn.MaxInfluence {
			var weight float64
			normalizedDist := distance / spdn.MaxInfluence
			
			switch spdn.FalloffType {
			case "linear":
				weight = 1.0 - normalizedDist
			case "quadratic":
				weight = (1.0 - normalizedDist) * (1.0 - normalizedDist)
			case "exponential":
				weight = math.Exp(-normalizedDist * 3.0)
			default:
				weight = 1.0 - normalizedDist
			}
			
			totalWeight += weight
			totalValue += weight * sample.Value
		}
	}

	if totalWeight > 0 {
		return float32(totalValue / totalWeight)
	}
	return 0.0
}