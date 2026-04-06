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
	roleCounts := make(map[climgen.TradeCorridorRole]int)
	totalFlow := 0.0
	totalRisk := 0.0
	totalSupport := 0.0
	modeCounts := make(map[string]int)
	for _, corridor := range result.Corridors {
		tierCounts[corridor.Tier]++
		roleCounts[corridor.Role]++
		totalFlow += corridor.Flow
		totalRisk += corridor.MeanRisk
		totalSupport += corridor.MeanSupport
		modeCounts[corridor.Mode]++
	}
	fmt.Printf(
		"    tradeNetwork: corridors=%d hubs=%d localNodes=%d meanFlow=%.2f meanRisk=%.2f meanSupport=%.2f\n",
		len(result.Corridors),
		len(result.MajorHubs),
		len(result.LocalNodes),
		totalFlow/float64(len(result.Corridors)),
		totalRisk/float64(len(result.Corridors)),
		totalSupport/float64(len(result.Corridors)),
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
	type roleCount struct {
		role  climgen.TradeCorridorRole
		count int
	}
	roles := make([]roleCount, 0, len(roleCounts))
	for role, count := range roleCounts {
		roles = append(roles, roleCount{role: role, count: count})
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i].role < roles[j].role })
	for _, entry := range roles {
		fmt.Printf("      routeRole[%s]=%d\n", climgen.TradeCorridorRoleName(entry.role), entry.count)
	}
	for mode, count := range modeCounts {
		fmt.Printf("      routeMode[%s]=%d\n", mode, count)
	}
	localKindCounts := make(map[climgen.LocalTradeNodeKind]int)
	for _, node := range result.LocalNodes {
		localKindCounts[node.Kind]++
	}
	type localKindCount struct {
		kind  climgen.LocalTradeNodeKind
		count int
	}
	localKinds := make([]localKindCount, 0, len(localKindCounts))
	for kind, count := range localKindCounts {
		localKinds = append(localKinds, localKindCount{kind: kind, count: count})
	}
	sort.Slice(localKinds, func(i, j int) bool { return localKinds[i].kind < localKinds[j].kind })
	for _, entry := range localKinds {
		fmt.Printf("      localNode[%s]=%d\n", climgen.LocalTradeNodeKindName(entry.kind), entry.count)
	}
	limitHandoff := 3
	if len(result.Handoffs) < limitHandoff {
		limitHandoff = len(result.Handoffs)
	}
	for i := 0; i < limitHandoff; i++ {
		handoff := result.Handoffs[i]
		fmt.Printf(
			"      handoff[%d]: feeder=%d trunk=%d mode=%s hub=%v\n",
			handoff.Node,
			handoff.FeederCorridors,
			handoff.TrunkCorridors,
			handoff.DominantMode,
			handoff.Hub,
		)
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
