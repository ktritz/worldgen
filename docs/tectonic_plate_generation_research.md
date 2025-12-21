# Tectonic Plate Generation: Research & Implementation Plan

**Date:** November 2025
**Purpose:** Develop realistic tectonic plate generation methods that match Earth's plate size distribution

---

## Executive Summary

Current plate generation uses a 5x overseeding + merging approach that achieves 0.34-0.39/1.0 Earth-benchmark validation. The main issue is that the simplified model doesn't create realistic plate size distributions matching Earth's. This research proposes 7 different generation methods inspired by geological theory and computational models, along with comprehensive automated evaluation metrics.

---

## 1. Earth's Plate Distribution (Target)

### Size Classes
- **7 Major plates** (>6% sphere): 20.3%, 11.3%, 10.7%, 10.5%, 8.9%, 8.1%, 6.9%
- **13 Minor plates** (0.18-6%): Range from 0.94% to 0.18%
- **19 Micro plates** (<0.18%): Range from 0.12% to 0.02%
- **Total: 39 plates**

### Key Statistical Properties
1. **Power Law Distribution**: Bird (2003) showed plate areas follow power law between 0.002 and 1 sr
2. **Coverage**: Major plates cover ~77%, minor ~7%, micro ~2%
3. **Continental Ratio**: ~28% of plates are continental (>50% continental crust)
4. **Size Gaps**: Natural separation between size classes (not uniform distribution)

### Earth Statistics Summary
```
Major plate count: 7
Minor plate count: 13
Micro plate count: 19
Largest plate: 20.3% (Pacific)
Smallest major: 6.9% (South American)
Largest minor: 0.94% (Somali)
Smallest minor: 0.18% (Amurian)
```

---

## 2. Current Implementation Analysis

### Current Approach (generation.go:74-130)
```
1. Start with 5x overseeding (targetPlates * 5)
2. Clustered seed placement around spreading centers
3. Basic distance assignment to nearest seed
4. Intelligent plate accretion merging down to target count
5. Continental landmass generation
```

### Strengths
- Creates natural convex boundaries through clustered growth
- Intelligent merging preserves size variation
- Achieves good geological features (ridges, subduction, etc.)

### Weaknesses
- Size distribution doesn't match Earth (too uniform)
- No power law emergence
- Ratio of major/minor/micro plates inconsistent
- Lacks the extreme size variation seen on Earth (20% to 0.02%)

---

## 3. Geological Research Findings

### Real-World Formation (Mantle Convection)
From recent 2025 research:
- Plates form from **stress-induced lithosphere breakup** over convection cells
- **Wadsleyite phase transitions** at 500-650km depth affect plume layering
- **Stress-history-dependent rheology** controls dynamics
- Plates naturally emerge from **self-organized criticality** (SOC) systems

### Key Phenomena
1. **Mantle upwelling** creates spreading centers → large plates
2. **Subduction zones** consume oceanic crust → plate shrinkage
3. **Rifting** fragments large plates → size distribution diversity
4. **Collisions** create super-plates → extreme sizes

### Power Law Emergence
- Many geophysical phenomena show power laws (earthquakes, craters, etc.)
- Power law exponent typically β ∈ [1, 2]
- Arises from **long-range interactions** and **self-organized criticality**
- Not uniform random distributions!

---

## 4. Evaluation Metrics System

### A. Size Distribution Metrics

#### 1. **Plate Count by Class**
```go
type PlateCountMetrics struct {
    MajorCount    int     // >6% sphere
    MinorCount    int     // 0.18-6%
    MicroCount    int     // <0.18%
    TargetMajor   int     // Earth: 7
    TargetMinor   int     // Earth: 13
    TargetMicro   int     // Earth: 19
    CountScore    float64 // Similarity to Earth distribution
}
```

#### 2. **Power Law Fit**
```go
type PowerLawMetrics struct {
    Exponent      float64 // β value (Earth: ~1.5)
    R2Fit         float64 // Goodness of fit
    ValidRange    [2]float64 // Range where power law holds
    KSStatistic   float64 // Kolmogorov-Smirnov test
}
```

#### 3. **Size Ratio Metrics**
```go
type SizeRatioMetrics struct {
    LargestToSmallest  float64 // Earth: ~1000x (20.3% / 0.02%)
    LargestToMedian    float64 // Earth: ~100x
    GiniCoefficient    float64 // Inequality measure (Earth: ~0.72)
    SizeGaps           []float64 // Natural breaks between classes
}
```

### B. Spatial Distribution Metrics

#### 4. **Boundary Characteristics**
```go
type BoundaryMetrics struct {
    AvgBoundaryLength     float64 // Per plate
    BoundaryComplexity    float64 // Fractal dimension
    ConvexityScores       []float64 // Per plate
    NeighborCount         []int   // Plates per neighbor count
}
```

#### 5. **Geographic Distribution**
```go
type SpatialMetrics struct {
    LatitudeBalance       float64 // N-S hemisphere balance
    AspectRatios          []float64 // Shape elongation
    CentroidDistribution  float64 // Uniformity of centers
    VoronoiDeviation      float64 // vs perfect Voronoi
}
```

### C. Composite Earth-Benchmark Score

```go
func CalculateEarthBenchmarkScore(metrics AllMetrics) float64 {
    weights := map[string]float64{
        "plateCountMatch":    0.25,  // Major/minor/micro counts
        "powerLawFit":        0.20,  // R² of power law
        "sizeVariation":      0.20,  // Gini + ratios
        "boundaryRealism":    0.15,  // Convexity + complexity
        "spatialDistribution": 0.10, // Geographic balance
        "continentalRatio":   0.10,  // Continental vs oceanic
    }

    // Weighted sum normalized to [0, 1.0]
    // Earth = 1.0 by definition
}
```

### D. Visual/AI Evaluation

```go
type VisualMetrics struct {
    MapImage          []byte  // Rendered plate map
    AIRealism Score   float64 // GPT-4 Vision assessment
    HumanRatings      []float64 // Optional user study
    EarthSimilarity   float64 // SSIM to Earth plate map
}
```

---

## 5. Proposed Generation Methods

### Method 1: **Hierarchical Voronoi with Weighted Sampling**

**Concept**: Use non-uniform centroid placement with area-based weighting to bias toward fewer large plates.

```go
// Pseudo-code
func HierarchicalVoronoi(targetCount int) []Plate {
    // Generate centroids with power-law distributed spacing
    centroids := PowerLawCentroidSampling(targetCount, beta=1.5)

    // Weighted Voronoi: larger "attractors" get more territory
    weights := PowerLawWeights(targetCount)
    plates := WeightedVoronoiPartition(centroids, weights)

    return plates
}
```

**Pros**: Simple, fast, directly encodes size distribution
**Cons**: Doesn't simulate geological processes, may lack realism
**Expected Score**: 0.50-0.65

---

### Method 2: **Stochastic Fracturing (Top-Down)**

**Concept**: Start with 1 plate, recursively split using power-law probability.

```go
func StochasticFracturing(minPlates, maxPlates int) []Plate {
    plates := []Plate{entireSphere}

    for len(plates) < targetPlates {
        // Select plate to split (bias toward large plates)
        plate := SelectByAreaProbability(plates)

        // Split with geologically-inspired fracture line
        fragments := FracturePlate(plate, numFragments=2-4)

        plates = append(plates, fragments...)
    }

    return plates
}
```

**Pros**: Natural hierarchical process, creates size variation
**Cons**: Hard to control final distribution, may need many iterations
**Expected Score**: 0.55-0.70

---

### Method 3: **Mantle Convection Approximation**

**Concept**: Simulate simplified mantle convection cells, plates form naturally.

```go
func MantleConvectionMethod() []Plate {
    // Generate convection cells (Rayleigh-Bénard-like)
    cells := GenerateConvectionCells(
        numUpwellings=8-12,  // Spreading centers
        numDownwellings=6-10, // Subduction zones
    )

    // Lithosphere stress field from convection
    stressField := CalculateStressField(cells)

    // Fracture where stress exceeds threshold
    boundaries := FindFractureBoundaries(stressField, threshold)

    // Extract plate regions
    plates := ExtractPlates(boundaries)

    return plates
}
```

**Pros**: Physically motivated, naturally creates variation
**Cons**: Computationally expensive, many parameters
**Expected Score**: 0.70-0.85

---

### Method 4: **Modified Accretion with Size Targets**

**Concept**: Improve current method by targeting specific size distribution.

```go
func TargetedAccretion(sizeTargets []float64) []Plate {
    // Start with heavy overseeding (10-20x)
    initialPlates := ClusteredSeeding(targetPlates * 15)

    // Merge intelligently toward size distribution
    plates := initialPlates
    while !MatchesDistribution(plates, sizeTargets) {
        // Find merger that improves distribution fit
        (plateA, plateB) := FindBestMergerForDistribution(plates, sizeTargets)
        plates = MergePlates(plateA, plateB)
    }

    return plates
}
```

**Pros**: Builds on current system, targeted results
**Cons**: May need many iterations, local minima issues
**Expected Score**: 0.60-0.75

---

### Method 5: **Cellular Automata with Growth Rules**

**Concept**: Grid-based CA where cells follow growth rules that produce power-law plates.

```go
func CellularAutomataPlates(gridSize int) []Plate {
    grid := InitializeGrid(gridSize)

    // Seed initial plate nuclei
    nuclei := RandomSeeds(count=targetPlates/2)

    // Growth rules inspired by SOC
    for iteration := 0; iteration < maxIterations; iteration++ {
        for each cell in grid {
            // Probability of joining neighboring plate
            prob := CalculateAccretionProbability(cell, neighbors)

            if rand() < prob {
                cell.plate = SelectNeighborPlate(cell)
            }

            // Spontaneous nucleation (rare)
            if rand() < nucleationRate && noPlate(cell) {
                cell.plate = NewPlate()
            }
        }
    }

    return ExtractPlates(grid)
}
```

**Pros**: Self-organizing, can produce power laws naturally
**Cons**: Grid artifacts, parameter tuning needed
**Expected Score**: 0.55-0.70

---

### Method 6: **Graph-Based Partition Optimization**

**Concept**: Formulate as graph partitioning problem with power-law objective.

```go
func GraphPartitionMethod(mesh Mesh) []Plate {
    graph := BuildDualGraph(mesh)

    // Objective: minimize cut + match power-law distribution
    objective := func(partition []int) float64 {
        cutCost := CalculateCutSize(graph, partition)
        distCost := PowerLawDeviationCost(partition)
        convexityCost := ConvexityPenalty(partition)

        return alpha*cutCost + beta*distCost + gamma*convexityCost
    }

    // Optimize using simulated annealing / genetic algorithm
    partition := OptimizePartition(graph, objective, targetPlates)

    return PartitionToPlates(partition)
}
```

**Pros**: Precise control, mathematically clean
**Cons**: Computationally expensive, may look "too perfect"
**Expected Score**: 0.65-0.80

---

### Method 7: **Hybrid: Convection + Optimization**

**Concept**: Best of both worlds - physics-inspired initialization + optimization refinement.

```go
func HybridMethod() []Plate {
    // Phase 1: Generate initial plates from simplified physics
    initialPlates := MantleConvectionApproximation()

    // Phase 2: Iterative refinement toward Earth distribution
    plates := initialPlates
    for iteration := 0; iteration < refinementSteps; iteration++ {
        // Evaluate current distribution
        score := EarthBenchmarkScore(plates)

        // Apply targeted modifications
        if needsMoreLargePlates(plates) {
            plates = MergeSmallNeighbors(plates)
        }
        if needsMoreMicroplates(plates) {
            plates = SplitLargePlates(plates, targetSize)
        }

        // Ensure geological plausibility
        plates = EnsureConvexity(plates)
        plates = FixBoundaries(plates)
    }

    return plates
}
```

**Pros**: Balanced realism and control, best expected performance
**Cons**: Most complex implementation
**Expected Score**: 0.75-0.90

---

## 6. Automated Testing Framework

### Test Harness Structure

```go
// testing/plate_generator_test.go

type PlateGenerationTest struct {
    Name           string
    Generator      GeneratorFunc
    Parameters     map[string]interface{}
    ExpectedScore  float64
    Runs           int  // Multiple runs for statistics
}

type TestResults struct {
    Method         string
    Scores         []float64
    AvgScore       float64
    StdDev         float64
    BestRun        PlateSet
    Metrics        DetailedMetrics
    GenerationTime time.Duration
    Images         []string  // Paths to visualization
}

func RunComparisonTest() {
    methods := []PlateGenerationTest{
        {"Current", CurrentMethod, params1, 0.37, 100},
        {"HierarchicalVoronoi", Method1, params2, 0.60, 100},
        {"StochasticFracturing", Method2, params3, 0.65, 100},
        {"MantleConvection", Method3, params4, 0.75, 50},
        {"TargetedAccretion", Method4, params5, 0.70, 100},
        {"CellularAutomata", Method5, params6, 0.65, 100},
        {"GraphPartition", Method6, params7, 0.75, 20},
        {"Hybrid", Method7, params8, 0.85, 50},
    }

    results := []TestResults{}

    for _, test := range methods {
        fmt.Printf("Testing %s (%d runs)...\n", test.Name, test.Runs)

        result := RunMultipleGenerations(test)
        results = append(results, result)

        // Generate visualizations
        GeneratePlateMap(result.BestRun, fmt.Sprintf("output/%s_best.png", test.Name))
        GenerateMetricsReport(result, fmt.Sprintf("output/%s_report.html", test.Name))
    }

    // Comparative analysis
    GenerateComparisonReport(results, "output/comparison.html")
    GenerateScoreChart(results, "output/scores.png")
}
```

### Evaluation Metrics Implementation

```go
// evaluation/metrics.go

type MetricsCalculator struct {
    earthData *EarthPlateData
}

func (m *MetricsCalculator) EvaluateGeneration(plates []Plate) DetailedMetrics {
    metrics := DetailedMetrics{}

    // 1. Size distribution
    metrics.PlateCount = m.PlateCountMetrics(plates)
    metrics.PowerLaw = m.PowerLawFit(plates)
    metrics.SizeRatio = m.SizeRatioMetrics(plates)

    // 2. Spatial metrics
    metrics.Boundary = m.BoundaryMetrics(plates)
    metrics.Spatial = m.SpatialDistributionMetrics(plates)

    // 3. Geological metrics
    metrics.Continental = m.ContinentalRatioMetrics(plates)
    metrics.Convexity = m.ConvexityMetrics(plates)

    // 4. Composite score
    metrics.EarthBenchmark = m.CalculateEarthBenchmarkScore(metrics)

    return metrics
}

func (m *MetricsCalculator) PowerLawFit(plates []Plate) PowerLawMetrics {
    // Extract sizes
    sizes := ExtractPlateSizes(plates)
    sort.Float64s(sizes)

    // Fit power law: P(x) = C * x^(-β)
    // Use log-log linear regression
    logSizes := Log(sizes)
    logRanks := Log(Ranks(sizes))

    beta, intercept, r2 := LinearRegression(logSizes, logRanks)

    // Kolmogorov-Smirnov test for goodness of fit
    ks := KSTest(sizes, PowerLawDistribution(beta))

    return PowerLawMetrics{
        Exponent: beta,
        R2Fit: r2,
        KSStatistic: ks,
        ValidRange: FindPowerLawRange(sizes, beta),
    }
}

func (m *MetricsCalculator) CalculateEarthBenchmarkScore(metrics DetailedMetrics) float64 {
    // Compare to Earth's known values

    // 1. Plate count similarity (target: 7 major, 13 minor, 19 micro)
    countScore := 1.0 - (
        abs(metrics.PlateCount.MajorCount - 7)/7.0 * 0.4 +
        abs(metrics.PlateCount.MinorCount - 13)/13.0 * 0.3 +
        abs(metrics.PlateCount.MicroCount - 19)/19.0 * 0.3
    )

    // 2. Power law fit quality (target: R² > 0.9, β ≈ 1.5)
    powerLawScore := metrics.PowerLaw.R2Fit *
                     (1.0 - abs(metrics.PowerLaw.Exponent - 1.5)/1.5)

    // 3. Size variation (target: Gini ≈ 0.72, largest/smallest ≈ 1000x)
    giniDiff := abs(metrics.SizeRatio.GiniCoefficient - 0.72) / 0.72
    ratioScore := min(metrics.SizeRatio.LargestToSmallest / 1000.0, 1.0)
    sizeVarScore := (ratioScore * 0.6 + (1.0 - giniDiff) * 0.4)

    // 4. Boundary realism (target: avg convexity > 0.85)
    avgConvexity := Mean(metrics.Convexity.Scores)
    boundaryScore := avgConvexity

    // 5. Continental ratio (target: 28% continental plates)
    continentalScore := 1.0 - abs(metrics.Continental.Ratio - 0.28) / 0.28

    // Weighted combination
    score := (
        countScore * 0.25 +
        powerLawScore * 0.20 +
        sizeVarScore * 0.20 +
        boundaryScore * 0.15 +
        continentalScore * 0.10 +
        metrics.Spatial.Score * 0.10
    )

    return Clamp(score, 0.0, 1.0)
}
```

### Visualization Generation

```go
// visualization/plate_maps.go

func GeneratePlateMap(plates []Plate, outputPath string) {
    // Create equirectangular projection map
    width, height := 3600, 1800  // 0.1° resolution
    img := image.NewRGBA(image.Rect(0, 0, width, height))

    // Color each plate uniquely
    colors := GenerateDistinctColors(len(plates))

    for y := 0; y < height; y++ {
        lat := 90.0 - float64(y) * 180.0 / float64(height)
        for x := 0; x < width; x++ {
            lon := float64(x) * 360.0 / float64(width) - 180.0

            // Find plate at this location
            point := LatLonToSphere(lat, lon)
            plate := FindPlateAtPoint(plates, point)

            img.Set(x, y, colors[plate.ID])
        }
    }

    // Draw boundaries
    DrawPlateBoundaries(img, plates)

    // Add legend
    DrawLegend(img, plates, colors)

    // Save
    SavePNG(img, outputPath)
}

func GenerateMetricsReport(result TestResults, outputPath string) {
    html := `<html><head><title>` + result.Method + ` Report</title></head><body>`
    html += `<h1>` + result.Method + `</h1>`
    html += `<h2>Earth Benchmark Score: ` + fmt.Sprintf("%.3f", result.AvgScore) + `</h2>`

    // Size distribution histogram
    html += `<h3>Plate Size Distribution</h3>`
    html += GenerateSizeHistogramSVG(result.Metrics)

    // Power law fit chart
    html += `<h3>Power Law Fit</h3>`
    html += GeneratePowerLawChartSVG(result.Metrics)

    // Detailed metrics table
    html += `<h3>Detailed Metrics</h3>`
    html += GenerateMetricsTableHTML(result.Metrics)

    // Sample plate maps
    html += `<h3>Generated Plates</h3>`
    for _, imgPath := range result.Images {
        html += `<img src="` + imgPath + `" width="800"/>`
    }

    html += `</body></html>`

    os.WriteFile(outputPath, []byte(html), 0644)
}
```

---

## 7. Implementation Roadmap

### Phase 1: Evaluation Framework (Week 1)
1. Implement metrics calculation system
2. Create Earth benchmark reference data
3. Build visualization generation
4. Test on current implementation

### Phase 2: Simple Methods (Week 2)
1. Implement Method 1: Hierarchical Voronoi
2. Implement Method 2: Stochastic Fracturing
3. Implement Method 4: Modified Accretion
4. Run comparative tests

### Phase 3: Advanced Methods (Week 3-4)
1. Implement Method 3: Mantle Convection Approximation
2. Implement Method 5: Cellular Automata
3. Implement Method 6: Graph Partition
4. Run comparative tests

### Phase 4: Hybrid & Optimization (Week 5)
1. Implement Method 7: Hybrid approach
2. Parameter tuning for best methods
3. Final comparative evaluation
4. Select winning approach

### Phase 5: Integration (Week 6)
1. Integrate best method into main codebase
2. Update pipeline to use new generation
3. Test with full tectonics simulation
4. Documentation and examples

---

## 8. Success Criteria

### Minimum Viable Success
- Earth benchmark score > 0.65 (vs current 0.37)
- Power law fit R² > 0.85
- 6-8 major plates, 10-15 minor plates

### Target Success
- Earth benchmark score > 0.75
- Power law fit R² > 0.90
- Realistic convexity (avg > 0.85)
- Visually indistinguishable from Earth plate distribution

### Stretch Goals
- Earth benchmark score > 0.85
- User study shows >80% prefer new method
- Generalizes to non-Earth planets (Mars, Venus, exoplanets)

---

## 9. References

### Academic Papers
- Cortial et al. (2019): "Procedural Tectonic Planets" - CGF
- Bird (2003): Power law distribution in tectonic plates
- Li et al. (2025): Phase transitions in mantle convection

### Key Concepts
- Power law distributions (Newman 2005)
- Self-organized criticality in geophysics
- Spherical Voronoi diagrams
- Graph partitioning algorithms

### Code References
- Current implementation: `landgen/tectonics/generation.go`
- Earth data: `landgen/tectonics/earth_plate_data.go`
- Plate algorithms: `landgen/tectonics/plate_algorithms.go`

---

## 10. Next Steps

1. **Immediate**: Implement evaluation metrics framework
2. **Short-term**: Test 2-3 simplest methods
3. **Medium-term**: Implement and compare all 7 methods
4. **Long-term**: Integrate winner into production pipeline

**Questions for User:**
- Which methods seem most promising to prioritize?
- Acceptable runtime for plate generation? (current: <2s)
- Preference for physical realism vs control/speed?
- Should we support non-Earth-like distributions (e.g., Pangaea worlds)?

---

**End of Research Document**
