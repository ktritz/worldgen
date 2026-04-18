package climgen

import "math"

type NodeGoodBalance struct {
	NodeID   int
	PolityID int
	Wealth   float64
	Supply   map[string]float64
	Demand   map[string]float64
	Surplus  map[string]float64
	Exports  []PolityGoodValue
	Imports  []PolityGoodValue
}

type NodeGoodsResult struct {
	Balances           []NodeGoodBalance
	PolityMarketWealth map[int]float64
}

func ComputeNodeGoods(
	cells []VoronoiCell,
	goods *TradeGoodResult,
	settings TradeGoodsSettings,
	polities *PolitySphereResult,
	profiles *PolityProfileResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
) *NodeGoodsResult {
	out := &NodeGoodsResult{}
	if goods == nil || network == nil || len(network.Nodes) == 0 {
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
	out.Balances = make([]NodeGoodBalance, 0, len(network.Nodes))
	for _, node := range network.Nodes {
		polityID := tradeNodePolity(node.ID, polities, network)
		assignment := profileByPolity[polityID]
		wealth := estimateNodeWealth(node, cells, settings, endowmentByGood, trade)
		rawPotentials := nodeRawCatchmentPotentials(settings, endowmentByGood, node, cells)
		balance := NodeGoodBalance{
			NodeID:   node.ID,
			PolityID: polityID,
			Wealth:   wealth,
			Supply:   map[string]float64{},
			Demand:   map[string]float64{},
			Surplus:  map[string]float64{},
		}
		for _, spec := range settings.Goods {
			scarcity := tradeGoodScarcity(goods, spec.Name)
			supply := nodeGoodSupply(spec, endowmentByGood, rawPotentials, node, cells, assignment, trade, balance.Supply, scarcity, tuning, production)
			balance.Supply[spec.Name] = supply
			demand := nodeGoodDemand(spec, node, assignment, trade, balance.Supply, wealth, scarcity, tuning, demandSettings)
			balance.Demand[spec.Name] = demand
			balance.Surplus[spec.Name] = supply - demand
		}
		balance.Exports = topPolityGoods(balance.Surplus, 3, true)
		balance.Imports = topPolityGoods(balance.Surplus, 3, false)
		out.Balances = append(out.Balances, balance)
	}
	out.PolityMarketWealth = aggregatePolityMarketWealth(out, network, trade)
	return out
}

func nodeGoodSupply(
	spec TradeGoodSpec,
	endowmentByGood map[string]TradeGoodEndowment,
	rawPotentials map[string]float64,
	node SettlementNode,
	cells []VoronoiCell,
	assignment PolityProfileAssignment,
	trade *TradeNetworkResult,
	currentSupply map[string]float64,
	scarcity float64,
	tuning TradeGoodsScarcitySettings,
	production TradeGoodsProductionSettings,
) float64 {
	endowment := endowmentByGood[spec.Name]
	localPotential := nodeCatchmentPotential(cells, node.CellIndex, nodeCatchmentRadius(node), endowment.Potential)
	if spec.Category == "raw" {
		if value, ok := rawPotentials[spec.Name]; ok {
			localPotential = value
		}
	}
	profileAffinity := profileGoodAffinity(spec.ProfileProductionAffinity, assignment)
	tradeAccess := nodeTradeAccess(node, trade)
	urbanity := nodeUrbanity(node)
	kindScale := nodeKindScale(node)
	productionDriver := nodeProductionDrivers(spec.ProductionDrivers, node, assignment, trade)
	if len(spec.Inputs) > 0 {
		inputAccess := 1.0
		for input, need := range spec.Inputs {
			if need <= 0 {
				continue
			}
			localInput := clamp01(currentSupply[input] / need)
			blendedInput := clamp01(0.66*localInput + 0.34*tradeAccess)
			inputAccess = math.Min(inputAccess, blendedInput)
		}
		workshop := clamp01(0.42*urbanity + 0.24*kindScale + 0.18*tradeAccess + 0.16*clamp01(node.CarryingCapacity))
		manufacturingSignal := clamp01(0.58*inputAccess + 0.42*workshop)
		supply := tradeGoodsManufacturingOutput(spec.Category, inputAccess, workshop, production) * profileAffinity * scarcitySupplyIncentiveWithSettings(spec, scarcity, tuning)
		return clamp01(supply * tradeGoodsProductionMultiplier(spec.Category, manufacturingSignal, production.ManufacturingPivot, production))
	}
	rawBase := clamp01(0.62*localPotential + 0.18*clamp01(node.CarryingCapacity) + 0.12*tradeAccess + 0.08*kindScale)
	specialization := tradeGoodsRawCatchmentSpecializationMultiplier(spec, rawPotentials, production)
	productionContext := tradeGoodsProductionDriverMultiplier(spec, productionDriver)
	supply := rawBase * specialization * productionContext * profileAffinity * scarcitySupplyIncentiveWithSettings(spec, scarcity, tuning)
	return clamp01(supply * tradeGoodsProductionMultiplier(spec.Category, localPotential, production.RawPotentialPivot, production))
}

func nodeGoodDemand(
	spec TradeGoodSpec,
	node SettlementNode,
	assignment PolityProfileAssignment,
	trade *TradeNetworkResult,
	currentSupply map[string]float64,
	wealth float64,
	scarcity float64,
	tuning TradeGoodsScarcitySettings,
	demandSettings TradeGoodsDemandSettings,
) float64 {
	popScale := clamp01(0.38*clamp01(node.CarryingCapacity) + 0.34*nodeUrbanity(node) + 0.18*nodeKindScale(node) + 0.10*nodeTradeAccess(node, trade))
	valueDemand := clamp01(0.25 + 0.55*spec.BaseValue + 0.20*(1-spec.Bulkiness))
	categoryDemand := tradeGoodCategoryDemand(spec.Category)
	driverDemand := nodeDemandDrivers(spec.DemandDrivers, node, assignment, trade)
	wealthDemand := nodeWealthDemand(spec, wealth)
	scarcityDemand := scarcityDemandPressureWithSettings(spec, scarcity, tuning)
	demandScale := nodeDemandScale(spec, node, wealth, trade)
	profileAffinity := profileGoodAffinity(spec.ProfileDemandAffinity, assignment)
	base := (0.26*popScale + 0.16*valueDemand + 0.12*categoryDemand + 0.14*driverDemand + 0.16*wealthDemand + 0.16*scarcityDemand)
	localRelief := tradeGoodsDemandRelief(spec.Category, currentSupply[spec.Name], demandSettings)
	driverSpecialization := tradeGoodsDriverDemandMultiplier(spec.Category, driverDemand, demandSettings)
	categoryScale := tradeGoodsCategorySetting(demandSettings.CategoryDemandScale, spec.Category, 1.0)
	return clamp01(base * demandScale * categoryScale * driverSpecialization * localRelief * profileAffinity)
}

func nodeDemandDrivers(drivers map[string]float64, node SettlementNode, assignment PolityProfileAssignment, trade *TradeNetworkResult) float64 {
	if len(drivers) == 0 {
		return 0.35
	}
	total := 0.0
	weight := 0.0
	for driver, driverWeight := range drivers {
		if driverWeight <= 0 {
			continue
		}
		total += driverWeight * nodeDemandDriverValue(driver, node, assignment, trade)
		weight += driverWeight
	}
	if weight == 0 {
		return 0.35
	}
	return clamp01(total / weight)
}

func nodeProductionDrivers(drivers map[string]float64, node SettlementNode, assignment PolityProfileAssignment, trade *TradeNetworkResult) float64 {
	if len(drivers) == 0 {
		return 0
	}
	return nodeDemandDrivers(drivers, node, assignment, trade)
}

func nodeDemandDriverValue(driver string, node SettlementNode, assignment PolityProfileAssignment, trade *TradeNetworkResult) float64 {
	switch driver {
	case "urban":
		return nodeUrbanity(node)
	case "mercantile":
		if hasProfileTag(assignment.ContextTags, "mercantile") || hasProfileTag(assignment.EnvironmentTags, "seaborne") {
			return clamp01(0.45 + 0.55*nodeTradeAccess(node, trade))
		}
		return nodeTradeAccess(node, trade)
	case "coastal":
		return bool01(node.Coastal || hasProfileTag(assignment.EnvironmentTags, "coastal"))
	case "river":
		return bool01(node.River || hasProfileTag(assignment.EnvironmentTags, "river"))
	case "highland":
		return bool01(hasProfileTag(assignment.EnvironmentTags, "mountain"))
	case "arid":
		return bool01(hasProfileTag(assignment.EnvironmentTags, "arid"))
	case "cold":
		return bool01(hasProfileTag(assignment.EnvironmentTags, "cold"))
	case "warlike":
		if assignment.Profile.Attitudes != nil {
			return clamp01(assignment.Profile.Attitudes.Aggression)
		}
		return bool01(hasProfileTag(assignment.ContextTags, "fortress") || hasProfileTag(assignment.Profile.Tags, "martial"))
	case "luxury":
		return clamp01(0.42*nodeUrbanity(node) + 0.32*nodeTradeAccess(node, trade) + 0.26*nodeKindScale(node))
	default:
		if hasProfileTag(assignment.ContextTags, driver) || hasProfileTag(assignment.EnvironmentTags, driver) || hasProfileTag(assignment.Profile.Tags, driver) {
			return 1
		}
	}
	return 0
}

func nodeCatchmentPotential(cells []VoronoiCell, startCell, radius int, potential []float64) float64 {
	if startCell < 0 || startCell >= len(potential) {
		return 0
	}
	if radius <= 0 || startCell >= len(cells) {
		return safeSliceValue(potential, startCell)
	}
	type frontier struct {
		cell int
		dist int
	}
	queue := []frontier{{cell: startCell, dist: 0}}
	seen := map[int]struct{}{startCell: {}}
	total := 0.0
	weight := 0.0
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		w := 1.0 / float64(1+cur.dist)
		total += w * safeSliceValue(potential, cur.cell)
		weight += w
		if cur.dist >= radius || cur.cell < 0 || cur.cell >= len(cells) {
			continue
		}
		for _, neighbor := range cells[cur.cell].NeighborSiteIndices {
			next := int(neighbor)
			if next < 0 || next >= len(potential) {
				continue
			}
			if _, ok := seen[next]; ok {
				continue
			}
			seen[next] = struct{}{}
			queue = append(queue, frontier{cell: next, dist: cur.dist + 1})
		}
	}
	if weight == 0 {
		return 0
	}
	return clamp01(total / weight)
}

func nodeCatchmentRadius(node SettlementNode) int {
	switch node.Kind {
	case SettlementNodeCity:
		return 2
	case SettlementNodeTown:
		return 2
	default:
		return 1
	}
}

func nodeRawCatchmentPotentials(
	settings TradeGoodsSettings,
	endowmentByGood map[string]TradeGoodEndowment,
	node SettlementNode,
	cells []VoronoiCell,
) map[string]float64 {
	out := map[string]float64{}
	radius := nodeCatchmentRadius(node)
	for _, spec := range settings.Goods {
		if spec.Category != "raw" {
			continue
		}
		endowment, ok := endowmentByGood[spec.Name]
		if !ok {
			continue
		}
		out[spec.Name] = nodeCatchmentPotential(cells, node.CellIndex, radius, endowment.Potential)
	}
	return out
}

func nodeUrbanity(node SettlementNode) float64 {
	return clamp01(0.55*clamp01(node.UrbanPotential) + 0.45*nodeKindScale(node))
}

func nodeKindScale(node SettlementNode) float64 {
	return clamp01(float64(node.Kind+1) / 4.0)
}

func nodeTradeAccess(node SettlementNode, trade *TradeNetworkResult) float64 {
	if trade == nil || trade.Diagnostics == nil || node.ID < 0 || node.ID >= len(trade.Diagnostics.NodeCentrality) {
		return 0
	}
	return clamp01(trade.Diagnostics.NodeCentrality[node.ID] / 0.55)
}
