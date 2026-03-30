package climgen

import (
	"math"
)

// =============================================================================
// ATMOSPHERIC CIRCULATION - PRESSURE CELLS AND GEOSTROPHIC WIND
// =============================================================================
// This file implements the 3-cell model of atmospheric circulation:
//   - Hadley cell (0-30°): Rising at equator, sinking at subtropics
//   - Ferrel cell (30-60°): Counter-rotating, driven by Hadley and Polar
//   - Polar cell (60-90°): Rising at subpolar, sinking at poles
//
// Pressure pattern:
//   - ITCZ (0°): Low pressure (rising air)
//   - Subtropical (~30°): High pressure (sinking air, horse latitudes)
//   - Subpolar (~60°): Low pressure (polar front)
//   - Polar (~90°): High pressure (cold dense air)

// Zone transition width in degrees for smooth blending at cell boundaries.
const zoneTransitionWidth = 5.0

// ITCZ (Intertropical Convergence Zone) parameters
// The doldrums - calm zone near equator where trade winds converge
const itczWidth = 5.0     // Half-width in degrees (full zone is ±5°)
const itczMinSpeed = 0.15 // Minimum flow strength at equator (vs normal 0.5-1.0)

// smoothstep returns a smooth interpolation value between 0 and 1.
// Returns 0 when t <= 0, returns 1 when t >= 1, and smoothly interpolates between.
func smoothstep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	// Hermite interpolation: 3t² - 2t³
	return t * t * (3 - 2*t)
}

// ComputeCirculationPressure computes the idealized zonal pressure field.
// Returns pressure values (normalized to [-1, 1]) and circulation zone classification.
func ComputeCirculationPressure(
	vertices []Vector3D,
	settings CirculationSettings,
) ([]float64, []CirculationZone) {
	numVertices := len(vertices)
	pressure := make([]float64, numVertices)
	zones := make([]CirculationZone, numVertices)

	// Convert cell boundaries to radians
	hadleyLat := settings.HadleyEdgeLat * math.Pi / 180
	ferrelLat := settings.FerrelEdgeLat * math.Pi / 180

	for i, v := range vertices {
		lat := effectiveCirculationLatitude(math.Asin(v.Y), settings)
		absLat := math.Abs(lat)

		// Classify zone
		zones[i] = getCirculationZone(lat, settings)

		// Pressure profile using smooth cosine transitions
		// Creates: low at equator, high at ~30°, low at ~60°, high at poles
		//
		// Pattern: p(lat) follows cos(2*scaled_lat) but with custom zone boundaries
		// - Zone 0-30°: rises from -1 to +1 (ITCZ to subtropical high)
		// - Zone 30-60°: falls from +1 to -1 (subtropical high to subpolar low)
		// - Zone 60-90°: rises from -1 to +1 (subpolar low to polar high)

		var p float64
		if absLat < hadleyLat {
			// Hadley zone: smooth rise from -1 (equator) to +1 (30°)
			t := absLat / hadleyLat
			// Use cosine for smooth transition: cos(π - πt) = -cos(πt)
			p = -math.Cos(math.Pi * t)
		} else if absLat < ferrelLat {
			// Ferrel zone: smooth fall from +1 (30°) to -1 (60°)
			t := (absLat - hadleyLat) / (ferrelLat - hadleyLat)
			p = math.Cos(math.Pi * t)
		} else {
			// Polar zone: smooth rise from -1 (60°) to +0.6 (90°)
			// Polar high is weaker than subtropical high
			t := (absLat - ferrelLat) / (math.Pi/2 - ferrelLat)
			p = -1.0 + 1.6*t // Linear rise from -1 to +0.6
		}

		pressure[i] = p * settings.PressureStrength
	}

	return pressure, zones
}

// ComputeCellDrivenWind computes surface wind from thermal circulation cells + Coriolis.
// This models the actual surface wind pattern:
//   - Hadley cell (0-30°): Equatorward surface flow → deflected to easterlies (trade winds)
//   - Ferrel cell (30-60°): Poleward surface flow → deflected to westerlies
//   - Polar cell (60-90°): Equatorward surface flow → deflected to polar easterlies
//
// The meridional flow strength varies smoothly within each cell, and Coriolis
// deflection increases with latitude (zero at equator, max at poles).
//
// Rossby wave perturbation adds longitudinal variation to create meandering patterns,
// strongest at mid-latitudes where the jet stream meanders on Earth.
func ComputeCellDrivenWind(
	vertices []Vector3D,
	zones []CirculationZone,
	settings CirculationSettings,
) []Vector3D {
	numVertices := len(vertices)
	wind := make([]Vector3D, numVertices)

	// Convert boundaries to radians
	hadleyLat := settings.HadleyEdgeLat * math.Pi / 180
	ferrelLat := settings.FerrelEdgeLat * math.Pi / 180

	// Rossby wave parameters
	rossbyK := float64(settings.RossbyWavenumber) // wavenumber
	rossbyAmp := settings.RossbyAmplitude
	rossbyPhase := settings.RossbyPhase

	for i, v := range vertices {
		lat := effectiveCirculationLatitude(math.Asin(v.Y), settings) // radians, signed, thermal-equator shifted
		absLat := math.Abs(lat)
		hemisphere := 1.0
		if lat < 0 {
			hemisphere = -1.0
		}

		// Get longitude for Rossby wave calculation
		lon := math.Atan2(v.Z, v.X) // radians, -π to π

		east, north := GetTangentVectors(v)

		// Compute flow characteristics for each zone
		// Then blend at transition boundaries for smooth wind patterns
		absLatDeg := absLat * 180 / math.Pi
		hadleyLatDeg := settings.HadleyEdgeLat
		ferrelLatDeg := settings.FerrelEdgeLat

		// Compute normalized position within each zone (0-1)
		tHadley := absLat / hadleyLat                             // 0 at equator, 1 at 30°
		tFerrel := (absLat - hadleyLat) / (ferrelLat - hadleyLat) // 0 at 30°, 1 at 60°
		tPolar := (absLat - ferrelLat) / (math.Pi/2 - ferrelLat)  // 0 at 60°, 1 at 90°
		tFerrel = math.Max(0, math.Min(1, tFerrel))
		tPolar = math.Max(0, math.Min(1, tPolar))

		// === Hadley zone flow characteristics ===
		// Surface flow toward equator, strongest near 30°, but still significant at equator
		hadleyFlowStrength := 0.5 + 0.5*math.Min(1, tHadley) // 0.5 to 1.0
		hadleyMeridionalFlow := -1.0                         // Equatorward
		hadleyRossbyStrength := math.Min(1, tHadley) * 0.5

		// === Ferrel zone flow characteristics ===
		// Surface flow toward poles, bell-shaped (strongest at ~45°)
		ferrelFlowStrength := math.Max(0.4, math.Sin(math.Pi*tFerrel))
		ferrelMeridionalFlow := 1.0 // Poleward
		ferrelRossbyStrength := math.Sin(math.Pi * tFerrel)

		// === Polar zone flow characteristics ===
		// Surface flow toward equator, fairly uniform strength
		polarFlowStrength := 0.5 + 0.5*tPolar // 0.5 to 1.0
		polarMeridionalFlow := -1.0           // Equatorward
		polarRossbyStrength := 1.0 - tPolar*0.7

		// Compute blend weights based on proximity to zone boundaries
		// Use smoothstep for gradual transitions over ±5° bands
		var meridionalFlow, flowStrength, rossbyStrength float64

		// Transition band at 30° (Hadley/Ferrel boundary)
		hadleyFerrelBlend := smoothstep((absLatDeg - (hadleyLatDeg - zoneTransitionWidth)) / (2 * zoneTransitionWidth))

		// Transition band at 60° (Ferrel/Polar boundary)
		ferrelPolarBlend := smoothstep((absLatDeg - (ferrelLatDeg - zoneTransitionWidth)) / (2 * zoneTransitionWidth))

		if absLatDeg < hadleyLatDeg-zoneTransitionWidth {
			// Pure Hadley zone (0° to 25°)
			flowStrength = hadleyFlowStrength
			meridionalFlow = hadleyMeridionalFlow
			rossbyStrength = hadleyRossbyStrength
		} else if absLatDeg < hadleyLatDeg+zoneTransitionWidth {
			// Transition zone: Hadley → Ferrel (25° to 35°)
			flowStrength = hadleyFlowStrength*(1-hadleyFerrelBlend) + ferrelFlowStrength*hadleyFerrelBlend
			meridionalFlow = hadleyMeridionalFlow*(1-hadleyFerrelBlend) + ferrelMeridionalFlow*hadleyFerrelBlend
			rossbyStrength = hadleyRossbyStrength*(1-hadleyFerrelBlend) + ferrelRossbyStrength*hadleyFerrelBlend
		} else if absLatDeg < ferrelLatDeg-zoneTransitionWidth {
			// Pure Ferrel zone (35° to 55°)
			flowStrength = ferrelFlowStrength
			meridionalFlow = ferrelMeridionalFlow
			rossbyStrength = ferrelRossbyStrength
		} else if absLatDeg < ferrelLatDeg+zoneTransitionWidth {
			// Transition zone: Ferrel → Polar (55° to 65°)
			flowStrength = ferrelFlowStrength*(1-ferrelPolarBlend) + polarFlowStrength*ferrelPolarBlend
			meridionalFlow = ferrelMeridionalFlow*(1-ferrelPolarBlend) + polarMeridionalFlow*ferrelPolarBlend
			rossbyStrength = ferrelRossbyStrength*(1-ferrelPolarBlend) + polarRossbyStrength*ferrelPolarBlend
		} else {
			// Pure Polar zone (65° to 90°)
			flowStrength = polarFlowStrength
			meridionalFlow = polarMeridionalFlow
			rossbyStrength = polarRossbyStrength
		}

		// ITCZ calm zone (doldrums) - reduce wind near equator
		// Trade winds from both hemispheres converge here, creating calm conditions
		if absLatDeg < itczWidth {
			// Smooth transition: minimum at equator, full strength at itczWidth
			itczFactor := smoothstep(absLatDeg / itczWidth)
			// Blend from itczMinSpeed at equator to current flowStrength at boundary
			flowStrength = itczMinSpeed + (flowStrength-itczMinSpeed)*itczFactor
		}

		// Rossby wave perturbation: sinusoidal variation with longitude
		// This creates the meandering pattern in mid-latitude winds
		// The perturbation adds/subtracts from the meridional component
		rossbyPerturbation := 0.0
		if rossbyAmp > 0 && rossbyStrength > 0 {
			// Sin wave around the globe
			rossbyPerturbation = rossbyAmp * rossbyStrength * math.Sin(rossbyK*lon+rossbyPhase)
		}

		// Meridional component (north-south)
		// In NH: poleward = +north, equatorward = -north
		// In SH: poleward = -north, equatorward = +north
		meridionalDir := meridionalFlow * hemisphere

		// Apply Rossby perturbation to meridional flow
		// This makes the flow meander north-south as you go around longitude
		meridionalDir += rossbyPerturbation * hemisphere

		// Coriolis deflection: model CUMULATIVE deflection along trajectory
		// Air accumulates zonal velocity as it travels through the cell.
		//
		// Hadley cell: air moves from 30° toward equator
		//   - At 30°: just started journey, minimal cumulative deflection
		//   - At equator: full trajectory traversed, maximum cumulative deflection
		//   → deflection INCREASES as lat DECREASES (opposite of local Coriolis!)
		//
		// Ferrel cell: air moves from 30° toward 60°
		//   - At 30°: just started, minimal deflection
		//   - At 60°: full trajectory, maximum deflection
		//   → deflection INCREASES with latitude
		//
		// Polar cell: air moves from 60° toward equator (drains toward 60°)
		//   - Actually moves poleward to equatorward, similar pattern to Hadley
		//
		// We compute deflection based on normalized position within each cell.

		var deflectionFactor float64

		// Use the blended t values to compute cumulative deflection
		if absLatDeg < hadleyLatDeg-zoneTransitionWidth {
			// Pure Hadley: deflection increases toward equator (end of trajectory)
			// tHadley=0 at equator, tHadley=1 at 30°, so use (1-tHadley)
			deflectionFactor = 0.3 + 0.7*(1.0-tHadley)
		} else if absLatDeg < hadleyLatDeg+zoneTransitionWidth {
			// Transition Hadley→Ferrel
			hadleyDeflection := 0.3 + 0.7*(1.0-tHadley)
			ferrelDeflection := 0.3 + 0.5*tFerrel // Ferrel starts weak
			deflectionFactor = hadleyDeflection*(1-hadleyFerrelBlend) + ferrelDeflection*hadleyFerrelBlend
		} else if absLatDeg < ferrelLatDeg-zoneTransitionWidth {
			// Pure Ferrel: deflection increases toward 60° (end of trajectory)
			deflectionFactor = 0.3 + 0.7*tFerrel
		} else if absLatDeg < ferrelLatDeg+zoneTransitionWidth {
			// Transition Ferrel→Polar
			ferrelDeflection := 0.3 + 0.7*tFerrel
			polarDeflection := 0.85 // Polar: high deflection due to strong Coriolis
			deflectionFactor = ferrelDeflection*(1-ferrelPolarBlend) + polarDeflection*ferrelPolarBlend
		} else {
			// Pure Polar: strong Coriolis means high deflection throughout
			// Polar easterlies are quite zonal
			deflectionFactor = 0.85
		}

		// Zonal component from Coriolis deflection
		// Coriolis deflects velocity perpendicular to motion:
		//   NH: to the RIGHT → equatorward motion gains westward component (easterlies)
		//   SH: to the LEFT → equatorward motion gains westward component (easterlies)
		// The sign works out: meridionalFlow<0 (equatorward) → zonalDir<0 (westward/easterly)
		zonalDir := meridionalFlow * deflectionFactor

		// Combine components
		// As deflection increases, meridional component decreases
		// Use 0.5 factor to preserve more meridional flow for moisture transport
		// (0.85 was too aggressive - made winds too zonal, starving continental interiors)
		meridionalComponent := meridionalDir * flowStrength * (1.0 - 0.5*deflectionFactor)
		zonalComponent := zonalDir * flowStrength

		// Build wind vector in tangent plane
		wind[i] = Add(Scale(east, zonalComponent), Scale(north, meridionalComponent))
	}

	return wind
}

// ComputeGeostrophicWind computes upper-level wind from pressure gradient.
// Geostrophic balance: wind flows parallel to isobars (90° from pressure gradient).
// In NH, low pressure is to the left of wind direction.
// In SH, low pressure is to the right of wind direction.
func ComputeGeostrophicWind(
	pressure []float64,
	vertices []Vector3D,
	adj *FlatAdjacency,
	settings CirculationSettings,
) []Vector3D {
	numVertices := len(vertices)
	wind := make([]Vector3D, numVertices)

	for i := range vertices {
		// Compute pressure gradient in tangent plane using least-squares fit
		// Same pattern as currents_streamfunction.go
		normal := vertices[i]
		east, north := GetTangentVectors(vertices[i])

		var sumDpDe, sumDpDn float64
		var sumWe, sumWn float64

		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= numVertices {
				continue
			}

			// Vector to neighbor, projected onto tangent plane
			diff := Sub(vertices[k], vertices[i])
			dotN := Dot(diff, normal)
			tangentDiff := Sub(diff, Scale(normal, dotN))

			// Components in east/north directions
			de := Dot(tangentDiff, east)
			dn := Dot(tangentDiff, north)
			dist := math.Sqrt(de*de + dn*dn)

			if dist < 1e-12 {
				continue
			}

			// Pressure difference
			dp := pressure[k] - pressure[i]

			// Weighted contribution to gradient
			weight := 1.0 / dist
			sumDpDe += weight * dp * de / dist
			sumDpDn += weight * dp * dn / dist
			sumWe += weight * de * de / (dist * dist)
			sumWn += weight * dn * dn / (dist * dist)
		}

		// Gradient components
		var gradE, gradN float64
		if sumWe > 1e-12 {
			gradE = sumDpDe / sumWe
		}
		if sumWn > 1e-12 {
			gradN = sumDpDn / sumWn
		}

		// Coriolis parameter: f = 2Ω sin(lat)
		// Simplified: we use sin(lat) scaled by CoriolisScale
		lat := math.Asin(vertices[i].Y)
		sinLat := math.Sin(lat)

		// Avoid division by zero near equator
		// Near equator, geostrophic balance breaks down anyway
		if math.Abs(sinLat) < 0.1 {
			sinLat = 0.1 * math.Copysign(1, sinLat)
		}
		coriolis := 2.0 * settings.CoriolisScale * sinLat

		// Geostrophic wind: perpendicular to pressure gradient
		// v = (1/f) * k × ∇p where k is vertical unit vector
		// In tangent plane: wind_east = -(1/f) * dp/dn, wind_north = (1/f) * dp/de
		// This makes wind flow with low pressure to the left (NH) or right (SH)
		windE := -gradN / coriolis
		windN := gradE / coriolis

		wind[i] = Add(Scale(east, windE), Scale(north, windN))
	}

	return wind
}
