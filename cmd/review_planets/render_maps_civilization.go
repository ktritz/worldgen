package main

import (
	"image"
	"image/color"
	"math"

	"worldgen/climgen"
	"worldgen/landgen/terrain"
)

func renderLandRouteRiskMap(
	sites []terrain.Vector3D,
	elevation []float64,
	index *terrain.SpatialIndex,
	result *climgen.LandRouteResult,
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
			risk := result.Diagnostics.RouteRisk[cellIdx]
			support := result.Diagnostics.WaystationSuitability[cellIdx]
			out := blendReviewColor(base, color.RGBA{208, 66, 52, 255}, 0.66*risk)
			out = blendReviewColor(out, color.RGBA{116, 180, 92, 255}, 0.34*support)
			img.Set(px, py, out)
		}
	}
	saveMapPNG(filename, img, "land route risk map")
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

func renderSettlementRegionMap(
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
	nodeRegion := map[int]int{}
	centerCells := map[int]struct{}{}
	for _, region := range result.Regions {
		for _, nodeIdx := range region.NodeIndices {
			nodeRegion[result.Nodes[nodeIdx].CellIndex] = region.ID
		}
		centerCells[result.Nodes[region.CenterNode].CellIndex] = struct{}{}
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			out := terrain.HypsometricColor(elevation[cellIdx])
			if regionID, ok := nodeRegion[cellIdx]; ok {
				out = blendReviewColor(out, settlementRegionColor(regionID), 0.62)
			}
			if _, ok := centerCells[cellIdx]; ok {
				out = blendReviewColor(out, color.RGBA{248, 236, 176, 255}, 0.72)
			}
			img.Set(px, py, out)
		}
	}
	saveMapPNG(filename, img, "settlement region map")
}

func renderProtoCivilizationMap(
	sites []terrain.Vector3D,
	elevation []float64,
	index *terrain.SpatialIndex,
	result *climgen.ProtoCivilizationResult,
	network *climgen.SettlementNetworkResult,
	filename string,
	width, height int,
) {
	if result == nil || result.Diagnostics == nil || network == nil || index == nil {
		return
	}
	centerCells := make(map[int]struct{}, len(result.Civilizations))
	for _, civ := range result.Civilizations {
		if civ.CenterNode >= 0 && civ.CenterNode < len(network.Nodes) {
			centerCells[network.Nodes[civ.CenterNode].CellIndex] = struct{}{}
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			out := terrain.HypsometricColor(elevation[cellIdx])
			if cellIdx >= 0 && cellIdx < len(result.Diagnostics.CivilizationByCell) {
				if civIdx := result.Diagnostics.CivilizationByCell[cellIdx]; civIdx >= 0 {
					out = blendReviewColor(out, climgen.ProtoCivilizationColor(civIdx), 0.58)
				}
			}
			if _, ok := centerCells[cellIdx]; ok {
				out = blendReviewColor(out, color.RGBA{248, 236, 176, 255}, 0.75)
			}
			img.Set(px, py, out)
		}
	}
	saveMapPNG(filename, img, "proto-civilization map")
}

func renderTradeNetworkMap(
	sites []terrain.Vector3D,
	elevation []float64,
	index *terrain.SpatialIndex,
	result *climgen.TradeNetworkResult,
	network *climgen.SettlementNetworkResult,
	filename string,
	width, height int,
) {
	if result == nil || result.Diagnostics == nil || network == nil || index == nil {
		return
	}
	cellTier := make(map[int]climgen.TradeCorridorTier)
	cellFlow := make(map[int]float64)
	for _, corridor := range result.Corridors {
		for _, cellIdx := range corridor.CellPath {
			if flow, ok := cellFlow[cellIdx]; !ok || corridor.Flow > flow {
				cellFlow[cellIdx] = corridor.Flow
				cellTier[cellIdx] = corridor.Tier
			}
		}
	}
	hubCells := make(map[int]struct{}, len(result.MajorHubs))
	for _, nodeIdx := range result.MajorHubs {
		if nodeIdx >= 0 && nodeIdx < len(network.Nodes) {
			hubCells[network.Nodes[nodeIdx].CellIndex] = struct{}{}
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
				alpha := 0.18 + 0.28*math.Min(cellFlow[cellIdx]/0.8, 1.0)
				out = blendReviewColor(out, climgen.TradeCorridorTierColor(tier), alpha)
			}
			if _, ok := hubCells[cellIdx]; ok {
				out = blendReviewColor(out, color.RGBA{242, 226, 164, 255}, 0.78)
			}
			img.Set(px, py, out)
		}
	}
	saveMapPNG(filename, img, "trade network map")
}

func renderPolitySphereMap(
	sites []terrain.Vector3D,
	elevation []float64,
	index *terrain.SpatialIndex,
	result *climgen.PolitySphereResult,
	network *climgen.SettlementNetworkResult,
	filename string,
	width, height int,
) {
	if result == nil || result.Diagnostics == nil || network == nil || index == nil {
		return
	}
	capitalCells := make(map[int]struct{}, len(result.Spheres))
	for _, sphere := range result.Spheres {
		if sphere.CapitalNode >= 0 && sphere.CapitalNode < len(network.Nodes) {
			capitalCells[network.Nodes[sphere.CapitalNode].CellIndex] = struct{}{}
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		lat := 90 - float64(py)/float64(height)*180
		for px := 0; px < width; px++ {
			lon := float64(px)/float64(width)*360 - 180
			cellIdx := index.FindNearest(lat, lon, sites)
			out := terrain.HypsometricColor(elevation[cellIdx])
			if cellIdx >= 0 && cellIdx < len(result.Diagnostics.PolityByCell) {
				if polityIdx := result.Diagnostics.PolityByCell[cellIdx]; polityIdx >= 0 {
					out = blendReviewColor(out, climgen.PolitySphereColor(polityIdx), 0.56)
				}
			}
			if _, ok := capitalCells[cellIdx]; ok {
				out = blendReviewColor(out, color.RGBA{246, 232, 168, 255}, 0.8)
			}
			img.Set(px, py, out)
		}
	}
	saveMapPNG(filename, img, "polity sphere map")
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

