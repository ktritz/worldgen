package terrain

// Plate generation and assignment for tectonic simulation
// Uses weighted BFS for power-law size distribution

import (
	"math"
	"math/rand"
	"sort"

	"worldgen/icosphere"
)

const (
	plateLayoutSearchAttempts       = 12
	plateLayoutSearchExtraAttempts  = 28
	plateLayoutReferenceSubdivision = 6
)

type plateLayout struct {
	plateR         []int
	plateCenters   []Vector3D
	plateWeights   []float64
	rPlate         []int
	plateSizes     map[int]int
	sortedPlates   []PlateSize
	plateNeighbors map[int]map[int]bool
	plateIsOcean   map[int]bool
	attempt        int
}

type plateBlueprint struct {
	centers     []Vector3D
	weights     []float64
	continental []bool
	attempt     int
}

// GeneratePlates creates plates using weighted BFS for power law size distribution
// Returns: plateR (list of plate center region indices), rPlate (region -> plate center mapping)
func GeneratePlates(sites []Vector3D, cells []VoronoiCell, numPlates int, rng *rand.Rand) ([]int, []int) {
	plateR, plateCenters, plateWeights := generatePlateSeeds(sites, numPlates, rng)
	plateWeight := make(map[int]float64, len(plateR))
	for i, centerR := range plateR {
		plateWeight[centerR] = plateWeights[i]
	}

	return plateR, assignPlateRegionsByWeightedDistance(sites, plateR, plateCenters, plateWeight)
}

func generatePlateSeeds(sites []Vector3D, numPlates int, rng *rand.Rand) ([]int, []Vector3D, []float64) {
	// Pick well-spaced physical points first, then map them to mesh regions.
	// Sampling region indices directly makes the same seed choose different
	// geography at different mesh levels.
	plateR, plateCenters := pickSpacedPlateCenters(sites, numPlates, rng)

	// Assign power law growth weights to each plate
	// Higher weight = grows faster = ends up larger.
	// Use a bounded skew so we still get a few large plates without a single plate
	// swallowing most of the sphere.
	plateWeights := make([]float64, len(plateR))
	for i := range plateR {
		u := rng.Float64()
		plateWeights[i] = 0.8 + 3.2*math.Pow(u, 3.0)
	}

	return plateR, plateCenters, plateWeights
}

func pickSpacedPlateCenters(sites []Vector3D, n int, rng *rand.Rand) ([]int, []Vector3D) {
	numRegions := len(sites)
	if n <= 0 || numRegions == 0 {
		return nil, nil
	}

	chosen := make(map[int]bool, n)
	plateRegions := make([]int, 0, n)
	plateCenters := make([]Vector3D, 0, n)
	firstTarget := randomUnitVector(rng)
	first := nearestRegionToVector(sites, firstTarget, chosen)
	chosen[first] = true
	plateRegions = append(plateRegions, first)
	plateCenters = append(plateCenters, firstTarget)

	sampleCount := 64

	for len(plateRegions) < n && len(plateRegions) < numRegions {
		bestTarget := Vector3D{}
		bestDistance := -1.0

		for sample := 0; sample < sampleCount; sample++ {
			candidate := randomUnitVector(rng)
			minDistance := math.Inf(1)
			for _, existing := range plateCenters {
				distance := angularDistance(candidate, existing)
				if distance < minDistance {
					minDistance = distance
				}
			}

			if minDistance > bestDistance {
				bestDistance = minDistance
				bestTarget = candidate
			}
		}

		bestIdx := nearestRegionToVector(sites, bestTarget, chosen)
		if bestIdx == -1 {
			break
		}

		chosen[bestIdx] = true
		plateRegions = append(plateRegions, bestIdx)
		plateCenters = append(plateCenters, bestTarget)
	}

	return plateRegions, plateCenters
}

func nearestRegionToVector(sites []Vector3D, target Vector3D, excluded map[int]bool) int {
	bestIdx := -1
	bestDot := math.Inf(-1)
	target = target.Normalize()
	for i, site := range sites {
		if excluded != nil && excluded[i] {
			continue
		}
		dot := site.Normalize().Dot(target)
		if dot > bestDot {
			bestDot = dot
			bestIdx = i
		}
	}
	return bestIdx
}

func assignPlateRegionsByWeightedDistance(
	sites []Vector3D,
	plateR []int,
	plateCenters []Vector3D,
	plateWeight map[int]float64,
) []int {
	rPlate := make([]int, len(sites))
	if len(plateR) == 0 {
		for i := range rPlate {
			rPlate[i] = -1
		}
		return rPlate
	}
	for i, site := range sites {
		bestPlate := plateR[0]
		bestScore := math.Inf(1)
		for p, center := range plateR {
			target := plateCenters[p]
			weight := plateWeight[center]
			if weight <= 0 {
				weight = 1
			}
			score := angularDistance(site, target) / weight
			if score < bestScore {
				bestScore = score
				bestPlate = center
			}
		}
		rPlate[i] = bestPlate
	}
	return rPlate
}

// SmoothPlateBoundaries eliminates single-cell plate protrusions
// This creates smoother plate boundaries while preserving coastline irregularity
// (coastlines depend on elevation, not plate assignment)
// Uses majority-neighbor voting: if >50% of neighbors are a different plate, reassign
func SmoothPlateBoundaries(cells []VoronoiCell, rPlate []int, iterations int) int {
	numRegions := len(rPlate)
	totalChanges := 0

	for iter := 0; iter < iterations; iter++ {
		changes := 0
		for r := 0; r < numRegions; r++ {
			currentPlate := rPlate[r]

			// Count neighbors by plate
			neighborCounts := make(map[int]int)
			totalNeighbors := 0
			for _, neighborIdx := range cells[r].NeighborSiteIndices {
				nIdx := int(neighborIdx)
				if nIdx < numRegions {
					neighborCounts[rPlate[nIdx]]++
					totalNeighbors++
				}
			}

			// Find if any other plate has majority of neighbors
			for plate, count := range neighborCounts {
				// Reassign if >50% neighbors are a different plate
				if plate != currentPlate && count*2 > totalNeighbors {
					rPlate[r] = plate
					changes++
					break
				}
			}
		}

		totalChanges += changes
		if changes == 0 {
			break // Converged
		}
	}

	return totalChanges
}

// AssignPlateTypes assigns plates as oceanic or continental
// Uses BFS growth from 2-3 seeds to create multiple continents
func AssignPlateTypes(
	sortedPlates []PlateSize,
	plateSizes map[int]int,
	plateNeighbors map[int]map[int]bool,
	totalRegions int,
	targetLandFraction float64,
) map[int]bool {
	plateIsOcean := make(map[int]bool)

	// Start with all plates as oceanic
	for _, ps := range sortedPlates {
		plateIsOcean[ps.Center] = true
	}

	bestAssignment, ok := findBestPlateTypeAssignment(sortedPlates, plateSizes, plateNeighbors, totalRegions, targetLandFraction)
	if ok {
		return bestAssignment
	}

	// Fallback for larger plate counts where exhaustive search is too expensive.
	targetContinentalRegions := int(float64(totalRegions) * targetContinentalFraction(targetLandFraction))
	continentalRegions := 0
	targetLargestContinent := int(float64(targetContinentalRegions) * 0.55)
	if targetLargestContinent < 1 {
		targetLargestContinent = 1
	}

	for i, ps := range sortedPlates {
		if i == 0 {
			continue // keep the largest plate oceanic to preserve a major ocean basin
		}
		if continentalRegions >= targetContinentalRegions {
			break
		}
		if ps.Size > targetLargestContinent {
			continue
		}
		plateIsOcean[ps.Center] = false
		continentalRegions += ps.Size
	}

	return plateIsOcean
}

func generatePlateLayout(
	sites []Vector3D,
	cells []VoronoiCell,
	numPlates int,
	rng *rand.Rand,
	attempt int,
) plateLayout {
	plateR, plateCenters, plateWeights := generatePlateSeeds(sites, numPlates, rng)
	plateWeight := make(map[int]float64, len(plateR))
	for i, centerR := range plateR {
		plateWeight[centerR] = plateWeights[i]
	}
	rPlate := assignPlateRegionsByWeightedDistance(sites, plateR, plateCenters, plateWeight)
	SmoothPlateBoundaries(cells, rPlate, 3)

	plateSizes := make(map[int]int)
	for _, plate := range rPlate {
		plateSizes[plate]++
	}

	activePlateR := make([]int, 0, len(plateR))
	activePlateCenters := make([]Vector3D, 0, len(plateR))
	activePlateWeights := make([]float64, 0, len(plateR))
	for i, center := range plateR {
		if plateSizes[center] > 0 {
			activePlateR = append(activePlateR, center)
			activePlateCenters = append(activePlateCenters, plateCenters[i])
			activePlateWeights = append(activePlateWeights, plateWeights[i])
		}
	}

	sortedPlates := make([]PlateSize, 0, len(activePlateR))
	for _, centerR := range activePlateR {
		sortedPlates = append(sortedPlates, PlateSize{
			Center: centerR,
			Size:   plateSizes[centerR],
		})
	}
	sortPlateSizes(sortedPlates)

	return plateLayout{
		plateR:         activePlateR,
		plateCenters:   activePlateCenters,
		plateWeights:   activePlateWeights,
		rPlate:         rPlate,
		plateSizes:     plateSizes,
		sortedPlates:   sortedPlates,
		plateNeighbors: FindPlateNeighbors(cells, rPlate, activePlateR),
		attempt:        attempt,
	}
}

func GenerateOptimizedPlateLayout(
	sites []Vector3D,
	cells []VoronoiCell,
	numPlates int,
	seed int64,
	targetLandFraction float64,
) plateLayout {
	if blueprint, ok := buildPlateBlueprint(numPlates, seed, targetLandFraction); ok {
		return buildPlateLayoutFromBlueprint(sites, cells, blueprint)
	}

	// Fallback preserves deterministic terrain generation if reference blueprint
	// creation ever fails.
	rng := rand.New(rand.NewSource(seed))
	return generatePlateLayout(sites, cells, numPlates, rng, 0)
}

func buildPlateBlueprint(
	numPlates int,
	seed int64,
	targetLandFraction float64,
) (plateBlueprint, bool) {
	refSites, refFaces := icosphere.CreateIcosphere(plateLayoutReferenceSubdivision)
	_, refCells := icosphere.GenerateSphericalVoronoi(refSites, refFaces)
	return buildPlateBlueprintFromReference(refSites, refCells, numPlates, seed, targetLandFraction)
}

func buildPlateBlueprintFromReference(
	refSites []Vector3D,
	refCells []VoronoiCell,
	numPlates int,
	seed int64,
	targetLandFraction float64,
) (plateBlueprint, bool) {
	bestScore := math.Inf(1)
	var bestLayout plateLayout
	var bestCandidate plateAssignmentCandidate
	found := false

	maxAttempts := plateLayoutSearchAttempts + plateLayoutSearchExtraAttempts
	for attempt := 0; attempt < maxAttempts; attempt++ {
		rng := rand.New(rand.NewSource(seed + int64(attempt)*7919))
		layout := generatePlateLayout(refSites, refCells, numPlates, rng, attempt)
		candidate, ok := findBestPlateTypeAssignmentCandidate(
			layout.sortedPlates,
			layout.plateSizes,
			layout.plateNeighbors,
			len(refSites),
			targetLandFraction,
		)
		if !ok {
			continue
		}
		if !found || candidate.score < bestScore {
			bestScore = candidate.score
			bestLayout = layout
			bestCandidate = candidate
			found = true
		}
		// After the baseline search budget, stop once the layout is broadly
		// plausible. We do not need every seed to converge to an Earth-like
		// continent histogram if the tectonic structure still makes sense.
		if attempt+1 >= plateLayoutSearchAttempts && found &&
			bestCandidate.stats.majorContinents >= 3 &&
			bestCandidate.stats.largestContinentShare <= 0.48 &&
			bestCandidate.stats.continentSizeGini >= 0.14 {
			break
		}
	}

	if found {
		return blueprintFromLayoutCandidate(bestLayout, bestCandidate), true
	}

	// Fallback to the first deterministic layout if scoring failed.
	rng := rand.New(rand.NewSource(seed))
	layout := generatePlateLayout(refSites, refCells, numPlates, rng, 0)
	plateIsOcean := AssignPlateTypes(
		layout.sortedPlates,
		layout.plateSizes,
		layout.plateNeighbors,
		len(refSites),
		targetLandFraction,
	)
	candidate := plateAssignmentCandidate{mask: assignmentMaskForPlateTypes(layout.sortedPlates, plateIsOcean)}
	return blueprintFromLayoutCandidate(layout, candidate), true
}

func blueprintFromLayoutCandidate(layout plateLayout, candidate plateAssignmentCandidate) plateBlueprint {
	continentalByCenter := make(map[int]bool, len(layout.sortedPlates))
	for i, ps := range layout.sortedPlates {
		continentalByCenter[ps.Center] = (candidate.mask & (uint64(1) << i)) != 0
	}

	blueprint := plateBlueprint{
		centers:     make([]Vector3D, 0, len(layout.plateR)),
		weights:     make([]float64, 0, len(layout.plateR)),
		continental: make([]bool, 0, len(layout.plateR)),
		attempt:     layout.attempt,
	}
	for i, center := range layout.plateR {
		blueprint.centers = append(blueprint.centers, layout.plateCenters[i])
		blueprint.weights = append(blueprint.weights, layout.plateWeights[i])
		blueprint.continental = append(blueprint.continental, continentalByCenter[center])
	}
	return blueprint
}

func assignmentMaskForPlateTypes(sortedPlates []PlateSize, plateIsOcean map[int]bool) uint64 {
	var mask uint64
	for i, ps := range sortedPlates {
		if !plateIsOcean[ps.Center] {
			mask |= uint64(1) << i
		}
	}
	return mask
}

func buildPlateLayoutFromBlueprint(
	sites []Vector3D,
	cells []VoronoiCell,
	blueprint plateBlueprint,
) plateLayout {
	plateR := make([]int, 0, len(blueprint.centers))
	plateCenters := make([]Vector3D, 0, len(blueprint.centers))
	plateWeights := make([]float64, 0, len(blueprint.centers))
	plateContinental := make([]bool, 0, len(blueprint.centers))
	chosen := make(map[int]bool, len(blueprint.centers))
	for i, center := range blueprint.centers {
		centerR := nearestRegionToVector(sites, center, chosen)
		if centerR == -1 {
			continue
		}
		chosen[centerR] = true
		plateR = append(plateR, centerR)
		plateCenters = append(plateCenters, center)
		plateWeights = append(plateWeights, blueprint.weights[i])
		plateContinental = append(plateContinental, blueprint.continental[i])
	}

	plateWeight := make(map[int]float64, len(plateR))
	for i, centerR := range plateR {
		plateWeight[centerR] = plateWeights[i]
	}
	rPlate := assignPlateRegionsByWeightedDistance(sites, plateR, plateCenters, plateWeight)
	SmoothPlateBoundaries(cells, rPlate, 3)

	plateSizes := make(map[int]int)
	for _, plate := range rPlate {
		plateSizes[plate]++
	}

	activePlateR := make([]int, 0, len(plateR))
	activePlateCenters := make([]Vector3D, 0, len(plateR))
	activePlateWeights := make([]float64, 0, len(plateR))
	activePlateIsOcean := make(map[int]bool, len(plateR))
	for i, center := range plateR {
		if plateSizes[center] == 0 {
			continue
		}
		activePlateR = append(activePlateR, center)
		activePlateCenters = append(activePlateCenters, plateCenters[i])
		activePlateWeights = append(activePlateWeights, plateWeights[i])
		activePlateIsOcean[center] = !plateContinental[i]
	}

	sortedPlates := make([]PlateSize, 0, len(activePlateR))
	for _, centerR := range activePlateR {
		sortedPlates = append(sortedPlates, PlateSize{
			Center: centerR,
			Size:   plateSizes[centerR],
		})
	}
	sortPlateSizes(sortedPlates)

	return plateLayout{
		plateR:         activePlateR,
		plateCenters:   activePlateCenters,
		plateWeights:   activePlateWeights,
		rPlate:         rPlate,
		plateSizes:     plateSizes,
		sortedPlates:   sortedPlates,
		plateNeighbors: FindPlateNeighbors(cells, rPlate, activePlateR),
		plateIsOcean:   activePlateIsOcean,
		attempt:        blueprint.attempt,
	}
}

// PlateSize holds plate center and size for sorting
type PlateSize struct {
	Center int
	Size   int
}

type plateAssignmentStats struct {
	continentalRegions    int
	continentalComponents int
	majorContinents       int
	oceanComponents       int
	continentalPlateCount int
	oceanPlateCount       int
	largestContinentShare float64
	continentSizeGini     float64
	includesLargestPlate  bool
}

type plateAssignmentCandidate struct {
	mask  uint64
	stats plateAssignmentStats
	score float64
}

func targetContinentalFraction(targetLandFraction float64) float64 {
	fraction := targetLandFraction + 0.02
	if fraction < 0.28 {
		return 0.28
	}
	if fraction > 0.36 {
		return 0.36
	}
	return fraction
}

func minimumContinentalFraction(targetLandFraction float64) float64 {
	fraction := targetLandFraction - 0.005
	if fraction < 0.24 {
		return 0.24
	}
	return fraction
}

func desiredContinentComponents(numPlates int) float64 {
	switch {
	case numPlates <= 8:
		return 3.5
	case numPlates <= 14:
		return 4.0
	default:
		return 4.5
	}
}

func findBestPlateTypeAssignment(
	sortedPlates []PlateSize,
	plateSizes map[int]int,
	plateNeighbors map[int]map[int]bool,
	totalRegions int,
	targetLandFraction float64,
) (map[int]bool, bool) {
	bestCandidate, ok := findBestPlateTypeAssignmentCandidate(
		sortedPlates,
		plateSizes,
		plateNeighbors,
		totalRegions,
		targetLandFraction,
	)
	if !ok {
		return nil, false
	}

	plateIsOcean := make(map[int]bool, len(sortedPlates))
	for i, ps := range sortedPlates {
		plateIsOcean[ps.Center] = (bestCandidate.mask & (uint64(1) << i)) == 0
	}
	return plateIsOcean, true
}

func findBestPlateTypeAssignmentCandidate(
	sortedPlates []PlateSize,
	plateSizes map[int]int,
	plateNeighbors map[int]map[int]bool,
	totalRegions int,
	targetLandFraction float64,
) (plateAssignmentCandidate, bool) {
	numPlates := len(sortedPlates)
	if numPlates == 0 || numPlates > 18 {
		return plateAssignmentCandidate{}, false
	}

	targetRegions := int(float64(totalRegions) * targetContinentalFraction(targetLandFraction))
	targetFraction := float64(targetRegions) / float64(totalRegions)
	minFraction := minimumContinentalFraction(targetLandFraction)

	candidates := make([]plateAssignmentCandidate, 0)
	maxMask := uint64(1) << numPlates
	for mask := uint64(1); mask < maxMask-1; mask++ {
		stats := evaluatePlateAssignment(mask, sortedPlates, plateSizes, plateNeighbors, totalRegions)
		if stats.continentalPlateCount < 2 || stats.oceanPlateCount < 2 {
			continue
		}

		continentalFraction := float64(stats.continentalRegions) / float64(totalRegions)
		if continentalFraction < minFraction || continentalFraction > 0.46 {
			continue
		}

		score := scorePlateAssignment(stats, continentalFraction, targetFraction, numPlates)
		candidates = append(candidates, plateAssignmentCandidate{
			mask:  mask,
			stats: stats,
			score: score,
		})
	}

	bestScore := math.Inf(1)
	var bestCandidate plateAssignmentCandidate
	bestFound := false
	tolerances := []float64{0.025, 0.04, 0.06}
	for _, tolerance := range tolerances {
		for _, candidate := range candidates {
			continentalFraction := float64(candidate.stats.continentalRegions) / float64(totalRegions)
			if math.Abs(continentalFraction-targetFraction) > tolerance {
				continue
			}
			if candidate.score < bestScore {
				bestScore = candidate.score
				bestCandidate = candidate
				bestFound = true
			}
		}
		if bestFound {
			break
		}
	}

	if !bestFound {
		for _, candidate := range candidates {
			if candidate.score < bestScore {
				bestScore = candidate.score
				bestCandidate = candidate
				bestFound = true
			}
		}
	}

	if !bestFound {
		return plateAssignmentCandidate{}, false
	}

	return bestCandidate, true
}

func scorePlateAssignment(
	stats plateAssignmentStats,
	continentalFraction float64,
	targetFraction float64,
	numPlates int,
) float64 {
	minMajorContinents := 3.0
	targetMajorContinents := desiredContinentComponents(numPlates)
	if numPlates <= 8 {
		minMajorContinents = 3.0
	}
	if numPlates >= 14 {
		minMajorContinents = 4.0
	}

	score := 0.0
	if continentalFraction < targetFraction {
		score += (targetFraction - continentalFraction) * 24.0
	} else {
		score += (continentalFraction - targetFraction) * 12.0
	}

	score += math.Abs(float64(stats.continentalComponents)-targetMajorContinents) * 1.25

	if float64(stats.majorContinents) < minMajorContinents {
		score += (minMajorContinents - float64(stats.majorContinents)) * 5.0
	} else if float64(stats.majorContinents) > targetMajorContinents+2 {
		score += (float64(stats.majorContinents) - (targetMajorContinents + 2)) * 0.75
	}

	if stats.oceanComponents == 0 {
		score += 10.0
	} else {
		score += math.Abs(float64(stats.oceanComponents)-1.0) * 1.25
	}

	if stats.includesLargestPlate {
		score += 2.0
	}

	// Oversized supercontinents are still a failure, but a single somewhat large
	// continent is acceptable if the overall tectonic layout is coherent.
	if stats.largestContinentShare > 0.42 {
		score += (stats.largestContinentShare - 0.42) * 10.0
	}
	if stats.largestContinentShare > 0.45 {
		score += (stats.largestContinentShare - 0.45) * 22.0
	}
	if stats.largestContinentShare > 0.55 {
		score += (stats.largestContinentShare - 0.55) * 70.0
	}
	if stats.largestContinentShare > 0.65 {
		score += (stats.largestContinentShare - 0.65) * 120.0
	}

	// Low inequality is acceptable; only very uniform continent sizes are suspect.
	if stats.continentSizeGini > 0.60 {
		score += (stats.continentSizeGini - 0.60) * 10.0
	}
	if stats.continentSizeGini < 0.18 {
		score += (0.18 - stats.continentSizeGini) * 12.0
	}
	if stats.continentSizeGini < 0.10 {
		score += (0.10 - stats.continentSizeGini) * 18.0
	}

	return score
}

func evaluatePlateAssignment(
	mask uint64,
	sortedPlates []PlateSize,
	plateSizes map[int]int,
	plateNeighbors map[int]map[int]bool,
	totalRegions int,
) plateAssignmentStats {
	stats := plateAssignmentStats{}
	plateIndexByCenter := make(map[int]int, len(sortedPlates))
	for i, ps := range sortedPlates {
		plateIndexByCenter[ps.Center] = i
		if (mask & (uint64(1) << i)) != 0 {
			stats.continentalRegions += plateSizes[ps.Center]
			stats.continentalPlateCount++
			if i == 0 {
				stats.includesLargestPlate = true
			}
		} else {
			stats.oceanPlateCount++
		}
	}

	continentSizes := componentSizesForMask(mask, sortedPlates, plateSizes, plateNeighbors, plateIndexByCenter, true)
	oceanSizes := componentSizesForMask(mask, sortedPlates, plateSizes, plateNeighbors, plateIndexByCenter, false)
	stats.continentalComponents = len(continentSizes)
	stats.oceanComponents = len(oceanSizes)

	if stats.continentalRegions > 0 && len(continentSizes) > 0 {
		majorThreshold := math.Ceil(majorLandmassFraction * float64(totalRegions))
		majorSizes := make([]float64, 0, len(continentSizes))
		largest := 0.0
		for _, size := range continentSizes {
			if size > largest {
				largest = size
			}
			if size >= majorThreshold {
				majorSizes = append(majorSizes, size)
			}
		}
		stats.largestContinentShare = largest / float64(stats.continentalRegions)
		stats.majorContinents = len(majorSizes)
		stats.continentSizeGini = computeGini(majorSizes)
	}

	return stats
}

func componentSizesForMask(
	mask uint64,
	sortedPlates []PlateSize,
	plateSizes map[int]int,
	plateNeighbors map[int]map[int]bool,
	plateIndexByCenter map[int]int,
	wantContinental bool,
) []float64 {
	visited := make([]bool, len(sortedPlates))
	sizes := make([]float64, 0)

	for i, ps := range sortedPlates {
		isContinental := (mask & (uint64(1) << i)) != 0
		if visited[i] || isContinental != wantContinental {
			continue
		}

		size := 0.0
		queue := []int{ps.Center}
		visited[i] = true

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			size += float64(plateSizes[current])

			for neighbor := range plateNeighbors[current] {
				neighborIdx, ok := plateIndexByCenter[neighbor]
				if !ok || visited[neighborIdx] {
					continue
				}
				neighborIsContinental := (mask & (uint64(1) << neighborIdx)) != 0
				if neighborIsContinental != wantContinental {
					continue
				}
				visited[neighborIdx] = true
				queue = append(queue, neighbor)
			}
		}

		sizes = append(sizes, size)
	}

	return sizes
}

func sortPlateSizes(sortedPlates []PlateSize) {
	sort.Slice(sortedPlates, func(i, j int) bool {
		return sortedPlates[i].Size > sortedPlates[j].Size
	})
}

// FindConnectedOceanGroups finds groups of oceanic plates that are connected to each other
func FindConnectedOceanGroups(plateR []int, plateIsOcean map[int]bool, plateNeighbors map[int]map[int]bool) map[int][]int {
	visited := make(map[int]bool)
	groups := make(map[int][]int)
	groupID := 0

	for _, centerR := range plateR {
		if !plateIsOcean[centerR] || visited[centerR] {
			continue
		}

		// BFS to find all connected oceanic plates
		var group []int
		queue := []int{centerR}
		visited[centerR] = true

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			group = append(group, current)

			for neighbor := range plateNeighbors[current] {
				if plateIsOcean[neighbor] && !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}

		groups[groupID] = group
		groupID++
	}

	return groups
}

// FindPlateNeighbors returns a map of plate -> set of neighboring plates
func FindPlateNeighbors(cells []VoronoiCell, rPlate []int, plateR []int) map[int]map[int]bool {
	numRegions := len(cells)
	neighbors := make(map[int]map[int]bool)

	// Initialize empty neighbor sets for each plate
	for _, centerR := range plateR {
		neighbors[centerR] = make(map[int]bool)
	}

	// Find all plate boundaries
	for r := 0; r < numRegions; r++ {
		myPlate := rPlate[r]
		for _, neighborIdx := range cells[r].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR < numRegions {
				neighborPlate := rPlate[neighborR]
				if myPlate != neighborPlate {
					neighbors[myPlate][neighborPlate] = true
				}
			}
		}
	}

	return neighbors
}

// pickRandomRegions selects n random distinct regions
func pickRandomRegions(numRegions, n int, rng *rand.Rand) []int {
	chosen := make(map[int]bool)
	var result []int
	for len(result) < n && len(result) < numRegions {
		r := rng.Intn(numRegions)
		if !chosen[r] {
			chosen[r] = true
			result = append(result, r)
		}
	}
	return result
}

// AssignPlateRotations assigns Euler pole rotations to plates for realistic curved motion.
// - Oceanic plates rotate toward continental neighbors (creates subduction)
// - Continental plates have some bias toward each other (creates mountain ranges)
// - Rotation around Euler poles creates curved velocity fields and realistic hotspot tracks
func AssignPlateRotations(
	layout plateLayout,
	plateIsOcean map[int]bool,
	plateNeighbors map[int]map[int]bool,
	seed int64,
) map[int]PlateRotation {
	plateRot := make(map[int]PlateRotation)
	centerPosByPlate := make(map[int]Vector3D, len(layout.plateR))
	for i, centerR := range layout.plateR {
		if i < len(layout.plateCenters) {
			centerPosByPlate[centerR] = layout.plateCenters[i].Normalize()
		}
	}

	for plateOrdinal, centerR := range layout.plateR {
		centerPos := centerPosByPlate[centerR]
		if centerPos.LengthSq() == 0 {
			continue
		}
		rng := terrainFeatureRNG(seed, 10, int64(plateOrdinal), terrainVectorSeedPart(centerPos))
		isOcean := plateIsOcean[centerR]
		neighbors := plateNeighbors[centerR]

		var targetDir Vector3D
		hasTarget := false

		if isOcean {
			// Oceanic plate: find continental neighbors and move toward them
			for neighborPlate := range neighbors {
				if !plateIsOcean[neighborPlate] {
					neighborPos := centerPosByPlate[neighborPlate]
					if neighborPos.LengthSq() == 0 {
						continue
					}
					targetDir.X += neighborPos.X - centerPos.X
					targetDir.Y += neighborPos.Y - centerPos.Y
					targetDir.Z += neighborPos.Z - centerPos.Z
					hasTarget = true
				}
			}
		} else {
			// Continental plate: 50% chance to move toward another continental neighbor
			if rng.Float64() < 0.5 {
				for neighborPlate := range neighbors {
					if !plateIsOcean[neighborPlate] {
						neighborPos := centerPosByPlate[neighborPlate]
						if neighborPos.LengthSq() == 0 {
							continue
						}
						targetDir.X += neighborPos.X - centerPos.X
						targetDir.Y += neighborPos.Y - centerPos.Y
						targetDir.Z += neighborPos.Z - centerPos.Z
						hasTarget = true
					}
				}
			}
		}

		var pole Vector3D
		// Angular velocity with wide variation (like real plates: 1-10 cm/year)
		// Use log-uniform distribution for natural spread: 0.1 to 1.0 (10x range)
		angularVel := 0.1 * math.Pow(10, rng.Float64()) // Range: 0.1 to 1.0

		if hasTarget {
			// Project target direction onto tangent plane at centerPos
			dot := centerPos.X*targetDir.X + centerPos.Y*targetDir.Y + centerPos.Z*targetDir.Z
			tangent := Vector3D{
				X: targetDir.X - dot*centerPos.X,
				Y: targetDir.Y - dot*centerPos.Y,
				Z: targetDir.Z - dot*centerPos.Z,
			}

			// Normalize tangent
			mag := math.Sqrt(tangent.X*tangent.X + tangent.Y*tangent.Y + tangent.Z*tangent.Z)
			if mag > 1e-6 {
				tangent.X /= mag
				tangent.Y /= mag
				tangent.Z /= mag

				// Euler pole = centerPos × tangent (perpendicular to both)
				// This creates rotation that moves centerPos in the tangent direction
				pole = Vector3D{
					X: centerPos.Y*tangent.Z - centerPos.Z*tangent.Y,
					Y: centerPos.Z*tangent.X - centerPos.X*tangent.Z,
					Z: centerPos.X*tangent.Y - centerPos.Y*tangent.X,
				}

				// Normalize pole
				poleMag := math.Sqrt(pole.X*pole.X + pole.Y*pole.Y + pole.Z*pole.Z)
				if poleMag > 1e-6 {
					pole.X /= poleMag
					pole.Y /= poleMag
					pole.Z /= poleMag
				}

				// Add significant jitter to pole position for variety in chain curvature
				// Larger jitter = more curved hotspot chains (pole closer to/further from 90°)
				jitter := 0.6
				pole.X += (rng.Float64() - 0.5) * jitter
				pole.Y += (rng.Float64() - 0.5) * jitter
				pole.Z += (rng.Float64() - 0.5) * jitter

				// Renormalize
				poleMag = math.Sqrt(pole.X*pole.X + pole.Y*pole.Y + pole.Z*pole.Z)
				if poleMag > 0 {
					pole.X /= poleMag
					pole.Y /= poleMag
					pole.Z /= poleMag
				}
			} else {
				// Fallback to random pole
				pole = randomUnitVector(rng)
			}
		} else {
			// Random Euler pole for plates without clear targets
			pole = randomUnitVector(rng)
			// Random sign for rotation direction
			if rng.Float64() < 0.5 {
				angularVel = -angularVel
			}
		}

		plateRot[centerR] = PlateRotation{
			Pole:            pole,
			AngularVelocity: angularVel,
		}
	}

	return plateRot
}

// randomUnitVector generates a uniformly random unit vector
func randomUnitVector(rng *rand.Rand) Vector3D {
	z := 2*rng.Float64() - 1
	theta := 2 * math.Pi * rng.Float64()
	r := math.Sqrt(1 - z*z)
	return Vector3D{
		X: r * math.Cos(theta),
		Y: r * math.Sin(theta),
		Z: z,
	}
}
