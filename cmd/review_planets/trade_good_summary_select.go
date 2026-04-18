package main

import (
	"sort"

	"worldgen/climgen"
)

type tradeGoodEndowmentLine struct {
	name     string
	category string
	value    float64
}

type tradeGoodMetricLine struct {
	name  string
	value float64
}

func topTradeGoodEndowments(result *climgen.TradeGoodResult, limit int) []tradeGoodEndowmentLine {
	out := make([]tradeGoodEndowmentLine, 0, len(result.Goods))
	for _, good := range result.Goods {
		out = append(out, tradeGoodEndowmentLine{name: good.Good, category: good.Category, value: meanFloat(good.Potential)})
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

func topScarceTradeGoods(result *climgen.TradeGoodResult, limit int) []tradeGoodMetricLine {
	if result == nil || result.Diagnostics == nil || len(result.Diagnostics.ScarcityByGood) == 0 {
		return nil
	}
	out := make([]tradeGoodMetricLine, 0, len(result.Diagnostics.ScarcityByGood))
	for name, value := range result.Diagnostics.ScarcityByGood {
		out = append(out, tradeGoodMetricLine{name: name, value: value})
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

func topPolityGoodBalances(result *climgen.PolityGoodsResult, limit int) []climgen.PolityGoodBalance {
	out := append([]climgen.PolityGoodBalance(nil), result.Balances...)
	sort.Slice(out, func(i, j int) bool {
		return leadingGoodValue(out[i].Exports)+leadingGoodValue(out[i].Imports) > leadingGoodValue(out[j].Exports)+leadingGoodValue(out[j].Imports)
	})
	if len(out) < limit {
		limit = len(out)
	}
	return out[:limit]
}

func topNodeGoodBalances(result *climgen.NodeGoodsResult, limit int, exports bool) []climgen.NodeGoodBalance {
	out := append([]climgen.NodeGoodBalance(nil), result.Balances...)
	sort.Slice(out, func(i, j int) bool {
		left := leadingGoodValue(out[i].Imports)
		right := leadingGoodValue(out[j].Imports)
		if exports {
			left = leadingGoodValue(out[i].Exports)
			right = leadingGoodValue(out[j].Exports)
		}
		if left != right {
			return left > right
		}
		return out[i].NodeID < out[j].NodeID
	})
	if len(out) < limit {
		limit = len(out)
	}
	return out[:limit]
}

func topTradeNodeMarkets(result *climgen.TradeNodeMarketResult, limit int) []climgen.TradeNodeMarket {
	out := append([]climgen.TradeNodeMarket(nil), result.Markets...)
	sort.Slice(out, func(i, j int) bool {
		left := out[i].Wealth + 0.10*float64(out[i].FeederNodes) + leadingGoodValue(out[i].Exports)
		right := out[j].Wealth + 0.10*float64(out[j].FeederNodes) + leadingGoodValue(out[j].Exports)
		if left != right {
			return left > right
		}
		return out[i].NodeID < out[j].NodeID
	})
	if len(out) < limit {
		limit = len(out)
	}
	return out[:limit]
}
