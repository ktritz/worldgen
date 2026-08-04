#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="/home/ktritz/projects/worldgen"
OUT_DIR="$ROOT_DIR/output/review_planets/sweeps/level7_trade_polity_validation"
SEEDS="4,42,84,91,123,255,314,777"
GO_BIN="/usr/local/go/bin/go"
GOCACHE_DIR="/tmp/go-build-cache"

mkdir -p "$OUT_DIR" "$GOCACHE_DIR"

{
  echo "started_at=$(date --iso-8601=seconds)"
  echo "out_dir=$OUT_DIR"
  echo "level=7"
  echo "seeds=$SEEDS"

  cd "$ROOT_DIR"

  echo "phase=cache_fill"
  /usr/bin/python3 scripts/refill_review_cache.py \
    --level 7 \
    --seeds "$SEEDS" \
    --out-dir "$OUT_DIR" \
    --force \
    --cache true \
    --go-bin "$GO_BIN" \
    --gocache "$GOCACHE_DIR"
  cp "$OUT_DIR/summary.tsv" "$OUT_DIR/summary_cache_fill.tsv"

  echo "phase=warmed_confirm"
  /usr/bin/python3 scripts/refill_review_cache.py \
    --level 7 \
    --seeds "$SEEDS" \
    --out-dir "$OUT_DIR" \
    --force \
    --cache true \
    --go-bin "$GO_BIN" \
    --gocache "$GOCACHE_DIR"
  cp "$OUT_DIR/summary.tsv" "$OUT_DIR/summary_warmed_confirm.tsv"

  echo "phase=trade_summaries"
  /usr/bin/python3 scripts/summarize_trade_sweep.py \
    --sweep-dir "$OUT_DIR"

  /usr/bin/python3 scripts/summarize_good_paths.py \
    --sweep-dir "$OUT_DIR"

  /usr/bin/python3 scripts/summarize_polity_goods.py \
    --sweep-dir "$OUT_DIR"

  /usr/bin/python3 scripts/summarize_processed_path.py \
    --sweep-dir "$OUT_DIR"

  /usr/bin/python3 scripts/summarize_trade_chains.py \
    --sweep-dir "$OUT_DIR"

  echo "completed_at=$(date --iso-8601=seconds)"
} > "$OUT_DIR/run.log" 2> "$OUT_DIR/run.err"
