# Worldgen Agent Notes

## Authoring Rules

- Put authored tuning data in JSON under `config/` whenever practical.
- Do not hardcode authored catalogs, race/profile weights, transport rosters, resource balances, or route-mode parameters in Go.
- Go code should primarily handle:
  - schema/types
  - loading
  - validation
  - merge/composition logic
  - scoring/simulation logic
- Minimal in-code fallbacks are acceptable only as emergency structural defaults when embedded JSON loading fails.

## Config Pattern

- Prefer one config file per subsystem or pack rather than one giant file.
- Good examples already in use:
  - `config/land_routes_earthlike.json`
  - `config/river_routes_earthlike.json`
  - `config/maritime_vessels_earthlike.json`
  - `config/profiles/fantasy/...`
- If a subsystem has many entries, prefer modular directories plus a small manifest/pack file.

## Modularity

- Keep files near a soft limit of roughly `400-500` lines.
- Split orchestration, rendering, metrics, pathing, and settings into separate files when a subsystem grows.
- Avoid letting review command files or renderer files accumulate unrelated responsibilities.

## Trade / Transport Modeling

- Keep transport layers separate:
  - land
  - river
  - coastal
  - ocean
- Keep route concepts separate:
  - movement cost
  - risk
  - support
  - viability
- Support feeder/trunk/handoff structure rather than forcing one mode to do every leg.

## Editing / Validation

- Use `apply_patch` for manual file edits.
- Add focused tests for new loaders, validators, and scoring behavior.
- Run targeted Go tests after substantial changes:
  - `env GOCACHE=/tmp/go-build-cache go test ./climgen ./cmd/review_planets`

## Current Expectations

- Humans are baseline-dominant through JSON-authored prevalence, but specialist biomes should still favor specialist ancestries.
- Ship/vessel rosters, land transport modes, and river route modes belong in JSON catalogs, not Go literals.
- Render/review code should stay modular as new map layers are added.
