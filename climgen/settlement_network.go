package climgen

import (
	"container/heap"
	"math"
	"sort"
)

type SettlementNodeKind int

const (
	SettlementNodeHamlet SettlementNodeKind = iota
	SettlementNodeVillage
	SettlementNodeTown
	SettlementNodeCity
)

func SettlementNodeKindName(kind SettlementNodeKind) string {
	names := []string{"Local Anchor", "District Anchor", "Regional Anchor", "Major Hub"}
	if int(kind) < len(names) {
		return names[kind]
	}
	return "Unknown"
}

type SettlementNode struct {
	ID               int
	CellIndex        int
	Kind             SettlementNodeKind
	Score            float64
	CarryingCapacity float64
	UrbanPotential   float64
	Coastal          bool
	River            bool
}

type SettlementLink struct {
	From       int
	To         int
	TravelCost float64
	Path       []int
}

type SettlementNetworkDiagnostics struct {
	MovementCost []float64
	NodeByCell   []int
	RegionByNode []int
}

type SettlementNetworkResult struct {
	Nodes       []SettlementNode
	Links       []SettlementLink
	Regions     []SettlementRegion
	Diagnostics *SettlementNetworkDiagnostics
}

type SettlementNetworkSettings struct {
	HamletThreshold  float64
	VillageThreshold float64
	TownThreshold    float64
	CityThreshold    float64

	HamletSpacingDeg  float64
	VillageSpacingDeg float64
	TownSpacingDeg    float64
	CitySpacingDeg    float64

	HamletMaxTravel  float64
	VillageMaxTravel float64
	TownMaxTravel    float64
	CityMaxTravel    float64
}

func DefaultSettlementNetworkSettings() SettlementNetworkSettings {
	return SettlementNetworkSettings{
		HamletThreshold:  0.38,
		VillageThreshold: 0.46,
		TownThreshold:    0.55,
		CityThreshold:    0.64,

		HamletSpacingDeg:  5.5,
		VillageSpacingDeg: 7.5,
		TownSpacingDeg:    12.0,
		CitySpacingDeg:    18.0,

		HamletMaxTravel:  6.4,
		VillageMaxTravel: 8.8,
		TownMaxTravel:    11.2,
		CityMaxTravel:    14.5,
	}
}

func BuildSettlementNetwork(
	sites []Vector3D,
	cells []VoronoiCell,
	settlements *SettlementResult,
	population *PopulationResult,
	biomes *BiomeResult,
	soils *SoilResult,
	resources *ResourceResult,
	elevation []float64,
	seaLevel float64,
	settings SettlementNetworkSettings,
) *SettlementNetworkResult {
	n := len(elevation)
	out := &SettlementNetworkResult{
		Diagnostics: &SettlementNetworkDiagnostics{
			MovementCost: make([]float64, n),
			NodeByCell:   make([]int, n),
		},
	}
	for i := range out.Diagnostics.NodeByCell {
		out.Diagnostics.NodeByCell[i] = -1
	}
	if settlements == nil || settlements.Diagnostics == nil || population == nil || population.Diagnostics == nil {
		return out
	}

	for i := 0; i < n; i++ {
		out.Diagnostics.MovementCost[i] = movementCostForCell(i, settlements, biomes, soils, elevation, seaLevel)
	}

	candidates := settlementNodeCandidates(sites, cells, settlements, population, resources, elevation, seaLevel, settings)
	out.Nodes = filterSettlementNodeCandidates(candidates, sites, settings)
	out.Nodes = addWaystationBridges(sites, cells, out.Nodes, out.Diagnostics.MovementCost, settlements, population, resources, elevation, seaLevel, settings)
	for i := range out.Nodes {
		out.Nodes[i].ID = i
		out.Diagnostics.NodeByCell[out.Nodes[i].CellIndex] = i
	}
	out.Links = buildSettlementLinks(cells, out.Nodes, out.Diagnostics.MovementCost, settlements, resources, settings)
	out.Nodes, out.Links, out.Diagnostics.NodeByCell = pruneIsolatedNodes(out.Nodes, out.Links, out.Diagnostics.NodeByCell, settlements, resources)
	assignSettlementRegions(out)
	return out
}

func movementCostForCell(
	idx int,
	settlements *SettlementResult,
	biomes *BiomeResult,
	soils *SoilResult,
	elevation []float64,
	seaLevel float64,
) float64 {
	if idx < 0 || idx >= len(elevation) || elevation[idx] < seaLevel {
		return math.Inf(1)
	}
	cost := 1.0
	if soils != nil && soils.Diagnostics != nil {
		if idx < len(soils.Diagnostics.Relief) {
			cost += 1.25 * smoothstep01(120, 1400, soils.Diagnostics.Relief[idx])
		}
		if idx < len(soils.Diagnostics.Rockiness) {
			cost += 0.70 * soils.Diagnostics.Rockiness[idx]
		}
	}
	if biomes != nil && biomes.Diagnostics != nil {
		if idx < len(biomes.Diagnostics.AnnualIceFraction) {
			cost += 1.80 * biomes.Diagnostics.AnnualIceFraction[idx]
		}
		if idx < len(biomes.Diagnostics.AridityRatio) {
			cost += 0.55 * smoothstep01(0.22, 0.95, biomes.Diagnostics.AridityRatio[idx])
		}
		if idx < len(biomes.Diagnostics.WetlandAffinity) {
			cost += 0.50 * biomes.Diagnostics.WetlandAffinity[idx]
		}
	}
	if settlements != nil && settlements.Diagnostics != nil {
		if idx < len(settlements.Diagnostics.RiverBonus) {
			cost -= 0.22 * settlements.Diagnostics.RiverBonus[idx]
		}
		if idx < len(settlements.Diagnostics.CoastalBonus) {
			cost -= 0.18 * settlements.Diagnostics.CoastalBonus[idx]
		}
	}
	return math.Max(0.35, cost)
}

func settlementNodeCandidates(
	sites []Vector3D,
	cells []VoronoiCell,
	settlements *SettlementResult,
	population *PopulationResult,
	resources *ResourceResult,
	elevation []float64,
	seaLevel float64,
	settings SettlementNetworkSettings,
) []SettlementNode {
	candidates := make([]SettlementNode, 0)
	for i := range elevation {
		if elevation[i] < seaLevel {
			continue
		}
		carrying := population.Diagnostics.CarryingCapacity[i]
		urban := population.Diagnostics.UrbanPotential[i]
		score := clamp01(0.58*carrying + 0.42*urban)
		kind, ok := classifySettlementNodeCandidate(carrying, urban, settings)
		if !ok {
			continue
		}
		if !isLocalSettlementPeak(i, score, cells, population) {
			continue
		}
		resourceExceptional := settlementResourceExceptional(resources, i)
		if settlements.Classes[i] < SettlementMarginal && !resourceExceptional {
			continue
		}
		candidates = append(candidates, SettlementNode{
			CellIndex:        i,
			Kind:             kind,
			Score:            score,
			CarryingCapacity: carrying,
			UrbanPotential:   urban,
			Coastal:          settlements.Diagnostics.CoastalBonus[i] >= 0.16,
			River:            settlements.Diagnostics.RiverBonus[i] >= 0.24,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Kind != candidates[j].Kind {
			return candidates[i].Kind > candidates[j].Kind
		}
		return candidates[i].Score > candidates[j].Score
	})
	return candidates
}

func classifySettlementNodeCandidate(carrying, urban float64, settings SettlementNetworkSettings) (SettlementNodeKind, bool) {
	switch {
	case urban >= settings.CityThreshold && carrying >= settings.TownThreshold:
		return SettlementNodeCity, true
	case urban >= settings.TownThreshold || carrying >= settings.TownThreshold:
		return SettlementNodeTown, true
	case carrying >= settings.VillageThreshold:
		return SettlementNodeVillage, true
	case carrying >= settings.HamletThreshold:
		return SettlementNodeHamlet, true
	default:
		return 0, false
	}
}

func isLocalSettlementPeak(idx int, score float64, cells []VoronoiCell, population *PopulationResult) bool {
	if score <= 0 {
		return false
	}
	for _, neighbor := range cells[idx].NeighborSiteIndices {
		j := int(neighbor)
		if j < 0 || j >= len(population.Diagnostics.CarryingCapacity) {
			continue
		}
		neighborScore := clamp01(0.58*population.Diagnostics.CarryingCapacity[j] + 0.42*population.Diagnostics.UrbanPotential[j])
		if neighborScore > score+0.02 {
			return false
		}
	}
	return true
}

func settlementResourceExceptional(resources *ResourceResult, idx int) bool {
	if resources == nil || resources.Diagnostics == nil {
		return false
	}
	score := resourceFuelSupport(resources, idx)
	if idx < len(resources.Diagnostics.GoldAffinity) {
		score += 0.35 * resources.Diagnostics.GoldAffinity[idx]
	}
	if idx < len(resources.Diagnostics.GemAffinity) {
		score += 0.35 * resources.Diagnostics.GemAffinity[idx]
	}
	return score >= 0.35
}

func filterSettlementNodeCandidates(candidates []SettlementNode, sites []Vector3D, settings SettlementNetworkSettings) []SettlementNode {
	kept := make([]SettlementNode, 0, len(candidates))
	for _, candidate := range candidates {
		tooClose := false
		for _, existing := range kept {
			minSpacing := math.Max(nodeSpacingDeg(candidate.Kind, settings), nodeSpacingDeg(existing.Kind, settings)*0.55)
			if greatCircleDistanceDeg(sites[candidate.CellIndex], sites[existing.CellIndex]) < minSpacing {
				tooClose = true
				break
			}
		}
		if !tooClose {
			kept = append(kept, candidate)
		}
	}
	return kept
}

func buildSettlementLinks(
	cells []VoronoiCell,
	nodes []SettlementNode,
	movementCost []float64,
	settlements *SettlementResult,
	resources *ResourceResult,
	settings SettlementNetworkSettings,
) []SettlementLink {
	if len(nodes) == 0 {
		return nil
	}
	nodeByCell := make(map[int]int, len(nodes))
	for i, node := range nodes {
		nodeByCell[node.CellIndex] = i
	}
	links := make([]SettlementLink, 0)
	seen := make(map[[2]int]struct{})
	for i, node := range nodes {
		maxTravel := nodeTravelLimit(node.Kind, settings)
		dist, prev := shortestPathsFromNode(node.CellIndex, cells, movementCost, maxTravel)
		targets := candidateLinkTargets(i, node, nodes, nodeByCell, dist, settlements, resources, maxTravel)
		for _, targetIdx := range targets {
			a, b := orderedNodePair(i, targetIdx)
			key := [2]int{a, b}
			if _, ok := seen[key]; ok {
				continue
			}
			if math.IsInf(dist[nodes[targetIdx].CellIndex], 1) {
				continue
			}
			seen[key] = struct{}{}
			links = append(links, SettlementLink{
				From:       a,
				To:         b,
				TravelCost: dist[nodes[targetIdx].CellIndex],
				Path:       reconstructSettlementPath(node.CellIndex, nodes[targetIdx].CellIndex, prev),
			})
		}
	}
	sort.Slice(links, func(i, j int) bool { return links[i].TravelCost < links[j].TravelCost })
	return links
}

func candidateLinkTargets(
	sourceIdx int,
	source SettlementNode,
	nodes []SettlementNode,
	nodeByCell map[int]int,
	dist []float64,
	settlements *SettlementResult,
	resources *ResourceResult,
	maxTravel float64,
) []int {
	type target struct {
		idx   int
		score float64
	}
	targets := make([]target, 0)
	for j, node := range nodes {
		if j == sourceIdx || math.IsInf(dist[node.CellIndex], 1) || dist[node.CellIndex] > maxTravel {
			continue
		}
		if source.Kind <= SettlementNodeVillage && node.Kind < source.Kind {
			continue
		}
		if source.Kind >= SettlementNodeTown && node.Kind == SettlementNodeHamlet {
			continue
		}
		value := 0.55*node.Score + 0.20*float64(node.Kind) - 0.07*dist[node.CellIndex]
		if node.Coastal {
			value += 0.06
		}
		if settlementResourceExceptional(resources, node.CellIndex) {
			value += 0.04
		}
		targets = append(targets, target{idx: j, score: value})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].score > targets[j].score })
	limit := 1
	switch source.Kind {
	case SettlementNodeVillage:
		limit = 2
	case SettlementNodeTown:
		limit = 2
	case SettlementNodeCity:
		limit = 3
	}
	if len(targets) < limit {
		limit = len(targets)
	}
	out := make([]int, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, targets[i].idx)
	}
	return out
}

func pruneIsolatedNodes(
	nodes []SettlementNode,
	links []SettlementLink,
	nodeByCell []int,
	settlements *SettlementResult,
	resources *ResourceResult,
) ([]SettlementNode, []SettlementLink, []int) {
	if len(nodes) == 0 {
		return nodes, links, nodeByCell
	}
	degree := make([]int, len(nodes))
	for _, link := range links {
		degree[link.From]++
		degree[link.To]++
	}
	keep := make([]bool, len(nodes))
	newNodes := make([]SettlementNode, 0, len(nodes))
	oldToNew := make([]int, len(nodes))
	for i := range oldToNew {
		oldToNew[i] = -1
	}
	for i, node := range nodes {
		exceptional := settlementResourceExceptional(resources, node.CellIndex) || node.Coastal || node.River
		if degree[i] == 0 {
			switch node.Kind {
			case SettlementNodeHamlet:
				if !settlementResourceExceptional(resources, node.CellIndex) {
					continue
				}
			case SettlementNodeVillage:
				if !settlementResourceExceptional(resources, node.CellIndex) && settlements.Diagnostics.Suitability[node.CellIndex] < 0.72 {
					continue
				}
			case SettlementNodeTown:
				if !exceptional && settlements.Diagnostics.Suitability[node.CellIndex] < 0.70 {
					continue
				}
			}
		}
		if degree[i] <= 1 && node.Kind == SettlementNodeHamlet && !exceptional && settlements.Diagnostics.AccessScore[node.CellIndex] < 0.58 {
			continue
		}
		keep[i] = true
		oldToNew[i] = len(newNodes)
		node.ID = len(newNodes)
		newNodes = append(newNodes, node)
	}
	newLinks := make([]SettlementLink, 0, len(links))
	for _, link := range links {
		if !keep[link.From] || !keep[link.To] {
			continue
		}
		newLinks = append(newLinks, SettlementLink{
			From:       oldToNew[link.From],
			To:         oldToNew[link.To],
			TravelCost: link.TravelCost,
			Path:       append([]int(nil), link.Path...),
		})
	}
	for i := range nodeByCell {
		if nodeByCell[i] >= 0 {
			nodeByCell[i] = oldToNew[nodeByCell[i]]
		}
	}
	return newNodes, newLinks, nodeByCell
}

func nodeSpacingDeg(kind SettlementNodeKind, settings SettlementNetworkSettings) float64 {
	switch kind {
	case SettlementNodeCity:
		return settings.CitySpacingDeg
	case SettlementNodeTown:
		return settings.TownSpacingDeg
	case SettlementNodeVillage:
		return settings.VillageSpacingDeg
	default:
		return settings.HamletSpacingDeg
	}
}

func nodeTravelLimit(kind SettlementNodeKind, settings SettlementNetworkSettings) float64 {
	switch kind {
	case SettlementNodeCity:
		return settings.CityMaxTravel
	case SettlementNodeTown:
		return settings.TownMaxTravel
	case SettlementNodeVillage:
		return settings.VillageMaxTravel
	default:
		return settings.HamletMaxTravel
	}
}

func greatCircleDistanceDeg(a, b Vector3D) float64 {
	dot := a.X*b.X + a.Y*b.Y + a.Z*b.Z
	if dot > 1 {
		dot = 1
	}
	if dot < -1 {
		dot = -1
	}
	return math.Acos(dot) * 180 / math.Pi
}

func orderedNodePair(a, b int) (int, int) {
	if a < b {
		return a, b
	}
	return b, a
}

type pathState struct {
	cell int
	dist float64
}

type pathHeap []pathState

func (h pathHeap) Len() int            { return len(h) }
func (h pathHeap) Less(i, j int) bool  { return h[i].dist < h[j].dist }
func (h pathHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *pathHeap) Push(x interface{}) { *h = append(*h, x.(pathState)) }
func (h *pathHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func shortestPathsFromNode(start int, cells []VoronoiCell, movementCost []float64, maxDist float64) ([]float64, []int) {
	dist := make([]float64, len(cells))
	prev := make([]int, len(cells))
	for i := range dist {
		dist[i] = math.Inf(1)
		prev[i] = -1
	}
	dist[start] = 0
	stepScale := meshPathCostResolutionScale(len(cells))
	pq := &pathHeap{{cell: start, dist: 0}}
	heap.Init(pq)
	for pq.Len() > 0 {
		cur := heap.Pop(pq).(pathState)
		if cur.dist > dist[cur.cell] || cur.dist > maxDist {
			continue
		}
		for _, neighbor := range cells[cur.cell].NeighborSiteIndices {
			j := int(neighbor)
			if j < 0 || j >= len(cells) || math.IsInf(movementCost[j], 1) {
				continue
			}
			step := 0.5 * (movementCost[cur.cell] + movementCost[j]) * stepScale
			nd := cur.dist + step
			if nd < dist[j] && nd <= maxDist {
				dist[j] = nd
				prev[j] = cur.cell
				heap.Push(pq, pathState{cell: j, dist: nd})
			}
		}
	}
	return dist, prev
}

func reconstructSettlementPath(start, goal int, prev []int) []int {
	if start == goal {
		return []int{start}
	}
	path := make([]int, 0)
	cur := goal
	for cur >= 0 {
		path = append(path, cur)
		if cur == start {
			break
		}
		cur = prev[cur]
	}
	if len(path) == 0 || path[len(path)-1] != start {
		return nil
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}
