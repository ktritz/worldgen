package climgen

import (
	"container/heap"
	"math"
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
	stopovers []MaritimeStopoverNode,
	elevation []float64,
	seaLevel float64,
	settings CoastalTradeSettings,
	civByNode []int,
) ([]coastalTradeEndpoint, [][]coastalEndpointEdge) {
	endpoints := make([]coastalTradeEndpoint, 0, len(ports.MajorPorts)+len(stopovers)+8)
	for _, nodeIdx := range candidateCoastalPorts(network, ports, settings) {
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
	for i := 0; i < len(endpoints); i++ {
		for j := i + 1; j < len(endpoints); j++ {
			path := shortestCoastalEndpointEdgePath(sites, cells, climate, elevation, seaLevel, endpoints[i].Cell, endpoints[j].Cell, ports.Mode, settings)
			if !path.ok {
				continue
			}
			adj[i] = append(adj[i], coastalEndpointEdge{neighbor: j, path: path})
			adj[j] = append(adj[j], coastalEndpointEdge{neighbor: i, path: path})
		}
	}
	return endpoints, adj
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
	legBudget := coastalLegBudget(mode, settings)
	path := shortestCoastalCellPath(sites, cells, climate, elevation, seaLevel, start, goal, mode, legBudget)
	if path.ok || mode.OpenOceanCapability <= 0 || mode.MaxOpenWaterLeg <= 0 {
		return path
	}
	return shortestMaritimeCellPath(sites, cells, climate, elevation, seaLevel, start, goal, mode, openWaterLegBudget(mode, settings), true)
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

func analyzeCoastalEndpointGraph(endpoints []coastalTradeEndpoint, adj [][]coastalEndpointEdge) CoastalTradeEndpointDiagnostics {
	out := CoastalTradeEndpointDiagnostics{
		EndpointCount:     len(endpoints),
		PortEndpointCount: 0,
		StopoverCount:     0,
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
	}
	out.EdgeCount /= 2

	seen := make([]bool, len(endpoints))
	for i := range endpoints {
		if seen[i] {
			continue
		}
		out.Components++
		size := 0
		portCount := 0
		queue := []int{i}
		seen[i] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			size++
			if endpoints[cur].IsPort {
				portCount++
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
			out.LargestPortComponent = portCount
		}
	}
	for i, endpoint := range endpoints {
		if !endpoint.IsPort || len(adj[i]) > 0 {
			continue
		}
		out.IsolatedPorts++
	}
	return out
}
