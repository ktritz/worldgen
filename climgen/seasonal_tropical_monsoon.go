package climgen

import "math"

const (
	seasonalDynamicTropicalSupportBlend = 0.44
	seasonalDynamicTropicalSupportIters = 5
)

type seasonalTropicalRegimeFields struct {
	Placement      []float64
	PersistentWet  []float64
	ITCZCrossing   []float64
	DryPocket      []float64
	MaritimeAccess []float64
}

// computeSeasonalTropicalMoisturePlacementField builds a broad low-frequency
// summer-tropical moisture-placement field over land. It is meant to guide
// where monsoon/tropical wet seasons land, especially just inland of coasts and
// along summer-heated tropical interiors, without turning every tropical coast
// into a uniformly wet rim.
func computeSeasonalTropicalMoisturePlacementField(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	solar SolarSettings,
	temperature []float64,
	annualMeanTemperature []float64,
	landInterior []float64,
) []float64 {
	return computeSeasonalTropicalRegimeFields(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		solar,
		temperature,
		annualMeanTemperature,
		landInterior,
	).Placement
}

func computeSeasonalTropicalRegimeFields(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	solar SolarSettings,
	temperature []float64,
	annualMeanTemperature []float64,
	landInterior []float64,
) seasonalTropicalRegimeFields {
	rawPlacement := make([]float64, len(vertices))
	rawPersistent := make([]float64, len(vertices))
	rawDryPocket := make([]float64, len(vertices))
	rawMaritime := make([]float64, len(vertices))
	declinationDeg := -solar.AxialTilt * math.Cos(2.0*math.Pi*solar.SeasonPhase)
	summerHemisphere := 0.0
	switch {
	case declinationDeg > 0.1:
		summerHemisphere = 1.0
	case declinationDeg < -0.1:
		summerHemisphere = -1.0
	default:
		return seasonalTropicalRegimeFields{
			Placement:      rawPlacement,
			PersistentWet:  rawPersistent,
			ITCZCrossing:   rawPersistent,
			DryPocket:      rawDryPocket,
			MaritimeAccess: rawMaritime,
		}
	}

	maxShiftDeg := math.Abs(defaultThermalEquatorShiftScale * solar.AxialTilt)
	localShift := computeSeasonalTropicalConvergenceLatitudeField(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		solar,
		temperature,
		annualMeanTemperature,
	)
	parent, strength := computeStrongestUpwindGraph(vertices, adj, wind)
	upwindLandSteps := computeUpwindLandStepCounts(parent, strength, elevation, seaLevelThreshold, precipInlandTransportSteps+2)
	rawCrossing := make([]float64, len(vertices))

	for i, v := range vertices {
		if i >= len(elevation) || elevation[i] < seaLevelThreshold {
			continue
		}
		latDeg := getLatitudeDeg(v)
		absLat := math.Abs(latDeg)
		if absLat > seasonalDynamicMonsoonMaxLatDeg+6.0 {
			continue
		}

		cellShift := SeasonalThermalEquatorShiftDeg(solar)
		if i < len(localShift) {
			cellShift = localShift[i]
		}
		climateLat := latDeg - cellShift
		absClimateLat := math.Abs(climateLat)
		tropicalCore := 1.0 - smoothRamp(4.0, 24.0, absClimateLat)
		if tropicalCore <= 0 {
			continue
		}

		interior := 0.0
		if i < len(landInterior) {
			interior = Clamp(landInterior[i], 0, 1)
		}
		onshore := coastalOnshoreScore(i, vertices, elevation, seaLevelThreshold, adj, wind)
		oceanFetch := computeUpwindOceanFetch(i, vertices, elevation, seaLevelThreshold, adj, wind, precipFetchMaxSteps)
		footprintSupport := computeUpwindOceanFootprintSupport(
			i,
			vertices,
			elevation,
			seaLevelThreshold,
			adj,
			wind,
			precipInlandTransportSteps+4,
		)
		neighborOcean := computeNeighborOceanFraction(i, elevation, seaLevelThreshold, adj)
		travel := 0.0
		if i < len(upwindLandSteps) && upwindLandSteps[i] >= 0 {
			travel = Clamp(float64(upwindLandSteps[i])/float64(precipInlandTransportSteps+2), 0, 1)
		}
		corridor := transportCorridorWeight(travel)
		coastalImmediate := Clamp(onshore*(1.0-travel), 0, 1)

		landHeating := 0.0
		if i < len(temperature) && i < len(annualMeanTemperature) {
			landHeating = Clamp((temperature[i]-annualMeanTemperature[i])/16.0, -1.0, 1.0)
		}

		summerSide := 0.0
		crossEquatorial := 0.0
		switch {
		case latDeg*summerHemisphere > 0:
			summerSide = 1.0
		case absClimateLat <= 10.0:
			crossEquatorial = (1.0 - smoothRamp(2.0, 10.0, absClimateLat)) * 0.65
		default:
			continue
		}

		monsoonBelt := 1.0 - smoothRamp(6.0, seasonalDynamicMonsoonMaxLatDeg, absLat)
		geometricMaritime := Clamp(
			0.45*oceanFetch+
				0.35*footprintSupport+
				0.20*neighborOcean,
			0,
			1,
		)
		maritimeAccess := Clamp(
			0.48*onshore+
				0.30*geometricMaritime+
				0.22*geometricMaritime*(0.20+0.80*corridor),
			0,
			1,
		)
		rawMaritime[i] = maritimeAccess
		inlandCarry := tropicalCore *
			(0.20 + 0.80*maritimeAccess) *
			(0.30 + 0.70*corridor) *
			(0.20 + 0.80*(0.30+0.70*interior))
		coastalSeed := onshore * (1.0 - 0.65*travel) * (0.25 + 0.75*geometricMaritime) * (0.40 + 0.60*tropicalCore)
		heatedInterior := tropicalCore * math.Max(0, landHeating) * (0.20 + 0.80*interior) * (0.20 + 0.80*maritimeAccess)
		persistentEquatorial := (1.0 - smoothRamp(3.0, 14.0, absClimateLat)) *
			(0.15 + 0.85*maritimeAccess) *
			(0.35 + 0.65*(0.25+0.75*interior))
		itczWidth := math.Max(maxShiftDeg, math.Abs(cellShift)) + 6.0
		itczCrossing := (1.0 - smoothRamp(2.0, itczWidth, absLat)) *
			(0.15 + 0.85*maritimeAccess) *
			(0.30 + 0.70*(0.20+0.80*interior))
		dryPocket := tropicalCore *
			(0.35 + 0.65*interior) *
			(1.0 - 0.75*maritimeAccess) *
			(0.45 + 0.55*smoothRamp(8.0, 24.0, absLat))
		placementGate := 0.15 + 0.85*maritimeAccess

		rawPlacement[i] = Clamp(
			(0.18*tropicalCore*(summerSide+crossEquatorial)+
				0.20*coastalSeed*(0.20+0.80*summerSide)+
				0.32*inlandCarry*(0.30+0.70*(summerSide+0.7*crossEquatorial))+
				0.20*heatedInterior*(0.30+0.70*summerSide)+
				0.10*monsoonBelt*(0.30+0.70*(summerSide+0.5*crossEquatorial)))*
				placementGate+
				0.08*geometricMaritime*(0.25+0.75*monsoonBelt)*(0.20+0.80*(summerSide+0.5*crossEquatorial))-
				0.18*coastalImmediate*(0.80+0.20*tropicalCore),
			0,
			1,
		)
		rawPersistent[i] = Clamp(
			0.38*persistentEquatorial+
				0.42*itczCrossing+
				0.20*tropicalCore*(0.25+0.75*crossEquatorial)+
				0.12*heatedInterior*(0.30+0.70*(0.25+0.75*interior))-
				0.20*dryPocket,
			0,
			1,
		)
		rawCrossing[i] = Clamp(
			0.75*itczCrossing+
				0.15*persistentEquatorial-
				0.15*dryPocket,
			0,
			1,
		)
		rawDryPocket[i] = Clamp(
			dryPocket-
				0.25*persistentEquatorial-
				0.20*itczCrossing-
				0.15*(summerSide+crossEquatorial)*corridor,
			0,
			1,
		)
	}

	smoothedPlacement := smoothSeasonalLandField(rawPlacement, elevation, seaLevelThreshold, adj, seasonalDynamicTropicalSupportBlend, seasonalDynamicTropicalSupportIters)
	smoothedPersistent := smoothSeasonalLandField(rawPersistent, elevation, seaLevelThreshold, adj, 0.36, 4)
	smoothedCrossing := smoothSeasonalLandField(rawCrossing, elevation, seaLevelThreshold, adj, 0.34, 4)
	smoothedDry := smoothSeasonalLandField(rawDryPocket, elevation, seaLevelThreshold, adj, 0.28, 3)
	smoothedMaritime := smoothSeasonalLandField(rawMaritime, elevation, seaLevelThreshold, adj, 0.22, 2)
	placementMemory := computeSeasonalStormMemoryField(
		smoothedPlacement,
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		landInterior,
	)
	persistentMemory := computeSeasonalStormMemoryField(
		smoothedPersistent,
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		landInterior,
	)
	crossingMemory := computeSeasonalStormMemoryField(
		smoothedCrossing,
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		landInterior,
	)
	outPlacement := make([]float64, len(rawPlacement))
	outPersistent := make([]float64, len(rawPlacement))
	outCrossing := make([]float64, len(rawPlacement))
	outDry := make([]float64, len(rawPlacement))
	outMaritime := make([]float64, len(rawPlacement))
	for i := range rawPlacement {
		interior := 0.0
		if i < len(landInterior) {
			interior = Clamp(landInterior[i], 0, 1)
		}
		outPlacement[i] = Clamp(
			0.16*rawPlacement[i]+
				0.48*smoothedPlacement[i]*(0.30+0.70*(0.25+0.75*interior))+
				0.58*placementMemory[i]*(0.25+0.75*(0.20+0.80*interior)),
			0,
			1,
		)
		outPersistent[i] = Clamp(
			0.30*rawPersistent[i]+
				0.44*smoothedPersistent[i]*(0.25+0.75*(0.25+0.75*interior))+
				0.42*persistentMemory[i]*(0.20+0.80*(0.20+0.80*interior)),
			0,
			1,
		)
		outCrossing[i] = Clamp(
			0.34*rawCrossing[i]+
				0.38*smoothedCrossing[i]*(0.25+0.75*(0.20+0.80*interior))+
				0.34*crossingMemory[i]*(0.20+0.80*(0.20+0.80*interior)),
			0,
			1,
		)
		outDry[i] = Clamp(
			0.46*rawDryPocket[i]+
				0.42*smoothedDry[i]*(0.25+0.75*interior),
			0,
			1,
		)
		outMaritime[i] = Clamp(
			0.62*rawMaritime[i]+
				0.38*smoothedMaritime[i],
			0,
			1,
		)
	}
	return seasonalTropicalRegimeFields{
		Placement:      outPlacement,
		PersistentWet:  outPersistent,
		ITCZCrossing:   outCrossing,
		DryPocket:      outDry,
		MaritimeAccess: outMaritime,
	}
}
