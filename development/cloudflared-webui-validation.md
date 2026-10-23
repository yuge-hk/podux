# Cloudflare Tunnel WebUI — Phase 2 implementation and evidence

> Historical evidence. Current acceptance: [Phase 4](cloudflared-phase4-validation.md).
> Raw artifacts and the original report are preserved in the [immutable archive](https://github.com/yuge-hk/podux/tree/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development); generated evidence is excluded from the feature diff.

Branch: `feat/local-cloudflare-tunnels`. Phase 1 and Phase 2 changes remain
uncommitted. No push, PR, production deployment or production Cloudflare Token.
The [Phase 2 file manifest](https://github.com/yuge-hk/podux/blob/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development/cloudflared-webui-files.txt) distinguishes this work
from the pre-existing Phase 1 backend changes in the same worktree.

## Delivered UI

Authenticated route: `/tunnels`, appended after the existing FRPC navigation.
The page follows the existing controller/view/hooks organization:

- `site/src/pages/Tunnels/index.tsx`: dialogs, save/delete and explicit restart orchestration.
- `TunnelsPage.view.tsx`: capabilities, responsive profile cards and safe error display.
- `TunnelForm.tsx`: unified create/edit form and conflict reload confirmation.
- `TunnelDetail.tsx`: runtime/configuration revisions, generation, health and logs.
- `components/TunnelActions.tsx`, `TunnelStatus.tsx`, `TunnelLogViewer.tsx`.
- `api.ts`: shared typed contracts, authenticated requests and frontend validation.
- `hooks.ts`: independent operation guards and bounded visibility-aware polling.

Radix Themes, existing MainLayout/PageHeader/FormItem/EmptyState/Loading, Sonner
and existing i18next/auth are reused. No alternate framework/auth/state system.
Vitest, Testing Library and jsdom are development dependencies only. Browser QA
uses an externally installed Playwright Core and existing Chromium executable;
no large E2E framework was added to project dependencies.

## Config fields and contract

All configuration fields are explicitly submitted even when advanced settings
are collapsed. Runtime/unknown secret response fields are excluded from writes.

| UI field | Backend field / values |
| --- | --- |
| Name, description | `name`, `description`; UTF-8 byte limits checked |
| Auto Start / Auto Restart | `autoStart`, `autoRestart` |
| Max Restart Attempts | `maxRestartAttempts`, 0..100; zero is no extra retries |
| Protocol / Region | `protocol`: auto/quic/http2; `region`: empty/us |
| Edge IP / bind address | `edgeIPVersion`: auto/4/6; validated literal `edgeBindAddress` |
| Forced post-quantum | `postQuantum`; only explicit QUIC permits enabling |
| Internal retries | `connectionRetries`, 0..20 |
| Grace Period in seconds | `gracePeriodSeconds`, 1..120 |
| Loopback Metrics port | `metricsPort`, 0..65535; zero is automatic, not disabled |
| Explicit Info logging | `extraArgs: [{name:loglevel,value:info}]` |
| HA connections | `extraArgs: [{name:ha-connections,value:1..4}]` |
| Editing concurrency | `revision` in PUT; revision query in DELETE |
| Token | Password input; write-only; omitted on blank edit |

No arbitrary command/argument input, TLS bypass switch, executable upload,
account/DNS/Ingress/Access management, or Cloudflare topology was introduced.

## Safety and concurrency

Tokens exist only in form memory and the necessary authenticated write request;
no browser storage, URL, console logging, reveal or mask-as-token submission.
On success/close the form clears the input. Unknown API errors are replaced with
safe localized messages; raw request/response/error objects are not logged.

Revision 409 preserves draft fields and new Token; reload requires explicit
confirmation. Saving a running config offers Restart now/later and never restarts
implicitly. Browser lifecycle QA verified that the old run revision is preserved
until an explicit restart and that the new saved revision then runs.

Two minimal backend corrections were required and tested:

1. Application mutation/command locks are per profile. Stop/Delete no longer wait
   under the shared service mutex; a blocked Stop A permits status/Start B.
   Same-profile commands/configuration writes remain serialized and revision CAS
   remains in place. A blocking-runner race regression test covers this.
2. Log responses add `epoch`, identifying manager lifetime. Existing paths,
   authentication and `items` are unchanged; older clients can ignore it. New
   clients reset on epoch change even if a restarted ring has advanced beyond the
   old cursor. Old servers remain supported via cursor-regression snapshots.
   Go API/manager tests and frontend epoch-reset tests cover the addition.

Neither change modifies the database schema, encryption, authentication model or
FRPC behavior. No additional migration was needed in Phase 2.

## Executed checks (Linux amd64)

| Command / check | Actual result |
| --- | --- |
| `pnpm --dir site run test` | **68 passed**, 2 test files (28 API/validation + 40 UI/polling tests) |
| `pnpm --dir site exec tsc -b` | Passed |
| `pnpm --dir site exec eslint src/pages/Tunnels src/test src/App.tsx src/layouts/MainLayout.tsx vitest.config.ts` | Passed, no new errors/warnings |
| `pnpm --dir site run build` | Passed, TypeScript + Vite production build |
| `pnpm --dir site run lint` | Existing **26 errors + 3 warnings**, unchanged baseline files |
| `pnpm --dir docs run docs:build` | Passed |
| `go vet ./...` | Passed |
| `go test -race ./...` | Passed, including migration and process tests |
| `go build ./...` | Passed after copying built frontend to ignored `pb_public` |
| `git diff --check` | Passed |

The frontend test cases cover empty/multiple profiles, required create input,
create/edit/Token replacement/preservation, mask rejection, conflicts, delete
confirmation/failure/success, independent Start/Stop/Restart, accepted responses,
explicit restart prompts, capability degradation, advanced field ranges/IP/PQ,
forbidden arguments, all six runtime states, unknown protocol, safe 401/errors,
log increments/truncation/epoch reset/failure/copy/scroll, polling unmount,
background visibility, non-overlap, secret display/storage and bilingual key parity.

## Native Chromium checks and screenshots

[General browser report](https://github.com/yuge-hk/podux/blob/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development/screenshots/browser-checks.json): **18 checks passed**
against the real disposable Podux database/API and official cloudflared 2025.4.0
using only a synthetic invalid Token. Tested actual login/route guard, empty
state, create/edit/reload persistence, mobile form/start/restart/logs, confirmed
delete, navigation and storage/console/URL safety. Its Stop UI state and missing
binary scenarios use explicitly declared fixtures. Invalid-token failure is not
a successful Cloudflare connection.

[Fixture lifecycle report](https://github.com/yuge-hk/podux/blob/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development/screenshots/lifecycle-browser-checks.json): **9 checks
passed** through real browser -> real Podux -> actual child processes. A
[test-only Go connector](../site/qa/testdata/cloudflared/main.go) supplies local
readiness/Metrics without any Cloudflare networking. Verified concurrent children,
different Metrics addresses, collected gauge, mobile Stop A while B runs,
replacement generation, saved-vs-running revisions, Token replacement + Restart
now, real 409 conflict and epoch/event delivery.

Inspected desktop light/dark and 390px/320px layouts. Forms/dialogs scroll, Token
inputs remain accessible, controls wrap, and no document horizontal overflow was
observed at either mobile width. Existing mobile bottom padding increases to
88px for the appended two-line navigation label. Chinese/English UI keys match;
actual Chinese dark-mode and English light-mode screenshots were inspected.
These are Chromium viewport checks, not physical iOS/Android device tests.

- [Desktop light, official binary / saved profiles](https://github.com/yuge-hk/podux/blob/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development/screenshots/cloudflare-desktop-light.png)
- [Mobile advanced form](https://github.com/yuge-hk/podux/blob/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development/screenshots/cloudflare-mobile-form.png)
- [Mobile dark Chinese](https://github.com/yuge-hk/podux/blob/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development/screenshots/cloudflare-mobile-dark-zh.png)
- [320px layout](https://github.com/yuge-hk/podux/blob/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development/screenshots/cloudflare-mobile-320.png)
- [Mobile actual fixture processes running](https://github.com/yuge-hk/podux/blob/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development/screenshots/cloudflare-fixture-mobile-running.png)
- [Mobile actual safe fixture events](https://github.com/yuge-hk/podux/blob/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development/screenshots/cloudflare-fixture-mobile-events.png)
- [Desktop dark Chinese, fixture processes](https://github.com/yuge-hk/podux/blob/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development/screenshots/cloudflare-fixture-desktop-dark-zh.png)
- [Existing FRPC dashboard](https://github.com/yuge-hk/podux/blob/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development/screenshots/frpc-dashboard-regression.png)

Fixture screenshots explicitly represent test connector readiness, not Cloudflare
Edge connectivity. Synthetic profile names/descriptions remain user data, not UI
translations.

## Reproducing browser QA

Use an isolated temporary data directory, test-only credentials and server ports.
Scripts require `PODUX_QA_DISPOSABLE=1` because they reset Tunnel records in that
explicitly disposable local backend. Never point them at an existing user instance.
Install Playwright Core outside the repository and use a local Chromium binary.
Example sequence (replace paths with your own temporary locations):

```sh
# Build site and copy site/dist to pb_public before building Podux.
pnpm --dir site run dev --host 127.0.0.1 --port 5173
# In another terminal, run the built Podux on 127.0.0.1:8090 with --dir /tmp/qa/data.
# Set PODUX_CLOUDFLARED_BINARY to an official >=2025.4.0 binary for general checks.
PODUX_QA_DISPOSABLE=1 PLAYWRIGHT_MODULE=/tmp/qa/node_modules/playwright-core/index.mjs \
  node site/qa/cloudflared-browser.mjs

# Stop that temporary Podux, build the fixture and restart the same disposable
# backend with PODUX_CLOUDFLARED_BINARY=/tmp/qa/connector-fixture.
go build -o /tmp/qa/connector-fixture ./site/qa/testdata/cloudflared
PODUX_QA_DISPOSABLE=1 PLAYWRIGHT_MODULE=/tmp/qa/node_modules/playwright-core/index.mjs \
  node site/qa/cloudflared-lifecycle-browser.mjs
```

The lifecycle script expects the general script's disposable initialized test
account. Override `CHROMIUM_PATH` for your browser. The scripts create reports and
screenshots under `development/screenshots`, and never print auth/runtime tokens.

## FRPC regression and remaining limits

Native browser checks rendered the authenticated original server list, proxy
list and dashboard/network topology with real APIs against an **empty** disposable
FRPC database, with no page errors. Source comparison confirms original FRPC
pages, topology, runner and domain models are unchanged. Original navigation
ordering is preserved; Cloudflare is appended. Existing login/setup guards are
reused and an anonymous `/tunnels` request reached the original login page.
Migration compatibility tests and full Go build/tests passed.

No real FRP server, populated production FRPC database or traffic forwarding was
available; those were not claimed as tested. No successful real Cloudflare Tunnel,
public route/origin reachability, PQ negotiation or protocol fallback was tested.
Native macOS/Windows lifecycle/parent-crash behavior remains at the Phase 1
validation boundary. Firefox, Safari, physical devices and real token rotation
against Cloudflare remain unverified.

For Phase 3: review the isolated backend fixes and external-key backup guidance,
run native platform CI and an authorized non-production Cloudflare/FRP smoke test,
review screenshots and the file manifest, then prepare an upstream PR description
with the exact test evidence and limits. Do not publish or push without the next
explicit instruction.
