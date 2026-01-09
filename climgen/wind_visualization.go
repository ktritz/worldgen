package climgen

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sync"
)

// WindVisualizationSettings controls wind map rendering.
type WindVisualizationSettings struct {
	Width           int     // Image width in pixels
	Height          int     // Image height in pixels
	ArrowSpacing    int     // Pixels between arrow origins
	ArrowScale      float64 // Multiplier for arrow length
	ArrowHeadSize   float64 // Size of arrowhead relative to shaft
	MinWindVisible  float64 // Minimum wind speed to draw arrow
	ShowPressure    bool    // Color-code pressure in ocean areas
	ShowElevation   bool    // Show land elevation
	ShowZones       bool    // Show circulation zone boundaries
}

// DefaultWindVisualizationSettings returns reasonable defaults.
func DefaultWindVisualizationSettings() WindVisualizationSettings {
	return WindVisualizationSettings{
		Width:          1024,
		Height:         512,
		ArrowSpacing:   16,
		ArrowScale:     80.0,
		ArrowHeadSize:  0.3,
		MinWindVisible: 0.001,
		ShowPressure:   true,
		ShowElevation:  true,
		ShowZones:      false,
	}
}

// RenderWindMap generates an equirectangular map with wind arrows.
func RenderWindMap(
	vertices []Vector3D,
	elevation []float64,
	result *WindResult,
	outputPath string,
	settings WindVisualizationSettings,
) error {
	width := settings.Width
	height := settings.Height

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Build spatial index (reuse from currents_visualization)
	gridRes := 360
	index := buildSpatialIndex(vertices, gridRes)

	// Find pressure range for normalization
	minPressure, maxPressure := result.Pressure[0], result.Pressure[0]
	for _, p := range result.Pressure {
		if p < minPressure {
			minPressure = p
		}
		if p > maxPressure {
			maxPressure = p
		}
	}
	pressureRange := maxPressure - minPressure
	if pressureRange < 1e-9 {
		pressureRange = 1.0
	}

	// Render background in parallel
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
					// Land - use elevation coloring
					if settings.ShowElevation {
						c = elevationToLandColor(elev)
					} else {
						c = color.RGBA{80, 120, 60, 255}
					}
				} else {
					// Ocean - use pressure coloring
					if settings.ShowPressure {
						p := result.Pressure[idx]
						c = pressureToColor(p, minPressure, pressureRange)
					} else {
						c = elevationToOceanColor(elev)
					}
				}

				img.Set(px, py, c)
			}
		}(py)
	}
	wg.Wait()

	// Draw zone boundaries if enabled
	if settings.ShowZones {
		drawZoneBoundaries(img, height)
	}

	// Draw wind arrows
	drawWindArrows(img, vertices, result.SurfaceWind, elevation, index, settings)

	// Save image
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// drawWindArrows draws vector arrows on the map.
func drawWindArrows(
	img *image.RGBA,
	vertices []Vector3D,
	wind []Vector3D,
	elevation []float64,
	index *spatialIndex,
	settings WindVisualizationSettings,
) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	// Arrow colors - white shaft, yellow head
	arrowColor := color.RGBA{255, 255, 255, 220}
	headColor := color.RGBA{255, 220, 100, 255}

	// Draw arrows at regular grid spacing
	for py := settings.ArrowSpacing / 2; py < height; py += settings.ArrowSpacing {
		for px := settings.ArrowSpacing / 2; px < width; px += settings.ArrowSpacing {
			lat := 90 - float64(py)/float64(height)*180
			lon := float64(px)/float64(width)*360 - 180

			idx := index.findNearest(lat, lon, vertices)

			windVec := wind[idx]
			speed := Length(windVec)

			if speed < settings.MinWindVisible {
				continue
			}

			// Project wind vector to 2D using local tangent frame
			vertex := vertices[idx]
			east, north := GetTangentVectors(vertex)

			// Wind components in local tangent frame
			eastComp := Dot(windVec, east)
			northComp := Dot(windVec, north)

			// Scale for visibility
			// Adjust for equirectangular projection stretch at high latitudes
			latRad := lat * math.Pi / 180
			cosLat := math.Cos(latRad)
			polarScale := 1.0
			if cosLat > 0.1 {
				polarScale = cosLat
			} else {
				polarScale = 0.1
			}

			dx := eastComp * settings.ArrowScale * polarScale
			dy := -northComp * settings.ArrowScale * polarScale // Negative: image Y increases downward

			// Draw arrow shaft
			drawLine(img, px, py, int(float64(px)+dx), int(float64(py)+dy), arrowColor)

			// Draw arrowhead
			if speed > settings.MinWindVisible*2 {
				drawArrowHead(img, px, py, dx, dy, settings.ArrowHeadSize, headColor)
			}
		}
	}
}

// pressureToColor maps pressure to a red-blue gradient.
// High pressure = warm colors (red/orange), Low pressure = cool colors (blue/purple)
func pressureToColor(pressure, minPressure, pressureRange float64) color.RGBA {
	// Normalize to [0, 1]
	t := (pressure - minPressure) / pressureRange

	// Blue (low) to white (mid) to red (high)
	if t < 0.5 {
		// Low pressure: blue to white
		s := t * 2 // 0 to 1
		return color.RGBA{
			uint8(60 + 195*s),  // 60 -> 255
			uint8(80 + 175*s),  // 80 -> 255
			uint8(180 + 75*s),  // 180 -> 255
			255,
		}
	}
	// High pressure: white to red
	s := (t - 0.5) * 2 // 0 to 1
	return color.RGBA{
		255,
		uint8(255 - 155*s), // 255 -> 100
		uint8(255 - 175*s), // 255 -> 80
		255,
	}
}

// drawZoneBoundaries draws horizontal lines at circulation zone boundaries.
func drawZoneBoundaries(img *image.RGBA, height int) {
	width := img.Bounds().Dx()
	lineColor := color.RGBA{200, 200, 200, 128}

	// Draw lines at 30° and 60° N and S (Hadley/Ferrel and Ferrel/Polar boundaries)
	latitudes := []float64{30, 60, -30, -60}

	for _, lat := range latitudes {
		py := int((90 - lat) / 180 * float64(height))
		for px := 0; px < width; px += 4 { // Dashed line
			if px+2 < width {
				img.Set(px, py, lineColor)
				img.Set(px+1, py, lineColor)
			}
		}
	}
}
