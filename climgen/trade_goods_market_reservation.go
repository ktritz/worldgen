package climgen

import "math"

func marketEffectiveManufacturingInputs(
	spec TradeGoodSpec,
	market *TradeNodeMarket,
	chainPressure map[string]float64,
	production TradeGoodsProductionSettings,
) (float64, float64, float64) {
	if market == nil || len(spec.Inputs) == 0 {
		return 0, 0, 0
	}
	unlockSignal := clamp01(maxFloat(chainPressure[spec.Name]/1.25, spec.MarketInputReservePriority))
	inputAccess := 1.0
	total := 0.0
	count := 0.0
	capacity := math.MaxFloat64
	for input, need := range spec.Inputs {
		if need <= 0 {
			continue
		}
		available := math.Max(market.Supply[input]-market.Demand[input], 0)
		reserved := marketReservedInputAmount(input, available, unlockSignal, chainPressure, production)
		effective := math.Max(available-reserved, 0)
		localInput := clamp01(effective / need)
		inputAccess = math.Min(inputAccess, localInput)
		total += localInput
		count++
		capacity = math.Min(capacity, effective/need)
	}
	if count == 0 || capacity == math.MaxFloat64 {
		return 0, 0, 0
	}
	return clamp01(inputAccess), clamp01(total / count), math.Max(capacity, 0)
}

func marketReservedInputAmount(
	input string,
	available float64,
	unlockSignal float64,
	chainPressure map[string]float64,
	production TradeGoodsProductionSettings,
) float64 {
	if available <= 0 {
		return 0
	}
	pressure := clamp01(chainPressure[input] / 1.25)
	if pressure <= 0 {
		return 0
	}
	inputWeight := tradeGoodsCategorySetting(production.MarketInputReservationByInput, input, 0)
	if inputWeight <= 0 {
		return 0
	}
	effectivePressure := clamp01(pressure - 0.85*unlockSignal)
	if effectivePressure <= 0 {
		return 0
	}
	reserveShare := clamp01(production.MarketInputReservationStrength * inputWeight * effectivePressure)
	reserveShare = math.Min(reserveShare, production.MarketInputReservationCap)
	return available * reserveShare
}
