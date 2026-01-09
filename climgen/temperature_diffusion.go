package climgen

import "math"

// =============================================================================
// TEMPERATURE - HEAT DIFFUSION
// =============================================================================
// Computes heat diffusion (neighbor averaging) with latitude-dependent coefficient.
// Based on Budyko/Sellers diffusive energy balance model approach.

// ComputeHeatDiffusion calculates temperature change from neighbor diffusion.
// Uses a physically-based diffusion coefficient that varies with latitude
// (stronger at mid-latitudes where meridional gradients are steepest).
//
// Returns the temperature change (K) at each vertex.
func ComputeHeatDiffusion(
	temperature []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	settings TransportSettings,
) []float64 {
	numVertices := len(temperature)
	diffusion := make([]float64, numVertices)

	for i := 0; i < numVertices; i++ {
		// Base diffusion coefficient
		var D float64
		if elevation[i] >= seaLevelThreshold {
			D = settings.DiffusionLand
		} else {
			D = settings.DiffusionWater
		}

		if D < 1e-12 {
			continue
		}

		// Latitude-dependent scaling: stronger diffusion at mid-latitudes
		// where meridional temperature gradients are largest
		lat := math.Abs(getLatitude(vertices[i]))
		latScale := 1.0 + 0.5*math.Sin(2*lat) // Peak at 45°

		neighbors := adj.GetNeighbors(i)
		if len(neighbors) == 0 {
			continue
		}

		sumTemp := 0.0
		count := 0
		for _, k := range neighbors {
			if k >= 0 && k < numVertices {
				sumTemp += temperature[k]
				count++
			}
		}

		if count == 0 {
			continue
		}

		avgNeighborTemp := sumTemp / float64(count)
		diffusion[i] = D * latScale * (avgNeighborTemp - temperature[i])
	}

	return diffusion
}
