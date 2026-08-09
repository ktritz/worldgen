package climgen

type ResourceType int

const (
	ResourceOcean ResourceType = iota
	ResourceNone
	ResourceIronOre
	ResourceCopperOre
	ResourceLeadSilverOre
	ResourceGoldOre
	ResourceGemstones
	ResourcePlacerAlluvial
	ResourceCoal
	ResourceOilGas
	ResourceEvaporite
	ResourceClayAggregate
	ResourceIndustrialStone
)

func ResourceName(r ResourceType) string {
	names := []string{
		"Ocean",
		"None",
		"Iron Ore",
		"Copper Ore",
		"Lead/Silver Ore",
		"Gold Ore",
		"Gemstones",
		"Placer Alluvial",
		"Coal",
		"Oil/Gas",
		"Evaporite",
		"Clay/Aggregate",
		"Industrial Stone",
	}
	if int(r) < len(names) {
		return names[r]
	}
	return "Unknown"
}

type ResourceGeologyInputs struct {
	HotspotStrength    []float64
	ContinentalHotspot []float64
}

type ResourceDiagnostics struct {
	IronAffinity       []float64
	CopperAffinity     []float64
	LeadSilverAffinity []float64
	GoldAffinity       []float64
	GemAffinity        []float64
	PlacerAffinity     []float64
	CoalAffinity       []float64
	OilGasAffinity     []float64
	EvaporiteAffinity  []float64
	ClayAffinity       []float64
	StoneAffinity      []float64
}

type ResourceResult struct {
	Types       []ResourceType
	Diagnostics *ResourceDiagnostics
}

func ClassifyResourcesWithSettings(
	climate *SeasonalClimateResult,
	biomes *BiomeResult,
	soils *SoilResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	geology *ResourceGeologyInputs,
	settings ResourceAbundanceSettings,
) *ResourceResult {
	return classifyResources(climate, biomes, soils, elevation, seaLevel, hydro, geology, settings)
}

func ClassifyResources(
	climate *SeasonalClimateResult,
	biomes *BiomeResult,
	soils *SoilResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	geology *ResourceGeologyInputs,
) *ResourceResult {
	return classifyResources(climate, biomes, soils, elevation, seaLevel, hydro, geology, DefaultResourceAbundanceSettings())
}

func classifyResources(
	climate *SeasonalClimateResult,
	biomes *BiomeResult,
	soils *SoilResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	geology *ResourceGeologyInputs,
	settings ResourceAbundanceSettings,
) *ResourceResult {
	n := len(elevation)
	out := &ResourceResult{
		Types: make([]ResourceType, n),
		Diagnostics: &ResourceDiagnostics{
			IronAffinity:       make([]float64, n),
			CopperAffinity:     make([]float64, n),
			LeadSilverAffinity: make([]float64, n),
			GoldAffinity:       make([]float64, n),
			GemAffinity:        make([]float64, n),
			PlacerAffinity:     make([]float64, n),
			CoalAffinity:       make([]float64, n),
			OilGasAffinity:     make([]float64, n),
			EvaporiteAffinity:  make([]float64, n),
			ClayAffinity:       make([]float64, n),
			StoneAffinity:      make([]float64, n),
		},
	}
	if biomes == nil || biomes.Diagnostics == nil || soils == nil || soils.Diagnostics == nil {
		return out
	}
	diag := biomes.Diagnostics
	sdiag := soils.Diagnostics
	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Types[i] = ResourceOcean
			continue
		}
		runoff := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
		channel := hydrologyChannelCorridorStrength(hydro, i)
		className := hydrologyClassName(hydro, i)
		classWet := hydrologyClassFactor(hydro, i)
		classDepositional := hydrologyDepositionalClassFactor(hydro, i)
		hotspot := geologyValue(geology, i, func(g *ResourceGeologyInputs) []float64 { return g.HotspotStrength })
		continentalHotspot := geologyValue(geology, i, func(g *ResourceGeologyInputs) []float64 { return g.ContinentalHotspot })

		lowRelief := 1 - smoothstep01(120, 700, sdiag.Relief[i])
		lowRock := 1 - smoothstep01(0.25, 0.75, sdiag.Rockiness[i])
		depositionalContext := clamp01(
			0.52*classDepositional +
				0.15*sdiag.Alluvial[i] +
				0.18*classWet,
		)
		volcanicBase := clamp01(
			(0.60*hotspot + 0.25*continentalHotspot + 0.15*smoothstep01(400, 2200, elevation[i])) *
				(0.55 + 0.45*sdiag.Rockiness[i]),
		)
		orogenicBase := clamp01(
			smoothstep01(700, 2600, elevation[i]) *
				smoothstep01(120, 900, sdiag.Relief[i]) *
				(0.55 + 0.45*sdiag.Rockiness[i]) *
				(1 - 0.45*diag.IceAffinity[i]),
		)
		hardRockMetalContext := clamp01(0.48*orogenicBase + 0.30*volcanicBase + 0.22*continentalHotspot)
		gemContext := clamp01(
			0.34*orogenicBase +
				0.18*volcanicBase +
				0.20*sdiag.Rockiness[i] +
				0.16*smoothstep01(700, 2400, elevation[i]) +
				0.12*smoothstep01(180, 850, sdiag.Relief[i]),
		)
		basinBase := clamp01(
			depositionalContext *
				(0.42 +
					0.22*lowRelief +
					0.16*lowRock +
					0.10*smoothstep01(0.70, 1.35, diag.AridityRatio[i]) +
					0.06*(1-smoothstep01(100, 900, elevation[i])) +
					0.04*smoothstep01(5, 80, runoff)) *
				(1 - 0.65*hotspot) *
				(1 - 0.35*orogenicBase),
		)

		out.Diagnostics.IronAffinity[i] = clamp01(
			0.70*orogenicBase +
				0.20*smoothstep01(500, 2200, elevation[i]) +
				0.10*sdiag.Rockiness[i],
		)
		out.Diagnostics.IronAffinity[i] = clamp01(out.Diagnostics.IronAffinity[i] * (1 - 0.16*hardRockMetalContext))
		out.Diagnostics.CopperAffinity[i] = clamp01(
			0.50*volcanicBase +
				0.25*orogenicBase +
				0.15*continentalHotspot +
				0.10*smoothstep01(600, 2000, elevation[i]),
		)
		out.Diagnostics.CopperAffinity[i] = clamp01(out.Diagnostics.CopperAffinity[i] * (1 - 0.10*gemContext))
		out.Diagnostics.LeadSilverAffinity[i] = clamp01(
			0.56*hardRockMetalContext +
				0.10*volcanicBase +
				0.08*continentalHotspot +
				0.12*(1-sdiag.Alluvial[i]) +
				0.10*smoothstep01(700, 2400, elevation[i]) +
				0.08*sdiag.Rockiness[i] +
				0.06*smoothstep01(0.6, 1.6, channel),
		)
		out.Diagnostics.GoldAffinity[i] = clamp01(
			0.52*hardRockMetalContext +
				0.10*volcanicBase +
				0.08*continentalHotspot +
				0.12*(1-sdiag.Alluvial[i]) +
				0.10*smoothstep01(900, 2800, elevation[i]) +
				0.10*smoothstep01(0.8, 2.0, channel)*(0.35+0.65*(1-sdiag.Alluvial[i])) +
				0.06*sdiag.Rockiness[i],
		)
		out.Diagnostics.GemAffinity[i] = clamp01(
			(0.40*gemContext +
				0.12*(1-sdiag.Alluvial[i]) +
				0.08*(1-smoothstep01(25, 140, runoff)) +
				0.06*(1-smoothstep01(0.7, 2.0, channel))) *
				smoothstep01(0.72, 0.96, sdiag.Rockiness[i]) *
				(0.65 + 0.35*smoothstep01(220, 850, sdiag.Relief[i])),
		)
		out.Diagnostics.PlacerAffinity[i] = clamp01(
			smoothstep01(0.7, 2.4, channel) *
				(0.55 + 0.45*sdiag.Alluvial[i]) *
				(0.50 + 0.50*smoothstep01(15, 120, runoff)) *
				(0.55 + 0.45*(0.40*out.Diagnostics.GoldAffinity[i]+0.25*out.Diagnostics.LeadSilverAffinity[i]+0.20*out.Diagnostics.CopperAffinity[i]+0.15*out.Diagnostics.GemAffinity[i])),
		)
		out.Diagnostics.CoalAffinity[i] = clamp01(
			basinBase *
				(0.42 +
					0.30*smoothstep01(45, 180, diag.AnnualPrecipCm[i]) +
					0.20*diag.ForestAffinity[i] +
					0.10*smoothstep01(2, 18, diag.AnnualMeanTempC[i]) +
					0.08*(1-smoothstep01(0.80, 1.80, diag.AridityRatio[i]))),
		)
		out.Diagnostics.OilGasAffinity[i] = clamp01(
			basinBase *
				(0.40 +
					0.20*lowRelief +
					0.15*(1-smoothstep01(0.35, 0.80, sdiag.Alluvial[i])) +
					0.15*smoothstep01(0.55, 1.25, diag.AridityRatio[i]) +
					0.10*smoothstep01(6, 26, diag.AnnualMeanTempC[i])),
		)
		out.Diagnostics.EvaporiteAffinity[i] = clamp01(
			(0.45*sdiag.Salinity[i] +
				0.20*smoothstep01(1.05, 0.35, diag.AridityRatio[i]) +
				0.20*bool01(classWet >= 0.8) +
				0.15*bool01(className == "lake" || className == "lake_complex")) *
				(1 - 0.35*sdiag.Drainage[i]),
		)
		out.Diagnostics.ClayAffinity[i] = clamp01(
			depositionalContext *
				(0.40 +
					0.32*sdiag.Alluvial[i] +
					0.16*lowRelief +
					0.12*smoothstep01(8, 60, runoff)),
		)
		out.Diagnostics.StoneAffinity[i] = clamp01(
			0.50*sdiag.Rockiness[i] +
				0.30*smoothstep01(120, 900, sdiag.Relief[i]) +
				0.20*smoothstep01(300, 1800, elevation[i]),
		)
		stoneSuppression := 1 - 0.45*clamp01(0.45*orogenicBase+0.30*volcanicBase+0.25*gemContext)
		out.Diagnostics.StoneAffinity[i] = clamp01(out.Diagnostics.StoneAffinity[i] * stoneSuppression)

		applyResourceAbundanceSettings(out.Diagnostics, i, settings)
		out.Types[i] = determineResourceType(resourceCandidates(out.Diagnostics, i, settings))
	}
	return out
}

type resourceCandidate struct {
	typ ResourceType
	val float64
}

func resourceCandidates(diag *ResourceDiagnostics, i int, settings ResourceAbundanceSettings) []resourceCandidate {
	return []resourceCandidate{
		{ResourceIronOre, clamp01(diag.IronAffinity[i] + settings.IronPrimaryBias)},
		{ResourceCopperOre, clamp01(diag.CopperAffinity[i] + settings.CopperPrimaryBias)},
		{ResourceLeadSilverOre, clamp01(diag.LeadSilverAffinity[i] + settings.LeadSilverPrimaryBias)},
		{ResourceGoldOre, clamp01(diag.GoldAffinity[i] + settings.GoldPrimaryBias)},
		{ResourceGemstones, clamp01(diag.GemAffinity[i] + settings.GemPrimaryBias)},
		{ResourcePlacerAlluvial, clamp01(diag.PlacerAffinity[i] + settings.PlacerPrimaryBias)},
		{ResourceCoal, clamp01(diag.CoalAffinity[i] + settings.CoalPrimaryBias)},
		{ResourceOilGas, clamp01(diag.OilGasAffinity[i] + settings.OilGasPrimaryBias)},
		{ResourceEvaporite, clamp01(diag.EvaporiteAffinity[i] + settings.EvaporitePrimaryBias)},
		{ResourceClayAggregate, clamp01(diag.ClayAffinity[i] + settings.ClayPrimaryBias)},
		{ResourceIndustrialStone, clamp01(diag.StoneAffinity[i] + settings.StonePrimaryBias)},
	}
}

func applyResourceAbundanceSettings(diag *ResourceDiagnostics, idx int, settings ResourceAbundanceSettings) {
	diag.IronAffinity[idx] = clamp01(diag.IronAffinity[idx] * settings.IronAffinityMultiplier)
	diag.CopperAffinity[idx] = clamp01(diag.CopperAffinity[idx] * settings.CopperAffinityMultiplier)
	diag.LeadSilverAffinity[idx] = clamp01(diag.LeadSilverAffinity[idx] * settings.LeadSilverAffinityMultiplier)
	diag.GoldAffinity[idx] = clamp01(diag.GoldAffinity[idx] * settings.GoldAffinityMultiplier)
	diag.GemAffinity[idx] = clamp01(diag.GemAffinity[idx] * settings.GemAffinityMultiplier)
	diag.PlacerAffinity[idx] = clamp01(diag.PlacerAffinity[idx] * settings.PlacerAffinityMultiplier)
	diag.CoalAffinity[idx] = clamp01(diag.CoalAffinity[idx] * settings.CoalAffinityMultiplier)
	diag.OilGasAffinity[idx] = clamp01(diag.OilGasAffinity[idx] * settings.OilGasAffinityMultiplier)
	diag.EvaporiteAffinity[idx] = clamp01(diag.EvaporiteAffinity[idx] * settings.EvaporiteAffinityMultiplier)
	diag.ClayAffinity[idx] = clamp01(diag.ClayAffinity[idx] * settings.ClayAffinityMultiplier)
	diag.StoneAffinity[idx] = clamp01(diag.StoneAffinity[idx] * settings.StoneAffinityMultiplier)
}

func determineResourceType(candidates []resourceCandidate) ResourceType {
	bestType := ResourceNone
	best := 0.0
	for _, c := range candidates {
		if c.val > best {
			best = c.val
			bestType = c.typ
		}
	}
	if best < 0.38 {
		return ResourceNone
	}
	return bestType
}

func geologyValue(geo *ResourceGeologyInputs, idx int, sel func(*ResourceGeologyInputs) []float64) float64 {
	if geo == nil {
		return 0
	}
	values := sel(geo)
	if idx < 0 || idx >= len(values) {
		return 0
	}
	return values[idx]
}

func hydrologyClassName(hydro *HydrologyBiomeInputs, idx int) string {
	if hydro == nil || idx < 0 || idx >= len(hydro.CellClass) {
		return ""
	}
	return hydro.CellClass[idx]
}
