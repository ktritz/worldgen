package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func printPopulationSummary(result *climgen.PopulationResult) {
	if result == nil || result.Diagnostics == nil {
		return
	}
	counts := make(map[climgen.PopulationClass]int)
	landCells := 0
	totalCarry := 0.0
	urbanHotspots := 0
	for i, class := range result.Classes {
		if class == climgen.PopulationOcean {
			continue
		}
		landCells++
		counts[class]++
		if i < len(result.Diagnostics.CarryingCapacity) {
			totalCarry += result.Diagnostics.CarryingCapacity[i]
		}
		if i < len(result.Diagnostics.UrbanPotential) && result.Diagnostics.UrbanPotential[i] >= 0.58 {
			urbanHotspots++
		}
	}
	if landCells == 0 {
		return
	}
	frontier := counts[climgen.PopulationSparseFrontier]
	settled := counts[climgen.PopulationRural] +
		counts[climgen.PopulationDenseRural] +
		counts[climgen.PopulationUrban]
	dense := counts[climgen.PopulationDenseRural] + counts[climgen.PopulationUrban]
	urban := counts[climgen.PopulationUrban]
	fmt.Printf("    populationMetrics: frontierSupport=%.1f%% settledSupport=%.1f%% denseSupport=%.1f%% urbanSupport=%.1f%% meanCarry=%.2f hotspots=%.1f%%\n",
		100*float64(frontier)/float64(landCells),
		100*float64(settled)/float64(landCells),
		100*float64(dense)/float64(landCells),
		100*float64(urban)/float64(landCells),
		totalCarry/float64(landCells),
		100*float64(urbanHotspots)/float64(landCells),
	)

	type populationCount struct {
		class climgen.PopulationClass
		count int
	}
	var sorted []populationCount
	for class, count := range counts {
		sorted = append(sorted, populationCount{class: class, count: count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	fmt.Println("    population summary:")
	for i := 0; i < limit; i++ {
		entry := sorted[i]
		fmt.Printf("      population[%s]=%d (%.1f%%)\n",
			climgen.PopulationClassName(entry.class),
			entry.count,
			100*float64(entry.count)/float64(landCells),
		)
	}
}
