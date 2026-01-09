# Temperature Implementation Plan

## Overview

Implement an energy balance model (EBM) for surface temperature following the patterns established in wind.go and currents.go. The system will produce realistic temperature distributions based on solar insolation, heat transport (wind + currents), and surface properties.

## File Structure

Following the established pattern of ~400-500 line files with clear separation of concerns:

```
climgen/
├── temperature.go              # Settings, result types, main entry point (~450 lines)
├── temperature_solar.go        # Solar insolation and albedo (~250 lines)
├── temperature_transport.go    # Heat diffusion and advection (~350 lines)
├── temperature_balance.go      # Energy balance iteration (~300 lines)
└── temperature_visualization.go # Rendering (~250 lines)
```

## Detailed File Contents

### 1. temperature.go (~450 lines)

**Constants:**
```go
// Physical constants (from Python reference)
SolarConstant          = 1361.0    // W/m²
StefanBoltzmann        = 5.67e-8   // W/m²/K⁴
LapseRate              = 0.0065    // K/m (6.5°C per 1000m)
FreezingPoint          = 273.15    // K

// Albedo values
AlbedoIce              = 0.45
AlbedoLand             = 0.20
AlbedoWater            = 0.08

// Heat capacity (J/m²/K)
HeatCapacityLand       = 1.0e5
HeatCapacityWater      = 4.2e6

// Emissivity
EmissivityLand         = 0.82
EmissivityWater        = 0.96

// Atmospheric transmissivity (greenhouse effect)
AtmosphericTransmissivity = 0.75

// Default iteration parameters
DefaultMaxIterations   = 5000
DefaultTolerance       = 0.01      // K
DefaultTimeStep        = 1800.0    // seconds (30 min)
```

**Settings structs:**
```go
// SolarSettings - controls insolation calculation
type SolarSettings struct {
    SolarLuminosity float64  // Multiplier for solar constant (1.0 = Earth)
    AxialTilt       float64  // Degrees (23.5 for Earth) - for seasonal variation
    SeasonPhase     float64  // 0-1, position in year (0 = northern winter solstice)
    Verbose         bool
}

// TransportSettings - controls heat diffusion and advection
type TransportSettings struct {
    DiffusionLand       float64  // Base diffusion rate over land
    DiffusionWater      float64  // Base diffusion rate over water
    WindTransportScale  float64  // How much wind affects heat transport (0-1)
    CurrentTransportScale float64 // How much currents affect heat transport (0-1)
    Verbose             bool
}

// BalanceSettings - controls energy balance iteration
type BalanceSettings struct {
    MaxIterations    int
    Tolerance        float64  // Convergence threshold in K
    TimeStep         float64  // Seconds per iteration
    IceAlbedoFeedback bool    // Enable ice-albedo feedback
    Verbose          bool
}

// TemperatureSettings - composite settings
type TemperatureSettings struct {
    Seed       int64
    Solar      SolarSettings
    Transport  TransportSettings
    Balance    BalanceSettings
    Verbose    bool
}
```

**Result struct:**
```go
type TemperatureResult struct {
    // Core outputs
    Temperature      []float64  // Surface temperature in Kelvin
    TemperatureCelsius []float64 // Convenience: T - 273.15

    // Diagnostic layers (for visualization/debugging)
    Insolation       []float64  // Absorbed solar radiation (W/m²)
    OutgoingLongwave []float64  // Emitted radiation (W/m²)
    Albedo           []float64  // Effective albedo at each vertex
    NetRadiation     []float64  // ASR - OLR at convergence
    HeatTransport    []float64  // Net heat transported to each vertex

    // Convergence info
    Iterations       int
    FinalMaxDelta    float64
}
```

**Main entry point:**
```go
func GenerateTemperature(
    vertices []Vector3D,
    elevation []float64,
    seaLevelThreshold float64,
    adj *FlatAdjacency,
    wind *WindResult,           // Optional, can be nil
    currents *OceanCurrentResult, // Optional, can be nil
    settings TemperatureSettings,
) (*TemperatureResult, error)
```

**Pipeline steps:**
1. Validate settings
2. Compute solar insolation (latitude-dependent, optional seasonal)
3. Initialize temperature field (start at 288K = 15°C)
4. Iterate energy balance until convergence:
   - Compute albedo (ice/land/water, with ice-albedo feedback)
   - Compute absorbed solar radiation (ASR)
   - Compute outgoing longwave radiation (OLR)
   - Compute heat transport (diffusion + wind/current advection)
   - Update temperature: dT = dt * (ASR - OLR) / Cp + transport
   - Check convergence
5. Apply elevation lapse rate correction
6. Return result

### 2. temperature_solar.go (~250 lines)

**Functions:**
```go
// ComputeInsolation calculates latitude-dependent solar radiation
// Uses the Legendre polynomial approximation from the Python model:
//   s(lat) = 1 - 0.48 * P2(sin(lat))
//   where P2(x) = 0.5 * (3x² - 1)
func ComputeInsolation(
    vertices []Vector3D,
    settings SolarSettings,
) []float64

// ComputeSeasonalInsolation adds seasonal variation based on axial tilt
// For each vertex: adjust solar angle based on declination
func ComputeSeasonalInsolation(
    vertices []Vector3D,
    settings SolarSettings,
) []float64

// ComputeAlbedo returns albedo for each vertex based on surface type
// Ice (T < 273.15K), Land, or Water
func ComputeAlbedo(
    temperature []float64,
    elevation []float64,
    seaLevelThreshold float64,
    settings SolarSettings,
) []float64

// ComputeAbsorbedSolar returns ASR = qs * (1 - albedo)
func ComputeAbsorbedSolar(
    insolation []float64,
    albedo []float64,
) []float64
```

### 3. temperature_transport.go (~350 lines)

**Functions:**
```go
// ComputeHeatDiffusion performs neighbor-weighted heat spreading
// Returns heat flux INTO each vertex (positive = warming)
// D is higher over water (ocean mixing) than land
func ComputeHeatDiffusion(
    temperature []float64,
    vertices []Vector3D,
    elevation []float64,
    seaLevelThreshold float64,
    adj *FlatAdjacency,
    settings TransportSettings,
) []float64

// ComputeWindAdvection weights heat transport by wind direction
// Heat flows preferentially in the wind direction
// Upwind neighbors contribute more to a vertex's temperature
func ComputeWindAdvection(
    temperature []float64,
    vertices []Vector3D,
    wind []Vector3D,
    adj *FlatAdjacency,
    settings TransportSettings,
) []float64

// ComputeCurrentAdvection computes heat transport by ocean currents
// Poleward currents carry warm water (warming effect)
// Equatorward currents carry cold water (cooling effect)
// Returns temperature anomaly for ocean vertices
func ComputeCurrentAdvection(
    temperature []float64,
    vertices []Vector3D,
    elevation []float64,
    seaLevelThreshold float64,
    currents []Vector3D,
    adj *FlatAdjacency,
    settings TransportSettings,
) []float64

// ComputeTotalHeatTransport combines diffusion + advection
func ComputeTotalHeatTransport(
    temperature []float64,
    vertices []Vector3D,
    elevation []float64,
    seaLevelThreshold float64,
    adj *FlatAdjacency,
    wind []Vector3D,      // nil if no wind
    currents []Vector3D,  // nil if no currents
    settings TransportSettings,
) []float64
```

### 4. temperature_balance.go (~300 lines)

**Functions:**
```go
// ComputeOutgoingLongwave returns OLR = emissivity * σ * T⁴ * transmissivity
func ComputeOutgoingLongwave(
    temperature []float64,
    elevation []float64,
    seaLevelThreshold float64,
) []float64

// ComputeHeatCapacity returns effective heat capacity for each vertex
// Ocean has ~42x higher heat capacity than land
func ComputeHeatCapacity(
    elevation []float64,
    seaLevelThreshold float64,
) []float64

// IterateEnergyBalance performs one iteration of the EBM
// Returns updated temperature and max change
func IterateEnergyBalance(
    temperature []float64,
    insolation []float64,
    vertices []Vector3D,
    elevation []float64,
    seaLevelThreshold float64,
    adj *FlatAdjacency,
    wind []Vector3D,
    currents []Vector3D,
    settings TemperatureSettings,
) (newTemp []float64, maxDelta float64)

// SolveEnergyBalance iterates until convergence
// Returns final temperature and convergence info
func SolveEnergyBalance(
    insolation []float64,
    vertices []Vector3D,
    elevation []float64,
    seaLevelThreshold float64,
    adj *FlatAdjacency,
    wind []Vector3D,
    currents []Vector3D,
    settings TemperatureSettings,
) (temperature []float64, iterations int, finalDelta float64)

// ApplyLapseRateCorrection adjusts temperature for elevation
// T_surface = T_sea_level - LapseRate * elevation
// Only applies to land (elevation > 0)
func ApplyLapseRateCorrection(
    temperature []float64,
    elevation []float64,
) []float64
```

### 5. temperature_visualization.go (~250 lines)

**Settings:**
```go
type TemperatureVisualizationSettings struct {
    Width          int
    Height         int
    MinTempC       float64  // Color scale minimum (°C)
    MaxTempC       float64  // Color scale maximum (°C)
    ShowIsotherms  bool     // Draw temperature contour lines
    IsothermStep   float64  // Degrees between isotherms
    ShowElevation  bool     // Blend elevation into land colors
}
```

**Functions:**
```go
// RenderTemperatureMap creates an equirectangular temperature map
// Uses a blue-white-red color gradient
func RenderTemperatureMap(
    vertices []Vector3D,
    elevation []float64,
    result *TemperatureResult,
    outputPath string,
    settings TemperatureVisualizationSettings,
) error

// temperatureToColor maps Celsius temperature to color
// Cold (blue) -> temperate (green/white) -> hot (red)
func temperatureToColor(tempC, minC, maxC float64) color.RGBA

// drawIsotherms overlays temperature contour lines
func drawIsotherms(img *image.RGBA, ...)
```

## Implementation Order

1. **temperature.go** - Settings, constants, result types, main entry point skeleton
2. **temperature_solar.go** - Insolation calculation (can test independently)
3. **temperature_balance.go** - OLR, heat capacity, basic iteration (no transport)
4. **temperature_transport.go** - Diffusion first, then wind/current advection
5. **temperature_visualization.go** - Rendering for validation
6. **cmd/test_temperature/main.go** - Test harness

## Testing Milestones

1. **Solar only**: Should show latitude gradient (hot equator, cold poles)
2. **+ Lapse rate**: Mountains should be cold
3. **+ Diffusion**: Should smooth out sharp gradients
4. **+ Wind transport**: Trade winds should warm western coasts
5. **+ Ocean currents**: Gulf Stream warming, California Current cooling
6. **+ Ice feedback**: Polar regions should develop ice caps

## Expected Output Ranges

For Earth-like parameters:
- Equatorial ocean: ~25-30°C
- Temperate ocean: ~10-20°C
- Polar ocean: ~-2 to 5°C
- Tropical land: ~25-35°C
- Temperate land: ~5-20°C
- Polar land: ~-30 to -10°C
- High mountains: ~-20 to 0°C

## Dependencies

- `types.go`: Vector3D, FlatAdjacency, helper functions
- `wind.go`: WindResult (optional input)
- `currents.go`: OceanCurrentResult (optional input)

## Helpers to Add to types.go or Create

Consider adding to avoid duplication:
- `estimateCellSize()` already exists in currents.go - may need to move to types.go
- Color interpolation helpers may be shared with existing visualization files

## Notes

- Start with steady-state (iterate to equilibrium), not time-stepping
- No seasonal variation in initial implementation
- Use same spatial index pattern as wind_visualization.go for rendering
- Wind and currents are optional inputs - system should work with just elevation
