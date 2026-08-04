package climgen

import (
	"math"
	"runtime"
	"sync"
)

// =============================================================================
// TEMPERATURE - COMBINED HEAT TRANSPORT (ORCHESTRATOR)
// =============================================================================
// This file combines all heat transport mechanisms into a single optimized pass.
// Individual transport mechanisms are in their own files:
//   - temperature_diffusion.go: Heat diffusion (neighbor averaging)
//   - temperature_advection.go: Wind advection
//   - temperature_currents.go: Ocean current advection and backtracking
//   - temperature_continentality.go: Continentality and marine influence

// ComputeTotalHeatTransport combines all heat transport mechanisms in a single pass.
// This is optimized to avoid multiple iterations over the vertex array.
// Uses parallel goroutines for large meshes.
func ComputeTotalHeatTransport(
	temperature []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	currents []Vector3D,
	settings TransportSettings,
) []float64 {
	numVertices := len(temperature)
	total := make([]float64, numVertices)

	// Precompute constants (avoid recalculating per-vertex)
	earthRadius := 6371.0e3 // meters
	avgCellSize := earthRadius * math.Sqrt(4*math.Pi/float64(numVertices))
	windCoeffBase := MaxSurfaceWindSpeed * AdvectionTimeStep / avgCellSize * settings.WindTransportScale
	currentCoeffBase := MaxOceanCurrentSpeed * AdvectionTimeStep / avgCellSize * settings.CurrentTransportScale

	hasWind := wind != nil && settings.WindTransportScale > 1e-9
	hasCurrents := currents != nil && settings.CurrentTransportScale > 1e-9

	// Use parallel processing for large meshes
	numWorkers := runtime.GOMAXPROCS(0)
	if numVertices < 1000 {
		numWorkers = 1 // Serial for small meshes
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
			computeTransportChunk(
				start, end, temperature, vertices, elevation, seaLevelThreshold,
				adj, wind, currents, total,
				settings.DiffusionLand, settings.DiffusionWater,
				windCoeffBase, currentCoeffBase, hasWind, hasCurrents,
			)
		}(start, end)
	}

	wg.Wait()
	return total
}

// computeTransportChunk processes a range of vertices for heat transport
func computeTransportChunk(
	start, end int,
	temperature []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	currents []Vector3D,
	total []float64,
	diffusionLand, diffusionWater float64,
	windCoeffBase, currentCoeffBase float64,
	hasWind, hasCurrents bool,
) {
	numVertices := len(temperature)

	for i := start; i < end; i++ {
		var transport float64
		isOcean := elevation[i] < seaLevelThreshold
		neighbors := adj.GetNeighbors(i)
		temp_i := temperature[i]
		v_i := vertices[i]

		// 1. Diffusion (always computed)
		var D float64
		if isOcean {
			D = diffusionWater
		} else {
			D = diffusionLand
		}

		if D > 1e-12 && len(neighbors) > 0 {
			lat := math.Abs(v_i.Y) // sin(latitude) for unit sphere
			// sin(2*asin(y)) == 2*y*sqrt(1-y^2); the identity avoids an asin and
			// a sin per vertex per energy-balance iteration.
			latSq := lat * lat
			if latSq > 1 {
				latSq = 1
			}
			latScale := 1.0 + lat*math.Sqrt(1-latSq)

			sumTemp := 0.0
			count := 0
			for _, k := range neighbors {
				if k >= 0 && k < numVertices {
					// Only diffuse with same surface type (land↔land, ocean↔ocean)
					// This prevents wind effects on land from bleeding into ocean
					neighborIsOcean := elevation[k] < seaLevelThreshold
					if neighborIsOcean == isOcean {
						sumTemp += temperature[k]
						count++
					}
				}
			}
			if count > 0 {
				transport += D * latScale * (sumTemp/float64(count) - temp_i)
			}
		}

		// 2. Wind advection (land only - ocean has too much thermal inertia)
		if hasWind && !isOcean {
			windVec := wind[i]
			windSpeed := math.Sqrt(windVec.X*windVec.X + windVec.Y*windVec.Y + windVec.Z*windVec.Z)

			if windSpeed > 0.01 {
				invSpeed := 1.0 / windSpeed
				windDirX := windVec.X * invSpeed
				windDirY := windVec.Y * invSpeed
				windDirZ := windVec.Z * invSpeed

				upwindSum := 0.0
				upwindWeight := 0.0

				for _, k := range neighbors {
					if k < 0 || k >= numVertices {
						continue
					}
					v_k := vertices[k]
					dx := v_i.X - v_k.X
					dy := v_i.Y - v_k.Y
					dz := v_i.Z - v_k.Z
					dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
					if dist < 1e-9 {
						continue
					}
					invDist := 1.0 / dist
					upwindness := windDirX*dx*invDist + windDirY*dy*invDist + windDirZ*dz*invDist
					if upwindness > 0 {
						upwindSum += temperature[k] * upwindness
						upwindWeight += upwindness
					}
				}

				if upwindWeight > 1e-9 {
					upwindTemp := upwindSum / upwindWeight
					transport += windCoeffBase * windSpeed * (upwindTemp - temp_i)
				}
			}
		}

		// 3. Current advection (ocean only)
		if hasCurrents && isOcean {
			currentVec := currents[i]
			currentSpeed := math.Sqrt(currentVec.X*currentVec.X + currentVec.Y*currentVec.Y + currentVec.Z*currentVec.Z)

			if currentSpeed > 0.01 {
				invSpeed := 1.0 / currentSpeed
				curDirX := currentVec.X * invSpeed
				curDirY := currentVec.Y * invSpeed
				curDirZ := currentVec.Z * invSpeed

				upstreamSum := 0.0
				upstreamWeight := 0.0

				for _, k := range neighbors {
					if k < 0 || k >= numVertices || elevation[k] >= seaLevelThreshold {
						continue
					}
					v_k := vertices[k]
					dx := v_i.X - v_k.X
					dy := v_i.Y - v_k.Y
					dz := v_i.Z - v_k.Z
					dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
					if dist < 1e-9 {
						continue
					}
					invDist := 1.0 / dist
					upstreamness := curDirX*dx*invDist + curDirY*dy*invDist + curDirZ*dz*invDist
					if upstreamness > 0 {
						upstreamSum += temperature[k] * upstreamness
						upstreamWeight += upstreamness
					}
				}

				if upstreamWeight > 1e-9 {
					upstreamTemp := upstreamSum / upstreamWeight
					transport += currentCoeffBase * currentSpeed * (upstreamTemp - temp_i)
				}
			}
		}

		total[i] = transport
	}
}
