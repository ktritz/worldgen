# Tectonic Plate Generation Parameters Guide

## Overview
This document describes all tunable parameters in the tectonic plate generation system, their effects, and how to configure them for different world types (Earth-like, Pangea supercontinent, Archipelago, etc.).

## Parameter Categories

### 1. Initial Plate Seeding
Controls the starting distribution and count of plates before refinement.

#### `initialSeedMultiplier` (Current: 2.3)
**Effect**: Controls how many initial plates are created before merging/splitting.
- **Increase (3.0+)**: More initial diversity → More micro plates, harder to create large majors → **Archipelago worlds**
- **Decrease (1.5-)**: Fewer starting plates → Easier large majors, fewer micros → **Pangea-like worlds** 
- **Earth-like**: 2.3x (89 initial plates for 39 target)

**World Type Applications**:
- **Pangea Supercontinent**: 1.2-1.5x (few massive plates)
- **Earth-like**: 2.0-2.5x (balanced distribution)  
- **Archipelago**: 3.0-4.0x (many small plates)

### 2. Major Plate Formation
Controls how aggressively plates are merged into Earth-scale major plates.

#### `maxMergeRatio` (Current: 0.85)
**Effect**: Percentage of plates that can be merged during major plate creation.
- **Increase (0.9+)**: Very aggressive merging → Few super-massive plates → **Pangea worlds**
- **Decrease (0.5-)**: Conservative merging → Many medium plates → **Fragmented worlds**
- **Earth-like**: 0.85 (85% of plates can be merged)

**World Type Applications**:
- **Pangea Supercontinent**: 0.9-0.95 (merge almost everything)
- **Earth-like**: 0.8-0.9 (aggressive but balanced)
- **Archipelago**: 0.3-0.6 (preserve many small plates)

### 3. Micro Plate Generation (Fracture Cascade)
Controls the creation of small plates through systematic fracturing.

#### `maxFractureRounds` (Current: 3)
**Effect**: How many rounds of minor→micro plate fracturing occur.
- **Increase (5+)**: More fracturing rounds → Many tiny plates → **Complex archipelago**
- **Decrease (1-2)**: Limited fracturing → Fewer micro plates → **Simpler geology**
- **Earth-like**: 3 rounds

#### `fracturesPerRound` (Current: max(2, neededMicros/3))
**Effect**: How many plates are fractured in each round.
- **Increase**: More aggressive fracturing → Higher micro plate density
- **Decrease**: Conservative fracturing → Fewer but larger micro plates

#### Fracture Size Thresholds
- **Large plates** (>4.2% area): Split into 4 pieces
- **Medium plates** (>2.4% area): Split into 3 pieces  
- **Small plates** (>0.18% area): Split into 2 pieces

**World Type Applications**:
- **Pangea**: maxFractureRounds=0-1 (no micro fragmentation)
- **Earth-like**: maxFractureRounds=2-4 (moderate complexity)
- **Archipelago**: maxFractureRounds=5-8 (maximum fragmentation)

### 4. Plate Protection Systems
Controls how well small plates survive the generation process.

#### Boundary Protection Bias
- **Micro plates**: +25% attraction bias (Current)
- **Minor plates**: +15% attraction bias (Current)
- **Major plates**: No extra bias

**Effect**: Higher bias = stronger protection from absorption.
- **Increase**: Better survival of small plates → More geological diversity
- **Decrease**: Large plates absorb small ones → Simpler, more uniform geology

#### Merge Protection Threshold (Current: 0.1%)
**Effect**: Plates below this size cannot be merged away.
- **Increase (0.2%+)**: Protect larger plates → More geological complexity
- **Decrease (0.05%)**: Less protection → Simpler plate structure

### 5. Continental vs Oceanic Assignment
Controls the balance between continental and oceanic plate types.

#### Continental Probability by Size
- **Major plates**: 60% continental (Current)
- **Minor plates**: 25% continental (Current)  
- **Micro plates**: 15% continental (Current)

**World Type Applications**:
- **Pangea**: Major=90%, Minor=70%, Micro=50% (land-dominated supercontinent)
- **Earth-like**: Major=60%, Minor=25%, Micro=15% (balanced)
- **Ocean World**: Major=20%, Minor=10%, Micro=5% (mostly oceanic)

## Configuration Presets

### Pangea Supercontinent Configuration
```json
{
  "initialSeedMultiplier": 1.3,
  "maxMergeRatio": 0.95,
  "maxFractureRounds": 1,
  "fracturesPerRound": 1,
  "microProtectionBias": 0.1,
  "continentalProbability": {
    "major": 0.9,
    "minor": 0.7,
    "micro": 0.5
  },
  "description": "Few massive continental plates forming supercontinents"
}
```

### Earth-like Configuration (Current Optimized)
```json
{
  "initialSeedMultiplier": 2.3,
  "maxMergeRatio": 0.85,
  "maxFractureRounds": 3,
  "fracturesPerRound": "neededMicros/3",
  "microProtectionBias": 0.25,
  "continentalProbability": {
    "major": 0.6,
    "minor": 0.25,
    "micro": 0.15
  },
  "description": "Balanced plate distribution matching Earth statistics"
}
```

### Archipelago World Configuration
```json
{
  "initialSeedMultiplier": 3.5,
  "maxMergeRatio": 0.4,
  "maxFractureRounds": 6,
  "fracturesPerRound": "neededMicros/2",
  "microProtectionBias": 0.4,
  "continentalProbability": {
    "major": 0.8,
    "minor": 0.6,
    "micro": 0.4
  },
  "description": "Many small continental plates creating island chains"
}
```

### Ocean World Configuration
```json
{
  "initialSeedMultiplier": 2.0,
  "maxMergeRatio": 0.7,
  "maxFractureRounds": 2,
  "fracturesPerRound": "neededMicros/4",
  "microProtectionBias": 0.15,
  "continentalProbability": {
    "major": 0.2,
    "minor": 0.1,
    "micro": 0.05
  },
  "description": "Primarily oceanic world with rare continental features"
}
```

## Parameter Interaction Effects

### Pangea Formation Strategy
1. **Low initial seeds** (1.3x) → Fewer starting plates
2. **Aggressive merging** (0.95) → Merge almost everything into supercontinents
3. **Minimal fracturing** (1 round) → Preserve large plate sizes
4. **High continental bias** (90%) → Land-dominated supercontinents

### Archipelago Formation Strategy  
1. **High initial seeds** (3.5x) → Many starting plates
2. **Conservative merging** (0.4) → Preserve small plate diversity
3. **Aggressive fracturing** (6 rounds) → Create many micro plates
4. **Moderate continental bias** (40-80%) → Mix of islands and ocean

### Complex Geology Strategy
1. **High fracture rounds** → Maximum micro plate diversity
2. **Strong protection** → Preserve geological complexity
3. **Balanced merging** → Mix of large and small features

## Implementation Notes

### Parameter Exposure Strategy
1. **Configuration files**: JSON presets for common world types
2. **CLI arguments**: `--world-type=pangea|earth|archipelago|ocean`
3. **Advanced mode**: Individual parameter override capabilities
4. **Validation**: Parameter range checking and warnings

### Testing Approach
1. **Benchmark scoring**: Each preset tested against geological realism
2. **Visual validation**: Generate sample worlds for each configuration
3. **Performance impact**: Document computational cost of different settings

This system enables procedural generation of diverse world types while maintaining geological plausibility based on the parameter choices.