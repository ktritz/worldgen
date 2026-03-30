package terrain

import (
	"fmt"
	"sort"
)

// LocalRefinementManager coordinates patch generation, artifact persistence,
// stale-neighbor lookup, and shared-boundary reconciliation.
type LocalRefinementManager struct {
	Sites      []Vector3D
	Cells      []VoronoiCell
	Elevation  []float64
	Hydrology  *HydrologyScaffold
	Refinement *TerrainRefinementScaffold
	Seed       int64
	Settings   LocalRefinementSettings
	Store      *LocalRefinementStore
}

// LocalRefinementManagedResult captures the end-to-end result of refining one
// cell and reconciling one neighbor.
type LocalRefinementManagedResult struct {
	CenterCell       int
	NeighborCell     int
	CenterPatch      *LocalRefinementPatch
	NeighborPatch    *LocalRefinementPatch
	CenterArtifact   *LocalRefinementArtifact
	NeighborArtifact *LocalRefinementArtifact
	Merge            *LocalRefinementMerge
	Contract         *SharedBoundaryContract
	Neighbors        []LocalRefinementManagedNeighbor
}

// LocalRefinementManagedNeighbor captures one reconciled neighbor refinement in
// a center-cell cascade run.
type LocalRefinementManagedNeighbor struct {
	Cell     int
	Patch    *LocalRefinementPatch
	Artifact *LocalRefinementArtifact
	Merge    *LocalRefinementMerge
	Contract *SharedBoundaryContract
}

// RefineCell refines one coarse cell, consults stale-neighbor queue state,
// refines one neighbor, and persists artifacts/contracts automatically.
func (m *LocalRefinementManager) RefineCell(centerCell int) (*LocalRefinementManagedResult, error) {
	return m.RefineCellNeighborhood(centerCell, 1)
}

// RefineCellNeighborhood refines one center cell and then reconciles up to
// `maxNeighbors` neighboring cells in deterministic order.
func (m *LocalRefinementManager) RefineCellNeighborhood(centerCell int, maxNeighbors int) (*LocalRefinementManagedResult, error) {
	if m.Hydrology == nil || m.Refinement == nil || m.Store == nil {
		return nil, fmt.Errorf("manager missing hydrology, refinement scaffold, or store")
	}
	if centerCell < 0 || centerCell >= len(m.Elevation) {
		return nil, fmt.Errorf("center cell %d out of range", centerCell)
	}
	if maxNeighbors < 0 {
		maxNeighbors = 0
	}
	neighbors := m.chooseNeighbors(centerCell, maxNeighbors)

	var initialContract *SharedBoundaryContract
	var err error
	if len(neighbors) > 0 {
		initialContract, err = m.Store.LoadContract(centerCell, neighbors[0])
		if err != nil {
			return nil, err
		}
	}

	centerPatch, err := BuildLocalRefinementPatch(
		m.Sites, m.Cells, m.Elevation,
		m.Hydrology, m.Refinement,
		centerCell, m.Seed, m.Settings, initialContract,
	)
	if err != nil {
		return nil, err
	}
	centerArtifact := BuildLocalRefinementArtifact(centerPatch, m.Seed, m.Settings)
	if err := m.Store.SaveArtifact(centerArtifact); err != nil {
		return nil, err
	}

	result := &LocalRefinementManagedResult{
		CenterCell:     centerCell,
		CenterPatch:    centerPatch,
		CenterArtifact: centerArtifact,
	}
	if len(neighbors) == 0 {
		return result, nil
	}
	for _, neighborCell := range neighbors {
		contract, err := m.Store.LoadContract(centerCell, neighborCell)
		if err != nil {
			return nil, err
		}
		if contract != nil {
			contract = ApplyArtifactToBoundaryContract(contract, centerArtifact)
			if err := m.Store.SaveContract(contract); err != nil {
				return nil, err
			}
		}

		neighborPatch, err := BuildLocalRefinementPatch(
			m.Sites, m.Cells, m.Elevation,
			m.Hydrology, m.Refinement,
			neighborCell, m.Seed, m.Settings, contract,
		)
		if err != nil {
			return nil, err
		}
		neighborArtifact := BuildLocalRefinementArtifact(neighborPatch, m.Seed, m.Settings)
		if err := m.Store.SaveArtifact(neighborArtifact); err != nil {
			return nil, err
		}

		prevContract, err := m.Store.LoadContract(centerCell, neighborCell)
		if err != nil {
			return nil, err
		}
		finalContract := BuildSharedBoundaryContract(prevContract, centerArtifact, neighborArtifact)
		if finalContract != nil {
			finalContract = ClearBoundaryDirtyCell(finalContract, neighborArtifact.Cell)
			if err := m.Store.SaveContract(finalContract); err != nil {
				return nil, err
			}
		}
		neighborResult := LocalRefinementManagedNeighbor{
			Cell:     neighborCell,
			Patch:    neighborPatch,
			Artifact: neighborArtifact,
			Merge:    MergeLocalRefinementArtifacts(centerArtifact, neighborArtifact),
			Contract: finalContract,
		}
		result.Neighbors = append(result.Neighbors, neighborResult)
	}
	if len(result.Neighbors) > 0 {
		first := result.Neighbors[0]
		result.NeighborCell = first.Cell
		result.NeighborPatch = first.Patch
		result.NeighborArtifact = first.Artifact
		result.Merge = first.Merge
		result.Contract = first.Contract
	}
	return result, nil
}

// QueuedDirtyNeighbors exposes the current stale-neighbor queue for a given
// center cell by scanning persisted shared-boundary contracts in the store.
func (m *LocalRefinementManager) QueuedDirtyNeighbors(centerCell int) ([]int, error) {
	if m.Store == nil {
		return nil, fmt.Errorf("manager missing store")
	}
	return m.Store.QueuedDirtyNeighbors(centerCell)
}

// ChooseAdjacentRefinementNeighbor selects a plausible neighboring cell when no
// stale-neighbor queue overrides the choice.
func ChooseAdjacentRefinementNeighbor(cells []VoronoiCell, scaffold *HydrologyScaffold, elevation []float64, center int) int {
	ranked := RankAdjacentRefinementNeighbors(cells, scaffold, elevation, center)
	if len(ranked) == 0 {
		return -1
	}
	return ranked[0]
}

// RankAdjacentRefinementNeighbors returns neighboring land cells ordered by
// their usefulness as refinement targets.
func RankAdjacentRefinementNeighbors(cells []VoronoiCell, scaffold *HydrologyScaffold, elevation []float64, center int) []int {
	if center < 0 || center >= len(cells) {
		return nil
	}
	type scoredNeighbor struct {
		idx   int
		score float64
	}
	scored := make([]scoredNeighbor, 0, len(cells[center].NeighborSiteIndices))
	for _, nidx := range cells[center].NeighborSiteIndices {
		idx := int(nidx)
		if idx < 0 || idx >= len(elevation) || elevation[idx] <= 0 {
			continue
		}
		score := 0.0
		if idx < len(scaffold.CellClass) {
			switch scaffold.CellClass[idx] {
			case "trunk":
				score += 4.0
			case "confluence":
				score += 3.5
			case "floodplain":
				score += 3.25
			case "delta":
				score += 3.0
			case "lake_reach":
				score += 2.5
			case "coast_outlet":
				score += 2.0
			case "headwater":
				score += 1.0
			case "hillslope":
				score += 0.5
			}
		}
		if idx < len(scaffold.ChannelStrength) {
			score += 1.5 * scaffold.ChannelStrength[idx]
		}
		if idx < len(scaffold.Accumulation) {
			score += 0.03 * scaffold.Accumulation[idx]
		}
		scored = append(scored, scoredNeighbor{idx: idx, score: score})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].idx < scored[j].idx
		}
		return scored[i].score > scored[j].score
	})
	out := make([]int, 0, len(scored))
	for _, neighbor := range scored {
		out = append(out, neighbor.idx)
	}
	return out
}

func (m *LocalRefinementManager) chooseNeighbor(centerCell int) int {
	neighbors := m.chooseNeighbors(centerCell, 1)
	if len(neighbors) == 0 {
		return -1
	}
	return neighbors[0]
}

func (m *LocalRefinementManager) chooseNeighbors(centerCell int, maxNeighbors int) []int {
	if maxNeighbors == 0 {
		return nil
	}
	out := make([]int, 0, maxNeighbors)
	if queued, err := m.Store.QueuedDirtyNeighbors(centerCell); err == nil {
		for _, cell := range queued {
			if cell != centerCell {
				out = append(out, cell)
				if maxNeighbors > 0 && len(out) >= maxNeighbors {
					return out
				}
			}
		}
	}
	for _, fallback := range RankAdjacentRefinementNeighbors(m.Cells, m.Hydrology, m.Elevation, centerCell) {
		seen := false
		for _, existing := range out {
			if existing == fallback {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, fallback)
			if maxNeighbors > 0 && len(out) >= maxNeighbors {
				break
			}
		}
	}
	if maxNeighbors > 0 && len(out) > maxNeighbors {
		out = out[:maxNeighbors]
	}
	return out
}
