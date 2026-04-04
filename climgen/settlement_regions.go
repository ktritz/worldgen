package climgen

type SettlementRegion struct {
	ID          int
	NodeIndices []int
	CenterNode  int
	MeanScore   float64
	Coastal     bool
	River       bool
}

func assignSettlementRegions(result *SettlementNetworkResult) {
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
	for _, link := range result.Links {
		if link.From < 0 || link.From >= len(adj) || link.To < 0 || link.To >= len(adj) {
			continue
		}
		adj[link.From] = append(adj[link.From], link.To)
		adj[link.To] = append(adj[link.To], link.From)
	}

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
}

func selectSettlementRegionCenter(component []int, result *SettlementNetworkResult, adj [][]int) int {
	bestIdx := component[0]
	bestScore := -1.0
	for _, nodeIdx := range component {
		node := result.Nodes[nodeIdx]
		score := node.Score +
			0.08*node.CarryingCapacity +
			0.08*node.UrbanPotential +
			0.05*float64(node.Kind) +
			0.03*float64(len(adj[nodeIdx]))
		if node.Coastal {
			score += 0.03
		}
		if node.River {
			score += 0.02
		}
		if score > bestScore {
			bestScore = score
			bestIdx = nodeIdx
		}
	}
	return bestIdx
}
