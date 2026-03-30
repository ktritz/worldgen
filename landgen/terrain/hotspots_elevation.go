package terrain

// Hotspot elevation application
// Handles oceanic islands, continental calderas, and underwater continental volcanism

import (
	"math"
	"math/rand"
)

// spreadOceanicElevation creates smooth volcanic shield slopes for oceanic islands
// Uses DIRECT RADIAL SELECTION instead of BFS to completely avoid hexagonal grid artifacts
// Each cell is evaluated based on its actual spherical distance and angle from center
func spreadOceanicElevation(
	elevation []float64,
	cells []VoronoiCell,
	sites []Vector3D,
	centerIdx int,
	targetRadius float64,
	cellAngularRadius float64,
	peakElev float64,
	baseOceanElev float64,
	minElev float64,
	numRegions int,
	cellFilter func(int) bool,
	hotspotCells map[int]HotspotCellInfo,
	rng *rand.Rand,
) int {
	if targetRadius <= 0 {
		return 0
	}

	_ = cells             // Not used in radial selection (no BFS)
	_ = cellAngularRadius // Not used directly

	cellCount := 0
	centerPos := sites[centerIdx].Normalize()
	peakHeight := peakElev - baseOceanElev

	// Create local tangent-plane coordinate frame
	up := Vector3D{0, 0, 1}
	if math.Abs(centerPos.Z) > 0.99 {
		up = Vector3D{1, 0, 0}
	}
	east := centerPos.Cross(up).Normalize()
	north := east.Cross(centerPos).Normalize()

	// Generate smooth angular radius variation for organic island shape
	// Low-frequency lobes create smooth elongation
	lobePhase := float64(uint32(centerIdx*7919)%1000) / 1000.0 * 2 * math.Pi
	numLobes := 2 + int(uint32(centerIdx*31337)%2)
	lobeAmps := make([]float64, numLobes)
	for i := range lobeAmps {
		h := uint32(centerIdx*104729 + i*7919)
		h ^= h >> 16
		h *= 0x85ebca6b
		lobeAmps[i] = 0.15 + 0.25*float64(h%1000)/1000.0
	}

	// Small high-frequency noise for subtle edge variation
	highFreqPhase := float64(uint32(centerIdx*24681)%1000) / 1000.0 * 2 * math.Pi
	highFreqAmp := 0.04 + 0.04*float64(uint32(centerIdx*54321)%1000)/1000.0

	// Per-island random seed for consistent variation
	islandRng := rand.New(rand.NewSource(int64(centerIdx * 12345)))

	// Function to get radius multiplier for a given angle
	getRadiusForAngle := func(angle float64) float64 {
		radius := 1.0
		for i, amp := range lobeAmps {
			freq := float64(i + 1)
			radius += amp * math.Sin(freq*angle+lobePhase+float64(i)*0.7)
		}
		radius += highFreqAmp * math.Sin(5*angle+highFreqPhase)
		return radius
	}

	// Maximum search radius with margin
	maxSearchRadius := targetRadius * 1.8

	// DIRECT RADIAL SELECTION: Check all cells within range
	for idx := 0; idx < numRegions; idx++ {
		if idx == centerIdx {
			continue
		}
		if !cellFilter(idx) {
			continue
		}

		// Calculate actual spherical distance
		cellPos := sites[idx]
		dot := centerPos.X*cellPos.X + centerPos.Y*cellPos.Y + centerPos.Z*cellPos.Z
		if dot > 1.0 {
			dot = 1.0
		}
		if dot < -1.0 {
			dot = -1.0
		}
		actualDist := math.Acos(dot)

		// Quick reject
		if actualDist > maxSearchRadius {
			continue
		}

		// Calculate angle on tangent plane
		dirFromCenter := cellPos.Subtract(centerPos)
		eastComp := dirFromCenter.Dot(east)
		northComp := dirFromCenter.Dot(north)
		angle := math.Atan2(northComp, eastComp)

		// Get noise-modulated radius for this angle
		effectiveRadius := targetRadius * getRadiusForAngle(angle)

		// Add small per-cell random variation for natural coastline
		effectiveRadius *= 0.92 + 0.16*islandRng.Float64()

		if actualDist > effectiveRadius {
			continue
		}

		// Normalized distance for elevation profile
		normalizedDist := actualDist / effectiveRadius
		if normalizedDist > 1.0 {
			normalizedDist = 1.0
		}

		// Smooth volcanic shield profile (squared cosine)
		t := normalizedDist
		profileFactor := math.Cos(t * math.Pi / 2)
		profileFactor = profileFactor * profileFactor

		// Calculate elevation
		localOceanFloor := elevation[idx]
		if localOceanFloor > 0 {
			localOceanFloor = baseOceanElev
		}
		targetElev := localOceanFloor + peakHeight*profileFactor

		// Apply minimum elevation
		ringMinElev := minElev * (0.3 + 0.7*profileFactor)
		if ringMinElev < 15 {
			ringMinElev = 15
		}
		if targetElev < ringMinElev {
			targetElev = ringMinElev
		}

		// Only raise elevation
		if targetElev > elevation[idx] {
			elevation[idx] = targetElev
			cellCount++
		}

		hotspotCells[idx] = HotspotCellInfo{IsOceanic: true, MinElevation: ringMinElev}
	}

	return cellCount
}

// spreadAdditiveElevation applies additive elevation boost with gaussian-like falloff
// Instead of setting absolute elevation, it ADDS boost to existing terrain
// Resolution-independent: normalizes distance to 0-1 range based on spreadDepth
func spreadAdditiveElevation(
	elevation []float64,
	cells []VoronoiCell,
	centerIdx int,
	spreadDepth int,
	centerBoost float64,
	numRegions int,
	cellFilter func(int) bool,
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
		// Normalize ring to 0-1 range for resolution-independent falloff
		normalizedDist := float64(ring) / float64(spreadDepth)
		// Gaussian-like falloff: at edge (dist=1), fraction is ~0.22
		ringFraction := math.Exp(-normalizedDist * 1.5)
		ringBoost := centerBoost * ringFraction

		if ringBoost < 50 {
			break // Stop when boost becomes negligible
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

				// Small random variation for natural look
				actualBoost := ringBoost * (0.8 + 0.4*rng.Float64())

				// Add boost to existing elevation
				elevation[nIdx] += actualBoost
				cellCount++

				// Track with minimum based on boost
				hotspotCells[nIdx] = HotspotCellInfo{
					IsOceanic:    false,
					MinElevation: elevation[nIdx] - actualBoost*0.5, // Allow some erosion
				}
				nextRing = append(nextRing, nIdx)
			}
		}
		currentRing = nextRing
	}

	return cellCount
}

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
	sites []Vector3D,
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
				elevation, cells, sites, chain, rPlate, plateIsOcean, cellAngularRadius, numRegions, rng)
		} else {
			// Continental plate: check if underwater or above water
			// Process each island in the chain separately based on local elevation
			cellCount, sizes, chainHotspotCells = applyContinentalChainElevationMixed(
				elevation, cells, sites, chain, rPlate, plateIsOcean, cellAngularRadius, numRegions, rng)
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
	sites []Vector3D,
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
		strength := island.Strength
		if strength <= 0 {
			strength = 1
		}
		strength = Clamp(strength, 0.35, 1.55)
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
		prelimPeakElev := baseElev * chainVigor * islandVariation * strength

		// Clamp preliminary peak for spread calculation
		if prelimPeakElev < 20 {
			prelimPeakElev = 20
		}
		if prelimPeakElev > 4200 {
			prelimPeakElev = 4200
		}

		// Determine island spread using CONTINUOUS function based on elevation AND age
		// This avoids discrete size buckets that cause bifurcation
		// Real volcanic islands have smooth size variation based on age/activity
		//
		// Reference sizes (radius in radians, 0.01 rad ≈ 64km):
		// - Big Island of Hawaii (young, massive): ~75km radius = 0.012 rad
		// - Maui (medium): ~60km radius = 0.009 rad
		// - Oahu (older): ~30km radius = 0.005 rad
		// - Small atolls: ~5km radius = 0.0008 rad
		//
		// Two effects reduce island size with age:
		// 1. Lower elevation → smaller spread (via sqrt scaling)
		// 2. Direct erosion of land area → additional radius reduction
		//
		// Hawaiian example progression:
		// - age 0.15 (peak): full size
		// - age 0.5: ~60% of peak radius (eroded)
		// - age 0.8: ~30% of peak radius (mostly eroded)
		// - age 0.95+: tiny atoll remnant
		var targetRadius float64
		if prelimPeakElev > 100 {
			// Continuous scaling: sqrt gives natural volcanic profile
			elevFactor := math.Sqrt(prelimPeakElev / 200.0)
			if elevFactor > 4.5 {
				elevFactor = 4.5
			}
			// Scale from tiny (0.001 rad = 6km) to large (0.014 rad = 90km)
			targetRadius = 0.001 + (maxIslandRadiusLarge-0.001)*(elevFactor/4.5)

			// Age-based erosion of land AREA (separate from height reduction)
			// Older islands physically shrink due to wave erosion, subsidence
			// Young islands (age < 0.2): full size
			// Old islands (age > 0.7): significantly reduced
			var ageSizeFactor float64
			if age < 0.2 {
				ageSizeFactor = 1.0
			} else if age < 0.5 {
				// Gradual reduction: 100% → 70%
				ageSizeFactor = 1.0 - 0.3*(age-0.2)/0.3
			} else if age < 0.8 {
				// Faster reduction: 70% → 30%
				ageSizeFactor = 0.7 - 0.4*(age-0.5)/0.3
			} else {
				// Atoll stage: 30% → 10%
				ageSizeFactor = 0.3 - 0.2*(age-0.8)/0.2
				if ageSizeFactor < 0.1 {
					ageSizeFactor = 0.1
				}
			}
			targetRadius *= ageSizeFactor

			// Add per-island random variation (±25%)
			targetRadius *= 0.75 + 0.5*rng.Float64()
			targetRadius *= math.Sqrt(strength)
		} else {
			targetRadius = 0
		}

		// Size-based peak cap using CONTINUOUS function based on targetRadius
		// Smaller islands erode faster and can't maintain extreme heights
		// Use smooth curve instead of discrete buckets
		// normalizedRadius = targetRadius / cellAngularRadius (continuous)
		normalizedRadius := targetRadius / cellAngularRadius
		// At radius 0: max ~800m (tiny seamount)
		// At radius 1: max ~1650m (small island)
		// At radius 2: max ~2500m (medium island)
		// At radius 3: max ~3350m (large island)
		// At radius 4+: max ~4200m (massive shield volcano)
		sizeCap := 800.0 + 850.0*math.Min(normalizedRadius, 4.0)

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

		// Get base ocean floor elevation for blending
		baseOceanElev := elevation[cellIdx]

		// Set peak elevation with smooth blend from ocean floor
		if peakElev > elevation[cellIdx] {
			elevation[cellIdx] = peakElev
			totalCells++
			islandCellCount++
		}
		// Track this cell as a hotspot cell
		hotspotCells[cellIdx] = HotspotCellInfo{IsOceanic: true, MinElevation: minElev}

		// Spread to neighbors with smooth gaussian-like falloff
		// This creates gradual volcanic shield slopes instead of sharp edges
		// Pass continuous targetRadius for smooth size variation (not discrete spreadDepth)
		spreadCells := spreadOceanicElevation(
			elevation, cells, sites, cellIdx, targetRadius, cellAngularRadius,
			peakElev, baseOceanElev, minElev, numRegions,
			func(nIdx int) bool { return plateIsOcean[rPlate[nIdx]] },
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
	sites []Vector3D,
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
				elevation, cells, sites, cellIdx, feature.Age, feature.Strength, chainVigor, cellAngularRadius, numRegions, rng)
			totalCells += cellCount
			featureSizes = append(featureSizes, sizes...)
			for idx, info := range cellHotspots {
				hotspotCells[idx] = info
			}
		} else {
			// Above water: Yellowstone-style caldera track
			cellCount, sizes, cellHotspots := applyContinentalCaldera(
				elevation, cells, cellIdx, feature.Age, feature.Strength, chainVigor, trackCells, rPlate, plateIsOcean, cellAngularRadius, numRegions, rng)
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
// Now uses same continuous sizing approach as oceanic islands
func applyUnderwaterContinentalIsland(
	elevation []float64,
	cells []VoronoiCell,
	sites []Vector3D,
	cellIdx int,
	age float64,
	strength float64,
	chainVigor float64,
	cellAngularRadius float64,
	numRegions int,
	rng *rand.Rand,
) (int, []int, map[int]HotspotCellInfo) {
	totalCells := 0
	islandSizes := make([]int, 0)
	hotspotCells := make(map[int]HotspotCellInfo)

	// Max radius smaller than oceanic due to thicker continental crust limiting magma
	// Kerguelen is ~50km across, so max radius ~25km = 0.004 rad
	const maxIslandRadius = 0.008 // ~50km radius max (vs 0.014 for oceanic)

	// Similar lifecycle to oceanic but with lower peak and slower decay
	// Kerguelen-style: max ~1800m instead of ~4000m for Hawaii
	peakAge := 0.15
	basePeak := UnderwaterContinentalPeakElevation // ~1800m
	if strength <= 0 {
		strength = 1
	}
	strength = Clamp(strength, 0.35, 1.55)

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
	islandVariation := 0.7 + 0.6*rng.Float64() // 0.7x to 1.3x
	prelimPeakElev := baseElev * chainVigor * islandVariation * strength

	// Clamp preliminary peak
	if prelimPeakElev < 20 {
		prelimPeakElev = 20
	}
	if prelimPeakElev > UnderwaterContinentalPeakElevation {
		prelimPeakElev = UnderwaterContinentalPeakElevation
	}

	// CONTINUOUS radius scaling (same approach as oceanic, but smaller scale)
	var targetRadius float64
	if prelimPeakElev > 100 {
		// Continuous scaling using sqrt for natural volcanic profile
		elevFactor := math.Sqrt(prelimPeakElev / 200.0)
		if elevFactor > 3.0 {
			elevFactor = 3.0 // Lower cap than oceanic
		}
		targetRadius = 0.001 + (maxIslandRadius-0.001)*(elevFactor/3.0)

		// Age-based area erosion (same as oceanic)
		var ageSizeFactor float64
		if age < 0.2 {
			ageSizeFactor = 1.0
		} else if age < 0.5 {
			ageSizeFactor = 1.0 - 0.3*(age-0.2)/0.3
		} else if age < 0.8 {
			ageSizeFactor = 0.7 - 0.4*(age-0.5)/0.3
		} else {
			ageSizeFactor = 0.3 - 0.2*(age-0.8)/0.2
			if ageSizeFactor < 0.1 {
				ageSizeFactor = 0.1
			}
		}
		targetRadius *= ageSizeFactor

		// Per-island random variation
		targetRadius *= 0.75 + 0.5*rng.Float64()
		targetRadius *= math.Sqrt(strength)
	}

	// Continuous size cap based on radius
	normalizedRadius := targetRadius / cellAngularRadius
	sizeCap := 600.0 + 400.0*math.Min(normalizedRadius, 3.0) // Max ~1800m

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

	// Get base ocean floor for blending
	baseOceanElev := elevation[cellIdx]

	// Set peak elevation
	if peakElev > elevation[cellIdx] {
		elevation[cellIdx] = peakElev
		totalCells++
		islandCellCount++
	}
	hotspotCells[cellIdx] = HotspotCellInfo{IsOceanic: false, MinElevation: minElev}

	// Use same spread function as oceanic for consistent irregular shapes
	spreadCells := spreadOceanicElevation(
		elevation, cells, sites, cellIdx, targetRadius, cellAngularRadius,
		peakElev, baseOceanElev, minElev, numRegions,
		func(nIdx int) bool { return true }, // No plate filter for underwater continental
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
// Uses ADDITIVE elevation boost blended with surrounding terrain
func applyContinentalCaldera(
	elevation []float64,
	cells []VoronoiCell,
	cellIdx int,
	age float64,
	strength float64,
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
	if strength <= 0 {
		strength = 1
	}
	strength = Clamp(strength, 0.35, 1.55)

	// Continental hotspot lifecycle:
	// - Age 0-0.12: Building up to peak caldera (uplift phase)
	// - Age 0.12-0.35: Active caldera plateau (peak phase)
	// - Age 0.35+: Subsided plain (thermal contraction)

	if age < ContinentalSubsidenceAge {
		// Active or recently active: add elevation BOOST to existing terrain
		// This creates gentle volcanic plateaus that blend with surroundings
		var boostElev float64
		if age < ContinentalPeakAge {
			// Rising phase: building toward peak boost
			t := age / ContinentalPeakAge
			// Max boost of ~800m for young active caldera
			boostElev = 100 + 700*t*t*(3-2*t)
		} else {
			// Plateau phase: near peak, slight decay
			t := (age - ContinentalPeakAge) / (ContinentalSubsidenceAge - ContinentalPeakAge)
			boostElev = 800 * (1.0 - 0.3*t)
		}

		// Apply chain vigor and random variation
		featureVariation := 0.7 + 0.6*rng.Float64()
		boostElev *= chainVigor * featureVariation
		boostElev *= 0.70 + 0.30*strength

		// Clamp boost to reasonable range (400-1000m above terrain)
		if boostElev > 1000 {
			boostElev = 1000
		}
		if boostElev < 200 {
			boostElev = 200
		}

		// Smaller footprint for subtler continental features
		targetRadius := ContinentalCalderaRadius * 0.5 * (0.7 + 0.3*rng.Float64())
		targetRadius *= math.Sqrt(0.70 + 0.30*strength)
		spreadDepth := int(targetRadius / cellAngularRadius)
		if spreadDepth > 2 {
			spreadDepth = 2 // Cap at 2 rings for subtler calderas
		}

		// Calculate average neighbor elevation for blending
		neighborSum := 0.0
		neighborCount := 0
		for _, neighborIdx := range cells[cellIdx].NeighborSiteIndices {
			nIdx := int(neighborIdx)
			if nIdx < numRegions && !plateIsOcean[rPlate[nIdx]] && elevation[nIdx] > 0 {
				neighborSum += elevation[nIdx]
				neighborCount++
			}
		}
		baseElev := elevation[cellIdx]
		if neighborCount > 0 {
			// Blend current elevation with neighbor average for smoother result
			neighborAvg := neighborSum / float64(neighborCount)
			baseElev = 0.7*elevation[cellIdx] + 0.3*neighborAvg
		}

		// Add boost to base elevation
		newElev := baseElev + boostElev
		if newElev > elevation[cellIdx] {
			elevation[cellIdx] = newElev
			totalCells++
			calderaCellCount++
		}

		// Track this cell with moderate minimum
		minElev := baseElev + boostElev*0.3
		hotspotCells[cellIdx] = HotspotCellInfo{IsOceanic: false, MinElevation: minElev}

		// Spread boost to neighbors with smooth gaussian-like falloff
		spreadCells := spreadAdditiveElevation(
			elevation, cells, cellIdx, spreadDepth, boostElev, numRegions,
			func(nIdx int) bool { return !plateIsOcean[rPlate[nIdx]] && elevation[nIdx] > 0 },
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
