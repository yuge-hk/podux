import { Button, Flex } from "@radix-ui/themes";
import { useTranslation } from "react-i18next";
import type { Action, TunnelStatus } from "../api";
export function TunnelActions({
  status,
  busy,
  available,
  hasToken,
  onAction,
}: {
  status?: TunnelStatus;
  busy: boolean;
  available: boolean;
  hasToken: boolean;
  onAction: (action: Action) => void;
}) {
  const { t } = useTranslation();
  const stopped = status?.phase === "stopped" || status?.phase === "failed";
  return (
    <Flex gap="2" wrap="wrap">
      <Button
        size="1"
        variant="soft"
        disabled={busy || !available || !hasToken || !stopped}
        onClick={() => onAction("start")}
      >
        {t("tunnels.start")}
      </Button>
      <Button
        size="1"
        variant="soft"
        disabled={busy || !status || stopped || status.phase === "stopping"}
        onClick={() => onAction("stop")}
      >
        {t("tunnels.stop")}
      </Button>
      <Button
        size="1"
        variant="soft"
        disabled={busy || !available || !hasToken || !status || status.phase === "stopping"}
        onClick={() => onAction("restart")}
      >
        {t("tunnels.restart")}
      </Button>
    </Flex>
  );
}
