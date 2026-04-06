package climgen

import "math"

type LandRouteDiagnostics struct {
	BaseCost              []float64
	ModeCost              []float64
	RouteRisk             []float64
	WaterSupport          []float64
	ForageSupport         []float64
	WaystationSuitability []float64
	RoadQuality           []float64
	CrossingPressure      []float64
	BridgeProxy           []float64
	FordProxy             []float64
}

type LandRouteResult struct {
	Mode        LandRouteModeSettings
	Diagnostics *LandRouteDiagnostics
}

func BuildLandRouteDiagnostics(
	settlements *SettlementResult,
	population *PopulationResult,
	biomes *BiomeResult,
	vegetation *VegetationResult,
	soils *SoilResult,
	wildlife *WildlifeResult,
	water *WaterResourceResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	settings LandRouteSettings,
) *LandRouteResult {
	n := len(elevation)
	mode, ok := settings.ModeByName(settings.DefaultMode)
	if !ok {
		mode = DefaultLandRouteSettings().Modes[0]
	}
	out := &LandRouteResult{
		Mode: mode,
		Diagnostics: &LandRouteDiagnostics{
			BaseCost:              make([]float64, n),
			ModeCost:              make([]float64, n),
			RouteRisk:             make([]float64, n),
			WaterSupport:          make([]float64, n),
			ForageSupport:         make([]float64, n),
			WaystationSuitability: make([]float64, n),
			RoadQuality:           make([]float64, n),
			CrossingPressure:      make([]float64, n),
			BridgeProxy:           make([]float64, n),
			FordProxy:             make([]float64, n),
		},
	}
	if biomes == nil || biomes.Diagnostics == nil {
		return out
	}
	diag := biomes.Diagnostics
	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Diagnostics.BaseCost[i] = math.Inf(1)
			out.Diagnostics.ModeCost[i] = math.Inf(1)
			continue
		}

		relief := 0.0
		rockiness := 0.0
		if soils != nil && soils.Diagnostics != nil {
			if i < len(soils.Diagnostics.Relief) {
				relief = smoothstep01(120, 1400, soils.Diagnostics.Relief[i])
			}
			if i < len(soils.Diagnostics.Rockiness) {
				rockiness = soils.Diagnostics.Rockiness[i]
			}
		}

		wetland := diag.WetlandAffinity[i]
		forest := 0.0
		if vegetation != nil && vegetation.Diagnostics != nil {
			forest = clamp01(0.70*vegetation.Diagnostics.TreeCover[i] + 0.30*vegetation.Diagnostics.ShrubCover[i])
			wetland = clamp01(0.55*wetland + 0.45*vegetation.Diagnostics.WetlandCover[i])
		}
		aridity := smoothstep01(0.22, 0.98, diag.AridityRatio[i])
		ice := diag.AnnualIceFraction[i]
		riverBonus := 0.0
		coastalBonus := 0.0
		accessScore := 0.0
		settlementSuitability := 0.0
		if settlements != nil && settlements.Diagnostics != nil {
			if i < len(settlements.Diagnostics.RiverBonus) {
				riverBonus = settlements.Diagnostics.RiverBonus[i]
			}
			if i < len(settlements.Diagnostics.CoastalBonus) {
				coastalBonus = settlements.Diagnostics.CoastalBonus[i]
			}
			if i < len(settlements.Diagnostics.AccessScore) {
				accessScore = settlements.Diagnostics.AccessScore[i]
			}
			if i < len(settlements.Diagnostics.Suitability) {
				settlementSuitability = settlements.Diagnostics.Suitability[i]
			}
		}
		carrying := 0.0
		urban := 0.0
		if population != nil && population.Diagnostics != nil {
			if i < len(population.Diagnostics.CarryingCapacity) {
				carrying = population.Diagnostics.CarryingCapacity[i]
			}
			if i < len(population.Diagnostics.UrbanPotential) {
				urban = population.Diagnostics.UrbanPotential[i]
			}
		}

		baseCost := 1.0 + 1.10*relief + 0.65*rockiness + 0.60*wetland + 0.45*aridity + 1.10*ice + 0.30*forest
		baseCost -= 0.18*riverBonus + 0.12*coastalBonus
		baseCost = math.Max(0.35, baseCost)

		roadQuality := clamp01(
			0.30*settlementSuitability +
				0.24*carrying +
				0.18*urban +
				0.18*accessScore +
				0.10*clamp01(0.55*riverBonus+0.45*coastalBonus),
		)
		crossingPressure := clamp01(
			0.52*smoothstep01(0.85, 2.60, hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.ChannelStrength })) +
				0.28*hydrologyClassCrossingPressure(hydro, i) +
				0.20*smoothstep01(16, 110, hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })),
		)
		bridgeProxy := clamp01(
			crossingPressure *
				(0.55*roadQuality + 0.25*urban + 0.20*accessScore),
		)
		fordProxy := clamp01(
			crossingPressure *
				(1 - 0.65*wetland) *
				(1 - 0.45*hydrologyClassRouteRisk(hydro, i)) *
				(0.35 + 0.65*(1-smoothstep01(0.9, 2.4, hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.ChannelStrength })))),
		)

		slopePenalty := 0.0
		if relief > mode.SlopeLimit {
			slopePenalty = smoothstep01(mode.SlopeLimit, 1.0, relief)
		}
		modeCost := mode.BaseCostMultiplier * baseCost
		modeCost += mode.ReliefMultiplier * relief
		modeCost += mode.RockinessMultiplier * rockiness
		modeCost += mode.ReliefMultiplier * slopePenalty
		modeCost += mode.WetlandMultiplier * wetland * (1 - 0.65*mode.MarshPassability - 0.35*mode.MarshAdaptation)
		modeCost += mode.AridityMultiplier * aridity * (1 - mode.AridAdaptation)
		modeCost += mode.ForestMultiplier * forest
		modeCost += mode.IceMultiplier * ice * (1 - 0.60*mode.SnowPassability - 0.40*mode.ColdTolerance)
		modeCost -= 0.18 * mode.RiverBonusMultiplier * riverBonus
		modeCost -= 0.10 * mode.CoastalBonusMultiplier * coastalBonus
		modeCost += 0.34 * mode.RoadDependence * (1 - roadQuality)
		modeCost += 0.30 * mode.BridgeDependence * crossingPressure * (1 - bridgeProxy)
		modeCost += 0.26 * (1 - mode.FordTolerance) * crossingPressure * (1 - fordProxy)
		modeCost /= math.Max(mode.SpeedMultiplier, 0.25)
		modeCost = math.Max(0.30, modeCost)

		waterSupport := 0.0
		if water != nil && water.Diagnostics != nil {
			waterSupport = clamp01(
				0.45*water.Diagnostics.SurfaceReliability[i] +
					0.25*water.Diagnostics.SeasonalAvailability[i] +
					0.20*water.Diagnostics.GroundwaterPotential[i] +
					0.10*water.Diagnostics.LakeAccess[i],
			)
		} else {
			runoff := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
			waterSupport = smoothstep01(4, 60, runoff)
		}

		forageSupport := 0.0
		if vegetation != nil && vegetation.Diagnostics != nil {
			forageSupport = clamp01(
				0.40*vegetation.Diagnostics.GrassCover[i] +
					0.28*vegetation.Diagnostics.ShrubCover[i] +
					0.18*vegetation.Diagnostics.TreeCover[i] +
					0.14*diag.GrasslandAffinity[i],
			)
		}

		floodRisk := clamp01(
			0.50*smoothstep01(18, 110, hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })) +
				0.25*smoothstep01(0.9, 2.6, hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.ChannelStrength })) +
				0.25*hydrologyClassRouteRisk(hydro, i),
		)
		desertRisk := clamp01(aridity * (1 - waterSupport))
		marshRisk := clamp01(wetland * (1 - mode.MarshAdaptation))
		wildlifeRisk := 0.0
		if wildlife != nil && wildlife.Diagnostics != nil {
			wildlifeRisk = clamp01(
				0.45*wildlife.Diagnostics.GamePotential[i] +
					0.30*wildlife.Diagnostics.WetlandGamePotential[i] +
					0.25*wildlife.Diagnostics.ForestGamePotential[i],
			)
		}
		coldSeverity := clamp01(0.65*ice + 0.35*peak01(diag.WarmestSeasonTempC[i], -4, 4, 12))
		coldRisk := clamp01(coldSeverity * (1 - 0.55*mode.ColdTolerance - 0.45*mode.SnowPassability))
		heatRisk := clamp01(smoothstep01(20, 35, diag.WarmestSeasonTempC[i]) * (1 - mode.HeatTolerance))

		risk := clamp01(
			mode.FloodRiskMultiplier*floodRisk*(1-0.45*mode.SeasonalityTolerance)*0.24 +
				mode.DesertRiskMultiplier*desertRisk*0.24 +
				mode.MarshRiskMultiplier*marshRisk*(1-0.55*mode.MarshPassability)*0.18 +
				mode.WildlifeRiskMultiplier*wildlifeRisk*(0.70+0.30*mode.BanditVulnerability)*0.14 +
				mode.ColdRiskMultiplier*coldRisk*0.12 +
				heatRisk*0.08 +
				0.10*mode.BridgeDependence*crossingPressure*(1-bridgeProxy) +
				0.08*(1-mode.FordTolerance)*crossingPressure*(1-fordProxy),
		)

		supplyReadiness := clamp01(
			mode.WaterSupportWeight*waterSupport*(1-0.65*mode.WaterNeed)*0.60 +
				mode.ForageSupportWeight*forageSupport*(1-0.65*mode.ForageNeed)*0.40,
		)
		support := clamp01(supplyReadiness * (0.70 + 0.15*mode.EnduranceMultiplier + 0.15*mode.DailyRange))
		waystation := clamp01(
			support *
				(1 - 0.60*risk) *
				(1 - 0.25*smoothstep01(1.8, 4.5, modeCost)) *
				(0.76 + 0.12*mode.FordTolerance + 0.12*bridgeProxy),
		)

		out.Diagnostics.BaseCost[i] = baseCost
		out.Diagnostics.ModeCost[i] = modeCost
		out.Diagnostics.RouteRisk[i] = risk
		out.Diagnostics.WaterSupport[i] = waterSupport
		out.Diagnostics.ForageSupport[i] = forageSupport
		out.Diagnostics.WaystationSuitability[i] = waystation
		out.Diagnostics.RoadQuality[i] = roadQuality
		out.Diagnostics.CrossingPressure[i] = crossingPressure
		out.Diagnostics.BridgeProxy[i] = bridgeProxy
		out.Diagnostics.FordProxy[i] = fordProxy
	}
	return out
}

func hydrologyClassRouteRisk(hydro *HydrologyBiomeInputs, idx int) float64 {
	if hydro == nil || idx < 0 || idx >= len(hydro.CellClass) {
		return 0
	}
	switch hydro.CellClass[idx] {
	case "floodplain":
		return 0.85
	case "delta":
		return 0.75
	case "lake_reach":
		return 0.45
	case "coast_outlet":
		return 0.32
	case "confluence":
		return 0.28
	default:
		return 0
	}
}

func hydrologyClassCrossingPressure(hydro *HydrologyBiomeInputs, idx int) float64 {
	if hydro == nil || idx < 0 || idx >= len(hydro.CellClass) {
		return 0
	}
	switch hydro.CellClass[idx] {
	case "confluence":
		return 0.85
	case "floodplain":
		return 0.70
	case "delta":
		return 0.62
	case "lake_reach":
		return 0.40
	case "coast_outlet":
		return 0.35
	default:
		return 0
	}
}
