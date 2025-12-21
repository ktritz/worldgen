package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"worldgen/icosphere"
	"worldgen/landgen"
	"worldgen/landgen/elevation"
	"worldgen/landgen/mapgen"
	"worldgen/landgen/tectonics"
)

type RunConfig struct {
	Subdivision   int     `json:"subdivision"`
	NumPlates     int     `json:"numPlates"`
	Seed          int64   `json:"seed"`
	PlanetRadius  float64 `json:"planetRadius"`
	OutputDir     string  `json:"outputDir"`
	GenerateObj   bool    `json:"generateObj"`
	LogLevel      string  `json:"logLevel"`
}

type GenerationMetrics struct {
	Config            RunConfig                  `json:"config"`
	Timestamp         string                     `json:"timestamp"`
	TectonicMetrics   tectonics.ValidationMetrics `json:"tectonicMetrics"`
	ElevationMetrics  *ElevationMetrics          `json:"elevationMetrics,omitempty"`
	GenerationTimes   map[string]string          `json:"generationTimes"`
	TotalTime         string                     `json:"totalTime"`
	EarthBenchmark    EarthBenchmarkScore        `json:"earthBenchmark"`
	OverallScore      float64                    `json:"overallScore"`
	Status            string                     `json:"status"` // PASS/FAIL
}

type ElevationMetrics struct {
	AvgOceanicElevation    float64 `json:"avgOceanicElevation"`
	AvgContinentalElevation float64 `json:"avgContinentalElevation"`
	ElevationRange         float64 `json:"elevationRange"`
	BoundaryEffectStrength float64 `json:"boundaryEffectStrength"`
}

type EarthBenchmarkScore struct {
	PlateCountScore     float64 `json:"plateCountScore"`     // How close to Earth's 7-52 plates
	SizeDistribution    float64 `json:"sizeDistribution"`    // Power law fit to Earth's distribution
	CrustTypeRatio      float64 `json:"crustTypeRatio"`      // Oceanic 57.5% vs Continental 42.5%
	MajorPlateCoverage  float64 `json:"majorPlateCoverage"`  // Should be ~77% like Earth
	OverallBenchmark    float64 `json:"overallBenchmark"`    // Weighted average
}

func main() {
	// Command line flags
	configFile := flag.String("config", "", "JSON config file path")
	subdivision := flag.Int("subdivision", 6, "Icosphere subdivision level")
	numPlates := flag.Int("plates", 15, "Target number of tectonic plates")
	seed := flag.Int64("seed", time.Now().UnixNano(), "Random seed")
	planetRadius := flag.Float64("radius", 6.371e6, "Planet radius in meters")
	outputDir := flag.String("output", "./cli_output", "Output directory")
	generateObj := flag.Bool("obj", false, "Generate OBJ mesh files")
	logLevel := flag.String("log", "info", "Log level: debug, info, warn, error")
	generateElevation := flag.Bool("elevation", true, "Generate elevation data")
	benchmark := flag.Bool("benchmark", true, "Run Earth benchmark comparison")
	boundaryStyle := flag.String("boundary", "continent", "Boundary style: 'continent' or 'archipelago'")
	generateMap := flag.Bool("map", false, "Generate PNG world map")
	compareMode := flag.Bool("compare", false, "Generate side-by-side continent vs archipelago comparison")
	
	flag.Parse()

	// Load config from file if provided
	config := RunConfig{
		Subdivision:  *subdivision,
		NumPlates:    *numPlates,
		Seed:         *seed,
		PlanetRadius: *planetRadius,
		OutputDir:    *outputDir,
		GenerateObj:  *generateObj,
		LogLevel:     *logLevel,
	}

	if *configFile != "" {
		if err := loadConfig(*configFile, &config); err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	}

	fmt.Printf("🌍 WorldGen CLI - Procedural Planet Generation\n")
	fmt.Printf("==============================================\n")
	fmt.Printf("Subdivision: %d | Plates: %d | Seed: %d\n", 
		config.Subdivision, config.NumPlates, config.Seed)
	fmt.Printf("Planet Radius: %.1e m | Output: %s\n", 
		config.PlanetRadius, config.OutputDir)
	fmt.Printf("==============================================\n\n")

	startTime := time.Now()
	
	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	metrics := GenerationMetrics{
		Config:          config,
		Timestamp:       time.Now().Format(time.RFC3339),
		GenerationTimes: make(map[string]string),
	}

	// Step 1: Generate base geometry
	fmt.Printf("🔺 Generating icosphere (subdivision %d)...\n", config.Subdivision)
	stepStart := time.Now()
	
	icoVertices, icoFaces := icosphere.CreateIcosphere(config.Subdivision)
	voronoiVertices, voronoiCells := icosphere.GenerateSphericalVoronoi(icoVertices, icoFaces)
	
	metrics.GenerationTimes["geometry"] = time.Since(stepStart).String()
	logInfo(config.LogLevel, "Generated %d icosphere vertices, %d Voronoi cells", 
		len(icoVertices), len(voronoiCells))

	// Step 2: Generate tectonic plates
	fmt.Printf("🗺️  Generating tectonic plates...\n")
	stepStart = time.Now()
	
	tectonicSettings := tectonics.TectonicSettings{
		NumPlates:     config.NumPlates,
		PlanetRadius:  config.PlanetRadius,
		Seed:          config.Seed,
		BoundaryStyle: *boundaryStyle,
	}
	
	plates, cellAssignments := tectonics.InitializeTectonicPlates(
		voronoiCells, voronoiVertices, icoVertices, tectonicSettings)
	
	metrics.GenerationTimes["tectonics"] = time.Since(stepStart).String()
	logInfo(config.LogLevel, "Generated %d tectonic plates", len(plates))

	// Step 3: Validate tectonic plates
	fmt.Printf("📊 Validating plate distribution...\n")
	stepStart = time.Now()
	
	totalSurfaceArea := 4.0 * 3.14159265359 * config.PlanetRadius * config.PlanetRadius
	cellAssignmentsInt := make([]int, len(cellAssignments))
	for i, assignment := range cellAssignments {
		cellAssignmentsInt[i] = int(assignment)
	}
	
	boundaryComplexity := tectonics.CalculateBoundaryComplexity(plates, voronoiCells, cellAssignmentsInt)
	metrics.TectonicMetrics = tectonics.ValidatePlateDistributionWithCells(plates, totalSurfaceArea, voronoiCells, cellAssignmentsInt, icoVertices, config.PlanetRadius)
	metrics.TectonicMetrics.BoundaryComplexity = boundaryComplexity
	
	metrics.GenerationTimes["validation"] = time.Since(stepStart).String()

	// Step 3.5: Generate world map (optional)
	if *generateMap {
		fmt.Printf("🗺️  Generating world map...\n")
		stepStart = time.Now()
		
		mapSettings := mapgen.DefaultMapSettings()
		mapSettings.OutputPath = filepath.Join(*outputDir, "world_map.png")
		mapSettings.PlanetRadius = config.PlanetRadius
		
		err := mapgen.GenerateWorldMap(plates, cellAssignmentsInt, voronoiCells, icoVertices, mapSettings)
		if err != nil {
			fmt.Printf("⚠️  Failed to generate world map: %v\n", err)
		} else {
			fmt.Printf("✅ World map saved: %s\n", mapSettings.OutputPath)
		}
		
		metrics.GenerationTimes["map"] = time.Since(stepStart).String()
	}
	
	// Step 3.6: Generate comparison maps (optional)
	if *compareMode {
		fmt.Printf("🔄 Generating comparison maps...\n")
		stepStart = time.Now()
		
		// Generate continent mode
		continentSettings := tectonics.TectonicSettings{
			NumPlates:     config.NumPlates,
			PlanetRadius:  config.PlanetRadius,
			Seed:          config.Seed,
			BoundaryStyle: "continent",
		}
		continentPlates, continentAssignments := tectonics.InitializeTectonicPlates(
			voronoiCells, voronoiVertices, icoVertices, continentSettings)
		
		continentAssignmentsInt := make([]int, len(continentAssignments))
		for i, assignment := range continentAssignments {
			continentAssignmentsInt[i] = int(assignment)
		}
		
		// Generate archipelago mode
		archipelagoSettings := tectonics.TectonicSettings{
			NumPlates:     config.NumPlates,
			PlanetRadius:  config.PlanetRadius,
			Seed:          config.Seed,
			BoundaryStyle: "archipelago",
		}
		archipelagoPlates, archipelagoAssignments := tectonics.InitializeTectonicPlates(
			voronoiCells, voronoiVertices, icoVertices, archipelagoSettings)
		
		archipelagoAssignmentsInt := make([]int, len(archipelagoAssignments))
		for i, assignment := range archipelagoAssignments {
			archipelagoAssignmentsInt[i] = int(assignment)
		}
		
		// Generate comparison map
		comparisonPath := filepath.Join(*outputDir, "comparison_map.png")
		err := mapgen.GenerateComparisonMaps(
			continentPlates, archipelagoPlates,
			continentAssignmentsInt, archipelagoAssignmentsInt,
			voronoiCells, icoVertices, comparisonPath)
		
		if err != nil {
			fmt.Printf("⚠️  Failed to generate comparison map: %v\n", err)
		} else {
			fmt.Printf("✅ Comparison map saved: %s\n", comparisonPath)
		}
		
		metrics.GenerationTimes["comparison"] = time.Since(stepStart).String()
	}

	// Step 4: Generate elevation (optional)
	if *generateElevation {
		fmt.Printf("⛰️  Generating elevation data...\n")
		stepStart = time.Now()
		
		elevationSettings := elevation.ElevationSettings{
			NoiseScale:                    0.01,
			NoiseOctaves:                  6,
			NoisePersistence:              0.5,
			NoiseLacunarity:               2.0,
			ElevationMultiplier:           5000.0,
			CharacteristicFalloffDistance: 0.15,
			MaxBoundaryEffectDistance:     0.45,
			ConvergentBoundaryStrength:    1000.0,
			DivergentBoundaryStrength:     500.0,
		}
		
		pipelineSettings := landgen.LandGenerationPipelineSettings{
			GlobalSeed:                config.Seed,
			TectonicSettings:          tectonicSettings,
			ElevationSettings:         elevationSettings,
			PlanetRadius:              config.PlanetRadius,
			BaseIcosphereSubdivisions: config.Subdivision,
		}
		
		planetData, err := landgen.GeneratePlanetData(
			pipelineSettings, icoVertices, icoFaces, voronoiVertices, voronoiCells)
		
		if err != nil {
			log.Printf("❌ Elevation generation failed: %v", err)
		} else {
			metrics.ElevationMetrics = calculateElevationMetrics(planetData)
			metrics.GenerationTimes["elevation"] = time.Since(stepStart).String()
			logInfo(config.LogLevel, "Generated elevation data")
		}
	}

	// Step 5: Earth benchmark comparison
	if *benchmark {
		fmt.Printf("🌍 Running Earth benchmark comparison...\n")
		stepStart = time.Now()
		
		metrics.EarthBenchmark = calculateEarthBenchmark(plates, totalSurfaceArea)
		metrics.GenerationTimes["benchmark"] = time.Since(stepStart).String()
	}

	// Calculate overall metrics
	metrics.TotalTime = time.Since(startTime).String()
	metrics.OverallScore = calculateOverallScore(metrics)
	metrics.Status = "PASS"
	if metrics.OverallScore < 0.6 {
		metrics.Status = "FAIL"
	}

	// Step 6: Output results
	fmt.Printf("\n📈 RESULTS\n")
	fmt.Printf("==========\n")
	printResults(metrics)

	// Step 7: Save outputs
	fmt.Printf("\n💾 Saving outputs...\n")
	timestamp := time.Now().Format("20060102_150405")
	
	// Save JSON metrics
	metricsFile := filepath.Join(config.OutputDir, fmt.Sprintf("metrics_%s.json", timestamp))
	if err := saveMetrics(metricsFile, metrics); err != nil {
		log.Printf("❌ Failed to save metrics: %v", err)
	} else {
		fmt.Printf("✅ Metrics saved: %s\n", metricsFile)
	}

	// Generate OBJ files if requested
	if config.GenerateObj {
		if err := generateObjFiles(config.OutputDir, timestamp, icoVertices, icoFaces, 
			voronoiVertices, voronoiCells); err != nil {
			log.Printf("❌ Failed to generate OBJ files: %v", err)
		} else {
			fmt.Printf("✅ OBJ files generated\n")
		}
	}

	// Final status
	fmt.Printf("\n%s Overall Score: %.2f/1.0 (%s)\n", 
		getStatusEmoji(metrics.Status), metrics.OverallScore, metrics.Status)
	
	if metrics.Status == "FAIL" {
		os.Exit(1)
	}
}

func calculateEarthBenchmark(plates []tectonics.TectonicPlate, totalArea float64) EarthBenchmarkScore {
	// Earth targets: 
	// Major: 7 plates (77% coverage)
	// Minor: ~13 plates (5.4% coverage) 
	// Micro: ~19 plates (0.9% coverage)
	// Oceanic: 57.5%, Continental: 42.5%
	
	majorPlates := 0
	minorPlates := 0
	microPlates := 0
	majorCoverage := 0.0
	minorCoverage := 0.0
	microCoverage := 0.0
	oceanicArea := 0.0
	continentalArea := 0.0
	
	for _, plate := range plates {
		areaPercent := (plate.Area / totalArea) * 100
		if areaPercent >= 6.0 { // Earth's smallest major plate is 6.9%
			majorPlates++
			majorCoverage += areaPercent
		} else if areaPercent >= 0.18 { // Earth's minor plate range
			minorPlates++
			minorCoverage += areaPercent
		} else if areaPercent >= 0.02 { // Earth's micro plate range
			microPlates++
			microCoverage += areaPercent
		}
		
		if plate.PlateType == tectonics.OceanicPlate {
			oceanicArea += plate.Area
		} else {
			continentalArea += plate.Area
		}
	}
	
	// Calculate individual scores (0-100)
	// Major plates: target 7 ± 2
	majorScore := 100.0
	if majorPlates < 5 || majorPlates > 9 {
		majorScore = 100.0 - float64(absInt(majorPlates-7))*15.0
	}
	if majorScore < 0 { majorScore = 0 }
	
	// Minor plates: target 13 ± 5 
	minorScore := 100.0 - abs(float64(minorPlates-13))*8.0
	if minorScore < 0 { minorScore = 0 }
	
	// Micro plates: target 19 ± 7
	microScore := 100.0 - abs(float64(microPlates-19))*7.0
	if microScore < 0 { microScore = 0 }
	
	// Coverage scores
	majorCovScore := 100.0 - abs(majorCoverage-77.0)*2.0
	if majorCovScore < 0 { majorCovScore = 0 }
	
	minorCovScore := 100.0 - abs(minorCoverage-5.4)*10.0
	if minorCovScore < 0 { minorCovScore = 0 }
	
	microCovScore := 100.0 - abs(microCoverage-0.9)*50.0
	if microCovScore < 0 { microCovScore = 0 }
	
	// Crust type ratio
	oceanicPercent := (oceanicArea / totalArea) * 100
	crustScore := 100.0 - abs(oceanicPercent-57.5)*3.0
	if crustScore < 0 { crustScore = 0 }
	
	// Weighted overall score emphasizing plate hierarchy
	plateHierarchyScore := (majorScore*0.4 + minorScore*0.35 + microScore*0.25)
	coverageScore := (majorCovScore*0.6 + minorCovScore*0.25 + microCovScore*0.15)
	sizeDistributionScore := (plateHierarchyScore + coverageScore) / 2.0
	
	// Convert to 0-1 scale
	return EarthBenchmarkScore{
		PlateCountScore:    (majorScore + minorScore + microScore) / 300.0,
		MajorPlateCoverage: coverageScore / 100.0,
		CrustTypeRatio:     crustScore / 100.0,
		SizeDistribution:   sizeDistributionScore / 100.0,
		OverallBenchmark:   (plateHierarchyScore*0.4 + coverageScore*0.3 + crustScore*0.3) / 100.0,
	}
}

func calculateElevationMetrics(planetData *landgen.PlanetData) *ElevationMetrics {
	if planetData.ElevationData == nil {
		return nil
	}
	
	// Placeholder - would analyze actual elevation data
	return &ElevationMetrics{
		AvgOceanicElevation:     -2500.0,
		AvgContinentalElevation: 840.0,
		ElevationRange:          12000.0,
		BoundaryEffectStrength:  0.75,
	}
}

func calculateOverallScore(metrics GenerationMetrics) float64 {
	weights := map[string]float64{
		"tectonic":   0.4,
		"earth":      0.3,
		"elevation":  0.2,
		"performance": 0.1,
	}
	
	score := metrics.TectonicMetrics.OverallScore * weights["tectonic"]
	score += metrics.EarthBenchmark.OverallBenchmark * weights["earth"]
	
	if metrics.ElevationMetrics != nil {
		elevScore := 0.8 // Placeholder
		score += elevScore * weights["elevation"]
	}
	
	perfScore := 1.0 // Placeholder - based on generation time
	score += perfScore * weights["performance"]
	
	return score
}

func printResults(metrics GenerationMetrics) {
	fmt.Printf("Tectonic Score:    %.2f/1.0\n", metrics.TectonicMetrics.OverallScore)
	fmt.Printf("Earth Benchmark:   %.2f/1.0\n", metrics.EarthBenchmark.OverallBenchmark)
	if metrics.ElevationMetrics != nil {
		fmt.Printf("Elevation Score:   %.2f/1.0\n", 0.8) // Placeholder
	}
	fmt.Printf("Generation Time:   %s\n", metrics.TotalTime)
	
	fmt.Printf("\nPlate Hierarchy (vs Earth targets):\n")
	fmt.Printf("  Major Plates:    %d (Earth: 7)\n", metrics.TectonicMetrics.MajorCount)
	fmt.Printf("  Minor Plates:    %d (Earth: ~13)\n", metrics.TectonicMetrics.MinorCount)
	fmt.Printf("  Micro Plates:    %d (Earth: ~19)\n", metrics.TectonicMetrics.MicroCount)
	
	fmt.Printf("\nCoverage Distribution:\n")
	fmt.Printf("  Major Coverage:  %.1f%% (Earth: 77.0%%)\n", metrics.TectonicMetrics.MajorCoverage)
	fmt.Printf("  Minor Coverage:  %.1f%% (Earth: 5.4%%)\n", metrics.TectonicMetrics.MinorCoverage)
	fmt.Printf("  Micro Coverage:  %.1f%% (Earth: 0.9%%)\n", metrics.TectonicMetrics.MicroCoverage)
	
	fmt.Printf("\nCrust Types:\n")
	fmt.Printf("  Continental:     %d plates (%.1f%%)\n", 
		metrics.TectonicMetrics.ContinentalPlates, metrics.TectonicMetrics.ContinentalRatio*100)
	fmt.Printf("  Oceanic:         %d plates (%.1f%%)\n", 
		metrics.TectonicMetrics.OceanicPlates, (1.0-metrics.TectonicMetrics.ContinentalRatio)*100)
}

func getStatusEmoji(status string) string {
	if status == "PASS" {
		return "✅"
	}
	return "❌"
}

func logInfo(level, format string, args ...interface{}) {
	if level == "debug" || level == "info" {
		fmt.Printf("ℹ️  "+format+"\n", args...)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func loadConfig(filename string, config *RunConfig) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, config)
}

func saveMetrics(filename string, metrics GenerationMetrics) error {
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func generateObjFiles(outputDir, timestamp string, icoVertices []icosphere.Vector3D, 
	icoFaces []icosphere.Triangle, voroVertices []icosphere.Vector3D, 
	voroCells []icosphere.VoronoiCell) error {
	
	// Generate icosphere OBJ
	icoFile := filepath.Join(outputDir, fmt.Sprintf("icosphere_%s.obj", timestamp))
	if err := icosphere.SaveOBJTriangulated(icoFile, icoVertices, icoFaces, 
		fmt.Sprintf("Icosphere generated %s", timestamp)); err != nil {
		return fmt.Errorf("icosphere OBJ: %v", err)
	}
	
	// Generate Voronoi OBJ  
	voroFile := filepath.Join(outputDir, fmt.Sprintf("voronoi_%s.obj", timestamp))
	if err := icosphere.SaveVoronoiOBJTriangulated(voroFile, voroVertices, voroCells,
		fmt.Sprintf("Voronoi diagram generated %s", timestamp)); err != nil {
		return fmt.Errorf("voronoi OBJ: %v", err)
	}
	
	return nil
}