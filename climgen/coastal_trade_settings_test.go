package climgen

import "testing"

func TestDefaultCoastalTradeSettingsLoadsEmbeddedJSON(t *testing.T) {
	settings := DefaultCoastalTradeSettings()
	if settings.SchemaVersion != CoastalTradeSchemaVersion {
		t.Fatalf("expected coastal trade schema %q, got %q", CoastalTradeSchemaVersion, settings.SchemaVersion)
	}
	if settings.MaxPartnersPerPort < 1 {
		t.Fatalf("expected positive port partner limit")
	}
	if settings.CandidatePortSuitabilityFloor <= 0 || settings.CandidatePortFeatureFloor <= 0 {
		t.Fatalf("expected positive fallback coastal candidate floors")
	}
	if settings.RouteBudgetOpenOceanWeight <= 0 || settings.RouteBudgetLegWeight <= 0 {
		t.Fatalf("expected positive route-budget tuning weights")
	}
}
