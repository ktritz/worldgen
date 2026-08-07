package climgen

import "math"

type landCondensationDiagnostic struct {
	Condensed            float64
	BaseCondensation     float64
	SupersatCondensation float64
	SupersatSupport      float64
	TropicalCoastSupport float64
	CoastalPenalty       float64
	AscentFraction       float64
	ConvectivePotential  float64
	MixingFraction       float64
	EffectiveCapacity    float64
	SupersatHumidity     float64
}

func computeLandCondensation(
	q float64,
	capacity float64,
	uplift float64,
	convergence float64,
	oceanFetch float64,
	onshore float64,
	landTravel float64,
	landInterior float64,
	marineShare float64,
	condensationScale float64,
	rainfallFractionPerCell float64,
	temperature []float64,
	idx int,
	cellCount int,
) float64 {
	return computeLandCondensationDiagnostic(
		q,
		capacity,
		uplift,
		convergence,
		oceanFetch,
		onshore,
		landTravel,
		landInterior,
		marineShare,
		condensationScale,
		rainfallFractionPerCell,
		temperature,
		idx,
		cellCount,
	).Condensed
}

func computeLandCondensationDiagnostic(
	q float64,
	capacity float64,
	uplift float64,
	convergence float64,
	oceanFetch float64,
	onshore float64,
	landTravel float64,
	landInterior float64,
	marineShare float64,
	condensationScale float64,
	rainfallFractionPerCell float64,
	temperature []float64,
	idx int,
	cellCount int,
) landCondensationDiagnostic {
	if q <= 0 {
		return landCondensationDiagnostic{}
	}
	tempC := 12.0
	if idx >= 0 && idx < len(temperature) {
		tempC = temperature[idx] - 273.15
	}
	convective := computeConvectiveCondensationPotential(q, capacity, tempC, convergence, landInterior)
	continentality := Clamp(landInterior, 0, 1)
	maritimeLift := 0.25 + 0.75*Clamp(onshore, 0, 1)
	positiveConvergence := math.Max(0, convergence)
	subsidence := math.Max(0, -convergence)
	coastalImmediate := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * (1.0 - Clamp(landTravel, 0, 1))
	nearInlandCorridor := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * transportCorridorWeight(landTravel)
	deepInteriorCarry := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * Clamp(landTravel, 0, 1)
	maritimeCarry := Clamp(marineShare, 0, 1) * Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1)
	tropicalOnshoreSupport := smoothRamp(8.0, 24.0, tempC) * coastalImmediate * (0.50 + 0.50*Clamp(marineShare, 0, 1))
	coastalPenalty := (0.22 + 0.20*Clamp(marineShare, 0, 1)) * coastalImmediate * (1.0 - 0.75*tropicalOnshoreSupport)
	coldCondenseEfficiency := 0.45 + 0.55*smoothRamp(-18.0, 6.0, tempC)
	coastalOrographicFetch := Clamp(oceanFetch, 0, 1) * (0.40 + 0.25*coastalImmediate)
	ascent := precipOrographicCondenseFraction*uplift*(0.06+(0.08+0.20*coldCondenseEfficiency)*coastalOrographicFetch*maritimeLift) +
		precipConvergenceCondenseFraction*positiveConvergence*(0.18+0.08*coastalOrographicFetch*maritimeLift+0.28*convective+0.72*nearInlandCorridor+0.34*deepInteriorCarry+0.10*maritimeCarry)
	mixingFraction := Clamp(
		0.50+0.30*continentality+0.12*convective+0.18*Clamp(ascent, 0, 1)+0.24*nearInlandCorridor+0.16*deepInteriorCarry+0.08*maritimeCarry-coastalPenalty-0.16*subsidence,
		0.40,
		1.0,
	)
	activeHumidity := q * mixingFraction
	effectiveCapacity := capacity * (1.0 - 0.30*coldCondenseEfficiency*Clamp(ascent, 0, 1))
	if effectiveCapacity < 0.12*capacity {
		effectiveCapacity = 0.12 * capacity
	}

	supersatHumidity := q * (0.60 + 0.40*mixingFraction)
	supersat := supersatHumidity - effectiveCapacity
	if supersat < 0 {
		supersat = 0
	}
	backgroundFraction := precipLandBaseCondenseFraction + 0.26*continentality + precipLandConvectiveFraction*convective + 0.24*nearInlandCorridor + 0.12*deepInteriorCarry + 0.06*maritimeCarry - coastalPenalty - 0.08*subsidence
	if backgroundFraction < 0.02 {
		backgroundFraction = 0.02
	}
	supersatSupport := Clamp(
		0.18+
			0.24*positiveConvergence+
			0.18*convective+
			0.16*Clamp(ascent, 0, 1)+
			0.10*coastalImmediate+
			0.08*nearInlandCorridor+
			0.06*deepInteriorCarry+
			0.06*maritimeCarry,
		0.18,
		1.0,
	)
	baseCondensation := activeHumidity * rainfallFractionPerCell * backgroundFraction
	// baseCondensation is already per physical distance through
	// rainfallFractionPerCell. The supersaturation drain is a fraction removed
	// per iteration, i.e. per cell traversed, so it needs the same treatment or
	// a finer mesh drains supersaturation more times over the same physical
	// distance. It takes the *rate* form rather than the profile form because
	// the caller divides this condensate by the step size to report an
	// intensity — see precipitationPerStepRate.
	supersatFraction := precipitationPerStepRate(
		precipLandSupersatFraction*(0.72+0.28*coldCondenseEfficiency)*supersatSupport,
		cellCount,
	)
	supersatCondensation := supersat * supersatFraction
	condensed := baseCondensation + supersatCondensation
	scale := Clamp(condensationScale, 0.35, 2.2)
	condensed *= scale
	if condensed > q {
		condensed = q
	}
	return landCondensationDiagnostic{
		Condensed:            condensed,
		BaseCondensation:     baseCondensation * scale,
		SupersatCondensation: supersatCondensation * scale,
		SupersatSupport:      supersatSupport,
		TropicalCoastSupport: tropicalOnshoreSupport,
		CoastalPenalty:       coastalPenalty,
		AscentFraction:       Clamp(ascent, 0, 1),
		ConvectivePotential:  convective,
		MixingFraction:       mixingFraction,
		EffectiveCapacity:    effectiveCapacity,
		SupersatHumidity:     supersatHumidity,
	}
}

func computeLandRetainedHumidity(
	q float64,
	condensed float64,
	capacity float64,
	uplift float64,
	oceanFetch float64,
	onshore float64,
	landTravel float64,
	landInterior float64,
	marineShare float64,
	retentionScale float64,
) float64 {
	residual := q - condensed
	if residual < 0 {
		residual = 0
	}
	continentality := Clamp(landInterior, 0, 1)
	blocking := Clamp(uplift, 0, 1) * (0.20 + 0.80*Clamp(oceanFetch, 0, 1)*Clamp(onshore, 0, 1))
	coastalImmediate := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * (1.0 - Clamp(landTravel, 0, 1))
	nearInlandCorridor := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * transportCorridorWeight(landTravel)
	deepInteriorCarry := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * Clamp(landTravel, 0, 1)
	retentionFloor := precipLandRetentionFloor + 0.18*continentality + 0.10*(1.0-blocking) + 0.06*(1.0-Clamp(oceanFetch, 0, 1)) + (0.30+0.18*Clamp(marineShare, 0, 1))*coastalImmediate + (0.28+0.12*Clamp(marineShare, 0, 1))*nearInlandCorridor + (0.16+0.08*Clamp(marineShare, 0, 1))*deepInteriorCarry
	retentionFloor *= Clamp(retentionScale, 0.5, 1.9)
	floor := retentionFloor * capacity
	if floor > residual {
		return floor
	}
	return residual
}

func transportCorridorWeight(landTravel float64) float64 {
	travel := Clamp(landTravel, 0, 1)
	rise := smoothRamp(0.05, 0.45, travel)
	fall := 1.0 - smoothRamp(0.85, 1.0, travel)
	return Clamp(rise*fall, 0, 1)
}
