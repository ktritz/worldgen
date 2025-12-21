package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
)

func main() {
	// Command line flags
	subdivision := flag.Int("subdivision", 6, "Icosphere subdivision level (minimum 6 recommended)")
	numPlates := flag.Int("plates", 15, "Target number of tectonic plates")
	seed := flag.Int64("seed", time.Now().UnixNano(), "Random seed for generation")
	planetRadius := flag.Float64("radius", 6.371e6, "Planet radius in meters (Earth = 6.371e6)")
	outputDir := flag.String("output", "./validation_output", "Directory for output files")
	
	flag.Parse()
	
	// Validate inputs
	if *subdivision < 4 {
		log.Fatal("Subdivision level must be at least 4")
	}
	if *numPlates < 1 {
		log.Fatal("Number of plates must be at least 1")
	}
	
	fmt.Printf("=== Tectonic Plate Generation & Validation ===\n")
	fmt.Printf("Subdivision level: %d\n", *subdivision)
	fmt.Printf("Target plates: %d\n", *numPlates)
	fmt.Printf("Planet radius: %.3e m\n", *planetRadius)
	fmt.Printf("Random seed: %d\n", *seed)
	fmt.Printf("Output directory: %s\n", *outputDir)
	fmt.Printf("===============================================\n\n")
	
	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	
	// Step 1: Generate icosphere
	fmt.Printf("Step 1: Generating icosphere (subdivision %d)...\n", *subdivision)
	startTime := time.Now()
	
	icoVertices, icoFaces := icosphere.CreateIcosphere(*subdivision)
	fmt.Printf("  Generated icosphere: %d vertices, %d faces in %v\n", 
		len(icoVertices), len(icoFaces), time.Since(startTime))
	
	// Step 2: Generate Voronoi diagram
	fmt.Printf("\nStep 2: Computing Voronoi diagram...\n")
	startTime = time.Now()
	
	voronoiVertices, voronoiCells := icosphere.GenerateSphericalVoronoi(icoVertices, icoFaces)
	fmt.Printf("  Generated Voronoi: %d cells, %d vertices in %v\n", 
		len(voronoiCells), len(voronoiVertices), time.Since(startTime))
	
	// Verify neighbor data
	neighborCount := 0
	for _, cell := range voronoiCells {
		neighborCount += len(cell.NeighborSiteIndices)
	}
	fmt.Printf("  Neighbor connections: %d total (%d avg per cell)\n", 
		neighborCount, neighborCount/len(voronoiCells))
	
	// Step 3: Initialize tectonic settings
	fmt.Printf("\nStep 3: Configuring tectonic settings...\n")
	settings := tectonics.TectonicSettings{
		NumPlates:    *numPlates,
		PlanetRadius: *planetRadius,
		Seed:         *seed,
	}
	
	// Step 4: Generate tectonic plates
	fmt.Printf("\nStep 4: Generating tectonic plates...\n")
	startTime = time.Now()
	
	plates, cellAssignments := tectonics.InitializeTectonicPlates(
		voronoiCells,
		voronoiVertices,
		icoVertices,
		settings,
	)
	
	generationTime := time.Since(startTime)
	fmt.Printf("  Generated %d plates in %v\n", len(plates), generationTime)
	
	// Step 5: Calculate validation metrics
	fmt.Printf("\nStep 5: Running validation analysis...\n")
	startTime = time.Now()
	
	totalSurfaceArea := 4.0 * 3.14159265359 * (*planetRadius) * (*planetRadius)
	
	// Convert cellAssignments from []int32 to []int for validation functions
	cellAssignmentsInt := make([]int, len(cellAssignments))
	for i, assignment := range cellAssignments {
		cellAssignmentsInt[i] = int(assignment)
	}
	
	// Calculate boundary complexity
	boundaryComplexity := tectonics.CalculateBoundaryComplexity(plates, voronoiCells, cellAssignmentsInt)
	
	// Get comprehensive validation metrics
	validationMetrics := tectonics.ValidatePlateDistribution(plates, totalSurfaceArea)
	validationMetrics.BoundaryComplexity = boundaryComplexity
	
	validationTime := time.Since(startTime)
	fmt.Printf("  Validation analysis completed in %v\n", validationTime)
	
	// Step 6: Display results
	fmt.Printf("\nStep 6: Validation Results\n")
	tectonics.PrintValidationReport(validationMetrics)
	
	// Step 7: Save detailed results to file
	fmt.Printf("\nStep 7: Saving detailed results...\n")
	
	// Create timestamped filename
	timestamp := time.Now().Format("20060102_150405")
	resultsFile := filepath.Join(*outputDir, fmt.Sprintf("validation_results_%s_sub%d.txt", timestamp, *subdivision))
	
	if err := saveDetailedResults(resultsFile, *subdivision, *numPlates, *seed, *planetRadius, 
		plates, validationMetrics, generationTime, validationTime); err != nil {
		log.Printf("Warning: Failed to save results file: %v", err)
	} else {
		fmt.Printf("  Results saved to: %s\n", resultsFile)
	}
	
	// Summary
	fmt.Printf("\n=== SUMMARY ===\n")
	fmt.Printf("Generated %d plates (target: %d)\n", len(plates), *numPlates)
	fmt.Printf("Overall validation score: %.2f/1.0\n", validationMetrics.OverallScore)
	fmt.Printf("Boundary realism score: %.2f/1.0\n", validationMetrics.BoundaryComplexity)
	fmt.Printf("Total runtime: %v\n", generationTime + validationTime)
	
	if validationMetrics.OverallScore >= 0.6 {
		fmt.Printf("✓ PASSED: Algorithm produces realistic tectonic plates\n")
	} else {
		fmt.Printf("✗ FAILED: Algorithm needs improvement (score < 0.6)\n")
	}
}

func saveDetailedResults(filename string, subdivision, numPlates int, seed int64, radius float64,
	plates []tectonics.TectonicPlate, metrics tectonics.ValidationMetrics, 
	genTime, valTime time.Duration) error {
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	fmt.Fprintf(file, "Tectonic Plate Generation & Validation Results\n")
	fmt.Fprintf(file, "==============================================\n\n")
	
	fmt.Fprintf(file, "GENERATION PARAMETERS:\n")
	fmt.Fprintf(file, "Subdivision level: %d\n", subdivision)
	fmt.Fprintf(file, "Target plates: %d\n", numPlates)
	fmt.Fprintf(file, "Actual plates generated: %d\n", len(plates))
	fmt.Fprintf(file, "Random seed: %d\n", seed)
	fmt.Fprintf(file, "Planet radius: %.3e m\n", radius)
	fmt.Fprintf(file, "Generation time: %v\n", genTime)
	fmt.Fprintf(file, "Validation time: %v\n", valTime)
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "VALIDATION SCORES:\n")
	fmt.Fprintf(file, "Overall Score: %.3f/1.0\n", metrics.OverallScore)
	fmt.Fprintf(file, "Size Distribution: %.3f/1.0\n", metrics.SizeDistributionScore)
	fmt.Fprintf(file, "Coverage Match: %.3f/1.0\n", metrics.CoverageScore)
	fmt.Fprintf(file, "Continental Mix: %.3f/1.0\n", metrics.ContinentalScore)
	fmt.Fprintf(file, "Boundary Complexity: %.3f/1.0\n", metrics.BoundaryComplexity)
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "PLATE DISTRIBUTION:\n")
	fmt.Fprintf(file, "Major plates (≥4%%): %d\n", metrics.MajorCount)
	fmt.Fprintf(file, "Minor plates (0.5-4%%): %d\n", metrics.MinorCount)
	fmt.Fprintf(file, "Micro plates (0.1-0.5%%): %d\n", metrics.MicroCount)
	fmt.Fprintf(file, "Continental plates: %d\n", metrics.ContinentalPlates)
	fmt.Fprintf(file, "Oceanic plates: %d\n", metrics.OceanicPlates)
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "SURFACE COVERAGE:\n")
	fmt.Fprintf(file, "Major plate coverage: %.1f%%\n", metrics.MajorCoverage)
	fmt.Fprintf(file, "Minor plate coverage: %.1f%%\n", metrics.MinorCoverage)
	fmt.Fprintf(file, "Micro plate coverage: %.1f%%\n", metrics.MicroCoverage)
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "INDIVIDUAL PLATE DETAILS:\n")
	for i, plate := range plates {
		areaPercent := (plate.Area / (4.0 * 3.14159265359 * radius * radius)) * 100
		fmt.Fprintf(file, "Plate %d: Area=%.2f%%, Type=%s\n", i+1, areaPercent, plate.PlateType)
	}
	
	return nil
}