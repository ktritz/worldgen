package terrain

// Elevation computation using distance field algorithm
// Computes elevation from boundary seeds using inverse-distance formula

import (
	"fmt"
	"math"
	"sort"
)

// ComputeElevation computes elevation using distance field algorithm
// Returns normalized elevation and coastline regions
func ComputeElevation(
	sites []Vector3D,
	cells []VoronoiCell,
	plateIsOcean map[int]bool,
	rPlate []int,
	plateRot map[int]PlateRotation,
	seed int64,
) ([]float64, map[int]bool, map[int]bool, map[int]bool, map[int]bool, map[int]bool, map[int]bool) {
	elevation, seeds := ComputeElevationWithSeeds(sites, cells, plateIsOcean, rPlate, plateRot, seed)
	return elevation, seeds.Coastline, seeds.Mountain, seeds.Collision, seeds.Arc, seeds.Ridge, seeds.Trench
}

// ComputeElevationWithSeeds is ComputeElevation with the full boundary seed set
// returned instead of the historical positional subset. It exists so callers
// that need seed classes the tuple never carried (notably rift seeds) can reach
// them without recomputing collisions.
func ComputeElevationWithSeeds(
	sites []Vector3D,
	cells []VoronoiCell,
	plateIsOcean map[int]bool,
	rPlate []int,
	plateRot map[int]PlateRotation,
	seed int64,
) ([]float64, BoundarySeeds) {
	numRegions := len(sites)
	epsilon := 1e-3

	// Find collision zones
	seeds := FindCollisions(sites, cells, plateIsOcean, rPlate, plateRot)

	fmt.Printf("  Seeds - Mountain: %d, Coastline: %d, Ocean: %d (ridges: %d, trenches: %d)\n",
		len(seeds.Mountain), len(seeds.Coastline), len(seeds.Ocean), len(seeds.Ridge), len(seeds.Trench))

	// Build stop set (all seeds)
	stopR := make(map[int]bool)
	for r := range seeds.Mountain {
		stopR[r] = true
	}
	for r := range seeds.Coastline {
		stopR[r] = true
	}
	for r := range seeds.Ocean {
		stopR[r] = true
	}
	for r := range seeds.Rift {
		stopR[r] = true
	}

	oceanLikeSeeds := make(map[int]bool, len(seeds.Ocean)+len(seeds.Rift))
	for r := range seeds.Ocean {
		oceanLikeSeeds[r] = true
	}
	for r := range seeds.Rift {
		oceanLikeSeeds[r] = true
	}

	// Compute distance fields
	rDistanceA := AssignDistanceField(cells, seeds.Mountain, oceanLikeSeeds)
	rDistanceB := AssignDistanceField(cells, oceanLikeSeeds, seeds.Coastline)
	rDistanceC := AssignDistanceField(cells, seeds.Coastline, stopR)

	// Compute elevation using original formula
	elevation := make([]float64, numRegions)
	for r := 0; r < numRegions; r++ {
		a := rDistanceA[r] + epsilon
		b := rDistanceB[r] + epsilon
		c := rDistanceC[r] + epsilon

		if math.IsInf(a, 1) && math.IsInf(b, 1) {
			elevation[r] = 0.1
		} else {
			elevation[r] = (1/a - 1/b) / (1/a + 1/b + 1/c)
		}

		// Add noise
		elevation[r] += NoiseAmplitude * FBMNoise(sites[r], seed)
	}

	// Post-process: adjust ridges
	for r := range seeds.Ridge {
		elevation[r] += 0.15
	}

	// Continental rifts should bias toward low elevations so some of them remain
	// as inland seas after hypsometric remapping instead of welding continents together.
	for r := range seeds.Rift {
		elevation[r] -= 0.22
	}

	// Compute forearc elevation profile for subduction zones
	// This creates gradual transition: trench → forearc basin → volcanic arc
	applySubductionProfile(sites, cells, seeds.Trench, seeds.Arc, rPlate, plateIsOcean, elevation)

	// Divergent continental boundaries should start from a lower baseline so they
	// can plausibly survive later remapping as rift valleys or inland seas.
	for r := range seeds.Rift {
		elevation[r] -= 0.22
	}

	// Debug: print some trench locations
	if len(seeds.Trench) > 0 {
		count := 0
		for r := range seeds.Trench {
			if count < 5 {
				lat := math.Asin(sites[r].Z) * 180 / math.Pi
				lon := math.Atan2(sites[r].Y, sites[r].X) * 180 / math.Pi
				fmt.Printf("    Trench at lat=%.1f, lon=%.1f\n", lat, lon)
				count++
			}
		}
	}

	return elevation, seeds
}

// AssignDistanceField computes graph distance from seeds using deterministic BFS.
// Randomized queue order made terrain sensitive to map iteration and to unrelated
// RNG consumption; stable ordering keeps this structural field reproducible.
func AssignDistanceField(cells []VoronoiCell, seedsR map[int]bool, stopR map[int]bool) []float64 {
	numRegions := len(cells)
	rDistance := make([]float64, numRegions)
	for i := range rDistance {
		rDistance[i] = math.Inf(1)
	}

	// Initialize queue with seeds
	queue := make([]int, 0, len(seedsR))
	for r := range seedsR {
		if r >= 0 && r < numRegions {
			queue = append(queue, r)
			rDistance[r] = 0
		}
	}
	sort.Ints(queue)

	// Stable FIFO BFS.
	for queueOut := 0; queueOut < len(queue); queueOut++ {
		currentR := queue[queueOut]
		for _, neighborIdx := range cells[currentR].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR < numRegions && math.IsInf(rDistance[neighborR], 1) && !stopR[neighborR] {
				rDistance[neighborR] = rDistance[currentR] + 1
				queue = append(queue, neighborR)
			}
		}
	}

	return rDistance
}

// ComputeDistanceFromCoast computes distance from coastline for each continental region
func ComputeDistanceFromCoast(cells []VoronoiCell, coastlineR map[int]bool, rPlate []int, plateIsOcean map[int]bool) []float64 {
	numRegions := len(cells)
	distFromCoast := make([]float64, numRegions)
	for r := range distFromCoast {
		distFromCoast[r] = math.Inf(1)
	}

	// BFS from coastline regions inward
	var queue []int
	for r := range coastlineR {
		distFromCoast[r] = 0
		queue = append(queue, r)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		currentDist := distFromCoast[current]

		for _, nIdx := range cells[current].NeighborSiteIndices {
			neighbor := int(nIdx)
			if neighbor >= numRegions {
				continue
			}
			// Only propagate within continental plates
			if !plateIsOcean[rPlate[neighbor]] {
				newDist := currentDist + 1
				if newDist < distFromCoast[neighbor] {
					distFromCoast[neighbor] = newDist
					queue = append(queue, neighbor)
				}
			}
		}
	}

	return distFromCoast
}

// ComputeContinentalComponentMaxDistance returns, for each continental region,
// the maximum inland distance within its connected continental domain.
func ComputeContinentalComponentMaxDistance(cells []VoronoiCell, rPlate []int, plateIsOcean map[int]bool, distFromCoast []float64) []float64 {
	numRegions := len(cells)
	componentMax := make([]float64, numRegions)
	visited := make([]bool, numRegions)

	for start := 0; start < numRegions; start++ {
		if visited[start] || plateIsOcean[rPlate[start]] {
			continue
		}

		queue := []int{start}
		component := make([]int, 0)
		visited[start] = true
		maxDist := 0.0

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			component = append(component, current)

			if !math.IsInf(distFromCoast[current], 1) && distFromCoast[current] > maxDist {
				maxDist = distFromCoast[current]
			}

			for _, nIdx := range cells[current].NeighborSiteIndices {
				neighbor := int(nIdx)
				if neighbor < 0 || neighbor >= numRegions || visited[neighbor] || plateIsOcean[rPlate[neighbor]] {
					continue
				}
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}

		if maxDist < 1 {
			maxDist = 1
		}
		for _, region := range component {
			componentMax[region] = maxDist
		}
	}

	return componentMax
}

// ComputeDistanceFromMountainSeeds computes distance from tectonic mountain belts
// within continental domains. This is used to avoid turning every continental
// interior into a monotonic broad dome.
func ComputeDistanceFromMountainSeeds(
	cells []VoronoiCell,
	mountainR map[int]bool,
	rPlate []int,
	plateIsOcean map[int]bool,
) []float64 {
	numRegions := len(cells)
	distFromMountain := make([]float64, numRegions)
	for r := range distFromMountain {
		distFromMountain[r] = math.Inf(1)
	}

	var queue []int
	for r := range mountainR {
		if r < 0 || r >= numRegions || plateIsOcean[rPlate[r]] {
			continue
		}
		distFromMountain[r] = 0
		queue = append(queue, r)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		currentDist := distFromMountain[current]

		for _, nIdx := range cells[current].NeighborSiteIndices {
			neighbor := int(nIdx)
			if neighbor < 0 || neighbor >= numRegions || plateIsOcean[rPlate[neighbor]] {
				continue
			}
			newDist := currentDist + 1
			if newDist < distFromMountain[neighbor] {
				distFromMountain[neighbor] = newDist
				queue = append(queue, neighbor)
			}
		}
	}

	return distFromMountain
}

// ComputeContinentalComponentMaxTectonicDistance returns, for each continental
// region, the maximum finite distance from tectonic mountain belts within its
// connected continental domain.
func ComputeContinentalComponentMaxTectonicDistance(
	cells []VoronoiCell,
	rPlate []int,
	plateIsOcean map[int]bool,
	distFromMountain []float64,
) []float64 {
	numRegions := len(cells)
	componentMax := make([]float64, numRegions)
	visited := make([]bool, numRegions)

	for start := 0; start < numRegions; start++ {
		if visited[start] || plateIsOcean[rPlate[start]] {
			continue
		}

		queue := []int{start}
		component := make([]int, 0)
		visited[start] = true
		maxDist := 0.0

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			component = append(component, current)

			if !math.IsInf(distFromMountain[current], 1) && distFromMountain[current] > maxDist {
				maxDist = distFromMountain[current]
			}

			for _, nIdx := range cells[current].NeighborSiteIndices {
				neighbor := int(nIdx)
				if neighbor < 0 || neighbor >= numRegions || visited[neighbor] || plateIsOcean[rPlate[neighbor]] {
					continue
				}
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}

		if maxDist < 1 {
			maxDist = 1
		}
		for _, region := range component {
			componentMax[region] = maxDist
		}
	}

	return componentMax
}

// ComputeOceanDistanceFromCoast computes distance from coastline for ocean regions.
func ComputeOceanDistanceFromCoast(cells []VoronoiCell, coastlineR map[int]bool, rPlate []int, plateIsOcean map[int]bool) []float64 {
	numRegions := len(cells)
	distFromCoast := make([]float64, numRegions)
	for r := range distFromCoast {
		distFromCoast[r] = math.Inf(1)
	}

	var queue []int
	for r := range coastlineR {
		if plateIsOcean[rPlate[r]] {
			distFromCoast[r] = 0
			queue = append(queue, r)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		currentDist := distFromCoast[current]

		for _, nIdx := range cells[current].NeighborSiteIndices {
			neighbor := int(nIdx)
			if neighbor >= numRegions {
				continue
			}
			if plateIsOcean[rPlate[neighbor]] {
				newDist := currentDist + 1
				if newDist < distFromCoast[neighbor] {
					distFromCoast[neighbor] = newDist
					queue = append(queue, neighbor)
				}
			}
		}
	}

	return distFromCoast
}

// ComputeOceanDistanceFromSeeds computes ocean-only distance from a seed set.
func ComputeOceanDistanceFromSeeds(cells []VoronoiCell, seedR map[int]bool, rPlate []int, plateIsOcean map[int]bool) []float64 {
	numRegions := len(cells)
	dist := make([]float64, numRegions)
	for r := range dist {
		dist[r] = math.Inf(1)
	}

	var queue []int
	for r := range seedR {
		if r < 0 || r >= numRegions || !plateIsOcean[rPlate[r]] {
			continue
		}
		dist[r] = 0
		queue = append(queue, r)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		currentDist := dist[current]

		for _, nIdx := range cells[current].NeighborSiteIndices {
			neighbor := int(nIdx)
			if neighbor < 0 || neighbor >= numRegions || !plateIsOcean[rPlate[neighbor]] {
				continue
			}
			nextDist := currentDist + 1
			if nextDist < dist[neighbor] {
				dist[neighbor] = nextDist
				queue = append(queue, neighbor)
			}
		}
	}

	return dist
}

// ComputeOceanPlateMaxDistance returns, for each oceanic region, the maximum
// finite distance observed on its plate in the provided distance field.
func ComputeOceanPlateMaxDistance(rPlate []int, plateIsOcean map[int]bool, dist []float64) []float64 {
	plateMax := make(map[int]float64)
	for r, value := range dist {
		plate := rPlate[r]
		if !plateIsOcean[plate] || math.IsInf(value, 1) {
			continue
		}
		if value > plateMax[plate] {
			plateMax[plate] = value
		}
	}

	componentMax := make([]float64, len(dist))
	for r := range dist {
		plate := rPlate[r]
		if !plateIsOcean[plate] {
			continue
		}
		maxDist := plateMax[plate]
		if maxDist < 1 {
			maxDist = 1
		}
		componentMax[r] = maxDist
	}
	return componentMax
}

// ApplyBimodalElevation applies bimodal elevation with continental slope
func ApplyBimodalElevation(
	elevation []float64,
	distFromCoast []float64,
	oceanDistFromCoast []float64,
	componentMaxDist []float64,
	distFromMountain []float64,
	componentMaxMountainDist []float64,
	distFromCollision []float64,
	componentMaxCollisionDist []float64,
	distFromArc []float64,
	componentMaxArcDist []float64,
	rPlate []int,
	plateIsOcean map[int]bool,
	maxDist float64,
	maxOceanDist float64,
) {
	numRegions := len(elevation)

	for r := 0; r < numRegions; r++ {
		plate := rPlate[r]
		e := elevation[r]

		if plateIsOcean[plate] {
			dist := oceanDistFromCoast[r]
			if math.IsInf(dist, 1) {
				dist = maxOceanDist
			}

			shelfWidth := math.Max(2.0, maxOceanDist*0.06)
			slopeWidth := math.Max(shelfWidth+2.0, maxOceanDist*0.22)

			switch {
			case dist <= shelfWidth:
				t := SmoothStep(0, shelfWidth, dist)
				baseElev := Lerp(-0.05, -0.16, t)
				elevation[r] = baseElev + e*0.06
			case dist <= slopeWidth:
				t := SmoothStep(shelfWidth, slopeWidth, dist)
				baseElev := Lerp(-0.16, -0.68, t)
				elevation[r] = baseElev + e*0.14
			default:
				t := SmoothStep(slopeWidth, maxOceanDist, dist)
				abyssBonus := SmoothStep(0.55, 1.0, t)
				baseElev := Lerp(-0.68, -0.86, t) - 0.05*abyssBonus
				elevation[r] = baseElev + e*(0.22+0.02*abyssBonus)
			}
		} else {
			// Continental: base elevation depends on distance from coast, but use
			// local continental-domain scale so smaller continents can retain
			// meaningful interiors instead of being flattened by the global max.
			localMaxDist := componentMaxDist[r]
			if localMaxDist <= 0 {
				localMaxDist = maxDist
			}
			localNorm := distFromCoast[r] / localMaxDist
			globalNorm := distFromCoast[r] / maxDist
			normalizedDist := 0.75*localNorm + 0.25*globalNorm
			if math.IsInf(distFromCoast[r], 1) {
				normalizedDist = 0.5
			}
			if normalizedDist < 0 {
				normalizedDist = 0
			}
			if normalizedDist > 1 {
				normalizedDist = 1
			}

			inlandness := SmoothStep(0, 1, normalizedDist)

			genericSupport := 0.0
			if localMaxMountainDist := componentMaxMountainDist[r]; localMaxMountainDist > 0 && !math.IsInf(distFromMountain[r], 1) {
				normMountainDist := distFromMountain[r] / localMaxMountainDist
				if normMountainDist < 0 {
					normMountainDist = 0
				}
				if normMountainDist > 1 {
					normMountainDist = 1
				}
				genericSupport = 1.0 - SmoothStep(0, 1, normMountainDist)
			}

			collisionSupport := 0.0
			if localMaxCollisionDist := componentMaxCollisionDist[r]; localMaxCollisionDist > 0 && !math.IsInf(distFromCollision[r], 1) {
				normCollisionDist := distFromCollision[r] / localMaxCollisionDist
				if normCollisionDist < 0 {
					normCollisionDist = 0
				}
				if normCollisionDist > 1 {
					normCollisionDist = 1
				}
				collisionSupport = 1.0 - SmoothStep(0, 1, normCollisionDist)
			}

			arcSupport := 0.0
			if localMaxArcDist := componentMaxArcDist[r]; localMaxArcDist > 0 && !math.IsInf(distFromArc[r], 1) {
				normArcDist := distFromArc[r] / localMaxArcDist
				if normArcDist < 0 {
					normArcDist = 0
				}
				if normArcDist > 1 {
					normArcDist = 1
				}
				arcSupport = 1.0 - SmoothStep(0, 1, normArcDist)
			}

			genericBelt := math.Sqrt(genericSupport)
			collisionBelt := math.Sqrt(collisionSupport)
			arcBelt := math.Sqrt(arcSupport)
			coastalness := 1.0 - SmoothStep(0.08, 0.78, inlandness)
			collisionPlateau := collisionSupport * SmoothStep(0.18, 0.92, inlandness)
			collisionCore := collisionBelt * (0.45 + 0.55*inlandness)
			arcCordillera := arcBelt * (0.65 + 0.35*coastalness)

			// Cratonic interiors should not rise monotonically all the way to a
			// central dome. Collision belts create broad elevated interiors;
			// volcanic arcs create tighter, coast-parallel uplift.
			cratonLift := 0.05 + inlandness*0.16
			tectonicLift := genericBelt * (0.10 + inlandness*0.20)
			collisionPlateauLift := collisionPlateau * (0.08 + inlandness*0.16)
			collisionRangeLift := collisionCore * (0.05 + inlandness*0.14)
			arcLift := arcCordillera * (0.06 + coastalness*0.11)
			basinPenalty := inlandness * (1.0 - 0.55*collisionSupport) * (1.0 - genericSupport) * (1.0 - 0.35*arcSupport) * 0.13
			noiseScale := 0.16 + inlandness*0.05 + genericBelt*0.16 + collisionPlateau*0.05 + collisionCore*0.08 + arcCordillera*0.10

			elevation[r] = cratonLift + tectonicLift + collisionPlateauLift + collisionRangeLift + arcLift - basinPenalty + e*noiseScale
		}
	}
}

// ApplyOceanBasinStructure adds pre-hypsometry structure to ocean basins based
// on spreading age, subduction proximity, and plate-motion-aligned lineation.
func ApplyOceanBasinStructure(
	elevation []float64,
	sites []Vector3D,
	rPlate []int,
	plateIsOcean map[int]bool,
	plateRot map[int]PlateRotation,
	oceanDistFromCoast []float64,
	distFromRidge []float64,
	maxRidgeDist []float64,
	distFromTrench []float64,
	maxOceanDist float64,
	seed int64,
) {
	type plateFrame struct {
		axisA Vector3D
		axisB Vector3D
	}

	frames := make(map[int]plateFrame)
	for plateID, rotation := range plateRot {
		if !plateIsOcean[plateID] {
			continue
		}
		pole := rotation.Pole.Normalize()
		ref := Vector3D{X: 0, Y: 0, Z: 1}
		if math.Abs(pole.Dot(ref)) > 0.90 {
			ref = Vector3D{X: 1, Y: 0, Z: 0}
		}
		axisA := pole.Cross(ref).Normalize()
		axisB := pole.Cross(axisA).Normalize()
		frames[plateID] = plateFrame{axisA: axisA, axisB: axisB}
	}

	if maxOceanDist < 1 {
		maxOceanDist = 1
	}

	// distFromRidge is a BFS hop count; convert hops to physical distance so
	// the abyssal-hill band wavelength does not shrink as the mesh is refined.
	stepScale := meshPathCostResolutionScale(len(elevation))

	for r, elev := range elevation {
		plate := rPlate[r]
		if !plateIsOcean[plate] {
			continue
		}

		ridgeScale := maxRidgeDist[r]
		if ridgeScale <= 0 {
			ridgeScale = maxOceanDist
		}
		ridgeAge := distFromRidge[r] / ridgeScale
		if math.IsInf(distFromRidge[r], 1) || ridgeAge > 1 {
			ridgeAge = 1
		}
		if ridgeAge < 0 {
			ridgeAge = 0
		}
		matureOcean := SmoothStep(0.18, 0.90, ridgeAge)

		coastNorm := oceanDistFromCoast[r] / maxOceanDist
		if math.IsInf(oceanDistFromCoast[r], 1) || coastNorm > 1 {
			coastNorm = 1
		}
		if coastNorm < 0 {
			coastNorm = 0
		}
		openOcean := SmoothStep(0.10, 0.55, coastNorm)

		ridgeSupport := 0.0
		if !math.IsInf(distFromRidge[r], 1) {
			ridgeSupport = 1.0 - SmoothStep(0, math.Max(2.0, ridgeScale*0.10), distFromRidge[r])
		}

		trenchSupport := 0.0
		if !math.IsInf(distFromTrench[r], 1) {
			trenchSupport = 1.0 - SmoothStep(0, math.Max(3.0, ridgeScale*0.14), distFromTrench[r])
		}

		// Ridge-flank uplift, old-basin deepening, and trench sharpening.
		structure := 0.055*ridgeSupport - 0.085*matureOcean*openOcean - 0.075*trenchSupport

		// Abyssal hills are organized roughly parallel to ridge flanks, so use
		// distance from spreading centers to create ridge-parallel bands.
		ridgeBandFreq := 0.75 + 0.18*(1.0+FBMNoiseWithFreq(sites[r], seed+141414, 6.0, 2))
		ridgeBands := math.Sin(distFromRidge[r] * stepScale * ridgeBandFreq)
		ridgeHillAmp := 0.014 * (0.35 + 0.65*(1.0-matureOcean)) * (0.25 + 0.75*openOcean)
		structure += ridgeBands * ridgeHillAmp

		// Broad motion-aligned lineation approximates fracture-zone style
		// organization without requiring a full spreading-history model.
		frame := frames[plate]
		pos := sites[r].Normalize()
		azimuth := math.Atan2(pos.Dot(frame.axisB), pos.Dot(frame.axisA))
		lineationPhase := azimuth*(7.0+2.0*FBMNoiseWithFreq(pos, seed+242424, 2.5, 2)) +
			1.2*FBMNoiseWithFreq(pos, seed+343434, 8.0, 2)
		lineation := math.Sin(lineationPhase)
		lineationAmp := 0.010 * SmoothStep(0.10, 0.65, ridgeAge) * openOcean * (1.0 - 0.70*trenchSupport)
		structure += lineation * lineationAmp

		// Old, open-ocean crust can host sparse non-hotspot seamount provinces.
		provinceNoise := FBMNoiseWithFreq(pos, seed+454545, 3.2, 3)
		provinceMask := SmoothStep(0.46, 0.78, provinceNoise)
		provinceBands := 0.5 + 0.5*math.Sin(azimuth*5.5+2.0*FBMNoiseWithFreq(pos, seed+565656, 2.0, 2))
		structure += 0.018 * provinceMask * provinceBands * matureOcean * openOcean * (1.0 - trenchSupport)

		elevation[r] = elev + structure
	}
}

// FBMNoise generates fractal Brownian motion noise
func FBMNoise(pos Vector3D, seed int64) float64 {
	persistence := 2.0 / 3.0
	sum := 0.0
	sumOfAmplitudes := 0.0

	for octave := 0; octave < 5; octave++ {
		amplitude := math.Pow(persistence, float64(octave))
		frequency := float64(int(1) << octave)
		sum += amplitude * SimplexNoise3D(pos.X*frequency, pos.Y*frequency, pos.Z*frequency, seed+int64(octave)*1000)
		sumOfAmplitudes += amplitude
	}

	return sum / sumOfAmplitudes
}

// SimplexNoise3D is a hash-based gradient noise
func SimplexNoise3D(x, y, z float64, seed int64) float64 {
	ix := int64(math.Floor(x))
	iy := int64(math.Floor(y))
	iz := int64(math.Floor(z))

	fx := x - float64(ix)
	fy := y - float64(iy)
	fz := z - float64(iz)

	// Smoothstep
	ux := fx * fx * (3 - 2*fx)
	uy := fy * fy * (3 - 2*fy)
	uz := fz * fz * (3 - 2*fz)

	// Hash corners
	n000 := hashGradient(ix, iy, iz, seed, fx, fy, fz)
	n100 := hashGradient(ix+1, iy, iz, seed, fx-1, fy, fz)
	n010 := hashGradient(ix, iy+1, iz, seed, fx, fy-1, fz)
	n110 := hashGradient(ix+1, iy+1, iz, seed, fx-1, fy-1, fz)
	n001 := hashGradient(ix, iy, iz+1, seed, fx, fy, fz-1)
	n101 := hashGradient(ix+1, iy, iz+1, seed, fx-1, fy, fz-1)
	n011 := hashGradient(ix, iy+1, iz+1, seed, fx, fy-1, fz-1)
	n111 := hashGradient(ix+1, iy+1, iz+1, seed, fx-1, fy-1, fz-1)

	// Trilinear interpolation
	nx00 := n000*(1-ux) + n100*ux
	nx10 := n010*(1-ux) + n110*ux
	nx01 := n001*(1-ux) + n101*ux
	nx11 := n011*(1-ux) + n111*ux

	nxy0 := nx00*(1-uy) + nx10*uy
	nxy1 := nx01*(1-uy) + nx11*uy

	return nxy0*(1-uz) + nxy1*uz
}

func hashGradient(ix, iy, iz, seed int64, fx, fy, fz float64) float64 {
	h := seed
	h ^= ix * 374761393
	h ^= iy * 668265263
	h ^= iz * 1274126177
	h = h*h*h*60493 + h

	gx := float64((h>>0)&0xFF)/127.5 - 1
	gy := float64((h>>8)&0xFF)/127.5 - 1
	gz := float64((h>>16)&0xFF)/127.5 - 1

	return gx*fx + gy*fy + gz*fz
}

// ApplyElevationScaledNoise adds fractal noise with amplitude proportional to elevation
// This creates:
// - Rough, jagged mountain peaks
// - Gently rolling plains and lowlands
// - Moderate roughness on ocean floor (abyssal hills, ridges)
//
// Parameters:
// - baseFrequency: base noise frequency (higher = more detail, smaller features)
// - mountainAmplitude: max noise amplitude for high mountains (meters)
// - plainAmplitude: noise amplitude for low-elevation land (meters)
// - oceanAmplitude: noise amplitude for ocean floor (meters)
func ApplyElevationScaledNoise(
	sites []Vector3D,
	cells []VoronoiCell,
	elevation []float64,
	oceanDistFromCoast []float64,
	seed int64,
	baseFrequency float64,
	mountainAmplitude float64,
	plainAmplitude float64,
	oceanAmplitude float64,
) {
	// Use a different seed offset for this noise layer
	noiseSeed := seed + 999999
	coastalExposure := ComputeCoastalExposure(cells, elevation, oceanDistFromCoast)

	for i := range elevation {
		elev := elevation[i]

		// Determine amplitude based on elevation
		var amplitude float64
		if elev > 0 {
			// Land: scale from plainAmplitude at sea level to mountainAmplitude at high elevation
			// Use sqrt scaling so mountains get rougher faster
			normalizedElev := math.Min(1.0, elev/4000.0) // 4000m as "full mountain"
			amplitude = plainAmplitude + (mountainAmplitude-plainAmplitude)*math.Sqrt(normalizedElev)
		} else {
			// Ocean: use ocean amplitude, slightly scaled by depth
			normalizedDepth := math.Min(1.0, math.Abs(elev)/5000.0)
			amplitude = oceanAmplitude * (0.5 + 0.5*normalizedDepth)
		}

		// Coasts and shelves should avoid tiny serration, but they still need
		// coherent medium-scale bays and capes. Suppress high-frequency detail
		// near sea level and add a broader coastal-shape band instead.
		highFreqCoastalFactor := 0.16 + 0.84*SmoothStep(350, 1800, math.Abs(elev))
		coastalBand := 1.0 - SmoothStep(80, 900, math.Abs(elev))
		exposure := coastalExposure[i]
		amplitude *= highFreqCoastalFactor * (0.72 + 0.28*exposure)

		// Generate multi-octave noise at this position
		pos := sites[i]
		detailNoise := FBMNoiseWithFreq(pos, noiseSeed, baseFrequency, 6)
		coastalNoise := FBMNoiseWithFreq(pos, noiseSeed+424242, baseFrequency*0.16, 3)
		macroCoastalNoise := FBMNoiseWithFreq(pos, noiseSeed+717171, baseFrequency*0.05, 2)

		coastalAmplitude := coastalBand * (8.0 + 120.0*exposure)
		openMarginFactor := SmoothStep(0.45, 0.95, exposure)
		macroCoastalAmplitude := coastalBand * openMarginFactor * (18.0 + 72.0*SmoothStep(0, 1200, math.Abs(elev)))

		elevation[i] += amplitude*detailNoise + coastalAmplitude*coastalNoise + macroCoastalAmplitude*macroCoastalNoise
	}
}

// ComputeCoastalExposure estimates whether a near-coastal location faces open
// ocean or a cramped seaway. Open margins can support broader embayments;
// crowded margins should stay simpler to avoid inflated tortuosity.
func ComputeCoastalExposure(cells []VoronoiCell, elevation []float64, oceanDistFromCoast []float64) []float64 {
	exposure := make([]float64, len(elevation))
	maxOceanDist := 1.0
	for i := range oceanDistFromCoast {
		if !math.IsInf(oceanDistFromCoast[i], 1) && oceanDistFromCoast[i] > maxOceanDist {
			maxOceanDist = oceanDistFromCoast[i]
		}
	}
	exposureScale := math.Max(2.0, maxOceanDist*0.18)
	// Sample a fixed PHYSICAL neighborhood: 2 hops at baseline resolution,
	// proportionally more hops on finer meshes.
	maxDepth := meshResolutionAdjustedSteps(2, len(cells))

	type item struct {
		region int
		depth  int
	}

	// Reuse one visited-stamp array and one queue buffer across all cells:
	// a fresh map[int]bool per cell dominated this function's cost at L7.
	visited := make([]int32, len(cells))
	stamp := int32(0)
	queue := make([]item, 0, 64)

	for r := range cells {
		if math.Abs(elevation[r]) > 1200 {
			continue
		}

		stamp++
		queue = append(queue[:0], item{region: r, depth: 0})
		visited[r] = stamp
		head := 0
		sum := 0.0
		weightSum := 0.0

		for head < len(queue) {
			current := queue[head]
			head++

			weight := 1.0 / float64(current.depth+1)
			if !math.IsInf(oceanDistFromCoast[current.region], 1) {
				sum += oceanDistFromCoast[current.region] * weight
				weightSum += weight
			}

			if current.depth >= maxDepth {
				continue
			}

			for _, neighborIdx := range cells[current.region].NeighborSiteIndices {
				neighbor := int(neighborIdx)
				if neighbor < 0 || neighbor >= len(cells) || visited[neighbor] == stamp {
					continue
				}
				visited[neighbor] = stamp
				queue = append(queue, item{region: neighbor, depth: current.depth + 1})
			}
		}

		if weightSum == 0 {
			continue
		}

		meanOceanDistance := sum / weightSum
		exposure[r] = SmoothStep(1.5, exposureScale, meanOceanDistance)
	}

	return exposure
}

// RegularizeCoastlines removes single-cell serration near sea level without
// erasing medium-scale embayments or tectonic seaways.
func RegularizeCoastlines(cells []VoronoiCell, elevation []float64, coastalExposure []float64, iterations int) {
	if iterations <= 0 {
		return
	}

	const coastalWindow = 600.0
	buffer := make([]float64, len(elevation))

	for iter := 0; iter < iterations; iter++ {
		copy(buffer, elevation)

		for r, cell := range cells {
			elev := elevation[r]
			if math.Abs(elev) > coastalWindow {
				continue
			}

			neighborCount := 0
			landNeighbors := 0
			oceanNeighbors := 0
			sum := 0.0

			for _, neighborIdx := range cell.NeighborSiteIndices {
				neighbor := int(neighborIdx)
				if neighbor < 0 || neighbor >= len(elevation) {
					continue
				}
				neighborCount++
				sum += elevation[neighbor]
				if elevation[neighbor] > 0 {
					landNeighbors++
				} else {
					oceanNeighbors++
				}
			}

			if neighborCount == 0 || landNeighbors == 0 || oceanNeighbors == 0 {
				continue
			}

			mixedness := 1.0 - math.Abs(float64(landNeighbors-oceanNeighbors))/float64(neighborCount)
			shorelineFactor := 1.0 - SmoothStep(60, coastalWindow, math.Abs(elev))
			exposure := 0.0
			if r < len(coastalExposure) {
				exposure = coastalExposure[r]
			}
			blend := (0.09 + 0.20*mixedness + 0.20*(1.0-exposure)) * shorelineFactor
			if blend <= 0 {
				continue
			}

			neighborMean := sum / float64(neighborCount)
			buffer[r] = elev*(1.0-blend) + neighborMean*blend
		}

		copy(elevation, buffer)
	}
}

// ReinforceTectonicMountains restores some high-relief tail after hypsometric
// remapping by boosting only tectonically supported uplands.
func ReinforceTectonicMountains(
	elevation []float64,
	rPlate []int,
	plateIsOcean map[int]bool,
	distFromMountain []float64,
	componentMaxMountainDist []float64,
	distFromCollision []float64,
	componentMaxCollisionDist []float64,
	distFromArc []float64,
	componentMaxArcDist []float64,
) {
	for r := range elevation {
		if plateIsOcean[rPlate[r]] || elevation[r] <= 800 {
			continue
		}

		genericSupport := normalizedSeedSupport(distFromMountain[r], componentMaxMountainDist[r])
		collisionSupport := normalizedSeedSupport(distFromCollision[r], componentMaxCollisionDist[r])
		arcSupport := normalizedSeedSupport(distFromArc[r], componentMaxArcDist[r])
		if genericSupport <= 0 && collisionSupport <= 0 && arcSupport <= 0 {
			continue
		}

		uplandFactor := SmoothStep(1200, 3000, elevation[r])
		if uplandFactor <= 0 {
			continue
		}

		plateauFactor := SmoothStep(900, 2600, elevation[r])
		arcPeakFactor := SmoothStep(1500, 3400, elevation[r])
		boost := 120*genericSupport*uplandFactor +
			240*collisionSupport*plateauFactor +
			260*collisionSupport*uplandFactor +
			360*arcSupport*arcPeakFactor
		elevation[r] += boost
	}
}

func normalizedSeedSupport(distance float64, maxDistance float64) float64 {
	if maxDistance <= 0 || math.IsInf(distance, 1) {
		return 0
	}
	norm := distance / maxDistance
	if norm < 0 {
		norm = 0
	}
	if norm > 1 {
		norm = 1
	}
	return 1.0 - SmoothStep(0, 1, norm)
}

// FBMNoiseWithFreq generates fractal Brownian motion noise with configurable base frequency
func FBMNoiseWithFreq(pos Vector3D, seed int64, baseFreq float64, octaves int) float64 {
	persistence := 0.5
	sum := 0.0
	sumOfAmplitudes := 0.0

	for octave := 0; octave < octaves; octave++ {
		amplitude := math.Pow(persistence, float64(octave))
		frequency := baseFreq * float64(int(1)<<octave)
		sum += amplitude * SimplexNoise3D(pos.X*frequency, pos.Y*frequency, pos.Z*frequency, seed+int64(octave)*1000)
		sumOfAmplitudes += amplitude
	}

	return sum / sumOfAmplitudes
}

// applySubductionProfile creates gradual elevation from trench to volcanic arc
// Real subduction zones have: trench → accretionary wedge → forearc basin → volcanic arc → back-arc
func applySubductionProfile(
	sites []Vector3D,
	cells []VoronoiCell,
	trenchR map[int]bool,
	arcR map[int]bool,
	rPlate []int,
	plateIsOcean map[int]bool,
	elevation []float64,
) {
	numRegions := len(sites)

	// Apply trench depression
	for trenchRegion := range trenchR {
		elevation[trenchRegion] -= 0.3 // Reduced from 0.5
	}

	distFromTrench := computeContinentalSeedDistance(sites, cells, trenchR, rPlate, plateIsOcean, VolcanoDistanceRadians*2.0)
	distFromArc := computeContinentalSeedDistance(sites, cells, arcR, rPlate, plateIsOcean, VolcanoDistanceRadians*1.5)

	// Apply elevation profile based on distance from nearest trench
	for r := 0; r < numRegions; r++ {
		if plateIsOcean[rPlate[r]] {
			continue
		}

		trenchDist := distFromTrench[r]
		arcDist := distFromArc[r]
		adjustment := 0.0

		if !math.IsInf(trenchDist, 1) && trenchDist <= VolcanoDistanceRadians*1.5 {
			// Normalize: 0 = at coast, 1 = at volcanic arc distance
			t := trenchDist / VolcanoDistanceRadians
			if t > 1.0 {
				t = 1.0
			}

			// t=0: trench/coastal depression
			// t=0.4-0.7: forearc basin
			// t>0.7: broad rise toward arc/back-arc plateau
			if t < 0.4 {
				adjustment += -0.04 * (1 - t/0.4)
			} else if t < 0.7 {
				basinT := (t - 0.4) / 0.3
				adjustment += -0.03 * (1 - 4*(basinT-0.5)*(basinT-0.5))
			} else {
				backArcT := (t - 0.7) / 0.3
				adjustment += 0.04*backArcT + 0.03*backArcT*backArcT
			}
		}

		if !math.IsInf(arcDist, 1) && arcDist <= ArcHalfWidthRadians*3.0 {
			arcT := arcDist / (ArcHalfWidthRadians * 3.0)
			if arcT > 1.0 {
				arcT = 1.0
			}
			adjustment += 0.12 * (1 - arcT*arcT)
		}

		if adjustment != 0 {
			elevation[r] += adjustment
		}
	}
}

func computeContinentalSeedDistance(
	sites []Vector3D,
	cells []VoronoiCell,
	seedR map[int]bool,
	rPlate []int,
	plateIsOcean map[int]bool,
	maxDistance float64,
) []float64 {
	distFromSeed := make([]float64, len(sites))
	for i := range distFromSeed {
		distFromSeed[i] = math.Inf(1)
	}

	type queueItem struct {
		region    int
		sourcePos Vector3D
	}

	queue := make([]queueItem, 0, len(seedR))
	for seed := range seedR {
		if seed < 0 || seed >= len(cells) {
			continue
		}
		for _, neighborIdx := range cells[seed].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR < 0 || neighborR >= len(cells) || plateIsOcean[rPlate[neighborR]] {
				continue
			}
			dist := Distance(sites[neighborR], sites[seed])
			if dist < distFromSeed[neighborR] {
				distFromSeed[neighborR] = dist
				queue = append(queue, queueItem{region: neighborR, sourcePos: sites[seed]})
			}
		}
	}

	for queueIdx := 0; queueIdx < len(queue); queueIdx++ {
		item := queue[queueIdx]
		if distFromSeed[item.region] > maxDistance {
			continue
		}

		for _, neighborIdx := range cells[item.region].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR < 0 || neighborR >= len(cells) || plateIsOcean[rPlate[neighborR]] {
				continue
			}
			neighborDist := Distance(sites[neighborR], item.sourcePos)
			if neighborDist < distFromSeed[neighborR] {
				distFromSeed[neighborR] = neighborDist
				queue = append(queue, queueItem{region: neighborR, sourcePos: item.sourcePos})
			}
		}
	}

	return distFromSeed
}
