#!/usr/bin/env python3

import argparse
import json
import re
from collections import defaultdict
from pathlib import Path


PATH_RE = re.compile(
    r"tradeGoodPath\[(?P<good>[^\]]+)\]: "
    r"nodeSupply=(?P<node_supply>[0-9.]+) "
    r"nodeSurplus=(?P<node_surplus>[0-9.]+) "
    r"marketSupply=(?P<market_supply>[0-9.]+) "
    r"marketDemand=(?P<market_demand>[0-9.]+) "
    r"marketSurplus=(?P<market_surplus>[0-9.]+) "
    r"made=(?P<made>[0-9.]+) "
    r"politySupply=(?P<polity_supply>[0-9.]+) "
    r"polityDemand=(?P<polity_demand>[0-9.]+) "
    r"politySurplus=(?P<polity_surplus>[0-9.]+) "
    r"exporters=(?P<exporters>[0-9]+) "
    r"importers=(?P<importers>[0-9]+) "
    r"tradeScore=(?P<trade_score>[0-9.]+) "
    r"tradeVolume=(?P<trade_volume>[0-9.]+) "
    r"tradePairs=(?P<trade_pairs>[0-9]+)"
)


def parse_args():
    parser = argparse.ArgumentParser(description="Summarize tradeGoodPath diagnostics from a review sweep.")
    parser.add_argument("--sweep-dir", required=True, help="Directory containing seed_*.txt outputs")
    return parser.parse_args()


def parse_seed_file(path: Path):
    rows = []
    seed = path.stem.split("_", 1)[1]
    for line in path.read_text().splitlines():
        match = PATH_RE.search(line)
        if not match:
            continue
        data = match.groupdict()
        rows.append(
            {
                "seed": seed,
                "good": data["good"],
                "node_supply": float(data["node_supply"]),
                "node_surplus": float(data["node_surplus"]),
                "market_supply": float(data["market_supply"]),
                "market_demand": float(data["market_demand"]),
                "market_surplus": float(data["market_surplus"]),
                "made": float(data["made"]),
                "polity_supply": float(data["polity_supply"]),
                "polity_demand": float(data["polity_demand"]),
                "polity_surplus": float(data["polity_surplus"]),
                "exporters": int(data["exporters"]),
                "importers": int(data["importers"]),
                "trade_score": float(data["trade_score"]),
                "trade_volume": float(data["trade_volume"]),
                "trade_pairs": int(data["trade_pairs"]),
            }
        )
    return rows


def write_tsv(path: Path, rows):
    headers = [
        "seed",
        "good",
        "node_supply",
        "node_surplus",
        "market_supply",
        "market_demand",
        "market_surplus",
        "made",
        "polity_supply",
        "polity_demand",
        "polity_surplus",
        "exporters",
        "importers",
        "trade_score",
        "trade_volume",
        "trade_pairs",
    ]
    with path.open("w", encoding="utf-8") as handle:
        handle.write("\t".join(headers) + "\n")
        for row in rows:
            handle.write("\t".join(str(row[h]) for h in headers) + "\n")


def average(values):
    return sum(values) / len(values) if values else 0.0


def summarize_goods(rows):
    grouped = defaultdict(list)
    for row in rows:
        grouped[row["good"]].append(row)

    total_seeds = len({row["seed"] for row in rows})
    summary = []
    for good in sorted(grouped):
        good_rows = grouped[good]
        traded_rows = [row for row in good_rows if row["trade_score"] > 0]
        summary.append(
            {
                "good": good,
                "seeds": total_seeds,
                "present": len(good_rows),
                "traded": len(traded_rows),
                "avg_polity_supply": f"{average([row['polity_supply'] for row in good_rows]):.2f}",
                "avg_polity_demand": f"{average([row['polity_demand'] for row in good_rows]):.2f}",
                "avg_polity_surplus": f"{average([row['polity_surplus'] for row in good_rows]):.2f}",
                "avg_exporters": f"{average([row['exporters'] for row in good_rows]):.2f}",
                "avg_importers": f"{average([row['importers'] for row in good_rows]):.2f}",
                "avg_trade_score": f"{average([row['trade_score'] for row in good_rows]):.2f}",
                "avg_trade_volume": f"{average([row['trade_volume'] for row in good_rows]):.2f}",
                "max_trade_score": f"{max([row['trade_score'] for row in good_rows], default=0):.2f}",
                "max_trade_volume": f"{max([row['trade_volume'] for row in good_rows], default=0):.2f}",
                "avg_trade_pairs": f"{average([row['trade_pairs'] for row in good_rows]):.2f}",
            }
        )
    return sorted(summary, key=lambda row: (-int(row["traded"]), -float(row["avg_trade_score"]), row["good"]))


def write_summary_tsv(path: Path, rows):
    headers = [
        "good",
        "seeds",
        "present",
        "traded",
        "avg_polity_supply",
        "avg_polity_demand",
        "avg_polity_surplus",
        "avg_exporters",
        "avg_importers",
        "avg_trade_score",
        "avg_trade_volume",
        "max_trade_score",
        "max_trade_volume",
        "avg_trade_pairs",
    ]
    with path.open("w", encoding="utf-8") as handle:
        handle.write("\t".join(headers) + "\n")
        for row in rows:
            handle.write("\t".join(str(row[h]) for h in headers) + "\n")


def main():
    args = parse_args()
    sweep_dir = Path(args.sweep_dir)
    rows = []
    for path in sorted(sweep_dir.glob("seed_*.txt")):
        rows.extend(parse_seed_file(path))

    rows.sort(key=lambda row: (int(row["seed"]), row["good"]))
    write_tsv(sweep_dir / "good_path_summary.tsv", rows)
    (sweep_dir / "good_path_summary.json").write_text(json.dumps(rows, indent=2) + "\n", encoding="utf-8")
    aggregate_rows = summarize_goods(rows)
    write_summary_tsv(sweep_dir / "good_path_aggregate.tsv", aggregate_rows)
    (sweep_dir / "good_path_aggregate.json").write_text(json.dumps(aggregate_rows, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
