package main

import (
	"fmt"
	"math"
	"worldgen/landgen/tectonics"

	"github.com/kyroy/kdtree"
)

// PlatePixelLookup provides utilities for finding plate IDs at specific pixel coordinates
type PlatePixelLookup struct {
	kdTree           *kdtree.KDTree
	voronoiCells     []tectonics.VoronoiCell
	icosphereVertices []tectonics.Vector3D
	cellAssignments  []int
	plates           []tectonics.TectonicPlate
	mapWidth         int
	mapHeight        int
	planetRadius     float64
}

// kdtreeCellSitePoint wraps a Voronoi cell site for KD-tree nearest neighbor searches
type kdtreeCellSitePoint struct {
	Coordinates tectonics.Vector3D
	CellIndex   int
}

func (p kdtreeCellSitePoint) Dimensions() int         { return p.Coordinates.Dimensions() }
func (p kdtreeCellSitePoint) Dimension(i int) float64 { return p.Coordinates.Dimension(i) }

// NewPlatePixelLookup creates a new plate pixel lookup utility
func NewPlatePixelLookup(
	plates []tectonics.TectonicPlate,
	cellAssignments []int,
	voronoiCells []tectonics.VoronoiCell,
	icosphereVertices []tectonics.Vector3D,
	mapWidth, mapHeight int,
	planetRadius float64) *PlatePixelLookup {

	// Build KD-tree for fast nearest neighbor searches
	kdPoints := make([]kdtree.Point, len(voronoiCells))
	for i, cell := range voronoiCells {
		if int(cell.SiteIndex) >= len(icosphereVertices) {
			continue
		}
		cellPos := icosphereVertices[cell.SiteIndex]
		kdPoints[i] = kdtreeCellSitePoint{
			Coordinates: cellPos,
			CellIndex:   i,
		}
	}
	kdTree := kdtree.New(kdPoints)

	return &PlatePixelLookup{
		kdTree:           kdTree,
		voronoiCells:     voronoiCells,
		icosphereVertices: icosphereVertices,
		cellAssignments:  cellAssignments,
		plates:           plates,
		mapWidth:         mapWidth,
		mapHeight:        mapHeight,
		planetRadius:     planetRadius,
	}
}

// GetPlateIDAtPixel returns the plate ID at the specified pixel coordinates
// Uses the same coordinate conversion as the map generation code
func (lookup *PlatePixelLookup) GetPlateIDAtPixel(x, y int) (int32, error) {
	if x < 0 || x >= lookup.mapWidth || y < 0 || y >= lookup.mapHeight {
		return -1, fmt.Errorf("pixel coordinates (%d, %d) are outside map bounds (%dx%d)", x, y, lookup.mapWidth, lookup.mapHeight)
	}

	// Convert pixel coordinates to spherical coordinates (equirectangular projection)
	lat, lon := pixelToSpherical(x, y, lookup.mapWidth, lookup.mapHeight)

	// Convert spherical to 3D Cartesian coordinates
	pos := sphericalToCartesian(lat, lon)

	// Find the closest Voronoi cell using KD-tree
	closestCellIdx := lookup.findClosestCellKDTree(pos)

	if closestCellIdx < 0 || closestCellIdx >= len(lookup.cellAssignments) {
		return -1, fmt.Errorf("no valid cell found for pixel coordinates (%d, %d)", x, y)
	}

	// Get the plate ID from cell assignments
	plateID := lookup.cellAssignments[closestCellIdx]
	if plateID < 0 || plateID >= len(lookup.plates) {
		return -1, fmt.Errorf("invalid plate ID %d for pixel coordinates (%d, %d)", plateID, x, y)
	}

	return int32(plateID), nil
}

// GetPlateInfoAtPixel returns detailed plate information at the specified pixel coordinates
func (lookup *PlatePixelLookup) GetPlateInfoAtPixel(x, y int) (*PlatePixelInfo, error) {
	plateID, err := lookup.GetPlateIDAtPixel(x, y)
	if err != nil {
		return nil, err
	}

	if plateID < 0 || int(plateID) >= len(lookup.plates) {
		return nil, fmt.Errorf("invalid plate ID %d", plateID)
	}

	plate := lookup.plates[plateID]

	// Convert pixel to geographic coordinates
	lat, lon := pixelToSpherical(x, y, lookup.mapWidth, lookup.mapHeight)

	return &PlatePixelInfo{
		PixelX:       x,
		PixelY:       y,
		Latitude:     lat * 180 / math.Pi,  // Convert to degrees
		Longitude:    lon * 180 / math.Pi,  // Convert to degrees
		PlateID:      plateID,
		PlateType:    plate.PlateType,
		PlateCenter:  plate.Center,
		PlateArea:    plate.Area,
		RotationAxis: plate.RotationAxis,
		RotationSpeed: plate.RotationSpeed,
	}, nil
}

// PlatePixelInfo contains detailed information about a plate at a specific pixel
type PlatePixelInfo struct {
	PixelX        int
	PixelY        int
	Latitude      float64  // In degrees
	Longitude     float64  // In degrees
	PlateID       int32
	PlateType     tectonics.PlateType
	PlateCenter   tectonics.Vector3D
	PlateArea     float64
	RotationAxis  tectonics.Vector3D
	RotationSpeed float64
}

// String returns a formatted string representation of the plate information
func (info *PlatePixelInfo) String() string {
	return fmt.Sprintf("Pixel (%d, %d) -> Lat: %.2f°, Lon: %.2f° -> Plate ID: %d (%s), Area: %.2e m², Rotation: %.4f rad/Myr",
		info.PixelX, info.PixelY, info.Latitude, info.Longitude, info.PlateID, info.PlateType, info.PlateArea, info.RotationSpeed)
}

// pixelToSpherical converts pixel coordinates to latitude/longitude using equirectangular projection
// This matches the implementation in mapgen/map_generator.go
func pixelToSpherical(x, y, width, height int) (lat, lon float64) {
	// Equirectangular projection: simple linear mapping
	// x maps to longitude [-π, π]
	// y maps to latitude [-π/2, π/2]
	
	lon = (float64(x)/float64(width))*2*math.Pi - math.Pi
	lat = (float64(y)/float64(height))*math.Pi - math.Pi/2
	
	return lat, lon
}

// sphericalToCartesian converts lat/lon to 3D unit sphere coordinates
// This matches the implementation in mapgen/map_generator.go
func sphericalToCartesian(lat, lon float64) tectonics.Vector3D {
	// Standard spherical to Cartesian conversion
	x := math.Cos(lat) * math.Cos(lon)
	y := math.Cos(lat) * math.Sin(lon)
	z := math.Sin(lat)
	
	return tectonics.Vector3D{X: x, Y: y, Z: z}
}

// findClosestCellKDTree finds the Voronoi cell closest to a given 3D position using KD-tree
// This matches the implementation in mapgen/map_generator.go
func (lookup *PlatePixelLookup) findClosestCellKDTree(pos tectonics.Vector3D) int {
	nearestNeighbors := lookup.kdTree.KNN(pos, 1)
	if len(nearestNeighbors) == 0 {
		return -1
	}
	
	if cellPoint, ok := nearestNeighbors[0].(kdtreeCellSitePoint); ok {
		return cellPoint.CellIndex
	}
	
	return -1
}

// LookupPlateIDAtCoordinate provides a standalone function to lookup plate ID
// without needing to create a PlatePixelLookup struct
func LookupPlateIDAtCoordinate(
	x, y int,
	plates []tectonics.TectonicPlate,
	cellAssignments []int,
	voronoiCells []tectonics.VoronoiCell,
	icosphereVertices []tectonics.Vector3D,
	mapWidth, mapHeight int,
	planetRadius float64) (int32, error) {

	lookup := NewPlatePixelLookup(plates, cellAssignments, voronoiCells, icosphereVertices, mapWidth, mapHeight, planetRadius)
	return lookup.GetPlateIDAtPixel(x, y)
}