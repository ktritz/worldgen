package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func printRiverRouteSummary(result *climgen.RiverRouteResult) {
	if result == nil || result.Diagnostics == nil || len(result.Diagnostics.Navigability) == 0 {
		return
	}
	n := float64(len(result.Diagnostics.Navigability))
	sumNav := 0.0
	sumTransfer := 0.0
	usable := 0
	mainChannel := 0
	for i := range result.Diagnostics.Navigability {
		nav := result.Diagnostics.Navigability[i]
		sumNav += nav
		sumTransfer += result.Diagnostics.TransferSupport[i]
		if nav > 0 {
			usable++
		}
		if result.Diagnostics.MainChannel[i] >= 0.55 {
			mainChannel++
		}
	}
	fmt.Printf(
		"    riverRoutes[%s]: navigable=%.1f%% mainChannel=%.1f%% meanNav=%.2f meanTransfer=%.2f\n",
		result.Mode.Name,
		100*float64(usable)/n,
		100*float64(mainChannel)/n,
		sumNav/n,
		sumTransfer/n,
	)
}

func printRiverTradeSummary(result *climgen.RiverTradeResult, network *climgen.SettlementNetworkResult) {
	if result == nil {
		return
	}
	if len(result.Corridors) == 0 {
		fmt.Println("    riverTrade: corridors=0 ports=0")
		return
	}
	tierCounts := make(map[climgen.RiverTradeCorridorTier]int)
	totalFlow := 0.0
	totalNav := 0.0
	totalTransfer := 0.0
	inter := 0
	for _, corridor := range result.Corridors {
		tierCounts[corridor.Tier]++
		totalFlow += corridor.Flow
		totalNav += corridor.MeanNavigability
		totalTransfer += corridor.MeanTransfer
		if corridor.InterCivilization {
			inter++
		}
	}
	fmt.Printf(
		"    riverTrade: corridors=%d ports=%d inter=%d meanFlow=%.2f meanNav=%.2f meanTransfer=%.2f\n",
		len(result.Corridors),
		len(result.MajorPorts),
		inter,
		totalFlow/float64(len(result.Corridors)),
		totalNav/float64(len(result.Corridors)),
		totalTransfer/float64(len(result.Corridors)),
	)
	type tierCount struct {
		tier  climgen.RiverTradeCorridorTier
		count int
	}
	tiers := make([]tierCount, 0, len(tierCounts))
	for tier, count := range tierCounts {
		tiers = append(tiers, tierCount{tier: tier, count: count})
	}
	sort.Slice(tiers, func(i, j int) bool { return tiers[i].tier > tiers[j].tier })
	for _, entry := range tiers {
		fmt.Printf("      riverRoute[%s]=%d\n", climgen.RiverTradeCorridorTierName(entry.tier), entry.count)
	}
	limit := 4
	if len(result.MajorPorts) < limit {
		limit = len(result.MajorPorts)
	}
	for i := 0; i < limit; i++ {
		nodeIdx := result.MajorPorts[i]
		node := network.Nodes[nodeIdx]
		fmt.Printf(
			"      riverPort[%d]: kind=%s river=%v coastal=%v centrality=%.2f\n",
			nodeIdx,
			climgen.SettlementNodeKindName(node.Kind),
			node.River,
			node.Coastal,
			result.Diagnostics.NodeCentrality[nodeIdx],
		)
	}
}

