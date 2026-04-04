package climgen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCoastalResourceSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "coastal.json")
	data := []byte(`{
  "schemaVersion": "coastal-resources/v1",
  "openFisheryMultiplier": 1.2,
  "estuarineFisheryMultiplier": 0.9,
  "shellfishMultiplier": 0.8,
  "saltworksMultiplier": 1.1,
  "openFisheryPrimaryBias": 0.0,
  "estuarineFisheryPrimaryBias": 0.02,
  "shellfishPrimaryBias": 0.03,
  "saltworksPrimaryBias": 0.01
}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	settings, err := LoadCoastalResourceSettings(path)
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.OpenFisheryMultiplier != 1.2 {
		t.Fatalf("unexpected open fishery multiplier %.2f", settings.OpenFisheryMultiplier)
	}
	if settings.ShellfishPrimaryBias != 0.03 {
		t.Fatalf("unexpected shellfish primary bias %.2f", settings.ShellfishPrimaryBias)
	}
}

func TestValidateCoastalResourceSettingsRejectsNegativeMultiplier(t *testing.T) {
	settings := DefaultCoastalResourceSettings()
	settings.ShellfishMultiplier = -1
	if err := ValidateCoastalResourceSettings(settings); err == nil {
		t.Fatal("expected validation error for negative shellfish multiplier")
	}
}
