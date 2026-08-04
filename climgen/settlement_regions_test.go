package climgen

import "testing"

func TestAssignSettlementRegionsUsesPhysicalAnchorReach(t *testing.T) {
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1}},
	}
	result := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, Score: 0.52},
			{ID: 1, CellIndex: 2, Kind: SettlementNodeVillage, Score: 0.51},
		},
		Diagnostics: &SettlementNetworkDiagnostics{
			MovementCost: []float64{1, 1, 1},
		},
	}
	settings := DefaultSettlementNetworkSettings()

	assignSettlementRegions(result, cells, settings)

	if len(result.Regions) != 1 {
		t.Fatalf("expected physically reachable village anchors to share one region, got %d", len(result.Regions))
	}
	if result.Diagnostics.RegionFormation.PhysicalClusterLinks != 1 {
		t.Fatalf("expected one physical cluster link, got %d", result.Diagnostics.RegionFormation.PhysicalClusterLinks)
	}
}

func TestAssignSettlementRegionsCanPhysicallyMergeSeparateTransportComponents(t *testing.T) {
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2, 4}},
		{SiteIndex: 4, NeighborSiteIndices: []int32{3}},
	}
	result := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, Score: 0.52},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeVillage, Score: 0.51},
			{ID: 2, CellIndex: 3, Kind: SettlementNodeVillage, Score: 0.52},
			{ID: 3, CellIndex: 4, Kind: SettlementNodeVillage, Score: 0.51},
		},
		Links: []SettlementLink{
			{From: 0, To: 1, TravelCost: 1},
			{From: 2, To: 3, TravelCost: 1},
		},
		Diagnostics: &SettlementNetworkDiagnostics{
			MovementCost: []float64{1, 1, 1, 1, 1},
		},
	}

	assignSettlementRegions(result, cells, DefaultSettlementNetworkSettings())

	if len(result.Regions) != 1 {
		t.Fatalf("expected physical fallback to merge physically reachable transport components, got %d", len(result.Regions))
	}
	if result.Diagnostics.RegionFormation.PhysicalClusterLinks == 0 {
		t.Fatalf("expected physical fallback link between separate transport components")
	}
	if result.Diagnostics.RegionFormation.PhysicalSkippedCrossComponentPairs == 0 {
		t.Fatalf("expected cross-component physical reach to be diagnosed")
	}
}

func TestAssignSettlementRegionsSkipsPhysicalLinksWithinTransportComponent(t *testing.T) {
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1}},
	}
	result := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, Score: 0.52},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeVillage, Score: 0.51},
			{ID: 2, CellIndex: 2, Kind: SettlementNodeVillage, Score: 0.50},
		},
		Links: []SettlementLink{
			{From: 0, To: 1, TravelCost: 1},
			{From: 1, To: 2, TravelCost: 1},
		},
		Diagnostics: &SettlementNetworkDiagnostics{
			MovementCost: []float64{1, 1, 1},
		},
	}

	assignSettlementRegions(result, cells, DefaultSettlementNetworkSettings())

	if len(result.Regions) != 1 {
		t.Fatalf("expected existing transport component to remain one region, got %d", len(result.Regions))
	}
	if result.Diagnostics.RegionFormation.PhysicalClusterLinks != 0 {
		t.Fatalf("expected no redundant physical links within transport component, got %d", result.Diagnostics.RegionFormation.PhysicalClusterLinks)
	}
	if result.Diagnostics.RegionFormation.PhysicalSkippedSameComponentPairs == 0 {
		t.Fatalf("expected same-component physical reach to be skipped")
	}
}

func TestSettlementRegionCenterPrefersEffectiveRankOverLocalPeakScore(t *testing.T) {
	result := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeVillage, Score: 0.70, CarryingCapacity: 0.70, UrbanPotential: 0.70, PhysicalSupportArea: 1.25, Coastal: true, River: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, Score: 0.50, CarryingCapacity: 0.55, UrbanPotential: 0.55, PhysicalSupportArea: 0.75},
		},
	}
	center := selectSettlementRegionCenter([]int{0, 1}, result, [][]int{{1, 2, 3}, {0}})
	if center != 1 {
		t.Fatalf("expected supported regional anchor to remain region center, got node %d", center)
	}
}
