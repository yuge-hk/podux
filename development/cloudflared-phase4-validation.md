# Cloudflare Tunnel Phase 4 acceptance and PR review

## Phase 4 decision (historical)

**Code/PR preparation is complete for review; live Edge acceptance is Blocked.** No authorized nonproduction Tunnel Token or test origin was provided. No production credential was read or borrowed. No Cloudflare Edge/public-origin success is claimed, and no upstream PR has been created.

The Fork feature branch is `yuge-hk/podux:feat/local-cloudflare-tunnels`. At the start of this phase its HEAD was `9307c8211aeec2eb4d74811ec922fd455b3f6294`. Fresh upstream main is still `1136797a36a2f1186055278351f55e28b907d6d5`, the feature's original base, so no merge/rebase or upstream conflict resolution was needed. Pre-existing uncommitted Phase 3 control evidence was preserved before cleanup.

`Passed` means actually executed successfully; `Failed` means actually executed unsuccessfully; `Skipped` means not executed; `Blocked` means a required condition was missing. Cross compilation is not counted as native execution.

## Scope of this phase

No new business capability is added. The only backend hardening change suppresses the new Tunnel collection's SQL debug/query/exec/performance callbacks before request serving. The audit observed encrypted Token values in PocketBase development-mode interpolated SQL, although no plaintext Token was observed. A focused regression now verifies create/update/transaction writes do not reach those loggers and unrelated SQL logging remains enabled. AES-GCM storage, collection migration, existing authentication, APIs and runner lifecycle otherwise retain their original behavior.

Documentation now includes README links, discoverable Chinese/English guides, installation/container/key requirements, two labelled fixture screenshots and known limits. Browser QA accepts explicitly disposable loopback endpoints so it can use isolated ports without touching an existing service. The original publishing workflow is byte-identical to main; duplicated earlier Cloudflare checks were consolidated into the already existing standalone native workflow, with no test case disabled. FRPC source/models, go.mod/go.sum and FRP v0.68.0 are unchanged.

## Actual local validation

Local environment: Linux amd64, Go 1.25.5, Node 24.20.0, pnpm 10.34.7. The native Actions frontend uses the repository's Node 20 baseline, recorded separately below.

| Check | Result | Evidence/scope |
| --- | --- | --- |
| Frozen frontend install, TypeScript, production build | Passed | Existing lockfile unchanged; current application builds |
| Vitest | Passed | 68 tests |
| Targeted ESLint / QA JavaScript syntax and formatting | Passed | New Tunnel frontend and changed QA scripts |
| Existing full frontend lint | Failed | Same 26 errors / 3 warnings in unrelated existing files |
| Go vet / application build | Passed | Real freshly built frontend copied into ignored pb_public first |
| Go offline unit/process/API/migration suite | Passed | 28 Linux tests, including new SQL-log regression |
| Same offline suite with race | Passed | 28 tests; optional official/FRPS integration are separately scoped, not silently counted as run |
| Official cloudflared 2026.10.3 | Passed | Local invalid-token parser/token-file/failure/cleanup checks; not a successful Edge session |
| Browser general regression | Passed | 18 checks against disposable actual backend and official binary local-failure path |
| Browser fixture lifecycle | Passed | 9 checks, including concurrent child processes, independent Stop/Restart, Metrics, token updates and revision conflict |
| Actual Podux restart/recovery acceptance | Passed | 7 Linux checks with real application restarts and test-only connector; details below |
| VitePress documentation build | Passed | Chinese/English routes and public screenshots |
| Workflow lint / CI helper unit tests | Passed | actionlint 1.7.7; 3 helper tests |
| Authenticated real Edge connection / two real tunnels | Blocked | Authorized nonproduction credentials absent |
| Real Edge restart/recovery, protocol/region behavior | Blocked | Fixture/parser checks cannot establish account-side behavior |
| Public-origin reachability | Blocked | No authorized Tunnel credentials or test origin |
| Extended 1–2 hour soak | Skipped | Short lifecycle checks do not establish long-term stability |

The actual application recovery probe uses only a new temporary data directory, explicitly separate key file, random loopback port, synthetic Token and test connector. It starts two child processes with distinct Metrics listeners, verifies encrypted persistence/non-secret APIs, restarts Podux and observes only autoStart restoration, force-stops the sole test child and observes one bounded automatic retry, verifies manual Stop preserves next-start autoStart behavior, then confirms key continuity, cleanup and port release. It also scans captured development output for both plaintext and encrypted test Token values after the fix. No test database/key is committed.

An initial probe attempted to rebind a TCP port without SO_REUSEADDR and encountered TIME_WAIT after a clean shutdown. The corrected probe first verifies the listener refuses connections, then uses the same Unix reuse semantics as Go net.Listen. No product test/assertion was disabled to hide a process leak.

## Native GitHub Actions

Historical run [39845439391](https://github.com/yuge-hk/podux/actions/runs/39845439391), tested SHA `c2e6e7d5328add30eec04177a6630cd3e7e42454`, actually passed all four platforms' Cloudflare suites, official binary checks and FRPC functional regression. All four separate FRPC race steps Failed. First run [39844432704](https://github.com/yuge-hk/podux/actions/runs/39844432704) is also retained.

Fresh Phase 4 [run 39964271753](https://github.com/yuge-hk/podux/actions/runs/39964271753) tested executable-code SHA `583bbc19d1779431b1a0dec4ab47364a3d64a5a1`. Frontend Passed. All native build/vet, unit, Cloudflare race, official-binary and functional FRPC steps Passed. The overall workflow is **Failed** because three FRPC race steps failed; macOS did not reproduce the FRPC race in this run. No step uses continue-on-error. The new SQL-log regression actually passed on all four native platforms.

| Native runner / CPU | Phase 4 Cloudflare suite | Phase 4 official binary | Phase 4 FRPC functional | Phase 4 FRPC race |
| --- | --- | --- | --- | --- |
| ubuntu-24.04 / amd64 | Passed (unit + race) | Passed | Passed | Failed |
| ubuntu-24.04-arm / arm64 | Passed (unit + race) | Passed | Passed | Failed |
| windows-2025 / amd64 | Passed (unit + race) | Passed | Passed | Failed |
| macos-15 / arm64 | Passed (unit + race) | Passed | Passed | Passed (not reproduced) |

## FRP boundary and upstream fixes

The independent pure-main v0.68.0 control used test SHA `6ff9263f6242e5a95b4e127b844642364923a13e`, [run 39847653720](https://github.com/yuge-hk/podux/actions/runs/39847653720), and reproduced matching pre-existing FRP races without Cloudflare business code. [Complete control report](https://github.com/yuge-hk/podux/blob/6a7ef34b713e344872f41271d6b98f2ce8a86253/development/frpc-mainline-race-baseline.md).

FRP merged [#5585: generation-local UDPProxy channels](https://github.com/fatedier/frp/pull/5585), merge commit `8d5e9d2b21e573f2bcac068a8a438571171c4920`, and [#5586: concurrent logger prefix updates](https://github.com/fatedier/frp/pull/5586), merge commit `93b52799bb7e6dad7ae1152c8b593eefcc17d595`. These repair references are preserved, but **Podux still embeds FRP v0.68.0**. No FRP upgrade, replacement, vendor patch or FRPC implementation change is made here. Old-version race failures remain Failed and may still keep the full workflow red; an upstream merge is not evidence that this pinned dependency has changed. Historical investigations are separated from the feature diff.

## Security review

Reviewed: AES-256-GCM random nonces and record-bound associated data; external key location/permissions and fail-closed restoration; hidden ciphertext collection field and locked rules; write-only custom API/JSON fields; strict body/argument validation and revision checking; existing authentication without query-token fallback on Tunnel routes; exec argv/no shell/minimal child environment; protected token files; one Wait owner; per-profile commands/state/retry; loopback-only bounded Metrics; bounded allowlisted events; safe frontend errors and no Token persistence/console output. The SQL ciphertext observation above was corrected with a regression, not dismissed because it was encrypted.

Fresh dependency checks remain **Failed for baseline findings**: 54 reachable Go advisory IDs; frontend 67 findings (42 high, 23 moderate, 2 low, zero critical). Both advisory-ID sets have no additions versus the recorded main baseline. govulncheck JSON mode exits zero even when findings exist; that is not treated as a clean scan. No broad dependency upgrade was made to hide these findings.

The final staged diff is checked for private-key/GitHub-token patterns, runtime databases/keys/binaries, generated web output and unintended files. Synthetic test Tokens are deliberately nonproduction. No real Token, encryption key, database, session credential or raw backend SQL dump belongs in the feature commit/evidence publication. Go formatting and local relative links are checked. Podux/cloudflared are Apache-2.0; implementation is independent and no cfui code is copied.

## Evidence retention and review footprint

Complete Phase 1–3 reports, original screenshots, raw race/audit/resource outputs and the inherited uncommitted control evidence were committed and pushed first to archive SHA `a7576af0b5d4610aca38f6708a28a8e4bfe00dcb` ([browse archive](https://github.com/yuge-hk/podux/tree/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development)). Copies also remain in the local validation delivery directory. Only after confirming the archive push were generated raw evidence and duplicate screenshots removed from this feature tip. No functional test source or capability was removed. Generated QA output folders are ignored to avoid accidental reintroduction; two current, explicitly labelled screenshots are versioned in docs/public/screenshots.

Current safe evidence is collected outside the source tree at `/root/公共/podux-phase4-validation`; credential-bearing disposable backend logs/data are not published. Curated current proof and exact CI metadata are retained at evidence-only archive SHA `c58489a2bd9a0ea80963652740382e2eadf4282a` ([browse Phase 4 evidence](https://github.com/yuge-hk/podux/tree/c58489a2bd9a0ea80963652740382e2eadf4282a/development/phase4-evidence)). This includes full four-platform race stacks and all outcomes; original Windows artifact CRLF is retained. The final feature-tip commit after executable-code SHA `583bbc1` changes only this report and the PR draft, so it does not imply another code test run. The [final file manifest](cloudflared-phase4-files.txt) lists the complete proposed diff against upstream main, and the [English PR draft](cloudflared-pr-draft.md) describes scope, validation and outstanding acceptance.

## Remaining review decisions

Real Edge/public-origin tests remain Blocked until dedicated nonproduction credentials and an authorized test origin are available. Known FRP race/dependency audit findings remain outside the requested change. Normal native lifecycle coverage does not establish universal orphan-process handling on forced macOS/Windows parent crashes. Windows custom key overrides must currently remain on the same volume as the data directory; cross-volume paths fail closed. No 1–2 hour soak was performed. These limits must remain visible in the PR; this report does not claim completed production/live acceptance.

The default review proposal is one coherent backend/migration/UI/docs feature, with a separate FRP dependency change if maintainers authorize it. A backend-first/UI-second PR split is possible without discarding the Fork implementation, but no upstream PR or automatic merge is performed in this round.


## Phase 5 pre-submission verification — 2026-10-23

**Draft PR material is prepared; live acceptance remains Blocked. No upstream PR was created.** This section supersedes no historical result and does not turn the Phase 4 fixture tests into Edge evidence.

### Repository and scope

Phase 5 started from clean feature SHA `c4b4f6fbe84ae0122161ba4cd22db4df7836152a`. A fresh `git fetch origin main` confirmed upstream `1136797a36a2f1186055278351f55e28b907d6d5`, with zero upstream commits missing from the feature and no merge/rebase needed. The executable source is unchanged from native-tested SHA `583bbc19d1779431b1a0dec4ab47364a3d64a5a1`. Phase 5 changes only review documentation; the complete proposed manifest remains 90 files: 37 backend/migration, 23 frontend/QA, 10 CI/helper and 20 documentation/review files. Each manifest path was checked against the actual diff and its scope recorded locally. Historical race investigation artifacts stay in their immutable archive branches. No FRPC/model, FRP module, Docker, release or existing publishing-workflow changes were introduced.

### Real Edge acceptance

The current host's listening sockets did not expose a running Podux test instance. The user was asked for the local test URL and the two dedicated profile names, plus an optional authorized test domain; these details were not supplied during this verification. No Token, credential file, database, browser session store or process environment was read to discover credentials. No unknown service was stopped or restarted. A remote/manual test environment may exist, but its absence cannot be inferred from this host's sockets alone.

| Requested live scenario | Result | Missing evidence / condition |
| --- | --- | --- |
| Successful Edge connection | Blocked | Identified authenticated test instance with a manually saved dedicated nonproduction Token |
| Podux vs Cloudflare Dashboard | Blocked | Live status plus user-observed Dashboard comparison; no Cloudflare account/API integration is added |
| Start / Stop / Restart | Blocked | Identified dedicated live profile; prior synthetic lifecycle coverage remains separate |
| Two simultaneous independent Tunnels | Blocked | Two identified dedicated profiles and their independent Edge sessions |
| Whole-Podux restart and autoStart recovery | Blocked | Identified isolated test service and live profiles; no unknown service was restarted |
| Safe live Token replacement / profile deletion | Blocked | User performs Token entry/replacement manually; replacement/deletion needs designated disposable profiles |
| Live logs / Metrics / database / error confidentiality | Blocked | Live data not available; synthetic/API/encryption/logging regressions Passed separately |
| Public HTTP / test-origin response | Blocked | Authorized dedicated hostname and expected origin response were not supplied |

For a later live run, collect only whitelisted status fields (phase/ready/connections/protocol/generation/revision/retry count), fixed safe event codes, Metrics counts and timestamps. Correlate them with a user-confirmed Dashboard state. Do not capture token fields, raw request bodies, browser network archives, raw child output, database dumps, temporary token-file contents or encryption keys. Existing write-only UI must handle manual replacement; automation must not retrieve or replay the plaintext Token. A dedicated hostname can be tested only after the user identifies it and its expected non-sensitive response. Real Token confidentiality cannot be proved by the synthetic tests alone.

### Fresh local checks

All checks below actually ran on Linux amd64 at Phase 5 base SHA `c4b4f6f`; no executable code changed afterwards. Raw synthetic-only output and the per-file scope/hash inventory remain outside Git at `/root/公共/podux-phase5-validation`.

| Check | Result | Actual scope |
| --- | --- | --- |
| Frontend frozen install / TypeScript / Vite build | Passed | Repository lockfile retained |
| Vitest | Passed | 68 tests |
| Tunnel-targeted ESLint | Passed | Tunnel pages, test setup, App/layout and Vitest configuration |
| Full frontend lint | Failed | Existing 26 errors / 3 warnings; no unrelated cleanup |
| Go vet / build | Passed | Newly built frontend prepared in ignored pb_public |
| Offline Go tests | Passed | 28 Passed; official binary and FRPS opt-ins explicitly Skipped |
| Offline Go race tests | Passed | 28 Passed; same two explicit opt-in skips |
| Official cloudflared binary | Passed | 2026.10.3, TestOfficialBinary separately enabled; synthetic invalid-token local path only |
| Documentation frozen install / build | Passed | Chinese/English VitePress site |
| New native workflow run | Skipped | No executable/workflow change; verified existing native run and immutable evidence instead |
| Dependency audit refresh | Skipped | Phase 4's 54 Go advisory IDs and 67 frontend findings remain disclosed, not resolved or reclassified |

[Actions run 39964271753](https://github.com/yuge-hk/podux/actions/runs/39964271753) was rechecked using GitHub CLI. Its overall result remains **Failed**. All four platforms' Cloudflare unit/race/official-binary tests and FRPC functional steps Passed. FRPC race Failed on Linux amd64, Linux arm64 and Windows amd64; macOS arm64 did not reproduce it that run. This is native evidence for executable SHA `583bbc1`, not a newly executed Phase 5 matrix. The pinned dependency remains v0.68.0. GitHub metadata reconfirmed upstream fixes [#5585](https://github.com/fatedier/frp/pull/5585) and [#5586](https://github.com/fatedier/frp/pull/5586) are merged; they have not been incorporated here.

### Review readiness and limitations

The English guide, labelled fixture screenshots, installation/key backup requirements, capability/TLS boundaries and native process-cleanup limits remain accurate; no screenshot is presented as a successful live Edge connection. The file manifest matches all 90 paths. Runtime data/build outputs remain ignored. The full proposed file set passed private-key/GitHub-credential pattern and prohibited runtime-file checks, and the diff passed whitespace checks. These checks do not constitute proof that all possible secret formats are detectable. No real credential was available or introduced during this phase.

The [English draft](cloudflared-pr-draft.md) is prepared for a future **Draft** PR, with live acceptance Blocked, known dependency/lint findings and the red FRPC race check visible. No claim of production readiness is made. Creating any upstream PR still requires explicit user authorization. No production environment or Cloudflare account configuration was modified.


## Phase 5B — isolated environment prepared, awaiting manual Tokens

Phase 5B prepares the missing test environment rather than repeating the completed full code review/build suites. The feature branch started clean at `379437808da1f9e062baccc5bdd8887e8e8bedb1`; no business code, dependency, FRPC implementation or production resource was changed.

A new disposable runtime was created outside the repository at `/root/公共/podux-phase5b-runtime`, with its own migrated PocketBase data directory, fresh external AES-GCM master key, private local login file and server-controlled configuration. Runtime root/data/key directory permissions are 0700 and the new master key is 0600. No existing database or key was reused. The service binds only to loopback; raw console output is discarded to avoid collecting credential-bearing debug dumps. Login credentials and runtime data are not committed. The service is intentionally left running for manual Token entry; it must not be deleted before the live test resumes.

- WebUI: `http://127.0.0.1:48179/tunnels` (on the test host; use SSH local forwarding when accessing remotely).
- Actual official binary: cloudflared `2026.10.3`, built `2026-10-21-13:11 UTC`; capability detection reports available and token-file support.
- Named profiles: `podux-test-a` and `podux-test-b`. Both are stopped, not ready, zero connections and autoStart=false; autoRestart=true with three additional attempts is prepared for subsequent live testing. Metrics selects separate loopback ports on Start.
- The create API requires a Token. Both initial records therefore contain a clearly documented, synthetic invalid placeholder, with a description instructing manual replacement before Start. The placeholder is not an authorized credential; hasToken only means a stored encrypted value and must not be interpreted as live readiness. No connector was started with a placeholder. The actual user Token is neither requested through chat/CLI nor inspected by the preparation script.
- **Passed preparation checks:** real HTML/assets entry served; unauthenticated Tunnel API rejected with HTTP 401; authenticated API lists exactly the two intended stopped profiles; binary capabilities available; separate master-key permissions checked. These are readiness checks, not another full regression suite or proof of an Edge connection.

The sole required user action is to log in to this test WebUI using the private local `login.json`, edit each named profile's Token field, replace the placeholder with the corresponding dedicated nonproduction connector Token and save, then report that entry is complete without sending either value. [Cloudflare's official instructions](https://developers.cloudflare.com/tunnel/reference/tunnel-tokens/) identify Dashboard **Networking → Tunnels → select tunnel → Add a replica**; obtain the Token locally without running the service-install command or sending its contents to this task. No Cloudflare API/account management is added. Do not use production Tunnel credentials.

**Live Edge status remains Blocked pending those manual replacements.** Successful Edge connections, dual-profile independence, real Start/Stop/Restart, whole-application autoStart recovery, abnormal-exit retries and Dashboard correlation have not yet run. Future automation may use existing authenticated status/control endpoints and safe Metrics fields; it must not retrieve, decrypt or copy the saved Tokens or capture browser request bodies. Token replacement stays a manual UI action. Optional public-origin testing additionally depends on an existing authorized dedicated route/domain; no public HTTP success is claimed.

Local preparation proof and lifecycle helper remain private under the runtime directory; only non-secret status fields are collected. No real-token leak audit is claimed before live data exists. Historical FRPC race failures, existing frontend lint errors and baseline dependency findings remain as disclosed above. The English Draft PR text is updated to distinguish a ready test environment from completed live acceptance. No upstream PR has been created.


## Phase 5B follow-up — integrated server management and stop-error correction

The user reported normal connection behavior but observed a connection-error message on Stop, requested integration into the Servers page, and requested Start prevention without a Token. This follow-up changes only the Cloudflare runner/application guards, related UI wiring and documentation/tests. FRPC implementation and FRP dependency remain unchanged.

Root cause: cloudflared emits ERR-level connection-close output during graceful cancellation. The output sink previously mapped every ERR to LastError, and a normal stop retained that field. The sink now ignores connection errors in stopping/stopped phases, while normal completion clears current error state. Genuine runtime errors and historical events remain visible. A real-child regression emits ERR during graceful termination, verifies no false stop failure, and separately proves active-runtime errors are still reported.

Cloudflare is now a tab in `/servers?type=cloudflare`, sharing the server header, Radix components and existing Tailwind card/grid conventions. `/tunnels` redirects there, and the separate navigation entry is removed. FRPC hooks mount only in their own tab. Existing server-view behavior is retained with an optional embedded-header flag. Missing-Token profiles disable both Start and Restart; the backend rejects absent credential metadata/ciphertext or invalid decrypted Token structures before either operation can reach the runner. Creating a profile still requires Token input, and blank Token edits continue to preserve an existing credential.

Actual local validation for this revision:

- Passed: 70 frontend tests, including missing-Token controls and URL-driven server-tab switching/lazy FRPC mounting.
- Passed: frontend TypeScript/production build and targeted ESLint. Existing repository-wide lint/dependency findings remain disclosed; no unrelated cleanup or dependency upgrade is included.
- Passed: runner/application tests with `go test -race`, including graceful shutdown output and missing/empty credential guards.
- Passed: real-browser legacy-route redirect, removal of separate navigation, both server tabs and 390px viewport without horizontal overflow. Initial browser probes read tab state before React committed navigation and measured immediately after viewport resize; probes were corrected to wait for selected-tab/layout completion, without relaxing product assertions.
- Passed: isolated local instance updated using its existing paired data/key; only its test processes restarted. Saved Tokens were not read, decrypted, exported or replayed by automation. Pre-update running/stopped choices were restored.
- Passed live regression: `podux-test-b` observed running/ready, protocol QUIC, two HA connections. Manual Stop returned stopped with no LastError and no new connection-error event. Manual Start restored running/ready, QUIC and two connections. `podux-test-a` was left stopped. Safe proof is local at `/root/公共/podux-phase5b-runtime/stop-fix-proof.json`; browser proof is in the same private runtime directory.

This supersedes the earlier blanket Edge Blocked statement for **one connector's observed readiness and Stop/Start behavior only**. Dual live Tunnels, Dashboard comparison, real abnormal-exit retry and public-origin responses are not claimed verified by this follow-up. No plaintext Token, database, encryption key, login/session credential or raw child output is published. The former 90-file manifest gains the two existing Servers source paths and their new integration test; the manifest is regenerated from the actual upstream diff. Historical test counts/SHAs remain historical, and old native CI is not presented as a test of this revision. No upstream PR was created.
