package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

func renderVegetationMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	result *climgen.VegetationResult,
	filename string,
	width, height int,
) {
	if result == nil || index == nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(result.Types) {
				img.Set(px, py, climgen.VegetationColor(result.Types[cellIdx]))
			}
		}
	}
	saveMapPNG(filename, img, "vegetation map")
}

func renderSoilMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	result *climgen.SoilResult,
	filename string,
	width, height int,
) {
	if result == nil || index == nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(result.Types) {
				img.Set(px, py, climgen.SoilColor(result.Types[cellIdx]))
			}
		}
	}
	saveMapPNG(filename, img, "soil map")
}

func renderAgricultureMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	result *climgen.AgricultureResult,
	filename string,
	width, height int,
) {
	if result == nil || index == nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(result.Types) {
				img.Set(px, py, climgen.AgricultureColor(result.Types[cellIdx]))
			}
		}
	}
	saveMapPNG(filename, img, "agriculture map")
}

func renderWildlifeMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	result *climgen.WildlifeResult,
	filename string,
	width, height int,
) {
	if result == nil || index == nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(result.Types) {
				img.Set(px, py, climgen.WildlifeColor(result.Types[cellIdx]))
			}
		}
	}
	saveMapPNG(filename, img, "wildlife map")
}

func renderCoastalResourceMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	result *climgen.CoastalResourceResult,
	filename string,
	width, height int,
) {
	if result == nil || index == nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(result.Types) {
				img.Set(px, py, climgen.CoastalResourceColor(result.Types[cellIdx]))
			}
		}
	}
	saveMapPNG(filename, img, "coastal resource map")
}

func renderCoastalUpwellingMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	result *climgen.CoastalResourceResult,
	filename string,
	width, height int,
) {
	if result == nil || result.Diagnostics == nil || index == nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(result.Types) {
				if result.Types[cellIdx] == climgen.CoastalResourceOcean {
					img.Set(px, py, climgen.CoastalResourceColor(climgen.CoastalResourceOcean))
					continue
				}
				v := climgen.Clamp(result.Diagnostics.UpwellingPotential[cellIdx], 0, 1)
				r := uint8(232 + v*(46-232))
				g := uint8(227 + v*(116-227))
				b := uint8(212 + v*(173-212))
				img.Set(px, py, color.RGBA{R: r, G: g, B: b, A: 255})
			}
		}
	}
	saveMapPNG(filename, img, "coastal upwelling map")
}

func renderWaterResourceMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	result *climgen.WaterResourceResult,
	filename string,
	width, height int,
) {
	if result == nil || index == nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(result.Types) {
				img.Set(px, py, climgen.WaterResourceColor(result.Types[cellIdx]))
			}
		}
	}
	saveMapPNG(filename, img, "water resource map")
}

func renderResourceMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	result *climgen.ResourceResult,
	filename string,
	width, height int,
) {
	if result == nil || index == nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(result.Types) {
				img.Set(px, py, climgen.ResourceColor(result.Types[cellIdx]))
			}
		}
	}
	saveMapPNG(filename, img, "resource map")
}

func renderResourcePotentialMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	result *climgen.ResourceResult,
	resource climgen.ResourceType,
	filename string,
	width, height int,
) {
	if result == nil || result.Diagnostics == nil || index == nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(result.Types) {
				if result.Types[cellIdx] == climgen.ResourceOcean {
					img.Set(px, py, climgen.ResourceColor(climgen.ResourceOcean))
					continue
				}
				potential := climgen.ResourcePotential(result.Diagnostics, resource, cellIdx)
				img.Set(px, py, climgen.ResourcePotentialColor(resource, potential))
			}
		}
	}
	saveMapPNG(filename, img, "resource potential map")
}

func renderSettlementMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	result *climgen.SettlementResult,
	filename string,
	width, height int,
) {
	if result == nil || index == nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(result.Classes) {
				img.Set(px, py, climgen.SettlementColor(result.Classes[cellIdx]))
			}
		}
	}
	saveMapPNG(filename, img, "settlement map")
}

func renderPopulationMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	result *climgen.PopulationResult,
	filename string,
	width, height int,
) {
	if result == nil || index == nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(result.Classes) {
				img.Set(px, py, climgen.PopulationColor(result.Classes[cellIdx]))
			}
		}
	}
	saveMapPNG(filename, img, "population map")
}

func renderSettlementNetworkMap(
	sites []terrain.Vector3D,
	elevation []float64,
	index *terrain.SpatialIndex,
	result *climgen.SettlementNetworkResult,
	filename string,
	width, height int,
) {
	if result == nil || index == nil {
		return
	}
	pathCells := make(map[int]float64)
	for _, link := range result.Links {
		for _, cellIdx := range link.Path {
			if link.TravelCost > pathCells[cellIdx] {
				pathCells[cellIdx] = link.TravelCost
			}
		}
	}
	nodeByCell := make(map[int]climgen.SettlementNodeKind, len(result.Nodes))
	for _, node := range result.Nodes {
		nodeByCell[node.CellIndex] = node.Kind
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			base := terrain.HypsometricColor(elevation[cellIdx])
			out := base
			if pathCost, ok := pathCells[cellIdx]; ok {
				blend := 0.18 + 0.22*math.Min(pathCost/12.0, 1.0)
				out = blendReviewColor(out, color.RGBA{112, 92, 70, 255}, blend)
			}
			if kind, ok := nodeByCell[cellIdx]; ok {
				out = blendReviewColor(out, climgen.SettlementNodeColor(kind), 0.72)
			}
			img.Set(px, py, out)
		}
	}
	saveMapPNG(filename, img, "settlement network map")
}

func renderSettlementPreferenceMap(
	sites []terrain.Vector3D,
	index *terrain.SpatialIndex,
	results []*climgen.SettlementPreferenceResult,
	filename string,
	width, height int,
) {
	if len(results) == 0 || index == nil {
		return
	}
	best := climgen.DominantSettlementPreference(results)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			if cellIdx >= 0 && cellIdx < len(best) {
				profile := results[best[cellIdx]].Profile.Name
				img.Set(px, py, climgen.SettlementPreferenceColor(profile))
			}
		}
	}
	saveMapPNG(filename, img, "settlement preference map")
}

func saveMapPNG(filename string, img image.Image, label string) {
	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("  create %s %s: %v\n", label, filename, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Printf("  encode %s %s: %v\n", label, filename, err)
		return
	}
	fmt.Printf("  Saved %s\n", filename)
}

func blendReviewColor(base, over color.RGBA, alpha float64) color.RGBA {
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	return color.RGBA{
		R: uint8(float64(base.R)*(1-alpha) + float64(over.R)*alpha),
		G: uint8(float64(base.G)*(1-alpha) + float64(over.G)*alpha),
		B: uint8(float64(base.B)*(1-alpha) + float64(over.B)*alpha),
		A: 255,
	}
}
