package climgen

import "image/color"

func AgricultureColor(a AgricultureType) color.RGBA {
	switch a {
	case AgricultureOcean:
		return color.RGBA{65, 105, 180, 255}
	case AgricultureUnsuitable:
		return color.RGBA{196, 188, 176, 255}
	case AgriculturePastoral:
		return color.RGBA{194, 170, 102, 255}
	case AgricultureDryFarming:
		return color.RGBA{192, 156, 82, 255}
	case AgricultureMixedFarming:
		return color.RGBA{126, 168, 84, 255}
	case AgricultureIntensiveCropland:
		return color.RGBA{86, 170, 70, 255}
	case AgricultureFloodplainCropland:
		return color.RGBA{86, 174, 120, 255}
	default:
		return color.RGBA{255, 0, 255, 255}
	}
}
