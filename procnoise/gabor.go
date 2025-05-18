package procnoise

import (
	"math"
	"math/rand"
)

// Impulse2D stores parameters for a single 2D Gabor impulse.
type Impulse2D struct {
	Position    Vec2    // Center of the impulse kernel
	Weight      float64 // Contribution weight of this impulse
	Frequency   float64 // Frequency of the sinusoidal component
	Orientation float64 // Orientation angle (radians) of the kernel's local X-axis
	Bandwidth   float64 // Controls the falloff rate of the Gaussian envelope
	Phase       float64 // Phase offset of the sinusoidal component (radians)
	AspectRatio float64 // Controls ellipticity of Gaussian: Y-size / X-size in kernel's local space
	CutoffSq    float64 // Squared cutoff radius for this impulse's influence
}

// ImpulseConfig2D defines the range of parameters for generating 2D impulses.
type ImpulseConfig2D struct {
	NumImpulses int
	DomainMin   Vec2
	DomainMax   Vec2

	MinFrequency float64
	MaxFrequency float64

	MinBandwidth float64 // Must be > 0
	MaxBandwidth float64

	MinAspectRatio float64 // Must be > 0
	MaxAspectRatio float64

	MinWeight float64
	MaxWeight float64

	// If true, orientation is random (0 to 2*PI), otherwise 0.
	RandomizeOrientation bool
	// If true, phase is random (0 to 2*PI), otherwise 0.
	RandomizePhase bool

	// KernelCutoffFactor determines the cutoff radius: cutoff = KernelCutoffFactor / MinBandwidth.
	// A value around 3-4 means Gaussian drops to a very small value.
	KernelCutoffFactor float64
}

// GaborNoise2D is a generator for 2D Gabor noise.
type GaborNoise2D struct {
	impulses []Impulse2D
	rng      *rand.Rand
}

// randFloat64 returns a random float64 in [min, max).
func randFloat64(rng *rand.Rand, min, max float64) float64 {
	return min + rng.Float64()*(max-min)
}

// NewGaborNoise2D creates and initializes a 2D Gabor noise generator.
func NewGaborNoise2D(seed int64, cfg ImpulseConfig2D) *GaborNoise2D {
	if cfg.MinBandwidth <= 0 {
		panic("MinBandwidth must be greater than 0")
	}
	if cfg.MinAspectRatio <= 0 {
		panic("MinAspectRatio must be greater than 0")
	}
	if cfg.KernelCutoffFactor <= 0 {
		cfg.KernelCutoffFactor = 3.0 // Default cutoff factor
	}

	rng := rand.New(rand.NewSource(seed))
	impulses := make([]Impulse2D, cfg.NumImpulses)
	baseCutoffRadius := cfg.KernelCutoffFactor / cfg.MinBandwidth

	for i := 0; i < cfg.NumImpulses; i++ {
		pos := Vec2{
			X: randFloat64(rng, cfg.DomainMin.X, cfg.DomainMax.X),
			Y: randFloat64(rng, cfg.DomainMin.Y, cfg.DomainMax.Y),
		}
		freq := randFloat64(rng, cfg.MinFrequency, cfg.MaxFrequency)
		bandwidth := randFloat64(rng, cfg.MinBandwidth, cfg.MaxBandwidth)
		aspectRatio := randFloat64(rng, cfg.MinAspectRatio, cfg.MaxAspectRatio)
		weight := randFloat64(rng, cfg.MinWeight, cfg.MaxWeight)

		orientation := 0.0
		if cfg.RandomizeOrientation {
			orientation = rng.Float64() * 2 * math.Pi
		}
		phase := 0.0
		if cfg.RandomizePhase {
			phase = rng.Float64() * 2 * math.Pi
		}

		// Individual cutoff can be more precise if bandwidth varies a lot
		// For simplicity, using a general cutoff based on MinBandwidth.
		// A more precise cutoff for this impulse would be: cfg.KernelCutoffFactor / bandwidth
		// We use a common one for now for simpler culling.
		impulseCutoff := cfg.KernelCutoffFactor / bandwidth
		if impulseCutoff > baseCutoffRadius { // Ensure it's not larger than the general search radius
			impulseCutoff = baseCutoffRadius
		}

		impulses[i] = Impulse2D{
			Position:    pos,
			Weight:      weight,
			Frequency:   freq,
			Orientation: orientation,
			Bandwidth:   bandwidth,
			Phase:       phase,
			AspectRatio: aspectRatio,
			CutoffSq:    impulseCutoff * impulseCutoff,
		}
	}

	return &GaborNoise2D{
		impulses: impulses,
		rng:      rng,
	}
}

// GetNoise evaluates the 2D Gabor noise at point p.
func (gn *GaborNoise2D) GetNoise(p Vec2) float64 {
	var totalNoise float64

	for _, imp := range gn.impulses {
		delta := p.Sub(imp.Position)

		// Basic culling based on squared distance
		if delta.LengthSq() > imp.CutoffSq {
			continue
		}

		// Rotate point into kernel's local coordinate system
		cosTheta := math.Cos(imp.Orientation)
		sinTheta := math.Sin(imp.Orientation)

		// Kernel's local X-axis is aligned with imp.Orientation
		// Kernel's local Y-axis is perpendicular to it
		// Transform p into kernel's local frame (centered at impulse position)
		// If imp.Orientation is angle of kernel's X axis w.r.t global X:
		// localX = delta.X * cos(theta) + delta.Y * sin(theta)
		// localY = -delta.X * sin(theta) + delta.Y * cos(theta)
		// This rotates the vector delta by -imp.Orientation
		localX := delta.X*cosTheta + delta.Y*sinTheta
		localY := -delta.X*sinTheta + delta.Y*cosTheta

		// Gaussian component
		// exp(-PI * Bandwidth^2 * (X_local^2 + Y_local^2 / AspectRatio^2))
		// AspectRatio controls stretch along kernel's Y axis.
		// If AspectRatio = 1, circular Gaussian. If > 1, wider along local Y.
		gaussTermX := localX * localX
		gaussTermY := (localY * localY) / (imp.AspectRatio * imp.AspectRatio)
		gaussian := math.Exp(-math.Pi * imp.Bandwidth * imp.Bandwidth * (gaussTermX + gaussTermY))

		// Sinusoidal component (wave along kernel's local X-axis)
		// cos(2 * PI * Frequency * X_local + Phase)
		sinusoid := math.Cos(2*math.Pi*imp.Frequency*localX + imp.Phase)

		totalNoise += imp.Weight * gaussian * sinusoid
	}
	return totalNoise
}

// Impulse3D stores parameters for a single 3D Gabor impulse.
type Impulse3D struct {
	Position     Vec3
	Weight       float64
	Frequency    float64
	Orientation  Quaternion // Rotates world space to kernel's local space
	Bandwidth    float64
	Phase        float64
	AspectRatioY float64 // Gaussian scale along kernel's local Y relative to local X
	AspectRatioZ float64 // Gaussian scale along kernel's local Z relative to local X
	CutoffSq     float64 // Squared cutoff radius for this impulse's influence
}

// ImpulseConfig3D defines the range of parameters for generating 3D impulses.
type ImpulseConfig3D struct {
	NumImpulses int
	DomainMin   Vec3
	DomainMax   Vec3

	MinFrequency float64
	MaxFrequency float64

	MinBandwidth float64 // Must be > 0
	MaxBandwidth float64

	MinAspectRatioY float64 // Must be > 0
	MaxAspectRatioY float64
	MinAspectRatioZ float64 // Must be > 0
	MaxAspectRatioZ float64

	MinWeight float64
	MaxWeight float64

	RandomizeOrientation bool // If true, orientation is random
	RandomizePhase       bool // If true, phase is random (0 to 2*PI), otherwise 0

	KernelCutoffFactor float64 // cutoff = KernelCutoffFactor / MinBandwidth
}

// GaborNoise3D is a generator for 3D Gabor noise.
type GaborNoise3D struct {
	impulses []Impulse3D
	rng      *rand.Rand
}

// NewGaborNoise3D creates and initializes a 3D Gabor noise generator.
func NewGaborNoise3D(seed int64, cfg ImpulseConfig3D) *GaborNoise3D {
	if cfg.MinBandwidth <= 0 {
		panic("MinBandwidth must be greater than 0")
	}
	if cfg.MinAspectRatioY <= 0 || cfg.MinAspectRatioZ <= 0 {
		panic("MinAspectRatios must be greater than 0")
	}
	if cfg.KernelCutoffFactor <= 0 {
		cfg.KernelCutoffFactor = 3.0 // Default cutoff factor
	}

	rng := rand.New(rand.NewSource(seed))
	impulses := make([]Impulse3D, cfg.NumImpulses)
	baseCutoffRadius := cfg.KernelCutoffFactor / cfg.MinBandwidth

	for i := 0; i < cfg.NumImpulses; i++ {
		pos := Vec3{
			X: randFloat64(rng, cfg.DomainMin.X, cfg.DomainMax.X),
			Y: randFloat64(rng, cfg.DomainMin.Y, cfg.DomainMax.Y),
			Z: randFloat64(rng, cfg.DomainMin.Z, cfg.DomainMax.Z),
		}
		freq := randFloat64(rng, cfg.MinFrequency, cfg.MaxFrequency)
		bandwidth := randFloat64(rng, cfg.MinBandwidth, cfg.MaxBandwidth)
		aspectRatioY := randFloat64(rng, cfg.MinAspectRatioY, cfg.MaxAspectRatioY)
		aspectRatioZ := randFloat64(rng, cfg.MinAspectRatioZ, cfg.MaxAspectRatioZ)
		weight := randFloat64(rng, cfg.MinWeight, cfg.MaxWeight)

		orientation := NewQuaternionIdentity()
		if cfg.RandomizeOrientation {
			// Generate a random unit vector for axis
			axis := Vec3{
				X: rng.NormFloat64(), // Standard normal distribution
				Y: rng.NormFloat64(),
				Z: rng.NormFloat64(),
			}
			lenSq := axis.X*axis.X + axis.Y*axis.Y + axis.Z*axis.Z
			if lenSq > 1e-6 { // Avoid division by zero for zero vector
				invLen := 1.0 / math.Sqrt(lenSq)
				axis.X *= invLen
				axis.Y *= invLen
				axis.Z *= invLen
				angle := rng.Float64() * 2 * math.Pi
				orientation = FromAngleAxis(angle, axis)
			}
		}

		phase := 0.0
		if cfg.RandomizePhase {
			phase = rng.Float64() * 2 * math.Pi
		}

		impulseCutoff := cfg.KernelCutoffFactor / bandwidth
		if impulseCutoff > baseCutoffRadius {
			impulseCutoff = baseCutoffRadius
		}

		impulses[i] = Impulse3D{
			Position:     pos,
			Weight:       weight,
			Frequency:    freq,
			Orientation:  orientation,
			Bandwidth:    bandwidth,
			Phase:        phase,
			AspectRatioY: aspectRatioY,
			AspectRatioZ: aspectRatioZ,
			CutoffSq:     impulseCutoff * impulseCutoff,
		}
	}

	return &GaborNoise3D{
		impulses: impulses,
		rng:      rng,
	}
}

// GetNoise evaluates the 3D Gabor noise at point p.
func (gn *GaborNoise3D) GetNoise(p Vec3) float64 {
	var totalNoise float64

	for _, imp := range gn.impulses {
		delta := p.Sub(imp.Position)

		if delta.LengthSq() > imp.CutoffSq {
			continue
		}

		// Transform delta into kernel's local coordinate system
		// The impulse's orientation quaternion rotates from kernel local space to world space.
		// So, its inverse rotates from world space to kernel local space.
		invOrientation := imp.Orientation.Inverse()
		localCoords := invOrientation.Rotate(delta)

		// Gaussian component
		// Assumes sinusoid is along kernel's local X-axis.
		// AspectRatioY/Z control Gaussian shape in local YZ plane.
		gaussTermX := localCoords.X * localCoords.X
		gaussTermY := (localCoords.Y * localCoords.Y) / (imp.AspectRatioY * imp.AspectRatioY)
		gaussTermZ := (localCoords.Z * localCoords.Z) / (imp.AspectRatioZ * imp.AspectRatioZ)
		gaussian := math.Exp(-math.Pi * imp.Bandwidth * imp.Bandwidth * (gaussTermX + gaussTermY + gaussTermZ))

		// Sinusoidal component (wave along kernel's local X-axis)
		sinusoid := math.Cos(2*math.Pi*imp.Frequency*localCoords.X + imp.Phase)

		totalNoise += imp.Weight * gaussian * sinusoid
	}
	return totalNoise
}
