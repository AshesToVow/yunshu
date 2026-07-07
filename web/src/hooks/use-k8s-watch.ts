import { useEffect, useRef } from "react";
import { message } from "antd";
import { streamK8sResourceWatch } from "../services/k8s-watch";
import { extractApiErrorMessage } from "../services/http";
import { useEditGuardStore } from "../stores/edit-guard-store";

export type UseK8sWatchOptions = {
  enabled: boolean;
  clusterId?: number;
  namespace?: string;
  resource?: string;
  onRefresh: () => void;
  onDisabled?: () => void;
};

/** K8s 资源 SSE Watch；编辑/弹窗打开时不触发 onRefresh。 */
export function useK8sWatch({
  enabled,
  clusterId,
  namespace,
  resource = "pods",
  onRefresh,
  onDisabled,
}: UseK8sWatchOptions) {
  const onRefreshRef = useRef(onRefresh);
  onRefreshRef.current = onRefresh;
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    abortRef.current?.abort();
    abortRef.current = null;

    if (!enabled || !clusterId || !namespace) return;

    let cancelled = false;
    const ac = new AbortController();
    abortRef.current = ac;

    const run = async () => {
      while (!cancelled && !ac.signal.aborted) {
        try {
          await streamK8sResourceWatch(
            { cluster_id: clusterId, namespace, resource, timeout_seconds: 3600 },
            () => {
              if (!useEditGuardStore.getState().isEditing) {
                onRefreshRef.current();
              }
            },
            ac.signal,
          );
        } catch (err: unknown) {
          if (ac.signal.aborted || cancelled) return;
          message.warning(extractApiErrorMessage(err, "Watch 已断开"));
          onDisabled?.();
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
    };
  }, [enabled, clusterId, namespace, resource, onDisabled]);
}
