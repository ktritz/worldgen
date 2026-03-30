package terrain

import (
	"math"
	"sort"
)

const majorLandmassFraction = 0.01

type landComponent struct {
	Cells            []int
	Size             int
	ShorelineEdges   int
	ShorelineLength  float64
	CoastlineSiteSet map[int]struct{}
}

func computeSpatialMetrics(metrics *TerrainMetrics, sites []Vector3D, cells []VoronoiCell, elevation []float64) {
	computeReliefMetrics(metrics, cells, elevation)
	computeDrainageMetrics(metrics, cells, elevation)

	components := findLandComponents(sites, cells, elevation)
	if len(components) == 0 {
		return
	}

	computeContinentMetrics(metrics, components, len(cells))
	computeCoastlineMetrics(metrics, sites, cells, components, len(cells))
}

func findLandComponents(sites []Vector3D, cells []VoronoiCell, elevation []float64) []landComponent {
	visited := make([]bool, len(cells))
	components := make([]landComponent, 0)
	hasSites := len(sites) == len(cells)

	for start := range cells {
		if visited[start] || !IsLand(elevation[start]) {
			continue
		}

		component := landComponent{
			Cells:            make([]int, 0),
			CoastlineSiteSet: make(map[int]struct{}),
		}

		queue := []int{start}
		visited[start] = true

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]

			component.Cells = append(component.Cells, current)
			component.Size++

			for _, neighborSite := range cells[current].NeighborSiteIndices {
				neighbor := int(neighborSite)
				if neighbor < 0 || neighbor >= len(cells) {
					continue
				}

				if IsLand(elevation[neighbor]) {
					if !visited[neighbor] {
						visited[neighbor] = true
						queue = append(queue, neighbor)
					}
					continue
				}

				component.ShorelineEdges++
				component.CoastlineSiteSet[current] = struct{}{}
				component.CoastlineSiteSet[neighbor] = struct{}{}
				if hasSites {
					component.ShorelineLength += angularDistance(sites[current], sites[neighbor])
				}
			}
		}

		components = append(components, component)
	}

	return components
}

func computeContinentMetrics(metrics *TerrainMetrics, components []landComponent, totalCells int) {
	if totalCells == 0 {
		return
	}

	totalLandCells := 0
	largestLandmass := 0
	majorThreshold := int(math.Ceil(majorLandmassFraction * float64(totalCells)))
	if majorThreshold < 1 {
		majorThreshold = 1
	}

	majorSizes := make([]float64, 0)
	for _, component := range components {
		totalLandCells += component.Size
		if component.Size > largestLandmass {
			largestLandmass = component.Size
		}
		if component.Size >= majorThreshold {
			majorSizes = append(majorSizes, float64(component.Size))
		}
	}

	if totalLandCells > 0 {
		metrics.LargestContinentPct = float64(largestLandmass) / float64(totalLandCells)
	}

	metrics.NumMajorLandmasses = len(majorSizes)
	metrics.ContinentGini = computeGini(majorSizes)
}

func computeCoastlineMetrics(
	metrics *TerrainMetrics,
	sites []Vector3D,
	cells []VoronoiCell,
	components []landComponent,
	totalCells int,
) {
	if totalCells == 0 {
		return
	}

	majorThreshold := int(math.Ceil(majorLandmassFraction * float64(totalCells)))
	if majorThreshold < 1 {
		majorThreshold = 1
	}

	avgCellArea := 4 * math.Pi / float64(totalCells)
	totalActualPerimeter := 0.0
	totalEquivalentPerimeter := 0.0
	coastlineSites := make(map[int]struct{})
	hasSites := len(sites) == len(cells)
	if !hasSites {
		return
	}

	for _, component := range components {
		if component.Size < majorThreshold {
			continue
		}

		for idx := range component.CoastlineSiteSet {
			coastlineSites[idx] = struct{}{}
		}

		totalActualPerimeter += component.ShorelineLength

		componentArea := float64(component.Size) * avgCellArea
		totalEquivalentPerimeter += equivalentSphericalPerimeter(componentArea)
	}

	if totalEquivalentPerimeter > 0 {
		metrics.TortuosityRatio = totalActualPerimeter / totalEquivalentPerimeter
	}

	if hasSites && len(coastlineSites) >= 8 {
		shoreline := make([]int, 0, len(coastlineSites))
		for idx := range coastlineSites {
			shoreline = append(shoreline, idx)
		}
		metrics.FractalDimension = computeCoastlineFractalDimension(sites, shoreline)
	}
}

func computeReliefMetrics(metrics *TerrainMetrics, cells []VoronoiCell, elevation []float64) {
	localRelief := make([]float64, 0, len(cells))
	mountainCells := 0
	clusteredMountains := 0

	for i, cell := range cells {
		minElev := elevation[i]
		maxElev := elevation[i]
		mountainNeighbors := 0

		for _, neighborSite := range cell.NeighborSiteIndices {
			neighbor := int(neighborSite)
			if neighbor < 0 || neighbor >= len(elevation) {
				continue
			}

			if elevation[neighbor] < minElev {
				minElev = elevation[neighbor]
			}
			if elevation[neighbor] > maxElev {
				maxElev = elevation[neighbor]
			}
			if elevation[neighbor] >= 2000 {
				mountainNeighbors++
			}
		}

		localRelief = append(localRelief, maxElev-minElev)

		if IsMountain(elevation[i]) {
			mountainCells++
			if mountainNeighbors >= 2 {
				clusteredMountains++
			}
		}
	}

	if len(localRelief) > 0 {
		sort.Float64s(localRelief)
		sum := 0.0
		for _, relief := range localRelief {
			sum += relief
		}
		metrics.MeanLocalRelief = sum / float64(len(localRelief))

		p95Index := int(math.Ceil(0.95*float64(len(localRelief)))) - 1
		if p95Index < 0 {
			p95Index = 0
		}
		if p95Index >= len(localRelief) {
			p95Index = len(localRelief) - 1
		}
		metrics.P95LocalRelief = localRelief[p95Index]
	}

	if mountainCells > 0 {
		metrics.MountainClustered = float64(clusteredMountains) / float64(mountainCells)
	}
}

func equivalentSphericalPerimeter(area float64) float64 {
	if area <= 0 {
		return 0
	}

	cosRadius := Clamp(1-area/(2*math.Pi), -1, 1)
	sinRadius := math.Sqrt(math.Max(0, 1-cosRadius*cosRadius))
	return 2 * math.Pi * sinRadius
}

func computeCoastlineFractalDimension(sites []Vector3D, coastlineSites []int) float64 {
	resolutions := []int{8, 16, 32, 64}
	logScales := make([]float64, 0, len(resolutions))
	logCounts := make([]float64, 0, len(resolutions))

	for _, resolution := range resolutions {
		count := countOccupiedLatLonBoxes(sites, coastlineSites, resolution)
		if count <= 1 {
			continue
		}
		logScales = append(logScales, math.Log(float64(resolution)))
		logCounts = append(logCounts, math.Log(float64(count)))
	}

	if len(logScales) < 2 {
		return 0
	}

	return linearRegressionSlope(logScales, logCounts)
}

func countOccupiedLatLonBoxes(sites []Vector3D, coastlineSites []int, resolution int) int {
	if resolution <= 0 {
		return 0
	}

	width := resolution * 2
	occupied := make(map[int]struct{}, len(coastlineSites))

	for _, idx := range coastlineSites {
		site := sites[idx].Normalize()
		lat := math.Asin(Clamp(site.Z, -1, 1))
		lon := math.Atan2(site.Y, site.X)
		if lon < 0 {
			lon += 2 * math.Pi
		}

		latBin := int(((lat + math.Pi/2) / math.Pi) * float64(resolution))
		lonBin := int((lon / (2 * math.Pi)) * float64(width))

		if latBin >= resolution {
			latBin = resolution - 1
		}
		if lonBin >= width {
			lonBin = width - 1
		}

		occupied[lonBin*resolution+latBin] = struct{}{}
	}

	return len(occupied)
}

func linearRegressionSlope(xs, ys []float64) float64 {
	if len(xs) != len(ys) || len(xs) < 2 {
		return 0
	}

	xMean := 0.0
	yMean := 0.0
	for i := range xs {
		xMean += xs[i]
		yMean += ys[i]
	}
	xMean /= float64(len(xs))
	yMean /= float64(len(ys))

	numerator := 0.0
	denominator := 0.0
	for i := range xs {
		dx := xs[i] - xMean
		numerator += dx * (ys[i] - yMean)
		denominator += dx * dx
	}

	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func computeGini(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)

	total := 0.0
	for _, value := range sorted {
		total += value
	}
	if total == 0 {
		return 0
	}

	weightedSum := 0.0
	for i, value := range sorted {
		weightedSum += float64(i+1) * value
	}

	n := float64(len(sorted))
	return (2*weightedSum)/(n*total) - (n+1)/n
}

func angularDistance(a, b Vector3D) float64 {
	dot := Clamp(a.Normalize().Dot(b.Normalize()), -1, 1)
	return math.Acos(dot)
}
