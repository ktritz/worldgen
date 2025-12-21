package elevation

// elevation_map.go - 2D elevation map generation with bathymetric color scheme

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sync"
	"worldgen/icosphere"

	"github.com/kyroy/kdtree"
)

// ElevationMapSettings controls elevation map generation parameters
type ElevationMapSettings struct {
	Width         int     // Map width in pixels
	Height        int     // Map height in pixels
	Projection    string  // "mercator", "equirectangular"
	OutputPath    string  // Where to save the PNG file
	PlanetRadius  float64 // Planet radius for coordinate conversion
	MinElevation  float64 // Minimum elevation for color scaling
	MaxElevation  float64 // Maximum elevation for color scaling
}

// kdtreeElevationSitePoint wraps an icosphere site for KD-tree nearest neighbor searches
type kdtreeElevationSitePoint struct {
	Coordinates icosphere.Vector3D
	SiteIndex   int
	Elevation   float64
}

func (p kdtreeElevationSitePoint) Dimensions() int         { return 3 }
func (p kdtreeElevationSitePoint) Dimension(i int) float64 { 
	switch i {
	case 0: return p.Coordinates.X
	case 1: return p.Coordinates.Y
	case 2: return p.Coordinates.Z
	default: return 0
	}
}

// DefaultElevationMapSettings returns reasonable defaults for elevation map generation
func DefaultElevationMapSettings() ElevationMapSettings {
	return ElevationMapSettings{
		Width:        1024, // Reasonable resolution that matches tectonics maps
		Height:       512,  // Standard 2:1 aspect ratio
		Projection:   "equirectangular", // Simple projection for spherical data
		OutputPath:   "elevation_map.png",
		PlanetRadius: 6.371e6,
		MinElevation: -4000.0, // Typical ocean depth range
		MaxElevation: 8000.0,  // Typical mountain height range
	}
}

// GenerateElevationMap creates a 2D map showing elevation with bathymetric colors
func GenerateElevationMap(
	sites []icosphere.Vector3D,
	elevations []float64,
	settings ElevationMapSettings) error {

	if len(sites) != len(elevations) {
		return fmt.Errorf("sites and elevations arrays must have same length")
	}

	// Calculate actual elevation range for auto-scaling if not provided
	if settings.MinElevation == 0 && settings.MaxElevation == 0 {
		minElev, maxElev := findElevationRange(elevations)
		settings.MinElevation = minElev
		settings.MaxElevation = maxElev
	}

	// Build KD-tree for efficient nearest neighbor searches
	kdTree := buildElevationKDTree(sites, elevations)

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, settings.Width, settings.Height))

	// Use conservative parallel processing like tectonics maps
	numWorkers := 4 // Conservative worker count
	if numWorkers > settings.Height {
		numWorkers = settings.Height
	}
	if numWorkers <= 0 {
		numWorkers = 1
	}
	
	var wg sync.WaitGroup
	rowsPerWorker := (settings.Height + numWorkers - 1) / numWorkers
	
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			startRow := workerID * rowsPerWorker
			endRow := (workerID + 1) * rowsPerWorker
			if endRow > settings.Height {
				endRow = settings.Height
			}
			
			// Process rows assigned to this worker
			for y := startRow; y < endRow; y++ {
				for x := 0; x < settings.Width; x++ {
					// Convert pixel to spherical coordinates
					lon, lat := pixelToSpherical(x, y, settings.Width, settings.Height, settings.Projection)
					
					// Convert spherical to 3D coordinates
					point := sphericalToVector3D(lon, lat, settings.PlanetRadius)
					
					// Find nearest elevation sample
					elevation := findNearestElevation(kdTree, point)
					
					// Convert elevation to color
					pixelColor := elevationToColor(elevation, settings.MinElevation, settings.MaxElevation)
					img.Set(x, y, pixelColor)
				}
			}
		}(w)
	}
	wg.Wait()

	// Save image
	return saveElevationMap(img, settings.OutputPath)
}

// buildElevationKDTree creates a KD-tree for efficient elevation lookups
func buildElevationKDTree(sites []icosphere.Vector3D, elevations []float64) *kdtree.KDTree {
	points := make([]kdtree.Point, len(sites))
	for i, site := range sites {
		points[i] = kdtreeElevationSitePoint{
			Coordinates: site,
			SiteIndex:   i,
			Elevation:   elevations[i],
		}
	}
	return kdtree.New(points)
}

// findNearestElevation finds the elevation at the nearest site to a given point
func findNearestElevation(kdTree *kdtree.KDTree, point icosphere.Vector3D) float64 {
	queryPoint := kdtreeElevationSitePoint{Coordinates: point}
	nearest := kdTree.KNN(&queryPoint, 1)
	
	if len(nearest) > 0 {
		if elevPoint, ok := nearest[0].(kdtreeElevationSitePoint); ok {
			return elevPoint.Elevation
		}
	}
	
	return 0.0 // Default to sea level if no nearest point found
}

// elevationToColor converts an elevation value to a bathymetric color
func elevationToColor(elevation, minElev, maxElev float64) color.RGBA {
	// Clamp elevation to range
	if elevation < minElev {
		elevation = minElev
	}
	if elevation > maxElev {
		elevation = maxElev
	}
	
	if elevation < 0 {
		// Ocean depths - blue gradient
		return oceanDepthToColor(elevation, minElev)
	} else {
		// Land elevations - green to brown to white gradient
		return landElevationToColor(elevation, maxElev)
	}
}

// oceanDepthToColor converts ocean depth to blue color gradient
func oceanDepthToColor(depth, minDepth float64) color.RGBA {
	// Normalize depth to [0,1] where 0 = sea level, 1 = deepest
	t := -depth / -minDepth // Both negative, so this gives positive [0,1]
	if t > 1.0 {
		t = 1.0
	}
	
	// Define depth color stops based on the reference image
	if t < 0.0625 { // 0m to -250m: light blue/cyan to turquoise
		// Interpolate from light cyan to turquoise
		sub_t := t / 0.0625
		return color.RGBA{
			R: uint8(interpolate(180, 64, sub_t)),   // cyan to darker cyan
			G: uint8(interpolate(255, 224, sub_t)),  // bright cyan to aqua
			B: uint8(interpolate(255, 208, sub_t)),  // full blue to aqua blue
			A: 255,
		}
	} else if t < 0.25 { // -250m to -1000m: turquoise to medium blue
		sub_t := (t - 0.0625) / (0.25 - 0.0625)
		return color.RGBA{
			R: uint8(interpolate(64, 32, sub_t)),    // darker cyan to blue
			G: uint8(interpolate(224, 128, sub_t)),  // aqua to medium blue
			B: uint8(interpolate(208, 192, sub_t)),  // aqua blue to blue
			A: 255,
		}
	} else { // -1000m to -4000m: medium blue to dark blue
		sub_t := (t - 0.25) / (1.0 - 0.25)
		return color.RGBA{
			R: uint8(interpolate(32, 8, sub_t)),     // blue to dark blue
			G: uint8(interpolate(128, 64, sub_t)),   // medium blue to dark blue
			B: uint8(interpolate(192, 128, sub_t)),  // blue to darker blue
			A: 255,
		}
	}
}

// landElevationToColor converts land elevation to green-brown-white gradient
func landElevationToColor(elevation, maxElev float64) color.RGBA {
	// Normalize elevation to [0,1] where 0 = sea level, 1 = highest peak
	t := elevation / maxElev
	if t > 1.0 {
		t = 1.0
	}
	
	// Define elevation color stops based on the reference image
	if t < 0.03125 { // 0m to 250m: dark green (good contrast with shallow water)
		// Dark green with slight variation
		sub_t := t / 0.03125
		return color.RGBA{
			R: uint8(interpolate(34, 45, sub_t)),    // dark green
			G: uint8(interpolate(102, 120, sub_t)),  // forest green
			B: uint8(interpolate(51, 60, sub_t)),    // dark green
			A: 255,
		}
	} else if t < 0.125 { // 250m to 1000m: dark green to medium green
		sub_t := (t - 0.03125) / (0.125 - 0.03125)
		return color.RGBA{
			R: uint8(interpolate(45, 85, sub_t)),    // dark to medium green
			G: uint8(interpolate(120, 160, sub_t)),  // forest to medium green
			B: uint8(interpolate(60, 70, sub_t)),    // dark to medium green
			A: 255,
		}
	} else if t < 0.25 { // 1000m to 2000m: medium green to yellow-green
		sub_t := (t - 0.125) / (0.25 - 0.125)
		return color.RGBA{
			R: uint8(interpolate(85, 140, sub_t)),   // medium green to yellow-green
			G: uint8(interpolate(160, 180, sub_t)),  // medium green to light green
			B: uint8(interpolate(70, 85, sub_t)),    // medium green
			A: 255,
		}
	} else if t < 0.5 { // 2000m to 4000m: yellow-green to brown
		sub_t := (t - 0.25) / (0.5 - 0.25)
		return color.RGBA{
			R: uint8(interpolate(140, 160, sub_t)),  // yellow-green to brown
			G: uint8(interpolate(180, 140, sub_t)),  // light green to brown
			B: uint8(interpolate(85, 90, sub_t)),    // green to brown
			A: 255,
		}
	} else { // 4000m to 8000m: brown to tan/white
		sub_t := (t - 0.5) / (1.0 - 0.5)
		return color.RGBA{
			R: uint8(interpolate(160, 240, sub_t)),  // brown to tan/white
			G: uint8(interpolate(140, 220, sub_t)),  // brown to tan
			B: uint8(interpolate(90, 200, sub_t)),   // brown to light tan
			A: 255,
		}
	}
}

// interpolate linearly interpolates between two values
func interpolate(a, b, t float64) float64 {
	return a + (b-a)*t
}

// pixelToSpherical converts pixel coordinates to longitude/latitude
func pixelToSpherical(x, y, width, height int, projection string) (lon, lat float64) {
	// Normalize pixel coordinates to [0,1]
	u := float64(x) / float64(width)
	v := float64(y) / float64(height)
	
	switch projection {
	case "equirectangular":
		// Simple equirectangular projection
		lon = (u - 0.5) * 2 * math.Pi  // [-π, π]
		lat = (0.5 - v) * math.Pi      // [-π/2, π/2]
	case "mercator":
		// Mercator projection (more complex)
		lon = (u - 0.5) * 2 * math.Pi
		lat = math.Atan(math.Sinh(math.Pi * (0.5 - v)))
	default:
		// Default to equirectangular
		lon = (u - 0.5) * 2 * math.Pi
		lat = (0.5 - v) * math.Pi
	}
	
	return lon, lat
}

// sphericalToVector3D converts longitude/latitude to 3D coordinates
func sphericalToVector3D(lon, lat, radius float64) icosphere.Vector3D {
	cosLat := math.Cos(lat)
	return icosphere.Vector3D{
		X: radius * cosLat * math.Cos(lon),
		Y: radius * math.Sin(lat),
		Z: radius * cosLat * math.Sin(lon),
	}
}

// findElevationRange finds the minimum and maximum elevation values
func findElevationRange(elevations []float64) (min, max float64) {
	if len(elevations) == 0 {
		return 0, 0
	}
	
	min = elevations[0]
	max = elevations[0]
	
	for _, elev := range elevations {
		if elev < min {
			min = elev
		}
		if elev > max {
			max = elev
		}
	}
	
	return min, max
}

// saveElevationMap saves the elevation image to a PNG file
func saveElevationMap(img *image.RGBA, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	return png.Encode(file, img)
}

// GenerateElevationMapFromPipeline generates an elevation map from pipeline data
func GenerateElevationMapFromPipeline(elevationData *ElevationData, sites []icosphere.Vector3D, outputPath string) error {
	settings := DefaultElevationMapSettings()
	settings.OutputPath = outputPath
	
	// Use the final combined elevations
	elevations := elevationData.SiteElevations
	
	return GenerateElevationMap(sites, elevations, settings)
}