package climgen

import "image/color"

func PopulationColor(c PopulationClass) color.RGBA {
	switch c {
	case PopulationOcean:
		return color.RGBA{50, 90, 150, 255}
	case PopulationUninhabited:
		return color.RGBA{78, 68, 62, 255}
	case PopulationSparseFrontier:
		return color.RGBA{165, 146, 96, 255}
	case PopulationRural:
		return color.RGBA{136, 168, 104, 255}
	case PopulationDenseRural:
		return color.RGBA{90, 148, 92, 255}
	case PopulationUrban:
		return color.RGBA{124, 74, 68, 255}
	default:
		return color.RGBA{0, 0, 0, 255}
	}
}
