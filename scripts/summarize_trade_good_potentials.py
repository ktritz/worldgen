#!/usr/bin/env python3
import argparse
import csv
import json
import math
import re
from pathlib import Path


TRADE_GOODS_KEY_RE = re.compile(
    r"phase done: layer=trade_goods cache=\w+ elapsed_ms=\d+ key=([0-9a-f]+)"
)
TERRAIN_KEY_RE = re.compile(
    r"phase done: layer=terrain cache=\w+ elapsed_ms=\d+ key=([0-9a-f]+)"
)
SEED_RE = re.compile(r"seed_(\d+)\.txt$")


def percentile(values, pct):
    if not values:
        return 0.0
    values = sorted(values)
    idx = (len(values) - 1) * pct / 100.0
    lo = math.floor(idx)
    hi = math.ceil(idx)
    if lo == hi:
        return values[lo]
    return values[lo] * (hi - idx) + values[hi] * (idx - lo)


def summarize_values(values, prefix=""):
    if not values:
        return {
            f"{prefix}cells": 0,
            f"{prefix}mean": 0.0,
            f"{prefix}p50": 0.0,
            f"{prefix}p90": 0.0,
            f"{prefix}p95": 0.0,
            f"{prefix}p99": 0.0,
            f"{prefix}max": 0.0,
            f"{prefix}nonzero_share": 0.0,
            f"{prefix}positive_mean": 0.0,
            f"{prefix}share_gt_010": 0.0,
            f"{prefix}share_gt_025": 0.0,
            f"{prefix}share_gt_050": 0.0,
        }
    n = len(values)
    positives = [value for value in values if value > 0]
    return {
        f"{prefix}cells": n,
        f"{prefix}mean": sum(values) / n,
        f"{prefix}p50": percentile(values, 50),
        f"{prefix}p90": percentile(values, 90),
        f"{prefix}p95": percentile(values, 95),
        f"{prefix}p99": percentile(values, 99),
        f"{prefix}max": max(values),
        f"{prefix}nonzero_share": len(positives) / n,
        f"{prefix}positive_mean": sum(positives) / len(positives) if positives else 0.0,
        f"{prefix}share_gt_010": sum(1 for value in values if value > 0.10) / n,
        f"{prefix}share_gt_025": sum(1 for value in values if value > 0.25) / n,
        f"{prefix}share_gt_050": sum(1 for value in values if value > 0.50) / n,
    }


def mean(rows, key):
    vals = [float(row[key]) for row in rows]
    return sum(vals) / len(vals) if vals else 0.0


def load_land_mask(cache_dir, cache_key, cache):
    if not cache_key:
        return []
    if cache_key in cache:
        return cache[cache_key]
    terrain_path = cache_dir / f"{cache_key}.json"
    if not terrain_path.exists():
        return []
    data = json.loads(terrain_path.read_text())
    mask = [bool(value) for value in data.get("isLand", [])]
    cache[cache_key] = mask
    return mask


def land_values(values, land_mask):
    if not land_mask or len(land_mask) != len(values):
        return []
    return [value for value, is_land in zip(values, land_mask) if is_land]


GOOD_FIELDS = [
    "seed",
    "good",
    "cache_key",
    "terrain_key",
    "cells",
    "mean",
    "p50",
    "p90",
    "p95",
    "p99",
    "max",
    "nonzero_share",
    "positive_mean",
    "share_gt_010",
    "share_gt_025",
    "share_gt_050",
    "land_cells",
    "land_mean",
    "land_p50",
    "land_p90",
    "land_p95",
    "land_p99",
    "land_max",
    "land_nonzero_share",
    "land_positive_mean",
    "land_share_gt_010",
    "land_share_gt_025",
    "land_share_gt_050",
]


SOURCE_FIELDS = [
    "seed",
    "source",
    "cache_key",
    "terrain_key",
    *GOOD_FIELDS[4:],
]


def build_aggregate_rows(rows, name_key):
    by_name = {}
    for row in rows:
        by_name.setdefault(row[name_key], []).append(row)
    aggregate = []
    metric_fields = [field for field in rows[0].keys() if field not in {name_key, "seed", "cache_key", "terrain_key"}] if rows else []
    for name, name_rows in sorted(by_name.items()):
        out = {name_key: name, "seeds": len(name_rows)}
        for field in metric_fields:
            out[f"avg_{field}"] = mean(name_rows, field)
        aggregate.append(out)
    return aggregate


def write_rows(path, rows, fields):
    with path.open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fields, delimiter="\t")
        writer.writeheader()
        for row in rows:
            writer.writerow(row)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--sweep-dir", required=True)
    parser.add_argument("--cache-dir", default="output/review_planets/cache/trade_goods")
    parser.add_argument("--terrain-cache-dir", default="output/review_planets/cache/terrain")
    parser.add_argument("--goods", default="")
    parser.add_argument("--sources", default="")
    args = parser.parse_args()

    sweep_dir = Path(args.sweep_dir)
    cache_dir = Path(args.cache_dir)
    terrain_cache_dir = Path(args.terrain_cache_dir)
    selected_goods = {part.strip() for part in args.goods.split(",") if part.strip()}
    selected_sources = {part.strip() for part in args.sources.split(",") if part.strip()}

    rows = []
    source_rows = []
    land_masks = {}
    for seed_file in sorted(sweep_dir.glob("seed_*.txt")):
        seed_match = SEED_RE.search(seed_file.name)
        if not seed_match:
            continue
        text = seed_file.read_text()
        key_match = TRADE_GOODS_KEY_RE.search(text)
        if not key_match:
            continue
        terrain_key_match = TERRAIN_KEY_RE.search(text)
        terrain_key = terrain_key_match.group(1) if terrain_key_match else ""
        land_mask = load_land_mask(terrain_cache_dir, terrain_key, land_masks)
        cache_key = key_match.group(1)
        cache_path = cache_dir / f"{cache_key}.json"
        if not cache_path.exists():
            continue
        data = json.loads(cache_path.read_text())
        seed = int(seed_match.group(1))
        for good in data.get("Goods", []):
            name = good.get("Good", "")
            if selected_goods and name not in selected_goods:
                continue
            values = [float(value) for value in good.get("Potential", [])]
            summary = {
                **summarize_values(values),
                **summarize_values(land_values(values, land_mask), "land_"),
            }
            rows.append(
                {
                    "seed": seed,
                    "good": name,
                    "cache_key": cache_key,
                    "terrain_key": terrain_key,
                    **summary,
                }
            )
        for source, raw_values in data.get("Diagnostics", {}).get("SourceFields", {}).items():
            if selected_sources and source not in selected_sources:
                continue
            values = [float(value) for value in raw_values]
            source_rows.append(
                {
                    "seed": seed,
                    "source": source,
                    "cache_key": cache_key,
                    "terrain_key": terrain_key,
                    **summarize_values(values),
                    **summarize_values(land_values(values, land_mask), "land_"),
                }
            )

    rows.sort(key=lambda row: (row["good"], row["seed"]))
    source_rows.sort(key=lambda row: (row["source"], row["seed"]))
    write_rows(sweep_dir / "trade_good_potential_summary.tsv", rows, GOOD_FIELDS)
    aggregate_rows = build_aggregate_rows(rows, "good")
    aggregate_fields = ["good", "seeds"] + [field for field in aggregate_rows[0].keys() if field not in {"good", "seeds"}] if aggregate_rows else ["good", "seeds"]
    write_rows(sweep_dir / "trade_good_potential_aggregate.tsv", aggregate_rows, aggregate_fields)
    write_rows(sweep_dir / "trade_good_source_field_summary.tsv", source_rows, SOURCE_FIELDS)
    source_aggregate_rows = build_aggregate_rows(source_rows, "source")
    source_aggregate_fields = ["source", "seeds"] + [field for field in source_aggregate_rows[0].keys() if field not in {"source", "seeds"}] if source_aggregate_rows else ["source", "seeds"]
    write_rows(sweep_dir / "trade_good_source_field_aggregate.tsv", source_aggregate_rows, source_aggregate_fields)
    (sweep_dir / "trade_good_potential_summary.json").write_text(json.dumps(rows, indent=2))
    (sweep_dir / "trade_good_potential_aggregate.json").write_text(json.dumps(aggregate_rows, indent=2))
    (sweep_dir / "trade_good_source_field_summary.json").write_text(json.dumps(source_rows, indent=2))
    (sweep_dir / "trade_good_source_field_aggregate.json").write_text(json.dumps(source_aggregate_rows, indent=2))
    print(
        f"wrote {len(rows)} potential rows, {len(aggregate_rows)} potential aggregates, "
        f"{len(source_rows)} source rows, and {len(source_aggregate_rows)} source aggregates to {sweep_dir}"
    )


if __name__ == "__main__":
    main()
