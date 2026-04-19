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

## Schema Discovery

- When working with authored config, look for schema/types first before inferring field names from JSON examples.
- Primary schema entry points:
  - `climgen/trade_goods_settings_types.go`
  - `climgen/land_routes_settings_types.go`
  - `climgen/river_routes_settings_types.go`
  - `climgen/maritime_route_settings_types.go`
  - `climgen/resource_abundance_settings.go`
  - `climgen/population_support_settings.go`
- Primary authored packs:
  - `config/trade_goods_earthlike.json`
  - `config/land_routes_earthlike.json`
  - `config/river_routes_earthlike.json`
  - `config/maritime_vessels_earthlike.json`
  - `config/resource_abundance_earthlike.json`
  - `config/population_support_earthlike.json`
- Human-readable schema index:
  - `docs/CONFIG_SCHEMAS.md`
- If you add or rename JSON fields, update both the Go schema/types and `docs/CONFIG_SCHEMAS.md` in the same change.

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
- For long validation sweeps, prefer a user systemd unit with repo-persisted logs instead of `/tmp`, `nohup`, or an open exec session. `/tmp` may be cleared by reboot, and background children may be cleaned up when the command wrapper exits.
- Use an absolute Go binary path for `systemd-run`, because it may not inherit the interactive shell `PATH`:
  - `mkdir -p output/review_planets/sweeps`
  - `systemd-run --user --unit=worldgen-l6-sweep --working-directory=/home/ktritz/projects/worldgen /usr/bin/zsh -lc 'GOCACHE=/tmp/go-build-cache /usr/local/go/bin/go run ./cmd/review_planets -level 6 -seeds 4,6,7,42,55,84,91,101,128,144,177,202,255,314,512,777,999,1337 -width 64 -render=false -maritime-compare-vessels caravel > output/review_planets/sweeps/worldgen_l6_overnight_sweep.txt 2> output/review_planets/sweeps/worldgen_l6_overnight_sweep.err'`
- Check long sweep progress with:
  - `systemctl --user status worldgen-l6-sweep.service --no-pager`
  - `tail -80 output/review_planets/sweeps/worldgen_l6_overnight_sweep.txt`
  - `python3 scripts/summarize_review_planets.py output/review_planets/sweeps/worldgen_l6_overnight_sweep.txt`

## Current Expectations

- Humans are baseline-dominant through JSON-authored prevalence, but specialist biomes should still favor specialist ancestries.
- Ship/vessel rosters, land transport modes, and river route modes belong in JSON catalogs, not Go literals.
- Render/review code should stay modular as new map layers are added.
