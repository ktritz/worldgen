package climgen

// =============================================================================
// BIOME CLASSIFICATION - WHITTAKER DIAGRAM APPROACH
// =============================================================================
// Classifies cells into biomes based on temperature and precipitation.
// Uses a simplified Whittaker diagram with adjustments for elevation.

// Biome represents a major ecosystem type
type Biome int

const (
	BiomeOcean Biome = iota
	BiomeIceCap
	BiomeTundra
	BiomeBorealForest // Taiga
	BiomeTemperateRainforest
	BiomeTemperateForest // Deciduous
	BiomeTemperateGrassland
	BiomeMediterranean
	BiomeDesertCold
	BiomeDesertHot
	BiomeSemiArid // Steppe
	BiomeSavanna
	BiomeTropicalSeasonalForest
	BiomeTropicalRainforest
	BiomeWetland
	BiomeAlpine // High elevation
)

// BiomeName returns human-readable name for a biome
func BiomeName(b Biome) string {
	names := []string{
		"Ocean",
		"Ice Cap",
		"Tundra",
		"Boreal Forest",
		"Temperate Rainforest",
		"Temperate Forest",
		"Temperate Grassland",
		"Mediterranean",
		"Cold Desert",
		"Hot Desert",
		"Semi-Arid",
		"Savanna",
		"Tropical Seasonal Forest",
		"Tropical Rainforest",
		"Wetland",
		"Alpine",
	}
	if int(b) < len(names) {
		return names[b]
	}
	return "Unknown"
}

// BiomeResult holds computed biomes for each cell
type BiomeResult struct {
	Biomes      []Biome
	Diagnostics *BiomeDiagnostics
}

// ClassifyBiomes assigns a biome to each cell based on temperature and precipitation.
//
// Parameters:
//   - temperature: temperature in Celsius for each cell
//   - precipitation: precipitation in cm/year for each cell
//   - elevation: elevation in meters for each cell
//   - seaLevel: threshold for ocean (typically 0)
func ClassifyBiomes(
	temperature []float64,
	precipitation []float64,
	elevation []float64,
	seaLevel float64,
) *BiomeResult {
	n := len(temperature)
	result := &BiomeResult{
		Biomes: make([]Biome, n),
	}

	for i := 0; i < n; i++ {
		result.Biomes[i] = classifyCell(
			temperature[i],
			precipitation[i],
			elevation[i],
			seaLevel,
		)
	}

	return result
}

// classifyCell determines the biome for a single cell
func classifyCell(tempC, precipCm, elev, seaLevel float64) Biome {
	// Ocean
	if elev < seaLevel {
		return BiomeOcean
	}

	// High elevation alpine (above treeline, roughly 2500m+ adjusted for latitude)
	// Treeline drops at higher latitudes, but simplified here
	if elev > 3000 {
		if tempC < -10 {
			return BiomeIceCap
		}
		return BiomeAlpine
	}

	// Ice cap: extremely cold
	if tempC < -10 {
		return BiomeIceCap
	}

	// Tundra: cold with permafrost
	// Low evapotranspiration means less water needed for vegetation
	// Real tundra: 15-30cm/year
	if tempC < 0 {
		if precipCm < 15 {
			return BiomeDesertCold // Polar desert
		}
		return BiomeTundra
	}

	// Cold climates (0-10°C) - Boreal zone
	// Low evapotranspiration: boreal forests thrive with 30-50cm/year
	if tempC < 10 {
		if precipCm < 20 {
			return BiomeDesertCold
		}
		if precipCm < 35 {
			return BiomeSemiArid
		}
		if precipCm > 150 {
			return BiomeTemperateRainforest
		}
		return BiomeBorealForest // 35+ cm in cold climate
	}

	// Cool temperate (10-18°C)
	// Widened grassland band: prairies, steppes at 35-80cm
	if tempC < 18 {
		if precipCm < 25 {
			return BiomeDesertCold
		}
		if precipCm < 35 {
			return BiomeSemiArid
		}
		if precipCm < 80 {
			return BiomeTemperateGrassland // 35-80 cm (prairies, steppes)
		}
		if precipCm > 175 {
			return BiomeTemperateRainforest
		}
		return BiomeTemperateForest // 80-175 cm
	}

	// Warm temperate / subtropical (18-24°C)
	// Higher evapotranspiration than cool temperate
	if tempC < 24 {
		if precipCm < 30 {
			return BiomeDesertHot
		}
		if precipCm < 50 {
			return BiomeSemiArid
		}
		if precipCm < 90 {
			return BiomeTemperateGrassland // 50-90 cm (warm prairies)
		}
		if precipCm < 120 {
			return BiomeMediterranean // 90-120 cm (dry subtropical)
		}
		if precipCm > 200 {
			return BiomeTemperateRainforest
		}
		return BiomeTemperateForest // 120-200 cm
	}

	// Tropical (>24°C)
	if precipCm < 25 {
		return BiomeDesertHot
	}
	if precipCm < 50 {
		return BiomeSemiArid
	}
	if precipCm < 100 {
		return BiomeSavanna
	}
	if precipCm < 200 {
		return BiomeTropicalSeasonalForest
	}
	return BiomeTropicalRainforest
}

// GetBiomeStats returns counts of each biome type (land only)
func GetBiomeStats(result *BiomeResult) map[Biome]int {
	counts := make(map[Biome]int)
	for _, b := range result.Biomes {
		if b != BiomeOcean {
			counts[b]++
		}
	}
	return counts
}
