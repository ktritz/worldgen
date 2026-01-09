package climgen

import (
	"math"
	"runtime"
	"sync"
)

// =============================================================================
// MARITIME INFLUENCE - COASTAL SPREADING ALGORITHM
// =============================================================================
// Computes how ocean air carried by wind affects land temperatures.
//
// Algorithm:
//   Phase 1: Initialize at coast - find land cells with ocean neighbors,
//            compute onshore wind component, capture ocean temperature
//   Phase 2: Spread inland following wind - iteratively propagate influence
//            and source temperature downwind with distance decay
//
// This creates the maritime climate effect where:
//   - Gulf Stream warmth reaches northern Europe via westerlies
//   - California Current cold reaches the US west coast
//   - Effect is strongest at coast and decays inland

// MaritimeSettings controls the maritime influence calculation
type MaritimeSettings struct {
	DecayDistanceKm float64 // Distance at which influence drops to 1/e (~37%)
	MaxDistanceKm   float64 // Maximum distance influence can spread inland
	MinWindSpeed    float64 // Minimum wind speed to carry maritime air (0-1 normalized)
	BlendStrength   float64 // How much to blend toward ocean temp (0-1)
}

// DefaultMaritimeSettings returns reasonable defaults
func DefaultMaritimeSettings() MaritimeSettings {
	return MaritimeSettings{
		DecayDistanceKm: 300.0,  // Influence at ~37% by 300km inland
		MaxDistanceKm:   1500.0, // Maritime effects can reach 1500km inland
		MinWindSpeed:    0.05,   // Very light winds still carry some maritime air
		BlendStrength:   0.5,    // Blend 50% toward ocean temp at full influence
	}
}

// MaritimeResult holds computed maritime influence for each cell
type MaritimeResult struct {
	// Influence is 0+ for each land cell (can exceed 1 with multiple sources)
	// 0 = continental (no maritime influence)
	// Higher = more maritime influence from multiple sources
	Influence []float64

	// SourceTemp is the weighted average ocean temperature affecting this cell (Kelvin)
	// This is the actual simulated ocean temp, including current effects
	SourceTemp []float64
}

// ComputeMaritimeInfluence calculates maritime influence using coastal spreading.
//
// Parameters:
//   - vertices: cell center positions on unit sphere
//   - elevation: elevation of each cell (< seaLevel = ocean)
//   - seaLevelThreshold: elevation threshold for ocean
//   - adj: cell adjacency structure
//   - wind: surface wind vectors for each cell
//   - oceanTemp: current ocean temperatures in Kelvin (from energy balance)
//   - settings: algorithm parameters
//
// Returns MaritimeResult with influence strength and source temperatures.
func ComputeMaritimeInfluence(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	oceanTemp []float64,
	settings MaritimeSettings,
) *MaritimeResult {
	n := len(vertices)
	result := &MaritimeResult{
		Influence:  make([]float64, n),
		SourceTemp: make([]float64, n),
	}

	if wind == nil || oceanTemp == nil {
		return result
	}

	// Compute cell size for distance calculations
	earthRadiusKm := 6371.0
	cellSizeKm := earthRadiusKm * math.Sqrt(4*math.Pi/float64(n))

	// Phase 1: Initialize coastal cells (parallelized)
	initializeCoastalCells(result, vertices, elevation, seaLevelThreshold, adj, wind, oceanTemp, settings)

	// Phase 2: Spread inland following wind (parallelized with local buffers)
	numIterations := int(settings.MaxDistanceKm/cellSizeKm) + 1
	spreadInfluenceInland(result, vertices, elevation, seaLevelThreshold, adj, wind, cellSizeKm, settings, numIterations)

	return result
}

// initializeCoastalCells sets up maritime influence at land cells adjacent to ocean.
// Parallelized - each cell only writes to itself, no conflicts.
func initializeCoastalCells(
	result *MaritimeResult,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	oceanTemp []float64,
	settings MaritimeSettings,
) {
	n := len(vertices)

	numWorkers := runtime.GOMAXPROCS(0)
	if n < 1000 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	chunkSize := (n + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > n {
			end = n
		}
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()

			for i := start; i < end; i++ {
				// Skip ocean cells
				if elevation[i] < seaLevel {
					continue
				}

				// Get wind at this cell
				windVec := wind[i]
				windSpeed := math.Sqrt(windVec.X*windVec.X + windVec.Y*windVec.Y + windVec.Z*windVec.Z)
				if windSpeed < settings.MinWindSpeed {
					continue
				}

				// Normalize wind direction
				windDir := Vector3D{
					X: windVec.X / windSpeed,
					Y: windVec.Y / windSpeed,
					Z: windVec.Z / windSpeed,
				}

				// Find ocean neighbors and compute influence
				// Two components:
				// 1. Base coastal influence (any adjacent ocean)
				// 2. Wind-driven bonus (onshore wind adds more)
				var oceanTempSum float64
				var oceanWeightSum float64
				var onshoreBonus float64
				oceanCount := 0

				for _, k := range adj.GetNeighbors(i) {
					if k < 0 || k >= n {
						continue
					}
					if elevation[k] >= seaLevel {
						continue // Not ocean
					}

					oceanCount++
					// Always capture ocean temperature (weighted equally for base)
					oceanTempSum += oceanTemp[k]
					oceanWeightSum += 1.0

					// Direction FROM ocean neighbor TO this land cell
					fromOcean := Sub(vertices[i], vertices[k])
					fromOcean = Normalize(fromOcean)

					// Onshore component: how much wind aligns with "from ocean" direction
					// Positive = wind blowing from ocean onto land
					onshore := Dot(windDir, fromOcean)

					if onshore > 0 {
						// Add wind-driven bonus
						onshoreBonus += onshore * windSpeed
					}
				}

				if oceanCount > 0 {
					// Base influence: 0.3 for any coastal cell (islands get this on all sides)
					// Wind bonus: up to ~1.0 additional for strong onshore wind
					baseInfluence := 0.3
					windInfluence := onshoreBonus / float64(oceanCount)
					result.Influence[i] = baseInfluence + windInfluence
					result.SourceTemp[i] = oceanTempSum / oceanWeightSum
				}
			}
		}(start, end)
	}

	wg.Wait()
}

// spreadInfluenceInland propagates maritime influence from coast inland following wind.
// Uses wavefront propagation with distance decay.
// Key insight: Only the "wavefront" (fresh influence this iteration) spreads to neighbors.
// Total influence accumulates, but we don't re-spread already-accumulated influence.
// Optimized with sparse active sets to avoid processing unchanged cells.
func spreadInfluenceInland(
	result *MaritimeResult,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	cellSizeKm float64,
	settings MaritimeSettings,
	numIterations int,
) {
	n := len(vertices)

	// Precompute decay per cell step
	decayPerStep := math.Exp(-cellSizeKm / settings.DecayDistanceKm)

	numWorkers := runtime.GOMAXPROCS(0)
	if n < 1000 {
		numWorkers = 1
	}

	// Wavefront tracking:
	// - wavefront[i] = influence arriving this iteration (to be spread next)
	// - totalInfluence = accumulated influence from all iterations
	// - totalSourceTemp = weighted average ocean temp
	wavefront := make([]float64, n)
	wavefrontTemp := make([]float64, n) // source temp for wavefront
	totalInfluence := result.Influence  // initialized from coastal init
	totalSourceTemp := result.SourceTemp

	// Initialize wavefront from coastal cells
	copy(wavefront, totalInfluence)
	copy(wavefrontTemp, totalSourceTemp)

	// Track active cells (cells with wavefront > 0)
	// Use maps for sparse iteration
	activeSet := make(map[int]struct{})
	for i := 0; i < n; i++ {
		if wavefront[i] > 0.001 {
			activeSet[i] = struct{}{}
		}
	}

	// Local contribution accumulator per worker
	type contribution struct {
		influence float64
		tempSum   float64
	}

	for iter := 0; iter < numIterations; iter++ {
		if len(activeSet) == 0 {
			break
		}

		// Convert active set to slice for parallel processing
		activeSlice := make([]int, 0, len(activeSet))
		for i := range activeSet {
			activeSlice = append(activeSlice, i)
		}

		// Next wavefront accumulator - use map for sparse storage
		nextWavefront := make(map[int]contribution)
		var nextMu sync.Mutex

		// Parallel spreading from active cells
		var wg sync.WaitGroup
		chunkSize := (len(activeSlice) + numWorkers - 1) / numWorkers
		if chunkSize < 1 {
			chunkSize = 1
		}

		for w := 0; w < numWorkers; w++ {
			start := w * chunkSize
			end := start + chunkSize
			if end > len(activeSlice) {
				end = len(activeSlice)
			}
			if start >= end {
				continue
			}

			wg.Add(1)
			go func(start, end int) {
				defer wg.Done()

				// Local buffer to batch updates
				localContribs := make(map[int]contribution)

				for idx := start; idx < end; idx++ {
					i := activeSlice[idx]

					wf := wavefront[i]
					if wf < 0.001 {
						continue
					}

					// Get wind at this cell
					windVec := wind[i]
					windSpeed := math.Sqrt(windVec.X*windVec.X + windVec.Y*windVec.Y + windVec.Z*windVec.Z)
					if windSpeed < settings.MinWindSpeed {
						continue
					}

					// Wind direction
					windDir := Vector3D{
						X: windVec.X / windSpeed,
						Y: windVec.Y / windSpeed,
						Z: windVec.Z / windSpeed,
					}

					srcTemp := wavefrontTemp[i]

					// Spread to downwind land neighbors
					for _, k := range adj.GetNeighbors(i) {
						if k < 0 || k >= n {
							continue
						}
						if elevation[k] < seaLevel {
							continue // Don't spread to ocean
						}

						// Direction from this cell to neighbor
						toNeighbor := Sub(vertices[k], vertices[i])
						toNeighbor = Normalize(toNeighbor)

						// Downwind alignment: positive if neighbor is downwind
						downwind := Dot(windDir, toNeighbor)

						if downwind > 0.1 { // Must be reasonably downwind
							// Contribution to neighbor - use wavefront, not total
							contrib := wf * decayPerStep * downwind * windSpeed

							if contrib > 0.001 {
								c := localContribs[k]
								c.influence += contrib
								c.tempSum += contrib * srcTemp
								localContribs[k] = c
							}
						}
					}
				}

				// Merge local contributions into shared map
				if len(localContribs) > 0 {
					nextMu.Lock()
					for k, c := range localContribs {
						nc := nextWavefront[k]
						nc.influence += c.influence
						nc.tempSum += c.tempSum
						nextWavefront[k] = nc
					}
					nextMu.Unlock()
				}
			}(start, end)
		}

		wg.Wait()

		// No new spreading - done
		if len(nextWavefront) == 0 {
			break
		}

		// Reset wavefront for next iteration
		for i := range activeSet {
			wavefront[i] = 0
		}
		activeSet = make(map[int]struct{})

		// Apply next wavefront: add to total and set up for next iteration
		for i, c := range nextWavefront {
			// Update wavefront for next iteration
			wavefront[i] = c.influence
			wavefrontTemp[i] = c.tempSum / c.influence
			activeSet[i] = struct{}{}

			// Accumulate into total influence (capped to prevent hotspots)
			oldInf := totalInfluence[i]
			newInf := oldInf + c.influence

			// Cap total influence - there's only so maritime a cell can be
			const maxInfluence = 1.5
			if newInf > maxInfluence {
				newInf = maxInfluence
			}

			if oldInf > 0 {
				// Weighted average of existing and new source temps
				totalSourceTemp[i] = (oldInf*totalSourceTemp[i] + c.tempSum) / (oldInf + c.influence)
			} else {
				totalSourceTemp[i] = c.tempSum / c.influence
			}
			totalInfluence[i] = newInf
		}
	}

	// Results are already in result.Influence and result.SourceTemp
	// (we used those slices directly)
}

// ApplyMaritimeEffect blends land temperatures toward maritime source temperatures.
// This should be called after the energy balance has computed base temperatures.
func ApplyMaritimeEffect(
	temperature []float64,
	elevation []float64,
	seaLevel float64,
	maritime *MaritimeResult,
	settings MaritimeSettings,
) []float64 {
	n := len(temperature)
	result := make([]float64, n)
	copy(result, temperature)

	if maritime == nil {
		return result
	}

	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			continue // Ocean unchanged
		}

		influence := maritime.Influence[i]
		if influence < 0.001 {
			continue // No maritime influence
		}

		// Clamp influence to reasonable range
		if influence > 1.0 {
			influence = 1.0
		}

		// Blend toward source temperature
		blend := influence * settings.BlendStrength
		result[i] = temperature[i]*(1-blend) + maritime.SourceTemp[i]*blend
	}

	return result
}

// GetMaritimeStats returns statistics about the maritime influence calculation
func GetMaritimeStats(result *MaritimeResult, elevation []float64, seaLevel float64) (landCells, withInfluence int, avgInfluence, maxInfluence float64) {
	for i, inf := range result.Influence {
		if elevation[i] >= seaLevel {
			landCells++
			if inf > 0.001 {
				withInfluence++
				avgInfluence += inf
				if inf > maxInfluence {
					maxInfluence = inf
				}
			}
		}
	}
	if withInfluence > 0 {
		avgInfluence /= float64(withInfluence)
	}
	return
}
