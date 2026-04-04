package climgen

import "image/color"

func WaterResourceColor(w WaterResourceType) color.RGBA {
	switch w {
	case WaterResourceOcean:
		return color.RGBA{50, 83, 138, 255}
	case WaterResourceScarce:
		return color.RGBA{205, 194, 170, 255}
	case WaterResourceSeasonal:
		return color.RGBA{183, 170, 91, 255}
	case WaterResourceReliableSurface:
		return color.RGBA{72, 135, 196, 255}
	case WaterResourceGroundwater:
		return color.RGBA{94, 150, 125, 255}
	case WaterResourceLakeOasis:
		return color.RGBA{74, 172, 182, 255}
	default:
		return color.RGBA{255, 0, 255, 255}
	}
}
