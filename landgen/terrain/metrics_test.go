package terrain

import (
	"math"
	"testing"
)

// tolerance for floating point comparisons
const epsilon = 0.0001

func floatEquals(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

// --- Coverage Tests ---

func TestLandOceanCoverage(t *testing.T) {
	tests := []struct {
		name          string
		elevation     []float64
		wantLand      float64
		wantOcean     float64
	}{
		{
			name:      "all land",
			elevation: []float64{100, 200, 300, 400, 500},
			wantLand:  1.0,
			wantOcean: 0.0,
		},
		{
			name:      "all ocean",
			elevation: []float64{-100, -200, -300, -400, -500},
			wantLand:  0.0,
			wantOcean: 1.0,
		},
		{
			name:      "mixed 30% land",
			elevation: []float64{100, 200, 300, -100, -200, -300, -400, -500, -600, -700},
			wantLand:  0.3,
			wantOcean: 0.7,
		},
		{
			name:      "sea level is ocean",
			elevation: []float64{0, 0, 0, 0, 100},
			wantLand:  0.2,
			wantOcean: 0.8,
		},
		{
			name:      "empty",
			elevation: []float64{},
			wantLand:  0.0,
			wantOcean: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLand, gotOcean := computeLandOceanCoverage(tt.elevation)
			if !floatEquals(gotLand, tt.wantLand) {
				t.Errorf("land coverage = %v, want %v", gotLand, tt.wantLand)
			}
			if !floatEquals(gotOcean, tt.wantOcean) {
				t.Errorf("ocean coverage = %v, want %v", gotOcean, tt.wantOcean)
			}
		})
	}
}

func TestMountainCoverage(t *testing.T) {
	tests := []struct {
		name      string
		elevation []float64
		want      float64
	}{
		{
			name:      "no mountains",
			elevation: []float64{100, 500, 1000, 2000, 2999},
			want:      0.0,
		},
		{
			name:      "all mountains",
			elevation: []float64{3001, 4000, 5000, 6000, 8000},
			want:      1.0,
		},
		{
			name:      "20% mountains",
			elevation: []float64{3001, 100, 200, 300, 400},
			want:      0.2,
		},
		{
			name:      "boundary at 3000m is not mountain",
			elevation: []float64{3000, 3001},
			want:      0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeMountainCoverage(tt.elevation)
			if !floatEquals(got, tt.want) {
				t.Errorf("mountain coverage = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeepOceanCoverage(t *testing.T) {
	tests := []struct {
		name      string
		elevation []float64
		want      float64
	}{
		{
			name:      "no deep ocean",
			elevation: []float64{-1000, -2000, -3000, -4000, -4999},
			want:      0.0,
		},
		{
			name:      "all deep ocean",
			elevation: []float64{-5001, -6000, -7000, -8000, -10000},
			want:      1.0,
		},
		{
			name:      "20% deep ocean",
			elevation: []float64{-5001, -1000, -2000, -3000, -4000},
			want:      0.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeDeepOceanCoverage(tt.elevation)
			if !floatEquals(got, tt.want) {
				t.Errorf("deep ocean coverage = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShelfCoverage(t *testing.T) {
	tests := []struct {
		name      string
		elevation []float64
		want      float64
	}{
		{
			name:      "all shelf",
			elevation: []float64{0, -50, -100, -150, -199},
			want:      1.0,
		},
		{
			name:      "no shelf",
			elevation: []float64{100, 200, -201, -300, -400},
			want:      0.0,
		},
		{
			name:      "40% shelf",
			elevation: []float64{-50, -100, 100, 200, -300},
			want:      0.4,
		},
		{
			name:      "boundary -200m is not shelf",
			elevation: []float64{-199, -200},
			want:      0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeShelfCoverage(tt.elevation)
			if !floatEquals(got, tt.want) {
				t.Errorf("shelf coverage = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Elevation Statistics Tests ---

func TestMeanLandElevation(t *testing.T) {
	tests := []struct {
		name      string
		elevation []float64
		want      float64
	}{
		{
			name:      "simple average",
			elevation: []float64{100, 200, 300, -100, -200},
			want:      200.0, // (100+200+300)/3
		},
		{
			name:      "no land",
			elevation: []float64{-100, -200, -300},
			want:      0.0,
		},
		{
			name:      "single land point",
			elevation: []float64{500, -100, -200},
			want:      500.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeMeanLandElevation(tt.elevation)
			if !floatEquals(got, tt.want) {
				t.Errorf("mean land elevation = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMeanOceanDepth(t *testing.T) {
	tests := []struct {
		name      string
		elevation []float64
		want      float64
	}{
		{
			name:      "simple average",
			elevation: []float64{100, 200, -100, -200, -300},
			want:      -200.0, // (-100-200-300)/3
		},
		{
			name:      "no ocean",
			elevation: []float64{100, 200, 300},
			want:      0.0,
		},
		{
			name:      "includes sea level",
			elevation: []float64{100, 0, -100},
			want:      -50.0, // (0-100)/2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeMeanOceanDepth(tt.elevation)
			if !floatEquals(got, tt.want) {
				t.Errorf("mean ocean depth = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobalStats(t *testing.T) {
	tests := []struct {
		name       string
		elevation  []float64
		wantMean   float64
		wantStdDev float64
	}{
		{
			name:       "simple values",
			elevation:  []float64{2, 4, 4, 4, 5, 5, 7, 9},
			wantMean:   5.0,
			wantStdDev: 2.0,
		},
		{
			name:       "all same",
			elevation:  []float64{100, 100, 100},
			wantMean:   100.0,
			wantStdDev: 0.0,
		},
		{
			name:       "negative values",
			elevation:  []float64{-100, -200, -300},
			wantMean:   -200.0,
			wantStdDev: 81.6496580927726, // sqrt(((100^2 + 0 + 100^2)/3))
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMean, gotStdDev := computeGlobalStats(tt.elevation)
			if !floatEquals(gotMean, tt.wantMean) {
				t.Errorf("mean = %v, want %v", gotMean, tt.wantMean)
			}
			if !floatEquals(gotStdDev, tt.wantStdDev) {
				t.Errorf("stdDev = %v, want %v", gotStdDev, tt.wantStdDev)
			}
		})
	}
}

func TestMinMax(t *testing.T) {
	tests := []struct {
		name      string
		elevation []float64
		wantMax   float64
		wantMin   float64
	}{
		{
			name:      "simple range",
			elevation: []float64{100, 200, 300, -100, -200},
			wantMax:   300,
			wantMin:   -200,
		},
		{
			name:      "single value",
			elevation: []float64{42},
			wantMax:   42,
			wantMin:   42,
		},
		{
			name:      "empty",
			elevation: []float64{},
			wantMax:   0,
			wantMin:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMax, gotMin := computeMinMax(tt.elevation)
			if gotMax != tt.wantMax {
				t.Errorf("max = %v, want %v", gotMax, tt.wantMax)
			}
			if gotMin != tt.wantMin {
				t.Errorf("min = %v, want %v", gotMin, tt.wantMin)
			}
		})
	}
}

// --- Hypsometric Curve Tests ---

func TestHypsometricCurve(t *testing.T) {
	// Create elevation data: 70% below 0, 30% above
	elevation := make([]float64, 100)
	for i := 0; i < 70; i++ {
		elevation[i] = float64(-4000 + i*50) // -4000 to -500
	}
	for i := 70; i < 100; i++ {
		elevation[i] = float64((i - 70) * 100) // 0 to 2900
	}

	curve := computeHypsometricCurve(elevation)

	// Check sea level threshold
	if curve[0] < 0.65 || curve[0] > 0.75 {
		t.Errorf("cumulative at 0m = %v, want ~0.70", curve[0])
	}
}

// --- ComputeMetrics Integration Test ---

func TestComputeMetrics(t *testing.T) {
	// Create Earth-like distribution
	elevation := make([]float64, 1000)
	for i := 0; i < 700; i++ {
		elevation[i] = float64(-4000 + i*5) // Ocean: -4000 to -500
	}
	for i := 700; i < 980; i++ {
		elevation[i] = float64((i - 700) * 10) // Land: 0 to 2800
	}
	for i := 980; i < 1000; i++ {
		elevation[i] = float64(3000 + (i-980)*250) // Mountains: 3000 to 8000
	}

	metrics := ComputeMetrics(nil, elevation)

	// Check land coverage is approximately 30%
	if metrics.LandCoverage < 0.25 || metrics.LandCoverage > 0.35 {
		t.Errorf("land coverage = %v, want ~0.30", metrics.LandCoverage)
	}

	// Check mountain coverage is approximately 2%
	if metrics.MountainCoverage < 0.01 || metrics.MountainCoverage > 0.03 {
		t.Errorf("mountain coverage = %v, want ~0.02", metrics.MountainCoverage)
	}

	// Check mean land elevation is positive
	if metrics.MeanLandElevation <= 0 {
		t.Errorf("mean land elevation = %v, want positive", metrics.MeanLandElevation)
	}

	// Check mean ocean depth is negative
	if metrics.MeanOceanDepth >= 0 {
		t.Errorf("mean ocean depth = %v, want negative", metrics.MeanOceanDepth)
	}

	// Check min/max are reasonable
	if metrics.MaxElevation < 5000 {
		t.Errorf("max elevation = %v, want > 5000", metrics.MaxElevation)
	}
	if metrics.MinElevation > -3000 {
		t.Errorf("min elevation = %v, want < -3000", metrics.MinElevation)
	}
}

// --- Terrain Classification Tests ---

func TestClassifyTerrain(t *testing.T) {
	tests := []struct {
		elevation float64
		want      TerrainType
	}{
		{-6000, TerrainDeepOcean},
		{-5000, TerrainOcean},
		{-4000, TerrainOcean},
		{-200, TerrainShelf},
		{-100, TerrainShelf},
		{0, TerrainCoast},
		{100, TerrainCoast},
		{500, TerrainLowland},
		{1500, TerrainHighland},
		{4000, TerrainMountain},
	}

	for _, tt := range tests {
		t.Run(tt.want.String(), func(t *testing.T) {
			got := ClassifyTerrain(tt.elevation)
			if got != tt.want {
				t.Errorf("ClassifyTerrain(%v) = %v, want %v", tt.elevation, got, tt.want)
			}
		})
	}
}

// --- Histogram Tests ---

func TestElevationHistogram(t *testing.T) {
	elevation := []float64{0, 100, 200, 300, 400, 500, 600, 700, 800, 900}
	hist := ComputeElevationHistogram(elevation, 5)

	if hist.Total != 10 {
		t.Errorf("total = %v, want 10", hist.Total)
	}

	// Each bin should have 2 values
	for i, c := range hist.Counts {
		if c != 2 {
			t.Errorf("bin %d count = %v, want 2", i, c)
		}
	}

	// Check frequencies sum to 1
	freq := hist.Frequencies()
	sum := 0.0
	for _, f := range freq {
		sum += f
	}
	if !floatEquals(sum, 1.0) {
		t.Errorf("frequencies sum = %v, want 1.0", sum)
	}
}

// --- Percentile Tests ---

func TestPercentiles(t *testing.T) {
	// Create sorted data 0-99
	elevation := make([]float64, 100)
	for i := range elevation {
		elevation[i] = float64(i)
	}

	percentiles := ComputePercentiles(elevation, []float64{0, 25, 50, 75, 100})

	if percentiles[0] != 0 {
		t.Errorf("p0 = %v, want 0", percentiles[0])
	}
	if percentiles[50] < 49 || percentiles[50] > 50 {
		t.Errorf("p50 = %v, want ~49.5", percentiles[50])
	}
	if percentiles[100] != 99 {
		t.Errorf("p100 = %v, want 99", percentiles[100])
	}
}

// --- Summary Tests ---

func TestElevationSummary(t *testing.T) {
	elevation := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	summary := ComputeElevationSummary(elevation)

	if summary.Count != 10 {
		t.Errorf("count = %v, want 10", summary.Count)
	}
	if summary.Min != 1 {
		t.Errorf("min = %v, want 1", summary.Min)
	}
	if summary.Max != 10 {
		t.Errorf("max = %v, want 10", summary.Max)
	}
	if !floatEquals(summary.Mean, 5.5) {
		t.Errorf("mean = %v, want 5.5", summary.Mean)
	}
}
