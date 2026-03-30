package climgen

import (
	"fmt"
	"strings"
)

// FormatClimateDiagnostics renders a compact optimization-oriented summary.
func FormatClimateDiagnostics(d ClimateDiagnostics) string {
	var sb strings.Builder
	sb.WriteString("=== CLIMATE DIAGNOSTICS ===\n")

	if d.Wind.MaxSpeed > 0 {
		sb.WriteString("WIND:\n")
		sb.WriteString(fmt.Sprintf("  mean/p95/max speed:      %.3f / %.3f / %.3f\n", d.Wind.MeanSpeed, d.Wind.P95Speed, d.Wind.MaxSpeed))
		sb.WriteString(fmt.Sprintf("  tangent max error:       %.6f\n", d.Wind.TangencyMaxError))
		sb.WriteString(fmt.Sprintf("  trade/westerly/polar ok: %.1f%% / %.1f%% / %.1f%%\n", d.Wind.TradeWestFraction*100, d.Wind.WesterlyEastFraction*100, d.Wind.PolarWestFraction*100))
		sb.WriteString(fmt.Sprintf("  Hadley/Ferrel/Polar mer: %.1f%% / %.1f%% / %.1f%%\n", d.Wind.HadleyConvergenceFraction*100, d.Wind.FerrelPolewardFraction*100, d.Wind.PolarEquatorwardFraction*100))
		sb.WriteString(fmt.Sprintf("  calm fraction:           %.1f%%\n", d.Wind.CalmFraction*100))
	}

	if d.Currents.MaxSpeed > 0 {
		sb.WriteString("CURRENTS:\n")
		sb.WriteString(fmt.Sprintf("  mean/p95/max speed:      %.3f / %.3f / %.3f\n", d.Currents.MeanSpeed, d.Currents.P95Speed, d.Currents.MaxSpeed))
		sb.WriteString(fmt.Sprintf("  coherence / speed CoV:   %.3f / %.3f\n", d.Currents.FlowCoherence, d.Currents.SpeedCoV))
		sb.WriteString(fmt.Sprintf("  coast normal p95:        %.3f\n", d.Currents.CoastNormalP95))
		sb.WriteString(fmt.Sprintf("  coast violation pct:     %.2f%%\n", d.Currents.CoastNormalViolationPt*100))
		sb.WriteString(fmt.Sprintf("  speed anomaly pct:       %.2f%%\n", d.Currents.SpeedAnomalyFraction*100))
		sb.WriteString(fmt.Sprintf("  gateway pct / align:     %.2f%% / %.3f\n", d.Currents.GatewayFraction*100, d.Currents.GatewayAlignment))
		sb.WriteString(fmt.Sprintf("  basins / largest share:  %d / %.1f%%\n", d.Currents.BasinCount, d.Currents.LargestBasinFraction*100))
	}

	if d.OceanClimate.SourceAnomalyP90AbsC > 0 {
		sb.WriteString("OCEAN CLIMATE:\n")
		sb.WriteString(fmt.Sprintf("  source anomaly mean/p90: %.2f / %.2f C\n", d.OceanClimate.SourceAnomalyMeanAbsC, d.OceanClimate.SourceAnomalyP90AbsC))
		sb.WriteString(fmt.Sprintf("  west-warm / east-cold:   %.2f / %.2f C\n", d.OceanClimate.WarmWesternBoundarySignalC, d.OceanClimate.ColdEasternBoundarySignalC))
		sb.WriteString(fmt.Sprintf("  coastal coupling corr:   %.3f\n", d.OceanClimate.CoastalLandCouplingCorr))
		if d.OceanClimate.WarmAdjacentLandResidualC != 0 || d.OceanClimate.ColdAdjacentLandResidualC != 0 {
			sb.WriteString(fmt.Sprintf("  warm/cold coast resid:   %.2f / %.2f C\n", d.OceanClimate.WarmAdjacentLandResidualC, d.OceanClimate.ColdAdjacentLandResidualC))
		}
	}

	if d.Temperature.MaxC != 0 || d.Temperature.MinC != 0 {
		sb.WriteString("TEMPERATURE:\n")
		sb.WriteString(fmt.Sprintf("  mean land/ocean:         %.1f / %.1f / %.1f C\n", d.Temperature.MeanC, d.Temperature.LandMeanC, d.Temperature.OceanMeanC))
		sb.WriteString(fmt.Sprintf("  min/max:                 %.1f / %.1f C\n", d.Temperature.MinC, d.Temperature.MaxC))
		sb.WriteString(fmt.Sprintf("  equator/polar mean:      %.1f / %.1f C\n", d.Temperature.EquatorMeanC, d.Temperature.PolarMeanC))
		sb.WriteString(fmt.Sprintf("  equator-pole gradient:   %.1f C\n", d.Temperature.EquatorPoleGradientC))
		sb.WriteString(fmt.Sprintf("  abs(lat)-temp corr:      %.3f\n", d.Temperature.AbsLatitudeTempCorr))
		sb.WriteString(fmt.Sprintf("  local residual p95:      %.1f C\n", d.Temperature.LocalResidualP95C))
		sb.WriteString(fmt.Sprintf("  local anomaly pct:       %.2f%%\n", d.Temperature.LocalAnomalyFraction*100))
	}

	if d.Precipitation.Max > 0 {
		sb.WriteString("PRECIPITATION:\n")
		sb.WriteString(fmt.Sprintf("  mean/p90/max land:       %.1f / %.1f / %.1f cm/yr\n", d.Precipitation.LandMean, d.Precipitation.LandP90, d.Precipitation.Max))
		sb.WriteString(fmt.Sprintf("  rain/snow fraction:      %.1f%% / %.1f%%\n", d.Precipitation.RainFraction*100, d.Precipitation.SnowFraction*100))
		sb.WriteString(fmt.Sprintf("  dry/wet/extreme-wet:     %.1f%% / %.1f%% / %.1f%%\n", d.Precipitation.DryLandFraction*100, d.Precipitation.WetLandFraction*100, d.Precipitation.ExtremeWetLandFraction*100))
		sb.WriteString(fmt.Sprintf("  coastal wetness ratio:   %.2f\n", d.Precipitation.CoastalWetnessRatio))
		if d.Precipitation.OnshoreOffshoreRatio > 0 {
			sb.WriteString(fmt.Sprintf("  on/offshore coast rain:  %.2f (%.1f / %.1f)\n", d.Precipitation.OnshoreOffshoreRatio, d.Precipitation.OnshoreCoastalMean, d.Precipitation.OffshoreCoastalMean))
		}
		sb.WriteString(fmt.Sprintf("  tropical/subtropic rain: %.2f\n", d.Precipitation.TropicalToSubtropicRain))
		sb.WriteString(fmt.Sprintf("  orographic contrast:     %.2f\n", d.Precipitation.OrographicContrast))
		if d.Precipitation.ColdCoastalMean > 0 || d.Precipitation.ColdInteriorMean > 0 || d.Precipitation.ColdAlpineMean > 0 {
			sb.WriteString(fmt.Sprintf("  cold coast/interior/alp: %.1f / %.1f / %.1f cm/yr\n", d.Precipitation.ColdCoastalMean, d.Precipitation.ColdInteriorMean, d.Precipitation.ColdAlpineMean))
			sb.WriteString(fmt.Sprintf("  cold coast/int snow:     %.1f%% / %.1f%%\n", d.Precipitation.ColdCoastalSnowFraction*100, d.Precipitation.ColdInteriorSnowFraction*100))
		}
		sb.WriteString(fmt.Sprintf("  local anomaly pct:       %.2f%%\n", d.Precipitation.LocalAnomalyFraction*100))
	}

	if len(d.Flags) > 0 {
		sb.WriteString("FLAGS:\n")
		for _, flag := range d.Flags {
			sb.WriteString("  - " + flag + "\n")
		}
	}

	return sb.String()
}
