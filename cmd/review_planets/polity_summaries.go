package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func printPolitySphereSummary(result *climgen.PolitySphereResult, network *climgen.SettlementNetworkResult) {
	if result == nil {
		return
	}
	if len(result.Spheres) == 0 {
		fmt.Printf("    politySpheres: spheres=0 mergedMinor=%d\n", result.MergedMinor)
		return
	}
	meanTerritory := 0.0
	meanCore := 0.0
	for _, sphere := range result.Spheres {
		meanTerritory += float64(sphere.TerritoryCells)
		meanCore += float64(sphere.CoreCells)
	}
	meanTerritory /= float64(len(result.Spheres))
	meanCore /= float64(len(result.Spheres))
	fmt.Printf(
		"    politySpheres: spheres=%d relations=%d mergedMinor=%d meanTerritory=%.1f meanCore=%.1f\n",
		len(result.Spheres),
		len(result.Relations),
		result.MergedMinor,
		meanTerritory,
		meanCore,
	)
	for _, sphere := range topPolitySpheres(result, 5) {
		capital := network.Nodes[sphere.CapitalNode]
		fmt.Printf(
			"      polity[%d]: capital=%s secondary=%v style=%s territory=%d core=%d meanSupport=%.2f influence=%.2f coastal=%v river=%v\n",
			sphere.ID,
			climgen.SettlementNodeKindName(capital.Kind),
			sphere.Secondary,
			climgen.ProtoCivilizationStyleName(sphere.Style),
			sphere.TerritoryCells,
			sphere.CoreCells,
			sphere.MeanSupport,
			sphere.MeanInfluence,
			sphere.Coastal,
			sphere.River,
		)
	}
	for _, relation := range topPolityRelations(result, 4) {
		fmt.Printf(
			"      relation[%s]: overlord=%d subject=%d strength=%.2f\n",
			climgen.PolityRelationTypeName(relation.Kind),
			relation.Overlord,
			relation.Subject,
			relation.Strength,
		)
	}
}

func topPolitySpheres(result *climgen.PolitySphereResult, limit int) []climgen.PolitySphere {
	if result == nil || len(result.Spheres) == 0 {
		return nil
	}
	sorted := append([]climgen.PolitySphere(nil), result.Spheres...)
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

func topPolityRelations(result *climgen.PolitySphereResult, limit int) []climgen.PolitySphereRelation {
	if result == nil || len(result.Relations) == 0 {
		return nil
	}
	sorted := append([]climgen.PolitySphereRelation(nil), result.Relations...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Strength > sorted[j].Strength
	})
	if len(sorted) < limit {
		limit = len(sorted)
	}
	return sorted[:limit]
}
