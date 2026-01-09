package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: diff_images <image1.png> <image2.png> <output.png>")
		fmt.Println("Shows (image2 - image1) with blue=colder, red=warmer")
		os.Exit(1)
	}

	img1, err := loadPNG(os.Args[1])
	if err != nil {
		fmt.Printf("Error loading %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}

	img2, err := loadPNG(os.Args[2])
	if err != nil {
		fmt.Printf("Error loading %s: %v\n", os.Args[2], err)
		os.Exit(1)
	}

	bounds := img1.Bounds()
	diff := image.NewRGBA(bounds)

	maxDiff := 0.0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r1, g1, b1, _ := img1.At(x, y).RGBA()
			r2, g2, b2, _ := img2.At(x, y).RGBA()

			// Convert to brightness (simple average)
			bright1 := (float64(r1) + float64(g1) + float64(b1)) / 3.0
			bright2 := (float64(r2) + float64(g2) + float64(b2)) / 3.0

			d := bright2 - bright1
			if d > maxDiff {
				maxDiff = d
			}
			if -d > maxDiff {
				maxDiff = -d
			}
		}
	}

	fmt.Printf("Max brightness diff: %.0f\n", maxDiff)

	// Scale factor for visibility (normalize to ±128)
	scale := 128.0 * 256.0 / maxDiff
	if maxDiff < 1 {
		scale = 1.0
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r1, g1, b1, _ := img1.At(x, y).RGBA()
			r2, g2, b2, _ := img2.At(x, y).RGBA()

			bright1 := (float64(r1) + float64(g1) + float64(b1)) / 3.0
			bright2 := (float64(r2) + float64(g2) + float64(b2)) / 3.0

			d := (bright2 - bright1) * scale / 256.0

			var c color.RGBA
			if d > 0 {
				// Warmer = red
				c = color.RGBA{uint8(min(255, 128+d)), 128, uint8(max(0, 128-d)), 255}
			} else {
				// Colder = blue
				c = color.RGBA{uint8(max(0, 128+d)), 128, uint8(min(255, 128-d)), 255}
			}
			diff.Set(x, y, c)
		}
	}

	f, err := os.Create(os.Args[3])
	if err != nil {
		fmt.Printf("Error creating output: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	png.Encode(f, diff)
	fmt.Printf("Saved diff to %s\n", os.Args[3])
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
