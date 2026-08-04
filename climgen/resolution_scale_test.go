package climgen

import "testing"

func TestMeshPathCostResolutionScale(t *testing.T) {
	if scale := meshPathCostResolutionScale(10242); scale < 0.99 || scale > 1.01 {
		t.Fatalf("expected level-5-ish scale near 1.0, got %.3f", scale)
	}
	if scale := meshPathCostResolutionScale(40962); scale < 0.49 || scale > 0.51 {
		t.Fatalf("expected level-6-ish scale near 0.5, got %.3f", scale)
	}
	if scale := meshPathCostResolutionScale(163842); scale < 0.24 || scale > 0.26 {
		t.Fatalf("expected level-7-ish scale near 0.25, got %.3f", scale)
	}
	if scale := meshPathCostResolutionScale(655362); scale < 0.124 || scale > 0.126 {
		t.Fatalf("expected level-8-ish scale near 0.125, got %.3f", scale)
	}
	if scale := meshPathCostResolutionScale(2621442); scale != 0.125 {
		t.Fatalf("expected finer-than-L8 mesh to clamp at 0.125, got %.3f", scale)
	}
	if scale := meshPathCostResolutionScale(2562); scale != 1.0 {
		t.Fatalf("expected coarser-than-baseline mesh to clamp at 1.0, got %.3f", scale)
	}
}

func TestMeshResolutionAdjustedDiffusionIterations(t *testing.T) {
	if got := meshResolutionAdjustedDiffusionIterations(2, 10242); got != 2 {
		t.Fatalf("expected baseline mesh to keep 2 iterations, got %d", got)
	}
	if got := meshResolutionAdjustedDiffusionIterations(2, 40962); got != 8 {
		t.Fatalf("expected level-6 mesh to need 8 iterations (2 / 0.25 area scale), got %d", got)
	}
	if got := meshResolutionAdjustedDiffusionIterations(2, 163842); got != 32 {
		t.Fatalf("expected level-7 mesh to need 32 iterations, got %d", got)
	}
}

func TestBorderPressureFromCountIsResolutionInvariant(t *testing.T) {
	// A fixed physical border registers ~2x the adjacency pairs at level 6 and ~4x
	// at level 7; the derived pressure must match the baseline value.
	base := borderPressureFromCount(8, 10242)
	l6 := borderPressureFromCount(16, 40962)
	l7 := borderPressureFromCount(32, 163842)
	if diff := l6 - base; diff < -1e-3 || diff > 1e-3 {
		t.Fatalf("level-6 border pressure %.4f != baseline %.4f", l6, base)
	}
	if diff := l7 - base; diff < -1e-3 || diff > 1e-3 {
		t.Fatalf("level-7 border pressure %.4f != baseline %.4f", l7, base)
	}
	if got := borderPressureFromCount(1000, 10242); got != 0.03*12 {
		t.Fatalf("expected saturation cap at baseline, got %.4f", got)
	}
}

func TestMeshResolutionAdjustedSteps(t *testing.T) {
	if steps := meshResolutionAdjustedSteps(2, 10242); steps != 2 {
		t.Fatalf("expected baseline catchment to stay at 2 steps, got %d", steps)
	}
	if steps := meshResolutionAdjustedSteps(2, 40962); steps != 4 {
		t.Fatalf("expected doubled-resolution catchment to expand to 4 graph steps, got %d", steps)
	}
}

func TestMeshScaledTerritoryMeasuresStayStableAcrossResolution(t *testing.T) {
	baseLinear := meshScaledTerritoryLinearCells(40, 10242)
	highLinear := meshScaledTerritoryLinearCells(160, 40962)
	if diff := highLinear - baseLinear; diff < -0.01 || diff > 0.01 {
		t.Fatalf("expected equivalent physical territory linear size, got base=%.3f high=%.3f", baseLinear, highLinear)
	}

	baseArea := meshScaledTerritoryAreaCells(40, 10242)
	highArea := meshScaledTerritoryAreaCells(160, 40962)
	if diff := highArea - baseArea; diff < -0.01 || diff > 0.01 {
		t.Fatalf("expected equivalent physical territory area, got base=%.3f high=%.3f", baseArea, highArea)
	}
}

func TestResolutionAdjustedCatchmentReachesEquivalentPhysicalRadius(t *testing.T) {
	cells := make([]VoronoiCell, 40962)
	for i := 0; i < 4; i++ {
		cells[i].NeighborSiteIndices = append(cells[i].NeighborSiteIndices, int32(i+1))
		cells[i+1].NeighborSiteIndices = append(cells[i+1].NeighborSiteIndices, int32(i))
	}
	potential := make([]float64, len(cells))
	potential[4] = 1

	base := nodeCatchmentPotential(cells, 0, 2, potential)
	scaled := nodeCatchmentPotential(cells, 0, resolutionAdjustedCatchmentRadius(2, len(cells)), potential)
	if base != 0 {
		t.Fatalf("expected unscaled 2-step catchment not to reach distant refined cell, got %.3f", base)
	}
	if scaled <= 0 {
		t.Fatalf("expected scaled catchment to reach equivalent physical radius")
	}
}

func TestNodeCatchmentPotentialWeightsPhysicalDistanceAcrossResolution(t *testing.T) {
	coarseCells := make([]VoronoiCell, 10242)
	coarseCells[0].NeighborSiteIndices = []int32{1}
	coarseCells[1].NeighborSiteIndices = []int32{0, 2}
	coarseCells[2].NeighborSiteIndices = []int32{1}
	coarsePotential := make([]float64, len(coarseCells))
	coarsePotential[1] = 1

	fineCells := make([]VoronoiCell, 40962)
	for i := 0; i < 4; i++ {
		fineCells[i].NeighborSiteIndices = append(fineCells[i].NeighborSiteIndices, int32(i+1))
		fineCells[i+1].NeighborSiteIndices = append(fineCells[i+1].NeighborSiteIndices, int32(i))
	}
	finePotential := make([]float64, len(fineCells))
	finePotential[1] = 0.5
	finePotential[2] = 1
	finePotential[3] = 0.5

	coarse := nodeCatchmentPotential(coarseCells, 1, 1, coarsePotential)
	fine := nodeCatchmentPotential(fineCells, 2, resolutionAdjustedCatchmentRadius(1, len(fineCells)), finePotential)
	if diff := fine - coarse; diff < -0.001 || diff > 0.001 {
		t.Fatalf("expected equivalent physical catchment weighting, got coarse=%.3f fine=%.3f", coarse, fine)
	}
}

func TestNodeCatchmentPotentialLeavesUniformFieldsUnchanged(t *testing.T) {
	cells := make([]VoronoiCell, 10242)
	cells[0].NeighborSiteIndices = []int32{1}
	cells[1].NeighborSiteIndices = []int32{0, 2}
	cells[2].NeighborSiteIndices = []int32{1}
	potential := make([]float64, len(cells))
	potential[0] = 0.42
	potential[1] = 0.42
	potential[2] = 0.42

	got := nodeCatchmentPotential(cells, 1, 1, potential)
	if diff := got - 0.42; diff < -0.001 || diff > 0.001 {
		t.Fatalf("expected uniform catchment to remain unchanged, got %.3f", got)
	}
}

func TestResolutionAdjustedHydrologyInputsSpreadPhysicalSupport(t *testing.T) {
	coarseCells := makeLineCells(10242, 3)
	coarseElevation := make([]float64, len(coarseCells))
	for i := range coarseElevation {
		coarseElevation[i] = 100
	}
	coarseHydro := &HydrologyBiomeInputs{
		ChannelStrength: make([]float64, len(coarseCells)),
		CellClass:       make([]string, len(coarseCells)),
	}
	coarseHydro.ChannelStrength[0] = 1
	coarseHydro.CellClass[0] = "floodplain"
	coarse := ResolutionAdjustedHydrologyBiomeInputs(coarseCells, coarseElevation, 0, coarseHydro)
	if coarse.ChannelStrength[0] != 1 || coarse.ChannelStrength[1] != 0 {
		t.Fatalf("expected channel strength to remain on normalized source cells, got %.3f %.3f", coarse.ChannelStrength[0], coarse.ChannelStrength[1])
	}

	fineCells := makeLineCells(40962, 3)
	fineElevation := make([]float64, len(fineCells))
	for i := range fineElevation {
		fineElevation[i] = 100
	}
	fineHydro := &HydrologyBiomeInputs{
		ChannelStrength: make([]float64, len(fineCells)),
		CellClass:       make([]string, len(fineCells)),
	}
	fineHydro.ChannelStrength[0] = 1
	fineHydro.CellClass[0] = "floodplain"
	fine := ResolutionAdjustedHydrologyBiomeInputs(fineCells, fineElevation, 0, fineHydro)
	if fine.ChannelStrength[2] != 0 {
		t.Fatalf("expected channel strength to remain a raw centerline field, got %.3f", fine.ChannelStrength[2])
	}
	if fine.WetlandClassSupport[2] <= 0 {
		t.Fatalf("expected refined hydrology class support to reach equivalent physical distance")
	}
	if fine.RiparianChannelSupport[2] != 0 {
		t.Fatalf("expected riparian support to remain on normalized channel source cells, got %.3f", fine.RiparianChannelSupport[2])
	}
	if fine.DepositionalClassSupport[2] <= 0 {
		t.Fatalf("expected refined depositional support to reach equivalent physical distance")
	}
}

func makeLineCells(total, active int) []VoronoiCell {
	cells := make([]VoronoiCell, total)
	for i := 0; i < active; i++ {
		if i > 0 {
			cells[i].NeighborSiteIndices = append(cells[i].NeighborSiteIndices, int32(i-1))
		}
		if i+1 < active {
			cells[i].NeighborSiteIndices = append(cells[i].NeighborSiteIndices, int32(i+1))
		}
	}
	return cells
}
