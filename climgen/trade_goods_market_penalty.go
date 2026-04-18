package climgen

func marketDominancePenaltyMultiplier(
	spec TradeGoodSpec,
	market *TradeNodeMarket,
	settings TradeGoodsSettings,
	production TradeGoodsProductionSettings,
) (float64, float64) {
	if market == nil {
		return 1, 0
	}
	goodPenalty := spec.MarketDominancePenalty
	if goodPenalty <= 0 {
		goodPenalty = tradeGoodsCategorySetting(production.MarketDominancePenalty, spec.Category, 0)
	}
	if goodPenalty <= 0 {
		return 1, 0
	}
	goodOutput := clamp01(market.Manufactured[spec.Name] / 4.0)
	categoryOutput := clamp01(marketManufacturedCategoryTotal(market, spec.Category, settings) / 8.0)
	pressure := clamp01(0.65*goodOutput + 0.35*categoryOutput)
	penalty := Clamp(1-goodPenalty*pressure, 0.35, 1.0)
	return penalty, 1 - penalty
}

func marketManufacturedCategoryTotal(market *TradeNodeMarket, category string, settings TradeGoodsSettings) float64 {
	if market == nil || len(market.Manufactured) == 0 || category == "" {
		return 0
	}
	categoryByGood := map[string]string{}
	for _, spec := range settings.Goods {
		categoryByGood[spec.Name] = spec.Category
	}
	total := 0.0
	for good, value := range market.Manufactured {
		if value <= 0 {
			continue
		}
		if categoryByGood[good] == category {
			total += value
		}
	}
	return total
}
