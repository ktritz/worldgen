package climgen

import "math"

type maritimeAccessDiagnostic struct {
	EffectiveFetch   float64
	EffectiveOnshore float64
	MarineSignal     float64
	GeometricSupport float64
}

func computeNeighborOceanFraction(
	i int,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
) float64 {
	if i < 0 || i >= len(elevation) || elevation[i] < seaLevel {
		return 0
	}
	neighbors := adj.GetNeighbors(i)
	if len(neighbors) == 0 {
		return 0
	}
	ocean := 0.0
	total := 0.0
	for _, k := range neighbors {
		if k < 0 || k >= len(elevation) {
			continue
		}
		total++
		if elevation[k] < seaLevel {
			ocean++
		}
	}
	if total <= 0 {
		return 0
	}
	return Clamp(ocean/total, 0, 1)
}

func computeUpwindOceanFootprintSupport(
	i int,
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	maxDepth int,
) float64 {
	weights := computeUpwindFootprintWeights(
		i,
		vertices,
		adj,
		wind,
		maxDepth,
		precipUpwindFootprintMinAlignment,
	)
	if len(weights) == 0 {
		return 0
	}
	support := 0.0
	for donor, weight := range weights {
		if donor < 0 || donor >= len(elevation) {
			continue
		}
		if elevation[donor] < seaLevel {
			support += weight
		}
	}
	return Clamp(support, 0, 1)
}

func deriveEffectiveMaritimeAccessDiagnostic(
	rawFetch float64,
	rawOnshore float64,
	footprintSupport float64,
	neighborOceanFraction float64,
	landTravel float64,
	landInterior float64,
	marineIncoming float64,
) maritimeAccessDiagnostic {
	travel := Clamp(landTravel, 0, 1)
	interior := Clamp(landInterior, 0, 1)
	corridor := transportCorridorWeight(travel)
	marineSignal := smoothRamp(0.02, 0.16, marineIncoming)
	geomSupport := Clamp(
		0.50*Clamp(rawFetch, 0, 1)+
			0.35*Clamp(footprintSupport, 0, 1)+
			0.15*Clamp(neighborOceanFraction, 0, 1),
		0,
		1,
	)

	inlandMarineSupport := marineSignal *
		(0.10 + 0.45*corridor + 0.15*travel) *
		(0.10 + 0.60*geomSupport + 0.30*Clamp(neighborOceanFraction, 0, 1))
	marineFetchCap := Clamp(
		0.08+
			0.28*Clamp(footprintSupport, 0, 1)+
			0.18*Clamp(neighborOceanFraction, 0, 1)+
			0.18*corridor*(0.25+0.75*geomSupport)+
			0.06*(1.0-interior),
		0.08,
		0.75,
	)
	marineFetchBoost := math.Min(inlandMarineSupport, marineFetchCap)
	effectiveFetch := Clamp(maxFloat(rawFetch, maxFloat(footprintSupport, marineFetchBoost)), 0, 1)

	onshoreGeometry := Clamp(
		rawOnshore+
			0.30*Clamp(footprintSupport, 0, 1)*(1.0-0.55*travel)+
			0.18*Clamp(neighborOceanFraction, 0, 1)*(1.0-0.40*travel),
		0,
		1,
	)
	inlandExposure := Clamp(
		(0.05+0.18*corridor+0.08*travel)*
			(0.20+0.60*geomSupport+0.20*Clamp(neighborOceanFraction, 0, 1)),
		0,
		0.35,
	)
	marineOnshore := marineSignal * inlandExposure
	effectiveOnshore := Clamp(
		maxFloat(
			rawOnshore,
			math.Min(
				maxFloat(onshoreGeometry, marineOnshore),
				effectiveFetch*(0.20+0.60*geomSupport),
			),
		),
		0,
		1,
	)
	return maritimeAccessDiagnostic{
		EffectiveFetch:   effectiveFetch,
		EffectiveOnshore: effectiveOnshore,
		MarineSignal:     marineSignal,
		GeometricSupport: geomSupport,
	}
}

func computeEffectiveMaritimeAccess(
	vertices []Vector3D,
	elevation []float64,
	seaLevel float64,
	adj *FlatAdjacency,
	wind []Vector3D,
	marineIncoming []float64,
	rawFetch []float64,
	rawOnshore []float64,
	landTravel []float64,
	landInterior []float64,
) ([]float64, []float64) {
	effectiveFetch := append([]float64(nil), rawFetch...)
	effectiveOnshore := append([]float64(nil), rawOnshore...)
	for i := range effectiveFetch {
		if i >= len(elevation) || elevation[i] < seaLevel {
			continue
		}
		neighborOceanFraction := computeNeighborOceanFraction(i, elevation, seaLevel, adj)
		footprintSupport := computeUpwindOceanFootprintSupport(
			i,
			vertices,
			elevation,
			seaLevel,
			adj,
			wind,
			precipInlandTransportSteps+4,
		)
		marine := 0.0
		if i < len(marineIncoming) {
			marine = marineIncoming[i]
		}
		travel := 0.0
		if i < len(landTravel) {
			travel = landTravel[i]
		}
		interior := 0.0
		if i < len(landInterior) {
			interior = landInterior[i]
		}
		diag := deriveEffectiveMaritimeAccessDiagnostic(
			rawFetch[i],
			rawOnshore[i],
			footprintSupport,
			neighborOceanFraction,
			travel,
			interior,
			marine,
		)
		effectiveFetch[i] = diag.EffectiveFetch
		effectiveOnshore[i] = diag.EffectiveOnshore
	}
	return effectiveFetch, effectiveOnshore
}
