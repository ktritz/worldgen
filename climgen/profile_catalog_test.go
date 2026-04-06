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
      "baselinePrevalence": 0.75,
      "traits": {
        "order": 0.6
      },
      "affinities": [
        {"targetType": "tag", "target": "forest", "weight": 0.2}
      ],
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
      "traits": {
        "mercantile": 0.8
      },
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
	if catalog.Ancestries[0].BaselinePrevalence != 0.75 {
		t.Fatalf("expected baseline prevalence to load, got %.2f", catalog.Ancestries[0].BaselinePrevalence)
	}
	if len(catalog.Cultures) != 1 || catalog.Cultures[0].Name != "Test Culture" {
		t.Fatalf("unexpected cultures: %+v", catalog.Cultures)
	}
	if catalog.Ancestries[0].Traits["order"] != 0.6 {
		t.Fatalf("expected ancestry trait to load")
	}
	profiles := ExtractSettlementProfiles(catalog)
	if len(profiles) != 1 || profiles[0].Name != "Test Folk" {
		t.Fatalf("unexpected extracted settlement profiles: %+v", profiles)
	}
}

func TestValidateProfileCatalogRejectsNegativeBaselinePrevalence(t *testing.T) {
	catalog := &ProfileCatalog{
		SchemaVersion: ProfileCatalogSchemaVersion,
		Ancestries: []AncestryProfile{
			{Name: "Bad Folk", BaselinePrevalence: -0.1},
		},
	}
	if err := ValidateProfileCatalog(catalog); err == nil {
		t.Fatalf("expected negative baselinePrevalence to fail validation")
	}
}

func TestLoadProfileCatalogPack(t *testing.T) {
	root := t.TempDir()
	write := func(rel, data string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	write("config/catalog.json", `{
  "schemaVersion": "profile-catalog-pack/v1",
  "ancestryFiles": ["profiles/ancestries/test_folk.json"],
  "cultureFiles": ["profiles/cultures/test_culture.json"],
  "compositionFiles": ["profiles/compositions/core.json"]
}`)
	write("config/profiles/ancestries/test_folk.json", `{
  "name": "Test Folk",
  "traits": {"order": 0.6},
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
  }
}`)
	write("config/profiles/cultures/test_culture.json", `{
  "name": "Test Culture",
  "traits": {"mercantile": 0.8}
}`)
	write("config/profiles/compositions/core.json", `{
  "compositions": [
    {"name": "Test Folk + Test Culture", "ancestry": "Test Folk", "culture": "Test Culture"}
  ]
}`)
	catalog, err := LoadProfileCatalog(filepath.Join(root, "config/catalog.json"))
	if err != nil {
		t.Fatalf("load pack catalog: %v", err)
	}
	if len(catalog.Ancestries) != 1 || catalog.Ancestries[0].Name != "Test Folk" {
		t.Fatalf("unexpected pack ancestries: %+v", catalog.Ancestries)
	}
	if len(catalog.Cultures) != 1 || catalog.Cultures[0].Name != "Test Culture" {
		t.Fatalf("unexpected pack cultures: %+v", catalog.Cultures)
	}
	if len(catalog.Compositions) != 1 || catalog.Compositions[0].Name != "Test Folk + Test Culture" {
		t.Fatalf("unexpected pack compositions: %+v", catalog.Compositions)
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
			Openness:  0.5,
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

func TestComposeResolvedProfileMergesTagsTraitsAndAffinities(t *testing.T) {
	ancestry := AncestryProfile{
		Name:       "Elf",
		Tags:       []string{"forest", "long-lived"},
		Traits:     ProfileTraitMap{"order": 0.4, "warlike": 0.1},
		Affinities: []ProfileAffinityRule{{TargetType: "ancestry", Target: "Dwarf", Weight: -0.2}},
	}
	culture := &CultureProfile{
		Name:       "Merchant League",
		Tags:       []string{"mercantile", "coastal"},
		Traits:     ProfileTraitMap{"mercantile": 0.8, "order": 0.6},
		Affinities: []ProfileAffinityRule{{TargetType: "tag", Target: "coastal", Weight: 0.3}},
	}
	resolved := ComposeResolvedProfile(ancestry, culture)
	if resolved.Name != "Elf + Merchant League" {
		t.Fatalf("unexpected resolved name: %q", resolved.Name)
	}
	if !hasProfileTag(resolved.Tags, "forest") || !hasProfileTag(resolved.Tags, "coastal") {
		t.Fatalf("expected merged tags, got %+v", resolved.Tags)
	}
	if resolved.Traits["order"] != 0.6 || resolved.Traits["mercantile"] != 0.8 {
		t.Fatalf("expected merged traits, got %+v", resolved.Traits)
	}
	if len(resolved.Affinities) != 2 {
		t.Fatalf("expected merged affinities, got %+v", resolved.Affinities)
	}
}

func TestScoreProfileAffinity(t *testing.T) {
	subject := &ResolvedProfile{
		Name:         "Elf + Merchant League",
		AncestryName: "Elf",
		CultureName:  "Merchant League",
		Tags:         []string{"forest", "mercantile"},
		Affinities: []ProfileAffinityRule{
			{TargetType: "ancestry", Target: "Dwarf", Weight: -0.2},
			{TargetType: "tag", Target: "coastal", Weight: 0.3},
			{TargetType: "trait", Target: "order", Weight: 0.5},
		},
	}
	target := ProfileAffinityContext{
		ProfileName:  "Dwarf + Merchant League",
		AncestryName: "Dwarf",
		CultureName:  "Merchant League",
		Tags:         []string{"coastal", "mountain"},
		Traits:       map[string]float64{"order": 0.8},
	}
	score := ScoreProfileAffinity(subject, target)
	if score <= 0.49 || score >= 0.51 {
		t.Fatalf("unexpected affinity score %.3f", score)
	}
}

func floatPtr(v float64) *float64 { return &v }
