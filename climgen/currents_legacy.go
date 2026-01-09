package climgen

import (
	"fmt"
	"math"
)

// =============================================================================
// BASIN-BASED CURRENT GENERATION (LEGACY)
// =============================================================================
// This approach uses detected ocean basins to create gyre patterns.
// Kept for future use (basin naming, regional climate properties, etc.)
// The current preferred approach is streamfunction-based (see streamfunction.go)

// GenerateCurrentsFromBasins creates ocean current vectors based on detected basins.
// Uses gyre rotation forcing: creates circular flow patterns around basin centroids,
// with opposite rotation directions for northern vs southern hemispheres.
// All water vertices receive currents based on nearest basin influence, allowing
// adjacent gyres to coalesce into larger circulation patterns.
// coastLandDirs provides coast-parallel constraint (can be nil to skip).
func GenerateCurrentsFromBasins(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	basinAssignments []int,
	basins []Basin,
	settings CurrentSettings,
	coastLandDirs []Vector3D,
) []Vector3D {
	numVertices := len(vertices)
	currents := make([]Vector3D, numVertices)

	if len(basins) == 0 {
		return currents
	}

	if settings.Verbose {
		fmt.Println("=== Current Generation ===")
		fmt.Printf("  Applying gyre rotation forcing to %d basins...\n", len(basins))
	}

	// Apply gyre rotation forcing to EVERY water vertex
	// Vertices in basins get full gyre forcing
	// Vertices outside basins get influence from nearest basin with distance falloff
	for i := 0; i < numVertices; i++ {
		if elevation[i] >= seaLevelThreshold {
			continue // Skip land
		}

		vertex := vertices[i]
		basinID := basinAssignments[i]

		var current Vector3D

		if basinID >= 0 && basinID < len(basins) {
			// Vertex is in a basin - apply direct gyre forcing
			current = computeGyreForcing(vertex, basins[basinID], settings.TargetEdgeSpeed)
		} else {
			// Vertex is outside basins - blend influence from nearby basins
			current = computeBlendedGyreForcing(vertex, basins, settings.TargetEdgeSpeed)
		}

		// Add latitude-based wind-driven background current
		windCurrent := computeWindDrivenCurrent(vertex, settings.TargetEdgeSpeed*0.3)
		result := Add(current, windCurrent)

		// Apply coast-parallel constraint if near coastline
		if coastLandDirs != nil && i < len(coastLandDirs) {
			landDir := coastLandDirs[i]
			landDirLenSq := LengthSq(landDir)
			if landDirLenSq > 1e-12 {
				// Remove component perpendicular to coast
				dotPerp := Dot(result, landDir)
				result = Sub(result, Scale(landDir, dotPerp))
			}
		}

		currents[i] = result
	}

	if settings.Verbose {
		nonZeroCount := 0
		maxSpeed := 0.0
		for _, c := range currents {
			speed := Length(c)
			if speed > 1e-9 {
				nonZeroCount++
				if speed > maxSpeed {
					maxSpeed = speed
				}
			}
		}
		fmt.Printf("  Assigned currents to %d vertices, max speed: %.4f\n", nonZeroCount, maxSpeed)
	}

	return currents
}

// computeGyreForcing calculates the gyre rotation vector for a vertex in a basin.
// Includes western boundary intensification for realistic narrow boundary currents.
func computeGyreForcing(vertex Vector3D, basin Basin, targetSpeed float64) Vector3D {
	centroid := basin.Centroid
	maxRadius := basin.MaxRadius

	if maxRadius < 1e-9 {
		return Vector3D{}
	}

	// Tangential direction for clockwise rotation (NH): centroid × vertex
	// This gives the correct direction: east of center → flow south, etc.
	tangent := Cross(centroid, vertex)
	tangentLen := Length(tangent)

	if tangentLen < 1e-9 {
		return Vector3D{}
	}

	tangent = Scale(tangent, 1.0/tangentLen)

	// Northern hemisphere gyres rotate clockwise, southern counter-clockwise
	if !basin.IsNorthern() {
		tangent = Scale(tangent, -1.0)
	}

	// Speed scales with distance from center (faster at edges)
	angularDist := AngularDistance(vertex, centroid)
	scaleFactor := angularDist / maxRadius
	if scaleFactor > 1.0 {
		scaleFactor = 1.0
	}
	// Use smoother scaling (less quadratic, more linear for better blending)
	scaleFactor = 0.3 + 0.7*scaleFactor

	// Western boundary intensification:
	// Real gyres have narrow, fast western boundary currents (Gulf Stream, Kuroshio)
	// The "western" side is where flow is poleward (northward in NH, southward in SH)
	_, north := GetTangentVectors(vertex)
	polewardComponent := Dot(tangent, north)
	if !basin.IsNorthern() {
		polewardComponent = -polewardComponent // In SH, poleward is southward
	}

	// Get longitude relative to basin center to find western side
	// Western side = negative longitude relative to centroid (for standard projection)
	east, _ := GetTangentVectors(centroid)
	toVertex := Sub(vertex, centroid)
	relLon := Dot(toVertex, east) // Positive = east of center, negative = west

	// Boost if on western side AND flowing poleward
	if relLon < 0 && polewardComponent > 0.3 {
		// Western boundary intensification: narrower, faster current
		westFactor := 1.0 + 1.5*(-relLon/maxRadius)*polewardComponent
		if westFactor > 2.5 {
			westFactor = 2.5
		}
		scaleFactor *= westFactor
	}

	return Scale(tangent, targetSpeed*scaleFactor)
}

// computeBlendedGyreForcing blends influence from multiple nearby basins.
// Creates smooth transitions and connecting boundary currents between adjacent gyres.
func computeBlendedGyreForcing(vertex Vector3D, basins []Basin, targetSpeed float64) Vector3D {
	var totalCurrent Vector3D
	totalWeight := 0.0

	// Track influences for boundary current boost
	var influences []struct {
		basin  Basin
		weight float64
		dist   float64
	}

	for _, basin := range basins {
		dist := AngularDistance(vertex, basin.Centroid)

		// Influence radius: moderate extension for coalescence while preserving structure
		influenceRadius := basin.MaxRadius * 3.5
		if dist > influenceRadius {
			continue
		}

		// Weight falls off with distance (quadratic) scaled by basin size
		weight := 1.0 - (dist / influenceRadius)
		weight = weight * weight          // Quadratic falloff preserves gyre cores
		weight *= basin.MaxRadius         // Larger basins have more influence

		influences = append(influences, struct {
			basin  Basin
			weight float64
			dist   float64
		}{basin, weight, dist})

		// Get gyre contribution from this basin
		gyreCurrent := computeGyreForcing(vertex, basin, targetSpeed)

		totalCurrent = Add(totalCurrent, Scale(gyreCurrent, weight))
		totalWeight += weight
	}

	if totalWeight > 0 {
		result := Scale(totalCurrent, 1.0/totalWeight)

		// Boundary current boost: when between two basins in same hemisphere,
		// boost the current to create connecting boundary currents
		if len(influences) >= 2 {
			// Find top two influences
			var top1, top2 int
			maxW := 0.0
			for i, inf := range influences {
				if inf.weight > maxW {
					top2 = top1
					top1 = i
					maxW = inf.weight
				}
			}
			if top1 != top2 && len(influences) > top2 {
				b1, b2 := influences[top1].basin, influences[top2].basin
				// If same hemisphere, boost boundary current
				if b1.IsNorthern() == b2.IsNorthern() {
					// Boost proportional to how balanced the influences are
					balance := influences[top2].weight / influences[top1].weight
					if balance > 0.3 { // Significant overlap
						boostFactor := 1.0 + 0.5*balance
						result = Scale(result, boostFactor)
					}
				}
			}
		}

		return result
	}

	return Vector3D{}
}

// computeWindDrivenCurrent adds latitude-based wind forcing.
// Trade winds (easterlies) near equator, westerlies at mid-latitudes.
func computeWindDrivenCurrent(vertex Vector3D, strength float64) Vector3D {
	// Get latitude (-1 to 1, where Y is up)
	lat := vertex.Y

	// Get local east direction
	east, _ := GetTangentVectors(vertex)

	// Wind pattern by latitude:
	// 0-30°: Trade winds (blow west, push current west)
	// 30-60°: Westerlies (blow east, push current east)
	// 60-90°: Polar easterlies (blow west)
	absLat := lat
	if absLat < 0 {
		absLat = -absLat
	}

	var windDir float64
	if absLat < 0.5 { // ~30° latitude
		// Trade winds - westward flow
		windDir = -1.0
		// Stronger near 15°, weaker at equator and 30°
		windDir *= 0.5 + 0.5*(1.0-4.0*(absLat-0.25)*(absLat-0.25))
	} else if absLat < 0.87 { // ~60° latitude
		// Westerlies - eastward flow
		windDir = 1.0
		// Strongest around 45°
		t := (absLat - 0.5) / 0.37
		windDir *= 0.6 + 0.4*(1.0-4.0*(t-0.5)*(t-0.5))
	} else {
		// Polar easterlies - westward, weaker
		windDir = -0.5
	}

	return Scale(east, strength*windDir)
}

// GenerateStreamfunction creates a scalar streamfunction field from basin data.
// This is the older basin-based approach, kept for comparison.
// Northern hemisphere basins get positive Ψ (clockwise flow when viewed from above)
// Southern hemisphere basins get negative Ψ (counter-clockwise flow)
func GenerateStreamfunction(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	basins []Basin,
	strength float64,
) []float64 {
	numVertices := len(vertices)
	psi := make([]float64, numVertices)

	// Identify coastline vertices (water adjacent to land)
	isCoastline := make([]bool, numVertices)
	for i := 0; i < numVertices; i++ {
		if elevation[i] >= seaLevelThreshold {
			continue
		}
		for _, k := range adj.GetNeighbors(i) {
			if k >= 0 && k < numVertices && elevation[k] >= seaLevelThreshold {
				isCoastline[i] = true
				break
			}
		}
	}

	// Add basin contributions to streamfunction
	for _, basin := range basins {
		sign := 1.0
		if !basin.IsNorthern() {
			sign = -1.0 // Southern hemisphere: negative Ψ
		}

		for i := 0; i < numVertices; i++ {
			if elevation[i] >= seaLevelThreshold {
				continue // Skip land
			}
			if isCoastline[i] {
				continue // Coastlines stay at Ψ = 0
			}

			dist := AngularDistance(vertices[i], basin.Centroid)
			if dist > basin.MaxRadius*2.5 {
				continue // Beyond influence range
			}

			// Radial profile: Gaussian-like bump centered on basin
			// Ψ = A * exp(-r²/σ²) where σ scales with basin size
			sigma := basin.MaxRadius * 0.8
			profile := math.Exp(-(dist * dist) / (sigma * sigma))

			// Scale by basin size (larger basins = stronger Ψ)
			basinStrength := strength * basin.MaxRadius * basin.MaxRadius

			psi[i] += sign * basinStrength * profile
		}
	}

	return psi
}
