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
	primary := 0
	secondary := 0
	cellCount := 0
	if result.Diagnostics != nil {
		cellCount = len(result.Diagnostics.PolityByCell)
	}
	areaScale := reviewAreaEquivalentScale(cellCount)
	for _, sphere := range result.Spheres {
		meanTerritory += float64(sphere.TerritoryCells)
		meanCore += float64(sphere.CoreCells)
		if sphere.Secondary {
			secondary++
		} else {
			primary++
		}
	}
	meanTerritory /= float64(len(result.Spheres))
	meanCore /= float64(len(result.Spheres))
	fmt.Printf(
		"    politySpheres: spheres=%d primary=%d secondary=%d relations=%d mergedMinor=%d meanTerritory=%.1f meanTerritoryEq=%.1f meanCore=%.1f meanCoreEq=%.1f\n",
		len(result.Spheres),
		primary,
		secondary,
		len(result.Relations),
		result.MergedMinor,
		meanTerritory,
		meanTerritory*areaScale,
		meanCore,
		meanCore*areaScale,
	)
	for _, sphere := range topPolitySpheres(result, 12) {
		capital := network.Nodes[sphere.CapitalNode]
		regionID := -1
		if network.Diagnostics != nil && sphere.CapitalNode >= 0 && sphere.CapitalNode < len(network.Diagnostics.RegionByNode) {
			regionID = network.Diagnostics.RegionByNode[sphere.CapitalNode]
		}
		fmt.Printf(
			"      polity[%d]: proto=%d capitalNode=%d region=%d capitalCell=%d capitalSupport=%.2f capital=%s secondary=%v style=%s territory=%d territoryEq=%.1f core=%d coreEq=%.1f meanSupport=%.2f influence=%.2f coastal=%v river=%v\n",
			sphere.ID,
			sphere.ProtoCivilizationID,
			sphere.CapitalNode,
			regionID,
			capital.CellIndex,
			capital.PhysicalSupportArea,
			climgen.SettlementNodeKindName(capital.Kind),
			sphere.Secondary,
			climgen.ProtoCivilizationStyleName(sphere.Style),
			sphere.TerritoryCells,
			float64(sphere.TerritoryCells)*areaScale,
			sphere.CoreCells,
			float64(sphere.CoreCells)*areaScale,
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
