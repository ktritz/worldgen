package climgen

import "testing"

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
