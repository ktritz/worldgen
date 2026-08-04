package terrain

import (
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
	for name, field := range distanceFields {
		if len(field.values) != numRegions {
			t.Fatalf("%s length = %d, want %d", name, len(field.values), numRegions)
		}
		finite := 0
		for r, value := range field.values {
			if value == TectonicDistanceUnreachable {
				continue
			}
			if value < 0 || math.IsInf(value, 0) || math.IsNaN(value) {
				t.Fatalf("%s[%d] = %v, want a non-negative hop count or %v", name, r, value, TectonicDistanceUnreachable)
			}
			if value != math.Trunc(value) {
				t.Fatalf("%s[%d] = %v, want an integral hop count", name, r, value)
			}
			finite++
		}
		if finite > 0 && field.seeds == 0 {
			t.Errorf("%s has %d reachable cells but no seeds of that class", name, finite)
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
			if tec.DistFromMountain[r] != TectonicDistanceUnreachable {
				t.Fatalf("DistFromMountain[%d] = %v on an oceanic cell", r, tec.DistFromMountain[r])
			}
			if tec.DistFromRift[r] != TectonicDistanceUnreachable {
				t.Fatalf("DistFromRift[%d] = %v on an oceanic cell", r, tec.DistFromRift[r])
			}
			continue
		}
		if tec.OceanDistFromCoast[r] != TectonicDistanceUnreachable {
			t.Fatalf("OceanDistFromCoast[%d] = %v on a continental cell", r, tec.OceanDistFromCoast[r])
		}
		if tec.DistFromRidge[r] != TectonicDistanceUnreachable {
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

	// The diagnostics are cached as JSON by review tooling, so they must not
	// carry the generator's internal +Inf sentinels.
	if _, err := json.Marshal(tec); err != nil {
		t.Fatalf("tectonic diagnostics are not JSON-encodable: %v", err)
	}
}

func countReachable(field []float64) int {
	reachable := 0
	for _, value := range field {
		if value != TectonicDistanceUnreachable {
			reachable++
		}
	}
	return reachable
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
	if tec.DistFromCoast[2] != TectonicDistanceUnreachable {
		t.Fatalf("DistFromCoast[2] = %v, want %v", tec.DistFromCoast[2], TectonicDistanceUnreachable)
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
