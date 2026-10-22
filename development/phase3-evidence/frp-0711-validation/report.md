# FRP v0.71.1 independent race validation

Validation date: 2026-10-23. **Overall Actions result: Failed.** TCP/UDP functionality passes, but two race classes remain reproducible. No production change, upstream write, Issue publication or PR was performed.

## Exact source and executions

| Item | Value |
| --- | --- |
| Official Podux main | `1136797a36a2f1186055278351f55e28b907d6d5` |
| Personal Fork branch | `yuge-hk/podux:test/frp-0711-race-20261023` |
| Tested SHA | `8ae7bd98818946262a3f830d3279d03eb18e9047` |
| Native Actions | https://github.com/yuge-hk/podux/actions/runs/39851903747 |
| Go | **1.26.10**, pinned in the temporary branch's go.mod |
| FRPC and local FRPS | **v0.71.1**, no replacement module or patch |
| Previous mainline v0.68.0 test SHA | `6ff9263f6242e5a95b4e127b844642364923a13e` |
| Previous v0.68.0 Actions | https://github.com/yuge-hk/podux/actions/runs/39847653720 |

[Workflow at the tested SHA](https://github.com/yuge-hk/podux/blob/8ae7bd98818946262a3f830d3279d03eb18e9047/.github/workflows/frp-0711-race.yml). Only the temporary branch upgrades Go/FRP and the indirect module versions required by dependency resolution. PocketBase remains 0.35.0. No Podux API adaptation was needed: all 20 original backend application/domain/infrastructure/migration/main source files are byte-identical to main. The integration test is byte-identical to the previous v0.68.0 control test. Provenance, source hashes, module delta and feature-worktree before/after receipts are retained with the evidence.

The original Cloudflare branch remains `9307c8211aeec2eb4d74811ec922fd455b3f6294`; its pre-existing local document changes are also untouched. New outputs are outside that worktree, in the independent validation delivery directory.

## Reproduction and trial policy

The workflow builds main's frontend with frozen pnpm install and provides its real pb_public artifact before native Go build/vet. Go comes from the temporary go.mod. Native Go target/host and hardware architecture are checked. Windows CGO compiler discovery matches the previous control. All four native platforms ran race with CGO; no race check was skipped or replaced by cross compilation.

One functional run and **three predetermined race runs per platform** use the original scenario and unchanged single-run flags:

```sh
# PODUX_TEST_FRPS points to the native v0.71.1 FRPS built by build/ci/build_frps.py.
go test -json -count=1 -timeout 3m ./internal/integration -run '^TestFRPC'
CGO_ENABLED=1 go test -race -json -count=1 -timeout 3m ./internal/integration -run '^TestFRPC'
```

The three race steps run even after an earlier race failure; any failure makes the job fail. Passed trials are retained alongside failed ones. No continue-on-error, disabled race detector, dropped assertions or repeat-until-pass policy is used.

The test uses isolated temporary PocketBase data, unchanged Podux FRPC service, synthetic local auth, real TCP/UDP echo origins and FRPS on random loopback ports. It checks forwarding, single-proxy disable/restore, UDP recovery after reload, FRPS outage/reconnection, topology, normal shutdown and TCP/UDP port release. There is no Cloudflare package, migration or runner in this branch. No public/production FRPS is contacted. Functional runs pass all assertions; race runs still fail when the detector reports a race.

## Actual native results

| Platform | Build/vet | TCP/UDP/reload/reconnect/exit | Race trials 1 / 2 / 3 | xlog detections | UDP detections | Overall job |
| --- | --- | --- | --- | --- | --- | --- |
| ubuntu-24.04 / amd64 | Passed | Passed | Failed / Passed / Failed | 2/3 | 1/3 | Failed |
| ubuntu-24.04-arm / arm64 | Passed | Passed | Failed / Failed / Failed | 3/3 | 0/3 | Failed |
| windows-2025 / amd64 | Passed | Passed | Failed / Passed / Failed | 2/3 | 0/3 | Failed |
| macos-15 / arm64 | Passed | Passed | Failed / Failed / Passed | 2/3 | 2/3 | Failed |

All 12 planned race trials executed: **9 Failed, 3 Passed**. Old Service controller race: **0/12**. xlog prefix race: **9/12**. UDPProxy race: **3/12** (each affected trial reports both channel-field accesses). No other race category was observed. These are observations, not estimates of race probability.

## Comparison by race class

### Service.stop / keepControllerWorking: old access-pair fix present

Mainline v0.68.0 independently reported `client/service.go:501` (stop writes ctl) versus `:286` (keepControllerWorking reads ctl) on Linux amd64 and macOS arm64. In v0.71.1, keepControllerWorking snapshots ctl while holding ctlMu.RLock, for both initial and reconnect-loop reads; stop writes under ctlMu.Lock. No recurrence appeared in any of the 12 new native trials.

The [controller reconnect fix PR #5547](https://github.com/fatedier/frp/pull/5547) and subsequent [startup/shutdown PR #5560](https://github.com/fatedier/frp/pull/5560) are merged ancestors of v0.71.1, verified through the GitHub comparison API. [Tagged source](https://github.com/fatedier/frp/blob/v0.71.1/client/service.go#L289) corroborates the relevant synchronization.

Classification: **the specific previously observed controller access pair has source-level repair evidence and did not recur**. This does not establish that every Service lifecycle race is fixed, nor is zero detections alone treated as proof.

### xlog.AddPrefix / Tracef: still exists

The old baseline Windows run reported the prefix write in renderPrefixString (called by AddPrefix) racing with Tracef's read. v0.71.1 reproduces the same field-access class on **all four OS/CPU combinations**, in nine native trials. The logging call now uses WithPrefix, but constructing that call still reads the shared prefixString without synchronizing with the writer.

Relevant v0.71.1 sites: `pkg/util/xlog/xlog.go:85` (renderPrefixString write) versus `:115` (Tracef read). [Tagged source](https://github.com/fatedier/frp/blob/v0.71.1/pkg/util/xlog/xlog.go#L80). The configured log level remains the original test's `error`; this does not prevent the observed prefix access.

Classification: **still present, independently reproduced on latest requested release**.

### UDPProxy: still exists, now independently reproduced without Cloudflare

The first v0.68.0 feature run had UDPProxy.InWorkConn writes at `client/proxy/udp.go:102` / `:103` versus reads at `:159`; the later pure-main v0.68.0 single-trial control did not detect this class. Those two contexts are kept separate in the evidence.

v0.71.1 independently reproduces the same channel-field pattern on pure main plus the dependency upgrade: writes to readCh/sendCh at `client/proxy/udp.go:111` / `:112` versus reads at `:168` in overlapping InWorkConn calls. It occurs in Linux amd64 trial 1 and macOS arm64 trials 1 and 2. The writer uses the proxy mutex, but the relevant field reads passed into Forwarder occur outside that critical section. [Tagged source](https://github.com/fatedier/frp/blob/v0.71.1/client/proxy/udp.go#L109).

Classification: **still present, with independent current-version native reproduction**. Linux arm64 and Windows did not detect UDP races in their three runs; that is not evidence of a platform-specific fix.

### Limits of the comparison

The previous pure-main control used Go 1.25.5/FRP 0.68.0 and one trial per platform; the new test requires Go 1.26.10, newer FRP transitive dependencies and three trials. Therefore failure rates are not directly comparable and version-only causal attribution from counts is not justified. The scenario/test source is identical. The Service conclusion combines code synchronization evidence with observed non-recurrence; xlog/UDP conclusions rely on positive runtime detections. No general race-free or production-upgrade approval is implied. Server internals are not race-instrumented by this embedded-client test.

## Complete evidence

[Permanent evidence directory on the personal test branch](https://github.com/yuge-hk/podux/tree/test/frp-0711-race-20261023/development/phase3-evidence/frp-0711-validation).

- `native/<os>-<arch>/`: actual environment (including Go version), FRPS version/hash, outcomes, functional Go JSON and all three complete race Go JSON files; each race file also has readable complete output.
- `baseline-v068/<os>-<arch>/`: the full prior pure-main control stacks/outcomes; `feature-first-linux-arm64-race.json` is explicitly the earlier feature context supplying the old UDP stack.
- `comparison.json`: all detected access pairs with functions and source lines, per trial, alongside prior v0.68.0 frames.
- `run.json`, `failed-steps.txt`: real GitHub run metadata and failed step logs. Since stdout is redirected into artifacts, complete stacks are in the Go JSON/readable outputs rather than the short exit-code step log.
- `isolation-proof.json`, feature-worktree before/after receipts, source hashes, module delta, upstream PR merge/ancestry and search snapshots.
- `local-functional.json`, `local-race.json`: supporting local Linux preflight, distinguished from GitHub native evidence.

## FRP Issue assessment

**Worth preparing two narrowly scoped upstream reports:** xlog prefix mutation/read race and UDPProxy channel-field races on v0.71.1. Both have positive native detections, exact release/toolchain, complete goroutine creation stacks, public workflow and a source-pinned integration reproducer. Keep the already repaired controller access pair out of those new Issue titles; mention it only as comparison context.

A read-only search found the closed [Issue #5557](https://github.com/fatedier/frp/issues/5557) and its lifecycle fixes. Searches for AddPrefix/prefix and UDPProxy/UDP races did not identify a dedicated matching report, but search results are not exhaustive; recheck duplicates immediately before filing. Two **unpublished** drafts are retained beside the evidence. No Issue/comment/PR was sent to FRP.

This test supports reporting the remaining races. It does **not** support calling v0.71.1 race-clean or recommending the dependency upgrade as the sole solution to the CI failures. Production deployment and broader Podux migration review remain outside this experiment.

## Retention and cleanup

Source/configuration and the later evidence-only archive are pushed only to the personal temporary test branch. An evidence-only `[skip ci]` commit preserves results without producing a third code revision test claim; it does not turn the failed run green. The local temporary worktree/branch are removed after the evidence push, while the remote branch and Actions record remain. No upstream push, FRP patch, Cloudflare-branch commit or production operation is performed.
