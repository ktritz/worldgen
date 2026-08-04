# Crops v1 migration notes

Companion to `crops_earthlike_v1.json` (21) and `crops_fantasy_v1.json` (7); neither is wired into a
loader. Schema: `crop_schema_v1_proposal.jsonc`. Rules: `docs/FUTURE_WORK_SCOPING.md` §G1/G1a/G1b.

## Envelope provenance — 20 EcoCrop, 1 estimated

Extracted programmatically (no hand transcription) from the FAO EcoCrop parameter table shipped as
`inst/parameters/ecocrop.rds` in the R **Recocrop** package (github.com/cropmodels/Recocrop, 1710
records). Mapping: `TMIN/TOPMN/TOPMX/TMAX→temp_c`, `RMIN/ROPMN/ROPMX/RMAX→precip_mm_yr`,
`GMIN/GMAX→cycle_days`, `PHMIN/PHMAX→soil_ph`, `KTMP→frost_kills_at_c`, `TEXTR/DRAR/DEPR/FER/FERR→
soil`, `LIGR→light_min`, `PP→photoperiod`. Every entry names its record in `envelope_ref`.

**Estimated: `madder` only.** EcoCrop has no *Rubia tinctorum* (nor *Isatis tinctoria*) record — dye
plants are systematically absent. Marked `"envelope_source": "estimated"`; no EcoCrop citation claimed.

Non-obvious record picks: rice = *Rice, paddy (Indica)*; millet = *Pearl millet*; sorghum = *low
altitude* variant; flax = *Linseed* (same species, no fibre-flax record); cotton = *Levant cotton*
(**G. herbaceum**, Old World, not American upland); highland-goosefoot = *Quinoa*.

All 7 fantasy crops are `estimated`: each inherits one named EcoCrop record shifted on exactly one
axis — frostwheat/ember-pepper/silverleaf temperature (wheat −12 °C, *C. frutescens* +11 °C, tea
−8 °C), cave-fungus/duskvine light (potato→0, grape→0.08), mireroot water (taro ×1.6 precip,
drainage narrowed to saturated), skywheat cycle length (barley 90–240 → 45–75 d). Anything v0 wanted
that would have been a *second* twist was dropped, recorded per-crop in `notes`.

## Proposed new trade goods (10 + 1 fantasy)

| good | rationale |
|---|---|
| `wine` | Export half of Mediterranean polyculture; highest-value thing a hillside makes. |
| `olive_oil` | The other half — and the input the existing `soap` good has no source for. |
| `plant_oil` | Generic pressed seed oil (linseed/hemp/cottonseed); gives fibre crops a live 2nd product. |
| `fresh_fruit` | Perishable orchard/vine output; near-untradeable by design → hinterland demand, not trade paths. |
| `dried_fruit` | Raisins/figs/dates: the caravan-legal form, and how the arid package exports calories. |
| `cider` | Northern-temperate analogue of wine; gives apple a value good where the grape envelope fails. |
| `root_crop` | Non-cereal perishable staple; kills the "every tuber is `grain`" collapse and makes the chuño gate mean something. |
| `fodder` | Straw/stalks/pomace/oilcake — the missing link from crops to the domesticate catalogue. |
| `cordage` | Hemp/palm rope; gives the existing `ships` strategic good an agricultural input. |
| `dyestuff` | Concentrated dye cake vs raw `dye_plants`; indigo loses 50× its mass in the vat. |
| `fungus` *(fantasy)* | Subterranean staple, distinct from `root_crop` by being decoupled from sunlight. |

## Processing gates

Only two of the 21 need one. **olive** → `required_for:"any"`, press/cure (raw drupes inedible:
`without_gate.edible="none"`). **highland_tuber** → `required_for:"staple"`, freeze-drying (chuño):
calories are fine without it, *storability* is not, so `without_gate` carries a `shelf_life_days: 90`
override — a small extension to the proposal schema. Maize, cassava and acorns are **not in the v0
catalogue**, so their gates have nothing to attach to. Quinoa saponin washing is a product `process`,
not a gate: labour, no tech.

## Envelope vs. historical package — disagreements

- **highland-tuber**: EcoCrop *Potato* is TOPMN 15 / TOPMX 25 / KTMP −1, a temperate **lowland**
  envelope, but v0 tags it `highland` only. The truly alpine record is *Potato, Bitter* (*Solanum ×
  juzepczukii*, TOPMN 6 / TOPMX 14 / KTMP −5) — which is also the actual chuño cultivar. Kept
  *Potato* for breadth; swapping is a one-field change.
- **rye vs oats**: EcoCrop gives oats KTMP −15 but **rye KTMP 0**, inverting rye's entire reputation
  as *the* cold/poor-soil cereal. Suspected data error; left as sourced but flagged in the entry.
- **olive**: KTMP 0 understates the real constraint, which per §G1a is catastrophic-frost *return
  period* (>~10 yr). Needs a call-site check.
- **fig** is 12 °C hardier than olive (KTMP −12) — a Mediterranean package gated on the olive line
  will place fig far outside it. **grape** at TMIN 10 / RMIN 400 admits most of temperate Europe,
  wider than its `mediterranean` tag. **date-palm** is a `staple` at RMAX 400 mm, so its staple
  status rests entirely on `water.irrigation_response` 0.95. **tea** is the only obligate-acid-soil
  crop (PHMAX 6.0) yet tagged `highland` at TOPMN 20.

## Open questions

1. EcoCrop has **no high-temperature-damage parameter** (§G1a): TMAX is a hard cut, no curve.
2. EcoCrop `PP` is unreliable for perennials — rye, grape and date-palm all carried values
   inconsistent with their real latitude ranges and were overridden (`photoperiod_source:
   "corrected"`). Nothing else was hand-edited.
3. `TEXTR='W'` (13 of 20) collapses to "all textures", losing v0's `thin-rocky`/`volcanic`/
   `alluvial`. `soil.depth_tolerance` (from `DEPR`) recovers the thin-rocky half only.
4. Yields, `labor_days_ha`, `storage`, products and gates are **authored**, not EcoCrop — only the
   climate envelope is sourced. Wheat/barley labour is anchored on ACOUP IVb (per-iugerum → per-ha).
5. Tea, peppervine and ember-pepper all collapse into `herbs`. A `spice` good with its own demand
   curve is probably warranted; left out to keep the proposal list tight.
6. `precip_mm_yr` stores EcoCrop's annual values verbatim; Recocrop's own model rescales them to
   per-cycle-month. If suitability is scored per growing *season*, that rescale must happen at the
   call site or short-cycle crops will read as drought-stressed.
