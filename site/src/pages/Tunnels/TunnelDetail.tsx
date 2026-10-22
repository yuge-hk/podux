import { Button, Dialog, Flex, Heading, Text } from "@radix-ui/themes";
import { useTranslation } from "react-i18next";
import { runtimeErrorKey, type Tunnel, type TunnelStatus as Status } from "./api";
import { TunnelStatus } from "./components/TunnelStatus";
import { TunnelLogViewer } from "./components/TunnelLogViewer";
export function TunnelDetail({
  tunnel,
  status,
  onClose,
}: {
  tunnel: Tunnel;
  status?: Status;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const value = (v: unknown) =>
    typeof v === "boolean"
      ? t(v ? "tunnels.yes" : "tunnels.no")
      : v === undefined || v === ""
        ? t("tunnels.unknown")
        : String(v);
  const rows: [string, unknown][] = [
    ["ready", status?.ready],
    ["connections", status?.connections],
    ["actualProtocol", status?.protocol],
    ["generation", status?.generation],
    ["runningRevision", status?.revision],
    ["revision", tunnel.revision],
    ["restartCount", status?.restartCount],
    ["nextRetry", status?.nextRetry],
    ["lastError", status?.lastError],
    ["protocol", tunnel.protocol],
    ["region", tunnel.region || t("tunnels.global")],
    ["edgeIPVersion", tunnel.edgeIPVersion],
    ["edgeBindAddress", tunnel.edgeBindAddress],
    ["postQuantum", tunnel.postQuantum],
    ["connectionRetries", tunnel.connectionRetries],
    ["gracePeriodSeconds", tunnel.gracePeriodSeconds],
    ["metricsPort", tunnel.metricsPort],
    ["maxRestartAttempts", tunnel.maxRestartAttempts],
    ["autoStart", tunnel.autoStart],
    ["autoRestart", tunnel.autoRestart],
  ];
  return (
    <Dialog.Root
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <Dialog.Content className="tunnel-dialog" maxWidth="760px">
        <Dialog.Title>{tunnel.name}</Dialog.Title>
        <Dialog.Description size="2" mb="3">
          {tunnel.description || t("tunnels.details")}
        </Dialog.Description>
        <Flex direction="column" gap="4">
          <TunnelStatus status={status} />
          <Text color="gray" size="1">
            {t("tunnels.readyHelp")}
          </Text>
          <dl className="tunnel-info">
            {rows.map(([key, v]) => (
              <div key={key} style={{ display: "contents" }}>
                <dt>{t(`tunnels.fields.${key}`)}</dt>
                <dd>
                  {key === "actualProtocol" && v !== "quic" && v !== "http2"
                    ? t("tunnels.unknown")
                    : key === "lastError" && typeof v === "string" && v
                      ? t(runtimeErrorKey(v))
                      : value(v)}
                </dd>
              </div>
            ))}
          </dl>
          <Heading size="3">{t("tunnels.logs.title")}</Heading>
          <TunnelLogViewer key={tunnel.id} id={tunnel.id} />
          <Flex justify="end">
            <Dialog.Close>
              <Button variant="soft">{t("tunnels.close")}</Button>
            </Dialog.Close>
          </Flex>
        </Flex>
      </Dialog.Content>
    </Dialog.Root>
  );
}
