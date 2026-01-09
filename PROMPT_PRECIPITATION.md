# Next Session Prompt

Continue tuning the precipitation system. Read HANDOFF_CLIMATE.md for full context.

## Immediate Task

Test the parameter changes just made:
```bash
go build -o test_temperature ./cmd/test_temperature && ./test_temperature --precip
```

**Parameters just changed (untested):**
- `RainfallFraction`: 0.4 → 0.15 (for smoother wet/dry gradients)
- `PrecipitationScale`: 700 → 1400 (to compensate)

## Goal

Get precipitation distribution matching Earth:
- Desert (<25 cm): ~30%
- Semi-arid (25-50 cm): ~15% ← was too low at 8.6%
- Temperate (100-150 cm): ~20%
- Avg: ~100 cm/year

## Key Files
- `climgen/precipitation.go` - Algorithm and parameters
- `cmd/test_temperature/test_precipitation.go` - Test with histograms
- `output/temperature/precipitation.png` - Visual output

## What Was Done This Session
1. Refactored test_temperature/main.go (1357 lines → 7 files)
2. Fixed precipitation algorithm (iterative moisture diffusion)
3. Added temperature-based moisture capacity (Clausius-Clapeyron)
4. Added ITCZ boost for tropical rainfall
5. Added subtropical dry belt suppression
6. Changed output to cm/year scale
7. Created discrete color map visualization
