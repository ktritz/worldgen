package climgen

import "math"

func ApplyMultimodalTradeToPolityProfiles(profiles *PolityProfileResult, trade *MultimodalTradeResult) *PolityProfileResult {
	if profiles == nil || trade == nil || len(trade.Pairs) == 0 || len(profiles.Attitudes) == 0 {
		return profiles
	}
	out := &PolityProfileResult{
		Assignments: append([]PolityProfileAssignment(nil), profiles.Assignments...),
		Attitudes:   append([]PolityAttitude(nil), profiles.Attitudes...),
	}
	pairValue := make(map[[2]int]float64, len(trade.Pairs))
	for _, pair := range trade.Pairs {
		if pair.FromPolity < 0 || pair.ToPolity < 0 || pair.FromPolity == pair.ToPolity || pair.Value <= 0 {
			continue
		}
		pairValue[[2]int{pair.FromPolity, pair.ToPolity}] += pair.Value
	}
	for i := range out.Attitudes {
		attitude := &out.Attitudes[i]
		direct := pairValue[[2]int{attitude.From, attitude.To}]
		reverse := pairValue[[2]int{attitude.To, attitude.From}]
		bonus := goodsTradeAttitudeBonus(direct, reverse)
		if bonus <= 0 {
			continue
		}
		attitude.TradeBonus += bonus
		attitude.Score += bonus
		if attitude.Stance != PolityAttitudeAllied {
			attitude.Stance = classifyDetailedPolityAttitude(attitude.Score, attitude.AllianceBonus, false)
		}
	}
	return out
}

func ComputeMultimodalTrade(
	goods *PolityGoodsResult,
	settings TradeGoodsSettings,
	polities *PolitySphereResult,
	network *SettlementNetworkResult,
	land *TradeNetworkResult,
	river *RiverTradeResult,
	coastal *CoastalTradeResult,
	ocean *OceanTradeResult,
) *MultimodalTradeResult {
	return computeMultimodalTrade(goods, settings, polities, network, land, river, coastal, ocean, nil)
}

func ComputeMultimodalTradeWithNodeMarkets(
	goods *PolityGoodsResult,
	settings TradeGoodsSettings,
	polities *PolitySphereResult,
	network *SettlementNetworkResult,
	land *TradeNetworkResult,
	river *RiverTradeResult,
	coastal *CoastalTradeResult,
	ocean *OceanTradeResult,
	nodeMarkets *TradeNodeMarketResult,
) *MultimodalTradeResult {
	return computeMultimodalTrade(goods, settings, polities, network, land, river, coastal, ocean, nodeMarkets)
}

func computeMultimodalTrade(
	goods *PolityGoodsResult,
	settings TradeGoodsSettings,
	polities *PolitySphereResult,
	network *SettlementNetworkResult,
	land *TradeNetworkResult,
	river *RiverTradeResult,
	coastal *CoastalTradeResult,
	ocean *OceanTradeResult,
	nodeMarkets *TradeNodeMarketResult,
) *MultimodalTradeResult {
	out := &MultimodalTradeResult{}
	if goods == nil || len(goods.Balances) == 0 || polities == nil || network == nil {
		return out
	}
	tuning := settings.EffectiveScarcitySettings()
	multimodal := settings.EffectiveMultimodalSettings()
	balances := tradeGoodBalanceByPolity(goods)
	specs := tradeGoodSpecByName(settings)
	marketsByNode := tradeNodeMarketByNode(nodeMarkets)

	if land != nil {
		for i, corridor := range land.Corridors {
			if corridor.Role == TradeCorridorRoleFeeder {
				continue
			}
			from := tradeNodePolity(corridor.FromNode, polities, network)
			to := tradeNodePolity(corridor.ToNode, polities, network)
			quality := (0.72 + 0.28*clamp01(corridor.MeanSupport)) * (1.0 - 0.42*clamp01(corridor.MeanRisk))
			appendRouteGoodExchanges(out, balances, specs, goods.GlobalScarcityByGood, marketsByNode, corridor.FromNode, corridor.ToNode, from, to, "land", resolvedRouteID(corridor.ID, i), corridor.Flow, corridor.TravelCost, quality, tuning, multimodal)
		}
	}
	if river != nil {
		for i, corridor := range river.Corridors {
			from := tradeNodePolity(corridor.FromNode, polities, network)
			to := tradeNodePolity(corridor.ToNode, polities, network)
			quality := 0.58 + 0.27*clamp01(corridor.MeanNavigability) + 0.15*clamp01(corridor.MeanTransfer)
			appendRouteGoodExchanges(out, balances, specs, goods.GlobalScarcityByGood, marketsByNode, corridor.FromNode, corridor.ToNode, from, to, "river", resolvedRouteID(corridor.ID, i), corridor.Flow, corridor.TravelCost, quality, tuning, multimodal)
		}
	}
	if coastal != nil {
		for i, corridor := range coastal.Corridors {
			from := tradeNodePolity(corridor.FromNode, polities, network)
			to := tradeNodePolity(corridor.ToNode, polities, network)
			quality := (0.90 + 0.16*clamp01(corridor.MeanCurrentAssist)) * (1.0 - 0.30*clamp01(corridor.MeanExposure))
			appendRouteGoodExchanges(out, balances, specs, goods.GlobalScarcityByGood, marketsByNode, corridor.FromNode, corridor.ToNode, from, to, "coastal", resolvedRouteID(corridor.ID, i), corridor.Flow, corridor.TravelCost, quality, tuning, multimodal)
		}
	}
	if ocean != nil {
		for i, corridor := range ocean.Corridors {
			from := tradeNodePolity(corridor.FromNode, polities, network)
			to := tradeNodePolity(corridor.ToNode, polities, network)
			quality := (0.84 + 0.20*clamp01(corridor.MeanCurrentAssist)) * (1.0 - 0.38*clamp01(corridor.MeanExposure))
			appendRouteGoodExchanges(out, balances, specs, goods.GlobalScarcityByGood, marketsByNode, corridor.FromNode, corridor.ToNode, from, to, "ocean", resolvedRouteID(corridor.ID, i), corridor.Flow, corridor.TravelCost, quality, tuning, multimodal)
		}
	}
	out.Pairs = aggregateTradeGoodPairs(out.Exchanges)
	populateMultimodalTradeDiagnostics(out)
	return out
}

func tradeGoodBalanceByPolity(goods *PolityGoodsResult) map[int]PolityGoodBalance {
	out := make(map[int]PolityGoodBalance, len(goods.Balances))
	for _, balance := range goods.Balances {
		out[balance.PolityID] = balance
	}
	return out
}

func tradeGoodSpecByName(settings TradeGoodsSettings) map[string]TradeGoodSpec {
	out := make(map[string]TradeGoodSpec, len(settings.Goods))
	for _, spec := range settings.Goods {
		out[spec.Name] = spec
	}
	return out
}

func goodsTradeAttitudeBonus(direct, reverse float64) float64 {
	if direct <= 0 && reverse <= 0 {
		return 0
	}
	return 0.18 * math.Tanh(9.0*math.Max(direct, 0)+5.0*math.Max(reverse, 0))
}

func tradeNodePolity(nodeIdx int, polities *PolitySphereResult, network *SettlementNetworkResult) int {
	if nodeIdx < 0 || network == nil || nodeIdx >= len(network.Nodes) || polities == nil || polities.Diagnostics == nil {
		return -1
	}
	cellIdx := network.Nodes[nodeIdx].CellIndex
	if cellIdx >= 0 && cellIdx < len(polities.Diagnostics.PolityByCell) {
		if polityID := polities.Diagnostics.PolityByCell[cellIdx]; polityID >= 0 {
			return polityID
		}
	}
	if nodeIdx < len(polities.Diagnostics.PolityByNode) {
		sphereIdx := polities.Diagnostics.PolityByNode[nodeIdx]
		if sphereIdx >= 0 && sphereIdx < len(polities.Spheres) {
			return polities.Spheres[sphereIdx].ID
		}
	}
	return -1
}
