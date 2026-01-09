package climgen

import (
	"fmt"
	"time"
)

// GenerateOceanCurrents runs the complete ocean current generation pipeline.
// It finds ocean basins, applies gyre rotation forcing, and smooths the currents.
func GenerateOceanCurrents(
	vertices []Vector3D,
	cells []VoronoiCell,
	elevation []float64,
	seaLevelThreshold float64,
	settings OceanCurrentSettings,
) (*OceanCurrentResult, error) {
	settings.ApplyVerbose()

	if err := settings.Validate(); err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	numVertices := len(vertices)

	if settings.Verbose {
		fmt.Println("=== Ocean Current Generation Pipeline ===")
		fmt.Printf("  Vertices: %d\n", numVertices)
	}

	startTime := time.Now()

	// Step 1: Build flat adjacency for efficient iteration
	if settings.Verbose {
		fmt.Println("Step 1: Building flat adjacency structure...")
	}
	adj := BuildFlatAdjacency(cells)

	// NOTE: Basin detection is no longer used for current generation.
	// Wind-driven Sverdrup model creates gyres from latitude-based wind stress.
	// Basin code is preserved for future use: ocean body identification,
	// naming separate oceans/seas, temperature/salinity distribution, etc.
	// See basins.go for the detection algorithm.

	// Step 2: Generate currents using wind-driven streamfunction approach
	// This guarantees divergence-free flow and coast-parallel boundary conditions
	// Gyres emerge naturally from wind stress curl patterns
	if settings.Verbose {
		fmt.Println("Step 2: Generating currents via wind-driven streamfunction...")
	}
	streamStart := time.Now()
	smoothedCurrents := GenerateCurrentsStreamfunction(
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		nil, // basins not used in wind-driven approach
		settings.Current,
	)
	if settings.Verbose {
		fmt.Printf("  Streamfunction generation took: %.2fs\n", time.Since(streamStart).Seconds())
	}

	// Create empty basin data for result (preserved for API compatibility)
	basinAssignments := make([]int, numVertices)
	for i := range basinAssignments {
		basinAssignments[i] = -1
	}
	var basins []Basin

	// Calculate final statistics and quality metrics
	if settings.Verbose {
		maxSpeed := 0.0
		nonZeroCount := 0
		for _, c := range smoothedCurrents {
			speed := Length(c)
			if speed > 1e-9 {
				nonZeroCount++
				if speed > maxSpeed {
					maxSpeed = speed
				}
			}
		}
		fmt.Printf("\n=== Generation Complete ===\n")
		fmt.Printf("  Total time: %.2fs\n", time.Since(startTime).Seconds())
		fmt.Printf("  Basins: %d\n", len(basins))
		fmt.Printf("  Vertices with currents: %d\n", nonZeroCount)
		fmt.Printf("  Max speed: %.4f\n", maxSpeed)

		// Compute coalescence quality metrics
		metrics := ComputeCoalescenceMetrics(
			smoothedCurrents, vertices, elevation, seaLevelThreshold, adj, basins,
		)
		fmt.Printf("\n=== Coalescence Quality Metrics ===\n")
		fmt.Printf("  Flow coherence:       %.3f (1.0 = perfect alignment)\n", metrics.FlowCoherence)
		fmt.Printf("  Multi-basin vertices: %.1f%% (higher = better blending)\n", metrics.MultiBsinPct*100)
		fmt.Printf("  Speed uniformity:     %.3f (lower CoV = more uniform)\n", metrics.SpeedCoV)
		fmt.Printf("  Boundary smoothness:  %.3f (1.0 = seamless transitions)\n", metrics.BoundarySmoothness)
		fmt.Printf("  Avg vorticity:        %.4f (higher = more circular)\n", metrics.AvgVorticity)
		fmt.Printf("  Vorticity ratio:      %.3f (0=linear, 1=circular)\n", metrics.VorticityRatio)
	}

	return &OceanCurrentResult{
		Currents:         smoothedCurrents,
		BasinAssignments: basinAssignments,
		Basins:           basins,
	}, nil
}

// GenerateOceanCurrentsSimple is a convenience wrapper using default settings.
func GenerateOceanCurrentsSimple(
	vertices []Vector3D,
	cells []VoronoiCell,
	elevation []float64,
) (*OceanCurrentResult, error) {
	settings := DefaultOceanCurrentSettings()
	settings.Verbose = true
	return GenerateOceanCurrents(vertices, cells, elevation, 0.0, settings)
}
