package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Simplified test to generate land data for visual analysis
func main() {
	fmt.Println("🧪 Generating Test Data for Visual Analysis")
	fmt.Println("==========================================")

	// Step 1: Generate Icosphere
	fmt.Println("1. Generating icosphere and Voronoi data...")
	
	icospherePayload := map[string]interface{}{
		"subdivisions":    5,     // High subdivision for fine resolution (2562 vertices)
		"seed":           42,
		"voronoiEnable":  true,   // Enable Voronoi generation
		"out":            "test_icosphere.obj",
		"voronoiOut":     "test_voronoi.obj",
	}

	icosphereData, err := makeRequest("http://localhost:8080/api/generate", icospherePayload)
	if err != nil {
		fmt.Printf("❌ Failed to generate icosphere: %v\n", err)
		return
	}

	fmt.Printf("✅ Icosphere generated successfully\n")

	// Step 2: Generate Land with modularized parameters
	fmt.Println("2. Generating land with tectonic plates and elevation...")

	landPayload := map[string]interface{}{
		// General parameters
		"landSeed": 42,

		// Tectonic parameters (updated field names)
		"numPlates":                   12, // More plates for interesting boundaries
		"baseSpeed":                   0.02,
		"speedVariationFactor":        0.8, // High variation for diverse plate speeds
		"targetContinentalProportion": 0.4,  // 40% continental
		"numInitialContinentalSeeds":  4,

		// Elevation parameters  
		"noiseScale":          1.0,
		"noiseOctaves":        4,
		"noisePersistence":    0.5,
		"noiseLacunarity":     2.0,
		"elevationMultiplier": 150, // Higher for more dramatic elevation

		// Tectonic boundary effects (Earth-scale realistic parameters)
		"characteristicFalloffDistance": 0.03,  // ~200km (200000m / 6371000m = 0.031)
		"maxBoundaryEffectDistance":     0.08,  // ~500km (500000m / 6371000m = 0.078)
		"convergentBoundaryStrength":    2000,  // Strong mountain formation
		"divergentBoundaryStrength":     800,   // Significant rift formation

		// Mesh data from icosphere generation
		"baseIcosphereData":     icosphereData["icosphereData"],
		"baseVoronoiData":       icosphereData["voronoiData"],
		"icosphereSubdivisions": icosphereData["subdivisionLevel"],
		"landOutputName":        "test_analysis_output.png",
	}

	landData, err := makeRequest("http://localhost:8080/api/generate_land", landPayload)
	if err != nil {
		fmt.Printf("❌ Failed to generate land: %v\n", err)
		return
	}

	fmt.Printf("✅ Land generated successfully\n")

	// Step 3: Save elevation data for analysis
	if err := saveElevationData(landData); err != nil {
		fmt.Printf("⚠️  Warning: Could not save elevation data: %v\n", err)
	}

	// Step 4: Generate analysis summary
	if err := generateAnalysisSummary(icosphereData, landData); err != nil {
		fmt.Printf("⚠️  Warning: Could not generate analysis: %v\n", err)
	}

	fmt.Println("\n🎯 Test data generation complete!")
	fmt.Println("📊 Check the following files for visual analysis:")
	fmt.Println("   - elevation_data.json (raw elevation values)")
	fmt.Println("   - analysis_summary.txt (statistical summary)")
	fmt.Println("   - Any auxiliary output files from the server")
}

func makeRequest(url string, payload map[string]interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %v", err)
	}

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %v", err)
	}

	return result, nil
}

func saveElevationData(landData map[string]interface{}) error {
	elevationData, ok := landData["elevationData"]
	if !ok {
		return fmt.Errorf("no elevation data in response")
	}

	jsonData, err := json.MarshalIndent(elevationData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal elevation data: %v", err)
	}

	if err := os.WriteFile("elevation_data.json", jsonData, 0644); err != nil {
		return fmt.Errorf("write elevation data: %v", err)
	}

	fmt.Println("💾 Saved: elevation_data.json")
	return nil
}

func generateAnalysisSummary(icosphereData, landData map[string]interface{}) error {
	var summary bytes.Buffer
	
	summary.WriteString("LANDGEN TEST DATA ANALYSIS SUMMARY\n")
	summary.WriteString("===================================\n\n")

	// Icosphere analysis
	if ico, ok := icosphereData["icosphereData"].(map[string]interface{}); ok {
		if vertices, ok := ico["vertices"].([]interface{}); ok {
			summary.WriteString(fmt.Sprintf("Icosphere Vertices: %d\n", len(vertices)/3))
		}
		if faces, ok := ico["faces"].([]interface{}); ok {
			summary.WriteString(fmt.Sprintf("Icosphere Faces: %d\n", len(faces)/3))
		}
	}

	if vor, ok := icosphereData["voronoiData"].(map[string]interface{}); ok {
		if cells, ok := vor["cells"].([]interface{}); ok {
			summary.WriteString(fmt.Sprintf("Voronoi Cells (Potential Plates): %d\n", len(cells)))
		}
	}

	summary.WriteString("\n")

	// Elevation analysis
	if elevData, ok := landData["elevationData"].(map[string]interface{}); ok {
		if cellElevations, ok := elevData["cellElevations"].(map[string]interface{}); ok {
			var elevations []float64
			for _, v := range cellElevations {
				if elev, ok := v.(float64); ok {
					elevations = append(elevations, elev)
				}
			}

			if len(elevations) > 0 {
				min, max := elevations[0], elevations[0]
				sum := 0.0
				for _, elev := range elevations {
					if elev < min { min = elev }
					if elev > max { max = elev }
					sum += elev
				}
				avg := sum / float64(len(elevations))

				summary.WriteString(fmt.Sprintf("Elevation Analysis:\n"))
				summary.WriteString(fmt.Sprintf("  Sites with elevation: %d\n", len(elevations)))
				summary.WriteString(fmt.Sprintf("  Elevation range: %.1f to %.1f meters\n", min, max))
				summary.WriteString(fmt.Sprintf("  Average elevation: %.1f meters\n", avg))
				summary.WriteString(fmt.Sprintf("  Total relief: %.1f meters\n", max - min))

				// Categorize elevations
				oceanic := 0
				lowland := 0
				highland := 0
				mountain := 0

				for _, elev := range elevations {
					if elev < 0 { 
						oceanic++ 
					} else if elev < 200 { 
						lowland++ 
					} else if elev < 1000 { 
						highland++ 
					} else { 
						mountain++ 
					}
				}

				summary.WriteString(fmt.Sprintf("\nElevation Distribution:\n"))
				summary.WriteString(fmt.Sprintf("  Oceanic (< 0m): %d (%.1f%%)\n", oceanic, float64(oceanic)/float64(len(elevations))*100))
				summary.WriteString(fmt.Sprintf("  Lowland (0-200m): %d (%.1f%%)\n", lowland, float64(lowland)/float64(len(elevations))*100))
				summary.WriteString(fmt.Sprintf("  Highland (200-1000m): %d (%.1f%%)\n", highland, float64(highland)/float64(len(elevations))*100))
				summary.WriteString(fmt.Sprintf("  Mountain (>1000m): %d (%.1f%%)\n", mountain, float64(mountain)/float64(len(elevations))*100))
			}
		}
	}

	summary.WriteString("\nExpected Behavior:\n")
	summary.WriteString("- Should see mix of oceanic (negative) and continental (positive) elevations\n")
	summary.WriteString("- Continental plates should show higher base elevations (200-600m range)\n")
	summary.WriteString("- Oceanic plates should show lower base elevations (-4500 to -3500m range)\n")
	summary.WriteString("- Convergent boundaries should create mountains (elevated areas)\n")
	summary.WriteString("- Divergent boundaries should create rifts (depressed areas)\n")
	summary.WriteString("- Total relief should be significant (>3000m) indicating tectonic effects\n")

	if err := os.WriteFile("analysis_summary.txt", summary.Bytes(), 0644); err != nil {
		return fmt.Errorf("write analysis summary: %v", err)
	}

	fmt.Println("📊 Saved: analysis_summary.txt")
	return nil
}