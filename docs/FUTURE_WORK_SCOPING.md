# Future Work Scoping — Toward a Full Fantasy World Simulator

Research date: 2026-08-02. Sources: deep-dive on Dwarf Fortress design (wiki, Tarn Adams
interviews/GDC talks) and a survey of Victoria 2/3, EU4, CK3, Songs of the Eons (source read),
Caves of Qud (papers/GDC), Ultima Ratio Regum, Azgaar's FMG (source read), plus Dominions,
Wildermyth, King of Dragon Pass, Colonization, Mount & Blade, Talk of the Town. Source URLs in
the appendix of each section's research notes (kept in this doc where load-bearing).

## Where we stand

What this project already has is **unusually strong for the genre** — a better physical substrate
than any surveyed game, including DF:

- Plates → elevation → seasonal climate → biomes → hydrology → soils/agriculture/resources/wildlife
- Population → settlement networks → multimodal trade routing (land/river/coastal/ocean) with
  real route costs
- Polities with profile selection, attitudes, spheres; **12 fantasy ancestries + 9 cultures**
  already defined with traits/tags/affinities (`config/profiles/fantasy/`)
- ~30 trade goods with categories, endowments, processing inputs, market scope/priority/reservation

What every surveyed game gets its narrative power from, and we lack entirely:

1. **Time** — everything is a static equilibrium snapshot; no history, no change, no ruins
2. **Names/language** — no proper nouns anywhere in output
3. **Religion/myth** — nothing
4. **Characters** — nothing individual
5. **Dynamic (price-forming) economy** — markets are supply/demand-shaped but static
6. **A legends surface** — no way to read the world as stories

## The central lesson from the research

DF's magic is not simulation depth (its physical model is far below ours; its economy has been
*disabled since 2008*). Its value comes from four cheap mechanisms:

1. A crude zero-player strategy loop whose **typed event log is the product**
   ("history is just a record of that" — Tarn Adams)
2. **Notability promotion**: populations are counters; a tiny fraction get names and persistent
   records only when they do something notable
3. **Event collections** (war ⊃ battle ⊃ duel) that keep thousands of years narratable
4. **Provenance-bearing objects** (artifacts, unique books, relics) threading through the log

Caves of Qud goes further: **no simulation at all** — ~19 event templates × ~10 thematic domains,
rationalizing causes after picking effects, each fact surfaced 2+ times in different media
(shrines, books, quests). Overlap, not simulation, is what reads as coherence.

Meta-caution from the failures: SotE and URR both died of **bottom-up depth starving the top
layers** (SotE: three climate rewrites, culture ended as a name generator; URR: 34 generators in
a frozen pipeline, 10 years, burnout). We are past the danger zone physically — the priority now
is spending *up the stack*, not deepening the substrate.

---

## Theme A — Naming & language (do first: cheapest, transforms everything)

**Crib**: Azgaar's FMG name generator wholesale (MIT, source read; local clone in scratchpad):
~43 real-toponym syllable bases, Markov over pseudo-syllables keyed by preceding letter, per-culture
min/max/doubling params, plus its state-name morphology table (-land, -maa, -stan, Al- prefix,
"Kingdom of X" vs "Xian Empire" by area quantile × culture). URR's per-culture greeting/dialect
fragments as a later garnish.

**Fit**: new `namegen` package; assign a name base per culture/ancestry (the 12+9 catalog gives the
hook); naming pass over polities, settlements, rivers, mountain ranges, seas, trade goods variants.
Deterministic from seed. Output feeds review summaries first (instant legibility win), map labels later.

**Effort**: small. No coupling to sim internals.

## Theme B — Culture & religion geography

**Crib**:
- Azgaar culture spread: culture *type* classified from seed-cell physical situation (Naval/River/
  Highland/Nomadic/Hunting), type-specific expansion costs flooded over the map. We drive this from
  real biomes/habitability/routes instead of his heuristic tables — our version is strictly better.
- Azgaar religions: one non-expanding **folk religion per culture**; organized religions seeded at
  top-population settlements and **spread along the trade network** (his costs: ÷3 on trails, ~1 on
  roads, 50 over water with a sea route). We have real routes — the part he fakes.
- CK3 structure for later mutation: culture = pillar vector (heritage/language/ethos/martial/
  aesthetics) + traditions, with **drift/merge/split** operators and pairwise acceptance; faith =
  3 tenets + doctrines + holy sites + **fervor** (low fervor → heresy defection: endogenous churn).
- URR's ideology-axes → correlated cascade; Qud's domain tokens: give each culture/religion a small
  set of thematic tokens that every downstream generator re-reads (architecture, art, oaths, bans).
- DF entity schema: ethics/values vectors per polity, generated from our polity profiles;
  value-distance as diplomacy/war pressure (we already have attitudes — this deepens them).
- Dominions scales diffusion (one-line probabilistic relaxation toward local "dominion") as the
  spatial spread/decay rule for cultural traits.

**Fit**: extends `polity_profiles`/`profile_catalog`; new culture-region + religion layers between
population and polities. Trade-route religion spread reuses the existing route graphs directly.

**Effort**: medium. Static-world version needs no time axis; mutation operators activate in Theme C.

## Theme C — History engine (the spine; largest single investment)

**Crib** (architecture is settled across DF+CK3+Qud, remarkably consistent):
- **Zero-player strategy loop**: coarse ticks (year or era) over the *existing* polity/trade/
  population machinery. Polities expand/contract/split/merge, found/abandon settlements, fight,
  trade. Bad AI is fine. The log is the product.
- **Typed event log**: (actor, verb, target, place, year) tuples + **event collections**
  (war ⊃ battle ⊃ duel; migration; plague; founding). Structured JSON export from day one —
  DF's XML dump created its entire tooling ecosystem.
- **CK3 on_action pulses** instead of per-entity polling: fixed-interval pulses → trigger gate over
  rich sim state → weighted random pool (with explicit no-event weights) → one event. Coherence
  from gates, variety from pools, cost O(entities × pulses).
- **Notability promotion**: named figures only when notable (founders, war leaders, prophets,
  master artisans); statistical battle resolution with **sticky consequences** for named figures.
- **CK3 partition succession** as the polity fragmentation engine (realms bloat under strong
  rulers, shatter on death) — cheap source of border churn DF gets from wars alone.
- **Era names derived from state** (DF ages emerge from live counts, not scripts).
- Wars *about something*: DF wars are ethics-distance + grudges only; our trade/endowment layers
  can motivate wars over routes, ports, and scarce endowments — a genuine advance on DF.
- **Chronicler/sifting pass** (Talk of the Town lesson): generating events is cheap; selecting and
  surfacing the tellable ones is the hard part. Budget for it explicitly.

**Fit**: new `histgen` package orchestrating existing climgen civilization/trade stages per tick.
**The current resolution-independence work is a direct enabler** — re-running polity/trade
evaluation per tick requires those algorithms to be stable and cheap. Ruins/abandoned sites feed
back into settlement scoring; route decay/growth feeds trade.

**Effort**: large but incremental — a 5-tick era loop with 10 event types already produces legends.
Danger (DF's "Big Wait" lesson): ship it incrementally, never as a monolith.

## Theme D — Dynamic economy (after C; prices need time to mean anything)

**Crib**:
- **Vic3 order-ratio price**: `price = base × (1 + 0.75 × clamp((buy−sell)/min(buy,sell), −1, +1))`
  — stateless, clamped, best value per line of code of the three proven tiers. Per market node.
- **MAPI-style local price blend** keyed to route quality (Vic3 1.5): regional price geography
  emerges from our existing network quality — we already compute everything this needs.
- **Trade as arbitrage** with per-route capacity and inertia (Vic3 1.9 scrapped hand-managed
  routes for autonomous arbitrage — "player sets policy, agents arbitrage").
- Quantity-side inertia (Project Alice): production-scale drift, employment-shift caps, EMA
  satisfaction. All smoothing lives on the quantity side; the price formula stays memoryless.
- DF intrinsic value formula (form × material × quality) as the price *floor/prior*; provenance
  tracking on high-value goods (ties economy to Theme E).
- Anno tier-needs ladder: demand escalation as progression; substitution baskets for needs.
- Colonization's stepped price ladder as the fallback-simplest first implementation.

**Skip/warnings** (all documented failures): closed money loops (Vic2 inflation/liquidity death),
wage/rent microsim without demand feedback (DF 40d economy — disabled ever since), per-agent
auctions at world scale, goods-diffusion-as-trade (SotE's never-replaced placeholder), first-class
stockpiles except storable staples.

**Fit**: extends `trade_goods_markets`; the static supply/demand machinery becomes the initial
condition; per-tick price update runs inside Theme C's loop.

**Effort**: medium once C exists. Do not attempt before a time axis exists.

## Theme E — Myth, artifacts, legends surface (cheap, huge yield; partly parallel with C)

**Crib**:
- **Qud's historical rationalization**: decide the event, invent the cause after (write the needed
  property onto the entity). ~19 templates × thematic domains; grammar over one "spice" JSON.
  The cheapest believable myth layer known.
- **Myth as postdiction**: DF's myth generator makes myth *cause* geography because DF's geography
  is weak; we invert — generate myths that *explain* our simulated terrain ("the range is the
  serpent's spine"). Costs nothing physically, pays rent everywhere. Optionally let 1–2 myth facts
  have real map consequences so the layer feels load-bearing.
- **DF sphere ontology**: ~130 spheres with parent/friend/preclude relations as the semantic glue
  naming gods, seeding religions, theming art and curses. Small, static, high-leverage.
- **Provenance objects**: artifacts, named hero gear, unique books, holy relics; chain-of-custody
  events through the history log. Books as **transmissible knowledge objects** (a necromancy
  secret in a book → tower → war) — knowledge with provenance, same pattern as artifacts.
- **Every fact surfaces ≥2 times in different media** (Qud): legends log → engravings, shrines,
  book titles, settlement epithets, route names.
- Talk of the Town: let *beliefs* diverge from ground truth (rumor/gossip decay) — myth vs history
  divergence for free.

**Fit**: consumes Theme C's event log (gospel generation can start against the static world:
"founding myths" per culture/polity from terrain + endowments). Sphere ontology slots into the
existing profile tag system.

**Effort**: small-medium for static-world myths + naming epithets; grows with C.

## Theme F — Characters (last; only meaningful once C exists)

**Crib**: CK3 traits-as-AI-weights (one data field drives decisions, opinions, and flavor) +
stress as out-of-character cost; Wildermyth aspect/role casting (events cast roles by filtering
entities on aspects; uncast optional roles simply don't appear); SotE's character-mediated macro
events (named leader fronts a migration); KoDP-style casting queries against the sim.
**Skip**: full scheme/secret pipelines, perk trees, opinion-modifier zoos.

## Theme G — Content taxonomies (crops, creatures, vehicles, materials)

The systems themes above are the verbs; this is the nouns. The repo already contains the correct
architectural pattern: `config/maritime_vessels_earthlike.json` defines 8 historical vessel
classes (dugout-canoe → caravel) as ~24-parameter capability envelopes that the routing sim
consumes (payload, range, coastal/open-ocean capability, wind/current response, storm tolerance…).
**Every taxonomy below should follow that pattern**: a JSON catalog entry = a physical/capability
envelope validated against existing per-cell fields. The aggregate simulation fields stay as the
substrate; species/instances are *annotations sampled from those fields* with a small number of
mechanical hooks — never a parallel resimulation. (DF's plant/creature raws prove the
schema-not-content point: the raws format is the crib, the hand-written content is not.)

### G1 — Crops
Current state: `agriculture_productivity` has farming *regimes* (dry/mixed/intensive/pastoral/
floodplain) and multipliers, but zero named crops.
- Crop catalog with climate envelopes: temperature range/growing-season need, water demand
  (rain-fed vs irrigation vs floodplain), soil preference, storability, labor intensity, value
  class. We already compute seasonal temperature, rainfall, soil, floodplain, and irrigation —
  suitability is a pure lookup, no new simulation.
- **Crop packages** as the regional identity unit (Mediterranean wheat/olive/vine; monsoon rice;
  steppe herds + hardy grains): assign per culture-region from the envelope match, let cuisine
  and trade-good variants derive from the package (differentiated demand plugs into existing
  goods demand shares).
- History hooks (Theme C): crop diffusion along trade routes as events (Columbian-exchange
  dynamics — a route carrying a new staple is a *legible* historical event); famine years from
  seasonal-variance rolls; storable staples are the one justified stockpile (sieges, granaries).
- Fantasy crops are just catalog entries with unusual envelopes + sphere tags (frost-wheat in
  boreal margins, cave fungus decoupled from sunlight → underground settlement viability).

#### G1a — Crop data sources (researched 2026-08-02)

**Envelopes — start here:** FAO **EcoCrop** (~2,500 crops; absolute/optimal temperature and
rainfall min/max, pH, soil depth/texture, photoperiod, cycle length). Practical access is the R
**`Recocrop`** package (also `dismo::ecocrop`) rather than FAO's site. Caveats to record before
trusting it: parameters are expert opinion not measurement; the model is very sensitive to
`duration` (maize is listed 65–365 days, absurdly wide); and there is **no high-temperature-damage
parameter** — we must add one. FAO/IIASA **GAEZ v4** contributes the *length-of-growing-period*
concept, the best single "does this crop fit this cell" primitive.

**Packages assemble slowly and incompletely — the most important design finding.** In the
Southwest-Asian founder-crop dataset (Arranz-Otaegui & Roe 2023; machine-readable compendium on
[Zenodo](https://doi.org/10.5281/zenodo.5911218) / [GitHub](https://github.com/joeroe/SWAsiaNeolithicFounderCrops),
135 sites / 240 assemblages), **only 3 of 240 assemblages (1.25%) contain all eight founder
crops**; Cyprus shows five separate introduction waves over ~2,500 years. Use ubiquity as sampling
weights — barley 0.68, lentil 0.68, emmer 0.44, einkorn 0.34, bitter vetch 0.22, chickpea ~0.15,
flax ~0.13 — and **never emit a complete package at t=0**. Likewise the Mesoamerican "three
sisters" is not an ancient trio: squash ~10,000 BP, maize ~9,000, beans only ~2,500 BP.

**Processing-technology gates — the best single mechanic.** A package should not be adoptable
without its matching processing tech:
- **Nixtamalization** (alkali cooking) releases bound niacin; cultures where maize is *the* staple
  nixtamalize. Adopting maize as a staple without it should carry a health penalty — historically
  pellagra in Italy from the 1730s.
- **Chuño** (freeze-dried tuber, needs ~3,800 m freeze/thaw nights) converts a 3-month perishable
  into a decade-stable ration at ⅕ the weight. Perishable tubers cannot be taxed, stored, or
  marched — chuño is *how a tuber economy becomes an empire*.
- **Rotary quern → flour → baking** is why wheat took ~1,500 years to be adopted in China (whole
  wheat tastes wrong boiled/steamed); adoption was fast where grinding-baking culture existed.

**Drop-in numbers worth having:**
- Mediterranean (Garnsey): wheat needs ~300 mm in critical months, barley 200–250 → **wheat fails
  1 year in 4, barley 1 in 20**; the observed 4:1 barley:wheat area ratio falls out of that risk
  model rather than culture. Per-tile annual roll.
- Olive: 5–8 yr token crop → 25–30 yr full bearing, 300–500 yr lifespan; limited by
  **catastrophic-frost return period (>~10 yr)**, not mean temperature. A 5–30 year payback only
  makes sense with defensible inheritance → clean causal chain from land tenure → perennials →
  exportable oil/wine → imported grain.
- Labor/yield (ACOUP 2025 "Life, Work, Death and the Peasant" IVb — the most directly implementable
  table found): per iugerum, wheat 14.25–19.5 labour-days for 100–235 kg net wheat-equivalent;
  barley 9–12 days for 75–175 kg. **Barley = more volume, ~35% fewer calories, ~⅔ the labour** —
  a real tradeoff, not dominance. Yield ratios 4:1 bad / 6:1 average / 8:1 good; subsistence
  338 kg w.e. per adult male; respectability ≈ 2× subsistence.
- Transport (same series): **river 5× cheaper than land, sea 20× cheaper** — sanity check against
  our existing mode costs.
- Milpa land-equivalent ratio **1.6–1.9** in the tropics but only 1.06–1.3 in temperate trials —
  intercropping is climate-dependent, not universal.
- China: **Qinling–Huai line ≈ 800 mm isohyet ≈ 0 °C mean-January isotherm ≈ 33°N** — one
  threshold with a triple coincidence, ideal for a wheat/rice divide.
- Africa: **tsetse belt is the master constraint** (Alsan 2015, AER) — no draught animals → no
  plough → no manure → long-fallow shifting cultivation → low density → weak states; flag where
  rainfall >1000 mm and altitude <1500 m, with a highland exemption (Ethiopia is the exception
  that proves it). **Seed retention**: African millets 1–3% and teff ~1% of harvest vs European
  wheat 20–30% — a large, underused lever.
- Rotation is **gated by climate, not tech**: three-field's gain comes entirely from the
  spring-sown course, which needs April–June rain, so Mediterranean climates are capped at
  two-field until irrigation. Three-field ≈ **+33% sown area** (50%→67%), not doubled yields.

**Diffusion vs adoption (Theme C hook).** Diamond's axis rates give the physics — Philippines→
Polynesia 3.2 mi/yr E–W, SW Asia→Europe ~0.7, Mexico→US Southwest <0.5 N–S — best implemented as a
**3–10× penalty on north–south movement over a latitude-band adjacency graph**, not a distance
metric. But **adoption friction dominates once contact exists**: chili ~50 yr to spread globally,
potato in Europe ~130, cassava in Africa ~250, tomato in Italy/China ~350. Friction factors per
(crop, culture) pair: processing requirement, similarity to existing practice (maize ≈ sorghum →
instant in Africa), an incumbent occupying the niche (yam blocks sweet potato), missing downstream
tech, and elite-first prestige channels. Build only Diamond's model and crops spread far too
uniformly. Champa rice (1012 CE) is the ideal *diffusion-event* template: state procures seed,
distributes free with written instructions, initially lower-yielding, bred up over decades.

**Free full-text references** (all verified accessible): Cambridge World History of Food on
archive.org (greppable; ~60 staple chapters + a dictionary of >1,000 plant foods — good for
narrative and origins, but it has **no tabulated climate tolerances**); NRC *Lost Crops of the
Incas* (30+ Andean crops) and *Lost Crops of Africa* I–III (~49 species) — free, and they cover
exactly what mainstream references under-serve.

**Unverified — do not ship without checking:** Slicher van Bath's yield-ratio sequence
(3.7/4.7/7.0/10.6) could not be confirmed; quantitative manuring rates; chinampa carrying-capacity
figures behind the widely-repeated "3 t/ha".

#### G1b — Crop schema (proposal at `config/drafts/crop_schema_v1_proposal.jsonc`)

**DF's actual contribution is the product fan-out, not the agronomy.** One `[PLANT:X]` block holds
several `USE_MATERIAL_TEMPLATE` sub-materials (STRUCTURAL/DRINK/MILL/THREAD/SEED/LEAF/OIL/EXTRACT)
and the product tokens are pointers into them (`LOCAL_PLANT_MAT:<mat>`) — that is the whole
one-to-many mechanism, and it generalizes to a single `process` enum so adding a workshop doesn't
require a new token. Two independent time systems worth copying: `GROWDUR` (farm-plot maturation)
vs `GROWTH_TIMING` (an absolute annual window for a detachable sub-product like fruit).

**What DF does NOT model** — and where our substrate is already ahead: soil pH; soil beyond a
binary plantable/not; yield curves (harvest is a fixed stack, not a function of conditions);
irrigation or any ongoing water budget; rotation, nutrient depletion, fallow; multi-year perennials;
disease/blight; and *any* temperature or precipitation numbers at all — biome flags stand in for
climate, so there is no °C anywhere in DF's plant raws. **Do not crib DF's envelope model; crib its
product graph.**

**RimWorld's floor**: the entire envelope is one fertility scalar (`fertilitySensitivity` 0–1), one
light scalar (`growMinGlow`), and a *global* temperature band shared by all plants (optimal 6–42 °C)
— six numbers total, and it sustains a whole game. Useful as a complexity floor.
**Anno's gate**: island fertility is a hard boolean — a farm simply cannot be placed. The lesson is
that a hard gate plus a rare expensive unlock reads better than a soft gradient.

**Fantasy crop design rules** (extracted from DF's six underground crops and RimWorld's
devilstrand/psychoid):
1. One twist deep from an obvious real referent (cave wheat = wheat, underground).
2. Every crop fills a *distinct non-substitutable slot*. DF's six: staple / grain / fiber / sugar /
   dye / spice+oil. The fantasy is in the slot, not the name.
3. Value lives in processing, not the plant (sweet pod raw = 2, milled = 20).
4. Cost is paid in a real currency — time, skill, or tech gating (devilstrand: 22.5 grow-days,
   full fertility sensitivity, research + skill-10).
5. It obeys the same constraint schema as real crops; it just occupies an extreme corner of the
   parameter space, never an exemption from the rules.

**Schema decision**: v1 uses **absolute units** (°C, mm/yr, days), deliberately breaking the 0–1
house convention, because EcoCrop and our climate output are both absolute and normalizing would
mean two lossy conversions. This closes follow-up item 3 below. The `processing_gate` field is the
one thing none of the surveyed games model and is what lets maize, cassava, acorns, and chuño share
a representation.

**Validated 2026-08-02 — EcoCrop's `KTMP` is growing-season frost sensitivity, NOT winter
hardiness** (`config/drafts/CROPS_V1_VALIDATION.md`). Established first-hand from the Recocrop
package rather than secondary docs: `man/parameters.Rd` defines KTMP as "the temperature below
which the plant dies" and opens with *"User beware: these parameters are expert opinion"*; and
`src/ecocrop.cpp`'s `movingmin_circular()` only ever scores months inside the candidate growing
season, so **overwintering is structurally invisible to the model**. The 35-column schema has no
vernalization, sowing-season, or dormancy field, and no winter-type records exist — the rye/oats
inversion is systematic at genus level (every *Triticum* is 0 or −3, every *Avena* is −10/−15), so
choosing a different record cannot rescue it.

Consequences for the schema:
- **Split the field in two.** `frost_kills_at_c` (= KTMP, actively growing tissue, use as-is) and a
  new `winter_hardy_to_c` (cold-hardened dormant plant, `null` where there is no overwintering
  phase). Only the latter may gate placement of autumn-sown cereals and perennials. Rye is −4
  growing but −25…−30 hardened; that single distinction is the whole rye story.
- **Treat KTMP 0 as `null`, not as a number.** 819 of 1710 EcoCrop records (47.9%) are exactly 0
  and only one is NA, so 0 is largely a "not assessed / frost-sensitive" default. It currently
  affects 10 of our 21 crops — if any call site reads 0 as "dies at first frost", half the
  catalogue shares one arbitrary threshold.
- **Oats −15 is wrong under either reading** and must be overridden regardless.
- Fig-vs-olive: the *direction* was right but the magnitude wasn't — about one USDA zone (~5 °C),
  not 12 °C.
- Swap `highland_tuber` to EcoCrop's **"Potato, Bitter"** (*Solanum × juzepczukii*, TOPMN 6 /
  TOPMX 14 / KTMP −5) — genuinely alpine, and literally the chuño cultivar, which makes the
  processing gate geographically self-justifying.
- `madder`'s KTMP −12 is authored, not sourced, yet sits on the same scale as real values — mark
  or null it.

**Parse verified clean**: all 20 EcoCrop-sourced crops diffed field-by-field against the source
table — 13/13 numeric fields exact for 20/20 crops, zero mismatches. The defect is entirely
upstream in EcoCrop's semantics, not in our extraction. Keeping `envelope_source`/`envelope_ref`
in the schema permanently is what made this auditable; do not drop them.

**Superseded 2026-08-02 — a better EcoCrop source exists, and it already has the two-field split.**
A mirror survey (`config/drafts/ECOCROP_MIRRORS.md`) found that the Recocrop/dismo table we used
(1710 × 35, ~2019 — and dismo is byte-identical to Recocrop, not an independent source) is a
*reduced* copy. The `supersistence/EcoCrop-ScrapeR` scrape of FAO's actual datasheets has
**2568 records × 63 columns** (~2023), and its columns include
`Killing.temp..during.rest` **and** `Killing.temp..early.growth` as separate fields. Our single
`KTMP` collapsed the two inconsistently, keeping the early-growth value for exactly the crops whose
winter hardiness matters:

| crop | old `KTMP` | FAO during rest | FAO early growth |
|---|---|---|---|
| wheat | 0 | **−20** | 0 |
| rye | 0 | **−18** | −1 |

Verified first-hand against the vendored file. **The hand-sourced "consensus" overrides are
therefore mostly unnecessary** — the data we wanted exists upstream. Also gained: ~830 species,
Climate.Zone, photoperiod, life span, latitude/altitude envelopes, cropping system, uses, and
per-crop **free-text `Notes`** (2049 filled, mean 822 chars) that discuss winter/vernalization for
263 crops — no mirror has an explicit winter-vs-spring column, but the notes carry it in prose.

Actions: re-derive `crops_earthlike_v1.json` from `cropbasics_scrape.csv.gz` joined to
`crop_view_data.csv.gz`; map the two killing-temp columns onto `frost_kills_at_c` /
`winter_hardy_to_c`; keep the 2019 table only as a cross-check (the vintages disagree on 4–8% of
values per field, so never mix them silently). Both tables are vendored under
`config/drafts/ecocrop_source/` with a dependency-free RDS reader.

**Residual override still needed — FAO's during-rest column does not respect the known ranking.**
The agronomic consensus, sourced: *"winter rye the most cold-hardy, followed by winter wheat,
winter barley and oat"* — Karen Tanino, Univ. of Saskatchewan
([Top Crop Manager](https://www.topcropmanager.com/advancing-cold-hardiness-of-winter-wheat-and-rye/)).
But FAO 2023 gives rye −18 and wheat −20, i.e. it makes **wheat hardier than rye**, inverting the
top two. So the two-field split fixes the gross error (0 °C for both) without fully fixing the
ordering — apply a small ordering override for the winter cereals and note it, rather than assuming
the better source is correct throughout. Also useful from the same source: the **crown** is the
organ that overwinters (leaves and roots die back), and hardiness differs *between tissues within
the crown* — confirming the split is conceptually right and that even finer granularity exists
upstream if ever needed.

**Licensing: not a blocker.** Facts are not copyrightable (*Feist*, 1991 — "sweat of the brow"
rejected; compilations get thin protection only for selection/arrangement, which does not reach the
underlying data). Using EcoCrop's envelope numbers in the catalog and the simulation is fine, and
derived output is ours. The only narrow question is redistributing the *tables verbatim* — which is
what `ecocrop_source/` does, appropriately, as the provenance record; revisit only if the repo goes
public, or ship just the derived catalog. Attribute FAO (not the mirrors; the scraper repo has no
LICENSE and the GPL covers the Recocrop package, not the data) — the schema's `envelope_source` /
`envelope_ref` fields already do this per crop. Caveats that don't block: the EU sui generis
database right is a different analysis from US law, and FAO's terms of use are contract rather than
copyright (in practice permissive, likely attribution-only). See
`config/drafts/ecocrop_source/README.md` for the full reasoning.

#### G1c — Fantasy crops: add them with their consuming chain, never alone

**Decision: fantasy crops are a goods-chain design problem, not a crop design problem.** A reagent
with no alchemy system is an herb with a strange name. So the rule is: *a fantasy crop enters the
catalog at the same time as the production chain that consumes it, justified by a specific output.*

Assessment of the current 7 (`crops_fantasy_v2.json`) against "does removing this remove a
possibility from the world?":
- **Earn their slot**: `cave_fungus` (`light_min: 0.0` — the only crop that grows without sunlight,
  so subterranean agriculture and underground settlement become possible at all; also yields
  `preserved_food`), `frostwheat` (winter-hardy −32 °C, grows to −7 °C — no real *staple* reaches
  there, rye stops at −18, so it opens polar/boreal grain).
- **Marginal**: `duskvine` (`light_min: 0.08`, near-dark luxury vine — keep only if the
  subterranean economy is a thing we want), `mireroot` (taro already covers wetland staple).
- **Cut 2026-08-02** (done — catalog now holds 4): `ember_pepper` (max 51 °C vs real date palm's 52
  — a hot pepper with a name), `silverleaf` (tea reskin, same `herbs` product as ember_pepper),
  `skywheat` (nothing exceeds barley). `herbs` remains a valid good, now supplied only by real
  crops.

**Principle established: reskins are per-world customization content, not default catalog
entries.** A world that wants elven silverleaf gets it by naming a real crop in that world's
config — which is the culture-cultivar naming path above — rather than by shipping a renamed tea in
the shared catalog. The default catalog carries only crops that change what the world can do; the
flavor layer is per-world and costs nothing to vary.

**Flavor and mechanics are different jobs and only one needs a catalog entry.** Culture-specific
cultivar names on the real 82 crops ("the elves call flax *silverleaf*") cost one string each,
have zero simulation cost, and `namegen` already generates them per culture. Reserve catalog
entries for crops that change what the world can do.

**The justification pattern — reagents as chokepoints.** The strongest reason to invent a crop is
to create a *geographic chokepoint* for a chain the world needs. Structurally this is already
supported: our goods have raw/processed/finished categories with declared inputs, so
`herb → apothecary → salve` is the same machinery as `grain → mill → flour`. The payoff is that a
reagent growing in only three places, needed by every healer, simultaneously creates a trade route,
a strategic resource, and a casus belli — all reusing systems that already exist. That is the DF
lesson restated (sweet pod: worth 2 raw, 20 milled; strategic scarcity as a war cause).

Design order, therefore:
1. Define the fantasy chains first — what does an apothecary/alchemist/enchanter actually produce,
   and who demands it? (Potions, salves, balms, inks, incense, preservatives, alchemical reagents.)
2. Derive the input crops from those outputs, one crop per chain that needs a distinctive input.
3. Tier them against the existing `value_class`: common herbs (widely grown, cheap salves) vs rare
   reagents (single-region, luxury, chokepoint).
4. Theme the reagents with the **sphere ontology** (Theme E) so the myth layer and the economy share
   one vocabulary — a frost-sphere reagent, a death-sphere reagent. This is URR's "chains of
   meaning": nothing should relate only to itself.

Products worth designing chains around, because they hook systems we already simulate:
a **preservative** (extends how far perishables travel → literally reshapes the route network),
a **monopoly dye or spice** (single-source luxury → routes worth defending), a **structural fiber**
(feeds the existing `cordage`/`ships` chain).

#### G1d — Subterranean agriculture (the one domain where invention is mandatory)

This is the clearest case for invented crops: `light_min: 0` is biologically impossible for an
autotroph, so no real record can support underground civilization. It also passes the bar from
G1c decisively — it makes an entire settlement geography possible that otherwise cannot exist.

**It is a slot SYSTEM, not a crop.** DF's underground set is six crops filling six
non-substitutable roles — staple (plump helmet), grain (cave wheat), fiber (pig tail), sugar
(sweet pod), dye (dimple cup), spice+oil (quarry bush) — which is precisely why a fortress can be
self-sufficient. We currently have two (`cave_fungus` staple, `duskvine` luxury wine) and would
need roughly: fiber, dye/pigment, oil or fat source, fodder for subterranean livestock, and a
storable secondary staple.

**The energy question is the real design fork.** Fungi are heterotrophs — they consume organic
matter rather than making it. Two options, and they produce very different worlds:
- *Autarkic* (DF's choice): handwave it, crops grow on muddy rock. Underground holds become
  self-sufficient and isolated.
- *Dependent* (recommended): subterranean crops require imported organic substrate — timber,
  straw, waste, guano — so underground settlements are **structurally trade-dependent on the
  surface**. This is mechanically richer at zero new machinery: it makes every underground hold a
  trade node with a permanent import need, gives surface polities leverage, and turns a siege into
  a food crisis. It reuses the existing goods/route system rather than adding one.
Exception worth allowing: geothermal or chemosynthetic vents as rare, genuinely autarkic sites —
scarce enough to be strategically valuable.

**Architecture gap — the blocker is the place, not the crops.** There is currently *no*
underground representation in the codebase: no lithology, no karst, no cave, no depth dimension.
The only trace is a `surface` environment tag in polity profiles, implying a counterpart that was
never built. The mesh is a spherical surface, and adding a true depth dimension is not warranted.
Cheapest viable path:
1. A per-cell **`subterranean_potential`** scalar [0,1] derived from fields we have or can cheaply
   add — relief/elevation, volcanism (already computed in landgen), and a minimal **lithology**
   layer (limestone/karst is what actually makes caves; we have no rock type at all today, so this
   is the one genuinely new input).
2. Underground settlements are then ordinary surface cells with high `subterranean_potential`,
   carrying the `subterranean` env tag — no new dimension, and the settlement/trade/polity stack
   works unchanged.
3. **Depth bands as a crop tag only** (shallow/deep), the analog of DF's `UNDERGROUND_DEPTH:min:max`
   — purely a gating string on the crop, not a simulated axis.

Sequencing: lithology + `subterranean_potential` first (it is also independently useful — stone,
ore, clay, and gem endowments all want rock type), then the underground crop slot set, then the
`subterranean` settlement/culture profiles that already have a name reserved.

#### G1e — Subterranean livestock

**Source verdict: the biology must be invented, because real cave ecology cannot support an
economy.** Real guano food webs sustain beetles and blind fish — biomass orders of magnitude below
a city. Surface photosynthesis captures vastly more energy per unit area than any bat colony can
import, so a "realistic" underground settlement starves. **Invention here is load-bearing at the
energy level, not decorative.** Use DF and D&D (deep rothé, purring maggots) as naming referents
only; per G1c that content belongs in per-world config.

**The design constraint that makes these distinct from surface animals: they must not eat the
staple.** If subterranean livestock consume the fungal crop, they compete with people and are
strictly a loss. They earn their place by converting *inedible* substrate — cave moss, fungal
waste, detritus, guano — into food, which is exactly what surface grazers do with grass.

Proposed slot set (names are placeholders; worlds skin them):
1. **Detritus grazer** (cattle analog) — eats fungal waste and cave moss → meat, milk, hide, and
   **tallow**. Tallow matters: underground civilization needs artificial light, so fat is a
   structural requirement, not a luxury. This single dependency does a lot of work.
2. **Tunnel pack beast** (donkey analog, low clearance) — hauls ore and goods where an ox cannot
   fit. Tunnel geometry, not terrain, is the differentiator; it slots into the existing
   `land_transport` ladder with a `clearance` constraint.
3. **Silk/fiber producer** (cave spider or silkworm analog) — the only underground fiber source,
   since flax and cotton need sun. Feeds the existing `cordage`/`cloth`/`ships` chains.
4. **Guano colony** (bat analog) — semi-domesticated; forages the surface, returns underground,
   yields guano (fertilizer for the fungal beds) plus meat. **This is the elegant one**: it closes
   part of the substrate loop that G1d otherwise fills by import, so a hold with guano colonies is
   more self-sufficient than one without — a real geographic advantage worth siting for.
5. *Optional* **riding/war beast** — only if the military layer ever needs it; skip until then.

Interaction with G1d's dependency model: the two food chains compound. A hold needs organic
substrate for crops *and* fodder for livestock, so guano colonies and geothermal vents become the
scarce things that determine whether an underground settlement is autarkic or a permanent importer.
That is the interesting political geography, and it comes free from the existing trade system.

**Invent exactly one thing: an underground primary producer.** The single premise to add is an
organism that fixes energy without sunlight *at agricultural rates* — real chemosynthesis exists at
vents but nowhere near the required scale, so this is the fantastical element. Everything else
(crops, livestock, guano supplements) then follows with internal consistency. This is the
"one twist, consistently propagated" discipline from §G1b and URR's chains-of-meaning principle:
one invented premise, many derived consequences, no further exemptions.

**Recommended premise: radiant minerals — a photosynthesis substitute emitted by rock, which
depletes.** Certain minerals emit an energy the plants use exactly as they use sunlight. Two
alternatives were considered and this one wins clearly:

- *Chemolithotrophy* ("the stone feeds them" — organisms consume minerals directly): coherent, but
  needs a new metabolic pathway modelled from scratch.
- **Radiance (recommended)**: because it is a *sunlight substitute*, it reuses machinery that
  already exists. The crop schema already carries `light_min`; a glowing vein simply supplies that
  light underground, and the entire suitability model transfers unchanged. It also falls off with
  distance from the source, so farm geometry is a radius-and-decay problem — the same shape as the
  physical-radius kernels already implemented in `hydrology_resolution.go`.

Either way the **lithology layer from §G1d does double duty**: rock type is needed anyway for
stone/ore/clay/gem endowments and cave formation, and now also determines where underground
farming is possible. One new field, three payoffs.

**Depletion timescale is the design dial, and it is the most valuable part.** Radiant veins decay,
and the half-life is a per-mineral parameter:

| decay span | resulting world |
|---|---|
| decades | boom-and-bust holds, prospecting rushes, ghost towns within one lifetime |
| centuries | a full civilizational arc — rise, peak, decline — inside recorded history |
| millennia | ancient stable holds; the "elder" underground powers |
| effectively renewing (rare) | the contested prize; worth wars |

This gives underground civilization something the surface structurally lacks: **a clock.** Surface
agriculture is renewable, so surface polities decline for political reasons; underground polities
decline for *physical* ones, on a schedule the generator sets. That asymmetry is a narrative engine
— it manufactures ruins, migrations, deep-delving expeditions to find fresh veins, and conflict
over the slow-decaying ones, all of which the history layer (Theme C) consumes directly. It also
makes "why is this magnificent hold abandoned?" answerable by simulation rather than by fiat.

**The emission sits elsewhere on the electromagnetic spectrum — it is not light.** This is the key
refinement: the minerals radiate outside the visible/PAR band (ionizing is the most productive
choice), so ordinary plants *cannot* use it at all — their pigments are tuned to 400–700 nm. Real
crops underground are not merely inefficient; they are unpowered. Adaptation is therefore
mandatory, and it is a specific, nameable adaptation rather than a hand-wave.

**This is barely fantastical, which is a strength.** Radiotrophy is real: *Cladosporium
sphaerospermum* — the fungus colonising the Chernobyl reactor — uses melanin to harvest ionizing
radiation. The invented part is only that it works at *agricultural* productivity. Everything else
follows from physics we can borrow rather than invent:

- **Decay is literal half-life.** The "decades → millennia" dial from the table above is exactly
  the range of real isotope half-lives, so the depletion mechanic is not an arbitrary parameter —
  it is the same number physicists use, and different minerals naturally sit at different points.
- **Falloff is inverse-square**, which motivates the radial farm zoning below on physical grounds.
- **Shielding is real**: rock and water attenuate, so farm architecture becomes a deliberate
  problem of exposing beds while shielding the people working them.
- **Bioaccumulation is literal** — radionuclides concentrate in tissue — so food-safety processing
  is a genuine requirement, not a flavour gate.
- **The visual signature is derived, not invented**: melanin-analog pigments mean underground crops
  are *black* or near-black. The look of the biome falls out of its mechanics.
- **Farming carries a human cost.** Tending the beds is hazardous, which raises questions the
  social layer can answer — who farms, with what shielding, on what rotation, and at what life
  expectancy. That is a genuine cultural differentiator no surface biome produces.

With that established, the crop-side adaptations:

1. **New crop field: a `radiance` envelope** (`abs_min`/`opt_min`/`opt_max`/`abs_max`), exactly
   parallel in shape to `temp_c` so it reuses the same suitability code. Surface crops have
   near-zero radiance efficiency — they *can* survive on it but at a heavy yield penalty, so they
   are never the answer underground. Adapted natives have a real envelope and full yield.
   Underground agriculture is therefore **sorted by radiance the way surface agriculture is sorted
   by temperature** — a different organizing axis, not a different value on the same one.
2. **Radial geometry instead of planar.** Sunlight is diffuse and uniform across a field; radiance
   is a point/line source with falloff, so farmland is a *shell* around a vein. That yields inner /
   middle / outer zones with different crops in each — the near-vein core may be too intense for
   most plants (a hostile core with specialist crops), the outer edge too dim for staples. Surface
   farming has no equivalent of this zoning, and it reuses the existing radius-and-decay kernels.
3. **Bioaccumulation makes the products distinct — the real payoff.** Radiance is magical
   radiation; crops concentrate it. That splits the catalog by *what the concentration is for*:
   - **Food crops must shed or tolerate it** → several require a `processing_gate` (leaching,
     curing) before they are edible. This is exactly the nixtamalization pattern, and the field
     already exists.
   - **Reagent crops are valuable precisely because they concentrate it** → alchemical inputs,
     glowing dyes, and self-luminous products. Underground can manufacture things the surface
     cannot, which is the §G1c principle (value lives in products) applied at the biome level, and
     it supplies the chokepoint goods the alchemy chains want.
4. **Decay migrates the zones inward.** As a vein dims, the outer shell fails first, then the
   middle — so a declining hold loses farmland progressively from the rim, forcing "which fields do
   we abandon" decisions rather than a single cliff. Gradual, legible, and it gives the history
   layer a slow-motion crisis to narrate.

**Net: the two crop sets are mutually non-substitutable, in both directions.** This is the point of
the whole premise, and it should be enforced in the schema rather than left as a tendency:

- **Surface crops cannot go underground.** Their pigments cannot absorb the emission band at all,
  so they are unpowered, not merely low-yield. A hold cannot solve a famine by planting wheat in a
  gallery — it must import, which is what makes underground polities permanent trade participants.
- **Underground crops cannot go to the surface.** Radiotrophs need an ionizing source that daylight
  does not provide, and their melanin-analog pigments are built for the wrong band. They also
  bioaccumulate, so their food products carry processing requirements surface crops do not. You
  cannot relieve an underground famine by farming the valley above either.

Schema consequence: `light_min` and the new `radiance` envelope are **two separate energy channels,
and a crop draws on exactly one.** A crop with a radiance envelope and no light response is
underground-only; the converse is surface-only. Do not model radiance as "light by another name" or
the substitution sneaks back in through the suitability code.

The result is two genuinely distinct agricultural systems that trade with each other out of
structural necessity rather than convenience — surface grain and fibre flowing down, underground
reagents, luminous goods, metals and stone flowing up. That asymmetric, non-optional exchange is
the thing a reskin could never produce, and it comes almost entirely from one honest physical
premise.

Consequences that fall out for free:
1. **Underground farms are sited by rock type and vein proximity, not sunlight** — a genuinely
   different geography from the surface, rather than a reskinned one. Prospecting for fresh veins
   becomes a real activity, and deeper delving a real motive.
2. **Depletion becomes possible on a designed schedule.** A hold outlives its veins, which produces
   migration, abandoned holds, and ruins — narrative material the history layer (Theme C) can use,
   and something surface agriculture only weakly offers.
3. **Autarky becomes a spectrum with real causes.** Rich strata → self-sufficient underground
   civilization; poor strata → permanent importer dependent on surface trade. Guano colonies and
   geothermal vents survive as *supplements* that shift a hold along that spectrum, not as the
   energy base.
4. **One nutrient-budget model still serves both environments.** Surface rotation (fallow / manure
   / legume fixation) and underground agriculture (lithotrophic yield / depletion / imported
   substrate) are the same accounting with different terms — §G1a established that pre-modern
   nitrogen sources are mostly *transfers*, and that new input is what breaks the ceiling. Here the
   new input is mineral rather than biological. Implement the budget once; skin it per environment.
5. **Carrying capacity stays hard and legible** — bounded by strata richness plus imports, so an
   underground hold's size cap and trade dependency are quantitative rather than hand-waved.

#### G1f — Don't hardcode the magic: define a subterranean energy-source *interface*

Radiotrophy (§G1e) should be the **default, not the mechanism**. The simulation never needs to know
it is radiation; it needs five parameters. Define those as the interface, ship radiotrophy as the
stock implementation, and let a world — or a *region within* a world — select a different source.

**The interface (all a source must supply):**

| field | meaning |
|---|---|
| `magnitude` | per-cell primary-production potential, drives yield |
| `falloff` | spatial decay from the source (inverse-square, linear, uniform, patchy) → farm geometry |
| `depletion` | decay rate; may be zero (renewing) or cyclical |
| `adaptation_channel` | the energy channel crops key on; a crop draws on exactly ONE channel, which is what keeps surface and underground crops non-substitutable (§G1e) |
| `hazard` | cost to workers: none / radiation / heat / mutation / toxicity |
| `accumulates` | whether products concentrate it → processing gates and reagent value |

**Discipline: an alternative must be a parameterization, never a new code path.** If a proposed
source needs new simulation code, it does not belong in the interface yet. This is the specific
failure the postmortems warn about — URR froze 34 generators into a fixed order and spent a decade
paying for it.

**Alternatives that fit the interface unchanged**, each producing a materially different world:

| source | falloff | depletion | hazard | what it changes strategically |
|---|---|---|---|---|
| **Radiotrophic minerals** (default) | inverse-square from veins | half-life, decades→millennia | radiation | prospecting, hazardous labour, ruins on a schedule |
| Chemolithotrophic strata | ore-body extent | consumption by use | heavy-metal toxicity | farming *is* mining; the two compete for the same rock |
| Geothermal / thermosynthetic | vent proximity | slow cooling | heat | holds cluster on volcanism; §G1d's rare autarkic vents become the norm |
| Ley-line / ambient field | ley geometry, not point-source | zero or cyclical | mutation | linear farm corridors; renewable, so no decline clock — ancient stable holds |
| Deep-root parasitism | surface vegetation *above* the cell | tied to surface deforestation | none | **vertical coupling** — logging the forest starves the hold beneath it |
| Wrought/divine light | artificial placement | destructible | none | the source is a made thing, so it is a war target and a ruin |

Two payoffs beyond variety:
1. **Sources can coexist within one world**, differentiating underground *regions* from each other
   rather than only differentiating worlds. A radiant-vein hold, a ley-corridor hold, and a
   root-parasite hold under a forest are three different economies on one map.
2. **It hooks the sphere ontology (Theme E)** — a death-sphere underground can run on something
   different from a fire-sphere one, so the myth layer and the agricultural substrate share a
   vocabulary instead of being bolted together.

**Keep the derived consequences attached to the source, not to the biome.** Black melanin-analog
crops, food-safety processing gates, and hazardous-labour social structure are *radiotrophy's*
consequences. Express them as source attributes (`hazard`, `accumulates`, pigment signature) so a
ley-line world gets its own coherent look and costs rather than inheriting radiation's.

#### G1g — Correction: autotrophy is not the common case

A survey of 19 mechanisms across fiction, games, and real underground settlements
(`config/drafts/SUBTERRANEAN_SOURCES.md`) exposed a flaw in the six above: **they all assume the
underground produces its own food.** The dominant answers in both fiction and reality do not.
Add these; they are distinguished by *political relationship with the surface*, which is where the
design leverage actually is:

| mechanism | precedent | why it matters |
|---|---|---|
| **Mineral rent** — sell metal/gems, buy grain | Erebor & Dale, Moria, WH dwarfs | The single biggest gap. The dominant fantasy answer, and it needs no underground agriculture at all. Creates mutual dependence, embargo as a weapon, and intermediary market cities. |
| **Below-ground housing, surface fields** | Derinkuyu (verified), Coober Pedy, Matmata, yaodong | The realistic default — real underground settlements were surface farmers living below for thermal or defensive reasons. **Dissolves the "separate underground polity" assumption entirely.** |
| **Allochthonous detritus** | real cave ecology, marine snow | Dependence with *no diplomatic channel* — the surface has leverage it does not know it holds. |
| **Predatory husbandry** | Morlocks/Eloi | The underground protects and cultivates surface prosperity while suppressing its development. Nothing else produces this stance. |
| **Vault stockpile / failing legacy infrastructure** | *Silo*, Fallout, *The Machine Stops* | Isolation by ideology on a certain clock, ending in forced emergence. In the *Machine Stops* variant the engineer caste becomes sovereign — competence as the political axis. |
| **Non-eating population** | undead/construct polities | Immune to embargo, siege, and famine — removes every economic lever the surface has. |
| **Lithovore fuel export** | Oxygen Not Included (hatch→coal) | **Inverts the dependency**: the surface buys energy from below. |
| **External metropole resupply** | Deep Rock Galactic | A non-sovereign underground — company outpost, useful for colonial/extractive scenarios. |

**Adopt as a rule, not a mechanism: every fungal economy must name its substrate.** Fungi convert;
they do not produce. DF's plump helmets and deep rothé are unexplained autotrophy — any fungal
underground must pick what feeds it (detritus, imported biomass, or one of the six producers).
This is the single cheapest discipline for keeping underground economies honest.

Also noted: hydrothermal vents are a variant of chemolithotrophy but add a *geography* modifier
(point-source oases; decade-scale vent death forces nomadism); Nidhogg gnawing Yggdrasil's root is
the mythic cite for root-parasitism. Deliberate scarcity (Silo's lottery, Ember's rationing) is a
*policy layer*, not a source.

**Correction (browser-verified): faerzress DOES feed the Underdark — it is a direct precedent for
§G1e, not merely for the ambient-field option.** The survey's claim that it "canonically feeds no
one" is refuted by the Forgotten Realms wiki: faerzress is *"a magical radiation only found in the
Underdark… areas of high concentration would manifest physical signals… visibly glowing rock
formations"*, and on plant life: *"Since the Underdark lacked food or light for plant life and
fungi, many plants and fungi found ways to live on a diet of faerzress. It was the sole food source
of a plant-like life form called magivores, which could serve as a foodsource for Underdark
settlements."*

That is precisely the chain we derived independently — ambient radiation from glowing rock →
organisms adapted to feed on it → those organisms feed settlements. Published precedent for the
radiotrophic premise, and worth citing as such.
([Faerzress, Forgotten Realms Wiki](https://forgottenrealms.fandom.com/wiki/Faerzress))

Still unverified: Fallen London's false-stars / Neath sustenance (wiki fetch returned nothing
usable).

*Provenance caveat*: the session's web-search budget was exhausted before this survey ran, so most
entries are model knowledge marked unverified in the source file. Derinkuyu and the Dwarf Fortress
entries were fetch-verified; Underdark canon, Fallen London's false-stars, and Sunless Sea need
confirmation before being cited as fact.

### G2 — Domesticates & wildlife species
Current state: aggregate game/grazing/pelt/timber productivity fields; no species.
- Domesticate catalog with biome envelopes and yields (milk/meat/wool/draft/mount): the
  horse/camel/yak/llama distinction is what gives pastoral cultures their flavor, cavalry its
  geography, and caravans their mode parameters (a camel-string entry lowers desert land-route
  costs — direct hook into existing route cost fields).
- Wildlife species lists per biome sampled from the existing productivity fields (annotation,
  not simulation); hunting/depletion and monster-prey availability read the same fields.
- Megafauna as named huntable populations — feeds Theme C events (great hunts, ivory trade) and
  Theme E provenance goods.

**Draft delivered**: `config/drafts/domesticates_v2.json` — **31 entries** (27 real + 4
subterranean) in absolute units (°C bands, mm/yr floors, altitude m, litres/day, kg pack load,
kgf draft pull), replacing the 12-entry normalized v0. v0 had **zero boreal** domesticates and no
tropical bovine; reindeer, goose, zebu, elephant and guinea fowl close both. Supplies 11 new goods
(dairy, eggs, honey, beeswax, **tallow** — 12 emitters, the lamp-oil source underground —
draft_animals, warhorses, ivory, guano). **The silk chain is now closed**: crops v2 emitted `silk`
through a `sericulture` gate whose `silkworm_shed` had no occupant; `silkworm` (with
`requires_host_crop: "mulberry"`) fills it. All four `animalRequirement` references in
`land_transport_earthlike.json` resolve via a `legacyAliases` map.

**Diamond's Anna Karenina criteria earned their place as a design filter, and as modelling signal:**
- They *killed* the obvious cave-spider silk producer on the diet criterion — a predator needs ~10×
  its own biomass in prey and so competes with people for protein, which is exactly §G1e's "must
  not eat the staple" rule arriving independently. Re-specified as a detritivorous moth larva.
- `guano_colony` fails disposition and panic response outright and is kept anyway, justified by the
  honeybee precedent: **a colony is an installation, not a herd** — which is what makes the §G1e
  slot defensible rather than hand-waved.
- **Real animals fail the criteria too, and that is useful rather than embarrassing**: elephant is
  capture-dependent (a wild-forest dependency, not a self-sustaining herd — so elephant cultures
  need standing forest), mule is a sterile hybrid forcing maintenance of both parent stocks, and
  dog has the highest feed competition in the catalogue.

Two follow-ups noted: `horse-wagon` currently resolves to `horse_war` and should point at
`horse_draft`; `pack-camel-string` should select bactrian on cold-desert routes. Also, yak required
an altitude **minimum** (~2000 m) — a field shape no existing envelope anticipated.

### G3 — Monsters & megabeasts
The DF model is the proven one and fits us unusually well:
- **Seeded populations spent down by history** — megabeast counts are world state, and era names
  derive from them (Age of Myth ends when the giants do). Cheap and self-narrating.
- Placement from fields we already have: low population + high relief + deep forest/swamp →
  lair suitability (DF's "savagery" field is the one physical-layer thing worth adding — a
  per-cell wildness scalar we can derive from population/route distance).
- **Mechanical hooks, all into existing systems**: a lair projects a danger field that raises
  route costs (trade genuinely reroutes around the dragon pass — our cost model supports this
  today); settlement suitability penalty; Theme C events (rampage, slaying quest → hero
  promotion, casus belli); hoards as artifact/provenance stores (Theme E).
- Procedural beasts from the sphere ontology (Theme E) rather than a bestiary database; keep a
  small authored core (dragon/giant/leviathan archetypes) + generated variants, DF-style.

### G4 — Vehicles (historical and fantastical)
- **The vessel catalog is the template and it's already live.** Extend the same schema to:
  - Land transport ladder: porter → pack animal → wagon → caravan (terrain-dependent costs,
    road-quality response, payload); ties into G2 (which animals a region can field).
  - River craft (partially covered by `riverCapability` today — split into its own small catalog
    when river trade gets vessel-differentiated).
  - Tech progression (Theme C): which vehicle classes a polity can field per era — route
    networks then *evolve* across history ticks (the caravel unlock is a world-historical event).
- Fantastical transport as **new modes in the existing multimodal framework** — a mode is a cost
  field + endpoint rules, so airships (need mooring towers = new port-like nodes, weather-bound),
  gryphon post (tiny payload, relief-insensitive), or portal circles (point-to-point, hub
  capture) are additive. Gate them behind the magic-ubiquity/lawfulness sliders (Theme E) and
  give each real constraints so it *reshapes* trade geography instead of deleting it.

### G4a — Lithology (rock type): the missing layer four systems want

Draft: `config/drafts/lithology_earthlike.json` (18 classes) + `LITHOLOGY_NOTES.md`. Classes span
igneous volcanic/plutonic, metamorphic, and sedimentary clastic/carbonate/evaporite, each carrying
hardness, erodibility, **excavatability** (what separates a *dug* hold from a *found* cave),
permeability, karst and pseudokarst propensity (lava tubes are real caves and are not dissolution),
soil parent quality, and endowment multipliers keyed to the existing 11 affinity names.

**The blocker is plumbing, not physics — verified.** `landgen/terrain/generate.go:81-85` already
computes `coastlineR, mountainR, collisionR, arcR, ridgeR, trenchR` and `distFromCoast`, plus
`rPlate` and `plateIsOcean` — **all as local variables that are discarded**.
`PlanetGenerationDiagnostics` (types.go:530) exports only `HotspotChains` and `Hydrology`, and
climgen receives just two hotspot arrays. **Adding a `Tectonics` block to that struct unblocks ~80%
of lithology derivation with no new simulation.** That is a small, high-value change and should be
near the front of any Theme-G work.

Smaller pipeline gaps behind the rest: `BoundarySeeds.Rift` is computed but never rasterised to a
distance field (one missing call); there is no crustal-age/craton field (a seeded per-plate
`plateAge` is the cheap fix, a proxy works meanwhile); and **no deposition/basin-fill field** —
`ApplyFluvialErosion` moves material and records nothing, so a `depositionAccum` written during that
pass is the single most valuable addition, because it separates a *basin* from merely *flat*, which
four sedimentary classes need.

**Endowments that improve most** (all currently inferred from terrain proxies): **iron** is
`0.70·orogenicBase` — literally "tall mountains" — so banded-iron shields and coal-measures
ironstone are impossible, and coal+iron co-location cannot occur today; **gems** are
`rockiness×relief`, so they exist wherever mountains do and have no rarity structure at all;
**lead/silver** cannot generate the lowland carbonate-platform silver provinces that historically
mattered; **copper** gains both porphyry peaks and sediment-hosted redbed copper away from
volcanism; **stone** gains quality tiers and lime for mortar. Placer should inherit from *upstream*
cells via the hydrology `Receivers` graph.

**Unexpected fourth consumer — soil pH does not exist.** Verified: there is no pH field anywhere in
`climgen/soil.go` or the soil diagnostics. But the v2 crop catalog carries `soil_ph` envelopes
straight from EcoCrop (e.g. rye 4.5–8.2), so **crop pH gating is currently unusable** — we have the
crop side of a comparison whose world side is missing. Lithology's `ph_tendency` (basalt fertile
and near-neutral, granite acidic, limestone alkaline) is the only plausible source. This makes
lithology a prerequisite for using the crop envelopes fully, not merely a nice-to-have.

Derivation sketch (must be derivable, never hand-painted): score all classes from
craton / arc / collide / rift / margin / basin / unroofed and take the argmax, keeping the
runner-up. `unroofed = relief·elevation` separates arc andesite from porphyry intrusive and yields
granite batholith with no new field; marble is limestone overprinted by collision — an *ordering*
rule rather than a new one; kimberlite and BIF must be sparse point/patch processes or every shield
cell has diamonds; and the output needs low-frequency `procnoise` plus two neighbour-majority
passes (the `RegularizeCoastlines` pattern) or it comes out salt-and-pepper. Alluvium and loess are
overlays so bedrock survives beneath them for caves and radiant veins.

Radiance hook (§G1e): hosts are granite batholith, silicic tuff, and sandstone redbed, governed by
*richness × decay_rate ≈ constant* — so short-lived veins are rich and long-lived ones are lean.
Bonanza deposits key off a sandstone-on-craton **adjacency** (an unconformity — rare and findable,
which makes prospecting meaningful); disseminated granite/tuff veins are the millennia-scale
"elder hold" substrate.

### G5 — Materials
- Fantasy materials as endowment entries (the mithril vein is just a rare affinity layer) with
  strategic scarcity — the research's clearest lesson on making wars *about* something.
- DF's material/form/quality multipliers as the intrinsic-value prior for Theme D prices; sphere
  tags on materials feed artifact generation.

Sequencing note: G1/G2/G4-land are static-world safe and can proceed alongside A/B; G3's hooks
land best after Theme C exists (a monster that never does anything is scenery); G5 rides on
Theme D/E.

## Recommended sequencing

```
A (names)  ──►  B (cultures/religions, static)  ──►  C (history loop, incremental)
                                                       │
                        E (myth/artifacts; static-world gospels can start alongside B)
                                                       │
                                                  D (dynamic prices, inside C's tick)
                                                       │
                                                  F (characters)
```

Each earlier theme makes the later ones legible (names → readable events; cultures → event
flavor; history → things for prices and characters to be about). A/B/E-static are independent
of the resolution remediation; C should wait for the remediation to stabilize since it re-runs
the polity/trade stack per tick.

### Plumbing prerequisites — do these before more content

**The single clearest lesson from the 2026-08-02 drafting session: the data was easy and the
plumbing was the gap.** Three independent agents drafting three unrelated catalogs each hit the
same wall — the content was straightforward, but the engine could not consume it. These are small,
concrete engine changes that unblock disproportionate amounts of Theme G, and they should come
before any further catalog expansion.

| # | Change | Unblocks | Size |
|---|---|---|---|
| P1 | **Export a `Tectonics` block** from `PlanetGenerationDiagnostics`. `generate.go:81-85` already computes `coastlineR/mountainR/collisionR/arcR/ridgeR/trenchR`, `distFromCoast`, `rPlate`, `plateIsOcean` — and discards them as locals. | ~80% of lithology derivation (§G4a), which in turn unblocks endowments, caves, and soil pH | small |
| P2 | **Add a soil pH field.** None exists anywhere in climgen, yet the v2 crop catalog carries EcoCrop `soil_ph` envelopes — half a comparison. Source it from lithology parent material (P1). | crop pH gating; better soil realism | small, after P1 |
| P3 | **Derive per-crop endowment source fields** from crop climate envelopes. Only 20 *generic* source fields exist (`crop`, `herbs`, `resin`…), so spices/coffee/cocoa/tea/opium all place identically. | **the entire luxury-chokepoint design (§G1c)** — without it, new goods get chains but no geography | medium |
| P4 | **Harden the trade-goods validator**: reject unknown input names, detect cycles, verify inputs precede consumers. Array order is load-bearing today and unchecked. | safe catalog growth | small |
| P5 | **Write `depositionAccum`** during `ApplyFluvialErosion` (it moves material and records nothing). Separates *basin* from merely *flat*. | four sedimentary lithology classes | small |
| P6 | **Rasterise `BoundarySeeds.Rift`** to a distance field — computed, never converted. | rift-associated lithology | trivial |
| P7 | **Fix the L7 performance regressions** — `checkDepth` rescaled from an L7-calibrated constant (use base 4, not 15); measure the erosion diffusion-iteration cost (16× at L7, 64× at L8). | making L7 validation practical at all | small |

P7 is not optional if L7 is meant to be a routine validation target: the current tree takes
~2.5+ hours per L7 seed, which makes an 8-seed sweep a full day.

### Content areas still undrafted

For completeness — these have design notes above but no draft catalog yet: **G3 monsters and
megabeasts** (needs the wildness/`subterranean_potential` fields first), **G4 vehicles** beyond the
6-entry v0 land ladder (no river craft split, no fantastical modes), **G5 materials** (fantasy
materials as endowment entries), and **wildlife species lists** per biome (the non-domesticate half
of G2). None are blocked on anything except the plumbing above, and G3 in particular is better done
after Theme C exists, since a monster that never does anything is scenery.

## Ready-to-wire follow-ups (concrete gaps found 2026-08-02)

Small, well-defined items discovered while building the first drafts. None are blocked on the
big themes; all are blocked on the resolution sweep only insofar as they touch climgen.

1. **Polity size tiers don't exist — and the current size distribution won't support 8 of them.**
   `namegen` exposes 8 classes (empire, kingdom, principality, duchy, city_state, league,
   confederacy, tribe) but climgen has no named rank.

   *Measured from the level-6 wave-1 sweep (8 seeds, 38 polities, `territoryEq` = baseline-
   equivalent cells):*

   | | min | p25 | median | p75 | p90 | max |
   |---|---|---|---|---|---|---|
   | territoryEq | 70.8 | 125.3 | 167.5 | 223.3 | 255.8 | 384.3 |

   Mean 177.4, σ 70.0; influence spans only 5.05–8.22; **4–6 polities per world**.

   **Finding: every generated world is a handful of similar mid-size regional realms.** Largest
   to smallest is only ~5.4×, and there is no long tail in either direction — no hegemonic empire,
   no city-states, no tribal periphery. Three consequences:
   - Map to **3 tiers, not 8** (e.g. ≤120 / 120–260 / >260 territoryEq), and let the remaining
     namegen classes stay unused until the distribution justifies them.
   - Use **absolute thresholds, not per-world quantiles**. With 4–6 polities, quantile tiering
     would manufacture an "empire" in every world even when all six are the same size.
   - The flat distribution is itself a **worldbuilding gap worth its own investigation**: it is
     an artifact of every polity growing from one proto-civ seed under identical rules, with no
     conquest, inheritance, or collapse. Theme C (history) is what produces size *variance* —
     CK3-style partition succession and war are precisely the mechanisms that make some realms
     sprawl and others fragment. Tiering is therefore better done *after* the history loop
     exists; doing it now would encode the current flatness as if it were intended.

   Deferred on that basis; the measurement above is the input for revisiting it.
2. **Trade goods: 22 proposed additions, and this is the highest-value expansion available.**
   The v2 crop catalog (82 crops, `config/drafts/crops_earthlike_v2.json`) needs goods the
   economy does not have. From v1: `wine`, `olive_oil`, `plant_oil`, `fresh_fruit`, `dried_fruit`,
   `cider`, `root_crop`, `fodder`, `cordage`, `dyestuff`. From v2: `pulses`, `sugar`, `spices`,
   `silk`, `tea`, `coffee`, `cocoa`, `tobacco`, `nuts`, `opium`, `ale`, `fuel_fiber`.
   Two are load-bearing beyond filling gaps: `olive_oil` gives the *existing* `soap` good an input
   it currently lacks, and `cordage` gives the *existing* `ships` good an agricultural one.
   The spice/silk/coffee group matters most — these are historically the cargo that justified
   bulk-inefficient long-distance routes at all (high value per weight, geographically
   concentrated, universally demanded), and the multimodal routing and luxury categories are
   already built and waiting for them. Without these, Mediterranean polyculture and the entire
   luxury trade have nowhere to land, and non-cereal staples all collapse into `grain`.
   Draft: `config/drafts/trade_goods_expansion.json` — **25 new goods** (raw 8, processed 9,
   luxury 8) in exact `climgen.TradeGoodSpec` shape, taking the merged catalog 29 → 54, with five
   3-step chains matching the depth of the existing `iron_ore → iron_goods → weapons_armor`.
   Resolves every crop product including the loose `fungus`; adds `silk_cocoon` so silk has a real
   gate. Correction to an earlier note here: `soap` and `ships` **do** already have inputs — the
   patches are extensions (ships should *replace* `fiber` with `cordage` rather than add it, or the
   MIN-gate double-counts), supplied as an advisory `existingGoodPatches` block rather than edits.

   **Blocking finding — the chokepoint geography does not exist yet.** Verified: the live config
   uses only **20 generic source fields** (`crop`, `herbs`, `resin`, `fiber`, `pasture`, `timber`,
   `fish`, `game`, ores, `clay`, `salt`…) and there is **no per-crop source field**. So spices,
   coffee, cocoa, tea, opium and tobacco would all be placed by the same `crop`+`herbs`+`resin`
   blend — meaning adding the goods gives us the *chains* but not the *geography*. The design in
   §G1c (a reagent growing in only three places creates a route, a strategic resource and a casus
   belli) requires **endowment fields derived per-crop from the crop catalog's climate envelopes**.
   That derivation, not the goods list, is the real gate on luxury long-distance trade.

   Two further schema constraints worth knowing before building on this:
   - **`inputs` and `sourceWeights` are mutually exclusive in effect.** Verified at
     `trade_goods_polity.go:150-178`: `localPotential` is computed, then `if len(spec.Inputs) > 0`
     takes the manufacturing branch and never reads it. A good is either endowment-placed or
     input-placed, never both.
   - **`inputAccess` is a MIN over inputs with no substitutable inputs** — "olive_oil OR plant_oil
     OR tallow" is inexpressible; the only lever is need magnitude (small need ⇒ soft gate).
   - **Catalog array order is load-bearing and unvalidated**: inputs must precede consumers, and
     the validator checks neither unknown input names, nor cycles, nor ordering. Cheap hardening
     win — add those three checks.
3. ~~**Crop schema units decision.**~~ **RESOLVED** — v1 uses absolute units (°C, mm/yr, days);
   see §G1b and `config/drafts/crop_schema_v1_proposal.jsonc`. Remaining work is migrating the
   v0-draft catalogs (21 + 7 crops) to the v1 schema and sourcing their envelope numbers from
   EcoCrop rather than by hand.
4. **`namegen` wiring** (also in `namegen/README.md`): join `CultureBase` on polity culture
   profiles; name polities/capitals in `cmd/review_planets` output; name major rivers/seas/ranges
   using landgen feature IDs as keys. Package is standalone, tested, and unimported today.
5. **Draft catalogs are unwired** — `config/drafts/*.json` (21 crops, 7 fantasy crops, 12
   domesticates, 6 land-transport tiers) have no loader and no consumer. They are proposals for
   review, not live config.

## Guardrails (from the postmortems)

1. **Stop deepening the substrate** — SotE died rewriting climate three times while culture was
   a name generator. Our substrate is done; ship upward.
2. **No frozen pipelines** — URR's 34-generator fixed order forced decade-long hacks. Keep each
   theme a package with a data interface (the event log, the name table, the culture map).
3. **No monoliths** — DF's Myth & Magic "Big Wait" stalled a decade; they now ship it in slices.
4. **The log is the product; the chronicler is half the work** — budget sifting/rendering
   (summaries, legends output) as first-class, not as an afterthought.
5. **Economy last among equals** — two of the most sophisticated attempts (DF 40d, Vic2) are
   canonical failures; the proven minimal loop (Vic3 order-ratio + quantity inertia) is small,
   but only worth running once time exists.
