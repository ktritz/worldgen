# World Generation Project

Procedural world generation system for Earth-like planets.

**Pipeline:** Plates → Elevation → Climate → Biomes → Civilizations → Trade

**Current Phase:** Elevation complete, ready for Climate

## Architecture

```
worldgen/
├── landgen/
│   ├── terrain/          # Active elevation system (Red Blob Games approach)
│   ├── tectonics/        # Plate generation & research code
│   ├── mapper/           # Color mapping utilities
│   ├── visualization/    # Spheremap rendering
│   └── pipeline.go       # Pipeline orchestration
├── icosphere/            # Spherical mesh & Voronoi (CVT)
├── procnoise/            # Noise library
├── server/               # Web visualization (Three.js)
├── cmd/generate_planet/  # Main generation tool
└── deprecated/           # Archived research code
```

## Current Status

### Elevation System (Complete)

See `landgen/terrain/README.md` for full documentation.

Working features:
- Bimodal hypsometry (ocean floor ~-4000m, land ~300m)
- Power-law plate sizing
- Subduction zones with trench/forearc/arc profiles
- Hotspot island chains (Hawaii-style)
- Erosion smoothing
- 29% land coverage target
- ~7 seconds for 3 planets

### Build & Run

```bash
go build -o cmd/generate_planet/generate_planet ./cmd/generate_planet
./cmd/generate_planet/generate_planet
```

Output: `output/maps/planet_seed*_{elevation,landocean,globe_*,hypsometry}.png`

## Next Phase: Climate

Potential approach:
1. **Temperature**: Latitude-based with elevation adjustment
2. **Precipitation**: Hadley cells, orographic effects, rain shadows
3. **Ocean currents**: Coriolis-driven gyres, upwelling zones
4. **Wind patterns**: Trade winds, westerlies, polar easterlies

## Deprecated Code

Archived in `deprecated/`:
- `elevation/` - Old 7-module elevation system
- `algorithms/` - Plate generation experiments
- `tests/` - Old test programs
- `docs/` - Research PDFs

The `landgen/tectonics/` directory contains plate generation research (quota-based, convection methods) that achieved 73% Earth benchmark scores. This work informed the current terrain system but is not actively used.

## Commands

```bash
# Generate planets
go build -o cmd/generate_planet/generate_planet ./cmd/generate_planet
./cmd/generate_planet/generate_planet

# Run web server
cd server && go run main.go

# Build all
go build ./...
```
