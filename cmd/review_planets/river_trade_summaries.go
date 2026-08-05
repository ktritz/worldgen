package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

// riverRouteNavigability exposes the per-cell navigability field used as the
// river-extent denominator, or nil when river routes were not built.
func riverRouteNavigability(result *climgen.RiverRouteResult) []float64 {
	if result == nil || result.Diagnostics == nil {
		return nil
	}
	return result.Diagnostics.Navigability
}

// printMeshDensityDenominators reports the physical extents that the structural
// densities divide by, so a count change can be attributed to the denominator
// growing (the mesh resolves more river/coast) rather than to the density.
func printMeshDensityDenominators(denominators climgen.MeshDensityDenominators) {
	fmt.Printf(
		"    meshDensity: cells=%d landCells=%d oceanCells=%d coastalCells=%d meanSpacing=%.6f meanCellArea=%.8f navLength=%.6f coastLength=%.6f oceanArea=%.6f\n",
		denominators.CellCount,
		denominators.LandCells,
		denominators.OceanCells,
		denominators.CoastalCells,
		denominators.MeanCellSpacing,
		denominators.MeanCellArea,
		denominators.NavLength,
		denominators.CoastLength,
		denominators.OceanArea,
	)
}

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

func printRiverTradeSummary(
	result *climgen.RiverTradeResult,
	network *climgen.SettlementNetworkResult,
	denominators climgen.MeshDensityDenominators,
) {
	if result == nil {
		return
	}
	if len(result.Corridors) == 0 {
		terminals := 0
		if result.Diagnostics != nil {
			for _, cellIdx := range result.Diagnostics.NodeTerminalCell {
				if cellIdx >= 0 {
					terminals++
				}
			}
		}
		fmt.Printf(
			"    riverTrade: corridors=0 ports=0 terminals=%d navLength=%.6f terminalsPerNavLength=%.6f riverCorridorsPerNavLength=%.6f\n",
			terminals,
			denominators.NavLength,
			climgen.PerExtent(terminals, denominators.NavLength),
			0.0,
		)
		return
	}
	tierCounts := make(map[climgen.RiverTradeCorridorTier]int)
	totalFlow := 0.0
	totalNav := 0.0
	totalTransfer := 0.0
	inter := 0
	terminals := 0
	if result.Diagnostics != nil {
		for _, cellIdx := range result.Diagnostics.NodeTerminalCell {
			if cellIdx >= 0 {
				terminals++
			}
		}
	}
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
		"    riverTrade: corridors=%d ports=%d terminals=%d inter=%d meanFlow=%.2f meanNav=%.2f meanTransfer=%.2f navLength=%.6f terminalsPerNavLength=%.6f riverCorridorsPerNavLength=%.6f\n",
		len(result.Corridors),
		len(result.MajorPorts),
		terminals,
		inter,
		totalFlow/float64(len(result.Corridors)),
		totalNav/float64(len(result.Corridors)),
		totalTransfer/float64(len(result.Corridors)),
		denominators.NavLength,
		climgen.PerExtent(terminals, denominators.NavLength),
		climgen.PerExtent(len(result.Corridors), denominators.NavLength),
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
