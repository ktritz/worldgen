// Package terrain implements noise-first terrain generation with domain warping.
// It produces irregular coastlines and Earth-like elevation distributions
// through a 3-stage pipeline: continental mask, base elevation, and terrain detail.
package terrain

import (
	"fmt"
	"math"

	"worldgen/icosphere"
)

// --- Type Aliases for External Dependencies ---

// Vector3D uses the definition from the icosphere package.
type Vector3D = icosphere.Vector3D

// VoronoiCell uses the definition from the icosphere package.
type VoronoiCell = icosphere.VoronoiCell

// --- Earth Reference Constants ---
// Source: Various geological surveys and hypsometric data

const (
	// Coverage metrics
	EarthLandCoverage      = 0.292 // 29.2% of surface
	EarthOceanCoverage     = 0.708 // 70.8% of surface
	EarthMountainCoverage  = 0.020 // 2% above 3000m
	EarthDeepOceanCoverage = 0.160 // 16.0% below -5000m
	EarthShelfCoverage     = 0.054 // 5.4% between 0 and -200m

	// Elevation statistics
	EarthMeanLandElevation = 840.0    // meters
	EarthMeanOceanDepth    = -3688.0  // meters
	EarthGlobalMean        = -2430.0  // meters
	EarthGlobalStdDev      = 2500.0   // meters
	EarthMaxElevation      = 8848.0   // Mount Everest
	EarthMinElevation      = -10994.0 // Mariana Trench

	// Hypsometric bimodal peaks
	EarthBimodalPeakOcean = -4200.0 // meters
	EarthBimodalPeakLand  = 300.0   // meters

	// Coastline metrics
	EarthCoastlineFractalD   = 1.30 // Fractal dimension (1.0 = smooth, 2.0 = space-filling)
	EarthTortuosityRatio     = 3.0  // Actual length / straight-line distance
	EarthMajorLandmasses     = 7    // Number of major continents
	EarthLargestContinentPct = 0.30 // Eurasia as % of land
	EarthContinentGini       = 0.45 // Size inequality measure
)

// HypsometricTargets contains Earth's cumulative elevation distribution at key thresholds.
// Key: elevation threshold in meters, Value: cumulative percentage below that threshold.
var HypsometricTargets = map[float64]float64{
	-6000: 0.010, // 1.0% below -6000m
	-5000: 0.160, // 16.0% below -5000m
	-4000: 0.390, // 39.0% below -4000m
	-3000: 0.532, // 53.2% below -3000m
	-200:  0.654, // 65.4% below the shelf break
	0:     0.708, // 70.8% below sea level
	500:   0.825, // 82.5% below 500m
	1000:  0.902, // 90.2% below 1000m
	2000:  0.960, // 96.0% below 2000m
	3000:  0.980, // 98.0% below 3000m
}

// --- Terrain Classification ---

// TerrainType classifies terrain for elevation scaling and detail amplitude.
type TerrainType int

const (
	TerrainDeepOcean TerrainType = iota // Below -5000m
	TerrainOcean                        // -5000m to -200m
	TerrainShelf                        // -200m to 0m
	TerrainCoast                        // 0m to 200m
	TerrainLowland                      // 200m to 1000m
	TerrainHighland                     // 1000m to 3000m
	TerrainMountain                     // Above 3000m
)

// String returns the terrain type name.
func (t TerrainType) String() string {
	names := []string{"DeepOcean", "Ocean", "Shelf", "Coast", "Lowland", "Highland", "Mountain"}
	if int(t) < len(names) {
		return names[t]
	}
	return "Unknown"
}

// ClassifyTerrain returns the terrain type for a given elevation.
func ClassifyTerrain(elevation float64) TerrainType {
	switch {
	case elevation < -5000:
		return TerrainDeepOcean
	case elevation < -200:
		return TerrainOcean
	case elevation < 0:
		return TerrainShelf
	case elevation < 200:
		return TerrainCoast
	case elevation < 1000:
		return TerrainLowland
	case elevation < 3000:
		return TerrainHighland
	default:
		return TerrainMountain
	}
}

// --- Settings Structs ---

// MaskSettings controls continental mask generation (Stage 1).
type MaskSettings struct {
	Seed                 int64   `json:"seed"`
	ContinentalFrequency float64 `json:"continentalFrequency"` // Size of continents (1.0-3.0)
	WarpAmplitude        float64 `json:"warpAmplitude"`        // Coastline irregularity (0.1-0.5)
	WarpFrequency        float64 `json:"warpFrequency"`        // Scale of wiggles (2.0-8.0)
	Octaves              int     `json:"octaves"`              // FBM octave count (2-4)
	Lacunarity           float64 `json:"lacunarity"`           // Frequency multiplier per octave
	Persistence          float64 `json:"persistence"`          // Amplitude multiplier per octave
	TargetLandCoverage   float64 `json:"targetLandCoverage"`   // Target land percentage (0.27-0.32)
	Verbose              bool    `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s MaskSettings) Validate() error {
	if s.ContinentalFrequency <= 0 || s.ContinentalFrequency > 10 {
		return fmt.Errorf("continentalFrequency must be in (0, 10], got %f", s.ContinentalFrequency)
	}
	if s.WarpAmplitude < 0 || s.WarpAmplitude > 1 {
		return fmt.Errorf("warpAmplitude must be in [0, 1], got %f", s.WarpAmplitude)
	}
	if s.WarpFrequency <= 0 || s.WarpFrequency > 20 {
		return fmt.Errorf("warpFrequency must be in (0, 20], got %f", s.WarpFrequency)
	}
	if s.Octaves < 1 || s.Octaves > 8 {
		return fmt.Errorf("octaves must be in [1, 8], got %d", s.Octaves)
	}
	if s.Lacunarity <= 0 {
		return fmt.Errorf("lacunarity must be positive, got %f", s.Lacunarity)
	}
	if s.Persistence <= 0 || s.Persistence >= 1 {
		return fmt.Errorf("persistence must be in (0, 1), got %f", s.Persistence)
	}
	if s.TargetLandCoverage < 0.1 || s.TargetLandCoverage > 0.5 {
		return fmt.Errorf("targetLandCoverage must be in [0.1, 0.5], got %f", s.TargetLandCoverage)
	}
	return nil
}

// DefaultMaskSettings returns Earth-like defaults for continental mask generation.
func DefaultMaskSettings() MaskSettings {
	return MaskSettings{
		Seed:                 42,
		ContinentalFrequency: 0.5, // Very low frequency for large continents
		WarpAmplitude:        0.1, // Subtle warp for irregular coastlines
		WarpFrequency:        2.0, // Warp detail scale
		Octaves:              2,   // Few octaves for coherent landmasses
		Lacunarity:           2.0,
		Persistence:          0.4, // Reduce high-frequency contribution
		TargetLandCoverage:   EarthLandCoverage,
		Verbose:              false,
	}
}

// ElevationSettings controls base elevation assignment (Stage 2).
type ElevationSettings struct {
	Seed                 int64   `json:"seed"`
	ContinentalBase      float64 `json:"continentalBase"`      // Base continental height (m)
	ContinentalVariation float64 `json:"continentalVariation"` // Max additional height based on inlandness
	OceanicBase          float64 `json:"oceanicBase"`          // Base ocean depth (m, negative)
	OceanicVariation     float64 `json:"oceanicVariation"`     // Max additional depth
	ShelfWidth           float64 `json:"shelfWidth"`           // Mask range for shelf transition
	ShelfDepth           float64 `json:"shelfDepth"`           // Max shelf depth (m, negative)
	Verbose              bool    `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s ElevationSettings) Validate() error {
	if s.ContinentalBase < 0 {
		return fmt.Errorf("continentalBase must be non-negative, got %f", s.ContinentalBase)
	}
	if s.ContinentalVariation < 0 {
		return fmt.Errorf("continentalVariation must be non-negative, got %f", s.ContinentalVariation)
	}
	if s.OceanicBase > 0 {
		return fmt.Errorf("oceanicBase must be non-positive, got %f", s.OceanicBase)
	}
	if s.OceanicVariation < 0 {
		return fmt.Errorf("oceanicVariation must be non-negative, got %f", s.OceanicVariation)
	}
	if s.ShelfWidth <= 0 || s.ShelfWidth > 0.5 {
		return fmt.Errorf("shelfWidth must be in (0, 0.5], got %f", s.ShelfWidth)
	}
	if s.ShelfDepth > 0 {
		return fmt.Errorf("shelfDepth must be non-positive, got %f", s.ShelfDepth)
	}
	return nil
}

// DefaultElevationSettings returns Earth-like defaults for base elevation.
func DefaultElevationSettings() ElevationSettings {
	return ElevationSettings{
		Seed:                 42,
		ContinentalBase:      400.0,   // Base height before mountains
		ContinentalVariation: 500.0,   // Variation based on inlandness
		OceanicBase:          -3900.0, // Tuned for 95%+ score
		OceanicVariation:     1800.0,  // Variation for trenches
		ShelfWidth:           0.08,    // Shelf transition width
		ShelfDepth:           -200.0,
		Verbose:              false,
	}
}

// MountainSettings controls mountain generation.
type MountainSettings struct {
	Seed              int64   `json:"seed"`
	MountainFrequency float64 `json:"mountainFrequency"` // Noise frequency for mountain potential
	MountainThreshold float64 `json:"mountainThreshold"` // Threshold above which mountains form
	MaxMountainHeight float64 `json:"maxMountainHeight"` // Maximum mountain peak height (m)
	TargetCoverage    float64 `json:"targetCoverage"`    // Target mountain coverage (fraction of land)
	Verbose           bool    `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s MountainSettings) Validate() error {
	if s.MountainFrequency <= 0 || s.MountainFrequency > 20 {
		return fmt.Errorf("mountainFrequency must be in (0, 20], got %f", s.MountainFrequency)
	}
	if s.MountainThreshold < 0 || s.MountainThreshold > 1 {
		return fmt.Errorf("mountainThreshold must be in [0, 1], got %f", s.MountainThreshold)
	}
	if s.MaxMountainHeight <= 0 || s.MaxMountainHeight > 15000 {
		return fmt.Errorf("maxMountainHeight must be in (0, 15000], got %f", s.MaxMountainHeight)
	}
	if s.TargetCoverage < 0 || s.TargetCoverage > 0.2 {
		return fmt.Errorf("targetCoverage must be in [0, 0.2], got %f", s.TargetCoverage)
	}
	return nil
}

// DefaultMountainSettings returns Earth-like defaults for mountain generation.
func DefaultMountainSettings() MountainSettings {
	return MountainSettings{
		Seed:              42,
		MountainFrequency: 5.0,
		MountainThreshold: 0.75, // Tuned for ~2% mountain coverage
		MaxMountainHeight: 7000.0,
		TargetCoverage:    EarthMountainCoverage,
		Verbose:           false,
	}
}

// DetailSettings controls terrain detail generation (Stage 3).
type DetailSettings struct {
	Seed                 int64   `json:"seed"`
	DetailFrequency      float64 `json:"detailFrequency"`      // Base frequency for detail noise
	DetailOctaves        int     `json:"detailOctaves"`        // Number of FBM octaves
	Lacunarity           float64 `json:"lacunarity"`           // Frequency multiplier per octave
	Persistence          float64 `json:"persistence"`          // Amplitude multiplier per octave
	BaseAmplitude        float64 `json:"baseAmplitude"`        // Base amplitude for detail (m)
	MountainDetailScale  float64 `json:"mountainDetailScale"`  // Multiplier for mountain areas
	DeepOceanDetailScale float64 `json:"deepOceanDetailScale"` // Multiplier for deep ocean
	Verbose              bool    `json:"verbose"`
}

// Validate checks that all settings are within acceptable ranges.
func (s DetailSettings) Validate() error {
	if s.DetailFrequency <= 0 || s.DetailFrequency > 50 {
		return fmt.Errorf("detailFrequency must be in (0, 50], got %f", s.DetailFrequency)
	}
	if s.DetailOctaves < 1 || s.DetailOctaves > 10 {
		return fmt.Errorf("detailOctaves must be in [1, 10], got %d", s.DetailOctaves)
	}
	if s.Lacunarity <= 0 {
		return fmt.Errorf("lacunarity must be positive, got %f", s.Lacunarity)
	}
	if s.Persistence <= 0 || s.Persistence >= 1 {
		return fmt.Errorf("persistence must be in (0, 1), got %f", s.Persistence)
	}
	if s.BaseAmplitude < 0 {
		return fmt.Errorf("baseAmplitude must be non-negative, got %f", s.BaseAmplitude)
	}
	if s.MountainDetailScale < 0 {
		return fmt.Errorf("mountainDetailScale must be non-negative, got %f", s.MountainDetailScale)
	}
	if s.DeepOceanDetailScale < 0 {
		return fmt.Errorf("deepOceanDetailScale must be non-negative, got %f", s.DeepOceanDetailScale)
	}
	return nil
}

// DefaultDetailSettings returns Earth-like defaults for terrain detail.
func DefaultDetailSettings() DetailSettings {
	return DetailSettings{
		Seed:                 42,
		DetailFrequency:      8.0,
		DetailOctaves:        4,
		Lacunarity:           2.0,
		Persistence:          0.5,
		BaseAmplitude:        200.0,
		MountainDetailScale:  2.5,
		DeepOceanDetailScale: 0.5,
		Verbose:              false,
	}
}

// TerrainSettings is the composite settings for full terrain generation.
type TerrainSettings struct {
	Seed      int64             `json:"seed"` // Master seed (overrides individual seeds if non-zero)
	Mask      MaskSettings      `json:"mask"`
	Elevation ElevationSettings `json:"elevation"`
	Mountain  MountainSettings  `json:"mountain"`
	Detail    DetailSettings    `json:"detail"`
	Verbose   bool              `json:"verbose"`
}

// Validate checks all nested settings.
func (s TerrainSettings) Validate() error {
	if err := s.Mask.Validate(); err != nil {
		return fmt.Errorf("mask settings: %w", err)
	}
	if err := s.Elevation.Validate(); err != nil {
		return fmt.Errorf("elevation settings: %w", err)
	}
	if err := s.Mountain.Validate(); err != nil {
		return fmt.Errorf("mountain settings: %w", err)
	}
	if err := s.Detail.Validate(); err != nil {
		return fmt.Errorf("detail settings: %w", err)
	}
	return nil
}

// DefaultTerrainSettings returns Earth-like defaults for full terrain generation.
func DefaultTerrainSettings() TerrainSettings {
	return TerrainSettings{
		Seed:      42,
		Mask:      DefaultMaskSettings(),
		Elevation: DefaultElevationSettings(),
		Mountain:  DefaultMountainSettings(),
		Detail:    DefaultDetailSettings(),
		Verbose:   false,
	}
}

// ApplyMasterSeed propagates the master seed to all sub-settings if it's set.
// Each sub-setting gets a derived seed to ensure different but reproducible noise.
func (s *TerrainSettings) ApplyMasterSeed() {
	if s.Seed != 0 {
		s.Mask.Seed = s.Seed
		s.Elevation.Seed = s.Seed + 1000
		s.Mountain.Seed = s.Seed + 2000
		s.Detail.Seed = s.Seed + 3000
	}
	if s.Verbose {
		s.Mask.Verbose = true
		s.Elevation.Verbose = true
		s.Mountain.Verbose = true
		s.Detail.Verbose = true
	}
}

// --- Metrics Structs ---

// TerrainMetrics contains all computed metrics for terrain evaluation.
// Use ComputeMetricsWithCells() when mesh topology is available so coastline,
// continent, and relief metrics are populated as well.
type TerrainMetrics struct {
	// Primary coverage metrics (fraction of total surface)
	LandCoverage      float64 `json:"landCoverage"`
	OceanCoverage     float64 `json:"oceanCoverage"`
	MountainCoverage  float64 `json:"mountainCoverage"`  // Above 3000m
	DeepOceanCoverage float64 `json:"deepOceanCoverage"` // Below -5000m
	ShelfCoverage     float64 `json:"shelfCoverage"`     // 0 to -200m

	// Elevation statistics (meters)
	MeanLandElevation float64 `json:"meanLandElevation"`
	MeanOceanDepth    float64 `json:"meanOceanDepth"`
	GlobalMean        float64 `json:"globalMean"`
	GlobalStdDev      float64 `json:"globalStdDev"`
	MaxElevation      float64 `json:"maxElevation"`
	MinElevation      float64 `json:"minElevation"`
	MeanLocalRelief   float64 `json:"meanLocalRelief"`
	P95LocalRelief    float64 `json:"p95LocalRelief"`
	MountainClustered float64 `json:"mountainClustered"`

	// Hypsometric curve (cumulative % at thresholds)
	// Key: elevation threshold, Value: fraction below that elevation
	HypsometricCurve map[float64]float64 `json:"hypsometricCurve"`

	// Coastline metrics
	FractalDimension float64 `json:"fractalDimension"`
	TortuosityRatio  float64 `json:"tortuosityRatio"`

	// Continental distribution
	NumMajorLandmasses  int     `json:"numMajorLandmasses"`
	LargestContinentPct float64 `json:"largestContinentPct"`
	ContinentGini       float64 `json:"continentGini"`

	// Drainage / fluvial structure
	FluvialChannelCoverage  float64 `json:"fluvialChannelCoverage"`
	EndorheicCatchmentPct   float64 `json:"endorheicCatchmentPct"`
	InlandLakeCoverage      float64 `json:"inlandLakeCoverage"`
	NumMajorEndorheicBasins int     `json:"numMajorEndorheicBasins"`

	// Hotspot track metrics
	HotspotChainCount   int     `json:"hotspotChainCount"`
	HotspotSpacingCV    float64 `json:"hotspotSpacingCV"`
	HotspotBurstiness   float64 `json:"hotspotBurstiness"`
	HotspotBendFraction float64 `json:"hotspotBendFraction"`
}

// EvaluationResult contains the scoring output from terrain evaluation.
type EvaluationResult struct {
	Score         float64        `json:"score"`         // Overall score 0-100
	Metrics       TerrainMetrics `json:"metrics"`       // All computed metrics
	FailedMetrics []string       `json:"failedMetrics"` // Names of metrics outside acceptable range
	Passed        bool           `json:"passed"`        // True if score >= target and no critical failures
}

// HydrologyDiagnostics captures final DEM drainage-readiness signals after the
// terrain pipeline has applied detail noise and other late-stage edits.
type HydrologyDiagnostics struct {
	PostDetailBreachedSinks int                        `json:"postDetailBreachedSinks"`
	FluvialChannelCoverage  float64                    `json:"fluvialChannelCoverage"`
	EndorheicCatchmentPct   float64                    `json:"endorheicCatchmentPct"`
	InlandLakeCoverage      float64                    `json:"inlandLakeCoverage"`
	NumMajorEndorheicBasins int                        `json:"numMajorEndorheicBasins"`
	Regions                 []HydrologyRegionSummary   `json:"regions,omitempty"`
	Classes                 []HydrologyClassSummary    `json:"classes,omitempty"`
	Scaffold                *HydrologyScaffold         `json:"scaffold,omitempty"`
	TerrainRefinement       *TerrainRefinementScaffold `json:"terrainRefinement,omitempty"`
}

// HydrologyRegionSummary groups land cells by broad runoff regime so we can
// verify that wetter regions actually carry denser drainage than drier ones.
type HydrologyRegionSummary struct {
	Name                  string  `json:"name"`
	CellCount             int     `json:"cellCount"`
	MeanRunoff            float64 `json:"meanRunoff"`
	MeanAccumulation      float64 `json:"meanAccumulation"`
	ChannelCoverage       float64 `json:"channelCoverage"`
	EndorheicCatchmentPct float64 `json:"endorheicCatchmentPct"`
	InlandLakeReachPct    float64 `json:"inlandLakeReachPct"`
}

// HydrologyClassSummary counts coarse hydrology roles so review tooling can
// see whether a world is dominated by plausible headwaters/trunks/outlets
// rather than everything collapsing into one generic river cell type.
type HydrologyClassSummary struct {
	Class     string `json:"class"`
	CellCount int    `json:"cellCount"`
}

// HydrologyBoundaryFlow describes coarse water crossings through a cell
// boundary. These are the contracts a later high-resolution local generator
// must preserve so neighboring zoomed cells agree on where rivers enter/leave.
type HydrologyBoundaryFlow struct {
	InflowNeighbors   []int     `json:"inflowNeighbors,omitempty"`
	InflowBearingDeg  []float64 `json:"inflowBearingDeg,omitempty"`
	InflowStrength    []float64 `json:"inflowStrength,omitempty"`
	OutflowNeighbor   int       `json:"outflowNeighbor"`
	OutflowBearingDeg float64   `json:"outflowBearingDeg"`
	OutflowStrength   float64   `json:"outflowStrength"`
}

// HydrologyBoundarySideFlow aggregates gross flow by directional sector around
// a cell boundary. Fine-grained local mapping can split one sector into
// multiple concrete river crossings while preserving total coarse flow.
type HydrologyBoundarySideFlow struct {
	Sector           string  `json:"sector"`
	BearingCenterDeg float64 `json:"bearingCenterDeg"`
	InflowStrength   float64 `json:"inflowStrength"`
	OutflowStrength  float64 `json:"outflowStrength"`
}

// HydrologyScaffold stores the coarse routing graph and related fields that a
// later high-resolution local generator can use to deterministically synthesize
// rivers, lakes, floodplains, and outlet geometry inside a coarse cell.
type HydrologyScaffold struct {
	Receivers        []int                         `json:"receivers"`
	TerminalSinks    []int                         `json:"terminalSinks"`
	Runoff           []float64                     `json:"runoff"`
	Accumulation     []float64                     `json:"accumulation"`
	ChannelStrength  []float64                     `json:"channelStrength"`
	WaterBodyLabel   []int                         `json:"waterBodyLabel"`
	CellClass        []string                      `json:"cellClass,omitempty"`
	OutletMode       []string                      `json:"outletMode,omitempty"`
	MaxOutflows      []int                         `json:"maxOutflows,omitempty"`
	BoundaryFlow     []HydrologyBoundaryFlow       `json:"boundaryFlow,omitempty"`
	BoundarySideFlow [][]HydrologyBoundarySideFlow `json:"boundarySideFlow,omitempty"`
}

// TerrainBoundaryConstraint describes the coarse terrain state at a boundary to
// a neighboring cell. A later local DEM refinement should honor these anchors
// before adding subcell detail.
type TerrainBoundaryConstraint struct {
	Neighbor          int     `json:"neighbor"`
	BearingDeg        float64 `json:"bearingDeg"`
	BoundaryElevation float64 `json:"boundaryElevation"`
	NeighborElevation float64 `json:"neighborElevation"`
	CrossingClass     string  `json:"crossingClass,omitempty"`
	CrossingStrength  float64 `json:"crossingStrength,omitempty"`
}

// TerrainRefinementCellConstraint captures the low-frequency terrain contract
// for refining a single coarse cell into a higher-resolution local patch.
type TerrainRefinementCellConstraint struct {
	BaseElevation         float64                     `json:"baseElevation"`
	MeanNeighborElevation float64                     `json:"meanNeighborElevation"`
	LocalRelief           float64                     `json:"localRelief"`
	DownslopeBearingDeg   float64                     `json:"downslopeBearingDeg"`
	DownslopeStrength     float64                     `json:"downslopeStrength"`
	ChannelBearingDeg     float64                     `json:"channelBearingDeg"`
	ChannelStrength       float64                     `json:"channelStrength"`
	Boundary              []TerrainBoundaryConstraint `json:"boundary,omitempty"`
}

// TerrainRefinementScaffold holds per-cell terrain-shape constraints for local
// elevation synthesis. It complements, rather than replaces, the hydrology
// scaffold.
type TerrainRefinementScaffold struct {
	Cells []TerrainRefinementCellConstraint `json:"cells"`
}

// PlanetGenerationDiagnostics carries optional generator-side metadata useful
// for review tooling and higher-fidelity evaluation.
type PlanetGenerationDiagnostics struct {
	HotspotChains []HotspotChain       `json:"hotspotChains"`
	Hydrology     HydrologyDiagnostics `json:"hydrology"`
	// Not serialized: these are ten per-cell arrays, so caching them adds tens
	// of MB per seed at L7 and passes every element through the reflective
	// cache sanitizer. Nothing consumes them yet; give this a json tag when a
	// consumer lands and is worth the cache cost.
	Tectonics TectonicDiagnostics `json:"-"`
}

// TectonicDiagnostics exports the plate scaffolding that terrain generation
// already computes internally and previously discarded. It carries the plate
// assignment plus the per-cell distance fields to each boundary class, which is
// the structural context a later lithology / crustal-age pass needs without
// rerunning any part of the tectonic simulation.
//
// All Dist* fields are physical distances in great-circle radians on the unit
// sphere (multiply by the planet radius for length units). They are measured
// along the Voronoi neighbor graph and are therefore slightly longer than the
// straight great-circle separation, but they do not depend on mesh resolution:
// the internal hop counts are scaled by the mesh's mean neighbor spacing, so
// the same physical world exports the same values at L5 and at L8.
//
// Each field is restricted to the domain it is meaningful in: DistFromCoast,
// DistFromMountain, DistFromCollision, DistFromArc and DistFromRift propagate
// through continental cells only, while OceanDistFromCoast, DistFromRidge and
// DistFromTrench propagate through oceanic cells only. Cells outside a field's
// domain, or with no reachable seed of that class, hold NaN - see
// TectonicDistanceUndefined for why, and use TectonicDistanceIsDefined to test
// for it. NaN is not JSON-encodable; this struct is not serialized (see the
// json:"-" tag on PlanetGenerationDiagnostics.Tectonics), and a future consumer
// that wants it cached must map the undefined cells to something explicit
// rather than let review_planets' cache sanitizer silently rewrite NaN to 0
// (which would read as "on the boundary").
type TectonicDiagnostics struct {
	NumPlates        int    `json:"numPlates"`
	NumOceanicPlates int    `json:"numOceanicPlates"`
	PlateID          []int  `json:"plateId"`
	PlateIsOcean     []bool `json:"plateIsOcean"`
	// CellAngularSpacing is the mean center-to-center neighbor spacing of the
	// generating mesh, in radians. It is the quantum of the Dist* fields:
	// divide by it to recover hop counts.
	CellAngularSpacing float64 `json:"cellAngularSpacing"`
	CoastlineSeeds     int     `json:"coastlineSeeds"`
	MountainSeeds      int     `json:"mountainSeeds"`
	CollisionSeeds     int     `json:"collisionSeeds"`
	ArcSeeds           int     `json:"arcSeeds"`
	RidgeSeeds         int     `json:"ridgeSeeds"`
	TrenchSeeds        int     `json:"trenchSeeds"`
	RiftSeeds          int     `json:"riftSeeds"`
	// Distances in great-circle radians; NaN where undefined.
	DistFromCoast      []float64 `json:"distFromCoast"`
	OceanDistFromCoast []float64 `json:"oceanDistFromCoast"`
	DistFromMountain   []float64 `json:"distFromMountain"`
	DistFromCollision  []float64 `json:"distFromCollision"`
	DistFromArc        []float64 `json:"distFromArc"`
	DistFromRidge      []float64 `json:"distFromRidge"`
	DistFromTrench     []float64 `json:"distFromTrench"`
	DistFromRift       []float64 `json:"distFromRift"`
}

// --- Scoring Weights ---

// ScoringWeights defines the relative importance of each metric category.
type ScoringWeights struct {
	PrimaryMetrics   float64 `json:"primaryMetrics"`   // Coverage and mean elevation
	HypsometricCurve float64 `json:"hypsometricCurve"` // Distribution shape
	CoastlineMetrics float64 `json:"coastlineMetrics"` // Irregularity
	ContinentMetrics float64 `json:"continentMetrics"` // Distribution of landmasses
}

// DefaultScoringWeights returns standard weights that sum to 100.
func DefaultScoringWeights() ScoringWeights {
	return ScoringWeights{
		PrimaryMetrics:   60.0, // 60%
		HypsometricCurve: 25.0, // 25%
		CoastlineMetrics: 10.0, // 10%
		ContinentMetrics: 5.0,  // 5%
	}
}

// --- Tectonic Constants ---

const (
	// Plate tectonics simulation
	CollisionThreshold  = 0.5  // Threshold for normalized plate collision detection
	DivergenceThreshold = 0.15 // Threshold for normalized divergent motion at continental rifts
	DeltaTime           = 1e-2 // Time step for velocity simulation
	NoiseAmplitude      = 0.1  // Amplitude of elevation noise
	// VolcanoDistanceRadians is the angular distance (in radians) from trench to volcanic arc
	// Real Earth: ~100-300km, on unit sphere ~0.016-0.047 radians
	// We use 0.03 radians (~190km on Earth-sized planet)
	VolcanoDistanceRadians = 0.03
	// ArcSeedSpacingRadians keeps volcanic-arc seeds dense enough to form belts
	// instead of isolated peaks while avoiding overpainting every trench cell.
	ArcSeedSpacingRadians = 0.02
	// ArcHalfWidthRadians widens volcanic arcs into a narrow belt rather than a
	// single-cell thread.
	ArcHalfWidthRadians = 0.012
	// CollisionBeltDistanceRadians expands continent-continent sutures inland on
	// both sides, approximating broad fold-and-thrust belts.
	CollisionBeltDistanceRadians = 0.04
	// CollisionLinkDistanceRadians links nearby collision seeds along strike so
	// sutures read as continuous mountain systems instead of broken patches.
	CollisionLinkDistanceRadians = 0.035
	// ArcLinkDistanceRadians connects nearby volcanic-arc seeds into longer
	// coast-parallel chains on the overriding plate.
	ArcLinkDistanceRadians = 0.09
)

// PlateRotation represents a plate's rotational motion around an Euler pole.
// Real tectonic plates rotate around Euler poles (points on the sphere),
// creating curved velocity fields and realistic hotspot tracks.
type PlateRotation struct {
	// Pole is the Euler pole (rotation axis) as a unit vector
	// Points on the sphere move in circles around this axis
	Pole Vector3D

	// AngularVelocity in radians per unit time (positive = counterclockwise when viewed from pole)
	// Earth plates: ~0.01-0.1 radians/million years, we use normalized values
	AngularVelocity float64
}

// VelocityAt computes the linear velocity vector at a given position on the sphere.
// For rotation around an Euler pole: v = ω × r (cross product of rotation axis and position)
// The result is tangent to the sphere at that point.
func (pr PlateRotation) VelocityAt(pos Vector3D) Vector3D {
	// v = ω × r, where ω = angularVelocity * pole
	// Cross product: (a × b) = (a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x)
	omega := pr.AngularVelocity
	return Vector3D{
		X: omega * (pr.Pole.Y*pos.Z - pr.Pole.Z*pos.Y),
		Y: omega * (pr.Pole.Z*pos.X - pr.Pole.X*pos.Z),
		Z: omega * (pr.Pole.X*pos.Y - pr.Pole.Y*pos.X),
	}
}

// RotatePoint rotates a point around the Euler pole by angle radians.
// Uses Rodrigues' rotation formula.
func (pr PlateRotation) RotatePoint(pos Vector3D, angle float64) Vector3D {
	// Rodrigues' formula: v_rot = v*cos(θ) + (k × v)*sin(θ) + k*(k·v)*(1-cos(θ))
	// where k is the rotation axis (pole) and θ is the angle
	k := pr.Pole
	cosA := math.Cos(angle)
	sinA := math.Sin(angle)

	// k × v (cross product)
	cross := Vector3D{
		X: k.Y*pos.Z - k.Z*pos.Y,
		Y: k.Z*pos.X - k.X*pos.Z,
		Z: k.X*pos.Y - k.Y*pos.X,
	}

	// k · v (dot product)
	dot := k.X*pos.X + k.Y*pos.Y + k.Z*pos.Z

	// Apply Rodrigues' formula
	return Vector3D{
		X: pos.X*cosA + cross.X*sinA + k.X*dot*(1-cosA),
		Y: pos.Y*cosA + cross.Y*sinA + k.Y*dot*(1-cosA),
		Z: pos.Z*cosA + cross.Z*sinA + k.Z*dot*(1-cosA),
	}
}

// BoundaryType classifies plate boundary regions
type BoundaryType int

const (
	BoundaryNone      BoundaryType = iota
	BoundaryMountain               // Continental collision / volcanic arc
	BoundaryCoastline              // Ocean-land boundary
	BoundaryOcean                  // Generic ocean
	BoundaryRidge                  // Mid-ocean spreading ridge
	BoundaryTrench                 // Subduction trench
	BoundaryRift                   // Divergent continental boundary / inland sea
)

// --- Helper Functions ---

// IsLand returns true if the elevation is above sea level.
func IsLand(elevation float64) bool {
	return elevation > 0
}

// IsOcean returns true if the elevation is at or below sea level.
func IsOcean(elevation float64) bool {
	return elevation <= 0
}

// IsMountain returns true if the elevation is above mountain threshold (3000m).
func IsMountain(elevation float64) bool {
	return elevation > 3000
}

// IsDeepOcean returns true if the elevation is below deep ocean threshold (-5000m).
func IsDeepOcean(elevation float64) bool {
	return elevation < -5000
}

// IsShelf returns true if the elevation is in the continental shelf range (0 to -200m).
func IsShelf(elevation float64) bool {
	return elevation <= 0 && elevation > -200
}

// Clamp constrains a value to the range [min, max].
func Clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Lerp performs linear interpolation between a and b by t.
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// InverseLerp returns the t value for linear interpolation.
func InverseLerp(a, b, value float64) float64 {
	if math.Abs(b-a) < 1e-10 {
		return 0
	}
	return (value - a) / (b - a)
}

// SmoothStep performs smooth Hermite interpolation between 0 and 1.
func SmoothStep(edge0, edge1, x float64) float64 {
	t := Clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}
