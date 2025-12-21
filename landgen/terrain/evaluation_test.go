package terrain

import (
	"math"
	"strings"
	"testing"
)

// --- scoreMetric Tests ---

func TestScoreMetricInRange(t *testing.T) {
	target := MetricTarget{
		Name:   "Test",
		Min:    0.27,
		Max:    0.32,
		Ideal:  0.292,
		Weight: 10.0,
	}

	tests := []struct {
		name       string
		value      float64
		wantPassed bool
		wantMin    float64 // minimum acceptable score
		wantMax    float64 // maximum acceptable score
	}{
		{
			name:       "at ideal",
			value:      0.292,
			wantPassed: true,
			wantMin:    9.5, // nearly full score
			wantMax:    10.0,
		},
		{
			name:       "at min edge",
			value:      0.27,
			wantPassed: true,
			wantMin:    8.0, // some penalty for being at edge
			wantMax:    9.5,
		},
		{
			name:       "at max edge",
			value:      0.32,
			wantPassed: true,
			wantMin:    8.0,
			wantMax:    9.5,
		},
		{
			name:       "in middle of range",
			value:      0.295,
			wantPassed: true,
			wantMin:    9.0,
			wantMax:    10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, passed := scoreMetric(tt.value, target)
			if passed != tt.wantPassed {
				t.Errorf("passed = %v, want %v", passed, tt.wantPassed)
			}
			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("score = %v, want in [%v, %v]", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestScoreMetricOutOfRange(t *testing.T) {
	target := MetricTarget{
		Name:   "Test",
		Min:    0.27,
		Max:    0.32,
		Ideal:  0.292,
		Weight: 10.0,
	}

	tests := []struct {
		name       string
		value      float64
		wantPassed bool
		wantMin    float64
		wantMax    float64
	}{
		{
			name:       "slightly below min",
			value:      0.26,
			wantPassed: false,
			wantMin:    5.0, // partial credit
			wantMax:    10.0,
		},
		{
			name:       "far below min",
			value:      0.10,
			wantPassed: false,
			wantMin:    0.0,
			wantMax:    2.0,
		},
		{
			name:       "slightly above max",
			value:      0.33,
			wantPassed: false,
			wantMin:    5.0,
			wantMax:    10.0,
		},
		{
			name:       "far above max",
			value:      0.50,
			wantPassed: false,
			wantMin:    0.0,
			wantMax:    2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, passed := scoreMetric(tt.value, target)
			if passed != tt.wantPassed {
				t.Errorf("passed = %v, want %v", passed, tt.wantPassed)
			}
			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("score = %v, want in [%v, %v]", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// --- EvaluateMetrics Tests ---

func TestEvaluateMetricsEarthlike(t *testing.T) {
	// Create metrics that match Earth targets
	metrics := TerrainMetrics{
		LandCoverage:       EarthLandCoverage,
		OceanCoverage:      EarthOceanCoverage,
		MountainCoverage:   EarthMountainCoverage,
		DeepOceanCoverage:  EarthDeepOceanCoverage,
		ShelfCoverage:      EarthShelfCoverage,
		MeanLandElevation:  EarthMeanLandElevation,
		MeanOceanDepth:     EarthMeanOceanDepth,
		GlobalMean:         EarthGlobalMean,
		GlobalStdDev:       EarthGlobalStdDev,
		MaxElevation:       EarthMaxElevation,
		MinElevation:       EarthMinElevation,
		HypsometricCurve:   make(map[float64]float64),
	}

	// Set hypsometric curve to Earth targets
	for elev, target := range HypsometricTargets {
		metrics.HypsometricCurve[elev] = target
	}

	result := EvaluateMetrics(metrics)

	// Should score very high (near 100%)
	if result.Score < 90 {
		t.Errorf("Earth-like metrics scored %.1f%%, want >= 90%%", result.Score)
	}

	// Should have no or few failed metrics
	if len(result.FailedMetrics) > 0 {
		t.Errorf("Earth-like metrics failed: %v", result.FailedMetrics)
	}

	// Should pass
	if !result.Passed {
		t.Error("Earth-like metrics should pass")
	}
}

func TestEvaluateMetricsPoorTerrain(t *testing.T) {
	// Create metrics that don't match Earth
	metrics := TerrainMetrics{
		LandCoverage:       0.50, // Way too much land
		OceanCoverage:      0.50,
		MountainCoverage:   0.10, // Way too many mountains
		DeepOceanCoverage:  0.00,
		ShelfCoverage:      0.00,
		MeanLandElevation:  2000, // Too high
		MeanOceanDepth:     -1000, // Too shallow
		GlobalMean:         0,
		GlobalStdDev:       1000,
		MaxElevation:       3000,
		MinElevation:       -2000,
		HypsometricCurve:   make(map[float64]float64),
	}

	// Set hypsometric curve to wrong values
	for elev := range HypsometricTargets {
		metrics.HypsometricCurve[elev] = 0.5 // Wrong for all thresholds
	}

	result := EvaluateMetrics(metrics)

	// Should score low
	if result.Score > 50 {
		t.Errorf("Poor metrics scored %.1f%%, want < 50%%", result.Score)
	}

	// Should have multiple failed metrics
	if len(result.FailedMetrics) < 3 {
		t.Errorf("Poor metrics should have many failures, got %d", len(result.FailedMetrics))
	}

	// Should not pass
	if result.Passed {
		t.Error("Poor metrics should not pass")
	}
}

// --- EvaluateTerrain Integration Test ---

func TestEvaluateTerrain(t *testing.T) {
	// Create Earth-like elevation distribution
	elevation := make([]float64, 1000)

	// 70% ocean
	for i := 0; i < 700; i++ {
		elevation[i] = -4000 + float64(i)*5 // -4000 to -500
	}
	// 28% land
	for i := 700; i < 980; i++ {
		elevation[i] = float64((i - 700) * 10) // 0 to 2800
	}
	// 2% mountains
	for i := 980; i < 1000; i++ {
		elevation[i] = 3000 + float64((i-980)*250) // 3000 to 8000
	}

	result := EvaluateTerrain(nil, elevation)

	// Should score reasonably well
	if result.Score < 40 {
		t.Errorf("Reasonable distribution scored %.1f%%, want >= 40%%", result.Score)
	}

	// Check that metrics were computed
	if result.Metrics.LandCoverage == 0 {
		t.Error("LandCoverage not computed")
	}
}

// --- QuickScore Tests ---

func TestQuickScore(t *testing.T) {
	elevation := make([]float64, 100)
	for i := range elevation {
		elevation[i] = float64(i*100 - 5000)
	}

	score := QuickScore(nil, elevation)

	// Should return a valid score
	if score < 0 || score > 100 {
		t.Errorf("QuickScore = %v, want in [0, 100]", score)
	}
}

// --- CompareToTargets Tests ---

func TestCompareToTargets(t *testing.T) {
	metrics := TerrainMetrics{
		LandCoverage:      EarthLandCoverage,
		OceanCoverage:     EarthOceanCoverage,
		MountainCoverage:  EarthMountainCoverage,
		DeepOceanCoverage: EarthDeepOceanCoverage,
		ShelfCoverage:     EarthShelfCoverage,
		MeanLandElevation: EarthMeanLandElevation,
		MeanOceanDepth:    EarthMeanOceanDepth,
	}

	comparisons := CompareToTargets(metrics)

	// Should have all primary metrics
	if len(comparisons) != len(PrimaryMetricTargets) {
		t.Errorf("got %d comparisons, want %d", len(comparisons), len(PrimaryMetricTargets))
	}

	// All should pass with Earth values
	for _, c := range comparisons {
		if !c.Passed {
			t.Errorf("metric %s should pass with Earth value", c.Name)
		}
		if math.Abs(c.Deviation) > 0.5 {
			t.Errorf("metric %s deviation = %v, want < 0.5", c.Name, c.Deviation)
		}
	}
}

// --- GetScoreBreakdown Tests ---

func TestGetScoreBreakdown(t *testing.T) {
	metrics := TerrainMetrics{
		LandCoverage:      EarthLandCoverage,
		OceanCoverage:     EarthOceanCoverage,
		MountainCoverage:  EarthMountainCoverage,
		DeepOceanCoverage: EarthDeepOceanCoverage,
		ShelfCoverage:     EarthShelfCoverage,
		MeanLandElevation: EarthMeanLandElevation,
		MeanOceanDepth:    EarthMeanOceanDepth,
		HypsometricCurve:  make(map[float64]float64),
	}

	for elev, target := range HypsometricTargets {
		metrics.HypsometricCurve[elev] = target
	}

	breakdown := GetScoreBreakdown(metrics)

	// Primary should have score
	if breakdown.PrimaryScore <= 0 {
		t.Error("PrimaryScore should be > 0")
	}
	if breakdown.PrimaryMax != 60 {
		t.Errorf("PrimaryMax = %v, want 60", breakdown.PrimaryMax)
	}

	// Hypsometric should have score
	if breakdown.HypsometricScore <= 0 {
		t.Error("HypsometricScore should be > 0")
	}
	if math.Abs(breakdown.HypsometricMax-25) > 0.01 {
		t.Errorf("HypsometricMax = %v, want ~25", breakdown.HypsometricMax)
	}

	// Total should be reasonable
	if breakdown.FinalPercent < 80 {
		t.Errorf("FinalPercent = %v, want >= 80", breakdown.FinalPercent)
	}
}

// --- Format Functions Tests ---

func TestFormatEvaluationResult(t *testing.T) {
	result := EvaluationResult{
		Score: 75.5,
		Metrics: TerrainMetrics{
			LandCoverage:      0.30,
			OceanCoverage:     0.70,
			MountainCoverage:  0.02,
			DeepOceanCoverage: 0.025,
			ShelfCoverage:     0.08,
			MeanLandElevation: 840,
			MeanOceanDepth:    -3688,
			GlobalMean:        -2430,
			GlobalStdDev:      2500,
			MaxElevation:      8848,
			MinElevation:      -10994,
			HypsometricCurve: map[float64]float64{
				-4000: 0.524,
				0:     0.708,
				1000:  0.902,
			},
		},
		FailedMetrics: []string{},
		Passed:        true,
	}

	output := FormatEvaluationResult(result)

	// Check key elements are present
	if !strings.Contains(output, "TERRAIN EVALUATION") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "75.5%") {
		t.Error("missing score")
	}
	if !strings.Contains(output, "PRIMARY METRICS") {
		t.Error("missing primary metrics section")
	}
	if !strings.Contains(output, "HYPSOMETRIC CURVE") {
		t.Error("missing hypsometric section")
	}
	if !strings.Contains(output, "ELEVATION STATISTICS") {
		t.Error("missing elevation stats section")
	}
	if !strings.Contains(output, "[PASS]") {
		t.Error("missing pass indicator")
	}
}

func TestFormatScoreBreakdown(t *testing.T) {
	breakdown := ScoreBreakdown{
		PrimaryScore:     55.0,
		PrimaryMax:       60.0,
		HypsometricScore: 22.0,
		HypsometricMax:   25.0,
		CoastlineScore:   8.0,
		CoastlineMax:     10.0,
		ContinentScore:   4.0,
		ContinentMax:     5.0,
		TotalScore:       89.0,
		TotalMax:         100.0,
		FinalPercent:     89.0,
	}

	output := FormatScoreBreakdown(breakdown)

	// Check key elements
	if !strings.Contains(output, "SCORE BREAKDOWN") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "Primary Metrics") {
		t.Error("missing primary metrics")
	}
	if !strings.Contains(output, "Hypsometric Curve") {
		t.Error("missing hypsometric")
	}
	if !strings.Contains(output, "TOTAL") {
		t.Error("missing total")
	}
	if !strings.Contains(output, "89.0%") {
		t.Error("missing final percent")
	}
}

// --- Edge Cases ---

func TestEvaluateEmptyElevation(t *testing.T) {
	result := EvaluateTerrain(nil, []float64{})

	// Should not panic and return a valid score
	if result.Score < 0 || result.Score > 100 {
		t.Errorf("empty elevation score = %v, want in [0, 100]", result.Score)
	}
	// Should not pass with empty data
	if result.Passed {
		t.Error("empty elevation should not pass")
	}
}

func TestScoreMetricZeroRange(t *testing.T) {
	target := MetricTarget{
		Name:   "Test",
		Min:    5.0,
		Max:    5.0, // Same as min
		Ideal:  5.0,
		Weight: 10.0,
	}

	// Should not panic
	score, passed := scoreMetric(5.0, target)
	if !passed {
		t.Error("exact match should pass")
	}
	if score != 10.0 {
		t.Errorf("exact match score = %v, want 10.0", score)
	}
}
