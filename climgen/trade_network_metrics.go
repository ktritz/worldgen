package climgen

import "sort"

func dedupeTradeCorridors(corridors []TradeCorridor) []TradeCorridor {
	type key struct {
		role      TradeCorridorRole
		fromNode  int
		toNode    int
		fromLocal int
		toLocal   int
	}
	best := make(map[key]TradeCorridor)
	for _, corridor := range corridors {
		k := key{
			role:      corridor.Role,
			fromNode:  corridor.FromNode,
			toNode:    corridor.ToNode,
			fromLocal: corridor.FromLocalNode,
			toLocal:   corridor.ToLocalNode,
		}
		if corridor.Role != TradeCorridorRoleFeeder {
			if k.fromNode > k.toNode {
				k.fromNode, k.toNode = k.toNode, k.fromNode
			}
		} else if k.fromLocal == -1 && k.toLocal == -1 && k.fromNode > k.toNode {
			k.fromNode, k.toNode = k.toNode, k.fromNode
		}
		current, ok := best[k]
		if !ok || corridor.Flow > current.Flow {
			best[k] = corridor
		}
	}
	out := make([]TradeCorridor, 0, len(best))
	for _, corridor := range best {
		out = append(out, corridor)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Flow > out[j].Flow })
	for i := range out {
		out[i].ID = i
	}
	return out
}

func buildTradeHandoffs(corridors []TradeCorridor, majorHubs []int) []TradeHandoff {
	if len(corridors) == 0 {
		return nil
	}
	hubSet := make(map[int]struct{}, len(majorHubs))
	for _, nodeIdx := range majorHubs {
		hubSet[nodeIdx] = struct{}{}
	}
	type agg struct {
		feeder int
		trunk  int
		modes  map[string]int
	}
	byNode := map[int]*agg{}
	for _, corridor := range corridors {
		if corridor.Role == TradeCorridorRoleFeeder && corridor.HandoffNode >= 0 {
			entry := byNode[corridor.HandoffNode]
			if entry == nil {
				entry = &agg{modes: map[string]int{}}
				byNode[corridor.HandoffNode] = entry
			}
			entry.feeder++
			entry.modes[corridor.Mode]++
		}
		if corridor.Role != TradeCorridorRoleFeeder {
			nodes := []int{corridor.FromNode, corridor.ToNode}
			for _, nodeIdx := range nodes {
				entry := byNode[nodeIdx]
				if entry == nil {
					entry = &agg{modes: map[string]int{}}
					byNode[nodeIdx] = entry
				}
				entry.trunk++
				entry.modes[corridor.Mode]++
			}
		}
	}
	out := make([]TradeHandoff, 0, len(byNode))
	for nodeIdx, entry := range byNode {
		if entry.feeder == 0 || entry.trunk == 0 {
			continue
		}
		dominantMode := ""
		dominantCount := -1
		for mode, count := range entry.modes {
			if count > dominantCount {
				dominantMode = mode
				dominantCount = count
			}
		}
		_, isHub := hubSet[nodeIdx]
		out = append(out, TradeHandoff{
			Node:            nodeIdx,
			FeederCorridors: entry.feeder,
			TrunkCorridors:  entry.trunk,
			DominantMode:    dominantMode,
			Hub:             isHub,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FeederCorridors != out[j].FeederCorridors {
			return out[i].FeederCorridors > out[j].FeederCorridors
		}
		return out[i].TrunkCorridors > out[j].TrunkCorridors
	})
	return out
}

func classifyTradeCorridors(corridors []TradeCorridor, settings TradeNetworkSettings) {
	for i := range corridors {
		switch {
		case corridors[i].Flow >= settings.PrimaryFlow:
			corridors[i].Tier = TradeCorridorPrimary
		case corridors[i].Flow >= settings.RegionalFlow || corridors[i].InterCivilization:
			corridors[i].Tier = TradeCorridorRegional
		default:
			corridors[i].Tier = TradeCorridorLocal
		}
	}
}

func applyTradeDiagnostics(corridors []TradeCorridor, diagnostics *TradeNetworkDiagnostics) {
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
				diagnostics.RouteRiskIntensity[cellIdx] += corridor.Flow * corridor.MeanRisk
			}
		}
	}
}

func identifyMajorTradeHubs(
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	diagnostics *TradeNetworkDiagnostics,
	settings TradeNetworkSettings,
) []int {
	maxCentrality := 0.0
	for _, value := range diagnostics.NodeCentrality {
		if value > maxCentrality {
			maxCentrality = value
		}
	}
	hubs := make([]int, 0)
	for i, node := range network.Nodes {
		centerBonus := 0.0
		for _, civ := range proto.Civilizations {
			if civ.CenterNode == i {
				centerBonus = 0.12
				break
			}
		}
		centralityNorm := 0.0
		if maxCentrality > 0 {
			centralityNorm = diagnostics.NodeCentrality[i] / maxCentrality
		}
		score := node.Score + 0.18*(float64(node.Kind)/3.0) + 0.24*centralityNorm + centerBonus
		if node.Coastal {
			score += 0.05
		}
		if node.River {
			score += 0.04
		}
		diagnostics.HubScore[i] = score
		if node.Kind >= SettlementNodeTown && score >= settings.HubThreshold {
			hubs = append(hubs, i)
		}
	}
	sort.Slice(hubs, func(i, j int) bool {
		return diagnostics.HubScore[hubs[i]] > diagnostics.HubScore[hubs[j]]
	})
	return hubs
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
