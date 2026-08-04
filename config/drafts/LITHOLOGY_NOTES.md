# Lithology layer — derivation notes (draft)

Companion to `config/drafts/lithology_earthlike.json`. Not wired in. Rule: every class is **scored from
fields the pipeline already computes**; nothing is hand-painted.

## 1. Inputs — what exists, and the plumbing gap

`landgen/terrain/generate.go` already computes, as **local variables**, everything assignment needs:
`rPlate`, `plateIsOcean`, `distFromCoast/Ridge/Trench/Arc/Collision/Mountain` + normalizers, and
`BoundarySeeds` distinguishes `Rift/Collision/Arc/Trench/Ridge/Coastline`. **None of it escapes** —
`PlanetGenerationDiagnostics` (types.go:530) exports only `HotspotChains`+`Hydrology`, and
`cmd/review_planets/pipeline_physical_helpers.go` gives climgen two hotspot arrays. Gaps:

- **A (plumbing only).** Add a `Tectonics` block to `PlanetGenerationDiagnostics` — `rPlate`,
  `plateIsOcean`, the distance rasters, per-cell `BoundaryType`. This is the whole blocker.
- **B (one line).** `Rift` seeds are found but never made into a distance raster. Needed by
  `evaporite_basin`, `basalt_province`, rift `sandstone_redbed`.
- **C (missing).** No **crustal age / craton** field. Proxy: continental, far from all boundaries, low
  `Relief`. Cheap real fix: a seeded per-plate `plateAge ∈ [0,1]`, else every world's shields look alike.
- **D (missing).** No **deposition/basin-fill** field — `ApplyFluvialErosion` moves material and records
  nothing. A `depositionAccum []float64` written in that pass is the most valuable new field: it
  separates *basin* from merely *flat*, which four sedimentary classes need. Interim proxy: hydrology
  `TerminalSinks`/`WaterBodyLabel`/`CellClass` + low relief + interior.
- **E (accept).** No paleoclimate; use present climate, as `resources.go` already does for coal.

## 2. Per-cell assignment sketch

Score all 18 per land cell, argmax, keep runner-up as `secondary`.

```
craton = interior·(1−relief)·Π(1−boundaryProximity_k)·plateAge   basin = depositionAccum (or proxy)
arc = f(distFromArc)   collide = f(distFromCollision)   rift = f(distFromRift)
margin = continental·nearCoast·(1−relief)·(no active boundary within R)
unroofed = relief·elevation                        // how deeply the column has been cut

arc_andesite = arc·(1−unroofed)          porphyry_intrusive   = arc·unroofed
granite_batholith = collide·unroofed     orogenic_schist      = collide·relief
ophiolite_ultramafic = collide·narrowBand  silicic_tuff       = continentalHotspot + arc·caldera
basalt_province   = hotspot ∨ (rift·lowRelief)
limestone_platform = margin·warm·(1−aridity)
marble_belt        = limestone_platform score, then overprinted by collide  ← ordering, not a new rule
coal_measures = basin·humid·forest       shale_basin = basin·(1−aridity)·lowEnergy
sandstone_redbed = basin·aridity·nearUplift
evaporite_basin  = basin·aridity·(endorheic ∨ rift-restricted)
alluvium(overlay) = soil.Alluvial/channel   loess(overlay) = downwind(arid|ice)·lowRelief
craton_basement  = craton (default for old continental interior)
banded_iron_formation, kimberlite_field = sparse POINT/PATCH processes seeded on craton cells
```

- **Rare classes are point processes, not fields** — a smooth score puts diamonds on every shield cell.
- **Output must be spatially autocorrelated** — add low-frequency `procnoise` (~150–300 km) to the
  scores + 2 neighbour-majority passes (as `RegularizeCoastlines` does), else it is salt-and-pepper.
- **Alluvium/loess are overlays** — overlay drives soil and placer, bedrock beneath drives caves/radiance.

## 3. Endowments that improve most (`climgen/resources.go`)

| rank | endowment | today | with lithology |
|---|---|---|---|
| 1 | iron | `0.70·orogenicBase` = "tall mountains" | BIF on shields + coal-measures ironstone; makes coal+iron co-location (basis of heavy-industry geography) possible at all. Biggest ordinal error in the file. |
| 2 | gem | `rockiness×relief` = gems wherever mountains are | kimberlite point fields, pegmatite, marble corundum — restores *rarity structure*, the whole point of gems |
| 3 | leadSilver | smear of `hardRockMetalContext` | MVT lead-zinc on flat carbonate platforms: a lowland silver province currently ungeneratable |
| 4 | copper | keyed to volcanism | sharp porphyry peak (arc+unroofing) *and* redbed copper with no volcano in sight |
| 5 | stone | one `rockiness` blend | quality tiers (granite/marble/slate/limestone vs shale/alluvium) + lime for mortar |
| 6 | coal, evaporite, oilGas | climate gate already right | adds the missing *substrate* gate — no more coal on shield/volcanic cells |
| 7 | placer | least changed | should inherit source affinity from **upstream** cells via the hydrology `Receivers` graph |

## 4. `subterranean_potential` (§G1d)

```
karst_eff = karst_propensity · smoothstep(precip)  // arid limestone ≈ no caves
          · smoothstep(relief)                      // vadose thickness above the water table
          · (1 − waterlogged)                       // lake/marsh cells drown their own caves
potential = clamp01(0.45·karst_eff + 0.20·pseudokarst·volcanism + 0.20·excavatability
                    + 0.15·smoothstep(relief)) · (1 − 0.5·alluvium_overlay)
```
Two routes: **found** caves (limestone/gypsum/marble karst, lava tubes) and **dug** holds (tuff, loess —
high `excavatability`). Marble belts top out: karst + orogenic relief = deep vertical systems. Gypsum
scores high but should carry an instability tag.

## 5. Soil — plus a fourth consumer

`climgen/soil.go` derives fertility from climate and hydrology only; parent material is absent. Add
`+0.20·parent.fertility_delta`: basalt/loess fertile, granite and shield acidic and poor, serpentine
barren, limestone thin but alkaline. **Bonus:** the crop drafts carry a `soil_ph` requirement but
**climgen has no pH diagnostic anywhere** (`SoilDiagnostics` has no such field). `ph_tendency` is the
only plausible source — a fourth consumer beyond the three named in §G1d/§G1e.

## 6. Radiance hook (§G1e)

Hosts are the three `radiance_host` classes, under one rule: **richness × decay_rate ≈ constant** — a
vein is rich and short-lived or weak and long-lived, never both. Bonanza/decades and rich/centuries
sites prefer `sandstone_redbed` on `craton_basement` (unconformity type — a two-class *adjacency* rule,
so they are rare, sharp, findable, worth fighting over); millennia/disseminated sites prefer
`granite_batholith` and `silicic_tuff` (common, diffuse — the "elder hold" substrate). The renewing band
is vent-fed off the existing hotspot field: exactly the autarkic-vent exception §G1d already reserves.
