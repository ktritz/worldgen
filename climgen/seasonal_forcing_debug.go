package climgen

type SeasonalForcingDebug struct {
	RawConvergence     float64
	BaseConvergence    float64
	ClimateConvergence float64
	RawDivergence      float64
	WindSpeed          float64
	SpeedScale         float64
	NeighborCount      int
	DominantNeighbor   float64
	FitResidual        float64
	GeometrySpanRatio  float64
	FrontalExposure    float64
	StormMoisture      float64
	StormBandSupport   float64
	TropicalPlacement  float64
	PersistentWet      float64
	ITCZCrossing       float64
	DryPocket          float64
	TropicalMaritime   float64
}

func ComputeSeasonalForcingDebug(
	idx int,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	solar SolarSettings,
	temperature []float64,
	annualMeanTemperature []float64,
	landInterior []float64,
) SeasonalForcingDebug {
	debug := SeasonalForcingDebug{}
	if idx < 0 || idx >= len(vertices) {
		return debug
	}
	rawDiag := computeWindConvergenceDiagnostic(idx, vertices, adj, wind)
	rawConvergence := rawDiag.Convergence
	climateConvergence := rawConvergence
	climateField := computeClimateConvergenceField(vertices, adj, wind)
	if idx < len(climateField) {
		climateConvergence = climateField[idx]
	}

	frontalExposureField := computeSeasonalFrontalExposureField(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		solar,
	)
	stormMoistureField := computeSeasonalStormMoisturePotentialField(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		solar,
		landInterior,
		frontalExposureField,
	)
	stormBandSupportField := computeSeasonalStormBandSupportField(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		solar,
		landInterior,
	)
	tropicalRegimes := computeSeasonalTropicalRegimeFields(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		solar,
		temperature,
		annualMeanTemperature,
		landInterior,
	)

	debug.RawConvergence = rawConvergence
	debug.BaseConvergence = rawDiag.BaseConvergence
	debug.ClimateConvergence = climateConvergence
	debug.RawDivergence = rawDiag.RawDivergence
	debug.WindSpeed = rawDiag.WindSpeed
	debug.SpeedScale = rawDiag.SpeedScale
	debug.NeighborCount = rawDiag.NeighborCount
	debug.DominantNeighbor = rawDiag.DominantContribution
	debug.FitResidual = rawDiag.FitResidual
	debug.GeometrySpanRatio = rawDiag.GeometrySpanRatio
	if idx < len(frontalExposureField) {
		debug.FrontalExposure = frontalExposureField[idx]
	}
	if idx < len(stormMoistureField) {
		debug.StormMoisture = stormMoistureField[idx]
	}
	if idx < len(stormBandSupportField) {
		debug.StormBandSupport = stormBandSupportField[idx]
	}
	if idx < len(tropicalRegimes.Placement) {
		debug.TropicalPlacement = tropicalRegimes.Placement[idx]
	}
	if idx < len(tropicalRegimes.PersistentWet) {
		debug.PersistentWet = tropicalRegimes.PersistentWet[idx]
	}
	if idx < len(tropicalRegimes.ITCZCrossing) {
		debug.ITCZCrossing = tropicalRegimes.ITCZCrossing[idx]
	}
	if idx < len(tropicalRegimes.DryPocket) {
		debug.DryPocket = tropicalRegimes.DryPocket[idx]
	}
	if idx < len(tropicalRegimes.MaritimeAccess) {
		debug.TropicalMaritime = tropicalRegimes.MaritimeAccess[idx]
	}
	return debug
}
