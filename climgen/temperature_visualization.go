package climgen

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sync"
)

// =============================================================================
// TEMPERATURE GENERATION - VISUALIZATION
// =============================================================================
// This file contains functions for rendering temperature maps.

// TemperatureVisualizationSettings controls temperature map rendering.
type TemperatureVisualizationSettings struct {
	Width          int     // Image width in pixels
	Height         int     // Image height in pixels
	MinTempC       float64 // Color scale minimum (°C)
	MaxTempC       float64 // Color scale maximum (°C)
	ShowIsotherms  bool    // Draw temperature contour lines
	IsothermStep   float64 // Degrees between isotherms
	ShowCoastlines bool    // Draw coastline outlines
	ShowElevation  bool    // Blend elevation into land colors
	ShowOceanBath  bool    // Show ocean bathymetry
}

// DefaultTemperatureVisualizationSettings returns reasonable defaults.
func DefaultTemperatureVisualizationSettings() TemperatureVisualizationSettings {
	return TemperatureVisualizationSettings{
		Width:          1024,
		Height:         512,
		MinTempC:       -40.0, // Arctic/Antarctic
		MaxTempC:       40.0,  // Tropics
		ShowIsotherms:  false, // Off by default - can be clunky
		IsothermStep:   10.0,  // 10°C intervals
		ShowCoastlines: true,  // Show land outlines
		ShowElevation:  false, // Pure temperature colors
		ShowOceanBath:  false,
	}
}

// RenderTemperatureMap generates an equirectangular temperature map.
// Uses a blue-white-red color gradient for temperature.
func RenderTemperatureMap(
	vertices []Vector3D,
	elevation []float64,
	result *TemperatureResult,
	outputPath string,
	settings TemperatureVisualizationSettings,
) error {
	width := settings.Width
	height := settings.Height

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Build spatial index (reused from currents_visualization.go)
	gridRes := 360
	index := buildSpatialIndex(vertices, gridRes)

	// Render temperature map in parallel
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
				tempC := result.TemperatureCelsius[idx]

				var c color.RGBA

				if elev >= 0 {
					// Land: blend temperature color with elevation
					tempColor := temperatureToColor(tempC, settings.MinTempC, settings.MaxTempC)

					if settings.ShowElevation {
						// Blend with elevation-based shading
						elevShade := elevationShade(elev)
						c = blendColors(tempColor, elevShade, 0.3)
					} else {
						c = tempColor
					}
				} else {
					// Ocean: use temperature color with optional bathymetry
					tempColor := temperatureToColor(tempC, settings.MinTempC, settings.MaxTempC)

					if settings.ShowOceanBath {
						// Blend with depth-based shading
						depthShade := depthShade(elev)
						c = blendColors(tempColor, depthShade, 0.2)
					} else {
						c = tempColor
					}
				}

				img.Set(px, py, c)
			}
		}(py)
	}
	wg.Wait()

	// Draw coastlines if enabled
	if settings.ShowCoastlines {
		drawCoastlines(img, vertices, elevation, index)
	}

	// Draw isotherms if enabled
	if settings.ShowIsotherms {
		drawIsotherms(img, vertices, result.TemperatureCelsius, index, settings)
	}

	// Save image
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// RenderTemperatureWithTerrain generates a map showing temperature over terrain.
// Land shows elevation coloring with temperature isotherms overlay.
// Ocean shows temperature coloring.
func RenderTemperatureWithTerrain(
	vertices []Vector3D,
	elevation []float64,
	result *TemperatureResult,
	outputPath string,
	settings TemperatureVisualizationSettings,
) error {
	width := settings.Width
	height := settings.Height

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	gridRes := 360
	index := buildSpatialIndex(vertices, gridRes)

	// Render in parallel
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
				tempC := result.TemperatureCelsius[idx]

				var c color.RGBA

				if elev >= 0 {
					// Land: elevation coloring
					c = elevationToLandColor(elev)

					// Add frost/snow effect for cold temperatures
					if tempC < 0 {
						// Blend toward white for freezing temperatures
						frostFactor := math.Min(-tempC/20.0, 1.0) // Full frost at -20°C
						c = blendColors(c, color.RGBA{255, 255, 255, 255}, frostFactor*0.5)
					}
				} else {
					// Ocean: temperature coloring
					c = temperatureToColor(tempC, settings.MinTempC, settings.MaxTempC)
				}

				img.Set(px, py, c)
			}
		}(py)
	}
	wg.Wait()

	// Draw isotherms over everything
	if settings.ShowIsotherms {
		drawIsotherms(img, vertices, result.TemperatureCelsius, index, settings)
	}

	// Save
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// --- Color Functions ---

// temperatureToColor maps temperature to a full-spectrum gradient.
// Purple (coldest) → Blue → Cyan → Green → Yellow → Orange → Red (hottest)
// Based on the "turbo" colormap style for scientific visualization.
func temperatureToColor(tempC, minC, maxC float64) color.RGBA {
	// Normalize to [0, 1]
	t := (tempC - minC) / (maxC - minC)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	// Control points: purple → blue → cyan → green → yellow → orange → red
	type colorPoint struct {
		pos     float64
		r, g, b float64
	}
	points := []colorPoint{
		{0.00, 60, 20, 110},   // Deep purple (coldest)
		{0.10, 70, 40, 150},   // Purple
		{0.20, 50, 80, 180},   // Blue-purple
		{0.30, 30, 120, 200},  // Blue
		{0.40, 40, 170, 190},  // Cyan
		{0.50, 60, 190, 130},  // Teal/Green
		{0.60, 100, 200, 80},  // Green
		{0.70, 170, 210, 50},  // Yellow-green
		{0.80, 230, 190, 40},  // Yellow
		{0.90, 250, 130, 40},  // Orange
		{1.00, 200, 50, 30},   // Red (hottest)
	}

	// Find the two control points we're between
	var p1, p2 colorPoint
	for i := 0; i < len(points)-1; i++ {
		if t >= points[i].pos && t <= points[i+1].pos {
			p1 = points[i]
			p2 = points[i+1]
			break
		}
	}

	// Linear interpolation between the two points
	segmentT := (t - p1.pos) / (p2.pos - p1.pos)

	r := p1.r + (p2.r-p1.r)*segmentT
	g := p1.g + (p2.g-p1.g)*segmentT
	b := p1.b + (p2.b-p1.b)*segmentT

	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

// elevationShade returns a grayscale shade based on elevation for blending.
func elevationShade(elev float64) color.RGBA {
	// Normalize: 0m = 128, 5000m = 255
	shade := 128 + int(elev/5000*127)
	if shade > 255 {
		shade = 255
	}
	if shade < 0 {
		shade = 0
	}
	return color.RGBA{uint8(shade), uint8(shade), uint8(shade), 255}
}

// depthShade returns a shade based on ocean depth.
func depthShade(elev float64) color.RGBA {
	depth := -elev
	if depth < 0 {
		depth = 0
	}
	if depth > 8000 {
		depth = 8000
	}
	// Deeper = darker
	shade := 180 - int(depth/8000*120)
	return color.RGBA{uint8(shade), uint8(shade), uint8(shade), 255}
}

// blendColors blends two colors with the given weight for the second color.
func blendColors(c1, c2 color.RGBA, weight float64) color.RGBA {
	w1 := 1.0 - weight
	w2 := weight
	return color.RGBA{
		uint8(float64(c1.R)*w1 + float64(c2.R)*w2),
		uint8(float64(c1.G)*w1 + float64(c2.G)*w2),
		uint8(float64(c1.B)*w1 + float64(c2.B)*w2),
		255,
	}
}

// --- Coastline Drawing ---

// drawCoastlines draws land/ocean boundaries on the image.
func drawCoastlines(
	img *image.RGBA,
	vertices []Vector3D,
	elevation []float64,
	index *spatialIndex,
) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	coastColor := color.RGBA{30, 30, 30, 255} // Dark gray

	// For each pixel, check if it's at a land/ocean boundary
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180

		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180

			idx := index.findNearest(lat, lon, vertices)
			isLand := elevation[idx] >= 0

			// Check if any neighbor pixel has different land/ocean status
			isCoast := false
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					npx := px + dx
					npy := py + dy
					if npx < 0 || npx >= width || npy < 0 || npy >= height {
						continue
					}

					nlat := 90 - float64(npy)/float64(height)*180
					nlon := float64(npx)/float64(width)*360 - 180
					nidx := index.findNearest(nlat, nlon, vertices)
					nIsLand := elevation[nidx] >= 0

					if isLand != nIsLand {
						isCoast = true
						break
					}
				}
				if isCoast {
					break
				}
			}

			if isCoast {
				img.Set(px, py, coastColor)
			}
		}
	}
}

// --- Isotherm Drawing ---

// drawIsotherms draws temperature contour lines at regular intervals.
func drawIsotherms(
	img *image.RGBA,
	vertices []Vector3D,
	tempC []float64,
	index *spatialIndex,
	settings TemperatureVisualizationSettings,
) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	step := settings.IsothermStep

	// Isotherm color: dark gray, semi-transparent effect via dithering
	isothermColor := color.RGBA{40, 40, 40, 255}

	// For each pixel, check if an isotherm passes through it
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180

		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180

			idx := index.findNearest(lat, lon, vertices)
			t := tempC[idx]

			// Check if we're near an isotherm boundary
			// An isotherm at value V is where t crosses V
			tMod := math.Mod(t-settings.MinTempC, step)
			if tMod < 0 {
				tMod += step
			}

			// Near isotherm if we're within a small fraction of the step
			threshold := step * 0.08 // 8% of step width
			if tMod < threshold || tMod > step-threshold {
				// Check neighboring pixels to confirm this is an edge
				hasLower := false
				hasHigher := false

				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						npx := px + dx
						npy := py + dy
						if npx < 0 || npx >= width || npy < 0 || npy >= height {
							continue
						}

						nlat := 90 - float64(npy)/float64(height)*180
						nlon := float64(npx)/float64(width)*360 - 180
						nidx := index.findNearest(nlat, nlon, vertices)
						nt := tempC[nidx]

						// Round to nearest isotherm
						iso := math.Round(t/step) * step
						if nt < iso {
							hasLower = true
						}
						if nt > iso {
							hasHigher = true
						}
					}
				}

				// Only draw if this is actually a contour crossing
				if hasLower && hasHigher {
					img.Set(px, py, isothermColor)
				}
			}
		}
	}
}

// --- Diagnostic Visualizations ---

// RenderAlbedoMap renders the surface albedo values.
func RenderAlbedoMap(
	vertices []Vector3D,
	elevation []float64,
	result *TemperatureResult,
	outputPath string,
	width, height int,
) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	gridRes := 360
	index := buildSpatialIndex(vertices, gridRes)

	var wg sync.WaitGroup
	for py := 0; py < height; py++ {
		wg.Add(1)
		go func(py int) {
			defer wg.Done()

			lat := 90 - float64(py)/float64(height)*180

			for px := 0; px < width; px++ {
				lon := float64(px)/float64(width)*360 - 180

				idx := index.findNearest(lat, lon, vertices)
				albedo := result.Albedo[idx]

				// Albedo 0 (black) to 1 (white)
				v := uint8(albedo * 255)
				img.Set(px, py, color.RGBA{v, v, v, 255})
			}
		}(py)
	}
	wg.Wait()

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// RenderTemperatureDiffMap renders the difference between two temperature arrays.
// Uses a diverging colormap: blue (colder) - white (no change) - red (warmer).
// The scale is symmetric and auto-scaled to the maximum absolute difference.
func RenderTemperatureDiffMap(
	vertices []Vector3D,
	elevation []float64,
	baselineTemps []float64, // Reference temperatures (e.g., no wind)
	comparisonTemps []float64, // Comparison temperatures (e.g., with wind)
	outputPath string,
	width, height int,
) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	gridRes := 360
	index := buildSpatialIndex(vertices, gridRes)

	// Compute differences and find max absolute difference
	diffs := make([]float64, len(baselineTemps))
	maxAbsDiff := 0.0
	for i := range diffs {
		diffs[i] = comparisonTemps[i] - baselineTemps[i]
		if math.Abs(diffs[i]) > maxAbsDiff {
			maxAbsDiff = math.Abs(diffs[i])
		}
	}

	// Compute stats for logging
	landSum, landCount := 0.0, 0
	oceanSum, oceanCount := 0.0, 0
	for i, d := range diffs {
		if elevation[i] >= 0 {
			landSum += d
			landCount++
		} else {
			oceanSum += d
			oceanCount++
		}
	}

	// Print stats
	if landCount > 0 {
		// fmt.Printf uses import from os package indirectly, use simple approach
		_ = landSum / float64(landCount)
	}
	if oceanCount > 0 {
		_ = oceanSum / float64(oceanCount)
	}

	// Use at least 1°C scale to avoid over-amplifying tiny differences
	if maxAbsDiff < 1.0 {
		maxAbsDiff = 1.0
	}

	var wg sync.WaitGroup
	for py := 0; py < height; py++ {
		wg.Add(1)
		go func(py int) {
			defer wg.Done()

			lat := 90 - float64(py)/float64(height)*180

			for px := 0; px < width; px++ {
				lon := float64(px)/float64(width)*360 - 180

				idx := index.findNearest(lat, lon, vertices)
				diff := diffs[idx]

				// Normalize to [-1, 1]
				t := diff / maxAbsDiff
				if t < -1 {
					t = -1
				}
				if t > 1 {
					t = 1
				}

				c := diffToColor(t)
				img.Set(px, py, c)
			}
		}(py)
	}
	wg.Wait()

	// Draw coastlines for reference
	drawCoastlines(img, vertices, elevation, index)

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// diffToColor maps a normalized difference [-1, 1] to a diverging color.
// -1 = blue (colder), 0 = white (no change), +1 = red (warmer)
func diffToColor(t float64) color.RGBA {
	if t < 0 {
		// Blue side: white to blue
		factor := -t // 0 to 1
		return color.RGBA{
			uint8(255 * (1 - factor)),         // R decreases
			uint8(255 * (1 - factor*0.5)),     // G decreases slower
			255,                                // B stays max
			255,
		}
	} else {
		// Red side: white to red
		factor := t // 0 to 1
		return color.RGBA{
			255,                                // R stays max
			uint8(255 * (1 - factor*0.7)),     // G decreases
			uint8(255 * (1 - factor)),         // B decreases
			255,
		}
	}
}

// RenderInsolationMap renders the solar insolation distribution.
func RenderInsolationMap(
	vertices []Vector3D,
	result *TemperatureResult,
	outputPath string,
	width, height int,
) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	gridRes := 360
	index := buildSpatialIndex(vertices, gridRes)

	// Find range
	minQ, maxQ := result.Insolation[0], result.Insolation[0]
	for _, q := range result.Insolation {
		if q < minQ {
			minQ = q
		}
		if q > maxQ {
			maxQ = q
		}
	}
	rangeQ := maxQ - minQ
	if rangeQ < 1 {
		rangeQ = 1
	}

	var wg sync.WaitGroup
	for py := 0; py < height; py++ {
		wg.Add(1)
		go func(py int) {
			defer wg.Done()

			lat := 90 - float64(py)/float64(height)*180

			for px := 0; px < width; px++ {
				lon := float64(px)/float64(width)*360 - 180

				idx := index.findNearest(lat, lon, vertices)
				q := result.Insolation[idx]

				// Yellow-red gradient for solar radiation
				t := (q - minQ) / rangeQ
				c := color.RGBA{
					uint8(200 + 55*t),
					uint8(200 - 150*t),
					uint8(50),
					255,
				}
				img.Set(px, py, c)
			}
		}(py)
	}
	wg.Wait()

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}
