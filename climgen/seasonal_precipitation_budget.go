package climgen

import "math"

const (
	seasonalResidualITCZBoostScale      = 0.28
	seasonalResidualDryBeltSuppression  = 0.08
	seasonalResidualStormTrackBoost     = 0.10
	seasonalResidualPolarDryScale       = 0.18
	seasonalResidualMonsoonBoostScale   = 0.20
	seasonalResidualWinterDryScale      = 0.08
	seasonalPrecipRainfallFractionGain  = 0.18
	seasonalPrecipEvaporationBase       = 0.92
	seasonalPrecipEvaporationAnomScale  = 0.22
	seasonalPrecipOceanTempEffectFloor  = 0.28
	seasonalPrecipOceanTempEffectCeil   = 0.55
	seasonalPrecipLandSourceBase        = 1.00
	seasonalPrecipLandSourceWarmScale   = 0.35
	seasonalPrecipLandRecycleBase       = 0.92
	seasonalPrecipLandRecycleWarmScale  = 0.95
)

// DeriveSeasonalPrecipitationSettings converts the annual precipitation
// settings into a seasonal variant using the actual seasonal thermal state
// instead of relying primarily on post-shaped annual precipitation.
func DeriveSeasonalPrecipitationSettings(
	base PrecipitationSettings,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	solar SolarSettings,
	seasonalTemperature []float64,
	annualMeanTemperature []float64,
) PrecipitationSettings {
	settings := base
	oceanAnom := meanOceanTemperatureAnomaly(
		elevation,
		seaLevelThreshold,
		seasonalTemperature,
		annualMeanTemperature,
	)
	shiftDeg := math.Abs(SeasonalThermalEquatorShiftDeg(solar))
	seasonality := Clamp(shiftDeg/12.0, 0, 1)
	landWarmAnom := meanPositiveLandTemperatureAnomaly(
		elevation,
		seaLevelThreshold,
		seasonalTemperature,
		annualMeanTemperature,
	)
	landColdAnom := meanNegativeLandTemperatureAnomaly(
		elevation,
		seaLevelThreshold,
		seasonalTemperature,
		annualMeanTemperature,
	)

	settings.OceanEvaporationTempEffect = Clamp(
		math.Max(base.OceanEvaporationTempEffect, seasonalPrecipOceanTempEffectFloor)+0.10*seasonality,
		seasonalPrecipOceanTempEffectFloor,
		seasonalPrecipOceanTempEffectCeil,
	)
	settings.EvaporationRate *= Clamp(
		seasonalPrecipEvaporationBase+seasonalPrecipEvaporationAnomScale*oceanAnom,
		0.80,
		1.18,
	)
	settings.RainfallFraction *= 1.0 + seasonalPrecipRainfallFractionGain*seasonality
	settings.LandSourceScale *= Clamp(
		seasonalPrecipLandSourceBase+seasonalPrecipLandSourceWarmScale*landWarmAnom-0.05*landColdAnom,
		0.85,
		1.25,
	)
	settings.LandRecyclingScale *= Clamp(
		seasonalPrecipLandRecycleBase+seasonalPrecipLandRecycleWarmScale*landWarmAnom-0.08*landColdAnom,
		0.75,
		1.60,
	)
	return settings
}

// ApplySeasonalPrecipitationResidual applies a smaller seasonal correction on
// top of the physical seasonal moisture-budget solve. This is intended as a
// residual adjustment, not the primary source of seasonality.
func ApplySeasonalPrecipitationResidual(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	solar SolarSettings,
	temperature []float64,
	annualMeanTemperature []float64,
	landInterior []float64,
	precipitation []float64,
) []float64 {
	adjusted := applySeasonalPrecipitationPatternScaled(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		solar,
		temperature,
		annualMeanTemperature,
		landInterior,
		precipitation,
		seasonalResidualITCZBoostScale,
		seasonalResidualDryBeltSuppression,
		seasonalResidualStormTrackBoost,
		seasonalResidualPolarDryScale,
		seasonalResidualMonsoonBoostScale,
		seasonalResidualWinterDryScale,
	)
	if len(adjusted) != len(vertices) {
		return adjusted
	}

	stormBand := computeSeasonalStormBandSupportField(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		solar,
		landInterior,
	)
	for i := range adjusted {
		if i >= len(elevation) || elevation[i] < seaLevelThreshold || i >= len(stormBand) {
			continue
		}
		interior := 0.0
		if i < len(landInterior) {
			interior = Clamp(landInterior[i], 0, 1)
		}
		boost := 1.0 + seasonalResidualStormBandBoost*stormBand[i]*(0.15+0.85*interior)
		adjusted[i] *= Clamp(boost, 1.0, 1.20)
	}
	return adjusted
}

func meanOceanTemperatureAnomaly(
	elevation []float64,
	seaLevelThreshold float64,
	seasonalTemperature []float64,
	annualMeanTemperature []float64,
) float64 {
	if len(seasonalTemperature) == 0 || len(seasonalTemperature) != len(annualMeanTemperature) {
		return 0
	}
	sum := 0.0
	count := 0
	for i := range seasonalTemperature {
		if elevation[i] >= seaLevelThreshold {
			continue
		}
		sum += seasonalTemperature[i] - annualMeanTemperature[i]
		count++
	}
	if count == 0 {
		return 0
	}
	return Clamp((sum/float64(count))/12.0, -1, 1)
}

func meanPositiveLandTemperatureAnomaly(
	elevation []float64,
	seaLevelThreshold float64,
	seasonalTemperature []float64,
	annualMeanTemperature []float64,
) float64 {
	return meanLandTemperatureAnomalyComponent(
		elevation, seaLevelThreshold, seasonalTemperature, annualMeanTemperature, true,
	)
}

func meanNegativeLandTemperatureAnomaly(
	elevation []float64,
	seaLevelThreshold float64,
	seasonalTemperature []float64,
	annualMeanTemperature []float64,
) float64 {
	return meanLandTemperatureAnomalyComponent(
		elevation, seaLevelThreshold, seasonalTemperature, annualMeanTemperature, false,
	)
}

func meanLandTemperatureAnomalyComponent(
	elevation []float64,
	seaLevelThreshold float64,
	seasonalTemperature []float64,
	annualMeanTemperature []float64,
	positive bool,
) float64 {
	if len(seasonalTemperature) == 0 || len(seasonalTemperature) != len(annualMeanTemperature) {
		return 0
	}
	sum := 0.0
	count := 0
	for i := range seasonalTemperature {
		if elevation[i] < seaLevelThreshold {
			continue
		}
		anomaly := seasonalTemperature[i] - annualMeanTemperature[i]
		if positive {
			anomaly = math.Max(0, anomaly)
		} else {
			anomaly = math.Max(0, -anomaly)
		}
		sum += anomaly
		count++
	}
	if count == 0 {
		return 0
	}
	return Clamp((sum/float64(count))/10.0, 0, 1)
}
