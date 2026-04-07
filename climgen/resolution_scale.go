package climgen

import "math"

const baselinePathCostCells = 10242.0

func meshPathCostResolutionScale(cellCount int) float64 {
	if cellCount <= 0 {
		return 1
	}
	scale := math.Sqrt(baselinePathCostCells / float64(cellCount))
	if scale < 0.25 {
		return 0.25
	}
	if scale > 1.0 {
		return 1.0
	}
	return scale
}
