package terrain

// Hotspot placement and chain tracing
// Uses Euler pole rotation for realistic curved chain paths (like Hawaii-Emperor chain)

import (
	"math"
	"math/rand"
)

// PlaceHotspots creates hotspots and traces their island chains
// Uses plate rotations (Euler poles) for realistic curved chain paths
func PlaceHotspots(
	sites []Vector3D,
	cells []VoronoiCell,
	rPlate []int,
	plateRot map[int]PlateRotation,
	plateIsOcean map[int]bool,
	rng *rand.Rand,
) []HotspotChain {
	numRegions := len(sites)
	chains := make([]HotspotChain, 0, HotspotsPerPlanet)

	for i := 0; i < HotspotsPerPlanet; i++ {
		// Pick a random location on the sphere
		hotspotPos := randomPointOnSphere(rng)

		// Find the nearest cell to this position
		hotspotCell := findNearestCell(sites, hotspotPos)
		if hotspotCell < 0 || hotspotCell >= numRegions {
			continue
		}

		plateID := rPlate[hotspotCell]
		isOceanic := plateIsOcean[plateID]

		hotspot := Hotspot{
			Position:  hotspotPos,
			PlateID:   plateID,
			IsOceanic: isOceanic,
		}

		// Trace the chain backward along plate rotation (curved path)
		// Both oceanic and continental hotspots create chains
		chain := traceHotspotChain(sites, cells, rPlate, plateRot, plateIsOcean, hotspot, rng)
		chain.IsOceanic = isOceanic
		if len(chain.Islands) >= MinChainLength {
			chains = append(chains, chain)
		}
	}

	return chains
}

// traceHotspotChain traces an island chain backward along plate rotation
// Uses Euler pole rotation for realistic curved paths (like Hawaii-Emperor chain)
// Chain length varies with plate velocity (faster plates = longer chains)
// Islands get progressively older (and lower) as we trace back
// Includes randomness in spacing, island formation probability, and size
func traceHotspotChain(
	sites []Vector3D,
	cells []VoronoiCell,
	rPlate []int,
	plateRot map[int]PlateRotation,
	plateIsOcean map[int]bool,
	hotspot Hotspot,
	rng *rand.Rand,
) HotspotChain {
	numRegions := len(sites)
	chain := HotspotChain{
		Hotspot: hotspot,
		Islands: make([]HotspotIsland, 0, 20),
	}

	// Get plate rotation
	rotation := plateRot[hotspot.PlateID]

	// Check if plate has significant rotation
	if math.Abs(rotation.AngularVelocity) < 1e-6 {
		// Plate not moving - no chain forms
		return chain
	}

	// Chain length scales with angular velocity + hotspot lifetime
	// Faster plates have traveled further, creating longer chains
	// Hotspot lifetime varies greatly - some are young (short chains), others
	// have been active for tens of millions of years (long chains like Hawaii-Emperor)
	velocityFactor := math.Abs(rotation.AngularVelocity)

	// Hotspot lifetime: use a distribution biased toward shorter chains
	// (most hotspots are relatively young, few are ancient)
	// Range: 0.2x to 2.5x with median around 0.7x
	lifetimeRoll := rng.Float64()
	var hotspotLifetime float64
	if lifetimeRoll < 0.3 {
		// 30% young hotspots: 0.2x to 0.5x (short chains, few islands)
		hotspotLifetime = 0.2 + 0.3*rng.Float64()
	} else if lifetimeRoll < 0.7 {
		// 40% middle-aged: 0.5x to 1.2x (moderate chains)
		hotspotLifetime = 0.5 + 0.7*rng.Float64()
	} else if lifetimeRoll < 0.9 {
		// 20% mature: 1.2x to 1.8x (long chains)
		hotspotLifetime = 1.2 + 0.6*rng.Float64()
	} else {
		// 10% ancient: 1.8x to 2.5x (very long chains like Hawaii-Emperor)
		hotspotLifetime = 1.8 + 0.7*rng.Float64()
	}

	maxChainLength := (MinChainLengthRadians + (BaseChainLengthRadians-MinChainLengthRadians)*velocityFactor) * hotspotLifetime

	// Create a modified pole for this specific hotspot chain
	// The curvature of the chain depends on the angular distance from pole to hotspot
	// - Pole at 90° from hotspot = great circle (straight)
	// - Pole closer to hotspot = tighter curve
	// We shift the pole toward or away from the hotspot to create variety
	pole := rotation.Pole

	// Calculate angular distance from pole to hotspot
	dotPoleHotspot := pole.X*hotspot.Position.X + pole.Y*hotspot.Position.Y + pole.Z*hotspot.Position.Z

	// Shift pole toward or away from hotspot for curvature
	// Random shift: -0.4 to +0.4 (negative = pole closer = tighter curve)
	poleShift := (rng.Float64() - 0.5) * 0.8
	pole.X += hotspot.Position.X * poleShift
	pole.Y += hotspot.Position.Y * poleShift
	pole.Z += hotspot.Position.Z * poleShift

	// Renormalize pole
	poleMag := math.Sqrt(pole.X*pole.X + pole.Y*pole.Y + pole.Z*pole.Z)
	if poleMag > 0.1 {
		pole.X /= poleMag
		pole.Y /= poleMag
		pole.Z /= poleMag
	} else {
		pole = rotation.Pole // Fallback if degenerate
	}

	_ = dotPoleHotspot // Suppress unused warning

	// To trace the chain backward, we rotate in the OPPOSITE direction
	backwardRotation := PlateRotation{
		Pole:            pole,
		AngularVelocity: -rotation.AngularVelocity,
	}

	// Start at the hotspot position
	currentPos := hotspot.Position

	// Track visited cells to avoid duplicates
	visited := make(map[int]bool)

	// Base spacing with variation range
	baseSpacing := IslandSpacingRadians
	minSpacing := baseSpacing * 0.5 // Clustered islands
	maxSpacing := baseSpacing * 2.5 // Gaps in chain

	// Orthogonal jitter - perpendicular displacement for natural wandering
	// Amount scales with spacing to maintain proportions
	jitterScale := baseSpacing * 0.3 // Up to 30% of spacing as sideways wander

	// Direction change (bend) parameters
	// Longer chains have higher probability of experiencing a plate motion change
	// Hawaiian-Emperor bend is ~60°, we'll use 30-70° range
	bendProbability := 0.0
	if hotspotLifetime > 1.5 {
		bendProbability = 0.4 // 40% chance for ancient chains
	} else if hotspotLifetime > 1.0 {
		bendProbability = 0.2 // 20% chance for mature chains
	} else if hotspotLifetime > 0.7 {
		bendProbability = 0.08 // 8% chance for middle-aged
	}
	hasBend := rng.Float64() < bendProbability
	bendPosition := 0.3 + 0.4*rng.Float64() // Bend occurs 30-70% along chain
	bendApplied := false

	chainLength := 0.0
	stepsSinceLastIsland := 0

	for chainLength < maxChainLength {
		// Find the nearest cell to current position
		cellIdx := findNearestCell(sites, currentPos)
		if cellIdx < 0 || cellIdx >= numRegions {
			break
		}

		// Check if we've left the appropriate plate type
		// Oceanic hotspots trace through oceanic plates, continental through continental
		cellIsOceanic := plateIsOcean[rPlate[cellIdx]]
		if hotspot.IsOceanic && !cellIsOceanic {
			break // Oceanic hotspot hit continental plate
		}
		if !hotspot.IsOceanic && cellIsOceanic {
			break // Continental hotspot hit oceanic plate
		}

		// Random spacing for this step
		thisSpacing := minSpacing + rng.Float64()*(maxSpacing-minSpacing)

		// Skip if already visited
		if visited[cellIdx] {
			currentPos = backwardRotation.RotatePoint(currentPos, thisSpacing)
			chainLength += thisSpacing
			stepsSinceLastIsland++
			continue
		}

		// Probability of forming an island at this position
		// Higher probability if we haven't had an island in a while (ensures some continuity)
		// Lower probability allows for occasional gaps
		formationProb := 0.75 + 0.08*float64(stepsSinceLastIsland) // 75% base, increases with gaps
		if formationProb > 0.98 {
			formationProb = 0.98
		}

		if rng.Float64() < formationProb {
			visited[cellIdx] = true
			stepsSinceLastIsland = 0

			// Add island with age based on chain position
			age := chainLength / maxChainLength
			chain.Islands = append(chain.Islands, HotspotIsland{
				CellIndex: cellIdx,
				Age:       age,
			})
		} else {
			stepsSinceLastIsland++
		}

		// Check for direction change (bend) at this position
		chainProgress := chainLength / maxChainLength
		if hasBend && !bendApplied && chainProgress >= bendPosition {
			// Apply a significant change in pole direction (30-70 degrees)
			bendAngle := (30.0 + 40.0*rng.Float64()) * math.Pi / 180.0

			// Rotate the pole around the current position
			// This simulates a change in plate motion direction
			sinB, cosB := math.Sin(bendAngle), math.Cos(bendAngle)

			// Create rotation around currentPos axis
			oldPole := backwardRotation.Pole
			// Rodrigues' rotation formula components
			dot := oldPole.X*currentPos.X + oldPole.Y*currentPos.Y + oldPole.Z*currentPos.Z
			crossX := currentPos.Y*oldPole.Z - currentPos.Z*oldPole.Y
			crossY := currentPos.Z*oldPole.X - currentPos.X*oldPole.Z
			crossZ := currentPos.X*oldPole.Y - currentPos.Y*oldPole.X

			newPole := Vector3D{
				X: oldPole.X*cosB + crossX*sinB + currentPos.X*dot*(1-cosB),
				Y: oldPole.Y*cosB + crossY*sinB + currentPos.Y*dot*(1-cosB),
				Z: oldPole.Z*cosB + crossZ*sinB + currentPos.Z*dot*(1-cosB),
			}

			// Normalize new pole
			poleMag := math.Sqrt(newPole.X*newPole.X + newPole.Y*newPole.Y + newPole.Z*newPole.Z)
			if poleMag > 0.1 {
				backwardRotation.Pole = Vector3D{
					X: newPole.X / poleMag,
					Y: newPole.Y / poleMag,
					Z: newPole.Z / poleMag,
				}
			}
			bendApplied = true
		}

		// Rotate backward around the Euler pole
		currentPos = backwardRotation.RotatePoint(currentPos, thisSpacing)
		chainLength += thisSpacing

		// Apply orthogonal jitter - sideways displacement perpendicular to travel
		// Find perpendicular direction by crossing current position with pole
		perpX := currentPos.Y*backwardRotation.Pole.Z - currentPos.Z*backwardRotation.Pole.Y
		perpY := currentPos.Z*backwardRotation.Pole.X - currentPos.X*backwardRotation.Pole.Z
		perpZ := currentPos.X*backwardRotation.Pole.Y - currentPos.Y*backwardRotation.Pole.X
		perpMag := math.Sqrt(perpX*perpX + perpY*perpY + perpZ*perpZ)

		if perpMag > 0.01 {
			// Normalize and apply random jitter
			jitterAmount := (rng.Float64() - 0.5) * 2 * jitterScale
			currentPos.X += (perpX / perpMag) * jitterAmount
			currentPos.Y += (perpY / perpMag) * jitterAmount
			currentPos.Z += (perpZ / perpMag) * jitterAmount
		}

		// Normalize to stay on unit sphere
		mag := math.Sqrt(currentPos.X*currentPos.X + currentPos.Y*currentPos.Y + currentPos.Z*currentPos.Z)
		if mag > 0 {
			currentPos.X /= mag
			currentPos.Y /= mag
			currentPos.Z /= mag
		}
	}

	return chain
}

// randomPointOnSphere generates a uniformly random point on the unit sphere
func randomPointOnSphere(rng *rand.Rand) Vector3D {
	// Use spherical coordinates with uniform distribution
	z := 2*rng.Float64() - 1 // Range [-1, 1]
	theta := 2 * math.Pi * rng.Float64()
	r := math.Sqrt(1 - z*z)

	return Vector3D{
		X: r * math.Cos(theta),
		Y: r * math.Sin(theta),
		Z: z,
	}
}

// findNearestCell finds the cell index nearest to a given position
// Uses simple linear search - could be optimized with spatial index if needed
func findNearestCell(sites []Vector3D, pos Vector3D) int {
	bestIdx := -1
	bestDist := math.Inf(1)

	for i, site := range sites {
		dist := Distance(site, pos)
		if dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}

	return bestIdx
}
