package elevation

// elevation_validation.go - Earth-realistic validation and comparison metrics

import (
	"math"
	"worldgen/landgen/tectonics"
)

// ValidateElevationSystem performs comprehensive validation of the entire elevation system
func ValidateElevationSystem(elevationData *ElevationData, tectonicData *TectonicsData, params ElevationParameters) ElevationValidationReport {
	var report ElevationValidationReport
	
	// Basic elevation statistics
	report.BasicStats = calculateBasicElevationStats(elevationData.SiteElevations)
	
	// Earth comparison metrics
	report.EarthComparison = compareToEarthMetrics(elevationData, tectonicData)
	
	// Tectonic consistency validation
	report.TectonicConsistency = validateTectonicConsistency(elevationData, tectonicData)
	
	// Physical realism checks
	report.PhysicalRealism = validatePhysicalRealism(elevationData, params)
	
	// Component analysis
	report.ComponentAnalysis = analyzeElevationComponents(elevationData)
	
	// Generate warnings and recommendations
	report.Warnings = generateValidationWarnings(report)
	report.Recommendations = generateRecommendations(report, params)
	
	// Overall quality score
	report.OverallQuality = calculateOverallQuality(report)
	
	return report
}

// calculateBasicElevationStats computes fundamental elevation statistics
func calculateBasicElevationStats(elevations []float64) BasicElevationStats {
	var stats BasicElevationStats
	
	if len(elevations) == 0 {
		return stats
	}
	
	// Initialize min/max
	stats.MinElevation = elevations[0]
	stats.MaxElevation = elevations[0]
	
	// Calculate basic statistics
	totalElevation := 0.0
	landElevations := []float64{}
	oceanElevations := []float64{}
	
	for _, elevation := range elevations {
		totalElevation += elevation
		
		if elevation < stats.MinElevation {
			stats.MinElevation = elevation
		}
		if elevation > stats.MaxElevation {
			stats.MaxElevation = elevation
		}
		
		if elevation > 0 {
			landElevations = append(landElevations, elevation)
		} else {
			oceanElevations = append(oceanElevations, elevation)
		}
	}
	
	stats.MeanElevation = totalElevation / float64(len(elevations))
	stats.ElevationRange = stats.MaxElevation - stats.MinElevation
	stats.LandPercentage = float64(len(landElevations)) / float64(len(elevations)) * 100.0
	
	if len(landElevations) > 0 {
		stats.MeanLandElevation = calculateMean(landElevations)
	}
	
	if len(oceanElevations) > 0 {
		stats.MeanOceanDepth = calculateMean(oceanElevations)
	}
	
	// Calculate standard deviation
	variance := 0.0
	for _, elevation := range elevations {
		diff := elevation - stats.MeanElevation
		variance += diff * diff
	}
	stats.ElevationStdDev = math.Sqrt(variance / float64(len(elevations)))
	
	return stats
}

// compareToEarthMetrics compares generated elevations to Earth's characteristics
func compareToEarthMetrics(elevationData *ElevationData, tectonicData *TectonicsData) EarthComparisonMetrics {
	var metrics EarthComparisonMetrics
	
	elevations := elevationData.SiteElevations
	if len(elevations) == 0 {
		return metrics
	}
	
	// Earth reference values
	earthMeanLandElevation := 840.0      // meters
	earthMeanOceanDepth := -3688.0       // meters
	earthHighestPoint := 8848.0          // Mount Everest
	earthLowestPoint := -11034.0         // Challenger Deep
	earthLandPercentage := 29.2          // percent
	
	// Calculate our metrics
	stats := calculateBasicElevationStats(elevations)
	
	// Compare land percentage
	metrics.LandPercentageRatio = stats.LandPercentage / earthLandPercentage
	
	// Compare elevation ranges
	if stats.MeanLandElevation != 0 {
		metrics.MeanLandElevationRatio = stats.MeanLandElevation / earthMeanLandElevation
	}
	
	if stats.MeanOceanDepth != 0 {
		metrics.MeanOceanDepthRatio = stats.MeanOceanDepth / earthMeanOceanDepth
	}
	
	metrics.MaxElevationRatio = stats.MaxElevation / earthHighestPoint
	metrics.MinElevationRatio = stats.MinElevation / earthLowestPoint
	
	// Hypsometric analysis (elevation distribution)
	metrics.HypsometricCurve = calculateHypsometricCurve(elevations)
	metrics.HypsometricIntegral = calculateHypsometricIntegral(metrics.HypsometricCurve)
	
	// Earth's hypsometric integral is approximately 0.5
	earthHypsometricIntegral := 0.5
	metrics.HypsometricRatio = metrics.HypsometricIntegral / earthHypsometricIntegral
	
	// Calculate fit scores (1.0 = perfect match to Earth)
	metrics.LandPercentageFit = calculateFitScore(metrics.LandPercentageRatio)
	metrics.ElevationRangeFit = calculateFitScore((stats.MaxElevation - stats.MinElevation) / (earthHighestPoint - earthLowestPoint))
	metrics.HypsometricFit = calculateFitScore(metrics.HypsometricRatio)
	
	return metrics
}

// calculateHypsometricCurve computes the cumulative elevation distribution
func calculateHypsometricCurve(elevations []float64) []HypsometricPoint {
	if len(elevations) == 0 {
		return nil
	}
	
	// Sort elevations
	sortedElevations := make([]float64, len(elevations))
	copy(sortedElevations, elevations)
	// Simple sort implementation (would use sort.Float64s in production)
	for i := 0; i < len(sortedElevations); i++ {
		for j := i + 1; j < len(sortedElevations); j++ {
			if sortedElevations[i] > sortedElevations[j] {
				sortedElevations[i], sortedElevations[j] = sortedElevations[j], sortedElevations[i]
			}
		}
	}
	
	// Calculate curve points
	var curve []HypsometricPoint
	numPoints := 20 // Generate 20 curve points
	
	for i := 0; i <= numPoints; i++ {
		fraction := float64(i) / float64(numPoints)
		index := int(fraction * float64(len(sortedElevations)-1))
		
		if index >= len(sortedElevations) {
			index = len(sortedElevations) - 1
		}
		
		point := HypsometricPoint{
			AreaFraction:      fraction,
			ElevationFraction: (sortedElevations[index] - sortedElevations[0]) / (sortedElevations[len(sortedElevations)-1] - sortedElevations[0]),
		}
		
		curve = append(curve, point)
	}
	
	return curve
}

// calculateHypsometricIntegral computes the area under the hypsometric curve
func calculateHypsometricIntegral(curve []HypsometricPoint) float64 {
	if len(curve) < 2 {
		return 0.0
	}
	
	integral := 0.0
	for i := 1; i < len(curve); i++ {
		// Trapezoidal integration
		width := curve[i].AreaFraction - curve[i-1].AreaFraction
		height := (curve[i].ElevationFraction + curve[i-1].ElevationFraction) / 2.0
		integral += width * height
	}
	
	return integral
}

// validateTectonicConsistency checks if elevations are consistent with tectonic setting
func validateTectonicConsistency(elevationData *ElevationData, tectonicData *TectonicsData) TectonicConsistencyMetrics {
	var metrics TectonicConsistencyMetrics
	
	elevations := elevationData.SiteElevations
	if len(elevations) == 0 || len(tectonicData.Plates) == 0 {
		return metrics
	}
	
	// Analyze elevation by plate type
	continentalElevations := []float64{}
	oceanicElevations := []float64{}
	
	for siteIdx, elevation := range elevations {
		siteID := int32(siteIdx)
		if siteID >= int32(len(tectonicData.SitePlateIDs)) {
			continue
		}
		
		plateID := tectonicData.SitePlateIDs[siteID]
		plate := findPlateByID(tectonicData.Plates, plateID)
		
		if plate == nil {
			continue
		}
		
		switch plate.PlateType {
		case tectonics.ContinentalPlate:
			continentalElevations = append(continentalElevations, elevation)
		case tectonics.OceanicPlate:
			oceanicElevations = append(oceanicElevations, elevation)
		}
	}
	
	// Calculate consistency metrics
	if len(continentalElevations) > 0 {
		metrics.MeanContinentalElevation = calculateMean(continentalElevations)
		metrics.ContinentalElevationRange = calculateRange(continentalElevations)
	}
	
	if len(oceanicElevations) > 0 {
		metrics.MeanOceanicElevation = calculateMean(oceanicElevations)
		metrics.OceanicElevationRange = calculateRange(oceanicElevations)
	}
	
	// Check if continental crust is generally higher than oceanic
	if len(continentalElevations) > 0 && len(oceanicElevations) > 0 {
		metrics.ContinentalOceanicContrast = metrics.MeanContinentalElevation - metrics.MeanOceanicElevation
	}
	
	// Analyze boundary effects
	metrics.BoundaryConsistency = analyzeBoundaryElevationConsistency(elevations, tectonicData)
	
	// Calculate consistency scores
	metrics.PlateTypeConsistencyScore = calculatePlateTypeConsistency(continentalElevations, oceanicElevations)
	metrics.BoundaryConsistencyScore = metrics.BoundaryConsistency
	
	return metrics
}

// analyzeBoundaryElevationConsistency checks elevation patterns at tectonic boundaries
func analyzeBoundaryElevationConsistency(elevations []float64, tectonicData *TectonicsData) float64 {
	convergentElevations := []float64{}
	divergentElevations := []float64{}
	transformElevations := []float64{}
	
	for siteID, boundaryType := range tectonicData.SiteBoundaryTypes {
		if int(siteID) >= len(elevations) {
			continue
		}
		
		elevation := elevations[siteID]
		
		switch boundaryType {
		case tectonics.Convergent:
			convergentElevations = append(convergentElevations, elevation)
		case tectonics.Divergent:
			divergentElevations = append(divergentElevations, elevation)
		case tectonics.Passive:
			transformElevations = append(transformElevations, elevation)
		}
	}
	
	// Convergent boundaries should generally have higher elevations
	// Divergent boundaries should have mixed or lower elevations
	// This is a simplified consistency check
	
	score := 1.0
	
	if len(convergentElevations) > 0 && len(divergentElevations) > 0 {
		convergentMean := calculateMean(convergentElevations)
		divergentMean := calculateMean(divergentElevations)
		
		// Good consistency if convergent > divergent
		if convergentMean > divergentMean {
			score *= 1.2
		} else {
			score *= 0.8
		}
	}
	
	return math.Min(1.0, score)
}

// calculatePlateTypeConsistency scores how well plate types match expected elevations
func calculatePlateTypeConsistency(continentalElevations, oceanicElevations []float64) float64 {
	if len(continentalElevations) == 0 || len(oceanicElevations) == 0 {
		return 0.5 // Partial score if missing data
	}
	
	continentalMean := calculateMean(continentalElevations)
	oceanicMean := calculateMean(oceanicElevations)
	
	// Continental crust should be higher than oceanic
	contrast := continentalMean - oceanicMean
	
	// Expected contrast: ~4000-5000m (continental ~800m, oceanic ~-3700m)
	expectedContrast := 4500.0
	
	ratio := contrast / expectedContrast
	
	// Score based on how close to expected contrast
	return calculateFitScore(ratio)
}

// validatePhysicalRealism checks physical constraints and realistic values
func validatePhysicalRealism(elevationData *ElevationData, params ElevationParameters) PhysicalRealismMetrics {
	var metrics PhysicalRealismMetrics
	
	elevations := elevationData.SiteElevations
	if len(elevations) == 0 {
		return metrics
	}
	
	// Check for extreme values
	for _, elevation := range elevations {
		if elevation > 15000.0 { // Higher than any Earth mountain
			metrics.ExtremeHighCount++
		}
		if elevation < -15000.0 { // Deeper than deepest ocean trench
			metrics.ExtremeDeepCount++
		}
	}
	
	metrics.ExtremeValueRatio = float64(metrics.ExtremeHighCount+metrics.ExtremeDeepCount) / float64(len(elevations))
	
	// Check elevation gradients for unrealistic steepness
	metrics.SteepGradientRatio = calculateSteepGradientRatio(elevations, params)
	
	// Check isostatic balance (simplified)
	metrics.IsostaticBalanceScore = calculateIsostaticBalance(elevationData)
	
	// Overall physical realism score
	realismScore := 1.0
	
	// Penalize extreme values
	if metrics.ExtremeValueRatio > 0.01 { // More than 1% extreme values
		realismScore *= 0.5
	}
	
	// Penalize unrealistic gradients
	if metrics.SteepGradientRatio > 0.1 { // More than 10% too steep
		realismScore *= 0.7
	}
	
	metrics.OverallRealismScore = realismScore
	
	return metrics
}

// calculateSteepGradientRatio finds the ratio of unrealistically steep gradients
func calculateSteepGradientRatio(elevations []float64, params ElevationParameters) float64 {
	// This would need actual neighbor distances for proper calculation
	// Simplified version counts large elevation differences
	
	steepCount := 0
	totalPairs := 0
	
	// Check adjacent elevation differences (simplified)
	for i := 1; i < len(elevations); i++ {
		diff := math.Abs(elevations[i] - elevations[i-1])
		totalPairs++
		
		// Assume some characteristic distance between points
		// In reality, this would use actual spherical distances
		if diff > 1000.0 { // More than 1km difference between neighbors
			steepCount++
		}
	}
	
	if totalPairs > 0 {
		return float64(steepCount) / float64(totalPairs)
	}
	
	return 0.0
}

// calculateIsostaticBalance checks for reasonable elevation distribution
func calculateIsostaticBalance(elevationData *ElevationData) float64 {
	// Simplified isostatic balance check
	// Real version would consider crustal thickness and density
	
	elevations := elevationData.SiteElevations
	if len(elevations) == 0 {
		return 0.0
	}
	
	// Calculate mass distribution approximation
	totalMass := 0.0
	for _, elevation := range elevations {
		// Assume some relationship between elevation and crustal mass
		if elevation > 0 {
			totalMass += elevation * 2.7 // Continental crust density approximation
		} else {
			totalMass += elevation * 3.0 // Oceanic crust + water
		}
	}
	
	// Isostatic balance suggests total mass should be relatively stable
	// This is a very simplified check
	averageMass := totalMass / float64(len(elevations))
	
	// Score based on how balanced the mass distribution is
	// Perfect balance would have average mass near zero
	balance := 1.0 / (1.0 + math.Abs(averageMass)/1000.0)
	
	return balance
}

// analyzeElevationComponents breaks down contribution of each elevation component
func analyzeElevationComponents(elevationData *ElevationData) ComponentAnalysisMetrics {
	var metrics ComponentAnalysisMetrics
	
	if len(elevationData.SiteElevations) == 0 {
		return metrics
	}
	
	// Calculate contribution statistics for each component
	if len(elevationData.BaseElevations) > 0 {
		metrics.BaseElevationContribution = calculateMeanAbsolute(elevationData.BaseElevations)
	}
	
	if len(elevationData.VolcanicElevations) > 0 {
		metrics.VolcanicContribution = calculateMeanAbsolute(elevationData.VolcanicElevations)
	}
	
	if len(elevationData.SeafloorElevations) > 0 {
		metrics.SeafloorContribution = calculateMeanAbsolute(elevationData.SeafloorElevations)
	}
	
	if len(elevationData.RidgeElevations) > 0 {
		metrics.RidgeContribution = calculateMeanAbsolute(elevationData.RidgeElevations)
	}
	
	if len(elevationData.TectonicElevations) > 0 {
		metrics.TectonicContribution = calculateMeanAbsolute(elevationData.TectonicElevations)
	}
	
	if len(elevationData.ErosionElevations) > 0 {
		metrics.ErosionContribution = calculateMeanAbsolute(elevationData.ErosionElevations)
	}
	
	if len(elevationData.NoiseElevations) > 0 {
		metrics.NoiseContribution = calculateMeanAbsolute(elevationData.NoiseElevations)
	}
	
	// Calculate total contribution
	metrics.TotalContribution = metrics.BaseElevationContribution + metrics.VolcanicContribution +
		metrics.SeafloorContribution + metrics.RidgeContribution + metrics.TectonicContribution +
		metrics.ErosionContribution + metrics.NoiseContribution
	
	return metrics
}

// calculateMeanAbsolute is now in utilities.go

// calculateFitScore is now in utilities.go

// generateValidationWarnings creates warning messages based on validation results
func generateValidationWarnings(report ElevationValidationReport) []string {
	var warnings []string
	
	// Check basic statistics
	if report.BasicStats.LandPercentage < 10.0 || report.BasicStats.LandPercentage > 50.0 {
		warnings = append(warnings, "Land percentage is outside realistic range (10-50%)")
	}
	
	if report.BasicStats.MaxElevation > 12000.0 {
		warnings = append(warnings, "Maximum elevation exceeds realistic mountain heights")
	}
	
	if report.BasicStats.MinElevation < -12000.0 {
		warnings = append(warnings, "Minimum elevation exceeds realistic ocean depths")
	}
	
	// Check Earth comparison
	if report.EarthComparison.LandPercentageFit < 0.5 {
		warnings = append(warnings, "Land percentage poorly matches Earth values")
	}
	
	if report.EarthComparison.ElevationRangeFit < 0.5 {
		warnings = append(warnings, "Elevation range poorly matches Earth values")
	}
	
	// Check tectonic consistency
	if report.TectonicConsistency.PlateTypeConsistencyScore < 0.5 {
		warnings = append(warnings, "Plate type elevations are inconsistent with expectations")
	}
	
	// Check physical realism
	if report.PhysicalRealism.ExtremeValueRatio > 0.05 {
		warnings = append(warnings, "High percentage of extreme elevation values")
	}
	
	if report.PhysicalRealism.OverallRealismScore < 0.6 {
		warnings = append(warnings, "Overall physical realism is questionable")
	}
	
	return warnings
}

// generateRecommendations provides suggestions for improving elevation generation
func generateRecommendations(report ElevationValidationReport, params ElevationParameters) []string {
	var recommendations []string
	
	// Recommendations based on validation results
	if report.EarthComparison.LandPercentageRatio < 0.8 {
		recommendations = append(recommendations, "Increase continental plate size or elevation to achieve more realistic land percentage")
	}
	
	if report.TectonicConsistency.ContinentalOceanicContrast < 3000.0 {
		recommendations = append(recommendations, "Increase contrast between continental and oceanic elevations")
	}
	
	if report.ComponentAnalysis.BaseElevationContribution < 1000.0 {
		recommendations = append(recommendations, "Increase base elevation parameters for more pronounced topography")
	}
	
	if report.PhysicalRealism.SteepGradientRatio > 0.1 {
		recommendations = append(recommendations, "Apply more smoothing or erosion to reduce unrealistic gradients")
	}
	
	if report.OverallQuality < 0.7 {
		recommendations = append(recommendations, "Consider adjusting multiple parameters for overall improvement")
	}
	
	return recommendations
}

// calculateOverallQuality computes a single quality score from all metrics
func calculateOverallQuality(report ElevationValidationReport) float64 {
	// Weight different aspects of quality
	weights := QualityWeights{
		EarthSimilarity:       0.3,
		TectonicConsistency:   0.25,
		PhysicalRealism:       0.25,
		ComponentBalance:      0.2,
	}
	
	// Calculate weighted average
	earthScore := (report.EarthComparison.LandPercentageFit + report.EarthComparison.ElevationRangeFit + 
		report.EarthComparison.HypsometricFit) / 3.0
	
	tectonicScore := (report.TectonicConsistency.PlateTypeConsistencyScore + 
		report.TectonicConsistency.BoundaryConsistencyScore) / 2.0
	
	physicalScore := report.PhysicalRealism.OverallRealismScore
	
	// Component balance score (prefer balanced contributions)
	componentBalance := 1.0
	if report.ComponentAnalysis.TotalContribution > 0 {
		// Calculate how balanced the components are
		components := []float64{
			report.ComponentAnalysis.BaseElevationContribution,
			report.ComponentAnalysis.VolcanicContribution,
			report.ComponentAnalysis.TectonicContribution,
		}
		
		variance := 0.0
		mean := calculateMean(components)
		for _, comp := range components {
			diff := comp - mean
			variance += diff * diff
		}
		variance /= float64(len(components))
		
		// Lower variance = better balance
		componentBalance = 1.0 / (1.0 + variance/1000000.0)
	}
	
	overallQuality := earthScore*weights.EarthSimilarity +
		tectonicScore*weights.TectonicConsistency +
		physicalScore*weights.PhysicalRealism +
		componentBalance*weights.ComponentBalance
	
	return math.Max(0.0, math.Min(1.0, overallQuality))
}

// Data structures for validation results

type ElevationValidationReport struct {
	BasicStats           BasicElevationStats       `json:"basic_stats"`
	EarthComparison      EarthComparisonMetrics    `json:"earth_comparison"`
	TectonicConsistency  TectonicConsistencyMetrics `json:"tectonic_consistency"`
	PhysicalRealism      PhysicalRealismMetrics    `json:"physical_realism"`
	ComponentAnalysis    ComponentAnalysisMetrics  `json:"component_analysis"`
	Warnings             []string                  `json:"warnings"`
	Recommendations      []string                  `json:"recommendations"`
	OverallQuality       float64                   `json:"overall_quality"`
}

type BasicElevationStats struct {
	MinElevation      float64 `json:"min_elevation"`
	MaxElevation      float64 `json:"max_elevation"`
	MeanElevation     float64 `json:"mean_elevation"`
	ElevationRange    float64 `json:"elevation_range"`
	ElevationStdDev   float64 `json:"elevation_std_dev"`
	LandPercentage    float64 `json:"land_percentage"`
	MeanLandElevation float64 `json:"mean_land_elevation"`
	MeanOceanDepth    float64 `json:"mean_ocean_depth"`
}

type EarthComparisonMetrics struct {
	LandPercentageRatio     float64           `json:"land_percentage_ratio"`
	MeanLandElevationRatio  float64           `json:"mean_land_elevation_ratio"`
	MeanOceanDepthRatio     float64           `json:"mean_ocean_depth_ratio"`
	MaxElevationRatio       float64           `json:"max_elevation_ratio"`
	MinElevationRatio       float64           `json:"min_elevation_ratio"`
	HypsometricCurve        []HypsometricPoint `json:"hypsometric_curve"`
	HypsometricIntegral     float64           `json:"hypsometric_integral"`
	HypsometricRatio        float64           `json:"hypsometric_ratio"`
	LandPercentageFit       float64           `json:"land_percentage_fit"`
	ElevationRangeFit       float64           `json:"elevation_range_fit"`
	HypsometricFit          float64           `json:"hypsometric_fit"`
}

type HypsometricPoint struct {
	AreaFraction      float64 `json:"area_fraction"`
	ElevationFraction float64 `json:"elevation_fraction"`
}

type TectonicConsistencyMetrics struct {
	MeanContinentalElevation     float64 `json:"mean_continental_elevation"`
	MeanOceanicElevation         float64 `json:"mean_oceanic_elevation"`
	ContinentalElevationRange    float64 `json:"continental_elevation_range"`
	OceanicElevationRange        float64 `json:"oceanic_elevation_range"`
	ContinentalOceanicContrast   float64 `json:"continental_oceanic_contrast"`
	BoundaryConsistency          float64 `json:"boundary_consistency"`
	PlateTypeConsistencyScore    float64 `json:"plate_type_consistency_score"`
	BoundaryConsistencyScore     float64 `json:"boundary_consistency_score"`
}

type PhysicalRealismMetrics struct {
	ExtremeHighCount      int     `json:"extreme_high_count"`
	ExtremeDeepCount      int     `json:"extreme_deep_count"`
	ExtremeValueRatio     float64 `json:"extreme_value_ratio"`
	SteepGradientRatio    float64 `json:"steep_gradient_ratio"`
	IsostaticBalanceScore float64 `json:"isostatic_balance_score"`
	OverallRealismScore   float64 `json:"overall_realism_score"`
}

type ComponentAnalysisMetrics struct {
	BaseElevationContribution float64 `json:"base_elevation_contribution"`
	VolcanicContribution      float64 `json:"volcanic_contribution"`
	SeafloorContribution      float64 `json:"seafloor_contribution"`
	RidgeContribution         float64 `json:"ridge_contribution"`
	TectonicContribution      float64 `json:"tectonic_contribution"`
	ErosionContribution       float64 `json:"erosion_contribution"`
	NoiseContribution         float64 `json:"noise_contribution"`
	TotalContribution         float64 `json:"total_contribution"`
}

type QualityWeights struct {
	EarthSimilarity     float64 `json:"earth_similarity"`
	TectonicConsistency float64 `json:"tectonic_consistency"`
	PhysicalRealism     float64 `json:"physical_realism"`
	ComponentBalance    float64 `json:"component_balance"`
}