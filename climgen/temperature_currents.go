package climgen

import (
	"math"
	"runtime"
	"sync"
)

// =============================================================================
// TEMPERATURE - OCEAN CURRENT EFFECTS
// =============================================================================
// Computes temperature effects from ocean currents:
//   - Current advection (local upstream temperature transport)
//   - Lagrangian backtracking (trace currents to source for warm/cold patterns)
//   - Temperature forcing toward source temperatures
//
// These mechanisms create realistic effects like:
//   - Gulf Stream warming Western Europe
//   - California Current cooling the US west coast
//   - Kuroshio warming Japan

// Physical constants for current advection
const (
	// Maximum ocean current speed (Gulf Stream peak) in m/s
	// Used to convert normalized currents (0-1) to physical velocities
	MaxOceanCurrentSpeed = 2.5 // m/s

	// Time step for advection calculation (should match energy balance TimeStep)
	AdvectionTimeStep = 1800.0 // seconds (30 minutes)
)

// ComputeCurrentAdvection calculates temperature anomalies from ocean currents.
// Uses physically-based advection: dT/dt = -v · ∇T
// Discretized as: ΔT = v * dt / cellSize * (T_upstream - T_here)
//
// This is automatically resolution-independent because:
//   - At higher resolution: cellSize smaller, but ΔT between neighbors also smaller
//   - Physical distance traversed per iteration scales correctly
func ComputeCurrentAdvection(
	temperature []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	currents []Vector3D,
	adj *FlatAdjacency,
	settings TransportSettings,
) []float64 {
	numVertices := len(temperature)
	advection := make([]float64, numVertices)

	if currents == nil || settings.CurrentTransportScale < 1e-9 {
		return advection
	}

	// Compute average cell size in meters for physical advection
	earthRadius := 6371.0e3 // meters
	avgCellSize := earthRadius * math.Sqrt(4*math.Pi/float64(numVertices))

	for i := 0; i < numVertices; i++ {
		if elevation[i] >= seaLevelThreshold {
			continue // Only ocean
		}

		currentVec := currents[i]
		normalizedSpeed := Length(currentVec)

		if normalizedSpeed < 0.01 {
			continue
		}

		// Convert normalized speed (0-1) to physical velocity (m/s)
		physicalSpeed := normalizedSpeed * MaxOceanCurrentSpeed

		currentDir := Scale(currentVec, 1.0/normalizedSpeed)

		// Upwind advection: find upstream temperature
		neighbors := adj.GetNeighbors(i)
		upstreamSum := 0.0
		upstreamWeight := 0.0

		for _, k := range neighbors {
			if k < 0 || k >= numVertices {
				continue
			}
			if elevation[k] >= seaLevelThreshold {
				continue
			}

			// Direction from neighbor k to this cell i
			dirFromK := Sub(vertices[i], vertices[k])
			dist := Length(dirFromK)
			if dist < 1e-9 {
				continue
			}
			dirFromK = Scale(dirFromK, 1.0/dist)

			// How much is k upstream of us?
			upstreamness := Dot(currentDir, dirFromK)
			if upstreamness > 0 {
				weight := upstreamness
				upstreamSum += temperature[k] * weight
				upstreamWeight += weight
			}
		}

		if upstreamWeight > 1e-9 {
			upstreamTemp := upstreamSum / upstreamWeight

			// Physical advection coefficient: v * dt / cellSize
			// This is dimensionless and resolution-independent
			advectionCoeff := physicalSpeed * AdvectionTimeStep / avgCellSize

			// Apply transport scale as a tuning multiplier
			advection[i] = settings.CurrentTransportScale * advectionCoeff *
				(upstreamTemp - temperature[i])
		}
	}

	return advection
}

// ComputeCurrentSourceTemperatures traces each ocean cell backwards along current
// streamlines to find where the water came from, then returns the equilibrium
// temperature at that source location. This creates realistic warm/cold current patterns.
//
// The algorithm:
// 1. Compute base equilibrium temperature for all cells (latitude-based)
// 2. For each ocean cell, trace backward along current for specified distance
// 3. Return the equilibrium temperature at the source location
//
// This naturally produces:
// - Warm western boundary currents (Gulf Stream traces back to tropics)
// - Cold eastern boundary currents (California Current traces back to subpolar)
func ComputeCurrentSourceTemperatures(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	currents []Vector3D,
	backtrackDistanceKm float64, // Physical distance to trace back (resolution-independent)
) []float64 {
	numVertices := len(vertices)
	sourceTemp := make([]float64, numVertices)

	if currents == nil || backtrackDistanceKm < 1.0 {
		// Return equilibrium temperatures
		for i := 0; i < numVertices; i++ {
			sourceTemp[i] = LatitudeEquilibriumTemp(vertices[i].Y)
		}
		return sourceTemp
	}

	// Compute cell size in km for resolution-independent backtracking
	earthRadiusKm := 6371.0
	avgCellSizeKm := earthRadiusKm * math.Sqrt(4*math.Pi/float64(numVertices))
	maxSteps := int(backtrackDistanceKm/avgCellSizeKm) + 1

	// Parallel processing
	numWorkers := runtime.GOMAXPROCS(0)
	if numVertices < 1000 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	chunkSize := (numVertices + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > numVertices {
			end = numVertices
		}
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				if elevation[i] >= seaLevelThreshold {
					// Land: use local equilibrium
					sourceTemp[i] = LatitudeEquilibriumTemp(vertices[i].Y)
					continue
				}

				// Trace backward along current streamline
				currentIdx := i
				distanceTraveled := 0.0

				for step := 0; step < maxSteps && distanceTraveled < backtrackDistanceKm; step++ {
					current := currents[currentIdx]
					speed := math.Sqrt(current.X*current.X + current.Y*current.Y + current.Z*current.Z)
					if speed < 0.01 {
						break // No current, stop tracing
					}

					// Find upstream neighbor (opposite of current direction)
					upstreamDir := Vector3D{
						X: -current.X / speed,
						Y: -current.Y / speed,
						Z: -current.Z / speed,
					}

					// Find neighbor most aligned with upstream direction
					bestNeighbor := -1
					bestAlignment := -2.0

					for _, k := range adj.GetNeighbors(currentIdx) {
						if k < 0 || k >= numVertices {
							continue
						}
						if elevation[k] >= seaLevelThreshold {
							continue // Skip land
						}

						// Direction to neighbor
						toNeighbor := Sub(vertices[k], vertices[currentIdx])
						dist := Length(toNeighbor)
						if dist < 1e-9 {
							continue
						}
						toNeighbor = Scale(toNeighbor, 1.0/dist)

						alignment := Dot(toNeighbor, upstreamDir)
						if alignment > bestAlignment {
							bestAlignment = alignment
							bestNeighbor = k
						}
					}

					if bestNeighbor < 0 || bestAlignment < 0.1 {
						break // No good upstream neighbor
					}

					currentIdx = bestNeighbor
					distanceTraveled += avgCellSizeKm
				}

				// Use equilibrium temperature at the source location
				sourceTemp[i] = LatitudeEquilibriumTemp(vertices[currentIdx].Y)
			}
		}(start, end)
	}

	wg.Wait()

	// Smooth current source temps to avoid sharp boundaries
	// Use physical distance for resolution independence (~2000 km smoothing radius)
	smoothingDistanceKm := 2000.0
	smoothingIterations := int(smoothingDistanceKm/avgCellSizeKm) + 1
	smoothed := SmoothCurrentSourceTemps(sourceTemp, elevation, seaLevelThreshold, adj, smoothingIterations)

	return smoothed
}

// SmoothCurrentSourceTemps applies neighbor averaging to current source temperatures
// to reduce sharp boundaries in ocean. Only smooths ocean cells. Parallelized.
func SmoothCurrentSourceTemps(
	sourceTemp []float64,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	iterations int,
) []float64 {
	numVertices := len(sourceTemp)
	current := make([]float64, numVertices)
	copy(current, sourceTemp)

	numWorkers := runtime.GOMAXPROCS(0)
	if numVertices < 1000 {
		numWorkers = 1
	}

	for iter := 0; iter < iterations; iter++ {
		next := make([]float64, numVertices)

		var wg sync.WaitGroup
		chunkSize := (numVertices + numWorkers - 1) / numWorkers

		for w := 0; w < numWorkers; w++ {
			start := w * chunkSize
			end := start + chunkSize
			if end > numVertices {
				end = numVertices
			}
			if start >= end {
				continue
			}

			wg.Add(1)
			go func(start, end int) {
				defer wg.Done()
				for i := start; i < end; i++ {
					if elevation[i] >= seaLevelThreshold {
						next[i] = current[i]
						continue
					}

					sum := current[i]
					count := 1.0
					for _, k := range adj.GetNeighbors(i) {
						if k >= 0 && k < numVertices && elevation[k] < seaLevelThreshold {
							sum += current[k]
							count++
						}
					}
					next[i] = sum / count
				}
			}(start, end)
		}

		wg.Wait()
		current = next
	}

	return current
}

// LatitudeEquilibriumTemp returns the equilibrium ocean temperature for a given
// sine of latitude (Y coordinate on unit sphere). Based on Earth's ocean temps.
func LatitudeEquilibriumTemp(sinLat float64) float64 {
	// Clamp
	if sinLat > 1 {
		sinLat = 1
	} else if sinLat < -1 {
		sinLat = -1
	}

	absLat := math.Abs(math.Asin(sinLat)) // radians

	// Earth-like temperature profile:
	// Equator: ~28°C, Poles: ~-2°C (seawater freezing)
	// Use cosine profile for smooth transition
	// T = T_equator * cos(lat)^n + T_pole * (1 - cos(lat)^n)
	cosLat := math.Cos(absLat)
	cosFactor := cosLat * cosLat // cos²(lat) gives good Earth-like profile

	equatorTemp := 28.0 + 273.15 // K
	poleTemp := -2.0 + 273.15    // K (seawater freezing point)

	return equatorTemp*cosFactor + poleTemp*(1-cosFactor)
}

// ComputeCurrentTemperatureForcing computes forcing to push temperatures toward
// current-transported source temperatures.
func ComputeCurrentTemperatureForcing(
	temperature []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	currents []Vector3D,
	strength float64,            // W/m² forcing strength
	backtrackDistanceKm float64, // Physical distance to backtrack (km)
) []float64 {
	numVertices := len(temperature)
	forcing := make([]float64, numVertices)

	if currents == nil || strength < 1e-9 {
		return forcing
	}

	// Get source temperatures by backtracking (already parallelized)
	sourceTemps := ComputeCurrentSourceTemperatures(
		vertices, elevation, seaLevelThreshold, adj, currents, backtrackDistanceKm,
	)

	// Parallel forcing computation
	numWorkers := runtime.GOMAXPROCS(0)
	if numVertices < 1000 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	chunkSize := (numVertices + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > numVertices {
			end = numVertices
		}
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				if elevation[i] >= seaLevelThreshold {
					continue
				}

				current := currents[i]
				speed := math.Sqrt(current.X*current.X + current.Y*current.Y + current.Z*current.Z)
				if speed < 0.01 {
					continue
				}

				tempDiff := sourceTemps[i] - temperature[i]
				forcing[i] = strength * speed * tempDiff
			}
		}(start, end)
	}

	wg.Wait()
	return forcing
}
