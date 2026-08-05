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
    # Density metrics. Raw counts of discrete objects are small integers whose
    # L6/L7 ratio swings wildly on a single corridor forming; a density divides
    # by a physical extent summed over cells, which barely quantizes. Keeping
    # both side by side is what distinguishes "the finer mesh resolves more
    # river/coast" (denominator grew, density flat) from "the finer mesh puts
    # more objects on the same feature" (density grew).
    nav_length: float | None = None
    coast_length: float | None = None
    ocean_area: float | None = None
    river_terminals: int | None = None
    terminals_per_nav_length: float | None = None
    river_corridors_per_nav_length: float | None = None
    coastal_ports: dict[str, int] = field(default_factory=dict)
    coastal_stopovers: dict[str, int] = field(default_factory=dict)
    ports_per_coast_length: dict[str, float] = field(default_factory=dict)
    coastal_corridors_per_coast_length: dict[str, float] = field(default_factory=dict)
    ocean_ports: dict[str, int] = field(default_factory=dict)
    ocean_stopovers: dict[str, int] = field(default_factory=dict)
    stopovers_per_ocean_area: dict[str, float] = field(default_factory=dict)
    ocean_corridors_per_ocean_area: dict[str, float] = field(default_factory=dict)
    ancestry: dict[str, int] = field(default_factory=dict)
    stances: dict[str, int] = field(default_factory=dict)


def parse_kv(line: str) -> dict[str, str]:
    return {match.group(1): match.group(2) for match in KV_RE.finditer(line)}


def optional_int(raw: str | None) -> int | None:
    if raw is None:
        return None
    try:
        return int(raw)
    except ValueError:
        return None


def optional_float(raw: str | None) -> float | None:
    if raw is None:
        return None
    try:
        return float(raw)
    except ValueError:
        return None


def set_optional(target: dict, key: str, value) -> None:
    """Only record present fields, so older reports stay parseable and the
    comparison script skips columns it cannot find rather than seeing zeros."""
    if value is not None:
        target[key] = value


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
            current.river_inter = int(kv.get("inter", "0"))
            current.river_terminals = optional_int(kv.get("terminals"))
            current.nav_length = optional_float(kv.get("navLength"))
            current.terminals_per_nav_length = optional_float(kv.get("terminalsPerNavLength"))
            current.river_corridors_per_nav_length = optional_float(kv.get("riverCorridorsPerNavLength"))
        elif match := TRADE_RE.match(line):
            kind, vessel = match.groups()
            corridors = int(kv["corridors"])
            inter = int(kv.get("inter", "0"))
            ports = optional_int(kv.get("candidatePorts"))
            stopovers = optional_int(kv.get("stopovers"))
            coast_length = optional_float(kv.get("coastLength"))
            if coast_length is not None:
                current.coast_length = coast_length
            ocean_area = optional_float(kv.get("oceanArea"))
            if ocean_area is not None:
                current.ocean_area = ocean_area
            if kind == "coastalTrade":
                current.coastal[vessel] = corridors
                current.coastal_inter[vessel] = inter
                set_optional(current.coastal_ports, vessel, ports)
                set_optional(current.coastal_stopovers, vessel, stopovers)
                set_optional(current.ports_per_coast_length, vessel, optional_float(kv.get("portsPerCoastLength")))
                set_optional(
                    current.coastal_corridors_per_coast_length,
                    vessel,
                    optional_float(kv.get("coastalCorridorsPerCoastLength")),
                )
            else:
                current.ocean[vessel] = corridors
                current.ocean_inter[vessel] = inter
                set_optional(current.ocean_ports, vessel, ports)
                set_optional(current.ocean_stopovers, vessel, stopovers)
                set_optional(current.stopovers_per_ocean_area, vessel, optional_float(kv.get("stopoversPerOceanArea")))
                set_optional(
                    current.ocean_corridors_per_ocean_area,
                    vessel,
                    optional_float(kv.get("oceanCorridorsPerOceanArea")),
                )
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


def value(value: int | float | None) -> str:
    if value is None:
        return "-"
    if isinstance(value, float):
        # 6 significant digits: densities must resolve changes far smaller than
        # the +-1 quantization that makes raw counts unusable across meshes.
        return f"{value:.6g}"
    return str(value)


def print_markdown(summaries: list[SeedSummary], vessel: str, ancestry_limit: int) -> None:
    print(
        "| seed | proto | polities | land | river | river_inter | "
        f"coastal_{vessel} | coastal_inter | ocean_{vessel} | ocean_inter | "
        "navLen | term/navLen | riverCor/navLen | coastLen | ports/coastLen | "
        "coastCor/coastLen | oceanArea | stops/oceanArea | oceanCor/oceanArea | "
        "ancestry | stances |"
    )
    print(
        "|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|"
    )
    for summary in summaries:
        print(
            f"| {summary.seed} | {value(summary.proto)} | {value(summary.polities)} | "
            f"{value(summary.land)} | {value(summary.river)} | {value(summary.river_inter)} | "
            f"{value(summary.coastal.get(vessel))} | {value(summary.coastal_inter.get(vessel))} | "
            f"{value(summary.ocean.get(vessel))} | {value(summary.ocean_inter.get(vessel))} | "
            f"{value(summary.nav_length)} | {value(summary.terminals_per_nav_length)} | "
            f"{value(summary.river_corridors_per_nav_length)} | {value(summary.coast_length)} | "
            f"{value(summary.ports_per_coast_length.get(vessel))} | "
            f"{value(summary.coastal_corridors_per_coast_length.get(vessel))} | "
            f"{value(summary.ocean_area)} | {value(summary.stopovers_per_ocean_area.get(vessel))} | "
            f"{value(summary.ocean_corridors_per_ocean_area.get(vessel))} | "
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
                "river_terminals",
                "nav_length",
                "terminals_per_nav_length",
                "river_corridors_per_nav_length",
                "coast_length",
                f"coastal_ports_{vessel}",
                f"ports_per_coast_length_{vessel}",
                f"coastal_corridors_per_coast_length_{vessel}",
                "ocean_area",
                f"ocean_stopovers_{vessel}",
                f"stopovers_per_ocean_area_{vessel}",
                f"ocean_corridors_per_ocean_area_{vessel}",
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
                    value(summary.river_terminals),
                    value(summary.nav_length),
                    value(summary.terminals_per_nav_length),
                    value(summary.river_corridors_per_nav_length),
                    value(summary.coast_length),
                    value(summary.coastal_ports.get(vessel)),
                    value(summary.ports_per_coast_length.get(vessel)),
                    value(summary.coastal_corridors_per_coast_length.get(vessel)),
                    value(summary.ocean_area),
                    value(summary.ocean_stopovers.get(vessel)),
                    value(summary.stopovers_per_ocean_area.get(vessel)),
                    value(summary.ocean_corridors_per_ocean_area.get(vessel)),
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
