# Cloudflared backend validation

Baseline: Podux `1136797a36a2f1186055278351f55e28b907d6d5`.
Reference-only cfui: `506bc8f7280ca70d6f1fd50402c197079e7dac3c`.
No cfui code copied. No production account/Token, PR or deployment was used.

## Executed on Linux amd64

- `go test ./internal/infrastructure/credentials ./internal/infrastructure/cloudflared ./internal/infrastructure/persistence ./migrations`: passed during 1A; repository/migration behavioral tests added subsequently and passed.
- `go test -race ./internal/infrastructure/... ./internal/application/tunnel ./internal/interfaces/http ./migrations`: passed (including helper-process lifecycle and Linux parent death).
- `go vet ./...`, `go test ./...`, `go build ./...`: passed with the existing frontend built/copied into ignored `pb_public`.
- Existing `site` frozen dependency install and production build: passed. No React source or package changes.
- `pnpm --dir docs run docs:build`: passed after correcting a link outside the documentation root.
- `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=darwin GOARCH=arm64 go build ./...`: passed; updated runner/credential test suites also cross-compiled. Native execution was not performed on these systems.
- Full executable smoke test in a disposable data directory: actual `serve`, existing authentication, capabilities, encrypted CRUD, real cloudflared invalid-token failure, safe logs, deletion and SIGTERM shutdown passed.
- Existing frontend lint: failed with **26 errors and 3 warnings** in unchanged baseline files (primarily explicit `any`, unused variables and hooks). No unrelated frontend fixes made.

## Official binary checks

Downloaded official release assets to `/tmp` only:

| Version | SHA-256 | Result |
| --- | --- | --- |
| 2025.4.0 | `df13e7e0a027f648c410b5cc701fbcff028724d0e93209796cdbb79ec38695d4` | parser/token-file/local failure tests passed |
| 2026.10.3 | `053b80b0a23031a038fdbf3c073f59f82b8f1f152aa67971d7383ffafc0dbab0` | parser/token-file/local failure tests passed |

2026.10.3 digest matches the GitHub release asset metadata. The older release did
not provide a digest in that metadata; its computed hash records the tested bytes,
not an independent signature verification.

Run the opt-in test with `PODUX_TEST_CLOUDFLARED=/path/to/binary go test
./internal/infrastructure/cloudflared -run TestOfficialBinary -v`.
It exercises auto/quic/http2, region, edge IP/bind, retries=0, grace period,
post-quantum and ha-connections parser acceptance; real file credential input;
invalid-token failure; child cleanup and safe logs. It does not authenticate a
real Tunnel or prove successful Edge connectivity/fallback/PQ negotiation.

TLS local override is deliberately unavailable. Unsupported/injection-oriented
extra parameters and unknown API fields are rejected. Metrics readiness and
allowlisted connection gauges are tested against real loopback HTTP listeners in
Go helper children; conflict and per-profile isolation tests require no cloud.

## Limits

Linux lifecycle and parent-death tests are actual native executions. macOS/Windows
are cross-compiled locally; native runtime execution is deferred to the added CI
matrix, and no parent-crash guarantee is claimed there. No production FRPC server
connection was available; its source and collections are unchanged, and existing
frontend/full backend builds plus migration compatibility tests passed.
