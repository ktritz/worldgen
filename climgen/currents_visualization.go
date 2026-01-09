package climgen

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sync"
)

// VisualizationSettings controls ocean current map rendering.
type VisualizationSettings struct {
	Width           int     // Image width in pixels
	Height          int     // Image height in pixels
	ArrowSpacing    int     // Pixels between arrow origins
	ArrowScale      float64 // Multiplier for arrow length
	ArrowHeadSize   float64 // Size of arrowhead relative to shaft
	MinSpeedVisible float64 // Minimum speed to draw arrow
	ShowBasins      bool    // Color-code basins in background
	ShowElevation   bool    // Show land elevation in background
}

// DefaultVisualizationSettings returns reasonable defaults for visualization.
func DefaultVisualizationSettings() VisualizationSettings {
	return VisualizationSettings{
		Width:           1024,
		Height:          512,
		ArrowSpacing:    16,
		ArrowScale:      100.0,
		ArrowHeadSize:   0.3,
		MinSpeedVisible: 0.001,
		ShowBasins:      true,
		ShowElevation:   true,
	}
}

// spatialIndex for fast nearest-neighbor lookups during rendering.
type spatialIndex struct {
	buckets map[int][]int
	gridRes int
}

func buildSpatialIndex(vertices []Vector3D, gridRes int) *spatialIndex {
	idx := &spatialIndex{
		buckets: make(map[int][]int),
		gridRes: gridRes,
	}

	for i, v := range vertices {
		lat := math.Asin(v.Y) * 180 / math.Pi // Y-up
		lon := math.Atan2(v.Z, v.X) * 180 / math.Pi

		bx := int((lon + 180) / 360 * float64(gridRes))
		by := int((lat + 90) / 180 * float64(gridRes))
		if bx >= gridRes {
			bx = gridRes - 1
		}
		if by >= gridRes {
			by = gridRes - 1
		}

		key := by*gridRes + bx
		idx.buckets[key] = append(idx.buckets[key], i)
	}

	return idx
}

func (idx *spatialIndex) findNearest(lat, lon float64, vertices []Vector3D) int {
	// Convert to 3D for distance comparison
	latRad := lat * math.Pi / 180
	lonRad := lon * math.Pi / 180
	qx := math.Cos(latRad) * math.Cos(lonRad)
	qy := math.Sin(latRad) // Y-up
	qz := math.Cos(latRad) * math.Sin(lonRad)

	bx := int((lon + 180) / 360 * float64(idx.gridRes))
	by := int((lat + 90) / 180 * float64(idx.gridRes))

	// Expand search radius near poles
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
				v := vertices[i]
				d := (qx-v.X)*(qx-v.X) + (qy-v.Y)*(qy-v.Y) + (qz-v.Z)*(qz-v.Z)
				if d < bestDist {
					bestDist = d
					bestIdx = i
				}
			}
		}
	}

	return bestIdx
}

// RenderOceanCurrentsMap generates an equirectangular map with ocean current arrows.
func RenderOceanCurrentsMap(
	vertices []Vector3D,
	elevation []float64,
	result *OceanCurrentResult,
	outputPath string,
	settings VisualizationSettings,
) error {
	width := settings.Width
	height := settings.Height

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Build spatial index
	gridRes := 360
	index := buildSpatialIndex(vertices, gridRes)

	// Render background (elevation + basins) in parallel
	var wg sync.WaitGroup
	for py := 0; py < height; py++ {
		wg.Add(1)
		go func(py int) {
			defer wg.Done()

			lat := 90 - float64(py)/float64(height)*180

			for px := 0; px < width; px++ {
				lon := float64(px)/float64(width)*360 - 180

				idx := index.findNearest(lat, lon, vertices)
				elev := elevation[idx]

				var c color.RGBA

				if elev >= 0 {
					// Land
					c = elevationToLandColor(elev)
				} else {
					// Ocean
					if settings.ShowBasins && result.BasinAssignments[idx] >= 0 {
						basinID := result.BasinAssignments[idx]
						c = basinToColor(basinID, elev)
					} else {
						c = elevationToOceanColor(elev)
					}
				}

				img.Set(px, py, c)
			}
		}(py)
	}
	wg.Wait()

	// Draw current arrows
	drawCurrentArrows(img, vertices, result.Currents, elevation, index, settings)

	// Save image
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// drawCurrentArrows draws vector arrows on the map.
func drawCurrentArrows(
	img *image.RGBA,
	vertices []Vector3D,
	currents []Vector3D,
	elevation []float64,
	index *spatialIndex,
	settings VisualizationSettings,
) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	arrowColor := color.RGBA{255, 255, 255, 220}
	headColor := color.RGBA{255, 200, 100, 255}

	// Draw arrows at regular grid spacing
	for py := settings.ArrowSpacing / 2; py < height; py += settings.ArrowSpacing {
		for px := settings.ArrowSpacing / 2; px < width; px += settings.ArrowSpacing {
			lat := 90 - float64(py)/float64(height)*180
			lon := float64(px)/float64(width)*360 - 180

			idx := index.findNearest(lat, lon, vertices)

			// Skip land
			if elevation[idx] >= 0 {
				continue
			}

			current := currents[idx]
			speed := Length(current)

			if speed < settings.MinSpeedVisible {
				continue
			}

			// Project current vector to 2D (tangent to sphere at this point)
			// Approximate: use local east/north components
			vertex := vertices[idx]
			east, north := GetTangentVectors(vertex)

			// Current components in local tangent frame
			eastComp := Dot(current, east)
			northComp := Dot(current, north)

			// Scale for visibility
			// Multiply by cos(lat) to compensate for equirectangular projection stretch at high latitudes
			latRad := lat * math.Pi / 180
			cosLat := math.Cos(latRad)
			polarScale := 1.0
			if cosLat > 0.1 {
				polarScale = cosLat // Scale down at high latitudes for visual consistency
			} else {
				polarScale = 0.1 // Minimum scale near poles
			}

			dx := eastComp * settings.ArrowScale * polarScale
			dy := -northComp * settings.ArrowScale * polarScale // Negative because image Y increases downward

			// Draw arrow shaft
			drawLine(img, px, py, int(float64(px)+dx), int(float64(py)+dy), arrowColor)

			// Draw arrowhead
			if speed > settings.MinSpeedVisible*2 {
				drawArrowHead(img, px, py, dx, dy, settings.ArrowHeadSize, headColor)
			}
		}
	}
}

// drawLine draws a line using Bresenham's algorithm.
func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)

	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}

	err := dx - dy

	bounds := img.Bounds()

	for {
		if x0 >= 0 && x0 < bounds.Dx() && y0 >= 0 && y0 < bounds.Dy() {
			img.Set(x0, y0, c)
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// drawArrowHead draws a small triangle at the end of an arrow.
func drawArrowHead(img *image.RGBA, baseX, baseY int, dx, dy float64, sizeRatio float64, c color.RGBA) {
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 3 {
		return
	}

	// Normalize direction
	ndx := dx / length
	ndy := dy / length

	// Tip of arrow
	tipX := float64(baseX) + dx
	tipY := float64(baseY) + dy

	// Perpendicular direction
	perpX := -ndy
	perpY := ndx

	// Arrowhead size
	headLen := length * sizeRatio
	headWidth := headLen * 0.5

	// Two base points of arrowhead
	backX := tipX - ndx*headLen
	backY := tipY - ndy*headLen

	p1x := int(backX + perpX*headWidth)
	p1y := int(backY + perpY*headWidth)
	p2x := int(backX - perpX*headWidth)
	p2y := int(backY - perpY*headWidth)

	// Draw triangle edges
	drawLine(img, int(tipX), int(tipY), p1x, p1y, c)
	drawLine(img, int(tipX), int(tipY), p2x, p2y, c)
	drawLine(img, p1x, p1y, p2x, p2y, c)
}

// elevationToLandColor maps land elevation to a green-brown-white gradient.
func elevationToLandColor(elev float64) color.RGBA {
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
	}
	// High mountains
	t := math.Min((elev-3000)/5000, 1.0)
	v := uint8(200 + 55*t)
	return color.RGBA{v, v, v, 255}
}

// elevationToOceanColor maps ocean depth to a blue gradient.
func elevationToOceanColor(elev float64) color.RGBA {
	depth := -elev
	if depth > 8000 {
		depth = 8000
	}
	t := depth / 8000

	return color.RGBA{
		uint8(20 + 80*(1-t)),
		uint8(60 + 100*(1-t)),
		uint8(120 + 80*(1-t)),
		255,
	}
}

// basinToColor gives each basin a distinct tinted color.
func basinToColor(basinID int, elev float64) color.RGBA {
	// Base ocean color
	base := elevationToOceanColor(elev)

	// Hue shift based on basin ID
	hues := []color.RGBA{
		{40, 80, 160, 255},   // Deep blue
		{60, 120, 140, 255},  // Teal
		{80, 100, 180, 255},  // Purple-blue
		{30, 100, 120, 255},  // Dark teal
		{50, 90, 200, 255},   // Bright blue
		{70, 130, 130, 255},  // Cyan
		{40, 110, 150, 255},  // Steel blue
		{60, 80, 170, 255},   // Indigo
	}

	hue := hues[basinID%len(hues)]

	// Blend with base (70% hue, 30% base)
	return color.RGBA{
		uint8(float64(hue.R)*0.7 + float64(base.R)*0.3),
		uint8(float64(hue.G)*0.7 + float64(base.G)*0.3),
		uint8(float64(hue.B)*0.7 + float64(base.B)*0.3),
		255,
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
