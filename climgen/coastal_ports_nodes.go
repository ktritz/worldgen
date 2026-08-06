package climgen

import (
	"math"
	"sort"
)

func identifyMajorCoastalPorts(
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	riverTrade *RiverTradeResult,
	diag *CoastalPortDiagnostics,
	settings MaritimePortSettings,
) []int {
	if network == nil || diag == nil {
		return nil
	}
	maxTrade := 0.0
	if trade != nil && trade.Diagnostics != nil {
		for _, v := range trade.Diagnostics.NodeCentrality {
			if v > maxTrade {
				maxTrade = v
			}
		}
	}
	maxRiver := 0.0
	if riverTrade != nil && riverTrade.Diagnostics != nil {
		for _, v := range riverTrade.Diagnostics.NodeCentrality {
			if v > maxRiver {
				maxRiver = v
			}
		}
	}
	hubSet := map[int]struct{}{}
	if trade != nil {
		for _, idx := range trade.MajorHubs {
			hubSet[idx] = struct{}{}
		}
	}
	riverPortSet := map[int]struct{}{}
	if riverTrade != nil {
		for _, idx := range riverTrade.MajorPorts {
			riverPortSet[idx] = struct{}{}
		}
	}

	out := make([]int, 0)
	for i, node := range network.Nodes {
		if node.CellIndex < 0 || node.CellIndex >= len(diag.PortSuitability) {
			continue
		}
		if !hasCoastalTerminal(diag, i) {
			continue
		}
		portSuit := diag.PortSuitability[node.CellIndex]
		rawNodeSuit := 0.0
		if i < len(diag.NodeBasePortScore) && diag.NodeBasePortScore[i] > 0 {
			rawNodeSuit = diag.NodeBasePortScore[i]
		} else if i < len(diag.NodePortScore) {
			rawNodeSuit = diag.NodePortScore[i]
		}
		if math.Max(portSuit, rawNodeSuit) < settings.PortSuitabilityFloor && !(node.Coastal || node.River) {
			continue
		}
		effectiveSuit := math.Max(rawNodeSuit, math.Max(portSuit, 0.22*bool01(node.Coastal)+0.12*bool01(node.River)))
		tradeNorm := 0.0
		if trade != nil && trade.Diagnostics != nil && i < len(trade.Diagnostics.NodeCentrality) && maxTrade > 0 {
			tradeNorm = trade.Diagnostics.NodeCentrality[i] / maxTrade
		}
		riverNorm := 0.0
		if riverTrade != nil && riverTrade.Diagnostics != nil && i < len(riverTrade.Diagnostics.NodeCentrality) && maxRiver > 0 {
			riverNorm = riverTrade.Diagnostics.NodeCentrality[i] / maxRiver
		}
		score := settings.PortSuitabilityWeight*effectiveSuit +
			settings.NodeScoreWeight*node.Score +
			settings.TradeCentralityWeight*tradeNorm +
			settings.RiverCentralityWeight*riverNorm
		switch settlementNodePortAnchorTier(node) {
		case SettlementNodeCity, SettlementNodeTown:
			score += settings.RegionalAnchorBonus
		case SettlementNodeVillage:
			score += settings.DistrictAnchorBonus
		default:
			score += settings.LocalAnchorBonus
		}
		if _, ok := hubSet[i]; ok {
			score += settings.MajorHubBonus
		}
		if node.River {
			score += 0.5 * settings.RiverHandoffBonus
		}
		if _, ok := riverPortSet[i]; ok {
			score += settings.RiverHandoffBonus
		}
		if node.Coastal {
			score += 0.12
		}
		diag.NodePortScore[i] = score
		if !eligibleMajorCoastalPort(node, tradeNorm, riverNorm, settings) {
			continue
		}
		if score >= settings.MajorPortThreshold {
			out = append(out, i)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return diag.NodePortScore[out[i]] > diag.NodePortScore[out[j]]
	})
	return out
}

func identifyMajorDeepwaterPorts(
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	riverTrade *RiverTradeResult,
	diag *CoastalPortDiagnostics,
	settings MaritimePortSettings,
) []int {
	if network == nil || diag == nil {
		return nil
	}
	maxTrade, maxRiver := coastalPortCentralityMaxima(trade, riverTrade)
	hubSet := map[int]struct{}{}
	if trade != nil {
		for _, idx := range trade.MajorHubs {
			hubSet[idx] = struct{}{}
		}
	}
	riverPortSet := map[int]struct{}{}
	if riverTrade != nil {
		for _, idx := range riverTrade.MajorPorts {
			riverPortSet[idx] = struct{}{}
		}
	}

	out := make([]int, 0)
	for i, node := range network.Nodes {
		if node.CellIndex < 0 || node.CellIndex >= len(diag.DeepwaterSuitability) {
			continue
		}
		if !hasDeepwaterTerminal(diag, i) {
			continue
		}
		rawNodeSuit := 0.0
		if i < len(diag.NodeBaseDeepwaterScore) && diag.NodeBaseDeepwaterScore[i] > 0 {
			rawNodeSuit = diag.NodeBaseDeepwaterScore[i]
		} else if i < len(diag.NodeDeepwaterScore) {
			rawNodeSuit = diag.NodeDeepwaterScore[i]
		}
		effectiveSuit := math.Max(rawNodeSuit, diag.DeepwaterSuitability[node.CellIndex])
		if effectiveSuit < settings.PortSuitabilityFloor && !(node.Coastal || node.River) {
			continue
		}
		tradeNorm := 0.0
		if trade != nil && trade.Diagnostics != nil && i < len(trade.Diagnostics.NodeCentrality) && maxTrade > 0 {
			tradeNorm = trade.Diagnostics.NodeCentrality[i] / maxTrade
		}
		riverNorm := 0.0
		if riverTrade != nil && riverTrade.Diagnostics != nil && i < len(riverTrade.Diagnostics.NodeCentrality) && maxRiver > 0 {
			riverNorm = riverTrade.Diagnostics.NodeCentrality[i] / maxRiver
		}
		score := settings.DeepwaterNodeWeight*effectiveSuit +
			settings.NodeScoreWeight*node.Score +
			settings.TradeCentralityWeight*tradeNorm +
			settings.RiverCentralityWeight*riverNorm
		switch settlementNodePortAnchorTier(node) {
		case SettlementNodeCity, SettlementNodeTown:
			score += settings.RegionalAnchorBonus
		case SettlementNodeVillage:
			score += settings.DistrictAnchorBonus
		default:
			score += settings.LocalAnchorBonus
		}
		if _, ok := hubSet[i]; ok {
			score += settings.MajorHubBonus
		}
		if _, ok := riverPortSet[i]; ok {
			score += 0.75 * settings.RiverHandoffBonus
		}
		if node.Coastal {
			score += 0.14
		}
		diag.NodeDeepwaterScore[i] = score
		if !eligibleMajorCoastalPort(node, tradeNorm, riverNorm, settings) {
			continue
		}
		if score >= settings.MajorDeepwaterPortThreshold {
			out = append(out, i)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return diag.NodeDeepwaterScore[out[i]] > diag.NodeDeepwaterScore[out[j]]
	})
	return out
}

func hasCoastalTerminal(diag *CoastalPortDiagnostics, nodeIdx int) bool {
	return diag != nil &&
		nodeIdx >= 0 &&
		nodeIdx < len(diag.NodeTerminalCell) &&
		diag.NodeTerminalCell[nodeIdx] >= 0
}

func hasDeepwaterTerminal(diag *CoastalPortDiagnostics, nodeIdx int) bool {
	return diag != nil &&
		nodeIdx >= 0 &&
		nodeIdx < len(diag.NodeDeepwaterTermCell) &&
		diag.NodeDeepwaterTermCell[nodeIdx] >= 0
}

func coastalPortCentralityMaxima(trade *TradeNetworkResult, riverTrade *RiverTradeResult) (float64, float64) {
	maxTrade := 0.0
	if trade != nil && trade.Diagnostics != nil {
		for _, v := range trade.Diagnostics.NodeCentrality {
			if v > maxTrade {
				maxTrade = v
			}
		}
	}
	maxRiver := 0.0
	if riverTrade != nil && riverTrade.Diagnostics != nil {
		for _, v := range riverTrade.Diagnostics.NodeCentrality {
			if v > maxRiver {
				maxRiver = v
			}
		}
	}
	return maxTrade, maxRiver
}

func eligibleMajorCoastalPort(node SettlementNode, tradeNorm, riverNorm float64, settings MaritimePortSettings) bool {
	centrality := math.Max(tradeNorm, riverNorm)
	switch settlementNodePortAnchorTier(node) {
	case SettlementNodeCity, SettlementNodeTown:
		return centrality >= settings.RegionalMinCentrality
	case SettlementNodeVillage:
		return centrality >= settings.DistrictMinCentrality
	default:
		return centrality >= settings.LocalMinCentrality
	}
}

func settlementNodePortAnchorTier(node SettlementNode) SettlementNodeKind {
	effectiveRank := settlementNodeEffectiveRank(node)
	switch {
	case effectiveRank >= float64(SettlementNodeCity):
		return SettlementNodeCity
	case effectiveRank >= float64(SettlementNodeTown):
		return SettlementNodeTown
	case effectiveRank >= float64(SettlementNodeVillage):
		return SettlementNodeVillage
	default:
		return SettlementNodeHamlet
	}
}

func populateBaseCoastalNodeScores(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	elevation []float64,
	seaLevel float64,
	diag *CoastalPortDiagnostics,
	settings MaritimePortSettings,
) {
	if network == nil || diag == nil || len(diag.NodePortScore) == 0 {
		return
	}
	adj := BuildFlatAdjacency(cells)
	cellCount := len(cells)
	if cellCount <= 0 {
		cellCount = len(elevation)
	}
	catchmentHops := meshResolutionAdjustedSteps(settings.NodeCatchmentHops, cellCount)
	stepScale := meshPathCostResolutionScale(cellCount)
	for i, node := range network.Nodes {
		cellIdx := node.CellIndex
		if cellIdx < 0 || cellIdx >= len(elevation) {
			continue
		}
		best := 0.0
		bestCell := -1
		bestDeepwater := 0.0
		bestDeepwaterCell := -1
		portSamples := make([]float64, 0, 16)
		deepwaterSamples := make([]float64, 0, 16)
		type state struct {
			cell int
			hops int
		}
		seen := map[int]struct{}{cellIdx: {}}
		queue := []state{{cell: cellIdx, hops: 0}}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			decay := math.Pow(settings.NodeCatchmentDecay, float64(cur.hops)*stepScale)
			if isCoastalLand(cur.cell, elevation, seaLevel, adj) {
				candidate := decay * (diag.PortSuitability[cur.cell] + settings.NodeFeatureWeight*maritimePortFeatureScore(cur.cell, diag, settings))
				portSamples = append(portSamples, candidate)
				if candidate > best {
					best = candidate
					bestCell = cur.cell
				}
				deepwaterCandidate := decay * (diag.DeepwaterSuitability[cur.cell] + settings.NodeFeatureWeight*maritimeDeepwaterFeatureScore(cur.cell, diag, settings))
				deepwaterSamples = append(deepwaterSamples, deepwaterCandidate)
				if deepwaterCandidate > bestDeepwater {
					bestDeepwater = deepwaterCandidate
					bestDeepwaterCell = cur.cell
				}
			}
			if cur.hops >= catchmentHops {
				continue
			}
			for _, neighbor := range adj.GetNeighbors(cur.cell) {
				if neighbor < 0 || neighbor >= len(elevation) || elevation[neighbor] < seaLevel {
					continue
				}
				if _, ok := seen[neighbor]; ok {
					continue
				}
				seen[neighbor] = struct{}{}
				queue = append(queue, state{cell: neighbor, hops: cur.hops + 1})
			}
		}
		// The catchment disc holds ~1/stepScale times as many coastal cells at fine
		// meshes as at the baseline, so a raw max over it inflates with resolution and
		// clears the absolute port thresholds more often. Score with the scale-stable
		// order statistic; keep the raw argmax for *locating* the terminal cell, which
		// is a positional choice and carries no threshold comparison.
		portScore := meshScaleStableMaxOfLinearSamples(portSamples, cellCount)
		deepwaterScore := meshScaleStableMaxOfLinearSamples(deepwaterSamples, cellCount)
		if bestCell < 0 {
			portScore = best
		}
		if bestDeepwaterCell < 0 {
			deepwaterScore = bestDeepwater
		}
		diag.NodePortScore[i] = clamp01(portScore)
		if i < len(diag.NodeBasePortScore) {
			diag.NodeBasePortScore[i] = clamp01(portScore)
		}
		if i < len(diag.NodeDeepwaterScore) {
			diag.NodeDeepwaterScore[i] = clamp01(deepwaterScore)
		}
		if i < len(diag.NodeBaseDeepwaterScore) {
			diag.NodeBaseDeepwaterScore[i] = clamp01(deepwaterScore)
		}
		if i < len(diag.NodeTerminalCell) {
			diag.NodeTerminalCell[i] = bestCell
		}
		if i < len(diag.NodeDeepwaterTermCell) {
			diag.NodeDeepwaterTermCell[i] = bestDeepwaterCell
		}
	}
}

func maritimePortFeatureScore(cellIdx int, diag *CoastalPortDiagnostics, settings MaritimePortSettings) float64 {
	if diag == nil || cellIdx < 0 {
		return 0
	}
	harbor := 0.0
	if cellIdx < len(diag.HarborShelter) {
		harbor = diag.HarborShelter[cellIdx]
	}
	estuary := 0.0
	if cellIdx < len(diag.EstuaryAccess) {
		estuary = diag.EstuaryAccess[cellIdx]
	}
	transfer := 0.0
	if cellIdx < len(diag.RiverTransfer) {
		transfer = diag.RiverTransfer[cellIdx]
	}
	stopover := 0.0
	if cellIdx < len(diag.StopoverValue) {
		stopover = diag.StopoverValue[cellIdx]
	}
	maxFeature := math.Max(harbor, math.Max(estuary, math.Max(transfer, stopover)))
	weightedFeature := settings.NodeFeatureHarborWeight*harbor +
		settings.NodeFeatureEstuaryWeight*estuary +
		settings.NodeFeatureRiverTransferWeight*transfer +
		settings.NodeFeatureStopoverWeight*stopover
	return clamp01(math.Max(maxFeature, weightedFeature))
}

func maritimeDeepwaterFeatureScore(cellIdx int, diag *CoastalPortDiagnostics, settings MaritimePortSettings) float64 {
	if diag == nil || cellIdx < 0 {
		return 0
	}
	deepwater := 0.0
	if cellIdx < len(diag.DeepwaterAccess) {
		deepwater = diag.DeepwaterAccess[cellIdx]
	}
	harbor := 0.0
	if cellIdx < len(diag.HarborShelter) {
		harbor = diag.HarborShelter[cellIdx]
	}
	estuary := 0.0
	if cellIdx < len(diag.EstuaryAccess) {
		estuary = diag.EstuaryAccess[cellIdx]
	}
	transfer := 0.0
	if cellIdx < len(diag.RiverTransfer) {
		transfer = diag.RiverTransfer[cellIdx]
	}
	return clamp01(
		settings.DeepwaterAccessWeight*deepwater +
			settings.DeepwaterHarborWeight*harbor +
			settings.DeepwaterEstuaryWeight*estuary +
			settings.DeepwaterTransferWeight*transfer,
	)
}
