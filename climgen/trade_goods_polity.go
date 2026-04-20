package climgen

import "math"

func ComputePolityGoods(
	goods *TradeGoodResult,
	settings TradeGoodsSettings,
	polities *PolitySphereResult,
	profiles *PolityProfileResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
) *PolityGoodsResult {
	return computePolityGoods(goods, settings, polities, profiles, network, trade, nil)
}

func ComputePolityGoodsWithNodeMarkets(
	goods *TradeGoodResult,
	settings TradeGoodsSettings,
	polities *PolitySphereResult,
	profiles *PolityProfileResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	nodeGoods *NodeGoodsResult,
) *PolityGoodsResult {
	return computePolityGoods(goods, settings, polities, profiles, network, trade, nodeGoods)
}

func computePolityGoods(
	goods *TradeGoodResult,
	settings TradeGoodsSettings,
	polities *PolitySphereResult,
	profiles *PolityProfileResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	nodeGoods *NodeGoodsResult,
) *PolityGoodsResult {
	out := &PolityGoodsResult{}
	if goods == nil || polities == nil || polities.Diagnostics == nil || len(polities.Spheres) == 0 {
		return out
	}
	tuning := settings.EffectiveScarcitySettings()
	production := settings.EffectiveProductionSettings()
	demandSettings := settings.EffectiveDemandSettings()
	endowmentByGood := map[string]TradeGoodEndowment{}
	for _, good := range goods.Goods {
		endowmentByGood[good.Good] = good
	}
	profileByPolity := map[int]PolityProfileAssignment{}
	if profiles != nil {
		for _, assignment := range profiles.Assignments {
			profileByPolity[assignment.PolityID] = assignment
		}
	}
	marketWealth := map[int]float64{}
	if nodeGoods != nil && nodeGoods.PolityMarketWealth != nil {
		marketWealth = nodeGoods.PolityMarketWealth
	}
	nodeSupplyMeanByPolity, nodeDemandMeanByPolity := aggregateNodeGoodMeansByPolity(nodeGoods)
	out.Balances = make([]PolityGoodBalance, 0, len(polities.Spheres))
	if goods.Diagnostics != nil && len(goods.Diagnostics.ScarcityByGood) > 0 {
		out.GlobalScarcityByGood = cloneFloatMap(goods.Diagnostics.ScarcityByGood)
	}
	for _, sphere := range polities.Spheres {
		assignment := profileByPolity[sphere.ID]
		balance := PolityGoodBalance{
			PolityID:     sphere.ID,
			MarketWealth: marketWealth[sphere.ID],
			Supply:       map[string]float64{},
			Demand:       map[string]float64{},
			Surplus:      map[string]float64{},
		}
		for _, spec := range settings.Goods {
			scarcity := tradeGoodScarcity(goods, spec.Name)
			supply := polityGoodSupply(spec, endowmentByGood, sphere, assignment, polities, network, trade, balance.Supply, balance.MarketWealth, scarcity, tuning, production)
			if len(spec.Inputs) > 0 && nodeSupplyMeanByPolity != nil {
				supply = clamp01(0.50*supply + 0.50*nodeSupplyMeanByPolity[sphere.ID][spec.Name])
			}
			balance.Supply[spec.Name] = supply
			demand := polityGoodDemand(spec, sphere, assignment, network, trade, balance.Supply, balance.MarketWealth, scarcity, tuning, demandSettings)
			if len(spec.Inputs) > 0 && nodeDemandMeanByPolity != nil {
				demand = clamp01(0.50*demand + 0.50*nodeDemandMeanByPolity[sphere.ID][spec.Name])
			}
			balance.Demand[spec.Name] = demand
			balance.Surplus[spec.Name] = supply - demand
		}
		balance.Exports = topPolityGoods(balance.Surplus, 4, true)
		balance.Imports = topPolityGoods(balance.Surplus, 4, false)
		out.Balances = append(out.Balances, balance)
	}
	return out
}

func aggregateNodeGoodMeansByPolity(nodeGoods *NodeGoodsResult) (map[int]map[string]float64, map[int]map[string]float64) {
	if nodeGoods == nil || len(nodeGoods.Balances) == 0 {
		return nil, nil
	}
	supplyByPolity := map[int]map[string]float64{}
	demandByPolity := map[int]map[string]float64{}
	weightByPolity := map[int]float64{}
	for _, balance := range nodeGoods.Balances {
		weight := 1.0 + balance.Wealth
		supplyMap := supplyByPolity[balance.PolityID]
		if supplyMap == nil {
			supplyMap = map[string]float64{}
			supplyByPolity[balance.PolityID] = supplyMap
		}
		demandMap := demandByPolity[balance.PolityID]
		if demandMap == nil {
			demandMap = map[string]float64{}
			demandByPolity[balance.PolityID] = demandMap
		}
		weightByPolity[balance.PolityID] += weight
		for good, value := range balance.Supply {
			supplyMap[good] += value * weight
		}
		for good, value := range balance.Demand {
			demandMap[good] += value * weight
		}
	}
	for polityID, weight := range weightByPolity {
		if weight <= 0 {
			continue
		}
		for good, total := range supplyByPolity[polityID] {
			supplyByPolity[polityID][good] = total / weight
		}
		for good, total := range demandByPolity[polityID] {
			demandByPolity[polityID][good] = total / weight
		}
	}
	return supplyByPolity, demandByPolity
}

func polityGoodSupply(
	spec TradeGoodSpec,
	endowmentByGood map[string]TradeGoodEndowment,
	sphere PolitySphere,
	assignment PolityProfileAssignment,
	polities *PolitySphereResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	currentSupply map[string]float64,
	marketWealth float64,
	scarcity float64,
	tuning TradeGoodsScarcitySettings,
	production TradeGoodsProductionSettings,
) float64 {
	endowment := endowmentByGood[spec.Name]
	localPotential := polityMeanPotential(sphere.ID, endowment.Potential, polities)
	profileAffinity := profileGoodAffinity(spec.ProfileProductionAffinity, assignment)
	support := clamp01(sphere.MeanSupport / 0.45)
	capitalTier := polityCapitalTier(sphere, network)
	tradeAccess := polityTradeAccess(sphere, trade)
	productionDriver := polityProductionDrivers(spec.ProductionDrivers, sphere, assignment, network, trade)
	if len(spec.Inputs) > 0 {
		if !tradeGoodsHasLocalInputCapability(spec, currentSupply) {
			return 0
		}
		inputAccess := 1.0
		for input, need := range spec.Inputs {
			if need <= 0 {
				continue
			}
			localInput := clamp01(currentSupply[input] / need)
			blendedInput := clamp01(0.72*localInput + 0.28*tradeAccess)
			inputAccess = math.Min(inputAccess, blendedInput)
		}
		inputAccess = clamp01(inputAccess)
		workshop := clamp01(0.32*capitalTier + 0.26*tradeAccess + 0.22*support + 0.20*marketWealth)
		manufacturingSignal := clamp01(0.56*inputAccess + 0.44*workshop)
		supply := tradeGoodsManufacturingOutput(spec.Category, inputAccess, workshop, production) * profileAffinity * scarcitySupplyIncentiveWithSettings(spec, scarcity, tuning)
		return clamp01(supply * tradeGoodsProductionMultiplier(spec.Category, manufacturingSignal, production.ManufacturingPivot, production))
	}
	rawBase := clamp01(0.72*localPotential + 0.18*support + 0.10*tradeAccess)
	productionContext := tradeGoodsProductionDriverMultiplier(spec, productionDriver)
	supply := rawBase * productionContext * profileAffinity * scarcitySupplyIncentiveWithSettings(spec, scarcity, tuning)
	return clamp01(supply * tradeGoodsProductionMultiplier(spec.Category, localPotential, production.RawPotentialPivot, production))
}

func polityGoodDemand(
	spec TradeGoodSpec,
	sphere PolitySphere,
	assignment PolityProfileAssignment,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	currentSupply map[string]float64,
	marketWealth float64,
	scarcity float64,
	tuning TradeGoodsScarcitySettings,
	demandSettings TradeGoodsDemandSettings,
) float64 {
	popScale := clamp01(0.42*sphere.MeanSupport/0.45 + 0.28*math.Sqrt(math.Max(float64(sphere.TerritoryCells), 0))/18.0 + 0.30*polityCapitalTier(sphere, network))
	valueDemand := clamp01(0.25 + 0.55*spec.BaseValue + 0.20*(1-spec.Bulkiness))
	categoryDemand := tradeGoodCategoryDemand(spec.Category)
	driverDemand := profileDemandDrivers(spec.DemandDrivers, sphere, assignment, network, trade)
	wealthDemand := polityWealthDemand(spec, marketWealth)
	scarcityDemand := scarcityDemandPressureWithSettings(spec, scarcity, tuning)
	profileAffinity := profileGoodAffinity(spec.ProfileDemandAffinity, assignment)
	base := 0.24*popScale + 0.16*valueDemand + 0.14*categoryDemand + 0.12*driverDemand + 0.18*wealthDemand + 0.16*scarcityDemand
	localRelief := tradeGoodsDemandRelief(spec.Category, currentSupply[spec.Name], demandSettings)
	driverSpecialization := tradeGoodsDriverDemandMultiplier(spec.Category, driverDemand, demandSettings)
	categoryScale := tradeGoodsCategorySetting(demandSettings.CategoryDemandScale, spec.Category, 1.0)
	return clamp01(base * categoryScale * driverSpecialization * localRelief * profileAffinity)
}

func polityMeanPotential(polityID int, potential []float64, polities *PolitySphereResult) float64 {
	if len(potential) == 0 || polities == nil || polities.Diagnostics == nil {
		return 0
	}
	total := 0.0
	count := 0.0
	for cellIdx, owner := range polities.Diagnostics.PolityByCell {
		if owner != polityID || cellIdx < 0 || cellIdx >= len(potential) {
			continue
		}
		total += potential[cellIdx]
		count++
	}
	if count == 0 {
		return 0
	}
	return total / count
}

func polityCapitalTier(sphere PolitySphere, network *SettlementNetworkResult) float64 {
	if network == nil || sphere.CapitalNode < 0 || sphere.CapitalNode >= len(network.Nodes) {
		return 0.30
	}
	return clamp01(float64(network.Nodes[sphere.CapitalNode].Kind) / 3.0)
}

func polityTradeAccess(sphere PolitySphere, trade *TradeNetworkResult) float64 {
	if trade == nil || trade.Diagnostics == nil || sphere.CapitalNode < 0 || sphere.CapitalNode >= len(trade.Diagnostics.NodeCentrality) {
		return 0
	}
	return clamp01(trade.Diagnostics.NodeCentrality[sphere.CapitalNode] / 0.55)
}

func profileGoodAffinity(affinities map[string]float64, assignment PolityProfileAssignment) float64 {
	multiplier := 1.0
	for key, value := range affinities {
		if profilePreferenceMatches(key, assignment) {
			multiplier *= value
		}
	}
	return Clamp(multiplier, 0.20, 2.20)
}

func profileDemandDrivers(drivers map[string]float64, sphere PolitySphere, assignment PolityProfileAssignment, network *SettlementNetworkResult, trade *TradeNetworkResult) float64 {
	if len(drivers) == 0 {
		return 0.35
	}
	total := 0.0
	weight := 0.0
	for driver, driverWeight := range drivers {
		if driverWeight <= 0 {
			continue
		}
		total += driverWeight * polityDemandDriverValue(driver, sphere, assignment, network, trade)
		weight += driverWeight
	}
	if weight == 0 {
		return 0.35
	}
	return clamp01(total / weight)
}

func polityProductionDrivers(drivers map[string]float64, sphere PolitySphere, assignment PolityProfileAssignment, network *SettlementNetworkResult, trade *TradeNetworkResult) float64 {
	if len(drivers) == 0 {
		return 0
	}
	return profileDemandDrivers(drivers, sphere, assignment, network, trade)
}

func polityDemandDriverValue(driver string, sphere PolitySphere, assignment PolityProfileAssignment, network *SettlementNetworkResult, trade *TradeNetworkResult) float64 {
	switch driver {
	case "urban":
		if hasProfileTag(assignment.ContextTags, "urban") || hasProfileTag(assignment.EnvironmentTags, "settled-core") {
			return 1
		}
		return polityCapitalTier(sphere, network)
	case "mercantile":
		if hasProfileTag(assignment.ContextTags, "mercantile") || hasProfileTag(assignment.EnvironmentTags, "seaborne") {
			return 1
		}
		return polityTradeAccess(sphere, trade)
	case "coastal":
		return bool01(sphere.Coastal || hasProfileTag(assignment.EnvironmentTags, "coastal"))
	case "river":
		return bool01(sphere.River || hasProfileTag(assignment.EnvironmentTags, "river"))
	case "highland":
		return bool01(sphere.Style == ProtoCivilizationHighland || hasProfileTag(assignment.EnvironmentTags, "mountain"))
	case "arid":
		return bool01(sphere.Style == ProtoCivilizationArid || hasProfileTag(assignment.EnvironmentTags, "arid"))
	case "cold":
		return bool01(hasProfileTag(assignment.EnvironmentTags, "cold"))
	case "hot":
		return bool01(hasProfileTag(assignment.EnvironmentTags, "hot"))
	case "warlike":
		if assignment.Profile.Attitudes != nil {
			return clamp01(assignment.Profile.Attitudes.Aggression)
		}
		return bool01(hasProfileTag(assignment.ContextTags, "fortress") || hasProfileTag(assignment.Profile.Tags, "martial"))
	case "luxury":
		return clamp01(0.45*polityCapitalTier(sphere, network) + 0.55*polityTradeAccess(sphere, trade))
	default:
		if hasProfileTag(assignment.ContextTags, driver) || hasProfileTag(assignment.EnvironmentTags, driver) || hasProfileTag(assignment.Profile.Tags, driver) {
			return 1
		}
	}
	return 0
}
