#!/usr/bin/env python3
"""Compare matched review_planets summaries across two mesh levels."""

from __future__ import annotations

import argparse
import csv
import json
import os
import tempfile
from pathlib import Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--level-a", type=int, required=True)
    parser.add_argument("--summary-a", required=True)
    parser.add_argument("--review-a", required=True)
    parser.add_argument("--level-b", type=int, required=True)
    parser.add_argument("--summary-b", required=True)
    parser.add_argument("--review-b", required=True)
    parser.add_argument("--out-dir", required=True)
    return parser.parse_args()


def read_tsv(path: str) -> dict[str, dict[str, str]]:
    with Path(path).open(encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle, delimiter="\t")
        return {row["seed"]: row for row in reader}


def as_float(value: str) -> float:
    if value in ("", "-"):
        return 0.0
    return float(value)


def as_int(value: str) -> int:
    if value in ("", "-"):
        return 0
    return int(value)


def ratio(numerator: float, denominator: float) -> str:
    if denominator == 0:
        return ""
    return f"{numerator / denominator:.3f}"


def average(values: list[float]) -> float:
    if not values:
        return 0.0
    return sum(values) / len(values)


def write_text_atomic(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", dir=path.parent, delete=False, encoding="utf-8") as handle:
        handle.write(text)
        handle.flush()
        os.fsync(handle.fileno())
        temp_path = Path(handle.name)
    temp_path.replace(path)


def write_tsv_atomic(path: Path, fieldnames: list[str], rows: list[dict[str, str]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", dir=path.parent, delete=False, encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames, delimiter="\t")
        writer.writeheader()
        writer.writerows(rows)
        handle.flush()
        os.fsync(handle.fileno())
        temp_path = Path(handle.name)
    temp_path.replace(path)


def main() -> int:
    args = parse_args()
    summary_a = read_tsv(args.summary_a)
    summary_b = read_tsv(args.summary_b)
    review_a = read_tsv(args.review_a)
    review_b = read_tsv(args.review_b)
    seeds = sorted(set(summary_a) & set(summary_b) & set(review_a) & set(review_b), key=int)

    fields = [
        "seed",
        f"score_l{args.level_a}",
        f"score_l{args.level_b}",
        "score_ratio",
        f"volume_l{args.level_a}",
        f"volume_l{args.level_b}",
        "volume_ratio",
        f"proto_l{args.level_a}",
        f"proto_l{args.level_b}",
        "proto_delta",
        f"polities_l{args.level_a}",
        f"polities_l{args.level_b}",
        "polities_delta",
        f"land_l{args.level_a}",
        f"land_l{args.level_b}",
        f"river_l{args.level_a}",
        f"river_l{args.level_b}",
        f"coastal_caravel_l{args.level_a}",
        f"coastal_caravel_l{args.level_b}",
        f"ocean_caravel_l{args.level_a}",
        f"ocean_caravel_l{args.level_b}",
    ]
    rows: list[dict[str, str]] = []
    for seed in seeds:
        score_a = as_float(summary_a[seed]["trade_score"])
        score_b = as_float(summary_b[seed]["trade_score"])
        volume_a = as_float(summary_a[seed]["trade_volume"])
        volume_b = as_float(summary_b[seed]["trade_volume"])
        proto_a = as_int(review_a[seed]["proto"])
        proto_b = as_int(review_b[seed]["proto"])
        polities_a = as_int(review_a[seed]["polities"])
        polities_b = as_int(review_b[seed]["polities"])
        rows.append(
            {
                "seed": seed,
                f"score_l{args.level_a}": f"{score_a:.2f}",
                f"score_l{args.level_b}": f"{score_b:.2f}",
                "score_ratio": ratio(score_b, score_a),
                f"volume_l{args.level_a}": f"{volume_a:.2f}",
                f"volume_l{args.level_b}": f"{volume_b:.2f}",
                "volume_ratio": ratio(volume_b, volume_a),
                f"proto_l{args.level_a}": str(proto_a),
                f"proto_l{args.level_b}": str(proto_b),
                "proto_delta": str(proto_b - proto_a),
                f"polities_l{args.level_a}": str(polities_a),
                f"polities_l{args.level_b}": str(polities_b),
                "polities_delta": str(polities_b - polities_a),
                f"land_l{args.level_a}": review_a[seed]["land"],
                f"land_l{args.level_b}": review_b[seed]["land"],
                f"river_l{args.level_a}": review_a[seed]["river"],
                f"river_l{args.level_b}": review_b[seed]["river"],
                f"coastal_caravel_l{args.level_a}": review_a[seed]["coastal_caravel"],
                f"coastal_caravel_l{args.level_b}": review_b[seed]["coastal_caravel"],
                f"ocean_caravel_l{args.level_a}": review_a[seed]["ocean_caravel"],
                f"ocean_caravel_l{args.level_b}": review_b[seed]["ocean_caravel"],
            }
        )

    score_ratios = [as_float(row["score_ratio"]) for row in rows if row["score_ratio"]]
    volume_ratios = [as_float(row["volume_ratio"]) for row in rows if row["volume_ratio"]]
    proto_deltas = [as_float(row["proto_delta"]) for row in rows]
    polity_deltas = [as_float(row["polities_delta"]) for row in rows]
    aggregate = {
        "levels": [args.level_a, args.level_b],
        "seeds": seeds,
        "seed_count": len(seeds),
        "mean_score_ratio": round(average(score_ratios), 3),
        "mean_volume_ratio": round(average(volume_ratios), 3),
        "mean_proto_delta": round(average(proto_deltas), 3),
        "mean_polities_delta": round(average(polity_deltas), 3),
        "max_score_ratio": round(max(score_ratios), 3) if score_ratios else 0,
        "min_score_ratio": round(min(score_ratios), 3) if score_ratios else 0,
    }

    out_dir = Path(args.out_dir)
    write_tsv_atomic(out_dir / "level_compare.tsv", fields, rows)
    write_text_atomic(out_dir / "level_compare.json", json.dumps({"aggregate": aggregate, "rows": rows}, indent=2, sort_keys=True) + "\n")
    print(json.dumps(aggregate, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
