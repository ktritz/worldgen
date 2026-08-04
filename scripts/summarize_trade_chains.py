#!/usr/bin/env python3
"""Summarize raw-to-processed trade chain diagnostics from review_planets outputs."""

from __future__ import annotations

import argparse
import csv
import json
import os
import re
import tempfile
from collections import defaultdict
from pathlib import Path


CHAIN_RE = re.compile(
    r"tradeChain\[(?P<raw>[^\]-]+)->(?P<processed>[^\]]+)\]: "
    r"nodeSupply=(?P<node_supply>[0-9.]+) "
    r"nodeSurplus=(?P<node_surplus>[0-9.]+) "
    r"marketSupply=(?P<market_supply>[0-9.]+) "
    r"marketSurplus=(?P<market_surplus>[0-9.]+) "
    r"marketDemand=(?P<market_demand>[0-9.]+) "
    r"made=(?P<made>[0-9.]+) "
    r"demandGap=(?P<demand_gap>[0-9.]+) "
    r"blockedMarkets=(?P<blocked_markets>[0-9]+)"
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


def parse_seed_file(path: Path) -> list[dict[str, str | float | int]]:
    seed = path.stem.split("_", 1)[1]
    rows: list[dict[str, str | float | int]] = []
    for raw_line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        match = CHAIN_RE.search(raw_line)
        if not match:
            continue
        data = match.groupdict()
        rows.append(
            {
                "seed": seed,
                "chain": f"{data['raw']}->{data['processed']}",
                "raw": data["raw"],
                "processed": data["processed"],
                "node_supply": float(data["node_supply"]),
                "node_surplus": float(data["node_surplus"]),
                "market_supply": float(data["market_supply"]),
                "market_surplus": float(data["market_surplus"]),
                "market_demand": float(data["market_demand"]),
                "made": float(data["made"]),
                "demand_gap": float(data["demand_gap"]),
                "blocked_markets": int(data["blocked_markets"]),
            }
        )
    return rows


def avg(values: list[float]) -> float:
    return sum(values) / len(values) if values else 0.0


def summarize(rows: list[dict[str, str | float | int]]) -> list[dict[str, str]]:
    grouped: dict[str, list[dict[str, str | float | int]]] = defaultdict(list)
    for row in rows:
        grouped[str(row["chain"])].append(row)

    out: list[dict[str, str]] = []
    total_seeds = len({str(row["seed"]) for row in rows})
    for chain in sorted(grouped):
        chain_rows = grouped[chain]
        made = [float(row["made"]) for row in chain_rows]
        demand_gap = [float(row["demand_gap"]) for row in chain_rows]
        blocked = [int(row["blocked_markets"]) for row in chain_rows]
        market_surplus = [float(row["market_surplus"]) for row in chain_rows]
        market_demand = [float(row["market_demand"]) for row in chain_rows]
        out.append(
            {
                "chain": chain,
                "seeds": str(total_seeds),
                "present": str(len(chain_rows)),
                "made_present": str(sum(1 for value in made if value > 0)),
                "blocked_present": str(sum(1 for value in blocked if value > 0)),
                "avg_made": f"{avg(made):.2f}",
                "avg_market_surplus": f"{avg(market_surplus):.2f}",
                "avg_market_demand": f"{avg(market_demand):.2f}",
                "avg_demand_gap": f"{avg(demand_gap):.2f}",
                "max_demand_gap": f"{max(demand_gap) if demand_gap else 0:.2f}",
                "max_blocked_markets": str(max(blocked) if blocked else 0),
            }
        )
    return out


def stringify_rows(rows: list[dict[str, str | float | int]]) -> list[dict[str, str]]:
    return [{key: str(value) for key, value in row.items()} for row in rows]


def main() -> int:
    args = parse_args()
    sweep_dir = Path(args.sweep_dir)
    rows: list[dict[str, str | float | int]] = []
    for path in sorted(sweep_dir.glob("seed_*.txt")):
        rows.extend(parse_seed_file(path))
    rows.sort(key=lambda row: (int(str(row["seed"])), str(row["chain"])))

    detail_fields = [
        "seed",
        "chain",
        "raw",
        "processed",
        "node_supply",
        "node_surplus",
        "market_supply",
        "market_surplus",
        "market_demand",
        "made",
        "demand_gap",
        "blocked_markets",
    ]
    write_tsv_atomic(sweep_dir / "trade_chain_summary.tsv", detail_fields, stringify_rows(rows))
    write_text_atomic(sweep_dir / "trade_chain_summary.json", json.dumps(rows, indent=2, sort_keys=True) + "\n")

    aggregate_fields = [
        "chain",
        "seeds",
        "present",
        "made_present",
        "blocked_present",
        "avg_made",
        "avg_market_surplus",
        "avg_market_demand",
        "avg_demand_gap",
        "max_demand_gap",
        "max_blocked_markets",
    ]
    aggregate_rows = summarize(rows)
    write_tsv_atomic(sweep_dir / "trade_chain_aggregate.tsv", aggregate_fields, aggregate_rows)
    write_text_atomic(
        sweep_dir / "trade_chain_aggregate.json",
        json.dumps(aggregate_rows, indent=2, sort_keys=True) + "\n",
    )
    print(f"wrote {len(rows)} trade-chain rows and {len(aggregate_rows)} aggregate rows to {sweep_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
