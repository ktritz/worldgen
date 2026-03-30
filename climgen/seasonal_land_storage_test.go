package climgen

import "testing"

func TestAdvanceSeasonalLandStorageRechargesWetInterior(t *testing.T) {
	elevation := []float64{100}
	interior := []float64{1.0}
	temperature := []float64{295.15}
	initial := []float64{0.25}
	rain := []float64{48.0}
	snow := []float64{0.0}

	next := AdvanceSeasonalLandStorage(initial, elevation, 0.0, temperature, rain, snow, interior)
	if next[0] <= initial[0] {
		t.Fatalf("expected wet season to recharge storage: got %.3f <= %.3f", next[0], initial[0])
	}
}

func TestAdvanceSeasonalLandStorageDriesHotExposedLand(t *testing.T) {
	elevation := []float64{100}
	interior := []float64{0.0}
	temperature := []float64{304.15}
	initial := []float64{0.80}
	rain := []float64{3.0}
	snow := []float64{0.0}

	next := AdvanceSeasonalLandStorage(initial, elevation, 0.0, temperature, rain, snow, interior)
	if next[0] >= initial[0] {
		t.Fatalf("expected hot dry season to reduce storage: got %.3f >= %.3f", next[0], initial[0])
	}
}

func TestAdvanceSeasonalLandStorageTreatsColdSnowAsLimitedRecharge(t *testing.T) {
	elevation := []float64{100}
	interior := []float64{0.6}
	initial := []float64{0.30}
	coldTemp := []float64{266.15}
	warmTemp := []float64{280.15}
	rain := []float64{0.0}
	snow := []float64{40.0}

	cold := AdvanceSeasonalLandStorage(initial, elevation, 0.0, coldTemp, rain, snow, interior)
	warm := AdvanceSeasonalLandStorage(initial, elevation, 0.0, warmTemp, rain, snow, interior)
	if warm[0] <= cold[0] {
		t.Fatalf("expected thaw-season snow recharge to exceed cold-season recharge: warm=%.3f cold=%.3f", warm[0], cold[0])
	}
}

func TestApplySeasonalLandStorageToSettingsCreatesLocalScales(t *testing.T) {
	settings := DefaultPrecipitationSettings()
	elevation := []float64{100, 100, -100}
	temperature := []float64{300.15, 281.15, 290.15}
	storage := []float64{0.85, 0.20, 0.0}

	adjusted := ApplySeasonalLandStorageToSettings(settings, elevation, 0.0, temperature, storage)
	if len(adjusted.LandSourceLocalScale) != len(storage) {
		t.Fatalf("expected local source scale length %d, got %d", len(storage), len(adjusted.LandSourceLocalScale))
	}
	if len(adjusted.LandRecyclingLocalScale) != len(storage) {
		t.Fatalf("expected local recycling scale length %d, got %d", len(storage), len(adjusted.LandRecyclingLocalScale))
	}
	if adjusted.LandSourceLocalScale[0] <= adjusted.LandSourceLocalScale[1] {
		t.Fatalf("expected warm wet land to have stronger source scale: %.3f <= %.3f", adjusted.LandSourceLocalScale[0], adjusted.LandSourceLocalScale[1])
	}
	if adjusted.LandRecyclingLocalScale[0] <= adjusted.LandRecyclingLocalScale[1] {
		t.Fatalf("expected warm wet land to have stronger recycling scale: %.3f <= %.3f", adjusted.LandRecyclingLocalScale[0], adjusted.LandRecyclingLocalScale[1])
	}
	if adjusted.LandSourceLocalScale[2] != 0 || adjusted.LandRecyclingLocalScale[2] != 0 {
		t.Fatalf("expected ocean cells to carry no local land-storage scale, got %.3f / %.3f", adjusted.LandSourceLocalScale[2], adjusted.LandRecyclingLocalScale[2])
	}
}
