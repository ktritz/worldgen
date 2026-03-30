package terrain

// Hotspot placement and chain tracing
// Uses Euler pole rotation for realistic curved chain paths (like Hawaii-Emperor chain)

import (
	"math"
	"math/rand"
	"sort"
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
	minSpacing := baseSpacing * 0.45 // Clustered islands
	maxSpacing := baseSpacing * 2.9  // Broad gaps in chain

	// Orthogonal jitter - use correlated low-frequency wander instead of
	// independent per-step offsets, so chains meander more naturally and avoid
	// looking like evenly dotted bead strings.
	jitterScale := baseSpacing * 0.45
	lateralOffset := 0.0
	lateralVelocity := (rng.Float64() - 0.5) * jitterScale * 0.4
	spacingFactor := 0.85 + 0.5*rng.Float64()
	spacingDrift := (rng.Float64() - 0.5) * 0.08
	baselineActivity := 0.50 + 0.25*rng.Float64()
	if hotspot.IsOceanic {
		baselineActivity += 0.05
	} else {
		baselineActivity -= 0.05
	}
	activity := baselineActivity
	activityVelocity := (rng.Float64() - 0.5) * 0.06
	quietStepsRemaining := 0
	burstStepsRemaining := 0

	// Some chains preserve a simple motion history, with one or two changes in
	// direction and eruption regime rather than a single stationary process.
	historyEvents := sampleHotspotHistoryEvents(hotspotLifetime, hotspot.IsOceanic, rng)
	nextHistoryEvent := 0

	chainLength := 0.0
	stepsSinceLastIsland := 0
	distanceSinceLastIsland := baseSpacing

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

		// Let spacing and eruptive activity evolve gradually. This produces
		// clustered bursts and quieter runs instead of a chain that drops one
		// similarly sized island every roughly fixed interval.
		spacingDrift += (rng.Float64() - 0.5) * 0.06
		spacingDrift = Clamp(spacingDrift, -0.12, 0.12)
		spacingFactor += spacingDrift
		spacingFactor = Clamp(spacingFactor, 0.65, 1.9)

		activityVelocity += (rng.Float64() - 0.5) * 0.10
		activityVelocity *= 0.72
		activityVelocity = Clamp(activityVelocity, -0.10, 0.10)
		activity += activityVelocity + (baselineActivity-activity)*0.18

		if quietStepsRemaining == 0 && burstStepsRemaining == 0 {
			roll := rng.Float64()
			switch {
			case roll < 0.10:
				quietStepsRemaining = 2 + rng.Intn(5)
			case roll < 0.28:
				burstStepsRemaining = 2 + rng.Intn(5)
			}
		}

		thisSpacing := baseSpacing * spacingFactor
		if quietStepsRemaining > 0 {
			activity -= 0.16 + 0.12*rng.Float64()
			thisSpacing *= 1.30 + 0.40*rng.Float64()
			quietStepsRemaining--
		} else if burstStepsRemaining > 0 {
			activity += 0.18 + 0.12*rng.Float64()
			thisSpacing *= 0.55 + 0.25*rng.Float64()
			burstStepsRemaining--
		}
		activity = Clamp(activity, 0.12, 1.35)
		thisSpacing *= Clamp(1.25-0.40*activity, 0.55, 1.55)
		thisSpacing = Clamp(thisSpacing, minSpacing, maxSpacing)

		// Skip if already visited
		if visited[cellIdx] {
			currentPos = backwardRotation.RotatePoint(currentPos, thisSpacing)
			chainLength += thisSpacing
			stepsSinceLastIsland++
			distanceSinceLastIsland += thisSpacing
			continue
		}

		// Probability of forming an island at this position
		// Higher probability if we haven't had an island in a while (ensures some continuity)
		// Lower probability allows for occasional gaps
		minIslandSpacing := baseSpacing * Clamp(1.35-0.55*activity, 0.55, 1.60)
		formationProb := 0.28 + 0.45*activity + 0.07*float64(stepsSinceLastIsland)
		if distanceSinceLastIsland < 0.70*minIslandSpacing {
			formationProb *= 0.35
		}
		if quietStepsRemaining > 0 {
			formationProb -= 0.18
		} else if burstStepsRemaining > 0 {
			formationProb += 0.14
		}
		if formationProb > 0.98 {
			formationProb = 0.98
		}
		if formationProb < 0.10 {
			formationProb = 0.10
		}

		if rng.Float64() < formationProb && (distanceSinceLastIsland >= 0.45*minIslandSpacing || stepsSinceLastIsland >= 2) {
			visited[cellIdx] = true
			stepsSinceLastIsland = 0

			// Add island with age based on chain position
			age := chainLength / maxChainLength
			strength := 0.45 + 0.75*activity + (rng.Float64()-0.5)*0.25
			strength = Clamp(strength, 0.35, 1.55)
			chain.Islands = append(chain.Islands, HotspotIsland{
				CellIndex: cellIdx,
				Age:       age,
				Strength:  strength,
			})
			distanceSinceLastIsland = 0
		} else {
			stepsSinceLastIsland++
		}

		// Apply any motion-history changes reached at this point in the track.
		chainProgress := chainLength / maxChainLength
		for nextHistoryEvent < len(historyEvents) && chainProgress >= historyEvents[nextHistoryEvent].Progress {
			event := historyEvents[nextHistoryEvent]
			backwardRotation.Pole = rotatePoleAroundPoint(backwardRotation.Pole, currentPos, event.BendAngle)
			backwardRotation.AngularVelocity *= event.VelocityScale
			baselineActivity = Clamp(baselineActivity+event.ActivityShift, 0.18, 0.98)
			activity = Clamp(activity+event.ActivityShift*0.75, 0.12, 1.35)
			spacingFactor = Clamp(spacingFactor*(1.0-0.18*event.ActivityShift), 0.60, 2.10)
			lateralVelocity *= 0.45
			lateralOffset *= 0.55
			nextHistoryEvent++
		}

		// Rotate backward around the Euler pole
		currentPos = backwardRotation.RotatePoint(currentPos, thisSpacing)
		chainLength += thisSpacing

		// Apply correlated orthogonal wander - sideways displacement perpendicular
		// to travel, with inertia so the chain bends gradually instead of hopping.
		perpX := currentPos.Y*backwardRotation.Pole.Z - currentPos.Z*backwardRotation.Pole.Y
		perpY := currentPos.Z*backwardRotation.Pole.X - currentPos.X*backwardRotation.Pole.Z
		perpZ := currentPos.X*backwardRotation.Pole.Y - currentPos.Y*backwardRotation.Pole.X
		perpMag := math.Sqrt(perpX*perpX + perpY*perpY + perpZ*perpZ)

		if perpMag > 0.01 {
			lateralVelocity += (rng.Float64() - 0.5) * jitterScale * (0.10 + 0.08*activity)
			lateralVelocity = Clamp(lateralVelocity, -jitterScale*(0.45+0.20*activity), jitterScale*(0.45+0.20*activity))
			lateralOffset += lateralVelocity
			lateralOffset = Clamp(lateralOffset, -jitterScale*(0.9+0.35*activity), jitterScale*(0.9+0.35*activity))

			currentPos.X += (perpX / perpMag) * lateralOffset
			currentPos.Y += (perpY / perpMag) * lateralOffset
			currentPos.Z += (perpZ / perpMag) * lateralOffset
		}

		// Normalize to stay on unit sphere
		mag := math.Sqrt(currentPos.X*currentPos.X + currentPos.Y*currentPos.Y + currentPos.Z*currentPos.Z)
		if mag > 0 {
			currentPos.X /= mag
			currentPos.Y /= mag
			currentPos.Z /= mag
		}
		distanceSinceLastIsland += thisSpacing
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

type hotspotHistoryEvent struct {
	Progress      float64
	BendAngle     float64
	VelocityScale float64
	ActivityShift float64
}

func sampleHotspotHistoryEvents(hotspotLifetime float64, isOceanic bool, rng *rand.Rand) []hotspotHistoryEvent {
	events := make([]hotspotHistoryEvent, 0, 2)

	firstChance := 0.0
	secondChance := 0.0
	switch {
	case hotspotLifetime > 1.8:
		firstChance, secondChance = 0.65, 0.35
	case hotspotLifetime > 1.2:
		firstChance, secondChance = 0.38, 0.15
	case hotspotLifetime > 0.8:
		firstChance = 0.16
	}

	if rng.Float64() < firstChance {
		events = append(events, randomHistoryEvent(0.22, 0.58, isOceanic, rng))
	}
	if rng.Float64() < secondChance {
		events = append(events, randomHistoryEvent(0.52, 0.84, isOceanic, rng))
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Progress < events[j].Progress })
	if len(events) == 2 && events[1].Progress-events[0].Progress < 0.12 {
		events[1].Progress = Clamp(events[0].Progress+0.12, 0.58, 0.88)
	}
	return events
}

func randomHistoryEvent(minProgress, maxProgress float64, isOceanic bool, rng *rand.Rand) hotspotHistoryEvent {
	bendDegrees := 12.0 + 26.0*rng.Float64()
	if isOceanic {
		bendDegrees += 8.0 * rng.Float64()
	}
	if rng.Float64() < 0.5 {
		bendDegrees = -bendDegrees
	}

	return hotspotHistoryEvent{
		Progress:      minProgress + (maxProgress-minProgress)*rng.Float64(),
		BendAngle:     bendDegrees * math.Pi / 180.0,
		VelocityScale: 0.75 + 0.55*rng.Float64(),
		ActivityShift: (rng.Float64() - 0.5) * 0.45,
	}
}

func rotatePoleAroundPoint(pole, axis Vector3D, angle float64) Vector3D {
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	dot := pole.X*axis.X + pole.Y*axis.Y + pole.Z*axis.Z
	crossX := axis.Y*pole.Z - axis.Z*pole.Y
	crossY := axis.Z*pole.X - axis.X*pole.Z
	crossZ := axis.X*pole.Y - axis.Y*pole.X

	rotated := Vector3D{
		X: pole.X*cosA + crossX*sinA + axis.X*dot*(1-cosA),
		Y: pole.Y*cosA + crossY*sinA + axis.Y*dot*(1-cosA),
		Z: pole.Z*cosA + crossZ*sinA + axis.Z*dot*(1-cosA),
	}
	if rotated.Length() < 1e-9 {
		return pole
	}
	return rotated.Normalize()
}
