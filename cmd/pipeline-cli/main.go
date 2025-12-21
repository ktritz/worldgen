package main

// Pipeline CLI tool for modular world generation

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"worldgen/landgen"
	"worldgen/landgen/elevation"
	"worldgen/landgen/mapper"
	// "worldgen/landgen/tectonics" // Removed unused import
)

func main() {
	// Command line flags
	var (
		configFile     = flag.String("config", "", "Configuration file (JSON)")
		// pipelineFile   = flag.String("pipeline", "", "Pipeline state file (JSON)") // Removed unused
		cacheDir       = flag.String("cache", "cache", "Cache directory for intermediate data")
		outputDir      = flag.String("output", "output", "Output directory")
		
		// Stage control flags
		runIcosphere   = flag.Bool("icosphere", false, "Run icosphere generation stage")
		runTectonics   = flag.Bool("tectonics", false, "Run tectonics generation stage")
		runElevation   = flag.Bool("elevation", false, "Run elevation generation stage")
		runAll         = flag.Bool("all", false, "Run all stages")
		
		// Force regeneration flags
		forceIcosphere = flag.Bool("force-icosphere", false, "Force regeneration of icosphere stage")
		forceTectonics = flag.Bool("force-tectonics", false, "Force regeneration of tectonics stage")
		forceElevation = flag.Bool("force-elevation", false, "Force regeneration of elevation stage")
		forceAll       = flag.Bool("force-all", false, "Force regeneration of all stages")
		
		// Configuration override flags
		seed           = flag.Int64("seed", 0, "Global seed (overrides config)")
		subdivision    = flag.Int("subdiv", 0, "Icosphere subdivision level (overrides config)")
		
		// Output flags
		status         = flag.Bool("status", false, "Show pipeline status")
		savePipeline   = flag.String("save-pipeline", "", "Save pipeline state to file")
		loadPipeline   = flag.String("load-pipeline", "", "Load pipeline state from file")
		generateMap    = flag.Bool("elevation-map", false, "Generate 2D elevation map (requires completed elevation stage)")
		generateTecMap = flag.Bool("tectonics-map", false, "Generate 2D tectonics map (requires completed tectonics stage)")
		generatePlateMap = flag.Bool("plate-map", false, "Generate 2D plate ID map with distinct colors (requires completed tectonics stage)")
		plateLookup    = flag.String("lookup-plate", "", "Look up plate ID at pixel coordinates (format: x,y, e.g., '350,280')")
		
		// Example configurations
		createExample  = flag.String("create-example", "", "Create example configuration file")
	)
	flag.Parse()

	// Handle example configuration creation
	if *createExample != "" {
		if err := createExampleConfig(*createExample); err != nil {
			log.Fatalf("Failed to create example config: %v", err)
		}
		fmt.Printf("Example configuration saved to: %s\n", *createExample)
		return
	}

	// Create or load pipeline
	var pipeline *landgen.Pipeline
	var err error

	if *loadPipeline != "" {
		fmt.Printf("Loading pipeline from: %s\n", *loadPipeline)
		pipeline, err = landgen.LoadPipelineFromFile(*loadPipeline)
		if err != nil {
			log.Fatalf("Failed to load pipeline: %v", err)
		}
	} else {
		// Create new pipeline with configuration
		config := loadConfiguration(*configFile, *seed, *subdivision, *outputDir)
		pipeline = landgen.NewPipeline(config, *cacheDir)
	}

	// Show status if requested
	if *status {
		showPipelineStatus(pipeline)
		return
	}

	// Determine force regeneration settings
	forceRegenerate := map[string]bool{
		"icosphere": *forceIcosphere || *forceAll,
		"tectonics": *forceTectonics || *forceAll,
		"elevation": *forceElevation || *forceAll,
	}

	// Run requested stages
	if *runAll {
		fmt.Println("Running full pipeline...")
		if err := pipeline.RunFullPipeline(forceRegenerate); err != nil {
			log.Fatalf("Pipeline failed: %v", err)
		}
	} else {
		// Run individual stages
		if *runIcosphere {
			if err := pipeline.RunIcosphereStage(forceRegenerate["icosphere"]); err != nil {
				log.Fatalf("Icosphere stage failed: %v", err)
			}
		}
		
		if *runTectonics {
			if err := pipeline.RunTectonicsStage(forceRegenerate["tectonics"]); err != nil {
				log.Fatalf("Tectonics stage failed: %v", err)
			}
		}
		
		if *runElevation {
			if err := pipeline.RunElevationStage(forceRegenerate["elevation"]); err != nil {
				log.Fatalf("Elevation stage failed: %v", err)
			}
		}
	}
	
	// Generate elevation map if requested
	if *generateMap {
		fmt.Println("=== GENERATING ELEVATION MAP ===")
		if err := generateElevationMap(pipeline); err != nil {
			log.Fatalf("Elevation map generation failed: %v", err)
		}
		fmt.Println("Elevation map generated successfully")
	}
	
	// Generate tectonics map if requested
	if *generateTecMap {
		fmt.Println("=== GENERATING TECTONICS MAP ===")
		if err := generateTectonicsMap(pipeline); err != nil {
			log.Fatalf("Tectonics map generation failed: %v", err)
		}
		fmt.Println("Tectonics map generated successfully")
	}

	// Generate plate ID map if requested
	if *generatePlateMap {
		fmt.Println("=== GENERATING PLATE ID MAP ===")
		if err := generatePlateIDMap(pipeline); err != nil {
			log.Fatalf("Plate ID map generation failed: %v", err)
		}
		fmt.Println("Plate ID map generated successfully")
	}

	// Look up plate ID at coordinates if requested
	if *plateLookup != "" {
		fmt.Println("=== PLATE LOOKUP ===")
		if err := lookupPlateAtCoordinates(pipeline, *plateLookup); err != nil {
			log.Fatalf("Plate lookup failed: %v", err)
		}
	}

	// Save pipeline state if requested
	if *savePipeline != "" {
		fmt.Printf("Saving pipeline to: %s\n", *savePipeline)
		if err := pipeline.SavePipelineToFile(*savePipeline); err != nil {
			log.Fatalf("Failed to save pipeline: %v", err)
		}
	}

	// Show final status
	showPipelineStatus(pipeline)
}

// loadConfiguration loads and creates pipeline configuration
func loadConfiguration(configFile string, seedOverride int64, subdivOverride int, outputDirOverride string) *landgen.PipelineConfig {
	var config *landgen.PipelineConfig

	if configFile != "" {
		fmt.Printf("Loading configuration from: %s\n", configFile)
		
		data, err := os.ReadFile(configFile)
		if err != nil {
			log.Fatalf("Failed to read config file: %v", err)
		}
		
		config = &landgen.PipelineConfig{}
		if err := json.Unmarshal(data, config); err != nil {
			log.Fatalf("Failed to parse config file: %v", err)
		}
	} else {
		fmt.Println("Using default configuration")
		config = landgen.DefaultPipelineConfig()
	}

	// Apply command line overrides
	if seedOverride != 0 {
		config.GlobalSeed = seedOverride
		fmt.Printf("Overriding seed: %d\n", seedOverride)
	}
	
	if subdivOverride != 0 {
		config.IcosphereSubdiv = subdivOverride
		fmt.Printf("Overriding subdivision: %d\n", subdivOverride)
	}
	
	if outputDirOverride != "" {
		config.OutputDir = outputDirOverride
		fmt.Printf("Overriding output directory: %s\n", outputDirOverride)
	}

	return config
}

// showPipelineStatus displays the current pipeline status
func showPipelineStatus(pipeline *landgen.Pipeline) {
	status := pipeline.GetStatus()
	
	fmt.Println("\n=== PIPELINE STATUS ===")
	fmt.Printf("Icosphere: %s", getStatusString(status.IcosphereCompleted))
	if status.IcosphereCompleted {
		fmt.Printf(" (%d sites)", status.IcosphereSites)
	}
	fmt.Println()
	
	fmt.Printf("Tectonics: %s", getStatusString(status.TectonicsCompleted))
	if status.TectonicsCompleted {
		fmt.Printf(" (%d plates, score: %.3f)", status.TectonicsPlates, status.TectonicsScore)
	}
	fmt.Println()
	
	fmt.Printf("Elevation: %s", getStatusString(status.ElevationCompleted))
	if status.ElevationCompleted {
		fmt.Printf(" (quality: %.3f)", status.ElevationQuality)
	}
	fmt.Println()
	
	fmt.Println("========================")
}

// getStatusString returns a colored status string
func getStatusString(completed bool) string {
	if completed {
		return "✓ COMPLETED"
	}
	return "✗ PENDING"
}

// createExampleConfig creates an example configuration file
func createExampleConfig(filename string) error {
	// Create different example configurations
	var config *landgen.PipelineConfig
	
	// Determine which type of example based on filename
	basename := strings.ToLower(filepath.Base(filename))
	
	switch {
	case strings.Contains(basename, "earth"):
		config = createEarthLikeConfig()
	case strings.Contains(basename, "archipelago"):
		config = createArchipelagoConfig()
	case strings.Contains(basename, "continental"):
		config = createContinentalConfig()
	case strings.Contains(basename, "quick"):
		config = createQuickTestConfig()
	default:
		config = landgen.DefaultPipelineConfig()
	}

	// Save to file
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// createEarthLikeConfig creates Earth-like world configuration
func createEarthLikeConfig() *landgen.PipelineConfig {
	config := landgen.DefaultPipelineConfig()
	config.GlobalSeed = 12345
	config.IcosphereSubdiv = 6
	
	// Earth-like tectonics settings
	config.TectonicsSettings.Seed = 12345
	config.TectonicsSettings.NumPlates = 12
	config.TectonicsSettings.TargetContinentalProportion = 0.6
	config.TectonicsSettings.BoundaryStyle = "continent"
	
	// Earth-like elevation settings
	config.ElevationSettings = elevation.DefaultElevationSettings()
	
	return config
}

// createArchipelagoConfig creates archipelago world configuration
func createArchipelagoConfig() *landgen.PipelineConfig {
	config := landgen.DefaultPipelineConfig()
	config.GlobalSeed = 54321
	config.IcosphereSubdiv = 6
	
	// Archipelago tectonics settings
	config.TectonicsSettings.Seed = 54321
	config.TectonicsSettings.NumPlates = 20
	config.TectonicsSettings.TargetContinentalProportion = 0.8
	config.TectonicsSettings.BoundaryStyle = "archipelago"
	
	// Archipelago elevation settings
	config.ElevationSettings = elevation.ArchipelagoElevationSettings()
	
	return config
}

// createContinentalConfig creates continental world configuration
func createContinentalConfig() *landgen.PipelineConfig {
	config := landgen.DefaultPipelineConfig()
	config.GlobalSeed = 99999
	config.IcosphereSubdiv = 6
	
	// Continental tectonics settings
	config.TectonicsSettings.Seed = 99999
	config.TectonicsSettings.NumPlates = 8
	config.TectonicsSettings.TargetContinentalProportion = 0.9
	config.TectonicsSettings.BoundaryStyle = "continent"
	
	// Continental elevation settings
	config.ElevationSettings = elevation.DefaultElevationSettings()
	config.ElevationSettings.ElevationMultiplier = 1.5 // Higher mountains for continental world
	
	return config
}

// createQuickTestConfig creates a quick test configuration
func createQuickTestConfig() *landgen.PipelineConfig {
	config := landgen.DefaultPipelineConfig()
	config.GlobalSeed = 42
	config.IcosphereSubdiv = 4 // Smaller for faster testing
	
	// Quick test settings - simplified
	config.TectonicsSettings.Seed = 42
	config.TectonicsSettings.NumPlates = 8
	config.TectonicsSettings.TargetContinentalProportion = 0.5
	config.TectonicsSettings.BoundaryStyle = "continent"
	
	// Simplified elevation settings
	config.ElevationSettings = elevation.DefaultElevationSettings()
	config.ElevationSettings.NoiseOctaves = 4 // Fewer octaves for speed
	
	return config
}

// Example usage function
func printUsageExamples() {
	fmt.Println(`
Pipeline CLI Usage Examples:

1. Create example configurations:
   pipeline-cli -create-example earth_config.json
   pipeline-cli -create-example archipelago_config.json
   pipeline-cli -create-example quick_test.json

2. Run individual stages:
   pipeline-cli -config earth_config.json -icosphere
   pipeline-cli -config earth_config.json -tectonics
   pipeline-cli -config earth_config.json -elevation

3. Run full pipeline:
   pipeline-cli -config earth_config.json -all

4. Force regeneration:
   pipeline-cli -config earth_config.json -elevation -force-elevation

5. Save/load pipeline state:
   pipeline-cli -config earth_config.json -all -save-pipeline pipeline_state.json
   pipeline-cli -load-pipeline pipeline_state.json -elevation -force-elevation

6. Override settings:
   pipeline-cli -config earth_config.json -seed 12345 -subdiv 5 -all

7. Check status:
   pipeline-cli -load-pipeline pipeline_state.json -status
`)
}

// generateElevationMap creates a 2D elevation map from completed elevation data
func generateElevationMap(pipeline *landgen.Pipeline) error {
	// Check that elevation stage is complete
	if pipeline.ElevationData == nil {
		return fmt.Errorf("elevation stage must be completed before generating elevation map")
	}
	
	if pipeline.IcosphereData == nil {
		return fmt.Errorf("icosphere data required for elevation map generation")
	}
	
	// Generate elevation map using the new generic mapper
	outputPath := "elevation_map.png"
	if pipeline.Config.OutputDir != "" {
		outputPath = filepath.Join(pipeline.Config.OutputDir, "elevation_map.png")
	}
	
	// Debug: Check elevation range and distribution
	elevations := pipeline.ElevationData.ElevationData.SiteElevations
	minElev, maxElev := elevations[0], elevations[0]
	landCount, oceanCount := 0, 0
	deepOceanCount, shallowOceanCount := 0, 0
	lowLandCount, highLandCount := 0, 0
	
	for _, elev := range elevations {
		if elev < minElev {
			minElev = elev
		}
		if elev > maxElev {
			maxElev = elev
		}
		if elev >= 0 {
			landCount++
			if elev > 2000 {
				highLandCount++
			} else {
				lowLandCount++
			}
		} else {
			oceanCount++
			if elev < -2000 {
				deepOceanCount++
			} else {
				shallowOceanCount++
			}
		}
	}
	fmt.Printf("Elevation range: %.1fm to %.1fm\n", minElev, maxElev)
	fmt.Printf("Land: %d sites (%.1f%%) - Low land: %d, High land: %d\n", 
		landCount, float64(landCount)/float64(len(elevations))*100, lowLandCount, highLandCount)
	fmt.Printf("Ocean: %d sites (%.1f%%) - Shallow: %d, Deep: %d\n", 
		oceanCount, float64(oceanCount)/float64(len(elevations))*100, shallowOceanCount, deepOceanCount)
	
	// Check ocean depth distribution more carefully
	var depthRanges [5]int // 0-1000m, 1000-3000m, 3000-6000m, 6000-12000m, 12000m+
	for _, elev := range elevations {
		if elev < 0 {
			depth := -elev
			if depth < 1000 {
				depthRanges[0]++
			} else if depth < 3000 {
				depthRanges[1]++
			} else if depth < 6000 {
				depthRanges[2]++
			} else if depth < 12000 {
				depthRanges[3]++
			} else {
				depthRanges[4]++
			}
		}
	}
	fmt.Printf("Ocean depth ranges: 0-1km: %d, 1-3km: %d, 3-6km: %d, 6-12km: %d, 12km+: %d\n",
		depthRanges[0], depthRanges[1], depthRanges[2], depthRanges[3], depthRanges[4])
		
	// Also check the tectonics plate distribution
	if pipeline.TectonicsData != nil && pipeline.TectonicsData.TectonicsData != nil {
		plates := pipeline.TectonicsData.TectonicsData.Plates
		continentalPlates, oceanicPlates := 0, 0
		for _, plate := range plates {
			if plate.PlateType == "Continental" {
				continentalPlates++
			} else {
				oceanicPlates++
			}
		}
		fmt.Printf("Tectonic plates: %d continental, %d oceanic (%.1f%% continental plates)\n", 
			continentalPlates, oceanicPlates, float64(continentalPlates)/float64(len(plates))*100)
	}
	
	fmt.Printf("Generating elevation map: %s\n", outputPath)
	
	// Create quick map settings
	settings := mapper.DefaultQuickMapSettings()
	settings.OutputPath = outputPath
	settings.ColorScheme = "elevation"
	settings.Width = 1024
	settings.Height = 512
	
	// Generate map using the generic mapper
	err := mapper.GenerateElevationMap(
		pipeline.IcosphereData.Sites,
		pipeline.ElevationData.ElevationData,
		settings,
	)
	
	if err != nil {
		return fmt.Errorf("failed to generate elevation map: %w", err)
	}
	
	return nil
}

// generateTectonicsMap creates a 2D tectonics map from completed tectonics data
func generateTectonicsMap(pipeline *landgen.Pipeline) error {
	// Check that tectonics stage is complete
	if pipeline.TectonicsData == nil {
		return fmt.Errorf("tectonics stage must be completed before generating tectonics map")
	}
	
	if pipeline.IcosphereData == nil {
		return fmt.Errorf("icosphere data required for tectonics map generation")
	}
	
	// Generate tectonics map using the generic mapper
	outputPath := "tectonics_map.png"
	if pipeline.Config.OutputDir != "" {
		outputPath = filepath.Join(pipeline.Config.OutputDir, "tectonics_map.png")
	}
	
	fmt.Printf("Generating tectonics map: %s\n", outputPath)
	
	// Create quick map settings
	settings := mapper.DefaultQuickMapSettings()
	settings.OutputPath = outputPath
	settings.ColorScheme = "tectonics"
	settings.Width = 1024
	settings.Height = 512
	
	// Generate map using the generic mapper
	err := mapper.GenerateTectonicsMap(
		pipeline.IcosphereData.Sites,
		pipeline.TectonicsData.TectonicsData,
		settings,
	)
	
	if err != nil {
		return fmt.Errorf("failed to generate tectonics map: %w", err)
	}
	
	return nil
}

// generatePlateIDMap creates a 2D plate ID map with distinct colors for each plate
func generatePlateIDMap(pipeline *landgen.Pipeline) error {
	// Check that tectonics stage is complete
	if pipeline.TectonicsData == nil {
		return fmt.Errorf("tectonics stage must be completed before generating plate ID map")
	}
	
	if pipeline.IcosphereData == nil {
		return fmt.Errorf("icosphere data required for plate ID map generation")
	}
	
	// Generate plate ID map using the generic mapper
	outputPath := "plate_id_map.png"
	if pipeline.Config.OutputDir != "" {
		outputPath = filepath.Join(pipeline.Config.OutputDir, "plate_id_map.png")
	}
	
	fmt.Printf("Generating plate ID map: %s\n", outputPath)
	
	// Create quick map settings
	settings := mapper.DefaultQuickMapSettings()
	settings.OutputPath = outputPath
	settings.ColorScheme = "plate_id"
	settings.Width = 1024
	settings.Height = 512
	
	// Generate map using the generic mapper
	err := mapper.GeneratePlateIDMap(
		pipeline.IcosphereData.Sites,
		pipeline.TectonicsData.TectonicsData,
		settings,
	)
	
	if err != nil {
		return fmt.Errorf("failed to generate plate ID map: %w", err)
	}
	
	return nil
}

// lookupPlateAtCoordinates looks up the plate ID at the specified pixel coordinates
func lookupPlateAtCoordinates(pipeline *landgen.Pipeline, coordinates string) error {
	// Parse coordinates
	parts := strings.Split(coordinates, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid coordinate format. Expected 'x,y', got: %s", coordinates)
	}
	
	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return fmt.Errorf("invalid x coordinate: %s", parts[0])
	}
	
	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return fmt.Errorf("invalid y coordinate: %s", parts[1])
	}
	
	// Check that required stages are complete
	if pipeline.TectonicsData == nil {
		return fmt.Errorf("tectonics stage must be completed before plate lookup")
	}
	
	if pipeline.IcosphereData == nil {
		return fmt.Errorf("icosphere data required for plate lookup")
	}
	
	// Get the data we need for lookup
	tectonicsData := pipeline.TectonicsData.TectonicsData
	voronoiCells := pipeline.IcosphereData.VoronoiCells
	icosphereVertices := pipeline.IcosphereData.Sites
	
	// Convert TectonicsData to the formats expected by PlatePixelLookup
	plates := tectonicsData.Plates
	
	// Create cell assignments array from the site assignments
	cellAssignments := make([]int, len(voronoiCells))
	for cellIdx, cell := range voronoiCells {
		if int(cell.SiteIndex) < len(tectonicsData.SitePlateIDs) {
			cellAssignments[cellIdx] = int(tectonicsData.SitePlateIDs[cell.SiteIndex])
		}
	}
	
	// Create plate lookup utility (map dimensions match plate ID map generation)
	lookup := NewPlatePixelLookup(
		plates, 
		cellAssignments, 
		voronoiCells, 
		icosphereVertices,
		1024, 512, // Match the plate map dimensions
		pipeline.Config.PlanetRadius,
	)
	
	// Look up the plate information
	info, err := lookup.GetPlateInfoAtPixel(x, y)
	if err != nil {
		return fmt.Errorf("plate lookup failed: %w", err)
	}
	
	// Display the results
	fmt.Printf("Plate lookup at pixel coordinates (%d, %d):\n", x, y)
	fmt.Printf("  Geographic location: %.2f°S, %.2f°W\n", -info.Latitude, -info.Longitude)
	fmt.Printf("  Plate ID: %d\n", info.PlateID)
	fmt.Printf("  Plate type: %s\n", info.PlateType)
	fmt.Printf("  Plate area: %.2e m²\n", info.PlateArea)
	fmt.Printf("  Plate rotation speed: %.4f rad/Myr\n", info.RotationSpeed)
	fmt.Printf("  Plate center: (%.3f, %.3f, %.3f)\n", info.PlateCenter.X, info.PlateCenter.Y, info.PlateCenter.Z)
	
	return nil
}

// showHelp displays help information
func showHelp() {
	fmt.Println("Pipeline CLI - Modular World Generation Tool")
	fmt.Println("============================================")
	flag.PrintDefaults()
	printUsageExamples()
}

func init() {
	// Override default help
	flag.Usage = showHelp
}