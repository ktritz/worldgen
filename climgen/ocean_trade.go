package climgen

import (
	"math"
	"sort"
)

type OceanTradeCorridorTier int

const (
	OceanTradeCorridorLocal OceanTradeCorridorTier = iota
	OceanTradeCorridorRegional
	OceanTradeCorridorPrimary
)

func OceanTradeCorridorTierName(tier OceanTradeCorridorTier) string {
	names := []string{"Local Ocean Route", "Regional Ocean Route", "Primary Ocean Route"}
	if int(tier) < len(names) {
		return names[tier]
	}
	return "Unknown"
}

type OceanTradeCorridor struct {
	ID                int
	FromNode          int
	ToNode            int
	FromCivilization  int
	ToCivilization    int
	TravelCost        float64
	Flow              float64
	MeanExposure      float64
	MeanCurrentAssist float64
	Tier              OceanTradeCorridorTier
	CellPath          []int
	InterCivilization bool
}

type OceanTradeDiagnostics struct {
	RouteIntensity []float64
	NodeCentrality []float64
	RouteExposure  []float64
}

type OceanTradeResult struct {
	Mode                MaritimeVesselSettings
	Corridors           []OceanTradeCorridor
	CandidatePorts      []int
	Stopovers           []MaritimeStopoverNode
	MajorPorts          []int
	Diagnostics         *OceanTradeDiagnostics
	EndpointDiagnostics CoastalTradeEndpointDiagnostics
	PairDiagnostics     CoastalTradePairDiagnostics
}

type oceanTradeCandidate struct {
	from int
	to   int
	path coastalEndpointPath
	flow float64
}

func BuildOceanTradeNetwork(
	sites []Vector3D,
	cells []VoronoiCell,
	climate *SeasonalClimateResult,
	network *SettlementNetworkResult,
	proto *ProtoCivilizationResult,
	ports *CoastalPortResult,
	elevation []float64,
	seaLevel float64,
	settings OceanTradeSettings,
) *OceanTradeResult {
	out := &OceanTradeResult{}
	if network == nil || proto == nil || ports == nil || ports.Diagnostics == nil {
		return out
	}
	out.Mode = ports.Mode
	out.Diagnostics = &OceanTradeDiagnostics{
		RouteIntensity: make([]float64, len(cells)),
		NodeCentrality: make([]float64, len(network.Nodes)),
		RouteExposure:  make([]float64, len(cells)),
	}
	if ports.Mode.OpenOceanCapability < settings.MinOpenOceanCapability || ports.Mode.MaxOpenWaterLeg <= 0 {
		return out
	}

	civByNode := civilizationByNode(network, proto)
	candidatePorts := candidateOceanPorts(network, ports, settings)
	out.CandidatePorts = append(out.CandidatePorts, candidatePorts...)
	stopovers := selectOceanStopovers(BuildMaritimeStopoverNodes(cells, network, ports, elevation, seaLevel), ports, settings)
	out.Stopovers = stopovers
	if len(candidatePorts)+len(stopovers) < 2 {
		out.MajorPorts = append(out.MajorPorts, ports.MajorDeepwaterPorts...)
		return out
	}

	endpoints, edges := buildOceanTradeEndpointGraph(sites, cells, climate, network, ports, stopovers, elevation, seaLevel, settings, civByNode)
	out.EndpointDiagnostics = analyzeCoastalEndpointGraph(endpoints, edges)
	routeBudget := oceanRouteBudget(ports.Mode, settings)
	candidates := make([]oceanTradeCandidate, 0)
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
			flow := oceanTradeFlow(network, proto, ports, civByNode, fromNode, toNode, path.cost, ports.Mode)
			if flow < settings.MinFlow {
				out.PairDiagnostics.FlowBelowMin++
				recordBestRejectedCoastalPair(&out.PairDiagnostics, fromNode, toNode, path.cost, flow)
				continue
			}
			out.PairDiagnostics.ViableCandidates++
			candidates = append(candidates, oceanTradeCandidate{from: fromNode, to: toNode, path: path, flow: flow})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].flow > candidates[j].flow })
	degrees := make([]int, len(network.Nodes))
	civDegrees := make([]int, len(proto.Civilizations))
	for _, cand := range candidates {
		fromCiv := civIDForNode(civByNode, cand.from)
		toCiv := civIDForNode(civByNode, cand.to)
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
		out.Corridors = append(out.Corridors, buildOceanTradeCorridor(cand.from, cand.to, fromCiv, toCiv, cand.flow, cand.path))
		degrees[cand.from]++
		degrees[cand.to]++
		if fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv {
			civDegrees[fromCiv]++
			civDegrees[toCiv]++
		}
	}
	classifyOceanTradeCorridors(out.Corridors, settings)
	applyOceanTradeDiagnostics(out.Corridors, out.Diagnostics)
	out.MajorPorts = identifyMajorOceanTradePorts(network, ports, out.Diagnostics)
	return out
}

func candidateOceanPorts(network *SettlementNetworkResult, ports *CoastalPortResult, settings OceanTradeSettings) []int {
	if network == nil || ports == nil || ports.Diagnostics == nil {
		return nil
	}
	out := make([]int, 0, len(ports.MajorDeepwaterPorts))
	seen := map[int]struct{}{}
	for _, nodeIdx := range ports.MajorDeepwaterPorts {
		if nodeIdx < 0 || nodeIdx >= len(network.Nodes) || nodeIdx >= len(ports.Diagnostics.NodeDeepwaterScore) {
			continue
		}
		if !hasDeepwaterTerminal(ports.Diagnostics, nodeIdx) {
			continue
		}
		if ports.Diagnostics.NodeDeepwaterScore[nodeIdx] < settings.CandidatePortThreshold {
			continue
		}
		seen[nodeIdx] = struct{}{}
		out = append(out, nodeIdx)
	}
	for i, node := range network.Nodes {
		if _, ok := seen[i]; ok || i >= len(ports.Diagnostics.NodeDeepwaterScore) {
			continue
		}
		if !hasDeepwaterTerminal(ports.Diagnostics, i) {
			continue
		}
		if node.Kind < SettlementNodeVillage {
			continue
		}
		if ports.Diagnostics.NodeDeepwaterScore[i] >= settings.CandidateSecondaryPortFloor {
			out = append(out, i)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return ports.Diagnostics.NodeDeepwaterScore[out[i]] > ports.Diagnostics.NodeDeepwaterScore[out[j]]
	})
	return out
}

func selectOceanStopovers(stopovers []MaritimeStopoverNode, ports *CoastalPortResult, settings OceanTradeSettings) []MaritimeStopoverNode {
	if len(stopovers) == 0 {
		return nil
	}
	selected := make([]MaritimeStopoverNode, 0, settings.MaxStopovers)
	for _, stop := range stopovers {
		score := stop.Score
		if ports != nil && ports.Diagnostics != nil && stop.CellIndex >= 0 && stop.CellIndex < len(ports.Diagnostics.DeepwaterSuitability) {
			score = math.Max(score, ports.Diagnostics.DeepwaterSuitability[stop.CellIndex])
		}
		if score < settings.StopoverScoreFloor {
			continue
		}
		stop.Score = score
		selected = append(selected, stop)
		if settings.MaxStopovers > 0 && len(selected) >= settings.MaxStopovers {
			break
		}
	}
	return selected
}

func oceanRouteBudget(mode MaritimeVesselSettings, settings OceanTradeSettings) float64 {
	factor := settings.RouteBudgetBaseFactor +
		settings.RouteBudgetLongHaulWeight*mode.LongHaulTolerance +
		settings.RouteBudgetOpenOceanWeight*mode.OpenOceanCapability +
		settings.RouteBudgetDailyRangeWeight*mode.DailyRange +
		settings.RouteBudgetStopoverWeight*(1-mode.StopoverNeed)
	extraReach := settings.RouteBudgetLegWeight * mode.MaxOpenWaterLeg
	return settings.MaxRouteCost*math.Max(0.25, factor) + extraReach
}

func oceanLegBudget(mode MaritimeVesselSettings, settings OceanTradeSettings) float64 {
	return settings.BaseLegCost + settings.LegScale*mode.MaxOpenWaterLeg*(0.35+0.65*mode.OpenOceanCapability)
}

func oceanTradeFlow(network *SettlementNetworkResult, proto *ProtoCivilizationResult, ports *CoastalPortResult, civByNode []int, fromNode, toNode int, travelCost float64, mode MaritimeVesselSettings) float64 {
	from := network.Nodes[fromNode]
	to := network.Nodes[toNode]
	fromPort := 0.0
	toPort := 0.0
	if fromNode < len(ports.Diagnostics.NodeDeepwaterScore) {
		fromPort = ports.Diagnostics.NodeDeepwaterScore[fromNode]
	}
	if toNode < len(ports.Diagnostics.NodeDeepwaterScore) {
		toPort = ports.Diagnostics.NodeDeepwaterScore[toNode]
	}
	base := 0.14 + 0.20*from.Score + 0.20*to.Score + 0.18*fromPort + 0.18*toPort + 0.10*(float64(from.Kind)+float64(to.Kind))/6.0
	modeFactor := (0.42 + 0.58*mode.PayloadCapacity) * (0.28 + 0.72*mode.OpenOceanCapability) * (0.45 + 0.55*mode.LongHaulTolerance)
	reachPenalty := 1.0 / (1.0 + 0.08*travelCost)
	flow := base * modeFactor * reachPenalty
	fromCiv := civIDForNode(civByNode, fromNode)
	toCiv := civIDForNode(civByNode, toNode)
	if fromCiv >= 0 && toCiv >= 0 {
		if fromCiv == toCiv {
			flow *= 0.82
		} else {
			flow *= 1.10
		}
	}
	return flow
}

func civIDForNode(civByNode []int, nodeIdx int) int {
	if nodeIdx >= 0 && nodeIdx < len(civByNode) {
		return civByNode[nodeIdx]
	}
	return -1
}
