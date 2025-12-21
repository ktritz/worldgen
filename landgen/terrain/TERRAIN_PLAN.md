# Simplified Terrain Generation Plan

## Problem Statement

The current approach has two fundamental issues:

1. **Straight boundaries**: Voronoi-based plate boundaries follow cell edges, producing unnaturally straight coastlines and plate edges
2. **Complex module interactions**: 7 elevation modules (base, volcanic, seafloor, ridge, tectonic, erosion, noise) create a calibration nightmare where changes cascade unpredictably

## New Approach: Noise-First with Domain Warping

Instead of: `plates → boundaries → elevation → noise details`

We use: `warped noise → elevation → threshold coastlines`

### Core Insight

Earth's hypsometric curve (cumulative elevation distribution) has a distinctive bimodal shape:
- ~30% land clustered around 0-1000m
- ~70% ocean clustered around -4000m
- Sharp transition at coastlines

We can **directly target this distribution** rather than trying to emergently produce it from 7 interacting modules.

---

## Automated Evaluation Metrics

All terrain generation must be evaluated against these metrics automatically. Each metric has a target value and acceptable range based on Earth data.

### Primary Metrics (Must Pass)

| Metric | Earth Value | Target Range | Weight |
|--------|-------------|--------------|--------|
| Land Coverage | 29.2% | 27-32% | 20% |
| Ocean Coverage | 70.8% | 68-73% | 20% |
| Mean Land Elevation | 840m | 600-1000m | 15% |
| Mean Ocean Depth | -3688m | -4000 to -3400m | 15% |
| Mountain Coverage (>3000m) | 2.0% | 1.5-3.0% | 10% |
| Deep Ocean Coverage (<-5000m) | 2.5% | 1.5-4.0% | 10% |
| Continental Shelf (0 to -200m) | 8.0% | 5-12% | 10% |

### Hypsometric Curve Targets

Earth's cumulative elevation distribution at key thresholds:

| Elevation Threshold | Earth Cumulative % | Target Range |
|--------------------|-------------------|--------------|
| Below -6000m | 1.2% | 0.5-2.5% |
| Below -5000m | 2.5% | 1.5-4.0% |
| Below -4000m | 52.4% | 48-58% |
| Below -3000m | 68.2% | 63-73% |
| Below 0m (sea level) | 70.8% | 68-73% |
| Below 500m | 82.5% | 78-87% |
| Below 1000m | 90.2% | 86-94% |
| Below 2000m | 96.8% | 94-99% |
| Below 3000m | 98.0% | 96-99.5% |

### Coastline Irregularity Metrics

| Metric | Earth Typical | Target Range | Description |
|--------|---------------|--------------|-------------|
| Fractal Dimension | 1.2-1.4 | 1.15-1.50 | Coastline complexity (1.0 = smooth, 2.0 = space-filling) |
| Tortuosity Ratio | 2.5-4.0 | 2.0-5.0 | Actual length / straight-line distance |
| Coastline Vertices/km | 0.5-2.0 | 0.3-3.0 | Boundary detail density |

### Continental Distribution Metrics

| Metric | Earth Value | Target Range | Description |
|--------|-------------|--------------|-------------|
| Number of Major Landmasses | 6-7 | 4-10 | Continents > 5% of land area |
| Largest Continent % | 30% (Eurasia) | 20-45% | Prevents single supercontinent |
| Smallest Major Continent % | 5% (Australia) | 3-10% | Ensures variety |
| Continent Gini Coefficient | 0.45 | 0.30-0.60 | Size inequality measure |
| Archipelago Coverage | 3-5% | 1-8% | Small islands as % of land |

### Elevation Statistics

| Statistic | Earth Value | Target Range |
|-----------|-------------|--------------|
| Global Mean | -2,430m | -2800 to -2000m |
| Global Std Dev | 2,500m | 2000-3000m |
| Maximum Elevation | 8,848m | 6000-10000m |
| Minimum Elevation | -10,994m | -12000 to -8000m |
| Bimodal Peak 1 (ocean) | -4,200m | -4500 to -3800m |
| Bimodal Peak 2 (land) | +300m | 0 to 600m |

---

## Scoring Function

```go
type TerrainMetrics struct {
    // Primary coverage metrics
    LandCoverage      float64
    OceanCoverage     float64
    MountainCoverage  float64
    DeepOceanCoverage float64
    ShelfCoverage     float64

    // Elevation statistics
    MeanLandElevation  float64
    MeanOceanDepth     float64
    GlobalMean         float64
    GlobalStdDev       float64
    MaxElevation       float64
    MinElevation       float64

    // Hypsometric curve (cumulative % at thresholds)
    HypsometricCurve   map[float64]float64  // elevation -> cumulative %

    // Coastline metrics
    FractalDimension   float64
    TortuosityRatio    float64

    // Continental distribution
    NumMajorLandmasses int
    LargestContinentPct float64
    ContinentGini      float64
}

// EvaluateTerrain returns a score from 0-100 based on Earth similarity
func EvaluateTerrain(sites []Vector3D, elevation []float64) (float64, TerrainMetrics) {
    metrics := computeMetrics(sites, elevation)
    score := 0.0
    maxScore := 0.0

    // Primary metrics (60% of score)
    score += scoreInRange(metrics.LandCoverage, 0.27, 0.32, 0.292) * 12
    score += scoreInRange(metrics.OceanCoverage, 0.68, 0.73, 0.708) * 12
    score += scoreInRange(metrics.MeanLandElevation, 600, 1000, 840) * 9
    score += scoreInRange(metrics.MeanOceanDepth, -4000, -3400, -3688) * 9
    score += scoreInRange(metrics.MountainCoverage, 0.015, 0.030, 0.020) * 6
    score += scoreInRange(metrics.DeepOceanCoverage, 0.015, 0.040, 0.025) * 6
    score += scoreInRange(metrics.ShelfCoverage, 0.05, 0.12, 0.08) * 6
    maxScore += 60

    // Hypsometric curve matching (25% of score)
    hypsTargets := map[float64]float64{
        -6000: 0.012, -5000: 0.025, -4000: 0.524, -3000: 0.682,
        0: 0.708, 500: 0.825, 1000: 0.902, 2000: 0.968, 3000: 0.980,
    }
    for elev, target := range hypsTargets {
        actual := metrics.HypsometricCurve[elev]
        tolerance := 0.05  // 5% tolerance
        score += scoreInRange(actual, target-tolerance, target+tolerance, target) * (25.0/9.0)
    }
    maxScore += 25

    // Coastline irregularity (10% of score)
    score += scoreInRange(metrics.FractalDimension, 1.15, 1.50, 1.30) * 5
    score += scoreInRange(metrics.TortuosityRatio, 2.0, 5.0, 3.0) * 5
    maxScore += 10

    // Continental distribution (5% of score)
    score += scoreInRange(float64(metrics.NumMajorLandmasses), 4, 10, 7) * 2.5
    score += scoreInRange(metrics.ContinentGini, 0.30, 0.60, 0.45) * 2.5
    maxScore += 5

    return (score / maxScore) * 100, metrics
}

// scoreInRange returns 0-1 based on how well value fits in [min, max] with ideal target
func scoreInRange(value, min, max, ideal float64) float64 {
    if value < min || value > max {
        // Outside range: score based on distance
        if value < min {
            return math.Max(0, 1 - (min-value)/(max-min))
        }
        return math.Max(0, 1 - (value-max)/(max-min))
    }
    // Inside range: full score, bonus for being near ideal
    distFromIdeal := math.Abs(value - ideal) / (max - min)
    return 1.0 - distFromIdeal*0.2  // Max 20% penalty for being at range edge
}
```

---

## Architecture: 3-Stage Pipeline

```
┌─────────────────────────────────────────────────────────────────┐
│  STAGE 1: Continental Mask (what is land vs ocean)              │
│  ─────────────────────────────────────────────────────────────  │
│  • Low-frequency noise defines continental shapes               │
│  • Domain warping creates irregular coastlines                  │
│  • Threshold tuned to achieve ~30% land coverage                │
│  METRICS: Land%, FractalDimension, NumContinents                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STAGE 2: Base Elevation (smooth continental/oceanic heights)   │
│  ─────────────────────────────────────────────────────────────  │
│  • Continental: base height ~800m                               │
│  • Oceanic: base depth ~-4000m                                  │
│  • Smooth transition at coastlines (shelf/slope)                │
│  • Mountain seeds from noise peaks on continents                │
│  METRICS: MeanLandElev, MeanOceanDepth, Bimodality              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STAGE 3: Terrain Detail (fractal roughness)                    │
│  ─────────────────────────────────────────────────────────────  │
│  • Multi-octave noise adds terrain texture                      │
│  • Amplitude varies by terrain type (mountains vs plains)       │
│  • Preserves hypsometric distribution                           │
│  METRICS: HypsometricCurve, StdDev, Min/Max                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  EVALUATION: Automatic Scoring                                  │
│  ─────────────────────────────────────────────────────────────  │
│  • Compute all metrics                                          │
│  • Compare to Earth targets                                     │
│  • Return score 0-100%                                          │
│  • Flag any metrics outside acceptable range                    │
└─────────────────────────────────────────────────────────────────┘
```

---

## Stage 1: Continental Mask with Domain Warping

### The Straight-Boundary Problem

Current approach:
```
Site A (plate 1) ──── Site B (plate 2)
        │
        └── boundary is a straight line between Voronoi cells
```

Domain warping solution:
```
Sample position P = (x, y, z) on sphere
Warped position P' = P + warp_offset(P)
Continental value = noise(P')
```

The warp offset uses curl noise or similar to displace the sampling position, creating organic, non-straight boundaries even though the underlying noise field is smooth.

### Algorithm

```go
func GenerateContinentalMask(sites []Vector3D, settings MaskSettings) []float64 {
    mask := make([]float64, len(sites))

    for i, site := range sites {
        // Apply domain warping to create irregular boundaries
        warped := applyDomainWarp(site, settings.WarpAmplitude, settings.WarpFrequency)

        // Sample continental noise at warped position
        // Low frequency = large continental shapes
        value := noise.Sample3D(warped.X, warped.Y, warped.Z, settings.ContinentalFrequency)

        // Apply FBM for multi-scale continental shapes
        value = fbm(warped, settings.Octaves, settings.Lacunarity, settings.Persistence)

        mask[i] = value
    }

    return mask
}

func applyDomainWarp(pos Vector3D, amplitude, frequency float64) Vector3D {
    // Use curl noise for divergence-free warping (no "sinks" or "sources")
    warpX := curlNoise.SampleX(pos.X*frequency, pos.Y*frequency, pos.Z*frequency)
    warpY := curlNoise.SampleY(pos.X*frequency, pos.Y*frequency, pos.Z*frequency)
    warpZ := curlNoise.SampleZ(pos.X*frequency, pos.Y*frequency, pos.Z*frequency)

    return Vector3D{
        X: pos.X + warpX*amplitude,
        Y: pos.Y + warpY*amplitude,
        Z: pos.Z + warpZ*amplitude,
    }.Normalize()  // Re-project onto sphere
}
```

### Key Parameters

| Parameter | Purpose | Typical Range | Affects Metric |
|-----------|---------|---------------|----------------|
| ContinentalFrequency | Size of continents | 1.0-3.0 | NumContinents |
| WarpAmplitude | Coastline irregularity | 0.1-0.5 | FractalDimension |
| WarpFrequency | Scale of coastline wiggles | 2.0-8.0 | Tortuosity |
| Threshold | Land/ocean cutoff | auto-tuned | LandCoverage |
| Octaves | Continental detail levels | 2-4 | ContinentGini |

---

## Stage 2: Base Elevation

Once we have the continental mask, elevation assignment is straightforward:

```go
func GenerateBaseElevation(sites []Vector3D, mask []float64, settings ElevationSettings) []float64 {
    elevation := make([]float64, len(sites))

    for i, site := range sites {
        if mask[i] > settings.LandThreshold {
            // Continental
            // Higher mask values = more inland = potentially higher elevation
            inlandness := (mask[i] - settings.LandThreshold) / (1.0 - settings.LandThreshold)
            elevation[i] = settings.ContinentalBase + inlandness*settings.ContinentalVariation
        } else {
            // Oceanic
            // Lower mask values = deeper ocean
            oceanDepth := (settings.LandThreshold - mask[i]) / (settings.LandThreshold + 1.0)
            elevation[i] = settings.OceanicBase - oceanDepth*settings.OceanicVariation
        }
    }

    // Add continental shelf transition
    addShelfTransition(elevation, mask, settings)

    return elevation
}
```

### Mountain Generation

Instead of complex tectonic boundaries, mountains emerge from:

1. **Noise peaks on continents**: Where continental noise is highest
2. **Secondary noise layer**: High-frequency "mountain potential" noise
3. **Combined threshold**: Only where both continental AND mountain noise are high

```go
func AddMountains(elevation []float64, sites []Vector3D, mask []float64, settings MountainSettings) {
    for i, site := range sites {
        if mask[i] < settings.LandThreshold {
            continue // Skip ocean
        }

        // Mountain potential from separate noise
        mountainNoise := mountainNoiseField.Sample3D(site, settings.MountainFrequency)

        // Only create mountains where potential is high AND we're on land
        if mountainNoise > settings.MountainThreshold {
            // Height based on how far above threshold
            peakFactor := (mountainNoise - settings.MountainThreshold) / (1.0 - settings.MountainThreshold)
            elevation[i] += peakFactor * settings.MaxMountainHeight
        }
    }
}
```

---

## Stage 3: Terrain Detail

Final pass adds fractal detail while preserving the hypsometric distribution:

```go
func AddTerrainDetail(elevation []float64, sites []Vector3D, settings DetailSettings) {
    for i, site := range sites {
        // Sample detail noise
        detail := fbmNoise(site, settings.DetailOctaves, settings.Lacunarity, settings.Persistence)

        // Scale amplitude by terrain type
        amplitude := settings.BaseAmplitude
        if elevation[i] > 3000 {
            amplitude *= settings.MountainDetailScale  // More detail in mountains
        } else if elevation[i] < -3000 {
            amplitude *= settings.DeepOceanDetailScale  // Less detail in deep ocean
        }

        elevation[i] += detail * amplitude
    }
}
```

---

## Automatic Threshold Calibration

Instead of manually tuning the land/ocean threshold, automatically find it:

```go
// FindOptimalThreshold uses binary search to achieve target land coverage
func FindOptimalThreshold(mask []float64, targetLandPct float64) float64 {
    low, high := -1.0, 1.0
    tolerance := 0.001  // 0.1% accuracy

    for high - low > tolerance {
        mid := (low + high) / 2
        landPct := countAboveThreshold(mask, mid)

        if landPct > targetLandPct {
            low = mid  // Need higher threshold to reduce land
        } else {
            high = mid  // Need lower threshold to increase land
        }
    }

    return (low + high) / 2
}
```

---

## Coastline Fractal Dimension Calculation

```go
// CalculateFractalDimension estimates the fractal dimension of coastlines
// using the box-counting method
func CalculateFractalDimension(coastlineSites []int, sites []Vector3D) float64 {
    // Box-counting: count how many boxes of size ε contain coastline points
    // D = -lim(log(N(ε)) / log(ε)) as ε → 0

    epsilons := []float64{0.5, 0.25, 0.125, 0.0625, 0.03125}  // Box sizes
    counts := make([]float64, len(epsilons))

    for i, eps := range epsilons {
        counts[i] = float64(countBoxes(coastlineSites, sites, eps))
    }

    // Linear regression of log(N) vs log(1/ε)
    slope := linearRegressionSlope(
        log1OverEpsilon(epsilons),
        logCounts(counts),
    )

    return slope  // This is the fractal dimension
}
```

---

## Parameter Auto-Tuning

For each generation, the system can automatically optimize parameters:

```go
type ParameterSearchResult struct {
    Parameters    TerrainSettings
    Score         float64
    Metrics       TerrainMetrics
    FailedMetrics []string  // Which metrics were outside acceptable range
}

func AutoTuneParameters(sites []Vector3D, targetScore float64, maxIterations int) ParameterSearchResult {
    best := ParameterSearchResult{Score: 0}

    for i := 0; i < maxIterations; i++ {
        // Generate with current/perturbed parameters
        params := perturbParameters(best.Parameters, i)
        elevation := GenerateTerrain(sites, params)
        score, metrics := EvaluateTerrain(sites, elevation)

        if score > best.Score {
            best = ParameterSearchResult{
                Parameters: params,
                Score: score,
                Metrics: metrics,
                FailedMetrics: findFailedMetrics(metrics),
            }
        }

        if score >= targetScore {
            break  // Good enough
        }
    }

    return best
}
```

---

## Test Output Format

Every test run should output:

```
=== TERRAIN EVALUATION ===
Score: 78.5% (target: 75%+)

PRIMARY METRICS:
  Land Coverage:      29.1% (target: 29.2% ±2%)     ✓ PASS
  Ocean Coverage:     70.9% (target: 70.8% ±2%)     ✓ PASS
  Mean Land Elev:     812m (target: 840m ±200m)     ✓ PASS
  Mean Ocean Depth:   -3721m (target: -3688m ±300m) ✓ PASS
  Mountain Coverage:  1.8% (target: 2.0% ±0.5%)     ✓ PASS
  Deep Ocean:         2.1% (target: 2.5% ±1.5%)     ✓ PASS
  Continental Shelf:  7.2% (target: 8.0% ±3%)       ✓ PASS

HYPSOMETRIC CURVE:
  Below -4000m:       51.2% (target: 52.4% ±5%)     ✓ PASS
  Below 0m:           70.9% (target: 70.8% ±2%)     ✓ PASS
  Below 1000m:        89.1% (target: 90.2% ±4%)     ✓ PASS

COASTLINE METRICS:
  Fractal Dimension:  1.28 (target: 1.30 ±0.15)     ✓ PASS
  Tortuosity Ratio:   2.8 (target: 3.0 ±2.0)        ✓ PASS

CONTINENTAL DISTRIBUTION:
  Major Landmasses:   6 (target: 4-10)              ✓ PASS
  Continent Gini:     0.42 (target: 0.45 ±0.15)     ✓ PASS

ELEVATION STATISTICS:
  Global Mean:        -2512m
  Global Std Dev:     2340m
  Maximum:            7234m
  Minimum:            -9821m

All 14 metrics within acceptable range.
```

---

## Why This Approach Solves the Problems

### Problem 1: Straight Boundaries

**Before**: Boundaries follow Voronoi cell edges
**After**: Domain warping displaces sampling positions, creating organic curves

The key is that we're not modifying the mesh - we're warping the *noise field sampling*. The mesh stays the same (your existing icosphere), but coastlines naturally curve because adjacent sites sample from warped positions.

### Problem 2: Complex Module Interactions

**Before**: 7 modules each modify elevation, interactions are unpredictable
**After**: 3 stages with clear data flow, each stage's output is the next stage's input

| Old System | New System |
|------------|------------|
| Base + Volcanic + Seafloor + Ridge + Tectonic + Erosion + Noise | Mask → Base → Detail |
| 7 interacting parameters per module | ~15 total parameters |
| Non-linear interactions | Linear pipeline |
| Hard to debug | Each stage independently testable |

---

## Implementation Plan

### Phase 1: Metrics & Evaluation Framework
- [ ] `metrics.go` - All metric calculations
- [ ] `hypsometric.go` - Hypsometric curve analysis
- [ ] `coastline_analysis.go` - Fractal dimension, tortuosity
- [ ] `evaluation.go` - Scoring function, output formatting

### Phase 2: Continental Mask (Core)
- [ ] `continental_mask.go` - Domain-warped noise for land/ocean
- [ ] `domain_warp.go` - Curl noise-based warping functions
- [ ] `test_continental_mask.go` - Visualization and parameter testing

### Phase 3: Base Elevation
- [ ] `base_elevation.go` - Continental/oceanic height assignment
- [ ] `mountain_seeds.go` - Mountain peak placement
- [ ] `test_base_elevation.go` - Hypsometric validation

### Phase 4: Terrain Detail
- [ ] `terrain_detail.go` - FBM detail layer
- [ ] `generation.go` - Pipeline orchestration
- [ ] `test_full_pipeline.go` - End-to-end testing

### Phase 5: Auto-Tuning & Integration
- [ ] `auto_tune.go` - Parameter optimization
- [ ] Server integration for visualization
- [ ] Comparison tooling vs Earth data

---

## File Structure

```
landgen/terrain/
├── TERRAIN_PLAN.md          # This document
├── types.go                 # Settings structs, interfaces
├── metrics.go               # Metric calculations
├── hypsometric.go           # Hypsometric curve analysis
├── coastline_analysis.go    # Fractal dimension, tortuosity
├── evaluation.go            # Scoring function
├── continental_mask.go      # Stage 1: Land/ocean mask
├── domain_warp.go           # Domain warping utilities
├── base_elevation.go        # Stage 2: Base heights
├── mountain_seeds.go        # Mountain generation
├── terrain_detail.go        # Stage 3: Fractal detail
├── generation.go            # Pipeline orchestration
├── auto_tune.go             # Parameter optimization
└── test/
    ├── test_metrics.go
    ├── test_mask.go
    ├── test_elevation.go
    └── test_pipeline.go
```

---

## Comparison: Old vs New

| Aspect | Old (Tectonic) | New (Noise-First) |
|--------|----------------|-------------------|
| Coastlines | Straight (Voronoi edges) | Irregular (domain warped) |
| Complexity | 7 interacting modules | 3 linear stages |
| Parameters | ~50+ | ~15 |
| Evaluation | Manual inspection | Automatic scoring |
| Iteration speed | Slow (cascade effects) | Fast (isolated stages) |
| Target matching | Emergent (hard to control) | Direct (hypsometric curve) |
| Mountains | From plate boundaries | From noise peaks |
| Debug-ability | Difficult | Each stage testable |

---

## Success Criteria

The new system is considered successful when:

1. **Score ≥ 75%** on the automatic evaluation
2. **All primary metrics** within acceptable ranges
3. **Fractal dimension > 1.15** (coastlines are not straight)
4. **Generation time < 5s** for level 7 mesh
5. **Reproducible** given the same seed

---

## Optional: Tectonic Features as Overlay

If you want tectonic-looking features (mountain chains, rift valleys) without the complexity:

1. Generate plates using existing quota-based system
2. Find boundaries (already implemented)
3. **Instead of using boundaries for elevation**, use them to:
   - Boost mountain noise near convergent boundaries
   - Create linear valley features near divergent boundaries
4. Domain-warp the boundary positions for irregular shapes

This gives tectonic "flavor" without the tight coupling.

---

## Next Steps

1. Implement evaluation framework first (metrics, scoring)
2. Implement Stage 1 (continental mask with domain warping)
3. Validate coastline irregularity with fractal dimension metric
4. Implement Stage 2 (base elevation)
5. Compare hypsometric curve to Earth
6. Implement Stage 3 (terrain detail)
7. Iterate on parameters until score ≥ 75%

The key advantage: each stage can be developed, tested, and tuned independently before moving to the next.
