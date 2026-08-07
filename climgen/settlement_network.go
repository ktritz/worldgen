package climgen

import (
	"container/heap"
	"fmt"
	"math"
	"sort"
)

type SettlementNodeKind int

const (
	SettlementNodeHamlet SettlementNodeKind = iota
	SettlementNodeVillage
	SettlementNodeTown
	SettlementNodeCity
)

func SettlementNodeKindName(kind SettlementNodeKind) string {
	names := []string{"Local Anchor", "District Anchor", "Regional Anchor", "Major Hub"}
	if int(kind) < len(names) {
		return names[kind]
	}
	return "Unknown"
}

type SettlementNode struct {
	ID                  int
	CellIndex           int
	Kind                SettlementNodeKind
	Score               float64
	CarryingCapacity    float64
	UrbanPotential      float64
	PhysicalSupportArea float64
	Coastal             bool
	River               bool
}

type SettlementLink struct {
	From       int
	To         int
	TravelCost float64
	Path       []int
}

type SettlementNetworkDiagnostics struct {
	MovementCost    []float64
	NodeByCell      []int
	RegionByNode    []int
	NodeFormation   SettlementNodeFormationDiagnostics
	LinkFormation   SettlementLinkFormationDiagnostics
	RegionFormation RegionFormationDiagnostics
	// KindCalibrationError is non-empty when the per-level kind calibration
	// table was rejected and the run fell back to the absolute settings
	// thresholds. A rejected table is a configuration bug, not a world
	// property: it is reported here rather than silently classifying against
	// garbage cut points.
	KindCalibrationError string
}

// SettlementFieldDistribution summarizes a per-land-cell support field so the
// same absolute thresholds can be compared across mesh resolutions.
type SettlementFieldDistribution struct {
	Count int
	Mean  float64
	P50   float64
	P90   float64
	P99   float64
	Max   float64
	// AboveThresholdFraction is indexed by SettlementNodeKind and holds the
	// fraction of land cells at or above that kind's effective threshold.
	AboveThresholdFraction [4]float64
	// Thresholds are the effective cut points this field was measured against,
	// indexed by SettlementNodeKind. With per-level calibration they move with
	// the mesh size, so reporting them shows how far the resolved cut point
	// sits from the level-5 absolute setting.
	Thresholds [4]float64
}

type SettlementNodeFormationDiagnostics struct {
	LandCells                int
	ThresholdEligible        int
	ThresholdEligibleByKind  [4]int
	SupportRejected          int
	SupportDowngraded        int
	ClassifiedCount          int
	ClassifiedByKind         [4]int
	PeakRejected             int
	PeakRejectedByKind       [4]int
	PeakRejectedUnscaledPass int
	PeakRejectedRankPass     int
	PeakRadiusHops           int
	PeakDiscCellsMean        float64
	CarryingDistribution     SettlementFieldDistribution
	UrbanDistribution        SettlementFieldDistribution
	SettlementClassRejected  int
	CandidateCount           int
	CandidateByKind          [4]int
	SpacingKept              int
	SpacingRejected          int
	SpacingKeptByKind        [4]int
	SpacingRejectedByKind    [4]int
	SpacingBlockerByKind     [4]int
	SpacingRejectedPairKind  [4][4]int
	SpacingKeptSupportArea   [4]float64
	SpacingRejectSupportArea [4]float64
	WaystationsAdded         int
	PrePruneCount            int
	PrunedCount              int
	FinalCount               int
	FinalByKind              [4]int
}

type RegionFormationDiagnostics struct {
	TransportLinks                         int
	PhysicalClusterLinks                   int
	PhysicalReachablePairs                 int
	PhysicalAlreadyLinkedPairs             int
	PhysicalSkippedTransportConnectedPairs int
	PhysicalSkippedSameComponentPairs      int
	PhysicalSkippedCrossComponentPairs     int
	PhysicalSingletonCandidatePairs        int
	RegionCount                            int
}

type SettlementLinkFormationDiagnostics struct {
	SourceNodes         int
	SourceByKind        [4]int
	ReachableTargets    int
	ReachableByKind     [4]int
	ReachableTargetKind [4]int
	NearTargets         int
	NearByKind          [4]int
	NearTargetKind      [4]int
	SelectedTargets     int
	SelectedByKind      [4]int
	SelectedTargetKind  [4]int
	SelectedPairKind    [4][4]int
	CreatedPairKind     [4][4]int
	CreatedLinks        int
	DuplicateSelections int
	NoReachableSources  int
	NoReachableByKind   [4]int
	NoSelectedSources   int
	NoSelectedByKind    [4]int
	TargetLimitedNodes  int
	TargetLimitedByKind [4]int
}

type SettlementNetworkResult struct {
	Nodes   []SettlementNode
	Links   []SettlementLink
	Regions []SettlementRegion
	// KindThresholds records the per-field kind thresholds this run actually
	// classified with, so consumers that re-derive settlement classes from a
	// finished (possibly cached) network use the same cut points.
	KindThresholds *SettlementKindThresholds
	Diagnostics    *SettlementNetworkDiagnostics
}

// SettlementKindThresholds holds the effective classification cut points for
// each SettlementNodeKind, separately for the carrying-capacity and
// urban-potential fields.
type SettlementKindThresholds struct {
	Carrying [4]float64
	Urban    [4]float64
}

// SettlementKindLevelCalibration is one row of the per-level threshold table:
// the resolution correction that reproduces the level-5 reference land
// fractions on a mesh of Cells cells.
//
// The correction is stored as a per-kind multiplier on the absolute settings
// thresholds, not as an absolute cut point, so the settings stay the tuning
// knob: raising TownThreshold moves the town cut at every resolution together,
// and the level-5 row is 1.0 by construction rather than by a rounded
// coincidence with the constants.
type SettlementKindLevelCalibration struct {
	Cells int
	// CarryingScale and UrbanScale are indexed by SettlementNodeKind and
	// multiply the kind's absolute settings threshold. The city entry is
	// carried for shape only: the city cut stays absolute at every level.
	CarryingScale [4]float64
	UrbanScale    [4]float64
}

type SettlementNetworkSettings struct {
	HamletThreshold  float64
	VillageThreshold float64
	TownThreshold    float64
	CityThreshold    float64

	// KindCalibration holds the per-level resolution correction: for each
	// calibrated mesh size, the multiplier on the absolute thresholds above
	// that selects the same mean land fraction at that resolution as the
	// unscaled thresholds select on the level-5 reference mesh. Entries must be
	// sorted by strictly increasing positive Cells and carry finite positive
	// scales; a mesh between two entries interpolates in log cell count and one
	// outside the range clamps to the nearest entry. An empty table — or one
	// that fails ValidateSettlementKindCalibration — falls back to the absolute
	// thresholds.
	//
	// Because the table only scales the settings, the four threshold fields
	// above remain the tuning knob at every resolution.
	//
	// The two fields are calibrated separately because CarryingCapacity and
	// UrbanPotential have visibly different distributions: at the same absolute
	// threshold they select different fractions of land, so one shared
	// correction would silently move one of the two fields' cut points.
	//
	// Because the correction is keyed on the mesh and not on the world, the cut
	// point is a constant within a level: a barren world still yields fewer
	// settlements than a lush one, while the resolution artifact is removed.
	KindCalibration []SettlementKindLevelCalibration

	HamletSpacingDeg  float64
	VillageSpacingDeg float64
	TownSpacingDeg    float64
	CitySpacingDeg    float64

	HamletMaxTravel  float64
	VillageMaxTravel float64
	TownMaxTravel    float64
	CityMaxTravel    float64

	// resolved caches the thresholds looked up from KindCalibration for this
	// mesh. It is filled once per run by resolveSettlementKindThresholds and
	// read by every threshold comparison; nil means "use the absolute
	// thresholds".
	resolved *SettlementKindThresholds
}

func DefaultSettlementNetworkSettings() SettlementNetworkSettings {
	return SettlementNetworkSettings{
		HamletThreshold:  0.38,
		VillageThreshold: 0.46,
		TownThreshold:    0.55,
		CityThreshold:    0.64,

		KindCalibration: defaultSettlementKindCalibration(),

		HamletSpacingDeg:  5.5,
		VillageSpacingDeg: 7.5,
		TownSpacingDeg:    12.0,
		CitySpacingDeg:    18.0,

		HamletMaxTravel:  6.4,
		VillageMaxTravel: 8.8,
		TownMaxTravel:    11.2,
		CityMaxTravel:    14.5,
	}
}

// defaultSettlementKindCalibration returns the per-level resolution correction:
// for each mesh size, the multiplier on the absolute thresholds that makes the
// CarryingCapacity / UrbanPotential cut select the same mean land fraction as
// the unscaled thresholds (0.38 / 0.46 / 0.55) select on level-5 meshes.
//
// Method: land-cell distributions were measured on reference worlds and the cut
// point solved per level so that the *mean* land fraction it selects across the
// seed set equals the mean fraction the unscaled level-5 thresholds select on
// the same seeds at level 5, so seed-to-seed habitability differences cancel out
// of the correction. The solved absolute cut point was then divided by the
// level-5 threshold it corrects, giving the scale stored here.
//
// Seed set: 4, 42, 84, 91, 123, 255 at levels 5, 6 and 7 — the six seeds of the
// level67_linear_footprint sweep, plus level-5 runs of the same six. Level 5 is
// the reference by construction and its row is exactly 1.0; running the same
// solver against level 5 returns the absolute constants (0.3800 / 0.4600 /
// 0.5500), which is the correctness check on the procedure.
//
// The solved absolute cut points these scales reproduce, for reference:
//
//	L5 carrying 0.3800 / 0.4600 / 0.5500   urban 0.3800 / 0.4600 / 0.5500
//	L6 carrying 0.3794 / 0.4730 / 0.5521   urban 0.3842 / 0.4773 / 0.5520
//	L7 carrying 0.3303 / 0.4778 / 0.5564   urban 0.3663 / 0.4828 / 0.5528
//	L8 carrying 0.2813 / 0.4826 / 0.5608   urban 0.3484 / 0.4883 / 0.5536
//
// The level-5 target fractions, meaned over the six seeds:
//
//	carrying hamlet 0.2720 village 0.0801 town 0.0233
//	urban    hamlet 0.2144 village 0.0502 town 0.0199
//
// The corrections are not monotone in level: the hamlet cut is near-flat from L5
// to L6 and then falls sharply at L7 (the fraction above 0.38 drops as fine
// cells stop averaging sub-cell variation into the middle of the range), while
// the village and town cuts rise at both steps.
//
// Levels 6 and 7 previously rested on two seeds (42 and 123). Widening to six
// left the village and town scales essentially unchanged (<0.015 in scale) but
// moved the hamlet scales materially: L6 carrying 1.0200 -> 0.9983, L7 urban
// 0.9326 -> 0.9639. Scored on seeds the old fit had never seen, the two-seed
// table selected too much land at level 7 — mean fraction error +0.042 (urban
// hamlet) and +0.018 (carrying hamlet). Under leave-one-out over the six seeds
// the same errors are +0.003 and +0.002. The residual per-seed spread is
// unchanged, as it must be: a single per-level constant cannot track individual
// worlds, only remove the systematic offset.
//
// The level-8 row is EXTRAPOLATED, not measured — no level-8 reference world
// exists and generating one is out of reach (a level-7 world already takes over
// an hour per seed and level 8 quadruples the cell count on top of a footprint
// depth that grows with resolution). The row continues the measured L6 -> L7
// step by one more factor-of-four in cell count, i.e. scale(L8) = 2*scale(L7) -
// scale(L6) per kind and field, which is linear extrapolation in log cell count
// since the levels are equally spaced there. It is re-derived from the six-seed
// L6/L7 rows above, so it moved with them. Treat it as provisional and
// re-derive it once an L8 world is available.
//
// The city column is not calibrated: at level 5 the city cut selects well under
// one cell per world, so a fraction target there is meaningless. It stays at
// 1.0 and resolveSettlementKindThresholds re-pins it to the absolute
// CityThreshold regardless.
func defaultSettlementKindCalibration() []SettlementKindLevelCalibration {
	return []SettlementKindLevelCalibration{
		{Cells: 10242, CarryingScale: [4]float64{1.000000, 1.000000, 1.000000, 1.0}, UrbanScale: [4]float64{1.000000, 1.000000, 1.000000, 1.0}},
		{Cells: 40962, CarryingScale: [4]float64{0.998307, 1.028207, 1.003761, 1.0}, UrbanScale: [4]float64{1.011027, 1.037552, 1.003651, 1.0}},
		{Cells: 163842, CarryingScale: [4]float64{0.869222, 1.038639, 1.011677, 1.0}, UrbanScale: [4]float64{0.963909, 1.049515, 1.005103, 1.0}},
		{Cells: 655362, CarryingScale: [4]float64{0.740138, 1.049070, 1.019593, 1.0}, UrbanScale: [4]float64{0.916790, 1.061478, 1.006556, 1.0}},
	}
}

func BuildSettlementNetwork(
	sites []Vector3D,
	cells []VoronoiCell,
	settlements *SettlementResult,
	population *PopulationResult,
	biomes *BiomeResult,
	soils *SoilResult,
	resources *ResourceResult,
	elevation []float64,
	seaLevel float64,
	settings SettlementNetworkSettings,
) *SettlementNetworkResult {
	n := len(elevation)
	out := &SettlementNetworkResult{
		Diagnostics: &SettlementNetworkDiagnostics{
			MovementCost: make([]float64, n),
			NodeByCell:   make([]int, n),
		},
	}
	for i := range out.Diagnostics.NodeByCell {
		out.Diagnostics.NodeByCell[i] = -1
	}
	if settlements == nil || settlements.Diagnostics == nil || population == nil || population.Diagnostics == nil {
		return out
	}

	// Resolve the calibrated kind thresholds once, before any per-cell work. A
	// rejected calibration table falls back to the absolute thresholds and is
	// reported in the diagnostics instead of failing silently.
	settings, calibrationErr := resolveSettlementKindThresholds(settings, n)
	if calibrationErr != nil {
		out.Diagnostics.KindCalibrationError = calibrationErr.Error()
	}
	thresholds := settlementKindThresholds(settings)
	out.KindThresholds = &thresholds

	for i := 0; i < n; i++ {
		out.Diagnostics.MovementCost[i] = movementCostForCell(i, settlements, biomes, soils, elevation, seaLevel)
	}

	candidates := settlementNodeCandidates(sites, cells, settlements, population, resources, elevation, seaLevel, settings, &out.Diagnostics.NodeFormation)
	out.Nodes = filterSettlementNodeCandidates(candidates, sites, cells, population, settings, &out.Diagnostics.NodeFormation)
	beforeWaystations := len(out.Nodes)
	out.Nodes = addWaystationBridges(sites, cells, out.Nodes, out.Diagnostics.MovementCost, settlements, population, resources, elevation, seaLevel, settings)
	out.Diagnostics.NodeFormation.WaystationsAdded = len(out.Nodes) - beforeWaystations
	out.Diagnostics.NodeFormation.PrePruneCount = len(out.Nodes)
	for i := range out.Nodes {
		out.Nodes[i].ID = i
		out.Diagnostics.NodeByCell[out.Nodes[i].CellIndex] = i
	}
	out.Links = buildSettlementLinks(cells, out.Nodes, out.Diagnostics.MovementCost, settlements, resources, settings, &out.Diagnostics.LinkFormation)
	prePruneCount := len(out.Nodes)
	out.Nodes, out.Links, out.Diagnostics.NodeByCell = pruneIsolatedNodes(out.Nodes, out.Links, out.Diagnostics.NodeByCell, settlements, resources)
	out.Diagnostics.NodeFormation.PrunedCount = prePruneCount - len(out.Nodes)
	out.Diagnostics.NodeFormation.FinalCount = len(out.Nodes)
	out.Diagnostics.NodeFormation.FinalByKind = settlementNodeKindCounts(out.Nodes)
	assignSettlementRegions(out, cells, settings)
	return out
}

func movementCostForCell(
	idx int,
	settlements *SettlementResult,
	biomes *BiomeResult,
	soils *SoilResult,
	elevation []float64,
	seaLevel float64,
) float64 {
	if idx < 0 || idx >= len(elevation) || elevation[idx] < seaLevel {
		return math.Inf(1)
	}
	cost := 1.0
	if soils != nil && soils.Diagnostics != nil {
		if idx < len(soils.Diagnostics.Relief) {
			cost += 1.25 * smoothstep01(120, 1400, soils.Diagnostics.Relief[idx])
		}
		if idx < len(soils.Diagnostics.Rockiness) {
			cost += 0.70 * soils.Diagnostics.Rockiness[idx]
		}
	}
	if biomes != nil && biomes.Diagnostics != nil {
		if idx < len(biomes.Diagnostics.AnnualIceFraction) {
			cost += 1.80 * biomes.Diagnostics.AnnualIceFraction[idx]
		}
		if idx < len(biomes.Diagnostics.AridityRatio) {
			cost += 0.55 * smoothstep01(0.22, 0.95, biomes.Diagnostics.AridityRatio[idx])
		}
		if idx < len(biomes.Diagnostics.WetlandAffinity) {
			cost += 0.50 * biomes.Diagnostics.WetlandAffinity[idx]
		}
	}
	if settlements != nil && settlements.Diagnostics != nil {
		if idx < len(settlements.Diagnostics.RiverBonus) {
			cost -= 0.22 * settlements.Diagnostics.RiverBonus[idx]
		}
		if idx < len(settlements.Diagnostics.CoastalBonus) {
			cost -= 0.18 * settlements.Diagnostics.CoastalBonus[idx]
		}
	}
	return math.Max(0.35, cost)
}

func settlementNodeCandidates(
	sites []Vector3D,
	cells []VoronoiCell,
	settlements *SettlementResult,
	population *PopulationResult,
	resources *ResourceResult,
	elevation []float64,
	seaLevel float64,
	settings SettlementNetworkSettings,
	diag *SettlementNodeFormationDiagnostics,
) []SettlementNode {
	candidates := make([]SettlementNode, 0)
	var carryingSamples, urbanSamples []float64
	peakDiscCells := 0
	peakDiscSamples := 0
	if diag != nil {
		diag.PeakRadiusHops = meshResolutionAdjustedSteps(1, len(cells))
	}
	for i := range elevation {
		if elevation[i] < seaLevel {
			continue
		}
		carrying := population.Diagnostics.CarryingCapacity[i]
		urban := population.Diagnostics.UrbanPotential[i]
		if diag != nil {
			diag.LandCells++
			carryingSamples = append(carryingSamples, carrying)
			urbanSamples = append(urbanSamples, urban)
		}
		score := clamp01(0.58*carrying + 0.42*urban)
		rawKind, rawOK := rawSettlementNodeKind(i, cells, population, settings)
		if diag != nil && rawOK {
			diag.ThresholdEligible++
			incrementSettlementKindCount(&diag.ThresholdEligibleByKind, rawKind)
		}
		kind, ok := classifySettlementNodeCandidateAt(i, cells, population, settings)
		if !ok {
			if diag != nil && rawOK {
				diag.SupportRejected++
			}
			continue
		}
		if diag != nil {
			if rawOK && kind != rawKind {
				diag.SupportDowngraded++
			}
			diag.ClassifiedCount++
			if int(kind) >= 0 && int(kind) < len(diag.ClassifiedByKind) {
				diag.ClassifiedByKind[kind]++
			}
			peakDiscCells += len(cellsWithinHops(cells, i, diag.PeakRadiusHops))
			peakDiscSamples++
		}
		if !isLocalSettlementPeak(i, score, cells, population) {
			if diag != nil {
				diag.PeakRejected++
				incrementSettlementKindCount(&diag.PeakRejectedByKind, kind)
				if isLocalSettlementPeakWithRadius(i, score, cells, population, 1) {
					diag.PeakRejectedUnscaledPass++
				}
				if isBaselineWindowSettlementPeak(i, score, cells, population, diag.PeakRadiusHops) {
					diag.PeakRejectedRankPass++
				}
			}
			continue
		}
		resourceExceptional := settlementResourceExceptional(resources, i)
		if settlements.Classes[i] < SettlementMarginal && !resourceExceptional {
			if diag != nil {
				diag.SettlementClassRejected++
			}
			continue
		}
		candidates = append(candidates, SettlementNode{
			CellIndex:           i,
			Kind:                kind,
			Score:               score,
			CarryingCapacity:    carrying,
			UrbanPotential:      urban,
			PhysicalSupportArea: SettlementNodePhysicalSupportArea(i, kind, cells, population, settings),
			Coastal:             settlements.Diagnostics.CoastalBonus[i] >= 0.16,
			River:               settlements.Diagnostics.RiverBonus[i] >= 0.24,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Kind != candidates[j].Kind {
			return candidates[i].Kind > candidates[j].Kind
		}
		return candidates[i].Score > candidates[j].Score
	})
	if diag != nil {
		diag.CandidateCount = len(candidates)
		diag.CandidateByKind = settlementNodeKindCounts(candidates)
		if peakDiscSamples > 0 {
			diag.PeakDiscCellsMean = float64(peakDiscCells) / float64(peakDiscSamples)
		}
		thresholds := settlementKindThresholds(settings)
		diag.CarryingDistribution = settlementFieldDistribution(carryingSamples, thresholds.Carrying)
		diag.UrbanDistribution = settlementFieldDistribution(urbanSamples, thresholds.Urban)
	}
	return candidates
}

// resolveSettlementKindThresholds returns a copy of settings whose kind
// thresholds are pinned to this mesh's calibrated absolute cut points.
//
// CarryingCapacity and UrbanPotential change distribution shape with mesh
// resolution: coarse cells average sub-cell variation away while fine cells
// keep the extremes, so mean/p50 fall and p90/p99/max rise as resolution goes
// up. A single absolute threshold therefore selects a different fraction of the
// world at every resolution — the middle (hamlet) band is squeezed hardest.
//
// The artifact is a property of the mesh, not of the world, so it is corrected
// at mesh granularity: each supported level has its own absolute cut point,
// calibrated so it selects the same mean land fraction over reference seeds as
// the level-5 absolute thresholds select at level 5. Within a level the cut
// point is constant, so world-to-world habitability differences survive.
//
// Because the table stores a multiplier rather than an absolute cut point, the
// four threshold settings stay authoritative: at level 5 every scale is 1.0 and
// the resolved cut points equal the settings exactly, and changing a setting
// moves that kind's cut at every resolution.
//
// This is called once per run; per-cell comparisons then read the cached array.
// A calibration table that fails validation is rejected with an error and the
// caller falls back to the absolute thresholds, which are always finite — a
// malformed table must never yield NaN cut points, because every threshold
// comparison against NaN is false and the world would silently come back with
// no settlement nodes at all.
func resolveSettlementKindThresholds(settings SettlementNetworkSettings, cellCount int) (SettlementNetworkSettings, error) {
	if len(settings.KindCalibration) == 0 || cellCount <= 0 {
		return settings, nil
	}
	if err := ValidateSettlementKindCalibration(settings.KindCalibration); err != nil {
		return settings, err
	}
	scale := interpolateSettlementKindCalibration(settings.KindCalibration, cellCount)
	var resolved SettlementKindThresholds
	for kind := SettlementNodeHamlet; kind <= SettlementNodeCity; kind++ {
		absolute := settlementNodeKindThreshold(kind, settings)
		resolved.Carrying[kind] = absolute * scale.Carrying[kind]
		resolved.Urban[kind] = absolute * scale.Urban[kind]
	}
	enforceSettlementKindThresholdOrder(&resolved.Carrying, settings.CityThreshold)
	enforceSettlementKindThresholdOrder(&resolved.Urban, settings.CityThreshold)
	if !settlementKindThresholdsFinite(resolved) {
		return settings, fmt.Errorf("settlement kind calibration produced non-finite thresholds at %d cells", cellCount)
	}
	settings.resolved = &resolved
	return settings, nil
}

// ValidateSettlementKindCalibration reports whether a calibration table can be
// interpolated safely. Callers that build their own table should check it at
// configuration time; BuildSettlementNetwork also checks it and records the
// failure in its diagnostics rather than classifying against bad cut points.
//
// Cells must be positive and strictly increasing: a zero or negative row makes
// the log-cell-count span infinite, which passes a naive span > 0 guard and
// yields a NaN interpolation fraction and NaN thresholds.
func ValidateSettlementKindCalibration(table []SettlementKindLevelCalibration) error {
	prev := 0
	for i, row := range table {
		if row.Cells <= 0 {
			return fmt.Errorf("settlement kind calibration row %d has non-positive Cells %d", i, row.Cells)
		}
		if row.Cells <= prev {
			return fmt.Errorf("settlement kind calibration row %d has Cells %d, want strictly greater than the previous row's %d", i, row.Cells, prev)
		}
		prev = row.Cells
		for kind := range row.CarryingScale {
			if !validSettlementKindScale(row.CarryingScale[kind]) {
				return fmt.Errorf("settlement kind calibration row %d kind %d has invalid carrying scale %v", i, kind, row.CarryingScale[kind])
			}
			if !validSettlementKindScale(row.UrbanScale[kind]) {
				return fmt.Errorf("settlement kind calibration row %d kind %d has invalid urban scale %v", i, kind, row.UrbanScale[kind])
			}
		}
	}
	return nil
}

func validSettlementKindScale(scale float64) bool {
	return scale > 0 && !math.IsInf(scale, 0)
}

func settlementKindThresholdsFinite(thresholds SettlementKindThresholds) bool {
	for kind := range thresholds.Carrying {
		if math.IsNaN(thresholds.Carrying[kind]) || math.IsInf(thresholds.Carrying[kind], 0) {
			return false
		}
		if math.IsNaN(thresholds.Urban[kind]) || math.IsInf(thresholds.Urban[kind], 0) {
			return false
		}
	}
	return true
}

// settlementKindScales holds the per-kind resolution multipliers resolved for a
// single mesh, in the same [4]float64 layout as the threshold arrays.
type settlementKindScales struct {
	Carrying [4]float64
	Urban    [4]float64
}

// interpolateSettlementKindCalibration looks up cellCount in the calibration
// table, interpolating in log cell count between rows and clamping outside the
// calibrated range. The table must already have passed
// ValidateSettlementKindCalibration, so every Cells value is positive and the
// log span between two rows is finite and non-zero.
func interpolateSettlementKindCalibration(table []SettlementKindLevelCalibration, cellCount int) settlementKindScales {
	var out settlementKindScales
	if len(table) == 0 {
		return out
	}
	if cellCount <= table[0].Cells {
		return settlementKindScales{Carrying: table[0].CarryingScale, Urban: table[0].UrbanScale}
	}
	last := table[len(table)-1]
	if cellCount >= last.Cells {
		return settlementKindScales{Carrying: last.CarryingScale, Urban: last.UrbanScale}
	}
	hi := 1
	for hi < len(table) && table[hi].Cells < cellCount {
		hi++
	}
	lo := hi - 1
	span := math.Log(float64(table[hi].Cells)) - math.Log(float64(table[lo].Cells))
	frac := 0.0
	if span > 0 && !math.IsInf(span, 0) {
		frac = (math.Log(float64(cellCount)) - math.Log(float64(table[lo].Cells))) / span
	}
	for kind := range out.Carrying {
		out.Carrying[kind] = table[lo].CarryingScale[kind] + frac*(table[hi].CarryingScale[kind]-table[lo].CarryingScale[kind])
		out.Urban[kind] = table[lo].UrbanScale[kind] + frac*(table[hi].UrbanScale[kind]-table[lo].UrbanScale[kind])
	}
	return out
}

// enforceSettlementKindThresholdOrder makes the calibrated cut points
// non-decreasing and pins the city cut to its absolute setting.
//
// The city cut point stays absolute at every level: at level 5 it selects well
// under one cell per world, so a fraction target there is meaningless and a
// calibrated value would just track the mesh maximum. The pin therefore has to
// survive the ordering pass — a calibration row whose town cut lands above
// CityThreshold clamps the lower cuts down to the city cut instead of raising
// the city cut above the configured absolute.
func enforceSettlementKindThresholdOrder(values *[4]float64, cityThreshold float64) {
	for i := 1; i <= int(SettlementNodeTown); i++ {
		if values[i] < values[i-1] {
			values[i] = values[i-1]
		}
	}
	values[SettlementNodeCity] = cityThreshold
	for i := int(SettlementNodeTown); i >= 0; i-- {
		if values[i] > cityThreshold {
			values[i] = cityThreshold
		}
	}
}

// SettlementNetworkSettingsWithKindThresholds pins settings to the kind
// thresholds a finished network was built with, so downstream consumers
// (proto-civilization gating, review diagnostics) classify cells the same way
// the network did — including when the network came back from a cache.
func SettlementNetworkSettingsWithKindThresholds(settings SettlementNetworkSettings, network *SettlementNetworkResult) SettlementNetworkSettings {
	if network == nil || network.KindThresholds == nil {
		return settings
	}
	thresholds := *network.KindThresholds
	settings.resolved = &thresholds
	return settings
}

// SettlementCarryingKindThreshold and SettlementUrbanKindThreshold report the
// effective per-field cut point for a kind under these settings.
func SettlementCarryingKindThreshold(kind SettlementNodeKind, settings SettlementNetworkSettings) float64 {
	if settings.resolved != nil && int(kind) >= 0 && int(kind) < len(settings.resolved.Carrying) {
		return settings.resolved.Carrying[kind]
	}
	return settlementNodeKindThreshold(kind, settings)
}

func SettlementUrbanKindThreshold(kind SettlementNodeKind, settings SettlementNetworkSettings) float64 {
	if settings.resolved != nil && int(kind) >= 0 && int(kind) < len(settings.resolved.Urban) {
		return settings.resolved.Urban[kind]
	}
	return settlementNodeKindThreshold(kind, settings)
}

func settlementKindThresholds(settings SettlementNetworkSettings) SettlementKindThresholds {
	var out SettlementKindThresholds
	for kind := SettlementNodeHamlet; kind <= SettlementNodeCity; kind++ {
		out.Carrying[kind] = SettlementCarryingKindThreshold(kind, settings)
		out.Urban[kind] = SettlementUrbanKindThreshold(kind, settings)
	}
	return out
}

func settlementNodeKindThreshold(kind SettlementNodeKind, settings SettlementNetworkSettings) float64 {
	switch kind {
	case SettlementNodeCity:
		return settings.CityThreshold
	case SettlementNodeTown:
		return settings.TownThreshold
	case SettlementNodeVillage:
		return settings.VillageThreshold
	default:
		return settings.HamletThreshold
	}
}

func settlementFieldDistribution(values []float64, thresholds [4]float64) SettlementFieldDistribution {
	out := SettlementFieldDistribution{Count: len(values), Thresholds: thresholds}
	if len(values) == 0 {
		return out
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	out.Mean = sum / float64(len(sorted))
	out.P50 = sortedFloatPercentile(sorted, 0.50)
	out.P90 = sortedFloatPercentile(sorted, 0.90)
	out.P99 = sortedFloatPercentile(sorted, 0.99)
	out.Max = sorted[len(sorted)-1]
	for kind := SettlementNodeHamlet; kind <= SettlementNodeCity; kind++ {
		idx := sort.SearchFloat64s(sorted, thresholds[kind])
		out.AboveThresholdFraction[kind] = float64(len(sorted)-idx) / float64(len(sorted))
	}
	return out
}

// rawSettlementNodeKind reports the threshold-only classification of a cell,
// before the physical-support downgrade pass. Diagnostics use it to separate
// threshold rejections from support rejections.
func rawSettlementNodeKind(idx int, cells []VoronoiCell, population *PopulationResult, settings SettlementNetworkSettings) (SettlementNodeKind, bool) {
	if population == nil || population.Diagnostics == nil || idx < 0 || idx >= len(population.Diagnostics.CarryingCapacity) || idx >= len(population.Diagnostics.UrbanPotential) {
		return 0, false
	}
	carrying := population.Diagnostics.CarryingCapacity[idx]
	urban := population.Diagnostics.UrbanPotential[idx]
	switch {
	case physicallySignificantCityCandidate(idx, cells, population, settings):
		return SettlementNodeCity, true
	case urban >= SettlementUrbanKindThreshold(SettlementNodeTown, settings) ||
		carrying >= SettlementCarryingKindThreshold(SettlementNodeTown, settings):
		return SettlementNodeTown, true
	case carrying >= SettlementCarryingKindThreshold(SettlementNodeVillage, settings):
		return SettlementNodeVillage, true
	case carrying >= SettlementCarryingKindThreshold(SettlementNodeHamlet, settings):
		return SettlementNodeHamlet, true
	default:
		return 0, false
	}
}

func classifySettlementNodeCandidateAt(idx int, cells []VoronoiCell, population *PopulationResult, settings SettlementNetworkSettings) (SettlementNodeKind, bool) {
	rawKind, ok := rawSettlementNodeKind(idx, cells, population, settings)
	if !ok {
		return 0, false
	}
	return settlementNodeKindWithPhysicalSupport(idx, rawKind, cells, population, settings)
}

func settlementNodeKindWithPhysicalSupport(idx int, rawKind SettlementNodeKind, cells []VoronoiCell, population *PopulationResult, settings SettlementNetworkSettings) (SettlementNodeKind, bool) {
	for kind := rawKind; kind >= SettlementNodeHamlet; kind-- {
		if !physicallySupportsSettlementNodeKind(idx, kind, population, settings) {
			continue
		}
		// Regional anchors represent a physically meaningful catchment, not a
		// single refined-cell spike. This matches the existing supported-regional
		// diagnostic convention used by proto/polity checks.
		if kind >= SettlementNodeTown && SettlementNodePhysicalSupportArea(idx, kind, cells, population, settings) < 0.5 {
			continue
		}
		return kind, true
	}
	return 0, false
}

func classifySettlementNodeCandidate(carrying, urban float64, settings SettlementNetworkSettings) (SettlementNodeKind, bool) {
	switch {
	case urban >= SettlementUrbanKindThreshold(SettlementNodeCity, settings) &&
		carrying >= SettlementCarryingKindThreshold(SettlementNodeTown, settings):
		return SettlementNodeCity, true
	case urban >= SettlementUrbanKindThreshold(SettlementNodeTown, settings) ||
		carrying >= SettlementCarryingKindThreshold(SettlementNodeTown, settings):
		return SettlementNodeTown, true
	case carrying >= SettlementCarryingKindThreshold(SettlementNodeVillage, settings):
		return SettlementNodeVillage, true
	case carrying >= SettlementCarryingKindThreshold(SettlementNodeHamlet, settings):
		return SettlementNodeHamlet, true
	default:
		return 0, false
	}
}

func physicallySignificantCityCandidate(idx int, cells []VoronoiCell, population *PopulationResult, settings SettlementNetworkSettings) bool {
	carrying := population.Diagnostics.CarryingCapacity[idx]
	urban := population.Diagnostics.UrbanPotential[idx]
	townCarrying := SettlementCarryingKindThreshold(SettlementNodeTown, settings)
	cityUrban := SettlementUrbanKindThreshold(SettlementNodeCity, settings)
	if carrying < townCarrying || urban < cityUrban-0.02 {
		return false
	}
	cityLikeCells := 1
	radius := meshResolutionAdjustedSteps(1, len(cells))
	disc := cellsWithinHops(cells, idx, radius)
	for _, cellIdx := range disc {
		if cellIdx < 0 || cellIdx >= len(population.Diagnostics.CarryingCapacity) || cellIdx >= len(population.Diagnostics.UrbanPotential) {
			continue
		}
		if population.Diagnostics.CarryingCapacity[cellIdx] >= townCarrying-0.02 &&
			population.Diagnostics.UrbanPotential[cellIdx] >= cityUrban-0.02 {
			cityLikeCells++
		}
	}
	cityArea := settlementSupportDiscArea(cityLikeCells, len(disc))
	if urban >= cityUrban && cityArea >= 1.0 {
		return true
	}
	return cityArea >= 1.5
}

// settlementSupportDiscBaselineCells is the baseline (L5) size of a one-hop
// support disc around a hexagonal cell: the cell itself plus its six neighbours.
const settlementSupportDiscBaselineCells = 7.0

// settlementSupportDiscArea converts a supported-cell count inside a one-hop
// support disc into baseline-equivalent cells.
//
// A hop disc of radius r holds ~3r²+3r cells, so its *physical* area shrinks as
// resolution rises even though the hop radius is scaled: 7.00 baseline cells at
// L5, 4.75 at L6, 3.81 at L7. Converting the raw count with
// meshScaledTerritoryAreaCells (count × scale²) therefore does not hold a fixed
// physical window, and every absolute threshold downstream (0.5, 0.75, 1.0, 1.5)
// silently tightened with resolution — the unscaled `+1` self-seed alone dropped
// the achievable minimum from 1.00 at L5 to 0.0625 at L7.
//
// Reporting the supported *fraction* of the measured disc, rescaled to the L5
// seven-cell reference, keeps the window at a constant 7.0 baseline-equivalent
// cells at every resolution, so those thresholds retain their L5 meaning. At L5
// this is exactly the old value for hexagonal cells (7/7 × count); it differs
// only for the twelve pentagonal cells, whose disc holds six cells (7/6 × count).
func settlementSupportDiscArea(supportedCells, discCells int) float64 {
	if supportedCells <= 0 {
		return 0
	}
	if discCells <= 0 {
		// Degenerate mesh with no measurable window (never happens on a Voronoi
		// sphere, where every cell has five or six neighbours): credit one
		// baseline cell per supporting cell rather than a full window.
		return float64(supportedCells)
	}
	// Credit per supporting cell is capped at one baseline cell. A pentagonal
	// cell's 1-hop window holds only six cells, so the uncapped fraction would
	// award 7/6 baseline cells each — an over-credit, since that window really
	// does cover ~6 baseline areas. Capping keeps L5 bit-identical to the
	// previous count*scale^2 conversion for every cell including the twelve
	// pentagons, and is inert at finer meshes where the factor is well below 1
	// (0.37 at L6, 0.11 at L7).
	//
	// The self-seed is deliberately scaled along with the neighbours: this
	// reports a supported *physical area*, and a lone supported cell at L7
	// really does cover a sixteenth of the area a lone L5 cell covers. A
	// constant supported fraction of the window therefore maps to a constant
	// area at every resolution, which is what
	// TestSettlementNodePhysicalSupportAreaHoldsPartialNeighborhood pins down.
	perCell := settlementSupportDiscBaselineCells / float64(discCells+1)
	if perCell > 1 {
		perCell = 1
	}
	return float64(supportedCells) * perCell
}

func SettlementNodePhysicalSupportArea(idx int, kind SettlementNodeKind, cells []VoronoiCell, population *PopulationResult, settings SettlementNetworkSettings) float64 {
	if population == nil || population.Diagnostics == nil || idx < 0 || idx >= len(cells) || idx >= len(population.Diagnostics.CarryingCapacity) {
		return 0
	}
	areaCells := 1
	radius := meshResolutionAdjustedSteps(1, len(cells))
	disc := cellsWithinHops(cells, idx, radius)
	for _, cellIdx := range disc {
		if physicallySupportsSettlementNodeKind(cellIdx, kind, population, settings) {
			areaCells++
		}
	}
	return settlementSupportDiscArea(areaCells, len(disc))
}

func settlementNodeRecordedSupportArea(node SettlementNode, cells []VoronoiCell, population *PopulationResult, settings SettlementNetworkSettings) float64 {
	if node.PhysicalSupportArea > 0 {
		return node.PhysicalSupportArea
	}
	return SettlementNodePhysicalSupportArea(node.CellIndex, node.Kind, cells, population, settings)
}

func SettlementNodePhysicalSupportWeight(node SettlementNode) float64 {
	if node.Kind < SettlementNodeTown || node.PhysicalSupportArea <= 0 {
		return 1
	}
	if node.PhysicalSupportArea >= 0.75 {
		return 1
	}
	return math.Max(0.35, math.Sqrt(math.Max(node.PhysicalSupportArea, 0)))
}

func settlementNodeEffectiveRank(node SettlementNode) float64 {
	return float64(node.Kind) * SettlementNodePhysicalSupportWeight(node)
}

func physicallySupportsSettlementNodeKind(idx int, kind SettlementNodeKind, population *PopulationResult, settings SettlementNetworkSettings) bool {
	if population == nil || population.Diagnostics == nil || idx < 0 || idx >= len(population.Diagnostics.CarryingCapacity) || idx >= len(population.Diagnostics.UrbanPotential) {
		return false
	}
	carrying := population.Diagnostics.CarryingCapacity[idx]
	urban := population.Diagnostics.UrbanPotential[idx]
	switch kind {
	case SettlementNodeCity:
		return carrying >= SettlementCarryingKindThreshold(SettlementNodeTown, settings)-0.02 &&
			urban >= SettlementUrbanKindThreshold(SettlementNodeCity, settings)-0.02
	case SettlementNodeTown:
		return urban >= SettlementUrbanKindThreshold(SettlementNodeTown, settings) ||
			carrying >= SettlementCarryingKindThreshold(SettlementNodeTown, settings)
	case SettlementNodeVillage:
		return carrying >= SettlementCarryingKindThreshold(SettlementNodeVillage, settings)
	default:
		return carrying >= SettlementCarryingKindThreshold(SettlementNodeHamlet, settings)
	}
}

func isLocalSettlementPeak(idx int, score float64, cells []VoronoiCell, population *PopulationResult) bool {
	return isLocalSettlementPeakWithRadius(idx, score, cells, population, meshResolutionAdjustedSteps(1, len(cells)))
}

func isLocalSettlementPeakWithRadius(idx int, score float64, cells []VoronoiCell, population *PopulationResult, peakRadius int) bool {
	if score <= 0 {
		return false
	}
	for _, j := range cellsWithinHops(cells, idx, peakRadius) {
		if j < 0 || j >= len(population.Diagnostics.CarryingCapacity) {
			continue
		}
		neighborScore := clamp01(0.58*population.Diagnostics.CarryingCapacity[j] + 0.42*population.Diagnostics.UrbanPotential[j])
		if neighborScore > score+0.02 {
			return false
		}
	}
	return true
}

// isBaselineWindowSettlementPeak is a diagnostic-only counterfactual for the
// local-peak test. Instead of requiring a cell to beat every cell in the scaled
// hop disc, it allows it to lose to at most one cell per baseline-equivalent
// (L5 seven-cell) window inside that disc. At L5 the disc holds six neighbours,
// so zero losses are allowed and the rule is identical to isLocalSettlementPeak.
func isBaselineWindowSettlementPeak(idx int, score float64, cells []VoronoiCell, population *PopulationResult, peakRadius int) bool {
	if score <= 0 {
		return false
	}
	disc := cellsWithinHops(cells, idx, peakRadius)
	allowedBetter := int(float64(len(disc)+1)/settlementSupportDiscBaselineCells) - 1
	if allowedBetter < 0 {
		allowedBetter = 0
	}
	better := 0
	for _, j := range disc {
		if j < 0 || j >= len(population.Diagnostics.CarryingCapacity) {
			continue
		}
		neighborScore := clamp01(0.58*population.Diagnostics.CarryingCapacity[j] + 0.42*population.Diagnostics.UrbanPotential[j])
		if neighborScore > score+0.02 {
			better++
			if better > allowedBetter {
				return false
			}
		}
	}
	return true
}

func settlementResourceExceptional(resources *ResourceResult, idx int) bool {
	if resources == nil || resources.Diagnostics == nil {
		return false
	}
	score := resourceFuelSupport(resources, idx)
	if idx < len(resources.Diagnostics.GoldAffinity) {
		score += 0.35 * resources.Diagnostics.GoldAffinity[idx]
	}
	if idx < len(resources.Diagnostics.GemAffinity) {
		score += 0.35 * resources.Diagnostics.GemAffinity[idx]
	}
	return score >= 0.35
}

func filterSettlementNodeCandidates(
	candidates []SettlementNode,
	sites []Vector3D,
	cells []VoronoiCell,
	population *PopulationResult,
	settings SettlementNetworkSettings,
	diag *SettlementNodeFormationDiagnostics,
) []SettlementNode {
	kept := make([]SettlementNode, 0, len(candidates))
	for _, candidate := range candidates {
		tooClose := false
		blocker := SettlementNodeHamlet
		for _, existing := range kept {
			minSpacing := math.Max(nodeSpacingDeg(candidate.Kind, settings), nodeSpacingDeg(existing.Kind, settings)*0.55)
			if greatCircleDistanceDeg(sites[candidate.CellIndex], sites[existing.CellIndex]) < minSpacing {
				tooClose = true
				blocker = existing.Kind
				break
			}
		}
		if !tooClose {
			kept = append(kept, candidate)
			if diag != nil {
				recordSettlementSpacingSupport(&diag.SpacingKeptSupportArea, candidate, cells, population, settings)
			}
			continue
		}
		if diag != nil {
			incrementSettlementKindCount(&diag.SpacingRejectedByKind, candidate.Kind)
			incrementSettlementKindCount(&diag.SpacingBlockerByKind, blocker)
			recordSettlementLinkPairKind(&diag.SpacingRejectedPairKind, candidate.Kind, blocker)
			recordSettlementSpacingSupport(&diag.SpacingRejectSupportArea, candidate, cells, population, settings)
		}
	}
	if diag != nil {
		diag.SpacingKept = len(kept)
		diag.SpacingRejected = len(candidates) - len(kept)
		diag.SpacingKeptByKind = settlementNodeKindCounts(kept)
	}
	return kept
}

func recordSettlementSpacingSupport(
	out *[4]float64,
	node SettlementNode,
	cells []VoronoiCell,
	population *PopulationResult,
	settings SettlementNetworkSettings,
) {
	if out == nil || int(node.Kind) < 0 || int(node.Kind) >= len(out) {
		return
	}
	out[node.Kind] += settlementNodeRecordedSupportArea(node, cells, population, settings)
}

func settlementNodeKindCounts(nodes []SettlementNode) [4]int {
	var counts [4]int
	for _, node := range nodes {
		if int(node.Kind) >= 0 && int(node.Kind) < len(counts) {
			counts[node.Kind]++
		}
	}
	return counts
}

func buildSettlementLinks(
	cells []VoronoiCell,
	nodes []SettlementNode,
	movementCost []float64,
	settlements *SettlementResult,
	resources *ResourceResult,
	settings SettlementNetworkSettings,
	diag *SettlementLinkFormationDiagnostics,
) []SettlementLink {
	if len(nodes) == 0 {
		return nil
	}
	nodeByCell := make(map[int]int, len(nodes))
	for i, node := range nodes {
		nodeByCell[node.CellIndex] = i
	}
	links := make([]SettlementLink, 0)
	seen := make(map[[2]int]struct{})
	for i, node := range nodes {
		maxTravel := nodeTravelLimitForNode(node, settings)
		dist, prev := shortestPathsFromNode(node.CellIndex, cells, movementCost, maxTravel*1.10)
		targets, candidateDiag := candidateLinkTargets(i, node, nodes, nodeByCell, dist, settlements, resources, maxTravel)
		recordSettlementLinkFormationSource(diag, node.Kind, candidateDiag, targets, nodes)
		for _, targetIdx := range targets {
			a, b := orderedNodePair(i, targetIdx)
			key := [2]int{a, b}
			if _, ok := seen[key]; ok {
				if diag != nil {
					diag.DuplicateSelections++
				}
				continue
			}
			if math.IsInf(dist[nodes[targetIdx].CellIndex], 1) {
				continue
			}
			seen[key] = struct{}{}
			if diag != nil {
				diag.CreatedLinks++
				recordSettlementLinkPairKind(&diag.CreatedPairKind, nodes[a].Kind, nodes[b].Kind)
			}
			links = append(links, SettlementLink{
				From:       a,
				To:         b,
				TravelCost: dist[nodes[targetIdx].CellIndex],
				Path:       reconstructSettlementPath(node.CellIndex, nodes[targetIdx].CellIndex, prev),
			})
		}
	}
	sort.Slice(links, func(i, j int) bool { return links[i].TravelCost < links[j].TravelCost })
	return links
}

func candidateLinkTargets(
	sourceIdx int,
	source SettlementNode,
	nodes []SettlementNode,
	nodeByCell map[int]int,
	dist []float64,
	settlements *SettlementResult,
	resources *ResourceResult,
	maxTravel float64,
) ([]int, settlementLinkCandidateDiagnostics) {
	type target struct {
		idx   int
		score float64
	}
	targets := make([]target, 0)
	var diag settlementLinkCandidateDiagnostics
	for j, node := range nodes {
		if j == sourceIdx || math.IsInf(dist[node.CellIndex], 1) {
			continue
		}
		if source.Kind <= SettlementNodeVillage && node.Kind < source.Kind {
			continue
		}
		if source.Kind >= SettlementNodeTown && node.Kind == SettlementNodeHamlet {
			continue
		}
		if dist[node.CellIndex] > maxTravel {
			if dist[node.CellIndex] <= maxTravel*1.10 {
				diag.nearTargets++
				incrementSettlementKindCount(&diag.nearTargetKind, node.Kind)
			}
			continue
		}
		diag.reachableTargets++
		incrementSettlementKindCount(&diag.reachableTargetKind, node.Kind)
		value := 0.55*node.Score + 0.20*settlementNodeEffectiveRank(node) - 0.07*dist[node.CellIndex]
		if node.Coastal {
			value += 0.06
		}
		if settlementResourceExceptional(resources, node.CellIndex) {
			value += 0.04
		}
		targets = append(targets, target{idx: j, score: value})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].score > targets[j].score })
	limit := nodeLinkTargetLimit(source)
	if len(targets) < limit {
		limit = len(targets)
	}
	out := make([]int, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, targets[i].idx)
	}
	return out, diag
}

type settlementLinkCandidateDiagnostics struct {
	reachableTargets    int
	reachableTargetKind [4]int
	nearTargets         int
	nearTargetKind      [4]int
}

func recordSettlementLinkFormationSource(
	diag *SettlementLinkFormationDiagnostics,
	kind SettlementNodeKind,
	candidateDiag settlementLinkCandidateDiagnostics,
	selectedTargets []int,
	nodes []SettlementNode,
) {
	if diag == nil {
		return
	}
	reachableTargets := candidateDiag.reachableTargets
	nearTargets := reachableTargets + candidateDiag.nearTargets
	selectedTargetCount := len(selectedTargets)
	diag.SourceNodes++
	if int(kind) >= 0 && int(kind) < len(diag.SourceByKind) {
		diag.SourceByKind[kind]++
		diag.ReachableByKind[kind] += reachableTargets
		diag.NearByKind[kind] += nearTargets
		diag.SelectedByKind[kind] += selectedTargetCount
	}
	diag.ReachableTargets += reachableTargets
	diag.NearTargets += nearTargets
	diag.SelectedTargets += selectedTargetCount
	addSettlementKindCounts(&diag.ReachableTargetKind, candidateDiag.reachableTargetKind)
	addSettlementKindCounts(&diag.NearTargetKind, candidateDiag.reachableTargetKind)
	addSettlementKindCounts(&diag.NearTargetKind, candidateDiag.nearTargetKind)
	for _, targetIdx := range selectedTargets {
		if targetIdx < 0 || targetIdx >= len(nodes) {
			continue
		}
		incrementSettlementKindCount(&diag.SelectedTargetKind, nodes[targetIdx].Kind)
		recordSettlementLinkPairKind(&diag.SelectedPairKind, kind, nodes[targetIdx].Kind)
	}
	if reachableTargets == 0 {
		diag.NoReachableSources++
		if int(kind) >= 0 && int(kind) < len(diag.NoReachableByKind) {
			diag.NoReachableByKind[kind]++
		}
	}
	if selectedTargetCount == 0 {
		diag.NoSelectedSources++
		if int(kind) >= 0 && int(kind) < len(diag.NoSelectedByKind) {
			diag.NoSelectedByKind[kind]++
		}
	}
	if reachableTargets > selectedTargetCount {
		diag.TargetLimitedNodes++
		if int(kind) >= 0 && int(kind) < len(diag.TargetLimitedByKind) {
			diag.TargetLimitedByKind[kind]++
		}
	}
}

func incrementSettlementKindCount(counts *[4]int, kind SettlementNodeKind) {
	if counts == nil || int(kind) < 0 || int(kind) >= len(counts) {
		return
	}
	counts[kind]++
}

func addSettlementKindCounts(dst *[4]int, src [4]int) {
	if dst == nil {
		return
	}
	for i, count := range src {
		dst[i] += count
	}
}

func recordSettlementLinkPairKind(matrix *[4][4]int, a, b SettlementNodeKind) {
	if matrix == nil || int(a) < 0 || int(a) >= len(matrix) || int(b) < 0 || int(b) >= len(matrix) {
		return
	}
	x, y := int(a), int(b)
	if x > y {
		x, y = y, x
	}
	matrix[x][y]++
}

func pruneIsolatedNodes(
	nodes []SettlementNode,
	links []SettlementLink,
	nodeByCell []int,
	settlements *SettlementResult,
	resources *ResourceResult,
) ([]SettlementNode, []SettlementLink, []int) {
	if len(nodes) == 0 {
		return nodes, links, nodeByCell
	}
	degree := make([]int, len(nodes))
	for _, link := range links {
		degree[link.From]++
		degree[link.To]++
	}
	keep := make([]bool, len(nodes))
	newNodes := make([]SettlementNode, 0, len(nodes))
	oldToNew := make([]int, len(nodes))
	for i := range oldToNew {
		oldToNew[i] = -1
	}
	for i, node := range nodes {
		exceptional := settlementResourceExceptional(resources, node.CellIndex) || node.Coastal || node.River
		if degree[i] == 0 {
			switch node.Kind {
			case SettlementNodeHamlet:
				if !settlementResourceExceptional(resources, node.CellIndex) {
					continue
				}
			case SettlementNodeVillage:
				if !settlementResourceExceptional(resources, node.CellIndex) && settlements.Diagnostics.Suitability[node.CellIndex] < 0.72 {
					continue
				}
			case SettlementNodeTown:
				if !exceptional && settlements.Diagnostics.Suitability[node.CellIndex] < 0.70 {
					continue
				}
			}
		}
		if degree[i] <= 1 && node.Kind == SettlementNodeHamlet && !exceptional && settlements.Diagnostics.AccessScore[node.CellIndex] < 0.58 {
			continue
		}
		keep[i] = true
		oldToNew[i] = len(newNodes)
		node.ID = len(newNodes)
		newNodes = append(newNodes, node)
	}
	newLinks := make([]SettlementLink, 0, len(links))
	for _, link := range links {
		if !keep[link.From] || !keep[link.To] {
			continue
		}
		newLinks = append(newLinks, SettlementLink{
			From:       oldToNew[link.From],
			To:         oldToNew[link.To],
			TravelCost: link.TravelCost,
			Path:       append([]int(nil), link.Path...),
		})
	}
	for i := range nodeByCell {
		if nodeByCell[i] >= 0 {
			nodeByCell[i] = oldToNew[nodeByCell[i]]
		}
	}
	return newNodes, newLinks, nodeByCell
}

func nodeSpacingDeg(kind SettlementNodeKind, settings SettlementNetworkSettings) float64 {
	switch kind {
	case SettlementNodeCity:
		return settings.CitySpacingDeg
	case SettlementNodeTown:
		return settings.TownSpacingDeg
	case SettlementNodeVillage:
		return settings.VillageSpacingDeg
	default:
		return settings.HamletSpacingDeg
	}
}

func nodeTravelLimit(kind SettlementNodeKind, settings SettlementNetworkSettings) float64 {
	switch kind {
	case SettlementNodeCity:
		return settings.CityMaxTravel
	case SettlementNodeTown:
		return settings.TownMaxTravel
	case SettlementNodeVillage:
		return settings.VillageMaxTravel
	default:
		return settings.HamletMaxTravel
	}
}

func nodeTravelLimitForNode(node SettlementNode, settings SettlementNetworkSettings) float64 {
	return nodeTravelLimit(node.Kind, settings) * (0.70 + 0.30*SettlementNodePhysicalSupportWeight(node))
}

func nodeLinkTargetLimit(node SettlementNode) int {
	base := 1
	switch node.Kind {
	case SettlementNodeVillage:
		base = 2
	case SettlementNodeTown:
		base = 2
	case SettlementNodeCity:
		base = 3
	}
	if node.Kind < SettlementNodeTown || node.PhysicalSupportArea <= 0 {
		return base
	}
	limit := int(math.Round(float64(base) * SettlementNodePhysicalSupportWeight(node)))
	if limit < 1 {
		return 1
	}
	if limit > base {
		return base
	}
	return limit
}

func greatCircleDistanceDeg(a, b Vector3D) float64 {
	dot := a.X*b.X + a.Y*b.Y + a.Z*b.Z
	if dot > 1 {
		dot = 1
	}
	if dot < -1 {
		dot = -1
	}
	return math.Acos(dot) * 180 / math.Pi
}

func orderedNodePair(a, b int) (int, int) {
	if a < b {
		return a, b
	}
	return b, a
}

type pathState struct {
	cell int
	dist float64
}

type pathHeap []pathState

func (h pathHeap) Len() int            { return len(h) }
func (h pathHeap) Less(i, j int) bool  { return h[i].dist < h[j].dist }
func (h pathHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *pathHeap) Push(x interface{}) { *h = append(*h, x.(pathState)) }
func (h *pathHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func shortestPathsFromNode(start int, cells []VoronoiCell, movementCost []float64, maxDist float64) ([]float64, []int) {
	dist := make([]float64, len(cells))
	prev := make([]int, len(cells))
	for i := range dist {
		dist[i] = math.Inf(1)
		prev[i] = -1
	}
	dist[start] = 0
	stepScale := meshPathCostResolutionScale(len(cells))
	pq := &pathHeap{{cell: start, dist: 0}}
	heap.Init(pq)
	for pq.Len() > 0 {
		cur := heap.Pop(pq).(pathState)
		if cur.dist > dist[cur.cell] || cur.dist > maxDist {
			continue
		}
		for _, neighbor := range cells[cur.cell].NeighborSiteIndices {
			j := int(neighbor)
			if j < 0 || j >= len(cells) || math.IsInf(movementCost[j], 1) {
				continue
			}
			step := 0.5 * (movementCost[cur.cell] + movementCost[j]) * stepScale
			nd := cur.dist + step
			if nd < dist[j] && nd <= maxDist {
				dist[j] = nd
				prev[j] = cur.cell
				heap.Push(pq, pathState{cell: j, dist: nd})
			}
		}
	}
	return dist, prev
}

func reconstructSettlementPath(start, goal int, prev []int) []int {
	if start == goal {
		return []int{start}
	}
	path := make([]int, 0)
	cur := goal
	for cur >= 0 {
		path = append(path, cur)
		if cur == start {
			break
		}
		cur = prev[cur]
	}
	if len(path) == 0 || path[len(path)-1] != start {
		return nil
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}
