// Package landgen will contain the logic for land generation.
package landgen

import (
	"fmt"
	"math"               // For math.Sin and math.Abs
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

	// IcosphereSubdivisions is kept for reference/logging if needed.
	IcosphereSubdivisions int `json:"icosphereSubdivisions"`
}

// ElevationData represents the output of land generation.
type ElevationData struct {
	CellElevations map[int32]float64 `json:"cellElevations"`
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
) (*ElevationData, error) {

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

	return generatedElevationData, nil
}
