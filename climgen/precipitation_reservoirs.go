package climgen

func splitCondensationReservoirs(
	condensedTotal float64,
	marineQ float64,
	landQ float64,
	oceanFetch float64,
	onshore float64,
	landTravel float64,
	landInterior float64,
	convective float64,
) (marineCondensed, landCondensed float64) {
	total := marineQ + landQ
	if condensedTotal <= 0 || total <= 1e-9 {
		return 0, 0
	}

	coastalImmediate := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * (1.0 - Clamp(landTravel, 0, 1))
	nearInlandCorridor := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * transportCorridorWeight(landTravel)
	deepInteriorCarry := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * Clamp(landTravel, 0, 1)
	continentality := Clamp(landInterior, 0, 1)

	marineWeight := marineQ * (1.0 + 1.20*coastalImmediate + 0.70*nearInlandCorridor + 0.20*deepInteriorCarry)
	landWeight := landQ * (1.0 + 0.55*continentality + 0.60*convective)

	marineCondensed = condensedTotal * reservoirShare(marineWeight, landWeight)
	landCondensed = condensedTotal - marineCondensed
	if marineCondensed > marineQ {
		marineCondensed = marineQ
	}
	if landCondensed > landQ {
		landCondensed = landQ
	}

	remaining := condensedTotal - (marineCondensed + landCondensed)
	if remaining > 1e-9 {
		if marineQ-marineCondensed >= landQ-landCondensed {
			extra := minFloat(remaining, marineQ-marineCondensed)
			marineCondensed += extra
			remaining -= extra
		}
		if remaining > 1e-9 {
			landCondensed += minFloat(remaining, landQ-landCondensed)
		}
	}
	return marineCondensed, landCondensed
}

func splitRetainedReservoirs(
	retainedTotal float64,
	marineRemaining float64,
	landRemaining float64,
	oceanFetch float64,
	onshore float64,
	landTravel float64,
	landInterior float64,
) (marineRetained, landRetained float64) {
	total := marineRemaining + landRemaining
	if retainedTotal <= 0 || total <= 1e-9 {
		return 0, 0
	}

	coastalImmediate := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * (1.0 - Clamp(landTravel, 0, 1))
	nearInlandCorridor := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * transportCorridorWeight(landTravel)
	deepInteriorCarry := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * Clamp(landTravel, 0, 1)
	continentality := Clamp(landInterior, 0, 1)

	marineWeight := marineRemaining * (1.0 + 0.95*coastalImmediate + 0.75*nearInlandCorridor + 0.25*deepInteriorCarry)
	landWeight := landRemaining * (1.0 + 0.65*continentality + 0.20*deepInteriorCarry)

	marineRetained = retainedTotal * reservoirShare(marineWeight, landWeight)
	landRetained = retainedTotal - marineRetained
	if marineRetained > marineRemaining {
		marineRetained = marineRemaining
	}
	if landRetained > landRemaining {
		landRetained = landRemaining
	}

	remaining := retainedTotal - (marineRetained + landRetained)
	if remaining > 1e-9 {
		if marineRemaining-marineRetained >= landRemaining-landRetained {
			extra := minFloat(remaining, marineRemaining-marineRetained)
			marineRetained += extra
			remaining -= extra
		}
		if remaining > 1e-9 {
			landRetained += minFloat(remaining, landRemaining-landRetained)
		}
	}
	return marineRetained, landRetained
}

func reservoirShare(primary, secondary float64) float64 {
	total := primary + secondary
	if total <= 1e-9 {
		return 0.5
	}
	return Clamp(primary/total, 0, 1)
}

func marineToLandMixFraction(
	oceanFetch float64,
	onshore float64,
	landTravel float64,
	landInterior float64,
) float64 {
	coastalImmediate := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * (1.0 - Clamp(landTravel, 0, 1))
	nearInlandCorridor := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * transportCorridorWeight(landTravel)
	deepInteriorCarry := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * Clamp(landTravel, 0, 1)
	continentality := Clamp(landInterior, 0, 1)

	mix := 0.04 + 0.18*nearInlandCorridor + 0.22*deepInteriorCarry + 0.12*continentality - 0.10*coastalImmediate
	return Clamp(mix, 0, 0.45)
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
