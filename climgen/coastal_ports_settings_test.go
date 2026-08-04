package climgen

import (
	"encoding/json"
	"strings"
	"testing"

	worldgen "worldgen"
)

func TestDefaultMaritimePortSettingsLoadsEmbeddedJSON(t *testing.T) {
	settings := DefaultMaritimePortSettings()
	if settings.SchemaVersion != MaritimePortSchemaVersion {
		t.Fatalf("expected maritime port schema %q, got %q", MaritimePortSchemaVersion, settings.SchemaVersion)
	}
	if settings.MajorPortThreshold <= 0 {
		t.Fatalf("expected positive major port threshold")
	}
	if settings.NodeCatchmentHops < 1 || settings.NodeCatchmentDecay <= 0 {
		t.Fatalf("expected positive coastal node catchment settings")
	}
	if settings.DeepDraftHarborWeight <= 0 || settings.NodeFeatureHarborWeight <= 0 {
		t.Fatalf("expected positive vessel-port compatibility weights")
	}
	if settings.DeepwaterAccessWeight <= 0 || settings.MajorDeepwaterPortThreshold <= 0 {
		t.Fatalf("expected positive deepwater port settings")
	}
	if settings.StopoverSelection.FullComponentAreaEq <= 0 || settings.StopoverSelection.MinComponentScoreFactor <= 0 {
		t.Fatalf("expected positive stopover selection area taper settings")
	}
}

// A v1 document predates the required stopoverSelection block. The loader must
// report the version mismatch rather than a field-level error about the missing
// block, which would point the author at the wrong fix.
func TestMaritimePortSettingsRejectV1SchemaWithVersionError(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal(worldgen.EmbeddedMaritimePortSettings(), &doc); err != nil {
		t.Fatalf("decode embedded maritime port settings: %v", err)
	}
	doc["schemaVersion"] = "maritime-ports/v1"
	delete(doc, "stopoverSelection")
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("re-encode maritime port settings: %v", err)
	}
	if _, err := loadMaritimePortSettingsData(raw); err == nil {
		t.Fatalf("expected v1 maritime port document to be rejected")
	} else if !strings.Contains(err.Error(), "unsupported maritime port schemaVersion") {
		t.Fatalf("expected schemaVersion error, got %v", err)
	}
}
