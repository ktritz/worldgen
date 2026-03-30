package climgen

import "math"

const (
	oceanClimateMinCoastCurrent = 0.02
	oceanClimateMinSignalDeg    = 10.0
	oceanClimateMaxSignalDeg    = 55.0
	oceanClimateWarmThresholdC  = 0.75
	oceanClimateColdThresholdC  = -0.75
	oceanClimateMaxLandElevM    = 1200.0
)

func computeOceanClimateDiagnostics(
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	currents *OceanCurrentResult,
	temperature *TemperatureResult,
) OceanClimateDiagnostics {
	var d OceanClimateDiagnostics
	if currents == nil || len(currents.Currents) != len(vertices) {
		return d
	}

	sourceTemps := ComputeCurrentSourceTemperatures(
		vertices,
		elevation,
		seaLevel,
		adj,
		currents.Currents,
		DefaultCurrentBacktrackDistance,
	)

	sourceAnomsAbs := make([]float64, 0, len(vertices))
	coastLandDirs := CalculateCoastlineLandDirs(vertices, elevation, seaLevel, adj)
	warmSignals := make([]float64, 0, 128)
	coldSignals := make([]float64, 0, 128)
	coastalOceanAnom := make([]float64, len(vertices))
	for i := range coastalOceanAnom {
		coastalOceanAnom[i] = math.NaN()
	}

	for i, current := range currents.Currents {
		if elevation[i] >= seaLevel {
			continue
		}

		localEqC := LatitudeEquilibriumTemp(vertices[i].Y) - FreezingPoint
		sourceAnomC := sourceTemps[i] - FreezingPoint - localEqC
		sourceAnomsAbs = append(sourceAnomsAbs, math.Abs(sourceAnomC))
		coastalOceanAnom[i] = sourceAnomC

		if !isCoastalOcean(i, elevation, seaLevel, adj) {
			continue
		}
		if LengthSq(coastLandDirs[i]) < 1e-12 {
			continue
		}

		speed := Length(current)
		if speed < oceanClimateMinCoastCurrent {
			continue
		}

		east, north := GetTangentVectors(vertices[i])
		landEast := Dot(coastLandDirs[i], east) / Length(coastLandDirs[i])
		poleward := Dot(current, north)
		if vertices[i].Y < 0 {
			poleward = -poleward
		}

		absLatDeg := math.Abs(getLatitudeDeg(vertices[i]))
		if absLatDeg < oceanClimateMinSignalDeg || absLatDeg > oceanClimateMaxSignalDeg {
			continue
		}

		if landEast < -0.2 && poleward > 0.15*speed {
			warmSignals = append(warmSignals, sourceAnomC)
		}
		if landEast > 0.2 && poleward < -0.15*speed {
			coldSignals = append(coldSignals, -sourceAnomC)
		}
	}

	d.SourceAnomalyMeanAbsC = mean(sourceAnomsAbs)
	d.SourceAnomalyP90AbsC = percentile(sourceAnomsAbs, 0.90)
	d.WarmWesternBoundarySignalC = mean(warmSignals)
	d.ColdEasternBoundarySignalC = mean(coldSignals)

	if temperature == nil || len(temperature.TemperatureCelsius) != len(vertices) {
		return d
	}

	xs := make([]float64, 0, 128)
	ys := make([]float64, 0, 128)
	warmLandResiduals := make([]float64, 0, 64)
	coldLandResiduals := make([]float64, 0, 64)

	for i, tempC := range temperature.TemperatureCelsius {
		if !isCoastalLand(i, elevation, seaLevel, adj) {
			continue
		}
		if elevation[i] > oceanClimateMaxLandElevM {
			continue
		}

		oceanAnom, ok := adjacentOceanSourceAnomaly(i, elevation, seaLevel, adj, coastalOceanAnom)
		if !ok {
			continue
		}
		if math.Abs(oceanAnom) < 0.5 {
			continue
		}

		localEqC := LatitudeEquilibriumTemp(vertices[i].Y) - FreezingPoint - LapseRate*elevation[i]
		landResidualC := tempC - localEqC
		xs = append(xs, oceanAnom)
		ys = append(ys, landResidualC)

		if oceanAnom >= oceanClimateWarmThresholdC {
			warmLandResiduals = append(warmLandResiduals, landResidualC)
		}
		if oceanAnom <= oceanClimateColdThresholdC {
			coldLandResiduals = append(coldLandResiduals, landResidualC)
		}
	}

	d.CoastalLandCouplingCorr = corr(xs, ys)
	d.WarmAdjacentLandResidualC = mean(warmLandResiduals)
	d.ColdAdjacentLandResidualC = mean(coldLandResiduals)
	return d
}

func isCoastalOcean(i int, elevation []float64, seaLevel float64, adj *FlatAdjacency) bool {
	if elevation[i] >= seaLevel {
		return false
	}
	for _, k := range adj.GetNeighbors(i) {
		if k >= 0 && k < len(elevation) && elevation[k] >= seaLevel {
			return true
		}
	}
	return false
}

func adjacentOceanSourceAnomaly(
	i int,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	sourceAnom []float64,
) (float64, bool) {
	sum := 0.0
	count := 0
	for _, k := range adj.GetNeighbors(i) {
		if k < 0 || k >= len(elevation) || elevation[k] >= seaLevel {
			continue
		}
		if math.IsNaN(sourceAnom[k]) {
			continue
		}
		sum += sourceAnom[k]
		count++
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}
