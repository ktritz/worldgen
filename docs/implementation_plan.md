# Tectonic Plate Generation: Implementation Plan
## Focus: Earth-like Power Law Distribution with Comprehensive Evaluation

**Priority**: Physical realism > Speed
**Goal**: Achieve Earth benchmark score > 0.75
**Timeline**: 6-8 weeks for full implementation

---

## Phase 1: Comprehensive Evaluation Framework (Week 1-2)

### 1.1 Core Metrics Implementation

Create `landgen/tectonics/evaluation/` package:

```
landgen/tectonics/evaluation/
├── metrics.go              # Core metrics calculation
├── earth_benchmark.go      # Earth comparison scoring
├── power_law.go            # Power law fitting and analysis
├── visualization.go        # Map and chart generation
├── report.go              # HTML report generation
└── metrics_test.go        # Unit tests
```

**Key Components:**

#### A. Power Law Analysis
```go
// power_law.go
type PowerLawAnalysis struct {
    Exponent        float64   // β parameter
    R2Fit           float64   // Goodness of fit (0-1)
    KSStatistic     float64   // Kolmogorov-Smirnov test
    ValidRange      [2]float64 // Size range where power law holds
    PlateSizes      []float64  // Sorted plate sizes
    TheoreticalFit  []float64  // Expected power law values
}

func AnalyzePowerLaw(plates []TectonicPlate, planetRadius float64) PowerLawAnalysis {
    // 1. Extract and sort plate sizes (as % of sphere)
    // 2. Fit log-log linear regression: log(P(x)) = log(C) - β*log(x)
    // 3. Calculate R² goodness of fit
    // 4. Run Kolmogorov-Smirnov test
    // 5. Find valid range (typically 0.002 - 1.0 sr for Earth)
    // 6. Return complete analysis
}
```

#### B. Size Distribution Metrics
```go
// metrics.go
type SizeDistributionMetrics struct {
    // Plate counts by class
    MajorCount      int       // >6% sphere
    MinorCount      int       // 0.18-6%
    MicroCount      int       // 0.02-0.18%
    NanoCount       int       // <0.02% (usually absorbed)

    // Statistical measures
    GiniCoefficient float64   // Inequality: 0 (equal) to 1 (extreme)
    LargestPlate    float64   // % of sphere
    SmallestPlate   float64   // % of sphere
    SizeRatio       float64   // Largest/smallest
    MedianSize      float64   // Median plate size

    // Distribution shape
    Mean            float64
    StdDev          float64
    Skewness        float64   // Asymmetry of distribution
    Kurtosis        float64   // Tail heaviness

    // Earth comparison
    EarthSimilarity float64   // 0-1 score vs Earth distribution
}

func CalculateSizeDistribution(plates []TectonicPlate, sphereArea float64) SizeDistributionMetrics
```

#### C. Earth Benchmark Score
```go
// earth_benchmark.go
type EarthBenchmark struct {
    OverallScore       float64  // Composite 0-1 score

    // Component scores (each 0-1)
    PlateCountScore    float64  // Major/minor/micro match
    PowerLawScore      float64  // Power law fit quality
    SizeVariationScore float64  // Gini + size ratios
    BoundaryScore      float64  // Convexity + realism
    SpatialScore       float64  // Geographic distribution
    ContinentalScore   float64  // Continental plate ratio

    // Detailed breakdown
    PlateCountDelta    [3]int   // Difference from Earth [major, minor, micro]
    PowerLawExponent   float64  // β value (Earth ≈ 1.5)
    PowerLawR2         float64  // Fit quality (target > 0.9)

    // Comparison data
    EarthData          *EarthPlateDistribution
}

func CalculateEarthBenchmark(plates []TectonicPlate, settings TectonicSettings) EarthBenchmark {
    // Weighted scoring:
    // - 25% plate count match (7 major, 13 minor, 19 micro)
    // - 20% power law fit quality
    // - 20% size variation (Gini ≈ 0.72, ratio ≈ 1000x)
    // - 15% boundary realism
    // - 10% spatial distribution
    // - 10% continental ratio (≈ 28%)
}
```

#### D. Visualization Generation
```go
// visualization.go

// Generate equirectangular plate map
func GeneratePlateMap(plates []TectonicPlate, cellAssignments []int,
                       voronoiCells []VoronoiCell, icosphereSites []Vector3D,
                       outputPath string, width, height int)

// Generate size distribution histogram with power law overlay
func GenerateSizeHistogram(metrics SizeDistributionMetrics,
                           powerLaw PowerLawAnalysis,
                           outputPath string)

// Generate log-log plot showing power law fit
func GeneratePowerLawPlot(powerLaw PowerLawAnalysis, outputPath string)

// Generate comparative radar chart (current vs Earth)
func GenerateComparisonRadar(benchmark EarthBenchmark, outputPath string)

// Generate convexity heatmap
func GenerateConvexityMap(plates []TectonicPlate, convexityScores []float64,
                          outputPath string)
```

#### E. Report Generation
```go
// report.go
type EvaluationReport struct {
    MethodName      string
    Timestamp       time.Time

    Metrics         SizeDistributionMetrics
    PowerLaw        PowerLawAnalysis
    Benchmark       EarthBenchmark

    Visualizations  map[string]string  // name -> file path
    Parameters      map[string]interface{}

    GenerationTime  time.Duration
    PlateCount      int
}

func GenerateHTMLReport(report EvaluationReport, outputPath string) {
    // Create comprehensive HTML report with:
    // - Executive summary (benchmark score)
    // - Plate size table
    // - All visualizations embedded
    // - Detailed metrics tables
    // - Parameter listing
    // - Comparison to Earth
}
```

### 1.2 Testing Infrastructure

```go
// cmd/evaluate_plates/main.go
// Standalone tool for evaluating any plate generation

func main() {
    // Load generated plates from JSON/binary
    plates := LoadPlates("path/to/plates.json")

    // Run comprehensive evaluation
    metrics := evaluation.CalculateSizeDistribution(plates, sphereArea)
    powerLaw := evaluation.AnalyzePowerLaw(plates, planetRadius)
    benchmark := evaluation.CalculateEarthBenchmark(plates, settings)

    // Generate all visualizations
    evaluation.GeneratePlateMap(plates, "output/plate_map.png", 3600, 1800)
    evaluation.GenerateSizeHistogram(metrics, powerLaw, "output/histogram.png")
    evaluation.GeneratePowerLawPlot(powerLaw, "output/powerlaw.png")

    // Create HTML report
    report := evaluation.EvaluationReport{
        MethodName: "Current Implementation",
        Metrics: metrics,
        PowerLaw: powerLaw,
        Benchmark: benchmark,
        // ...
    }
    evaluation.GenerateHTMLReport(report, "output/report.html")

    // Print summary
    fmt.Printf("Earth Benchmark Score: %.3f\n", benchmark.OverallScore)
    fmt.Printf("Power Law Exponent: %.2f (R² = %.3f)\n",
               powerLaw.Exponent, powerLaw.R2Fit)
}
```

**Deliverables:**
- ✅ Complete evaluation package
- ✅ Validation against current implementation (baseline score)
- ✅ Validation against Earth data (should score 1.0)
- ✅ Standalone evaluation tool

---

## Phase 2: Mantle Convection-Inspired Generation (Week 3-5)

### Rationale
Mantle convection naturally produces power-law distributions through self-organized criticality. This is the most physically grounded approach.

### 2.1 Simplified Mantle Dynamics Model

**Key Insight from Research:**
- Plates form from stress-induced lithosphere breakup
- Upwelling zones → spreading centers → large plates
- Downwelling zones → subduction → plate consumption
- Natural size hierarchy emerges from convection cell distribution

```go
// landgen/tectonics/mantle_convection.go

type ConvectionCell struct {
    Center    Vector3D
    Type      CellType  // Upwelling or Downwelling
    Strength  float64   // Flow intensity
    Radius    float64   // Cell influence radius
}

type CellType int
const (
    Upwelling   CellType = iota  // Spreading center
    Downwelling                  // Subduction zone
)

type StressField struct {
    Sites      []Vector3D
    Stress     []float64  // Stress magnitude at each site
    Direction  []Vector3D // Stress direction (velocity field)
}

// Generate convection cells using physical constraints
func GenerateConvectionCells(settings ConvectionSettings, rng *rand.Rand) []ConvectionCell {
    cells := []ConvectionCell{}

    // Upwelling cells (fewer, larger) - create spreading centers
    numUpwellings := settings.NumUpwellings  // 6-10 for Earth-like
    upwellings := PlaceConvectionCells(numUpwellings, Upwelling, settings, rng)
    cells = append(cells, upwellings...)

    // Downwelling cells (more numerous, varied sizes)
    numDownwellings := settings.NumDownwellings  // 8-15 for Earth-like
    downwellings := PlaceConvectionCells(numDownwellings, Downwelling, settings, rng)
    cells = append(cells, downwellings...)

    // Adjust cell strengths to create size hierarchy
    // Larger cells = stronger flow = larger plates
    AssignCellStrengths(cells, settings.StrengthDistribution, rng)

    return cells
}

func PlaceConvectionCells(count int, cellType CellType,
                          settings ConvectionSettings,
                          rng *rand.Rand) []ConvectionCell {
    cells := []ConvectionCell{}

    // Use hierarchical placement with power-law spacing
    // This creates natural size variation in resulting plates

    if settings.UseHierarchicalPlacement {
        // Generate large cells first, fill gaps with smaller ones
        largeCells := PlaceLargeCells(count/3, cellType, settings, rng)
        mediumCells := PlaceMediumCells(count/3, cellType, largeCells, settings, rng)
        smallCells := PlaceSmallCells(count - len(largeCells) - len(mediumCells),
                                      cellType, append(largeCells, mediumCells...),
                                      settings, rng)

        cells = append(cells, largeCells...)
        cells = append(cells, mediumCells...)
        cells = append(cells, smallCells...)
    } else {
        // Simpler Poisson disk sampling with variable radii
        cells = PoissonDiskSampling(count, cellType, settings, rng)
    }

    return cells
}

func AssignCellStrengths(cells []ConvectionCell, distribution string, rng *rand.Rand) {
    // Strength distribution affects plate sizes
    // Options: "uniform", "power_law", "exponential", "lognormal"

    switch distribution {
    case "power_law":
        // Assign strengths following power law
        // This directly creates power law plate distribution
        exponent := 1.5  // Tunable parameter

        // Sort cells by size for consistent assignment
        sort.Slice(cells, func(i, j int) bool {
            return cells[i].Radius > cells[j].Radius
        })

        for i := range cells {
            rank := float64(i + 1)
            // Power law: strength ∝ rank^(-α)
            cells[i].Strength = math.Pow(rank, -exponent)
        }

    case "lognormal":
        // Lognormal distribution (realistic for natural processes)
        mu := 0.0
        sigma := 1.0
        for i := range cells {
            cells[i].Strength = math.Exp(rng.NormFloat64()*sigma + mu)
        }

    // ... other distributions
    }

    // Normalize strengths
    NormalizeStrengths(cells)
}
```

### 2.2 Stress Field Calculation

```go
// Calculate lithosphere stress from convection cells
func CalculateStressField(cells []ConvectionCell,
                         icosphereSites []Vector3D,
                         planetRadius float64) StressField {

    stress := make([]float64, len(icosphereSites))
    direction := make([]Vector3D, len(icosphereSites))

    // For each point on lithosphere
    for i, site := range icosphereSites {
        totalStress := Vector3D{X: 0, Y: 0, Z: 0}

        // Accumulate influence from all convection cells
        for _, cell := range cells {
            // Distance from cell center
            dist := CalculateSphericalDistance(site, cell.Center, planetRadius)

            // Influence falls off with distance
            influence := cell.Strength * InfluenceFunction(dist, cell.Radius)

            if influence < 0.001 {
                continue  // Negligible influence
            }

            // Direction: away from upwellings, toward downwellings
            dir := site.Subtract(cell.Center).Normalize()
            if cell.Type == Downwelling {
                dir = dir.Scale(-1)  // Reverse for downwelling
            }

            // Add stress contribution
            stressContribution := dir.Scale(influence)
            totalStress = totalStress.Add(stressContribution)
        }

        stress[i] = totalStress.Length()
        if stress[i] > 0 {
            direction[i] = totalStress.Scale(1.0 / stress[i])
        }
    }

    return StressField{
        Sites: icosphereSites,
        Stress: stress,
        Direction: direction,
    }
}

func InfluenceFunction(distance, cellRadius float64) float64 {
    // Gaussian falloff
    sigma := cellRadius * 0.5
    return math.Exp(-0.5 * (distance*distance) / (sigma*sigma))
}
```

### 2.3 Plate Boundary Detection

```go
// Find plate boundaries where stress field creates fractures
func DetectPlateBoundaries(stressField StressField,
                          voronoiCells []VoronoiCell,
                          threshold float64) []bool {

    isBoundary := make([]bool, len(voronoiCells))

    for cellIdx, cell := range voronoiCells {
        siteIdx := cell.SiteIndex

        if int(siteIdx) >= len(stressField.Stress) {
            continue
        }

        localStress := stressField.Stress[siteIdx]
        localDir := stressField.Direction[siteIdx]

        // Check stress divergence with neighbors
        maxDivergence := 0.0

        for _, neighborIdx := range cell.NeighborSiteIndices {
            if int(neighborIdx) >= len(stressField.Stress) {
                continue
            }

            neighborStress := stressField.Stress[neighborIdx]
            neighborDir := stressField.Direction[neighborIdx]

            // Divergence = difference in stress direction + magnitude
            dirDiff := 1.0 - localDir.Dot(neighborDir)  // 0 (same) to 2 (opposite)
            stressDiff := math.Abs(localStress - neighborStress) /
                         math.Max(localStress, neighborStress)

            divergence := dirDiff + stressDiff

            if divergence > maxDivergence {
                maxDivergence = divergence
            }
        }

        // Boundary where stress is highly divergent
        isBoundary[cellIdx] = maxDivergence > threshold
    }

    return isBoundary
}
```

### 2.4 Plate Region Growing

```go
// Grow plates from convection cell centers following stress field
func GrowPlatesFromStressField(stressField StressField,
                               cells []ConvectionCell,
                               voronoiCells []VoronoiCell,
                               icosphereSites []Vector3D,
                               planetRadius float64) []int {

    cellAssignments := make([]int, len(voronoiCells))
    for i := range cellAssignments {
        cellAssignments[i] = -1  // Unassigned
    }

    // Create plate seeds at upwelling centers
    plateSeeds := []int{}
    for _, cell := range cells {
        if cell.Type == Upwelling {
            // Find nearest voronoi cell to convection cell center
            nearestIdx := FindNearestVoronoiCell(cell.Center, voronoiCells, icosphereSites)
            plateSeeds = append(plateSeeds, nearestIdx)
        }
    }

    // Also create seeds in low-stress regions (micro-plates)
    microPlateSeeds := FindLowStressRegions(stressField, voronoiCells,
                                            plateSeeds,
                                            threshold=0.2)
    plateSeeds = append(plateSeeds, microPlateSeeds...)

    // Assign initial seeds
    for plateIdx, seedIdx := range plateSeeds {
        cellAssignments[seedIdx] = plateIdx
    }

    // Grow plates using competitive region growing
    // Priority based on stress field alignment

    queue := make(PriorityQueue, 0)

    // Initialize queue with seed neighbors
    for plateIdx, seedIdx := range plateSeeds {
        cell := voronoiCells[seedIdx]
        for _, neighborIdx := range cell.NeighborSiteIndices {
            if cellAssignments[neighborIdx] == -1 {
                priority := CalculateGrowthPriority(
                    neighborIdx, plateIdx, plateSeeds[plateIdx],
                    stressField, voronoiCells, icosphereSites, planetRadius)

                queue.Push(&QueueItem{
                    CellIdx: int(neighborIdx),
                    PlateIdx: plateIdx,
                    Priority: priority,
                })
            }
        }
    }

    // Competitive growing
    for queue.Len() > 0 {
        item := heap.Pop(&queue).(*QueueItem)

        if cellAssignments[item.CellIdx] != -1 {
            continue  // Already assigned
        }

        cellAssignments[item.CellIdx] = item.PlateIdx

        // Add unassigned neighbors to queue
        cell := voronoiCells[item.CellIdx]
        for _, neighborIdx := range cell.NeighborSiteIndices {
            if cellAssignments[neighborIdx] == -1 {
                priority := CalculateGrowthPriority(
                    neighborIdx, item.PlateIdx, plateSeeds[item.PlateIdx],
                    stressField, voronoiCells, icosphereSites, planetRadius)

                queue.Push(&QueueItem{
                    CellIdx: int(neighborIdx),
                    PlateIdx: item.PlateIdx,
                    Priority: priority,
                })
            }
        }
    }

    return cellAssignments
}

func CalculateGrowthPriority(cellIdx, plateIdx, seedIdx int,
                             stressField StressField,
                             voronoiCells []VoronoiCell,
                             icosphereSites []Vector3D,
                             planetRadius float64) float64 {

    cell := voronoiCells[cellIdx]
    siteIdx := cell.SiteIndex

    // Priority based on:
    // 1. Stress field alignment (flowing toward plate)
    // 2. Distance from seed
    // 3. Stress magnitude (higher stress = stronger plate attachment)

    cellPos := icosphereSites[siteIdx]
    seedPos := icosphereSites[voronoiCells[seedIdx].SiteIndex]

    // Direction from seed to cell
    toCell := cellPos.Subtract(seedPos).Normalize()

    // Stress direction at cell
    stressDir := stressField.Direction[siteIdx]

    // Alignment: higher if stress points away from seed (growing outward)
    alignment := toCell.Dot(stressDir)

    // Distance factor (closer = higher priority)
    dist := CalculateSphericalDistance(cellPos, seedPos, planetRadius)
    distFactor := 1.0 / (1.0 + dist/planetRadius)

    // Stress magnitude (higher stress = clearer plate assignment)
    stressMagnitude := stressField.Stress[siteIdx]

    // Combined priority
    priority := alignment * 0.4 + distFactor * 0.3 + stressMagnitude * 0.3

    return priority
}
```

### 2.5 Main Generation Function

```go
// landgen/tectonics/generation_mantle_convection.go

type ConvectionSettings struct {
    NumUpwellings         int     // 6-10 for Earth-like
    NumDownwellings       int     // 8-15 for Earth-like
    StrengthDistribution  string  // "power_law", "lognormal", etc.
    PowerLawExponent      float64 // For strength distribution (≈1.5)
    StressThreshold       float64 // Boundary detection sensitivity
    UseHierarchicalPlacement bool // Create size hierarchy in cells

    // Size variation parameters
    LargeCellRadius       float64 // Radius for major plate-forming cells
    MediumCellRadius      float64
    SmallCellRadius       float64

    // Micro-plate generation
    MicroPlateThreshold   float64 // Stress level for micro-plate seeds
    MinMicroPlateCount    int
}

func InitializeTectonicPlatesMantle Convection(
    voronoiCells []VoronoiCell,
    voronoiVertices []Vector3D,
    icosphereSites []Vector3D,
    settings TectonicSettings,
) ([]TectonicPlate, []int32) {

    fmt.Printf("=== MANTLE CONVECTION PLATE GENERATION ===\n")

    rng := rand.New(rand.NewSource(settings.Seed))

    // Step 1: Generate convection cells
    fmt.Printf("Generating convection cells...\n")
    cells := GenerateConvectionCells(settings.ConvectionConfig, rng)
    fmt.Printf("  %d upwelling cells, %d downwelling cells\n",
               CountCellType(cells, Upwelling),
               CountCellType(cells, Downwelling))

    // Step 2: Calculate stress field
    fmt.Printf("Calculating lithosphere stress field...\n")
    stressField := CalculateStressField(cells, icosphereSites, settings.PlanetRadius)

    // Step 3: Grow plates from stress field
    fmt.Printf("Growing tectonic plates...\n")
    cellAssignments := GrowPlatesFromStressField(
        stressField, cells, voronoiCells, icosphereSites, settings.PlanetRadius)

    numPlates := CountUniquePlates(cellAssignments)
    fmt.Printf("  Generated %d initial plates\n", numPlates)

    // Step 4: Remove nano-plates (optional cleanup)
    if settings.RemoveNanoPlates {
        fmt.Printf("Removing nano-plates...\n")
        cellAssignments = AbsorbNanoPlates(
            voronoiCells, icosphereSites, cellAssignments,
            settings.NanoPlateThreshold, settings.PlanetRadius)
    }

    // Step 5: Extract seed indices from assignments
    seedIndices := ExtractPlateSeeds(cellAssignments, voronoiCells)

    // Step 6: Calculate plate areas
    sphereArea := 4.0 * math.Pi * settings.PlanetRadius * settings.PlanetRadius
    plateAreas := CalculatePlateAreas(cellAssignments, len(seedIndices), sphereArea, len(voronoiCells))

    // Step 7: Create plate structures
    fmt.Printf("Creating plate structures...\n")
    plates := createSimplifiedPlates(voronoiCells, icosphereSites,
                                    seedIndices, plateAreas, settings, rng)

    // Step 8: Assign plate types (continental vs oceanic)
    fmt.Printf("Assigning plate types...\n")
    assignSimplifiedPlateTypes(plates, settings, rng)

    // Step 9: Analyze results
    fmt.Printf("=== GENERATION COMPLETE ===\n")
    AnalyzeGenerationResults(plates, sphereArea)

    // Convert to site assignments
    siteAssignments := convertCellAssignmentsToSites(icosphereSites, voronoiCells, cellAssignments)

    return plates, siteAssignments
}

func AnalyzeGenerationResults(plates []TectonicPlate, sphereArea float64) {
    // Quick analysis
    majorCount := 0
    minorCount := 0
    microCount := 0

    majorThreshold := sphereArea * 0.06
    minorThreshold := sphereArea * 0.0018

    for _, plate := range plates {
        if plate.Area >= majorThreshold {
            majorCount++
        } else if plate.Area >= minorThreshold {
            minorCount++
        } else {
            microCount++
        }
    }

    fmt.Printf("Plate distribution: %d major, %d minor, %d micro\n",
               majorCount, minorCount, microCount)
    fmt.Printf("Total: %d plates\n", len(plates))
}
```

**Deliverables:**
- ✅ Complete mantle convection generation method
- ✅ Multiple convection configurations for testing
- ✅ Integration with existing pipeline
- ✅ Comparative evaluation vs current method

---

## Phase 3: Parameter Tuning & Optimization (Week 6-7)

### 3.1 Automated Parameter Search

```go
// cmd/optimize_plates/main.go

type ParameterConfig struct {
    NumUpwellings        []int     // Try: [6, 7, 8, 9, 10]
    NumDownwellings      []int     // Try: [8, 10, 12, 15]
    PowerLawExponent     []float64 // Try: [1.3, 1.4, 1.5, 1.6, 1.7]
    StressThreshold      []float64 // Try: [0.3, 0.4, 0.5, 0.6]
    MicroPlateThreshold  []float64 // Try: [0.15, 0.20, 0.25]
}

func GridSearch(config ParameterConfig, numRuns int) []OptimizationResult {
    results := []OptimizationResult{}

    // Generate all parameter combinations
    combinations := GenerateCombinations(config)

    fmt.Printf("Testing %d parameter combinations (%d runs each)...\n",
               len(combinations), numRuns)

    for i, params := range combinations {
        fmt.Printf("Combination %d/%d: %v\n", i+1, len(combinations), params)

        scores := []float64{}
        var bestPlates []TectonicPlate
        bestScore := 0.0

        for run := 0; run < numRuns; run++ {
            // Generate plates with these parameters
            settings := CreateSettings(params, run)
            plates, _ := InitializeTectonicPlatesMantle Convection(
                voronoiCells, voronoiVertices, icosphereSites, settings)

            // Evaluate
            benchmark := evaluation.CalculateEarthBenchmark(plates, settings)
            score := benchmark.OverallScore
            scores = append(scores, score)

            if score > bestScore {
                bestScore = score
                bestPlates = plates
            }
        }

        avgScore := Mean(scores)
        stdDev := StdDev(scores)

        result := OptimizationResult{
            Parameters: params,
            AvgScore: avgScore,
            StdDev: stdDev,
            BestScore: bestScore,
            BestPlates: bestPlates,
        }

        results = append(results, result)

        // Save intermediate results
        SaveResults(results, "optimization_progress.json")
    }

    // Sort by score
    sort.Slice(results, func(i, j int) bool {
        return results[i].AvgScore > results[j].AvgScore
    })

    return results
}

func main() {
    config := ParameterConfig{
        NumUpwellings: []int{6, 7, 8, 9, 10},
        NumDownwellings: []int{8, 10, 12, 15},
        PowerLawExponent: []float64{1.3, 1.5, 1.7},
        StressThreshold: []float64{0.4, 0.5, 0.6},
        MicroPlateThreshold: []float64{0.15, 0.20, 0.25},
    }

    // Run grid search (5*4*3*3*3 = 540 combinations)
    results := GridSearch(config, numRuns=20)  // 10,800 total runs

    // Print top 10 configurations
    fmt.Printf("\n=== TOP 10 CONFIGURATIONS ===\n")
    for i := 0; i < 10 && i < len(results); i++ {
        r := results[i]
        fmt.Printf("%d. Score: %.3f ± %.3f\n", i+1, r.AvgScore, r.StdDev)
        fmt.Printf("   Parameters: %v\n", r.Parameters)
    }

    // Generate detailed report for best configuration
    best := results[0]
    GenerateDetailedReport(best, "output/best_configuration.html")
}
```

### 3.2 Evolutionary Optimization (if grid search insufficient)

```go
// Use genetic algorithm for continuous parameter optimization
type Genome struct {
    NumUpwellings       float64  // Continuous, round to int
    NumDownwellings     float64
    PowerLawExponent    float64
    StressThreshold     float64
    MicroPlateThreshold float64
    LargeCellRadius     float64
    MediumCellRadius    float64
    SmallCellRadius     float64
}

func GeneticAlgorithm(populationSize, generations int) Genome {
    // Standard GA: selection, crossover, mutation
    // Fitness = Earth benchmark score
    // Can explore continuous parameter space efficiently
}
```

**Deliverables:**
- ✅ Optimal parameters for Earth-like distribution
- ✅ Sensitivity analysis (which parameters matter most)
- ✅ Parameter ranges for different world types

---

## Phase 4: Generalization & World Types (Week 8)

### 4.1 World Type Presets

Once we have Earth-like working, create parameterized presets:

```go
// landgen/tectonics/world_types.go

type WorldType string

const (
    EarthLike    WorldType = "earth"      // Default
    Pangaea      WorldType = "pangaea"    // Supercontinent
    Archipelago  WorldType = "archipelago" // Many small continents
    Waterworld   WorldType = "waterworld" // Few/no continents
    ShatteredWorld WorldType = "shattered" // Many micro-plates
)

func GetWorldTypeSettings(worldType WorldType, baseSettings TectonicSettings) TectonicSettings {
    settings := baseSettings

    switch worldType {
    case EarthLike:
        settings.ConvectionConfig = ConvectionSettings{
            NumUpwellings: 8,
            NumDownwellings: 12,
            PowerLawExponent: 1.5,
            StressThreshold: 0.5,
            MicroPlateThreshold: 0.20,
            UseHierarchicalPlacement: true,
        }
        settings.NumPlates = 20  // Target range
        settings.MajorPlateRatio = 0.30
        settings.TargetContinentalProportion = 0.29

    case Pangaea:
        // Fewer, larger plates → supercontinent
        settings.ConvectionConfig = ConvectionSettings{
            NumUpwellings: 4,           // Fewer spreading centers
            NumDownwellings: 6,
            PowerLawExponent: 2.0,      // More extreme size variation
            StressThreshold: 0.4,       // Larger plates
            MicroPlateThreshold: 0.15,
            UseHierarchicalPlacement: true,
        }
        settings.NumPlates = 12
        settings.MajorPlateRatio = 0.50  // More major plates
        settings.TargetContinentalProportion = 0.40

    case Archipelago:
        // Many small plates → island chains
        settings.ConvectionConfig = ConvectionSettings{
            NumUpwellings: 15,          // Many spreading centers
            NumDownwellings: 20,
            PowerLawExponent: 1.2,      // Less extreme variation
            StressThreshold: 0.6,       // Smaller plates
            MicroPlateThreshold: 0.25,
            UseHierarchicalPlacement: false,
        }
        settings.NumPlates = 40
        settings.MajorPlateRatio = 0.15  // Fewer major plates
        settings.TargetContinentalProportion = 0.20

    case Waterworld:
        // Mostly oceanic
        settings.ConvectionConfig = ConvectionSettings{
            NumUpwellings: 10,
            NumDownwellings: 12,
            PowerLawExponent: 1.5,
            StressThreshold: 0.5,
            MicroPlateThreshold: 0.20,
            UseHierarchicalPlacement: true,
        }
        settings.NumPlates = 18
        settings.MajorPlateRatio = 0.30
        settings.TargetContinentalProportion = 0.05  // Very low

    case ShatteredWorld:
        // Extreme fragmentation → many micro-plates
        settings.ConvectionConfig = ConvectionSettings{
            NumUpwellings: 20,
            NumDownwellings: 30,
            PowerLawExponent: 1.0,      // Less size variation
            StressThreshold: 0.7,       // Smaller plates
            MicroPlateThreshold: 0.30,
            UseHierarchicalPlacement: false,
        }
        settings.NumPlates = 60
        settings.MajorPlateRatio = 0.10
        settings.TargetContinentalProportion = 0.25
    }

    return settings
}
```

### 4.2 Continuous Parameters

```go
// Allow fine-tuning between presets
type WorldTypeParameters struct {
    BaseType              WorldType

    // Continuous adjustment factors (0.5 - 2.0)
    SizeVariationFactor   float64  // Adjust power law exponent
    PlateCountFactor      float64  // Multiply plate counts
    ContinentalFactor     float64  // Adjust continental proportion
    FragmentationFactor   float64  // More/fewer micro-plates
}

func ApplyWorldTypeParameters(params WorldTypeParameters) TectonicSettings {
    baseSettings := GetWorldTypeSettings(params.BaseType, DefaultSettings())

    // Adjust parameters continuously
    baseSettings.ConvectionConfig.NumUpwellings = int(
        float64(baseSettings.ConvectionConfig.NumUpwellings) * params.PlateCountFactor)

    baseSettings.ConvectionConfig.PowerLawExponent *= params.SizeVariationFactor

    baseSettings.TargetContinentalProportion *= params.ContinentalFactor

    // Clamp to valid ranges
    baseSettings = ClampToValidRanges(baseSettings)

    return baseSettings
}
```

**Deliverables:**
- ✅ 5 world type presets validated
- ✅ Continuous parameter system
- ✅ Documentation with examples
- ✅ Visual comparison of world types

---

## Phase 5: Integration & Documentation (Week 8+)

### 5.1 Integration with Existing Pipeline

```go
// Update landgen/tectonics/generation.go

func InitializeTectonicPlates(
    voronoiCells []VoronoiCell,
    voronoiVertices []Vector3D,
    icosphereSites []Vector3D,
    settings TectonicSettings,
) ([]TectonicPlate, []int32) {

    // Route to appropriate generation method
    switch settings.GenerationMethod {
    case "mantle_convection":
        return InitializeTectonicPlatesMantle Convection(
            voronoiCells, voronoiVertices, icosphereSites, settings)

    case "accretion":  // Current method
        return InitializeTectonicPlatesAccretion(
            voronoiCells, voronoiVertices, icosphereSites, settings)

    default:
        // Default to mantle convection if it scores better
        return InitializeTectonicPlatesMantle Convection(
            voronoiCells, voronoiVertices, icosphereSites, settings)
    }
}
```

### 5.2 Configuration Files

```json
// configs/earth_like.json
{
  "planet_radius": 6370000,
  "num_plates": 20,
  "generation_method": "mantle_convection",
  "world_type": "earth",
  "seed": 42,

  "convection_config": {
    "num_upwellings": 8,
    "num_downwellings": 12,
    "strength_distribution": "power_law",
    "power_law_exponent": 1.5,
    "stress_threshold": 0.5,
    "micro_plate_threshold": 0.20,
    "use_hierarchical_placement": true,
    "large_cell_radius": 2500000,
    "medium_cell_radius": 1500000,
    "small_cell_radius": 800000
  },

  "target_continental_proportion": 0.29,
  "major_plate_ratio": 0.30
}
```

### 5.3 Documentation

Create comprehensive documentation:
- `docs/plate_generation_theory.md` - Explains the science
- `docs/parameter_guide.md` - How to tune parameters
- `docs/world_types.md` - Preset descriptions and examples
- `docs/evaluation_metrics.md` - Understanding the scores
- Tutorial notebook demonstrating workflow

**Deliverables:**
- ✅ Clean API with backward compatibility
- ✅ Example configs for all world types
- ✅ Complete documentation
- ✅ Tutorial/examples

---

## Success Metrics

### Minimum Success (Phase 2 complete)
- ✅ Earth benchmark score > 0.65 (vs current 0.37)
- ✅ Power law fit R² > 0.85
- ✅ 6-8 major plates, 10-15 minor plates
- ✅ Natural convex boundaries

### Target Success (Phase 3 complete)
- ✅ Earth benchmark score > 0.75
- ✅ Power law fit R² > 0.90
- ✅ Plate count matches Earth ±2 per category
- ✅ Gini coefficient ≈ 0.70-0.75
- ✅ Size ratio > 500x

### Stretch Success (Phase 4 complete)
- ✅ Earth benchmark score > 0.85
- ✅ Multiple validated world types
- ✅ Parameterized control for custom worlds
- ✅ Visual realism comparable to Earth

---

## Timeline Summary

| Phase | Duration | Key Deliverable |
|-------|----------|-----------------|
| 1. Evaluation Framework | 2 weeks | Complete metrics & visualization system |
| 2. Mantle Convection | 3 weeks | New generation method with power law |
| 3. Parameter Tuning | 2 weeks | Optimal Earth-like parameters |
| 4. World Types | 1 week | Presets for different planet types |
| 5. Integration | 1+ weeks | Production-ready implementation |

**Total: 8-10 weeks for complete implementation**

---

## Next Immediate Steps

1. **This Week**: Implement core evaluation framework
   - Create `evaluation/` package structure
   - Implement power law analysis
   - Build Earth benchmark scoring
   - Test on current implementation (establish baseline)

2. **Week 2**: Complete evaluation system
   - Add all visualizations
   - Build HTML report generator
   - Create standalone evaluation tool
   - Validate against Earth data

3. **Week 3**: Begin mantle convection implementation
   - Convection cell generation
   - Stress field calculation
   - Initial plate growing algorithm

Would you like me to start with Phase 1 (evaluation framework) implementation?
