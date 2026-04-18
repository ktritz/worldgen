package climgen

import "sort"

type marketManufacturingPlan struct {
	Spec     TradeGoodSpec
	Priority float64
}

func buildMarketManufacturingPlans(
	market *TradeNodeMarket,
	settings TradeGoodsSettings,
	node SettlementNode,
	trade *TradeNetworkResult,
	assignment PolityProfileAssignment,
	production TradeGoodsProductionSettings,
	chainPressure map[string]float64,
) []marketManufacturingPlan {
	if market == nil {
		return nil
	}
	plans := make([]marketManufacturingPlan, 0, len(settings.Goods))
	for _, spec := range settings.Goods {
		if len(spec.Inputs) == 0 {
			continue
		}
		if !marketManufacturingEligibleForMarket(spec, node, market) {
			continue
		}
		inputAccess, _, capacity := marketEffectiveManufacturingInputs(spec, market, chainPressure, production)
		if inputAccess <= 0.01 || capacity <= 0.01 {
			continue
		}
		profileAffinity := profileGoodAffinity(spec.ProfileProductionAffinity, assignment)
		contextFit := marketProductionContextMultiplier(spec, node, market, trade, assignment)
		priority := marketManufacturingPriority(
			spec,
			market,
			settings,
			node,
			trade,
			assignment,
			profileAffinity,
			contextFit,
			inputAccess,
			capacity,
			chainPressure,
		)
		if priority <= 0 {
			continue
		}
		plans = append(plans, marketManufacturingPlan{
			Spec:     spec,
			Priority: priority,
		})
	}
	sort.Slice(plans, func(i, j int) bool {
		if plans[i].Priority != plans[j].Priority {
			return plans[i].Priority > plans[j].Priority
		}
		if plans[i].Spec.BaseValue != plans[j].Spec.BaseValue {
			return plans[i].Spec.BaseValue > plans[j].Spec.BaseValue
		}
		return plans[i].Spec.Name < plans[j].Spec.Name
	})
	return plans
}

func marketManufacturingPriority(
	spec TradeGoodSpec,
	market *TradeNodeMarket,
	settings TradeGoodsSettings,
	node SettlementNode,
	trade *TradeNetworkResult,
	assignment PolityProfileAssignment,
	profileAffinity float64,
	contextFit float64,
	inputAccess float64,
	capacity float64,
	chainPressure map[string]float64,
) float64 {
	if market == nil {
		return 0
	}
	demandGap := maxFloat(market.Demand[spec.Name]-market.Supply[spec.Name], 0)
	demandBase := maxFloat(market.Demand[spec.Name], 0.12)
	demandSignal := clamp01(demandGap / demandBase)
	capacitySignal := clamp01(capacity / 1.5)
	valueSignal := clamp01(0.72*spec.BaseValue + 0.20*(1-spec.Bulkiness) + 0.08*(1-spec.Perishability))
	contextSignal := clamp01((contextFit - 0.50) / 1.25)
	profileSignal := clamp01((profileAffinity - 0.20) / 2.0)
	wealthSignal := clamp01(0.64*market.Wealth + 0.36*nodeUrbanity(node))
	unlockSignal := clamp01(chainPressure[spec.Name] / 1.25)
	consumedPressure := marketConsumedInputPressure(spec, chainPressure)
	categoryShare := marketManufacturedCategoryShare(market, spec.Category, settings)
	goodShare := clamp01(market.Manufactured[spec.Name] / 3.0)
	diversitySignal := clamp01(1 - 0.65*categoryShare - 0.35*goodShare)
	contestPenalty := clamp01(consumedPressure * Clamp(0.78-0.30*demandSignal-0.42*unlockSignal, 0.18, 0.85))

	priority := 0.0
	switch spec.Category {
	case "luxury":
		priority = 0.27*demandSignal + 0.16*valueSignal + 0.14*contextSignal + 0.12*profileSignal + 0.10*wealthSignal + 0.10*diversitySignal + 0.04*inputAccess + 0.04*capacitySignal + 0.03*unlockSignal - 0.06*contestPenalty
	case "finished", "strategic":
		priority = 0.22*demandSignal + 0.16*valueSignal + 0.12*contextSignal + 0.08*profileSignal + 0.08*diversitySignal + 0.08*capacitySignal + 0.08*inputAccess + 0.20*unlockSignal - 0.06*contestPenalty
	default:
		priority = 0.27*demandSignal + 0.18*valueSignal + 0.10*contextSignal + 0.08*profileSignal + 0.08*diversitySignal + 0.08*capacitySignal + 0.08*inputAccess + 0.12*unlockSignal - 0.10*contestPenalty
	}
	return clamp01(priority)
}

func marketManufacturedCategoryShare(market *TradeNodeMarket, category string, settings TradeGoodsSettings) float64 {
	total := marketManufacturedTotal(market)
	if total <= 0 {
		return 0
	}
	return clamp01(marketManufacturedCategoryTotal(market, category, settings) / total)
}

func marketManufacturedTotal(market *TradeNodeMarket) float64 {
	if market == nil || len(market.Manufactured) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range market.Manufactured {
		if value > 0 {
			total += value
		}
	}
	return total
}
