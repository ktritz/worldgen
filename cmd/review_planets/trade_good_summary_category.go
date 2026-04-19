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
