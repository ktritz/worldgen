package elevation

// tectonic_elevation.go - Tectonic boundary elevation effects and mountain building

import (
	"math"
	"worldgen/landgen/tectonics"
)

// GenerateTectonicElevations calculates elevation contributions from tectonic boundary interactions
func GenerateTectonicElevations(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64 {
	tectonicElevations := make([]float64, len(icosphereSites))
	
	// Calculate subduction zone effects separately since they operate on the full site array
	subductionEffects := CalculateSubductionZoneEffects(icosphereSites, tectonicData, params)
	
	for siteIdx := range icosphereSites {
		siteID := int32(siteIdx)
		
		// Apply boundary effects based on distance to boundaries
		boundaryEffect := calculateBoundaryElevationEffect(siteID, tectonicData, params)
		
		// Apply convergent zone mountain building
		convergentEffect := calculateConvergentZoneEffect(siteID, tectonicData, params)
		
		// Apply divergent zone rift effects
		divergentEffect := calculateDivergentZoneEffect(siteID, tectonicData, params)
		
		// Apply transform fault effects
		transformEffect := calculateTransformFaultEffect(siteID, tectonicData, params)
		
		// Get subduction zone effect for this site
		subductionEffect := subductionEffects[siteIdx]
		
		// Combine all tectonic effects
		tectonicElevations[siteIdx] = boundaryEffect + convergentEffect + divergentEffect + transformEffect + subductionEffect
	}
	
	return tectonicElevations
}

// calculateBoundaryElevationEffect applies general boundary proximity effects
// Note: Convergent boundaries are handled separately in calculateConvergentZoneEffect
func calculateBoundaryElevationEffect(siteID int32, tectonicData *TectonicsData, params ElevationParameters) float64 {
	// Get distance to nearest boundary
	if int(siteID) >= len(tectonicData.SiteDistancesToBoundary) {
		return 0.0
	}

	distToBoundaryAbs := tectonicData.SiteDistancesToBoundary[siteID] * params.PlanetRadius

	if distToBoundaryAbs >= params.MaxBoundaryEffectDistAbs {
		return 0.0 // Outside influence zone
	}

	// Get boundary type
	nearestBoundaryIdx := tectonicData.NearestBoundarySiteIndices[siteID]
	if nearestBoundaryIdx == -1 {
		return 0.0
	}

	boundaryType, exists := tectonicData.SiteBoundaryTypes[nearestBoundaryIdx]
	if !exists {
		return 0.0
	}

	// Calculate falloff factor
	falloffFactor := calculateBoundaryFalloff(distToBoundaryAbs, params)

	// Apply effect based on boundary type
	switch boundaryType {
	case tectonics.Convergent:
		// Convergent boundaries are handled by calculateConvergentZoneEffect
		// to properly distinguish ocean-ocean vs continent-continent
		return 0.0
	case tectonics.Divergent:
		return -params.DivergentStrength * falloffFactor // Negative for subsidence
	case tectonics.Passive:
		return calculateTransformTopography(distToBoundaryAbs, params) * falloffFactor
	default:
		return 0.0
	}
}

// calculateBoundaryFalloff computes distance-based falloff for boundary effects
func calculateBoundaryFalloff(distance float64, params ElevationParameters) float64 {
	if distance >= params.MaxBoundaryEffectDistAbs {
		return 0.0
	}
	
	// Characteristic distance for exponential decay
	charDist := params.CharacteristicFalloffDistAbs
	if charDist <= 0 {
		charDist = params.MaxBoundaryEffectDistAbs * 0.3 // Default to 30% of max distance
	}
	
	// Exponential decay with distance
	return math.Exp(-distance / charDist)
}

// calculateConvergentZoneEffect applies specific convergent boundary elevation based on plate types
// This function applies to BOTH boundary sites AND inland sites (for volcanic arcs)
// Uses rich boundary information for O(1) queries
func calculateConvergentZoneEffect(siteID int32, tectonicData *TectonicsData, params ElevationParameters) float64 {
	// Use rich boundary info for efficient O(1) lookup
	if int(siteID) >= len(tectonicData.NearestBoundaryInfo) {
		return 0.0
	}

	boundaryInfo := tectonicData.NearestBoundaryInfo[siteID]

	// Only process sites near convergent boundaries
	if boundaryInfo.BoundaryType != tectonics.Convergent {
		return 0.0
	}

	// Get distance to boundary in kilometers
	distanceKm := boundaryInfo.Distance * params.PlanetRadius / 1000.0

	// Determine base elevation based on BOTH plate types involved
	var baseElevation float64

	if boundaryInfo.LocalPlateType == tectonics.ContinentalPlate && boundaryInfo.AdjacentPlateType == tectonics.ContinentalPlate {
		// ===== CONTINENT-CONTINENT CONVERGENCE =====
		// Creates the highest mountains on Earth, but NOT all are Himalayas!
		// Himalayas are exceptional (16 cm/year convergence, young, active)
		// Most continent-continent boundaries are older and lower (Appalachians, Urals)
		// Use noise to vary mountain heights - only ~20% should be Himalaya-scale

		convergenceIntensity := getNormalizedNoise(siteID, params.ElevationSeed+999)

		// Distance falloff for continent-continent collision
		intensityFactor := math.Max(0.3, 1.0-distanceKm/500.0) // Active up to 500km from boundary

		if convergenceIntensity > 0.85 {
			// Exceptional collision like Himalayas (15-20% of boundaries)
			// Increased to compensate for erosion
			baseElevation = 7000.0 * intensityFactor // Up to 7000m (Himalayan peaks)
		} else if convergenceIntensity > 0.5 {
			// Moderate collision like Alps or Rockies (30% of boundaries)
			baseElevation = 4500.0 * intensityFactor
		} else {
			// Old/slow collision like Appalachians or Urals (50% of boundaries)
			baseElevation = 2500.0 * intensityFactor
		}

	} else if boundaryInfo.LocalPlateType == tectonics.OceanicPlate && boundaryInfo.AdjacentPlateType == tectonics.OceanicPlate {
		// ===== OCEAN-OCEAN CONVERGENCE =====
		// Creates DEEP TRENCHES (Mariana: -11,000m), NOT uplift!
		// The boundary itself is a trench, volcanic islands form BEHIND it

		// Only create trench near the actual boundary
		if distanceKm < 100.0 {
			intensityFactor := 1.0 - distanceKm/100.0
			baseElevation = -4000.0 * intensityFactor // Deep trench at boundary
		} else {
			baseElevation = 0.0
		}

	} else if boundaryInfo.LocalPlateType == tectonics.OceanicPlate && boundaryInfo.AdjacentPlateType == tectonics.ContinentalPlate {
		// ===== OCEAN PLATE SIDE OF OCEAN-CONTINENT CONVERGENCE =====
		// Creates offshore trench

		if distanceKm < 150.0 {
			intensityFactor := 1.0 - distanceKm/150.0
			baseElevation = -3000.0 * intensityFactor // Offshore trench
		} else {
			baseElevation = 0.0
		}

	} else if boundaryInfo.LocalPlateType == tectonics.ContinentalPlate && boundaryInfo.AdjacentPlateType == tectonics.OceanicPlate {
		// ===== CONTINENTAL PLATE SIDE OF OCEAN-CONTINENT CONVERGENCE =====
		// CRITICAL GEOLOGY: This is where volcanic arcs form!
		// Real subduction zone structure:
		//   0-50 km:     Nearshore (flat coastal plain, NO mountains)
		//   50-100 km:   Forearc basin (subsidence, LOW elevation)
		//   100-300 km:  VOLCANIC ARC (WHERE MOUNTAINS FORM!)
		//   300-500 km:  Backarc (moderate elevation, decreasing)
		//   >500 km:     Far inland (minimal tectonic effect)

		activityNoise := getNormalizedNoise(siteID, params.ElevationSeed+777)

		// Determine subduction intensity (not all subduction zones are equally active)
		var maxArcElevation float64
		if activityNoise > 0.7 {
			// Active subduction: significant mountains (Andes-like, 4-7km peaks)
			// Increased to compensate for erosion and ensure >2000m mountains
			maxArcElevation = 6000.0
		} else if activityNoise > 0.4 {
			// Moderate subduction: moderate mountains (Cascades-like, 2-3km)
			maxArcElevation = 4000.0
		} else {
			// Weak/old subduction: hills (1km)
			maxArcElevation = 2500.0
		}

		// Apply distance-based volcanic arc zones
		if distanceKm < 50.0 {
			// Nearshore: flat coastal plain
			baseElevation = 100.0 * (1.0 - distanceKm/50.0)
		} else if distanceKm < 100.0 {
			// Forearc basin: slight subsidence to low elevation
			t := (distanceKm - 50.0) / 50.0
			baseElevation = 100.0 * (1.0 - t) // Decreases from 100m to 0m
		} else if distanceKm < 300.0 {
			// VOLCANIC ARC ZONE: This is where mountains form!
			// Peak elevation around 200km from trench (center of arc)
			centerDistance := 200.0
			arcWidth := 100.0 // Mountains span 100-300km

			// Gaussian-like profile centered at 200km
			distFromCenter := math.Abs(distanceKm - centerDistance)
			arcFactor := math.Exp(-math.Pow(distFromCenter/arcWidth, 2))

			baseElevation = maxArcElevation * arcFactor
		} else if distanceKm < 500.0 {
			// Backarc: decreasing elevation inland
			t := (distanceKm - 300.0) / 200.0
			baseElevation = maxArcElevation * 0.3 * (1.0 - t) // Decreases from 30% of max to 0
		} else {
			// Far inland: no tectonic effect
			baseElevation = 0.0
		}
	}

	// Add some randomness for realistic distribution
	randomFactor := 0.8 + 0.4*getNormalizedNoise(siteID, params.ElevationSeed)

	return baseElevation * randomFactor
}

// findAdjacentPlateType determines the type of adjacent plate at a convergent boundary
// For inland sites, this checks the plate type at the nearest boundary
func findAdjacentPlateType(siteID int32, tectonicData *TectonicsData) tectonics.PlateType {
	thisPlateID := tectonicData.SitePlateIDs[siteID]
	thisPlate := findPlateByID(tectonicData.Plates, thisPlateID)

	if thisPlate == nil {
		return tectonics.OceanicPlate
	}

	// Get nearest boundary site
	nearestBoundaryIdx := tectonicData.NearestBoundarySiteIndices[siteID]
	if nearestBoundaryIdx == -1 || int(nearestBoundaryIdx) >= len(tectonicData.SitePlateIDs) {
		// No boundary info available - use heuristic
		// Continental plates near convergent boundaries are more likely near oceans (subduction)
		if thisPlate.PlateType == tectonics.ContinentalPlate {
			return tectonics.OceanicPlate // Assume ocean-continent subduction
		}
		return tectonics.OceanicPlate
	}

	// Get the plate ID at the nearest boundary
	boundaryPlateID := tectonicData.SitePlateIDs[nearestBoundaryIdx]

	// If boundary site is on a different plate, that's our adjacent plate
	if boundaryPlateID != thisPlateID {
		adjacentPlate := findPlateByID(tectonicData.Plates, boundaryPlateID)
		if adjacentPlate != nil {
			return adjacentPlate.PlateType
		}
	}

	// Fallback: For continental plates, assume they're near oceanic subduction
	// (This is the most common source of mountains - volcanic arcs)
	if thisPlate.PlateType == tectonics.ContinentalPlate {
		return tectonics.OceanicPlate
	}

	// For oceanic plates, assume oceanic collision
	return tectonics.OceanicPlate
}

// calculateDivergentZoneEffect applies rift valley and spreading center effects
func calculateDivergentZoneEffect(siteID int32, tectonicData *TectonicsData, params ElevationParameters) float64 {
	// Check if this site is at a divergent boundary
	boundaryType, exists := tectonicData.SiteBoundaryTypes[siteID]
	if !exists || boundaryType != tectonics.Divergent {
		return 0.0
	}
	
	plateID := tectonicData.SitePlateIDs[siteID]
	plate := findPlateByID(tectonicData.Plates, plateID)
	
	if plate == nil {
		return 0.0
	}
	
	// Different effects for continental vs oceanic rifting
	switch plate.PlateType {
	case tectonics.ContinentalPlate:
		// Continental rifting creates rift valleys
		return calculateContinentalRiftEffect(siteID, tectonicData, params)
	case tectonics.OceanicPlate:
		// Oceanic spreading centers (handled more in ridge_elevation.go)
		return calculateOceanicSpreadingEffect(siteID, tectonicData, params)
	default:
		return 0.0
	}
}

// calculateContinentalRiftEffect models continental rift valley formation
func calculateContinentalRiftEffect(siteID int32, tectonicData *TectonicsData, params ElevationParameters) float64 {
	// Continental rifts create valleys and escarpments
	// Examples: East African Rift, Basin and Range Province
	
	distToBoundary := tectonicData.SiteDistancesToBoundary[siteID] * params.PlanetRadius
	
	// Rift valley profile
	if distToBoundary < 10000.0 { // Within 10 km of rift axis
		// Central rift valley
		return -800.0 // 800m deep rift valley
	} else if distToBoundary < 50000.0 { // Within 50 km
		// Rift shoulders/escarpments
		elevationGain := 600.0 * math.Exp(-distToBoundary/20000.0) // Exponential decay
		return elevationGain
	}
	
	return 0.0
}

// calculateOceanicSpreadingEffect models oceanic spreading center topography
func calculateOceanicSpreadingEffect(siteID int32, tectonicData *TectonicsData, params ElevationParameters) float64 {
	// Basic spreading center elevation (detailed effects in ridge_elevation.go)
	distToBoundary := tectonicData.SiteDistancesToBoundary[siteID] * params.PlanetRadius
	
	// Small ridge elevation at spreading centers
	if distToBoundary < 5000.0 { // Within 5 km of spreading axis
		return 500.0 * (1.0 - distToBoundary/5000.0) // Linear decay
	}
	
	return 0.0
}

// calculateTransformFaultEffect applies transform fault topographic effects
func calculateTransformFaultEffect(siteID int32, tectonicData *TectonicsData, params ElevationParameters) float64 {
	// Check if this site is at a transform boundary
	boundaryType, exists := tectonicData.SiteBoundaryTypes[siteID]
	if !exists || boundaryType != tectonics.Passive {
		return 0.0
	}
	
	distToBoundary := tectonicData.SiteDistancesToBoundary[siteID] * params.PlanetRadius
	
	// Transform faults create linear valleys and scarps
	return calculateTransformTopography(distToBoundary, params)
}

// calculateTransformTopography models transform fault topographic signature
func calculateTransformTopography(distance float64, params ElevationParameters) float64 {
	if distance > 20000.0 { // 20 km influence zone
		return 0.0
	}
	
	// Transform faults create linear valleys
	if distance < 2000.0 { // Within 2 km of fault trace
		// Central fault valley
		return -200.0 * (1.0 - distance/2000.0) // Up to 200m deep valley
	} else if distance < 10000.0 { // Within 10 km
		// Fault zone damage and minor topography
		return -50.0 * math.Exp(-distance/5000.0)
	}
	
	return 0.0
}

// getPlateVelocityAt retrieves plate velocity at a given site
func getPlateVelocityAt(siteID int32, tectonicData *TectonicsData, sites []Vector3D) [3]float64 {
	if int(siteID) >= len(sites) {
		return [3]float64{0, 0, 0}
	}
	
	plateID := tectonicData.SitePlateIDs[siteID]
	plate := findPlateByID(tectonicData.Plates, plateID)
	
	if plate == nil {
		return [3]float64{0, 0, 0}
	}
	
	// Calculate velocity at this site using plate rotation
	sitePosition := sites[siteID]
	velocity := plate.GetVelocityAtPoint(sitePosition)
	
	return [3]float64{velocity.X, velocity.Y, velocity.Z}
}

// getNormalizedNoise generates consistent pseudo-random noise for a site
func getNormalizedNoise(siteID int32, seed int64) float64 {
	// Simple hash-based noise generation
	hash := uint64(seed) ^ uint64(siteID)*1103515245 + 12345
	hash = ((hash >> 16) ^ hash) * 0x45d9f3b
	hash = ((hash >> 16) ^ hash) * 0x45d9f3b
	hash = (hash >> 16) ^ hash
	
	return float64(hash&0xFFFF) / 65535.0 // Normalize to [0,1]
}

// CalculateSubductionZoneEffects models specific subduction zone topography
func CalculateSubductionZoneEffects(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64 {
	subductionEffects := make([]float64, len(icosphereSites))
	
	// Process subduction zones if available
	if len(tectonicData.SubductionZones) == 0 {
		return subductionEffects
	}
	
	for siteIdx, site := range icosphereSites {
		maxSubductionEffect := 0.0
		
		for _, subZone := range tectonicData.SubductionZones {
			// Calculate distance to subduction zone
			distanceToZone := calculateDistanceToSubductionZone(site, subZone, params.PlanetRadius)
			
			if distanceToZone < 500000.0 { // 500 km influence zone
				effect := calculateSubductionTopography(distanceToZone, subZone, params)
				if math.Abs(effect) > math.Abs(maxSubductionEffect) {
					maxSubductionEffect = effect
				}
			}
		}
		
		subductionEffects[siteIdx] = maxSubductionEffect
	}
	
	return subductionEffects
}

// calculateDistanceToSubductionZone computes distance to a subduction zone
func calculateDistanceToSubductionZone(site Vector3D, subZone SubductionZone, planetRadius float64) float64 {
	// Use distance to closest trench point
	if len(subZone.TrenchPoints) == 0 {
		return planetRadius // Very far if no trench points
	}
	
	minDistance := planetRadius
	for _, trenchPoint := range subZone.TrenchPoints {
		distance := calculateSphericalDistance(site, trenchPoint, planetRadius)
		if distance < minDistance {
			minDistance = distance
		}
	}
	
	return minDistance
}

// calculateSubductionTopography models subduction zone topographic profile
func calculateSubductionTopography(distance float64, subZone SubductionZone, params ElevationParameters) float64 {
	// Subduction zones create trenches and volcanic arcs
	
	if distance < 50000.0 { // Within 50 km - trench area
		// Ocean trench - deeper for faster convergence
		baseTrenchDepth := -8000.0 * params.TrenchDepthMultiplier
		convergenceFactor := math.Min(2.0, subZone.ConvergenceRate/5.0) // Scale by convergence rate
		trenchDepth := baseTrenchDepth * convergenceFactor
		return trenchDepth * math.Exp(-distance/20000.0)
		
	} else if distance > subZone.ArcDistance*1000.0-50000.0 && distance < subZone.ArcDistance*1000.0+100000.0 {
		// Volcanic arc behind trench - use actual arc distance from subduction zone
		arcCenterDistance := subZone.ArcDistance * 1000.0 // Convert km to meters
		arcDistance := math.Abs(distance - arcCenterDistance)
		arcWidth := 100000.0 // 100 km arc width
		
		if arcDistance < arcWidth {
			arcProfile := math.Exp(-math.Pow(arcDistance/(arcWidth*0.5), 2))
			return params.ArcElevation * arcProfile
		}
	}
	
	return 0.0
}

// ValidateTectonicElevations performs validation on tectonic elevation effects
func ValidateTectonicElevations(tectonicElevations []float64, tectonicData *TectonicsData) (TectonicElevationMetrics, []string) {
	var metrics TectonicElevationMetrics
	var warnings []string
	
	if len(tectonicElevations) == 0 {
		warnings = append(warnings, "No tectonic elevations generated")
		return metrics, warnings
	}
	
	// Separate positive (mountain building) and negative (subsidence) effects
	var positiveEffects, negativeEffects []float64
	maxPositive := 0.0
	minNegative := 0.0
	
	for _, elevation := range tectonicElevations {
		if elevation > 50.0 { // Significant positive effect
			positiveEffects = append(positiveEffects, elevation)
			if elevation > maxPositive {
				maxPositive = elevation
			}
		} else if elevation < -50.0 { // Significant negative effect
			negativeEffects = append(negativeEffects, elevation)
			if elevation < minNegative {
				minNegative = elevation
			}
		}
	}
	
	// Calculate metrics
	metrics.MountainBuildingSites = len(positiveEffects)
	metrics.SubsidenceSites = len(negativeEffects)
	metrics.MaxMountainElevation = maxPositive
	metrics.MaxSubsidenceDepth = minNegative
	
	if len(positiveEffects) > 0 {
		metrics.MeanMountainElevation = calculateMean(positiveEffects)
	}
	
	if len(negativeEffects) > 0 {
		metrics.MeanSubsidenceDepth = calculateMean(negativeEffects)
	}
	
	// Count boundary types
	convergentSites := 0
	divergentSites := 0
	transformSites := 0
	
	for _, boundaryType := range tectonicData.SiteBoundaryTypes {
		switch boundaryType {
		case tectonics.Convergent:
			convergentSites++
		case tectonics.Divergent:
			divergentSites++
		case tectonics.Passive:
			transformSites++
		}
	}
	
	metrics.ConvergentBoundarySites = convergentSites
	metrics.DivergentBoundarySites = divergentSites
	metrics.TransformBoundarySites = transformSites
	
	// Validation checks
	if convergentSites > 0 && len(positiveEffects) == 0 {
		warnings = append(warnings, "No mountain building despite convergent boundaries")
	}
	
	if divergentSites > 0 && len(negativeEffects) == 0 {
		warnings = append(warnings, "No subsidence despite divergent boundaries")
	}
	
	if maxPositive > 8000.0 {
		warnings = append(warnings, "Maximum mountain elevation is very high (>8km)")
	}
	
	if minNegative < -12000.0 {
		warnings = append(warnings, "Maximum subsidence is very deep (<-12km)")
	}
	
	// Earth comparison
	if metrics.MeanMountainElevation < 1000.0 && len(positiveEffects) > 0 {
		warnings = append(warnings, "Mean mountain elevation is low compared to Earth mountain ranges")
	}
	
	return metrics, warnings
}

// TectonicElevationMetrics contains statistics about tectonic elevation effects
type TectonicElevationMetrics struct {
	// Mountain building metrics
	MountainBuildingSites   int     // Number of sites with mountain building
	MaxMountainElevation    float64 // Maximum mountain elevation effect (m)
	MeanMountainElevation   float64 // Mean mountain elevation where present (m)
	
	// Subsidence metrics
	SubsidenceSites         int     // Number of sites with subsidence
	MaxSubsidenceDepth      float64 // Maximum subsidence depth (m, negative)
	MeanSubsidenceDepth     float64 // Mean subsidence depth where present (m, negative)
	
	// Boundary type counts
	ConvergentBoundarySites int     // Number of convergent boundary sites
	DivergentBoundarySites  int     // Number of divergent boundary sites
	TransformBoundarySites  int     // Number of transform boundary sites
}

// CalculateTectonicRoughness measures topographic roughness from tectonic processes
func CalculateTectonicRoughness(tectonicElevations []float64, icosphereSites []Vector3D, planetRadius float64) float64 {
	if len(tectonicElevations) < 2 {
		return 0.0
	}
	
	// Calculate local elevation gradients
	totalGradient := 0.0
	gradientCount := 0
	
	for i := range tectonicElevations {
		neighbors := findNearestNeighbors(i, icosphereSites, planetRadius, 50000.0) // 50 km search
		
		for _, neighborIdx := range neighbors {
			if neighborIdx < len(tectonicElevations) {
				distance := calculateSphericalDistance(icosphereSites[i], icosphereSites[neighborIdx], planetRadius)
				elevationDiff := math.Abs(tectonicElevations[i] - tectonicElevations[neighborIdx])
				
				if distance > 0 {
					gradient := elevationDiff / distance
					totalGradient += gradient
					gradientCount++
				}
			}
		}
	}
	
	if gradientCount > 0 {
		return totalGradient / float64(gradientCount)
	}
	
	return 0.0
}

