// @ts-nocheck
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from '@umijs/max';
import type { NamespaceOption } from "../components/k8s/yaml-crud-page";
import { getClusters, listNamespaces } from "../services/clusters";
import type { ClusterItem } from "../services/clusters";
import { useK8sContextStore } from "../stores/k8s-context-store";

export type UseK8sContextOptions = {
  needNamespace?: boolean;
  syncUrl?: boolean;
  onLoadNamespaces?: (clusterId: number) => Promise<NamespaceOption[]>;
};

export function useK8sContext(options?: UseK8sContextOptions) {
  const { needNamespace = true, syncUrl = true, onLoadNamespaces } = options ?? {};
  const [searchParams, setSearchParams] = useSearchParams();
  const clusterId = useK8sContextStore((s) => s.clusterId);
  const namespace = useK8sContextStore((s) => s.namespace);
  const setStoreClusterId = useK8sContextStore((s) => s.setClusterId);
  const setStoreNamespace = useK8sContextStore((s) => s.setNamespace);

  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [namespaceOptions, setNamespaceOptions] = useState<NamespaceOption[]>([]);
  const [loading, setLoading] = useState(true);
  const urlHydratedRef = useRef(false);

  useEffect(() => {
    if (!syncUrl || urlHydratedRef.current) return;
    urlHydratedRef.current = true;
    const clusterRaw = searchParams.get("cluster");
    const nsRaw = searchParams.get("ns");
    if (clusterRaw) {
      const id = Number.parseInt(clusterRaw, 10);
      if (!Number.isNaN(id) && id > 0) setStoreClusterId(id);
    }
    if (nsRaw) setStoreNamespace(nsRaw);
  }, [searchParams, setStoreClusterId, setStoreNamespace, syncUrl]);

  const syncUrlParams = useCallback(
    (nextClusterId?: number, nextNamespace?: string) => {
      if (!syncUrl) return;
      const next = new URLSearchParams(searchParams);
      if (nextClusterId) next.set("cluster", String(nextClusterId));
      else next.delete("cluster");
      if (needNamespace && nextNamespace) next.set("ns", nextNamespace);
      else next.delete("ns");
      setSearchParams(next, { replace: true });
    },
    [needNamespace, searchParams, setSearchParams, syncUrl],
  );

  const setClusterId = useCallback(
    (id: number) => {
      setStoreClusterId(id);
      syncUrlParams(id, namespace);
    },
    [namespace, setStoreClusterId, syncUrlParams],
  );

  const setNamespace = useCallback(
    (ns: string) => {
      setStoreNamespace(ns);
      syncUrlParams(clusterId, ns);
    },
    [clusterId, setStoreNamespace, syncUrlParams],
  );

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      setLoading(true);
      try {
        const res = await getClusters({ page: 1, page_size: 200 });
        if (cancelled) return;
        const list = res.list ?? [];
        setClusters(list);
        const active = list.filter((c) => c.status === 1);
        if (!clusterId && active[0]) {
          setStoreClusterId(active[0].id);
          syncUrlParams(active[0].id, namespace);
          return;
        }
        if (clusterId && !active.some((c) => c.id === clusterId) && active[0]) {
          setStoreClusterId(active[0].id);
          syncUrlParams(active[0].id, namespace);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!needNamespace || !clusterId) return;
    let cancelled = false;
    void (async () => {
      try {
        const loader =
          onLoadNamespaces ??
          (async (cid: number) => {
            const res = await listNamespaces(cid);
            return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
          });
        const opts = await loader(clusterId);
        if (cancelled) return;
        setNamespaceOptions(opts);
        if (!opts.some((o) => o.value === namespace)) {
          const first = opts[0]?.value ?? "default";
          setStoreNamespace(first);
          syncUrlParams(clusterId, first);
        }
      } catch {
        // http 拦截器已 toast
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [clusterId, needNamespace, namespace, onLoadNamespaces, setStoreNamespace, syncUrlParams]);

  const clusterOptions = useMemo(
    () =>
      clusters.map((c) => ({
        label: c.status === 1 ? c.name : `${c.name}（已停用）`,
        value: c.id,
        disabled: c.status !== 1,
      })),
    [clusters],
  );

  return {
    clusterId,
    namespace: needNamespace ? namespace : undefined,
    setClusterId,
    setNamespace,
    clusters,
    clusterOptions,
    namespaceOptions,
    loading,
  };
}
