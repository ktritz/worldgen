# Trade-goods expansion — notes

`trade_goods_expansion.json`: **25 new goods** (raw 8 / processed 9 / luxury 8; no new finished or
strategic) covering every good the two crops_v2 files emit and the economy lacks. Merged catalog:
29 → 54 (raw 24, processed 15, finished 2, strategic 2, luxury 11). Not wired to a loader; entry
shape is exactly `climgen.TradeGoodSpec`, ordered topologically for the merge.

## Chains (`input → output`; `*` = existing good)

    crop*                → fodder, fuel_fiber, root_crop, pulses          (endowment-only)
    crop* + timber*      → nuts, fresh_fruit, silk_cocoon                 (endowment-only)
    crop* + herbs*/resin*→ plant_oil, tobacco, tea, coffee, cocoa, spices, opium
    timber*+clay*+crop*  → fungus                                          (proxy, see Q5)
    livestock*           → tallow
    fuel_fiber + timber* → sugar                    [mill needs fuel: bagasse, then wood]
    silk_cocoon          → silk                     crop → silk_cocoon → silk
    fresh_fruit          → dried_fruit, cider
    fresh_fruit + ceramics* → wine, olive_oil       crop → fresh_fruit → wine / olive_oil
    grain*               → ale                      crop → grain → ale
    fiber*               → cordage → ships*         crop/wetland → fiber → cordage → ships
    dye_plants*          → dyestuff → fine_clothing* dye_plants → dyestuff → fine_clothing
    olive_oil/plant_oil/tallow → soap*              (see patch note below)

Five 3-step chains land (fruit→wine, fruit→olive_oil, fiber→cordage→ships,
dye_plants→dyestuff→fine_clothing, crop→silk_cocoon→silk), matching existing
`iron_ore → iron_goods → weapons_armor`. Nothing goes deeper than 3.

## Existing goods that gain / change inputs (`existingGoodPatches`, advisory — file not modified)

**Correction: `soap` and `ships` already have inputs** (soap = salt 0.14 + livestock 0.10 +
herbs 0.03; ships = timber 0.55 + fiber 0.22 + resin 0.18), so these are extensions, not fixes:

- `ships` — **replace** `fiber 0.22` with `cordage 0.20` (not add: `inputAccess` is a MIN over
  inputs, so keeping both double-gates every shipyard).
- `soap` — add `tallow 0.07` + `plant_oil 0.07` at low need (soft gate), no capability floors.
- `fine_clothing` — replace `dye_plants 0.20` with `dyestuff 0.16`; `cloth` keeps raw `dye_plants`,
  which is what makes the two dye tiers non-substitutable.
- `perfumes` — add `spices 0.05`, the luxury tier's only internal link. Plus `tuningPatches` for
  `goodDemandScale`, `localSupplyReliefByGood`, `marketInputReservationByInput` (fresh_fruit is
  contested by 4 consumers, fiber by 3).

## Crop products with no good

**None left.** All 34 goods named across both crop files now resolve; `fungus` (cave_fungus, absent
from the 22-good proposal) and `silk_cocoon` are the additions beyond that list. Caveat:
**`silk_cocoon` requires a crops_v2 edit** — mulberry emits `silk` directly today
(`leaf → silk, sericulture`); it should emit `silk_cocoon`, with reeling as the trade-good step.
Unmodelled: crop `process` verbs (`mill`, `leach`, `freeze_dry`) have no trade-good representation —
crops_v2 folds milled flour into `preserved_food`, so no `flour`/`bread` goods are proposed.

## Open questions / what the code constrains

1. **`inputs` and `sourceWeights` are mutually exclusive in effect.** `polityGoodSupply`
   (trade_goods_polity.go:156) takes the manufacturing branch whenever `len(Inputs) > 0` and never
   reads `localPotential` — and `productionDrivers` is computed but unused on that branch too. A
   good is either endowment-placed or input-placed, never both.
2. **Therefore the luxury tier has no real geography yet.** The source-field vocabulary is hard-coded
   in `tradeGoodSourceFields` (21 names: crop, pasture, fish, shellfish, salt, timber, game, pelts,
   ores/gems, coal, evaporite, clay, stone, herbs, dye_plants, fiber, resin) — there is no
   per-crop field, so spices/coffee/cocoa/tea/opium/tobacco all draw the same `crop`+`herbs`+`resin`
   blend and differ only by `productionDrivers` and `rawCatchmentSensitivity`. **The promised
   chokepoint geography needs crop-catalog-derived endowment fields** (e.g. one field per crop slot
   group, or a suitability field per crop) — the single largest follow-up.
3. **No substitutable inputs.** `inputAccess = min over inputs`, so "olive_oil OR plant_oil OR
   tallow" is inexpressible; the only lever is need magnitude (small need ⇒ soft gate, since
   `0.28*tradeAccess` alone nearly clears it). All optional-ish inputs here use need ≤ 0.08.
4. **Catalog order is load-bearing and unvalidated.** `computePolityGoods` iterates `settings.Goods`
   in array order against `balance.Supply` accumulated so far, so inputs must precede consumers.
   `ValidateTradeGoodsSettings` checks neither that an input names a known good, nor cycles, nor
   order — three cheap validator additions worth making before a 54-good catalog lands.
5. `fungus` has no honest source field (proxied by timber+clay+crop); subterranean production needs
   a new field, per FUTURE_WORK_SCOPING G1d.
6. The `luxury` demand driver is `0.45*capitalTier + 0.55*tradeAccess`, i.e. self-reinforcing — 8
   new luxuries keyed to it may homogenize rich-polity demand. Cost: the goods loop is
   O(goods × polities × nodes × markets), so 54 goods is ~1.9× today's work.
