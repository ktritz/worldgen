package climgen

import "math"

// =============================================================================
// TEMPERATURE - WIND ADVECTION
// =============================================================================
// Computes temperature change from wind-driven heat transport over land.
// Heat is advected in the wind direction - a location receives heat from upwind.
//
// This is particularly important for:
//   - Westerlies bringing maritime air to west coasts (warming in winter)
//   - Trade winds bringing warm air across oceans

// Physical constants for wind advection
const (
	// Maximum surface wind speed in m/s
	// Trade winds ~6-8 m/s, Westerlies ~10-15 m/s, storms up to 30 m/s
	// Use moderate value for typical conditions
	MaxSurfaceWindSpeed = 15.0 // m/s
)

// ComputeWindAdvection calculates temperature change from wind-driven heat transport.
// Uses physically-based advection: dT/dt = -v · ∇T
// Discretized as: ΔT = v * dt / cellSize * (T_upwind - T_here)
func ComputeWindAdvection(
	temperature []float64,
	vertices []Vector3D,
	wind []Vector3D,
	adj *FlatAdjacency,
	settings TransportSettings,
) []float64 {
	numVertices := len(temperature)
	advection := make([]float64, numVertices)

	if wind == nil || settings.WindTransportScale < 1e-9 {
		return advection
	}

	// Compute average cell size in meters for physical advection
	earthRadius := 6371.0e3 // meters
	avgCellSize := earthRadius * math.Sqrt(4*math.Pi/float64(numVertices))

	for i := 0; i < numVertices; i++ {
		windVec := wind[i]
		normalizedSpeed := Length(windVec)

		if normalizedSpeed < 0.01 {
			continue
		}

		// Convert normalized speed (0-1) to physical velocity (m/s)
		physicalSpeed := normalizedSpeed * MaxSurfaceWindSpeed

		windDir := Scale(windVec, 1.0/normalizedSpeed)
		neighbors := adj.GetNeighbors(i)
		if len(neighbors) == 0 {
			continue
		}

		// Compute upwind-weighted temperature
		upwindSum := 0.0
		upwindWeight := 0.0

		for _, k := range neighbors {
			if k < 0 || k >= numVertices {
				continue
			}

			dirFromK := Sub(vertices[i], vertices[k])
			dist := Length(dirFromK)
			if dist < 1e-9 {
				continue
			}
			dirFromK = Scale(dirFromK, 1.0/dist)

			upwindness := Dot(windDir, dirFromK)
			if upwindness > 0 {
				weight := upwindness
				upwindSum += temperature[k] * weight
				upwindWeight += weight
			}
		}

		if upwindWeight > 1e-9 {
			upwindTemp := upwindSum / upwindWeight

			// Physical advection coefficient: v * dt / cellSize
			// This is dimensionless and resolution-independent
			advectionCoeff := physicalSpeed * AdvectionTimeStep / avgCellSize

			// Apply transport scale as a tuning multiplier
			advection[i] = settings.WindTransportScale * advectionCoeff *
				(upwindTemp - temperature[i])
		}
	}

	return advection
}
