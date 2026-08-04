package climgen

import "strings"

func marketManufacturingEligible(spec TradeGoodSpec, node SettlementNode) bool {
	minKind, ok := tradeGoodsMarketMinNodeKind(spec)
	if !ok {
		return true
	}
	return settlementNodeEffectiveRank(node) >= float64(minKind)
}

func marketManufacturingEligibleForMarket(spec TradeGoodSpec, node SettlementNode, market *TradeNodeMarket) bool {
	if !marketManufacturingEligible(spec, node) {
		return false
	}
	return marketManufacturingInputCapable(spec, market)
}

func marketManufacturingInputCapable(spec TradeGoodSpec, market *TradeNodeMarket) bool {
	if market == nil || len(spec.MarketInputCapabilityFloor) == 0 {
		return true
	}
	for input, floor := range spec.MarketInputCapabilityFloor {
		if floor <= 0 {
			continue
		}
		need := spec.Inputs[input]
		if need <= 0 {
			continue
		}
		available := maxFloat(market.Supply[input]-market.Demand[input], 0)
		if clamp01(available/need) < floor {
			return false
		}
	}
	return true
}

func tradeGoodsMarketMinNodeKind(spec TradeGoodSpec) (SettlementNodeKind, bool) {
	if spec.MarketMinNodeKind == "" {
		return 0, false
	}
	switch strings.ToLower(strings.TrimSpace(spec.MarketMinNodeKind)) {
	case "hamlet", "local_anchor", "local anchor":
		return SettlementNodeHamlet, true
	case "village", "district_anchor", "district anchor":
		return SettlementNodeVillage, true
	case "town", "regional_anchor", "regional anchor":
		return SettlementNodeTown, true
	case "city", "major_hub", "major hub":
		return SettlementNodeCity, true
	default:
		return 0, false
	}
}
