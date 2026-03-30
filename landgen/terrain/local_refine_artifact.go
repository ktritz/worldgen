package terrain

import (
	"fmt"
	"math"
	"sort"
)

// LocalRefinementArtifact is the persisted result for a refined coarse cell.
// It stores the center-cell boundary contracts inferred from the local patch so
// later refinements can reuse them instead of regenerating everything from the
// coarse scaffold.
type LocalRefinementArtifact struct {
	Cell                 int                         `json:"cell"`
	Seed                 int64                       `json:"seed"`
	SettingsSignature    string                      `json:"settingsSignature"`
	MeanBoundaryMismatch float64                     `json:"meanBoundaryMismatch"`
	MaxBoundaryMismatch  float64                     `json:"maxBoundaryMismatch"`
	LakeCoveragePct      float64                     `json:"lakeCoveragePct"`
	ChannelCoveragePct   float64                     `json:"channelCoveragePct"`
	BoundaryCrossings    []LocalCellBoundaryCrossing `json:"boundaryCrossings,omitempty"`
}

// CanonicalBoundaryCrossing is a reconciled crossing on a shared coarse cell
// boundary after two refined neighboring cells have both been generated.
type CanonicalBoundaryCrossing struct {
	CellA        int     `json:"cellA"`
	CellB        int     `json:"cellB"`
	Kind         string  `json:"kind"`
	OffsetA      float64 `json:"offsetA"`
	OffsetB      float64 `json:"offsetB"`
	BearingABDeg float64 `json:"bearingABDeg"`
	WidthPx      int     `json:"widthPx"`
	Strength     float64 `json:"strength"`
	SourceCount  int     `json:"sourceCount"`
}

// LocalRefinementMerge summarizes the canonical shared-boundary contracts
// extracted from two neighboring local refinement artifacts.
type LocalRefinementMerge struct {
	CellA      int                         `json:"cellA"`
	CellB      int                         `json:"cellB"`
	Crossings  []CanonicalBoundaryCrossing `json:"crossings,omitempty"`
	UnmatchedA []LocalCellBoundaryCrossing `json:"unmatchedA,omitempty"`
	UnmatchedB []LocalCellBoundaryCrossing `json:"unmatchedB,omitempty"`
}

// BoundaryContractSource records which refined cell artifacts most recently
// contributed to a canonical shared-boundary contract.
type BoundaryContractSource struct {
	Cell              int    `json:"cell"`
	Seed              int64  `json:"seed"`
	SettingsSignature string `json:"settingsSignature"`
}

// SharedBoundaryContract is the canonical persisted contract for a coarse
// boundary shared by two neighboring cells.
type SharedBoundaryContract struct {
	CellA      int                         `json:"cellA"`
	CellB      int                         `json:"cellB"`
	Version    int                         `json:"version"`
	Crossings  []CanonicalBoundaryCrossing `json:"crossings,omitempty"`
	Sources    []BoundaryContractSource    `json:"sources,omitempty"`
	DirtyCells []int                       `json:"dirtyCells,omitempty"`
}

// BuildLocalRefinementArtifact converts a refined patch into a persisted
// center-cell artifact keyed by the patch's center coarse cell.
func BuildLocalRefinementArtifact(
	patch *LocalRefinementPatch,
	seed int64,
	settings LocalRefinementSettings,
) *LocalRefinementArtifact {
	if patch == nil {
		return nil
	}
	return &LocalRefinementArtifact{
		Cell:                 patch.Debug.CenterCell,
		Seed:                 seed,
		SettingsSignature:    localRefinementSettingsSignature(settings),
		MeanBoundaryMismatch: patch.Debug.MeanBoundaryMismatch,
		MaxBoundaryMismatch:  patch.Debug.MaxBoundaryMismatch,
		LakeCoveragePct:      patch.Debug.LakeCoveragePct,
		ChannelCoveragePct:   patch.Debug.ChannelCoveragePct,
		BoundaryCrossings:    append([]LocalCellBoundaryCrossing(nil), patch.Debug.CellBoundaryCrossings...),
	}
}

// MergeLocalRefinementArtifacts reconciles the shared-boundary crossings of two
// neighboring refined coarse cells into canonical contracts.
func MergeLocalRefinementArtifacts(a, b *LocalRefinementArtifact) *LocalRefinementMerge {
	if a == nil || b == nil {
		return nil
	}
	aCross := make([]LocalCellBoundaryCrossing, 0)
	for _, crossing := range a.BoundaryCrossings {
		if crossing.Neighbor == b.Cell {
			aCross = append(aCross, crossing)
		}
	}
	bCross := make([]LocalCellBoundaryCrossing, 0)
	for _, crossing := range b.BoundaryCrossings {
		if crossing.Neighbor == a.Cell {
			bCross = append(bCross, crossing)
		}
	}

	usedB := make([]bool, len(bCross))
	out := make([]CanonicalBoundaryCrossing, 0)
	unmatchedA := make([]LocalCellBoundaryCrossing, 0)
	for _, ca := range aCross {
		best := -1
		bestScore := math.Inf(1)
		for j, cb := range bCross {
			if usedB[j] {
				continue
			}
			score := math.Abs(ca.Offset + cb.Offset)
			if score < bestScore {
				bestScore = score
				best = j
			}
		}
		if best < 0 {
			unmatchedA = append(unmatchedA, ca)
			continue
		}
		usedB[best] = true
		cb := bCross[best]
		kind := ca.Kind
		if kind == "" {
			kind = cb.Kind
		}
		if kind == "" {
			kind = "channel"
		}
		out = append(out, CanonicalBoundaryCrossing{
			CellA:        a.Cell,
			CellB:        b.Cell,
			Kind:         kind,
			OffsetA:      ca.Offset,
			OffsetB:      cb.Offset,
			BearingABDeg: ca.BearingDeg,
			WidthPx:      int(math.Round(0.5 * float64(ca.WidthPx+cb.WidthPx))),
			Strength:     0.5 * (ca.Strength + cb.Strength),
			SourceCount:  2,
		})
	}
	unmatchedB := make([]LocalCellBoundaryCrossing, 0)
	for j, cb := range bCross {
		if !usedB[j] {
			unmatchedB = append(unmatchedB, cb)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].OffsetA < out[j].OffsetA
	})
	return &LocalRefinementMerge{
		CellA:      a.Cell,
		CellB:      b.Cell,
		Crossings:  out,
		UnmatchedA: unmatchedA,
		UnmatchedB: unmatchedB,
	}
}

func localRefinementSettingsSignature(settings LocalRefinementSettings) string {
	return fmt.Sprintf("r%d_res%d_nf%.3f_no%d_na%.3f_cd%.3f_m%.3f_lf%.3f_sf%.3f",
		settings.Radius,
		settings.Resolution,
		settings.NoiseFrequency,
		settings.NoiseOctaves,
		settings.NoiseAmplitude,
		settings.ChannelDepthScale,
		settings.MarginScale,
		settings.LakeFlattening,
		settings.SideflowSplitScale,
	)
}

// BuildSharedBoundaryContract converts two refined cell artifacts into a
// canonical versioned boundary contract. If a previous contract is supplied,
// the version increments only when the canonical crossings materially change.
func BuildSharedBoundaryContract(
	prev *SharedBoundaryContract,
	a, b *LocalRefinementArtifact,
) *SharedBoundaryContract {
	merge := MergeLocalRefinementArtifacts(a, b)
	if merge == nil {
		return nil
	}
	cellA, cellB := canonicalCellPair(a.Cell, b.Cell)
	contract := &SharedBoundaryContract{
		CellA:     cellA,
		CellB:     cellB,
		Version:   1,
		Crossings: append([]CanonicalBoundaryCrossing(nil), merge.Crossings...),
		Sources: []BoundaryContractSource{
			{Cell: a.Cell, Seed: a.Seed, SettingsSignature: a.SettingsSignature},
			{Cell: b.Cell, Seed: b.Seed, SettingsSignature: b.SettingsSignature},
		},
	}
	sort.Slice(contract.Sources, func(i, j int) bool { return contract.Sources[i].Cell < contract.Sources[j].Cell })
	if prev != nil {
		contract.Version = prev.Version
		if boundaryCrossingsChanged(prev.Crossings, contract.Crossings) {
			contract.Version++
		}
	}
	contract.DirtyCells = nil
	return contract
}

// ApplyArtifactToBoundaryContract updates an existing shared-boundary contract
// from one side only. If the proposing artifact materially changes the
// canonical crossing set, the opposite cell is marked stale so it can be
// rerun against the new boundary contract later.
func ApplyArtifactToBoundaryContract(
	prev *SharedBoundaryContract,
	updated *LocalRefinementArtifact,
) *SharedBoundaryContract {
	if prev == nil || updated == nil {
		return prev
	}
	if updated.Cell != prev.CellA && updated.Cell != prev.CellB {
		return prev
	}
	otherCell := prev.CellA
	updatedIsA := true
	if updated.Cell == prev.CellA {
		otherCell = prev.CellB
	} else {
		otherCell = prev.CellA
		updatedIsA = false
	}

	updatedCrossings := make([]LocalCellBoundaryCrossing, 0)
	for _, crossing := range updated.BoundaryCrossings {
		if crossing.Neighbor == otherCell {
			updatedCrossings = append(updatedCrossings, crossing)
		}
	}

	usedPrev := make([]bool, len(prev.Crossings))
	nextCrossings := make([]CanonicalBoundaryCrossing, 0, len(updatedCrossings))
	for _, uc := range updatedCrossings {
		best := -1
		bestScore := math.Inf(1)
		for i, pc := range prev.Crossings {
			if usedPrev[i] {
				continue
			}
			score := math.Abs(contractOffsetForCell(pc, updated.Cell) - uc.Offset)
			if score < bestScore {
				bestScore = score
				best = i
			}
		}
		canonical := CanonicalBoundaryCrossing{
			CellA:        prev.CellA,
			CellB:        prev.CellB,
			Kind:         uc.Kind,
			BearingABDeg: uc.BearingDeg,
			WidthPx:      uc.WidthPx,
			Strength:     uc.Strength,
			SourceCount:  1,
		}
		if canonical.Kind == "" {
			canonical.Kind = "channel"
		}
		if updatedIsA {
			canonical.OffsetA = uc.Offset
		} else {
			canonical.OffsetB = uc.Offset
		}
		if best >= 0 {
			usedPrev[best] = true
			pc := prev.Crossings[best]
			canonical.Kind = firstNonEmpty(canonical.Kind, pc.Kind, "channel")
			canonical.WidthPx = int(math.Round(0.5 * float64(canonical.WidthPx+pc.WidthPx)))
			canonical.Strength = 0.5 * (canonical.Strength + pc.Strength)
			canonical.SourceCount = 2
			if updatedIsA {
				canonical.OffsetB = pc.OffsetB
			} else {
				canonical.OffsetA = pc.OffsetA
			}
		} else {
			if updatedIsA {
				canonical.OffsetB = -uc.Offset
			} else {
				canonical.OffsetA = -uc.Offset
			}
		}
		nextCrossings = append(nextCrossings, canonical)
	}

	sort.Slice(nextCrossings, func(i, j int) bool {
		return nextCrossings[i].OffsetA < nextCrossings[j].OffsetA
	})

	dirty := make([]int, 0, 1)
	for _, cell := range prev.DirtyCells {
		if cell != updated.Cell && cell != otherCell {
			dirty = append(dirty, cell)
		}
	}
	version := prev.Version
	if boundaryCrossingsChanged(prev.Crossings, nextCrossings) {
		version++
		dirty = appendUniqueInt(dirty, otherCell)
	}

	out := &SharedBoundaryContract{
		CellA:      prev.CellA,
		CellB:      prev.CellB,
		Version:    version,
		Crossings:  nextCrossings,
		Sources:    replaceBoundarySource(prev.Sources, updated),
		DirtyCells: dirty,
	}
	return out
}

// BuildRefinementRerunQueue returns the cells that should be rerun next for a
// given shared-boundary contract, ordered by cell id for deterministic use.
func BuildRefinementRerunQueue(contract *SharedBoundaryContract) []int {
	if contract == nil || len(contract.DirtyCells) == 0 {
		return nil
	}
	out := append([]int(nil), contract.DirtyCells...)
	sort.Ints(out)
	return out
}

// ClearBoundaryDirtyCell removes one cell from a contract's stale set after it
// has been rerun against the latest shared boundary.
func ClearBoundaryDirtyCell(contract *SharedBoundaryContract, cell int) *SharedBoundaryContract {
	if contract == nil || len(contract.DirtyCells) == 0 {
		return contract
	}
	out := *contract
	out.DirtyCells = make([]int, 0, len(contract.DirtyCells))
	for _, dirty := range contract.DirtyCells {
		if dirty != cell {
			out.DirtyCells = append(out.DirtyCells, dirty)
		}
	}
	return &out
}

func canonicalCellPair(a, b int) (int, int) {
	if a <= b {
		return a, b
	}
	return b, a
}

func boundaryCrossingsChanged(a, b []CanonicalBoundaryCrossing) bool {
	if len(a) != len(b) {
		return true
	}
	for i := range a {
		if a[i].CellA != b[i].CellA || a[i].CellB != b[i].CellB || a[i].Kind != b[i].Kind {
			return true
		}
		if math.Abs(a[i].OffsetA-b[i].OffsetA) > 1e-3 || math.Abs(a[i].OffsetB-b[i].OffsetB) > 1e-3 {
			return true
		}
		if math.Abs(a[i].Strength-b[i].Strength) > 1e-3 {
			return true
		}
		if a[i].WidthPx != b[i].WidthPx {
			return true
		}
	}
	return false
}

func contractOffsetForCell(c CanonicalBoundaryCrossing, cell int) float64 {
	if cell == c.CellA {
		return c.OffsetA
	}
	return c.OffsetB
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func replaceBoundarySource(prev []BoundaryContractSource, updated *LocalRefinementArtifact) []BoundaryContractSource {
	out := make([]BoundaryContractSource, 0, len(prev)+1)
	replaced := false
	for _, src := range prev {
		if src.Cell == updated.Cell {
			out = append(out, BoundaryContractSource{
				Cell:              updated.Cell,
				Seed:              updated.Seed,
				SettingsSignature: updated.SettingsSignature,
			})
			replaced = true
			continue
		}
		out = append(out, src)
	}
	if !replaced {
		out = append(out, BoundaryContractSource{
			Cell:              updated.Cell,
			Seed:              updated.Seed,
			SettingsSignature: updated.SettingsSignature,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Cell < out[j].Cell })
	return out
}

func appendUniqueInt(values []int, value int) []int {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
