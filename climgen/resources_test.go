package climgen

import "testing"

func TestDetermineResourceType(t *testing.T) {
	tests := []struct {
		name string
		args []resourceCandidate
		want ResourceType
	}{
		{"none below threshold", []resourceCandidate{{ResourceIronOre, 0.2}, {ResourceClayAggregate, 0.3}}, ResourceNone},
		{"iron wins", []resourceCandidate{{ResourceIronOre, 0.8}, {ResourceCopperOre, 0.3}, {ResourceIndustrialStone, 0.4}}, ResourceIronOre},
		{"copper wins", []resourceCandidate{{ResourceIronOre, 0.2}, {ResourceCopperOre, 0.7}, {ResourceCoal, 0.3}}, ResourceCopperOre},
		{"lead silver wins", []resourceCandidate{{ResourceCopperOre, 0.4}, {ResourceLeadSilverOre, 0.76}, {ResourceGoldOre, 0.58}}, ResourceLeadSilverOre},
		{"gold wins", []resourceCandidate{{ResourceLeadSilverOre, 0.5}, {ResourceGoldOre, 0.77}, {ResourceGemstones, 0.45}}, ResourceGoldOre},
		{"gems win", []resourceCandidate{{ResourceGoldOre, 0.52}, {ResourceGemstones, 0.81}, {ResourceIndustrialStone, 0.3}}, ResourceGemstones},
		{"placer wins", []resourceCandidate{{ResourceIronOre, 0.2}, {ResourcePlacerAlluvial, 0.75}, {ResourceCoal, 0.2}}, ResourcePlacerAlluvial},
		{"evaporite wins", []resourceCandidate{{ResourceEvaporite, 0.8}, {ResourceClayAggregate, 0.4}, {ResourceIndustrialStone, 0.2}}, ResourceEvaporite},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineResourceType(tt.args)
			if got != tt.want {
				t.Fatalf("got %s, want %s", ResourceName(got), ResourceName(tt.want))
			}
		})
	}
}

func TestDefaultResourceAbundanceSettingsValidate(t *testing.T) {
	if err := ValidateResourceAbundanceSettings(DefaultResourceAbundanceSettings()); err != nil {
		t.Fatalf("default settings should validate: %v", err)
	}
}

func TestClassifyResourcesProducesDistinctMetalAndGemBelts(t *testing.T) {
	biomes := &BiomeResult{
		Diagnostics: &BiomeDiagnostics{
			AridityRatio:    []float64{0.95, 0.85, 1.10},
			IceAffinity:     []float64{0, 0, 0},
			AnnualPrecipCm:  []float64{70, 55, 65},
			ForestAffinity:  []float64{0.2, 0.1, 0.15},
			AnnualMeanTempC: []float64{12, 10, 11},
		},
	}
	soils := &SoilResult{
		Diagnostics: &SoilDiagnostics{
			Fertility: []float64{0.30, 0.20, 0.18},
			Drainage:  []float64{0.55, 0.60, 0.70},
			Alluvial:  []float64{0.10, 0.05, 0.02},
			Rockiness: []float64{0.65, 0.85, 0.95},
			Relief:    []float64{450, 700, 620},
			Salinity:  []float64{0.00, 0.00, 0.00},
		},
	}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{18, 14, 8},
		ChannelStrength: []float64{0.5, 0.7, 0.2},
		CellClass:       []string{"trunk", "hillslope", "mountain"},
	}
	geology := &ResourceGeologyInputs{
		HotspotStrength:    []float64{0.10, 0.90, 0.25},
		ContinentalHotspot: []float64{0.05, 0.75, 0.05},
	}
	got := ClassifyResources(nil, biomes, soils, []float64{1400, 1900, 2150}, 0, hydro, geology)
	if got.Types[1] != ResourceCopperOre && got.Types[1] != ResourceGoldOre && got.Types[1] != ResourceLeadSilverOre {
		t.Fatalf("volcanic-orogenic highland should classify as hard-rock metal belt, got %s", ResourceName(got.Types[1]))
	}
	if got.Types[2] != ResourceGemstones && got.Types[2] != ResourceIndustrialStone {
		t.Fatalf("very rocky high-relief upland should classify as gem/stone terrain, got %s", ResourceName(got.Types[2]))
	}
}

func TestClassifyResourcesSedimentaryNeedsDepositionalContext(t *testing.T) {
	biomes := &BiomeResult{
		Diagnostics: &BiomeDiagnostics{
			AridityRatio:    []float64{0.95, 0.95},
			IceAffinity:     []float64{0, 0},
			AnnualPrecipCm:  []float64{60, 90},
			ForestAffinity:  []float64{0.2, 0.5},
			AnnualMeanTempC: []float64{12, 14},
		},
	}
	soils := &SoilResult{
		Diagnostics: &SoilDiagnostics{
			Fertility: []float64{0.35, 0.65},
			Drainage:  []float64{0.75, 0.45},
			Alluvial:  []float64{0.05, 0.75},
			Rockiness: []float64{0.20, 0.10},
			Relief:    []float64{80, 60},
			Salinity:  []float64{0.00, 0.05},
		},
	}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{12, 45},
		ChannelStrength: []float64{0.2, 1.3},
		CellClass:       []string{"hillslope", "endorheic_basin"},
	}
	geology := &ResourceGeologyInputs{
		HotspotStrength:    []float64{0, 0},
		ContinentalHotspot: []float64{0, 0},
	}
	got := ClassifyResources(nil, biomes, soils, []float64{220, 180}, 0, hydro, geology)
	if got.Types[0] == ResourceCoal || got.Types[0] == ResourceOilGas {
		t.Fatalf("generic low-relief hillslope should not classify as oil/gas or coal basin stand-in")
	}
	if got.Types[1] != ResourceCoal && got.Types[1] != ResourceOilGas && got.Types[1] != ResourceClayAggregate {
		t.Fatalf("endorheic alluvial lowland should classify as depositional resource, got %s", ResourceName(got.Types[1]))
	}
}
