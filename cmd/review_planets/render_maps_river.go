package main

import (
	"image"
	"image/color"
	"math"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

func renderRiverNavigabilityMap(
	sites []terrain.Vector3D,
	elevation []float64,
	index *terrain.SpatialIndex,
	result *climgen.RiverRouteResult,
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
			base := terrain.HypsometricColor(elevation[cellIdx])
			nav := result.Diagnostics.Navigability[cellIdx]
			transfer := result.Diagnostics.TransferSupport[cellIdx]
			out := blendReviewColor(base, color.RGBA{78, 128, 198, 255}, 0.72*nav)
			out = blendReviewColor(out, color.RGBA{236, 214, 148, 255}, 0.18*transfer)
			img.Set(px, py, out)
		}
	}
	saveMapPNG(filename, img, "river navigability map")
}

func renderRiverTradeMap(
	sites []terrain.Vector3D,
	elevation []float64,
	index *terrain.SpatialIndex,
	result *climgen.RiverTradeResult,
	network *climgen.SettlementNetworkResult,
	filename string,
	width, height int,
) {
	if result == nil || result.Diagnostics == nil || network == nil || index == nil {
		return
	}
	cellTier := make(map[int]climgen.RiverTradeCorridorTier)
	cellFlow := make(map[int]float64)
	for _, corridor := range result.Corridors {
		for _, cellIdx := range corridor.CellPath {
			if flow, ok := cellFlow[cellIdx]; !ok || corridor.Flow > flow {
				cellFlow[cellIdx] = corridor.Flow
				cellTier[cellIdx] = corridor.Tier
			}
		}
	}
	portCells := make(map[int]struct{}, len(result.MajorPorts))
	for _, nodeIdx := range result.MajorPorts {
		if nodeIdx >= 0 && nodeIdx < len(network.Nodes) {
			portCells[network.Nodes[nodeIdx].CellIndex] = struct{}{}
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			out := terrain.HypsometricColor(elevation[cellIdx])
			if tier, ok := cellTier[cellIdx]; ok {
				alpha := 0.20 + 0.28*math.Min(cellFlow[cellIdx]/0.8, 1.0)
				out = blendReviewColor(out, riverTradeTierColor(tier), alpha)
			}
			if _, ok := portCells[cellIdx]; ok {
				out = blendReviewColor(out, color.RGBA{242, 226, 164, 255}, 0.78)
			}
			img.Set(px, py, out)
		}
	}
	saveMapPNG(filename, img, "river trade map")
}

func riverTradeTierColor(tier climgen.RiverTradeCorridorTier) color.RGBA {
	switch tier {
	case climgen.RiverTradeCorridorPrimary:
		return color.RGBA{54, 112, 188, 255}
	case climgen.RiverTradeCorridorRegional:
		return color.RGBA{88, 150, 208, 255}
	default:
		return color.RGBA{132, 186, 224, 255}
	}
}

