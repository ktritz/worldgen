package climgen

import "testing"

func TestMaritimeEndpointPairWithinBudgetLowerBound(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0, Y: 1, Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0}},
	}
	meanNeighborDeg := maritimeMeanNeighborDegrees(sites, cells)
	if meanNeighborDeg < 89 || meanNeighborDeg > 91 {
		t.Fatalf("expected roughly 90 degree neighbor spacing, got %.2f", meanNeighborDeg)
	}
	if maritimeEndpointPairWithinBudgetLowerBound(sites, cells, 0, 1, 0.08, 0.18, meanNeighborDeg) {
		t.Fatalf("expected lower-bound prefilter to reject a pair below the minimum possible cost")
	}
	if !maritimeEndpointPairWithinBudgetLowerBound(sites, cells, 0, 1, 0.10, 0.18, meanNeighborDeg) {
		t.Fatalf("expected lower-bound prefilter to keep a pair within the conservative budget")
	}
}

func TestMaritimeEndpointPairWithinBudgetLowerBoundFallsBackWhenUncertain(t *testing.T) {
	if !maritimeEndpointPairWithinBudgetLowerBound(nil, nil, 0, 1, 0.01, 0.18, 0) {
		t.Fatalf("expected invalid geometry to keep pair rather than pruning uncertain work")
	}
}
