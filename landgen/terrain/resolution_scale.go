package terrain

import "math"

const baselinePathCostCells = 10242.0

// baselineCellAngularRadius is the mean cell angular radius (radians) of the
// baseline L5 mesh (2/sqrt(10242) ~= 0.0198). Formulas that convert a physical
// angular size into a physical outcome (e.g. hotspot peak-height caps) must
// normalize by this constant rather than the actual mesh spacing, so identical
// physical features produce identical results at every resolution.
var baselineCellAngularRadius = 2.0 / math.Sqrt(baselinePathCostCells)

// meanCellAngularSpacing returns the mean center-to-center angular distance, in
// radians, between neighbouring cells of a geodesic mesh with cellCount cells.
// Cells tile the unit sphere hexagonally, so a cell of area 4*pi/n has center
// spacing d satisfying (sqrt(3)/2)*d^2 = 4*pi/n. Unlike
// meshPathCostResolutionScale this is unclamped: it is a statement about the
// mesh, not a tuning envelope, and must stay exact at every resolution so that
// hop counts converted through it are resolution-stable.
func meanCellAngularSpacing(cellCount int) float64 {
	if cellCount <= 0 {
		return 0
	}
	return math.Sqrt(8 * math.Pi / (math.Sqrt(3) * float64(cellCount)))
}

func meshPathCostResolutionScale(cellCount int) float64 {
	if cellCount <= 0 {
		return 1
	}
	// Envelope L5..L8: lower clamp 0.125 is exactly L8 (655362 cells); upper clamp
	// 1.0 leaves coarser-than-baseline meshes (L4 and below) intentionally uncorrected.
	scale := math.Sqrt(baselinePathCostCells / float64(cellCount))
	if scale < 0.125 {
		return 0.125
	}
	if scale > 1.0 {
		return 1.0
	}
	return scale
}

func meshAreaResolutionScale(cellCount int) float64 {
	scale := meshPathCostResolutionScale(cellCount)
	return scale * scale
}

func meshResolutionAdjustedSteps(baseSteps int, cellCount int) int {
	if baseSteps <= 0 {
		return 0
	}
	scale := meshPathCostResolutionScale(cellCount)
	if scale <= 0 {
		return baseSteps
	}
	steps := int(math.Ceil(float64(baseSteps) / scale))
	if steps < baseSteps {
		return baseSteps
	}
	return steps
}

func meshResolutionAdjustedDiffusionIterations(baseIterations int, cellCount int) int {
	if baseIterations <= 0 {
		return 0
	}
	areaScale := meshAreaResolutionScale(cellCount)
	if areaScale <= 0 {
		return baseIterations
	}
	iterations := int(math.Ceil(float64(baseIterations) / areaScale))
	if iterations < baseIterations {
		return baseIterations
	}
	return iterations
}
