package climgen

import "testing"

func TestBuildRiverTradeNetworkCreatesCorridorBetweenRiverCenters(t *testing.T) {
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.66, CarryingCapacity: 0.58, UrbanPotential: 0.54, River: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeVillage, Score: 0.52, CarryingCapacity: 0.48, UrbanPotential: 0.30, River: true},
			{ID: 2, CellIndex: 2, Kind: SettlementNodeTown, Score: 0.69, CarryingCapacity: 0.60, UrbanPotential: 0.56, River: true},
		},
		Links: []SettlementLink{
			{From: 0, To: 1, TravelCost: 2.0, Path: []int{0, 1}},
			{From: 1, To: 2, TravelCost: 2.1, Path: []int{1, 2}},
		},
		Regions: []SettlementRegion{
			{ID: 0, NodeIndices: []int{0, 1}, CenterNode: 0, MeanScore: 0.59, River: true},
			{ID: 1, NodeIndices: []int{2}, CenterNode: 2, MeanScore: 0.69, River: true},
		},
	}
	proto := &ProtoCivilizationResult{
		Civilizations: []ProtoCivilization{
			{ID: 0, RegionID: 0, CenterNode: 0, CenterKind: SettlementNodeTown, TerritoryCells: 28, MeanSupport: 0.40},
			{ID: 1, RegionID: 1, CenterNode: 2, CenterKind: SettlementNodeTown, TerritoryCells: 24, MeanSupport: 0.36},
		},
	}
	riverRoutes := &RiverRouteResult{
		Mode: DefaultRiverRouteSettings().Modes[0],
		Diagnostics: &RiverRouteDiagnostics{
			Navigability:         []float64{0.78, 0.82, 0.80},
			MainChannel:          []float64{0.84, 0.88, 0.86},
			TransferSupport:      []float64{0.42, 0.38, 0.44},
			PortageSuitability:   []float64{0.22, 0.24, 0.20},
			UpstreamTravelCost:   []float64{1.2, 1.1, 1.2},
			DownstreamTravelCost: []float64{0.9, 0.8, 0.9},
		},
	}

	result := BuildRiverTradeNetwork(cells, network, proto, riverRoutes, []float64{0.5, 0.3, 0.1}, DefaultRiverTradeSettings())
	if len(result.Corridors) == 0 {
		t.Fatalf("expected river trade corridor")
	}
	foundInter := false
	for _, corridor := range result.Corridors {
		if corridor.InterCivilization {
			foundInter = true
			if corridor.MeanNavigability <= 0 {
				t.Fatalf("expected inter-civilization river corridor to report mean navigability")
			}
			break
		}
	}
	if !foundInter {
		t.Fatalf("expected at least one inter-civilization river corridor")
	}
	if len(result.MajorPorts) == 0 {
		t.Fatalf("expected at least one major river port candidate")
	}
}

func TestBuildRiverTradeNetworkUsesNearbyRiverTerminalCells(t *testing.T) {
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2, 4}},
		{SiteIndex: 4, NeighborSiteIndices: []int32{3}},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.66, CarryingCapacity: 0.58, UrbanPotential: 0.54, River: true},
			{ID: 1, CellIndex: 4, Kind: SettlementNodeTown, Score: 0.69, CarryingCapacity: 0.60, UrbanPotential: 0.56, River: true},
		},
		Regions: []SettlementRegion{
			{ID: 0, NodeIndices: []int{0}, CenterNode: 0, MeanScore: 0.59, River: true},
			{ID: 1, NodeIndices: []int{1}, CenterNode: 1, MeanScore: 0.69, River: true},
		},
	}
	proto := &ProtoCivilizationResult{
		Civilizations: []ProtoCivilization{
			{ID: 0, RegionID: 0, CenterNode: 0, CenterKind: SettlementNodeTown, TerritoryCells: 28, MeanSupport: 0.40},
			{ID: 1, RegionID: 1, CenterNode: 1, CenterKind: SettlementNodeTown, TerritoryCells: 24, MeanSupport: 0.36},
		},
	}
	riverRoutes := &RiverRouteResult{
		Mode: DefaultRiverRouteSettings().Modes[0],
		Diagnostics: &RiverRouteDiagnostics{
			Navigability:         []float64{0.0, 0.78, 0.82, 0.80, 0.0},
			MainChannel:          []float64{0.0, 0.84, 0.88, 0.86, 0.0},
			TransferSupport:      []float64{0.22, 0.42, 0.38, 0.44, 0.22},
			PortageSuitability:   []float64{0.18, 0.22, 0.24, 0.20, 0.18},
			UpstreamTravelCost:   []float64{0, 1.2, 1.1, 1.2, 0},
			DownstreamTravelCost: []float64{0, 0.9, 0.8, 0.9, 0},
		},
	}

	result := BuildRiverTradeNetwork(cells, network, proto, riverRoutes, []float64{0.5, 0.4, 0.3, 0.2, 0.1}, DefaultRiverTradeSettings())
	if len(result.Corridors) == 0 {
		t.Fatalf("expected river trade corridor through nearby terminal cells")
	}
	if result.Diagnostics.NodeTerminalCell[0] != 1 {
		t.Fatalf("expected node 0 to use nearby river terminal 1, got %d", result.Diagnostics.NodeTerminalCell[0])
	}
	if result.Diagnostics.NodeTerminalCell[1] != 3 {
		t.Fatalf("expected node 1 to use nearby river terminal 3, got %d", result.Diagnostics.NodeTerminalCell[1])
	}
}

func TestRiverTradeTerminalCatchmentStepsScaleWithResolution(t *testing.T) {
	if steps := riverTradeTerminalCatchmentSteps(10242); steps != 1 {
		t.Fatalf("expected level-5-ish catchment of 1 step, got %d", steps)
	}
	if steps := riverTradeTerminalCatchmentSteps(40962); steps != 2 {
		t.Fatalf("expected level-6-ish catchment of 2 steps, got %d", steps)
	}
	if steps := riverTradeTerminalCatchmentSteps(163842); steps != 4 {
		t.Fatalf("expected level-7-ish catchment of 4 steps, got %d", steps)
	}
	if steps := riverTradeTerminalCatchmentSteps(655362); steps != 8 {
		t.Fatalf("expected level-8-ish catchment of 8 steps, got %d", steps)
	}
}

func TestRiverTradeFlowBetweenCivilizationsUsesPhysicalTerritoryScale(t *testing.T) {
	mode := DefaultRiverRouteSettings().Modes[0]
	a := ProtoCivilization{TerritoryCells: 28, MeanSupport: 0.40}
	b := ProtoCivilization{TerritoryCells: 24, MeanSupport: 0.36}
	refinedA := a
	refinedB := b
	refinedA.TerritoryCells = 112
	refinedB.TerritoryCells = 96
	centerA := SettlementNode{Score: 0.66}
	centerB := SettlementNode{Score: 0.69}

	base := riverTradeFlowBetweenCivilizations(a, b, centerA, centerB, 5.0, mode, 10242)
	refined := riverTradeFlowBetweenCivilizations(refinedA, refinedB, centerA, centerB, 5.0, mode, 40962)
	if diff := refined - base; diff < -0.01 || diff > 0.01 {
		t.Fatalf("expected equivalent physical territories to keep river flow stable, got base=%.3f refined=%.3f", base, refined)
	}
}

func TestRiverTerminalScorePenalizesPhysicalDistanceAcrossResolution(t *testing.T) {
	mode := DefaultRiverRouteSettings().Modes[0]
	makeRoutes := func(total, terminalIdx int) *RiverRouteResult {
		diag := &RiverRouteDiagnostics{
			Navigability:       make([]float64, total),
			MainChannel:        make([]float64, total),
			TransferSupport:    make([]float64, total),
			PortageSuitability: make([]float64, total),
		}
		diag.Navigability[terminalIdx] = 0.80
		diag.MainChannel[terminalIdx] = 0.85
		diag.TransferSupport[terminalIdx] = 0.40
		diag.PortageSuitability[terminalIdx] = 0.25
		return &RiverRouteResult{Mode: mode, Diagnostics: diag}
	}
	node := SettlementNode{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, River: true}

	coarseCells := makeLineCells(10242, 2)
	coarse, ok := bestRiverTerminalForNode(coarseCells, node, makeRoutes(2, 1), riverTradeTerminalCatchmentSteps(len(coarseCells)))
	if !ok || coarse.cell != 1 || coarse.distance != 1 {
		t.Fatalf("expected coarse terminal at cell 1 distance 1, got %+v ok=%v", coarse, ok)
	}
	expected := 0.62*0.80 + 0.16*0.85 + 0.14*0.40 + 0.08*0.25 - 0.035*1 + 0.06 + 0.04
	if diff := coarse.score - expected; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("expected baseline terminal score to be an exact no-op, got %.6f want %.6f", coarse.score, expected)
	}

	// The same physical offset is two hops on a level-6 mesh; the penalty must match.
	fineCells := makeLineCells(40962, 3)
	fine, ok := bestRiverTerminalForNode(fineCells, node, makeRoutes(3, 2), riverTradeTerminalCatchmentSteps(len(fineCells)))
	if !ok || fine.cell != 2 || fine.distance != 2 {
		t.Fatalf("expected fine terminal at cell 2 distance 2, got %+v ok=%v", fine, ok)
	}
	if diff := fine.score - coarse.score; diff < -1e-3 || diff > 1e-3 {
		t.Fatalf("expected resolution-invariant terminal score, got coarse=%.5f fine=%.5f", coarse.score, fine.score)
	}
}

func TestRiverLinkDirectionalTravelCostIsResolutionInvariant(t *testing.T) {
	mode := DefaultRiverRouteSettings().Modes[0]
	makeRoutes := func(pathCells int) *RiverRouteResult {
		diag := &RiverRouteDiagnostics{
			Navigability:         make([]float64, pathCells),
			TransferSupport:      make([]float64, pathCells),
			PortageSuitability:   make([]float64, pathCells),
			UpstreamTravelCost:   make([]float64, pathCells),
			DownstreamTravelCost: make([]float64, pathCells),
		}
		for i := 0; i < pathCells; i++ {
			diag.Navigability[i] = 0.80
			diag.TransferSupport[i] = 0.40
			diag.PortageSuitability[i] = 0.25
			diag.UpstreamTravelCost[i] = 1.2
			diag.DownstreamTravelCost[i] = 0.8
		}
		return &RiverRouteResult{Mode: mode, Diagnostics: diag}
	}
	makePath := func(pathCells int) []int {
		path := make([]int, pathCells)
		for i := range path {
			path[i] = i
		}
		return path
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0},
			{ID: 1, CellIndex: 1},
		},
	}
	elevation := []float64{0.1, 0.1}

	coarseLink := SettlementLink{From: 0, To: 1, Path: makePath(4)}
	coarseForward, coarseReverse := riverLinkDirectionalTravelCost(network, coarseLink, makeRoutes(4), elevation, 10242)
	if diff := coarseForward - 4*0.8; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("expected baseline forward cost to be an exact no-op, got %.6f want %.6f", coarseForward, 4*0.8)
	}
	if diff := coarseReverse - 4*1.2; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("expected baseline reverse cost to be an exact no-op, got %.6f want %.6f", coarseReverse, 4*1.2)
	}

	// The same physical link spans ~2x the cells at level 6; directional cost must match.
	fineLink := SettlementLink{From: 0, To: 1, Path: makePath(8)}
	fineForward, fineReverse := riverLinkDirectionalTravelCost(network, fineLink, makeRoutes(8), elevation, 40962)
	if rel := (fineForward - coarseForward) / coarseForward; rel < -1e-3 || rel > 1e-3 {
		t.Fatalf("expected resolution-invariant forward river link cost, got coarse=%.4f fine=%.4f", coarseForward, fineForward)
	}
	if rel := (fineReverse - coarseReverse) / coarseReverse; rel < -1e-3 || rel > 1e-3 {
		t.Fatalf("expected resolution-invariant reverse river link cost, got coarse=%.4f fine=%.4f", coarseReverse, fineReverse)
	}
}
