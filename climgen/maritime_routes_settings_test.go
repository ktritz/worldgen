package climgen

import "testing"

func TestDefaultMaritimeRouteSettingsLoadsEmbeddedJSON(t *testing.T) {
	settings := DefaultMaritimeRouteSettings()
	if settings.DefaultVessel != "coastal-sloop" {
		t.Fatalf("expected coastal-sloop default vessel, got %q", settings.DefaultVessel)
	}
	if len(settings.Vessels) < 6 {
		t.Fatalf("expected embedded maritime catalog, got %d vessels", len(settings.Vessels))
	}
	if _, ok := settings.VesselByName("caravel"); !ok {
		t.Fatalf("expected caravel in embedded maritime catalog")
	}
}

func TestMaritimeRouteSettingsValidateRejectsUnknownDefault(t *testing.T) {
	settings := MaritimeRouteSettings{
		DefaultVessel: "missing",
		Vessels: []MaritimeVesselSettings{
			{
				Name:              "coastal-sloop",
				TechLevel:         "medieval",
				Propulsion:        "sail",
				RouteClass:        "coastal",
				PayloadCapacity:   0.46,
				DailyRange:        0.72,
				LongHaulTolerance: 0.46,
			},
		},
	}
	if err := settings.Validate(); err == nil {
		t.Fatalf("expected validation error for unknown default vessel")
	}
}
