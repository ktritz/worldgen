package elevation

// base_elevation.go - Base elevation assignment based on plate types and basic geological principles

import (
	"math"
	"math/rand"
	"worldgen/landgen/tectonics"
)

// GenerateBaseElevations calculates base elevations for all sites based on plate types
func GenerateBaseElevations(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64 {
	baseElevations := make([]float64, len(icosphereSites))
	
	for siteIdx := range icosphereSites {
		siteID := int32(siteIdx)
		plateID := tectonicData.SitePlateIDs[siteID]
		
		// Find the plate for this site
		plate := findPlateByID(tectonicData.Plates, plateID)
		if plate == nil {
			baseElevations[siteIdx] = 0.0 // Default sea level
			continue
		}
		
		baseElevations[siteIdx] = calculatePlateTypeElevation(plate, siteID, params)
	}
	
	return baseElevations
}

// calculatePlateTypeElevation determines base elevation based on plate type and characteristics
func calculatePlateTypeElevation(plate *TectonicPlate, siteID int32, params ElevationParameters) float64 {
	// Create site-specific random generator for consistent results
	rng := rand.New(rand.NewSource(params.ElevationSeed + int64(siteID)))
	
	switch plate.PlateType {
	case tectonics.ContinentalPlate:
		return generateContinentalElevation(plate, rng, params)
	case tectonics.OceanicPlate:
		return generateOceanicElevation(plate, rng, params)
	default:
		return 0.0 // Default to sea level
	}
}

// generateContinentalElevation creates realistic continental elevation distribution
func generateContinentalElevation(plate *TectonicPlate, rng *rand.Rand, params ElevationParameters) float64 {
	// Continental plates have diverse topography
	// Earth continental elevation: Mean ~840m, range 0-8848m (sea level to Everest)
	
	// Base continental elevation with area dependency
	// Larger continental plates tend to have higher mean elevations
	areaFactor := math.Min(1.5, math.Sqrt(plate.Area/0.1)) // Larger plates -> higher elevation

	baseContinentalElevation := 1600.0 * areaFactor // 1600-2400m base (increased to balance realistic noise levels)

	// Add variation for continental diversity
	// Continental areas have wide elevation ranges
	variationRange := 800.0 * areaFactor // 800-1200m variation
	variation := (rng.Float64() - 0.5) * variationRange
	
	elevation := baseContinentalElevation + variation
	
	// Ensure continental areas are generally above sea level
	// But allow some coastal lowlands and inland seas
	if elevation < -200.0 {
		elevation = -200.0 + rng.Float64()*400.0 // -200 to +200m for coastal areas
	}
	
	return elevation
}

// generateOceanicElevation creates realistic oceanic elevation distribution  
func generateOceanicElevation(plate *TectonicPlate, rng *rand.Rand, params ElevationParameters) float64 {
	// Oceanic plates are predominantly below sea level
	// Earth oceanic depth: Mean ~3800m, range 0 to -11034m (Mariana Trench)

	// Base oceanic depth (calibrated to achieve Earth-like land/ocean ratio)
	baseOceanicDepth := -2650.0 // Final calibration after fixing ocean-ocean convergent boundaries to create realistic trenches
	
	// Variation for abyssal plains, seamounts, etc.
	variationRange := 1500.0 // ±1500m variation
	variation := (rng.Float64() - 0.5) * variationRange
	
	depth := baseOceanicDepth + variation
	
	// Clamp to reasonable oceanic range
	if depth > 0.0 {
		// Very rare oceanic areas above sea level (volcanic islands will be added separately)
		depth = -100.0 - rng.Float64()*500.0 // -100 to -600m for shallow seas
	}
	if depth < -6000.0 {
		depth = -6000.0 // Deep abyssal plains limit (trenches added separately)
	}
	
	return depth
}

// AdjustElevationForPlateAge modifies base elevation based on plate age and thermal evolution
func AdjustElevationForPlateAge(baseElevations []float64, icosphereSites []Vector3D, 
	tectonicData *TectonicsData, params ElevationParameters) []float64 {
	
	adjustedElevations := make([]float64, len(baseElevations))
	copy(adjustedElevations, baseElevations)
	
	// Only apply age adjustments if seafloor ages are available
	if len(tectonicData.SeafloorAges) != len(icosphereSites) {
		return adjustedElevations // No age data available
	}
	
	for siteIdx := range icosphereSites {
		siteID := int32(siteIdx)
		plateID := tectonicData.SitePlateIDs[siteID]
		
		plate := findPlateByID(tectonicData.Plates, plateID)
		if plate == nil || plate.PlateType != tectonics.OceanicPlate {
			continue // Only adjust oceanic plates
		}
		
		// Get seafloor age for this site
		age := tectonicData.SeafloorAges[siteIdx]
		
		// Apply thermal subsidence for oceanic crust
		ageAdjustment := calculateThermalSubsidence(age, params.SeafloorModel)
		adjustedElevations[siteIdx] += ageAdjustment
	}
	
	return adjustedElevations
}

// calculateThermalSubsidence computes thermal subsidence based on crustal age
func calculateThermalSubsidence(age float64, model SeafloorAgeModel) float64 {
	if age <= 0 {
		return 0.0
	}
	
	// Use the seafloor age model for thermal subsidence
	// This returns depth, so we need the relative change from reference
	ageDepth := model.GetSeafloorDepth(age)
	referenceDepth := model.GetSeafloorDepth(0.0)
	
	// Return the additional depth due to age (negative value = deeper)
	return ageDepth - referenceDepth
}

// ClassifyTerrainType determines terrain type based on elevation and plate context
func ClassifyTerrainType(elevation float64, plate *TectonicPlate, distanceToRidge, distanceToTrench float64) TerrainType {
	if plate == nil {
		if elevation > 0 {
			return ContinentalLowland
		} else {
			return AbyssalPlain
		}
	}
	
	switch plate.PlateType {
	case tectonics.ContinentalPlate:
		return classifyContinentalTerrain(elevation)
	case tectonics.OceanicPlate:
		return classifyOceanicTerrain(elevation, distanceToRidge, distanceToTrench)
	default:
		return AbyssalPlain
	}
}

// classifyContinentalTerrain categorizes continental terrain based on elevation
func classifyContinentalTerrain(elevation float64) TerrainType {
	switch {
	case elevation < -100.0:
		return ContinentalShelf // Shallow coastal waters
	case elevation < 200.0:
		return ContinentalLowland // Coastal plains and lowlands
	case elevation < 1000.0:
		return ContinentalHighland // Hills and plateaus
	default:
		return MountainRange // High mountains
	}
}

// classifyOceanicTerrain categorizes oceanic terrain based on elevation and context
func classifyOceanicTerrain(elevation float64, distanceToRidge, distanceToTrench float64) TerrainType {
	// Consider proximity to ridges and trenches
	isNearRidge := distanceToRidge < 100.0 // Within 100km of ridge
	isNearTrench := distanceToTrench < 50.0 // Within 50km of trench
	
	switch {
	case elevation > 0.0:
		return VolcanicIsland // Above sea level in oceanic setting
	case isNearTrench && elevation < -6000.0:
		return OceanTrench // Deep trench areas
	case isNearRidge && elevation > -3000.0:
		return MidOceanRidgeType // Ridge topography
	case elevation > -200.0:
		return ContinentalShelf // Shallow ocean areas
	case elevation > -1000.0:
		return ContinentalSlope // Slope to deep ocean
	default:
		return AbyssalPlain // Deep ocean floor
	}
}

// CalculateIsostasy applies isostatic adjustment based on crustal loading
func CalculateIsostasy(elevations []float64, plates []TectonicPlate, siteIDs []int32, params ElevationParameters) []float64 {
	if params.CrustalDensity <= 0 || params.MantleDensity <= 0 {
		return elevations // No isostatic adjustment if densities not set
	}
	
	adjustedElevations := make([]float64, len(elevations))
	copy(adjustedElevations, elevations)
	
	// Simple isostatic adjustment based on Archimedes' principle
	// Higher elevations create crustal loading that depresses the crust
	densityRatio := params.CrustalDensity / params.MantleDensity // ~0.82 for Earth
	
	for i, elevation := range elevations {
		if i >= len(siteIDs) {
			continue
		}
		
		plateID := siteIDs[i]
		plate := findPlateByID(plates, plateID)
		
		if plate == nil {
			continue
		}
		
		// Apply isostatic adjustment
		// Positive elevations (mountains) cause subsidence
		// Negative elevations (ocean basins) cause uplift
		isostaticAdjustment := -elevation * densityRatio * 0.1 // 10% effect
		
		adjustedElevations[i] += isostaticAdjustment
	}
	
	return adjustedElevations
}

// ValidateBaseElevations performs validation checks on base elevation distribution
func ValidateBaseElevations(baseElevations []float64, tectonicData *TectonicsData) (BaseElevationMetrics, []string) {
	var metrics BaseElevationMetrics
	var warnings []string
	
	if len(baseElevations) == 0 {
		warnings = append(warnings, "No base elevations generated")
		return metrics, warnings
	}
	
	// Calculate basic statistics
	var continentalElevations, oceanicElevations []float64
	
	for siteIdx, elevation := range baseElevations {
		siteID := int32(siteIdx)
		if siteID >= int32(len(tectonicData.SitePlateIDs)) {
			continue
		}
		
		plateID := tectonicData.SitePlateIDs[siteID]
		plate := findPlateByID(tectonicData.Plates, plateID)
		
		if plate == nil {
			continue
		}
		
		if plate.PlateType == tectonics.ContinentalPlate {
			continentalElevations = append(continentalElevations, elevation)
		} else {
			oceanicElevations = append(oceanicElevations, elevation)
		}
	}
	
	// Calculate metrics
	if len(continentalElevations) > 0 {
		metrics.MeanContinentalElevation = calculateMean(continentalElevations)
		metrics.ContinentalElevationRange = calculateRange(continentalElevations)
	}
	
	if len(oceanicElevations) > 0 {
		metrics.MeanOceanicDepth = calculateMean(oceanicElevations)
		metrics.OceanicDepthRange = calculateRange(oceanicElevations)
	}
	
	metrics.ContinentalSiteCount = len(continentalElevations)
	metrics.OceanicSiteCount = len(oceanicElevations)
	
	// Validation checks
	// Earth values: Continental mean ~840m, Oceanic mean ~-3800m
	if metrics.MeanContinentalElevation < 0 {
		warnings = append(warnings, "Mean continental elevation is below sea level")
	}
	if metrics.MeanContinentalElevation > 2000 {
		warnings = append(warnings, "Mean continental elevation is very high compared to Earth (~840m)")
	}
	
	if metrics.MeanOceanicDepth > -1000 {
		warnings = append(warnings, "Mean oceanic depth is very shallow compared to Earth (~-3800m)")
	}
	if metrics.MeanOceanicDepth < -6000 {
		warnings = append(warnings, "Mean oceanic depth is very deep")
	}
	
	return metrics, warnings
}

// BaseElevationMetrics contains statistics about base elevation distribution
type BaseElevationMetrics struct {
	MeanContinentalElevation float64 // Mean continental elevation (m)
	MeanOceanicDepth         float64 // Mean oceanic depth (m)
	ContinentalElevationRange float64 // Range of continental elevations (m)
	OceanicDepthRange        float64 // Range of oceanic depths (m)
	ContinentalSiteCount     int     // Number of continental sites
	OceanicSiteCount         int     // Number of oceanic sites
}

// Helper functions are now in utilities.go