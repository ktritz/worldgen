package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func printTradeNetworkSummary(result *climgen.TradeNetworkResult, network *climgen.SettlementNetworkResult) {
	if result == nil {
		return
	}
	if len(result.Corridors) == 0 {
		fmt.Println("    tradeNetwork: corridors=0 hubs=0")
		return
	}
	tierCounts := make(map[climgen.TradeCorridorTier]int)
	totalFlow := 0.0
	for _, corridor := range result.Corridors {
		tierCounts[corridor.Tier]++
		totalFlow += corridor.Flow
	}
	fmt.Printf(
		"    tradeNetwork: corridors=%d hubs=%d meanFlow=%.2f\n",
		len(result.Corridors),
		len(result.MajorHubs),
		totalFlow/float64(len(result.Corridors)),
	)
	type tierCount struct {
		tier  climgen.TradeCorridorTier
		count int
	}
	tiers := make([]tierCount, 0, len(tierCounts))
	for tier, count := range tierCounts {
		tiers = append(tiers, tierCount{tier: tier, count: count})
	}
	sort.Slice(tiers, func(i, j int) bool { return tiers[i].tier > tiers[j].tier })
	for _, entry := range tiers {
		fmt.Printf("      route[%s]=%d\n", climgen.TradeCorridorTierName(entry.tier), entry.count)
	}
	limit := 4
	if len(result.MajorHubs) < limit {
		limit = len(result.MajorHubs)
	}
	for i := 0; i < limit; i++ {
		nodeIdx := result.MajorHubs[i]
		node := network.Nodes[nodeIdx]
		fmt.Printf(
			"      hub[%d]: kind=%s coastal=%v river=%v centrality=%.2f score=%.2f\n",
			nodeIdx,
			climgen.SettlementNodeKindName(node.Kind),
			node.Coastal,
			node.River,
			result.Diagnostics.NodeCentrality[nodeIdx],
			result.Diagnostics.HubScore[nodeIdx],
		)
	}
}
