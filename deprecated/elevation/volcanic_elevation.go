package elevation

// volcanic_elevation.go - Volcanic elevation effects from hotspots and volcanic features

import (
	"math"
	"worldgen/landgen/tectonics"
)

// GenerateVolcanicElevations calculates elevation contributions from volcanic features
func GenerateVolcanicElevations(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64 {
	volcanicElevations := make([]float64, len(icosphereSites))
	
	// Process hotspot thermal effects
	if len(tectonicData.Hotspots) > 0 {
		addHotspotThermalEffects(volcanicElevations, icosphereSites, tectonicData.Hotspots, params)
	}
	
	// Process individual volcanic features
	if len(tectonicData.VolcanicFeatures) > 0 {
		addVolcanicFeatureEffects(volcanicElevations, icosphereSites, tectonicData.VolcanicFeatures, params)
	}
	
	// Process volcanic arcs from subduction zones
	addVolcanicArcEffects(volcanicElevations, icosphereSites, tectonicData, params)
	
	return volcanicElevations
}

// addHotspotThermalEffects applies thermal uplift around active hotspots
func addHotspotThermalEffects(elevations []float64, sites []Vector3D, hotspots []Hotspot, params ElevationParameters) {
	for siteIdx, site := range sites {
		thermalUplift := 0.0
		
		for _, hotspot := range hotspots {
			if !hotspot.IsActive {
				continue // Only active hotspots cause thermal uplift
			}
			
			// Calculate distance from site to hotspot
			distance := calculateSphericalDistance(site, hotspot.Position, params.PlanetRadius)
			
			// Apply thermal effect if within influence radius
			if distance < params.HotspotInfluenceRadiusAbs {
				uplift := calculateHotspotThermalUplift(distance, hotspot, params)
				thermalUplift = math.Max(thermalUplift, uplift) // Take maximum effect
			}
		}
		
		elevations[siteIdx] += thermalUplift
	}
}

// calculateHotspotThermalUplift computes thermal uplift from a hotspot
func calculateHotspotThermalUplift(distance float64, hotspot Hotspot, params ElevationParameters) float64 {
	if distance >= params.HotspotInfluenceRadiusAbs {
		return 0.0
	}
	
	// Calculate distance-based falloff
	normalizedDistance := distance / params.HotspotInfluenceRadiusAbs
	falloff := math.Exp(-normalizedDistance * 3.0) // Exponential decay
	
	// Base thermal uplift scaled by hotspot intensity
	// Earth examples: Hawaii ~1000m, Yellowstone ~2000m thermal uplift
	baseThermalUplift := 800.0 * hotspot.Intensity * params.VolcanicMultiplier
	
	return baseThermalUplift * falloff
}

// addVolcanicFeatureEffects applies elevation from individual volcanic features
func addVolcanicFeatureEffects(elevations []float64, sites []Vector3D, features []VolcanicFeature, params ElevationParameters) {
	for siteIdx, site := range sites {
		maxVolcanicElevation := 0.0
		
		for _, feature := range features {
			// Calculate distance from site to volcanic feature
			distance := calculateSphericalDistance(site, feature.Position, params.PlanetRadius)
			
			// Check if site is within the volcanic feature's influence
			featureInfluenceRadius := feature.Diameter / 2.0 * 1000.0 // Convert km to meters
			
			if distance < featureInfluenceRadius {
				featureElevation := calculateVolcanicFeatureElevation(distance, feature, params)
				maxVolcanicElevation = math.Max(maxVolcanicElevation, featureElevation)
			}
		}
		
		elevations[siteIdx] += maxVolcanicElevation
	}
}

// calculateVolcanicFeatureElevation computes elevation contribution from a volcanic feature
func calculateVolcanicFeatureElevation(distance float64, feature VolcanicFeature, params ElevationParameters) float64 {
	featureRadius := feature.Diameter / 2.0 * 1000.0 // Convert km to meters
	
	if distance >= featureRadius {
		return 0.0
	}
	
	// Calculate volcanic profile based on feature type
	profile := calculateVolcanicProfile(distance, featureRadius, feature.FeatureType)
	
	// Scale by feature height and age
	ageDecay := calculateVolcanicAgeDecay(feature.FormationAge)
	effectiveHeight := feature.Height * params.VolcanicMultiplier * ageDecay
	
	return effectiveHeight * profile
}

// calculateVolcanicProfile determines the shape of volcanic elevation profile
func calculateVolcanicProfile(distance, radius float64, featureType tectonics.VolcanicType) float64 {
	normalizedDistance := distance / radius
	
	switch featureType {
	case tectonics.ShieldVolcano:
		// Shield volcanoes have gentle, broad profiles
		return math.Max(0, 1.0-normalizedDistance*normalizedDistance) // Parabolic profile
		
	case tectonics.VolcanicIsland:
		// Volcanic islands have moderate steepness
		return math.Max(0, math.Cos(normalizedDistance*math.Pi/2))
		
	case tectonics.Seamount:
		// Seamounts have steep-sided profiles
		if normalizedDistance < 0.7 {
			return 1.0 - normalizedDistance*0.5 // Flat top, steep sides
		} else {
			return math.Max(0, (1.0-normalizedDistance)/0.3)
		}
		
	case tectonics.ContinentalArc:
		// Continental arc volcanoes have steep profiles
		return math.Max(0, math.Exp(-normalizedDistance*4.0))
		
	case tectonics.VolcanicArc:
		// Oceanic arc volcanoes have moderate profiles
		return math.Max(0, math.Exp(-normalizedDistance*2.5))
		
	case tectonics.VolcanicPlateau:
		// Volcanic plateaus are broad and flat
		if normalizedDistance < 0.8 {
			return 0.9 + 0.1*math.Cos(normalizedDistance*math.Pi*2) // Flat with slight variation
		} else {
			return math.Max(0, (1.0-normalizedDistance)/0.2) // Edge falloff
		}
		
	default:
		// Default conical profile
		return math.Max(0, 1.0-normalizedDistance)
	}
}

// calculateVolcanicAgeDecay applies age-based erosion to volcanic features
func calculateVolcanicAgeDecay(age float64) float64 {
	// Volcanic features erode and subside over time
	// Half-life of ~10 Myr for volcanic elevation
	halfLife := 10.0 // Million years
	return math.Exp(-age * math.Ln2 / halfLife)
}

// addVolcanicArcEffects applies elevation from subduction zone volcanic arcs
func addVolcanicArcEffects(elevations []float64, sites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) {
	// This would require subduction zone data from the tectonics module
	// For now, we'll use a simplified approach based on convergent boundaries
	
	for siteIdx := range sites {
		siteID := int32(siteIdx)
		
		// Check if this site is on a convergent boundary
		if boundaryType, exists := tectonicData.SiteBoundaryTypes[siteID]; exists {
			if boundaryType == tectonics.Convergent {
				// Check if this is a likely volcanic arc location
				distToBoundary := tectonicData.SiteDistancesToBoundary[siteIdx] * params.PlanetRadius
				
				// Volcanic arcs typically form 100-300 km from trenches
				if distToBoundary > 50000 && distToBoundary < 400000 { // 50-400 km
					plateID := tectonicData.SitePlateIDs[siteID]
					plate := findPlateByID(tectonicData.Plates, plateID)
					
					// Arc volcanism is more prominent on overriding plates
					if plate != nil {
						arcElevation := calculateVolcanicArcElevation(distToBoundary, plate, params)
						elevations[siteIdx] += arcElevation
					}
				}
			}
		}
	}
}

// calculateVolcanicArcElevation computes elevation from volcanic arc formation
func calculateVolcanicArcElevation(distToBoundary float64, plate *TectonicPlate, params ElevationParameters) float64 {
	// Optimal arc distance is ~150-200 km from trench
	optimalDistance := 175000.0 // 175 km in meters
	distanceFromOptimal := math.Abs(distToBoundary - optimalDistance)
	
	// Arc effect decreases with distance from optimal
	if distanceFromOptimal > 150000.0 { // 150 km tolerance
		return 0.0
	}
	
	// Distance-based falloff
	falloff := math.Exp(-distanceFromOptimal / 50000.0) // 50 km decay
	
	// Base arc elevation depends on plate type
	baseArcElevation := params.ArcElevation
	if plate.PlateType == tectonics.OceanicPlate {
		// Oceanic arcs (island arcs) are typically lower
		baseArcElevation *= 0.7
	}
	
	return baseArcElevation * falloff
}


// ValidateVolcanicElevations performs validation on volcanic elevation effects
func ValidateVolcanicElevations(volcanicElevations []float64, tectonicData *TectonicsData) (VolcanicElevationMetrics, []string) {
	var metrics VolcanicElevationMetrics
	var warnings []string
	
	if len(volcanicElevations) == 0 {
		warnings = append(warnings, "No volcanic elevations generated")
		return metrics, warnings
	}
	
	// Count sites with volcanic effects
	affectedSites := 0
	maxVolcanicEffect := 0.0
	totalVolcanicEffect := 0.0
	
	for _, elevation := range volcanicElevations {
		if elevation > 10.0 { // Minimum significant volcanic effect
			affectedSites++
			totalVolcanicEffect += elevation
			if elevation > maxVolcanicEffect {
				maxVolcanicEffect = elevation
			}
		}
	}
	
	metrics.VolcanicAffectedSites = affectedSites
	metrics.MaxVolcanicElevation = maxVolcanicEffect
	
	if affectedSites > 0 {
		metrics.MeanVolcanicElevation = totalVolcanicEffect / float64(affectedSites)
		metrics.VolcanicCoveragePercent = float64(affectedSites) / float64(len(volcanicElevations)) * 100.0
	}
	
	// Validation checks
	if len(tectonicData.Hotspots) > 0 && affectedSites == 0 {
		warnings = append(warnings, "No volcanic elevation effects despite hotspots being present")
	}
	
	if maxVolcanicEffect > 10000.0 {
		warnings = append(warnings, "Maximum volcanic elevation is very high (>10km)")
	}
	
	if metrics.VolcanicCoveragePercent > 20.0 {
		warnings = append(warnings, "Volcanic coverage is very high (>20% of surface)")
	}
	
	return metrics, warnings
}

// VolcanicElevationMetrics contains statistics about volcanic elevation effects
type VolcanicElevationMetrics struct {
	VolcanicAffectedSites   int     // Number of sites with significant volcanic effects
	MaxVolcanicElevation    float64 // Maximum volcanic elevation effect (m)
	MeanVolcanicElevation   float64 // Mean volcanic elevation where present (m)
	VolcanicCoveragePercent float64 // Percentage of surface affected by volcanism
}

// GetVolcanicFeaturesByType returns volcanic features grouped by type
func GetVolcanicFeaturesByType(features []VolcanicFeature) map[tectonics.VolcanicType][]VolcanicFeature {
	featuresByType := make(map[tectonics.VolcanicType][]VolcanicFeature)
	
	for _, feature := range features {
		featuresByType[feature.FeatureType] = append(featuresByType[feature.FeatureType], feature)
	}
	
	return featuresByType
}

// CalculateVolcanicHazardZones identifies areas at risk from volcanic activity
func CalculateVolcanicHazardZones(sites []Vector3D, features []VolcanicFeature, planetRadius float64) []float64 {
	hazardLevels := make([]float64, len(sites))
	
	for siteIdx, site := range sites {
		maxHazard := 0.0
		
		for _, feature := range features {
			if !feature.IsActive {
				continue // Only consider active volcanic features
			}
			
			distance := calculateSphericalDistance(site, feature.Position, planetRadius)
			
			// Hazard decreases with distance
			hazardRadius := feature.Diameter * 1000.0 * 2.0 // 2x feature diameter in meters
			
			if distance < hazardRadius {
				hazard := calculateVolcanicHazard(distance, hazardRadius, feature)
				if hazard > maxHazard {
					maxHazard = hazard
				}
			}
		}
		
		hazardLevels[siteIdx] = maxHazard
	}
	
	return hazardLevels
}

// calculateVolcanicHazard determines hazard level based on feature and distance
func calculateVolcanicHazard(distance, hazardRadius float64, feature VolcanicFeature) float64 {
	// Normalize distance
	normalizedDistance := distance / hazardRadius
	
	// Base hazard depends on feature type
	var baseHazard float64
	switch feature.FeatureType {
	case tectonics.VolcanicIsland:
		baseHazard = 0.9 // Very high hazard
	case tectonics.ShieldVolcano:
		baseHazard = 0.8 // High hazard
	case tectonics.ContinentalArc, tectonics.VolcanicArc:
		baseHazard = 0.7 // High hazard
	case tectonics.VolcanicPlateau:
		baseHazard = 0.5 // Moderate hazard
	case tectonics.Seamount:
		baseHazard = 0.3 // Lower hazard (underwater)
	default:
		baseHazard = 0.5
	}
	
	// Distance falloff
	distanceFactor := math.Max(0, 1.0-normalizedDistance)
	
	// Recent features are more hazardous
	ageFactor := math.Exp(-feature.FormationAge / 5.0) // 5 Myr half-life
	
	return baseHazard * distanceFactor * ageFactor
}