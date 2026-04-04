package climgen

import "testing"

func TestDetermineSoilTypeSpecialCases(t *testing.T) {
	diag := &BiomeDiagnostics{
		IceAffinity:         []float64{0.7, 0, 0, 0, 0},
		WarmestSeasonTempC:  []float64{-2, 22, 18, 28, 16},
		AnnualMeanTempC:     []float64{-8, 14, 8, 24, 12},
		DesertAffinity:      []float64{0, 0.6, 0, 0, 0},
		GrasslandAffinity:   []float64{0, 0, 0.6, 0, 0},
		TropicalWetAffinity: []float64{0, 0, 0, 0.6, 0},
	}

	tests := []struct {
		name string
		idx  int
		args [8]float64
		want SoilType
	}{
		{"cryosol", 0, [8]float64{0.2, 0.5, 0.2, 0.1, 0, 0, 0.1, 0.2}, SoilCryosol},
		{"saline", 1, [8]float64{0.2, 0.2, 0.1, 0.1, 0.6, 0.2, 0.2, 0.1}, SoilSalineCoastal},
		{"peat", 2, [8]float64{0.7, 0.1, 0.3, 0.2, 0, 0.2, 0.7, 0.1}, SoilPeat},
		{"alluvial", 4, [8]float64{0.6, 0.4, 0.7, 0.2, 0, 0.8, 0.3, 0.1}, SoilAlluvial},
		{"tropical weathered", 3, [8]float64{0.8, 0.4, 0.4, 0.8, 0, 0.2, 0.2, 0.1}, SoilTropicalWeathered},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineSoilType(diag, tt.idx, tt.args[0], tt.args[1], tt.args[2], tt.args[3], tt.args[4], tt.args[5], tt.args[6], tt.args[7])
			if got != tt.want {
				t.Fatalf("got %s, want %s", SoilName(got), SoilName(tt.want))
			}
		})
	}
}

func TestComputeLocalRelief(t *testing.T) {
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1, 2}},
		{NeighborSiteIndices: []int32{0}},
		{NeighborSiteIndices: []int32{0}},
	}
	elevation := []float64{100, 160, 40}
	relief := computeLocalRelief(cells, elevation, 0)
	if relief[0] <= 0 {
		t.Fatalf("expected positive relief, got %.2f", relief[0])
	}
	if relief[1] != 60 {
		t.Fatalf("expected relief 60, got %.2f", relief[1])
	}
}
