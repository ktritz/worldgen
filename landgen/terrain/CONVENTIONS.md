# Code Conventions for Terrain Module

> These conventions ensure AI sessions produce consistent, maintainable code.
> Read this file at the start of every session.

---

## Critical Principles

### 1. Reproducibility First

Every function that involves randomness MUST accept a seed parameter. The same seed MUST produce identical output.

```go
// Good - seed is explicit
func GenerateMask(sites []Vector3D, settings MaskSettings) []float64 {
    rng := rand.New(rand.NewSource(settings.Seed))
    // ...
}

// Bad - uses global random state
func GenerateMask(sites []Vector3D, settings MaskSettings) []float64 {
    value := rand.Float64()  // Non-deterministic!
    // ...
}
```

### 2. Validate Early

Settings must be validated before use. Create a `Validate()` method for each settings struct:

```go
func (s MaskSettings) Validate() error {
    if s.ContinentalFrequency <= 0 {
        return fmt.Errorf("continentalFrequency must be positive, got %f", s.ContinentalFrequency)
    }
    if s.WarpAmplitude < 0 || s.WarpAmplitude > 1 {
        return fmt.Errorf("warpAmplitude must be in [0,1], got %f", s.WarpAmplitude)
    }
    return nil
}
```

### 3. Stage Independence

Each pipeline stage should be testable in isolation. Use interfaces:

```go
// Stage interfaces allow testing with mock inputs
type MaskGenerator interface {
    Generate(sites []Vector3D) ([]float64, error)
}

type ElevationGenerator interface {
    Generate(sites []Vector3D, mask []float64) ([]float64, error)
}
```

### 4. Fail Fast, Log Context

When something goes wrong, fail immediately with enough context to debug:

```go
if landPct < 0.10 || landPct > 0.50 {
    return nil, fmt.Errorf("land coverage %.1f%% outside valid range [10%%, 50%%] - check ContinentalFrequency (current: %.2f)",
        landPct*100, settings.ContinentalFrequency)
}
```

---

## File Size & Organization

### Target: 400-500 lines per file (hard max: 600)

If a file grows beyond this:
1. Identify logical groupings of functions
2. Extract to a new file with a clear name
3. Keep related types together

### File Naming

| Pattern | Example | Contents |
|---------|---------|----------|
| `types.go` | `types.go` | All structs, interfaces, constants |
| `<noun>.go` | `metrics.go` | Functions operating on that concept |
| `<noun>_<aspect>.go` | `coastline_analysis.go` | Subset of functions for a specific aspect |
| `test_<name>.go` | `test_metrics.go` | Test program (main package) |

### Standard File Structure

```go
package terrain

import (
    // Standard library first
    "fmt"
    "math"

    // Then external packages
    "github.com/kyroy/kdtree"

    // Then internal packages
    "worldgen/icosphere"
    "worldgen/procnoise"
)

// --- Section Header (use sparingly) ---

// FunctionName does X.
// Detailed description if needed.
func FunctionName(params) returns {
    // Implementation
}
```

---

## Type Definitions

### Type Aliases (in types.go)

Alias external types at the top of `types.go`:

```go
// --- Type Aliases for External Dependencies ---

// Vector3D uses the definition from the icosphere package.
type Vector3D = icosphere.Vector3D

// VoronoiCell uses the definition from the icosphere package.
type VoronoiCell = icosphere.VoronoiCell
```

### Constants with Types

Group constants with their type definitions:

```go
// TerrainType classifies terrain for elevation scaling.
type TerrainType int

const (
    TerrainOcean TerrainType = iota
    TerrainShelf
    TerrainCoast
    TerrainLowland
    TerrainHighland
    TerrainMountain
)
```

### Settings Structs

Use JSON tags for all settings (enables config file loading):

```go
// MaskSettings controls continental mask generation.
type MaskSettings struct {
    Seed                 int64   `json:"seed"`
    ContinentalFrequency float64 `json:"continentalFrequency"` // Size of continents (1.0-3.0)
    WarpAmplitude        float64 `json:"warpAmplitude"`        // Coastline irregularity (0.1-0.5)
    WarpFrequency        float64 `json:"warpFrequency"`        // Scale of wiggles (2.0-8.0)
    Octaves              int     `json:"octaves"`              // FBM octave count
    Verbose              bool    `json:"verbose"`              // Enable logging
}
```

### Default Settings Functions

Provide defaults for every settings struct:

```go
// DefaultMaskSettings returns Earth-like defaults.
func DefaultMaskSettings() MaskSettings {
    return MaskSettings{
        Seed:                 42,
        ContinentalFrequency: 2.0,
        WarpAmplitude:        0.3,
        WarpFrequency:        4.0,
        Octaves:              3,
        Verbose:              false,
    }
}
```

---

## Function Patterns

### Public API Functions

Large orchestrating functions should:
1. Log progress with step numbers
2. Check for Verbose flag before logging
3. Return errors rather than panicking

```go
// GenerateTerrain runs the complete terrain generation pipeline.
func GenerateTerrain(sites []Vector3D, settings TerrainSettings) ([]float64, error) {
    if settings.Verbose {
        fmt.Println("=== TERRAIN GENERATION ===")
    }

    // Step 1: Generate continental mask
    if settings.Verbose {
        fmt.Println("Step 1: Generating continental mask...")
    }
    mask, err := GenerateContinentalMask(sites, settings.Mask)
    if err != nil {
        return nil, fmt.Errorf("continental mask failed: %w", err)
    }

    // Step 2: ...
}
```

### Internal Helper Functions

Keep helpers small and focused:

```go
// sumValues returns the sum of all values in the slice.
func sumValues(vals []float64) float64 {
    sum := 0.0
    for _, v := range vals {
        sum += v
    }
    return sum
}
```

### Methods on Types

Use methods for operations that belong to a type:

```go
// IsLand returns true if this site is above sea level.
func (m *TerrainMetrics) IsLand(elevation float64) bool {
    return elevation > 0
}
```

---

## Error Handling

### Return errors, don't panic

```go
// Good
func LoadSettings(path string) (TerrainSettings, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return TerrainSettings{}, fmt.Errorf("reading settings file: %w", err)
    }
    // ...
}

// Bad
func LoadSettings(path string) TerrainSettings {
    data, err := os.ReadFile(path)
    if err != nil {
        panic(err)  // Don't do this
    }
    // ...
}
```

### Wrap errors with context

```go
if err != nil {
    return fmt.Errorf("generating mask at octave %d: %w", i, err)
}
```

---

## Comments & Documentation

### Doc Comments

Every exported function, type, and constant needs a doc comment:

```go
// TerrainMetrics contains all computed metrics for terrain evaluation.
// Use EvaluateTerrain() to compute these from elevation data.
type TerrainMetrics struct {
```

### Inline Comments

Use inline comments for non-obvious logic:

```go
// Normalize to sphere surface (domain warping can push points off-sphere)
warped = warped.Normalize()

// Earth's hypsometric curve shows 70.8% below sea level
const seaLevelCumulative = 0.708
```

### Section Headers

Use sparingly to separate major sections in longer files:

```go
// --- Metric Calculations ---

// --- Scoring Functions ---

// --- Helper Functions ---
```

---

## Testing Patterns

### Two Types of Tests

**1. Unit Tests (`*_test.go`)** - For validating calculations:

```go
// metrics_test.go
package terrain

import (
    "testing"
    "math"
)

func TestLandCoverageCalculation(t *testing.T) {
    // Known input: 3 land sites, 7 ocean sites
    elevation := []float64{100, 200, 50, -100, -200, -300, -400, -500, -600, -700}

    metrics := ComputeMetrics(elevation)

    expected := 0.30  // 30%
    if math.Abs(metrics.LandCoverage - expected) > 0.001 {
        t.Errorf("LandCoverage = %f, want %f", metrics.LandCoverage, expected)
    }
}
```

Run with: `go test ./landgen/terrain/...`

**2. Integration Tests (`test_*.go`)** - For visual validation and full pipeline:

```go
// test_mask.go
package main

func main() {
    // Generate and visualize
    // Save output image
    // Print metrics
}
```

Run with: `go build -o test_mask test_mask.go && ./test_mask`

### Test Output Format

Use consistent output formatting:

```go
fmt.Println("=== TEST NAME ===")
fmt.Printf("  Parameter: %v\n", value)
fmt.Printf("  Result: %.2f%% (target: %.2f%%)\n", actual, target)
if passed {
    fmt.Println("  ✓ PASS")
} else {
    fmt.Println("  ✗ FAIL")
}
```

---

## Visualization & Output

### Output Directory Structure

All generated outputs go in `output/` subdirectory (gitignored):

```
landgen/terrain/
├── output/                    # Generated files (gitignored)
│   ├── mask_seed42.png        # Continental mask visualization
│   ├── elevation_seed42.png   # Elevation map
│   ├── hypsometric_seed42.png # Hypsometric curve comparison
│   └── results.json           # Machine-readable metrics history
└── ...
```

### Visualization Function Pattern

Every major output should have a visualization function:

```go
// VisualizeMask saves a PNG of the continental mask.
// Returns the filepath written.
func VisualizeMask(sites []Vector3D, mask []float64, seed int64) (string, error) {
    filename := fmt.Sprintf("output/mask_seed%d.png", seed)
    // ... generate image ...
    return filename, nil
}
```

### Map Projection

Use equirectangular projection for consistency (matches existing server):

```go
// ProjectToEquirectangular converts 3D sphere coordinates to 2D image coordinates.
func ProjectToEquirectangular(v Vector3D, width, height int) (x, y int) {
    lon := math.Atan2(v.Y, v.X)           // -π to π
    lat := math.Asin(v.Z)                  // -π/2 to π/2
    x = int((lon/math.Pi + 1) * 0.5 * float64(width-1))
    y = int((0.5 - lat/math.Pi) * float64(height-1))
    return x, y
}
```

---

## Regression Tracking

### Results Log

Every test run appends to `output/results.json`:

```go
type TestResult struct {
    Timestamp   time.Time       `json:"timestamp"`
    Seed        int64           `json:"seed"`
    GitCommit   string          `json:"gitCommit,omitempty"`
    Score       float64         `json:"score"`
    Metrics     TerrainMetrics  `json:"metrics"`
    Settings    TerrainSettings `json:"settings"`
    Notes       string          `json:"notes,omitempty"`
}

// AppendResult adds a result to the log file.
func AppendResult(result TestResult) error {
    // Read existing, append, write back
}
```

### Score History

Track best scores in SESSION_CONTEXT.md:

```markdown
## Score History

| Date | Seed | Score | Notes |
|------|------|-------|-------|
| 2024-12-20 | 42 | 68.2% | Initial mask implementation |
| 2024-12-21 | 42 | 72.1% | Tuned warp amplitude |
```

### Regression Detection

Before merging changes, verify score doesn't drop:

```go
func CheckForRegression(newScore, previousBest float64) error {
    if newScore < previousBest - 1.0 {  // Allow 1% variance
        return fmt.Errorf("regression detected: %.1f%% < %.1f%% (previous best)",
            newScore, previousBest)
    }
    return nil
}
```

---

## Earth Reference Constants

Keep all Earth reference values in a single location (`types.go` or `constants.go`):

```go
// --- Earth Reference Values ---
// Source: Various geological surveys, see TERRAIN_PLAN.md for citations

const (
    EarthLandCoverage      = 0.292  // 29.2% of surface
    EarthMeanLandElevation = 840.0  // meters
    EarthMeanOceanDepth    = -3688.0 // meters
    EarthMaxElevation      = 8848.0  // Mount Everest
    EarthMinElevation      = -10994.0 // Mariana Trench
    EarthMountainCoverage  = 0.02   // 2% above 3000m
    EarthShelfCoverage     = 0.08   // 8% between 0 and -200m
    EarthCoastlineFractalD = 1.30   // Fractal dimension
)
```

---

## Parallelization Pattern

For expensive operations over all sites:

```go
func processParallel(sites []Vector3D, fn func(Vector3D) float64) []float64 {
    result := make([]float64, len(sites))
    numWorkers := runtime.NumCPU()

    var wg sync.WaitGroup
    chunkSize := (len(sites) + numWorkers - 1) / numWorkers

    for w := 0; w < numWorkers; w++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            start := workerID * chunkSize
            end := min(start + chunkSize, len(sites))

            for i := start; i < end; i++ {
                result[i] = fn(sites[i])
            }
        }(w)
    }

    wg.Wait()
    return result
}
```

---

## Import Organization

Always organize imports in this order:

```go
import (
    // 1. Standard library
    "fmt"
    "math"
    "sync"

    // 2. External packages (blank line separator)
    "github.com/kyroy/kdtree"

    // 3. Internal packages (blank line separator)
    "worldgen/icosphere"
    "worldgen/procnoise"
)
```

---

## API Stability

### Public Function Signatures

Once a public function is used by other code, its signature should not change without:
1. Updating all callers
2. Noting the change in SESSION_CONTEXT.md under "Breaking Changes"

If you need different behavior, prefer:
- Adding optional fields to settings structs (with zero-value defaults)
- Creating a new function variant (`GenerateMaskV2`)
- Using functional options pattern for complex cases

### Dependency Direction

```
types.go          <- imported by everything
    ↓
metrics.go        <- no internal dependencies except types
    ↓
continental_mask.go  <- may use metrics for validation
    ↓
base_elevation.go    <- uses mask output
    ↓
terrain_detail.go    <- uses elevation output
    ↓
generation.go        <- orchestrates all stages
```

Never create circular dependencies. If two files need to share code, extract to a third file.

---

## Checklist for New Files

Before committing a new file, verify:

- [ ] File is under 500 lines
- [ ] Package declaration matches directory
- [ ] Imports are organized (stdlib, external, internal)
- [ ] All exported items have doc comments
- [ ] Settings structs have JSON tags
- [ ] Settings structs have `Default*()` functions
- [ ] Settings structs have `Validate()` methods
- [ ] All random operations use seeded RNG
- [ ] Errors are returned, not panicked
- [ ] Errors include context (parameter values, etc.)
- [ ] Verbose logging uses `if settings.Verbose`
- [ ] Earth reference values use named constants
- [ ] Unit tests exist for non-trivial calculations

---

## Checklist for AI Sessions

At the start of each session:

1. Read `SESSION_CONTEXT.md` for current state
2. Read this file (`CONVENTIONS.md`) for patterns
3. Check the "Current Task" in SESSION_CONTEXT.md
4. Note the current best score in Score History
5. Follow the conventions above

At the end of each session:

1. Run tests: `go test ./landgen/terrain/...`
2. Run integration test and record score
3. Update `SESSION_CONTEXT.md`:
   - Implementation status table
   - Current task
   - Score history (if score changed)
   - Session log entry
4. Check for regressions (score shouldn't drop >1%)
5. Note any breaking changes or new patterns
