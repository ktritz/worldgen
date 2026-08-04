# Resolution-Independence Remediation Tracker

Source: `docs/RESOLUTION_AUDIT.md` (section numbers below reference it). This is the live status
document — update the Status column as work lands.

**Supported resolution envelope (decision, Phase 0):** L5 (10242) through L8 (655362).
Rationale: `cmd/review_planets` defaults to level 6 and branches for 7/8; `cmd/generate_planet`
runs level 7; no tool generates L4. The scale upper clamp stays at 1.0 (L4 intentionally
unsupported/uncorrected); the lower clamp extends 0.25 → 0.125 so L8 stops saturating.

## Workflow per item

1. Fix + focused unit test proving cross-resolution invariance (same physical scenario expressed
   at L5 and L6/L7 cell counts must agree within tolerance).
2. `go build ./... && go test ./climgen/ ./landgen/... ./cmd/...` (Go test caching keeps warm
   re-runs fast; use `GOCACHE=/tmp/go-build-cache` to share with the validation scripts).
3. Integration validation per wave (not per item): bump ONLY the affected review-cache phase
   versions in `cmd/review_planets/cache.go` (each key chains upstream keys, so a climate bump
   keeps terrain cached), then run the L6/L7 matched-seed sweep:
   `scripts/run_level67_resolution_civ_validation.sh` (or `refill_review_cache.py` +
   `compare_review_levels.py` directly). Ratios in the compare output should move toward 1.0.
4. Cache-version bump ownership: ONLY the orchestrator bumps versions, once per wave, to avoid
   conflicting edits. Terrain bumps invalidate everything — batch terrain fixes into one wave.

Cache phases: terrain-v8 / climate-v9 / derived-v11 / tradegoods-v2 / civilization-v65 /
maritime-v45 / economy-v48 (values as of wave start).

## Status legend

`TODO` · `IN PROGRESS` · `FIXED` (unit-tested) · `VALIDATED` (cross-level sweep confirmed) ·
`DEFERRED` (with reason)

---

## Phase 0 — Foundation (orchestrator, inline)

| ID | Audit | Item | Files | Status |
|----|-------|------|-------|--------|
| R0.1 | §0.1 | Extend lower scale clamp 0.25→0.125 (L8); document envelope L5–L8; keep upper clamp 1.0 | climgen/resolution_scale.go, landgen/terrain/resolution_scale.go, tests | FIXED |
| R0.2 | §0.1 | Deduplicate scale copies in cmd/review_planets (civilization_summaries.go:91, network_summaries.go:371,422, summaries.go:182) → export shared helpers from climgen | climgen/resolution_scale.go, cmd/review_planets/* | FIXED |
| R0.3 | §3 | Add `meshResolutionAdjustedDiffusionIterations` to climgen (port from landgen) for Phase 3 | climgen/resolution_scale.go | FIXED |
| R0.4 | §2.2 | Polity border pressure: scale adjacency-pair count by linear scale before the cap of 12 | climgen/polity_profiles.go | FIXED |

## Phase 1 — Mechanical unit fixes (parallel subagents, wave 1)

### Agent A — Trade routing

| ID | Audit | Item | Files | Status |
|----|-------|------|-------|--------|
| R1.1 | §1.1 | `tradeLinkTravelCost`: multiply per-cell accumulation by `meshPathCostResolutionScale`; thread cell count from `trade_network.go:130` | climgen/trade_network_paths.go, trade_network.go | FIXED |
| R1.2 | §1.2 | River terminal per-hop penalty `0.035*distance` → `*stepScale` | climgen/river_trade_terminals.go | FIXED |
| R1.3 | §1.5 | `riverTradeTerminalCatchmentSteps` → delegate to `meshResolutionAdjustedSteps(1, n)` | climgen/river_trade_terminals.go | FIXED |
| R1.4 | §1.5 | Dead node-graph river path (`riverLinkDirectionalTravelCost`, `shortestRiverNodePath`): apply same stepScale fix so it is correct if revived | climgen/river_trade_paths.go | FIXED |

### Agent B — Wind & precipitation per-hop decays / footprints

| ID | Audit | Item | Files | Status |
|----|-------|------|-------|--------|
| R1.5 | §3.3 | Lee shadow: `pow(decayRate, stepScale)` per hop | climgen/wind_orographic.go | FIXED |
| R1.6 | §3.3 | `leeShadowIters` clamp [3,10] → `meshResolutionAdjustedSteps`-based (both copies) | climgen/wind.go, wind_seasonal_debug.go | FIXED |
| R1.7 | §4.1 | Upwind footprint decay `pow(0.84, depth)` → `pow(0.84, depth*stepScale)` | climgen/precipitation_upwind_footprint.go | FIXED |
| R1.8 | §4.2 | Hardcoded footprint depths 3/2/4/4 → `resolutionAdjustedPrecipSteps` | climgen/precipitation_budget.go:549, precipitation_frontal_reservoir.go:389,430, precipitation_land_flux.go:84,94 | FIXED |
| R1.9 | §4.8 | Storm memory: scale iterations and `pow(carry, stepScale)` | climgen/seasonal_stormband.go | FIXED |
| R1.10 | §4.9 | Hop-harmonic weights `1/(step+1)` → `1/(1+step*stepScale)` | climgen/precipitation_transport_path.go:80, seasonal_stormband.go:372 | FIXED |

### Agent C — Currents & components

| ID | Audit | Item | Files | Status |
|----|-------|------|-------|--------|
| R1.11 | §3.6 | Coast-parallel band `maxRings := 3` → `meshResolutionAdjustedSteps(3, n)`; keep `strength` ramp physical | climgen/currents_boundary.go | FIXED |
| R1.12 | §3.7 | Ocean component gates (8 / 24 / 232) → baseline-equivalent area via `meshScaledTerritoryAreaCells` | climgen/currents_components.go | FIXED |
| R1.13 | §3.7/§5.2 | `DefaultMinComponentSize = 50` raw-cell threshold → area-scaled at consumers | climgen/types.go, currents_components.go, currents_basins.go, generation.go | FIXED |
| R1.14 | §3.8 | Normalize `toNeighbor` before east/west boundary dot-product thresholds | climgen/currents_components.go:255, currents_boundary.go:71,174 | FIXED |

### Agent D — Terrain

| ID | Audit | Item | Files | Status |
|----|-------|------|-------|--------|
| R1.15 | §6.1 | Hotspot `sizeCap` normalize against baseline cell radius, not actual mesh | landgen/terrain/hotspots_elevation.go:328,507,513,722-723 | FIXED |
| R1.16 | §6.2 | Continental caldera spread → radial angular-distance selection (no int-truncated hop cap) | landgen/terrain/hotspots_elevation.go:830-835 | FIXED |
| R1.17 | §6.3 | Abyssal-hill band wavelength: multiply hop distance by scale | landgen/terrain/elevation.go:650-653 | FIXED |
| R1.18 | §6.4 | `BreachDrainageSinks` maxRise thresholds and carve depths × scale | landgen/terrain/erosion.go:283,300,420-423, drainage_metrics.go:103 | FIXED |
| R1.19 | §6.5 | Coastal exposure BFS depth 2 → `meshResolutionAdjustedSteps(2, n)` | landgen/terrain/elevation.go:845 | FIXED |
| R1.20 | §6.11 | Arc-seed determinism: sort trench cells before greedy spacing pass | landgen/terrain/boundaries.go:285 | FIXED |

## Phase 2 — Structural fixes (wave 2)

Design notes (from orchestrator code review, 2026-08-02):
- **R2.1**: `cellsWithinHops` returns ~3r²+3r cells (6 at L5 r=1). Fix
  `SettlementNodePhysicalSupportArea` to return `7.0 * areaCells / (discSize+1)` — the supported
  fraction of the physical disc rescaled to the L5 7-cell reference. Bit-identical at L5 for
  hexagonal cells (tiny change only for the 12 pentagon cells); invariant for physical features
  at L6/L7 since numerator and denominator use same-size cells over the same physical disc.
  Same treatment in `physicallySignificantCityCandidate`. Downstream thresholds
  (0.5/0.75/1.0/1.5) then keep their L5 meaning unchanged.
- **R2.2**: hop-discs aren't circular — count×scale² drifts ~25% because 3r²+3r doesn't scale
  as r². Normalize supported counts by the *measured* union/disc size ratio to the baseline
  disc (19 cells at r=2 incl. center), not by scale² alone.
- **Sequencing**: no Go source edits while a validation sweep is running — `refill_review_cache.py`
  invokes `go run` per seed, so mid-sweep edits make different seeds run different code.

| ID | Audit | Item | Files | Status |
|----|-------|------|-------|--------|
| R2.1 | §2.1 | Settlement physical-support disc → fraction-of-disc normalized to L5 reference (fixes 6-finding cascade: node kind gates, support weight, city candidacy, proto-civ anchors) | climgen/settlement_network.go:397-426,351,374-395, proto_civilizations.go:253,323-324 | TODO |
| R2.2 | §2.3 | Region population-support disc: normalize by measured disc size | climgen/proto_civilizations.go:329-375 | TODO |
| R2.3 | §3.1 | Temperature diffusion: `D / scale²` (dx² normalization); dedupe the two copies | climgen/temperature_heat_transport.go:129, temperature_diffusion.go:64 | TODO |
| R2.4 | §3.2 | `ApplyMarineInfluence`: propagate through `influenced[]` (bug) + physical-distance kernel | climgen/temperature_continentality.go:158-186 | TODO |
| R2.5 | §5.1 | Maritime influence: per-distance wind factors `pow(f, cellKm/refKm)`; scale 0.001 cutoffs | climgen/maritime.go | TODO |
| R2.6 | §4.4/X2 | Precip iteration floor of 18: remove (convergence early-exit exists); unify 70 km vs 223 km baselines | climgen/precipitation_budget_setup.go, precipitation_budget.go | TODO |
| R2.7 | §4.6 | Land budget per-iteration fractions (source 0.020, recycling 0.12, supersat 0.68) → `precipitationPerStepFraction` | climgen/precipitation_budget.go:256-257, precipitation_land_flux.go:41-42, precipitation_land_condensation.go:121 | TODO |
| R2.8 | §4.5 | Orographic rise: `localRise /= stepScale` (not sqrt); `footprintRise` uncorrected | climgen/precipitation_wind_geometry.go:78,113 | TODO |

## Phase 3 — Diffusive smoothing policy (wave 3)

Route all fixed/linear-scaled iteration counts of neighbor-averaging smoothers through
`meshResolutionAdjustedDiffusionIterations` (iterations ∝ N for fixed physical σ).

| ID | Audit | Sites | Status |
|----|-------|-------|--------|
| R3.1 | §3.4 | wind_seasonal_tropics.go:99,108,13-14,227-234 | TODO |
| R3.2 | §3.5 | wind.go:329,412; wind_marine.go:28; wind_pressure_context.go:70; temperature_currents.go:250; currents.go:100,152; currents_windcurl.go:103,171; currents_streamfunction.go:176 | TODO |
| R3.3 | §4.3 | frontal/marine/convergence iteration constants (frontal storm transport is advective → `meshResolutionAdjustedSteps`) | TODO |
| R3.4 | §4.7 | seasonal_stormband.go:9,29,81,176; seasonal_tropical_monsoon.go:7,229-233; seasonal_tropical_convergence.go:81-87 | TODO |
| R3.5 | §5.3 | `DefaultSmoothingIterations = 2` legacy consumers | TODO |

## Phase 4 — Sampling & statistics (wave 4)

| ID | Audit | Item | Status |
|----|-------|------|--------|
| R4.1 | §1.3 | Port catchment max-statistic → percentile/area-weighted | TODO |
| R4.2 | §1.4 | Port dedupe by physical separation, not cell identity | TODO |
| R4.3 | §3.9 | Gateway/openness/orographic-deflection 1-ring detectors → scaled neighborhoods | TODO |
| R4.4 | §3.10 | `ComputeWindwardness` gradient normalization (latent — no caller) | TODO |
| R4.5 | §6.6-6.9 | Noise octave Nyquist cap; RegularizeCoastlines intent; hop floors; drainage majorThreshold floor | TODO |
| R4.6 | §1.5 | MaxStopovers top-K composition; disk-vs-ring neighbor stats; feeder floors (LOW items) | TODO |

## Phase 5 — Diagnostics (wave 5)

| ID | Audit | Item | Status |
|----|-------|------|--------|
| R5.1 | §7 | 1-ring residual/coastal/orographic-contrast diagnostics → fixed physical radius | TODO |
| R5.2 | §5.4 | Vorticity diagnostic 1/length units | TODO |
| R5.3 | §6.10 | Terrain evaluation metrics (relief per distance, fixed-ruler coastline) | TODO |

## Deferred / separate decisions

| ID | Audit | Item | Reason |
|----|-------|------|--------|
| D1 | §5.6 | Runoff units mismatch (moisture proxy 0.08–1.35 thresholded as cm/yr → all runoff terms zero planet-wide) | Not a resolution bug; fixing changes world output substantially and needs its own tuning pass. Do after resolution work stabilizes baselines. |
| D2 | §6.11 | `ApplyLandmassErosion` ~100× runtime at L7 | Performance, separate effort |
| D3 | §0.1 | L4 support | Out of envelope by decision above |
| D4 | §1.5 | Prefilter minStepCost literals → shared constants | Cosmetic hardening, fold into any trade-path touch |

## Deliberate level-5 behaviour changes (the complete list)

The working rule is that a resolution fix must be an exact no-op at the L5 baseline. These changes
break that rule **on purpose**, because the baseline itself was wrong. Anyone auditing this work
should expect L5 output to differ from pre-branch `main`:

1. **Plate layout is built on a fixed L6 reference mesh** (`landgen/terrain/plates.go`,
   `plateLayoutReferenceSubdivision = 6`) and projected onto the target mesh, replacing direct
   region-index sampling. Sampling indices made the same seed choose *different geography* at
   different mesh levels — the deepest resolution defect found. Changes plate boundaries,
   oceanic/continental assignment and Euler poles at every level including L5.
2. **Per-feature deterministic RNG** (`landgen/terrain/deterministic_rng.go`) replaces a shared
   `rand.Rand` for hotspot chains and islands, so a feature's randomness no longer depends on how
   many features were generated before it. Changes chain vigour, island variation, radius jitter.
3. **`MinEmergentHotspotIslandRadius` gate** drops islands below the emergence threshold; some
   previously emerged at L5.
4. **`AssignDistanceField` is derandomised** (`landgen/terrain/elevation.go`) — the old BFS popped
   from a Go map, so terrain was genuinely not reproducible for a fixed seed. Now a sorted FIFO.
   Changes the distance fields and therefore all elevation.
5. **`leeShadowIters` clamp** — the old floor of 3 overshot the stated 0.05 rad target at baseline.
6. **Continental caldera spread** — the old hop depth truncated to 0 at L5, so the feature never
   appeared at baseline at all.
7. **`checkDepth` base constant** — its 15 was calibrated at L7, not L5.
8. **`coastDirectionCosineThreshold`** approximates rather than reproduces the old unnormalized
   test: it derives from the *mean* L5 offset (0.035 rad), so cells whose offset differs can flip
   near the boundary. Approximate, not exact — recorded honestly.
9. **Waystation gates** converted from hop counts to great-circle degrees; the physical conversion
   is right but shifts which paths qualify at baseline.

Items 1–4 were found by pre-merge review, not by the original audit, and are the reason the earlier
"three deliberate exceptions" framing in this document was wrong.

## Adversarial review findings (2026-08-02, post-wave-1)

An independent read-only review of the working tree found real defects. **Its attribution is
unreliable** — it treated "not in HEAD" as "introduced by wave 1", but the tree carried ~60 files
of pre-existing uncommitted work before wave 1 started. Triaged below by whether wave-1 agents
actually touched the code.

### Confirmed wave-1 regression — fix before the next sweep

| ID | Item | Status |
|----|------|--------|
| W1.a | `seasonal_stormband.go:366-387` (R1.10): `oceanHits`/`stormHits` are **unnormalized sums** fed to `Clamp(...,0,1)`. `fetchSteps` was already resolution-scaled, so changing the weight to `u/(1+step·stepScale)` roughly doubles the sum at L6 and quadruples at L7 before clamping — magnitude was resolution-*independent* before, and is not now. Fix: `weight := upwindness * stepScale / (1.0 + float64(step)*stepScale)` (exact no-op at L5). | TODO |
| W1.b | `trade_network_paths.go` (R1.1): the two early returns still hand back raw `link.TravelCost`. Verify whether the `TravelCost` set at `settlement_network.go:589` comes from the stepScale-corrected `shortestPathsFromNode` (`:906/:919`) or an unscaled path builder. If unscaled, the two branches of the same function differ by 4× at L7, and `polity_spheres.go:143` uses only the unscaled branch. | VERIFY |
| W1.c | `precipitation_transport_path.go:25-32`: `1-pow(1-frac,1.0)` is not bit-exact at L5 (0.20 → 0.19999999999999996) inside a loop with a convergence early-exit. Add an `s == 1 → return frac` short-circuit. | TODO |
| W1.d | Three new tests are vacuous (fixtures small enough that scale clamps to 1.0, so they exercise only the L5 no-op path and pass against the pre-fix code): `TestNeighborOceanFractionUsesPhysicalCoastalBand`, `TestCoastalOnshoreScoreUsesPhysicalCoastalBand`, `TestHydrologyChannelThresholdFollowsAccumulationHierarchy`. | TODO |

### Pre-existing — already tracked, NOT wave-1 regressions
Confirmed against the original audit: `precipitation_wind_geometry.go:78,113` (`rise /= sqrt(stepScale)`
at both sites) is audit §4.5 = **R2.8**, and `settlement_network.go:397-409`
(`meshScaledTerritoryAreaCells(areaCells, …)` = count × scale²) is the civilization audit's root
finding = **R2.1**. Both were flagged HIGH by the reviewer as if new. They are the Phase 2 work
this plan already schedules, and the review's numbers reinforce the design notes above (support
area floor: 1.0 at L5 vs 0.0625 at L7). Also pre-existing: `erosion.go:169` `checkDepth`
(audit §6.11), `seasonal_precipitation.go` `coastalOnshoreScore`, `drainage_metrics.go`
`hydrologyChannelThreshold` (R4.5). **Reviewer's useful correction on checkDepth**: its constant
was calibrated at **L7**, not L5 (`// ~15 neighbor hops at level 7`), so wrapping it in
`meshResolutionAdjustedSteps(15, n)` quadruples the intended radius — use base 4, not 15.

### Process lesson
Wave-1 agents were briefed on specific items but several touched adjacent code (partial R2.7,
R2.8, R4.5 landed unrequested). Future waves: brief agents to **stop at the item boundary** and
report adjacent defects rather than fixing them.

## NEW BUG CLASS — absolute thresholds on fields whose *distribution shape* is resolution-dependent

Found 2026-08-03 by instrumenting the settlement-node funnel. **The original audit explicitly listed
"percentile/quantile normalization" and "[0,1] per-cell fields" as typically fine. That is wrong**,
and this is probably the largest remaining category of resolution dependence in the codebase.

**The mechanism.** A per-cell field can be perfectly well-formed and still change *distribution
shape* with resolution: coarse cells average sub-cell variation away (regression toward the mean),
fine cells preserve extremes. Measured on `CarryingCapacity` over land cells:

| | seed 42 L6 → L7 | seed 123 L6 → L7 |
|---|---|---|
| mean | .2414 → .2288 | .2044 → .1864 |
| p50 | .1904 → .1714 | .1594 → .1555 |
| p90 / p99 / max | all **rise** | all **rise** |

Mean and median fall while the upper tail fattens — the field *sharpens*. Any **absolute** cut on
such a field therefore selects a different fraction of the world at each resolution. The damage is
worst for a *middle band*, because it is squeezed from both sides:

| band (carrying) | s42 L6 → L7 | s123 L6 → L7 |
|---|---|---|
| hamlet [.38, .46) | .1636 → .1318 (−19%) | .1322 → **.0688 (−48%)** |
| village [.46, .55) | .1284 → .1256 (−2%) | .0526 → .0449 (−15%) |
| town+ [.55, ∞) | .0295 → **.0327 (+11%)** | .0140 → **.0174 (+24%)** |

The hamlet band is the entire supply of Local Anchors, and **13 of the 15 settlement nodes lost at
L7 are Local Anchors** — which is the head of the whole causal chain measured earlier (fewer nodes →
wider spacing → fewer links inside fixed travel budgets → fragmented regions → anchor shortfall →
fewer proto-civs → fewer polities → collapsed trade score).

**Fix (recommended, not yet applied): make the four kind thresholds percentiles, not constants.**
Calibrate `Hamlet/Village/Town/CityThreshold` once as the L5 land-cell quantiles of their current
absolute values, then resolve them per-mesh against the actual `CarryingCapacity`/`UrbanPotential`
distributions. Each tier's eligible population then becomes invariant by construction. Projected to
restore seed 123's hamlet band from .0688 to .1322 at L7.

**Audit implication: this pattern needs a dedicated sweep.** Every absolute threshold applied to a
derived per-cell field is suspect — biome classification, resource affinities, settlement scoring,
agriculture suitability, port thresholds. The test is not "is the field [0,1]?" but "does this
field's *distribution* hold shape across resolutions?"

### FIXED 2026-08-03 — percentile-calibrated kind thresholds (civilization-v70)

Kind thresholds now resolve per mesh from land-cell quantiles calibrated at L5 over six seeds:
carrying 0.2885 / 0.0933 / 0.0228, urban 0.2382 / 0.0541 / 0.0202.

- **Separate quantiles per field are required** — at the same absolute threshold the two fields
  select different fractions (hamlet 0.288 carrying vs 0.238 urban), so the previously shared
  setting was silently conflating them.
- **City deliberately stays absolute**: at L5 the city cut selects <1 cell per world, so a quantile
  there tracks the mesh maximum and would manufacture a Major Hub on every world.

Results — the hamlet band stops collapsing and goes exactly flat:

| | s42 L6 | s42 L7 | s123 L6 | s123 L7 |
|---|---|---|---|---|
| hamlet before | .1636 | .1318 | .1322 | **.0688** |
| hamlet after | **.1952** | **.1952** | **.1952** | **.1952** |

Seed 123 Local Anchors: 17 → 8 (−53%) becomes 23 → 28 (+22%). Summed |L6→L7| deltas across both
seeds: nodes 15 → 11, Local Anchors 13 → 7, proto-civs 4 → 2, polities 4 → 2.

**OPEN DECISION — percentile thresholds flatten seed-to-seed habitability.** Every world now has
exactly 28.85% of its land in hamlet-or-better *by construction*; only the absolute quality of those
cells differs. Per-seed L5 node counts narrowed from a 41–83 range to 47–72. For a world generator
that is probably the wrong trade.

### SUPERSEDED — replaced by per-level calibration (civilization-v71)

Per-world quantiles are gone; one mechanism remains. `SettlementNetworkSettings.KindCalibration`
holds per-level absolute cuts, looked up by mesh cell count with **log-cell-count interpolation**
and clamping, resolved once per run. City stays absolute at every level; empty table ⇒ absolute
fallback.

| level | cells | carrying h / v / t | urban h / v / t |
|---|---|---|---|
| L5 | 10242 | **0.3800 / 0.4600 / 0.5500** | **0.3800 / 0.4599 / 0.5500** |
| L6 | 40962 | 0.3876 / 0.4705 / 0.5507 | 0.3890 / 0.4704 / 0.5496 |
| L7 | 163842 | 0.3244 / 0.4714 / 0.5558 | 0.3544 / 0.4738 / 0.5500 |
| L8 | 655362 | 0.2612 / 0.4723 / 0.5609 *(extrapolated)* | 0.3198 / 0.4772 / 0.5504 *(extrapolated)* |

L5 solves back to the original absolute constants exactly — the built-in correctness check.

**Surprise: the corrections are not monotone in level.** The hamlet cut *rises* L5→L6
(0.380 → 0.388) then *falls* sharply at L7 (0.324), while village and town rise at both steps. So
the distribution change is not a simple monotone sharpening, and any single "coarsening factor"
would have been wrong.

**Both properties now hold:**
- *World variation restored*: per-seed L5 node spread 49–72 → **35–80**, and per-seed hamlet band
  fractions differ again (.1172–.2839, a 2.4× spread) instead of being pinned at .1952.
- *Resolution invariance retained*: bands near-flat (|Δ| ≤ .0041, vs .0318/.0634 pre-fix). Summed
  |L6→L7| deltas: nodes **9** (was 11 quantile, 15 pre-fix), proto **2**, polity **2**, but Local
  Anchors **10** (was 7). Total drift 23 vs 22 — a wash in aggregate, gaining on node count and
  losing in the tier sitting closest to the hamlet cut.

Not strictly dominant, and honestly so. But the quantile version bought those three Local Anchors by
making every world 28.85% settleable by construction, which is the worse trade for a world generator.

**Limiting factor is calibration data, not design**: L6/L7 rows rest on only **2 seeds** (the only
cached worlds at those levels); L5 used six. L8 is extrapolated from the L6→L7 step and marked as
such in code. Widening the reference set should tighten the residual — cheap to do once the
linear-footprint restructure lands and L7 generation drops from ~1.7 h/seed to minutes.

**Residual**: L6→L7 node drift is not zero (+4, +7). What remains is in the local-peak test and the
spacing filter, not the thresholds — candidate counts still grow with resolution (s123 628 → 1317).

### Hypotheses tested and refuted (recorded so they are not re-litigated)

- **`isLocalSettlementPeak` is NOT the root cause.** The candidate pool *grows* at L7 (860 → 2207
  for seed 42), and two counterfactuals measured worse: an unscaled radius gives 4.5× (tracks cell
  count exactly) and a baseline-window rank rule 4.6×, against today's 2.57×. The current scaled
  radius is the least-bad of the three. **Do not "fix" it with a rank rule.**
- It *is* a real amplifier of the **kind mix**, though: the disc grows 18 → 60 cells against a fixed
  `+0.02` tolerance, and survival is strongly kind-selective (Local 14.9% → 5.5%, Regional
  62.4% → 51.0%). A cheap secondary option is scaling the tolerance with radius
  (`+0.02*peakRadius`), L5-bit-identical — but it changes L6 relative to L5, trading one
  equivalence for another. Prefer the percentile fix first.
- **The physical-support gate is completely inert** — `supportRejected = 0` at every level and seed,
  confirming the R2.1/R2.2 work is neutral here rather than over-correcting.
- Movement cost is resolution-invariant (mean 2.16 → 2.12, p50 1.79 → 1.79), so travel budgets are
  not themselves biased — only the node spacing they are measured against.

## Performance — measured 2026-08-02 (work in `~/projects/worldgen-plumbing`)

L7 validation was taking **~2.5 h/seed**, making the whole remediation programme hard to verify.
Profiled properly rather than guessed. **Terrain was never the problem** (~23 s/seed at L7 even
before fixes); the cost is almost entirely precipitation.

| stage | before | after | note |
|---|---|---|---|
| `ApplyLandmassErosion` (L7) | 14.62 s | 1.01 s | `checkDepth` base 15 → 4 (constant was L7-calibrated, so it was being quadrupled) |
| terrain pipeline (L7) | ~22.9 s | 7.65 s | + `ComputeCoastalExposure` map→stamped array (was 22.6% of terrain) |
| climate stage (L5) | 96.1 s | ~26 s | **3.7×**, no physics changed |
| ↳ `computePrecipitationBudget` | 81% / 88 s | 56% / 19.6 s | |
| ↳ upwind footprint BFS | 82.5 s | 15.8 s | 5.2× |

Profile findings: `computeUpwindFootprintWeights` alone was **75.9% cumulative** of the climate
stage, and Go **map machinery was ~45% of the entire stage** (`mapassign`/`makemap`/`memhash64`) —
the same per-call `map[int]…` anti-pattern found in terrain. Fixed with a pooled workspace using
stamped index arrays, reused donor scratch buffers, and a cached per-edge unit-direction table.
Also: `precipitation_land_flux.go` was traversing the *same* footprint twice (mean and max) — now
one pass. Only behavioural delta is that footprint summation is now deterministic insertion order
instead of randomized map order, which is strictly better for reproducibility.

Deliberately NOT changed: no scaled budget, depth, iteration count, or decay constant — all the
resolution-correctness fixes are behaviourally byte-identical.

### Scheduled next: make the footprint operator linear (the structural win)

**L7 is still not practical** — measured L5→L6 scaling is 14.4× for 4× the cells, matching the
structural cost `O(N · depth²)` (footprint depth scales linearly with resolution, BFS cone area as
depth²). That projects **~1.2–1.6 h/seed at L7** after the above: a real 3–4×, but still overnight.

The fix, identified but not attempted (too invasive for one session): **the footprint operator is
linear.** `accum_i = Σ_d c_d (P^d δ_i)` for a fixed transition matrix `P` and depth coefficients
`c_d`, and every production consumer except `upwindFootprintMax` needs only `Σ_x accum_i(x)·g(x)`
— the per-`i` normalisation cancels in the mean, and the ocean-support / fetch / uplift consumers
are linear too. So every cell's footprint mean can be produced by **`depth` global sparse passes:
`O(N·depth)` instead of `O(N·depth²)`**, exactly mathematically equivalent. That removes roughly
another 16× at L7 and puts an L7 seed in the **few-minute** range — i.e. it turns L7 (and L8) from
an overnight job into routine validation. High value; schedule it before the next full sweep.

## Validation log

| Date | Wave | Cache bumps | Sweep | Result |
|------|------|-------------|-------|--------|
| 2026-08-03 | Wave 1 + perf + stormband fix | terrain-v10 | L6/L7 3-seed (42,123,255) | **Divergence persists — see below** |

### Wave 1 sweep result (2026-08-03, `sweeps/level67_wave1_optimized`)

| seed | score L6 | score L7 | ratio | proto Δ | polities Δ |
|---|---|---|---|---|---|
| 42 | 7.70 | 2.24 | 0.291 | −3 | −2 |
| 123 | 0.47 | 0.00 | **0.000** | −2 | −2 |
| 255 | 2.11 | 5.54 | 2.626 | 0 | 0 |

**Verdict: wave 1 was necessary but not sufficient.** Ratios remain scattered (0.00 – 2.63); the
reported `mean_score_ratio` of 0.972 is an artefact of averaging 0.29, 0.00 and 2.63 and should not
be read as convergence. Seed 123 produces **no trade at all at L7** (score and volume both 0.00).

**The signature points squarely at Phase 2 R2.1.** Proto-civilization and polity counts are
*consistently lower at L7* (−3/−2/0 and −2/−2/0, never higher) — exactly the behaviour the
civilization audit predicted for `SettlementNodePhysicalSupportArea`: the support-area floor
collapses from 1.0 at L5 to 0.0625 at L7, so the `< 0.5` and `>= 0.75` gates are inert at baseline
and bite hard at high resolution, demoting nodes and starving region anchors. **That fix is still
TODO.** Fewer polities ⇒ fewer trade partners ⇒ collapsed trade score, which plausibly explains the
score ratios downstream of it.

Caveat on comparability: this run includes the `checkDepth` correction, which changes terrain output
at L5/L6, so these numbers are **not** directly comparable to the pre-fix 8-seed baseline (different
worlds). Read the ratios and the Δ signs, not absolute deltas against the old table.

**Timings** (the perf work, measured end to end): L6 ~330 s/seed, L7 ~6100–6400 s/seed (~1.7 h),
whole 3-seed sweep 5.5 h wall clock. Down from ~2.5 h/seed, so a real improvement, but L7 is still
not routine — the linear-footprint restructure above remains the item that would fix that.

### R2.1 + R2.2 result (2026-08-03, `sweeps/level67_r21_validation`, civilization-v66)

Support area now returns the supported *fraction of the physical window* rescaled to the L5
seven-cell reference, capped at one baseline cell per supporting cell (which keeps L5 bit-identical
including the twelve pentagonal cells). Measured on a hex-lattice mesh, an identical physical
neighbourhood now scores 7.00 / 7.00 / 7.00 at L5/L6/L7 (was 7.00 / 4.75 / 3.81); the R2.2 region
disc scores 18.00 at all three (was 18.00 / 15.00 / 13.50).

| seed | ratio before | ratio after | proto Δ | polities Δ |
|---|---|---|---|---|
| 42 | 0.291 | **0.595** | −3 → −2 | −2 → −2 |
| 123 | 0.000 | 0.000 | −2 → −2 | −2 → −2 |
| 255 | 2.626 | **0.843** | 0 → 0 | 0 → −1 |

**Partial success.** Excluding the degenerate seed 123, worst-case deviation from a 1.0 ratio fell
from **1.626 → 0.405** (~4×): seed 255 went from 163% over to 16% under, seed 42 roughly doubled
toward parity. **But proto/polity counts remain consistently lower at L7** (−2/−2/−1, never
positive), so the support-area cascade was a major contributor to the *trade-score* divergence but
is not the sole cause of the *civilization-count* gap.

Unexplained and worth understanding before further tuning: seed 42's **L6** score moved 7.70 → 4.27
and its L6 proto count 7 → 6, even though the fix gives L6 *more* support (4.75 → 7.00 for a full
window) and should therefore promote more nodes. That non-monotonic response suggests an
interaction — more cities altering spacing or claim geometry and yielding fewer regions.

Known limit of the fix: invariance is exact for window-scale features but sub-window features stay
quantized (an L5 window has only 7 samples, so partial fractions land on ~14% steps). The half-plane
case is 5.3% spread vs 48% before — the fix removes ~9/10 of the drift, not all of it.

Process note: five existing tests failed because they **encoded the bug** — they declared
`cellCount = 40962` while building star meshes whose 2-hop disc held 8–9 cells instead of the ~19 a
real L6 neighbourhood holds. Intent preserved, fixtures rebuilt on a hex lattice, no assertion
weakened.

**Next action: measure, don't guess.** Instrumenting `protoCivilizationRegionEligibilityReason`
(which already returns a reason string that every caller discards) to tally accept/reject reasons
plus mean driver values per reason, then diff L6 vs L7 for the same seeds. That converts the
remaining count gap from a hypothesis into a measurement, and the instrumentation is permanently
useful.
