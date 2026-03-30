package climgen

import "math"

const (
	frontalLandDiffusionBlend       = 0.28
	frontalLandDiffusionIterations  = 3
	frontalStormTransportBlend      = 0.42
	frontalStormTransportIterations = 2
)

func computeFrontalMarineCaptureFraction(
	oceanFetch float64,
	onshore float64,
	landTravel float64,
	landInterior float64,
	sourceScale float64,
) float64 {
	if sourceScale <= 1e-9 {
		return 0
	}
	coastalImmediate := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * (1.0 - Clamp(landTravel, 0, 1))
	nearInlandCorridor := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * transportCorridorWeight(landTravel)
	deepInteriorCarry := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * Clamp(landTravel, 0, 1)
	continentality := Clamp(landInterior, 0, 1)

	frac := sourceScale * (0.05 + 0.22*nearInlandCorridor + 0.18*deepInteriorCarry + 0.10*continentality - 0.04*coastalImmediate)
	return Clamp(frac, 0, 0.46)
}

func computeFrontalStormSource(
	i int,
	marineField []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	oceanFetch []float64,
	landTravel []float64,
	landInterior []float64,
	frontalSourceScale []float64,
	frontalTransportScale []float64,
) float64 {
	transportScale := localOptionalPrecipitationScale(frontalTransportScale, i)
	if transportScale <= 1e-9 || i < 0 || i >= len(elevation) || elevation[i] < seaLevel {
		return 0
	}

	localMarine := 0.0
	if i >= 0 && i < len(marineField) {
		localMarine = marineField[i]
	}
	directUpwind, _ := frontalUpwindMean(i, marineField, vertices, elevation, seaLevel, adj, wind)
	secondHop := frontalTwoHopUpwindMean(i, marineField, vertices, elevation, seaLevel, adj, wind)

	travel := 0.0
	if i < len(landTravel) {
		travel = Clamp(landTravel[i], 0, 1)
	}
	interior := 0.0
	if i < len(landInterior) {
		interior = Clamp(landInterior[i], 0, 1)
	}
	coastalImmediate := 0.0
	if i < len(oceanFetch) {
		coastalImmediate = Clamp(oceanFetch[i], 0, 1) * (1.0 - travel)
	}
	sourceScale := localOptionalPrecipitationScale(frontalSourceScale, i)
	corridor := transportCorridorWeight(travel)

	available := 0.30*localMarine + 0.48*directUpwind + 0.22*secondHop
	added := available *
		transportScale *
		(0.05 + 0.06*corridor + 0.14*interior + 0.06*transportScale) *
		(0.35 + 0.65*sourceScale) *
		(1.0 - 0.42*coastalImmediate)
	return Clamp(added, 0, 0.40*available)
}

func splitCondensationReservoirsWithFrontal(
	condensedTotal float64,
	marineQ float64,
	landQ float64,
	frontalQ float64,
	oceanFetch float64,
	onshore float64,
	landTravel float64,
	landInterior float64,
	convective float64,
	frontalScale float64,
) (marineCondensed, landCondensed, frontalCondensed float64) {
	total := marineQ + landQ + frontalQ
	if condensedTotal <= 0 || total <= 1e-9 {
		return 0, 0, 0
	}

	coastalImmediate := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * (1.0 - Clamp(landTravel, 0, 1))
	nearInlandCorridor := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * transportCorridorWeight(landTravel)
	deepInteriorCarry := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * Clamp(landTravel, 0, 1)
	continentality := Clamp(landInterior, 0, 1)
	frontalBias := Clamp(frontalScale, 0, 2.5)

	marineWeight := marineQ * (1.0 + 1.15*coastalImmediate + 0.55*nearInlandCorridor + 0.15*deepInteriorCarry)
	landWeight := landQ * (1.0 + 0.55*continentality + 0.60*convective)
	frontalWeight := frontalQ * (1.0 + 0.45*nearInlandCorridor + 0.68*deepInteriorCarry + 0.50*continentality + 0.60*frontalBias)

	weightTotal := marineWeight + landWeight + frontalWeight
	if weightTotal <= 1e-9 {
		return 0, 0, 0
	}
	marineCondensed = condensedTotal * marineWeight / weightTotal
	landCondensed = condensedTotal * landWeight / weightTotal
	frontalCondensed = condensedTotal - marineCondensed - landCondensed

	if marineCondensed > marineQ {
		marineCondensed = marineQ
	}
	if landCondensed > landQ {
		landCondensed = landQ
	}
	if frontalCondensed > frontalQ {
		frontalCondensed = frontalQ
	}

	remaining := condensedTotal - (marineCondensed + landCondensed + frontalCondensed)
	if remaining > 1e-9 {
		if frontalQ-frontalCondensed >= marineQ-marineCondensed && frontalQ-frontalCondensed >= landQ-landCondensed {
			extra := minFloat(remaining, frontalQ-frontalCondensed)
			frontalCondensed += extra
			remaining -= extra
		}
		if remaining > 1e-9 && marineQ-marineCondensed >= landQ-landCondensed {
			extra := minFloat(remaining, marineQ-marineCondensed)
			marineCondensed += extra
			remaining -= extra
		}
		if remaining > 1e-9 {
			landCondensed += minFloat(remaining, landQ-landCondensed)
		}
	}
	return marineCondensed, landCondensed, frontalCondensed
}

func splitRetainedReservoirsWithFrontal(
	retainedTotal float64,
	marineRemaining float64,
	landRemaining float64,
	frontalRemaining float64,
	oceanFetch float64,
	onshore float64,
	landTravel float64,
	landInterior float64,
	frontalRetentionScale float64,
) (marineRetained, landRetained, frontalRetained float64) {
	total := marineRemaining + landRemaining + frontalRemaining
	if retainedTotal <= 0 || total <= 1e-9 {
		return 0, 0, 0
	}

	coastalImmediate := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * (1.0 - Clamp(landTravel, 0, 1))
	nearInlandCorridor := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * transportCorridorWeight(landTravel)
	deepInteriorCarry := Clamp(oceanFetch, 0, 1) * Clamp(onshore, 0, 1) * Clamp(landTravel, 0, 1)
	continentality := Clamp(landInterior, 0, 1)
	frontalBias := Clamp(frontalRetentionScale, 0, 2.5)

	marineWeight := marineRemaining * (1.0 + 0.85*coastalImmediate + 0.55*nearInlandCorridor + 0.15*deepInteriorCarry)
	landWeight := landRemaining * (1.0 + 0.60*continentality + 0.20*deepInteriorCarry)
	frontalWeight := frontalRemaining * (1.0 + 0.55*nearInlandCorridor + 0.82*deepInteriorCarry + 0.60*continentality + 0.70*frontalBias)

	weightTotal := marineWeight + landWeight + frontalWeight
	if weightTotal <= 1e-9 {
		return 0, 0, 0
	}
	marineRetained = retainedTotal * marineWeight / weightTotal
	landRetained = retainedTotal * landWeight / weightTotal
	frontalRetained = retainedTotal - marineRetained - landRetained

	if marineRetained > marineRemaining {
		marineRetained = marineRemaining
	}
	if landRetained > landRemaining {
		landRetained = landRemaining
	}
	if frontalRetained > frontalRemaining {
		frontalRetained = frontalRemaining
	}

	remaining := retainedTotal - (marineRetained + landRetained + frontalRetained)
	if remaining > 1e-9 {
		if frontalRemaining-frontalRetained >= marineRemaining-marineRetained && frontalRemaining-frontalRetained >= landRemaining-landRetained {
			extra := minFloat(remaining, frontalRemaining-frontalRetained)
			frontalRetained += extra
			remaining -= extra
		}
		if remaining > 1e-9 && marineRemaining-marineRetained >= landRemaining-landRetained {
			extra := minFloat(remaining, marineRemaining-marineRetained)
			marineRetained += extra
			remaining -= extra
		}
		if remaining > 1e-9 {
			landRetained += minFloat(remaining, landRemaining-landRetained)
		}
	}
	return marineRetained, landRetained, frontalRetained
}

func applyFrontalLandDiffusion(
	frontal []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	oceanFetch []float64,
	landTravel []float64,
	landInterior []float64,
) {
	if len(frontal) == 0 {
		return
	}
	current := append([]float64(nil), frontal...)
	for iter := 0; iter < frontalLandDiffusionIterations; iter++ {
		next := append([]float64(nil), current...)
		for i := range current {
			if i >= len(elevation) || elevation[i] < seaLevel {
				continue
			}
			travel := 0.0
			if i < len(landTravel) {
				travel = Clamp(landTravel[i], 0, 1)
			}
			interior := 0.0
			if i < len(landInterior) {
				interior = Clamp(landInterior[i], 0, 1)
			}
			coastalImmediate := 0.0
			if i < len(oceanFetch) {
				coastalImmediate = Clamp(oceanFetch[i], 0, 1) * (1.0 - travel)
			}
			absLat := math.Abs(getLatitudeDeg(vertices[i]))
			midlat := smoothRamp(20.0, 38.0, absLat) * (1.0 - smoothRamp(63.0, 80.0, absLat))
			blend := frontalLandDiffusionBlend *
				(0.20 + 0.80*(transportCorridorWeight(travel)+0.6*interior)) *
				(0.45 + 0.55*midlat) *
				(1.0 - 0.60*coastalImmediate)
			if blend <= 1e-6 {
				continue
			}
			sum := 0.0
			weightSum := 0.0
			for _, k := range adj.GetNeighbors(i) {
				if k < 0 || k >= len(current) || k >= len(elevation) || elevation[k] < seaLevel {
					continue
				}
				weight := frontalDiffusionNeighborWeight(i, k, vertices, wind)
				sum += current[k] * weight
				weightSum += weight
			}
			if weightSum <= 1e-9 {
				continue
			}
			neighborMean := sum / weightSum
			next[i] = Clamp((1.0-blend)*current[i]+blend*neighborMean, 0, 10)
		}
		current = next
	}
	copy(frontal, current)
}

func applyFrontalStormTransport(
	frontal []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	oceanFetch []float64,
	landTravel []float64,
	landInterior []float64,
	frontalSourceScale []float64,
	frontalRetentionScale []float64,
	frontalTransportScale []float64,
) {
	if len(frontal) == 0 {
		return
	}
	current := append([]float64(nil), frontal...)
	for iter := 0; iter < frontalStormTransportIterations; iter++ {
		next := append([]float64(nil), current...)
		for i := range current {
			if i >= len(elevation) || elevation[i] < seaLevel {
				continue
			}
			directUpwind, ok := frontalUpwindMean(i, current, vertices, elevation, seaLevel, adj, wind)
			if !ok {
				continue
			}
			secondHop := frontalTwoHopUpwindMean(i, current, vertices, elevation, seaLevel, adj, wind)
			lateral := frontalCrossWindMean(i, current, vertices, elevation, seaLevel, adj, wind)

			travel := 0.0
			if i < len(landTravel) {
				travel = Clamp(landTravel[i], 0, 1)
			}
			interior := 0.0
			if i < len(landInterior) {
				interior = Clamp(landInterior[i], 0, 1)
			}
			coastalImmediate := 0.0
			if i < len(oceanFetch) {
				coastalImmediate = Clamp(oceanFetch[i], 0, 1) * (1.0 - travel)
			}
			sourceScale := localOptionalPrecipitationScale(frontalSourceScale, i)
			retentionScale := localOptionalPrecipitationScale(frontalRetentionScale, i)
			transportScale := localOptionalPrecipitationScale(frontalTransportScale, i)
			regime := marineDiffusionRegimeFactor(i, vertices, wind)
			corridor := transportCorridorWeight(travel)

			propagated := 0.56*directUpwind + 0.29*secondHop + 0.15*lateral
			blend := frontalStormTransportBlend *
				regime *
				(0.20 + 0.45*corridor + 0.35*interior) *
				(0.12 + 0.22*sourceScale + 0.22*retentionScale + 0.44*transportScale) *
				(0.35 + 0.65*travel) *
				(1.0 - 0.55*coastalImmediate)
			blend = Clamp(blend, 0, 0.46)
			if blend <= 1e-6 {
				continue
			}

			target := propagated
			if transportScale > 1e-6 {
				supportFloor := directUpwind * transportScale * (0.14 + 0.30*corridor + 0.28*interior)
				if supportFloor > target {
					target = supportFloor
				}
			}
			if target < current[i] {
				target = current[i]*(1.0-0.35*blend) + target*(0.35*blend)
			}
			next[i] = Clamp(current[i]+blend*(target-current[i]), 0, 10)
		}
		current = next
	}
	copy(frontal, current)
}

func frontalDiffusionNeighborWeight(i int, k int, vertices []Vector3D, wind []Vector3D) float64 {
	if i < 0 || i >= len(vertices) || i >= len(wind) || k < 0 || k >= len(vertices) {
		return 1.0
	}
	speed := Length(wind[i])
	if speed < 1e-9 {
		return 1.0
	}
	windDir := Scale(wind[i], 1.0/speed)
	toNeighbor := Normalize(Sub(vertices[k], vertices[i]))
	alignment := math.Abs(Dot(windDir, toNeighbor))
	return 0.72 + 0.28*alignment
}

func frontalUpwindMean(
	i int,
	field []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
) (float64, bool) {
	localMean, localOK := localUpwindMean(
		i,
		field,
		vertices,
		adj,
		wind,
		0.02,
		func(idx int) bool {
			return idx >= 0 && idx < len(elevation) && elevation[idx] >= seaLevel
		},
	)
	footprintMean, footprintOK := upwindFootprintMean(
		i,
		field,
		vertices,
		adj,
		wind,
		2,
		0.02,
		func(idx int) bool {
			return idx >= 0 && idx < len(elevation) && elevation[idx] >= seaLevel
		},
	)
	if !localOK {
		return footprintMean, footprintOK
	}
	if !footprintOK {
		return localMean, true
	}
	return 0.40*localMean + 0.60*footprintMean, true
}

func frontalTwoHopUpwindMean(
	i int,
	field []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
) float64 {
	localMean, localOK := localUpwindMean(
		i,
		field,
		vertices,
		adj,
		wind,
		0.02,
		func(idx int) bool {
			return idx >= 0 && idx < len(elevation) && elevation[idx] >= seaLevel
		},
	)
	mean, ok := upwindFootprintMean(
		i,
		field,
		vertices,
		adj,
		wind,
		4,
		0.02,
		func(idx int) bool {
			return idx >= 0 && idx < len(elevation) && elevation[idx] >= seaLevel
		},
	)
	if !ok {
		if localOK {
			return localMean
		}
		return 0
	}
	if localOK {
		return 0.25*localMean + 0.75*mean
	}
	return mean
}

func frontalCrossWindMean(
	i int,
	field []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
) float64 {
	if i < 0 || i >= len(vertices) || i >= len(wind) {
		return 0
	}
	speed := Length(wind[i])
	if speed < 1e-9 {
		return 0
	}
	windDir := Scale(wind[i], 1.0/speed)
	sum := 0.0
	weightSum := 0.0
	for _, k := range adj.GetNeighbors(i) {
		if k < 0 || k >= len(field) || k >= len(elevation) || elevation[k] < seaLevel {
			continue
		}
		toNeighbor := Normalize(Sub(vertices[k], vertices[i]))
		cross := 1.0 - math.Abs(Dot(windDir, toNeighbor))
		weight := 0.20 + 0.80*cross*cross
		sum += field[k] * weight
		weightSum += weight
	}
	if weightSum <= 1e-9 {
		return 0
	}
	return sum / weightSum
}
