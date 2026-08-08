package climgen

import (
	"testing"
)

// TestChannelCorridorWidthIsMeshInvariant pins the fix for the defect that made
// proto-civilizations fail to nucleate at L7.
//
// Channel initiation deliberately selects a linear feature, so the *centerline*
// cell count grows only with the square root of the mesh and its share of land
// halves every level. Consumers that cover area must therefore read a widened
// corridor; reading the centerline made the river-adjacent land fraction halve
// per level, which propagated into carrying capacity and the town gate.
//
// The corridor is measured as total weight rather than cell count, because the
// weight is what downstream scores integrate: a cell at corridor distance d
// contributes 1/(1+d*stepScale), and the number of cells covering a fixed
// physical width necessarily grows with the mesh.
func TestChannelCorridorWidthIsMeshInvariant(t *testing.T) {
	const width, height = 64, 64
	baseline := 0.0
	for _, level := range meshResolutionRefinements {
		cells := hexLatticeMesh(width, height, level.cellCount)
		elevation := make([]float64, level.cellCount)
		channel := make([]float64, level.cellCount)

		// A straight watercourse of constant physical length: the number of
		// cells it occupies doubles with each level, exactly as a real channel
		// network's does.
		for step := 0; step < 8*level.factor; step++ {
			idx := hexLatticeIndex(width, height, width/2-4*level.factor+step, height/2)
			channel[idx] = 3.0
		}

		hydro := &HydrologyBiomeInputs{
			ChannelStrength: channel,
			CellClass:       make([]string, level.cellCount),
		}
		got := ResolutionAdjustedHydrologyBiomeInputs(cells, elevation, 0, hydro)
		if len(got.ChannelCorridorStrength) != level.cellCount {
			t.Fatalf("%s: expected a corridor field of %d cells, got %d", level.name, level.cellCount, len(got.ChannelCorridorStrength))
		}

		// Total corridor weight scaled to baseline cell areas: each cell covers
		// scale^2 of a baseline cell.
		scale := meshPathCostResolutionScale(level.cellCount)
		total := 0.0
		for _, v := range got.ChannelCorridorStrength {
			total += v
		}
		area := total * scale * scale

		if level.name == "L5" {
			baseline = area
			// The corridor radius is zero at the baseline, so the field must be
			// the untouched centerline.
			for i := range channel {
				if got.ChannelCorridorStrength[i] != channel[i] {
					t.Fatalf("L5: corridor must leave the baseline centerline untouched, cell %d got %v want %v",
						i, got.ChannelCorridorStrength[i], channel[i])
				}
			}
			continue
		}
		if baseline == 0 {
			t.Fatalf("%s: no baseline captured", level.name)
		}
		if ratio := area / baseline; ratio < 0.85 || ratio > 1.45 {
			t.Errorf("%s: corridor area %.4f is %.2fx the L5 area %.4f; the centerline alone would give about 0.5x per level",
				level.name, area, ratio, baseline)
		}
	}
}
