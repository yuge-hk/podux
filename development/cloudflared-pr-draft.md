# PR title

feat: manage local Cloudflare Tunnel connectors

# Submission state

Prepared for a **Draft** PR. No upstream PR has been created; submission requires explicit user authorization. Live Edge acceptance remains Blocked.

# PR description

Podux currently manages FRPC clients but has no lifecycle controls for existing Cloudflare Tunnel connectors. This adds an authenticated Tunnel page and independent backend runner so users can save multiple token-based profiles, run them concurrently, and start/stop/restart each one separately.

## Behavior and scope

- Reuses the existing React layout, authentication, Radix components, bilingual resources and application/domain/infrastructure/interfaces structure.
- Adds a reversible PocketBase collection migration; the collection API stays locked and custom endpoints require the existing authentication. Profiles follow the existing shared authenticated-user management model.
- Supports version/capability-checked cloudflared parameters, loopback Metrics, observed readiness/protocol, bounded safe events, auto-start, bounded retries/backoff and revision-checked updates. Saving configuration does not silently restart a running profile.
- A fresh Podux start restores only autoStart profiles; manually started non-autoStart profiles remain stopped. Manual Stop leaves autoStart unchanged for the next application restart. Stored PIDs are never adopted.
- Does not add Cloudflare account/API, DNS, cloud-side Tunnel lifecycle, Public Hostname/Ingress, Access or topology management. FRPC source/models and the FRP dependency remain unchanged.

## Credentials and lifecycle

Tokens are write-only, AES-256-GCM encrypted with record-bound associated data and an external paired master key. Responses exclude plaintext/ciphertext/key paths. The UI never reveals an existing Token or stores a newly entered one in browser storage/URLs. A protected temporary --token-file is passed through an exec argument array; no shell is used. Child output is reduced to fixed allowlisted events, and the new collection's SQL logging is suppressed even in development mode so encrypted credential values do not enter debug logs.

Each profile owns its process, operation lock, generation, retry budget, Metrics reservation and bounded events. Each child has one Wait owner. Normal shutdown stops children before releasing the data-directory lock. macOS/Windows parent-crash/descendant guarantees are explicitly limited; Windows uses best-effort graceful signaling with forced direct-child termination on timeout.

## Deployment and recovery

Install cloudflared >=2025.4.0 separately; startup also verifies token-file/flag capabilities. The standard Docker image does not bundle it. Persist both the data directory and its external master key; a database-only backup cannot restore encrypted Tokens. Missing/wrong keys fail closed and are not silently replaced. Stop managed processes and retain a complete paired backup before rolling back the new collection. Master-key rotation is not implemented.

## Validation

The [Phase 4 report](cloudflared-phase4-validation.md) records exact commits, native Actions links, build/test counts, screenshots, security review and outstanding items. Locally, frontend build/type checking, 68 frontend tests, targeted lint, backend build/vet, 28 offline Go tests and their race run passed. Official cloudflared 2026.10.3 local invalid-token checks passed. Browser scenarios cover 18 general plus 9 real test-connector lifecycle checks; seven additional actual Podux restart/recovery checks passed in a disposable Linux environment.

Fresh [native run 39964271753](https://github.com/yuge-hk/podux/actions/runs/39964271753), executable-code SHA `583bbc19d1779431b1a0dec4ab47364a3d64a5a1`, passed Linux amd64/arm64, Windows amd64 and macOS arm64 Cloudflare unit/race/official-binary checks and functional FRPC regression. Overall CI Failed: the unchanged FRP v0.68.0 race reproduced on both Linux platforms and Windows; macOS did not reproduce it this run. The independent mainline control and full historical evidence are retained via immutable archive links rather than thousands of generated lines in this feature diff. Existing full-frontend lint/dependency audit findings are disclosed in the report, not claimed resolved.

Phase 5 repeated the frontend frozen install/build, 68 tests, targeted lint, Go vet/build, 28 offline unit tests and their race run, official-binary local-failure test, and documentation build successfully. It fetched upstream main (still `1136797`) and confirmed the 90-file scope and unchanged FRPC/FRP. No live test instance/profile names or dedicated test hostname were supplied, so real Edge and Dashboard/replacement/deletion checks remain Blocked. No plaintext Token was read, copied or uploaded. The existing native result above was rechecked rather than represented as a new run. See the Phase 5 section of the report for precise skips and limitations.

## Outstanding acceptance / review notes

- Real Cloudflare Edge connections and public-origin reachability are **Blocked** without authorized nonproduction credentials. Fixture readiness and invalid-token tests are not substitutes for live acceptance. No production Token was used.
- FRP upstream merged [UDPProxy fix #5585](https://github.com/fatedier/frp/pull/5585) and [xlog fix #5586](https://github.com/fatedier/frp/pull/5586). Podux still pins v0.68.0, so those fixes are not present here and prior failed checks are not reclassified as Passed. An FRP upgrade belongs in a separate change.
- This is a review draft, not a claim of complete live acceptance or merge readiness. No upstream PR has been created.

## Suggested review split

The feature can remain one coherent backend/migration/UI/docs change, with FRP dependency remediation kept separate. If maintainers prefer smaller reviews, split the backend/migration/tests first and UI/docs second while retaining the complete implementation in the Fork. Historical race investigations and generated evidence already live in a separate archive and need not be merged as product source.
