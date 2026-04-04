package climgen

import "image/color"

func SoilColor(s SoilType) color.RGBA {
	switch s {
	case SoilOcean:
		return color.RGBA{65, 105, 180, 255}
	case SoilCryosol:
		return color.RGBA{212, 216, 222, 255}
	case SoilRocky:
		return color.RGBA{120, 110, 102, 255}
	case SoilAridMineral:
		return color.RGBA{210, 180, 132, 255}
	case SoilDrySteppe:
		return color.RGBA{168, 151, 97, 255}
	case SoilTemperateLoam:
		return color.RGBA{132, 106, 72, 255}
	case SoilTropicalWeathered:
		return color.RGBA{168, 82, 58, 255}
	case SoilAlluvial:
		return color.RGBA{110, 124, 88, 255}
	case SoilOrganicWet:
		return color.RGBA{74, 88, 64, 255}
	case SoilPeat:
		return color.RGBA{56, 62, 48, 255}
	case SoilSalineCoastal:
		return color.RGBA{188, 178, 154, 255}
	default:
		return color.RGBA{255, 0, 255, 255}
	}
}
