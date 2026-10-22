"""Download only the pinned official release matching the native host.
Hashes originate from the release's official GitHub SHA-256 asset metadata.
No credentials, shell, package installation or Tunnel connection is involved.
"""
import argparse
import hashlib
import io
import json
import os
import platform
from pathlib import Path
import tarfile
import urllib.request


def asset_for(system, machine):
    arch = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}.get(machine.lower())
    name = {"Linux": "linux", "Windows": "windows", "Darwin": "darwin"}.get(system)
    if not arch or not name:
        raise ValueError("Unsupported native platform")
    suffix = ".exe" if name == "windows" else ".tgz" if name == "darwin" else ""
    return f"cloudflared-{name}-{arch}{suffix}"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    manifest = json.loads(Path(__file__).with_name("cloudflared-checksums.json").read_text())
    asset = asset_for(platform.system(), platform.machine())
    expected = manifest["assets"][asset]  # No unpinned asset fallback.
    url = f"https://github.com/cloudflare/cloudflared/releases/download/{manifest['version']}/{asset}"
    request = urllib.request.Request(url, headers={"User-Agent": "podux-cloudflared-ci"})
    with urllib.request.urlopen(request, timeout=120) as response:
        data = response.read(100 * 1024 * 1024 + 1)
    if len(data) > 100 * 1024 * 1024 or hashlib.sha256(data).hexdigest() != expected:
        raise ValueError("Official binary SHA-256 mismatch")
    if asset.endswith(".tgz"):
        with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as archive:
            members = [m for m in archive.getmembers() if m.name == "cloudflared" and m.isfile()]
            if len(members) != 1:
                raise ValueError("Unexpected official archive layout")
            with archive.extractfile(members[0]) as source:
                data = source.read(100 * 1024 * 1024 + 1)
    output = Path(args.output).resolve()
    if platform.system() == "Windows":
        output = output.with_suffix(".exe")
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_bytes(data)
    output.chmod(0o755)
    print(json.dumps({"version": manifest["version"], "asset": asset, "sha256": expected, "binary": str(output)}))
    # Dedicated path file, not an environment dump. Only the official-binary test
    # step opts in; regular/race tests remain deterministic and offline.
    Path(str(output) + ".path").write_text(str(output))


if __name__ == "__main__":
    main()
