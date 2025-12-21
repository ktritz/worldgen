package elevation

// elevation_noise.go - Fractal noise and terrain detail generation

import (
	"math"
	"worldgen/landgen/tectonics"
)

// GenerateElevationNoise adds fractal noise for realistic terrain detail
func GenerateElevationNoise(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64 {
	elevationNoise := make([]float64, len(icosphereSites))
	
	if params.NoiseAmplitude <= 0 {
		return elevationNoise
	}
	
	for siteIdx, site := range icosphereSites {
		// Generate multi-octave fractal noise
		noiseValue := generateFractalNoise(site, params)
		
		// Apply tectonic context-dependent scaling
		scaledNoise := applyTectonicNoiseScaling(int32(siteIdx), noiseValue, tectonicData, params)
		
		elevationNoise[siteIdx] = scaledNoise
	}
	
	return elevationNoise
}

// generateFractalNoise creates multi-octave Perlin-style noise
func generateFractalNoise(position Vector3D, params ElevationParameters) float64 {
	totalNoise := 0.0
	amplitude := params.NoiseAmplitude
	frequency := params.NoiseScale
	maxValue := 0.0
	
	// Generate multiple octaves
	for octave := 0; octave < params.NoiseOctaves; octave++ {
		// Generate noise at this frequency
		noiseValue := perlinNoise3D(
			position.X*frequency,
			position.Y*frequency,
			position.Z*frequency,
			params.ElevationSeed+int64(octave),
		)
		
		totalNoise += noiseValue * amplitude
		maxValue += amplitude
		
		// Increase frequency and decrease amplitude for next octave
		frequency *= params.NoiseLacunarity
		amplitude *= params.NoisePersistence
	}
	
	// Normalize to [-1, 1] range
	if maxValue > 0 {
		totalNoise /= maxValue
	}
	
	return totalNoise
}

// perlinNoise3D generates 3D Perlin noise
func perlinNoise3D(x, y, z float64, seed int64) float64 {
	// Simplified 3D Perlin noise implementation
	// In production, would use a proper noise library
	
	// Integer coordinates
	xi := int(math.Floor(x)) & 255
	yi := int(math.Floor(y)) & 255
	zi := int(math.Floor(z)) & 255
	
	// Fractional coordinates
	xf := x - math.Floor(x)
	yf := y - math.Floor(y)
	zf := z - math.Floor(z)
	
	// Fade curves
	u := fade(xf)
	v := fade(yf)
	w := fade(zf)
	
	// Hash coordinates of 8 cube corners
	aaa := grad3D(hash3D(xi, yi, zi, seed), xf, yf, zf)
	aba := grad3D(hash3D(xi, yi+1, zi, seed), xf, yf-1, zf)
	aab := grad3D(hash3D(xi, yi, zi+1, seed), xf, yf, zf-1)
	abb := grad3D(hash3D(xi, yi+1, zi+1, seed), xf, yf-1, zf-1)
	baa := grad3D(hash3D(xi+1, yi, zi, seed), xf-1, yf, zf)
	bba := grad3D(hash3D(xi+1, yi+1, zi, seed), xf-1, yf-1, zf)
	bab := grad3D(hash3D(xi+1, yi, zi+1, seed), xf-1, yf, zf-1)
	bbb := grad3D(hash3D(xi+1, yi+1, zi+1, seed), xf-1, yf-1, zf-1)
	
	// Interpolate
	x1 := lerp(aaa, baa, u)
	x2 := lerp(aba, bba, u)
	y1 := lerp(x1, x2, v)
	
	x1 = lerp(aab, bab, u)
	x2 = lerp(abb, bbb, u)
	y2 := lerp(x1, x2, v)
	
	return lerp(y1, y2, w)
}


// grad3D computes gradient in 3D
func grad3D(hash int, x, y, z float64) float64 {
	h := hash & 15
	u := x
	if h >= 8 {
		u = y
	}
	
	var v float64
	if h < 4 {
		v = y
	} else if h == 12 || h == 14 {
		v = x
	} else {
		v = z
	}
	
	result := u
	if (h & 1) != 0 {
		result = -result
	}
	if (h & 2) != 0 {
		v = -v
	}
	
	return result + v
}


// applyTectonicNoiseScaling scales noise based on tectonic and elevation context
func applyTectonicNoiseScaling(siteID int32, noiseValue float64, tectonicData *TectonicsData, params ElevationParameters) float64 {
	plateID := tectonicData.SitePlateIDs[siteID]
	plate := findPlateByID(tectonicData.Plates, plateID)

	if plate == nil {
		return noiseValue * params.NoiseAmplitude
	}

	// Start with base amplitude (should be 50-100m for realistic terrain)
	amplitude := params.NoiseAmplitude

	// Plate type base scaling
	plateTypeScale := 1.0
	switch plate.PlateType {
	case tectonics.ContinentalPlate:
		plateTypeScale = 1.0 // Continental gets full amplitude
	case tectonics.OceanicPlate:
		plateTypeScale = 0.3 // Oceanic is much smoother
	}

	// Distance to boundary scaling (near boundaries = more roughness)
	boundaryScale := 0.5 // Base scale for interior regions
	if int(siteID) < len(tectonicData.SiteDistancesToBoundary) {
		distKm := tectonicData.SiteDistancesToBoundary[siteID] / 1000.0 // Convert to km

		// Tectonic effects extend ~500km from boundaries
		normalizedDist := math.Min(1.0, distKm/500.0)

		// Near boundaries (0-500km): scale from 2.0 to 0.5
		// Far from boundaries (>500km): constant 0.5
		boundaryScale = 2.0 - (1.5 * normalizedDist)
	}

	// Check if we're on a boundary for extra roughness
	boundaryType, onBoundary := tectonicData.SiteBoundaryTypes[siteID]
	if onBoundary {
		switch boundaryType {
		case tectonics.Convergent:
			boundaryScale *= 1.5 // Mountains are extra rough
		case tectonics.Divergent:
			boundaryScale *= 1.2 // Rifts have moderate roughness
		}
	}

	// Combine scales
	finalScale := amplitude * plateTypeScale * boundaryScale

	return noiseValue * finalScale
}

// GenerateRidgedNoise creates ridged multifractal noise for mountain terrain
func GenerateRidgedNoise(icosphereSites []Vector3D, params ElevationParameters) []float64 {
	ridgedNoise := make([]float64, len(icosphereSites))
	
	for siteIdx, site := range icosphereSites {
		totalNoise := 0.0
		amplitude := params.NoiseAmplitude * 0.8
		frequency := params.NoiseScale
		weight := 1.0
		
		for octave := 0; octave < params.NoiseOctaves; octave++ {
			// Generate noise
			signal := perlinNoise3D(
				site.X*frequency,
				site.Y*frequency,
				site.Z*frequency,
				params.ElevationSeed+int64(octave)+1000,
			)
			
			// Create ridged effect
			signal = math.Abs(signal)
			signal = 1.0 - signal
			signal = signal * signal
			
			// Weight by previous octave
			signal *= weight
			weight = signal * 1.0
			weight = math.Max(0.0, math.Min(1.0, weight))
			
			totalNoise += signal * amplitude
			frequency *= params.NoiseLacunarity
			amplitude *= params.NoisePersistence
		}
		
		ridgedNoise[siteIdx] = totalNoise
	}
	
	return ridgedNoise
}

// GenerateBillowNoise creates billow noise for rolling terrain
func GenerateBillowNoise(icosphereSites []Vector3D, params ElevationParameters) []float64 {
	billowNoise := make([]float64, len(icosphereSites))
	
	for siteIdx, site := range icosphereSites {
		totalNoise := 0.0
		amplitude := params.NoiseAmplitude * 0.6
		frequency := params.NoiseScale
		
		for octave := 0; octave < params.NoiseOctaves; octave++ {
			// Generate noise
			signal := perlinNoise3D(
				site.X*frequency,
				site.Y*frequency,
				site.Z*frequency,
				params.ElevationSeed+int64(octave)+2000,
			)
			
			// Create billow effect (always positive)
			signal = math.Abs(signal)
			
			totalNoise += signal * amplitude
			frequency *= params.NoiseLacunarity
			amplitude *= params.NoisePersistence
		}
		
		billowNoise[siteIdx] = totalNoise
	}
	
	return billowNoise
}

// GenerateTurbulenceNoise creates turbulence for terrain detail
func GenerateTurbulenceNoise(icosphereSites []Vector3D, params ElevationParameters) []float64 {
	turbulenceNoise := make([]float64, len(icosphereSites))
	
	for siteIdx, site := range icosphereSites {
		// Generate multiple noise components
		noiseX := perlinNoise3D(site.X*params.NoiseScale, 
			site.Y*params.NoiseScale, 
			site.Z*params.NoiseScale, 
			params.ElevationSeed+3000)
		
		noiseY := perlinNoise3D(site.X*params.NoiseScale, 
			site.Y*params.NoiseScale, 
			site.Z*params.NoiseScale, 
			params.ElevationSeed+4000)
		
		noiseZ := perlinNoise3D(site.X*params.NoiseScale, 
			site.Y*params.NoiseScale, 
			site.Z*params.NoiseScale, 
			params.ElevationSeed+5000)
		
		// Calculate turbulence magnitude
		turbulence := math.Sqrt(noiseX*noiseX + noiseY*noiseY + noiseZ*noiseZ)
		
		turbulenceNoise[siteIdx] = turbulence * params.NoiseAmplitude * 0.4
	}
	
	return turbulenceNoise
}

// CombineNoiseTypes combines different noise types for realistic terrain
func CombineNoiseTypes(icosphereSites []Vector3D, tectonicData *TectonicsData, params ElevationParameters) []float64 {
	combinedNoise := make([]float64, len(icosphereSites))
	
	// Generate different noise types
	fractalNoise := GenerateElevationNoise(icosphereSites, tectonicData, params)
	ridgedNoise := GenerateRidgedNoise(icosphereSites, params)
	billowNoise := GenerateBillowNoise(icosphereSites, params)
	turbulenceNoise := GenerateTurbulenceNoise(icosphereSites, params)
	
	for siteIdx := range icosphereSites {
		siteID := int32(siteIdx)
		
		// Determine noise mixing based on tectonic context
		mixingWeights := calculateNoiseMixingWeights(siteID, tectonicData, params)
		
		// Combine noise types
		combined := fractalNoise[siteIdx]*mixingWeights.FractalWeight +
			ridgedNoise[siteIdx]*mixingWeights.RidgedWeight +
			billowNoise[siteIdx]*mixingWeights.BillowWeight +
			turbulenceNoise[siteIdx]*mixingWeights.TurbulenceWeight
		
		combinedNoise[siteIdx] = combined
	}
	
	return combinedNoise
}

// calculateNoiseMixingWeights determines how to mix different noise types
func calculateNoiseMixingWeights(siteID int32, tectonicData *TectonicsData, params ElevationParameters) NoiseMixingWeights {
	plateID := tectonicData.SitePlateIDs[siteID]
	plate := findPlateByID(tectonicData.Plates, plateID)
	
	weights := NoiseMixingWeights{
		FractalWeight:    0.5,
		RidgedWeight:     0.2,
		BillowWeight:     0.2,
		TurbulenceWeight: 0.1,
	}
	
	if plate == nil {
		return weights
	}
	
	// Adjust weights based on plate type
	switch plate.PlateType {
	case tectonics.ContinentalPlate:
		// Continental terrain benefits from ridged noise for mountains
		weights.RidgedWeight = 0.4
		weights.BillowWeight = 0.3
		weights.FractalWeight = 0.2
		weights.TurbulenceWeight = 0.1
		
	case tectonics.OceanicPlate:
		// Oceanic terrain is generally smoother
		weights.FractalWeight = 0.6
		weights.BillowWeight = 0.3
		weights.RidgedWeight = 0.05
		weights.TurbulenceWeight = 0.05
	}
	
	// Adjust for boundary proximity
	boundaryType, exists := tectonicData.SiteBoundaryTypes[siteID]
	if exists {
		switch boundaryType {
		case tectonics.Convergent: // Convergent - increase ridged noise for mountains
			weights.RidgedWeight *= 2.0
			weights.TurbulenceWeight *= 1.5
			
		case tectonics.Divergent: // Divergent - increase fractal noise for rifting
			weights.FractalWeight *= 1.5
			weights.TurbulenceWeight *= 1.2
			
		case tectonics.Passive: // Transform/Passive - increase turbulence
			weights.TurbulenceWeight *= 2.0
		}
		
		// Normalize weights
		total := weights.FractalWeight + weights.RidgedWeight + weights.BillowWeight + weights.TurbulenceWeight
		if total > 0 {
			weights.FractalWeight /= total
			weights.RidgedWeight /= total
			weights.BillowWeight /= total
			weights.TurbulenceWeight /= total
		}
	}
	
	return weights
}

// NoiseMixingWeights contains weights for combining different noise types
type NoiseMixingWeights struct {
	FractalWeight    float64 // Weight for fractal/Perlin noise
	RidgedWeight     float64 // Weight for ridged multifractal
	BillowWeight     float64 // Weight for billow noise
	TurbulenceWeight float64 // Weight for turbulence
}

// GenerateDetailNoise adds fine-scale terrain detail including coastline fractalization
func GenerateDetailNoise(icosphereSites []Vector3D, baseElevations []float64, params ElevationParameters) []float64 {
	detailNoise := make([]float64, len(icosphereSites))

	if params.NoiseAmplitude <= 0 {
		return detailNoise
	}

	for siteIdx, site := range icosphereSites {
		// High-frequency detail noise
		detailValue := perlinNoise3D(
			site.X*params.NoiseScale*4.0,
			site.Y*params.NoiseScale*4.0,
			site.Z*params.NoiseScale*4.0,
			params.ElevationSeed+6000,
		)

		// Determine elevation-dependent scaling
		elevationFactor := 1.0
		if len(baseElevations) > siteIdx {
			elevation := baseElevations[siteIdx]

			// COASTLINE FRACTALIZATION
			// Add maximum detail near sea level (±500m) for fractal coastlines
			// Research shows coastlines have fractal dimension 1.1-1.5
			distanceFromSeaLevel := math.Abs(elevation)

			if distanceFromSeaLevel < 500.0 {
				// Within coastal zone: maximum detail
				// Scale from 3.0 at sea level to 1.0 at ±500m
				coastalFactor := 3.0 - (2.0 * distanceFromSeaLevel / 500.0)
				elevationFactor = coastalFactor
			} else if elevation > 2000.0 {
				// High mountains: coarser but higher amplitude features
				elevationFactor = 1.5
			} else {
				// Lowlands: moderate detail
				elevationFactor = 1.0
			}
		}

		detailNoise[siteIdx] = detailValue * params.NoiseAmplitude * 0.3 * elevationFactor
	}

	return detailNoise
}

// ValidateNoiseEffects performs validation on noise generation
func ValidateNoiseEffects(noiseEffects []float64, params ElevationParameters) (NoiseMetrics, []string) {
	var metrics NoiseMetrics
	var warnings []string
	
	if len(noiseEffects) == 0 {
		warnings = append(warnings, "No noise effects generated")
		return metrics, warnings
	}
	
	// Calculate noise statistics
	minNoise := noiseEffects[0]
	maxNoise := noiseEffects[0]
	totalNoise := 0.0
	
	for _, noise := range noiseEffects {
		if noise < minNoise {
			minNoise = noise
		}
		if noise > maxNoise {
			maxNoise = noise
		}
		totalNoise += math.Abs(noise)
	}
	
	metrics.MinNoiseValue = minNoise
	metrics.MaxNoiseValue = maxNoise
	metrics.NoiseRange = maxNoise - minNoise
	metrics.MeanAbsNoise = totalNoise / float64(len(noiseEffects))
	
	// Calculate standard deviation
	meanNoise := calculateMean(noiseEffects)
	variance := 0.0
	for _, noise := range noiseEffects {
		diff := noise - meanNoise
		variance += diff * diff
	}
	metrics.NoiseStdDev = math.Sqrt(variance / float64(len(noiseEffects)))
	
	// Validation checks
	if metrics.NoiseRange > params.NoiseAmplitude*10.0 {
		warnings = append(warnings, "Noise range is much larger than expected amplitude")
	}
	
	if metrics.MeanAbsNoise < params.NoiseAmplitude*0.1 {
		warnings = append(warnings, "Mean noise magnitude is very low")
	}
	
	if metrics.NoiseStdDev < params.NoiseAmplitude*0.2 {
		warnings = append(warnings, "Noise has very low variation")
	}
	
	return metrics, warnings
}

// NoiseMetrics contains statistics about noise effects
type NoiseMetrics struct {
	MinNoiseValue float64 // Minimum noise value
	MaxNoiseValue float64 // Maximum noise value
	NoiseRange    float64 // Range of noise values
	MeanAbsNoise  float64 // Mean absolute noise magnitude
	NoiseStdDev   float64 // Standard deviation of noise
}