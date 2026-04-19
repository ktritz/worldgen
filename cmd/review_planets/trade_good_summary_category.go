package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

type summaryLabelValue struct {
	label string
	value float64
}

type categoryPairFlow struct {
	label  string
	score  float64
	volume float64
}

func printTradeMarketCategorySummary(result *climgen.TradeNodeMarketResult, network *climgen.SettlementNetworkResult, settings climgen.TradeGoodsSettings, category string) {
	exporters := topTradeNodeMarketsByCategory(result, network, settings, category, true, false, 3)
	importers := topTradeNodeMarketsByCategory(result, network, settings, category, false, false, 3)
	makers := topTradeNodeMarketsByCategory(result, network, settings, category, true, true, 3)
	if len(exporters) == 0 && len(importers) == 0 && len(makers) == 0 {
		return
	}
	fmt.Printf(
		"      categoryMarket[%s]: exports=%s imports=%s made=%s\n",
		category,
		formatSummaryLabelValues(exporters, 3),
		formatSummaryLabelValues(importers, 3),
		formatSummaryLabelValues(makers, 3),
	)
}

func topTradeNodeMarketsByCategory(result *climgen.TradeNodeMarketResult, network *climgen.SettlementNetworkResult, settings climgen.TradeGoodsSettings, category string, exports bool, manufactured bool, limit int) []summaryLabelValue {
	if result == nil || len(result.Markets) == 0 {
		return nil
	}
	specByGood := tradeGoodCategoryLookup(settings)
	values := make([]summaryLabelValue, 0, len(result.Markets))
	for _, market := range result.Markets {
		total := 0.0
		switch {
		case manufactured:
			for good, value := range market.Manufactured {
				if value <= 0 || specByGood[good] != category {
					continue
				}
				total += value
			}
		case exports:
			for good, value := range market.Surplus {
				if value <= 0 || specByGood[good] != category {
					continue
				}
				total += value
			}
		default:
			for good, value := range market.Surplus {
				if value >= 0 || specByGood[good] != category {
					continue
				}
				total += -value
			}
		}
		if total <= 0 {
			continue
		}
		values = append(values, summaryLabelValue{
			label: formatChainNodeLabel(market.NodeID, network),
			value: total,
		})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].value != values[j].value {
			return values[i].value > values[j].value
		}
		return values[i].label < values[j].label
	})
	if len(values) < limit {
		limit = len(values)
	}
	return values[:limit]
}

func tradeGoodCategoryLookup(settings climgen.TradeGoodsSettings) map[string]string {
	specByGood := make(map[string]string, len(settings.Goods))
	for _, spec := range settings.Goods {
		specByGood[spec.Name] = spec.Category
	}
	return specByGood
}

func formatSummaryLabelValues(values []summaryLabelValue, limit int) string {
	if len(values) == 0 {
		return "none"
	}
	if len(values) < limit {
		limit = len(values)
	}
	out := ""
	for i := 0; i < limit; i++ {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%s:%.2f", values[i].label, values[i].value)
	}
	return out
}

func printTradeFlowCategorySummary(result *climgen.MultimodalTradeResult, settings climgen.TradeGoodsSettings, category string) {
	score, volume, pairs := tradeFlowCategoryTotals(result, settings, category)
	if score <= 0 && volume <= 0 {
		return
	}
	fmt.Printf(
		"      categoryFlow[%s]: score=%.2f volume=%.2f pairs=%s\n",
		category,
		score,
		volume,
		formatCategoryPairFlows(pairs, 3),
	)
}

func tradeFlowCategoryTotals(result *climgen.MultimodalTradeResult, settings climgen.TradeGoodsSettings, category string) (float64, float64, []categoryPairFlow) {
	if result == nil || len(result.Pairs) == 0 {
		return 0, 0, nil
	}
	specByGood := tradeGoodCategoryLookup(settings)
	score := 0.0
	volume := 0.0
	pairs := make([]categoryPairFlow, 0, len(result.Pairs))
	for _, pair := range result.Pairs {
		pairScore := 0.0
		pairVolume := 0.0
		for _, good := range pair.Goods {
			if specByGood[good.Good] != category {
				continue
			}
			pairScore += good.Score
			pairVolume += good.Volume
		}
		if pairScore <= 0 && pairVolume <= 0 {
			continue
		}
		score += pairScore
		volume += pairVolume
		pairs = append(pairs, categoryPairFlow{
			label:  fmt.Sprintf("%d->%d", pair.FromPolity, pair.ToPolity),
			score:  pairScore,
			volume: pairVolume,
		})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].score != pairs[j].score {
			return pairs[i].score > pairs[j].score
		}
		if pairs[i].volume != pairs[j].volume {
			return pairs[i].volume > pairs[j].volume
		}
		return pairs[i].label < pairs[j].label
	})
	return score, volume, pairs
}

func formatCategoryPairFlows(values []categoryPairFlow, limit int) string {
	if len(values) == 0 {
		return "none"
	}
	if len(values) < limit {
		limit = len(values)
	}
	out := ""
	for i := 0; i < limit; i++ {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%s:s%.2f/v%.2f", values[i].label, values[i].score, values[i].volume)
	}
	return out
}
