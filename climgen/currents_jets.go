package climgen

import "math"

const (
	jetAlignmentBlend    = 0.66
	jetSubtropicalBoost  = 0.88
	openOceanSmoothBlend = 0.20
	openOceanMinOpenness = 0.70
)

// ApplyWesternBoundaryJetShaping narrows boundary currents by aligning them
// with the local coastline tangent while preserving the existing flow direction.
// A modest extra boost is applied to subtropical poleward jets.
func ApplyWesternBoundaryJetShaping(
	currents []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	coastLandDirs []Vector3D,
	westernBoundary []float64,
) []Vector3D {
	shaped := make([]Vector3D, len(currents))
	copy(shaped, currents)

	for i, current := range shaped {
		if elevation[i] >= seaLevelThreshold || i >= len(coastLandDirs) || i >= len(westernBoundary) {
			continue
		}
		west := Clamp(westernBoundary[i], 0, 1)
		if west < 0.05 {
			continue
		}

		speed := Length(current)
		landDir := coastLandDirs[i]
		if speed < 1e-9 || LengthSq(landDir) < 1e-12 {
			continue
		}

		axis := Cross(vertices[i], landDir)
		axisLen := Length(axis)
		if axisLen < 1e-9 {
			continue
		}
		axis = Scale(axis, 1.0/axisLen)
		if Dot(current, axis) < 0 {
			axis = Scale(axis, -1)
		}

		alongMag := Dot(current, axis)
		along := Scale(axis, alongMag)
		cross := Sub(current, along)

		blend := jetAlignmentBlend * west
		aligned := Add(along, Scale(cross, 1.0-blend))

		latDeg := math.Abs(getLatitudeDeg(vertices[i]))
		if latDeg >= 12 && latDeg <= 50 {
			_, north := GetTangentVectors(vertices[i])
			poleward := Dot(aligned, north)
			if vertices[i].Y < 0 {
				poleward = -poleward
			}
			if poleward > 0 {
				aligned = Scale(aligned, 1.0+jetSubtropicalBoost*west*Clamp(poleward/Length(aligned), 0, 1))
			}
		}

		shaped[i] = aligned
	}

	return shaped
}

// SmoothOpenOceanCurrents lightly damps residual small-scale texture in broad
// open-ocean interiors while leaving gateways and boundary currents intact.
func SmoothOpenOceanCurrents(
	currents []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	openness []float64,
	westernBoundary []float64,
	gatewayStrength []float64,
) []Vector3D {
	smoothed := make([]Vector3D, len(currents))
	copy(smoothed, currents)

	for i, current := range currents {
		if elevation[i] >= seaLevelThreshold || i >= len(openness) || i >= len(westernBoundary) || i >= len(gatewayStrength) {
			continue
		}
		if openness[i] < openOceanMinOpenness || westernBoundary[i] > 0.18 || gatewayStrength[i] > 0.08 {
			continue
		}

		var sum Vector3D
		count := 0
		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= len(currents) || elevation[k] >= seaLevelThreshold {
				continue
			}
			if openness[k] < openOceanMinOpenness || gatewayStrength[k] > 0.08 {
				continue
			}
			sum = Add(sum, currents[k])
			count++
		}
		if count == 0 {
			continue
		}

		avg := Scale(sum, 1.0/float64(count))
		blended := Add(Scale(current, 1.0-openOceanSmoothBlend), Scale(avg, openOceanSmoothBlend))
		dotNormal := Dot(blended, vertices[i])
		smoothed[i] = Sub(blended, Scale(vertices[i], dotNormal))
	}

	return smoothed
}
