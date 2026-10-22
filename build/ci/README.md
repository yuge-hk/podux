# Native Cloudflared validation

The standalone workflow builds the frontend once and runs native Go verification on Linux amd64/arm64, Windows amd64 and macOS arm64. It does not publish artifacts to production or release packages. See ../../development/cloudflared-phase3-validation.md for actual results and blockers.

Before a first test push, authenticate GitHub CLI locally, then run:

```sh
python3 build/ci/check_fork.py yuge-hk/podux --gh /tmp/podux-phase3-tools/gh
```

The helper is read only. It verifies personal ownership, direct upstream relationship, write permission and Actions enablement; it never authorizes a push. Obtain explicit user approval before creating the first test commit or pushing. Use only the personal Fork test branch, never upstream main/dev. A first workflow on a feature branch should be triggered by push, since workflow_dispatch requires default-branch availability.

After an authorized push, use `gh run list --repo yuge-hk/podux --branch feat/local-cloudflare-tunnels` and `gh run view --repo yuge-hk/podux RUN_ID`. Download artifacts and retain the actual commit SHA, run URL and failure evidence in the validation report. Do not replace failed tests with successful cross compilation or suppress race failures.

The pinned official binary manifest records SHA-256 metadata. The isolated FRPS builder uses the unchanged embedded module version and Go toolchain, with the upstream noweb tag, and never edits Podux's module dependencies. All FRPS endpoints are loopback-only. Opt-in official tests use synthetic invalid Tokens and do not claim real Edge connectivity.
