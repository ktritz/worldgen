#!/usr/bin/env python3
"""Summarize where processed goods appear across node, market, and flow outputs."""

from __future__ import annotations

import argparse
import csv
import json
import os
import re
import tempfile
from pathlib import Path


SEED_RE = re.compile(r"^seed=(\d+)$")
NODE_EXPORT_RE = re.compile(r"nodeExport\[(.*?)\]:.*exports=(.*)$")
TRADE_MARKET_RE = re.compile(r"tradeMarket\[(.*?)\]:.*exports=(.*?) imports=(.*?) made=(.*?) used=")
CATEGORY_GOODS_RE = re.compile(r"categoryGoods\[(\w+)\]=(.*)$")


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


def contains_good(segment: str, good: str) -> bool:
    return f"{good}:" in segment


def summarize_seed(path: Path) -> dict[str, str]:
    summary = {
        "seed": "",
        "node_paper": "",
        "node_soap": "",
        "market_export_paper": "",
        "market_export_soap": "",
        "market_made_paper": "",
        "market_made_soap": "",
        "processed_goods": "",
    }
    for raw_line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        line = raw_line.strip()
        seed_match = SEED_RE.match(line)
        if seed_match:
            summary["seed"] = seed_match.group(1)
            continue
        node_match = NODE_EXPORT_RE.search(line)
        if node_match:
            exports = node_match.group(2)
            if not summary["node_paper"] and contains_good(exports, "paper"):
                summary["node_paper"] = node_match.group(1)
            if not summary["node_soap"] and contains_good(exports, "soap"):
                summary["node_soap"] = node_match.group(1)
            continue
        market_match = TRADE_MARKET_RE.search(line)
        if market_match:
            label = market_match.group(1)
            exports = market_match.group(2)
            made = market_match.group(4)
            if not summary["market_export_paper"] and contains_good(exports, "paper"):
                summary["market_export_paper"] = label
            if not summary["market_export_soap"] and contains_good(exports, "soap"):
                summary["market_export_soap"] = label
            if not summary["market_made_paper"] and contains_good(made, "paper"):
                summary["market_made_paper"] = label
            if not summary["market_made_soap"] and contains_good(made, "soap"):
                summary["market_made_soap"] = label
            continue
        goods_match = CATEGORY_GOODS_RE.search(line)
        if goods_match and goods_match.group(1) == "processed":
            summary["processed_goods"] = goods_match.group(2).strip()
    return summary


def main() -> int:
    args = parse_args()
    sweep_dir = Path(args.sweep_dir)
    rows = [summarize_seed(path) for path in sorted(sweep_dir.glob("seed_*.txt"))]
    fieldnames = [
        "seed",
        "node_paper",
        "node_soap",
        "market_export_paper",
        "market_export_soap",
        "market_made_paper",
        "market_made_soap",
        "processed_goods",
    ]
    tsv_path = sweep_dir / "processed_path_summary.tsv"
    with tempfile.NamedTemporaryFile("w", dir=sweep_dir, delete=False, encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames, delimiter="\t")
        writer.writeheader()
        writer.writerows(rows)
        handle.flush()
        os.fsync(handle.fileno())
        temp_path = Path(handle.name)
    temp_path.replace(tsv_path)
    write_text_atomic(sweep_dir / "processed_path_summary.json", json.dumps(rows, indent=2, sort_keys=True) + "\n")
    print(f"wrote {len(rows)} processed path rows to {sweep_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
