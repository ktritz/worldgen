package climgen

import "math"

type WaterResourceType int

const (
	WaterResourceOcean WaterResourceType = iota
	WaterResourceScarce
	WaterResourceSeasonal
	WaterResourceReliableSurface
	WaterResourceGroundwater
	WaterResourceLakeOasis
)

func WaterResourceName(w WaterResourceType) string {
	names := []string{
		"Ocean",
		"Scarce",
		"Seasonal Surface Water",
		"Reliable Surface Water",
		"Groundwater",
		"Lake/Oasis Water",
	}
	if int(w) < len(names) {
		return names[w]
	}
	return "Unknown"
}

type WaterResourceDiagnostics struct {
	SurfaceReliability   []float64
	SeasonalAvailability []float64
	GroundwaterPotential []float64
	LakeAccess           []float64
	DroughtResilience    []float64
}

type WaterResourceResult struct {
	Types       []WaterResourceType
	Diagnostics *WaterResourceDiagnostics
}

func ClassifyWaterResources(
	biomes *BiomeResult,
	soils *SoilResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	settings WaterResourceSettings,
) *WaterResourceResult {
	n := len(elevation)
	out := &WaterResourceResult{
		Types: make([]WaterResourceType, n),
		Diagnostics: &WaterResourceDiagnostics{
			SurfaceReliability:   make([]float64, n),
			SeasonalAvailability: make([]float64, n),
			GroundwaterPotential: make([]float64, n),
			LakeAccess:           make([]float64, n),
			DroughtResilience:    make([]float64, n),
		},
	}
	if biomes == nil || biomes.Diagnostics == nil {
		return out
	}
	diag := biomes.Diagnostics
	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Types[i] = WaterResourceOcean
			continue
		}

		runoff := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
		channel := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.ChannelStrength })
		classWet := hydrologyClassFactor(hydro, i)
		riparianChannel := hydrologyRiparianChannelSupport(hydro, i)

		soilDrainage := 0.5
		soilAlluvial := 0.0
		soilSalinity := 0.0
		soilOrganic := 0.0
		if soils != nil && soils.Diagnostics != nil {
			if i < len(soils.Diagnostics.Drainage) {
				soilDrainage = soils.Diagnostics.Drainage[i]
			}
			if i < len(soils.Diagnostics.Alluvial) {
				soilAlluvial = soils.Diagnostics.Alluvial[i]
			}
			if i < len(soils.Diagnostics.Salinity) {
				soilSalinity = soils.Diagnostics.Salinity[i]
			}
			if i < len(soils.Diagnostics.Organic) {
				soilOrganic = soils.Diagnostics.Organic[i]
			}
		}

		droughtResilience := clamp01(
			0.45*smoothstep01(0.80, 2.10, diag.AridityRatio[i]) +
				0.25*smoothstep01(0.45, 0.10, diag.DrySeasonRatio[i]) +
				0.20*peak01(diag.AnnualPrecipCm[i], 25, 95, 210) +
				0.10*(1-diag.AnnualIceFraction[i]),
		)
		surfaceReliable := clamp01(
			(0.38*smoothstep01(8, 110, runoff) +
				0.18*smoothstep01(0.6, 2.2, channel) +
				0.10*riparianChannel +
				0.14*classWet +
				0.10*peak01(diag.AnnualPrecipCm[i], 30, 100, 220) +
				0.10*droughtResilience) *
				settings.SurfaceReliabilityMultiplier,
		)
		seasonalAvail := clamp01(
			(0.34*peak01(diag.AnnualPrecipCm[i], 15, 55, 130) +
				0.24*smoothstep01(0.55, 0.12, diag.DrySeasonRatio[i]) +
				0.18*smoothstep01(4, 42, runoff) +
				0.12*soilAlluvial +
				0.12*(1-soilSalinity)) *
				settings.SeasonalAvailabilityMultiplier,
		)
		groundwater := clamp01(
			(0.30*peak01(soilDrainage, 0.25, 0.60, 0.92) +
				0.20*soilAlluvial +
				0.16*(1-soilSalinity) +
				0.14*peak01(diag.AnnualPrecipCm[i], 18, 70, 170) +
				0.10*droughtResilience +
				0.10*(1-soilOrganic)) *
				settings.GroundwaterMultiplier,
		)
		lakeAccess := clamp01(
			(0.55*lakeClassFactor(hydro, i) +
				0.18*classWet +
				0.10*smoothstep01(0.4, 1.6, channel) +
				0.07*soilAlluvial +
				0.10*droughtResilience) *
				settings.LakeAccessMultiplier,
		)

		out.Diagnostics.DroughtResilience[i] = droughtResilience
		out.Diagnostics.SurfaceReliability[i] = surfaceReliable
		out.Diagnostics.SeasonalAvailability[i] = seasonalAvail
		out.Diagnostics.GroundwaterPotential[i] = groundwater
		out.Diagnostics.LakeAccess[i] = lakeAccess
		out.Types[i] = determineWaterResourceType(surfaceReliable, seasonalAvail, groundwater, lakeAccess, settings)
	}
	return out
}

type waterResourceCandidate struct {
	typ WaterResourceType
	val float64
}

func determineWaterResourceType(
	surfaceReliable, seasonalAvail, groundwater, lakeAccess float64,
	settings WaterResourceSettings,
) WaterResourceType {
	if lakeAccess >= 0.36 && lakeAccess >= groundwater-0.04 && lakeAccess >= seasonalAvail-0.04 {
		return WaterResourceLakeOasis
	}
	candidates := []waterResourceCandidate{
		{WaterResourceReliableSurface, clamp01(surfaceReliable + settings.SurfacePrimaryBias)},
		{WaterResourceSeasonal, clamp01(seasonalAvail + settings.SeasonalPrimaryBias)},
		{WaterResourceGroundwater, clamp01(groundwater + settings.GroundwaterPrimaryBias)},
		{WaterResourceLakeOasis, clamp01(lakeAccess + settings.LakePrimaryBias)},
	}
	best := 0.0
	bestType := WaterResourceScarce
	for _, c := range candidates {
		if c.val > best {
			best = c.val
			bestType = c.typ
		}
	}
	if best < 0.28 {
		return WaterResourceScarce
	}
	return bestType
}

func lakeClassFactor(hydro *HydrologyBiomeInputs, idx int) float64 {
	if hydro == nil || idx < 0 {
		return 0
	}
	support := 0.0
	if idx < len(hydro.LakeClassSupport) {
		support = hydro.LakeClassSupport[idx]
	}
	if idx < len(hydro.CellClass) {
		support = math.Max(support, directLakeClassFactor(hydro.CellClass[idx]))
	}
	return support
}
