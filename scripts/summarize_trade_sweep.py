#!/usr/bin/env python3
"""Summarize trade diagnostics from review_planets sweep outputs."""

from __future__ import annotations

import argparse
import csv
import json
import os
import re
import tempfile
from collections import defaultdict
from pathlib import Path


SEED_RE = re.compile(r"^seed=(\d+)$")
MARKET_DIAG_RE = re.compile(r"marketMakeDiag:\s+(.*)$")
CATEGORY_MARKET_RE = re.compile(r"categoryMarket\[(\w+)\]:\s+(.*)$")
CATEGORY_FLOW_MIX_RE = re.compile(r"categoryFlowMix=(.*)$")
CATEGORY_GOODS_RE = re.compile(r"categoryGoods\[(\w+)\]=(.*)$")
NAMED_VALUE_RE = re.compile(r"(?P<name>[A-Za-z0-9_]+):(?P<value>[0-9.]+)")
CATEGORY_GOOD_RE = re.compile(r"(?P<name>[A-Za-z0-9_]+):s(?P<score>[0-9.]+)/v(?P<volume>[0-9.]+)")


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


def parse_named_values(payload: str) -> list[tuple[str, float]]:
    return [(match.group("name"), float(match.group("value"))) for match in NAMED_VALUE_RE.finditer(payload)]


def parse_category_goods(payload: str) -> list[tuple[str, float, float]]:
    return [
        (match.group("name"), float(match.group("score")), float(match.group("volume")))
        for match in CATEGORY_GOOD_RE.finditer(payload)
    ]


def extract_seed_summary(path: Path) -> dict[str, str]:
    summary = {
        "seed": "",
        "cap_winners": "",
        "processed_market_exports": "",
        "processed_market_made": "",
        "processed_flow_mix": "",
        "processed_goods": "",
        "finished_goods": "",
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
            elif category == "finished":
                summary["finished_goods"] = goods
            elif category == "luxury":
                summary["luxury_goods"] = goods
            elif category == "strategic":
                summary["strategic_goods"] = goods
    return summary


def average(values: list[float]) -> float:
    if not values:
        return 0
    return sum(values) / len(values)


def summarize_cap_winners(rows: list[dict[str, str]]) -> list[dict[str, str]]:
    total_seeds = len(rows)
    counts: dict[str, int] = defaultdict(int)
    first_places: dict[str, int] = defaultdict(int)
    ranks: dict[str, list[float]] = defaultdict(list)
    winner_counts: dict[str, list[float]] = defaultdict(list)

    for row in rows:
        for rank, (good, count) in enumerate(parse_named_values(row["cap_winners"]), start=1):
            counts[good] += 1
            ranks[good].append(float(rank))
            winner_counts[good].append(count)
            if rank == 1:
                first_places[good] += 1

    summary = []
    for good in sorted(counts, key=lambda name: (-counts[name], -first_places[name], average(ranks[name]), name)):
        summary.append(
            {
                "good": good,
                "seeds": str(total_seeds),
                "present": str(counts[good]),
                "first_place": str(first_places[good]),
                "avg_rank": f"{average(ranks[good]):.2f}",
                "avg_winner_count": f"{average(winner_counts[good]):.2f}",
                "max_winner_count": f"{max(winner_counts[good]):.0f}",
            }
        )
    return summary


def summarize_category_goods(rows: list[dict[str, str]]) -> list[dict[str, str]]:
    total_seeds = len(rows)
    category_fields = {
        "processed": "processed_goods",
        "finished": "finished_goods",
        "luxury": "luxury_goods",
        "strategic": "strategic_goods",
    }
    scores: dict[tuple[str, str], list[float]] = defaultdict(list)
    volumes: dict[tuple[str, str], list[float]] = defaultdict(list)

    for row in rows:
        for category, field in category_fields.items():
            for good, score, volume in parse_category_goods(row[field]):
                key = (category, good)
                scores[key].append(score)
                volumes[key].append(volume)

    summary = []
    for category, good in sorted(scores, key=lambda key: (key[0], -len(scores[key]), -average(scores[key]), key[1])):
        key = (category, good)
        summary.append(
            {
                "category": category,
                "good": good,
                "seeds": str(total_seeds),
                "present": str(len(scores[key])),
                "avg_score": f"{average(scores[key]):.2f}",
                "avg_volume": f"{average(volumes[key]):.2f}",
                "max_score": f"{max(scores[key]):.2f}",
                "max_volume": f"{max(volumes[key]):.2f}",
            }
        )
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
        "finished_goods",
        "luxury_goods",
        "strategic_goods",
    ]
    tsv_path = sweep_dir / "trade_focus_summary.tsv"
    write_tsv_atomic(tsv_path, fieldnames, rows)
    write_text_atomic(sweep_dir / "trade_focus_summary.json", json.dumps(rows, indent=2, sort_keys=True) + "\n")

    winner_fields = ["good", "seeds", "present", "first_place", "avg_rank", "avg_winner_count", "max_winner_count"]
    winner_rows = summarize_cap_winners(rows)
    write_tsv_atomic(sweep_dir / "trade_focus_winners.tsv", winner_fields, winner_rows)

    category_fields = ["category", "good", "seeds", "present", "avg_score", "avg_volume", "max_score", "max_volume"]
    category_rows = summarize_category_goods(rows)
    write_tsv_atomic(sweep_dir / "trade_focus_category_goods.tsv", category_fields, category_rows)
    write_text_atomic(
        sweep_dir / "trade_focus_aggregate.json",
        json.dumps({"cap_winners": winner_rows, "category_goods": category_rows}, indent=2, sort_keys=True) + "\n",
    )

    print(f"wrote {len(rows)} trade focus rows and aggregate summaries to {sweep_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
