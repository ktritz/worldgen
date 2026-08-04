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

type OceanCandidatePortDiagnostics struct {
	RawCandidateCount           int
	CivilizedCandidateCount     int
	FinalCandidateCount         int
	UncivilizedRejected         int
	CivilizationCapRejected     int
	RawPhysical030              int
	RawPhysical036              int
	FinalPhysical030            int
	FinalPhysical036            int
	CandidateMajorPorts         int
	CandidateSecondaryPorts     int
	RawMajorPorts               int
	RawSecondaryPorts           int
	CandidateCivilizations      int
	CivilizationsWithMultiPorts int
	MaxPortsPerCivilization     int
	MeanPortsPerCivilization    float64
	MeanDeepwaterScore          float64
	P10DeepwaterScore           float64
	MedianDeepwaterScore        float64
	P90DeepwaterScore           float64
}

type OceanTradeResult struct {
	Mode                 MaritimeVesselSettings
	Corridors            []OceanTradeCorridor
	CandidatePorts       []int
	Stopovers            []MaritimeStopoverNode
	MajorPorts           []int
	Diagnostics          *OceanTradeDiagnostics
	StopoverDiagnostics  MaritimeStopoverDiagnostics
	EndpointDiagnostics  CoastalTradeEndpointDiagnostics
	PairDiagnostics      CoastalTradePairDiagnostics
	CandidateDiagnostics OceanCandidatePortDiagnostics
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
	rawCandidatePorts := candidateOceanPorts(network, ports, settings)
	civilizedCandidatePorts := civilizedMaritimeCandidatePorts(rawCandidatePorts, civByNode)
	candidatePorts, civCapRejected := capOceanCandidatePortsByCivilization(civilizedCandidatePorts, civByNode, settings.MaxCandidatePortsPerCiv)
	out.CandidateDiagnostics = oceanCandidatePortDiagnostics(rawCandidatePorts, civilizedCandidatePorts, candidatePorts, civCapRejected, network, ports, civByNode)
	out.CandidatePorts = append(out.CandidatePorts, candidatePorts...)
	baseStopovers, stopoverDiagnostics := BuildMaritimeStopoverNodesWithDiagnostics(sites, cells, network, ports, elevation, seaLevel)
	stopovers, stopoverDiagnostics := selectOceanStopovers(baseStopovers, sites, cells, ports, settings, stopoverDiagnostics)
	out.Stopovers = stopovers
	out.StopoverDiagnostics = stopoverDiagnostics
	if len(candidatePorts)+len(stopovers) < 2 {
		out.MajorPorts = append(out.MajorPorts, ports.MajorDeepwaterPorts...)
		return out
	}

	endpoints, edges, distancePrunedPairs := buildOceanTradeEndpointGraph(sites, cells, climate, network, ports, candidatePorts, stopovers, elevation, seaLevel, settings, civByNode)
	out.EndpointDiagnostics = analyzeCoastalEndpointGraph(endpoints, edges, distancePrunedPairs)
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
			recordViableMaritimePair(&out.PairDiagnostics, civIDForNode(civByNode, fromNode), civIDForNode(civByNode, toNode))
			candidates = append(candidates, oceanTradeCandidate{from: fromNode, to: toNode, path: path, flow: flow})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].flow > candidates[j].flow })
	externalDegrees := make([]int, len(network.Nodes))
	internalDegrees := make([]int, len(network.Nodes))
	externalCivDegrees := make([]int, len(proto.Civilizations))
	internalCivDegrees := make([]int, len(proto.Civilizations))
	for _, cand := range candidates {
		fromCiv := civIDForNode(civByNode, cand.from)
		toCiv := civIDForNode(civByNode, cand.to)
		if fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv {
			if externalDegrees[cand.from] >= settings.MaxPartnersPerPort || externalDegrees[cand.to] >= settings.MaxPartnersPerPort {
				out.PairDiagnostics.RejectedPortCap++
				continue
			}
			if externalCivDegrees[fromCiv] >= settings.MaxPartnersPerCivilization || externalCivDegrees[toCiv] >= settings.MaxPartnersPerCivilization {
				out.PairDiagnostics.RejectedCivCap++
				continue
			}
		} else if fromCiv >= 0 && toCiv >= 0 && fromCiv == toCiv {
			if internalDegrees[cand.from] >= settings.MaxPartnersPerPort || internalDegrees[cand.to] >= settings.MaxPartnersPerPort {
				out.PairDiagnostics.RejectedPortCap++
				continue
			}
			if internalCivDegrees[fromCiv] >= settings.MaxPartnersPerCivilization {
				out.PairDiagnostics.RejectedCivCap++
				continue
			}
		} else {
			if externalDegrees[cand.from] >= settings.MaxPartnersPerPort || externalDegrees[cand.to] >= settings.MaxPartnersPerPort {
				out.PairDiagnostics.RejectedPortCap++
				continue
			}
		}
		out.Corridors = append(out.Corridors, buildOceanTradeCorridor(cand.from, cand.to, fromCiv, toCiv, cand.flow, cand.path))
		recordSelectedMaritimePair(&out.PairDiagnostics, fromCiv, toCiv)
		if fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv {
			externalDegrees[cand.from]++
			externalDegrees[cand.to]++
			externalCivDegrees[fromCiv]++
			externalCivDegrees[toCiv]++
		} else if fromCiv >= 0 && toCiv >= 0 && fromCiv == toCiv {
			internalDegrees[cand.from]++
			internalDegrees[cand.to]++
			internalCivDegrees[fromCiv]++
		} else {
			externalDegrees[cand.from]++
			externalDegrees[cand.to]++
		}
	}
	classifyOceanTradeCorridors(out.Corridors, settings)
	applyOceanTradeDiagnostics(out.Corridors, out.Diagnostics)
	out.MajorPorts = identifyMajorOceanTradePorts(network, ports, out.Diagnostics)
	return out
}

func capOceanCandidatePortsByCivilization(candidates []int, civByNode []int, maxPerCiv int) ([]int, int) {
	if maxPerCiv <= 0 || len(candidates) == 0 {
		return candidates, 0
	}
	out := make([]int, 0, len(candidates))
	countByCiv := make(map[int]int)
	rejected := 0
	for _, nodeIdx := range candidates {
		civID := civIDForNode(civByNode, nodeIdx)
		if civID >= 0 && countByCiv[civID] >= maxPerCiv {
			rejected++
			continue
		}
		out = append(out, nodeIdx)
		if civID >= 0 {
			countByCiv[civID]++
		}
	}
	return out, rejected
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
		node := network.Nodes[nodeIdx]
		if !oceanCandidateHasSettlementSupport(node) {
			continue
		}
		if !hasDeepwaterTerminal(ports.Diagnostics, nodeIdx) {
			continue
		}
		if !qualifiesOceanCandidateScore(ports.Diagnostics, nodeIdx, settings.CandidatePortThreshold, settings.CandidatePhysicalDeepwaterFloor) {
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
		if !oceanCandidateHasSettlementSupport(node) {
			continue
		}
		if qualifiesOceanCandidateScore(ports.Diagnostics, i, settings.CandidateSecondaryPortFloor, settings.CandidatePhysicalDeepwaterFloor) {
			out = append(out, i)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return ports.Diagnostics.NodeDeepwaterScore[out[i]] > ports.Diagnostics.NodeDeepwaterScore[out[j]]
	})
	return out
}

func oceanCandidateHasSettlementSupport(node SettlementNode) bool {
	if node.PhysicalSupportArea <= 0 {
		return true
	}
	return node.PhysicalSupportArea >= 0.5
}

func qualifiesOceanCandidateScore(diag *CoastalPortDiagnostics, nodeIdx int, compositeFloor, physicalFloor float64) bool {
	if diag == nil || nodeIdx < 0 || nodeIdx >= len(diag.NodeDeepwaterScore) {
		return false
	}
	if diag.NodeDeepwaterScore[nodeIdx] < compositeFloor {
		return false
	}
	if physicalFloor <= 0 {
		return true
	}
	return oceanCandidatePhysicalDeepwaterScore(diag, nodeIdx) >= physicalFloor
}

func oceanCandidatePhysicalDeepwaterScore(diag *CoastalPortDiagnostics, nodeIdx int) float64 {
	if diag == nil || nodeIdx < 0 {
		return 0
	}
	if nodeIdx < len(diag.NodeBaseDeepwaterScore) && diag.NodeBaseDeepwaterScore[nodeIdx] > 0 {
		return diag.NodeBaseDeepwaterScore[nodeIdx]
	}
	if nodeIdx < len(diag.NodeDeepwaterScore) {
		return diag.NodeDeepwaterScore[nodeIdx]
	}
	return 0
}

func oceanCandidatePortDiagnostics(raw, civilized, final []int, civCapRejected int, network *SettlementNetworkResult, ports *CoastalPortResult, civByNode []int) OceanCandidatePortDiagnostics {
	diag := OceanCandidatePortDiagnostics{
		RawCandidateCount:       len(raw),
		CivilizedCandidateCount: len(civilized),
		FinalCandidateCount:     len(final),
		UncivilizedRejected:     len(raw) - len(civilized),
		CivilizationCapRejected: civCapRejected,
	}
	if network == nil || ports == nil || ports.Diagnostics == nil {
		return diag
	}
	major := make(map[int]struct{}, len(ports.MajorDeepwaterPorts))
	for _, nodeIdx := range ports.MajorDeepwaterPorts {
		major[nodeIdx] = struct{}{}
	}
	for _, nodeIdx := range raw {
		if _, ok := major[nodeIdx]; ok {
			diag.RawMajorPorts++
		} else {
			diag.RawSecondaryPorts++
		}
		physical := oceanCandidatePhysicalDeepwaterScore(ports.Diagnostics, nodeIdx)
		if physical >= 0.30 {
			diag.RawPhysical030++
		}
		if physical >= 0.36 {
			diag.RawPhysical036++
		}
	}
	scores := make([]float64, 0, len(final))
	portsByCiv := make(map[int]int)
	for _, nodeIdx := range final {
		if _, ok := major[nodeIdx]; ok {
			diag.CandidateMajorPorts++
		} else {
			diag.CandidateSecondaryPorts++
		}
		if nodeIdx >= 0 && nodeIdx < len(ports.Diagnostics.NodeDeepwaterScore) {
			scores = append(scores, ports.Diagnostics.NodeDeepwaterScore[nodeIdx])
		}
		physical := oceanCandidatePhysicalDeepwaterScore(ports.Diagnostics, nodeIdx)
		if physical >= 0.30 {
			diag.FinalPhysical030++
		}
		if physical >= 0.36 {
			diag.FinalPhysical036++
		}
		civID := civIDForNode(civByNode, nodeIdx)
		if civID >= 0 {
			portsByCiv[civID]++
		}
	}
	diag.CandidateCivilizations = len(portsByCiv)
	for _, count := range portsByCiv {
		diag.MeanPortsPerCivilization += float64(count)
		if count > diag.MaxPortsPerCivilization {
			diag.MaxPortsPerCivilization = count
		}
		if count > 1 {
			diag.CivilizationsWithMultiPorts++
		}
	}
	if len(portsByCiv) > 0 {
		diag.MeanPortsPerCivilization /= float64(len(portsByCiv))
	}
	if len(scores) > 0 {
		sort.Float64s(scores)
		total := 0.0
		for _, score := range scores {
			total += score
		}
		diag.MeanDeepwaterScore = total / float64(len(scores))
		diag.P10DeepwaterScore = sortedFloatPercentile(scores, 0.10)
		diag.MedianDeepwaterScore = sortedFloatPercentile(scores, 0.50)
		diag.P90DeepwaterScore = sortedFloatPercentile(scores, 0.90)
	}
	return diag
}

func sortedFloatPercentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func selectOceanStopovers(stopovers []MaritimeStopoverNode, sites []Vector3D, cells []VoronoiCell, ports *CoastalPortResult, settings OceanTradeSettings, diagnostics MaritimeStopoverDiagnostics) ([]MaritimeStopoverNode, MaritimeStopoverDiagnostics) {
	if len(stopovers) == 0 {
		diagnostics.SelectedCount = 0
		return nil, diagnostics
	}
	selected := make([]MaritimeStopoverNode, 0, settings.MaxStopovers)
	minSpacingHops := meshResolutionAdjustedSteps(settings.StopoverSpacingHops, len(cells))
	minSpacingDeg := maritimeStopoverSpacingDegrees(sites, cells, minSpacingHops)
	diagnostics.MeanNeighborDegrees = maritimeMeanNeighborDegrees(sites, cells)
	diagnostics.MinSpacingDegrees = minSpacingDeg
	for _, stop := range stopovers {
		score := stop.Score
		if ports != nil && ports.Diagnostics != nil && stop.CellIndex >= 0 && stop.CellIndex < len(ports.Diagnostics.DeepwaterSuitability) {
			score = math.Max(score, ports.Diagnostics.DeepwaterSuitability[stop.CellIndex])
		}
		if score < settings.StopoverScoreFloor {
			diagnostics.OceanScoreRejected++
			continue
		}
		if minSpacingHops > 0 && !respectsMaritimeStopoverPhysicalSpacing(stop.CellIndex, selected, sites, cells, minSpacingHops, minSpacingDeg) {
			diagnostics.OceanSpacingRejected++
			continue
		}
		stop.Score = score
		selected = append(selected, stop)
		if settings.MaxStopovers > 0 && len(selected) >= settings.MaxStopovers {
			break
		}
	}
	diagnostics.SelectedCount = len(selected)
	fillMaritimeStopoverSelectedDiagnostics(selected, &diagnostics)
	fillMaritimeStopoverSpacingDiagnostics(selected, sites, &diagnostics)
	return selected, diagnostics
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
	fromSupport := SettlementNodePhysicalSupportWeight(from)
	toSupport := SettlementNodePhysicalSupportWeight(to)
	base := 0.14 + 0.20*from.Score*fromSupport + 0.20*to.Score*toSupport + 0.18*fromPort + 0.18*toPort + 0.10*(settlementNodeEffectiveRank(from)+settlementNodeEffectiveRank(to))/6.0
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
