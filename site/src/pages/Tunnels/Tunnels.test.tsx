import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { Theme } from "@radix-ui/themes";
import { TunnelsPage } from "./index";
import { TunnelForm } from "./TunnelForm";
import { TunnelLogViewer } from "./components/TunnelLogViewer";
import { TunnelStatus as StatusBadge } from "./components/TunnelStatus";
import {
  defaults,
  tunnelAPI,
  TunnelAPIError,
  type Tunnel,
  type TunnelStatus,
  type Capabilities,
} from "./api";
import { usePolling } from "./hooks";
import i18n from "../../i18n";
import en from "../../locales/en.json";
import zh from "../../locales/zh.json";
vi.mock("./api", async (original) => ({
  ...(await original<typeof import("./api")>()),
  tunnelAPI: {
    list: vi.fn(),
    capabilities: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
    control: vi.fn(),
    status: vi.fn(),
    logs: vi.fn(),
  },
}));
const item: Tunnel = {
  ...defaults,
  name: "Home",
  description: "Local connector",
  id: "aaaaaaaaaaaaaaa",
  revision: 1,
  hasToken: true,
};
const stopped: TunnelStatus = {
  phase: "stopped",
  ready: false,
  connections: 0,
  protocol: "unknown",
  generation: 0,
  revision: 0,
  restartCount: 0,
};
const caps: Capabilities = {
  available: true,
  version: "2025.4.0",
  build: "test",
  tokenFile: true,
  tlsLocalOverride: false,
  flags: ["post-quantum", "ha-connections", "loglevel"],
};
const renderUI = (node: React.ReactNode) => render(<Theme>{node}</Theme>);
beforeEach(() => {
  vi.mocked(tunnelAPI.list).mockResolvedValue({ items: [item] });
  vi.mocked(tunnelAPI.status).mockResolvedValue(stopped);
  vi.mocked(tunnelAPI.capabilities).mockResolvedValue(caps);
  vi.mocked(tunnelAPI.logs).mockResolvedValue({ items: [] });
  vi.mocked(tunnelAPI.create).mockResolvedValue(item);
  vi.mocked(tunnelAPI.update).mockResolvedValue({
    tunnel: { ...item, revision: 2 },
    restartRequired: false,
  });
  vi.mocked(tunnelAPI.control).mockResolvedValue({ ...stopped, phase: "starting" });
  vi.mocked(tunnelAPI.remove).mockResolvedValue();
});
const form = (tunnel?: Tunnel) => {
  const saved = vi.fn();
  renderUI(<TunnelForm tunnel={tunnel} caps={caps} onClose={vi.fn()} onSaved={saved} />);
  return saved;
};
const fillNew = () => {
  fireEvent.change(screen.getByLabelText("Name"), { target: { value: "New profile" } });
  fireEvent.change(screen.getByLabelText("Tunnel Token"), { target: { value: "synthetic-token" } });
};
async function select(label: string, value: string) {
  fireEvent.keyDown(screen.getByRole("combobox", { name: label }), { key: "Enter" });
  fireEvent.click(await screen.findByRole("option", { name: value }));
}
describe("profile forms", () => {
  it("creates a full config with a password Token input", async () => {
    const saved = form();
    fillNew();
    expect(screen.getByLabelText("Tunnel Token")).toHaveAttribute("type", "password");
    fireEvent.click(screen.getByText("Save", { selector: "button" }));
    await waitFor(() => expect(saved).toHaveBeenCalled());
    expect(tunnelAPI.create).toHaveBeenCalledWith(
      expect.objectContaining({ name: "New profile", metricsPort: 0, gracePeriodSeconds: 30 }),
      "synthetic-token"
    );
  });
  it("keeps edit Token empty and preserves it on save", async () => {
    form(item);
    expect(screen.getByLabelText("Tunnel Token")).toHaveValue("");
    fireEvent.click(screen.getByText("Save", { selector: "button" }));
    await waitFor(() =>
      expect(tunnelAPI.update).toHaveBeenCalledWith(
        expect.objectContaining({ revision: 1 }),
        expect.anything(),
        ""
      )
    );
  });
  it("passes an explicitly replaced Token", async () => {
    form(item);
    fireEvent.change(screen.getByLabelText("Tunnel Token"), {
      target: { value: "replacement-token" },
    });
    fireEvent.click(screen.getByText("Save", { selector: "button" }));
    await waitFor(() =>
      expect(tunnelAPI.update).toHaveBeenCalledWith(
        expect.anything(),
        expect.anything(),
        "replacement-token"
      )
    );
  });
  it("never submits a mask as a replacement", () => {
    form(item);
    fireEvent.change(screen.getByLabelText("Tunnel Token"), { target: { value: "********" } });
    fireEvent.click(screen.getByText("Save", { selector: "button" }));
    expect(tunnelAPI.update).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });
  it("requires a name and new Token", () => {
    form();
    fillNew();
    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "" } });
    fireEvent.click(screen.getByText("Save", { selector: "button" }));
    expect(tunnelAPI.create).not.toHaveBeenCalled();
  });
  it("preserves draft and Token on revision conflicts", async () => {
    vi.mocked(tunnelAPI.update).mockRejectedValue(new TunnelAPIError(409));
    form(item);
    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "Unsaved edit" } });
    fireEvent.change(screen.getByLabelText("Tunnel Token"), {
      target: { value: "replacement-token" },
    });
    fireEvent.click(screen.getByText("Save", { selector: "button" }));
    expect(await screen.findByText(/Your current input has been kept/)).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toHaveValue("Unsaved edit");
    expect(screen.getByLabelText("Tunnel Token")).toHaveValue("replacement-token");
    expect(tunnelAPI.get).not.toHaveBeenCalled();
  });
  it("gates PQ on explicit QUIC and clears it when switching to HTTP2", async () => {
    form(item);
    fireEvent.click(screen.getByText("Advanced network & runtime settings"));
    expect(screen.getByRole("switch", { name: "Require Post Quantum" })).toBeDisabled();
    await select("Configured Protocol", "quic");
    fireEvent.click(screen.getByRole("switch", { name: "Require Post Quantum" }));
    await select("Configured Protocol", "http2");
    expect(screen.getByRole("switch", { name: "Require Post Quantum" })).not.toBeChecked();
    expect(screen.getByRole("switch", { name: "Require Post Quantum" })).toBeDisabled();
  });
  it("rejects mismatched IP versions and invalid number ranges", async () => {
    form({ ...item, edgeIPVersion: "4", edgeBindAddress: "::1" });
    fireEvent.click(screen.getByText("Save", { selector: "button" }));
    expect(tunnelAPI.update).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });
  it("disables unsupported capabilities and offers no TLS or arbitrary CLI field", () => {
    renderUI(
      <TunnelForm tunnel={item} caps={{ ...caps, flags: [] }} onClose={vi.fn()} onSaved={vi.fn()} />
    );
    fireEvent.click(screen.getByText("Advanced network & runtime settings"));
    expect(screen.getByRole("combobox", { name: "HA Connections" })).toBeDisabled();
    expect(
      screen.queryByRole("textbox", { name: /command|extra args|TLS/i })
    ).not.toBeInTheDocument();
  });
});
describe("list and independent controls", () => {
  it("shows the empty state", async () => {
    vi.mocked(tunnelAPI.list).mockResolvedValue({ items: [] });
    renderUI(<TunnelsPage />);
    expect(await screen.findByText("No Tunnel profiles yet")).toBeInTheDocument();
  });
  it("renders multiple profiles with observed unknown protocol", async () => {
    vi.mocked(tunnelAPI.list).mockResolvedValue({
      items: [item, { ...item, id: "bbbbbbbbbbbbbbb", name: "Office" }],
    });
    renderUI(<TunnelsPage />);
    expect(await screen.findByRole("article", { name: "Office" })).toBeInTheDocument();
    expect(screen.getAllByText(/Connector not ready/)).toHaveLength(2);
  });
  it("keeps other profiles operable while one request is pending", async () => {
    vi.mocked(tunnelAPI.list).mockResolvedValue({
      items: [item, { ...item, id: "bbbbbbbbbbbbbbb", name: "Office" }],
    });
    let complete!: (s: TunnelStatus) => void;
    vi.mocked(tunnelAPI.control).mockReturnValueOnce(
      new Promise((resolve) => {
        complete = resolve;
      })
    );
    renderUI(<TunnelsPage />);
    const home = await screen.findByRole("article", { name: "Home" });
    await waitFor(() => expect(within(home).getByRole("button", { name: "Start" })).toBeEnabled());
    fireEvent.click(within(home).getByRole("button", { name: "Start" }));
    expect(within(home).getByRole("button", { name: "Start" })).toBeDisabled();
    expect(
      within(screen.getByRole("article", { name: "Office" })).getByRole("button", { name: "Start" })
    ).toBeEnabled();
    await act(async () => complete({ ...stopped, phase: "starting" }));
  });
  it.each(["Start", "Stop", "Restart"])("sends %s and refreshes observed state", async (label) => {
    vi.mocked(tunnelAPI.status).mockResolvedValue(
      label === "Start" ? stopped : { ...stopped, phase: "running", ready: true, protocol: "quic" }
    );
    renderUI(<TunnelsPage />);
    const card = await screen.findByRole("article", { name: "Home" });
    await waitFor(() => expect(within(card).getByRole("button", { name: label })).toBeEnabled());
    fireEvent.click(within(card).getByRole("button", { name: label }));
    await waitFor(() =>
      expect(tunnelAPI.control).toHaveBeenCalledWith(item.id, label.toLowerCase())
    );
    await waitFor(() => expect(vi.mocked(tunnelAPI.status).mock.calls.length).toBeGreaterThan(1));
  });
  it("requires deletion confirmation and retains record after a failed delete", async () => {
    vi.mocked(tunnelAPI.remove).mockRejectedValue(new TunnelAPIError(409));
    renderUI(<TunnelsPage />);
    const card = await screen.findByRole("article", { name: "Home" });
    fireEvent.click(within(card).getByRole("button", { name: "Delete" }));
    expect(tunnelAPI.remove).not.toHaveBeenCalled();
    const dialog = screen.getByRole("alertdialog");
    fireEvent.click(within(dialog).getByRole("button", { name: "Delete" }));
    expect(await screen.findByText(/Your current input has been kept/)).toBeInTheDocument();
    expect(screen.getByRole("article", { name: "Home", hidden: true })).toBeInTheDocument();
  });
  it("removes records only after a successful delete", async () => {
    renderUI(<TunnelsPage />);
    const card = await screen.findByRole("article", { name: "Home" });
    fireEvent.click(within(card).getByRole("button", { name: "Delete" }));
    vi.mocked(tunnelAPI.list).mockResolvedValue({ items: [] });
    fireEvent.click(
      within(screen.getByRole("alertdialog")).getByRole("button", { name: "Delete" })
    );
    expect(await screen.findByText("No Tunnel profiles yet")).toBeInTheDocument();
  });
  it("prompts for explicit restart after saving, without automatic control", async () => {
    vi.mocked(tunnelAPI.update).mockResolvedValue({
      tunnel: { ...item, revision: 2 },
      restartRequired: true,
    });
    renderUI(<TunnelsPage />);
    fireEvent.click(
      within(await screen.findByRole("article", { name: "Home" })).getByRole("button", {
        name: "Edit",
      })
    );
    fireEvent.click(screen.getByText("Save", { selector: "button" }));
    expect(
      await screen.findByText("Configuration saved. Restart the Tunnel to apply the changes.")
    ).toBeInTheDocument();
    expect(tunnelAPI.control).not.toHaveBeenCalled();
    fireEvent.click(screen.getByText("Restart later", { selector: "button" }));
    expect(tunnelAPI.control).not.toHaveBeenCalled();
  });
  it("degrades gracefully without cloudflared while allowing editing", async () => {
    vi.mocked(tunnelAPI.capabilities).mockResolvedValue({ ...caps, available: false, version: "" });
    renderUI(<TunnelsPage />);
    const card = await screen.findByRole("article", { name: "Home" });
    await screen.findByText(/Install cloudflared 2025/);
    expect(within(card).getByRole("button", { name: "Start" })).toBeDisabled();
    expect(within(card).getByRole("button", { name: "Edit" })).toBeEnabled();
  });
  it("shows safe 401 errors", async () => {
    vi.mocked(tunnelAPI.list).mockRejectedValue(new TunnelAPIError(401));
    renderUI(<TunnelsPage />);
    expect(
      await screen.findByText("Authentication expired. Please sign in again.")
    ).toBeInTheDocument();
  });
  it("displays backoff separately from running", () => {
    renderUI(<StatusBadge status={{ ...stopped, phase: "backoff" }} />);
    expect(screen.getByText("Backoff")).toBeInTheDocument();
    expect(screen.getByText(/Unknown/)).toBeInTheDocument();
  });
});
describe("logs and polling", () => {
  it("shows empty logs", async () => {
    renderUI(<TunnelLogViewer id={item.id} />);
    await waitFor(() =>
      expect(tunnelAPI.logs).toHaveBeenCalledWith(item.id, 0, expect.any(AbortSignal))
    );
    expect(screen.getByText("No runtime events yet")).toBeInTheDocument();
  });
  it("increments cursors, reports truncation and resets after restart", async () => {
    vi.mocked(tunnelAPI.logs)
      .mockResolvedValueOnce({ items: [{ cursor: 5, message: "safe event" }] })
      .mockResolvedValueOnce({ items: [] })
      .mockResolvedValueOnce({ items: [{ cursor: 1, message: "new process" }] });
    renderUI(<TunnelLogViewer id={item.id} />);
    expect(await screen.findByText("5 safe event")).toBeInTheDocument();
    expect(screen.getByText(/Earlier logs were truncated/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(await screen.findByText("1 new process")).toBeInTheDocument();
    expect(tunnelAPI.logs).toHaveBeenCalledWith(item.id, 5, undefined);
    expect(screen.queryByText("5 safe event")).not.toBeInTheDocument();
  });
  it("keeps existing logs on request failure", async () => {
    vi.mocked(tunnelAPI.logs)
      .mockResolvedValueOnce({ items: [{ cursor: 1, message: "kept" }] })
      .mockRejectedValueOnce(new TunnelAPIError(500));
    renderUI(<TunnelLogViewer id={item.id} />);
    await screen.findByText("1 kept");
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    await screen.findByRole("alert");
    expect(screen.getByText("1 kept")).toBeInTheDocument();
  });
  it("supports copying and pausing auto-scroll", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    vi.mocked(tunnelAPI.logs).mockResolvedValue({ items: [{ cursor: 1, message: "safe" }] });
    renderUI(<TunnelLogViewer id={item.id} />);
    await screen.findByText("1 safe");
    fireEvent.click(screen.getByRole("switch", { name: "Auto-scroll" }));
    expect(screen.getByRole("switch", { name: "Auto-scroll" })).not.toBeChecked();
    fireEvent.click(screen.getByRole("button", { name: "Copy safe logs" }));
    expect(writeText).toHaveBeenCalledWith("1 safe");
  });
  it("cleans polling timers and aborts requests on unmount", async () => {
    vi.useFakeTimers();
    let signal: AbortSignal | undefined;
    const task = vi.fn(async (s: AbortSignal) => {
      signal = s;
    });
    function Poll() {
      usePolling(task, 3000);
      return null;
    }
    const { unmount } = render(<Poll />);
    await act(async () => {});
    expect(task).toHaveBeenCalledTimes(1);
    unmount();
    await act(async () => vi.advanceTimersByTime(10000));
    expect(task).toHaveBeenCalledTimes(1);
    expect(signal?.aborted).toBe(true);
  });
  it("does not overlap slow requests", async () => {
    vi.useFakeTimers();
    let resolve!: () => void;
    const task = vi.fn(
      () =>
        new Promise<void>((r) => {
          resolve = r;
        })
    );
    function Poll() {
      usePolling(task, 3000);
      return null;
    }
    render(<Poll />);
    await act(async () => vi.advanceTimersByTime(12000));
    expect(task).toHaveBeenCalledTimes(1);
    await act(async () => resolve());
  });
  it("has matching bilingual keys and Chinese status labels", async () => {
    const keys = (o: object): string[] =>
      Object.entries(o).flatMap(([k, v]) =>
        typeof v === "object" ? keys(v).map((x) => `${k}.${x}`) : [k]
      );
    expect(keys(en.tunnels).sort()).toEqual(keys(zh.tunnels).sort());
    await i18n.changeLanguage("zh");
    renderUI(<StatusBadge status={{ ...stopped, phase: "running", ready: true }} />);
    expect(screen.getByText("运行中")).toBeInTheDocument();
  });
});

describe("polling and restart regressions", () => {
  it("retains the current list after a temporary request failure", async () => {
    renderUI(<TunnelsPage />);
    await screen.findByText(/cloudflared available/);
    vi.mocked(tunnelAPI.list).mockRejectedValueOnce(new TunnelAPIError(500));
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    await screen.findByRole("alert");
    expect(screen.getByRole("article", { name: "Home" })).toBeInTheDocument();
  });
  it("only restarts a saved configuration after explicit Restart now", async () => {
    vi.mocked(tunnelAPI.update).mockResolvedValue({
      tunnel: { ...item, revision: 2 },
      restartRequired: true,
    });
    renderUI(<TunnelsPage />);
    await screen.findByText(/cloudflared available/);
    fireEvent.click(
      within(screen.getByRole("article", { name: "Home" })).getByRole("button", { name: "Edit" })
    );
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await screen.findByText("Restart now");
    expect(tunnelAPI.control).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Restart now" }));
    await waitFor(() => expect(tunnelAPI.control).toHaveBeenCalledWith(item.id, "restart"));
  });
  it("requires the Token for creation", () => {
    form();
    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "New" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(tunnelAPI.create).not.toHaveBeenCalled();
    expect(screen.getByLabelText("Tunnel Token")).toBeRequired();
  });
  it("pauses polling when the document is hidden and resumes visibly", async () => {
    vi.useFakeTimers();
    let visibility = "visible";
    const property = vi
      .spyOn(document, "visibilityState", "get")
      .mockImplementation(() => visibility as DocumentVisibilityState);
    const task = vi.fn(async () => {});
    function Poll() {
      usePolling(task, 3000);
      return null;
    }
    render(<Poll />);
    await act(async () => {});
    visibility = "hidden";
    await act(async () => document.dispatchEvent(new Event("visibilitychange")));
    await act(async () => vi.advanceTimersByTime(10000));
    expect(task).toHaveBeenCalledTimes(1);
    visibility = "visible";
    await act(async () => document.dispatchEvent(new Event("visibilitychange")));
    expect(task).toHaveBeenCalledTimes(2);
    property.mockRestore();
  });
  it.each(["stopped", "starting", "running", "backoff", "stopping", "failed"] as const)(
    "shows %s as a distinct runtime state",
    (phase) => {
      renderUI(<StatusBadge status={{ ...stopped, phase }} />);
      expect(screen.getByText(en.tunnels.states[phase])).toBeInTheDocument();
    }
  );
  it("does not display unexpected secret fields from a normal profile response", async () => {
    vi.mocked(tunnelAPI.list).mockResolvedValue({
      items: [{ ...item, token: "SYNTHETIC-SECRET", ciphertext: "CIPHER-VALUE" } as Tunnel],
    });
    renderUI(<TunnelsPage />);
    await screen.findByRole("article", { name: "Home" });
    expect(document.body.textContent).not.toContain("SYNTHETIC-SECRET");
    expect(document.body.textContent).not.toContain("CIPHER-VALUE");
  });
});
it("resets logs across server epochs even when a restarted cursor exceeds the old one", async () => {
  vi.mocked(tunnelAPI.logs)
    .mockResolvedValueOnce({ epoch: "old", items: [{ cursor: 5, message: "old event" }] })
    .mockResolvedValueOnce({ epoch: "new", items: [{ cursor: 6, message: "new tail" }] })
    .mockResolvedValueOnce({
      epoch: "new",
      items: [
        { cursor: 1, message: "new beginning" },
        { cursor: 6, message: "new tail" },
      ],
    });
  renderUI(<TunnelLogViewer id={item.id} />);
  await screen.findByText("5 old event");
  fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
  expect(await screen.findByText("1 new beginning")).toBeInTheDocument();
  expect(screen.queryByText("5 old event")).not.toBeInTheDocument();
});
