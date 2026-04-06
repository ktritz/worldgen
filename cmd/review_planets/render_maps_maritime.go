package main

import (
	"image"
	"image/color"
	"math"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

func renderCoastalPortSuitabilityMap(
	sites []terrain.Vector3D,
	elevation []float64,
	index *terrain.SpatialIndex,
	result *climgen.CoastalPortResult,
	network *climgen.SettlementNetworkResult,
	filename string,
	width, height int,
) {
	if result == nil || result.Diagnostics == nil || index == nil {
		return
	}
	portCells := make(map[int]struct{}, len(result.MajorPorts))
	for _, nodeIdx := range result.MajorPorts {
		if network != nil && nodeIdx >= 0 && nodeIdx < len(network.Nodes) {
			portCells[coastalPortTerminalCell(result, network, nodeIdx)] = struct{}{}
		}
	}
	for _, nodeIdx := range result.MajorDeepwaterPorts {
		if network != nil && nodeIdx >= 0 && nodeIdx < len(network.Nodes) {
			portCells[coastalDeepwaterTerminalCell(result, network, nodeIdx)] = struct{}{}
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			base := terrain.HypsometricColor(elevation[cellIdx])
			suit := result.Diagnostics.PortSuitability[cellIdx]
			deepwater := 0.0
			if cellIdx < len(result.Diagnostics.DeepwaterSuitability) {
				deepwater = result.Diagnostics.DeepwaterSuitability[cellIdx]
			}
			stopover := result.Diagnostics.StopoverValue[cellIdx]
			out := blendReviewColor(base, color.RGBA{74, 138, 198, 255}, 0.76*suit)
			out = blendReviewColor(out, color.RGBA{34, 74, 146, 255}, 0.62*deepwater)
			out = blendReviewColor(out, color.RGBA{232, 212, 144, 255}, 0.28*stopover)
			if _, ok := portCells[cellIdx]; ok {
				out = blendReviewColor(out, color.RGBA{248, 236, 176, 255}, 0.82)
			}
			img.Set(px, py, out)
		}
	}
	saveMapPNG(filename, img, "coastal port map")
}

func renderCoastalTradeMap(
	sites []terrain.Vector3D,
	elevation []float64,
	index *terrain.SpatialIndex,
	result *climgen.CoastalTradeResult,
	network *climgen.SettlementNetworkResult,
	filename string,
	width, height int,
) {
	if result == nil || result.Diagnostics == nil || index == nil {
		return
	}
	cellTier := make(map[int]climgen.CoastalTradeCorridorTier)
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
		if network != nil && nodeIdx >= 0 && nodeIdx < len(network.Nodes) {
			portCells[coastalPortTerminalCellFromTrade(result, network, nodeIdx)] = struct{}{}
		}
	}
	stopoverCells := make(map[int]struct{}, len(result.Stopovers))
	for _, stop := range result.Stopovers {
		stopoverCells[stop.CellIndex] = struct{}{}
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			out := terrain.HypsometricColor(elevation[cellIdx])
			if tier, ok := cellTier[cellIdx]; ok {
				alpha := 0.20 + 0.28*math.Min(cellFlow[cellIdx]/0.4, 1.0)
				out = blendReviewColor(out, coastalTradeTierColor(tier), alpha)
			}
			if _, ok := portCells[cellIdx]; ok {
				out = blendReviewColor(out, color.RGBA{248, 236, 176, 255}, 0.82)
			} else if _, ok := stopoverCells[cellIdx]; ok {
				out = blendReviewColor(out, color.RGBA{214, 238, 176, 255}, 0.66)
			}
			img.Set(px, py, out)
		}
	}
	saveMapPNG(filename, img, "coastal trade map")
}

func renderOceanTradeMap(
	sites []terrain.Vector3D,
	elevation []float64,
	index *terrain.SpatialIndex,
	result *climgen.OceanTradeResult,
	network *climgen.SettlementNetworkResult,
	filename string,
	width, height int,
) {
	if result == nil || result.Diagnostics == nil || index == nil {
		return
	}
	cellTier := make(map[int]climgen.OceanTradeCorridorTier)
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
		if network != nil && nodeIdx >= 0 && nodeIdx < len(network.Nodes) {
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
				alpha := 0.20 + 0.32*math.Min(cellFlow[cellIdx]/0.35, 1.0)
				out = blendReviewColor(out, oceanTradeTierColor(tier), alpha)
			}
			if _, ok := portCells[cellIdx]; ok {
				out = blendReviewColor(out, color.RGBA{240, 238, 190, 255}, 0.84)
			}
			img.Set(px, py, out)
		}
	}
	saveMapPNG(filename, img, "ocean trade map")
}

func coastalPortTerminalCell(result *climgen.CoastalPortResult, network *climgen.SettlementNetworkResult, nodeIdx int) int {
	if result != nil && result.Diagnostics != nil && nodeIdx >= 0 && nodeIdx < len(result.Diagnostics.NodeTerminalCell) && result.Diagnostics.NodeTerminalCell[nodeIdx] >= 0 {
		return result.Diagnostics.NodeTerminalCell[nodeIdx]
	}
	if network != nil && nodeIdx >= 0 && nodeIdx < len(network.Nodes) {
		return network.Nodes[nodeIdx].CellIndex
	}
	return -1
}

func coastalDeepwaterTerminalCell(result *climgen.CoastalPortResult, network *climgen.SettlementNetworkResult, nodeIdx int) int {
	if result != nil && result.Diagnostics != nil && nodeIdx >= 0 && nodeIdx < len(result.Diagnostics.NodeDeepwaterTermCell) && result.Diagnostics.NodeDeepwaterTermCell[nodeIdx] >= 0 {
		return result.Diagnostics.NodeDeepwaterTermCell[nodeIdx]
	}
	return coastalPortTerminalCell(result, network, nodeIdx)
}

func coastalPortTerminalCellFromTrade(result *climgen.CoastalTradeResult, network *climgen.SettlementNetworkResult, nodeIdx int) int {
	for _, corridor := range result.Corridors {
		if corridor.FromNode == nodeIdx && len(corridor.CellPath) > 0 {
			return corridor.CellPath[0]
		}
		if corridor.ToNode == nodeIdx && len(corridor.CellPath) > 0 {
			return corridor.CellPath[len(corridor.CellPath)-1]
		}
	}
	if network != nil && nodeIdx >= 0 && nodeIdx < len(network.Nodes) {
		return network.Nodes[nodeIdx].CellIndex
	}
	return -1
}

func coastalTradeTierColor(tier climgen.CoastalTradeCorridorTier) color.RGBA {
	switch tier {
	case climgen.CoastalTradeCorridorPrimary:
		return color.RGBA{52, 116, 190, 255}
	case climgen.CoastalTradeCorridorRegional:
		return color.RGBA{88, 156, 214, 255}
	default:
		return color.RGBA{138, 198, 232, 255}
	}
}

func oceanTradeTierColor(tier climgen.OceanTradeCorridorTier) color.RGBA {
	switch tier {
	case climgen.OceanTradeCorridorPrimary:
		return color.RGBA{34, 70, 164, 255}
	case climgen.OceanTradeCorridorRegional:
		return color.RGBA{54, 108, 198, 255}
	default:
		return color.RGBA{92, 150, 226, 255}
	}
}
