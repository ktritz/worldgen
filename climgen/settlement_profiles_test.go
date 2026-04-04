package climgen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettlementProfilesShiftPreference(t *testing.T) {
	base := &SettlementResult{
		Diagnostics: &SettlementDiagnostics{
			ClimateScore:  []float64{0.55},
			WaterScore:    []float64{0.45},
			TerrainScore:  []float64{0.75},
			SoilScore:     []float64{0.35},
			AccessScore:   []float64{0.25},
			ResourceScore: []float64{0.85},
			HazardPenalty: []float64{0.20},
			RiverBonus:    []float64{0.20},
			CoastalBonus:  []float64{0.05},
		},
	}
	biomes := &BiomeResult{
		Diagnostics: &BiomeDiagnostics{
			AnnualIceFraction: []float64{0.10},
			AridityRatio:      []float64{0.90},
		},
	}
	soils := &SoilResult{
		Diagnostics: &SoilDiagnostics{
			Alluvial:  []float64{0.10},
			Fertility: []float64{0.35},
			Rockiness: []float64{0.85},
		},
	}
	vegetation := &VegetationResult{
		Types: []VegetationType{VegetationShrubland},
		Diagnostics: &VegetationDiagnostics{
			TreeCover:            []float64{0.10},
			MoistureAvailability: []float64{0.20},
			WetlandCover:         []float64{0.05},
		},
	}
	elevation := []float64{1800}

	profiles := DefaultFantasySettlementProfiles()
	var human, dwarf SettlementPreferenceProfile
	for _, p := range profiles {
		switch p.Name {
		case "Human":
			human = p
		case "Dwarf":
			dwarf = p
		}
	}

	humanResult := ClassifySettlementPreference(base, biomes, soils, vegetation, elevation, 0, human)
	dwarfResult := ClassifySettlementPreference(base, biomes, soils, vegetation, elevation, 0, dwarf)
	if dwarfResult.Suitability[0] <= humanResult.Suitability[0] {
		t.Fatalf("dwarf profile should prefer rugged resource-rich terrain more than human profile")
	}
}

func TestSettlementProfilesFavorForestForElves(t *testing.T) {
	base := &SettlementResult{
		Diagnostics: &SettlementDiagnostics{
			ClimateScore:  []float64{0.72},
			WaterScore:    []float64{0.55},
			TerrainScore:  []float64{0.55},
			SoilScore:     []float64{0.60},
			AccessScore:   []float64{0.35},
			ResourceScore: []float64{0.20},
			HazardPenalty: []float64{0.08},
			RiverBonus:    []float64{0.25},
			CoastalBonus:  []float64{0.10},
		},
	}
	biomes := &BiomeResult{
		Diagnostics: &BiomeDiagnostics{
			AnnualIceFraction: []float64{0.00},
			AridityRatio:      []float64{1.25},
		},
	}
	soils := &SoilResult{
		Diagnostics: &SoilDiagnostics{
			Alluvial:  []float64{0.15},
			Fertility: []float64{0.65},
			Rockiness: []float64{0.15},
		},
	}
	vegetation := &VegetationResult{
		Types: []VegetationType{VegetationForest},
		Diagnostics: &VegetationDiagnostics{
			TreeCover:            []float64{0.85},
			MoistureAvailability: []float64{0.75},
			WetlandCover:         []float64{0.05},
		},
	}
	elevation := []float64{500}

	profiles := DefaultFantasySettlementProfiles()
	var elf, dwarf SettlementPreferenceProfile
	for _, p := range profiles {
		switch p.Name {
		case "Elf":
			elf = p
		case "Dwarf":
			dwarf = p
		}
	}

	elfResult := ClassifySettlementPreference(base, biomes, soils, vegetation, elevation, 0, elf)
	dwarfResult := ClassifySettlementPreference(base, biomes, soils, vegetation, elevation, 0, dwarf)
	if elfResult.Suitability[0] <= dwarfResult.Suitability[0] {
		t.Fatalf("elf profile should prefer forested terrain more than dwarf profile")
	}
}

func TestLoadSettlementPreferenceProfiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	data := `{
  "profiles": [
    {
      "Name": "Test Folk",
      "ClimateWeight": 1.0,
      "WaterWeight": 1.1,
      "TerrainWeight": 0.9,
      "SoilWeight": 1.0,
      "AccessWeight": 1.0,
      "ResourceWeight": 0.8,
      "HazardWeight": 1.0,
      "RiverBias": 0.03,
      "CoastalBias": 0.01,
      "AlluvialBias": 0.02,
      "FertilityBias": 0.04,
      "ForestBias": 0.05,
      "WetlandBias": -0.02,
      "RockBias": -0.01,
      "ElevationBias": 0.00,
      "ColdTolerance": 0.01,
      "AridityTolerance": -0.01,
      "FavorableThreshold": 0.40,
      "PrimeThreshold": 0.62
    }
  ]
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write profile file: %v", err)
	}
	profiles, err := LoadSettlementPreferenceProfiles(path)
	if err != nil {
		t.Fatalf("load profiles: %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != "Test Folk" || profiles[0].PrimeThreshold != 0.62 {
		t.Fatalf("unexpected profiles: %+v", profiles)
	}
}
