package climgen

import "math"

type oceanAtmosphereDiagnostics struct {
	OceanSource  []float64
	WarmMarine   []float64
	DownwindLand []float64
	Retention    []float64
}

func computeCoastalMarineRetention(oceanSource, warmMarine, downwindLand float64) float64 {
	retention := 1.0 - 0.10*downwindLand*warmMarine
	hotLandfallSource := smoothRamp(0.75, 0.95, oceanSource) *
		smoothRamp(0.55, 0.90, downwindLand) *
		smoothRamp(0.55, 0.95, warmMarine)
	retention -= 0.12 * hotLandfallSource
	return Clamp(retention, 0.76, 1.0)
}

func computeDownwindLandExposure(
	i int,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
) float64 {
	if i < 0 || i >= len(vertices) || i >= len(elevation) || i >= len(wind) || elevation[i] >= seaLevel {
		return 0
	}
	windSpeed := Length(wind[i])
	if windSpeed < 1e-9 {
		return 0
	}
	windDir := Scale(wind[i], 1.0/windSpeed)
	exposure := 0.0
	weightSum := 0.0
	for _, k := range adj.GetNeighbors(i) {
		if k < 0 || k >= len(vertices) || k >= len(elevation) || elevation[k] < seaLevel {
			continue
		}
		toNeighbor := Normalize(Sub(vertices[k], vertices[i]))
		downwind := Dot(windDir, toNeighbor)
		if downwind <= 0 {
			continue
		}
		exposure += downwind
		weightSum += 1.0
	}
	if weightSum <= 1e-9 {
		return 0
	}
	return Clamp(exposure/weightSum, 0, 1)
}

func computeOceanAtmosphericMoisture(
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	oceanSource []float64,
	moistureCap []float64,
	rainfallFractionPerCell float64,
	maxIterations int,
) ([]float64, oceanAtmosphereDiagnostics) {
	moisture := append([]float64(nil), oceanSource...)
	stepScale := precipitationPhysicalStepScale(len(vertices))
	diag := oceanAtmosphereDiagnostics{
		OceanSource:  append([]float64(nil), oceanSource...),
		WarmMarine:   make([]float64, len(oceanSource)),
		DownwindLand: make([]float64, len(oceanSource)),
		Retention:    make([]float64, len(oceanSource)),
	}
	for iter := 0; iter < maxIterations; iter++ {
		next := make([]float64, len(moisture))
		maxChange := 0.0
		for i := range moisture {
			if i >= len(elevation) || elevation[i] >= seaLevel {
				continue
			}
			incoming := advectedSpecificHumidity(i, vertices, adj, wind, moisture)
			q := incoming + oceanSource[i]
			condensed := computeOceanCondensation(q, moistureCap[i], rainfallFractionPerCell)
			warmMarine := 0.0
			if i < len(moistureCap) {
				// warm-ocean depletion is driven by SST/air temperature proxy through moisture capacity source scaling;
				// use source strength as a robust local proxy for warm humid marine air.
				warmMarine = smoothRamp(0.45, 1.05, oceanSource[i])
			}
			downwindLand := computeDownwindLandExposure(i, vertices, elevation, seaLevel, adj, wind)
			coastalMarineRetention := computeCoastalMarineRetention(oceanSource[i], warmMarine, downwindLand)
			diag.WarmMarine[i] = warmMarine
			diag.DownwindLand[i] = downwindLand
			diag.Retention[i] = coastalMarineRetention
			retention := math.Pow(precipOceanRetentionFraction*coastalMarineRetention, stepScale)
			next[i] = maxFloat(0, (q-condensed)*retention)
			change := absPrecipFloat(next[i] - moisture[i])
			if change > maxChange {
				maxChange = change
			}
		}
		moisture = next
		if maxChange < 0.0005 {
			break
		}
	}
	return moisture, diag
}

func absPrecipFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
