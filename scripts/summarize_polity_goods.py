#!/usr/bin/env python3
"""Summarize polity-level supply, demand, surplus, and deficits by trade good."""

from __future__ import annotations

import argparse
import csv
import json
import os
import re
import tempfile
from collections import defaultdict
from pathlib import Path


GOOD_RE = re.compile(
    r"polityGoodAggregate\[(?P<good>[^\]]+)\]: "
    r"category=(?P<category>\S+) "
    r"supply=(?P<supply>[0-9.]+) "
    r"demand=(?P<demand>[0-9.]+) "
    r"surplus=(?P<surplus>[0-9.]+) "
    r"deficit=(?P<deficit>[0-9.]+) "
    r"exporters=(?P<exporters>[0-9]+) "
    r"importers=(?P<importers>[0-9]+) "
    r"maxExport=(?P<max_export>[0-9.]+) "
    r"maxImport=(?P<max_import>[0-9.]+) "
    r"scarcity=(?P<scarcity>n/a|[0-9.]+)"
)
CATEGORY_RE = re.compile(
    r"polityCategoryAggregate\[(?P<category>[^\]]+)\]: "
    r"supply=(?P<supply>[0-9.]+) "
    r"demand=(?P<demand>[0-9.]+) "
    r"surplus=(?P<surplus>[0-9.]+) "
    r"deficit=(?P<deficit>[0-9.]+) "
    r"exporters=(?P<exporters>[0-9]+) "
    r"importers=(?P<importers>[0-9]+) "
    r"maxExport=(?P<max_export>[0-9.]+) "
    r"maxImport=(?P<max_import>[0-9.]+)"
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sweep-dir", required=True, help="Directory containing seed_<n>.txt review outputs")
    return parser.parse_args()


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


def parse_seed_file(path: Path) -> tuple[list[dict[str, str | float | int]], list[dict[str, str | float | int]]]:
    seed = path.stem.split("_", 1)[1]
    goods: list[dict[str, str | float | int]] = []
    categories: list[dict[str, str | float | int]] = []
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        good_match = GOOD_RE.search(line)
        if good_match:
            data = good_match.groupdict()
            scarcity = data["scarcity"]
            goods.append(
                {
                    "seed": seed,
                    "good": data["good"],
                    "category": data["category"],
                    "supply": float(data["supply"]),
                    "demand": float(data["demand"]),
                    "surplus": float(data["surplus"]),
                    "deficit": float(data["deficit"]),
                    "exporters": int(data["exporters"]),
                    "importers": int(data["importers"]),
                    "max_export": float(data["max_export"]),
                    "max_import": float(data["max_import"]),
                    "scarcity": None if scarcity == "n/a" else float(scarcity),
                }
            )
            continue
        category_match = CATEGORY_RE.search(line)
        if category_match:
            data = category_match.groupdict()
            categories.append(
                {
                    "seed": seed,
                    "category": data["category"],
                    "supply": float(data["supply"]),
                    "demand": float(data["demand"]),
                    "surplus": float(data["surplus"]),
                    "deficit": float(data["deficit"]),
                    "exporters": int(data["exporters"]),
                    "importers": int(data["importers"]),
                    "max_export": float(data["max_export"]),
                    "max_import": float(data["max_import"]),
                }
            )
    return goods, categories


def avg(values: list[float]) -> float:
    return sum(values) / len(values) if values else 0.0


def stringify(value: str | float | int | None) -> str:
    if value is None:
        return ""
    if isinstance(value, float):
        return f"{value:.2f}"
    return str(value)


def stringify_rows(rows: list[dict[str, str | float | int | None]]) -> list[dict[str, str]]:
    return [{key: stringify(value) for key, value in row.items()} for row in rows]


def summarize_goods(rows: list[dict[str, str | float | int | None]]) -> list[dict[str, str | float | int]]:
    grouped: dict[str, list[dict[str, str | float | int | None]]] = defaultdict(list)
    for row in rows:
        grouped[str(row["good"])].append(row)

    total_seeds = len({str(row["seed"]) for row in rows})
    out: list[dict[str, str | float | int]] = []
    for good, good_rows in grouped.items():
        category = str(good_rows[0]["category"])
        supplies = [float(row["supply"]) for row in good_rows]
        demands = [float(row["demand"]) for row in good_rows]
        surpluses = [float(row["surplus"]) for row in good_rows]
        deficits = [float(row["deficit"]) for row in good_rows]
        exporters = [float(row["exporters"]) for row in good_rows]
        importers = [float(row["importers"]) for row in good_rows]
        max_exports = [float(row["max_export"]) for row in good_rows]
        max_imports = [float(row["max_import"]) for row in good_rows]
        scarcity = [float(row["scarcity"]) for row in good_rows if row["scarcity"] is not None]
        out.append(
            {
                "good": good,
                "category": category,
                "seeds": total_seeds,
                "present": len(good_rows),
                "importer_present": sum(1 for value in importers if value > 0),
                "exporter_present": sum(1 for value in exporters if value > 0),
                "avg_supply": avg(supplies),
                "avg_demand": avg(demands),
                "avg_surplus": avg(surpluses),
                "avg_deficit": avg(deficits),
                "avg_exporters": avg(exporters),
                "avg_importers": avg(importers),
                "avg_max_export": avg(max_exports),
                "avg_max_import": avg(max_imports),
                "avg_scarcity": avg(scarcity),
                "deficit_to_demand": avg(deficits) / avg(demands) if avg(demands) > 0 else 0,
                "surplus_to_supply": avg(surpluses) / avg(supplies) if avg(supplies) > 0 else 0,
            }
        )
    return sorted(out, key=lambda row: (str(row["category"]), -float(row["avg_deficit"]), str(row["good"])))


def summarize_categories(rows: list[dict[str, str | float | int]]) -> list[dict[str, str | float | int]]:
    grouped: dict[str, list[dict[str, str | float | int]]] = defaultdict(list)
    for row in rows:
        grouped[str(row["category"])].append(row)

    total_seeds = len({str(row["seed"]) for row in rows})
    out: list[dict[str, str | float | int]] = []
    for category, category_rows in grouped.items():
        supplies = [float(row["supply"]) for row in category_rows]
        demands = [float(row["demand"]) for row in category_rows]
        surpluses = [float(row["surplus"]) for row in category_rows]
        deficits = [float(row["deficit"]) for row in category_rows]
        exporters = [float(row["exporters"]) for row in category_rows]
        importers = [float(row["importers"]) for row in category_rows]
        out.append(
            {
                "category": category,
                "seeds": total_seeds,
                "present": len(category_rows),
                "avg_supply": avg(supplies),
                "avg_demand": avg(demands),
                "avg_surplus": avg(surpluses),
                "avg_deficit": avg(deficits),
                "avg_exporters": avg(exporters),
                "avg_importers": avg(importers),
                "deficit_to_demand": avg(deficits) / avg(demands) if avg(demands) > 0 else 0,
                "surplus_to_supply": avg(surpluses) / avg(supplies) if avg(supplies) > 0 else 0,
            }
        )
    return sorted(out, key=lambda row: (-float(row["avg_deficit"]), str(row["category"])))


def main() -> int:
    args = parse_args()
    sweep_dir = Path(args.sweep_dir)
    good_rows: list[dict[str, str | float | int | None]] = []
    category_rows: list[dict[str, str | float | int]] = []
    for path in sorted(sweep_dir.glob("seed_*.txt")):
        goods, categories = parse_seed_file(path)
        good_rows.extend(goods)
        category_rows.extend(categories)

    good_fields = [
        "seed",
        "good",
        "category",
        "supply",
        "demand",
        "surplus",
        "deficit",
        "exporters",
        "importers",
        "max_export",
        "max_import",
        "scarcity",
    ]
    write_tsv_atomic(sweep_dir / "polity_goods_summary.tsv", good_fields, stringify_rows(good_rows))
    write_text_atomic(sweep_dir / "polity_goods_summary.json", json.dumps(good_rows, indent=2, sort_keys=True) + "\n")

    category_fields = [
        "seed",
        "category",
        "supply",
        "demand",
        "surplus",
        "deficit",
        "exporters",
        "importers",
        "max_export",
        "max_import",
    ]
    write_tsv_atomic(sweep_dir / "polity_category_summary.tsv", category_fields, stringify_rows(category_rows))
    write_text_atomic(
        sweep_dir / "polity_category_summary.json",
        json.dumps(category_rows, indent=2, sort_keys=True) + "\n",
    )

    good_aggregate = summarize_goods(good_rows)
    good_aggregate_fields = [
        "good",
        "category",
        "seeds",
        "present",
        "importer_present",
        "exporter_present",
        "avg_supply",
        "avg_demand",
        "avg_surplus",
        "avg_deficit",
        "avg_exporters",
        "avg_importers",
        "avg_max_export",
        "avg_max_import",
        "avg_scarcity",
        "deficit_to_demand",
        "surplus_to_supply",
    ]
    write_tsv_atomic(
        sweep_dir / "polity_goods_aggregate.tsv",
        good_aggregate_fields,
        stringify_rows(good_aggregate),
    )
    write_text_atomic(
        sweep_dir / "polity_goods_aggregate.json",
        json.dumps(good_aggregate, indent=2, sort_keys=True) + "\n",
    )

    category_aggregate = summarize_categories(category_rows)
    category_aggregate_fields = [
        "category",
        "seeds",
        "present",
        "avg_supply",
        "avg_demand",
        "avg_surplus",
        "avg_deficit",
        "avg_exporters",
        "avg_importers",
        "deficit_to_demand",
        "surplus_to_supply",
    ]
    write_tsv_atomic(
        sweep_dir / "polity_category_aggregate.tsv",
        category_aggregate_fields,
        stringify_rows(category_aggregate),
    )
    write_text_atomic(
        sweep_dir / "polity_category_aggregate.json",
        json.dumps(category_aggregate, indent=2, sort_keys=True) + "\n",
    )
    print(
        f"wrote {len(good_rows)} polity-good rows, {len(category_rows)} category rows, "
        f"{len(good_aggregate)} good aggregates, and {len(category_aggregate)} category aggregates to {sweep_dir}"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
