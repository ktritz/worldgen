package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// IcosphereRequest represents the payload for icosphere generation
type IcosphereRequest struct {
	Subdivision int   `json:"subdivision"`
	Seed        int64 `json:"seed"`
}

// IcosphereResponse represents the response from icosphere generation
type IcosphereResponse struct {
	Status            string                 `json:"status"`
	Message           string                 `json:"message"`
	IcosphereData     map[string]interface{} `json:"icosphereData"`
	VoronoiData       map[string]interface{} `json:"voronoiData"`
	SubdivisionLevel  int                    `json:"subdivisionLevel"`
}

// MeshData represents the structure expected by the server
type MeshData struct {
	Vertices []float32 `json:"vertices"`
	Faces    []int32   `json:"faces"`
}

// VoronoiCellData represents a single Voronoi cell
type VoronoiCellData struct {
	SiteIndex           int32   `json:"siteIndex"`
	NeighborSiteIndices []int32 `json:"neighborSiteIndices"`
	VertexIndices       []int32 `json:"vertexIndices"`
}

// VoronoiMeshData represents the Voronoi mesh structure
type VoronoiMeshData struct {
	Vertices []float32         `json:"vertices"`
	Cells    []VoronoiCellData `json:"cells"`
}

// LandGenRequest represents the payload for land generation with updated parameters
type LandGenRequest struct {
	// General land parameters
	LandSeed int64 `json:"landSeed"`

	// Updated tectonic parameters
	NumPlates                   int     `json:"numPlates"`
	BaseSpeed                   float64 `json:"baseSpeed"`
	SpeedVariationFactor        float64 `json:"speedVariationFactor"`
	TargetContinentalProportion float64 `json:"targetContinentalProportion"`
	NumInitialContinentalSeeds  int     `json:"numInitialContinentalSeeds"`

	// Elevation parameters
	NoiseScale          float64 `json:"noiseScale"`
	NoiseOctaves        int     `json:"noiseOctaves"`
	NoisePersistence    float64 `json:"noisePersistence"`
	NoiseLacunarity     float64 `json:"noiseLacunarity"`
	ElevationMultiplier float64 `json:"elevationMultiplier"`

	// New tectonic boundary effects
	CharacteristicFalloffDistance float64 `json:"characteristicFalloffDistance"`
	MaxBoundaryEffectDistance     float64 `json:"maxBoundaryEffectDistance"`
	ConvergentBoundaryStrength    float64 `json:"convergentBoundaryStrength"`
	DivergentBoundaryStrength     float64 `json:"divergentBoundaryStrength"`

	// Base mesh data
	BaseIcosphereData     *MeshData        `json:"baseIcosphereData"`
	BaseVoronoiData       *VoronoiMeshData `json:"baseVoronoiData"`
	IcosphereSubdivisions int              `json:"icosphereSubdivisions"`
	LandOutputName        string           `json:"landOutputName"`
}

// LandGenResponse represents the response from land generation
type LandGenResponse struct {
	Status        string                 `json:"status"`
	Message       string                 `json:"message"`
	ElevationData map[string]interface{} `json:"elevationData"`
}

func testIcosphereGeneration() (*IcosphereResponse, error) {
	fmt.Println("Testing icosphere generation...")
	
	payload := IcosphereRequest{
		Subdivision: 2,
		Seed:        12345,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal icosphere request: %v", err)
	}

	resp, err := http.Post("http://localhost:8080/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to make icosphere request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read icosphere response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("icosphere generation failed with status %d: %s", resp.StatusCode, string(body))
	}

	var icosphereResp IcosphereResponse
	if err := json.Unmarshal(body, &icosphereResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal icosphere response: %v", err)
	}

	// Calculate vertex and face counts
	vertices, ok := icosphereResp.IcosphereData["vertices"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid icosphere vertices data")
	}
	faces, ok := icosphereResp.IcosphereData["faces"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid icosphere faces data")
	}

	fmt.Printf("✅ Icosphere generation successful\n")
	fmt.Printf("   - Vertices: %d\n", len(vertices)/3)
	fmt.Printf("   - Faces: %d\n", len(faces)/3)

	// Debug: Print the response structure
	fmt.Printf("   - Debug: IcosphereData keys: ")
	for key := range icosphereResp.IcosphereData {
		fmt.Printf("%s ", key)
	}
	fmt.Println()
	
	fmt.Printf("   - Debug: VoronoiData keys: ")
	for key := range icosphereResp.VoronoiData {
		fmt.Printf("%s ", key)
	}
	fmt.Println()
	
	// Print the entire response to debug
	responseBytes, _ := json.MarshalIndent(icosphereResp, "", "  ")
	fmt.Printf("   - Debug: Full response structure:\n%s\n", string(responseBytes))

	return &icosphereResp, nil
}

// convertToMeshData converts raw interface{} data to MeshData structure
func convertToMeshData(rawData map[string]interface{}) (*MeshData, error) {
	verticesRaw, ok := rawData["vertices"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid vertices data")
	}
	
	facesRaw, ok := rawData["faces"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid faces data")
	}

	// Convert vertices
	vertices := make([]float32, len(verticesRaw))
	for i, v := range verticesRaw {
		if f, ok := v.(float64); ok {
			vertices[i] = float32(f)
		} else {
			return nil, fmt.Errorf("invalid vertex value at index %d", i)
		}
	}

	// Convert faces
	faces := make([]int32, len(facesRaw))
	for i, v := range facesRaw {
		if f, ok := v.(float64); ok {
			faces[i] = int32(f)
		} else {
			return nil, fmt.Errorf("invalid face value at index %d", i)
		}
	}

	return &MeshData{
		Vertices: vertices,
		Faces:    faces,
	}, nil
}

// convertToVoronoiMeshData converts raw interface{} data to VoronoiMeshData structure
func convertToVoronoiMeshData(rawData map[string]interface{}) (*VoronoiMeshData, error) {
	verticesRaw, ok := rawData["vertices"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid voronoi vertices data")
	}
	
	cellsRaw, ok := rawData["cells"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid voronoi cells data")
	}

	// Convert vertices
	vertices := make([]float32, len(verticesRaw))
	for i, v := range verticesRaw {
		if f, ok := v.(float64); ok {
			vertices[i] = float32(f)
		} else {
			return nil, fmt.Errorf("invalid voronoi vertex value at index %d", i)
		}
	}

	// Convert cells
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

		// Convert neighbor indices
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

		// Convert vertex indices
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

	return &VoronoiMeshData{
		Vertices: vertices,
		Cells:    cells,
	}, nil
}

func testLandGenWithUpdatedParams(icosphereData *IcosphereResponse) (*LandGenResponse, error) {
	fmt.Println("\nTesting land generation with updated parameters...")

	// Convert the raw data to structured format
	meshData, err := convertToMeshData(icosphereData.IcosphereData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert icosphere data: %v", err)
	}

	voronoiData, err := convertToVoronoiMeshData(icosphereData.VoronoiData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert voronoi data: %v", err)
	}

	payload := LandGenRequest{
		// General land parameters
		LandSeed: 12345,

		// Updated tectonic parameters
		NumPlates:                   8,
		BaseSpeed:                   0.01,
		SpeedVariationFactor:        0.5,
		TargetContinentalProportion: 0.35,
		NumInitialContinentalSeeds:  3,

		// Elevation parameters
		NoiseScale:          1.0,
		NoiseOctaves:        4,
		NoisePersistence:    0.5,
		NoiseLacunarity:     2.0,
		ElevationMultiplier: 100,

		// New tectonic boundary effects
		CharacteristicFalloffDistance: 0.15,
		MaxBoundaryEffectDistance:     0.45,
		ConvergentBoundaryStrength:    1000,
		DivergentBoundaryStrength:     500,

		// Base mesh data
		BaseIcosphereData:     meshData,
		BaseVoronoiData:       voronoiData,
		IcosphereSubdivisions: icosphereData.SubdivisionLevel,
		LandOutputName:        "test_output.png",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal land generation request: %v", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post("http://localhost:8080/api/generate_land", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to make land generation request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read land generation response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("land generation failed with status %d: %s", resp.StatusCode, string(body))
	}

	var landGenResp LandGenResponse
	if err := json.Unmarshal(body, &landGenResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal land generation response: %v", err)
	}

	fmt.Printf("✅ Land generation successful\n")
	fmt.Printf("   - Status: %s\n", landGenResp.Status)

	// Check elevation data
	if elevationData, ok := landGenResp.ElevationData["cellElevations"].(map[string]interface{}); ok {
		var elevations []float64
		for _, v := range elevationData {
			if elev, ok := v.(float64); ok {
				elevations = append(elevations, elev)
			}
		}

		if len(elevations) > 0 {
			min, max := elevations[0], elevations[0]
			for _, elev := range elevations {
				if elev < min {
					min = elev
				}
				if elev > max {
					max = elev
				}
			}
			fmt.Printf("   - Elevation points: %d\n", len(elevations))
			fmt.Printf("   - Elevation range: %.1f to %.1f meters\n", min, max)
		} else {
			fmt.Printf("   - No elevation values found\n")
		}
	} else {
		fmt.Printf("   - No elevation data found\n")
	}

	return &landGenResp, nil
}

func main() {
	fmt.Println("🧪 Testing UI Integration with Modularized Backend")
	fmt.Println("==================================================")

	// Test icosphere generation first
	icosphereData, err := testIcosphereGeneration()
	if err != nil {
		fmt.Printf("❌ Icosphere test failed: %v\n", err)
		return
	}

	// Test land generation with new parameters
	_, err = testLandGenWithUpdatedParams(icosphereData)
	if err != nil {
		fmt.Printf("❌ Land generation test failed: %v\n", err)
		return
	}

	fmt.Println("\n✅ All tests passed! UI integration successful.")
}