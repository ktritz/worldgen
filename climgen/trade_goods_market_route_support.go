package climgen

func applyRouteAwareMarketExternalRawInputSupport(
	markets []TradeNodeMarket,
	settings TradeGoodsSettings,
	polities *PolitySphereResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	river *RiverTradeResult,
	coastal *CoastalTradeResult,
	ocean *OceanTradeResult,
	profileByPolity map[int]PolityProfileAssignment,
	polityGoods *PolityGoodsResult,
	production TradeGoodsProductionSettings,
) {
	if len(markets) == 0 || network == nil || polities == nil || polityGoods == nil {
		return
	}
	specByName := map[string]TradeGoodSpec{}
	for _, spec := range settings.Goods {
		specByName[spec.Name] = spec
	}
	balanceByPolity := map[int]PolityGoodBalance{}
	for _, balance := range polityGoods.Balances {
		balanceByPolity[balance.PolityID] = balance
	}
	connectivity := polityTradeConnectivity(polities, network, trade, river, coastal, ocean)
	inputNeedByMarket := map[int]map[string]float64{}
	byPolity := map[int][]int{}
	for i := range markets {
		market := &markets[i]
		if market.NodeID < 0 || market.NodeID >= len(network.Nodes) {
			continue
		}
		node := network.Nodes[market.NodeID]
		inputNeedByMarket[i] = marketEligibleManufacturingRawInputNeed(
			market,
			settings,
			node,
			trade,
			profileByPolity[market.PolityID],
		)
		byPolity[market.PolityID] = append(byPolity[market.PolityID], i)
	}
	for polityID, indices := range byPolity {
		for input, spec := range specByName {
			if spec.Category != "raw" {
				continue
			}
			supportScale := tradeGoodsExternalInputSupportScale(production, input, spec.Category)
			if supportScale <= 0 {
				continue
			}
			compactness := clamp01(0.60*(1-spec.Bulkiness) + 0.40*(1-spec.Perishability))
			if compactness <= 0.25 {
				continue
			}
			donorSupport := 0.0
			for donorID, donor := range balanceByPolity {
				if donorID == polityID {
					continue
				}
				access := connectivity[[2]int{donorID, polityID}]
				if access <= 0 {
					continue
				}
				surplus := maxFloat(donor.Surplus[input], 0)
				if surplus <= 0 {
					continue
				}
				donorSupport += surplus * Clamp(0.35+0.65*access, 0.0, 1.0)
			}
			if donorSupport <= 0.01 {
				continue
			}
			needs := map[int]float64{}
			totalNeed := 0.0
			for _, idx := range indices {
				market := &markets[idx]
				node := network.Nodes[market.NodeID]
				rawNeed := 0.0
				if inputNeed, ok := inputNeedByMarket[idx]; ok {
					rawNeed = inputNeed[input]
				}
				deficit := maxFloat(market.Demand[input]-market.Supply[input], 0)
				weightedNeed := rawNeed + 0.60*deficit
				if weightedNeed <= 0.01 {
					continue
				}
				localAccess := clamp01(0.55*nodeTradeAccess(node, trade) + 0.25*nodeKindScale(node) + 0.20*clamp01(market.Wealth))
				weight := weightedNeed * clamp01(0.35+0.65*localAccess)
				if weight <= 0.01 {
					continue
				}
				needs[idx] = weight
				totalNeed += weight
			}
			if totalNeed <= 0.01 {
				continue
			}
			compactSupport := Clamp(0.55+0.45*compactness, 0.0, 1.0)
			transfer := minFloat(donorSupport*supportScale*compactSupport, totalNeed)
			if transfer <= 0.01 {
				continue
			}
			for idx, weight := range needs {
				share := transfer * weight / totalNeed
				markets[idx].Supply[input] += share
			}
		}
	}
}

func polityTradeConnectivity(
	polities *PolitySphereResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	river *RiverTradeResult,
	coastal *CoastalTradeResult,
	ocean *OceanTradeResult,
) map[[2]int]float64 {
	out := map[[2]int]float64{}
	if polities == nil || network == nil {
		return out
	}
	if trade != nil {
		for _, corridor := range trade.Corridors {
			if corridor.Role == TradeCorridorRoleFeeder {
				continue
			}
			from := tradeNodePolity(corridor.FromNode, polities, network)
			to := tradeNodePolity(corridor.ToNode, polities, network)
			if from < 0 || to < 0 || from == to {
				continue
			}
			quality := (0.72 + 0.28*clamp01(corridor.MeanSupport)) * (1.0 - 0.42*clamp01(corridor.MeanRisk))
			weight := clamp01(corridor.Flow * quality)
			out[[2]int{from, to}] = clamp01(out[[2]int{from, to}] + weight)
			out[[2]int{to, from}] = clamp01(out[[2]int{to, from}] + weight)
		}
	}
	if river != nil {
		for _, corridor := range river.Corridors {
			from := tradeNodePolity(corridor.FromNode, polities, network)
			to := tradeNodePolity(corridor.ToNode, polities, network)
			if from < 0 || to < 0 || from == to {
				continue
			}
			quality := 0.58 + 0.27*clamp01(corridor.MeanNavigability) + 0.15*clamp01(corridor.MeanTransfer)
			weight := clamp01(corridor.Flow * quality)
			out[[2]int{from, to}] = clamp01(out[[2]int{from, to}] + weight)
			out[[2]int{to, from}] = clamp01(out[[2]int{to, from}] + weight)
		}
	}
	if coastal != nil {
		for _, corridor := range coastal.Corridors {
			from := tradeNodePolity(corridor.FromNode, polities, network)
			to := tradeNodePolity(corridor.ToNode, polities, network)
			if from < 0 || to < 0 || from == to {
				continue
			}
			quality := (0.90 + 0.16*clamp01(corridor.MeanCurrentAssist)) * (1.0 - 0.30*clamp01(corridor.MeanExposure))
			weight := clamp01(corridor.Flow * quality)
			out[[2]int{from, to}] = clamp01(out[[2]int{from, to}] + weight)
			out[[2]int{to, from}] = clamp01(out[[2]int{to, from}] + weight)
		}
	}
	if ocean != nil {
		for _, corridor := range ocean.Corridors {
			from := tradeNodePolity(corridor.FromNode, polities, network)
			to := tradeNodePolity(corridor.ToNode, polities, network)
			if from < 0 || to < 0 || from == to {
				continue
			}
			quality := (0.84 + 0.20*clamp01(corridor.MeanCurrentAssist)) * (1.0 - 0.38*clamp01(corridor.MeanExposure))
			weight := clamp01(corridor.Flow * quality)
			out[[2]int{from, to}] = clamp01(out[[2]int{from, to}] + weight)
			out[[2]int{to, from}] = clamp01(out[[2]int{to, from}] + weight)
		}
	}
	return out
}
