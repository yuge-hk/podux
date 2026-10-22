"""Offline checks for native release selection and pinned integrity metadata."""
import json
from pathlib import Path
import unittest
from download_cloudflared import asset_for

class DownloaderTests(unittest.TestCase):
    def test_native_assets(self):
        for system, machine, wanted in [("Linux", "x86_64", "cloudflared-linux-amd64"), ("Linux", "aarch64", "cloudflared-linux-arm64"), ("Windows", "AMD64", "cloudflared-windows-amd64.exe"), ("Darwin", "arm64", "cloudflared-darwin-arm64.tgz")]:
            with self.subTest(system=system, machine=machine):
                self.assertEqual(asset_for(system, machine), wanted)
    def test_unknown_arch_rejected(self):
        with self.assertRaises(ValueError):
            asset_for("Linux", "mips64")
    def test_all_pinned_assets_have_sha256(self):
        manifest = json.loads(Path(__file__).with_name("cloudflared-checksums.json").read_text())
        self.assertEqual(manifest["version"], "2026.10.3")
        self.assertEqual(len(manifest["assets"]), 4)
        for value in manifest["assets"].values():
            self.assertRegex(value, "^[0-9a-f]{64}$")

if __name__ == "__main__":
    unittest.main()
