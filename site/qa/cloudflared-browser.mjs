// Native browser QA without adding an E2E framework to Podux dependencies.
// Requires a disposable backend on :8090 and Vite on :5173. Never use production.
// PLAYWRIGHT_MODULE=/absolute/path/playwright-core/index.mjs node site/qa/cloudflared-browser.mjs
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE);
assert.equal(
  process.env.PODUX_QA_DISPOSABLE,
  "1",
  "Use only a disposable backend and set PODUX_QA_DISPOSABLE=1"
);
const backend = "http://127.0.0.1:8090";
const email = "ui-test@example.invalid",
  password = "temporary-ui-test-password";
const init = await fetch(backend + "/api/system/initialized").then((r) => r.json());
if (!init.initialized) {
  const r = await fetch(backend + "/api/system/initialize", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password, language: "en" }),
  });
  assert.equal(r.status, 200);
}
// Reset only profiles in the explicitly disposable test instance.
const session = await fetch(backend + "/api/collections/fh_users/auth-with-password", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ identity: email, password }),
}).then((r) => r.json());
assert(session.token);
const old = await fetch(backend + "/api/cloudflared/tunnels", {
  headers: { Authorization: "Bearer " + session.token },
}).then((r) => r.json());
for (const profile of old.items) {
  const r = await fetch(
    backend + `/api/cloudflared/tunnels/${profile.id}?revision=${profile.revision}`,
    { method: "DELETE", headers: { Authorization: "Bearer " + session.token } }
  );
  assert.equal(r.status, 204);
}
const browser = await chromium.launch({
  executablePath:
    process.env.CHROMIUM_PATH || "/opt/autopdd-browsers/chromium-1248/chrome-linux64/chrome",
  headless: true,
  args: ["--no-sandbox"],
});
const proof = process.env.PROOF_DIR || "development/screenshots";
await mkdir(proof, { recursive: true });
const checks = [];
const page = await browser.newPage({ viewport: { width: 1366, height: 950 } });
const errors = [];
page.on("pageerror", (e) => errors.push(e.message));
const syntheticToken = Buffer.from(
  JSON.stringify({ a: "test", t: "00000000-0000-0000-0000-000000000000", s: "invalid!" })
).toString("base64");
const consoleText = [];
page.on("console", (message) => consoleText.push(message.text()));
const requests = [];
page.on("request", (request) => {
  if (request.url().includes("/api/cloudflared/")) {
    assert(!request.url().includes("token="));
    assert(request.headers().authorization);
    requests.push({ method: request.method(), path: new URL(request.url()).pathname });
  }
});
try {
  await page.goto("http://127.0.0.1:5173/tunnels");
  await page.waitForURL("**/login");
  checks.push("Unauthenticated /tunnels redirects to existing login");
  await page.locator('input[type="email"]').fill(email);
  await page.locator('input[type="password"]').fill(password);
  await page.locator('button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.endsWith("/login"));
  await page.goto("http://127.0.0.1:5173/tunnels");
  await page.getByText("No Tunnel profiles yet", { exact: true }).waitFor();
  checks.push("Existing login and empty Tunnel page");
  const create = async (name) => {
    await page.getByRole("button", { name: "Add Tunnel", exact: true }).click();
    await page.getByLabel("Name", { exact: true }).fill(name);
    await page.getByLabel("Tunnel Token", { exact: true }).fill(syntheticToken);
    await page.getByRole("button", { name: "Save", exact: true }).click();
    await page.getByRole("article", { name, exact: true }).waitFor();
  };
  await create("Home connector");
  await create("Office connector");
  checks.push("Create two real persisted profiles through the form");
  const home = page.getByRole("article", { name: "Home connector", exact: true });
  await home.getByRole("button", { name: "Edit", exact: true }).click();
  assert.equal(await page.getByLabel("Tunnel Token", { exact: true }).inputValue(), "");
  await page
    .getByLabel("Description", { exact: true })
    .fill("Token retained; local process management");
  await page.getByText("Advanced network & runtime settings", { exact: true }).click();
  await page.getByLabel("Metrics Port", { exact: true }).fill("0");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await home.getByText("Token retained; local process management").waitFor();
  checks.push("Edit retains empty Token and saves full advanced config");
  await page.reload();
  await page.getByRole("article", { name: "Home connector", exact: true }).waitFor();
  checks.push("Profiles persist after page reload");
  await page.screenshot({ path: proof + "/cloudflare-desktop-light.png", fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.evaluate(() => (document.documentElement.style.scrollBehavior = "auto"));
  await home.getByRole("button", { name: "Start", exact: true }).click();
  await home.getByText("Failed", { exact: true }).waitFor({ timeout: 15000 });
  await home.getByRole("button", { name: "Stop", exact: true }).isDisabled();
  checks.push("Mobile Start 202 is observed as local invalid-token failure, not connected");
  // Actual Stop/Restart API are tested through an injected still-running UI state;
  // the real runner has already exited on invalid credentials, so Stop is disabled.
  await home.getByRole("button", { name: "Restart", exact: true }).click();
  await home.getByText("Failed", { exact: true }).waitFor();
  checks.push("Mobile Restart calls actual backend");
  await home.getByRole("button", { name: "Edit", exact: true }).click();
  await page.getByLabel("Description", { exact: true }).fill("Edited on mobile");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await home.getByText("Edited on mobile").waitFor();
  checks.push("Mobile edit/save");
  await page.getByRole("button", { name: "Add Tunnel", exact: true }).click();
  await page.getByLabel("Name", { exact: true }).fill("Mobile connector");
  await page.getByLabel("Tunnel Token", { exact: true }).fill(syntheticToken);
  await page.getByText("Advanced network & runtime settings", { exact: true }).click();
  await page.screenshot({ path: proof + "/cloudflare-mobile-form.png", fullPage: true });
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page.getByRole("article", { name: "Mobile connector", exact: true }).waitFor();
  checks.push("Mobile create, scrollable form and advanced settings");
  await home.getByRole("button", { name: "Details & logs", exact: true }).click();
  await page.getByRole("log").waitFor();
  await page
    .getByRole("log")
    .getByText(/cloudflared process/)
    .first()
    .waitFor();
  await page.getByRole("button", { name: "Close", exact: true }).scrollIntoViewIfNeeded();
  await page.screenshot({ path: proof + "/cloudflare-mobile-logs.png", fullPage: true });
  await page.keyboard.press("Escape");
  checks.push("Mobile details and safe logs");
  assert(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth));
  checks.push("390px no global horizontal overflow");
  // Exercise a real Stop from a controlled UI status fixture, without real cloud credentials.
  await page.route("**/api/cloudflared/tunnels/*/status", async (route) => {
    const res = await route.fetch();
    const data = await res.json();
    await route.fulfill({ json: { ...data, phase: "running", ready: true, protocol: "quic" } });
  });
  await page.getByRole("button", { name: "Refresh", exact: true }).click();
  await home.getByRole("button", { name: "Stop", exact: true }).waitFor();
  await home.getByRole("button", { name: "Stop", exact: true }).click();
  checks.push("Mobile Stop request (UI running state fixture; real backend endpoint)");
  await page.unroute("**/api/cloudflared/tunnels/*/status");
  await page.evaluate(() => {
    localStorage.setItem("theme", "dark");
    localStorage.setItem("user-language", "zh");
  });
  await page.reload();
  await page.getByRole("button", { name: "新建 Tunnel", exact: true }).waitFor();
  await page.getByRole("article", { name: "Home connector", exact: true }).waitFor();
  await page.screenshot({ path: proof + "/cloudflare-mobile-dark-zh.png", fullPage: true });
  checks.push("Chinese dark theme");
  await page.setViewportSize({ width: 320, height: 740 });
  await page.screenshot({ path: proof + "/cloudflare-mobile-320.png", fullPage: true });
  assert(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth));
  checks.push("320px no horizontal overflow");
  // Return to English/light and check original navigation/screens with real APIs.
  await page.evaluate(() => {
    localStorage.setItem("theme", "light");
    localStorage.setItem("user-language", "en");
  });
  await page.setViewportSize({ width: 1366, height: 950 });
  await page.reload();
  for (const path of ["/servers", "/proxies", "/dashboard"]) {
    await page.goto("http://127.0.0.1:5173" + path);
    await page.waitForTimeout(1000);
    assert(!page.url().includes("/login"));
    assert((await page.locator("main").count()) === 0 || (await page.locator("body").innerText()));
  }
  await page.screenshot({ path: proof + "/frpc-dashboard-regression.png", fullPage: true });
  checks.push("Existing server/proxy/dashboard pages render with authenticated real APIs");
  await page.goto("http://127.0.0.1:5173/tunnels");
  await page.getByRole("article", { name: "Mobile connector", exact: true }).waitFor();
  await page
    .getByRole("article", { name: "Mobile connector", exact: true })
    .getByRole("button", { name: "Delete", exact: true })
    .click();
  await page.getByRole("alertdialog").getByRole("button", { name: "Delete", exact: true }).click();
  await page
    .getByRole("article", { name: "Mobile connector", exact: true })
    .waitFor({ state: "detached" });
  checks.push("Confirmed deletion updates real persisted list");
  await page.route("**/api/cloudflared/capabilities", (route) =>
    route.fulfill({
      json: {
        available: false,
        version: "",
        build: "",
        tokenFile: false,
        tlsLocalOverride: false,
        flags: [],
      },
    })
  );
  await page.reload();
  await page.getByText(/Install cloudflared 2025/).waitFor();
  assert(
    await page
      .getByRole("article", { name: "Home connector", exact: true })
      .getByRole("button", { name: "Start", exact: true })
      .isDisabled()
  );
  checks.push("Unavailable binary UI degrades with editing retained (capability fixture)");
  const storage = await page.evaluate(() =>
    JSON.stringify({ local: { ...localStorage }, session: { ...sessionStorage } })
  );
  assert(!storage.includes(syntheticToken));
  assert(!(await page.locator("body").innerText()).includes(syntheticToken));
  assert(!consoleText.join("\n").includes(syntheticToken));
  assert.deepEqual(errors, []);
  checks.push("No Tunnel Token in browser storage, displayed text, URL or console; no page errors");
  await writeFile(
    proof + "/browser-checks.json",
    JSON.stringify(
      {
        checks,
        requests,
        limits: [
          "Synthetic invalid Token only; no successful Cloudflare connection",
          "Stop UI and missing binary scenarios use explicit fixtures",
          "FRPC regression uses empty disposable database; no live FRP server",
        ],
      },
      null,
      2
    )
  );
  console.log(JSON.stringify({ passed: checks.length, checks }, null, 2));
} catch (error) {
  console.log("QA failure URL", page.url());
  console.log((await page.locator("body").innerText()).slice(0, 3000));
  await page.screenshot({ path: "/tmp/podux-ui-qa/failure.png", fullPage: true });
  throw error;
} finally {
  await browser.close();
}
