package main

import (
	"fmt"
	"strings"

	"worldgen/climgen"
)

func printOceanTradeSummary(
	result *climgen.OceanTradeResult,
	network *climgen.SettlementNetworkResult,
	ports *climgen.CoastalPortResult,
	denominators climgen.MeshDensityDenominators,
) {
	if result == nil {
		return
	}
	if len(result.Corridors) == 0 {
		fmt.Printf(
			"    oceanTrade[%s]: corridors=0 candidatePorts=%d stopovers=%d majorPorts=%d coastLength=%.6f oceanArea=%.6f portsPerCoastLength=%.6f stopoversPerOceanArea=%.6f oceanCorridorsPerOceanArea=%.6f\n",
			result.Mode.Name,
			len(result.CandidatePorts),
			len(result.Stopovers),
			len(result.MajorPorts),
			denominators.CoastLength,
			denominators.OceanArea,
			climgen.PerExtent(len(result.CandidatePorts), denominators.CoastLength),
			climgen.PerExtent(len(result.Stopovers), denominators.OceanArea),
			0.0,
		)
		fmt.Printf(
			"      oceanEndpointDiag: endpoints=%d ports=%d stopovers=%d pairs=%d distancePruned=%d edges=%d meanDegree=%.2f maxDegree=%d meanEdgeCost=%.2f p90EdgeCost=%.2f components=%d largest=%d portComponents=%d multiPortComponents=%d largestPorts=%d secondPorts=%d meanPortsPerComponent=%.2f isolatedPorts=%d\n",
			result.EndpointDiagnostics.EndpointCount,
			result.EndpointDiagnostics.PortEndpointCount,
			result.EndpointDiagnostics.StopoverCount,
			result.EndpointDiagnostics.PairCount,
			result.EndpointDiagnostics.DistancePrunedPairs,
			result.EndpointDiagnostics.EdgeCount,
			result.EndpointDiagnostics.MeanDegree,
			result.EndpointDiagnostics.MaxDegree,
			result.EndpointDiagnostics.MeanEdgeCost,
			result.EndpointDiagnostics.P90EdgeCost,
			result.EndpointDiagnostics.Components,
			result.EndpointDiagnostics.LargestComponent,
			result.EndpointDiagnostics.PortComponents,
			result.EndpointDiagnostics.MultiPortComponents,
			result.EndpointDiagnostics.LargestPortComponent,
			result.EndpointDiagnostics.SecondPortComponent,
			result.EndpointDiagnostics.MeanPortComponent,
			result.EndpointDiagnostics.IsolatedPorts,
		)
		printEndpointComponentDetails("oceanEndpoint", result.EndpointDiagnostics, 4)
		if result.PairDiagnostics.TotalPairs > 0 {
			fmt.Printf(
				"      oceanPairDiag: total=%d viable=%d viableRel=%d/%d/%d selectedRel=%d/%d/%d noPath=%d flowBelow=%d portCap=%d civCap=%d bestRejectedFlow=%.2f bestRejectedCost=%.2f pair=%d->%d\n",
				result.PairDiagnostics.TotalPairs,
				result.PairDiagnostics.ViableCandidates,
				result.PairDiagnostics.ViableInternal,
				result.PairDiagnostics.ViableExternal,
				result.PairDiagnostics.ViableUnknown,
				result.PairDiagnostics.SelectedInternal,
				result.PairDiagnostics.SelectedExternal,
				result.PairDiagnostics.SelectedUnknown,
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
		printOceanCandidatePortDiagnostics(result.CandidateDiagnostics)
		printOceanCandidatePortDetails(result, network, ports, 8)
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
		"    oceanTrade[%s]: corridors=%d candidatePorts=%d stopovers=%d majorPorts=%d inter=%d meanFlow=%.2f meanExposure=%.2f meanAssist=%.2f coastLength=%.6f oceanArea=%.6f portsPerCoastLength=%.6f stopoversPerOceanArea=%.6f oceanCorridorsPerOceanArea=%.6f\n",
		result.Mode.Name,
		len(result.Corridors),
		len(result.CandidatePorts),
		len(result.Stopovers),
		len(result.MajorPorts),
		inter,
		totalFlow/float64(len(result.Corridors)),
		totalExposure/float64(len(result.Corridors)),
		totalAssist/float64(len(result.Corridors)),
		denominators.CoastLength,
		denominators.OceanArea,
		climgen.PerExtent(len(result.CandidatePorts), denominators.CoastLength),
		climgen.PerExtent(len(result.Stopovers), denominators.OceanArea),
		climgen.PerExtent(len(result.Corridors), denominators.OceanArea),
	)
	fmt.Printf(
		"      oceanPairDiag: total=%d viable=%d viableRel=%d/%d/%d selectedRel=%d/%d/%d noPath=%d flowBelow=%d portCap=%d civCap=%d\n",
		result.PairDiagnostics.TotalPairs,
		result.PairDiagnostics.ViableCandidates,
		result.PairDiagnostics.ViableInternal,
		result.PairDiagnostics.ViableExternal,
		result.PairDiagnostics.ViableUnknown,
		result.PairDiagnostics.SelectedInternal,
		result.PairDiagnostics.SelectedExternal,
		result.PairDiagnostics.SelectedUnknown,
		result.PairDiagnostics.NoPath,
		result.PairDiagnostics.FlowBelowMin,
		result.PairDiagnostics.RejectedPortCap,
		result.PairDiagnostics.RejectedCivCap,
	)
	printOceanCandidatePortDiagnostics(result.CandidateDiagnostics)
	printOceanCandidatePortDetails(result, network, ports, 8)
	fmt.Printf(
		"      oceanEndpointDiag: endpoints=%d ports=%d stopovers=%d pairs=%d distancePruned=%d edges=%d meanDegree=%.2f maxDegree=%d meanEdgeCost=%.2f p90EdgeCost=%.2f components=%d largest=%d portComponents=%d multiPortComponents=%d largestPorts=%d secondPorts=%d meanPortsPerComponent=%.2f isolatedPorts=%d\n",
		result.EndpointDiagnostics.EndpointCount,
		result.EndpointDiagnostics.PortEndpointCount,
		result.EndpointDiagnostics.StopoverCount,
		result.EndpointDiagnostics.PairCount,
		result.EndpointDiagnostics.DistancePrunedPairs,
		result.EndpointDiagnostics.EdgeCount,
		result.EndpointDiagnostics.MeanDegree,
		result.EndpointDiagnostics.MaxDegree,
		result.EndpointDiagnostics.MeanEdgeCost,
		result.EndpointDiagnostics.P90EdgeCost,
		result.EndpointDiagnostics.Components,
		result.EndpointDiagnostics.LargestComponent,
		result.EndpointDiagnostics.PortComponents,
		result.EndpointDiagnostics.MultiPortComponents,
		result.EndpointDiagnostics.LargestPortComponent,
		result.EndpointDiagnostics.SecondPortComponent,
		result.EndpointDiagnostics.MeanPortComponent,
		result.EndpointDiagnostics.IsolatedPorts,
	)
	printEndpointComponentDetails("oceanEndpoint", result.EndpointDiagnostics, 4)
	fmt.Printf(
		"      oceanStopoverDiag: candidates=%d candidateIsland=%d candidateStrait=%d candidateRoadstead=%d candidateTinyEq=%d candidateSmallEq=%d candidateMediumEq=%d candidateLargeEq=%d meanCandidateAreaEq=%.2f baseSelected=%d baseSpacingRejected=%d oceanScoreRejected=%d oceanSpacingRejected=%d selected=%d island=%d strait=%d roadstead=%d selectedTinyEq=%d selectedSmallEq=%d selectedMediumEq=%d selectedLargeEq=%d meanSelectedAreaEq=%.2f meanScore=%.2f meanNeighborDeg=%.3f minSpacingDeg=%.3f meanSelectedSpacingDeg=%.3f\n",
		result.StopoverDiagnostics.CandidateCount,
		result.StopoverDiagnostics.CandidateIslandCount,
		result.StopoverDiagnostics.CandidateStraitCount,
		result.StopoverDiagnostics.CandidateRoadsteadCount,
		result.StopoverDiagnostics.CandidateTinyComponentEq,
		result.StopoverDiagnostics.CandidateSmallComponentEq,
		result.StopoverDiagnostics.CandidateMediumComponentEq,
		result.StopoverDiagnostics.CandidateLargeComponentEq,
		result.StopoverDiagnostics.MeanCandidateComponentEq,
		result.StopoverDiagnostics.BaseSelectedCount,
		result.StopoverDiagnostics.BaseSpacingRejected,
		result.StopoverDiagnostics.OceanScoreRejected,
		result.StopoverDiagnostics.OceanSpacingRejected,
		result.StopoverDiagnostics.SelectedCount,
		result.StopoverDiagnostics.IslandCount,
		result.StopoverDiagnostics.StraitCount,
		result.StopoverDiagnostics.RoadsteadCount,
		result.StopoverDiagnostics.SelectedTinyComponentEq,
		result.StopoverDiagnostics.SelectedSmallComponentEq,
		result.StopoverDiagnostics.SelectedMediumComponentEq,
		result.StopoverDiagnostics.SelectedLargeComponentEq,
		result.StopoverDiagnostics.MeanSelectedComponentEq,
		result.StopoverDiagnostics.MeanScore,
		result.StopoverDiagnostics.MeanNeighborDegrees,
		result.StopoverDiagnostics.MinSpacingDegrees,
		result.StopoverDiagnostics.MeanSelectedSpacingDegrees,
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

func printOceanCandidatePortDetails(result *climgen.OceanTradeResult, network *climgen.SettlementNetworkResult, ports *climgen.CoastalPortResult, limit int) {
	if result == nil || network == nil || ports == nil || ports.Diagnostics == nil || len(result.CandidatePorts) == 0 {
		return
	}
	if len(result.CandidatePorts) < limit {
		limit = len(result.CandidatePorts)
	}
	for i := 0; i < limit; i++ {
		nodeIdx := result.CandidatePorts[i]
		if nodeIdx < 0 || nodeIdx >= len(network.Nodes) {
			continue
		}
		node := network.Nodes[nodeIdx]
		baseDeep := 0.0
		if nodeIdx < len(ports.Diagnostics.NodeBaseDeepwaterScore) {
			baseDeep = ports.Diagnostics.NodeBaseDeepwaterScore[nodeIdx]
		}
		deepScore := 0.0
		if nodeIdx < len(ports.Diagnostics.NodeDeepwaterScore) {
			deepScore = ports.Diagnostics.NodeDeepwaterScore[nodeIdx]
		}
		centrality := 0.0
		if result.Diagnostics != nil && nodeIdx < len(result.Diagnostics.NodeCentrality) {
			centrality = result.Diagnostics.NodeCentrality[nodeIdx]
		}
		fmt.Printf(
			"      oceanCandidate[%d]: node=%d kind=%s cell=%d support=%.2f coastal=%v river=%v baseDeep=%.2f deepScore=%.2f centrality=%.2f\n",
			i,
			nodeIdx,
			climgen.SettlementNodeKindName(node.Kind),
			node.CellIndex,
			node.PhysicalSupportArea,
			node.Coastal,
			node.River,
			baseDeep,
			deepScore,
			centrality,
		)
	}
}

func printOceanCandidateScoreDistribution(dist climgen.OceanCandidateScoreDistribution) {
	if dist.EligibleNodes == 0 {
		return
	}
	scoreParts := make([]string, 0, len(dist.ScoreThresholds))
	for i, threshold := range dist.ScoreThresholds {
		count, civilized, both := 0, 0, 0
		if i < len(dist.ScoreCounts) {
			count = dist.ScoreCounts[i]
		}
		if i < len(dist.ScoreCivilized) {
			civilized = dist.ScoreCivilized[i]
		}
		if i < len(dist.ScorePassBoth) {
			both = dist.ScorePassBoth[i]
		}
		scoreParts = append(scoreParts, fmt.Sprintf("%.2f:%d/%d/%d", threshold, count, civilized, both))
	}
	physParts := make([]string, 0, len(dist.PhysicalThreshold))
	for i, threshold := range dist.PhysicalThreshold {
		count := 0
		if i < len(dist.PhysicalCounts) {
			count = dist.PhysicalCounts[i]
		}
		physParts = append(physParts, fmt.Sprintf("%.2f:%d", threshold, count))
	}
	fmt.Printf(
		"      oceanPortDist: eligible=%d civilized=%d scoreP50=%.3f scoreP75=%.3f scoreP90=%.3f scoreP95=%.3f scoreMax=%.3f physP50=%.3f physP75=%.3f physP90=%.3f physMax=%.3f scoreAtLeast[all/civ/both]=%s physAtLeast=%s\n",
		dist.EligibleNodes,
		dist.CivilizedNodes,
		dist.ScoreP50,
		dist.ScoreP75,
		dist.ScoreP90,
		dist.ScoreP95,
		dist.ScoreMax,
		dist.PhysicalP50,
		dist.PhysicalP75,
		dist.PhysicalP90,
		dist.PhysicalMax,
		strings.Join(scoreParts, ","),
		strings.Join(physParts, ","),
	)
}

func printOceanCandidatePortDiagnostics(diag climgen.OceanCandidatePortDiagnostics) {
	printOceanCandidateScoreDistribution(diag.ScoreDistribution)
	if diag.RawCandidateCount == 0 && diag.CivilizedCandidateCount == 0 {
		return
	}
	fmt.Printf(
		"      oceanPortDiag: raw=%d civilized=%d final=%d uncivilized=%d civCapRejected=%d rawPhys>=0.30=%d rawPhys>=0.36=%d finalPhys>=0.30=%d finalPhys>=0.36=%d candidateMajor=%d candidateSecondary=%d rawMajor=%d rawSecondary=%d civs=%d multiPortCivs=%d meanPortsPerCiv=%.2f maxPortsPerCiv=%d scoreMean=%.2f scoreP10=%.2f scoreP50=%.2f scoreP90=%.2f\n",
		diag.RawCandidateCount,
		diag.CivilizedCandidateCount,
		diag.FinalCandidateCount,
		diag.UncivilizedRejected,
		diag.CivilizationCapRejected,
		diag.RawPhysical030,
		diag.RawPhysical036,
		diag.FinalPhysical030,
		diag.FinalPhysical036,
		diag.CandidateMajorPorts,
		diag.CandidateSecondaryPorts,
		diag.RawMajorPorts,
		diag.RawSecondaryPorts,
		diag.CandidateCivilizations,
		diag.CivilizationsWithMultiPorts,
		diag.MeanPortsPerCivilization,
		diag.MaxPortsPerCivilization,
		diag.MeanDeepwaterScore,
		diag.P10DeepwaterScore,
		diag.MedianDeepwaterScore,
		diag.P90DeepwaterScore,
	)
}
