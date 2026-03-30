package climgen

// BuildTemperatureTransportWindField selects the wind field used by the
// temperature solver: marine wind over ocean, terrain-aware surface wind over
// land. This keeps basin-scale transport coherent over water without removing
// local land effects.
func BuildTemperatureTransportWindField(
	wind *WindResult,
	elevation []float64,
	seaLevelThreshold float64,
) []Vector3D {
	if wind == nil {
		return nil
	}

	if len(wind.SurfaceWind) == 0 && len(wind.MarineWind) == 0 {
		return nil
	}

	result := make([]Vector3D, len(elevation))
	for i := range elevation {
		switch {
		case elevation[i] < seaLevelThreshold && i < len(wind.MarineWind):
			result[i] = wind.MarineWind[i]
		case i < len(wind.SurfaceWind):
			result[i] = wind.SurfaceWind[i]
		case i < len(wind.MarineWind):
			result[i] = wind.MarineWind[i]
		}
	}
	return result
}

// ApplyResolvedMaritimeInfluence applies the stronger wind-aware maritime
// model when marine wind is available, otherwise it falls back to the older
// coastal moderation shortcut.
func ApplyResolvedMaritimeInfluence(
	temperature []float64,
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind *WindResult,
	currents []Vector3D,
	transport TransportSettings,
) []float64 {
	if wind != nil && len(wind.MarineWind) == len(temperature) {
		settings := DefaultMaritimeSettings()
		maritime := ComputeMaritimeInfluence(
			vertices,
			elevation,
			seaLevelThreshold,
			adj,
			wind.MarineWind,
			temperature,
			settings,
		)
		adjusted := ApplyMaritimeEffect(temperature, elevation, seaLevelThreshold, maritime, settings)

		if len(currents) == len(temperature) && transport.CurrentBacktrackDistance > 0 {
			sourceTemps := ComputeCurrentSourceTemperatures(
				vertices,
				elevation,
				seaLevelThreshold,
				adj,
				currents,
				transport.CurrentBacktrackDistance,
			)
			sourceAnomaly := make([]float64, len(temperature))
			for i := range sourceAnomaly {
				if elevation[i] >= seaLevelThreshold {
					continue
				}
				sourceAnomaly[i] = sourceTemps[i] - LatitudeEquilibriumTemp(vertices[i].Y)
			}
			anomalyMaritime := ComputeMaritimeInfluence(
				vertices,
				elevation,
				seaLevelThreshold,
				adj,
				wind.MarineWind,
				sourceAnomaly,
				settings,
			)
			adjusted = ApplyMaritimeAnomalyEffect(
				adjusted, elevation, seaLevelThreshold, anomalyMaritime, settings,
			)
		}

		return adjusted
	}

	return ApplyMarineInfluence(
		temperature, vertices, elevation, seaLevelThreshold, adj, 0.5, 350.0,
	)
}

// ApplyMaritimeAnomalyEffect carries current-driven SST anomalies inland on top
// of the base maritime moderation. This transfers warm/cold current signals
// without forcing land temperatures directly toward absolute ocean temperature.
func ApplyMaritimeAnomalyEffect(
	temperature []float64,
	elevation []float64,
	seaLevel float64,
	maritime *MaritimeResult,
	settings MaritimeSettings,
) []float64 {
	result := make([]float64, len(temperature))
	copy(result, temperature)

	if maritime == nil {
		return result
	}

	for i := range result {
		if elevation[i] < seaLevel {
			continue
		}
		influence := Clamp(maritime.Influence[i], 0, 1)
		if influence < 0.001 {
			continue
		}
		result[i] += settings.AnomalyBlendStrength * influence * maritime.SourceTemp[i]
	}

	return result
}
