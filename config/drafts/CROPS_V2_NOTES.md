# Crops v2 draft — notes

`crops_earthlike_v2.json` (82 real, `crops/v2-draft`) + `crops_fantasy_v2.json` (7 fantasy). Re-derived
from `ecocrop_source/cropbasics_scrape.csv.gz` (2568 × 63, FAO ~2023) joined to `crop_view_data.csv.gz`
for `Notes`. v1's 21 were **re-extracted, not copied**. Not wired to a loader.

## Zeros-as-null: the old default is gone

**The 2023 columns do NOT share the old table's 47.9 %-zeros problem — they use a string sentinel.**

| column | `"no input"` | literal `0` | real numbers |
|---|---|---|---|
| `Killing.temp..during.rest`  | 1410 (54.9 %) | **0** | 1158 |
| `Killing.temp..early.growth` | 1410 (54.9 %) | **390** | 768 |

The rule inverts what v1 feared: `"no input"` → null, and **a literal 0 is a measurement, kept**. The
old table's 819 zeros were `"no input"` collapsed to 0 by the Recocrop packagers — the whole bug.
`during.rest` has *not one* zero in 2568 rows, which proves it: nobody ever wrote 0 there, so every 0
in the other column is real. Result: `frost_kills_at_c` null for **32/82**, `winter_hardy_to_c` for
**41/82** — honest, not lossy; null means "not assessed", never "frost-proof".

**Split rule.** `frost_kills_at_c` ← early-growth. `winter_hardy_to_c` ← during-rest *only where the
crop overwinters* (perennial, or `autumn` in `sow_seasons`); else null plus a `killing_temp_note`
holding the discarded value — for a spring-sown annual "during rest" describes a seed in a sack, not a
field (pearl millet, cowpea, potato, sweet potato, sunflower, flax, tobacco, quinoa, amaranth).
Positive during-rest values on tropical perennials are **kept**: cacao/clove `+5` is a real
chilling-death threshold, and gives the right ordinal against pear's −34.

## Coverage: crops per (slot group × package tag)

|                 | temp | medi | mons | trop | arid | step | high | bore | n |
|---|--:|--:|--:|--:|--:|--:|--:|--:|--:|
| cereals         | 9 | 4 | 6 | 3 | 4 | 4 | 6 | 4 | 14 |
| pulses          | 6 | 5 | 4 | 3 | 3 | 2 | 3 | 1 | 9 |
| fodder          | 4 | 2 | 0 | 0 | 1 | 1 | 1 | 2 | 4 |
| roots/tubers    | 2 | 0 | 4 | 4 | 0 | 0 | 2 | 1 | 6 |
| tropical staple | 0 | 0 | 2 | 4 | 0 | 0 | 0 | 0 | 4 |
| oils            | 2 | 2 | 2 | 2 | 3 | 2 | 0 | 1 | 6 |
| fiber           | 2 | 1 | 4 | 3 | 1 | 0 | 0 | 2 | 5 |
| fruit/nut       | 6 | 6 | 2 | 1 | 6 | 3 | 2 | 2 | 13 |
| stimulants      | 2 | 2 | 4 | 4 | 0 | 0 | 3 | 0 | 5 |
| spices          | 1 | 3 | 5 | 7 | 2 | 3 | 2 | 0 | 10 |
| dyes            | 2 | 1 | 1 | 1 | 0 | 0 | 0 | 1 | 3 |
| special         | 1 | 1 | 2 | 1 | 0 | 1 | 2 | 1 | 3 |
| **any tag**     | 37 | 27 | 36 | 33 | 20 | 16 | 21 | 15 | 82 |

Every crop carries a unique `slot` (asserted at build). Gaps are deliberate: no fodder or tropical
staple is `mediterranean`/`arid` because the Norfolk rotation and the banana belt really are absent
there. **79 ecocrop-sourced / 3 estimated** (flagged in `envelope_ref`): einkorn and woad are absent
from the 2023 scrape entirely, and madder is present as code 9344 *with no envelope values at all* —
presence in the table is not evidence of coverage.

## Proposed new trade goods (12 new; 10 carried from v1)

| good | class | why |
|---|---|---|
| `pulses` | raw | Storable protein, demand curve unlike grain, carrier of the N-fixation/rotation mechanic. Nine crops have nothing to sell today. |
| `sugar` | processed | Crystallised cane: worthless standing, imperial after a mill. No sweetener exists today. |
| `spices` | luxury | Pepper, cinnamon, clove, nutmeg, cardamom, ginger, turmeric, saffron, cumin, coriander. **The** long-distance luxury, today collapsed into generic `herbs`. |
| `silk` | luxury | Mulberry leaf → sericulture. Highest value-to-bulk tradeable; what makes a trans-continental route pay for itself. |
| `tea` | luxury | Obligate acid soil, monsoon highland — different soil, climate and demand geography from coffee. |
| `coffee` | luxury | Tropical highland only ⇒ a forced import for every temperate polity. |
| `cocoa` | luxury | Equatorial lowland, gated on controlled fermentation; the only luxury that is also a fat. |
| `tobacco` | luxury | Grows anywhere warm, worthless uncured — cleanest test of the processing-gate mechanic on a luxury. |
| `nuts` | raw | Dry, storable, high-fat, protein-bearing; unlike `dried_fruit` it can provision a caravan or a siege. |
| `opium` | luxury | 0.02 t/ha at value_mult 60 — extreme value-to-weight, plus the only analgesic input. |
| `ale` | processed | v1 gave the vine wine and the orchard cider and left the grain belt nothing. Ale is how grain surplus becomes urban calories and tax. |
| `fuel_fiber` | raw | Bagasse, stover, husk, prunings — in the schema example, never proposed. Makes sugar milling viable in a deforested tropics. |

Carried from v1: `wine`, `olive_oil`, `plant_oil`, `fresh_fruit`, `dried_fruit`, `cider`, `root_crop`,
`fodder`, `cordage`, `dyestuff`. Unresolved: cave fungus emits a `fungus` good in neither list.

## EcoCrop vs the historical package, and fantasy

- **Olive** — the envelope alone (Cs, RMAX 1200) spreads olives over half of temperate Europe. Per G1a
  the limiter is catastrophic-frost *return period* (>10 yr), which no column encodes; the split makes
  the gap visible (`winter_hardy −10` / `frost_kills 0`) but the check belongs at the call site.
- **Maize** `cycle_days` 65–365 is EcoCrop's known-absurd duration range (G1a) — advisory only.
- **Durum vs bread wheat** — FAO gives durum the *wetter* absolute floor (400 vs 300 mm) yet a drier
  optimum ceiling (700 vs 900); the optimum band, not the floor, reproduces the real distribution.
- **Cotton** `G. herbaceum` TMIN 18 clips the historical Indus/Nile range; kept because the alternative
  `G. hirsutum` is the American species. **Sorghum**: 2023 splits sweet vs grain, not by altitude, so
  v1's "low altitude" record is gone (grain sorghum, ROPMX 600, used). **Photoperiod** for rye, grape
  and date palm stays `corrected` as in v1 — FAO says short-day, all three fruit far north of that.
- **Fantasy**: each keeps `analog`, `sphere_tags`, `affinities` and one twist axis, envelope
  re-inherited from the analog's 2023 record and killing temps shifted by the same delta — frostwheat
  now reads −12 / −32 instead of v1's conflated −28. `cave_fungus`'s analog `highland-tuber` no longer
  exists (v2 splits it into `potato` / `bitter_potato`) and is repointed at `potato`; `ember_pepper`
  keeps *Capsicum frutescens* (EcoCrop 621), excluded from the real catalog as a black-pepper duplicate.

**Ordinal sanity, checked at build:** rice RMIN 1000 > pearl millet 200 · barley (TMIN 2, RMIN 200)
colder+drier than bread wheat (5, 300) · date palm TMAX 52 = catalog max · coffee 14–28 °C, cacao/clove
narrow tropical bands · **rye `winter_hardy −18` vs `frost_kills −1`, the v1 bug is gone** · pear −34
coldest overall, alfalfa −25 coldest herbaceous.
