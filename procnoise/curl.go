package procnoise

import (
	"worldgen/icosphere"
)

// --- Curl Noise Implementation ---

// ScalarField3D defines an interface for any 3D scalar noise field.
// SphericalWaveletNoise and FastNoiseLiteScalarField (defined below) satisfy this interface.
type ScalarField3D interface {
	GetNoise(p icosphere.Vector3D) float32
}

// CurlNoiseGenerator3D generates 3D curl noise.
type CurlNoiseGenerator3D struct {
	PotentialX ScalarField3D
	PotentialY ScalarField3D
	PotentialZ ScalarField3D
	Epsilon    float64
}

// NewCurlNoiseGenerator3D creates a new 3D Curl Noise generator.
//   - potentialX, potentialY, potentialZ: Scalar noise fields for the components of the vector potential ψ.
//     These can be SphericalWaveletNoise instances, FastNoiseLiteScalarField instances,
//     or any other type implementing ScalarField3D.
//     Example using SphericalWaveletNoise:
//     psiX := procnoise.NewSphericalWaveletNoise(seed1, grids)
//     curlGen := procnoise.NewCurlNoiseGenerator3D(psiX, psiY, psiZ, epsilon)
//     Example using FastNoiseLiteScalarField:
//     fnlStateX := procnoise.New[float32]() // procnoise refers to FastNoiseLite here
//     fnlStateX.NoiseType(procnoise.OpenSimplex2)
//     fnlStateX.Frequency = 0.05
//     psiX_fnl := procnoise.NewFastNoiseLiteScalarField(fnlStateX)
//     curlGen := procnoise.NewCurlNoiseGenerator3D(psiX_fnl, psiY_fnl, psiZ_fnl, epsilon)
//   - epsilon: A small step size used for approximating derivatives via finite differences.
func NewCurlNoiseGenerator3D(potentialX, potentialY, potentialZ ScalarField3D, epsilon float64) *CurlNoiseGenerator3D {
	if potentialX == nil || potentialY == nil || potentialZ == nil {
		panic("Potential field components (potentialX, potentialY, potentialZ) cannot be nil")
	}
	if epsilon <= 1e-9 {
		panic("Epsilon must be a small positive value (e.g., 0.001 or 0.01)")
	}
	return &CurlNoiseGenerator3D{
		PotentialX: potentialX,
		PotentialY: potentialY,
		PotentialZ: potentialZ,
		Epsilon:    epsilon,
	}
}

// GetCurl evaluates the 3D curl noise vector at point p.
func (cg *CurlNoiseGenerator3D) GetCurl(p icosphere.Vector3D) icosphere.Vector3D {
	eps := cg.Epsilon
	twoEps := 2.0 * eps

	psiZ_pyPlus := cg.PotentialZ.GetNoise(icosphere.Vector3D{X: p.X, Y: p.Y + eps, Z: p.Z})
	psiZ_pyMinus := cg.PotentialZ.GetNoise(icosphere.Vector3D{X: p.X, Y: p.Y - eps, Z: p.Z})
	psiY_pzPlus := cg.PotentialY.GetNoise(icosphere.Vector3D{X: p.X, Y: p.Y, Z: p.Z + eps})
	psiY_pzMinus := cg.PotentialY.GetNoise(icosphere.Vector3D{X: p.X, Y: p.Y, Z: p.Z - eps})

	psiX_pzPlus := cg.PotentialX.GetNoise(icosphere.Vector3D{X: p.X, Y: p.Y, Z: p.Z + eps})
	psiX_pzMinus := cg.PotentialX.GetNoise(icosphere.Vector3D{X: p.X, Y: p.Y, Z: p.Z - eps})
	psiZ_pxPlus := cg.PotentialZ.GetNoise(icosphere.Vector3D{X: p.X + eps, Y: p.Y, Z: p.Z})
	psiZ_pxMinus := cg.PotentialZ.GetNoise(icosphere.Vector3D{X: p.X - eps, Y: p.Y, Z: p.Z})

	psiY_pxPlus := cg.PotentialY.GetNoise(icosphere.Vector3D{X: p.X + eps, Y: p.Y, Z: p.Z})
	psiY_pxMinus := cg.PotentialY.GetNoise(icosphere.Vector3D{X: p.X - eps, Y: p.Y, Z: p.Z})
	psiX_pyPlus := cg.PotentialX.GetNoise(icosphere.Vector3D{X: p.X, Y: p.Y + eps, Z: p.Z})
	psiX_pyMinus := cg.PotentialX.GetNoise(icosphere.Vector3D{X: p.X, Y: p.Y - eps, Z: p.Z})

	dPsiZ_dy := (psiZ_pyPlus - psiZ_pyMinus) / float32(twoEps)
	dPsiY_dz := (psiY_pzPlus - psiY_pzMinus) / float32(twoEps)
	dPsiX_dz := (psiX_pzPlus - psiX_pzMinus) / float32(twoEps)
	dPsiZ_dx := (psiZ_pxPlus - psiZ_pxMinus) / float32(twoEps)
	dPsiY_dx := (psiY_pxPlus - psiY_pxMinus) / float32(twoEps)
	dPsiX_dy := (psiX_pyPlus - psiX_pyMinus) / float32(twoEps)

	curlX := float64(dPsiZ_dy - dPsiY_dz)
	curlY := float64(dPsiX_dz - dPsiZ_dx)
	curlZ := float64(dPsiY_dx - dPsiX_dy)

	return icosphere.Vector3D{X: curlX, Y: curlY, Z: curlZ}
}

// GetTangentCurl evaluates the 3D curl noise and projects it onto the tangent plane
// of the sphere at point p.
func (cg *CurlNoiseGenerator3D) GetTangentCurl(p icosphere.Vector3D) icosphere.Vector3D {
	curlVector3D := cg.GetCurl(p)
	pNormal := p.Normalize()
	dotProd := curlVector3D.Dot(pNormal)
	projectionOntoNormal := pNormal.Scale(dotProd)
	tangentCurl := curlVector3D.Subtract(projectionOntoNormal)
	return tangentCurl
}
