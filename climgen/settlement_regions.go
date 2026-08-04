package climgen

import "math"

type SettlementRegion struct {
	ID          int
	NodeIndices []int
	CenterNode  int
	MeanScore   float64
	Coastal     bool
	River       bool
}

func assignSettlementRegions(result *SettlementNetworkResult, cells []VoronoiCell, settings SettlementNetworkSettings) {
	if result == nil || len(result.Nodes) == 0 {
		return
	}
	if result.Diagnostics == nil {
		result.Diagnostics = &SettlementNetworkDiagnostics{}
	}
	result.Diagnostics.RegionByNode = make([]int, len(result.Nodes))
	for i := range result.Diagnostics.RegionByNode {
		result.Diagnostics.RegionByNode[i] = -1
	}

	adj := make([][]int, len(result.Nodes))
	seen := make(map[[2]int]struct{}, len(result.Links))
	for _, link := range result.Links {
		if link.From < 0 || link.From >= len(adj) || link.To < 0 || link.To >= len(adj) {
			continue
		}
		addSettlementRegionAdjacency(adj, seen, link.From, link.To)
		result.Diagnostics.RegionFormation.TransportLinks++
	}
	transportDegree := make([]int, len(adj))
	for i, links := range adj {
		transportDegree[i] = len(links)
	}
	transportComponent := settlementRegionComponentIDs(adj)
	result.Diagnostics.RegionFormation.PhysicalClusterLinks = addPhysicalSettlementRegionAdjacency(result, cells, settings, adj, seen, transportDegree, transportComponent)

	regions := make([]SettlementRegion, 0)
	visited := make([]bool, len(result.Nodes))
	for start := range result.Nodes {
		if visited[start] {
			continue
		}
		stack := []int{start}
		visited[start] = true
		component := make([]int, 0)
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			component = append(component, cur)
			for _, next := range adj[cur] {
				if !visited[next] {
					visited[next] = true
					stack = append(stack, next)
				}
			}
		}

		center := selectSettlementRegionCenter(component, result, adj)
		meanScore := 0.0
		coastal := false
		river := false
		for _, nodeIdx := range component {
			result.Diagnostics.RegionByNode[nodeIdx] = len(regions)
			meanScore += result.Nodes[nodeIdx].Score
			coastal = coastal || result.Nodes[nodeIdx].Coastal
			river = river || result.Nodes[nodeIdx].River
		}
		meanScore /= float64(len(component))
		regions = append(regions, SettlementRegion{
			ID:          len(regions),
			NodeIndices: component,
			CenterNode:  center,
			MeanScore:   meanScore,
			Coastal:     coastal,
			River:       river,
		})
	}
	result.Regions = regions
	result.Diagnostics.RegionFormation.RegionCount = len(regions)
}

func addPhysicalSettlementRegionAdjacency(
	result *SettlementNetworkResult,
	cells []VoronoiCell,
	settings SettlementNetworkSettings,
	adj [][]int,
	seen map[[2]int]struct{},
	transportDegree []int,
	transportComponent []int,
) int {
	if result == nil || result.Diagnostics == nil || len(cells) == 0 || len(result.Diagnostics.MovementCost) != len(cells) {
		return 0
	}
	added := 0
	maxClusterTravel := maxFloat(
		maxFloat(settings.HamletMaxTravel, settings.VillageMaxTravel),
		maxFloat(settings.TownMaxTravel, settings.CityMaxTravel),
	)
	for i, source := range result.Nodes {
		if source.Kind < SettlementNodeVillage || source.CellIndex < 0 || source.CellIndex >= len(cells) {
			continue
		}
		dist, _ := shortestPathsFromNode(source.CellIndex, cells, result.Diagnostics.MovementCost, maxClusterTravel)
		for j := i + 1; j < len(result.Nodes); j++ {
			target := result.Nodes[j]
			if target.Kind < SettlementNodeVillage || target.CellIndex < 0 || target.CellIndex >= len(dist) {
				continue
			}
			pairTravel := maxFloat(nodeTravelLimitForNode(source, settings), nodeTravelLimitForNode(target, settings))
			if dist[target.CellIndex] > pairTravel {
				continue
			}
			result.Diagnostics.RegionFormation.PhysicalReachablePairs++
			x, y := orderedNodePair(i, j)
			if _, ok := seen[[2]int{x, y}]; ok {
				result.Diagnostics.RegionFormation.PhysicalAlreadyLinkedPairs++
				continue
			}
			sameTransportComponent := i < len(transportComponent) &&
				j < len(transportComponent) &&
				transportComponent[i] >= 0 &&
				transportComponent[i] == transportComponent[j]
			if i < len(transportDegree) && j < len(transportDegree) && transportDegree[i] > 0 && transportDegree[j] > 0 && sameTransportComponent {
				result.Diagnostics.RegionFormation.PhysicalSkippedSameComponentPairs++
				result.Diagnostics.RegionFormation.PhysicalSkippedTransportConnectedPairs++
				continue
			}
			if i < len(transportDegree) && j < len(transportDegree) && transportDegree[i] > 0 && transportDegree[j] > 0 {
				result.Diagnostics.RegionFormation.PhysicalSkippedCrossComponentPairs++
			}
			result.Diagnostics.RegionFormation.PhysicalSingletonCandidatePairs++
			if addSettlementRegionAdjacency(adj, seen, i, j) {
				added++
			}
		}
	}
	return added
}

func settlementRegionComponentIDs(adj [][]int) []int {
	component := make([]int, len(adj))
	for i := range component {
		component[i] = -1
	}
	componentID := 0
	for start := range adj {
		if component[start] >= 0 {
			continue
		}
		stack := []int{start}
		component[start] = componentID
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, next := range adj[cur] {
				if next < 0 || next >= len(adj) || component[next] >= 0 {
					continue
				}
				component[next] = componentID
				stack = append(stack, next)
			}
		}
		componentID++
	}
	return component
}

func addSettlementRegionAdjacency(adj [][]int, seen map[[2]int]struct{}, a, b int) bool {
	if a < 0 || a >= len(adj) || b < 0 || b >= len(adj) || a == b {
		return false
	}
	x, y := orderedNodePair(a, b)
	key := [2]int{x, y}
	if _, ok := seen[key]; ok {
		return false
	}
	seen[key] = struct{}{}
	adj[a] = append(adj[a], b)
	adj[b] = append(adj[b], a)
	return true
}

func selectSettlementRegionCenter(component []int, result *SettlementNetworkResult, adj [][]int) int {
	bestIdx := component[0]
	bestRank := -1.0
	bestScore := -1.0
	for _, nodeIdx := range component {
		node := result.Nodes[nodeIdx]
		rank := settlementNodeEffectiveRank(node)
		score := node.Score +
			0.08*node.CarryingCapacity +
			0.08*node.UrbanPotential +
			0.03*float64(len(adj[nodeIdx]))
		if node.Coastal {
			score += 0.03
		}
		if node.River {
			score += 0.02
		}
		if rank > bestRank+0.05 || (math.Abs(rank-bestRank) <= 0.05 && score > bestScore) {
			bestRank = rank
			bestScore = score
			bestIdx = nodeIdx
		}
	}
	return bestIdx
}
