package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"worldgen/icosphere"
	"worldgen/landgen/terrain"
)

func main() {
	level := flag.Int("level", 6, "icosphere subdivision level")
	numPlates := flag.Int("plates", 12, "number of tectonic plates")
	landFrac := flag.Float64("land", 0.29, "target land fraction")
	seed := flag.Int64("seed", 55, "world seed")
	center := flag.Int("center", -1, "center cell index (-1 = auto-select)")
	neighbors := flag.Int("neighbors", 2, "maximum neighboring cells to refine after the center cell")
	radius := flag.Int("radius", 1, "connected-cell BFS radius")
	resolution := flag.Int("resolution", 320, "local patch resolution")
	outDir := flag.String("out", "output/local_cells", "output directory")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output dir: %v\n", err)
		os.Exit(1)
	}
	store := terrain.NewLocalRefinementStore(filepath.Join(*outDir, "state"), *seed)

	vertices, faces := icosphere.CreateIcosphere(*level)
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)

	sites := make([]terrain.Vector3D, len(vertices))
	for i, v := range vertices {
		sites[i] = terrain.Vector3D{X: v.X, Y: v.Y, Z: v.Z}
	}
	cells := make([]terrain.VoronoiCell, len(voronoiCells))
	for i, cell := range voronoiCells {
		cells[i] = terrain.VoronoiCell{
			SiteIndex:           int32(cell.SiteIndex),
			NeighborSiteIndices: append([]int32(nil), cell.NeighborSiteIndices...),
		}
	}

	elevation, _, diagnostics := terrain.GeneratePlanetElevationWithDiagnostics(sites, cells, *numPlates, *seed, *landFrac)
	if diagnostics.Hydrology.Scaffold == nil || diagnostics.Hydrology.TerrainRefinement == nil {
		fmt.Fprintln(os.Stderr, "missing hydrology or terrain refinement scaffolds")
		os.Exit(1)
	}

	centerCell := *center
	if centerCell < 0 {
		centerCell = chooseCenterCell(diagnostics.Hydrology.Scaffold, elevation)
	}
	settings := terrain.DefaultLocalRefinementSettings()
	settings.Radius = *radius
	settings.Resolution = *resolution
	manager := &terrain.LocalRefinementManager{
		Sites:      sites,
		Cells:      cells,
		Elevation:  elevation,
		Hydrology:  diagnostics.Hydrology.Scaffold,
		Refinement: diagnostics.Hydrology.TerrainRefinement,
		Seed:       *seed,
		Settings:   settings,
		Store:      store,
	}

	result, err := manager.RefineCellNeighborhood(centerCell, *neighbors)
	if err != nil {
		fmt.Fprintf(os.Stderr, "refine local cells: %v\n", err)
		os.Exit(1)
	}
	patch := result.CenterPatch

	filename := filepath.Join(*outDir, fmt.Sprintf("seed_%d_center_%d_patch.png", *seed, centerCell))
	if err := terrain.RenderLocalRefinementPatch(patch, filename); err != nil {
		fmt.Fprintf(os.Stderr, "render patch: %v\n", err)
		os.Exit(1)
	}
	hydroFilename := filepath.Join(*outDir, fmt.Sprintf("seed_%d_center_%d_hydrology.png", *seed, centerCell))
	if err := terrain.RenderLocalHydrologyPatch(patch, hydroFilename); err != nil {
		fmt.Fprintf(os.Stderr, "render hydrology patch: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("local patch saved: %s\n", filename)
	fmt.Printf("local hydrology saved: %s\n", hydroFilename)
	fmt.Printf("center=%d selected=%d support=%d boundary mismatch mean/max=%.2f/%.2f m samples=%d channel carve mean/max=%.2f/%.2f edges=%d lake/channel coverage=%.2f%%/%.2f%%\n",
		patch.Debug.CenterCell,
		len(patch.Debug.SelectedCells),
		len(patch.Debug.SupportCells),
		patch.Debug.MeanBoundaryMismatch,
		patch.Debug.MaxBoundaryMismatch,
		patch.Debug.NumBoundarySamples,
		patch.Debug.MeanChannelCarve,
		patch.Debug.MaxChannelCarve,
		patch.Debug.NumChannelEdges,
		patch.Debug.LakeCoveragePct,
		patch.Debug.ChannelCoveragePct,
	)
	for _, idx := range patch.Debug.SelectedCells {
		class := diagnostics.Hydrology.Scaffold.CellClass[idx]
		outlet := diagnostics.Hydrology.Scaffold.OutletMode[idx]
		maxOut := diagnostics.Hydrology.Scaffold.MaxOutflows[idx]
		fmt.Printf("  cell %d: class=%s outlet=%s maxOut=%d runoff=%.3f accum=%.2f\n",
			idx, class, outlet, maxOut,
			diagnostics.Hydrology.Scaffold.Runoff[idx],
			diagnostics.Hydrology.Scaffold.Accumulation[idx],
		)
	}
	for _, crossing := range patch.Debug.BoundaryCrossings {
		fmt.Printf("  refined crossing: side=%s bearing=%.1f kind=%s width=%d strength=%.2f\n",
			crossing.Side, crossing.BearingDeg, crossing.Kind, crossing.WidthPx, crossing.Strength,
		)
	}
	for _, crossing := range patch.Debug.CellBoundaryCrossings {
		fmt.Printf("  center boundary crossing: neighbor=%d bearing=%.1f offset=%.4f kind=%s width=%d strength=%.2f\n",
			crossing.Neighbor, crossing.BearingDeg, crossing.Offset, crossing.Kind, crossing.WidthPx, crossing.Strength,
		)
	}
	if queued, err := manager.QueuedDirtyNeighbors(centerCell); err == nil && len(queued) > 0 {
		fmt.Printf("queued stale neighbors after center refine: %v\n", queued)
	}

	if len(result.Neighbors) == 0 {
		return
	}
	artifactFile := store.ArtifactPath(centerCell)
	fmt.Printf("artifact saved: %s\n", artifactFile)
	for _, neighbor := range result.Neighbors {
		adjacentCenter := neighbor.Cell
		adjacentPatch := neighbor.Patch
		merge := neighbor.Merge
		contract := neighbor.Contract

		adjFilename := filepath.Join(*outDir, fmt.Sprintf("seed_%d_center_%d_patch.png", *seed, adjacentCenter))
		if err := terrain.RenderLocalRefinementPatch(adjacentPatch, adjFilename); err != nil {
			fmt.Fprintf(os.Stderr, "render adjacent patch: %v\n", err)
			os.Exit(1)
		}
		adjHydroFilename := filepath.Join(*outDir, fmt.Sprintf("seed_%d_center_%d_hydrology.png", *seed, adjacentCenter))
		if err := terrain.RenderLocalHydrologyPatch(adjacentPatch, adjHydroFilename); err != nil {
			fmt.Fprintf(os.Stderr, "render adjacent hydrology patch: %v\n", err)
			os.Exit(1)
		}

		shared := intersectCells(patch, adjacentPatch)
		meanOverlap, maxOverlap, overlapSamples := terrain.CompareLocalRefinementOverlap(patch, adjacentPatch, sites, shared)
		fmt.Printf("adjacent local patch saved: %s\n", adjFilename)
		fmt.Printf("adjacent local hydrology saved: %s\n", adjHydroFilename)
		fmt.Printf("adjacent center=%d selected=%d support=%d boundary mismatch mean/max=%.2f/%.2f m samples=%d channel carve mean/max=%.2f/%.2f edges=%d lake/channel coverage=%.2f%%/%.2f%%\n",
			adjacentPatch.Debug.CenterCell,
			len(adjacentPatch.Debug.SelectedCells),
			len(adjacentPatch.Debug.SupportCells),
			adjacentPatch.Debug.MeanBoundaryMismatch,
			adjacentPatch.Debug.MaxBoundaryMismatch,
			adjacentPatch.Debug.NumBoundarySamples,
			adjacentPatch.Debug.MeanChannelCarve,
			adjacentPatch.Debug.MaxChannelCarve,
			adjacentPatch.Debug.NumChannelEdges,
			adjacentPatch.Debug.LakeCoveragePct,
			adjacentPatch.Debug.ChannelCoveragePct,
		)
		fmt.Printf("overlap cells=%d elevation mismatch mean/max=%.2f/%.2f m\n",
			overlapSamples, meanOverlap, maxOverlap,
		)
		for _, crossing := range adjacentPatch.Debug.BoundaryCrossings {
			fmt.Printf("  adjacent refined crossing: side=%s bearing=%.1f kind=%s width=%d strength=%.2f\n",
				crossing.Side, crossing.BearingDeg, crossing.Kind, crossing.WidthPx, crossing.Strength,
			)
		}
		for _, crossing := range adjacentPatch.Debug.CellBoundaryCrossings {
			fmt.Printf("  adjacent center boundary crossing: neighbor=%d bearing=%.1f offset=%.4f kind=%s width=%d strength=%.2f\n",
				crossing.Neighbor, crossing.BearingDeg, crossing.Offset, crossing.Kind, crossing.WidthPx, crossing.Strength,
			)
		}

		adjArtifactFile := store.ArtifactPath(adjacentCenter)
		if merge != nil {
			mergeFile := filepath.Join(*outDir, fmt.Sprintf("seed_%d_cells_%d_%d_merge.json", *seed, centerCell, adjacentCenter))
			if err := writeJSON(mergeFile, merge); err != nil {
				fmt.Fprintf(os.Stderr, "write merge artifact: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("merge artifact saved: %s\n", mergeFile)
			for _, crossing := range merge.Crossings {
				fmt.Printf("  canonical crossing: %d<->%d kind=%s offsetA=%.4f offsetB=%.4f width=%d strength=%.2f\n",
					crossing.CellA, crossing.CellB, crossing.Kind, crossing.OffsetA, crossing.OffsetB, crossing.WidthPx, crossing.Strength,
				)
			}
		}
		if contract != nil {
			contractFile := store.ContractPath(contract.CellA, contract.CellB)
			fmt.Printf("boundary contract saved: %s\n", contractFile)
			fmt.Printf("  contract version=%d crossings=%d dirty=%v\n", contract.Version, len(contract.Crossings), contract.DirtyCells)
		}
		fmt.Printf("adjacent artifact saved: %s\n", adjArtifactFile)
	}
}

func chooseCenterCell(scaffold *terrain.HydrologyScaffold, elevation []float64) int {
	best := -1
	bestScore := -1.0
	for i, class := range scaffold.CellClass {
		if i >= len(elevation) || elevation[i] <= 0 {
			continue
		}
		score := 0.0
		switch class {
		case "trunk":
			score = 4.0
		case "confluence":
			score = 3.5
		case "floodplain":
			score = 3.0
		case "delta":
			score = 2.5
		case "coast_outlet":
			score = 2.0
		case "headwater":
			score = 1.0
		}
		if i < len(scaffold.ChannelStrength) {
			score += scaffold.ChannelStrength[i]
		}
		if score > bestScore {
			best = i
			bestScore = score
		}
	}
	if best >= 0 {
		return best
	}
	for i, elev := range elevation {
		if elev > 0 {
			return i
		}
	}
	return 0
}

func intersectCells(a, b *terrain.LocalRefinementPatch) []int {
	seen := make(map[int]bool, len(a.Debug.SelectedCells)+len(a.Debug.SupportCells))
	for _, idx := range a.Debug.SelectedCells {
		seen[idx] = true
	}
	for _, idx := range a.Debug.SupportCells {
		seen[idx] = true
	}
	out := make([]int, 0)
	addIfShared := func(idx int) {
		if seen[idx] {
			out = append(out, idx)
			delete(seen, idx)
		}
	}
	for _, idx := range b.Debug.SelectedCells {
		addIfShared(idx)
	}
	for _, idx := range b.Debug.SupportCells {
		addIfShared(idx)
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
