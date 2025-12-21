# Data Structures and API Reference

## Overview

This document defines the key data structures used throughout the worldgen codebase and their relationships. **Always consult this document before implementing new features** to avoid redundant calculations and ensure optimal performance.

---

## Core Data Structures

### 1. Icosphere Mesh

#### `Vector3D` (icosphere/vector.go)
```go
type Vector3D struct {
    X, Y, Z float64
}
```
- 3D point on a unit sphere
- All vertices are normalized (length = 1.0)
- Scale by `planetRadius` for actual coordinates

#### `Triangle` (icosphere/icosphere.go)
```go
type Triangle struct {
    V1, V2, V3 int  // Indices into vertex array
}
```
- Delaunay triangulation face
- Indices reference the `icosphereSites` array

#### **Generation Pipeline**
```go
vertices, faces := icosphere.CreateIcosphere(subdivision)
// vertices: []Vector3D - icosphere sites (normalized to unit sphere)
// faces: []Triangle - Delaunay triangulation
```

**IMPORTANT:** `vertices` represent BOTH:
- Icosphere site positions (Voronoi cell centers)
- Delaunay mesh vertices

---

### 2. Voronoi Diagram

#### `VoronoiCell` (icosphere/voronoi.go)
```go
type VoronoiCell struct {
    SiteIndex           int32   // Index of icosphere site (into vertices array)
    NeighborSiteIndices []int32 // PRE-COMPUTED neighbors (REUSE THIS!)
    VertexIndices       []int32 // Indices into Voronoi vertex array
}
```

**CRITICAL:** `NeighborSiteIndices` contains **pre-computed adjacency** from the Delaunay triangulation!

#### **Generation Pipeline**
```go
voronoiVertices, voronoiCells := icosphere.GenerateSphericalVoronoi(vertices, faces)
// voronoiVertices: []Vector3D - Voronoi polygon vertices (circumcenters)
// voronoiCells: []VoronoiCell - one per icosphere site
```

**Performance Note:**
- Building adjacency from `faces` is O(faces) ≈ O(6N) for N sites
- `voronoiCells[i].NeighborSiteIndices` is **already computed** during Voronoi generation
- **ALWAYS pass `voronoiCells` to functions that need adjacency**

---

### 3. Tectonic Plates

#### `TectonicPlate` (landgen/tectonics/types.go)
```go
type TectonicPlate struct {
    ID                  int32
    Center              Vector3D
    AngularVelocity     Vector3D  // Euler pole (rad/Myr)
    SiteIndices         []int32   // Sites belonging to this plate
    PlateType           PlateTypeEnum  // Continental or Oceanic
    // ... other fields
}
```

#### **Plate Assignment**
```go
plates, cellAssignments, _ := tectonics.QuotaBasedGeneration(
    voronoiCells, vertices, planetRadius, settings)
// plates: []TectonicPlate
// cellAssignments: []int - plate ID for each site (index into plates)
```

**Important:** `cellAssignments[siteIdx]` = plate ID (NOT array index - use plateMap)

---

### 4. Plate Boundaries

#### `FindPlateBoundariesAndTypes()` (landgen/tectonics/boundaries.go)

**CORRECT USAGE:**
```go
isBoundary, siteBoundaryTypes, distances, nearestIndices, pairTypes, proximityInfo :=
    tectonics.FindPlateBoundariesAndTypes(
        vertices,              // icosphere sites
        faces,                 // Delaunay triangulation
        sitePlateIDs,          // []int32 plate assignments
        plates,                // []TectonicPlate
        voronoiCells,          // ← ALWAYS PASS THIS! (pre-computed neighbors)
    )
```

**WRONG - Don't do this:**
```go
// ❌ BAD: Passing nil forces O(faces) adjacency rebuild
isBoundary, ... := tectonics.FindPlateBoundariesAndTypes(
    vertices, faces, sitePlateIDs, plates, nil)  // ← Wastes time!
```

#### Returns:
- `isBoundary []bool` - true if site is on a boundary
- `siteBoundaryTypes map[int32]PlateInteractionType` - Divergent/Convergent/Transform/Passive
- `distances []float64` - distance to nearest boundary
- `nearestIndices []int32` - index of nearest boundary site
- `pairTypes map[FrozensetVal]PlateInteractionType` - interaction per plate pair
- `proximityInfo []BoundaryProximityInfo` - rich boundary data for volcanic arcs, etc.

---

## Common Anti-Patterns (DON'T DO THIS)

### ❌ **Anti-Pattern 1: Rebuilding Adjacency**
```go
// WRONG - Rebuilding adjacency from faces
func processBoundaries(vertices, faces, ...) {
    adjacency := make(map[int32][]int32)
    for _, face := range faces {  // O(faces) unnecessary work
        // build adjacency...
    }
}
```

**✅ CORRECT:**
```go
// Use pre-computed VoronoiCell neighbors
func processBoundaries(voronoiCells []VoronoiCell, ...) {
    for _, cell := range voronoiCells {
        neighbors := cell.NeighborSiteIndices  // Already computed!
        // process neighbors...
    }
}
```

### ❌ **Anti-Pattern 2: Scanning All Faces for Neighbors**
```go
// WRONG - O(sites * faces) complexity
for siteIdx := 0; siteIdx < numSites; siteIdx++ {
    for _, face := range faces {  // DON'T DO THIS!
        // check if face contains siteIdx...
    }
}
```

**✅ CORRECT:**
```go
// O(sites * avg_neighbors) ≈ O(6N) for icosphere
adjacency := buildFromVoronoiCells(voronoiCells)
for siteIdx := 0; siteIdx < numSites; siteIdx++ {
    neighbors := adjacency[siteIdx]  // O(1) lookup
}
```

---

## Data Flow Diagram

```
CreateIcosphere(subdivision)
    ↓
vertices ([]Vector3D), faces ([]Triangle)
    ↓
GenerateSphericalVoronoi(vertices, faces)
    ↓
voronoiVertices ([]Vector3D), voronoiCells ([]VoronoiCell)
    ↓                                    ↓
    |                      ┌─────────────┘
    |                      ↓
    |         voronoiCells[i].NeighborSiteIndices ← PRE-COMPUTED!
    ↓
QuotaBasedGeneration(voronoiCells, vertices, ...)
    ↓
plates ([]TectonicPlate), cellAssignments ([]int)
    ↓
FindPlateBoundariesAndTypes(vertices, faces, ..., voronoiCells)
    ↑                                                 ↑
    └─────────────────────────────────────────────────┘
                  ALWAYS PASS voronoiCells!
```

---

## Performance Notes

### Complexity Analysis

| Operation | Without VoronoiCells | With VoronoiCells | Speedup |
|-----------|---------------------|-------------------|---------|
| Build adjacency | O(faces) ≈ O(6N) | O(N) copy | ~6x |
| Find neighbors for 1 site | O(faces) | O(1) | ~983,520x (level 7) |
| Find neighbors for all boundaries | O(B × faces) | O(B) | ~983,520x |
| Process all sites' neighbors | O(N × faces) | O(N × 6) | ~163,680x |

**Real-world impact (Level 7: 163,842 sites, 327,680 faces):**
- Boundary detection: 189s → 2.2s (86x faster) by using pre-computed adjacency
- Memory: ~10 MB for adjacency map (negligible)

### Memory Layout

```
Level 7 (163,842 sites):
- vertices: 163,842 × 24 bytes = 3.9 MB
- faces: 327,680 × 12 bytes = 3.9 MB
- voronoiCells: 163,842 × ~48 bytes = 7.9 MB
- voronoiCells[].NeighborSiteIndices: ~6/site × 4 bytes = 3.9 MB
TOTAL: ~20 MB (easily fits in L3 cache)
```

---

## Usage Checklist

Before implementing a feature that needs site neighbors:

- [ ] Check if `VoronoiCell.NeighborSiteIndices` is available
- [ ] Pass `voronoiCells` to functions instead of rebuilding adjacency
- [ ] Verify function signature accepts `voronoiCells []VoronoiCell`
- [ ] Add fallback `if voronoiCells == nil` for backward compatibility
- [ ] Document WHY pre-computed data is used (performance)

---

## Revision History

- **v1.0** (2025-01-04): Initial documentation after discovering redundant adjacency calculations in boundary detection
- Documented VoronoiCell pre-computed neighbors
- Added anti-patterns and performance impact

---

## See Also

- `icosphere/voronoi.go` - VoronoiCell generation
- `landgen/tectonics/boundaries.go` - Boundary detection algorithms
- `landgen/tectonics/boundary_geometry.go` - Realistic boundary transformation
- `DEV_CONTEXT.md` - Project overview and current status
