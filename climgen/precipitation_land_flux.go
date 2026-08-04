package climgen

import "math"

func localPrecipitationScale(scales []float64, idx int) float64 {
	if idx < 0 || idx >= len(scales) {
		return 1.0
	}
	return Clamp(scales[idx], 0.15, 2.5)
}

func localOptionalPrecipitationScale(scales []float64, idx int) float64 {
	if idx < 0 || idx >= len(scales) {
		return 0.0
	}
	return Clamp(scales[idx], 0, 2.5)
}

func localPrecipitationStorage(storage []float64, idx int) float64 {
	if idx < 0 || idx >= len(storage) {
		return 0.0
	}
	return Clamp(storage[idx], 0, 1)
}

func computeLandRecyclingSource(
	localHumidity float64,
	temperature []float64,
	idx int,
	landInterior float64,
	surfaceStorage float64,
	recycleScale float64,
) float64 {
	if idx < 0 || idx >= len(temperature) {
		return 0
	}
	tempC := temperature[idx] - 273.15
	evapPotential := smoothRamp(-2.0, 26.0, tempC)
	warmSeasonBoost := 0.55 + 0.45*smoothRamp(8.0, 30.0, tempC)
	recycleBias := 0.25 + 0.75*Clamp(landInterior, 0, 1)
	humidityFlux := localHumidity * precipLandRecyclingFraction * recycleScale * evapPotential * warmSeasonBoost * recycleBias
	storageFlux := Clamp(surfaceStorage, 0, 1) * precipLandRecyclingFraction * 0.30 * evapPotential * (0.30 + 0.70*warmSeasonBoost) * recycleBias
	return humidityFlux + storageFlux
}

func computeTropicalMarineSource(
	i int,
	marineField []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	oceanFetch []float64,
	coastalOnshore []float64,
	landTravel []float64,
	landInterior []float64,
	tropicalSourceScale []float64,
) float64 {
	scale := localOptionalPrecipitationScale(tropicalSourceScale, i)
	if scale <= 1e-9 || i < 0 || i >= len(elevation) || elevation[i] < seaLevel {
		return 0
	}

	localMarine := 0.0
	if i < len(marineField) {
		localMarine = marineField[i]
	}
	directLocal, _ := localUpwindMean(
		i,
		marineField,
		vertices,
		adj,
		wind,
		0.02,
		nil,
	)
	// Mean and max share the same footprint, so a single traversal serves both.
	footprintMean, _, footprintMax, _ := upwindFootprintMeanMax(
		i,
		marineField,
		vertices,
		adj,
		wind,
		resolutionAdjustedPrecipSteps(4, len(vertices)),
		0.02,
		nil,
	)

	travel := 0.0
	if i < len(landTravel) {
		travel = Clamp(landTravel[i], 0, 1)
	}
	interior := 0.0
	if i < len(landInterior) {
		interior = Clamp(landInterior[i], 0, 1)
	}
	onshore := 0.0
	if i < len(coastalOnshore) {
		onshore = Clamp(coastalOnshore[i], 0, 1)
	}
	coastalImmediate := 0.0
	if i < len(oceanFetch) {
		coastalImmediate = Clamp(oceanFetch[i], 0, 1) * onshore * (1.0 - travel)
	}
	corridor := transportCorridorWeight(travel)
	support := math.Max(directLocal, 0.65*footprintMax+0.35*footprintMean)

	available := 0.18*localMarine + 0.52*support + 0.30*footprintMean
	added := available *
		scale *
		(0.03 + 0.04*onshore + 0.11*corridor + 0.15*interior + 0.10*travel) *
		(1.0 - 0.42*coastalImmediate)
	return Clamp(added, 0, 0.38*available)
}
