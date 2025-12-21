# Terrain Generation System

Red Blob Games-inspired tectonic planet generation with Earth-like hypsometry.

## Features

- **Bimodal hypsometry**: Ocean floor peak ~-4000m, land peak ~300m
- **Continental slope**: Interior high, slopes to coast
- **Plate generation**: Power-law sizing (few large, many small)
- **Land coverage control**: Target 29%
- **Resolution-independent**: Parameters scale across mesh levels
- **Convergent boundaries**: 20-35% subduction zones with trench/forearc/arc profile
- **Erosion smoothing**: Selective erosion creates natural mountain ranges
- **Euler pole plate rotation**: Curved plate motion
- **Hotspot island chains**: Hawaii-style curved chains with age-graded elevation
- **Parallel processing**: 3x speedup via goroutines

## Code Structure

```
landgen/terrain/
├── types.go              # Types, constants, PlateRotation (Euler poles)
├── plates.go             # GeneratePlates, AssignPlateTypes, AssignPlateRotations
├── boundaries.go         # FindCollisions (uses rotational velocity), placeVolcanicArcs
├── elevation.go          # ComputeElevation, applySubductionProfile (BFS-optimized)
├── erosion.go            # ApplySelectiveErosion (parallel), ApplyThermalErosion
├── generate.go           # GeneratePlanetElevation (main pipeline)
├── hotspots_types.go     # Hotspot constants and types
├── hotspots_generation.go# PlaceHotspots
├── hotspots_elevation.go # ApplyHotspotElevation (curved chain tracing)
├── hypsometry.go         # ApplyEarthHypsometry
├── base_elevation.go     # Base terrain elevation
├── collision_elevation.go# Collision zone elevation
├── continental_mask.go   # Continental masking
├── hybrid_generation.go  # Hybrid plate generation
├── evaluation.go         # Terrain evaluation metrics
└── metrics.go            # Performance metrics
```

## Pipeline (generate.go)

1. Generate plates (weighted BFS)
2. Assign plate types (oceanic/continental)
3. Assign plate rotations (Euler poles for curved motion)
4. Compute elevation (distance fields + boundary classification)
5. Compute continental distance from coast
6. Apply bimodal elevation
7. Place hotspot island chains (curved tracing via Euler poles)
8. Apply erosion
9. Apply Earth hypsometry
10. Determine land/ocean

## Key Parameters (types.go)

- `CollisionThreshold = 0.5` - Normalized, resolution-independent
- `DeltaTime = 1e-2` - Velocity simulation timestep
- `VolcanoDistanceRadians = 0.03` - ~190km trench-to-arc distance
- `NoiseAmplitude = 0.1` - Elevation noise

## Hotspot System

- **PlateRotation** stores Euler pole and angular velocity
- **VelocityAt()** computes velocity at any point: v = omega x r
- **RotatePoint()** uses Rodrigues' formula for curved tracing
- ~15 hotspots per planet, only oceanic ones create chains
- Islands traced backward along rotation, age-graded elevation

### Hotspot Parameters (hotspots_types.go)

- `HotspotsPerPlanet = 15`
- `IslandSpacingRadians = 0.024` (~150km)
- `MaxChainLengthRadians = 0.31` (~2000km)
- `HotspotElevationBoost = 0.9`

## Build & Run

```bash
go build -o cmd/generate_planet/generate_planet ./cmd/generate_planet
./cmd/generate_planet/generate_planet  # ~7 seconds for 3 planets
```

## Output

```
output/maps/
├── planet_seed42_elevation.png      # Elevation color map
├── planet_seed42_landocean.png      # Land/ocean with terrain
├── planet_seed42_globe_front.png    # Orthographic globe view
├── planet_seed42_globe_side.png
├── planet_seed42_globe_south.png
└── planet_seed42_hypsometry.png     # Elevation histogram
```

## Test Seeds

- **Seed 42**: Single mega-continent, 19.5% convergent, 48 mountains
- **Seed 84**: Multiple continents, 27.3% convergent, 692 mountains
- **Seed 8**: Two continents, 23.7% convergent, 301 mountains

## Potential Enhancements

- **Mid-Ocean Ridges**: Elevated ridge axis, age-based depth, transform faults
- **Continental Hotspots**: Volcanic plateaus (Yellowstone-style)
- **Back-Arc Basins**: Extension behind subduction zones
- **Passive Margins**: Wider shelves, gradual slopes
