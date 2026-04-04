package climgen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProfileCatalog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	data := `{
  "schemaVersion": "profile-catalog/v1",
  "ancestries": [
    {
      "name": "Test Folk",
      "settlement": {
        "Name": "Test Folk",
        "ClimateWeight": 1.0,
        "WaterWeight": 1.0,
        "TerrainWeight": 1.0,
        "SoilWeight": 1.0,
        "AccessWeight": 1.0,
        "ResourceWeight": 1.0,
        "HazardWeight": 1.0,
        "RiverBias": 0.0,
        "CoastalBias": 0.0,
        "AlluvialBias": 0.0,
        "FertilityBias": 0.0,
        "ForestBias": 0.0,
        "WetlandBias": 0.0,
        "RockBias": 0.0,
        "ElevationBias": 0.0,
        "ColdTolerance": 0.0,
        "AridityTolerance": 0.0,
        "FavorableThreshold": 0.42,
        "PrimeThreshold": 0.66
      },
      "social": {
        "openness": 0.5,
        "hierarchyPreference": 0.4,
        "traditionPreference": 0.6,
        "clanBias": 0.3,
        "guildBias": 0.7
      }
    }
  ],
  "cultures": [
    {
      "name": "Test Culture",
      "governance": {
        "centralizationPreference": 0.5,
        "legalismPreference": 0.5,
        "meritPreference": 0.5,
        "republicBias": 0.4,
        "autocracyBias": 0.2,
        "theocracyBias": 0.1
      }
    }
  ]
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	catalog, err := LoadProfileCatalog(path)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if len(catalog.Ancestries) != 1 || catalog.Ancestries[0].Name != "Test Folk" {
		t.Fatalf("unexpected ancestries: %+v", catalog.Ancestries)
	}
	if len(catalog.Cultures) != 1 || catalog.Cultures[0].Name != "Test Culture" {
		t.Fatalf("unexpected cultures: %+v", catalog.Cultures)
	}
	profiles := ExtractSettlementProfiles(catalog)
	if len(profiles) != 1 || profiles[0].Name != "Test Folk" {
		t.Fatalf("unexpected extracted settlement profiles: %+v", profiles)
	}
}

func TestDefaultFantasyProfileCatalogExtractsSettlementProfiles(t *testing.T) {
	catalog := DefaultFantasyProfileCatalog()
	if err := ValidateProfileCatalog(catalog); err != nil {
		t.Fatalf("validate default catalog: %v", err)
	}
	profiles := ExtractSettlementProfiles(catalog)
	if len(profiles) < 4 {
		t.Fatalf("expected default fantasy catalog to expose settlement profiles")
	}
}

func TestExtractComposedSettlementProfiles(t *testing.T) {
	catalog := &ProfileCatalog{
		SchemaVersion: ProfileCatalogSchemaVersion,
		Ancestries: []AncestryProfile{
			{
				Name: "Human",
				Settlement: &SettlementPreferenceProfile{
					Name:               "Human",
					ClimateWeight:      1.0,
					ResourceWeight:     1.0,
					FavorableThreshold: 0.42,
					PrimeThreshold:     0.66,
				},
			},
		},
		Cultures: []CultureProfile{
			{
				Name: "Merchant League",
				Settlement: &SettlementPreferenceOverrides{
					ResourceWeight: floatPtr(1.1),
				},
			},
		},
		Compositions: []ProfileCompositionSpec{
			{Name: "Human + Merchant League", Ancestry: "Human", Culture: "Merchant League"},
		},
	}
	if err := ValidateProfileCatalog(catalog); err != nil {
		t.Fatalf("validate catalog: %v", err)
	}
	profiles := ExtractComposedSettlementProfiles(catalog)
	if len(profiles) != 1 {
		t.Fatalf("expected one composed profile, got %d", len(profiles))
	}
	if profiles[0].Name != "Human + Merchant League" || profiles[0].ResourceWeight != 1.1 {
		t.Fatalf("unexpected composed profile: %+v", profiles[0])
	}
}

func TestComposeResolvedProfileAppliesCultureOverrides(t *testing.T) {
	ancestry := AncestryProfile{
		Name: "Human",
		Settlement: &SettlementPreferenceProfile{
			Name:               "Human",
			ClimateWeight:      1.0,
			ResourceWeight:     1.0,
			FavorableThreshold: 0.42,
			PrimeThreshold:     0.66,
		},
		Social: &ProfileSocialModule{
			Openness: 0.5,
			GuildBias: 0.6,
		},
		Economy: &ProfileEconomicModule{
			TradeBias: 0.6,
			ProfessionAptitudes: map[string]float64{
				"merchant": 0.6,
			},
		},
	}
	resourceWeight := 1.2
	guildBias := 0.8
	culture := &CultureProfile{
		Name: "Merchant League",
		Settlement: &SettlementPreferenceOverrides{
			ResourceWeight: &resourceWeight,
		},
		Social: &ProfileSocialOverrides{
			GuildBias: &guildBias,
		},
		Economy: &ProfileEconomicOverrides{
			ProfessionAptitudes: map[string]float64{
				"merchant": 0.9,
				"scribe":   0.7,
			},
		},
	}
	resolved := ComposeResolvedProfile(ancestry, culture)
	if resolved.Settlement == nil || resolved.Settlement.ResourceWeight != 1.2 {
		t.Fatalf("expected culture override on settlement resource weight, got %+v", resolved.Settlement)
	}
	if resolved.Settlement.ClimateWeight != 1.0 {
		t.Fatalf("expected ancestry settlement values to survive when not overridden")
	}
	if resolved.Social == nil || resolved.Social.GuildBias != 0.8 || resolved.Social.Openness != 0.5 {
		t.Fatalf("unexpected social composition: %+v", resolved.Social)
	}
	if resolved.Economy == nil || resolved.Economy.ProfessionAptitudes["merchant"] != 0.9 || resolved.Economy.ProfessionAptitudes["scribe"] != 0.7 {
		t.Fatalf("unexpected economy composition: %+v", resolved.Economy)
	}
}

func floatPtr(v float64) *float64 { return &v }
