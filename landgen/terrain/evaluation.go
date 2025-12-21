package terrain

import (
	"fmt"
	"math"
	"strings"
)

// --- Metric Targets ---

// MetricTarget defines the acceptable range and ideal value for a metric.
type MetricTarget struct {
	Name   string  // Display name
	Min    float64 // Minimum acceptable value
	Max    float64 // Maximum acceptable value
	Ideal  float64 // Ideal target value
	Weight float64 // Weight in scoring (points out of 100)
}

// PrimaryMetricTargets defines the targets for primary coverage and elevation metrics.
var PrimaryMetricTargets = []MetricTarget{
	{"Land Coverage", 0.27, 0.32, EarthLandCoverage, 12},
	{"Ocean Coverage", 0.68, 0.73, EarthOceanCoverage, 12},
	{"Mean Land Elevation", 600, 1000, EarthMeanLandElevation, 9},
	{"Mean Ocean Depth", -4000, -3400, EarthMeanOceanDepth, 9},
	{"Mountain Coverage", 0.015, 0.030, EarthMountainCoverage, 6},
	{"Deep Ocean Coverage", 0.015, 0.040, EarthDeepOceanCoverage, 6},
	{"Shelf Coverage", 0.05, 0.12, EarthShelfCoverage, 6},
}

// HypsometricThresholds defines the elevation thresholds to check for hypsometric curve.
var HypsometricThresholds = []float64{-6000, -5000, -4000, -3000, 0, 500, 1000, 2000, 3000}

// CoastlineMetricTargets defines targets for coastline irregularity.
var CoastlineMetricTargets = []MetricTarget{
	{"Fractal Dimension", 1.15, 1.50, EarthCoastlineFractalD, 5},
	{"Tortuosity Ratio", 2.0, 5.0, EarthTortuosityRatio, 5},
}

// ContinentMetricTargets defines targets for continental distribution.
var ContinentMetricTargets = []MetricTarget{
	{"Major Landmasses", 4, 10, float64(EarthMajorLandmasses), 2.5},
	{"Continent Gini", 0.30, 0.60, EarthContinentGini, 2.5},
}

// --- Scoring Functions ---

// EvaluateTerrain computes a score from 0-100 based on Earth similarity.
// Returns an EvaluationResult with score, metrics, and list of failed metrics.
func EvaluateTerrain(sites []Vector3D, elevation []float64) EvaluationResult {
	metrics := ComputeMetrics(sites, elevation)
	return EvaluateMetrics(metrics)
}

// EvaluateMetrics scores pre-computed metrics against Earth targets.
func EvaluateMetrics(metrics TerrainMetrics) EvaluationResult {
	result := EvaluationResult{
		Metrics:       metrics,
		FailedMetrics: []string{},
	}

	totalScore := 0.0
	maxScore := 0.0

	// Score primary metrics (60 points)
	primaryValues := []float64{
		metrics.LandCoverage,
		metrics.OceanCoverage,
		metrics.MeanLandElevation,
		metrics.MeanOceanDepth,
		metrics.MountainCoverage,
		metrics.DeepOceanCoverage,
		metrics.ShelfCoverage,
	}

	for i, target := range PrimaryMetricTargets {
		score, passed := scoreMetric(primaryValues[i], target)
		totalScore += score
		maxScore += target.Weight
		if !passed {
			result.FailedMetrics = append(result.FailedMetrics, target.Name)
		}
	}

	// Score hypsometric curve (25 points)
	hypsWeight := 25.0 / float64(len(HypsometricThresholds))
	for _, threshold := range HypsometricThresholds {
		earthTarget, ok := HypsometricTargets[threshold]
		if !ok {
			continue
		}
		actual := metrics.HypsometricCurve[threshold]
		tolerance := 0.05 // 5% tolerance

		target := MetricTarget{
			Name:   fmt.Sprintf("Hypsometric %.0fm", threshold),
			Min:    earthTarget - tolerance,
			Max:    earthTarget + tolerance,
			Ideal:  earthTarget,
			Weight: hypsWeight,
		}

		score, passed := scoreMetric(actual, target)
		totalScore += score
		maxScore += hypsWeight
		if !passed {
			result.FailedMetrics = append(result.FailedMetrics, target.Name)
		}
	}

	// Score coastline metrics (10 points) - only if computed
	if metrics.FractalDimension > 0 {
		coastlineValues := []float64{metrics.FractalDimension, metrics.TortuosityRatio}
		for i, target := range CoastlineMetricTargets {
			score, passed := scoreMetric(coastlineValues[i], target)
			totalScore += score
			maxScore += target.Weight
			if !passed {
				result.FailedMetrics = append(result.FailedMetrics, target.Name)
			}
		}
	}

	// Score continent metrics (5 points) - only if computed
	if metrics.NumMajorLandmasses > 0 {
		continentValues := []float64{float64(metrics.NumMajorLandmasses), metrics.ContinentGini}
		for i, target := range ContinentMetricTargets {
			score, passed := scoreMetric(continentValues[i], target)
			totalScore += score
			maxScore += target.Weight
			if !passed {
				result.FailedMetrics = append(result.FailedMetrics, target.Name)
			}
		}
	}

	// Calculate final score
	if maxScore > 0 {
		result.Score = (totalScore / maxScore) * 100
	}

	// Determine if passed (score >= 75 and no critical failures)
	result.Passed = result.Score >= 75.0 && len(result.FailedMetrics) <= 3

	return result
}

// scoreMetric returns the score for a single metric and whether it passed.
// Returns (score, passed) where score is in [0, weight] and passed is true if in range.
func scoreMetric(value float64, target MetricTarget) (score float64, passed bool) {
	passed = value >= target.Min && value <= target.Max

	if passed {
		// Inside range: score based on distance from ideal
		rangeWidth := target.Max - target.Min
		if rangeWidth == 0 {
			return target.Weight, true
		}
		distFromIdeal := math.Abs(value - target.Ideal)
		// Max 20% penalty for being at range edge
		penalty := (distFromIdeal / rangeWidth) * 0.2
		score = target.Weight * (1.0 - penalty)
	} else {
		// Outside range: partial score based on how far outside
		rangeWidth := target.Max - target.Min
		if rangeWidth == 0 {
			return 0, false
		}

		var distance float64
		if value < target.Min {
			distance = target.Min - value
		} else {
			distance = value - target.Max
		}

		// Score decreases linearly as distance increases
		// At distance = rangeWidth, score = 0
		ratio := distance / rangeWidth
		score = target.Weight * math.Max(0, 1-ratio)
	}

	return score, passed
}

// --- Formatted Output ---

// FormatEvaluationResult returns a formatted string of the evaluation results.
func FormatEvaluationResult(result EvaluationResult) string {
	var sb strings.Builder

	sb.WriteString("=== TERRAIN EVALUATION ===\n")
	sb.WriteString(fmt.Sprintf("Score: %.1f%% (target: 75%%+)\n\n", result.Score))

	// Primary metrics
	sb.WriteString("PRIMARY METRICS:\n")
	formatMetricLine(&sb, "Land Coverage", result.Metrics.LandCoverage*100, EarthLandCoverage*100, 2, "%", true)
	formatMetricLine(&sb, "Ocean Coverage", result.Metrics.OceanCoverage*100, EarthOceanCoverage*100, 2, "%", true)
	formatMetricLine(&sb, "Mean Land Elev", result.Metrics.MeanLandElevation, EarthMeanLandElevation, 200, "m", false)
	formatMetricLine(&sb, "Mean Ocean Depth", result.Metrics.MeanOceanDepth, EarthMeanOceanDepth, 300, "m", false)
	formatMetricLine(&sb, "Mountain Coverage", result.Metrics.MountainCoverage*100, EarthMountainCoverage*100, 0.5, "%", true)
	formatMetricLine(&sb, "Deep Ocean", result.Metrics.DeepOceanCoverage*100, EarthDeepOceanCoverage*100, 1.5, "%", true)
	formatMetricLine(&sb, "Continental Shelf", result.Metrics.ShelfCoverage*100, EarthShelfCoverage*100, 3, "%", true)
	sb.WriteString("\n")

	// Hypsometric curve (selected thresholds)
	sb.WriteString("HYPSOMETRIC CURVE:\n")
	keyThresholds := []float64{-4000, 0, 1000}
	for _, t := range keyThresholds {
		actual := result.Metrics.HypsometricCurve[t]
		target := HypsometricTargets[t]
		formatMetricLine(&sb, fmt.Sprintf("Below %.0fm", t), actual*100, target*100, 5, "%", true)
	}
	sb.WriteString("\n")

	// Coastline metrics (if available)
	if result.Metrics.FractalDimension > 0 {
		sb.WriteString("COASTLINE METRICS:\n")
		formatMetricLine(&sb, "Fractal Dimension", result.Metrics.FractalDimension, EarthCoastlineFractalD, 0.15, "", false)
		formatMetricLine(&sb, "Tortuosity Ratio", result.Metrics.TortuosityRatio, EarthTortuosityRatio, 2.0, "", false)
		sb.WriteString("\n")
	}

	// Continental distribution (if available)
	if result.Metrics.NumMajorLandmasses > 0 {
		sb.WriteString("CONTINENTAL DISTRIBUTION:\n")
		formatMetricLineInt(&sb, "Major Landmasses", result.Metrics.NumMajorLandmasses, EarthMajorLandmasses, 4, 10)
		formatMetricLine(&sb, "Continent Gini", result.Metrics.ContinentGini, EarthContinentGini, 0.15, "", false)
		sb.WriteString("\n")
	}

	// Elevation statistics
	sb.WriteString("ELEVATION STATISTICS:\n")
	sb.WriteString(fmt.Sprintf("  Global Mean:        %.0fm\n", result.Metrics.GlobalMean))
	sb.WriteString(fmt.Sprintf("  Global Std Dev:     %.0fm\n", result.Metrics.GlobalStdDev))
	sb.WriteString(fmt.Sprintf("  Maximum:            %.0fm\n", result.Metrics.MaxElevation))
	sb.WriteString(fmt.Sprintf("  Minimum:            %.0fm\n", result.Metrics.MinElevation))
	sb.WriteString("\n")

	// Summary
	if len(result.FailedMetrics) == 0 {
		sb.WriteString("All metrics within acceptable range.\n")
	} else {
		sb.WriteString(fmt.Sprintf("Failed metrics (%d): %s\n",
			len(result.FailedMetrics),
			strings.Join(result.FailedMetrics, ", ")))
	}

	if result.Passed {
		sb.WriteString("\n[PASS] Score >= 75% with acceptable metric failures.\n")
	} else {
		sb.WriteString("\n[FAIL] Score < 75% or too many failed metrics.\n")
	}

	return sb.String()
}

// formatMetricLine formats a single metric line with pass/fail indicator.
func formatMetricLine(sb *strings.Builder, name string, actual, target, tolerance float64, unit string, isPercent bool) {
	var format string
	if isPercent {
		format = "%.1f%s"
	} else {
		format = "%.0f%s"
	}

	actualStr := fmt.Sprintf(format, actual, unit)
	targetStr := fmt.Sprintf(format, target, unit)

	passed := math.Abs(actual-target) <= tolerance
	status := "PASS"
	if !passed {
		status = "FAIL"
	}

	sb.WriteString(fmt.Sprintf("  %-18s %10s (target: %s +/-%v%s)  %s\n",
		name+":", actualStr, targetStr, tolerance, unit, status))
}

// formatMetricLineInt formats an integer metric line.
func formatMetricLineInt(sb *strings.Builder, name string, actual, target, min, max int) {
	passed := actual >= min && actual <= max
	status := "PASS"
	if !passed {
		status = "FAIL"
	}

	sb.WriteString(fmt.Sprintf("  %-18s %10d (target: %d-%d)           %s\n",
		name+":", actual, min, max, status))
}

// --- Quick Evaluation ---

// QuickScore returns just the score without full evaluation details.
// Useful for optimization loops.
func QuickScore(sites []Vector3D, elevation []float64) float64 {
	result := EvaluateTerrain(sites, elevation)
	return result.Score
}

// QuickScoreFromMetrics returns score from pre-computed metrics.
func QuickScoreFromMetrics(metrics TerrainMetrics) float64 {
	result := EvaluateMetrics(metrics)
	return result.Score
}

// --- Metric Comparison ---

// MetricComparison holds the comparison of a single metric to its target.
type MetricComparison struct {
	Name      string
	Actual    float64
	Target    float64
	Min       float64
	Max       float64
	Deviation float64 // (actual - target) / rangeWidth
	Passed    bool
	Score     float64
}

// CompareToTargets returns detailed comparisons for all metrics.
func CompareToTargets(metrics TerrainMetrics) []MetricComparison {
	var comparisons []MetricComparison

	// Primary metrics
	primaryValues := []float64{
		metrics.LandCoverage,
		metrics.OceanCoverage,
		metrics.MeanLandElevation,
		metrics.MeanOceanDepth,
		metrics.MountainCoverage,
		metrics.DeepOceanCoverage,
		metrics.ShelfCoverage,
	}

	for i, target := range PrimaryMetricTargets {
		score, passed := scoreMetric(primaryValues[i], target)
		rangeWidth := target.Max - target.Min
		deviation := 0.0
		if rangeWidth > 0 {
			deviation = (primaryValues[i] - target.Ideal) / rangeWidth
		}

		comparisons = append(comparisons, MetricComparison{
			Name:      target.Name,
			Actual:    primaryValues[i],
			Target:    target.Ideal,
			Min:       target.Min,
			Max:       target.Max,
			Deviation: deviation,
			Passed:    passed,
			Score:     score,
		})
	}

	return comparisons
}

// --- Score Breakdown ---

// ScoreBreakdown shows how the score is composed.
type ScoreBreakdown struct {
	PrimaryScore     float64
	PrimaryMax       float64
	HypsometricScore float64
	HypsometricMax   float64
	CoastlineScore   float64
	CoastlineMax     float64
	ContinentScore   float64
	ContinentMax     float64
	TotalScore       float64
	TotalMax         float64
	FinalPercent     float64
}

// GetScoreBreakdown returns the breakdown of scores by category.
func GetScoreBreakdown(metrics TerrainMetrics) ScoreBreakdown {
	breakdown := ScoreBreakdown{}

	// Primary metrics
	primaryValues := []float64{
		metrics.LandCoverage,
		metrics.OceanCoverage,
		metrics.MeanLandElevation,
		metrics.MeanOceanDepth,
		metrics.MountainCoverage,
		metrics.DeepOceanCoverage,
		metrics.ShelfCoverage,
	}

	for i, target := range PrimaryMetricTargets {
		score, _ := scoreMetric(primaryValues[i], target)
		breakdown.PrimaryScore += score
		breakdown.PrimaryMax += target.Weight
	}

	// Hypsometric curve
	hypsWeight := 25.0 / float64(len(HypsometricThresholds))
	for _, threshold := range HypsometricThresholds {
		earthTarget, ok := HypsometricTargets[threshold]
		if !ok {
			continue
		}
		actual := metrics.HypsometricCurve[threshold]

		target := MetricTarget{
			Min:    earthTarget - 0.05,
			Max:    earthTarget + 0.05,
			Ideal:  earthTarget,
			Weight: hypsWeight,
		}

		score, _ := scoreMetric(actual, target)
		breakdown.HypsometricScore += score
		breakdown.HypsometricMax += hypsWeight
	}

	// Coastline metrics (if available)
	if metrics.FractalDimension > 0 {
		coastlineValues := []float64{metrics.FractalDimension, metrics.TortuosityRatio}
		for i, target := range CoastlineMetricTargets {
			score, _ := scoreMetric(coastlineValues[i], target)
			breakdown.CoastlineScore += score
			breakdown.CoastlineMax += target.Weight
		}
	}

	// Continent metrics (if available)
	if metrics.NumMajorLandmasses > 0 {
		continentValues := []float64{float64(metrics.NumMajorLandmasses), metrics.ContinentGini}
		for i, target := range ContinentMetricTargets {
			score, _ := scoreMetric(continentValues[i], target)
			breakdown.ContinentScore += score
			breakdown.ContinentMax += target.Weight
		}
	}

	// Totals
	breakdown.TotalScore = breakdown.PrimaryScore + breakdown.HypsometricScore +
		breakdown.CoastlineScore + breakdown.ContinentScore
	breakdown.TotalMax = breakdown.PrimaryMax + breakdown.HypsometricMax +
		breakdown.CoastlineMax + breakdown.ContinentMax

	if breakdown.TotalMax > 0 {
		breakdown.FinalPercent = (breakdown.TotalScore / breakdown.TotalMax) * 100
	}

	return breakdown
}

// FormatScoreBreakdown returns a formatted string of the score breakdown.
func FormatScoreBreakdown(breakdown ScoreBreakdown) string {
	var sb strings.Builder

	sb.WriteString("=== SCORE BREAKDOWN ===\n")
	sb.WriteString(fmt.Sprintf("Primary Metrics:    %5.1f / %5.1f (%.1f%%)\n",
		breakdown.PrimaryScore, breakdown.PrimaryMax,
		100*breakdown.PrimaryScore/breakdown.PrimaryMax))
	sb.WriteString(fmt.Sprintf("Hypsometric Curve:  %5.1f / %5.1f (%.1f%%)\n",
		breakdown.HypsometricScore, breakdown.HypsometricMax,
		100*breakdown.HypsometricScore/breakdown.HypsometricMax))

	if breakdown.CoastlineMax > 0 {
		sb.WriteString(fmt.Sprintf("Coastline Metrics:  %5.1f / %5.1f (%.1f%%)\n",
			breakdown.CoastlineScore, breakdown.CoastlineMax,
			100*breakdown.CoastlineScore/breakdown.CoastlineMax))
	}

	if breakdown.ContinentMax > 0 {
		sb.WriteString(fmt.Sprintf("Continent Metrics:  %5.1f / %5.1f (%.1f%%)\n",
			breakdown.ContinentScore, breakdown.ContinentMax,
			100*breakdown.ContinentScore/breakdown.ContinentMax))
	}

	sb.WriteString(fmt.Sprintf("─────────────────────────────────\n"))
	sb.WriteString(fmt.Sprintf("TOTAL:              %5.1f / %5.1f = %.1f%%\n",
		breakdown.TotalScore, breakdown.TotalMax, breakdown.FinalPercent))

	return sb.String()
}
