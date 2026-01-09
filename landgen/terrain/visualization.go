// Terrain visualization functions for rendering elevation maps
package terrain

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
	"sync"
)

// SpatialIndex provides efficient nearest-site lookups for rendering
type SpatialIndex struct {
	buckets map[int][]int
	gridRes int
}

// BuildSpatialIndex creates a spatial index from site positions
func BuildSpatialIndex(sites []Vector3D) *SpatialIndex {
	gridRes := 360
	idx := &SpatialIndex{
		buckets: make(map[int][]int),
		gridRes: gridRes,
	}
	for i, s := range sites {
		lat := math.Asin(s.Z) * 180 / math.Pi
		lon := math.Atan2(s.Y, s.X) * 180 / math.Pi
		bx := int((lon + 180) / 360 * float64(gridRes))
		by := int((lat + 90) / 180 * float64(gridRes))
		key := by*gridRes + bx
		idx.buckets[key] = append(idx.buckets[key], i)
	}
	return idx
}

// FindNearest returns the index of the nearest site to the given lat/lon
func (idx *SpatialIndex) FindNearest(lat, lon float64, sites []Vector3D) int {
	// Convert query point to 3D
	latRad := lat * math.Pi / 180
	lonRad := lon * math.Pi / 180
	qx := math.Cos(latRad) * math.Cos(lonRad)
	qy := math.Cos(latRad) * math.Sin(lonRad)
	qz := math.Sin(latRad)

	bx := int((lon + 180) / 360 * float64(idx.gridRes))
	by := int((lat + 90) / 180 * float64(idx.gridRes))

	// Near poles, search more buckets horizontally (longitudes converge)
	searchRadius := 1
	if math.Abs(lat) > 70 {
		searchRadius = 8
	} else if math.Abs(lat) > 50 {
		searchRadius = 3
	}

	bestDist := math.Inf(1)
	bestIdx := 0

	for dy := -searchRadius; dy <= searchRadius; dy++ {
		for dx := -searchRadius; dx <= searchRadius; dx++ {
			nbx := (bx + dx + idx.gridRes) % idx.gridRes
			nby := by + dy
			if nby < 0 || nby >= idx.gridRes {
				continue
			}
			key := nby*idx.gridRes + nbx
			for _, i := range idx.buckets[key] {
				s := sites[i]
				// 3D Euclidean distance (equivalent to spherical for nearest)
				d := (qx-s.X)*(qx-s.X) + (qy-s.Y)*(qy-s.Y) + (qz-s.Z)*(qz-s.Z)
				if d < bestDist {
					bestDist = d
					bestIdx = i
				}
			}
		}
	}
	return bestIdx
}

// HypsometricColor returns a natural-looking color for terrain elevation
// Based on professional cartographic color schemes
func HypsometricColor(elev float64) color.RGBA {
	if elev < 0 {
		// Ocean bathymetry: deep navy → medium blue → cyan shelf
		if elev < -6000 {
			// Abyssal: dark navy
			return color.RGBA{10, 30, 70, 255}
		} else if elev < -4000 {
			// Deep ocean
			t := (elev + 6000) / 2000
			return lerpColor(color.RGBA{10, 30, 70, 255}, color.RGBA{20, 50, 100, 255}, t)
		} else if elev < -2000 {
			// Mid ocean
			t := (elev + 4000) / 2000
			return lerpColor(color.RGBA{20, 50, 100, 255}, color.RGBA{40, 80, 140, 255}, t)
		} else if elev < -200 {
			// Continental slope
			t := (elev + 2000) / 1800
			return lerpColor(color.RGBA{40, 80, 140, 255}, color.RGBA{70, 130, 180, 255}, t)
		} else {
			// Continental shelf (shallow)
			t := (elev + 200) / 200
			return lerpColor(color.RGBA{70, 130, 180, 255}, color.RGBA{120, 180, 200, 255}, t)
		}
	}

	// Land hypsometry: green lowlands → tan → brown → gray → white peaks
	if elev < 100 {
		// Coastal lowlands: rich green
		t := elev / 100
		return lerpColor(color.RGBA{85, 140, 70, 255}, color.RGBA{95, 150, 75, 255}, t)
	} else if elev < 300 {
		// Low plains: green to yellow-green
		t := (elev - 100) / 200
		return lerpColor(color.RGBA{95, 150, 75, 255}, color.RGBA{140, 170, 90, 255}, t)
	} else if elev < 600 {
		// Rolling hills: yellow-green to tan
		t := (elev - 300) / 300
		return lerpColor(color.RGBA{140, 170, 90, 255}, color.RGBA{180, 170, 110, 255}, t)
	} else if elev < 1200 {
		// Foothills: tan to light brown
		t := (elev - 600) / 600
		return lerpColor(color.RGBA{180, 170, 110, 255}, color.RGBA{185, 155, 105, 255}, t)
	} else if elev < 2000 {
		// Mountains: light brown to brown
		t := (elev - 1200) / 800
		return lerpColor(color.RGBA{185, 155, 105, 255}, color.RGBA{160, 130, 100, 255}, t)
	} else if elev < 3000 {
		// High mountains: brown to gray-brown
		t := (elev - 2000) / 1000
		return lerpColor(color.RGBA{160, 130, 100, 255}, color.RGBA{140, 130, 125, 255}, t)
	} else if elev < 4500 {
		// Alpine: gray-brown to light gray
		t := (elev - 3000) / 1500
		return lerpColor(color.RGBA{140, 130, 125, 255}, color.RGBA{190, 190, 190, 255}, t)
	} else {
		// Permanent snow: light gray to white
		t := math.Min((elev-4500)/2000, 1.0)
		return lerpColor(color.RGBA{190, 190, 190, 255}, color.RGBA{250, 250, 255, 255}, t)
	}
}

// lerpColor linearly interpolates between two colors
func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.RGBA{
		uint8(float64(a.R) + t*(float64(b.R)-float64(a.R))),
		uint8(float64(a.G) + t*(float64(b.G)-float64(a.G))),
		uint8(float64(a.B) + t*(float64(b.B)-float64(a.B))),
		255,
	}
}

// ElevationColor returns a color for the given elevation value (legacy, uses HypsometricColor)
func ElevationColor(elev float64) color.RGBA {
	return HypsometricColor(elev)
}

// RenderElevationMap renders an equirectangular elevation map to a file
func RenderElevationMap(sites []Vector3D, elevation []float64, index *SpatialIndex, filename string) {
	width, height := 1024, 512
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	var wg sync.WaitGroup
	for py := 0; py < height; py++ {
		wg.Add(1)
		go func(py int) {
			defer wg.Done()
			lat := 90 - float64(py)/float64(height)*180
			for px := 0; px < width; px++ {
				lon := float64(px)/float64(width)*360 - 180
				idx := index.FindNearest(lat, lon, sites)
				elev := elevation[idx]
				img.Set(px, py, ElevationColor(elev))
			}
		}(py)
	}
	wg.Wait()

	f, _ := os.Create(filename)
	png.Encode(f, img)
	f.Close()
	fmt.Printf("  Saved %s\n", filename)
}

// RenderLandOceanMap renders a land/ocean classification map to a file
func RenderLandOceanMap(sites []Vector3D, elevation []float64, isLand []bool, index *SpatialIndex, filename string) {
	width, height := 1024, 512
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	var wg sync.WaitGroup
	for py := 0; py < height; py++ {
		wg.Add(1)
		go func(py int) {
			defer wg.Done()
			lat := 90 - float64(py)/float64(height)*180
			for px := 0; px < width; px++ {
				lon := float64(px)/float64(width)*360 - 180
				idx := index.FindNearest(lat, lon, sites)
				elev := elevation[idx]

				var c color.RGBA
				if isLand[idx] {
					t := math.Min(elev/4000, 1.0)
					c = color.RGBA{
						uint8(80 + 100*t),
						uint8(160 - 80*t),
						uint8(60 - 30*t),
						255,
					}
				} else {
					depth := math.Min(-elev/8000, 1.0)
					c = color.RGBA{
						0,
						uint8(100 + 100*(1-depth)),
						uint8(180 + 50*(1-depth)),
						255,
					}
				}
				img.Set(px, py, c)
			}
		}(py)
	}
	wg.Wait()

	f, _ := os.Create(filename)
	png.Encode(f, img)
	f.Close()
	fmt.Printf("  Saved %s\n", filename)
}

// RenderOrthoView renders an orthographic projection (globe view) centered on centerLat, centerLon
func RenderOrthoView(sites []Vector3D, elevation []float64, index *SpatialIndex, centerLat, centerLon float64, filename string) {
	size := 512
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Background color (space)
	bgColor := color.RGBA{10, 10, 30, 255}
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			img.Set(px, py, bgColor)
		}
	}

	// Convert center to radians
	centerLatRad := centerLat * math.Pi / 180
	centerLonRad := centerLon * math.Pi / 180

	// Precompute trig values
	cosLat := math.Cos(-centerLatRad)
	sinLat := math.Sin(-centerLatRad)
	cosLon := math.Cos(centerLonRad)
	sinLon := math.Sin(centerLonRad)

	var wg sync.WaitGroup
	for py := 0; py < size; py++ {
		wg.Add(1)
		go func(py int) {
			defer wg.Done()
			for px := 0; px < size; px++ {
				x := (float64(px) - float64(size)/2) / (float64(size) / 2)
				y := (float64(size)/2 - float64(py)) / (float64(size) / 2)

				r2 := x*x + y*y
				if r2 > 1 {
					continue
				}

				z := math.Sqrt(1 - r2)

				y1 := y*cosLat - z*sinLat
				z1 := y*sinLat + z*cosLat

				x2 := x*cosLon + z1*sinLon
				z2 := -x*sinLon + z1*cosLon

				lat := math.Asin(y1) * 180 / math.Pi
				lon := math.Atan2(x2, z2) * 180 / math.Pi

				idx := index.FindNearest(lat, lon, sites)
				elev := elevation[idx]
				img.Set(px, py, ElevationColor(elev))
			}
		}(py)
	}
	wg.Wait()

	f, _ := os.Create(filename)
	png.Encode(f, img)
	f.Close()
	fmt.Printf("  Saved %s\n", filename)
}

// RenderHypsometry generates an elevation histogram
func RenderHypsometry(elevation []float64, filename string) {
	bins := 60
	minE, maxE := -10000.0, 9000.0
	hist := make([]int, bins)

	for _, e := range elevation {
		bin := int((e - minE) / (maxE - minE) * float64(bins))
		if bin < 0 {
			bin = 0
		}
		if bin >= bins {
			bin = bins - 1
		}
		hist[bin]++
	}

	maxCount := 0
	for _, c := range hist {
		if c > maxCount {
			maxCount = c
		}
	}

	width, height := 600, 300
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// White background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Draw bars
	barWidth := width / bins
	for i, count := range hist {
		barHeight := int(float64(count) / float64(maxCount) * float64(height-30))
		x0 := i * barWidth

		elev := minE + (float64(i)+0.5)/float64(bins)*(maxE-minE)
		c := ElevationColor(elev)

		for x := x0; x < x0+barWidth-1; x++ {
			for y := height - 20 - barHeight; y < height-20; y++ {
				img.Set(x, y, c)
			}
		}
	}

	// Sea level line
	seaX := int((-minE) / (maxE - minE) * float64(width))
	for y := 0; y < height-20; y++ {
		img.Set(seaX, y, color.RGBA{255, 0, 0, 255})
	}

	f, _ := os.Create(filename)
	png.Encode(f, img)
	f.Close()
	fmt.Printf("  Saved %s\n", filename)
}

// RenderShadedElevationMap renders an equirectangular elevation map with hillshading
func RenderShadedElevationMap(sites []Vector3D, elevation []float64, index *SpatialIndex, filename string, width, height int) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Precompute elevation grid for hillshading
	elevGrid := make([][]float64, height)
	for py := 0; py < height; py++ {
		elevGrid[py] = make([]float64, width)
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			idx := index.FindNearest(lat, lon, sites)
			elevGrid[py][px] = elevation[idx]
		}
	}

	// Calculate pixel size in meters (at equator) for proper gradient scaling
	earthCircumference := 40075000.0 // meters
	pixelSizeX := earthCircumference / float64(width)
	pixelSizeY := earthCircumference / 2 / float64(height)

	// Render with multi-directional hillshading
	var wg sync.WaitGroup
	for py := 0; py < height; py++ {
		wg.Add(1)
		go func(py int) {
			defer wg.Done()

			// Latitude for this row (affects E-W pixel size)
			lat := 90 - float64(py)/float64(height)*180
			latRad := lat * math.Pi / 180
			cosLat := math.Cos(latRad)
			if cosLat < 0.1 {
				cosLat = 0.1 // Avoid division issues near poles
			}
			localPixelSizeX := pixelSizeX * cosLat

			for px := 0; px < width; px++ {
				elev := elevGrid[py][px]

				// Compute hillshade using proper gradient calculation
				shade := 0.5 // neutral ambient

				if py > 0 && py < height-1 {
					// Handle wraparound for longitude
					pxLeft := (px - 1 + width) % width
					pxRight := (px + 1) % width

					// Gradient in meters rise per meter run
					dzdx := (elevGrid[py][pxRight] - elevGrid[py][pxLeft]) / (2 * localPixelSizeX)
					dzdy := (elevGrid[py-1][px] - elevGrid[py+1][px]) / (2 * pixelSizeY)

					// Exaggerate vertical scale for visibility (z-factor)
					zFactor := 3.0
					dzdx *= zFactor
					dzdy *= zFactor

					// Multi-directional hillshading (Swiss-style)
					// Primary light from NW (315°), secondary from N and W
					shade = computeMultiHillshade(dzdx, dzdy)
				}

				// Get base color and apply shading
				c := HypsometricColor(elev)

				// Apply shading with better contrast
				// shade ranges ~0.2 to ~1.0, we want visible but not harsh
				r := float64(c.R) * (0.4 + 0.6*shade)
				g := float64(c.G) * (0.4 + 0.6*shade)
				b := float64(c.B) * (0.4 + 0.6*shade)

				// Slight blue tint in shadows (atmospheric perspective)
				if shade < 0.5 {
					shadowT := (0.5 - shade) * 0.15
					b = b + shadowT*40
				}

				img.Set(px, py, color.RGBA{
					uint8(math.Min(255, math.Max(0, r))),
					uint8(math.Min(255, math.Max(0, g))),
					uint8(math.Min(255, math.Max(0, b))),
					255,
				})
			}
		}(py)
	}
	wg.Wait()

	f, _ := os.Create(filename)
	png.Encode(f, img)
	f.Close()
	fmt.Printf("  Saved %s\n", filename)
}

// computeMultiHillshade computes hillshading from multiple light directions
// for a Swiss-style relief look. Returns value 0-1.
func computeMultiHillshade(dzdx, dzdy float64) float64 {
	// Compute slope and aspect
	slope := math.Sqrt(dzdx*dzdx + dzdy*dzdy)
	slopeAngle := math.Atan(slope)
	aspect := math.Atan2(dzdy, -dzdx)

	// Light sources (azimuth in radians from north, altitude in radians)
	type light struct {
		az, alt, weight float64
	}
	lights := []light{
		{315 * math.Pi / 180, 45 * math.Pi / 180, 0.6},  // Primary: NW, 45° up
		{270 * math.Pi / 180, 60 * math.Pi / 180, 0.25}, // Secondary: W, higher
		{0 * math.Pi / 180, 70 * math.Pi / 180, 0.15},   // Fill: N, nearly overhead
	}

	totalShade := 0.0
	for _, l := range lights {
		zenith := math.Pi/2 - l.alt
		shade := math.Cos(zenith)*math.Cos(slopeAngle) +
			math.Sin(zenith)*math.Sin(slopeAngle)*math.Cos(l.az-aspect)
		if shade < 0 {
			shade = 0
		}
		totalShade += shade * l.weight
	}

	// Normalize and add ambient
	ambient := 0.15
	return math.Min(1.0, ambient+totalShade*0.85)
}

// PrintStats prints elevation statistics to stdout
func PrintStats(elevation []float64, isLand []bool) {
	sorted := make([]float64, len(elevation))
	copy(sorted, elevation)
	sort.Float64s(sorted)

	landCount := 0
	for _, l := range isLand {
		if l {
			landCount++
		}
	}

	fmt.Printf("\n  Statistics:\n")
	fmt.Printf("    Land coverage: %.1f%%\n", 100*float64(landCount)/float64(len(isLand)))
	fmt.Printf("    Elevation range: %.0fm to %.0fm\n", sorted[0], sorted[len(sorted)-1])
	fmt.Printf("    Ocean floor (25%%): %.0fm\n", sorted[len(sorted)/4])
	fmt.Printf("    Sea level (at %.0f%%): 0m\n", 100*float64(len(sorted)-landCount)/float64(len(sorted)))
	fmt.Printf("    Median land: %.0fm\n", sorted[(len(sorted)+landCount)/2])
}
