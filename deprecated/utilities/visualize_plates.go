package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
)

func main() {
	fmt.Println("=== GENERATING TECTONIC PLATE VISUALIZATION ===")

	planetRadius := 6371000.0
	subdivision := 5

	vertices, faces := icosphere.CreateIcosphere(subdivision)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
	icosphereSites := vertices

	// Generate plates using quota-based method
	settings := tectonics.DefaultQuotaSettings()
	settings.Verbose = false
	plates, cellAssignments, _ := tectonics.QuotaBasedGeneration(
		voronoiCells, icosphereSites, planetRadius, settings)

	fmt.Printf("Generated %d plates\n", len(plates))

	// Create equirectangular projection map
	width := 3600  // 0.1 degree resolution
	height := 1800
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Generate colors for each plate
	plateColors := make([]color.RGBA, len(plates))
	for i := range plates {
		plateColors[i] = generatePlateColor(i, len(plates))
	}

	fmt.Println("Rendering map...")

	// For each pixel, determine which plate it belongs to
	for y := 0; y < height; y++ {
		lat := 90.0 - (float64(y)/float64(height))*180.0 // -90 to +90
		latRad := lat * math.Pi / 180.0

		for x := 0; x < width; x++ {
			lon := (float64(x)/float64(width))*360.0 - 180.0 // -180 to +180
			lonRad := lon * math.Pi / 180.0

			// Convert lat/lon to 3D coordinates
			px := planetRadius * math.Cos(latRad) * math.Cos(lonRad)
			py := planetRadius * math.Cos(latRad) * math.Sin(lonRad)
			pz := planetRadius * math.Sin(latRad)

			// Find nearest icosphere site
			nearestSite := 0
			minDist := math.MaxFloat64

			for i, site := range icosphereSites {
				dx := site.X - px
				dy := site.Y - py
				dz := site.Z - pz
				dist := dx*dx + dy*dy + dz*dz

				if dist < minDist {
					minDist = dist
					nearestSite = i
				}
			}

			// Get plate assignment for this site
			plateIdx := cellAssignments[nearestSite]
			if plateIdx >= 0 && plateIdx < len(plateColors) {
				img.Set(x, y, plateColors[plateIdx])
			} else {
				img.Set(x, y, color.RGBA{0, 0, 0, 255}) // Black for unassigned
			}
		}

		if y%100 == 0 {
			fmt.Printf("Progress: %.1f%%\r", 100.0*float64(y)/float64(height))
		}
	}

	fmt.Println("Progress: 100.0%")

	// Save to file
	filename := "tectonic_plates_map.png"
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		fmt.Printf("Error encoding PNG: %v\n", err)
		return
	}

	fmt.Printf("\n✓ Saved visualization to %s\n", filename)
	fmt.Printf("  Image size: %dx%d pixels\n", width, height)
	fmt.Printf("  Plates: %d total\n", len(plates))
}

// generatePlateColor creates a distinct color for each plate
func generatePlateColor(plateIdx, totalPlates int) color.RGBA {
	// Use golden ratio for well-distributed hues
	golden := 0.618033988749895
	hue := math.Mod(float64(plateIdx)*golden, 1.0)

	// Vary saturation and value for more distinction
	saturation := 0.6 + 0.4*math.Mod(float64(plateIdx)*0.37, 1.0)
	value := 0.7 + 0.3*math.Mod(float64(plateIdx)*0.41, 1.0)

	r, g, b := hsvToRGB(hue, saturation, value)

	return color.RGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: 255,
	}
}

// hsvToRGB converts HSV to RGB
func hsvToRGB(h, s, v float64) (r, g, b float64) {
	if s == 0 {
		return v, v, v
	}

	h = h * 6.0
	i := math.Floor(h)
	f := h - i
	p := v * (1.0 - s)
	q := v * (1.0 - s*f)
	t := v * (1.0 - s*(1.0-f))

	switch int(i) % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}
