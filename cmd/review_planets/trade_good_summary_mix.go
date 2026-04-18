package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func tradeBalanceConcentration(surplus map[string]float64, settings climgen.TradeGoodsSettings, exports bool) (float64, float64, string) {
	type line struct {
		good     string
		value    float64
		category string
	}
	specByGood := map[string]string{}
	for _, spec := range settings.Goods {
		specByGood[spec.Name] = spec.Category
	}
	lines := make([]line, 0, len(surplus))
	total := 0.0
	categoryTotals := map[string]float64{}
	for good, value := range surplus {
		if exports {
			if value <= 0 {
				continue
			}
		} else {
			if value >= 0 {
				continue
			}
			value = -value
		}
		category := specByGood[good]
		lines = append(lines, line{good: good, value: value, category: category})
		total += value
		categoryTotals[category] += value
	}
	if total <= 0 {
		return 0, 0, "none"
	}
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].value != lines[j].value {
			return lines[i].value > lines[j].value
		}
		return lines[i].good < lines[j].good
	})
	top1 := lines[0].value / total
	top3Value := 0.0
	for i := 0; i < len(lines) && i < 3; i++ {
		top3Value += lines[i].value
	}
	return top1, top3Value / total, formatCategoryMix(categoryTotals, total)
}

func formatCategoryMix(totals map[string]float64, total float64) string {
	if total <= 0 || len(totals) == 0 {
		return "none"
	}
	type mix struct {
		category string
		value    float64
	}
	lines := make([]mix, 0, len(totals))
	for category, value := range totals {
		if value > 0 {
			lines = append(lines, mix{category: category, value: value})
		}
	}
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].value != lines[j].value {
			return lines[i].value > lines[j].value
		}
		return lines[i].category < lines[j].category
	})
	if len(lines) > 3 {
		lines = lines[:3]
	}
	out := ""
	for i, line := range lines {
		if i > 0 {
			out += ","
		}
		out += fmt.Sprintf("%s:%.2f", line.category, line.value/total)
	}
	return out
}

func manufacturedExportShare(market climgen.TradeNodeMarket, settings climgen.TradeGoodsSettings) float64 {
	specByGood := map[string]string{}
	for _, spec := range settings.Goods {
		specByGood[spec.Name] = spec.Category
	}
	total := 0.0
	manufactured := 0.0
	for good, value := range market.Surplus {
		if value <= 0 {
			continue
		}
		total += value
		switch specByGood[good] {
		case "processed", "finished", "luxury", "strategic":
			manufactured += value
		}
	}
	if total <= 0 {
		return 0
	}
	return manufactured / total
}
