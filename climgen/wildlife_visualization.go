package climgen

import "image/color"

func WildlifeColor(w WildlifeType) color.RGBA {
	switch w {
	case WildlifeOcean:
		return color.RGBA{65, 105, 180, 255}
	case WildlifeSparse:
		return color.RGBA{198, 190, 176, 255}
	case WildlifeGrazingGame:
		return color.RGBA{184, 166, 96, 255}
	case WildlifeForestGame:
		return color.RGBA{92, 132, 78, 255}
	case WildlifeWetlandGame:
		return color.RGBA{96, 146, 118, 255}
	case WildlifePelts:
		return color.RGBA{118, 110, 144, 255}
	case WildlifeTimber:
		return color.RGBA{78, 120, 64, 255}
	default:
		return color.RGBA{255, 0, 255, 255}
	}
}
