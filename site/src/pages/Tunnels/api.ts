import { apiFetch } from "../../lib/api";

export interface ExtraArg {
  name: "loglevel" | "ha-connections";
  value: string;
}
export interface TunnelConfig {
  name: string;
  description: string;
  autoStart: boolean;
  autoRestart: boolean;
  maxRestartAttempts: number;
  protocol: "auto" | "quic" | "http2";
  region: "" | "us";
  postQuantum: boolean;
  connectionRetries: number;
  gracePeriodSeconds: number;
  edgeIPVersion: "auto" | "4" | "6";
  edgeBindAddress: string;
  metricsPort: number;
  extraArgs: ExtraArg[];
}
export interface Tunnel extends TunnelConfig {
  id: string;
  revision: number;
  hasToken: boolean;
}
export type Phase = "stopped" | "starting" | "running" | "backoff" | "stopping" | "failed";
export interface TunnelStatus {
  phase: Phase;
  ready: boolean;
  connections: number;
  protocol: string;
  generation: number;
  revision: number;
  restartCount: number;
  nextRetry?: string;
  lastError?: string;
  metricsAddress?: string;
}
export interface Capabilities {
  available: boolean;
  version: string;
  build: string;
  tokenFile: boolean;
  tlsLocalOverride: boolean;
  flags: string[];
}
export interface TunnelLog {
  cursor: number;
  message: string;
}
export type Action = "start" | "stop" | "restart";
export const defaults: TunnelConfig = {
  name: "",
  description: "",
  autoStart: false,
  autoRestart: false,
  maxRestartAttempts: 5,
  protocol: "auto",
  region: "",
  postQuantum: false,
  connectionRetries: 5,
  gracePeriodSeconds: 30,
  edgeIPVersion: "auto",
  edgeBindAddress: "",
  metricsPort: 0,
  extraArgs: [],
};
export class TunnelAPIError extends Error {
  readonly status: number;
  constructor(status: number) {
    super("Tunnel request failed");
    this.status = status;
  }
}
// Do not retain response/request objects or echo arbitrary server error text.
export function errorKey(error: unknown, conflict = false): string {
  const status = error instanceof TunnelAPIError ? error.status : 0;
  if (status === 409 && conflict) return "tunnels.errors.revision";
  return `tunnels.errors.${[400, 401, 404, 409, 503].includes(status) ? status : "generic"}`;
}
const root = "/api/cloudflared";
async function request<T>(
  path: string,
  method = "GET",
  body?: unknown,
  signal?: AbortSignal
): Promise<T> {
  let res: Response;
  try {
    res = await apiFetch(root + path, {
      method,
      signal,
      ...(body === undefined
        ? {}
        : { headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }),
    });
  } catch (error) {
    if (signal?.aborted) throw error;
    throw new TunnelAPIError(0);
  }
  if (!res.ok) throw new TunnelAPIError(res.status);
  if (res.status === 204) return undefined as T;
  try {
    return (await res.json()) as T;
  } catch {
    throw new TunnelAPIError(0);
  }
}
export const tunnelAPI = {
  capabilities: (signal?: AbortSignal) =>
    request<Capabilities>("/capabilities", "GET", undefined, signal),
  list: (signal?: AbortSignal) =>
    request<{ items: Tunnel[] }>("/tunnels", "GET", undefined, signal),
  get: (id: string, signal?: AbortSignal) =>
    request<Tunnel>(`/tunnels/${encodeURIComponent(id)}`, "GET", undefined, signal),
  create: (config: TunnelConfig, token: string) =>
    request<Tunnel>("/tunnels", "POST", { ...config, token }),
  update: (tunnel: Tunnel, config: TunnelConfig, token: string) =>
    request<{ tunnel: Tunnel; restartRequired: boolean }>(`/tunnels/${tunnel.id}`, "PUT", {
      ...config,
      revision: tunnel.revision,
      ...(token.trim() ? { token: token.trim() } : {}),
    }),
  remove: (tunnel: Tunnel) =>
    request<void>(`/tunnels/${tunnel.id}?revision=${tunnel.revision}`, "DELETE"),
  control: (id: string, action: Action) =>
    request<TunnelStatus>(`/tunnels/${id}/${action}`, "POST"),
  status: (id: string, signal?: AbortSignal) =>
    request<TunnelStatus>(`/tunnels/${id}/status`, "GET", undefined, signal),
  logs: (id: string, cursor: number, signal?: AbortSignal) =>
    request<{ items: TunnelLog[]; epoch?: string }>(
      `/tunnels/${id}/logs?after=${cursor}`,
      "GET",
      undefined,
      signal
    ),
};
// Runtime/secret response fields are never spread into a write payload.
export function configuration(tunnel: Tunnel): TunnelConfig {
  return Object.fromEntries(
    Object.keys(defaults).map((key) => [
      key,
      tunnel[key as keyof TunnelConfig] ?? defaults[key as keyof TunnelConfig],
    ])
  ) as unknown as TunnelConfig;
}
function ipVersion(value: string): "4" | "6" | undefined {
  if (
    /^(\d{1,3}\.){3}\d{1,3}$/.test(value) &&
    value.split(".").every((v) => Number(v) <= 255 && String(Number(v)) === v)
  )
    return "4";
  if (value.includes(":") && !/[[\]%/\s]/.test(value)) {
    try {
      if (new URL(`http://[${value}]/`).hostname) return "6";
    } catch {
      /* invalid IP */
    }
  }
}
export function validConfig(c: TunnelConfig): boolean {
  const bytes = (s: string) => new TextEncoder().encode(s).length;
  const range = (v: number, min: number, max: number) =>
    Number.isInteger(v) && v >= min && v <= max;
  const ip = c.edgeBindAddress ? ipVersion(c.edgeBindAddress) : undefined;
  return (
    !!c.name.trim() &&
    bytes(c.name) <= 120 &&
    bytes(c.description) <= 2000 &&
    ["auto", "quic", "http2"].includes(c.protocol) &&
    ["", "us"].includes(c.region) &&
    ["auto", "4", "6"].includes(c.edgeIPVersion) &&
    (!c.postQuantum || c.protocol === "quic") &&
    (!c.edgeBindAddress || (!!ip && (c.edgeIPVersion === "auto" || c.edgeIPVersion === ip))) &&
    range(c.maxRestartAttempts, 0, 100) &&
    range(c.connectionRetries, 0, 20) &&
    range(c.gracePeriodSeconds, 1, 120) &&
    range(c.metricsPort, 0, 65535) &&
    c.extraArgs.length <= 2 &&
    new Set(c.extraArgs.map((a) => a.name)).size === c.extraArgs.length &&
    c.extraArgs.every(
      (a) =>
        (a.name === "loglevel" && a.value === "info") ||
        (a.name === "ha-connections" && ["1", "2", "3", "4"].includes(a.value))
    )
  );
}

export function runtimeErrorKey(message: string): string {
  const known: Record<string, string> = {
    "cloudflared exited or could not start": "tunnels.errors.process",
    "cloudflared reported a connection error": "tunnels.errors.connection",
    "tunnel credentials unavailable": "tunnels.errors.503",
    "cloudflared unavailable or unsupported": "tunnels.errors.503",
  };
  return known[message] ?? "tunnels.errors.generic";
}
