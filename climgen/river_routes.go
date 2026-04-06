package climgen

import "math"

type RiverRouteDiagnostics struct {
	Navigability        []float64
	MainChannel         []float64
	FloodRisk           []float64
	TransferSupport     []float64
	PortageSuitability  []float64
	TowpathSupport      []float64
	UpstreamTravelCost  []float64
	DownstreamTravelCost []float64
}

type RiverRouteResult struct {
	Mode        RiverRouteModeSettings
	Diagnostics *RiverRouteDiagnostics
}

func BuildRiverRouteDiagnostics(
	settlements *SettlementResult,
	population *PopulationResult,
	soils *SoilResult,
	water *WaterResourceResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	settings RiverRouteSettings,
) *RiverRouteResult {
	n := len(elevation)
	mode, ok := settings.ModeByName(settings.DefaultMode)
	if !ok {
		mode = DefaultRiverRouteSettings().Modes[0]
	}
	out := &RiverRouteResult{
		Mode: mode,
		Diagnostics: &RiverRouteDiagnostics{
			Navigability:         make([]float64, n),
			MainChannel:          make([]float64, n),
			FloodRisk:            make([]float64, n),
			TransferSupport:      make([]float64, n),
			PortageSuitability:   make([]float64, n),
			TowpathSupport:       make([]float64, n),
			UpstreamTravelCost:   make([]float64, n),
			DownstreamTravelCost: make([]float64, n),
		},
	}
	if hydro == nil {
		return out
	}
	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Diagnostics.UpstreamTravelCost[i] = math.Inf(1)
			out.Diagnostics.DownstreamTravelCost[i] = math.Inf(1)
			continue
		}

		channel := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.ChannelStrength })
		runoff := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
		channelNav := smoothstep01(mode.MinChannelStrength, mode.MinChannelStrength+1.6, channel)
		runoffNav := smoothstep01(mode.MinRunoff, mode.MinRunoff+70.0, runoff)
		classBonus := riverClassNavigability(hydro, i)
		mainChannel := clamp01(0.68*channelNav + 0.22*runoffNav + 0.10*classBonus)
		navigability := clamp01(0.54*channelNav + 0.28*runoffNav + 0.18*classBonus)

		floodRisk := clamp01(
			0.52*smoothstep01(18, 110, runoff) +
				0.28*smoothstep01(mode.MinChannelStrength, mode.MinChannelStrength+1.8, channel) +
				0.20*riverClassFloodRisk(hydro, i),
		)

		waterSupport := 0.0
		if water != nil && water.Diagnostics != nil && i < len(water.Diagnostics.SurfaceReliability) {
			waterSupport = clamp01(
				0.50*water.Diagnostics.SurfaceReliability[i] +
					0.25*water.Diagnostics.SeasonalAvailability[i] +
					0.15*water.Diagnostics.GroundwaterPotential[i] +
					0.10*water.Diagnostics.LakeAccess[i],
			)
		} else {
			waterSupport = smoothstep01(8, 50, runoff)
		}

		access := 0.0
		if settlements != nil && settlements.Diagnostics != nil {
			if i < len(settlements.Diagnostics.AccessScore) {
				access = settlements.Diagnostics.AccessScore[i]
			}
			if i < len(settlements.Diagnostics.RiverBonus) {
				access = clamp01(0.72*access + 0.28*settlements.Diagnostics.RiverBonus[i])
			}
		}
		carrying := 0.0
		if population != nil && population.Diagnostics != nil && i < len(population.Diagnostics.CarryingCapacity) {
			carrying = population.Diagnostics.CarryingCapacity[i]
		}
		transfer := clamp01(0.36*access + 0.28*waterSupport + 0.22*carrying + 0.14*classBonus)
		relief := 0.0
		if soils != nil && soils.Diagnostics != nil && i < len(soils.Diagnostics.Relief) {
			relief = soils.Diagnostics.Relief[i]
		}
		flatness := 1 - smoothstep01(mode.TowpathReliefLimit*0.55, mode.TowpathReliefLimit, relief)
		towpath := clamp01(
			flatness *
				(0.54*transfer + 0.18*access + 0.16*riverClassTowpathSupport(hydro, i) + 0.12*(1-floodRisk)),
		)
		portage := clamp01(
			(1-navigability) *
				(0.52*transfer + 0.28*waterSupport + 0.20*(1-floodRisk)),
		)

		if navigability < mode.MinNavigability {
			navigability = 0
			mainChannel = 0
		}

		downstream := math.Inf(1)
		upstream := math.Inf(1)
		if navigability > 0 {
			base := 1.0 / math.Max(0.18+0.82*navigability, 0.1)
			downstream = math.Max(0.22, base*(1-0.45*mode.DownstreamBonus)*(1+0.25*floodRisk*(1-mode.FloodTolerance)))
			upstream = math.Max(0.28, base*(1+mode.UpstreamPenalty)*(1+0.18*floodRisk*(1-mode.FloodTolerance)))
			upstream *= math.Max(0.58, 1-mode.TowpathBenefit*towpath)
			if isLakeLikeRiverCell(hydro, i) {
				lakeFactor := math.Max(0.65, 1-0.45*mode.LakeBonus)
				downstream *= lakeFactor
				upstream *= lakeFactor
			}
		}

		out.Diagnostics.Navigability[i] = navigability
		out.Diagnostics.MainChannel[i] = mainChannel
		out.Diagnostics.FloodRisk[i] = floodRisk
		out.Diagnostics.TransferSupport[i] = transfer
		out.Diagnostics.PortageSuitability[i] = portage
		out.Diagnostics.TowpathSupport[i] = towpath
		out.Diagnostics.UpstreamTravelCost[i] = upstream
		out.Diagnostics.DownstreamTravelCost[i] = downstream
	}
	return out
}

func riverClassNavigability(hydro *HydrologyBiomeInputs, idx int) float64 {
	if hydro == nil || idx < 0 || idx >= len(hydro.CellClass) {
		return 0
	}
	switch hydro.CellClass[idx] {
	case "confluence":
		return 0.92
	case "delta":
		return 0.78
	case "lake_reach", "lake_complex":
		return 0.72
	case "coast_outlet":
		return 0.66
	case "floodplain":
		return 0.36
	default:
		return 0
	}
}

func riverClassFloodRisk(hydro *HydrologyBiomeInputs, idx int) float64 {
	if hydro == nil || idx < 0 || idx >= len(hydro.CellClass) {
		return 0
	}
	switch hydro.CellClass[idx] {
	case "floodplain":
		return 0.92
	case "delta":
		return 0.82
	case "coast_outlet":
		return 0.52
	case "confluence":
		return 0.48
	case "lake_reach", "lake_complex":
		return 0.18
	default:
		return 0
	}
}

func riverClassTowpathSupport(hydro *HydrologyBiomeInputs, idx int) float64 {
	if hydro == nil || idx < 0 || idx >= len(hydro.CellClass) {
		return 0
	}
	switch hydro.CellClass[idx] {
	case "floodplain":
		return 0.86
	case "lake_reach", "lake_complex":
		return 0.72
	case "coast_outlet":
		return 0.64
	case "confluence":
		return 0.44
	default:
		return 0
	}
}

func isLakeLikeRiverCell(hydro *HydrologyBiomeInputs, idx int) bool {
	if hydro == nil || idx < 0 || idx >= len(hydro.CellClass) {
		return false
	}
	switch hydro.CellClass[idx] {
	case "lake_reach", "lake_complex":
		return true
	default:
		return false
	}
}
