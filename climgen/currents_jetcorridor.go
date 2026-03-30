package climgen

const (
	jetCorridorBands        = 18
	jetCorridorSeedWest     = 0.28
	jetCorridorAffectWest   = 0.06
	jetCorridorMaxGateway   = 0.08
	jetCorridorBlendBase    = 0.18
	jetCorridorBlendScale   = 0.34
	jetCorridorSpeedRetain  = 0.80
	jetCorridorPolewardOnly = 0.10
)

// ApplyWesternBoundaryJetCorridor extends coherent boundary-current structure a
// short distance away from the strongest western boundary cells. This keeps
// poleward western-boundary jets from broadening immediately into generic
// interior flow while still leaving gateways and open-ocean gyres untouched.
func ApplyWesternBoundaryJetCorridor(
	currents []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	coastLandDirs []Vector3D,
	westernBoundary []float64,
	gatewayStrength []float64,
	componentAssignments []int,
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

	type jetBand struct {
		speedSum  []float64
		weightSum []float64
	}
	jetBands := make([]jetBand, maxComponent+1)
	for i := range jetBands {
		jetBands[i] = jetBand{
			speedSum:  make([]float64, jetCorridorBands),
			weightSum: make([]float64, jetCorridorBands),
		}
	}

	for i, current := range currents {
		if !eligibleJetSeedCell(i, elevation, seaLevelThreshold, coastLandDirs, westernBoundary, gatewayStrength, componentAssignments) {
			continue
		}
		speed := Length(current)
		if speed < 1e-9 {
			continue
		}
		axis := coastParallelAxis(vertices[i], coastLandDirs[i], current)
		if LengthSq(axis) < 1e-12 {
			continue
		}
		along := Dot(current, axis)
		if along <= jetCorridorPolewardOnly*speed {
			continue
		}

		componentID := componentAssignments[i]
		band := latitudeBandIndex(vertices[i], jetCorridorBands)
		weight := Clamp(westernBoundary[i], 0, 1)
		jetBands[componentID].speedSum[band] += along * weight
		jetBands[componentID].weightSum[band] += weight
	}

	smoothedSpeed := make([][]float64, len(jetBands))
	for i, band := range jetBands {
		smoothedSpeed[i] = smoothBandMeans(band.speedSum, band.weightSum)
	}

	shaped := make([]Vector3D, len(currents))
	copy(shaped, currents)

	for i, current := range currents {
		if !eligibleJetCorridorCell(i, elevation, seaLevelThreshold, coastLandDirs, westernBoundary, gatewayStrength, componentAssignments) {
			continue
		}

		componentID := componentAssignments[i]
		band := latitudeBandIndex(vertices[i], jetCorridorBands)
		targetSpeed := smoothedSpeed[componentID][band]
		if targetSpeed <= 1e-6 {
			continue
		}

		axis := coastParallelAxis(vertices[i], coastLandDirs[i], current)
		if LengthSq(axis) < 1e-12 {
			continue
		}

		currentSpeed := Length(current)
		if currentSpeed < 1e-9 {
			continue
		}

		target := Scale(axis, targetSpeed)
		blend := jetCorridorBlendBase + jetCorridorBlendScale*Clamp(westernBoundary[i], 0, 1)
		mixed := Add(Scale(current, 1.0-blend), Scale(target, blend))
		mixed = Sub(mixed, Scale(vertices[i], Dot(mixed, vertices[i])))
		mixedLen := Length(mixed)
		if mixedLen < 1e-9 {
			continue
		}

		finalSpeed := currentSpeed*jetCorridorSpeedRetain + targetSpeed*(1.0-jetCorridorSpeedRetain)
		shaped[i] = Scale(mixed, finalSpeed/mixedLen)
	}

	return shaped
}

func eligibleJetSeedCell(
	i int,
	elevation []float64,
	seaLevelThreshold float64,
	coastLandDirs []Vector3D,
	westernBoundary []float64,
	gatewayStrength []float64,
	componentAssignments []int,
) bool {
	if i < 0 || i >= len(elevation) || elevation[i] >= seaLevelThreshold {
		return false
	}
	if i >= len(coastLandDirs) || LengthSq(coastLandDirs[i]) < 1e-12 {
		return false
	}
	if i >= len(westernBoundary) || westernBoundary[i] < jetCorridorSeedWest {
		return false
	}
	if i >= len(gatewayStrength) || gatewayStrength[i] > jetCorridorMaxGateway {
		return false
	}
	return componentAssignments[i] >= 0
}

func eligibleJetCorridorCell(
	i int,
	elevation []float64,
	seaLevelThreshold float64,
	coastLandDirs []Vector3D,
	westernBoundary []float64,
	gatewayStrength []float64,
	componentAssignments []int,
) bool {
	if i < 0 || i >= len(elevation) || elevation[i] >= seaLevelThreshold {
		return false
	}
	if i >= len(coastLandDirs) || LengthSq(coastLandDirs[i]) < 1e-12 {
		return false
	}
	if i >= len(westernBoundary) || westernBoundary[i] < jetCorridorAffectWest {
		return false
	}
	if i >= len(gatewayStrength) || gatewayStrength[i] > jetCorridorMaxGateway {
		return false
	}
	return componentAssignments[i] >= 0
}

func coastParallelAxis(vertex Vector3D, landDir Vector3D, current Vector3D) Vector3D {
	axis := Cross(vertex, landDir)
	axisLen := Length(axis)
	if axisLen < 1e-9 {
		return Vector3D{}
	}
	axis = Scale(axis, 1.0/axisLen)
	if Dot(current, axis) < 0 {
		axis = Scale(axis, -1)
	}
	return axis
}
