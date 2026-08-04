package terrain

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"

	"worldgen/icosphere"
)

func TestGeneratePlanetElevationExportsTectonicDiagnostics(t *testing.T) {
	sites, faces := icosphere.CreateIcosphere(3)
	_, cells := icosphere.GenerateSphericalVoronoi(sites, faces)
	numRegions := len(sites)

	_, _, diagnostics := GeneratePlanetElevationWithDiagnostics(sites, cells, 8, 42, 0.29)
	tec := diagnostics.Tectonics

	if len(tec.PlateID) != numRegions {
		t.Fatalf("PlateID length = %d, want %d", len(tec.PlateID), numRegions)
	}
	if len(tec.PlateIsOcean) != numRegions {
		t.Fatalf("PlateIsOcean length = %d, want %d", len(tec.PlateIsOcean), numRegions)
	}

	// A boundary class can legitimately be absent on a given world (small meshes
	// often produce no continental rift or suture), so the invariant is that a
	// field only carries reachable cells when its seed class actually exists.
	type distanceField struct {
		values []float64
		seeds  int
	}
	distanceFields := map[string]distanceField{
		"distFromCoast":      {tec.DistFromCoast, tec.CoastlineSeeds},
		"oceanDistFromCoast": {tec.OceanDistFromCoast, tec.CoastlineSeeds},
		"distFromMountain":   {tec.DistFromMountain, tec.MountainSeeds},
		"distFromCollision":  {tec.DistFromCollision, tec.CollisionSeeds},
		"distFromArc":        {tec.DistFromArc, tec.ArcSeeds},
		"distFromRidge":      {tec.DistFromRidge, tec.RidgeSeeds},
		"distFromTrench":     {tec.DistFromTrench, tec.TrenchSeeds},
		"distFromRift":       {tec.DistFromRift, tec.RiftSeeds},
	}
	spacing := tec.CellAngularSpacing
	if spacing <= 0 || spacing > math.Pi {
		t.Fatalf("CellAngularSpacing = %v, want a positive angular spacing below pi", spacing)
	}
	for name, field := range distanceFields {
		if len(field.values) != numRegions {
			t.Fatalf("%s length = %d, want %d", name, len(field.values), numRegions)
		}
		finite := 0
		for r, value := range field.values {
			if !TectonicDistanceIsDefined(value) {
				continue
			}
			// Defined values are great-circle radians: non-negative, bounded by
			// half a circumference plus graph-path slack, and an exact multiple
			// of the mesh's neighbor spacing (they are hop counts scaled).
			if value < 0 || math.IsInf(value, 0) {
				t.Fatalf("%s[%d] = %v, want a non-negative angular distance or NaN", name, r, value)
			}
			hops := value / spacing
			if math.Abs(hops-math.Round(hops)) > 1e-9 {
				t.Fatalf("%s[%d] = %v is not an integral number of %v-radian hops", name, r, value, spacing)
			}
			finite++
		}
		if finite > 0 && field.seeds == 0 {
			t.Errorf("%s has %d reachable cells but no seeds of that class", name, finite)
		}
	}

	// The trap this sentinel exists to avoid: a naive proximity predicate must
	// not select cells where the field is undefined. On a mostly-ocean world the
	// continental-domain fields are undefined nearly everywhere, so -1 (or any
	// value sorting below real distances) would make this select most of the
	// sphere.
	nearThreshold := 3 * spacing
	for name, field := range distanceFields {
		selected := 0
		for r, value := range field.values {
			if value < nearThreshold {
				if !TectonicDistanceIsDefined(value) {
					t.Fatalf("%s[%d] = %v satisfies a proximity predicate while undefined", name, r, value)
				}
				selected++
			}
		}
		if defined := countReachable(field.values); selected > defined {
			t.Fatalf("%s: proximity predicate selected %d cells but only %d are defined", name, selected, defined)
		}
	}

	// Coastline seeds always exist, so both coast fields must be populated.
	if countReachable(tec.DistFromCoast) == 0 {
		t.Error("distFromCoast has no reachable cells; field looks unpopulated")
	}
	if countReachable(tec.OceanDistFromCoast) == 0 {
		t.Error("oceanDistFromCoast has no reachable cells; field looks unpopulated")
	}

	// Plate ids must be valid region indices, and every cell on a plate must
	// agree about that plate's oceanic/continental type.
	plateType := make(map[int]bool, tec.NumPlates)
	plateSeen := make(map[int]bool, tec.NumPlates)
	for r := 0; r < numRegions; r++ {
		plate := tec.PlateID[r]
		if plate < 0 || plate >= numRegions {
			t.Fatalf("PlateID[%d] = %d, want a region index in [0,%d)", r, plate, numRegions)
		}
		if seen, ok := plateType[plate]; ok && seen != tec.PlateIsOcean[r] {
			t.Fatalf("plate %d has inconsistent PlateIsOcean at region %d", plate, r)
		}
		plateType[plate] = tec.PlateIsOcean[r]
		plateSeen[plate] = true
	}
	if tec.NumPlates != len(plateSeen) {
		t.Fatalf("NumPlates = %d, want %d distinct plate ids", tec.NumPlates, len(plateSeen))
	}
	if tec.NumOceanicPlates < 0 || tec.NumOceanicPlates > tec.NumPlates {
		t.Fatalf("NumOceanicPlates = %d, want within [0,%d]", tec.NumOceanicPlates, tec.NumPlates)
	}

	// Ocean-domain fields must be unreachable on continental cells and vice
	// versa, which is what makes the sentinel meaningful downstream.
	for r := 0; r < numRegions; r++ {
		if tec.PlateIsOcean[r] {
			if TectonicDistanceIsDefined(tec.DistFromMountain[r]) {
				t.Fatalf("DistFromMountain[%d] = %v on an oceanic cell", r, tec.DistFromMountain[r])
			}
			if TectonicDistanceIsDefined(tec.DistFromRift[r]) {
				t.Fatalf("DistFromRift[%d] = %v on an oceanic cell", r, tec.DistFromRift[r])
			}
			continue
		}
		if TectonicDistanceIsDefined(tec.OceanDistFromCoast[r]) {
			t.Fatalf("OceanDistFromCoast[%d] = %v on a continental cell", r, tec.OceanDistFromCoast[r])
		}
		if TectonicDistanceIsDefined(tec.DistFromRidge[r]) {
			t.Fatalf("DistFromRidge[%d] = %v on a continental cell", r, tec.DistFromRidge[r])
		}
	}

	seedCounts := map[string]int{
		"coastline": tec.CoastlineSeeds,
		"mountain":  tec.MountainSeeds,
		"collision": tec.CollisionSeeds,
		"arc":       tec.ArcSeeds,
		"ridge":     tec.RidgeSeeds,
		"trench":    tec.TrenchSeeds,
		"rift":      tec.RiftSeeds,
	}
	for name, count := range seedCounts {
		if count < 0 || count > numRegions {
			t.Fatalf("%s seed count = %d, want within [0,%d]", name, count, numRegions)
		}
	}
	if tec.CoastlineSeeds == 0 {
		t.Error("expected at least one coastline seed")
	}

	// The NaN sentinel is not JSON-encodable, which is fine only because the
	// diagnostics carry a json:"-" tag on Tectonics. Guard that: the enclosing
	// diagnostics struct - what review tooling actually caches - must still
	// marshal, and it must not have quietly started serializing Tectonics.
	encoded, err := json.Marshal(diagnostics)
	if err != nil {
		t.Fatalf("planet diagnostics are not JSON-encodable: %v", err)
	}
	if bytes.Contains(encoded, []byte("distFromCoast")) {
		t.Fatal("Tectonics is being serialized; NaN distances cannot survive JSON, " +
			"so a consumer must map undefined cells explicitly first")
	}
}

func countReachable(field []float64) int {
	reachable := 0
	for _, value := range field {
		if TectonicDistanceIsDefined(value) {
			reachable++
		}
	}
	return reachable
}

// A physical feature is roughly twice as many hops away on a mesh with four
// times the cells, so exporting hop counts would double the reported distance
// per refinement level. Exporting radians must cancel that exactly.
func TestTectonicDiagnosticsDistancesAreResolutionStable(t *testing.T) {
	const coarseCells = 4
	const fineCells = 16 // one refinement level: 4x cells, 2x hops for the same feature

	build := func(cells int, hops float64) float64 {
		rPlate := make([]int, cells)
		buf := make([]float64, cells)
		for i := range buf {
			buf[i] = hops
		}
		tec := buildTectonicDiagnostics(rPlate, map[int]bool{0: false}, BoundarySeeds{}, tectonicDistanceFields{coast: buf})
		return tec.DistFromCoast[0]
	}

	coarse := build(coarseCells, 2)
	fine := build(fineCells, 4)
	if math.Abs(coarse-fine) > 1e-12 {
		t.Fatalf("same feature exports %v radians at %d cells but %v at %d cells; distances are resolution-dependent",
			coarse, coarseCells, fine, fineCells)
	}
}

func TestBuildTectonicDiagnosticsCopiesSourceBuffers(t *testing.T) {
	rPlate := []int{0, 0, 1, 1}
	plateIsOcean := map[int]bool{0: false, 1: true}
	dist := tectonicDistanceFields{
		coast:      []float64{0, 1, math.Inf(1), math.Inf(1)},
		oceanCoast: []float64{math.Inf(1), math.Inf(1), 0, 1},
		mountain:   []float64{2, 3, math.Inf(1), math.Inf(1)},
		collision:  []float64{4, 5, math.Inf(1), math.Inf(1)},
		arc:        []float64{6, 7, math.Inf(1), math.Inf(1)},
		ridge:      []float64{math.Inf(1), math.Inf(1), 8, 9},
		trench:     []float64{math.Inf(1), math.Inf(1), 10, 11},
		rift:       []float64{12, 13, math.Inf(1), math.Inf(1)},
	}
	seeds := BoundarySeeds{Rift: map[int]bool{0: true}}

	tec := buildTectonicDiagnostics(rPlate, plateIsOcean, seeds, dist)
	if tec.NumPlates != 2 || tec.NumOceanicPlates != 1 {
		t.Fatalf("plate counts = (%d,%d), want (2,1)", tec.NumPlates, tec.NumOceanicPlates)
	}
	if tec.RiftSeeds != 1 {
		t.Fatalf("RiftSeeds = %d, want 1", tec.RiftSeeds)
	}
	if TectonicDistanceIsDefined(tec.DistFromCoast[2]) {
		t.Fatalf("DistFromCoast[2] = %v, want the undefined sentinel", tec.DistFromCoast[2])
	}

	// Hops are exported as physical angular distance, not hop counts.
	spacing := meanCellAngularSpacing(len(rPlate))
	if tec.CellAngularSpacing != spacing {
		t.Fatalf("CellAngularSpacing = %v, want %v", tec.CellAngularSpacing, spacing)
	}
	if got, want := tec.DistFromCoast[1], 1*spacing; math.Abs(got-want) > 1e-12 {
		t.Fatalf("DistFromCoast[1] = %v, want %v (one hop in radians)", got, want)
	}
	if got, want := tec.DistFromMountain[1], 3*spacing; math.Abs(got-want) > 1e-12 {
		t.Fatalf("DistFromMountain[1] = %v, want %v (three hops in radians)", got, want)
	}

	// Mutating the source buffers afterwards must not reach the snapshot.
	dist.coast[0] = 99
	rPlate[0] = 7
	if tec.DistFromCoast[0] != 0 {
		t.Fatalf("DistFromCoast[0] = %v, want 0 (snapshot aliased source buffer)", tec.DistFromCoast[0])
	}
	if tec.PlateID[0] != 0 {
		t.Fatalf("PlateID[0] = %d, want 0 (snapshot aliased source buffer)", tec.PlateID[0])
	}
}
