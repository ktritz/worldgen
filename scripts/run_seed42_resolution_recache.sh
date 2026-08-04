#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="/home/ktritz/projects/worldgen"
OUT_ROOT="$ROOT_DIR/output/review_planets/sweeps/resolution_fix_seed42_terrain_v4"
GO_BIN="/usr/local/go/bin/go"
GOCACHE_DIR="/tmp/go-build-cache"
SEED="42"

summarize_level() {
  local out_dir="$1"
  /usr/bin/python3 scripts/summarize_review_planets.py "$out_dir/seed_${SEED}.txt" > "$out_dir/review_summary.txt"
  /usr/bin/python3 scripts/summarize_trade_good_potentials.py \
    --sweep-dir "$out_dir" \
    --goods dye_plants,fiber,wool,iron_ore,coal,timber,fish,salt \
    --sources dye_plants,fiber,pasture,timber,fish,coal,iron_ore,salt
  /usr/bin/python3 scripts/summarize_derived_fields.py \
    --sweep-dir "$out_dir"
}

compare_levels() {
  /usr/bin/python3 - "$OUT_ROOT/level6" "$OUT_ROOT/level7" "$OUT_ROOT" <<'PY'
import csv
import json
import sys
from pathlib import Path

level6 = Path(sys.argv[1])
level7 = Path(sys.argv[2])
out_root = Path(sys.argv[3])

jobs = [
    ("derived_field_aggregate.tsv", "field", ["avg_land_mean", "avg_land_p95", "avg_land_share_gt_025"]),
    ("trade_good_source_field_aggregate.tsv", "source", ["avg_land_mean", "avg_land_p95", "avg_land_share_gt_025"]),
    ("trade_good_potential_aggregate.tsv", "good", ["avg_land_mean", "avg_land_p95", "avg_land_share_gt_025"]),
]

def load_rows(path, key):
    with path.open(newline="") as f:
        return {row[key]: row for row in csv.DictReader(f, delimiter="\t")}

rows = []
for filename, key, metrics in jobs:
    lhs = load_rows(level6 / filename, key)
    rhs = load_rows(level7 / filename, key)
    for name in sorted(set(lhs) & set(rhs)):
        row = {"summary": filename, "name": name}
        for metric in metrics:
            base = float(lhs[name][metric])
            fine = float(rhs[name][metric])
            row[f"l6_{metric}"] = base
            row[f"l7_{metric}"] = fine
            row[f"ratio_{metric}"] = fine / base if base else 0.0
        rows.append(row)

fields = [
    "summary",
    "name",
    "l6_avg_land_mean",
    "l7_avg_land_mean",
    "ratio_avg_land_mean",
    "l6_avg_land_p95",
    "l7_avg_land_p95",
    "ratio_avg_land_p95",
    "l6_avg_land_share_gt_025",
    "l7_avg_land_share_gt_025",
    "ratio_avg_land_share_gt_025",
]
with (out_root / "level7_vs_level6_ratios.tsv").open("w", newline="") as f:
    writer = csv.DictWriter(f, fieldnames=fields, delimiter="\t")
    writer.writeheader()
    writer.writerows(rows)
(out_root / "level7_vs_level6_ratios.json").write_text(json.dumps(rows, indent=2))
print(f"wrote {len(rows)} level7/level6 ratio rows to {out_root}")
PY
}

mkdir -p "$OUT_ROOT/level6" "$OUT_ROOT/level7" "$GOCACHE_DIR"

{
  echo "started_at=$(date --iso-8601=seconds)"
  echo "out_root=$OUT_ROOT"
  echo "seed=$SEED"
  echo "purpose=seed42_resolution_recache_after_terrain_v4"

  cd "$ROOT_DIR"

  for level in 6 7; do
    out_dir="$OUT_ROOT/level${level}"
    echo "phase=cache_fill level=$level"
    /usr/bin/python3 scripts/refill_review_cache.py \
      --level "$level" \
      --seeds "$SEED" \
      --out-dir "$out_dir" \
      --force \
      --cache true \
      --go-bin "$GO_BIN" \
      --gocache "$GOCACHE_DIR" \
      --review-arg=-maritime-compare-vessels=caravel
    cp "$out_dir/summary.tsv" "$out_dir/summary_cache_fill.tsv"

    echo "phase=warmed_confirm level=$level"
    /usr/bin/python3 scripts/refill_review_cache.py \
      --level "$level" \
      --seeds "$SEED" \
      --out-dir "$out_dir" \
      --force \
      --cache true \
      --go-bin "$GO_BIN" \
      --gocache "$GOCACHE_DIR" \
      --review-arg=-maritime-compare-vessels=caravel
    cp "$out_dir/summary.tsv" "$out_dir/summary_warmed_confirm.tsv"

    echo "phase=summaries level=$level"
    summarize_level "$out_dir"
  done

  echo "phase=compare"
  compare_levels
  echo "completed_at=$(date --iso-8601=seconds)"
} > "$OUT_ROOT/run.log" 2> "$OUT_ROOT/run.err"
