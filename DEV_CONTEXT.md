# World Generation Project

Procedural world generation system for Earth-like planets.

**Pipeline:** Plates → Elevation → Climate → Biomes → Civilizations → Trade

**Current Phase:** Active climate, civilization, trade, and review diagnostics work

## Current Working Entry Points

- Review/diagnostic driver:
  - `cmd/review_planets/main.go`
- Climate/trade package:
  - `climgen/`
- Authored config packs:
  - `config/`
- Human-readable schema index:
  - `docs/CONFIG_SCHEMAS.md`
- Current project status and next-step snapshot:
  - `docs/CURRENT_STATUS.md`

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

### Terrain And Climate

Terrain is stable enough to support downstream climate and civilization work.
Most active iteration is now happening in:
- climate diagnostics
- resource generation
- settlement/trade modeling
- review/caching infrastructure

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

## Config And Schema Notes

- Authored tuning belongs in JSON under `config/` whenever practical.
- Schema/types, loading, validation, and merge logic belong in Go.
- For config schema discovery, start with `docs/CONFIG_SCHEMAS.md`.

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
