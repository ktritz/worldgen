# Macroscopic Coastline Complexity Parameters

## Overview
The macroscopic coastline complexity system provides tessellation-independent measurement of coastline realism through geometric analysis of tortuosity, direction changes, and curvature variation. This document details the parameters that control complexity and their effects on realism.

## Core Algorithm Components

### 1. Complexity Measurement Formula
```
Complexity = 0.4 × Tortuosity + 0.3 × Direction Changes + 0.3 × Curvature Variation
```

**Target Scores:**
- **Realistic coastlines**: 0.60+ (target for natural appearance)
- **Current achievement**: 0.458/1.0 (good but can be improved)
- **Voronoi-limited baseline**: 0.33/1.0 (geometric constraint)

### 2. Geometric Analysis Components

#### Tortuosity (40% weight)
- **Measurement**: Actual path length ÷ straight-line distance
- **Effect**: Higher values create more winding, fjord-like coastlines
- **Realistic range**: 1.5-3.0 (straight line = 1.0)

#### Direction Changes (30% weight)
- **Measurement**: Average angular deviation between path segments
- **Effect**: Higher values create more irregular, turning coastlines
- **Realistic range**: 0.3-0.8 radians average change

#### Curvature Variation (30% weight)
- **Measurement**: Standard deviation of local curvature along path
- **Effect**: Higher values create varied coastal features (bays, peninsulas)
- **Realistic range**: 0.2-0.6 standard deviation

## Noise Configuration Parameters

### Primary Parameters (CoastlineComplexityConfig)

#### Frequency Settings
```go
Frequency:        12.0   // Base noise frequency for detail level
WarpFrequency:    5.0    // Domain warp frequency multiplier
```

**Effect on Complexity:**
- **Higher Frequency (15.0+)**: More fine-scale coastal detail, increases direction changes
- **Lower Frequency (8.0-)**: Broader coastal features, increases tortuosity
- **Optimal Range**: 10.0-15.0 for balanced complexity

#### Amplitude Settings
```go
Amplitude:        0.25   // Maximum displacement as fraction of planet radius
WarpStrength:     3.5    // Domain warp distortion strength
```

**Effect on Complexity:**
- **Higher Amplitude (0.30+)**: More aggressive coastline displacement, increases all metrics
- **Lower Amplitude (0.15-)**: Subtle coastal variations, may reduce complexity
- **Optimal Range**: 0.20-0.30 for realistic displacement

#### Octave Configuration
```go
Octaves:          8      // Number of noise layers for detail hierarchy
Lacunarity:       3.0    // Frequency scaling between octaves
Gain:             0.7    // Amplitude scaling between octaves
```

**Effect on Complexity:**
- **More Octaves (10+)**: Finer detail hierarchy, increases curvature variation
- **Higher Lacunarity (3.5+)**: Sharper feature contrast, increases direction changes
- **Higher Gain (0.8+)**: Stronger detail persistence, increases overall complexity

#### Turbulence Parameters
```go
TurbulenceRough:  0.98   // Turbulence roughness (0-1)
TurbulenceWarp:   0.9    // Turbulent warp strength
```

**Effect on Complexity:**
- **Higher Roughness (0.95+)**: More chaotic coastal patterns, increases all metrics
- **Higher Warp (0.85+)**: More distorted coastlines, particularly affects direction changes

### Secondary Parameters

#### Boundary Zone Configuration
```go
BoundaryWidth:    0.12   // Fraction of sphere affected by enhancement
BlendFactor:      0.99   // Noise application strength (0-1)
FalloffType:      "cosine" // Falloff curve type
```

**Effect on Complexity:**
- **Wider Boundary (0.15+)**: More cells affected, potentially higher complexity
- **Higher Blend Factor (0.95+)**: Stronger noise application, increases all metrics
- **Cosine Falloff**: Smoother transitions vs linear/smooth for natural appearance

#### Multi-Scale Hierarchy
```go
// 5-scale noise combination in createFractalIslandPattern
MacroScale:   35% weight  // Major peninsulas (high tortuosity)
LargeScale:   25% weight  // Regional features (direction changes)
MediumScale:  20% weight  // Local inlets (curvature variation)
FineScale:    15% weight  // Detailed features
MicroScale:   5% weight   // Surface irregularities
```

## Parameter Tuning Guidelines

### To Increase Tortuosity (Winding Coastlines)
1. **Increase MacroScale weight** in fractal combination (35% → 40%)
2. **Reduce Frequency** for broader features (12.0 → 10.0)
3. **Increase Amplitude** for stronger displacement (0.25 → 0.30)
4. **Lower Lacunarity** for smoother scaling (3.0 → 2.5)

### To Increase Direction Changes (Angular Coastlines)
1. **Increase LargeScale weight** in fractal combination (25% → 30%)
2. **Increase WarpStrength** for more distortion (3.5 → 4.0)
3. **Higher TurbulenceRough** for chaotic patterns (0.98 → 0.99)
4. **Increase Octaves** for more detail layers (8 → 10)

### To Increase Curvature Variation (Diverse Features)
1. **Increase MediumScale weight** in fractal combination (20% → 25%)
2. **Higher Lacunarity** for feature contrast (3.0 → 3.5)
3. **Increase TurbulenceWarp** for local distortion (0.9 → 0.95)
4. **Higher Gain** for detail persistence (0.7 → 0.8)

## Realistic Configuration Examples

### Conservative Realism (0.40-0.50 complexity)
```go
Frequency: 10.0, Amplitude: 0.20, Octaves: 6
WarpStrength: 2.5, Lacunarity: 2.5, Gain: 0.6
TurbulenceRough: 0.90, BoundaryWidth: 0.10
```

### Balanced Realism (0.45-0.55 complexity)
```go
Frequency: 12.0, Amplitude: 0.25, Octaves: 8
WarpStrength: 3.5, Lacunarity: 3.0, Gain: 0.7
TurbulenceRough: 0.98, BoundaryWidth: 0.12
```

### Aggressive Realism (0.55-0.65 complexity)
```go
Frequency: 15.0, Amplitude: 0.30, Octaves: 10
WarpStrength: 4.0, Lacunarity: 3.5, Gain: 0.8
TurbulenceRough: 0.99, BoundaryWidth: 0.15
```

## Performance Considerations

### Computational Cost Factors
1. **Octaves**: Linear scaling - each octave doubles computation time
2. **BoundaryWidth**: Quadratic scaling - wider zones exponentially increase processing
3. **Multi-scale hierarchy**: 5x noise evaluations per cell in boundary zone
4. **Path analysis**: Scales with coastline length and connectivity

### Optimization Strategies
1. **Adaptive octaves**: Use fewer octaves for larger plates
2. **Distance-based falloff**: Reduce computation far from boundaries
3. **Path simplification**: Limit analysis to significant coastline segments
4. **Caching**: Store noise values for repeated evaluations

## Validation and Tuning

### Testing Methodology
1. **Run multiple seeds** (5-10) to average complexity scores
2. **Compare to Earth benchmarks** - real coastlines score 0.65-0.85
3. **Visual validation** through 3D visualization
4. **Performance profiling** for acceptable generation times

### Target Achievements
- **Macroscopic complexity**: 0.60+ for realistic appearance
- **Boundary modification rate**: 8-15% of boundary cells
- **General boundary complexity**: 0.80+ overall system score
- **Performance**: <5 seconds for subdivision 6 generation

## Current Status
- **Implementation**: Complete in `boundary_noise.go`
- **Integration**: Phase 9.5 of tectonic generation pipeline
- **Current score**: 0.458/1.0 (good progress toward 0.60+ target)
- **Modification rate**: 8.7% of boundary cells successfully enhanced
- **System stability**: Comprehensive bounds checking prevents crashes