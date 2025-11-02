package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"time"
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
)

func main() {
	fmt.Println("=== LEVEL 7 PLATE VISUALIZATION ===")
	start := time.Now()

	planetRadius := 6371000.0
	subdivision := 7 // Full resolution!

	// Load data
	vertices, faces := icosphere.CreateIcosphere(subdivision)
	fmt.Printf("Vertices: %d\n", len(vertices))

	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)

	settings := tectonics.DefaultQuotaSettings()
	settings.Verbose = false
	plates, cellAssignments, _ := tectonics.QuotaBasedGeneration(
		voronoiCells, vertices, planetRadius, settings)
	fmt.Printf("Plates: %d\n", len(plates))

	// Colors
	plateColors := make([]color.RGBA, len(plates))
	for i := range plates {
		h := math.Mod(float64(i)*0.618033988749895, 1.0)
		r, g, b := hsvToRGB(h, 0.7, 0.8)
		plateColors[i] = color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
	}

	// Render using spatial coherence
	width, height := 3600, 1800
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	pix := img.Pix

	fmt.Println("Rendering with spatial coherence...")
	renderStart := time.Now()

	// Helper function to find nearest using coherence
	findNearest := func(px, py, pz float64, hint int) int {
		// Check hint and its neighbors first (spatial coherence!)
		candidates := []int{hint}
		if hint < len(voronoiCells) {
			for _, neighIdx := range voronoiCells[hint].NeighborSiteIndices {
				if int(neighIdx) < len(vertices) {
					candidates = append(candidates, int(neighIdx))
				}
			}
		}

		nearestIdx := hint
		minDist := math.MaxFloat64

		// Check candidates
		for _, idx := range candidates {
			if idx >= len(vertices) {
				continue
			}
			site := vertices[idx]
			dx := site.X - px
			dy := site.Y - py
			dz := site.Z - pz
			dist := dx*dx + dy*dy + dz*dz

			if dist < minDist {
				minDist = dist
				nearestIdx = idx
			}
		}

		return nearestIdx
	}

	// Start with a good hint (first site)
	rowStartHint := 0

	for y := 0; y < height; y++ {
		lat := (0.5 - float64(y)/float64(height)) * math.Pi
		cosLat := math.Cos(lat)
		sinLat := math.Sin(lat)

		// Use hint from previous row's start to avoid left-edge artifacts
		currentHint := rowStartHint

		for x := 0; x < width; x++ {
			lon := (float64(x)/float64(width) - 0.5) * 2.0 * math.Pi

			px := planetRadius * cosLat * math.Cos(lon)
			py := planetRadius * cosLat * math.Sin(lon)
			pz := planetRadius * sinLat

			// Use spatial coherence - adjacent pixels likely hit same/neighbor sites
			nearestIdx := findNearest(px, py, pz, currentHint)
			currentHint = nearestIdx // Use this as hint for next pixel

			// Save first pixel of row as hint for next row
			if x == 0 {
				rowStartHint = nearestIdx
			}

			// Write pixel
			plateIdx := cellAssignments[nearestIdx]
			offset := (y*width + x) * 4
			if plateIdx >= 0 && plateIdx < len(plateColors) {
				c := plateColors[plateIdx]
				pix[offset] = c.R
				pix[offset+1] = c.G
				pix[offset+2] = c.B
				pix[offset+3] = c.A
			}
		}

		if y%100 == 0 {
			elapsed := time.Since(renderStart).Seconds()
			pct := 100.0 * float64(y) / float64(height)
			fmt.Printf("%.1f%% (%.1fs)\r", pct, elapsed)
		}
	}

	fmt.Printf("100%% in %.1fs\n", time.Since(renderStart).Seconds())

	// Save
	f, _ := os.Create("tectonic_plates_level7.png")
	defer f.Close()
	png.Encode(f, img)

	fmt.Printf("\n✓ Done in %.1fs\n", time.Since(start).Seconds())
	fmt.Printf("Saved: tectonic_plates_level7.png (%dx%d)\n", width, height)
}

func hsvToRGB(h, s, v float64) (r, g, b float64) {
	if s == 0 {
		return v, v, v
	}
	h = h * 6.0
	i := int(h)
	f := h - float64(i)
	p := v * (1.0 - s)
	q := v * (1.0 - s*f)
	t := v * (1.0 - s*(1.0-f))
	switch i % 6 {
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
