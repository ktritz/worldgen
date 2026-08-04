package terrain

// Export of the plate-tectonic scaffolding computed during generation.
// This is a read-only snapshot: it never feeds back into terrain generation.

import "math"

// TectonicDistanceUnreachable marks cells where a tectonic distance field has
// no finite value, either because the cell lies outside that field's domain
// (ocean-only fields on continental cells and vice versa) or because no seed of
// that boundary class is reachable from it. The generator represents this as
// +Inf internally; the exported form uses a negative sentinel so the
// diagnostics remain JSON-encodable.
const TectonicDistanceUnreachable = -1.0

// tectonicDistanceFields groups the internal distance buffers so the exported
// snapshot can be assembled in one place.
type tectonicDistanceFields struct {
	coast      []float64
	oceanCoast []float64
	mountain   []float64
	collision  []float64
	arc        []float64
	ridge      []float64
	trench     []float64
	rift       []float64
}

// buildTectonicDiagnostics copies the generator's plate assignment, boundary
// seed counts, and distance fields into an exportable struct. Every slice is
// copied so later passes can never alias or mutate the diagnostics.
func buildTectonicDiagnostics(
	rPlate []int,
	plateIsOcean map[int]bool,
	seeds BoundarySeeds,
	dist tectonicDistanceFields,
) TectonicDiagnostics {
	numRegions := len(rPlate)

	plateID := make([]int, numRegions)
	isOcean := make([]bool, numRegions)
	seenPlates := make(map[int]bool, len(plateIsOcean))
	numOceanicPlates := 0
	for r := 0; r < numRegions; r++ {
		plate := rPlate[r]
		plateID[r] = plate
		isOcean[r] = plateIsOcean[plate]
		if !seenPlates[plate] {
			seenPlates[plate] = true
			if plateIsOcean[plate] {
				numOceanicPlates++
			}
		}
	}

	return TectonicDiagnostics{
		NumPlates:          len(seenPlates),
		NumOceanicPlates:   numOceanicPlates,
		PlateID:            plateID,
		PlateIsOcean:       isOcean,
		CoastlineSeeds:     len(seeds.Coastline),
		MountainSeeds:      len(seeds.Mountain),
		CollisionSeeds:     len(seeds.Collision),
		ArcSeeds:           len(seeds.Arc),
		RidgeSeeds:         len(seeds.Ridge),
		TrenchSeeds:        len(seeds.Trench),
		RiftSeeds:          len(seeds.Rift),
		DistFromCoast:      snapshotTectonicDistance(dist.coast, numRegions),
		OceanDistFromCoast: snapshotTectonicDistance(dist.oceanCoast, numRegions),
		DistFromMountain:   snapshotTectonicDistance(dist.mountain, numRegions),
		DistFromCollision:  snapshotTectonicDistance(dist.collision, numRegions),
		DistFromArc:        snapshotTectonicDistance(dist.arc, numRegions),
		DistFromRidge:      snapshotTectonicDistance(dist.ridge, numRegions),
		DistFromTrench:     snapshotTectonicDistance(dist.trench, numRegions),
		DistFromRift:       snapshotTectonicDistance(dist.rift, numRegions),
	}
}

// snapshotTectonicDistance copies a distance buffer, replacing non-finite
// entries with TectonicDistanceUnreachable.
func snapshotTectonicDistance(dist []float64, numRegions int) []float64 {
	out := make([]float64, numRegions)
	for r := 0; r < numRegions; r++ {
		if r >= len(dist) || math.IsInf(dist[r], 0) || math.IsNaN(dist[r]) {
			out[r] = TectonicDistanceUnreachable
			continue
		}
		out[r] = dist[r]
	}
	return out
}
