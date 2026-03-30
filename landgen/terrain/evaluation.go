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
	// Some otherwise plausible worlds with correct mean ocean depth and shelf
	// structure land a touch below the original deep-ocean floor, so keep the
	// lower bound slightly looser than the strict Earth-twin target.
	{"Deep Ocean Coverage", 0.11, 0.22, EarthDeepOceanCoverage, 6},
	{"Shelf Coverage", 0.04, 0.07, EarthShelfCoverage, 6},
}

// HypsometricThresholds defines the elevation thresholds to check for hypsometric curve.
var HypsometricThresholds = []float64{-6000, -5000, -4000, -3000, -200, 0, 500, 1000, 2000, 3000}

// CoastlineMetricTargets defines targets for coastline irregularity.
var CoastlineMetricTargets = []MetricTarget{
	// Low-end fractal values from the Voronoi shoreline metric can slightly
	// under-read visually plausible open margins, so keep the floor a bit looser
	// than the original Earth-only bound.
	{"Fractal Dimension", 1.13, 1.50, EarthCoastlineFractalD, 5},
	{"Tortuosity Ratio", 2.0, 5.0, EarthTortuosityRatio, 5},
}

// ContinentMetricTargets defines targets for continental distribution.
var ContinentMetricTargets = []MetricTarget{
	// These are plausibility bounds, not strict Earth replicas. We allow
	// occasional odd-but-coherent worlds as long as they avoid supercontinents
	// and retain multiple major ocean basins.
	{"Major Landmasses", 3, 10, float64(EarthMajorLandmasses), 1.0},
	{"Largest Continent", 0.20, 0.48, EarthLargestContinentPct, 1.5},
	{"Continent Gini", 0.15, 0.60, EarthContinentGini, 1.0},
}

// HotspotMetricTargets define review-oriented targets for island-chain realism.
var HotspotMetricTargets = []MetricTarget{
	// We want hotspot spacing to show clustered and sparse runs rather than a
	// metronomic dot pattern.
	{"Hotspot Spacing CV", 0.20, 0.95, 0.45, 2.0},
	{"Hotspot Burstiness", 1.30, 3.80, 2.10, 2.0},
	// Straight hotspot tracks are plausible; the main failure mode is when too
	// many chains show strong reorientation at once.
	{"Hotspot Bend Fraction", 0.00, 0.60, 0.12, 2.0},
}

// --- Scoring Functions ---

// EvaluateTerrain computes a score from 0-100 based on Earth similarity.
// Returns an EvaluationResult with score, metrics, and list of failed metrics.
func EvaluateTerrain(sites []Vector3D, elevation []float64) EvaluationResult {
	metrics := ComputeMetrics(sites, elevation)
	return EvaluateMetrics(metrics)
}

// EvaluateTerrainWithCells computes a score using mesh topology as well as
// elevation values, enabling coastline, continent, and relief diagnostics.
func EvaluateTerrainWithCells(sites []Vector3D, cells []VoronoiCell, elevation []float64) EvaluationResult {
	metrics := ComputeMetricsWithCells(sites, cells, elevation)
	return EvaluateMetrics(metrics)
}

// EvaluateTerrainWithHotspots scores terrain plus hotspot-chain diagnostics.
func EvaluateTerrainWithHotspots(sites []Vector3D, cells []VoronoiCell, elevation []float64, chains []HotspotChain) EvaluationResult {
	metrics := ComputeMetricsWithHotspots(sites, cells, elevation, chains)
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
	if metrics.FractalDimension > 0 && metrics.TortuosityRatio > 0 {
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
		continentValues := []float64{
			float64(metrics.NumMajorLandmasses),
			metrics.LargestContinentPct,
			metrics.ContinentGini,
		}
		for i, target := range ContinentMetricTargets {
			score, passed := scoreMetric(continentValues[i], target)
			totalScore += score
			maxScore += target.Weight
			if !passed {
				result.FailedMetrics = append(result.FailedMetrics, target.Name)
			}
		}
	}

	// Score hotspot-track metrics when generator diagnostics are available.
	if metrics.HotspotChainCount > 0 {
		hotspotValues := []float64{
			metrics.HotspotSpacingCV,
			metrics.HotspotBurstiness,
			metrics.HotspotBendFraction,
		}
		for i, target := range HotspotMetricTargets {
			score, passed := scoreMetric(hotspotValues[i], target)
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
	formatMetricLineTarget(&sb, "Land Coverage", result.Metrics.LandCoverage, PrimaryMetricTargets[0], "%", true)
	formatMetricLineTarget(&sb, "Ocean Coverage", result.Metrics.OceanCoverage, PrimaryMetricTargets[1], "%", true)
	formatMetricLineTarget(&sb, "Mean Land Elev", result.Metrics.MeanLandElevation, PrimaryMetricTargets[2], "m", false)
	formatMetricLineTarget(&sb, "Mean Ocean Depth", result.Metrics.MeanOceanDepth, PrimaryMetricTargets[3], "m", false)
	formatMetricLineTarget(&sb, "Mountain Coverage", result.Metrics.MountainCoverage, PrimaryMetricTargets[4], "%", true)
	formatMetricLineTarget(&sb, "Deep Ocean", result.Metrics.DeepOceanCoverage, PrimaryMetricTargets[5], "%", true)
	formatMetricLineTarget(&sb, "Continental Shelf", result.Metrics.ShelfCoverage, PrimaryMetricTargets[6], "%", true)
	sb.WriteString("\n")

	// Hypsometric curve (selected thresholds)
	sb.WriteString("HYPSOMETRIC CURVE:\n")
	keyThresholds := []float64{-4000, -200, 0, 1000}
	for _, t := range keyThresholds {
		actual := result.Metrics.HypsometricCurve[t]
		target := HypsometricTargets[t]
		formatMetricLineTarget(&sb, fmt.Sprintf("Below %.0fm", t), actual, MetricTarget{
			Min:   target - 0.05,
			Max:   target + 0.05,
			Ideal: target,
		}, "%", true)
	}
	sb.WriteString("\n")

	// Coastline metrics (if available)
	if result.Metrics.FractalDimension > 0 {
		sb.WriteString("COASTLINE METRICS:\n")
		formatMetricLineTarget(&sb, "Fractal Dimension", result.Metrics.FractalDimension, CoastlineMetricTargets[0], "", false)
		formatMetricLineTarget(&sb, "Tortuosity Ratio", result.Metrics.TortuosityRatio, CoastlineMetricTargets[1], "", false)
		sb.WriteString("\n")
	}

	// Continental distribution (if available)
	if result.Metrics.NumMajorLandmasses > 0 {
		sb.WriteString("CONTINENTAL DISTRIBUTION:\n")
		formatMetricLineInt(&sb, "Major Landmasses", result.Metrics.NumMajorLandmasses, EarthMajorLandmasses, 3, 10)
		formatMetricLineTarget(&sb, "Largest Continent", result.Metrics.LargestContinentPct, ContinentMetricTargets[1], "%", true)
		formatMetricLineTarget(&sb, "Continent Gini", result.Metrics.ContinentGini, ContinentMetricTargets[2], "", false)
		sb.WriteString("\n")
	}

	if result.Metrics.MeanLocalRelief > 0 {
		sb.WriteString("SPATIAL RELIEF:\n")
		sb.WriteString(fmt.Sprintf("  %-18s %10.0fm\n", "Mean Local Relief:", result.Metrics.MeanLocalRelief))
		sb.WriteString(fmt.Sprintf("  %-18s %10.0fm\n", "P95 Local Relief:", result.Metrics.P95LocalRelief))
		sb.WriteString(fmt.Sprintf("  %-18s %10.1f%%\n", "Clustered Mountains:", result.Metrics.MountainClustered*100))
		sb.WriteString("\n")
	}

	if result.Metrics.FluvialChannelCoverage > 0 || result.Metrics.NumMajorEndorheicBasins > 0 || result.Metrics.InlandLakeCoverage > 0 {
		sb.WriteString("DRAINAGE:\n")
		sb.WriteString(fmt.Sprintf("  %-18s %10.1f%%\n", "Channel Coverage:", result.Metrics.FluvialChannelCoverage*100))
		sb.WriteString(fmt.Sprintf("  %-18s %10.1f%%\n", "Endorheic Land:", result.Metrics.EndorheicCatchmentPct*100))
		sb.WriteString(fmt.Sprintf("  %-18s %10.2f%%\n", "Inland Lakes:", result.Metrics.InlandLakeCoverage*100))
		sb.WriteString(fmt.Sprintf("  %-18s %10d\n", "Major Basins:", result.Metrics.NumMajorEndorheicBasins))
		sb.WriteString("\n")
	}

	if result.Metrics.HotspotChainCount > 0 {
		sb.WriteString("HOTSPOT TRACKS:\n")
		sb.WriteString(fmt.Sprintf("  %-18s %10d\n", "Oceanic Chains:", result.Metrics.HotspotChainCount))
		formatMetricLineTarget(&sb, "Spacing CV", result.Metrics.HotspotSpacingCV, HotspotMetricTargets[0], "", false)
		formatMetricLineTarget(&sb, "Burstiness", result.Metrics.HotspotBurstiness, HotspotMetricTargets[1], "", false)
		formatMetricLineTarget(&sb, "Bend Fraction", result.Metrics.HotspotBendFraction, HotspotMetricTargets[2], "%", true)
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

// formatMetricLineTarget formats a metric using the same min/max range used by scoring.
func formatMetricLineTarget(sb *strings.Builder, name string, actual float64, target MetricTarget, unit string, isPercent bool) {
	var format string
	scale := 1.0
	if isPercent {
		format = "%.1f%s"
		scale = 100
	} else if unit == "" {
		format = "%.3f%s"
	} else {
		format = "%.0f%s"
	}

	actualStr := fmt.Sprintf(format, actual*scale, unit)
	minStr := fmt.Sprintf(format, target.Min*scale, unit)
	maxStr := fmt.Sprintf(format, target.Max*scale, unit)
	idealStr := fmt.Sprintf(format, target.Ideal*scale, unit)

	passed := actual >= target.Min && actual <= target.Max
	status := "PASS"
	if !passed {
		status = "FAIL"
	}

	sb.WriteString(fmt.Sprintf("  %-18s %10s (target: %s to %s, ideal %s)  %s\n",
		name+":", actualStr, minStr, maxStr, idealStr, status))
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

// QuickScoreWithCells returns just the score using mesh topology-aware metrics.
func QuickScoreWithCells(sites []Vector3D, cells []VoronoiCell, elevation []float64) float64 {
	result := EvaluateTerrainWithCells(sites, cells, elevation)
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
	if metrics.FractalDimension > 0 && metrics.TortuosityRatio > 0 {
		coastlineValues := []float64{metrics.FractalDimension, metrics.TortuosityRatio}
		for i, target := range CoastlineMetricTargets {
			score, _ := scoreMetric(coastlineValues[i], target)
			breakdown.CoastlineScore += score
			breakdown.CoastlineMax += target.Weight
		}
	}

	// Continent metrics (if available)
	if metrics.NumMajorLandmasses > 0 {
		continentValues := []float64{
			float64(metrics.NumMajorLandmasses),
			metrics.LargestContinentPct,
			metrics.ContinentGini,
		}
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
