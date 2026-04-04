package climgen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResourceAbundanceSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "resource_abundance.json")
	data := `{
		"schemaVersion":"resource-abundance/v1",
		"ironAffinityMultiplier":1.0,
		"copperAffinityMultiplier":0.9,
		"leadSilverAffinityMultiplier":0.8,
		"goldAffinityMultiplier":0.7,
		"gemAffinityMultiplier":0.6,
		"placerAffinityMultiplier":1.0,
		"coalAffinityMultiplier":1.0,
		"oilGasAffinityMultiplier":1.0,
		"evaporiteAffinityMultiplier":1.0,
		"clayAffinityMultiplier":1.0,
		"stoneAffinityMultiplier":1.0,
		"ironPrimaryBias":0.0,
		"copperPrimaryBias":0.0,
		"leadSilverPrimaryBias":0.02,
		"goldPrimaryBias":0.04,
		"gemPrimaryBias":0.03,
		"placerPrimaryBias":0.0,
		"coalPrimaryBias":0.0,
		"oilGasPrimaryBias":0.0,
		"evaporitePrimaryBias":0.0,
		"clayPrimaryBias":0.0,
		"stonePrimaryBias":0.0
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	settings, err := LoadResourceAbundanceSettings(path)
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.GoldAffinityMultiplier != 0.7 {
		t.Fatalf("got gold multiplier %.2f, want 0.7", settings.GoldAffinityMultiplier)
	}
}

func TestResourceAbundanceMultipliersAffectPotentials(t *testing.T) {
	biomes := &BiomeResult{
		Diagnostics: &BiomeDiagnostics{
			AridityRatio:    []float64{0.95},
			IceAffinity:     []float64{0},
			AnnualPrecipCm:  []float64{65},
			ForestAffinity:  []float64{0.2},
			AnnualMeanTempC: []float64{11},
		},
	}
	soils := &SoilResult{
		Diagnostics: &SoilDiagnostics{
			Fertility: []float64{0.22},
			Drainage:  []float64{0.62},
			Alluvial:  []float64{0.03},
			Rockiness: []float64{0.92},
			Relief:    []float64{640},
			Salinity:  []float64{0.0},
		},
	}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{10},
		ChannelStrength: []float64{0.3},
		CellClass:       []string{"hillslope"},
	}
	geology := &ResourceGeologyInputs{
		HotspotStrength:    []float64{0.35},
		ContinentalHotspot: []float64{0.20},
	}
	elevation := []float64{2150}

	base := DefaultResourceAbundanceSettings()
	baseResult := ClassifyResourcesWithSettings(nil, biomes, soils, elevation, 0, hydro, geology, base)

	tuned := DefaultResourceAbundanceSettings()
	tuned.GemAffinityMultiplier = 2.0
	tuned.StoneAffinityMultiplier = 0.2
	tuned.GemPrimaryBias = 0.10
	tunedResult := ClassifyResourcesWithSettings(nil, biomes, soils, elevation, 0, hydro, geology, tuned)

	baseGem := baseResult.Diagnostics.GemAffinity[0]
	tunedGem := tunedResult.Diagnostics.GemAffinity[0]
	if tunedGem <= baseGem {
		t.Fatalf("expected tuned gem affinity %.3f to exceed base %.3f", tunedGem, baseGem)
	}
	if tunedResult.Types[0] != ResourceGemstones {
		t.Fatalf("expected tuned settings to promote gemstones, got %s", ResourceName(tunedResult.Types[0]))
	}
}

func TestResourceAbundanceSettingsCanSuppressGold(t *testing.T) {
	biomes := &BiomeResult{
		Diagnostics: &BiomeDiagnostics{
			AridityRatio:    []float64{0.90},
			IceAffinity:     []float64{0},
			AnnualPrecipCm:  []float64{60},
			ForestAffinity:  []float64{0.1},
			AnnualMeanTempC: []float64{12},
		},
	}
	soils := &SoilResult{
		Diagnostics: &SoilDiagnostics{
			Fertility: []float64{0.25},
			Drainage:  []float64{0.65},
			Alluvial:  []float64{0.02},
			Rockiness: []float64{0.84},
			Relief:    []float64{780},
			Salinity:  []float64{0.0},
		},
	}
	hydro := &HydrologyBiomeInputs{
		Runoff:          []float64{12},
		ChannelStrength: []float64{0.5},
		CellClass:       []string{"trunk"},
	}
	geology := &ResourceGeologyInputs{
		HotspotStrength:    []float64{0.85},
		ContinentalHotspot: []float64{0.65},
	}
	elevation := []float64{2300}

	base := DefaultResourceAbundanceSettings()
	baseResult := ClassifyResourcesWithSettings(nil, biomes, soils, elevation, 0, hydro, geology, base)

	tuned := DefaultResourceAbundanceSettings()
	tuned.GoldAffinityMultiplier = 0.0
	tuned.GoldPrimaryBias = 0.0
	tunedResult := ClassifyResourcesWithSettings(nil, biomes, soils, elevation, 0, hydro, geology, tuned)

	if baseResult.Diagnostics.GoldAffinity[0] <= 0 {
		t.Fatalf("expected base gold affinity to be positive")
	}
	if tunedResult.Diagnostics.GoldAffinity[0] != 0 {
		t.Fatalf("expected tuned gold affinity to be zero, got %.3f", tunedResult.Diagnostics.GoldAffinity[0])
	}
}
