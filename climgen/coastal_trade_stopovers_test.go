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
