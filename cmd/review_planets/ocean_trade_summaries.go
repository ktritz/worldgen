package main

import (
	"fmt"

	"worldgen/climgen"
)

func printOceanTradeSummary(result *climgen.OceanTradeResult, network *climgen.SettlementNetworkResult) {
	if result == nil {
		return
	}
	if len(result.Corridors) == 0 {
		fmt.Printf("    oceanTrade[%s]: corridors=0 candidatePorts=%d stopovers=%d majorPorts=%d\n", result.Mode.Name, len(result.CandidatePorts), len(result.Stopovers), len(result.MajorPorts))
		fmt.Printf(
			"      oceanEndpointDiag: endpoints=%d ports=%d stopovers=%d edges=%d components=%d largest=%d largestPorts=%d isolatedPorts=%d\n",
			result.EndpointDiagnostics.EndpointCount,
			result.EndpointDiagnostics.PortEndpointCount,
			result.EndpointDiagnostics.StopoverCount,
			result.EndpointDiagnostics.EdgeCount,
			result.EndpointDiagnostics.Components,
			result.EndpointDiagnostics.LargestComponent,
			result.EndpointDiagnostics.LargestPortComponent,
			result.EndpointDiagnostics.IsolatedPorts,
		)
		if result.PairDiagnostics.TotalPairs > 0 {
			fmt.Printf(
				"      oceanPairDiag: total=%d viable=%d noPath=%d flowBelow=%d portCap=%d civCap=%d bestRejectedFlow=%.2f bestRejectedCost=%.2f pair=%d->%d\n",
				result.PairDiagnostics.TotalPairs,
				result.PairDiagnostics.ViableCandidates,
				result.PairDiagnostics.NoPath,
				result.PairDiagnostics.FlowBelowMin,
				result.PairDiagnostics.RejectedPortCap,
				result.PairDiagnostics.RejectedCivCap,
				result.PairDiagnostics.BestRejectedFlow,
				result.PairDiagnostics.BestRejectedCost,
				result.PairDiagnostics.BestRejectedFrom,
				result.PairDiagnostics.BestRejectedTo,
			)
		}
		return
	}
	totalFlow := 0.0
	totalExposure := 0.0
	totalAssist := 0.0
	inter := 0
	tierCounts := map[climgen.OceanTradeCorridorTier]int{}
	for _, corridor := range result.Corridors {
		totalFlow += corridor.Flow
		totalExposure += corridor.MeanExposure
		totalAssist += corridor.MeanCurrentAssist
		if corridor.InterCivilization {
			inter++
		}
		tierCounts[corridor.Tier]++
	}
	fmt.Printf(
		"    oceanTrade[%s]: corridors=%d candidatePorts=%d stopovers=%d majorPorts=%d inter=%d meanFlow=%.2f meanExposure=%.2f meanAssist=%.2f\n",
		result.Mode.Name,
		len(result.Corridors),
		len(result.CandidatePorts),
		len(result.Stopovers),
		len(result.MajorPorts),
		inter,
		totalFlow/float64(len(result.Corridors)),
		totalExposure/float64(len(result.Corridors)),
		totalAssist/float64(len(result.Corridors)),
	)
	fmt.Printf(
		"      oceanPairDiag: total=%d viable=%d noPath=%d flowBelow=%d portCap=%d civCap=%d\n",
		result.PairDiagnostics.TotalPairs,
		result.PairDiagnostics.ViableCandidates,
		result.PairDiagnostics.NoPath,
		result.PairDiagnostics.FlowBelowMin,
		result.PairDiagnostics.RejectedPortCap,
		result.PairDiagnostics.RejectedCivCap,
	)
	fmt.Printf(
		"      oceanEndpointDiag: endpoints=%d ports=%d stopovers=%d edges=%d components=%d largest=%d largestPorts=%d isolatedPorts=%d\n",
		result.EndpointDiagnostics.EndpointCount,
		result.EndpointDiagnostics.PortEndpointCount,
		result.EndpointDiagnostics.StopoverCount,
		result.EndpointDiagnostics.EdgeCount,
		result.EndpointDiagnostics.Components,
		result.EndpointDiagnostics.LargestComponent,
		result.EndpointDiagnostics.LargestPortComponent,
		result.EndpointDiagnostics.IsolatedPorts,
	)
	for _, tier := range []climgen.OceanTradeCorridorTier{
		climgen.OceanTradeCorridorPrimary,
		climgen.OceanTradeCorridorRegional,
		climgen.OceanTradeCorridorLocal,
	} {
		if count := tierCounts[tier]; count > 0 {
			fmt.Printf("      oceanRoute[%s]=%d\n", climgen.OceanTradeCorridorTierName(tier), count)
		}
	}
	limit := 4
	if len(result.MajorPorts) < limit {
		limit = len(result.MajorPorts)
	}
	for i := 0; i < limit; i++ {
		nodeIdx := result.MajorPorts[i]
		node := network.Nodes[nodeIdx]
		fmt.Printf(
			"      oceanPort[%d]: kind=%s river=%v coastal=%v centrality=%.2f\n",
			nodeIdx,
			climgen.SettlementNodeKindName(node.Kind),
			node.River,
			node.Coastal,
			result.Diagnostics.NodeCentrality[nodeIdx],
		)
	}
}
