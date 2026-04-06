package main

import (
	"fmt"

	"worldgen/climgen"
)

func printLandRouteSummary(result *climgen.LandRouteResult) {
	if result == nil || result.Diagnostics == nil || len(result.Diagnostics.ModeCost) == 0 {
		return
	}
	count := 0.0
	totalCost := 0.0
	totalRisk := 0.0
	totalSupport := 0.0
	totalWaystation := 0.0
	totalRoad := 0.0
	totalCrossing := 0.0
	totalBridge := 0.0
	totalFord := 0.0
	for i, cost := range result.Diagnostics.ModeCost {
		if cost <= 0 || cost > 1e8 {
			continue
		}
		count++
		totalCost += cost
		totalRisk += result.Diagnostics.RouteRisk[i]
		totalSupport += 0.5 * (result.Diagnostics.WaterSupport[i] + result.Diagnostics.ForageSupport[i])
		totalWaystation += result.Diagnostics.WaystationSuitability[i]
		totalRoad += result.Diagnostics.RoadQuality[i]
		totalCrossing += result.Diagnostics.CrossingPressure[i]
		totalBridge += result.Diagnostics.BridgeProxy[i]
		totalFord += result.Diagnostics.FordProxy[i]
	}
	if count == 0 {
		return
	}
	fmt.Printf(
		"    landRoutes[%s]: meanCost=%.2f meanRisk=%.2f meanSupport=%.2f waystation=%.2f road=%.2f crossing=%.2f bridge=%.2f ford=%.2f\n",
		result.Mode.Name,
		totalCost/count,
		totalRisk/count,
		totalSupport/count,
		totalWaystation/count,
		totalRoad/count,
		totalCrossing/count,
		totalBridge/count,
		totalFord/count,
	)
}
