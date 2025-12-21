# Baseline Evaluation Results

**Date**: 2025-11-01
**Method**: Current implementation (5x overseeding + intelligent merging)
**Resolution**: Subdivision 5 (10,242 sites)
**Configuration**: Earth-like (20 target plates, 30% major ratio)

---

## Overall Score

**Earth Benchmark Score: 49.4%** (Poor)

- **Current**: 49.4%
- **Target**: >75.0%
- **Improvement needed**: 1.52x (25.6 percentage points)

This establishes our baseline for comparison with the mantle convection method.

---

## Component Breakdown

| Component | Score | Status | Notes |
|-----------|-------|--------|-------|
| **Plate Count** | 46.6% | ✗ Poor | Missing micro plates, too few majors |
| **Power Law Fit** | 47.4% | ✗ Poor | Wrong exponent (1.098 vs 0.390) |
| **Size Variation** | 19.5% | ✗ Poor | Insufficient inequality (Gini 0.433 vs 0.811) |
| **Boundary Quality** | 87.5% | ✓ Excellent | Good convexity from merging |
| **Spatial Distribution** | 51.2% | △ Fair | Reasonable geographic spread |
| **Continental Ratio** | 61.6% | △ Fair | Too few continental plates |

---

## Detailed Analysis

### Size Distribution (46.6%)

**Generated vs Earth:**
- Major plates (>6%): **4 vs 7** (-3, 43% deficit)
- Minor plates (0.18-6%): **16 vs 13** (+3, 23% surplus)
- Micro plates (<0.18%): **0 vs 19** (-19, 100% deficit)
- Total plates: **20 vs 39** (-19, 49% deficit)

**Key Issue**: Method cannot generate micro plates - creates too uniform distribution.

### Coverage Analysis

**Generated vs Earth:**
- Major coverage: **55.6% vs 76.7%** (-21.1 pp)
- Minor coverage: **44.4% vs 5.4%** (+39.0 pp)
- Micro coverage: **0.0% vs 0.9%** (-0.9 pp)

**Key Issue**: Minor plates are covering far too much area - should be dominated by majors.

### Size Variation (19.5%)

**Metrics:**
- Gini coefficient: **0.433 vs 0.811** (47% lower inequality)
- Size ratio (largest/smallest): **11x vs 1015x** (99% less extreme)
- Largest plate: **19.8% vs 20.3%** (close!)
- Smallest plate: **1.83% vs 0.02%** (92x too large)

**Key Issue**: Method cannot create extreme size variation - too uniform.

### Power Law Analysis (47.4%)

**Metrics:**
- Exponent (β): **1.098 vs 0.390** (2.8x too steep)
- R² fit: **0.948 vs 0.920** (excellent fit quality)
- KS statistic: **0.792** (high, poor match to Earth distribution)
- Valid range: **1.84% - 19.8%**

**Key Finding**: The method *does* produce a power law distribution (R² = 0.948), but with the **wrong exponent**. Earth's β ≈ 0.39 indicates extreme inequality, while our β ≈ 1.10 indicates moderate inequality. This is the fundamental problem.

### Continental Distribution (61.6%)

**Metrics:**
- Continental plates: **3 (15%) vs 13 (33%)** (18 pp deficit)
- Continental area: **35.8% vs 33.0%** (close!)

**Note**: Continental *area* is correct, but spread across too few plates.

### Boundary Quality (87.5%)

**Strengths:**
- Natural convexity from intelligent merging
- Organic curved boundaries
- Realistic plate shapes

**Weaknesses:**
- 12 plates with severe concavity (>15% concave boundary)
- 8 plates with moderate concavity (8-15%)
- 0 plates with excellent convexity (<8%)

Despite concavity issues, boundary quality is the **only component above 75%**.

---

## Root Cause Analysis

### Why the Current Method Fails

The **5x overseeding + intelligent merging** approach has fundamental limitations:

1. **No mechanism for extreme inequality**
   - Merging creates moderate size variation (Gini ≈ 0.43)
   - Cannot achieve Earth's extreme inequality (Gini ≈ 0.81)
   - Results in too-uniform plate sizes

2. **Cannot generate micro plates**
   - Small plates get absorbed during merging
   - No mechanism to preserve very small plates
   - Missing entire size class (0 micro plates)

3. **Wrong power law exponent**
   - Produces β ≈ 1.1 (moderate inequality)
   - Earth requires β ≈ 0.39 (extreme inequality)
   - Geometric merging doesn't match geological process

4. **Plate count limitation**
   - Can only create as many plates as target (20)
   - Earth has 39 plates with extreme size range
   - Method optimizes for count, not size distribution

### What Works Well

1. **Boundary quality** - Intelligent merging creates realistic shapes
2. **Major plate placement** - Strategic clustering works
3. **Continental coverage** - Area targets achieved
4. **Computational efficiency** - Fast generation (~10 seconds)

---

## Comparison to Documentation

From `DEV_CONTEXT.md`:
> "Current implementation achieves 0.34-0.39/1.0 validation score"

Our evaluation: **0.494** (49.4%)

**Difference**: We scored ~10 percentage points higher than documented. This may be due to:
- Different validation metrics (we use new comprehensive framework)
- Different configuration (20 plates vs documented tests)
- Improvements from recent code changes

However, both scores confirm: **current method is far from Earth-like** (need >75%).

---

## Key Insights for Phase 2

### What We Learned

1. **Power law is critical**: Earth's β = 0.39 creates extreme inequality naturally
2. **Micro plates matter**: Missing 19 micro plates costs significant score
3. **Merging has limits**: Cannot achieve extreme size variation through merging alone
4. **Physical processes needed**: Geological realism requires process-based generation

### Requirements for Mantle Convection Method

To achieve >75% score, the new method must:

1. **Generate correct power law** (β ≈ 0.39)
   - Use convection cell strength distribution
   - Natural self-organized criticality
   - Target R² > 0.90

2. **Create extreme size variation** (Gini ≈ 0.81)
   - Support 1000x size ratios
   - Enable very small micro plates
   - Preserve size hierarchy

3. **Generate micro plates** (target: ~19)
   - Mechanism for small stable plates
   - Prevent over-absorption
   - Realistic size distribution

4. **Maintain boundary quality** (current: 87.5%)
   - Preserve convexity from current method
   - Natural boundaries from stress field
   - Don't sacrifice for size distribution

5. **Correct plate counts** (7 major, 13 minor, 19 micro)
   - Parameter tuning for Earth-like distribution
   - Flexible for other world types
   - Predictable results

---

## Next Steps

### Immediate (This Week)
1. ✅ Baseline established (49.4% score)
2. ⬜ Begin mantle convection core algorithm
3. ⬜ Implement convection cell placement

### Short-term (Next 2 Weeks)
1. ⬜ Stress field calculation
2. ⬜ Plate boundary detection
3. ⬜ Initial test generation
4. ⬜ Compare to baseline score

### Target
- **Achieve >65%** with initial implementation (30% improvement)
- **Achieve >75%** with parameter optimization (50% improvement)
- **Maintain boundary quality** (>80% in that component)

---

## Testing Strategy

### Development Testing (Subdivision 5)
- Fast iteration (~10 seconds per test)
- Sufficient for algorithm development
- Good for parameter exploration
- Used for baseline (10,242 sites)

### Final Validation (Subdivision 6-7)
- High resolution verification
- Subdivision 6: ~40,962 sites
- Subdivision 7: ~163,842 sites
- Used for final method comparison

---

## Conclusion

The current plate generation method achieves **49.4% Earth similarity**, well below the **75% target**.

**Primary weaknesses:**
- Wrong power law exponent (1.098 vs 0.390)
- Insufficient size inequality (Gini 0.433 vs 0.811)
- Missing micro plates entirely (0 vs 19)

**Primary strength:**
- Good boundary quality (87.5%)

The evaluation framework is working correctly and provides clear, quantitative feedback. We are ready to begin Phase 2: implementing the mantle convection method with the goal of achieving >75% Earth similarity.

**Baseline established. Ready for Phase 2 implementation.**
