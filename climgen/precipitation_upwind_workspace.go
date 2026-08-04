package climgen

import "sync"

// upwindFootprintWorkspace holds reusable scratch buffers for the upwind
// footprint BFS. The BFS used to allocate two maps per depth level plus two
// donor slices per visited cell; at high resolution the footprint depth scales
// with the mesh so those allocations dominated the precipitation stage. The
// workspace replaces the maps with stamped index arrays and reuses the donor
// scratch buffers across calls. Results are numerically identical apart from
// summation order (which is now deterministic rather than map-iteration order).
type upwindFootprintWorkspace struct {
	// Accumulator: cell -> slot in accCell/accW, valid when accGen[cell] == gen.
	accGen  []int32
	accSlot []int32
	accCell []int32
	accW    []float64

	// Frontier for the next depth level, same stamped-slot scheme with dgen.
	nextGen  []int32
	nextSlot []int32
	nextCell []int32
	nextW    []float64

	curCell []int32
	curW    []float64

	gen  int32
	dgen int32

	donorBuf  []int32
	weightBuf []float64

	// Cached per-edge upwind unit directions for the current (adj, vertices)
	// pair. dirs is parallel to adj.Neighbors: dirs[e] is the unit vector from
	// neighbor adj.Neighbors[e] toward its owning cell.
	dirs      []Vector3D
	dirsAdj   *FlatAdjacency
	dirsVerts *Vector3D
	dirsCount int
}

var upwindWorkspacePool = sync.Pool{
	New: func() any { return &upwindFootprintWorkspace{} },
}

func acquireUpwindWorkspace(n int) *upwindFootprintWorkspace {
	ws, _ := upwindWorkspacePool.Get().(*upwindFootprintWorkspace)
	if ws == nil {
		ws = &upwindFootprintWorkspace{}
	}
	ws.reset(n)
	return ws
}

func releaseUpwindWorkspace(ws *upwindFootprintWorkspace) {
	upwindWorkspacePool.Put(ws)
}

func (ws *upwindFootprintWorkspace) reset(n int) {
	if len(ws.accGen) < n {
		// Freshly allocated arrays are zeroed, and generation stamps start at 1,
		// so no explicit clear is needed here.
		ws.accGen = make([]int32, n)
		ws.accSlot = make([]int32, n)
		ws.nextGen = make([]int32, n)
		ws.nextSlot = make([]int32, n)
		ws.gen = 0
		ws.dgen = 0
	}
	const genWrapLimit = 1 << 30
	if ws.gen >= genWrapLimit || ws.dgen >= genWrapLimit {
		for i := range ws.accGen {
			ws.accGen[i] = 0
			ws.nextGen[i] = 0
		}
		ws.gen = 0
		ws.dgen = 0
	}
}

// edgeDirs returns the cached per-edge upwind unit directions for the given
// mesh, building the table on first use. The table only depends on geometry, so
// it survives across seasons and call sites within a run.
func (ws *upwindFootprintWorkspace) edgeDirs(vertices []Vector3D, adj *FlatAdjacency) []Vector3D {
	if len(vertices) == 0 || adj == nil {
		return nil
	}
	if ws.dirsAdj == adj && ws.dirsVerts == &vertices[0] && ws.dirsCount == len(vertices) &&
		len(ws.dirs) == len(adj.Neighbors) {
		return ws.dirs
	}
	dirs := make([]Vector3D, len(adj.Neighbors))
	for i := 0; i+1 < len(adj.Offsets); i++ {
		if i >= len(vertices) {
			break
		}
		start := adj.Offsets[i]
		end := adj.Offsets[i+1]
		for e := start; e < end; e++ {
			k := adj.Neighbors[e]
			if k < 0 || k >= len(vertices) {
				continue
			}
			dirs[e] = Normalize(Sub(vertices[i], vertices[k]))
		}
	}
	ws.dirs = dirs
	ws.dirsAdj = adj
	ws.dirsVerts = &vertices[0]
	ws.dirsCount = len(vertices)
	return dirs
}

// weightedUpwindDonorsInto is the allocation-free form of
// computeWeightedUpwindDonors. It appends into the workspace scratch buffers
// and uses the cached edge-direction table.
func (ws *upwindFootprintWorkspace) weightedUpwindDonorsInto(
	i int,
	vertices []Vector3D,
	adj *FlatAdjacency,
	wind []Vector3D,
	minUpwind float64,
	dirs []Vector3D,
) ([]int32, []float64) {
	donors := ws.donorBuf[:0]
	weights := ws.weightBuf[:0]
	if i < 0 || i >= len(vertices) || i >= len(wind) || i+1 >= len(adj.Offsets) {
		return nil, nil
	}
	windVec := wind[i]
	windSpeed := Length(windVec)
	if windSpeed < 1e-9 {
		return nil, nil
	}
	windDir := Scale(windVec, 1.0/windSpeed)

	weightSum := 0.0
	start := adj.Offsets[i]
	end := adj.Offsets[i+1]
	for e := start; e < end; e++ {
		k := adj.Neighbors[e]
		if k < 0 || k >= len(vertices) {
			continue
		}
		var fromNeighbor Vector3D
		if dirs != nil {
			fromNeighbor = dirs[e]
		} else {
			fromNeighbor = Normalize(Sub(vertices[i], vertices[k]))
		}
		upwind := Dot(windDir, fromNeighbor)
		if upwind <= minUpwind {
			continue
		}
		weight := upwind * upwind
		donors = append(donors, int32(k))
		weights = append(weights, weight)
		weightSum += weight
	}
	ws.donorBuf = donors
	ws.weightBuf = weights
	if weightSum <= 1e-9 {
		return nil, nil
	}
	for idx := range weights {
		weights[idx] /= weightSum
	}
	return donors, weights
}
