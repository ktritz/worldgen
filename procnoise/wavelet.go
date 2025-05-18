package procnoise

import (
	"math/rand"
	"worldgen/icosphere"
)

// interpolateOnSphere performs barycentric interpolation of noise values on a single SphericalNoiseLayer.
// p: a 3D point in space. It will be normalized to the unit sphere for lookup.
// Uses icosphere.Vector3D.
func (swn *SphericalWaveletNoise) interpolateOnSphere(layer *SphericalNoiseLayer, p icosphere.Vector3D) float32 {
	pOnSphere := p.Normalize()
	v0Idx, v1Idx, v2Idx, bary, found := layer.Grid.FindTriangleAndBarycentricCoords(pOnSphere)

	if !found {
		return 0
	}
	if v0Idx < 0 || v0Idx >= len(layer.Values) ||
		v1Idx < 0 || v1Idx >= len(layer.Values) ||
		v2Idx < 0 || v2Idx >= len(layer.Values) {
		return 0
	}

	val0 := layer.Values[v0Idx]
	val1 := layer.Values[v1Idx]
	val2 := layer.Values[v2Idx]

	return bary.U*val0 + bary.V*val1 + bary.W*val2
}

// GetNoise evaluates the Spherical Wavelet Noise at point p.
// p is a 3D point in space. Its projection onto the unit sphere is used for noise evaluation.
// Uses icosphere.Vector3D.
func (swn *SphericalWaveletNoise) GetNoise(p icosphere.Vector3D) float32 {
	var totalNoise float32
	for _, layer := range swn.NoiseLayers {
		totalNoise += swn.interpolateOnSphere(layer, p)
	}
	return totalNoise
}

// SphericalWaveletNoise generates noise by summing contributions from multiple SphericalNoiseLayers.
type SphericalWaveletNoise struct {
	NoiseLayers []*SphericalNoiseLayer
}

// NewSphericalWaveletNoise creates a new Spherical Wavelet Noise generator.
// - seed: For the random number generator.
// - grids: A slice of SphericalGrid objects, ordered from FINEST to COARSEST.
func NewSphericalWaveletNoise(seed int64, grids []SphericalGrid) *SphericalWaveletNoise {
	if len(grids) == 0 {
		panic("At least one SphericalGrid must be provided")
	}

	rng := rand.New(rand.NewSource(seed))
	numOctaves := len(grids)
	layers := make([]*SphericalNoiseLayer, numOctaves)

	if grids[0] == nil {
		panic("Finest grid (grids[0]) cannot be nil")
	}
	layers[0] = NewSphericalNoiseLayerRandom(grids[0], rng)

	for i := 1; i < numOctaves; i++ {
		if grids[i] == nil {
			panic("Grid (grids[i]) cannot be nil")
		}
		if layers[i-1] == nil {
			panic("Previous noise layer is nil, cannot downsample")
		}
		layers[i] = downsampleSphericalLayer(layers[i-1], grids[i])
	}

	return &SphericalWaveletNoise{
		NoiseLayers: layers,
	}
}
