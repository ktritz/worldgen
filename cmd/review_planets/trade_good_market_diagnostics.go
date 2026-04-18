package main

import (
	"fmt"

	"worldgen/climgen"
)

func formatMarketManufacturingDiagnostics(diag climgen.TradeNodeMarketManufacturingDiagnostics) string {
	return fmt.Sprintf(
		"cand=%d made=%d noInput=%d noCap=%d lowYield=%d",
		diag.CandidateCount,
		diag.ProducedCount,
		diag.NoInputCount,
		diag.NoCapacityCount,
		diag.LowYieldCount,
	)
}

func formatMarketManufacturingCandidates(values []climgen.TradeNodeMarketManufacturingCandidate, limit int) string {
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
		score := values[i].Priority
		if score <= 0 {
			score = reviewMaxFloat(values[i].DemandGap, values[i].Potential)
		}
		out += fmt.Sprintf("%s:p%.2f@%s(%.2f/%.2f eff=%.2f)", values[i].Good, score, values[i].Bottleneck, values[i].BottleneckAvailable, values[i].BottleneckNeed, values[i].BottleneckEffective)
	}
	return out
}

func manufacturingCategoryMix(values map[string]float64, settings climgen.TradeGoodsSettings) string {
	if len(values) == 0 {
		return "none"
	}
	categoryByGood := map[string]string{}
	for _, spec := range settings.Goods {
		categoryByGood[spec.Name] = spec.Category
	}
	totals := map[string]float64{}
	total := 0.0
	for good, value := range values {
		if value <= 0 {
			continue
		}
		category := categoryByGood[good]
		totals[category] += value
		total += value
	}
	return formatCategoryMix(totals, total)
}

func candidateCategoryMix(values []climgen.TradeNodeMarketManufacturingCandidate, settings climgen.TradeGoodsSettings) string {
	if len(values) == 0 {
		return "none"
	}
	categoryByGood := map[string]string{}
	for _, spec := range settings.Goods {
		categoryByGood[spec.Name] = spec.Category
	}
	totals := map[string]float64{}
	total := 0.0
	for _, value := range values {
		weight := value.DemandGap
		if weight <= 0 {
			weight = value.Potential
		}
		if weight <= 0 {
			continue
		}
		category := categoryByGood[value.Good]
		totals[category] += weight
		total += weight
	}
	return formatCategoryMix(totals, total)
}

func penalizedCategoryMix(values map[string]float64, settings climgen.TradeGoodsSettings) string {
	if len(values) == 0 {
		return "none"
	}
	categoryByGood := map[string]string{}
	for _, spec := range settings.Goods {
		categoryByGood[spec.Name] = spec.Category
	}
	totals := map[string]float64{}
	total := 0.0
	for good, value := range values {
		if value <= 0 {
			continue
		}
		category := categoryByGood[good]
		totals[category] += value
		total += value
	}
	return formatCategoryMix(totals, total)
}
