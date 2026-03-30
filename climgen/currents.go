package climgen

import (
	"fmt"
)

// =============================================================================
// OCEAN CURRENT GENERATION - MAIN ENTRY POINTS
// =============================================================================
// This file contains the main entry points for ocean current generation.
// The implementation is split across several files:
//   - currents.go (this file): Main entry points and current smoothing
//   - streamfunction.go: Streamfunction generation and velocity computation
//   - boundary.go: Western boundary layer and coastline utilities
//   - currents_basin.go: Legacy basin-based approach (preserved for future use)

// GenerateCurrentsStreamfunction is the main entry point for streamfunction-based currents.
// Uses wind-driven Sverdrup model with western boundary intensification.
func GenerateCurrentsStreamfunction(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	componentAssignments []int,
	components []OceanComponent,
	settings CurrentSettings,
) []Vector3D {
	if settings.Verbose {
		fmt.Println("  Generating wind-driven streamfunction (Sverdrup model)...")
	}

	psi := GenerateWindDrivenStreamfunction(
		vertices, elevation, seaLevelThreshold, adj, componentAssignments, components,
		settings.TargetEdgeSpeed*15.0, // Scale factor for Ψ magnitude (higher = stronger gradients)
	)
	return currentsFromStreamfunction(
		vertices, elevation, seaLevelThreshold, adj, psi, componentAssignments, components, settings,
	)
}

// GenerateCurrentsStreamfunctionFromWind derives the Sverdrup streamfunction
// from the generated surface wind field instead of the fixed latitude-only curl
// pattern. A small fallback blend of the idealized curl is retained for
// stability when local wind curl is weak or noisy.
func GenerateCurrentsStreamfunctionFromWind(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	componentAssignments []int,
	components []OceanComponent,
	settings CurrentSettings,
) []Vector3D {
	if len(wind) != len(vertices) {
		return GenerateCurrentsStreamfunction(vertices, elevation, seaLevelThreshold, adj, componentAssignments, components, settings)
	}
	if settings.Verbose {
		fmt.Println("  Generating streamfunction from surface-wind stress curl...")
	}

	psi := GenerateWindDrivenStreamfunctionFromWind(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		componentAssignments,
		components,
		settings.TargetEdgeSpeed*15.0,
	)
	return currentsFromStreamfunction(
		vertices, elevation, seaLevelThreshold, adj, psi, componentAssignments, components, settings,
	)
}

func currentsFromStreamfunction(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	psi []float64,
	componentAssignments []int,
	components []OceanComponent,
	settings CurrentSettings,
) []Vector3D {
	if settings.Verbose {
		minPsi, maxPsi := psi[0], psi[0]
		for _, p := range psi {
			if p < minPsi {
				minPsi = p
			}
			if p > maxPsi {
				maxPsi = p
			}
		}
		fmt.Printf("  Initial Ψ range: [%.4f, %.4f]\n", minPsi, maxPsi)
	}

	cellSize := estimateCellSize(vertices, adj)
	const targetDiffusionAngular = 0.02
	scaledIterations := int(targetDiffusionAngular/cellSize) + 1
	if scaledIterations < 1 {
		scaledIterations = 1
	}
	if settings.Verbose {
		fmt.Printf("  Smoothing streamfunction (%d iterations, cellSize=%.4f)...\n", scaledIterations, cellSize)
	}
	psi = SmoothStreamfunction(
		psi, vertices, elevation, seaLevelThreshold, adj,
		scaledIterations, settings.SmoothingFactor,
	)

	if settings.Verbose {
		minPsi, maxPsi := psi[0], psi[0]
		for _, p := range psi {
			if p < minPsi {
				minPsi = p
			}
			if p > maxPsi {
				maxPsi = p
			}
		}
		fmt.Printf("  Smoothed Ψ range: [%.4f, %.4f]\n", minPsi, maxPsi)
	}

	if settings.Verbose {
		fmt.Println("  Computing velocity from ∇Ψ...")
	}
	velocity := ComputeVelocityFromStreamfunction(
		psi, vertices, elevation, seaLevelThreshold, adj,
	)
	coastLandDirs := CalculateCoastlineLandDirs(vertices, elevation, seaLevelThreshold, adj)
	velocity = ApplyOceanStructure(
		velocity, vertices, elevation, seaLevelThreshold, adj, coastLandDirs, componentAssignments, components,
	)
	velocity = EnforceCoastParallelFlow(
		velocity, vertices, elevation, seaLevelThreshold, coastLandDirs, settings.CoastParallelBoost, settings.MaxAllowedSpeedSq,
	)

	if settings.Verbose {
		maxSpeed := 0.0
		for _, v := range velocity {
			speed := Length(v)
			if speed > maxSpeed {
				maxSpeed = speed
			}
		}
		fmt.Printf("  Raw velocity max: %.4f\n", maxSpeed)
	}

	const velocitySmoothAngular = 0.035
	velocitySmoothIters := int(velocitySmoothAngular/cellSize) + 1
	if velocitySmoothIters < 2 {
		velocitySmoothIters = 2
	}
	if settings.Verbose {
		fmt.Printf("  Smoothing velocity field (%d iterations)...\n", velocitySmoothIters)
	}
	velocity = SmoothCurrents(
		velocity, vertices, elevation, seaLevelThreshold, adj,
		coastLandDirs,
		CurrentSettings{
			SmoothingIterations: velocitySmoothIters,
			SmoothingFactor:     0.35,
			CoastParallelBoost:  settings.CoastParallelBoost,
			MaxAllowedSpeedSq:   settings.MaxAllowedSpeedSq,
		},
	)
	velocity = EnforceCoastParallelFlow(
		velocity, vertices, elevation, seaLevelThreshold, coastLandDirs, settings.CoastParallelBoost, settings.MaxAllowedSpeedSq,
	)

	if settings.Verbose {
		maxSpeed := 0.0
		for _, v := range velocity {
			speed := Length(v)
			if speed > maxSpeed {
				maxSpeed = speed
			}
		}
		fmt.Printf("  Smoothed velocity max: %.4f\n", maxSpeed)
	}

	return velocity
}

// EnforceCoastParallelFlow removes the normal-to-coast component of nearshore
// currents using precomputed tangent directions toward land.
func EnforceCoastParallelFlow(
	currents []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	coastLandDirs []Vector3D,
	coastParallelBoost float64,
	maxAllowedSpeedSq float64,
) []Vector3D {
	constrained := make([]Vector3D, len(currents))
	copy(constrained, currents)

	for i, current := range constrained {
		if elevation[i] >= seaLevelThreshold || i >= len(coastLandDirs) {
			continue
		}

		landDir := coastLandDirs[i]
		if LengthSq(landDir) <= 1e-12 {
			continue
		}

		dotPerp := Dot(current, landDir) / LengthSq(landDir)
		parallel := Scale(Sub(current, Scale(landDir, dotPerp)), coastParallelBoost)
		surfaceNormal := vertices[i]
		dotNormal := Dot(parallel, surfaceNormal)
		parallel = Sub(parallel, Scale(surfaceNormal, dotNormal))
		parallelLenSq := LengthSq(parallel)
		if !IsFinite(parallelLenSq) || parallelLenSq > maxAllowedSpeedSq {
			constrained[i] = Vector3D{}
			continue
		}
		constrained[i] = parallel
	}

	return constrained
}

// SmoothCurrents performs iterative smoothing passes to diffuse ocean currents.
// Uses neighbor averaging with coast-parallel boundary conditions.
func SmoothCurrents(
	currents []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	coastLandDirs []Vector3D,
	settings CurrentSettings,
) []Vector3D {
	numVertices := len(vertices)
	smoothed := make([]Vector3D, numVertices)
	copy(smoothed, currents)

	next := make([]Vector3D, numVertices)

	for iter := 0; iter < settings.SmoothingIterations; iter++ {
		for i := 0; i < numVertices; i++ {
			if elevation[i] >= seaLevelThreshold {
				continue // Skip land
			}

			// Average neighbor currents (only water neighbors)
			var neighborSum Vector3D
			waterNeighborCount := 0

			for _, k := range adj.GetNeighbors(i) {
				if k >= 0 && k < numVertices && elevation[k] < seaLevelThreshold {
					neighborSum = Add(neighborSum, smoothed[k])
					waterNeighborCount++
				}
			}

			if waterNeighborCount == 0 {
				next[i] = smoothed[i]
				continue
			}

			avgNeighbor := Scale(neighborSum, 1.0/float64(waterNeighborCount))

			// Blend current value with neighbor average
			blend := Add(
				Scale(smoothed[i], 1.0-settings.SmoothingFactor),
				Scale(avgNeighbor, settings.SmoothingFactor),
			)

			// Check if this is a coastline vertex
			landDir := coastLandDirs[i]
			landDirLenSq := LengthSq(landDir)

			if landDirLenSq > 1e-12 {
				// Coastline vertex: remove component perpendicular to coast
				dotPerp := Dot(blend, landDir) / landDirLenSq
				parallel := Sub(blend, Scale(landDir, dotPerp))
				surfaceNormal := vertices[i]
				dotNormal := Dot(parallel, surfaceNormal)
				parallel = Sub(parallel, Scale(surfaceNormal, dotNormal))

				// Apply coast parallel flow boost
				parallel = Scale(parallel, settings.CoastParallelBoost)

				// Clamp for stability
				parallelLenSq := LengthSq(parallel)
				if !IsFinite(parallelLenSq) || parallelLenSq > settings.MaxAllowedSpeedSq {
					parallel = Vector3D{}
				}

				next[i] = parallel
			} else {
				// Interior water vertex: project onto tangent plane
				surfaceNormal := vertices[i] // Already unit sphere
				dotNormal := Dot(blend, surfaceNormal)
				tangent := Sub(blend, Scale(surfaceNormal, dotNormal))

				// Clamp for stability
				tangentLenSq := LengthSq(tangent)
				if !IsFinite(tangentLenSq) || tangentLenSq > settings.MaxAllowedSpeedSq {
					tangent = Vector3D{}
				}

				next[i] = tangent
			}
		}

		// Swap buffers
		smoothed, next = next, smoothed
	}

	return smoothed
}
