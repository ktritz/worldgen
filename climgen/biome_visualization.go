package climgen

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

type biomeAffinityFamily int

const (
	affinityDesert biomeAffinityFamily = iota
	affinityGrassland
	affinityForest
	affinityTropicalWet
	affinityIce
	affinityTundra
	affinityBoreal
	affinityWetland
	affinityAlpine
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
	case BiomeWetland:
		return color.RGBA{48, 116, 108, 255} // Marsh teal-green
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

// RenderFuzzyBiomeMap renders the primary biome color, then softly blends
// toward the secondary affinity-family color when the runner-up signal is both
// strong enough and close enough to the primary signal to indicate a genuine
// transition zone.
func RenderFuzzyBiomeMap(
	biomes []Biome,
	diagnostics *BiomeDiagnostics,
	elevation []float64,
	cellLookup []int,
	width, height int,
	outputPath string,
	secondThreshold float64,
) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			cellIdx := cellLookup[idx]
			if cellIdx < 0 || cellIdx >= len(biomes) {
				continue
			}

			base := BiomeColor(biomes[cellIdx])
			if cellIdx < len(elevation) && elevation[cellIdx] >= 0 && diagnostics != nil {
				_, firstValue, secondFamily, secondValue := topTwoAffinityFamilies(diagnostics, cellIdx)
				if secondValue >= secondThreshold {
					closeness := 0.0
					if firstValue > 1e-6 {
						closeness = secondValue / firstValue
					}
					blendAmount := 0.38 *
						smoothstepVisual(secondThreshold, 0.85, secondValue) *
						smoothstepVisual(0.55, 0.92, closeness)
					if blendAmount > 0 {
						base = blendColorsRGBA(base, affinityFamilyColor(secondFamily), blendAmount)
					}
				}
			}
			img.Set(x, y, base)
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// RenderBiomeScalarMap renders a diagnostic scalar field with a simple
// worldgen-oriented palette for biome inspection.
func RenderBiomeScalarMap(
	values []float64,
	elevation []float64,
	cellLookup []int,
	width, height int,
	outputPath string,
	minValue, maxValue float64,
	palette string,
) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			cellIdx := cellLookup[idx]
			if cellIdx < 0 || cellIdx >= len(values) {
				continue
			}
			if cellIdx < len(elevation) && elevation[cellIdx] < 0 {
				img.Set(x, y, color.RGBA{50, 78, 120, 255})
				continue
			}
			img.Set(x, y, biomeScalarColor(values[cellIdx], minValue, maxValue, palette))
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

func biomeScalarColor(value, minValue, maxValue float64, palette string) color.RGBA {
	t := 0.0
	if maxValue > minValue {
		t = (value - minValue) / (maxValue - minValue)
	}
	t = math.Max(0, math.Min(1, t))

	switch palette {
	case "precip":
		return gradientColor(t,
			color.RGBA{181, 138, 86, 255},
			color.RGBA{185, 214, 108, 255},
			color.RGBA{56, 148, 87, 255},
			color.RGBA{19, 84, 102, 255},
		)
	case "dryness":
		return gradientColor(t,
			color.RGBA{158, 111, 74, 255},
			color.RGBA{215, 197, 129, 255},
			color.RGBA{115, 170, 135, 255},
			color.RGBA{42, 108, 122, 255},
		)
	case "continentality":
		return gradientColor(t,
			color.RGBA{28, 102, 151, 255},
			color.RGBA{112, 178, 166, 255},
			color.RGBA{240, 207, 122, 255},
			color.RGBA{191, 90, 46, 255},
		)
	default: // thermal
		return gradientColor(t,
			color.RGBA{77, 57, 171, 255},
			color.RGBA{41, 148, 197, 255},
			color.RGBA{246, 208, 92, 255},
			color.RGBA{188, 55, 31, 255},
		)
	}
}

func gradientColor(t float64, stops ...color.RGBA) color.RGBA {
	if len(stops) == 0 {
		return color.RGBA{255, 0, 255, 255}
	}
	if len(stops) == 1 {
		return stops[0]
	}
	segments := len(stops) - 1
	scaled := t * float64(segments)
	idx := int(math.Min(float64(segments-1), math.Floor(scaled)))
	localT := scaled - float64(idx)
	return lerpRGBA(stops[idx], stops[idx+1], localT)
}

func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	lerp := func(x, y uint8) uint8 {
		return uint8(math.Round((1-t)*float64(x) + t*float64(y)))
	}
	return color.RGBA{
		R: lerp(a.R, b.R),
		G: lerp(a.G, b.G),
		B: lerp(a.B, b.B),
		A: 255,
	}
}

func topTwoAffinityFamilies(diag *BiomeDiagnostics, idx int) (biomeAffinityFamily, float64, biomeAffinityFamily, float64) {
	values := [9]float64{
		diag.DesertAffinity[idx],
		diag.GrasslandAffinity[idx],
		diag.ForestAffinity[idx],
		diag.TropicalWetAffinity[idx],
		diag.IceAffinity[idx],
		diag.TundraAffinity[idx],
		diag.BorealAffinity[idx],
		diag.WetlandAffinity[idx],
		diag.AlpineAffinity[idx],
	}

	firstFamily := affinityDesert
	secondFamily := affinityGrassland
	firstValue := -1.0
	secondValue := -1.0

	for i, value := range values {
		family := biomeAffinityFamily(i)
		if value > firstValue {
			secondFamily, secondValue = firstFamily, firstValue
			firstFamily, firstValue = family, value
		} else if value > secondValue {
			secondFamily, secondValue = family, value
		}
	}

	return firstFamily, firstValue, secondFamily, secondValue
}

func affinityFamilyColor(family biomeAffinityFamily) color.RGBA {
	switch family {
	case affinityDesert:
		return color.RGBA{166, 103, 55, 255}
	case affinityGrassland:
		return color.RGBA{208, 195, 102, 255}
	case affinityForest:
		return color.RGBA{20, 94, 54, 255}
	case affinityTropicalWet:
		return color.RGBA{18, 125, 103, 255}
	case affinityIce:
		return color.RGBA{223, 232, 255, 255}
	case affinityTundra:
		return color.RGBA{160, 185, 188, 255}
	case affinityBoreal:
		return color.RGBA{58, 97, 102, 255}
	case affinityWetland:
		return color.RGBA{63, 132, 123, 255}
	case affinityAlpine:
		return color.RGBA{130, 130, 130, 255}
	default:
		return color.RGBA{255, 0, 255, 255}
	}
}

func blendColorsRGBA(a, b color.RGBA, amount float64) color.RGBA {
	amount = clamp01Visual(amount)
	return color.RGBA{
		R: uint8(math.Round((1-amount)*float64(a.R) + amount*float64(b.R))),
		G: uint8(math.Round((1-amount)*float64(a.G) + amount*float64(b.G))),
		B: uint8(math.Round((1-amount)*float64(a.B) + amount*float64(b.B))),
		A: 255,
	}
}

func smoothstepVisual(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		if x >= edge1 {
			return 1
		}
		return 0
	}
	t := (x - edge0) / (edge1 - edge0)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return t * t * (3 - 2*t)
}

func clamp01Visual(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
