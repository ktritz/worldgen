package terrain

// Plate generation and assignment for tectonic simulation
// Uses weighted BFS for power-law size distribution

import (
	"math"
	"math/rand"
	"sort"
)

const (
	plateLayoutSearchAttempts      = 12
	plateLayoutSearchExtraAttempts = 28
)

type plateLayout struct {
	plateR        []int
	rPlate        []int
	plateSizes    map[int]int
	sortedPlates  []PlateSize
	plateNeighbors map[int]map[int]bool
	attempt       int
}

// GeneratePlates creates plates using weighted BFS for power law size distribution
// Returns: plateR (list of plate center region indices), rPlate (region -> plate center mapping)
func GeneratePlates(sites []Vector3D, cells []VoronoiCell, numPlates int, rng *rand.Rand) ([]int, []int) {
	numRegions := len(sites)

	// Pick well-spaced regions as plate centers to avoid superplates caused by clustered seeds.
	plateR := pickSpacedRegions(sites, numPlates, rng)

	// Assign power law growth weights to each plate
	// Higher weight = grows faster = ends up larger.
	// Use a bounded skew so we still get a few large plates without a single plate
	// swallowing most of the sphere.
	plateWeight := make(map[int]float64)
	for _, centerR := range plateR {
		u := rng.Float64()
		plateWeight[centerR] = 0.8 + 3.2*math.Pow(u, 3.0)
	}

	// Initialize r_plate: maps each region to its plate's center region
	rPlate := make([]int, numRegions)
	for i := range rPlate {
		rPlate[i] = -1
	}

	// Initialize queue with plate centers
	type queueItem struct {
		region int
		weight float64
	}
	var queue []queueItem
	for _, r := range plateR {
		rPlate[r] = r
		queue = append(queue, queueItem{r, plateWeight[r]})
	}

	// Weighted BFS - higher weight plates expand more often
	for len(queue) > 0 {
		// Weighted random selection
		totalWeight := 0.0
		for _, item := range queue {
			totalWeight += item.weight
		}

		target := rng.Float64() * totalWeight
		cumulative := 0.0
		selectedIdx := 0
		for i, item := range queue {
			cumulative += item.weight
			if cumulative >= target {
				selectedIdx = i
				break
			}
		}

		// Remove selected item
		selected := queue[selectedIdx]
		queue[selectedIdx] = queue[len(queue)-1]
		queue = queue[:len(queue)-1]

		// Expand to unassigned neighbors
		plateCenter := rPlate[selected.region]
		weight := plateWeight[plateCenter]
		for _, neighborIdx := range cells[selected.region].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR < numRegions && rPlate[neighborR] == -1 {
				rPlate[neighborR] = plateCenter
				queue = append(queue, queueItem{neighborR, weight})
			}
		}
	}

	return plateR, rPlate
}

func pickSpacedRegions(sites []Vector3D, n int, rng *rand.Rand) []int {
	numRegions := len(sites)
	if n <= 0 || numRegions == 0 {
		return nil
	}

	chosen := make(map[int]bool, n)
	result := make([]int, 0, n)
	first := rng.Intn(numRegions)
	chosen[first] = true
	result = append(result, first)

	sampleCount := 64
	if numRegions < sampleCount {
		sampleCount = numRegions
	}

	for len(result) < n && len(result) < numRegions {
		bestIdx := -1
		bestDistance := -1.0

		for sample := 0; sample < sampleCount; sample++ {
			candidate := rng.Intn(numRegions)
			if chosen[candidate] {
				continue
			}

			minDistance := math.Inf(1)
			for _, existing := range result {
				distance := angularDistance(sites[candidate], sites[existing])
				if distance < minDistance {
					minDistance = distance
				}
			}

			if minDistance > bestDistance {
				bestDistance = minDistance
				bestIdx = candidate
			}
		}

		if bestIdx == -1 {
			for candidate := 0; candidate < numRegions; candidate++ {
				if !chosen[candidate] {
					bestIdx = candidate
					break
				}
			}
		}

		if bestIdx == -1 {
			break
		}

		chosen[bestIdx] = true
		result = append(result, bestIdx)
	}

	return result
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
	plateR, rPlate := GeneratePlates(sites, cells, numPlates, rng)
	SmoothPlateBoundaries(cells, rPlate, 3)

	plateSizes := make(map[int]int)
	for _, plate := range rPlate {
		plateSizes[plate]++
	}

	activePlateR := make([]int, 0, len(plateR))
	for _, center := range plateR {
		if plateSizes[center] > 0 {
			activePlateR = append(activePlateR, center)
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
	bestScore := math.Inf(1)
	var bestLayout plateLayout
	var bestCandidate plateAssignmentCandidate
	found := false

	maxAttempts := plateLayoutSearchAttempts + plateLayoutSearchExtraAttempts
	for attempt := 0; attempt < maxAttempts; attempt++ {
		rng := rand.New(rand.NewSource(seed + int64(attempt)*7919))
		layout := generatePlateLayout(sites, cells, numPlates, rng, attempt)
		candidate, ok := findBestPlateTypeAssignmentCandidate(
			layout.sortedPlates,
			layout.plateSizes,
			layout.plateNeighbors,
			len(sites),
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
		return bestLayout
	}

	// Fallback to the first deterministic layout if scoring failed.
	rng := rand.New(rand.NewSource(seed))
	return generatePlateLayout(sites, cells, numPlates, rng, 0)
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
	sites []Vector3D,
	cells []VoronoiCell,
	plateR []int,
	plateIsOcean map[int]bool,
	plateNeighbors map[int]map[int]bool,
	rng *rand.Rand,
) map[int]PlateRotation {
	plateRot := make(map[int]PlateRotation)

	for _, centerR := range plateR {
		isOcean := plateIsOcean[centerR]
		neighbors := plateNeighbors[centerR]
		centerPos := sites[centerR]

		var targetDir Vector3D
		hasTarget := false

		if isOcean {
			// Oceanic plate: find continental neighbors and move toward them
			for neighborPlate := range neighbors {
				if !plateIsOcean[neighborPlate] {
					neighborPos := sites[neighborPlate]
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
						neighborPos := sites[neighborPlate]
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
