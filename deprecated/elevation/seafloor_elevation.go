package elevation

// seafloor_elevation.go - Seafloor elevation generation based on age-depth relationships

import (
	"math"
	"worldgen/landgen/tectonics"
)

// GenerateSeafloorElevations calculates elevation contributions from seafloor age-depth relationships
func GenerateSeafloorElevations(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64 {
	seafloorElevations := make([]float64, len(icosphereSites))
	
	// Only apply seafloor age-depth if data is available
	if len(tectonicData.SeafloorAges) != len(icosphereSites) {
		return seafloorElevations // Return zeros if no age data
	}
	
	for siteIdx := range icosphereSites {
		siteID := int32(siteIdx)
		plateID := tectonicData.SitePlateIDs[siteID]
		
		// Find the plate for this site
		plate := findPlateByID(tectonicData.Plates, plateID)
		if plate == nil {
			continue
		}
		
		// Apply age-depth relationships primarily to oceanic plates
		if plate.PlateType == tectonics.OceanicPlate {
			age := tectonicData.SeafloorAges[siteIdx]
			seafloorElevations[siteIdx] = calculateAgeDepthElevation(age, params.SeafloorModel)
		} else {
			// Continental plates can have some age-related subsidence
			age := tectonicData.SeafloorAges[siteIdx]
			seafloorElevations[siteIdx] = calculateContinentalAgeEffect(age, plate, params)
		}
	}
	
	return seafloorElevations
}

// calculateAgeDepthElevation computes elevation from seafloor age using empirical models
func calculateAgeDepthElevation(age float64, model SeafloorAgeModel) float64 {
	if age <= 0 {
		return 0.0 // No age data or ridge crest
	}
	
	// Use the seafloor age model to get depth
	depth := model.GetSeafloorDepth(age)
	
	// Convert depth to elevation contribution relative to reference
	// At age 0 (ridge crest), elevation should be maximum
	ridgeCrestDepth := model.GetSeafloorDepth(0.0)
	
	// Return elevation relative to ridge crest (negative = deeper)
	return ridgeCrestDepth - depth
}

// calculateContinentalAgeEffect applies age-related effects to continental plates
func calculateContinentalAgeEffect(age float64, plate *TectonicPlate, params ElevationParameters) float64 {
	if age <= 0 {
		return 0.0
	}
	
	// Continental crust undergoes thermal subsidence but less than oceanic
	// Also affected by erosion and sedimentation over time
	
	// Thermal subsidence for continental lithosphere (much smaller than oceanic)
	thermalSubsidence := -50.0 * math.Sqrt(age) // ~50m per sqrt(Myr)
	
	// Limit thermal subsidence for very old continental crust
	if thermalSubsidence < -800.0 {
		thermalSubsidence = -800.0
	}
	
	// Add some variation based on plate size (larger plates have more stable interiors)
	stabilityFactor := math.Min(1.5, math.Sqrt(plate.Area/0.1))
	thermalSubsidence /= stabilityFactor
	
	return thermalSubsidence
}

// CalculateSeafloorSpreadingEffects applies elevation effects from active spreading centers
func CalculateSeafloorSpreadingEffects(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64 {
	spreadingEffects := make([]float64, len(icosphereSites))
	
	if len(tectonicData.MidOceanRidges) == 0 {
		return spreadingEffects // No ridges to process
	}
	
	for siteIdx, site := range icosphereSites {
		maxSpreadingEffect := 0.0
		
		for _, ridge := range tectonicData.MidOceanRidges {
			if !true {
				continue
			}
			
			// Calculate distance to ridge axis
			distanceToRidge := calculateDistanceToRidgeAxis(site, ridge, params.PlanetRadius)
			
			// Apply spreading effects if within influence zone
			if distanceToRidge < params.RidgeInfluenceDistAbs {
				spreadingEffect := calculateSpreadingElevationEffect(distanceToRidge, ridge, params)
				if spreadingEffect > maxSpreadingEffect {
					maxSpreadingEffect = spreadingEffect
				}
			}
		}
		
		spreadingEffects[siteIdx] = maxSpreadingEffect
	}
	
	return spreadingEffects
}

// calculateDistanceToRidgeAxis computes the shortest distance from a point to a ridge axis
func calculateDistanceToRidgeAxis(site Vector3D, ridge MidOceanRidge, planetRadius float64) float64 {
	// For now, use simple distance to ridge center
	// In a more sophisticated version, this would calculate distance to the ridge line/axis
	return calculateSphericalDistance(site, calculateRidgeCenter(ridge), planetRadius)
}

// calculateSpreadingElevationEffect computes elevation effects from seafloor spreading
func calculateSpreadingElevationEffect(distance float64, ridge MidOceanRidge, params ElevationParameters) float64 {
	if distance >= params.RidgeInfluenceDistAbs {
		return 0.0
	}
	
	// Ridge axis elevation effect decreases with distance
	normalizedDistance := distance / params.RidgeInfluenceDistAbs
	
	// Exponential decay with distance from ridge axis
	falloff := math.Exp(-normalizedDistance * 2.5)
	
	// Base ridge elevation scaled by spreading rate
	// Faster spreading ridges tend to be higher and broader
	spreadingRateMultiplier := math.Min(2.0, ridge.SpreadingRate/20.0) // Normalize by 20 mm/yr
	baseRidgeElevation := params.RidgeElevation * spreadingRateMultiplier
	
	return baseRidgeElevation * falloff
}

// ValidateSeafloorElevations performs validation on seafloor elevation effects
func ValidateSeafloorElevations(seafloorElevations []float64, tectonicData *TectonicsData) (SeafloorElevationMetrics, []string) {
	var metrics SeafloorElevationMetrics
	var warnings []string
	
	if len(seafloorElevations) == 0 {
		warnings = append(warnings, "No seafloor elevations generated")
		return metrics, warnings
	}
	
	// Separate oceanic and continental effects
	var oceanicEffects, continentalEffects []float64
	
	for siteIdx, elevation := range seafloorElevations {
		if math.Abs(elevation) < 1.0 { // Skip negligible effects
			continue
		}
		
		siteID := int32(siteIdx)
		if siteID >= int32(len(tectonicData.SitePlateIDs)) {
			continue
		}
		
		plateID := tectonicData.SitePlateIDs[siteID]
		plate := findPlateByID(tectonicData.Plates, plateID)
		
		if plate == nil {
			continue
		}
		
		if plate.PlateType == tectonics.OceanicPlate {
			oceanicEffects = append(oceanicEffects, elevation)
		} else {
			continentalEffects = append(continentalEffects, elevation)
		}
	}
	
	// Calculate metrics
	if len(oceanicEffects) > 0 {
		metrics.MeanOceanicAgeEffect = calculateMean(oceanicEffects)
		metrics.OceanicAgeRange = calculateRange(oceanicEffects)
		metrics.OceanicSitesAffected = len(oceanicEffects)
	}
	
	if len(continentalEffects) > 0 {
		metrics.MeanContinentalAgeEffect = calculateMean(continentalEffects)
		metrics.ContinentalSitesAffected = len(continentalEffects)
	}
	
	// Calculate age statistics if available
	if len(tectonicData.SeafloorAges) == len(seafloorElevations) {
		ages := tectonicData.SeafloorAges
		metrics.MaxSeafloorAge = 0.0
		metrics.MeanSeafloorAge = 0.0
		ageSum := 0.0
		validAges := 0
		
		for _, age := range ages {
			if age > 0 {
				ageSum += age
				validAges++
				if age > metrics.MaxSeafloorAge {
					metrics.MaxSeafloorAge = age
				}
			}
		}
		
		if validAges > 0 {
			metrics.MeanSeafloorAge = ageSum / float64(validAges)
		}
	}
	
	// Validation checks
	if len(tectonicData.SeafloorAges) > 0 && len(oceanicEffects) == 0 {
		warnings = append(warnings, "No oceanic age effects despite seafloor age data being present")
	}
	
	if metrics.MaxSeafloorAge > 250.0 {
		warnings = append(warnings, "Maximum seafloor age is very old (>250 Myr), may exceed Earth observations")
	}
	
	if metrics.MeanOceanicAgeEffect > -500.0 {
		warnings = append(warnings, "Mean oceanic age effect is unexpectedly shallow")
	}
	
	if metrics.MeanOceanicAgeEffect < -3000.0 {
		warnings = append(warnings, "Mean oceanic age effect is very deep")
	}
	
	return metrics, warnings
}

// SeafloorElevationMetrics contains statistics about seafloor elevation effects
type SeafloorElevationMetrics struct {
	// Age-depth relationship metrics
	MeanOceanicAgeEffect      float64 // Mean elevation effect from oceanic age-depth (m)
	MeanContinentalAgeEffect  float64 // Mean elevation effect from continental age (m)
	OceanicAgeRange           float64 // Range of oceanic age effects (m)
	OceanicSitesAffected      int     // Number of oceanic sites with age effects
	ContinentalSitesAffected  int     // Number of continental sites with age effects
	
	// Age statistics
	MaxSeafloorAge            float64 // Maximum seafloor age (Myr)
	MeanSeafloorAge           float64 // Mean seafloor age (Myr)
	
	// Ridge statistics
	RidgeCount                int     // Number of active ridges
	MeanRidgeElevation        float64 // Mean ridge elevation effect (m)
	RidgeInfluenceArea        float64 // Percentage of surface influenced by ridges
}

// CalculateAgeDepthValidation compares age-depth relationships to Earth models
func CalculateAgeDepthValidation(seafloorAges []float64, elevations []float64, model SeafloorAgeModel) AgeDepthValidation {
	var validation AgeDepthValidation
	
	if len(seafloorAges) != len(elevations) || len(seafloorAges) == 0 {
		return validation
	}
	
	// Sample points for age-depth curve validation
	agePoints := []float64{0, 10, 20, 50, 100, 150, 200}
	
	for _, age := range agePoints {
		if age <= 0 {
			continue
		}
		
		// Find elevations near this age
		var elevationsNearAge []float64
		ageToleranceMyr := 5.0 // ±5 Myr tolerance
		
		for i, siteAge := range seafloorAges {
			if math.Abs(siteAge-age) < ageToleranceMyr {
				elevationsNearAge = append(elevationsNearAge, elevations[i])
			}
		}
		
		if len(elevationsNearAge) > 0 {
			observedMean := calculateMean(elevationsNearAge)
			expectedDepth := model.GetSeafloorDepth(age)
			
			validation.AgeDepthPoints = append(validation.AgeDepthPoints, AgeDepthPoint{
				Age:              age,
				ObservedElevation: observedMean,
				ExpectedDepth:    expectedDepth,
				SampleCount:      len(elevationsNearAge),
			})
		}
	}
	
	// Calculate overall fit quality
	if len(validation.AgeDepthPoints) > 0 {
		totalError := 0.0
		for _, point := range validation.AgeDepthPoints {
			error := math.Abs(point.ObservedElevation - point.ExpectedDepth)
			totalError += error
		}
		validation.MeanAbsoluteError = totalError / float64(len(validation.AgeDepthPoints))
		
		// Score: 1.0 for perfect fit, 0.0 for >1000m average error
		validation.AgeDepthFitScore = math.Max(0.0, 1.0-validation.MeanAbsoluteError/1000.0)
	}
	
	return validation
}

// AgeDepthValidation contains validation data for age-depth relationships
type AgeDepthValidation struct {
	AgeDepthPoints     []AgeDepthPoint // Age-depth curve points
	MeanAbsoluteError  float64         // Mean absolute error between observed and expected (m)
	AgeDepthFitScore   float64         // Quality of fit to Earth age-depth model (0-1)
}

// AgeDepthPoint represents a point on the age-depth validation curve
type AgeDepthPoint struct {
	Age               float64 // Seafloor age (Myr)
	ObservedElevation float64 // Observed elevation at this age (m)
	ExpectedDepth     float64 // Expected depth from model (m)
	SampleCount       int     // Number of sites averaged for this point
}

// GetSeafloorSpreadingRate estimates spreading rate from age gradients
func GetSeafloorSpreadingRate(icosphereSites []Vector3D, seafloorAges []float64, planetRadius float64) []float64 {
	spreadingRates := make([]float64, len(icosphereSites))
	
	if len(seafloorAges) != len(icosphereSites) {
		return spreadingRates // No age data available
	}
	
	// Build spatial index for efficient neighbor finding
	// This is a simplified version - a full implementation would use a proper spatial index
	for i := range icosphereSites {
		if seafloorAges[i] <= 0 {
			continue // No age data for this site
		}
		
		// Find nearby sites and calculate age gradients
		nearbyAgeGradients := []float64{}
		
		for j := range icosphereSites {
			if i == j || seafloorAges[j] <= 0 {
				continue
			}
			
			distance := calculateSphericalDistance(icosphereSites[i], icosphereSites[j], planetRadius)
			
			// Only consider nearby sites (within 100 km)
			if distance < 100000.0 {
				ageDifference := math.Abs(seafloorAges[i] - seafloorAges[j])
				if ageDifference > 0.1 { // Minimum age difference threshold
					// Spreading rate = distance / (2 * age difference)
					// Factor of 2 because spreading occurs on both sides of ridge
					rate := distance / (2.0 * ageDifference * 1e6) // Convert Myr to years
					rate *= 1000.0 // Convert m/yr to mm/yr
					
					if rate < 200.0 { // Reasonable spreading rate limit
						nearbyAgeGradients = append(nearbyAgeGradients, rate)
					}
				}
			}
		}
		
		// Use median of nearby gradients to estimate local spreading rate
		if len(nearbyAgeGradients) > 0 {
			spreadingRates[i] = calculateMedian(nearbyAgeGradients)
		}
	}
	
	return spreadingRates
}

