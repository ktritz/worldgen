package climgen

import "math"

func populateTradeGoodScarcity(result *TradeGoodResult, settings TradeGoodsSettings) {
	if result == nil || result.Diagnostics == nil {
		return
	}
	tuning := settings.EffectiveScarcitySettings()
	endowmentByGood := map[string]TradeGoodEndowment{}
	specByGood := map[string]TradeGoodSpec{}
	for _, good := range result.Goods {
		endowmentByGood[good.Good] = good
	}
	for _, spec := range settings.Goods {
		specByGood[spec.Name] = spec
	}
	cache := map[string]float64{}
	visiting := map[string]bool{}
	for _, spec := range settings.Goods {
		availability := estimateTradeGoodAvailability(spec.Name, endowmentByGood, specByGood, cache, visiting, tuning)
		result.Diagnostics.AvailabilityByGood[spec.Name] = availability
		result.Diagnostics.ScarcityByGood[spec.Name] = tradeGoodScarcityFromAvailability(spec, availability, tuning)
	}
}

func estimateTradeGoodAvailability(
	good string,
	endowmentByGood map[string]TradeGoodEndowment,
	specByGood map[string]TradeGoodSpec,
	cache map[string]float64,
	visiting map[string]bool,
	tuning TradeGoodsScarcitySettings,
) float64 {
	if availability, ok := cache[good]; ok {
		return availability
	}
	if visiting[good] {
		return 0
	}
	spec, ok := specByGood[good]
	if !ok {
		return 0
	}
	visiting[good] = true
	defer delete(visiting, good)
	if len(spec.SourceWeights) > 0 {
		availability := 0.0
		if endowment, ok := endowmentByGood[good]; ok {
			availability = rawTradeGoodAvailability(endowment.Potential, tuning)
		}
		cache[good] = availability
		return availability
	}
	if len(spec.Inputs) == 0 {
		cache[good] = 0
		return 0
	}
	minInput := 1.0
	avgInput := 0.0
	count := 0.0
	for input := range spec.Inputs {
		availability := estimateTradeGoodAvailability(input, endowmentByGood, specByGood, cache, visiting, tuning)
		if availability < minInput {
			minInput = availability
		}
		avgInput += availability
		count++
	}
	if count == 0 {
		cache[good] = 0
		return 0
	}
	avgInput /= count
	availability := inputTradeGoodAvailability(minInput, avgInput, spec.Category, tuning)
	cache[good] = availability
	return availability
}

func tradeGoodScarcity(goods *TradeGoodResult, good string) float64 {
	if goods != nil && goods.Diagnostics != nil {
		if scarcity, ok := goods.Diagnostics.ScarcityByGood[good]; ok {
			return clamp01(scarcity)
		}
	}
	return 0.5
}

func scarcityDemandPressure(spec TradeGoodSpec, scarcity float64) float64 {
	return scarcityResponseValue(spec, scarcity, DefaultTradeGoodsScarcitySettings().DemandResponse)
}

func scarcityDemandPressureWithSettings(spec TradeGoodSpec, scarcity float64, tuning TradeGoodsScarcitySettings) float64 {
	return scarcityResponseValue(spec, scarcity, tuning.DemandResponse)
}

func scarcitySupplyIncentive(spec TradeGoodSpec, scarcity float64) float64 {
	return scarcityResponseValue(spec, scarcity, DefaultTradeGoodsScarcitySettings().SupplyResponse)
}

func scarcitySupplyIncentiveWithSettings(spec TradeGoodSpec, scarcity float64, tuning TradeGoodsScarcitySettings) float64 {
	return scarcityResponseValue(spec, scarcity, tuning.SupplyResponse)
}

func scarcityTradeValue(spec TradeGoodSpec, scarcity float64) float64 {
	return scarcityResponseValue(spec, scarcity, DefaultTradeGoodsScarcitySettings().TradeValueResponse)
}

func scarcityTradeValueWithSettings(spec TradeGoodSpec, scarcity float64, tuning TradeGoodsScarcitySettings) float64 {
	return scarcityResponseValue(spec, scarcity, tuning.TradeValueResponse)
}

func scarcityResponseValue(spec TradeGoodSpec, scarcity float64, curves map[string]TradeGoodsResponseCurve) float64 {
	scarcity = clamp01(scarcity)
	curve, ok := curves[spec.Category]
	if !ok {
		curve = curves["default"]
	}
	return clamp01(curve.Base + curve.Slope*scarcity)
}

func meanFloatClamped(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	count := 0.0
	for _, value := range values {
		total += clamp01(value)
		count++
	}
	if count == 0 {
		return 0
	}
	return total / count
}

func rawTradeGoodAvailability(values []float64, tuning TradeGoodsScarcitySettings) float64 {
	raw := tuning.RawAvailability
	mean, coverage, strongCoverage, peak := tradeGoodPotentialStatsWithThresholds(
		values,
		raw.CoverageThreshold,
		raw.StrongCoverageThreshold,
	)
	return clamp01(
		raw.MeanWeight*sqrt01(mean) +
			raw.CoverageWeight*sqrt01(coverage) +
			raw.StrongCoverageWeight*strongCoverage +
			raw.PeakWeight*peak,
	)
}

func tradeGoodPotentialStats(values []float64) (mean, coverage, strongCoverage, peak float64) {
	return tradeGoodPotentialStatsWithThresholds(values, 0.10, 0.30)
}

func tradeGoodPotentialStatsWithThresholds(values []float64, coverageThreshold, strongCoverageThreshold float64) (mean, coverage, strongCoverage, peak float64) {
	if len(values) == 0 {
		return 0, 0, 0, 0
	}
	total := 0.0
	covered := 0.0
	strong := 0.0
	count := 0.0
	maxValue := 0.0
	for _, raw := range values {
		value := clamp01(raw)
		total += value
		count++
		if value >= coverageThreshold {
			covered++
		}
		if value >= strongCoverageThreshold {
			strong++
		}
		if value > maxValue {
			maxValue = value
		}
	}
	if count == 0 {
		return 0, 0, 0, 0
	}
	return total / count, covered / count, strong / count, maxValue
}

func inputTradeGoodAvailability(minInput, avgInput float64, category string, tuning TradeGoodsScarcitySettings) float64 {
	inputs := tuning.InputAvailability
	availability := clamp01(inputs.MinWeight*minInput + inputs.AvgWeight*avgInput)
	if fit, ok := tuning.CategoryAvailabilityFit[category]; ok {
		availability = clamp01(fit.Offset + fit.Scale*availability)
	}
	return availability
}

func tradeGoodScarcityFromAvailability(spec TradeGoodSpec, availability float64, tuning TradeGoodsScarcitySettings) float64 {
	availability = clamp01(availability)
	power, ok := tuning.CategoryScarcityPower[spec.Category]
	if !ok {
		power = tuning.CategoryScarcityPower["default"]
	}
	return clamp01(1 - pow01(availability, power))
}

func sqrt01(v float64) float64 {
	return pow01(v, 0.5)
}

func pow01(v, exp float64) float64 {
	v = clamp01(v)
	if v <= 0 {
		return 0
	}
	return Clamp(math.Pow(v, exp), 0, 1)
}
