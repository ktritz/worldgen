package main

import (
	"fmt"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

func printHydrologySupportDiagnostics(raw *terrain.HydrologyScaffold, adjusted *climgen.HydrologyBiomeInputs, elevation []float64, seaLevel float64) {
	if raw == nil && adjusted == nil {
		return
	}
	fmt.Println("    hydrologySupport:")
	if raw != nil {
		printFieldDistribution("hydroRawRunoff", landFloatValues(elevation, seaLevel, raw.Runoff))
		printFieldDistribution("hydroRawChannel", landFloatValues(elevation, seaLevel, raw.ChannelStrength))
		printFieldDistribution("hydroRawAccumulation", landFloatValues(elevation, seaLevel, raw.Accumulation))
	}
	if adjusted != nil {
		printFieldDistribution("hydroAdjustedRunoff", landFloatValues(elevation, seaLevel, adjusted.Runoff))
		printFieldDistribution("hydroAdjustedChannel", landFloatValues(elevation, seaLevel, adjusted.ChannelStrength))
		printFieldDistribution("hydroWetlandClassSupport", landFloatValues(elevation, seaLevel, adjusted.WetlandClassSupport))
		printFieldDistribution("hydroLakeClassSupport", landFloatValues(elevation, seaLevel, adjusted.LakeClassSupport))
		printFieldDistribution("hydroDepositionalClassSupport", landFloatValues(elevation, seaLevel, adjusted.DepositionalClassSupport))
		printFieldDistribution("hydroRiparianChannelSupport", landFloatValues(elevation, seaLevel, adjusted.RiparianChannelSupport))
	}
}

func landFloatValues(elevation []float64, seaLevel float64, values []float64) []float64 {
	out := make([]float64, 0, len(values))
	for i, value := range values {
		if i >= len(elevation) || elevation[i] < seaLevel {
			continue
		}
		out = append(out, value)
	}
	return out
}
