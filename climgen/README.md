# climgen - Climate Generation Package

Procedural climate simulation for planet generation. Generates physically-motivated ocean currents, wind patterns, temperature distributions, and biomes.

## Schema Entry Points

When changing authored JSON packs, start with the Go schema/types file instead
of inferring fields from examples.

- Trade goods:
  - schema: `trade_goods_settings_types.go`
  - defaults: `trade_goods_settings_defaults.go`
  - merge: `trade_goods_settings_merge.go`
  - validation: `trade_goods_settings_validate.go`
  - pack: `../config/trade_goods_earthlike.json`
- Human-readable schema index:
  - `../docs/CONFIG_SCHEMAS.md`

If a JSON field is added, renamed, or repurposed, update the schema doc in the
same change.

## File Organization

Files are prefixed by system (`currents_`, `wind_`, `temp_`, `biome_`):

```
climgen/
├── README.md                    # This file
├── types.go                     # Shared types (Vector3D, VoronoiCell, settings)
├── generation.go                # Main orchestrator entry point
│
├── currents.go                  # Ocean currents - main entry point
├── currents_streamfunction.go   # Sverdrup model, velocity computation
├── currents_boundary.go         # Coastline utilities, resolution scaling
├── currents_basins.go           # Basin detection for future use
├── currents_smoothing.go        # Parallel diffusion utilities
├── currents_visualization.go    # Equirectangular map rendering
├── currents_legacy.go           # Legacy basin-based approach (preserved)
│
├── wind.go                      # Atmospheric wind - entry point, settings, types
├── wind_circulation.go          # Pressure cells, geostrophic wind
├── wind_surface.go              # Surface friction effects
├── wind_orographic.go           # Mountain blocking/deflection
├── wind_visualization.go        # Wind map rendering
│
├── temp_*.go                    # (future) Temperature distribution
└── biome_*.go                   # (future) Biome classification
```

## Ocean Currents

### Algorithm

Wind-driven Sverdrup streamfunction with proper eastern boundary dynamics:

1. **Estimate cell size** - Sample neighbor distances for resolution independence
2. **Compute eastern fetch** - BFS from eastern coastlines (where land is EAST of water)
3. **Generate streamfunction Ψ**:
   - Wind curl: `sin(3φ) * cos(φ)` creates subtropical/subpolar gyres
   - Sverdrup balance: `Ψ ∝ curl(τ) / β * eastFetch`
   - Result: Ψ=0 at east coast, builds westward
4. **Smooth Ψ** - Diffuse while enforcing Ψ=0 at coastlines
5. **Compute velocity** - `v = n × ∇Ψ` (divergence-free by construction)
6. **Smooth velocity** - Removes numerical spikes, preserves coherent currents
7. **Coastal dampening** - Reduces velocity near coastlines

### Key Properties

- **Divergence-free flow** - Guaranteed by streamfunction formulation
- **Coast-parallel boundary** - Ψ=0 at coastlines enforces no-flow-through
- **Resolution independent** - All parameters use angular distances (radians)
- **Western intensification** - Emerges naturally from eastern fetch integration

### Parameters

```go
// Streamfunction smoothing (currents.go)
targetDiffusionAngular = 0.02   // ~1.1° diffusion distance

// Velocity smoothing (currents.go)
velocitySmoothAngular = 0.035   // ~2° - removes spikes, preserves bulk flow
velocitySmoothFactor = 0.35     // Blend weight per iteration

// Coastal dampening (currents_streamfunction.go)
dampeningAngular = 0.03         // ~1.7° zone width
// Velocity scales 0.2 at coast → 1.0 at zone edge
```

### Resolution Scaling

```
Level 4:   2,562 verts, cellSize ≈ 4.3°
Level 5:  10,242 verts, cellSize ≈ 2.1°
Level 6:  40,962 verts, cellSize ≈ 1.1°
Level 7: 163,842 verts, cellSize ≈ 0.5°
```

## Usage

```go
import "worldgen/climgen"

// Generate ocean currents
settings := climgen.DefaultOceanCurrentSettings()
settings.Verbose = true

result, err := climgen.GenerateOceanCurrents(
    sites,           // []Vector3D - vertex positions
    cells,           // []VoronoiCell - adjacency
    elevation,       // []float64 - terrain height
    0.0,             // sea level threshold
    settings,
)

// result.Currents contains velocity vectors for each vertex
```

## Test Program

```bash
go build ./cmd/test_ocean_currents/
./cmd/test_ocean_currents/test_ocean_currents

# Output: output/ocean_currents/ocean_currents_seed*.png
```

## Physics Reference

### Sverdrup Balance

```
β * v = curl(τ) / (ρH)

Integrated westward from eastern boundary:
Ψ(x) = ∫[x_east to x] curl(τ) / β dx
```

### Wind Curl Pattern

```go
windCurl := math.Sin(3.0*lat) * math.Cos(lat)
```

- `sin(3φ)`: Zero-crossings at 0°, 30°, 60° create subtropical/subpolar bands
- `cos(φ)`: Tapers to zero at poles where thermohaline circulation dominates

## Atmospheric Wind

### Algorithm

Pressure-driven geostrophic balance with surface friction and orographic effects:

1. **Compute circulation pressure** - 3-cell model (Hadley, Ferrel, Polar)
2. **Smooth pressure** - Resolution-independent diffusion
3. **Compute geostrophic wind** - `v = (1/f) × ∇p` perpendicular to pressure gradient
4. **Apply surface friction** - Reduces speed, deflects toward low pressure
5. **Apply orographic deflection** - Mountains block and redirect flow
6. **Smooth wind field** - Final numerical stabilization

### Circulation Cells

```
Latitude    Pressure    Surface Wind
─────────────────────────────────────
90° (pole)    HIGH      Polar easterlies
60°           LOW
              ↑         Westerlies (SW in NH, NW in SH)
30°           HIGH
              ↓         Trade winds (NE in NH, SE in SH)
0° (ITCZ)     LOW
```

### Key Properties

- **Geostrophic balance** - Wind flows parallel to isobars (90° from pressure gradient)
- **Coriolis deflection** - Low pressure to left (NH) or right (SH) of wind
- **Surface friction** - 10% speed reduction over ocean, 30% over land
- **Orographic effects** - Mountains block/deflect low-level airflow

### Parameters

```go
// Circulation (wind.go)
HadleyEdgeLat = 30.0       // Trade winds → Westerlies boundary
FerrelEdgeLat = 60.0       // Westerlies → Polar easterlies boundary
PressureStrength = 1.0     // Pressure gradient magnitude

// Surface (wind.go)
LandFriction = 0.3         // 30% speed reduction over land
OceanFriction = 0.1        // 10% speed reduction over ocean

// Orographic (wind.go)
BlockingThreshold = 1500.0 // Meters - mountains block above this
DeflectionStrength = 0.7   // How strongly mountains deflect (0-1)
```

### Usage

```go
import "worldgen/climgen"

settings := climgen.DefaultWindSettings()
settings.Verbose = true

result, err := climgen.GenerateWindField(
    sites,           // []Vector3D - vertex positions
    elevation,       // []float64 - terrain height
    0.0,             // sea level threshold
    adj,             // *FlatAdjacency - neighbor structure
    settings,
)

// result.SurfaceWind contains wind vectors
// result.Pressure contains pressure field
// result.CirculationZone contains zone classification
```

### Test Program

```bash
go build ./cmd/test_wind/
./cmd/test_wind/test_wind

# Output: output/wind/wind_seed*.png
```

### Visualization

Pressure is color-coded: **blue** (low) → **white** (mid) → **red** (high)

Expected pattern:
- Blue bands at equator (ITCZ) and ~60° (subpolar lows)
- Red bands at ~30° (subtropical highs) and poles
