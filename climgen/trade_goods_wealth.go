package climgen

func estimateNodeWealth(
	node SettlementNode,
	cells []VoronoiCell,
	settings TradeGoodsSettings,
	endowmentByGood map[string]TradeGoodEndowment,
	trade *TradeNetworkResult,
) float64 {
	kindScale := nodeKindScale(node)
	urbanity := nodeUrbanity(node)
	tradeAccess := nodeTradeAccess(node, trade)
	rawOpportunity := nodeRawExportOpportunity(node, cells, settings, endowmentByGood)
	logistics := clamp01(0.58*tradeAccess + 0.18*bool01(node.Coastal) + 0.14*bool01(node.River) + 0.10*kindScale)
	return clamp01(
		0.26*urbanity +
			0.18*clamp01(node.CarryingCapacity) +
			0.22*logistics +
			0.22*rawOpportunity +
			0.12*kindScale,
	)
}

func nodeRawExportOpportunity(
	node SettlementNode,
	cells []VoronoiCell,
	settings TradeGoodsSettings,
	endowmentByGood map[string]TradeGoodEndowment,
) float64 {
	total := 0.0
	weight := 0.0
	for _, spec := range settings.Goods {
		if len(spec.Inputs) > 0 {
			continue
		}
		endowment, ok := endowmentByGood[spec.Name]
		if !ok {
			continue
		}
		localPotential := nodeCatchmentPotential(cells, node.CellIndex, nodeCatchmentRadius(node), endowment.Potential)
		valueWeight := 0.30 + 0.70*spec.BaseValue
		total += valueWeight * localPotential
		weight += valueWeight
	}
	if weight == 0 {
		return 0
	}
	return clamp01(total / weight)
}

func nodeWealthDemand(spec TradeGoodSpec, wealth float64) float64 {
	wealth = clamp01(wealth)
	switch spec.Category {
	case "raw":
		return clamp01(0.24 + 0.28*wealth + 0.16*spec.BaseValue)
	case "processed":
		return clamp01(0.20 + 0.60*wealth + 0.20*spec.BaseValue)
	case "finished":
		return clamp01(0.16 + 0.72*wealth + 0.18*spec.BaseValue)
	case "luxury":
		return clamp01(0.10 + 0.84*wealth + 0.14*spec.BaseValue)
	case "strategic":
		return clamp01(0.18 + 0.66*wealth + 0.20*spec.BaseValue)
	default:
		return clamp01(0.20 + 0.55*wealth + 0.15*spec.BaseValue)
	}
}

func nodeDemandScale(spec TradeGoodSpec, node SettlementNode, wealth float64, trade *TradeNetworkResult) float64 {
	wealth = clamp01(wealth)
	urbanity := nodeUrbanity(node)
	tradeAccess := nodeTradeAccess(node, trade)
	kindScale := nodeKindScale(node)
	switch spec.Category {
	case "raw":
		return clamp01(0.68 + 0.16*kindScale + 0.16*wealth)
	case "processed":
		return clamp01(0.34 + 0.30*urbanity + 0.20*kindScale + 0.16*wealth)
	case "finished":
		return clamp01(0.16 + 0.38*urbanity + 0.16*tradeAccess + 0.16*kindScale + 0.14*wealth)
	case "luxury":
		return clamp01(0.08 + 0.34*urbanity + 0.22*tradeAccess + 0.14*kindScale + 0.22*wealth)
	case "strategic":
		return clamp01(0.12 + 0.28*urbanity + 0.18*tradeAccess + 0.24*kindScale + 0.18*wealth)
	default:
		return clamp01(0.20 + 0.32*urbanity + 0.16*tradeAccess + 0.16*wealth)
	}
}

func polityWealthDemand(spec TradeGoodSpec, wealth float64) float64 {
	wealth = clamp01(wealth)
	switch spec.Category {
	case "raw":
		return clamp01(0.28 + 0.24*wealth + 0.12*spec.BaseValue)
	case "processed":
		return clamp01(0.22 + 0.54*wealth + 0.18*spec.BaseValue)
	case "finished":
		return clamp01(0.18 + 0.66*wealth + 0.18*spec.BaseValue)
	case "luxury":
		return clamp01(0.10 + 0.82*wealth + 0.12*spec.BaseValue)
	case "strategic":
		return clamp01(0.18 + 0.62*wealth + 0.18*spec.BaseValue)
	default:
		return clamp01(0.20 + 0.50*wealth + 0.15*spec.BaseValue)
	}
}

func aggregatePolityMarketWealth(result *NodeGoodsResult, network *SettlementNetworkResult, trade *TradeNetworkResult) map[int]float64 {
	out := map[int]float64{}
	if result == nil || network == nil {
		return out
	}
	totalWeight := map[int]float64{}
	for _, balance := range result.Balances {
		if balance.PolityID < 0 || balance.NodeID < 0 || balance.NodeID >= len(network.Nodes) {
			continue
		}
		node := network.Nodes[balance.NodeID]
		weight := 0.34 + 0.34*nodeKindScale(node) + 0.20*nodeTradeAccess(node, trade) + 0.12*nodeUrbanity(node)
		out[balance.PolityID] += balance.Wealth * weight
		totalWeight[balance.PolityID] += weight
	}
	for polityID, total := range out {
		weight := totalWeight[polityID]
		if weight > 0 {
			out[polityID] = clamp01(total / weight)
		}
	}
	return out
}
