package procnoise

import (
	"math"
	"math/cmplx"
	"math/rand"
	"worldgen/icosphere"
)

// --- Spectral Synthesis Implementation ---

// Complex64 represents a complex number for spectral calculations.
type Complex64 complex64

// SpectralType defines the type of spectral synthesis algorithm.
type SpectralType int

const (
	// SpectralOceanWaves generates ocean wave patterns using Pierson-Moskowitz spectrum
	SpectralOceanWaves SpectralType = iota
	// SpectralAtmospheric generates atmospheric patterns using Kolmogorov spectrum
	SpectralAtmospheric
	// SpectralTerrain generates terrain using custom power-law spectra
	SpectralTerrain
	// SpectralTurbulent generates turbulent flow using energy cascade
	SpectralTurbulent
	// SpectralPeriodic generates periodic patterns with harmonic content
	SpectralPeriodic
)

// SpectralSettings holds configuration for spectral synthesis.
type SpectralSettings struct {
	Type        SpectralType // Type of spectral synthesis
	Resolution  int          // Resolution of the spectral grid (power of 2)
	DomainSize  float64      // Physical size of the domain
	
	// Spectrum parameters
	PowerLawExponent float64   // Exponent for power-law spectra (e.g., -5/3 for Kolmogorov)
	EnergyScale      float64   // Overall energy scaling
	MinWavelength    float64   // Minimum wavelength (high frequency cutoff)
	MaxWavelength    float64   // Maximum wavelength (low frequency cutoff)
	
	// Ocean wave specific parameters
	WindSpeed        float64   // Wind speed for ocean waves (m/s)
	WindDirection    float64   // Wind direction (radians)
	FetchLength      float64   // Fetch length for wave development
	PeakFrequency    float64   // Peak frequency for wave spectrum
	
	// Atmospheric parameters
	CorrelationLength float64  // Correlation length for atmospheric phenomena
	DissipationRate   float64  // Energy dissipation rate
	
	// Time evolution parameters
	TimeStep         float64   // Time step for animation
	GroupVelocity    float64   // Group velocity for wave propagation
	
	// Random parameters
	Seed             int64     // Random seed
	PhaseRandomness  float64   // Amount of phase randomness [0,1]
}

// SpectralSynthesis generates noise using spectral methods.
type SpectralSynthesis struct {
	Settings     SpectralSettings
	FreqGrid     [][]float64    // Frequency grid
	Spectrum     [][]float64    // Power spectrum
	Phases       [][]float64    // Random phases
	RealField    [][]float64    // Real space field
	ComplexField [][]complex128 // Complex frequency domain field
	RNG          *rand.Rand
	
	// Precomputed values
	kx, ky       []float64      // Wave number arrays
	initialized  bool
}

// NewSpectralSynthesis creates a new spectral synthesis generator.
func NewSpectralSynthesis(settings SpectralSettings) *SpectralSynthesis {
	if settings.Resolution <= 0 || (settings.Resolution&(settings.Resolution-1)) != 0 {
		panic("Resolution must be a positive power of 2")
	}
	if settings.DomainSize <= 0 {
		settings.DomainSize = 1.0
	}
	if settings.EnergyScale <= 0 {
		settings.EnergyScale = 1.0
	}
	
	n := settings.Resolution
	ss := &SpectralSynthesis{
		Settings:     settings,
		FreqGrid:     make([][]float64, n),
		Spectrum:     make([][]float64, n),
		Phases:       make([][]float64, n),
		RealField:    make([][]float64, n),
		ComplexField: make([][]complex128, n),
		RNG:          rand.New(rand.NewSource(settings.Seed)),
		kx:           make([]float64, n),
		ky:           make([]float64, n),
	}
	
	// Initialize arrays
	for i := 0; i < n; i++ {
		ss.FreqGrid[i] = make([]float64, n)
		ss.Spectrum[i] = make([]float64, n)
		ss.Phases[i] = make([]float64, n)
		ss.RealField[i] = make([]float64, n)
		ss.ComplexField[i] = make([]complex128, n)
	}
	
	ss.initializeWaveNumbers()
	ss.generateSpectrum()
	ss.generateRandomPhases()
	
	return ss
}

// initializeWaveNumbers sets up the wave number arrays.
func (ss *SpectralSynthesis) initializeWaveNumbers() {
	n := ss.Settings.Resolution
	L := ss.Settings.DomainSize
	
	// Fundamental wave number
	k0 := 2.0 * math.Pi / L
	
	for i := 0; i < n; i++ {
		if i <= n/2 {
			ss.kx[i] = float64(i) * k0
			ss.ky[i] = float64(i) * k0
		} else {
			ss.kx[i] = float64(i-n) * k0
			ss.ky[i] = float64(i-n) * k0
		}
	}
	
	// Compute frequency magnitude grid
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			ss.FreqGrid[i][j] = math.Sqrt(ss.kx[i]*ss.kx[i] + ss.ky[j]*ss.ky[j])
		}
	}
}

// generateSpectrum creates the power spectrum based on the spectral type.
func (ss *SpectralSynthesis) generateSpectrum() {
	n := ss.Settings.Resolution
	
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			k := ss.FreqGrid[i][j]
			if k == 0 {
				ss.Spectrum[i][j] = 0
				continue
			}
			
			switch ss.Settings.Type {
			case SpectralOceanWaves:
				ss.Spectrum[i][j] = ss.piersonMoskowitzSpectrum(k)
			case SpectralAtmospheric:
				ss.Spectrum[i][j] = ss.kolmogorovSpectrum(k)
			case SpectralTerrain:
				ss.Spectrum[i][j] = ss.powerLawSpectrum(k)
			case SpectralTurbulent:
				ss.Spectrum[i][j] = ss.turbulentSpectrum(k)
			case SpectralPeriodic:
				ss.Spectrum[i][j] = ss.periodicSpectrum(k)
			default:
				ss.Spectrum[i][j] = ss.powerLawSpectrum(k)
			}
		}
	}
}

// piersonMoskowitzSpectrum generates ocean wave spectrum.
func (ss *SpectralSynthesis) piersonMoskowitzSpectrum(k float64) float64 {
	if k <= 0 {
		return 0
	}
	
	// Convert to frequency domain
	omega := math.Sqrt(9.81 * k) // Deep water dispersion relation
	
	// Pierson-Moskowitz parameters
	alpha := 0.0081 // Phillips constant
	g := 9.81       // Gravitational acceleration
	U := ss.Settings.WindSpeed
	if U <= 0 {
		U = 10.0 // Default wind speed
	}
	
	omegaP := 0.855 * g / U // Peak frequency
	
	// Pierson-Moskowitz spectrum
	spectrum := alpha * g * g / math.Pow(omega, 5) * 
		math.Exp(-1.25 * math.Pow(omegaP/omega, 4))
	
	return spectrum * ss.Settings.EnergyScale
}

// kolmogorovSpectrum generates atmospheric turbulence spectrum.
func (ss *SpectralSynthesis) kolmogorovSpectrum(k float64) float64 {
	if k <= 0 {
		return 0
	}
	
	// Kolmogorov -5/3 law
	exponent := ss.Settings.PowerLawExponent
	if exponent == 0 {
		exponent = -5.0/3.0
	}
	
	// Energy dissipation rate
	epsilon := ss.Settings.DissipationRate
	if epsilon <= 0 {
		epsilon = 0.1
	}
	
	// Kolmogorov constant
	C := 1.5 // Typical value
	
	spectrum := C * math.Pow(epsilon, 2.0/3.0) * math.Pow(k, exponent)
	
	// Apply cutoffs
	if ss.Settings.MinWavelength > 0 {
		kMax := 2.0 * math.Pi / ss.Settings.MinWavelength
		if k > kMax {
			spectrum *= math.Exp(-(k-kMax)/(kMax*0.1))
		}
	}
	
	if ss.Settings.MaxWavelength > 0 {
		kMin := 2.0 * math.Pi / ss.Settings.MaxWavelength
		if k < kMin {
			spectrum *= math.Exp(-(kMin-k)/(kMin*0.1))
		}
	}
	
	return spectrum * ss.Settings.EnergyScale
}

// powerLawSpectrum generates generic power-law spectrum for terrain.
func (ss *SpectralSynthesis) powerLawSpectrum(k float64) float64 {
	if k <= 0 {
		return 0
	}
	
	exponent := ss.Settings.PowerLawExponent
	if exponent == 0 {
		exponent = -2.0 // Default for terrain
	}
	
	spectrum := math.Pow(k, exponent)
	
	// Apply cutoffs
	if ss.Settings.MinWavelength > 0 {
		kMax := 2.0 * math.Pi / ss.Settings.MinWavelength
		if k > kMax {
			spectrum *= math.Exp(-math.Pow((k-kMax)/(kMax*0.2), 2))
		}
	}
	
	if ss.Settings.MaxWavelength > 0 {
		kMin := 2.0 * math.Pi / ss.Settings.MaxWavelength
		if k < kMin {
			spectrum *= math.Exp(-math.Pow((kMin-k)/(kMin*0.2), 2))
		}
	}
	
	return spectrum * ss.Settings.EnergyScale
}

// turbulentSpectrum generates energy cascade spectrum.
func (ss *SpectralSynthesis) turbulentSpectrum(k float64) float64 {
	if k <= 0 {
		return 0
	}
	
	// Inertial range with -5/3 slope
	inertialSpectrum := math.Pow(k, -5.0/3.0)
	
	// Dissipation range cutoff
	kd := 2.0 * math.Pi / ss.Settings.MinWavelength
	if kd <= 0 {
		kd = 100.0 // Default dissipation wavenumber
	}
	
	dissipationCutoff := math.Exp(-math.Pow(k/kd, 2))
	
	return inertialSpectrum * dissipationCutoff * ss.Settings.EnergyScale
}

// periodicSpectrum generates spectrum with harmonic content.
func (ss *SpectralSynthesis) periodicSpectrum(k float64) float64 {
	if k <= 0 {
		return 0
	}
	
	// Fundamental frequency
	k0 := 2.0 * math.Pi / ss.Settings.MaxWavelength
	if k0 <= 0 {
		k0 = 1.0
	}
	
	// Sum of harmonics
	spectrum := 0.0
	for n := 1; n <= 8; n++ { // First 8 harmonics
		kn := float64(n) * k0
		width := k0 * 0.1 // Narrow peaks
		contribution := math.Exp(-math.Pow((k-kn)/width, 2)) / float64(n*n)
		spectrum += contribution
	}
	
	return spectrum * ss.Settings.EnergyScale
}

// generateRandomPhases creates random phases for spectral components.
func (ss *SpectralSynthesis) generateRandomPhases() {
	n := ss.Settings.Resolution
	
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			ss.Phases[i][j] = ss.RNG.Float64() * 2.0 * math.Pi
		}
	}
}

// Generate creates the spectral field using inverse FFT.
func (ss *SpectralSynthesis) Generate() [][]float64 {
	n := ss.Settings.Resolution
	
	// Create complex field with proper amplitudes and phases
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			amplitude := math.Sqrt(ss.Spectrum[i][j])
			phase := ss.Phases[i][j]
			
			ss.ComplexField[i][j] = complex(
				amplitude*math.Cos(phase),
				amplitude*math.Sin(phase),
			)
		}
	}
	
	// Apply Hermitian symmetry for real output
	ss.enforceHermitianSymmetry()
	
	// Perform inverse FFT
	ss.inverseFFT2D()
	
	return ss.RealField
}

// enforceHermitianSymmetry ensures the field produces real output after IFFT.
func (ss *SpectralSynthesis) enforceHermitianSymmetry() {
	n := ss.Settings.Resolution
	
	// Ensure F(0,0) is real
	ss.ComplexField[0][0] = complex(real(ss.ComplexField[0][0]), 0)
	
	// Ensure F(n/2, 0) and F(0, n/2) are real if they exist
	if n%2 == 0 {
		ss.ComplexField[n/2][0] = complex(real(ss.ComplexField[n/2][0]), 0)
		ss.ComplexField[0][n/2] = complex(real(ss.ComplexField[0][n/2]), 0)
		ss.ComplexField[n/2][n/2] = complex(real(ss.ComplexField[n/2][n/2]), 0)
	}
	
	// Enforce F(-kx, -ky) = F*(kx, ky)
	for i := 1; i < n; i++ {
		for j := 1; j < n; j++ {
			ii := (n - i) % n
			jj := (n - j) % n
			ss.ComplexField[ii][jj] = cmplx.Conj(ss.ComplexField[i][j])
		}
	}
}

// Simple 2D inverse FFT implementation (for demonstration - production code should use FFTW or similar)
func (ss *SpectralSynthesis) inverseFFT2D() {
	n := ss.Settings.Resolution
	
	// This is a simplified implementation
	// In production, use a proper FFT library like gonum/fourier or cgo bindings to FFTW
	
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			sum := complex(0, 0)
			for ki := 0; ki < n; ki++ {
				for kj := 0; kj < n; kj++ {
					phase := 2.0 * math.Pi * (float64(i*ki+j*kj) / float64(n))
					sum += ss.ComplexField[ki][kj] * complex(math.Cos(phase), math.Sin(phase))
				}
			}
			ss.RealField[i][j] = real(sum) / float64(n*n)
		}
	}
}

// GetNoise2D evaluates the spectral field at a 2D position using interpolation.
func (ss *SpectralSynthesis) GetNoise2D(x, y float64) float64 {
	if !ss.initialized {
		ss.Generate()
		ss.initialized = true
	}
	
	n := ss.Settings.Resolution
	L := ss.Settings.DomainSize
	
	// Map world coordinates to grid coordinates
	fx := (x / L + 0.5) * float64(n)
	fy := (y / L + 0.5) * float64(n)
	
	// Wrap coordinates
	fx = fx - math.Floor(fx/float64(n))*float64(n)
	fy = fy - math.Floor(fy/float64(n))*float64(n)
	
	// Bilinear interpolation
	i0 := int(fx) % n
	j0 := int(fy) % n
	i1 := (i0 + 1) % n
	j1 := (j0 + 1) % n
	
	tx := fx - float64(i0)
	ty := fy - float64(j0)
	
	v00 := ss.RealField[i0][j0]
	v10 := ss.RealField[i1][j0]
	v01 := ss.RealField[i0][j1]
	v11 := ss.RealField[i1][j1]
	
	v0 := v00*(1-tx) + v10*tx
	v1 := v01*(1-tx) + v11*tx
	
	return v0*(1-ty) + v1*ty
}

// GetNoise evaluates the spectral field at a 3D position (using X,Y coordinates).
// Implements ScalarField3D interface.
func (ss *SpectralSynthesis) GetNoise(pos icosphere.Vector3D) float32 {
	return float32(ss.GetNoise2D(pos.X, pos.Y))
}

// UpdateTime evolves the spectral field in time.
func (ss *SpectralSynthesis) UpdateTime(deltaTime float64) {
	n := ss.Settings.Resolution
	
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			k := ss.FreqGrid[i][j]
			if k > 0 {
				// Dispersion relation for the given type
				var omega float64
				switch ss.Settings.Type {
				case SpectralOceanWaves:
					omega = math.Sqrt(9.81 * k) // Deep water waves
				case SpectralAtmospheric:
					omega = ss.Settings.GroupVelocity * k // Linear dispersion
				default:
					omega = k // Simple linear dispersion
				}
				
				// Update phase
				ss.Phases[i][j] += omega * deltaTime
				if ss.Phases[i][j] > 2*math.Pi {
					ss.Phases[i][j] -= 2*math.Pi
				}
			}
		}
	}
	
	ss.initialized = false // Force regeneration
}

// --- Spectral Synthesis Builder ---

// SpectralSynthesisBuilder provides a fluent interface for building spectral synthesis.
type SpectralSynthesisBuilder struct {
	settings SpectralSettings
}

// NewSpectralSynthesisBuilder creates a new builder for spectral synthesis.
func NewSpectralSynthesisBuilder() *SpectralSynthesisBuilder {
	return &SpectralSynthesisBuilder{
		settings: SpectralSettings{
			Type:             SpectralTerrain,
			Resolution:       64,
			DomainSize:       1.0,
			PowerLawExponent: -2.0,
			EnergyScale:      1.0,
			MinWavelength:    0.01,
			MaxWavelength:    1.0,
			WindSpeed:        10.0,
			GroupVelocity:    1.0,
			TimeStep:         0.1,
			PhaseRandomness:  1.0,
		},
	}
}

// SetType sets the spectral type.
func (b *SpectralSynthesisBuilder) SetType(sType SpectralType) *SpectralSynthesisBuilder {
	b.settings.Type = sType
	return b
}

// SetResolution sets the grid resolution.
func (b *SpectralSynthesisBuilder) SetResolution(resolution int) *SpectralSynthesisBuilder {
	b.settings.Resolution = resolution
	return b
}

// SetDomainSize sets the physical domain size.
func (b *SpectralSynthesisBuilder) SetDomainSize(size float64) *SpectralSynthesisBuilder {
	b.settings.DomainSize = size
	return b
}

// SetPowerLaw sets the power law exponent.
func (b *SpectralSynthesisBuilder) SetPowerLaw(exponent float64) *SpectralSynthesisBuilder {
	b.settings.PowerLawExponent = exponent
	return b
}

// SetEnergyScale sets the overall energy scaling.
func (b *SpectralSynthesisBuilder) SetEnergyScale(scale float64) *SpectralSynthesisBuilder {
	b.settings.EnergyScale = scale
	return b
}

// SetWavelengthRange sets the wavelength cutoffs.
func (b *SpectralSynthesisBuilder) SetWavelengthRange(min, max float64) *SpectralSynthesisBuilder {
	b.settings.MinWavelength = min
	b.settings.MaxWavelength = max
	return b
}

// SetOceanParameters sets parameters for ocean wave generation.
func (b *SpectralSynthesisBuilder) SetOceanParameters(windSpeed, windDirection, fetchLength float64) *SpectralSynthesisBuilder {
	b.settings.WindSpeed = windSpeed
	b.settings.WindDirection = windDirection
	b.settings.FetchLength = fetchLength
	return b
}

// SetAtmosphericParameters sets parameters for atmospheric generation.
func (b *SpectralSynthesisBuilder) SetAtmosphericParameters(correlationLength, dissipationRate float64) *SpectralSynthesisBuilder {
	b.settings.CorrelationLength = correlationLength
	b.settings.DissipationRate = dissipationRate
	return b
}

// SetSeed sets the random seed.
func (b *SpectralSynthesisBuilder) SetSeed(seed int64) *SpectralSynthesisBuilder {
	b.settings.Seed = seed
	return b
}

// Build creates the spectral synthesis generator.
func (b *SpectralSynthesisBuilder) Build() *SpectralSynthesis {
	return NewSpectralSynthesis(b.settings)
}