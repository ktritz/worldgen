package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func printCoastalTradeSummary(result *climgen.CoastalTradeResult, network *climgen.SettlementNetworkResult) {
	if result == nil {
		return
	}
	if len(result.Corridors) == 0 {
		fmt.Printf("    coastalTrade[%s]: corridors=0 candidatePorts=%d stopovers=%d majorPorts=%d\n", result.Mode.Name, len(result.CandidatePorts), len(result.Stopovers), len(result.MajorPorts))
		fmt.Printf(
			"      endpointDiag: endpoints=%d ports=%d stopovers=%d pairs=%d distancePruned=%d edges=%d meanDegree=%.2f maxDegree=%d meanEdgeCost=%.2f p90EdgeCost=%.2f components=%d largest=%d portComponents=%d multiPortComponents=%d largestPorts=%d secondPorts=%d meanPortsPerComponent=%.2f isolatedPorts=%d\n",
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
		printEndpointComponentDetails("endpoint", result.EndpointDiagnostics, 4)
		if result.PairDiagnostics.TotalPairs > 0 {
			fmt.Printf(
				"      pairDiag: total=%d viable=%d viableRel=%d/%d/%d selectedRel=%d/%d/%d routeBudget=%.2f noPath=%d noPathRel=%d/%d/%d overBudgetSamples=%d overBudgetRel=%d/%d/%d bestOverBudget=%.2f pair=%d->%d civ=%d->%d bestExternalOverBudget=%.2f pair=%d->%d civ=%d->%d flowBelow=%d portCap=%d civCap=%d bestRejectedFlow=%.2f bestRejectedCost=%.2f pair=%d->%d\n",
				result.PairDiagnostics.TotalPairs,
				result.PairDiagnostics.ViableCandidates,
				result.PairDiagnostics.ViableInternal,
				result.PairDiagnostics.ViableExternal,
				result.PairDiagnostics.ViableUnknown,
				result.PairDiagnostics.SelectedInternal,
				result.PairDiagnostics.SelectedExternal,
				result.PairDiagnostics.SelectedUnknown,
				result.PairDiagnostics.RouteBudget,
				result.PairDiagnostics.NoPath,
				result.PairDiagnostics.NoPathInternal,
				result.PairDiagnostics.NoPathExternal,
				result.PairDiagnostics.NoPathUnknown,
				result.PairDiagnostics.OverBudgetSamples,
				result.PairDiagnostics.OverBudgetInternal,
				result.PairDiagnostics.OverBudgetExternal,
				result.PairDiagnostics.OverBudgetUnknown,
				result.PairDiagnostics.BestOverBudgetCost,
				result.PairDiagnostics.BestOverBudgetFrom,
				result.PairDiagnostics.BestOverBudgetTo,
				result.PairDiagnostics.BestOverBudgetFromCiv,
				result.PairDiagnostics.BestOverBudgetToCiv,
				result.PairDiagnostics.BestOverBudgetExternal,
				result.PairDiagnostics.BestOverBudgetExternalFrom,
				result.PairDiagnostics.BestOverBudgetExternalTo,
				result.PairDiagnostics.BestOverBudgetExternalFromCiv,
				result.PairDiagnostics.BestOverBudgetExternalToCiv,
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
	totalPathDegrees := 0.0
	totalCostPerDegree := 0.0
	totalEndpointScore := 0.0
	pathDegreeValues := make([]float64, 0, len(result.Corridors))
	costPerDegreeValues := make([]float64, 0, len(result.Corridors))
	inter := 0
	tierCounts := map[climgen.CoastalTradeCorridorTier]int{}
	for _, corridor := range result.Corridors {
		totalFlow += corridor.Flow
		totalExposure += corridor.MeanExposure
		totalAssist += corridor.MeanCurrentAssist
		totalPathDegrees += corridor.PathDegrees
		totalCostPerDegree += corridor.CostPerDegree
		totalEndpointScore += 0.5 * (corridor.FromPortScore + corridor.ToPortScore)
		pathDegreeValues = append(pathDegreeValues, corridor.PathDegrees)
		costPerDegreeValues = append(costPerDegreeValues, corridor.CostPerDegree)
		if corridor.InterCivilization {
			inter++
		}
		tierCounts[corridor.Tier]++
	}
	fmt.Printf(
		"    coastalTrade[%s]: corridors=%d candidatePorts=%d stopovers=%d majorPorts=%d inter=%d meanFlow=%.2f meanExposure=%.2f meanAssist=%.2f\n",
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
		"      routeGeomDiag: meanPathDeg=%.2f p50PathDeg=%.2f meanCostPerDeg=%.2f p50CostPerDeg=%.2f meanEndpointScore=%.2f\n",
		totalPathDegrees/float64(len(result.Corridors)),
		reviewMedianFloat64(pathDegreeValues),
		totalCostPerDegree/float64(len(result.Corridors)),
		reviewMedianFloat64(costPerDegreeValues),
		totalEndpointScore/float64(len(result.Corridors)),
	)
	fmt.Printf(
		"      pairDiag: total=%d viable=%d viableRel=%d/%d/%d selectedRel=%d/%d/%d routeBudget=%.2f noPath=%d noPathRel=%d/%d/%d overBudgetSamples=%d overBudgetRel=%d/%d/%d bestOverBudget=%.2f pair=%d->%d civ=%d->%d bestExternalOverBudget=%.2f pair=%d->%d civ=%d->%d flowBelow=%d portCap=%d civCap=%d\n",
		result.PairDiagnostics.TotalPairs,
		result.PairDiagnostics.ViableCandidates,
		result.PairDiagnostics.ViableInternal,
		result.PairDiagnostics.ViableExternal,
		result.PairDiagnostics.ViableUnknown,
		result.PairDiagnostics.SelectedInternal,
		result.PairDiagnostics.SelectedExternal,
		result.PairDiagnostics.SelectedUnknown,
		result.PairDiagnostics.RouteBudget,
		result.PairDiagnostics.NoPath,
		result.PairDiagnostics.NoPathInternal,
		result.PairDiagnostics.NoPathExternal,
		result.PairDiagnostics.NoPathUnknown,
		result.PairDiagnostics.OverBudgetSamples,
		result.PairDiagnostics.OverBudgetInternal,
		result.PairDiagnostics.OverBudgetExternal,
		result.PairDiagnostics.OverBudgetUnknown,
		result.PairDiagnostics.BestOverBudgetCost,
		result.PairDiagnostics.BestOverBudgetFrom,
		result.PairDiagnostics.BestOverBudgetTo,
		result.PairDiagnostics.BestOverBudgetFromCiv,
		result.PairDiagnostics.BestOverBudgetToCiv,
		result.PairDiagnostics.BestOverBudgetExternal,
		result.PairDiagnostics.BestOverBudgetExternalFrom,
		result.PairDiagnostics.BestOverBudgetExternalTo,
		result.PairDiagnostics.BestOverBudgetExternalFromCiv,
		result.PairDiagnostics.BestOverBudgetExternalToCiv,
		result.PairDiagnostics.FlowBelowMin,
		result.PairDiagnostics.RejectedPortCap,
		result.PairDiagnostics.RejectedCivCap,
	)
	fmt.Printf(
		"      endpointDiag: endpoints=%d ports=%d stopovers=%d pairs=%d distancePruned=%d edges=%d meanDegree=%.2f maxDegree=%d meanEdgeCost=%.2f p90EdgeCost=%.2f components=%d largest=%d portComponents=%d multiPortComponents=%d largestPorts=%d secondPorts=%d meanPortsPerComponent=%.2f isolatedPorts=%d\n",
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
	printEndpointComponentDetails("endpoint", result.EndpointDiagnostics, 4)
	printCoastalCandidatePortCivDiagnostics(result.CandidatePorts, network)
	fmt.Printf(
		"      stopoverDiag: candidates=%d candidateIsland=%d candidateStrait=%d candidateRoadstead=%d candidateTinyEq=%d candidateSmallEq=%d candidateMediumEq=%d candidateLargeEq=%d meanCandidateAreaEq=%.2f baseSelected=%d baseSpacingRejected=%d oceanScoreRejected=%d oceanSpacingRejected=%d selected=%d island=%d strait=%d roadstead=%d selectedTinyEq=%d selectedSmallEq=%d selectedMediumEq=%d selectedLargeEq=%d meanSelectedAreaEq=%.2f meanScore=%.2f meanNeighborDeg=%.3f minSpacingDeg=%.3f meanSelectedSpacingDeg=%.3f\n",
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
	for _, tier := range []climgen.CoastalTradeCorridorTier{
		climgen.CoastalTradeCorridorPrimary,
		climgen.CoastalTradeCorridorRegional,
		climgen.CoastalTradeCorridorLocal,
	} {
		if count := tierCounts[tier]; count > 0 {
			fmt.Printf("      coastalRoute[%s]=%d\n", climgen.CoastalTradeCorridorTierName(tier), count)
		}
	}
	printTopCoastalCorridorDiagnostics(result.Corridors, 5)
	printInterCoastalCorridorDiagnostics(result.Corridors, network, result, 8)
	limit := 4
	if len(result.MajorPorts) < limit {
		limit = len(result.MajorPorts)
	}
	for i := 0; i < limit; i++ {
		nodeIdx := result.MajorPorts[i]
		node := network.Nodes[nodeIdx]
		fmt.Printf(
			"      seaPort[%d]: kind=%s river=%v coastal=%v centrality=%.2f\n",
			nodeIdx,
			climgen.SettlementNodeKindName(node.Kind),
			node.River,
			node.Coastal,
			result.Diagnostics.NodeCentrality[nodeIdx],
		)
	}
	if len(result.Stopovers) > 0 {
		limit = 3
		if len(result.Stopovers) < limit {
			limit = len(result.Stopovers)
		}
		for i := 0; i < limit; i++ {
			stop := result.Stopovers[i]
			fmt.Printf("      stopover[%d]: kind=%s score=%.2f\n", stop.ID, climgen.MaritimeStopoverKindName(stop.Kind), stop.Score)
		}
	}
}

func printCoastalCandidatePortCivDiagnostics(candidatePorts []int, network *climgen.SettlementNetworkResult) {
	if len(candidatePorts) == 0 || network == nil || network.Diagnostics == nil || len(network.Diagnostics.RegionByNode) == 0 {
		return
	}
	countByRegion := map[int]int{}
	regionPorts := map[int][]int{}
	for _, nodeIdx := range candidatePorts {
		if nodeIdx < 0 || nodeIdx >= len(network.Diagnostics.RegionByNode) {
			continue
		}
		regionID := network.Diagnostics.RegionByNode[nodeIdx]
		if regionID < 0 {
			continue
		}
		countByRegion[regionID]++
		regionPorts[regionID] = append(regionPorts[regionID], nodeIdx)
	}
	regions := make([]int, 0, len(countByRegion))
	for regionID := range countByRegion {
		regions = append(regions, regionID)
	}
	sort.Ints(regions)
	for _, regionID := range regions {
		ports := regionPorts[regionID]
		sort.Ints(ports)
		fmt.Printf(
			"      candidatePortRegion[%d]: ports=%d nodes=%s\n",
			regionID,
			countByRegion[regionID],
			formatIntList(ports, 8),
		)
	}
}

func printInterCoastalCorridorDiagnostics(corridors []climgen.CoastalTradeCorridor, network *climgen.SettlementNetworkResult, result *climgen.CoastalTradeResult, limit int) {
	if len(corridors) == 0 || network == nil || result == nil || limit <= 0 {
		return
	}
	values := make([]climgen.CoastalTradeCorridor, 0)
	for _, corridor := range corridors {
		if corridor.InterCivilization {
			values = append(values, corridor)
		}
	}
	if len(values) == 0 {
		return
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Flow != values[j].Flow {
			return values[i].Flow > values[j].Flow
		}
		return values[i].TravelCost < values[j].TravelCost
	})
	if len(values) < limit {
		limit = len(values)
	}
	for i := 0; i < limit; i++ {
		corridor := values[i]
		fromKind := "Unknown"
		toKind := "Unknown"
		fromCentrality := 0.0
		toCentrality := 0.0
		if corridor.FromNode >= 0 && corridor.FromNode < len(network.Nodes) {
			fromKind = climgen.SettlementNodeKindName(network.Nodes[corridor.FromNode].Kind)
			if result.Diagnostics != nil && corridor.FromNode < len(result.Diagnostics.NodeCentrality) {
				fromCentrality = result.Diagnostics.NodeCentrality[corridor.FromNode]
			}
		}
		if corridor.ToNode >= 0 && corridor.ToNode < len(network.Nodes) {
			toKind = climgen.SettlementNodeKindName(network.Nodes[corridor.ToNode].Kind)
			if result.Diagnostics != nil && corridor.ToNode < len(result.Diagnostics.NodeCentrality) {
				toCentrality = result.Diagnostics.NodeCentrality[corridor.ToNode]
			}
		}
		fmt.Printf(
			"      coastalInterRouteDetail[%d]: nodes=%d->%d civ=%d->%d kind=%s->%s centrality=%.2f/%.2f flow=%.2f cost=%.2f pathDeg=%.2f costPerDeg=%.2f endpointScore=%.2f/%.2f cells=%d\n",
			corridor.ID,
			corridor.FromNode,
			corridor.ToNode,
			corridor.FromCivilization,
			corridor.ToCivilization,
			fromKind,
			toKind,
			fromCentrality,
			toCentrality,
			corridor.Flow,
			corridor.TravelCost,
			corridor.PathDegrees,
			corridor.CostPerDegree,
			corridor.FromPortScore,
			corridor.ToPortScore,
			len(corridor.CellPath),
		)
	}
}

func printTopCoastalCorridorDiagnostics(corridors []climgen.CoastalTradeCorridor, limit int) {
	if len(corridors) == 0 || limit <= 0 {
		return
	}
	values := append([]climgen.CoastalTradeCorridor(nil), corridors...)
	sort.Slice(values, func(i, j int) bool {
		if values[i].Flow != values[j].Flow {
			return values[i].Flow > values[j].Flow
		}
		return values[i].TravelCost < values[j].TravelCost
	})
	if len(values) < limit {
		limit = len(values)
	}
	for i := 0; i < limit; i++ {
		corridor := values[i]
		fmt.Printf(
			"      coastalRouteDetail[%d]: nodes=%d->%d civ=%d->%d flow=%.2f cost=%.2f pathDeg=%.2f costPerDeg=%.2f endpointScore=%.2f/%.2f cells=%d inter=%v\n",
			corridor.ID,
			corridor.FromNode,
			corridor.ToNode,
			corridor.FromCivilization,
			corridor.ToCivilization,
			corridor.Flow,
			corridor.TravelCost,
			corridor.PathDegrees,
			corridor.CostPerDegree,
			corridor.FromPortScore,
			corridor.ToPortScore,
			len(corridor.CellPath),
			corridor.InterCivilization,
		)
	}
}

func reviewMedianFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return 0.5 * (sorted[mid-1] + sorted[mid])
}
