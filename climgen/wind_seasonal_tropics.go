package climgen

import "math"

const (
	seasonalTropicalPressureGradientStrength = 0.38
	seasonalTropicalMarineBlend              = 0.70
	seasonalTropicalSurfaceBlend             = 0.58
	seasonalTropicalSurfaceSmoothBlend       = 0.18
	seasonalTropicalMarineSmoothBlend        = 0.16
	seasonalTropicalSmoothLandBlend          = 0.34
	seasonalTropicalSmoothOceanBlend         = 0.42
	seasonalTropicalSmoothLandIters          = 5
	seasonalTropicalSmoothOceanIters         = 4
)

// ApplySeasonalTropicalWindAdjustment adds a season-specific tropical
// circulation response driven by the actual seasonal temperature anomaly over
// land and a broad equatorial/oceanic pressure geometry. This sits upstream of
// precipitation so low-lat regimes are not inferred only from precipitation
// support fields.
func ApplySeasonalTropicalWindAdjustment(
	base *WindResult,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	solar SolarSettings,
	temperature []float64,
	annualMeanTemperature []float64,
) *WindResult {
	if base == nil || len(vertices) == 0 || len(temperature) != len(vertices) || len(annualMeanTemperature) != len(vertices) {
		return base
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
	if !hasMeaningfulScalarSignal(anomaly, 1e-5) {
		return base
	}

	pressureWind := ComputePressureGradientWind(
		anomaly,
		vertices,
		adj,
		seasonalTropicalPressureGradientStrength,
	)
	landInterior := ComputeSurfaceInteriorFraction(elevation, seaLevelThreshold, adj, 2200.0, true)

	result := &WindResult{
		MarineWind:      append([]Vector3D(nil), base.MarineWind...),
		SurfaceWind:     append([]Vector3D(nil), base.SurfaceWind...),
		Pressure:        append([]float64(nil), base.Pressure...),
		GeostrophicWind: append([]Vector3D(nil), base.GeostrophicWind...),
		CirculationZone: append([]CirculationZone(nil), base.CirculationZone...),
	}

	for i, v := range vertices {
		absLat := math.Abs(getLatitudeDeg(v))
		tropicalReach := 1.0 - smoothRamp(6.0, 40.0, absLat)
		if tropicalReach <= 0 {
			continue
		}
		interior := 0.0
		if i < len(landInterior) {
			interior = Clamp(landInterior[i], 0, 1)
		}
		surfaceWeight := seasonalTropicalSurfaceBlend * tropicalReach
		if i < len(elevation) && elevation[i] >= seaLevelThreshold {
			// Keep the adjustment broad over land, but avoid turning every land cell
			// into a direct coastal monsoon response.
			surfaceWeight *= 0.82 + 0.18*(0.25+0.75*interior)
		}
		marineWeight := seasonalTropicalMarineBlend * tropicalReach
		result.SurfaceWind[i] = projectTangent(
			Add(result.SurfaceWind[i], Scale(pressureWind[i], surfaceWeight)),
			v,
		)
		result.MarineWind[i] = projectTangent(
			Add(result.MarineWind[i], Scale(pressureWind[i], marineWeight)),
			v,
		)
		result.Pressure[i] += anomaly[i]
	}

	result.SurfaceWind = SmoothVectorFieldBySurface(
		result.SurfaceWind,
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		2,
		seasonalTropicalSurfaceSmoothBlend,
	)
	result.MarineWind = SmoothVectorFieldBySurface(
		result.MarineWind,
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		2,
		seasonalTropicalMarineSmoothBlend,
	)
	return result
}

func computeSeasonalTropicalPressureAnomaly(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	solar SolarSettings,
	temperature []float64,
	annualMeanTemperature []float64,
) []float64 {
	raw := make([]float64, len(vertices))
	declinationDeg := -solar.AxialTilt * math.Cos(2.0*math.Pi*solar.SeasonPhase)
	maxShiftDeg := math.Abs(defaultThermalEquatorShiftScale * solar.AxialTilt)
	summerHemisphere := 0.0
	switch {
	case declinationDeg > 0.1:
		summerHemisphere = 1.0
	case declinationDeg < -0.1:
		summerHemisphere = -1.0
	}

	landInterior := ComputeSurfaceInteriorFraction(elevation, seaLevelThreshold, adj, 2400.0, true)
	oceanExposure := ComputeSurfaceInteriorFraction(elevation, seaLevelThreshold, adj, 2800.0, false)
	localShift := computeSeasonalTropicalConvergenceLatitudeField(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		solar,
		temperature,
		annualMeanTemperature,
	)

	for i, v := range vertices {
		latDeg := getLatitudeDeg(v)
		absLat := math.Abs(latDeg)
		cellShift := SeasonalThermalEquatorShiftDeg(solar)
		if i < len(localShift) {
			cellShift = localShift[i]
		}
		climateLat := latDeg - cellShift
		absClimateLat := math.Abs(climateLat)
		tropicalCore := 1.0 - smoothRamp(4.0, 26.0, absClimateLat)
		if tropicalCore <= 0 {
			continue
		}

		tempAnom := Clamp((temperature[i]-annualMeanTemperature[i])/14.0, -1.0, 1.0)
		equatorialCore := 1.0 - smoothRamp(2.0, 10.0+0.45*maxShiftDeg, absClimateLat)
		itczCrossing := 1.0 - smoothRamp(2.0, math.Max(maxShiftDeg, math.Abs(cellShift))+6.0, absLat)
		monsoonBelt := smoothRamp(4.0, 10.0, absLat) * (1.0 - smoothRamp(18.0, 34.0, absLat))

		if i < len(elevation) && elevation[i] >= seaLevelThreshold {
			interior := Clamp(landInterior[i], 0, 1)
			maritimeAccess := Clamp(1.0-0.70*interior, 0, 1)
			summerSide := 0.0
			winterSide := 0.0
			if summerHemisphere != 0 {
				if latDeg*summerHemisphere > 0 {
					summerSide = 1.0
				} else if latDeg*(-summerHemisphere) > 0 {
					winterSide = 1.0
				}
			}

			heatedLandLow := tropicalCore *
				math.Max(0, tempAnom) *
				(0.35 + 0.65*monsoonBelt) *
				(0.55 + 0.45*(0.25+0.75*interior)) *
				(0.55 + 0.45*maritimeAccess)
			persistentEquatorialLow := equatorialCore *
				(0.45 + 0.55*itczCrossing) *
				(0.50 + 0.50*maritimeAccess)
			winterDryHigh := tropicalCore *
				math.Max(0, -tempAnom) *
				(0.35 + 0.65*(0.20+0.80*interior)) *
				(0.35 + 0.65*monsoonBelt)

			raw[i] -= 0.65 * persistentEquatorialLow
			raw[i] -= 0.95 * summerSide * heatedLandLow
			raw[i] -= 0.30 * itczCrossing * math.Max(0, tempAnom) * (0.35 + 0.65*maritimeAccess)
			raw[i] += 0.55 * winterSide * winterDryHigh
			continue
		}

		basin := 0.0
		if i < len(oceanExposure) {
			basin = Clamp(oceanExposure[i], 0, 1)
		}
		winterTradeHigh := 0.0
		if summerHemisphere != 0 && latDeg*(-summerHemisphere) > 0 {
			winterTradeHigh = smoothRamp(5.0, 10.0, absLat) * (1.0 - smoothRamp(18.0, 34.0, absLat))
		}
		itczOceanLow := equatorialCore * (0.45 + 0.55*itczCrossing) * (0.25 + 0.75*math.Sqrt(basin))
		summerOceanLow := 0.0
		if summerHemisphere != 0 && latDeg*summerHemisphere > 0 {
			summerOceanLow = monsoonBelt * (0.25 + 0.75*math.Sqrt(basin))
		}
		raw[i] += 0.58 * winterTradeHigh * (0.25 + 0.75*math.Sqrt(basin))
		raw[i] -= 0.42 * itczOceanLow
		raw[i] -= 0.18 * summerOceanLow
	}

	landMask := make([]bool, len(vertices))
	oceanMask := make([]bool, len(vertices))
	for i := range vertices {
		landMask[i] = i < len(elevation) && elevation[i] >= seaLevelThreshold
		oceanMask[i] = !landMask[i]
	}

	landSmoothed := SmoothScalarFieldMasked(
		raw,
		landMask,
		adj,
		seasonalTropicalSmoothLandIters,
		seasonalTropicalSmoothLandBlend,
	)
	oceanSmoothed := SmoothScalarFieldMasked(
		raw,
		oceanMask,
		adj,
		seasonalTropicalSmoothOceanIters,
		seasonalTropicalSmoothOceanBlend,
	)

	out := make([]float64, len(raw))
	for i := range raw {
		if landMask[i] {
			out[i] = landSmoothed[i]
		} else {
			out[i] = oceanSmoothed[i]
		}
	}
	return out
}

func hasMeaningfulScalarSignal(field []float64, eps float64) bool {
	for _, v := range field {
		if math.Abs(v) > eps {
			return true
		}
	}
	return false
}

func projectTangent(v Vector3D, normal Vector3D) Vector3D {
	return Sub(v, Scale(normal, Dot(v, normal)))
}
