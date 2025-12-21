# Elevation Generation System API

## Overview

The elevation generation system provides comprehensive, Earth-realistic terrain generation through a modular architecture. The system integrates tectonic processes, volcanic activity, seafloor spreading, erosion, and fractal noise to create detailed planetary elevation maps.

## Core Architecture

### Main Generation Function

```go
func GenerateElevation(
    icosphereSites []icosphere.Vector3D,
    tectonicData *tectonics.TectonicsData,
    settings ElevationSettings,
    globalSeed int64,
    planetRadius float64,
) (*ElevationData, error)
```

**Purpose**: Orchestrates the complete elevation generation process using all available modules.

**Process Flow**:
1. Base elevation from plate types
2. Volcanic elevation from hotspots and features
3. Seafloor elevation from age-depth relationships
4. Ridge elevation from mid-ocean ridge topography
5. Tectonic elevation from boundary interactions
6. Erosion effects from age-based weathering
7. Fractal noise for terrain detail
8. Component combination and validation

## Module Components

### 1. Base Elevation (`base_elevation.go`)

Generates fundamental elevation differences between continental and oceanic plates.

#### Key Functions

```go
func GenerateBaseElevations(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
```

**Continental Plates**: 400-600m base elevation with ±800-1200m variation
**Oceanic Plates**: -3800m mean depth with ±1500m variation

#### Validation

```go
func ValidateBaseElevations(baseElevations []float64, tectonicData *TectonicsData) (BaseElevationMetrics, []string)
```

Returns statistics and warnings about plate type elevation consistency.

### 2. Volcanic Elevation (`volcanic_elevation.go`)

Models elevation effects from hotspots, volcanic features, and volcanic arcs.

#### Key Functions

```go
func GenerateVolcanicElevations(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
func CalculateHotspotElevationEffects(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
func CalculateVolcanicFeatureElevations(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
```

**Features**:
- Thermal uplift from active hotspots (up to 800m × intensity)
- Volcanic feature profiles (shield volcanoes, seamounts, volcanic arcs)
- Age-based erosion and decay
- Distance-based falloff effects

#### Validation

```go
func ValidateVolcanicElevations(volcanicElevations []float64, tectonicData *TectonicsData) (VolcanicElevationMetrics, []string)
```

### 3. Seafloor Elevation (`seafloor_elevation.go`)

Implements age-depth relationships and seafloor spreading effects.

#### Key Functions

```go
func GenerateSeafloorElevations(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
func CalculateSeafloorSpreadingEffects(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
```

**Age-Depth Models**:
- Oceanic crust: Exponential age-depth relationship
- Continental crust: Thermal subsidence (much reduced)
- Ridge axis effects with spreading rate scaling

#### Age-Depth Validation

```go
func CalculateAgeDepthValidation(seafloorAges []float64, elevations []float64, model SeafloorAgeModel) AgeDepthValidation
```

### 4. Ridge Elevation (`ridge_elevation.go`)

Models mid-ocean ridge topography and morphology.

#### Key Functions

```go
func GenerateRidgeElevations(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
func CalculateRidgeAxisTopography(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []RidgeAxisPoint
```

**Ridge Types**:
- **Fast-spreading** (>50 mm/yr): Broad, gentle profiles (East Pacific Rise)
- **Intermediate-spreading** (20-50 mm/yr): Moderate profiles with roughness
- **Slow-spreading** (<20 mm/yr): Narrow, steep profiles with rift valleys (Mid-Atlantic Ridge)

#### Ridge Analysis

```go
func CalculateRidgeSegmentation(ridges []MidOceanRidge, planetRadius float64) []RidgeSegment
func CalculateRidgeAsymmetry(icosphereSites []Vector3D, ridgeElevations []float64, tectonicData *TectonicsData, params ElevationParameters) []RidgeAsymmetry
```

### 5. Tectonic Elevation (`tectonic_elevation.go`)

Handles elevation effects from tectonic boundary interactions and mountain building.

#### Key Functions

```go
func GenerateTectonicElevations(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
func CalculateSubductionZoneEffects(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
```

**Boundary Effects**:
- **Convergent**: Mountain building (1500m+ scaled by velocity)
- **Divergent**: Continental rifts (-800m valleys) or oceanic spreading
- **Transform**: Linear valleys and scarps (-200m fault zones)

#### Subduction Zones

```go
func calculateSubductionTopography(distance float64, subZone SubductionZone, params ElevationParameters) float64
```

**Features**:
- Ocean trenches (-8000m × multiplier)
- Volcanic arcs (100-300km behind trench, 1500m elevation)

### 6. Erosion Modeling (`erosion_modeling.go`)

Applies age-based erosion and weathering effects.

#### Key Functions

```go
func GenerateErosionEffects(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
func CalculateDifferentialErosion(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
func CalculateGlacialErosion(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
```

**Erosion Rates**:
- **Young continental** (0-50 Myr): 80 mm/kyr
- **Mature continental** (50-200 Myr): 50 mm/kyr
- **Old continental** (200-1000 Myr): 20 mm/kyr
- **Ancient cratonic** (>1000 Myr): 10 mm/kyr

#### Rock Types

```go
type RockType string
const (
    Igneous     RockType = "igneous"     // 2.0× resistance
    Metamorphic RockType = "metamorphic" // 1.5× resistance
    Sedimentary RockType = "sedimentary" // 1.0× resistance
)
```

### 7. Elevation Noise (`elevation_noise.go`)

Adds fractal noise for realistic terrain detail.

#### Key Functions

```go
func GenerateElevationNoise(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
func CombineNoiseTypes(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64
```

**Noise Types**:
- **Fractal/Perlin**: General terrain variation
- **Ridged Multifractal**: Mountain ranges and ridges
- **Billow**: Rolling hills and terrain
- **Turbulence**: Fine-scale detail

#### Noise Mixing

```go
type NoiseMixingWeights struct {
    FractalWeight    float64 // General terrain
    RidgedWeight     float64 // Mountain ranges
    BillowWeight     float64 // Rolling hills
    TurbulenceWeight float64 // Fine detail
}
```

### 8. Elevation Validation (`elevation_validation.go`)

Comprehensive validation against Earth metrics and physical realism.

#### Main Validation

```go
func ValidateElevationSystem(elevationData *ElevationData, tectonicData *TectonicsData, params ElevationParameters) ElevationValidationReport
```

#### Validation Components

**Basic Statistics**:
- Elevation range, mean, standard deviation
- Land/ocean percentage and mean elevations

**Earth Comparison**:
- Land percentage ratio (target: 29.2%)
- Mean land elevation ratio (target: 840m)
- Mean ocean depth ratio (target: -3688m)
- Hypsometric curve analysis

**Tectonic Consistency**:
- Continental vs oceanic elevation contrast
- Boundary effect validation
- Plate type consistency scoring

**Physical Realism**:
- Extreme value detection
- Gradient steepness analysis
- Isostatic balance approximation

## Data Structures

### ElevationData

```go
type ElevationData struct {
    // Legacy compatibility
    CellElevations map[int32]float64 `json:"cell_elevations"`
    
    // Component breakdowns
    BaseElevations     []float64 `json:"base_elevations"`
    VolcanicElevations []float64 `json:"volcanic_elevations"`
    SeafloorElevations []float64 `json:"seafloor_elevations"`
    RidgeElevations    []float64 `json:"ridge_elevations"`
    TectonicElevations []float64 `json:"tectonic_elevations"`
    ErosionEffects     []float64 `json:"erosion_effects"`
    NoiseElevations    []float64 `json:"noise_elevations"`
    FinalElevations    []float64 `json:"final_elevations"`
}
```

### ElevationSettings

```go
type ElevationSettings struct {
    // Basic parameters
    ElevationMultiplier float64 `json:"elevation_multiplier"`
    
    // Tectonic parameters
    EnableVolcanicElevation      bool    `json:"enable_volcanic_elevation"`
    EnableRidgeTopography        bool    `json:"enable_ridge_topography"`
    VolcanicMultiplier          float64 `json:"volcanic_multiplier"`
    
    // Boundary effects
    ConvergentBoundaryStrength  float64 `json:"convergent_boundary_strength"`
    DivergentBoundaryStrength   float64 `json:"divergent_boundary_strength"`
    CharacteristicFalloffDistance float64 `json:"characteristic_falloff_distance"`
    MaxBoundaryEffectDistance   float64 `json:"max_boundary_effect_distance"`
    
    // Seafloor parameters
    SeafloorModel SeafloorAgeModelType `json:"seafloor_model"`
    
    // Erosion parameters
    ErosionEfficiency       float64 `json:"erosion_efficiency"`
    EnableGlacialErosion    bool    `json:"enable_glacial_erosion"`
    GlacialErosionIntensity float64 `json:"glacial_erosion_intensity"`
    
    // Noise parameters
    EnableFractalNoise      bool    `json:"enable_fractal_noise"`
    NoiseAmplitude         float64 `json:"noise_amplitude"`
    NoiseFrequency         float64 `json:"noise_frequency"`
    NoiseOctaves           int     `json:"noise_octaves"`
    NoiseLacunarity        float64 `json:"noise_lacunarity"`
    NoisePersistence       float64 `json:"noise_persistence"`
    
    // Advanced noise parameters
    ContinentalNoiseMultiplier float64 `json:"continental_noise_multiplier"`
    OceanicNoiseMultiplier     float64 `json:"oceanic_noise_multiplier"`
    BoundaryNoiseMultiplier    float64 `json:"boundary_noise_multiplier"`
    
    // Ridged noise
    RidgedNoiseAmplitude   float64 `json:"ridged_noise_amplitude"`
    RidgedNoiseFrequency   float64 `json:"ridged_noise_frequency"`
    RidgedNoiseOctaves     int     `json:"ridged_noise_octaves"`
    RidgedNoiseLacunarity  float64 `json:"ridged_noise_lacunarity"`
    RidgedNoisePersistence float64 `json:"ridged_noise_persistence"`
    RidgedNoiseGain        float64 `json:"ridged_noise_gain"`
    
    // Billow noise
    BillowNoiseAmplitude   float64 `json:"billow_noise_amplitude"`
    BillowNoiseFrequency   float64 `json:"billow_noise_frequency"`
    BillowNoiseOctaves     int     `json:"billow_noise_octaves"`
    BillowNoiseLacunarity  float64 `json:"billow_noise_lacunarity"`
    BillowNoisePersistence float64 `json:"billow_noise_persistence"`
    
    // Turbulence
    TurbulenceAmplitude float64 `json:"turbulence_amplitude"`
    TurbulenceFrequency float64 `json:"turbulence_frequency"`
    
    // Detail noise
    EnableDetailNoise    bool    `json:"enable_detail_noise"`
    DetailNoiseAmplitude float64 `json:"detail_noise_amplitude"`
    DetailNoiseFrequency float64 `json:"detail_noise_frequency"`
    
    // Subduction zones
    TrenchDepthMultiplier float64 `json:"trench_depth_multiplier"`
    ArcElevation         float64 `json:"arc_elevation"`
    
    // Sedimentation
    SedimentationEfficiency float64 `json:"sedimentation_efficiency"`
    
    // Ridge parameters
    RidgeElevation        float64 `json:"ridge_elevation"`
    RidgeInfluenceDistance float64 `json:"ridge_influence_distance"`
    
    // Hotspot parameters
    HotspotInfluenceDistance float64 `json:"hotspot_influence_distance"`
}
```

## Usage Examples

### Basic Elevation Generation

```go
// Configure elevation settings
settings := ElevationSettings{
    ElevationMultiplier:        1.0,
    EnableVolcanicElevation:    true,
    EnableRidgeTopography:      true,
    EnableFractalNoise:         true,
    ConvergentBoundaryStrength: 2000.0,
    DivergentBoundaryStrength:  500.0,
    NoiseAmplitude:            100.0,
    NoiseOctaves:              6,
}

// Generate elevation data
elevationData, err := GenerateElevation(
    icosphereSites,
    tectonicData,
    settings,
    globalSeed,
    planetRadius,
)

if err != nil {
    return err
}

// Access final elevations
finalElevations := elevationData.FinalElevations
```

### Component Analysis

```go
// Analyze individual components
baseContribution := calculateMeanAbsolute(elevationData.BaseElevations)
volcanicContribution := calculateMeanAbsolute(elevationData.VolcanicElevations)
tectonicContribution := calculateMeanAbsolute(elevationData.TectonicElevations)

fmt.Printf("Base: %.1fm, Volcanic: %.1fm, Tectonic: %.1fm\n",
    baseContribution, volcanicContribution, tectonicContribution)
```

### Custom Validation

```go
// Run comprehensive validation
validationReport := ValidateElevationSystem(elevationData, tectonicData, params)

// Check quality score
if validationReport.OverallQuality < 0.7 {
    fmt.Println("Elevation quality below threshold")
    for _, warning := range validationReport.Warnings {
        fmt.Printf("Warning: %s\n", warning)
    }
}

// Earth comparison metrics
earthComparison := validationReport.EarthComparison
fmt.Printf("Land percentage fit: %.2f\n", earthComparison.LandPercentageFit)
fmt.Printf("Elevation range fit: %.2f\n", earthComparison.ElevationRangeFit)
```

## Default Parameter Values

| Parameter | Default Value | Description |
|-----------|---------------|-------------|
| `ElevationMultiplier` | 1.0 | Global elevation scaling factor |
| `ConvergentBoundaryStrength` | 2000.0m | Mountain building strength |
| `DivergentBoundaryStrength` | 500.0m | Rift/spreading strength |
| `VolcanicMultiplier` | 1.0 | Volcanic feature scaling |
| `RidgeElevation` | 2500.0m | Base ridge elevation |
| `NoiseAmplitude` | 100.0m | Fractal noise amplitude |
| `NoiseOctaves` | 6 | Number of noise octaves |
| `ErosionEfficiency` | 1.0 | Erosion rate multiplier |
| `TrenchDepthMultiplier` | 1.0 | Subduction trench depth scaling |
| `ArcElevation` | 1500.0m | Volcanic arc elevation |

## Earth Reference Values

| Metric | Earth Value | Purpose |
|--------|-------------|---------|
| Mean land elevation | 840m | Continental elevation validation |
| Mean ocean depth | -3688m | Oceanic depth validation |
| Land percentage | 29.2% | Surface distribution validation |
| Highest point | 8848m (Everest) | Maximum elevation reference |
| Lowest point | -11034m (Challenger Deep) | Minimum depth reference |
| Hypsometric integral | ~0.5 | Elevation distribution shape |

## Performance Considerations

- **Modular Design**: Each component can be enabled/disabled independently
- **Validation**: Comprehensive but can be skipped for performance
- **Noise Generation**: Most computationally expensive component
- **Memory Usage**: Stores all component arrays for analysis

## Integration Notes

The elevation system integrates with:
- **Tectonics Module**: Requires complete tectonic data including plates, boundaries, hotspots, and ridges
- **Icosphere Module**: Uses spherical geometry for distance calculations
- **Visualization**: Provides detailed component breakdowns for analysis
- **Export Systems**: Compatible with standard elevation data formats

## Validation and Quality Assurance

The system includes comprehensive validation:
- **Earth Similarity**: Compares to real Earth metrics
- **Tectonic Consistency**: Validates geological relationships
- **Physical Realism**: Checks for impossible values
- **Component Balance**: Ensures reasonable contribution ratios

Quality scores range from 0.0 to 1.0, with values above 0.7 indicating good Earth-realistic results.