package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

type chainValue struct {
	label string
	value float64
}

func printTradeChainSummary(nodeGoods *climgen.NodeGoodsResult, markets *climgen.TradeNodeMarketResult, network *climgen.SettlementNetworkResult) {
	printSpecificTradeChainSummary("fish", "preserved_food", nodeGoods, markets, network)
	printSpecificTradeChainSummary("wool", "woolens", nodeGoods, markets, network)
	printSpecificTradeChainSummary("cloth", "fine_clothing", nodeGoods, markets, network)
}

func printTradeGoodPathSummary(good string, nodeGoods *climgen.NodeGoodsResult, markets *climgen.TradeNodeMarketResult, polityGoods *climgen.PolityGoodsResult, multimodal *climgen.MultimodalTradeResult, network *climgen.SettlementNetworkResult) {
	if nodeGoods == nil || markets == nil || polityGoods == nil || multimodal == nil {
		return
	}

	nodeSupply := 0.0
	nodeSurplus := 0.0
	marketSupply := 0.0
	marketDemand := 0.0
	marketSurplus := 0.0
	marketDeficit := 0.0
	marketDeficitCount := 0
	marketMade := 0.0
	politySupply := 0.0
	polityDemand := 0.0
	politySurplus := 0.0
	polityExporters := 0
	polityImporters := 0
	tradeScore := 0.0
	tradeVolume := 0.0
	tradePairs := 0

	for _, balance := range nodeGoods.Balances {
		nodeSupply += balance.Supply[good]
		nodeSurplus += reviewMaxFloat(balance.Surplus[good], 0)
	}
	for _, market := range markets.Markets {
		marketSupply += market.Supply[good]
		marketDemand += market.Demand[good]
		marketSurplus += reviewMaxFloat(market.Surplus[good], 0)
		deficit := reviewMaxFloat(-market.Surplus[good], 0)
		marketDeficit += deficit
		if deficit > 0 {
			marketDeficitCount++
		}
		marketMade += market.Manufactured[good]
	}
	for _, balance := range polityGoods.Balances {
		supply := balance.Supply[good]
		demand := balance.Demand[good]
		surplus := balance.Surplus[good]
		politySupply += supply
		polityDemand += demand
		politySurplus += reviewMaxFloat(surplus, 0)
		if surplus > 0 {
			polityExporters++
		}
		if surplus < 0 {
			polityImporters++
		}
	}
	for _, pair := range multimodal.Pairs {
		for _, flow := range pair.Goods {
			if flow.Good != good {
				continue
			}
			tradeScore += flow.Score
			tradeVolume += flow.Volume
			tradePairs++
		}
	}

	fmt.Printf(
		"      tradeGoodPath[%s]: nodeSupply=%.2f nodeSurplus=%.2f marketSupply=%.2f marketDemand=%.2f marketSurplus=%.2f marketDeficit=%.2f marketDeficitNodes=%d made=%.2f politySupply=%.2f polityDemand=%.2f politySurplus=%.2f exporters=%d importers=%d tradeScore=%.2f tradeVolume=%.2f tradePairs=%d\n",
		good,
		nodeSupply,
		nodeSurplus,
		marketSupply,
		marketDemand,
		marketSurplus,
		marketDeficit,
		marketDeficitCount,
		marketMade,
		politySupply,
		polityDemand,
		politySurplus,
		polityExporters,
		polityImporters,
		tradeScore,
		tradeVolume,
		tradePairs,
	)
	fmt.Printf("      %sPolities=%s\n", good, formatChainValues(topPolityChainValues(good, polityGoods, 4), 4))
	fmt.Printf("      %sPolityImports=%s\n", good, formatChainValues(topPolityImportChainValues(good, polityGoods, 4), 4))
	fmt.Printf("      %sTradePairs=%s\n", good, formatTradePairChainValues(good, multimodal, 4))
}

func printSpecificTradeChainSummary(rawGood, processedGood string, nodeGoods *climgen.NodeGoodsResult, markets *climgen.TradeNodeMarketResult, network *climgen.SettlementNetworkResult) {
	if nodeGoods == nil || markets == nil {
		return
	}
	nodeRawSurplus := 0.0
	nodeRawSupply := 0.0
	marketRawSurplus := 0.0
	marketRawSupply := 0.0
	marketRawDemand := 0.0
	processedMade := 0.0
	processedDemandGap := 0.0
	blockedCount := 0

	for _, balance := range nodeGoods.Balances {
		supply := balance.Supply[rawGood]
		surplus := reviewMaxFloat(balance.Surplus[rawGood], 0)
		nodeRawSupply += supply
		nodeRawSurplus += surplus
	}
	for _, market := range markets.Markets {
		supply := market.Supply[rawGood]
		demand := market.Demand[rawGood]
		surplus := reviewMaxFloat(market.Surplus[rawGood], 0)
		marketRawSupply += supply
		marketRawDemand += demand
		marketRawSurplus += surplus
		processedMade += market.Manufactured[processedGood]
		processedDemandGap += reviewMaxFloat(market.Demand[processedGood]-market.Supply[processedGood], 0)
		for _, blocked := range market.Diagnostics.Blocked {
			if blocked.Good == processedGood {
				blockedCount++
				break
			}
		}
	}

	fmt.Printf(
		"      tradeChain[%s->%s]: nodeSupply=%.2f nodeSurplus=%.2f marketSupply=%.2f marketSurplus=%.2f marketDemand=%.2f made=%.2f demandGap=%.2f blockedMarkets=%d\n",
		rawGood,
		processedGood,
		nodeRawSupply,
		nodeRawSurplus,
		marketRawSupply,
		marketRawSurplus,
		marketRawDemand,
		processedMade,
		processedDemandGap,
		blockedCount,
	)

	fmt.Printf(
		"      %sNodes=%s\n",
		rawGood,
		formatChainValues(topNodeChainValues(rawGood, nodeGoods, network, 3), 3),
	)
	fmt.Printf(
		"      %sMarkets=%s\n",
		rawGood,
		formatChainValues(topMarketChainValues(rawGood, markets, network, 3), 3),
	)
	fmt.Printf(
		"      %sImportMarkets=%s\n",
		rawGood,
		formatChainValues(topMarketImportChainValues(rawGood, markets, network, 3), 3),
	)
	fmt.Printf(
		"      %sMakers=%s\n",
		processedGood,
		formatChainValues(topMarketManufacturedChainValues(processedGood, markets, network, 3), 3),
	)
	fmt.Printf(
		"      blocked%s=%s\n",
		processedGood,
		formatBlockedChainMarkets(processedGood, markets, network, 3),
	)
}

func topNodeChainValues(good string, nodeGoods *climgen.NodeGoodsResult, network *climgen.SettlementNetworkResult, limit int) []chainValue {
	values := make([]chainValue, 0, len(nodeGoods.Balances))
	for _, balance := range nodeGoods.Balances {
		surplus := reviewMaxFloat(balance.Surplus[good], 0)
		if surplus <= 0 {
			continue
		}
		values = append(values, chainValue{
			label: formatChainNodeLabel(balance.NodeID, network),
			value: surplus,
		})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].value != values[j].value {
			return values[i].value > values[j].value
		}
		return values[i].label < values[j].label
	})
	if len(values) < limit {
		limit = len(values)
	}
	return values[:limit]
}

func topMarketChainValues(good string, markets *climgen.TradeNodeMarketResult, network *climgen.SettlementNetworkResult, limit int) []chainValue {
	values := make([]chainValue, 0, len(markets.Markets))
	for _, market := range markets.Markets {
		surplus := reviewMaxFloat(market.Surplus[good], 0)
		if surplus <= 0 {
			continue
		}
		values = append(values, chainValue{
			label: formatChainNodeLabel(market.NodeID, network),
			value: surplus,
		})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].value != values[j].value {
			return values[i].value > values[j].value
		}
		return values[i].label < values[j].label
	})
	if len(values) < limit {
		limit = len(values)
	}
	return values[:limit]
}

func topMarketImportChainValues(good string, markets *climgen.TradeNodeMarketResult, network *climgen.SettlementNetworkResult, limit int) []chainValue {
	values := make([]chainValue, 0, len(markets.Markets))
	for _, market := range markets.Markets {
		deficit := reviewMaxFloat(-market.Surplus[good], 0)
		if deficit <= 0 {
			continue
		}
		values = append(values, chainValue{
			label: formatChainNodeLabel(market.NodeID, network),
			value: deficit,
		})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].value != values[j].value {
			return values[i].value > values[j].value
		}
		return values[i].label < values[j].label
	})
	if len(values) < limit {
		limit = len(values)
	}
	return values[:limit]
}

func topMarketManufacturedChainValues(good string, markets *climgen.TradeNodeMarketResult, network *climgen.SettlementNetworkResult, limit int) []chainValue {
	values := make([]chainValue, 0, len(markets.Markets))
	for _, market := range markets.Markets {
		made := market.Manufactured[good]
		if made <= 0 {
			continue
		}
		values = append(values, chainValue{
			label: formatChainNodeLabel(market.NodeID, network),
			value: made,
		})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].value != values[j].value {
			return values[i].value > values[j].value
		}
		return values[i].label < values[j].label
	})
	if len(values) < limit {
		limit = len(values)
	}
	return values[:limit]
}

func topPolityChainValues(good string, polityGoods *climgen.PolityGoodsResult, limit int) []chainValue {
	values := make([]chainValue, 0, len(polityGoods.Balances))
	for _, balance := range polityGoods.Balances {
		surplus := reviewMaxFloat(balance.Surplus[good], 0)
		if surplus <= 0 {
			continue
		}
		values = append(values, chainValue{
			label: fmt.Sprintf("%d", balance.PolityID),
			value: surplus,
		})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].value != values[j].value {
			return values[i].value > values[j].value
		}
		return values[i].label < values[j].label
	})
	if len(values) < limit {
		limit = len(values)
	}
	return values[:limit]
}

func topPolityImportChainValues(good string, polityGoods *climgen.PolityGoodsResult, limit int) []chainValue {
	values := make([]chainValue, 0, len(polityGoods.Balances))
	for _, balance := range polityGoods.Balances {
		deficit := reviewMaxFloat(-balance.Surplus[good], 0)
		if deficit <= 0 {
			continue
		}
		values = append(values, chainValue{
			label: fmt.Sprintf("%d", balance.PolityID),
			value: deficit,
		})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].value != values[j].value {
			return values[i].value > values[j].value
		}
		return values[i].label < values[j].label
	})
	if len(values) < limit {
		limit = len(values)
	}
	return values[:limit]
}

func formatTradePairChainValues(good string, multimodal *climgen.MultimodalTradeResult, limit int) string {
	values := make([]chainValue, 0, len(multimodal.Pairs))
	lines := make(map[string]string)
	for _, pair := range multimodal.Pairs {
		for _, flow := range pair.Goods {
			if flow.Good != good {
				continue
			}
			label := fmt.Sprintf("%d->%d", pair.FromPolity, pair.ToPolity)
			values = append(values, chainValue{label: label, value: flow.Score})
			lines[label] = fmt.Sprintf("%s:s%.2f/v%.2f", label, flow.Score, flow.Volume)
			break
		}
	}
	if len(values) == 0 {
		return "none"
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].value != values[j].value {
			return values[i].value > values[j].value
		}
		return values[i].label < values[j].label
	})
	if len(values) < limit {
		limit = len(values)
	}
	out := ""
	for i := 0; i < limit; i++ {
		if i > 0 {
			out += ", "
		}
		out += lines[values[i].label]
	}
	return out
}

func formatBlockedChainMarkets(processedGood string, markets *climgen.TradeNodeMarketResult, network *climgen.SettlementNetworkResult, limit int) string {
	values := make([]chainValue, 0)
	lines := make(map[string]string)
	for _, market := range markets.Markets {
		for _, blocked := range market.Diagnostics.Blocked {
			if blocked.Good != processedGood {
				continue
			}
			label := formatChainNodeLabel(market.NodeID, network)
			score := blocked.Priority
			if score <= 0 {
				score = reviewMaxFloat(blocked.DemandGap, blocked.Potential)
			}
			values = append(values, chainValue{label: label, value: score})
			lines[label] = fmt.Sprintf(
				"%s:p%.2f@%s raw=%.2f eff=%.2f need=%.2f procGap=%.2f rawGap=%.2f made=%.2f",
				label,
				score,
				blocked.Bottleneck,
				blocked.BottleneckAvailable,
				blocked.BottleneckEffective,
				blocked.BottleneckNeed,
				reviewMaxFloat(market.Demand[processedGood]-market.Supply[processedGood], 0),
				reviewMaxFloat(-market.Surplus[blocked.Bottleneck], 0),
				market.Manufactured[processedGood],
			)
			break
		}
	}
	if len(values) == 0 {
		return "none"
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].value != values[j].value {
			return values[i].value > values[j].value
		}
		return values[i].label < values[j].label
	})
	if len(values) < limit {
		limit = len(values)
	}
	out := ""
	for i := 0; i < limit; i++ {
		if i > 0 {
			out += ", "
		}
		out += lines[values[i].label]
	}
	return out
}

func formatChainNodeLabel(nodeID int, network *climgen.SettlementNetworkResult) string {
	if network != nil && nodeID >= 0 && nodeID < len(network.Nodes) {
		return fmt.Sprintf("%d %s", nodeID, climgen.SettlementNodeKindName(network.Nodes[nodeID].Kind))
	}
	return fmt.Sprintf("%d", nodeID)
}

func formatChainValues(values []chainValue, limit int) string {
	if len(values) == 0 {
		return "none"
	}
	if len(values) < limit {
		limit = len(values)
	}
	out := ""
	for i := 0; i < limit; i++ {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%s:%.2f", values[i].label, values[i].value)
	}
	return out
}
