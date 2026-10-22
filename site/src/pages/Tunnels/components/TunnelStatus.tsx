import { Badge, Flex, Text } from "@radix-ui/themes";
import { useTranslation } from "react-i18next";
import type { Phase, TunnelStatus as Status } from "../api";
const colors = {
  stopped: "gray",
  starting: "blue",
  running: "green",
  backoff: "amber",
  stopping: "orange",
  failed: "red",
} as const;
export function TunnelStatus({ status }: { status?: Status }) {
  const { t } = useTranslation();
  const phase: Phase | undefined =
    status && Object.hasOwn(colors, status.phase) ? status.phase : undefined;
  return (
    <Flex gap="2" align="center" wrap="wrap">
      <Badge color={phase ? colors[phase] : "gray"}>
        {t(phase ? `tunnels.states.${phase}` : "tunnels.unknown")}
      </Badge>
      <Text size="1" color="gray">
        {t(status?.ready ? "tunnels.ready" : "tunnels.notReady")} ·{" "}
        {status?.protocol === "quic" || status?.protocol === "http2"
          ? status.protocol
          : t("tunnels.unknown")}
      </Text>
    </Flex>
  );
}
