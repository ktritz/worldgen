package terrain

// Collision-based elevation generation based on Red Blob Games approach.
// https://www.redblobgames.com/x/1843-planet-generation/
//
// Key insight: Elevation is driven by plate tectonics, not noise.
// - Mountains form at convergent boundaries (plates pushing together)
// - Ocean basins form at divergent boundaries (plates pulling apart)
// - Elevation is calculated via distance fields from collision zones
// - Land coverage is controlled by finding the threshold that gives target %

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"worldgen/landgen/tectonics"
)

// TectonicPlate is imported from tectonics package
type TectonicPlate = tectonics.TectonicPlate

// CollisionZones holds the classified boundary regions
type CollisionZones struct {
	Mountain  []int // Sites where plates collide (high elevation)
	Coastline []int // Sites at ocean-land transitions
	Ocean     []int // Sites in ocean basins (low elevation)
}

// CollisionElevationConfig holds parameters for elevation generation
type CollisionElevationConfig struct {
	// Land coverage control
	TargetLandFraction float64 // Target land coverage (0.0-1.0), e.g., 0.29 for Earth
	OceanPlateRatio    float64 // Fraction of plates that are oceanic (affects distribution)

	// Collision detection
	CollisionThreshold float64 // Velocity threshold for collision detection

	// Elevation scaling (meters)
	MaxMountainHeight float64 // Maximum mountain elevation (e.g., 8848 for Everest)
	MaxOceanDepth     float64 // Maximum ocean depth (e.g., -10994 for Mariana)
	MeanLandElevation float64 // Target mean land elevation (e.g., 840m for Earth)
	MeanOceanDepth    float64 // Target mean ocean depth (e.g., -3688m for Earth)

	// Noise
	NoiseAmplitude float64 // Fraction of elevation range to add as noise (0.0-0.2)
	Seed           int64
}

// DefaultCollisionElevationConfig returns Earth-like defaults
func DefaultCollisionElevationConfig() CollisionElevationConfig {
	return CollisionElevationConfig{
		// Earth-like land coverage
		TargetLandFraction: 0.29, // 29% land
		OceanPlateRatio:    0.6,  // 60% oceanic plates

		// Collision detection
		CollisionThreshold: 0.001,

		// Earth elevation ranges (meters)
		MaxMountainHeight: 8848.0,   // Everest
		MaxOceanDepth:     -10994.0, // Mariana Trench
		MeanLandElevation: 840.0,    // Earth average
		MeanOceanDepth:    -3688.0,  // Earth average

		// Noise
		NoiseAmplitude: 0.05, // 5% noise
		Seed:           42,
	}
}

// CollisionElevationResult contains the output of elevation generation
type CollisionElevationResult struct {
	Elevation     []float64 // Elevation in meters for each site
	IsLand        []bool    // True if site is land
	LandFraction  float64   // Actual land coverage achieved
	SeaLevel      float64   // Threshold used (in raw units before scaling)
	PlateIsOcean  []bool    // Which plates are oceanic
	CollisionZones CollisionZones
}

// GenerateCollisionElevation generates elevation based on plate tectonics.
// Returns elevation in meters with guaranteed land coverage matching target.
func GenerateCollisionElevation(
	sites []Vector3D,
	cells []VoronoiCell,
	plates []TectonicPlate,
	siteToPlate []int32,
	config CollisionElevationConfig,
) CollisionElevationResult {
	numSites := len(sites)
	rng := rand.New(rand.NewSource(config.Seed))

	fmt.Println("=== COLLISION-BASED ELEVATION GENERATION ===")
	fmt.Printf("Target land coverage: %.1f%%\n", config.TargetLandFraction*100)

	// Step 1: Assign plates as oceanic or continental
	fmt.Println("Step 1: Assigning plate types...")
	plateIsOcean := assignOceanicPlates(plates, config.OceanPlateRatio, rng)
	oceanCount := 0
	for _, isOcean := range plateIsOcean {
		if isOcean {
			oceanCount++
		}
	}
	fmt.Printf("  %d oceanic, %d continental plates\n", oceanCount, len(plates)-oceanCount)

	// Step 2: Find collision zones based on plate motion
	fmt.Println("Step 2: Detecting plate collisions...")
	zones := findCollisionZones(sites, cells, plates, siteToPlate, plateIsOcean, config.CollisionThreshold)
	fmt.Printf("  Mountains: %d, Coastlines: %d, Ocean seeds: %d\n",
		len(zones.Mountain), len(zones.Coastline), len(zones.Ocean))

	// Step 3: Calculate distance fields
	fmt.Println("Step 3: Computing distance fields...")
	distToMountain := assignDistanceField(cells, zones.Mountain, zones.Ocean)
	distToOcean := assignDistanceField(cells, zones.Ocean, zones.Coastline)
	distToCoast := assignDistanceField(cells, zones.Coastline, mergeSlices(zones.Mountain, zones.Ocean))

	// Step 4: Calculate raw elevation
	// Key insight: continental plates should be inherently higher than oceanic
	fmt.Println("Step 4: Computing raw elevation...")
	rawElevation := make([]float64, numSites)

	// First pass: base elevation from plate type + distance fields
	for i := 0; i < numSites; i++ {
		plateIdx := int(siteToPlate[i])
		isOnContinentalPlate := plateIdx >= 0 && plateIdx < len(plateIsOcean) && !plateIsOcean[plateIdx]

		a := distToMountain[i] + 0.001
		b := distToOcean[i] + 0.001
		c := distToCoast[i] + 0.001

		// Distance-based component (creates mountains at collisions)
		var distComponent float64
		if math.IsInf(a, 1) && math.IsInf(b, 1) {
			distComponent = 0.0
		} else {
			distComponent = (1/a - 1/b) / (1/a + 1/b + 1/c)
		}

		// Plate type component: continental plates get a significant boost
		var plateComponent float64
		if isOnContinentalPlate {
			plateComponent = 0.5 // Continental crust is inherently higher
		} else {
			plateComponent = -0.3 // Oceanic crust is inherently lower
		}

		// Combine: plate type dominates, distance fields add variation
		rawElevation[i] = plateComponent*0.7 + distComponent*0.3
	}

	// Second pass: smooth elevation to reduce noise (simple neighbor averaging)
	smoothed := make([]float64, numSites)
	for i, cell := range cells {
		sum := rawElevation[i] * 2 // Weight self more
		count := 2.0
		for _, neighborIdx := range cell.NeighborSiteIndices {
			if int(neighborIdx) < numSites {
				sum += rawElevation[neighborIdx]
				count++
			}
		}
		smoothed[i] = sum / count
	}
	rawElevation = smoothed

	// Step 5: Find sea level threshold for target land coverage
	fmt.Println("Step 5: Finding sea level for target land coverage...")
	seaLevel := findSeaLevelThreshold(rawElevation, config.TargetLandFraction)
	fmt.Printf("  Sea level threshold: %.4f (raw units)\n", seaLevel)

	// Step 6: Classify land/ocean and scale to meters
	fmt.Println("Step 6: Scaling to meters...")
	elevation := make([]float64, numSites)
	isLand := make([]bool, numSites)

	// Find raw elevation range for scaling
	var rawMin, rawMax float64 = rawElevation[0], rawElevation[0]
	for _, e := range rawElevation {
		if e < rawMin {
			rawMin = e
		}
		if e > rawMax {
			rawMax = e
		}
	}

	landCount := 0
	for i := 0; i < numSites; i++ {
		isLand[i] = rawElevation[i] > seaLevel

		if isLand[i] {
			// Scale land elevation: seaLevel -> 0m, rawMax -> MaxMountainHeight
			// Use power curve for more realistic distribution
			t := (rawElevation[i] - seaLevel) / (rawMax - seaLevel + 0.001)
			t = math.Pow(t, 0.7) // Compress high elevations
			elevation[i] = t * config.MaxMountainHeight
			landCount++
		} else {
			// Scale ocean depth: seaLevel -> 0m, rawMin -> MaxOceanDepth
			t := (seaLevel - rawElevation[i]) / (seaLevel - rawMin + 0.001)
			t = math.Pow(t, 0.8) // Slightly compress deep ocean
			elevation[i] = -t * math.Abs(config.MaxOceanDepth)
		}
	}

	actualLandFraction := float64(landCount) / float64(numSites)

	// Report statistics
	var minE, maxE, sumLand, sumOcean float64 = elevation[0], elevation[0], 0, 0
	landN, oceanN := 0, 0
	for i, e := range elevation {
		if e < minE {
			minE = e
		}
		if e > maxE {
			maxE = e
		}
		if isLand[i] {
			sumLand += e
			landN++
		} else {
			sumOcean += e
			oceanN++
		}
	}

	meanLand := sumLand / float64(landN+1)
	meanOcean := sumOcean / float64(oceanN+1)

	fmt.Printf("  Elevation range: %.0fm to %.0fm\n", minE, maxE)
	fmt.Printf("  Land coverage: %.1f%% (target: %.1f%%)\n", actualLandFraction*100, config.TargetLandFraction*100)
	fmt.Printf("  Mean land elevation: %.0fm (target: %.0fm)\n", meanLand, config.MeanLandElevation)
	fmt.Printf("  Mean ocean depth: %.0fm (target: %.0fm)\n", meanOcean, config.MeanOceanDepth)

	return CollisionElevationResult{
		Elevation:      elevation,
		IsLand:         isLand,
		LandFraction:   actualLandFraction,
		SeaLevel:       seaLevel,
		PlateIsOcean:   plateIsOcean,
		CollisionZones: zones,
	}
}

// findSeaLevelThreshold finds the elevation threshold that gives target land coverage
func findSeaLevelThreshold(elevation []float64, targetLandFraction float64) float64 {
	// Sort elevations to find percentile
	sorted := make([]float64, len(elevation))
	copy(sorted, elevation)
	sort.Float64s(sorted)

	// Land fraction = fraction above threshold
	// So we want the (1 - targetLandFraction) percentile
	oceanFraction := 1.0 - targetLandFraction
	index := int(oceanFraction * float64(len(sorted)-1))
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// assignOceanicPlates randomly assigns plates as oceanic or continental
func assignOceanicPlates(plates []TectonicPlate, oceanRatio float64, rng *rand.Rand) []bool {
	isOcean := make([]bool, len(plates))
	for i := range plates {
		isOcean[i] = rng.Float64() < oceanRatio
	}
	return isOcean
}

// findCollisionZones detects where plates are colliding based on velocity
func findCollisionZones(
	sites []Vector3D,
	cells []VoronoiCell,
	plates []TectonicPlate,
	siteToPlate []int32,
	plateIsOcean []bool,
	collisionThreshold float64,
) CollisionZones {
	var zones CollisionZones
	deltaTime := 1e-2

	for siteIdx, cell := range cells {
		myPlate := int(siteToPlate[siteIdx])
		if myPlate < 0 || myPlate >= len(plates) {
			continue
		}

		myPos := sites[siteIdx]
		myVel := plates[myPlate].GetVelocityAtPoint(myPos)

		bestCompression := math.Inf(-1)
		bestNeighborPlate := -1

		for _, neighborIdx := range cell.NeighborSiteIndices {
			if int(neighborIdx) >= len(siteToPlate) {
				continue
			}
			neighborPlate := int(siteToPlate[neighborIdx])
			if neighborPlate == myPlate || neighborPlate < 0 || neighborPlate >= len(plates) {
				continue
			}

			neighborPos := sites[neighborIdx]
			neighborVel := plates[neighborPlate].GetVelocityAtPoint(neighborPos)

			distBefore := vectorDistance(myPos, neighborPos)
			myPosAfter := vectorAdd(myPos, vectorScale(myVel, deltaTime))
			neighborPosAfter := vectorAdd(neighborPos, vectorScale(neighborVel, deltaTime))
			distAfter := vectorDistance(myPosAfter, neighborPosAfter)

			compression := distBefore - distAfter
			if compression > bestCompression {
				bestCompression = compression
				bestNeighborPlate = neighborPlate
			}
		}

		if bestNeighborPlate == -1 {
			continue
		}

		collided := bestCompression > collisionThreshold*deltaTime
		myIsOcean := plateIsOcean[myPlate]
		neighborIsOcean := plateIsOcean[bestNeighborPlate]

		if myIsOcean && neighborIsOcean {
			if collided {
				zones.Coastline = append(zones.Coastline, siteIdx)
			} else {
				zones.Ocean = append(zones.Ocean, siteIdx)
			}
		} else if !myIsOcean && !neighborIsOcean {
			if collided {
				zones.Mountain = append(zones.Mountain, siteIdx)
			}
		} else {
			if collided {
				zones.Mountain = append(zones.Mountain, siteIdx)
			} else {
				zones.Coastline = append(zones.Coastline, siteIdx)
			}
		}
	}

	// Add plate centers to zones - THIS IS CRITICAL
	// Continental plate centers → coastline seeds (makes whole plate tend toward land)
	// Oceanic plate centers → ocean seeds (makes whole plate tend toward ocean)
	// This ensures cohesive continents, not just rings around collision zones
	// Find first cell of each plate as approximate center
	plateFound := make([]bool, len(plates))
	for siteIdx, pIdx := range siteToPlate {
		plateIdx := int(pIdx)
		if plateIdx >= 0 && plateIdx < len(plates) && !plateFound[plateIdx] {
			plateFound[plateIdx] = true
			if plateIsOcean[plateIdx] {
				zones.Ocean = append(zones.Ocean, siteIdx)
			} else {
				zones.Coastline = append(zones.Coastline, siteIdx)
			}
		}
	}

	return zones
}

// assignDistanceField computes distance from seeds using randomized BFS
func assignDistanceField(cells []VoronoiCell, seeds []int, stopAt []int) []float64 {
	numSites := len(cells)
	distance := make([]float64, numSites)
	for i := range distance {
		distance[i] = math.Inf(1)
	}

	stopSet := make(map[int]bool)
	for _, s := range stopAt {
		stopSet[s] = true
	}

	queue := make([]int, len(seeds))
	copy(queue, seeds)
	for _, s := range seeds {
		if s >= 0 && s < numSites {
			distance[s] = 0
		}
	}

	rng := rand.New(rand.NewSource(12345))
	for queueOut := 0; queueOut < len(queue); queueOut++ {
		pos := queueOut + rng.Intn(len(queue)-queueOut)
		current := queue[pos]
		queue[pos] = queue[queueOut]

		if current < 0 || current >= numSites {
			continue
		}

		for _, neighborIdx := range cells[current].NeighborSiteIndices {
			neighbor := int(neighborIdx)
			if neighbor >= 0 && neighbor < numSites {
				if math.IsInf(distance[neighbor], 1) && !stopSet[neighbor] {
					distance[neighbor] = distance[current] + 1
					queue = append(queue, neighbor)
				}
			}
		}
	}

	return distance
}

// fbmNoise3D generates fractal Brownian motion noise
func fbmNoise3D(pos Vector3D, seed int64, octaves int) float64 {
	sum := 0.0
	amplitude := 1.0
	frequency := 1.0
	maxValue := 0.0

	for i := 0; i < octaves; i++ {
		n := hashNoise3D(pos.X*frequency, pos.Y*frequency, pos.Z*frequency, seed+int64(i)*1000)
		sum += n * amplitude
		maxValue += amplitude
		amplitude *= 0.5
		frequency *= 2.0
	}

	return sum / maxValue
}

func hashNoise3D(x, y, z float64, seed int64) float64 {
	h := seed
	h ^= int64(x*1000) * 374761393
	h ^= int64(y*1000) * 668265263
	h ^= int64(z*1000) * 2147483647
	h = (h ^ (h >> 13)) * 1274126177
	return float64(h%1000)/500.0 - 1.0
}

func vectorDistance(a, b Vector3D) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func vectorAdd(a, b Vector3D) Vector3D {
	return Vector3D{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}

func vectorScale(v Vector3D, s float64) Vector3D {
	return Vector3D{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

func mergeSlices(a, b []int) []int {
	result := make([]int, len(a)+len(b))
	copy(result, a)
	copy(result[len(a):], b)
	return result
}
