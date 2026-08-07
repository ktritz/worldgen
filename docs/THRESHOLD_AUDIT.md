# Threshold Audit — absolute gates vs the fields they gate

Survey date: 2026-08-06. No fixes applied; this is a findings document.

## Why this audit exists

The same defect appeared five separate times during resolution work: **an absolute threshold applied
to a derived field that nobody had measured the distribution of.** It has two failure modes, both
invisible without printing the distribution:

- **Threshold above P90/max** → the feature is silently *absent*. The gate looks like a filter and
  is actually an off switch.
- **Threshold below P10** → the gate is *inert*, admitting nearly everything while appearing to
  discriminate.

The instigating case: the ocean candidate gate's physical deepwater floor was **0.24 against a field
whose P90 is 0.21–0.28 and whose maximum is 0.22–0.32** — above P90 in five of six seeds and above
the *maximum* in one. It produced 1 ocean corridor across six worlds. It survived indefinitely
because the only distribution ever printed was computed over *survivors*, so with one survivor every
percentile read the same value and looked healthy.

**Method for this audit**: measure over the full candidate pool, never over survivors. Report the
*fraction selected*, which is interpretable, rather than the threshold value, which is not.

## Findings — settlements / population / resources / routes / trade / ports

Measured at L6 and L5, seeds 42/123/255, from serialized cache structs.

### P0 — `candidatePortSuitabilityFloor = 0.29` is above P99 for deep-draft vessels

Gates cell-level `PortSuitability` in `qualifiesFallbackCoastalCandidate`
(`climgen/coastal_trade.go:383`, value in `config/coastal_trade_earthlike.json`).

| vessel | seed 42 | seed 123 | seed 255 |
|---|---|---|---|
| coastal-sloop | P99 0.404 → 15.3% | P99 0.338 → 8.9% | P99 0.345 → 11.6% |
| caravel | P99 **0.280** → **0.82%** | P99 0.242 → **0.59%** | P99 0.234 → **0.10%** |

For deep-draft hulls the fallback coastal-candidate path admits ~1 cell in 1000 — effectively off.
`PortSuitability` is rescaled per vessel, so **one absolute floor cannot be correct across the
vessel table**.

**Also grossly resolution-dependent**: fraction selected (coastal-sloop) goes L5 2.9%/2.7%/0.85% →
L6 15.3%/8.9%/11.6%, a 4–13× shift for one mesh level.

### P1 — `SettlementNodeCity` is never produced at L6

Node-kind histograms at L6: seed 42 `{hamlet 37, village 38, town 11, city 0}`, seed 123
`{32, 22, 9, 0}`, seed 255 `{45, 38, 13, 0}`. L5 seed 42 has exactly one city; every other measured
world has zero.

Any gate keyed on `SettlementNodeCity` or `MarketMinNodeKind: "city"` is dead by construction.
Nothing in the shipped goods catalog requires city today, so nothing is *broken* — but the city tier
is an empty feature class that both `coastal_ports_nodes.go` and `trade_goods_market_scope.go`
branch on.

### P1 — `marketMinNodeKind: "town"` halves its selected fraction between mesh levels

`settlementNodeEffectiveRank >= Town`: L5 20.8%/34.2%/17.1% of nodes → L6 11.6%/12.7%/10.4%.
**Ten of the thirteen manufactured goods** (iron_goods, weapons_armor, ships, paper, soap, woolens,
perfumes, fine_clothing, preserved_food, jewelry) gate on it, and their market coverage tracks it
exactly. **Manufacturing breadth is currently a function of mesh resolution.**

### P1 — the urban population class is ~9 cells per world

`urbanThreshold = 0.56` on `UrbanPotential` is above P90 (above P99 in two of three seeds). But the
class also requires `carrying >= 0.52` **and** `urban >= carrying + 0.02`; the conjunction selects
**11 / 6 / 9 cells out of ~11,750** = 0.09%/0.05%/0.08% of land. At L5 the same conjunction gives
0.069%/0.204%/0.202% — near-empty *and* resolution-shifting.

### P2 — the degree-0 prune escapes are all above P99, and they compound

`settlement_network.go:1306,1310` keep an isolated village at `Suitability >= 0.72` and a town at
`>= 0.70`. Measured P99 is 0.639/0.632/0.639 → selects 0.06%/0.04%/0.07%. The suitability escape
effectively never fires.

Its only backstop, `settlementResourceExceptional >= 0.35` (`settlement_network.go:1013`), is *also*
above P99 (P99 0.126/0.262/0.19) and selects ~1–5 cells per world. **Both escapes from the isolated
-node prune are near-dead, and they are each other's fallback.**

### P2 — major port thresholds are above P90 and structurally resolution-coupled

`MajorPortThreshold 0.64` → 9.4%/9.8%/3.3% (8/6/3 ports); `MajorDeepwaterPortThreshold 0.58` →
7.1%/8.2%/**2.2%** (6/5/2). At L5 seed 255 the deepwater gate is above P99 and yields **1 port**.

Above P90 is defensible for a "major" tier, but the *count* is unpinned and a world can land on 1–2
major deepwater ports — which is the input to the ocean-trade layer. Worse, both scores add
`tradeNorm`/`riverNorm` **normalized by the max over the network**, an order statistic that grows
with node count, so these thresholds carry the same defect as the coastal port scoring already
fixed. `LocalMinCentrality = 0.18` gates the same max-normalized centrality.

### P3 — `CoastalPortHarbor` is never produced in any seed, at any vessel

`determineCoastalPortType` (`coastal_ports.go:386`) takes an argmax of four candidates. Type
histograms: coastal-sloop `{BeachLanding 749, Harbor 0, Estuary 0, IslandStopover 0}`; caravel
`{BeachLanding 257, Harbor 0, Estuary 16, IslandStopover 40}`. The `landing` expression carries a
large constant floor that the harbor expression (`0.62*suitability + 0.38*harbor`, suitability P50
≈ 0.15–0.22) cannot beat. **A whole port class is unreachable** — an argmax collapse rather than a
threshold, but the same symptom.

### P3 — coastal resource floor above P90

`determineCoastalResourceType` floor 0.28 selects 11.5%/8.8%/~9% at L6 vs 7.6%/6.5%/4.4% at L5.
Sub-classes are marginal: saltworks 26/30 cells, open fishery 44/37, against estuarine fishery
366/295.

## Findings — hydrology / soil / vegetation / agriculture / water / biomes

Measured at L5/L6/L7, seeds 4/42/123, replaying the derived pipeline off terrain+climate caches.

### P0 — SIX more classes with a threshold above the field's MAXIMUM

Each is produced **zero times** across 7 worlds at 3 mesh levels.

| gate | field | field max (L5/L6/L7) | threshold | result |
|---|---|---|---|---|
| `soil.go:174` salinity | `Salinity` | 0.034 / 0.024 / 0.048 | **0.45** | `SoilSalineCoastal` never produced — threshold is 10–19× the field max |
| `vegetation.go:404` | `MangroveAffinity` | 0.314 / 0.373 / 0.398 | 0.48 | `VegetationMangrove` never produced |
| `vegetation.go:406` | `SaltMarshAffinity` | 0.362 / 0.366 / 0.383 | 0.50 | `VegetationSaltMarsh` never produced |
| `vegetation.go:410` | `CloudForestAffinity` | 0.441 / 0.452 / 0.467 | 0.60 | `VegetationCloudForest` never produced |
| `agriculture.go:181` | `CropPotential` | 0.643 / 0.645 / 0.644 | **0.74** | `AgricultureIntensiveCropland` never produced — **the top agricultural tier does not exist in any world** |
| `vegetation.go:412` | `RiparianAffinity` | 0.611 / 0.635 / 0.692 | 0.58 | ~2 cells/world clear it, none survive the `treeCover>=0.28` co-gate |

**One root cause for five of them.** These fields are products of 4–6 multiplied
`smoothstep01`/`peak01` factors. Each factor typically lands at 0.5–0.8, so the product lands at
0.2–0.4 — but the gates were calibrated as if the field were a weighted *average*, which would sit
near 0.6–0.8. Any product-of-factors field needs its gate derived from the product's actual range.

**A false-confidence trap worth internalising**: `climgen/vegetation_test.go` and
`climgen/soil_test.go` unit-test exactly these branches by hand-feeding values like
`mangrove = 0.8` — **a value the pipeline can never produce**. The tests pass, the branch is
verified, and the feature is dead. A unit test that constructs its own input cannot tell you the
input is reachable.

### P1 — near-absent classes

`SoilPeat` (organic ≥ 0.68 vs P99 0.719/0.498/0.306) falls to **0.001%** of land at L7.
`SoilOrganicWet`, `AlpineMeadow` (0.03–0.05%), `Peatland` (0–0.012%), `SoilAlluvial`
(3.06% → 1.52% → **0.62%**), `FloodplainCropland` (~0.1%), `Lake/Oasis` (0.79–1.36%, where two
`>= other − 0.04` co-conditions eat 85% of qualifying cells).

### P2a — our channel-initiation fix pushed the artifact into its 2-D consumers

`ChannelStrength = accumulation / channelThreshold`, and after the critical-drainage-area fix the
threshold is a rank pinned to `pathScale`. So **every absolute cut on `ChannelStrength` halves per
mesh level**: `channel >= 1.0` selects 6.52% → 3.25% → 1.63%, exactly ×0.5.

That is correct for the channel *centerline*, a 1-D feature. It is **not** correct for the 2-D
fields fed from it: `hydrology_resolution.go` deliberately does not spread
`RiparianChannelSupport`, so the riparian/floodplain corridor stays one cell wide at every
resolution and its *physical width* halves per level. Downstream: `SoilAlluvial` 3.06→1.52→0.62%,
`SoilPeat` 0.29→0.034→0.001%, `VegetationWetland` 4.29→2.62→1.81%, `BiomeWetland`
4.75→2.96→2.22%, soil `Fertility ≥ 0.40` 5.4→2.4→0.9%.

**This is a consequence of the channel fix**, and it compounds with P1: classes already sitting
near their gates get pushed under at high resolution. The centerline needs a physical-width spread
before its 2-D consumers are correct.

### P2b — THE PRECIPITATION FIELD IS NOT MESH-INVARIANT

`AnnualPrecipCm` P50 = **32.1 (L5) → 19.3 (L6) → 10.2 (L7)**. `AridityRatio` P50 = 1.29 → 0.82 →
0.42. The thresholds that consume them are fine; the field is not.

Consequence: **worlds get monotonically drier and more desert as the mesh refines.**
`classifySeasonalCell`'s fixed `aridityRatio < 0.50` desert cut selects 30.0% → 39.4% → **53.0%**
of land. `Hot Desert` 19.4→20.8→25.0%, `Cold Desert` 5.9→12.9→17.0%, `Arid Mineral` soil
25.5→29.1→35.5%, `Desert Sparse` vegetation 28.1→32.0→33.5%.

This is upstream of the audited area and **every biome, soil, vegetation and agriculture threshold
inherits it**. It is almost certainly the largest single remaining resolution defect in the
codebase, and it explains why threshold-level fixes downstream keep only partly working.

### Also noted

`climgen/biome.go`'s legacy Whittaker `ClassifyBiomes`/`classifyCell` (~25 hardcoded cuts) has **no
callers** — dead code, and a decoy for anyone auditing thresholds.

Healthy in this area: all `AnnualPrecipCm` biome cuts *at a fixed level*, temperature cuts, ice
affinity, grassland/tropical-wet/wetland affinities, all soil `Relief` cuts (notably **stable**
across levels — `computeLocalRelief`'s `stepScale` division is working), tree/grass/shrub/bare
cover, dry-farming and pastoral thresholds, river `MinNavigability`, `MinRunoff`, surface
reliability.

Inert (below P10): the `peak01(soilDrainage, 0.25, …)` left edge admits 99%, `fertility <= 0.45`
as a `SoilRocky` co-gate admits 98% (rockiness alone decides it), and `TerrainSuitability` is
saturated at P50 0.9987 so it contributes almost no discrimination to crop/pasture.

## Inert gates — below P10, admitting nearly everything

| gate | field | fraction selected |
|---|---|---|
| `candidatePortFeatureFloor 0.22` | `coastalTerminalFeatureScore` | **99–100%** |
| `coastal < 0.14` (`coastal_ports.go:389`) | `CoastalAccess` on port-diag cells | **100%** |
| `marketInputCapabilityFloor` / `localInputCapabilityFloor` | market supply/need ratios | selects *exactly* the `minKind=town` set in 10/13 goods — dominated by the node-kind gate; only jewelry, weapons_armor and perfumes ever bind |

## Checked and healthy

`candidatePortThreshold 0.42` (41–60%), `portSuitabilityFloor 0.18` (68–77%), `sparseThreshold 0.18`
(sits at the median), `ruralThreshold 0.34` (23–38%), `classifySettlementClass` 0.42/0.22,
`determineResourceType 0.38` (~29% of land gets a type), `node.Coastal 0.16` / `node.River 0.24`
(20–29%), `minStopoverValue 0.26` (22–28%), land-route slope limits (penalty only, never blocks),
and the post-fix ocean gates (8.8–18%, producing 5–8 corridors).

Borderline: `denseRuralThreshold 0.52` (2.0%/1.1%/1.4%, ~2× L5→L6 shift), `Prime 0.66`
(0.3–0.4%, 2× shift), hamlet degree≤1 keep at `AccessScore >= 0.58` (0.4–1.2%).

Not a bug: coastal-sloop shows zero ocean corridors because
`openOceanCapability 0.24 < minOpenOceanCapability 0.35` — ocean trade is correctly disabled for
that hull.

## Suggested fix order

1. Make `candidatePortSuitabilityFloor` **vessel-relative** (a quantile of that vessel's own
   `PortSuitability`) — currently above P99 for half the vessel table and swinging 4–13× by level.
2. Replace `MajorPortThreshold` / `MajorDeepwaterPortThreshold` with rank/quantile selection, and
   de-max-normalize `tradeNorm`/`riverNorm`.
3. Recalibrate `urbanThreshold` and the `urban >= carrying + 0.02` tiebreak — the urban class is
   ~9 cells per world.
4. Rebalance `determineCoastalPortType`'s `landing` expression so `Harbor` is reachable.
5. Drop or re-derive `candidatePortFeatureFloor` and the `coastal < 0.14` gate — both admit ~100%.
6. Re-derive the degree-0 prune escapes (0.72 / 0.70 / 0.35), all above P99 and mutually dependent.
7. Decide whether the city tier should exist. If yes, `CityThreshold` needs recalibrating; if no,
   remove the branches that pretend it does.
