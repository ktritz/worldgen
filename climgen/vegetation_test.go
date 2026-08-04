package climgen

import "testing"

func TestDetermineVegetationTypeSpecials(t *testing.T) {
	tests := []struct {
		name  string
		want  VegetationType
		args  []float64
		biome Biome
	}{
		{
			name:  "mangrove wins",
			want:  VegetationMangrove,
			biome: BiomeWetland,
			args:  []float64{0.2, 0.1, 0.1, 0.8, 0.1, 0.8, 0.1, 0.1, 0.1, 0.1, 0.1, 0.4, 0.0, 0.1},
		},
		{
			name:  "salt marsh wins",
			want:  VegetationSaltMarsh,
			biome: BiomeWetland,
			args:  []float64{0.2, 0.1, 0.1, 0.8, 0.1, 0.1, 0.8, 0.1, 0.1, 0.1, 0.1, 0.2, 0.0, 0.1},
		},
		{
			name:  "cloud forest wins",
			want:  VegetationCloudForest,
			biome: BiomeTemperateRainforest,
			args:  []float64{0.7, 0.2, 0.1, 0.2, 0.1, 0.1, 0.1, 0.1, 0.1, 0.75, 0.1, 0.3, 0.0, 0.1},
		},
		{
			name:  "rainforest from broad cover",
			want:  VegetationRainforest,
			biome: BiomeTropicalRainforest,
			args:  []float64{0.85, 0.2, 0.1, 0.15, 0.05, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.7, 0.0, 0.1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineVegetationType(
				tt.biome,
				tt.args[0], tt.args[1], tt.args[2], tt.args[3], tt.args[4],
				tt.args[5], tt.args[6], tt.args[7], tt.args[8], tt.args[9], tt.args[10],
				tt.args[11], tt.args[12], tt.args[13],
			)
			if got != tt.want {
				t.Fatalf("got %s, want %s", VegetationName(got), VegetationName(tt.want))
			}
		})
	}
}

func TestComputeCoastalExposure(t *testing.T) {
	cells := []VoronoiCell{
		{NeighborSiteIndices: []int32{1, 2}},
		{NeighborSiteIndices: []int32{0}},
		{NeighborSiteIndices: []int32{0}},
	}
	elevation := []float64{100, -10, 120}
	exposure := ComputeCoastalExposure(cells, elevation, 0)
	if exposure[0] <= 0.3 {
		t.Fatalf("expected coastal cell exposure > 0.3, got %.2f", exposure[0])
	}
	if exposure[2] >= exposure[0] {
		t.Fatalf("expected inland-support cell exposure %.2f to stay below direct coast %.2f", exposure[2], exposure[0])
	}
}

func TestComputeCoastalExposureUsesPhysicalRadius(t *testing.T) {
	cells := makeLineCells(40962, 4)
	elevation := make([]float64, len(cells))
	for i := range elevation {
		elevation[i] = 100
	}
	elevation[0] = -10

	exposure := ComputeCoastalExposure(cells, elevation, 0)
	if exposure[2] <= 0 {
		t.Fatalf("expected refined two-step land cell to retain coastal exposure")
	}
	if exposure[3] >= exposure[2] {
		t.Fatalf("expected coastal exposure to decay inland, got cell2=%.3f cell3=%.3f", exposure[2], exposure[3])
	}
}
