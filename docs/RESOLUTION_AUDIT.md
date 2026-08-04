# Resolution-Independence Audit

Audited: full working tree, 2026-08-02. Six parallel subsystem audits (climate fields; hydrology/land cover;
civilization; trade routing; trade goods; terrain/mesh). Reference pattern: `climgen/resolution_scale.go`
(`meshPathCostResolutionScale`, `meshResolutionAdjustedSteps`, `meshScaledTerritoryLinearCells/AreaCells`) and
the fixes in commits `81137ae` / `8827840`.

Mesh levels: L4 = 2562 cells (~446 km), L5 = 10242 (~223 km, baseline), L6 = 40962 (~112 km), L7 = 163842 (~56 km).
`meshPathCostResolutionScale`: L4→1.0 (clamped, true value 2.0), L5→1.0, L6→0.5, L7→0.25 (exactly the lower clamp).

Key scaling rule discovered during the audit: **iterative neighbor-averaging (diffusive) smoothers spread
σ ≈ cellSize·√iters, so holding a fixed physical smoothing radius requires iterations ∝ N (∝ 1/scale²).
Directional/advective propagation (1 cell per pass) spreads iters·cellSize and needs iterations ∝ 1/scale.**
Most existing `int(targetAngle/cellSize)+1` patterns use the advective (linear) rule on diffusive smoothers.

---

## 0. Systemic

### 0.1 `meshPathCostResolutionScale` clamp `[0.25, 1.0]` — flagged independently by all six audits
`climgen/resolution_scale.go:12-17`
- **Upper clamp 1.0:** L4's true scale is 2.0, clamped to 1.0, so *no coarse-mesh correction is ever applied*.
  Every hop budget, step cost, and territory-area conversion treats L4 as L5 while L4 cells are 2× wider / 4× larger.
  Consequences: travel/claim budgets reach 2× the physical distance at L4 (oversized polities, over-connected
  networks), `soil.go` relief ~2× too steep (rockier terrain classification, higher route costs), coastal/hydrology
  halos 2× too wide, territory areas over-reported 4×.
- **Lower clamp 0.25 = exactly L7:** L7 is fine; L8+ (655362) saturates — maritime budgets truncate (`NoPath`
  explodes), hop budgets halve physically.
- **Decision required:** either (a) support L4: widen upper clamp to 2.0 and re-verify the
  `meshResolutionAdjustedSteps` floor (`if steps < baseSteps { return baseSteps }` blocks downward adjustment),
  or (b) document L5 as the coarsest and L7 as the finest supported mesh and assert at pipeline entry.
- `landgen/terrain/resolution_scale.go` mirrors these helpers and the same decision applies there.

### 0.2 Diagnostics are themselves resolution-dependent
Tuning gates measured at mesh scale will mis-direct tuning across resolutions (see §7 and terrain §6.10).
Fix or reinterpret these before using them to validate the other fixes.

---

## 1. Trade routing & trade goods

### 1.1 HIGH — `tradeLinkTravelCost` uses a raw cell count (the one unscaled Dijkstra left in the trade stack)
`climgen/trade_network_paths.go:147-172` (esp. `:171`): `count * meanCost * (1 + 0.55*meanRisk + 0.22*(1-meanSupport))`
where `count` = number of usable cells in `link.Path`. River/coastal/ocean paths all apply `stepScale`; the
fallback branch in this same function uses already-scaled `SettlementLink.TravelCost` — mixed units in one function.
Edge costs accumulate against absolute budgets (`MaxRouteCost` 28.0 × reach multipliers).
- Symptom: land corridor costs inflate 2× at L6 / 4× at L7 → fewer trade corridors, collapsed hubs, depressed
  centrality → fewer major ports; flows shrink (`flow ∝ 1/cost`). L4 opposite.
- Also the root cause of the land-mode collapse in `routeGoodCapacity`
  (`trade_goods_multimodal_routes.go:288-297`): `CapacityScaleByMode` divisors (land 17 etc.) are
  L5-calibrated absolute cost scales; land friction doubles/quadruples with resolution while water modes stay
  physical → mode mix skews to shipping at high res.
- Fix: multiply by `meshPathCostResolutionScale(len(cells))` in `tradeLinkTravelCost` (thread cell count from
  `trade_network.go:130`). Fixing it at the source also fixes `routeGoodCapacity`.

### 1.2 HIGH — per-hop score penalty in river terminal selection
`climgen/river_trade_terminals.go:143`: `score -= 0.035 * float64(distance)` (hops) mixed with [0,1] reward terms;
catchment radius correctly scales 1→2→4 so max penalty grows 0.035→0.14 while rewards don't.
- Symptom: at L7 distant-but-good river cells fail the `score < minNav*0.48` gate → fewer river terminals,
  corridors, ports. Fix: `* meshPathCostResolutionScale(len(cells))`.

### 1.3 MEDIUM — max-statistic over catchment disks with growing sample counts
`climgen/coastal_ports_nodes.go:275-336`: radius and decay are correctly scaled, but the disk holds ≈126 cells at
L5 vs ≈1800 at L7; `best = max(...)` over 14× more samples is systematically higher, compared against absolute
port thresholds (0.64/0.58/0.42/0.18).
- Symptom: monotonically more ports qualify at higher res. Fix: scale-weighted top-percentile or area-weighted
  mean instead of raw max; or make thresholds sample-count-aware.
- Same pattern: `trade_network_feeders.go:82,92,113,412-424` (top-3 feeder nodes from a candidate pool that grows
  ∝1/scale²; positions drift with resolution) — LOW-MED.

### 1.4 MEDIUM — port dedupe by terminal cell identity
`climgen/coastal_trade.go:337-359`: suppression keyed on shared terminal *cell index*; cells shrink with
resolution so collisions vanish at L6/L7 → candidate port count grows super-linearly, changing which corridors
win the fixed partner caps. Fix: dedupe by great-circle separation or scaled hop radius.

### 1.5 Smaller / latent
- `river_trade_terminals.go:41-53` reimplements `meshResolutionAdjustedSteps(1, n)` with magic constants —
  replace with the helper (latent divergence).
- `river_trade_paths.go:78-119, 289-341` — dead node-graph path code carries the §1.1 unit bug (no non-test
  callers). Delete or fix.
- `ocean_trade.go:411,431-433` — global `MaxStopovers=56` top-K over a pool whose scores inflate with resolution;
  basin coverage composition shifts. MED-LOW.
- `coastal_ports.go:255,353` — scaled radius returns a filled disk at L6/L7 vs the 1-ring at L5; disk-vs-ring
  averaging shifts `oceanFrac`/`landFrac` vs absolute smoothsteps. MED-LOW.
- `trade_network_feeders.go:163` — spacing rule inert at L5 (`minHops<=1`), active only at L6/L7. LOW.
- `coastal_trade_endpoint_paths.go:116` / `ocean_trade_paths.go:53` — prefilter `minStepCost` literals duplicate
  the step-cost floors; promote to shared constants. LOW.
- Trade goods LOW items: feeder aggregation into `market.Supply` unnormalized (`trade_goods_markets.go:107-120`);
  feeder-count constants 4/8/6 (`trade_goods_markets.go:247,272`, `trade_goods_multimodal_routes.go:308-314`);
  scarcity `peak` = max over all cells — use p99 (`trade_goods_scarcity.go:472-499`);
  `endpointSinkCapacityByPolityGood` and external-input support unnormalized sums (inert with default settings).

### Verified correct (trade)
Maritime/ocean/river/feeder cell Dijkstras (stepScale applied once, correct arrays); `openWaterHopAllowance`
capped before scaling; stopover spacing hops × mean neighbor degree (invariant); `maritimeLandComponentAreaEq`
uses scale² correctly; `maritime_endpoint_prefilter.go:46` lower bound consistent; `nodeCatchmentPotential`
(`trade_goods_nodes.go:216-259`) is the textbook-correct catchment: scaled hop budget, physical-distance decay,
weighted mean.

---

## 2. Civilization / settlements / polities

### 2.1 HIGH — `SettlementNodePhysicalSupportArea` fixed-hop disc is not resolution-stable (root of a 6-finding cascade)
`climgen/settlement_network.go:397-409`: counts supporting cells in a `meshResolutionAdjustedSteps(1,n)`-hop disc,
converts with `meshScaledTerritoryAreaCells`. The disc's baseline-equivalent area and especially its *floor*
(the unscaled `+1` self-seed) collapse: range [1.0, 7.0] at L5 → [0.0625, 3.81] at L7 (16× floor drop). All
absolute thresholds (0.5, 0.75, 1.0, 1.5) were tuned against the L5 floor of 1.0.

Cascade (all HIGH, all inherit this):
- `settlement_network.go:351` — `< 0.5` town/city gate: vacuous at L4/L5, binding at L6/L7 → silent demotions.
- `settlement_network.go:374-395` — `physicallySignificantCityCandidate`: `>= 1.0` auto-true at L5, needs 16/62
  cells at L7 → cities nearly vanish at L7, over-produced at L4/L5.
- `settlement_network.go:418-426` — support weight: always exactly 1 at L4/L5 (mechanism inert), 0.35–0.6 at L7
  → link ranges, node degree caps, influence/claim budgets, secondary capitals all shift
  (`proto_civilizations.go:429,472`, `polity_spheres.go:162,222,257`).
- `proto_civilizations.go:253` — `>= 0.5` regional-anchor gate: same asymmetry → fewer proto-civs at L7.
- `proto_civilizations.go:323-324` — `min(area,1.0)` saturates at L4/L5, real shrinking discount at L6/L7 →
  monotonically fewer civilizations as resolution rises; smaller territories via AnchorCount.
- Fix at the source: compute a *fraction* of the disc (`supported / discSize`) rescaled to the L5 reference
  (×7.0), or accumulate real cell areas normalized by baseline cell area.

### 2.2 HIGH — polity border pressure counts unscaled cell-adjacency pairs
`climgen/polity_profiles.go:191-218`: border-pair count is a linear measure ∝ sqrt(cellCount) (2× at L6, 4× at
L7), compared to the absolute cap 12.
- Symptom: at L6/L7 every adjacent polity pair saturates → `borderAdj = 0.36` uniformly → alliances impossible,
  attitudes pinned Wary/Hostile; at L4 borders under-register.
- Fix: `count × meshPathCostResolutionScale(len(cells))` before the cap.

### 2.3 MEDIUM — region population-support discs
`proto_civilizations.go:329-375,193`: r=2 disc baseline-equivalent area shrinks ~25% L5→L7 vs the hard `9.0`
constant → "area-support" escape hatch fires less at L7, compounding 2.1.

### 2.4 LOW — `selectWaystationCells` path-length floor in hops
`settlement_network_waystations.go:107`: `len(path) < 4` (physical spacing guard mostly covers this).

### Verified correct (civ)
Settlement spacing in degrees; `populationCatchmentSupports` (scaled radius + mean); settlement region travel
budgets (scaled Dijkstra); polity territory comparisons via `meshScaledTerritoryAreaCells`;
`polityProtoTerritoryAreaEq` cell-count source verified.

---

## 3. Temperature / wind / currents

### 3.1 HIGH — heat diffusion has no dx² normalization (biggest temperature finding)
`climgen/temperature_heat_transport.go:129` (dup `temperature_diffusion.go:64`): `D * (neighborMean − T)` is a
discrete Laplacian ≈ dx²·∇²T, so effective physical diffusivity ∝ D/N — while the advection terms in the same
file (lines 169/210) correctly divide by cell size.
- Symptom: poleward/ocean heat diffusion 4× weaker at L6, **16× weaker at L7** → much sharper equator–pole
  gradients and SST fronts; L4 over-smoothed.
- Fix: `D × (N/10242)`, i.e. `D / scale²`.

### 3.2 HIGH — `ApplyMarineInfluence` never propagates past the first ring (also a plain logic bug)
`climgen/temperature_continentality.go:158-186`: the loop reads the *original* `temperature[k]` from ocean
neighbors only, so land ≥2 hops from the sea never receives signal; `distanceCells` grows with resolution, so
more blending is applied to a physically thinner strip.
- Symptom: at L7 maritime moderation is a 56 km band applied 9× over; interiors go continental. This is the
  fallback path when `wind.MarineWind` is absent.
- Fix: propagate through `influenced[k]` (or use the `spreadPhysicalMaxSignal` pattern).

### 3.3 HIGH — lee shadow decays per hop
`climgen/wind_orographic.go:320` with `decayRate = 0.75` per hop (`wind.go:406`): e-folding ~3.5 hops at any
resolution → shadow length 780 km (L5) → 195 km (L7). Fix: `pow(decayRate, stepScale)`.
Compounded by `wind.go:398-404` (dup `wind_seasonal_debug.go:236-242`): `leeShadowIters` clamp `[3,10]` forces
shadows 2–4× too long at L4/L5 and will bite again at L8+.

### 3.4 HIGH — fixed smoothing iterations on seasonal tropical wind/pressure fields
`wind_seasonal_tropics.go:99,108` (literal 2) and `:13-14,227-234` (5/4 iterations): monsoon/ITCZ pressure
anomaly smoothed over σ≈500 km at L5 but ≈125 km at L7 → stronger, fragmented local monsoon winds at high res.
Fix: iterations ∝ N for a fixed physical σ.

### 3.5 MEDIUM — diffusive smoothers scaled linearly (wrong class) with floors that dominate at coarse res
`wind.go:329-332, 412-415`, `wind_marine.go:28-31`, `wind_pressure_context.go:70-73`,
`temperature_currents.go:250-252` (SST source smoothing σ: 668 km L5 → 335 km L7),
`currents.go:100-105, 152-156`, `currents_windcurl.go:103-106, 171-174`,
`currents_streamfunction.go:176-179` (floor of 2 → damping band 2.3× too wide at L5, 4.7× at L4).
All use `int(angle/cellSize)+1` (advective rule) on neighbor-averaging (diffusive) smoothers.
Fix class: iterations ∝ N, or single-pass physical-radius kernels.

### 3.6 HIGH — `maxRings := 3` coast-parallel constraint band
`climgen/currents_boundary.go:269`: drives `EnforceCoastParallelFlow`, boundary-jet shaping, `CoastNormalP95`.
Band is 670 km at L5, 170 km at L7, 1340 km at L4. Fix: `meshResolutionAdjustedSteps(3, n)`.

### 3.7 HIGH — ocean component gates compare raw cell counts to constants
`climgen/currents_components.go:140-146`: `len(component.Vertices) < 8`, `(count−24)/232` — cell count ∝ N for
fixed physical area (16× L5→L7) → inland seas get full open-ocean gyre currents at high res, near-zero at low.
Fix: `meshScaledTerritoryAreaCells` before comparison. Related: `DefaultMinComponentSize = 50`
(`types.go:25`, consumed `currents_components.go:98`, `currents_basins.go:95`, `generation.go:39,137`) — same
issue for basin filtering (~64× physical swing L4→L7).

### 3.8 MEDIUM-HIGH — absolute threshold on a cellSize-scaled dot product
`currents_components.go:255`, `currents_boundary.go:71,174`: `Dot(toNeighbor, east) > 0.001` where
`|toNeighbor|` ≈ angular cell size → east/west coast classification gets stricter with resolution → Sverdrup
forcing seeds sparsen. Fix: normalize `toNeighbor` first.

### 3.9 MEDIUM-HIGH — 1-ring structural detectors
- `currents_gateway.go:31-113`: strait/gateway detection sees only straits narrower than ~one cell → 200 km
  straits register at L5, read as open water at L7; gates jet corridors and gyre guidance.
- `currents_structure.go:38-55`: openness neighbor-ratio is a 1-cell coastal indicator vs absolute thresholds.
- `wind_orographic.go:67-118`: orographic deflection barrier detection is 1-ring only — ranges stop deflecting
  flow from a distance as resolution rises.
Fix class: evaluate over `meshResolutionAdjustedSteps(1, n)` neighborhoods or physical-radius BFS.

### 3.10 MEDIUM (latent) — `ComputeWindwardness` threshold on cellSize²-scaled slope
`wind_orographic.go:399-419`: `Min(slopeLen*1000, 1)` where `slopeLen ∝ cellSize²` (16× regime shift L5→L7).
No in-tree caller today. Fix before wiring up: normalize to a true m/km gradient as `ApplySlopeEffects` does.

---

## 4. Precipitation & seasonal

Well done already: `precipitation_marine_sweep.go` per-step factors (`pow(f, stepScale)`),
`precipitation_ocean_transport.go`, `precipitation_maritime_access.go`, `coastalOnshoreScore`
(`seasonal_precipitation.go:207-247`), `runoff_seasonal.go` (fully intensive, completely clean).
The findings below are the parts that pattern missed.

### 4.1 HIGH — upwind footprint decay per hop
`precipitation_upwind_footprint.go:7,59`: `pow(0.84, depth)` is the kernel behind every upwind footprint
(orographic lift, ocean fetch, tropical marine source, frontal transport). Callers scale `maxDepth` but the
weights die 4× faster per km at L7, canceling the scaled budget.
Fix: `pow(0.84, depth × stepScale)`.

### 4.2 HIGH — hardcoded footprint depths
`precipitation_budget.go:549` (depth 3 in the core advection operator), `precipitation_frontal_reservoir.go:389`
(2), `:430` (4), `precipitation_land_flux.go:84,94` (4) — vs the correctly-wrapped call sites at
`precipitation_wind_geometry.go:97`, `precipitation_maritime_access.go:175`.
Symptom: advection/frontal/tropical kernels 4× more local at L7 → interior and monsoon rainfall collapse.
Fix: `resolutionAdjustedPrecipSteps(d, n)`.

### 4.3 HIGH — fixed iteration counts in moisture reservoirs
`frontalLandDiffusionIterations = 3`, `frontalStormTransportIterations = 2` (advective — literally caps frontal
storm reach at 2 hops: 450 km L5 → 112 km L7), `marineLandDiffusionIterations = 2` (the one uncorrected per-step
quantity in the marine sweep), `precipConvergenceSmoothIters = 4` (ITCZ band width).
Fix: advective → `meshResolutionAdjustedSteps`; diffusive → iterations ∝ N.

### 4.4 HIGH — iteration floor makes moisture penetration depth resolution-dependent
`precipitation_budget_setup.go:10-18` + `precipitation_budget.go:6-8`: intended relaxation depth 18×70 = 1260 km,
but the floor of 18 forces 4000 km at L5, 8000 km at L4 (correct only at L7). Interiors dry out as you refine.
Also (MED-HIGH, X2): the iteration budget is scaled against a 70 km baseline while transport steps use the
223 km `resolution_scale.go` baseline — two inconsistent references; pick one.
Fix: remove the floor (loop already early-exits on convergence) and unify the baseline.

### 4.5 MEDIUM-HIGH — orographic rise correction uses the wrong exponent, both directions
`precipitation_wind_geometry.go:78,113`: `rise /= sqrt(stepScale)` — the 1-hop local rise needs `/stepScale`
(under-corrected: halves at L7), while the footprint rise spans a fixed physical distance and needs *no*
correction (inflated 2× at L7). Net: orographic contrast roughly halves L5→L7.
Fix: `localRise /= stepScale`; leave `footprintRise` alone.

### 4.6 MEDIUM-HIGH — per-iteration source/condensation fractions unscaled in the land budget loop
`precipitation_budget.go:256-257` (`precipLandSourceFraction` 0.020, recycling 0.12/iter),
`precipitation_land_condensation.go:121` (supersat 0.68/iter) — while the marine transfers in the same loop
(`:216-239`) correctly use `precipitationPerStepFraction`. The budget fixed point becomes iteration-count
dependent → resolution-dependent land/marine precipitation split.
Fix: wrap all three in `precipitationPerStepFraction/Factor`.

### 4.7 HIGH — seasonal storm-band / monsoon field smoothing fixed
`seasonal_stormband.go:9,29,81,176` (4/2/6 iters), `seasonal_tropical_monsoon.go:7,229-233` (5/4/4/3/2),
`seasonal_tropical_convergence.go:7-8,81-87` (6 iters — deliberately physical spreading per its own comment).
σ ≈ 550 km at L5 → ≈137 km at L7: storm-band support and monsoon/ITCZ placement collapse toward coasts and
fragment. Fix: iterations ∝ N.

### 4.8 HIGH — storm memory: fixed advective hops with unscaled per-hop carry
`seasonal_stormband.go:11-12,216-240`: 3 hops × carry 0.85/hop → inland reach 670 km L5 → 168 km L7, applied to
three fields. Fix: `meshResolutionAdjustedSteps(3,n)` + `pow(0.85, stepScale)`.

### 4.9 MEDIUM — hop-harmonic weights where physical distance intended
`precipitation_transport_path.go:80`, `seasonal_stormband.go:372`: `weight = upwindness/(step+1)` with scaled
step budgets → decay 4× faster per km at L7. Fix: `/(1 + step×stepScale)`.

### 4.10 MEDIUM-LOW — wind convergence diagnostic gain
`precipitation_wind_geometry.go:274`: fixed `2.1` gain and ±1 clamp on a divergence whose distribution sharpens
with resolution (mitigated by fixing 3.5). Prefer percentile normalization as in
`currents_windcurl.go:186`.

---

## 5. Hydrology / biome / resources / maritime influence

### 5.1 HIGH — maritime influence compounds per-hop wind factors
`climgen/maritime.go:347` (with `:231,266,309,349,398`): only `decayPerStep` is physically normalized; the
sub-unity `downwind × windSpeed` multipliers apply once per hop, and absolute cutoffs (0.001) truncate the
accumulation. Influence reaches ~890 km inland at L5 but ~390 km at L7.
- Symptom: maritime temperature moderation collapses toward the coast at high res → continental interiors,
  cascading into continentality, biome splits, and everything downstream.
- Fix: per-distance factor `pow(downwind×windSpeed, cellSizeKm/referenceCellKm)` (or restructure as a
  physical-distance spread); scale the 0.001 cutoffs.

### 5.2 HIGH — basin component size threshold in raw cells
`climgen/types.go:25` `DefaultMinComponentSize = 50` (see also §3.7): 2.0% of the sphere at L4 vs 0.031% at L7.
Fix: `meshScaledTerritoryAreaCells(len(component), len(elevation))` vs baseline 50.

### 5.3 MEDIUM — `DefaultSmoothingIterations = 2` (`types.go:35`)
Fixed neighbor-averaging count for `SmoothCurrents`; the live streamfunction path overrides it with
angular-scaled counts, so impact is limited to the legacy path and external JSON settings. Delete or derive.

### 5.4 MEDIUM-HIGH — vorticity diagnostic carries 1/length units
`types.go:617-632`: unit-normalized `edgeTangent` makes `circulation/edgeSum` ∝ 1/cellSize; `AvgVorticity` /
`VorticityRatio` ~2× per level. Diagnostics only. Fix: use the raw edge vector.

### 5.5 Cross-file — `CalculateCoastlineLandDirs` `maxRings := 3`
Same site as §3.6. Coastal-resource upwelling detection is insulated (only consumes 1-ring cells); currents are
the real consumer.

### 5.6 NOT RESOLUTION, BUT NEUTERS MANY FIELDS — runoff units mismatch
`cmd/review_planets/pipeline_physical_helpers.go:261` copies `scaffold.Runoff` (a clamped 0.08–1.35 moisture
proxy from `landgen/terrain/erosion.go:343`) into `HydrologyBiomeInputs.Runoff`, which ~25 consumer sites
threshold as cm/yr (e.g. `smoothstep01(20,120,runoff)`): soil alluvium, wetland affinity, water reliability,
irrigation, placer deposits, flood risk, river navigability. With inputs ≤1.35 these all evaluate to **exactly 0
planet-wide**. (`ChannelStrength` is fine — percentile-normalized upstream.)
Fix: convert the proxy to the cm/yr domain, or retune the thresholds.

### Verified correct (hydro/land)
`hydrology_resolution.go` is the reference implementation (scaled radius + physical-distance kernel; the
deliberate non-spread of `RiparianChannelSupport` is right). `soil.go` relief divides by stepScale (subject to
§0.1 at L4). `vegetation.go` coastal exposure is a ratio of identically-kerneled sums (invariant).
`resources.go` per-cell classification fractions are invariant by construction. `biome.go`/`biome_seasonal.go`
threshold physical units. `land_routes.go`/`river_routes.go` are per-cell cost fields with no hop budgets.

---

## 6. Terrain / mesh (landgen, icosphere, procnoise)

`landgen/terrain/resolution_scale.go` already mirrors the climgen helpers but is used **only in erosion.go**.
Plate layout is the model to follow (fixed reference L6 mesh, physical seed points, angular distances;
boundaries.go widths all in radians). Erosion is fully scaled except the items below.

### 6.1 HIGH — hotspot island peak caps scale with sqrt(cellCount)
`landgen/terrain/hotspots_elevation.go:328,507,513` and `:722-723`: `sizeCap` divides a physical radius by the
*actual* cell angular radius → caps ≈1400 m at L5 vs ≈3200 m at L7 (islands squashed at L4). Directly moves
`MaxElevation` and hypsometry tails. Fix: normalize against the baseline cell radius `2/sqrt(10242)` or cap on
physical radius.

### 6.2 HIGH — continental caldera spread broken both directions
`hotspots_elevation.go:830-835`: `spreadDepth = int(targetRadius/cellAngularRadius)` truncates to 0 at L4/L5
(feature absent) and hard-caps at 2 hops at L6+ (shrinking physical footprint). Fix: radial angular-distance
selection like `spreadOceanicElevation`.

### 6.3 HIGH — abyssal-hill bands have wavelength in hops
`elevation.go:650-653`: `sin(distFromRidge × freq)` where dist is BFS hops → 250 km bands at L5, 60 km noise at
L7, continent-ripples at L4. Fix: multiply by `meshPathCostResolutionScale(n)`.

### 6.4 HIGH — drainage-sink breaching thresholds are per-hop metres
`erosion.go:283,300`, `drainage_metrics.go:103`: fixed `maxRise` (220/120/260 m) against an adjacent-cell
elevation difference ∝ cellSize → 4× steeper effective slope threshold at L7 → endorheic basins/lakes
progressively erased as resolution rises (same file normalizes slope correctly elsewhere). Fix:
`maxRise × meshPathCostResolutionScale(n)`; also the fixed carve depths at `erosion.go:420-423`.

### 6.5 HIGH — coastal exposure BFS depth fixed at 2 hops
`elevation.go:845`: exposure window shrinks 4× by L7 → systematic shifts in coastal noise amplitude and
coastline regularization. Fix: `meshResolutionAdjustedSteps(2, n)`.

### 6.6 MEDIUM — fixed noise octave count vs mesh Nyquist
`generate.go:179` → `elevation.go:795`: octaves 3–6 are aliasing at L5, coherent at L7 → hypsometry invariant
but neighbor correlation (local relief, drainage character) is not. Fix: cap octaves at mesh Nyquist,
renormalize variance.

### 6.7 MEDIUM — `RegularizeCoastlines(…, 2)` fixed iterations
`generate.go:180`: decide intent — physical smoothing (scale it) vs de-serration (document as cell-relative).

### 6.8 MEDIUM — fixed hop floors on otherwise self-normalizing ratios
`elevation.go:467-468,637,642,818,864`: the ×0.06/0.10/0.18 hop-ratio terms are invariant; the `max(2.0, …)` /
`max(3.0, …)` floors bind at L4 (shelf forced ≥ ~1400 km). Fix: divide floors by scale or express in radians.

### 6.9 MEDIUM — `majorThreshold` floor of 60 cells
`drainage_metrics.go:245`: floor dominates below ~20k land cells → "major basin" means 8% of land at L4 vs 0.3%
at L7. Fix: scale the floor by area or drop it.

### 6.10 MEDIUM-HIGH — evaluation metrics measured at mesh scale
`spatial_metrics.go:116-158,169-220` vs fixed Earth targets in `evaluation.go:42-43`: 1-ring relief ∝ cellSize;
mesh-edge coastline length is fractal → the same planet scores differently per level; L5 tuning fails the gates
at L7. Fix: relief per unit distance; fixed-physical-ruler coastline measurement.

### 6.11 Notes
- `boundaries.go:285`: arc seeding iterates a Go map (randomized order) into a greedy spacing filter →
  non-deterministic for fixed seed. Sort first. (Determinism bug, not resolution.)
- `erosion.go:169`: `checkDepth` scales to 60 at L7 with a per-land-cell BFS → `ApplyLandmassErosion` ~100×
  L5 runtime at L7. Performance cliff to know about before enabling L7.
- `plates.go:273,457`: 3 majority-vote smoothing iterations — near-self-scaling (removes mesh-scale artifacts);
  leave unless boundary raggedness matters. LOW.
- Production path calls `CreateIcosphere` without `RelaxMesh` (`cmd/review_planets/main.go:88`) — consistent
  across levels, not a resolution issue, but noted.
- icosphere/, procnoise/, hypsometry, base_elevation/continental_mask legacy path: clean.

---

## 7. Climate diagnostics (gate reliability)

- `climate_diagnostics.go:335-341,451-457,259-262`: 1-ring residual anomaly detectors vs absolute thresholds —
  anomaly fractions fall monotonically with resolution; spike flags go silent at L7 regardless of quality.
- `climate_diagnostics_helpers.go:102-112`, `climate_diagnostics_ocean.go:136-146`: "coastal" = 1 ring
  (223 km at L5, 56 km at L7) → `CoastalWetnessRatio`, `ColdInteriorMean`, coupling correlations all shift.
- `climate_diagnostics.go:513-527`: orographic contrast samples ±1 ring across a peak — at L7 both samples sit
  on the same side; metric →1.0 while the physics (§3.3, §4.5) genuinely degrades → the two mask each other.
- `GatewayFraction` inherits the 1-ring gateway detection (§3.9).
Fix class: fixed-physical-radius sampling (`meshResolutionAdjustedSteps(1, n)`), or stepScale-scaled thresholds.

---

## Suggested fix order

1. **Decide the resolution support envelope** (§0.1) — the clamp decision changes several fixes.
2. **Mechanical high-confidence unit fixes** (small diffs, big effect):
   `tradeLinkTravelCost` (§1.1), river terminal penalty (§1.2), lee-shadow decay (§3.3), upwind footprint decay
   (§4.1), hardcoded footprint depths (§4.2), storm memory carry (§4.8), hop-harmonic weights (§4.9),
   `maxRings` (§3.6), component gates (§3.7, §5.2), border pressure (§2.2), terrain items §6.1–6.5.
3. **Structural fixes**: settlement support-area disc (§2.1 — unblocks the whole civ cascade),
   temperature diffusion dx² (§3.1), `ApplyMarineInfluence` propagation (§3.2), maritime per-hop factors (§5.1),
   precipitation iteration floor + baseline unification (§4.4), per-iteration budget fractions (§4.6),
   orographic rise exponents (§4.5).
4. **Smoothing-iteration policy**: introduce a shared "diffusive iterations for physical σ" helper
   (iterations ∝ N) and route §3.4, §3.5, §4.3, §4.7 through it.
5. **Diagnostics** (§7, §6.10) so cross-resolution validation is trustworthy.
6. **Validation harness**: generate the same seed at L5/L6 (and L4 if supported) and diff invariants —
   land fraction, hypsometry, precipitation totals and interior/coastal split, biome area fractions,
   civ/settlement/port counts, trade corridor counts and mode mix, polity attitude distribution.

## Bonus non-resolution bugs found along the way

- Runoff units mismatch nulls ~25 consumer sites (§5.6) — likely the highest-impact single bug in the audit.
- `ApplyMarineInfluence` reads the wrong array — marine signal can't propagate past 1 ring (§3.2).
- Arc-seed non-determinism from map iteration order (§6.11).
- Dead-but-buggy river node-path code (§1.5).
