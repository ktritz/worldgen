package climgen

import (
	"math"
	"testing"
)

func TestBuildSettlementNetworkCreatesLinkBetweenNearbyPeaks(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0.99, Y: 0.1, Z: 0},
		{X: 0.96, Y: 0.2, Z: 0},
		{X: 0.93, Y: 0.3, Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2}},
	}
	settlements := &SettlementResult{
		Classes: []SettlementClass{SettlementFavorable, SettlementMarginal, SettlementMarginal, SettlementFavorable},
		Diagnostics: &SettlementDiagnostics{
			WaterScore:    []float64{0.8, 0.5, 0.5, 0.8},
			TerrainScore:  []float64{0.8, 0.7, 0.7, 0.8},
			SoilScore:     []float64{0.8, 0.6, 0.6, 0.8},
			AccessScore:   []float64{0.7, 0.4, 0.4, 0.7},
			ResourceScore: []float64{0.4, 0.3, 0.3, 0.4},
			HazardPenalty: []float64{0.0, 0.0, 0.0, 0.0},
			RiverBonus:    []float64{0.4, 0.2, 0.2, 0.4},
			CoastalBonus:  []float64{0.0, 0.0, 0.0, 0.0},
			Suitability:   []float64{0.75, 0.40, 0.40, 0.74},
		},
	}
	population := &PopulationResult{
		Classes: []PopulationClass{PopulationDenseRural, PopulationSparseFrontier, PopulationSparseFrontier, PopulationDenseRural},
		Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: []float64{0.66, 0.24, 0.25, 0.64},
			UrbanPotential:   []float64{0.57, 0.18, 0.19, 0.56},
		},
	}
	soils := &SoilResult{Diagnostics: &SoilDiagnostics{
		Relief:    []float64{100, 120, 120, 100},
		Rockiness: []float64{0.1, 0.1, 0.1, 0.1},
	}}
	biomes := &BiomeResult{Diagnostics: &BiomeDiagnostics{
		AnnualIceFraction: []float64{0, 0, 0, 0},
		AridityRatio:      []float64{1, 1, 1, 1},
		WetlandAffinity:   []float64{0, 0, 0, 0},
	}}
	elevation := []float64{100, 100, 100, 100}

	result := BuildSettlementNetwork(sites, cells, settlements, population, biomes, soils, nil, elevation, 0, DefaultSettlementNetworkSettings())
	if len(result.Nodes) < 2 {
		t.Fatalf("expected at least 2 nodes, got %d", len(result.Nodes))
	}
	if len(result.Links) == 0 {
		t.Fatalf("expected at least 1 link between nearby supported nodes")
	}
}

func TestLocalSettlementPeakScalesNeighborhoodByMeshResolution(t *testing.T) {
	cellCount := 40962
	cells := make([]VoronoiCell, cellCount)
	for i := range cells {
		cells[i].SiteIndex = int32(i)
	}
	cells[0].NeighborSiteIndices = []int32{1}
	cells[1].NeighborSiteIndices = []int32{0, 2}
	cells[2].NeighborSiteIndices = []int32{1}
	population := &PopulationResult{
		Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: make([]float64, cellCount),
			UrbanPotential:   make([]float64, cellCount),
		},
	}
	population.Diagnostics.CarryingCapacity[0] = 0.50
	population.Diagnostics.UrbanPotential[0] = 0.50
	population.Diagnostics.CarryingCapacity[2] = 0.56
	population.Diagnostics.UrbanPotential[2] = 0.56

	if isLocalSettlementPeak(0, 0.50, cells, population) {
		t.Fatalf("expected second-hop stronger candidate to suppress local peak at refined resolution")
	}
}

func TestCityCandidateRequiresPhysicalCluster(t *testing.T) {
	cellCount := 40962
	cells := make([]VoronoiCell, cellCount)
	for i := range cells {
		cells[i].SiteIndex = int32(i)
	}
	cells[0].NeighborSiteIndices = []int32{1, 2, 3, 4, 5, 6, 7, 8}
	for i := 1; i <= 8; i++ {
		cells[i].NeighborSiteIndices = []int32{0}
	}
	population := &PopulationResult{
		Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: make([]float64, cellCount),
			UrbanPotential:   make([]float64, cellCount),
		},
	}
	settings := DefaultSettlementNetworkSettings()
	population.Diagnostics.CarryingCapacity[0] = settings.TownThreshold
	population.Diagnostics.UrbanPotential[0] = settings.CityThreshold + 0.02
	if physicallySignificantCityCandidate(0, cells, population, settings) {
		t.Fatalf("expected isolated high-resolution city-like spike to stay below major hub rank")
	}

	for i := 0; i < 4; i++ {
		population.Diagnostics.CarryingCapacity[i] = settings.TownThreshold
		population.Diagnostics.UrbanPotential[i] = settings.CityThreshold + 0.02
	}
	if !physicallySignificantCityCandidate(0, cells, population, settings) {
		t.Fatalf("expected physically meaningful cluster of city-like cells to qualify")
	}
}

func TestSettlementNodePhysicalSupportAreaScalesByResolution(t *testing.T) {
	const width, height = 64, 64
	cellCount := 40962
	cells := hexLatticeMesh(width, height, cellCount)
	center := hexLatticeIndex(width, height, width/2, height/2)
	population := &PopulationResult{
		Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: make([]float64, cellCount),
			UrbanPotential:   make([]float64, cellCount),
		},
	}
	settings := DefaultSettlementNetworkSettings()
	population.Diagnostics.CarryingCapacity[center] = settings.TownThreshold + 0.03
	population.Diagnostics.UrbanPotential[center] = settings.TownThreshold + 0.03
	area := SettlementNodePhysicalSupportArea(center, SettlementNodeTown, cells, population, settings)
	if area >= 0.5 {
		t.Fatalf("expected isolated refined town-like spike to have sub-baseline support area, got %.2f", area)
	}

	// One baseline hop of the refined window supported: a physically meaningful
	// catchment rather than a single refined-cell spike.
	for _, idx := range hexLatticeNeighborhood(width, height, width/2, height/2, 1, nil) {
		population.Diagnostics.CarryingCapacity[idx] = settings.TownThreshold + 0.03
		population.Diagnostics.UrbanPotential[idx] = settings.TownThreshold + 0.03
	}
	area = SettlementNodePhysicalSupportArea(center, SettlementNodeTown, cells, population, settings)
	if area < 0.75 {
		t.Fatalf("expected clustered town-like cells to reach regional support area, got %.2f", area)
	}
}

func TestSettlementTownCandidateDemotesWithoutPhysicalSupport(t *testing.T) {
	const width, height = 64, 64
	cellCount := 40962
	cells := hexLatticeMesh(width, height, cellCount)
	center := hexLatticeIndex(width, height, width/2, height/2)
	population := &PopulationResult{
		Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: make([]float64, cellCount),
			UrbanPotential:   make([]float64, cellCount),
		},
	}
	settings := DefaultSettlementNetworkSettings()
	population.Diagnostics.CarryingCapacity[center] = settings.TownThreshold + 0.03
	population.Diagnostics.UrbanPotential[center] = settings.TownThreshold + 0.03
	kind, ok := classifySettlementNodeCandidateAt(center, cells, population, settings)
	if !ok {
		t.Fatalf("expected isolated refined candidate to retain a lower-rank settlement role")
	}
	if kind >= SettlementNodeTown {
		t.Fatalf("expected isolated refined town-like spike to demote below regional rank, got %s", SettlementNodeKindName(kind))
	}

	for _, idx := range hexLatticeNeighborhood(width, height, width/2, height/2, 1, nil) {
		population.Diagnostics.CarryingCapacity[idx] = settings.TownThreshold + 0.03
		population.Diagnostics.UrbanPotential[idx] = settings.TownThreshold + 0.03
	}
	kind, ok = classifySettlementNodeCandidateAt(center, cells, population, settings)
	if !ok || kind != SettlementNodeTown {
		t.Fatalf("expected physically supported cluster to stay regional, got %s ok=%v", SettlementNodeKindName(kind), ok)
	}
}

func TestSettlementSpacingDiagnosticsRecordRejectedSupport(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: math.Cos(2 * math.Pi / 180), Y: math.Sin(2 * math.Pi / 180), Z: 0},
		{X: math.Cos(30 * math.Pi / 180), Y: math.Sin(30 * math.Pi / 180), Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1}},
	}
	settings := DefaultSettlementNetworkSettings()
	population := &PopulationResult{
		Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: []float64{settings.TownThreshold + 0.03, settings.TownThreshold + 0.02, settings.VillageThreshold + 0.02},
			UrbanPotential:   []float64{settings.TownThreshold + 0.03, settings.TownThreshold + 0.02, settings.VillageThreshold},
		},
	}
	candidates := []SettlementNode{
		{CellIndex: 0, Kind: SettlementNodeTown, Score: 0.9},
		{CellIndex: 1, Kind: SettlementNodeTown, Score: 0.8},
		{CellIndex: 2, Kind: SettlementNodeVillage, Score: 0.7},
	}
	diag := SettlementNodeFormationDiagnostics{}

	kept := filterSettlementNodeCandidates(candidates, sites, cells, population, settings, &diag)
	if len(kept) != 2 {
		t.Fatalf("expected one close town candidate rejected, got %d kept", len(kept))
	}
	if diag.SpacingRejectedByKind[SettlementNodeTown] != 1 {
		t.Fatalf("expected rejected town diagnostic, got %+v", diag.SpacingRejectedByKind)
	}
	if diag.SpacingBlockerByKind[SettlementNodeTown] != 1 {
		t.Fatalf("expected town blocker diagnostic, got %+v", diag.SpacingBlockerByKind)
	}
	if diag.SpacingRejectSupportArea[SettlementNodeTown] <= 0 || diag.SpacingKeptSupportArea[SettlementNodeTown] <= 0 {
		t.Fatalf("expected spacing support areas to be recorded, kept=%+v rejected=%+v", diag.SpacingKeptSupportArea, diag.SpacingRejectSupportArea)
	}
}

// hexLatticeMesh builds a wrapped triangular lattice in which every lattice cell
// has exactly six neighbours, padded with isolated cells up to cellCount. Hop
// discs on this mesh hold the 3r²+3r cells a real Goldberg-mesh neighbourhood
// holds, which the ad-hoc star meshes elsewhere in these tests do not; support
// areas are only meaningful against a realistic neighbourhood size.
func hexLatticeMesh(width, height, cellCount int) []VoronoiCell {
	if cellCount < width*height {
		cellCount = width * height
	}
	cells := make([]VoronoiCell, cellCount)
	for i := range cells {
		cells[i].SiteIndex = int32(i)
	}
	for r := 0; r < height; r++ {
		for q := 0; q < width; q++ {
			idx := hexLatticeIndex(width, height, q, r)
			cells[idx].NeighborSiteIndices = []int32{
				int32(hexLatticeIndex(width, height, q+1, r)),
				int32(hexLatticeIndex(width, height, q-1, r)),
				int32(hexLatticeIndex(width, height, q, r+1)),
				int32(hexLatticeIndex(width, height, q, r-1)),
				int32(hexLatticeIndex(width, height, q+1, r-1)),
				int32(hexLatticeIndex(width, height, q-1, r+1)),
			}
		}
	}
	return cells
}

func hexLatticeIndex(width, height, q, r int) int {
	return ((r%height)+height)%height*width + ((q%width)+width)%width
}

func hexAxialDistance(dq, dr int) int {
	return (abs(dq) + abs(dr) + abs(dq+dr)) / 2
}

// hexLatticeNeighborhood returns the lattice cells (centre included) whose axial
// offset from (q0,r0) satisfies keep. Offsets are unwrapped, so callers must keep
// the neighbourhood small relative to the lattice.
func hexLatticeNeighborhood(width, height, q0, r0, radius int, keep func(dq, dr int) bool) []int {
	out := make([]int, 0, 3*radius*radius+3*radius+1)
	for dr := -radius; dr <= radius; dr++ {
		for dq := -radius; dq <= radius; dq++ {
			if hexAxialDistance(dq, dr) > radius {
				continue
			}
			if keep != nil && !keep(dq, dr) {
				continue
			}
			out = append(out, hexLatticeIndex(width, height, q0+dq, r0+dr))
		}
	}
	return out
}

// meshResolutionRefinements maps supported mesh sizes to their linear refinement
// factor against the L5 baseline: one L5 hop equals k hops at that resolution.
var meshResolutionRefinements = []struct {
	name      string
	cellCount int
	factor    int
}{
	{name: "L5", cellCount: 10242, factor: 1},
	{name: "L6", cellCount: 40962, factor: 2},
	{name: "L7", cellCount: 163842, factor: 4},
}

// TestSettlementNodePhysicalSupportAreaHoldsPhysicalNeighborhood is the R2.1
// invariance proof: the same physical neighbourhood, sampled at L5/L6/L7, must
// report the same support area. The legacy count×scale² conversion collapsed it
// by ~46% over the same range because a hop disc holds ~3r²+3r cells, not r².
func TestSettlementNodePhysicalSupportAreaHoldsPhysicalNeighborhood(t *testing.T) {
	const width, height = 64, 64
	settings := DefaultSettlementNetworkSettings()
	areas := make([]float64, 0, len(meshResolutionRefinements))
	legacy := make([]float64, 0, len(meshResolutionRefinements))
	for _, level := range meshResolutionRefinements {
		cells := hexLatticeMesh(width, height, level.cellCount)
		if got := meshResolutionAdjustedSteps(1, level.cellCount); got != level.factor {
			t.Fatalf("%s: expected a one-hop baseline radius of %d, got %d", level.name, level.factor, got)
		}
		center := hexLatticeIndex(width, height, width/2, height/2)
		disc := cellsWithinHops(cells, center, level.factor)
		wantDisc := 3*level.factor*level.factor + 3*level.factor
		if len(disc) != wantDisc {
			t.Fatalf("%s: expected a %d-cell hop disc, got %d", level.name, wantDisc, len(disc))
		}
		population := &PopulationResult{Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: make([]float64, level.cellCount),
			UrbanPotential:   make([]float64, level.cellCount),
		}}
		// Support one physical baseline hop in every direction (the whole window).
		supported := hexLatticeNeighborhood(width, height, width/2, height/2, level.factor, nil)
		for _, idx := range supported {
			population.Diagnostics.CarryingCapacity[idx] = settings.TownThreshold + 0.03
			population.Diagnostics.UrbanPotential[idx] = settings.TownThreshold + 0.03
		}
		areas = append(areas, SettlementNodePhysicalSupportArea(center, SettlementNodeTown, cells, population, settings))
		legacy = append(legacy, meshScaledTerritoryAreaCells(len(supported), level.cellCount))
	}
	for i, level := range meshResolutionRefinements {
		if math.Abs(areas[i]-areas[0]) > 0.02*areas[0] {
			t.Fatalf("%s: support area %.4f drifted more than 2%% from the L5 value %.4f", level.name, areas[i], areas[0])
		}
		if math.Abs(areas[i]-settlementSupportDiscBaselineCells) > 1e-9 {
			t.Fatalf("%s: expected a fully supported window to report the baseline seven cells, got %.4f", level.name, areas[i])
		}
	}
	// Guard the regression this replaces: the old conversion drifted far more.
	if legacy[len(legacy)-1] > 0.7*legacy[0] {
		t.Fatalf("expected the legacy conversion to collapse across resolutions, got %.3f then %.3f", legacy[0], legacy[len(legacy)-1])
	}
}

// TestSettlementNodePhysicalSupportAreaHoldsPartialNeighborhood repeats the
// invariance check for a partially supported window (a physical half-plane).
// Residual spread here is L5 quantization — the baseline window holds only seven
// samples, so any fraction is quantized to 1/7 — not resolution drift.
func TestSettlementNodePhysicalSupportAreaHoldsPartialNeighborhood(t *testing.T) {
	const width, height = 64, 64
	settings := DefaultSettlementNetworkSettings()
	areas := make([]float64, 0, len(meshResolutionRefinements))
	legacy := make([]float64, 0, len(meshResolutionRefinements))
	for _, level := range meshResolutionRefinements {
		cells := hexLatticeMesh(width, height, level.cellCount)
		center := hexLatticeIndex(width, height, width/2, height/2)
		population := &PopulationResult{Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: make([]float64, level.cellCount),
			UrbanPotential:   make([]float64, level.cellCount),
		}}
		// Cartesian x of an axial offset is dq + dr/2; support the x >= 0 half.
		supported := hexLatticeNeighborhood(width, height, width/2, height/2, level.factor, func(dq, dr int) bool {
			return float64(dq)+0.5*float64(dr) >= 0
		})
		for _, idx := range supported {
			population.Diagnostics.CarryingCapacity[idx] = settings.TownThreshold + 0.03
			population.Diagnostics.UrbanPotential[idx] = settings.TownThreshold + 0.03
		}
		areas = append(areas, SettlementNodePhysicalSupportArea(center, SettlementNodeTown, cells, population, settings))
		legacy = append(legacy, meshScaledTerritoryAreaCells(len(supported), level.cellCount))
	}
	for i, level := range meshResolutionRefinements {
		if math.Abs(areas[i]-areas[0]) > 0.08*areas[0] {
			t.Fatalf("%s: half-supported window %.4f drifted more than 8%% from the L5 value %.4f", level.name, areas[i], areas[0])
		}
	}
	if legacy[len(legacy)-1] > 0.7*legacy[0] {
		t.Fatalf("expected the legacy conversion to collapse across resolutions, got %.3f then %.3f", legacy[0], legacy[len(legacy)-1])
	}
}

// TestSettlementNodePhysicalSupportAreaBaselineFloor pins the L5 meaning of the
// downstream gates: an isolated supported node still reports exactly one
// baseline cell, so the 0.5 and 0.75 gates stay inert at the baseline mesh.
func TestSettlementNodePhysicalSupportAreaBaselineFloor(t *testing.T) {
	const width, height = 64, 64
	settings := DefaultSettlementNetworkSettings()
	cells := hexLatticeMesh(width, height, 10242)
	center := hexLatticeIndex(width, height, width/2, height/2)
	population := &PopulationResult{Diagnostics: &PopulationDiagnostics{
		CarryingCapacity: make([]float64, 10242),
		UrbanPotential:   make([]float64, 10242),
	}}
	population.Diagnostics.CarryingCapacity[center] = settings.TownThreshold + 0.03
	population.Diagnostics.UrbanPotential[center] = settings.TownThreshold + 0.03
	area := SettlementNodePhysicalSupportArea(center, SettlementNodeTown, cells, population, settings)
	if math.Abs(area-1.0) > 1e-9 {
		t.Fatalf("expected the baseline mesh floor to stay at one cell, got %.4f", area)
	}
	if area < 0.5 || area < 0.75 {
		t.Fatalf("expected the baseline floor to stay above the demotion gates, got %.4f", area)
	}
}

// TestProtoCivilizationRegionPopulationSupportHoldsPhysicalArea is the R2.2
// invariance proof: a region covering the same physical area reports the same
// support strength at L5/L6/L7 instead of the ~25% shrink the count×scale²
// conversion produced against the fixed threshold in the eligibility check.
func TestProtoCivilizationRegionPopulationSupportHoldsPhysicalArea(t *testing.T) {
	const width, height = 64, 64
	networkSettings := DefaultSettlementNetworkSettings()
	strengths := make([]float64, 0, len(meshResolutionRefinements))
	legacy := make([]float64, 0, len(meshResolutionRefinements))
	for _, level := range meshResolutionRefinements {
		radius := 2 * level.factor
		if got := meshResolutionAdjustedSteps(2, level.cellCount); got != radius {
			t.Fatalf("%s: expected a two-hop baseline radius of %d, got %d", level.name, radius, got)
		}
		cells := hexLatticeMesh(width, height, level.cellCount)
		center := hexLatticeIndex(width, height, width/2, height/2)
		population := &PopulationResult{Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: make([]float64, level.cellCount),
			UrbanPotential:   make([]float64, level.cellCount),
		}}
		supported := hexLatticeNeighborhood(width, height, width/2, height/2, radius, nil)
		for _, idx := range supported {
			population.Diagnostics.CarryingCapacity[idx] = networkSettings.TownThreshold + 0.03
			population.Diagnostics.UrbanPotential[idx] = networkSettings.TownThreshold + 0.03
		}
		network := &SettlementNetworkResult{Nodes: []SettlementNode{{ID: 0, CellIndex: center, Kind: SettlementNodeTown}}}
		region := SettlementRegion{NodeIndices: []int{0}, CenterNode: 0}
		strengths = append(strengths, ProtoCivilizationRegionPopulationSupportStrength(region, network, cells, population, networkSettings))
		legacy = append(legacy, meshScaledTerritoryAreaCells(3*radius*radius+3*radius, level.cellCount))
	}
	for i, level := range meshResolutionRefinements {
		if math.Abs(strengths[i]-strengths[0]) > 0.02*strengths[0] {
			t.Fatalf("%s: region support %.4f drifted more than 2%% from the L5 value %.4f", level.name, strengths[i], strengths[0])
		}
	}
	if math.Abs(strengths[0]-protoRegionSupportDiscBaselineCells) > 1e-9 {
		t.Fatalf("expected a fully supported baseline disc to report %.1f cells, got %.4f", protoRegionSupportDiscBaselineCells, strengths[0])
	}
	if legacy[len(legacy)-1] > 0.85*legacy[0] {
		t.Fatalf("expected the legacy conversion to shrink across resolutions, got %.3f then %.3f", legacy[0], legacy[len(legacy)-1])
	}
}

func TestSettlementPathLengthUsesPhysicalDistance(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0.999847695, Y: 0.017452406, Z: 0},
		{X: 0.999390827, Y: 0.034899497, Z: 0},
	}
	path := []int{0, 1, 2}
	length := settlementPathLengthDeg(path, sites)
	if length < 1.9 || length > 2.1 {
		t.Fatalf("expected about two degrees of physical path length, got %.3f", length)
	}
}

func mustResolveSettlementKindThresholds(t *testing.T, settings SettlementNetworkSettings, cellCount int) SettlementNetworkSettings {
	t.Helper()
	resolved, err := resolveSettlementKindThresholds(settings, cellCount)
	if err != nil {
		t.Fatalf("resolveSettlementKindThresholds(%d) failed: %v", cellCount, err)
	}
	return resolved
}

func TestResolveSettlementKindThresholdsUsesCalibratedLevelValues(t *testing.T) {
	settings := DefaultSettlementNetworkSettings()
	table := settings.KindCalibration
	if len(table) < 2 {
		t.Fatalf("expected a multi-level calibration table, got %d rows", len(table))
	}
	// The level-5 row is the reference: its scales are exactly 1.0, so the
	// resolved cut points equal the absolute settings there by construction.
	for kind := SettlementNodeHamlet; kind <= SettlementNodeCity; kind++ {
		if got := table[0].CarryingScale[kind]; got != 1.0 {
			t.Fatalf("level-5 carrying scale for kind %d = %v, want exactly 1.0", kind, got)
		}
		if got := table[0].UrbanScale[kind]; got != 1.0 {
			t.Fatalf("level-5 urban scale for kind %d = %v, want exactly 1.0", kind, got)
		}
	}
	reference := mustResolveSettlementKindThresholds(t, settings, table[0].Cells)
	for kind := SettlementNodeHamlet; kind <= SettlementNodeCity; kind++ {
		want := settlementNodeKindThreshold(kind, settings)
		if got := SettlementCarryingKindThreshold(kind, reference); got != want {
			t.Fatalf("level-5 carrying threshold for kind %d = %v, want the absolute %v", kind, got, want)
		}
		if got := SettlementUrbanKindThreshold(kind, reference); got != want {
			t.Fatalf("level-5 urban threshold for kind %d = %v, want the absolute %v", kind, got, want)
		}
	}
	for _, row := range table {
		resolved := mustResolveSettlementKindThresholds(t, settings, row.Cells)
		if resolved.resolved == nil {
			t.Fatalf("expected resolved thresholds for cells=%d", row.Cells)
		}
		for kind := SettlementNodeHamlet; kind <= SettlementNodeTown; kind++ {
			absolute := settlementNodeKindThreshold(kind, settings)
			wantCarrying := absolute * row.CarryingScale[kind]
			if got := SettlementCarryingKindThreshold(kind, resolved); math.Abs(got-wantCarrying) > 1e-9 {
				t.Fatalf("cells=%d kind=%d carrying threshold %.6f, want %.6f", row.Cells, kind, got, wantCarrying)
			}
			wantUrban := absolute * row.UrbanScale[kind]
			if got := SettlementUrbanKindThreshold(kind, resolved); math.Abs(got-wantUrban) > 1e-9 {
				t.Fatalf("cells=%d kind=%d urban threshold %.6f, want %.6f", row.Cells, kind, got, wantUrban)
			}
		}
		// The city cut stays absolute at every level.
		if got := SettlementUrbanKindThreshold(SettlementNodeCity, resolved); got != settings.CityThreshold {
			t.Fatalf("cells=%d city urban threshold %.6f, want absolute %.6f", row.Cells, got, settings.CityThreshold)
		}
	}
}

// The shipped scales must still reproduce the measured absolute cut points the
// calibration was derived from, so re-expressing the table as ratios did not
// move any level's classification.
func TestSettlementKindCalibrationReproducesMeasuredCutPoints(t *testing.T) {
	settings := DefaultSettlementNetworkSettings()
	measured := []struct {
		cells    int
		carrying [3]float64
		urban    [3]float64
	}{
		{10242, [3]float64{0.3800, 0.4600, 0.5500}, [3]float64{0.3800, 0.4600, 0.5500}},
		{40962, [3]float64{0.3876, 0.4705, 0.5507}, [3]float64{0.3890, 0.4704, 0.5496}},
		{163842, [3]float64{0.3244, 0.4714, 0.5558}, [3]float64{0.3544, 0.4738, 0.5500}},
		{655362, [3]float64{0.2612, 0.4723, 0.5609}, [3]float64{0.3198, 0.4772, 0.5504}},
	}
	for _, want := range measured {
		resolved := mustResolveSettlementKindThresholds(t, settings, want.cells)
		for kind := SettlementNodeHamlet; kind <= SettlementNodeTown; kind++ {
			if got := SettlementCarryingKindThreshold(kind, resolved); math.Abs(got-want.carrying[kind]) > 5e-5 {
				t.Fatalf("cells=%d kind=%d carrying threshold %.6f, want the measured %.4f", want.cells, kind, got, want.carrying[kind])
			}
			if got := SettlementUrbanKindThreshold(kind, resolved); math.Abs(got-want.urban[kind]) > 5e-5 {
				t.Fatalf("cells=%d kind=%d urban threshold %.6f, want the measured %.4f", want.cells, kind, got, want.urban[kind])
			}
		}
	}
}

// The absolute settings must stay the tuning knob at every resolution: the
// calibration table is a resolution correction, not a replacement.
func TestSettlementKindThresholdSettingsMoveEveryLevel(t *testing.T) {
	base := DefaultSettlementNetworkSettings()
	tuned := DefaultSettlementNetworkSettings()
	tuned.TownThreshold = 0.60
	for _, cells := range []int{10242, 40962, 163842, 655362} {
		baseResolved := mustResolveSettlementKindThresholds(t, base, cells)
		tunedResolved := mustResolveSettlementKindThresholds(t, tuned, cells)
		baseTown := SettlementCarryingKindThreshold(SettlementNodeTown, baseResolved)
		tunedTown := SettlementCarryingKindThreshold(SettlementNodeTown, tunedResolved)
		if tunedTown <= baseTown {
			t.Fatalf("cells=%d: raising TownThreshold left the resolved town cut at %.6f (base %.6f)", cells, tunedTown, baseTown)
		}
		// The move is proportional to the setting, so the ratio between the two
		// resolutions of the same level matches the ratio of the settings.
		wantRatio := tuned.TownThreshold / base.TownThreshold
		if got := tunedTown / baseTown; math.Abs(got-wantRatio) > 1e-9 {
			t.Fatalf("cells=%d: town cut scaled by %.6f, want %.6f", cells, got, wantRatio)
		}
		// Untouched kinds must not move.
		if got, want := SettlementCarryingKindThreshold(SettlementNodeVillage, tunedResolved), SettlementCarryingKindThreshold(SettlementNodeVillage, baseResolved); got != want {
			t.Fatalf("cells=%d: village cut moved to %.6f from %.6f when only TownThreshold changed", cells, got, want)
		}
	}
}

func TestResolveSettlementKindThresholdsInterpolatesAndClamps(t *testing.T) {
	settings := DefaultSettlementNetworkSettings()
	table := settings.KindCalibration
	hamletAbsolute := settings.HamletThreshold
	villageAbsolute := settings.VillageThreshold
	low := mustResolveSettlementKindThresholds(t, settings, table[0].Cells/4)
	wantLow := hamletAbsolute * table[0].CarryingScale[SettlementNodeHamlet]
	if got := SettlementCarryingKindThreshold(SettlementNodeHamlet, low); math.Abs(got-wantLow) > 1e-9 {
		t.Fatalf("below-range mesh clamped to %.6f, want %.6f", got, wantLow)
	}
	last := table[len(table)-1]
	high := mustResolveSettlementKindThresholds(t, settings, last.Cells*4)
	wantHigh := hamletAbsolute * last.CarryingScale[SettlementNodeHamlet]
	if got := SettlementCarryingKindThreshold(SettlementNodeHamlet, high); math.Abs(got-wantHigh) > 1e-9 {
		t.Fatalf("above-range mesh clamped to %.6f, want %.6f", got, wantHigh)
	}
	// A mesh halfway between two rows in log cell count lands halfway between
	// their cut points.
	mid := mustResolveSettlementKindThresholds(t, settings, int(math.Round(math.Sqrt(float64(table[0].Cells)*float64(table[1].Cells)))))
	want := 0.5 * villageAbsolute * (table[0].CarryingScale[SettlementNodeVillage] + table[1].CarryingScale[SettlementNodeVillage])
	if got := SettlementCarryingKindThreshold(SettlementNodeVillage, mid); math.Abs(got-want) > 1e-3 {
		t.Fatalf("interpolated village threshold %.6f, want about %.6f", got, want)
	}
}

func TestSettlementKindThresholdsFallBackToAbsoluteWithoutCalibration(t *testing.T) {
	settings := DefaultSettlementNetworkSettings()
	settings.KindCalibration = nil
	resolved := mustResolveSettlementKindThresholds(t, settings, 40962)
	if resolved.resolved != nil {
		t.Fatalf("expected no resolution without a calibration table")
	}
	if got := SettlementCarryingKindThreshold(SettlementNodeVillage, resolved); got != settings.VillageThreshold {
		t.Fatalf("village threshold = %v, want absolute %v", got, settings.VillageThreshold)
	}
	if got := SettlementUrbanKindThreshold(SettlementNodeCity, resolved); got != settings.CityThreshold {
		t.Fatalf("city threshold = %v, want absolute %v", got, settings.CityThreshold)
	}
}

// A non-positive Cells row makes the log-cell-count span infinite, which used to
// pass the span > 0 guard and produce NaN cut points — every comparison against
// which is false, so the world came back with no settlement nodes at all and no
// error. The table must be rejected loudly and the run must fall back to the
// finite absolute thresholds.
func TestResolveSettlementKindThresholdsRejectsNonPositiveCells(t *testing.T) {
	for _, cells := range []int{0, -1} {
		settings := DefaultSettlementNetworkSettings()
		settings.KindCalibration = []SettlementKindLevelCalibration{
			{Cells: cells, CarryingScale: [4]float64{1, 1, 1, 1}, UrbanScale: [4]float64{1, 1, 1, 1}},
			{Cells: 163842, CarryingScale: [4]float64{0.85, 1.02, 1.01, 1}, UrbanScale: [4]float64{0.93, 1.03, 1.0, 1}},
		}
		if err := ValidateSettlementKindCalibration(settings.KindCalibration); err == nil {
			t.Fatalf("Cells=%d: expected a validation error", cells)
		}
		resolved, err := resolveSettlementKindThresholds(settings, 40962)
		if err == nil {
			t.Fatalf("Cells=%d: expected resolveSettlementKindThresholds to report the bad table", cells)
		}
		if resolved.resolved != nil {
			t.Fatalf("Cells=%d: expected a fall back to the absolute thresholds", cells)
		}
		for kind := SettlementNodeHamlet; kind <= SettlementNodeCity; kind++ {
			carrying := SettlementCarryingKindThreshold(kind, resolved)
			urban := SettlementUrbanKindThreshold(kind, resolved)
			if math.IsNaN(carrying) || math.IsInf(carrying, 0) || math.IsNaN(urban) || math.IsInf(urban, 0) {
				t.Fatalf("Cells=%d kind=%d: non-finite thresholds carrying=%v urban=%v", cells, kind, carrying, urban)
			}
			if want := settlementNodeKindThreshold(kind, settings); carrying != want || urban != want {
				t.Fatalf("Cells=%d kind=%d: thresholds %v/%v, want the absolute %v", cells, kind, carrying, urban, want)
			}
		}
	}
}

func TestValidateSettlementKindCalibrationRejectsBadRows(t *testing.T) {
	if err := ValidateSettlementKindCalibration(DefaultSettlementNetworkSettings().KindCalibration); err != nil {
		t.Fatalf("shipped calibration table rejected: %v", err)
	}
	cases := map[string][]SettlementKindLevelCalibration{
		"unsorted cells": {
			{Cells: 40962, CarryingScale: [4]float64{1, 1, 1, 1}, UrbanScale: [4]float64{1, 1, 1, 1}},
			{Cells: 10242, CarryingScale: [4]float64{1, 1, 1, 1}, UrbanScale: [4]float64{1, 1, 1, 1}},
		},
		"duplicate cells": {
			{Cells: 10242, CarryingScale: [4]float64{1, 1, 1, 1}, UrbanScale: [4]float64{1, 1, 1, 1}},
			{Cells: 10242, CarryingScale: [4]float64{1, 1, 1, 1}, UrbanScale: [4]float64{1, 1, 1, 1}},
		},
		"zero scale": {
			{Cells: 10242, CarryingScale: [4]float64{1, 0, 1, 1}, UrbanScale: [4]float64{1, 1, 1, 1}},
		},
		"nan scale": {
			{Cells: 10242, CarryingScale: [4]float64{1, 1, 1, 1}, UrbanScale: [4]float64{1, math.NaN(), 1, 1}},
		},
		"inf scale": {
			{Cells: 10242, CarryingScale: [4]float64{1, 1, math.Inf(1), 1}, UrbanScale: [4]float64{1, 1, 1, 1}},
		},
	}
	for name, table := range cases {
		if err := ValidateSettlementKindCalibration(table); err == nil {
			t.Fatalf("%s: expected a validation error", name)
		}
	}
}

// The city cut is absolute at every level: a calibration row whose town cut
// lands above CityThreshold must clamp the town cut down rather than let the
// non-decreasing pass drag the city cut above the configured absolute.
func TestResolveSettlementKindThresholdsKeepsCityAbsolute(t *testing.T) {
	settings := DefaultSettlementNetworkSettings()
	// TownThreshold 0.55 * 1.4 = 0.77, well above the 0.64 city cut.
	settings.KindCalibration = []SettlementKindLevelCalibration{
		{Cells: 10242, CarryingScale: [4]float64{1, 1, 1.4, 1}, UrbanScale: [4]float64{1, 1, 1.4, 1}},
	}
	resolved := mustResolveSettlementKindThresholds(t, settings, 10242)
	if got := SettlementCarryingKindThreshold(SettlementNodeCity, resolved); got != settings.CityThreshold {
		t.Fatalf("city carrying cut = %.6f, want the absolute %.6f", got, settings.CityThreshold)
	}
	if got := SettlementUrbanKindThreshold(SettlementNodeCity, resolved); got != settings.CityThreshold {
		t.Fatalf("city urban cut = %.6f, want the absolute %.6f", got, settings.CityThreshold)
	}
	// The over-tall town cut is clamped down to the city cut, so the cut points
	// stay non-decreasing without overriding the pin.
	if got := SettlementCarryingKindThreshold(SettlementNodeTown, resolved); got != settings.CityThreshold {
		t.Fatalf("town carrying cut = %.6f, want it clamped to the city cut %.6f", got, settings.CityThreshold)
	}
	previous := 0.0
	for kind := SettlementNodeHamlet; kind <= SettlementNodeCity; kind++ {
		got := SettlementCarryingKindThreshold(kind, resolved)
		if got < previous {
			t.Fatalf("kind %d carrying cut %.6f is below the previous %.6f", kind, got, previous)
		}
		previous = got
	}
}
