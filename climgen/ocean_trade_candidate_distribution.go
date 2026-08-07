package climgen

import "sort"

// oceanCandidateScoreThresholds are the composite NodeDeepwaterScore cut points
// reported by the candidate-port distribution diagnostic.
var oceanCandidateScoreThresholds = []float64{0.30, 0.34, 0.38, 0.42, 0.44, 0.46, 0.48, 0.52, 0.56, 0.60}

// oceanCandidatePhysicalThresholds are the physical (base) deepwater score cut
// points reported by the candidate-port distribution diagnostic.
var oceanCandidatePhysicalThresholds = []float64{0.06, 0.12, 0.18, 0.24, 0.30, 0.36}

// OceanCandidateScoreDistribution summarizes the deepwater score field over
// every node that clears the structural prerequisites for an ocean candidate
// port (deepwater terminal, settlement support, village-or-larger unless it is
// already a major deepwater port). It exists so candidate thresholds can be
// chosen from the measured field rather than from the surviving candidates,
// which are by construction all above the current threshold.
type OceanCandidateScoreDistribution struct {
	EligibleNodes     int
	CivilizedNodes    int
	ScoreP50          float64
	ScoreP75          float64
	ScoreP90          float64
	ScoreP95          float64
	ScoreMax          float64
	PhysicalP50       float64
	PhysicalP75       float64
	PhysicalP90       float64
	PhysicalMax       float64
	ScoreThresholds   []float64
	ScoreCounts       []int
	ScoreCivilized    []int
	ScorePassBoth     []int
	PhysicalThreshold []float64
	PhysicalCounts    []int
}

func oceanCandidateScoreDistribution(
	network *SettlementNetworkResult,
	ports *CoastalPortResult,
	civByNode []int,
	physicalFloor float64,
) OceanCandidateScoreDistribution {
	dist := OceanCandidateScoreDistribution{
		ScoreThresholds:   append([]float64(nil), oceanCandidateScoreThresholds...),
		ScoreCounts:       make([]int, len(oceanCandidateScoreThresholds)),
		ScoreCivilized:    make([]int, len(oceanCandidateScoreThresholds)),
		ScorePassBoth:     make([]int, len(oceanCandidateScoreThresholds)),
		PhysicalThreshold: append([]float64(nil), oceanCandidatePhysicalThresholds...),
		PhysicalCounts:    make([]int, len(oceanCandidatePhysicalThresholds)),
	}
	if network == nil || ports == nil || ports.Diagnostics == nil {
		return dist
	}
	major := make(map[int]struct{}, len(ports.MajorDeepwaterPorts))
	for _, nodeIdx := range ports.MajorDeepwaterPorts {
		major[nodeIdx] = struct{}{}
	}
	scores := make([]float64, 0, len(network.Nodes))
	physicals := make([]float64, 0, len(network.Nodes))
	for i, node := range network.Nodes {
		if i >= len(ports.Diagnostics.NodeDeepwaterScore) {
			continue
		}
		if !hasDeepwaterTerminal(ports.Diagnostics, i) {
			continue
		}
		if !oceanCandidateHasSettlementSupport(node) {
			continue
		}
		if _, isMajor := major[i]; !isMajor && node.Kind < SettlementNodeVillage {
			continue
		}
		score := ports.Diagnostics.NodeDeepwaterScore[i]
		physical := oceanCandidatePhysicalDeepwaterScore(ports.Diagnostics, i)
		civilized := civIDForNode(civByNode, i) >= 0
		dist.EligibleNodes++
		if civilized {
			dist.CivilizedNodes++
		}
		scores = append(scores, score)
		physicals = append(physicals, physical)
		for t, threshold := range oceanCandidateScoreThresholds {
			if score < threshold {
				continue
			}
			dist.ScoreCounts[t]++
			if !civilized {
				continue
			}
			dist.ScoreCivilized[t]++
			if physicalFloor <= 0 || physical >= physicalFloor {
				dist.ScorePassBoth[t]++
			}
		}
		for t, threshold := range oceanCandidatePhysicalThresholds {
			if physical >= threshold {
				dist.PhysicalCounts[t]++
			}
		}
	}
	if len(scores) == 0 {
		return dist
	}
	sort.Float64s(scores)
	sort.Float64s(physicals)
	dist.ScoreP50 = sortedFloatPercentile(scores, 0.50)
	dist.ScoreP75 = sortedFloatPercentile(scores, 0.75)
	dist.ScoreP90 = sortedFloatPercentile(scores, 0.90)
	dist.ScoreP95 = sortedFloatPercentile(scores, 0.95)
	dist.ScoreMax = scores[len(scores)-1]
	dist.PhysicalP50 = sortedFloatPercentile(physicals, 0.50)
	dist.PhysicalP75 = sortedFloatPercentile(physicals, 0.75)
	dist.PhysicalP90 = sortedFloatPercentile(physicals, 0.90)
	dist.PhysicalMax = physicals[len(physicals)-1]
	return dist
}
