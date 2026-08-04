package climgen

import "math"

type SeasonalWindDebug struct {
	BaseThermalShiftDeg float64
	LocalShiftDeg       float64
	PressureAnomaly     float64
	PressureWindSpeed   float64
	PressureWindEast    float64
	PressureWindNorth   float64
	SurfaceWeight       float64
	MarineWeight        float64
	BaseSurfaceSpeed    float64
	FinalSurfaceSpeed   float64
	BaseMarineSpeed     float64
	FinalMarineSpeed    float64
	BaseSurfaceEast     float64
	BaseSurfaceNorth    float64
	FinalSurfaceEast    float64
	FinalSurfaceNorth   float64
	BaseSurfaceConv     float64
	FinalSurfaceConv    float64
	BaseSurfaceRawDiv   float64
	FinalSurfaceRawDiv  float64
	CombinedConv        float64
	FrictionConv        float64
	SlopeConv           float64
	OrographicConv      float64
	SmoothedConv        float64
	CombinedRawDiv      float64
	FrictionRawDiv      float64
	SlopeRawDiv         float64
	OrographicRawDiv    float64
	SmoothedRawDiv      float64
	CombinedSpeed       float64
	FrictionSpeed       float64
	SlopeSpeed          float64
	OrographicSpeed     float64
	SmoothedSpeed       float64
}

func ComputeSeasonalWindDebug(
	idx int,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	settings WindSettings,
	solar SolarSettings,
	temperature []float64,
	annualMeanTemperature []float64,
) (SeasonalWindDebug, error) {
	debug := SeasonalWindDebug{}
	if idx < 0 || idx >= len(vertices) {
		return debug, nil
	}

	shifted := settings
	shifted.ApplyVerbose()
	debug.BaseThermalShiftDeg = SeasonalThermalEquatorShiftDeg(solar)
	shifted.Circulation.ThermalEquatorShiftDeg = debug.BaseThermalShiftDeg
	base, err := GenerateWindField(vertices, elevation, seaLevelThreshold, adj, shifted)
	if err != nil {
		return debug, err
	}
	final := ApplySeasonalTropicalWindAdjustment(
		base,
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		solar,
		temperature,
		annualMeanTemperature,
	)

	localShift := computeSeasonalTropicalConvergenceLatitudeField(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		solar,
		temperature,
		annualMeanTemperature,
	)
	if idx < len(localShift) {
		debug.LocalShiftDeg = localShift[idx]
	} else {
		debug.LocalShiftDeg = debug.BaseThermalShiftDeg
	}

	anomaly := computeSeasonalTropicalPressureAnomaly(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		solar,
		temperature,
		annualMeanTemperature,
	)
	if idx < len(anomaly) {
		debug.PressureAnomaly = anomaly[idx]
	}
	pressureWind := ComputePressureGradientWind(
		anomaly,
		vertices,
		adj,
		seasonalTropicalPressureGradientStrength,
	)
	if idx < len(pressureWind) {
		east, north := GetTangentVectors(vertices[idx])
		debug.PressureWindSpeed = Length(pressureWind[idx])
		debug.PressureWindEast = Dot(pressureWind[idx], east)
		debug.PressureWindNorth = Dot(pressureWind[idx], north)
	}

	landInterior := ComputeSurfaceInteriorFraction(elevation, seaLevelThreshold, adj, 2200.0, true)
	absLat := math.Abs(getLatitudeDeg(vertices[idx]))
	tropicalReach := 1.0 - smoothRamp(6.0, 40.0, absLat)
	if tropicalReach > 0 {
		interior := 0.0
		if idx < len(landInterior) {
			interior = Clamp(landInterior[idx], 0, 1)
		}
		debug.SurfaceWeight = seasonalTropicalSurfaceBlend * tropicalReach
		if idx < len(elevation) && elevation[idx] >= seaLevelThreshold {
			debug.SurfaceWeight *= 0.82 + 0.18*(0.25+0.75*interior)
		}
		debug.MarineWeight = seasonalTropicalMarineBlend * tropicalReach
	}

	east, north := GetTangentVectors(vertices[idx])
	if idx < len(base.SurfaceWind) {
		debug.BaseSurfaceSpeed = Length(base.SurfaceWind[idx])
		debug.BaseSurfaceEast = Dot(base.SurfaceWind[idx], east)
		debug.BaseSurfaceNorth = Dot(base.SurfaceWind[idx], north)
	}
	if idx < len(final.SurfaceWind) {
		debug.FinalSurfaceSpeed = Length(final.SurfaceWind[idx])
		debug.FinalSurfaceEast = Dot(final.SurfaceWind[idx], east)
		debug.FinalSurfaceNorth = Dot(final.SurfaceWind[idx], north)
	}
	if idx < len(base.MarineWind) {
		debug.BaseMarineSpeed = Length(base.MarineWind[idx])
	}
	if idx < len(final.MarineWind) {
		debug.FinalMarineSpeed = Length(final.MarineWind[idx])
	}

	baseConv := computeWindConvergenceDiagnostic(idx, vertices, adj, base.SurfaceWind)
	finalConv := computeWindConvergenceDiagnostic(idx, vertices, adj, final.SurfaceWind)
	debug.BaseSurfaceConv = baseConv.Convergence
	debug.FinalSurfaceConv = finalConv.Convergence
	debug.BaseSurfaceRawDiv = baseConv.RawDivergence
	debug.FinalSurfaceRawDiv = finalConv.RawDivergence

	stageDebug := computeWindPipelineDebug(idx, vertices, elevation, seaLevelThreshold, adj, shifted)
	debug.CombinedConv = stageDebug.CombinedConv
	debug.FrictionConv = stageDebug.FrictionConv
	debug.SlopeConv = stageDebug.SlopeConv
	debug.OrographicConv = stageDebug.OrographicConv
	debug.SmoothedConv = stageDebug.SmoothedConv
	debug.CombinedRawDiv = stageDebug.CombinedRawDiv
	debug.FrictionRawDiv = stageDebug.FrictionRawDiv
	debug.SlopeRawDiv = stageDebug.SlopeRawDiv
	debug.OrographicRawDiv = stageDebug.OrographicRawDiv
	debug.SmoothedRawDiv = stageDebug.SmoothedRawDiv
	debug.CombinedSpeed = stageDebug.CombinedSpeed
	debug.FrictionSpeed = stageDebug.FrictionSpeed
	debug.SlopeSpeed = stageDebug.SlopeSpeed
	debug.OrographicSpeed = stageDebug.OrographicSpeed
	debug.SmoothedSpeed = stageDebug.SmoothedSpeed

	return debug, nil
}

type windPipelineDebug struct {
	CombinedConv     float64
	FrictionConv     float64
	SlopeConv        float64
	OrographicConv   float64
	SmoothedConv     float64
	CombinedRawDiv   float64
	FrictionRawDiv   float64
	SlopeRawDiv      float64
	OrographicRawDiv float64
	SmoothedRawDiv   float64
	CombinedSpeed    float64
	FrictionSpeed    float64
	SlopeSpeed       float64
	OrographicSpeed  float64
	SmoothedSpeed    float64
}

func computeWindPipelineDebug(
	idx int,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	settings WindSettings,
) windPipelineDebug {
	debug := windPipelineDebug{}
	if idx < 0 || idx >= len(vertices) {
		return debug
	}

	settings.Circulation.RossbyPhase = float64(settings.Seed%1000) * 2 * math.Pi / 1000

	pressure, zones := ComputeCirculationPressure(vertices, settings.Circulation)
	pressure = AddGeographicPressureAnomalies(
		pressure, vertices, elevation, seaLevelThreshold, adj, settings.Circulation,
	)

	cellWind := ComputeCellDrivenWind(vertices, zones, settings.Circulation)
	cellSize := estimateCellSize(vertices, adj)
	const pressureSmoothAngular = 0.03
	pressureSmoothIters := int(pressureSmoothAngular/cellSize) + 1
	if pressureSmoothIters < 2 {
		pressureSmoothIters = 2
	}
	smoothedPressure := SmoothScalarField(pressure, vertices, adj, pressureSmoothIters, settings.Circulation.SmoothingFactor)
	pressureWind := ComputePressureGradientWind(smoothedPressure, vertices, adj, 0.24)

	combined := make([]Vector3D, len(cellWind))
	for i := range cellWind {
		combined[i] = Add(cellWind[i], pressureWind[i])
	}
	friction := ApplySurfaceFrictionSimple(combined, vertices, elevation, seaLevelThreshold, settings.Surface)
	slope := ApplySlopeEffects(friction, vertices, elevation, adj)

	orographic := append([]Vector3D(nil), slope...)
	if settings.Orographic.DeflectionStrength > 0 {
		orographic = ApplyOrographicDeflection(orographic, vertices, elevation, adj, settings.Orographic)
		orographic = PropagateLeeShadow(orographic, vertices, elevation, adj, leeShadowIterationCount(cellSize), 0.75)
	}

	const windSmoothAngular = 0.025
	windSmoothIters := int(windSmoothAngular/cellSize) + 1
	if windSmoothIters < 2 {
		windSmoothIters = 2
	}
	smoothed := SmoothVectorFieldBySurface(
		orographic, vertices, elevation, seaLevelThreshold, adj, windSmoothIters, 0.3,
	)

	combinedDiag := computeWindConvergenceDiagnostic(idx, vertices, adj, combined)
	frictionDiag := computeWindConvergenceDiagnostic(idx, vertices, adj, friction)
	slopeDiag := computeWindConvergenceDiagnostic(idx, vertices, adj, slope)
	orographicDiag := computeWindConvergenceDiagnostic(idx, vertices, adj, orographic)
	smoothedDiag := computeWindConvergenceDiagnostic(idx, vertices, adj, smoothed)

	debug.CombinedConv = combinedDiag.Convergence
	debug.FrictionConv = frictionDiag.Convergence
	debug.SlopeConv = slopeDiag.Convergence
	debug.OrographicConv = orographicDiag.Convergence
	debug.SmoothedConv = smoothedDiag.Convergence
	debug.CombinedRawDiv = combinedDiag.RawDivergence
	debug.FrictionRawDiv = frictionDiag.RawDivergence
	debug.SlopeRawDiv = slopeDiag.RawDivergence
	debug.OrographicRawDiv = orographicDiag.RawDivergence
	debug.SmoothedRawDiv = smoothedDiag.RawDivergence
	debug.CombinedSpeed = Length(combined[idx])
	debug.FrictionSpeed = Length(friction[idx])
	debug.SlopeSpeed = Length(slope[idx])
	debug.OrographicSpeed = Length(orographic[idx])
	debug.SmoothedSpeed = Length(smoothed[idx])
	return debug
}
