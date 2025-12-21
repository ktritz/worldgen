// Planet generation tool using Red Blob Games algorithm
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"worldgen/icosphere"
	"worldgen/landgen/terrain"
)

const outputDir = "output/maps"

func main() {
	fmt.Println("=== Planet Generation Tool ===")

	// Ensure output directory exists
	os.MkdirAll(outputDir, 0755)

	// Generate base mesh once
	level := 7
	fmt.Printf("Generating icosphere level %d...\n", level)
	vertices, faces := icosphere.CreateIcosphere(level)
	fmt.Printf("  %d vertices, %d faces\n", len(vertices), len(faces))

	fmt.Println("Generating Voronoi cells...")
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)

	// Convert to terrain types
	sites := make([]terrain.Vector3D, len(vertices))
	for i, v := range vertices {
		sites[i] = terrain.Vector3D{X: v.X, Y: v.Y, Z: v.Z}
	}
	cells := make([]terrain.VoronoiCell, len(voronoiCells))
	for i, cell := range voronoiCells {
		cells[i] = terrain.VoronoiCell{
			SiteIndex:           int32(cell.SiteIndex),
			NeighborSiteIndices: make([]int32, len(cell.NeighborSiteIndices)),
		}
		for j, idx := range cell.NeighborSiteIndices {
			cells[i].NeighborSiteIndices[j] = int32(idx)
		}
	}

	// Build spatial index once
	fmt.Println("Building spatial index...")
	index := buildSpatialIndex(sites)

	// Generate planets with different seeds for variety
	// Seeds selected to show different continent configurations:
	// - Seed 42: typical single mega-continent
	// - Seed 84: high convergent rate (~40%)
	// - Seed 8: two continents
	numPlates := 12
	testSeeds := []int64{42, 84, 8}
	landFrac := 0.29

	for _, seed := range testSeeds {
		fmt.Printf("\n========================================\n")
		fmt.Printf("Generating planet with seed %d, %.0f%% land, %d plates\n", seed, landFrac*100, numPlates)
		fmt.Printf("========================================\n")

		elevation, isLand := terrain.GeneratePlanetElevation(sites, cells, numPlates, seed, landFrac)

		prefix := fmt.Sprintf("planet_seed%d", seed)

		// Render elevation map
		renderElevationMap(sites, elevation, index, filepath.Join(outputDir, prefix+"_elevation.png"))

		// Render land/ocean map
		renderLandOceanMap(sites, elevation, isLand, index, filepath.Join(outputDir, prefix+"_landocean.png"))

		// Render orthographic (globe) views from multiple angles - no polar distortion
		renderOrthoView(sites, elevation, index, 0, 0, filepath.Join(outputDir, prefix+"_globe_front.png"))
		renderOrthoView(sites, elevation, index, 0, 90, filepath.Join(outputDir, prefix+"_globe_side.png"))
		renderOrthoView(sites, elevation, index, -45, 0, filepath.Join(outputDir, prefix+"_globe_south.png"))

		// Generate hypsometry histogram
		renderHypsometry(elevation, filepath.Join(outputDir, prefix+"_hypsometry.png"))

		// Print stats
		printStats(elevation, isLand)
	}

	fmt.Println("\n=== Generation complete ===")
	fmt.Printf("Output saved to %s/\n", outputDir)
}

type spatialIndex struct {
	buckets map[int][]int
	gridRes int
}

func buildSpatialIndex(sites []terrain.Vector3D) *spatialIndex {
	gridRes := 360
	idx := &spatialIndex{
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

func (idx *spatialIndex) findNearest(lat, lon float64, sites []terrain.Vector3D) int {
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

func renderElevationMap(sites []terrain.Vector3D, elevation []float64, index *spatialIndex, filename string) {
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
				idx := index.findNearest(lat, lon, sites)
				elev := elevation[idx]
				img.Set(px, py, elevationColor(elev))
			}
		}(py)
	}
	wg.Wait()

	f, _ := os.Create(filename)
	png.Encode(f, img)
	f.Close()
	fmt.Printf("  Saved %s\n", filename)
}

func renderLandOceanMap(sites []terrain.Vector3D, elevation []float64, isLand []bool, index *spatialIndex, filename string) {
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
				idx := index.findNearest(lat, lon, sites)
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

// renderOrthoView renders an orthographic projection (globe view) centered on centerLat, centerLon
func renderOrthoView(sites []terrain.Vector3D, elevation []float64, index *spatialIndex, centerLat, centerLon float64, filename string) {
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

				idx := index.findNearest(lat, lon, sites)
				elev := elevation[idx]
				img.Set(px, py, elevationColor(elev))
			}
		}(py)
	}
	wg.Wait()

	f, _ := os.Create(filename)
	png.Encode(f, img)
	f.Close()
	fmt.Printf("  Saved %s\n", filename)
}

func elevationColor(elev float64) color.RGBA {
	if elev < 0 {
		// Ocean: dark blue to light blue
		depth := math.Min(-elev/8000, 1.0)
		return color.RGBA{
			0,
			uint8(80 + 120*(1-depth)),
			uint8(120 + 100*(1-depth)),
			255,
		}
	}
	// Land gradient
	if elev < 500 {
		return color.RGBA{80, 160, 60, 255} // Green lowlands
	} else if elev < 1500 {
		t := (elev - 500) / 1000
		return color.RGBA{
			uint8(80 + 80*t),
			uint8(160 - 40*t),
			uint8(60 - 20*t),
			255,
		}
	} else if elev < 3000 {
		t := (elev - 1500) / 1500
		return color.RGBA{
			uint8(160 + 40*t),
			uint8(120 - 40*t),
			uint8(40 + 40*t),
			255,
		}
	} else {
		// High mountains: gray to white
		t := math.Min((elev-3000)/5000, 1.0)
		v := uint8(200 + 55*t)
		return color.RGBA{v, v, v, 255}
	}
}

func renderHypsometry(elevation []float64, filename string) {
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
		c := elevationColor(elev)

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

func printStats(elevation []float64, isLand []bool) {
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
