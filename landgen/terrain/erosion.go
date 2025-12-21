package terrain

import (
	"runtime"
	"sync"
)

// Erosion and smoothing passes for terrain
// Applied before hypsometric mapping to create more natural mountain ranges

// LandmassInfo stores info about a connected landmass for erosion calculations
type LandmassInfo struct {
	ID           int
	Size         int     // Number of cells
	MaxDistCoast float64 // Maximum distance from coast (interior depth)
}

// ApplyThermalErosion smooths elevation by averaging with neighbors
// Simulates material sliding down steep slopes over time
// iterations: number of passes (2-5 typical)
// strength: how much to blend with neighbors (0.0-1.0, typically 0.3-0.5)
func ApplyThermalErosion(
	cells []VoronoiCell,
	elevation []float64,
	rPlate []int,
	plateIsOcean map[int]bool,
	iterations int,
	strength float64,
) {
	numRegions := len(elevation)
	buffer := make([]float64, numRegions)

	for iter := 0; iter < iterations; iter++ {
		copy(buffer, elevation)

		for r := 0; r < numRegions; r++ {
			isOcean := plateIsOcean[rPlate[r]]

			// Average with same-type neighbors (don't smooth across coastlines)
			sum := elevation[r]
			count := 1.0

			for _, neighborIdx := range cells[r].NeighborSiteIndices {
				neighborR := int(neighborIdx)
				if neighborR >= numRegions {
					continue
				}

				neighborIsOcean := plateIsOcean[rPlate[neighborR]]

				// Only smooth within same domain (land-land or ocean-ocean)
				if isOcean == neighborIsOcean {
					sum += elevation[neighborR]
					count++
				}
			}

			avg := sum / count
			buffer[r] = elevation[r]*(1-strength) + avg*strength
		}

		copy(elevation, buffer)
	}
}

// ApplySelectiveErosion applies stronger erosion to peaks, preserving valleys
// This creates more realistic mountain ranges with gradual slopes
// Uses parallel processing for better performance
func ApplySelectiveErosion(
	cells []VoronoiCell,
	elevation []float64,
	rPlate []int,
	plateIsOcean map[int]bool,
	iterations int,
) {
	numRegions := len(elevation)
	buffer := make([]float64, numRegions)
	numWorkers := runtime.NumCPU()

	for iter := 0; iter < iterations; iter++ {
		copy(buffer, elevation)

		var wg sync.WaitGroup
		chunkSize := (numRegions + numWorkers - 1) / numWorkers

		for w := 0; w < numWorkers; w++ {
			start := w * chunkSize
			end := start + chunkSize
			if end > numRegions {
				end = numRegions
			}

			wg.Add(1)
			go func(start, end int) {
				defer wg.Done()
				for r := start; r < end; r++ {
					isOcean := plateIsOcean[rPlate[r]]
					currentElev := elevation[r]

					// Find min and max neighbor elevation (same domain only)
					minNeighbor := currentElev
					maxNeighbor := currentElev
					sum := currentElev
					count := 1.0

					for _, neighborIdx := range cells[r].NeighborSiteIndices {
						neighborR := int(neighborIdx)
						if neighborR >= numRegions {
							continue
						}

						neighborIsOcean := plateIsOcean[rPlate[neighborR]]
						if isOcean == neighborIsOcean {
							neighborElev := elevation[neighborR]
							sum += neighborElev
							count++
							if neighborElev < minNeighbor {
								minNeighbor = neighborElev
							}
							if neighborElev > maxNeighbor {
								maxNeighbor = neighborElev
							}
						}
					}

					avg := sum / count

					// Stronger erosion for peaks (cells higher than all neighbors)
					// Weaker erosion for valleys
					if currentElev > maxNeighbor-0.01 {
						// Peak - erode more strongly
						buffer[r] = currentElev*0.6 + avg*0.4
					} else if currentElev < minNeighbor+0.01 {
						// Valley - preserve
						buffer[r] = currentElev*0.9 + avg*0.1
					} else {
						// Slope - moderate erosion
						buffer[r] = currentElev*0.75 + avg*0.25
					}
				}
			}(start, end)
		}

		wg.Wait()
		copy(elevation, buffer)
	}
}

// ApplyLandmassErosion caps mountain elevations based on ocean proximity
// Mountains surrounded by ocean can't be as tall as continental interior mountains
// Uses a radius check: counts what fraction of nearby cells are ocean
// Resolution-independent: uses angular distance, not cell counts
func ApplyLandmassErosion(
	cells []VoronoiCell,
	elevation []float64,
	rPlate []int,
	plateIsOcean map[int]bool,
	distFromCoast []float64,
) {
	numRegions := len(elevation)

	// For each land cell, count ocean cells within a radius using BFS
	// Radius of ~1000km - large enough to detect isthmuses and small landmasses
	const checkDepth = 15 // ~15 neighbor hops at level 7 ≈ 1000km radius

	for r := 0; r < numRegions; r++ {
		// Only process cells above sea level
		if elevation[r] <= 0 {
			continue
		}

		// BFS to count land vs ocean cells within radius
		visited := make(map[int]bool)
		queue := []int{r}
		visited[r] = true

		landCount := 0
		oceanCount := 0

		for depth := 0; depth < checkDepth && len(queue) > 0; depth++ {
			nextQueue := []int{}
			for _, current := range queue {
				// Count this cell
				if elevation[current] > 0 {
					landCount++
				} else {
					oceanCount++
				}

				// Add unvisited neighbors
				for _, neighborIdx := range cells[current].NeighborSiteIndices {
					neighborR := int(neighborIdx)
					if neighborR >= numRegions || visited[neighborR] {
						continue
					}
					visited[neighborR] = true
					nextQueue = append(nextQueue, neighborR)
				}
			}
			queue = nextQueue
		}

		// Calculate land fraction in the radius
		totalChecked := landCount + oceanCount
		if totalChecked == 0 {
			continue
		}
		landFraction := float64(landCount) / float64(totalChecked)

		// Max elevation scales with land fraction - AGGRESSIVE scaling
		// Need very high land fraction to support tall mountains
		// Isolated island (landFraction < 0.2): max 200-400m
		// Small landmass/isthmus (0.2-0.5): max 400-1500m
		// Coastal/peninsula (0.5-0.8): max 1500-3500m
		// Continental interior (0.8-1.0): max 3500-6000m
		var maxElev float64
		if landFraction < 0.2 {
			// Tiny island
			maxElev = 200 + 1000*landFraction // 200-400m
		} else if landFraction < 0.5 {
			// Small landmass or isthmus
			maxElev = 400 + 3667*(landFraction-0.2) // 400-1500m
		} else if landFraction < 0.8 {
			// Coastal/peninsula
			maxElev = 1500 + 6667*(landFraction-0.5) // 1500-3500m
		} else {
			// Continental interior
			maxElev = 3500 + 12500*(landFraction-0.8) // 3500-6000m
		}

		// Apply cap
		if elevation[r] > maxElev {
			elevation[r] = maxElev
		}
	}
}
