package climgen

import "testing"

func TestClassifyLocalTradeNodeKindPrefersWaystationForModerateCrossings(t *testing.T) {
	diag := &LandRouteDiagnostics{
		CrossingPressure: []float64{0.22, 0.36, 0.08},
		BridgeProxy:      []float64{0.04, 0.10, 0.00},
		FordProxy:        []float64{0.05, 0.12, 0.00},
		RoadQuality:      []float64{0.24, 0.28, 0.18},
	}

	if got := classifyLocalTradeNodeKind(diag, 0, 0.34, 0.33); got != LocalTradeNodeWaystation {
		t.Fatalf("expected moderate crossing cell to classify as waystation, got %s", LocalTradeNodeKindName(got))
	}
	if got := classifyLocalTradeNodeKind(diag, 1, 0.32, 0.29); got != LocalTradeNodeCrossingDepot {
		t.Fatalf("expected strong supported crossing cell to classify as crossing depot, got %s", LocalTradeNodeKindName(got))
	}
	if got := classifyLocalTradeNodeKind(diag, 2, 0.18, 0.16); got != LocalTradeNodeCollectionPoint {
		t.Fatalf("expected weak logistics cell to classify as collection point, got %s", LocalTradeNodeKindName(got))
	}
}

