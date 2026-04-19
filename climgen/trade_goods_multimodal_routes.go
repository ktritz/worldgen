package climgen

import (
	"math"
	"sort"
)

func appendRouteGoodExchanges(
	out *MultimodalTradeResult,
	balances map[int]PolityGoodBalance,
	specs map[string]TradeGoodSpec,
	globalScarcity map[string]float64,
	marketsByNode map[int]TradeNodeMarket,
	fromNode, toNode int,
	fromPolity, toPolity int,
	mode string,
	routeID int,
	flow, cost, quality float64,
	tuning TradeGoodsScarcitySettings,
	multimodal TradeGoodsMultimodalSettings,
) {
	if fromPolity < 0 || toPolity < 0 || fromPolity == toPolity {
		return
	}
	appendOneWayRouteGoodExchange(out, balances, specs, globalScarcity, marketsByNode, fromNode, toNode, fromPolity, toPolity, mode, routeID, flow, cost, quality, tuning, multimodal)
	appendOneWayRouteGoodExchange(out, balances, specs, globalScarcity, marketsByNode, toNode, fromNode, toPolity, fromPolity, mode, routeID, flow, cost, quality, tuning, multimodal)
}

func appendOneWayRouteGoodExchange(
	out *MultimodalTradeResult,
	balances map[int]PolityGoodBalance,
	specs map[string]TradeGoodSpec,
	globalScarcity map[string]float64,
	marketsByNode map[int]TradeNodeMarket,
	fromNode, toNode int,
	fromPolity, toPolity int,
	mode string,
	routeID int,
	flow, cost, quality float64,
	tuning TradeGoodsScarcitySettings,
	multimodal TradeGoodsMultimodalSettings,
) {
	out.Diagnostics.RouteCandidates++
	from, ok := balances[fromPolity]
	if !ok {
		return
	}
	to, ok := balances[toPolity]
	if !ok {
		return
	}
	capacity := routeGoodCapacity(mode, flow, cost, quality, multimodal)
	if capacity <= 0 {
		return
	}
	var fromMarket *TradeNodeMarket
	var toMarket *TradeNodeMarket
	if market, ok := marketsByNode[fromNode]; ok {
		fromMarket = &market
	}
	if market, ok := marketsByNode[toNode]; ok {
		toMarket = &market
	}
	volumeCapacity := routeVolumeCapacity(mode, capacity, fromMarket, toMarket, multimodal)
	goods := matchedRouteGoods(from, to, specs, globalScarcity, fromMarket, toMarket, mode, capacity, volumeCapacity, tuning, multimodal, &out.Diagnostics)
	value := sumTradeGoodFlowScores(goods)
	if value < 0.005 {
		return
	}
	out.Diagnostics.RouteActive++
	volume := sumTradeGoodFlowVolumes(goods)
	matched := sumTradeGoodFlowMatched(goods)
	out.Exchanges = append(out.Exchanges, TradeGoodExchange{
		FromPolity:         fromPolity,
		ToPolity:           toPolity,
		Mode:               mode,
		RouteID:            routeID,
		RouteFlow:          flow,
		TravelCost:         cost,
		Value:              value,
		Volume:             volume,
		Matched:            matched,
		Capacity:           capacity,
		VolumeCapacity:     volumeCapacity,
		AvgTransportFactor: averageTradeGoodFlowFactor(goods, tradeGoodFlowTransportFactor),
		AvgLocalNeedFactor: averageTradeGoodFlowFactor(goods, tradeGoodFlowLocalNeedFactor),
		AvgRarityFactor:    averageTradeGoodFlowFactor(goods, tradeGoodFlowRarityFactor),
		AvgMarketFit:       averageTradeGoodFlowFactor(goods, tradeGoodFlowMarketFit),
		Goods:              goods,
	})
}

func matchedRouteGoods(
	from, to PolityGoodBalance,
	specs map[string]TradeGoodSpec,
	globalScarcity map[string]float64,
	fromMarket, toMarket *TradeNodeMarket,
	mode string,
	capacity, volumeCapacity float64,
	tuning TradeGoodsScarcitySettings,
	multimodal TradeGoodsMultimodalSettings,
	diagnostics *MultimodalTradeDiagnostics,
) []TradeGoodFlowValue {
	out := make([]TradeGoodFlowValue, 0)
	for good, spec := range specs {
		if diagnostics != nil {
			diagnostics.CandidateGoods++
		}
		recordCategoryTradeDiagnostic(diagnostics, spec.Category, func(entry *MultimodalTradeCategoryDiagnostics) {
			entry.CandidateGoods++
		})
		surplus := from.Surplus[good]
		if surplus <= 0 {
			if diagnostics != nil {
				diagnostics.NoSourceSurplus++
			}
			recordCategoryTradeDiagnostic(diagnostics, spec.Category, func(entry *MultimodalTradeCategoryDiagnostics) {
				entry.NoSourceSurplus++
			})
			continue
		}
		need := -to.Surplus[good]
		if toMarket != nil {
			endpointNeedShare := tradeGoodsCategorySetting(multimodal.EndpointNeedShareByCategory, spec.Category, 0)
			if endpointNeedShare > 0 {
				need = math.Max(need, endpointNeedShare*marketSinkCapacity(good, *toMarket))
			}
		}
		if need <= 0 {
			if diagnostics != nil {
				diagnostics.NoSinkNeed++
			}
			recordCategoryTradeDiagnostic(diagnostics, spec.Category, func(entry *MultimodalTradeCategoryDiagnostics) {
				entry.NoSinkNeed++
			})
			continue
		}
		sourceCapacity := surplus
		if fromMarket != nil {
			marketSource := marketSourceCapacity(good, *fromMarket)
			if marketSource <= 0 {
				if diagnostics != nil {
					diagnostics.NoEndpointSupply++
				}
				recordCategoryTradeDiagnostic(diagnostics, spec.Category, func(entry *MultimodalTradeCategoryDiagnostics) {
					entry.NoEndpointSupply++
				})
				continue
			}
			sourceCapacity = math.Min(sourceCapacity, marketSource)
		}
		if sourceCapacity < surplus && diagnostics != nil {
			diagnostics.SourceConstrained++
		}
		if sourceCapacity < surplus {
			recordCategoryTradeDiagnostic(diagnostics, spec.Category, func(entry *MultimodalTradeCategoryDiagnostics) {
				entry.SourceConstrained++
			})
		}
		if need < sourceCapacity && diagnostics != nil {
			diagnostics.NeedConstrained++
		}
		if need < sourceCapacity {
			recordCategoryTradeDiagnostic(diagnostics, spec.Category, func(entry *MultimodalTradeCategoryDiagnostics) {
				entry.NeedConstrained++
			})
		}
		matched := math.Min(sourceCapacity, need)
		if matched <= 0 {
			continue
		}
		marketFit := nodeMarketGoodFit(good, fromMarket, toMarket)
		if marketFit < 0.70 && diagnostics != nil {
			diagnostics.LowMarketFit++
		}
		if marketFit < 0.70 {
			recordCategoryTradeDiagnostic(diagnostics, spec.Category, func(entry *MultimodalTradeCategoryDiagnostics) {
				entry.LowMarketFit++
			})
		}
		if volumeCapacity < multimodal.LowCapacityVolumeThreshold && diagnostics != nil {
			diagnostics.LowCapacity++
		}
		if volumeCapacity < multimodal.LowCapacityVolumeThreshold {
			recordCategoryTradeDiagnostic(diagnostics, spec.Category, func(entry *MultimodalTradeCategoryDiagnostics) {
				entry.LowCapacity++
			})
		}
		volume := matched * volumeCapacity * marketFit
		transport := tradeGoodTransportValue(spec, mode)
		localNeed := tradeGoodLocalNeedValueForSpec(spec, tradeGoodLocalNeedForSpec(good, to, toMarket), multimodal)
		rarity := tradeGoodGlobalRarityForSpec(spec, globalScarcity, multimodal)
		score := volume * transport * localNeed * rarity
		if score < 0.001 {
			if diagnostics != nil {
				diagnostics.LowScoreFiltered++
			}
			recordCategoryTradeDiagnostic(diagnostics, spec.Category, func(entry *MultimodalTradeCategoryDiagnostics) {
				entry.LowScoreFiltered++
			})
			continue
		}
		if diagnostics != nil {
			diagnostics.AcceptedGoods++
		}
		recordCategoryTradeDiagnostic(diagnostics, spec.Category, func(entry *MultimodalTradeCategoryDiagnostics) {
			entry.AcceptedGoods++
			entry.TotalScore += score
			entry.TotalVolume += volume
		})
		out = append(out, TradeGoodFlowValue{
			Good:      good,
			Score:     score,
			Volume:    volume,
			Matched:   matched,
			Transport: transport,
			LocalNeed: localNeed,
			Rarity:    rarity,
			MarketFit: marketFit,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Good < out[j].Good
	})
	return out
}

func tradeGoodLocalNeedForSpec(good string, to PolityGoodBalance, toMarket *TradeNodeMarket) float64 {
	need := clamp01(math.Max(-to.Surplus[good], 0))
	if toMarket != nil {
		marketNeed := clamp01(marketSinkCapacity(good, *toMarket) + 0.35*toMarket.Demand[good])
		need = clamp01(0.55*need + 0.45*marketNeed)
	}
	return need
}

func tradeGoodGlobalRarityForSpec(spec TradeGoodSpec, scarcityByGood map[string]float64, settings TradeGoodsMultimodalSettings) float64 {
	scarcity := 0.5
	if value, ok := scarcityByGood[spec.Name]; ok {
		scarcity = clamp01(value)
	}
	return scarcityResponseValue(spec, scarcity, settings.GlobalRarityResponse)
}

func tradeGoodLocalNeedValueForSpec(spec TradeGoodSpec, localNeed float64, settings TradeGoodsMultimodalSettings) float64 {
	return scarcityResponseValue(spec, localNeed, settings.LocalNeedResponse)
}

func marketSourceCapacity(good string, market TradeNodeMarket) float64 {
	return math.Max(market.Surplus[good], 0)
}

func marketSinkCapacity(good string, market TradeNodeMarket) float64 {
	return math.Max(-market.Surplus[good], 0)
}

func routeGoodCapacity(mode string, flow, cost, quality float64, settings TradeGoodsMultimodalSettings) float64 {
	if flow <= 0 {
		return 0
	}
	scale := tradeGoodsModeSetting(settings.CapacityScaleByMode, mode)
	modeFactor := tradeGoodsModeSetting(settings.CapacityFactorByMode, mode)
	friction := 1.0 + math.Max(cost, 0)/scale
	baseCapacity := math.Max(flow, 0) * modeFactor * Clamp(quality, 0.10, 1.20) / friction
	return baseCapacity
}

func routeVolumeCapacity(mode string, capacity float64, fromMarket, toMarket *TradeNodeMarket, settings TradeGoodsMultimodalSettings) float64 {
	if capacity <= 0 {
		return 0
	}
	base := tradeGoodsModeSetting(settings.VolumeBaseByMode, mode)
	wealthLink := 0.45
	feederScale := 1.0
	if fromMarket != nil && toMarket != nil {
		wealthLink = settings.DualMarketWealthBase + settings.DualMarketWealthScale*math.Sqrt(clamp01(fromMarket.Wealth)*clamp01(toMarket.Wealth))
		feederScale += settings.DualMarketFeederScale * math.Min(float64(fromMarket.FeederNodes+toMarket.FeederNodes), 8.0)
	} else if fromMarket != nil {
		wealthLink = settings.SingleMarketWealthBase + settings.SingleMarketWealthScale*clamp01(fromMarket.Wealth)
		feederScale += settings.SingleMarketFeederScale * math.Min(float64(fromMarket.FeederNodes), 6.0)
	} else if toMarket != nil {
		wealthLink = settings.SingleMarketWealthBase + settings.SingleMarketWealthScale*clamp01(toMarket.Wealth)
		feederScale += settings.SingleMarketFeederScale * math.Min(float64(toMarket.FeederNodes), 6.0)
	}
	return math.Max(capacity, 0) * base * wealthLink * feederScale
}

func tradeGoodsModeSetting(values map[string]float64, mode string) float64 {
	if value, ok := values[mode]; ok && value > 0 {
		return value
	}
	if value, ok := values["default"]; ok && value > 0 {
		return value
	}
	return 1
}

func tradeGoodTransportValue(spec TradeGoodSpec, mode string) float64 {
	base := 0.32 + spec.BaseValue
	bulkFit := 1.0 - clamp01(spec.Bulkiness)
	durableFit := 1.0 - clamp01(spec.Perishability)
	switch mode {
	case "land":
		return base * (0.42 + 0.38*bulkFit + 0.20*durableFit)
	case "river":
		return base * (0.58 + 0.17*bulkFit + 0.25*durableFit)
	case "coastal":
		return base * (0.62 + 0.13*bulkFit + 0.25*durableFit)
	case "ocean":
		return base * (0.55 + 0.12*bulkFit + 0.33*durableFit)
	default:
		return base * (0.45 + 0.25*bulkFit + 0.30*durableFit)
	}
}

func tradeGoodFlowTransportFactor(flow TradeGoodFlowValue) float64 {
	return flow.Transport
}

func tradeGoodFlowLocalNeedFactor(flow TradeGoodFlowValue) float64 {
	return flow.LocalNeed
}

func tradeGoodFlowRarityFactor(flow TradeGoodFlowValue) float64 {
	return flow.Rarity
}

func tradeGoodFlowMarketFit(flow TradeGoodFlowValue) float64 {
	return flow.MarketFit
}
