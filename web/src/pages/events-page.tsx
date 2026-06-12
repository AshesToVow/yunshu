import { Card, Input, Radio, Select, Space, Table, Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { getClusters, listNamespaces as listClusterNamespaces } from "../services/clusters";
import type { ClusterItem } from "../services/clusters";
import { listEvents, listEventsGrouped, type EventGroupItem, type EventItem } from "../services/events";
import { buildK8sResourceLink } from "../utils/k8s-resource-links";

type ViewMode = "list" | "grouped";

function InvolvedObjectLink({
  clusterId,
  namespace,
  kind,
  name,
}: {
  clusterId?: number;
  namespace?: string;
  kind?: string;
  name?: string;
}) {
  const href = buildK8sResourceLink({ kind, name, clusterId, namespace });
  const label = `${kind ?? "-"} / ${name ?? "-"}`;
  if (!href) return <span>{label}</span>;
  return (
    <Link to={href}>
      <Tag color="blue">{kind || "?"}</Tag> {name || "-"}
    </Link>
  );
}

export function EventsPage() {
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [clusterId, setClusterId] = useState<number | undefined>(undefined);
  const [namespace, setNamespace] = useState<string>("default");
  const [namespaceOptions, setNamespaceOptions] = useState<{ label: string; value: string }[]>([]);
  const [keyword, setKeyword] = useState("");
  const [viewMode, setViewMode] = useState<ViewMode>("list");
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<EventItem[]>([]);
  const [grouped, setGrouped] = useState<EventGroupItem[]>([]);

  const filterRef = useRef({ clusterId, namespace, keyword, viewMode });
  filterRef.current = { clusterId, namespace, keyword, viewMode };

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
    const { clusterId: cid, namespace: ns, keyword: kw, viewMode: mode } = filterRef.current;
    if (!cid) return;
    const effectiveKeyword = (overrideKeyword ?? kw).trim();
    setLoading(true);
    try {
      const params = {
        cluster_id: cid,
        namespace: ns,
        keyword: effectiveKeyword || undefined,
        limit: 500,
      };
      if (mode === "grouped") {
        setGrouped(await listEventsGrouped(params));
      } else {
        setData(await listEvents(params));
      }
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
        // ignore
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
        // ignore
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [clusterId]);

  useEffect(() => {
    void reload();
  }, [clusterId, namespace, viewMode, reload]);

  useEffect(() => {
    if (!clusterId) return;
    const timer = window.setInterval(() => void reload(), 10000);
    return () => window.clearInterval(timer);
  }, [clusterId, namespace, viewMode, reload]);

  const listColumns: ColumnsType<EventItem> = [
    { title: "时间", dataIndex: "last_time", width: 180, render: (v: string, r) => v || r.creation_time || "-" },
    { title: "命名空间", dataIndex: "namespace", width: 140 },
    { title: "类型", dataIndex: "type", width: 90, render: (v: string) => <Tag color={v === "Warning" ? "red" : "green"}>{v || "-"}</Tag> },
    { title: "原因", dataIndex: "reason", width: 160 },
    {
      title: "对象",
      key: "obj",
      width: 240,
      render: (_, r) => (
        <InvolvedObjectLink clusterId={clusterId} namespace={r.namespace} kind={r.involved_kind} name={r.involved_name} />
      ),
    },
    { title: "次数", dataIndex: "count", width: 80 },
    { title: "消息", dataIndex: "message", ellipsis: true },
  ];

  const groupedColumns: ColumnsType<EventGroupItem> = [
    { title: "最近时间", dataIndex: "last_time", width: 180 },
    { title: "命名空间", dataIndex: "namespace", width: 120 },
    {
      title: "对象",
      key: "obj",
      width: 240,
      render: (_, r) => (
        <InvolvedObjectLink clusterId={clusterId} namespace={r.namespace} kind={r.involved_kind} name={r.involved_name} />
      ),
    },
    { title: "原因", dataIndex: "reason", width: 140 },
    { title: "类型", dataIndex: "type", width: 90, render: (v) => <Tag color={v === "Warning" ? "red" : "green"}>{v}</Tag> },
    { title: "事件条数", dataIndex: "event_count", width: 90 },
    { title: "累计次数", dataIndex: "total_count", width: 90 },
    { title: "最新消息", dataIndex: "message", ellipsis: true },
  ];

  return (
    <Card className="table-card" title="Event 事件管理">
      <div style={{ display: "flex", gap: 12, alignItems: "center", justifyContent: "space-between", marginBottom: 12, flexWrap: "wrap" }}>
        <Space wrap>
          <Select placeholder="选择集群" style={{ minWidth: 240 }} value={clusterId} onChange={setClusterId} options={clusterOptions} />
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
        <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value as ViewMode)}>
          <Radio.Button value="list">列表</Radio.Button>
          <Radio.Button value="grouped">按对象聚合</Radio.Button>
        </Radio.Group>
      </div>
      {viewMode === "list" ? (
        <Table
          rowKey={(r) => `${r.namespace}/${r.involved_kind}/${r.involved_name}/${r.last_time}/${r.reason}`}
          loading={loading}
          dataSource={data}
          columns={listColumns}
          pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
        />
      ) : (
        <Table
          rowKey={(r) => `${r.namespace}/${r.involved_kind}/${r.involved_name}/${r.reason}`}
          loading={loading}
          dataSource={grouped}
          columns={groupedColumns}
          expandable={{
            expandedRowRender: (r) => (
              <Table
                size="small"
                pagination={false}
                dataSource={r.events ?? []}
                rowKey={(e) => `${e.last_time}/${e.message}`}
                columns={[
                  { title: "时间", dataIndex: "last_time", width: 160 },
                  { title: "次数", dataIndex: "count", width: 70 },
                  { title: "消息", dataIndex: "message" },
                ]}
              />
            ),
          }}
          pagination={{ pageSize: 10, showSizeChanger: true }}
        />
      )}
    </Card>
  );
}
