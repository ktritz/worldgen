package climgen

import (
	"fmt"
	"math"
)

// =============================================================================
// WIND/ATMOSPHERE GENERATION - MAIN ENTRY POINT AND TYPES
// =============================================================================
// This file contains the main entry point and all types for wind generation.
// The implementation is split across several files:
//   - wind.go (this file): Entry point, settings, result types
//   - wind_circulation.go: Pressure cells and geostrophic wind
//   - wind_surface.go: Surface friction effects
//   - wind_orographic.go: Mountain blocking and deflection

// --- Wind Constants ---
// Based on Earth atmospheric circulation parameters

const (
	// Circulation cell boundaries (degrees latitude)
	DefaultHadleyEdgeLat = 30.0 // Trade winds -> Westerlies transition
	DefaultFerrelEdgeLat = 60.0 // Westerlies -> Polar easterlies transition

	// Pressure and wind parameters
	DefaultPressureStrength = 1.0 // Base pressure gradient magnitude
	DefaultCoriolisScale    = 1.0 // Coriolis effect multiplier

	// Surface friction coefficients
	DefaultLandFriction  = 0.3 // Higher friction over land (30% speed reduction)
	DefaultOceanFriction = 0.1 // Lower friction over ocean (10% speed reduction)
	DefaultMaxWindSpeed  = 1.0 // Normalized max speed for stability

	// Orographic parameters
	DefaultBlockingThreshold  = 1500.0 // Meters - mountains block flow above this
	DefaultDeflectionStrength = 0.7    // How strongly mountains deflect (0-1)
	DefaultChannelSpeedup     = 1.3    // Speed increase through gaps

	// Smoothing parameters
	DefaultWindSmoothingFactor = 0.3 // Blend weight for diffusion
)

// CirculationZone represents the atmospheric circulation cell a vertex belongs to.
type CirculationZone int

const (
	ZoneHadley CirculationZone = 0 // 0-30 degrees: Trade winds
	ZoneFerrel CirculationZone = 1 // 30-60 degrees: Westerlies
	ZonePolar  CirculationZone = 2 // 60-90 degrees: Polar easterlies
)

// String returns the zone name.
func (z CirculationZone) String() string {
	names := map[CirculationZone]string{
		ZoneHadley: "Hadley",
		ZoneFerrel: "Ferrel",
		ZonePolar:  "Polar",
	}
	if name, ok := names[z]; ok {
		return name
	}
	return "Unknown"
}

// --- Settings Structs ---

// CirculationSettings controls Hadley/Ferrel/Polar cell parameters.
type CirculationSettings struct {
	HadleyEdgeLat          float64 `json:"hadleyEdgeLat"`          // Latitude of Hadley cell edge (degrees)
	FerrelEdgeLat          float64 `json:"ferrelEdgeLat"`          // Latitude of Ferrel cell edge (degrees)
	ThermalEquatorShiftDeg float64 `json:"thermalEquatorShiftDeg"` // Seasonal shift of circulation cells (degrees, north positive)
	PressureStrength       float64 `json:"pressureStrength"`       // Base pressure gradient magnitude
	CoriolisScale          float64 `json:"coriolisScale"`          // Coriolis effect multiplier
	SmoothingFactor        float64 `json:"smoothingFactor"`        // Blend weight for pressure smoothing

	// Rossby wave parameters (mid-latitude meandering)
	RossbyWavenumber int     `json:"rossbyWavenumber"` // Number of waves around globe (Earth: 4-6)
	RossbyAmplitude  float64 `json:"rossbyAmplitude"`  // Strength of meridional perturbation (0-1)
	RossbyPhase      float64 `json:"rossbyPhase"`      // Phase offset in radians (for seed variation)

	// Pressure basin parameters (geographic modulation of pressure field)
	SubtropicalHighStrength float64 `json:"subtropicalHighStrength"` // Strength of subtropical ocean highs (~30°)
	SubpolarLowStrength     float64 `json:"subpolarLowStrength"`     // Strength of subpolar ocean lows (~60°)
	ContinentalLowStrength  float64 `json:"continentalLowStrength"`  // Strength of thermal lows over continents

	Verbose bool `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s CirculationSettings) Validate() error {
	if s.HadleyEdgeLat <= 0 || s.HadleyEdgeLat >= 90 {
		return fmt.Errorf("hadleyEdgeLat must be in (0, 90), got %f", s.HadleyEdgeLat)
	}
	if s.FerrelEdgeLat <= s.HadleyEdgeLat || s.FerrelEdgeLat >= 90 {
		return fmt.Errorf("ferrelEdgeLat must be in (%f, 90), got %f", s.HadleyEdgeLat, s.FerrelEdgeLat)
	}
	if math.Abs(s.ThermalEquatorShiftDeg) > 45 {
		return fmt.Errorf("thermalEquatorShiftDeg must be in [-45, 45], got %f", s.ThermalEquatorShiftDeg)
	}
	if s.PressureStrength <= 0 {
		return fmt.Errorf("pressureStrength must be positive, got %f", s.PressureStrength)
	}
	if s.CoriolisScale <= 0 {
		return fmt.Errorf("coriolisScale must be positive, got %f", s.CoriolisScale)
	}
	if s.SmoothingFactor < 0 || s.SmoothingFactor > 1 {
		return fmt.Errorf("smoothingFactor must be in [0, 1], got %f", s.SmoothingFactor)
	}
	return nil
}

// DefaultCirculationSettings returns Earth-like defaults for circulation.
func DefaultCirculationSettings() CirculationSettings {
	return CirculationSettings{
		HadleyEdgeLat:          DefaultHadleyEdgeLat,
		FerrelEdgeLat:          DefaultFerrelEdgeLat,
		ThermalEquatorShiftDeg: 0.0,
		PressureStrength:       DefaultPressureStrength,
		CoriolisScale:          DefaultCoriolisScale,
		SmoothingFactor:        DefaultWindSmoothingFactor,
		RossbyWavenumber:       5,   // Earth typically has 4-6 Rossby waves
		RossbyAmplitude:        0.8, // Visible meandering (0.4=subtle, 1.2=extreme)
		RossbyPhase:            0.0, // Will be set from seed
		// Pressure basin strengths (fraction of base pressure)
		SubtropicalHighStrength: 0.3, // Ocean highs at ~30° (Azores, Pacific High)
		SubpolarLowStrength:     0.3, // Ocean lows at ~60° (Icelandic, Aleutian Low)
		ContinentalLowStrength:  0.2, // Thermal lows over hot continents
		Verbose:                 false,
	}
}

// SurfaceWindSettings controls ground-level wind derivation.
type SurfaceWindSettings struct {
	LandFriction  float64 `json:"landFriction"`  // Friction coefficient over land (0-1)
	OceanFriction float64 `json:"oceanFriction"` // Friction coefficient over ocean (0-1)
	MaxWindSpeed  float64 `json:"maxWindSpeed"`  // Stability clamp for wind speed
	Verbose       bool    `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s SurfaceWindSettings) Validate() error {
	if s.LandFriction < 0 || s.LandFriction > 1 {
		return fmt.Errorf("landFriction must be in [0, 1], got %f", s.LandFriction)
	}
	if s.OceanFriction < 0 || s.OceanFriction > 1 {
		return fmt.Errorf("oceanFriction must be in [0, 1], got %f", s.OceanFriction)
	}
	if s.MaxWindSpeed <= 0 {
		return fmt.Errorf("maxWindSpeed must be positive, got %f", s.MaxWindSpeed)
	}
	return nil
}

// DefaultSurfaceWindSettings returns Earth-like defaults for surface winds.
func DefaultSurfaceWindSettings() SurfaceWindSettings {
	return SurfaceWindSettings{
		LandFriction:  DefaultLandFriction,
		OceanFriction: DefaultOceanFriction,
		MaxWindSpeed:  DefaultMaxWindSpeed,
		Verbose:       false,
	}
}

// OrographicSettings controls mountain interaction with wind.
type OrographicSettings struct {
	BlockingThreshold  float64 `json:"blockingThreshold"`  // Elevation (m) above which flow is blocked
	DeflectionStrength float64 `json:"deflectionStrength"` // How strongly mountains deflect (0-1)
	ChannelSpeedup     float64 `json:"channelSpeedup"`     // Speed multiplier through gaps
	Verbose            bool    `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s OrographicSettings) Validate() error {
	if s.BlockingThreshold < 0 {
		return fmt.Errorf("blockingThreshold must be non-negative, got %f", s.BlockingThreshold)
	}
	if s.DeflectionStrength < 0 || s.DeflectionStrength > 1 {
		return fmt.Errorf("deflectionStrength must be in [0, 1], got %f", s.DeflectionStrength)
	}
	if s.ChannelSpeedup < 1 {
		return fmt.Errorf("channelSpeedup must be >= 1, got %f", s.ChannelSpeedup)
	}
	return nil
}

// DefaultOrographicSettings returns Earth-like defaults for orographic effects.
func DefaultOrographicSettings() OrographicSettings {
	return OrographicSettings{
		BlockingThreshold:  DefaultBlockingThreshold,
		DeflectionStrength: DefaultDeflectionStrength,
		ChannelSpeedup:     DefaultChannelSpeedup,
		Verbose:            false,
	}
}

// WindSettings contains all parameters for wind generation.
type WindSettings struct {
	Seed        int64               `json:"seed"`
	Circulation CirculationSettings `json:"circulation"`
	Surface     SurfaceWindSettings `json:"surface"`
	Orographic  OrographicSettings  `json:"orographic"`
	Verbose     bool                `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s WindSettings) Validate() error {
	if err := s.Circulation.Validate(); err != nil {
		return fmt.Errorf("circulation settings: %w", err)
	}
	if err := s.Surface.Validate(); err != nil {
		return fmt.Errorf("surface settings: %w", err)
	}
	if err := s.Orographic.Validate(); err != nil {
		return fmt.Errorf("orographic settings: %w", err)
	}
	return nil
}

// ApplyVerbose propagates the verbose flag to all sub-settings.
func (s *WindSettings) ApplyVerbose() {
	if s.Verbose {
		s.Circulation.Verbose = true
		s.Surface.Verbose = true
		s.Orographic.Verbose = true
	}
}

// DefaultWindSettings returns Earth-like defaults for wind generation.
func DefaultWindSettings() WindSettings {
	return WindSettings{
		Seed:        42,
		Circulation: DefaultCirculationSettings(),
		Surface:     DefaultSurfaceWindSettings(),
		Orographic:  DefaultOrographicSettings(),
		Verbose:     false,
	}
}

// --- Result Structures ---

// WindResult contains the output from wind generation.
type WindResult struct {
	// Core outputs
	MarineWind  []Vector3D // Basin-scale marine wind for ocean forcing/SST transport
	SurfaceWind []Vector3D // Terrain-aware near-surface wind for local climate
	Pressure    []float64  // Atmospheric pressure at each vertex (normalized)

	// Diagnostic layers (for visualization/debugging)
	GeostrophicWind []Vector3D        // Upper-level wind before friction
	CirculationZone []CirculationZone // Zone classification per vertex
}

// --- Main Entry Point ---

// GenerateWindField is the main entry point for wind generation.
// Uses pressure-driven geostrophic balance with friction and orographic effects.
func GenerateWindField(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	settings WindSettings,
) (*WindResult, error) {
	settings.ApplyVerbose()

	if err := settings.Validate(); err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	numVertices := len(vertices)
	if settings.Verbose {
		fmt.Printf("=== Wind Generation ===\n")
		fmt.Printf("  Vertices: %d\n", numVertices)
	}

	// Set Rossby wave phase from seed for procedural variation
	// Use a simple hash of the seed to get a phase offset
	settings.Circulation.RossbyPhase = float64(settings.Seed%1000) * 2 * math.Pi / 1000

	// Step 1: Compute pressure field and zone classification
	if settings.Verbose {
		fmt.Println("  Computing circulation pressure field...")
	}
	pressure, zones := ComputeCirculationPressure(vertices, settings.Circulation)

	// Step 1b: Add geographic pressure anomalies (ocean highs, continental lows)
	if settings.Verbose {
		fmt.Println("  Adding geographic pressure anomalies...")
	}
	pressure = AddGeographicPressureAnomalies(
		pressure, vertices, elevation, seaLevelThreshold, adj, settings.Circulation,
	)

	if settings.Verbose {
		minP, maxP := pressure[0], pressure[0]
		for _, p := range pressure {
			if p < minP {
				minP = p
			}
			if p > maxP {
				maxP = p
			}
		}
		fmt.Printf("  Pressure range: [%.4f, %.4f]\n", minP, maxP)
	}

	// Step 2: Compute cell-driven surface wind (thermal circulation + Coriolis)
	// This gives the characteristic curved trade winds, westerlies, and polar easterlies
	if settings.Verbose {
		fmt.Println("  Computing cell-driven circulation wind...")
	}
	cellWind := ComputeCellDrivenWind(vertices, zones, settings.Circulation)

	if settings.Verbose {
		maxSpeed := 0.0
		for _, v := range cellWind {
			speed := Length(v)
			if speed > maxSpeed {
				maxSpeed = speed
			}
		}
		fmt.Printf("  Cell-driven wind max: %.4f\n", maxSpeed)
	}

	// Smooth pressure field for gradient computation
	cellSize := estimateCellSize(vertices, adj)
	const pressureSmoothAngular = 0.03
	pressureSmoothIters := int(pressureSmoothAngular/cellSize) + 1
	if pressureSmoothIters < 2 {
		pressureSmoothIters = 2
	}
	smoothedPressure := SmoothScalarField(pressure, vertices, adj, pressureSmoothIters, settings.Circulation.SmoothingFactor)

	// Step 2b: Add pressure gradient wind from geographic anomalies
	// This makes winds curve around subtropical highs, subpolar lows, etc.
	if settings.Verbose {
		fmt.Println("  Computing pressure gradient wind...")
	}
	pressureGradientStrength := 0.24 // Keep basin-scale steering without overpowering zonal cells
	pressureWind := ComputePressureGradientWind(smoothedPressure, vertices, adj, pressureGradientStrength)

	// Combine cell-driven wind with pressure gradient perturbations
	for i := range cellWind {
		cellWind[i] = Add(cellWind[i], pressureWind[i])
	}

	if settings.Verbose {
		maxSpeed := 0.0
		for _, v := range cellWind {
			speed := Length(v)
			if speed > maxSpeed {
				maxSpeed = speed
			}
		}
		fmt.Printf("  Combined wind max: %.4f\n", maxSpeed)
	}

	// Keep geostrophic wind for the result struct (diagnostic layer)
	geostrophicWind := ComputeGeostrophicWind(smoothedPressure, vertices, adj, settings.Circulation)

	// Step 3: Build a smoother marine wind product before terrain perturbations.
	if settings.Verbose {
		fmt.Println("  Building basin-scale marine wind...")
	}
	marineWind := BuildMarineWind(
		cellWind, vertices, elevation, seaLevelThreshold, adj, settings.Surface,
	)

	// Step 4: Apply surface friction (speed reduction over land)
	if settings.Verbose {
		fmt.Println("  Applying surface friction...")
	}
	surfaceWind := ApplySurfaceFrictionSimple(
		cellWind, vertices, elevation, seaLevelThreshold, settings.Surface,
	)

	// Step 5: Apply slope effects (uphill deceleration, downhill acceleration)
	if settings.Verbose {
		fmt.Println("  Applying slope effects...")
	}
	surfaceWind = ApplySlopeEffects(surfaceWind, vertices, elevation, adj)

	// Step 6: Apply orographic deflection
	if settings.Orographic.DeflectionStrength > 0 {
		if settings.Verbose {
			fmt.Println("  Applying orographic deflection...")
		}
		surfaceWind = ApplyOrographicDeflection(
			surfaceWind, vertices, elevation, adj, settings.Orographic,
		)

		// Step 6b: Propagate lee-side wind shadows
		// Wind stays slow downwind of mountains
		if settings.Verbose {
			fmt.Println("  Propagating lee-side wind shadows...")
		}
		leeShadowIters := int(0.05/cellSize) + 1 // ~3 degrees of shadow propagation
		if leeShadowIters < 3 {
			leeShadowIters = 3
		}
		if leeShadowIters > 10 {
			leeShadowIters = 10
		}
		surfaceWind = PropagateLeeShadow(
			surfaceWind, vertices, elevation, adj, leeShadowIters, 0.75,
		)
	}

	// Step 7: Final smoothing for stability
	const windSmoothAngular = 0.025 // radians ≈ 1.4 degrees
	windSmoothIters := int(windSmoothAngular/cellSize) + 1
	if windSmoothIters < 2 {
		windSmoothIters = 2
	}
	if settings.Verbose {
		fmt.Printf("  Smoothing wind field (%d iterations)...\n", windSmoothIters)
	}
	surfaceWind = SmoothVectorFieldBySurface(
		surfaceWind, vertices, elevation, seaLevelThreshold, adj, windSmoothIters, 0.3,
	)

	if settings.Verbose {
		maxSpeed := 0.0
		for _, v := range surfaceWind {
			speed := Length(v)
			if speed > maxSpeed {
				maxSpeed = speed
			}
		}
		fmt.Printf("  Final wind max: %.4f\n", maxSpeed)
	}

	return &WindResult{
		MarineWind:      marineWind,
		SurfaceWind:     surfaceWind,
		Pressure:        pressure,
		GeostrophicWind: geostrophicWind,
		CirculationZone: zones,
	}, nil
}

// --- Helper: Get circulation zone for latitude ---

func getCirculationZone(lat float64, settings CirculationSettings) CirculationZone {
	absLat := math.Abs(effectiveCirculationLatitude(lat, settings)) * 180.0 / math.Pi
	if absLat < settings.HadleyEdgeLat {
		return ZoneHadley
	} else if absLat < settings.FerrelEdgeLat {
		return ZoneFerrel
	}
	return ZonePolar
}
