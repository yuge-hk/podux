# Draft: Manage local Cloudflare Tunnel processes in Podux

Adds authenticated management of multiple token-based cloudflared configurations through the existing PocketBase backend and React application. Each tunnel has independent lifecycle, retry policy, bounded redacted events and runtime status. Users can start, stop and restart one tunnel while others continue running.

The implementation follows existing domain/application/infrastructure/interfaces layers, adds an incremental collection migration, encrypts Tokens with AES-GCM using an external paired key file, and passes Tokens through protected temporary token files. Normal API responses exclude plaintext/ciphertext and internal credential paths. Updates use revisions. Unsupported local TLS overrides are not exposed. Existing FRPC models and runner are unchanged. This does not add Cloudflare account/API, DNS, ingress, Access or remote tunnel provisioning.

Backups must retain both the database and its external encryption key; missing keys with existing encrypted records fail closed instead of silently replacing the key. Platform-specific child cleanup guarantees are limited to verified behavior. Rollback removes the new collection and should only follow a complete paired backup and shutdown of managed processes.

Validation: local Go build/vet/offline tests and offline race suite, official cloudflared local rejection test, migration/restore and API tests passed. Frontend has 68 passing tests and 27 browser checks. Local FRPS TCP/UDP/reconnect/coexistence tests passed without race. Full lint retains pre-existing failures.

**Not merge-ready:** the opt-in real FRPC race regression fails in unchanged FRP 0.68.0; baseline dependency advisories remain. Native four-platform GitHub Actions has been prepared but not yet run, pending authorized Fork push. Authorized Edge and long soak tests remain unverified. See [validation report](cloudflared-phase3-validation.md) for evidence and current status. This is a description draft only; no PR has been created.
