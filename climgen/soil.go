package climgen

import "math"

type SoilType int

const (
	SoilOcean SoilType = iota
	SoilCryosol
	SoilRocky
	SoilAridMineral
	SoilDrySteppe
	SoilTemperateLoam
	SoilTropicalWeathered
	SoilAlluvial
	SoilOrganicWet
	SoilPeat
	SoilSalineCoastal
)

func SoilName(s SoilType) string {
	names := []string{
		"Ocean",
		"Cryosol",
		"Rocky",
		"Arid Mineral",
		"Dry Steppe",
		"Temperate Loam",
		"Tropical Weathered",
		"Alluvial",
		"Organic Wet",
		"Peat",
		"Saline Coastal",
	}
	if int(s) < len(names) {
		return names[s]
	}
	return "Unknown"
}

type SoilDiagnostics struct {
	Moisture      []float64
	Drainage      []float64
	Fertility     []float64
	Weathering    []float64
	Salinity      []float64
	Alluvial      []float64
	Organic       []float64
	Rockiness     []float64
	Relief        []float64
	Coastal       []float64
}

type SoilResult struct {
	Types       []SoilType
	Diagnostics *SoilDiagnostics
}

func ClassifySoils(
	cells []VoronoiCell,
	climate *SeasonalClimateResult,
	biomes *BiomeResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	coastalExposure []float64,
) *SoilResult {
	n := len(elevation)
	out := &SoilResult{
		Types: make([]SoilType, n),
		Diagnostics: &SoilDiagnostics{
			Moisture:   make([]float64, n),
			Drainage:   make([]float64, n),
			Fertility:  make([]float64, n),
			Weathering: make([]float64, n),
			Salinity:   make([]float64, n),
			Alluvial:   make([]float64, n),
			Organic:    make([]float64, n),
			Rockiness:  make([]float64, n),
			Relief:     computeLocalRelief(cells, elevation, seaLevel),
			Coastal:    append([]float64(nil), coastalExposure...),
		},
	}
	if climate == nil || biomes == nil || biomes.Diagnostics == nil {
		return out
	}
	diag := biomes.Diagnostics
	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Types[i] = SoilOcean
			continue
		}
		runoff := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
		channel := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.ChannelStrength })
		classFactor := hydrologyClassFactor(hydro, i)
		alluvial := clamp01(0.45*smoothstep01(20, 120, runoff) + 0.35*smoothstep01(0.8, 2.5, channel) + 0.20*classFactor)
		moisture := clamp01(
			0.50*smoothstep01(20, 180, diag.AnnualPrecipCm[i]) +
				0.25*smoothstep01(0.45, 2.20, diag.AridityRatio[i]) +
				0.25*smoothstep01(10, 110, runoff),
		)
		poorDrain := clamp01(
			0.45*smoothstep01(18, 110, runoff) +
				0.20*smoothstep01(0.8, 2.2, channel) +
				0.10*(1-smoothstep01(120, 1100, out.Diagnostics.Relief[i])) +
				0.15*classFactor,
		)
		drainage := clamp01(1 - poorDrain)
		weathering := clamp01(
			smoothstep01(6, 26, diag.AnnualMeanTempC[i]) *
				smoothstep01(45, 220, diag.AnnualPrecipCm[i]) *
				(1 - 0.55*diag.IceAffinity[i]),
		)
		organic := clamp01(
			(0.55*poorDrain +
				0.30*smoothstep01(-4, 10, diag.AnnualMeanTempC[i]) +
				0.15*smoothstep01(55, 170, diag.AnnualPrecipCm[i])) *
				smoothstep01(0.45, 0.95, poorDrain),
		)
		rockiness := clamp01(
			0.60*smoothstep01(120, 1800, out.Diagnostics.Relief[i]) +
				0.25*smoothstep01(900, 2600, elevation[i]) +
				0.15*(1-moisture),
		)
		salinity := clamp01(
			coastalValue(coastalExposure, i) *
				smoothstep01(1.05, 0.30, diag.AridityRatio[i]) *
				smoothstep01(0.35, 0.85, poorDrain),
		)
		fertility := clamp01(
			0.40*alluvial +
				0.25*organic +
				0.25*peak01(diag.AridityRatio[i], 0.55, 1.10, 1.85) +
				0.20*smoothstep01(25, 140, diag.AnnualPrecipCm[i]) -
				0.25*weathering -
				0.25*rockiness -
				0.20*salinity,
		)

		out.Diagnostics.Alluvial[i] = alluvial
		out.Diagnostics.Moisture[i] = moisture
		out.Diagnostics.Drainage[i] = drainage
		out.Diagnostics.Weathering[i] = weathering
		out.Diagnostics.Organic[i] = organic
		out.Diagnostics.Rockiness[i] = rockiness
		out.Diagnostics.Salinity[i] = salinity
		out.Diagnostics.Fertility[i] = fertility

		out.Types[i] = determineSoilType(
			diag,
			i,
			moisture,
			drainage,
			fertility,
			weathering,
			salinity,
			alluvial,
			organic,
			rockiness,
		)
	}
	return out
}

func determineSoilType(
	diag *BiomeDiagnostics,
	i int,
	moisture, drainage, fertility, weathering, salinity, alluvial, organic, rockiness float64,
) SoilType {
	switch {
	case diag == nil:
		return SoilTemperateLoam
	case diag.IceAffinity[i] >= 0.45 || diag.WarmestSeasonTempC[i] <= 2:
		return SoilCryosol
	case salinity >= 0.45:
		return SoilSalineCoastal
	case organic >= 0.68 && moisture >= 0.55 && diag.AnnualMeanTempC[i] <= 10:
		return SoilPeat
	case alluvial >= 0.58 && fertility >= 0.40:
		return SoilAlluvial
	case organic >= 0.58 && moisture >= 0.55 && drainage <= 0.45:
		return SoilOrganicWet
	case rockiness >= 0.60 && fertility <= 0.45:
		return SoilRocky
	case diag.DesertAffinity[i] >= 0.48 && moisture <= 0.28:
		return SoilAridMineral
	case diag.GrasslandAffinity[i] >= 0.40 && moisture <= 0.52:
		return SoilDrySteppe
	case weathering >= 0.60 && diag.TropicalWetAffinity[i] >= 0.35:
		return SoilTropicalWeathered
	default:
		return SoilTemperateLoam
	}
}

func computeLocalRelief(cells []VoronoiCell, elevation []float64, seaLevel float64) []float64 {
	relief := make([]float64, len(elevation))
	for i, cell := range cells {
		if elevation[i] < seaLevel || len(cell.NeighborSiteIndices) == 0 {
			continue
		}
		sum := 0.0
		count := 0.0
		for _, n := range cell.NeighborSiteIndices {
			ni := int(n)
			if ni < 0 || ni >= len(elevation) {
				continue
			}
			sum += math.Abs(elevation[i] - elevation[ni])
			count++
		}
		if count > 0 {
			relief[i] = sum / count
		}
	}
	return relief
}

func hydrologyValue(hydro *HydrologyBiomeInputs, idx int, sel func(*HydrologyBiomeInputs) []float64) float64 {
	if hydro == nil {
		return 0
	}
	values := sel(hydro)
	if idx < 0 || idx >= len(values) {
		return 0
	}
	return values[idx]
}

func hydrologyClassFactor(hydro *HydrologyBiomeInputs, idx int) float64 {
	if hydro == nil || idx < 0 || idx >= len(hydro.CellClass) {
		return 0
	}
	switch hydro.CellClass[idx] {
	case "floodplain":
		return 1.0
	case "delta":
		return 0.95
	case "lake_reach":
		return 0.85
	case "coast_outlet":
		return 0.65
	case "confluence":
		return 0.55
	case "trunk":
		return 0.35
	default:
		return 0
	}
}
