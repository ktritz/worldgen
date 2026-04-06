package climgen

import "testing"

func TestIdentifyMajorRiverPortsPrefersHigherTierAnchors(t *testing.T) {
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, Kind: SettlementNodeHamlet, Score: 0.44, River: true},
			{ID: 1, Kind: SettlementNodeVillage, Score: 0.56, River: true},
			{ID: 2, Kind: SettlementNodeTown, Score: 0.64, River: true},
		},
	}
	diag := &RiverTradeDiagnostics{
		NodeCentrality: []float64{0.16, 0.34, 0.42},
	}
	ports := identifyMajorRiverPorts(
		network,
		map[int]struct{}{0: {}, 1: {}, 2: {}},
		diag,
		DefaultRiverTradeSettings(),
	)
	if len(ports) == 0 {
		t.Fatalf("expected at least one major river port")
	}
	for _, nodeIdx := range ports {
		if nodeIdx == 0 {
			t.Fatalf("expected local anchor with moderate centrality to be filtered out of major ports")
		}
	}
}
