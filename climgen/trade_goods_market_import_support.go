package climgen

func applyPolityMarketImportedInputSupport(
	markets []TradeNodeMarket,
	settings TradeGoodsSettings,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	profileByPolity map[int]PolityProfileAssignment,
	production TradeGoodsProductionSettings,
) {
	if len(markets) == 0 || network == nil {
		return
	}
	specByName := map[string]TradeGoodSpec{}
	for _, spec := range settings.Goods {
		specByName[spec.Name] = spec
	}
	byPolity := map[int][]int{}
	for i := range markets {
		byPolity[markets[i].PolityID] = append(byPolity[markets[i].PolityID], i)
	}
	for _, indices := range byPolity {
		if len(indices) < 2 {
			continue
		}
		inputNeedByMarket := map[int]map[string]float64{}
		for _, idx := range indices {
			market := &markets[idx]
			if market.NodeID < 0 || market.NodeID >= len(network.Nodes) {
				continue
			}
			inputNeedByMarket[idx] = marketEligibleManufacturingRawInputNeed(
				market,
				settings,
				network.Nodes[market.NodeID],
				trade,
				profileByPolity[market.PolityID],
			)
		}
		for input, spec := range specByName {
			if spec.Category != "raw" {
				continue
			}
			supportScale := tradeGoodsImportedInputSupportScale(production, input, spec.Category)
			if supportScale <= 0 {
				continue
			}
			compactness := clamp01(0.60*(1-spec.Bulkiness) + 0.40*(1-spec.Perishability))
			if compactness <= 0.25 {
				continue
			}
			totalMovable := 0.0
			totalNeed := 0.0
			donorMove := map[int]float64{}
			recipientNeed := map[int]float64{}
			for _, idx := range indices {
				market := &markets[idx]
				node := network.Nodes[market.NodeID]
				access := clamp01(0.55*nodeTradeAccess(node, trade) + 0.25*nodeKindScale(node) + 0.20*clamp01(market.Wealth))
				manufacturingNeed := 0.0
				if inputNeed, ok := inputNeedByMarket[idx]; ok {
					manufacturingNeed = inputNeed[input]
				}
				surplus := maxFloat(market.Supply[input]-market.Demand[input]-manufacturingNeed, 0)
				if surplus > 0.01 {
					movable := surplus * clamp01((0.30+0.70*access)*compactness*supportScale)
					if movable > 0.01 {
						donorMove[idx] = movable
						totalMovable += movable
					}
				}
				deficit := maxFloat(market.Demand[input]-market.Supply[input], 0)
				need := 0.10 * deficit
				if manufacturingNeed > 0.01 {
					need = manufacturingNeed + 0.70*deficit
				}
				if need > 0.01 {
					weightedNeed := need * clamp01(0.40+0.60*access)
					recipientNeed[idx] = weightedNeed
					totalNeed += weightedNeed
				}
			}
			if totalMovable <= 0.01 || totalNeed <= 0.01 {
				continue
			}
			transfer := minFloat(totalMovable, totalNeed)
			for idx, movable := range donorMove {
				share := transfer * movable / totalMovable
				markets[idx].Supply[input] = maxFloat(markets[idx].Supply[input]-share, 0)
			}
			for idx, need := range recipientNeed {
				share := transfer * need / totalNeed
				markets[idx].Supply[input] += share
			}
		}
	}
}

func applyMarketExternalRawInputSupport(
	markets []TradeNodeMarket,
	settings TradeGoodsSettings,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	profileByPolity map[int]PolityProfileAssignment,
	production TradeGoodsProductionSettings,
) {
	if len(markets) == 0 || network == nil {
		return
	}
	specByName := map[string]TradeGoodSpec{}
	for _, spec := range settings.Goods {
		specByName[spec.Name] = spec
	}
	for i := range markets {
		market := &markets[i]
		if market.NodeID < 0 || market.NodeID >= len(network.Nodes) {
			continue
		}
		node := network.Nodes[market.NodeID]
		access := clamp01(0.60*nodeTradeAccess(node, trade) + 0.25*nodeKindScale(node) + 0.15*clamp01(market.Wealth))
		if access <= 0.05 {
			continue
		}
		inputNeed := marketEligibleManufacturingRawInputNeed(market, settings, node, trade, profileByPolity[market.PolityID])
		for input, need := range inputNeed {
			if need <= 0.01 {
				continue
			}
			spec, ok := specByName[input]
			if !ok || spec.Category != "raw" {
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
			currentSurplus := maxFloat(market.Supply[input]-market.Demand[input], 0)
			deficit := maxFloat(market.Demand[input]-market.Supply[input], 0)
			target := need + 0.25*deficit
			missing := maxFloat(target-currentSurplus, 0)
			if missing <= 0.01 {
				continue
			}
			wealth := clamp01(market.Wealth)
			supportCap := supportScale * compactness * (0.25 + 0.75*access) * (0.25 + 0.75*wealth)
			support := minFloat(missing, supportCap)
			if support <= 0.01 {
				continue
			}
			market.Supply[input] += support
		}
	}
}

func tradeGoodsImportedInputSupportScale(
	production TradeGoodsProductionSettings,
	input string,
	category string,
) float64 {
	if value, ok := production.MarketImportedInputSupportScale[input]; ok {
		return value
	}
	if value, ok := production.MarketImportedInputSupportScale[category]; ok {
		return value
	}
	return production.MarketImportedInputSupportScale["default"]
}

func tradeGoodsExternalInputSupportScale(
	production TradeGoodsProductionSettings,
	input string,
	category string,
) float64 {
	if value, ok := production.MarketExternalInputSupportScale[input]; ok {
		return value
	}
	if value, ok := production.MarketExternalInputSupportScale[category]; ok {
		return value
	}
	return production.MarketExternalInputSupportScale["default"]
}

func marketEligibleManufacturingRawInputNeed(
	market *TradeNodeMarket,
	settings TradeGoodsSettings,
	node SettlementNode,
	trade *TradeNetworkResult,
	assignment PolityProfileAssignment,
) map[string]float64 {
	if market == nil {
		return nil
	}
	need := map[string]float64{}
	specByName := map[string]TradeGoodSpec{}
	for _, spec := range settings.Goods {
		specByName[spec.Name] = spec
	}
	for _, spec := range settings.Goods {
		if len(spec.Inputs) == 0 || !marketManufacturingEligibleForMarket(spec, node, market) {
			continue
		}
		demandGap := maxFloat(market.Demand[spec.Name]-market.Supply[spec.Name], 0)
		if demandGap <= 0.05 {
			continue
		}
		contextFit := marketProductionContextMultiplier(spec, node, market, trade, assignment)
		profileAffinity := profileGoodAffinity(spec.ProfileProductionAffinity, assignment)
		demandSignal := clamp01(demandGap / maxFloat(market.Demand[spec.Name], 0.12))
		outputNeed := clamp01(demandSignal * contextFit * profileAffinity)
		if outputNeed <= 0.01 {
			continue
		}
		for input, amount := range spec.Inputs {
			inputSpec, ok := specByName[input]
			if !ok || inputSpec.Category != "raw" || amount <= 0 {
				continue
			}
			coInputFit := marketImportedInputCoInputFit(spec, market, input)
			if coInputFit <= 0.05 {
				continue
			}
			reserveWeight := 0.25 + 1.25*clamp01(spec.MarketInputReservePriority)
			need[input] += outputNeed * amount * coInputFit * reserveWeight
		}
	}
	return need
}

func marketImportedInputCoInputFit(spec TradeGoodSpec, market *TradeNodeMarket, targetInput string) float64 {
	if market == nil || len(spec.Inputs) == 0 {
		return 0
	}
	total := 0.0
	count := 0.0
	for input, amount := range spec.Inputs {
		if input == targetInput || amount <= 0 {
			continue
		}
		available := maxFloat(market.Supply[input]-0.60*market.Demand[input], 0)
		fit := clamp01(available / maxFloat(amount, 0.05))
		total += fit
		count++
	}
	if count == 0 {
		return 1
	}
	return clamp01(total / count)
}
