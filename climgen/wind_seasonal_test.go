package climgen

import (
	"math"
	"testing"
)

func TestGetCirculationZoneUsesThermalEquatorShift(t *testing.T) {
	settings := DefaultCirculationSettings()
	settings.ThermalEquatorShiftDeg = 10.0

	lat35North := 35.0 * (3.141592653589793 / 180.0)
	if zone := getCirculationZone(lat35North, settings); zone != ZoneHadley {
		t.Fatalf("expected shifted 35N cell to fall into Hadley zone, got %v", zone)
	}
}

func TestSeasonalTropicalPressureAnomalyAddsSummerTropicalLandLow(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(4)
	annualMean := make([]float64, len(vertices))
	seasonalTemp := make([]float64, len(vertices))
	for i := range annualMean {
		annualMean[i] = FreezingPoint + 26
		seasonalTemp[i] = annualMean[i]
	}

	tropicalLand := 0
	for i, v := range vertices {
		latDeg := getLatitudeDeg(v)
		lonDeg := math.Atan2(v.Z, v.X) * 180.0 / math.Pi
		if latDeg >= 4 && latDeg <= 28 && lonDeg >= -95 && lonDeg <= 95 {
			elevation[i] = 800
			seasonalTemp[i] += 10
			tropicalLand++
		}
	}
	if tropicalLand == 0 {
		t.Fatalf("expected test world to include tropical land samples")
	}

	solar := SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5}

	base := computeSeasonalTropicalPressureAnomaly(
		vertices,
		elevation,
		0.0,
		adj,
		solar,
		annualMean,
		annualMean,
	)
	heated := computeSeasonalTropicalPressureAnomaly(
		vertices,
		elevation,
		0.0,
		adj,
		solar,
		seasonalTemp,
		annualMean,
	)
	baseMean, heatedMean := 0.0, 0.0
	count := 0
	for i, v := range vertices {
		latDeg := getLatitudeDeg(v)
		lonDeg := math.Atan2(v.Z, v.X) * 180.0 / math.Pi
		if latDeg < 4 || latDeg > 28 || lonDeg < -95 || lonDeg > 95 || elevation[i] < 0 {
			continue
		}
		baseMean += base[i]
		heatedMean += heated[i]
		count++
	}
	if count == 0 {
		t.Fatalf("no heated tropical land cells found in pressure comparison")
	}
	baseMean /= float64(count)
	heatedMean /= float64(count)
	if heatedMean >= baseMean-0.02 {
		t.Fatalf("expected heated summer tropical land pressure to drop: base=%.4f heated=%.4f", baseMean, heatedMean)
	}
}

func TestSeasonalTropicalConvergenceLatitudeShiftsTowardHeatedSummerLand(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(4)
	annualMean := make([]float64, len(vertices))
	seasonalTemp := make([]float64, len(vertices))
	for i := range annualMean {
		annualMean[i] = FreezingPoint + 26
		seasonalTemp[i] = annualMean[i]
	}

	targetIdx := -1
	for i, v := range vertices {
		latDeg := getLatitudeDeg(v)
		lonDeg := math.Atan2(v.Z, v.X) * 180.0 / math.Pi
		if latDeg >= 6 && latDeg <= 24 && lonDeg >= -95 && lonDeg <= 95 {
			elevation[i] = 800
			seasonalTemp[i] += 10
			if targetIdx < 0 && math.Abs(latDeg-14) < 4 && math.Abs(lonDeg) < 30 {
				targetIdx = i
			}
		}
	}
	if targetIdx < 0 {
		t.Fatalf("failed to choose a heated tropical land sample")
	}

	solar := SolarSettings{AxialTilt: 23.5, SeasonPhase: 0.5}
	base := computeSeasonalTropicalConvergenceLatitudeField(
		vertices,
		elevation,
		0.0,
		adj,
		solar,
		annualMean,
		annualMean,
	)
	heated := computeSeasonalTropicalConvergenceLatitudeField(
		vertices,
		elevation,
		0.0,
		adj,
		solar,
		seasonalTemp,
		annualMean,
	)
	if heated[targetIdx] <= base[targetIdx]+0.5 {
		t.Fatalf("expected local tropical convergence latitude to shift poleward over heated summer land: base=%.2f heated=%.2f", base[targetIdx], heated[targetIdx])
	}
}
