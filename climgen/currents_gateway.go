package climgen

import "math"

const gatewayStrengthThreshold = 0.08

// BuildGatewayField identifies likely straits/chokepoints and estimates the
// local along-strait axis for coherent gateway transport.
func BuildGatewayField(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	componentAssignments []int,
	openness []float64,
) ([]float64, []Vector3D) {
	strength := make([]float64, len(vertices))
	axis := make([]Vector3D, len(vertices))

	for i := range vertices {
		if elevation[i] >= seaLevelThreshold {
			continue
		}

		east, north := GetTangentVectors(vertices[i])
		type dir2 struct{ x, y float64 }
		var dirs []dir2
		waterNeighbors := 0
		totalNeighbors := 0

		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= len(vertices) {
				continue
			}
			totalNeighbors++
			if elevation[k] >= seaLevelThreshold {
				continue
			}
			if len(componentAssignments) == len(vertices) && componentAssignments[k] != componentAssignments[i] {
				continue
			}

			diff := Sub(vertices[k], vertices[i])
			dotN := Dot(diff, vertices[i])
			tangent := Sub(diff, Scale(vertices[i], dotN))
			tLen := Length(tangent)
			if tLen < 1e-9 {
				continue
			}
			waterNeighbors++
			tangent = Scale(tangent, 1.0/tLen)
			dirs = append(dirs, dir2{x: Dot(tangent, east), y: Dot(tangent, north)})
		}

		if len(dirs) < 2 || totalNeighbors == 0 {
			continue
		}

		var xx, xy, yy float64
		for _, d := range dirs {
			xx += d.x * d.x
			xy += d.x * d.y
			yy += d.y * d.y
		}

		trace := xx + yy
		det := xx*yy - xy*xy
		disc := trace*trace/4 - det
		if disc < 0 {
			disc = 0
		}
		root := math.Sqrt(disc)
		l1 := trace/2 + root
		l2 := trace/2 - root
		if l1 < 1e-9 {
			continue
		}

		aniso := Clamp((l1-l2)/(l1+l2), 0, 1)
		if aniso < 0.2 {
			continue
		}

		axX, axY := 1.0, 0.0
		if math.Abs(xy) > 1e-9 || math.Abs(l1-xx) > 1e-9 {
			axX = xy
			axY = l1 - xx
		}
		axLen := math.Hypot(axX, axY)
		if axLen < 1e-9 {
			continue
		}
		axX /= axLen
		axY /= axLen

		pos := 0.0
		neg := 0.0
		for _, d := range dirs {
			proj := d.x*axX + d.y*axY
			switch {
			case proj > 0.25:
				pos += proj
			case proj < -0.25:
				neg += -proj
			}
		}
		if pos < 0.2 || neg < 0.2 {
			continue
		}

		neighborRatio := float64(waterNeighbors) / float64(totalNeighbors)
		bilateral := math.Min(pos, neg) / math.Max(pos, neg)
		constriction := 0.5 + 0.5*math.Sqrt(Clamp(1.0-neighborRatio, 0, 1))
		score := aniso * bilateral * constriction
		if score < gatewayStrengthThreshold {
			continue
		}

		strength[i] = Clamp(score, 0, 1)
		axis[i] = Normalize(Add(Scale(east, axX), Scale(north, axY)))
	}

	return strength, axis
}
