package climgen

import "sort"

func dedupeRiverTradeCorridors(corridors []RiverTradeCorridor) []RiverTradeCorridor {
	type key struct{ a, b int }
	best := make(map[key]RiverTradeCorridor)
	for _, corridor := range corridors {
		k := key{a: corridor.FromNode, b: corridor.ToNode}
		if k.a > k.b {
			k.a, k.b = k.b, k.a
		}
		current, ok := best[k]
		if !ok || corridor.Flow > current.Flow {
			best[k] = corridor
		}
	}
	out := make([]RiverTradeCorridor, 0, len(best))
	for _, corridor := range best {
		out = append(out, corridor)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Flow > out[j].Flow })
	for i := range out {
		out[i].ID = i
	}
	return out
}

func classifyRiverTradeCorridors(corridors []RiverTradeCorridor, settings RiverTradeSettings) {
	for i := range corridors {
		switch {
		case corridors[i].Flow >= settings.PrimaryFlow:
			corridors[i].Tier = RiverTradeCorridorPrimary
		case corridors[i].Flow >= settings.RegionalFlow || corridors[i].InterCivilization:
			corridors[i].Tier = RiverTradeCorridorRegional
		default:
			corridors[i].Tier = RiverTradeCorridorLocal
		}
	}
}

func applyRiverTradeDiagnostics(corridors []RiverTradeCorridor, diagnostics *RiverTradeDiagnostics) {
	if diagnostics == nil {
		return
	}
	for _, corridor := range corridors {
		for _, nodeIdx := range corridor.NodePath {
			if nodeIdx >= 0 && nodeIdx < len(diagnostics.NodeCentrality) {
				diagnostics.NodeCentrality[nodeIdx] += corridor.Flow
			}
		}
		for _, cellIdx := range corridor.CellPath {
			if cellIdx >= 0 && cellIdx < len(diagnostics.RouteIntensity) {
				diagnostics.RouteIntensity[cellIdx] += corridor.Flow
			}
		}
	}
}

func identifyMajorRiverPorts(
	network *SettlementNetworkResult,
	riverNodes map[int]struct{},
	diagnostics *RiverTradeDiagnostics,
	settings RiverTradeSettings,
) []int {
	maxCentrality := 0.0
	for _, value := range diagnostics.NodeCentrality {
		if value > maxCentrality {
			maxCentrality = value
		}
	}
	out := make([]int, 0)
	for nodeIdx := range riverNodes {
		node := network.Nodes[nodeIdx]
		centralityNorm := 0.0
		if maxCentrality > 0 {
			centralityNorm = diagnostics.NodeCentrality[nodeIdx] / maxCentrality
		}
		if !eligibleMajorRiverPort(node, centralityNorm) {
			continue
		}
		score := node.Score + 0.22*(float64(node.Kind)/3.0) + 0.34*centralityNorm
		if node.River {
			score += 0.06
		}
		if node.Coastal {
			score += 0.04
		}
		if score >= settings.PortThreshold {
			out = append(out, nodeIdx)
		}
	}
	sort.Slice(out, func(i, j int) bool { return diagnostics.NodeCentrality[out[i]] > diagnostics.NodeCentrality[out[j]] })
	return out
}

func eligibleMajorRiverPort(node SettlementNode, centralityNorm float64) bool {
	switch node.Kind {
	case SettlementNodeCity, SettlementNodeTown:
		return centralityNorm >= 0.10
	case SettlementNodeVillage:
		return centralityNorm >= 0.18
	default:
		return centralityNorm >= 0.42
	}
}
