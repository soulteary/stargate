#!/usr/bin/env python3
"""Build first, then alternate two revisions against the same real Redis."""
import argparse
import json
import os
from pathlib import Path
import platform
import random
import re
import statistics
import socket
import subprocess
import tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--baseline", required=True, help="full commit SHA")
parser.add_argument("--output", required=True, type=Path)
parser.add_argument("--rounds", type=int, default=12)
parser.add_argument("--operations", type=int, default=5000)
args = parser.parse_args()
if not re.fullmatch(r"[0-9a-f]{40}", args.baseline) or args.rounds < 1 or args.operations < 1:
    parser.error("require a full baseline SHA and positive rounds/operations")
if not os.environ.get("STARGATE_TEST_REDIS_ADDR"):
    parser.error("STARGATE_TEST_REDIS_ADDR must identify an isolated test Redis")
root = Path(__file__).resolve().parents[2]
out = args.output.resolve()
out.mkdir(parents=True, exist_ok=True)


def command(argv, cwd=root):
    return subprocess.check_output(argv, cwd=cwd, text=True, stderr=subprocess.STDOUT)


def redis_info(section):
    host, _, port = os.environ["STARGATE_TEST_REDIS_ADDR"].rpartition(":")
    with socket.create_connection((host.strip("[]"), int(port)), timeout=10) as connection:
        connection.sendall(f"*2\r\n$4\r\nINFO\r\n${len(section)}\r\n{section}\r\n".encode())
        stream = connection.makefile("rb")
        length = int(stream.readline()[1:])
        payload = stream.read(length).decode()
    return dict(line.split(":", 1) for line in payload.splitlines() if line and not line.startswith("#"))


metadata = {
    "redis": redis_info("server")["redis_version"],
    "baseline": args.baseline,
    "candidate": command(["git", "rev-parse", "HEAD"]).strip(),
    "candidate_dirty": bool(command(["git", "status", "--porcelain"]).strip()),
    "go": command(["go", "version"]).strip(),
    "platform": platform.platform(),
    "cpus": os.cpu_count(),
    "memory": next(line for line in Path("/proc/meminfo").read_text().splitlines() if line.startswith("MemTotal:")),
    "gomaxprocs": os.environ.get("GOMAXPROCS", "Go default"),
    "rounds": args.rounds,
    "operations_per_case": args.operations,
    "order": "baseline/candidate on odd rounds; candidate/baseline on even rounds",
}
records = []
pattern = re.compile(r"BenchmarkRedisRateLimit/cold=(true|false)-\d+\s+(\d+)\s+([\d.]+) ns/op\s+(\d+) B/op\s+(\d+) allocs/op")
with tempfile.TemporaryDirectory(prefix="stargate-redis-bench-") as temporary:
    temporary = Path(temporary)
    baseline = temporary / "baseline"
    command(["git", "worktree", "add", "--detach", str(baseline), args.baseline])
    try:
        binaries = {}
        for name, source in (("baseline", baseline), ("candidate", root)):
            binary = temporary / (name + ".test")
            (out / (name + "-build.log")).write_text(command(["go", "test", "-c", "-o", str(binary), "./src/internal/handlers"], source))
            binaries[name] = binary
        # Neither compilation nor an unrelated test suite runs in the timed phase.
        for round_number in range(1, args.rounds + 1):
            order = ("baseline", "candidate") if round_number % 2 else ("candidate", "baseline")
            for name in order:
                load_before = Path("/proc/loadavg").read_text().strip()
                commands_before = redis_info("commandstats")
                stdout = command([str(binaries[name]), "-test.run=^$", "-test.bench=^BenchmarkRedisRateLimit$", f"-test.benchtime={args.operations}x", "-test.count=1"])
                (out / f"{round_number:02d}-{name}.log").write_text(stdout)
                commands_after = redis_info("commandstats")
                command_deltas = {}
                for name_key, value in commands_after.items():
                    current = dict(field.split("=") for field in value.split(","))
                    previous = dict(field.split("=") for field in commands_before.get(name_key, "calls=0,usec=0").split(","))
                    calls = int(current["calls"]) - int(previous["calls"])
                    if calls and name_key != "cmdstat_info":
                        command_deltas[name_key] = {"calls": calls, "execution_usec": int(current["usec"]) - int(previous["usec"])}
                matches = pattern.findall(stdout)
                if len(matches) != 2 or {row[0] for row in matches} != {"true", "false"}:
                    raise RuntimeError("expected both real Redis benchmark cases; skipping is not a measurement")
                for cold, operations, ns, size, allocations in matches:
                    if int(operations) != args.operations:
                        raise RuntimeError("benchmark operation count changed")
                    records.append({"round": round_number, "implementation": name, "cold": cold == "true", "ns_per_op": float(ns), "bytes_per_op": int(size), "allocations_per_op": int(allocations), "server_commands_including_setup": command_deltas, "load_before": load_before, "load_after": Path("/proc/loadavg").read_text().strip()})
        # Profiles are separate runs; profiling never changes the timed samples.
        for name, binary in binaries.items():
            profile = out / (name + ".allocs.pprof")
            stdout = command([str(binary), "-test.run=^$", "-test.bench=^BenchmarkRedisRateLimit$/cold=false$", "-test.benchtime=5000x", "-test.memprofilerate=1", "-test.memprofile=" + str(profile)])
            (out / (name + "-profile.log")).write_text(stdout)
            (out / (name + "-allocations.txt")).write_text(command(["go", "tool", "pprof", "-top", "-alloc_objects", "-focus=redisRateLimitStore.*Incr", str(binary), str(profile)]))
    finally:
        command(["git", "worktree", "remove", str(baseline)])
summary = []
for cold in (False, True):
    before = [r for r in records if r["cold"] == cold and r["implementation"] == "baseline"]
    after = [r for r in records if r["cold"] == cold and r["implementation"] == "candidate"]
    ratios = [b["ns_per_op"] / a["ns_per_op"] for a, b in zip(before, after)]
    rng = random.Random(0)
    bootstrap = sorted(statistics.median(rng.choices(ratios, k=len(ratios))) for _ in range(10000))
    row = {"cold": cold, "paired_latency_ratio_median": statistics.median(ratios), "paired_latency_ratio_bootstrap_95pct": [bootstrap[249], bootstrap[9749]], "candidate_slower_rounds": sum(r > 1 for r in ratios)}
    for label, values in (("baseline", before), ("candidate", after)):
        row[label] = {key: statistics.median(r[key] for r in values) for key in ("ns_per_op", "bytes_per_op", "allocations_per_op")}
        row[label]["serial_ops_per_second"] = 1e9 / row[label]["ns_per_op"]
    summary.append(row)
(out / "results.json").write_text(json.dumps({"environment": metadata, "summary": summary, "samples": records}, indent=2) + "\n")
print(json.dumps(summary, indent=2))
