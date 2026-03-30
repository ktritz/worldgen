package climgen

import "testing"

func TestSeasonalThermalEquatorShiftDegTracksSolarDeclination(t *testing.T) {
	if shift := SeasonalThermalEquatorShiftDeg(SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.0}); shift >= 0 {
		t.Fatalf("expected NH winter shift southward, got %.2f", shift)
	}
	if shift := SeasonalThermalEquatorShiftDeg(SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5}); shift <= 0 {
		t.Fatalf("expected NH summer shift northward, got %.2f", shift)
	}
}

func TestSummarizeSeasonalPrecipitationTracksExtremes(t *testing.T) {
	snapshots := []SeasonalClimateSnapshot{
		{Precipitation: []float64{10, 20, 5}},
		{Precipitation: []float64{30, 15, 8}},
		{Precipitation: []float64{12, 40, 4}},
	}

	wettest, driest, rng := summarizeSeasonalPrecipitation(snapshots)
	if wettest[0] != 1 || driest[0] != 0 {
		t.Fatalf("expected cell 0 wettest=1 driest=0, got %d %d", wettest[0], driest[0])
	}
	if wettest[1] != 2 || rng[1] != 25 {
		t.Fatalf("expected cell 1 wettest=2 range=25, got %d %.1f", wettest[1], rng[1])
	}
	if driest[2] != 2 || rng[2] != 4 {
		t.Fatalf("expected cell 2 driest=2 range=4, got %d %.1f", driest[2], rng[2])
	}
}
