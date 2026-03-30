package terrain

import "testing"

func TestLocalRefinementStoreSaveLoad(t *testing.T) {
	root := t.TempDir()
	store := NewLocalRefinementStore(root, 55)

	artifact := &LocalRefinementArtifact{
		Cell:              27706,
		Seed:              55,
		SettingsSignature: "sig",
		BoundaryCrossings: []LocalCellBoundaryCrossing{
			{Neighbor: 27707, BearingDeg: 90, Offset: 0.0, WidthPx: 8, Strength: 0.3, Kind: "channel"},
		},
	}
	if err := store.SaveArtifact(artifact); err != nil {
		t.Fatalf("save artifact: %v", err)
	}
	loadedArtifact, err := store.LoadArtifact(27706)
	if err != nil {
		t.Fatalf("load artifact: %v", err)
	}
	if loadedArtifact == nil || loadedArtifact.Cell != 27706 {
		t.Fatalf("unexpected loaded artifact: %+v", loadedArtifact)
	}

	contract := &SharedBoundaryContract{
		CellA:   27706,
		CellB:   27707,
		Version: 2,
		Crossings: []CanonicalBoundaryCrossing{
			{CellA: 27706, CellB: 27707, Kind: "channel", OffsetA: 0, OffsetB: 0, WidthPx: 10, Strength: 0.4},
		},
		DirtyCells: []int{27707},
	}
	if err := store.SaveContract(contract); err != nil {
		t.Fatalf("save contract: %v", err)
	}
	loadedContract, err := store.LoadContract(27706, 27707)
	if err != nil {
		t.Fatalf("load contract: %v", err)
	}
	if loadedContract == nil || loadedContract.Version != 2 {
		t.Fatalf("unexpected loaded contract: %+v", loadedContract)
	}
}
