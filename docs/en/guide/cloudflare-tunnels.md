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
