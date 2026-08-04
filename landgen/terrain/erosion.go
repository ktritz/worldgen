package terrain

import (
	"math"
	"runtime"
	"sort"
	"sync"
)

// Erosion and smoothing passes for terrain
// Applied before hypsometric mapping to create more natural mountain ranges

// landmassErosionBaseCheckDepth is the ApplyLandmassErosion ocean-fraction BFS
// radius expressed in *baseline L5* neighbor hops (see meshResolutionAdjustedSteps).
//
// Calibration note: the original hardcoded constant was 15 and its comment read
// "~15 neighbor hops at level 7 ~= 1000km radius" - i.e. it was calibrated at L7,
// not at the L5 baseline the scaling helper assumes. Feeding 15 to
// meshResolutionAdjustedSteps therefore quadrupled the intended physical radius at
// L7 (60 hops) and cost ~(60/16)^2 ~= 14x in the per-land-cell BFS.
//
// A base of 4 reproduces the intended physical radius: mean cell angular radius is
// 2/sqrt(n), so hop count x cell radius is ~constant across levels.
//
//	L4  2562 cells: scale 1.000 ->  4 hops (~ 158 km/hop, uncorrected by design)
//	L5 10242 cells: scale 1.000 ->  4 hops (~ 126 km/hop -> ~ 504 km radius)
//	L6 40962 cells: scale 0.500 ->  8 hops (~  63 km/hop -> ~ 504 km radius)
//	L7 163842 cells: scale 0.250 -> 16 hops (~  31 km/hop -> ~ 504 km radius)
//	L8 655362 cells: scale 0.125 -> 32 hops (~  16 km/hop -> ~ 504 km radius)
//
// 16 hops at L7 is within one hop of the value the constant was originally tuned
// against, so L7 output is essentially unchanged while L5/L6 gain the physical
// radius they should always have had.
const landmassErosionBaseCheckDepth = 4

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
	iterations = meshResolutionAdjustedDiffusionIterations(iterations, numRegions)
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
//
// Cost note (measured 2026-08-02, i7-6700K, 8 threads, 163842-cell mesh):
// meshResolutionAdjustedDiffusionIterations is the correct policy here - a
// nearest-neighbour diffusion pass smooths a fixed number of *cells*, so holding
// the physical smoothing radius constant genuinely requires iterations ~ N. Base
// 3 becomes 3/12/48/192 iterations at L5/L6/L7/L8. Measured wall time for the
// whole call: 1.1 ms (L5), 11.8 ms (L6), 185 ms (L7) - i.e. ~2% of a 7.7 s L7
// terrain run, and ~3 s extrapolated at L8. That is cheap enough that trading
// accuracy for a wider-stencil single pass (which would need explicit k-ring
// neighbourhoods this mesh does not precompute) is not worth it. Keep as is; the
// L7 performance problem was ApplyLandmassErosion's BFS radius, not this.
func ApplySelectiveErosion(
	cells []VoronoiCell,
	elevation []float64,
	rPlate []int,
	plateIsOcean map[int]bool,
	iterations int,
) {
	numRegions := len(elevation)
	iterations = meshResolutionAdjustedDiffusionIterations(iterations, numRegions)
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
// Uses a radius check: counts what fraction of a physical-equivalent nearby
// cell band is ocean.
func ApplyLandmassErosion(
	cells []VoronoiCell,
	elevation []float64,
	rPlate []int,
	plateIsOcean map[int]bool,
	distFromCoast []float64,
	distFromMountain []float64,
) {
	numRegions := len(elevation)

	// For each land cell, count ocean cells within a physical-equivalent radius.
	// Radius of ~1000km - large enough to detect isthmuses and small landmasses.
	checkDepth := meshResolutionAdjustedSteps(landmassErosionBaseCheckDepth, numRegions)
	stepScale := meshPathCostResolutionScale(numRegions)
	visited := make([]int, numRegions)
	stamp := 0

	for r := 0; r < numRegions; r++ {
		// Only process cells above sea level
		if elevation[r] <= 0 {
			continue
		}

		// BFS to count land vs ocean cells within radius
		stamp++
		queue := []int{r}
		visited[r] = stamp

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
					if neighborR < 0 || neighborR >= numRegions || visited[neighborR] == stamp {
						continue
					}
					visited[neighborR] = stamp
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

		// Tectonically supported continental margins can sustain major cordilleras
		// even when they are close to the coast. Relax the cap near collision belts
		// and volcanic arcs, but keep oceanic/island peaks constrained.
		if !plateIsOcean[rPlate[r]] {
			tectonicSupport := hopDistanceSupport(distFromMountain[r]*stepScale, 8)
			maxElev += 2200 * tectonicSupport
		}

		// Apply cap
		if elevation[r] > maxElev {
			elevation[r] = maxElev
		}
	}
}

func hopDistanceSupport(distance float64, maxDistance float64) float64 {
	if math.IsInf(distance, 1) || maxDistance <= 0 {
		return 0
	}
	if distance <= 0 {
		return 1
	}
	norm := distance / maxDistance
	if norm >= 1 {
		return 0
	}
	return 1 - norm
}

// ApplyFluvialErosion carves a drainage network using topography-driven flow
// routing and a long-timescale runoff proxy. It intentionally does not depend
// on the present-day climate model; instead it uses broad wet/dry bands,
// coastal access to moisture, and low-frequency noise to approximate
// geological-time erosional potential.
func ApplyFluvialErosion(
	sites []Vector3D,
	cells []VoronoiCell,
	elevation []float64,
	distFromCoast []float64,
	maxDist float64,
	seed int64,
) {
	runoff := ComputeLongTermRunoffProxy(sites, elevation, distFromCoast, maxDist, seed)
	receivers := ComputeDrainageReceivers(cells, elevation)
	BreachDrainageSinks(cells, elevation, receivers, 220)
	receivers = ComputeDrainageReceivers(cells, elevation)
	accumulation, order := ComputeFlowAccumulation(receivers, elevation, runoff)
	physicalAccumulation := scaleFlowAccumulationForMesh(accumulation, len(cells))
	FormEndorheicLakes(cells, elevation, receivers, physicalAccumulation, runoff)
	receivers = ComputeDrainageReceivers(cells, elevation)
	accumulation, order = ComputeFlowAccumulation(receivers, elevation, runoff)
	physicalAccumulation = scaleFlowAccumulationForMesh(accumulation, len(cells))
	carveFluvialChannels(cells, elevation, receivers, runoff, physicalAccumulation)
	applyFluvialDeposition(cells, elevation, receivers, runoff, physicalAccumulation, order)
}

// ApplyPostDetailDrainageConditioning removes shallow synthetic traps created
// by late-stage noise and coastline regularization without trying to erase real
// endorheic structure. It is intentionally milder than the main fluvial pass.
func ApplyPostDetailDrainageConditioning(cells []VoronoiCell, elevation []float64) int {
	receivers := ComputeDrainageReceivers(cells, elevation)
	return BreachDrainageSinks(cells, elevation, receivers, 120)
}

// ComputeLongTermRunoffProxy estimates geological-timescale runoff potential.
// It is intentionally smoother and lower-frequency than a present-day climate
// field so terrain is not overfit to transient weather patterns.
func ComputeLongTermRunoffProxy(
	sites []Vector3D,
	elevation []float64,
	distFromCoast []float64,
	maxDist float64,
	seed int64,
) []float64 {
	runoff := make([]float64, len(elevation))
	if maxDist < 1 {
		maxDist = 1
	}

	for i, elev := range elevation {
		if elev <= 0 {
			continue
		}

		lat := math.Asin(Clamp(sites[i].Normalize().Z, -1, 1))
		absLatDeg := math.Abs(lat * 180 / math.Pi)

		// Broad wet tropical and mid-latitude storm belts, with drier subtropics.
		tropicalWet := 1.0 - SmoothStep(8, 28, absLatDeg)
		midLatitudeWet := SmoothStep(32, 45, absLatDeg) * (1.0 - SmoothStep(58, 72, absLatDeg))
		subtropicalDry := SmoothStep(18, 27, absLatDeg) * (1.0 - SmoothStep(35, 44, absLatDeg))
		polarDry := SmoothStep(62, 80, absLatDeg)
		latMoisture := 0.55 + 0.42*tropicalWet + 0.24*midLatitudeWet - 0.28*subtropicalDry - 0.18*polarDry

		coastalness := 0.0
		if !math.IsInf(distFromCoast[i], 1) {
			coastalNorm := distFromCoast[i] / maxDist
			coastalness = 1.0 - SmoothStep(0.10, 0.85, coastalNorm)
		}

		reliefBoost := SmoothStep(600, 3200, elev)
		lowFreqNoise := FBMNoiseWithFreq(sites[i], seed+868686, 2.0, 3)

		moisture := latMoisture + 0.24*coastalness + 0.10*reliefBoost + 0.12*lowFreqNoise
		runoff[i] = Clamp(moisture, 0.08, 1.35)
	}

	return runoff
}

// ComputeDrainageReceivers selects the steepest downslope neighbor for each
// land cell. Ocean cells and sinks have receiver -1.
func ComputeDrainageReceivers(cells []VoronoiCell, elevation []float64) []int {
	receivers := make([]int, len(elevation))
	for i := range receivers {
		receivers[i] = -1
	}

	for r, elev := range elevation {
		if elev <= 0 {
			continue
		}

		bestNeighbor := -1
		bestDrop := 0.0
		for _, neighborIdx := range cells[r].NeighborSiteIndices {
			neighbor := int(neighborIdx)
			if neighbor < 0 || neighbor >= len(elevation) {
				continue
			}
			drop := elev - elevation[neighbor]
			if drop > bestDrop+1e-6 {
				bestDrop = drop
				bestNeighbor = neighbor
			}
		}
		receivers[r] = bestNeighbor
	}

	return receivers
}

// BreachDrainageSinks lowers shallow outlet saddles so local basins can drain
// instead of creating too many persistent endorheic pits. This is a simple
// geomorphic approximation of spillway incision, not a full depression-filling
// model.
func BreachDrainageSinks(cells []VoronoiCell, elevation []float64, receivers []int, maxRise float64) int {
	if maxRise <= 0 {
		return 0
	}

	// maxRise and the carve depths compare elevations across a SINGLE cell hop,
	// so for a fixed physical slope they scale with cell size. Scale them by the
	// mesh step so breach aggressiveness stays resolution-independent (same
	// convention as physicalSlope := slope / stepScale elsewhere in this file).
	stepScale := meshPathCostResolutionScale(len(cells))
	if stepScale <= 0 {
		stepScale = 1
	}
	maxRise *= stepScale

	breached := 0
	for pass := 0; pass < 2; pass++ {
		passBreached := 0
		for r, receiver := range receivers {
			if receiver >= 0 || elevation[r] <= 0 {
				continue
			}

			lowestNeighbor := -1
			lowestElev := math.Inf(1)
			for _, neighborIdx := range cells[r].NeighborSiteIndices {
				neighbor := int(neighborIdx)
				if neighbor < 0 || neighbor >= len(elevation) {
					continue
				}
				if elevation[neighbor] < lowestElev {
					lowestElev = elevation[neighbor]
					lowestNeighbor = neighbor
				}
			}
			if lowestNeighbor < 0 {
				continue
			}

			rise := lowestElev - elevation[r]
			if rise <= 0 || rise > maxRise {
				continue
			}

			// Carve a shallow spillway through the lowest saddle.
			// maxRise is already step-scaled above, so 0.25*maxRise follows it.
			targetElev := elevation[r] - math.Min(6.0*stepScale, 0.25*maxRise)
			if targetElev >= elevation[lowestNeighbor] {
				targetElev = elevation[r] - 2.0*stepScale
			}
			if targetElev < elevation[lowestNeighbor] {
				elevation[lowestNeighbor] = targetElev
				passBreached++
			}
		}

		if passBreached == 0 {
			break
		}
		breached += passBreached
		receivers = ComputeDrainageReceivers(cells, elevation)
	}

	return breached
}

// FormEndorheicLakes preserves a small number of plausible lowland internal
// basins as lakes instead of breaching every sink away.
func FormEndorheicLakes(
	cells []VoronoiCell,
	elevation []float64,
	receivers []int,
	accumulation []float64,
	runoff []float64,
) int {
	lakeCells := 0
	for r, receiver := range receivers {
		if receiver >= 0 || elevation[r] <= 0 || elevation[r] > 140 {
			continue
		}

		lakeStrength := math.Sqrt(accumulation[r] * math.Max(runoff[r], 0.1))
		if lakeStrength < 35 {
			continue
		}

		targetSurface := -Clamp(6+3.5*math.Log1p(lakeStrength), 6, 22)
		if elevation[r] > targetSurface {
			elevation[r] = targetSurface
			lakeCells++
		}

		lakeCells += spreadEndorheicLakeSurface(cells, elevation, r, targetSurface+6, accumulation[r], accumulation)
	}
	return lakeCells
}

func spreadEndorheicLakeSurface(
	cells []VoronoiCell,
	elevation []float64,
	center int,
	neighborSurface float64,
	centerAccumulation float64,
	accumulation []float64,
) int {
	if center < 0 || center >= len(cells) {
		return 0
	}
	maxHops := meshResolutionAdjustedSteps(1, len(cells))
	type state struct {
		cell int
		hops int
	}
	seen := map[int]struct{}{center: {}}
	queue := []state{{cell: center, hops: 0}}
	changed := 0
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.hops > 0 {
			if cur.cell < 0 || cur.cell >= len(elevation) || elevation[cur.cell] <= 0 || elevation[cur.cell] > 120 {
				continue
			}
			if cur.cell >= len(accumulation) || accumulation[cur.cell] < centerAccumulation*0.25 {
				continue
			}
			if elevation[cur.cell] > neighborSurface {
				elevation[cur.cell] = neighborSurface
				changed++
			}
		}
		if cur.hops >= maxHops {
			continue
		}
		for _, neighborIdx := range cells[cur.cell].NeighborSiteIndices {
			neighbor := int(neighborIdx)
			if neighbor < 0 || neighbor >= len(cells) {
				continue
			}
			if _, ok := seen[neighbor]; ok {
				continue
			}
			seen[neighbor] = struct{}{}
			queue = append(queue, state{cell: neighbor, hops: cur.hops + 1})
		}
	}
	return changed
}

// ComputeFlowAccumulation routes runoff downslope in descending-elevation order.
func ComputeFlowAccumulation(receivers []int, elevation []float64, runoff []float64) ([]float64, []int) {
	accumulation := make([]float64, len(elevation))
	copy(accumulation, runoff)

	order := make([]int, len(elevation))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool {
		return elevation[order[i]] > elevation[order[j]]
	})

	for _, r := range order {
		receiver := receivers[r]
		if receiver >= 0 {
			accumulation[receiver] += accumulation[r]
		}
	}

	return accumulation, order
}

func scaleFlowAccumulationForMesh(accumulation []float64, cellCount int) []float64 {
	areaScale := meshAreaResolutionScale(cellCount)
	if areaScale == 1 {
		return accumulation
	}
	scaled := make([]float64, len(accumulation))
	for i, value := range accumulation {
		scaled[i] = value * areaScale
	}
	return scaled
}

func carveFluvialChannels(
	cells []VoronoiCell,
	elevation []float64,
	receivers []int,
	runoff []float64,
	accumulation []float64,
) {
	buffer := make([]float64, len(elevation))
	copy(buffer, elevation)
	stepScale := meshPathCostResolutionScale(len(cells))
	if stepScale <= 0 {
		stepScale = 1
	}

	for r, elev := range elevation {
		if elev <= 0 {
			continue
		}

		receiver := receivers[r]
		if receiver < 0 {
			continue
		}

		slope := elev - elevation[receiver]
		if slope <= 0 {
			continue
		}
		physicalSlope := slope / stepScale

		channelScale := math.Log1p(accumulation[r])
		if channelScale <= 0.5 {
			continue
		}

		runoffFactor := math.Sqrt(runoff[r])
		slopeFactor := SmoothStep(20, 900, physicalSlope)
		incision := 16.0 * runoffFactor * channelScale * slopeFactor
		if elev > 2400 {
			incision *= 1.15
		}
		if elev < 250 {
			incision *= 0.65
		}
		incision = Clamp(incision, 0, 180)
		buffer[r] = elev - incision

		// Light valley widening around major channels so drainage reads as a
		// network instead of isolated single-cell pits.
		if channelScale > 2.3 {
			spreadChannelWidening(cells, elevation, buffer, r, receiver, accumulation, incision)
		}
	}

	copy(elevation, buffer)
}

func spreadChannelWidening(
	cells []VoronoiCell,
	elevation []float64,
	buffer []float64,
	center int,
	receiver int,
	accumulation []float64,
	incision float64,
) {
	if center < 0 || center >= len(cells) || incision <= 0 {
		return
	}
	maxHops := meshResolutionAdjustedSteps(1, len(cells))
	widening := incision * 0.18
	if widening > 24 {
		widening = 24
	}
	type state struct {
		cell int
		hops int
	}
	seen := map[int]struct{}{center: {}}
	queue := []state{{cell: center, hops: 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.hops > 0 {
			if cur.cell < 0 || cur.cell >= len(elevation) || elevation[cur.cell] <= 0 || cur.cell == receiver {
				continue
			}
			if cur.cell < len(accumulation) && accumulation[cur.cell] > accumulation[center]*0.7 {
				continue
			}
			if buffer[cur.cell] > elevation[cur.cell]-widening {
				buffer[cur.cell] = elevation[cur.cell] - widening
			}
		}
		if cur.hops >= maxHops {
			continue
		}
		for _, neighborIdx := range cells[cur.cell].NeighborSiteIndices {
			neighbor := int(neighborIdx)
			if neighbor < 0 || neighbor >= len(cells) {
				continue
			}
			if _, ok := seen[neighbor]; ok {
				continue
			}
			seen[neighbor] = struct{}{}
			queue = append(queue, state{cell: neighbor, hops: cur.hops + 1})
		}
	}
}

func applyFluvialDeposition(
	cells []VoronoiCell,
	elevation []float64,
	receivers []int,
	runoff []float64,
	accumulation []float64,
	order []int,
) {
	buffer := make([]float64, len(elevation))
	copy(buffer, elevation)
	stepScale := meshPathCostResolutionScale(len(cells))
	if stepScale <= 0 {
		stepScale = 1
	}

	// Traverse from low to high so downstream lowlands can receive small
	// deposits from major upstream systems.
	for i := len(order) - 1; i >= 0; i-- {
		r := order[i]
		if elevation[r] <= 0 {
			continue
		}

		receiver := receivers[r]
		if receiver < 0 {
			continue
		}

		slope := elevation[r] - elevation[receiver]
		physicalSlope := slope / stepScale
		channelScale := math.Log1p(accumulation[r])
		if channelScale < 2.0 {
			continue
		}

		// Deposit in low-gradient inland lowlands and near river mouths.
		depositionFactor := (1.0 - SmoothStep(25, 160, physicalSlope)) * math.Sqrt(runoff[r])
		if depositionFactor <= 0 {
			continue
		}

		deposit := 10.0 * channelScale * depositionFactor
		if elevation[receiver] <= 0 {
			// River-mouth / proto-delta case: spread deposition across the mouth
			// cell and nearby low coastal ground so large outlets build visible
			// fans and coastal plains.
			deposit *= 1.05
			if deposit > 26 {
				deposit = 26
			}
			buffer[r] += deposit
			spreadDeltaDeposit(cells, elevation, buffer, r, deposit*0.65)
			continue
		}

		if deposit > 28 {
			deposit = 28
		}
		buffer[receiver] += deposit

		// Major low-gradient rivers should also broaden adjoining floodplains.
		if channelScale > 2.6 && physicalSlope < 90 {
			spreadFloodplainDeposit(cells, elevation, buffer, receiver, deposit*0.35)
		}
	}

	copy(elevation, buffer)
}

func spreadDeltaDeposit(cells []VoronoiCell, elevation []float64, buffer []float64, mouth int, deposit float64) {
	if deposit <= 0 {
		return
	}
	spreadDepositByPhysicalRadius(cells, elevation, buffer, mouth, deposit, 2, 220, 160, 0.45)
}

func spreadFloodplainDeposit(cells []VoronoiCell, elevation []float64, buffer []float64, center int, deposit float64) {
	if deposit <= 0 {
		return
	}
	spreadDepositByPhysicalRadius(cells, elevation, buffer, center, deposit*0.55, 1, 500, 500, 1)
}

func spreadDepositByPhysicalRadius(
	cells []VoronoiCell,
	elevation []float64,
	buffer []float64,
	start int,
	deposit float64,
	baseRadius int,
	innerMaxElevation float64,
	outerMaxElevation float64,
	outerMultiplier float64,
) {
	if start < 0 || start >= len(cells) || deposit <= 0 {
		return
	}
	maxHops := meshResolutionAdjustedSteps(baseRadius, len(cells))
	stepScale := meshPathCostResolutionScale(len(cells))
	type state struct {
		cell int
		hops int
	}
	seen := map[int]struct{}{start: {}}
	queue := []state{{cell: start, hops: 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.hops > 0 {
			if cur.cell < 0 || cur.cell >= len(elevation) || elevation[cur.cell] <= 0 {
				continue
			}
			physicalDistance := float64(cur.hops) * stepScale
			maxElevation := innerMaxElevation
			amount := deposit
			if physicalDistance > 1 {
				maxElevation = outerMaxElevation
				amount *= outerMultiplier
			}
			if elevation[cur.cell] <= maxElevation {
				buffer[cur.cell] += amount
			}
		}
		if cur.hops >= maxHops {
			continue
		}
		for _, neighborIdx := range cells[cur.cell].NeighborSiteIndices {
			neighbor := int(neighborIdx)
			if neighbor < 0 || neighbor >= len(cells) {
				continue
			}
			if _, ok := seen[neighbor]; ok {
				continue
			}
			seen[neighbor] = struct{}{}
			queue = append(queue, state{cell: neighbor, hops: cur.hops + 1})
		}
	}
}
