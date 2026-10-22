# Cloudflared Phase 3 validation

## Scope and publication status

Validation date: 2026-10-23. Base: `1136797a36a2f1186055278351f55e28b907d6d5`.
Test branch: `feat/local-cloudflare-tunnels`.
Personal Fork: https://github.com/yuge-hk/podux ; upstream: https://github.com/luckjiawei/podux (read only).
Authenticated preflight confirmed account `yuge-hk`, direct fork relationship, ADMIN/write permission and Actions enabled.
The existing `origin` still points to upstream. User explicitly authorized test commits/pushes. First test commit was pushed only to the personal Fork feature branch; upstream remains unchanged.
First GitHub Actions run: https://github.com/yuge-hk/podux/actions/runs/39844432704

Test commit SHA: `cb2f4aeeeb8420b2f496b9f4d098c8717d6e4fce`. First run completed: overall **Failed**. Frontend Passed; macOS job Passed; Linux amd64/arm64 and Windows jobs Failed only at the opt-in FRPC race integration. Evidence: [run metadata](phase3-evidence/actions-39844432704/run.json). No cross compilation was used for these results.

`Passed` means actually executed successfully; `Failed` means actually executed unsuccessfully; `Skipped` means not executed; `Blocked` means a required condition is missing. Cross compilation is never reported as native execution.

## Native Actions plan

Workflow: [cloudflared-ci.yml](../.github/workflows/cloudflared-ci.yml). Push to the test branch triggers the first run; workflow_dispatch requires the workflow on the default branch. Permissions are contents:read, with no deployment or publishing jobs. The Docker/release workflow is unchanged by this phase.

| Environment | Native build/unit tests | Offline race suite | Official binary | FRPC with race | Overall job |
| --- | --- | --- | --- | --- | --- |
| ubuntu-24.04 / amd64 | Passed | Passed | Passed | Failed | Failed |
| ubuntu-24.04-arm / arm64 | Passed | Passed | Passed | Failed | Failed |
| windows-2025 / amd64 | Passed | Passed | Passed | Failed | Failed |
| macos-15 / arm64 | Passed | Passed | Passed | Passed | Passed |

First-run evidence under `phase3-evidence/actions-39844432704/<os>-<arch>/` includes native environment, test counts, official binary checksum/provenance, actual outcomes and complete FRPC output. Offline suites explicitly skip the two opt-in tests; both are executed in separate mandatory steps. All four race toolchains ran; none was skipped. First-run FRPC combined functional/race assertions in one execution; follow-up workflow separates non-race functional and race steps, retaining the race failure as a job failure. FRP races are scheduling-dependent: one passing macOS run does not establish the dependency is race-free.

The workflow first runs frozen pnpm install, TypeScript, Vitest, targeted ESLint and Vite build, then provides the built pb_public to native Go jobs. Go comes from go.mod. Each job asserts host, target and hardware architecture, builds and vets, tests the runner/credentials/API/migrations, runs race with CGO when a compiler is present, and records an explicit reason if race cannot run. Failure is not hidden with continue-on-error. Logs and actual outcomes are uploaded even on failure.

Official cloudflared is pinned to **2026.10.3**. Four platform assets and SHA-256 digests from official release metadata are recorded in `build/ci/cloudflared-checksums.json`; download verification precedes use. The official-binary test uses a synthetic invalid Token for local rejection, never a production credential or real Edge connectivity claim.

## Follow-up native execution

Run: https://github.com/yuge-hk/podux/actions/runs/39845439391 (completed; **Failed**)

Test commit: `c2e6e7d5328add30eec04177a6630cd3e7e42454`.
This changes test presentation only: FRPC functional regression executes first without race, followed by the same regression with race. Both remain required; a race failure still fails the job. No production code, FRPC runner or dependency was changed. The frontend passed again.

| Platform | Build/unit | Offline race | Official binary | FRPC functional | FRPC race | Overall job |
| --- | --- | --- | --- | --- | --- | --- |
| Linux amd64 | Passed | Passed | Passed | Passed | Failed | Failed |
| Linux arm64 | Passed | Passed | Passed | Passed | Failed | Failed |
| Windows amd64 | Passed | Passed | Passed | Passed | Failed | Failed |
| macOS arm64 | Passed | Passed | Passed | Passed | Failed | Failed |

Evidence is retained under `phase3-evidence/actions-39845439391/`. Full Go JSON logs remain in the corresponding run artifacts; checked-in evidence includes test counts/skips, environment, official binary provenance and complete FRPC functional/race output. The macOS follow-up was initially queued, then ran natively and completed. Overall second-run result: **Failed**, solely at the FRPC race steps on all four platforms.

### Process and credential conclusions

On completed native jobs, runner lifecycle tests actually exercise independent Start/Stop/Restart, exit/retry budget, forced-stop timeout, metrics conflicts and credential removal. FRPC functional logs separately confirm TCP/UDP recovery, coexistence, joint shutdown and port cleanup. The official cloudflared executable parses supported flags and rejects a synthetic invalid Token locally; this does not prove healthy Edge connection, connected metrics or graceful draining of live Cloudflare traffic. Those require authorized disposable credentials and remain Blocked.

Linux/macOS process-group termination and Windows CTRL_BREAK with forced-kill fallback must be distinguished. Successful normal shutdown tests do not prove arbitrary descendant cleanup after every parent crash on every OS. Linux-specific parent-death/symlink tests are compiled only for their supported platforms; lower Windows test count is not represented as equivalent Unix security coverage. No untested cross-platform orphan-process guarantee is made.

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

The FRPC race failure includes FRP xlog prefix rendering/mutation and Service.stop versus keepControllerWorking. No FRPC source or FRP dependency was changed to make this pass. CI deliberately retains the failing race integration. Existing FRPC reload temporarily interrupts UDP; the regression checks eventual recovery, not uninterrupted UDP service. Native ARM64 also reports UDPProxy.InWorkConn concurrent state access; Windows reports xlog mutation/read races. These are dependency frames, not Runner setup failures. This remains a merge blocker requiring a separately authorized dependency/runtime fix or upstream resolution.

## Resource and cleanup evidence

[Enhanced Linux resource evidence](phase3-evidence/resources-linux-with-child-stats.json) records 12 start/restart/stop cycles using the test-only connector, not real cloudflared Edge traffic. Podux RSS was 37,324 KiB idle, 37,520 with one connector and 37,668 with two. Each connector used approximately 8.6–9.1 MiB RSS. Podux FD count returned from 24 to its baseline 18; child count returned to zero. Median start/restart/stop were approximately 0.256s / 0.257s / 0.0013s. Normal shutdown was approximately 0.0032s, no credential directories remained, and SQLite reopen quick_check returned ok. Short samples cannot establish long-term stability.

The first disposable first-run experiment observed an unrelated PocketBase installer xdg-open zombie; its failure evidence is retained in `resources-first-run-installer-failure.json`. Subsequent probes preseed the disposable superuser to avoid launching the desktop installer. This does not establish that the existing installer process behavior was fixed. A resource sampler was also hardened to represent a child without VmRSS explicitly instead of crashing on a transient/zombie process.

## Dependency security

Security is **not clean**. Govulncheck identifies 54 reachable advisory IDs both in the baseline and current tree; no new advisory IDs were introduced. Reachability is static evidence, not proof of exploitability. See [Go audit summary](phase3-evidence/go-audit-summary.json). Govulncheck JSON mode exit code zero is not treated as a clean scan.

The initial new Vitest 3 dependency chain added advisories, including critical findings. Upgrading only the new test dependency to Vitest 4.1.12 removed the added advisories. Current frontend audit has 67 baseline findings (42 high, 23 moderate, 2 low, zero critical), the same advisory set as the original baseline. See [frontend comparison](phase3-evidence/frontend-audit-summary.json). Existing FRP, PocketBase, Go and frontend dependencies require separate remediation review; they were not broadly upgraded in this phase.

## Remaining work and review readiness

1. User authorization, first Fork push and four native executions are complete.
2. Follow-up separates FRPC functional/race results and records both; no test or assertion is disabled.
3. Retain native failures as review blockers; no platform is made green by dropping checks.
4. Resolve or explicitly review the existing FRP race blocker and baseline security findings before merge readiness.
5. Authorized real Edge testing and long soak remain unverified. Platform-specific orphan-process guarantees require native execution evidence.

No upstream PR, upstream push, release, Docker publication or production change has been performed.

## Final publication record

Both test commits were pushed only to `yuge-hk/podux:feat/local-cloudflare-tunnels`. The final evidence/documentation commit changes no executable code or workflow and uses `[skip ci]` solely to avoid a third duplicate run for recorded results; it does not change or disable any failing case. Test conclusions apply to the exact SHAs above, not an unexecuted code revision. Original FRPC source/models, go.mod/go.sum, Docker publishing and release configuration are unchanged from the base. The working copy retains the upstream remote only; pushes used the explicit personal Fork URL.

No Cloudflare feature-specific compatibility defect was observed in these two runs. No FRPC fix is included because changing FRPC or upgrading FRP is outside the authorized scope. Resolving that dependency race is required before describing the overall CI as Passed or the PR as merge-ready. Baseline vulnerability findings, real Edge validation and long soak limitations remain as recorded above.
