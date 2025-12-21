package procnoise

import (
	"math"
	"math/rand"
	"worldgen/icosphere"
)

// --- Turbulence Noise Implementation ---

// TurbulenceType defines the type of turbulence algorithm.
type TurbulenceType int

const (
	// TurbulenceBasic creates standard turbulence using absolute values
	TurbulenceBasic TurbulenceType = iota
	// TurbulenceRidged creates ridged turbulence with sharp features
	TurbulenceRidged
	// TurbulenceBillow creates billowy cloud-like turbulence
	TurbulenceBillow
	// TurbulenceSwiss creates Swiss-style turbulence with derivative control
	TurbulenceSwiss
	// TurbulenceJordan creates Jordan turbulence with gain control
	TurbulenceJordan
)

// TurbulenceSettings holds configuration for turbulence noise generation.
type TurbulenceSettings struct {
	BaseNoise       ScalarField3D    // Base noise function
	Type            TurbulenceType   // Type of turbulence algorithm
	Octaves         int              // Number of octaves
	Lacunarity      float64          // Frequency multiplier between octaves
	Persistence     float64          // Amplitude multiplier between octaves
	
	// Turbulence-specific parameters
	Power         float64 // Power parameter for turbulence intensity
	Roughness     float64 // Roughness parameter for Swiss/Jordan turbulence
	Warp          float64 // Warp parameter for Swiss turbulence
	Damp          float64 // Damping parameter for Swiss turbulence
	DampScale     float64 // Damping scale for Swiss turbulence
	
	// Domain warping parameters
	WarpStrength  float64          // Strength of domain warping
	WarpNoise     ScalarField3D    // Noise function for domain warping
	
	// Scaling parameters
	Frequency     float64 // Base frequency
	Amplitude     float64 // Base amplitude
	Seed          int64   // Random seed for turbulence variations
}

// TurbulenceNoise3D generates turbulence noise in 3D space.
type TurbulenceNoise3D struct {
	Settings TurbulenceSettings
	RNG      *rand.Rand
	
	// Precomputed amplitude values for each octave
	amplitudes []float64
}

// NewTurbulenceNoise3D creates a new 3D turbulence noise generator.
func NewTurbulenceNoise3D(settings TurbulenceSettings) *TurbulenceNoise3D {
	if settings.BaseNoise == nil {
		panic("BaseNoise cannot be nil for TurbulenceNoise3D")
	}
	
	// Set default values
	if settings.Octaves <= 0 {
		settings.Octaves = 4
	}
	if settings.Lacunarity <= 0 {
		settings.Lacunarity = 2.0
	}
	if settings.Persistence <= 0 {
		settings.Persistence = 0.5
	}
	if settings.Power <= 0 {
		settings.Power = 1.0
	}
	if settings.Roughness <= 0 {
		settings.Roughness = 0.5
	}
	if settings.Frequency <= 0 {
		settings.Frequency = 0.01
	}
	if settings.Amplitude <= 0 {
		settings.Amplitude = 1.0
	}
	
	// Precompute amplitude values
	amplitudes := make([]float64, settings.Octaves)
	amp := 1.0
	for i := 0; i < settings.Octaves; i++ {
		amplitudes[i] = amp
		amp *= settings.Persistence
	}
	
	var rng *rand.Rand
	if settings.Seed != 0 {
		rng = rand.New(rand.NewSource(settings.Seed))
	}
	
	return &TurbulenceNoise3D{
		Settings:   settings,
		RNG:        rng,
		amplitudes: amplitudes,
	}
}

// GetNoise evaluates the turbulence noise at a given 3D position.
// Implements ScalarField3D interface.
func (tn *TurbulenceNoise3D) GetNoise(pos icosphere.Vector3D) float32 {
	// Apply domain warping if enabled
	warpedPos := pos
	if tn.Settings.WarpNoise != nil && tn.Settings.WarpStrength > 0 {
		warpedPos = tn.applyDomainWarp(pos)
	}
	
	switch tn.Settings.Type {
	case TurbulenceBasic:
		return tn.getBasicTurbulence(warpedPos)
	case TurbulenceRidged:
		return tn.getRidgedTurbulence(warpedPos)
	case TurbulenceBillow:
		return tn.getBillowTurbulence(warpedPos)
	case TurbulenceSwiss:
		return tn.getSwissTurbulence(warpedPos)
	case TurbulenceJordan:
		return tn.getJordanTurbulence(warpedPos)
	default:
		return tn.getBasicTurbulence(warpedPos)
	}
}

// applyDomainWarp applies domain warping to the input position.
func (tn *TurbulenceNoise3D) applyDomainWarp(pos icosphere.Vector3D) icosphere.Vector3D {
	if tn.Settings.WarpNoise == nil {
		return pos
	}
	
	// Generate warp offsets
	warpX := tn.Settings.WarpNoise.GetNoise(pos)
	warpY := tn.Settings.WarpNoise.GetNoise(pos.Add(icosphere.Vector3D{X: 73.12, Y: 0, Z: 0}))
	warpZ := tn.Settings.WarpNoise.GetNoise(pos.Add(icosphere.Vector3D{X: 0, Y: 131.7, Z: 0}))
	
	return pos.Add(icosphere.Vector3D{
		X: float64(warpX) * tn.Settings.WarpStrength,
		Y: float64(warpY) * tn.Settings.WarpStrength,
		Z: float64(warpZ) * tn.Settings.WarpStrength,
	})
}

// getBasicTurbulence generates basic turbulence using absolute values.
func (tn *TurbulenceNoise3D) getBasicTurbulence(pos icosphere.Vector3D) float32 {
	var value float64
	p := pos.Scale(tn.Settings.Frequency)
	
	for i := 0; i < tn.Settings.Octaves; i++ {
		noise := tn.Settings.BaseNoise.GetNoise(p)
		value += math.Abs(float64(noise)) * tn.amplitudes[i]
		p = p.Scale(tn.Settings.Lacunarity)
	}
	
	// Apply power curve
	if tn.Settings.Power != 1.0 {
		value = math.Pow(value, tn.Settings.Power)
	}
	
	return float32(value * tn.Settings.Amplitude)
}

// getRidgedTurbulence generates ridged turbulence with sharp features.
func (tn *TurbulenceNoise3D) getRidgedTurbulence(pos icosphere.Vector3D) float32 {
	var value float64
	var weight float64 = 1.0
	p := pos.Scale(tn.Settings.Frequency)
	
	for i := 0; i < tn.Settings.Octaves; i++ {
		noiseVal := tn.Settings.BaseNoise.GetNoise(p)
		
		// Create ridges by inverting absolute value
		noise := 1.0 - math.Abs(float64(noiseVal))
		
		// Square the signal to sharpen ridges
		noise = noise * noise
		
		// Weight successive octaves by previous values
		noise *= weight
		weight = noise
		
		// Clamp weight
		if weight > 1.0 {
			weight = 1.0
		}
		if weight < 0.0 {
			weight = 0.0
		}
		
		value += noise * tn.amplitudes[i]
		p = p.Scale(tn.Settings.Lacunarity)
	}
	
	return float32(value * tn.Settings.Amplitude)
}

// getBillowTurbulence generates billowy cloud-like turbulence.
func (tn *TurbulenceNoise3D) getBillowTurbulence(pos icosphere.Vector3D) float32 {
	var value float64
	p := pos.Scale(tn.Settings.Frequency)
	
	for i := 0; i < tn.Settings.Octaves; i++ {
		noiseVal := tn.Settings.BaseNoise.GetNoise(p)
		
		// Create billowy effect by using absolute value and scaling
		noise := 2.0*math.Abs(float64(noiseVal)) - 1.0
		
		value += noise * tn.amplitudes[i]
		p = p.Scale(tn.Settings.Lacunarity)
	}
	
	return float32(value * tn.Settings.Amplitude)
}

// getSwissTurbulence generates Swiss-style turbulence with derivative control.
func (tn *TurbulenceNoise3D) getSwissTurbulence(pos icosphere.Vector3D) float32 {
	var value float64
	var derivative float64 = 1.0
	p := pos.Scale(tn.Settings.Frequency)
	
	for i := 0; i < tn.Settings.Octaves; i++ {
		noise := tn.Settings.BaseNoise.GetNoise(p)
		noiseVal := float64(noise)
		
		// Calculate derivative approximation
		epsilon := 0.001
		noiseX := tn.Settings.BaseNoise.GetNoise(p.Add(icosphere.Vector3D{X: epsilon, Y: 0, Z: 0}))
		noiseY := tn.Settings.BaseNoise.GetNoise(p.Add(icosphere.Vector3D{X: 0, Y: epsilon, Z: 0}))
		noiseZ := tn.Settings.BaseNoise.GetNoise(p.Add(icosphere.Vector3D{X: 0, Y: 0, Z: epsilon}))
		
		gradMag := math.Sqrt(
			math.Pow((float64(noiseX)-noiseVal)/epsilon, 2) +
			math.Pow((float64(noiseY)-noiseVal)/epsilon, 2) +
			math.Pow((float64(noiseZ)-noiseVal)/epsilon, 2))
		
		// Apply Swiss turbulence formula
		noiseVal = math.Abs(noiseVal)
		noiseVal *= derivative
		
		// Warp the coordinate space
		warpFactor := tn.Settings.Warp * noiseVal
		p = p.Add(icosphere.Vector3D{
			X: warpFactor * gradMag,
			Y: warpFactor * gradMag * 0.7,
			Z: warpFactor * gradMag * 0.3,
		})
		
		value += noiseVal * tn.amplitudes[i]
		
		// Update derivative with damping
		derivative *= 1.0 - tn.Settings.Damp*noiseVal
		if derivative < 0.0 {
			derivative = 0.0
		}
		
		p = p.Scale(tn.Settings.Lacunarity)
	}
	
	return float32(value * tn.Settings.Amplitude)
}

// getJordanTurbulence generates Jordan turbulence with gain control.
func (tn *TurbulenceNoise3D) getJordanTurbulence(pos icosphere.Vector3D) float32 {
	var value float64
	var gain float64 = 1.0
	p := pos.Scale(tn.Settings.Frequency)
	
	for i := 0; i < tn.Settings.Octaves; i++ {
		noise := tn.Settings.BaseNoise.GetNoise(p)
		noiseVal := float64(noise)
		
		// Apply Jordan turbulence formula
		noiseVal = math.Abs(noiseVal)
		noiseVal = noiseVal * noiseVal * gain
		
		value += noiseVal * tn.amplitudes[i]
		
		// Update gain
		gain *= tn.Settings.Roughness
		
		p = p.Scale(tn.Settings.Lacunarity)
	}
	
	return float32(value * tn.Settings.Amplitude)
}

// --- Turbulence Utilities ---

// EstimateTurbulenceBounds estimates the output bounds for turbulence noise.
func EstimateTurbulenceBounds(settings TurbulenceSettings) (float64, float64) {
	var minVal, maxVal float64
	
	switch settings.Type {
	case TurbulenceBasic:
		maxVal = 0.0
		for i := 0; i < settings.Octaves; i++ {
			maxVal += math.Pow(settings.Persistence, float64(i))
		}
		minVal = 0.0 // Basic turbulence is always positive
		
	case TurbulenceRidged:
		maxVal = 0.0
		for i := 0; i < settings.Octaves; i++ {
			maxVal += math.Pow(settings.Persistence, float64(i))
		}
		minVal = 0.0 // Ridged turbulence is always positive
		
	case TurbulenceBillow:
		maxVal = 0.0
		minVal = 0.0
		for i := 0; i < settings.Octaves; i++ {
			amp := math.Pow(settings.Persistence, float64(i))
			maxVal += amp
			minVal -= amp
		}
		
	default:
		// Default rough estimate
		maxVal = 0.0
		for i := 0; i < settings.Octaves; i++ {
			maxVal += math.Pow(settings.Persistence, float64(i))
		}
		minVal = 0.0
	}
	
	return minVal * settings.Amplitude, maxVal * settings.Amplitude
}

// --- Turbulence Noise Builder ---

// TurbulenceNoiseBuilder provides a fluent interface for building turbulence noise.
type TurbulenceNoiseBuilder struct {
	settings TurbulenceSettings
}

// NewTurbulenceNoiseBuilder creates a new builder for turbulence noise.
func NewTurbulenceNoiseBuilder(baseNoise ScalarField3D) *TurbulenceNoiseBuilder {
	return &TurbulenceNoiseBuilder{
		settings: TurbulenceSettings{
			BaseNoise:    baseNoise,
			Type:         TurbulenceBasic,
			Octaves:      4,
			Lacunarity:   2.0,
			Persistence:  0.5,
			Power:        1.0,
			Roughness:    0.5,
			Warp:         0.15,
			Damp:         0.8,
			DampScale:    1.0,
			WarpStrength: 0.0,
			Frequency:    0.01,
			Amplitude:    1.0,
		},
	}
}

// SetType sets the turbulence type.
func (b *TurbulenceNoiseBuilder) SetType(tType TurbulenceType) *TurbulenceNoiseBuilder {
	b.settings.Type = tType
	return b
}

// SetOctaves sets the number of octaves.
func (b *TurbulenceNoiseBuilder) SetOctaves(octaves int) *TurbulenceNoiseBuilder {
	b.settings.Octaves = octaves
	return b
}

// SetLacunarity sets the lacunarity value.
func (b *TurbulenceNoiseBuilder) SetLacunarity(lacunarity float64) *TurbulenceNoiseBuilder {
	b.settings.Lacunarity = lacunarity
	return b
}

// SetPersistence sets the persistence value.
func (b *TurbulenceNoiseBuilder) SetPersistence(persistence float64) *TurbulenceNoiseBuilder {
	b.settings.Persistence = persistence
	return b
}

// SetPower sets the power parameter.
func (b *TurbulenceNoiseBuilder) SetPower(power float64) *TurbulenceNoiseBuilder {
	b.settings.Power = power
	return b
}

// SetRoughness sets the roughness parameter.
func (b *TurbulenceNoiseBuilder) SetRoughness(roughness float64) *TurbulenceNoiseBuilder {
	b.settings.Roughness = roughness
	return b
}

// SetWarp sets the warp parameter for Swiss turbulence.
func (b *TurbulenceNoiseBuilder) SetWarp(warp float64) *TurbulenceNoiseBuilder {
	b.settings.Warp = warp
	return b
}

// SetDamp sets the damping parameter for Swiss turbulence.
func (b *TurbulenceNoiseBuilder) SetDamp(damp float64) *TurbulenceNoiseBuilder {
	b.settings.Damp = damp
	return b
}

// SetDomainWarp sets domain warping parameters.
func (b *TurbulenceNoiseBuilder) SetDomainWarp(warpNoise ScalarField3D, strength float64) *TurbulenceNoiseBuilder {
	b.settings.WarpNoise = warpNoise
	b.settings.WarpStrength = strength
	return b
}

// SetFrequency sets the base frequency.
func (b *TurbulenceNoiseBuilder) SetFrequency(frequency float64) *TurbulenceNoiseBuilder {
	b.settings.Frequency = frequency
	return b
}

// SetAmplitude sets the base amplitude.
func (b *TurbulenceNoiseBuilder) SetAmplitude(amplitude float64) *TurbulenceNoiseBuilder {
	b.settings.Amplitude = amplitude
	return b
}

// SetSeed sets the random seed.
func (b *TurbulenceNoiseBuilder) SetSeed(seed int64) *TurbulenceNoiseBuilder {
	b.settings.Seed = seed
	return b
}

// Build creates the turbulence noise generator.
func (b *TurbulenceNoiseBuilder) Build() *TurbulenceNoise3D {
	return NewTurbulenceNoise3D(b.settings)
}

// --- Specialized Turbulence Generators ---

// VortexTurbulence generates vortex-like turbulence patterns.
type VortexTurbulence struct {
	BaseNoise     ScalarField3D
	VortexCenters []icosphere.Vector3D
	VortexRadii   []float64
	VortexStrength []float64
	Frequency     float64
	Amplitude     float64
}

// NewVortexTurbulence creates a new vortex turbulence generator.
func NewVortexTurbulence(baseNoise ScalarField3D, centers []icosphere.Vector3D, radii []float64, strengths []float64) *VortexTurbulence {
	if len(centers) != len(radii) || len(centers) != len(strengths) {
		panic("VortexTurbulence: centers, radii, and strengths must have the same length")
	}
	
	return &VortexTurbulence{
		BaseNoise:      baseNoise,
		VortexCenters:  centers,
		VortexRadii:    radii,
		VortexStrength: strengths,
		Frequency:      0.01,
		Amplitude:      1.0,
	}
}

// GetNoise evaluates the vortex turbulence at a given 3D position.
func (vt *VortexTurbulence) GetNoise(pos icosphere.Vector3D) float32 {
	baseValue := vt.BaseNoise.GetNoise(pos.Scale(vt.Frequency))
	
	// Apply vortex influences
	totalVortexInfluence := 0.0
	for i, center := range vt.VortexCenters {
		distance := pos.Subtract(center).Length()
		if distance < vt.VortexRadii[i] {
			// Calculate vortex influence
			normalizedDist := distance / vt.VortexRadii[i]
			influence := (1.0 - normalizedDist) * vt.VortexStrength[i]
			
			// Add swirling effect
			angle := math.Atan2(pos.Y-center.Y, pos.X-center.X)
			swirl := math.Sin(angle * 3.0 + distance * 10.0)
			influence *= swirl
			
			totalVortexInfluence += influence
		}
	}
	
	return float32((float64(baseValue) + totalVortexInfluence) * vt.Amplitude)
}