package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"worldgen/icosphere" // Your icosphere library
	"worldgen/landgen"   // Your new landgen library
)

// --- Struct Definitions ---

// Icosphere & Voronoi related structs
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
type LandGenerationRequestParams struct {
	Seed                int     `json:"landSeed"`
	NoiseScale          float64 `json:"noiseScale"`
	NoiseOctaves        int     `json:"noiseOctaves"`
	NoisePersistence    float64 `json:"noisePersistence"`
	NoiseLacunarity     float64 `json:"noiseLacunarity"`
	ElevationMultiplier float64 `json:"elevationMultiplier"`
	OutputName          string  `json:"landOutputName"`

	BaseIcosphereData *MeshData        `json:"baseIcosphereData"` // Contains vertices for sites
	BaseVoronoiData   *VoronoiMeshData `json:"baseVoronoiData"`
}

type LandGenerationResponse struct {
	Status        string                 `json:"status"`
	Message       string                 `json:"message"`
	HeightmapUrl  string                 `json:"heightmapUrl,omitempty"`
	ElevationData *landgen.ElevationData `json:"elevationData,omitempty"`
}

// --- Helper Functions ---

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
			FixedInitial:          params.FixedInitial,
			K:                     params.RelaxK,
			Damping:               params.RelaxDamping,
			DtInitial:             params.RelaxDtInitial,
			MaxIterations:         params.RelaxMaxIterations,
			Tolerance:             params.RelaxTolerance,
			AdaptiveDt:            params.RelaxAdaptiveDt,
			DtMin:                 params.RelaxDtMin,
			DtMax:                 params.RelaxDtMax,
			DtIncreaseFactor:      params.RelaxDtIncreaseFactor,
			DtDecreaseFactor:      params.RelaxDtDecreaseFactor,
			MovementThresholdLow:  params.RelaxMovThreshLow,
			MovementThresholdHigh: params.RelaxMovThreshHigh,
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
			if queryType == "icosphere" {
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
			if queryType == "voronoi" {
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

	if response.Message == "" {
		response.Message = fmt.Sprintf("No data loaded for type %s, subdivision %d.", queryType, subdivisions)
		if response.Status == "success" {
			response.Status = "partial"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
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

// --- Land Generation Handler (Uses active mesh data sent from frontend) ---
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

	log.Printf("API: Received land generation request with land params: %+v\n", reqPayload)

	if reqPayload.BaseIcosphereData == nil || reqPayload.BaseVoronoiData == nil {
		log.Println("API (LandGen): Missing base Icosphere or Voronoi data in request.")
		http.Error(w, "Missing baseIcosphereData or baseVoronoiData in request payload.", http.StatusBadRequest)
		return
	}

	// --- Step 1: Unflatten/Convert received base mesh data ---
	icoSites := unflattenToVector3D(reqPayload.BaseIcosphereData.Vertices)

	voroVerticesForLandgen := unflattenToVector3D(reqPayload.BaseVoronoiData.Vertices)
	voroCellsForLandgen := convertToIcosphereVoronoiCells(reqPayload.BaseVoronoiData.Cells)

	if len(icoSites) == 0 || len(voroCellsForLandgen) == 0 {
		log.Println("API (LandGen): Base Icosphere sites or Voronoi cells are empty after unflattening.")
		http.Error(w, "Provided base mesh data is invalid or empty.", http.StatusBadRequest)
		return
	}
	log.Printf("API (LandGen): Using provided base mesh data - Icosphere Sites: %d, Voronoi Cells: %d, Voronoi Vertices: %d\n",
		len(icoSites), len(voroCellsForLandgen), len(voroVerticesForLandgen))

	// --- Step 2: Prepare parameters for the landgen library ---
	landLibParams := landgen.LandGenerationParams{
		Seed:                reqPayload.Seed,
		NoiseScale:          reqPayload.NoiseScale,
		NoiseOctaves:        reqPayload.NoiseOctaves,
		NoisePersistence:    reqPayload.NoisePersistence,
		NoiseLacunarity:     reqPayload.NoiseLacunarity,
		ElevationMultiplier: reqPayload.ElevationMultiplier,
		OutputName:          reqPayload.OutputName,
	}

	// --- Step 3: Define output path for any auxiliary generated files ---
	landFileOutputDir := filepath.Join("./output_from_server", "land")
	if err := os.MkdirAll(landFileOutputDir, 0755); err != nil {
		log.Printf("Error creating land output directory %s: %v", landFileOutputDir, err)
		http.Error(w, "Failed to create land output directory", http.StatusInternalServerError)
		return
	}
	fullOutputFilePath := filepath.Join(landFileOutputDir, reqPayload.OutputName)

	// --- Step 4: Call the land generation library function ---
	log.Println("API (LandGen): Calling landgen.GenerateLandData...")
	// Call now matches the revised landgen.GenerateLandData signature (4 args)
	elevationData, err := landgen.GenerateLandData(landLibParams, icoSites, voroVerticesForLandgen, voroCellsForLandgen, fullOutputFilePath)
	if err != nil {
		log.Printf("Error generating land data: %v", err)
		http.Error(w, fmt.Sprintf("Failed to generate land data: %v", err), http.StatusInternalServerError)
		return
	}

	// --- Step 5: Prepare and send the response ---
	apiResponse := LandGenerationResponse{
		Status:        "success",
		Message:       "Land data generated successfully.",
		ElevationData: elevationData,
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
	http.HandleFunc("/api/generate_land", apiGenerateLandHandler)

	port := "8080"
	log.Printf("Starting server on http://localhost:%s\n", port)
	log.Println("Serving static files from ./static/ folder under /static/ route")
	log.Println("Serving generated output files from ./output_from_server/ folder under /output/ route")
	log.Println("Mesh data (Icosphere/Voronoi) will be cached in ./mesh_cache/")
	log.Println("Land generation output (e.g., images) will be in ./output_from_server/land/")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("ListenAndServe Error: %v", err)
	}
}
