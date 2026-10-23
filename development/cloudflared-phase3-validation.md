# Cloudflared Phase 3 validation history

Current acceptance is recorded in [Phase 4](cloudflared-phase4-validation.md).
The [complete original reports and raw evidence](https://github.com/yuge-hk/podux/tree/a7576af0b5d4610aca38f6708a28a8e4bfe00dcb/development) are preserved at immutable archive commit `a7576af0b5d4610aca38f6708a28a8e4bfe00dcb`, outside the product feature diff. No historical test has been reclassified because an upstream fix later merged.

## Feature native executions

| Test SHA | GitHub Actions | Actual result |
| --- | --- | --- |
| `cb2f4aeeeb8420b2f496b9f4d098c8717d6e4fce` | [39844432704](https://github.com/yuge-hk/podux/actions/runs/39844432704) | Cloudflare suites/official binary passed on all four native platforms; FRPC combined race regression failed on Linux amd64/arm64 and Windows, passed once on macOS |
| `c2e6e7d5328add30eec04177a6630cd3e7e42454` | [39845439391](https://github.com/yuge-hk/podux/actions/runs/39845439391) | Cloudflare suites, official binary and separate FRPC functional regression passed on all four platforms; separate FRPC race regression failed on all four |

Both runs used Go 1.25.5 and unchanged FRP v0.68.0 on ubuntu-24.04 amd64, ubuntu-24.04-arm arm64, windows-2025 amd64 and macos-15 arm64. These were native executions, not cross compilation. The official cloudflared 2026.10.3 assets were SHA-256 verified. Invalid-token checks exercised local failure, not authorized Cloudflare Edge connectivity. Offline suites explicitly skipped the two opt-in official/FRPS tests, which were run in separate required steps. Race was executed on all platforms; none was skipped.

## Independent FRPC mainline control

The control starts directly from official Podux main `1136797a36a2f1186055278351f55e28b907d6d5`, without Cloudflare business code. Only the FRPC scenario and CI helpers were added in temporary test commit `6ff9263f6242e5a95b4e127b844642364923a13e`.

[Control run 39847653720](https://github.com/yuge-hk/podux/actions/runs/39847653720) passed functional tests everywhere. Race failed on Linux amd64, Windows amd64 and macOS arm64, while that single Linux arm64 trial passed. The Service controller and xlog access pairs matched feature-branch reports. A passing single trial does not prove absence of a scheduling-dependent race.

[Complete control report and stacks](https://github.com/yuge-hk/podux/blob/6a7ef34b713e344872f41271d6b98f2ce8a86253/development/frpc-mainline-race-baseline.md). The temporary local control worktree was cleaned and its remote evidence retained.

A separate, temporary FRP v0.71.1/Go 1.26.10 experiment did not modify this feature branch. [Run 39851903747](https://github.com/yuge-hk/podux/actions/runs/39851903747), test SHA `8ae7bd98818946262a3f830d3279d03eb18e9047`, retained all 12 predetermined native race trials: the old Service controller pair did not recur, xlog appeared in 9, UDPProxy in 3. [Report and full evidence](https://github.com/yuge-hk/podux/blob/057ae61dd2ddfac5aa7155ebf5c6096ca456252b/development/frp-0711-validation.md). That experiment is not a dependency upgrade in this PR.

## Subsequent upstream repair

FRP merged [UDPProxy fix #5585](https://github.com/fatedier/frp/pull/5585) and [xlog fix #5586](https://github.com/fatedier/frp/pull/5586). Podux still pins **v0.68.0**. Those upstream changes are not automatically incorporated here; existing FRPC race failures remain Failed and the detector is not disabled. Dependency remediation belongs in a separate authorized change.

## Other evidence and limits

The archive retains paired database/key restoration and incremental migration tests, 68 frontend tests, 27 browser checks, Linux short resource/lifecycle measurements, full dependency audit comparisons and original failure/fix notes. Historical full lint and baseline vulnerability findings remain disclosed; no security-clean claim is made. The first disposable resource experiment saw an unrelated first-run installer child; later tests preseeded the test account rather than claiming that existing installer behavior was fixed.

Authorized real Edge/public-origin validation and the 1–2 hour soak were not executed in Phase 3. See Phase 4 for current Passed/Failed/Skipped/Blocked statuses and the independent SQL-log hardening regression.
