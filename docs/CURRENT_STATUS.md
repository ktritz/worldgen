# Current Status

Last updated: 2026-04-30

## Summary

The project is in active climate, civilization, trade, and review-diagnostics work.

Current trade/cache state is materially better than it was a few iterations ago:
- level-6 review caching is robust, phase-aware, and script-driven
- trade tuning no longer invalidates derived, civilization, or maritime review caches
- processed trade is no longer artificially suppressed at inter-polity level
- `paper` and `soap` now appear as real traded processed goods on the validated level-6 slice
- broader `ceramics` and `paper` dominance has been reduced with JSON-only tuning

The current working baseline should be treated as:
- cache/tooling stable enough for repeated diagnostics
- trade model good enough for validation-first iteration instead of blind tuning

## Recent Milestones

Recent committed milestones:
- `6823411` `Tighten ceramics and ships market tuning`
- `e9ea701` `Refine processed trade capability gating`
- `7dd7515` `Add processed trade path summary script`
- `6db391d` `Add trade sweep summary script`
- `6922339` `Add review cache manifest builder`
- `923b127` `Record cache phase summaries in refill outputs`
- `6f4e802` `Add phase-aware review refill progress`
- `dd95612` `Add refill heartbeat progress updates`

Key completed work behind those commits:
- broad review cache coverage across terrain, climate, derived, civilization, maritime, and economy layers
- atomic cache writes and stronger cache invalidation keys
- durable refill script with `progress.json` and `summary.tsv`
- cache manifest builder for post-run cache inspection
- focused sweep-summary scripts for trade diagnostics
- processed-trade capability gating that fixed `paper` / `soap` underexpression

## Current Validated Baselines

### Level-6 Cache / Refill

Clean refill and confirmation tooling exists and is working:
- `scripts/refill_review_cache.py`
- `scripts/build_review_cache_manifest.py`

Important sweep directories:
- `output/review_planets/sweeps/level6_clean_refill`
- `output/review_planets/sweeps/level6_clean_confirm`

What is established:
- first fills can still be expensive
- fully warmed replays are fast
- trade-only JSON changes now recompute only trade-good endowments and economy outputs without forcing derived, civilization, or maritime recomputation
- layer-by-layer hit/miss summaries are available in `summary.tsv`
- live phase information is available in `progress.json`

### Level-6 Processed-Trade Validation

The processed-trade fix is validated on:
- `output/review_planets/sweeps/level6_processed_capability_validation`

Validated seed slice:
- `4,7,42,58,84,91,123`

Current result:
- `paper` is now a real processed trade good
- `soap` is now a real processed trade good
- processed inter-polity trade is no longer limited to `woolens`, `preserved_food`, and `cloth`

Representative processed outcomes from that slice:
- seed `58`: `paper:s10.87/v12.34`, `soap:s8.06/v9.29`
- seed `84`: `paper:s4.63/v5.34`, `soap:s1.68/v2.05`
- seed `91`: `paper:s5.98/v6.61`, `soap:s1.95/v2.32`
- seed `123`: `paper:s0.69/v0.79`, `soap:s4.92/v5.76`

### Broader Level-6 Trade Validation

The broader validation sweep is:
- `output/review_planets/sweeps/level6_trade_broader_validation`

Validated seed slice:
- `4,6,7,42,55,58,84,91,101,123,128,144,177,202,255,314,512,777,999,1337`

What the broader pass found:
- the earlier 7-seed slice understated `ceramics` dominance
- `ceramics` was first-place cap winner in `10/20` seeds
- `paper` was top processed good in `18/20` seeds
- `ships` stayed healthy as strategic trade in `20/20` seeds

### Latest Trade Tuning Validation

The latest accepted trade-tuning validation is:
- `output/review_planets/sweeps/level6_trade_tuning_trial3`

What changed:
- JSON-only tuning in `config/trade_goods_earthlike.json`
- `ceramics` market production was narrowed by reducing conversion/context and raising input floors/dominance penalty
- `paper` kept production viable but had value, demand affinity, context, and profile affinities reduced

Status:
- run completed
- `20/20` seeds completed with `ok`
- summarized with `scripts/summarize_trade_sweep.py`, `scripts/summarize_good_paths.py`, and `scripts/summarize_processed_path.py`
- decision: values are reasonable; stop this trade-tuning pass

Key comparison versus `level6_trade_broader_validation`:
- `ceramics` first-place cap winners dropped from `10/20` seeds to `1/20`
- `ceramics` average cap-winner rank moved from `1.75` to `3.06`
- `ceramics` average winner count dropped from `31.80` to `23.94`
- `paper` top processed good dropped from `18/20` seeds to `16/20`
- `paper` average processed score dropped from `9.70` to `7.95`
- `paper` average processed score share dropped from `66.6%` to `62.4%`
- `soap` remained present in processed trade in `20/20` seeds
- `ships` stayed present as strategic trade in `20/20` seeds

Latest cap-winner aggregate:
- `ships`: present `19/20`, first `7/20`, avg rank `2.21`, avg count `29.05`
- `ceramics`: present `18/20`, first `1/20`, avg rank `3.06`, avg count `23.94`
- `soap`: present `15/20`, first `6/20`, avg rank `1.93`, avg count `28.47`
- `paper`: present `14/20`, first `0/20`, avg rank `3.29`, avg count `24.21`
- `weapons_armor`: present `11/20`, first `6/20`, avg rank `1.64`, avg count `31.91`

Textile-chain check:
- summarized with `scripts/summarize_trade_chains.py`
- `cloth->fine_clothing` is produced in `20/20` seeds with `0` blocked markets
- average `cloth->fine_clothing` made is `6.55`, average cloth market surplus is `30.30`, and average downstream demand gap is `0.01`
- current interpretation: cloth production is not blocked; cloth and fine clothing are mostly locally/regionally absorbed rather than becoming major inter-polity flow
- future review output now includes `tradeGoodPath` diagnostics for `cloth`, `woolens`, and `fine_clothing`

### Level-6 Polity Trade Validation

The polity-level validation sweep is:
- `output/review_planets/sweeps/level6_trade_polity_validation`

Status:
- cache fill completed for `20/20` seeds
- warmed confirmation completed for `20/20` seeds
- warmed pass hit terrain, climate, derived, trade goods, civilization, maritime, and economy caches
- detailed analysis recorded in `analysis.md`

Main finding:
- raw goods form broad, plausible import deficits
- `soap`, `ships`, `jewelry`, `perfumes`, and `iron_goods` have meaningful importer structure
- several manufactured goods are over-supplied at polity level and have weak or absent true importers

Important aggregate signals:
- raw category: avg deficit `59.94`, avg importers `224.65`, deficit/demand `0.38`
- processed category: avg surplus `36.49`, avg deficit `3.86`, deficit/demand `0.07`
- `cloth`: avg exporters `18.50`, avg importers `0.00`
- `woolens`: avg exporters `18.50`, avg importers `0.00`
- `fine_clothing`: avg exporters `18.45`, avg importers `0.05`
- `ceramics`: avg exporters `18.45`, avg importers `0.05`
- `weapons_armor`: avg exporters `18.30`, avg importers `0.20`

Decision:
- do not tune transport or route capacity for this issue
- next trade work should target polity-level demand, local supply relief, and per-good demand differentiation for manufactured goods
- preserve current healthy behavior for `soap`, `ships`, `jewelry`, `perfumes`, and `iron_goods`

### Latest Polity Demand Tuning

The latest accepted polity-demand tuning sweep is:
- `output/review_planets/sweeps/level6_polity_demand_tuning_trial1`

What changed:
- added per-good demand tuning fields to the trade-good schema:
  - `goodDemandScale`
  - `localSupplyReliefByGood`
  - `marketGoodDemandScale`
- applied JSON tuning for `fine_clothing`, `weapons_armor`, `ceramics`, `cloth`, `woolens`, and `preserved_food`

Status:
- `20/20` seeds completed with `ok`
- seed stderr total was `0`
- summarized with trade focus, good path, polity goods, processed path, and chain summaries
- detailed analysis recorded in `analysis.md`
- decision: accept trial 1; do not increase the same knobs again until a wider or higher-resolution pass

Key comparison versus `level6_trade_polity_validation`:
- average total trade score moved from `47.90` to `52.85`
- average total trade volume moved from `67.82` to `74.13`
- `fine_clothing` traded seeds moved from `1/20` to `20/20`
- `cloth` traded seeds moved from `15/20` to `20/20`
- `woolens` traded seeds moved from `17/20` to `20/20`
- `preserved_food` traded seeds moved from `15/20` to `20/20`
- `ceramics` traded seeds moved from `1/20` to `20/20`
- `weapons_armor` traded seeds moved from `3/20` to `8/20`
- `paper`, `soap`, `ships`, `jewelry`, and `iron_goods` stayed stable

Remaining risks:
- textile demand now has real unmet pressure; watch `cloth->fine_clothing` demand gaps on wider/higher-resolution sweeps
- `leather` remains broadly local and over-supplied, but this may be acceptable as a common processed staple
- `weapons_armor` importer structure remains sparse, which may be acceptable for strategic goods

### Level-7 Trade / Polity Validation

The higher-resolution validation sweep completed:
- script: `scripts/run_level7_trade_polity_validation.sh`
- output directory: `output/review_planets/sweeps/level7_trade_polity_validation`
- seed slice: `4,42,84,91,123,255,314,777`
- analysis: `output/review_planets/sweeps/level7_trade_polity_validation/analysis.md`

Status:
- `8/8` cold-cache seeds completed with `ok`
- `8/8` warmed-confirm seeds completed with `ok`
- warmed confirmation hit terrain, climate, derived, trade goods, civilization, maritime, and economy caches
- warmed elapsed range was about `21s` to `27s` per seed
- seed stderr total was `0`

Main finding:
- cache and route behavior hold up at level 7
- raw trade remains plausible
- `paper`, `soap`, and `ceramics` do not show dominance regressions
- total trade and manufactured importer structure are materially weaker than the same seed slice at level 6
- `fine_clothing`, `weapons_armor`, and textile-chain conversion need follow-up before level-7 trade is considered fully validated

### Resolution-Scale Fix Trial

A follow-up level-7 validation completed after fixing clear resolution-scaling leaks:
- script: `scripts/run_level7_resolution_scale_validation.sh`
- output directory: `output/review_planets/sweeps/level7_resolution_scale_validation`
- seed slice: `4,42,84,91,123,255,314,777`

Code changes validated:
- graph-hop raw/resource catchments now expand with mesh resolution
- polity demand uses physical-equivalent territory linear size
- land and river inter-civilization flow use physical-equivalent territory area
- large-polity environment tags and territorial crowding use physical-equivalent territory area
- civilization cache bumped to `civilization-v3`
- economy cache bumped to `economy-v12`

Follow-up finding:
- textile conversion still regressed at level 7 because expanded catchments were weighted by raw graph distance, diluting localized raw inputs such as `dye_plants`, `fiber`, and `wool`
- catchment weighting now uses physical graph distance, but a rejected peak-blend experiment showed that patching catchment values can recover textiles while over-amplifying other chains
- source-field summaries show the deeper resolution leak is upstream of node/market trade: level-7 raw potential fields are physically sparser before catchment aggregation, especially `dye_plants`, `coal`, `timber`, and `fish`
- derived-field summaries localize that leak further into vegetation/biome/soil/hydrology-derived fields: `TreeCover`, `WetlandCover`, `ForestAffinity`, `WetlandAffinity`, `Alluvial`, coastal fisheries, and water reliability shrink materially at level 7, while crop/pasture mostly hold
- current fix in progress: hydrology classes/channel support, coastal exposure, and precipitation transport/fetch lengths now scale by physical-equivalent mesh steps rather than fixed graph hops
- terrain cache is now `terrain-v3`; climate cache is now `climate-v4`; derived cache is now `derived-v8`; downstream review caches will refill because physical hydrology/derived inputs changed
- smoke validation before the precipitation cache bump showed hydrology/coastal goods improved (`fiber`/`wool` held; `fish` and coastal fisheries moved closer), but forest/tree/timber still need full climate-cache validation because precipitation is now being recomputed under the new physical transport scaling
- next economy cache version remains `economy-v12`
- durable source-field summary script: `scripts/summarize_trade_good_potentials.py`
- durable derived-field summary script: `scripts/summarize_derived_fields.py`

Latest focused diagnostic:
- single-seed diagnostic dirs: `output/review_planets/sweeps/resolution_fix_diag_seed42_l6` and `output/review_planets/sweeps/resolution_fix_diag_seed42_l7`
- follow-up fixes added physical support fields for riparian channel proximity and depositional hydrology classes, plus physical spread for coastal current/upwelling support
- derived cache is now `derived-v6`
- seed `42` result after those fixes: `coal` mean ratio improved to about `1.01`; `dye_plants`, `fiber`, and `wool` are near parity; `timber` is about `0.89`; `fish` remains low at about `0.77`; `RiparianAffinity`, `Alluvial`, and `SurfaceReliability` remain the main suspects
- seed `42` climate diagnostic: annual precipitation is near parity (`0.97`), but aridity ratio remains low (`0.65`) because level 7 is substantially warmer on the land mask and therefore has a higher aridity threshold; do not patch this with trade/vegetation tuning without proving it is algorithmic rather than terrain/climate variance

Latest hydrology threshold diagnostic:
- single-seed diagnostic dirs: `output/review_planets/sweeps/resolution_fix_diag_seed42_threshold_l6` and `output/review_planets/sweeps/resolution_fix_diag_seed42_threshold_l7`
- terrain channel normalization now uses an upper-tail accumulation percentile rather than absolute land-cell count; this fixes the direct source leak where raw channel strength was about `0.45x` at level 7 while runoff was about `1.03x`
- seed `42` post-fix trade-source ratios: `fish` about `0.95`, `alluvial`/soil support about `0.92`, `surfaceReliability` about `0.88`, `timber` about `0.90`, `coal` about `1.15`, `dye_plants` about `1.11`, `fiber` about `1.06`, and `wool` about `1.06`
- remaining focused caveat: `RiparianAffinity` is now high at level 7 (`~1.59x`) but at very low absolute mean; broad validation should check whether that averages out or requires a narrower riparian support footprint

Riparian follow-up:
- single-seed diagnostic dirs: `output/review_planets/sweeps/resolution_fix_diag_seed42_riparian_v8_l6` and `output/review_planets/sweeps/resolution_fix_diag_seed42_riparian_v8_l7`
- decomposition showed the inflated ratio came from spreading riparian channel support after channel normalization was fixed; raw riparian source support was near parity
- `RiparianAffinity` now uses stable raw channel support plus bounded floodplain/class support, and vegetation diagnostics expose `RiparianHydrology`, `RiparianAridity`, `RiparianHeat`, and `RiparianPrecip`
- seed `42` post-fix riparian ratio is about `1.10` with usable absolute mean (`0.0099` at level 6, `0.0109` at level 7); riparian hydrology itself is about `0.92`, with the remaining lift explained by the warmer level-7 climate

Terrain plate resolution follow-up:
- single-seed diagnostic dirs: `output/review_planets/sweeps/resolution_fix_diag_seed42_terrain_v3_l6` and `output/review_planets/sweeps/resolution_fix_diag_seed42_terrain_v3_l7`
- terrain plate seeding now chooses continuous physical centers and assigns regions by deterministic weighted spherical Voronoi distance, instead of selecting random mesh indices and growing plates through resolution-dependent graph BFS
- cache version is now `terrain-v3`
- seed `42` terrain/geography comparison is now stable: land share `28.86%` vs `29.07%`, northern land share `49.35%` vs `49.93%`, tropical land share `42.10%` vs `41.98%`, polar land share `14.29%` vs `14.84%`, and mean absolute latitude differs by only `0.19` degrees
- seed `42` climate comparison no longer shows the earlier warm-land-mask failure: annual land temperature is `10.69C` at level 6 and `10.22C` at level 7, while seasonal/biome temperature is `3.21C` vs `3.58C`
- follow-up precipitation diagnosis showed the remaining gap came from applying per-hop marine/ocean retention and reservoir-transfer fractions more times at higher mesh resolution over the same physical path
- precipitation transport now scales those per-hop factors by physical-equivalent cell length; climate cache is now `climate-v9`
- single-seed diagnostic dirs: `output/review_planets/sweeps/resolution_fix_diag_seed42_precip_v9_l6` and `output/review_planets/sweeps/resolution_fix_diag_seed42_precip_v9_l7`
- seed `42` precipitation comparison improved materially: annual precipitation ratio moved from about `0.78` to `0.951`, precipitation range ratio to `0.949`, and derived aridity ratio to `1.004`
- coastal onshore support now uses a physical coastal band rather than only immediate ocean neighbors: `CoastalOnshore` is about `1.12x`, `EffectiveOnshore` about `1.08x`, and `NeighborOceanFraction` about `0.91x` at level 7 versus level 6
- orographic local and footprint rise now use mesh-scale relief compensation: `OrographicLocalRise` is about `1.01x`, `OrographicFootprint` about `0.95x`, barrier persistence about `1.00x`, and wind factor about `0.99x`
- remaining focused caveat: raw upwind ocean path diagnostics still shrink at level 7 (`OceanFetch` and `FootprintOceanSupport` remain about `0.59-0.60x` as land-wide means), but effective onshore access, annual precipitation, aridity, and most climate-derived trade goods are now stable enough for a broader warmed-cache check
- runtime note: physically scaling upwind-footprint decay and smoothing/diffusion iteration counts made level-7 climate diagnostics too slow, so those experiments were reverted; keep future fixes runtime-aware or precompute/cache broad footprint fields
- coastal port node catchments now scale hop radius and decay by physical-equivalent mesh length, matching the existing trade-good catchment scaling; maritime cache is now `maritime-v2`
- local feeder discovery, maritime stopover spacing, and open-water coastal allowance now use physical-equivalent mesh steps; civilization cache is now `civilization-v4`
- soil local-relief support now normalizes immediate-neighbor elevation deltas by physical-equivalent mesh length; derived cache is now `derived-v9`
- terrain erosion now scales diffusion iterations, landmass/coastal radius checks, tectonic support distance, fluvial contributing area, fluvial slope thresholds, lake/channel widening, and delta/floodplain spread kernels by physical-equivalent mesh scale; terrain cache is now `terrain-v4`
- multimodal trade now caps duplicate per-route pair-good matches to the available source/sink polity surplus so alternate routes/modes cannot repeatedly count the same surplus/need; economy cache is now `economy-v13`
- seed `42` terrain-v4/economy-v13 warmed rerun: L6 trade `20.85/29.42`, L7 trade `41.42/55.76`; duplicate route accounting was a real overcount, but remaining L7 lift now appears in settlement/market and maritime-route cardinality rather than raw derived potentials

Progress checks:
- `systemctl --user status worldgen-l7-resolution-scale-validation.service --no-pager`
- `cat output/review_planets/sweeps/level7_resolution_scale_validation/progress.json`
- `tail -80 output/review_planets/sweeps/level7_resolution_scale_validation/run.log`
- `cat output/review_planets/sweeps/level7_resolution_scale_validation/run.err`

Older `ceramics` / `ships` validation:
- `output/review_planets/sweeps/level6_ceramics_ships_validation`

What that pass was trying to do:
- reduce `ceramics` as the most universal workshop winner
- tighten `ships` slightly without damaging strategic trade

Key comparison versus `level6_processed_capability_validation`:
- `ceramics` first-place cap winners dropped from `5/7` seeds to `1/7`
- `ceramics` average cap-winner rank moved from `1.43` to `2.29`
- `ceramics` average winner count dropped from `34.86` to `30.29`
- `ships` stayed present as strategic trade in `7/7` seeds
- `ships` average strategic score stayed stable at `5.64`

Latest cap-winner aggregate:
- `ceramics`: present `7/7`, first `1/7`, avg rank `2.29`, avg count `30.29`
- `paper`: present `6/7`, first `2/7`, avg rank `2.83`, avg count `27.33`
- `ships`: present `6/7`, first `0/7`, avg rank `3.17`, avg count `28.67`
- `soap`: present `5/7`, first `1/7`, avg rank `2.60`, avg count `28.80`
- `weapons_armor`: present `4/7`, first `3/7`, avg rank `1.25`, avg count `35.00`

## Current Working Scripts

Repeatable workflow scripts now in use:
- `scripts/refill_review_cache.py`
- `scripts/run_level6_trade_broader_validation.sh`
- `scripts/run_level6_trade_polity_validation.sh`
- `scripts/run_level7_trade_polity_validation.sh`
- `scripts/run_level7_resolution_scale_validation.sh`
- `scripts/build_review_cache_manifest.py`
- `scripts/summarize_trade_sweep.py`
- `scripts/summarize_trade_chains.py`
- `scripts/summarize_polity_goods.py`
- `scripts/summarize_processed_path.py`
- `scripts/summarize_good_paths.py`

These should be preferred over ad hoc shell commands for:
- cache refills
- level sweeps
- trade diagnostics
- post-run summaries

## Current Risks / Open Questions

### Trade Model

Latest seed-314 terrain-v8 rerun:
- Terrain/hotspot materialization now matches closely between L6 and L7: both have `12` hotspot chains and `50` hotspot islands, with nearly identical land latitude distribution.
- Full rerun still exposed a downstream trade divergence: before the maritime fix, L6 was `1.78/2.46` and L7 was `6.03/8.34`.
- Root cause found in coastal endpoint routing: refined L7 settlement nodes could share the same physical harbor terminal cell, creating a zero-cost coastal route between two nodes at the same terminal. That single same-cell route inflated coastal multimodal capacity and contributed about `2.4` score by itself.
- `maritime-v20` now de-duplicates coastal candidate ports by terminal cell and skips same-cell endpoint edges; `economy-v28` invalidates stale multimodal caches.
- Seed 314 after the fix: L6 remains `1.78/2.46`; L7 drops to `3.93/5.53`; zero-cost maritime corridors are gone.
- `maritime-v21` / `economy-v29` add focused route diagnostics: physical path degrees, cost per degree, endpoint port scores, and top multimodal route-exchange capacity lines.
- Seed 314 route diagnostic output: `output/review_planets/sweeps/seed314_route_capacity_diag`.
- The diagnostics argue against a remaining basic path-cost scaling bug: coastal mean cost-per-degree is close (`0.37` at L6 vs `0.39` at L7).
- The remaining seed-314 gap is from route/endpoint topology and market structure: L7 accepted coastal exchanges have much higher volume capacity (`5.11` vs `2.27`) because active route costs are shorter (`9.24` vs `28.27`), endpoint scores are higher (`0.79` vs `0.65`), and nearby same-civilization/different-polity routes carry large processed/raw exchanges.
- Next focused target should be polity/proto endpoint topology and port-flow normalization, especially whether composite `NodePortScore` should be treated as bounded port quality inside `coastalTradeFlow` instead of allowing centrality/rank bonuses to raise transport capacity.
- `maritime-v22` fixes the port-flow normalization issue: coastal route flow now uses bounded physical terminal quality from the terminal cell (`PortSuitability` / harbor-estuary-transfer-stopover feature), while composite `NodePortScore` remains for candidate selection and ranking. Seed 314 moves from L6 `1.78/2.46`, L7 `3.93/5.53` to L6 `1.62/2.22`, L7 `3.52/4.94`.
- `economy-v31` adds route civilization IDs to multimodal exchange diagnostics plus `routeCivDiag`.
- Seed 314 route-civ diagnostic output: `output/review_planets/sweeps/seed314_route_civ_diag_v2`.
- Current root signal: L6 has no same-civilization multimodal trade (`same=0/s0.00/v0.00`) and `1.59` inter-civilization score; L7 has `1.90` same-civilization score and `1.53` inter-civilization score. The remaining excess is therefore not a coastal cost or port-quality scaling bug; it is a proto/polity topology issue where refined resolution creates short same-civilization inter-polity exchanges, especially secondary/suzerainty trade.
- `economy-v32` separates internal same-civilization exchanges from external trade accounting. Internal exchanges remain in `Exchanges` and route diagnostics for audit, but `Pairs`, headline `multimodalTrade` score/volume, trade-good path summaries, and polity attitude trade bonuses now use external exchanges only.
- Seed 314 internal/external output: `output/review_planets/sweeps/seed314_internal_external_trade`.
- Seed 314 after the accounting split: L6 external trade is `1.62/2.22`; L7 external trade is `1.62/2.54`, with L7 separately reporting `internalExchanges=5 internalScore=1.90 internalVolume=2.41`. This closes the apparent score divergence without suppressing real internal circulation diagnostics.

Current seed-42 resolution check after the latest fixes:
- `economy-v14` adds global source/sink polity-good conservation after duplicate route capping.
- `maritime-v4` samples coastal port geometry over a resolution-scaled physical neighborhood and adds JSON-authored `stopoverSpacingHops` for ocean stopovers.
- `civilization-v5` samples settlement local peaks over a resolution-scaled physical neighborhood.
- `civilization-v6` requires proto-civilization seeds to have a real regional anchor cluster instead of letting isolated refined-resolution town peaks become independent civilizations.
- `economy-v15` caps endpoint-market imports by total destination polity/good endpoint deficit so local endpoint need cannot scale unbounded with route/source count.
- `maritime-v5` filters maritime trade endpoints to civilized settlement nodes; outpost harbors can support geography but are not polity trade endpoints.
- `maritime-v6` adds durable stopover and endpoint edge diagnostics to the review output.
- `maritime-v7` uses physical angular spacing for stopover selection when mesh sites are available, with hop spacing retained as fallback for synthetic callers.
- `maritime-v8` applies civilization-level caps to internal coastal/ocean maritime corridors, preventing large refined civilizations with more eligible ports from producing unbounded same-civ trunk routes.
- `maritime-v10` reuses maritime endpoint graph adjacency/coast-distance data and Dijkstra work buffers during graph construction, reducing cold high-resolution maritime recompute time without changing seed-42 trade totals.
- Targeted tests pass with `env GOCACHE=/tmp/go-build-cache go test ./landgen/terrain ./climgen ./cmd/review_planets`.

Latest single-seed results:
- Before settlement peak scaling, `maritime-v4/economy-v14`: L6 `12.87/17.75`, L7 `27.89/37.56`.
- After settlement peak scaling, `civilization-v5`: L6 `10.48/14.84`, L7 `28.85/39.80`.
- Settlement node counts improved from `77 vs 97` to `68 vs 73`, but proto/polity fragmentation worsened: L6 `9` proto civs / `14` polities, L7 `12` proto civs / `17` polities.
- After proto eligibility, endpoint caps, and civilized maritime endpoint filtering: L6 `11.05/15.60`, L7 `14.49/20.53`.
- Proto/polity counts now match on seed 42: L6 `9` proto civs / `14` polities, L7 `9` proto civs / `14` polities.
- Maritime candidate endpoint counts are much closer: coastal `36` vs `40`, ocean `16` vs `23`; residual L7 trade excess is about `1.31x`, down from the earlier `~2x-3x` range.
- After physical stopover spacing and internal route caps: L6 `11.19/15.81`, L7 `13.59/19.45`, or about `1.21x` by score and `1.23x` by volume.
- Cold L7 maritime runtime improved from roughly `582s` to `486s` with resolution fixes, then to `281s` after endpoint graph runtime optimization on seed 42; L6 is roughly `37-41s` on the same warmed physical/civilization caches.
- Additional stopover diagnostics show stopover spacing itself is resolution-independent on seed 42: coastal min spacing is `4.328°` at both L6 and L7; ocean min spacing is `8.657°` at both levels.
- The remaining maritime stopover difference is driven by terrain/topology candidate supply, not route spacing: L7 has many more tiny land components (`67` land components / `59` tiny scaled components) than L6 (`24` / `17`), and many more island stopover candidates (`264` vs `49`).
- Hotspot diagnostics are implicated in the same direction: total hotspot islands are `117` at L7 vs `78` at L6, with oceanic hotspot islands `100` vs `43`.
- A direct hotspot spacing diagnostic shows the likely resolution-dependence mechanism: L6 oceanic hotspot spacing is only `1.39x` the mean mesh neighbor spacing, so multiple physical island events collapse to the same coarse cell; L7 spacing is `3.21x` the mesh spacing, so the same angular process can materialize as many more distinct cells/components.
- A terrain-source trial using physical hotspot spacing and minimum emergent island radius reduced tiny components in terrain-only checks, but was not retained because the full L6 seed-42 pipeline regressed to `19.47/26.73` trade by changing climate/civilization/economy conditions. Keep future fixes narrower and validate full L6 before running L7.
- `maritime-v12` adds stopover component-area diagnostics using the same physical-equivalent buckets as `landComponentDiag`. On seed 42, L6 ocean selected stopovers are `7` tiny-equivalent / `1` small-equivalent / `15` large-equivalent, while L7 is `19` tiny-equivalent / `3` small-equivalent / `21` large-equivalent. Tiny/small island materialization is a real contributor to the remaining ocean lift, but not the only contributor because selected large-component stopovers also increase.
- `maritime-v13` moves base maritime stopover selection weights and thresholds into `config/maritime_ports_earthlike.json` and applies a physical component-area score taper. Stronger taper validation on seed 42 reduced L7 ocean tiny/small selected stopovers from `22/43` to `17/41` while keeping L6 stable at `11.21/15.85`, but L7 multimodal trade remained `13.59/19.45`. Tiny-island support is now constrained, and the remaining trade gap appears to come from endpoint/market flow structure rather than tiny stopover count alone.
- `economy-v17` adds endpoint sink/source cap diagnostics and makes endpoint market need residual after aggregate polity surplus via JSON-authored `endpointSurplusReliefByCategory`. Seed-42 tuned relief (`processed 0.60`, `finished 0.35`) gives L6 `10.12/14.37` and L7 `11.88/17.22`, improving the score ratio to about `1.17x` while removing much of the paper/processed endpoint-need leak.
- Selected stopover counts remain higher at L7, but the stopover-area taper shows that reducing tiny-island support alone does not reduce the current trade score gap. Endpoint surplus relief reduced the largest processed-good leak; the next resolution-independence target should be raw/coastal route composition and market manufacturing balance before further terrain or demand tuning.
- Seed-84 diagnostics showed a separate resolution-dependent cardinality amplifier: L7 had `10` proto civs / `15` polities versus L6 `6` / `8`, and ocean candidates jumped from `6` to `24`, including `10` secondary ocean ports and up to `8` ports in one civilization.
- Ocean trade now has `maxCandidatePortsPerCivilization` in `config/ocean_trade_earthlike.json`; the current value is `1`, treating ocean routes as trunk gateways and leaving local secondary movement to coastal routes. On seed 84 this reduced L7 ocean candidates from `24` to `9`, removed secondary ocean candidates, and reduced L7 multimodal trade from `18.01/25.59` to about `13.33/18.94` before the settlement-waystation fix.
- Settlement waystation bridging no longer uses raw path cell counts (`len(path) >= 10`) or a fixed cell search window; it uses physical path length and a resolution-scaled search radius. This fixes a direct civilization-chain resolution leak, but seed 84 still shows residual polity-count divergence after the fix: L6 `7` proto civs / `9` polities and trade `6.77/9.25`; L7 `10` proto civs / `15` polities and trade `13.10/18.69`.
- Proto eligibility now uses weighted regional-anchor strength, so hamlets/village satellites no longer make a lone town qualify as a proto-civilization. This keeps the broad-village fallback but removes the high-resolution one-town-plus-satellites artifact; civilization cache is now `civilization-v10`.
- Secondary polity spawning now compares proto territory with `SecondaryLargeProtoCells` using physical-equivalent area, not raw cell count. On seed 84 this reduced secondary polities from L6 `2` / L7 `5` to L6 `1` / L7 `3`.
- Settlement-network summaries now include node-formation diagnostics: classified cells, local-peak candidates, spacing-kept nodes, waystations, prune counts, and kind breakdowns. Civilization cache is now `civilization-v11`.
- Current seed-84 result after weighted-anchor/area fixes: L6 `6` proto civs / `7` polities / trade `5.76/7.90`; L7 `8` proto civs / `11` polities / trade `8.13/11.33`. The score ratio was about `1.41x`, down from `2.8x` at the start of the seed-84 investigation. Node formation itself was close (`53` vs `58` final nodes), but L7 kept more eligible region strength.
- Seed 42 moved in the opposite direction under the same rules: L6 `9` proto civs / `12` polities / trade `9.79/13.61`; L7 `7` proto civs / `9` polities / trade `5.65/7.84`. Node formation was also close there (`68` vs `73` final nodes, `24` vs `27` regional anchors), so the remaining instability was in settlement-region grouping and proto eligibility, not raw settlement node generation.
- `civilization-v12` makes settlement-region formation use physical anchor reach in addition to the sparse transport links. This removes a target-cap/ranking artifact where high-resolution node insertion could split or merge proto regions even when physical settlement reach was similar. Region summaries now print `regionFormation` transport-link and physical-cluster-link counts.
- `civilization-v12` also filters proto-civilization and polity minimum territories by physical-equivalent area, not raw cell count. This closes a latent high-resolution leak where tiny refined fragments could survive because `MinTerritoryCells` was counted directly.
- Current warmed check after `civilization-v12`:
- Seed 84: L6 `6` proto civs / `7` polities / trade `5.52/7.38`; L7 `7` proto civs / `10` polities / trade `6.38/8.96`.
- Seed 42: L6 `9` proto civs / `12` polities / trade `9.89/13.87`; L7 `8` proto civs / `10` polities / trade `8.43/11.66`.
- Current interpretation: the most direct remaining trade-chain resolution leak was settlement-region formation from sparse target-capped links. The two-seed check is materially better, but not broad enough to declare the whole chain resolution-independent; next validation should be a warmed multi-seed L6/L7 slice using `civilization-v12`/`economy-v18`.
- The matched 8-seed L6/L7 validation exposed a separate route-endpoint ownership bug. Seed 255 had L6 proto/polities and inter maritime corridors but `multimodalTrade: exchanges=0`; diagnostics showed `tradeDiag: routes=0/0`, meaning multimodal rejected all route endpoints before goods matching. The cause was `PolityByNode` being populated only for capital nodes, while `tradeNodePolity` needed ownership for all route endpoint nodes. If an endpoint cell was outside the claimed territory raster, the route became unowned even when the node belonged to a proto/trade civilization.
- `civilization-v13` now assigns polity ownership to every settlement node: claimed cell ownership wins, otherwise the node falls back to the primary polity for its proto-civilization via `TradeNetworkDiagnostics.CivilizationByNode`; capital nodes are still forced to their sphere. This is a structural fix, not a score patch.
- Seed 255 after `civilization-v13`: L6 multimodal trade recovers from `0/0` to `2.06/3.23` with `tradeDiag routes=5/8`; L7 is `8.32/11.97` with `routes=30/36`. The binary failure is fixed, but this seed still has a residual inter-route/hub topology gap.
- Seed 91 after `civilization-v13`: L6 `4.29/6.00`, L7 `4.26/6.25`; the previous `7x` score outlier was primarily the node-ownership bug.
- Current remaining suspect after `civilization-v13`: settlement-node rank / region / route topology. Seed 255 L6 has no major hubs, `26` regions, `4` proto civs, `6` polities, and only `3` coastal + `1` ocean inter routes; L7 has `2` major hubs, `13` regions, `5` proto civs, `8` polities, and `6` coastal + `6` ocean inter routes. This points upstream of multimodal goods, likely settlement rank/region connectivity or maritime endpoint route formation.
- `derived-v11` fixes the seed-255 support-field leak at its source. `ResolutionAdjustedHydrologyBiomeInputs` no longer overwrites normalized `ChannelStrength` with a physically spread support field; raw channel coverage was already stable (`~12%` channel-like area at both L6 and L7), but the spread field inflated adjusted channel support to `31%` at L6 and `38%` at L7. Downstream settlement/access/water/population layers were reading that support as centerline strength.
- Population catchment smoothing remains available through authored `population_support_earthlike.json` fields `catchmentHops` and `catchmentBlend`, but `catchmentBlend` is currently `0.0`; the earlier `0.70` trial is treated as diagnostic only and is not retained.
- Seed 255 after the hydrology support fix and with catchment smoothing disabled: L6 full trade is `7.78/10.91`, L7 full trade is `7.54/10.78`. The original L6 under-trade outlier is gone without seed-specific tuning. Node counts now match exactly (`82` final settlement nodes at both levels), while residual differences remain in regional anchors/proto/polity counts (`L6 4 proto / 6 polities`, `L7 5 proto / 9 polities`).
- `civilization-v27` addresses the next confirmed resolution leak in proto-civilization eligibility. Diagnostics showed seed 84 L7 proto-eligible regions were dominated by weak physical regional-anchor footprints (`eligibleRegionalLow=59.1%`) while L6 eligible regions had no low-support regional anchors (`0.0%`). Proto anchor-kind eligibility now requires at least one physically supported regional anchor; this leaves settlement ranks/links unchanged and filters weak high-resolution proto seeds as `regional-support` outposts.
- Four-seed full validation after the proto-support fix: L6 scores are seed42 `6.25`, seed84 `1.58`, seed91 `5.00`, seed255 `7.78`; L7 scores are seed42 `8.08`, seed84 `1.73`, seed91 `5.20`, seed255 `7.54`. The seed84 score ratio improved from `2.81x` to `1.09x`, and the four-seed score ratios are now approximately `1.29x`, `1.09x`, `1.04x`, `0.97x`.
- Civilization cache keys now include the default settlement-network, proto-civilization, trade-network, river-trade, and polity-sphere settings in the settings digest. This prevents future network/proto behavior changes from silently reusing stale civilization caches. `linkFormation near10` diagnostics are also fixed to search beyond the hard travel cutoff, so near-threshold misses are real diagnostics rather than duplicates of reachable counts.
- `civilization-v30` addresses two additional confirmed resolution leaks rather than retuning trade values. First, polity profile `large-polity` context now uses physical-equivalent territory area, matching the environment tag logic. Second, local feeder trade nodes are physically de-duplicated so a refined mesh cannot place multiple feeder depots inside one coarse-cell neighborhood. Feeder centrality is also split from trunk centrality; local feeder flow can support markets, but major hubs, secondary polity spawning, and polity claim reach now use trunk/political centrality instead of feeder-inflated total centrality.
- Narrow v30 validation: seed42 is now close, with L6 `7` proto / `7` polities / trade `4.53/6.62` and L7 `5` proto / `6` polities / trade `4.31/6.08` (`0.95x` score ratio). Seed84 no longer has feeder-driven secondary-hub inflation, but still differs structurally: L6 `3` proto / `3` polities / trade `0.80/1.16`; L7 `5` proto / `6` polities / trade `1.40/2.09` (`1.75x`). The remaining seed84 gap is upstream proto/polity topology, not a direct local-feeder or multimodal capacity issue.
- `civilization-v34` keeps two structural follow-ups from the seed84 investigation. Proto claim radius now uses physical-support-weighted anchor strength so weak refined anchors do not inflate territory reach, and settlement-region physical fallback only connects transport-isolated nodes instead of merging already connected transport components. Region diagnostics now print raw/physical anchor strength and supported regional-anchor counts.
- Narrow v34 validation: seed42 remains stable with L6 `6` proto / `6` polities / trade `4.14/5.84` and L7 `5` proto / `6` polities / trade `4.31/6.08` (`1.04x` score ratio). Seed84 still diverges: L6 `3` proto / `3` polities / trade `0.79/1.15`; L7 `5` proto / `6` polities / trade `1.54/2.26` (`1.95x`). The current seed84 root is settlement transport-link topology: L6 has `31` links, `27` regions, and `19.6%` isolated nodes; L7 has `43` links, `16` regions, and `8.6%` isolated nodes.
- `civilization-v47` was a diagnostic-only cache bump for settlement-link and spacing topology. It kept the v38 reachable/near/selected target-kind totals and selected/created endpoint-kind pair matrices, and added spacing rejection/blocker/support-area diagnostics so retained high-rank anchors could be compared against rejected support. Seed84 under v38 showed the extra L7 connectivity is concentrated in high-rank links: L6 created `1` regional-regional and `16` district-regional links, while L7 created `9` regional-regional and `23` district-regional links. L7 regional sources also had `48` reachable targets versus `19` at L6.
- Link support diagnostics show seed84 L7 has many more high-rank links touching physically weak regional anchors: L6 has `3/18` weak high-endpoint links and `0` below `0.25` physical support, while L7 has `24/35` weak high-endpoint links and `5` below `0.25`. Seed42 L7 also has `5` below `0.25`, but its trade remains stable, so weak support is a contributing diagnostic rather than a standalone rejection rule.
- High-rank source diagnostics show the raw population fields are not simply overproducing high-rank equivalent area at L7. For seed84, high-rank peak equivalent area is L6 `53.5` vs L7 `51.5`, but spacing keeps L6 `19` regional anchors vs L7 `25`. The resolution leak is therefore in how dense refined peak candidates pass greedy spacing/selection into final anchors, not just in the raw classified population support.
- Spacing diagnostics under `civilization-v47` confirmed the leak is rank cardinality rather than real support. Seed84 L6 keeps `19` regional anchors with `15.8` kept regional support area, while L7 keeps `25` regional anchors with only `14.3` kept regional support area; seed42 is similar but less harmful (`23` anchors / `21.3` area at L6, `24` anchors / `16.6` area at L7). Downstream settlement links and regions still treat those smaller L7 anchors as full regional anchors, producing more high-rank links and proto regions. The next fix should make settlement rank influence, link budget/reach, and possibly trade/polity centrality depend on physical support footprint, not raw anchor count.
- `civilization-v50` is the retained topology fix for that leak. Settlement nodes now carry physical support area, and high-rank settlement link reach/target budget, link target priority, region-center rank scoring, and proto/polity claim influence use a bounded physical-support weight. Anchors with at least `0.75` equivalent support area remain full-strength; only sub-baseline high-rank footprints are downweighted. The broad v48 attempt that also weighted maritime, river, trade, port, and market scoring is rejected because it collapsed seed42 L6 trade to `1.69/2.37`. The narrower v49 topology attempt was directionally right but still undercut seed42 L6 (`2.34/3.17`), so v50 keeps the topology fix while preserving coarse valid anchors.
- Narrow v50 validation on seeds `42,84,91,255`: seed42 L6/L7 trade is `4.14/5.83` vs `4.34/6.10`; seed84 is `0.79/1.15` vs `1.08/1.63`; seed91 is `2.09/2.83` vs `2.82/3.79`; seed255 is `8.13/10.70` vs `7.34/10.39`. Proto/polity counts are now close enough for focused validation: seed42 `6/6` vs `4/4`, seed84 `3/3` vs `4/4`, seed91 `5/5` vs `5/5`, seed255 `6/6` vs `6/7`.
- Rejected seed84 trials: a mesh-step link tolerance improved seed84 but regressed seed42 badly; physical-strength proto eligibility over-suppressed legitimate L6 regions; refined-resolution town-rank downgrading reduced regional anchors but increased L7 connectivity and trade; capping regional-anchor selected targets at one reduced links but regressed seed42 L7 trade to `7.70`; filtering weak high-rank region bridges at `0.5` over-suppressed both seeds; filtering only tiny `<0.25` high-rank region bridges preserved seed42 but pushed seed84 L7 down to `0.43/0.63` trade while still leaving L7 at `4` proto / `5` polities versus L6 `3` / `3`; support-weighted effective link rank left seed84 at `5` proto / `6` polities and raised L7 trade to `1.71/2.57`; support-inflated spacing changed L6 too much and regressed seed42 L7 trade to `8.22/11.03`; support-first candidate ordering under-traded seed84 L7 (`0.38/0.56`) and shifted L6; weak-support-only candidate ordering over-amplified L7 (`seed42 7.34/10.73`, seed84 `3.87/5.67`); coverage-oriented high-rank candidate ordering partially reduced seed84 L7 proto count but destabilized L6 badly (`seed42 11.06/15.18`) and shifted seed84 L6 topology. Do not revive these as score patches.
- Current next step: treat v50 as the retained focused fix and run a broader warmed-cache L6/L7 validation before changing additional trade or maritime scoring. Remaining seed84 and seed91 L7 lift is moderate rather than catastrophic, and should be checked across more seeds before further algorithm changes.

Main resolved issue:
- processed goods were being over-smoothed or locally absorbed before inter-polity trade; this is now substantially fixed
- broader `ceramics` workshop dominance is now substantially fixed on the 20-seed slice
- `paper` is still the leading processed good, but its score and share are reduced enough to avoid another immediate tuning pass

Other likely future trade targets, in order:
1. let the active resolution-scale validation finish and compare it to the prior level-7 sweep
2. investigate any remaining level-7 settlement favorability / polity-count gap
3. inspect textile and finished-good market conversion before increasing demand knobs again
4. add diagnostics that decompose market `made` into input availability, capability, demand, and cap limits
5. revisit `leather` only if it should become a meaningful inter-polity good rather than a local staple
6. revisit `paper` only if demand-side fixes leave it dominating too broadly
7. investigate strategic-category diversity if `ships` being the only consistent strategic traded good becomes a design problem
8. avoid reopening multimodal demand-side hacks unless diagnostics show a real sink-formation problem again

### Validation / Tooling

Cache tooling is in good shape after the cache-boundary split:
- derived review caches terrain/climate-derived resources, settlements, and population without trade-good settings
- trade-good review caches authored-trade-dependent endowments separately
- civilization review caches networks, land/river routes, polities, and profiles without trade-good settings
- economy review owns trade-good dependent node goods, polity goods, node markets, and multimodal flows

Remaining validation need:
- finish the active level-7 resolution-scale validation before further broad sweeps

## Recommended Next Steps

Recommended immediate next step:
1. do not tune transport capacity first
2. wait for the active resolution-scale validation
3. compare total trade, textile-chain made, importer counts, and score-per-polity against the prior level-7 sweep
4. only then decide whether textile/fine-clothing and strategic/finished JSON tuning is needed

If one more trade pass is needed, keep it narrow:
- prefer JSON-only tuning
- target only the most universal remaining winner
- rerun the same validated level-6 slice instead of broadening scope

## Entry Points

Primary working entry points:
- review driver: `cmd/review_planets/main.go`
- trade/climate code: `climgen/`
- authored tuning packs: `config/`
- schema index: `docs/CONFIG_SCHEMAS.md`
- repo guidance: `AGENTS.md`

## Source Field Follow-Up

- A confirmed trade-good source-field discontinuity was fixed: resource goods no longer turn discrete winning resource classes into fixed high source values. Ore, coal, evaporite, clay, stone, and placer-derived components now follow continuous affinity fields instead of boosting marginal classified cells to hardcoded deposit strengths. Trade-goods cache is `tradegoods-v2`; economy cache is `economy-v27`.
- Targeted seed314 rerun after this fix: L6 trade moved from `3.91/5.26` to `3.65/4.78`, and L7 moved from `1.95/2.57` to `1.92/2.50`. This removes a real resolution amplifier for resource sources, but it is not the main seed314 outlier.
- Remaining seed314 source mismatch is dominated by upstream climate/vegetation/agriculture fields: L7 still has higher timber/resin/fiber/crop source support, e.g. timber land mean `0.239 -> 0.304`, resin `0.220 -> 0.264`, fiber `0.285 -> 0.319`, and crop `0.394 -> 0.415`. That keeps L7 polities locally supplied for timber/resin-derived processed and strategic goods, erasing import need.
- Next investigation should target the upstream vegetation/climate-derived source fields, not multimodal matching or transport capacity.

## Terrain Blueprint Follow-Up

- The seed314 climate/source mismatch traced upstream to terrain layout, not precipitation or trade tuning. L6 and L7 had similar total land fraction but very different land geography: L6 land mean latitude was about `-6.8°` with `49%` tropical land, while L7 was about `+14.4°` with `64%` tropical land. That explains the warmer/wetter L7 vegetation and stronger timber/resin/fiber/crop source fields.
- The direct terrain cause was mesh-dependent optimized plate-layout selection. For the same seed, L6 selected layout attempt `11` and L7 selected attempt `5`, with different continental/oceanic plate assignments.
- `terrain-v5` fixes this by selecting a plate blueprint on a fixed reference mesh and projecting the same physical plate centers, growth weights, and continental/oceanic assignments onto the requested mesh. This targets the resolution-dependent root instead of patching downstream climate or trade values.
- Added a terrain unit test that projects the same optimized plate blueprint across two mesh levels and checks stable plate centers, stable ocean/continent assignment, and bounded continental latitude drift.
- Targeted seed314 validation after `terrain-v5`: both L6 and L7 now select layout attempt `11` with `7` oceanic and `5` continental plates. Land geography now matches closely: L6 land mean latitude `-5.1°`, mean absolute latitude `27.4°`, tropical land `49.7%`; L7 land mean latitude `-5.5°`, mean absolute latitude `27.6°`, tropical land `49.7%`.
- Downstream climate/source summaries are now much closer: crop `28.3%` vs `28.0%`, wildlife timber `21.5%` vs `18.9%`, raw runoff ge42 `48.3%` vs `46.8%`, and raw channel ge42 `12.2%` vs `12.0%`. Seed314 trade still differs moderately (`1.46/2.07` at L6 vs `0.90/1.31` at L7), but the original warm/wet/source-field resolution dependency is no longer the cause.
- Review output now includes `landLatitudeDiag` so future L6/L7 summaries expose land latitude distribution directly instead of requiring ad hoc terrain-cache parsing.
- Follow-up investigation found another structural terrain resolution dependency: hotspot placement/elevation used the same RNG stream after plate rotations. Plate-rotation random consumption can differ with mesh/topology, so the same seed could receive different hotspot histories after the same plate blueprint. `terrain-v6` decouples rotation, hotspot placement, and hotspot elevation RNG streams by deterministic stage seed.
- Terrain-only seed314 check after the stage-RNG split: L6/L7 land latitude remains matched (`-2.9°` vs `-3.2°` mean land latitude; tropical land `49.8%` vs `49.1%`). Land-component divergence is much smaller than the `terrain-v5` full run: components `37` vs `26`, tiny-equivalent islands `25` vs `21`, and large-equivalent components `5` vs `4`. This validates the RNG-stream drift as a real remaining resolution dependency in terrain feature materialization.
- RNG audit follow-up retained as `terrain-v7`: terrain distance fields no longer use randomized BFS seeded from map iteration; plate rotations use independent per-plate RNG streams; hotspot placement uses independent per-hotspot RNG streams; hotspot elevation uses per-chain/per-island streams keyed from physical positions; and hotspot tracing uses physical plate-layout ownership rather than nearest-cell ownership to decide whether a hotspot has crossed a plate-type boundary. This prevents resolution-dependent branch length in one feature from shifting random draws for later features.
- The audit also showed a remaining non-RNG materialization issue: even when hotspot histories are isolated, the same physical hotspot chain can still produce different realized island-component counts because physical events collapse into fewer coarse cells at L6 and survive as separate cells/components at L7. Treat remaining hotspot/component count divergence as a materialization/resampling problem, not shared RNG drift.
- `terrain-v8` addresses that materialization issue. Hotspot events closer than `0.75 * IslandSpacingRadians` are coalesced into one physical volcanic complex before rasterization, sub-resolution hotspot islands below `MinEmergentHotspotIslandRadius` remain submerged rather than becoming one-cell land components, and plate rotations now use physical blueprint centers instead of nearest mesh cell centers.
- Terrain-only seed314 after `terrain-v8`: both L6 and L7 have identical hotspot chain structure (`12` chains, `9` oceanic, `3` continental; oceanic lengths `[3 3 6 3 8 8 2 4 3]`; continental lengths `[2 5 3]`). Land components are now `12` vs `11`, tiny-equivalent islands `8` vs `5`, large-equivalent components `4` vs `4`, and land latitude remains matched (`-5.0°` vs `-4.9°` mean land latitude).
