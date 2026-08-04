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

func TestResolveSettlementKindThresholdsUsesCalibratedLevelValues(t *testing.T) {
	settings := DefaultSettlementNetworkSettings()
	table := settings.KindCalibration
	if len(table) < 2 {
		t.Fatalf("expected a multi-level calibration table, got %d rows", len(table))
	}
	// The level-5 row must reproduce the absolute reference thresholds: that is
	// the correctness check on the calibration procedure itself.
	if got := table[0].Carrying[SettlementNodeHamlet]; math.Abs(got-settings.HamletThreshold) > 1e-9 {
		t.Fatalf("level-5 carrying hamlet cut = %.6f, want the absolute %.6f", got, settings.HamletThreshold)
	}
	for _, row := range table {
		resolved := resolveSettlementKindThresholds(settings, row.Cells)
		if resolved.resolved == nil {
			t.Fatalf("expected resolved thresholds for cells=%d", row.Cells)
		}
		for kind := SettlementNodeHamlet; kind <= SettlementNodeTown; kind++ {
			if got := SettlementCarryingKindThreshold(kind, resolved); math.Abs(got-row.Carrying[kind]) > 1e-9 {
				t.Fatalf("cells=%d kind=%d carrying threshold %.6f, want %.6f", row.Cells, kind, got, row.Carrying[kind])
			}
			if got := SettlementUrbanKindThreshold(kind, resolved); math.Abs(got-row.Urban[kind]) > 1e-9 {
				t.Fatalf("cells=%d kind=%d urban threshold %.6f, want %.6f", row.Cells, kind, got, row.Urban[kind])
			}
		}
		// The city cut stays absolute at every level.
		if got := SettlementUrbanKindThreshold(SettlementNodeCity, resolved); got != settings.CityThreshold {
			t.Fatalf("cells=%d city urban threshold %.6f, want absolute %.6f", row.Cells, got, settings.CityThreshold)
		}
	}
}

func TestResolveSettlementKindThresholdsInterpolatesAndClamps(t *testing.T) {
	settings := DefaultSettlementNetworkSettings()
	table := settings.KindCalibration
	low := resolveSettlementKindThresholds(settings, table[0].Cells/4)
	if got := SettlementCarryingKindThreshold(SettlementNodeHamlet, low); math.Abs(got-table[0].Carrying[SettlementNodeHamlet]) > 1e-9 {
		t.Fatalf("below-range mesh clamped to %.6f, want %.6f", got, table[0].Carrying[SettlementNodeHamlet])
	}
	last := table[len(table)-1]
	high := resolveSettlementKindThresholds(settings, last.Cells*4)
	if got := SettlementCarryingKindThreshold(SettlementNodeHamlet, high); math.Abs(got-last.Carrying[SettlementNodeHamlet]) > 1e-9 {
		t.Fatalf("above-range mesh clamped to %.6f, want %.6f", got, last.Carrying[SettlementNodeHamlet])
	}
	// A mesh halfway between two rows in log cell count lands halfway between
	// their cut points.
	mid := resolveSettlementKindThresholds(settings, int(math.Round(math.Sqrt(float64(table[0].Cells)*float64(table[1].Cells)))))
	want := 0.5 * (table[0].Carrying[SettlementNodeVillage] + table[1].Carrying[SettlementNodeVillage])
	if got := SettlementCarryingKindThreshold(SettlementNodeVillage, mid); math.Abs(got-want) > 1e-3 {
		t.Fatalf("interpolated village threshold %.6f, want about %.6f", got, want)
	}
}

func TestSettlementKindThresholdsFallBackToAbsoluteWithoutCalibration(t *testing.T) {
	settings := DefaultSettlementNetworkSettings()
	settings.KindCalibration = nil
	resolved := resolveSettlementKindThresholds(settings, 40962)
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
