package climgen

import "math"

type windConvergenceDiagnostic struct {
	Convergence          float64
	BaseConvergence      float64
	RawDivergence        float64
	WindSpeed            float64
	SpeedScale           float64
	NeighborCount        int
	DominantContribution float64
	FitResidual          float64
	GeometrySpanRatio    float64
}

type orographicLiftDiagnostic struct {
	LiftMeters          float64
	Lift                float64
	LocalRiseMeters     float64
	FootprintRiseMeters float64
	BarrierPersistence  float64
	WindFactor          float64
}

const (
	precipConvergenceSpeedMin  = 0.12
	precipConvergenceSpeedFull = 0.55
)

func precipitationConvergenceSpeedScale(speed float64) float64 {
	return 0.15 + 0.85*smoothRamp(precipConvergenceSpeedMin, precipConvergenceSpeedFull, speed)
}

func computeOrographicLift(
	i int,
	vertices []Vector3D,
	elevation []float64,
	adj *FlatAdjacency,
	wind []Vector3D,
) float64 {
	return computeOrographicLiftDiagnostic(i, vertices, elevation, adj, wind).Lift
}

func computeOrographicLiftDiagnostic(
	i int,
	vertices []Vector3D,
	elevation []float64,
	adj *FlatAdjacency,
	wind []Vector3D,
) orographicLiftDiagnostic {
	if i < 0 || i >= len(vertices) || i >= len(wind) {
		return orographicLiftDiagnostic{}
	}
	speed := Length(wind[i])
	if speed < 1e-9 {
		return orographicLiftDiagnostic{}
	}
	windDir := Scale(wind[i], 1.0/speed)
	localRiseSum := 0.0
	localWeightSum := 0.0
	localBarrierWeight := 0.0
	localPositiveWeight := 0.0
	localRiseScale := math.Sqrt(precipitationPhysicalStepScale(len(vertices)))
	for _, k := range adj.GetNeighbors(i) {
		if k < 0 || k >= len(vertices) {
			continue
		}
		fromNeighbor := Normalize(Sub(vertices[i], vertices[k]))
		upwind := Dot(windDir, fromNeighbor)
		if upwind <= 0 {
			continue
		}
		localBarrierWeight += upwind
		rise := elevation[i] - elevation[k]
		if rise <= 0 {
			continue
		}
		rise /= localRiseScale
		localRiseSum += upwind * rise
		localWeightSum += upwind
		localPositiveWeight += upwind
	}
	localRise := 0.0
	if localWeightSum > 1e-9 {
		localRise = localRiseSum / localWeightSum
	}
	localBarrierFrac := 0.0
	if localBarrierWeight > 1e-9 {
		localBarrierFrac = Clamp(localPositiveWeight/localBarrierWeight, 0, 1)
	}

	ws := acquireUpwindWorkspace(len(vertices))
	defer releaseUpwindWorkspace(ws)
	footprintCells, footprintWeights := computeUpwindFootprintWeightsInto(
		ws,
		i,
		vertices,
		adj,
		wind,
		resolutionAdjustedPrecipSteps(precipUpliftTraceSteps, len(vertices)),
		precipUpwindFootprintMinAlignment,
	)
	footprintRise := 0.0
	footprintWeight := 0.0
	footprintBarrierWeight := 0.0
	footprintPositiveWeight := 0.0
	riseScale := math.Sqrt(precipitationPhysicalStepScale(len(vertices)))
	for fidx, donor32 := range footprintCells {
		donor := int(donor32)
		weight := footprintWeights[fidx]
		if donor < 0 || donor >= len(elevation) {
			continue
		}
		footprintBarrierWeight += weight
		rise := elevation[i] - elevation[donor]
		if rise <= 0 {
			continue
		}
		rise /= riseScale
		footprintRise += weight * rise
		footprintWeight += weight
		footprintPositiveWeight += weight
	}
	if footprintWeight > 1e-9 {
		footprintRise /= footprintWeight
	}
	footprintBarrierFrac := 0.0
	if footprintBarrierWeight > 1e-9 {
		footprintBarrierFrac = Clamp(footprintPositiveWeight/footprintBarrierWeight, 0, 1)
	}

	barrierPersistence := Clamp(
		0.30*localBarrierFrac+
			0.70*footprintBarrierFrac,
		0,
		1,
	)
	windFactor := 0.10 + 0.90*smoothRamp(0.16, 0.58, speed)
	sharedRise := math.Min(localRise, footprintRise)
	riseSupport := 0.18*localRise + 0.32*footprintRise + 0.50*sharedRise
	liftMeters := riseSupport * (0.18 + 0.82*barrierPersistence) * windFactor
	lift := smoothRamp(140.0, precipUpliftScaleMeters*1.35, liftMeters)
	return orographicLiftDiagnostic{
		LiftMeters:          liftMeters,
		Lift:                lift,
		LocalRiseMeters:     localRise,
		FootprintRiseMeters: footprintRise,
		BarrierPersistence:  barrierPersistence,
		WindFactor:          windFactor,
	}
}

func computeWindConvergence(
	i int,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
) float64 {
	return computeWindConvergenceDiagnostic(i, vertices, adj, wind).Convergence
}

func computeWindConvergenceDiagnostic(
	i int,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
) windConvergenceDiagnostic {
	if i < 0 || i >= len(vertices) || i >= len(wind) {
		return windConvergenceDiagnostic{}
	}
	east, north := GetTangentVectors(vertices[i])
	ui := Dot(wind[i], east)
	vi := Dot(wind[i], north)
	a00 := 0.0
	a01 := 0.0
	a11 := 0.0
	bu0 := 0.0
	bu1 := 0.0
	bv0 := 0.0
	bv1 := 0.0
	neighborCount := 0
	maxWeight := 0.0
	speed := Length(wind[i])
	type sample struct {
		de, dn float64
		du, dv float64
		weight float64
	}
	samples := make([]sample, 0, 8)

	for _, k := range adj.GetNeighbors(i) {
		if k < 0 || k >= len(vertices) || k >= len(wind) {
			continue
		}
		diff := Sub(vertices[k], vertices[i])
		dotN := Dot(diff, vertices[i])
		tangentDiff := Sub(diff, Scale(vertices[i], dotN))
		de := Dot(tangentDiff, east)
		dn := Dot(tangentDiff, north)
		distSq := de*de + dn*dn
		if distSq < 1e-12 {
			continue
		}

		uk := Dot(wind[k], east)
		vk := Dot(wind[k], north)
		du := uk - ui
		dv := vk - vi
		weight := 1.0 / (distSq + 1e-9)
		a00 += weight * de * de
		a01 += weight * de * dn
		a11 += weight * dn * dn
		bu0 += weight * de * du
		bu1 += weight * dn * du
		bv0 += weight * de * dv
		bv1 += weight * dn * dv
		neighborCount++
		if weight > maxWeight {
			maxWeight = weight
		}
		samples = append(samples, sample{de: de, dn: dn, du: du, dv: dv, weight: weight})
	}
	if neighborCount < 2 {
		return windConvergenceDiagnostic{
			WindSpeed:     speed,
			SpeedScale:    precipitationConvergenceSpeedScale(speed),
			NeighborCount: neighborCount,
		}
	}
	det := a00*a11 - a01*a01
	if math.Abs(det) < 1e-12 {
		return windConvergenceDiagnostic{
			WindSpeed:         speed,
			SpeedScale:        precipitationConvergenceSpeedScale(speed),
			NeighborCount:     neighborCount,
			GeometrySpanRatio: 0,
		}
	}
	inv00 := a11 / det
	inv01 := -a01 / det
	inv11 := a00 / det

	dudx := inv00*bu0 + inv01*bu1
	dudy := inv01*bu0 + inv11*bu1
	dvdx := inv00*bv0 + inv01*bv1
	dvdy := inv01*bv0 + inv11*bv1
	divergence := dudx + dvdy

	trace := a00 + a11
	disc := trace*trace - 4.0*det
	if disc < 0 {
		disc = 0
	}
	root := math.Sqrt(disc)
	lmax := 0.5 * (trace + root)
	lmin := 0.5 * (trace - root)
	spanRatio := 0.0
	if lmax > 1e-12 {
		spanRatio = Clamp(lmin/lmax, 0, 1)
	}

	weightedResidual := 0.0
	weightSum := 0.0
	for _, s := range samples {
		pu := dudx*s.de + dudy*s.dn
		pv := dvdx*s.de + dvdy*s.dn
		errU := pu - s.du
		errV := pv - s.dv
		weightedResidual += s.weight * math.Sqrt(errU*errU+errV*errV)
		weightSum += s.weight
	}
	residual := 0.0
	if weightSum > 1e-12 {
		residual = weightedResidual / weightSum
	}
	dominant := 0.0
	if weightSum > 1e-12 {
		dominant = Clamp(maxWeight/weightSum, 0, 1)
	}
	baseConvergence := Clamp(-2.1*divergence, -1, 1)
	speedScale := precipitationConvergenceSpeedScale(speed)
	convergence := baseConvergence
	if convergence > 0 {
		convergence *= speedScale
	}
	return windConvergenceDiagnostic{
		Convergence:          convergence,
		BaseConvergence:      baseConvergence,
		RawDivergence:        divergence,
		WindSpeed:            speed,
		SpeedScale:           speedScale,
		NeighborCount:        neighborCount,
		DominantContribution: dominant,
		FitResidual:          residual,
		GeometrySpanRatio:    spanRatio,
	}
}

func computeUpwindOceanFetch(
	i int,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	maxSteps int,
) float64 {
	if i < 0 || i >= len(vertices) || i >= len(wind) || elevation[i] < seaLevel {
		return 0
	}
	current := i
	pathFetch := 0.0
	stepScale := precipitationPhysicalStepScale(len(vertices))
	for step := 0; step < maxSteps; step++ {
		next, upwindness := strongestUpwindNeighbor(current, vertices, adj, wind)
		if next < 0 || upwindness <= 0.05 {
			break
		}
		if elevation[next] < seaLevel {
			pathFetch += stepScale
		} else {
			pathFetch *= 0.6
			break
		}
		current = next
	}
	pathFetch = Clamp(pathFetch/(float64(maxSteps)*stepScale), 0, 1)

	ws := acquireUpwindWorkspace(len(vertices))
	defer releaseUpwindWorkspace(ws)
	footprintCells, footprintWeights := computeUpwindFootprintWeightsInto(
		ws,
		i,
		vertices,
		adj,
		wind,
		maxSteps,
		precipUpwindFootprintMinAlignment,
	)
	if len(footprintCells) == 0 {
		return pathFetch
	}
	fetch := 0.0
	landPenalty := 0.0
	for fidx, donor32 := range footprintCells {
		donor := int(donor32)
		if donor < 0 || donor >= len(elevation) {
			continue
		}
		if elevation[donor] < seaLevel {
			fetch += footprintWeights[fidx]
		} else {
			landPenalty += footprintWeights[fidx]
		}
	}
	footprintFetch := Clamp(fetch-0.35*landPenalty, 0, 1)
	absLat := math.Abs(getLatitudeDeg(vertices[i]))
	footprintWeight := smoothRamp(18.0, 34.0, absLat)
	return pathFetch*(1.0-footprintWeight) + footprintFetch*footprintWeight
}

func strongestUpwindNeighbor(
	i int,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
) (int, float64) {
	if i < 0 || i >= len(vertices) || i >= len(wind) {
		return -1, 0
	}
	speed := Length(wind[i])
	if speed < 1e-9 {
		return -1, 0
	}
	windDir := Scale(wind[i], 1.0/speed)
	best := -1
	bestScore := 0.0
	for _, k := range adj.GetNeighbors(i) {
		if k < 0 || k >= len(vertices) {
			continue
		}
		fromNeighbor := Normalize(Sub(vertices[i], vertices[k]))
		upwind := Dot(windDir, fromNeighbor)
		if upwind > bestScore {
			bestScore = upwind
			best = k
		}
	}
	return best, bestScore
}
