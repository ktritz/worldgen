package climgen

import (
	"math"
	"sort"
)

type CoastalTradeCorridorTier int

const (
	CoastalTradeCorridorLocal CoastalTradeCorridorTier = iota
	CoastalTradeCorridorRegional
	CoastalTradeCorridorPrimary
)

func CoastalTradeCorridorTierName(tier CoastalTradeCorridorTier) string {
	names := []string{"Local Coastal Route", "Regional Coastal Route", "Primary Coastal Route"}
	if int(tier) < len(names) {
		return names[tier]
	}
	return "Unknown"
}

type CoastalTradeCorridor struct {
	ID                int
	FromNode          int
	ToNode            int
	FromStopover      int
	ToStopover        int
	FromCivilization  int
	ToCivilization    int
	TravelCost        float64
	Flow              float64
	MeanExposure      float64
	MeanCurrentAssist float64
	Tier              CoastalTradeCorridorTier
	CellPath          []int
	InterCivilization bool
}

type CoastalTradeDiagnostics struct {
	RouteIntensity []float64
	NodeCentrality []float64
	RouteExposure  []float64
}

type CoastalTradePairDiagnostics struct {
	TotalPairs       int
	MissingEndpoint  int
	NoPath           int
	FlowBelowMin     int
	RejectedPortCap  int
	RejectedCivCap   int
	ViableCandidates int
	BestRejectedCost float64
	BestRejectedFlow float64
	BestRejectedFrom int
	BestRejectedTo   int
}

type CoastalTradeEndpointDiagnostics struct {
	EndpointCount        int
	PortEndpointCount    int
	StopoverCount        int
	EdgeCount            int
	Components           int
	LargestComponent     int
	LargestPortComponent int
	IsolatedPorts        int
}

type CoastalTradeResult struct {
	Mode                MaritimeVesselSettings
	Corridors           []CoastalTradeCorridor
	CandidatePorts      []int
	Stopovers           []MaritimeStopoverNode
	MajorPorts          []int
	Diagnostics         *CoastalTradeDiagnostics
	EndpointDiagnostics CoastalTradeEndpointDiagnostics
	PairDiagnostics     CoastalTradePairDiagnostics
}

func BuildCoastalTradeNetwork(
	sites []Vector3D,
	cells []VoronoiCell,
	climate *SeasonalClimateResult,
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	ports *CoastalPortResult,
	elevation []float64,
	seaLevel float64,
	settings CoastalTradeSettings,
) *CoastalTradeResult {
	out := &CoastalTradeResult{}
	if network == nil || proto == nil || ports == nil || ports.Diagnostics == nil {
		return out
	}
	out.Mode = ports.Mode
	out.Diagnostics = &CoastalTradeDiagnostics{
		RouteIntensity: make([]float64, len(cells)),
		NodeCentrality: make([]float64, len(network.Nodes)),
		RouteExposure:  make([]float64, len(cells)),
	}

	civByNode := civilizationByNode(network, proto)
	candidatePorts := candidateCoastalPorts(network, ports, settings)
	out.CandidatePorts = append(out.CandidatePorts, candidatePorts...)
	stopovers := BuildMaritimeStopoverNodes(cells, network, ports, elevation, seaLevel)
	out.Stopovers = stopovers
	if len(candidatePorts)+len(stopovers) < 2 {
		out.MajorPorts = append(out.MajorPorts, ports.MajorPorts...)
		return out
	}

	endpoints, edges := buildCoastalTradeEndpointGraph(sites, cells, climate, network, proto, ports, stopovers, elevation, seaLevel, settings, civByNode)
	out.EndpointDiagnostics = analyzeCoastalEndpointGraph(endpoints, edges)
	candidates := make([]coastalTradeCandidate, 0)
	degrees := make([]int, len(network.Nodes))
	civDegrees := make([]int, len(proto.Civilizations))
	routeBudget := coastalRouteBudget(ports.Mode, settings)
	for i := 0; i < len(candidatePorts); i++ {
		for j := i + 1; j < len(candidatePorts); j++ {
			out.PairDiagnostics.TotalPairs++
			fromNode := candidatePorts[i]
			toNode := candidatePorts[j]
			startIdx, endIdx := endpointIndexForNode(endpoints, fromNode), endpointIndexForNode(endpoints, toNode)
			if startIdx < 0 || endIdx < 0 {
				out.PairDiagnostics.MissingEndpoint++
				continue
			}
			path := shortestCoastalEndpointPath(endpoints, edges, startIdx, endIdx, routeBudget)
			if !path.ok {
				out.PairDiagnostics.NoPath++
				continue
			}
			flow := coastalTradeFlow(network, proto, ports, civByNode, fromNode, toNode, path.cost, ports.Mode)
			if flow < settings.MinFlow {
				out.PairDiagnostics.FlowBelowMin++
				recordBestRejectedCoastalPair(&out.PairDiagnostics, fromNode, toNode, path.cost, flow)
				continue
			}
			out.PairDiagnostics.ViableCandidates++
			candidates = append(candidates, coastalTradeCandidate{
				from: fromNode,
				to:   toNode,
				path: path,
				flow: flow,
			})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].flow > candidates[j].flow })
	out.Corridors = make([]CoastalTradeCorridor, 0)
	for _, cand := range candidates {
		fromCiv := -1
		toCiv := -1
		if cand.from >= 0 && cand.from < len(civByNode) {
			fromCiv = civByNode[cand.from]
		}
		if cand.to >= 0 && cand.to < len(civByNode) {
			toCiv = civByNode[cand.to]
		}
		if degrees[cand.from] >= settings.MaxPartnersPerPort || degrees[cand.to] >= settings.MaxPartnersPerPort {
			out.PairDiagnostics.RejectedPortCap++
			continue
		}
		if fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv {
			if civDegrees[fromCiv] >= settings.MaxPartnersPerCivilization || civDegrees[toCiv] >= settings.MaxPartnersPerCivilization {
				out.PairDiagnostics.RejectedCivCap++
				continue
			}
		}
		out.Corridors = append(out.Corridors, buildCoastalTradeCorridor(cand.from, cand.to, fromCiv, toCiv, cand.flow, cand.path))
		degrees[cand.from]++
		degrees[cand.to]++
		if fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv {
			civDegrees[fromCiv]++
			civDegrees[toCiv]++
		}
	}
	classifyCoastalTradeCorridors(out.Corridors, settings)
	applyCoastalTradeDiagnostics(out.Corridors, out.Diagnostics)
	out.MajorPorts = identifyMajorCoastalTradePorts(network, ports, out.Diagnostics)
	return out
}

type coastalTradeCandidate struct {
	from int
	to   int
	path coastalEndpointPath
	flow float64
}

func candidateCoastalPorts(network *SettlementNetworkResult, ports *CoastalPortResult, settings CoastalTradeSettings) []int {
	if network == nil || ports == nil || ports.Diagnostics == nil {
		return nil
	}
	out := make([]int, 0)
	for i, node := range network.Nodes {
		if i >= len(ports.Diagnostics.NodePortScore) {
			continue
		}
		if !hasCoastalTerminal(ports.Diagnostics, i) {
			continue
		}
		cellIdx := node.CellIndex
		if cellIdx < 0 || cellIdx >= len(ports.Diagnostics.PortSuitability) {
			continue
		}
		terminalCell := ports.Diagnostics.NodeTerminalCell[i]
		if !node.Coastal && !(node.River && ports.Diagnostics.NodePortScore[i] >= settings.CandidatePortSuitabilityFloor) {
			continue
		}
		if ports.Diagnostics.NodePortScore[i] >= settings.CandidatePortThreshold || qualifiesFallbackCoastalCandidate(node, terminalCell, ports.Diagnostics, settings) {
			out = append(out, i)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return ports.Diagnostics.NodePortScore[out[i]] > ports.Diagnostics.NodePortScore[out[j]]
	})
	return out
}

func qualifiesFallbackCoastalCandidate(node SettlementNode, cellIdx int, diag *CoastalPortDiagnostics, settings CoastalTradeSettings) bool {
	if diag == nil || cellIdx < 0 || cellIdx >= len(diag.PortSuitability) {
		return false
	}
	if node.Kind < SettlementNodeVillage {
		return false
	}
	feature := coastalTerminalFeatureScore(cellIdx, diag)
	if diag.PortSuitability[cellIdx] < settings.CandidatePortSuitabilityFloor || feature < settings.CandidatePortFeatureFloor {
		return false
	}
	return true
}

func coastalTerminalFeatureScore(cellIdx int, diag *CoastalPortDiagnostics) float64 {
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
	return math.Max(harbor, math.Max(estuary, math.Max(transfer, stopover)))
}

func coastalLegBudget(mode MaritimeVesselSettings, settings CoastalTradeSettings) float64 {
	return settings.BaseLegCost + settings.LegScale*mode.MaxCoastalLeg*(0.35+0.65*mode.CoastalCapability)
}

func coastalRouteBudget(mode MaritimeVesselSettings, settings CoastalTradeSettings) float64 {
	factor := settings.RouteBudgetBaseFactor +
		settings.RouteBudgetLongHaulWeight*mode.LongHaulTolerance +
		settings.RouteBudgetCoastalWeight*mode.CoastalCapability +
		settings.RouteBudgetOpenOceanWeight*mode.OpenOceanCapability +
		settings.RouteBudgetDailyRangeWeight*mode.DailyRange +
		settings.RouteBudgetStopoverWeight*(1-mode.StopoverNeed)
	extraReach := settings.RouteBudgetLegWeight * (mode.MaxCoastalLeg + 0.75*mode.MaxOpenWaterLeg)
	return settings.MaxRouteCost*math.Max(0.25, factor) + extraReach
}

func openWaterLegBudget(mode MaritimeVesselSettings, settings CoastalTradeSettings) float64 {
	if mode.MaxOpenWaterLeg <= 0 || mode.OpenOceanCapability <= 0 {
		return 0
	}
	ratio := mode.MaxOpenWaterLeg / math.Max(mode.MaxCoastalLeg, 1e-6)
	factor := math.Max(0.80, clamp01(0.25+0.90*ratio))
	return coastalLegBudget(mode, settings) * factor * (0.65 + 0.35*mode.OpenOceanCapability)
}

func coastalTradeFlow(
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	ports *CoastalPortResult,
	civByNode []int,
	fromNode, toNode int,
	travelCost float64,
	mode MaritimeVesselSettings,
) float64 {
	from := network.Nodes[fromNode]
	to := network.Nodes[toNode]
	fromPort := 0.0
	toPort := 0.0
	if fromNode < len(ports.Diagnostics.NodePortScore) {
		fromPort = ports.Diagnostics.NodePortScore[fromNode]
	}
	if toNode < len(ports.Diagnostics.NodePortScore) {
		toPort = ports.Diagnostics.NodePortScore[toNode]
	}
	base := 0.16 +
		0.24*from.Score +
		0.24*to.Score +
		0.12*fromPort +
		0.12*toPort +
		0.08*(float64(from.Kind)+float64(to.Kind))/6.0
	modeFactor := (0.45 + 0.55*mode.PayloadCapacity) * (0.40 + 0.60*mode.CoastalCapability)
	reachPenalty := 1.0 / (1.0 + 0.12*travelCost)
	flow := base * modeFactor * reachPenalty
	fromCiv := -1
	toCiv := -1
	if fromNode >= 0 && fromNode < len(civByNode) {
		fromCiv = civByNode[fromNode]
	}
	if toNode >= 0 && toNode < len(civByNode) {
		toCiv = civByNode[toNode]
	}
	if fromCiv >= 0 && toCiv >= 0 {
		if fromCiv == toCiv {
			flow *= 0.88
		} else {
			flow *= 1.04
		}
	}
	return flow
}

func endpointIndexForNode(endpoints []coastalTradeEndpoint, nodeIdx int) int {
	for i, endpoint := range endpoints {
		if endpoint.Node == nodeIdx {
			return i
		}
	}
	return -1
}

func bool01f(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func recordBestRejectedCoastalPair(diag *CoastalTradePairDiagnostics, from, to int, cost, flow float64) {
	if diag == nil {
		return
	}
	if flow > diag.BestRejectedFlow || (flow == diag.BestRejectedFlow && (diag.BestRejectedCost == 0 || cost < diag.BestRejectedCost)) {
		diag.BestRejectedFlow = flow
		diag.BestRejectedCost = cost
		diag.BestRejectedFrom = from
		diag.BestRejectedTo = to
	}
}
