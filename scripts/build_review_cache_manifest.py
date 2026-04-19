#!/usr/bin/env python3
"""Build a per-seed cache-layer manifest from review sweep outputs."""

from __future__ import annotations

import argparse
import csv
import json
import os
import re
import tempfile
from pathlib import Path


PHASE_DONE_PREFIX_RE = re.compile(r"^\s+phase done:\s+")
SEED_RE = re.compile(r"^seed=(\d+)$")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sweep-dir", required=True, help="Directory containing seed_<n>.txt outputs")
    parser.add_argument(
        "--cache-root",
        default="output/review_planets/cache",
        help="Root cache directory used to check whether manifest keys exist on disk",
    )
    return parser.parse_args()


def write_text_atomic(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", dir=path.parent, delete=False, encoding="utf-8") as handle:
        handle.write(text)
        handle.flush()
        os.fsync(handle.fileno())
        temp_path = Path(handle.name)
    temp_path.replace(path)


def parse_phase_fields(line: str) -> dict[str, str]:
    match = PHASE_DONE_PREFIX_RE.search(line)
    if not match:
        return {}
    fields: dict[str, str] = {}
    remainder = line[match.end() :].strip()
    for token in remainder.split():
        if "=" not in token:
            continue
        key, value = token.split("=", 1)
        fields[key] = value
    return fields


def phase_manifest_rows(seed_file: Path, cache_root: Path) -> list[dict[str, str]]:
    seed = ""
    rows: list[dict[str, str]] = []
    text = seed_file.read_text(encoding="utf-8", errors="replace")
    for line in text.splitlines():
        seed_match = SEED_RE.match(line.strip())
        if seed_match:
            seed = seed_match.group(1)
            continue
        fields = parse_phase_fields(line)
        if not fields:
            continue
        layer = fields.get("layer", "")
        key = fields.get("key", "")
        vessel = fields.get("vessel", "")
        cache_file = cache_root / layer / f"{key}.json" if key else None
        rows.append(
            {
                "seed": seed,
                "layer": layer,
                "cache": fields.get("cache", ""),
                "elapsed_ms": fields.get("elapsed_ms", ""),
                "key": key,
                "vessel": vessel,
                "cache_file": "" if cache_file is None else str(cache_file),
                "cache_file_exists": "" if cache_file is None else str(cache_file.exists()).lower(),
                "source_file": str(seed_file),
            }
        )
    return rows


def write_manifest_tsv(path: Path, rows: list[dict[str, str]]) -> None:
    fieldnames = [
        "seed",
        "layer",
        "cache",
        "elapsed_ms",
        "key",
        "vessel",
        "cache_file_exists",
        "cache_file",
        "source_file",
    ]
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
    sweep_dir = Path(args.sweep_dir)
    cache_root = Path(args.cache_root)

    rows: list[dict[str, str]] = []
    for seed_file in sorted(sweep_dir.glob("seed_*.txt")):
        rows.extend(phase_manifest_rows(seed_file, cache_root))

    write_manifest_tsv(sweep_dir / "cache_manifest.tsv", rows)
    write_text_atomic(sweep_dir / "cache_manifest.json", json.dumps(rows, indent=2, sort_keys=True) + "\n")
    print(f"wrote {len(rows)} cache manifest rows to {sweep_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
