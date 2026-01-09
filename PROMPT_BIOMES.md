# Next Session Prompt

Continue investigating biome distribution. Read HANDOFF_CLIMATE.md for full context.

## Immediate Task

Run the multi-seed analysis and compare to Earth:

```bash
go build -o analyze_biomes ./cmd/analyze_biomes
./analyze_biomes 20
```

## What to Look For

1. **Temperature distribution**: Is there enough land in each temperature zone?
2. **Precipitation in cold zones**: What % of boreal/tundra zones are desert-dry?
3. **Grassland band**: What % of temperate zone falls in 40-75cm range?

## Known Issues (from 5-seed preliminary)

- Temperate Grassland: 1-4% (should be 8-10%)
- Boreal Forest: 3-9% (should be 10-17%)
- Tundra: 1-3% (should be 3-10%)

## Suspected Causes

1. Clausius-Clapeyron effect too strong → cold regions too dry
2. Temperature gradient too sharp → not enough tundra zone
3. Precipitation bimodal → desert OR forest, not grassland

## Key Files

- `climgen/biome.go` - Classification thresholds
- `climgen/precipitation.go:102-116` - Clausius-Clapeyron moisture capacity
- `cmd/analyze_biomes/main.go` - Analysis tool

## After Analysis

Based on results, either:
1. Adjust biome thresholds (if temp/precip distribution is correct)
2. Adjust precipitation algorithm (if distribution is wrong)
3. Adjust temperature algorithm (if temp zones are wrong)

Do NOT make changes until you have the 20-seed statistics.
