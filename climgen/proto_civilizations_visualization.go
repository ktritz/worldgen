package climgen

import "image/color"

func ProtoCivilizationColor(id int) color.RGBA {
	palette := []color.RGBA{
		{R: 172, G: 92, B: 84, A: 255},
		{R: 94, G: 132, B: 188, A: 255},
		{R: 124, G: 156, B: 92, A: 255},
		{R: 196, G: 150, B: 86, A: 255},
		{R: 138, G: 108, B: 178, A: 255},
		{R: 82, G: 164, B: 156, A: 255},
		{R: 184, G: 112, B: 128, A: 255},
		{R: 112, G: 148, B: 96, A: 255},
	}
	if len(palette) == 0 {
		return color.RGBA{R: 160, G: 160, B: 160, A: 255}
	}
	return palette[id%len(palette)]
}
