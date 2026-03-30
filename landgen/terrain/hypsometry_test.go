package terrain

import (
	"math"
	"testing"
)

func TestEarthElevationAtFractionMatchesTargets(t *testing.T) {
	checks := map[float64]float64{
		HypsometricTargets[-6000]: -6000,
		HypsometricTargets[-5000]: -5000,
		HypsometricTargets[-4000]: -4000,
		HypsometricTargets[-3000]: -3000,
		HypsometricTargets[-200]:  -200,
		HypsometricTargets[0]:     0,
		HypsometricTargets[500]:   500,
		HypsometricTargets[1000]:  1000,
		HypsometricTargets[2000]:  2000,
		HypsometricTargets[3000]:  3000,
	}

	for fraction, wantElevation := range checks {
		got := earthElevationAtFraction(fraction)
		if math.Abs(got-wantElevation) > 1.0 {
			t.Fatalf("earthElevationAtFraction(%v) = %v, want %v", fraction, got, wantElevation)
		}
	}
}

func TestEarthReferenceCurveIsConsistent(t *testing.T) {
	shelfCoverage := HypsometricTargets[0] - HypsometricTargets[-200]
	if math.Abs(shelfCoverage-EarthShelfCoverage) > 0.0001 {
		t.Fatalf("shelf coverage = %v, want %v", shelfCoverage, EarthShelfCoverage)
	}

	deepOceanCoverage := HypsometricTargets[-5000]
	if math.Abs(deepOceanCoverage-EarthDeepOceanCoverage) > 0.0001 {
		t.Fatalf("deep ocean coverage = %v, want %v", deepOceanCoverage, EarthDeepOceanCoverage)
	}

	thresholds := []float64{-6000, -5000, -4000, -3000, -200, 0, 500, 1000, 2000, 3000}
	prev := -1.0
	for _, threshold := range thresholds {
		value := HypsometricTargets[threshold]
		if value < prev {
			t.Fatalf("hypsometric target at %.0f (%v) decreased from previous target %v", threshold, value, prev)
		}
		prev = value
	}
}
