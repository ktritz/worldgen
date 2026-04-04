package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func printProtoCivilizationSummary(result *climgen.ProtoCivilizationResult) {
	if result == nil {
		return
	}
	if len(result.Civilizations) == 0 {
		fmt.Printf("    protoCivilizations: seeds=0 outposts=%d\n", result.OutpostRegions)
		return
	}
	meanTerritory := 0.0
	meanCore := 0.0
	for _, civ := range result.Civilizations {
		meanTerritory += float64(civ.TerritoryCells)
		meanCore += float64(civ.CoreCells)
	}
	meanTerritory /= float64(len(result.Civilizations))
	meanCore /= float64(len(result.Civilizations))
	fmt.Printf(
		"    protoCivilizations: seeds=%d outposts=%d meanTerritory=%.1f meanCore=%.1f\n",
		len(result.Civilizations),
		result.OutpostRegions,
		meanTerritory,
		meanCore,
	)
	for _, civ := range topProtoCivilizations(result, 5) {
		fmt.Printf(
			"      civ[%d]: anchors=%d center=%s style=%s coastal=%v river=%v territory=%d core=%d meanSupport=%.2f\n",
			civ.ID,
			civ.AnchorCount,
			climgen.SettlementNodeKindName(civ.CenterKind),
			climgen.ProtoCivilizationStyleName(civ.Style),
			civ.Coastal,
			civ.River,
			civ.TerritoryCells,
			civ.CoreCells,
			civ.MeanSupport,
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
