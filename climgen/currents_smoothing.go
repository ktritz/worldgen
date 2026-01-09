package climgen

import (
	"runtime"
	"sync"
)

// SmoothCurrentsParallel performs iterative smoothing passes using goroutine parallelism.
// This is the production version optimized for large meshes.
func SmoothCurrentsParallel(
	currents []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	coastLandDirs []Vector3D,
	settings CurrentSettings,
) []Vector3D {
	numVertices := len(vertices)
	numWorkers := runtime.NumCPU()

	// Pre-compute water mask for fast lookups
	isWater := make([]bool, numVertices)
	for i := 0; i < numVertices; i++ {
		isWater[i] = elevation[i] < seaLevelThreshold
	}

	// Pre-compute coastline mask
	isCoastline := make([]bool, numVertices)
	for i := 0; i < numVertices; i++ {
		isCoastline[i] = LengthSq(coastLandDirs[i]) > 1e-12
	}

	smoothed := make([]Vector3D, numVertices)
	copy(smoothed, currents)

	next := make([]Vector3D, numVertices)

	chunkSize := (numVertices + numWorkers - 1) / numWorkers

	for iter := 0; iter < settings.SmoothingIterations; iter++ {
		var wg sync.WaitGroup

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			start := w * chunkSize
			end := start + chunkSize
			if end > numVertices {
				end = numVertices
			}

			go func(start, end int) {
				defer wg.Done()

				for i := start; i < end; i++ {
					if !isWater[i] {
						continue
					}

					// Average neighbor currents
					neighborSum := Vector3D{}
					waterNeighborCount := 0

					for _, k := range adj.GetNeighbors(i) {
						if k >= 0 && k < numVertices && isWater[k] {
							neighborSum.X += smoothed[k].X
							neighborSum.Y += smoothed[k].Y
							neighborSum.Z += smoothed[k].Z
							waterNeighborCount++
						}
					}

					if waterNeighborCount == 0 {
						next[i] = smoothed[i]
						continue
					}

					invCount := 1.0 / float64(waterNeighborCount)
					avgX := neighborSum.X * invCount
					avgY := neighborSum.Y * invCount
					avgZ := neighborSum.Z * invCount

					// Blend
					oneMinusFactor := 1.0 - settings.SmoothingFactor
					blendX := smoothed[i].X*oneMinusFactor + avgX*settings.SmoothingFactor
					blendY := smoothed[i].Y*oneMinusFactor + avgY*settings.SmoothingFactor
					blendZ := smoothed[i].Z*oneMinusFactor + avgZ*settings.SmoothingFactor

					if isCoastline[i] {
						// Coastline: remove perpendicular-to-coast component
						landDir := coastLandDirs[i]
						dotPerp := blendX*landDir.X + blendY*landDir.Y + blendZ*landDir.Z

						parallelX := (blendX - landDir.X*dotPerp) * settings.CoastParallelBoost
						parallelY := (blendY - landDir.Y*dotPerp) * settings.CoastParallelBoost
						parallelZ := (blendZ - landDir.Z*dotPerp) * settings.CoastParallelBoost

						parallelLenSq := parallelX*parallelX + parallelY*parallelY + parallelZ*parallelZ
						if !IsFinite(parallelLenSq) || parallelLenSq > settings.MaxAllowedSpeedSq {
							next[i] = Vector3D{}
						} else {
							next[i] = Vector3D{X: parallelX, Y: parallelY, Z: parallelZ}
						}
					} else {
						// Interior: project onto tangent plane
						nx, ny, nz := vertices[i].X, vertices[i].Y, vertices[i].Z
						dotNormal := blendX*nx + blendY*ny + blendZ*nz

						tangentX := blendX - nx*dotNormal
						tangentY := blendY - ny*dotNormal
						tangentZ := blendZ - nz*dotNormal

						tangentLenSq := tangentX*tangentX + tangentY*tangentY + tangentZ*tangentZ
						if !IsFinite(tangentLenSq) || tangentLenSq > settings.MaxAllowedSpeedSq {
							next[i] = Vector3D{}
						} else {
							next[i] = Vector3D{X: tangentX, Y: tangentY, Z: tangentZ}
						}
					}
				}
			}(start, end)
		}

		wg.Wait()

		// Swap buffers
		smoothed, next = next, smoothed
	}

	return smoothed
}

// SmoothCurrentsSIMDFriendly is an alternative smoothing implementation
// with data layout optimized for potential SIMD vectorization.
// Uses separate X, Y, Z arrays for better cache utilization.
func SmoothCurrentsSIMDFriendly(
	currentsX, currentsY, currentsZ []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	coastLandDirs []Vector3D,
	settings CurrentSettings,
) ([]float64, []float64, []float64) {
	numVertices := len(vertices)
	numWorkers := runtime.NumCPU()

	// Pre-compute masks
	isWater := make([]bool, numVertices)
	isCoastline := make([]bool, numVertices)
	for i := 0; i < numVertices; i++ {
		isWater[i] = elevation[i] < seaLevelThreshold
		isCoastline[i] = LengthSq(coastLandDirs[i]) > 1e-12
	}

	// Working arrays
	smoothedX := make([]float64, numVertices)
	smoothedY := make([]float64, numVertices)
	smoothedZ := make([]float64, numVertices)
	copy(smoothedX, currentsX)
	copy(smoothedY, currentsY)
	copy(smoothedZ, currentsZ)

	nextX := make([]float64, numVertices)
	nextY := make([]float64, numVertices)
	nextZ := make([]float64, numVertices)

	chunkSize := (numVertices + numWorkers - 1) / numWorkers

	for iter := 0; iter < settings.SmoothingIterations; iter++ {
		var wg sync.WaitGroup

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			start := w * chunkSize
			end := start + chunkSize
			if end > numVertices {
				end = numVertices
			}

			go func(start, end int) {
				defer wg.Done()

				for i := start; i < end; i++ {
					if !isWater[i] {
						continue
					}

					// Average neighbors
					sumX, sumY, sumZ := 0.0, 0.0, 0.0
					count := 0

					for _, k := range adj.GetNeighbors(i) {
						if k >= 0 && k < numVertices && isWater[k] {
							sumX += smoothedX[k]
							sumY += smoothedY[k]
							sumZ += smoothedZ[k]
							count++
						}
					}

					if count == 0 {
						nextX[i] = smoothedX[i]
						nextY[i] = smoothedY[i]
						nextZ[i] = smoothedZ[i]
						continue
					}

					invCount := 1.0 / float64(count)
					oneMinusFactor := 1.0 - settings.SmoothingFactor

					blendX := smoothedX[i]*oneMinusFactor + sumX*invCount*settings.SmoothingFactor
					blendY := smoothedY[i]*oneMinusFactor + sumY*invCount*settings.SmoothingFactor
					blendZ := smoothedZ[i]*oneMinusFactor + sumZ*invCount*settings.SmoothingFactor

					if isCoastline[i] {
						landDir := coastLandDirs[i]
						dotPerp := blendX*landDir.X + blendY*landDir.Y + blendZ*landDir.Z

						px := (blendX - landDir.X*dotPerp) * settings.CoastParallelBoost
						py := (blendY - landDir.Y*dotPerp) * settings.CoastParallelBoost
						pz := (blendZ - landDir.Z*dotPerp) * settings.CoastParallelBoost

						lenSq := px*px + py*py + pz*pz
						if !IsFinite(lenSq) || lenSq > settings.MaxAllowedSpeedSq {
							nextX[i], nextY[i], nextZ[i] = 0, 0, 0
						} else {
							nextX[i], nextY[i], nextZ[i] = px, py, pz
						}
					} else {
						v := vertices[i]
						dot := blendX*v.X + blendY*v.Y + blendZ*v.Z

						tx := blendX - v.X*dot
						ty := blendY - v.Y*dot
						tz := blendZ - v.Z*dot

						lenSq := tx*tx + ty*ty + tz*tz
						if !IsFinite(lenSq) || lenSq > settings.MaxAllowedSpeedSq {
							nextX[i], nextY[i], nextZ[i] = 0, 0, 0
						} else {
							nextX[i], nextY[i], nextZ[i] = tx, ty, tz
						}
					}
				}
			}(start, end)
		}

		wg.Wait()

		// Swap buffers
		smoothedX, nextX = nextX, smoothedX
		smoothedY, nextY = nextY, smoothedY
		smoothedZ, nextZ = nextZ, smoothedZ
	}

	return smoothedX, smoothedY, smoothedZ
}
