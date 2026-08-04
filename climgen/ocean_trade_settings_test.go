package climgen

import (
	"encoding/json"
	"strings"
	"testing"

	worldgen "worldgen"
)

func TestDefaultOceanTradeSettingsLoadsEmbeddedJSON(t *testing.T) {
	settings := DefaultOceanTradeSettings()
	if settings.SchemaVersion != OceanTradeSchemaVersion {
		t.Fatalf("expected ocean trade schema %q, got %q", OceanTradeSchemaVersion, settings.SchemaVersion)
	}
	if settings.MaxStopovers <= 0 || settings.MaxCandidatePortsPerCiv <= 0 || settings.RouteBudgetOpenOceanWeight <= 0 || settings.CandidatePhysicalDeepwaterFloor <= 0 {
		t.Fatalf("expected positive ocean trade stopover, port cap, physical port floor, and route budget settings")
	}
}

// A v1 document predates maxCandidatePortsPerCivilization. The loader must report
// the version mismatch rather than the field-level count error.
func TestOceanTradeSettingsRejectV1SchemaWithVersionError(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal(worldgen.EmbeddedOceanTradeSettings(), &doc); err != nil {
		t.Fatalf("decode embedded ocean trade settings: %v", err)
	}
	doc["schemaVersion"] = "ocean-trade/v1"
	delete(doc, "maxCandidatePortsPerCivilization")
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("re-encode ocean trade settings: %v", err)
	}
	if _, err := loadOceanTradeSettingsData(raw); err == nil {
		t.Fatalf("expected v1 ocean trade document to be rejected")
	} else if !strings.Contains(err.Error(), "unsupported ocean trade schemaVersion") {
		t.Fatalf("expected schemaVersion error, got %v", err)
	}
}
