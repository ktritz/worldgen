# Next Session Prompt - Temperature Wind Effects

## Quick Context

Read `climgen/HANDOFF_TEMPERATURE_V4.md` for full details.

We implemented wind maritime influence using a **fetch-based approach** (not backtracking). Backtracking failed because small wind direction differences compound over distance, creating patchy artifacts.

The fetch-based algorithm:
1. For each land cell, trace upwind
2. Count weighted ocean fraction along the path
3. Blend local temp with ocean temp based on influence

## Immediate Problem: Broken Diff Tool

The image-based diff tool (`cmd/diff_images/`) is broken - it uses pixel brightness, but the temperature colormap doesn't map linearly to brightness. Shows wrong results.

**Fix needed:** Create a proper temperature diff visualization:
1. Run test twice (baseline vs wind-on), save `result.TemperatureCelsius` arrays
2. Compute per-cell difference: `wind_temp - baseline_temp`
3. Render with diverging colormap: blue=colder with wind, red=warmer

Options:
- Add a `--save-temps` flag to test_temperature that dumps binary temp data
- Or create a new comparison mode in the test itself
- Render diff using same color scheme as main temp map but centered on zero

## Verify Wind Effect Direction

After fixing diff, check:
1. Orange stripe at ~30°N on western coast of main landmass - bug or correct?
2. Does it match physics? (onshore winds from warm/cold ocean currents)
3. Compare with wind direction visualization in `output/wind/`

## Current Settings (temperature.go)

```go
DefaultWindOriginForcing         = 8.0    // W/m²
DefaultWindBacktrackDistance     = 800.0  // km
```

May need tuning after we can visualize the effect properly.

## Key Files

- `climgen/temperature_transport.go` - `ComputeWindMaritimeInfluence()` (line ~690)
- `climgen/temperature_balance.go` - Where wind forcing is applied
- `cmd/diff_images/main.go` - BROKEN diff tool (delete or fix)

## Test

```bash
go run ./cmd/test_temperature
# Output: output/temperature/temp_full_climate.png
```

Compare against baseline (wind=0) to verify effects.
