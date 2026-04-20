package climgen

func tradeGoodsHasLocalInputCapability(spec TradeGoodSpec, currentSupply map[string]float64) bool {
	if len(spec.LocalInputCapabilityFloor) == 0 {
		return true
	}
	for input, floor := range spec.LocalInputCapabilityFloor {
		if floor <= 0 {
			continue
		}
		if currentSupply[input] < floor {
			return false
		}
	}
	return true
}
