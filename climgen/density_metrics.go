package climgen

import "math"

// Density denominators for cross-resolution validation.
//
// Raw counts of discrete objects (corridors, ports, terminals) are small
// integers, so a single object forming or not swings an L6/L7 ratio by tens of
// percent, and a count cannot separate "the finer mesh puts more terminals on
// the same river" from "the finer mesh genuinely resolves more river". Both
// questions are answered by dividing the count by a physically meaningful
// extent: a sum over cells rather than a count of objects. Such a sum is large,
// so it barely quantizes, and the resulting density is the quantity that should
// actually be resolution-invariant.

// MeanCellAngularSpacing returns the mean center-to-center angular distance, in
// radians, between neighbouring cells of a geodesic mesh with cellCount cells.
// Cells tile the unit sphere hexagonally, so a cell of area 4*pi/n has center
// spacing d satisfying (sqrt(3)/2)*d^2 = 4*pi/n, hence d = sqrt(8*pi/(sqrt(3)*n)).
// This mirrors terrain.meanCellAngularSpacing and is deliberately unclamped: it
// is a statement about the mesh, not a tuning envelope.
func MeanCellAngularSpacing(cellCount int) float64 {
	if cellCount <= 0 {
		return 0
	}
	return math.Sqrt(8 * math.Pi / (math.Sqrt(3) * float64(cellCount)))
}

// MeanCellSolidAngle returns the mean solid angle (steradians) subtended by one
// cell of a geodesic mesh with cellCount cells: 4*pi/n.
func MeanCellSolidAngle(cellCount int) float64 {
	if cellCount <= 0 {
		return 0
	}
	return 4 * math.Pi / float64(cellCount)
}

// MeshDensityDenominators holds resolution-invariant physical extents derived
// from the mesh. Lengths are angular (radians on the unit sphere); areas are
// solid angles (steradians). Multiply by a planet radius to get km / km^2.
type MeshDensityDenominators struct {
	CellCount       int
	MeanCellSpacing float64 // radians
	MeanCellArea    float64 // steradians

	LandCells    int
	OceanCells   int
	CoastalCells int

	// NavLength is the total navigable river extent: the per-cell navigability
	// field summed over land cells and converted to a length by the mean cell
	// spacing. Navigability is a [0,1] weight, so this is an effective length,
	// not a count of navigable cells.
	NavLength float64 // radians

	// CoastLength is the total coastline extent: land cells adjacent to ocean,
	// converted to a length by the mean cell spacing.
	CoastLength float64 // radians

	// OceanArea is the total ocean solid angle: ocean cells times mean cell area.
	OceanArea float64 // steradians
}

// ComputeMeshDensityDenominators sums the physical extents used as denominators
// for structural densities. navigability may be nil, in which case NavLength is
// zero. Land is elevation >= seaLevel.
func ComputeMeshDensityDenominators(
	cells []VoronoiCell,
	elevation []float64,
	seaLevel float64,
	navigability []float64,
) MeshDensityDenominators {
	out := MeshDensityDenominators{CellCount: len(elevation)}
	if len(elevation) == 0 {
		return out
	}
	out.MeanCellSpacing = MeanCellAngularSpacing(out.CellCount)
	out.MeanCellArea = MeanCellSolidAngle(out.CellCount)

	sumNav := 0.0
	for i := range elevation {
		if elevation[i] < seaLevel {
			out.OceanCells++
			continue
		}
		out.LandCells++
		if i < len(navigability) {
			sumNav += navigability[i]
		}
		if i < len(cells) && cellTouchesOcean(cells[i], elevation, seaLevel) {
			out.CoastalCells++
		}
	}
	out.NavLength = sumNav * out.MeanCellSpacing
	out.CoastLength = float64(out.CoastalCells) * out.MeanCellSpacing
	out.OceanArea = float64(out.OceanCells) * out.MeanCellArea
	return out
}

func cellTouchesOcean(cell VoronoiCell, elevation []float64, seaLevel float64) bool {
	for _, neighbor := range cell.NeighborSiteIndices {
		idx := int(neighbor)
		if idx < 0 || idx >= len(elevation) {
			continue
		}
		if elevation[idx] < seaLevel {
			return true
		}
	}
	return false
}

// PerExtent divides a count by a physical extent, returning 0 when the extent is
// degenerate so summary lines stay parseable instead of printing Inf/NaN.
func PerExtent(count int, extent float64) float64 {
	if extent <= 0 {
		return 0
	}
	return float64(count) / extent
}
