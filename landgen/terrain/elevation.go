package terrain

// Elevation computation using distance field algorithm
// Computes elevation from boundary seeds using inverse-distance formula

import (
	"fmt"
	"math"
	"math/rand"
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
) ([]float64, map[int]bool) {
	numRegions := len(sites)
	rng := rand.New(rand.NewSource(seed + 12345))
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

	// Compute distance fields
	rDistanceA := AssignDistanceField(cells, seeds.Mountain, seeds.Ocean, rng)
	rDistanceB := AssignDistanceField(cells, seeds.Ocean, seeds.Coastline, rng)
	rDistanceC := AssignDistanceField(cells, seeds.Coastline, stopR, rng)

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

	// Compute forearc elevation profile for subduction zones
	// This creates gradual transition: trench → forearc basin → volcanic arc
	applySubductionProfile(sites, cells, seeds.Trench, seeds.Mountain, rPlate, plateIsOcean, elevation)

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

	return elevation, seeds.Coastline
}

// AssignDistanceField computes distance from seeds using randomized BFS
func AssignDistanceField(cells []VoronoiCell, seedsR map[int]bool, stopR map[int]bool, rng *rand.Rand) []float64 {
	numRegions := len(cells)
	rDistance := make([]float64, numRegions)
	for i := range rDistance {
		rDistance[i] = math.Inf(1)
	}

	// Initialize queue with seeds
	var queue []int
	for r := range seedsR {
		queue = append(queue, r)
		rDistance[r] = 0
	}

	// Randomized BFS
	for queueOut := 0; queueOut < len(queue); queueOut++ {
		pos := queueOut + rng.Intn(len(queue)-queueOut)
		currentR := queue[pos]
		queue[pos] = queue[queueOut]

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

// ApplyBimodalElevation applies bimodal elevation with continental slope
func ApplyBimodalElevation(
	elevation []float64,
	distFromCoast []float64,
	rPlate []int,
	plateIsOcean map[int]bool,
	maxDist float64,
) {
	numRegions := len(elevation)

	for r := 0; r < numRegions; r++ {
		plate := rPlate[r]
		e := elevation[r]

		if plateIsOcean[plate] {
			// Ocean: base at -0.7 (abyssal), variation from ridges/trenches
			elevation[r] = -0.7 + e*0.3
		} else {
			// Continental: base elevation depends on distance from coast
			normalizedDist := distFromCoast[r] / maxDist
			if math.IsInf(distFromCoast[r], 1) {
				normalizedDist = 0.5
			}

			// Base: 0.1 at coast, 0.5 at interior
			baseElev := 0.1 + normalizedDist*0.4
			elevation[r] = baseElev + e*0.4
		}
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

// applySubductionProfile creates gradual elevation from trench to volcanic arc
// Real subduction zones have: trench → accretionary wedge → forearc basin → volcanic arc → back-arc
func applySubductionProfile(
	sites []Vector3D,
	cells []VoronoiCell,
	trenchR map[int]bool,
	mountainR map[int]bool,
	rPlate []int,
	plateIsOcean map[int]bool,
	elevation []float64,
) {
	numRegions := len(sites)

	// Apply trench depression
	for trenchRegion := range trenchR {
		elevation[trenchRegion] -= 0.3 // Reduced from 0.5
	}

	// Compute distance from nearest trench using BFS (O(regions) instead of O(regions*trenches))
	distFromTrench := make([]float64, numRegions)
	for i := range distFromTrench {
		distFromTrench[i] = math.Inf(1)
	}

	// BFS from all trenches simultaneously - track source trench in queue
	type queueItem struct {
		region   int
		trenchR  int     // originating trench
		trenchPos Vector3D
	}
	var queue []queueItem

	// Seed with continental neighbors of each trench
	for trenchRegion := range trenchR {
		trenchPos := sites[trenchRegion]
		for _, neighborIdx := range cells[trenchRegion].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR < numRegions && !plateIsOcean[rPlate[neighborR]] {
				dist := Distance(sites[neighborR], trenchPos)
				if dist < distFromTrench[neighborR] {
					distFromTrench[neighborR] = dist
					queue = append(queue, queueItem{neighborR, trenchRegion, trenchPos})
				}
			}
		}
	}

	// Process BFS queue
	for queueIdx := 0; queueIdx < len(queue); queueIdx++ {
		item := queue[queueIdx]

		// Only process cells within range
		if distFromTrench[item.region] > VolcanoDistanceRadians*2.0 {
			continue
		}

		for _, neighborIdx := range cells[item.region].NeighborSiteIndices {
			neighborR := int(neighborIdx)
			if neighborR >= numRegions || plateIsOcean[rPlate[neighborR]] {
				continue
			}

			neighborDist := Distance(sites[neighborR], item.trenchPos)
			if neighborDist < distFromTrench[neighborR] {
				distFromTrench[neighborR] = neighborDist
				queue = append(queue, queueItem{neighborR, item.trenchR, item.trenchPos})
			}
		}
	}

	// Apply elevation profile based on distance from nearest trench
	for r := 0; r < numRegions; r++ {
		if plateIsOcean[rPlate[r]] {
			continue
		}

		dist := distFromTrench[r]
		if dist > VolcanoDistanceRadians*1.5 || math.IsInf(dist, 1) {
			continue
		}

		// Normalize: 0 = at coast, 1 = at volcanic arc distance
		t := dist / VolcanoDistanceRadians
		if t > 1.0 {
			t = 1.0
		}

		// Gentler elevation profile (reduced from previous values)
		// t=0: slight coastal depression
		// t=0.4-0.6: forearc basin
		// t=0.8-1.0: volcanic arc rise
		var adjustment float64
		if t < 0.4 {
			// Gentle slope from coast
			adjustment = -0.03 * (1 - t/0.4)
		} else if t < 0.7 {
			// Forearc basin - very subtle
			basinT := (t - 0.4) / 0.3
			adjustment = -0.02 * (1 - 4*(basinT-0.5)*(basinT-0.5))
		} else {
			// Rising toward volcanic arc - gentler
			arcT := (t - 0.7) / 0.3
			adjustment = 0.08 * arcT * arcT
		}

		elevation[r] += adjustment
	}
}
