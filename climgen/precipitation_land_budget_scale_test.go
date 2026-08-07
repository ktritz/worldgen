package climgen

import (
	"math"
	"testing"
)

// precipTestBaselineCells is the L5 mesh the precipitation tuning is calibrated
// against; every per-physical-step conversion must be an exact no-op there.
const precipTestBaselineCells = 10242

// precipTestFineCells is L7 (four subdivisions of area per level -> a physical
// step scale of 0.25, i.e. four mesh steps per baseline step).
const precipTestFineCells = 163842

// landBudgetChannelTotals runs a one-dimensional caricature of the land moisture
// budget along a *fixed physical distance* at the given mesh size. It walks the
// same production entry points the relaxation in computePrecipitationBudget
// uses - the land moisture source, the surface recycling source and the land
// condensation drain - so the three land-side fractions are exercised exactly as
// they are in the real budget.
//
// baselineSteps is the number of hops the L5 mesh needs to cover the distance;
// a finer mesh covers the same distance in baselineSteps/stepScale hops. A
// mesh-invariant budget must therefore end with the same accumulated
// condensation and the same residual humidity regardless of cell count.
func landBudgetChannelTotals(t *testing.T, cellCount int, baselineSteps int) (condensedTotal, finalHumidity float64) {
	t.Helper()

	settings := DefaultPrecipitationSettings()
	vertices := make([]Vector3D, cellCount)
	elevation := make([]float64, cellCount)
	temperature := make([]float64, cellCount)
	for i := range vertices {
		// A fixed mid-latitude site: the source term reads latitude, and every
		// cell must agree so the channel below is homogeneous.
		vertices[i] = Vector3D{X: math.Cos(0.35), Y: math.Sin(0.35), Z: 0}
		elevation[i] = 200
		temperature[i] = 293.15
	}
	landSource := computeLandMoistureSource(vertices, elevation, 0, temperature, settings)

	stepScale := precipitationPhysicalStepScale(cellCount)
	steps := int(math.Round(float64(baselineSteps) / stepScale))
	// rainfallFractionPerCell is already per physical distance in production;
	// mirror that so the base condensation term is set up the same way.
	rainfallFractionPerCell := settings.RainfallFraction * estimateClimateCellSizeKm(cellCount)
	capacity := computeMoistureCapacity(temperature)[0]

	q := 0.9 * capacity
	for step := 0; step < steps; step++ {
		q += landSource[0]
		q += computeLandRecyclingSource(q, temperature, 0, 0.65, 0.25, settings.LandRecyclingScale, cellCount)
		diag := computeLandCondensationDiagnostic(
			q,
			capacity,
			0.20, // uplift
			0.15, // convergence
			0.40, // oceanFetch
			0.35, // onshore
			0.50, // landTravel
			0.60, // landInterior
			0.30, // marineShare
			1.0,  // condensationScale
			rainfallFractionPerCell,
			temperature,
			0,
			cellCount,
		)
		condensedTotal += diag.Condensed
		q = math.Max(0, q-diag.Condensed)
	}
	return condensedTotal, q
}

// TestLandBudgetFractionsArePerPhysicalDistance is the mesh-invariance guard for
// the land side of the precipitation budget. The marine transfers beside these
// terms are already per physical step; when the land source, the recycling
// fraction and the supersaturation drain are left per *iteration*, a finer mesh
// applies them more times over the same physical distance and the whole budget
// drifts. This asserts the totals over a fixed physical distance agree at L5 and
// L7.
func TestLandBudgetFractionsArePerPhysicalDistance(t *testing.T) {
	const baselineSteps = 12
	baseCondensed, baseHumidity := landBudgetChannelTotals(t, precipTestBaselineCells, baselineSteps)
	fineCondensed, fineHumidity := landBudgetChannelTotals(t, precipTestFineCells, baselineSteps)

	if baseCondensed <= 0 || fineCondensed <= 0 {
		t.Fatalf("degenerate channel: base=%.6f fine=%.6f", baseCondensed, fineCondensed)
	}
	t.Logf(
		"condensed L5=%.6f L7=%.6f ratio=%.4f | humidity L5=%.6f L7=%.6f ratio=%.4f",
		baseCondensed, fineCondensed, fineCondensed/baseCondensed,
		baseHumidity, fineHumidity, fineHumidity/baseHumidity,
	)
	// The two meshes are different discretisations of the same continuous
	// budget, so first-order splitting error remains; anything past a few
	// percent means a term is still counted per iteration instead of per
	// physical step. Unfixed, this ratio is off by more than 2x.
	const tolerance = 0.06
	condensedRatio := fineCondensed / baseCondensed
	if math.Abs(condensedRatio-1) > tolerance {
		t.Errorf(
			"land condensation over a fixed physical distance is not mesh-invariant: L5=%.6f L7=%.6f ratio=%.4f",
			baseCondensed, fineCondensed, condensedRatio,
		)
	}
	// The residual humidity is the fixed point of the source/drain splitting, so
	// it carries a larger first-order offset between the two step sizes than the
	// integrated condensate does. Unfixed it is off by 1.50x, so 0.12 still
	// separates the two cases cleanly.
	const humidityTolerance = 0.12
	humidityRatio := fineHumidity / baseHumidity
	if math.Abs(humidityRatio-1) > humidityTolerance {
		t.Errorf(
			"residual land humidity over a fixed physical distance is not mesh-invariant: L5=%.6f L7=%.6f ratio=%.4f",
			baseHumidity, fineHumidity, humidityRatio,
		)
	}
}

// TestLandBudgetFractionsAreExactAtBaseline pins the no-op requirement: the
// per-physical-step conversions must not perturb the L5 tuning at all.
func TestLandBudgetFractionsAreExactAtBaseline(t *testing.T) {
	if got := precipitationPerStepFraction(precipLandSourceFraction, precipTestBaselineCells); got != precipLandSourceFraction {
		t.Errorf("land source fraction changed at baseline: got %v want %v", got, precipLandSourceFraction)
	}
	if got := precipitationPerStepFraction(precipLandRecyclingFraction, precipTestBaselineCells); got != precipLandRecyclingFraction {
		t.Errorf("land recycling fraction changed at baseline: got %v want %v", got, precipLandRecyclingFraction)
	}
	if got := precipitationPerStepFraction(precipLandSupersatFraction, precipTestBaselineCells); got != precipLandSupersatFraction {
		t.Errorf("land supersat fraction changed at baseline: got %v want %v", got, precipLandSupersatFraction)
	}
}
