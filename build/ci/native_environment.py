"""Fail rather than silently running a cross-compiled target under emulation."""
import json
import os
import platform
import subprocess

env = json.loads(subprocess.check_output(["go", "env", "-json", "GOOS", "GOARCH", "GOHOSTOS", "GOHOSTARCH", "GOVERSION"], text=True))
expected = (os.environ["EXPECTED_OS"], os.environ["EXPECTED_ARCH"])
for pair in [(env["GOOS"], env["GOARCH"]), (env["GOHOSTOS"], env["GOHOSTARCH"])]:
    if pair != expected:
        raise SystemExit(f"Native architecture mismatch: expected {expected}, got {pair}")
hardware = platform.machine().lower()
if hardware not in ({"amd64", "x86_64"} if expected[1] == "amd64" else {"arm64", "aarch64"}):
    raise SystemExit("Runner hardware architecture mismatch")
print(json.dumps({"go": env, "hardware": hardware, "platform": platform.platform(), "repository": os.getenv("GITHUB_REPOSITORY", "local"), "sha": os.getenv("GITHUB_SHA", "uncommitted"), "run": os.getenv("GITHUB_RUN_ID", "local"), "runner": os.getenv("RUNNER_NAME", "local")}, indent=2))
