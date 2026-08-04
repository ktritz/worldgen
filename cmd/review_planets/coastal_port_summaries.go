package main

import (
	"fmt"
	"sort"

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
	printCoastalPortNodeDiagnostics(result)
	printCoastalPortRegionDiagnostics(result, network)
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

func printCoastalPortRegionDiagnostics(result *climgen.CoastalPortResult, network *climgen.SettlementNetworkResult) {
	if result == nil || result.Diagnostics == nil || network == nil || network.Diagnostics == nil || len(network.Diagnostics.RegionByNode) == 0 {
		return
	}
	type regionPortDiag struct {
		regionID     int
		nodes        int
		terminals    int
		bestScore    float64
		bestNode     int
		bestTerminal int
	}
	byRegion := map[int]*regionPortDiag{}
	for nodeIdx := range network.Nodes {
		if nodeIdx >= len(network.Diagnostics.RegionByNode) {
			continue
		}
		regionID := network.Diagnostics.RegionByNode[nodeIdx]
		if regionID < 0 {
			continue
		}
		entry := byRegion[regionID]
		if entry == nil {
			entry = &regionPortDiag{regionID: regionID, bestNode: -1, bestTerminal: -1}
			byRegion[regionID] = entry
		}
		entry.nodes++
		terminal := -1
		if nodeIdx < len(result.Diagnostics.NodeTerminalCell) {
			terminal = result.Diagnostics.NodeTerminalCell[nodeIdx]
		}
		if terminal >= 0 {
			entry.terminals++
		}
		score := 0.0
		if nodeIdx < len(result.Diagnostics.NodePortScore) {
			score = result.Diagnostics.NodePortScore[nodeIdx]
		}
		if score > entry.bestScore || entry.bestNode < 0 {
			entry.bestScore = score
			entry.bestNode = nodeIdx
			entry.bestTerminal = terminal
		}
	}
	regions := make([]int, 0, len(byRegion))
	for regionID := range byRegion {
		regions = append(regions, regionID)
	}
	sort.Ints(regions)
	for _, regionID := range regions {
		entry := byRegion[regionID]
		fmt.Printf(
			"      portRegion[%d]: nodes=%d terminals=%d bestNode=%d bestScore=%.2f bestTerminal=%d\n",
			entry.regionID,
			entry.nodes,
			entry.terminals,
			entry.bestNode,
			entry.bestScore,
			entry.bestTerminal,
		)
	}
}

func printCoastalPortNodeDiagnostics(result *climgen.CoastalPortResult) {
	if result == nil || result.Diagnostics == nil {
		return
	}
	deepScores := make([]float64, 0, len(result.Diagnostics.NodeDeepwaterScore))
	baseDeepScores := make([]float64, 0, len(result.Diagnostics.NodeDeepwaterScore))
	portScores := make([]float64, 0, len(result.Diagnostics.NodePortScore))
	deepTerminals := 0
	portTerminals := 0
	deepScore048 := 0
	deepScore056 := 0
	baseDeepScore024 := 0
	baseDeepScore030 := 0
	baseDeepScore036 := 0
	baseDeepScore048 := 0
	baseDeepScore056 := 0
	portScore030 := 0
	for i, score := range result.Diagnostics.NodeDeepwaterScore {
		if i < len(result.Diagnostics.NodeDeepwaterTermCell) && result.Diagnostics.NodeDeepwaterTermCell[i] >= 0 {
			deepTerminals++
			deepScores = append(deepScores, score)
			if score >= 0.48 {
				deepScore048++
			}
			if score >= 0.56 {
				deepScore056++
			}
			baseScore := score
			if i < len(result.Diagnostics.NodeBaseDeepwaterScore) && result.Diagnostics.NodeBaseDeepwaterScore[i] > 0 {
				baseScore = result.Diagnostics.NodeBaseDeepwaterScore[i]
			}
			baseDeepScores = append(baseDeepScores, baseScore)
			if baseScore >= 0.24 {
				baseDeepScore024++
			}
			if baseScore >= 0.30 {
				baseDeepScore030++
			}
			if baseScore >= 0.36 {
				baseDeepScore036++
			}
			if baseScore >= 0.48 {
				baseDeepScore048++
			}
			if baseScore >= 0.56 {
				baseDeepScore056++
			}
		}
	}
	for i, score := range result.Diagnostics.NodePortScore {
		if i < len(result.Diagnostics.NodeTerminalCell) && result.Diagnostics.NodeTerminalCell[i] >= 0 {
			portTerminals++
			portScores = append(portScores, score)
			if score >= 0.30 {
				portScore030++
			}
		}
	}
	fmt.Printf(
		"      portNodeDiag: terminals=%d deepTerminals=%d portScore>=0.30=%d deepScore>=0.48=%d deepScore>=0.56=%d baseDeep>=0.24=%d baseDeep>=0.30=%d baseDeep>=0.36=%d baseDeep>=0.48=%d baseDeep>=0.56=%d portScoreP50=%.2f portScoreP90=%.2f deepScoreP50=%.2f deepScoreP90=%.2f baseDeepP50=%.2f baseDeepP90=%.2f\n",
		portTerminals,
		deepTerminals,
		portScore030,
		deepScore048,
		deepScore056,
		baseDeepScore024,
		baseDeepScore030,
		baseDeepScore036,
		baseDeepScore048,
		baseDeepScore056,
		percentileFloat64(portScores, 0.50),
		percentileFloat64(portScores, 0.90),
		percentileFloat64(deepScores, 0.50),
		percentileFloat64(deepScores, 0.90),
		percentileFloat64(baseDeepScores, 0.50),
		percentileFloat64(baseDeepScores, 0.90),
	)
}
