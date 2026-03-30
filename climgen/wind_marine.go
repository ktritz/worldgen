package climgen

import "math"

// BuildMarineWind derives a smoother, basin-scale wind field intended for
// ocean stress and maritime transport. It keeps the large-scale circulation and
// pressure steering but avoids terrain and lee-shadow perturbations.
func BuildMarineWind(
	wind []Vector3D,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	settings SurfaceWindSettings,
) []Vector3D {
	marineSettings := settings
	marineSettings.OceanFriction *= 0.6
	if marineSettings.OceanFriction < 0.03 {
		marineSettings.OceanFriction = 0.03
	}
	marineSettings.LandFriction = math.Max(marineSettings.LandFriction, 0.55)

	marineWind := ApplySurfaceFrictionSimple(
		wind, vertices, elevation, seaLevelThreshold, marineSettings,
	)

	cellSize := estimateCellSize(vertices, adj)
	smoothIters := int(0.08/cellSize) + 1
	if smoothIters < 3 {
		smoothIters = 3
	}

	marineWind = SmoothVectorFieldBySurface(
		marineWind, vertices, elevation, seaLevelThreshold, adj, smoothIters, 0.45,
	)

	coastalReach := ComputeSurfaceInteriorFraction(
		elevation, seaLevelThreshold, adj, 1200.0, true,
	)
	result := make([]Vector3D, len(marineWind))
	for i := range marineWind {
		if elevation[i] < seaLevelThreshold {
			result[i] = marineWind[i]
			continue
		}

		// Keep only a coastal marine signal over land so this product stays
		// usable near shore without pretending to be a full land wind field.
		reach := 1.0 - coastalReach[i]
		if reach < 0 {
			reach = 0
		}
		reach *= reach
		result[i] = Scale(marineWind[i], 0.45*reach)
	}

	return result
}
