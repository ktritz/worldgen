package climgen

import "math"

const (
	precipBaseCellSizeKm               = 70.0
	precipMinIterations                = 18
	precipMaxIterations                = 96
	precipOceanCondensationFraction    = 0.08
	precipLandSupersatFraction         = 0.68
	precipOrographicCondenseFraction   = 0.15
	precipConvergenceCondenseFraction  = 0.20
	precipOceanRetentionFraction       = 0.97
	precipLandRetentionFloor           = 0.38
	precipResidualDryPrecipCm          = 2.0
	precipUpliftScaleMeters            = 1300.0
	precipLandSourceFraction           = 0.020
	precipLandRecyclingFraction        = 0.12
	precipTropicalConvergenceMinLatDeg = 3.0
	precipTropicalConvergenceMaxLatDeg = 18.0
	precipFetchMaxSteps                = 10
	precipUpliftTraceSteps             = 4
	precipLandBaseCondenseFraction     = 0.08
	precipLandConvectiveFraction       = 0.42
)

func computePrecipitationBudget(
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	temperature []float64,
	settings PrecipitationSettings,
) *PrecipitationResult {
	n := len(vertices)
	result := &PrecipitationResult{
		Precipitation:        make([]float64, n),
		Moisture:             make([]float64, n),
		MarineMoisture:       make([]float64, n),
		LandMoisture:         make([]float64, n),
		FrontalMoisture:      make([]float64, n),
		Rainfall:             make([]float64, n),
		Snowfall:             make([]float64, n),
		MarinePrecipitation:  make([]float64, n),
		LandPrecipitation:    make([]float64, n),
		FrontalPrecipitation: make([]float64, n),
		Debug:                newPrecipitationDebugFields(n),
	}
	if wind == nil {
		return result
	}

	avgCellSizeKm := estimateClimateCellSizeKm(n)
	maxIterations := scaledPrecipIterations(avgCellSizeKm)
	rainfallFractionPerCell := settings.RainfallFraction * avgCellSizeKm
	transportSteps := resolutionAdjustedPrecipSteps(precipInlandTransportSteps, n)
	fetchSteps := resolutionAdjustedPrecipSteps(precipFetchMaxSteps, n)
	footprintSteps := resolutionAdjustedPrecipSteps(precipInlandTransportSteps+4, n)

	isOcean := make([]bool, n)
	for i := range elevation {
		isOcean[i] = elevation[i] < seaLevel
	}

	moistureCap := computeMoistureCapacity(temperature)
	oceanSource := computeOceanMoistureSource(isOcean, temperature, settings)
	oceanAtmosphere, oceanDiag := computeOceanAtmosphericMoisture(
		vertices,
		elevation,
		seaLevel,
		adj,
		wind,
		oceanSource,
		moistureCap,
		rainfallFractionPerCell,
		maxIterations,
	)
	landSource := computeLandMoistureSource(vertices, elevation, seaLevel, temperature, settings)
	landInterior := ComputeSurfaceInteriorFraction(elevation, seaLevel, adj, 1800.0, true)
	uplift := make([]float64, n)
	convergence := computeClimateConvergenceField(vertices, adj, wind)
	oceanFetch := make([]float64, n)
	coastalOnshore := make([]float64, n)
	landTravel := make([]float64, n)
	upwindParent, upwindStrength := computeStrongestUpwindGraph(vertices, adj, wind)
	upwindLandSteps := computeUpwindLandStepCounts(upwindParent, upwindStrength, elevation, seaLevel, transportSteps)
	for i := range vertices {
		if i < len(oceanAtmosphere) {
			result.Debug.OceanAtmosphere[i] = oceanAtmosphere[i]
		}
		upliftDiag := computeOrographicLiftDiagnostic(i, vertices, elevation, adj, wind)
		uplift[i] = upliftDiag.Lift
		oceanFetch[i] = computeUpwindOceanFetch(i, vertices, elevation, seaLevel, adj, wind, fetchSteps)
		coastalOnshore[i] = coastalOnshoreScore(i, vertices, elevation, seaLevel, adj, wind)
		if upwindLandSteps[i] >= 0 {
			landTravel[i] = Clamp(float64(upwindLandSteps[i])/float64(transportSteps), 0, 1)
		}
		parent := -1
		if i < len(upwindParent) {
			parent = upwindParent[i]
		}
		result.Debug.OrographicLift[i] = uplift[i]
		result.Debug.OrographicLocalRise[i] = upliftDiag.LocalRiseMeters
		result.Debug.OrographicFootprint[i] = upliftDiag.FootprintRiseMeters
		result.Debug.OrographicBarrier[i] = upliftDiag.BarrierPersistence
		result.Debug.OrographicWindFactor[i] = upliftDiag.WindFactor
		result.Debug.Convergence[i] = convergence[i]
		result.Debug.OceanFetch[i] = oceanFetch[i]
		result.Debug.CoastalOnshore[i] = coastalOnshore[i]
		result.Debug.FootprintOceanSupport[i] = computeUpwindOceanFootprintSupport(
			i,
			vertices,
			elevation,
			seaLevel,
			adj,
			wind,
			footprintSteps,
		)
		result.Debug.NeighborOceanFraction[i] = computeNeighborOceanFraction(i, elevation, seaLevel, adj)
		result.Debug.OceanDownwindLand[i] = computeDownwindLandExposure(i, vertices, elevation, seaLevel, adj, wind)
		result.Debug.UpwindParent[i] = float64(parent)
		if i < len(upwindStrength) {
			result.Debug.UpwindParentStrength[i] = upwindStrength[i]
		}
		result.Debug.LandTravel[i] = landTravel[i]
		result.Debug.LandInterior[i] = landInterior[i]
		result.Debug.MoistureCapacity[i] = moistureCap[i]
		result.Debug.LandSource[i] = landSource[i]
		tempC := 12.0
		if i >= 0 && i < len(temperature) {
			tempC = temperature[i] - 273.15
		}
		coastalImmediate := Clamp(oceanFetch[i], 0, 1) * Clamp(coastalOnshore[i], 0, 1) * (1.0 - Clamp(landTravel[i], 0, 1))
		result.Debug.MarineEntryScale[i] = marineLandfallEntryScale(coastalImmediate, uplift[i], landTravel[i], tempC)
	}
	marineSweepOrder := buildMarineSweepOrder(upwindLandSteps, elevation, seaLevel)
	marineTransported, marineDiag := computeMarineSweepTransport(
		vertices,
		elevation,
		seaLevel,
		adj,
		wind,
		upwindParent,
		upwindStrength,
		marineSweepOrder,
		oceanAtmosphere,
		oceanDiag,
		temperature,
		moistureCap,
		uplift,
		convergence,
		oceanFetch,
		coastalOnshore,
		landTravel,
		landInterior,
		settings,
		rainfallFractionPerCell,
	)
	effectiveFetch, effectiveOnshore := computeEffectiveMaritimeAccess(
		vertices,
		elevation,
		seaLevel,
		adj,
		wind,
		marineTransported,
		oceanFetch,
		coastalOnshore,
		landTravel,
		landInterior,
	)
	for i := range vertices {
		neighborOceanFraction := result.Debug.NeighborOceanFraction[i]
		diag := deriveEffectiveMaritimeAccessDiagnostic(
			oceanFetch[i],
			coastalOnshore[i],
			result.Debug.FootprintOceanSupport[i],
			neighborOceanFraction,
			landTravel[i],
			landInterior[i],
			marineTransported[i],
		)
		result.Debug.EffectiveFetch[i] = effectiveFetch[i]
		result.Debug.EffectiveOnshore[i] = effectiveOnshore[i]
		result.Debug.MaritimeSignal[i] = diag.MarineSignal
		result.Debug.MaritimeGeomSupport[i] = diag.GeometricSupport
		result.Debug.MarineDonor[i] = marineDiag.DonorIndex[i]
		result.Debug.MarineDonorStrength[i] = marineDiag.DonorStrength[i]
		result.Debug.MarineDonorOutgoing[i] = marineDiag.DonorOutgoing[i]
		result.Debug.MarineDonorOceanAtm[i] = marineDiag.DonorOceanAtm[i]
		result.Debug.MarineDonorDownwind[i] = marineDiag.DonorDownwind[i]
		result.Debug.MarineRootSource[i] = marineDiag.RootIndex[i]
		result.Debug.MarineRootStrength[i] = marineDiag.RootStrength[i]
		result.Debug.MarineRootOceanAtm[i] = marineDiag.RootOceanAtm[i]
		result.Debug.MarineRootDownwind[i] = marineDiag.RootDownwind[i]
		result.Debug.MarineRootOceanSource[i] = marineDiag.RootSource[i]
		result.Debug.MarineRootRetention[i] = marineDiag.RootRetention[i]
		result.Debug.MarineRootPathSteps[i] = marineDiag.RootSteps[i]
	}
	landTransported := make([]float64, n)
	frontalTransported := make([]float64, n)

	for iter := 0; iter < maxIterations; iter++ {
		nextLand := make([]float64, n)
		nextFrontal := make([]float64, n)
		maxChange := 0.0
		for i := range vertices {
			incomingMarine := marineTransported[i]
			incomingLand := advectedSpecificHumidity(i, vertices, adj, wind, landTransported)
			incomingFrontal := advectedSpecificHumidity(i, vertices, adj, wind, frontalTransported)
			marineQ := incomingMarine
			landQ := incomingLand
			frontalQ := incomingFrontal
			if !isOcean[i] {
				frontalSourceScale := localOptionalPrecipitationScale(settings.FrontalSourceLocalScale, i)
				marineToFrontal := marineQ * precipitationPerStepFraction(computeFrontalMarineCaptureFraction(
					effectiveFetch[i],
					effectiveOnshore[i],
					landTravel[i],
					landInterior[i],
					frontalSourceScale,
				), n)
				marineQ -= marineToFrontal
				frontalQ += marineToFrontal
				frontalQ += computeFrontalStormSource(
					i,
					marineTransported,
					vertices,
					elevation,
					seaLevel,
					adj,
					wind,
					effectiveFetch,
					landTravel,
					landInterior,
					settings.FrontalSourceLocalScale,
					settings.FrontalTransportLocalScale,
				)
				marineToLand := marineQ * precipitationPerStepFraction(marineToLandMixFraction(effectiveFetch[i], effectiveOnshore[i], landTravel[i], landInterior[i]), n)
				marineQ -= marineToLand
				landQ += marineToLand
				landQ += computeTropicalMarineSource(
					i,
					marineTransported,
					vertices,
					elevation,
					seaLevel,
					adj,
					wind,
					effectiveFetch,
					effectiveOnshore,
					landTravel,
					landInterior,
					settings.TropicalSourceLocalScale,
				)
				landQ += landSource[i]
				landQ += computeLandRecyclingSource(
					landTransported[i],
					temperature,
					i,
					landInterior[i],
					localPrecipitationStorage(settings.LandSurfaceStorage, i),
					settings.LandRecyclingScale*localPrecipitationScale(settings.LandRecyclingLocalScale, i),
				)
			}
			if isOcean[i] {
				nextLand[i] = 0
				nextFrontal[i] = 0
			} else {
				tempC := 12.0
				if i >= 0 && i < len(temperature) {
					tempC = temperature[i] - 273.15
				}
				frontalRetentionScale := localOptionalPrecipitationScale(settings.FrontalRetentionLocalScale, i)
				q := marineQ + landQ + frontalQ
				marineShare := 0.0
				if q > 1e-9 {
					marineShare = (marineQ + 0.75*frontalQ) / q
				}
				convective := computeConvectiveCondensationPotential(q, moistureCap[i], tempC, convergence[i], landInterior[i])
				condDiag := computeLandCondensationDiagnostic(
					q,
					moistureCap[i],
					uplift[i],
					convergence[i],
					oceanFetch[i],
					coastalOnshore[i],
					landTravel[i],
					landInterior[i],
					marineShare,
					localPrecipitationScale(settings.CondensationLocalScale, i),
					rainfallFractionPerCell,
					temperature,
					i,
				)
				condensed := condDiag.Condensed
				marineCondensed, landCondensed, frontalCondensed := splitCondensationReservoirsWithFrontal(
					condensed,
					marineQ,
					landQ,
					frontalQ,
					oceanFetch[i],
					coastalOnshore[i],
					landTravel[i],
					landInterior[i],
					convective,
					frontalRetentionScale,
				)
				marineRemaining := math.Max(0, marineQ-marineCondensed)
				landRemaining := math.Max(0, landQ-landCondensed)
				frontalRemaining := math.Max(0, frontalQ-frontalCondensed)
				retained := computeLandRetainedHumidity(
					q,
					condensed,
					moistureCap[i],
					uplift[i],
					oceanFetch[i],
					coastalOnshore[i],
					landTravel[i],
					landInterior[i],
					marineShare,
					localPrecipitationScale(settings.LandRetentionLocalScale, i),
				)
				if retained > q {
					retained = q
				}
				_, nextLand[i], nextFrontal[i] = splitRetainedReservoirsWithFrontal(
					math.Max(0, retained),
					marineRemaining,
					landRemaining,
					frontalRemaining,
					oceanFetch[i],
					coastalOnshore[i],
					landTravel[i],
					landInterior[i],
					frontalRetentionScale,
				)
			}

			change := math.Abs(nextLand[i] - landTransported[i])
			if change > maxChange {
				maxChange = change
			}
		}
		applyFrontalLandDiffusion(
			nextFrontal,
			vertices,
			elevation,
			seaLevel,
			adj,
			wind,
			effectiveFetch,
			landTravel,
			landInterior,
		)
		applyFrontalStormTransport(
			nextFrontal,
			vertices,
			elevation,
			seaLevel,
			adj,
			wind,
			effectiveFetch,
			landTravel,
			landInterior,
			settings.FrontalSourceLocalScale,
			settings.FrontalRetentionLocalScale,
			settings.FrontalTransportLocalScale,
		)
		for i := range nextFrontal {
			frontalChange := math.Abs(nextFrontal[i] - frontalTransported[i])
			if frontalChange > maxChange {
				maxChange = frontalChange
			}
		}
		landTransported = nextLand
		frontalTransported = nextFrontal
		if maxChange < 0.0005 {
			break
		}
	}

	copy(result.MarineMoisture, marineTransported)
	copy(result.LandMoisture, landTransported)
	copy(result.FrontalMoisture, frontalTransported)
	for i := range result.Moisture {
		result.Moisture[i] = marineTransported[i] + landTransported[i] + frontalTransported[i]
	}
	for i := range vertices {
		if isOcean[i] {
			continue
		}
		incomingMarine := marineTransported[i]
		incomingLand := advectedSpecificHumidity(i, vertices, adj, wind, landTransported)
		incomingFrontal := advectedSpecificHumidity(i, vertices, adj, wind, frontalTransported)
		frontalSourceScale := localOptionalPrecipitationScale(settings.FrontalSourceLocalScale, i)
		result.Debug.FrontalSourceScale[i] = frontalSourceScale
		result.Debug.FrontalRetentionScale[i] = localOptionalPrecipitationScale(settings.FrontalRetentionLocalScale, i)
		result.Debug.TropicalSourceScale[i] = localOptionalPrecipitationScale(settings.TropicalSourceLocalScale, i)
		result.Debug.CondensationScale[i] = localPrecipitationScale(settings.CondensationLocalScale, i)
		result.Debug.LandRetentionScale[i] = localPrecipitationScale(settings.LandRetentionLocalScale, i)
		marineToFrontal := incomingMarine * precipitationPerStepFraction(computeFrontalMarineCaptureFraction(
			effectiveFetch[i],
			effectiveOnshore[i],
			landTravel[i],
			landInterior[i],
			frontalSourceScale,
		), n)
		incomingMarine -= marineToFrontal
		incomingFrontal += marineToFrontal
		incomingFrontal += computeFrontalStormSource(
			i,
			marineTransported,
			vertices,
			elevation,
			seaLevel,
			adj,
			wind,
			effectiveFetch,
			landTravel,
			landInterior,
			settings.FrontalSourceLocalScale,
			settings.FrontalTransportLocalScale,
		)
		result.Debug.FrontalSource[i] = incomingFrontal
		marineToLand := incomingMarine * precipitationPerStepFraction(marineToLandMixFraction(effectiveFetch[i], effectiveOnshore[i], landTravel[i], landInterior[i]), n)
		incomingMarine -= marineToLand
		incomingLand += marineToLand
		tropicalSource := computeTropicalMarineSource(
			i,
			marineTransported,
			vertices,
			elevation,
			seaLevel,
			adj,
			wind,
			effectiveFetch,
			effectiveOnshore,
			landTravel,
			landInterior,
			settings.TropicalSourceLocalScale,
		)
		incomingLand += tropicalSource
		result.Debug.TropicalSource[i] = tropicalSource
		incomingLand += landSource[i] + computeLandRecyclingSource(
			landTransported[i],
			temperature,
			i,
			landInterior[i],
			localPrecipitationStorage(settings.LandSurfaceStorage, i),
			settings.LandRecyclingScale*localPrecipitationScale(settings.LandRecyclingLocalScale, i),
		)
		tempC := 12.0
		if i >= 0 && i < len(temperature) {
			tempC = temperature[i] - 273.15
		}
		frontalRetentionScale := localOptionalPrecipitationScale(settings.FrontalRetentionLocalScale, i)
		incoming := incomingMarine + incomingLand + incomingFrontal
		marineShare := 0.0
		if incoming > 1e-9 {
			marineShare = (incomingMarine + 0.75*incomingFrontal) / incoming
		}
		convective := computeConvectiveCondensationPotential(incoming, moistureCap[i], tempC, convergence[i], landInterior[i])
		condDiag := computeLandCondensationDiagnostic(
			incoming,
			moistureCap[i],
			uplift[i],
			convergence[i],
			oceanFetch[i],
			coastalOnshore[i],
			landTravel[i],
			landInterior[i],
			marineShare,
			localPrecipitationScale(settings.CondensationLocalScale, i),
			rainfallFractionPerCell,
			temperature,
			i,
		)
		condensedTotal := condDiag.Condensed
		result.Debug.MarineIncoming[i] = incomingMarine
		result.Debug.LandIncoming[i] = incomingLand
		result.Debug.FrontalIncoming[i] = incomingFrontal
		result.Debug.MarineToLand[i] = marineToLand
		result.Debug.MarineToFrontal[i] = marineToFrontal
		result.Debug.CondensedTotal[i] = condensedTotal
		result.Debug.CondensedBase[i] = condDiag.BaseCondensation
		result.Debug.CondensedSupersat[i] = condDiag.SupersatCondensation
		result.Debug.CondensedSupersatSupport[i] = condDiag.SupersatSupport
		result.Debug.CondensedTropicalCoast[i] = condDiag.TropicalCoastSupport
		result.Debug.CondensedCoastalPenalty[i] = condDiag.CoastalPenalty
		result.Debug.CondensedAscent[i] = condDiag.AscentFraction
		result.Debug.CondensedConvective[i] = condDiag.ConvectivePotential
		result.Debug.CondensedMixing[i] = condDiag.MixingFraction
		result.Debug.CondensedEffCapacity[i] = condDiag.EffectiveCapacity
		result.Debug.CondensedSupersatHum[i] = condDiag.SupersatHumidity
		marineCondensed, landCondensed, frontalCondensed := splitCondensationReservoirsWithFrontal(
			condensedTotal,
			incomingMarine,
			incomingLand,
			incomingFrontal,
			oceanFetch[i],
			coastalOnshore[i],
			landTravel[i],
			landInterior[i],
			convective,
			frontalRetentionScale,
		)
		result.MarinePrecipitation[i] = marineCondensed
		result.LandPrecipitation[i] = landCondensed
		result.FrontalPrecipitation[i] = frontalCondensed
		result.Precipitation[i] = marineCondensed + landCondensed + frontalCondensed
		result.Debug.RetainedHumidity[i] = math.Max(0, incoming-condensedTotal)
		if settings.PrecipitationScale > 0 {
			result.MarinePrecipitation[i] *= settings.PrecipitationScale
			result.LandPrecipitation[i] *= settings.PrecipitationScale
			result.FrontalPrecipitation[i] *= settings.PrecipitationScale
			result.Precipitation[i] *= settings.PrecipitationScale
		}
		if result.Precipitation[i] < precipResidualDryPrecipCm {
			total := result.MarinePrecipitation[i] + result.LandPrecipitation[i] + result.FrontalPrecipitation[i]
			marineShare := reservoirShare(result.MarinePrecipitation[i], total-result.MarinePrecipitation[i])
			landShare := reservoirShare(result.LandPrecipitation[i], result.FrontalPrecipitation[i])
			result.Precipitation[i] = precipResidualDryPrecipCm
			result.MarinePrecipitation[i] = precipResidualDryPrecipCm * marineShare
			remaining := precipResidualDryPrecipCm - result.MarinePrecipitation[i]
			result.LandPrecipitation[i] = remaining * landShare
			result.FrontalPrecipitation[i] = remaining - result.LandPrecipitation[i]
		}
	}

	partitionPrecipitationPhase(result, elevation, seaLevel, temperature)
	return result
}

func advectedSpecificHumidity(
	i int,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	specificHumidity []float64,
) float64 {
	direct := pullMoistureFromUpwind(i, vertices, adj, wind, specificHumidity)
	mean, ok := upwindFootprintMean(
		i,
		specificHumidity,
		vertices,
		adj,
		wind,
		resolutionAdjustedPrecipSteps(3, len(vertices)),
		precipUpwindFootprintMinAlignment,
		nil,
	)
	if !ok {
		return direct
	}
	absLat := math.Abs(getLatitudeDeg(vertices[i]))
	footprintWeight := smoothRamp(18.0, 34.0, absLat)
	return direct*(1.0-footprintWeight) + mean*footprintWeight
}

func computeOceanCondensation(q, capacity, rainfallFractionPerCell float64) float64 {
	if q <= 0 {
		return 0
	}
	supersat := math.Max(0, q-capacity)
	background := q * rainfallFractionPerCell * precipOceanCondensationFraction
	condensed := background + supersat*0.45
	if condensed > q {
		return q
	}
	return condensed
}

func computeConvectiveCondensationPotential(
	q float64,
	capacity float64,
	tempC float64,
	convergence float64,
	landInterior float64,
) float64 {
	if capacity <= 1e-9 || q <= 0 {
		return 0
	}
	warmth := smoothRamp(4.0, 28.0, tempC)
	humidityRatio := Clamp(q/capacity, 0, 1.6)
	humidity := smoothRamp(0.58, 1.02, humidityRatio)
	continentality := 0.45 + 0.55*Clamp(landInterior, 0, 1)
	dynamicLift := 0.40 + 0.60*math.Max(0, convergence)
	return Clamp(warmth*humidity*continentality*dynamicLift, 0, 1)
}
