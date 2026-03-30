package climgen

const (
	seasonalLandStorageRechargeMinCm = 6.0
	seasonalLandStorageRechargeMaxCm = 70.0
	seasonalLandStorageEvapMinC      = 4.0
	seasonalLandStorageEvapMaxC      = 30.0
)

// InitializeSeasonalLandStorage seeds a persistent land-water storage field for
// the seasonal hydrology loop. Continental interiors keep a slightly larger
// reservoir than exposed coasts.
func InitializeSeasonalLandStorage(
	elevation []float64,
	seaLevelThreshold float64,
	landInterior []float64,
) []float64 {
	storage := make([]float64, len(elevation))
	for i := range elevation {
		if elevation[i] < seaLevelThreshold {
			continue
		}
		interior := 0.0
		if i < len(landInterior) {
			interior = Clamp(landInterior[i], 0, 1)
		}
		storage[i] = 0.32 + 0.28*interior
	}
	return storage
}

// ApplySeasonalLandStorageToSettings lets antecedent moisture affect how much
// seasonal land moisture can be sourced and recycled.
func ApplySeasonalLandStorageToSettings(
	settings PrecipitationSettings,
	elevation []float64,
	seaLevelThreshold float64,
	temperature []float64,
	storage []float64,
) PrecipitationSettings {
	if len(storage) == 0 {
		return settings
	}
	warmStorage := meanWarmLandStorage(elevation, seaLevelThreshold, temperature, storage)
	settings.LandSourceScale *= 0.88 + 0.30*warmStorage
	settings.LandRecyclingScale *= 0.70 + 0.85*warmStorage
	settings.LandSourceLocalScale = seasonalLandStorageSourceScale(
		elevation,
		seaLevelThreshold,
		temperature,
		storage,
	)
	settings.LandRecyclingLocalScale = seasonalLandStorageRecyclingScale(
		elevation,
		seaLevelThreshold,
		temperature,
		storage,
	)
	settings.LandSurfaceStorage = append([]float64(nil), storage...)
	return settings
}

// AdvanceSeasonalLandStorage updates the persistent land-water state using the
// current seasonal precipitation totals and evaporative demand.
func AdvanceSeasonalLandStorage(
	previous []float64,
	elevation []float64,
	seaLevelThreshold float64,
	temperature []float64,
	rainfall []float64,
	snowfall []float64,
	landInterior []float64,
) []float64 {
	next := make([]float64, len(previous))
	copy(next, previous)
	for i := range next {
		if elevation[i] < seaLevelThreshold {
			next[i] = 0
			continue
		}
		rechargeInput := seasonalLandRechargeInput(temperature, i, rainfall, snowfall)
		recharge := smoothRamp(
			seasonalLandStorageRechargeMinCm,
			seasonalLandStorageRechargeMaxCm,
			rechargeInput,
		)
		tempC := temperature[i] - 273.15
		evapDemand := smoothRamp(seasonalLandStorageEvapMinC, seasonalLandStorageEvapMaxC, tempC)
		interior := 0.0
		if i < len(landInterior) {
			interior = Clamp(landInterior[i], 0, 1)
		}
		drainage := 0.18 + 0.20*(1.0-interior)
		evapLoss := evapDemand * (0.16 + 0.20*(1.0-interior))
		next[i] = Clamp(previous[i]+0.38*recharge-evapLoss-drainage*0.04, 0.08, 1.0)
	}
	return next
}

func seasonalLandRechargeInput(
	temperature []float64,
	idx int,
	rainfall []float64,
	snowfall []float64,
) float64 {
	rain := 0.0
	if idx >= 0 && idx < len(rainfall) {
		rain = rainfall[idx]
	}
	snow := 0.0
	if idx >= 0 && idx < len(snowfall) {
		snow = snowfall[idx]
	}
	tempC := 0.0
	if idx >= 0 && idx < len(temperature) {
		tempC = temperature[idx] - 273.15
	}
	meltFraction := smoothRamp(-2.0, 6.0, tempC)
	return rain + snow*(0.08+0.92*meltFraction)
}

func meanWarmLandStorage(
	elevation []float64,
	seaLevelThreshold float64,
	temperature []float64,
	storage []float64,
) float64 {
	if len(storage) == 0 {
		return 0
	}
	sum := 0.0
	count := 0
	for i := range storage {
		if elevation[i] < seaLevelThreshold {
			continue
		}
		tempC := temperature[i] - 273.15
		warmth := smoothRamp(6.0, 28.0, tempC)
		sum += storage[i] * warmth
		count++
	}
	if count == 0 {
		return 0
	}
	return Clamp(sum/float64(count), 0, 1)
}

func seasonalLandStorageSourceScale(
	elevation []float64,
	seaLevelThreshold float64,
	temperature []float64,
	storage []float64,
) []float64 {
	scales := make([]float64, len(storage))
	for i := range scales {
		if i >= len(elevation) || elevation[i] < seaLevelThreshold {
			continue
		}
		tempC := temperature[i] - 273.15
		warmth := smoothRamp(4.0, 28.0, tempC)
		stored := Clamp(storage[i], 0, 1)
		scales[i] = Clamp(0.72+0.22*warmth+0.52*stored*warmth, 0.65, 1.38)
	}
	return scales
}

func seasonalLandStorageRecyclingScale(
	elevation []float64,
	seaLevelThreshold float64,
	temperature []float64,
	storage []float64,
) []float64 {
	scales := make([]float64, len(storage))
	for i := range scales {
		if i >= len(elevation) || elevation[i] < seaLevelThreshold {
			continue
		}
		tempC := temperature[i] - 273.15
		warmth := smoothRamp(2.0, 30.0, tempC)
		stored := Clamp(storage[i], 0, 1)
		scales[i] = Clamp(0.48+0.28*warmth+0.92*stored*(0.30+0.70*warmth), 0.35, 1.72)
	}
	return scales
}
