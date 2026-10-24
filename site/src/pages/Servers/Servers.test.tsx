import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { Theme } from "@radix-ui/themes";
import { expect, it, vi } from "vitest";
import { ServersPage } from "./index";
import { useServers } from "./useServers";
vi.mock("./useServers", () => ({ useServers: vi.fn(() => ({})) }));
vi.mock("./ServersPage.view", () => ({ ServersView: () => <div>FRPC content</div> }));
vi.mock("../Tunnels", () => ({ TunnelsPage: () => <div>Cloudflare content</div> }));
it("opens the Cloudflare server tab from its URL and mounts FRPC only on selection", () => {
  render(
    <Theme>
      <MemoryRouter initialEntries={["/servers?type=cloudflare"]}>
        <ServersPage />
      </MemoryRouter>
    </Theme>
  );
  expect(screen.getByText("Cloudflare content")).toBeVisible();
  expect(useServers).not.toHaveBeenCalled();
  fireEvent.mouseDown(screen.getByRole("tab", { name: "FRPC" }), { button: 0, ctrlKey: false });
  expect(screen.getByText("FRPC content")).toBeVisible();
  expect(screen.queryByText("Cloudflare content")).not.toBeInTheDocument();
  expect(useServers).toHaveBeenCalled();
});
