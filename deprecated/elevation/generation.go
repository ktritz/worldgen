package elevation

// DEPRECATED: This elevation package is replaced by terrain.GenerateCollisionElevation
// which uses the simpler Red Blob Games approach:
// - Elevation driven by plate collision physics
// - Distance fields from mountain/coastline/ocean seeds
// - Guaranteed land coverage via threshold adjustment
// - Real meters scaling
//
// New code should use:
//   terrain.GenerateCollisionElevation(sites, cells, plates, siteToPlate, config)
//
// This package remains for reference but is no longer actively maintained.

import (
	"fmt"
	"math/rand"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
)

// GenerateElevation creates comprehensive elevation data for all icosphere sites using modular generation.
// This function orchestrates multiple elevation generation modules:
// - Base elevation from plate types (continental vs oceanic)
// - Volcanic elevation from hotspots and volcanic features
// - Seafloor elevation from age-depth relationships
// - Ridge elevation from mid-ocean ridge topography
// - Tectonic elevation from boundary interactions
// - Erosion effects from age-based weathering
// - Fractal noise for terrain detail
func GenerateElevation(
	icosphereSites []icosphere.Vector3D,
	tectonicData *tectonics.TectonicsData,
	settings ElevationSettings,
	globalSeed int64,
	planetRadius float64,
) (*ElevationData, error) {

	fmt.Println("-- Running Advanced Elevation Module --")
	
	// Convert icosphere sites to internal Vector3D format (which is an alias)
	sites := make([]Vector3D, len(icosphereSites))
	for i, site := range icosphereSites {
		sites[i] = Vector3D{X: site.X, Y: site.Y, Z: site.Z}
	}
	
	// Convert tectonics data to internal format
	internalTectonicData := convertTectonicsData(tectonicData)
	
	// Prepare elevation parameters
	params := prepareElevationParameters(settings, globalSeed, planetRadius)
	
	// Initialize elevation data with all component arrays
	elevationData := &ElevationData{
		CellElevations:      make(map[int32]float64),
		BaseElevations:      make([]float64, len(sites)),
		VolcanicElevations:  make([]float64, len(sites)),
		SeafloorElevations:  make([]float64, len(sites)),
		RidgeElevations:     make([]float64, len(sites)),
		TectonicElevations:  make([]float64, len(sites)),
		ErosionElevations:   make([]float64, len(sites)),
		NoiseElevations:     make([]float64, len(sites)),
		SiteElevations:      make([]float64, len(sites)),
	}

	fmt.Println("  Generating base elevations from plate types...")
	elevationData.BaseElevations = GenerateBaseElevations(sites, internalTectonicData, params)
	
	fmt.Println("  Generating volcanic elevations from hotspots...")
	elevationData.VolcanicElevations = GenerateVolcanicElevations(sites, internalTectonicData, params)
	
	fmt.Println("  Generating seafloor elevations from age-depth relationships...")
	elevationData.SeafloorElevations = GenerateSeafloorElevations(sites, internalTectonicData, params)
	
	fmt.Println("  Generating ridge elevations from mid-ocean ridges...")
	elevationData.RidgeElevations = GenerateRidgeElevations(sites, internalTectonicData, params)
	
	fmt.Println("  Generating tectonic elevations from boundary effects...")
	elevationData.TectonicElevations = GenerateTectonicElevations(sites, internalTectonicData, params)
	
	if settings.EnableErosion {
		fmt.Println("  Generating erosion effects from age-based weathering...")
		elevationData.ErosionElevations = GenerateErosionEffects(sites, internalTectonicData, params)
	}
	
	if params.NoiseAmplitude > 0 {
		fmt.Println("  Generating fractal noise for terrain detail...")
		elevationData.NoiseElevations = CombineNoiseTypes(sites, internalTectonicData, params)
	}

	// Combine all elevation components
	fmt.Println("  Combining elevation components...")
	for i := range sites {
		totalElevation := elevationData.BaseElevations[i] +
			elevationData.VolcanicElevations[i] +
			elevationData.SeafloorElevations[i] +
			elevationData.RidgeElevations[i] +
			elevationData.TectonicElevations[i] +
			elevationData.ErosionElevations[i] +
			elevationData.NoiseElevations[i]

		elevationData.SiteElevations[i] = totalElevation
		elevationData.CellElevations[int32(i)] = totalElevation
	}

	// Add fractal displacement to create realistic coastlines
	fmt.Println("  Applying fractal displacement to boundaries for realistic coastlines...")
	fractalConfig := tectonics.FractalDisplacementConfig{
		SubdivisionDepth:    6,       // 6 octaves for fine detail
		DisplacementScale:   0.8,     // 80% displacement
		HurstExponent:       0.5,     // Standard coastline roughness (H=0.5)
		MinEdgeLength:       5.0,     // 5km minimum
		ElevationAdjustment: 1200.0,  // ±1200m base amplitude
	}

	// Create RNG for fractal displacement
	fractalRng := rand.New(rand.NewSource(params.ElevationSeed + 9000))

	// Apply fractal displacement to create jagged coastlines
	adjustedElevations := tectonics.ApplyFractalDisplacementToElevations(
		sites,
		internalTectonicData.SitePlateIDs,
		elevationData.SiteElevations,
		internalTectonicData.Plates,
		internalTectonicData.IsBoundarySite,
		internalTectonicData.SiteBoundaryTypes,
		fractalConfig,
		params.PlanetRadius,
		fractalRng,
	)

	// Update elevations with fractal adjustments
	for i := range sites {
		fractalAdjustment := adjustedElevations[i] - elevationData.SiteElevations[i]
		elevationData.SiteElevations[i] = adjustedElevations[i]
		elevationData.CellElevations[int32(i)] = adjustedElevations[i]
		elevationData.NoiseElevations[i] += fractalAdjustment // Track as noise contribution
	}

	// Measure and report coastline fractal dimension
	PrintCoastlineFractalReport(sites, elevationData.SiteElevations, params.PlanetRadius)
	
	// Perform validation
	fmt.Println("  Validating elevation system...")
	validationReport := ValidateElevationSystem(elevationData, internalTectonicData, params)
	
	// Print validation summary
	printValidationSummary(validationReport, len(sites))
	
	fmt.Println("-- Advanced Elevation Module Complete --")
	
	return elevationData, nil
}

// prepareElevationParameters computes comprehensive elevation parameters from settings
func prepareElevationParameters(settings ElevationSettings, globalSeed int64, planetRadius float64) ElevationParameters {
	// Set default values if not specified in settings
	elevationMultiplier := settings.ElevationMultiplier
	if elevationMultiplier == 0 {
		elevationMultiplier = 1.0
	}

	// Convert relative distances to absolute distances
	characteristicFalloffDistAbs := settings.CharacteristicFalloffDistance * planetRadius
	if characteristicFalloffDistAbs == 0 {
		characteristicFalloffDistAbs = 0.15 * planetRadius
	}

	maxBoundaryEffectDistAbs := settings.MaxBoundaryEffectDistance * planetRadius
	if maxBoundaryEffectDistAbs == 0 {
		maxBoundaryEffectDistAbs = characteristicFalloffDistAbs * 3.0
	}

	convergentStrength := settings.ConvergentBoundaryStrength
	if convergentStrength == 0 {
		convergentStrength = 2000.0 * elevationMultiplier
	}

	divergentStrength := settings.DivergentBoundaryStrength
	if divergentStrength == 0 {
		divergentStrength = 500.0 * elevationMultiplier
	}

	return ElevationParameters{
		// Basic parameters
		BaseAmplitude:                elevationMultiplier,
		ElevationSeed:                globalSeed,
		PlanetRadius:                 planetRadius,
		
		// Boundary effect parameters
		CharacteristicFalloffDistAbs: characteristicFalloffDistAbs,
		MaxBoundaryEffectDistAbs:     maxBoundaryEffectDistAbs,
		ConvergentStrength:           convergentStrength,
		DivergentStrength:            divergentStrength,
		
		// Volcanic parameters
		VolcanicMultiplier:           settings.VolcanicElevationMultiplier,
		HotspotInfluenceRadiusAbs:    settings.HotspotInfluenceRadius * 1000.0,
		
		// Ridge parameters
		RidgeElevation:               settings.RidgeElevationAboveSeafloor,
		RidgeInfluenceDistAbs:        settings.RidgeInfluenceDistance * 1000.0,
		
		// Seafloor parameters
		SeafloorModel:                settings.SeafloorModel,
		
		// Erosion parameters
		ErosionRate:                 settings.ErosionRate,
		MaxErosionAge:               settings.MaxErosionAge,
		
		// Noise parameters
		NoiseAmplitude:              settings.NoiseAmplitude,
		NoiseScale:                  settings.NoiseScale,
		NoiseOctaves:                settings.NoiseOctaves,
		NoiseLacunarity:             settings.NoiseLacunarity,
		NoisePersistence:            settings.NoisePersistence,
		
		// Subduction zone parameters
		TrenchDepthMultiplier:       settings.TrenchDepthMultiplier,
		ArcElevation:                settings.VolcanicArcElevation,
		
		// Isostatic parameters
		CrustalDensity:              settings.CrustalDensity,
		MantleDensity:               settings.MantleDensity,
	}
}

// Helper functions for default values

func createDefaultSeafloorAgeModel() SeafloorAgeModel {
	// Return a default seafloor age model using tectonics.DefaultSeafloorAgeModel
	return tectonics.DefaultSeafloorAgeModel()
}

// ConvertElevationSettings converts ElevationSettings to ElevationParameters
func ConvertElevationSettings(settings *ElevationSettings) ElevationParameters {
	// Convert settings to parameters using the existing logic
	elevationMultiplier := getFloatOrDefault(settings.ElevationMultiplier, 1.0)
	planetRadius := 6371000.0 // Default Earth radius
	
	// Calculate absolute distances from relative settings
	characteristicFalloffDistAbs := settings.CharacteristicFalloffDistance * planetRadius
	maxBoundaryEffectDistAbs := settings.MaxBoundaryEffectDistance * planetRadius
	
	// Default boundary strengths if not specified
	convergentStrength := getFloatOrDefault(settings.ConvergentBoundaryStrength, 2000.0)
	divergentStrength := getFloatOrDefault(settings.DivergentBoundaryStrength, 800.0)
	
	return ElevationParameters{
		// Basic parameters
		BaseAmplitude:                elevationMultiplier,
		ElevationSeed:                settings.GlobalSeed,
		PlanetRadius:                 planetRadius,
		
		// Boundary effect parameters
		CharacteristicFalloffDistAbs: characteristicFalloffDistAbs,
		MaxBoundaryEffectDistAbs:     maxBoundaryEffectDistAbs,
		ConvergentStrength:           convergentStrength,
		DivergentStrength:            divergentStrength,
		
		// Volcanic parameters
		VolcanicMultiplier:           settings.VolcanicElevationMultiplier,
		HotspotInfluenceRadiusAbs:    settings.HotspotInfluenceRadius * 1000.0,
		
		// Ridge parameters
		RidgeElevation:               settings.RidgeElevationAboveSeafloor,
		RidgeInfluenceDistAbs:        settings.RidgeInfluenceDistance * 1000.0,
		
		// Seafloor parameters
		SeafloorModel:                settings.SeafloorModel,
		
		// Subduction zone parameters
		TrenchDepthMultiplier:        settings.TrenchDepthMultiplier,
		ArcElevation:                 settings.VolcanicArcElevation,
		
		// Erosion parameters
		ErosionRate:                  settings.ErosionRate,
		MaxErosionAge:                settings.MaxErosionAge,
		
		// Noise parameters
		NoiseAmplitude:               settings.NoiseAmplitude,
		NoiseScale:                   settings.NoiseScale,
		NoiseOctaves:                 settings.NoiseOctaves,
		NoiseLacunarity:              settings.NoiseLacunarity,
		NoisePersistence:             settings.NoisePersistence,
		
		// Isostatic parameters
		CrustalDensity:               settings.CrustalDensity,
		MantleDensity:                settings.MantleDensity,
	}
}

// convertTectonicsData converts external tectonics data to internal format
func convertTectonicsData(tectonicData *tectonics.TectonicsData) *TectonicsData {
	// Since TectonicsData is now an alias to tectonics.TectonicsData, we can use it directly
	// We just need to ensure the conversion handles any differences in the internal structures
	return (*TectonicsData)(tectonicData)
}


// printValidationSummary prints a summary of the elevation validation results
func printValidationSummary(report ElevationValidationReport, numSites int) {
	fmt.Printf("  Validation Results:\n")
	fmt.Printf("    Overall Quality Score: %.2f/1.0\n", report.OverallQuality)
	fmt.Printf("    Elevation Range: %.1fm to %.1fm (%.1fm total)\n", 
		report.BasicStats.MinElevation, report.BasicStats.MaxElevation, report.BasicStats.ElevationRange)
	fmt.Printf("    Land Coverage: %.1f%% (Mean elevation: %.1fm)\n", 
		report.BasicStats.LandPercentage, report.BasicStats.MeanLandElevation)
	fmt.Printf("    Ocean Coverage: %.1f%% (Mean depth: %.1fm)\n", 
		100.0-report.BasicStats.LandPercentage, report.BasicStats.MeanOceanDepth)
	
	if len(report.Warnings) > 0 {
		fmt.Printf("    Warnings: %d\n", len(report.Warnings))
		for _, warning := range report.Warnings {
			fmt.Printf("      - %s\n", warning)
		}
	}
	
	fmt.Printf("    Component Contributions:\n")
	fmt.Printf("      Base: %.1fm, Volcanic: %.1fm, Seafloor: %.1fm\n",
		report.ComponentAnalysis.BaseElevationContribution,
		report.ComponentAnalysis.VolcanicContribution,
		report.ComponentAnalysis.SeafloorContribution)
	fmt.Printf("      Ridge: %.1fm, Tectonic: %.1fm, Erosion: %.1fm, Noise: %.1fm\n",
		report.ComponentAnalysis.RidgeContribution,
		report.ComponentAnalysis.TectonicContribution,
		report.ComponentAnalysis.ErosionContribution,
		report.ComponentAnalysis.NoiseContribution)
}