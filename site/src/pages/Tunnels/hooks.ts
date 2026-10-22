import { useCallback, useEffect, useRef, useState } from "react";
import { tunnelAPI, type Action, type Capabilities, type Tunnel, type TunnelStatus } from "./api";

// Self-scheduling, visibility-aware polling: a slow request cannot stack ticks.
export function usePolling(task: (signal: AbortSignal) => Promise<void>, delay: number) {
  useEffect(() => {
    let active = true;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let controller: AbortController | undefined;
    let running = false;
    const run = async () => {
      if (!active || running || document.visibilityState === "hidden") return;
      running = true;
      controller = new AbortController();
      try {
        await task(controller.signal);
      } catch {
        /* task owns safe error presentation */
      } finally {
        running = false;
        if (active) timer = setTimeout(() => void run(), delay);
      }
    };
    const visible = () => {
      if (timer) clearTimeout(timer);
      if (document.visibilityState === "hidden") controller?.abort();
      else void run();
    };
    void run();
    document.addEventListener("visibilitychange", visible);
    return () => {
      active = false;
      controller?.abort();
      if (timer) clearTimeout(timer);
      document.removeEventListener("visibilitychange", visible);
    };
  }, [task, delay]);
}
export function useTunnels() {
  const [items, setItems] = useState<Tunnel[]>([]);
  const [statuses, setStatuses] = useState<Record<string, TunnelStatus>>({});
  const [caps, setCaps] = useState<Capabilities>();
  const [error, setError] = useState<unknown>();
  const [loading, setLoading] = useState(true);
  const [pending, setPending] = useState<Record<string, boolean>>({});
  const active = useRef(true);
  const operations = useRef(new Set<string>());
  const refreshBusy = useRef<AbortSignal | true | null>(null);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  const refresh = useCallback(async (signal?: AbortSignal) => {
    if (refreshBusy.current && (refreshBusy.current === true || !refreshBusy.current.aborted))
      return;
    const owner = signal ?? true;
    refreshBusy.current = owner;
    try {
      const list = await tunnelAPI.list(signal);
      if (signal?.aborted || !active.current) return;
      setItems(list.items);
      setError(undefined);
      // Four requests at most in flight, including large profile lists.
      for (let offset = 0; offset < list.items.length; offset += 4) {
        const chunk = await Promise.all(
          list.items.slice(offset, offset + 4).map(async (item) => {
            try {
              return [item.id, await tunnelAPI.status(item.id, signal)] as const;
            } catch (e) {
              if (!signal?.aborted && active.current) setError(e);
              return undefined;
            }
          })
        );
        if (signal?.aborted || !active.current) return;
        setStatuses((prev) => ({
          ...prev,
          ...Object.fromEntries(chunk.filter((v) => v !== undefined)),
        }));
      }
      try {
        const value = await tunnelAPI.capabilities(signal);
        if (!signal?.aborted && active.current) setCaps(value);
      } catch (e) {
        if (!signal?.aborted && active.current) setError(e);
      }
    } catch (e) {
      if (!signal?.aborted && active.current) setError(e);
    } finally {
      if (refreshBusy.current === owner) refreshBusy.current = null;
      if (active.current && !signal?.aborted) setLoading(false);
    }
  }, []);
  usePolling(refresh, 5000);
  const control = useCallback(async (id: string, action: Action) => {
    if (operations.current.has(id)) return;
    operations.current.add(id);
    setPending((prev) => ({ ...prev, [id]: true }));
    try {
      const accepted = await tunnelAPI.control(id, action);
      if (active.current) setStatuses((prev) => ({ ...prev, [id]: accepted }));
      const observed = await tunnelAPI.status(id);
      if (active.current) setStatuses((prev) => ({ ...prev, [id]: observed }));
    } finally {
      operations.current.delete(id);
      if (active.current) setPending((prev) => ({ ...prev, [id]: false }));
    }
  }, []);
  return { items, statuses, caps, error, loading, pending, refresh, control };
}
