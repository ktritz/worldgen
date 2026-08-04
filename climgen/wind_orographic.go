package climgen

import (
	"math"
)

// =============================================================================
// OROGRAPHIC EFFECTS - MOUNTAIN BLOCKING AND DEFLECTION
// =============================================================================
// Mountains interact with wind in several ways:
//   1. Blocking: High mountains stop low-level airflow
//   2. Deflection: Wind flows around obstacles
//   3. Channeling: Speed increases through gaps (Venturi effect)
//   4. Slope effects: Uphill deceleration, downhill acceleration
//   5. Lee shadow: Reduced wind persists downwind of mountains

// Slope effect constants
const (
	earthRadiusKm          = 6371.0
	slopeDeadbandMPerKm    = 0.8
	slopeUphillResponse    = 0.022
	slopeDownhillResponse  = 0.015
	slopeUphillMaxFactor   = 0.78
	slopeDownhillMaxFactor = 1.12
)

// ApplyOrographicDeflection modifies wind field for mountain effects.
// Mountains block flow and cause deflection around obstacles.
func ApplyOrographicDeflection(
	wind []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	adj *FlatAdjacency,
	settings OrographicSettings,
) []Vector3D {
	numVertices := len(vertices)
	deflectedWind := make([]Vector3D, numVertices)
	copy(deflectedWind, wind)

	// Compute mountain mask (vertices above blocking threshold)
	isMountain := make([]bool, numVertices)
	for i := range vertices {
		isMountain[i] = elevation[i] > settings.BlockingThreshold
	}

	// For each non-mountain vertex, check if wind would hit a mountain
	for i := range vertices {
		if isMountain[i] {
			// Mountains have severely reduced surface wind (blocked)
			deflectedWind[i] = Scale(wind[i], 0.25)
			continue
		}

		windSpeed := Length(wind[i])
		if windSpeed < 1e-9 {
			continue
		}

		windDir := Scale(wind[i], 1.0/windSpeed)
		normal := vertices[i]

		// Check neighbors for mountains in the downwind direction
		var blockingForce Vector3D
		var alignedWeight float64
		var barrierWeight float64

		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= numVertices {
				continue
			}

			// Vector to neighbor in tangent plane
			diff := Sub(vertices[k], vertices[i])
			dotN := Dot(diff, normal)
			tangentDiff := Sub(diff, Scale(normal, dotN))
			tangentLen := Length(tangentDiff)

			if tangentLen < 1e-12 {
				continue
			}
			tangentDir := Scale(tangentDiff, 1.0/tangentLen)

			// Is this neighbor downwind?
			downwindness := Dot(tangentDir, windDir)
			if downwindness <= 0.15 {
				continue
			}
			alignedWeight += downwindness

			if isMountain[k] {
				// Downwind mountain - contributes to blocking
				mountainStrength := smoothRamp(
					settings.BlockingThreshold,
					settings.BlockingThreshold*2.5,
					elevation[k],
				)
				barrier := downwindness * (0.35 + 0.65*mountainStrength)
				barrierWeight += barrier

				// Push perpendicular to wind direction (around the obstacle)
				// Use cross product with normal to get perpendicular in tangent plane
				perpToWind := Cross(normal, windDir)
				perpLen := Length(perpToWind)
				if perpLen > 1e-9 {
					perpToWind = Scale(perpToWind, 1.0/perpLen)

					// Determine which way to deflect based on mountain position
					// Deflect away from the mountain
					towardMountain := Dot(tangentDir, perpToWind)
					deflectDir := perpToWind
					if towardMountain > 0 {
						deflectDir = Scale(perpToWind, -1) // Deflect the other way
					}

					blockingForce = Add(blockingForce, Scale(deflectDir, barrier))
				}
			}
		}

		terrainResponse := smoothRamp(0.18, 0.55, windSpeed)
		barrierFraction := 0.0
		if alignedWeight > 1e-9 {
			barrierFraction = Clamp(barrierWeight/alignedWeight, 0, 1)
		}
		effectiveBarrier := barrierFraction * (0.25 + 0.75*terrainResponse)
		// Terrain should bend and modestly slow low-level flow, but not collapse it
		// into numerical sinks because a few neighbors happen to be mountainous.
		channelFactor := 1.0 - 0.18*effectiveBarrier
		channelFactor += 0.08 * (settings.ChannelSpeedup - 1.0) * effectiveBarrier * (1.0 - effectiveBarrier)
		channelFactor = Clamp(channelFactor, 0.78, settings.ChannelSpeedup)

		// Apply deflection
		blockingLen := Length(blockingForce)
		if blockingLen > 1e-9 {
			blockingForce = Scale(blockingForce, 1.0/blockingLen)
			deflectStrength := settings.DeflectionStrength * (0.18 + 0.82*effectiveBarrier)

			// Blend original direction with deflected direction
			deflectedDir := Add(
				Scale(windDir, 1.0-deflectStrength),
				Scale(blockingForce, deflectStrength),
			)

			// Renormalize
			deflectedLen := Length(deflectedDir)
			if deflectedLen > 1e-9 {
				deflectedDir = Scale(deflectedDir, 1.0/deflectedLen)
			} else {
				deflectedDir = windDir
			}

			// Project back to tangent plane (ensure no radial component)
			dotN := Dot(deflectedDir, normal)
			deflectedDir = Sub(deflectedDir, Scale(normal, dotN))
			deflectedLen = Length(deflectedDir)
			if deflectedLen > 1e-9 {
				deflectedDir = Scale(deflectedDir, 1.0/deflectedLen)
			}

			deflectedWind[i] = Scale(deflectedDir, windSpeed*channelFactor)
		} else {
			// No mountain influence, just apply channeling
			deflectedWind[i] = Scale(wind[i], channelFactor)
		}
	}

	return deflectedWind
}

// ApplySlopeEffects modifies wind speed based on terrain slope in wind direction.
// Uphill wind decelerates, downhill wind accelerates (foehn/chinook effect).
func ApplySlopeEffects(
	wind []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	adj *FlatAdjacency,
) []Vector3D {
	numVertices := len(wind)
	result := make([]Vector3D, numVertices)
	copy(result, wind)

	for i := range vertices {
		windSpeed := Length(wind[i])
		if windSpeed < 1e-9 {
			continue
		}
		windDir := Scale(wind[i], 1.0/windSpeed)
		normal := vertices[i]
		var totalWeight float64
		var directionalSlope float64

		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= numVertices {
				continue
			}

			// Vector to neighbor in tangent plane
			diff := Sub(vertices[k], vertices[i])
			dotN := Dot(diff, normal)
			tangentDiff := Sub(diff, Scale(normal, dotN))
			tangentLen := Length(tangentDiff)

			if tangentLen < 1e-12 {
				continue
			}
			tangentDir := Scale(tangentDiff, 1.0/tangentLen)

			alignment := Dot(tangentDir, windDir)
			if alignment <= 0 {
				continue
			}
			angularDist := math.Acos(Clamp(Dot(vertices[i], vertices[k]), -1, 1))
			distKm := angularDist * earthRadiusKm
			if distKm < 1e-6 {
				continue
			}
			localSlope := (elevation[k] - elevation[i]) / distKm
			weight := alignment * alignment
			directionalSlope += localSlope * weight
			totalWeight += weight
		}

		if totalWeight < 1e-9 {
			continue
		}
		directionalSlope /= totalWeight
		if math.Abs(directionalSlope) <= slopeDeadbandMPerKm {
			continue
		}

		slopeFactor := 1.0
		if directionalSlope > 0 {
			effectiveSlope := directionalSlope - slopeDeadbandMPerKm
			slopeFactor = math.Max(
				slopeUphillMaxFactor,
				1.0-effectiveSlope*slopeUphillResponse,
			)
		} else {
			effectiveSlope := -directionalSlope - slopeDeadbandMPerKm
			slopeFactor = math.Min(
				slopeDownhillMaxFactor,
				1.0+effectiveSlope*slopeDownhillResponse,
			)
		}
		result[i] = Scale(wind[i], slopeFactor)
	}

	return result
}

// PropagateLeeShadow spreads reduced wind speed downwind of mountains.
// Uses iterative diffusion: cells downwind of slow-wind cells also slow down.
func PropagateLeeShadow(
	wind []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	adj *FlatAdjacency,
	iterations int,
	decayRate float64, // How much the shadow effect decays per iteration (0.7 = 30% decay)
) []Vector3D {
	numVertices := len(wind)
	result := make([]Vector3D, numVertices)
	copy(result, wind)

	// Track "shadow intensity" - how much each cell is in a wind shadow
	// 1.0 = full shadow (mountain), 0.0 = no shadow
	shadow := make([]float64, numVertices)

	// Initialize shadow from current wind speeds
	// Cells with very low wind relative to neighbors are in shadow
	for i := range vertices {
		speed := Length(wind[i])
		if speed < 0.15 { // Mountains and blocked areas
			shadow[i] = 1.0
		}
	}

	// Iterate: propagate shadow downwind.
	// Raise the per-hop decay to the physical hop length so the shadow's
	// e-folding distance stays fixed across mesh resolutions (exact no-op at
	// the L5 baseline where the scale is 1).
	effectiveDecay := math.Pow(decayRate, meshPathCostResolutionScale(len(vertices)))
	nextShadow := make([]float64, numVertices)
	for iter := 0; iter < iterations; iter++ {
		copy(nextShadow, shadow)

		for i := range vertices {
			if shadow[i] > 0.9 {
				// Already fully shadowed (mountain)
				continue
			}

			windDir := Normalize(result[i])
			if Length(result[i]) < 1e-9 {
				continue
			}

			normal := vertices[i]

			// Check upwind neighbors for shadow
			maxUpwindShadow := 0.0

			for _, k := range adj.GetNeighbors(i) {
				if k < 0 || k >= numVertices {
					continue
				}

				// Vector to neighbor
				diff := Sub(vertices[k], vertices[i])
				dotN := Dot(diff, normal)
				tangentDiff := Sub(diff, Scale(normal, dotN))
				tangentLen := Length(tangentDiff)

				if tangentLen < 1e-12 {
					continue
				}
				tangentDir := Scale(tangentDiff, 1.0/tangentLen)

				// Is this neighbor upwind? (negative dot = neighbor is behind us)
				upwindness := -Dot(tangentDir, windDir)

				if upwindness > 0.3 && shadow[k] > 0 {
					// Upwind neighbor is in shadow - we inherit some of it
					inheritedShadow := shadow[k] * upwindness * effectiveDecay

					// Shadow is weaker if we're lower than the upwind cell (descending air recovers)
					elevDiff := elevation[i] - elevation[k]
					if elevDiff < 0 {
						// We're lower - shadow persists or intensifies
						inheritedShadow *= 1.0
					} else {
						// We're higher - shadow breaks up faster
						inheritedShadow *= 0.5
					}

					if inheritedShadow > maxUpwindShadow {
						maxUpwindShadow = inheritedShadow
					}
				}
			}

			// Update shadow for this cell
			if maxUpwindShadow > nextShadow[i] {
				nextShadow[i] = maxUpwindShadow
			}
		}

		// Swap buffers
		shadow, nextShadow = nextShadow, shadow
	}

	// Apply shadow to wind speeds
	for i := range result {
		if shadow[i] > 0 {
			// Reduce wind speed based on shadow intensity
			// Shadow of 1.0 → 20% speed, shadow of 0.5 → 60% speed
			speedFactor := 1.0 - 0.8*shadow[i]
			result[i] = Scale(result[i], speedFactor)
		}
	}

	return result
}

// ComputeWindwardness returns how "windward" each vertex is relative to prevailing wind.
// Positive = facing into wind (upwind slope), Negative = facing away (downwind/lee)
// This is useful for precipitation calculations (future feature).
func ComputeWindwardness(
	wind []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	adj *FlatAdjacency,
) []float64 {
	numVertices := len(vertices)
	windwardness := make([]float64, numVertices)

	for i := range vertices {
		windDir := Normalize(wind[i])
		if Length(wind[i]) < 1e-9 {
			continue
		}

		normal := vertices[i]

		// Compute elevation gradient (slope direction)
		var gradE, gradN float64
		east, north := GetTangentVectors(vertices[i])
		count := 0

		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= numVertices {
				continue
			}

			diff := Sub(vertices[k], vertices[i])
			dotN := Dot(diff, normal)
			tangentDiff := Sub(diff, Scale(normal, dotN))

			de := Dot(tangentDiff, east)
			dn := Dot(tangentDiff, north)

			dElev := elevation[k] - elevation[i]
			gradE += dElev * de
			gradN += dElev * dn
			count++
		}

		if count > 0 {
			gradE /= float64(count)
			gradN /= float64(count)
		}

		// Slope direction (uphill direction in tangent plane)
		slopeDir := Add(Scale(east, gradE), Scale(north, gradN))
		slopeLen := Length(slopeDir)

		if slopeLen > 1e-9 {
			slopeDir = Scale(slopeDir, 1.0/slopeLen)

			// Windwardness = dot product of wind direction and uphill direction
			// Positive = wind blowing uphill (windward slope)
			// Negative = wind blowing downhill (leeward slope)
			windwardness[i] = Dot(windDir, slopeDir) * math.Min(slopeLen*1000, 1.0)
		}
	}

	return windwardness
}
