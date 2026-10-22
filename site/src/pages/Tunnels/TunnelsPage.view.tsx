import { Badge, Button, Callout, Card, Flex, Heading, Text } from "@radix-ui/themes";
import { useTranslation } from "react-i18next";
import { PageHeader } from "../../components/PageHeader";
import { EmptyState } from "../../components/EmptyState";
import { Loading } from "../../components/Loading";
import {
  errorKey,
  runtimeErrorKey,
  type Action,
  type Capabilities,
  type Tunnel,
  type TunnelStatus as Status,
} from "./api";
import { TunnelStatus } from "./components/TunnelStatus";
import { TunnelActions } from "./components/TunnelActions";
interface Props {
  items: Tunnel[];
  statuses: Record<string, Status>;
  caps?: Capabilities;
  error: unknown;
  loading: boolean;
  pending: Record<string, boolean>;
  onAdd: () => void;
  onEdit: (t: Tunnel) => void;
  onDetail: (t: Tunnel) => void;
  onDelete: (t: Tunnel) => void;
  onAction: (id: string, action: Action) => void;
  onRefresh: () => void;
}
export function TunnelsPageView(p: Props) {
  const { t } = useTranslation();
  const unknown = t("tunnels.unknown");
  return (
    <Flex direction="column" gap="5" className="tunnels-page">
      <PageHeader title={t("tunnels.title")} description={t("tunnels.subtitle")} />
      <Flex gap="3" justify="between" wrap="wrap">
        <Badge color={p.caps?.available ? "green" : "gray"}>
          {t(p.caps?.available ? "tunnels.available" : "tunnels.unavailable")} ·{" "}
          {p.caps?.version || unknown}
        </Badge>
        <Flex gap="2">
          <Button variant="soft" onClick={p.onRefresh}>
            {t("tunnels.refresh")}
          </Button>
          <Button onClick={p.onAdd}>{t("tunnels.add")}</Button>
        </Flex>
      </Flex>
      {p.error !== undefined && (
        <Callout.Root color="red" role="alert">
          <Callout.Text>{t(errorKey(p.error))}</Callout.Text>
        </Callout.Root>
      )}
      {p.caps && !p.caps.available && (
        <Callout.Root color="amber">
          <Callout.Text>{t("tunnels.installHelp")}</Callout.Text>
        </Callout.Root>
      )}
      <details>
        <summary>{t("tunnels.capabilities")}</summary>
        <dl className="tunnel-info">
          <dt>{t("tunnels.version")}</dt>
          <dd>{p.caps?.version || unknown}</dd>
          <dt>{t("tunnels.build")}</dt>
          <dd>{p.caps?.build || unknown}</dd>
          <dt>{t("tunnels.tokenFile")}</dt>
          <dd>{p.caps ? t(p.caps.tokenFile ? "tunnels.yes" : "tunnels.no") : unknown}</dd>
          <dt>{t("tunnels.tlsOverride")}</dt>
          <dd>{p.caps ? t(p.caps.tlsLocalOverride ? "tunnels.yes" : "tunnels.no") : unknown}</dd>
          <dt>{t("tunnels.flags")}</dt>
          <dd style={{ overflowWrap: "anywhere" }}>{p.caps?.flags?.join(", ") || unknown}</dd>
        </dl>
      </details>
      {p.loading && !p.items.length ? (
        <Loading text={t("tunnels.loading")} />
      ) : !p.items.length && p.error === undefined ? (
        <EmptyState title={t("tunnels.empty")} description={t("tunnels.emptyHelp")} />
      ) : (
        <div className="tunnel-grid">
          {p.items.map((item) => {
            const status = p.statuses[item.id];
            const restart =
              !!status?.revision &&
              status.revision !== item.revision &&
              status.phase !== "stopped" &&
              status.phase !== "failed";
            return (
              <Card key={item.id} asChild>
                <article aria-label={item.name}>
                  <Flex direction="column" gap="3">
                    <Heading size="4" style={{ overflowWrap: "anywhere" }}>
                      {item.name}
                    </Heading>
                    <Text as="p" size="2" color="gray">
                      {item.description}
                    </Text>
                    <TunnelStatus status={status} />
                    <Text size="1" color="gray">
                      {t("tunnels.fields.protocol")}: {item.protocol} ·{" "}
                      {t("tunnels.fields.autoStart")}:{" "}
                      {t(item.autoStart ? "tunnels.yes" : "tunnels.no")} ·{" "}
                      {t("tunnels.fields.autoRestart")}:{" "}
                      {t(item.autoRestart ? "tunnels.yes" : "tunnels.no")}
                    </Text>
                    <Text size="1">
                      {t(item.hasToken ? "tunnels.hasToken" : "tunnels.noToken")}
                    </Text>
                    {status?.lastError && (
                      <Text as="p" size="2" color="red">
                        {t(runtimeErrorKey(status.lastError))}
                      </Text>
                    )}
                    {status?.phase === "backoff" && (
                      <Text size="1">
                        {t("tunnels.retry", {
                          count: status.restartCount,
                          time: status.nextRetry || unknown,
                        })}
                      </Text>
                    )}
                    {restart && <Badge color="amber">{t("tunnels.restartRequired")}</Badge>}
                    <TunnelActions
                      status={status}
                      busy={!!p.pending[item.id]}
                      available={!!p.caps?.available}
                      onAction={(a) => p.onAction(item.id, a)}
                    />
                    <Flex gap="3" wrap="wrap">
                      <Button
                        size="1"
                        variant="ghost"
                        disabled={p.pending[item.id]}
                        onClick={() => p.onEdit(item)}
                      >
                        {t("tunnels.edit")}
                      </Button>
                      <Button size="1" variant="ghost" onClick={() => p.onDetail(item)}>
                        {t("tunnels.details")}
                      </Button>
                      <Button
                        size="1"
                        variant="ghost"
                        color="red"
                        disabled={p.pending[item.id]}
                        onClick={() => p.onDelete(item)}
                      >
                        {t("tunnels.delete")}
                      </Button>
                    </Flex>
                  </Flex>
                </article>
              </Card>
            );
          })}
        </div>
      )}
    </Flex>
  );
}
