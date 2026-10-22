"""Build the exact FRP version already used by Podux, in an isolated module copy.
FRP's module contains a replace directive, so go install module@version is not
suitable. Building this copy honors upstream's lockfiles without changing Podux.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

p = argparse.ArgumentParser()
p.add_argument("--output", required=True)
args = p.parse_args()
module = json.loads(subprocess.check_output(["go", "list", "-m", "-json", "github.com/fatedier/frp"], text=True))
if module["Version"] != "v0.68.0" or module.get("Replace"):
    raise SystemExit("Unexpected FRP version/replacement; review instead of silently upgrading")
output = Path(args.output).resolve()
if os.name == "nt":
    output = output.with_suffix(".exe")
output.parent.mkdir(parents=True, exist_ok=True)
env = os.environ.copy()
env["GOTOOLCHAIN"] = subprocess.check_output(["go", "env", "GOVERSION"], text=True).strip()
with tempfile.TemporaryDirectory(prefix="podux-frps-source-") as tmp:
    source = Path(tmp) / "frp"
    shutil.copytree(module["Dir"], source)
    subprocess.run(["go", "build", "-mod=readonly", "-tags=noweb", "-o", str(output), "./cmd/frps"], cwd=source, env=env, check=True)
print(json.dumps({"version": module["Version"], "binary": str(output), "sha256": hashlib.sha256(output.read_bytes()).hexdigest()}))
