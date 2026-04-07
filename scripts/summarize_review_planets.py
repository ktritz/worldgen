#!/usr/bin/env python3
"""Summarize cmd/review_planets text output into a compact seed table."""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path


KV_RE = re.compile(r"([A-Za-z]+)=([A-Za-z0-9.+-]+)")
COUNT_RE = re.compile(r"(ancestryCount|stanceCount)\[([^\]]+)\]=([0-9]+)")
TRADE_RE = re.compile(r"(coastalTrade|oceanTrade)\[([^\]]+)\]:")


@dataclass
class SeedSummary:
    seed: str
    proto: int | None = None
    polities: int | None = None
    land: int | None = None
    river: int | None = None
    river_inter: int | None = None
    coastal: dict[str, int] = field(default_factory=dict)
    coastal_inter: dict[str, int] = field(default_factory=dict)
    ocean: dict[str, int] = field(default_factory=dict)
    ocean_inter: dict[str, int] = field(default_factory=dict)
    ancestry: dict[str, int] = field(default_factory=dict)
    stances: dict[str, int] = field(default_factory=dict)


def parse_kv(line: str) -> dict[str, str]:
    return {match.group(1): match.group(2) for match in KV_RE.finditer(line)}


def parse_lines(lines: list[str]) -> list[SeedSummary]:
    summaries: list[SeedSummary] = []
    by_seed: dict[str, SeedSummary] = {}
    current: SeedSummary | None = None

    for raw in lines:
        line = raw.strip()
        if line.startswith("seed="):
            seed = line.split("=", 1)[1]
            current = SeedSummary(seed=seed)
            if seed in by_seed:
                index = summaries.index(by_seed[seed])
                summaries[index] = current
            else:
                summaries.append(current)
            by_seed[seed] = current
            continue
        if current is None:
            continue

        kv = parse_kv(line)
        if line.startswith("protoCivilizations:"):
            current.proto = int(kv["seeds"])
        elif line.startswith("politySpheres:"):
            current.polities = int(kv["spheres"])
        elif line.startswith("tradeNetwork:"):
            current.land = int(kv["corridors"])
        elif line.startswith("riverTrade:"):
            current.river = int(kv["corridors"])
            current.river_inter = int(kv["inter"])
        elif match := TRADE_RE.match(line):
            kind, vessel = match.groups()
            corridors = int(kv["corridors"])
            inter = int(kv.get("inter", "0"))
            if kind == "coastalTrade":
                current.coastal[vessel] = corridors
                current.coastal_inter[vessel] = inter
            else:
                current.ocean[vessel] = corridors
                current.ocean_inter[vessel] = inter
        elif match := COUNT_RE.match(line):
            count_kind, key, value = match.groups()
            target = current.ancestry if count_kind == "ancestryCount" else current.stances
            target[key] = int(value)

    return summaries


def top_counts(counts: dict[str, int], limit: int) -> str:
    if not counts:
        return "-"
    ordered = sorted(counts.items(), key=lambda item: (-item[1], item[0]))
    return ",".join(f"{key}:{value}" for key, value in ordered[:limit])


def value(value: int | None) -> str:
    return "-" if value is None else str(value)


def print_markdown(summaries: list[SeedSummary], vessel: str, ancestry_limit: int) -> None:
    print(
        "| seed | proto | polities | land | river | river_inter | "
        f"coastal_{vessel} | coastal_inter | ocean_{vessel} | ocean_inter | ancestry | stances |"
    )
    print(
        "|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|"
    )
    for summary in summaries:
        print(
            f"| {summary.seed} | {value(summary.proto)} | {value(summary.polities)} | "
            f"{value(summary.land)} | {value(summary.river)} | {value(summary.river_inter)} | "
            f"{value(summary.coastal.get(vessel))} | {value(summary.coastal_inter.get(vessel))} | "
            f"{value(summary.ocean.get(vessel))} | {value(summary.ocean_inter.get(vessel))} | "
            f"{top_counts(summary.ancestry, ancestry_limit)} | {top_counts(summary.stances, 5)} |"
        )


def print_tsv(summaries: list[SeedSummary], vessel: str, ancestry_limit: int) -> None:
    print(
        "\t".join(
            [
                "seed",
                "proto",
                "polities",
                "land",
                "river",
                "river_inter",
                f"coastal_{vessel}",
                "coastal_inter",
                f"ocean_{vessel}",
                "ocean_inter",
                "ancestry",
                "stances",
            ]
        )
    )
    for summary in summaries:
        print(
            "\t".join(
                [
                    summary.seed,
                    value(summary.proto),
                    value(summary.polities),
                    value(summary.land),
                    value(summary.river),
                    value(summary.river_inter),
                    value(summary.coastal.get(vessel)),
                    value(summary.coastal_inter.get(vessel)),
                    value(summary.ocean.get(vessel)),
                    value(summary.ocean_inter.get(vessel)),
                    top_counts(summary.ancestry, ancestry_limit),
                    top_counts(summary.stances, 5),
                ]
            )
        )


def read_inputs(paths: list[str]) -> list[str]:
    if not paths:
        return sys.stdin.readlines()
    lines: list[str] = []
    for path in paths:
        lines.extend(Path(path).read_text(encoding="utf-8").splitlines())
    return lines


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("paths", nargs="*", help="review_planets output files; reads stdin when omitted")
    parser.add_argument("--vessel", default="caravel", help="maritime vessel column to report")
    parser.add_argument("--format", choices=("markdown", "tsv"), default="markdown")
    parser.add_argument("--ancestry-limit", type=int, default=4)
    args = parser.parse_args()

    summaries = parse_lines(read_inputs(args.paths))
    if args.format == "markdown":
        print_markdown(summaries, args.vessel, args.ancestry_limit)
    else:
        print_tsv(summaries, args.vessel, args.ancestry_limit)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
