package climgen

import "math"

type TradeNodeMarket struct {
	NodeID       int
	PolityID     int
	Wealth       float64
	FeederNodes  int
	FeederWealth float64
	Supply       map[string]float64
	Demand       map[string]float64
	Surplus      map[string]float64
	Exports      []PolityGoodValue
	Imports      []PolityGoodValue
	Manufactured map[string]float64
	Consumed     map[string]float64
	Diagnostics  TradeNodeMarketManufacturingDiagnostics
}

type TradeNodeMarketResult struct {
	Markets []TradeNodeMarket
}

func ComputeTradeNodeMarkets(
	cells []VoronoiCell,
	goods *TradeGoodResult,
	settings TradeGoodsSettings,
	polities *PolitySphereResult,
	profiles *PolityProfileResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	nodeGoods *NodeGoodsResult,
) *TradeNodeMarketResult {
	return computeTradeNodeMarkets(cells, goods, settings, polities, profiles, network, trade, nil, nil, nil, nil, nodeGoods)
}

func ComputeTradeNodeMarketsWithRouteSupport(
	cells []VoronoiCell,
	goods *TradeGoodResult,
	settings TradeGoodsSettings,
	polities *PolitySphereResult,
	profiles *PolityProfileResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	river *RiverTradeResult,
	coastal *CoastalTradeResult,
	ocean *OceanTradeResult,
	polityGoods *PolityGoodsResult,
	nodeGoods *NodeGoodsResult,
) *TradeNodeMarketResult {
	return computeTradeNodeMarkets(cells, goods, settings, polities, profiles, network, trade, river, coastal, ocean, polityGoods, nodeGoods)
}

func computeTradeNodeMarkets(
	cells []VoronoiCell,
	goods *TradeGoodResult,
	settings TradeGoodsSettings,
	polities *PolitySphereResult,
	profiles *PolityProfileResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	river *RiverTradeResult,
	coastal *CoastalTradeResult,
	ocean *OceanTradeResult,
	polityGoods *PolityGoodsResult,
	nodeGoods *NodeGoodsResult,
) *TradeNodeMarketResult {
	out := &TradeNodeMarketResult{}
	if nodeGoods == nil || network == nil || len(network.Nodes) == 0 {
		return out
	}
	out.Markets = make([]TradeNodeMarket, 0, len(nodeGoods.Balances))
	marketsByNode := make(map[int]*TradeNodeMarket, len(nodeGoods.Balances))
	for _, balance := range nodeGoods.Balances {
		market := TradeNodeMarket{
			NodeID:       balance.NodeID,
			PolityID:     balance.PolityID,
			Wealth:       balance.Wealth,
			Supply:       cloneFloatMap(balance.Supply),
			Demand:       cloneFloatMap(balance.Demand),
			Surplus:      cloneFloatMap(balance.Surplus),
			Exports:      append([]PolityGoodValue(nil), balance.Exports...),
			Imports:      append([]PolityGoodValue(nil), balance.Imports...),
			Manufactured: map[string]float64{},
			Consumed:     map[string]float64{},
		}
		out.Markets = append(out.Markets, market)
		marketsByNode[balance.NodeID] = &out.Markets[len(out.Markets)-1]
	}
	production := settings.EffectiveProductionSettings()
	demandSettings := settings.EffectiveDemandSettings()
	profileByPolity := map[int]PolityProfileAssignment{}
	if profiles != nil {
		for _, assignment := range profiles.Assignments {
			profileByPolity[assignment.PolityID] = assignment
		}
	}
	if goods != nil && trade != nil && len(trade.LocalNodes) > 0 {
		endowmentByGood := map[string]TradeGoodEndowment{}
		for _, good := range goods.Goods {
			endowmentByGood[good.Good] = good
		}
		for _, local := range trade.LocalNodes {
			if local.HandoffNode < 0 {
				continue
			}
			market := marketsByNode[local.HandoffNode]
			if market == nil {
				continue
			}
			assignment := profileByPolity[market.PolityID]
			wealth, supply := localTradeNodeContribution(local, cells, settings, endowmentByGood, assignment, production)
			market.FeederNodes++
			market.FeederWealth += wealth
			market.Wealth = clamp01(market.Wealth + 0.12*wealth)
			for good, value := range supply {
				market.Supply[good] += value
			}
		}
	}
	applyPolityMarketImportedInputSupport(out.Markets, settings, network, trade, profileByPolity, production)
	if polityGoods != nil && polities != nil {
		applyRouteAwareMarketExternalRawInputSupport(out.Markets, settings, polities, network, trade, river, coastal, ocean, profileByPolity, polityGoods, production)
	} else {
		applyMarketExternalRawInputSupport(out.Markets, settings, network, trade, profileByPolity, production)
	}
	for i := range out.Markets {
		market := &out.Markets[i]
		assignment := profileByPolity[market.PolityID]
		applyMarketDemandShaping(market, settings, network, trade, demandSettings)
		applyMarketManufacturing(market, settings, network, trade, assignment, production)
		market.Diagnostics = analyzeTradeNodeMarketManufacturing(market, settings, network, trade, assignment, production)
		recomputeMarketSurplus(market)
	}
	return out
}

func localTradeNodeContribution(
	node LocalTradeNode,
	cells []VoronoiCell,
	settings TradeGoodsSettings,
	endowmentByGood map[string]TradeGoodEndowment,
	assignment PolityProfileAssignment,
	production TradeGoodsProductionSettings,
) (float64, map[string]float64) {
	wealth := clamp01(0.34*node.Score + 0.32*node.Support + 0.20*node.Waystation + 0.14*localTradeNodeKindScale(node.Kind))
	supply := map[string]float64{}
	rawPotentials := localTradeNodeRawPotentials(settings, endowmentByGood, node, cells)
	for _, spec := range settings.Goods {
		if len(spec.Inputs) > 0 {
			continue
		}
		endowment, ok := endowmentByGood[spec.Name]
		if !ok {
			continue
		}
		localPotential := nodeCatchmentPotential(cells, node.CellIndex, localTradeNodeRawCatchmentRadius(spec), endowment.Potential)
		if localPotential <= 0 {
			continue
		}
		profileAffinity := profileGoodAffinity(spec.ProfileProductionAffinity, assignment)
		base := clamp01(0.56*localPotential + 0.24*node.Score + 0.12*node.Support + 0.08*localTradeNodeKindScale(node.Kind))
		specialization := tradeGoodsRawCatchmentSpecializationMultiplier(spec, rawPotentials, production)
		productionDriver := localTradeNodeProductionDrivers(spec.ProductionDrivers, node, assignment)
		productionContext := tradeGoodsProductionDriverMultiplier(spec, productionDriver)
		supply[spec.Name] = clamp01(base * specialization * productionContext * profileAffinity * tradeGoodsProductionMultiplier(spec.Category, localPotential, production.RawPotentialPivot, production))
	}
	return wealth, supply
}

func applyMarketManufacturing(
	market *TradeNodeMarket,
	settings TradeGoodsSettings,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	assignment PolityProfileAssignment,
	production TradeGoodsProductionSettings,
) {
	if market == nil || network == nil || market.NodeID < 0 || market.NodeID >= len(network.Nodes) {
		return
	}
	node := network.Nodes[market.NodeID]
	for pass := 0; pass < 3; pass++ {
		changed := false
		chainPressure := marketManufacturingChainPressure(market, settings, node, trade, assignment)
		plans := buildMarketManufacturingPlans(market, settings, node, trade, assignment, production, chainPressure)
		if len(plans) == 0 {
			return
		}
		for _, plan := range plans {
			spec := plan.Spec
			inputAccess, inputRichness, outputCapacity := marketEffectiveManufacturingInputs(spec, market, chainPressure, production)
			if inputAccess <= 0.01 {
				continue
			}
			if outputCapacity <= 0.01 {
				continue
			}
			profileAffinity := profileGoodAffinity(spec.ProfileProductionAffinity, assignment)
			workshop := marketManufacturingWorkshop(node, market, trade, spec.Category, inputRichness, production)
			manufacturingSignal := clamp01(0.44*inputAccess + 0.24*inputRichness + 0.32*workshop)
			efficiency := tradeGoodsManufacturingOutput(spec.Category, inputAccess, workshop, production)
			contextFit := marketProductionContextMultiplier(spec, node, market, trade, assignment)
			conversionScale := tradeGoodsCategorySetting(production.MarketConversionShare, spec.Category, 0.40)
			if spec.MarketConversionScale > 0 {
				conversionScale *= spec.MarketConversionScale
			}
			dominancePenalty, penaltyWeight := marketDominancePenaltyMultiplier(spec, market, settings, production)
			outputScale := Clamp(conversionScale*efficiency*profileAffinity*contextFit*dominancePenalty*tradeGoodsProductionMultiplier(spec.Category, manufacturingSignal, production.ManufacturingPivot, production), 0, 1)
			produced := outputCapacity * outputScale
			if produced <= 0.01 {
				continue
			}
			for input, need := range spec.Inputs {
				consume := produced * need
				market.Supply[input] = math.Max(0, market.Supply[input]-consume)
				market.Consumed[input] += consume
			}
			market.Supply[spec.Name] += produced
			market.Manufactured[spec.Name] += produced
			if penaltyWeight > 0 {
				if market.Diagnostics.Penalized == nil {
					market.Diagnostics.Penalized = map[string]float64{}
				}
				market.Diagnostics.Penalized[spec.Name] += produced * penaltyWeight
			}
			changed = true
		}
		if !changed {
			return
		}
	}
}

func applyMarketDemandShaping(
	market *TradeNodeMarket,
	settings TradeGoodsSettings,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	demandSettings TradeGoodsDemandSettings,
) {
	if market == nil || network == nil || market.NodeID < 0 || market.NodeID >= len(network.Nodes) {
		return
	}
	node := network.Nodes[market.NodeID]
	feederScale := clamp01(float64(market.FeederNodes) / 4.0)
	avgFeederWealth := clamp01(market.FeederWealth / math.Max(float64(marketMaxInt(market.FeederNodes, 1)), 1))
	wealthSignal := clamp01(0.58*clamp01(market.Wealth) + 0.24*nodeUrbanity(node) + 0.18*nodeTradeAccess(node, trade))
	feederSignal := clamp01(0.56*feederScale + 0.22*nodeKindScale(node) + 0.22*avgFeederWealth)
	for _, spec := range settings.Goods {
		base := market.Demand[spec.Name]
		if base <= 0 {
			continue
		}
		categoryScale := tradeGoodsCategorySetting(demandSettings.MarketCategoryDemandScale, spec.Category, 1.0)
		wealthPull := 1 + tradeGoodsCategorySetting(demandSettings.MarketWealthPullScale, spec.Category, 0.0)*wealthSignal
		feederPull := 1 + tradeGoodsCategorySetting(demandSettings.MarketFeederPullScale, spec.Category, 0.0)*feederSignal
		market.Demand[spec.Name] = clamp01(base * categoryScale * wealthPull * feederPull)
	}
}

func marketManufacturingWorkshop(
	node SettlementNode,
	market *TradeNodeMarket,
	trade *TradeNetworkResult,
	category string,
	inputRichness float64,
	production TradeGoodsProductionSettings,
) float64 {
	feederScale := clamp01(float64(market.FeederNodes) / 4.0)
	feederWealth := clamp01(market.FeederWealth / math.Max(float64(marketMaxInt(market.FeederNodes, 1)), 1))
	base := clamp01(
		0.28*nodeUrbanity(node) +
			0.20*nodeKindScale(node) +
			0.18*nodeTradeAccess(node, trade) +
			0.18*clamp01(market.Wealth) +
			0.10*feederScale +
			0.06*feederWealth,
	)
	return clamp01(
		base*tradeGoodsCategorySetting(production.MarketWorkshopBias, category, 1.0) +
			tradeGoodsCategorySetting(production.MarketInputRichnessScale, category, 0.0)*clamp01(inputRichness),
	)
}

func localTradeNodeRawPotentials(
	settings TradeGoodsSettings,
	endowmentByGood map[string]TradeGoodEndowment,
	node LocalTradeNode,
	cells []VoronoiCell,
) map[string]float64 {
	out := map[string]float64{}
	for _, spec := range settings.Goods {
		if spec.Category != "raw" {
			continue
		}
		endowment, ok := endowmentByGood[spec.Name]
		if !ok {
			continue
		}
		out[spec.Name] = nodeCatchmentPotential(cells, node.CellIndex, localTradeNodeRawCatchmentRadius(spec), endowment.Potential)
	}
	return out
}

func localTradeNodeRawCatchmentRadius(spec TradeGoodSpec) int {
	compactness := clamp01(0.65*(1-spec.Bulkiness) + 0.35*(1-spec.Perishability))
	switch {
	case compactness >= 0.82:
		return 3
	case compactness >= 0.62:
		return 2
	default:
		return 1
	}
}

func localTradeNodeProductionDrivers(drivers map[string]float64, node LocalTradeNode, assignment PolityProfileAssignment) float64 {
	if len(drivers) == 0 {
		return 0
	}
	total := 0.0
	weight := 0.0
	for driver, driverWeight := range drivers {
		if driverWeight <= 0 {
			continue
		}
		total += driverWeight * localTradeNodeProductionDriverValue(driver, node, assignment)
		weight += driverWeight
	}
	if weight == 0 {
		return 0
	}
	return clamp01(total / weight)
}

func localTradeNodeProductionDriverValue(driver string, node LocalTradeNode, assignment PolityProfileAssignment) float64 {
	switch driver {
	case "urban":
		return clamp01(0.15 + 0.25*localTradeNodeKindScale(node.Kind))
	case "mercantile":
		return clamp01(0.42*node.Score + 0.34*node.Waystation + 0.24*node.Support)
	case "coastal", "river", "highland", "arid", "cold", "wetland", "marsh", "delta", "wooded", "forest", "frontier":
		return bool01(hasProfileTag(assignment.EnvironmentTags, driver) || hasProfileTag(assignment.ContextTags, driver) || hasProfileTag(assignment.Profile.Tags, driver))
	case "warlike":
		if assignment.Profile.Attitudes != nil {
			return clamp01(assignment.Profile.Attitudes.Aggression)
		}
		return bool01(hasProfileTag(assignment.ContextTags, "fortress") || hasProfileTag(assignment.Profile.Tags, "martial"))
	case "luxury":
		return clamp01(0.18 + 0.22*node.Score + 0.18*node.Waystation)
	default:
		if hasProfileTag(assignment.ContextTags, driver) || hasProfileTag(assignment.EnvironmentTags, driver) || hasProfileTag(assignment.Profile.Tags, driver) {
			return 1
		}
	}
	return 0
}

func marketProductionContextMultiplier(
	spec TradeGoodSpec,
	node SettlementNode,
	market *TradeNodeMarket,
	trade *TradeNetworkResult,
	assignment PolityProfileAssignment,
) float64 {
	drivers := spec.DemandDrivers
	productionScoped := false
	if len(spec.ProductionDrivers) > 0 {
		drivers = spec.ProductionDrivers
		productionScoped = true
	}
	if len(drivers) == 0 {
		return 1.0
	}
	driverFit := nodeDemandDrivers(drivers, node, assignment, trade)
	wealthFit := clamp01(0.70*clamp01(market.Wealth) + 0.30*nodeUrbanity(node))
	kindFit := nodeKindScale(node)
	tradeFit := nodeTradeAccess(node, trade)
	if productionScoped {
		switch spec.Category {
		case "strategic":
			return applyGoodContextSensitivity(spec, Clamp(0.60+0.52*driverFit+0.12*kindFit+0.10*tradeFit+0.06*wealthFit, 0.52, 1.45))
		case "finished", "processed":
			return applyGoodContextSensitivity(spec, Clamp(0.62+0.48*driverFit+0.12*kindFit+0.10*tradeFit+0.06*wealthFit, 0.54, 1.40))
		case "luxury":
			return applyGoodContextSensitivity(spec, Clamp(0.64+0.44*driverFit+0.10*kindFit+0.08*tradeFit+0.10*wealthFit, 0.56, 1.40))
		default:
			return applyGoodContextSensitivity(spec, Clamp(0.72+0.34*driverFit+0.10*kindFit, 0.62, 1.24))
		}
	}
	switch spec.Category {
	case "luxury":
		return applyGoodContextSensitivity(spec, Clamp(0.75+0.40*driverFit+0.20*wealthFit, 0.70, 1.45))
	case "strategic":
		return applyGoodContextSensitivity(spec, Clamp(0.78+0.42*driverFit+0.10*tradeFit, 0.72, 1.45))
	case "finished", "processed":
		return applyGoodContextSensitivity(spec, Clamp(0.82+0.36*driverFit+0.12*wealthFit, 0.76, 1.40))
	default:
		return applyGoodContextSensitivity(spec, Clamp(0.88+0.24*driverFit, 0.84, 1.24))
	}
}

func applyGoodContextSensitivity(spec TradeGoodSpec, base float64) float64 {
	sensitivity := spec.MarketContextSensitivity
	if sensitivity <= 0 {
		sensitivity = 1.0
	}
	return Clamp(1+sensitivity*(base-1), 0.50, 1.75)
}

func recomputeMarketSurplus(market *TradeNodeMarket) {
	if market == nil {
		return
	}
	if market.Surplus == nil {
		market.Surplus = map[string]float64{}
	}
	for good, supply := range market.Supply {
		market.Surplus[good] = supply - market.Demand[good]
	}
	for good, demand := range market.Demand {
		if _, ok := market.Surplus[good]; ok {
			continue
		}
		market.Surplus[good] = -demand
	}
	market.Exports = topPolityGoods(market.Surplus, 3, true)
	market.Imports = topPolityGoods(market.Surplus, 3, false)
}

func tradeNodeMarketByNode(result *TradeNodeMarketResult) map[int]TradeNodeMarket {
	out := map[int]TradeNodeMarket{}
	if result == nil {
		return out
	}
	for _, market := range result.Markets {
		out[market.NodeID] = market
	}
	return out
}

func nodeMarketGoodFit(good string, fromMarket, toMarket *TradeNodeMarket) float64 {
	fit := 1.0
	if fromMarket != nil {
		source := clamp01(maxFloat(fromMarket.Surplus[good], 0) + 0.45*fromMarket.Supply[good])
		fit *= 0.78 + 0.34*source
	}
	if toMarket != nil {
		sink := clamp01(maxFloat(-toMarket.Surplus[good], 0) + 0.45*toMarket.Demand[good])
		fit *= 0.78 + 0.34*sink
	}
	if fromMarket != nil && toMarket != nil {
		wealthLink := math.Sqrt(clamp01(fromMarket.Wealth) * clamp01(toMarket.Wealth))
		fit *= 0.90 + 0.16*wealthLink
	}
	return Clamp(fit, 0.45, 1.35)
}

func localTradeNodeKindScale(kind LocalTradeNodeKind) float64 {
	switch kind {
	case LocalTradeNodeCrossingDepot:
		return 0.66
	case LocalTradeNodeWaystation:
		return 0.48
	default:
		return 0.30
	}
}

func cloneFloatMap(in map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func marketMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
