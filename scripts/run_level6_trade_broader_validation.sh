#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="/home/ktritz/projects/worldgen"
OUT_DIR="$ROOT_DIR/output/review_planets/sweeps/level6_trade_broader_validation"
SEEDS="4,6,7,42,55,58,84,91,101,123,128,144,177,202,255,314,512,777,999,1337"
GO_BIN="/usr/local/go/bin/go"
GOCACHE_DIR="/tmp/go-build-cache"

mkdir -p "$OUT_DIR" "$GOCACHE_DIR"

{
  echo "started_at=$(date --iso-8601=seconds)"
  echo "out_dir=$OUT_DIR"
  echo "seeds=$SEEDS"

  cd "$ROOT_DIR"

  /usr/bin/python3 scripts/refill_review_cache.py \
    --level 6 \
    --seeds "$SEEDS" \
    --out-dir "$OUT_DIR" \
    --cache true \
    --go-bin "$GO_BIN" \
    --gocache "$GOCACHE_DIR"

  /usr/bin/python3 scripts/summarize_trade_sweep.py \
    --sweep-dir "$OUT_DIR"

  /usr/bin/python3 scripts/summarize_good_paths.py \
    --sweep-dir "$OUT_DIR"

  /usr/bin/python3 scripts/summarize_processed_path.py \
    --sweep-dir "$OUT_DIR"

  echo "completed_at=$(date --iso-8601=seconds)"
} > "$OUT_DIR/run.log" 2> "$OUT_DIR/run.err"
