package terrain

import "testing"

func TestMergeLocalRefinementArtifactsSharedBoundary(t *testing.T) {
	a := &LocalRefinementArtifact{
		Cell: 10,
		BoundaryCrossings: []LocalCellBoundaryCrossing{
			{Neighbor: 11, BearingDeg: 90, Offset: 0.012, WidthPx: 8, Strength: 0.30, Kind: "channel"},
			{Neighbor: 11, BearingDeg: 90, Offset: -0.021, WidthPx: 6, Strength: 0.22, Kind: "channel"},
		},
	}
	b := &LocalRefinementArtifact{
		Cell: 11,
		BoundaryCrossings: []LocalCellBoundaryCrossing{
			{Neighbor: 10, BearingDeg: 270, Offset: -0.011, WidthPx: 10, Strength: 0.28, Kind: "channel"},
			{Neighbor: 10, BearingDeg: 270, Offset: 0.020, WidthPx: 4, Strength: 0.24, Kind: "channel"},
		},
	}

	merge := MergeLocalRefinementArtifacts(a, b)
	if merge == nil {
		t.Fatalf("expected merge")
	}
	if len(merge.Crossings) != 2 {
		t.Fatalf("expected 2 merged crossings, got %d", len(merge.Crossings))
	}
	if len(merge.UnmatchedA) != 0 || len(merge.UnmatchedB) != 0 {
		t.Fatalf("expected no unmatched crossings, got %d and %d", len(merge.UnmatchedA), len(merge.UnmatchedB))
	}
	if merge.Crossings[0].CellA != 10 || merge.Crossings[0].CellB != 11 {
		t.Fatalf("unexpected merge cells: %+v", merge.Crossings[0])
	}
	if merge.Crossings[0].SourceCount != 2 {
		t.Fatalf("expected source count 2, got %d", merge.Crossings[0].SourceCount)
	}
}

func TestBuildSharedBoundaryContractVersioning(t *testing.T) {
	a := &LocalRefinementArtifact{
		Cell:              10,
		Seed:              1,
		SettingsSignature: "sig-a",
		BoundaryCrossings: []LocalCellBoundaryCrossing{
			{Neighbor: 11, BearingDeg: 90, Offset: 0.010, WidthPx: 8, Strength: 0.30, Kind: "channel"},
		},
	}
	b := &LocalRefinementArtifact{
		Cell:              11,
		Seed:              1,
		SettingsSignature: "sig-b",
		BoundaryCrossings: []LocalCellBoundaryCrossing{
			{Neighbor: 10, BearingDeg: 270, Offset: -0.010, WidthPx: 8, Strength: 0.30, Kind: "channel"},
		},
	}
	contract1 := BuildSharedBoundaryContract(nil, a, b)
	if contract1 == nil || contract1.Version != 1 {
		t.Fatalf("expected version 1 contract, got %+v", contract1)
	}
	contract2 := BuildSharedBoundaryContract(contract1, a, b)
	if contract2 == nil || contract2.Version != 1 {
		t.Fatalf("expected unchanged contract to keep version 1, got %+v", contract2)
	}

	b.BoundaryCrossings[0].Strength = 0.45
	contract3 := BuildSharedBoundaryContract(contract2, a, b)
	if contract3 == nil || contract3.Version != 2 {
		t.Fatalf("expected changed contract to bump version to 2, got %+v", contract3)
	}
}

func TestApplyArtifactToBoundaryContractMarksOtherCellStale(t *testing.T) {
	prev := &SharedBoundaryContract{
		CellA:   10,
		CellB:   11,
		Version: 3,
		Crossings: []CanonicalBoundaryCrossing{
			{CellA: 10, CellB: 11, Kind: "channel", OffsetA: 0.010, OffsetB: -0.010, WidthPx: 8, Strength: 0.30},
		},
		Sources: []BoundaryContractSource{
			{Cell: 10, Seed: 1, SettingsSignature: "a"},
			{Cell: 11, Seed: 1, SettingsSignature: "b"},
		},
	}
	updated := &LocalRefinementArtifact{
		Cell:              10,
		Seed:              2,
		SettingsSignature: "a2",
		BoundaryCrossings: []LocalCellBoundaryCrossing{
			{Neighbor: 11, BearingDeg: 90, Offset: 0.025, WidthPx: 10, Strength: 0.45, Kind: "channel"},
		},
	}
	next := ApplyArtifactToBoundaryContract(prev, updated)
	if next == nil {
		t.Fatalf("expected updated contract")
	}
	if next.Version != 4 {
		t.Fatalf("expected version bump to 4, got %d", next.Version)
	}
	if len(next.DirtyCells) != 1 || next.DirtyCells[0] != 11 {
		t.Fatalf("expected opposite cell 11 to be marked dirty, got %+v", next.DirtyCells)
	}
	if len(next.Crossings) != 1 || next.Crossings[0].OffsetA != 0.025 {
		t.Fatalf("expected updated crossing to be applied, got %+v", next.Crossings)
	}
}

func TestBuildRefinementRerunQueueAndClear(t *testing.T) {
	contract := &SharedBoundaryContract{
		CellA:      10,
		CellB:      11,
		Version:    2,
		DirtyCells: []int{11, 10},
	}
	queue := BuildRefinementRerunQueue(contract)
	if len(queue) != 2 || queue[0] != 10 || queue[1] != 11 {
		t.Fatalf("unexpected rerun queue: %+v", queue)
	}
	cleared := ClearBoundaryDirtyCell(contract, 10)
	if len(cleared.DirtyCells) != 1 || cleared.DirtyCells[0] != 11 {
		t.Fatalf("unexpected dirty cells after clear: %+v", cleared.DirtyCells)
	}
}
