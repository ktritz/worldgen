package climgen

import "math"

func addWaystationBridges(
	sites []Vector3D,
	cells []VoronoiCell,
	nodes []SettlementNode,
	movementCost []float64,
	settlements *SettlementResult,
	population *PopulationResult,
	resources *ResourceResult,
	elevation []float64,
	seaLevel float64,
	settings SettlementNetworkSettings,
) []SettlementNode {
	if len(nodes) == 0 {
		return nodes
	}
	nodeByCell := make(map[int]int, len(nodes))
	for i, node := range nodes {
		nodeByCell[node.CellIndex] = i
	}
	degree := make([]int, len(nodes))
	initialLinks := buildSettlementLinks(cells, nodes, movementCost, settlements, resources, settings)
	for _, link := range initialLinks {
		degree[link.From]++
		degree[link.To]++
	}

	for i, node := range nodes {
		if degree[i] > 0 {
			continue
		}
		if node.Kind < SettlementNodeTown && !(node.Kind == SettlementNodeVillage && node.CarryingCapacity >= settings.TownThreshold-0.02) {
			continue
		}
		maxTravel := nodeTravelLimit(node.Kind, settings) * 1.8
		dist, prev := shortestPathsFromNode(node.CellIndex, cells, movementCost, maxTravel)
		targetIdx := nearestBridgeTarget(i, node, nodes, dist)
		if targetIdx < 0 {
			continue
		}
		path := reconstructSettlementPath(node.CellIndex, nodes[targetIdx].CellIndex, prev)
		if len(path) < 4 {
			continue
		}
		bridgeCells := selectWaystationCells(path, sites, nodeByCell, settlements, population, elevation, seaLevel, settings)
		for _, cellIdx := range bridgeCells {
			if _, exists := nodeByCell[cellIdx]; exists {
				continue
			}
			carrying := population.Diagnostics.CarryingCapacity[cellIdx]
			urban := population.Diagnostics.UrbanPotential[cellIdx]
			kind, ok := classifySettlementNodeCandidate(carrying, urban, settings)
			if !ok || kind > SettlementNodeVillage {
				kind = SettlementNodeVillage
			}
			if kind < SettlementNodeHamlet {
				kind = SettlementNodeHamlet
			}
			nodeByCell[cellIdx] = len(nodes)
			nodes = append(nodes, SettlementNode{
				ID:               len(nodes),
				CellIndex:        cellIdx,
				Kind:             kind,
				Score:            clamp01(0.58*carrying + 0.42*urban),
				CarryingCapacity: carrying,
				UrbanPotential:   urban,
				Coastal:          settlements.Diagnostics.CoastalBonus[cellIdx] >= 0.16,
				River:            settlements.Diagnostics.RiverBonus[cellIdx] >= 0.24,
			})
		}
	}
	return nodes
}

func nearestBridgeTarget(sourceIdx int, source SettlementNode, nodes []SettlementNode, dist []float64) int {
	bestIdx := -1
	bestDist := math.Inf(1)
	for j, node := range nodes {
		if j == sourceIdx || node.Kind < SettlementNodeVillage {
			continue
		}
		d := dist[node.CellIndex]
		if d < bestDist {
			bestDist = d
			bestIdx = j
		}
	}
	return bestIdx
}

func selectWaystationCells(
	path []int,
	sites []Vector3D,
	nodeByCell map[int]int,
	settlements *SettlementResult,
	population *PopulationResult,
	elevation []float64,
	seaLevel float64,
	settings SettlementNetworkSettings,
) []int {
	if len(path) < 4 {
		return nil
	}
	targetFractions := []float64{0.5}
	if len(path) >= 10 {
		targetFractions = []float64{0.33, 0.66}
	}
	bridgeCells := make([]int, 0, len(targetFractions))
	for _, frac := range targetFractions {
		center := int(frac * float64(len(path)-1))
		bestCell := -1
		bestScore := -1.0
		for offset := -2; offset <= 2; offset++ {
			idx := center + offset
			if idx <= 0 || idx >= len(path)-1 {
				continue
			}
			cellIdx := path[idx]
			if elevation[cellIdx] < seaLevel {
				continue
			}
			if _, exists := nodeByCell[cellIdx]; exists {
				continue
			}
			if settlements.Classes[cellIdx] < SettlementMarginal {
				continue
			}
			carrying := population.Diagnostics.CarryingCapacity[cellIdx]
			if carrying < settings.HamletThreshold {
				continue
			}
			score := 0.65*carrying + 0.35*settlements.Diagnostics.AccessScore[cellIdx]
			if score > bestScore && respectsWaystationSpacing(cellIdx, appendKnownCells(nodeByCell, bridgeCells), sites, settings.HamletSpacingDeg) {
				bestScore = score
				bestCell = cellIdx
			}
		}
		if bestCell >= 0 {
			bridgeCells = append(bridgeCells, bestCell)
		}
	}
	return bridgeCells
}

func appendKnownCells(nodeByCell map[int]int, extra []int) []int {
	cells := make([]int, 0, len(nodeByCell)+len(extra))
	for cellIdx := range nodeByCell {
		cells = append(cells, cellIdx)
	}
	cells = append(cells, extra...)
	return cells
}

func respectsWaystationSpacing(cellIdx int, existingCells []int, sites []Vector3D, minDeg float64) bool {
	for _, other := range existingCells {
		if greatCircleDistanceDeg(sites[cellIdx], sites[other]) < minDeg {
			return false
		}
	}
	return true
}
