package climgen

import (
	"math"
)

// =============================================================================
// SURFACE WIND - FRICTION AND BOUNDARY LAYER EFFECTS
// =============================================================================
// Surface friction reduces wind speed (more over land, less over ocean).
// The cell-driven circulation model already produces correct wind directions,
// so friction is mainly about speed reduction.

// ApplySurfaceFrictionSimple reduces wind speed based on surface type.
// Used with cell-driven wind which already has correct directions.
func ApplySurfaceFrictionSimple(
	wind []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	settings SurfaceWindSettings,
) []Vector3D {
	numVertices := len(vertices)
	result := make([]Vector3D, numVertices)

	for i := range vertices {
		// Determine friction coefficient based on surface type
		friction := settings.OceanFriction
		if elevation[i] >= seaLevelThreshold {
			friction = settings.LandFriction
		}

		// Speed reduction from friction
		speedFactor := 1.0 - friction

		// Apply speed reduction while preserving direction
		result[i] = Scale(wind[i], speedFactor)

		// Clamp to max speed
		speed := Length(result[i])
		if speed > settings.MaxWindSpeed {
			result[i] = Scale(result[i], settings.MaxWindSpeed/speed)
		}
	}

	return result
}

// =============================================================================
// LEGACY FRICTION MODEL (with deflection toward low pressure)
// =============================================================================
// This was used with geostrophic wind but is less accurate for surface winds.
// Kept for reference and potential future use.

// ApplySurfaceFriction modifies geostrophic wind for surface friction effects.
// Friction reduces speed and deflects wind toward low pressure.
func ApplySurfaceFriction(
	geostrophicWind []Vector3D,
	pressure []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	settings SurfaceWindSettings,
) []Vector3D {
	numVertices := len(vertices)
	surfaceWind := make([]Vector3D, numVertices)

	for i := range vertices {
		geoWind := geostrophicWind[i]
		geoSpeed := Length(geoWind)

		if geoSpeed < 1e-9 {
			continue
		}

		// Determine friction coefficient based on surface type
		friction := settings.OceanFriction
		if elevation[i] >= seaLevelThreshold {
			friction = settings.LandFriction
		}

		// Speed reduction from friction
		speedFactor := 1.0 - friction

		// Compute pressure gradient direction (toward low pressure)
		normal := vertices[i]
		east, north := GetTangentVectors(vertices[i])

		// Simple gradient estimation from neighbors
		var gradE, gradN float64
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

			dp := pressure[k] - pressure[i]
			gradE += dp * de
			gradN += dp * dn
			count++
		}

		if count > 0 {
			gradE /= float64(count)
			gradN /= float64(count)
		}

		// Low pressure direction (negative of gradient)
		lowPressureDir := Add(Scale(east, -gradE), Scale(north, -gradN))
		lpLen := Length(lowPressureDir)

		// Geostrophic wind direction
		geoDir := Scale(geoWind, 1.0/geoSpeed)

		// Deflection angle depends on friction
		// More friction = more deflection toward low pressure
		// Typically 10-15° over ocean, 25-45° over land
		deflectionAngle := friction * math.Pi / 4 // Max ~45 degrees at friction=1

		var surfaceDir Vector3D
		if lpLen > 1e-9 {
			lowPressureDir = Scale(lowPressureDir, 1.0/lpLen)

			// Blend geostrophic direction toward low pressure direction
			// Using rotation: rotate geoDir toward lowPressureDir by deflectionAngle
			surfaceDir = Add(
				Scale(geoDir, math.Cos(deflectionAngle)),
				Scale(lowPressureDir, math.Sin(deflectionAngle)),
			)
			surfaceDir = Normalize(surfaceDir)
		} else {
			surfaceDir = geoDir
		}

		// Apply speed reduction
		surfaceSpeed := geoSpeed * speedFactor

		// Clamp to max speed
		if surfaceSpeed > settings.MaxWindSpeed {
			surfaceSpeed = settings.MaxWindSpeed
		}

		surfaceWind[i] = Scale(surfaceDir, surfaceSpeed)
	}

	return surfaceWind
}
