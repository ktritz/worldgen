package climgen

import "sort"

type MaritimeStopoverKind int

const (
	MaritimeStopoverIsland MaritimeStopoverKind = iota
	MaritimeStopoverStrait
	MaritimeStopoverRoadstead
)

func MaritimeStopoverKindName(kind MaritimeStopoverKind) string {
	names := []string{"Island Stopover", "Strait Stopover", "Roadstead"}
	if int(kind) < len(names) {
		return names[kind]
	}
	return "Unknown"
}

type MaritimeStopoverNode struct {
	ID        int
	CellIndex int
	Kind      MaritimeStopoverKind
	Score     float64
}

func BuildMaritimeStopoverNodes(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	ports *CoastalPortResult,
	elevation []float64,
	seaLevel float64,
) []MaritimeStopoverNode {
	if ports == nil || ports.Diagnostics == nil || len(cells) == 0 {
		return nil
	}
	adj := BuildFlatAdjacency(cells)
	anchored := map[int]struct{}{}
	if network != nil {
		for _, node := range network.Nodes {
			anchored[node.CellIndex] = struct{}{}
		}
	}
	candidates := make([]MaritimeStopoverNode, 0)
	for cellIdx, kind := range ports.Types {
		if cellIdx < 0 || cellIdx >= len(elevation) || elevation[cellIdx] < seaLevel {
			continue
		}
		if _, ok := anchored[cellIdx]; ok {
			continue
		}
		stopover := ports.Diagnostics.StopoverValue[cellIdx]
		suitability := ports.Diagnostics.PortSuitability[cellIdx]
		if stopover < 0.26 && suitability < 0.22 {
			continue
		}
		oceanFrac, _, _, landFrac := coastalNeighborStats(cellIdx, adj, elevation, seaLevel, nil, nil)
		score := clamp01(0.52*stopover + 0.22*suitability + 0.16*oceanFrac + 0.10*(1-landFrac))
		if score < 0.28 {
			continue
		}
		candidates = append(candidates, MaritimeStopoverNode{
			CellIndex: cellIdx,
			Kind:      classifyMaritimeStopoverKind(kind, stopover, oceanFrac, landFrac),
			Score:     score,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].CellIndex < candidates[j].CellIndex
	})
	selected := make([]MaritimeStopoverNode, 0, len(candidates))
	for _, cand := range candidates {
		if !respectsMaritimeStopoverSpacing(cand.CellIndex, selected, cells, 2) {
			continue
		}
		selected = append(selected, cand)
	}
	for i := range selected {
		selected[i].ID = i
	}
	return selected
}

func classifyMaritimeStopoverKind(portType CoastalPortType, stopover, oceanFrac, landFrac float64) MaritimeStopoverKind {
	switch {
	case portType == CoastalPortIslandStopover || stopover >= 0.50:
		return MaritimeStopoverIsland
	case oceanFrac >= 0.55 && landFrac <= 0.35:
		return MaritimeStopoverStrait
	default:
		return MaritimeStopoverRoadstead
	}
}

func respectsMaritimeStopoverSpacing(cellIdx int, existing []MaritimeStopoverNode, cells []VoronoiCell, minHops int) bool {
	for _, node := range existing {
		if graphHopDistance(cells, cellIdx, node.CellIndex, minHops) < minHops {
			return false
		}
	}
	return true
}

func graphHopDistance(cells []VoronoiCell, start, goal, maxHops int) int {
	if start == goal {
		return 0
	}
	type state struct {
		cell int
		hops int
	}
	seen := map[int]struct{}{start: {}}
	queue := []state{{cell: start, hops: 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.hops >= maxHops {
			continue
		}
		for _, raw := range cells[cur.cell].NeighborSiteIndices {
			neighbor := int(raw)
			if neighbor == goal {
				return cur.hops + 1
			}
			if _, ok := seen[neighbor]; ok {
				continue
			}
			seen[neighbor] = struct{}{}
			queue = append(queue, state{cell: neighbor, hops: cur.hops + 1})
		}
	}
	return maxHops + 1
}
