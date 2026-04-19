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

type categoryValue struct {
	category string
	score    float64
	volume   float64
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

func printTradeFlowCategoryMix(result *climgen.MultimodalTradeResult, settings climgen.TradeGoodsSettings) {
	values := tradeFlowCategoryMix(result, settings)
	if len(values) == 0 {
		return
	}
	fmt.Printf("      categoryFlowMix=%s\n", formatCategoryValues(values, 5))
}

func printTradeFlowCategoryDiagnostics(result *climgen.MultimodalTradeResult) {
	values := tradeFlowCategoryDiagnostics(result)
	for _, value := range values {
		fmt.Printf(
			"      categoryDiag[%s]: cand=%d accepted=%d score=%.2f volume=%.2f noSurplus=%d noNeed=%d noEndpoint=%d srcCap=%d needCap=%d lowCap=%d lowFit=%d lowScore=%d\n",
			value.Category,
			value.CandidateGoods,
			value.AcceptedGoods,
			value.TotalScore,
			value.TotalVolume,
			value.NoSourceSurplus,
			value.NoSinkNeed,
			value.NoEndpointSupply,
			value.SourceConstrained,
			value.NeedConstrained,
			value.LowCapacity,
			value.LowMarketFit,
			value.LowScoreFiltered,
		)
	}
}

func printTradeFlowCategoryModeSummary(result *climgen.MultimodalTradeResult, settings climgen.TradeGoodsSettings, category string) {
	values := tradeFlowCategoryModeMix(result, settings, category)
	if len(values) == 0 {
		return
	}
	fmt.Printf("      categoryMode[%s]=%s\n", category, formatModeValues(values, 4))
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

func tradeFlowCategoryMix(result *climgen.MultimodalTradeResult, settings climgen.TradeGoodsSettings) []categoryValue {
	if result == nil || len(result.Pairs) == 0 {
		return nil
	}
	specByGood := tradeGoodCategoryLookup(settings)
	totals := map[string]categoryValue{}
	for _, pair := range result.Pairs {
		for _, good := range pair.Goods {
			category := specByGood[good.Good]
			if category == "" {
				continue
			}
			entry := totals[category]
			entry.category = category
			entry.score += good.Score
			entry.volume += good.Volume
			totals[category] = entry
		}
	}
	values := make([]categoryValue, 0, len(totals))
	for _, entry := range totals {
		if entry.score <= 0 && entry.volume <= 0 {
			continue
		}
		values = append(values, entry)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].score != values[j].score {
			return values[i].score > values[j].score
		}
		if values[i].volume != values[j].volume {
			return values[i].volume > values[j].volume
		}
		return values[i].category < values[j].category
	})
	return values
}

func tradeFlowCategoryModeMix(result *climgen.MultimodalTradeResult, settings climgen.TradeGoodsSettings, category string) []climgen.TradeModeValue {
	if result == nil || len(result.Exchanges) == 0 {
		return nil
	}
	specByGood := tradeGoodCategoryLookup(settings)
	totals := map[string]float64{}
	for _, exchange := range result.Exchanges {
		modeTotal := 0.0
		for _, good := range exchange.Goods {
			if specByGood[good.Good] != category {
				continue
			}
			modeTotal += good.Score
		}
		if modeTotal <= 0 {
			continue
		}
		totals[exchange.Mode] += modeTotal
	}
	return topSummaryModeValues(totals, len(totals))
}

func tradeFlowCategoryDiagnostics(result *climgen.MultimodalTradeResult) []climgen.MultimodalTradeCategoryDiagnostics {
	if result == nil || len(result.Diagnostics.ByCategory) == 0 {
		return nil
	}
	values := make([]climgen.MultimodalTradeCategoryDiagnostics, 0, len(result.Diagnostics.ByCategory))
	for _, value := range result.Diagnostics.ByCategory {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].TotalScore != values[j].TotalScore {
			return values[i].TotalScore > values[j].TotalScore
		}
		if values[i].AcceptedGoods != values[j].AcceptedGoods {
			return values[i].AcceptedGoods > values[j].AcceptedGoods
		}
		return values[i].Category < values[j].Category
	})
	return values
}

func formatCategoryValues(values []categoryValue, limit int) string {
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
		out += fmt.Sprintf("%s:s%.2f/v%.2f", values[i].category, values[i].score, values[i].volume)
	}
	return out
}
