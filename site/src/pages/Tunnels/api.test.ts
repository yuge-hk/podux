import { afterEach, describe, expect, it, vi } from "vitest";
import {
  configuration,
  defaults,
  errorKey,
  tunnelAPI,
  TunnelAPIError,
  validConfig,
  type TunnelConfig,
} from "./api";
const valid = (): TunnelConfig => ({ ...defaults, name: "Local Tunnel" });
afterEach(() => vi.unstubAllGlobals());
describe("configuration validation", () => {
  it("accepts defaults and explicit QUIC/PQ", () => {
    expect(validConfig(valid())).toBe(true);
    expect(validConfig({ ...valid(), protocol: "quic", postQuantum: true })).toBe(true);
  });
  it.each([
    { name: "" },
    { protocol: "http2", postQuantum: true },
    { region: "cn" },
    { edgeBindAddress: "1.2.3.999" },
    { edgeBindAddress: "::1", edgeIPVersion: "4" },
    { edgeBindAddress: "127.0.0.1", edgeIPVersion: "6" },
    { metricsPort: 65536 },
    { connectionRetries: 21 },
    { gracePeriodSeconds: 0 },
    { maxRestartAttempts: -1 },
    { maxRestartAttempts: 101 },
    { metricsPort: 0.1 },
    { extraArgs: [{ name: "token", value: "secret" }] },
    { extraArgs: [{ name: "ha-connections", value: "5" }] },
    { extraArgs: [{ name: "loglevel", value: "debug" }] },
  ])("rejects invalid network/range/extra argument combination %j", (patch) => {
    expect(validConfig({ ...valid(), ...patch } as TunnelConfig)).toBe(false);
  });
  it("accepts IPv6, auto Metrics and zero additional restarts", () => {
    expect(
      validConfig({
        ...valid(),
        edgeBindAddress: "::1",
        edgeIPVersion: "6",
        metricsPort: 0,
        maxRestartAttempts: 0,
      })
    ).toBe(true);
  });
  it("strips runtime and unexpected secret fields from config payload", () => {
    const config = configuration({
      ...valid(),
      id: "abc",
      revision: 1,
      hasToken: true,
      ciphertext: "unwanted",
    } as never);
    expect(config).not.toHaveProperty("ciphertext");
    expect(config).not.toHaveProperty("revision");
  });
});
describe("authenticated API", () => {
  it("uses Authorization, keeps blank edit Token omitted and includes full config/revision", async () => {
    const fetch = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ tunnel: {}, restartRequired: true }), { status: 200 })
      );
    vi.stubGlobal("fetch", fetch);
    await tunnelAPI.update(
      { ...valid(), id: "aaaaaaaaaaaaaaa", revision: 7, hasToken: true },
      valid(),
      ""
    );
    const [url, init] = fetch.mock.calls[0];
    expect(url).toBe("/api/cloudflared/tunnels/aaaaaaaaaaaaaaa");
    expect(url).not.toContain("token=");
    expect(init.headers).toHaveProperty("Authorization");
    const body = JSON.parse(init.body);
    expect(body).not.toHaveProperty("token");
    expect(body.revision).toBe(7);
    expect(body.metricsPort).toBe(0);
    expect(body.gracePeriodSeconds).toBe(30);
  });
  it("replaces Token only on explicit input without browser storage", async () => {
    const storage = vi.spyOn(Storage.prototype, "setItem");
    const fetch = vi.fn().mockResolvedValue(new Response("{}", { status: 200 }));
    vi.stubGlobal("fetch", fetch);
    await tunnelAPI.update(
      { ...valid(), id: "aaaaaaaaaaaaaaa", revision: 1, hasToken: true },
      valid(),
      "synthetic-new-token"
    );
    expect(JSON.parse(fetch.mock.calls[0][1].body).token).toBe("synthetic-new-token");
    expect(storage).not.toHaveBeenCalled();
  });
  it("handles 202 as an accepted runtime response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response('{"phase":"starting","ready":false}', { status: 202 }))
    );
    expect((await tunnelAPI.control("a", "start")).ready).toBe(false);
  });
  it.each([400, 401, 404, 409, 500, 503])("does not echo raw error bodies (%s)", async (status) => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response('{"error":"SECRET raw process output"}', { status }))
    );
    try {
      await tunnelAPI.list();
      throw Error("expected failure");
    } catch (error) {
      expect(error).toBeInstanceOf(TunnelAPIError);
      expect(String(error)).not.toContain("SECRET");
      expect(errorKey(error)).toMatch(/^tunnels.errors./);
    }
  });
  it("distinguishes revision conflict", () =>
    expect(errorKey(new TunnelAPIError(409), true)).toBe("tunnels.errors.revision"));
});
