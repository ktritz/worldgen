package landgen

import (
	"fmt"
	"math"
	"time" // Used for default seed generation if none is provided

	// Assuming your icosphere package provides Vector3D and Triangle types.
	// If tectonics.go is in the same 'landgen' package, direct calls are fine.
	// If icosphere types are needed directly here (e.g. in PlanetData), it must be imported.
	"worldgen/icosphere"
)

// PlanetData will hold all generated data for the planet.
// This struct will be expanded as more modules (hotspots, climate, erosion) are added.
type PlanetData struct {
	IcosphereSites            []icosphere.Vector3D // The original sites used for Voronoi generation
	IcosphereFaces            []icosphere.Triangle // Faces of the icosphere, useful for adjacency
	NumIcosphereVertices      int                  // Number of vertices in the icosphere (sites)
	VoronoiVertices           []icosphere.Vector3D // Vertices of the Voronoi cells
	VoronoiCells              []icosphere.VoronoiCell
	TectonicData              *TectonicsData // Populated by the tectonics module
	ElevationData             *ElevationData // Populated by the elevation module
	HotspotData               interface{}    // Placeholder for future hotspot data
	ClimateData               interface{}    // Placeholder for future climate data
	ErosionData               interface{}    // Placeholder for future erosion data
	BaseIcosphereSubdivisions int            // Subdivision level of the base icosphere for reference
}

// LandGenerationPipelineSettings holds all settings for the entire land generation pipeline.
// This will be populated from the server based on UI inputs.
type LandGenerationPipelineSettings struct {
	GlobalSeed           int64             `json:"globalSeed"` // A main seed, can be used to derive other seeds
	TectonicSettings     TectonicSettings  `json:"tectonicSettings"`
	ElevationSettings    ElevationSettings `json:"elevationSettings"`    // Settings for the elevation generation step
	OutputPath           string            `json:"outputPath"`           // Base path for any output files from modules
	OutputAuxiliaryFiles bool              `json:"outputAuxiliaryFiles"` // Flag to control saving of debug/intermediate files
	// Add other module settings here, e.g., HotspotSettings, ErosionSettings
}

// ElevationSettings holds parameters specifically for the elevation generation module.
// This replaces parts of the old LandGenerationParams from the original landgen.go.
type ElevationSettings struct {
	NoiseScale          float64 `json:"noiseScale"`          // Example: Used as frequency for placeholder sine wave
	NoiseOctaves        int     `json:"noiseOctaves"`        // Example: For more complex noise (not fully used by current placeholder)
	NoisePersistence    float64 `json:"noisePersistence"`    // Example: For more complex noise
	NoiseLacunarity     float64 `json:"noiseLacunarity"`     // Example: For more complex noise
	ElevationMultiplier float64 `json:"elevationMultiplier"` // Example: Used as amplitude for placeholder sine wave
	// Future params: base_height, mountain_factor, continental_shelf_depth, etc.
}

// ElevationData represents the output of the elevation generation module.
// It stores the calculated elevation for each Voronoi cell, keyed by the original icosphere site ID.
type ElevationData struct {
	CellElevations map[int32]float64 `json:"cellElevations"`
	// Could also include vertex_elevations if direct rendering on Voronoi vertices is needed without texture lookup.
}

// GeneratePlanetData is the main orchestrator function for the land generation pipeline.
// It takes the pipeline settings and the necessary base mesh data (icosphere sites, faces,
// Voronoi vertices, and cells) as input.
// It then calls various sub-modules (like tectonics, elevation generation, etc.) in sequence.
// Returns the fully populated PlanetData struct or an error.
func GeneratePlanetData(
	pipelineSettings LandGenerationPipelineSettings,
	baseIcoSites []icosphere.Vector3D,
	baseIcoFaces []icosphere.Triangle,
	baseVoroVertices []icosphere.Vector3D,
	baseVoroCells []icosphere.VoronoiCell,
	baseIcosphereSubdivisions int,
) (*PlanetData, error) {

	fmt.Println("--- Starting Land Generation Pipeline ---")

	// Initialize the main data structure to hold all planetary information.
	planet := &PlanetData{
		IcosphereSites:            baseIcoSites,
		IcosphereFaces:            baseIcoFaces,
		NumIcosphereVertices:      len(baseIcoSites), // Store the count for convenience
		VoronoiVertices:           baseVoroVertices,
		VoronoiCells:              baseVoroCells,
		BaseIcosphereSubdivisions: baseIcosphereSubdivisions,
		TectonicData:              &TectonicsData{},                                        // Initialize to an empty TectonicsData struct
		ElevationData:             &ElevationData{CellElevations: make(map[int32]float64)}, // Initialize map
		// Initialize other data placeholders as nil or empty structs as appropriate
	}

	// --- 1. Tectonic Plate Generation Module ---
	fmt.Println("\n-- Running Tectonics Module --")
	// Ensure the tectonic module has a seed. If not set in its specific settings,
	// derive it from the global seed or generate one.
	if pipelineSettings.TectonicSettings.Seed == 0 {
		if pipelineSettings.GlobalSeed != 0 {
			fmt.Printf("  Tectonics: Using global seed %d as tectonic-specific seed is 0.\n", pipelineSettings.GlobalSeed)
			pipelineSettings.TectonicSettings.Seed = pipelineSettings.GlobalSeed
		} else {
			// Fallback if no seeds are provided at all (UI should ideally prevent this)
			pipelineSettings.TectonicSettings.Seed = time.Now().UnixNano()
			fmt.Printf("  Tectonics: No global or tectonic seed provided, using current time as seed: %d\n", pipelineSettings.TectonicSettings.Seed)
		}
	}

	// Create the tectonic plates.
	planet.TectonicData.Plates = CreateTectonicPlates(pipelineSettings.TectonicSettings)
	if len(planet.TectonicData.Plates) == 0 {
		return nil, fmt.Errorf("tectonics module failed to create plates")
	}

	// Assign icosphere sites to the generated tectonic plates.
	// The icosphere sites act as representatives for the Voronoi regions.
	sitePlateIDs := AssignVerticesToPlates(planet.IcosphereSites, planet.TectonicData.Plates)
	if len(sitePlateIDs) != len(planet.IcosphereSites) {
		return nil, fmt.Errorf("tectonics module: sitePlateIDs length mismatch after assignment")
	}

	// Find plate boundaries. This analysis is done on the icosphere mesh (sites and Delaunay faces).
	// The results (isBoundarySite, siteBoundaryTypes, etc.) will refer to the icosphere sites.
	fmt.Println("  Finding plate boundaries based on icosphere sites and their Delaunay triangulation (IcosphereFaces)...")
	isBoundarySite, siteBoundaryTypes, siteDistToBoundary, nearestBoundarySite, adjPlateInteractions := FindPlateBoundariesAndTypes(
		planet.IcosphereSites,             // Vertices being analyzed are the icosphere sites
		planet.IcosphereFaces,             // Faces are the Delaunay triangles connecting these sites
		sitePlateIDs,                      // Plate ID assigned to each icosphere site
		planet.TectonicData.Plates,        // Full plate data for velocity calculations
		pipelineSettings.TectonicSettings, // Settings for the tectonics module
	)
	// Store the results in our main PlanetData structure.
	planet.TectonicData.VertexPlateIDs = sitePlateIDs // These are IDs for *sites*
	planet.TectonicData.IsBoundaryVertex = isBoundarySite
	planet.TectonicData.VertexBoundaryTypes = siteBoundaryTypes
	planet.TectonicData.VertexDistancesToBoundary = siteDistToBoundary
	planet.TectonicData.NearestBoundaryIndices = nearestBoundarySite
	planet.TectonicData.AdjacentPlateInteractions = adjPlateInteractions
	fmt.Println("-- Tectonics Module Complete --")

	// --- 2. Hotspot Generation (Placeholder for future module) ---
	fmt.Println("\n-- Running Hotspots Module (Placeholder) --")
	// Example call:
	// planet.HotspotData, err = hotspots.GenerateHotspots(planet, pipelineSettings.HotspotSettings)
	// if err != nil { return nil, fmt.Errorf("hotspot module failed: %w", err) }
	fmt.Println("-- Hotspots Module Complete --")

	// --- 3. Elevation Generation Module ---
	fmt.Println("\n-- Running Elevation Module --")
	// The seed for elevation can be derived from the global seed or be specific to elevation settings.
	elevationSeed := pipelineSettings.GlobalSeed
	if pipelineSettings.ElevationSettings.NoiseScale > 0 { // A simple check if specific settings are active
		// Example: Use a different seed derived from global for elevation processes
		// elevationSeed = pipelineSettings.GlobalSeed + 1 // Or a dedicated seed in ElevationSettings
	}

	// Parameters for the current placeholder elevation logic (sinusoidal + tectonic influence)
	frequencyFactor := pipelineSettings.ElevationSettings.NoiseScale * 5.0
	phaseShift := float64(int(elevationSeed)%1000) * 0.01 // Ensure phase shift is somewhat controlled by seed
	baseAmplitude := pipelineSettings.ElevationSettings.ElevationMultiplier

	// Parameters for tectonic feature influence on elevation.
	// These should ideally be part of ElevationSettings or TectonicSettings for UI control.
	characteristicFalloffDistance := 0.15 // Relative to planet radius (e.g., 0.15 means 15%)
	maxBoundaryEffectDistance := characteristicFalloffDistance * 3.0
	convergentBoundaryStrength := baseAmplitude * 3.0 // Multiplier for convergent uplift
	divergentBoundaryStrength := baseAmplitude * 2.0  // Multiplier for divergent subsidence (negative effect)

	// Generate elevation for each Voronoi cell (represented by its icosphere site).
	for i, site := range planet.IcosphereSites {
		siteID := int32(i)
		// Start with a base sinusoidal elevation pattern (placeholder for more complex noise).
		baseElevation := math.Sin(site.X*frequencyFactor+phaseShift) * baseAmplitude

		// Modify elevation based on tectonic features.
		distToBoundary := planet.TectonicData.VertexDistancesToBoundary[siteID] // Distance of this *site* to nearest boundary *site*

		// Check if the site is within the influence range of a boundary.
		if distToBoundary < maxBoundaryEffectDistance {
			nearestBoundarySiteIndex := planet.TectonicData.NearestBoundaryIndices[siteID]
			if nearestBoundarySiteIndex != -1 { // Ensure a valid nearest boundary was found
				// Calculate falloff factor: 1.0 at boundary, fading to 0.0 at maxBoundaryEffectDistance.
				// A squared falloff creates a more pronounced effect near the boundary.
				falloffFactor := (maxBoundaryEffectDistance - distToBoundary) / maxBoundaryEffectDistance
				falloffFactor = math.Max(0, falloffFactor)    // Clamp to ensure non-negative
				falloffFactor = falloffFactor * falloffFactor // Apply squared falloff

				// Get the interaction type of the *nearest boundary site*.
				boundaryTypeForEffect, typeExists := planet.TectonicData.VertexBoundaryTypes[nearestBoundarySiteIndex]

				if typeExists {
					switch boundaryTypeForEffect {
					case Convergent:
						baseElevation += convergentBoundaryStrength * falloffFactor
					case Divergent:
						baseElevation -= divergentBoundaryStrength * falloffFactor
					case Passive:
						// Placeholder for more subtle effects at passive boundaries if desired.
						// e.g., small, sharp ridges or noisy terrain.
						// baseElevation += baseAmplitude * 0.1 * falloffFactor * (rand.New(rand.NewSource(elevationSeed + int64(siteID))).Float64()*2.0 - 1.0)
					}
				}
			}
		}
		planet.ElevationData.CellElevations[siteID] = baseElevation
	}
	fmt.Printf("  Generated elevations for %d cells, incorporating tectonic features.\n", len(planet.ElevationData.CellElevations))
	fmt.Println("-- Elevation Module Complete --")

	// --- 4. Other modules (Erosion, Climate, etc. - Placeholders for future integration) ---
	fmt.Println("\n-- Running Erosion Module (Placeholder) --")
	// Example call:
	// planet.ErosionData, err = erosion.SimulateErosion(planet, pipelineSettings.ErosionSettings)
	// if err != nil { return nil, fmt.Errorf("erosion module failed: %w", err) }
	fmt.Println("-- Erosion Module Complete --")

	fmt.Println("\n-- Running Climate Module (Placeholder) --")
	// Example call:
	// planet.ClimateData, err = climate.SimulateClimate(planet, pipelineSettings.ClimateSettings)
	// if err != nil { return nil, fmt.Errorf("climate module failed: %w", err) }
	fmt.Println("-- Climate Module Complete --")

	fmt.Println("\n--- Land Generation Pipeline Finished ---")
	return planet, nil
}
