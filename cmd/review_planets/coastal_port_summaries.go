package main

import (
	"fmt"

	"worldgen/climgen"
)

func printCoastalPortSummary(result *climgen.CoastalPortResult, network *climgen.SettlementNetworkResult) {
	if result == nil || result.Diagnostics == nil || len(result.Types) == 0 {
		return
	}
	n := float64(len(result.Types))
	suitable := 0
	deepwaterSuitable := 0
	sumSuitability := 0.0
	sumDeepwater := 0.0
	typeCounts := map[climgen.CoastalPortType]int{}
	for i, kind := range result.Types {
		if kind == climgen.CoastalPortOcean {
			continue
		}
		typeCounts[kind]++
		sumSuitability += result.Diagnostics.PortSuitability[i]
		if i < len(result.Diagnostics.DeepwaterSuitability) {
			sumDeepwater += result.Diagnostics.DeepwaterSuitability[i]
			if result.Diagnostics.DeepwaterSuitability[i] >= 0.24 {
				deepwaterSuitable++
			}
		}
		if result.Diagnostics.PortSuitability[i] >= 0.24 {
			suitable++
		}
	}
	fmt.Printf(
		"    coastalPorts[%s]: suitable=%.1f%% deepwater=%.1f%% major=%d deepMajor=%d meanSuit=%.2f meanDeep=%.2f\n",
		result.Mode.Name,
		100*float64(suitable)/n,
		100*float64(deepwaterSuitable)/n,
		len(result.MajorPorts),
		len(result.MajorDeepwaterPorts),
		sumSuitability/n,
		sumDeepwater/n,
	)
	for _, kind := range []climgen.CoastalPortType{
		climgen.CoastalPortHarbor,
		climgen.CoastalPortEstuary,
		climgen.CoastalPortIslandStopover,
		climgen.CoastalPortBeachLanding,
	} {
		if count := typeCounts[kind]; count > 0 {
			fmt.Printf("      portType[%s]=%d\n", climgen.CoastalPortTypeName(kind), count)
		}
	}
	if network == nil || result.Diagnostics == nil {
		return
	}
	limit := 4
	if len(result.MajorPorts) < limit {
		limit = len(result.MajorPorts)
	}
	for i := 0; i < limit; i++ {
		nodeIdx := result.MajorPorts[i]
		node := network.Nodes[nodeIdx]
		score := 0.0
		if nodeIdx < len(result.Diagnostics.NodePortScore) {
			score = result.Diagnostics.NodePortScore[nodeIdx]
		}
		fmt.Printf(
			"      coastalPort[%d]: kind=%s river=%v coastal=%v score=%.2f\n",
			nodeIdx,
			climgen.SettlementNodeKindName(node.Kind),
			node.River,
			node.Coastal,
			score,
		)
	}
	limit = 3
	if len(result.MajorDeepwaterPorts) < limit {
		limit = len(result.MajorDeepwaterPorts)
	}
	for i := 0; i < limit; i++ {
		nodeIdx := result.MajorDeepwaterPorts[i]
		node := network.Nodes[nodeIdx]
		score := 0.0
		if nodeIdx < len(result.Diagnostics.NodeDeepwaterScore) {
			score = result.Diagnostics.NodeDeepwaterScore[nodeIdx]
		}
		fmt.Printf(
			"      deepwaterPort[%d]: kind=%s river=%v coastal=%v score=%.2f\n",
			nodeIdx,
			climgen.SettlementNodeKindName(node.Kind),
			node.River,
			node.Coastal,
			score,
		)
	}
}
