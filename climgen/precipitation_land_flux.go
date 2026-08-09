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

// computeLandRecyclingSource returns the surface-recycled moisture added for one
// land-budget iteration. One iteration advects moisture one cell, so the
// recycling fraction is a per-step quantity and is converted to a
// per-physical-step fraction (exact no-op at the L5 baseline); otherwise a finer
// mesh recycles more times over the same physical distance.
func computeLandRecyclingSource(
	localHumidity float64,
	temperature []float64,
	idx int,
	landInterior float64,
	surfaceStorage float64,
	recycleScale float64,
	cellCount int,
) float64 {
	if idx < 0 || idx >= len(temperature) {
		return 0
	}
	tempC := temperature[idx] - 273.15
	evapPotential := smoothRamp(-2.0, 26.0, tempC)
	warmSeasonBoost := 0.55 + 0.45*smoothRamp(8.0, 30.0, tempC)
	recycleBias := 0.25 + 0.75*Clamp(landInterior, 0, 1)
	// Recycling is a rate feeding a reservoir, not a survival profile, so it
	// takes the linear per-step form; the profile form overshoots 3.2% at L6 and
	// 6.5% at L7. Exact no-op at L5.
	humidityFraction := precipitationPerStepRate(precipLandRecyclingFraction, cellCount)
	storageFraction := precipitationPerStepRate(precipLandRecyclingFraction*0.30, cellCount)
	humidityFlux := localHumidity * humidityFraction * recycleScale * evapPotential * warmSeasonBoost * recycleBias
	storageFlux := Clamp(surfaceStorage, 0, 1) * storageFraction * evapPotential * (0.30 + 0.70*warmSeasonBoost) * recycleBias
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
	return combineTropicalMarineSource(
		i,
		localMarine,
		directLocal,
		footprintMean,
		footprintMax,
		scale,
		oceanFetch,
		coastalOnshore,
		landTravel,
		landInterior,
	)
}

// combineTropicalMarineSource is the weight-independent tail of
// computeTropicalMarineSource. It is shared with the batched path so the two
// forms cannot drift apart.
func combineTropicalMarineSource(
	i int,
	localMarine float64,
	directLocal float64,
	footprintMean float64,
	footprintMax float64,
	scale float64,
	oceanFetch []float64,
	coastalOnshore []float64,
	landTravel []float64,
	landInterior []float64,
) float64 {
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

// tropicalMarineUpwind holds the batched upwind footprint inputs that
// computeTropicalMarineSource would otherwise recompute with one frontier BFS
// per cell. The marine field is constant across the land-budget iterations, so
// the whole table is built once.
type tropicalMarineUpwind struct {
	directLocal   []float64
	footprintMean []float64
	footprintMax  []float64
}

func computeTropicalMarineUpwind(
	marineField []float64,
	vertices []Vector3D,
	cache *upwindTransitionCache,
) *tropicalMarineUpwind {
	p := cache.get(0.02)
	coeffs := upwindFootprintCoeffs(resolutionAdjustedPrecipSteps(4, len(vertices)), len(vertices))
	directLocal, _ := batchLocalUpwindMean(p, marineField, nil)
	footprintMean, _ := batchUpwindFootprintMean(p, coeffs, marineField, nil)
	footprintMax, _ := batchUpwindFootprintMax(p, coeffs, marineField, nil)
	return &tropicalMarineUpwind{
		directLocal:   directLocal,
		footprintMean: footprintMean,
		footprintMax:  footprintMax,
	}
}

// source is the batched equivalent of computeTropicalMarineSource for cell i.
func (u *tropicalMarineUpwind) source(
	i int,
	marineField []float64,
	elevation []float64,
	seaLevel float64,
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
	return combineTropicalMarineSource(
		i,
		localMarine,
		u.directLocal[i],
		u.footprintMean[i],
		u.footprintMax[i],
		scale,
		oceanFetch,
		coastalOnshore,
		landTravel,
		landInterior,
	)
}
