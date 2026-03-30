package climgen

import "math"

const (
	oceanInteriorDistanceKm   = 1800.0
	minGatewayFlowScale       = 0.40
	westernBoundaryBoostScale = 1.35
	westernBoundaryDampScale  = 0.25
)

// BuildOceanOpennessField estimates how open each water cell is to basin-scale
// flow. Narrow gateways and cramped marginal seas get lower values than broad
// open-ocean interiors.
func BuildOceanOpennessField(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	componentAssignments []int,
	components []OceanComponent,
) []float64 {
	openness := make([]float64, len(vertices))
	if len(vertices) == 0 {
		return openness
	}

	componentScale := BuildComponentScaleField(vertices, componentAssignments, components)
	interior := ComputeSurfaceInteriorFraction(
		elevation, seaLevelThreshold, adj, oceanInteriorDistanceKm, false,
	)

	for i := range vertices {
		if elevation[i] >= seaLevelThreshold {
			continue
		}

		sameWater := 0
		totalNeighbors := 0
		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= len(vertices) {
				continue
			}
			totalNeighbors++
			if elevation[k] < seaLevelThreshold && componentAssignments[k] == componentAssignments[i] {
				sameWater++
			}
		}

		neighborRatio := 0.0
		if totalNeighbors > 0 {
			neighborRatio = float64(sameWater) / float64(totalNeighbors)
		}

		openScore := math.Sqrt(Clamp(neighborRatio, 0, 1))
		openScore *= 0.35 + 0.65*Clamp(interior[i], 0, 1)
		openScore = Clamp(0.10+0.90*openScore, 0, 1)

		if len(componentScale) == len(vertices) {
			openScore *= 0.35 + 0.65*componentScale[i]
		}

		openness[i] = Clamp(openScore, 0, 1)
	}

	return openness
}

// ApplyStructuredCurrentScaling damps flow through cramped gateways and boosts
// poleward western-boundary currents.
func ApplyStructuredCurrentScaling(
	currents []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	westernBoundary []float64,
	openness []float64,
	gatewayStrength []float64,
	gatewayAxis []Vector3D,
) []Vector3D {
	scaled := make([]Vector3D, len(currents))
	copy(scaled, currents)

	for i, current := range scaled {
		if elevation[i] >= seaLevelThreshold {
			continue
		}

		speed := Length(current)
		if speed < 1e-9 {
			continue
		}

		openScale := 1.0
		if i < len(openness) {
			openScale = minGatewayFlowScale + (1.0-minGatewayFlowScale)*Clamp(openness[i], 0, 1)
		}

		if i < len(gatewayStrength) && gatewayStrength[i] > 0 && i < len(gatewayAxis) && LengthSq(gatewayAxis[i]) > 1e-12 {
			axis := gatewayAxis[i]
			alongMag := Dot(current, axis)
			along := Scale(axis, alongMag)
			cross := Sub(current, along)

			gate := Clamp(gatewayStrength[i], 0, 1)
			alongScale := 0.55 + 0.35*Clamp(openness[i], 0, 1) + 0.10*gate
			crossScale := alongScale * (1.0 - 0.85*gate)
			current = Add(Scale(along, alongScale), Scale(cross, crossScale))
			speed = Length(current)
			openScale = 1.0
		}

		westScale := 1.0
		if i < len(westernBoundary) {
			_, north := GetTangentVectors(vertices[i])
			poleward := Dot(current, north)
			if vertices[i].Y < 0 {
				poleward = -poleward
			}
			polewardFrac := Clamp(poleward/speed, -1, 1)

			if polewardFrac > 0 {
				westScale += westernBoundaryBoostScale * Clamp(westernBoundary[i], 0, 1) * polewardFrac
			} else {
				westScale -= westernBoundaryDampScale * Clamp(westernBoundary[i], 0, 1) * -polewardFrac
			}
		}

		if westScale < 0.3 {
			westScale = 0.3
		}

		scaled[i] = Scale(current, openScale*westScale)
	}

	return scaled
}

func ApplyOceanStructure(
	currents []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	coastLandDirs []Vector3D,
	componentAssignments []int,
	components []OceanComponent,
) []Vector3D {
	if len(currents) != len(vertices) {
		return currents
	}

	westernBoundary := ComputeWesternBoundaryLayer(
		vertices, elevation, seaLevelThreshold, adj,
	)
	openness := BuildOceanOpennessField(
		vertices, elevation, seaLevelThreshold, adj, componentAssignments, components,
	)
	gatewayStrength, gatewayAxis := BuildGatewayField(
		vertices, elevation, seaLevelThreshold, adj, componentAssignments, openness,
	)

	structured := ApplyStructuredCurrentScaling(
		currents, vertices, elevation, seaLevelThreshold, westernBoundary, openness, gatewayStrength, gatewayAxis,
	)
	structured = ApplyWesternBoundaryJetShaping(
		structured, vertices, elevation, seaLevelThreshold, coastLandDirs, westernBoundary,
	)
	structured = ApplyWesternBoundaryJetCorridor(
		structured, vertices, elevation, seaLevelThreshold, coastLandDirs, westernBoundary, gatewayStrength, componentAssignments,
	)
	structured = ApplyGyreInteriorGuidance(
		structured, vertices, elevation, seaLevelThreshold, componentAssignments, openness, westernBoundary, gatewayStrength,
	)
	return SmoothOpenOceanCurrents(
		structured, vertices, elevation, seaLevelThreshold, adj, openness, westernBoundary, gatewayStrength,
	)
}
