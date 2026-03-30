package climgen

import "math"

type VegetationType int

const (
	VegetationOcean VegetationType = iota
	VegetationIceBarren
	VegetationDesertSparse
	VegetationShrubland
	VegetationGrassland
	VegetationWoodland
	VegetationForest
	VegetationRainforest
	VegetationWetland
	VegetationMangrove
	VegetationSaltMarsh
	VegetationPeatland
	VegetationRiparianForest
	VegetationCloudForest
	VegetationAlpineMeadow
)

func VegetationName(v VegetationType) string {
	names := []string{
		"Ocean",
		"Ice/Barren",
		"Desert Sparse",
		"Shrubland",
		"Grassland",
		"Woodland",
		"Forest",
		"Rainforest",
		"Wetland",
		"Mangrove",
		"Salt Marsh",
		"Peatland",
		"Riparian Forest",
		"Cloud Forest",
		"Alpine Meadow",
	}
	if int(v) < len(names) {
		return names[v]
	}
	return "Unknown"
}

type VegetationDiagnostics struct {
	GrowingHeat          []float64
	MoistureAvailability []float64
	DryStress            []float64
	ColdStress           []float64
	Waterlogging         []float64
	CoastalExposure      []float64
	TreeCover            []float64
	GrassCover           []float64
	ShrubCover           []float64
	WetlandCover         []float64
	BareCover            []float64
	MangroveAffinity     []float64
	SaltMarshAffinity    []float64
	PeatlandAffinity     []float64
	RiparianAffinity     []float64
	CloudForestAffinity  []float64
	AlpineMeadowAffinity []float64
}

type VegetationResult struct {
	Types       []VegetationType
	Diagnostics *VegetationDiagnostics
}

func ComputeCoastalExposure(cells []VoronoiCell, elevation []float64, seaLevel float64) []float64 {
	exposure := make([]float64, len(elevation))
	for i, cell := range cells {
		if elevation[i] < seaLevel || len(cell.NeighborSiteIndices) == 0 {
			continue
		}
		direct := 0.0
		second := 0.0
		totalDirect := 0.0
		totalSecond := 0.0
		for _, n := range cell.NeighborSiteIndices {
			ni := int(n)
			if ni < 0 || ni >= len(elevation) {
				continue
			}
			totalDirect++
			if elevation[ni] < seaLevel {
				direct++
			}
			if ni >= 0 && ni < len(cells) {
				for _, n2 := range cells[ni].NeighborSiteIndices {
					n2i := int(n2)
					if n2i < 0 || n2i >= len(elevation) || n2i == i {
						continue
					}
					totalSecond++
					if elevation[n2i] < seaLevel {
						second++
					}
				}
			}
		}
		directFrac := 0.0
		if totalDirect > 0 {
			directFrac = direct / totalDirect
		}
		secondFrac := 0.0
		if totalSecond > 0 {
			secondFrac = second / totalSecond
		}
		exposure[i] = clamp01(0.75*directFrac + 0.25*secondFrac)
	}
	return exposure
}

func ClassifyVegetation(
	climate *SeasonalClimateResult,
	biomes *BiomeResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	coastalExposure []float64,
) *VegetationResult {
	n := len(elevation)
	out := &VegetationResult{
		Types: make([]VegetationType, n),
		Diagnostics: &VegetationDiagnostics{
			GrowingHeat:          make([]float64, n),
			MoistureAvailability: make([]float64, n),
			DryStress:            make([]float64, n),
			ColdStress:           make([]float64, n),
			Waterlogging:         make([]float64, n),
			CoastalExposure:      append([]float64(nil), coastalExposure...),
			TreeCover:            make([]float64, n),
			GrassCover:           make([]float64, n),
			ShrubCover:           make([]float64, n),
			WetlandCover:         make([]float64, n),
			BareCover:            make([]float64, n),
			MangroveAffinity:     make([]float64, n),
			SaltMarshAffinity:    make([]float64, n),
			PeatlandAffinity:     make([]float64, n),
			RiparianAffinity:     make([]float64, n),
			CloudForestAffinity:  make([]float64, n),
			AlpineMeadowAffinity: make([]float64, n),
		},
	}
	if biomes == nil || biomes.Diagnostics == nil {
		return out
	}
	diag := biomes.Diagnostics
	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Types[i] = VegetationOcean
			continue
		}
		out.Diagnostics.GrowingHeat[i] = smoothstep01(2, 24, diag.WarmestSeasonTempC[i])
		out.Diagnostics.MoistureAvailability[i] = clamp01(
			smoothstep01(20, 180, diag.AnnualPrecipCm[i]) *
				smoothstep01(0.40, 2.20, diag.AridityRatio[i]),
		)
		out.Diagnostics.DryStress[i] = smoothstep01(0.55, 0.05, diag.DrySeasonRatio[i])
		out.Diagnostics.ColdStress[i] = clamp01(math.Max(
			diag.IceAffinity[i],
			smoothstep01(12, -10, diag.WarmestSeasonTempC[i]),
		))

		waterlogRunoff := 0.0
		waterlogChannel := 0.0
		waterlogClass := 0.0
		if hydro != nil {
			if i < len(hydro.Runoff) {
				waterlogRunoff = smoothstep01(18, 110, hydro.Runoff[i])
			}
			if i < len(hydro.ChannelStrength) {
				waterlogChannel = smoothstep01(0.9, 2.6, hydro.ChannelStrength[i])
			}
			if i < len(hydro.CellClass) {
				switch hydro.CellClass[i] {
				case "floodplain":
					waterlogClass = 1.0
				case "delta":
					waterlogClass = 0.9
				case "lake_reach":
					waterlogClass = 0.8
				case "coast_outlet":
					waterlogClass = 0.55
				case "confluence":
					waterlogClass = 0.45
				}
			}
		}
		out.Diagnostics.Waterlogging[i] = clamp01(0.40*waterlogRunoff + 0.35*waterlogChannel + 0.25*waterlogClass)

		channel := 0.0
		if hydro != nil && i < len(hydro.ChannelStrength) {
			channel = hydro.ChannelStrength[i]
		}
		riparianSupport := clamp01(
			smoothstep01(0.7, 2.2, channel) *
				smoothstep01(0.25, 1.8, diag.AridityRatio[i]),
		)

		treePotential := clamp01(
			0.50*diag.ForestAffinity[i] +
				0.30*diag.TropicalWetAffinity[i] +
				0.18*diag.BorealAffinity[i] +
				0.20*riparianSupport,
		)
		grassPotential := clamp01(0.55*diag.GrasslandAffinity[i] + 0.25*smoothstep01(0.6, 1.4, diag.AridityRatio[i]) + 0.20*smoothstep01(10, 28, diag.WarmestSeasonTempC[i]))
		shrubPotential := clamp01(0.45*diag.DesertAffinity[i] + 0.30*diag.GrasslandAffinity[i] + 0.20*out.Diagnostics.DryStress[i])
		wetlandPotential := clamp01(0.65*diag.WetlandAffinity[i] + 0.35*out.Diagnostics.Waterlogging[i])
		barePotential := clamp01(math.Max(diag.IceAffinity[i], diag.DesertAffinity[i]*(0.55+0.45*out.Diagnostics.DryStress[i])))

		treeDryPenalty := 1 - 0.45*out.Diagnostics.DryStress[i]*(1-0.40*out.Diagnostics.MoistureAvailability[i])
		treeColdPenalty := 1 - 0.55*out.Diagnostics.ColdStress[i]
		treeWetPenalty := 1 - 0.25*smoothstep01(0.65, 1.0, out.Diagnostics.Waterlogging[i])
		treeMoisture := clamp01(0.55 + 0.45*out.Diagnostics.MoistureAvailability[i] + 0.20*riparianSupport)

		out.Diagnostics.TreeCover[i] = clamp01(
			treePotential *
				(0.45 + 0.55*out.Diagnostics.GrowingHeat[i]) *
				treeMoisture *
				treeDryPenalty *
				treeColdPenalty *
				treeWetPenalty,
		)
		out.Diagnostics.WetlandCover[i] = clamp01(
			wetlandPotential *
				(0.45 + 0.55*math.Max(out.Diagnostics.MoistureAvailability[i], out.Diagnostics.Waterlogging[i])) *
				(1 - 0.35*out.Diagnostics.ColdStress[i]),
		)
		out.Diagnostics.GrassCover[i] = clamp01(
			grassPotential *
				(0.35 + 0.65*out.Diagnostics.GrowingHeat[i]) *
				(0.35 + 0.65*(1-out.Diagnostics.ColdStress[i])) *
				(1 - 0.55*out.Diagnostics.TreeCover[i]),
		)
		out.Diagnostics.ShrubCover[i] = clamp01(
			shrubPotential *
				(0.25 + 0.75*out.Diagnostics.GrowingHeat[i]) *
				(1 - 0.40*out.Diagnostics.ColdStress[i]) *
				(1 - 0.35*out.Diagnostics.TreeCover[i]),
		)
		out.Diagnostics.BareCover[i] = clamp01(
			barePotential * math.Max(1-0.55*out.Diagnostics.MoistureAvailability[i], 0.25+0.75*out.Diagnostics.ColdStress[i]),
		)

		tropicalSupport := 0.0
		if i < len(biomes.Biomes) {
			switch biomes.Biomes[i] {
			case BiomeTropicalRainforest, BiomeTropicalSeasonalForest, BiomeSavanna, BiomeWetland:
				tropicalSupport = 1.0
			}
		}
		coastalWetSupport := clamp01(
			0.45*wetlandPotential +
				0.20*out.Diagnostics.Waterlogging[i] +
				0.20*out.Diagnostics.MoistureAvailability[i] +
				0.15*riparianSupport,
		)
		out.Diagnostics.MangroveAffinity[i] = clamp01(
			coastalValue(coastalExposure, i) *
				coastalWetSupport *
				smoothstep01(12, 24, diag.ColdestSeasonTempC[i]) *
				smoothstep01(50, 180, diag.AnnualPrecipCm[i]) *
				(0.45 + 0.55*tropicalSupport),
		)
		out.Diagnostics.SaltMarshAffinity[i] = clamp01(
			coastalValue(coastalExposure, i) *
				coastalWetSupport *
				smoothstep01(-6, 12, diag.AnnualMeanTempC[i]) *
				(1 - smoothstep01(16, 24, diag.ColdestSeasonTempC[i])),
		)
		out.Diagnostics.PeatlandAffinity[i] = clamp01(
			out.Diagnostics.WetlandCover[i] *
				smoothstep01(-2, 10, diag.AnnualMeanTempC[i]) *
				smoothstep01(60, 160, diag.AnnualPrecipCm[i]) *
				smoothstep01(0.45, 0.95, diag.DrySeasonRatio[i]) *
				(1 - coastalValue(coastalExposure, i)),
		)
		out.Diagnostics.RiparianAffinity[i] = clamp01(
			smoothstep01(0.7, 2.2, channel) *
				peak01(diag.AridityRatio[i], 0.15, 0.9, 2.2) *
				smoothstep01(6, 20, diag.WarmestSeasonTempC[i]) *
				(0.55 + 0.45*smoothstep01(25, 120, diag.AnnualPrecipCm[i])),
		)
		out.Diagnostics.CloudForestAffinity[i] = clamp01(
			out.Diagnostics.TreeCover[i] *
				smoothstep01(800, 1800, elevation[i]) *
				smoothstep01(3200, 1600, elevation[i]) *
				smoothstep01(120, 240, diag.AnnualPrecipCm[i]) *
				smoothstep01(8, 18, diag.WarmestSeasonTempC[i]) *
				smoothstep01(28, 20, diag.WarmestSeasonTempC[i]),
		)
		out.Diagnostics.AlpineMeadowAffinity[i] = clamp01(
			bool01(i < len(biomes.Biomes) && biomes.Biomes[i] == BiomeAlpine) *
				smoothstep01(35, 90, diag.AnnualPrecipCm[i]) *
				smoothstep01(4, 14, diag.WarmestSeasonTempC[i]),
		)

		out.Types[i] = determineVegetationType(
			biomes.Biomes[i],
			out.Diagnostics.TreeCover[i],
			out.Diagnostics.GrassCover[i],
			out.Diagnostics.ShrubCover[i],
			out.Diagnostics.WetlandCover[i],
			out.Diagnostics.BareCover[i],
			out.Diagnostics.MangroveAffinity[i],
			out.Diagnostics.SaltMarshAffinity[i],
			out.Diagnostics.PeatlandAffinity[i],
			out.Diagnostics.RiparianAffinity[i],
			out.Diagnostics.CloudForestAffinity[i],
			out.Diagnostics.AlpineMeadowAffinity[i],
			diag.TropicalWetAffinity[i],
			diag.IceAffinity[i],
			out.Diagnostics.ColdStress[i],
		)
	}
	return out
}

func determineVegetationType(
	biome Biome,
	treeCover, grassCover, shrubCover, wetlandCover, bareCover float64,
	mangrove, saltMarsh, peatland, riparian, cloudForest, alpineMeadow float64,
	tropicalWet, iceAffinity, coldStress float64,
) VegetationType {
	switch {
	case mangrove >= 0.48:
		return VegetationMangrove
	case saltMarsh >= 0.50:
		return VegetationSaltMarsh
	case peatland >= 0.55:
		return VegetationPeatland
	case cloudForest >= 0.60:
		return VegetationCloudForest
	case riparian >= 0.58 && treeCover >= 0.28:
		return VegetationRiparianForest
	case alpineMeadow >= 0.58:
		return VegetationAlpineMeadow
	case biome == BiomeIceCap || (iceAffinity >= 0.65 && coldStress >= 0.60) || (bareCover >= 0.75 && coldStress >= 0.65):
		return VegetationIceBarren
	case wetlandCover >= 0.55:
		return VegetationWetland
	case treeCover >= 0.68 && tropicalWet >= 0.42:
		return VegetationRainforest
	case treeCover >= 0.54:
		return VegetationForest
	case treeCover >= 0.28:
		return VegetationWoodland
	case grassCover >= 0.38 && grassCover >= shrubCover:
		return VegetationGrassland
	case shrubCover >= 0.32:
		return VegetationShrubland
	default:
		return VegetationDesertSparse
	}
}

func coastalValue(values []float64, idx int) float64 {
	if idx >= 0 && idx < len(values) {
		return values[idx]
	}
	return 0
}

func bool01(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
