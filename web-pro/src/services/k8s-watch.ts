// @ts-nocheck
export type K8sWatchEvent = {
  type: string;
  object?: Record<string, unknown>;
  raw?: string;
};

export type K8sWatchQuery = {
  cluster_id: number;
  namespace?: string;
  resource?: string;
  label_selector?: string;
  timeout_seconds?: number;
};

/** 使用 fetch 读取 SSE（支持 Bearer），避免 EventSource 无法带头。 */
export async function streamK8sResourceWatch(
  query: K8sWatchQuery,
  onEvent: (ev: K8sWatchEvent) => void,
  signal?: AbortSignal,
): Promise<void> {
  const params = new URLSearchParams();
  params.set("cluster_id", String(query.cluster_id));
  if (query.namespace) params.set("namespace", query.namespace);
  params.set("resource", query.resource || "pods");
  if (query.label_selector) params.set("label_selector", query.label_selector);
  if (query.timeout_seconds) params.set("timeout_seconds", String(query.timeout_seconds));

  const res = await fetch(`/api/v1/k8s/resource-watch/stream?${params.toString()}`, {
    credentials: "include",
    signal,
  });
  if (!res.ok || !res.body) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `Watch 连接失败 (${res.status})`);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = "";
  let eventType = "message";
  let dataLines: string[] = [];

  const flush = () => {
    if (dataLines.length === 0) return;
    const raw = dataLines.join("\n");
    dataLines = [];
    try {
      const parsed = JSON.parse(raw) as K8sWatchEvent;
      onEvent({ ...parsed, type: parsed.type || eventType });
    } catch {
      onEvent({ type: eventType, raw });
    }
    eventType = "message";
  };

  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    const parts = buf.split("\n");
    buf = parts.pop() || "";
    for (const line of parts) {
      if (line.startsWith("event:")) {
        flush();
        eventType = line.slice(6).trim();
      } else if (line.startsWith("data:")) {
        dataLines.push(line.slice(5).trim());
      } else if (line === "") {
        flush();
      }
    }
  }
  flush();
}
