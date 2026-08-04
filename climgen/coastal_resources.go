package climgen

type CoastalResourceType int

const (
	CoastalResourceOcean CoastalResourceType = iota
	CoastalResourceNone
	CoastalResourceOpenFishery
	CoastalResourceEstuarineFishery
	CoastalResourceShellfish
	CoastalResourceSaltworks
)

func CoastalResourceName(c CoastalResourceType) string {
	names := []string{
		"Ocean",
		"None",
		"Open Fishery",
		"Estuarine Fishery",
		"Shellfish",
		"Saltworks",
	}
	if int(c) < len(names) {
		return names[c]
	}
	return "Unknown"
}

type CoastalResourceDiagnostics struct {
	CoastalAccess       []float64
	CurrentProductivity []float64
	UpwellingPotential  []float64
	OpenFishery         []float64
	EstuarineFishery    []float64
	ShellfishPotential  []float64
	SaltworksPotential  []float64
}

type CoastalResourceResult struct {
	Types       []CoastalResourceType
	Diagnostics *CoastalResourceDiagnostics
}

func ClassifyCoastalResources(
	vertices []Vector3D,
	cells []VoronoiCell,
	climate *SeasonalClimateResult,
	biomes *BiomeResult,
	soils *SoilResult,
	vegetation *VegetationResult,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
	coastalExposure []float64,
	settings CoastalResourceSettings,
) *CoastalResourceResult {
	n := len(elevation)
	out := &CoastalResourceResult{
		Types: make([]CoastalResourceType, n),
		Diagnostics: &CoastalResourceDiagnostics{
			CoastalAccess:       make([]float64, n),
			CurrentProductivity: make([]float64, n),
			UpwellingPotential:  make([]float64, n),
			OpenFishery:         make([]float64, n),
			EstuarineFishery:    make([]float64, n),
			ShellfishPotential:  make([]float64, n),
			SaltworksPotential:  make([]float64, n),
		},
	}
	if biomes == nil || biomes.Diagnostics == nil {
		return out
	}
	diag := biomes.Diagnostics
	currentProductivity, upwellingSupport := deriveCoastalCurrentSignals(vertices, cells, elevation, seaLevel, climate)
	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Types[i] = CoastalResourceOcean
			continue
		}

		coastal := coastalValue(coastalExposure, i)
		runoff := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.Runoff })
		channel := hydrologyValue(hydro, i, func(h *HydrologyBiomeInputs) []float64 { return h.ChannelStrength })
		classWet := hydrologyClassFactor(hydro, i)
		riparianChannel := hydrologyRiparianChannelSupport(hydro, i)

		alluvial := 0.0
		salinity := 0.0
		if soils != nil && soils.Diagnostics != nil {
			if i < len(soils.Diagnostics.Alluvial) {
				alluvial = soils.Diagnostics.Alluvial[i]
			}
			if i < len(soils.Diagnostics.Salinity) {
				salinity = soils.Diagnostics.Salinity[i]
			}
		}

		wetlandCover := 0.0
		mangrove := 0.0
		saltMarsh := 0.0
		if vegetation != nil && vegetation.Diagnostics != nil {
			if i < len(vegetation.Diagnostics.WetlandCover) {
				wetlandCover = vegetation.Diagnostics.WetlandCover[i]
			}
			if i < len(vegetation.Diagnostics.MangroveAffinity) {
				mangrove = vegetation.Diagnostics.MangroveAffinity[i]
			}
			if i < len(vegetation.Diagnostics.SaltMarshAffinity) {
				saltMarsh = vegetation.Diagnostics.SaltMarshAffinity[i]
			}
		}

		coldPenalty := clamp01(maxf(
			diag.AnnualIceFraction[i],
			smoothstep01(10, -5, diag.WarmestSeasonTempC[i]),
		))
		openFishery := clamp01(
			coastal *
				(0.30*smoothstep01(6, 105, runoff) +
					0.14*smoothstep01(0.6, 2.1, channel) +
					0.08*riparianChannel +
					0.18*peak01(diag.AnnualPrecipCm[i], 35, 105, 230) +
					0.10*currentProductivity[i] +
					0.10*upwellingSupport[i] +
					0.18*(1-coldPenalty) +
					0.12*smoothstep01(4, 24, diag.AnnualMeanTempC[i])) *
				(1 - 0.25*classWet) *
				settings.OpenFisheryMultiplier,
		)
		estuarine := clamp01(
			coastal *
				(0.28*alluvial +
					0.22*classWet +
					0.18*smoothstep01(10, 105, runoff) +
					0.10*smoothstep01(0.8, 2.4, channel) +
					0.06*riparianChannel +
					0.08*currentProductivity[i] +
					0.10*wetlandCover +
					0.08*(1-coldPenalty)) *
				settings.EstuarineFisheryMultiplier,
		)
		shellfish := clamp01(
			coastal *
				(0.18*wetlandCover +
					0.14*mangrove +
					0.14*saltMarsh +
					0.14*classWet +
					0.12*peak01(diag.AnnualMeanTempC[i], 2, 16, 28) +
					0.12*peak01(runoff, 3, 22, 85) +
					0.08*peak01(salinity, 0.04, 0.28, 0.72) +
					0.10*currentProductivity[i] +
					0.10*upwellingSupport[i] +
					0.08*alluvial +
					0.06*(1-coldPenalty) +
					0.06*(1-smoothstep01(1.2, 3.2, channel))) *
				settings.ShellfishMultiplier,
		)
		saltworks := clamp01(
			coastal *
				(0.34*salinity +
					0.24*smoothstep01(1.10, 0.32, diag.AridityRatio[i]) +
					0.16*(1-smoothstep01(28, 120, diag.AnnualPrecipCm[i])) +
					0.12*smoothstep01(16, 32, diag.AnnualMeanTempC[i]) +
					0.08*(1-smoothstep01(10, 75, runoff)) +
					0.06*(1-wetlandCover)) *
				settings.SaltworksMultiplier,
		)

		out.Diagnostics.CoastalAccess[i] = coastal
		out.Diagnostics.CurrentProductivity[i] = currentProductivity[i]
		out.Diagnostics.UpwellingPotential[i] = upwellingSupport[i]
		out.Diagnostics.OpenFishery[i] = openFishery
		out.Diagnostics.EstuarineFishery[i] = estuarine
		out.Diagnostics.ShellfishPotential[i] = shellfish
		out.Diagnostics.SaltworksPotential[i] = saltworks
		out.Types[i] = determineCoastalResourceType(coastal, openFishery, estuarine, shellfish, saltworks, settings)
	}
	return out
}

type coastalResourceCandidate struct {
	typ CoastalResourceType
	val float64
}

func determineCoastalResourceType(
	coastal, openFishery, estuarine, shellfish, saltworks float64,
	settings CoastalResourceSettings,
) CoastalResourceType {
	if coastal < 0.14 {
		return CoastalResourceNone
	}
	candidates := []coastalResourceCandidate{
		{CoastalResourceOpenFishery, clamp01(openFishery + settings.OpenFisheryPrimaryBias)},
		{CoastalResourceEstuarineFishery, clamp01(estuarine + settings.EstuarineFisheryPrimaryBias)},
		{CoastalResourceShellfish, clamp01(shellfish + settings.ShellfishPrimaryBias)},
		{CoastalResourceSaltworks, clamp01(saltworks + settings.SaltworksPrimaryBias)},
	}
	best := 0.0
	bestType := CoastalResourceNone
	for _, c := range candidates {
		if c.val > best {
			best = c.val
			bestType = c.typ
		}
	}
	if best < 0.28 {
		return CoastalResourceNone
	}
	return bestType
}

func deriveCoastalCurrentSignals(
	vertices []Vector3D,
	cells []VoronoiCell,
	elevation []float64,
	seaLevel float64,
	climate *SeasonalClimateResult,
) ([]float64, []float64) {
	n := len(elevation)
	currentProductivity := make([]float64, n)
	upwellingSupport := make([]float64, n)
	if climate == nil || len(climate.Currents) != n || len(vertices) != n || len(cells) != n {
		return currentProductivity, upwellingSupport
	}

	adj := BuildFlatAdjacency(cells)
	coastLandDirs := CalculateCoastlineLandDirs(vertices, elevation, seaLevel, adj)
	sourceTemps := ComputeCurrentSourceTemperatures(vertices, elevation, seaLevel, adj, climate.Currents, DefaultCurrentBacktrackDistance)

	oceanCurrentProductivity := make([]float64, n)
	oceanUpwelling := make([]float64, n)
	for i, current := range climate.Currents {
		if !isCoastalOcean(i, elevation, seaLevel, adj) {
			continue
		}
		landDir := coastLandDirs[i]
		landLen := Length(landDir)
		speed := Length(current)
		if landLen < 1e-6 || speed < 1e-6 {
			continue
		}

		east, north := GetTangentVectors(vertices[i])
		landEast := Dot(landDir, east) / landLen
		poleward := Dot(current, north)
		if vertices[i].Y < 0 {
			poleward = -poleward
		}
		localEqC := LatitudeEquilibriumTemp(vertices[i].Y) - FreezingPoint
		sourceAnomC := sourceTemps[i] - FreezingPoint - localEqC
		coldAnomaly := smoothstep01(-0.25, -3.0, sourceAnomC)
		currentStrength := smoothstep01(0.02, 0.16, speed)
		upwelling := clamp01(
			smoothstep01(0.10, 0.85, landEast) *
				smoothstep01(-0.05, -0.35, poleward/maxf(speed, 1e-6)) *
				(0.55 + 0.45*coldAnomaly) *
				currentStrength,
		)

		oceanCurrentProductivity[i] = clamp01(0.65*currentStrength + 0.35*coldAnomaly)
		oceanUpwelling[i] = upwelling
	}

	radius := meshResolutionAdjustedSteps(2, n)
	currentProductivity = spreadPhysicalMaxSignal(cells, elevation, seaLevel, oceanCurrentProductivity, radius)
	upwellingSupport = spreadPhysicalMaxSignal(cells, elevation, seaLevel, oceanUpwelling, radius)
	for i, elev := range elevation {
		if elev < seaLevel {
			currentProductivity[i] = 0
			upwellingSupport[i] = 0
		}
	}
	return currentProductivity, upwellingSupport
}
