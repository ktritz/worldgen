package climgen

import (
	"container/heap"
	"math"
	"sort"
)

type coastalTradeEndpoint struct {
	Node     int
	Stopover int
	Cell     int
	Civ      int
	Score    float64
	IsPort   bool
}

type coastalEndpointEdge struct {
	neighbor int
	path     coastalCellPath
}

type coastalEndpointPath struct {
	ok           bool
	cost         float64
	cells        []int
	meanExposure float64
	meanAssist   float64
	stopovers    []int
}

type coastalEndpointState struct {
	endpoint int
	cost     float64
}

type coastalEndpointHeap []coastalEndpointState

func (h coastalEndpointHeap) Len() int            { return len(h) }
func (h coastalEndpointHeap) Less(i, j int) bool  { return h[i].cost < h[j].cost }
func (h coastalEndpointHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *coastalEndpointHeap) Push(x interface{}) { *h = append(*h, x.(coastalEndpointState)) }
func (h *coastalEndpointHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func buildCoastalTradeEndpointGraph(
	sites []Vector3D,
	cells []VoronoiCell,
	climate *SeasonalClimateResult,
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	ports *CoastalPortResult,
	candidatePorts []int,
	stopovers []MaritimeStopoverNode,
	elevation []float64,
	seaLevel float64,
	settings CoastalTradeSettings,
	civByNode []int,
) ([]coastalTradeEndpoint, [][]coastalEndpointEdge, int) {
	endpoints := make([]coastalTradeEndpoint, 0, len(ports.MajorPorts)+len(stopovers)+8)
	for _, nodeIdx := range candidatePorts {
		cellIdx := network.Nodes[nodeIdx].CellIndex
		if ports.Diagnostics != nil && nodeIdx >= 0 && nodeIdx < len(ports.Diagnostics.NodeTerminalCell) && ports.Diagnostics.NodeTerminalCell[nodeIdx] >= 0 {
			cellIdx = ports.Diagnostics.NodeTerminalCell[nodeIdx]
		}
		civ := -1
		if nodeIdx >= 0 && nodeIdx < len(civByNode) {
			civ = civByNode[nodeIdx]
		}
		score := 0.0
		if nodeIdx < len(ports.Diagnostics.NodePortScore) {
			score = ports.Diagnostics.NodePortScore[nodeIdx]
		}
		endpoints = append(endpoints, coastalTradeEndpoint{
			Node:   nodeIdx,
			Cell:   cellIdx,
			Civ:    civ,
			Score:  score,
			IsPort: true,
		})
	}
	for _, stopover := range stopovers {
		endpoints = append(endpoints, coastalTradeEndpoint{
			Node:     -1,
			Stopover: stopover.ID,
			Cell:     stopover.CellIndex,
			Civ:      -1,
			Score:    stopover.Score,
			IsPort:   false,
		})
	}
	adj := make([][]coastalEndpointEdge, len(endpoints))
	cellAdj := BuildFlatAdjacency(cells)
	var coastDist []int
	if ports.Mode.OpenOceanCapability > 0 && ports.Mode.MaxOpenWaterLeg > 0 {
		openWaterAllowance := meshResolutionAdjustedSteps(openWaterHopAllowance(ports.Mode), len(cells))
		coastDist = coastalWaterDistanceFromLand(cellAdj, elevation, seaLevel, openWaterAllowance)
	}
	maxLegBudget := coastalLegBudget(ports.Mode, settings)
	if ports.Mode.OpenOceanCapability > 0 && ports.Mode.MaxOpenWaterLeg > 0 {
		maxLegBudget = math.Max(maxLegBudget, openWaterLegBudget(ports.Mode, settings))
	}
	meanNeighborDeg := maritimeMeanNeighborDegrees(sites, cells)
	distancePrunedPairs := 0
	workspace := newMaritimeCellPathWorkspace(len(cells))
	for i := 0; i < len(endpoints); i++ {
		for j := i + 1; j < len(endpoints); j++ {
			if endpoints[i].Cell == endpoints[j].Cell {
				continue
			}
			if !maritimeEndpointPairWithinBudgetLowerBound(sites, cells, endpoints[i].Cell, endpoints[j].Cell, maxLegBudget, 0.18, meanNeighborDeg) {
				distancePrunedPairs++
				continue
			}
			path := shortestCoastalEndpointEdgePathWithWorkspace(sites, cells, cellAdj, coastDist, climate, elevation, seaLevel, endpoints[i].Cell, endpoints[j].Cell, ports.Mode, settings, workspace)
			if !path.ok {
				continue
			}
			adj[i] = append(adj[i], coastalEndpointEdge{neighbor: j, path: path})
			adj[j] = append(adj[j], coastalEndpointEdge{neighbor: i, path: path})
		}
	}
	return endpoints, adj, distancePrunedPairs
}

func shortestCoastalEndpointEdgePath(
	sites []Vector3D,
	cells []VoronoiCell,
	climate *SeasonalClimateResult,
	elevation []float64,
	seaLevel float64,
	start, goal int,
	mode MaritimeVesselSettings,
	settings CoastalTradeSettings,
) coastalCellPath {
	adj := BuildFlatAdjacency(cells)
	var coastDist []int
	if mode.OpenOceanCapability > 0 && mode.MaxOpenWaterLeg > 0 {
		openWaterAllowance := meshResolutionAdjustedSteps(openWaterHopAllowance(mode), len(cells))
		coastDist = coastalWaterDistanceFromLand(adj, elevation, seaLevel, openWaterAllowance)
	}
	return shortestCoastalEndpointEdgePathWithAdj(sites, cells, adj, coastDist, climate, elevation, seaLevel, start, goal, mode, settings)
}

func shortestCoastalEndpointEdgePathWithAdj(
	sites []Vector3D,
	cells []VoronoiCell,
	adj *FlatAdjacency,
	coastDist []int,
	climate *SeasonalClimateResult,
	elevation []float64,
	seaLevel float64,
	start, goal int,
	mode MaritimeVesselSettings,
	settings CoastalTradeSettings,
) coastalCellPath {
	legBudget := coastalLegBudget(mode, settings)
	path := shortestMaritimeCellPathWithWorkspace(sites, cells, adj, nil, climate, elevation, seaLevel, start, goal, mode, legBudget, false, newMaritimeCellPathWorkspace(len(cells)))
	if path.ok || mode.OpenOceanCapability <= 0 || mode.MaxOpenWaterLeg <= 0 {
		return path
	}
	return shortestMaritimeCellPathWithWorkspace(sites, cells, adj, coastDist, climate, elevation, seaLevel, start, goal, mode, openWaterLegBudget(mode, settings), true, newMaritimeCellPathWorkspace(len(cells)))
}

func shortestCoastalEndpointEdgePathWithWorkspace(
	sites []Vector3D,
	cells []VoronoiCell,
	adj *FlatAdjacency,
	coastDist []int,
	climate *SeasonalClimateResult,
	elevation []float64,
	seaLevel float64,
	start, goal int,
	mode MaritimeVesselSettings,
	settings CoastalTradeSettings,
	workspace *maritimeCellPathWorkspace,
) coastalCellPath {
	legBudget := coastalLegBudget(mode, settings)
	path := shortestMaritimeCellPathWithWorkspace(sites, cells, adj, nil, climate, elevation, seaLevel, start, goal, mode, legBudget, false, workspace)
	if path.ok || mode.OpenOceanCapability <= 0 || mode.MaxOpenWaterLeg <= 0 {
		return path
	}
	return shortestMaritimeCellPathWithWorkspace(sites, cells, adj, coastDist, climate, elevation, seaLevel, start, goal, mode, openWaterLegBudget(mode, settings), true, workspace)
}

func shortestCoastalEndpointPath(endpoints []coastalTradeEndpoint, adj [][]coastalEndpointEdge, start, goal int, maxCost float64) coastalEndpointPath {
	if start < 0 || goal < 0 || start >= len(endpoints) || goal >= len(endpoints) {
		return coastalEndpointPath{}
	}
	dist := make([]float64, len(endpoints))
	prev := make([]int, len(endpoints))
	prevEdge := make([]int, len(endpoints))
	for i := range dist {
		dist[i] = math.Inf(1)
		prev[i] = -1
		prevEdge[i] = -1
	}
	dist[start] = 0
	pq := &coastalEndpointHeap{{endpoint: start, cost: 0}}
	heap.Init(pq)
	for pq.Len() > 0 {
		cur := heap.Pop(pq).(coastalEndpointState)
		if cur.cost > dist[cur.endpoint] || cur.cost > maxCost {
			continue
		}
		if cur.endpoint == goal {
			break
		}
		for edgeIdx, edge := range adj[cur.endpoint] {
			nextCost := cur.cost + edge.path.cost
			if nextCost < dist[edge.neighbor] && nextCost <= maxCost {
				dist[edge.neighbor] = nextCost
				prev[edge.neighbor] = cur.endpoint
				prevEdge[edge.neighbor] = edgeIdx
				heap.Push(pq, coastalEndpointState{endpoint: edge.neighbor, cost: nextCost})
			}
		}
	}
	if math.IsInf(dist[goal], 1) {
		return coastalEndpointPath{}
	}
	sequence := make([]int, 0)
	for cur := goal; cur >= 0; cur = prev[cur] {
		sequence = append(sequence, cur)
		if cur == start {
			break
		}
	}
	for i, j := 0, len(sequence)-1; i < j; i, j = i+1, j-1 {
		sequence[i], sequence[j] = sequence[j], sequence[i]
	}
	cells := make([]int, 0)
	stopovers := make([]int, 0)
	sumExposure := 0.0
	sumAssist := 0.0
	legs := 0.0
	for i := 0; i+1 < len(sequence); i++ {
		from := sequence[i]
		to := sequence[i+1]
		edge := adj[from][prevEdge[to]]
		if i == 0 {
			cells = append(cells, edge.path.cells...)
		} else {
			cells = append(cells, edge.path.cells[1:]...)
		}
		sumExposure += edge.path.meanExposure
		sumAssist += edge.path.meanAssist
		legs++
		if !endpoints[to].IsPort && endpoints[to].Stopover >= 0 && to != goal {
			stopovers = append(stopovers, endpoints[to].Stopover)
		}
	}
	meanExposure := 0.0
	meanAssist := 0.0
	if legs > 0 {
		meanExposure = sumExposure / legs
		meanAssist = sumAssist / legs
	}
	return coastalEndpointPath{
		ok:           true,
		cost:         dist[goal],
		cells:        cells,
		meanExposure: meanExposure,
		meanAssist:   meanAssist,
		stopovers:    stopovers,
	}
}

func analyzeCoastalEndpointGraph(endpoints []coastalTradeEndpoint, adj [][]coastalEndpointEdge, distancePrunedPairs int) CoastalTradeEndpointDiagnostics {
	out := CoastalTradeEndpointDiagnostics{
		EndpointCount:       len(endpoints),
		PortEndpointCount:   0,
		StopoverCount:       0,
		PairCount:           len(endpoints) * (len(endpoints) - 1) / 2,
		DistancePrunedPairs: distancePrunedPairs,
	}
	for _, endpoint := range endpoints {
		if endpoint.IsPort {
			out.PortEndpointCount++
		} else {
			out.StopoverCount++
		}
	}
	for i := range adj {
		out.EdgeCount += len(adj[i])
		if len(adj[i]) > out.MaxDegree {
			out.MaxDegree = len(adj[i])
		}
		for _, edge := range adj[i] {
			if edge.path.cost > 0 {
				out.MeanEdgeCost += edge.path.cost
			}
		}
	}
	out.EdgeCount /= 2
	if len(adj) > 0 {
		out.MeanDegree = float64(out.EdgeCount*2) / float64(len(adj))
	}
	if out.EdgeCount > 0 {
		out.MeanEdgeCost /= float64(out.EdgeCount * 2)
		edgeCosts := make([]float64, 0, out.EdgeCount)
		for i := range adj {
			for _, edge := range adj[i] {
				if i < edge.neighbor {
					edgeCosts = append(edgeCosts, edge.path.cost)
				}
			}
		}
		sort.Float64s(edgeCosts)
		if len(edgeCosts) > 0 {
			idx := int(math.Ceil(0.90*float64(len(edgeCosts)))) - 1
			if idx < 0 {
				idx = 0
			}
			if idx >= len(edgeCosts) {
				idx = len(edgeCosts) - 1
			}
			out.P90EdgeCost = edgeCosts[idx]
		}
	}

	seen := make([]bool, len(endpoints))
	for i := range endpoints {
		if seen[i] {
			continue
		}
		out.Components++
		size := 0
		portCount := 0
		portNodes := make([]int, 0)
		civilizations := make([]int, 0)
		queue := []int{i}
		seen[i] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			size++
			if endpoints[cur].IsPort {
				portCount++
				if endpoints[cur].Node >= 0 {
					portNodes = append(portNodes, endpoints[cur].Node)
				}
				if endpoints[cur].Civ >= 0 {
					civilizations = append(civilizations, endpoints[cur].Civ)
				}
			}
			for _, edge := range adj[cur] {
				if edge.neighbor < 0 || edge.neighbor >= len(endpoints) || seen[edge.neighbor] {
					continue
				}
				seen[edge.neighbor] = true
				queue = append(queue, edge.neighbor)
			}
		}
		if size > out.LargestComponent {
			out.LargestComponent = size
		}
		if portCount > out.LargestPortComponent {
			out.SecondPortComponent = out.LargestPortComponent
			out.LargestPortComponent = portCount
		} else if portCount > out.SecondPortComponent {
			out.SecondPortComponent = portCount
		}
		if portCount > 0 {
			out.PortComponents++
			out.MeanPortComponent += float64(portCount)
			sort.Ints(portNodes)
			sort.Ints(civilizations)
			out.PortComponentsDetail = append(out.PortComponentsDetail, CoastalTradePortComponentDiagnostics{
				Endpoints:     size,
				Ports:         portCount,
				PortNodes:     uniqueSortedInts(portNodes),
				Civilizations: uniqueSortedInts(civilizations),
			})
			if portCount > 1 {
				out.MultiPortComponents++
			}
		}
	}
	if out.PortComponents > 0 {
		out.MeanPortComponent /= float64(out.PortComponents)
	}
	sort.Slice(out.PortComponentsDetail, func(i, j int) bool {
		left := out.PortComponentsDetail[i]
		right := out.PortComponentsDetail[j]
		if left.Ports != right.Ports {
			return left.Ports > right.Ports
		}
		if left.Endpoints != right.Endpoints {
			return left.Endpoints > right.Endpoints
		}
		if len(left.PortNodes) == 0 || len(right.PortNodes) == 0 {
			return len(left.PortNodes) > len(right.PortNodes)
		}
		return left.PortNodes[0] < right.PortNodes[0]
	})
	for i, endpoint := range endpoints {
		if !endpoint.IsPort || len(adj[i]) > 0 {
			continue
		}
		out.IsolatedPorts++
	}
	return out
}

func uniqueSortedInts(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	out := values[:0]
	last := 0
	for i, value := range values {
		if i > 0 && value == last {
			continue
		}
		out = append(out, value)
		last = value
	}
	return append([]int(nil), out...)
}
