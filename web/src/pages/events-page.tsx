import { Card, Input, Select, Space, Table, Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getClusters, listNamespaces as listClusterNamespaces } from "../services/clusters";
import type { ClusterItem } from "../services/clusters";
import { listEvents, type EventItem } from "../services/events";

export function EventsPage() {
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [clusterId, setClusterId] = useState<number | undefined>(undefined);
  const [namespace, setNamespace] = useState<string>("default");
  const [namespaceOptions, setNamespaceOptions] = useState<{ label: string; value: string }[]>([]);
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<EventItem[]>([]);

  const filterRef = useRef({ clusterId, namespace, keyword });
  filterRef.current = { clusterId, namespace, keyword };

  const clusterOptions = useMemo(
    () =>
      clusters.map((c) => ({
        label: c.status === 1 ? c.name : `${c.name}（已停用）`,
        value: c.id,
        disabled: c.status !== 1,
      })),
    [clusters],
  );

  const reload = useCallback(async (overrideKeyword?: string) => {
    const { clusterId: cid, namespace: ns, keyword: kw } = filterRef.current;
    if (!cid) return;
    const effectiveKeyword = (overrideKeyword ?? kw).trim();
    setLoading(true);
    try {
      const items = await listEvents({
        cluster_id: cid,
        namespace: ns,
        keyword: effectiveKeyword || undefined,
        limit: 500,
      });
      setData(items ?? []);
    } catch {
      // http 拦截器已 toast
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const res = await getClusters({ page: 1, page_size: 200 });
        if (cancelled) return;
        setClusters(res.list ?? []);
        if (!filterRef.current.clusterId) {
          const first = (res.list ?? []).find((c) => c.status === 1);
          if (first) setClusterId(first.id);
        }
      } catch {
        // http 拦截器已 toast
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!clusterId) return;
    let cancelled = false;
    void (async () => {
      try {
        const res = await listClusterNamespaces(clusterId);
        if (cancelled) return;
        const opts = (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
        setNamespaceOptions(opts);
        if (!opts.some((o) => o.value === filterRef.current.namespace)) {
          setNamespace(opts[0]?.value ?? "default");
        }
      } catch {
        // http 拦截器已 toast
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [clusterId]);

  useEffect(() => {
    void reload();
  }, [clusterId, namespace, reload]);

  useEffect(() => {
    if (!clusterId) return;
    const timer = window.setInterval(() => {
      void reload();
    }, 10000);
    return () => window.clearInterval(timer);
  }, [clusterId, namespace, reload]);

  const columns: ColumnsType<EventItem> = [
    { title: "时间", dataIndex: "last_time", width: 180, render: (v: string, r) => v || r.creation_time || "-" },
    { title: "命名空间", dataIndex: "namespace", width: 140 },
    { title: "类型", dataIndex: "type", width: 90, render: (v: string) => <Tag color={v === "Warning" ? "red" : "green"}>{v || "-"}</Tag> },
    { title: "原因", dataIndex: "reason", width: 160 },
    { title: "对象", key: "obj", width: 220, render: (_, r) => `${r.involved_kind ?? "-"} / ${r.involved_name ?? "-"}` },
    { title: "次数", dataIndex: "count", width: 80 },
    { title: "消息", dataIndex: "message" },
  ];

  return (
    <Card className="table-card" title="Event 事件管理">
      <div style={{ display: "flex", gap: 12, alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
        <Space wrap>
          <Select
            placeholder="选择集群"
            style={{ minWidth: 240 }}
            value={clusterId}
            onChange={setClusterId}
            options={clusterOptions}
          />
          <Select
            placeholder="命名空间"
            style={{ minWidth: 200 }}
            value={namespace}
            onChange={setNamespace}
            options={namespaceOptions}
            showSearch
            optionFilterProp="label"
          />
          <Input.Search
            allowClear
            placeholder="搜索 reason/message/对象"
            style={{ width: 320 }}
            onSearch={(v) => {
              setKeyword(v);
              void reload(v);
            }}
          />
        </Space>
      </div>
      <Table
        rowKey={(r) => `${r.namespace}/${r.involved_kind}/${r.involved_name}/${r.last_time}/${r.reason}`}
        loading={loading}
        dataSource={data}
        columns={columns}
        pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
      />
    </Card>
  );
}
