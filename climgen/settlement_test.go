package climgen

import "testing"

func TestClassifySettlementClass(t *testing.T) {
	tests := []struct {
		score float64
		want  SettlementClass
	}{
		{0.10, SettlementUnsuitable},
		{0.30, SettlementMarginal},
		{0.55, SettlementFavorable},
		{0.82, SettlementPrime},
	}
	for _, tt := range tests {
		if got := classifySettlementClass(tt.score); got != tt.want {
			t.Fatalf("score %.2f => %s, want %s", tt.score, SettlementClassName(got), SettlementClassName(tt.want))
		}
	}
}

func TestSettlementResourceScoreRewardsUsefulClasses(t *testing.T) {
	result := &ResourceResult{
		Types: []ResourceType{ResourceNone, ResourcePlacerAlluvial, ResourceIndustrialStone},
		Diagnostics: &ResourceDiagnostics{
			PlacerAffinity:    []float64{0.0, 0.8, 0.1},
			StoneAffinity:     []float64{0.0, 0.2, 0.7},
			IronAffinity:      []float64{0.0, 0.1, 0.0},
			CopperAffinity:    []float64{0.0, 0.0, 0.0},
			CoalAffinity:      []float64{0.0, 0.2, 0.1},
			OilGasAffinity:    []float64{0.0, 0.0, 0.0},
			EvaporiteAffinity: []float64{0.0, 0.0, 0.0},
		},
	}
	if settlementResourceScore(result, 1) <= settlementResourceScore(result, 0) {
		t.Fatalf("placer resource should improve settlement resource score")
	}
	if settlementResourceScore(result, 2) <= settlementResourceScore(result, 0) {
		t.Fatalf("industrial stone should improve settlement resource score")
	}
}
