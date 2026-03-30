package climgen

import "math"

const (
	precipConvergenceSmoothIters = 4
	precipConvergenceSmoothBlend = 0.42
)

func computeClimateConvergenceField(
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
) []float64 {
	raw := make([]float64, len(vertices))
	for i := range vertices {
		raw[i] = computeWindConvergence(i, vertices, adj, wind)
	}
	smoothed := SmoothScalarField(raw, vertices, adj, precipConvergenceSmoothIters, precipConvergenceSmoothBlend)
	out := make([]float64, len(vertices))
	for i, v := range vertices {
		absLat := math.Abs(getLatitudeDeg(v))
		tropicalBlend := 1.0 - smoothRamp(16.0, 34.0, absLat)
		out[i] = Clamp(raw[i]*(1.0-0.65*tropicalBlend)+smoothed[i]*(0.65*tropicalBlend), -1, 1)
	}
	return out
}
