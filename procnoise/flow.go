package procnoise

import (
	"worldgen/icosphere"
)

// --- Flow Noise Implementation (Advection-based) ---

// AdvectionSource defines a function type that returns a displacement vector
// based on a position and time. This vector is used to advect coordinates.
// The returned vector should ideally be tangent to the sphere if operating on a sphere.
type AdvectionSource func(p icosphere.Vector3D, time float64) icosphere.Vector3D

// FlowNoiseGenerator produces animated scalar noise by advecting lookup coordinates
// into a base scalar noise field using a time-dependent advection field.
type FlowNoiseGenerator struct {
	BaseNoise         ScalarField3D   // The underlying static scalar noise (e.g., Simplex from FastNoiseLite or SphericalWaveletNoise)
	AdvectionFunc     AdvectionSource // Function that provides the advection vector field, dependent on time.
	AdvectionStrength float64         // Scales the displacement due to advection.
}

// NewFlowNoiseGenerator creates a new FlowNoiseGenerator.
// - baseNoise: The scalar field to sample after advection.
// - advectionFunc: A function (point, time) -> displacement_vector.
// - advectionStrength: Multiplier for the displacement.
func NewFlowNoiseGenerator(baseNoise ScalarField3D, advectionFunc AdvectionSource, advectionStrength float64) *FlowNoiseGenerator {
	if baseNoise == nil {
		panic("BaseNoise cannot be nil for FlowNoiseGenerator")
	}
	if advectionFunc == nil {
		panic("AdvectionFunc cannot be nil for FlowNoiseGenerator")
	}
	return &FlowNoiseGenerator{
		BaseNoise:         baseNoise,
		AdvectionFunc:     advectionFunc,
		AdvectionStrength: advectionStrength,
	}
}

// GetNoise evaluates the animated flow noise at point p and time t.
// It advects the point p using the AdvectionFunc before sampling the BaseNoise.
// The result is a scalar noise value.
func (fg *FlowNoiseGenerator) GetNoise(p icosphere.Vector3D, time float64) float32 {
	advectionVector := fg.AdvectionFunc(p, time)
	displacement := advectionVector.Scale(fg.AdvectionStrength) // Scale from icosphere.Vector3D

	// Advect point p. For spherical context, it's often best to re-normalize if the
	// base noise expects points on the unit sphere, or if advection might significantly
	// alter distance from origin. SphericalWaveletNoise normalizes internally.
	// FastNoiseLite takes Cartesian coords, so slight deviations might be fine.
	pAdvected := p.Add(displacement) // Add from icosphere.Vector3D

	return fg.BaseNoise.GetNoise(pAdvected)
}

// --- Example Advection Sources for Flow Noise ---

// StaticCurlAdvectionSource creates an AdvectionSource from a non-animated CurlNoiseGenerator3D.
// The 'time' parameter in the returned AdvectionSource will be ignored.
// The advection field will be tangent to the sphere.
func StaticCurlAdvectionSource(curlNoiseGen *CurlNoiseGenerator3D) AdvectionSource {
	if curlNoiseGen == nil {
		panic("CurlNoiseGenerator3D cannot be nil for StaticCurlAdvectionSource")
	}
	return func(p icosphere.Vector3D, time float64) icosphere.Vector3D {
		// 'time' is ignored here, resulting in a static advection field.
		return curlNoiseGen.GetTangentCurl(p)
	}
}

// AnimatedPotentialFunc defines a function that returns a scalar noise value
// based on a 3D position and a time parameter.
type AnimatedPotentialFunc func(p icosphere.Vector3D, time float64) float32

// TimeVaryingPotentialProvider wraps a FastNoiseLite.State to make it time-variant
// by incorporating 'time' into one of the spatial coordinates fed to FastNoiseLite.
// This can serve as an AnimatedPotentialFunc.
type TimeVaryingPotentialProvider struct {
	FNLState   *State[float32] // FastNoiseLite state (assumed to be float32)
	TimeScale  float32         // Factor by which time influences the chosen coordinate
	InputScale float32         // Optional scaling for x,y,z inputs if FNLState.Frequency is effectively 1
	TimeAxis   int             // 0 for X, 1 for Y, 2 for Z - axis to offset with time
}

// NewTimeVaryingPotentialProvider creates a new provider.
// timeAxis: 0 for X, 1 for Y, 2 for Z.
func NewTimeVaryingPotentialProvider(fnlState *State[float32], timeScale float32, inputScale float32, timeAxis int) *TimeVaryingPotentialProvider {
	if fnlState == nil {
		panic("FNLState cannot be nil for TimeVaryingPotentialProvider")
	}
	if inputScale <= 0 {
		inputScale = 1.0
	}
	if timeAxis < 0 || timeAxis > 2 {
		timeAxis = 2
	} // Default to Z-axis if invalid
	return &TimeVaryingPotentialProvider{
		FNLState:   fnlState,
		TimeScale:  timeScale,
		InputScale: inputScale,
		TimeAxis:   timeAxis,
	}
}

// GetNoiseWithTime provides a noise value that changes with time.
func (tvpp *TimeVaryingPotentialProvider) GetNoiseWithTime(p icosphere.Vector3D, time float64) float32 {
	x, y, z := float32(p.X)*tvpp.InputScale, float32(p.Y)*tvpp.InputScale, float32(p.Z)*tvpp.InputScale
	t := tvpp.TimeScale * float32(time)

	switch tvpp.TimeAxis {
	case 0:
		x += t
	case 1:
		y += t
	default:
		z += t // case 2 and default
	}
	return tvpp.FNLState.GetNoise3D(x, y, z)
}

// AnimatedScalarFieldAdapter adapts an AnimatedPotentialFunc to the ScalarField3D interface
// by fixing a specific time. This is useful for feeding time-varying potentials
// into a CurlNoiseGenerator3D which expects static ScalarField3D.
type AnimatedScalarFieldAdapter struct {
	NoiseFunc AnimatedPotentialFunc
	Time      float64 // The specific time slice this adapter represents
}

func (asfa *AnimatedScalarFieldAdapter) GetNoise(p icosphere.Vector3D) float32 {
	return asfa.NoiseFunc(p, asfa.Time)
}

// AnimatedCurlAdvectionSource creates an AdvectionSource where the curl noise field itself is animated.
// This is achieved by using three AnimatedPotentialFuncs for the curl noise generator's potentials.
// Note: This creates a new CurlNoiseGenerator3D on each call to the AdvectionSource,
// which might be inefficient for very frequent calls. Consider optimizing if performance is critical.
func AnimatedCurlAdvectionSource(
	potentialXFunc, potentialYFunc, potentialZFunc AnimatedPotentialFunc,
	epsilon float64,
) AdvectionSource {
	if potentialXFunc == nil || potentialYFunc == nil || potentialZFunc == nil {
		panic("AnimatedPotentialFuncs cannot be nil for AnimatedCurlAdvectionSource")
	}
	return func(p icosphere.Vector3D, time float64) icosphere.Vector3D {
		// Create temporary adapters that capture the current time.
		adapterX := &AnimatedScalarFieldAdapter{NoiseFunc: potentialXFunc, Time: time}
		adapterY := &AnimatedScalarFieldAdapter{NoiseFunc: potentialYFunc, Time: time}
		adapterZ := &AnimatedScalarFieldAdapter{NoiseFunc: potentialZFunc, Time: time}

		// A new CurlNoiseGenerator is created for each time step with the time-sliced potentials.
		tempCurlGen := NewCurlNoiseGenerator3D(adapterX, adapterY, adapterZ, epsilon)
		return tempCurlGen.GetTangentCurl(p)
	}
}
