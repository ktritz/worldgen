#!/usr/bin/env python3
"""Summarize trade diagnostics from review_planets sweep outputs."""

from __future__ import annotations

import argparse
import csv
import json
import os
import re
import tempfile
from pathlib import Path


SEED_RE = re.compile(r"^seed=(\d+)$")
MARKET_DIAG_RE = re.compile(r"marketMakeDiag:\s+(.*)$")
CATEGORY_MARKET_RE = re.compile(r"categoryMarket\[(\w+)\]:\s+(.*)$")
CATEGORY_FLOW_MIX_RE = re.compile(r"categoryFlowMix=(.*)$")
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


def extract_seed_summary(path: Path) -> dict[str, str]:
    summary = {
        "seed": "",
        "cap_winners": "",
        "processed_market_exports": "",
        "processed_market_made": "",
        "processed_flow_mix": "",
        "processed_goods": "",
        "luxury_goods": "",
        "strategic_goods": "",
    }
    for raw_line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        line = raw_line.strip()
        seed_match = SEED_RE.match(line)
        if seed_match:
            summary["seed"] = seed_match.group(1)
            continue
        market_diag_match = MARKET_DIAG_RE.search(line)
        if market_diag_match and "capWinners=" in market_diag_match.group(1):
            diag = market_diag_match.group(1)
            cap_winners = diag.split("capWinners=", 1)[1].split(" capLosers=", 1)[0].strip()
            summary["cap_winners"] = cap_winners
            continue
        category_market_match = CATEGORY_MARKET_RE.search(line)
        if category_market_match:
            category = category_market_match.group(1)
            payload = category_market_match.group(2)
            if category == "processed":
                exports = payload.split("exports=", 1)[1].split(" imports=", 1)[0].strip()
                made = payload.split(" made=", 1)[1].strip() if " made=" in payload else ""
                summary["processed_market_exports"] = exports
                summary["processed_market_made"] = made
            continue
        flow_mix_match = CATEGORY_FLOW_MIX_RE.search(line)
        if flow_mix_match:
            payload = flow_mix_match.group(1)
            for part in [segment.strip() for segment in payload.split(",")]:
                if part.startswith("processed:"):
                    summary["processed_flow_mix"] = part
            continue
        category_goods_match = CATEGORY_GOODS_RE.search(line)
        if category_goods_match:
            category = category_goods_match.group(1)
            goods = category_goods_match.group(2).strip()
            if category == "processed":
                summary["processed_goods"] = goods
            elif category == "luxury":
                summary["luxury_goods"] = goods
            elif category == "strategic":
                summary["strategic_goods"] = goods
    return summary


def main() -> int:
    args = parse_args()
    sweep_dir = Path(args.sweep_dir)
    rows = [extract_seed_summary(path) for path in sorted(sweep_dir.glob("seed_*.txt"))]

    fieldnames = [
        "seed",
        "cap_winners",
        "processed_market_exports",
        "processed_market_made",
        "processed_flow_mix",
        "processed_goods",
        "luxury_goods",
        "strategic_goods",
    ]
    tsv_path = sweep_dir / "trade_focus_summary.tsv"
    with tempfile.NamedTemporaryFile("w", dir=sweep_dir, delete=False, encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames, delimiter="\t")
        writer.writeheader()
        writer.writerows(rows)
        handle.flush()
        os.fsync(handle.fileno())
        temp_path = Path(handle.name)
    temp_path.replace(tsv_path)
    write_text_atomic(sweep_dir / "trade_focus_summary.json", json.dumps(rows, indent=2, sort_keys=True) + "\n")
    print(f"wrote {len(rows)} trade focus rows to {sweep_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
