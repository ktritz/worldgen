package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	// Added for default seed generation
	"worldgen/icosphere" // Your icosphere library
	"worldgen/landgen"   // Your landgen library (now an orchestrator)
)

// --- Struct Definitions ---

// Icosphere & Voronoi related structs (remain the same)
type GenerationParams struct {
	Subdivisions          int     `json:"subdivisions"`
	Out                   string  `json:"out"`
	RelaxEnable           bool    `json:"relaxEnable"`
	FixedInitial          bool    `json:"fixedInitial"`
	RelaxK                float64 `json:"relaxK"`
	RelaxDamping          float64 `json:"relaxDamping"`
	RelaxDtInitial        float64 `json:"relaxDtInitial"`
	RelaxMaxIterations    int     `json:"relaxMaxIterations"`
	RelaxTolerance        float64 `json:"relaxTolerance"`
	RelaxAdaptiveDt       bool    `json:"relaxAdaptiveDt"`
	RelaxDtMin            float64 `json:"relaxDtMin"`
	RelaxDtMax            float64 `json:"relaxDtMax"`
	RelaxDtIncreaseFactor float64 `json:"relaxDtIncreaseFactor"`
	RelaxDtDecreaseFactor float64 `json:"relaxDtDecreaseFactor"`
	RelaxMovThreshLow     float64 `json:"relaxMovThreshLow"`
	RelaxMovThreshHigh    float64 `json:"relaxMovThreshHigh"`
	VoronoiEnable         bool    `json:"voronoiEnable"`
	VoronoiOut            string  `json:"voronoiOut"`
	VoronoiNgonSave       bool    `json:"voronoiNgonSave"`
}

type MeshData struct {
	Vertices []float32 `json:"vertices"`
	Faces    []int32   `json:"faces"`
}

type VoronoiCellData struct {
	SiteIndex     int32   `json:"siteIndex"`
	VertexIndices []int32 `json:"vertexIndices"`
}

type VoronoiMeshData struct {
	Vertices []float32         `json:"vertices"`
	Cells    []VoronoiCellData `json:"cells"`
}

type GenerationResponse struct {
	Status              string           `json:"status"`
	Message             string           `json:"message"`
	IcosphereObjRelPath string           `json:"icosphereObjRelPath,omitempty"`
	VoronoiObjRelPath   string           `json:"voronoiObjRelPath,omitempty"`
	IcosphereData       *MeshData        `json:"icosphereData,omitempty"`
	VoronoiData         *VoronoiMeshData `json:"voronoiData,omitempty"`
}

type CheckSavedResponse struct {
	Subdivisions    int  `json:"subdivisions"`
	IcosphereExists bool `json:"icosphereExists"`
	VoronoiExists   bool `json:"voronoiExists"`
}

// --- Land Generation Structs ---

// LandGenerationRequestParams reflects what the frontend sends for the land generation tab.
// It includes existing elevation params and will be used to populate LandGenerationPipelineSettings.
type LandGenerationRequestParams struct {
	// Seed from UI, will be used as GlobalSeed
	LandSeed int64 `json:"landSeed"`

	// Parameters for Tectonic Plate Generation (defaults will be used if not provided by UI)
	NumPlates   int     `json:"numPlates,omitempty"`
	BaseSpeed   float64 `json:"baseSpeed,omitempty"`
	SpeedFactor float64 `json:"speedFactor,omitempty"`
	PConvergent float64 `json:"pConvergent,omitempty"`
	PDivergent  float64 `json:"pDivergent,omitempty"`
	// TectonicSeed will be derived from LandSeed/GlobalSeed in landgen package if not set directly

	// Parameters for Elevation Generation (from existing UI form)
	NoiseScale          float64 `json:"noiseScale"`
	NoiseOctaves        int     `json:"noiseOctaves"`
	NoisePersistence    float64 `json:"noisePersistence"`
	NoiseLacunarity     float64 `json:"noiseLacunarity"`
	ElevationMultiplier float64 `json:"elevationMultiplier"`

	// Output and Base Mesh Data
	LandOutputName        string           `json:"landOutputName"` // For auxiliary files
	BaseIcosphereData     *MeshData        `json:"baseIcosphereData"`
	BaseVoronoiData       *VoronoiMeshData `json:"baseVoronoiData"`
	IcosphereSubdivisions int              `json:"icosphereSubdivisions"`
}

// LandGenerationResponse is sent back to the client.
// It primarily contains ElevationData for visualization.
type LandGenerationResponse struct {
	Status        string                 `json:"status"`
	Message       string                 `json:"message"`
	HeightmapUrl  string                 `json:"heightmapUrl,omitempty"` // For auxiliary image output
	ElevationData *landgen.ElevationData `json:"elevationData,omitempty"`
}

// --- Helper Functions (flattenVertices, etc. remain the same) ---
func flattenVertices(vertices []icosphere.Vector3D) []float32 {
	flat := make([]float32, 0, len(vertices)*3)
	for _, v := range vertices {
		flat = append(flat, float32(v.X), float32(v.Y), float32(v.Z))
	}
	return flat
}

func flattenFaces(faces []icosphere.Triangle) []int32 {
	flat := make([]int32, 0, len(faces)*3)
	for _, f := range faces {
		flat = append(flat, int32(f.V1), int32(f.V2), int32(f.V3))
	}
	return flat
}

func unflattenToVector3D(flatVertices []float32) []icosphere.Vector3D {
	if len(flatVertices)%3 != 0 {
		log.Printf("Warning: unflattenToVector3D received flatVertices length not divisible by 3: %d", len(flatVertices))
		return []icosphere.Vector3D{}
	}
	numVectors := len(flatVertices) / 3
	vectors := make([]icosphere.Vector3D, numVectors)
	for i := 0; i < numVectors; i++ {
		vectors[i] = icosphere.Vector3D{
			X: float64(flatVertices[i*3+0]),
			Y: float64(flatVertices[i*3+1]),
			Z: float64(flatVertices[i*3+2]),
		}
	}
	return vectors
}

func unflattenToTriangles(flatFaces []int32, numVertices int) []icosphere.Triangle {
	if len(flatFaces)%3 != 0 {
		log.Printf("Warning: unflattenToTriangles received flatFaces length not divisible by 3: %d", len(flatFaces))
		return []icosphere.Triangle{}
	}
	numTriangles := len(flatFaces) / 3
	triangles := make([]icosphere.Triangle, numTriangles)
	validTriangleCount := 0
	for i := 0; i < numTriangles; i++ {
		v1 := int(flatFaces[i*3+0])
		v2 := int(flatFaces[i*3+1])
		v3 := int(flatFaces[i*3+2])
		// Basic validation: check if indices are within bounds
		if v1 >= 0 && v1 < numVertices && v2 >= 0 && v2 < numVertices && v3 >= 0 && v3 < numVertices {
			triangles[validTriangleCount] = icosphere.Triangle{V1: v1, V2: v2, V3: v3}
			validTriangleCount++
		} else {
			log.Printf("Warning: Invalid face index found when unflattening: V1=%d, V2=%d, V3=%d (NumVertices=%d)", v1, v2, v3, numVertices)
		}
	}
	return triangles[:validTriangleCount]
}

func convertToIcosphereVoronoiCells(jsonDataCells []VoronoiCellData) []icosphere.VoronoiCell {
	icosphereCells := make([]icosphere.VoronoiCell, len(jsonDataCells))
	for i, dataCell := range jsonDataCells {
		icosphereCells[i] = icosphere.VoronoiCell{
			SiteIndex:     dataCell.SiteIndex,
			VertexIndices: dataCell.VertexIndices,
		}
	}
	return icosphereCells
}

func convertToJSONVoronoiCellData(icosphereCells []icosphere.VoronoiCell) []VoronoiCellData {
	jsonDataCells := make([]VoronoiCellData, len(icosphereCells))
	for i, icoCell := range icosphereCells {
		jsonDataCells[i] = VoronoiCellData{
			SiteIndex:     icoCell.SiteIndex,
			VertexIndices: icoCell.VertexIndices,
		}
	}
	return jsonDataCells
}

func saveJSONData(filePath string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON for %s: %w", filePath, err)
	}
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON data to file %s: %w", filePath, err)
	}
	log.Printf("Successfully saved JSON data to %s", filePath)
	return nil
}

func loadJSONData(filePath string, target interface{}) error {
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file %s: %w", filePath, err)
	}
	err = json.Unmarshal(jsonData, target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON data from %s: %w", filePath, err)
	}
	log.Printf("Successfully loaded JSON data from %s", filePath)
	return nil
}

// --- HTTP Handlers ---

func homeHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Serving index.html for %s from %s\n", r.URL.Path, r.RemoteAddr)
	http.ServeFile(w, r, "./static/index.html")
}

// apiGenerateHandler for Icosphere and Voronoi (remains largely the same)
func apiGenerateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var params GenerationParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Printf("Error decoding icosphere/voronoi JSON request: %v", err)
		http.Error(w, fmt.Sprintf("Error decoding JSON request: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("API: Received icosphere/voronoi generation request with params: %+v\n", params)

	outputDir := "./output_from_server"
	jsonCacheDir := "./mesh_cache"

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("Error creating output directory %s: %v", outputDir, err)
		http.Error(w, "Failed to create output directory", http.StatusInternalServerError)
		return
	}
	if err := os.MkdirAll(jsonCacheDir, 0755); err != nil {
		log.Printf("Error creating JSON cache directory %s: %v", jsonCacheDir, err)
		http.Error(w, "Failed to create JSON cache directory", http.StatusInternalServerError)
		return
	}

	icosphereObjRelPath := params.Out
	voronoiObjRelPath := params.VoronoiOut
	icosphereObjFullPath := filepath.Join(outputDir, icosphereObjRelPath)
	voronoiObjFullPath := filepath.Join(outputDir, voronoiObjRelPath)

	icosphereJSONFilename := fmt.Sprintf("icosphere_sub%d_data.json", params.Subdivisions)
	icosphereJSONFullPath := filepath.Join(jsonCacheDir, icosphereJSONFilename)

	voronoiJSONFilename := fmt.Sprintf("voronoi_sub%d_data.json", params.Subdivisions)
	voronoiJSONFullPath := filepath.Join(jsonCacheDir, voronoiJSONFilename)

	apiResponse := GenerationResponse{
		Status:              "processing",
		Message:             "Starting icosphere generation...",
		IcosphereObjRelPath: icosphereObjRelPath,
	}

	log.Println("API: Generating icosphere...")
	icoVertices, icoFaces := icosphere.CreateIcosphere(params.Subdivisions)
	log.Printf("API: Icosphere generated with %d vertices, %d faces", len(icoVertices), len(icoFaces))

	currentVertices := icoVertices
	currentFaces := icoFaces

	if params.RelaxEnable {
		relaxParams := icosphere.RelaxMeshParameters{
			FixedInitial: params.FixedInitial, K: params.RelaxK, Damping: params.RelaxDamping,
			DtInitial: params.RelaxDtInitial, MaxIterations: params.RelaxMaxIterations, Tolerance: params.RelaxTolerance,
			AdaptiveDt: params.RelaxAdaptiveDt, DtMin: params.RelaxDtMin, DtMax: params.RelaxDtMax,
			DtIncreaseFactor: params.RelaxDtIncreaseFactor, DtDecreaseFactor: params.RelaxDtDecreaseFactor,
			MovementThresholdLow: params.RelaxMovThreshLow, MovementThresholdHigh: params.RelaxMovThreshHigh,
		}
		log.Println("API: Relaxing icosphere mesh...")
		icosphere.RelaxMesh(currentVertices, currentFaces, relaxParams)
		log.Println("API: Mesh relaxation complete.")
	}

	apiResponse.IcosphereData = &MeshData{
		Vertices: flattenVertices(currentVertices),
		Faces:    flattenFaces(currentFaces),
	}

	if err := saveJSONData(icosphereJSONFullPath, apiResponse.IcosphereData); err != nil {
		log.Printf("Error saving icosphere JSON data: %v", err)
	}
	if err := icosphere.SaveOBJTriangulated(icosphereObjFullPath, currentVertices, currentFaces, "Icosphere mesh"); err != nil {
		log.Printf("Error saving icosphere OBJ: %v", err)
	} else {
		log.Printf("API: Saved icosphere OBJ to %s", icosphereObjFullPath)
	}

	if params.VoronoiEnable {
		apiResponse.VoronoiObjRelPath = voronoiObjRelPath
		log.Println("API: Generating Voronoi diagram...")
		voronoiMeshVertices, voronoiCellsStructs := icosphere.GenerateSphericalVoronoi(currentVertices, currentFaces)
		log.Printf("API: Voronoi generated with %d vertices, %d cells", len(voronoiMeshVertices), len(voronoiCellsStructs))

		apiResponse.VoronoiData = &VoronoiMeshData{
			Vertices: flattenVertices(voronoiMeshVertices),
			Cells:    convertToJSONVoronoiCellData(voronoiCellsStructs),
		}
		if err := saveJSONData(voronoiJSONFullPath, apiResponse.VoronoiData); err != nil {
			log.Printf("Error saving Voronoi JSON data: %v", err)
		}
		var errObj error
		if params.VoronoiNgonSave {
			errObj = icosphere.SaveVoronoiOBJ_NGon(voronoiObjFullPath, voronoiMeshVertices, voronoiCellsStructs, "N-gon Spherical Voronoi mesh")
		} else {
			errObj = icosphere.SaveVoronoiOBJTriangulated(voronoiObjFullPath, voronoiMeshVertices, voronoiCellsStructs, "Triangulated Spherical Voronoi mesh")
		}
		if errObj != nil {
			log.Printf("Error saving Voronoi OBJ: %v", errObj)
		} else {
			log.Printf("API: Saved Voronoi OBJ to %s", voronoiObjFullPath)
		}
	}

	apiResponse.Status = "success"
	apiResponse.Message = "Icosphere/Voronoi generation process complete. Data saved to JSON cache."
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiResponse)
}

// apiLoadMeshHandler (remains the same)
func apiLoadMeshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}
	querySubdivisions := r.URL.Query().Get("subdivisions")
	queryType := r.URL.Query().Get("type")
	if querySubdivisions == "" {
		http.Error(w, "Missing 'subdivisions' query parameter", http.StatusBadRequest)
		return
	}
	if queryType == "" {
		http.Error(w, "Missing 'type' query parameter (icosphere, voronoi, both)", http.StatusBadRequest)
		return
	}
	subdivisions, err := strconv.Atoi(querySubdivisions)
	if err != nil {
		http.Error(w, "'subdivisions' must be an integer", http.StatusBadRequest)
		return
	}

	jsonCacheDir := "./mesh_cache"
	response := GenerationResponse{Status: "success", Message: ""}
	loadedSomething := false

	if queryType == "icosphere" || queryType == "both" {
		icosphereJSONFilename := fmt.Sprintf("icosphere_sub%d_data.json", subdivisions)
		icosphereJSONFullPath := filepath.Join(jsonCacheDir, icosphereJSONFilename)
		var icoData MeshData
		if err := loadJSONData(icosphereJSONFullPath, &icoData); err == nil {
			response.IcosphereData = &icoData
			response.Message += fmt.Sprintf("Loaded icosphere data for subdivision %d. ", subdivisions)
			loadedSomething = true
		} else {
			log.Printf("Could not load icosphere data for sub %d: %v", subdivisions, err)
			if queryType == "icosphere" { // Only error out if specifically requesting icosphere and it fails
				response.Status = "error"
				response.Message = fmt.Sprintf("Failed to load icosphere data for subdivision %d: File not found or corrupt.", subdivisions)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(response)
				return
			}
			response.Message += fmt.Sprintf("No cached icosphere data found for subdivision %d. ", subdivisions)
		}
	}

	if queryType == "voronoi" || queryType == "both" {
		voronoiJSONFilename := fmt.Sprintf("voronoi_sub%d_data.json", subdivisions)
		voronoiJSONFullPath := filepath.Join(jsonCacheDir, voronoiJSONFilename)
		var voroData VoronoiMeshData
		if err := loadJSONData(voronoiJSONFullPath, &voroData); err == nil {
			response.VoronoiData = &voroData
			response.Message += fmt.Sprintf("Loaded Voronoi data for subdivision %d. ", subdivisions)
			loadedSomething = true
		} else {
			log.Printf("Could not load Voronoi data for sub %d: %v", subdivisions, err)
			if queryType == "voronoi" { // Only error out if specifically requesting voronoi and it fails
				response.Status = "error"
				response.Message = fmt.Sprintf("Failed to load Voronoi data for subdivision %d: File not found or corrupt.", subdivisions)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(response)
				return
			}
			response.Message += fmt.Sprintf("No cached Voronoi data found for subdivision %d. ", subdivisions)
		}
	}

	if !loadedSomething && queryType == "both" {
		response.Status = "error"
		response.Message = fmt.Sprintf("No cached icosphere or Voronoi data found for subdivision %d.", subdivisions)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	if response.Message == "" { // Should not happen if loadedSomething is true, but as a fallback
		response.Message = fmt.Sprintf("No data loaded for type %s, subdivision %d.", queryType, subdivisions)
		if response.Status == "success" {
			response.Status = "partial" // If it was success but message is empty, something is off
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// apiCheckSavedMeshHandler (remains the same)
func apiCheckSavedMeshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}
	querySubdivisions := r.URL.Query().Get("subdivisions")
	if querySubdivisions == "" {
		http.Error(w, "Missing 'subdivisions' query parameter", http.StatusBadRequest)
		return
	}
	subdivisions, err := strconv.Atoi(querySubdivisions)
	if err != nil {
		http.Error(w, "'subdivisions' must be an integer", http.StatusBadRequest)
		return
	}

	jsonCacheDir := "./mesh_cache"
	response := CheckSavedResponse{Subdivisions: subdivisions}

	icosphereJSONFilename := fmt.Sprintf("icosphere_sub%d_data.json", subdivisions)
	icosphereJSONFullPath := filepath.Join(jsonCacheDir, icosphereJSONFilename)
	if _, err := os.Stat(icosphereJSONFullPath); err == nil {
		response.IcosphereExists = true
	}

	voronoiJSONFilename := fmt.Sprintf("voronoi_sub%d_data.json", subdivisions)
	voronoiJSONFullPath := filepath.Join(jsonCacheDir, voronoiJSONFilename)
	if _, err := os.Stat(voronoiJSONFullPath); err == nil {
		response.VoronoiExists = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// --- Land Generation Handler (Updated) ---
func apiGenerateLandHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqPayload LandGenerationRequestParams
	if err := json.NewDecoder(r.Body).Decode(&reqPayload); err != nil {
		log.Printf("Error decoding land generation JSON request: %v", err)
		http.Error(w, fmt.Sprintf("Error decoding JSON request: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("API: Received land generation request with params: %+v\n", reqPayload)

	if reqPayload.BaseIcosphereData == nil || reqPayload.BaseVoronoiData == nil {
		log.Println("API (LandGen): Missing base Icosphere or Voronoi data in request.")
		http.Error(w, "Missing baseIcosphereData or baseVoronoiData in request payload.", http.StatusBadRequest)
		return
	}

	// --- Construct LandGenerationPipelineSettings ---
	pipelineSettings := landgen.LandGenerationPipelineSettings{
		GlobalSeed: reqPayload.LandSeed, // Use landSeed from UI as the global seed
		TectonicSettings: landgen.TectonicSettings{
			NumPlates:   reqPayload.NumPlates,
			Seed:        0, // Will default to GlobalSeed in landgen.GeneratePlanetData if 0
			BaseSpeed:   reqPayload.BaseSpeed,
			SpeedFactor: reqPayload.SpeedFactor,
			PConvergent: reqPayload.PConvergent,
			PDivergent:  reqPayload.PDivergent,
			// NumWorkers is now handled automatically in tectonics.go
		},
		ElevationSettings: landgen.ElevationSettings{
			NoiseScale:          reqPayload.NoiseScale,
			NoiseOctaves:        reqPayload.NoiseOctaves,
			NoisePersistence:    reqPayload.NoisePersistence,
			NoiseLacunarity:     reqPayload.NoiseLacunarity,
			ElevationMultiplier: reqPayload.ElevationMultiplier,
		},
		OutputPath:           filepath.Join("./output_from_server", "land"), // Example base output path
		OutputAuxiliaryFiles: true,                                          // Example
	}

	// Set defaults for TectonicSettings if not provided by UI (or if UI doesn't send them yet)
	if pipelineSettings.TectonicSettings.NumPlates == 0 {
		pipelineSettings.TectonicSettings.NumPlates = 10 // Default number of plates
		log.Printf("  LandGen: NumPlates not provided, defaulting to %d", pipelineSettings.TectonicSettings.NumPlates)
	}
	if pipelineSettings.TectonicSettings.BaseSpeed == 0 {
		pipelineSettings.TectonicSettings.BaseSpeed = 0.01 // Default base speed
		log.Printf("  LandGen: BaseSpeed not provided, defaulting to %f", pipelineSettings.TectonicSettings.BaseSpeed)
	}
	if pipelineSettings.TectonicSettings.SpeedFactor == 0 {
		pipelineSettings.TectonicSettings.SpeedFactor = 1.0 // Default speed factor
		log.Printf("  LandGen: SpeedFactor not provided, defaulting to %f", pipelineSettings.TectonicSettings.SpeedFactor)
	}
	if pipelineSettings.TectonicSettings.PConvergent == 0 && pipelineSettings.TectonicSettings.PDivergent == 0 {
		pipelineSettings.TectonicSettings.PConvergent = 0.4 // Default probability
		pipelineSettings.TectonicSettings.PDivergent = 0.4  // Default probability
		log.Printf("  LandGen: PConvergent/PDivergent not provided, defaulting to 0.4 each")
	}

	// --- Prepare base mesh data for the pipeline ---
	baseIcoSites := unflattenToVector3D(reqPayload.BaseIcosphereData.Vertices)
	baseIcoFaces := unflattenToTriangles(reqPayload.BaseIcosphereData.Faces, len(baseIcoSites))
	baseVoroVertices := unflattenToVector3D(reqPayload.BaseVoronoiData.Vertices)
	baseVoroCells := convertToIcosphereVoronoiCells(reqPayload.BaseVoronoiData.Cells)

	if len(baseIcoSites) == 0 {
		log.Println("API (LandGen): Base Icosphere sites are empty after unflattening.")
		http.Error(w, "Provided base icosphere data is invalid or empty.", http.StatusBadRequest)
		return
	}
	if len(baseIcoFaces) == 0 && reqPayload.IcosphereSubdivisions > 0 { // Faces are important for adjacency
		log.Println("API (LandGen): Base Icosphere faces are empty after unflattening (and subdivisions > 0).")
		// http.Error(w, "Provided base icosphere faces data is invalid or empty.", http.StatusBadRequest)
		// return // Allow proceeding if it's a 0-subdivision icosphere (just points for Voronoi sites)
	}

	// --- Call the land generation pipeline ---
	log.Println("API (LandGen): Calling landgen.GeneratePlanetData...")
	planetData, err := landgen.GeneratePlanetData(
		pipelineSettings,
		baseIcoSites,
		baseIcoFaces,
		baseVoroVertices,
		baseVoroCells,
		reqPayload.IcosphereSubdivisions,
	)
	if err != nil {
		log.Printf("Error generating planet data: %v", err)
		http.Error(w, fmt.Sprintf("Failed to generate planet data: %v", err), http.StatusInternalServerError)
		return
	}

	// --- Prepare and send the response ---
	// The client currently expects ElevationData for visualization.
	apiResponse := LandGenerationResponse{
		Status:        "success",
		Message:       "Land generation pipeline completed successfully.",
		ElevationData: planetData.ElevationData, // Extract ElevationData from PlanetData
		// HeightmapUrl could be set if an auxiliary image was saved by a module
	}
	if pipelineSettings.OutputAuxiliaryFiles && reqPayload.LandOutputName != "" {
		apiResponse.HeightmapUrl = filepath.ToSlash(filepath.Join("/output/land", reqPayload.LandOutputName))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiResponse)
}

// --- Main Function ---
func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	outputFs := http.FileServer(http.Dir("./output_from_server"))
	http.Handle("/output/", http.StripPrefix("/output/", outputFs))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/generate", apiGenerateHandler)
	http.HandleFunc("/api/load_mesh", apiLoadMeshHandler)
	http.HandleFunc("/api/check_saved_mesh", apiCheckSavedMeshHandler)
	http.HandleFunc("/api/generate_land", apiGenerateLandHandler) // New endpoint

	port := "8080"
	log.Printf("Starting server on http://localhost:%s\n", port)
	// ... (other log messages)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("ListenAndServe Error: %v", err)
	}
}
