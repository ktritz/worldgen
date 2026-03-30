package climgen

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func VegetationColor(v VegetationType) color.RGBA {
	switch v {
	case VegetationOcean:
		return color.RGBA{65, 105, 180, 255}
	case VegetationIceBarren:
		return color.RGBA{232, 232, 236, 255}
	case VegetationDesertSparse:
		return color.RGBA{218, 191, 150, 255}
	case VegetationShrubland:
		return color.RGBA{161, 140, 92, 255}
	case VegetationGrassland:
		return color.RGBA{170, 196, 88, 255}
	case VegetationWoodland:
		return color.RGBA{97, 147, 77, 255}
	case VegetationForest:
		return color.RGBA{45, 109, 52, 255}
	case VegetationRainforest:
		return color.RGBA{18, 81, 42, 255}
	case VegetationWetland:
		return color.RGBA{68, 132, 118, 255}
	case VegetationMangrove:
		return color.RGBA{42, 92, 80, 255}
	case VegetationSaltMarsh:
		return color.RGBA{112, 156, 132, 255}
	case VegetationPeatland:
		return color.RGBA{87, 95, 71, 255}
	case VegetationRiparianForest:
		return color.RGBA{30, 121, 74, 255}
	case VegetationCloudForest:
		return color.RGBA{38, 102, 89, 255}
	case VegetationAlpineMeadow:
		return color.RGBA{140, 166, 116, 255}
	default:
		return color.RGBA{255, 0, 255, 255}
	}
}

func RenderVegetationMap(
	types []VegetationType,
	cellLookup []int,
	width, height int,
	outputPath string,
) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			cellIdx := cellLookup[idx]
			if cellIdx >= 0 && cellIdx < len(types) {
				img.Set(x, y, VegetationColor(types[cellIdx]))
			}
		}
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
