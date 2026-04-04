package main

import (
	"fmt"
	"math"
	"sort"

	"worldgen/climgen"
)

func printSettlementNetworkSummary(result *climgen.SettlementNetworkResult) {
	if result == nil {
		return
	}
	nodeCount := len(result.Nodes)
	linkCount := len(result.Links)
	if nodeCount == 0 {
		fmt.Println("    settlementNetwork: nodes=0 links=0")
		return
	}
	classCounts := make(map[climgen.SettlementNodeKind]int)
	for _, node := range result.Nodes {
		classCounts[node.Kind]++
	}
	linkCosts := make([]float64, 0, len(result.Links))
	degree := make([]int, len(result.Nodes))
	for _, link := range result.Links {
		linkCosts = append(linkCosts, link.TravelCost)
		if link.From >= 0 && link.From < len(degree) {
			degree[link.From]++
		}
		if link.To >= 0 && link.To < len(degree) {
			degree[link.To]++
		}
	}
	nearestCosts := nearestNeighborCosts(result)
	isolated := 0
	for _, d := range degree {
		if d == 0 {
			isolated++
		}
	}
	fmt.Printf(
		"    settlementNetwork: nodes=%d links=%d meanLinkCost=%.2f medianLinkCost=%.2f nearestMean=%.2f nearestMedian=%.2f isolated=%.1f%%\n",
		nodeCount,
		linkCount,
		meanFloat(linkCosts),
		medianFloat(linkCosts),
		meanFloat(nearestCosts),
		medianFloat(nearestCosts),
		100*float64(isolated)/float64(nodeCount),
	)

	type nodeClassCount struct {
		kind  climgen.SettlementNodeKind
		count int
	}
	sorted := make([]nodeClassCount, 0, len(classCounts))
	for kind, count := range classCounts {
		sorted = append(sorted, nodeClassCount{kind: kind, count: count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].kind > sorted[j].kind })
	for _, entry := range sorted {
		fmt.Printf("      node[%s]=%d\n", climgen.SettlementNodeKindName(entry.kind), entry.count)
	}
}

func meanFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func medianFloat(values []float64) float64 {
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

func nearestNeighborCosts(result *climgen.SettlementNetworkResult) []float64 {
	if result == nil || len(result.Nodes) == 0 {
		return nil
	}
	nearest := make([]float64, 0, len(result.Nodes))
	for i := range result.Nodes {
		best := math.Inf(1)
		for _, link := range result.Links {
			switch {
			case link.From == i && link.TravelCost < best:
				best = link.TravelCost
			case link.To == i && link.TravelCost < best:
				best = link.TravelCost
			}
		}
		if !math.IsInf(best, 1) {
			nearest = append(nearest, best)
		}
	}
	return nearest
}
