package climgen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLandRouteSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "land_routes.json")
	data := `{
		"defaultMode":"pack-lizard",
		"modes":[
			{"name":"pack","speedMultiplier":1.0},
			{"name":"pack-lizard","speedMultiplier":0.9,"marshPassability":0.95}
		]
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	settings, err := LoadLandRouteSettings(path)
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.DefaultMode != "pack-lizard" {
		t.Fatalf("expected default mode to load")
	}
	mode, ok := settings.ModeByName("pack-lizard")
	if !ok || mode.MarshPassability != 0.95 {
		t.Fatalf("expected named mode to load with marsh passability, got %+v", mode)
	}
}
