package climgen

import "image/color"

func PolitySphereColor(id int) color.RGBA {
	palette := []color.RGBA{
		{R: 178, G: 92, B: 90, A: 255},
		{R: 82, G: 126, B: 182, A: 255},
		{R: 108, G: 152, B: 90, A: 255},
		{R: 190, G: 148, B: 82, A: 255},
		{R: 140, G: 102, B: 176, A: 255},
		{R: 82, G: 156, B: 148, A: 255},
		{R: 186, G: 112, B: 140, A: 255},
	}
	return palette[id%len(palette)]
}
