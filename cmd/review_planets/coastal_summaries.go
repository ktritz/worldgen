package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func printCoastalResourceSummary(result *climgen.CoastalResourceResult) {
	if result == nil {
		return
	}
	counts := make(map[climgen.CoastalResourceType]int)
	landCells := 0
	coastalAccess := 0
	upwellingHotspots := 0
	for i, typ := range result.Types {
		if typ == climgen.CoastalResourceOcean {
			continue
		}
		landCells++
		counts[typ]++
		if result.Diagnostics != nil && i < len(result.Diagnostics.CoastalAccess) && result.Diagnostics.CoastalAccess[i] >= 0.18 {
			coastalAccess++
		}
		if result.Diagnostics != nil && i < len(result.Diagnostics.UpwellingPotential) && result.Diagnostics.UpwellingPotential[i] >= 0.30 {
			upwellingHotspots++
		}
	}
	if landCells == 0 {
		return
	}
	fishery := counts[climgen.CoastalResourceOpenFishery] + counts[climgen.CoastalResourceEstuarineFishery]
	shellfish := counts[climgen.CoastalResourceShellfish]
	saltworks := counts[climgen.CoastalResourceSaltworks]
	fmt.Printf("    coastalResourceMetrics: access=%.1f%% fishery=%.1f%% shellfish=%.1f%% saltworks=%.1f%% upwelling=%.1f%%\n",
		100*float64(coastalAccess)/float64(landCells),
		100*float64(fishery)/float64(landCells),
		100*float64(shellfish)/float64(landCells),
		100*float64(saltworks)/float64(landCells),
		100*float64(upwellingHotspots)/float64(landCells),
	)

	type coastalCount struct {
		typ   climgen.CoastalResourceType
		count int
	}
	var sorted []coastalCount
	for typ, count := range counts {
		sorted = append(sorted, coastalCount{typ: typ, count: count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	fmt.Println("    coastal resources summary:")
	for i := 0; i < limit; i++ {
		entry := sorted[i]
		fmt.Printf("      coastal[%s]=%d (%.1f%%)\n",
			climgen.CoastalResourceName(entry.typ),
			entry.count,
			100*float64(entry.count)/float64(landCells),
		)
	}
}
