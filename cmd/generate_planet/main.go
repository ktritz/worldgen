// Planet generation tool using Red Blob Games algorithm
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"worldgen/icosphere"
	"worldgen/landgen/terrain"
)

const outputDir = "output/maps"

func main() {
	fmt.Println("=== Planet Generation Tool ===")

	// Ensure output directory exists
	os.MkdirAll(outputDir, 0755)

	// Generate base mesh once
	level := 7
	fmt.Printf("Generating icosphere level %d...\n", level)
	vertices, faces := icosphere.CreateIcosphere(level)
	fmt.Printf("  %d vertices, %d faces\n", len(vertices), len(faces))

	fmt.Println("Generating Voronoi cells...")
	_, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)

	// Convert to terrain types
	sites := make([]terrain.Vector3D, len(vertices))
	for i, v := range vertices {
		sites[i] = terrain.Vector3D{X: v.X, Y: v.Y, Z: v.Z}
	}
	cells := make([]terrain.VoronoiCell, len(voronoiCells))
	for i, cell := range voronoiCells {
		cells[i] = terrain.VoronoiCell{
			SiteIndex:           int32(cell.SiteIndex),
			NeighborSiteIndices: make([]int32, len(cell.NeighborSiteIndices)),
		}
		for j, idx := range cell.NeighborSiteIndices {
			cells[i].NeighborSiteIndices[j] = int32(idx)
		}
	}

	// Build spatial index once
	fmt.Println("Building spatial index...")
	index := terrain.BuildSpatialIndex(sites)

	// Generate planets with different seeds for variety
	// Seeds selected to show different continent configurations:
	// - Seed 42: typical single mega-continent
	// - Seed 84: high convergent rate (~40%)
	// - Seed 8: two continents
	numPlates := 12
	testSeeds := []int64{42, 84, 8}
	landFrac := 0.29

	for _, seed := range testSeeds {
		fmt.Printf("\n========================================\n")
		fmt.Printf("Generating planet with seed %d, %.0f%% land, %d plates\n", seed, landFrac*100, numPlates)
		fmt.Printf("========================================\n")

		elevation, isLand := terrain.GeneratePlanetElevation(sites, cells, numPlates, seed, landFrac)

		prefix := fmt.Sprintf("planet_seed%d", seed)

		// Render elevation map
		terrain.RenderElevationMap(sites, elevation, index, filepath.Join(outputDir, prefix+"_elevation.png"))

		// Render land/ocean map
		terrain.RenderLandOceanMap(sites, elevation, isLand, index, filepath.Join(outputDir, prefix+"_landocean.png"))

		// Render orthographic (globe) views from multiple angles - no polar distortion
		terrain.RenderOrthoView(sites, elevation, index, 0, 0, filepath.Join(outputDir, prefix+"_globe_front.png"))
		terrain.RenderOrthoView(sites, elevation, index, 0, 90, filepath.Join(outputDir, prefix+"_globe_side.png"))
		terrain.RenderOrthoView(sites, elevation, index, -45, 0, filepath.Join(outputDir, prefix+"_globe_south.png"))

		// Generate hypsometry histogram
		terrain.RenderHypsometry(elevation, filepath.Join(outputDir, prefix+"_hypsometry.png"))

		// Print stats
		terrain.PrintStats(elevation, isLand)
	}

	fmt.Println("\n=== Generation complete ===")
	fmt.Printf("Output saved to %s/\n", outputDir)
}
