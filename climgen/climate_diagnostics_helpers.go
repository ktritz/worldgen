package climgen

import (
	"math"
	"sort"
)

func windFlags(d WindDiagnostics) []string {
	var flags []string
	if d.TangencyMaxError > 1e-3 {
		flags = append(flags, "wind tangent error high")
	}
	if d.TradeWestFraction > 0 && d.TradeWestFraction < 0.75 {
		flags = append(flags, "trade-wind zonal agreement weak")
	}
	if d.WesterlyEastFraction > 0 && d.WesterlyEastFraction < 0.70 {
		flags = append(flags, "westerly zonal agreement weak")
	}
	if d.HadleyConvergenceFraction > 0 && d.HadleyConvergenceFraction < 0.65 {
		flags = append(flags, "Hadley convergence weak")
	}
	return flags
}

func currentFlags(d CurrentDiagnostics) []string {
	var flags []string
	if d.CoastNormalViolationPt > 0.01 {
		flags = append(flags, "coastal current normal-flow violations")
	}
	if d.SpeedAnomalyFraction > 0.03 {
		flags = append(flags, "current speed spikes frequent")
	}
	if d.VorticityRatio > 12 {
		flags = append(flags, "current field overly rotational")
	}
	if d.GatewayFraction > 0.03 && d.GatewayAlignment < 0.55 {
		flags = append(flags, "gateway throughflow coherence weak")
	}
	return flags
}

func oceanClimateFlags(d OceanClimateDiagnostics) []string {
	var flags []string
	if d.SourceAnomalyP90AbsC > 0 && d.SourceAnomalyP90AbsC < 1.25 {
		flags = append(flags, "current source-temperature anomalies weak")
	}
	if d.WarmWesternBoundarySignalC > 0 && d.WarmWesternBoundarySignalC < 0.75 {
		flags = append(flags, "warm western-boundary current signal weak")
	}
	if d.ColdEasternBoundarySignalC > 0 && d.ColdEasternBoundarySignalC < 0.75 {
		flags = append(flags, "cold eastern-boundary current signal weak")
	}
	if d.CoastalLandCouplingCorr != 0 && d.CoastalLandCouplingCorr < 0.08 {
		flags = append(flags, "coastal land temperatures weakly coupled to offshore current anomalies")
	}
	return flags
}

func temperatureFlags(d TemperatureDiagnostics) []string {
	var flags []string
	if !d.Converged {
		flags = append(flags, "temperature solve did not converge")
	}
	if d.EquatorPoleGradientC < 20 {
		flags = append(flags, "equator-pole temperature gradient weak")
	}
	if d.AbsLatitudeTempCorr > -0.45 {
		flags = append(flags, "temperature latitude correlation weak")
	}
	if d.LocalAnomalyFraction > 0.03 {
		flags = append(flags, "temperature field has local spikes")
	}
	return flags
}

func precipitationFlags(d PrecipitationDiagnostics) []string {
	var flags []string
	if d.CoastalWetnessRatio > 0 && d.CoastalWetnessRatio < 1.05 {
		flags = append(flags, "coastal precipitation not stronger than inland")
	}
	if d.OnshoreOffshoreRatio > 0 && d.OnshoreOffshoreRatio < 1.20 {
		flags = append(flags, "onshore coasts not wetter than offshore coasts")
	}
	if d.TropicalToSubtropicRain > 0 && d.TropicalToSubtropicRain < 1.10 {
		flags = append(flags, "tropical rain belt too weak")
	}
	if d.OrographicContrast > 0 && d.OrographicContrast < 1.05 {
		flags = append(flags, "orographic rain-shadow contrast weak")
	}
	if d.LocalAnomalyFraction > 0.05 {
		flags = append(flags, "precipitation field has local spikes")
	}
	if d.ColdInteriorMean > 0 && d.ColdCoastalMean > 0 && d.ColdInteriorMean > 0.70*d.ColdCoastalMean {
		flags = append(flags, "cold interior precipitation too close to cold coastal belt")
	}
	if d.ColdInteriorMean > 40 {
		flags = append(flags, "cold interior land unusually wet")
	}
	return flags
}

func isCoastalLand(i int, elevation []float64, seaLevel float64, adj *FlatAdjacency) bool {
	if elevation[i] < seaLevel {
		return false
	}
	for _, k := range adj.GetNeighbors(i) {
		if k >= 0 && k < len(elevation) && elevation[k] < seaLevel {
			return true
		}
	}
	return false
}

func neighborMean(i int, field []float64, adj *FlatAdjacency) (float64, bool) {
	sum := 0.0
	count := 0
	for _, k := range adj.GetNeighbors(i) {
		if k >= 0 && k < len(field) {
			sum += field[k]
			count++
		}
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}

func neighborMeanSpeed(i int, field []Vector3D, elevation []float64, seaLevel float64, adj *FlatAdjacency) (float64, bool) {
	sum := 0.0
	count := 0
	for _, k := range adj.GetNeighbors(i) {
		if k >= 0 && k < len(field) && elevation[k] < seaLevel {
			sum += Length(field[k])
			count++
		}
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}

func safeFrac(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func percentile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	if q <= 0 {
		return cp[0]
	}
	if q >= 1 {
		return cp[len(cp)-1]
	}
	idx := q * float64(len(cp)-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return cp[lo]
	}
	t := idx - float64(lo)
	return cp[lo]*(1-t) + cp[hi]*t
}

func corr(xs, ys []float64) float64 {
	if len(xs) == 0 || len(xs) != len(ys) {
		return 0
	}
	mx, my := mean(xs), mean(ys)
	var num, dx2, dy2 float64
	for i := range xs {
		dx := xs[i] - mx
		dy := ys[i] - my
		num += dx * dy
		dx2 += dx * dx
		dy2 += dy * dy
	}
	if dx2 < 1e-12 || dy2 < 1e-12 {
		return 0
	}
	return num / math.Sqrt(dx2*dy2)
}
