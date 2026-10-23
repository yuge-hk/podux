# Cloudflare Tunnel Phase 4 acceptance and PR review

## Current decision

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

A fresh Phase 4 feature-branch push will validate the SQL-log hardening on the same native matrix. Results must be recorded from the real run, not inferred from historical success:

| Native runner / CPU | Phase 4 Cloudflare suite | Phase 4 official binary | Phase 4 FRPC functional | Phase 4 FRPC race |
| --- | --- | --- | --- | --- |
| ubuntu-24.04 / amd64 | Pending | Pending | Pending | Pending |
| ubuntu-24.04-arm / arm64 | Pending | Pending | Pending | Pending |
| windows-2025 / amd64 | Pending | Pending | Pending | Pending |
| macos-15 / arm64 | Pending | Pending | Pending | Pending |

## FRP boundary and upstream fixes

The independent pure-main v0.68.0 control used test SHA `6ff9263f6242e5a95b4e127b844642364923a13e`, [run 39847653720](https://github.com/yuge-hk/podux/actions/runs/39847653720), and reproduced matching pre-existing FRP races without Cloudflare business code. [Complete control report](https://github.com/yuge-hk/podux/blob/6a7ef34b713e344872f41271d6b98f2ce8a86253/development/frpc-mainline-race-baseline.md).

FRP merged [#5585: generation-local UDPProxy channels](https://github.com/fatedier/frp/pull/5585), merge commit `8d5e9d2b21e573f2bcac068a8a438571171c4920`, and [#5586: concurrent logger prefix updates](https://github.com/fatedier/frp/pull/5586), merge commit `93b52799bb7e6dad7ae1152c8b593eefcc17d595`. These repair references are preserved, but **Podux still embeds FRP v0.68.0**. No FRP upgrade, replacement, vendor patch or FRPC implementation change is made here. Old-version race failures remain Failed and may still keep the full workflow red; an upstream merge is not evidence that this pinned dependency has changed. Historical investigations are separated from the feature diff.

## Security review

Reviewed: AES-256-GCM random nonces and record-bound associated data; external key location/permissions and fail-closed restoration; hidden ciphertext collection field and locked rules; write-only custom API/JSON fields; strict body/argument validation and revision checking; existing authentication without query-token fallback on Tunnel routes; exec argv/no shell/minimal child environment; protected token files; one Wait owner; per-profile commands/state/retry; loopback-only bounded Metrics; bounded allowlisted events; safe frontend errors and no Token persistence/console output. The SQL ciphertext observation above was corrected with a regression, not dismissed because it was encrypted.

Fresh dependency checks remain **Failed for baseline findings**: 54 reachable Go advisory IDs; frontend 67 findings (42 high, 23 moderate, 2 low, zero critical). Both advisory-ID sets have no additions versus the recorded main baseline. govulncheck JSON mode exits zero even when findings exist; that is not treated as a clean scan. No broad dependency upgrade was made to hide these findings.

The final staged diff is checked for private-key/GitHub-token patterns, runtime databases/keys/binaries, generated web output and unintended files. Synthetic test Tokens are deliberately nonproduction. No real Token, encryption key, database, session credential or raw backend SQL dump belongs in the feature commit/evidence publication. Go formatting and local relative links are checked. Podux/cloudflared are Apache-2.0; implementation is independent and no cfui code is copied.

## Evidence retention and review footprint

Complete Phase 1–3 reports, original screenshots, raw race/audit/resource outputs and the inherited uncommitted control evidence were committed and pushed first to archive SHA `a7576af0b5d4610aca38f6708a28a8e4bfe00dcb` ([browse archive](https://github.com/yuge-hk/podux/tree/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development)). Copies also remain in the local validation delivery directory. Only after confirming the archive push were generated raw evidence and duplicate screenshots removed from this feature tip. No functional test source or capability was removed. Generated QA output folders are ignored to avoid accidental reintroduction; two current, explicitly labelled screenshots are versioned in docs/public/screenshots.

Current safe evidence is collected outside the source tree at `/root/公共/podux-phase4-validation`; credential-bearing disposable backend logs/data are not published. A later evidence-only archive will retain the curated current proof and exact CI metadata. The [final file manifest](cloudflared-phase4-files.txt) lists the complete proposed diff against upstream main, and the [English PR draft](cloudflared-pr-draft.md) describes scope, validation and outstanding acceptance.

## Remaining review decisions

Real Edge/public-origin tests remain Blocked until dedicated nonproduction credentials and an authorized test origin are available. Known FRP race/dependency audit findings remain outside the requested change. Normal native lifecycle coverage does not establish universal orphan-process handling on forced macOS/Windows parent crashes. Windows custom key overrides must currently remain on the same volume as the data directory; cross-volume paths fail closed. No 1–2 hour soak was performed. These limits must remain visible in the PR; this report does not claim completed production/live acceptance.

The default review proposal is one coherent backend/migration/UI/docs feature, with a separate FRP dependency change if maintainers authorize it. A backend-first/UI-second PR split is possible without discarding the Fork implementation, but no upstream PR or automatic merge is performed in this round.
