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
	EvaporationRate    float64 // Moisture added per ocean cell (0-1)
	OrographicStrength float64 // How strongly elevation limits moisture (0-1)
	RainfallFraction   float64 // Fraction of moisture that falls as rain per km traveled
	MinElevation       float64 // Elevation floor for normalization (meters, e.g., -10000)
	MaxElevation       float64 // Elevation ceiling for normalization (meters, e.g., 8000)
	TemperatureEffect  float64 // How much temperature affects evaporation (0-1)
	PrecipitationScale float64 // Scale factor to convert to cm/year (0 = keep normalized)
}

// DefaultPrecipitationSettings returns reasonable defaults
func DefaultPrecipitationSettings() PrecipitationSettings {
	return PrecipitationSettings{
		EvaporationRate:    1.0,     // Full moisture saturation over ocean
		OrographicStrength: 0.65,    // Moderate orographic effect (reduced from 0.8)
		RainfallFraction:   0.001,   // Fraction per km - balance coast/interior rain
		MinElevation:       -10000,  // Ocean floor
		MaxElevation:       6000,    // High mountains
		TemperatureEffect:  0.3,     // Reduced temp effect to ensure consistent moisture
		PrecipitationScale: 2400.0,  // Scale to cm/year (target 100 cm avg with 20cm baseline)
	}
}

// PrecipitationResult holds computed precipitation for each cell
type PrecipitationResult struct {
	// Precipitation is relative rainfall amount (0-1+ scale)
	// Higher = more rainfall, can exceed 1 in very wet areas
	Precipitation []float64

	// Moisture is the atmospheric moisture at each cell after transport
	Moisture []float64
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
	n := len(vertices)
	result := &PrecipitationResult{
		Precipitation: make([]float64, n),
		Moisture:      make([]float64, n),
	}

	if wind == nil {
		return result
	}

	// Compute average cell size for resolution-independent calculations
	// Earth radius = 6371 km, sphere surface area = 4*pi*r^2
	earthRadius := 6371.0
	avgCellSizeKm := earthRadius * math.Sqrt(4*math.Pi/float64(n))

	// Scale rainfall fraction by cell size (settings.RainfallFraction is per-km)
	rainfallFractionPerCell := settings.RainfallFraction * avgCellSizeKm

	// Scale iterations by mesh density (more cells need more iterations for same distance)
	// Base: 20 iterations at level 6 (~70km cells), scale proportionally
	baseCellSizeKm := 70.0
	maxIterations := int(20.0 * baseCellSizeKm / avgCellSizeKm)
	if maxIterations < 20 {
		maxIterations = 20
	}
	if maxIterations > 100 {
		maxIterations = 100
	}

	// Normalize elevation to 0-1 range for orographic calculation
	elevRange := settings.MaxElevation - settings.MinElevation
	if elevRange < 1 {
		elevRange = 1
	}

	// Identify ocean and land cells
	isOcean := make([]bool, n)
	for i := 0; i < n; i++ {
		isOcean[i] = elevation[i] < seaLevel
	}

	// Compute moisture capacity based on temperature (Clausius-Clapeyron)
	// Warm air holds much more moisture than cold air
	// Note: Real CC doubles every ~10°C, but for climate modeling we use a
	// softer effect (every 15°C) because:
	// 1. Air masses don't instantly equilibrate to local temperature
	// 2. Precipitation from temperature change takes time
	// 3. Weather systems transport moisture further before dumping it
	moistureCap := make([]float64, n)
	for i := 0; i < n; i++ {
		if temperature != nil {
			// Softened Clausius-Clapeyron: capacity doubles every ~15°C
			// Normalize: 1.0 at 30°C, decreasing exponentially with cold
			tempC := temperature[i] - 273.15
			// Cap ranges from ~0.1 at -30°C to 1.0 at 30°C
			moistureCap[i] = math.Pow(2, (tempC-30)/15.0)
			moistureCap[i] = math.Max(0.1, math.Min(1.0, moistureCap[i]))
		} else {
			moistureCap[i] = 1.0
		}
	}

	// Pass 1: Set ocean moisture based on evaporation
	// Key insight: Ocean evaporation should NOT be limited by air temperature above it.
	// The ocean surface evaporates based on water temperature (moderated by currents),
	// and the moisture then moves with the air mass. Temperature limiting happens
	// when the air reaches land and cools - that's when precipitation occurs.
	// Use full evaporation rate for all oceans.
	for i := 0; i < n; i++ {
		if isOcean[i] {
			evap := settings.EvaporationRate // Full evaporation regardless of temp
			result.Moisture[i] = evap
		}
	}

	// Pass 2+: Iterate moisture transport over land until stable
	// This ensures moisture propagates inland regardless of processing order
	//
	// KEY INSIGHT: We track two things separately:
	// 1. "moisture" - the actual moisture content (capped at capacity)
	// 2. "moistureFlux" - total moisture that PASSED THROUGH (for precipitation)
	//
	// Cold regions should get precipitation when warm moist air arrives,
	// even if local capacity is low. The excess precipitates.
	moistureFlux := make([]float64, n) // Track total incoming moisture

	// maxIterations was computed above based on cell size
	for iter := 0; iter < maxIterations; iter++ {
		maxChange := 0.0

		for i := 0; i < n; i++ {
			if isOcean[i] {
				continue
			}

			// Pull moisture from ALL neighbors (weighted by wind alignment)
			incoming := pullMoistureFromUpwind(i, vertices, adj, wind, result.Moisture)

			// Track cumulative flux (first iteration only to avoid double counting)
			if iter == 0 {
				moistureFlux[i] = incoming
			} else {
				// Update flux with new incoming (weighted average with previous)
				moistureFlux[i] = math.Max(moistureFlux[i], incoming)
			}

			// Temperature capacity limit (cold air holds less moisture)
			tempCap := moistureCap[i]

			// Orographic effect (high elevation also reduces capacity)
			normElev := (elevation[i] - settings.MinElevation) / elevRange
			normElev = math.Max(0, math.Min(1, normElev))
			elevCap := 1.0 - settings.OrographicStrength*normElev
			elevCap = math.Max(0.1, elevCap)

			// Combined capacity: minimum of temperature and elevation limits
			combinedCap := math.Min(tempCap, elevCap)

			// Cap moisture at capacity (what remains in the air)
			moisture := incoming
			if moisture > combinedCap {
				moisture = combinedCap
			}

			// Track change for convergence
			change := math.Abs(moisture - result.Moisture[i])
			if change > maxChange {
				maxChange = change
			}

			result.Moisture[i] = moisture
		}

		// Check convergence
		if maxChange < 0.001 {
			break
		}
	}

	// Convert flux to precipitation: excess over capacity precipitates
	for i := 0; i < n; i++ {
		if isOcean[i] {
			continue
		}
		tempCap := moistureCap[i]
		normElev := (elevation[i] - settings.MinElevation) / elevRange
		normElev = math.Max(0, math.Min(1, normElev))
		elevCap := 1.0 - settings.OrographicStrength*normElev
		elevCap = math.Max(0.1, elevCap)
		combinedCap := math.Min(tempCap, elevCap)

		// Precipitation = excess flux over capacity
		if moistureFlux[i] > combinedCap {
			result.Precipitation[i] += (moistureFlux[i] - combinedCap) * 0.5
		}
	}

	// Final pass: Compute precipitation from final moisture values
	for i := 0; i < n; i++ {
		if isOcean[i] {
			continue
		}

		moisture := result.Moisture[i]

		// Temperature capacity (cold air dumps moisture)
		tempCap := moistureCap[i]

		// Orographic capacity (high elevation dumps moisture)
		normElev := (elevation[i] - settings.MinElevation) / elevRange
		normElev = math.Max(0, math.Min(1, normElev))
		elevCap := 1.0 - settings.OrographicStrength*normElev
		elevCap = math.Max(0.1, elevCap)

		// Combined capacity
		combinedCap := math.Min(tempCap, elevCap)

		// Pull fresh moisture to see if we exceed capacity
		// Excess moisture precipitates (cold front / orographic effect)
		freshMoisture := pullMoistureFromUpwind(i, vertices, adj, wind, result.Moisture)
		if freshMoisture > combinedCap {
			result.Precipitation[i] += freshMoisture - combinedCap
		}

		// Regular precipitation: fraction of moisture falls as rain
		// Uses resolution-scaled fraction (rainfallFractionPerCell)
		rain := moisture * rainfallFractionPerCell
		result.Precipitation[i] += rain

		// ITCZ boost: enhance precipitation in tropics (0-15° latitude)
		// This simulates the Intertropical Convergence Zone where trade winds meet
		// Earth's ITCZ produces 200-400+ cm/yr in rainforests
		lat := math.Asin(vertices[i].Y) * 180.0 / math.Pi
		absLat := math.Abs(lat)
		if absLat < 15 {
			// Boost peaks at equator (3x), tapers to 1x at 15°
			itczBoost := 1.0 + 2.0*(1.0-absLat/15.0)
			result.Precipitation[i] *= itczBoost
		}

		// Subtropical dry zone (15-35°): Hadley cell descent suppresses rain
		if absLat >= 15 && absLat < 35 {
			// Minimum suppression at 25° (desert belt)
			// 0.78 gives ~20-25% desert coverage (Earth-like)
			distFrom25 := math.Abs(absLat - 25)
			suppression := 0.78 + 0.22*(distFrom25/10.0) // 0.78 at 25°, 1.0 at 15° and 35°
			suppression = math.Min(1.0, suppression)
			result.Precipitation[i] *= suppression
		}

		// Apply scale to convert to cm/year
		if settings.PrecipitationScale > 0 {
			result.Precipitation[i] *= settings.PrecipitationScale
		}

		// Add baseline precipitation (even driest areas get some moisture)
		// This creates realistic semi-arid transition zones
		result.Precipitation[i] += 19.0
	}

	return result
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
