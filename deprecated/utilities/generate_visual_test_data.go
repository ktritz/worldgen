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

// Use the same structures as the server expects
type MeshData struct {
	Vertices []float32 `json:"vertices"`
	Faces    []int32   `json:"faces"`
}

type VoronoiCellData struct {
	SiteIndex           int32   `json:"siteIndex"`
	NeighborSiteIndices []int32 `json:"neighborSiteIndices"`
	VertexIndices       []int32 `json:"vertexIndices"`
}

type VoronoiMeshData struct {
	Vertices []float32         `json:"vertices"`
	Cells    []VoronoiCellData `json:"cells"`
}

type IcosphereResponse struct {
	Status            string                 `json:"status"`
	Message           string                 `json:"message"`
	IcosphereData     map[string]interface{} `json:"icosphereData"`
	VoronoiData       map[string]interface{} `json:"voronoiData"`
	SubdivisionLevel  int                    `json:"subdivisionLevel"`
}

type LandGenRequest struct {
	LandSeed                      int64            `json:"landSeed"`
	NumPlates                     int              `json:"numPlates"`
	BaseSpeed                     float64          `json:"baseSpeed"`
	SpeedVariationFactor          float64          `json:"speedVariationFactor"`
	TargetContinentalProportion   float64          `json:"targetContinentalProportion"`
	NumInitialContinentalSeeds    int              `json:"numInitialContinentalSeeds"`
	NoiseScale                    float64          `json:"noiseScale"`
	NoiseOctaves                  int              `json:"noiseOctaves"`
	NoisePersistence              float64          `json:"noisePersistence"`
	NoiseLacunarity               float64          `json:"noiseLacunarity"`
	ElevationMultiplier           float64          `json:"elevationMultiplier"`
	CharacteristicFalloffDistance float64          `json:"characteristicFalloffDistance"`
	MaxBoundaryEffectDistance     float64          `json:"maxBoundaryEffectDistance"`
	ConvergentBoundaryStrength    float64          `json:"convergentBoundaryStrength"`
	DivergentBoundaryStrength     float64          `json:"divergentBoundaryStrength"`
	BaseIcosphereData             *MeshData        `json:"baseIcosphereData"`
	BaseVoronoiData               *VoronoiMeshData `json:"baseVoronoiData"`
	IcosphereSubdivisions         int              `json:"icosphereSubdivisions"`
	LandOutputName                string           `json:"landOutputName"`
}

type LandGenResponse struct {
	Status        string                 `json:"status"`
	Message       string                 `json:"message"`
	ElevationData map[string]interface{} `json:"elevationData"`
}

func main() {
	fmt.Println("🧪 Generating Test Data for Visual Analysis (Go)")
	fmt.Println("===============================================")

	// Step 1: Generate Icosphere
	fmt.Println("1. Generating icosphere and Voronoi data...")
	
	icospherePayload := map[string]interface{}{
		"subdivision": 3, // Higher subdivision for more detail
		"seed":        42,
	}

	icosphereResp, err := generateIcosphere(icospherePayload)
	if err != nil {
		fmt.Printf("❌ Failed to generate icosphere: %v\n", err)
		return
	}

	fmt.Printf("✅ Icosphere generated successfully\n")

	// Save icosphere data for inspection
	if err := saveJSONFile("icosphere_data.json", icosphereResp); err != nil {
		fmt.Printf("⚠️  Warning: Could not save icosphere data: %v\n", err)
	}

	// Step 2: Generate Land with modularized parameters
	fmt.Println("2. Generating land with tectonic plates and elevation...")

	landResp, err := generateLandData(icosphereResp)
	if err != nil {
		fmt.Printf("❌ Failed to generate land: %v\n", err)
		return
	}

	fmt.Printf("✅ Land generated successfully\n")

	// Save land data for inspection
	if err := saveJSONFile("land_data.json", landResp); err != nil {
		fmt.Printf("⚠️  Warning: Could not save land data: %v\n", err)
	}

	// Step 3: Generate analysis summary
	if err := generateAnalysisSummary(icosphereResp, landResp); err != nil {
		fmt.Printf("⚠️  Warning: Could not generate analysis: %v\n", err)
	}

	fmt.Println("\n🎯 Test data generation complete!")
	fmt.Println("📊 Check the following files for visual analysis:")
	fmt.Println("   - icosphere_data.json (icosphere and Voronoi data)")
	fmt.Println("   - land_data.json (land generation results)")
	fmt.Println("   - analysis_summary.txt (statistical summary)")
}

func generateIcosphere(payload map[string]interface{}) (*IcosphereResponse, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %v", err)
	}

	resp, err := http.Post("http://localhost:8080/api/generate", "application/json", bytes.NewBuffer(jsonData))
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

	var result IcosphereResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %v", err)
	}

	return &result, nil
}

func generateLandData(icosphereResp *IcosphereResponse) (*LandGenResponse, error) {
	// Convert raw data to structured format
	meshData, err := convertToMeshData(icosphereResp.IcosphereData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert icosphere data: %v", err)
	}

	voronoiData, err := convertToVoronoiMeshData(icosphereResp.VoronoiData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert voronoi data: %v", err)
	}

	payload := LandGenRequest{
		// General parameters
		LandSeed: 42,

		// Tectonic parameters (updated field names)
		NumPlates:                   12, // More plates for interesting boundaries
		BaseSpeed:                   0.02,
		SpeedVariationFactor:        0.8, // High variation for diverse plate speeds
		TargetContinentalProportion: 0.4, // 40% continental
		NumInitialContinentalSeeds:  4,

		// Elevation parameters  
		NoiseScale:          1.0,
		NoiseOctaves:        4,
		NoisePersistence:    0.5,
		NoiseLacunarity:     2.0,
		ElevationMultiplier: 150, // Higher for more dramatic elevation

		// Tectonic boundary effects (new parameters)
		CharacteristicFalloffDistance: 0.1,  // Closer falloff for sharper boundaries
		MaxBoundaryEffectDistance:     0.3,  // Wider effect area
		ConvergentBoundaryStrength:    2000, // Strong mountain formation
		DivergentBoundaryStrength:     800,  // Significant rift formation

		// Mesh data from icosphere generation
		BaseIcosphereData:     meshData,
		BaseVoronoiData:       voronoiData,
		IcosphereSubdivisions: icosphereResp.SubdivisionLevel,
		LandOutputName:        "test_analysis_output.png",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %v", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post("http://localhost:8080/api/generate_land", "application/json", bytes.NewBuffer(jsonData))
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

	var result LandGenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %v", err)
	}

	return &result, nil
}

func convertToMeshData(rawData map[string]interface{}) (*MeshData, error) {
	verticesRaw, ok := rawData["vertices"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid vertices data")
	}
	
	facesRaw, ok := rawData["faces"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid faces data")
	}

	vertices := make([]float32, len(verticesRaw))
	for i, v := range verticesRaw {
		if f, ok := v.(float64); ok {
			vertices[i] = float32(f)
		} else {
			return nil, fmt.Errorf("invalid vertex value at index %d", i)
		}
	}

	faces := make([]int32, len(facesRaw))
	for i, v := range facesRaw {
		if f, ok := v.(float64); ok {
			faces[i] = int32(f)
		} else {
			return nil, fmt.Errorf("invalid face value at index %d", i)
		}
	}

	return &MeshData{Vertices: vertices, Faces: faces}, nil
}

func convertToVoronoiMeshData(rawData map[string]interface{}) (*VoronoiMeshData, error) {
	verticesRaw, ok := rawData["vertices"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid voronoi vertices data")
	}
	
	cellsRaw, ok := rawData["cells"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid voronoi cells data")
	}

	vertices := make([]float32, len(verticesRaw))
	for i, v := range verticesRaw {
		if f, ok := v.(float64); ok {
			vertices[i] = float32(f)
		} else {
			return nil, fmt.Errorf("invalid voronoi vertex value at index %d", i)
		}
	}

	cells := make([]VoronoiCellData, len(cellsRaw))
	for i, cellRaw := range cellsRaw {
		cellMap, ok := cellRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid cell data at index %d", i)
		}

		siteIndex, ok := cellMap["siteIndex"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid siteIndex at cell %d", i)
		}

		neighborsRaw, ok := cellMap["neighborSiteIndices"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid neighborSiteIndices at cell %d", i)
		}
		neighbors := make([]int32, len(neighborsRaw))
		for j, n := range neighborsRaw {
			if nf, ok := n.(float64); ok {
				neighbors[j] = int32(nf)
			} else {
				return nil, fmt.Errorf("invalid neighbor index at cell %d, neighbor %d", i, j)
			}
		}

		vertexIndicesRaw, ok := cellMap["vertexIndices"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid vertexIndices at cell %d", i)
		}
		vertexIndices := make([]int32, len(vertexIndicesRaw))
		for j, v := range vertexIndicesRaw {
			if vf, ok := v.(float64); ok {
				vertexIndices[j] = int32(vf)
			} else {
				return nil, fmt.Errorf("invalid vertex index at cell %d, vertex %d", i, j)
			}
		}

		cells[i] = VoronoiCellData{
			SiteIndex:           int32(siteIndex),
			NeighborSiteIndices: neighbors,
			VertexIndices:       vertexIndices,
		}
	}

	return &VoronoiMeshData{Vertices: vertices, Cells: cells}, nil
}

func saveJSONFile(filename string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal error: %v", err)
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("write error: %v", err)
	}

	fmt.Printf("💾 Saved: %s\n", filename)
	return nil
}

func generateAnalysisSummary(icosphereResp *IcosphereResponse, landResp *LandGenResponse) error {
	var summary bytes.Buffer
	
	summary.WriteString("LANDGEN TEST DATA ANALYSIS SUMMARY\n")
	summary.WriteString("===================================\n\n")

	// Icosphere analysis
	if vertices, ok := icosphereResp.IcosphereData["vertices"].([]interface{}); ok {
		summary.WriteString(fmt.Sprintf("Icosphere Vertices: %d\n", len(vertices)/3))
	}
	if faces, ok := icosphereResp.IcosphereData["faces"].([]interface{}); ok {
		summary.WriteString(fmt.Sprintf("Icosphere Faces: %d\n", len(faces)/3))
	}
	if cells, ok := icosphereResp.VoronoiData["cells"].([]interface{}); ok {
		summary.WriteString(fmt.Sprintf("Voronoi Cells (Potential Plates): %d\n", len(cells)))
	}

	summary.WriteString("\n")

	// Elevation analysis
	if elevData, ok := landResp.ElevationData["cellElevations"].(map[string]interface{}); ok {
		var elevations []float64
		for _, v := range elevData {
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
			oceanic, lowland, highland, mountain := 0, 0, 0, 0
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

			total := float64(len(elevations))
			summary.WriteString(fmt.Sprintf("\nElevation Distribution:\n"))
			summary.WriteString(fmt.Sprintf("  Oceanic (< 0m): %d (%.1f%%)\n", oceanic, float64(oceanic)/total*100))
			summary.WriteString(fmt.Sprintf("  Lowland (0-200m): %d (%.1f%%)\n", lowland, float64(lowland)/total*100))
			summary.WriteString(fmt.Sprintf("  Highland (200-1000m): %d (%.1f%%)\n", highland, float64(highland)/total*100))
			summary.WriteString(fmt.Sprintf("  Mountain (>1000m): %d (%.1f%%)\n", mountain, float64(mountain)/total*100))
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

	fmt.Printf("📊 Saved: analysis_summary.txt\n")
	return nil
}