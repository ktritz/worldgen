package climgen

import (
	"math"
	"sort"
)

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
	ID              int
	CellIndex       int
	Kind            MaritimeStopoverKind
	Score           float64
	ComponentAreaEq float64
}

func BuildMaritimeStopoverNodes(
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	ports *CoastalPortResult,
	elevation []float64,
	seaLevel float64,
) []MaritimeStopoverNode {
	stopovers, _ := BuildMaritimeStopoverNodesWithDiagnostics(nil, cells, network, ports, elevation, seaLevel)
	return stopovers
}

func BuildMaritimeStopoverNodesWithDiagnostics(
	sites []Vector3D,
	cells []VoronoiCell,
	network *SettlementNetworkResult,
	ports *CoastalPortResult,
	elevation []float64,
	seaLevel float64,
) ([]MaritimeStopoverNode, MaritimeStopoverDiagnostics) {
	diagnostics := MaritimeStopoverDiagnostics{}
	if ports == nil || ports.Diagnostics == nil || len(cells) == 0 {
		return nil, diagnostics
	}
	adj := BuildFlatAdjacency(cells)
	anchored := map[int]struct{}{}
	if network != nil {
		for _, node := range network.Nodes {
			anchored[node.CellIndex] = struct{}{}
		}
	}
	settings := effectiveMaritimeStopoverSelectionSettings(ports)
	componentAreaEq := maritimeLandComponentAreaEq(cells, elevation, seaLevel)
	candidates := make([]MaritimeStopoverNode, 0)
	for cellIdx, portKind := range ports.Types {
		if cellIdx < 0 || cellIdx >= len(elevation) || elevation[cellIdx] < seaLevel {
			continue
		}
		if _, ok := anchored[cellIdx]; ok {
			continue
		}
		stopover := ports.Diagnostics.StopoverValue[cellIdx]
		suitability := ports.Diagnostics.PortSuitability[cellIdx]
		if stopover < settings.MinStopoverValue && suitability < settings.MinPortSuitability {
			continue
		}
		oceanFrac, _, _, landFrac := coastalNeighborStats(cellIdx, adj, elevation, seaLevel, nil, nil)
		areaEq := 0.0
		if cellIdx < len(componentAreaEq) {
			areaEq = componentAreaEq[cellIdx]
		}
		score := maritimeStopoverScore(stopover, suitability, oceanFrac, landFrac, areaEq, settings)
		if score < settings.ScoreFloor {
			continue
		}
		kind := classifyMaritimeStopoverKind(portKind, stopover, oceanFrac, landFrac)
		diagnostics.CandidateCount++
		switch kind {
		case MaritimeStopoverIsland:
			diagnostics.CandidateIslandCount++
		case MaritimeStopoverStrait:
			diagnostics.CandidateStraitCount++
		case MaritimeStopoverRoadstead:
			diagnostics.CandidateRoadsteadCount++
		}
		addMaritimeStopoverComponentCandidateDiagnostics(areaEq, &diagnostics)
		candidates = append(candidates, MaritimeStopoverNode{
			CellIndex:       cellIdx,
			Kind:            kind,
			Score:           score,
			ComponentAreaEq: areaEq,
		})
	}
	if diagnostics.CandidateCount > 0 {
		diagnostics.MeanCandidateComponentEq /= float64(diagnostics.CandidateCount)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].CellIndex < candidates[j].CellIndex
	})
	selected := make([]MaritimeStopoverNode, 0, len(candidates))
	minSpacingHops := meshResolutionAdjustedSteps(2, len(cells))
	minSpacingDeg := maritimeStopoverSpacingDegrees(sites, cells, minSpacingHops)
	diagnostics.MeanNeighborDegrees = maritimeMeanNeighborDegrees(sites, cells)
	diagnostics.MinSpacingDegrees = minSpacingDeg
	for _, cand := range candidates {
		if !respectsMaritimeStopoverPhysicalSpacing(cand.CellIndex, selected, sites, cells, minSpacingHops, minSpacingDeg) {
			diagnostics.BaseSpacingRejected++
			continue
		}
		selected = append(selected, cand)
	}
	for i := range selected {
		selected[i].ID = i
	}
	diagnostics.BaseSelectedCount = len(selected)
	diagnostics.SelectedCount = len(selected)
	fillMaritimeStopoverSelectedDiagnostics(selected, &diagnostics)
	fillMaritimeStopoverSpacingDiagnostics(selected, sites, &diagnostics)
	return selected, diagnostics
}

func effectiveMaritimeStopoverSelectionSettings(ports *CoastalPortResult) MaritimeStopoverSelectionSettings {
	if ports != nil {
		settings := ports.StopoverSelection
		if settings.Validate() == nil {
			return settings
		}
	}
	return DefaultMaritimeStopoverSelectionSettings()
}

func maritimeStopoverScore(stopover, suitability, oceanFrac, landFrac, areaEq float64, settings MaritimeStopoverSelectionSettings) float64 {
	raw := settings.StopoverValueWeight*stopover +
		settings.PortSuitabilityWeight*suitability +
		settings.OceanExposureWeight*oceanFrac +
		settings.LandScarcityWeight*(1-landFrac)
	return clamp01(raw * maritimeStopoverComponentScoreFactor(areaEq, settings))
}

func maritimeStopoverComponentScoreFactor(areaEq float64, settings MaritimeStopoverSelectionSettings) float64 {
	minFactor := clamp01(settings.MinComponentScoreFactor)
	if areaEq <= 0 || settings.FullComponentAreaEq <= 0 {
		return minFactor
	}
	ratio := clamp01(areaEq / settings.FullComponentAreaEq)
	if settings.ComponentTaperPower > 0 && settings.ComponentTaperPower != 1 {
		ratio = math.Pow(ratio, settings.ComponentTaperPower)
	}
	return minFactor + (1-minFactor)*ratio
}

func fillMaritimeStopoverSelectedDiagnostics(stopovers []MaritimeStopoverNode, diagnostics *MaritimeStopoverDiagnostics) {
	if diagnostics == nil {
		return
	}
	diagnostics.IslandCount = 0
	diagnostics.StraitCount = 0
	diagnostics.RoadsteadCount = 0
	diagnostics.SelectedTinyComponentEq = 0
	diagnostics.SelectedSmallComponentEq = 0
	diagnostics.SelectedMediumComponentEq = 0
	diagnostics.SelectedLargeComponentEq = 0
	diagnostics.MeanSelectedComponentEq = 0
	diagnostics.MeanScore = 0
	for _, stop := range stopovers {
		diagnostics.MeanScore += stop.Score
		addMaritimeStopoverComponentSelectedDiagnostics(stop.ComponentAreaEq, diagnostics)
		switch stop.Kind {
		case MaritimeStopoverIsland:
			diagnostics.IslandCount++
		case MaritimeStopoverStrait:
			diagnostics.StraitCount++
		case MaritimeStopoverRoadstead:
			diagnostics.RoadsteadCount++
		}
	}
	if len(stopovers) > 0 {
		diagnostics.MeanScore /= float64(len(stopovers))
		diagnostics.MeanSelectedComponentEq /= float64(len(stopovers))
	}
}

func addMaritimeStopoverComponentCandidateDiagnostics(areaEq float64, diagnostics *MaritimeStopoverDiagnostics) {
	if diagnostics == nil || areaEq <= 0 {
		return
	}
	diagnostics.MeanCandidateComponentEq += areaEq
	switch maritimeComponentAreaBucket(areaEq) {
	case 0:
		diagnostics.CandidateTinyComponentEq++
	case 1:
		diagnostics.CandidateSmallComponentEq++
	case 2:
		diagnostics.CandidateMediumComponentEq++
	default:
		diagnostics.CandidateLargeComponentEq++
	}
}

func addMaritimeStopoverComponentSelectedDiagnostics(areaEq float64, diagnostics *MaritimeStopoverDiagnostics) {
	if diagnostics == nil || areaEq <= 0 {
		return
	}
	diagnostics.MeanSelectedComponentEq += areaEq
	switch maritimeComponentAreaBucket(areaEq) {
	case 0:
		diagnostics.SelectedTinyComponentEq++
	case 1:
		diagnostics.SelectedSmallComponentEq++
	case 2:
		diagnostics.SelectedMediumComponentEq++
	default:
		diagnostics.SelectedLargeComponentEq++
	}
}

func maritimeComponentAreaBucket(areaEq float64) int {
	switch {
	case areaEq < 1:
		return 0
	case areaEq < 4:
		return 1
	case areaEq < 16:
		return 2
	default:
		return 3
	}
}

func maritimeLandComponentAreaEq(cells []VoronoiCell, elevation []float64, seaLevel float64) []float64 {
	if len(cells) == 0 || len(elevation) == 0 {
		return nil
	}
	areaEq := make([]float64, len(cells))
	seen := make([]bool, len(cells))
	scale := meshPathCostResolutionScale(len(cells))
	areaScale := scale * scale
	for start := range cells {
		if start >= len(elevation) || seen[start] || elevation[start] < seaLevel {
			continue
		}
		component := make([]int, 0)
		queue := []int{start}
		seen[start] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			component = append(component, cur)
			for _, raw := range cells[cur].NeighborSiteIndices {
				neighbor := int(raw)
				if neighbor < 0 || neighbor >= len(cells) || neighbor >= len(elevation) || seen[neighbor] || elevation[neighbor] < seaLevel {
					continue
				}
				seen[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
		eq := float64(len(component)) * areaScale
		for _, cellIdx := range component {
			areaEq[cellIdx] = eq
		}
	}
	return areaEq
}

func fillMaritimeStopoverSpacingDiagnostics(stopovers []MaritimeStopoverNode, sites []Vector3D, diagnostics *MaritimeStopoverDiagnostics) {
	if diagnostics == nil {
		return
	}
	diagnostics.MeanSelectedSpacingDegrees = 0
	if len(stopovers) < 2 || len(sites) == 0 {
		return
	}
	total := 0.0
	count := 0
	for i := 0; i < len(stopovers); i++ {
		cellIdx := stopovers[i].CellIndex
		if cellIdx < 0 || cellIdx >= len(sites) {
			continue
		}
		best := 0.0
		for j := 0; j < len(stopovers); j++ {
			if i == j {
				continue
			}
			other := stopovers[j].CellIndex
			if other < 0 || other >= len(sites) {
				continue
			}
			d := greatCircleDistanceDeg(sites[cellIdx], sites[other])
			if best == 0 || d < best {
				best = d
			}
		}
		if best > 0 {
			total += best
			count++
		}
	}
	if count > 0 {
		diagnostics.MeanSelectedSpacingDegrees = total / float64(count)
	}
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

func respectsMaritimeStopoverPhysicalSpacing(cellIdx int, existing []MaritimeStopoverNode, sites []Vector3D, cells []VoronoiCell, minHops int, minDeg float64) bool {
	if minDeg > 0 && len(sites) == len(cells) && cellIdx >= 0 && cellIdx < len(sites) {
		for _, node := range existing {
			if node.CellIndex < 0 || node.CellIndex >= len(sites) {
				continue
			}
			if greatCircleDistanceDeg(sites[cellIdx], sites[node.CellIndex]) < minDeg {
				return false
			}
		}
		return true
	}
	return respectsMaritimeStopoverSpacing(cellIdx, existing, cells, minHops)
}

func maritimeStopoverSpacingDegrees(sites []Vector3D, cells []VoronoiCell, minHops int) float64 {
	if minHops <= 0 {
		return 0
	}
	meanNeighborDeg := maritimeMeanNeighborDegrees(sites, cells)
	if meanNeighborDeg <= 0 {
		return 0
	}
	return float64(minHops) * meanNeighborDeg
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
