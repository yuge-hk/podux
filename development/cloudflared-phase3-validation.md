# Cloudflared Phase 3 validation

## Scope and publication status

Validation date: 2026-10-23. Base: `1136797a36a2f1186055278351f55e28b907d6d5`.
Test branch: `feat/local-cloudflare-tunnels`.
Personal Fork: https://github.com/yuge-hk/podux ; upstream: https://github.com/luckjiawei/podux (read only).
Authenticated preflight confirmed account `yuge-hk`, direct fork relationship, ADMIN/write permission and Actions enabled.
The existing `origin` still points to upstream. No commit or push has been made; first test commit/push awaits explicit user authorization.
GitHub Actions run URL: **Blocked — not yet triggered**. Test commit SHA: **Blocked — not yet created**.

`Passed` means actually executed successfully; `Failed` means actually executed unsuccessfully; `Skipped` means not executed; `Blocked` means a required condition is missing. Cross compilation is never reported as native execution.

## Native Actions plan

Workflow: [cloudflared-ci.yml](../.github/workflows/cloudflared-ci.yml). Push to the test branch triggers the first run; workflow_dispatch requires the workflow on the default branch. Permissions are contents:read, with no deployment or publishing jobs. The Docker/release workflow is unchanged by this phase.

| Environment | Native Actions result | Race result |
| --- | --- | --- |
| ubuntu-24.04 / amd64 | Blocked: awaiting authorized push | Blocked |
| ubuntu-24.04-arm / arm64 | Blocked: awaiting authorized push | Blocked |
| windows-2025 / amd64 | Blocked: awaiting authorized push | Blocked |
| macos-15 / arm64 | Blocked: awaiting authorized push | Blocked |

The workflow first runs frozen pnpm install, TypeScript, Vitest, targeted ESLint and Vite build, then provides the built pb_public to native Go jobs. Go comes from go.mod. Each job asserts host, target and hardware architecture, builds and vets, tests the runner/credentials/API/migrations, runs race with CGO when a compiler is present, and records an explicit reason if race cannot run. Failure is not hidden with continue-on-error. Logs and actual outcomes are uploaded even on failure.

Official cloudflared is pinned to **2026.10.3**. Four platform assets and SHA-256 digests from official release metadata are recorded in `build/ci/cloudflared-checksums.json`; download verification precedes use. The official-binary test uses a synthetic invalid Token for local rejection, never a production credential or real Edge connectivity claim.

## Actual local Linux results

| Verification | Result and scope |
| --- | --- |
| Go vet/build/default offline tests | Passed; explicit FRPS and official binary tests are separate opt-in runs |
| Offline race suite | Passed; does not include opt-in live FRPS integration |
| Official binary TestOfficialBinary | Passed, Linux amd64, pinned official binary; [evidence](phase3-evidence/official-linux.json) |
| Native runner start/stop/restart, retries, temporary credentials | Passed in local backend tests; no claim about other operating systems |
| API auth, invalid configuration, oversized bodies | Passed |
| Incremental migration, rollback preserving FRPC record, paired key/database restore | Passed |
| Local FRPS TCP/UDP, reconnect, FRPC + connector coexistence and shutdown | Passed without race, using unchanged FRP v0.68.0 and loopback-only temporary FRPS |
| Same FRPC integration with race detector | **Failed**, existing FRP data races; [raw evidence](phase3-evidence/frpc-linux-race.json) |
| Frontend tests | Passed: 68 Vitest tests; frozen install, tsc, targeted ESLint, Vite build and documentation build passed |
| Browser regression | Passed: 18 general + 9 real fixture lifecycle checks; [general](phase3-evidence/browser/browser-checks.json), [lifecycle](phase3-evidence/browser/lifecycle-browser-checks.json) |
| Workflow lint and helper tests | Passed: actionlint 1.7.7; 3 Python helper unit tests |
| Whole existing frontend lint | Failed: 26 errors and 3 warnings in existing code; targeted changed frontend files pass |
| Real Cloudflare Edge / two authorized tunnels | Blocked: no authorized disposable test credentials |
| 1–2 hour soak | Skipped |

The FRPC race failure includes FRP xlog prefix rendering/mutation and Service.stop versus keepControllerWorking. No FRPC source or FRP dependency was changed to make this pass. CI deliberately retains the failing race integration. Existing FRPC reload temporarily interrupts UDP; the regression checks eventual recovery, not uninterrupted UDP service. This remains a merge blocker requiring a separately authorized dependency/runtime fix or upstream resolution.

## Resource and cleanup evidence

[Enhanced Linux resource evidence](phase3-evidence/resources-linux-with-child-stats.json) records 12 start/restart/stop cycles using the test-only connector, not real cloudflared Edge traffic. Podux RSS was 37,324 KiB idle, 37,520 with one connector and 37,668 with two. Each connector used approximately 8.6–9.1 MiB RSS. Podux FD count returned from 24 to its baseline 18; child count returned to zero. Median start/restart/stop were approximately 0.256s / 0.257s / 0.0013s. Normal shutdown was approximately 0.0032s, no credential directories remained, and SQLite reopen quick_check returned ok. Short samples cannot establish long-term stability.

The first disposable first-run experiment observed an unrelated PocketBase installer xdg-open zombie; its failure evidence is retained in `resources-first-run-installer-failure.json`. Subsequent probes preseed the disposable superuser to avoid launching the desktop installer. This does not establish that the existing installer process behavior was fixed. A resource sampler was also hardened to represent a child without VmRSS explicitly instead of crashing on a transient/zombie process.

## Dependency security

Security is **not clean**. Govulncheck identifies 54 reachable advisory IDs both in the baseline and current tree; no new advisory IDs were introduced. Reachability is static evidence, not proof of exploitability. See [Go audit summary](phase3-evidence/go-audit-summary.json). Govulncheck JSON mode exit code zero is not treated as a clean scan.

The initial new Vitest 3 dependency chain added advisories, including critical findings. Upgrading only the new test dependency to Vitest 4.1.12 removed the added advisories. Current frontend audit has 67 baseline findings (42 high, 23 moderate, 2 low, zero critical), the same advisory set as the original baseline. See [frontend comparison](phase3-evidence/frontend-audit-summary.json). Existing FRP, PocketBase, Go and frontend dependencies require separate remediation review; they were not broadly upgraded in this phase.

## Remaining work and review readiness

1. Obtain explicit first test commit/push approval for the verified personal Fork and test branch.
2. Commit only reviewed source, CI, documentation and safe evidence; push to the Fork, record actual SHA and Actions URL here.
3. Inspect every native job, retain failure logs and fix feature-specific compatibility bugs without disabling failing tests.
4. Resolve or explicitly review the existing FRP race blocker and baseline security findings before merge readiness.
5. Authorized real Edge testing and long soak remain unverified. Platform-specific orphan-process guarantees require native execution evidence.

No upstream PR, upstream push, release, Docker publication or production change has been performed.
