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
				"%s:p%.2f@%s raw=%.2f eff=%.2f need=%.2f",
				label,
				score,
				blocked.Bottleneck,
				blocked.BottleneckAvailable,
				blocked.BottleneckEffective,
				blocked.BottleneckNeed,
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
