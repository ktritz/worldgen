package climgen

import "math"

// Supported mesh resolution envelope: L5 (10242 cells, baseline) through L8 (655362).
// The upper clamp of 1.0 means coarser meshes (L4 and below) receive no correction and
// are intentionally unsupported. The lower clamp of 0.125 is exactly L8; finer meshes
// would saturate and are out of envelope.
const baselinePathCostCells = 10242.0

func meshPathCostResolutionScale(cellCount int) float64 {
	if cellCount <= 0 {
		return 1
	}
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

// meshResolutionAdjustedDiffusionIterations scales an iteration count for
// neighbor-averaging (diffusive) smoothers, which spread sigma ~ cellSize*sqrt(iters):
// holding a fixed physical smoothing radius requires iterations proportional to cell
// count. Directional one-cell-per-pass propagation should use
// meshResolutionAdjustedSteps instead.
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

func meshScaledTerritoryLinearCells(territoryCells int, meshCellCount int) float64 {
	if territoryCells <= 0 {
		territoryCells = 1
	}
	return math.Sqrt(float64(territoryCells)) * meshPathCostResolutionScale(meshCellCount)
}

func meshScaledTerritoryAreaCells(territoryCells int, meshCellCount int) float64 {
	if territoryCells <= 0 {
		territoryCells = 1
	}
	scale := meshPathCostResolutionScale(meshCellCount)
	return float64(territoryCells) * scale * scale
}

// Exported wrappers for external tooling (cmd/review_planets summaries) so the
// scale definition lives in exactly one place.

func MeshPathCostResolutionScale(cellCount int) float64 {
	return meshPathCostResolutionScale(cellCount)
}

func MeshAreaResolutionScale(cellCount int) float64 {
	return meshAreaResolutionScale(cellCount)
}

func MeshResolutionAdjustedSteps(baseSteps int, cellCount int) int {
	return meshResolutionAdjustedSteps(baseSteps, cellCount)
}

func MeshScaledTerritoryAreaCells(territoryCells int, meshCellCount int) float64 {
	return meshScaledTerritoryAreaCells(territoryCells, meshCellCount)
}
