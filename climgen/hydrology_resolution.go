package climgen

// ResolutionAdjustedHydrologyBiomeInputs converts routed hydrology centerlines
// and one-cell classes into physical support fields before downstream layers
// consume them. Higher-resolution meshes represent the same river/floodplain
// width with more graph steps, so the support radius scales with cell size.
func ResolutionAdjustedHydrologyBiomeInputs(
	cells []VoronoiCell,
	elevation []float64,
	seaLevel float64,
	hydro *HydrologyBiomeInputs,
) *HydrologyBiomeInputs {
	if hydro == nil {
		return nil
	}
	out := &HydrologyBiomeInputs{
		Runoff:          append([]float64(nil), hydro.Runoff...),
		ChannelStrength: append([]float64(nil), hydro.ChannelStrength...),
		CellClass:       append([]string(nil), hydro.CellClass...),
		WaterBodyLabel:  append([]int(nil), hydro.WaterBodyLabel...),
	}
	n := len(out.ChannelStrength)
	if n == 0 {
		n = len(out.CellClass)
	}
	if len(cells) == 0 || len(elevation) == 0 || n == 0 {
		return out
	}
	radius := meshResolutionAdjustedSteps(1, len(cells))
	if radius <= 0 {
		return out
	}

	wetlandClass := make([]float64, len(out.CellClass))
	lakeClass := make([]float64, len(out.CellClass))
	depositionalClass := make([]float64, len(out.CellClass))
	for i := range out.CellClass {
		wetlandClass[i] = directHydrologyClassFactor(out.CellClass[i])
		lakeClass[i] = directLakeClassFactor(out.CellClass[i])
		depositionalClass[i] = directDepositionalClassFactor(out.CellClass[i])
	}
	out.WetlandClassSupport = spreadPhysicalMaxSignal(cells, elevation, seaLevel, wetlandClass, radius)
	out.LakeClassSupport = spreadPhysicalMaxSignal(cells, elevation, seaLevel, lakeClass, radius)
	out.DepositionalClassSupport = spreadPhysicalMaxSignal(cells, elevation, seaLevel, depositionalClass, radius)
	if len(hydro.ChannelStrength) > 0 {
		riparianChannel := make([]float64, len(hydro.ChannelStrength))
		for i, channel := range hydro.ChannelStrength {
			riparianChannel[i] = smoothstep01(0.7, 2.2, channel)
		}
		// Channel initiation deliberately selects a *linear* feature: the
		// threshold is channelInitiationPercentile(n) = 100 - 6.5*pathScale, so
		// channel cells are 6.5% of land at L5, 3.25% at L6, 1.6% at L7. That
		// halving is correct for a river network, whose cell count grows with
		// the square root of the mesh, and consumers that follow the watercourse
		// want exactly it.
		//
		// Consumers that cover *area* do not: reading the bare centerline made
		// the riparian land fraction halve every level (4.3% / 2.2% / 1.1%
		// measured), which propagated through settlement water and access
		// scores into carrying capacity, halving the above-town fraction per
		// level and starving anchors until proto-civilizations failed to
		// nucleate at all on some L7 seeds.
		//
		// Widening to a constant physical corridor fixes the area without
		// touching the centerline. The spread is a max, not a sum, so the
		// overlapping footprints of adjacent channel cells along a watercourse
		// collapse to the corridor rather than accumulating — the earlier
		// concern about over-expanding at high resolution applies to summed
		// spreads. The radius is one step short of the physical one-hop radius,
		// which is zero at L5 and so leaves the baseline byte-identical, while
		// the 1/(1+d*stepScale) falloff makes the effective corridor width
		// 1.00 / 1.17 / 1.27 baseline cells across L5-L7 instead of 1.00 / 0.50
		// / 0.25.
		corridorRadius := radius - 1
		out.RiparianChannelSupport = spreadPhysicalMaxSignal(cells, elevation, seaLevel, riparianChannel, corridorRadius)
		out.ChannelCorridorStrength = spreadPhysicalMaxSignal(cells, elevation, seaLevel, hydro.ChannelStrength, corridorRadius)
	}
	return out
}

func spreadPhysicalMaxSignal(cells []VoronoiCell, elevation []float64, seaLevel float64, source []float64, radius int) []float64 {
	out := append([]float64(nil), source...)
	if radius <= 0 || len(cells) == 0 || len(source) == 0 {
		return out
	}
	stepScale := meshPathCostResolutionScale(len(cells))
	seen := make([]int, len(cells))
	stamp := 0
	queueCells := make([]int, 0, 64)
	queueDists := make([]int, 0, 64)
	for start, value := range source {
		if value <= 0 || start < 0 || start >= len(cells) {
			continue
		}
		stamp++
		if stamp == 0 {
			for i := range seen {
				seen[i] = 0
			}
			stamp = 1
		}
		queueCells = queueCells[:0]
		queueDists = queueDists[:0]
		queueCells = append(queueCells, start)
		queueDists = append(queueDists, 0)
		seen[start] = stamp
		for head := 0; head < len(queueCells); head++ {
			cell := queueCells[head]
			dist := queueDists[head]
			if cell >= 0 && cell < len(elevation) && elevation[cell] >= seaLevel {
				physicalDist := float64(dist) * stepScale
				supported := value / (1.0 + physicalDist)
				if cell < len(out) && supported > out[cell] {
					out[cell] = supported
				}
			}
			if dist >= radius {
				continue
			}
			for _, neighborIdx := range cells[cell].NeighborSiteIndices {
				neighbor := int(neighborIdx)
				if neighbor < 0 || neighbor >= len(cells) || seen[neighbor] == stamp {
					continue
				}
				seen[neighbor] = stamp
				queueCells = append(queueCells, neighbor)
				queueDists = append(queueDists, dist+1)
			}
		}
	}
	return out
}

func spreadPhysicalSumSignal(cells []VoronoiCell, elevation []float64, seaLevel float64, source []float64, radius int) []float64 {
	out := make([]float64, len(source))
	if radius <= 0 || len(cells) == 0 || len(source) == 0 {
		return out
	}
	stepScale := meshPathCostResolutionScale(len(cells))
	seen := make([]int, len(cells))
	stamp := 0
	queueCells := make([]int, 0, 64)
	queueDists := make([]int, 0, 64)
	for start, value := range source {
		if value <= 0 || start < 0 || start >= len(cells) {
			continue
		}
		stamp++
		if stamp == 0 {
			for i := range seen {
				seen[i] = 0
			}
			stamp = 1
		}
		queueCells = queueCells[:0]
		queueDists = queueDists[:0]
		queueCells = append(queueCells, start)
		queueDists = append(queueDists, 0)
		seen[start] = stamp
		for head := 0; head < len(queueCells); head++ {
			cell := queueCells[head]
			dist := queueDists[head]
			if dist > 0 && cell >= 0 && cell < len(elevation) && elevation[cell] >= seaLevel {
				physicalDist := float64(dist) * stepScale
				if cell < len(out) {
					out[cell] += value / (1.0 + physicalDist)
				}
			}
			if dist >= radius {
				continue
			}
			for _, neighborIdx := range cells[cell].NeighborSiteIndices {
				neighbor := int(neighborIdx)
				if neighbor < 0 || neighbor >= len(cells) || seen[neighbor] == stamp {
					continue
				}
				seen[neighbor] = stamp
				queueCells = append(queueCells, neighbor)
				queueDists = append(queueDists, dist+1)
			}
		}
	}
	return out
}

func directHydrologyClassFactor(className string) float64 {
	switch className {
	case "floodplain":
		return 1.0
	case "delta":
		return 0.95
	case "lake_reach":
		return 0.85
	case "coast_outlet":
		return 0.65
	case "confluence":
		return 0.55
	case "trunk":
		return 0.35
	default:
		return 0
	}
}

func directLakeClassFactor(className string) float64 {
	switch className {
	case "lake":
		return 1.0
	case "lake_complex":
		return 0.90
	case "lake_reach":
		return 0.35
	case "endorheic_basin":
		return 0.30
	case "delta":
		return 0.18
	default:
		return 0
	}
}

func directDepositionalClassFactor(className string) float64 {
	switch className {
	case "lake", "lake_complex":
		return 1.0
	case "endorheic_basin":
		return 0.85
	case "lake_reach":
		return 0.70
	case "floodplain", "delta":
		return 0.65
	case "coast_outlet", "confluence":
		return 0.35
	default:
		return 0
	}
}
