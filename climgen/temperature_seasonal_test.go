package climgen

import (
	"math"
	"testing"
)

func TestSeasonalResponseFactorFavorsContinentalLand(t *testing.T) {
	settings := SeasonalTemperatureSettings{
		LandResponse:  0.85,
		OceanResponse: 0.30,
	}

	ocean := seasonalResponseFactor(-100, 0, 0, settings)
	coast := seasonalResponseFactor(100, 0, 0, settings)
	interior := seasonalResponseFactor(100, 0, 1, settings)

	if math.Abs(ocean-settings.OceanResponse) > 1e-9 {
		t.Fatalf("expected ocean response %.2f, got %.2f", settings.OceanResponse, ocean)
	}
	if math.Abs(coast-settings.OceanResponse) > 1e-9 {
		t.Fatalf("expected coastal land to start at ocean response %.2f, got %.2f", settings.OceanResponse, coast)
	}
	if math.Abs(interior-settings.LandResponse) > 1e-9 {
		t.Fatalf("expected inland response %.2f, got %.2f", settings.LandResponse, interior)
	}
}

func TestSummarizeSeasonalSnapshotsTracksExtremes(t *testing.T) {
	snapshots := []SeasonalTemperatureSnapshot{
		{Temperature: []float64{280, 285, 290}},
		{Temperature: []float64{275, 288, 289}},
		{Temperature: []float64{282, 281, 295}},
	}

	mins, maxs, ranges, warmest, coldest := summarizeSeasonalSnapshots(snapshots)
	if mins[0] != 275 || coldest[0] != 1 {
		t.Fatalf("expected first cell min 275 at season 1, got %.1f at %d", mins[0], coldest[0])
	}
	if maxs[2] != 295 || warmest[2] != 2 {
		t.Fatalf("expected third cell max 295 at season 2, got %.1f at %d", maxs[2], warmest[2])
	}
	if ranges[1] != 7 {
		t.Fatalf("expected second cell range 7, got %.1f", ranges[1])
	}
}

func TestSeasonalBlendResponseBoostsFrozenLandWarming(t *testing.T) {
	settings := SeasonalTemperatureSettings{
		LandResponse:      0.85,
		OceanResponse:     0.30,
		ThawResponseBoost: 0.35,
	}

	base := seasonalResponseFactor(100, 0, 1, settings)
	boosted := seasonalBlendResponse(base, FreezingPoint-18, FreezingPoint+8, 100, 0, 1, settings, 1.0, 0.0)
	if boosted <= base {
		t.Fatalf("expected thaw blend response %.3f to exceed base %.3f", boosted, base)
	}
	if boosted > 1.0 {
		t.Fatalf("expected thaw blend response <= 1, got %.3f", boosted)
	}

	oceanBoosted := seasonalBlendResponse(base, FreezingPoint-18, FreezingPoint+8, -100, 0, 1, settings, 1.0, 0.0)
	if math.Abs(oceanBoosted-base) > 1e-9 {
		t.Fatalf("expected ocean thaw response to remain at base %.3f, got %.3f", base, oceanBoosted)
	}
}

func TestUpdateSeasonalLandIceStateThawsLandFasterThanFreeze(t *testing.T) {
	settings := SeasonalTemperatureSettings{
		LandIceThawPersistence:   0.35,
		LandIceFreezePersistence: 0.80,
	}
	current := []float64{1.0, 0.0}
	equilibrium := []float64{0.0, 1.0}
	elevation := []float64{100, 100}

	next := updateSeasonalLandIceState(current, equilibrium, elevation, 0, settings)
	if next[0] >= 0.5 {
		t.Fatalf("expected thawed land ice state to drop quickly, got %.2f", next[0])
	}
	if next[1] <= 0.0 || next[1] >= 0.5 {
		t.Fatalf("expected freezing land ice state to rise slowly into (0, 0.5), got %.2f", next[1])
	}
}
