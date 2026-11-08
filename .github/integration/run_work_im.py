"""Run only local synthetic work-IM regression; never load deployment config.

Pass explicit source checkouts containing work_im_pilot_test.go in each service.
The provider endpoint is a loopback fixture owned by the Herald test process.
"""
from pathlib import Path
import argparse, hashlib, json, os, subprocess, time


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--herald", type=Path, required=True)
    p.add_argument("--stargate", type=Path, required=True)
    p.add_argument("--go", required=True)
    p.add_argument("--go-cache", type=Path)
    p.add_argument("--output", type=Path, required=True)
    a = p.parse_args()
    herald, stargate = a.herald.resolve(strict=True), a.stargate.resolve(strict=True)
    go = Path(a.go).resolve(strict=True)
    out = a.output.absolute()
    out.mkdir(mode=0o700)
    allowed = ("PATH", "HOME", "TMPDIR", "GOROOT", "GOPATH", "GOMODCACHE")
    env = {k: os.environ[k] for k in allowed if k in os.environ}
    env.update(GOTOOLCHAIN="local", GOPROXY="off", GOCACHE=str(a.go_cache.absolute() if a.go_cache else out / "go-cache"))
    manifest = {"scope": "local-synthetic-Stargate-Herald-mock-provider",
                "source_sha256": {}, "steps": [], "passed": False,
                "live_IM_or_remote_or_deployment_configuration_used": False}
    for label, source, paths in (
        ("herald", herald, ("go.mod", "go.sum", "internal/destination/destination.go", "internal/destination/destination_test.go", "internal/handlers/work_im_pilot_test.go")),
        ("stargate", stargate, ("go.mod", "go.sum", "src/internal/handlers/work_im_bridge_integration_test.go")),
    ):
        manifest["source_sha256"][label] = {n: digest(source / n) for n in paths}
    binary = out / "herald-bridge.test"

    def verify_sources():
        for label, source in (("herald", herald), ("stargate", stargate)):
            for name, expected in manifest["source_sha256"][label].items():
                assert digest(source / name) == expected, "source changed during local regression"

    def step(name, args, cwd, extra=None):
        verify_sources()
        start = time.monotonic()
        with (out / (name + ".stdout")).open("xb") as stdout, (out / (name + ".stderr")).open("xb") as stderr:
            r = subprocess.run([str(go), *args], cwd=cwd, env={**env, **(extra or {})}, stdout=stdout, stderr=stderr, timeout=300)
        result = {"name": name, "go_args": args, "returncode": r.returncode,
                  "elapsed_seconds": round(time.monotonic() - start, 3),
                  "stdout_sha256": digest(out / (name + ".stdout")),
                  "stderr_sha256": digest(out / (name + ".stderr"))}
        if "-json" in args:
            events = [json.loads(line) for line in (out / (name + ".stdout")).read_text().splitlines()]
            result["tests_passed"] = sum(e.get("Action") == "pass" and "Test" in e for e in events)
            result["tests_failed"] = sum(e.get("Action") == "fail" and "Test" in e for e in events)
            result["tests_skipped"] = [e.get("Test") for e in events if e.get("Action") == "skip"]
            if name == "actual-bridge":
                result["expected_bridge_passed"] = any(e.get("Action") == "pass" and e.get("Test") == "TestWorkIMCurrentPair" for e in events)
        manifest["steps"].append(result)
        verify_sources()
        (out / "result.json").write_text(json.dumps(manifest, indent=2) + "\n")
        assert r.returncode == 0, name + " failed; retained original logs"
        if name == "actual-bridge":
            assert not result["tests_skipped"] and result["expected_bridge_passed"]

    step("go-version", ["version"], herald)
    step("herald-components", ["test", "./internal/destination", "./internal/handlers", "./internal/idempotency", "./internal/ratelimit", "-count=1", "-json"], herald)
    step("build-herald-fixture", ["test", "-c", "./internal/handlers", "-o", str(binary)], herald)
    manifest["herald_fixture_binary_sha256"] = digest(binary)
    step("actual-bridge", ["test", "./src/internal/handlers", "-run", "^TestWorkIMCurrentPair", "-count=1", "-json"], stargate, {"WORK_IM_HERALD_TEST_BINARY": str(binary)})
    manifest["passed"] = True
    (out / "result.json").write_text(json.dumps(manifest, indent=2) + "\n")
    print(json.dumps({"passed": True, "record": "result.json", "real_IM_delivery_proved": False}))


if __name__ == "__main__":
    main()
