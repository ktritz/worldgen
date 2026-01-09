package climgen

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

// BiomeColor returns the display color for each biome
func BiomeColor(b Biome) color.RGBA {
	switch b {
	case BiomeOcean:
		return color.RGBA{65, 105, 180, 255} // Steel blue
	case BiomeIceCap:
		return color.RGBA{250, 250, 255, 255} // Almost white
	case BiomeTundra:
		return color.RGBA{180, 200, 200, 255} // Grayish blue-green
	case BiomeBorealForest:
		return color.RGBA{34, 85, 68, 255} // Dark teal-green
	case BiomeTemperateRainforest:
		return color.RGBA{0, 100, 80, 255} // Deep teal
	case BiomeTemperateForest:
		return color.RGBA{34, 139, 34, 255} // Forest green
	case BiomeTemperateGrassland:
		return color.RGBA{154, 205, 50, 255} // Yellow-green
	case BiomeMediterranean:
		return color.RGBA{189, 183, 107, 255} // Dark khaki
	case BiomeDesertCold:
		return color.RGBA{210, 180, 140, 255} // Tan
	case BiomeDesertHot:
		return color.RGBA{237, 201, 175, 255} // Light sandy
	case BiomeSemiArid:
		return color.RGBA{195, 176, 145, 255} // Khaki
	case BiomeSavanna:
		return color.RGBA{218, 190, 130, 255} // Tan-gold
	case BiomeTropicalSeasonalForest:
		return color.RGBA{60, 150, 60, 255} // Medium green
	case BiomeTropicalRainforest:
		return color.RGBA{0, 80, 40, 255} // Very dark green
	case BiomeAlpine:
		return color.RGBA{139, 137, 137, 255} // Gray
	default:
		return color.RGBA{255, 0, 255, 255} // Magenta for unknown
	}
}

// RenderBiomeMap renders biomes to an equirectangular projection image.
// Uses pre-computed cell lookup for pixel-to-cell mapping.
func RenderBiomeMap(
	biomes []Biome,
	cellLookup []int,
	width, height int,
	outputPath string,
) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			cellIdx := cellLookup[idx]
			if cellIdx >= 0 && cellIdx < len(biomes) {
				img.Set(x, y, BiomeColor(biomes[cellIdx]))
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
