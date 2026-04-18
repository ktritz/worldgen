package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func leadingGoodValue(values []climgen.PolityGoodValue) float64 {
	if len(values) == 0 {
		return 0
	}
	return values[0].Value
}

func formatGoodValues(values []climgen.PolityGoodValue, limit int) string {
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
		out += fmt.Sprintf("%s:%.2f", values[i].Good, values[i].Value)
	}
	return out
}

func topSummaryModeValues(values map[string]float64, limit int) []climgen.TradeModeValue {
	out := make([]climgen.TradeModeValue, 0, len(values))
	for mode, value := range values {
		if value > 0 {
			out = append(out, climgen.TradeModeValue{Mode: mode, Value: value})
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

func formatModeValues(values []climgen.TradeModeValue, limit int) string {
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
		out += fmt.Sprintf("%s:%.2f", values[i].Mode, values[i].Value)
	}
	return out
}

func formatTradeFlowGoods(values []climgen.TradeGoodFlowValue, limit int) string {
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
		out += fmt.Sprintf("%s:s%.3f/v%.2f", values[i].Good, values[i].Score, values[i].Volume)
	}
	return out
}

func leadingTradeFlowFactor(values []climgen.TradeGoodFlowValue, selector func(climgen.TradeGoodFlowValue) float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return selector(values[0])
}

func topMetricMap(values map[string]float64, limit int) []tradeGoodMetricLine {
	out := make([]tradeGoodMetricLine, 0, len(values))
	for name, value := range values {
		if value > 0 {
			out = append(out, tradeGoodMetricLine{name: name, value: value})
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

func formatMetricValues(values []tradeGoodMetricLine, limit int) string {
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
		out += fmt.Sprintf("%s:%.2f", values[i].name, values[i].value)
	}
	return out
}
