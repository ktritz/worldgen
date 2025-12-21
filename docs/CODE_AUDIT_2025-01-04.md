# Code Audit: Redundant Data Structure Calculations
**Date:** 2025-01-04
**Auditor:** Claude (AI Assistant)
**Focus:** Identifying redundant adjacency/neighbor calculations

---

## Executive Summary

**Issue Found:** Multiple functions in `landgen/tectonics/` rebuild site adjacency from triangle faces instead of using pre-computed `VoronoiCell.NeighborSiteIndices`.

**Impact:**
- O(faces) redundant work on every call
- At Level 7: 327,680 faces scanned unnecessarily
- Already fixed in `boundaries.go` (189s → 2.2s improvement)
- **Still present** in `boundary_geometry.go`

---

## Audit Results

### ✅ FIXED: `landgen/tectonics/boundaries.go`

**Function:** `FindPlateBoundariesAndTypes()`

**Before:**
```go
// Line 257: Built adjacency from scratch every time
siteAdjacency := buildSiteAdjacencyList(icosphereFaces, numSites)
```

**After:**
```go
// Lines 260-270: Now accepts voronoiCells parameter
if voronoiCells != nil && len(voronoiCells) == numSites {
    fmt.Println("  Using pre-computed VoronoiCell neighbor data...")
    siteAdjacency = make(map[int32][]int32, numSites)
    for i := range voronoiCells {
        siteAdjacency[int32(i)] = voronoiCells[i].NeighborSiteIndices
    }
} else {
    fmt.Println("  Building site adjacency list from faces (VoronoiCells not provided)...")
    siteAdjacency = buildSiteAdjacencyList(icosphereFaces, numSites)
}
```

**Performance Impact:** 189s → 2.2s (86x faster)

---

### ⚠️ NEEDS FIX: `landgen/tectonics/boundary_geometry.go`

#### Issue 1: `ReassignSitesToTransformedBoundaries()` - Line 1036

**Location:** `boundary_geometry.go:1036`

**Current Code:**
```go
// Build adjacency graph to find sites within N hops of boundary
// This is much faster than checking all sites
adjacency := buildAdjacencyGraph(icosphereFaces)

// Find all sites within influence distance using breadth-first search
candidateSites := findSitesNearBoundaries(
    boundarySiteSet,
    adjacency,  // ← Built from faces every time!
    icosphereSites,
    influenceDistanceMeters,
    planetRadius,
)
```

**Problem:** Builds full adjacency from 327,680 faces on every call

**Frequency:** Called once per boundary transformation

**Fix Required:**
1. Accept `voronoiCells` parameter in `ReassignSitesToTransformedBoundaries()`
2. Convert `VoronoiCell.NeighborSiteIndices` to adjacency map
3. Fall back to `buildAdjacencyGraph()` if voronoiCells is nil

**Estimated Impact:** ~500ms savings at Level 7

---

#### Issue 2: `smoothReassignments()` - Line 1388

**Location:** `boundary_geometry.go:1388`

**Current Code:**
```go
func smoothReassignments(
    newAssignments []int32,
    originalAssignments []int32,
    icosphereFaces []Triangle,
) []int32 {
    // Build adjacency graph
    adjacency := make(map[int32][]int32)

    for _, face := range icosphereFaces {  // ← Rebuilds every time!
        vertices := [3]int32{int32(face.V1), int32(face.V2), int32(face.V3)}

        for i := 0; i < 3; i++ {
            v1 := vertices[i]
            v2 := vertices[(i+1)%3]

            adjacency[v1] = appendUnique(adjacency[v1], v2)
            adjacency[v2] = appendUnique(adjacency[v2], v1)
        }
    }

    // ... smoothing passes
}
```

**Problem:** Builds full adjacency from faces for smoothing passes

**Frequency:** Called once per boundary transformation

**Fix Required:**
1. Accept pre-computed `adjacency map[int32][]int32` parameter
2. Pass adjacency from caller (already available from VoronoiCells)
3. Remove face-building logic

**Estimated Impact:** ~500ms savings at Level 7

---

#### ✓ ACCEPTABLE: `buildBoundaryAdjacency()` - Line 273

**Location:** `boundary_geometry.go:273`

**Code:**
```go
func buildBoundaryAdjacency(
    icosphereFaces []Triangle,
    isBoundarySite []bool,
) map[int32][]int32 {
    adjacency := make(map[int32][]int32)

    for _, face := range icosphereFaces {
        vertices := [3]int32{int32(face.V1), int32(face.V2), int32(face.V3)}

        for i := 0; i < 3; i++ {
            v1 := vertices[i]
            v2 := vertices[(i+1)%3]

            // Only include edges where BOTH vertices are boundary sites
            if int(v1) < len(isBoundarySite) && int(v2) < len(isBoundarySite) {
                if isBoundarySite[v1] && isBoundarySite[v2] {
                    // Add bidirectional edge
                    adjacency[v1] = appendUnique(adjacency[v1], v2)
                    adjacency[v2] = appendUnique(adjacency[v2], v1)
                }
            }
        }
    }

    return adjacency
}
```

**Status:** ACCEPTABLE - This builds a **FILTERED** adjacency graph (boundary sites only)

**Reason:** Cannot easily derive from VoronoiCell neighbors because it requires filtering:
- Full neighbors: All adjacent sites
- Boundary-only: Only neighbors that are also boundary sites

**Performance:** Acceptable because:
- Only processes ~8,784 boundary sites at Level 7 (5% of total)
- Specialized logic that's hard to pre-compute
- Called once during chain extraction

---

## Summary of Findings

| Location | Function | Status | Impact | Priority |
|----------|----------|--------|--------|----------|
| boundaries.go:257 | FindPlateBoundariesAndTypes | ✅ FIXED | 86x faster | Done |
| boundary_geometry.go:1036 | ReassignSitesToTransformedBoundaries | ⚠️ NEEDS FIX | ~500ms | High |
| boundary_geometry.go:1388 | smoothReassignments | ⚠️ NEEDS FIX | ~500ms | High |
| boundary_geometry.go:273 | buildBoundaryAdjacency | ✓ ACCEPTABLE | Minimal | N/A |

**Total Potential Savings:** ~1 second at Level 7 (11.3s → ~10.3s)

---

## Recommendations

### Immediate Actions:

1. **Modify `ReassignSitesToTransformedBoundaries()`:**
   ```go
   func ReassignSitesToTransformedBoundaries(
       icosphereSites []Vector3D,
       icosphereFaces []Triangle,  // Keep for fallback
       originalAssignments []int32,
       transformedBoundaries []TransformedBoundary,
       influenceDistanceKm float64,
       planetRadius float64,
       voronoiCells []VoronoiCell,  // ← ADD THIS
   ) []int32 {
       // Use voronoiCells to build adjacency if available
       var adjacency map[int32][]int32
       if voronoiCells != nil {
           adjacency = voronoiCellsToAdjacency(voronoiCells)
       } else {
           adjacency = buildAdjacencyGraph(icosphereFaces)
       }
       // ... rest of function
   }
   ```

2. **Modify `smoothReassignments()`:**
   ```go
   func smoothReassignments(
       newAssignments []int32,
       originalAssignments []int32,
       adjacency map[int32][]int32,  // ← Accept pre-computed
   ) []int32 {
       // Remove face-building logic, use passed adjacency
       // ... rest of function
   }
   ```

3. **Update all callers** to pass `voronoiCells` and pre-computed adjacency

### Long-term:

1. **Add to DATA_STRUCTURES.md:**
   - Document these functions and their requirements
   - Add examples of correct usage
   - Warning about performance impact of rebuilding

2. **Create helper function:**
   ```go
   // voronoiCellsToAdjacency converts VoronoiCell neighbors to map format
   func voronoiCellsToAdjacency(cells []VoronoiCell) map[int32][]int32 {
       adjacency := make(map[int32][]int32, len(cells))
       for i := range cells {
           adjacency[int32(i)] = cells[i].NeighborSiteIndices
       }
       return adjacency
   }
   ```

---

## Testing Plan

After fixes:
1. Run `test_level7_boundary_geometry` - should see ~1s improvement
2. Verify "Using pre-computed VoronoiCell neighbor data" appears in logs
3. Confirm isolated cell count remains 0
4. Validate plate size changes remain under 60%

---

## Related Documentation

- `docs/DATA_STRUCTURES.md` - Data structure API reference
- `landgen/tectonics/boundaries.go` - Example of correct VoronoiCell usage
- `icosphere/voronoi.go` - VoronoiCell definition

---

## Revision History

- **v1.0** (2025-01-04): Initial audit
  - Found 2 redundant calculations in boundary_geometry.go
  - Documented acceptable filtered adjacency usage
  - Provided fix recommendations
