package terrain

import (
	"math"
	"sort"
)

// --- Coverage Calculations ---

// ComputeMetrics calculates all terrain metrics from elevation data.
// This is the primary entry point for terrain evaluation.
func ComputeMetrics(sites []Vector3D, elevation []float64) TerrainMetrics {
	if len(elevation) == 0 {
		return TerrainMetrics{}
	}

	metrics := TerrainMetrics{
		HypsometricCurve: make(map[float64]float64),
	}

	// Compute coverage metrics
	metrics.LandCoverage, metrics.OceanCoverage = computeLandOceanCoverage(elevation)
	metrics.MountainCoverage = computeMountainCoverage(elevation)
	metrics.DeepOceanCoverage = computeDeepOceanCoverage(elevation)
	metrics.ShelfCoverage = computeShelfCoverage(elevation)

	// Compute elevation statistics
	metrics.MeanLandElevation = computeMeanLandElevation(elevation)
	metrics.MeanOceanDepth = computeMeanOceanDepth(elevation)
	metrics.GlobalMean, metrics.GlobalStdDev = computeGlobalStats(elevation)
	metrics.MaxElevation, metrics.MinElevation = computeMinMax(elevation)

	// Compute hypsometric curve
	metrics.HypsometricCurve = computeHypsometricCurve(elevation)

	return metrics
}

// computeLandOceanCoverage returns the fraction of sites above and at/below sea level.
func computeLandOceanCoverage(elevation []float64) (landCoverage, oceanCoverage float64) {
	if len(elevation) == 0 {
		return 0, 0
	}

	landCount := 0
	for _, e := range elevation {
		if IsLand(e) {
			landCount++
		}
	}

	landCoverage = float64(landCount) / float64(len(elevation))
	oceanCoverage = 1.0 - landCoverage
	return
}

// computeMountainCoverage returns the fraction of sites above 3000m.
func computeMountainCoverage(elevation []float64) float64 {
	if len(elevation) == 0 {
		return 0
	}

	count := 0
	for _, e := range elevation {
		if IsMountain(e) {
			count++
		}
	}

	return float64(count) / float64(len(elevation))
}

// computeDeepOceanCoverage returns the fraction of sites below -5000m.
func computeDeepOceanCoverage(elevation []float64) float64 {
	if len(elevation) == 0 {
		return 0
	}

	count := 0
	for _, e := range elevation {
		if IsDeepOcean(e) {
			count++
		}
	}

	return float64(count) / float64(len(elevation))
}

// computeShelfCoverage returns the fraction of sites in the continental shelf range (0 to -200m).
func computeShelfCoverage(elevation []float64) float64 {
	if len(elevation) == 0 {
		return 0
	}

	count := 0
	for _, e := range elevation {
		if IsShelf(e) {
			count++
		}
	}

	return float64(count) / float64(len(elevation))
}

// --- Elevation Statistics ---

// computeMeanLandElevation returns the mean elevation of land sites (above sea level).
func computeMeanLandElevation(elevation []float64) float64 {
	sum := 0.0
	count := 0

	for _, e := range elevation {
		if IsLand(e) {
			sum += e
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// computeMeanOceanDepth returns the mean depth of ocean sites (at or below sea level).
func computeMeanOceanDepth(elevation []float64) float64 {
	sum := 0.0
	count := 0

	for _, e := range elevation {
		if IsOcean(e) {
			sum += e
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// computeGlobalStats returns the mean and standard deviation of all elevations.
func computeGlobalStats(elevation []float64) (mean, stdDev float64) {
	if len(elevation) == 0 {
		return 0, 0
	}

	// Compute mean
	sum := 0.0
	for _, e := range elevation {
		sum += e
	}
	mean = sum / float64(len(elevation))

	// Compute standard deviation
	sumSquares := 0.0
	for _, e := range elevation {
		diff := e - mean
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(elevation))
	stdDev = math.Sqrt(variance)

	return
}

// computeMinMax returns the maximum and minimum elevations.
func computeMinMax(elevation []float64) (max, min float64) {
	if len(elevation) == 0 {
		return 0, 0
	}

	max = elevation[0]
	min = elevation[0]

	for _, e := range elevation {
		if e > max {
			max = e
		}
		if e < min {
			min = e
		}
	}

	return
}

// --- Hypsometric Curve ---

// computeHypsometricCurve returns the cumulative fraction of sites below each threshold elevation.
// Uses the standard thresholds from HypsometricTargets.
func computeHypsometricCurve(elevation []float64) map[float64]float64 {
	if len(elevation) == 0 {
		return make(map[float64]float64)
	}

	// Get sorted thresholds
	thresholds := make([]float64, 0, len(HypsometricTargets))
	for t := range HypsometricTargets {
		thresholds = append(thresholds, t)
	}
	sort.Float64s(thresholds)

	// Count sites below each threshold
	curve := make(map[float64]float64)
	n := float64(len(elevation))

	for _, threshold := range thresholds {
		count := 0
		for _, e := range elevation {
			if e < threshold {
				count++
			}
		}
		curve[threshold] = float64(count) / n
	}

	return curve
}

// ComputeHypsometricCurveDetailed returns the cumulative distribution at many elevation points.
// Useful for plotting the full hypsometric curve.
func ComputeHypsometricCurveDetailed(elevation []float64, numPoints int) [][2]float64 {
	if len(elevation) == 0 || numPoints < 2 {
		return nil
	}

	// Find min and max
	maxElev, minElev := computeMinMax(elevation)

	// Create sorted copy for efficient cumulative calculation
	sorted := make([]float64, len(elevation))
	copy(sorted, elevation)
	sort.Float64s(sorted)

	// Generate curve points
	curve := make([][2]float64, numPoints)
	n := float64(len(sorted))

	for i := 0; i < numPoints; i++ {
		// Elevation threshold
		t := float64(i) / float64(numPoints-1)
		threshold := minElev + t*(maxElev-minElev)

		// Binary search for count below threshold
		idx := sort.SearchFloat64s(sorted, threshold)
		cumulative := float64(idx) / n

		curve[i] = [2]float64{threshold, cumulative}
	}

	return curve
}

// --- Terrain Classification Counts ---

// TerrainCounts holds the count of sites in each terrain category.
type TerrainCounts struct {
	DeepOcean int
	Ocean     int
	Shelf     int
	Coast     int
	Lowland   int
	Highland  int
	Mountain  int
	Total     int
}

// ComputeTerrainCounts returns the number of sites in each terrain category.
func ComputeTerrainCounts(elevation []float64) TerrainCounts {
	counts := TerrainCounts{Total: len(elevation)}

	for _, e := range elevation {
		switch ClassifyTerrain(e) {
		case TerrainDeepOcean:
			counts.DeepOcean++
		case TerrainOcean:
			counts.Ocean++
		case TerrainShelf:
			counts.Shelf++
		case TerrainCoast:
			counts.Coast++
		case TerrainLowland:
			counts.Lowland++
		case TerrainHighland:
			counts.Highland++
		case TerrainMountain:
			counts.Mountain++
		}
	}

	return counts
}

// Fractions returns the terrain counts as fractions of total.
func (c TerrainCounts) Fractions() map[TerrainType]float64 {
	if c.Total == 0 {
		return nil
	}
	n := float64(c.Total)
	return map[TerrainType]float64{
		TerrainDeepOcean: float64(c.DeepOcean) / n,
		TerrainOcean:     float64(c.Ocean) / n,
		TerrainShelf:     float64(c.Shelf) / n,
		TerrainCoast:     float64(c.Coast) / n,
		TerrainLowland:   float64(c.Lowland) / n,
		TerrainHighland:  float64(c.Highland) / n,
		TerrainMountain:  float64(c.Mountain) / n,
	}
}

// --- Elevation Histograms ---

// ElevationHistogram represents a binned elevation distribution.
type ElevationHistogram struct {
	BinMin   float64   // Minimum elevation of first bin
	BinMax   float64   // Maximum elevation of last bin
	BinWidth float64   // Width of each bin
	Counts   []int     // Count per bin
	Total    int       // Total count
}

// ComputeElevationHistogram creates a histogram of elevations.
func ComputeElevationHistogram(elevation []float64, numBins int) ElevationHistogram {
	if len(elevation) == 0 || numBins < 1 {
		return ElevationHistogram{}
	}

	maxElev, minElev := computeMinMax(elevation)

	// Add small padding to ensure max value falls in a bin
	binWidth := (maxElev - minElev) / float64(numBins)
	if binWidth == 0 {
		binWidth = 1
	}

	hist := ElevationHistogram{
		BinMin:   minElev,
		BinMax:   maxElev,
		BinWidth: binWidth,
		Counts:   make([]int, numBins),
		Total:    len(elevation),
	}

	for _, e := range elevation {
		bin := int((e - minElev) / binWidth)
		if bin >= numBins {
			bin = numBins - 1
		}
		if bin < 0 {
			bin = 0
		}
		hist.Counts[bin]++
	}

	return hist
}

// Frequencies returns the histogram as normalized frequencies.
func (h ElevationHistogram) Frequencies() []float64 {
	if h.Total == 0 {
		return nil
	}
	freq := make([]float64, len(h.Counts))
	for i, c := range h.Counts {
		freq[i] = float64(c) / float64(h.Total)
	}
	return freq
}

// BinCenters returns the center elevation of each bin.
func (h ElevationHistogram) BinCenters() []float64 {
	centers := make([]float64, len(h.Counts))
	for i := range centers {
		centers[i] = h.BinMin + (float64(i)+0.5)*h.BinWidth
	}
	return centers
}

// --- Bimodality Detection ---

// BimodalityMetrics contains statistics about the bimodal distribution of elevations.
type BimodalityMetrics struct {
	PeakOcean       float64 // Elevation of ocean peak
	PeakLand        float64 // Elevation of land peak
	ValleyElevation float64 // Elevation of valley between peaks
	BimodalityIndex float64 // Measure of separation (0 = unimodal, 1 = strongly bimodal)
}

// ComputeBimodalityMetrics analyzes the elevation distribution for bimodality.
// Earth has a characteristic bimodal distribution with peaks around -4200m and +300m.
func ComputeBimodalityMetrics(elevation []float64) BimodalityMetrics {
	if len(elevation) == 0 {
		return BimodalityMetrics{}
	}

	// Create histogram
	hist := ComputeElevationHistogram(elevation, 100)
	freq := hist.Frequencies()
	centers := hist.BinCenters()

	if len(freq) < 3 {
		return BimodalityMetrics{}
	}

	// Find peaks (local maxima)
	peaks := findPeaks(freq, centers)
	if len(peaks) < 2 {
		// Not bimodal
		return BimodalityMetrics{BimodalityIndex: 0}
	}

	// Sort peaks by frequency (descending)
	sort.Slice(peaks, func(i, j int) bool {
		return peaks[i].freq > peaks[j].freq
	})

	// Take top two peaks
	peak1 := peaks[0]
	peak2 := peaks[1]

	// Ensure peak1 is the lower elevation (ocean)
	if peak1.elev > peak2.elev {
		peak1, peak2 = peak2, peak1
	}

	// Find minimum between peaks (valley)
	valley := findValley(freq, centers, peak1.elev, peak2.elev)

	// Compute bimodality index
	// Based on the depth of the valley relative to the peaks
	avgPeakHeight := (peak1.freq + peak2.freq) / 2
	bimodality := 0.0
	if avgPeakHeight > 0 {
		bimodality = 1.0 - (valley.freq / avgPeakHeight)
	}
	bimodality = Clamp(bimodality, 0, 1)

	return BimodalityMetrics{
		PeakOcean:       peak1.elev,
		PeakLand:        peak2.elev,
		ValleyElevation: valley.elev,
		BimodalityIndex: bimodality,
	}
}

// peak represents a peak in the histogram.
type peak struct {
	elev float64
	freq float64
}

// findPeaks finds local maxima in the frequency distribution.
func findPeaks(freq []float64, centers []float64) []peak {
	var peaks []peak

	for i := 1; i < len(freq)-1; i++ {
		if freq[i] > freq[i-1] && freq[i] > freq[i+1] {
			peaks = append(peaks, peak{elev: centers[i], freq: freq[i]})
		}
	}

	return peaks
}

// findValley finds the minimum between two elevations.
func findValley(freq []float64, centers []float64, lowElev, highElev float64) peak {
	minFreq := math.MaxFloat64
	minElev := 0.0

	for i, c := range centers {
		if c >= lowElev && c <= highElev {
			if freq[i] < minFreq {
				minFreq = freq[i]
				minElev = c
			}
		}
	}

	return peak{elev: minElev, freq: minFreq}
}

// --- Elevation Percentiles ---

// ComputePercentiles returns elevation values at specified percentiles.
// Percentiles should be in range [0, 100].
func ComputePercentiles(elevation []float64, percentiles []float64) map[float64]float64 {
	if len(elevation) == 0 {
		return nil
	}

	// Sort elevations
	sorted := make([]float64, len(elevation))
	copy(sorted, elevation)
	sort.Float64s(sorted)

	result := make(map[float64]float64)
	n := len(sorted)

	for _, p := range percentiles {
		if p < 0 || p > 100 {
			continue
		}
		idx := int(p / 100.0 * float64(n-1))
		if idx >= n {
			idx = n - 1
		}
		result[p] = sorted[idx]
	}

	return result
}

// --- Summary Statistics ---

// ElevationSummary provides a quick statistical summary.
type ElevationSummary struct {
	Count   int
	Min     float64
	Max     float64
	Mean    float64
	StdDev  float64
	Median  float64
	P25     float64 // 25th percentile
	P75     float64 // 75th percentile
}

// ComputeElevationSummary returns summary statistics for elevations.
func ComputeElevationSummary(elevation []float64) ElevationSummary {
	if len(elevation) == 0 {
		return ElevationSummary{}
	}

	mean, stdDev := computeGlobalStats(elevation)
	max, min := computeMinMax(elevation)
	percentiles := ComputePercentiles(elevation, []float64{25, 50, 75})

	return ElevationSummary{
		Count:  len(elevation),
		Min:    min,
		Max:    max,
		Mean:   mean,
		StdDev: stdDev,
		Median: percentiles[50],
		P25:    percentiles[25],
		P75:    percentiles[75],
	}
}
