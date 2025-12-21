package procnoise

import (
	"math"
	"math/rand"
	"worldgen/icosphere"
)

// --- Reaction-Diffusion Implementation ---

// ReactionDiffusionType defines the type of reaction-diffusion system.
type ReactionDiffusionType int

const (
	// GrayScott classic Gray-Scott reaction-diffusion system
	GrayScott ReactionDiffusionType = iota
	// FitzHughNagumo excitable media model
	FitzHughNagumo
	// Brusselator oscillatory chemical system
	Brusselator
	// Turing pattern formation model
	Turing
	// Competition competitive exclusion model
	Competition
)

// ReactionDiffusionSettings holds configuration for reaction-diffusion systems.
type ReactionDiffusionSettings struct {
	Type        ReactionDiffusionType // Type of reaction-diffusion system
	GridSize    int                   // Size of the simulation grid
	TimeSteps   int                   // Number of simulation time steps
	DeltaTime   float64               // Time step size
	DeltaSpace  float64               // Spatial step size
	
	// Diffusion coefficients
	DiffusionA  float64 // Diffusion rate for species A
	DiffusionB  float64 // Diffusion rate for species B
	
	// Gray-Scott parameters
	FeedRate    float64 // Feed rate (f parameter)
	KillRate    float64 // Kill rate (k parameter)
	
	// FitzHugh-Nagumo parameters
	A_FHN       float64 // Excitability parameter
	B_FHN       float64 // Recovery parameter
	Tau         float64 // Time scale separation
	
	// Brusselator parameters
	A_Bruss     float64 // Reaction parameter A
	B_Bruss     float64 // Reaction parameter B
	
	// Turing parameters
	Alpha       float64 // Autocatalysis rate
	Beta        float64 // Cross-catalysis rate
	Gamma       float64 // Decay rate
	
	// Competition parameters
	GrowthRateA float64 // Growth rate of species A
	GrowthRateB float64 // Growth rate of species B
	CarryingCap float64 // Carrying capacity
	Competition float64 // Competition strength
	
	// Initial conditions
	InitialNoiseLevel float64 // Amount of initial random perturbation
	InitialA          float64 // Initial concentration of species A
	InitialB          float64 // Initial concentration of species B
	
	// Boundary conditions
	BoundaryType      string  // "periodic", "zero", "reflecting"
	
	// Pattern control
	SeedPoints        []Vec2  // Specific seed points for patterns
	SeedRadius        float64 // Radius of seed perturbations
	SeedStrength      float64 // Strength of seed perturbations
	
	// Random parameters
	Seed              int64   // Random seed
}

// ReactionDiffusionSystem simulates reaction-diffusion dynamics.
type ReactionDiffusionSystem struct {
	Settings     ReactionDiffusionSettings
	GridA        [][]float64 // Concentration grid for species A
	GridB        [][]float64 // Concentration grid for species B
	TempGridA    [][]float64 // Temporary grid for species A
	TempGridB    [][]float64 // Temporary grid for species B
	RNG          *rand.Rand
	CurrentStep  int
	Initialized  bool
}

// NewReactionDiffusionSystem creates a new reaction-diffusion system.
func NewReactionDiffusionSystem(settings ReactionDiffusionSettings) *ReactionDiffusionSystem {
	if settings.GridSize <= 0 {
		settings.GridSize = 128
	}
	if settings.TimeSteps <= 0 {
		settings.TimeSteps = 1000
	}
	if settings.DeltaTime <= 0 {
		settings.DeltaTime = 1.0
	}
	if settings.DeltaSpace <= 0 {
		settings.DeltaSpace = 1.0
	}
	if settings.BoundaryType == "" {
		settings.BoundaryType = "periodic"
	}
	
	n := settings.GridSize
	rds := &ReactionDiffusionSystem{
		Settings:    settings,
		GridA:       make([][]float64, n),
		GridB:       make([][]float64, n),
		TempGridA:   make([][]float64, n),
		TempGridB:   make([][]float64, n),
		RNG:         rand.New(rand.NewSource(settings.Seed)),
		CurrentStep: 0,
		Initialized: false,
	}
	
	// Initialize grids
	for i := 0; i < n; i++ {
		rds.GridA[i] = make([]float64, n)
		rds.GridB[i] = make([]float64, n)
		rds.TempGridA[i] = make([]float64, n)
		rds.TempGridB[i] = make([]float64, n)
	}
	
	rds.initializeSystem()
	return rds
}

// initializeSystem sets up initial conditions for the system.
func (rds *ReactionDiffusionSystem) initializeSystem() {
	n := rds.Settings.GridSize
	
	// Set base concentrations
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			rds.GridA[i][j] = rds.Settings.InitialA
			rds.GridB[i][j] = rds.Settings.InitialB
			
			// Add random noise
			if rds.Settings.InitialNoiseLevel > 0 {
				noiseA := (rds.RNG.Float64() - 0.5) * rds.Settings.InitialNoiseLevel
				noiseB := (rds.RNG.Float64() - 0.5) * rds.Settings.InitialNoiseLevel
				rds.GridA[i][j] += noiseA
				rds.GridB[i][j] += noiseB
			}
		}
	}
	
	// Apply seed points
	for _, seed := range rds.Settings.SeedPoints {
		rds.applySeedPerturbation(seed)
	}
	
	rds.Initialized = true
}

// applySeedPerturbation adds a localized perturbation at a seed point.
func (rds *ReactionDiffusionSystem) applySeedPerturbation(center Vec2) {
	n := rds.Settings.GridSize
	radius := rds.Settings.SeedRadius
	strength := rds.Settings.SeedStrength
	
	if radius <= 0 {
		radius = float64(n) * 0.05 // Default 5% of grid size
	}
	
	centerI := int(center.X * float64(n))
	centerJ := int(center.Y * float64(n))
	radiusGrid := int(radius)
	
	for di := -radiusGrid; di <= radiusGrid; di++ {
		for dj := -radiusGrid; dj <= radiusGrid; dj++ {
			i := (centerI + di + n) % n
			j := (centerJ + dj + n) % n
			
			distance := math.Sqrt(float64(di*di + dj*dj))
			if distance <= radius {
				// Gaussian perturbation
				factor := math.Exp(-distance*distance / (2*radius*radius/9))
				rds.GridB[i][j] += strength * factor
			}
		}
	}
}

// getNeighborValue returns the value at a neighboring grid point with boundary conditions.
func (rds *ReactionDiffusionSystem) getNeighborValue(grid [][]float64, i, j, di, dj int) float64 {
	n := rds.Settings.GridSize
	ni, nj := i+di, j+dj
	
	switch rds.Settings.BoundaryType {
	case "periodic":
		ni = (ni + n) % n
		nj = (nj + n) % n
		return grid[ni][nj]
		
	case "zero":
		if ni < 0 || ni >= n || nj < 0 || nj >= n {
			return 0.0
		}
		return grid[ni][nj]
		
	case "reflecting":
		if ni < 0 {
			ni = -ni
		} else if ni >= n {
			ni = 2*n - ni - 1
		}
		if nj < 0 {
			nj = -nj
		} else if nj >= n {
			nj = 2*n - nj - 1
		}
		return grid[ni][nj]
		
	default:
		return grid[i][j] // No boundary condition
	}
}

// computeLaplacian computes the discrete Laplacian using finite differences.
func (rds *ReactionDiffusionSystem) computeLaplacian(grid [][]float64, i, j int) float64 {
	center := grid[i][j]
	
	// 5-point stencil for 2D Laplacian
	left := rds.getNeighborValue(grid, i, j, -1, 0)
	right := rds.getNeighborValue(grid, i, j, 1, 0)
	up := rds.getNeighborValue(grid, i, j, 0, -1)
	down := rds.getNeighborValue(grid, i, j, 0, 1)
	
	dx2 := rds.Settings.DeltaSpace * rds.Settings.DeltaSpace
	return (left + right + up + down - 4*center) / dx2
}

// stepGrayScott performs one time step of the Gray-Scott system.
func (rds *ReactionDiffusionSystem) stepGrayScott() {
	n := rds.Settings.GridSize
	dt := rds.Settings.DeltaTime
	Da := rds.Settings.DiffusionA
	Db := rds.Settings.DiffusionB
	f := rds.Settings.FeedRate
	k := rds.Settings.KillRate
	
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			a := rds.GridA[i][j]
			b := rds.GridB[i][j]
			
			lapA := rds.computeLaplacian(rds.GridA, i, j)
			lapB := rds.computeLaplacian(rds.GridB, i, j)
			
			// Gray-Scott reaction terms
			reaction := a * b * b
			
			// Update equations
			rds.TempGridA[i][j] = a + dt*(Da*lapA - reaction + f*(1-a))
			rds.TempGridB[i][j] = b + dt*(Db*lapB + reaction - (k+f)*b)
		}
	}
	
	// Swap grids
	rds.GridA, rds.TempGridA = rds.TempGridA, rds.GridA
	rds.GridB, rds.TempGridB = rds.TempGridB, rds.GridB
}

// stepFitzHughNagumo performs one time step of the FitzHugh-Nagumo system.
func (rds *ReactionDiffusionSystem) stepFitzHughNagumo() {
	n := rds.Settings.GridSize
	dt := rds.Settings.DeltaTime
	Da := rds.Settings.DiffusionA
	Db := rds.Settings.DiffusionB
	a := rds.Settings.A_FHN
	b := rds.Settings.B_FHN
	tau := rds.Settings.Tau
	
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			u := rds.GridA[i][j] // Activator
			v := rds.GridB[i][j] // Inhibitor
			
			lapU := rds.computeLaplacian(rds.GridA, i, j)
			lapV := rds.computeLaplacian(rds.GridB, i, j)
			
			// FitzHugh-Nagumo reaction terms
			reactionU := u - u*u*u/3 - v
			reactionV := (u + a - b*v) / tau
			
			rds.TempGridA[i][j] = u + dt*(Da*lapU + reactionU)
			rds.TempGridB[i][j] = v + dt*(Db*lapV + reactionV)
		}
	}
	
	// Swap grids
	rds.GridA, rds.TempGridA = rds.TempGridA, rds.GridA
	rds.GridB, rds.TempGridB = rds.TempGridB, rds.GridB
}

// stepBrusselator performs one time step of the Brusselator system.
func (rds *ReactionDiffusionSystem) stepBrusselator() {
	n := rds.Settings.GridSize
	dt := rds.Settings.DeltaTime
	Da := rds.Settings.DiffusionA
	Db := rds.Settings.DiffusionB
	A := rds.Settings.A_Bruss
	B := rds.Settings.B_Bruss
	
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			x := rds.GridA[i][j]
			y := rds.GridB[i][j]
			
			lapX := rds.computeLaplacian(rds.GridA, i, j)
			lapY := rds.computeLaplacian(rds.GridB, i, j)
			
			// Brusselator reaction terms
			reactionX := A + x*x*y - B*x - x
			reactionY := B*x - x*x*y
			
			rds.TempGridA[i][j] = x + dt*(Da*lapX + reactionX)
			rds.TempGridB[i][j] = y + dt*(Db*lapY + reactionY)
		}
	}
	
	// Swap grids
	rds.GridA, rds.TempGridA = rds.TempGridA, rds.GridA
	rds.GridB, rds.TempGridB = rds.TempGridB, rds.GridB
}

// stepTuring performs one time step of a Turing pattern system.
func (rds *ReactionDiffusionSystem) stepTuring() {
	n := rds.Settings.GridSize
	dt := rds.Settings.DeltaTime
	Da := rds.Settings.DiffusionA
	Db := rds.Settings.DiffusionB
	alpha := rds.Settings.Alpha
	beta := rds.Settings.Beta
	gamma := rds.Settings.Gamma
	
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			u := rds.GridA[i][j] // Activator
			v := rds.GridB[i][j] // Inhibitor
			
			lapU := rds.computeLaplacian(rds.GridA, i, j)
			lapV := rds.computeLaplacian(rds.GridB, i, j)
			
			// Turing reaction terms (activator-inhibitor)
			reactionU := alpha*u - beta*u*v
			reactionV := beta*u*v - gamma*v
			
			rds.TempGridA[i][j] = u + dt*(Da*lapU + reactionU)
			rds.TempGridB[i][j] = v + dt*(Db*lapV + reactionV)
		}
	}
	
	// Swap grids
	rds.GridA, rds.TempGridA = rds.TempGridA, rds.GridA
	rds.GridB, rds.TempGridB = rds.TempGridB, rds.GridB
}

// stepCompetition performs one time step of a competition system.
func (rds *ReactionDiffusionSystem) stepCompetition() {
	n := rds.Settings.GridSize
	dt := rds.Settings.DeltaTime
	Da := rds.Settings.DiffusionA
	Db := rds.Settings.DiffusionB
	rA := rds.Settings.GrowthRateA
	rB := rds.Settings.GrowthRateB
	K := rds.Settings.CarryingCap
	c := rds.Settings.Competition
	
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			A := rds.GridA[i][j] // Species A
			B := rds.GridB[i][j] // Species B
			
			lapA := rds.computeLaplacian(rds.GridA, i, j)
			lapB := rds.computeLaplacian(rds.GridB, i, j)
			
			// Competition reaction terms (Lotka-Volterra with carrying capacity)
			reactionA := rA * A * (1 - (A + c*B)/K)
			reactionB := rB * B * (1 - (B + c*A)/K)
			
			rds.TempGridA[i][j] = math.Max(0, A + dt*(Da*lapA + reactionA))
			rds.TempGridB[i][j] = math.Max(0, B + dt*(Db*lapB + reactionB))
		}
	}
	
	// Swap grids
	rds.GridA, rds.TempGridA = rds.TempGridA, rds.GridA
	rds.GridB, rds.TempGridB = rds.TempGridB, rds.GridB
}

// Step performs one time step of the reaction-diffusion system.
func (rds *ReactionDiffusionSystem) Step() {
	if !rds.Initialized {
		rds.initializeSystem()
	}
	
	switch rds.Settings.Type {
	case GrayScott:
		rds.stepGrayScott()
	case FitzHughNagumo:
		rds.stepFitzHughNagumo()
	case Brusselator:
		rds.stepBrusselator()
	case Turing:
		rds.stepTuring()
	case Competition:
		rds.stepCompetition()
	default:
		rds.stepGrayScott()
	}
	
	rds.CurrentStep++
}

// Simulate runs the system for the specified number of time steps.
func (rds *ReactionDiffusionSystem) Simulate() {
	for step := 0; step < rds.Settings.TimeSteps; step++ {
		rds.Step()
	}
}

// GetConcentrationA returns the concentration of species A at grid position (i,j).
func (rds *ReactionDiffusionSystem) GetConcentrationA(i, j int) float64 {
	if i < 0 || i >= rds.Settings.GridSize || j < 0 || j >= rds.Settings.GridSize {
		return 0.0
	}
	return rds.GridA[i][j]
}

// GetConcentrationB returns the concentration of species B at grid position (i,j).
func (rds *ReactionDiffusionSystem) GetConcentrationB(i, j int) float64 {
	if i < 0 || i >= rds.Settings.GridSize || j < 0 || j >= rds.Settings.GridSize {
		return 0.0
	}
	return rds.GridB[i][j]
}

// GetNoise2D evaluates the reaction-diffusion field at a 2D position using interpolation.
func (rds *ReactionDiffusionSystem) GetNoise2D(x, y float64, useSpeciesB bool) float64 {
	if !rds.Initialized {
		rds.Simulate()
	}
	
	n := rds.Settings.GridSize
	
	// Map coordinates to grid space [0, n)
	fx := (x + 1.0) * 0.5 * float64(n) // Assume input in [-1, 1]
	fy := (y + 1.0) * 0.5 * float64(n)
	
	// Handle wrapping for periodic boundaries
	fx = fx - math.Floor(fx/float64(n))*float64(n)
	fy = fy - math.Floor(fy/float64(n))*float64(n)
	
	// Bilinear interpolation
	i0 := int(fx) % n
	j0 := int(fy) % n
	i1 := (i0 + 1) % n
	j1 := (j0 + 1) % n
	
	tx := fx - float64(i0)
	ty := fy - float64(j0)
	
	var v00, v10, v01, v11 float64
	if useSpeciesB {
		v00 = rds.GridB[i0][j0]
		v10 = rds.GridB[i1][j0]
		v01 = rds.GridB[i0][j1]
		v11 = rds.GridB[i1][j1]
	} else {
		v00 = rds.GridA[i0][j0]
		v10 = rds.GridA[i1][j0]
		v01 = rds.GridA[i0][j1]
		v11 = rds.GridA[i1][j1]
	}
	
	v0 := v00*(1-tx) + v10*tx
	v1 := v01*(1-tx) + v11*tx
	
	return v0*(1-ty) + v1*ty
}

// GetNoise evaluates the reaction-diffusion field at a 3D position (using X,Y coordinates).
// Implements ScalarField3D interface.
func (rds *ReactionDiffusionSystem) GetNoise(pos icosphere.Vector3D) float32 {
	return float32(rds.GetNoise2D(pos.X, pos.Y, false)) // Use species A by default
}

// --- Reaction-Diffusion Noise Generator ---

// ReactionDiffusionNoise provides a noise interface for reaction-diffusion patterns.
type ReactionDiffusionNoise struct {
	System      *ReactionDiffusionSystem
	UseSpeciesB bool    // Whether to use species B instead of species A
	Scale       float64 // Scaling factor for output values
	Offset      float64 // Offset for output values
}

// NewReactionDiffusionNoise creates a new reaction-diffusion noise generator.
func NewReactionDiffusionNoise(settings ReactionDiffusionSettings, useSpeciesB bool) *ReactionDiffusionNoise {
	return &ReactionDiffusionNoise{
		System:      NewReactionDiffusionSystem(settings),
		UseSpeciesB: useSpeciesB,
		Scale:       1.0,
		Offset:      0.0,
	}
}

// SetScale sets the scaling parameters for the output.
func (rdn *ReactionDiffusionNoise) SetScale(scale, offset float64) {
	rdn.Scale = scale
	rdn.Offset = offset
}

// GetNoise evaluates the reaction-diffusion noise at a 3D position.
// Implements ScalarField3D interface.
func (rdn *ReactionDiffusionNoise) GetNoise(pos icosphere.Vector3D) float32 {
	value := rdn.System.GetNoise2D(pos.X, pos.Y, rdn.UseSpeciesB)
	return float32(value*rdn.Scale + rdn.Offset)
}

// GetNoise2D evaluates the reaction-diffusion noise at a 2D position.
func (rdn *ReactionDiffusionNoise) GetNoise2D(x, y float64) float64 {
	value := rdn.System.GetNoise2D(x, y, rdn.UseSpeciesB)
	return value*rdn.Scale + rdn.Offset
}

// --- Reaction-Diffusion Builder ---

// ReactionDiffusionBuilder provides a fluent interface for building reaction-diffusion systems.
type ReactionDiffusionBuilder struct {
	settings ReactionDiffusionSettings
}

// NewReactionDiffusionBuilder creates a new builder for reaction-diffusion systems.
func NewReactionDiffusionBuilder() *ReactionDiffusionBuilder {
	return &ReactionDiffusionBuilder{
		settings: ReactionDiffusionSettings{
			Type:              GrayScott,
			GridSize:          128,
			TimeSteps:         1000,
			DeltaTime:         1.0,
			DeltaSpace:        1.0,
			DiffusionA:        1.0,
			DiffusionB:        0.5,
			FeedRate:          0.055,
			KillRate:          0.062,
			InitialA:          1.0,
			InitialB:          0.0,
			InitialNoiseLevel: 0.1,
			BoundaryType:      "periodic",
			SeedRadius:        5.0,
			SeedStrength:      1.0,
		},
	}
}

// SetType sets the reaction-diffusion type.
func (b *ReactionDiffusionBuilder) SetType(rdType ReactionDiffusionType) *ReactionDiffusionBuilder {
	b.settings.Type = rdType
	return b
}

// SetGridSize sets the simulation grid size.
func (b *ReactionDiffusionBuilder) SetGridSize(size int) *ReactionDiffusionBuilder {
	b.settings.GridSize = size
	return b
}

// SetTimeParameters sets time-related parameters.
func (b *ReactionDiffusionBuilder) SetTimeParameters(steps int, deltaTime float64) *ReactionDiffusionBuilder {
	b.settings.TimeSteps = steps
	b.settings.DeltaTime = deltaTime
	return b
}

// SetDiffusionRates sets the diffusion coefficients.
func (b *ReactionDiffusionBuilder) SetDiffusionRates(diffusionA, diffusionB float64) *ReactionDiffusionBuilder {
	b.settings.DiffusionA = diffusionA
	b.settings.DiffusionB = diffusionB
	return b
}

// SetGrayScottParameters sets parameters for Gray-Scott system.
func (b *ReactionDiffusionBuilder) SetGrayScottParameters(feedRate, killRate float64) *ReactionDiffusionBuilder {
	b.settings.FeedRate = feedRate
	b.settings.KillRate = killRate
	return b
}

// SetInitialConditions sets the initial concentration values.
func (b *ReactionDiffusionBuilder) SetInitialConditions(initialA, initialB, noiseLevel float64) *ReactionDiffusionBuilder {
	b.settings.InitialA = initialA
	b.settings.InitialB = initialB
	b.settings.InitialNoiseLevel = noiseLevel
	return b
}

// SetBoundaryType sets the boundary condition type.
func (b *ReactionDiffusionBuilder) SetBoundaryType(boundaryType string) *ReactionDiffusionBuilder {
	b.settings.BoundaryType = boundaryType
	return b
}

// AddSeedPoint adds a seed point for pattern initiation.
func (b *ReactionDiffusionBuilder) AddSeedPoint(x, y, radius, strength float64) *ReactionDiffusionBuilder {
	b.settings.SeedPoints = append(b.settings.SeedPoints, Vec2{X: x, Y: y})
	b.settings.SeedRadius = radius
	b.settings.SeedStrength = strength
	return b
}

// SetSeed sets the random seed.
func (b *ReactionDiffusionBuilder) SetSeed(seed int64) *ReactionDiffusionBuilder {
	b.settings.Seed = seed
	return b
}

// Build creates the reaction-diffusion system.
func (b *ReactionDiffusionBuilder) Build() *ReactionDiffusionSystem {
	return NewReactionDiffusionSystem(b.settings)
}

// BuildNoise creates a reaction-diffusion noise generator.
func (b *ReactionDiffusionBuilder) BuildNoise(useSpeciesB bool) *ReactionDiffusionNoise {
	return NewReactionDiffusionNoise(b.settings, useSpeciesB)
}