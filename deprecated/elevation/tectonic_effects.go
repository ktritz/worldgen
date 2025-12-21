package elevation

import (
	"math"

	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
)

// applyTectonicBoundaryEffects modifies elevation based on tectonic features at a site.
// This function handles:
// - Distance-based falloff from plate boundaries
// - Different effects for convergent, divergent, and passive boundaries
// - Elevation modifications based on boundary interaction types
func applyTectonicBoundaryEffects(
	baseSiteElevation float64,
	siteID int32,
	tectonicData *tectonics.TectonicsData,
	icosphereSites []icosphere.Vector3D,
	params ElevationParameters,
	planetRadius float64,
) float64 {
	
	// Modify elevation based on tectonic features at this site.
	// Use SiteDistancesToBoundary (angular distance for unit sphere, needs scaling by radius for absolute)
	// Or use NearestBoundarySiteIndices to get the type of the nearest boundary.

	// distToBoundary is angular if radius=1. Multiply by PlanetRadius for absolute distance.
	distToBoundaryAbs := tectonicData.SiteDistancesToBoundary[siteID] * planetRadius

	if distToBoundaryAbs < params.MaxBoundaryEffectDistAbs {
		// Ensure NearestBoundarySiteIndices[siteID] is valid
		nearestBoundaryIdx := tectonicData.NearestBoundarySiteIndices[siteID]
		if nearestBoundaryIdx != -1 && int(nearestBoundaryIdx) < len(icosphereSites) {

			// Calculate falloff factor: 1.0 at boundary, fading to 0.0 at maxBoundaryEffectDistance.
			falloffFactor := calculateFalloffFactor(distToBoundaryAbs, params.MaxBoundaryEffectDistAbs)

			// Get the interaction type of the *nearest boundary site*.
			boundaryTypeForEffect, typeExists := tectonicData.SiteBoundaryTypes[nearestBoundaryIdx]
			if typeExists {
				elevationModification := calculateElevationModification(boundaryTypeForEffect, params, falloffFactor)
				baseSiteElevation += elevationModification
			}
		}
	}
	
	return baseSiteElevation
}

// calculateFalloffFactor computes the distance-based falloff factor.
// Returns 1.0 at boundary (distance = 0), fading to 0.0 at maxDistance.
func calculateFalloffFactor(distanceToBoundary, maxBoundaryEffectDistance float64) float64 {
	if maxBoundaryEffectDistance <= 0 {
		return 0.0
	}
	
	falloffFactor := (maxBoundaryEffectDistance - distanceToBoundary) / maxBoundaryEffectDistance
	falloffFactor = math.Max(0, falloffFactor)    // Clamp to non-negative
	falloffFactor = falloffFactor * falloffFactor // Squared falloff for smoother transition
	
	return falloffFactor
}

// calculateElevationModification determines the elevation change based on boundary type.
func calculateElevationModification(
	boundaryType tectonics.PlateInteractionType,
	params ElevationParameters,
	falloffFactor float64,
) float64 {
	switch boundaryType {
	case tectonics.Convergent:
		// Convergent boundaries create mountains/highlands
		return params.ConvergentStrength * falloffFactor
		
	case tectonics.Divergent:
		// Divergent boundaries create valleys/subsidence
		return -params.DivergentStrength * falloffFactor // Negative for subsidence
		
	case tectonics.Passive:
		// Passive boundaries have minimal effect
		// Could add slight noise or small ridges here in the future
		// Minor effect for passive, e.g., slight noise or small ridges
		// baseSiteElevation += (rand.New(rand.NewSource(elevationSeed + int64(siteID))).Float64()*2.0 - 1.0) * (baseAmplitude * 0.05) * falloffFactor
		return 0.0
		
	default:
		return 0.0
	}
}