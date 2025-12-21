package procnoise

import (
	"math"
	"worldgen/icosphere"
)

// --- Multifractal Noise Implementation ---

// MultifractalType defines the type of multifractal noise algorithm.
type MultifractalType int

const (
	// MultifractalHeteroTerrain creates terrain with varying roughness
	MultifractalHeteroTerrain MultifractalType = iota
	// MultifractalHybridTerrain combines additive and multiplicative fractals
	MultifractalHybridTerrain
	// MultifractalRidgedTerrain creates sharp ridges with varying characteristics
	MultifractalRidgedTerrain
	// MultifractalVaryingLacunarity changes frequency scaling spatially
	MultifractalVaryingLacunarity
)

// MultifractalSettings holds configuration for multifractal noise generation.
type MultifractalSettings struct {
	BaseNoise   ScalarField3D    // Base noise function (e.g., FastNoiseLite, SphericalWavelet)
	Type        MultifractalType // Type of multifractal algorithm
	Octaves     int              // Number of octaves
	Lacunarity  float64          // Frequency multiplier between octaves
	Persistence float64          // Amplitude multiplier between octaves
	
	// Multifractal-specific parameters
	H         float64 // Hurst exponent (0.0 to 1.0) - controls roughness variation
	Offset    float64 // Offset parameter for terrain types
	Threshold float64 // Threshold parameter for hybrid terrain
	
	// Varying parameters (for VaryingLacunarity type)
	LacunarityNoise ScalarField3D // Noise function controlling lacunarity variation
	MinLacunarity   float64       // Minimum lacunarity value
	MaxLacunarity   float64       // Maximum lacunarity value
	
	// Scaling parameters
	Frequency float64 // Base frequency
	Amplitude float64 // Base amplitude
}

// MultifractalNoise3D generates multifractal noise in 3D space.
type MultifractalNoise3D struct {
	Settings MultifractalSettings
	
	// Precomputed values for efficiency
	exponentArray []float64 // Precomputed exponents for each octave
}

// NewMultifractalNoise3D creates a new 3D multifractal noise generator.
func NewMultifractalNoise3D(settings MultifractalSettings) *MultifractalNoise3D {
	if settings.BaseNoise == nil {
		panic("BaseNoise cannot be nil for MultifractalNoise3D")
	}
	if settings.Octaves <= 0 {
		settings.Octaves = 4
	}
	if settings.Lacunarity <= 0 {
		settings.Lacunarity = 2.0
	}
	if settings.Persistence <= 0 {
		settings.Persistence = 0.5
	}
	if settings.H <= 0 {
		settings.H = 0.5
	}
	if settings.Frequency <= 0 {
		settings.Frequency = 0.01
	}
	if settings.Amplitude <= 0 {
		settings.Amplitude = 1.0
	}
	
	// Precompute exponents for each octave
	exponentArray := make([]float64, settings.Octaves)
	frequency := 1.0
	
	for i := 0; i < settings.Octaves; i++ {
		// Compute the exponent for this octave
		exponentArray[i] = math.Pow(frequency, -settings.H)
		frequency *= settings.Lacunarity
	}
	
	return &MultifractalNoise3D{
		Settings:      settings,
		exponentArray: exponentArray,
	}
}

// GetNoise evaluates the multifractal noise at a given 3D position.
// Implements ScalarField3D interface.
func (mfn *MultifractalNoise3D) GetNoise(pos icosphere.Vector3D) float32 {
	switch mfn.Settings.Type {
	case MultifractalHeteroTerrain:
		return mfn.getHeteroTerrain(pos)
	case MultifractalHybridTerrain:
		return mfn.getHybridTerrain(pos)
	case MultifractalRidgedTerrain:
		return mfn.getRidgedTerrain(pos)
	case MultifractalVaryingLacunarity:
		return mfn.getVaryingLacunarity(pos)
	default:
		return mfn.getHeteroTerrain(pos)
	}
}

// getHeteroTerrain generates heterogeneous terrain with varying roughness.
func (mfn *MultifractalNoise3D) getHeteroTerrain(pos icosphere.Vector3D) float32 {
	var value float64
	var remainder float64
	
	// Scale position by base frequency
	p := pos.Scale(mfn.Settings.Frequency)
	
	// Get first octave
	noiseVal := mfn.Settings.BaseNoise.GetNoise(p)
	value = (float64(noiseVal) + mfn.Settings.Offset) * mfn.exponentArray[0]
	remainder = value
	
	// Add remaining octaves
	for i := 1; i < mfn.Settings.Octaves; i++ {
		// Increase frequency
		p = p.Scale(mfn.Settings.Lacunarity)
		
		// Get noise value for this octave
		noise := mfn.Settings.BaseNoise.GetNoise(p)
		noiseFloat := float64(noise)
		
		// Scale by the exponent and the previous value (heterogeneous)
		increment := (noiseFloat + mfn.Settings.Offset) * mfn.exponentArray[i] * remainder
		value += increment
		remainder *= noiseFloat + mfn.Settings.Offset
	}
	
	return float32(value * mfn.Settings.Amplitude)
}

// getHybridTerrain generates hybrid terrain combining additive and multiplicative fractals.
func (mfn *MultifractalNoise3D) getHybridTerrain(pos icosphere.Vector3D) float32 {
	var value float64
	var weight float64
	
	// Scale position by base frequency
	p := pos.Scale(mfn.Settings.Frequency)
	
	// Get first octave
	noiseVal := mfn.Settings.BaseNoise.GetNoise(p)
	value = (float64(noiseVal) + mfn.Settings.Offset) * mfn.exponentArray[0]
	weight = value
	
	// Add remaining octaves
	for i := 1; i < mfn.Settings.Octaves; i++ {
		// Increase frequency
		p = p.Scale(mfn.Settings.Lacunarity)
		
		// Get noise value for this octave
		noise := mfn.Settings.BaseNoise.GetNoise(p)
		noiseFloat := float64(noise)
		
		// Weight determines how much this octave contributes
		if weight > 1.0 {
			weight = 1.0
		}
		
		// Scale by the exponent and weight
		increment := (noiseFloat + mfn.Settings.Offset) * mfn.exponentArray[i] * weight
		value += increment
		
		// Update weight for next octave
		weight *= noiseFloat + mfn.Settings.Offset
	}
	
	// Apply threshold to create hybrid characteristics
	if value > mfn.Settings.Threshold {
		return float32((value - mfn.Settings.Threshold) * mfn.Settings.Amplitude)
	}
	return float32(value * mfn.Settings.Amplitude * 0.5)
}

// getRidgedTerrain generates ridged terrain with varying characteristics.
func (mfn *MultifractalNoise3D) getRidgedTerrain(pos icosphere.Vector3D) float32 {
	var value float64
	var weight float64
	
	// Scale position by base frequency
	p := pos.Scale(mfn.Settings.Frequency)
	
	// Get first octave - create ridges by using absolute value
	noiseVal := mfn.Settings.BaseNoise.GetNoise(p)
	noise := math.Abs(float64(noiseVal))
	noise = mfn.Settings.Offset - noise // Invert for ridges
	noise = noise * noise // Square for sharper ridges
	value = noise * mfn.exponentArray[0]
	weight = value
	
	// Add remaining octaves
	for i := 1; i < mfn.Settings.Octaves; i++ {
		// Increase frequency
		p = p.Scale(mfn.Settings.Lacunarity)
		
		// Get noise value for this octave
		noiseVal := mfn.Settings.BaseNoise.GetNoise(p)
		noise := math.Abs(float64(noiseVal))
		noise = mfn.Settings.Offset - noise
		noise = noise * noise
		
		// Weight determines how much this octave contributes
		if weight > 1.0 {
			weight = 1.0
		}
		
		// Scale by the exponent and weight
		increment := noise * mfn.exponentArray[i] * weight
		value += increment
		
		// Update weight for next octave
		weight *= noise
	}
	
	return float32(value * mfn.Settings.Amplitude)
}

// getVaryingLacunarity generates noise with spatially varying lacunarity.
func (mfn *MultifractalNoise3D) getVaryingLacunarity(pos icosphere.Vector3D) float32 {
	if mfn.Settings.LacunarityNoise == nil {
		// Fall back to regular hetero terrain if no lacunarity noise
		return mfn.getHeteroTerrain(pos)
	}
	
	var value float64
	var remainder float64
	
	// Get varying lacunarity value
	lacunarityNoise := mfn.Settings.LacunarityNoise.GetNoise(pos)
	// Map from [-1,1] to [MinLacunarity, MaxLacunarity]
	lacunarity := mfn.Settings.MinLacunarity + 
		(float64(lacunarityNoise)+1.0)*0.5*(mfn.Settings.MaxLacunarity-mfn.Settings.MinLacunarity)
	
	// Scale position by base frequency
	p := pos.Scale(mfn.Settings.Frequency)
	
	// Get first octave
	noiseVal := mfn.Settings.BaseNoise.GetNoise(p)
	value = (float64(noiseVal) + mfn.Settings.Offset) * mfn.exponentArray[0]
	remainder = value
	
	// Add remaining octaves with varying lacunarity
	for i := 1; i < mfn.Settings.Octaves; i++ {
		// Increase frequency by varying lacunarity
		p = p.Scale(lacunarity)
		
		// Get noise value for this octave
		noise := mfn.Settings.BaseNoise.GetNoise(p)
		noiseFloat := float64(noise)
		
		// Scale by the exponent and the previous value
		increment := (noiseFloat + mfn.Settings.Offset) * mfn.exponentArray[i] * remainder
		value += increment
		remainder *= noiseFloat + mfn.Settings.Offset
	}
	
	return float32(value * mfn.Settings.Amplitude)
}

// --- Multifractal Utilities ---

// ComputeSpectralWeights computes spectral weights for multifractal noise.
func ComputeSpectralWeights(h float64, lacunarity float64, octaves int) []float64 {
	weights := make([]float64, octaves)
	frequency := 1.0
	
	for i := 0; i < octaves; i++ {
		weights[i] = math.Pow(frequency, -h)
		frequency *= lacunarity
	}
	
	return weights
}

// EstimateMultifractalBounds estimates the output bounds for multifractal noise.
func EstimateMultifractalBounds(settings MultifractalSettings) (float64, float64) {
	var minVal, maxVal float64
	
	switch settings.Type {
	case MultifractalHeteroTerrain:
		// Estimate bounds for hetero terrain
		maxVal = 0.0
		for i := 0; i < settings.Octaves; i++ {
			maxVal += (1.0 + settings.Offset) * math.Pow(settings.Lacunarity, -settings.H*float64(i))
		}
		minVal = -maxVal * 0.5 // Rough estimate
		
	case MultifractalRidgedTerrain:
		// Ridged terrain is typically positive
		minVal = 0.0
		maxVal = 0.0
		for i := 0; i < settings.Octaves; i++ {
			maxVal += settings.Offset * settings.Offset * math.Pow(settings.Lacunarity, -settings.H*float64(i))
		}
		
	default:
		// Default rough estimate
		maxVal = 0.0
		for i := 0; i < settings.Octaves; i++ {
			maxVal += math.Pow(settings.Persistence, float64(i))
		}
		minVal = -maxVal
	}
	
	return minVal * settings.Amplitude, maxVal * settings.Amplitude
}

// --- Multifractal Noise Builder ---

// MultifractalNoiseBuilder provides a fluent interface for building multifractal noise.
type MultifractalNoiseBuilder struct {
	settings MultifractalSettings
}

// NewMultifractalNoiseBuilder creates a new builder for multifractal noise.
func NewMultifractalNoiseBuilder(baseNoise ScalarField3D) *MultifractalNoiseBuilder {
	return &MultifractalNoiseBuilder{
		settings: MultifractalSettings{
			BaseNoise:   baseNoise,
			Type:        MultifractalHeteroTerrain,
			Octaves:     4,
			Lacunarity:  2.0,
			Persistence: 0.5,
			H:           0.5,
			Offset:      0.7,
			Threshold:   0.5,
			Frequency:   0.01,
			Amplitude:   1.0,
		},
	}
}

// SetType sets the multifractal type.
func (b *MultifractalNoiseBuilder) SetType(mfType MultifractalType) *MultifractalNoiseBuilder {
	b.settings.Type = mfType
	return b
}

// SetOctaves sets the number of octaves.
func (b *MultifractalNoiseBuilder) SetOctaves(octaves int) *MultifractalNoiseBuilder {
	b.settings.Octaves = octaves
	return b
}

// SetLacunarity sets the lacunarity value.
func (b *MultifractalNoiseBuilder) SetLacunarity(lacunarity float64) *MultifractalNoiseBuilder {
	b.settings.Lacunarity = lacunarity
	return b
}

// SetPersistence sets the persistence value.
func (b *MultifractalNoiseBuilder) SetPersistence(persistence float64) *MultifractalNoiseBuilder {
	b.settings.Persistence = persistence
	return b
}

// SetH sets the Hurst exponent.
func (b *MultifractalNoiseBuilder) SetH(h float64) *MultifractalNoiseBuilder {
	b.settings.H = h
	return b
}

// SetOffset sets the offset parameter.
func (b *MultifractalNoiseBuilder) SetOffset(offset float64) *MultifractalNoiseBuilder {
	b.settings.Offset = offset
	return b
}

// SetThreshold sets the threshold parameter for hybrid terrain.
func (b *MultifractalNoiseBuilder) SetThreshold(threshold float64) *MultifractalNoiseBuilder {
	b.settings.Threshold = threshold
	return b
}

// SetFrequency sets the base frequency.
func (b *MultifractalNoiseBuilder) SetFrequency(frequency float64) *MultifractalNoiseBuilder {
	b.settings.Frequency = frequency
	return b
}

// SetAmplitude sets the base amplitude.
func (b *MultifractalNoiseBuilder) SetAmplitude(amplitude float64) *MultifractalNoiseBuilder {
	b.settings.Amplitude = amplitude
	return b
}

// SetVaryingLacunarity sets parameters for varying lacunarity.
func (b *MultifractalNoiseBuilder) SetVaryingLacunarity(lacunarityNoise ScalarField3D, minLac, maxLac float64) *MultifractalNoiseBuilder {
	b.settings.LacunarityNoise = lacunarityNoise
	b.settings.MinLacunarity = minLac
	b.settings.MaxLacunarity = maxLac
	return b
}

// Build creates the multifractal noise generator.
func (b *MultifractalNoiseBuilder) Build() *MultifractalNoise3D {
	return NewMultifractalNoise3D(b.settings)
}

// --- Spherical Multifractal Noise ---

// SphericalMultifractalNoise is a specialized version for spherical surfaces.
type SphericalMultifractalNoise struct {
	*MultifractalNoise3D
	NormalizeInput bool // Whether to normalize input positions to unit sphere
}

// NewSphericalMultifractalNoise creates a new spherical multifractal noise generator.
func NewSphericalMultifractalNoise(settings MultifractalSettings, normalizeInput bool) *SphericalMultifractalNoise {
	return &SphericalMultifractalNoise{
		MultifractalNoise3D: NewMultifractalNoise3D(settings),
		NormalizeInput:      normalizeInput,
	}
}

// GetNoise evaluates the spherical multifractal noise at a given 3D position.
func (smfn *SphericalMultifractalNoise) GetNoise(pos icosphere.Vector3D) float32 {
	if smfn.NormalizeInput {
		pos = pos.Normalize()
	}
	return smfn.MultifractalNoise3D.GetNoise(pos)
}