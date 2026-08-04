package climgen

const maritimeEndpointLowerBoundSafetyFactor = 0.50

func maritimeMeanNeighborDegrees(sites []Vector3D, cells []VoronoiCell) float64 {
	if len(sites) != len(cells) {
		return 0
	}
	total := 0.0
	count := 0
	for i, cell := range cells {
		if i < 0 || i >= len(sites) {
			continue
		}
		for _, raw := range cell.NeighborSiteIndices {
			j := int(raw)
			if j <= i || j < 0 || j >= len(sites) {
				continue
			}
			total += greatCircleDistanceDeg(sites[i], sites[j])
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func maritimeEndpointPairWithinBudgetLowerBound(
	sites []Vector3D,
	cells []VoronoiCell,
	fromCell, toCell int,
	maxCost float64,
	minStepCost float64,
	meanNeighborDeg float64,
) bool {
	if maxCost <= 0 || minStepCost <= 0 || meanNeighborDeg <= 0 || len(sites) != len(cells) {
		return true
	}
	if fromCell < 0 || toCell < 0 || fromCell >= len(sites) || toCell >= len(sites) {
		return true
	}
	angularDeg := greatCircleDistanceDeg(sites[fromCell], sites[toCell])
	minStepCount := angularDeg / meanNeighborDeg
	lowerBound := maritimeEndpointLowerBoundSafetyFactor * minStepCount * minStepCost * meshPathCostResolutionScale(len(cells))
	return lowerBound <= maxCost
}
