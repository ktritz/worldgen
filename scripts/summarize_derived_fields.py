#!/usr/bin/env python3
import argparse
import csv
import json
import math
import re
from pathlib import Path


DERIVED_KEY_RE = re.compile(r"phase done: layer=derived cache=\w+ elapsed_ms=\d+ key=([0-9a-f]+)")
TERRAIN_KEY_RE = re.compile(r"phase done: layer=terrain cache=\w+ elapsed_ms=\d+ key=([0-9a-f]+)")
SEED_RE = re.compile(r"seed_(\d+)\.txt$")


DEFAULT_FIELDS = [
    "vegetation.WetlandCover",
    "vegetation.ShrubCover",
    "vegetation.MangroveAffinity",
    "vegetation.SaltMarshAffinity",
    "vegetation.MoistureAvailability",
    "vegetation.Waterlogging",
    "vegetation.SoilAlluvial",
    "vegetation.TreeCover",
    "vegetation.GrassCover",
    "vegetation.RiparianHydrology",
    "vegetation.RiparianAridity",
    "vegetation.RiparianHeat",
    "vegetation.RiparianPrecip",
    "vegetation.RiparianAffinity",
    "agriculture.CropPotential",
    "agriculture.PasturePotential",
    "wildlife.TimberPotential",
    "wildlife.GamePotential",
    "wildlife.PeltPotential",
    "coastalResources.CoastalAccess",
    "coastalResources.CurrentProductivity",
    "coastalResources.UpwellingPotential",
    "coastalResources.OpenFishery",
    "coastalResources.EstuarineFishery",
    "coastalResources.ShellfishPotential",
    "waterResources.SurfaceReliability",
    "waterResources.SeasonalAvailability",
    "waterResources.GroundwaterPotential",
    "waterResources.LakeAccess",
    "waterResources.DroughtResilience",
    "resources.CoalAffinity",
    "resources.IronAffinity",
    "biome.AnnualMeanTempC",
    "biome.AnnualPrecipCm",
    "biome.AridityThresholdCm",
    "biome.AridityRatio",
    "biome.WarmSeasonPrecipShare",
    "biome.DrySeasonRatio",
    "biome.WarmestSeasonTempC",
    "biome.AnnualIceFraction",
    "biome.ForestAffinity",
    "biome.WetlandAffinity",
    "soils.Moisture",
    "soils.Drainage",
    "soils.Alluvial",
    "soils.Organic",
    "soils.Relief",
]


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


def summarize(values, prefix=""):
    if not values:
        return {
            f"{prefix}cells": 0,
            f"{prefix}mean": 0.0,
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


def load_land_mask(cache_dir, cache_key, cache):
    if not cache_key:
        return []
    if cache_key in cache:
        return cache[cache_key]
    path = cache_dir / f"{cache_key}.json"
    if not path.exists():
        return []
    data = json.loads(path.read_text())
    mask = [bool(value) for value in data.get("isLand", [])]
    cache[cache_key] = mask
    return mask


def land_values(values, land_mask):
    if not land_mask or len(land_mask) != len(values):
        return []
    return [value for value, is_land in zip(values, land_mask) if is_land]


def field_values(derived, field):
    subsystem, _, diagnostic = field.partition(".")
    if not subsystem or not diagnostic:
        return []
    result = derived.get(subsystem, {})
    diagnostics = result.get("Diagnostics", {})
    return [float(value) for value in diagnostics.get(diagnostic, [])]


def mean(rows, key):
    vals = [float(row[key]) for row in rows]
    return sum(vals) / len(vals) if vals else 0.0


def aggregate(rows):
    by_field = {}
    for row in rows:
        by_field.setdefault(row["field"], []).append(row)
    metric_fields = [
        key
        for key in rows[0].keys()
        if key not in {"seed", "field", "derived_key", "terrain_key"}
    ] if rows else []
    out = []
    for field, field_rows in sorted(by_field.items()):
        row = {"field": field, "seeds": len(field_rows)}
        for metric in metric_fields:
            row[f"avg_{metric}"] = mean(field_rows, metric)
        out.append(row)
    return out


def write_tsv(path, rows, fields):
    with path.open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fields, delimiter="\t")
        writer.writeheader()
        for row in rows:
            writer.writerow(row)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--sweep-dir", required=True)
    parser.add_argument("--derived-cache-dir", default="output/review_planets/cache/derived")
    parser.add_argument("--terrain-cache-dir", default="output/review_planets/cache/terrain")
    parser.add_argument("--fields", default=",".join(DEFAULT_FIELDS))
    args = parser.parse_args()

    sweep_dir = Path(args.sweep_dir)
    derived_cache_dir = Path(args.derived_cache_dir)
    terrain_cache_dir = Path(args.terrain_cache_dir)
    selected_fields = [part.strip() for part in args.fields.split(",") if part.strip()]

    land_masks = {}
    rows = []
    for seed_file in sorted(sweep_dir.glob("seed_*.txt")):
        seed_match = SEED_RE.search(seed_file.name)
        if not seed_match:
            continue
        text = seed_file.read_text()
        derived_match = DERIVED_KEY_RE.search(text)
        if not derived_match:
            continue
        terrain_match = TERRAIN_KEY_RE.search(text)
        terrain_key = terrain_match.group(1) if terrain_match else ""
        derived_key = derived_match.group(1)
        derived_path = derived_cache_dir / f"{derived_key}.json"
        if not derived_path.exists():
            continue
        derived = json.loads(derived_path.read_text())
        land_mask = load_land_mask(terrain_cache_dir, terrain_key, land_masks)
        seed = int(seed_match.group(1))
        for field in selected_fields:
            values = field_values(derived, field)
            rows.append(
                {
                    "seed": seed,
                    "field": field,
                    "derived_key": derived_key,
                    "terrain_key": terrain_key,
                    **summarize(values),
                    **summarize(land_values(values, land_mask), "land_"),
                }
            )

    rows.sort(key=lambda row: (row["field"], row["seed"]))
    fields = [
        "seed",
        "field",
        "derived_key",
        "terrain_key",
        "cells",
        "mean",
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
    write_tsv(sweep_dir / "derived_field_summary.tsv", rows, fields)
    aggregate_rows = aggregate(rows)
    aggregate_fields = ["field", "seeds"] + [
        key for key in aggregate_rows[0].keys() if key not in {"field", "seeds"}
    ] if aggregate_rows else ["field", "seeds"]
    write_tsv(sweep_dir / "derived_field_aggregate.tsv", aggregate_rows, aggregate_fields)
    (sweep_dir / "derived_field_summary.json").write_text(json.dumps(rows, indent=2))
    (sweep_dir / "derived_field_aggregate.json").write_text(json.dumps(aggregate_rows, indent=2))
    print(f"wrote {len(rows)} derived field rows and {len(aggregate_rows)} aggregate rows to {sweep_dir}")


if __name__ == "__main__":
    main()
