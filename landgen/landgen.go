// Package landgen will contain the logic for land generation.
package landgen

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math" // For math.Sin and math.Abs
	"os"
	"worldgen/icosphere" // Import to use Vector3D and VoronoiCell
)

// LandGenerationParams holds parameters for land generation.
type LandGenerationParams struct {
	Seed                int     `json:"landSeed"`
	NoiseScale          float64 `json:"noiseScale"` // Will be used as a frequency factor for the sinusoidal pattern
	NoiseOctaves        int     `json:"noiseOctaves"`
	NoisePersistence    float64 `json:"noisePersistence"`
	NoiseLacunarity     float64 `json:"noiseLacunarity"`
	ElevationMultiplier float64 `json:"elevationMultiplier"` // Amplitude of the sinusoidal pattern
	OutputName          string  `json:"landOutputName"`      // Filename for auxiliary output

	// Icosphere site vertices (original generator points for Voronoi)
	// Passed as flattened array from main.go
	// IcoSiteVertices []float32 `json:"-"` // Exclude from direct JSON if it's always derived/passed internally

	// IcosphereSubdivisions is kept for reference/logging if needed.
	IcosphereSubdivisions int `json:"icosphereSubdivisions"`
}

// ElevationData represents the output of land generation.
type ElevationData struct {
	CellElevations map[int32]float64 `json:"cellElevations"`
}

// Helper to unflatten []float32 to []icosphere.Vector3D (local to this package if needed)
func unflattenToVector3DLandgen(flatVertices []float32) []icosphere.Vector3D {
	if len(flatVertices)%3 != 0 {
		log.Printf("Warning (landgen): unflattenToVector3D received flatVertices length not divisible by 3: %d", len(flatVertices))
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

// GenerateLandData now takes fewer direct arguments for mesh data,
// relying on params for IcoSiteVertices.
// The elevation will be a sinusoidal function of the X-coordinate of the icosphere sites.
func GenerateLandData(
	params LandGenerationParams,
	icoSites []icosphere.Vector3D, // REMOVED - Now part of params
	voroVertices []icosphere.Vector3D, // These are the vertices of the Voronoi cells themselves
	voroCells []icosphere.VoronoiCell, // These link Voronoi cells to original icosphere sites
	outputPath string,
) (*ElevationData, string, error) {

	// Unflatten icosphere sites from params. These are the generator points for the Voronoi cells.
	// icoSites := unflattenToVector3DLandgen(params.IcoSiteVertices)
	// if len(params.IcoSiteVertices) > 0 && len(icoSites) == 0 {
	// 	return nil, "", fmt.Errorf("failed to unflatten icosphere site vertices from params")
	//}

	fmt.Printf("Land Generation (Sinusoidal Test): Received %d Voronoi cells, %d Voronoi vertices. Using %d Icosphere sites from params.\n", len(voroCells), len(voroVertices), len(icoSites))
	fmt.Printf("Base Icosphere Subdivisions (ref): %d\n", params.IcosphereSubdivisions)
	// Using NoiseScale as frequency factor and ElevationMultiplier as amplitude for the sine wave.
	fmt.Printf("Sine Wave Params: Axis=X, FrequencyFactor(NoiseScale)=%.2f, Amplitude(ElevationMultiplier)=%.2f, Seed(PhaseShift)=%d\n",
		params.NoiseScale, params.ElevationMultiplier, params.Seed)

	generatedElevationData := &ElevationData{
		CellElevations: make(map[int32]float64),
	}

	// The icosphere sites are normalized to be on a unit sphere, so their coordinates are typically in [-1, 1].
	// A frequency factor helps to see multiple periods of the sine wave across the sphere.
	// A small factor for seed ensures it acts as a phase shift.
	frequencyFactor := params.NoiseScale * 5.0 // Adjusted for more visible bands on a sphere
	phaseShift := float64(params.Seed) * 0.01

	for _, cell := range voroCells {
		if int(cell.SiteIndex) < len(icoSites) {
			siteCoordinate := icoSites[cell.SiteIndex] // Get the coordinate of the icosphere site

			// Calculate elevation based on the X-coordinate of the icosphere site
			// math.Sin(value) produces results in [-1, 1].
			// We multiply by ElevationMultiplier to control the amplitude.
			elevation := math.Sin(siteCoordinate.X*frequencyFactor+phaseShift) * params.ElevationMultiplier
			generatedElevationData.CellElevations[cell.SiteIndex] = elevation
		} else {
			fmt.Printf("Warning (landgen): Voronoi cell has SiteIndex %d out of bounds for icoSites (len %d)\n", cell.SiteIndex, len(icoSites))
			// Assign a default elevation (e.g., 0 or min value) for out-of-bounds indices if necessary,
			// though this case should ideally not happen with correct input data.
			generatedElevationData.CellElevations[cell.SiteIndex] = 0
		}
	}
	fmt.Printf("Generated sinusoidal elevations for %d cells.\n", len(generatedElevationData.CellElevations))

	var imagePathSaved string
	// The placeholder image generation logic can remain as is, or be removed if not needed for this specific test.
	// For this test, the primary output is the ElevationData.
	if params.OutputName != "" && (len(params.OutputName) > 4 && params.OutputName[len(params.OutputName)-4:] == ".png") {
		imgWidth, imgHeight := 256, 256
		img := image.NewGray(image.Rect(0, 0, imgWidth, imgHeight))
		// Create a simple gradient or pattern for the placeholder image,
		// as it's not directly representing the sinusoidal elevation here.
		for y := 0; y < imgHeight; y++ {
			for x := 0; x < imgWidth; x++ {
				// Simple gradient based on X for the placeholder image
				val := uint8((float64(x) / float64(imgWidth)) * 255)
				img.SetGray(x, y, color.Gray{Y: val})
			}
		}
		file, err := os.Create(outputPath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create placeholder heightmap file %s: %w", outputPath, err)
		}
		defer file.Close() // Ensure file is closed
		if err := png.Encode(file, img); err != nil {
			// file.Close() // Already handled by defer
			return nil, "", fmt.Errorf("failed to encode placeholder heightmap to PNG %s: %w", outputPath, err)
		}
		// file.Close() // Already handled by defer
		imagePathSaved = outputPath
		fmt.Printf("Placeholder diagnostic image generated and saved to: %s\n", imagePathSaved)
	} else {
		fmt.Println("No PNG output name specified, or filename is not a .png; skipping dummy image generation.")
	}

	return generatedElevationData, imagePathSaved, nil
}
