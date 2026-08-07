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
	PathDegrees       float64
	CostPerDegree     float64
	Flow              float64
	MeanExposure      float64
	MeanCurrentAssist float64
	FromPortScore     float64
	ToPortScore       float64
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
	TotalPairs                    int
	MissingEndpoint               int
	NoPath                        int
	NoPathInternal                int
	NoPathExternal                int
	NoPathUnknown                 int
	OverBudgetInternal            int
	OverBudgetExternal            int
	OverBudgetUnknown             int
	OverBudgetSamples             int
	BestOverBudgetCost            float64
	BestOverBudgetFrom            int
	BestOverBudgetTo              int
	BestOverBudgetFromCiv         int
	BestOverBudgetToCiv           int
	BestOverBudgetExternal        float64
	BestOverBudgetExternalFrom    int
	BestOverBudgetExternalTo      int
	BestOverBudgetExternalFromCiv int
	BestOverBudgetExternalToCiv   int
	RouteBudget                   float64
	FlowBelowMin                  int
	RejectedPortCap               int
	RejectedCivCap                int
	ViableCandidates              int
	ViableInternal                int
	ViableExternal                int
	ViableUnknown                 int
	SelectedInternal              int
	SelectedExternal              int
	SelectedUnknown               int
	BestRejectedCost              float64
	BestRejectedFlow              float64
	BestRejectedFrom              int
	BestRejectedTo                int
}

type CoastalTradeEndpointDiagnostics struct {
	EndpointCount        int
	PortEndpointCount    int
	StopoverCount        int
	PairCount            int
	DistancePrunedPairs  int
	EdgeCount            int
	MeanDegree           float64
	MaxDegree            int
	MeanEdgeCost         float64
	P90EdgeCost          float64
	Components           int
	LargestComponent     int
	PortComponents       int
	MultiPortComponents  int
	LargestPortComponent int
	SecondPortComponent  int
	MeanPortComponent    float64
	IsolatedPorts        int
	PortComponentsDetail []CoastalTradePortComponentDiagnostics
}

type CoastalTradePortComponentDiagnostics struct {
	Endpoints     int
	Ports         int
	PortNodes     []int
	Civilizations []int
}

type MaritimeStopoverDiagnostics struct {
	CandidateCount             int
	CandidateIslandCount       int
	CandidateStraitCount       int
	CandidateRoadsteadCount    int
	CandidateTinyComponentEq   int
	CandidateSmallComponentEq  int
	CandidateMediumComponentEq int
	CandidateLargeComponentEq  int
	MeanCandidateComponentEq   float64
	BaseSelectedCount          int
	BaseSpacingRejected        int
	OceanScoreRejected         int
	OceanSpacingRejected       int
	SelectedCount              int
	IslandCount                int
	StraitCount                int
	RoadsteadCount             int
	SelectedTinyComponentEq    int
	SelectedSmallComponentEq   int
	SelectedMediumComponentEq  int
	SelectedLargeComponentEq   int
	MeanSelectedComponentEq    float64
	MeanScore                  float64
	MeanNeighborDegrees        float64
	MinSpacingDegrees          float64
	MeanSelectedSpacingDegrees float64
}

type CoastalTradeResult struct {
	Mode                MaritimeVesselSettings
	Corridors           []CoastalTradeCorridor
	CandidatePorts      []int
	Stopovers           []MaritimeStopoverNode
	MajorPorts          []int
	Diagnostics         *CoastalTradeDiagnostics
	StopoverDiagnostics MaritimeStopoverDiagnostics
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
	candidatePorts := civilizedMaritimeCandidatePorts(candidateCoastalPorts(sites, network, ports, settings), civByNode)
	out.CandidatePorts = append(out.CandidatePorts, candidatePorts...)
	stopovers, stopoverDiagnostics := BuildMaritimeStopoverNodesWithDiagnostics(sites, cells, network, ports, elevation, seaLevel)
	out.Stopovers = stopovers
	out.StopoverDiagnostics = stopoverDiagnostics
	if len(candidatePorts)+len(stopovers) < 2 {
		out.MajorPorts = append(out.MajorPorts, ports.MajorPorts...)
		return out
	}

	endpoints, edges, distancePrunedPairs := buildCoastalTradeEndpointGraph(sites, cells, climate, network, proto, ports, candidatePorts, stopovers, elevation, seaLevel, settings, civByNode)
	out.EndpointDiagnostics = analyzeCoastalEndpointGraph(endpoints, edges, distancePrunedPairs)
	candidates := make([]coastalTradeCandidate, 0)
	externalDegrees := make([]int, len(network.Nodes))
	internalDegrees := make([]int, len(network.Nodes))
	externalCivDegrees := make([]int, len(proto.Civilizations))
	routeBudget := coastalRouteBudget(ports.Mode, settings)
	out.PairDiagnostics.RouteBudget = routeBudget
	overBudgetSamples := 0
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
				fromCiv, toCiv := civIDForNode(civByNode, fromNode), civIDForNode(civByNode, toNode)
				recordNoPathMaritimePair(&out.PairDiagnostics, fromCiv, toCiv)
				if fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv && overBudgetSamples < 16 {
					overBudgetSamples++
					out.PairDiagnostics.OverBudgetSamples = overBudgetSamples
					unboundedPath := shortestCoastalEndpointPath(endpoints, edges, startIdx, endIdx, math.Inf(1))
					if unboundedPath.ok {
						recordOverBudgetMaritimePair(&out.PairDiagnostics, fromNode, toNode, fromCiv, toCiv, unboundedPath.cost)
					}
				}
				continue
			}
			flow := coastalTradeFlow(network, proto, ports, civByNode, fromNode, toNode, path.cost, ports.Mode)
			if flow < settings.MinFlow {
				out.PairDiagnostics.FlowBelowMin++
				recordBestRejectedCoastalPair(&out.PairDiagnostics, fromNode, toNode, path.cost, flow)
				continue
			}
			out.PairDiagnostics.ViableCandidates++
			recordViableMaritimePair(&out.PairDiagnostics, civIDForNode(civByNode, fromNode), civIDForNode(civByNode, toNode))
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
	internalCivDegrees := make([]int, len(proto.Civilizations))
	for _, cand := range candidates {
		fromCiv := -1
		toCiv := -1
		if cand.from >= 0 && cand.from < len(civByNode) {
			fromCiv = civByNode[cand.from]
		}
		if cand.to >= 0 && cand.to < len(civByNode) {
			toCiv = civByNode[cand.to]
		}
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
		out.Corridors = append(out.Corridors, buildCoastalTradeCorridor(len(out.Corridors)+1, cand.from, cand.to, fromCiv, toCiv, cand.flow, cand.path, sites, ports))
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

func candidateCoastalPorts(sites []Vector3D, network *SettlementNetworkResult, ports *CoastalPortResult, settings CoastalTradeSettings) []int {
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
	return dedupeCoastalCandidatePortsByTerminal(sites, out, ports.Diagnostics)
}

// coastalPortMergeRadiusBaselineFraction sets the terminal merge radius as a
// fraction of the baseline (L5) mean cell spacing. Half a spacing separates no
// pair of distinct baseline cells, which keeps L5 selection unchanged.
const coastalPortMergeRadiusBaselineFraction = 0.5

// dedupeCoastalCandidatePortsByTerminal keeps the highest-scoring candidate in
// each stretch of coast. Merging by terminal *cell identity* made the merge
// radius one mesh cell — about 112 km at L5 but 28 km at L7 — while the
// catchment that picks the terminal is physically scaled, so a finer mesh
// offered more distinct terminal cells, collided less often, and kept ports a
// coarse mesh merged. Candidate counts diverged about 1.37x per level as a
// result, an artifact created by the dedupe rather than by the port scoring.
//
// Merging within a fixed angular radius instead holds the physical spacing
// constant. The radius is half the L5 mean cell spacing: adjacent cell centres
// on the baseline mesh sit about one full spacing apart, comfortably outside
// half of it even allowing for the spread in Voronoi cell size, so no pair of
// distinct L5 terminals merges and the baseline is an exact no-op. At finer
// meshes the same physical radius absorbs the extra terminals that refinement
// exposes.
func dedupeCoastalCandidatePortsByTerminal(sites []Vector3D, candidates []int, diag *CoastalPortDiagnostics) []int {
	if len(candidates) == 0 || diag == nil {
		return candidates
	}
	mergeRadius := coastalPortMergeRadiusBaselineFraction * MeanCellAngularSpacing(int(baselinePathCostCells))
	minCosine := math.Cos(mergeRadius)
	seenTerminal := make(map[int]struct{}, len(candidates))
	kept := make([]int, 0, len(candidates))
	out := make([]int, 0, len(candidates))
	for _, nodeIdx := range candidates {
		terminal := -1
		if nodeIdx >= 0 && nodeIdx < len(diag.NodeTerminalCell) {
			terminal = diag.NodeTerminalCell[nodeIdx]
		}
		if terminal < 0 {
			out = append(out, nodeIdx)
			continue
		}
		if _, ok := seenTerminal[terminal]; ok {
			continue
		}
		if terminal < len(sites) && withinAngularRadiusOfAny(sites, kept, terminal, minCosine) {
			continue
		}
		seenTerminal[terminal] = struct{}{}
		kept = append(kept, terminal)
		out = append(out, nodeIdx)
	}
	return out
}

func withinAngularRadiusOfAny(sites []Vector3D, kept []int, terminal int, minCosine float64) bool {
	for _, other := range kept {
		if other >= len(sites) {
			continue
		}
		if sites[terminal].Dot(sites[other]) >= minCosine {
			return true
		}
	}
	return false
}

func civilizedMaritimeCandidatePorts(candidates []int, civByNode []int) []int {
	if len(candidates) == 0 {
		return nil
	}
	out := make([]int, 0, len(candidates))
	for _, nodeIdx := range candidates {
		if nodeIdx < 0 || nodeIdx >= len(civByNode) || civByNode[nodeIdx] < 0 {
			continue
		}
		out = append(out, nodeIdx)
	}
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
	fromPort := coastalTerminalQualityForNode(fromNode, ports)
	toPort := coastalTerminalQualityForNode(toNode, ports)
	fromSupport := SettlementNodePhysicalSupportWeight(from)
	toSupport := SettlementNodePhysicalSupportWeight(to)
	base := 0.16 +
		0.24*from.Score*fromSupport +
		0.24*to.Score*toSupport +
		0.12*fromPort +
		0.12*toPort +
		0.08*(settlementNodeEffectiveRank(from)+settlementNodeEffectiveRank(to))/6.0
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

func coastalTerminalQualityForNode(nodeIdx int, ports *CoastalPortResult) float64 {
	if ports == nil || ports.Diagnostics == nil || nodeIdx < 0 {
		return 0
	}
	diag := ports.Diagnostics
	terminalCell := -1
	if nodeIdx < len(diag.NodeTerminalCell) {
		terminalCell = diag.NodeTerminalCell[nodeIdx]
	}
	if terminalCell < 0 {
		return 0
	}
	suitability := 0.0
	if terminalCell < len(diag.PortSuitability) {
		suitability = diag.PortSuitability[terminalCell]
	}
	feature := coastalTerminalFeatureScore(terminalCell, diag)
	return clamp01(math.Max(suitability, feature))
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

func recordViableMaritimePair(diag *CoastalTradePairDiagnostics, fromCiv, toCiv int) {
	if diag == nil {
		return
	}
	switch {
	case fromCiv >= 0 && toCiv >= 0 && fromCiv == toCiv:
		diag.ViableInternal++
	case fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv:
		diag.ViableExternal++
	default:
		diag.ViableUnknown++
	}
}

func recordSelectedMaritimePair(diag *CoastalTradePairDiagnostics, fromCiv, toCiv int) {
	if diag == nil {
		return
	}
	switch {
	case fromCiv >= 0 && toCiv >= 0 && fromCiv == toCiv:
		diag.SelectedInternal++
	case fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv:
		diag.SelectedExternal++
	default:
		diag.SelectedUnknown++
	}
}

func recordNoPathMaritimePair(diag *CoastalTradePairDiagnostics, fromCiv, toCiv int) {
	if diag == nil {
		return
	}
	switch {
	case fromCiv >= 0 && toCiv >= 0 && fromCiv == toCiv:
		diag.NoPathInternal++
	case fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv:
		diag.NoPathExternal++
	default:
		diag.NoPathUnknown++
	}
}

func recordOverBudgetMaritimePair(diag *CoastalTradePairDiagnostics, fromNode, toNode, fromCiv, toCiv int, cost float64) {
	if diag == nil {
		return
	}
	switch {
	case fromCiv >= 0 && toCiv >= 0 && fromCiv == toCiv:
		diag.OverBudgetInternal++
	case fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv:
		diag.OverBudgetExternal++
	case fromCiv < 0 || toCiv < 0:
		diag.OverBudgetUnknown++
	}
	if cost <= 0 {
		return
	}
	if diag.BestOverBudgetCost == 0 || cost < diag.BestOverBudgetCost {
		diag.BestOverBudgetCost = cost
		diag.BestOverBudgetFrom = fromNode
		diag.BestOverBudgetTo = toNode
		diag.BestOverBudgetFromCiv = fromCiv
		diag.BestOverBudgetToCiv = toCiv
	}
	isExternal := fromCiv >= 0 && toCiv >= 0 && fromCiv != toCiv
	if isExternal && (diag.BestOverBudgetExternal == 0 || cost < diag.BestOverBudgetExternal) {
		diag.BestOverBudgetExternal = cost
		diag.BestOverBudgetExternalFrom = fromNode
		diag.BestOverBudgetExternalTo = toNode
		diag.BestOverBudgetExternalFromCiv = fromCiv
		diag.BestOverBudgetExternalToCiv = toCiv
	}
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
