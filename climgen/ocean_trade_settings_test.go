package climgen

import "testing"

func TestDefaultOceanTradeSettingsLoadsEmbeddedJSON(t *testing.T) {
	settings := DefaultOceanTradeSettings()
	if settings.SchemaVersion != OceanTradeSchemaVersion {
		t.Fatalf("expected ocean trade schema %q, got %q", OceanTradeSchemaVersion, settings.SchemaVersion)
	}
	if settings.MaxStopovers <= 0 || settings.RouteBudgetOpenOceanWeight <= 0 {
		t.Fatalf("expected positive ocean trade stopover and route budget settings")
	}
}
