package climgen

import "math"

const (
	seasonalDynamicITCZWidthDeg        = 16.0
	seasonalDynamicDryBeltCenterDeg    = 24.0
	seasonalDynamicDryBeltHalfWidth    = 10.0
	seasonalDynamicMonsoonMaxLatDeg    = 32.0
	seasonalDynamicCondenseScaleMin    = 0.68
	seasonalDynamicCondenseScaleMax    = 1.55
	seasonalDynamicSourceScaleMin      = 0.70
	seasonalDynamicSourceScaleMax      = 1.55
	seasonalDynamicRecycleScaleMin     = 0.65
	seasonalDynamicRecycleScaleMax     = 1.65
	seasonalDynamicFrontalScaleMax     = 1.80
	seasonalDynamicFrontalTransportMax = 2.10
)

// ApplySeasonalDynamicPrecipitationForcing pushes more of the seasonal
// rain-belt and storm-track behavior into the physical moisture-budget solve by
// modulating local condensation efficiency from the actual seasonal wind field.
func ApplySeasonalDynamicPrecipitationForcing(
	settings PrecipitationSettings,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	solar SolarSettings,
	temperature []float64,
	annualMeanTemperature []float64,
	landInterior []float64,
) PrecipitationSettings {
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
	convergenceField := computeClimateConvergenceField(vertices, adj, wind)
	scales := make([]float64, len(vertices))
	retention := make([]float64, len(vertices))
	source := make([]float64, len(vertices))
	recycling := make([]float64, len(vertices))
	tropicalSource := make([]float64, len(vertices))
	frontalSource := make([]float64, len(vertices))
	frontalRetention := make([]float64, len(vertices))
	frontalTransport := make([]float64, len(vertices))
	shiftDeg := SeasonalThermalEquatorShiftDeg(solar)
	declinationDeg := -solar.AxialTilt * math.Cos(2.0*math.Pi*solar.SeasonPhase)
	summerHemisphere := 0.0
	switch {
	case declinationDeg > 0.1:
		summerHemisphere = 1.0
	case declinationDeg < -0.1:
		summerHemisphere = -1.0
	}

	for i, v := range vertices {
		if i >= len(elevation) || elevation[i] < seaLevelThreshold {
			continue
		}
		latDeg := getLatitudeDeg(v)
		climateLat := latDeg - shiftDeg
		absClimateLat := math.Abs(climateLat)
		positiveConvergence := 0.0
		if i < len(convergenceField) {
			positiveConvergence = math.Max(0, convergenceField[i])
		}
		onshore := coastalOnshoreScore(i, vertices, elevation, seaLevelThreshold, adj, wind)
		frontalExposure := 0.0
		if i < len(frontalExposureField) {
			frontalExposure = frontalExposureField[i]
		}
		stormMoisture := 0.0
		if i < len(stormMoistureField) {
			stormMoisture = stormMoistureField[i]
		}
		stormBandSupport := 0.0
		if i < len(stormBandSupportField) {
			stormBandSupport = stormBandSupportField[i]
		}
		tropicalPlacement := 0.0
		if i < len(tropicalRegimes.Placement) {
			tropicalPlacement = tropicalRegimes.Placement[i]
		}
		persistentWet := 0.0
		if i < len(tropicalRegimes.PersistentWet) {
			persistentWet = tropicalRegimes.PersistentWet[i]
		}
		itczCrossing := 0.0
		if i < len(tropicalRegimes.ITCZCrossing) {
			itczCrossing = tropicalRegimes.ITCZCrossing[i]
		}
		dryPocket := 0.0
		if i < len(tropicalRegimes.DryPocket) {
			dryPocket = tropicalRegimes.DryPocket[i]
		}
		interior := 0.0
		if i < len(landInterior) {
			interior = Clamp(landInterior[i], 0, 1)
		}
		inlandTropicalSupport := tropicalPlacement * (0.25 + 0.75*interior)
		persistentRegime := Clamp((0.55*persistentWet+0.75*itczCrossing)*(1.0-0.55*dryPocket), 0, 1)
		monsoonRegime := Clamp(tropicalPlacement*(1.0-0.75*persistentRegime), 0, 1)
		dryRegime := Clamp(dryPocket*(1.0-0.65*persistentRegime), 0, 1)

		tropicalLift := 1.0 - smoothRamp(5.0, seasonalDynamicITCZWidthDeg, absClimateLat)
		stormTrack := 0.0
		if summerHemisphere != 0 && latDeg*(-summerHemisphere) > 0 {
			distFromStormCore := math.Abs(absClimateLat - seasonalDynamicStormCenterDeg)
			if distFromStormCore < seasonalDynamicStormHalfWidthDeg {
				stormTrack = 1.0 - distFromStormCore/seasonalDynamicStormHalfWidthDeg
			}
		}

		dryBelt := 0.0
		distFromDryCore := math.Abs(absClimateLat - seasonalDynamicDryBeltCenterDeg)
		if distFromDryCore < seasonalDynamicDryBeltHalfWidth {
			dryBelt = 1.0 - distFromDryCore/seasonalDynamicDryBeltHalfWidth
		}

		monsoon := 0.0
		absLat := math.Abs(latDeg)
		if summerHemisphere != 0 && latDeg*summerHemisphere > 0 && absLat >= 8.0 && absLat <= seasonalDynamicMonsoonMaxLatDeg {
			monsoon = onshore * (1.0 - 0.55*interior)
		}

		thermalAnom := 0.0
		if i < len(temperature) && i < len(annualMeanTemperature) {
			thermalAnom = Clamp((temperature[i]-annualMeanTemperature[i])/16.0, -1.0, 1.0)
		}
		summerTropical := 0.0
		winterTropicalDry := 0.0
		if summerHemisphere != 0 && absLat <= 24.0 {
			tropicalSeason := 1.0 - smoothRamp(6.0, 24.0, absLat)
			if latDeg*summerHemisphere > 0 {
				summerTropical = tropicalSeason *
					(0.40 + 0.60*math.Max(0, thermalAnom)) *
					(0.45 + 0.35*tropicalLift + 0.20*onshore) *
					(0.60 + 0.40*(1.0-0.35*interior))
			} else if latDeg*(-summerHemisphere) > 0 {
				winterTropicalDry = tropicalSeason *
					(0.35 + 0.65*(1.0-math.Max(0, thermalAnom))) *
					(0.55 + 0.45*(1.0-0.30*interior))
			}
		}
		summerTropical *= 1.0 - 0.88*persistentRegime
		winterTropicalDry *= 1.0 - 0.92*persistentRegime
		monsoon *= 1.0 - 0.62*persistentRegime

		forcingSupport := Clamp(
			0.50*frontalExposure+
				0.55*stormMoisture+
				0.45*stormBandSupport+
				0.85*tropicalPlacement+
				0.90*persistentRegime+
				0.85*itczCrossing+
				0.20*onshore,
			0,
			1,
		)
		unsupportedConvergence := positiveConvergence *
			(1.0-forcingSupport) *
			(1.0-0.55*dryRegime) *
			(1.0-smoothRamp(18.0, 32.0, absLat))

		scale := 0.97 +
			0.14*tropicalLift +
			0.36*stormTrack +
			0.22*positiveConvergence +
			0.22*frontalExposure*(0.20+0.80*interior) +
			0.18*stormMoisture*(0.15+0.85*interior) +
			0.26*stormBandSupport*(0.15+0.85*interior) +
			0.18*tropicalPlacement*(0.25+0.75*(0.25+0.75*interior)) +
			0.18*itczCrossing*(0.20+0.80*(0.20+0.80*interior)) +
			0.22*persistentRegime*(0.20+0.80*(0.20+0.80*interior)) +
			0.08*monsoonRegime*(0.20+0.80*(0.20+0.80*interior)) +
			0.08*inlandTropicalSupport*(1.0-0.35*onshore) +
				0.10*monsoon +
				0.24*summerTropical -
				0.28*dryRegime*(0.30+0.70*interior) -
				0.26*unsupportedConvergence -
				0.14*winterTropicalDry +
				0.08*math.Max(0, thermalAnom) -
				0.16*dryBelt*(0.55+0.45*interior)

		scales[i] = Clamp(scale, seasonalDynamicCondenseScaleMin, seasonalDynamicCondenseScaleMax)
		winterWesterlyBase := stormTrack * (0.16 + 0.24*interior + 0.18*stormBandSupport)
		retention[i] = Clamp(
			0.94+
				0.48*frontalExposure*(0.20+0.80*interior)+
				0.42*stormMoisture*(0.25+0.75*interior)+
				0.34*stormBandSupport*(0.25+0.75*interior)+
				0.16*winterWesterlyBase+
				0.14*stormTrack*interior-
				0.08*stormTrack*(1.0-interior),
			0.82,
			1.65,
		)
		source[i] = Clamp(
			0.92+
				0.24*frontalExposure*(0.25+0.75*interior)+
				0.26*stormMoisture*(0.20+0.80*interior)+
				0.24*stormBandSupport*(0.25+0.75*interior)+
				0.24*tropicalPlacement*(0.20+0.80*(0.25+0.75*interior))+
				0.24*itczCrossing*(0.20+0.80*(0.20+0.80*interior))+
				0.28*persistentRegime*(0.20+0.80*(0.20+0.80*interior))+
				0.12*monsoonRegime*(0.20+0.80*(0.20+0.80*interior))+
				0.16*inlandTropicalSupport*(1.0-0.35*onshore)+
				0.16*summerTropical*(0.20+0.80*(1.0-0.25*interior))-
				0.22*dryRegime*(0.30+0.70*interior)-
				0.08*winterTropicalDry+
				0.20*winterWesterlyBase+
				0.18*stormTrack*interior+
				0.12*math.Max(0, thermalAnom)*interior-
				0.28*unsupportedConvergence-
				0.08*dryBelt*(0.40+0.60*(1.0-interior)),
			seasonalDynamicSourceScaleMin,
			seasonalDynamicSourceScaleMax,
		)
		recycling[i] = Clamp(
			0.90+
				0.34*frontalExposure*(0.20+0.80*interior)+
				0.34*stormMoisture*(0.20+0.80*interior)+
				0.26*stormBandSupport*(0.20+0.80*interior)+
				0.26*tropicalPlacement*(0.20+0.80*(0.25+0.75*interior))+
				0.28*itczCrossing*(0.20+0.80*(0.20+0.80*interior))+
				0.30*persistentRegime*(0.20+0.80*(0.20+0.80*interior))+
				0.10*monsoonRegime*(0.20+0.80*(0.20+0.80*interior))+
				0.18*inlandTropicalSupport*(1.0-0.30*onshore)+
				0.22*summerTropical*(0.20+0.80*(1.0-0.25*interior))-
				0.22*unsupportedConvergence-
				0.18*dryRegime*(0.25+0.75*interior)-
				0.08*winterTropicalDry+
				0.18*math.Max(0, thermalAnom)*(0.30+0.70*interior)-
				0.10*dryBelt,
			seasonalDynamicRecycleScaleMin,
			seasonalDynamicRecycleScaleMax,
		)
		tropicalSource[i] = Clamp(
			0.04+
				1.05*tropicalPlacement*(0.20+0.80*(0.25+0.75*interior))+
				0.92*itczCrossing*(0.25+0.75*(0.20+0.80*interior))+
				0.72*persistentRegime*(0.25+0.75*(0.20+0.80*interior))+
				0.16*monsoonRegime*(0.20+0.80*(0.20+0.80*interior))+
				0.78*summerTropical*(0.20+0.80*(1.0-0.25*interior))+
				0.30*tropicalLift*(0.35+0.65*math.Max(0, thermalAnom))+
				0.18*monsoon+
				0.10*frontalExposure*(0.20+0.80*interior)-
				0.36*unsupportedConvergence+
				0.06*onshore*(1.0-0.35*interior)-
				0.30*dryRegime*(0.30+0.70*interior)-
				0.18*winterTropicalDry-
				0.10*dryBelt-
				0.12*onshore*(1.0-interior),
			0,
			2.30,
		)
		frontalSource[i] = Clamp(
			0.12+
				0.55*stormTrack*(0.20+0.80*interior)+
				0.45*frontalExposure*(0.25+0.75*interior)+
				0.35*stormMoisture*(0.20+0.80*interior)+
				0.70*stormBandSupport*(0.25+0.75*interior)+
				0.55*winterWesterlyBase+
				0.10*positiveConvergence-
				0.12*dryBelt,
			0,
			seasonalDynamicFrontalScaleMax,
		)
		frontalRetention[i] = Clamp(
			0.08+
				0.42*stormTrack*(0.25+0.75*interior)+
				0.48*stormMoisture*(0.20+0.80*interior)+
				0.22*frontalExposure*(0.25+0.75*interior)-
				0.00*dryBelt+
				0.78*stormBandSupport*(0.25+0.75*interior)+
				0.48*winterWesterlyBase,
			0,
			seasonalDynamicFrontalScaleMax,
		)
		frontalTransport[i] = Clamp(
			0.14+
				0.82*stormBandSupport*(0.25+0.75*interior)+
				0.28*stormMoisture*(0.20+0.80*interior)+
				0.22*frontalExposure*(0.25+0.75*interior)+
				0.70*winterWesterlyBase+
				0.12*stormTrack*(0.20+0.80*interior)-
				0.08*dryBelt,
			0,
			seasonalDynamicFrontalTransportMax,
		)
	}

	settings.CondensationLocalScale = mergeLocalPrecipitationScale(settings.CondensationLocalScale, scales)
	settings.LandRetentionLocalScale = mergeRetentionScale(settings.LandRetentionLocalScale, retention)
	settings.LandSourceLocalScale = mergeBoundedLocalScale(settings.LandSourceLocalScale, source, seasonalDynamicSourceScaleMin, seasonalDynamicSourceScaleMax)
	settings.LandRecyclingLocalScale = mergeBoundedLocalScale(settings.LandRecyclingLocalScale, recycling, seasonalDynamicRecycleScaleMin, seasonalDynamicRecycleScaleMax)
	settings.TropicalSourceLocalScale = mergeBoundedLocalScale(settings.TropicalSourceLocalScale, tropicalSource, 0, 2.30)
	settings.FrontalSourceLocalScale = mergeBoundedLocalScale(settings.FrontalSourceLocalScale, frontalSource, 0, seasonalDynamicFrontalScaleMax)
	settings.FrontalRetentionLocalScale = mergeBoundedLocalScale(settings.FrontalRetentionLocalScale, frontalRetention, 0, seasonalDynamicFrontalScaleMax)
	settings.FrontalTransportLocalScale = mergeBoundedLocalScale(settings.FrontalTransportLocalScale, frontalTransport, 0, seasonalDynamicFrontalTransportMax)
	return settings
}

func mergeLocalPrecipitationScale(base []float64, mod []float64) []float64 {
	if len(mod) == 0 {
		return base
	}
	if len(base) == 0 {
		return append([]float64(nil), mod...)
	}
	n := len(mod)
	if len(base) < n {
		n = len(base)
	}
	out := append([]float64(nil), mod...)
	for i := 0; i < n; i++ {
		out[i] = Clamp(base[i]*mod[i], seasonalDynamicCondenseScaleMin, seasonalDynamicCondenseScaleMax)
	}
	return out
}

func mergeRetentionScale(base []float64, mod []float64) []float64 {
	if len(mod) == 0 {
		return base
	}
	if len(base) == 0 {
		return append([]float64(nil), mod...)
	}
	n := len(mod)
	if len(base) < n {
		n = len(base)
	}
	out := append([]float64(nil), mod...)
	for i := 0; i < n; i++ {
		out[i] = Clamp(base[i]*mod[i], 0.7, 1.5)
	}
	return out
}

func mergeBoundedLocalScale(base []float64, mod []float64, minValue float64, maxValue float64) []float64 {
	if len(mod) == 0 {
		return base
	}
	if len(base) == 0 {
		out := append([]float64(nil), mod...)
		for i := range out {
			out[i] = Clamp(out[i], minValue, maxValue)
		}
		return out
	}
	n := len(mod)
	if len(base) < n {
		n = len(base)
	}
	out := append([]float64(nil), mod...)
	for i := 0; i < n; i++ {
		out[i] = Clamp(base[i]*mod[i], minValue, maxValue)
	}
	return out
}

func seasonalFrontalExposure(
	i int,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	solar SolarSettings,
) float64 {
	field := computeSeasonalFrontalExposureField(vertices, elevation, seaLevelThreshold, adj, wind, solar)
	if i < 0 || i >= len(field) {
		return 0
	}
	return field[i]
}
