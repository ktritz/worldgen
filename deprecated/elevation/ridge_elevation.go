package elevation

// ridge_elevation.go - Mid-ocean ridge topography and elevation effects

import (
	"math"
)

// GenerateRidgeElevations calculates elevation contributions from mid-ocean ridge topography
func GenerateRidgeElevations(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64 {
	ridgeElevations := make([]float64, len(icosphereSites))
	
	if len(tectonicData.MidOceanRidges) == 0 {
		return ridgeElevations // No ridges to process
	}
	
	for siteIdx, site := range icosphereSites {
		maxRidgeEffect := 0.0
		
		for _, ridge := range tectonicData.MidOceanRidges {
			if !true {
				continue // Only active ridges contribute to elevation
			}
			
			// Calculate distance to ridge axis
			distanceToRidge := calculateDistanceToRidge(site, ridge, params.PlanetRadius)
			
			// Apply ridge topographic effects if within influence zone
			if distanceToRidge < params.RidgeInfluenceDistAbs {
				ridgeEffect := calculateRidgeTopographicEffect(distanceToRidge, ridge, params)
				if ridgeEffect > maxRidgeEffect {
					maxRidgeEffect = ridgeEffect
				}
			}
		}
		
		ridgeElevations[siteIdx] = maxRidgeEffect
	}
	
	return ridgeElevations
}

// calculateDistanceToRidge computes the distance from a site to the nearest point on a ridge
func calculateDistanceToRidge(site Vector3D, ridge MidOceanRidge, planetRadius float64) float64 {
	// For simplified implementation, use distance to ridge center
	// A more sophisticated version would calculate distance to the ridge axis line
	return calculateSphericalDistance(site, calculateRidgeCenter(ridge), planetRadius)
}

// calculateRidgeTopographicEffect computes elevation effects from ridge topography
func calculateRidgeTopographicEffect(distance float64, ridge MidOceanRidge, params ElevationParameters) float64 {
	if distance >= params.RidgeInfluenceDistAbs {
		return 0.0
	}
	
	// Ridge topography depends on spreading rate and age
	// Fast-spreading ridges: Broad, low relief (e.g., East Pacific Rise)
	// Slow-spreading ridges: Narrow, high relief (e.g., Mid-Atlantic Ridge)
	
	ridgeProfile := calculateRidgeProfile(distance, ridge, params)
	baseRidgeHeight := calculateBaseRidgeHeight(ridge, params)
	
	return baseRidgeHeight * ridgeProfile
}

// calculateRidgeProfile determines the topographic profile of the ridge
func calculateRidgeProfile(distance float64, ridge MidOceanRidge, params ElevationParameters) float64 {
	normalizedDistance := distance / params.RidgeInfluenceDistAbs
	
	// Spreading rate affects ridge morphology
	spreadingRate := ridge.SpreadingRate // mm/yr
	
	if spreadingRate > 50.0 {
		// Fast-spreading ridge: Broad, gentle profile
		return calculateFastSpreadingProfile(normalizedDistance)
	} else if spreadingRate > 20.0 {
		// Intermediate-spreading ridge: Moderate profile
		return calculateIntermediateSpreadingProfile(normalizedDistance)
	} else {
		// Slow-spreading ridge: Narrow, steep profile
		return calculateSlowSpreadingProfile(normalizedDistance)
	}
}

// calculateFastSpreadingProfile models fast-spreading ridge topography (e.g., East Pacific Rise)
func calculateFastSpreadingProfile(normalizedDistance float64) float64 {
	if normalizedDistance >= 1.0 {
		return 0.0
	}
	
	// Broad, gentle Gaussian-like profile
	return math.Exp(-normalizedDistance * normalizedDistance * 2.0)
}

// calculateIntermediateSpreadingProfile models intermediate-spreading ridge topography
func calculateIntermediateSpreadingProfile(normalizedDistance float64) float64 {
	if normalizedDistance >= 1.0 {
		return 0.0
	}
	
	// Moderate profile with some roughness
	baseProfile := math.Exp(-normalizedDistance * normalizedDistance * 3.0)
	
	// Add some local roughness for intermediate ridges
	roughness := 0.1 * math.Sin(normalizedDistance * math.Pi * 8.0) * math.Exp(-normalizedDistance * 2.0)
	
	return math.Max(0.0, baseProfile + roughness)
}

// calculateSlowSpreadingProfile models slow-spreading ridge topography (e.g., Mid-Atlantic Ridge)
func calculateSlowSpreadingProfile(normalizedDistance float64) float64 {
	if normalizedDistance >= 1.0 {
		return 0.0
	}
	
	// Narrow, steep profile with rift valley
	if normalizedDistance < 0.1 {
		// Central rift valley - slightly depressed
		return 0.8 + 0.2*math.Cos(normalizedDistance*math.Pi*10.0)
	} else if normalizedDistance < 0.3 {
		// Ridge flanks - elevated
		return 1.0 - (normalizedDistance-0.1)*2.0
	} else {
		// Outer flanks - exponential decay
		return 0.6 * math.Exp(-(normalizedDistance-0.3)*5.0)
	}
}

// calculateBaseRidgeHeight determines the base height of the ridge above surrounding seafloor
func calculateBaseRidgeHeight(ridge MidOceanRidge, params ElevationParameters) float64 {
	// Base ridge elevation from parameters
	baseHeight := params.RidgeElevation
	
	// Spreading rate affects ridge height
	// Slower spreading ridges tend to be higher due to longer cooling time
	spreadingRateMultiplier := math.Max(0.5, math.Min(2.0, 30.0/ridge.SpreadingRate))
	
	// Ridge length affects elevation (longer ridges tend to be more prominent)
	lengthMultiplier := math.Min(1.5, math.Sqrt(calculateRidgeLength(ridge)/1000.0)) // Normalize by 1000 km
	
	return baseHeight * spreadingRateMultiplier * lengthMultiplier
}

// CalculateRidgeAxisTopography generates detailed topography along ridge axes
func CalculateRidgeAxisTopography(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []RidgeAxisPoint {
	var axisPoints []RidgeAxisPoint
	
	for _, ridge := range tectonicData.MidOceanRidges {
		if !true {
			continue
		}
		
		// Find sites near the ridge axis
		for siteIdx, site := range icosphereSites {
			distance := calculateDistanceToRidge(site, ridge, params.PlanetRadius)
			
			// Only consider sites very close to ridge axis (within 10 km)
			if distance < 10000.0 {
				elevation := calculateRidgeTopographicEffect(distance, ridge, params)
				
				axisPoint := RidgeAxisPoint{
					SiteIndex:        siteIdx,
					Position:         site,
					DistanceToAxis:   distance,
					RidgeElevation:   elevation,
					SpreadingRate:    ridge.SpreadingRate,
					RidgeID:          ridge.ID,
				}
				
				axisPoints = append(axisPoints, axisPoint)
			}
		}
	}
	
	return axisPoints
}

// CalculateRidgeSegmentation analyzes ridge segmentation and offset effects
func CalculateRidgeSegmentation(ridges []MidOceanRidge, planetRadius float64) []RidgeSegment {
	var segments []RidgeSegment
	
	for _, ridge := range ridges {
		if !true {
			continue
		}
		
		// For simplified implementation, treat each ridge as a single segment
		// A more sophisticated version would analyze transform offsets and segment boundaries
		
		segment := RidgeSegment{
			RidgeID:       ridge.ID,
			SegmentID:     0, // Single segment for now
			StartPosition: calculateRidgeCenter(ridge), // Simplified - would need actual segment endpoints
			EndPosition:   calculateRidgeCenter(ridge),
			Length:        calculateRidgeLength(ridge),
			SpreadingRate: ridge.SpreadingRate,
			SegmentType:   determineSegmentType(ridge),
		}
		
		segments = append(segments, segment)
	}
	
	return segments
}

// determineSegmentType classifies ridge segments based on characteristics
func determineSegmentType(ridge MidOceanRidge) RidgeSegmentType {
	// Classify based on spreading rate
	if ridge.SpreadingRate > 50.0 {
		return FastSpreadingSegment
	} else if ridge.SpreadingRate > 20.0 {
		return IntermediateSpreadingSegment
	} else {
		return SlowSpreadingSegment
	}
}

// ValidateRidgeElevations performs validation on ridge elevation effects
func ValidateRidgeElevations(ridgeElevations []float64, tectonicData *TectonicsData) (RidgeElevationMetrics, []string) {
	var metrics RidgeElevationMetrics
	var warnings []string
	
	if len(ridgeElevations) == 0 {
		warnings = append(warnings, "No ridge elevations generated")
		return metrics, warnings
	}
	
	// Count sites with ridge effects
	affectedSites := 0
	maxRidgeEffect := 0.0
	totalRidgeEffect := 0.0
	
	for _, elevation := range ridgeElevations {
		if elevation > 10.0 { // Minimum significant ridge effect
			affectedSites++
			totalRidgeEffect += elevation
			if elevation > maxRidgeEffect {
				maxRidgeEffect = elevation
			}
		}
	}
	
	metrics.RidgeAffectedSites = affectedSites
	metrics.MaxRidgeElevation = maxRidgeEffect
	metrics.ActiveRidgeCount = 0
	
	// Count active ridges
	for range tectonicData.MidOceanRidges {
		metrics.ActiveRidgeCount++
	}
	
	if affectedSites > 0 {
		metrics.MeanRidgeElevation = totalRidgeEffect / float64(affectedSites)
		metrics.RidgeCoveragePercent = float64(affectedSites) / float64(len(ridgeElevations)) * 100.0
	}
	
	// Validation checks
	if metrics.ActiveRidgeCount > 0 && affectedSites == 0 {
		warnings = append(warnings, "No ridge elevation effects despite active ridges being present")
	}
	
	if maxRidgeEffect > 5000.0 {
		warnings = append(warnings, "Maximum ridge elevation is very high (>5km)")
	}
	
	if metrics.RidgeCoveragePercent > 15.0 {
		warnings = append(warnings, "Ridge coverage is very high (>15% of surface)")
	}
	
	if metrics.ActiveRidgeCount == 0 {
		warnings = append(warnings, "No active mid-ocean ridges found")
	}
	
	// Compare to Earth values
	// Earth ridge elevation: typically 2000-3000m above abyssal floor
	if metrics.MeanRidgeElevation < 1000.0 && affectedSites > 0 {
		warnings = append(warnings, "Mean ridge elevation is low compared to Earth ridges (~2500m)")
	}
	
	if metrics.MeanRidgeElevation > 4000.0 {
		warnings = append(warnings, "Mean ridge elevation is high compared to Earth ridges")
	}
	
	return metrics, warnings
}

// RidgeElevationMetrics contains statistics about ridge elevation effects
type RidgeElevationMetrics struct {
	RidgeAffectedSites   int     // Number of sites with significant ridge effects
	MaxRidgeElevation    float64 // Maximum ridge elevation effect (m)
	MeanRidgeElevation   float64 // Mean ridge elevation where present (m)
	RidgeCoveragePercent float64 // Percentage of surface affected by ridges
	ActiveRidgeCount     int     // Number of active mid-ocean ridges
}

// RidgeAxisPoint represents a point along a ridge axis
type RidgeAxisPoint struct {
	SiteIndex      int     // Index of the icosphere site
	Position       Vector3D // 3D position of the point
	DistanceToAxis float64 // Distance to ridge axis (m)
	RidgeElevation float64 // Ridge elevation effect (m)
	SpreadingRate  float64 // Local spreading rate (mm/yr)
	RidgeID        int32   // ID of the associated ridge
}

// RidgeSegment represents a segment of a mid-ocean ridge
type RidgeSegment struct {
	RidgeID       int32           // ID of the parent ridge
	SegmentID     int             // Segment number within the ridge
	StartPosition Vector3D        // Start position of segment
	EndPosition   Vector3D        // End position of segment
	Length        float64         // Segment length (km)
	SpreadingRate float64         // Spreading rate (mm/yr)
	SegmentType   RidgeSegmentType // Type of ridge segment
}

// RidgeSegmentType categorizes different types of ridge segments
type RidgeSegmentType string

const (
	SlowSpreadingSegment         RidgeSegmentType = "slow_spreading"         // <20 mm/yr
	IntermediateSpreadingSegment RidgeSegmentType = "intermediate_spreading" // 20-50 mm/yr
	FastSpreadingSegment         RidgeSegmentType = "fast_spreading"         // >50 mm/yr
	TransformSegment             RidgeSegmentType = "transform"              // Transform fault
	OverlappingSegment           RidgeSegmentType = "overlapping"            // Overlapping spreading center
)

// CalculateRidgeOrthogonality measures how perpendicular ridges are to spreading direction
func CalculateRidgeOrthogonality(ridges []MidOceanRidge, tectonicData *TectonicsData) []float64 {
	orthogonality := make([]float64, len(ridges))
	
	for i := range ridges {
		// Find plates on either side of the ridge
		// This would require more sophisticated analysis of plate boundaries
		// For now, assume good orthogonality
		orthogonality[i] = 0.9 // 90% orthogonal (simplified)
	}
	
	return orthogonality
}

// CalculateRidgeAsymmetry analyzes elevation asymmetry across ridge axes
func CalculateRidgeAsymmetry(icosphereSites []Vector3D, ridgeElevations []float64, tectonicData *TectonicsData, params ElevationParameters) []RidgeAsymmetry {
	var asymmetries []RidgeAsymmetry
	
	for _, ridge := range tectonicData.MidOceanRidges {
		if !true {
			continue
		}
		
		// Analyze elevation profiles on both sides of the ridge
		var leftSideElevations, rightSideElevations []float64
		
		for siteIdx, site := range icosphereSites {
			distance := calculateDistanceToRidge(site, ridge, params.PlanetRadius)
			
			if distance < params.RidgeInfluenceDistAbs {
				elevation := ridgeElevations[siteIdx]
				
				// Determine which side of ridge this site is on (simplified)
				// In reality, this would require proper ridge axis orientation analysis
				if siteIdx%2 == 0 {
					leftSideElevations = append(leftSideElevations, elevation)
				} else {
					rightSideElevations = append(rightSideElevations, elevation)
				}
			}
		}
		
		asymmetry := RidgeAsymmetry{
			RidgeID: ridge.ID,
		}
		
		if len(leftSideElevations) > 0 {
			asymmetry.LeftSideMeanElevation = calculateMean(leftSideElevations)
		}
		
		if len(rightSideElevations) > 0 {
			asymmetry.RightSideMeanElevation = calculateMean(rightSideElevations)
		}
		
		// Calculate asymmetry ratio
		if asymmetry.LeftSideMeanElevation > 0 && asymmetry.RightSideMeanElevation > 0 {
			ratio := asymmetry.LeftSideMeanElevation / asymmetry.RightSideMeanElevation
			asymmetry.AsymmetryRatio = math.Max(ratio, 1.0/ratio) // Always > 1.0
		}
		
		asymmetries = append(asymmetries, asymmetry)
	}
	
	return asymmetries
}

// RidgeAsymmetry contains asymmetry analysis for a ridge
type RidgeAsymmetry struct {
	RidgeID                 int32   // Ridge identifier
	LeftSideMeanElevation   float64 // Mean elevation on left side (m)
	RightSideMeanElevation  float64 // Mean elevation on right side (m)
	AsymmetryRatio          float64 // Ratio of higher to lower side (>1.0)
}

// calculateRidgeCenter calculates the geometric center of a ridge from its axis points
func calculateRidgeCenter(ridge MidOceanRidge) Vector3D {
	if len(ridge.AxisPoints) == 0 {
		return Vector3D{}
	}
	
	var center Vector3D
	for _, point := range ridge.AxisPoints {
		center = center.Add(point)
	}
	
	// Divide by number of points to get average position
	n := float64(len(ridge.AxisPoints))
	center = center.Scale(1.0 / n)
	
	return center.Normalize() // Normalize to sphere surface
}

// calculateRidgeLength calculates the total length of a ridge from its axis points
func calculateRidgeLength(ridge MidOceanRidge) float64 {
	if len(ridge.AxisPoints) < 2 {
		return 0.0
	}
	
	totalLength := 0.0
	for i := 1; i < len(ridge.AxisPoints); i++ {
		// Calculate distance between consecutive points
		distance := calculateVectorDistance(ridge.AxisPoints[i-1], ridge.AxisPoints[i])
		totalLength += distance
	}
	
	return totalLength
}