package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func printProtoCivilizationSummary(result *climgen.ProtoCivilizationResult, network *climgen.SettlementNetworkResult) {
	if result == nil {
		return
	}
	if len(result.Civilizations) == 0 {
		fmt.Printf("    protoCivilizations: seeds=0 outposts=%d\n", result.OutpostRegions)
		printProtoCivilizationEligibility(result)
		return
	}
	meanTerritory := 0.0
	meanCore := 0.0
	cellCount := 0
	if result.Diagnostics != nil {
		cellCount = len(result.Diagnostics.CivilizationByCell)
	}
	areaScale := reviewAreaEquivalentScale(cellCount)
	for _, civ := range result.Civilizations {
		meanTerritory += float64(civ.TerritoryCells)
		meanCore += float64(civ.CoreCells)
	}
	meanTerritory /= float64(len(result.Civilizations))
	meanCore /= float64(len(result.Civilizations))
	fmt.Printf(
		"    protoCivilizations: seeds=%d outposts=%d meanTerritory=%.1f meanTerritoryEq=%.1f meanCore=%.1f meanCoreEq=%.1f\n",
		len(result.Civilizations),
		result.OutpostRegions,
		meanTerritory,
		meanTerritory*areaScale,
		meanCore,
		meanCore*areaScale,
	)
	printProtoCivilizationEligibility(result)
	for _, civ := range topProtoCivilizations(result, 12) {
		centerCell := -1
		centerSupport := 0.0
		if network != nil && civ.CenterNode >= 0 && civ.CenterNode < len(network.Nodes) {
			center := network.Nodes[civ.CenterNode]
			centerCell = center.CellIndex
			centerSupport = center.PhysicalSupportArea
		}
		fmt.Printf(
			"      civ[%d]: region=%d centerNode=%d centerCell=%d centerSupport=%.2f anchors=%d center=%s style=%s coastal=%v river=%v territory=%d territoryEq=%.1f core=%d coreEq=%.1f meanSupport=%.2f\n",
			civ.ID,
			civ.RegionID,
			civ.CenterNode,
			centerCell,
			centerSupport,
			civ.AnchorCount,
			climgen.SettlementNodeKindName(civ.CenterKind),
			climgen.ProtoCivilizationStyleName(civ.Style),
			civ.Coastal,
			civ.River,
			civ.TerritoryCells,
			float64(civ.TerritoryCells)*areaScale,
			civ.CoreCells,
			float64(civ.CoreCells)*areaScale,
			civ.MeanSupport,
		)
	}
}

func printProtoCivilizationEligibility(result *climgen.ProtoCivilizationResult) {
	if result == nil || result.Diagnostics == nil || len(result.Diagnostics.EligibilityByReason) == 0 {
		return
	}
	tallies := result.Diagnostics.EligibilityByReason
	fmt.Printf("      eligibility: regions=%d\n", result.Diagnostics.EligibilityRegions)
	for _, reason := range climgen.ProtoCivilizationEligibilityTallyOrder(tallies) {
		tally := tallies[reason]
		verdict := "reject"
		if tally.Eligible {
			verdict = "accept"
		}
		fmt.Printf(
			"        %s[%s]: regions=%d anchor=%.2f physical=%.2f areaSupport=%.2f minStrength=%.2f centerKind=%.2f\n",
			verdict,
			reason,
			tally.Regions,
			tally.MeanAnchorStrength,
			tally.MeanPhysicalStrength,
			tally.MeanAreaSupport,
			tally.MeanMinStrength,
			tally.MeanCenterKind,
		)
	}
}

func topProtoCivilizations(result *climgen.ProtoCivilizationResult, limit int) []climgen.ProtoCivilization {
	if result == nil || len(result.Civilizations) == 0 {
		return nil
	}
	sorted := append([]climgen.ProtoCivilization(nil), result.Civilizations...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].TerritoryCells != sorted[j].TerritoryCells {
			return sorted[i].TerritoryCells > sorted[j].TerritoryCells
		}
		return sorted[i].MeanSupport > sorted[j].MeanSupport
	})
	if len(sorted) < limit {
		limit = len(sorted)
	}
	return sorted[:limit]
}

func reviewAreaEquivalentScale(cellCount int) float64 {
	return climgen.MeshAreaResolutionScale(cellCount)
}
