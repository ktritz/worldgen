package climgen

import "testing"

func TestBuildMaritimeStopoverNodesFindsIslandStopover(t *testing.T) {
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1}},
	}
	elevation := []float64{-10, 8, -10}
	ports := &CoastalPortResult{
		Types: []CoastalPortType{
			CoastalPortOcean,
			CoastalPortIslandStopover,
			CoastalPortOcean,
		},
		Diagnostics: &CoastalPortDiagnostics{
			StopoverValue:   []float64{0, 0.72, 0},
			PortSuitability: []float64{0, 0.40, 0},
		},
	}
	stopovers := BuildMaritimeStopoverNodes(cells, nil, ports, elevation, 0.0)
	if len(stopovers) == 0 {
		t.Fatalf("expected maritime stopover node")
	}
	if stopovers[0].Kind != MaritimeStopoverIsland {
		t.Fatalf("expected island stopover kind, got %s", MaritimeStopoverKindName(stopovers[0].Kind))
	}
}

func TestBuildMaritimeStopoverNodesScalesSpacingByMeshResolution(t *testing.T) {
	cellCount := 40962
	cells := make([]VoronoiCell, cellCount)
	elevation := make([]float64, cellCount)
	portTypes := make([]CoastalPortType, cellCount)
	for i := range cells {
		cells[i].SiteIndex = int32(i)
		elevation[i] = 10
	}
	cells[0].NeighborSiteIndices = []int32{1}
	cells[1].NeighborSiteIndices = []int32{0, 2}
	cells[2].NeighborSiteIndices = []int32{1, 3}
	cells[3].NeighborSiteIndices = []int32{2}
	portTypes[0] = CoastalPortIslandStopover
	portTypes[3] = CoastalPortIslandStopover

	ports := &CoastalPortResult{
		Types: portTypes,
		Diagnostics: &CoastalPortDiagnostics{
			PortSuitability: make([]float64, cellCount),
			StopoverValue:   make([]float64, cellCount),
		},
	}
	ports.Diagnostics.PortSuitability[0] = 0.30
	ports.Diagnostics.PortSuitability[3] = 0.30
	ports.Diagnostics.StopoverValue[0] = 0.80
	ports.Diagnostics.StopoverValue[3] = 0.80

	stopovers := BuildMaritimeStopoverNodes(cells, nil, ports, elevation, 0)
	if len(stopovers) != 1 || stopovers[0].CellIndex != 0 {
		t.Fatalf("expected scaled stopover spacing to keep only cell 0, got %+v", stopovers)
	}
}

func TestBuildMaritimeStopoverDiagnosticsBucketComponentAreaByResolution(t *testing.T) {
	cellCount := 40962
	cells := make([]VoronoiCell, cellCount)
	elevation := make([]float64, cellCount)
	portTypes := make([]CoastalPortType, cellCount)
	for i := range cells {
		cells[i].SiteIndex = int32(i)
		elevation[i] = -10
	}
	cells[0].NeighborSiteIndices = []int32{1}
	cells[1].NeighborSiteIndices = []int32{0}
	elevation[0] = 10
	portTypes[0] = CoastalPortIslandStopover

	ports := &CoastalPortResult{
		Types: portTypes,
		Diagnostics: &CoastalPortDiagnostics{
			PortSuitability: make([]float64, cellCount),
			StopoverValue:   make([]float64, cellCount),
		},
	}
	ports.Diagnostics.PortSuitability[0] = 0.40
	ports.Diagnostics.StopoverValue[0] = 0.80

	stopovers, diagnostics := BuildMaritimeStopoverNodesWithDiagnostics(nil, cells, nil, ports, elevation, 0)
	if len(stopovers) != 1 {
		t.Fatalf("expected one stopover, got %+v", stopovers)
	}
	if stopovers[0].ComponentAreaEq <= 0 || stopovers[0].ComponentAreaEq >= 1 {
		t.Fatalf("expected high-resolution one-cell island to be tiny equivalent area, got %.3f", stopovers[0].ComponentAreaEq)
	}
	if diagnostics.CandidateTinyComponentEq != 1 || diagnostics.SelectedTinyComponentEq != 1 {
		t.Fatalf("expected tiny component diagnostics, got %+v", diagnostics)
	}
}

func TestMaritimeStopoverComponentScoreTaperPreservesExceptionalTinyIslands(t *testing.T) {
	settings := DefaultMaritimeStopoverSelectionSettings()
	tiny := maritimeStopoverScore(0.80, 0.40, 1.0, 0.0, 0.25, settings)
	large := maritimeStopoverScore(0.80, 0.40, 1.0, 0.0, 16.0, settings)
	weakTiny := maritimeStopoverScore(0.32, 0.22, 0.6, 0.2, 0.25, settings)

	if tiny <= settings.ScoreFloor {
		t.Fatalf("expected exceptional tiny island to remain viable, got %.3f floor %.3f", tiny, settings.ScoreFloor)
	}
	if tiny >= large {
		t.Fatalf("expected tiny component taper to score below large component: tiny %.3f large %.3f", tiny, large)
	}
	if weakTiny >= settings.ScoreFloor {
		t.Fatalf("expected weak tiny island to fall below floor after taper, got %.3f floor %.3f", weakTiny, settings.ScoreFloor)
	}
}
