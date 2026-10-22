// Run against the test-only Go connector fixture, never a real Cloudflare account.
// Start a disposable Podux with PODUX_CLOUDFLARED_BINARY pointing at the compiled
// qa/testdata/cloudflared fixture. See the WebUI validation document.
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
assert.equal(process.env.PODUX_QA_DISPOSABLE, "1");
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE);
const proof = process.env.PROOF_DIR || "development/screenshots";
const api = "http://127.0.0.1:8090/api";
const auth = await fetch(api + "/collections/fh_users/auth-with-password", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    identity: "ui-test@example.invalid",
    password: "temporary-ui-test-password",
  }),
}).then((r) => r.json());
assert(auth.token);
const call = async (path, method = "GET", body) => {
  const r = await fetch(api + "/cloudflared" + path, {
    method,
    headers: { Authorization: "Bearer " + auth.token, "Content-Type": "application/json" },
    ...(body ? { body: JSON.stringify(body) } : {}),
  });
  assert(r.ok, `${method} ${path}: ${r.status}`);
  return r.status === 204 ? null : r.json();
};
const cfg = {
  name: "Lifecycle A",
  description: "Test fixture, no Cloudflare connection",
  autoStart: false,
  autoRestart: true,
  maxRestartAttempts: 2,
  protocol: "quic",
  region: "",
  postQuantum: false,
  connectionRetries: 5,
  gracePeriodSeconds: 1,
  edgeIPVersion: "auto",
  edgeBindAddress: "",
  metricsPort: 0,
  extraArgs: [],
};
for (const t of (await call("/tunnels")).items)
  await call(`/tunnels/${t.id}?revision=${t.revision}`, "DELETE");
const syntheticToken = Buffer.from(
  JSON.stringify({ a: "test", t: "00000000-0000-0000-0000-000000000000", s: "synthetic!" })
).toString("base64");
const a = await call("/tunnels", "POST", { ...cfg, token: syntheticToken });
const b = await call("/tunnels", "POST", { ...cfg, name: "Lifecycle B", token: syntheticToken });
const browser = await chromium.launch({
  executablePath:
    process.env.CHROMIUM_PATH || "/opt/autopdd-browsers/chromium-1248/chrome-linux64/chrome",
  args: ["--no-sandbox"],
  headless: true,
});
const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
const checks = [];
try {
  await page.goto("http://127.0.0.1:5173/login");
  await page.locator('input[type="email"]').fill("ui-test@example.invalid");
  await page.locator('input[type="password"]').fill("temporary-ui-test-password");
  await page.locator('button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.endsWith("/login"));
  await page.goto("http://127.0.0.1:5173/tunnels");
  const card = (name) => page.getByRole("article", { name, exact: true });
  for (const name of ["Lifecycle A", "Lifecycle B"]) {
    await card(name).getByRole("button", { name: "Start", exact: true }).click();
    await card(name).getByText("Running", { exact: true }).waitFor();
  }
  checks.push("Mobile Start creates two real fixture child processes and observes readiness");
  const sa = await call(`/tunnels/${a.id}/status`),
    sb = await call(`/tunnels/${b.id}/status`);
  assert.notEqual(sa.metricsAddress, sb.metricsAddress);
  assert.equal(sa.connections, 2);
  checks.push("Independent loopback Metrics endpoints and collected connection gauges");
  await mkdir(proof, { recursive: true });
  await page.screenshot({
    path: proof + "/cloudflare-fixture-mobile-running.png",
    fullPage: true,
  });
  await card("Lifecycle A").getByRole("button", { name: "Stop", exact: true }).click();
  await card("Lifecycle A").getByText("Stopped", { exact: true }).waitFor();
  assert.equal((await call(`/tunnels/${b.id}/status`)).phase, "running");
  checks.push("Mobile Stop terminates A's real child while B keeps running");
  await card("Lifecycle A").getByRole("button", { name: "Restart", exact: true }).click();
  await card("Lifecycle A").getByText("Running", { exact: true }).waitFor();
  assert((await call(`/tunnels/${a.id}/status`)).generation > sa.generation);
  checks.push("Mobile Restart starts a replacement child with a new generation");
  await card("Lifecycle A").getByRole("button", { name: "Edit", exact: true }).click();
  await page.getByLabel("Description", { exact: true }).fill("Changed live fixture configuration");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page.getByRole("button", { name: "Restart later", exact: true }).click();
  await card("Lifecycle A")
    .getByText("Restart required to apply configuration", { exact: true })
    .waitFor();
  assert.equal((await call(`/tunnels/${a.id}/status`)).revision, 1);
  checks.push("Saving a live config keeps the old run until explicit restart");
  await card("Lifecycle A").getByRole("button", { name: "Edit", exact: true }).click();
  await page.getByLabel("Tunnel Token", { exact: true }).fill(syntheticToken);
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page.getByRole("button", { name: "Restart now", exact: true }).click();
  await page.getByRole("alertdialog").waitFor({ state: "detached" });
  await card("Lifecycle A").getByText("Running", { exact: true }).waitFor();
  assert.equal((await call(`/tunnels/${a.id}/status`)).revision, 3);
  checks.push("Explicit Token replacement and Restart now apply the saved revision");
  await card("Lifecycle A").getByRole("button", { name: "Edit", exact: true }).click();
  await page.getByLabel("Description", { exact: true }).fill("Unsaved browser draft");
  const latest = await call(`/tunnels/${a.id}`);
  const write = Object.fromEntries(Object.keys(cfg).map((k) => [k, latest[k]]));
  await call(`/tunnels/${a.id}`, "PUT", {
    ...write,
    revision: latest.revision,
    description: "Other editor",
  });
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page.getByText(/Your current input has been kept/).waitFor();
  assert.equal(
    await page.getByLabel("Description", { exact: true }).inputValue(),
    "Unsaved browser draft"
  );
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  checks.push("A real concurrent API edit produces 409 and preserves the browser draft");
  await card("Lifecycle A").getByRole("button", { name: "Details & logs", exact: true }).click();
  await page
    .getByRole("log")
    .getByText(/tunnel connection registered/)
    .first()
    .waitFor();
  await page.getByRole("button", { name: "Close", exact: true }).scrollIntoViewIfNeeded();
  await page.screenshot({
    path: proof + "/cloudflare-fixture-mobile-events.png",
    fullPage: true,
  });
  const log = await call(`/tunnels/${a.id}/logs?after=0`);
  assert(log.epoch);
  checks.push("Actual backend log epoch and safe child lifecycle events reach the viewer");
  await page.getByRole("button", { name: "Close", exact: true }).click();
  await page.evaluate(() => {
    localStorage.setItem("theme", "dark");
    localStorage.setItem("user-language", "zh");
  });
  await page.setViewportSize({ width: 1366, height: 950 });
  await page.reload();
  await card("Lifecycle B").getByText("运行中", { exact: true }).waitFor();
  await page.screenshot({
    path: proof + "/cloudflare-fixture-desktop-dark-zh.png",
    fullPage: true,
  });
  checks.push("Desktop dark Chinese rendering with fixture running states");
  await writeFile(
    proof + "/lifecycle-browser-checks.json",
    JSON.stringify(
      {
        checks,
        fixture: "Go test-only connector; no Cloudflare networking or real account",
        passed: checks.length,
      },
      null,
      2
    )
  );
  console.log(JSON.stringify({ passed: checks.length, checks }, null, 2));
} finally {
  for (const t of (await call("/tunnels")).items)
    await call(`/tunnels/${t.id}?revision=${t.revision}`, "DELETE");
  await browser.close();
}
