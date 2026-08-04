package terrain

// Export of the plate-tectonic scaffolding computed during generation.
// This is a read-only snapshot: it never feeds back into terrain generation.

import "math"

// TectonicDistanceUndefined returns the value stored in a tectonic distance
// field for cells where that field has no finite value, either because the cell
// lies outside the field's domain (ocean-only fields on continental cells and
// vice versa) or because no seed of that boundary class is reachable from it.
//
// The sentinel is NaN, deliberately. A numeric sentinel such as -1 sorts below
// every real distance, so the natural proximity predicate `d < threshold`
// selects exactly the cells where the field is undefined - and since the fields
// are domain-restricted, that is most of the sphere. NaN makes every comparison
// false and poisons any arithmetic it touches, so a consumer that forgets to
// check gets an empty selection or a NaN-filled output rather than a plausible
// wrong answer.
//
// Use TectonicDistanceIsDefined to test a value; NaN != NaN, so direct equality
// comparison against this sentinel never matches.
func TectonicDistanceUndefined() float64 { return math.NaN() }

// TectonicDistanceIsDefined reports whether a tectonic distance value is a real
// measurement rather than the undefined sentinel.
func TectonicDistanceIsDefined(distance float64) bool { return !math.IsNaN(distance) }

// tectonicDistanceFields groups the internal distance buffers so the exported
// snapshot can be assembled in one place. The buffers hold graph-hop counts
// over the Voronoi neighbor graph; the snapshot converts them to physical
// angular distance.
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
//
// Hop counts are converted to great-circle radians using the mean neighbor
// spacing of a mesh with this many cells, so the same physical world exports
// the same distances at every resolution.
func buildTectonicDiagnostics(
	rPlate []int,
	plateIsOcean map[int]bool,
	seeds BoundarySeeds,
	dist tectonicDistanceFields,
) TectonicDiagnostics {
	numRegions := len(rPlate)
	hopRadians := meanCellAngularSpacing(numRegions)

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
		CellAngularSpacing: hopRadians,
		CoastlineSeeds:     len(seeds.Coastline),
		MountainSeeds:      len(seeds.Mountain),
		CollisionSeeds:     len(seeds.Collision),
		ArcSeeds:           len(seeds.Arc),
		RidgeSeeds:         len(seeds.Ridge),
		TrenchSeeds:        len(seeds.Trench),
		RiftSeeds:          len(seeds.Rift),
		DistFromCoast:      snapshotTectonicDistance(dist.coast, numRegions, hopRadians),
		OceanDistFromCoast: snapshotTectonicDistance(dist.oceanCoast, numRegions, hopRadians),
		DistFromMountain:   snapshotTectonicDistance(dist.mountain, numRegions, hopRadians),
		DistFromCollision:  snapshotTectonicDistance(dist.collision, numRegions, hopRadians),
		DistFromArc:        snapshotTectonicDistance(dist.arc, numRegions, hopRadians),
		DistFromRidge:      snapshotTectonicDistance(dist.ridge, numRegions, hopRadians),
		DistFromTrench:     snapshotTectonicDistance(dist.trench, numRegions, hopRadians),
		DistFromRift:       snapshotTectonicDistance(dist.rift, numRegions, hopRadians),
	}
}

// snapshotTectonicDistance copies a distance buffer, converting hop counts to
// great-circle radians and replacing non-finite entries with the undefined
// sentinel.
func snapshotTectonicDistance(dist []float64, numRegions int, hopRadians float64) []float64 {
	out := make([]float64, numRegions)
	for r := 0; r < numRegions; r++ {
		if r >= len(dist) || math.IsInf(dist[r], 0) || math.IsNaN(dist[r]) || dist[r] < 0 {
			out[r] = TectonicDistanceUndefined()
			continue
		}
		out[r] = dist[r] * hopRadians
	}
	return out
}
