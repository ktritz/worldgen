package climgen

import (
	"math"
	"sort"
)

// =============================================================================
// PRECIPITATION - WIND-DRIVEN MOISTURE TRANSPORT
// =============================================================================
// Simulates precipitation using wind-driven moisture advection.
//
// Algorithm (adapted from Amit Patel's mapgen4):
//   1. Sort cells by wind direction (upwind cells processed first)
//   2. Ocean cells add moisture (evaporation, temperature-dependent)
//   3. Land cells pull moisture from upwind neighbors
//   4. Orographic effect: moisture capacity = 1 - normalized_elevation
//   5. Excess moisture falls as orographic precipitation
//   6. A fraction of remaining moisture falls as regular precipitation
//
// This creates realistic patterns:
//   - Wet windward coasts
//   - Rain shadows behind mountains
//   - Dry continental interiors
//   - Tropical rainfall bands (ITCZ)

// PrecipitationSettings controls the precipitation calculation
type PrecipitationSettings struct {
	EvaporationRate            float64   // Moisture added per ocean cell (0-1)
	OrographicStrength         float64   // How strongly elevation limits moisture (0-1)
	RainfallFraction           float64   // Fraction of moisture that falls as rain per km traveled
	MinElevation               float64   // Elevation floor for normalization (meters, e.g., -10000)
	MaxElevation               float64   // Elevation ceiling for normalization (meters, e.g., 8000)
	TemperatureEffect          float64   // How much temperature affects land moisture capacity (0-1)
	OceanEvaporationTempEffect float64   // How much ocean temperature modulates evaporation (0-1)
	LandSourceScale            float64   // Global scale for direct land moisture source / recycling
	LandRecyclingScale         float64   // Global scale for evapotranspiration-driven land recycling
	LandSourceLocalScale       []float64 // Optional per-cell land-source modulation
	LandRecyclingLocalScale    []float64 // Optional per-cell recycling modulation
	TropicalSourceLocalScale   []float64 // Optional per-cell tropical/monsoon marine-source modulation
	FrontalSourceLocalScale    []float64 // Optional per-cell frontal storm-moisture source modulation
	FrontalRetentionLocalScale []float64 // Optional per-cell frontal storm-moisture retention modulation
	FrontalTransportLocalScale []float64 // Optional per-cell broad storm-band transport modulation
	LandSurfaceStorage         []float64 // Optional per-cell land-water storage for evapotranspiration
	CondensationLocalScale     []float64 // Optional per-cell condensation modulation
	LandRetentionLocalScale    []float64 // Optional per-cell moisture-retention modulation
	PrecipitationScale         float64   // Scale factor to convert to cm/year (0 = keep normalized)
}

// DefaultPrecipitationSettings returns reasonable defaults
func DefaultPrecipitationSettings() PrecipitationSettings {
	return PrecipitationSettings{
		EvaporationRate:            1.0,    // Full moisture saturation over ocean
		OrographicStrength:         0.65,   // Moderate orographic effect (reduced from 0.8)
		RainfallFraction:           0.001,  // Fraction per km - balance coast/interior rain
		MinElevation:               -10000, // Ocean floor
		MaxElevation:               6000,   // High mountains
		TemperatureEffect:          0.3,    // Reduced temp effect to ensure consistent moisture
		OceanEvaporationTempEffect: 0.18,   // Let warm/cold source waters modestly affect annual moisture supply
		LandSourceScale:            1.0,
		LandRecyclingScale:         1.0,
		PrecipitationScale:         2400.0, // Scale to cm/year (target 100 cm avg with 20cm baseline)
	}
}

// PrecipitationResult holds computed precipitation for each cell
type PrecipitationResult struct {
	// Precipitation is relative rainfall amount (0-1+ scale)
	// Higher = more rainfall, can exceed 1 in very wet areas
	Precipitation []float64

	// Moisture is the atmospheric moisture at each cell after transport
	Moisture        []float64
	MarineMoisture  []float64
	LandMoisture    []float64
	FrontalMoisture []float64

	// Rainfall and Snowfall partition total precipitation into liquid and
	// solid-water equivalents using local temperature.
	Rainfall             []float64
	Snowfall             []float64
	MarinePrecipitation  []float64
	LandPrecipitation    []float64
	FrontalPrecipitation []float64
	Debug                *PrecipitationDebugFields
}

// PrecipitationDebugFields carries cell-level internal solver state so bad
// climates can be explained without inferring causes from the final rain map.
type PrecipitationDebugFields struct {
	OceanFetch            []float64
	CoastalOnshore        []float64
	EffectiveFetch        []float64
	EffectiveOnshore      []float64
	FootprintOceanSupport []float64
	NeighborOceanFraction []float64
	MaritimeSignal        []float64
	MaritimeGeomSupport   []float64
	OceanAtmosphere       []float64
	OceanDownwindLand     []float64
	MarineEntryScale      []float64
	MarineDonor           []float64
	MarineDonorStrength   []float64
	MarineDonorOutgoing   []float64
	MarineDonorOceanAtm   []float64
	MarineDonorDownwind   []float64
	MarineRootSource      []float64
	MarineRootStrength    []float64
	MarineRootOceanAtm    []float64
	MarineRootDownwind    []float64
	MarineRootOceanSource []float64
	MarineRootRetention   []float64
	MarineRootPathSteps   []float64
	UpwindParent          []float64
	UpwindParentStrength  []float64
	LandTravel            []float64
	LandInterior          []float64
	OrographicLift        []float64
	OrographicLocalRise   []float64
	OrographicFootprint   []float64
	OrographicBarrier     []float64
	OrographicWindFactor  []float64
	Convergence           []float64
	MoistureCapacity      []float64
	LandSource            []float64
	TropicalSource        []float64
	FrontalSource         []float64
	MarineIncoming        []float64
	LandIncoming          []float64
	FrontalIncoming       []float64
	MarineToLand          []float64
	MarineToFrontal       []float64
	CondensedTotal        []float64
	CondensedBase         []float64
	CondensedSupersat     []float64
	CondensedSupersatSupport []float64
	CondensedTropicalCoast  []float64
	CondensedCoastalPenalty []float64
	CondensedAscent       []float64
	CondensedConvective   []float64
	CondensedMixing       []float64
	CondensedEffCapacity  []float64
	CondensedSupersatHum  []float64
	RetainedHumidity      []float64
	CondensationScale     []float64
	LandRetentionScale    []float64
	FrontalSourceScale    []float64
	FrontalRetentionScale []float64
	TropicalSourceScale   []float64
}

// ComputePrecipitation calculates precipitation using wind-driven moisture transport.
//
// Parameters:
//   - vertices: cell center positions on unit sphere
//   - elevation: elevation of each cell in meters
//   - seaLevel: threshold for ocean (typically 0)
//   - adj: cell adjacency structure
//   - wind: surface wind vectors for each cell
//   - temperature: temperature in Kelvin (for evaporation rate)
//   - settings: algorithm parameters
func ComputePrecipitation(
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	temperature []float64,
	settings PrecipitationSettings,
) *PrecipitationResult {
	return computePrecipitationBudget(vertices, elevation, seaLevel, adj, wind, temperature, settings)
}

func seasonalOceanEvaporationFactor(tempK float64, strength float64) float64 {
	tempC := tempK - 273.15
	// Use a mild SST-based modulation around a temperate-ocean reference so
	// seasonal runs can distinguish cold and warm source regions without
	// collapsing the annual moisture budget.
	anomaly := Clamp((tempC-15.0)/25.0, -0.8, 0.7)
	factor := 1.0 + strength*anomaly
	return Clamp(factor, 0.45, 1.45)
}

func partitionPrecipitationPhase(
	result *PrecipitationResult,
	elevation []float64,
	seaLevel float64,
	temperature []float64,
) {
	for i, total := range result.Precipitation {
		if i >= len(elevation) || elevation[i] < seaLevel {
			continue
		}
		snowFrac := precipitationSnowFraction(temperature, i)
		result.Snowfall[i] = total * snowFrac
		result.Rainfall[i] = total - result.Snowfall[i]
	}
}

func precipitationSnowFraction(temperature []float64, idx int) float64 {
	if idx < 0 || idx >= len(temperature) {
		return 0
	}
	tempC := temperature[idx] - 273.15
	switch {
	case tempC <= -3.0:
		return 1.0
	case tempC >= 3.0:
		return 0.0
	default:
		return 1.0 - smoothRamp(-3.0, 3.0, tempC)
	}
}

// computeAverageWind computes the mean wind direction
func computeAverageWind(wind []Vector3D) Vector3D {
	var sum Vector3D
	for _, w := range wind {
		sum.X += w.X
		sum.Y += w.Y
		sum.Z += w.Z
	}
	// Normalize
	mag := math.Sqrt(sum.X*sum.X + sum.Y*sum.Y + sum.Z*sum.Z)
	if mag < 0.001 {
		return Vector3D{X: 1, Y: 0, Z: 0}
	}
	return Vector3D{X: sum.X / mag, Y: sum.Y / mag, Z: sum.Z / mag}
}

// sortCellsByWindInPlace sorts cell indices in-place by wind direction (upwind first)
func sortCellsByWindInPlace(cells []int, vertices []Vector3D, windDir Vector3D) {
	windDot := make([]float64, len(vertices))
	for i, v := range vertices {
		windDot[i] = v.X*windDir.X + v.Y*windDir.Y + v.Z*windDir.Z
	}

	// Sort by wind dot (upwind first = most negative first)
	sort.Slice(cells, func(a, b int) bool {
		return windDot[cells[a]] < windDot[cells[b]]
	})
}

// pullMoistureFromUpwind computes moisture pulled from upwind neighbors
func pullMoistureFromUpwind(
	i int,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	moisture []float64,
) float64 {
	// Get local wind at this cell
	windVec := wind[i]
	windSpeed := math.Sqrt(windVec.X*windVec.X + windVec.Y*windVec.Y + windVec.Z*windVec.Z)

	if windSpeed < 0.01 {
		// No wind: average of all neighbors
		var sum float64
		count := 0
		for _, k := range adj.GetNeighbors(i) {
			if k >= 0 && k < len(moisture) {
				sum += moisture[k]
				count++
			}
		}
		if count > 0 {
			return sum / float64(count)
		}
		return 0
	}

	// Normalize wind direction
	windDir := Vector3D{
		X: windVec.X / windSpeed,
		Y: windVec.Y / windSpeed,
		Z: windVec.Z / windSpeed,
	}

	// Pull moisture from upwind neighbors (weighted by alignment)
	var moistureSum float64
	var weightSum float64

	for _, k := range adj.GetNeighbors(i) {
		if k < 0 || k >= len(moisture) {
			continue
		}

		// Direction FROM neighbor TO this cell
		fromNeighbor := Sub(vertices[i], vertices[k])
		fromNeighbor = Normalize(fromNeighbor)

		// Upwind alignment: positive if neighbor is upwind (wind comes from there)
		upwind := Dot(windDir, fromNeighbor)

		if upwind > 0 {
			// Weight by upwind alignment and wind speed
			weight := upwind * windSpeed
			moistureSum += moisture[k] * weight
			weightSum += weight
		}
	}

	if weightSum > 0 {
		return moistureSum / weightSum
	}
	return 0
}

// GetPrecipitationStats returns statistics about precipitation
func GetPrecipitationStats(result *PrecipitationResult, elevation []float64, seaLevel float64) (
	landCells int, avgPrecip, maxPrecip float64,
) {
	for i, p := range result.Precipitation {
		if elevation[i] >= seaLevel {
			landCells++
			avgPrecip += p
			if p > maxPrecip {
				maxPrecip = p
			}
		}
	}
	if landCells > 0 {
		avgPrecip /= float64(landCells)
	}
	return
}
