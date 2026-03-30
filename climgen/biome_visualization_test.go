package climgen

import "testing"

func TestTopTwoAffinityFamilies(t *testing.T) {
	diag := &BiomeDiagnostics{
		DesertAffinity:      []float64{0.20},
		GrasslandAffinity:   []float64{0.45},
		ForestAffinity:      []float64{0.30},
		TropicalWetAffinity: []float64{0.15},
		IceAffinity:         []float64{0.75},
		TundraAffinity:      []float64{0.20},
		BorealAffinity:      []float64{0.10},
		WetlandAffinity:     []float64{0.05},
		AlpineAffinity:      []float64{0.10},
	}

	firstFamily, firstValue, secondFamily, secondValue := topTwoAffinityFamilies(diag, 0)
	if firstFamily != affinityIce || firstValue != 0.75 {
		t.Fatalf("first affinity = (%v, %.2f), want (%v, 0.75)", firstFamily, firstValue, affinityIce)
	}
	if secondFamily != affinityGrassland || secondValue != 0.45 {
		t.Fatalf("second affinity = (%v, %.2f), want (%v, 0.45)", secondFamily, secondValue, affinityGrassland)
	}
}
