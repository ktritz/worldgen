package terrain

import (
	"math"
	"math/rand"
	"testing"
)

// Hotspot peak caps must depend only on the island's physical size, not the
// mesh resolution: the normalization uses the BASELINE (L5) cell angular
// radius, so a Hawaii-scale island gets the same cap at L5 and L7.
func TestHotspotPeakCapsUseBaselineCellRadius(t *testing.T) {
	baseline := 2.0 / math.Sqrt(10242.0)
	if math.Abs(baselineCellAngularRadius-baseline) > 1e-15 {
		t.Fatalf("baselineCellAngularRadius = %.9f, want %.9f (L5 cell radius)", baselineCellAngularRadius, baseline)
	}

	// Exact L5 no-op: matches the historical formula evaluated at 10242 cells.
	radius := 0.014 // Hawaii-scale island
	wantOceanic := 800.0 + 850.0*math.Min(radius/baseline, 4.0)
	if got := oceanicIslandPeakCap(radius); got != wantOceanic {
		t.Fatalf("oceanicIslandPeakCap(%.4f) = %.2f, want %.2f", radius, got, wantOceanic)
	}
	// Sanity: a Hawaii-scale island caps near ~1400m, far from the 4200m ceiling
	// that a finer mesh normalization would incorrectly allow.
	if got := oceanicIslandPeakCap(radius); got < 1300 || got > 1500 {
		t.Fatalf("oceanicIslandPeakCap(%.4f) = %.2f, want ~1400", radius, got)
	}
	// Ceiling for very large physical radii.
	if got := oceanicIslandPeakCap(1.0); got != 800.0+850.0*4.0 {
		t.Fatalf("oceanicIslandPeakCap ceiling = %.2f, want %.2f", got, 800.0+850.0*4.0)
	}

	wantContinental := 600.0 + 400.0*math.Min(radius/baseline, 3.0)
	if got := underwaterContinentalPeakCap(radius); got != wantContinental {
		t.Fatalf("underwaterContinentalPeakCap(%.4f) = %.2f, want %.2f", radius, got, wantContinental)
	}
	if got := underwaterContinentalPeakCap(1.0); got != 600.0+400.0*3.0 {
		t.Fatalf("underwaterContinentalPeakCap ceiling = %.2f, want %.2f", got, 600.0+400.0*3.0)
	}
}

// The continental caldera spread selects cells by true angular distance, so
// its footprint is physical and identical at any mesh resolution.
func TestSpreadAdditiveElevationRadialSelectsByAngularDistance(t *testing.T) {
	siteAt := func(angle float64) Vector3D {
		return Vector3D{X: math.Sin(angle), Y: 0, Z: math.Cos(angle)}
	}
	sites := []Vector3D{
		siteAt(0),     // center
		siteAt(0.004), // inside radius, near center
		siteAt(0.020), // outside radius
		siteAt(0.009), // inside radius, near edge
	}
	elevation := []float64{1000, 500, 500, 500}
	hotspotCells := map[int]HotspotCellInfo{}
	rng := rand.New(rand.NewSource(1))

	count := spreadAdditiveElevationRadial(
		elevation, sites, 0, 0.01, 800, len(sites),
		func(int) bool { return true },
		hotspotCells, rng,
	)

	if count != 2 {
		t.Fatalf("spread cell count = %d, want 2 (cells inside targetRadius)", count)
	}
	if elevation[0] != 1000 {
		t.Fatalf("center elevation modified to %.2f, want untouched 1000", elevation[0])
	}
	if elevation[2] != 500 {
		t.Fatalf("cell beyond targetRadius boosted to %.2f, want untouched 500", elevation[2])
	}
	nearBoost := elevation[1] - 500
	edgeBoost := elevation[3] - 500
	if nearBoost <= 0 || edgeBoost <= 0 {
		t.Fatalf("cells inside targetRadius not boosted: near=%.2f edge=%.2f", nearBoost, edgeBoost)
	}
	// Gaussian-like falloff: even with +/-20% random variation, the near-center
	// boost range [351,528] cannot overlap the near-edge range [166,250].
	if nearBoost <= edgeBoost {
		t.Fatalf("boost should decay with distance: near=%.2f edge=%.2f", nearBoost, edgeBoost)
	}
	if _, ok := hotspotCells[1]; !ok {
		t.Fatal("boosted cell 1 missing from hotspotCells")
	}
	if _, ok := hotspotCells[3]; !ok {
		t.Fatal("boosted cell 3 missing from hotspotCells")
	}
	if _, ok := hotspotCells[2]; ok {
		t.Fatal("cell outside targetRadius should not be tracked in hotspotCells")
	}
}

func TestSpreadAdditiveElevationRadialZeroRadiusIsNoop(t *testing.T) {
	sites := []Vector3D{{X: 0, Y: 0, Z: 1}, {X: 0.01, Y: 0, Z: 0.9999}}
	elevation := []float64{1000, 500}
	if count := spreadAdditiveElevationRadial(
		elevation, sites, 0, 0, 800, len(sites),
		func(int) bool { return true },
		map[int]HotspotCellInfo{}, rand.New(rand.NewSource(1)),
	); count != 0 {
		t.Fatalf("zero-radius spread modified %d cells, want 0", count)
	}
}
