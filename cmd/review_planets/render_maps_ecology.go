package main

import (
	"image"
	"image/color"

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

