package elevation

// erosion_modeling.go - Age-based erosion and denudation modeling

import (
	"math"
	"worldgen/landgen/tectonics"
)

// GenerateErosionEffects calculates elevation modifications from erosion and weathering
func GenerateErosionEffects(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64 {
	erosionEffects := make([]float64, len(icosphereSites))

	for siteIdx := range icosphereSites {
		siteID := int32(siteIdx)
		plateID := tectonicData.SitePlateIDs[siteID]
		plate := findPlateByID(tectonicData.Plates, plateID)

		if plate == nil {
			continue
		}

		// Get crustal age for erosion calculation
		age := getCrustalAge(siteID, tectonicData)

		// Calculate erosion based on plate type, age, and tectonic activity
		erosionEffect := calculateAgeBasedErosion(age, plate, siteID, tectonicData, params)
		erosionEffects[siteIdx] = erosionEffect
	}

	return erosionEffects
}

// getCrustalAge determines the age of crust at a site
func getCrustalAge(siteID int32, tectonicData *TectonicsData) float64 {
	// Try seafloor age first
	if int(siteID) < len(tectonicData.SeafloorAges) && tectonicData.SeafloorAges[siteID] > 0 {
		return tectonicData.SeafloorAges[siteID]
	}
	
	// For continental crust, use plate age or estimate
	plateID := tectonicData.SitePlateIDs[siteID]
	plate := findPlateByID(tectonicData.Plates, plateID)
	
	if plate != nil && plate.PlateType == tectonics.ContinentalPlate {
		// Continental crust is typically much older than seafloor
		// Use a reasonable estimate based on plate characteristics
		return estimateContinentalCrustAge(plate)
	}
	
	return 0.0 // Unknown age
}

// estimateContinentalCrustAge estimates continental crust age
func estimateContinentalCrustAge(plate *TectonicPlate) float64 {
	// Continental crust age varies widely (0.1 to 4.0 Gyr)
	// Use plate area as a proxy for stability/age
	// Larger plates tend to have older, more stable cores
	
	baseAge := 500.0 // 500 Myr base age
	
	// Larger plates have older cores
	areaFactor := math.Min(3.0, math.Sqrt(plate.Area/0.1)) // Normalize by ~10% of surface
	
	return baseAge * areaFactor
}

// calculateAgeBasedErosion computes erosion effects based on crustal age and tectonic activity
func calculateAgeBasedErosion(age float64, plate *TectonicPlate, siteID int32, tectonicData *TectonicsData, params ElevationParameters) float64 {
	if age <= 0 {
		return 0.0
	}

	// Continental crust erosion depends on proximity to active mountain building
	if plate.PlateType == tectonics.ContinentalPlate {
		// Get distance to nearest boundary
		distanceToBoundary := 1.0 // Normalized distance (0 = on boundary, 1 = far from boundary)
		if int(siteID) < len(tectonicData.SiteDistancesToBoundary) {
			rawDist := tectonicData.SiteDistancesToBoundary[siteID]
			// Normalize by planet radius (~6371 km)
			// Active mountain building within ~500 km of convergent boundaries
			distanceToBoundary = math.Min(1.0, rawDist/500000.0)
		}

		// Erosion rate depends on proximity to active tectonics
		// Near boundaries (active mountains): 20 m/Myr
		// Far from boundaries (stable cratons): 1 m/Myr
		minErosionRate := 1.0  // Stable cratons (m/Myr)
		maxErosionRate := 20.0 // Active orogens (m/Myr)
		erosionRate := minErosionRate + (maxErosionRate-minErosionRate)*(1.0-distanceToBoundary)

		// Only apply erosion over geologically recent time
		effectiveAge := math.Min(age, params.MaxErosionAge)

		totalErosion := erosionRate * effectiveAge
		return -totalErosion
	}

	// Oceanic crust: Minimal erosion (submarine weathering only)
	if plate.PlateType == tectonics.OceanicPlate {
		effectiveAge := math.Min(age, 200.0) // Max 200 Myr for oceanic crust
		erosionRate := 5.0                   // meters per million years
		totalErosion := erosionRate * effectiveAge
		return -totalErosion
	}

	return 0.0
}

// calculateContinentalErosionRate determines erosion rate for continental crust
func calculateContinentalErosionRate(age float64, params ElevationParameters) float64 {
	// Continental erosion rate decreases with age as topography is reduced
	
	// Young continental crust (0-50 Myr): High erosion rates
	if age < 50.0 {
		return 0.08 // 80 mm/kyr - active mountain building
	}
	
	// Mature continental crust (50-200 Myr): Moderate erosion
	if age < 200.0 {
		return 0.05 // 50 mm/kyr
	}
	
	// Old continental crust (200-1000 Myr): Low erosion
	if age < 1000.0 {
		return 0.02 // 20 mm/kyr
	}
	
	// Very old cratonic crust (>1000 Myr): Very low erosion
	return 0.01 // 10 mm/kyr - stable cratons
}

// calculateOceanicErosionRate determines erosion rate for oceanic crust
func calculateOceanicErosionRate(age float64, params ElevationParameters) float64 {
	// Oceanic crust erosion mainly through chemical weathering
	// Rate decreases with age as fresh basalt weathers
	
	if age < 10.0 {
		return 0.05 // 50 mm/kyr - fresh basalt
	}
	
	if age < 50.0 {
		return 0.03 // 30 mm/kyr
	}
	
	// Older oceanic crust has lower erosion rates
	return 0.02 // 20 mm/kyr
}

// calculateClimateErosionModifier applies climate-related erosion effects
func calculateClimateErosionModifier(siteID int32, tectonicData *TectonicsData, params ElevationParameters) float64 {
	// Simplified climate model based on latitude
	// In reality, this would use detailed climate data
	
	// Get approximate latitude (simplified)
	plateID := tectonicData.SitePlateIDs[siteID]
	plate := findPlateByID(tectonicData.Plates, plateID)
	
	if plate == nil {
		return 1.0
	}
	
	// Use plate position as proxy for climate zone
	latitude := math.Abs(plate.Center.Z) // Z-component approximates latitude
	
	// Climate zones affect erosion rates
	if latitude < 0.3 { // Tropical (high erosion)
		return 1.5
	} else if latitude < 0.6 { // Temperate (moderate erosion)
		return 1.0
	} else { // Polar (low erosion)
		return 0.5
	}
}

// calculateElevationErosionModifier applies elevation-dependent erosion
func calculateElevationErosionModifier(siteID int32, tectonicData *TectonicsData, params ElevationParameters) float64 {
	// Higher elevations experience more erosion due to:
	// - Increased precipitation orographic effects
	// - Freeze-thaw cycles
	// - Steeper slopes and gravity
	
	// This would normally use actual elevation data
	// For now, use proximity to mountain-building zones as proxy
	
	boundaryType, exists := tectonicData.SiteBoundaryTypes[siteID]
	if exists && boundaryType == tectonics.Convergent {
		// Near convergent boundaries (mountains) - higher erosion
		return 1.3
	}
	
	// Default erosion rate
	return 1.0
}

// CalculateDifferentialErosion models differential erosion based on rock type
func CalculateDifferentialErosion(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64 {
	differentialErosion := make([]float64, len(icosphereSites))
	
	for siteIdx := range icosphereSites {
		siteID := int32(siteIdx)
		plateID := tectonicData.SitePlateIDs[siteID]
		plate := findPlateByID(tectonicData.Plates, plateID)
		
		if plate == nil {
			continue
		}
		
		// Determine rock type based on tectonic setting
		rockType := determineRockType(siteID, plate, tectonicData)
		
		// Apply rock-specific erosion resistance
		erosionResistance := getRockErosionResistance(rockType)
		
		// Base differential erosion effect
		baseEffect := -100.0 // 100m base differential
		
		differentialErosion[siteIdx] = baseEffect / erosionResistance
	}
	
	return differentialErosion
}

// determineRockType infers rock type from tectonic setting
func determineRockType(siteID int32, plate *TectonicPlate, tectonicData *TectonicsData) RockType {
	// Simplified rock type determination
	
	boundaryType, exists := tectonicData.SiteBoundaryTypes[siteID]
	
	switch plate.PlateType {
	case tectonics.ContinentalPlate:
		if exists && boundaryType == tectonics.Convergent {
			return Metamorphic // Mountain building creates metamorphic rocks
		}
		return Sedimentary // Most continental surfaces are sedimentary
		
	case tectonics.OceanicPlate:
		if exists && boundaryType == tectonics.Divergent {
			return Igneous // Fresh basalt at ridges
		}
		return Sedimentary // Marine sediments on older seafloor
		
	default:
		return Sedimentary
	}
}

// getRockErosionResistance returns relative erosion resistance of rock types
func getRockErosionResistance(rockType RockType) float64 {
	switch rockType {
	case Igneous:
		return 2.0 // Resistant to erosion
	case Metamorphic:
		return 1.5 // Moderately resistant
	case Sedimentary:
		return 1.0 // Baseline resistance
	default:
		return 1.0
	}
}

// CalculateGlacialErosion models erosion from glacial processes
func CalculateGlacialErosion(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64 {
	glacialErosion := make([]float64, len(icosphereSites))
	
	if !false {
		return glacialErosion
	}
	
	for siteIdx := range icosphereSites {
		siteID := int32(siteIdx)
		plateID := tectonicData.SitePlateIDs[siteID]
		plate := findPlateByID(tectonicData.Plates, plateID)
		
		if plate == nil {
			continue
		}
		
		// Only continental plates experience significant glacial erosion
		if plate.PlateType != tectonics.ContinentalPlate {
			continue
		}
		
		// Glacial erosion depends on latitude and elevation
		glacialSusceptibility := calculateGlacialSusceptibility(plate)
		
		if glacialSusceptibility > 0.1 {
			// Apply glacial erosion (very effective)
			erosionDepth := glacialSusceptibility * 1.0
			glacialErosion[siteIdx] = -erosionDepth
		}
	}
	
	return glacialErosion
}

// calculateGlacialSusceptibility determines likelihood of glacial erosion
func calculateGlacialSusceptibility(plate *TectonicPlate) float64 {
	// High latitudes more susceptible to glaciation
	latitude := math.Abs(plate.Center.Z)
	
	if latitude > 0.7 { // High latitude
		return 1.0
	} else if latitude > 0.5 { // Mid latitude
		return 0.3
	} else { // Low latitude
		return 0.0
	}
}

// ValidateErosionEffects performs validation on erosion modeling
func ValidateErosionEffects(erosionEffects []float64, tectonicData *TectonicsData) (ErosionMetrics, []string) {
	var metrics ErosionMetrics
	var warnings []string
	
	if len(erosionEffects) == 0 {
		warnings = append(warnings, "No erosion effects generated")
		return metrics, warnings
	}
	
	// Separate by plate type
	var continentalErosion, oceanicErosion []float64
	maxErosion := 0.0
	totalErosion := 0.0
	
	for siteIdx, erosion := range erosionEffects {
		if math.Abs(erosion) < 1.0 { // Skip negligible effects
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
		
		erosionMagnitude := math.Abs(erosion)
		if erosionMagnitude > maxErosion {
			maxErosion = erosionMagnitude
		}
		totalErosion += erosionMagnitude
		
		if plate.PlateType == tectonics.ContinentalPlate {
			continentalErosion = append(continentalErosion, erosionMagnitude)
		} else {
			oceanicErosion = append(oceanicErosion, erosionMagnitude)
		}
	}
	
	// Calculate metrics
	metrics.MaxErosionDepth = maxErosion
	metrics.ErosionAffectedSites = len(continentalErosion) + len(oceanicErosion)
	
	if len(continentalErosion) > 0 {
		metrics.MeanContinentalErosion = calculateMean(continentalErosion)
		metrics.ContinentalErosionSites = len(continentalErosion)
	}
	
	if len(oceanicErosion) > 0 {
		metrics.MeanOceanicErosion = calculateMean(oceanicErosion)
		metrics.OceanicErosionSites = len(oceanicErosion)
	}
	
	if metrics.ErosionAffectedSites > 0 {
		metrics.MeanErosionDepth = totalErosion / float64(metrics.ErosionAffectedSites)
	}
	
	// Validation checks
	if maxErosion > 10000.0 {
		warnings = append(warnings, "Maximum erosion depth is very high (>10km)")
	}
	
	if metrics.MeanContinentalErosion > 5000.0 {
		warnings = append(warnings, "Mean continental erosion is very high")
	}
	
	if metrics.ErosionAffectedSites == 0 {
		warnings = append(warnings, "No significant erosion effects found")
	}
	
	// Earth comparison
	if metrics.MeanContinentalErosion < 100.0 && len(continentalErosion) > 0 {
		warnings = append(warnings, "Mean continental erosion is low compared to Earth rates")
	}
	
	return metrics, warnings
}

// ErosionMetrics contains statistics about erosion effects
type ErosionMetrics struct {
	MaxErosionDepth         float64 // Maximum erosion depth (m)
	MeanErosionDepth        float64 // Mean erosion depth across all sites (m)
	MeanContinentalErosion  float64 // Mean erosion on continental plates (m)
	MeanOceanicErosion      float64 // Mean erosion on oceanic plates (m)
	ErosionAffectedSites    int     // Total sites with significant erosion
	ContinentalErosionSites int     // Continental sites with erosion
	OceanicErosionSites     int     // Oceanic sites with erosion
}

// RockType represents different rock types for erosion modeling
type RockType string

const (
	Igneous     RockType = "igneous"
	Sedimentary RockType = "sedimentary"
	Metamorphic RockType = "metamorphic"
)

// CalculateErosionRates estimates current erosion rates from topographic gradients
func CalculateErosionRates(elevations []float64, icosphereSites []Vector3D, planetRadius float64) []float64 {
	erosionRates := make([]float64, len(elevations))
	
	for i := range elevations {
		// Find local slope
		neighbors := findNearestNeighbors(i, icosphereSites, planetRadius, 25000.0) // 25 km
		
		if len(neighbors) == 0 {
			continue
		}
		
		maxGradient := 0.0
		for _, neighborIdx := range neighbors {
			if neighborIdx < len(elevations) {
				distance := calculateSphericalDistance(icosphereSites[i], icosphereSites[neighborIdx], planetRadius)
				elevationDiff := math.Abs(elevations[i] - elevations[neighborIdx])
				
				if distance > 0 {
					gradient := elevationDiff / distance
					if gradient > maxGradient {
						maxGradient = gradient
					}
				}
			}
		}
		
		// Convert gradient to erosion rate (mm/yr)
		// Steeper slopes erode faster
		erosionRates[i] = maxGradient * 100.0 // Simplified relationship
	}
	
	return erosionRates
}

// CalculateSedimentationEffects models sediment deposition in basins
func CalculateSedimentationEffects(icosphereSites []Vector3D, tectonicData *TectonicsData, erosionEffects []float64, params ElevationParameters) []float64 {
	sedimentationEffects := make([]float64, len(icosphereSites))
	
	// Calculate total erosion for sediment budget
	totalErodedVolume := 0.0
	for _, erosion := range erosionEffects {
		if erosion < 0 { // Erosion is negative
			totalErodedVolume += -erosion
		}
	}
	
	if totalErodedVolume == 0 {
		return sedimentationEffects
	}
	
	// Identify sedimentary basins
	basins := identifySedimentaryBasins(icosphereSites, tectonicData, params)
	
	// Distribute eroded sediment to basins
	totalBasinArea := 0.0
	for _, basin := range basins {
		totalBasinArea += basin.Area
	}
	
	if totalBasinArea > 0 {
		sedimentPerUnitArea := totalErodedVolume / totalBasinArea
		
		for _, basin := range basins {
			depositedThickness := sedimentPerUnitArea * 0.5
			
			// Apply to sites within basin
			for _, siteIdx := range basin.SiteIndices {
				if siteIdx < len(sedimentationEffects) {
					sedimentationEffects[siteIdx] = depositedThickness
				}
			}
		}
	}
	
	return sedimentationEffects
}

// identifySedimentaryBasins finds areas suitable for sediment accumulation
func identifySedimentaryBasins(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []SedimentaryBasin {
	var basins []SedimentaryBasin
	
	// Simplified basin identification
	// Look for divergent boundaries and passive margins
	
	currentBasin := SedimentaryBasin{}
	for siteIdx := range icosphereSites {
		siteID := int32(siteIdx)
		
		boundaryType, exists := tectonicData.SiteBoundaryTypes[siteID]
		if exists && boundaryType == tectonics.Divergent {
			// Divergent boundaries create sedimentary basins
			currentBasin.SiteIndices = append(currentBasin.SiteIndices, siteIdx)
			currentBasin.Area += 1.0 // Simplified area calculation
		}
	}
	
	if len(currentBasin.SiteIndices) > 0 {
		basins = append(basins, currentBasin)
	}
	
	return basins
}

// SedimentaryBasin represents an area of sediment accumulation
type SedimentaryBasin struct {
	SiteIndices []int   // Indices of sites within the basin
	Area        float64 // Basin area (normalized units)
	CenterPos   Vector3D // Center position of basin
}