package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

type namedMetric struct {
	name  string
	value float64
}

func printTradeNodeMarketAggregateDiagnostics(result *climgen.TradeNodeMarketResult, settings climgen.TradeGoodsSettings) {
	if result == nil || len(result.Markets) == 0 {
		return
	}
	totalCand := 0
	totalMade := 0
	totalNoInput := 0
	totalNoCap := 0
	totalLowYield := 0
	blockedGoods := map[string]float64{}
	blockedInputs := map[string]float64{}
	latentGoods := map[string]float64{}
	madeGoods := map[string]float64{}
	usedInputs := map[string]float64{}
	madeCategories := map[string]float64{}
	blockedCategories := map[string]float64{}
	penalizedGoods := map[string]float64{}
	penalizedCategories := map[string]float64{}
	categoryByGood := tradeGoodCategoryByName(settings)
	for _, market := range result.Markets {
		diag := market.Diagnostics
		totalCand += diag.CandidateCount
		totalMade += diag.ProducedCount
		totalNoInput += diag.NoInputCount
		totalNoCap += diag.NoCapacityCount
		totalLowYield += diag.LowYieldCount
		for _, blocked := range diag.Blocked {
			blockedGoods[blocked.Good] += reviewMaxFloat(blocked.DemandGap, blocked.Potential)
			if category := categoryByGood[blocked.Good]; category != "" {
				blockedCategories[category] += reviewMaxFloat(blocked.DemandGap, blocked.Potential)
			}
			if blocked.Bottleneck != "" && blocked.Bottleneck != "none" {
				blockedInputs[blocked.Bottleneck] += reviewMaxFloat(blocked.DemandGap, 1)
			}
		}
		for _, latent := range diag.Latent {
			latentGoods[latent.Good] += reviewMaxFloat(latent.Potential, latent.DemandGap)
		}
		for good, value := range market.Manufactured {
			if value <= 0 {
				continue
			}
			madeGoods[good] += value
			if category := categoryByGood[good]; category != "" {
				madeCategories[category] += value
			}
		}
		for input, value := range market.Consumed {
			if value <= 0 {
				continue
			}
			usedInputs[input] += value
		}
		for good, value := range market.Diagnostics.Penalized {
			if value <= 0 {
				continue
			}
			penalizedGoods[good] += value
			if category := categoryByGood[good]; category != "" {
				penalizedCategories[category] += value
			}
		}
	}
	fmt.Printf(
		"      marketMakeDiag: cand=%d made=%d noInput=%d noCap=%d lowYield=%d madeCat=%s blockedCat=%s capWinners=%s capLosers=%s penalized=%s penalizedCat=%s blockedInputs=%s inputUse=%s latentGoods=%s\n",
		totalCand,
		totalMade,
		totalNoInput,
		totalNoCap,
		totalLowYield,
		formatNamedMetrics(topNamedMetrics(madeCategories, 4), 4),
		formatNamedMetrics(topNamedMetrics(blockedCategories, 4), 4),
		formatNamedMetrics(topNamedMetrics(madeGoods, 4), 4),
		formatNamedMetrics(topNamedMetrics(blockedGoods, 4), 4),
		formatNamedMetrics(topNamedMetrics(penalizedGoods, 4), 4),
		formatNamedMetrics(topNamedMetrics(penalizedCategories, 4), 4),
		formatNamedMetrics(topNamedMetrics(blockedInputs, 4), 4),
		formatNamedMetrics(topNamedMetrics(usedInputs, 4), 4),
		formatNamedMetrics(topNamedMetrics(latentGoods, 4), 4),
	)
}

func tradeGoodCategoryByName(settings climgen.TradeGoodsSettings) map[string]string {
	out := make(map[string]string, len(settings.Goods))
	for _, spec := range settings.Goods {
		out[spec.Name] = spec.Category
	}
	return out
}

func reviewMaxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func topNamedMetrics(values map[string]float64, limit int) []namedMetric {
	out := make([]namedMetric, 0, len(values))
	for name, value := range values {
		if value > 0 {
			out = append(out, namedMetric{name: name, value: value})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].value != out[j].value {
			return out[i].value > out[j].value
		}
		return out[i].name < out[j].name
	})
	if len(out) < limit {
		limit = len(out)
	}
	return out[:limit]
}

func formatNamedMetrics(values []namedMetric, limit int) string {
	if len(values) == 0 {
		return "none"
	}
	if len(values) < limit {
		limit = len(values)
	}
	out := ""
	for i := 0; i < limit; i++ {
		if i > 0 {
			out += ","
		}
		out += fmt.Sprintf("%s:%.0f", values[i].name, values[i].value)
	}
	return out
}
