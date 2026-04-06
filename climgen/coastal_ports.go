package climgen

import (
	"math"
)

type CoastalPortType int

const (
	CoastalPortOcean CoastalPortType = iota
	CoastalPortNone
	CoastalPortBeachLanding
	CoastalPortHarbor
	CoastalPortEstuary
	CoastalPortIslandStopover
)

func CoastalPortTypeName(kind CoastalPortType) string {
	names := []string{
		"Ocean",
		"None",
		"Beach Landing",
		"Harbor Port",
		"Estuary Port",
		"Island Stopover",
	}
	if int(kind) < len(names) {
		return names[kind]
	}
	return "Unknown"
}

type CoastalPortDiagnostics struct {
	CoastalAccess         []float64
	DeepwaterAccess       []float64
	HarborShelter         []float64
	EstuaryAccess         []float64
	RiverTransfer         []float64
	StopoverValue         []float64
	StormExposure         []float64
	PortSuitability       []float64
	DeepwaterSuitability  []float64
	NodePortScore         []float64
	NodeDeepwaterScore    []float64
	NodeTerminalCell      []int
	NodeDeepwaterTermCell []int
}

type CoastalPortResult struct {
	Mode                MaritimeVesselSettings
	Types               []CoastalPortType
	MajorPorts          []int
	MajorDeepwaterPorts []int
	Diagnostics         *CoastalPortDiagnostics
}

func BuildCoastalPorts(
	cells []VoronoiCell,
	climate *SeasonalClimateResult,
	network *SettlementNetworkResult,
	trade *TradeNetworkResult,
	riverTrade *RiverTradeResult,
	coastalResources *CoastalResourceResult,
	riverRoutes *RiverRouteResult,
	soils *SoilResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	maritime MaritimeRouteSettings,
	settings MaritimePortSettings,
) *CoastalPortResult {
	n := len(elevation)
	result := &CoastalPortResult{
		Types: make([]CoastalPortType, n),
		Diagnostics: &CoastalPortDiagnostics{
			CoastalAccess:        make([]float64, n),
			DeepwaterAccess:      make([]float64, n),
			HarborShelter:        make([]float64, n),
			EstuaryAccess:        make([]float64, n),
			RiverTransfer:        make([]float64, n),
			StopoverValue:        make([]float64, n),
			StormExposure:        make([]float64, n),
			PortSuitability:      make([]float64, n),
			DeepwaterSuitability: make([]float64, n),
		},
	}
	if network != nil {
		result.Diagnostics.NodePortScore = make([]float64, len(network.Nodes))
		result.Diagnostics.NodeDeepwaterScore = make([]float64, len(network.Nodes))
		result.Diagnostics.NodeTerminalCell = make([]int, len(network.Nodes))
		result.Diagnostics.NodeDeepwaterTermCell = make([]int, len(network.Nodes))
		for i := range result.Diagnostics.NodeTerminalCell {
			result.Diagnostics.NodeTerminalCell[i] = -1
			result.Diagnostics.NodeDeepwaterTermCell[i] = -1
		}
	}
	mode, ok := maritime.VesselByName(maritime.DefaultVessel)
	if !ok && len(maritime.Vessels) > 0 {
		mode = maritime.Vessels[0]
	}
	result.Mode = mode
	if n == 0 {
		return result
	}

	adj := BuildFlatAdjacency(cells)
	coastalExposure := ComputeCoastalExposure(cells, elevation, seaLevel)
	windSpeed := meanSeasonalSurfaceWind(climate)
	currentSpeed := meanCurrentSpeed(climate)

	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			result.Types[i] = CoastalPortOcean
			continue
		}
		if !isCoastalLand(i, elevation, seaLevel, adj) {
			result.Types[i] = CoastalPortNone
			continue
		}

		coastal := coastalValue(coastalExposure, i)
		deepwater := deriveDeepwaterAccess(i, adj, elevation, seaLevel, coastal)
		harbor := deriveHarborShelter(i, adj, elevation, seaLevel, coastal, windSpeed, currentSpeed)
		estuary := derivePortEstuaryAccess(i, hydro, soils, coastalResources, coastal)
		transfer := derivePortRiverTransfer(i, hydro, riverRoutes, coastal)
		stopover := deriveStopoverValue(i, adj, elevation, seaLevel, coastal, harbor, coastalResources)
		storm := deriveCoastalStormExposure(i, adj, elevation, seaLevel, coastal, windSpeed, currentSpeed)
		suitability := derivePortSuitability(harbor, estuary, transfer, stopover, storm, mode, settings)
		deepwaterSuitability := deriveDeepwaterSuitability(deepwater, harbor, estuary, transfer, storm, mode, settings)

		result.Diagnostics.CoastalAccess[i] = coastal
		result.Diagnostics.DeepwaterAccess[i] = deepwater
		result.Diagnostics.HarborShelter[i] = harbor
		result.Diagnostics.EstuaryAccess[i] = estuary
		result.Diagnostics.RiverTransfer[i] = transfer
		result.Diagnostics.StopoverValue[i] = stopover
		result.Diagnostics.StormExposure[i] = storm
		result.Diagnostics.PortSuitability[i] = suitability
		result.Diagnostics.DeepwaterSuitability[i] = deepwaterSuitability
		result.Types[i] = determineCoastalPortType(coastal, harbor, estuary, stopover, suitability, mode)
	}

	if network != nil {
		populateBaseCoastalNodeScores(cells, network, elevation, seaLevel, result.Diagnostics, settings)
		result.MajorPorts = identifyMajorCoastalPorts(network, trade, riverTrade, result.Diagnostics, settings)
		result.MajorDeepwaterPorts = identifyMajorDeepwaterPorts(network, trade, riverTrade, result.Diagnostics, settings)
	}
	return result
}

func meanSeasonalSurfaceWind(climate *SeasonalClimateResult) []float64 {
	if climate == nil || len(climate.Snapshots) == 0 {
		return nil
	}
	n := len(climate.Snapshots[0].SurfaceWind)
	out := make([]float64, n)
	for _, snap := range climate.Snapshots {
		for i, v := range snap.SurfaceWind {
			out[i] += Length(v)
		}
	}
	scale := 1.0 / float64(len(climate.Snapshots))
	for i := range out {
		out[i] *= scale
	}
	return out
}

func meanCurrentSpeed(climate *SeasonalClimateResult) []float64 {
	if climate == nil || len(climate.Currents) == 0 {
		return nil
	}
	out := make([]float64, len(climate.Currents))
	for i, v := range climate.Currents {
		out[i] = Length(v)
	}
	return out
}

func deriveHarborShelter(i int, adj *FlatAdjacency, elevation []float64, seaLevel, coastal float64, windSpeed, currentSpeed []float64) float64 {
	if coastal <= 0 {
		return 0
	}
	oceanFrac, meanWind, meanCurrent, landFrac := coastalNeighborStats(i, adj, elevation, seaLevel, windSpeed, currentSpeed)
	geometryShelter := 1 - smoothstep01(0.24, 0.72, oceanFrac)
	currentShelter := 1 - smoothstep01(0.02, 0.18, meanCurrent)
	windShelter := 1 - smoothstep01(0.04, 0.26, meanWind)
	return clamp01(coastal * (0.44*geometryShelter + 0.28*windShelter + 0.18*currentShelter + 0.10*landFrac))
}

func derivePortEstuaryAccess(i int, hydro *HydrologyBiomeInputs, soils *SoilResult, coastalResources *CoastalResourceResult, coastal float64) float64 {
	if coastal <= 0 {
		return 0
	}
	runoff := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
	channel := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.ChannelStrength })
	alluvial := 0.0
	if soils != nil && soils.Diagnostics != nil && i < len(soils.Diagnostics.Alluvial) {
		alluvial = soils.Diagnostics.Alluvial[i]
	}
	estuary := 0.0
	if coastalResources != nil && coastalResources.Diagnostics != nil && i < len(coastalResources.Diagnostics.EstuarineFishery) {
		estuary = coastalResources.Diagnostics.EstuarineFishery[i]
	}
	return clamp01(coastal * (0.34*alluvial + 0.28*smoothstep01(8, 105, runoff) + 0.20*smoothstep01(0.8, 2.2, channel) + 0.18*estuary))
}

func derivePortRiverTransfer(i int, hydro *HydrologyBiomeInputs, riverRoutes *RiverRouteResult, coastal float64) float64 {
	if coastal <= 0 {
		return 0
	}
	if riverRoutes != nil && riverRoutes.Diagnostics != nil &&
		i < len(riverRoutes.Diagnostics.Navigability) &&
		i < len(riverRoutes.Diagnostics.TransferSupport) {
		return clamp01(coastal * (0.56*riverRoutes.Diagnostics.Navigability[i] + 0.44*riverRoutes.Diagnostics.TransferSupport[i]))
	}
	runoff := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
	channel := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.ChannelStrength })
	return clamp01(coastal * (0.52*smoothstep01(8, 105, runoff) + 0.48*smoothstep01(0.8, 2.2, channel)))
}

func deriveStopoverValue(i int, adj *FlatAdjacency, elevation []float64, seaLevel, coastal, harbor float64, coastalResources *CoastalResourceResult) float64 {
	if coastal <= 0 {
		return 0
	}
	oceanFrac, _, _, landFrac := coastalNeighborStats(i, adj, elevation, seaLevel, nil, nil)
	islandGeom := 1 - smoothstep01(0.40, 0.86, landFrac)
	food := 0.0
	if coastalResources != nil && coastalResources.Diagnostics != nil {
		if i < len(coastalResources.Diagnostics.OpenFishery) {
			food += 0.55 * coastalResources.Diagnostics.OpenFishery[i]
		}
		if i < len(coastalResources.Diagnostics.ShellfishPotential) {
			food += 0.45 * coastalResources.Diagnostics.ShellfishPotential[i]
		}
	}
	return clamp01(coastal * (0.42*islandGeom + 0.24*harbor + 0.20*food + 0.14*oceanFrac))
}

func deriveCoastalStormExposure(i int, adj *FlatAdjacency, elevation []float64, seaLevel, coastal float64, windSpeed, currentSpeed []float64) float64 {
	if coastal <= 0 {
		return 0
	}
	oceanFrac, meanWind, meanCurrent, _ := coastalNeighborStats(i, adj, elevation, seaLevel, windSpeed, currentSpeed)
	return clamp01(coastal * (0.46*oceanFrac + 0.34*smoothstep01(0.04, 0.26, meanWind) + 0.20*smoothstep01(0.02, 0.18, meanCurrent)))
}

func coastalNeighborStats(i int, adj *FlatAdjacency, elevation []float64, seaLevel float64, windSpeed, currentSpeed []float64) (oceanFrac, meanWind, meanCurrent, landFrac float64) {
	neighbors := adj.GetNeighbors(i)
	if len(neighbors) == 0 {
		return 0, 0, 0, 0
	}
	ocean := 0.0
	land := 0.0
	for _, k := range neighbors {
		if k < 0 || k >= len(elevation) {
			continue
		}
		if elevation[k] < seaLevel {
			ocean++
			if k < len(windSpeed) {
				meanWind += windSpeed[k]
			}
			if k < len(currentSpeed) {
				meanCurrent += currentSpeed[k]
			}
		} else {
			land++
		}
	}
	total := ocean + land
	if total <= 0 {
		return 0, 0, 0, 0
	}
	if ocean > 0 {
		meanWind /= ocean
		meanCurrent /= ocean
	}
	return ocean / total, meanWind, meanCurrent, land / total
}

func derivePortSuitability(harbor, estuary, transfer, stopover, storm float64, vessel MaritimeVesselSettings, settings MaritimePortSettings) float64 {
	deepDraft := 1 - vessel.ShallowDraft
	beachable := math.Max(vessel.BeachingCapability, vessel.ShallowDraft)
	base := 0.16*math.Max(math.Max(harbor, estuary), math.Max(transfer, stopover)) +
		settings.HarborShelterWeight*harbor*(0.45+0.55*vessel.HarborDependence) +
		settings.EstuaryWeight*estuary*(0.35+0.65*vessel.ShallowDraft) +
		settings.RiverTransferWeight*transfer*(0.20+0.80*vessel.RiverCapability) +
		settings.StopoverWeight*stopover*(0.25+0.75*vessel.StopoverNeed) +
		settings.BeachingWeight*(1-harbor)*vessel.BeachingCapability +
		settings.ShallowDraftWeight*estuary*vessel.ShallowDraft +
		settings.DeepDraftHarborWeight*harbor*deepDraft*(0.35+0.65*vessel.HarborDependence) +
		settings.DeepDraftEstuaryWeight*estuary*deepDraft*(0.30+0.70*vessel.HarborDependence) +
		settings.BeachLandingWeight*stopover*beachable*(0.35+0.65*(1-vessel.HarborDependence))
	penalty := settings.ExposurePenalty*(1-harbor)*(0.30+0.70*vessel.HarborDependence) +
		settings.StormPenalty*storm*(1-vessel.StormTolerance) +
		settings.DeepDraftExposurePenalty*deepDraft*(1-harbor)*(1-estuary)
	return clamp01(1.55 * (base - 0.55*penalty))
}

func deriveDeepwaterAccess(i int, adj *FlatAdjacency, elevation []float64, seaLevel, coastal float64) float64 {
	if coastal <= 0 {
		return 0
	}
	neighbors := adj.GetNeighbors(i)
	if len(neighbors) == 0 {
		return 0
	}
	ocean := 0.0
	depth := 0.0
	for _, neighbor := range neighbors {
		if neighbor < 0 || neighbor >= len(elevation) || elevation[neighbor] >= seaLevel {
			continue
		}
		ocean++
		depth += clamp01((seaLevel - elevation[neighbor]) / 2600)
	}
	if ocean == 0 {
		return 0
	}
	oceanFrac := ocean / float64(len(neighbors))
	meanDepth := depth / ocean
	return clamp01(coastal * (0.56*meanDepth + 0.44*smoothstep01(0.28, 0.78, oceanFrac)))
}

func deriveDeepwaterSuitability(deepwater, harbor, estuary, transfer, storm float64, vessel MaritimeVesselSettings, settings MaritimePortSettings) float64 {
	if deepwater <= 0 {
		return 0
	}
	deepDraft := 1 - vessel.ShallowDraft
	bluewaterFit := 0.35 + 0.65*math.Max(vessel.OpenOceanCapability, deepDraft)
	base := settings.DeepwaterAccessWeight*deepwater*bluewaterFit +
		settings.DeepwaterHarborWeight*harbor*(0.35+0.65*vessel.HarborDependence) +
		settings.DeepwaterEstuaryWeight*estuary*(0.45+0.55*vessel.HarborDependence) +
		settings.DeepwaterTransferWeight*transfer*(0.30+0.70*vessel.LongHaulTolerance)
	penalty := settings.DeepwaterStormPenalty * storm * (1 - vessel.StormTolerance)
	return clamp01(1.45 * (base - 0.55*penalty))
}

func determineCoastalPortType(coastal, harbor, estuary, stopover, suitability float64, vessel MaritimeVesselSettings) CoastalPortType {
	if coastal < 0.14 || suitability < 0.18 {
		return CoastalPortNone
	}
	landing := clamp01(0.30*coastal + 0.26*vessel.BeachingCapability + 0.18*vessel.ShallowDraft + 0.16*(1-harbor))
	candidates := []struct {
		typ CoastalPortType
		val float64
	}{
		{CoastalPortBeachLanding, landing},
		{CoastalPortHarbor, clamp01(0.62*suitability + 0.38*harbor)},
		{CoastalPortEstuary, clamp01(0.68*estuary + 0.18*suitability + 0.14*(0.35+0.65*vessel.ShallowDraft))},
		{CoastalPortIslandStopover, clamp01(0.58*stopover + 0.22*suitability + 0.20*(1-vessel.StopoverNeed))},
	}
	best := CoastalPortNone
	bestVal := 0.0
	for _, c := range candidates {
		if c.val > bestVal {
			bestVal = c.val
			best = c.typ
		}
	}
	if bestVal < 0.28 {
		return CoastalPortNone
	}
	return best
}
