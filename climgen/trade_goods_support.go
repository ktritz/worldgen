package climgen

import (
	"math"
	"sort"
)

func tradeGoodCategoryDemand(category string) float64 {
	switch category {
	case "raw":
		return 0.38
	case "processed":
		return 0.52
	case "finished":
		return 0.66
	case "luxury":
		return 0.58
	case "strategic":
		return 0.74
	default:
		return 0.45
	}
}

func tradeGoodsDemandRelief(category string, localSupply float64, settings TradeGoodsDemandSettings) float64 {
	relief := tradeGoodsCategorySetting(settings.LocalSupplyReliefByCategory, category, 0)
	return math.Max(0.10, 1-relief*clamp01(localSupply))
}

func tradeGoodsDriverDemandMultiplier(category string, driverDemand float64, settings TradeGoodsDemandSettings) float64 {
	scale := tradeGoodsCategorySetting(settings.DriverSpecializationScale, category, 0)
	if driverDemand <= settings.DriverSpecializationPivot {
		return 1
	}
	excess := (driverDemand - settings.DriverSpecializationPivot) / math.Max(1-settings.DriverSpecializationPivot, 0.01)
	return 1 + scale*excess
}

func tradeGoodsProductionMultiplier(category string, signal, pivot float64, settings TradeGoodsProductionSettings) float64 {
	baseScale := tradeGoodsCategorySetting(settings.CategorySupplyScale, category, 1.0)
	specializationScale := tradeGoodsCategorySetting(settings.CategorySpecializationScale, category, 0.0)
	pivot = clamp01(pivot)
	signal = clamp01(signal)
	if signal <= pivot || specializationScale <= 0 {
		return math.Max(baseScale, 0.01)
	}
	excess := (signal - pivot) / math.Max(1-pivot, 0.01)
	return math.Max(baseScale*(1+specializationScale*excess), 0.01)
}

func tradeGoodsProductionDriverMultiplier(spec TradeGoodSpec, driverFit float64) float64 {
	if len(spec.ProductionDrivers) == 0 {
		return 1
	}
	return Clamp(0.78+0.52*clamp01(driverFit), 0.72, 1.30)
}

func tradeGoodsManufacturingOutput(category string, inputAccess, workshop float64, settings TradeGoodsProductionSettings) float64 {
	baseScale := tradeGoodsCategorySetting(settings.ManufacturingBaseScale, category, 0.45)
	workshopScale := tradeGoodsCategorySetting(settings.ManufacturingWorkshopScale, category, 0.55)
	return clamp01(clamp01(inputAccess) * (baseScale + workshopScale*clamp01(workshop)))
}

func tradeGoodsRawCatchmentSpecializationMultiplier(spec TradeGoodSpec, rawPotentials map[string]float64, settings TradeGoodsProductionSettings) float64 {
	if spec.Category != "raw" || len(rawPotentials) == 0 || settings.RawCatchmentSpecializationScale <= 0 {
		return 1
	}
	local := clamp01(rawPotentials[spec.Name])
	if local <= 0 {
		return 1
	}
	sum := 0.0
	maxv := 0.0
	count := 0.0
	for _, value := range rawPotentials {
		value = clamp01(value)
		if value <= 0 {
			continue
		}
		sum += value
		count++
		if value > maxv {
			maxv = value
		}
	}
	if count <= 1 || sum <= 0 || maxv <= 0 {
		return 1
	}
	mean := sum / count
	dominance := clamp01((local - mean) / math.Max(maxv-mean, 0.01))
	rankBias := clamp01(local / maxv)
	specialization := clamp01(0.65*dominance + 0.35*rankBias)
	base := settings.RawCatchmentSpecializationFloor + settings.RawCatchmentSpecializationScale*specialization
	return applyGoodRawCatchmentSensitivity(spec, base)
}

func applyGoodRawCatchmentSensitivity(spec TradeGoodSpec, base float64) float64 {
	sensitivity := spec.RawCatchmentSensitivity
	if sensitivity <= 0 {
		sensitivity = 1.0
	}
	return Clamp(1+sensitivity*(base-1), 0.35, 2.25)
}

func tradeGoodsCategorySetting(values map[string]float64, category string, fallback float64) float64 {
	if value, ok := values[category]; ok {
		return value
	}
	if value, ok := values["default"]; ok {
		return value
	}
	return fallback
}

func profilePreferenceMatches(key string, assignment PolityProfileAssignment) bool {
	if key == "" {
		return false
	}
	if assignment.Profile.Name == key || assignment.Profile.AncestryName == key || assignment.Profile.CultureName == key {
		return true
	}
	const tagPrefix = "tag:"
	if len(key) > len(tagPrefix) && key[:len(tagPrefix)] == tagPrefix {
		tag := key[len(tagPrefix):]
		return hasProfileTag(assignment.ContextTags, tag) || hasProfileTag(assignment.EnvironmentTags, tag) || hasProfileTag(assignment.Profile.Tags, tag)
	}
	return false
}

func topPolityGoods(values map[string]float64, limit int, exports bool) []PolityGoodValue {
	out := make([]PolityGoodValue, 0, len(values))
	for good, value := range values {
		if exports {
			if value < 0.01 {
				continue
			}
			out = append(out, PolityGoodValue{Good: good, Value: value})
		} else if value <= -0.01 {
			out = append(out, PolityGoodValue{Good: good, Value: -value})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Value != out[j].Value {
			return out[i].Value > out[j].Value
		}
		return out[i].Good < out[j].Good
	})
	if len(out) < limit {
		limit = len(out)
	}
	return out[:limit]
}

func safeSliceValue(values []float64, idx int) float64 {
	if idx < 0 || idx >= len(values) {
		return 0
	}
	return clamp01(values[idx])
}
