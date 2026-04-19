package climgen

import "sort"

func aggregateTradeGoodPairs(exchanges []TradeGoodExchange) []TradeGoodPairFlow {
	type pairAcc struct {
		from       int
		to         int
		value      float64
		volume     float64
		matched    float64
		modes      map[string]float64
		modeVolume map[string]float64
		goods      map[string]TradeGoodFlowValue
	}
	accs := map[[2]int]*pairAcc{}
	for _, exchange := range exchanges {
		key := [2]int{exchange.FromPolity, exchange.ToPolity}
		acc := accs[key]
		if acc == nil {
			acc = &pairAcc{
				from:       exchange.FromPolity,
				to:         exchange.ToPolity,
				modes:      map[string]float64{},
				modeVolume: map[string]float64{},
				goods:      map[string]TradeGoodFlowValue{},
			}
			accs[key] = acc
		}
		acc.value += exchange.Value
		acc.volume += exchange.Volume
		acc.matched += exchange.Matched
		acc.modes[exchange.Mode] += exchange.Value
		acc.modeVolume[exchange.Mode] += exchange.Volume
		for _, good := range exchange.Goods {
			flow := acc.goods[good.Good]
			flow.Good = good.Good
			existingScore := flow.Score
			newScore := good.Score
			totalScore := existingScore + newScore
			if totalScore > 0 {
				flow.Transport = (flow.Transport*existingScore + good.Transport*newScore) / totalScore
				flow.LocalNeed = (flow.LocalNeed*existingScore + good.LocalNeed*newScore) / totalScore
				flow.Rarity = (flow.Rarity*existingScore + good.Rarity*newScore) / totalScore
				flow.MarketFit = (flow.MarketFit*existingScore + good.MarketFit*newScore) / totalScore
			}
			flow.Score += good.Score
			flow.Volume += good.Volume
			flow.Matched += good.Matched
			acc.goods[good.Good] = flow
		}
	}
	out := make([]TradeGoodPairFlow, 0, len(accs))
	for _, acc := range accs {
		goods := make([]TradeGoodFlowValue, 0, len(acc.goods))
		for _, flow := range acc.goods {
			if flow.Score <= 0 {
				continue
			}
			goods = append(goods, flow)
		}
		sort.Slice(goods, func(i, j int) bool {
			if goods[i].Score != goods[j].Score {
				return goods[i].Score > goods[j].Score
			}
			return goods[i].Good < goods[j].Good
		})
		out = append(out, TradeGoodPairFlow{
			FromPolity: acc.from,
			ToPolity:   acc.to,
			Value:      acc.value,
			Volume:     acc.volume,
			Matched:    acc.matched,
			Modes:      topTradeModeValues(acc.modes, 4),
			ModeVolume: topTradeModeValues(acc.modeVolume, 4),
			Goods:      goods,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Value != out[j].Value {
			return out[i].Value > out[j].Value
		}
		if out[i].FromPolity != out[j].FromPolity {
			return out[i].FromPolity < out[j].FromPolity
		}
		return out[i].ToPolity < out[j].ToPolity
	})
	return out
}

func topTradeModeValues(values map[string]float64, limit int) []TradeModeValue {
	out := make([]TradeModeValue, 0, len(values))
	for mode, value := range values {
		if value > 0 {
			out = append(out, TradeModeValue{Mode: mode, Value: value})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Value != out[j].Value {
			return out[i].Value > out[j].Value
		}
		return out[i].Mode < out[j].Mode
	})
	if len(out) < limit {
		limit = len(out)
	}
	return out[:limit]
}

func sumTradeGoodFlowScores(values []TradeGoodFlowValue) float64 {
	total := 0.0
	for _, value := range values {
		total += value.Score
	}
	return total
}

func sumTradeGoodFlowVolumes(values []TradeGoodFlowValue) float64 {
	total := 0.0
	for _, value := range values {
		total += value.Volume
	}
	return total
}

func sumTradeGoodFlowMatched(values []TradeGoodFlowValue) float64 {
	total := 0.0
	for _, value := range values {
		total += value.Matched
	}
	return total
}

func averageTradeGoodFlowFactor(values []TradeGoodFlowValue, factor func(TradeGoodFlowValue) float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	weight := 0.0
	for _, value := range values {
		if value.Score <= 0 {
			continue
		}
		total += value.Score * factor(value)
		weight += value.Score
	}
	if weight <= 0 {
		return 0
	}
	return total / weight
}

func populateMultimodalTradeDiagnostics(result *MultimodalTradeResult) {
	if result == nil || len(result.Exchanges) == 0 {
		return
	}
	totalCapacity := 0.0
	totalVolumeCapacity := 0.0
	totalMarketFit := 0.0
	for _, exchange := range result.Exchanges {
		result.Diagnostics.TotalScore += exchange.Value
		result.Diagnostics.TotalVolume += exchange.Volume
		result.Diagnostics.TotalMatched += exchange.Matched
		totalCapacity += exchange.Capacity
		totalVolumeCapacity += exchange.VolumeCapacity
		totalMarketFit += exchange.AvgMarketFit
	}
	n := float64(len(result.Exchanges))
	result.Diagnostics.AvgCapacity = totalCapacity / n
	result.Diagnostics.AvgVolumeCapacity = totalVolumeCapacity / n
	result.Diagnostics.AvgMarketFit = totalMarketFit / n
}

func resolvedRouteID(routeID, fallback int) int {
	if routeID != 0 {
		return routeID
	}
	return fallback
}
