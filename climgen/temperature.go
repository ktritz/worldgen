package climgen

import (
	"fmt"
	"math"
)

// =============================================================================
// TEMPERATURE GENERATION - MAIN ENTRY POINT AND TYPES
// =============================================================================
// This file contains the main entry point and all types for temperature generation.
// The implementation is split across several files:
//   - temperature.go (this file): Entry point, settings, result types
//   - temperature_solar.go: Insolation and albedo calculation
//   - temperature_transport.go: Heat diffusion and advection
//   - temperature_balance.go: Energy balance iteration

// --- Physical Constants ---
// Based on Earth energy balance parameters

const (
	// Solar radiation
	SolarConstant = 1361.0 // W/m² (solar irradiance at Earth orbit)

	// Thermal radiation
	StefanBoltzmann = 5.67e-8 // W/m²/K⁴ (Stefan-Boltzmann constant)

	// Albedo values (includes partial cloud cover effect)
	AlbedoIce   = 0.55 // Ice/snow cover albedo
	AlbedoLand  = 0.25 // Land albedo (with some cloud cover)
	AlbedoWater = 0.12 // Ocean albedo (tropical oceans have few clouds)

	// Temperature thresholds
	FreezingPoint = 273.15 // K (0°C)

	// Lapse rate
	LapseRate = 0.0065 // K/m (6.5°C per 1000m elevation)

	// Heat capacity (J/m²/K) - effective thermal mass for ~1m surface layer
	HeatCapacityLand  = 1.0e5 // Land heats/cools quickly
	HeatCapacityWater = 4.2e6 // Ocean has high thermal inertia

	// Emissivity
	EmissivityLand  = 0.82 // Typical land surface
	EmissivityWater = 0.96 // Ocean water

	// Greenhouse effect - fraction of OLR escaping to space
	AtmosphericTransmissivity = 0.75

	// Default diffusion rates (s⁻¹)
	DefaultDiffusionLand  = 1e-6  // Minimal diffusion over land
	DefaultDiffusionWater = 5e-6  // Moderate diffusion smooths current boundaries

	// Default iteration parameters
	DefaultMaxIterations = 1000
	DefaultTolerance     = 0.1    // K convergence threshold (0.1°C is plenty accurate)
	DefaultTimeStep      = 1800.0 // 30 minutes

	// Default transport scales - multipliers on physical advection
	// 1.0 = pure physics (v * dt / cellSize), adjust if needed
	DefaultWindTransportScale     = 1.0   // Wind advection over land
	DefaultCurrentTransportScale  = 1.0   // Physical current advection (local neighbor blending)
	DefaultCurrentOriginForcing      = 100.0  // W/m² - how strongly currents impose source-latitude temps
	DefaultCurrentBacktrackDistance  = 3500.0 // km - how far to trace back along current streamlines (~30 cells at level 6)
	DefaultWindOriginForcing         = 8.0   // Test stronger forcing
	DefaultWindBacktrackDistance     = 800.0
)

// --- Settings Structs ---

// SolarSettings controls solar insolation calculation.
type SolarSettings struct {
	SolarLuminosity float64 `json:"solarLuminosity"` // Multiplier for solar constant (1.0 = Earth)
	AxialTilt       float64 `json:"axialTilt"`       // Degrees (23.5 for Earth) - for seasonal variation
	SeasonPhase     float64 `json:"seasonPhase"`     // 0-1, position in year (0 = northern winter solstice)
	Verbose         bool    `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s SolarSettings) Validate() error {
	if s.SolarLuminosity <= 0 {
		return fmt.Errorf("solarLuminosity must be positive, got %f", s.SolarLuminosity)
	}
	if s.AxialTilt < 0 || s.AxialTilt > 90 {
		return fmt.Errorf("axialTilt must be in [0, 90], got %f", s.AxialTilt)
	}
	if s.SeasonPhase < 0 || s.SeasonPhase > 1 {
		return fmt.Errorf("seasonPhase must be in [0, 1], got %f", s.SeasonPhase)
	}
	return nil
}

// DefaultSolarSettings returns Earth-like defaults for solar insolation.
func DefaultSolarSettings() SolarSettings {
	return SolarSettings{
		SolarLuminosity: 1.0,
		AxialTilt:       0.0,   // No seasonal variation by default
		SeasonPhase:     0.0,   // Not used when AxialTilt = 0
		Verbose:         false,
	}
}

// TransportSettings controls heat diffusion and advection.
type TransportSettings struct {
	DiffusionLand              float64 `json:"diffusionLand"`              // Base diffusion rate over land (s⁻¹)
	DiffusionWater             float64 `json:"diffusionWater"`             // Base diffusion rate over water (s⁻¹)
	WindTransportScale         float64 `json:"windTransportScale"`         // Multiplier on physical wind advection
	CurrentTransportScale      float64 `json:"currentTransportScale"`      // Multiplier on physical current advection
	CurrentOriginForcing       float64 `json:"currentOriginForcing"`       // W/m² forcing from current source-latitude temps
	CurrentBacktrackDistance   float64 `json:"currentBacktrackDistance"`   // km to backtrack along current streamlines
	WindOriginForcing          float64 `json:"windOriginForcing"`          // W/m² forcing from wind source temps on land
	WindBacktrackDistance      float64 `json:"windBacktrackDistance"`      // km to backtrack along wind streamlines
	AtmosphericHeatTransport   float64 `json:"atmosphericHeatTransport"`   // Meridional heat flux (W/m²) from Hadley/Ferrel/Polar cells
	Verbose                    bool    `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s TransportSettings) Validate() error {
	if s.DiffusionLand < 0 {
		return fmt.Errorf("diffusionLand must be non-negative, got %f", s.DiffusionLand)
	}
	if s.DiffusionWater < 0 {
		return fmt.Errorf("diffusionWater must be non-negative, got %f", s.DiffusionWater)
	}
	// Transport scales can exceed 1.0 for stronger effects
	if s.WindTransportScale < 0 || s.WindTransportScale > 10 {
		return fmt.Errorf("windTransportScale must be in [0, 10], got %f", s.WindTransportScale)
	}
	if s.CurrentTransportScale < 0 || s.CurrentTransportScale > 10 {
		return fmt.Errorf("currentTransportScale must be in [0, 10], got %f", s.CurrentTransportScale)
	}
	return nil
}

// DefaultTransportSettings returns Earth-like defaults for heat transport.
func DefaultTransportSettings() TransportSettings {
	return TransportSettings{
		DiffusionLand:            DefaultDiffusionLand,
		DiffusionWater:           DefaultDiffusionWater,
		WindTransportScale:       DefaultWindTransportScale,
		CurrentTransportScale:    DefaultCurrentTransportScale,
		CurrentOriginForcing:     DefaultCurrentOriginForcing,
		CurrentBacktrackDistance: DefaultCurrentBacktrackDistance,
		WindOriginForcing:        DefaultWindOriginForcing,
		WindBacktrackDistance:    DefaultWindBacktrackDistance,
		AtmosphericHeatTransport: 50.0, // W/m² - represents Hadley/Ferrel/Polar cell heat redistribution
		Verbose:                  false,
	}
}

// BalanceSettings controls energy balance iteration.
type BalanceSettings struct {
	MaxIterations     int     `json:"maxIterations"`     // Maximum iteration count
	Tolerance         float64 `json:"tolerance"`         // Convergence threshold in K
	TimeStep          float64 `json:"timeStep"`          // Seconds per iteration
	IceAlbedoFeedback bool    `json:"iceAlbedoFeedback"` // Enable ice-albedo feedback
	Verbose           bool    `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s BalanceSettings) Validate() error {
	if s.MaxIterations < 1 {
		return fmt.Errorf("maxIterations must be >= 1, got %d", s.MaxIterations)
	}
	if s.Tolerance <= 0 {
		return fmt.Errorf("tolerance must be positive, got %f", s.Tolerance)
	}
	if s.TimeStep <= 0 {
		return fmt.Errorf("timeStep must be positive, got %f", s.TimeStep)
	}
	return nil
}

// DefaultBalanceSettings returns defaults for energy balance iteration.
func DefaultBalanceSettings() BalanceSettings {
	return BalanceSettings{
		MaxIterations:     DefaultMaxIterations,
		Tolerance:         DefaultTolerance,
		TimeStep:          DefaultTimeStep,
		IceAlbedoFeedback: true,
		Verbose:           false,
	}
}

// TemperatureSettings contains all parameters for temperature generation.
type TemperatureSettings struct {
	Seed      int64             `json:"seed"`
	Solar     SolarSettings     `json:"solar"`
	Transport TransportSettings `json:"transport"`
	Balance   BalanceSettings   `json:"balance"`
	Verbose   bool              `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s TemperatureSettings) Validate() error {
	if err := s.Solar.Validate(); err != nil {
		return fmt.Errorf("solar settings: %w", err)
	}
	if err := s.Transport.Validate(); err != nil {
		return fmt.Errorf("transport settings: %w", err)
	}
	if err := s.Balance.Validate(); err != nil {
		return fmt.Errorf("balance settings: %w", err)
	}
	return nil
}

// ApplyVerbose propagates the verbose flag to all sub-settings.
func (s *TemperatureSettings) ApplyVerbose() {
	if s.Verbose {
		s.Solar.Verbose = true
		s.Transport.Verbose = true
		s.Balance.Verbose = true
	}
}

// DefaultTemperatureSettings returns Earth-like defaults for temperature generation.
func DefaultTemperatureSettings() TemperatureSettings {
	return TemperatureSettings{
		Seed:      42,
		Solar:     DefaultSolarSettings(),
		Transport: DefaultTransportSettings(),
		Balance:   DefaultBalanceSettings(),
		Verbose:   false,
	}
}

// --- Result Structures ---

// TemperatureResult contains the output from temperature generation.
type TemperatureResult struct {
	// Core outputs
	Temperature        []float64 // Surface temperature in Kelvin
	TemperatureCelsius []float64 // Convenience: T - 273.15

	// Diagnostic layers (for visualization/debugging)
	Insolation       []float64 // Incoming solar radiation (W/m²)
	AbsorbedSolar    []float64 // Absorbed solar radiation after albedo (W/m²)
	OutgoingLongwave []float64 // Emitted longwave radiation (W/m²)
	Albedo           []float64 // Effective albedo at each vertex
	HeatTransport    []float64 // Net heat transported to each vertex (W/m²)

	// Convergence info
	Iterations    int     // Number of iterations to converge
	FinalMaxDelta float64 // Maximum temperature change in final iteration
	Converged     bool    // Whether tolerance was reached
}

// TemperatureStats returns summary statistics for the temperature field.
type TemperatureStats struct {
	MinK, MaxK, MeanK float64 // Kelvin
	MinC, MaxC, MeanC float64 // Celsius
	LandMeanC         float64 // Mean land temperature
	OceanMeanC        float64 // Mean ocean temperature
}

// ComputeStats calculates summary statistics for the temperature result.
func (r *TemperatureResult) ComputeStats(elevation []float64, seaLevelThreshold float64) TemperatureStats {
	var stats TemperatureStats

	if len(r.Temperature) == 0 {
		return stats
	}

	stats.MinK = r.Temperature[0]
	stats.MaxK = r.Temperature[0]
	sumK := 0.0
	landSum, landCount := 0.0, 0
	oceanSum, oceanCount := 0.0, 0

	for i, t := range r.Temperature {
		if t < stats.MinK {
			stats.MinK = t
		}
		if t > stats.MaxK {
			stats.MaxK = t
		}
		sumK += t

		if elevation[i] >= seaLevelThreshold {
			landSum += t
			landCount++
		} else {
			oceanSum += t
			oceanCount++
		}
	}

	n := float64(len(r.Temperature))
	stats.MeanK = sumK / n
	stats.MinC = stats.MinK - FreezingPoint
	stats.MaxC = stats.MaxK - FreezingPoint
	stats.MeanC = stats.MeanK - FreezingPoint

	if landCount > 0 {
		stats.LandMeanC = (landSum / float64(landCount)) - FreezingPoint
	}
	if oceanCount > 0 {
		stats.OceanMeanC = (oceanSum / float64(oceanCount)) - FreezingPoint
	}

	return stats
}

// --- Main Entry Point ---

// GenerateTemperature is the main entry point for temperature generation.
// Uses an energy balance model with solar insolation, heat transport, and radiation.
//
// Parameters:
//   - vertices: Sphere mesh vertices (normalized to unit sphere)
//   - elevation: Elevation in meters (negative = ocean)
//   - seaLevelThreshold: Elevation threshold for land/ocean (typically 0)
//   - adj: Flat adjacency structure for neighbor queries
//   - wind: Wind field result (optional, can be nil)
//   - currents: Ocean current result (optional, can be nil)
//   - settings: Temperature generation settings
func GenerateTemperature(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind *WindResult,
	currents *OceanCurrentResult,
	settings TemperatureSettings,
) (*TemperatureResult, error) {
	settings.ApplyVerbose()

	if err := settings.Validate(); err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	numVertices := len(vertices)
	if settings.Verbose {
		fmt.Printf("=== Temperature Generation ===\n")
		fmt.Printf("  Vertices: %d\n", numVertices)
		fmt.Printf("  Max iterations: %d, tolerance: %.4f K\n",
			settings.Balance.MaxIterations, settings.Balance.Tolerance)
	}

	// Step 1: Compute solar insolation
	if settings.Verbose {
		fmt.Println("  Computing solar insolation...")
	}
	insolation := ComputeInsolation(vertices, settings.Solar)

	if settings.Verbose {
		minQ, maxQ := insolation[0], insolation[0]
		for _, q := range insolation {
			if q < minQ {
				minQ = q
			}
			if q > maxQ {
				maxQ = q
			}
		}
		fmt.Printf("  Insolation range: [%.1f, %.1f] W/m²\n", minQ, maxQ)
	}

	// Step 2: Extract wind and current vectors (if provided)
	var windVectors []Vector3D
	var currentVectors []Vector3D

	if wind != nil {
		windVectors = wind.SurfaceWind
		if settings.Verbose {
			fmt.Println("  Using wind field for heat advection")
		}
	}
	if currents != nil {
		currentVectors = currents.Currents
		if settings.Verbose {
			fmt.Println("  Using ocean currents for heat advection")
		}
	}

	// Step 3: Solve energy balance
	if settings.Verbose {
		fmt.Println("  Solving energy balance...")
	}
	temperature, iterations, finalDelta, converged := SolveEnergyBalance(
		insolation,
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		windVectors,
		currentVectors,
		settings,
	)

	if settings.Verbose {
		if converged {
			fmt.Printf("  Converged after %d iterations (max delta: %.4f K)\n",
				iterations, finalDelta)
		} else {
			fmt.Printf("  Did not converge after %d iterations (max delta: %.4f K)\n",
				iterations, finalDelta)
		}
	}

	// Step 4: Compute continentality and apply effects
	if settings.Verbose {
		fmt.Println("  Computing continentality effects...")
	}
	continentality := ComputeContinentality(vertices, elevation, seaLevelThreshold, adj, 1500.0)
	temperature = ApplyContinentalityEffect(temperature, vertices, continentality, 10.0)

	// Step 5: Apply marine influence (coasts moderated by ocean)
	if settings.Verbose {
		fmt.Println("  Applying marine influence on coasts...")
	}
	temperature = ApplyMarineInfluence(temperature, vertices, elevation, seaLevelThreshold, adj, 0.5, 350.0)

	// Step 6: Apply lapse rate correction for elevation
	if settings.Verbose {
		fmt.Println("  Applying elevation lapse rate correction...")
	}
	temperature = ApplyLapseRateCorrection(temperature, elevation, seaLevelThreshold)

	// Step 7: Compute diagnostic fields
	albedo := ComputeAlbedo(temperature, elevation, seaLevelThreshold, settings.Balance.IceAlbedoFeedback)
	absorbedSolar := ComputeAbsorbedSolar(insolation, albedo)
	olr := ComputeOutgoingLongwave(temperature, elevation, seaLevelThreshold)
	heatTransport := ComputeTotalHeatTransport(
		temperature, vertices, elevation, seaLevelThreshold, adj,
		windVectors, currentVectors, settings.Transport,
	)

	// Convert to Celsius for convenience
	tempCelsius := make([]float64, numVertices)
	for i, t := range temperature {
		tempCelsius[i] = t - FreezingPoint
	}

	result := &TemperatureResult{
		Temperature:        temperature,
		TemperatureCelsius: tempCelsius,
		Insolation:         insolation,
		AbsorbedSolar:      absorbedSolar,
		OutgoingLongwave:   olr,
		Albedo:             albedo,
		HeatTransport:      heatTransport,
		Iterations:         iterations,
		FinalMaxDelta:      finalDelta,
		Converged:          converged,
	}

	if settings.Verbose {
		stats := result.ComputeStats(elevation, seaLevelThreshold)
		fmt.Printf("\n=== Temperature Statistics ===\n")
		fmt.Printf("  Range: %.1f°C to %.1f°C\n", stats.MinC, stats.MaxC)
		fmt.Printf("  Global mean: %.1f°C\n", stats.MeanC)
		fmt.Printf("  Land mean: %.1f°C, Ocean mean: %.1f°C\n",
			stats.LandMeanC, stats.OceanMeanC)
	}

	return result, nil
}

// --- Helper Functions ---

// getLatitude returns the latitude in radians for a point on the unit sphere.
// Y-up coordinate system: Y = sin(latitude).
func getLatitude(v Vector3D) float64 {
	// Clamp Y to valid range for numerical safety
	y := v.Y
	if y > 1.0 {
		y = 1.0
	} else if y < -1.0 {
		y = -1.0
	}
	return math.Asin(y)
}

// getLatitudeDeg returns the latitude in degrees.
func getLatitudeDeg(v Vector3D) float64 {
	return getLatitude(v) * 180.0 / math.Pi
}
