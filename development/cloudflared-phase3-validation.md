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

## Independent upstream-main FRPC race control (2026-10-23)


Date: 2026-10-23. The race failures below are preserved; no implementation or dependency fix is included.

## Sources and exact executions

- Upstream main: `luckjiawei/podux@1136797a36a2f1186055278351f55e28b907d6d5`, freshly fetched and checked with ls-remote.
- Independent baseline test SHA: `6ff9263f6242e5a95b4e127b844642364923a13e`.
- Personal Fork test branch: `yuge-hk/podux:test/frpc-race-baseline-20261023`.
- [Baseline Actions run](https://github.com/yuge-hk/podux/actions/runs/39847653720): completed, **Failed**.
- [Baseline workflow at tested SHA](https://github.com/yuge-hk/podux/blob/6ff9263f6242e5a95b4e127b844642364923a13e/.github/workflows/frpc-race-baseline.yml).
- Feature test SHA: `c2e6e7d5328add30eec04177a6630cd3e7e42454`.
- [Feature Actions run](https://github.com/yuge-hk/podux/actions/runs/39845439391): completed, **Failed**.
- Feature branch HEAD remained `9307c8211aeec2eb4d74811ec922fd455b3f6294`; no feature business code, commit or remote branch was changed for this comparison. The requested Phase 3 report/evidence are updated locally.

## Isolation and comparability

The baseline starts directly from upstream main, without merging/cherry-picking feature business commits. Its tested delta is only six added files: an FRPC integration test, independent workflow, isolated FRPS builder, native host checker, results collector and provenance manifest. Original main.go, FRPC service/models/repositories, migrations and go.mod/go.sum are byte-identical to main; [isolation proof](phase3-evidence/frpc-mainline-baseline/isolation-proof.json) includes SHA-256 hashes for 22 original backend/module files. [Added-file manifest](phase3-evidence/frpc-mainline-baseline/added-files.json).

The test is derived from the feature integration test. [Exact adaptation diff](phase3-evidence/frpc-mainline-baseline/test-adaptation.patch) removes Cloudflare imports, test connector compilation, manager lifecycle and credential assertions; no Cloudflare implementation or migration is transplanted. Both traffic probe pairs surrounding those removed steps are retained. Test name changes to TestFRPCTCPUDPReconnectBaseline; the identical `^TestFRPC` selector selects it.

The original FRPC server/proxy record fields, synthetic local-only auth, TCP/UDP echo payloads, random loopback ports, heartbeat interval 1/timeout 3, 15s eventual deadline/100ms polling, reload/outage/reconnect/topology/shutdown and port-release assertions remain unchanged. FRPS is built with the same helper and noweb tag at unchanged **v0.68.0**. Both native workflows use Go **1.25.5** from go.mod, identical Runner labels/architectures, Python 3.12 and Windows CGO discovery. Baseline frontend is built from main's frozen lockfile to provide pb_public; no feature frontend is used.

Identical functional command:

```sh
go test -json -count=1 -timeout 3m ./internal/integration -run '^TestFRPC'
```

Identical race command (PODUX_TEST_FRPS set to the local pinned build):

```sh
CGO_ENABLED=1 go test -race -json -count=1 -timeout 3m ./internal/integration -run '^TestFRPC'
```

Native OS/CPU is checked against Go target/host and platform.machine. All four platforms actually ran race with CGO; none was skipped, emulated or substituted by cross compilation. No repeat-until-failure run, disabled assertion or continue-on-error is used. Removing Cloudflare activity changes scheduling; this is an independent FRPC baseline, not identical overall application load or a race-frequency experiment.

## Actual platform comparison

| Runner / CPU | Mainline build | Mainline functional | Mainline race | Feature functional | Feature FRPC race |
| --- | --- | --- | --- | --- | --- |
| ubuntu-24.04 / amd64 | Passed | Passed | **Failed** | Passed | **Failed** |
| ubuntu-24.04-arm / arm64 | Passed | Passed | Passed | Passed | **Failed** |
| windows-2025 / amd64 | Passed | Passed | **Failed** | Passed | **Failed** |
| macos-15 / arm64 | Passed | Passed | **Failed** | Passed | **Failed** |

Mainline Linux amd64 and macOS report the same FRP access pair as the feature branch:

- Write: `client.(*Service).stop`, `github.com/fatedier/frp@v0.68.0/client/service.go:501`.
- Read: `client.(*Service).keepControllerWorking.func1`, `client/service.go:286`.

Mainline Windows reports the same FRP logger access pair as the feature branch:

- Write: `pkg/util/xlog.(*Logger).renderPrefixString`, `pkg/util/xlog/xlog.go:85` (called by AddPrefix).
- Read: `pkg/util/xlog.(*Logger).Tracef`, `pkg/util/xlog/xlog.go:115`.

[Machine-readable comparison](phase3-evidence/frpc-mainline-baseline/comparison.json) records exact access pairs. Each platform's evidence directory contains the complete original Go JSON race output, readable complete race output, functional output, actual outcomes, native environment and FRPS version/hash. All functional runs validate traffic, reload/reconnect, shutdown and port release; no public FRPS or production configuration was accessed. The supporting local Linux baseline also passes functional tests and fails race; those logs are retained separately from native Actions evidence.

## Conclusion and limits

**Confirmed:** the mainline Podux integration independently reproduces the same Service shutdown and logger prefix races, with the competing reads/writes inside existing FRP v0.68.0 and no Cloudflare business code present. These two race classes predate the Cloudflare feature; the feature is not necessary to trigger them. This is pre-existence evidence, not a fix or an exhaustive API-contract/root-cause analysis.

Linux arm64's single baseline race run detected no race. That is a genuine Passed execution, not proof the shared dependency is race-free on ARM64. Detection varies with scheduling; feature macOS also passed one earlier run and failed the next. No detection-rate comparison is claimed. The UDPProxy race observed in the first feature run was not reproduced in this baseline; this subtype remains individually unconfirmed by the control experiment.

The baseline workflow correctly remains Failed. No FRPC source change, FRP upgrade or race remediation was made. No upstream push or PR was created. The feature CI result remains Failed; baseline attribution does not turn failed checks green or establish merge readiness.

## Evidence retention and cleanup

The remote temporary test branch and Actions run are retained. Complete safe evidence is committed separately to that branch using a documentation-only `[skip ci]` commit; this does not replace or rerun the tested code SHA. The independent local worktree and local test branch are removed after archiving. The original feature branch's commit/remote remain unchanged; only the explicitly requested local validation documents/evidence are updated.


Cleanup verified: the independent local worktree and local `test/frpc-race-baseline-20261023` branch were removed after the evidence push. Temporary baseline downloads were removed after copying evidence. The original feature HEAD and remote ref remain `9307c8211aeec2eb4d74811ec922fd455b3f6294`.

Persistent evidence-only archive SHA: `6a7ef34b713e344872f41271d6b98f2ce8a86253` on the remote baseline branch. [Archived report and complete stacks](https://github.com/yuge-hk/podux/blob/6a7ef34b713e344872f41271d6b98f2ce8a86253/development/frpc-mainline-race-baseline.md). The test result still belongs to `6ff9263f6242e5a95b4e127b844642364923a13e`; the later commit contains only documentation/evidence and intentionally does not trigger another run. Local Phase 3 document/evidence updates are left uncommitted on the feature worktree to preserve its commit history and remote ref.
