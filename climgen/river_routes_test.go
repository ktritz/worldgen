package climgen

import "testing"

func TestBuildRiverRouteDiagnosticsDetectsNavigableMainChannel(t *testing.T) {
	settings := DefaultRiverRouteSettings()
	result := BuildRiverRouteDiagnostics(
		nil,
		nil,
		nil,
		nil,
		[]float64{0.2, 0.3},
		0.0,
		&HydrologyBiomeInputs{
			Runoff:          []float64{95, 6},
			ChannelStrength: []float64{2.4, 0.25},
			CellClass:       []string{"confluence", "upland"},
		},
		settings,
	)
	if result.Diagnostics.Navigability[0] <= 0.6 {
		t.Fatalf("expected major confluence cell to be highly navigable, got %.2f", result.Diagnostics.Navigability[0])
	}
	if result.Diagnostics.Navigability[1] != 0 {
		t.Fatalf("expected weak upland cell to be non-navigable, got %.2f", result.Diagnostics.Navigability[1])
	}
	if result.Diagnostics.DownstreamTravelCost[0] >= result.Diagnostics.UpstreamTravelCost[0] {
		t.Fatalf("expected downstream travel to be cheaper than upstream, got down=%.2f up=%.2f", result.Diagnostics.DownstreamTravelCost[0], result.Diagnostics.UpstreamTravelCost[0])
	}
}
