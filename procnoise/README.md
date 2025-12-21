<!-----



Conversion time: 0.888 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β44
* Sat May 17 2025 18:06:01 GMT-0700 (PDT)
* Source doc: ok, recreate the README in standard Google Doc fo...
----->



# procnoise Module for Procedural World Generation

The procnoise Go module provides a suite of noise generation algorithms and utilities specifically tailored for procedural world generation. It aims to offer diverse tools for creating complex and naturalistic patterns suitable for various aspects of world building, from terrain and textures to fluid dynamics and resource distribution.

This document outlines the purpose and interface of each major component within the procnoise module.


## Core Concepts


### ScalarField3D Interface

A central concept in parts of this module is the ScalarField3D interface, defined in curl.go (and used by flow.go):

type ScalarField3D interface { \
    GetNoise(p icosphere.Vector3D) float32 \
} \


This interface allows different 3D scalar noise generators (like Spherical Wavelet Noise or an adapted FastNoiseLite instance) to be used interchangeably, particularly as potential fields for Curl Noise or as base noise for Flow Noise.


## Modules and Components

The procnoise module is composed of several key files, each providing distinct noise generation capabilities or utilities:


### 1. FastNoiseLite (fastnoise.go)

This component is a Go implementation of the FastNoiseLite library, offering a versatile and high-performance collection of noise algorithms.

**Purpose & Key Features:**



* **Multiple Noise Types:** Supports various foundational noise algorithms:
    * OpenSimplex2 and OpenSimplex2S: Modern simplex noise variants with good performance and reduced artifacts.
    * Cellular (Worley Noise): Generates patterns based on distance to feature points, excellent for cellular structures, cracks, or regional delineation. It offers multiple distance functions (Euclidean, Manhattan, etc.) and return types (CellValue, Distance, Distance2, Distance2Add, Distance2Sub, Distance2Mul, Distance2Div).
    * Perlin: Classic gradient noise.
    * ValueCubic and Value: Value-based noise types.
* **Fractal Noise:** Provides several fractal types to combine noise octaves for added detail:
    * FractalFBm (Fractional Brownian Motion): Standard fractal noise.
    * FractalRidged: Creates sharp ridges and valleys.
    * FractalPingPong: Produces a "ping-pong" effect on noise values.
    * FractalDomainWarpProgressive and FractalDomainWarpIndependent: Applies domain warping as a fractal type.
* **Domain Warping:** Allows distortion of the input domain using another noise source, leading to more complex and organic patterns. Supports DomainWarpOpenSimplex2, DomainWarpOpenSimplex2Reduced, and DomainWarpBasicGrid.
* **Configurability:** Noise generation is controlled via a State[T] object (where T can be float32 or float64), allowing detailed configuration of:
    * Seed, Frequency, NoiseType, FractalType
    * Fractal parameters: Octaves, Lacunarity, Gain, WeightedStrength, PingPongStrength
    * Cellular parameters: CellularDistanceFunc, CellularReturnType, CellularJitterMod
    * Domain Warp parameters: DomainWarpType, DomainWarpAmp
    * 3D Rotation: RotationType3D for transforming 3D noise coordinates.

**Interfacing:**



* **Initialization:** Create a new noise state using procnoise.New[float32]() or procnoise.New[float64]().
* **Configuration:** Set parameters directly on the State object (e.g., state.Frequency = 0.02, state.Octaves = 5) or use setter methods like state.NoiseType(procnoise.OpenSimplex2) and state.FractalType(procnoise.FractalFBm). Changes to NoiseType or FractalType automatically reconfigure the internal noise functions.
* **Noise Generation:**
    * state.GetNoise2D(x, y T) T
    * state.GetNoise3D(x, y, z T) T
    * Integer coordinate versions: state.Noise2D(x, y int) T, state.Noise3D(x, y, z int) T
* **Domain Warping (Direct):**
    * state.DomainWarp2D(x, y T) (T, T)
    * state.DomainWarp3D(x, y, z T) (T, T, T)
* **ScalarField3D Adapter:**
    * FastNoiseLiteScalarField: An adapter to use a *State[float32] as a ScalarField3D.
    * NewFastNoiseLiteScalarField(fnlState *State[float32]) *FastNoiseLiteScalarField

World Generation Context:

FastNoiseLite is suitable for a vast range of applications:



* Generating base terrain heightmaps (Simplex, Perlin, Value with FBM/Ridged).
* Creating textures for clouds, water, rock.
* Defining temperature, rainfall, or biome suitability maps.
* Cellular noise for tectonic plate outlines, resource clustering, or specific biome patterns (e.g., F2-F1 for boundaries).
* Domain warping for more complex terrain deformation, turbulent fluid appearances, or irregular resource distributions.


### 2. Spherical Wavelet Noise (wavelet.go and utils.go)

This component implements Spherical Wavelet Noise, designed for generating noise directly on the surface of a sphere. It's particularly useful for planet-scale procedural generation.

**Purpose & Key Features:**



* **Spherical Domain:** Operates on spherical grids (typically icospheres of varying subdivision levels).
* **Layered Approach:** Noise is generated by summing contributions from multiple SphericalNoiseLayers. Each layer corresponds to a different level of detail (frequency band).
* **Construction:**
    * The finest layer is initialized with random values at its grid vertices.
    * Subsequent coarser layers are generated by downsampling (averaging) values from the preceding finer layer.
* **Interpolation:** Noise values at arbitrary points on the sphere are obtained by barycentric interpolation within the triangles of each grid layer.

**Interfacing:**



* **Dependencies (from utils.go):**
    * SphericalGrid interface: Defines methods for spherical grid geometry (NumVertices, GetVertex, FindTriangleAndBarycentricCoords, GetVertexNeighbors).
    * IcosphereModel: An implementation of SphericalGrid using icosphere vertex and face data. Created with NewIcosphereModel(vertices []icosphere.Vector3D, faces []icosphere.Triangle).
    * SphericalNoiseLayer: Stores noise values for a single spherical grid.
* **Initialization:**
    * NewSphericalWaveletNoise(seed int64, grids []SphericalGrid) *SphericalWaveletNoise
        * grids: A slice of SphericalGrid objects, ordered from **finest to coarsest**.
* **Noise Generation:**
    * swn.GetNoise(p icosphere.Vector3D) float32: Evaluates the noise at point p. The point is normalized to the unit sphere.

**World Generation Context:**



* Generating planet-scale elevation maps or other scalar fields directly on a sphere, avoiding distortions from 2D map projections.
* Creating layered phenomena on a sphere, like atmospheric layers or large-scale geological features.
* Its GetNoise method makes it compatible with the ScalarField3D interface, allowing it to be used as a potential for Curl Noise or base noise for Flow Noise in spherical contexts.


### 3. Curl Noise (curl.go)

This component implements 3D Curl Noise, which generates divergence-free vector fields. Such fields are excellent for simulating incompressible fluid flow.

**Purpose & Key Features:**



* **Divergence-Free Vector Fields:** The primary characteristic is that the resulting vector field has no sources or sinks, mimicking natural fluid motion.
* **Potential Field Based:** Curl noise is derived by taking the curl of a 3D vector potential field (ψ).
* **ScalarField3D Dependency:** Each component of the vector potential (ψx, ψy, ψz) is defined by a ScalarField3D (e.g., Spherical Wavelet Noise or an adapted FastNoiseLite instance).
* **Finite Differences:** Partial derivatives for the curl calculation are approximated using finite differences.

**Interfacing:**



* **CurlNoiseGenerator3D Struct:**
    * PotentialX, PotentialY, PotentialZ ScalarField3D
    * Epsilon float64: Small step size for finite differences.
* **Initialization:**
    * NewCurlNoiseGenerator3D(potentialX, potentialY, potentialZ ScalarField3D, epsilon float64) *CurlNoiseGenerator3D
* **Noise Generation (Vector Output):**
    * cg.GetCurl(p icosphere.Vector3D) icosphere.Vector3D: Evaluates the 3D curl noise vector at point p.
    * cg.GetTangentCurl(p icosphere.Vector3D) icosphere.Vector3D: Evaluates the curl and projects it onto the tangent plane of a sphere at point p. Useful for spherical flow.

**World Generation Context:**



* Simulating wind patterns and air circulation for climate models.
* Generating ocean currents.
* Creating vector fields for particle advection in erosion simulations or visual effects (e.g., smoke, dust).
* The GetTangentCurl method is particularly useful for generating flow fields constrained to the surface of a sphere.


### 4. Flow Noise (flow.go)

This component implements advection-based Flow Noise, creating animated scalar noise by advecting lookup coordinates into a base scalar noise field.

**Purpose & Key Features:**



* **Animated Noise:** Produces dynamic, flowing patterns.
* **Advection-Based:** Achieved by displacing the sampling point in a BaseNoise field using a time-dependent AdvectionSource.
* **Modular Components:**
    * BaseNoise ScalarField3D: The underlying static scalar noise to be sampled.
    * AdvectionSource func(p icosphere.Vector3D, time float64) icosphere.Vector3D: A function providing the displacement vector field, dependent on position and time.
    * AdvectionStrength float64: Scales the displacement.

**Interfacing:**



* **FlowNoiseGenerator Struct:**
    * BaseNoise ScalarField3D
    * AdvectionFunc AdvectionSource
    * AdvectionStrength float64
* **Initialization:**
    * NewFlowNoiseGenerator(baseNoise ScalarField3D, advectionFunc AdvectionSource, advectionStrength float64) *FlowNoiseGenerator
* **Noise Generation (Scalar Output):**
    * fg.GetNoise(p icosphere.Vector3D, time float64) float32: Evaluates the animated flow noise at point p and time t.
* **Example Advection Sources:**
    * StaticCurlAdvectionSource(curlNoiseGen *CurlNoiseGenerator3D) AdvectionSource: Creates an advection source from a non-animated CurlNoiseGenerator3D. The advection field will be tangent to the sphere.
    * AnimatedCurlAdvectionSource(potentialXFunc, potentialYFunc, potentialZFunc AnimatedPotentialFunc, epsilon float64) AdvectionSource: Creates an advection source where the curl noise field itself is animated.
        * This uses AnimatedPotentialFunc func(p icosphere.Vector3D, time float64) float32 for the curl potentials.
        * TimeVaryingPotentialProvider can adapt a FastNoiseLite.State[float32] to be an AnimatedPotentialFunc by incorporating time into one of its spatial coordinates.
            * NewTimeVaryingPotentialProvider(fnlState *State[float32], timeScale float32, inputScale float32, timeAxis int) *TimeVaryingPotentialProvider
            * tvpp.GetNoiseWithTime(p icosphere.Vector3D, time float64) float32
        * AnimatedScalarFieldAdapter adapts an AnimatedPotentialFunc to ScalarField3D for a fixed time.

**World Generation Context:**



* Animating wind patterns, ocean currents, or river flow over time.
* Visualizing evolving weather systems, cloud formations, or atmospheric particle dispersal.
* Creating dynamic textures for water surfaces, flowing lava, etc.


### 5. Gabor Noise (gabor.go)

This component implements Gabor Noise in 2D and 3D, a type of sparse convolution noise known for its ability to create anisotropic (directional) patterns.

**Purpose & Key Features:**



* **Anisotropic Patterns:** Excellent for features with inherent directionality, like wood grain, muscle fibers, or certain geological formations.
* **Impulse-Based:** Generated by distributing random "impulses" (Gabor kernels) in space and summing their contributions.
* **Configurable Impulses:** Each impulse has parameters for:
    * Position, Weight
    * Frequency, Orientation (2D angle or 3D quaternion)
    * Bandwidth, Phase
    * AspectRatio (2D) or AspectRatioY/AspectRatioZ (3D) to control ellipticity/ellipsoidal shape.
    * CutoffSq: Squared cutoff radius for culling.
* **Configuration Structs:** ImpulseConfig2D and ImpulseConfig3D define ranges for randomizing impulse parameters.

**Interfacing:**



* **Structs:**
    * Impulse2D, Impulse3D
    * ImpulseConfig2D, ImpulseConfig3D
    * GaborNoise2D, GaborNoise3D (contain a slice of impulses and an RNG)
* **Initialization:**
    * NewGaborNoise2D(seed int64, cfg ImpulseConfig2D) *GaborNoise2D
    * NewGaborNoise3D(seed int64, cfg ImpulseConfig3D) *GaborNoise3D
* **Noise Generation (Scalar Output):**
    * gn2D.GetNoise(p Vec2) float64
    * gn3D.GetNoise(p Vec3) float64
    * Note: Vec2 and Vec3 are simple vector types defined in utils.go.

**World Generation Context:**



* Generating anisotropic terrain features like elongated sand dunes, wind-eroded ridges, or glacial scouring patterns.
* Creating textures for materials with directionality: wood grain, brushed metal, muscle fibers, aligned grasses.
* Modeling directional resource deposits, such as layered mineral veins following geological orientations.


### 6. Utilities (utils.go)

This file provides various helper functions and types used by the other noise modules. It does not generate noise directly but is crucial for the operation of modules like wavelet.go and gabor.go.

**Key Provisions:**



* **Mathematical Utilities:** fastMin, fastMax, fastAbs, fastSqrt, fastFloor, fastRound, lerp, interpHermite, interpQuintic, cubicLerp, pingPong, calculateFractalBounding.
* **Vector Math:** Vec2 and Vec3 types with basic operations (Sub, LengthSq).
* **Quaternion Math:** Quaternion type with NewQuaternionIdentity, Inverse, Rotate, FromAngleAxis. Used by 3D Gabor noise for orientation.
* **Spherical Grid Abstractions:**
    * BarycentricCoords struct.
    * SphericalGrid interface (described under Spherical Wavelet Noise).
    * IcosphereModel struct (implementation of SphericalGrid).
    * SphericalNoiseLayer struct (used by Spherical Wavelet Noise).
    * Helper functions: calculateBarycentric, NewSphericalNoiseLayerRandom, downsampleSphericalLayer.

Interfacing:

These utilities are primarily used internally by other procnoise components. Users creating custom spherical grids for SphericalWaveletNoise would need to implement the SphericalGrid interface.


## General Usage Notes



* **Combining Noises:** The true power of procedural generation often comes from combining different noise types. For example:
    * Use FastNoiseLite (Simplex or Perlin with FBM) for base terrain elevation.
    * Apply Worley noise (F2-F1) to define river networks or fault lines.
    * Use Curl Noise (with FastNoiseLite or SphericalWaveletNoise as potentials) to generate wind fields that influence climate.
    * Use Gabor noise for anisotropic details on rock formations.
    * Employ Domain Warping (from FastNoiseLite) to add complexity to almost any other noise pattern.
* **Performance:** Noise generation, especially with many octaves or in 3D/4D, can be computationally intensive. Profile your use cases and consider optimizations where necessary. FastNoiseLite is designed for performance.
* **Dimensionality:** Be mindful of the dimensionality of the noise required for different effects (e.g., 2D for heightmaps, 3D for volumetric textures or potential fields, 4D for time-animated 3D noise).


### 7. Poisson Disk Sampling (poisson.go)

This component implements Poisson disk sampling for generating well-distributed point sets with minimum distance constraints.

**Purpose & Key Features:**

* **Uniform Distribution:** Ensures minimum distance between samples while maximizing density
* **2D and 3D Support:** Both planar and spherical surface sampling
* **Noise Generation:** Converts sample distributions into continuous noise fields
* **Mitchell's Algorithm:** Uses fast grid-based approach for efficient generation

**Interfacing:**

* **2D Sampling:**
    * NewPoissonDiskSampler2D(seed int64, minDistance float64, domain [4]float64) *PoissonDiskSampler2D
    * sampler.Generate() []PoissonSample2D
    * NewPoissonDiskNoise2D(sampler, falloffType string, maxInfluence float64) *PoissonDiskNoise2D
* **Spherical Sampling:**
    * NewSphericalPoissonDiskSampler(seed int64, minAngularDistance float64) *SphericalPoissonDiskSampler
    * sampler.Generate() []SphericalPoissonSample
    * NewSphericalPoissonDiskNoise(sampler, falloffType string, maxInfluence float64) *SphericalPoissonDiskNoise

**World Generation Context:**

* Generating biome distribution points with natural spacing
* Placing resources, settlements, or geological features
* Creating territorial boundaries for agent-based systems
* Ecosystem modeling with competition constraints

### 8. Multifractal Noise (multifractal.go)

This component implements multifractal noise algorithms that vary roughness characteristics across space.

**Purpose & Key Features:**

* **Variable Roughness:** Unlike standard fractals, roughness changes spatially
* **Terrain Types:** Hetero, Hybrid, Ridged, and Varying Lacunarity algorithms
* **Realistic Geology:** Better models natural terrain variation
* **Hurst Exponent:** Controls roughness variation intensity

**Interfacing:**

* **Initialization:**
    * NewMultifractalNoiseBuilder(baseNoise ScalarField3D) *MultifractalNoiseBuilder
    * builder.SetType(MultifractalHeteroTerrain).SetH(0.5).Build() *MultifractalNoise3D
* **Types Available:**
    * MultifractalHeteroTerrain: Varying roughness terrain
    * MultifractalHybridTerrain: Combines additive and multiplicative fractals
    * MultifractalRidgedTerrain: Sharp ridges with varying characteristics
    * MultifractalVaryingLacunarity: Spatially varying frequency scaling

**World Generation Context:**

* Creating realistic mountain ranges with varying roughness
* Modeling geological diversity across continents
* Generating complex coastlines and terrain transitions
* Climate-influenced terrain characteristics

### 9. Turbulence Noise (turbulence.go)

This component implements various turbulence algorithms for chaotic, flowing patterns.

**Purpose & Key Features:**

* **Fluid Dynamics:** Models turbulent flow and atmospheric patterns
* **Multiple Algorithms:** Basic, Ridged, Billow, Swiss, Jordan, and Vortex turbulence
* **Domain Warping:** Adds complexity through coordinate distortion
* **Derivative Control:** Advanced algorithms use gradient information

**Interfacing:**

* **Initialization:**
    * NewTurbulenceNoiseBuilder(baseNoise ScalarField3D) *TurbulenceNoiseBuilder
    * builder.SetType(TurbulenceSwiss).SetRoughness(0.5).Build() *TurbulenceNoise3D
* **Specialized Generators:**
    * NewVortexTurbulence(baseNoise, centers, radii, strengths) *VortexTurbulence

**World Generation Context:**

* Atmospheric turbulence for weather systems
* Ocean current patterns and wave generation
* Erosion modeling with chaotic flow patterns
* Cloud and smoke effects for visual realism

### 10. Spectral Synthesis (spectral.go)

This component implements frequency-domain noise generation using physically-based spectra.

**Purpose & Key Features:**

* **Physically-Based:** Uses real-world energy spectra (ocean waves, atmospheric turbulence)
* **Frequency Control:** Direct manipulation of spectral content
* **Time Evolution:** Proper wave propagation and animation
* **Multiple Spectra:** Ocean waves, atmospheric, terrain, turbulent, and periodic

**Interfacing:**

* **Initialization:**
    * NewSpectralSynthesisBuilder().SetType(SpectralOceanWaves).Build() *SpectralSynthesis
    * synthesis.Generate() [][]float64
    * synthesis.UpdateTime(deltaTime) // For animation
* **Spectral Types:**
    * SpectralOceanWaves: Pierson-Moskowitz ocean wave spectrum
    * SpectralAtmospheric: Kolmogorov turbulence spectrum
    * SpectralTerrain: Power-law terrain spectra

**World Generation Context:**

* Realistic ocean wave simulation for coastal environments
* Atmospheric pressure systems and weather patterns
* Large-scale terrain with proper frequency content
* Time-evolving phenomena with correct physics

### 11. Reaction-Diffusion (reaction_diffusion.go)

This component implements reaction-diffusion systems for self-organizing pattern formation.

**Purpose & Key Features:**

* **Self-Organization:** Patterns emerge from simple rules without external input
* **Multiple Systems:** Gray-Scott, FitzHugh-Nagumo, Brusselator, Turing, Competition
* **Biological Realism:** Models real chemical and biological processes
* **Pattern Control:** Seed points and parameters control pattern characteristics

**Interfacing:**

* **Initialization:**
    * NewReactionDiffusionBuilder().SetType(GrayScott).Build() *ReactionDiffusionSystem
    * system.Simulate() // Run the simulation
    * NewReactionDiffusionNoise(settings, useSpeciesB) *ReactionDiffusionNoise
* **System Types:**
    * GrayScott: Classic spots and stripes patterns
    * Turing: Activator-inhibitor pattern formation
    * Competition: Species competition and territorial patterns

**World Generation Context:**

* Biome boundary formation through competition
* Vegetation patterns and ecological territories
* Mineral vein formation and geological patterns
* Animal territory and migration corridor modeling

## Advanced Usage Examples

### Combining Multiple Noise Types

```go
// Create a complex terrain combining multiple noise types
fnlState := procnoise.New[float32]()
fnlState.NoiseType(procnoise.OpenSimplex2)
fnlState.Frequency = 0.02

// Base terrain with multifractal noise
baseTerrain := procnoise.NewMultifractalNoiseBuilder(
    procnoise.NewFastNoiseLiteScalarField(fnlState),
).SetType(procnoise.MultifractalHeteroTerrain).
  SetH(0.7).
  Build()

// Add turbulent erosion patterns
erosionNoise := procnoise.NewTurbulenceNoiseBuilder(baseTerrain).
    SetType(procnoise.TurbulenceSwiss).
    SetWarp(0.15).
    Build()

// Distribute biomes with Poisson disk sampling
biomeSeeds := procnoise.NewSphericalPoissonDiskSampler(12345, 0.1)
biomeSeeds.Generate()
```

### Ocean Wave Simulation

```go
// Create realistic ocean waves
oceanWaves := procnoise.NewSpectralSynthesisBuilder().
    SetType(procnoise.SpectralOceanWaves).
    SetOceanParameters(15.0, 0.0, 1000.0). // 15 m/s wind
    SetResolution(256).
    Build()

// Animate waves over time
for t := 0.0; t < 100.0; t += 0.1 {
    oceanWaves.UpdateTime(0.1)
    waveField := oceanWaves.Generate()
    // Use waveField for rendering or physics
}
```

### Ecological Pattern Formation

```go
// Model competing vegetation species
vegetation := procnoise.NewReactionDiffusionBuilder().
    SetType(procnoise.Competition).
    SetDiffusionRates(0.1, 0.05). // Different dispersal rates
    SetGridSize(256).
    AddSeedPoint(0.3, 0.3, 10.0, 1.0). // Forest seed
    AddSeedPoint(0.7, 0.7, 8.0, 1.0).  // Grassland seed
    BuildNoise(false)

// Use as biome distribution noise
```

## Conclusion

The expanded procnoise module now offers a comprehensive toolkit for procedural noise generation in Go, covering everything from basic terrain generation to complex ecological modeling. The new modules fill critical gaps in world generation capabilities:

- **Poisson Disk Sampling** ensures natural feature distribution
- **Multifractal Noise** creates geologically realistic terrain
- **Turbulence Noise** models fluid dynamics and atmospheric phenomena
- **Spectral Synthesis** provides physically-based wave generation
- **Reaction-Diffusion** enables self-organizing biological patterns

By combining these tools strategically, developers can create rich, scientifically-grounded virtual worlds with unprecedented realism and complexity.
