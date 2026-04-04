package climgen

type SettlementClass int

const (
	SettlementOcean SettlementClass = iota
	SettlementUnsuitable
	SettlementMarginal
	SettlementFavorable
	SettlementPrime
)

func SettlementClassName(c SettlementClass) string {
	names := []string{
		"Ocean",
		"Unsuitable",
		"Marginal",
		"Favorable",
		"Prime",
	}
	if int(c) < len(names) {
		return names[c]
	}
	return "Unknown"
}

type SettlementDiagnostics struct {
	ClimateScore  []float64
	WaterScore    []float64
	TerrainScore  []float64
	SoilScore     []float64
	AccessScore   []float64
	ResourceScore []float64
	HazardPenalty []float64
	RiverBonus    []float64
	CoastalBonus  []float64
	Suitability   []float64
}

type SettlementResult struct {
	Classes     []SettlementClass
	Diagnostics *SettlementDiagnostics
}

func ClassifySettlementSuitability(
	climate *SeasonalClimateResult,
	biomes *BiomeResult,
	soils *SoilResult,
	vegetation *VegetationResult,
	waterResources *WaterResourceResult,
	resources *ResourceResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	coastalExposure []float64,
) *SettlementResult {
	_ = climate
	n := len(elevation)
	out := &SettlementResult{
		Classes: make([]SettlementClass, n),
		Diagnostics: &SettlementDiagnostics{
			ClimateScore:  make([]float64, n),
			WaterScore:    make([]float64, n),
			TerrainScore:  make([]float64, n),
			SoilScore:     make([]float64, n),
			AccessScore:   make([]float64, n),
			ResourceScore: make([]float64, n),
			HazardPenalty: make([]float64, n),
			RiverBonus:    make([]float64, n),
			CoastalBonus:  make([]float64, n),
			Suitability:   make([]float64, n),
		},
	}
	if biomes == nil || biomes.Diagnostics == nil || soils == nil || soils.Diagnostics == nil {
		return out
	}
	bdiag := biomes.Diagnostics
	sdiag := soils.Diagnostics
	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Classes[i] = SettlementOcean
			continue
		}

		warmth := peak01(bdiag.AnnualMeanTempC[i], 5, 14, 26)
		growing := smoothstep01(10, 24, bdiag.WarmestSeasonTempC[i])
		dryness := smoothstep01(0.40, 1.70, bdiag.AridityRatio[i])
		precip := peak01(bdiag.AnnualPrecipCm[i], 20, 75, 180)
		out.Diagnostics.ClimateScore[i] = clamp01(0.35*warmth + 0.25*growing + 0.20*dryness + 0.20*precip)

		runoff := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
		channel := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.ChannelStrength })
		waterlogging := 0.0
		if vegetation != nil && vegetation.Diagnostics != nil && i < len(vegetation.Diagnostics.Waterlogging) {
			waterlogging = vegetation.Diagnostics.Waterlogging[i]
		}
		waterResourceScore := settlementWaterResourceScore(waterResources, i)
		riverBonus := clamp01(0.55*smoothstep01(0.50, 2.20, channel) + 0.45*smoothstep01(10, 95, runoff))
		out.Diagnostics.RiverBonus[i] = riverBonus
		out.Diagnostics.WaterScore[i] = clamp01(
			0.28*peak01(runoff, 6, 28, 95) +
				0.26*riverBonus +
				0.18*peak01(bdiag.AnnualPrecipCm[i], 30, 90, 200) +
				0.28*waterResourceScore -
				0.14*smoothstep01(0.72, 1.00, waterlogging),
		)

		coastal := coastalValue(coastalExposure, i)
		out.Diagnostics.CoastalBonus[i] = coastal * (1 - 0.45*waterlogging)

		relief := sdiag.Relief[i]
		rock := sdiag.Rockiness[i]
		elevPenalty := smoothstep01(1600, 3800, elevation[i])
		out.Diagnostics.TerrainScore[i] = clamp01(
			0.45*(1-smoothstep01(140, 1200, relief)) +
				0.35*(1-smoothstep01(0.25, 0.85, rock)) +
				0.20*(1-elevPenalty),
		)

		out.Diagnostics.SoilScore[i] = clamp01(
			0.55*sdiag.Fertility[i] +
				0.20*sdiag.Alluvial[i] +
				0.15*(1-sdiag.Salinity[i]) +
				0.10*sdiag.Drainage[i],
		)

		out.Diagnostics.AccessScore[i] = clamp01(
			0.36*out.Diagnostics.CoastalBonus[i] +
				0.28*riverBonus +
				0.18*sdiag.Alluvial[i] +
				0.18*waterResourceScore,
		)

		out.Diagnostics.ResourceScore[i] = settlementResourceScore(resources, i)

		iceHazard := smoothstep01(0.10, 0.85, bdiag.AnnualIceFraction[i])
		coldHazard := smoothstep01(8, -12, bdiag.WarmestSeasonTempC[i])
		floodHazard := smoothstep01(0.80, 1.00, waterlogging)
		aridHazard := smoothstep01(0.30, 0.08, bdiag.DrySeasonRatio[i]) * smoothstep01(0.70, 0.25, bdiag.AridityRatio[i])
		out.Diagnostics.HazardPenalty[i] = clamp01(0.35*iceHazard + 0.25*coldHazard + 0.15*floodHazard + 0.15*aridHazard)

		score := clamp01(
			0.23*out.Diagnostics.ClimateScore[i] +
				0.21*out.Diagnostics.WaterScore[i] +
				0.17*out.Diagnostics.TerrainScore[i] +
				0.17*out.Diagnostics.SoilScore[i] +
				0.13*out.Diagnostics.AccessScore[i] +
				0.09*out.Diagnostics.ResourceScore[i] -
				0.24*out.Diagnostics.HazardPenalty[i],
		)

		score += 0.05 * clamp01(0.55*sdiag.Alluvial[i]+0.45*riverBonus)
		score += 0.03 * out.Diagnostics.CoastalBonus[i]

		if vegetation != nil && vegetation.Diagnostics != nil {
			if i < len(vegetation.Diagnostics.BareCover) {
				score -= 0.10 * vegetation.Diagnostics.BareCover[i]
			}
			if i < len(vegetation.Diagnostics.TreeCover) && i < len(vegetation.Types) {
				switch vegetation.Types[i] {
				case VegetationDesertSparse, VegetationIceBarren:
					score -= 0.08
				case VegetationGrassland, VegetationWoodland, VegetationRiparianForest:
					score += 0.05
				case VegetationWetland, VegetationSaltMarsh, VegetationPeatland:
					score -= 0.04
				}
			}
		}

		out.Diagnostics.Suitability[i] = clamp01(score)
		out.Classes[i] = classifySettlementClass(out.Diagnostics.Suitability[i])
	}
	return out
}

func classifySettlementClass(score float64) SettlementClass {
	switch {
	case score >= 0.66:
		return SettlementPrime
	case score >= 0.42:
		return SettlementFavorable
	case score >= 0.22:
		return SettlementMarginal
	default:
		return SettlementUnsuitable
	}
}

func settlementResourceScore(resources *ResourceResult, idx int) float64 {
	if resources == nil || resources.Diagnostics == nil || idx < 0 || idx >= len(resources.Types) {
		return 0
	}
	score := 0.0
	if idx < len(resources.Diagnostics.PlacerAffinity) {
		score += 0.35 * resources.Diagnostics.PlacerAffinity[idx]
	}
	if idx < len(resources.Diagnostics.StoneAffinity) {
		score += 0.20 * resources.Diagnostics.StoneAffinity[idx]
	}
	if idx < len(resources.Diagnostics.IronAffinity) {
		score += 0.08 * resources.Diagnostics.IronAffinity[idx]
	}
	if idx < len(resources.Diagnostics.CopperAffinity) {
		score += 0.07 * resources.Diagnostics.CopperAffinity[idx]
	}
	if idx < len(resources.Diagnostics.LeadSilverAffinity) {
		score += 0.05 * resources.Diagnostics.LeadSilverAffinity[idx]
	}
	if idx < len(resources.Diagnostics.GoldAffinity) {
		score += 0.04 * resources.Diagnostics.GoldAffinity[idx]
	}
	if idx < len(resources.Diagnostics.GemAffinity) {
		score += 0.03 * resources.Diagnostics.GemAffinity[idx]
	}
	if idx < len(resources.Diagnostics.CoalAffinity) {
		score += 0.06 * resources.Diagnostics.CoalAffinity[idx]
	}
	if idx < len(resources.Diagnostics.OilGasAffinity) {
		score += 0.04 * resources.Diagnostics.OilGasAffinity[idx]
	}
	switch resources.Types[idx] {
	case ResourcePlacerAlluvial:
		score += 0.20
	case ResourceIndustrialStone:
		score += 0.16
	case ResourceIronOre, ResourceCopperOre, ResourceLeadSilverOre, ResourceGoldOre, ResourceGemstones:
		score += 0.12
	case ResourceCoal, ResourceOilGas:
		score += 0.08
	case ResourceEvaporite:
		score += 0.04
	}
	return clamp01(score)
}

func settlementWaterResourceScore(waterResources *WaterResourceResult, idx int) float64 {
	if waterResources == nil || waterResources.Diagnostics == nil || idx < 0 || idx >= len(waterResources.Types) {
		return 0
	}
	score := 0.0
	if idx < len(waterResources.Diagnostics.SurfaceReliability) {
		score += 0.42 * waterResources.Diagnostics.SurfaceReliability[idx]
	}
	if idx < len(waterResources.Diagnostics.SeasonalAvailability) {
		score += 0.20 * waterResources.Diagnostics.SeasonalAvailability[idx]
	}
	if idx < len(waterResources.Diagnostics.GroundwaterPotential) {
		score += 0.26 * waterResources.Diagnostics.GroundwaterPotential[idx]
	}
	if idx < len(waterResources.Diagnostics.LakeAccess) {
		score += 0.22 * waterResources.Diagnostics.LakeAccess[idx]
	}
	switch waterResources.Types[idx] {
	case WaterResourceReliableSurface:
		score += 0.18
	case WaterResourceGroundwater:
		score += 0.14
	case WaterResourceLakeOasis:
		score += 0.12
	case WaterResourceSeasonal:
		score += 0.08
	}
	return clamp01(score)
}
