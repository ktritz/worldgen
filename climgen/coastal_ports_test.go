package climgen

import "testing"

func TestBuildCoastalPortsFavorsEstuaryForShallowDraftVessel(t *testing.T) {
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0}},
	}
	elevation := []float64{10, -10}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{120, 0},
		ChannelStrength: []float64{2.6, 0},
	}
	soils := &SoilResult{
		Diagnostics: &SoilDiagnostics{
			Alluvial: []float64{0.9, 0},
		},
	}
	coastal := &CoastalResourceResult{
		Diagnostics: &CoastalResourceDiagnostics{
			EstuarineFishery: []float64{0.8, 0},
		},
	}
	maritime := MaritimeRouteSettings{
		DefaultVessel: "shallow-test",
		Vessels: []MaritimeVesselSettings{
			{
				Name:               "shallow-test",
				TechLevel:          "early",
				Propulsion:         "paddle",
				RouteClass:         "riverine",
				PayloadCapacity:    0.2,
				DailyRange:         0.4,
				LongHaulTolerance:  0.1,
				RiverCapability:    0.9,
				ShallowDraft:       1.0,
				BeachingCapability: 0.7,
				HarborDependence:   0.2,
				StopoverNeed:       0.6,
				StormTolerance:     0.3,
			},
		},
	}
	result := BuildCoastalPorts(
		cells,
		nil,
		nil,
		nil,
		nil,
		coastal,
		nil,
		soils,
		elevation,
		0.0,
		hydro,
		maritime,
		DefaultMaritimePortSettings(),
	)
	if result.Types[0] != CoastalPortEstuary {
		t.Fatalf("expected estuary port for shallow-draft estuarine cell, got %s", CoastalPortTypeName(result.Types[0]))
	}
}

func TestBuildCoastalPortsNodeScoreUsesNearbyCoastalCatchment(t *testing.T) {
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1}},
	}
	elevation := []float64{10, 10, -10}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{10, 120, 0},
		ChannelStrength: []float64{0.4, 2.6, 0},
	}
	soils := &SoilResult{
		Diagnostics: &SoilDiagnostics{
			Alluvial: []float64{0.1, 0.9, 0},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, Score: 0.5, Coastal: true, River: true},
		},
	}
	maritime := MaritimeRouteSettings{
		DefaultVessel: "shallow-test",
		Vessels: []MaritimeVesselSettings{
			{
				Name:               "shallow-test",
				TechLevel:          "early",
				Propulsion:         "paddle",
				RouteClass:         "riverine",
				PayloadCapacity:    0.2,
				DailyRange:         0.4,
				LongHaulTolerance:  0.1,
				RiverCapability:    0.9,
				ShallowDraft:       1.0,
				BeachingCapability: 0.7,
				HarborDependence:   0.2,
				StopoverNeed:       0.6,
				StormTolerance:     0.3,
			},
		},
	}

	result := BuildCoastalPorts(
		cells,
		nil,
		network,
		nil,
		nil,
		nil,
		nil,
		soils,
		elevation,
		0.0,
		hydro,
		maritime,
		DefaultMaritimePortSettings(),
	)
	if len(result.Diagnostics.NodePortScore) != 1 {
		t.Fatalf("expected one node score, got %d", len(result.Diagnostics.NodePortScore))
	}
	if result.Diagnostics.NodePortScore[0] <= result.Diagnostics.PortSuitability[0] {
		t.Fatalf("expected node port score %.2f to exceed direct cell suitability %.2f via nearby catchment", result.Diagnostics.NodePortScore[0], result.Diagnostics.PortSuitability[0])
	}
	if result.Diagnostics.NodeTerminalCell[0] != 1 {
		t.Fatalf("expected node terminal cell to resolve to nearby better coastal cell 1, got %d", result.Diagnostics.NodeTerminalCell[0])
	}
}

func TestPortSuitabilitySeparatesDeepDraftAndBeachableCraft(t *testing.T) {
	settings := DefaultMaritimePortSettings()
	deepDraft := MaritimeVesselSettings{
		Name:               "deep-draft-test",
		HarborDependence:   0.85,
		ShallowDraft:       0.15,
		BeachingCapability: 0.05,
		StormTolerance:     0.55,
	}
	beachable := MaritimeVesselSettings{
		Name:               "beachable-test",
		HarborDependence:   0.20,
		ShallowDraft:       0.92,
		BeachingCapability: 0.95,
		StormTolerance:     0.35,
		StopoverNeed:       0.85,
	}

	deepHarbor := derivePortSuitability(0.90, 0.35, 0.10, 0.10, 0.12, deepDraft, settings)
	deepBeach := derivePortSuitability(0.08, 0.05, 0.02, 0.70, 0.18, deepDraft, settings)
	if deepHarbor <= deepBeach {
		t.Fatalf("expected deep-draft craft to prefer harbor %.2f over beach %.2f", deepHarbor, deepBeach)
	}

	beachableHarbor := derivePortSuitability(0.90, 0.20, 0.05, 0.08, 0.12, beachable, settings)
	beachableBeach := derivePortSuitability(0.10, 0.05, 0.02, 0.75, 0.18, beachable, settings)
	if beachableBeach <= 0 || beachableBeach >= beachableHarbor {
		t.Fatalf("expected beachable craft to retain usable weaker beach score, beach=%.2f harbor=%.2f", beachableBeach, beachableHarbor)
	}
}

func TestDeepwaterSuitabilityFavorsBlueWaterCraft(t *testing.T) {
	settings := DefaultMaritimePortSettings()
	bluewater := MaritimeVesselSettings{
		Name:                "bluewater-test",
		HarborDependence:    0.75,
		ShallowDraft:        0.20,
		OpenOceanCapability: 0.85,
		LongHaulTolerance:   0.80,
		StormTolerance:      0.65,
	}
	littoral := MaritimeVesselSettings{
		Name:                "littoral-test",
		HarborDependence:    0.25,
		ShallowDraft:        0.90,
		OpenOceanCapability: 0.05,
		LongHaulTolerance:   0.18,
		StormTolerance:      0.25,
	}

	bluewaterScore := deriveDeepwaterSuitability(0.82, 0.62, 0.28, 0.22, 0.24, bluewater, settings)
	littoralScore := deriveDeepwaterSuitability(0.82, 0.62, 0.28, 0.22, 0.24, littoral, settings)
	if bluewaterScore <= littoralScore {
		t.Fatalf("expected bluewater craft to score higher at deepwater port, bluewater=%.2f littoral=%.2f", bluewaterScore, littoralScore)
	}
}
