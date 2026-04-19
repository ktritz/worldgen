#!/usr/bin/env python3
"""Serial review_planets cache refill helper.

Runs one seed at a time, writes durable per-seed artifacts, and records a small
summary table so cache refill progress is easy to inspect after long runs or
reboots.
"""

from __future__ import annotations

import argparse
import csv
import json
import os
import re
import subprocess
import tempfile
import time
from pathlib import Path


CACHE_HIT_RE = re.compile(r"^\s+([a-z]+) cache hit:")
TRADE_RE = re.compile(r"multimodalTrade: .*?score=([0-9.]+) volume=([0-9.]+)")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--level", type=int, default=6)
    parser.add_argument("--seeds", default="42,91,123")
    parser.add_argument("--out-dir", default="output/review_planets/sweeps/cache_refill")
    parser.add_argument("--force", action="store_true")
    parser.add_argument("--cache", choices=("true", "false"), default="true")
    parser.add_argument("--go-bin", default="/usr/local/go/bin/go")
    parser.add_argument("--gocache", default="/tmp/go-build-cache")
    return parser.parse_args()


def parse_seed_list(value: str) -> list[int]:
    return [int(part.strip()) for part in value.split(",") if part.strip()]


def seed_paths(out_dir: Path, seed: int) -> dict[str, Path]:
    prefix = out_dir / f"seed_{seed}"
    return {
        "txt": prefix.with_suffix(".txt"),
        "err": prefix.with_suffix(".err"),
        "time": prefix.with_suffix(".time"),
    }


def output_complete(path: Path) -> bool:
    if not path.exists():
        return False
    text = path.read_text(encoding="utf-8", errors="replace")
    return "multimodalTrade:" in text


def write_text_atomic(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", dir=path.parent, delete=False, encoding="utf-8") as handle:
        handle.write(text)
        handle.flush()
        os.fsync(handle.fileno())
        temp_path = Path(handle.name)
    temp_path.replace(path)


def parse_cache_hits(text: str) -> str:
    hits: list[str] = []
    for line in text.splitlines():
        match = CACHE_HIT_RE.search(line)
        if match:
            hits.append(match.group(1))
    return ",".join(hits)


def parse_trade_metrics(text: str) -> tuple[str, str]:
    match = TRADE_RE.search(text)
    if not match:
        return "", ""
    return match.group(1), match.group(2)


def run_seed(args: argparse.Namespace, out_dir: Path, seed: int) -> dict[str, str]:
    paths = seed_paths(out_dir, seed)
    if not args.force and output_complete(paths["txt"]) and paths["time"].exists():
        text = paths["txt"].read_text(encoding="utf-8", errors="replace")
        timing = paths["time"].read_text(encoding="utf-8", errors="replace").strip().split("=", 1)[-1]
        score, volume = parse_trade_metrics(text)
        return {
            "seed": str(seed),
            "status": "cached",
            "elapsed_sec": timing,
            "cache_hits": parse_cache_hits(text),
            "trade_score": score,
            "trade_volume": volume,
        }

    cmd = [
        args.go_bin,
        "run",
        "./cmd/review_planets",
        "-render=false",
        f"-cache={args.cache}",
        "-level",
        str(args.level),
        "-seeds",
        str(seed),
    ]
    env = os.environ.copy()
    env["GOCACHE"] = args.gocache

    start = time.monotonic()
    result = subprocess.run(
        cmd,
        cwd=Path(__file__).resolve().parents[1],
        env=env,
        text=True,
        capture_output=True,
    )
    elapsed = time.monotonic() - start

    write_text_atomic(paths["txt"], result.stdout)
    write_text_atomic(paths["err"], result.stderr)
    write_text_atomic(paths["time"], f"elapsed_sec={elapsed:.2f}\n")

    score, volume = parse_trade_metrics(result.stdout)
    status = "ok" if result.returncode == 0 else f"failed:{result.returncode}"
    return {
        "seed": str(seed),
        "status": status,
        "elapsed_sec": f"{elapsed:.2f}",
        "cache_hits": parse_cache_hits(result.stdout),
        "trade_score": score,
        "trade_volume": volume,
    }


def write_summary(out_dir: Path, rows: list[dict[str, str]]) -> None:
    path = out_dir / "summary.tsv"
    fieldnames = ["seed", "status", "elapsed_sec", "cache_hits", "trade_score", "trade_volume"]
    with tempfile.NamedTemporaryFile("w", dir=out_dir, delete=False, encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames, delimiter="\t")
        writer.writeheader()
        writer.writerows(rows)
        handle.flush()
        os.fsync(handle.fileno())
        temp_path = Path(handle.name)
    temp_path.replace(path)


def write_progress(
    out_dir: Path,
    *,
    level: int,
    seeds: list[int],
    rows: list[dict[str, str]],
    status: str,
    active_seed: int | None,
    started_at_epoch: float,
) -> None:
    completed = len(rows)
    last_completed = rows[-1] if rows else None
    payload = {
        "status": status,
        "level": level,
        "total_seeds": len(seeds),
        "completed_seeds": completed,
        "pending_seeds": seeds[completed:] if active_seed is None else [seed for seed in seeds if str(seed) not in {row["seed"] for row in rows} and seed != active_seed],
        "active_seed": active_seed,
        "started_at_epoch": started_at_epoch,
        "updated_at_epoch": time.time(),
        "last_completed_seed": None if last_completed is None else int(last_completed["seed"]),
        "last_completed_elapsed_sec": None if last_completed is None else last_completed["elapsed_sec"],
        "rows": rows,
    }
    write_text_atomic(out_dir / "progress.json", json.dumps(payload, indent=2, sort_keys=True) + "\n")


def main() -> int:
    args = parse_args()
    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    seeds = parse_seed_list(args.seeds)
    started_at_epoch = time.time()
    rows: list[dict[str, str]] = []
    write_progress(
        out_dir,
        level=args.level,
        seeds=seeds,
        rows=rows,
        status="starting",
        active_seed=None,
        started_at_epoch=started_at_epoch,
    )
    for seed in seeds:
        write_progress(
            out_dir,
            level=args.level,
            seeds=seeds,
            rows=rows,
            status="running",
            active_seed=seed,
            started_at_epoch=started_at_epoch,
        )
        row = run_seed(args, out_dir, seed)
        rows.append(row)
        print(
            f"seed={row['seed']} status={row['status']} elapsed={row['elapsed_sec']} "
            f"cache_hits={row['cache_hits']} trade={row['trade_score']}/{row['trade_volume']}"
        )
        write_summary(out_dir, rows)
        write_progress(
            out_dir,
            level=args.level,
            seeds=seeds,
            rows=rows,
            status="running",
            active_seed=None,
            started_at_epoch=started_at_epoch,
        )
        if not row["status"].startswith("ok") and row["status"] != "cached":
            write_progress(
                out_dir,
                level=args.level,
                seeds=seeds,
                rows=rows,
                status="failed",
                active_seed=None,
                started_at_epoch=started_at_epoch,
            )
            return 1
    write_progress(
        out_dir,
        level=args.level,
        seeds=seeds,
        rows=rows,
        status="complete",
        active_seed=None,
        started_at_epoch=started_at_epoch,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
