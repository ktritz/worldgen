package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func printWaterResourceSummary(result *climgen.WaterResourceResult) {
	if result == nil {
		return
	}
	counts := make(map[climgen.WaterResourceType]int)
	landCells := 0
	for _, typ := range result.Types {
		if typ == climgen.WaterResourceOcean {
			continue
		}
		landCells++
		counts[typ]++
	}
	if landCells == 0 {
		return
	}
	reliable := counts[climgen.WaterResourceReliableSurface]
	seasonal := counts[climgen.WaterResourceSeasonal]
	groundwater := counts[climgen.WaterResourceGroundwater]
	lake := counts[climgen.WaterResourceLakeOasis]
	fmt.Printf("    waterMetrics: reliable=%.1f%% seasonal=%.1f%% groundwater=%.1f%% lake=%.1f%%\n",
		100*float64(reliable)/float64(landCells),
		100*float64(seasonal)/float64(landCells),
		100*float64(groundwater)/float64(landCells),
		100*float64(lake)/float64(landCells),
	)

	type waterCount struct {
		typ   climgen.WaterResourceType
		count int
	}
	var sorted []waterCount
	for typ, count := range counts {
		sorted = append(sorted, waterCount{typ: typ, count: count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	fmt.Println("    water resources summary:")
	for i := 0; i < limit; i++ {
		entry := sorted[i]
		fmt.Printf("      water[%s]=%d (%.1f%%)\n",
			climgen.WaterResourceName(entry.typ),
			entry.count,
			100*float64(entry.count)/float64(landCells),
		)
	}
}
