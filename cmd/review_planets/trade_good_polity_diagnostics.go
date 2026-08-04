package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

type polityGoodAggregate struct {
	good        string
	category    string
	supply      float64
	demand      float64
	surplus     float64
	deficit     float64
	exporters   int
	importers   int
	maxExport   float64
	maxImport   float64
	scarcity    float64
	hasScarcity bool
}

func printPolityGoodAggregateDiagnostics(result *climgen.PolityGoodsResult, settings climgen.TradeGoodsSettings) {
	if result == nil || len(result.Balances) == 0 || len(settings.Goods) == 0 {
		return
	}

	goods := make([]polityGoodAggregate, 0, len(settings.Goods))
	categoryTotals := make(map[string]*polityGoodAggregate)
	for _, spec := range settings.Goods {
		agg := polityGoodAggregate{
			good:     spec.Name,
			category: spec.Category,
		}
		if result.GlobalScarcityByGood != nil {
			if scarcity, ok := result.GlobalScarcityByGood[spec.Name]; ok {
				agg.scarcity = scarcity
				agg.hasScarcity = true
			}
		}
		for _, balance := range result.Balances {
			supply := balance.Supply[spec.Name]
			demand := balance.Demand[spec.Name]
			net := balance.Surplus[spec.Name]
			agg.supply += supply
			agg.demand += demand
			if net > 0 {
				agg.surplus += net
				agg.exporters++
				if net > agg.maxExport {
					agg.maxExport = net
				}
			} else if net < 0 {
				deficit := -net
				agg.deficit += deficit
				agg.importers++
				if deficit > agg.maxImport {
					agg.maxImport = deficit
				}
			}
		}
		goods = append(goods, agg)
		category := categoryTotals[spec.Category]
		if category == nil {
			category = &polityGoodAggregate{category: spec.Category}
			categoryTotals[spec.Category] = category
		}
		category.supply += agg.supply
		category.demand += agg.demand
		category.surplus += agg.surplus
		category.deficit += agg.deficit
		category.exporters += agg.exporters
		category.importers += agg.importers
		if agg.maxExport > category.maxExport {
			category.maxExport = agg.maxExport
		}
		if agg.maxImport > category.maxImport {
			category.maxImport = agg.maxImport
		}
	}

	sort.Slice(goods, func(i, j int) bool {
		if goods[i].category != goods[j].category {
			return goods[i].category < goods[j].category
		}
		return goods[i].good < goods[j].good
	})
	for _, agg := range goods {
		scarcity := "n/a"
		if agg.hasScarcity {
			scarcity = fmt.Sprintf("%.2f", agg.scarcity)
		}
		fmt.Printf(
			"      polityGoodAggregate[%s]: category=%s supply=%.2f demand=%.2f surplus=%.2f deficit=%.2f exporters=%d importers=%d maxExport=%.2f maxImport=%.2f scarcity=%s\n",
			agg.good,
			agg.category,
			agg.supply,
			agg.demand,
			agg.surplus,
			agg.deficit,
			agg.exporters,
			agg.importers,
			agg.maxExport,
			agg.maxImport,
			scarcity,
		)
	}

	categories := make([]string, 0, len(categoryTotals))
	for category := range categoryTotals {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	for _, category := range categories {
		agg := categoryTotals[category]
		fmt.Printf(
			"      polityCategoryAggregate[%s]: supply=%.2f demand=%.2f surplus=%.2f deficit=%.2f exporters=%d importers=%d maxExport=%.2f maxImport=%.2f\n",
			category,
			agg.supply,
			agg.demand,
			agg.surplus,
			agg.deficit,
			agg.exporters,
			agg.importers,
			agg.maxExport,
			agg.maxImport,
		)
	}
}
