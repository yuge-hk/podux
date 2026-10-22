import { useCallback, useEffect, useRef, useState } from "react";
import { Button, Callout, Flex, Switch, Text } from "@radix-ui/themes";
import { useTranslation } from "react-i18next";
import { errorKey, tunnelAPI, type TunnelLog } from "../api";
import { usePolling } from "../hooks";
export function TunnelLogViewer({ id }: { id: string }) {
  const { t } = useTranslation();
  const [logs, setLogs] = useState<TunnelLog[]>([]);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [scroll, setScroll] = useState(true);
  const [busy, setBusy] = useState(false);
  const cursor = useRef(0);
  const epoch = useRef<string | undefined>(undefined);
  const inFlight = useRef<AbortSignal | true | null>(null);
  const active = useRef(true);
  const bottom = useRef<HTMLDivElement>(null);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  const refresh = useCallback(
    async (signal?: AbortSignal) => {
      if (inFlight.current && (inFlight.current === true || !inFlight.current.aborted)) return;
      const owner = signal ?? true;
      inFlight.current = owner;
      setBusy(true);
      try {
        let result = await tunnelAPI.logs(id, cursor.current, signal);
        let reset = false;
        // New servers identify log-ring lifetimes; retain the bounded snapshot
        // fallback when connected to an older Phase 1 server without an epoch.
        if (result.epoch && epoch.current && result.epoch !== epoch.current) {
          result = await tunnelAPI.logs(id, 0, signal);
          reset = true;
        } else if (result.items.length === 0 && cursor.current > 0 && !result.epoch) {
          const snapshot = await tunnelAPI.logs(id, 0, signal);
          if ((snapshot.items.at(-1)?.cursor ?? 0) < cursor.current) {
            result = snapshot;
            reset = true;
          }
        }
        if (signal?.aborted || !active.current) return;
        if (result.epoch) epoch.current = result.epoch;
        const ordered = [...result.items].sort((a, b) => a.cursor - b.cursor);
        if (reset) {
          setNotice("tunnels.logs.reset");
          cursor.current = 0;
        }
        if (ordered.length && ordered[0].cursor > cursor.current + 1)
          setNotice("tunnels.logs.truncated");
        const newLogs = ordered.filter((l) => l.cursor > cursor.current);
        if (newLogs.length) cursor.current = newLogs[newLogs.length - 1].cursor;
        setLogs((previous) => {
          const merged = [...(reset ? [] : previous), ...newLogs];
          return merged.slice(-500);
        });
        setError("");
      } catch (e) {
        if (!signal?.aborted && active.current) setError(errorKey(e));
      } finally {
        if (inFlight.current === owner) {
          inFlight.current = null;
          if (active.current) setBusy(false);
        }
      }
    },
    [id]
  );
  usePolling(refresh, 3000);
  useEffect(() => {
    if (scroll) bottom.current?.scrollIntoView?.({ block: "nearest" });
  }, [logs, scroll]);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(logs.map((l) => `${l.cursor} ${l.message}`).join("\n"));
    } catch {
      setError("tunnels.errors.copy");
    }
  };
  return (
    <Flex direction="column" gap="3">
      <Text size="1" color="gray">
        {t("tunnels.logs.help")}
      </Text>
      {error && (
        <Callout.Root color="red" role="alert">
          <Callout.Text>{t(error)}</Callout.Text>
        </Callout.Root>
      )}
      {notice && (
        <Text role="status" size="1" color="amber">
          {t(notice)}
        </Text>
      )}
      <Flex gap="3" align="center" wrap="wrap">
        <Text as="label" size="2">
          <Flex gap="2" align="center">
            <Switch
              aria-label={t("tunnels.logs.autoScroll")}
              checked={scroll}
              onCheckedChange={setScroll}
            />
            {t("tunnels.logs.autoScroll")}
          </Flex>
        </Text>
        <Button size="1" variant="soft" loading={busy} onClick={() => void refresh()}>
          {t("tunnels.refresh")}
        </Button>
        <Button size="1" variant="soft" disabled={!logs.length} onClick={() => void copy()}>
          {t("tunnels.logs.copy")}
        </Button>
      </Flex>
      <div className="tunnel-log" role="log" aria-label={t("tunnels.logs.title")} aria-live="off">
        {logs.length ? (
          logs.map((l) => (
            <div key={l.cursor}>
              {l.cursor} {l.message}
            </div>
          ))
        ) : (
          <Text color="gray">{t("tunnels.logs.empty")}</Text>
        )}
        <div ref={bottom} />
      </div>
    </Flex>
  );
}
