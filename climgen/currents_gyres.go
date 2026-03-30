package climgen

const (
	gyreGuidanceBands       = 18
	gyreGuidanceMinOpenness = 0.62
	gyreGuidanceMaxWest     = 0.12
	gyreGuidanceMaxGateway  = 0.08
	gyreGuidanceBlendBase   = 0.10
	gyreGuidanceBlendScale  = 0.18
	gyreGuidanceSpeedBlend  = 0.16
)

// ApplyGyreInteriorGuidance nudges broad open-ocean interiors toward a smoother
// basin-scale return flow derived from component/latitude-band mean transport.
// This suppresses medium-scale swirl in gyre interiors without touching
// gateways or boundary-current zones.
func ApplyGyreInteriorGuidance(
	currents []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	componentAssignments []int,
	openness []float64,
	westernBoundary []float64,
	gatewayStrength []float64,
) []Vector3D {
	if len(currents) != len(vertices) || len(componentAssignments) != len(vertices) {
		return currents
	}

	maxComponent := -1
	for _, id := range componentAssignments {
		if id > maxComponent {
			maxComponent = id
		}
	}
	if maxComponent < 0 {
		return currents
	}

	type bandMean struct {
		eastSum   []float64
		northSum  []float64
		weightSum []float64
	}
	means := make([]bandMean, maxComponent+1)
	for i := range means {
		means[i] = bandMean{
			eastSum:   make([]float64, gyreGuidanceBands),
			northSum:  make([]float64, gyreGuidanceBands),
			weightSum: make([]float64, gyreGuidanceBands),
		}
	}

	for i, current := range currents {
		if !eligibleGyreInteriorCell(i, elevation, seaLevelThreshold, openness, westernBoundary, gatewayStrength, componentAssignments) {
			continue
		}
		speed := Length(current)
		if speed < 1e-9 {
			continue
		}

		componentID := componentAssignments[i]
		band := latitudeBandIndex(vertices[i], gyreGuidanceBands)
		east, north := GetTangentVectors(vertices[i])
		weight := 0.25 + 0.75*Clamp(openness[i], 0, 1)

		means[componentID].eastSum[band] += weight * Dot(current, east)
		means[componentID].northSum[band] += weight * Dot(current, north)
		means[componentID].weightSum[band] += weight
	}

	smoothedEast := make([][]float64, len(means))
	smoothedNorth := make([][]float64, len(means))
	for i, mean := range means {
		smoothedEast[i] = smoothBandMeans(mean.eastSum, mean.weightSum)
		smoothedNorth[i] = smoothBandMeans(mean.northSum, mean.weightSum)
	}

	guided := make([]Vector3D, len(currents))
	copy(guided, currents)

	for i, current := range currents {
		if !eligibleGyreInteriorCell(i, elevation, seaLevelThreshold, openness, westernBoundary, gatewayStrength, componentAssignments) {
			continue
		}

		componentID := componentAssignments[i]
		band := latitudeBandIndex(vertices[i], gyreGuidanceBands)
		east, north := GetTangentVectors(vertices[i])
		target := Add(
			Scale(east, smoothedEast[componentID][band]),
			Scale(north, smoothedNorth[componentID][band]),
		)

		targetSpeed := Length(target)
		currentSpeed := Length(current)
		if targetSpeed < 1e-6 || currentSpeed < 1e-9 {
			continue
		}

		align := Dot(current, target) / (currentSpeed * targetSpeed)
		misalign := Clamp(0.5*(1.0-align), 0, 1)
		blend := (gyreGuidanceBlendBase + gyreGuidanceBlendScale*Clamp(openness[i], 0, 1)) * (0.35 + 0.65*misalign)

		mixed := Add(Scale(current, 1.0-blend), Scale(target, blend))
		mixed = Sub(mixed, Scale(vertices[i], Dot(mixed, vertices[i])))
		mixedLen := Length(mixed)
		if mixedLen < 1e-9 {
			continue
		}

		targetMagnitude := currentSpeed*(1.0-gyreGuidanceSpeedBlend) + targetSpeed*gyreGuidanceSpeedBlend
		guided[i] = Scale(mixed, targetMagnitude/mixedLen)
	}

	return guided
}

func eligibleGyreInteriorCell(
	i int,
	elevation []float64,
	seaLevelThreshold float64,
	openness []float64,
	westernBoundary []float64,
	gatewayStrength []float64,
	componentAssignments []int,
) bool {
	if i < 0 || i >= len(elevation) || elevation[i] >= seaLevelThreshold {
		return false
	}
	if i >= len(openness) || openness[i] < gyreGuidanceMinOpenness {
		return false
	}
	if i >= len(westernBoundary) || westernBoundary[i] > gyreGuidanceMaxWest {
		return false
	}
	if i >= len(gatewayStrength) || gatewayStrength[i] > gyreGuidanceMaxGateway {
		return false
	}
	return componentAssignments[i] >= 0
}

func smoothBandMeans(sum []float64, weight []float64) []float64 {
	means := make([]float64, len(sum))
	for i := range means {
		if i < len(weight) && weight[i] > 1e-9 {
			means[i] = sum[i] / weight[i]
		}
	}

	smoothed := make([]float64, len(means))
	copy(smoothed, means)
	next := make([]float64, len(means))
	for iter := 0; iter < 2; iter++ {
		for i := range smoothed {
			total := smoothed[i]
			totalWeight := 1.0
			if i > 0 {
				total += 0.7 * smoothed[i-1]
				totalWeight += 0.7
			}
			if i+1 < len(smoothed) {
				total += 0.7 * smoothed[i+1]
				totalWeight += 0.7
			}
			next[i] = total / totalWeight
		}
		smoothed, next = next, smoothed
	}

	return smoothed
}
