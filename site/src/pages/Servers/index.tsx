import { Flex, Tabs } from "@radix-ui/themes";
import { useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { PageHeader } from "../../components/PageHeader";
import { useServers } from "./useServers";
import { ServersView } from "./ServersPage.view";
import { TunnelsPage } from "../Tunnels";

function FRPCServers() {
  const state = useServers();
  return <ServersView {...state} embedded />;
}

export function ServersPage() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  const type = params.get("type") === "cloudflare" ? "cloudflare" : "frpc";
  return (
    <Flex direction="column" gap="5" className="flex flex-1 flex-col">
      <PageHeader title={t("server.title")} description={t("server.manageServers")} />
      <Tabs.Root
        value={type}
        onValueChange={(value) =>
          setParams((previous) => {
            const next = new URLSearchParams(previous);
            if (value === "cloudflare") next.set("type", value);
            else next.delete("type");
            return next;
          })
        }
      >
        <Tabs.List>
          <Tabs.Trigger value="frpc" aria-label="FRPC">
            FRPC
          </Tabs.Trigger>
          <Tabs.Trigger value="cloudflare" aria-label="Cloudflare Tunnel">
            Cloudflare Tunnel
          </Tabs.Trigger>
        </Tabs.List>
        <Tabs.Content value="frpc">
          <FRPCServers />
        </Tabs.Content>
        <Tabs.Content value="cloudflare">
          <TunnelsPage embedded />
        </Tabs.Content>
      </Tabs.Root>
    </Flex>
  );
}
export default ServersPage;
