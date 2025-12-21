package terrain

// Hotspot elevation application
// Handles oceanic islands, continental calderas, and underwater continental volcanism

import (
	"math"
	"math/rand"
)

// spreadConfig configures the BFS elevation spreading for hotspot features
type spreadConfig struct {
	ringFractionBase float64 // Base fraction for ring elevation decay (e.g., 0.35-0.60)
	ringFractionVar  float64 // Random variation added to base (e.g., 0.15)
	skipProbability  float64 // Probability of skipping a cell (creates irregular edges)
	minRingElev      float64 // Minimum ring elevation before stopping
	isOceanic        bool    // Whether this is an oceanic hotspot
}

// spreadElevation applies BFS spreading from a center cell outward
// Returns count of modified cells
func spreadElevation(
	elevation []float64,
	cells []VoronoiCell,
	centerIdx int,
	spreadDepth int,
	peakElev float64,
	minElev float64,
	numRegions int,
	cellFilter func(int) bool, // Returns true if cell should be included
	cfg spreadConfig,
	hotspotCells map[int]HotspotCellInfo,
	rng *rand.Rand,
) int {
	if spreadDepth <= 0 {
		return 0
	}

	cellCount := 0
	visited := make(map[int]bool)
	visited[centerIdx] = true
	currentRing := []int{centerIdx}

	for ring := 1; ring <= spreadDepth; ring++ {
		nextRing := []int{}
		ringFraction := cfg.ringFractionBase + cfg.ringFractionVar*rng.Float64()
		ringElev := peakElev * math.Pow(ringFraction, float64(ring))

		if ringElev < cfg.minRingElev {
			break
		}

		for _, idx := range currentRing {
			for _, neighborIdx := range cells[idx].NeighborSiteIndices {
				nIdx := int(neighborIdx)
				if nIdx >= numRegions || visited[nIdx] {
					continue
				}
				visited[nIdx] = true

				if !cellFilter(nIdx) {
					continue
				}

				if rng.Float64() < cfg.skipProbability {
					continue
				}

				if elevation[nIdx] < ringElev {
					elevation[nIdx] = ringElev
					cellCount++
				}

				ringMinElev := minElev * 0.7
				if ringMinElev < 15 {
					ringMinElev = 15
				}
				hotspotCells[nIdx] = HotspotCellInfo{IsOceanic: cfg.isOceanic, MinElevation: ringMinElev}
				nextRing = append(nextRing, nIdx)
			}
		}
		currentRing = nextRing
	}

	return cellCount
}

// ApplyHotspotElevation adds elevation for hotspot features (call AFTER hypsometry)
// Works directly in meters. Handles three cases:
// - Oceanic plates: Hawaii-style island chains
// - Continental plates above water: Yellowstone-style caldera tracks
// - Continental plates underwater: Kerguelen-style volcanic islands (smaller than oceanic)
// Returns the number of cells modified, per-feature cell counts, and map of hotspot cells
func ApplyHotspotElevation(
	elevation []float64,
	cells []VoronoiCell,
	chains []HotspotChain,
	rPlate []int,
	plateIsOcean map[int]bool,
	rng *rand.Rand,
) (int, []int, map[int]HotspotCellInfo) {
	totalCells := 0
	numRegions := len(elevation)
	featureSizes := make([]int, 0)
	hotspotCells := make(map[int]HotspotCellInfo)

	// Calculate average cell angular radius for resolution-independent spreading
	// Total solid angle = 4π, per cell = 4π/n, angular radius ≈ sqrt(4/n)
	cellAngularRadius := 2.0 / math.Sqrt(float64(numRegions))

	for _, chain := range chains {
		var cellCount int
		var sizes []int
		var chainHotspotCells map[int]HotspotCellInfo

		if chain.IsOceanic {
			// True oceanic plate: Hawaii-style volcanic islands
			cellCount, sizes, chainHotspotCells = applyOceanicChainElevation(
				elevation, cells, chain, rPlate, plateIsOcean, cellAngularRadius, numRegions, rng)
		} else {
			// Continental plate: check if underwater or above water
			// Process each island in the chain separately based on local elevation
			cellCount, sizes, chainHotspotCells = applyContinentalChainElevationMixed(
				elevation, cells, chain, rPlate, plateIsOcean, cellAngularRadius, numRegions, rng)
		}

		totalCells += cellCount
		featureSizes = append(featureSizes, sizes...)

		// Merge chain hotspot cells into main map
		for idx, info := range chainHotspotCells {
			hotspotCells[idx] = info
		}
	}

	return totalCells, featureSizes, hotspotCells
}

// applyOceanicChainElevation handles oceanic hotspot island chains (Hawaii-style)
// Island elevation varies with age: emerging -> peak shield volcano -> subsiding atoll
func applyOceanicChainElevation(
	elevation []float64,
	cells []VoronoiCell,
	chain HotspotChain,
	rPlate []int,
	plateIsOcean map[int]bool,
	cellAngularRadius float64,
	numRegions int,
	rng *rand.Rand,
) (int, []int, map[int]HotspotCellInfo) {
	totalCells := 0
	islandSizes := make([]int, 0)
	hotspotCells := make(map[int]HotspotCellInfo)

	// Target island radii in radians (resolution-independent)
	// Hawaii's Big Island is ~120km across, so radius ~60km = 0.010 radians
	// To get spreadDepth=2 reliably at level 7 (cell radius ~0.005), need larger radius
	const (
		maxIslandRadiusLarge  = 0.014 // ~90km radius for largest islands (Hawaii-scale)
		maxIslandRadiusMedium = 0.008 // ~50km radius for medium islands
	)

	// Each chain has its own "vigor" - some hotspots are more active
	chainVigor := 0.5 + rng.Float64() // 0.5x to 1.5x elevation multiplier

	for _, island := range chain.Islands {
		cellIdx := island.CellIndex
		if cellIdx < 0 || cellIdx >= numRegions {
			continue
		}

		// Only place islands on oceanic cells
		if !plateIsOcean[rPlate[cellIdx]] {
			continue
		}

		// Volcanic island lifecycle curve (elevation in meters):
		// Hawaii reference: Mauna Kea 4207m (young, large island), Haleakala 3055m (Maui)
		// Smaller/older islands: 500-1500m, atolls: 50-200m
		//
		// - Age 0: emerging seamount (~300m base)
		// - Age ~0.15: peak shield volcano (~4000m base for large islands)
		// - Age 0.5+: eroded/subsiding atoll (~50m base)
		age := island.Age
		peakAge := 0.15

		// Higher base peak for realistic Hawaii-style volcanism
		basePeak := 3800.0

		var baseElev float64
		if age < peakAge {
			// Rising phase: 300m -> basePeak
			t := age / peakAge
			baseElev = 300 + (basePeak-300)*t*t*(3-2*t)
		} else {
			// Falling phase: exponential decay to atoll level
			// Decay rate of 3.0 means by age 0.5, height is ~15% of peak
			t := (age - peakAge) / (1 - peakAge)
			baseElev = 50 + (basePeak-50)*math.Exp(-3.0*t)
		}

		// Apply chain vigor and per-island random variation
		islandVariation := 0.6 + 0.8*rng.Float64() // 0.6x to 1.4x
		prelimPeakElev := baseElev * chainVigor * islandVariation

		// Clamp preliminary peak for spread calculation
		if prelimPeakElev < 20 {
			prelimPeakElev = 20
		}
		if prelimPeakElev > 4200 {
			prelimPeakElev = 4200
		}

		// Determine island spread based on preliminary peak elevation
		// Calculate target radius in radians, then convert to rings based on cell size
		var targetRadius float64
		if prelimPeakElev > 1500 {
			// Large shield volcano: up to 64km radius
			targetRadius = maxIslandRadiusLarge * (0.7 + 0.3*rng.Float64())
		} else if prelimPeakElev > 1000 {
			// Medium volcano: up to 38km radius
			targetRadius = maxIslandRadiusMedium * (0.7 + 0.3*rng.Float64())
		} else if prelimPeakElev > 500 {
			// Small volcano: minimal spread
			targetRadius = maxIslandRadiusMedium * 0.5 * rng.Float64()
		} else {
			targetRadius = 0
		}

		// Convert target radius to number of rings based on cell size
		spreadDepth := int(targetRadius / cellAngularRadius)
		if spreadDepth > 2 {
			spreadDepth = 2 // Cap at 2 rings max for any resolution
		}

		// Apply size-based cap: small islands can't support Hawaii-height peaks
		// This mimics natural erosion - isolated single-cell islands erode faster
		// spreadDepth 0 (1 cell): max 1500m - small isolated seamount
		// spreadDepth 1 (3-4 cells): max 2500m - medium volcanic island
		// spreadDepth 2 (5-7 cells): max 4000m - large shield volcano complex (Hawaii)
		var sizeCap float64
		switch spreadDepth {
		case 0:
			sizeCap = 1500
		case 1:
			sizeCap = 2500
		default:
			sizeCap = 4000
		}

		peakElev := prelimPeakElev
		if peakElev > sizeCap {
			peakElev = sizeCap
		}

		// Track cells for this island
		islandCellCount := 0

		// Minimum elevation based on age - young islands stay visible, old ones can be atolls
		// Young (age < 0.3): min 80-150m
		// Old (age > 0.6): min 20-50m (low atolls)
		var minElev float64
		if age < 0.3 {
			minElev = 80 + 70*(0.3-age)/0.3
		} else if age < 0.6 {
			minElev = 50 + 30*(0.6-age)/0.3
		} else {
			minElev = 20 + 30*(1.0-age)/0.4
		}

		// Set peak elevation
		if elevation[cellIdx] < peakElev {
			elevation[cellIdx] = peakElev
			totalCells++
			islandCellCount++
		}
		// Track this cell as a hotspot cell
		hotspotCells[cellIdx] = HotspotCellInfo{IsOceanic: true, MinElevation: minElev}

		// Spread to neighbors with decreasing elevation
		spreadCells := spreadElevation(
			elevation, cells, cellIdx, spreadDepth, peakElev, minElev, numRegions,
			func(nIdx int) bool { return plateIsOcean[rPlate[nIdx]] },
			spreadConfig{
				ringFractionBase: 0.35,
				ringFractionVar:  0.15,
				skipProbability:  0.3,
				minRingElev:      30,
				isOceanic:        true,
			},
			hotspotCells, rng,
		)
		totalCells += spreadCells
		islandCellCount += spreadCells

		if islandCellCount > 0 {
			islandSizes = append(islandSizes, islandCellCount)
		}
	}

	return totalCells, islandSizes, hotspotCells
}

// applyContinentalChainElevationMixed handles continental hotspot chains
// Dispatches each island to either land (caldera) or underwater (volcanic island) treatment
func applyContinentalChainElevationMixed(
	elevation []float64,
	cells []VoronoiCell,
	chain HotspotChain,
	rPlate []int,
	plateIsOcean map[int]bool,
	cellAngularRadius float64,
	numRegions int,
	rng *rand.Rand,
) (int, []int, map[int]HotspotCellInfo) {
	totalCells := 0
	featureSizes := make([]int, 0)
	hotspotCells := make(map[int]HotspotCellInfo)

	// Each chain has its own "vigor" - some hotspots are more active
	chainVigor := 0.6 + 0.8*rng.Float64() // 0.6x to 1.4x elevation multiplier

	// First pass: collect all track cell indices for subsidence calculation
	trackCells := make(map[int]bool)
	for _, feature := range chain.Islands {
		trackCells[feature.CellIndex] = true
	}

	for _, feature := range chain.Islands {
		cellIdx := feature.CellIndex
		if cellIdx < 0 || cellIdx >= numRegions {
			continue
		}

		// Skip if on oceanic plate (shouldn't happen for continental chain, but be safe)
		if plateIsOcean[rPlate[cellIdx]] {
			continue
		}

		// Check if this cell is underwater (continental shelf/flooded margin)
		if elevation[cellIdx] <= 0 {
			// Underwater continental: create Kerguelen-style volcanic islands
			cellCount, sizes, cellHotspots := applyUnderwaterContinentalIsland(
				elevation, cells, cellIdx, feature.Age, chainVigor, cellAngularRadius, numRegions, rng)
			totalCells += cellCount
			featureSizes = append(featureSizes, sizes...)
			for idx, info := range cellHotspots {
				hotspotCells[idx] = info
			}
		} else {
			// Above water: Yellowstone-style caldera track
			cellCount, sizes, cellHotspots := applyContinentalCaldera(
				elevation, cells, cellIdx, feature.Age, chainVigor, trackCells, rPlate, plateIsOcean, cellAngularRadius, numRegions, rng)
			totalCells += cellCount
			featureSizes = append(featureSizes, sizes...)
			for idx, info := range cellHotspots {
				hotspotCells[idx] = info
			}
		}
	}

	return totalCells, featureSizes, hotspotCells
}

// applyUnderwaterContinentalIsland creates volcanic islands on submerged continental crust
// Smaller than oceanic hotspots because thicker continental crust limits magma flow
// Reference: Kerguelen Islands (~1850m), Heard Island (~2745m)
func applyUnderwaterContinentalIsland(
	elevation []float64,
	cells []VoronoiCell,
	cellIdx int,
	age float64,
	chainVigor float64,
	cellAngularRadius float64,
	numRegions int,
	rng *rand.Rand,
) (int, []int, map[int]HotspotCellInfo) {
	totalCells := 0
	islandSizes := make([]int, 0)
	hotspotCells := make(map[int]HotspotCellInfo)

	// Target island radii - smaller than oceanic due to thicker crust
	const (
		maxIslandRadiusLarge  = 0.010 // ~64km radius max (smaller than oceanic 0.014)
		maxIslandRadiusMedium = 0.006 // ~38km radius for medium islands
	)

	// Similar lifecycle to oceanic but with lower peak
	// Kerguelen-style: max ~1800m instead of ~4000m for Hawaii
	peakAge := 0.15
	basePeak := UnderwaterContinentalPeakElevation // ~1800m

	var baseElev float64
	if age < peakAge {
		// Rising phase
		t := age / peakAge
		baseElev = 200 + (basePeak-200)*t*t*(3-2*t)
	} else {
		// Falling phase: slower decay than oceanic (thicker crust = more stable)
		t := (age - peakAge) / (1 - peakAge)
		baseElev = 50 + (basePeak-50)*math.Exp(-2.5*t)
	}

	// Apply chain vigor and per-island random variation
	islandVariation := 0.7 + 0.6*rng.Float64() // 0.7x to 1.3x (less variation)
	prelimPeakElev := baseElev * chainVigor * islandVariation

	// Clamp preliminary peak
	if prelimPeakElev < 20 {
		prelimPeakElev = 20
	}
	if prelimPeakElev > UnderwaterContinentalPeakElevation {
		prelimPeakElev = UnderwaterContinentalPeakElevation
	}

	// Determine spread - capped at 1 ring for underwater continental
	var targetRadius float64
	if prelimPeakElev > 1000 {
		targetRadius = maxIslandRadiusLarge * (0.7 + 0.3*rng.Float64())
	} else if prelimPeakElev > 500 {
		targetRadius = maxIslandRadiusMedium * (0.7 + 0.3*rng.Float64())
	} else {
		targetRadius = 0
	}

	spreadDepth := int(targetRadius / cellAngularRadius)
	if spreadDepth > UnderwaterContinentalMaxSpread {
		spreadDepth = UnderwaterContinentalMaxSpread
	}

	// Size-based cap for underwater continental
	var sizeCap float64
	switch spreadDepth {
	case 0:
		sizeCap = 800 // Single cell: small volcanic island
	default:
		sizeCap = UnderwaterContinentalPeakElevation // Multi-cell: up to Kerguelen scale
	}

	peakElev := prelimPeakElev
	if peakElev > sizeCap {
		peakElev = sizeCap
	}

	islandCellCount := 0

	// Minimum elevation based on age
	var minElev float64
	if age < 0.3 {
		minElev = 60 + 50*(0.3-age)/0.3
	} else if age < 0.6 {
		minElev = 40 + 20*(0.6-age)/0.3
	} else {
		minElev = 20 + 20*(1.0-age)/0.4
	}

	// Set peak elevation
	if elevation[cellIdx] < peakElev {
		elevation[cellIdx] = peakElev
		totalCells++
		islandCellCount++
	}
	// Track as continental (even though underwater) for erosion handling
	hotspotCells[cellIdx] = HotspotCellInfo{IsOceanic: false, MinElevation: minElev}

	// Spread to neighbors
	spreadCells := spreadElevation(
		elevation, cells, cellIdx, spreadDepth, peakElev, minElev, numRegions,
		func(nIdx int) bool { return true }, // No filter for underwater continental
		spreadConfig{
			ringFractionBase: 0.40,
			ringFractionVar:  0.15,
			skipProbability:  0.25,
			minRingElev:      30,
			isOceanic:        false,
		},
		hotspotCells, rng,
	)
	totalCells += spreadCells
	islandCellCount += spreadCells

	if islandCellCount > 0 {
		islandSizes = append(islandSizes, islandCellCount)
	}

	return totalCells, islandSizes, hotspotCells
}

// applyContinentalCaldera handles Yellowstone-style caldera tracks on land
func applyContinentalCaldera(
	elevation []float64,
	cells []VoronoiCell,
	cellIdx int,
	age float64,
	chainVigor float64,
	trackCells map[int]bool,
	rPlate []int,
	plateIsOcean map[int]bool,
	cellAngularRadius float64,
	numRegions int,
	rng *rand.Rand,
) (int, []int, map[int]HotspotCellInfo) {
	totalCells := 0
	calderaSizes := make([]int, 0)
	hotspotCells := make(map[int]HotspotCellInfo)

	calderaCellCount := 0

	// Continental hotspot lifecycle:
	// - Age 0-0.12: Building up to peak caldera (uplift phase)
	// - Age 0.12-0.35: Active caldera plateau (peak phase)
	// - Age 0.35+: Subsided plain (thermal contraction)

	if age < ContinentalSubsidenceAge {
		// Active or recently active: elevated plateau
		var plateauElev float64
		if age < ContinentalPeakAge {
			// Rising phase: building toward peak
			t := age / ContinentalPeakAge
			plateauElev = 500 + (ContinentalPeakElevation-500)*t*t*(3-2*t)
		} else {
			// Plateau phase: near peak, slight decay
			t := (age - ContinentalPeakAge) / (ContinentalSubsidenceAge - ContinentalPeakAge)
			plateauElev = ContinentalPeakElevation * (1.0 - 0.15*t)
		}

		// Apply chain vigor and random variation
		featureVariation := 0.8 + 0.4*rng.Float64()
		plateauElev *= chainVigor * featureVariation

		// Clamp to reasonable range
		if plateauElev > 3500 {
			plateauElev = 3500
		}

		// Larger footprint for continental calderas (resolution-independent)
		targetRadius := ContinentalCalderaRadius * (0.7 + 0.3*rng.Float64())
		spreadDepth := int(targetRadius / cellAngularRadius)
		if spreadDepth > 3 {
			spreadDepth = 3 // Cap at 3 rings for calderas
		}

		// Set peak elevation
		if elevation[cellIdx] < plateauElev {
			elevation[cellIdx] = plateauElev
			totalCells++
			calderaCellCount++
		}
		// Track this cell - continental features have higher minimum (on land)
		// Young features: min 400-800m, older features: min 200-400m
		minElev := 400.0 + 400.0*(ContinentalSubsidenceAge-age)/ContinentalSubsidenceAge
		hotspotCells[cellIdx] = HotspotCellInfo{IsOceanic: false, MinElevation: minElev}

		// Spread plateau to neighbors (flatter profile than shield volcanoes)
		spreadCells := spreadElevation(
			elevation, cells, cellIdx, spreadDepth, plateauElev, minElev, numRegions,
			func(nIdx int) bool { return !plateIsOcean[rPlate[nIdx]] && elevation[nIdx] > 0 },
			spreadConfig{
				ringFractionBase: 0.60,
				ringFractionVar:  0.15,
				skipProbability:  0.2,
				minRingElev:      0, // No minimum for continental plateaus
				isOceanic:        false,
			},
			hotspotCells, rng,
		)
		totalCells += spreadCells
		calderaCellCount += spreadCells
	} else {
		// Old track: apply subsidence (depression below neighbor average)
		// Calculate average elevation of neighbors NOT in the hotspot track
		neighborSum := 0.0
		neighborCount := 0
		for _, neighborIdx := range cells[cellIdx].NeighborSiteIndices {
			nIdx := int(neighborIdx)
			if nIdx >= numRegions {
				continue
			}
			// Only count neighbors not in the track and above water
			if !trackCells[nIdx] && !plateIsOcean[rPlate[nIdx]] && elevation[nIdx] > 0 {
				neighborSum += elevation[nIdx]
				neighborCount++
			}
		}

		if neighborCount > 0 {
			neighborAvg := neighborSum / float64(neighborCount)
			// Subsidence increases with age (older = more subsided)
			ageFactorBeyondSubsidence := (age - ContinentalSubsidenceAge) / (1.0 - ContinentalSubsidenceAge)
			subsidenceAmount := ContinentalSubsidence * (0.5 + 0.5*ageFactorBeyondSubsidence)
			subsidenceAmount *= (0.7 + 0.6*rng.Float64()) // Random variation

			subsidedElev := neighborAvg - subsidenceAmount
			// Don't go below sea level for continental track
			if subsidedElev < 100 {
				subsidedElev = 100 + 200*rng.Float64()
			}

			// Only apply if it lowers the elevation
			if elevation[cellIdx] > subsidedElev {
				elevation[cellIdx] = subsidedElev
				totalCells++
				calderaCellCount++
			}
			// Track subsided cells - low minimum since this is old subsided track
			hotspotCells[cellIdx] = HotspotCellInfo{IsOceanic: false, MinElevation: 100}
		}
	}

	if calderaCellCount > 0 {
		calderaSizes = append(calderaSizes, calderaCellCount)
	}

	return totalCells, calderaSizes, hotspotCells
}
