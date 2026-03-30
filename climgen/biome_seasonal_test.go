package climgen

import (
	"math"
	"testing"
)

func TestClassifySeasonalCellRegimes(t *testing.T) {
	tests := []struct {
		name   string
		biome  Biome
		elev   float64
		annT   float64
		warmT  float64
		coldT  float64
		cont   float64
		annP   float64
		dryP   float64
		dryR   float64
		warmPr float64
		aridR  float64
	}{
		{
			name:  "tropical rainforest",
			biome: BiomeTropicalRainforest,
			annT:  26, warmT: 29, coldT: 23, cont: 6,
			annP: 240, dryP: 9, dryR: 0.50, warmPr: 0.55, aridR: 4.0,
		},
		{
			name:  "savanna",
			biome: BiomeSavanna,
			annT:  25, warmT: 31, coldT: 20, cont: 11,
			annP: 95, dryP: 1.5, dryR: 0.08, warmPr: 0.82, aridR: 1.6,
		},
		{
			name:  "mediterranean",
			biome: BiomeMediterranean,
			annT:  17, warmT: 26, coldT: 8, cont: 18,
			annP: 70, dryP: 2.5, dryR: 0.20, warmPr: 0.22, aridR: 1.4,
		},
		{
			name:  "cold desert",
			biome: BiomeDesertCold,
			annT:  6, warmT: 19, coldT: -9, cont: 28,
			annP: 15, dryP: 1, dryR: 0.10, warmPr: 0.48, aridR: 0.35,
		},
		{
			name:  "boreal forest",
			biome: BiomeBorealForest,
			annT:  2, warmT: 16, coldT: -13, cont: 29,
			annP: 70, dryP: 5, dryR: 0.40, warmPr: 0.55, aridR: 2.1,
		},
		{
			name:  "alpine",
			biome: BiomeAlpine,
			elev:  3600,
			annT:  0, warmT: 8, coldT: -7, cont: 15,
			annP: 90, dryP: 5, dryR: 0.35, warmPr: 0.55, aridR: 2.0,
		},
		{
			name:  "wetland",
			biome: BiomeWetland,
			annT:  18, warmT: 25, coldT: 10, cont: 15,
			annP: 130, dryP: 9, dryR: 0.55, warmPr: 0.55, aridR: 2.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wetlandAffinity := 0.0
			if tt.biome == BiomeWetland {
				wetlandAffinity = 0.8
			}
			got := classifySeasonalCell(
				tt.elev, 0,
				tt.annT, tt.warmT, tt.coldT, tt.cont,
				tt.annP, tt.dryP, tt.dryR, tt.warmPr, tt.aridR, 0, 0, wetlandAffinity,
			)
			if got != tt.biome {
				t.Fatalf("got %s, want %s", BiomeName(got), BiomeName(tt.biome))
			}
		})
	}
}

func TestSummarizeBiomeClimateWarmSeasonShare(t *testing.T) {
	result := &SeasonalClimateResult{
		AnnualMean: &TemperatureResult{
			TemperatureCelsius: []float64{20},
		},
		AnnualPrecipitation: []float64{100},
		Snapshots: []SeasonalClimateSnapshot{
			{TemperatureCelsius: []float64{18}, Precipitation: []float64{10}},
			{TemperatureCelsius: []float64{24}, Precipitation: []float64{40}},
			{TemperatureCelsius: []float64{28}, Precipitation: []float64{30}},
			{TemperatureCelsius: []float64{16}, Precipitation: []float64{20}},
		},
	}

	diag := SummarizeBiomeClimate(result)
	if got, want := diag.WarmSeasonPrecipShare[0], 0.70; math.Abs(got-want) > 1e-9 {
		t.Fatalf("warm season share = %.2f, want %.2f", got, want)
	}
	if got, want := diag.DrySeasonRatio[0], 0.25; math.Abs(got-want) > 1e-9 {
		t.Fatalf("dry season ratio = %.2f, want %.2f", got, want)
	}
}

func TestComputeBiomeAffinities(t *testing.T) {
	diag := &BiomeDiagnostics{
		AnnualMeanTempC:          []float64{28, 18, 6},
		WarmestSeasonTempC:       []float64{31, 26, 11},
		ColdestSeasonTempC:       []float64{24, 10, -2},
		AnnualIceFraction:        []float64{0, 0, 0.1},
		WarmestSeasonIceFraction: []float64{0, 0, 0},
		AnnualPrecipCm:           []float64{20, 140, 120},
		DriestSeasonPrecipCm:     []float64{1, 12, 8},
		AridityRatio:             []float64{0.3, 2.4, 1.6},
		DesertAffinity:           make([]float64, 3),
		GrasslandAffinity:        make([]float64, 3),
		ForestAffinity:           make([]float64, 3),
		TropicalWetAffinity:      make([]float64, 3),
		ColdAffinity:             make([]float64, 3),
		IceAffinity:              make([]float64, 3),
		TundraAffinity:           make([]float64, 3),
		BorealAffinity:           make([]float64, 3),
		WetlandAffinity:          make([]float64, 3),
		AlpineAffinity:           make([]float64, 3),
	}
	elevation := []float64{200, 500, 3400}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{5, 80, 90},
		ChannelStrength: []float64{0.1, 2.2, 0.6},
		CellClass:       []string{"hillslope", "floodplain", "hillslope"},
	}
	computeBiomeAffinities(diag, elevation, 0, hydro)

	if diag.DesertAffinity[0] <= 0.7 {
		t.Fatalf("expected hot dry cell to have strong desert affinity, got %.2f", diag.DesertAffinity[0])
	}
	if diag.WetlandAffinity[1] <= 0.5 {
		t.Fatalf("expected floodplain-like cell to have strong wetland affinity, got %.2f", diag.WetlandAffinity[1])
	}
	if diag.AlpineAffinity[2] <= 0.5 {
		t.Fatalf("expected high mountain cell to have alpine affinity, got %.2f", diag.AlpineAffinity[2])
	}
	if diag.ForestAffinity[2] <= 0.3 {
		t.Fatalf("expected moist mountain cell to retain forest affinity, got %.2f", diag.ForestAffinity[2])
	}
	if diag.ColdAffinity[2] <= 0.1 {
		t.Fatalf("expected cool mountain cell to keep some cold affinity, got %.2f", diag.ColdAffinity[2])
	}
}
