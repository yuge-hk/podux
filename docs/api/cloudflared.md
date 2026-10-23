# Local Cloudflare Tunnel backend

This module manages **local cloudflared processes for existing remotely managed
Tunnel tokens**. It does not manage Cloudflare accounts, DNS, Access, public
hostnames, ingress, or cloud-side Tunnel creation/deletion. The authenticated React UI is available at `/tunnels`; see the
[user guide](../guide/cloudflare-tunnels.md). FRPC services and collections are unchanged.

## Deployment and credentials

Install an official cloudflared binary >= 2025.4.0 separately. Podux does not
silently download, update, or execute an uploaded binary. Set
`PODUX_CLOUDFLARED_BINARY` to a server-controlled executable path, or install
`cloudflared` on PATH. Version/help/parser probes run at Podux startup with
bounded output and timeouts; capabilities remain unavailable if they fail.
Restart Podux after replacing the binary. Missing cloudflared does not disable
FRPC or configuration APIs.

By default the AES-256-GCM key is stored at
`<absolute-PocketBase-data-directory>.cloudflared-secrets/master.key`, a sibling
of the data directory, outside SQLite and PocketBase's data directory. Override
with `PODUX_CLOUDFLARED_KEY_FILE`; paths within the data directory are rejected,
including paths resolved through symlinks. On Windows, the current path validation
requires a custom key location on the same volume as the data directory;
cross-volume paths fail closed. Protect that location and mount it
persistently in containers. Podux runs as the existing OS user. Unix key files
must have no group/other permissions; newly created files use 0600. Windows
creates a protected DACL for the current user, SYSTEM and Administrators.

**Back up the master key with its associated data-directory backup, storing the
key separately. PocketBase's built-in database/data backup alone is insufficient.**
A missing key with existing encrypted tokens is never regenerated. An invalid or
wrong key is never overwritten. Restore the matching key and restart Podux.
Configuration reads and deletion remain available when credentials cannot be
opened; secret writes and starts fail closed. Deleting all profiles does not
rotate an existing key. Key rotation is not implemented in this phase.

Tokens are locally validated as base64-encoded JSON with nonempty `a`, `t`, `s`
string properties. This is not a Cloudflare authentication check. Tokens are
AES-GCM encrypted with a random nonce and versioned per-record associated data.
Each run gets a private temporary token file passed with `--token-file`; the
Token never travels in argv or inherited environment variables. The explicit
empty configuration file prevents implicit local cloudflared config discovery.
Runtime files are removed after Wait completes and on the next locked startup.
Runtime paths and key paths are never returned by the API. The new collection's SQL debug/performance callbacks are suppressed before serving requests so even
encrypted Token values cannot appear in interpolated development-mode SQL logs;
logging for other collections remains unchanged. Local OS administrators
and the application account remain trusted. Protect disks, backups and host access.

## REST API

All routes below use the existing `requireAuth` and the Podux Bearer token in
`Authorization`. URL token authentication is not enabled. Profiles follow the
existing shared authenticated-user management model; no per-owner isolation is
introduced. PocketBase collection API rules are all locked; only trusted
superusers may bypass those rules.

| Method | Path under `/api/cloudflared` | Result |
| --- | --- | --- |
| GET | `/capabilities` | Available/version/build, supported flags, `tokenFile`, `tlsLocalOverride` |
| GET | `/tunnels` | `{items: [...]}`; all saved profiles |
| POST | `/tunnels` | Create, 201; requires `token` |
| GET | `/tunnels/{id}` | Non-secret profile |
| PUT | `/tunnels/{id}` | Full configuration update with current `revision`; `{tunnel, restartRequired}` |
| DELETE | `/tunnels/{id}?revision=N` | Stop, then delete, 204 |
| POST | `/tunnels/{id}/start` | Schedule independent start, 202 status |
| POST | `/tunnels/{id}/stop` | Await stop/Wait, 202 status |
| POST | `/tunnels/{id}/restart` | Await old process exit, schedule new run, 202 status |
| GET | `/tunnels/{id}/status` | Current runtime status |
| GET | `/tunnels/{id}/logs?after=N` | `{items: [{cursor,message}], epoch}` |

15-character lowercase alphanumeric IDs are validated. Invalid configurations
return 400, unauthenticated requests 401, missing profiles 404, revision/runtime
conflicts 409, unavailable binary/credentials 503, and other internal failures a
fixed generic 500 message. JSON bodies are limited to 32 KiB; unknown fields,
including unsupported TLS options, are rejected. No raw internal errors are
returned.

Create/update fields (full config is required; omitted numeric fields are zero):

| Field | Accepted values |
| --- | --- |
| `name`, `description` | Nonempty name <=120 bytes; description <=2000 bytes |
| `autoStart`, `autoRestart` | Booleans |
| `maxRestartAttempts` | 0..100 additional attempts after the initial launch |
| `protocol` | `auto`, `quic`, `http2` |
| `region` | Empty/global or `us` |
| `postQuantum` | Boolean; true requires explicit `quic` |
| `connectionRetries` | 0..20; cloudflared's own connection retry budget |
| `gracePeriodSeconds` | 1..120 |
| `edgeIPVersion` | `auto`, `4`, `6` |
| `edgeBindAddress` | Empty or a literal IP matching a fixed IP version; no zones |
| `metricsPort` | 0 for automatic allocation or 1..65535; always 127.0.0.1 |
| `extraArgs` | Array of `{name,value}`; only `loglevel=info`, `ha-connections=1..4`; no duplicates |
| `token` | Write-only; required on create; omitted/null on update preserves old value; empty rejects |
| `revision` | Required on update; starts at 1 and increments on every write |

Tokens, ciphertext and key paths never appear in profile responses; only
`hasToken` indicates presence. Saving a running profile does not mutate its run
snapshot. `restartRequired` compares saved and running revisions. Rotation takes
effect on the next explicit Start/Restart; a supervisor already running uses its
original configuration/token snapshot, including automatic retries. Use Restart
immediately after rotating a running profile. Concurrent writes/commands are
serialized per profile by the application service; long stops do not hold its
shared state lock. Each runner also has a separate operation lock.

## Runtime and recovery

States: `stopped`, `starting`, `running`, `backoff`, `stopping`, `failed`.
`running` requires a successful local Metrics `/ready` check, not merely successful
OS process creation. `ready`, allowlisted `connections`, actual observed
`protocol` (otherwise `unknown`), supervisor `generation`, `revision`, retry
count/time and a fixed safe `lastError` are returned. Metrics use an isolated
HTTP client with no proxy, redirects or arbitrary remote URLs. Only a known local
listener is queried. Metrics payload reads are bounded; no raw payload is exposed.

Metrics are always enabled on loopback for readiness/connection collection;
there is no unverified "disable Metrics" toggle. A manager port reservation and
OS bind check catch fixed conflicts. Auto ports are OS-selected, with a bounded
reservation-to-child handoff window. A bind failure is reported through child
failure and the configured supervisor retry policy. Each retry reselects an auto
port. An external process can still race the handoff; no claim of atomic socket
inheritance is made.

Every child has exactly one Wait owner. Manual Stop cancels backoff and sends a
graceful signal, then kills after grace + 1 second. Pipe cleanup has a 2-second
WaitDelay. Restart waits before starting a replacement. An unsolicited nonzero
exit/start failure may restart; exit code zero stops without automatic restart.
Backoff starts at 1 second, doubles to a 60-second cap, and resets the incident
budget after 5 minutes of process lifetime. `maxRestartAttempts=0` disables extra
attempts. Generation distinguishes explicit start/restart supervisor sessions;
automatic attempts use that session's generation.

At Podux startup only `autoStart=true` profiles are started, with fresh runtime
state and retry budgets. A manually started non-autoStart profile stays stopped
after restart. A manual Stop preserves autoStart, so the next Podux restart can
start it again. Stored PIDs are never adopted. A data-directory OS lock prevents
two managers sharing one directory. Stop Podux before rolling back the migration;
Down drops only the Tunnel collection, not the external master key.

Normal Podux termination/restart cancels recovery and stops managed children
before releasing the directory lock. Unix children have separate process groups;
Linux additionally sets Pdeathsig and has a dedicated parent-death test. macOS
has no equivalent parent-death guarantee. Windows uses a new process group,
best-effort CTRL_BREAK and direct-child forced termination; no Job Object or
parent-crash/descendant cleanup guarantee is claimed. Cross-compilation is not
runtime validation: run the supplied platform CI tests on their native runners.
After a forced parent crash on macOS/Windows, operators must inspect/stop any
remaining children before restarting Podux. This phase does not claim protection
against every abrupt host/process failure.

Logs are a 500-event in-memory ring per profile with monotonic cursors. They are
not persistent raw cloudflared logs. Only fixed lifecycle/error/listener events
and strictly parsed connection protocols are emitted. Arbitrary child output is
discarded to avoid exposing unknown secrets or remote origin configuration.
Cursor gaps indicate truncation. The optional additive `epoch` identifies a
manager lifetime; new clients reset their cursor and reload when it changes.
Older clients may ignore it, and new clients retain a cursor-regression fallback
for older servers. Rings are lost on Podux restart and profile deletion. Error detail is intentionally limited.

`tlsLocalOverride` is always false. `--no-tls-verify` affects origin validation
under applicable local origin configuration, and is not a verified override for
Token-delivered remote ingress. It is neither accepted nor advertised as a
configurable local capability. Cloudflare's Edge TLS verification stays enabled.

## Verification and attribution

Unit/process/API/migration tests use temporary directories, synthetic tokens and
Go helper processes (no shell). For optional official binary parser, token-file
and local-failure checks:

```sh
PODUX_TEST_CLOUDFLARED=/path/to/cloudflared go test ./internal/infrastructure/cloudflared -run TestOfficialBinary -v
```

These checks do not prove a successful connection to a real account. No production
Token is required or committed. See the repository file `development/cloudflared-validation.md` for test evidence.
Implementation is independent; no cfui source was copied. Podux and cloudflared
are Apache-2.0; cfui is MIT (Copyright 2025 Dockers of X). This change does not
redistribute cloudflared. If future packaging redistributes it, review the pinned
release's full third-party license/NOTICE obligations.

References: [official run parameters](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/configure-tunnels/run-parameters/),
[2025.4.0 token-file implementation](https://github.com/cloudflare/cloudflared/blob/2025.4.0/cmd/cloudflared/tunnel/subcommands.go),
[origin parameters](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/configure-tunnels/origin-parameters/).
