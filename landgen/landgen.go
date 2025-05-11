// Package landgen will contain the logic for land generation.
package landgen

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math" // For placeholder elevation
	"os"
	"worldgen/icosphere" // Import to use Vector3D and VoronoiCell
)

// LandGenerationParams holds parameters for land generation.
type LandGenerationParams struct {
	Seed                int     `json:"landSeed"`
	NoiseScale          float64 `json:"noiseScale"`
	NoiseOctaves        int     `json:"noiseOctaves"`
	NoisePersistence    float64 `json:"noisePersistence"`
	NoiseLacunarity     float64 `json:"noiseLacunarity"`
	ElevationMultiplier float64 `json:"elevationMultiplier"`
	OutputName          string  `json:"landOutputName"` // Filename for auxiliary output

	// Icosphere site vertices (original generator points for Voronoi)
	// Passed as flattened array from main.go
	IcoSiteVertices []float32 `json:"-"` // Exclude from direct JSON if it's always derived/passed internally

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
func GenerateLandData(
	params LandGenerationParams,
	// icoSites []icosphere.Vector3D, // REMOVED - Now part of params
	voroVertices []icosphere.Vector3D,
	voroCells []icosphere.VoronoiCell,
	outputPath string,
) (*ElevationData, string, error) {

	// Unflatten icosphere sites from params
	icoSites := unflattenToVector3DLandgen(params.IcoSiteVertices)
	if len(params.IcoSiteVertices) > 0 && len(icoSites) == 0 {
		return nil, "", fmt.Errorf("failed to unflatten icosphere site vertices from params")
	}

	fmt.Printf("Land Generation: Received %d Voronoi cells, %d Voronoi vertices. Using %d Icosphere sites from params.\n", len(voroCells), len(voroVertices), len(icoSites))
	fmt.Printf("Base Icosphere Subdivisions (ref): %d\n", params.IcosphereSubdivisions)
	fmt.Printf("Noise Params: Seed=%d, Scale=%.2f, Octaves=%d, Persistence=%.2f, Lacunarity=%.2f, ElevMult=%.2f\n",
		params.Seed, params.NoiseScale, params.NoiseOctaves, params.NoisePersistence, params.NoiseLacunarity, params.ElevationMultiplier)

	generatedElevationData := &ElevationData{
		CellElevations: make(map[int32]float64),
	}

	for _, cell := range voroCells {
		if int(cell.SiteIndex) < len(icoSites) {
			// siteCoordinate := icoSites[cell.SiteIndex] // Use this for noise sampling
			elevation := math.Sin(float64(cell.SiteIndex)*params.NoiseScale*0.1+float64(params.Seed)*0.01) * params.ElevationMultiplier
			generatedElevationData.CellElevations[cell.SiteIndex] = elevation
		} else {
			fmt.Printf("Warning (landgen): Voronoi cell has SiteIndex %d out of bounds for icoSites (len %d)\n", cell.SiteIndex, len(icoSites))
		}
	}
	fmt.Printf("Generated elevations for %d cells.\n", len(generatedElevationData.CellElevations))

	var imagePathSaved string
	if params.OutputName != "" && (len(params.OutputName) > 4 && params.OutputName[len(params.OutputName)-4:] == ".png") {
		imgWidth, imgHeight := 256, 256
		img := image.NewGray(image.Rect(0, 0, imgWidth, imgHeight))
		for y := 0; y < imgHeight; y++ {
			for x := 0; x < imgWidth; x++ {
				val := uint8((float64(x%50) / 50.0) * params.NoiseScale * 10 * math.Abs(params.ElevationMultiplier*100))
				img.SetGray(x, y, color.Gray{Y: val})
			}
		}
		file, err := os.Create(outputPath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create placeholder heightmap file %s: %w", outputPath, err)
		}
		if err := png.Encode(file, img); err != nil {
			file.Close()
			return nil, "", fmt.Errorf("failed to encode placeholder heightmap to PNG %s: %w", outputPath, err)
		}
		file.Close()
		imagePathSaved = outputPath
		fmt.Printf("Placeholder heightmap image generated and saved to: %s\n", imagePathSaved)
	} else {
		fmt.Println("No PNG output name specified, or filename is not a .png; skipping dummy image generation.")
	}

	return generatedElevationData, imagePathSaved, nil
}
