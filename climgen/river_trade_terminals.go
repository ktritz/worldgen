package climgen

import "math"

type riverTradeTerminal struct {
	node         int
	cell         int
	distance     int
	score        float64
	navigability float64
	transfer     float64
}

func buildRiverTradeTerminals(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	riverRoutes *RiverRouteResult,
) map[int]riverTradeTerminal {
	out := map[int]riverTradeTerminal{}
	if network == nil || riverRoutes == nil || riverRoutes.Diagnostics == nil {
		return out
	}
	maxSteps := riverTradeTerminalCatchmentSteps(len(cells))
	for _, node := range network.Nodes {
		terminal, ok := bestRiverTerminalForNode(cells, node, riverRoutes, maxSteps)
		if ok {
			out[node.ID] = terminal
		}
	}
	return out
}

func riverTradeTerminalNodeSet(terminals map[int]riverTradeTerminal) map[int]struct{} {
	out := make(map[int]struct{}, len(terminals))
	for nodeIdx := range terminals {
		out[nodeIdx] = struct{}{}
	}
	return out
}

func riverTradeTerminalCatchmentSteps(cellCount int) int {
	if cellCount <= 0 {
		return 1
	}
	steps := int(math.Ceil(math.Sqrt(float64(cellCount) / baselinePathCostCells)))
	if steps < 1 {
		return 1
	}
	if steps > 4 {
		return 4
	}
	return steps
}

func bestRiverTerminalForNode(
	cells []VoronoiCell,
	node SettlementNode,
	riverRoutes *RiverRouteResult,
	maxSteps int,
) (riverTradeTerminal, bool) {
	if node.CellIndex < 0 || node.CellIndex >= len(cells) {
		return riverTradeTerminal{}, false
	}
	best := riverTradeTerminal{node: node.ID, cell: -1, score: -1}
	for _, candidate := range riverTerminalCatchmentCells(cells, node.CellIndex, maxSteps) {
		score, ok := riverTerminalCellScore(candidate.cell, candidate.distance, node, riverRoutes)
		if !ok {
			continue
		}
		if score > best.score {
			best = riverTradeTerminal{
				node:         node.ID,
				cell:         candidate.cell,
				distance:     candidate.distance,
				score:        score,
				navigability: riverRoutes.Diagnostics.Navigability[candidate.cell],
				transfer:     riverRoutes.Diagnostics.TransferSupport[candidate.cell],
			}
		}
	}
	return best, best.cell >= 0
}

type riverTerminalCatchmentCell struct {
	cell     int
	distance int
}

func riverTerminalCatchmentCells(cells []VoronoiCell, start, maxSteps int) []riverTerminalCatchmentCell {
	if start < 0 || start >= len(cells) {
		return nil
	}
	dist := make([]int, len(cells))
	for i := range dist {
		dist[i] = -1
	}
	dist[start] = 0
	queue := []int{start}
	out := []riverTerminalCatchmentCell{{cell: start, distance: 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if dist[cur] >= maxSteps {
			continue
		}
		for _, rawNeighbor := range cells[cur].NeighborSiteIndices {
			neighbor := int(rawNeighbor)
			if neighbor < 0 || neighbor >= len(cells) || dist[neighbor] >= 0 {
				continue
			}
			dist[neighbor] = dist[cur] + 1
			queue = append(queue, neighbor)
			out = append(out, riverTerminalCatchmentCell{cell: neighbor, distance: dist[neighbor]})
		}
	}
	return out
}

func riverTerminalCellScore(
	cellIdx int,
	distance int,
	node SettlementNode,
	riverRoutes *RiverRouteResult,
) (float64, bool) {
	diag := riverRoutes.Diagnostics
	if cellIdx < 0 ||
		cellIdx >= len(diag.Navigability) ||
		cellIdx >= len(diag.MainChannel) ||
		cellIdx >= len(diag.TransferSupport) ||
		cellIdx >= len(diag.PortageSuitability) {
		return 0, false
	}
	nav := diag.Navigability[cellIdx]
	transfer := diag.TransferSupport[cellIdx]
	portage := diag.PortageSuitability[cellIdx]
	mainChannel := diag.MainChannel[cellIdx]
	minNav := riverRoutes.Mode.MinNavigability
	minTransfer := riverRoutes.Mode.TransferSupportFloor
	if nav < minNav*0.42 && !(transfer >= minTransfer*0.80 && portage >= 0.22) {
		return 0, false
	}
	score := 0.62*nav + 0.16*mainChannel + 0.14*transfer + 0.08*portage
	score -= 0.035 * float64(distance)
	if node.River {
		score += 0.06
	}
	if node.Kind >= SettlementNodeTown {
		score += 0.04
	}
	if score < minNav*0.48 {
		return 0, false
	}
	return score, true
}
