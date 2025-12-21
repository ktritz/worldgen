package elevation

// utilities.go - Common utility functions for elevation generation

import (
	"math"
)

// Mathematical utilities

// calculateMean computes the arithmetic mean of a slice of float64 values
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	
	return sum / float64(len(values))
}

// calculateMeanAbsolute calculates the mean of absolute values
func calculateMeanAbsolute(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	total := 0.0
	for _, value := range values {
		total += math.Abs(value)
	}
	
	return total / float64(len(values))
}

// calculateRange computes the range (max - min) of a slice of float64 values
func calculateRange(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	min := values[0]
	max := values[0]
	
	for _, value := range values {
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	
	return max - min
}

// calculateStandardDeviation computes the standard deviation of values
func calculateStandardDeviation(values []float64) float64 {
	if len(values) <= 1 {
		return 0.0
	}
	
	mean := calculateMean(values)
	variance := 0.0
	
	for _, value := range values {
		diff := value - mean
		variance += diff * diff
	}
	
	variance /= float64(len(values) - 1)
	return math.Sqrt(variance)
}

// calculateMedian calculates the median of a slice of float64 values
func calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	// Create a copy to avoid modifying original
	sorted := make([]float64, len(values))
	copy(sorted, values)
	
	// Simple sort implementation
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2.0
	}
	return sorted[n/2]
}

// Geometric utilities

// calculateSphericalDistance computes the great circle distance between two points on a sphere
func calculateSphericalDistance(p1, p2 Vector3D, radius float64) float64 {
	// Normalize vectors to unit sphere
	unit1 := p1.Normalize()
	unit2 := p2.Normalize()
	
	// Calculate dot product using icosphere method
	dotProduct := unit1.Dot(unit2)
	
	// Clamp to valid range to avoid numerical errors
	if dotProduct > 1.0 {
		dotProduct = 1.0
	} else if dotProduct < -1.0 {
		dotProduct = -1.0
	}
	
	// Calculate angular distance and convert to linear distance
	angularDistance := math.Acos(dotProduct)
	return angularDistance * radius
}

// calculateVectorDistance computes the Euclidean distance between two 3D points
func calculateVectorDistance(p1, p2 Vector3D) float64 {
	diff := p1.Subtract(p2)
	return diff.Length()
}

// normalizeVector normalizes a 3D vector to unit length
func normalizeVector(v Vector3D) Vector3D {
	return v.Normalize()
}

// dotProduct computes the dot product of two 3D vectors
func dotProduct(v1, v2 Vector3D) float64 {
	return v1.Dot(v2)
}

// crossProduct computes the cross product of two 3D vectors
func crossProduct(v1, v2 Vector3D) Vector3D {
	return v1.Cross(v2)
}

// Data structure utilities

// findPlateByID finds a tectonic plate by its ID
func findPlateByID(plates []TectonicPlate, plateID int32) *TectonicPlate {
	for i := range plates {
		if plates[i].ID == plateID {
			return &plates[i]
		}
	}
	return nil
}

// findHotspotByID finds a hotspot by its ID
func findHotspotByID(hotspots []Hotspot, hotspotID int32) *Hotspot {
	for i := range hotspots {
		if hotspots[i].ID == hotspotID {
			return &hotspots[i]
		}
	}
	return nil
}

// findVolcanicFeaturesByHotspot finds all volcanic features associated with a hotspot
func findVolcanicFeaturesByHotspot(features []VolcanicFeature, hotspotID int32) []VolcanicFeature {
	var result []VolcanicFeature
	for _, feature := range features {
		if feature.HotspotID == hotspotID {
			result = append(result, feature)
		}
	}
	return result
}

// Interpolation utilities

// linearInterpolate performs linear interpolation between two values
func linearInterpolate(a, b, t float64) float64 {
	return a + t*(b-a)
}

// smoothstep performs smooth interpolation with easing
func smoothstep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

// exponentialFalloff calculates exponential falloff with distance
func exponentialFalloff(distance, characteristicDistance float64) float64 {
	if characteristicDistance <= 0 {
		return 0.0
	}
	return math.Exp(-distance / characteristicDistance)
}

// gaussianFalloff calculates Gaussian falloff with distance
func gaussianFalloff(distance, standardDeviation float64) float64 {
	if standardDeviation <= 0 {
		return 0.0
	}
	return math.Exp(-0.5 * math.Pow(distance/standardDeviation, 2))
}

// Validation utilities

// isValidElevation checks if an elevation value is within reasonable bounds
func isValidElevation(elevation float64) bool {
	return elevation >= -15000.0 && elevation <= 15000.0 // Earth-like bounds
}

// clampElevation clamps an elevation value to reasonable bounds
func clampElevation(elevation float64) float64 {
	if elevation < -15000.0 {
		return -15000.0
	}
	if elevation > 15000.0 {
		return 15000.0
	}
	return elevation
}

// calculateFitScore converts a ratio to a fit score (1.0 = perfect, 0.0 = very poor)
func calculateFitScore(ratio float64) float64 {
	// Score is highest when ratio is close to 1.0
	deviation := math.Abs(ratio - 1.0)
	
	// Use exponential decay for scoring
	score := math.Exp(-deviation)
	
	return math.Max(0.0, math.Min(1.0, score))
}

// Array utilities

// findNearestNeighbors finds sites within a given distance (simplified implementation)
func findNearestNeighbors(siteIdx int, sites []Vector3D, planetRadius, maxDistance float64) []int {
	var neighbors []int
	
	if siteIdx >= len(sites) {
		return neighbors
	}
	
	for i, site := range sites {
		if i == siteIdx {
			continue
		}
		
		distance := calculateSphericalDistance(sites[siteIdx], site, planetRadius)
		if distance < maxDistance {
			neighbors = append(neighbors, i)
		}
		
		// Limit to reasonable number of neighbors for performance
		if len(neighbors) > 50 {
			break
		}
	}
	
	return neighbors
}

// removeOutliers removes extreme outliers from a dataset using IQR method
func removeOutliers(values []float64, multiplier float64) []float64 {
	if len(values) < 4 {
		return values
	}
	
	// Sort values
	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	// Calculate quartiles
	n := len(sorted)
	q1 := sorted[n/4]
	q3 := sorted[3*n/4]
	iqr := q3 - q1
	
	// Calculate bounds
	lowerBound := q1 - multiplier*iqr
	upperBound := q3 + multiplier*iqr
	
	// Filter outliers
	var filtered []float64
	for _, value := range values {
		if value >= lowerBound && value <= upperBound {
			filtered = append(filtered, value)
		}
	}
	
	return filtered
}

// Noise utilities

// fade applies smoothstep function for noise generation
func fade(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

// lerp performs linear interpolation
func lerp(a, b, t float64) float64 {
	return a + t*(b-a)
}

// hash1D generates a simple hash for 1D coordinates
func hash1D(x int, seed int64) int {
	hash := int(seed)
	hash = hash*1103515245 + 12345
	hash ^= x
	hash = hash*1103515245 + 12345
	return hash & 0xFF
}

// hash3D generates a hash value for 3D coordinates
func hash3D(x, y, z int, seed int64) int {
	hash := int(seed)
	hash = hash*1103515245 + 12345
	hash ^= x
	hash = hash*1103515245 + 12345
	hash ^= y
	hash = hash*1103515245 + 12345
	hash ^= z
	hash = hash*1103515245 + 12345
	return hash & 0xFF
}

// Constants for common calculations
const (
	EarthRadius        = 6371000.0 // meters
	EarthLandPercent   = 29.2      // percent
	EarthMeanLandElev  = 840.0     // meters
	EarthMeanOceanDepth = -3688.0  // meters
	EarthHighestPoint  = 8848.0    // Mount Everest, meters
	EarthLowestPoint   = -11034.0  // Challenger Deep, meters
)

// Default parameter functions

// getFloatOrDefault returns value if non-zero, otherwise returns defaultValue
func getFloatOrDefault(value, defaultValue float64) float64 {
	if value == 0 {
		return defaultValue
	}
	return value
}

// getIntOrDefault returns value if non-zero, otherwise returns defaultValue
func getIntOrDefault(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}

// getBoolOrDefault returns value if specified, otherwise returns defaultValue
func getBoolOrDefault(value, defaultValue bool) bool {
	// For booleans, we can't distinguish between false and unset
	// This function exists for API consistency
	return value || defaultValue
}