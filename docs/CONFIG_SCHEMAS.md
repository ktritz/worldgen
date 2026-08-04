# Config Schema Reference

This file is the quick entry point for humans and AI agents looking for authored
JSON schema definitions in this repo.

## How To Use This

1. Start with the Go schema/types file for the subsystem.
2. Check the matching defaults, merge, and validation files.
3. Then open the authored JSON pack under `config/`.

Do not infer field names only from existing JSON examples when a schema/types
file already exists.

## Trade Goods

- Schema/types:
  - `climgen/trade_goods_settings_types.go`
- Defaults:
  - `climgen/trade_goods_settings_defaults.go`
- Merge:
  - `climgen/trade_goods_settings_merge.go`
- Validation:
  - `climgen/trade_goods_settings_validate.go`
- Loader:
  - `climgen/trade_goods_settings_load.go`
- Authored pack:
  - `config/trade_goods_earthlike.json`

Top-level sections:
- `schemaVersion`
- `scarcity`
- `multimodal`
- `production`
- `demand`
- `goods`

The `demand` section supports category-wide and per-good controls:
- `categoryDemandScale`
- `goodDemandScale`
- `localSupplyReliefByCategory`
- `localSupplyReliefByGood`
- `driverSpecializationScale`
- `marketCategoryDemandScale`
- `marketGoodDemandScale`
- `marketWealthPullScale`
- `marketFeederPullScale`
- `driverSpecializationPivot`

The `multimodal` section supports:
- `capacityScaleByMode`
- `capacityFactorByMode`
- `volumeBaseByMode`
- `endpointNeedShareByCategory`
- `endpointSurplusReliefByCategory`
- `localNeedResponse`
- `globalRarityResponse`
- market wealth and feeder scaling fields

Each `goods[]` entry uses:
- `name`
- `category`
- `baseValue`
- `bulkiness`
- `perishability`
- optional source, input, production, demand, and market-tuning fields

## Transport

Land routes:
- Schema/types:
  - `climgen/land_routes_settings_types.go`
- Pack:
  - `config/land_routes_earthlike.json`

River routes:
- Schema/types:
  - `climgen/river_routes_settings_types.go`
- Pack:
  - `config/river_routes_earthlike.json`

Maritime vessels and route settings:
- Schema/types:
  - `climgen/maritime_route_settings_types.go`
  - `climgen/maritime_port_settings.go`
  - `climgen/coastal_trade_settings.go`
  - `climgen/ocean_trade_settings.go`
- Packs:
  - `config/maritime_vessels_earthlike.json`
  - `config/maritime_ports_earthlike.json`
  - `config/coastal_trade_earthlike.json`
  - `config/ocean_trade_earthlike.json`

Ocean trade stopover selection includes:
- `maxStopovers`
- `stopoverScoreFloor`
- `stopoverSpacingHops`

Ocean trade port candidate selection includes:
- `candidatePortThreshold`
- `candidateSecondaryPortFloor`
- `candidatePhysicalDeepwaterFloor`
- `maxCandidatePortsPerCivilization`

Maritime port stopover node selection includes:
- `stopoverSelection.minStopoverValue`
- `stopoverSelection.minPortSuitability`
- `stopoverSelection.scoreFloor`
- `stopoverSelection.stopoverValueWeight`
- `stopoverSelection.portSuitabilityWeight`
- `stopoverSelection.oceanExposureWeight`
- `stopoverSelection.landScarcityWeight`
- `stopoverSelection.fullComponentAreaEq`
- `stopoverSelection.minComponentScoreFactor`
- `stopoverSelection.componentTaperPower`

## Resources And Population

Resource abundance:
- Schema/types:
  - `climgen/resource_abundance_settings.go`
- Pack:
  - `config/resource_abundance_earthlike.json`

Population support:
- Schema/types:
  - `climgen/population_settings.go`
- Pack:
  - `config/population_support_earthlike.json`

Population support includes optional catchment smoothing fields for blending
cell-local food, water, trade, resource, suitability, and hazard inputs over a
physical neighborhood. The current earthlike pack keeps this disabled with
`catchmentBlend: 0.0` while upstream resolution fixes are validated:
- `catchmentHops`
- `catchmentBlend`

## Profiles

Profile catalog:
- Catalog loader/types:
  - `climgen/profile_catalog.go`
- Packs:
  - `config/profile_catalog_fantasy.json`
  - `config/profiles/fantasy/...`

## Review Cache Keys

When review output seems stale after model changes, check cache version keys in:
- `cmd/review_planets/cache.go`

Current review cache layers are:
- terrain
- climate
- derived
- trade_goods
- civilization
- maritime
- economy

If a change affects cached review behavior, the corresponding cache version must
be bumped in `cmd/review_planets/cache.go`.
