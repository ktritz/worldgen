package climgen

import "math"

// ClimateDiagnostics groups optimization-oriented metrics and anomaly flags for
// wind, currents, temperature, and precipitation fields.
type ClimateDiagnostics struct {
	Wind          WindDiagnostics
	Currents      CurrentDiagnostics
	OceanClimate  OceanClimateDiagnostics
	Temperature   TemperatureDiagnostics
	Precipitation PrecipitationDiagnostics
	Flags         []string
}

type WindDiagnostics struct {
	MeanSpeed                 float64
	P95Speed                  float64
	MaxSpeed                  float64
	CalmFraction              float64
	TangencyMaxError          float64
	NeighborAlignment         float64
	TradeWestFraction         float64
	WesterlyEastFraction      float64
	PolarWestFraction         float64
	HadleyConvergenceFraction float64
	FerrelPolewardFraction    float64
	PolarEquatorwardFraction  float64
}

type CurrentDiagnostics struct {
	MeanSpeed              float64
	P95Speed               float64
	MaxSpeed               float64
	ActiveFraction         float64
	CoastNormalP95         float64
	CoastNormalViolationPt float64
	SpeedAnomalyFraction   float64
	BasinCount             int
	LargestBasinFraction   float64
	FlowCoherence          float64
	SpeedCoV               float64
	AvgVorticity           float64
	VorticityRatio         float64
	GatewayFraction        float64
	GatewayAlignment       float64
}

type OceanClimateDiagnostics struct {
	SourceAnomalyMeanAbsC      float64
	SourceAnomalyP90AbsC       float64
	WarmWesternBoundarySignalC float64
	ColdEasternBoundarySignalC float64
	CoastalLandCouplingCorr    float64
	WarmAdjacentLandResidualC  float64
	ColdAdjacentLandResidualC  float64
}

type TemperatureDiagnostics struct {
	MeanC                 float64
	LandMeanC             float64
	OceanMeanC            float64
	MinC                  float64
	MaxC                  float64
	EquatorMeanC          float64
	PolarMeanC            float64
	EquatorPoleGradientC  float64
	AbsLatitudeTempCorr   float64
	LandOceanContrastC    float64
	LocalResidualP95C     float64
	LocalAnomalyFraction  float64
	FreezingLandFraction  float64
	FreezingOceanFraction float64
	Converged             bool
}

type PrecipitationDiagnostics struct {
	LandMean                 float64
	LandP90                  float64
	Max                      float64
	RainFraction             float64
	SnowFraction             float64
	DryLandFraction          float64
	WetLandFraction          float64
	ExtremeWetLandFraction   float64
	CoastalWetnessRatio      float64
	TropicalToSubtropicRain  float64
	OrographicContrast       float64
	LocalAnomalyFraction     float64
	ColdCoastalMean          float64
	ColdInteriorMean         float64
	ColdAlpineMean           float64
	ColdCoastalSnowFraction  float64
	ColdInteriorSnowFraction float64
	OnshoreCoastalMean       float64
	OffshoreCoastalMean      float64
	OnshoreOffshoreRatio     float64
}

// ComputeClimateDiagnostics derives a structured diagnostics bundle from the
// currently generated climate fields.
func ComputeClimateDiagnostics(
	vertices []Vector3D,
	elevation []float64,
	seaLevelThreshold float64,
	adj *FlatAdjacency,
	wind *WindResult,
	currents *OceanCurrentResult,
	temperature *TemperatureResult,
	precipitation *PrecipitationResult,
) ClimateDiagnostics {
	var d ClimateDiagnostics
	if wind != nil {
		d.Wind = computeWindDiagnostics(vertices, elevation, seaLevelThreshold, adj, wind)
		d.Flags = append(d.Flags, windFlags(d.Wind)...)
	}
	if currents != nil {
		d.Currents = computeCurrentDiagnostics(vertices, elevation, seaLevelThreshold, adj, currents)
		d.Flags = append(d.Flags, currentFlags(d.Currents)...)
	}
	if currents != nil {
		d.OceanClimate = computeOceanClimateDiagnostics(vertices, elevation, seaLevelThreshold, adj, currents, temperature)
		d.Flags = append(d.Flags, oceanClimateFlags(d.OceanClimate)...)
	}
	if temperature != nil {
		d.Temperature = computeTemperatureDiagnostics(vertices, elevation, seaLevelThreshold, adj, temperature)
		d.Flags = append(d.Flags, temperatureFlags(d.Temperature)...)
	}
	if precipitation != nil {
		d.Precipitation = computePrecipitationDiagnostics(vertices, elevation, seaLevelThreshold, adj, wind, temperature, precipitation)
		d.Flags = append(d.Flags, precipitationFlags(d.Precipitation)...)
	}
	return d
}

func computeWindDiagnostics(vertices []Vector3D, elevation []float64, seaLevel float64, adj *FlatAdjacency, result *WindResult) WindDiagnostics {
	d := WindDiagnostics{}
	speeds := make([]float64, len(result.SurfaceWind))
	alignSum, alignCount := 0.0, 0
	tradeOK, tradeN := 0, 0
	westerlyOK, westerlyN := 0, 0
	polarOK, polarN := 0, 0
	hadleyOK, hadleyN := 0, 0
	ferrelOK, ferrelN := 0, 0
	polarMeridOK, polarMeridN := 0, 0

	for i, wind := range result.SurfaceWind {
		speed := Length(wind)
		speeds[i] = speed
		if speed < 0.02 {
			d.CalmFraction++
		}
		tangentErr := math.Abs(Dot(vertices[i], wind))
		if tangentErr > d.TangencyMaxError {
			d.TangencyMaxError = tangentErr
		}

		east, north := GetTangentVectors(vertices[i])
		zonal := Dot(wind, east)
		merid := Dot(wind, north)
		lat := getLatitude(vertices[i])
		northHem := lat >= 0

		switch result.CirculationZone[i] {
		case ZoneHadley:
			tradeN++
			if zonal < 0 {
				tradeOK++
			}
			hadleyN++
			if (northHem && merid < 0) || (!northHem && merid > 0) {
				hadleyOK++
			}
		case ZoneFerrel:
			westerlyN++
			if zonal > 0 {
				westerlyOK++
			}
			ferrelN++
			if (northHem && merid > 0) || (!northHem && merid < 0) {
				ferrelOK++
			}
		case ZonePolar:
			polarN++
			if zonal < 0 {
				polarOK++
			}
			polarMeridN++
			if (northHem && merid < 0) || (!northHem && merid > 0) {
				polarMeridOK++
			}
		}

		if speed < 1e-9 {
			continue
		}
		ci := Scale(wind, 1.0/speed)
		for _, k := range adj.GetNeighbors(i) {
			if k <= i {
				continue
			}
			neighborSpeed := Length(result.SurfaceWind[k])
			if neighborSpeed < 1e-9 {
				continue
			}
			cj := Scale(result.SurfaceWind[k], 1.0/neighborSpeed)
			alignSum += Dot(ci, cj)
			alignCount++
		}
	}

	d.MeanSpeed = mean(speeds)
	d.P95Speed = percentile(speeds, 0.95)
	d.MaxSpeed = percentile(speeds, 1.0)
	d.CalmFraction /= float64(len(result.SurfaceWind))
	if alignCount > 0 {
		d.NeighborAlignment = alignSum / float64(alignCount)
	}
	d.TradeWestFraction = safeFrac(tradeOK, tradeN)
	d.WesterlyEastFraction = safeFrac(westerlyOK, westerlyN)
	d.PolarWestFraction = safeFrac(polarOK, polarN)
	d.HadleyConvergenceFraction = safeFrac(hadleyOK, hadleyN)
	d.FerrelPolewardFraction = safeFrac(ferrelOK, ferrelN)
	d.PolarEquatorwardFraction = safeFrac(polarMeridOK, polarMeridN)
	return d
}

func computeCurrentDiagnostics(vertices []Vector3D, elevation []float64, seaLevel float64, adj *FlatAdjacency, result *OceanCurrentResult) CurrentDiagnostics {
	d := CurrentDiagnostics{BasinCount: len(result.Basins)}
	speeds := make([]float64, 0, len(result.Currents))
	landDirs := CalculateCoastlineLandDirs(vertices, elevation, seaLevel, adj)
	coastNormals := make([]float64, 0, len(result.Currents))
	anomalyCount := 0
	waterCount := 0
	assignments, components := FindOceanComponents(elevation, seaLevel, adj)
	openness := BuildOceanOpennessField(vertices, elevation, seaLevel, adj, assignments, components)
	gatewayStrength, gatewayAxis := BuildGatewayField(vertices, elevation, seaLevel, adj, assignments, openness)
	gatewayAlignSum := 0.0
	gatewayN := 0

	for i, current := range result.Currents {
		if elevation[i] >= seaLevel {
			continue
		}
		waterCount++
		speed := Length(current)
		speeds = append(speeds, speed)
		if speed > 1e-6 {
			d.ActiveFraction++
		}
		if ld := Length(landDirs[i]); ld > 1e-6 && speed > 1e-6 {
			norm := math.Abs(Dot(current, landDirs[i]) / (speed * ld))
			coastNormals = append(coastNormals, norm)
			if norm > 0.12 {
				d.CoastNormalViolationPt++
			}
		}

		localMean, ok := neighborMeanSpeed(i, result.Currents, elevation, seaLevel, adj)
		if ok && speed > math.Max(0.08, 3.5*localMean) {
			anomalyCount++
		}
		if i < len(gatewayStrength) && gatewayStrength[i] > gatewayStrengthThreshold {
			d.GatewayFraction++
			if speed > 1e-6 && i < len(gatewayAxis) && Length(gatewayAxis[i]) > 1e-6 {
				align := math.Abs(Dot(current, gatewayAxis[i]) / speed)
				gatewayAlignSum += align
				gatewayN++
			}
		}
	}
	if waterCount > 0 {
		d.ActiveFraction /= float64(waterCount)
		d.CoastNormalViolationPt /= float64(waterCount)
		d.SpeedAnomalyFraction = float64(anomalyCount) / float64(waterCount)
		d.GatewayFraction /= float64(waterCount)
	}
	if gatewayN > 0 {
		d.GatewayAlignment = gatewayAlignSum / float64(gatewayN)
	}
	d.MeanSpeed = mean(speeds)
	d.P95Speed = percentile(speeds, 0.95)
	d.MaxSpeed = percentile(speeds, 1.0)
	d.CoastNormalP95 = percentile(coastNormals, 0.95)

	metrics := ComputeCoalescenceMetrics(result.Currents, vertices, elevation, seaLevel, adj, result.Basins)
	d.FlowCoherence = metrics.FlowCoherence
	d.SpeedCoV = metrics.SpeedCoV
	d.AvgVorticity = metrics.AvgVorticity
	d.VorticityRatio = metrics.VorticityRatio

	maxBasin := 0
	totalBasin := 0
	for _, basin := range result.Basins {
		totalBasin += len(basin.Vertices)
		if len(basin.Vertices) > maxBasin {
			maxBasin = len(basin.Vertices)
		}
	}
	if totalBasin > 0 {
		d.LargestBasinFraction = float64(maxBasin) / float64(totalBasin)
	}
	return d
}

func computeTemperatureDiagnostics(vertices []Vector3D, elevation []float64, seaLevel float64, adj *FlatAdjacency, result *TemperatureResult) TemperatureDiagnostics {
	stats := result.ComputeStats(elevation, seaLevel)
	d := TemperatureDiagnostics{
		MeanC:              stats.MeanC,
		LandMeanC:          stats.LandMeanC,
		OceanMeanC:         stats.OceanMeanC,
		MinC:               stats.MinC,
		MaxC:               stats.MaxC,
		LandOceanContrastC: stats.LandMeanC - stats.OceanMeanC,
		Converged:          result.Converged,
	}

	equatorTemps := make([]float64, 0, len(vertices)/6)
	polarTemps := make([]float64, 0, len(vertices)/6)
	residuals := make([]float64, 0, len(vertices))
	var freezeLand, landN, freezeOcean, oceanN int
	xs := make([]float64, 0, len(vertices))
	ys := make([]float64, 0, len(vertices))

	for i, tC := range result.TemperatureCelsius {
		absLat := math.Abs(getLatitudeDeg(vertices[i]))
		xs = append(xs, absLat)
		ys = append(ys, tC)
		if absLat < 15 {
			equatorTemps = append(equatorTemps, tC)
		}
		if absLat >= 60 {
			polarTemps = append(polarTemps, tC)
		}
		localMean, ok := neighborMean(i, result.TemperatureCelsius, adj)
		if ok {
			resid := math.Abs(tC - localMean)
			residuals = append(residuals, resid)
			if resid > 12 {
				d.LocalAnomalyFraction++
			}
		}
		if elevation[i] >= seaLevel {
			landN++
			if tC <= 0 {
				freezeLand++
			}
		} else {
			oceanN++
			if tC <= 0 {
				freezeOcean++
			}
		}
	}
	d.EquatorMeanC = mean(equatorTemps)
	d.PolarMeanC = mean(polarTemps)
	d.EquatorPoleGradientC = d.EquatorMeanC - d.PolarMeanC
	d.AbsLatitudeTempCorr = corr(xs, ys)
	d.LocalResidualP95C = percentile(residuals, 0.95)
	if len(residuals) > 0 {
		d.LocalAnomalyFraction /= float64(len(residuals))
	}
	d.FreezingLandFraction = safeFrac(freezeLand, landN)
	d.FreezingOceanFraction = safeFrac(freezeOcean, oceanN)
	return d
}

func computePrecipitationDiagnostics(vertices []Vector3D, elevation []float64, seaLevel float64, adj *FlatAdjacency, wind *WindResult, temperature *TemperatureResult, result *PrecipitationResult) PrecipitationDiagnostics {
	d := PrecipitationDiagnostics{}
	landPrecip := make([]float64, 0, len(vertices)/3)
	coastalPrecip := make([]float64, 0, len(vertices)/8)
	inlandPrecip := make([]float64, 0, len(vertices)/4)
	onshoreCoastal := make([]float64, 0, len(vertices)/10)
	offshoreCoastal := make([]float64, 0, len(vertices)/10)
	tropical := make([]float64, 0, len(vertices)/10)
	subtropical := make([]float64, 0, len(vertices)/10)
	coldCoastal := make([]float64, 0, len(vertices)/20)
	coldInterior := make([]float64, 0, len(vertices)/20)
	coldAlpine := make([]float64, 0, len(vertices)/30)
	totalRain := 0.0
	totalSnow := 0.0
	coldCoastalRain := 0.0
	coldCoastalSnow := 0.0
	coldInteriorRain := 0.0
	coldInteriorSnow := 0.0
	logResidualAnoms := 0
	residualN := 0
	for i, p := range result.Precipitation {
		if elevation[i] < seaLevel {
			continue
		}
		landPrecip = append(landPrecip, p)
		if i < len(result.Rainfall) {
			totalRain += result.Rainfall[i]
		}
		if i < len(result.Snowfall) {
			totalSnow += result.Snowfall[i]
		}
		switch {
		case p < 25:
			d.DryLandFraction++
		case p > 200:
			d.WetLandFraction++
		}
		if p > 400 {
			d.ExtremeWetLandFraction++
		}
		absLat := math.Abs(getLatitudeDeg(vertices[i]))
		if absLat < 15 {
			tropical = append(tropical, p)
		}
		if absLat >= 15 && absLat < 35 {
			subtropical = append(subtropical, p)
		}
		if isCoastalLand(i, elevation, seaLevel, adj) {
			coastalPrecip = append(coastalPrecip, p)
			if wind != nil {
				onshore := coastalOnshoreScore(i, vertices, elevation, seaLevel, adj, wind.SurfaceWind)
				switch {
				case onshore >= 0.18:
					onshoreCoastal = append(onshoreCoastal, p)
				case onshore <= 0.05:
					offshoreCoastal = append(offshoreCoastal, p)
				}
			}
		} else {
			inlandPrecip = append(inlandPrecip, p)
		}
		if temperature != nil && i < len(temperature.TemperatureCelsius) && temperature.TemperatureCelsius[i] <= -5.0 {
			if isCoastalLand(i, elevation, seaLevel, adj) {
				coldCoastal = append(coldCoastal, p)
				if i < len(result.Rainfall) {
					coldCoastalRain += result.Rainfall[i]
				}
				if i < len(result.Snowfall) {
					coldCoastalSnow += result.Snowfall[i]
				}
			} else {
				coldInterior = append(coldInterior, p)
				if i < len(result.Rainfall) {
					coldInteriorRain += result.Rainfall[i]
				}
				if i < len(result.Snowfall) {
					coldInteriorSnow += result.Snowfall[i]
				}
			}
			if elevation[i] >= 1500 {
				coldAlpine = append(coldAlpine, p)
			}
		}
		localMean, ok := neighborMean(i, result.Precipitation, adj)
		if ok {
			residualN++
			if math.Abs(math.Log1p(p)-math.Log1p(math.Max(localMean, 0))) > math.Log(4) {
				logResidualAnoms++
			}
		}
	}
	nLand := len(landPrecip)
	d.LandMean = mean(landPrecip)
	d.LandP90 = percentile(landPrecip, 0.90)
	d.Max = percentile(landPrecip, 1.0)
	if totalRain+totalSnow > 1e-9 {
		d.RainFraction = totalRain / (totalRain + totalSnow)
		d.SnowFraction = totalSnow / (totalRain + totalSnow)
	}
	if nLand > 0 {
		d.DryLandFraction /= float64(nLand)
		d.WetLandFraction /= float64(nLand)
		d.ExtremeWetLandFraction /= float64(nLand)
	}
	if mean(subtropical) > 1 {
		d.TropicalToSubtropicRain = mean(tropical) / mean(subtropical)
	}
	if mean(inlandPrecip) > 1 {
		d.CoastalWetnessRatio = mean(coastalPrecip) / mean(inlandPrecip)
	}
	if residualN > 0 {
		d.LocalAnomalyFraction = float64(logResidualAnoms) / float64(residualN)
	}
	d.OrographicContrast = computeOrographicContrast(vertices, elevation, seaLevel, adj, wind, result.Precipitation)
	d.ColdCoastalMean = mean(coldCoastal)
	d.ColdInteriorMean = mean(coldInterior)
	d.ColdAlpineMean = mean(coldAlpine)
	d.OnshoreCoastalMean = mean(onshoreCoastal)
	d.OffshoreCoastalMean = mean(offshoreCoastal)
	if d.OffshoreCoastalMean > 1 {
		d.OnshoreOffshoreRatio = d.OnshoreCoastalMean / d.OffshoreCoastalMean
	}
	if coldCoastalRain+coldCoastalSnow > 1e-9 {
		d.ColdCoastalSnowFraction = coldCoastalSnow / (coldCoastalRain + coldCoastalSnow)
	}
	if coldInteriorRain+coldInteriorSnow > 1e-9 {
		d.ColdInteriorSnowFraction = coldInteriorSnow / (coldInteriorRain + coldInteriorSnow)
	}
	return d
}

func computeOrographicContrast(vertices []Vector3D, elevation []float64, seaLevel float64, adj *FlatAdjacency, wind *WindResult, precip []float64) float64 {
	if wind == nil {
		return 0
	}
	ratios := make([]float64, 0, 64)
	for i, w := range wind.SurfaceWind {
		if elevation[i] < 1500 {
			continue
		}
		speed := Length(w)
		if speed < 0.02 {
			continue
		}
		var windward, leeward []float64
		for _, k := range adj.GetNeighbors(i) {
			if k < 0 || k >= len(vertices) || elevation[k] < seaLevel {
				continue
			}
			dir := Normalize(Sub(vertices[k], vertices[i]))
			along := Dot(Scale(w, 1.0/speed), dir)
			switch {
			case along < -0.2:
				windward = append(windward, precip[k])
			case along > 0.2:
				leeward = append(leeward, precip[k])
			}
		}
		if len(windward) > 0 && len(leeward) > 0 && mean(leeward) > 1 {
			ratios = append(ratios, mean(windward)/mean(leeward))
		}
	}
	return mean(ratios)
}
