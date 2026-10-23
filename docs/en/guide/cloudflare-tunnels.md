# Local Cloudflare Tunnel management

Open **Cloudflare Tunnel** in the authenticated navigation. This page manages
local cloudflared connectors for existing Tunnel tokens, without account, DNS,
Access, cloud-side Tunnel creation or ingress management.

Install cloudflared >= 2025.4.0 on the Podux server and restart Podux. The page
reports detected version/build and capabilities. Profiles remain editable when
the binary is unavailable, while Start/Restart are disabled. No binary upload or
automatic download is offered. Back up the external master key together with the
associated data backup: see the [backend security contract](../../api/cloudflared.md).

Create a named profile with an existing runtime Token. On edit, the password
field stays empty: blank preserves the saved Token, explicit input replaces it.
Tokens are never revealed or saved in browser storage or URLs. Revision conflicts
keep your draft. Reloading requires confirmation and discards unsaved changes.
Saving a running profile offers Restart now/later; it does not restart silently.

Auto Start applies when Podux starts. Auto Restart applies to abnormal exits;
zero additional attempts means no extra retries. Advanced settings cover protocol,
global/US region, Edge IP version/bind address, forced post-quantum with explicit
QUIC, internal connection retries, shutdown seconds and loopback Metrics port
(0 selects a port automatically). Extra options are limited to Info logging and
1–4 HA connections. There is no unsupported TLS bypass switch.

Accepted operations are not proof of connectivity. Runtime states and observed
protocol are polled about every five seconds, pausing in background tabs. Readiness
means connector health, not end-to-end route/origin availability. Details show
connections, generations, saved/running revisions and retry/error information.

While details are open, safe event logs poll incrementally about every three
seconds. Copy, manual refresh and auto-scroll controls are available. At most
500 events are retained; truncation is reported and epoch changes reset the cursor (with a regression
fallback for older servers). Logs may
vanish after Podux restarts. Raw process output is not exposed. Confirmed deletion
stops the child first and removes the profile/Token only on success.

## Installation requirements

Use the [official Cloudflare downloads](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/downloads/)
for the server OS/CPU. The minimum is 2025.4.0; startup also checks token-file and
flag-parser support. Native CI covers Linux amd64/arm64, Windows amd64 and macOS
arm64 local execution, with official binary checks pinned to 2026.10.3. Invalid
Token tests do not establish successful real-account or public-origin connectivity.

The executable must be runnable by the Podux account. Persist both the writable
data directory and the external key location, and permit outbound Edge traffic
for the selected protocol. Server-controlled path examples:

```sh
PODUX_CLOUDFLARED_BINARY=/usr/local/bin/cloudflared ./podux serve --dir /path/to/pb_data
```

```powershell
$env:PODUX_CLOUDFLARED_BINARY = 'C:\Tools\cloudflared.exe'
.\podux.exe serve --dir 'C:\Podux\data'
```

The standard Podux Docker image does not bundle cloudflared. Supply a compatible
executable inside your custom container and persist both data and the external
key. A pb_data-only volume does not include the default sibling key directory;
`PODUX_CLOUDFLARED_KEY_FILE` can select a separate persistent key file. Current
Windows validation requires the key and data directory on the same volume;
cross-volume overrides fail closed. Enter Tokens
through the authenticated page, not shell arguments, images or deployment files.
Use HTTPS when accessing Podux over a network.

## Screenshots

These isolated test-connector screenshots demonstrate UI/process behavior; they
are not evidence of a live Cloudflare Edge connection.

![Desktop dark theme, Chinese locale](/screenshots/cloudflare-tunnels-desktop.png)

<img src="/screenshots/cloudflare-tunnels-mobile.png" alt="Independent mobile connector controls" width="390" />

## Verification limits

Native CI covers normal lifecycle, cleanup, credential permissions and runner
behavior. macOS/Windows do not guarantee descendant cleanup after every forced
parent crash; Windows graceful signaling is best-effort with direct-child kill
on timeout. Safe events are bounded and nonpersistent. Token updates require an
explicit restart to affect a live process; master-key rotation is not provided.
Accepted Start and connector readiness are not proof of public-origin reachability.

Without authorized nonproduction credentials, real Edge/public-origin acceptance
is **Blocked**. See the [Phase 4 acceptance report](https://github.com/yuge-hk/podux/blob/feat/local-cloudflare-tunnels/development/cloudflared-phase4-validation.md)
for actual results and remaining constraints.
