# FRPC upstream-main race baseline

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
