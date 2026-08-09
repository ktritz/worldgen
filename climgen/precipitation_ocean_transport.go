package climgen

import "math"

type oceanAtmosphereDiagnostics struct {
	OceanSource  []float64
	WarmMarine   []float64
	DownwindLand []float64
	Retention    []float64
}

func computeCoastalMarineRetention(oceanSource, warmMarine, downwindLand float64) float64 {
	retention := 1.0 - 0.10*downwindLand*warmMarine
	hotLandfallSource := smoothRamp(0.75, 0.95, oceanSource) *
		smoothRamp(0.55, 0.90, downwindLand) *
		smoothRamp(0.55, 0.95, warmMarine)
	retention -= 0.12 * hotLandfallSource
	return Clamp(retention, 0.76, 1.0)
}

func computeDownwindLandExposure(
	i int,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
) float64 {
	if i < 0 || i >= len(vertices) || i >= len(elevation) || i >= len(wind) || elevation[i] >= seaLevel {
		return 0
	}
	windSpeed := Length(wind[i])
	if windSpeed < 1e-9 {
		return 0
	}
	windDir := Scale(wind[i], 1.0/windSpeed)
	exposure := 0.0
	weightSum := 0.0
	for _, k := range adj.GetNeighbors(i) {
		if k < 0 || k >= len(vertices) || k >= len(elevation) || elevation[k] < seaLevel {
			continue
		}
		toNeighbor := Normalize(Sub(vertices[k], vertices[i]))
		downwind := Dot(windDir, toNeighbor)
		if downwind <= 0 {
			continue
		}
		exposure += downwind
		weightSum += 1.0
	}
	if weightSum <= 1e-9 {
		return 0
	}
	return Clamp(exposure/weightSum, 0, 1)
}

func computeOceanAtmosphericMoisture(
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	oceanSource []float64,
	moistureCap []float64,
	rainfallFractionPerCell float64,
	maxIterations int,
) ([]float64, oceanAtmosphereDiagnostics) {
	moisture := append([]float64(nil), oceanSource...)
	stepScale := precipitationPhysicalStepScale(len(vertices))
	diag := oceanAtmosphereDiagnostics{
		OceanSource:  append([]float64(nil), oceanSource...),
		WarmMarine:   make([]float64, len(oceanSource)),
		DownwindLand: make([]float64, len(oceanSource)),
		Retention:    make([]float64, len(oceanSource)),
	}
	for iter := 0; iter < maxIterations; iter++ {
		next := make([]float64, len(moisture))
		maxChange := 0.0
		for i := range moisture {
			if i >= len(elevation) || elevation[i] >= seaLevel {
				continue
			}
			incoming := advectedSpecificHumidity(i, vertices, adj, wind, moisture)
			warmMarine := 0.0
			if i < len(moistureCap) {
				// warm-ocean depletion is driven by SST/air temperature proxy through moisture capacity source scaling;
				// use source strength as a robust local proxy for warm humid marine air.
				warmMarine = smoothRamp(0.45, 1.05, oceanSource[i])
			}
			downwindLand := computeDownwindLandExposure(i, vertices, elevation, seaLevel, adj, wind)
			coastalMarineRetention := computeCoastalMarineRetention(oceanSource[i], warmMarine, downwindLand)
			diag.WarmMarine[i] = warmMarine
			diag.DownwindLand[i] = downwindLand
			diag.Retention[i] = coastalMarineRetention
			// The per-step update is affine in the incoming humidity:
			//   next = a(x)*incoming + b(x),  x = stepScale
			// and a streamline relaxes to the local attractor b/(1-a). Converting
			// each term separately makes that attractor mesh-dependent, because
			// the supersaturation drain relaxes q toward moistureCap rather than
			// toward zero, so the capacity enters the fixed point as an effective
			// source of strength cap*(1-(1-F)^x). Per unit physical distance that
			// source rises from 0.450 at the baseline to 0.517 at L6 and 0.555 at
			// L7 while the true evaporation source stays exactly proportional to
			// x, lifting the whole marine field about 11% per halving and raising
			// its ceiling with it. It is a first-order operator-splitting error,
			// and the baseline sits ~24% below the continuum limit, which is why
			// the inflation is monotone and saturating rather than divergent.
			//
			// Integrating the affine relaxation exactly instead makes both the
			// attractor and the rate of approach to it independent of the step:
			// with a(x) = a1^x and b(x) = b1*(1-a1^x)/(1-a1), the fixed point is
			// b1/(1-a1) for every x. The baseline keeps the original expression
			// verbatim so it stays bit-identical.
			retentionBase := precipOceanRetentionFraction * coastalMarineRetention
			if stepScale == 1 {
				q := incoming + oceanSource[i]
				condensed := computeOceanCondensation(q, moistureCap[i], rainfallFractionPerCell, len(vertices))
				next[i] = maxFloat(0, (q-condensed)*retentionBase)
			} else {
				next[i] = maxFloat(0, advanceOceanMarineColumn(
					incoming,
					oceanSource[i],
					moistureCap[i],
					retentionBase,
					rainfallFractionPerCell,
					stepScale,
				))
			}
			change := absPrecipFloat(next[i] - moisture[i])
			if change > maxChange {
				maxChange = change
			}
		}
		moisture = next
		if maxChange < 0.0005 {
			break
		}
	}
	return moisture, diag
}

func absPrecipFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// advanceOceanMarineColumn advances the marine column one physical step by
// integrating the affine per-step relaxation exactly, rather than converting its
// retention, source and condensation terms separately.
//
// The baseline (unit-step) update is next = a1*incoming + b1. Raising a1 to the
// step and scaling b1 to match reproduces the same continuous relaxation at any
// step size, so the attractor b1/(1-a1) is identical at every resolution. The
// supersaturation branch is chosen on the *baseline* humidity, otherwise which
// branch a cell takes would itself depend on the mesh.
func advanceOceanMarineColumn(incoming, source, capacity, retentionBase, rainfallFractionPerCell, stepScale float64) float64 {
	baselineQ := incoming + source
	// rainfallFractionPerCell is a per-km rate times the cell width; divide the
	// step back out to recover the fraction condensed per baseline cell.
	background := rainfallFractionPerCell / stepScale * precipOceanCondensationFraction
	keep := 1 - background
	drain := 0.0
	if baselineQ > capacity {
		keep -= precipOceanSupersatFraction
		drain = capacity * precipOceanSupersatFraction
	}
	a1 := retentionBase * keep
	b1 := retentionBase * (source*keep + drain)
	if !(a1 > 0 && a1 < 1) {
		// Degenerate coefficients (no retention, or condensation removing the
		// whole column): fall back to the per-step form.
		return (baselineQ - math.Min(baselineQ, background*baselineQ+drain)) * math.Pow(retentionBase, stepScale)
	}
	aStep := math.Pow(a1, stepScale)
	return aStep*incoming + b1*(1-aStep)/(1-a1)
}
