#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="/home/ktritz/projects/worldgen"
OUT_DIR="${OUT_DIR:-$ROOT_DIR/output/review_planets/sweeps/level67_resolution_civ_v58_maritime_v37_validation}"
SEEDS="${SEEDS:-4,42,84,91,123,255,314,777}"
GO_BIN="${GO_BIN:-/usr/local/go/bin/go}"
GOCACHE_DIR="${GOCACHE_DIR:-/tmp/go-build-cache}"

mkdir -p "$OUT_DIR" "$GOCACHE_DIR"

run_level_phase() {
  local level="$1"
  local phase="$2"
  local phase_dir="$OUT_DIR/level${level}_${phase}"

  mkdir -p "$phase_dir"
  echo "phase=${phase} level=${level} started_at=$(date --iso-8601=seconds)"
  /usr/bin/python3 scripts/refill_review_cache.py \
    --level "$level" \
    --seeds "$SEEDS" \
    --out-dir "$phase_dir" \
    --force \
    --cache true \
    --go-bin "$GO_BIN" \
    --gocache "$GOCACHE_DIR" \
    --review-arg=-width \
    --review-arg=64 \
    --review-arg=-maritime-compare-vessels \
    --review-arg=caravel

  /usr/bin/python3 scripts/summarize_review_planets.py \
    "$phase_dir"/seed_*.txt \
    --vessel caravel \
    --format tsv \
    > "$phase_dir/review_summary.tsv"

  /usr/bin/python3 scripts/summarize_trade_sweep.py --sweep-dir "$phase_dir"
  /usr/bin/python3 scripts/summarize_good_paths.py --sweep-dir "$phase_dir"
  /usr/bin/python3 scripts/summarize_polity_goods.py --sweep-dir "$phase_dir"
  /usr/bin/python3 scripts/summarize_processed_path.py --sweep-dir "$phase_dir"
  /usr/bin/python3 scripts/summarize_trade_chains.py --sweep-dir "$phase_dir"
  echo "phase=${phase} level=${level} completed_at=$(date --iso-8601=seconds)"
}

{
  echo "started_at=$(date --iso-8601=seconds)"
  echo "out_dir=$OUT_DIR"
  echo "seeds=$SEEDS"
  echo "go_bin=$GO_BIN"
  echo "gocache=$GOCACHE_DIR"
  echo "purpose=matched_l6_l7_resolution_civilization_validation"

  cd "$ROOT_DIR"

  run_level_phase 6 cache_fill
  run_level_phase 7 cache_fill
  run_level_phase 6 warmed_confirm
  run_level_phase 7 warmed_confirm

  /usr/bin/python3 scripts/compare_review_levels.py \
    --level-a 6 \
    --summary-a "$OUT_DIR/level6_warmed_confirm/summary.tsv" \
    --review-a "$OUT_DIR/level6_warmed_confirm/review_summary.tsv" \
    --level-b 7 \
    --summary-b "$OUT_DIR/level7_warmed_confirm/summary.tsv" \
    --review-b "$OUT_DIR/level7_warmed_confirm/review_summary.tsv" \
    --out-dir "$OUT_DIR"

  echo "completed_at=$(date --iso-8601=seconds)"
} > "$OUT_DIR/run.log" 2> "$OUT_DIR/run.err"
