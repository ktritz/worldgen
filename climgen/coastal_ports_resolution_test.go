package climgen

import (
	"math"
	"math/rand"
	"testing"
)

// coastalChainScoringFixture builds `chains` independent coastal chains, each a
// straight run of `chainLen` coastal land cells sharing one ocean neighbour, with a
// settlement node anchored at the head of every chain. Chain length is chosen by the
// caller to hold the *physical* catchment extent fixed across mesh levels, so the only
// thing that changes with resolution is how densely the same physical coastline is
// sampled. Suitability is drawn i.i.d. per cell from the same distribution at every
// resolution, i.e. a high-frequency spatial field whose statistics do not depend on the
// mesh.
func coastalChainScoringFixture(cellCount, chainLen, chains int, seed int64) ([]VoronoiCell, []float64, *SettlementNetworkResult, *CoastalPortDiagnostics) {
	cells := make([]VoronoiCell, cellCount)
	elevation := make([]float64, cellCount)
	for i := range cells {
		cells[i].SiteIndex = int32(i)
		elevation[i] = -1000
	}
	rng := rand.New(rand.NewSource(seed))
	nodes := make([]SettlementNode, 0, chains)
	suitability := make([]float64, cellCount)
	stride := chainLen + 1
	for c := 0; c < chains; c++ {
		base := c * stride
		if base+stride > cellCount {
			break
		}
		oceanCell := base + chainLen
		for k := 0; k < chainLen; k++ {
			idx := base + k
			elevation[idx] = 10
			suitability[idx] = rng.Float64()
			neighbors := []int32{int32(oceanCell)}
			if k > 0 {
				neighbors = append(neighbors, int32(idx-1))
			}
			if k < chainLen-1 {
				neighbors = append(neighbors, int32(idx+1))
			}
			cells[idx].NeighborSiteIndices = neighbors
		}
		nodes = append(nodes, SettlementNode{ID: c, CellIndex: base, Kind: SettlementNodeVillage, Score: 0.5, Coastal: true})
	}
	network := &SettlementNetworkResult{Nodes: nodes}
	diag := &CoastalPortDiagnostics{
		PortSuitability:       suitability,
		DeepwaterSuitability:  make([]float64, cellCount),
		NodePortScore:         make([]float64, len(nodes)),
		NodeBasePortScore:     make([]float64, len(nodes)),
		NodeDeepwaterScore:    make([]float64, len(nodes)),
		NodeDeepwaterTermCell: make([]int, len(nodes)),
		NodeTerminalCell:      make([]int, len(nodes)),
	}
	for i := range diag.NodeTerminalCell {
		diag.NodeTerminalCell[i] = -1
		diag.NodeDeepwaterTermCell[i] = -1
	}
	return cells, elevation, network, diag
}

func coastalChainScoringSettings(baseHops int) MaritimePortSettings {
	settings := DefaultMaritimePortSettings()
	settings.NodeCatchmentHops = baseHops
	settings.NodeCatchmentDecay = 1
	settings.NodeFeatureWeight = 0
	return settings
}

func meanCoastalNodePortScore(t *testing.T, cellCount, chainLen, chains, baseHops int) float64 {
	t.Helper()
	cells, elevation, network, diag := coastalChainScoringFixture(cellCount, chainLen, chains, 20260802)
	if hops := meshResolutionAdjustedSteps(baseHops, cellCount); hops < chainLen-1 {
		t.Fatalf("fixture misconfigured: scaled catchment %d hops cannot span a %d-cell chain", hops, chainLen)
	}
	populateBaseCoastalNodeScores(cells, network, elevation, 0, diag, coastalChainScoringSettings(baseHops))
	total := 0.0
	for _, score := range diag.NodePortScore {
		total += score
	}
	return total / float64(len(diag.NodePortScore))
}

// The catchment disc holds ~2x as many coastal cells per mesh level, and a maximum over
// more samples is systematically larger. Because the resulting node score is compared
// against absolute port thresholds, that inflation alone admits more ports at fine
// meshes. The scored statistic must therefore depend on the physical catchment, not on
// how finely it happens to be sampled.
func TestPopulateBaseCoastalNodeScoresStatisticIsMeshResolutionStable(t *testing.T) {
	const chains = 400
	baseline := meanCoastalNodePortScore(t, 10242, 7, chains, 6)

	cases := []struct {
		name      string
		cellCount int
		chainLen  int
	}{
		{name: "level6", cellCount: 40962, chainLen: 13},
		{name: "level7", cellCount: 163842, chainLen: 25},
	}
	for _, tc := range cases {
		got := meanCoastalNodePortScore(t, tc.cellCount, tc.chainLen, chains, 6)
		ratio := got / baseline
		if math.Abs(ratio-1) > 0.05 {
			t.Errorf("%s mean node port score %.4f vs level5 baseline %.4f (ratio %.3f); catchment statistic is not mesh-resolution stable",
				tc.name, got, baseline, ratio)
		}
	}
}

// The level-5 mesh is the tuning baseline: the correction must not move it at all.
func TestPopulateBaseCoastalNodeScoresIsExactAtBaselineResolution(t *testing.T) {
	const (
		cellCount = 10242
		chainLen  = 7
		chains    = 64
	)
	cells, elevation, network, diag := coastalChainScoringFixture(cellCount, chainLen, chains, 7)
	populateBaseCoastalNodeScores(cells, network, elevation, 0, diag, coastalChainScoringSettings(6))
	for i := range network.Nodes {
		base := i * (chainLen + 1)
		want := 0.0
		for k := 0; k < chainLen; k++ {
			want = math.Max(want, diag.PortSuitability[base+k])
		}
		if diag.NodePortScore[i] != want {
			t.Fatalf("node %d: baseline-mesh score %.17g, want the catchment maximum %.17g", i, diag.NodePortScore[i], want)
		}
	}
}

func TestMeshScaleStableMaxOfLinearSamplesReducesToMaxAtBaseline(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for n := 1; n <= 40; n++ {
		samples := make([]float64, n)
		want := 0.0
		for i := range samples {
			samples[i] = rng.Float64()
			want = math.Max(want, samples[i])
		}
		if got := meshScaleStableMaxOfLinearSamples(samples, 10242); got != want {
			t.Fatalf("n=%d: got %.17g, want max %.17g", n, got, want)
		}
	}
}
