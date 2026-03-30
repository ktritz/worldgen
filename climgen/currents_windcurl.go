package climgen

import (
	"math"
	"sort"
)

const (
	windCurlBlendActual      = 0.55
	windCurlAnomalyBlend     = 0.20
	windStressSmoothAngular  = 0.12
	windCurlSmoothAngular    = 0.08
	windCurlLatitudeBands    = 18
	windCurlPercentileTarget = 0.90
	windCurlMeridionalWeight = 0.25
)

// GenerateWindDrivenStreamfunctionFromWind computes the Sverdrup streamfunction
// using curl derived from the actual surface wind field. A small idealized
// latitude-only component is retained to stabilize large-scale gyres.
func GenerateWindDrivenStreamfunctionFromWind(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	componentAssignments []int,
	components []OceanComponent,
	strength float64,
) []float64 {
	if len(wind) != len(vertices) {
		return GenerateWindDrivenStreamfunction(vertices, elevation, seaLevelThreshold, adj, componentAssignments, components, strength)
	}

	numVertices := len(vertices)
	psi := make([]float64, numVertices)
	componentScale := BuildComponentScaleField(vertices, componentAssignments, components)
	openness := BuildOceanOpennessField(
		vertices, elevation, seaLevelThreshold, adj, componentAssignments, components,
	)
	isWater, isCoastline := classifyOceanVertices(elevation, seaLevelThreshold, adj)
	eastFetch := ComputeEasternBoundaryFetchByComponent(
		vertices, elevation, seaLevelThreshold, adj, componentAssignments, components,
	)
	windCurl := ComputeNormalizedWindStressCurl(vertices, elevation, seaLevelThreshold, adj, wind)

	for i := 0; i < numVertices; i++ {
		if !isWater[i] || isCoastline[i] {
			continue
		}

		lat := math.Asin(vertices[i].Y)
		cosLat := math.Cos(lat)
		if cosLat < 0.1 {
			cosLat = 0.1
		}
		beta := cosLat

		idealCurl := idealWindCurlForLatitude(lat)
		blend := adaptiveWindCurlBlend(i, openness)
		combinedCurl := idealCurl + blend*(windCurl[i]-idealCurl)
		scale := 1.0
		if len(componentScale) == len(vertices) {
			scale = componentScale[i]
		}
		psi[i] = strength * combinedCurl / beta * eastFetch[i] * scale
	}

	return psi
}

func adaptiveWindCurlBlend(i int, openness []float64) float64 {
	blend := windCurlBlendActual
	if i >= 0 && i < len(openness) {
		blend = 0.20 + 0.35*Clamp(openness[i], 0, 1)
	}
	return Clamp(blend, 0, windCurlBlendActual)
}

// ComputeNormalizedWindStressCurl derives a vertical wind-stress curl proxy from
// the surface wind field and rescales it to an Earth-like [-1, 1] band.
func ComputeNormalizedWindStressCurl(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
) []float64 {
	numVertices := len(vertices)
	stressEast := make([]float64, numVertices)
	stressNorth := make([]float64, numVertices)
	for i := range vertices {
		if elevation[i] >= seaLevelThreshold {
			continue
		}
		east, north := GetTangentVectors(vertices[i])
		stress := windStressComponents(wind[i], east, north)
		stressEast[i] = stress.east
		stressNorth[i] = stress.north
	}

	cellSize := estimateCellSize(vertices, adj)
	stressSmoothIters := int(windStressSmoothAngular/cellSize) + 1
	if stressSmoothIters < 3 {
		stressSmoothIters = 3
	}
	stressEast = smoothOceanScalarField(stressEast, elevation, seaLevelThreshold, adj, stressSmoothIters, 0.45)
	stressNorth = smoothOceanScalarField(stressNorth, elevation, seaLevelThreshold, adj, stressSmoothIters, 0.45)

	rawCurl := make([]float64, numVertices)

	for i := range vertices {
		if elevation[i] >= seaLevelThreshold {
			continue
		}

		east, north := GetTangentVectors(vertices[i])
		selfStress := localWindStress{
			east:  stressEast[i],
			north: stressNorth[i],
		}

		var sumTauNDe, sumTauEDn float64
		var sumWe, sumWn float64
		waterNeighbors := 0

		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= len(vertices) || elevation[k] >= seaLevelThreshold {
				continue
			}
			waterNeighbors++

			diff := Sub(vertices[k], vertices[i])
			dotN := Dot(diff, vertices[i])
			tangentDiff := Sub(diff, Scale(vertices[i], dotN))
			de := Dot(tangentDiff, east)
			dn := Dot(tangentDiff, north)
			dist := math.Sqrt(de*de + dn*dn)
			if dist < 1e-12 {
				continue
			}

			neighborStress := localWindStress{
				east:  stressEast[k],
				north: stressNorth[k],
			}
			dTauE := neighborStress.east - selfStress.east
			dTauN := neighborStress.north - selfStress.north
			weight := 1.0 / dist

			sumTauNDe += weight * dTauN * de / dist
			sumTauEDn += weight * dTauE * dn / dist
			sumWe += weight * de * de / (dist * dist)
			sumWn += weight * dn * dn / (dist * dist)
		}

		if waterNeighbors < 2 {
			continue
		}

		var dTauNDe, dTauEDn float64
		if sumWe > 1e-12 {
			dTauNDe = sumTauNDe / sumWe
		}
		if sumWn > 1e-12 {
			dTauEDn = sumTauEDn / sumWn
		}
		rawCurl[i] = windCurlMeridionalWeight*dTauNDe - dTauEDn
	}

	curlSmoothIters := int(windCurlSmoothAngular/cellSize) + 1
	if curlSmoothIters < 2 {
		curlSmoothIters = 2
	}
	smoothed := smoothOceanScalarField(rawCurl, elevation, seaLevelThreshold, adj, curlSmoothIters, 0.4)
	zonalMean := computeLatitudeBandMeans(vertices, elevation, seaLevelThreshold, smoothed, windCurlLatitudeBands)
	blended := make([]float64, len(smoothed))
	for i, v := range smoothed {
		if elevation[i] >= seaLevelThreshold {
			continue
		}
		bandMean := zonalMean[latitudeBandIndex(vertices[i], windCurlLatitudeBands)]
		blended[i] = bandMean + windCurlAnomalyBlend*(v-bandMean)
	}

	scale := percentileAbs(blended, elevation, seaLevelThreshold, windCurlPercentileTarget)
	if scale < 1e-9 {
		return blended
	}

	normalized := make([]float64, len(blended))
	for i, v := range blended {
		if elevation[i] >= seaLevelThreshold {
			continue
		}
		normalized[i] = Clamp(v/scale, -1, 1)
	}
	return normalized
}

type localWindStress struct {
	east  float64
	north float64
}

func windStressComponents(wind Vector3D, east, north Vector3D) localWindStress {
	speed := Length(wind)
	if speed < 1e-9 {
		return localWindStress{}
	}
	return localWindStress{
		east:  Dot(wind, east) * speed,
		north: Dot(wind, north) * speed,
	}
}

func idealWindCurlForLatitude(lat float64) float64 {
	return math.Sin(3.0*lat) * math.Cos(lat)
}

func classifyOceanVertices(
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
) ([]bool, []bool) {
	numVertices := len(elevation)
	isWater := make([]bool, numVertices)
	isCoastline := make([]bool, numVertices)
	for i := 0; i < numVertices; i++ {
		isWater[i] = elevation[i] < seaLevelThreshold
		if !isWater[i] {
			continue
		}
		for _, k := range adj.GetNeighbors(i) {
			if k >= 0 && k < numVertices && elevation[k] >= seaLevelThreshold {
				isCoastline[i] = true
				break
			}
		}
	}
	return isWater, isCoastline
}

func smoothOceanScalarField(
	field []float64,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	iterations int,
	factor float64,
) []float64 {
	smoothed := make([]float64, len(field))
	copy(smoothed, field)
	next := make([]float64, len(field))

	for iter := 0; iter < iterations; iter++ {
		for i := range field {
			if elevation[i] >= seaLevelThreshold {
				next[i] = 0
				continue
			}

			sum := 0.0
			count := 0
			for _, k := range adj.GetNeighbors(i) {
				if k >= 0 && k < len(field) && elevation[k] < seaLevelThreshold {
					sum += smoothed[k]
					count++
				}
			}
			if count > 0 {
				avg := sum / float64(count)
				next[i] = smoothed[i]*(1-factor) + avg*factor
			} else {
				next[i] = smoothed[i]
			}
		}
		smoothed, next = next, smoothed
	}

	return smoothed
}

func computeLatitudeBandMeans(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	field []float64,
	bands int,
) []float64 {
	if bands < 1 {
		bands = 1
	}
	sums := make([]float64, bands)
	counts := make([]int, bands)
	for i, v := range field {
		if elevation[i] >= seaLevelThreshold {
			continue
		}
		band := latitudeBandIndex(vertices[i], bands)
		sums[band] += v
		counts[band]++
	}

	means := make([]float64, bands)
	for i := range means {
		if counts[i] > 0 {
			means[i] = sums[i] / float64(counts[i])
		}
	}

	smoothed := make([]float64, bands)
	copy(smoothed, means)
	for iter := 0; iter < 2; iter++ {
		for i := range means {
			sum := smoothed[i]
			count := 1.0
			if i > 0 {
				sum += smoothed[i-1]
				count++
			}
			if i+1 < bands {
				sum += smoothed[i+1]
				count++
			}
			means[i] = sum / count
		}
		copy(smoothed, means)
	}
	return smoothed
}

func latitudeBandIndex(v Vector3D, bands int) int {
	lat := math.Asin(v.Y)
	t := (lat + math.Pi/2) / math.Pi
	idx := int(t * float64(bands))
	if idx < 0 {
		return 0
	}
	if idx >= bands {
		return bands - 1
	}
	return idx
}

func percentileAbs(
	field []float64,
	elevation []float64,
	seaLevelThreshold float64,
	percentile float64,
) float64 {
	if percentile <= 0 {
		percentile = 0.5
	}
	if percentile > 1 {
		percentile = 1
	}

	values := make([]float64, 0, len(field))
	for i, v := range field {
		if elevation[i] >= seaLevelThreshold {
			continue
		}
		values = append(values, math.Abs(v))
	}
	if len(values) == 0 {
		return 0
	}

	sort.Float64s(values)

	idx := int(percentile * float64(len(values)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return values[idx]
}
