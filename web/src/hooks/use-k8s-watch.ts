import { useEffect, useRef } from "react";
import { message } from "antd";
import { streamK8sResourceWatch, type K8sWatchEvent } from "../services/k8s-watch";
import { extractApiErrorMessage } from "../services/http";
import { useEditGuardStore } from "../stores/edit-guard-store";

export type UseK8sWatchOptions = {
  enabled: boolean;
  clusterId?: number;
  namespace?: string;
  resource?: string;
  /** 命名空间资源为 true；Nodes/Namespaces/PV 等集群级资源为 false */
  requireNamespace?: boolean;
  /** Watch 事件后的静默刷新（勿在依赖中传不稳定内联函数；内部已用 ref 稳定） */
  onRefresh: () => void;
  onDisabled?: () => void;
  /** 合并短时间内的多次事件，默认 1200ms */
  debounceMs?: number;
};

const IGNORED_EVENT_TYPES = new Set(["error", "bookmark", "heartbeat"]);

/** K8s 资源 SSE Watch；编辑/弹窗打开时不触发 onRefresh。 */
export function useK8sWatch({
  enabled,
  clusterId,
  namespace,
  resource = "pods",
  requireNamespace = true,
  onRefresh,
  onDisabled,
  debounceMs = 1200,
}: UseK8sWatchOptions) {
  const onRefreshRef = useRef(onRefresh);
  onRefreshRef.current = onRefresh;
  const onDisabledRef = useRef(onDisabled);
  onDisabledRef.current = onDisabled;
  const abortRef = useRef<AbortController | null>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingRefreshRef = useRef(false);

  useEffect(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
      debounceTimerRef.current = null;
    }
    pendingRefreshRef.current = false;

    if (!enabled || !clusterId) return;
    if (requireNamespace && !namespace) return;

    let cancelled = false;
    const ac = new AbortController();
    abortRef.current = ac;

    const scheduleRefresh = () => {
      if (useEditGuardStore.getState().isEditing) return;
      pendingRefreshRef.current = true;
      if (debounceTimerRef.current) return;
      debounceTimerRef.current = setTimeout(() => {
        debounceTimerRef.current = null;
        if (!pendingRefreshRef.current || cancelled || ac.signal.aborted) return;
        pendingRefreshRef.current = false;
        if (useEditGuardStore.getState().isEditing) return;
        onRefreshRef.current();
      }, Math.max(300, debounceMs));
    };

    const handleEvent = (ev: K8sWatchEvent) => {
      const t = String(ev.type || "").toLowerCase();
      if (IGNORED_EVENT_TYPES.has(t)) return;
      scheduleRefresh();
    };

    const run = async () => {
      while (!cancelled && !ac.signal.aborted) {
        try {
          await streamK8sResourceWatch(
            {
              cluster_id: clusterId,
              namespace: requireNamespace ? namespace : namespace || undefined,
              resource,
              timeout_seconds: 3600,
            },
            handleEvent,
            ac.signal,
          );
        } catch (err: unknown) {
          if (ac.signal.aborted || cancelled) return;
          message.warning(extractApiErrorMessage(err, "Watch 已断开"));
          onDisabledRef.current?.();
          return;
        }
        if (ac.signal.aborted || cancelled) return;
        await new Promise((resolve) => setTimeout(resolve, 2000));
      }
    };

    void run();
    return () => {
      cancelled = true;
      ac.abort();
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
        debounceTimerRef.current = null;
      }
    };
  }, [enabled, clusterId, namespace, resource, requireNamespace, debounceMs]);
}
