package terrain

import (
	"math"
	"testing"

	"worldgen/icosphere"
)

func TestComputeMetricsWithCellsPopulatesSpatialMetrics(t *testing.T) {
	sites, faces := icosphere.CreateIcosphere(2)
	_, cells := icosphere.GenerateSphericalVoronoi(sites, faces)

	elevation := make([]float64, len(sites))
	for i, site := range sites {
		switch {
		case site.Z > 0.35:
			elevation[i] = 900 + 2400*math.Max(site.X, 0)
		case site.Z < -0.45 && site.X > -0.2:
			elevation[i] = 700 + 2200*math.Max(site.Y, 0)
		default:
			elevation[i] = -3600 - 1200*math.Abs(site.Z)
		}
	}

	// Create a small mountain cluster so clustering metrics are exercised.
	for i, site := range sites {
		if site.Z > 0.55 && site.X > 0.45 {
			elevation[i] = 4200
		}
	}

	metrics := ComputeMetricsWithCells(sites, cells, elevation)

	if metrics.NumMajorLandmasses < 2 {
		t.Fatalf("NumMajorLandmasses = %d, want at least 2", metrics.NumMajorLandmasses)
	}
	if metrics.LargestContinentPct <= 0 || metrics.LargestContinentPct >= 1 {
		t.Fatalf("LargestContinentPct = %v, want in (0, 1)", metrics.LargestContinentPct)
	}
	if metrics.TortuosityRatio <= 0 {
		t.Fatalf("TortuosityRatio = %v, want > 0", metrics.TortuosityRatio)
	}
	if metrics.FractalDimension <= 0 {
		t.Fatalf("FractalDimension = %v, want > 0", metrics.FractalDimension)
	}
	if metrics.MeanLocalRelief <= 0 {
		t.Fatalf("MeanLocalRelief = %v, want > 0", metrics.MeanLocalRelief)
	}
	if metrics.P95LocalRelief < metrics.MeanLocalRelief {
		t.Fatalf("P95LocalRelief = %v, want >= mean local relief %v", metrics.P95LocalRelief, metrics.MeanLocalRelief)
	}
	if metrics.MountainClustered <= 0 {
		t.Fatalf("MountainClustered = %v, want > 0", metrics.MountainClustered)
	}
}

func TestEvaluateTerrainWithCellsUsesContinentMetrics(t *testing.T) {
	sites, faces := icosphere.CreateIcosphere(1)
	_, cells := icosphere.GenerateSphericalVoronoi(sites, faces)

	elevation := make([]float64, len(sites))
	for i, site := range sites {
		if site.Z > 0.25 || (site.Z < -0.35 && site.X > 0) {
			elevation[i] = 1000 + 1500*math.Max(site.Y, 0)
			continue
		}
		elevation[i] = -4200
	}

	result := EvaluateTerrainWithCells(sites, cells, elevation)
	if result.Metrics.NumMajorLandmasses == 0 {
		t.Fatal("expected continent metrics to be populated")
	}

	breakdown := GetScoreBreakdown(result.Metrics)
	if breakdown.ContinentMax <= 0 {
		t.Fatalf("ContinentMax = %v, want > 0", breakdown.ContinentMax)
	}
}
