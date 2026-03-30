package climgen

import (
	"math"
	"sort"
)

const (
	marineLandDiffusionBlend      = 0.16
	marineLandDiffusionIterations = 2
	marineDiffusionRegimeMin      = 0.60
	marineDiffusionRegimeMax      = 1.35
)

type marineSweepDiagnostics struct {
	DonorIndex    []float64
	DonorStrength []float64
	DonorOutgoing []float64
	DonorOceanAtm []float64
	DonorDownwind []float64
	RootIndex     []float64
	RootStrength  []float64
	RootOceanAtm  []float64
	RootDownwind  []float64
	RootSource    []float64
	RootRetention []float64
	RootSteps     []float64
}

func newMarineSweepDiagnostics(n int) marineSweepDiagnostics {
	return marineSweepDiagnostics{
		DonorIndex:    make([]float64, n),
		DonorStrength: make([]float64, n),
		DonorOutgoing: make([]float64, n),
		DonorOceanAtm: make([]float64, n),
		DonorDownwind: make([]float64, n),
		RootIndex:     make([]float64, n),
		RootStrength:  make([]float64, n),
		RootOceanAtm:  make([]float64, n),
		RootDownwind:  make([]float64, n),
		RootSource:    make([]float64, n),
		RootRetention: make([]float64, n),
		RootSteps:     make([]float64, n),
	}
}

func marineLandfallEntryScale(coastalImmediate float64, uplift float64, landTravel float64, tempC float64) float64 {
	nearInlandCorridor := transportCorridorWeight(landTravel)
	scale := 1.0 - 0.18*
		Clamp(coastalImmediate, 0, 1)*
		(1.0-0.80*Clamp(uplift, 0, 1))*
		(0.35+0.65*smoothRamp(10.0, 28.0, tempC))*
		(1.0-0.65*nearInlandCorridor)
	return Clamp(scale, 0.72, 1.0)
}

func computeStrongestUpwindGraph(
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
) ([]int, []float64) {
	parent := make([]int, len(vertices))
	strength := make([]float64, len(vertices))
	for i := range parent {
		parent[i], strength[i] = strongestUpwindNeighbor(i, vertices, adj, wind)
	}
	return parent, strength
}

func computeUpwindLandStepCounts(
	parent []int,
	strength []float64,
	elevation []float64,
	seaLevel float64,
	maxSteps int,
) []int {
	steps := make([]int, len(parent))
	state := make([]uint8, len(parent))
	for i := range steps {
		steps[i] = -1
	}

	var dfs func(int) int
	dfs = func(i int) int {
		if i < 0 || i >= len(parent) || i >= len(elevation) {
			return -1
		}
		if elevation[i] < seaLevel {
			return 0
		}
		if steps[i] >= 0 {
			return steps[i]
		}
		if state[i] == 1 {
			return -1
		}
		state[i] = 1
		p := parent[i]
		if p < 0 || p >= len(elevation) || strength[i] <= 0.05 {
			state[i] = 2
			return -1
		}
		if elevation[p] < seaLevel {
			steps[i] = 0
			state[i] = 2
			return 0
		}
		up := dfs(p)
		if up < 0 || up+1 > maxSteps {
			state[i] = 2
			return -1
		}
		steps[i] = up + 1
		state[i] = 2
		return steps[i]
	}

	for i := range steps {
		if elevation[i] >= seaLevel {
			dfs(i)
		}
	}
	return steps
}

func buildMarineSweepOrder(stepCounts []int, elevation []float64, seaLevel float64) []int {
	order := make([]int, 0, len(stepCounts))
	for i, steps := range stepCounts {
		if i >= len(elevation) || elevation[i] < seaLevel || steps < 0 {
			continue
		}
		order = append(order, i)
	}
	sort.Slice(order, func(a, b int) bool {
		return stepCounts[order[a]] < stepCounts[order[b]]
	})
	return order
}

func computeMarineSweepTransport(
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	parent []int,
	strength []float64,
	order []int,
	oceanAtmosphere []float64,
	oceanDiag oceanAtmosphereDiagnostics,
	temperature []float64,
	moistureCap []float64,
	uplift []float64,
	convergence []float64,
	oceanFetch []float64,
	coastalOnshore []float64,
	landTravel []float64,
	landInterior []float64,
	settings PrecipitationSettings,
	rainfallFractionPerCell float64,
) ([]float64, marineSweepDiagnostics) {
	marineIncoming := make([]float64, len(elevation))
	marineOutgoing := make([]float64, len(elevation))
	processed := make([]bool, len(elevation))
	diag := newMarineSweepDiagnostics(len(elevation))
	for i := range marineIncoming {
		if i < len(elevation) && elevation[i] < seaLevel && i < len(oceanAtmosphere) {
			marineIncoming[i] = oceanAtmosphere[i]
			marineOutgoing[i] = oceanAtmosphere[i]
			processed[i] = true
			diag.RootIndex[i] = float64(i)
			diag.RootStrength[i] = 1.0
			diag.RootOceanAtm[i] = oceanAtmosphere[i]
			diag.RootDownwind[i] = computeDownwindLandExposure(i, vertices, elevation, seaLevel, adj, wind)
			if i < len(oceanDiag.OceanSource) {
				diag.RootSource[i] = oceanDiag.OceanSource[i]
			}
			if i < len(oceanDiag.Retention) {
				diag.RootRetention[i] = oceanDiag.Retention[i]
			}
			diag.RootSteps[i] = 0
		} else {
			diag.RootIndex[i] = -1
			diag.DonorIndex[i] = -1
			diag.DonorDownwind[i] = -1
			diag.RootDownwind[i] = -1
			diag.RootSource[i] = 0
			diag.RootRetention[i] = 0
			diag.RootSteps[i] = -1
		}
	}

	for _, i := range order {
		donors, donorWeights := computeWeightedUpwindDonors(i, vertices, adj, wind, 0.08)
		incoming := 0.0
		usedWeight := 0.0
		bestDonor := -1
		bestWeight := 0.0
		for idx, donor := range donors {
			if donor < 0 || donor >= len(marineOutgoing) {
				continue
			}
			if elevation[donor] >= seaLevel && !processed[donor] {
				continue
			}
			weight := donorWeights[idx]
			incoming += marineOutgoing[donor] * weight
			usedWeight += weight
			if weight > bestWeight {
				bestWeight = weight
				bestDonor = donor
			}
		}
		if usedWeight <= 1e-9 {
			p := -1
			if i < len(parent) {
				p = parent[i]
			}
			if p >= 0 && p < len(marineOutgoing) && (elevation[p] < seaLevel || processed[p]) {
				incoming = marineOutgoing[p]
				usedWeight = 1.0
				bestDonor = p
				bestWeight = 1.0
			}
		}
		if usedWeight > 1e-9 && usedWeight < 0.999 {
			incoming /= usedWeight
		}
		if incoming <= 1e-9 {
			continue
		}
		pathWeight := Clamp(0.92+0.06*strength[i], 0.84, 1.00)
		q := incoming * pathWeight
		coastalImmediate := Clamp(oceanFetch[i], 0, 1) * Clamp(coastalOnshore[i], 0, 1) * (1.0 - Clamp(landTravel[i], 0, 1))
		nearInlandCorridor := transportCorridorWeight(landTravel[i])
		tempC := 12.0
		if i >= 0 && i < len(temperature) {
			tempC = temperature[i] - 273.15
		}
		entryScale := marineLandfallEntryScale(coastalImmediate, uplift[i], landTravel[i], tempC)
		q *= entryScale
		marineIncoming[i] = q
		diag.DonorIndex[i] = float64(bestDonor)
		diag.DonorStrength[i] = bestWeight
		diag.DonorDownwind[i] = -1
		if bestDonor >= 0 {
			if bestDonor < len(marineOutgoing) {
				diag.DonorOutgoing[i] = marineOutgoing[bestDonor]
			}
			if bestDonor < len(oceanAtmosphere) {
				diag.DonorOceanAtm[i] = oceanAtmosphere[bestDonor]
			}
			diag.DonorDownwind[i] = computeDownwindLandExposure(bestDonor, vertices, elevation, seaLevel, adj, wind)
			if bestDonor < len(elevation) && elevation[bestDonor] < seaLevel {
				diag.RootIndex[i] = float64(bestDonor)
				diag.RootStrength[i] = bestWeight
				if bestDonor < len(oceanAtmosphere) {
					diag.RootOceanAtm[i] = oceanAtmosphere[bestDonor]
				}
				diag.RootDownwind[i] = diag.DonorDownwind[i]
				if bestDonor < len(oceanDiag.OceanSource) {
					diag.RootSource[i] = oceanDiag.OceanSource[bestDonor]
				}
				if bestDonor < len(oceanDiag.Retention) {
					diag.RootRetention[i] = oceanDiag.Retention[bestDonor]
				}
				diag.RootSteps[i] = 0
			} else if bestDonor < len(diag.RootIndex) && diag.RootIndex[bestDonor] >= 0 {
				diag.RootIndex[i] = diag.RootIndex[bestDonor]
				rootStrength := diag.RootStrength[bestDonor]
				if rootStrength <= 0 {
					rootStrength = 1.0
				}
				diag.RootStrength[i] = bestWeight * rootStrength
				diag.RootOceanAtm[i] = diag.RootOceanAtm[bestDonor]
				diag.RootDownwind[i] = diag.RootDownwind[bestDonor]
				diag.RootSource[i] = diag.RootSource[bestDonor]
				diag.RootRetention[i] = diag.RootRetention[bestDonor]
				rootSteps := diag.RootSteps[bestDonor]
				if rootSteps < 0 {
					rootSteps = 0
				}
				diag.RootSteps[i] = rootSteps + 1
			} else {
				diag.RootIndex[i] = -1
				diag.RootDownwind[i] = -1
				diag.RootSource[i] = 0
				diag.RootRetention[i] = 0
				diag.RootSteps[i] = -1
			}
		}
		condensed := computeLandCondensation(
			q,
			moistureCap[i],
			uplift[i],
			convergence[i],
			oceanFetch[i],
			coastalOnshore[i],
			landTravel[i],
			landInterior[i],
			1.0,
			localPrecipitationScale(settings.CondensationLocalScale, i),
			rainfallFractionPerCell,
			temperature,
			i,
		)
		retained := computeLandRetainedHumidity(
			q,
			condensed,
			moistureCap[i],
			uplift[i],
			oceanFetch[i],
			coastalOnshore[i],
			landTravel[i],
			landInterior[i],
			1.0,
			localPrecipitationScale(settings.LandRetentionLocalScale, i),
		)
		if retained > q {
			retained = q
		}
		retentionScale := Clamp(localPrecipitationScale(settings.LandRetentionLocalScale, i), 0.7, 1.5)
		mix := marineToLandMixFraction(oceanFetch[i], coastalOnshore[i], landTravel[i], landInterior[i])
		outgoingScale := Clamp(0.94+0.04*coastalImmediate+0.10*nearInlandCorridor+0.05*(retentionScale-1.0), 0.88, 1.00)
		marineOutgoing[i] = retained * outgoingScale * (1.0 - 0.05*mix)
		processed[i] = true
	}

	applyMarineLandDiffusion(
		marineIncoming,
		vertices,
		elevation,
		seaLevel,
		adj,
		wind,
		oceanFetch,
		coastalOnshore,
		landTravel,
		landInterior,
	)

	return marineIncoming, diag
}

func computeWeightedUpwindDonors(
	i int,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	minAlignment float64,
) ([]int, []float64) {
	if i < 0 || i >= len(vertices) || i >= len(wind) {
		return nil, nil
	}
	windVec := wind[i]
	windSpeed := Length(windVec)
	if windSpeed < 1e-9 {
		return nil, nil
	}
	windDir := Scale(windVec, 1.0/windSpeed)

	donors := make([]int, 0, 6)
	weights := make([]float64, 0, 6)
	weightSum := 0.0
	minUpwind := Clamp(minAlignment, 0, 0.5)
	for _, k := range adj.GetNeighbors(i) {
		if k < 0 || k >= len(vertices) {
			continue
		}
		fromNeighbor := Normalize(Sub(vertices[i], vertices[k]))
		upwind := Dot(windDir, fromNeighbor)
		if upwind <= minUpwind {
			continue
		}
		weight := upwind * upwind
		donors = append(donors, k)
		weights = append(weights, weight)
		weightSum += weight
	}
	if weightSum <= 1e-9 {
		return nil, nil
	}
	for idx := range weights {
		weights[idx] /= weightSum
	}
	return donors, weights
}

func applyMarineLandDiffusion(
	marineIncoming []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	oceanFetch []float64,
	coastalOnshore []float64,
	landTravel []float64,
	landInterior []float64,
) {
	if len(marineIncoming) == 0 {
		return
	}
	current := append([]float64(nil), marineIncoming...)
	for iter := 0; iter < marineLandDiffusionIterations; iter++ {
		next := append([]float64(nil), current...)
		for i := range current {
			if i >= len(elevation) || elevation[i] < seaLevel {
				continue
			}
			sum := 0.0
			weightSum := 0.0
			for _, k := range adj.GetNeighbors(i) {
				if k < 0 || k >= len(current) || k >= len(elevation) || elevation[k] < seaLevel {
					continue
				}
				w := marineDiffusionNeighborWeight(i, k, vertices, wind)
				if k < len(landTravel) {
					w += 0.35 * transportCorridorWeight(landTravel[k])
				}
				if k < len(landInterior) {
					w += 0.20 * Clamp(landInterior[k], 0, 1)
				}
				sum += current[k] * w
				weightSum += w
			}
			if weightSum <= 1e-9 {
				continue
			}
			neighborMean := sum / weightSum
			coastalImmediate := 0.0
			if i < len(oceanFetch) && i < len(coastalOnshore) && i < len(landTravel) {
				coastalImmediate = Clamp(oceanFetch[i], 0, 1) * Clamp(coastalOnshore[i], 0, 1) * (1.0 - Clamp(landTravel[i], 0, 1))
			}
			corridor := 0.0
			if i < len(landTravel) {
				corridor = transportCorridorWeight(landTravel[i])
			}
			interior := 0.0
			if i < len(landInterior) {
				interior = Clamp(landInterior[i], 0, 1)
			}
			regime := marineDiffusionRegimeFactor(i, vertices, wind)
			alpha := marineLandDiffusionBlend * regime * (0.15 + 0.55*corridor + 0.30*interior) * (1.0 - 0.75*coastalImmediate)
			alpha = Clamp(alpha, 0, 0.22)
			next[i] = (1.0-alpha)*current[i] + alpha*neighborMean
		}
		current = next
	}
	copy(marineIncoming, current)
}

func marineDiffusionNeighborWeight(
	i int,
	k int,
	vertices []Vector3D,
	wind []Vector3D,
) float64 {
	weight := 0.25
	if i < 0 || i >= len(vertices) || i >= len(wind) || k < 0 || k >= len(vertices) {
		return weight
	}
	windVec := wind[i]
	windSpeed := Length(windVec)
	if windSpeed < 1e-9 {
		return 1.0
	}
	windDir := Scale(windVec, 1.0/windSpeed)
	toNeighbor := Normalize(Sub(vertices[k], vertices[i]))
	alignment := math.Abs(Dot(windDir, toNeighbor))
	return 0.20 + 1.10*alignment*alignment
}

func marineDiffusionRegimeFactor(i int, vertices []Vector3D, wind []Vector3D) float64 {
	if i < 0 || i >= len(vertices) || i >= len(wind) {
		return 1.0
	}
	absLat := math.Abs(getLatitudeDeg(vertices[i]))
	midlat := smoothRamp(28.0, 42.0, absLat) * (1.0 - smoothRamp(60.0, 72.0, absLat))
	subtropics := smoothRamp(12.0, 20.0, absLat) * (1.0 - smoothRamp(34.0, 44.0, absLat))
	tropics := 1.0 - smoothRamp(10.0, 22.0, absLat)

	east, _ := GetTangentVectors(vertices[i])
	speed := Length(wind[i])
	westerly := 0.0
	if speed > 1e-9 {
		zonal := Dot(wind[i], east) / speed
		westerly = smoothRamp(0.05, 0.40, zonal)
	}

	regime := 0.82 + 0.42*midlat*westerly - 0.14*subtropics - 0.08*tropics
	return Clamp(regime, marineDiffusionRegimeMin, marineDiffusionRegimeMax)
}
