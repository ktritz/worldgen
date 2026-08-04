# domesticates_v2.json — notes

Successor to `domesticates_earthlike.json` (v0: 12 entries, all normalized 0–1). **31 entries** (27 real + 4 invented subterranean), `schemaVersion: "domesticates/v2-draft"`, built to `crops/v2-draft` conventions: absolute units, unique `slot`, `package_tags`, `envelope_source`/`envelope_ref`, one-to-many `products[]` → trade goods. Not wired into any loader.

**`envelope_source` is `"estimated"` on every real entry** (`"invented (G1e slot N)"` on the subterranean four): unlike crops v2 there is no dataset behind these numbers — authored from general husbandry knowledge, no citations claimed or implied. Units: °C; mm/yr precipitation floor + optimum band; metres altitude (`min` is load-bearing for yak and the camelids); L/day + days-without-water + wallow flag; ha/head stocking; kg / kgf / km·h⁻¹ / km·day⁻¹ of work; yields in kg per **head, hive, colony, or hectare of host crop** per year (`unit` on every product). Dimensionless only where it should be: `caravanCapability`, `warMountCapability`, `fodderNeed`, `herdability`, `feed_competition`, `disposition`, service magnitudes. Those four keep v0 camelCase (land_transport is written against them); new fields are snake_case like crops v2.

## Slot × zone coverage

| zone | n | slots | gaps |
|---|---|---|---|
| temperate | 15 | taurine bovine, ovine, omnivore, lagomorph, riding + draft horse, donkey, mule, 3 fowl, hive, sericulture, canine, mouser | — |
| mediterranean | 12 | taurine bovine, ovine, caprine, lagomorph, riding horse, donkey, mule, fowl, hive, sericulture, canine, mouser | — |
| monsoon | 10 | zebu, paddy bovine, omnivore, elephant, fowl, paddy waterfowl, hive, sericulture, canine, mouser | fibre is silk, not wool |
| tropical | 11 | zebu, paddy bovine, caprine, omnivore, dromedary, elephant, 3 fowl, canine, mouser | — |
| arid | 11 | zebu, caprine, donkey, mule, both camels, llama, fowl, guinea fowl, canine, mouser | — |
| steppe | 10 | taurine bovine, ovine, caprine, riding horse, pony, both camels, guinea fowl, hive, canine | — |
| highland | 11 | yak, ovine, caprine, pony, donkey, mule, bactrian, llama, alpaca, reindeer, canine | no eggs |
| boreal | 7 | taurine bovine, yak, draft horse, pony, **reindeer**, goose, canine | — |
| subterranean | 4 | detritus grazer, tunnel pack beast, shaftspinner, guano colony | no eggs; war beast deferred per §G1e |

v0 had **no boreal domesticate at all** and no tropical bovine. Every zone now has a milk source, a fat source, and a pack or draft animal.

## `animalRequirement` resolution (land_transport_earthlike.json)

All four resolve, three through the new `legacyAliases` map: `donkey`→direct; `camel`→`camel_dromedary` (a cold-desert route wants `camel_bactrian`); `cattle`→`cattle_taurine` (**`cattle_zebu` is correct in monsoon/tropical**); `horse`→`horse_war` (**horse-wagon should be re-pointed at `horse_draft`**). Nothing dangles. "Ox" is deliberately not a slot — it is a castrated male of a bovine slot, so draught rides on `work.draft_pull_kgf` + the `draft_animals` product and any bovine can be an ox team. Not yet modelled by land_transport: `mule` (mountain pack), `llama` (cart-free highland), `reindeer` (sled), `elephant_asian` (heavy/siege), `tunnel_pack_beast` (`clearance_m` 1.3).

## Newly supplied goods

`proposedGoods`: **dairy, eggs, honey, beeswax, tallow, draft_animals, warhorses, ivory, guano** (+ `silk` and `fuel_fiber` carried from crops v2). Existing goods now animal-fed: livestock, leather, wool, fiber, preserved_food, cordage.

- **silk** — the chain was broken: crops v2 mulberry emits `silk` through a `sericulture` gate whose facility is a `silkworm_shed`, and no animal existed to occupy it. `silkworm` fixes it (pupae → `preserved_food`, floss → `fiber`); `shaftspinner` is the subterranean second source.
- **tallow** — 12 emitters; lamp oil, soap input, candle stock. `soap` previously had only a plant-oil input. `detritus_grazer` leads at 25 kg/head/yr because an underground hold cannot see without fat.
- **eggs** (4 fowl); **honey + beeswax** (nothing supplied wax before — candles, seals, lost-wax casting); **guano** (closes part of the G1d/G1e substrate loop); dung → existing `fuel_fiber` (yak 900, buffalo 1000 kg/head/yr). Antler/horn fold into `ivory` rather than adding a good.

## Anna Karenina check

Every entry carries `domestication{months_to_maturity, captive_breeding, disposition, panic_response, social_structure}` + `forage.feed_competition` (the diet criterion), so invented animals face the same bar; the subterranean four add an explicit `domestication_check` with a per-criterion verdict. It changed the design twice. **The cave-spider silk producer was rejected on diet** — a predator needs ~10× its yield in prey biomass and competes with people for protein, precisely §G1e's "must not eat the staple" — and was re-specified as a detritivorous moth larva on spent fungal beds. **`guano_colony` fails disposition and panic response outright** but is kept, because the honeybee precedent legitimises a second management pattern: colonies are installations, not herds, so the herd criteria do not bind (`captive_breeding` PARTIAL — retained by roost fidelity, not bred). Real entries fail too, which is the point: `elephant_asian` fails captive breeding and growth rate (180 months) → `breeding: "capture_dependent"` + `requires_wild_population`, a forest dependency rather than a herd; `mule` is `hybrid_sterile` and needs both parent stocks maintained; `dog` carries the highest `feed_competition` (0.60) because carnivory is the real price of its services.

## Ordinals (validated) and open questions

Dromedary most arid (25 mm/yr, 12 days without water, 50 °C) < bactrian 50 < donkey 100; reindeer coldest (−55 °C) then yak (−50); water buffalo most water-dependent (900 mm floor, 1 day, obligate wallow); `horse_war` fastest sustained (8 km/h) and top `warMountCapability` 0.95; `elephant_asian` highest pull (1500 kgf). JSON parses, ids/slots unique, every product `good` exists in `trade_goods_earthlike.json`, crops v2, or `proposedGoods`.

1. Highland has no egg source — `chicken` tolerates 3500 m by envelope but is not tagged highland.
2. `stocking_ha_per_head` (reindeer 40, camels 12–14, sheep 0.35) should drive pastoral density but nothing reads it; needs a pasture-productivity term the climate stack does not expose.
3. `fodderNeed` + crops v2's `fodder` good imply a real feed balance (a draft horse at 0.95 needs oats a subsistence village cannot spare); left as scalars pending a decision on simulating feed.
4. A tropical stingless-bee hive was cut for size; tropical honey stretches the temperate `honeybee` envelope to 42 °C.
5. No disease geography — trypanosomiasis is the biggest real constraint on African livestock and would be a hard zero, not a tolerance curve.
6. Wildlife/megafauna (§G2 remainder) untouched; `ivory` is proposed here but wants a huntable-population model.
