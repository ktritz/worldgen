package climgen

import (
	"fmt"
	"math"
	"runtime"
	"sync"
)

// =============================================================================
// TEMPERATURE GENERATION - ENERGY BALANCE ITERATION
// =============================================================================
// This file contains the core energy balance model iteration logic.
// The model iterates until radiative equilibrium is reached.

// --- Outgoing Longwave Radiation ---

// ComputeOutgoingLongwave returns the outgoing longwave radiation (OLR) for each vertex.
//
// Uses the Stefan-Boltzmann law with surface emissivity and atmospheric absorption:
//
//	OLR = emissivity * σ * T⁴ * transmissivity
//
// where:
//   - emissivity depends on surface type (land ~0.82, water ~0.96)
//   - σ = Stefan-Boltzmann constant (5.67e-8 W/m²/K⁴)
//   - T = surface temperature in Kelvin
//   - transmissivity = atmospheric transmissivity (~0.75, represents greenhouse effect)
func ComputeOutgoingLongwave(
	temperature []float64,
	elevation []float64,
	seaLevelThreshold float64,
) []float64 {
	numVertices := len(temperature)
	olr := make([]float64, numVertices)

	for i, t := range temperature {
		var emissivity float64
		if elevation[i] >= seaLevelThreshold {
			emissivity = EmissivityLand
		} else {
			emissivity = EmissivityWater
		}

		// Stefan-Boltzmann law with atmospheric absorption
		t4 := t * t * t * t
		olr[i] = emissivity * StefanBoltzmann * t4 * AtmosphericTransmissivity
	}

	return olr
}

// --- Atmospheric Heat Transport ---

// ComputeAtmosphericHeatTransport calculates the meridional heat flux due to
// atmospheric circulation. On Earth, the atmosphere transports ~5 PW of heat
// from tropics to poles, reducing the equator-pole temperature gradient.
//
// This is modeled as:
//   - Heat divergence at tropics (net cooling)
//   - Heat convergence at high latitudes (net warming)
//   - Zero net transport at ~30-40° latitude
//
// Returns heat flux in W/m² (positive = warming, negative = cooling)
func ComputeAtmosphericHeatTransport(
	vertices []Vector3D,
	maxHeatFlux float64, // Typical: 50-100 W/m² peak transport
) []float64 {
	numVertices := len(vertices)
	heatFlux := make([]float64, numVertices)

	for i := 0; i < numVertices; i++ {
		// Get latitude (Y = sin(latitude) for unit sphere)
		sinLat := vertices[i].Y
		if sinLat > 1 {
			sinLat = 1
		} else if sinLat < -1 {
			sinLat = -1
		}
		lat := math.Asin(sinLat)
		absLat := math.Abs(lat)
		latDeg := absLat * 180.0 / math.Pi

		// Heat transport profile:
		// - Tropics (0-15°): net heat loss (divergence)
		// - Subtropics (15-35°): transition zone
		// - Mid-latitudes (35-60°): net heat gain
		// - Polar (60-90°): strong heat gain (convergence)
		var flux float64
		if latDeg < 15 {
			// Tropics: cooling (heat leaves to higher latitudes)
			flux = -maxHeatFlux * 0.5 * (1 - latDeg/15.0)
		} else if latDeg < 35 {
			// Subtropics: small effect (transition)
			t := (latDeg - 15) / 20.0
			flux = maxHeatFlux * 0.2 * t
		} else if latDeg < 60 {
			// Mid-latitudes: warming
			t := (latDeg - 35) / 25.0
			flux = maxHeatFlux * (0.2 + 0.5*t)
		} else {
			// Polar: strong warming
			t := (latDeg - 60) / 30.0
			flux = maxHeatFlux * (0.7 + 0.3*t)
		}

		heatFlux[i] = flux
	}

	return heatFlux
}

// --- Heat Capacity ---

// ComputeHeatCapacity returns the effective heat capacity for each vertex.
//
// Ocean has ~42x higher thermal inertia than land, meaning it:
//   - Heats and cools more slowly
//   - Moderates temperature extremes
//   - Stores more energy
func ComputeHeatCapacity(
	elevation []float64,
	seaLevelThreshold float64,
) []float64 {
	numVertices := len(elevation)
	cp := make([]float64, numVertices)

	for i, elev := range elevation {
		if elev >= seaLevelThreshold {
			cp[i] = HeatCapacityLand
		} else {
			cp[i] = HeatCapacityWater
		}
	}

	return cp
}

// --- Energy Balance Iteration ---

// IterateEnergyBalance performs one iteration of the energy balance model.
//
// The temperature update follows:
//
//	dT = dt * (ASR - OLR) / Cp + dT_transport
//
// where:
//   - ASR = absorbed solar radiation
//   - OLR = outgoing longwave radiation
//   - Cp = heat capacity
//   - dT_transport = temperature change from heat diffusion and advection
//
// Returns the updated temperature field and maximum temperature change.
func IterateEnergyBalance(
	temperature []float64,
	insolation []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	currents []Vector3D,
	currentSourceTemps []float64, // Precomputed source temps from backtracking (nil if no currents)
	settings TemperatureSettings,
) ([]float64, float64) {
	numVertices := len(temperature)
	newTemp := make([]float64, numVertices)

	dt := settings.Balance.TimeStep

	// Compute current albedo (with or without ice feedback)
	albedo := ComputeAlbedoSmooth(
		temperature, elevation, seaLevelThreshold, 3.0, // 3K transition width
	)

	// Compute absorbed solar radiation
	absorbed := ComputeAbsorbedSolar(insolation, albedo)

	// Compute outgoing longwave radiation
	olr := ComputeOutgoingLongwave(temperature, elevation, seaLevelThreshold)

	// Compute heat capacity
	cp := ComputeHeatCapacity(elevation, seaLevelThreshold)

	// Compute heat transport (diffusion + advection)
	transport := ComputeTotalHeatTransport(
		temperature, vertices, elevation, seaLevelThreshold, adj,
		wind, currents, settings.Transport,
	)

	// Compute atmospheric meridional heat transport (Hadley/Ferrel/Polar cells)
	var atmosTransport []float64
	if settings.Transport.AtmosphericHeatTransport > 0 {
		atmosTransport = ComputeAtmosphericHeatTransport(vertices, settings.Transport.AtmosphericHeatTransport)
	}

	// Compute ocean current temperature forcing using precomputed source temps
	var currentForcing []float64
	if currentSourceTemps != nil && settings.Transport.CurrentOriginForcing > 0 {
		currentForcing = make([]float64, numVertices)
		strength := settings.Transport.CurrentOriginForcing
		for i := 0; i < numVertices; i++ {
			if elevation[i] >= seaLevelThreshold || currents == nil {
				continue
			}
			current := currents[i]
			speed := math.Sqrt(current.X*current.X + current.Y*current.Y + current.Z*current.Z)
			if speed < 0.01 {
				continue
			}
			// Force toward source temperature
			tempDiff := currentSourceTemps[i] - temperature[i]
			currentForcing[i] = strength * speed * tempDiff
		}
	}

	// Update temperature at each vertex (parallelized)
	numWorkers := runtime.GOMAXPROCS(0)
	if numVertices < 1000 {
		numWorkers = 1
	}

	// Each worker tracks its local max delta
	localMaxDeltas := make([]float64, numWorkers)
	var wg sync.WaitGroup
	chunkSize := (numVertices + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > numVertices {
			end = numVertices
		}
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(workerID, start, end int) {
			defer wg.Done()
			localMax := 0.0

			for i := start; i < end; i++ {
				// Net radiative flux
				netRad := absorbed[i] - olr[i]

				// Add atmospheric heat transport (W/m²)
				if atmosTransport != nil {
					netRad += atmosTransport[i]
				}

				// Add ocean current forcing (W/m²) - pushes toward source-latitude temps
				if currentForcing != nil {
					netRad += currentForcing[i]
				}

				// Temperature change from radiation (W/m² -> K)
				dTRad := dt * netRad / cp[i]

				// Temperature change from local transport
				dTTransport := transport[i]

				// Total change
				dT := dTRad + dTTransport

				// Update temperature
				newTemp[i] = temperature[i] + dT

				// Clamp to physical range (100K to 400K)
				if newTemp[i] < 100.0 {
					newTemp[i] = 100.0
				} else if newTemp[i] > 400.0 {
					newTemp[i] = 400.0
				}

				// Track maximum change
				delta := math.Abs(dT)
				if delta > localMax {
					localMax = delta
				}
			}

			localMaxDeltas[workerID] = localMax
		}(w, start, end)
	}

	wg.Wait()

	// Find global max delta
	maxDelta := 0.0
	for _, d := range localMaxDeltas {
		if d > maxDelta {
			maxDelta = d
		}
	}

	return newTemp, maxDelta
}

// SolveEnergyBalance iterates the energy balance model until convergence.
//
// The model is considered converged when the maximum temperature change
// between iterations falls below the tolerance threshold.
//
// Returns:
//   - temperature: Final temperature field in Kelvin
//   - iterations: Number of iterations performed
//   - finalDelta: Maximum temperature change in the final iteration
//   - converged: Whether the tolerance was reached
func SolveEnergyBalance(
	insolation []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	currents []Vector3D,
	settings TemperatureSettings,
) ([]float64, int, float64, bool) {
	return SolveEnergyBalanceWithInitial(
		insolation,
		vertices,
		elevation,
		seaLevelThreshold,
		adj,
		wind,
		currents,
		nil,
		settings,
	)
}

// SolveEnergyBalanceWithInitial is like SolveEnergyBalance but starts from an
// explicit initial temperature field when provided.
func SolveEnergyBalanceWithInitial(
	insolation []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	currents []Vector3D,
	initialTemperature []float64,
	settings TemperatureSettings,
) ([]float64, int, float64, bool) {
	numVertices := len(vertices)

	// Initialize temperature field
	// Start at a reasonable Earth-like average (288K = 15°C)
	temperature := make([]float64, numVertices)
	if len(initialTemperature) == numVertices {
		copy(temperature, initialTemperature)
	} else {
		for i := range temperature {
			temperature[i] = 288.0
		}
	}

	maxIterations := settings.Balance.MaxIterations
	tolerance := settings.Balance.Tolerance
	verbose := settings.Balance.Verbose

	// Precompute current source temperatures ONCE (expensive backtracking)
	var currentSourceTemps []float64
	if currents != nil && settings.Transport.CurrentOriginForcing > 0 && settings.Transport.CurrentBacktrackDistance > 0 {
		currentSourceTemps = ComputeCurrentSourceTemperatures(
			vertices, elevation, seaLevelThreshold, adj, currents,
			settings.Transport.CurrentBacktrackDistance,
		)
	}

	var finalDelta float64
	var iterations int

	for iter := 0; iter < maxIterations; iter++ {
		temperature, finalDelta = IterateEnergyBalance(
			temperature,
			insolation,
			vertices,
			elevation,
			seaLevelThreshold,
			adj,
			wind,
			currents,
			currentSourceTemps,
			settings,
		)

		iterations = iter + 1

		// Check convergence
		if finalDelta < tolerance {
			if verbose {
				fmt.Printf("    Converged at iteration %d (max delta: %.6f K)\n",
					iterations, finalDelta)
			}
			return temperature, iterations, finalDelta, true
		}

		// Progress update
		if verbose && (iter+1)%500 == 0 {
			stats := computeTempStats(temperature)
			fmt.Printf("    Iter %d: max delta=%.4f K, range=[%.1f, %.1f] K\n",
				iter+1, finalDelta, stats.min, stats.max)
		}
	}

	if verbose {
		fmt.Printf("    Did not converge after %d iterations (max delta: %.6f K)\n",
			iterations, finalDelta)
	}

	return temperature, iterations, finalDelta, false
}

// --- Lapse Rate Correction ---

// ApplyLapseRateCorrection adjusts temperature for elevation effects.
//
// Temperature decreases with elevation at the environmental lapse rate:
//
//	T_surface = T_sea_level - LapseRate * elevation
//
// where LapseRate = 6.5°C per 1000m (0.0065 K/m).
//
// This correction is applied only to land (elevation > sea level threshold).
// Ocean temperatures are not adjusted (they're at sea level by definition).
func ApplyLapseRateCorrection(
	temperature []float64,
	elevation []float64,
	seaLevelThreshold float64,
) []float64 {
	numVertices := len(temperature)
	corrected := make([]float64, numVertices)

	for i, t := range temperature {
		if elevation[i] > seaLevelThreshold {
			// Apply lapse rate: temperature decreases with elevation
			correction := LapseRate * elevation[i]
			corrected[i] = t - correction

			// Don't let temperature go too low
			if corrected[i] < 180.0 { // -93°C, colder than Earth's coldest
				corrected[i] = 180.0
			}
		} else {
			// Ocean: no correction
			corrected[i] = t
		}
	}

	return corrected
}

// --- Helper Functions ---

type tempStats struct {
	min, max, mean float64
}

func computeTempStats(temperature []float64) tempStats {
	if len(temperature) == 0 {
		return tempStats{}
	}

	stats := tempStats{min: temperature[0], max: temperature[0]}
	sum := 0.0

	for _, t := range temperature {
		if t < stats.min {
			stats.min = t
		}
		if t > stats.max {
			stats.max = t
		}
		sum += t
	}

	stats.mean = sum / float64(len(temperature))
	return stats
}
