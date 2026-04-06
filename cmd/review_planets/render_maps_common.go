package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

func saveMapPNG(filename string, img image.Image, label string) {
	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("  create %s %s: %v\n", label, filename, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Printf("  encode %s %s: %v\n", label, filename, err)
		return
	}
	fmt.Printf("  Saved %s\n", filename)
}

func blendReviewColor(base, over color.RGBA, alpha float64) color.RGBA {
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	return color.RGBA{
		R: uint8(float64(base.R)*(1-alpha) + float64(over.R)*alpha),
		G: uint8(float64(base.G)*(1-alpha) + float64(over.G)*alpha),
		B: uint8(float64(base.B)*(1-alpha) + float64(over.B)*alpha),
		A: 255,
	}
}

func settlementRegionColor(regionID int) color.RGBA {
	palette := []color.RGBA{
		{176, 112, 88, 255},
		{104, 148, 102, 255},
		{90, 128, 174, 255},
		{168, 148, 86, 255},
		{136, 108, 162, 255},
		{88, 150, 156, 255},
	}
	if len(palette) == 0 {
		return color.RGBA{0, 0, 0, 255}
	}
	return palette[regionID%len(palette)]
}

