package climgen

func marketManufacturingChainPressure(
	market *TradeNodeMarket,
	settings TradeGoodsSettings,
	node SettlementNode,
	trade *TradeNetworkResult,
	assignment PolityProfileAssignment,
) map[string]float64 {
	pressure := map[string]float64{}
	if market == nil {
		return pressure
	}
	for _, dependent := range settings.Goods {
		if len(dependent.Inputs) == 0 {
			continue
		}
		demandGap := maxFloat(market.Demand[dependent.Name]-market.Supply[dependent.Name], 0)
		if demandGap <= 0.01 {
			continue
		}
		demandBase := maxFloat(market.Demand[dependent.Name], 0.12)
		demandSignal := clamp01(demandGap / demandBase)
		valueSignal := clamp01(0.72*dependent.BaseValue + 0.20*(1-dependent.Bulkiness) + 0.08*(1-dependent.Perishability))
		contextFit := marketProductionContextMultiplier(dependent, node, market, trade, assignment)
		contextSignal := clamp01((contextFit - 0.50) / 1.25)
		profileSignal := clamp01((profileGoodAffinity(dependent.ProfileProductionAffinity, assignment) - 0.20) / 2.0)
		pull := clamp01(0.42*demandSignal + 0.22*valueSignal + 0.16*contextSignal + 0.10*profileSignal + 0.10*clamp01(market.Wealth))
		for input, need := range dependent.Inputs {
			if need <= 0 {
				continue
			}
			pressure[input] += pull * clamp01(need/0.40)
		}
	}
	return pressure
}

func marketConsumedInputPressure(spec TradeGoodSpec, chainPressure map[string]float64) float64 {
	if len(spec.Inputs) == 0 || len(chainPressure) == 0 {
		return 0
	}
	total := 0.0
	weight := 0.0
	for input, need := range spec.Inputs {
		if need <= 0 {
			continue
		}
		inputWeight := clamp01(need / 0.40)
		total += clamp01(chainPressure[input]/1.25) * inputWeight
		weight += inputWeight
	}
	if weight <= 0 {
		return 0
	}
	return clamp01(total / weight)
}
