# worldgen — working guidance

Procedural world generation for Earth-like planets on an icosphere Voronoi mesh.
Pipeline: **Plates → Elevation → Climate → Biomes → Hydrology/Soils/Resources → Population →
Settlements → Trade → Polities → Trade Goods.**

Go 1.22, module `worldgen`, no framework. Orientation docs (read as needed, don't duplicate here):
`DEV_CONTEXT.md` (architecture), `docs/CURRENT_STATUS.md`, `docs/CONFIG_SCHEMAS.md` (config schema
index), `docs/RESOLUTION_AUDIT.md` + `docs/RESOLUTION_REMEDIATION.md` (active work),
`docs/FUTURE_WORK_SCOPING.md` (roadmap).

## Build & test

```bash
export GOCACHE=/tmp/go-build-cache          # shared with the validation scripts; keeps reruns fast
go build ./climgen/ ./landgen/... ./cmd/review_planets/
go test ./climgen/ ./landgen/... ./cmd/review_planets/
```

- **`cmd/compare_noise` is broken** against the current `landgen/terrain` API (pre-existing). Don't
  use `go build ./...` as a health check; build the packages above instead.
- Many files have **pre-existing CRLF line endings**, so `gofmt -l` lists files you didn't touch.
  Format only your own changes; don't bulk-rewrite line endings (it buries real diffs).

## Resolution independence — the primary correctness constraint

Cell counts by subdivision level: **L5 = 10242 (baseline), L6 = 40962, L7 = 163842, L8 = 655362.**
Supported envelope is **L5–L8**; L4 and coarser are intentionally uncorrected. Cell linear size
∝ 1/√cellCount, cell area ∝ 1/cellCount. **Algorithm outcomes must not change with mesh
resolution.**

Canonical helpers — `climgen/resolution_scale.go` (mirrored in `landgen/terrain/`):

| helper | use for |
|---|---|
| `meshPathCostResolutionScale(n)` | per-step linear scale, clamp [0.125, 1.0] |
| `meshResolutionAdjustedSteps(base, n)` | hop/step budgets — **advective** (1 cell per pass) |
| `meshResolutionAdjustedDiffusionIterations(base, n)` | **diffusive** neighbor-averaging smoothers |
| `meshScaledTerritoryLinearCells / AreaCells` | territory cell counts → baseline-equivalent |

Rules:
1. **Every fix must be an exact no-op at L5** (scale == 1.0). If it changes baseline output, that is
   a deliberate decision requiring a note and a cache bump — not a silent side effect.
2. **Diffusive vs advective scaling differ.** Neighbor-averaging spreads σ ≈ cellSize·√iters, so a
   fixed physical radius needs iterations ∝ N. Directional one-cell-per-pass propagation needs
   iterations ∝ 1/scale. Using the linear rule on a diffusive smoother is the most common bug here.
3. **Per-hop decay must become per-distance**: `pow(f, stepScale)`. If the decay compounds through a
   BFS frontier, the naive form is still wrong — see the telescoping exponent in
   `climgen/precipitation_upwind_footprint.go`.
4. **Raw cell counts vs constants are wrong** — convert with `meshScaledTerritory*` first.
5. **Compute the scale from the full mesh cell count**, never a filtered subset or path length.
6. Reference implementations: `climgen/hydrology_resolution.go` (physical-radius BFS),
   `climgen/trade_goods_nodes.go` `nodeCatchmentPotential` (scaled budget + physical decay + mean).
7. Every fix needs a test that constructs **L5- and L6-sized** cell arrays. A fixture small enough
   for the scale to clamp to 1.0 tests only the no-op path and is vacuous.

## Validation workflow

`cmd/review_planets` is the diagnostic driver (defaults to level 6). Results are cached per phase;
each key chains its upstream key, so bumping one version keeps earlier phases warm.

- Cache phase versions live in `cmd/review_planets/cache.go`
  (terrain → climate → derived → tradegoods → civilization → maritime → economy).
  **Bump only the phases your change actually invalidates**, and only once per wave.
- Matched cross-level sweep: `scripts/run_level67_resolution_civ_validation.sh`
  (or `scripts/refill_review_cache.py` + `scripts/compare_review_levels.py` directly).
- **Do not edit Go source while a sweep is running.** `refill_review_cache.py` invokes `go run` per
  seed, so mid-sweep edits make different seeds run different code. Level 7 seeds take ~4× level 6
  (~25 min/seed at L6), so sweeps are hours-long — plan doc/config work for that window.

## Conventions

- **Authored tuning belongs in JSON under `config/`**; schema, loading, validation, and merge logic
  belong in Go. Start from `docs/CONFIG_SCHEMAS.md`.
- `config/drafts/` holds unwired proposals — nothing loads them. Don't treat them as live config.
- Per-cell fields are normalized [0,1] unless physically meaningful (°C, mm/yr, m). Prefer
  percentile/quantile normalization over absolute thresholds on distributions that shift with
  resolution.
- Keep `deprecated/` out of scope; `landgen/tectonics/` is research code, not the live path.

## Repo state gotcha

The working tree usually carries **substantial uncommitted work**. When reviewing a diff,
`git diff` against HEAD does **not** distinguish "this change introduced it" from "already there" —
an adversarial review misattributed several pre-existing defects that way. State the baseline
explicitly before drawing conclusions, and prefer targeted `git show HEAD:<file>` comparisons.
